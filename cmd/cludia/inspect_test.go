// SPDX-FileCopyrightText: 2026 KeenWorks
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/tunesmith/cludia/internal/argfile"
	"github.com/tunesmith/cludia/internal/argument"
	"github.com/tunesmith/cludia/internal/evaluation"
)

func TestListIsolatedFindsDisconnectedExampleStatement(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "broken-window-workspace.arg")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"list", path, "--state", "isolated", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("list: %v\nstderr: %s", err, stderr.String())
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatalf("decode list JSON: %v", err)
	}
	assertExactKeys(t, raw, "schema_version", "profile", "evaluation", "state", "statements", "junctors", "direct_supports", "defeats", "diagnostics")
	var output listOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode typed list JSON: %v", err)
	}
	if len(output.Statements) != 1 || output.Statements[0].ID != "P4" || !output.Statements[0].Isolated {
		t.Fatalf("isolated statements = %#v", output.Statements)
	}
	if output.Statements[0].EffectiveTruth != argument.TruthTrue || output.Statements[0].TruthSource != evaluation.TruthAsserted {
		t.Fatalf("isolated evaluation = %#v", output.Statements[0])
	}
	if output.Junctors == nil || output.DirectSupports == nil || output.Defeats == nil {
		t.Fatalf("filtered relation arrays must be empty, not null: %#v", output)
	}
}

func TestShowGivesJunctorIDPrecedenceOverStatementSlug(t *testing.T) {
	path := referenceCollisionWorkspace(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"show", path, "shared", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("show collision: %v\n%s", err, stderr.String())
	}
	var output showOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if output.ElementType != "junctor" || output.Junctor == nil || output.Junctor.ID != "shared" {
		t.Fatalf("resolved output = %#v", output)
	}
}

func referenceCollisionWorkspace(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "collision.arg")
	doc := &argument.Document{
		ID: "collision", Title: "Collision",
		Metadata: []argument.Metadata{
			{Key: "profile", Value: "cludia"},
			{Key: argument.NextIDsMetadataKey, Value: "v1;P=4;L=2;C=1;CP=1;J=1"},
		},
		Statements: []argument.Statement{
			{ID: "p1", Slug: "shared", Role: argument.RolePremise, Kind: argument.KindFact, Truth: argument.TruthTrue, Text: "Isolated slug owner"},
			{ID: "P2", Role: argument.RolePremise, Kind: argument.KindFact, Truth: argument.TruthTrue, Text: "First source"},
			{ID: "P3", Role: argument.RolePremise, Kind: argument.KindFact, Truth: argument.TruthTrue, Text: "Second source"},
			{ID: "L1", Role: argument.RoleLemma, Kind: argument.KindFact, Truth: argument.TruthUnknown, Text: "Target"},
		},
		Junctors:       []argument.Junctor{{ID: "shared", Connector: argument.ConnectorAND, Sources: []string{"P2", "P3"}, Target: "L1", Order: 1}},
		DirectSupports: []argument.DirectSupport{}, Defeats: []argument.Defeat{},
	}
	if err := argfile.SaveAtomic(path, doc); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestShowBySlugIncludesStatementRelations(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "broken-window-workspace.arg")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"show", "--json", path, "no-returning-prints", "--relations"}, &stdout, &stderr); err != nil {
		t.Fatalf("show: %v\nstderr: %s", err, stderr.String())
	}
	var output showOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode show JSON: %v", err)
	}
	if output.ElementType != "statement" || output.Statement == nil || output.Statement.ID != "P2" {
		t.Fatalf("unexpected statement output: %#v", output)
	}
	if output.Relations == nil || len(output.Relations.OutgoingSupport) != 2 || len(output.Relations.DefeatsTargeting) != 1 {
		t.Fatalf("unexpected relations: %#v", output.Relations)
	}
}

func TestShowHumanUsesProofStatusForLemma(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "broken-window-workspace.arg")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"show", path, "L1"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	human := stdout.String()
	if !bytes.Contains([]byte(human), []byte("proof ⊢")) || bytes.Contains([]byte(human), []byte("truth T")) {
		t.Fatalf("lemma human status:\n%s", human)
	}
}

func TestShowJunctorIncludesUndercut(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "broken-window-workspace.arg")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"show", path, "J1", "--relations", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("show junctor: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatalf("decode show JSON: %v", err)
	}
	assertExactKeys(t, raw, "schema_version", "profile", "evaluation", "element_type", "junctor", "relations", "diagnostics")
	var output showOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode typed show JSON: %v", err)
	}
	if output.ElementType != "junctor" || output.Junctor == nil || output.Relations == nil || len(output.Relations.DefeatsTargeting) != 1 {
		t.Fatalf("unexpected junctor output: %#v", output)
	}
	if output.Junctor.EffectiveTruth != argument.TruthTrue {
		t.Fatalf("junctor evaluation = %#v", output.Junctor)
	}
}
