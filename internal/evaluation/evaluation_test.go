// SPDX-FileCopyrightText: 2026 KeenWorks
// SPDX-License-Identifier: GPL-3.0-or-later

package evaluation

import (
	"testing"

	"github.com/tunesmith/cludia/internal/argument"
)

func TestThreeValuedTruthTables(t *testing.T) {
	values := []argument.Truth{argument.TruthTrue, argument.TruthFalse, argument.TruthUnknown}
	for _, left := range values {
		for _, right := range values {
			wantAnd := argument.TruthTrue
			if left == argument.TruthFalse || right == argument.TruthFalse {
				wantAnd = argument.TruthFalse
			} else if left == argument.TruthUnknown || right == argument.TruthUnknown {
				wantAnd = argument.TruthUnknown
			}
			if got := And([]argument.Truth{left, right}); got != wantAnd {
				t.Fatalf("%s AND %s = %s, want %s", left, right, got, wantAnd)
			}
			wantOr := argument.TruthFalse
			if left == argument.TruthTrue || right == argument.TruthTrue {
				wantOr = argument.TruthTrue
			} else if left == argument.TruthUnknown || right == argument.TruthUnknown {
				wantOr = argument.TruthUnknown
			}
			if got := Or([]argument.Truth{left, right}); got != wantOr {
				t.Fatalf("%s OR %s = %s, want %s", left, right, got, wantOr)
			}
		}
	}
}

func TestSupportPropagationCoversChainsAlternativesDirectAndUnassigned(t *testing.T) {
	doc := truthDocument()
	doc.Statements = append(doc.Statements,
		statement("P4", argument.RolePremise, argument.TruthFalse),
		statement("L2", argument.RoleLemma, argument.TruthUnknown),
		statement("L3", argument.RoleLemma, argument.TruthUnknown),
		statement("L4", argument.RoleLemma, argument.TruthUnknown),
	)
	doc.Junctors = append(doc.Junctors,
		argument.Junctor{ID: "J2", Connector: argument.ConnectorAND, Sources: []string{"L1", "P3"}, Target: "L2"},
		argument.Junctor{ID: "J3", Connector: argument.ConnectorAND, Sources: []string{"P1", "P4"}, Target: "L3"},
		argument.Junctor{ID: "J4", Connector: argument.ConnectorOR, Sources: []string{"P4", "P3"}, Target: "L3"},
	)
	doc.DirectSupports = append(doc.DirectSupports, argument.DirectSupport{Source: "L2", Target: "L4", Connector: argument.ConnectorAND})
	result, err := Evaluate(doc)
	if err != nil {
		t.Fatal(err)
	}
	assertStatementTruth(t, result, "L1", argument.TruthTrue, TruthDerived)
	assertStatementTruth(t, result, "L2", argument.TruthUnknown, TruthDerived)
	assertStatementTruth(t, result, "L3", argument.TruthUnknown, TruthDerived)
	assertStatementTruth(t, result, "L4", argument.TruthUnknown, TruthDerived)
	doc.Statements = append(doc.Statements, statement("L5", argument.RoleLemma, argument.TruthUnknown))
	result, err = Evaluate(doc)
	if err != nil {
		t.Fatal(err)
	}
	assertStatementTruth(t, result, "L5", argument.TruthUnknown, TruthUnassigned)
}

func TestGroundedUndermineAndRecursiveCounterpoint(t *testing.T) {
	doc := truthDocument()
	doc.Statements = append(doc.Statements,
		statement("CP1", argument.RoleCounterpoint, argument.TruthTrue),
		statement("CP2", argument.RoleCounterpoint, argument.TruthTrue),
		statement("L2", argument.RoleLemma, argument.TruthUnknown),
	)
	doc.DirectSupports = append(doc.DirectSupports, argument.DirectSupport{Source: "L1", Target: "L2", Connector: argument.ConnectorAND})
	doc.Defeats = []argument.Defeat{{From: "CP1", Scope: argument.DefeatPremise, To: "P1"}}
	result, err := Evaluate(doc)
	if err != nil {
		t.Fatal(err)
	}
	assertStatementTruth(t, result, "P1", argument.TruthFalse, TruthAsserted)
	assertStatementTruth(t, result, "L1", argument.TruthFalse, TruthDerived)
	assertStatementTruth(t, result, "L2", argument.TruthFalse, TruthDerived)
	assertAcceptance(t, result, "CP1", AcceptanceIn)
	if !result.TruthChangedByDefeat("P1") || !result.TruthChangedByDefeat("L1") || !result.TruthChangedByDefeat("L2") {
		t.Fatalf("accepted undermine did not mark its propagated effect")
	}

	doc.Defeats = append(doc.Defeats, argument.Defeat{From: "CP2", Scope: argument.DefeatCounterpoint, To: "CP1"})
	result, err = Evaluate(doc)
	if err != nil {
		t.Fatal(err)
	}
	assertAcceptance(t, result, "CP1", AcceptanceOut)
	assertAcceptance(t, result, "CP2", AcceptanceIn)
	assertStatementTruth(t, result, "P1", argument.TruthTrue, TruthAsserted)
	assertStatementTruth(t, result, "L1", argument.TruthTrue, TruthDerived)
	assertStatementTruth(t, result, "L2", argument.TruthTrue, TruthDerived)
	if result.TruthChangedByDefeat("P1") || result.TruthChangedByDefeat("L1") || result.TruthChangedByDefeat("L2") {
		t.Fatalf("rebutted counterpoint still marked a truth effect")
	}
}

