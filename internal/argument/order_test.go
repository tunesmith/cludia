// SPDX-FileCopyrightText: 2026 KeenWorks
// SPDX-License-Identifier: GPL-3.0-or-later

package argument

import (
	"reflect"
	"testing"
)

func TestMoveStatementBeforeAndAfterPreservesOtherOrder(t *testing.T) {
	doc := orderDocument()
	next, move, err := MoveStatement(doc, "third", "P1", MoveBefore)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := statementIDs(next), []string{"P3", "P1", "P2", "CP1"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("before order = %v, want %v", got, want)
	}
	if !move.Changed || move.Statement.ID != "P3" || move.Anchor.ID != "P1" || move.PreviousPosition != 3 || move.CurrentPosition != 1 {
		t.Fatalf("before move = %#v", move)
	}
	if got := statementIDs(doc); !reflect.DeepEqual(got, []string{"P1", "P2", "P3", "CP1"}) {
		t.Fatalf("input document changed: %v", got)
	}

	next, move, err = MoveStatement(doc, "first", "P3", MoveAfter)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := statementIDs(next), []string{"P2", "P3", "P1", "CP1"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("after order = %v, want %v", got, want)
	}
	if move.PreviousPosition != 1 || move.CurrentPosition != 3 || move.Placement != MoveAfter {
		t.Fatalf("after move = %#v", move)
	}
}

func TestMoveStatementNoOpAndFailures(t *testing.T) {
	doc := orderDocument()
	next, move, err := MoveStatement(doc, "P2", "P1", MoveAfter)
	if err != nil {
		t.Fatal(err)
	}
	if move.Changed || !reflect.DeepEqual(statementIDs(next), statementIDs(doc)) {
		t.Fatalf("no-op move = %#v order=%v", move, statementIDs(next))
	}

	tests := []struct {
		name, statement, anchor string
		placement               MovePlacement
		code                    string
	}{
		{name: "missing statement", statement: "missing", anchor: "P1", placement: MoveBefore, code: "statement_not_found"},
		{name: "missing anchor", statement: "P1", anchor: "missing", placement: MoveBefore, code: "statement_anchor_not_found"},
		{name: "same statement", statement: "P1", anchor: "first", placement: MoveBefore, code: "statement_move_same_anchor"},
		{name: "invalid placement", statement: "P1", anchor: "P2", placement: "around", code: "statement_move_placement_invalid"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := MoveStatement(doc, test.statement, test.anchor, test.placement)
			moveErr, ok := err.(*StatementMoveError)
			if !ok || moveErr.Code != test.code {
				t.Fatalf("error = %#v, want code %s", err, test.code)
			}
		})
	}
}

func orderDocument() *Document {
	return &Document{
		ID: "order", Title: "Order", Metadata: []Metadata{{Key: "profile", Value: "cludia"}},
		Statements: []Statement{
			{ID: "P1", Slug: "first", Role: RolePremise, Kind: KindFact, Truth: TruthTrue, Text: "First"},
			{ID: "P2", Slug: "second", Role: RolePremise, Kind: KindFact, Truth: TruthTrue, Text: "Second"},
			{ID: "P3", Slug: "third", Role: RolePremise, Kind: KindFact, Truth: TruthTrue, Text: "Third"},
			{ID: "CP1", Slug: "challenge", Role: RoleCounterpoint, Kind: KindFact, Truth: TruthTrue, Text: "Challenge"},
		},
		Defeats: []Defeat{{From: "CP1", Scope: DefeatPremise, To: "P1"}},
	}
}

func statementIDs(doc *Document) []string {
	ids := make([]string, 0, len(doc.Statements))
	for _, statement := range doc.Statements {
		ids = append(ids, statement.ID)
	}
	return ids
}
