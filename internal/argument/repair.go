// SPDX-FileCopyrightText: 2026 KeenWorks
// SPDX-License-Identifier: GPL-3.0-or-later

package argument

import (
	"fmt"
	"strings"
)

type JunctorSourceMode string

const (
	SourceAdd     JunctorSourceMode = "add"
	SourceRemove  JunctorSourceMode = "remove"
	SourceReplace JunctorSourceMode = "replace"
)

type RepairJunctorOptions struct {
	JunctorID string
	Mode      JunctorSourceMode
	SourceRef string
	FromRef   string
	ToRef     string
}

type RepairJunctorResult struct {
	Previous      Junctor
	Current       Junctor
	SourceAdded   string
	SourceRemoved string
}

type RemoveJunctorResult struct {
	Previous Junctor
}

func RepairJunctor(doc *Document, options RepairJunctorOptions) (*Document, RepairJunctorResult, error) {
	if doc == nil {
		return nil, RepairJunctorResult{}, mutationError("document_nil", "document is nil", "")
	}
	next := doc.Clone()
	junctor, ok := next.Junctor(strings.TrimSpace(options.JunctorID))
	if !ok {
		return nil, RepairJunctorResult{}, mutationError("junctor_not_found", fmt.Sprintf("junctor %q not found", options.JunctorID), options.JunctorID)
	}
	if junctor.Connector != ConnectorAND {
		return nil, RepairJunctorResult{}, mutationError("junctor_not_editable", fmt.Sprintf("focused repair edits only AND junctors; %s uses %s", junctor.ID, junctor.Connector), junctor.ID)
	}
	previous := cloneJunctor(*junctor)
	result := RepairJunctorResult{Previous: previous}
	switch options.Mode {
	case SourceAdd:
		source, err := resolveRepairSource(next, options.SourceRef)
		if err != nil {
			return nil, RepairJunctorResult{}, err
		}
		for _, id := range junctor.Sources {
			if id == source.ID {
				return nil, RepairJunctorResult{}, mutationError("junctor_source_duplicate", fmt.Sprintf("statement %s is already a source of junctor %s", source.ID, junctor.ID), source.ID)
			}
		}
		junctor.Sources = append(junctor.Sources, source.ID)
		result.SourceAdded = source.ID
	case SourceRemove:
		source, err := resolveRepairSource(next, options.SourceRef)
		if err != nil {
			return nil, RepairJunctorResult{}, err
		}
		index := sourceIndex(junctor.Sources, source.ID)
		if index < 0 {
			return nil, RepairJunctorResult{}, mutationError("junctor_source_not_found", fmt.Sprintf("statement %s is not a source of junctor %s", source.ID, junctor.ID), source.ID)
		}
		if len(junctor.Sources)-1 < 2 {
			return nil, RepairJunctorResult{}, mutationError("junctor_sources_too_few", "junctor must have at least two sources", junctor.ID)
		}
		junctor.Sources = append(junctor.Sources[:index], junctor.Sources[index+1:]...)
		result.SourceRemoved = source.ID
	case SourceReplace:
		from, err := resolveRepairSource(next, options.FromRef)
		if err != nil {
			return nil, RepairJunctorResult{}, err
		}
		to, err := resolveRepairSource(next, options.ToRef)
		if err != nil {
			return nil, RepairJunctorResult{}, err
		}
		index := sourceIndex(junctor.Sources, from.ID)
		if index < 0 {
			return nil, RepairJunctorResult{}, mutationError("junctor_source_not_found", fmt.Sprintf("statement %s is not a source of junctor %s", from.ID, junctor.ID), from.ID)
		}
		if from.ID == to.ID {
			return nil, RepairJunctorResult{}, mutationError("source_replacement_same_statement", fmt.Sprintf("replacement source for junctor %s must differ from %s", junctor.ID, from.ID), from.ID)
		}
		for _, id := range junctor.Sources {
			if id == to.ID {
				return nil, RepairJunctorResult{}, mutationError("junctor_source_duplicate", fmt.Sprintf("statement %s is already a source of junctor %s", to.ID, junctor.ID), to.ID)
			}
		}
		junctor.Sources[index] = to.ID
		result.SourceRemoved, result.SourceAdded = from.ID, to.ID
	default:
		return nil, RepairJunctorResult{}, mutationError("junctor_source_mode_invalid", fmt.Sprintf("invalid junctor source mode %q", options.Mode), string(options.Mode))
	}
	result.Current = cloneJunctor(*junctor)
	return next, result, nil
}

func RemoveJunctor(doc *Document, reference string) (*Document, RemoveJunctorResult, error) {
	if doc == nil {
		return nil, RemoveJunctorResult{}, mutationError("document_nil", "document is nil", "")
	}
	next := doc.Clone()
	if err := EnsureNextIDs(next); err != nil {
		return nil, RemoveJunctorResult{}, err
	}
	junctor, ok := next.Junctor(strings.TrimSpace(reference))
	if !ok {
		return nil, RemoveJunctorResult{}, mutationError("junctor_not_found", fmt.Sprintf("junctor %q not found", reference), reference)
	}
	for _, defeat := range next.Defeats {
		if defeat.Scope == DefeatInference && defeat.JunctorID == junctor.ID {
			return nil, RemoveJunctorResult{}, mutationError("junctor_has_undercuts", fmt.Sprintf("junctor %s is targeted by undercut %s; remove the counterpoint first", junctor.ID, defeat.From), junctor.ID)
		}
	}
	previous := cloneJunctor(*junctor)
	junctors := make([]Junctor, 0, len(next.Junctors)-1)
	for _, candidate := range next.Junctors {
		if candidate.ID != junctor.ID {
			junctors = append(junctors, candidate)
		}
	}
	next.Junctors = junctors
	return next, RemoveJunctorResult{Previous: previous}, nil
}

func resolveRepairSource(doc *Document, reference string) (*Statement, error) {
	statement, ok := doc.Statement(strings.TrimSpace(reference))
	if !ok {
		return nil, mutationError("source_not_found", fmt.Sprintf("source statement %q not found", reference), reference)
	}
	return statement, nil
}

func sourceIndex(sources []string, id string) int {
	for index, source := range sources {
		if source == id {
			return index
		}
	}
	return -1
}

func cloneJunctor(junctor Junctor) Junctor {
	junctor.Sources = append([]string(nil), junctor.Sources...)
	return junctor
}
