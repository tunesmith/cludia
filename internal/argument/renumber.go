// SPDX-FileCopyrightText: 2026 KeenWorks
// SPDX-License-Identifier: GPL-3.0-or-later

package argument

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

type StatementIDMapping struct {
	PreviousID string `json:"previous_id"`
	CurrentID  string `json:"current_id"`
	Role       Role   `json:"role"`
	Slug       string `json:"slug,omitempty"`
}

type JunctorIDMapping struct {
	PreviousID string `json:"previous_id"`
	CurrentID  string `json:"current_id"`
}

type RenumberResult struct {
	StatementIDs        []StatementIDMapping
	JunctorIDs          []JunctorIDMapping
	RootMetadataUpdated bool
	NextIDsBefore       NextIDs
	NextIDsAfter        NextIDs
	IDsChanged          bool
	PlanToken           string
}

func RenumberDocument(doc *Document) (*Document, RenumberResult, error) {
	if doc == nil {
		return nil, RenumberResult{}, mutationError("document_nil", "document is nil", "")
	}
	_, _, before, _, _, err := InspectNextIDs(doc)
	if err != nil {
		return nil, RenumberResult{}, err
	}
	statementCounters := map[Role]int{RolePremise: 1, RoleLemma: 1, RoleConclusion: 1, RoleCounterpoint: 1}
	statementMappings := make([]StatementIDMapping, 0, len(doc.Statements))
	statementMap := make(map[string]string, len(doc.Statements))
	for _, statement := range doc.Statements {
		number := statementCounters[statement.Role]
		currentID, ok := CanonicalStatementID(statement.Role, number)
		if !ok {
			return nil, RenumberResult{}, mutationError("statement_role_invalid", fmt.Sprintf("statement %s has unsupported role %q", statement.ID, statement.Role), statement.ID)
		}
		statementCounters[statement.Role] = number + 1
		statementMappings = append(statementMappings, StatementIDMapping{PreviousID: statement.ID, CurrentID: currentID, Role: statement.Role, Slug: statement.Slug})
		statementMap[statement.ID] = currentID
	}
	junctorMappings := make([]JunctorIDMapping, 0, len(doc.Junctors))
	junctorMap := make(map[string]string, len(doc.Junctors))
	for index, junctor := range doc.Junctors {
		currentID := CanonicalJunctorID(index + 1)
		junctorMappings = append(junctorMappings, JunctorIDMapping{PreviousID: junctor.ID, CurrentID: currentID})
		junctorMap[junctor.ID] = currentID
	}

	next := doc.Clone()
	idsChanged := false
	for index := range next.Statements {
		mapping := statementMappings[index]
		next.Statements[index].ID = mapping.CurrentID
		idsChanged = idsChanged || mapping.PreviousID != mapping.CurrentID
	}
	for index := range next.Junctors {
		mapping := junctorMappings[index]
		junctor := &next.Junctors[index]
		junctor.ID = mapping.CurrentID
		junctor.Target = mappedReference(statementMap, junctor.Target)
		for sourceIndex := range junctor.Sources {
			junctor.Sources[sourceIndex] = mappedReference(statementMap, junctor.Sources[sourceIndex])
		}
		idsChanged = idsChanged || mapping.PreviousID != mapping.CurrentID
	}
	for index := range next.DirectSupports {
		support := &next.DirectSupports[index]
		support.Source = mappedReference(statementMap, support.Source)
		support.Target = mappedReference(statementMap, support.Target)
	}
	for index := range next.Defeats {
		defeat := &next.Defeats[index]
		defeat.From = mappedReference(statementMap, defeat.From)
		defeat.To = mappedReference(statementMap, defeat.To)
		defeat.AtTarget = mappedReference(statementMap, defeat.AtTarget)
		defeat.JunctorID = mappedReference(junctorMap, defeat.JunctorID)
	}
	rootUpdated := false
	for index := range next.Metadata {
		metadata := &next.Metadata[index]
		if metadata.Key == "root" {
			if current, ok := statementMap[metadata.Value]; ok && current != metadata.Value {
				metadata.Value = current
				rootUpdated = true
			}
		}
	}
	after := NextIDs{
		P: statementCounters[RolePremise], L: statementCounters[RoleLemma],
		C: statementCounters[RoleConclusion], CP: statementCounters[RoleCounterpoint],
		J: len(next.Junctors) + 1,
	}
	SetNextIDs(next, after)
	token, err := renumberToken(doc, statementMappings, junctorMappings)
	if err != nil {
		return nil, RenumberResult{}, err
	}
	return next, RenumberResult{
		StatementIDs: statementMappings, JunctorIDs: junctorMappings,
		RootMetadataUpdated: rootUpdated, NextIDsBefore: before, NextIDsAfter: after,
		IDsChanged: idsChanged, PlanToken: token,
	}, nil
}

func mappedReference(mapping map[string]string, id string) string {
	if current, ok := mapping[id]; ok {
		return current
	}
	return id
}

func renumberToken(doc *Document, statements []StatementIDMapping, junctors []JunctorIDMapping) (string, error) {
	encoded, err := marshalDocumentState(doc)
	if err != nil {
		return "", err
	}
	var mapping strings.Builder
	for _, item := range statements {
		fmt.Fprintf(&mapping, "\x00statement:%s=%s", item.PreviousID, item.CurrentID)
	}
	for _, item := range junctors {
		fmt.Fprintf(&mapping, "\x00junctor:%s=%s", item.PreviousID, item.CurrentID)
	}
	sum := sha256.Sum256([]byte("renumber-v1\x00" + string(encoded) + mapping.String()))
	return fmt.Sprintf("%x", sum[:]), nil
}
