// SPDX-FileCopyrightText: 2026 KeenWorks
// SPDX-License-Identifier: GPL-3.0-or-later

package argument

import (
	"fmt"
	"strings"
)

// BatchReference names either a new element by caller key or an element that
// existed before the batch by durable ID. Exactly one field must be set.
type BatchReference struct {
	Key string
	ID  string
}

type BatchStatementSpec struct {
	Key         string
	Text        string
	RequestedID string
	Slug        string
	Truth       *Truth
	Kind        Kind
	Role        *Role
}

type BatchDerivationSpec struct {
	Key     string
	Sources []BatchReference
	Target  BatchReference
}

type BatchDefeatSpec struct {
	From   BatchReference
	Scope  DefeatScope
	Target BatchReference
}

type AuthorBatchOptions struct {
	Statements  []BatchStatementSpec
	Derivations []BatchDerivationSpec
	Defeats     []BatchDefeatSpec
}

type BatchStatementResult struct {
	Key       string
	Statement Statement
}

type BatchDerivationResult struct {
	Key     string
	Target  Statement
	Junctor Junctor
}

type BatchDefeatResult struct {
	From   BatchReference
	Defeat Defeat
}

type AuthorBatchResult struct {
	Statements          []BatchStatementResult
	Derivations         []BatchDerivationResult
	Defeats             []BatchDefeatResult
	RoleChanges         []StatementRoleChange
	TruthChanges        []StatementTruthChange
	RootMetadataUpdated bool
}

