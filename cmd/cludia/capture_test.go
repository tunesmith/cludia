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
	"github.com/tunesmith/cludia/internal/diagnostic"
)

func TestInitAndAddCreateValidWorkspace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "question.arg")
	var stdout, stderr bytes.Buffer
	err := run([]string{
		"init", path, "--title", "How should people work with Cludia?",
		"--text", "Marlow completed the first milestone.", "--json",
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("init: %v\nstderr: %s", err, stderr.String())
	}
	var initialized map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &initialized); err != nil {
		t.Fatalf("decode init JSON: %v", err)
	}
	assertExactKeys(t, initialized, "schema_version", "action", "dry_run", "profile", "document", "statement", "changes", "diagnostics")
	var initializedStatement map[string]json.RawMessage
	if err := json.Unmarshal(initialized["statement"], &initializedStatement); err != nil {
		t.Fatalf("decode initialized statement: %v", err)
	}
	assertExactKeys(t, initializedStatement, "id", "slug", "role", "kind", "truth", "text")

	stdout.Reset()
	stderr.Reset()
	err = run([]string{
		"add", "--truth", "unknown", path, "--kind", "value", "--text", "Direct CLI access should remain available.", "--json",
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("add: %v\nstderr: %s", err, stderr.String())
	}
	var added mutationOutput
	if err := json.Unmarshal(stdout.Bytes(), &added); err != nil {
		t.Fatalf("decode add JSON: %v", err)
	}
	if added.Action != "add" || added.Statement.ID != "P2" || added.Statement.Kind != argument.KindValue || added.Statement.Truth != argument.TruthUnknown {
		t.Fatalf("unexpected add output: %#v", added)
	}
	if added.Diagnostics == nil {
		t.Fatal("add diagnostics must be an empty array, not null")
	}
	parsed := argfile.ParseFile(path)
	if diagnostic.HasErrors(parsed.Diagnostics) || len(parsed.Document.Statements) != 2 {
		t.Fatalf("created workspace: %#v, diagnostics %#v", parsed.Document, parsed.Diagnostics)
	}
}

func TestInitRefusesExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "existing.arg")
	const original = "existing\n"
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err := run([]string{"init", path, "--title", "Title", "--text", "First"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("init overwrote existing file")
	}
	data, readErr := os.ReadFile(path)
	if readErr != nil || string(data) != original {
		t.Fatalf("existing file changed: %q, err %v", data, readErr)
	}
}

func TestFailedAddLeavesWorkspaceUnchanged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workspace.arg")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"init", path, "--title", "Title", "--text", "First"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	err = run([]string{"add", path, "--id", "P1", "--text", "Duplicate", "--json"}, &stdout, &stderr)
	if !errors.Is(err, errValidationFailed) {
		t.Fatalf("add error = %v, want validation failure", err)
	}
	after, readErr := os.ReadFile(path)
	if readErr != nil || !bytes.Equal(before, after) {
		t.Fatalf("workspace changed after failed add: %v", readErr)
	}
	var failure failureOutput
	if err := json.Unmarshal(stdout.Bytes(), &failure); err != nil {
		t.Fatalf("decode failure JSON: %v", err)
	}
	if failure.OK || len(failure.Diagnostics) == 0 {
		t.Fatalf("failure output = %#v", failure)
	}
}

func TestEditChangesOnlyTextAndReportsPreviousStatement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workspace.arg")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"init", path, "--title", "Title", "--text", "Original wording"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	before := argfile.ParseFile(path).Document.Statements[0]
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"edit", path, before.Slug, "--text", "Narrower wording", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("edit: %v\nstderr: %s", err, stderr.String())
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatalf("decode edit JSON: %v", err)
	}
	assertExactKeys(t, raw, "schema_version", "action", "dry_run", "profile", "document", "statement", "previous_statement", "changes", "diagnostics")
	var output mutationOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode typed edit JSON: %v", err)
	}
	if output.PreviousStatement == nil || output.PreviousStatement.Text != "Original wording" || output.Statement.Text != "Narrower wording" {
		t.Fatalf("unexpected edit output: %#v", output)
	}
	if output.Statement.ID != before.ID || output.Statement.Slug != before.Slug {
		t.Fatalf("edit changed stable identity: before %#v after %#v", before, output.Statement)
	}
	parsed := argfile.ParseFile(path)
	if got := parsed.Document.Statements[0]; got.ID != before.ID || got.Slug != before.Slug || got.Text != "Narrower wording" {
		t.Fatalf("saved edit = %#v", got)
	}
}

func TestFailedEditLeavesWorkspaceUnchanged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workspace.arg")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"init", path, "--title", "Title", "--text", "Original"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	err = run([]string{"edit", path, "missing", "--text", "Changed", "--json"}, &stdout, &stderr)
	if !errors.Is(err, errValidationFailed) {
		t.Fatalf("edit error = %v, want validation failure", err)
	}
	after, readErr := os.ReadFile(path)
	if readErr != nil || !bytes.Equal(before, after) {
		t.Fatalf("workspace changed after failed edit: %v", readErr)
	}
}