func TestGroundedUndercutDisablesOnlySelectedJustification(t *testing.T) {
	doc := truthDocument()
	doc.Statements = append(doc.Statements, statement("CP1", argument.RoleCounterpoint, argument.TruthTrue))
	doc.Defeats = []argument.Defeat{{From: "CP1", Scope: argument.DefeatInference, JunctorID: "J1", AtTarget: "L1"}}
	result, err := Evaluate(doc)
	if err != nil {
		t.Fatal(err)
	}
	assertStatementTruth(t, result, "L1", argument.TruthFalse, TruthDerived)
	if !result.TruthChangedByDefeat("L1") {
		t.Fatal("effective undercut did not mark target")
	}
	if len(result.DisabledInferenceEdges) != 1 || result.DisabledInferenceEdges[0].JunctorID != "J1" {
		t.Fatalf("disabled edges = %#v", result.DisabledInferenceEdges)
	}
	doc.Junctors = append(doc.Junctors, argument.Junctor{ID: "J2", Connector: argument.ConnectorAND, Sources: []string{"P1", "P2"}, Target: "L1"})
	result, err = Evaluate(doc)
	if err != nil {
		t.Fatal(err)
	}
	assertStatementTruth(t, result, "L1", argument.TruthTrue, TruthDerived)
	if result.TruthChangedByDefeat("L1") {
		t.Fatal("ineffective undercut marked target despite alternative justification")
	}
}

func TestSupportedCounterpointUsesDerivedTruthForAcceptance(t *testing.T) {
	doc := truthDocument()
	doc.Statements = append(doc.Statements,
		statement("P4", argument.RolePremise, argument.TruthTrue),
		statement("CP1", argument.RoleCounterpoint, argument.TruthFalse),
	)
	doc.Junctors = append(doc.Junctors, argument.Junctor{ID: "J2", Connector: argument.ConnectorAND, Sources: []string{"P1", "P2"}, Target: "CP1"})
	doc.Defeats = []argument.Defeat{{From: "CP1", Scope: argument.DefeatPremise, To: "P4"}}
	result, err := Evaluate(doc)
	if err != nil {
		t.Fatal(err)
	}
	assertStatementTruth(t, result, "CP1", argument.TruthTrue, TruthDerived)
	assertAcceptance(t, result, "CP1", AcceptanceIn)
	assertStatementTruth(t, result, "P4", argument.TruthFalse, TruthAsserted)
}

func TestGroundedMutualCounterpointsRemainUndecided(t *testing.T) {
	doc := truthDocument()
	doc.Statements = append(doc.Statements,
		statement("CP1", argument.RoleCounterpoint, argument.TruthTrue),
		statement("CP2", argument.RoleCounterpoint, argument.TruthTrue),
	)
	doc.Defeats = []argument.Defeat{
		{From: "CP1", Scope: argument.DefeatCounterpoint, To: "CP2"},
		{From: "CP2", Scope: argument.DefeatCounterpoint, To: "CP1"},
	}
	result, err := Evaluate(doc)
	if err != nil {
		t.Fatal(err)
	}
	assertAcceptance(t, result, "CP1", AcceptanceUndecided)
	assertAcceptance(t, result, "CP2", AcceptanceUndecided)
}

func TestEvaluationRejectsSupportCycleWithoutLooping(t *testing.T) {
	doc := truthDocument()
	doc.Junctors = append(doc.Junctors, argument.Junctor{ID: "J2", Connector: argument.ConnectorAND, Sources: []string{"L1", "P3"}, Target: "P1"})
	_, err := Evaluate(doc)
	evaluationErr, ok := err.(*Error)
	if !ok || evaluationErr.Code != "evaluation_support_cycle" {
		t.Fatalf("error = %#v", err)
	}
}

func truthDocument() *argument.Document {
	return &argument.Document{
		ID: "truth", Title: "Truth",
		Statements: []argument.Statement{
			statement("P1", argument.RolePremise, argument.TruthTrue),
			statement("P2", argument.RolePremise, argument.TruthTrue),
			statement("P3", argument.RolePremise, argument.TruthUnknown),
			statement("L1", argument.RoleLemma, argument.TruthUnknown),
		},
		Junctors: []argument.Junctor{{ID: "J1", Connector: argument.ConnectorAND, Sources: []string{"P1", "P2"}, Target: "L1"}},
	}
}

func statement(id string, role argument.Role, truth argument.Truth) argument.Statement {
	return argument.Statement{ID: id, Role: role, Kind: argument.KindFact, Truth: truth, Text: id}
}

func assertStatementTruth(t *testing.T, result Result, id string, truth argument.Truth, source TruthSource) {
	t.Helper()
	value, ok := result.Statement(id)
	if !ok || value.EffectiveTruth != truth || value.TruthSource != source {
		t.Fatalf("statement %s = %#v, exists %t", id, value, ok)
	}
}

func assertAcceptance(t *testing.T, result Result, id string, acceptance Acceptance) {
	t.Helper()
	value, ok := result.Statement(id)
	if !ok || value.Acceptance != acceptance {
		t.Fatalf("counterpoint %s = %#v, exists %t", id, value, ok)
	}
}
