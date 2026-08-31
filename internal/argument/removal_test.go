// SPDX-FileCopyrightText: 2026 KeenWorks
// SPDX-License-Identifier: GPL-3.0-or-later

package argument

import "testing"

func TestRemoveCounterpointRemovesOwnedDefeatOnClone(t *testing.T) {
	doc := removalDocument()
	next, result, err := RemoveCounterpoint(doc, "challenge")
	if err != nil || result.Counterpoint.ID != "CP1" || len(result.DefeatsRemoved) != 1 || len(next.Statements) != len(doc.Statements)-1 || len(doc.Defeats) != 1 {
		t.Fatalf("remove counterpoint = %#v, %v", result, err)
	}
}

func TestRemoveCounterpointRejectsDependentChain(t *testing.T) {
	doc := removalDocument()
	doc.Statements = append(doc.Statements, Statement{ID: "CP2", Role: RoleCounterpoint, Kind: KindFact, Truth: TruthTrue, Text: "Reply"})
	doc.Defeats = append(doc.Defeats, Defeat{From: "CP2", Scope: DefeatCounterpoint, To: "CP1"})
	_, _, err := RemoveCounterpoint(doc, "CP1")
	mutationErr, ok := err.(*MutationError)
	if !ok || mutationErr.Code != "counterpoint_has_dependents" || len(doc.Defeats) != 2 {
		t.Fatalf("error = %#v", err)
	}
}

func TestDeleteStatementRemovesIncidentSupportAndPreservesCaller(t *testing.T) {
	doc := removalDocument()
	next, result, err := DeleteStatement(doc, "P2")
	if err != nil || result.Statement.ID != "P2" || len(result.JunctorsRemoved) != 1 || len(result.DirectSupportsRemoved) != 1 {
		t.Fatalf("delete = %#v, %v", result, err)
	}
	if len(next.Junctors) != 0 || len(next.DirectSupports) != 0 || len(doc.Junctors) != 1 || len(doc.DirectSupports) != 1 {
		t.Fatalf("next/caller relations = %d/%d %d/%d", len(next.Junctors), len(doc.Junctors), len(next.DirectSupports), len(doc.DirectSupports))
	}
}

func TestDeleteStatementRejectsAttachedDefeatWithoutMutation(t *testing.T) {
	doc := removalDocument()
	_, _, err := DeleteStatement(doc, "P1")
	mutationErr, ok := err.(*MutationError)
	if !ok || mutationErr.Code != "statement_has_defeats" || len(doc.Statements) != 5 {
		t.Fatalf("error = %#v", err)
	}
}

func removalDocument() *Document {
	return &Document{
		ID: "remove", Title: "Remove",
		Metadata: []Metadata{{Key: "profile", Value: "cludia"}, {Key: NextIDsMetadataKey, Value: "v1;P=4;L=2;C=1;CP=2;J=2"}},
		Statements: []Statement{
			{ID: "P1", Role: RolePremise, Kind: KindFact, Truth: TruthTrue, Text: "One"},
			{ID: "P2", Role: RolePremise, Kind: KindFact, Truth: TruthTrue, Text: "Two"},
			{ID: "P3", Role: RolePremise, Kind: KindFact, Truth: TruthTrue, Text: "Three"},
			{ID: "L1", Role: RoleLemma, Kind: KindFact, Truth: TruthUnknown, Text: "Target"},
			{ID: "CP1", Slug: "challenge", Role: RoleCounterpoint, Kind: KindFact, Truth: TruthTrue, Text: "Challenge"},
		},
		Junctors:       []Junctor{{ID: "J1", Connector: ConnectorAND, Sources: []string{"P1", "P2"}, Target: "L1"}},
		DirectSupports: []DirectSupport{{Source: "P2", Target: "P3", Connector: ConnectorAND}},
		Defeats:        []Defeat{{From: "CP1", Scope: DefeatPremise, To: "P1"}},
	}
}
