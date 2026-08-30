package argument

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
)

type ReplacementOptions struct {
	OldRef               string
	ReplacementRef       string
	SourceJunctors       []string
	RemoveJustifications []string
	RetargetRoot         bool
	DeleteOld            bool
}

type ReplacementIncidents struct {
	SourceJunctors []string        `json:"source_junctors"`
	TargetJunctors []string        `json:"target_junctors"`
	DirectSupports []DirectSupport `json:"direct_supports"`
	Defeats        []Defeat        `json:"defeats"`
	RootMetadata   bool            `json:"root_metadata"`
}

type ReplacementSourceRetarget struct {
	PreviousJunctor Junctor `json:"previous_junctor"`
	Junctor         Junctor `json:"junctor"`
}

type ReplacementBlocker struct {
	Relation string `json:"relation"`
	ID       string `json:"id"`
	Message  string `json:"message"`
}

type ReplacementResult struct {
	OldStatement          Statement
	ReplacementStatement  Statement
	SourceRetargets       []ReplacementSourceRetarget
	JustificationsRemoved []Junctor
	AffectedDefeats       []Defeat
	IncidentsBefore       ReplacementIncidents
	IncidentsRemaining    ReplacementIncidents
	RootRetargeted        bool
	OldDeleted            bool
	Blockers              []ReplacementBlocker
	Applicable            bool
	PlanToken             string
}

