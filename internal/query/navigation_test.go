// SPDX-FileCopyrightText: 2026 KeenWorks
// SPDX-License-Identifier: GPL-3.0-or-later

package query

import (
	"reflect"
	"testing"

	"github.com/tunesmith/cludia/internal/argument"
	"github.com/tunesmith/cludia/internal/evaluation"
)

func TestTopUsesDocumentOrderLongestDepthAndChallengeState(t *testing.T) {
	doc := navigationDocument()
	items := Top(doc)
	if len(items) != 2 {
		t.Fatalf("top items = %#v", items)
	}
	if items[0].Statement.ID != "L2" || items[0].Depth != 2 || !items[0].Challenged {
		t.Fatalf("first top item = %#v", items[0])
	}
	if items[1].Statement.ID != "P5" || items[1].Depth != 0 || items[1].Challenged {
		t.Fatalf("second top item = %#v", items[1])
	}
	for _, item := range items {
		if item.Statement.Role == argument.RoleCounterpoint {
			t.Fatalf("counterpoint included in top: %#v", item)
		}
	}
}

func TestLedgerIsStableSupportOnlyAndPreservesMultipleDerivations(t *testing.T) {
	doc := navigationDocument()
	root, rows, err := Ledger(doc, "final")
	if err != nil {
		t.Fatal(err)
	}
	if root != "L2" {
		t.Fatalf("root = %q", root)
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.Statement.ID)
	}
	want := []string{"P1", "P2", "P3", "P4", "L1", "L2"}
	if !reflect.DeepEqual(ids, want) {
		t.Fatalf("ledger order = %#v, want %#v", ids, want)
	}
	if len(rows[4].Derivations) != 2 || rows[4].Derivations[0].Connector != argument.ConnectorAND || rows[4].Derivations[1].Connector != argument.ConnectorOR {
		t.Fatalf("L1 derivations = %#v", rows[4].Derivations)
	}
	if len(rows[5].Derivations) != 2 || rows[5].Derivations[1].Type != "direct" || rows[5].Depth != 2 || !rows[5].Challenged {
		t.Fatalf("L2 row = %#v", rows[5])
	}
	for _, row := range rows {
		if row.Statement.Role == argument.RoleCounterpoint {
			t.Fatalf("challenge row leaked into ledger: %#v", row)
		}
	}
}

func TestLedgerAcceptsSlugAndRejectsCounterpointRoot(t *testing.T) {
	doc := navigationDocument()
	if root, _, err := Ledger(doc, "final"); err != nil || root != "L2" {
		t.Fatalf("ledger by slug root=%q err=%v", root, err)
	}
	if _, _, err := Ledger(doc, "CP1"); err == nil {
		t.Fatal("counterpoint ledger root accepted")
	}
}

func TestDirectChallengeRemainsInspectableWhenGroundedEffectIsRebutted(t *testing.T) {
	doc := navigationDocument()
	if !StatementDirectlyChallenged(doc, "P5") {
		t.Fatal("direct challenge was not discoverable")
	}
	evaluated, err := evaluation.Evaluate(doc)
	if err != nil {
		t.Fatal(err)
	}
	if evaluated.TruthChangedByDefeat("P5") {
		t.Fatal("rebutted direct challenge changed the displayed contestation state")
	}
}

func TestLedgerIncludesCounterpointExplicitlyUsedAsSupportSource(t *testing.T) {
	doc := &argument.Document{
		Statements: []argument.Statement{
			{ID: "P1", Role: argument.RolePremise, Kind: argument.KindFact, Truth: argument.TruthTrue, Text: "Premise"},
			{ID: "CP1", Role: argument.RoleCounterpoint, Kind: argument.KindFact, Truth: argument.TruthTrue, Text: "Counterpoint used as an authored support source"},
			{ID: "L1", Role: argument.RoleLemma, Kind: argument.KindFact, Truth: argument.TruthUnknown, Text: "Target"},
		},
		Junctors: []argument.Junctor{{ID: "J1", Connector: argument.ConnectorAND, Sources: []string{"P1", "CP1"}, Target: "L1"}},
	}
	_, rows, err := Ledger(doc, "L1")
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{rows[0].Statement.ID, rows[1].Statement.ID, rows[2].Statement.ID}
	if !reflect.DeepEqual(ids, []string{"P1", "CP1", "L1"}) {
		t.Fatalf("ledger support rows = %#v", ids)
	}
}

