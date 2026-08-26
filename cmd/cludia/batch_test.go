package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/tunesmith/cludia/internal/argfile"
	"github.com/tunesmith/cludia/internal/argument"
)

func TestAddBatchDryRunAndMutationReturnsKeyMapping(t *testing.T) {
	path := twoPremiseWorkspace(t)
	inputPath := writeBatchInput(t, `{
  "schema_version": 1,
  "statements": [
    {"key": "migration-status", "text": "Migration phase status is known."},
    {"key": "ticket-42061", "text": "42061 blocks the migration.", "truth": "U", "kind": "value"}
  ]
}`)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err := run([]string{"add-batch", "--json", path, "--input", inputPath, "--dry-run"}, &stdout, &stderr); err != nil {
		t.Fatalf("add-batch dry-run: %v\nstderr: %s", err, stderr.String())
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	assertExactKeys(t, raw, "schema_version", "action", "dry_run", "profile", "document", "statements", "changes", "diagnostics")
	var dryRun batchAddOutput
	if err := json.Unmarshal(stdout.Bytes(), &dryRun); err != nil {
		t.Fatal(err)
	}
	if !dryRun.DryRun || len(dryRun.Statements) != 2 || len(dryRun.Changes) != 2 {
		t.Fatalf("dry-run output = %#v", dryRun)
	}
	if got := dryRun.Statements[0]; got.Key != "migration-status" || got.Statement.ID != "P3" {
		t.Fatalf("first mapping = %#v", got)
	}
	if got := dryRun.Statements[1]; got.Key != "ticket-42061" || got.Statement.ID != "P4" || got.Statement.Slug != "statement-42061-blocks-migration" || got.Statement.Truth != argument.TruthUnknown || got.Statement.Kind != argument.KindValue {
		t.Fatalf("second mapping = %#v", got)
	}
	afterDryRun, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, afterDryRun) {
		t.Fatalf("dry-run changed file: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"add-batch", path, "--input", inputPath, "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("add-batch: %v\nstderr: %s", err, stderr.String())
	}
	parsed := argfile.ParseFile(path)
	if len(parsed.Document.Statements) != 4 || parsed.Document.Statements[2].ID != "P3" || parsed.Document.Statements[3].ID != "P4" {
		t.Fatalf("saved statements = %#v", parsed.Document.Statements)
	}
}

func TestAddBatchInvalidStatementLeavesWorkspaceUnchanged(t *testing.T) {
	path := twoPremiseWorkspace(t)
	inputPath := writeBatchInput(t, `{
  "schema_version": 1,
  "statements": [
    {"key": "valid", "text": "Valid statement."},
    {"key": "invalid", "text": "Invalid statement.", "slug": "42061-invalid"}
  ]
}`)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err = run([]string{"add-batch", path, "--input", inputPath, "--json"}, &stdout, &stderr)
	if !errors.Is(err, errValidationFailed) {
		t.Fatalf("add-batch error = %v", err)
	}
	var failure failureOutput
	if err := json.Unmarshal(stdout.Bytes(), &failure); err != nil {
		t.Fatal(err)
	}
	if len(failure.Diagnostics) != 1 || failure.Diagnostics[0].Code != "statement_slug_invalid" || failure.Diagnostics[0].Element != "invalid" {
		t.Fatalf("diagnostics = %#v", failure.Diagnostics)
	}
	after, readErr := os.ReadFile(path)
	if readErr != nil || !bytes.Equal(before, after) {
		t.Fatalf("workspace changed after failed batch: %v", readErr)
	}
}

func TestAddBatchRejectsInvalidInputWithoutWriting(t *testing.T) {
	tests := []struct {
		name  string
		input string
		code  string
	}{
		{name: "unsupported schema", input: `{"schema_version": 2, "statements": [{"key": "one", "text": "One"}]}`, code: "batch_schema_version_unsupported"},
		{name: "no statements", input: `{"schema_version": 1, "statements": []}`, code: "batch_statements_required"},
		{name: "duplicate key", input: `{"schema_version": 1, "statements": [{"key": "same", "text": "One"}, {"key": "same", "text": "Two"}]}`, code: "batch_key_duplicate"},
		{name: "duplicate durable id", input: `{"schema_version": 1, "statements": [{"key": "duplicate-id", "text": "One", "id": "P1"}]}`, code: "id_duplicate"},
		{name: "duplicate slug", input: `{"schema_version": 1, "statements": [{"key": "duplicate-slug", "text": "One", "slug": "first"}]}`, code: "statement_slug_duplicate"},
		{name: "unknown field", input: `{"schema_version": 1, "statements": [{"key": "one", "text": "One", "extra": true}]}`, code: "batch_input_invalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := twoPremiseWorkspace(t)
			inputPath := writeBatchInput(t, tt.input)
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			err = run([]string{"add-batch", path, "--input", inputPath, "--json"}, &stdout, &stderr)
			if !errors.Is(err, errValidationFailed) {
				t.Fatalf("add-batch error = %v", err)
			}
			var failure failureOutput
			if err := json.Unmarshal(stdout.Bytes(), &failure); err != nil {
				t.Fatal(err)
			}
			if len(failure.Diagnostics) == 0 || failure.Diagnostics[0].Code != tt.code {
				t.Fatalf("diagnostics = %#v", failure.Diagnostics)
			}
			after, readErr := os.ReadFile(path)
			if readErr != nil || !bytes.Equal(before, after) {
				t.Fatalf("workspace changed after invalid input: %v", readErr)
			}
		})
	}
}

func writeBatchInput(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "batch.json")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
