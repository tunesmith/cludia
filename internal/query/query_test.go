package query

import (
	"testing"

	"github.com/tunesmith/cludia/internal/argument"
)

func TestIsolationIncludesSupportAndDefeatIncidence(t *testing.T) {
	doc := &argument.Document{
		Statements: []argument.Statement{{ID: "P1"}, {ID: "P2"}, {ID: "P3"}, {ID: "CP1"}},
		Junctors:   []argument.Junctor{{ID: "J1", Sources: []string{"P1", "P2"}, Target: "P3"}},
		Defeats:    []argument.Defeat{{From: "CP1", Scope: argument.DefeatPremise, To: "P1"}},
	}
	isolated := IsolatedStatementIDs(doc)
	if len(isolated) != 0 {
		t.Fatalf("unexpected isolated statements: %#v", isolated)
	}
	doc.Statements = append(doc.Statements, argument.Statement{ID: "P4"})
	isolated = IsolatedStatementIDs(doc)
	if len(isolated) != 1 || !isolated["P4"] {
		t.Fatalf("isolated = %#v, want P4", isolated)
	}
}

func TestStatementAndJunctorRelations(t *testing.T) {
	doc := &argument.Document{
		Junctors: []argument.Junctor{{ID: "J1", Connector: argument.ConnectorAND, Sources: []string{"P1", "P2"}, Target: "L1"}},
		Defeats: []argument.Defeat{
			{From: "CP1", Scope: argument.DefeatInference, JunctorID: "J1", AtTarget: "L1"},
			{From: "CP2", Scope: argument.DefeatPremise, To: "P1"},
		},
	}
	statement := StatementRelations(doc, "P1")
	if len(statement.OutgoingSupport) != 1 || len(statement.DefeatsTargeting) != 1 {
		t.Fatalf("statement relations: %#v", statement)
	}
	target := StatementRelations(doc, "L1")
	if len(target.IncomingSupport) != 1 || len(target.DefeatsTargeting) != 1 {
		t.Fatalf("target relations: %#v", target)
	}
	junctor := JunctorRelations(doc, "J1")
	if len(junctor.DefeatsTargeting) != 1 {
		t.Fatalf("junctor relations: %#v", junctor)
	}
}

func TestComponentsGroupSupportDefeatAndIsolatedStatements(t *testing.T) {
	doc := &argument.Document{
		Statements: []argument.Statement{
			{ID: "P1", Slug: "first"}, {ID: "P2"}, {ID: "L1"},
			{ID: "P3"}, {ID: "CP1"}, {ID: "P4"},
		},
		Junctors:       []argument.Junctor{{ID: "J1", Sources: []string{"P1", "P2"}, Target: "L1"}},
		DirectSupports: []argument.DirectSupport{{Source: "P3", Target: "CP1"}},
		Defeats:        []argument.Defeat{{From: "CP1", Scope: argument.DefeatPremise, To: "P2"}},
	}
	components := Components(doc)
	if len(components) != 2 {
		t.Fatalf("components = %#v, want 2", components)
	}
	connected := components[0]
	if connected.Anchor != "P1" || len(connected.Statements) != 5 || len(connected.Junctors) != 1 || len(connected.DirectSupports) != 1 || len(connected.Defeats) != 1 || connected.Isolated {
		t.Fatalf("connected component = %#v", connected)
	}
	isolated := components[1]
	if isolated.Anchor != "P4" || len(isolated.Statements) != 1 || !isolated.Isolated {
		t.Fatalf("isolated component = %#v", isolated)
	}
}

func TestInferenceDefeatConnectsCounterpointToJunctorComponent(t *testing.T) {
	doc := &argument.Document{
		Statements: []argument.Statement{{ID: "P1"}, {ID: "P2"}, {ID: "L1"}, {ID: "CP1"}},
		Junctors:   []argument.Junctor{{ID: "J1", Sources: []string{"P1", "P2"}, Target: "L1"}},
		Defeats:    []argument.Defeat{{From: "CP1", Scope: argument.DefeatInference, JunctorID: "J1", AtTarget: "L1"}},
	}
	components := Components(doc)
	if len(components) != 1 || len(components[0].Statements) != 4 || len(components[0].Defeats) != 1 {
		t.Fatalf("inference defeat components = %#v", components)
	}
}

func TestComponentContainingAcceptsSlugAndJunctorID(t *testing.T) {
	doc := &argument.Document{
		Statements: []argument.Statement{{ID: "P1", Slug: "first"}, {ID: "P2"}, {ID: "L1"}},
		Junctors:   []argument.Junctor{{ID: "J1", Sources: []string{"P1", "P2"}, Target: "L1"}},
	}
	bySlug, ok := ComponentContaining(doc, "first")
	if !ok || bySlug.Anchor != "P1" {
		t.Fatalf("component by slug = %#v, %t", bySlug, ok)
	}
	byJunctor, ok := ComponentContaining(doc, "J1")
	if !ok || byJunctor.Anchor != "P1" {
		t.Fatalf("component by junctor = %#v, %t", byJunctor, ok)
	}
}
