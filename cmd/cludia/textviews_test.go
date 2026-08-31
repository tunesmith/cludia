// SPDX-FileCopyrightText: 2026 KeenWorks
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/tunesmith/cludia/internal/argfile"
	"github.com/tunesmith/cludia/internal/argument"
	"github.com/tunesmith/cludia/internal/query"
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
	for _, want := range []string{"LABEL", "∴", "DEPTH", "STATEMENT", "L2!", "⊬", "A deliberately long final statement whose complete text must wrap without being summarized or omitted", "P5!", "T → F"} {
		if !strings.Contains(flat, want) && !strings.Contains(human, want) {
			t.Fatalf("top human missing %q:\n%s", want, human)
		}
	}
	if strings.Contains(human, "TRUTH") || strings.Contains(human, "STATUS") || strings.Contains(human, "derived") || strings.Contains(human, "...") {
		t.Fatalf("top human truncated text:\n%s", human)
	}
	lines := strings.Split(human, "\n")
	header := lineContaining(lines, "∴")
	derived := lineContaining(lines, "L2!")
	if runeIndex(header, "∴") != runeIndex(derived, "⊬") {
		t.Fatalf("top status is not centered under ∴:\n%s", human)
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

func TestTopHumanUsesDiamondForUnknownLemmaWithoutStatusHeader(t *testing.T) {
	path := textViewWorkspace(t)
	parsed := argfile.Load(path)
	parsed.Document.Statements = append(parsed.Document.Statements, argument.Statement{
		ID: "L3", Role: argument.RoleLemma, Kind: argument.KindFact,
		Truth: argument.TruthUnknown, Text: "Unresolved derived statement",
	})
	if err := argfile.SaveAtomic(path, parsed.Document); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	t.Setenv("COLUMNS", "90")
	if err := run([]string{"top", path}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	human := stdout.String()
	lines := strings.Split(human, "\n")
	header := lineContaining(lines, "∴")
	lemma := lineContaining(lines, "L3")
	if runeIndex(header, "∴") != runeIndex(lemma, "◇") || strings.Contains(human, "TRUTH") || strings.Contains(human, "STATUS") {
		t.Fatalf("unknown lemma proof status missing:\n%s", human)
	}
}

func TestStatusCenteringUsesCompleteFixedWidth(t *testing.T) {
	if got := centerText("T", 5); got != "  T  " {
		t.Fatalf("centered truth = %q", got)
	}
	if got := centerText("T → F", 5); got != "T → F" {
		t.Fatalf("full-width transition = %q", got)
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
	for _, want := range []string{"LABEL", "∴", "STATEMENT", "DERIVATION", "⊢", "⊬", "AND(P1, P2)", "OR(P3, P4)", "AND(P2) [direct]", "A deliberately long final statement whose complete text", "must wrap without being summarized or omitted"} {
		if !strings.Contains(flat, want) && !strings.Contains(human, want) {
			t.Fatalf("ledger human missing %q:\n%s", want, human)
		}
	}
	if strings.Contains(human, "TRUTH") || strings.Contains(human, "STATUS") || strings.Contains(human, "J1") || strings.Contains(human, "derived") || strings.Contains(human, "justified by") || strings.Contains(human, "...") {
		t.Fatalf("ledger human exposed forbidden notation:\n%s", human)
	}
}

func TestLedgerSelectedInferenceNarrowsRootAndMarksTruthFromOtherRoutes(t *testing.T) {
	path := selectedInferenceWorkspace(t, false, true)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"ledger", path, "L1", "--inference", "J1", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("selected ledger JSON: %v\nstderr: %s", err, stderr.String())
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	assertExactKeys(t, raw, "schema_version", "profile", "evaluation", "document", "root", "selected_inference", "rows", "diagnostics")
	var selectedRaw map[string]json.RawMessage
	if err := json.Unmarshal(raw["selected_inference"], &selectedRaw); err != nil {
		t.Fatal(err)
	}
	assertExactKeys(t, selectedRaw, "junctor", "effective_truth", "disabled_by_undercut", "other_justifications_omitted", "other_routes_affect_truth")
	var output ledgerOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if output.SelectedInference == nil || output.SelectedInference.Junctor.ID != "J1" || output.SelectedInference.EffectiveTruth != argument.TruthFalse || output.SelectedInference.DisabledByUndercut || !output.SelectedInference.OtherJustificationsOmitted || !output.SelectedInference.OtherRoutesAffectTruth {
		t.Fatalf("selection = %#v", output.SelectedInference)
	}
	if got := ledgerRowIDs(output.Rows); !reflect.DeepEqual(got, []string{"P1", "P2", "L1"}) {
		t.Fatalf("selected rows = %#v", got)
	}
	if len(output.Rows[2].Derivations) != 1 || output.Rows[2].Derivations[0].ID != "J1" {
		t.Fatalf("root derivations = %#v", output.Rows[2].Derivations)
	}

	stdout.Reset()
	stderr.Reset()
	t.Setenv("COLUMNS", "100")
	if err := run([]string{"ledger", "--inference", "J1", path, "L1"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	human := stdout.String()
	if !strings.Contains(human, "⊢*") || !strings.Contains(human, "* proof comes from another justification not shown") {
		t.Fatalf("selected human ledger lacks compact truth note:\n%s", human)
	}
	lines := strings.Split(human, "\n")
	header, sourceLine, rootLine := lineContaining(lines, "STATEMENT"), lineContaining(lines, "First"), lineContaining(lines, "Target")
	statementColumn := runeIndex(header, "STATEMENT")
	if statementColumn < 0 || runeIndex(sourceLine, "First") != statementColumn || runeIndex(rootLine, "Target") != statementColumn {
		t.Fatalf("⊢* misaligned statement column %d:\n%s", statementColumn, human)
	}
}

func TestLedgerSelectedInferenceMarksAcceptedUndercutCompactly(t *testing.T) {
	path := selectedInferenceWorkspace(t, true, true)
	var stdout, stderr bytes.Buffer
	t.Setenv("COLUMNS", "100")
	if err := run([]string{"ledger", path, "L1", "--inference", "J1"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	human := stdout.String()
	if !strings.Contains(human, "⊢*") || !strings.Contains(human, "AND(P1, P2) [undercut]") || !strings.Contains(human, "* proof comes from another justification not shown") {
		t.Fatalf("undercut ledger =\n%s", human)
	}

	path = selectedInferenceWorkspace(t, true, false)
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"ledger", path, "L1", "--inference", "J1"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	human = stdout.String()
	if !strings.Contains(human, "L1!") || !strings.Contains(human, "⊬") || !strings.Contains(human, "AND(P1, P2) [undercut]") || strings.Contains(human, "⊬*") || strings.Contains(human, "proof comes from another") {
		t.Fatalf("sole undercut route ledger =\n%s", human)
	}
}

func TestLedgerSelectedInferenceReportsStableSelectionFailures(t *testing.T) {
	path := selectedInferenceWorkspace(t, false, true)
	for _, test := range []struct {
		id, code string
	}{{"missing", "ledger_inference_not_found"}, {"J3", "ledger_inference_target_mismatch"}} {
		var stdout, stderr bytes.Buffer
		err := run([]string{"ledger", path, "L1", "--inference", test.id, "--json"}, &stdout, &stderr)
		if !errors.Is(err, errValidationFailed) {
			t.Fatalf("%s error = %v", test.id, err)
		}
		var failure failureOutput
		if err := json.Unmarshal(stdout.Bytes(), &failure); err != nil || len(failure.Diagnostics) != 1 || failure.Diagnostics[0].Code != test.code {
			t.Fatalf("%s failure = %#v, err %v", test.id, failure, err)
		}
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
		ID: "text-views", Title: "Text Views", Metadata: []argument.Metadata{{Key: "profile", Value: "cludia"}},
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

func selectedInferenceWorkspace(t *testing.T, undercut, alternative bool) string {
	t.Helper()
	doc := &argument.Document{
		ID: "selected-ledger", Title: "Selected Ledger", Metadata: []argument.Metadata{{Key: "profile", Value: "cludia"}},
		Statements: []argument.Statement{
			{ID: "P1", Role: argument.RolePremise, Kind: argument.KindFact, Truth: argument.TruthTrue, Text: "First"},
			{ID: "P2", Role: argument.RolePremise, Kind: argument.KindFact, Truth: argument.TruthFalse, Text: "Second"},
			{ID: "P3", Role: argument.RolePremise, Kind: argument.KindFact, Truth: argument.TruthTrue, Text: "Third"},
			{ID: "P4", Role: argument.RolePremise, Kind: argument.KindFact, Truth: argument.TruthTrue, Text: "Fourth"},
			{ID: "L1", Role: argument.RoleLemma, Kind: argument.KindFact, Truth: argument.TruthUnknown, Text: "Target"},
			{ID: "P5", Role: argument.RolePremise, Kind: argument.KindFact, Truth: argument.TruthTrue, Text: "Other target source"},
			{ID: "L2", Role: argument.RoleLemma, Kind: argument.KindFact, Truth: argument.TruthUnknown, Text: "Other target"},
		},
		Junctors: []argument.Junctor{
			{ID: "J1", Connector: argument.ConnectorAND, Sources: []string{"P1", "P2"}, Target: "L1"},
			{ID: "J3", Connector: argument.ConnectorAND, Sources: []string{"P1", "P5"}, Target: "L2"},
		},
		DirectSupports: []argument.DirectSupport{}, Defeats: []argument.Defeat{},
	}
	if alternative {
		doc.Junctors = append(doc.Junctors, argument.Junctor{ID: "J2", Connector: argument.ConnectorAND, Sources: []string{"P3", "P4"}, Target: "L1"})
	}
	if undercut {
		doc.Statements[1].Truth = argument.TruthTrue
		doc.Statements = append(doc.Statements, argument.Statement{ID: "CP1", Role: argument.RoleCounterpoint, Kind: argument.KindFact, Truth: argument.TruthTrue, Text: "The first inference is undercut"})
		doc.Defeats = append(doc.Defeats, argument.Defeat{From: "CP1", Scope: argument.DefeatInference, JunctorID: "J1", AtTarget: "L1"})
	}
	path := filepath.Join(t.TempDir(), "selected-ledger.arg")
	if err := argfile.SaveAtomic(path, doc); err != nil {
		t.Fatal(err)
	}
	return path
}

func ledgerRowIDs(rows []query.LedgerRow) []string {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.Statement.ID)
	}
	return ids
}

func lineContaining(lines []string, fragment string) string {
	for _, line := range lines {
		if strings.Contains(line, fragment) {
			return line
		}
	}
	return ""
}

func runeIndex(value, fragment string) int {
	byteIndex := strings.Index(value, fragment)
	if byteIndex < 0 {
		return -1
	}
	return utf8.RuneCountInString(value[:byteIndex])
}