// AuthorBatch applies a complete caller-keyed authoring transaction to a
// clone. New statement roles are allocated from the final requested topology,
// so a newly created derivation target receives an L ID directly rather than
// consuming a transient P ID. Profile validation and persistence remain the
// workspace application's responsibility.
func AuthorBatch(doc *Document, options AuthorBatchOptions) (*Document, AuthorBatchResult, error) {
	empty := AuthorBatchResult{
		Statements: []BatchStatementResult{}, Derivations: []BatchDerivationResult{},
		Defeats: []BatchDefeatResult{}, RoleChanges: []StatementRoleChange{},
		TruthChanges: []StatementTruthChange{},
	}
	if doc == nil {
		return nil, empty, mutationError("document_nil", "document is nil", "")
	}
	if len(options.Statements)+len(options.Derivations)+len(options.Defeats) == 0 {
		return nil, empty, mutationError("batch_operations_required", "batch requires at least one statement, derivation, or defeat", "")
	}

	keyKinds := make(map[string]string, len(options.Statements)+len(options.Derivations))
	statementSpecs := make(map[string]BatchStatementSpec, len(options.Statements))
	for index, spec := range options.Statements {
		key := strings.TrimSpace(spec.Key)
		if err := registerBatchKey(keyKinds, key, "statement", fmt.Sprintf("statements[%d]", index)); err != nil {
			return nil, empty, err
		}
		spec.Key = key
		statementSpecs[key] = spec
	}
	for index, spec := range options.Derivations {
		key := strings.TrimSpace(spec.Key)
		if err := registerBatchKey(keyKinds, key, "derivation", fmt.Sprintf("derivations[%d]", index)); err != nil {
			return nil, empty, err
		}
		if len(spec.Sources) < 2 {
			return nil, empty, mutationError("derive_sources_too_few", fmt.Sprintf("derivation %q requires at least two source statements", key), key)
		}
		if err := validateBatchReference(spec.Target, fmt.Sprintf("derivations[%d].target", index)); err != nil {
			return nil, empty, err
		}
		for sourceIndex, source := range spec.Sources {
			if err := validateBatchReference(source, fmt.Sprintf("derivations[%d].sources[%d]", index, sourceIndex)); err != nil {
				return nil, empty, err
			}
		}
	}
	for index, spec := range options.Defeats {
		if err := validateBatchReference(spec.From, fmt.Sprintf("defeats[%d].from", index)); err != nil {
			return nil, empty, err
		}
		if err := validateBatchReference(spec.Target, fmt.Sprintf("defeats[%d].target", index)); err != nil {
			return nil, empty, err
		}
	}

	derivedStatementKeys := make(map[string]bool)
	for _, spec := range options.Derivations {
		if key := strings.TrimSpace(spec.Target.Key); key != "" {
			derivedStatementKeys[key] = true
		}
	}
	for key := range derivedStatementKeys {
		if kind := keyKinds[key]; kind != "statement" {
			return nil, empty, mutationError("batch_statement_key_not_found", fmt.Sprintf("derivation target key %q does not name a batch statement", key), key)
		}
	}

	next := doc.Clone()
	originalStatementIDs := make(map[string]string, len(doc.Statements))
	originalJunctorIDs := make(map[string]string, len(doc.Junctors))
	for _, statement := range doc.Statements {
		originalStatementIDs[statement.ID] = statement.ID
	}
	for _, junctor := range doc.Junctors {
		originalJunctorIDs[junctor.ID] = junctor.ID
	}
	statementIDsByKey := make(map[string]string, len(options.Statements))
	junctorIDsByKey := make(map[string]string, len(options.Derivations))

	allocator, err := NewIDAllocator(next)
	if err != nil {
		return nil, empty, err
	}
	for index, originalSpec := range options.Statements {
		spec := statementSpecs[strings.TrimSpace(originalSpec.Key)]
		role, inferred, err := batchStatementRole(spec, derivedStatementKeys[spec.Key])
		if err != nil {
			return nil, empty, err
		}
		kind := spec.Kind
		if kind == "" {
			kind = KindFact
		}
		truth := defaultTruthForRole(role)
		if spec.Truth != nil {
			truth = *spec.Truth
		}
		if role == RoleLemma || role == RoleConclusion {
			if truth != TruthUnknown {
				if !inferred {
					return nil, empty, mutationError("truth_assignment_role", fmt.Sprintf("batch statement %q has role %s and must use truth U", spec.Key, role), spec.Key)
				}
				truth = TruthUnknown
			}
		}
		if role == RoleCounterpoint && derivedStatementKeys[spec.Key] && truth != TruthUnknown {
			truth = TruthUnknown
		}
		statement, buildErr := buildStatementForRole(next, allocator, StatementInput{
			Text: spec.Text, RequestedID: spec.RequestedID, Slug: spec.Slug,
			Truth: truth, Kind: kind,
		}, role)
		if buildErr != nil {
			return nil, empty, batchContextError("statements", index, spec.Key, buildErr)
		}
		next.Statements = append(next.Statements, statement)
		statementIDsByKey[spec.Key] = statement.ID
		if spec.Truth != nil && *spec.Truth != truth {
			empty.TruthChanges = append(empty.TruthChanges, StatementTruthChange{ID: statement.ID, From: *spec.Truth, To: truth})
		}
	}
	allocator.Persist(next)

	for index, spec := range options.Derivations {
		sources := make([]string, 0, len(spec.Sources))
		for sourceIndex, source := range spec.Sources {
			id, resolveErr := resolveBatchStatementReference(next, source, statementIDsByKey, originalStatementIDs)
			if resolveErr != nil {
				return nil, empty, batchContextError("derivations", index, fmt.Sprintf("%s source %d", strings.TrimSpace(spec.Key), sourceIndex+1), resolveErr)
			}
			sources = append(sources, id)
		}
		targetID, resolveErr := resolveBatchStatementReference(next, spec.Target, statementIDsByKey, originalStatementIDs)
		if resolveErr != nil {
			return nil, empty, batchContextError("derivations", index, strings.TrimSpace(spec.Key), resolveErr)
		}
		derived, deriveResult, deriveErr := Derive(next, DeriveOptions{
			SourceRefs: sources, ExistingTargetRef: targetID,
		})
		if deriveErr != nil {
			return nil, empty, batchContextError("derivations", index, strings.TrimSpace(spec.Key), deriveErr)
		}
		next = derived
		for _, change := range deriveResult.RoleChanges {
			updateBatchCurrentIDs(statementIDsByKey, change.PreviousID, change.CurrentID)
			updateBatchCurrentIDs(originalStatementIDs, change.PreviousID, change.CurrentID)
		}
		junctorIDsByKey[strings.TrimSpace(spec.Key)] = deriveResult.Junctor.ID
		empty.RoleChanges = append(empty.RoleChanges, deriveResult.RoleChanges...)
		empty.TruthChanges = append(empty.TruthChanges, deriveResult.TruthChanges...)
		empty.RootMetadataUpdated = empty.RootMetadataUpdated || deriveResult.RootMetadataUpdated
	}

	defeatsBySource := make(map[string]bool, len(next.Defeats)+len(options.Defeats))
	for _, defeat := range next.Defeats {
		defeatsBySource[defeat.From] = true
	}
	for index, spec := range options.Defeats {
		fromID, resolveErr := resolveBatchStatementReference(next, spec.From, statementIDsByKey, originalStatementIDs)
		if resolveErr != nil {
			return nil, empty, batchContextError("defeats", index, "from", resolveErr)
		}
		from, _ := next.Statement(fromID)
		if from.Role != RoleCounterpoint {
			return nil, empty, mutationError("defeat_source_role", fmt.Sprintf("defeat source %s must be a counterpoint", from.ID), from.ID)
		}
		if defeatsBySource[from.ID] {
			return nil, empty, mutationError("defeat_source_multiple", fmt.Sprintf("counterpoint %s already has a defeat target in .arg syntax", from.ID), from.ID)
		}
		var targetID string
		if spec.Scope == DefeatInference {
			targetID, resolveErr = resolveBatchJunctorReference(next, spec.Target, junctorIDsByKey, originalJunctorIDs)
		} else {
			targetID, resolveErr = resolveBatchStatementReference(next, spec.Target, statementIDsByKey, originalStatementIDs)
		}
		if resolveErr != nil {
			return nil, empty, batchContextError("defeats", index, "target", resolveErr)
		}
		defeat, defeatErr := defeatForTarget(next, from.ID, spec.Scope, targetID)
		if defeatErr != nil {
			return nil, empty, batchContextError("defeats", index, from.ID, defeatErr)
		}
		next.Defeats = append(next.Defeats, defeat)
		defeatsBySource[from.ID] = true
		empty.Defeats = append(empty.Defeats, BatchDefeatResult{From: spec.From, Defeat: defeat})
	}

	for _, spec := range options.Statements {
		key := strings.TrimSpace(spec.Key)
		statement, _ := next.Statement(statementIDsByKey[key])
		empty.Statements = append(empty.Statements, BatchStatementResult{Key: key, Statement: *statement})
	}
	for _, spec := range options.Derivations {
		key := strings.TrimSpace(spec.Key)
		junctor, _ := next.Junctor(junctorIDsByKey[key])
		target, _ := next.Statement(junctor.Target)
		empty.Derivations = append(empty.Derivations, BatchDerivationResult{Key: key, Target: *target, Junctor: *junctor})
	}
	return next, empty, nil
}

