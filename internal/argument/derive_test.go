// SPDX-FileCopyrightText: 2026 KeenWorks
// SPDX-License-Identifier: GPL-3.0-or-later

package argument

import "testing"

func TestDeriveCreatesRoleAppropriateLemma(t *testing.T) {
	doc := deriveDocument()
	next, result, err := Derive(doc, DeriveOptions{
		SourceRefs: []string{"first", "P2"},
		NewTarget: &NewDerivedTarget{
			Text: "Derived target", Kind: KindFact, Role: RoleLemma,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Target.ID != "L1" || result.Target.Role != RoleLemma || result.Junctor.ID != "J1" || result.Junctor.Target != "L1" {
		t.Fatalf("derive result = %#v", result)
	}
	if len(doc.Statements) != 3 || len(doc.Junctors) != 0 {
		t.Fatal("derive mutated the caller's document")
	}
	if value, _ := next.MetadataValue(NextIDsMetadataKey); value != "v1;P=4;L=2;C=1;CP=1;J=2" {
		t.Fatalf("next IDs = %q", value)
	}
}

func TestDerivePromotionAssignsLemmaIDAndRewritesReferences(t *testing.T) {
	doc := deriveDocument()
	doc.Metadata = append(doc.Metadata, Metadata{Key: "root", Value: "P3"})
	doc.Statements = append(doc.Statements, Statement{
		ID: "L1", Slug: "existing-lemma", Role: RoleLemma, Kind: KindFact, Truth: TruthUnknown, Text: "Existing lemma",
	})
	doc.Junctors = append(doc.Junctors, Junctor{
		ID: "J1", Connector: ConnectorAND, Sources: []string{"P3", "P1"}, Target: "L1", Order: 1,
	})
	doc.DirectSupports = append(doc.DirectSupports, DirectSupport{Source: "P3", Target: "L1", Connector: ConnectorAND, Order: 2})

	next, result, err := Derive(doc, DeriveOptions{
		SourceRefs: []string{"P1", "P2"}, ExistingTargetRef: "candidate",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.RoleChanges) != 1 {
		t.Fatalf("role changes = %#v", result.RoleChanges)
	}
	change := result.RoleChanges[0]
	if change.PreviousID != "P3" || change.CurrentID != "L2" || change.From != RolePremise || change.To != RoleLemma {
		t.Fatalf("role change = %#v", change)
	}
	if !result.RootMetadataUpdated || result.Target.ID != "L2" || result.Target.Role != RoleLemma {
		t.Fatalf("derive result = %#v", result)
	}
	if _, exists := next.Statement("P3"); exists {
		t.Fatal("retired premise ID remains addressable")
	}
	if promoted, exists := next.Statement("L2"); !exists || promoted.Slug != "candidate" || promoted.Truth != TruthUnknown {
		t.Fatalf("promoted statement = %#v, exists %t", promoted, exists)
	}
	if got := next.Junctors[0].Sources[0]; got != "L2" {
		t.Fatalf("existing junctor source = %q", got)
	}
	if got := next.DirectSupports[0].Source; got != "L2" {
		t.Fatalf("direct support source = %q", got)
	}
	if root, _ := next.MetadataValue("root"); root != "L2" {
		t.Fatalf("root metadata = %q", root)
	}
	if value, _ := next.MetadataValue(NextIDsMetadataKey); value != "v1;P=4;L=3;C=1;CP=1;J=3" {
		t.Fatalf("next IDs = %q", value)
	}
	if doc.Statements[2].ID != "P3" || doc.Junctors[0].Sources[0] != "P3" {
		t.Fatal("derive mutated the caller's document")
	}
}

func TestDeriveReportsAllMissingSources(t *testing.T) {
	_, _, err := Derive(deriveDocument(), DeriveOptions{
		SourceRefs: []string{"missing-one", "missing-two"}, ExistingTargetRef: "candidate",
	})
	deriveErr, ok := err.(*DeriveError)
	if !ok || len(deriveErr.Failures) != 2 || deriveErr.Failures[0].Code != "source_not_found" || deriveErr.Failures[1].Code != "source_not_found" {
		t.Fatalf("derive error = %#v", err)
	}
}

func TestDeriveNormalizesSupportedCounterpointTruthAndRemovalDoesNotRestoreIt(t *testing.T) {
	doc := deriveDocument()
	doc.Statements = append(doc.Statements, Statement{
		ID: "CP1", Slug: "challenge", Role: RoleCounterpoint, Kind: KindFact, Truth: TruthTrue, Text: "Challenge",
	})
	doc.Metadata[1].Value = "v1;P=4;L=1;C=1;CP=2;J=1"
	next, result, err := Derive(doc, DeriveOptions{SourceRefs: []string{"P1", "P2"}, ExistingTargetRef: "challenge"})
	if err != nil || result.Target.Truth != TruthUnknown || len(result.TruthChanges) != 1 || result.TruthChanges[0].ID != "CP1" {
		t.Fatalf("derive = %#v, %v", result, err)
	}
	removed, _, err := RemoveJunctor(next, result.Junctor.ID)
	if err != nil {
		t.Fatal(err)
	}
	counterpoint, _ := removed.Statement("CP1")
	if counterpoint.Truth != TruthUnknown {
		t.Fatalf("truth restored after support removal: %#v", counterpoint)
	}
}

func deriveDocument() *Document {
	return &Document{
		ID: "derive", Title: "Derive",
		Metadata: []Metadata{
			{Key: "profile", Value: "cludia"},
			{Key: NextIDsMetadataKey, Value: "v1;P=4;L=1;C=1;CP=1;J=1"},
		},
		Statements: []Statement{
			{ID: "P1", Slug: "first", Role: RolePremise, Kind: KindFact, Truth: TruthTrue, Text: "First"},
			{ID: "P2", Slug: "second", Role: RolePremise, Kind: KindFact, Truth: TruthTrue, Text: "Second"},
			{ID: "P3", Slug: "candidate", Role: RolePremise, Kind: KindFact, Truth: TruthTrue, Text: "Candidate"},
		},
		Junctors: []Junctor{}, DirectSupports: []DirectSupport{}, Defeats: []Defeat{},
	}
}
