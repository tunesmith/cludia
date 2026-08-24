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
