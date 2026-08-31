// SPDX-FileCopyrightText: 2026 KeenWorks
// SPDX-License-Identifier: GPL-3.0-or-later

package query

import (
	"path/filepath"
	"testing"

	"github.com/tunesmith/cludia/internal/argfile"
	"github.com/tunesmith/cludia/internal/argument"
	"github.com/tunesmith/cludia/internal/diagnostic"
	"github.com/tunesmith/cludia/internal/validation"
)

func TestRootedIncludesAllJustificationsDefeatsAndExcludesUnrelatedIsland(t *testing.T) {
	parsed := argfile.ParseFile(filepath.Join("..", "..", "examples", "broken-window-workspace.arg"))
	if diagnostic.HasErrors(parsed.Diagnostics) {
		t.Fatalf("parse: %#v", parsed.Diagnostics)
	}
	rooted, err := Rooted(parsed.Document, "crossed-wall")
	if err != nil {
		t.Fatalf("Rooted: %v", err)
	}
	if len(rooted.Statements) != 8 || len(rooted.Junctors) != 2 || len(rooted.Defeats) != 4 {
		t.Fatalf("rooted counts: statements %d, junctors %d, defeats %d", len(rooted.Statements), len(rooted.Junctors), len(rooted.Defeats))
	}
	if _, found := rooted.Statement("P4"); found {
		t.Fatal("unrelated P4 included in rooted closure")
	}
	root, found := rooted.Statement("L1")
	if !found || root.Role != argument.RoleConclusion {
		t.Fatalf("root = %#v, found %t", root, found)
	}
	if profile, exists := rooted.MetadataValue("profile"); exists {
		t.Fatalf("workspace profile retained in export: %q", profile)
	}
	if rootMetadata, exists := rooted.MetadataValue("root"); !exists || rootMetadata != "crossed-wall" {
		t.Fatalf("root metadata = %q, exists %t", rootMetadata, exists)
	}
	if result := validation.Validate(rooted, validation.ProfileConcludia); !result.OK() || len(result.Diagnostics) != 0 {
		t.Fatalf("strict validation: %#v", result.Diagnostics)
	}
}

func TestRootedDoesNotFollowDownstreamSupport(t *testing.T) {
	doc := &argument.Document{
		ID: "downstream", Title: "Downstream",
		Statements: []argument.Statement{
			rootedStatement("P1", argument.RolePremise), rootedStatement("P2", argument.RolePremise),
			rootedStatement("L1", argument.RoleLemma), rootedStatement("P3", argument.RolePremise),
			rootedStatement("C1", argument.RoleConclusion),
		},
		Junctors: []argument.Junctor{
			{ID: "J1", Connector: argument.ConnectorAND, Sources: []string{"P1", "P2"}, Target: "L1"},
			{ID: "J2", Connector: argument.ConnectorAND, Sources: []string{"L1", "P3"}, Target: "C1"},
		},
	}
	rooted, err := Rooted(doc, "L1")
	if err != nil {
		t.Fatal(err)
	}
	if len(rooted.Statements) != 3 || len(rooted.Junctors) != 1 {
		t.Fatalf("rooted = %#v", rooted)
	}
	if _, found := rooted.Statement("C1"); found {
		t.Fatal("downstream conclusion included")
	}
}

func TestRootedOmitsCludiaNextIDMetadata(t *testing.T) {
	doc := &argument.Document{
		ID: "allocator-metadata", Title: "Allocator metadata",
		Metadata: []argument.Metadata{
			{Key: "author", Value: "test"},
			{Key: argument.NextIDsMetadataKey, Value: "v1;P=3;L=2;C=1;CP=1;J=2"},
		},
		Statements: []argument.Statement{
			rootedStatement("P1", argument.RolePremise),
			rootedStatement("P2", argument.RolePremise),
			rootedStatement("L1", argument.RoleLemma),
		},
		Junctors: []argument.Junctor{{ID: "J1", Connector: argument.ConnectorAND, Sources: []string{"P1", "P2"}, Target: "L1"}},
	}
	rooted, err := Rooted(doc, "L1")
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := rooted.MetadataValue(argument.NextIDsMetadataKey); exists {
		t.Fatal("rooted export retained Cludia allocator metadata")
	}
	if author, exists := rooted.MetadataValue("author"); !exists || author != "test" {
		t.Fatalf("unrelated metadata was not preserved: %q, %t", author, exists)
	}
}

func TestRootedFixedPointIncludesCounterpointSupportAndRecursiveDefeat(t *testing.T) {
	doc := &argument.Document{
		ID: "counterpoint-support", Title: "Counterpoint support",
		Statements: []argument.Statement{
			rootedStatement("P1", argument.RolePremise), rootedStatement("P2", argument.RolePremise),
			rootedStatement("L1", argument.RoleLemma), rootedStatement("P3", argument.RolePremise),
			rootedStatement("P4", argument.RolePremise), rootedStatement("CP1", argument.RoleCounterpoint),
			rootedStatement("CP2", argument.RoleCounterpoint),
		},
		Junctors: []argument.Junctor{
			{ID: "J1", Connector: argument.ConnectorAND, Sources: []string{"P1", "P2"}, Target: "L1"},
			{ID: "J2", Connector: argument.ConnectorAND, Sources: []string{"P3", "P4"}, Target: "CP1"},
		},
		Defeats: []argument.Defeat{
			{From: "CP1", Scope: argument.DefeatPremise, To: "P1"},
			{From: "CP2", Scope: argument.DefeatCounterpoint, To: "CP1"},
		},
	}
	rooted, err := Rooted(doc, "L1")
	if err != nil {
		t.Fatal(err)
	}
	if len(rooted.Statements) != 7 || len(rooted.Junctors) != 2 || len(rooted.Defeats) != 2 {
		t.Fatalf("fixed-point closure = %#v", rooted)
	}
}

func TestRootedRejectsCounterpointRoot(t *testing.T) {
	doc := &argument.Document{Statements: []argument.Statement{rootedStatement("CP1", argument.RoleCounterpoint)}}
	if _, err := Rooted(doc, "CP1"); err == nil {
		t.Fatal("counterpoint root accepted")
	}
}

func rootedStatement(id string, role argument.Role) argument.Statement {
	truth := argument.TruthUnknown
	if role == argument.RolePremise || role == argument.RoleCounterpoint {
		truth = argument.TruthTrue
	}
	return argument.Statement{ID: id, Role: role, Kind: argument.KindFact, Truth: truth, Text: id}
}