func registerBatchKey(keys map[string]string, key, kind, element string) error {
	if key == "" {
		return mutationError("batch_key_required", fmt.Sprintf("%s requires a non-empty caller key", element), element)
	}
	if prior := keys[key]; prior != "" {
		return mutationError("batch_key_duplicate", fmt.Sprintf("batch key %q is already used by a %s", key, prior), key)
	}
	keys[key] = kind
	return nil
}

func validateBatchReference(reference BatchReference, element string) error {
	key, id := strings.TrimSpace(reference.Key), strings.TrimSpace(reference.ID)
	if (key == "") == (id == "") {
		return mutationError("batch_reference_invalid", fmt.Sprintf("%s must contain exactly one of key or id", element), element)
	}
	return nil
}

func batchStatementRole(spec BatchStatementSpec, derived bool) (Role, bool, error) {
	if spec.Role == nil {
		if derived {
			return RoleLemma, true, nil
		}
		return RolePremise, true, nil
	}
	role := *spec.Role
	switch role {
	case RolePremise, RoleLemma, RoleConclusion, RoleCounterpoint:
	default:
		return "", false, mutationError("statement_role_invalid", fmt.Sprintf("batch statement %q has invalid role %q", spec.Key, role), spec.Key)
	}
	if derived && role == RolePremise {
		return "", false, mutationError("batch_statement_role_conflict", fmt.Sprintf("batch statement %q is a derivation target and cannot remain a premise; omit role to infer lemma or specify lemma, conclusion, or counterpoint", spec.Key), spec.Key)
	}
	return role, false, nil
}