func TestLedgerIntroducesPremiseNearUseAfterDerivedCompanion(t *testing.T) {
	statement := func(id string, role argument.Role) argument.Statement {
		truth := argument.TruthUnknown
		if role == argument.RolePremise {
			truth = argument.TruthTrue
		}
		return argument.Statement{ID: id, Role: role, Kind: argument.KindFact, Truth: truth, Text: id}
	}
	doc := &argument.Document{
		Statements: []argument.Statement{
			statement("P3", argument.RolePremise), // General order alone would front-load P3.
			statement("P1", argument.RolePremise), statement("P2", argument.RolePremise),
			statement("L34", argument.RoleLemma), statement("L35", argument.RoleLemma),
		},
		Junctors: []argument.Junctor{
			{ID: "J1", Connector: argument.ConnectorAND, Sources: []string{"P1", "P2"}, Target: "L34"},
			{ID: "J2", Connector: argument.ConnectorAND, Sources: []string{"L34", "P3"}, Target: "L35"},
		},
	}
	_, rows, err := Ledger(doc, "L35")
	if err != nil {
		t.Fatal(err)
	}
	ids := ledgerIDs(rows)
	if want := []string{"P1", "P2", "L34", "P3", "L35"}; !reflect.DeepEqual(ids, want) {
		t.Fatalf("local ledger order = %v, want %v", ids, want)
	}
}

func TestLedgerUsesDocumentOrderForEquivalentBranchesAndSharedSources(t *testing.T) {
	statement := func(id string, role argument.Role) argument.Statement {
		truth := argument.TruthUnknown
		if role == argument.RolePremise {
			truth = argument.TruthTrue
		}
		return argument.Statement{ID: id, Role: role, Kind: argument.KindFact, Truth: truth, Text: id}
	}
	doc := &argument.Document{
		Statements: []argument.Statement{
			statement("P0", argument.RolePremise), statement("P1", argument.RolePremise), statement("P2", argument.RolePremise),
			statement("L1", argument.RoleLemma), statement("L2", argument.RoleLemma), statement("L3", argument.RoleLemma),
		},
		Junctors: []argument.Junctor{
			{ID: "J1", Connector: argument.ConnectorAND, Sources: []string{"P0", "P1"}, Target: "L1"},
			{ID: "J2", Connector: argument.ConnectorAND, Sources: []string{"P0", "P2"}, Target: "L2"},
			{ID: "J3", Connector: argument.ConnectorAND, Sources: []string{"L1", "L2"}, Target: "L3"},
		},
	}
	_, rows, err := Ledger(doc, "L3")
	if err != nil {
		t.Fatal(err)
	}
	ids := ledgerIDs(rows)
	if want := []string{"P0", "P1", "L1", "P2", "L2", "L3"}; !reflect.DeepEqual(ids, want) {
		t.Fatalf("shared-source ledger order = %v, want %v", ids, want)
	}
	for i := 0; i < 5; i++ {
		_, again, repeatErr := Ledger(doc, "L3")
		if repeatErr != nil || !reflect.DeepEqual(ledgerIDs(again), ids) {
			t.Fatalf("ledger is not deterministic: %v %v", repeatErr, ledgerIDs(again))
		}
	}
}

func TestLedgerInferenceSelectsOnlyOneRootBranchAndKeepsFullSourceClosure(t *testing.T) {
	doc := selectedLedgerDocument()
	// Make P1 derived. Selecting J1 at L1 must narrow only L1; it must retain
	// P1's own complete justification.
	doc.Statements[0].Role = argument.RoleLemma
	doc.Statements[0].Truth = argument.TruthUnknown
	doc.Statements = append(doc.Statements,
		argument.Statement{ID: "P5", Role: argument.RolePremise, Kind: argument.KindFact, Truth: argument.TruthTrue, Text: "Fifth"},
		argument.Statement{ID: "P6", Role: argument.RolePremise, Kind: argument.KindFact, Truth: argument.TruthTrue, Text: "Sixth"},
	)
	doc.Junctors = append([]argument.Junctor{
		{ID: "J0", Connector: argument.ConnectorAND, Sources: []string{"P5", "P6"}, Target: "P1"},
	}, doc.Junctors...)
	evaluated, err := evaluation.Evaluate(doc)
	if err != nil {
		t.Fatal(err)
	}
	root, rows, selected, err := LedgerInferenceEvaluated(doc, "L1", "J1", evaluated)
	if err != nil {
		t.Fatal(err)
	}
	if root != "L1" || !reflect.DeepEqual(ledgerIDs(rows), []string{"P5", "P6", "P1", "P2", "L1"}) {
		t.Fatalf("selected rows = %s %#v", root, ledgerIDs(rows))
	}
	if len(rows[len(rows)-1].Derivations) != 1 || rows[len(rows)-1].Derivations[0].ID != "J1" {
		t.Fatalf("root derivations = %#v", rows[len(rows)-1].Derivations)
	}
	if selected.Junctor.ID != "J1" || selected.EffectiveTruth != argument.TruthFalse || selected.DisabledByUndercut || !selected.OtherJustificationsOmitted || !selected.OtherRoutesAffectTruth {
		t.Fatalf("selection = %#v", selected)
	}
}