func ReplaceStatement(doc *Document, options ReplacementOptions) (*Document, ReplacementResult, error) {
	if doc == nil {
		return nil, ReplacementResult{}, mutationError("document_nil", "document is nil", "")
	}
	old, ok := doc.Statement(strings.TrimSpace(options.OldRef))
	if !ok {
		return nil, ReplacementResult{}, mutationError("statement_not_found", fmt.Sprintf("old statement %q not found", options.OldRef), options.OldRef)
	}
	replacement, ok := doc.Statement(strings.TrimSpace(options.ReplacementRef))
	if !ok {
		return nil, ReplacementResult{}, mutationError("statement_not_found", fmt.Sprintf("replacement statement %q not found", options.ReplacementRef), options.ReplacementRef)
	}
	if old.ID == replacement.ID {
		return nil, ReplacementResult{}, mutationError("replacement_same_statement", "old and replacement statements must differ", old.ID)
	}
	if old.Role == RoleCounterpoint || replacement.Role == RoleCounterpoint {
		return nil, ReplacementResult{}, mutationError("replacement_counterpoint_unsupported", "focused material replacement does not replace counterpoint statements", old.ID)
	}
	if duplicate := duplicateReference(options.SourceJunctors); duplicate != "" {
		return nil, ReplacementResult{}, mutationError("replacement_selection_duplicate", fmt.Sprintf("retarget-source selection %q appears more than once", duplicate), duplicate)
	}
	if duplicate := duplicateReference(options.RemoveJustifications); duplicate != "" {
		return nil, ReplacementResult{}, mutationError("replacement_selection_duplicate", fmt.Sprintf("remove-justification selection %q appears more than once", duplicate), duplicate)
	}
	planToken, err := replacementToken(doc, old.ID, replacement.ID, options)
	if err != nil {
		return nil, ReplacementResult{}, err
	}

	next := doc.Clone()
	oldNext, _ := next.Statement(old.ID)
	replacementNext, _ := next.Statement(replacement.ID)
	result := ReplacementResult{
		OldStatement: *old, ReplacementStatement: *replacement,
		SourceRetargets: []ReplacementSourceRetarget{}, JustificationsRemoved: []Junctor{},
		AffectedDefeats: []Defeat{}, Blockers: []ReplacementBlocker{}, PlanToken: planToken,
	}
	result.IncidentsBefore = collectReplacementIncidents(next, oldNext)
	selectedJunctors := make(map[string]bool, len(options.SourceJunctors))
	for _, selected := range options.SourceJunctors {
		junctor, ok := next.Junctor(strings.TrimSpace(selected))
		if !ok {
			return nil, ReplacementResult{}, mutationError("junctor_not_found", fmt.Sprintf("retarget-source junctor %q not found", selected), selected)
		}
		if junctor.Connector != ConnectorAND {
			return nil, ReplacementResult{}, mutationError("junctor_not_editable", fmt.Sprintf("replacement source retargeting edits only AND junctors; %s uses %s", junctor.ID, junctor.Connector), junctor.ID)
		}
		index := sourceIndex(junctor.Sources, oldNext.ID)
		if index < 0 {
			return nil, ReplacementResult{}, mutationError("junctor_source_not_found", fmt.Sprintf("old statement %s is not a source of junctor %s", oldNext.ID, junctor.ID), junctor.ID)
		}
		if containsID(junctor.Sources, replacementNext.ID) {
			return nil, ReplacementResult{}, mutationError("junctor_source_duplicate", fmt.Sprintf("replacement statement %s is already a source of junctor %s", replacementNext.ID, junctor.ID), junctor.ID)
		}
		previous := cloneJunctor(*junctor)
		junctor.Sources[index] = replacementNext.ID
		result.SourceRetargets = append(result.SourceRetargets, ReplacementSourceRetarget{PreviousJunctor: previous, Junctor: cloneJunctor(*junctor)})
		selectedJunctors[junctor.ID] = true
	}

	removeSet := make(map[string]bool, len(options.RemoveJustifications))
	for _, selected := range options.RemoveJustifications {
		junctor, ok := next.Junctor(strings.TrimSpace(selected))
		if !ok {
			return nil, ReplacementResult{}, mutationError("junctor_not_found", fmt.Sprintf("remove-justification junctor %q not found", selected), selected)
		}
		if junctor.Target != oldNext.ID {
			return nil, ReplacementResult{}, mutationError("replacement_justification_target_mismatch", fmt.Sprintf("junctor %s targets %s rather than old statement %s", junctor.ID, junctor.Target, oldNext.ID), junctor.ID)
		}
		for _, defeat := range next.Defeats {
			if defeat.Scope == DefeatInference && defeat.JunctorID == junctor.ID {
				return nil, ReplacementResult{}, mutationError("junctor_has_undercuts", fmt.Sprintf("junctor %s is targeted by undercut %s; remove the counterpoint first", junctor.ID, defeat.From), junctor.ID)
			}
		}
		removeSet[junctor.ID] = true
		result.JustificationsRemoved = append(result.JustificationsRemoved, cloneJunctor(*junctor))
	}
	if len(removeSet) > 0 {
		junctors := make([]Junctor, 0, len(next.Junctors)-len(removeSet))
		for _, junctor := range next.Junctors {
			if !removeSet[junctor.ID] {
				junctors = append(junctors, junctor)
			}
		}
		next.Junctors = junctors
	}

	rootMatched := replacementRootMatches(next, oldNext)
	if options.RetargetRoot {
		if !rootMatched {
			return nil, ReplacementResult{}, mutationError("replacement_root_not_old", fmt.Sprintf("recognized root metadata does not reference old statement %s", oldNext.ID), oldNext.ID)
		}
		rootValue := replacementNext.ID
		if replacementNext.Slug != "" {
			rootValue = replacementNext.Slug
		}
		for index := range next.Metadata {
			metadata := &next.Metadata[index]
			if metadata.Key == "root" && (metadata.Value == oldNext.ID || (oldNext.Slug != "" && metadata.Value == oldNext.Slug)) {
				metadata.Value = rootValue
				result.RootRetargeted = true
			}
		}
	}
	for _, defeat := range next.Defeats {
		if defeat.Scope == DefeatInference && selectedJunctors[defeat.JunctorID] {
			result.AffectedDefeats = append(result.AffectedDefeats, defeat)
		}
	}
	result.IncidentsRemaining = collectReplacementIncidents(next, oldNext)
	if options.DeleteOld {
		result.Blockers = replacementDeletionBlockers(result.IncidentsRemaining)
	}
	if options.DeleteOld && len(result.Blockers) == 0 {
		statements := make([]Statement, 0, len(next.Statements)-1)
		for _, statement := range next.Statements {
			if statement.ID != oldNext.ID {
				statements = append(statements, statement)
			}
		}
		next.Statements = statements
		result.OldDeleted = true
	}
	if len(removeSet) > 0 || result.OldDeleted {
		if err := EnsureNextIDs(next); err != nil {
			return nil, ReplacementResult{}, err
		}
	}
	result.Applicable = len(result.Blockers) == 0
	if !result.Applicable {
		result.PlanToken = ""
	}
	return next, result, nil
}