func defaultTruthForRole(role Role) Truth {
	if role == RoleLemma || role == RoleConclusion {
		return TruthUnknown
	}
	return TruthTrue
}

func resolveBatchStatementReference(doc *Document, reference BatchReference, keyed, original map[string]string) (string, error) {
	if key := strings.TrimSpace(reference.Key); key != "" {
		id, ok := keyed[key]
		if !ok {
			return "", mutationError("batch_statement_key_not_found", fmt.Sprintf("batch statement key %q was not found", key), key)
		}
		if _, ok := statementByExactID(doc, id); !ok {
			return "", mutationError("batch_statement_key_stale", fmt.Sprintf("batch statement key %q no longer resolves to a current statement", key), key)
		}
		return id, nil
	}
	id := strings.TrimSpace(reference.ID)
	current, ok := original[id]
	if !ok {
		return "", mutationError("batch_statement_id_not_found", fmt.Sprintf("pre-existing statement id %q was not found", id), id)
	}
	if _, ok := statementByExactID(doc, current); !ok {
		return "", mutationError("batch_statement_id_stale", fmt.Sprintf("pre-existing statement id %q no longer resolves to a current statement", id), id)
	}
	return current, nil
}

func resolveBatchJunctorReference(doc *Document, reference BatchReference, keyed, original map[string]string) (string, error) {
	if key := strings.TrimSpace(reference.Key); key != "" {
		id, ok := keyed[key]
		if !ok {
			return "", mutationError("batch_derivation_key_not_found", fmt.Sprintf("batch derivation key %q was not found", key), key)
		}
		if _, ok := doc.Junctor(id); !ok {
			return "", mutationError("batch_derivation_key_stale", fmt.Sprintf("batch derivation key %q no longer resolves to a current junctor", key), key)
		}
		return id, nil
	}
	id := strings.TrimSpace(reference.ID)
	current, ok := original[id]
	if !ok {
		return "", mutationError("batch_junctor_id_not_found", fmt.Sprintf("pre-existing junctor id %q was not found", id), id)
	}
	if _, ok := doc.Junctor(current); !ok {
		return "", mutationError("batch_junctor_id_stale", fmt.Sprintf("pre-existing junctor id %q no longer resolves to a current junctor", id), id)
	}
	return current, nil
}

func statementByExactID(doc *Document, id string) (*Statement, bool) {
	for index := range doc.Statements {
		if doc.Statements[index].ID == id {
			return &doc.Statements[index], true
		}
	}
	return nil, false
}

func updateBatchCurrentIDs(values map[string]string, previousID, currentID string) {
	for key, value := range values {
		if value == previousID {
			values[key] = currentID
		}
	}
}

func batchContextError(section string, index int, key string, err error) error {
	prefix := fmt.Sprintf("%s[%d]", section, index)
	if strings.TrimSpace(key) != "" {
		prefix += " " + strings.TrimSpace(key)
	}
	switch typed := err.(type) {
	case *MutationError:
		return mutationError(typed.Code, prefix+": "+typed.Message, typed.Element)
	case *MutationErrors:
		failures := make([]MutationError, 0, len(typed.Failures))
		for _, failure := range typed.Failures {
			failures = append(failures, MutationError{Code: failure.Code, Message: prefix + ": " + failure.Message, Element: failure.Element})
		}
		return &MutationErrors{Failures: failures}
	case *DeriveError:
		failures := make([]MutationError, 0, len(typed.Failures))
		for _, failure := range typed.Failures {
			failures = append(failures, MutationError{Code: failure.Code, Message: prefix + ": " + failure.Message, Element: failure.Element})
		}
		return &MutationErrors{Failures: failures}
	case *AddDefeatError:
		return mutationError(typed.Failure.Code, prefix+": "+typed.Failure.Message, typed.Failure.Element)
	default:
		return err
	}
}
