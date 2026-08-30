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

	"github.com/tunesmith/cludia/internal/argfile"
	"github.com/tunesmith/cludia/internal/argument"
	"github.com/tunesmith/cludia/internal/query"
)

func TestMoveStatementJSONPersistsGeneralOrderWithoutChangingContent(t *testing.T) {
	path := statementMoveWorkspace(t)
	before := argfile.Load(path).Document
	allocatorBefore, _ := before.MetadataValue(argument.NextIDsMetadataKey)

	var stdout, stderr bytes.Buffer
	if err := run([]string{"move-statement", path, "third", "--before", "first", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("move-statement: %v\nstderr: %s", err, stderr.String())
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	assertExactKeys(t, raw, "schema_version", "action", "profile", "document", "statement", "anchor", "placement", "previous_position", "current_position", "changes", "diagnostics")
	var output statementMoveOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if output.Statement.ID != "P3" || output.Anchor.ID != "P1" || output.Placement != argument.MoveBefore || output.PreviousPosition != 3 || output.CurrentPosition != 1 || len(output.Changes) != 1 {
		t.Fatalf("output = %#v", output)
	}

	after := argfile.Load(path).Document
	if got, want := moveStatementIDs(after), []string{"P3", "P1", "P2", "CP1"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("saved order = %v, want %v", got, want)
	}
	allocatorAfter, _ := after.MetadataValue(argument.NextIDsMetadataKey)
	clearRelationOrder(before)
	clearRelationOrder(after)
	if allocatorAfter != allocatorBefore || !reflect.DeepEqual(after.Junctors, before.Junctors) || !reflect.DeepEqual(after.DirectSupports, before.DirectSupports) || !reflect.DeepEqual(after.Defeats, before.Defeats) {
		t.Fatalf("move changed non-order content\nbefore=%#v\nafter=%#v", before, after)
	}
}

func TestMoveStatementSupportsCounterpointsNoOpAndHumanOutput(t *testing.T) {
	path := statementMoveWorkspace(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"move-statement", "--after", "challenge", path, "first"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Moved P1 after CP1") {
		t.Fatalf("human output = %q", stdout.String())
	}

	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"move-statement", path, "P1", "--after", "CP1", "--json"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	var output statementMoveOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if len(output.Changes) != 0 || output.PreviousPosition != output.CurrentPosition {
		t.Fatalf("no-op output = %#v", output)
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("no-op rewrote file: %v", err)
	}
}

func TestMoveStatementStructuredFailuresDoNotWrite(t *testing.T) {
	tests := []struct {
		name string
		args func(string) []string
		code string
	}{
		{name: "missing statement", args: func(path string) []string {
			return []string{"move-statement", path, "missing", "--before", "P1", "--json"}
		}, code: "statement_not_found"},
		{name: "missing anchor", args: func(path string) []string {
			return []string{"move-statement", path, "P1", "--after", "missing", "--json"}
		}, code: "statement_anchor_not_found"},
		{name: "same anchor", args: func(path string) []string {
			return []string{"move-statement", path, "P1", "--after", "first", "--json"}
		}, code: "statement_move_same_anchor"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := statementMoveWorkspace(t)
			before, _ := os.ReadFile(path)
			var stdout, stderr bytes.Buffer
			err := run(test.args(path), &stdout, &stderr)
			if !errors.Is(err, errValidationFailed) {
				t.Fatalf("error = %v", err)
			}
			var failure failureOutput
			if err := json.Unmarshal(stdout.Bytes(), &failure); err != nil {
				t.Fatal(err)
			}
			if len(failure.Diagnostics) != 1 || failure.Diagnostics[0].Code != test.code {
				t.Fatalf("failure = %#v", failure)
			}
			after, _ := os.ReadFile(path)
			if !bytes.Equal(before, after) {
				t.Fatal("failed move changed workspace")
			}
		})
	}
}

func TestMovedGeneralOrderFeedsQueriesExportAndRenumberPlan(t *testing.T) {
	doc := &argument.Document{
		ID: "general-order", Title: "General Order", Metadata: []argument.Metadata{
			{Key: "profile", Value: "workspace"},
			{Key: argument.NextIDsMetadataKey, Value: "v1;P=4;L=2;C=1;CP=1;J=2"},
		},
		Statements: []argument.Statement{
			{ID: "P1", Role: argument.RolePremise, Kind: argument.KindFact, Truth: argument.TruthTrue, Text: "Common first"},
			{ID: "P2", Role: argument.RolePremise, Kind: argument.KindFact, Truth: argument.TruthTrue, Text: "Common second"},
			{ID: "L1", Role: argument.RoleLemma, Kind: argument.KindFact, Truth: argument.TruthUnknown, Text: "Common root"},
			{ID: "P3", Role: argument.RolePremise, Kind: argument.KindFact, Truth: argument.TruthTrue, Text: "Common isolated"},
		},
		Junctors: []argument.Junctor{{ID: "J1", Connector: argument.ConnectorAND, Sources: []string{"P1", "P2"}, Target: "L1", Order: 1}},
	}
	path := filepath.Join(t.TempDir(), "general.arg")
	if err := argfile.SaveAtomic(path, doc); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	for _, args := range [][]string{
		{"move-statement", path, "P3", "--before", "P1", "--json"},
		{"move-statement", path, "P2", "--before", "P1", "--json"},
	} {
		stdout.Reset()
		stderr.Reset()
		if err := run(args, &stdout, &stderr); err != nil {
			t.Fatalf("move %v: %v\n%s", args, err, stderr.String())
		}
	}
	moved := argfile.Load(path).Document
	if got, want := moveStatementIDs(moved), []string{"P3", "P2", "P1", "L1"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("general order = %v, want %v", got, want)
	}
	if top := query.Top(moved); len(top) != 2 || top[0].Statement.ID != "P3" || top[1].Statement.ID != "L1" {
		t.Fatalf("Top did not observe general order: %#v", top)
	}
	if matches := query.SearchStatements(moved, "common"); len(matches) != 4 || matches[0].Statement.ID != "P3" || matches[1].Statement.ID != "P2" {
		t.Fatalf("search did not observe general order: %#v", matches)
	}
	if components := query.Components(moved); len(components) != 2 || components[0].Anchor != "P3" || components[1].Anchor != "P2" {
		t.Fatalf("components did not observe general order: %#v", components)
	}
	rooted, err := query.Rooted(moved, "L1")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := moveStatementIDs(rooted), []string{"P2", "P1", "L1"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("rooted order = %v, want %v", got, want)
	}
	_, plan, err := argument.RenumberDocument(moved)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.StatementIDs) != 4 || plan.StatementIDs[0].PreviousID != "P3" || plan.StatementIDs[0].CurrentID != "P1" || plan.StatementIDs[1].PreviousID != "P2" || plan.StatementIDs[2].PreviousID != "P1" || plan.StatementIDs[2].CurrentID != "P3" {
		t.Fatalf("renumber plan did not observe general order: %#v", plan.StatementIDs)
	}
}

