// SPDX-FileCopyrightText: 2026 KeenWorks
// SPDX-License-Identifier: GPL-3.0-or-later

package argument

import "fmt"

// MovePlacement identifies which side of an anchor receives a moved statement.
type MovePlacement string

const (
	MoveBefore MovePlacement = "before"
	MoveAfter  MovePlacement = "after"
)

// StatementMove describes one durable document-order change. Positions are
// one-based so they match the human-readable statement sequence.
type StatementMove struct {
	Statement        Statement
	Anchor           Statement
	Placement        MovePlacement
	PreviousPosition int
	CurrentPosition  int
	Changed          bool
}

// StatementMoveError is a stable domain failure suitable for CLI diagnostics.
type StatementMoveError struct {
	Code    string
	Message string
	Element string
}

func (e *StatementMoveError) Error() string { return e.Message }

// MoveStatement returns a cloned document with one statement immediately
// before or after another. All statements other than the moved statement retain
// their relative order.
func MoveStatement(doc *Document, statementRef, anchorRef string, placement MovePlacement) (*Document, StatementMove, error) {
	if doc == nil {
		return nil, StatementMove{}, &StatementMoveError{Code: "document_nil", Message: "document is nil"}
	}
	if placement != MoveBefore && placement != MoveAfter {
		return nil, StatementMove{}, &StatementMoveError{
			Code: "statement_move_placement_invalid", Message: fmt.Sprintf("invalid statement move placement %q", placement), Element: string(placement),
		}
	}
	statement, ok := doc.Statement(statementRef)
	if !ok {
		return nil, StatementMove{}, &StatementMoveError{
			Code: "statement_not_found", Message: fmt.Sprintf("statement %q not found", statementRef), Element: statementRef,
		}
	}
	anchor, ok := doc.Statement(anchorRef)
	if !ok {
		return nil, StatementMove{}, &StatementMoveError{
			Code: "statement_anchor_not_found", Message: fmt.Sprintf("anchor statement %q not found", anchorRef), Element: anchorRef,
		}
	}
	if statement.ID == anchor.ID {
		return nil, StatementMove{}, &StatementMoveError{
			Code: "statement_move_same_anchor", Message: fmt.Sprintf("statement %s cannot be moved relative to itself", statement.ID), Element: statement.ID,
		}
	}

	previousIndex := statementIndex(doc.Statements, statement.ID)
	next := doc.Clone()
	without := make([]Statement, 0, len(next.Statements)-1)
	for _, candidate := range next.Statements {
		if candidate.ID != statement.ID {
			without = append(without, candidate)
		}
	}
	anchorIndex := statementIndex(without, anchor.ID)
	insertIndex := anchorIndex
	if placement == MoveAfter {
		insertIndex++
	}

	reordered := make([]Statement, 0, len(next.Statements))
	reordered = append(reordered, without[:insertIndex]...)
	reordered = append(reordered, *statement)
	reordered = append(reordered, without[insertIndex:]...)
	next.Statements = reordered
	currentIndex := statementIndex(next.Statements, statement.ID)

	return next, StatementMove{
		Statement: *statement, Anchor: *anchor, Placement: placement,
		PreviousPosition: previousIndex + 1, CurrentPosition: currentIndex + 1,
		Changed: previousIndex != currentIndex,
	}, nil
}

func statementIndex(statements []Statement, id string) int {
	for i := range statements {
		if statements[i].ID == id {
			return i
		}
	}
	return -1
}
