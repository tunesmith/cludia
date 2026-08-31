// SPDX-FileCopyrightText: 2026 KeenWorks
// SPDX-License-Identifier: GPL-3.0-or-later

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
	"github.com/tunesmith/cludia/internal/validation"
)

func TestRenumberDryRunAndApplyRewriteCompleteWorkspace(t *testing.T) {
	path := complexRenumberWorkspace(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err := run([]string{"renumber", "--json", path, "--dry-run"}, &stdout, &stderr); err != nil {
		t.Fatalf("renumber dry-run: %v\nstderr: %s", err, stderr.String())
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	assertExactKeys(t, raw, "schema_version", "action", "dry_run", "applicable", "profile", "document", "statement_ids", "junctor_ids", "root_metadata_updated", "next_ids_before", "next_ids_after", "plan_token", "changes", "diagnostics")
	var plan renumberOutput
	if err := json.Unmarshal(stdout.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	if !plan.DryRun || !plan.Applicable || plan.PlanToken == "" || !plan.RootMetadataUpdated {
		t.Fatalf("plan = %#v", plan)
	}
	wantStatements := []statementIDMapping{
		{PreviousID: "old-p", CurrentID: "P1", Role: argument.RolePremise, Slug: "old-premise"},
		{PreviousID: "P9", CurrentID: "P2", Role: argument.RolePremise, Slug: "second-premise"},
		{PreviousID: "P8", CurrentID: "L1", Role: argument.RoleLemma, Slug: "promoted-lemma"},
		{PreviousID: "conclusion-x", CurrentID: "C1", Role: argument.RoleConclusion, Slug: "conclusion"},
		{PreviousID: "objection", CurrentID: "CP1", Role: argument.RoleCounterpoint, Slug: "objection"},
		{PreviousID: "reply", CurrentID: "CP2", Role: argument.RoleCounterpoint, Slug: "reply"},
		{PreviousID: "inference-objection", CurrentID: "CP3", Role: argument.RoleCounterpoint, Slug: "inference-objection"},
	}
	if len(plan.StatementIDs) != len(wantStatements) {
		t.Fatalf("statement mapping count = %d", len(plan.StatementIDs))
	}
	for index, want := range wantStatements {
		if got := plan.StatementIDs[index]; got != want {
			t.Fatalf("statement mapping %d = %#v, want %#v", index, got, want)
		}
	}
	if len(plan.JunctorIDs) != 2 || plan.JunctorIDs[0].PreviousID != "old-j" || plan.JunctorIDs[0].CurrentID != "J1" || plan.JunctorIDs[1].PreviousID != "J9" || plan.JunctorIDs[1].CurrentID != "J2" {
		t.Fatalf("junctor mappings = %#v", plan.JunctorIDs)
	}
	if plan.NextIDsAfter != (argument.NextIDs{P: 3, L: 2, C: 2, CP: 4, J: 3}) {
		t.Fatalf("next ids after = %#v", plan.NextIDsAfter)
	}
	assertDiagnostic(t, plan.Diagnostics, "external_id_references_unchecked")
	afterDryRun, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, afterDryRun) {
		t.Fatalf("dry-run changed workspace: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"renumber", path, "--apply-token", plan.PlanToken, "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("renumber apply: %v\nstderr: %s", err, stderr.String())
	}
	var applied renumberOutput
	if err := json.Unmarshal(stdout.Bytes(), &applied); err != nil || applied.DryRun {
		t.Fatalf("applied output = %#v, decode %v", applied, err)
	}
	parsed := argfile.ParseFile(path)
	if result := validation.Validate(parsed.Document, validation.ProfileCludia); !result.OK() {
		t.Fatalf("renumbered workspace invalid: %#v", result.Diagnostics)
	}
	if result := validation.Validate(parsed.Document, validation.ProfileConcludia); !result.OK() {
		t.Fatalf("renumbered workspace lost Concludia compatibility: %#v", result.Diagnostics)
	}
	for _, id := range []string{"P1", "P2", "L1", "C1", "CP1", "CP2", "CP3"} {
		if _, ok := parsed.Document.Statement(id); !ok {
			t.Fatalf("renumbered statement %s missing", id)
		}
	}
	if parsed.Document.Junctors[0].ID != "J1" || strings.Join(parsed.Document.Junctors[0].Sources, ",") != "P1,P2" || parsed.Document.Junctors[0].Target != "L1" {
		t.Fatalf("first junctor = %#v", parsed.Document.Junctors[0])
	}
	if parsed.Document.Junctors[1].ID != "J2" || strings.Join(parsed.Document.Junctors[1].Sources, ",") != "L1,P2" || parsed.Document.Junctors[1].Target != "C1" {
		t.Fatalf("second junctor = %#v", parsed.Document.Junctors[1])
	}
	if got := parsed.Document.DirectSupports[0]; got.Source != "P2" || got.Target != "L1" {
		t.Fatalf("direct support = %#v", got)
	}
	if parsed.Document.Defeats[0].From != "CP1" || parsed.Document.Defeats[0].To != "P1" || parsed.Document.Defeats[1].From != "CP2" || parsed.Document.Defeats[1].To != "CP1" || parsed.Document.Defeats[2].From != "CP3" || parsed.Document.Defeats[2].JunctorID != "J2" || parsed.Document.Defeats[2].AtTarget != "C1" {
		t.Fatalf("defeats = %#v", parsed.Document.Defeats)
	}
	if root, _ := parsed.Document.MetadataValue("root"); root != "C1" {
		t.Fatalf("root metadata = %q", root)
	}
	if note, _ := parsed.Document.MetadataValue("note"); note != "External text still says P9" {
		t.Fatalf("unrelated metadata changed: %q", note)
	}
	if nextIDs, _ := parsed.Document.MetadataValue(argument.NextIDsMetadataKey); nextIDs != "v1;P=3;L=2;C=2;CP=4;J=3" {
		t.Fatalf("next ids = %q", nextIDs)
	}
}

func TestRenumberRejectsStalePlanWithoutWriting(t *testing.T) {
	path := complexRenumberWorkspace(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"renumber", path, "--dry-run", "--json"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	var plan renumberOutput
	if err := json.Unmarshal(stdout.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"edit", path, "old-p", "--text", "First revised", "--same-proposition"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	err = run([]string{"renumber", path, "--apply-token", plan.PlanToken, "--json"}, &stdout, &stderr)
	if !errors.Is(err, errValidationFailed) {
		t.Fatalf("stale apply error = %v", err)
	}
	var failure failureOutput
	if err := json.Unmarshal(stdout.Bytes(), &failure); err != nil || len(failure.Diagnostics) != 1 || failure.Diagnostics[0].Code != "renumber_plan_stale" {
		t.Fatalf("stale failure = %#v, decode %v", failure, err)
	}
	after, readErr := os.ReadFile(path)
	if readErr != nil || !bytes.Equal(before, after) {
		t.Fatalf("stale apply changed workspace: %v", readErr)
	}
}

func TestRenumberNoOpHasCompleteMappingWithoutExternalWarning(t *testing.T) {
	path := twoPremiseWorkspace(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"renumber", path, "--dry-run", "--json"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	var output renumberOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if len(output.StatementIDs) != 2 || len(output.JunctorIDs) != 0 {
		t.Fatalf("no-op mappings = %#v %#v", output.StatementIDs, output.JunctorIDs)
	}
	for _, item := range output.Diagnostics {
		if item.Code == "external_id_references_unchecked" {
			t.Fatalf("no-op plan warned about external references: %#v", output.Diagnostics)
		}
	}
}

func TestRenumberHumanOutputContainsCompleteMappingAndToken(t *testing.T) {
	path := complexRenumberWorkspace(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"renumber", path, "--dry-run"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Renumber plan", "Statements:", "old-p -> P1", "Junctors:", "old-j -> J1", "plan token:", "dry-run: no file changes written"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("human output missing %q:\n%s", want, stdout.String())
		}
	}
}

func complexRenumberWorkspace(t *testing.T) string {
	t.Helper()
	statement := func(id, slug string, role argument.Role) argument.Statement {
		truth := argument.TruthUnknown
		if role == argument.RolePremise || role == argument.RoleCounterpoint {
			truth = argument.TruthTrue
		}
		return argument.Statement{ID: id, Slug: slug, Role: role, Kind: argument.KindFact, Truth: truth, Text: id + " text"}
	}
	doc := &argument.Document{
		ID: "renumber", Title: "Renumber",
		Metadata: []argument.Metadata{
			{Key: "profile", Value: "cludia"},
			{Key: "root", Value: "conclusion-x"},
			{Key: "note", Value: "External text still says P9"},
		},
		Statements: []argument.Statement{
			statement("old-p", "old-premise", argument.RolePremise),
			statement("P9", "second-premise", argument.RolePremise),
			statement("P8", "promoted-lemma", argument.RoleLemma),
			statement("conclusion-x", "conclusion", argument.RoleConclusion),
			statement("objection", "objection", argument.RoleCounterpoint),
			statement("reply", "reply", argument.RoleCounterpoint),
			statement("inference-objection", "inference-objection", argument.RoleCounterpoint),
		},
		Junctors: []argument.Junctor{
			{ID: "old-j", Connector: argument.ConnectorAND, Sources: []string{"old-p", "P9"}, Target: "P8", Order: 1},
			{ID: "J9", Connector: argument.ConnectorOR, Sources: []string{"P8", "P9"}, Target: "conclusion-x", Order: 2},
		},
		DirectSupports: []argument.DirectSupport{{Source: "P9", Target: "P8", Connector: argument.ConnectorAND, Order: 3}},
		Defeats: []argument.Defeat{
			{From: "objection", Scope: argument.DefeatPremise, To: "old-p"},
			{From: "reply", Scope: argument.DefeatCounterpoint, To: "objection"},
			{From: "inference-objection", Scope: argument.DefeatInference, JunctorID: "J9", AtTarget: "conclusion-x"},
		},
	}
	if result := validation.Validate(doc, validation.ProfileCludia); !result.OK() {
		t.Fatalf("test fixture invalid: %#v", result.Diagnostics)
	}
	path := filepath.Join(t.TempDir(), "renumber.arg")
	if err := argfile.SaveAtomic(path, doc); err != nil {
		t.Fatal(err)
	}
	return path
}

func assertDiagnostic(t *testing.T, diagnostics []diagnostic.Diagnostic, code string) {
	t.Helper()
	for _, item := range diagnostics {
		if item.Code == code {
			return
		}
	}
	t.Fatalf("diagnostic %s missing from %#v", code, diagnostics)
}
