package argument

import "testing"

func TestRepairJunctorSupportsAddRemoveAndReplaceOnClones(t *testing.T) {
	doc := repairDocument()
	added, add, err := RepairJunctor(doc, RepairJunctorOptions{JunctorID: "J1", Mode: SourceAdd, SourceRef: "P4"})
	if err != nil || add.SourceAdded != "P4" || len(add.Current.Sources) != 4 || len(doc.Junctors[0].Sources) != 3 {
		t.Fatalf("add = %#v, %v", add, err)
	}
	removed, remove, err := RepairJunctor(added, RepairJunctorOptions{JunctorID: "J1", Mode: SourceRemove, SourceRef: "P2"})
	if err != nil || remove.SourceRemoved != "P2" || len(remove.Current.Sources) != 3 {
		t.Fatalf("remove = %#v, %v", remove, err)
	}
	_, replace, err := RepairJunctor(removed, RepairJunctorOptions{JunctorID: "J1", Mode: SourceReplace, FromRef: "P3", ToRef: "P2"})
	if err != nil || replace.SourceRemoved != "P3" || replace.SourceAdded != "P2" || replace.Current.Sources[1] != "P2" {
		t.Fatalf("replace = %#v, %v", replace, err)
	}
}

func TestRepairJunctorRejectsInvariantFailureWithoutMutatingCaller(t *testing.T) {
	doc := repairDocument()
	doc.Junctors[0].Sources = []string{"P1", "P2"}
	_, _, err := RepairJunctor(doc, RepairJunctorOptions{JunctorID: "J1", Mode: SourceRemove, SourceRef: "P2"})
	mutationErr, ok := err.(*MutationError)
	if !ok || mutationErr.Code != "junctor_sources_too_few" || len(doc.Junctors[0].Sources) != 2 {
		t.Fatalf("error = %#v, sources=%v", err, doc.Junctors[0].Sources)
	}
}

func TestRemoveJunctorPreservesCallerAndAllocatorState(t *testing.T) {
	doc := repairDocument()
	next, result, err := RemoveJunctor(doc, "J1")
	if err != nil || result.Previous.ID != "J1" || len(next.Junctors) != 0 || len(doc.Junctors) != 1 {
		t.Fatalf("remove = %#v, %v", result, err)
	}
	if value, _ := next.MetadataValue(NextIDsMetadataKey); value != "v1;P=5;L=2;C=1;CP=1;J=2" {
		t.Fatalf("allocator = %q", value)
	}
}

func repairDocument() *Document {
	return &Document{
		ID: "repair", Title: "Repair",
		Metadata: []Metadata{{Key: "profile", Value: "workspace"}, {Key: NextIDsMetadataKey, Value: "v1;P=5;L=2;C=1;CP=1;J=2"}},
		Statements: []Statement{
			{ID: "P1", Role: RolePremise, Kind: KindFact, Truth: TruthTrue, Text: "One"},
			{ID: "P2", Role: RolePremise, Kind: KindFact, Truth: TruthTrue, Text: "Two"},
			{ID: "P3", Role: RolePremise, Kind: KindFact, Truth: TruthTrue, Text: "Three"},
			{ID: "P4", Role: RolePremise, Kind: KindFact, Truth: TruthTrue, Text: "Four"},
			{ID: "L1", Role: RoleLemma, Kind: KindFact, Truth: TruthUnknown, Text: "Target"},
		},
		Junctors: []Junctor{{ID: "J1", Connector: ConnectorAND, Sources: []string{"P1", "P2", "P3"}, Target: "L1"}},
	}
}
