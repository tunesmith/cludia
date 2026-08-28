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
	if value, _ := parsed.Document.MetadataValue(argument.NextIDsMetadataKey); value != "v1;P=3;L=1;C=1;CP=1;J=1" {
		t.Fatalf("next-id metadata = %q", value)
	}
}

func TestFocusedExplicitIDsMustBeCanonicalExactNext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workspace.arg")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"init", path, "--title", "Title", "--text", "First"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		id, code string
	}{
		{id: "custom-id", code: "statement_id_not_canonical"},
		{id: "L1", code: "statement_id_not_canonical"},
		{id: "P3", code: "id_not_next"},
		{id: "P1", code: "id_not_next"},
	} {
		before, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		stdout.Reset()
		stderr.Reset()
		err = run([]string{"add", path, "--id", test.id, "--text", "Rejected", "--json"}, &stdout, &stderr)
		if !errors.Is(err, errValidationFailed) {
			t.Fatalf("add %s error = %v", test.id, err)
		}
		var failure failureOutput
		if err := json.Unmarshal(stdout.Bytes(), &failure); err != nil || len(failure.Diagnostics) != 1 || failure.Diagnostics[0].Code != test.code {
			t.Fatalf("add %s failure = %#v, decode %v", test.id, failure, err)
		}
		after, readErr := os.ReadFile(path)
		if readErr != nil || !bytes.Equal(before, after) {
			t.Fatalf("rejected add %s changed file: %v", test.id, readErr)
		}
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"add", path, "--id", "P2", "--text", "Accepted", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("exact-next add: %v", err)
	}
}

func TestDeletionLeavesMonotonicGap(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workspace.arg")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"init", path, "--title", "Title", "--text", "First"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := run([]string{"add", path, "--text", "Second"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := run([]string{"delete", path, "P2"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := run([]string{"add", path, "--text", "Third", "--json"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	var output mutationOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil || output.Statement.ID != "P3" {
		t.Fatalf("post-deletion add = %#v, decode %v", output, err)
	}
}

