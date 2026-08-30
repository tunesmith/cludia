package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tunesmith/cludia/internal/argfile"
	"github.com/tunesmith/cludia/internal/argument"
	"github.com/tunesmith/cludia/internal/evaluation"
)

func TestAddBatchHelpIncludesSchemaAndExampleCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"add-batch", "--help"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"cludia add-batch --example", `"schema_version": 2`, `"key": "needle-inference"`, "id, slug, truth (T|F|U), kind (fact|value)"} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("add-batch help missing %q:\n%s", want, stderr.String())
		}
	}
}

func TestAddBatchExamplePrintsValidMinimalInput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"add-batch", "--example"}, &stdout, &stderr); err != nil {
		t.Fatalf("add-batch --example: %v\nstderr: %s", err, stderr.String())
	}
	var input batchAuthorInput
	if err := json.Unmarshal(stdout.Bytes(), &input); err != nil {
		t.Fatalf("example is not valid JSON: %v\n%s", err, stdout.String())
	}
	if input.SchemaVersion != 2 || len(input.Statements) != 4 || len(input.Derivations) != 1 || len(input.Defeats) != 1 {
		t.Fatalf("example is not a valid authoring transaction: %#v", input)
	}
}

func TestAddBatchDryRunAndMutationReturnsKeyMapping(t *testing.T) {
	path := twoPremiseWorkspace(t)
	inputPath := writeBatchInput(t, `{
	  "schema_version": 2,
	  "statements": [
	    {"key": "migration-status", "text": "Migration phase status is known."},
	    {"key": "ticket-42061", "text": "42061 blocks the migration.", "truth": "U", "kind": "value"}
	  ],
	  "derivations": [],
	  "defeats": []
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
	assertExactKeys(t, raw, "schema_version", "action", "dry_run", "profile", "document", "statements", "derivations", "defeats", "role_changes", "truth_changes", "changes", "diagnostics")
	var dryRun batchAuthorOutput
	if err := json.Unmarshal(stdout.Bytes(), &dryRun); err != nil {
		t.Fatal(err)
	}
	if !dryRun.DryRun || len(dryRun.Statements) != 2 || len(dryRun.Changes) != 3 {
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
	  "schema_version": 2,
	  "statements": [
	    {"key": "valid", "text": "Valid statement."},
	    {"key": "invalid", "text": "Invalid statement.", "slug": "42061-invalid"}
	  ],
	  "derivations": [],
	  "defeats": []
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
	if len(failure.Diagnostics) != 1 || failure.Diagnostics[0].Code != "statement_slug_invalid" || failure.Diagnostics[0].Element != "P4" {
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
		{name: "retired schema", input: `{"schema_version": 1, "statements": [{"key": "one", "text": "One"}]}`, code: "batch_schema_version_unsupported"},
		{name: "no operations", input: `{"schema_version": 2, "statements": [], "derivations": [], "defeats": []}`, code: "batch_operations_required"},
		{name: "duplicate key", input: `{"schema_version": 2, "statements": [{"key": "same", "text": "One"}, {"key": "same", "text": "Two"}], "derivations": [], "defeats": []}`, code: "batch_key_duplicate"},
		{name: "retired or duplicate durable id", input: `{"schema_version": 2, "statements": [{"key": "duplicate-id", "text": "One", "id": "P1"}], "derivations": [], "defeats": []}`, code: "id_not_next"},
		{name: "duplicate slug", input: `{"schema_version": 2, "statements": [{"key": "duplicate-slug", "text": "One", "slug": "first"}], "derivations": [], "defeats": []}`, code: "statement_slug_duplicate"},
		{name: "unknown field", input: `{"schema_version": 2, "statements": [{"key": "one", "text": "One", "extra": true}], "derivations": [], "defeats": []}`, code: "batch_input_invalid"},
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

func TestBatchFieldDiagnosticsReportAllInvalidFieldsAgainstCallerKey(t *testing.T) {
	path := twoPremiseWorkspace(t)
	inputPath := writeBatchInput(t, `{"schema_version":2,"statements":[{"key":"invalid","text":"Invalid","truth":"bad-truth","kind":"bad-kind"}],"derivations":[],"defeats":[]}`)
	var stdout, stderr bytes.Buffer
	err := run([]string{"add-batch", path, "--input", inputPath, "--json"}, &stdout, &stderr)
	if !errors.Is(err, errValidationFailed) {
		t.Fatalf("error = %v", err)
	}
	var failure failureOutput
	if err := json.Unmarshal(stdout.Bytes(), &failure); err != nil || len(failure.Diagnostics) != 2 {
		t.Fatalf("diagnostics = %#v, %v", failure.Diagnostics, err)
	}
	for _, item := range failure.Diagnostics {
		if item.Element != "invalid" {
			t.Fatalf("diagnostic element = %q, want caller key", item.Element)
		}
	}
}

func TestAddBatchV2AuthorsStatementsRelationsAndDefeatsAtomically(t *testing.T) {
	path := twoPremiseWorkspace(t)
	inputPath := writeBatchInput(t, `{
  "schema_version": 2,
  "statements": [
    {"key": "target", "text": "The clasp caused the puncture."},
    {"key": "alternative", "role": "counterpoint", "text": "Another object could have caused it."},
    {"key": "answer", "role": "counterpoint", "text": "The groove excludes an unrelated object."}
  ],
  "derivations": [
    {"key": "needle-inference", "sources": [{"id": "P1"}, {"id": "P2"}], "target": {"key": "target"}}
  ],
  "defeats": [
    {"from": {"key": "alternative"}, "scope": "inference", "target": {"key": "needle-inference"}},
    {"from": {"key": "answer"}, "scope": "counterpoint", "target": {"key": "alternative"}}
  ]
}`)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err := run([]string{"add-batch", "--json", "--dry-run", path, "--input", inputPath}, &stdout, &stderr); err != nil {
		t.Fatalf("v2 dry-run: %v\nstderr: %s", err, stderr.String())
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	assertExactKeys(t, raw, "schema_version", "action", "dry_run", "profile", "document", "statements", "derivations", "defeats", "role_changes", "truth_changes", "changes", "diagnostics")
	var dryRun batchAuthorOutput
	if err := json.Unmarshal(stdout.Bytes(), &dryRun); err != nil {
		t.Fatal(err)
	}
	if !dryRun.DryRun || len(dryRun.Statements) != 3 || len(dryRun.Derivations) != 1 || len(dryRun.Defeats) != 2 {
		t.Fatalf("v2 dry-run output = %#v", dryRun)
	}
	if got := dryRun.Statements[0]; got.Key != "target" || got.Statement.ID != "L1" || got.Statement.Role != argument.RoleLemma || got.Statement.Truth != argument.TruthUnknown {
		t.Fatalf("final target mapping = %#v", got)
	}
	if got := dryRun.Derivations[0]; got.Key != "needle-inference" || got.Junctor.ID != "J1" || got.Junctor.Target != "L1" {
		t.Fatalf("derivation mapping = %#v", got)
	}
	if len(dryRun.RoleChanges) != 0 || len(dryRun.TruthChanges) != 0 {
		t.Fatalf("new final-role statements should not be promoted: roles=%#v truths=%#v", dryRun.RoleChanges, dryRun.TruthChanges)
	}
	afterDryRun, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, afterDryRun) {
		t.Fatalf("v2 dry-run changed workspace: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"add-batch", path, "--input", inputPath, "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("v2 apply: %v\nstderr: %s", err, stderr.String())
	}
	parsed := argfile.ParseFile(path)
	if len(parsed.Document.Statements) != 5 || len(parsed.Document.Junctors) != 1 || len(parsed.Document.Defeats) != 2 {
		t.Fatalf("saved transaction = %#v", parsed.Document)
	}
	if _, ok := parsed.Document.Statement("L1"); !ok {
		t.Fatal("final lemma L1 missing")
	}
	if _, ok := parsed.Document.Statement("P3"); ok {
		t.Fatal("batch consumed a transient premise ID for the derived target")
	}
	if value, _ := parsed.Document.MetadataValue(argument.NextIDsMetadataKey); value != "v1;P=3;L=2;C=1;CP=3;J=2" {
		t.Fatalf("allocator state = %q", value)
	}
	evaluated, err := evaluation.Evaluate(parsed.Document)
	if err != nil {
		t.Fatal(err)
	}
	target, ok := evaluated.Statement("L1")
	if !ok || target.EffectiveTruth != argument.TruthTrue || evaluated.TruthChangedByDefeat("L1") {
		t.Fatalf("grounded batch result = %#v", target)
	}
}

func TestAddBatchV2FinalValidationFailureWritesNothing(t *testing.T) {
	path := twoPremiseWorkspace(t)
	inputPath := writeBatchInput(t, `{
  "schema_version": 2,
  "statements": [],
  "derivations": [
    {"key": "self-support", "sources": [{"id": "P1"}, {"id": "P2"}], "target": {"id": "P1"}}
  ],
  "defeats": []
}`)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err = run([]string{"add-batch", path, "--input", inputPath, "--json"}, &stdout, &stderr)
	if !errors.Is(err, errValidationFailed) {
		t.Fatalf("invalid v2 batch error = %v", err)
	}
	var failure failureOutput
	if err := json.Unmarshal(stdout.Bytes(), &failure); err != nil {
		t.Fatal(err)
	}
	if len(failure.Diagnostics) == 0 || failure.Diagnostics[0].Code != "support_self" {
		t.Fatalf("diagnostics = %#v", failure.Diagnostics)
	}
	after, readErr := os.ReadFile(path)
	if readErr != nil || !bytes.Equal(before, after) {
		t.Fatalf("invalid v2 batch changed workspace: %v", readErr)
	}
}

func TestAddBatchV2ReportsExistingPremisePromotion(t *testing.T) {
	path := twoPremiseWorkspace(t)
	inputPath := writeBatchInput(t, `{
  "schema_version": 2,
  "statements": [{"key": "third-source", "text": "Third source."}],
  "derivations": [
    {"key": "existing-target", "sources": [{"id": "P2"}, {"key": "third-source"}], "target": {"id": "P1"}}
  ],
  "defeats": []
}`)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"add-batch", path, "--input", inputPath, "--dry-run", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("promotion dry-run: %v\nstderr: %s", err, stderr.String())
	}
	var output batchAuthorOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if len(output.RoleChanges) != 1 || output.RoleChanges[0] != (roleChangeOutput{PreviousID: "P1", CurrentID: "L1", From: argument.RolePremise, To: argument.RoleLemma}) {
		t.Fatalf("role changes = %#v", output.RoleChanges)
	}
	if len(output.Derivations) != 1 || output.Derivations[0].Junctor.Target != "L1" {
		t.Fatalf("derivations = %#v", output.Derivations)
	}
	if len(output.Diagnostics) != 1 || output.Diagnostics[0].Code != "external_id_references_unchecked" || output.Diagnostics[0].Element != "L1" {
		t.Fatalf("diagnostics = %#v", output.Diagnostics)
	}
}

func TestAddBatchV2RejectsAmbiguousAndSlugLikeReferences(t *testing.T) {
	tests := []struct {
		name  string
		input string
		code  string
	}{
		{
			name: "ambiguous", code: "batch_reference_invalid",
			input: `{"schema_version":2,"statements":[],"derivations":[{"key":"d","sources":[{"key":"new","id":"P1"},{"id":"P2"}],"target":{"id":"P1"}}],"defeats":[]}`,
		},
		{
			name: "durable ids only", code: "batch_statement_id_not_found",
			input: `{"schema_version":2,"statements":[],"derivations":[{"key":"d","sources":[{"id":"first"},{"id":"P2"}],"target":{"id":"P1"}}],"defeats":[]}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := twoPremiseWorkspace(t)
			inputPath := writeBatchInput(t, test.input)
			var stdout, stderr bytes.Buffer
			err := run([]string{"add-batch", path, "--input", inputPath, "--json"}, &stdout, &stderr)
			if !errors.Is(err, errValidationFailed) {
				t.Fatalf("error = %v", err)
			}
			var failure failureOutput
			if err := json.Unmarshal(stdout.Bytes(), &failure); err != nil || len(failure.Diagnostics) == 0 || failure.Diagnostics[0].Code != test.code {
				t.Fatalf("failure = %#v err=%v", failure, err)
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
