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
)

func TestTopJSONAndHumanContract(t *testing.T) {
	path := textViewWorkspace(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"top", "--json", path}, &stdout, &stderr); err != nil {
		t.Fatalf("top json: %v\nstderr: %s", err, stderr.String())
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	assertExactKeys(t, raw, "schema_version", "profile", "evaluation", "document", "statements", "diagnostics")
	var output topOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if len(output.Statements) != 2 || output.Statements[0].Statement.ID != "L2" || output.Statements[0].Depth != 2 || !output.Statements[0].Challenged || output.Statements[1].Statement.ID != "P5" {
		t.Fatalf("top output = %#v", output)
	}

	stdout.Reset()
	stderr.Reset()
	t.Setenv("COLUMNS", "84")
	if err := run([]string{"top", path}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	human := stdout.String()
	flat := strings.Join(strings.Fields(human), " ")
	for _, want := range []string{"LABEL", "TRUTH", "DEPTH", "STATEMENT", "L2!", "A deliberately long final statement whose complete text must wrap without being summarized or omitted", "P5!"} {
		if !strings.Contains(flat, want) && !strings.Contains(human, want) {
			t.Fatalf("top human missing %q:\n%s", want, human)
		}
	}
	if strings.Contains(human, "derived") || strings.Contains(human, "...") {
		t.Fatalf("top human truncated text:\n%s", human)
	}
}

func TestTopFiltersAndPaginatesMatchingStatements(t *testing.T) {
	path := textViewWorkspace(t)
	parsed := argfile.Load(path)
	parsed.Document.Statements = append(parsed.Document.Statements, argument.Statement{
		ID: "P6", Slug: "unchallenged", Text: "Unchallenged top statement",
		Role: argument.RolePremise, Kind: argument.KindFact, Truth: argument.TruthTrue,
	})
	if err := argfile.SaveAtomic(path, parsed.Document); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if err := run([]string{"top", path, "--challenged", "--offset", "1", "--limit", "1", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("filtered top: %v\nstderr: %s", err, stderr.String())
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	assertExactKeys(t, raw, "schema_version", "profile", "evaluation", "document", "statements", "diagnostics")
	var output topOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if len(output.Statements) != 1 || output.Statements[0].Statement.ID != "P5" || !output.Statements[0].Challenged {
		t.Fatalf("filtered top output = %#v", output.Statements)
	}

	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"top", "--limit", "1", path, "--json"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if len(output.Statements) != 1 || output.Statements[0].Statement.ID != "L2" {
		t.Fatalf("limited top output = %#v", output.Statements)
	}
}

func TestTopRejectsNegativePagination(t *testing.T) {
	path := textViewWorkspace(t)
	for _, args := range [][]string{{"top", path, "--limit", "-1"}, {"top", path, "--offset", "-1"}} {
		var stdout, stderr bytes.Buffer
		if err := run(args, &stdout, &stderr); err == nil {
			t.Fatalf("%v unexpectedly succeeded", args)
		}
	}
}

func TestLedgerJSONAndHumanContract(t *testing.T) {
	path := textViewWorkspace(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"ledger", path, "final", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("ledger json: %v\nstderr: %s", err, stderr.String())
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	assertExactKeys(t, raw, "schema_version", "profile", "evaluation", "document", "root", "rows", "diagnostics")
	var output ledgerOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if output.Root != "L2" || len(output.Rows) != 6 || len(output.Rows[4].Derivations) != 2 || len(output.Rows[5].Derivations) != 2 {
		t.Fatalf("ledger output = %#v", output)
	}

	stdout.Reset()
	stderr.Reset()
	t.Setenv("COLUMNS", "100")
	if err := run([]string{"ledger", path, "L2"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	human := stdout.String()
	flat := strings.Join(strings.Fields(human), " ")
	for _, want := range []string{"LABEL", "TRUTH", "STATEMENT", "DERIVATION", "AND(P1, P2)", "OR(P3, P4)", "AND(P2) [direct]", "A deliberately long final statement whose complete text", "must wrap without being summarized or omitted"} {
		if !strings.Contains(flat, want) && !strings.Contains(human, want) {
			t.Fatalf("ledger human missing %q:\n%s", want, human)
		}
	}
	if strings.Contains(human, "J1") || strings.Contains(human, "derived") || strings.Contains(human, "justified by") || strings.Contains(human, "...") {
		t.Fatalf("ledger human exposed forbidden notation:\n%s", human)
	}
}

func TestLedgerRejectsCounterpointRoot(t *testing.T) {
	path := textViewWorkspace(t)
	var stdout, stderr bytes.Buffer
	err := run([]string{"ledger", path, "CP1", "--json"}, &stdout, &stderr)
	if !errors.Is(err, errValidationFailed) {
		t.Fatalf("ledger counterpoint error = %v", err)
	}
	var failure failureOutput
	if err := json.Unmarshal(stdout.Bytes(), &failure); err != nil {
		t.Fatal(err)
	}
	if len(failure.Diagnostics) != 1 || failure.Diagnostics[0].Code != "ledger_root_invalid" {
		t.Fatalf("diagnostics = %#v", failure.Diagnostics)
	}
}

func textViewWorkspace(t *testing.T) string {
	t.Helper()
	statement := func(id, slug, text string, role argument.Role) argument.Statement {
		truth := argument.TruthUnknown
		if role == argument.RolePremise || role == argument.RoleCounterpoint {
			truth = argument.TruthTrue
		}
		return argument.Statement{ID: id, Slug: slug, Text: text, Role: role, Kind: argument.KindFact, Truth: truth}
	}
	doc := &argument.Document{
		ID: "text-views", Title: "Text Views", Metadata: []argument.Metadata{{Key: "profile", Value: "workspace"}},
		Statements: []argument.Statement{
			statement("P1", "one", "First source statement", argument.RolePremise),
			statement("P2", "two", "Second source statement", argument.RolePremise),
			statement("P3", "three", "Third source statement", argument.RolePremise),
			statement("P4", "four", "Fourth source statement", argument.RolePremise),
			statement("L1", "middle", "Intermediate statement", argument.RoleLemma),
			statement("L2", "final", "A deliberately long final statement whose complete text must wrap without being summarized or omitted", argument.RoleLemma),
			statement("P5", "isolated", "Isolated challenged statement", argument.RolePremise),
			statement("CP1", "challenge", "Challenge to isolated statement", argument.RoleCounterpoint),
			statement("CP2", "answer", "Counterpoint to the challenge", argument.RoleCounterpoint),
			statement("CP3", "undercut", "Challenge to final inference", argument.RoleCounterpoint),
			statement("CP4", "source-challenge", "Challenge to the direct-support source", argument.RoleCounterpoint),
			statement("CP5", "active-isolated-challenge", "Active challenge to the isolated statement", argument.RoleCounterpoint),
		},
		Junctors: []argument.Junctor{
			{ID: "J1", Connector: argument.ConnectorAND, Sources: []string{"P1", "P2"}, Target: "L1"},
			{ID: "J2", Connector: argument.ConnectorOR, Sources: []string{"P3", "P4"}, Target: "L1"},
			{ID: "J3", Connector: argument.ConnectorAND, Sources: []string{"L1", "P4"}, Target: "L2"},
		},
		DirectSupports: []argument.DirectSupport{{Source: "P2", Target: "L2", Connector: argument.ConnectorAND}},
		Defeats: []argument.Defeat{
			{From: "CP1", Scope: argument.DefeatPremise, To: "P5"},
			{From: "CP2", Scope: argument.DefeatCounterpoint, To: "CP1"},
			{From: "CP3", Scope: argument.DefeatInference, JunctorID: "J3", AtTarget: "L2"},
			{From: "CP4", Scope: argument.DefeatPremise, To: "P2"},
			{From: "CP5", Scope: argument.DefeatPremise, To: "P5"},
		},
	}
	path := filepath.Join(t.TempDir(), "workspace.arg")
	if err := argfile.SaveAtomic(path, doc); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	return path
}