func TestLegacyWorkspaceBootstrapsNextIDsOnFirstCreation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.arg")
	doc := &argument.Document{
		ID: "legacy", Title: "Legacy",
		Metadata: []argument.Metadata{{Key: "profile", Value: "workspace"}},
		Statements: []argument.Statement{
			{ID: "P1", Role: argument.RolePremise, Kind: argument.KindFact, Truth: argument.TruthTrue, Text: "First"},
			{ID: "P3", Role: argument.RolePremise, Kind: argument.KindFact, Truth: argument.TruthTrue, Text: "Third"},
			{ID: "custom-id", Role: argument.RolePremise, Kind: argument.KindFact, Truth: argument.TruthTrue, Text: "Custom"},
		},
	}
	if err := argfile.SaveAtomic(path, doc); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err := run([]string{"add", path, "--text", "Fourth", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("legacy add: %v", err)
	}
	var output mutationOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil || output.Statement.ID != "P4" {
		t.Fatalf("legacy output = %#v, decode %v", output, err)
	}
	parsed := argfile.ParseFile(path)
	if _, ok := parsed.Document.Statement("custom-id"); !ok {
		t.Fatal("ordinary mutation did not preserve existing custom id")
	}
	if value, _ := parsed.Document.MetadataValue(argument.NextIDsMetadataKey); value != "v1;P=5;L=1;C=1;CP=1;J=1" {
		t.Fatalf("bootstrapped next ids = %q", value)
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

func TestAddGeneratesValidSlugForDigitLeadingText(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workspace.arg")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"init", path, "--title", "Title", "--text", "First"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"add", path, "--text", "42061 blocks the migration", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("add: %v\nstderr: %s", err, stderr.String())
	}
	var output mutationOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if got, want := output.Statement.Slug, "statement-42061-blocks-migration"; got != want {
		t.Fatalf("generated slug = %q, want %q", got, want)
	}
	if !argument.ValidSlug(output.Statement.Slug) {
		t.Fatalf("generated slug %q is invalid", output.Statement.Slug)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	err = run([]string{"add", path, "--text", "Explicit invalid alias", "--slug", "42061-explicit", "--json"}, &stdout, &stderr)
	if !errors.Is(err, errValidationFailed) {
		t.Fatalf("explicit invalid slug error = %v, want validation failure", err)
	}
	after, readErr := os.ReadFile(path)
	if readErr != nil || !bytes.Equal(before, after) {
		t.Fatalf("workspace changed after explicit invalid slug: %v", readErr)
	}
}

func TestInitSeparatesDigitLeadingDocumentIDAndStatementSlugFallbacks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workspace.arg")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"init", path, "--title", "2026 migration", "--text", "42061 blocks the migration", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("init: %v\nstderr: %s", err, stderr.String())
	}
	var output mutationOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if got, want := output.Document.ID, "workspace-2026-migration"; got != want {
		t.Fatalf("document id = %q, want %q", got, want)
	}
	if got, want := output.Statement.Slug, "statement-42061-blocks-migration"; got != want {
		t.Fatalf("statement slug = %q, want %q", got, want)
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
	if err := run([]string{"edit", path, before.Slug, "--text", "Original wording.", "--same-proposition", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("edit: %v\nstderr: %s", err, stderr.String())
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatalf("decode edit JSON: %v", err)
	}
	assertExactKeys(t, raw, "schema_version", "action", "dry_run", "profile", "document", "statement", "previous_statement", "same_proposition", "changes", "diagnostics")
	var output mutationOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode typed edit JSON: %v", err)
	}
	if output.PreviousStatement == nil || output.PreviousStatement.Text != "Original wording" || output.Statement.Text != "Original wording." {
		t.Fatalf("unexpected edit output: %#v", output)
	}
	if output.SameProposition == nil || !*output.SameProposition {
		t.Fatalf("same-proposition intent not reported: %#v", output)
	}
	if output.Statement.ID != before.ID || output.Statement.Slug != before.Slug {
		t.Fatalf("edit changed stable identity: before %#v after %#v", before, output.Statement)
	}
	parsed := argfile.ParseFile(path)
	if got := parsed.Document.Statements[0]; got.ID != before.ID || got.Slug != before.Slug || got.Text != "Original wording." {
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
	err = run([]string{"edit", path, "missing", "--text", "Changed", "--same-proposition", "--json"}, &stdout, &stderr)
	if !errors.Is(err, errValidationFailed) {
		t.Fatalf("edit error = %v, want validation failure", err)
	}
	after, readErr := os.ReadFile(path)
	if readErr != nil || !bytes.Equal(before, after) {
		t.Fatalf("workspace changed after failed edit: %v", readErr)
	}
}

func TestEditChangesTruthAndKindWithoutChangingIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workspace.arg")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"init", path, "--title", "Title", "--text", "Original"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	before := argfile.ParseFile(path).Document.Statements[0]
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"edit", path, "P1", "--truth", "F", "--kind", "value", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("edit truth/kind: %v", err)
	}
	var output mutationOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if output.Statement.ID != before.ID || output.Statement.Slug != before.Slug || output.Statement.Text != before.Text || output.Statement.Truth != argument.TruthFalse || output.Statement.Kind != argument.KindValue {
		t.Fatalf("edited statement = %#v, before %#v", output.Statement, before)
	}
}

func TestEditRejectsExplicitTrueLemma(t *testing.T) {
	path := twoPremiseWorkspace(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"derive", path, "--source", "P1", "--source", "P2", "--target-text", "Lemma"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	err = run([]string{"edit", path, "L1", "--truth", "T", "--json"}, &stdout, &stderr)
	if !errors.Is(err, errValidationFailed) {
		t.Fatalf("edit lemma truth error = %v", err)
	}
	after, readErr := os.ReadFile(path)
	if readErr != nil || !bytes.Equal(before, after) {
		t.Fatalf("workspace changed after invalid truth edit: %v", readErr)
	}
}

func TestTextEditRequiresSamePropositionIntent(t *testing.T) {
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
	err = run([]string{"edit", path, "P1", "--text", "Changed", "--json"}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "--same-proposition") {
		t.Fatalf("edit error = %v, want same-proposition guidance", err)
	}
	after, readErr := os.ReadFile(path)
	if readErr != nil || !bytes.Equal(before, after) {
		t.Fatalf("workspace changed without continuity intent: %v", readErr)
	}
}
