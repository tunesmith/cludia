// SPDX-FileCopyrightText: 2026 KeenWorks
// SPDX-License-Identifier: GPL-3.0-or-later

package argument

import (
	"fmt"
	"strings"
)

// NewDerivedTarget describes a statement created together with its first
// justification.
type NewDerivedTarget struct {
	Text        string
	RequestedID string
	Slug        string
	Kind        Kind
	Role        Role
}

// DeriveOptions identifies the sources and either an existing or new target
// for one focused AND inference.
type DeriveOptions struct {
	SourceRefs         []string
	ExistingTargetRef  string
	NewTarget          *NewDerivedTarget
	RequestedJunctorID string
}

// StatementRoleChange reports a role transition and any role-driven durable
// ID change. PreviousID and CurrentID differ when a premise becomes a lemma.
type StatementRoleChange struct {
	PreviousID string
	CurrentID  string
	From       Role
	To         Role
}

type StatementTruthChange struct {
	ID   string
	From Truth
	To   Truth
}

// DeriveResult contains presentation-neutral facts about one derivation.
type DeriveResult struct {
	Target              Statement
	Junctor             Junctor
	RoleChanges         []StatementRoleChange
	TruthChanges        []StatementTruthChange
	RootMetadataUpdated bool
}

// DeriveFailure is a stable semantic failure suitable for interface-specific
// diagnostics.
type DeriveFailure struct {
	Code    string
	Message string
	Element string
}

// DeriveError may contain more than one independently unresolved source.
type DeriveError struct {
	Failures []DeriveFailure
}

func (e *DeriveError) Error() string {
	if len(e.Failures) == 0 {
		return "derive failed"
	}
	return e.Failures[0].Message
}

