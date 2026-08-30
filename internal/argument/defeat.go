package argument

import (
	"fmt"
	"strings"
)

// AddDefeatOptions describes one focused counterpoint and its typed target.
type AddDefeatOptions struct {
	Scope       DefeatScope
	TargetRef   string
	Text        string
	RequestedID string
	Slug        string
	Truth       Truth
	Kind        Kind
}

// AddDefeatResult contains presentation-neutral facts about a created
// counterpoint and defeat relation.
type AddDefeatResult struct {
	Counterpoint Statement
	Defeat       Defeat
}

// AddDefeatFailure is a stable semantic failure suitable for
// interface-specific diagnostics.
type AddDefeatFailure struct {
	Code    string
	Message string
	Element string
}

type AddDefeatError struct {
	Failure AddDefeatFailure
}

func (e *AddDefeatError) Error() string { return e.Failure.Message }

// AddDefeat returns a cloned document with one new counterpoint and defeat.
// It does not validate a profile or persist the result.
func AddDefeat(doc *Document, options AddDefeatOptions) (*Document, AddDefeatResult, error) {
	if doc == nil {
		return nil, AddDefeatResult{}, addDefeatError("document_nil", "document is nil", "")
	}
	targetRef := strings.TrimSpace(options.TargetRef)
	if targetRef == "" {
		return nil, AddDefeatResult{}, addDefeatError("defeat_target_required", "defeat target is required", "")
	}
	text := strings.TrimSpace(options.Text)
	if text == "" {
		return nil, AddDefeatResult{}, addDefeatError("statement_text_required", "counterpoint text is required", strings.TrimSpace(options.RequestedID))
	}
	next := doc.Clone()
	defeat, err := defeatForTarget(next, "", options.Scope, targetRef)
	if err != nil {
		return nil, AddDefeatResult{}, err
	}

	allocator, err := NewIDAllocator(next)
	if err != nil {
		return nil, AddDefeatResult{}, err
	}
	id, err := allocator.Statement(RoleCounterpoint, strings.TrimSpace(options.RequestedID))
	if err != nil {
		return nil, AddDefeatResult{}, err
	}
	slug := strings.TrimSpace(options.Slug)
	if slug == "" {
		slug = UniqueSlug(next, text)
	} else if elementType, existingID, collides := SlugIDCollision(next, slug, id); collides {
		return nil, AddDefeatResult{}, addDefeatError(
			"statement_slug_id_collision",
			fmt.Sprintf("slug %q would be shadowed by %s id %s; choose a different slug", slug, elementType, existingID),
			id,
		)
	}
	failures := make([]MutationError, 0, 2)
	if !validTruth(options.Truth) {
		failures = append(failures, MutationError{Code: "truth_invalid", Message: fmt.Sprintf("invalid truth %q; expected T, F, or U", options.Truth), Element: id})
	}
	if !validKind(options.Kind) {
		failures = append(failures, MutationError{Code: "kind_invalid", Message: fmt.Sprintf("invalid kind %q; expected fact or value", options.Kind), Element: id})
	}
	if len(failures) > 0 {
		return nil, AddDefeatResult{}, &MutationErrors{Failures: failures}
	}
	counterpoint := Statement{
		ID: id, Slug: slug, Role: RoleCounterpoint, Kind: options.Kind,
		Truth: options.Truth, Text: text,
	}
	defeat.From = id
	next.Statements = append(next.Statements, counterpoint)
	next.Defeats = append(next.Defeats, defeat)
	allocator.Persist(next)

	return next, AddDefeatResult{Counterpoint: counterpoint, Defeat: defeat}, nil
}

func defeatForTarget(doc *Document, from string, scope DefeatScope, targetRef string) (Defeat, error) {
	defeat := Defeat{From: from, Scope: scope}
	switch scope {
	case DefeatPremise:
		target, ok := doc.Statement(targetRef)
		if !ok {
			return Defeat{}, addDefeatError("target_not_found", fmt.Sprintf("premise %q not found", targetRef), targetRef)
		}
		if target.Role != RolePremise {
			return Defeat{}, addDefeatError("undermine_target_role", fmt.Sprintf("undermine target %s has role %s, expected premise", target.ID, target.Role), target.ID)
		}
		defeat.To = target.ID
	case DefeatInference:
		junctor, ok := doc.Junctor(targetRef)
		if !ok {
			return Defeat{}, addDefeatError("junctor_not_found", fmt.Sprintf("junctor %q not found", targetRef), targetRef)
		}
		defeat.JunctorID, defeat.AtTarget = junctor.ID, junctor.Target
	case DefeatCounterpoint:
		target, ok := doc.Statement(targetRef)
		if !ok {
			return Defeat{}, addDefeatError("target_not_found", fmt.Sprintf("counterpoint %q not found", targetRef), targetRef)
		}
		if target.Role != RoleCounterpoint {
			return Defeat{}, addDefeatError("counterpoint_target_role", fmt.Sprintf("counterpoint target %s has role %s, expected counterpoint", target.ID, target.Role), target.ID)
		}
		defeat.To = target.ID
	default:
		return Defeat{}, addDefeatError("defeat_scope_invalid", fmt.Sprintf("invalid defeat scope %q", scope), string(scope))
	}
	return defeat, nil
}

func addDefeatError(code, message, element string) *AddDefeatError {
	return &AddDefeatError{Failure: AddDefeatFailure{Code: code, Message: message, Element: element}}
}
