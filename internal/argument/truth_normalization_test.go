package argument

import "testing"

func TestNormalizeNonLeafTruthChangesOnlySourcedStatementsOnClone(t *testing.T) {
	doc := &Document{
		ID: "normalize", Title: "Normalize",
		Statements: []Statement{
			{ID: "P1", Role: RolePremise, Kind: KindFact, Truth: TruthTrue, Text: "One"},
			{ID: "P2", Role: RolePremise, Kind: KindFact, Truth: TruthTrue, Text: "Two"},
			{ID: "CP1", Role: RoleCounterpoint, Kind: KindFact, Truth: TruthFalse, Text: "Challenge"},
		},
		Junctors: []Junctor{{ID: "J1", Connector: ConnectorAND, Sources: []string{"P1", "P2"}, Target: "CP1"}},
	}
	next, result, err := NormalizeNonLeafTruth(doc)
	if err != nil || !result.Changed || len(result.Statements) != 1 || result.Statements[0].ID != "CP1" || result.PlanToken == "" {
		t.Fatalf("normalization = %#v, %v", result, err)
	}
	if next.Statements[2].Truth != TruthUnknown || doc.Statements[2].Truth != TruthFalse || next.Statements[0].Truth != TruthTrue {
		t.Fatalf("next/caller truths = %#v / %#v", next.Statements, doc.Statements)
	}
}

func TestNormalizeTruthTokenIsStateBound(t *testing.T) {
	doc := &Document{ID: "normalize", Title: "Normalize", Statements: []Statement{{ID: "P1", Role: RolePremise, Kind: KindFact, Truth: TruthTrue, Text: "One"}}}
	_, first, err := NormalizeNonLeafTruth(doc)
	if err != nil {
		t.Fatal(err)
	}
	doc.Statements[0].Text = "Changed"
	_, second, err := NormalizeNonLeafTruth(doc)
	if err != nil || first.PlanToken == second.PlanToken {
		t.Fatalf("tokens = %q / %q, err=%v", first.PlanToken, second.PlanToken, err)
	}
}
