package argument

import (
	"fmt"
	"strings"
)

type RemoveCounterpointResult struct {
	Counterpoint   Statement
	DefeatsRemoved []Defeat
}

type DeleteStatementResult struct {
	Statement             Statement
	JunctorsRemoved       []Junctor
	DirectSupportsRemoved []DirectSupport
	DefeatsRemoved        []Defeat
}

func RemoveCounterpoint(doc *Document, reference string) (*Document, RemoveCounterpointResult, error) {
	if doc == nil {
		return nil, RemoveCounterpointResult{}, mutationError("document_nil", "document is nil", "")
	}
	next := doc.Clone()
	if err := EnsureNextIDs(next); err != nil {
		return nil, RemoveCounterpointResult{}, err
	}
	statement, ok := next.Statement(strings.TrimSpace(reference))
	if !ok {
		return nil, RemoveCounterpointResult{}, mutationError("counterpoint_not_found", fmt.Sprintf("counterpoint %q not found", reference), reference)
	}
	if statement.Role != RoleCounterpoint {
		return nil, RemoveCounterpointResult{}, mutationError("counterpoint_role_required", fmt.Sprintf("statement %s has role %s, expected counterpoint", statement.ID, statement.Role), statement.ID)
	}
	for _, defeat := range next.Defeats {
		if defeat.Scope == DefeatCounterpoint && defeat.To == statement.ID {
			return nil, RemoveCounterpointResult{}, mutationError("counterpoint_has_dependents", fmt.Sprintf("counterpoint %s is targeted by %s; remove dependent counterpoints first", statement.ID, defeat.From), statement.ID)
		}
	}
	for _, junctor := range next.Junctors {
		if junctor.Target == statement.ID || containsID(junctor.Sources, statement.ID) {
			return nil, RemoveCounterpointResult{}, mutationError("counterpoint_has_support", fmt.Sprintf("counterpoint %s participates in junctor %s; remove its support relation first", statement.ID, junctor.ID), statement.ID)
		}
	}
	for _, support := range next.DirectSupports {
		if support.Source == statement.ID || support.Target == statement.ID {
			return nil, RemoveCounterpointResult{}, mutationError("counterpoint_has_support", fmt.Sprintf("counterpoint %s participates in direct support", statement.ID), statement.ID)
		}
	}
	removed := *statement
	statements := make([]Statement, 0, len(next.Statements)-1)
	for _, candidate := range next.Statements {
		if candidate.ID != statement.ID {
			statements = append(statements, candidate)
		}
	}
	next.Statements = statements
	removedDefeats := make([]Defeat, 0)
	defeats := make([]Defeat, 0, len(next.Defeats))
	for _, defeat := range next.Defeats {
		if defeat.From == statement.ID {
			removedDefeats = append(removedDefeats, defeat)
		} else {
			defeats = append(defeats, defeat)
		}
	}
	next.Defeats = defeats
	return next, RemoveCounterpointResult{Counterpoint: removed, DefeatsRemoved: removedDefeats}, nil
}

func DeleteStatement(doc *Document, reference string) (*Document, DeleteStatementResult, error) {
	if doc == nil {
		return nil, DeleteStatementResult{}, mutationError("document_nil", "document is nil", "")
	}
	next := doc.Clone()
	if err := EnsureNextIDs(next); err != nil {
		return nil, DeleteStatementResult{}, err
	}
	statement, ok := next.Statement(strings.TrimSpace(reference))
	if !ok {
		return nil, DeleteStatementResult{}, mutationError("statement_not_found", fmt.Sprintf("statement %q not found", reference), reference)
	}
	if statement.Role == RoleCounterpoint {
		return nil, DeleteStatementResult{}, mutationError("use_remove_counterpoint", fmt.Sprintf("statement %s is a counterpoint; use remove-counterpoint", statement.ID), statement.ID)
	}
	if len(next.Statements) == 1 {
		return nil, DeleteStatementResult{}, mutationError("last_statement", "a workspace must retain at least one statement", statement.ID)
	}
	removed := *statement
	removedJunctorIDs := make(map[string]bool)
	junctorsRemoved := make([]Junctor, 0)
	junctors := make([]Junctor, 0, len(next.Junctors))
	for _, junctor := range next.Junctors {
		if junctor.Target == statement.ID || containsID(junctor.Sources, statement.ID) {
			junctorsRemoved = append(junctorsRemoved, cloneJunctor(junctor))
			removedJunctorIDs[junctor.ID] = true
		} else {
			junctors = append(junctors, junctor)
		}
	}
	for _, defeat := range next.Defeats {
		if defeat.To == statement.ID || defeat.AtTarget == statement.ID || removedJunctorIDs[defeat.JunctorID] {
			return nil, DeleteStatementResult{}, mutationError("statement_has_defeats", fmt.Sprintf("deleting %s would detach counterpoint %s; remove the counterpoint first", statement.ID, defeat.From), statement.ID)
		}
	}
	next.Junctors = junctors
	directRemoved := make([]DirectSupport, 0)
	direct := make([]DirectSupport, 0, len(next.DirectSupports))
	for _, support := range next.DirectSupports {
		if support.Source == statement.ID || support.Target == statement.ID {
			directRemoved = append(directRemoved, support)
		} else {
			direct = append(direct, support)
		}
	}
	next.DirectSupports = direct
	defeatsRemoved := make([]Defeat, 0)
	defeats := make([]Defeat, 0, len(next.Defeats))
	for _, defeat := range next.Defeats {
		if defeat.From == statement.ID || defeat.To == statement.ID || defeat.AtTarget == statement.ID || removedJunctorIDs[defeat.JunctorID] {
			defeatsRemoved = append(defeatsRemoved, defeat)
		} else {
			defeats = append(defeats, defeat)
		}
	}
	next.Defeats = defeats
	statements := make([]Statement, 0, len(next.Statements)-1)
	for _, candidate := range next.Statements {
		if candidate.ID != statement.ID {
			statements = append(statements, candidate)
		}
	}
	next.Statements = statements
	return next, DeleteStatementResult{
		Statement: removed, JunctorsRemoved: junctorsRemoved,
		DirectSupportsRemoved: directRemoved, DefeatsRemoved: defeatsRemoved,
	}, nil
}

func containsID(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
