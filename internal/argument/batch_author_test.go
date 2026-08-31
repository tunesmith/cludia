// SPDX-FileCopyrightText: 2026 KeenWorks
// SPDX-License-Identifier: GPL-3.0-or-later

package argument

import (
	"reflect"
	"testing"
)

func TestAuthorBatchCreatesFinalRolesDerivationsAndRecursiveDefeats(t *testing.T) {
	doc := batchAuthorDocument()
	cpRole := RoleCounterpoint
	trueTruth := TruthTrue
	next, result, err := AuthorBatch(doc, AuthorBatchOptions{
		Statements: []BatchStatementSpec{
			{Key: "puncture", Text: "A puncture marked the thumb."},
			{Key: "groove", Text: "A groove marked the clasp."},
			{Key: "theory", Text: "The clasp caused the puncture.", Truth: &trueTruth},
			{Key: "alternative", Text: "Another object could explain the puncture.", Role: &cpRole},
			{Key: "answer", Text: "The groove excludes an unrelated object.", Role: &cpRole},
		},
		Derivations: []BatchDerivationSpec{{
			Key:     "needle-inference",
			Sources: []BatchReference{{Key: "puncture"}, {Key: "groove"}},
			Target:  BatchReference{Key: "theory"},
		}},
		Defeats: []BatchDefeatSpec{
			{From: BatchReference{Key: "alternative"}, Scope: DefeatInference, Target: BatchReference{Key: "needle-inference"}},
			{From: BatchReference{Key: "answer"}, Scope: DefeatCounterpoint, Target: BatchReference{Key: "alternative"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Statements) != 2 || len(doc.Junctors) != 0 || len(doc.Defeats) != 0 {
		t.Fatalf("caller document mutated: %#v", doc)
	}
	if len(result.Statements) != 5 || len(result.Derivations) != 1 || len(result.Defeats) != 2 {
		t.Fatalf("result = %#v", result)
	}
	mapped := make(map[string]Statement)
	for _, item := range result.Statements {
		mapped[item.Key] = item.Statement
	}
	if mapped["puncture"].ID != "P3" || mapped["groove"].ID != "P4" {
		t.Fatalf("premise mappings = %#v", mapped)
	}
	if mapped["theory"].ID != "L1" || mapped["theory"].Role != RoleLemma || mapped["theory"].Truth != TruthUnknown {
		t.Fatalf("derived target mapping = %#v", mapped["theory"])
	}
	if mapped["alternative"].ID != "CP1" || mapped["answer"].ID != "CP2" {
		t.Fatalf("counterpoint mappings = %#v", mapped)
	}
	if got := result.Derivations[0].Junctor; got.ID != "J1" || !reflect.DeepEqual(got.Sources, []string{"P3", "P4"}) || got.Target != "L1" {
		t.Fatalf("derivation = %#v", got)
	}
	if got := result.Defeats[0].Defeat; got.From != "CP1" || got.JunctorID != "J1" || got.AtTarget != "L1" {
		t.Fatalf("inference defeat = %#v", got)
	}
	if got := result.Defeats[1].Defeat; got.From != "CP2" || got.To != "CP1" {
		t.Fatalf("counterpoint defeat = %#v", got)
	}
	if len(result.TruthChanges) != 1 || result.TruthChanges[0] != (StatementTruthChange{ID: "L1", From: TruthTrue, To: TruthUnknown}) {
		t.Fatalf("truth changes = %#v", result.TruthChanges)
	}
	if value, _ := next.MetadataValue(NextIDsMetadataKey); value != "v1;P=5;L=2;C=1;CP=3;J=2" {
		t.Fatalf("next ids = %q", value)
	}
}

func TestAuthorBatchTracksExistingIDsAcrossPromotions(t *testing.T) {
	doc := batchAuthorDocument()
	doc.Statements = append(doc.Statements,
		Statement{ID: "P3", Role: RolePremise, Kind: KindFact, Truth: TruthTrue, Text: "Third"},
		Statement{ID: "P4", Role: RolePremise, Kind: KindFact, Truth: TruthTrue, Text: "Fourth"},
	)
	next, result, err := AuthorBatch(doc, AuthorBatchOptions{Derivations: []BatchDerivationSpec{
		{Key: "first", Sources: []BatchReference{{ID: "P2"}, {ID: "P3"}}, Target: BatchReference{ID: "P1"}},
		{Key: "second", Sources: []BatchReference{{ID: "P1"}, {ID: "P2"}}, Target: BatchReference{ID: "P4"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.RoleChanges) != 2 || result.RoleChanges[0].PreviousID != "P1" || result.RoleChanges[0].CurrentID != "L1" || result.RoleChanges[1].CurrentID != "L2" {
		t.Fatalf("role changes = %#v", result.RoleChanges)
	}
	if got := result.Derivations[1].Junctor.Sources; !reflect.DeepEqual(got, []string{"L1", "P2"}) {
		t.Fatalf("second sources = %#v", got)
	}
	if _, ok := next.Statement("P1"); ok {
		t.Fatal("retired P1 remains")
	}
}

func TestAuthorBatchFailureLeavesCallerUnchanged(t *testing.T) {
	doc := batchAuthorDocument()
	cpRole := RoleCounterpoint
	before, marshalErr := marshalDocumentState(doc)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	_, _, err := AuthorBatch(doc, AuthorBatchOptions{
		Statements: []BatchStatementSpec{{Key: "cp", Text: "One challenge", Role: &cpRole}},
		Defeats: []BatchDefeatSpec{
			{From: BatchReference{Key: "cp"}, Scope: DefeatPremise, Target: BatchReference{ID: "P1"}},
			{From: BatchReference{Key: "cp"}, Scope: DefeatPremise, Target: BatchReference{ID: "P2"}},
		},
	})
	mutationErr, ok := err.(*MutationError)
	if !ok || mutationErr.Code != "defeat_source_multiple" {
		t.Fatalf("error = %#v", err)
	}
	after, marshalErr := marshalDocumentState(doc)
	if marshalErr != nil || !reflect.DeepEqual(after, before) {
		t.Fatal("failed batch mutated caller")
	}
}

func TestAuthorBatchRejectsAmbiguousReferencesAndExplicitPremiseTargets(t *testing.T) {
	doc := batchAuthorDocument()
	_, _, err := AuthorBatch(doc, AuthorBatchOptions{Derivations: []BatchDerivationSpec{{
		Key: "bad", Sources: []BatchReference{{ID: "P1", Key: "both"}, {ID: "P2"}}, Target: BatchReference{ID: "P1"},
	}}})
	mutationErr, ok := err.(*MutationError)
	if !ok || mutationErr.Code != "batch_reference_invalid" {
		t.Fatalf("ambiguous reference error = %#v", err)
	}
	premise := RolePremise
	_, _, err = AuthorBatch(doc, AuthorBatchOptions{
		Statements: []BatchStatementSpec{
			{Key: "a", Text: "A"}, {Key: "b", Text: "B"}, {Key: "target", Text: "Target", Role: &premise},
		},
		Derivations: []BatchDerivationSpec{{Key: "d", Sources: []BatchReference{{Key: "a"}, {Key: "b"}}, Target: BatchReference{Key: "target"}}},
	})
	mutationErr, ok = err.(*MutationError)
	if !ok || mutationErr.Code != "batch_statement_role_conflict" {
		t.Fatalf("role conflict error = %#v", err)
	}
}

func batchAuthorDocument() *Document {
	return &Document{
		ID: "batch", Title: "Batch", Metadata: []Metadata{{Key: "profile", Value: "cludia"}},
		Statements: []Statement{
			{ID: "P1", Role: RolePremise, Kind: KindFact, Truth: TruthTrue, Text: "First"},
			{ID: "P2", Role: RolePremise, Kind: KindFact, Truth: TruthTrue, Text: "Second"},
		},
		Junctors: []Junctor{}, DirectSupports: []DirectSupport{}, Defeats: []Defeat{},
	}
}