// Derive returns a cloned document containing one new focused AND inference.
// An existing premise target is promoted to a lemma, assigned the next
// monotonic L ID, and all modeled references to its retired P ID are rewritten.
func Derive(doc *Document, options DeriveOptions) (*Document, DeriveResult, error) {
	if doc == nil {
		return nil, DeriveResult{}, &DeriveError{Failures: []DeriveFailure{{Code: "document_nil", Message: "document is nil"}}}
	}
	if len(options.SourceRefs) < 2 {
		return nil, DeriveResult{}, &DeriveError{Failures: []DeriveFailure{{
			Code: "derive_sources_too_few", Message: "derive requires at least two source statements",
		}}}
	}
	existingTargetRef := strings.TrimSpace(options.ExistingTargetRef)
	if (existingTargetRef == "") == (options.NewTarget == nil) {
		return nil, DeriveResult{}, &DeriveError{Failures: []DeriveFailure{{
			Code: "derive_target_invalid", Message: "derive requires exactly one existing or new target",
		}}}
	}

	next := doc.Clone()
	allocator, err := NewIDAllocator(next)
	if err != nil {
		return nil, DeriveResult{}, err
	}

	resolvedSources := make([]string, 0, len(options.SourceRefs))
	failures := make([]DeriveFailure, 0)
	for _, reference := range options.SourceRefs {
		cleanReference := strings.TrimSpace(reference)
		statement, ok := next.Statement(cleanReference)
		if !ok {
			failures = append(failures, DeriveFailure{
				Code: "source_not_found", Message: fmt.Sprintf("source statement %q not found", reference), Element: reference,
			})
			continue
		}
		resolvedSources = append(resolvedSources, statement.ID)
	}
	if len(failures) > 0 {
		return nil, DeriveResult{}, &DeriveError{Failures: failures}
	}

	result := DeriveResult{RoleChanges: []StatementRoleChange{}, TruthChanges: []StatementTruthChange{}}
	var target *Statement
	if existingTargetRef != "" {
		var ok bool
		target, ok = next.Statement(existingTargetRef)
		if !ok {
			return nil, DeriveResult{}, &DeriveError{Failures: []DeriveFailure{{
				Code: "target_not_found", Message: fmt.Sprintf("target statement %q not found", existingTargetRef), Element: existingTargetRef,
			}}}
		}
		if target.Role == RolePremise {
			previousID := target.ID
			previousTruth := target.Truth
			currentID, allocationErr := allocator.Statement(RoleLemma, "")
			if allocationErr != nil {
				return nil, DeriveResult{}, allocationErr
			}
			result.RootMetadataUpdated = rewriteStatementID(next, previousID, currentID)
			for index, sourceID := range resolvedSources {
				if sourceID == previousID {
					resolvedSources[index] = currentID
				}
			}
			target, _ = next.Statement(currentID)
			target.Role = RoleLemma
			target.Truth = TruthUnknown
			result.RoleChanges = append(result.RoleChanges, StatementRoleChange{
				PreviousID: previousID, CurrentID: currentID, From: RolePremise, To: RoleLemma,
			})
			if previousTruth != TruthUnknown {
				result.TruthChanges = append(result.TruthChanges, StatementTruthChange{ID: currentID, From: previousTruth, To: TruthUnknown})
			}
		} else if target.Role == RoleCounterpoint && target.Truth != TruthUnknown {
			previousTruth := target.Truth
			target.Truth = TruthUnknown
			result.TruthChanges = append(result.TruthChanges, StatementTruthChange{ID: target.ID, From: previousTruth, To: TruthUnknown})
		}
	} else {
		newTarget := options.NewTarget
		if strings.TrimSpace(newTarget.Text) == "" {
			return nil, DeriveResult{}, &DeriveError{Failures: []DeriveFailure{{
				Code: "statement_text_required", Message: "target statement text is required", Element: strings.TrimSpace(newTarget.RequestedID),
			}}}
		}
		if newTarget.Role != RoleLemma && newTarget.Role != RoleConclusion {
			return nil, DeriveResult{}, &DeriveError{Failures: []DeriveFailure{{
				Code: "role_invalid", Message: fmt.Sprintf("invalid target role %q; expected lemma or conclusion", newTarget.Role), Element: strings.TrimSpace(newTarget.RequestedID),
			}}}
		}
		id, allocationErr := allocator.Statement(newTarget.Role, strings.TrimSpace(newTarget.RequestedID))
		if allocationErr != nil {
			return nil, DeriveResult{}, allocationErr
		}
		slug := strings.TrimSpace(newTarget.Slug)
		if slug == "" {
			slug = UniqueSlug(next, newTarget.Text)
		} else if elementType, existingID, collides := SlugIDCollision(next, slug, id); collides {
			return nil, DeriveResult{}, &DeriveError{Failures: []DeriveFailure{{
				Code:    "statement_slug_id_collision",
				Message: fmt.Sprintf("slug %q would be shadowed by %s id %s; choose a different slug", slug, elementType, existingID),
				Element: id,
			}}}
		}
		next.Statements = append(next.Statements, Statement{
			ID: id, Slug: slug, Role: newTarget.Role, Kind: newTarget.Kind,
			Truth: TruthUnknown, Text: strings.TrimSpace(newTarget.Text),
		})
		target = &next.Statements[len(next.Statements)-1]
	}

	junctorID, allocationErr := allocator.Junctor(strings.TrimSpace(options.RequestedJunctorID))
	if allocationErr != nil {
		return nil, DeriveResult{}, allocationErr
	}
	junctor := Junctor{
		ID: junctorID, Connector: ConnectorAND,
		Sources: append([]string(nil), resolvedSources...), Target: target.ID,
		Order: nextRelationOrder(next),
	}
	next.Junctors = append(next.Junctors, junctor)
	allocator.Persist(next)

	result.Target = *target
	result.Junctor = junctor
	return next, result, nil
}

func rewriteStatementID(doc *Document, previousID, currentID string) bool {
	for index := range doc.Statements {
		if doc.Statements[index].ID == previousID {
			doc.Statements[index].ID = currentID
		}
	}
	for index := range doc.Junctors {
		junctor := &doc.Junctors[index]
		if junctor.Target == previousID {
			junctor.Target = currentID
		}
		for sourceIndex := range junctor.Sources {
			if junctor.Sources[sourceIndex] == previousID {
				junctor.Sources[sourceIndex] = currentID
			}
		}
	}
	for index := range doc.DirectSupports {
		support := &doc.DirectSupports[index]
		if support.Source == previousID {
			support.Source = currentID
		}
		if support.Target == previousID {
			support.Target = currentID
		}
	}
	for index := range doc.Defeats {
		defeat := &doc.Defeats[index]
		if defeat.From == previousID {
			defeat.From = currentID
		}
		if defeat.To == previousID {
			defeat.To = currentID
		}
		if defeat.AtTarget == previousID {
			defeat.AtTarget = currentID
		}
	}
	rootUpdated := false
	for index := range doc.Metadata {
		if doc.Metadata[index].Key == "root" && doc.Metadata[index].Value == previousID {
			doc.Metadata[index].Value = currentID
			rootUpdated = true
		}
	}
	return rootUpdated
}

func nextRelationOrder(doc *Document) int {
	maximum := 0
	for _, junctor := range doc.Junctors {
		if junctor.Order > maximum {
			maximum = junctor.Order
		}
	}
	for _, support := range doc.DirectSupports {
		if support.Order > maximum {
			maximum = support.Order
		}
	}
	return maximum + 1
}
