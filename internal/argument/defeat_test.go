// SPDX-FileCopyrightText: 2026 KeenWorks
// SPDX-License-Identifier: GPL-3.0-or-later

package argument

import "testing"

func TestAddDefeatCreatesEveryFocusedScopeWithoutMutatingCaller(t *testing.T) {
	tests := []struct {
		name       string
		scope      DefeatScope
		target     string
		wantTo     string
		wantJ      string
		wantTarget string
	}{
		{name: "premise", scope: DefeatPremise, target: "first", wantTo: "P1"},
		{name: "inference", scope: DefeatInference, target: "J1", wantJ: "J1", wantTarget: "L1"},
		{name: "counterpoint", scope: DefeatCounterpoint, target: "existing-challenge", wantTo: "CP1"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			doc := addDefeatDocument()
			next, result, err := AddDefeat(doc, AddDefeatOptions{
				Scope: test.scope, TargetRef: test.target, Text: "New challenge",
				Truth: TruthUnknown, Kind: KindValue,
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.Counterpoint.ID != "CP2" || result.Counterpoint.Role != RoleCounterpoint || result.Counterpoint.Truth != TruthUnknown || result.Counterpoint.Kind != KindValue {
				t.Fatalf("counterpoint = %#v", result.Counterpoint)
			}
			if result.Defeat.From != "CP2" || result.Defeat.Scope != test.scope || result.Defeat.To != test.wantTo || result.Defeat.JunctorID != test.wantJ || result.Defeat.AtTarget != test.wantTarget {
				t.Fatalf("defeat = %#v", result.Defeat)
			}
			if len(next.Statements) != len(doc.Statements)+1 || len(next.Defeats) != len(doc.Defeats)+1 {
				t.Fatalf("next document = %#v", next)
			}
			if len(doc.Statements) != 4 || len(doc.Defeats) != 1 {
				t.Fatal("AddDefeat mutated the caller's document")
			}
			if value, _ := next.MetadataValue(NextIDsMetadataKey); value != "v1;P=3;L=2;C=1;CP=3;J=2" {
				t.Fatalf("next IDs = %q", value)
			}
		})
	}
}

func TestAddDefeatReturnsTypedRoleFailureAndUnchangedCaller(t *testing.T) {
	doc := addDefeatDocument()
	_, _, err := AddDefeat(doc, AddDefeatOptions{
		Scope: DefeatPremise, TargetRef: "L1", Text: "Wrong target", Truth: TruthTrue, Kind: KindFact,
	})
	addErr, ok := err.(*AddDefeatError)
	if !ok || addErr.Failure.Code != "undermine_target_role" || addErr.Failure.Element != "L1" {
		t.Fatalf("error = %#v", err)
	}
	if len(doc.Statements) != 4 || len(doc.Defeats) != 1 {
		t.Fatal("failed AddDefeat mutated the caller's document")
	}
}

func TestAddDefeatRejectsExplicitSlugIDCollision(t *testing.T) {
	doc := addDefeatDocument()
	doc.Junctors[0].ID = "colliding-slug"
	_, _, err := AddDefeat(doc, AddDefeatOptions{
		Scope: DefeatPremise, TargetRef: "P1", Text: "Challenge", Slug: "colliding-slug",
		Truth: TruthTrue, Kind: KindFact,
	})
	addErr, ok := err.(*AddDefeatError)
	if !ok || addErr.Failure.Code != "statement_slug_id_collision" {
		t.Fatalf("error = %#v", err)
	}
}

func addDefeatDocument() *Document {
	return &Document{
		ID: "defeat", Title: "Defeat",
		Metadata: []Metadata{
			{Key: "profile", Value: "cludia"},
			{Key: NextIDsMetadataKey, Value: "v1;P=3;L=2;C=1;CP=2;J=2"},
		},
		Statements: []Statement{
			{ID: "P1", Slug: "first", Role: RolePremise, Kind: KindFact, Truth: TruthTrue, Text: "First"},
			{ID: "P2", Slug: "second", Role: RolePremise, Kind: KindFact, Truth: TruthTrue, Text: "Second"},
			{ID: "L1", Slug: "target", Role: RoleLemma, Kind: KindFact, Truth: TruthUnknown, Text: "Target"},
			{ID: "CP1", Slug: "existing-challenge", Role: RoleCounterpoint, Kind: KindFact, Truth: TruthTrue, Text: "Existing challenge"},
		},
		Junctors: []Junctor{{ID: "J1", Connector: ConnectorAND, Sources: []string{"P1", "P2"}, Target: "L1"}},
		Defeats:  []Defeat{{From: "CP1", Scope: DefeatPremise, To: "P2"}},
	}
}
