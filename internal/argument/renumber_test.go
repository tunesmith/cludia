package argument

import "testing"

func TestRenumberDocumentRewritesEveryModeledReferenceOnClone(t *testing.T) {
	doc := &Document{
		ID: "renumber", Title: "Renumber",
		Metadata: []Metadata{{Key: "profile", Value: "workspace"}, {Key: "root", Value: "old-c"}, {Key: NextIDsMetadataKey, Value: "v1;P=10;L=9;C=4;CP=7;J=8"}},
		Statements: []Statement{
			{ID: "old-p", Role: RolePremise, Kind: KindFact, Truth: TruthTrue, Text: "Premise"},
			{ID: "old-l", Role: RoleLemma, Kind: KindFact, Truth: TruthUnknown, Text: "Lemma"},
			{ID: "old-c", Role: RoleConclusion, Kind: KindFact, Truth: TruthUnknown, Text: "Conclusion"},
			{ID: "old-cp", Role: RoleCounterpoint, Kind: KindFact, Truth: TruthTrue, Text: "Counterpoint"},
		},
		Junctors:       []Junctor{{ID: "old-j", Connector: ConnectorAND, Sources: []string{"old-p", "old-l"}, Target: "old-c"}},
		DirectSupports: []DirectSupport{{Source: "old-p", Target: "old-l", Connector: ConnectorAND}},
		Defeats:        []Defeat{{From: "old-cp", Scope: DefeatInference, JunctorID: "old-j", AtTarget: "old-c"}},
	}
	next, result, err := RenumberDocument(doc)
	if err != nil || !result.IDsChanged || !result.RootMetadataUpdated || result.PlanToken == "" {
		t.Fatalf("renumber = %#v, %v", result, err)
	}
	if next.Junctors[0].ID != "J1" || next.Junctors[0].Sources[0] != "P1" || next.Junctors[0].Sources[1] != "L1" || next.Junctors[0].Target != "C1" {
		t.Fatalf("junctor = %#v", next.Junctors[0])
	}
	if next.DirectSupports[0].Source != "P1" || next.DirectSupports[0].Target != "L1" || next.Defeats[0].From != "CP1" || next.Defeats[0].JunctorID != "J1" || next.Defeats[0].AtTarget != "C1" {
		t.Fatalf("references = %#v %#v", next.DirectSupports, next.Defeats)
	}
	if root, _ := next.MetadataValue("root"); root != "C1" {
		t.Fatalf("root = %q", root)
	}
	if doc.Junctors[0].ID != "old-j" || doc.Statements[0].ID != "old-p" {
		t.Fatal("renumber mutated caller")
	}
}