func statementMoveWorkspace(t *testing.T) string {
	t.Helper()
	doc := &argument.Document{
		ID: "move", Title: "Move", Metadata: []argument.Metadata{
			{Key: "profile", Value: "workspace"},
			{Key: argument.NextIDsMetadataKey, Value: "v1;P=4;L=2;C=1;CP=2;J=2"},
		},
		Statements: []argument.Statement{
			{ID: "P1", Slug: "first", Role: argument.RolePremise, Kind: argument.KindFact, Truth: argument.TruthTrue, Text: "First\nmultiline"},
			{ID: "P2", Slug: "second", Role: argument.RolePremise, Kind: argument.KindFact, Truth: argument.TruthTrue, Text: "Second"},
			{ID: "P3", Slug: "third", Role: argument.RolePremise, Kind: argument.KindFact, Truth: argument.TruthTrue, Text: "Third"},
			{ID: "CP1", Slug: "challenge", Role: argument.RoleCounterpoint, Kind: argument.KindFact, Truth: argument.TruthTrue, Text: "Challenge"},
		},
		Junctors:       []argument.Junctor{{ID: "J1", Connector: argument.ConnectorOR, Sources: []string{"P1", "P2"}, Target: "P3", Order: 1}},
		DirectSupports: []argument.DirectSupport{{Source: "P1", Target: "P2", Connector: argument.ConnectorAND, Order: 2}},
		Defeats:        []argument.Defeat{{From: "CP1", Scope: argument.DefeatPremise, To: "P1"}},
	}
	path := filepath.Join(t.TempDir(), "workspace.arg")
	if err := argfile.SaveAtomic(path, doc); err != nil {
		t.Fatal(err)
	}
	return path
}

func moveStatementIDs(doc *argument.Document) []string {
	ids := make([]string, 0, len(doc.Statements))
	for _, statement := range doc.Statements {
		ids = append(ids, statement.ID)
	}
	return ids
}

func clearRelationOrder(doc *argument.Document) {
	for i := range doc.Junctors {
		doc.Junctors[i].Order = 0
	}
	for i := range doc.DirectSupports {
		doc.DirectSupports[i].Order = 0
	}
}