func TestLedgerInferenceDistinguishesUndercutAndOtherRouteEffect(t *testing.T) {
	doc := selectedLedgerDocument()
	doc.Statements[1].Truth = argument.TruthTrue
	doc.Statements = append(doc.Statements, argument.Statement{ID: "CP1", Role: argument.RoleCounterpoint, Kind: argument.KindFact, Truth: argument.TruthTrue, Text: "J1 is undercut"})
	doc.Defeats = append(doc.Defeats, argument.Defeat{From: "CP1", Scope: argument.DefeatInference, JunctorID: "J1", AtTarget: "L1"})
	evaluated, err := evaluation.Evaluate(doc)
	if err != nil {
		t.Fatal(err)
	}
	_, _, selected, err := LedgerInferenceEvaluated(doc, "L1", "J1", evaluated)
	if err != nil {
		t.Fatal(err)
	}
	if selected.EffectiveTruth != argument.TruthTrue || !selected.DisabledByUndercut || !selected.OtherRoutesAffectTruth {
		t.Fatalf("selection with alternative = %#v", selected)
	}
	doc.Junctors = doc.Junctors[:1]
	evaluated, err = evaluation.Evaluate(doc)
	if err != nil {
		t.Fatal(err)
	}
	_, _, selected, err = LedgerInferenceEvaluated(doc, "L1", "J1", evaluated)
	if err != nil {
		t.Fatal(err)
	}
	if !selected.DisabledByUndercut || selected.OtherJustificationsOmitted || selected.OtherRoutesAffectTruth {
		t.Fatalf("selection without alternative = %#v", selected)
	}
}

func TestLedgerInferenceRejectsMissingAndMismatchedJunctors(t *testing.T) {
	doc := selectedLedgerDocument()
	doc.Statements = append(doc.Statements, argument.Statement{ID: "L2", Role: argument.RoleLemma, Kind: argument.KindFact, Truth: argument.TruthUnknown, Text: "Other target"})
	doc.Junctors = append(doc.Junctors, argument.Junctor{ID: "J3", Connector: argument.ConnectorAND, Sources: []string{"P1", "P3"}, Target: "L2"})
	evaluated, err := evaluation.Evaluate(doc)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		id, code string
	}{{"missing", "ledger_inference_not_found"}, {"J3", "ledger_inference_target_mismatch"}} {
		_, _, _, err := LedgerInferenceEvaluated(doc, "L1", test.id, evaluated)
		selectionErr, ok := err.(*LedgerInferenceError)
		if !ok || selectionErr.Code != test.code {
			t.Fatalf("%s error = %#v", test.id, err)
		}
	}
}

func ledgerIDs(rows []LedgerRow) []string {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.Statement.ID)
	}
	return ids
}

func selectedLedgerDocument() *argument.Document {
	return &argument.Document{
		ID: "selected-ledger", Title: "Selected Ledger",
		Statements: []argument.Statement{
			{ID: "P1", Role: argument.RolePremise, Kind: argument.KindFact, Truth: argument.TruthTrue, Text: "First"},
			{ID: "P2", Role: argument.RolePremise, Kind: argument.KindFact, Truth: argument.TruthFalse, Text: "Second"},
			{ID: "P3", Role: argument.RolePremise, Kind: argument.KindFact, Truth: argument.TruthTrue, Text: "Third"},
			{ID: "P4", Role: argument.RolePremise, Kind: argument.KindFact, Truth: argument.TruthTrue, Text: "Fourth"},
			{ID: "L1", Role: argument.RoleLemma, Kind: argument.KindFact, Truth: argument.TruthUnknown, Text: "Target"},
		},
		Junctors: []argument.Junctor{
			{ID: "J1", Connector: argument.ConnectorAND, Sources: []string{"P1", "P2"}, Target: "L1"},
			{ID: "J2", Connector: argument.ConnectorAND, Sources: []string{"P3", "P4"}, Target: "L1"},
		},
		DirectSupports: []argument.DirectSupport{}, Defeats: []argument.Defeat{},
	}
}

func navigationDocument() *argument.Document {
	statement := func(id, slug string, role argument.Role) argument.Statement {
		truth := argument.TruthUnknown
		if role == argument.RolePremise || role == argument.RoleCounterpoint {
			truth = argument.TruthTrue
		}
		return argument.Statement{ID: id, Slug: slug, Role: role, Kind: argument.KindFact, Truth: truth, Text: "Full statement text for " + id}
	}
	return &argument.Document{
		ID: "navigation", Title: "Navigation",
		Statements: []argument.Statement{
			statement("P1", "one", argument.RolePremise), statement("P2", "two", argument.RolePremise),
			statement("P3", "three", argument.RolePremise), statement("P4", "four", argument.RolePremise),
			statement("L1", "middle", argument.RoleLemma), statement("L2", "final", argument.RoleLemma),
			statement("P5", "isolated", argument.RolePremise), statement("CP1", "challenge", argument.RoleCounterpoint),
			statement("CP2", "answer", argument.RoleCounterpoint), statement("CP3", "undercut", argument.RoleCounterpoint),
			statement("CP4", "source-challenge", argument.RoleCounterpoint),
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
		},
	}
}
