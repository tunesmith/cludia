package argument

import "testing"

func TestReplaceStatementAppliesExplicitChoicesOnClone(t *testing.T) {
	doc := replacementDocument()
	next, result, err := ReplaceStatement(doc, ReplacementOptions{
		OldRef: "old", ReplacementRef: "replacement",
		SourceJunctors: []string{"J2"}, RemoveJustifications: []string{"J1"}, DeleteOld: true,
	})
	if err != nil || !result.Applicable || !result.OldDeleted || result.PlanToken == "" {
		t.Fatalf("replacement = %#v, %v", result, err)
	}
	if len(result.SourceRetargets) != 1 || result.SourceRetargets[0].Junctor.Sources[0] != "P3" || len(result.JustificationsRemoved) != 1 {
		t.Fatalf("replacement effects = %#v", result)
	}
	if _, exists := next.Statement("L1"); exists {
		t.Fatal("old statement remains in planned document")
	}
	if _, exists := doc.Statement("L1"); !exists || len(doc.Junctors) != 2 {
		t.Fatal("replacement mutated caller")
	}
}

func TestReplaceStatementReportsDeletionBlockersWithoutMutatingCaller(t *testing.T) {
	doc := replacementDocument()
	_, result, err := ReplaceStatement(doc, ReplacementOptions{OldRef: "old", ReplacementRef: "replacement", DeleteOld: true})
	if err != nil || result.Applicable || len(result.Blockers) != 2 || result.PlanToken != "" {
		t.Fatalf("blocked replacement = %#v, %v", result, err)
	}
	if len(doc.Statements) != 6 || len(doc.Junctors) != 2 {
		t.Fatal("blocked replacement mutated caller")
	}
}

func TestReplacementTokenIsStateBound(t *testing.T) {
	doc := replacementDocument()
	_, first, err := ReplaceStatement(doc, ReplacementOptions{OldRef: "old", ReplacementRef: "replacement", SourceJunctors: []string{"J2"}})
	if err != nil {
		t.Fatal(err)
	}
	doc.Statements[0].Text = "Changed"
	_, second, err := ReplaceStatement(doc, ReplacementOptions{OldRef: "old", ReplacementRef: "replacement", SourceJunctors: []string{"J2"}})
	if err != nil || first.PlanToken == second.PlanToken {
		t.Fatalf("tokens first=%q second=%q err=%v", first.PlanToken, second.PlanToken, err)
	}
}

func TestReplacementBootstrapsLegacyAllocatorBeforeRemovingHighestIDs(t *testing.T) {
	doc := &Document{
		ID: "legacy", Title: "Legacy", Metadata: []Metadata{{Key: "profile", Value: "workspace"}},
		Statements: []Statement{
			{ID: "P1", Role: RolePremise, Kind: KindFact, Truth: TruthTrue, Text: "One"},
			{ID: "P2", Role: RolePremise, Kind: KindFact, Truth: TruthTrue, Text: "Two"},
			{ID: "L9", Slug: "old", Role: RoleLemma, Kind: KindFact, Truth: TruthUnknown, Text: "Old"},
			{ID: "P9", Slug: "replacement", Role: RolePremise, Kind: KindFact, Truth: TruthTrue, Text: "Replacement"},
		},
		Junctors: []Junctor{{ID: "J9", Connector: ConnectorAND, Sources: []string{"P1", "P2"}, Target: "L9"}},
	}
	next, result, err := ReplaceStatement(doc, ReplacementOptions{
		OldRef: "old", ReplacementRef: "replacement", RemoveJustifications: []string{"J9"}, DeleteOld: true,
	})
	if err != nil || !result.OldDeleted {
		t.Fatalf("replacement = %#v, %v", result, err)
	}
	value, ok := next.MetadataValue(NextIDsMetadataKey)
	if !ok {
		t.Fatal("replacement did not bootstrap allocator metadata")
	}
	ids, err := ParseNextIDs(value)
	if err != nil || ids.P != 10 || ids.L != 10 || ids.J != 10 {
		t.Fatalf("next IDs = %#v from %q, err=%v", ids, value, err)
	}
}

func replacementDocument() *Document {
	return &Document{
		ID: "replace", Title: "Replace",
		Metadata: []Metadata{{Key: "profile", Value: "workspace"}, {Key: NextIDsMetadataKey, Value: "v1;P=4;L=2;C=2;CP=1;J=3"}},
		Statements: []Statement{
			{ID: "P1", Role: RolePremise, Kind: KindFact, Truth: TruthTrue, Text: "One"},
			{ID: "P2", Role: RolePremise, Kind: KindFact, Truth: TruthTrue, Text: "Two"},
			{ID: "L1", Slug: "old", Role: RoleLemma, Kind: KindFact, Truth: TruthUnknown, Text: "Old"},
			{ID: "P3", Slug: "replacement", Role: RolePremise, Kind: KindFact, Truth: TruthTrue, Text: "Replacement"},
			{ID: "C1", Role: RoleConclusion, Kind: KindFact, Truth: TruthUnknown, Text: "Conclusion"},
			{ID: "custom", Role: RolePremise, Kind: KindFact, Truth: TruthTrue, Text: "Unrelated"},
		},
		Junctors: []Junctor{
			{ID: "J1", Connector: ConnectorAND, Sources: []string{"P1", "P2"}, Target: "L1"},
			{ID: "J2", Connector: ConnectorAND, Sources: []string{"L1", "P2"}, Target: "C1"},
		},
	}
}