func collectReplacementIncidents(doc *Document, statement *Statement) ReplacementIncidents {
	incidents := ReplacementIncidents{SourceJunctors: []string{}, TargetJunctors: []string{}, DirectSupports: []DirectSupport{}, Defeats: []Defeat{}}
	for _, junctor := range doc.Junctors {
		if containsID(junctor.Sources, statement.ID) {
			incidents.SourceJunctors = append(incidents.SourceJunctors, junctor.ID)
		}
		if junctor.Target == statement.ID {
			incidents.TargetJunctors = append(incidents.TargetJunctors, junctor.ID)
		}
	}
	for _, support := range doc.DirectSupports {
		if support.Source == statement.ID || support.Target == statement.ID {
			incidents.DirectSupports = append(incidents.DirectSupports, support)
		}
	}
	for _, defeat := range doc.Defeats {
		if defeat.From == statement.ID || defeat.To == statement.ID || defeat.AtTarget == statement.ID {
			incidents.Defeats = append(incidents.Defeats, defeat)
		}
	}
	incidents.RootMetadata = replacementRootMatches(doc, statement)
	return incidents
}

func replacementRootMatches(doc *Document, statement *Statement) bool {
	for _, metadata := range doc.Metadata {
		if metadata.Key == "root" && (metadata.Value == statement.ID || (statement.Slug != "" && metadata.Value == statement.Slug)) {
			return true
		}
	}
	return false
}

func replacementDeletionBlockers(incidents ReplacementIncidents) []ReplacementBlocker {
	blockers := make([]ReplacementBlocker, 0, len(incidents.SourceJunctors)+len(incidents.TargetJunctors)+len(incidents.DirectSupports)+len(incidents.Defeats)+1)
	for _, id := range incidents.SourceJunctors {
		blockers = append(blockers, ReplacementBlocker{Relation: "junctor_source", ID: id, Message: "old statement remains a source; select this junctor with --retarget-source or retain the old statement"})
	}
	for _, id := range incidents.TargetJunctors {
		blockers = append(blockers, ReplacementBlocker{Relation: "junctor_target", ID: id, Message: "old statement retains an incoming justification; select it with --remove-justification or retain the old statement"})
	}
	for _, support := range incidents.DirectSupports {
		blockers = append(blockers, ReplacementBlocker{Relation: "direct_support", ID: support.Source + "->" + support.Target, Message: "legacy direct support requires separate relation review"})
	}
	for _, defeat := range incidents.Defeats {
		blockers = append(blockers, ReplacementBlocker{Relation: "defeat", ID: defeat.From, Message: "attached counterpoint requires explicit review and removal before deleting the old statement"})
	}
	if incidents.RootMetadata {
		blockers = append(blockers, ReplacementBlocker{Relation: "metadata_root", ID: "root", Message: "recognized root metadata still references the old statement; select --retarget-root or retain the old statement"})
	}
	return blockers
}

func replacementToken(doc *Document, oldID, replacementID string, options ReplacementOptions) (string, error) {
	encoded, err := marshalDocumentState(doc)
	if err != nil {
		return "", err
	}
	sources := append([]string(nil), options.SourceJunctors...)
	removals := append([]string(nil), options.RemoveJustifications...)
	sort.Strings(sources)
	sort.Strings(removals)
	payload := fmt.Sprintf("%s\x00old=%s\x00new=%s\x00sources=%s\x00remove=%s\x00root=%t\x00delete=%t", encoded, oldID, replacementID, strings.Join(sources, ","), strings.Join(removals, ","), options.RetargetRoot, options.DeleteOld)
	sum := sha256.Sum256([]byte(payload))
	return fmt.Sprintf("%x", sum[:]), nil
}

func duplicateReference(values []string) string {
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if seen[value] {
			return value
		}
		seen[value] = true
	}
	return ""
}
