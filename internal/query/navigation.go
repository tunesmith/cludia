package query

import (
	"fmt"
	"sort"

	"github.com/tunesmith/cludia/internal/argument"
	"github.com/tunesmith/cludia/internal/evaluation"
)

// TopItem is one non-counterpoint statement with no outgoing support.
type TopItem struct {
	Statement      argument.Statement     `json:"statement"`
	Depth          int                    `json:"depth"`
	Challenged     bool                   `json:"challenged"`
	EffectiveTruth argument.Truth         `json:"effective_truth"`
	TruthSource    evaluation.TruthSource `json:"truth_source"`
	Acceptance     evaluation.Acceptance  `json:"acceptance,omitempty"`
}

// LedgerRow is one statement in a stable topological support derivation.
type LedgerRow struct {
	Statement      argument.Statement     `json:"statement"`
	Depth          int                    `json:"depth"`
	Challenged     bool                   `json:"challenged"`
	Derivations    []Support              `json:"derivations"`
	EffectiveTruth argument.Truth         `json:"effective_truth"`
	TruthSource    evaluation.TruthSource `json:"truth_source"`
	Acceptance     evaluation.Acceptance  `json:"acceptance,omitempty"`
}

// LedgerInference describes one explicitly selected incoming junctor and how
// it relates to the root's complete grounded truth.
type LedgerInference struct {
	Junctor                    argument.Junctor `json:"junctor"`
	EffectiveTruth             argument.Truth   `json:"effective_truth"`
	Undercut                   bool             `json:"undercut"`
	OtherJustificationsOmitted bool             `json:"other_justifications_omitted"`
	OtherRoutesAffectTruth     bool             `json:"other_routes_affect_truth"`
}

type LedgerInferenceError struct {
	Code    string
	Message string
	Element string
}

func (e *LedgerInferenceError) Error() string { return e.Message }

// Top returns non-counterpoint support sinks in document order.
func Top(doc *argument.Document) []TopItem {
	items := topItems(doc)
	evaluated, err := evaluation.Evaluate(doc)
	if err == nil {
		applyTopEvaluation(items, evaluated)
	}
	return items
}

func topItems(doc *argument.Document) []TopItem {
	if doc == nil {
		return []TopItem{}
	}
	hasOutgoingSupport := make(map[string]bool, len(doc.Statements))
	for _, junctor := range doc.Junctors {
		for _, source := range junctor.Sources {
			hasOutgoingSupport[source] = true
		}
	}
	for _, support := range doc.DirectSupports {
		hasOutgoingSupport[support.Source] = true
	}
	depths := supportDepths(doc)
	items := make([]TopItem, 0)
	for _, statement := range doc.Statements {
		if statement.Role == argument.RoleCounterpoint || hasOutgoingSupport[statement.ID] {
			continue
		}
		items = append(items, TopItem{
			Statement: statement, Depth: depths[statement.ID],
		})
	}
	return items
}

func TopEvaluated(doc *argument.Document, evaluated evaluation.Result) []TopItem {
	items := topItems(doc)
	applyTopEvaluation(items, evaluated)
	return items
}

func applyTopEvaluation(items []TopItem, evaluated evaluation.Result) {
	for index := range items {
		value, _ := evaluated.Statement(items[index].Statement.ID)
		items[index].Challenged = evaluated.TruthChangedByDefeat(items[index].Statement.ID)
		items[index].EffectiveTruth = value.EffectiveTruth
		items[index].TruthSource = value.TruthSource
		items[index].Acceptance = value.Acceptance
	}
}

// Ledger returns the support-only upstream closure of a selected statement in
// stable, proof-local topological order. Defeats contribute challenge state but
// not rows.
func Ledger(doc *argument.Document, reference string) (string, []LedgerRow, error) {
	root, rows, err := ledgerRows(doc, reference, "")
	if err != nil {
		return "", nil, err
	}
	evaluated, err := evaluation.Evaluate(doc)
	if err != nil {
		return "", nil, err
	}
	applyLedgerEvaluation(rows, evaluated)
	return root, rows, nil
}

func ledgerRows(doc *argument.Document, reference, selectedJunctorID string) (string, []LedgerRow, error) {
	if doc == nil {
		return "", nil, fmt.Errorf("document is nil")
	}
	root, ok := doc.Statement(reference)
	if !ok {
		return "", nil, fmt.Errorf("ledger root statement %q not found", reference)
	}
	if root.Role == argument.RoleCounterpoint {
		return "", nil, fmt.Errorf("ledger root statement %s is a counterpoint", root.ID)
	}
	if selectedJunctorID != "" {
		junctor, ok := doc.Junctor(selectedJunctorID)
		if !ok {
			return "", nil, &LedgerInferenceError{
				Code: "ledger_inference_not_found", Message: fmt.Sprintf("selected inference %q was not found", selectedJunctorID), Element: selectedJunctorID,
			}
		}
		if junctor.Target != root.ID {
			return "", nil, &LedgerInferenceError{
				Code:    "ledger_inference_target_mismatch",
				Message: fmt.Sprintf("selected inference %s targets %s rather than ledger root %s", junctor.ID, junctor.Target, root.ID),
				Element: junctor.ID,
			}
		}
	}

	included := map[string]bool{root.ID: true}
	for changed := true; changed; {
		changed = false
		for _, junctor := range doc.Junctors {
			if !included[junctor.Target] {
				continue
			}
			if selectedJunctorID != "" && junctor.Target == root.ID && junctor.ID != selectedJunctorID {
				continue
			}
			for _, source := range junctor.Sources {
				if _, exists := doc.Statement(source); !exists || included[source] {
					continue
				}
				included[source] = true
				changed = true
			}
		}
		for _, support := range doc.DirectSupports {
			if !included[support.Target] || included[support.Source] {
				continue
			}
			if selectedJunctorID != "" && support.Target == root.ID {
				continue
			}
			if _, exists := doc.Statement(support.Source); !exists {
				continue
			}
			included[support.Source] = true
			changed = true
		}
	}

	index := make(map[string]int, len(doc.Statements))
	outdegree := make(map[string]int, len(included))
	dependencies := make(map[string]map[string]bool, len(included))
	for i, statement := range doc.Statements {
		index[statement.ID] = i
		if included[statement.ID] {
			outdegree[statement.ID] = 0
			dependencies[statement.ID] = make(map[string]bool)
		}
	}
	addDependency := func(source, target string) {
		if !included[source] || !included[target] || dependencies[target][source] {
			return
		}
		dependencies[target][source] = true
		outdegree[source]++
	}
	for _, junctor := range doc.Junctors {
		if !included[junctor.Target] {
			continue
		}
		if selectedJunctorID != "" && junctor.Target == root.ID && junctor.ID != selectedJunctorID {
			continue
		}
		for _, source := range junctor.Sources {
			addDependency(source, junctor.Target)
		}
	}
	for _, support := range doc.DirectSupports {
		if selectedJunctorID != "" && support.Target == root.ID {
			continue
		}
		addDependency(support.Source, support.Target)
	}

	depths := ledgerSupportDepths(doc, root.ID, selectedJunctorID)
	ready := make([]string, 0)
	for _, statement := range doc.Statements {
		if included[statement.ID] && outdegree[statement.ID] == 0 {
			ready = append(ready, statement.ID)
		}
	}
	sortIDsForReverseLedger(ready, index, depths)
	reverseOrdered := make([]string, 0, len(included))
	for len(ready) > 0 {
		id := ready[0]
		ready = ready[1:]
		reverseOrdered = append(reverseOrdered, id)
		for source := range dependencies[id] {
			outdegree[source]--
			if outdegree[source] == 0 {
				ready = append(ready, source)
				sortIDsForReverseLedger(ready, index, depths)
			}
		}
	}
	if len(reverseOrdered) != len(included) {
		return "", nil, fmt.Errorf("ledger support graph contains a cycle")
	}
	ordered := make([]string, len(reverseOrdered))
	for i, id := range reverseOrdered {
		ordered[len(reverseOrdered)-1-i] = id
	}

	rows := make([]LedgerRow, 0, len(ordered))
	for _, id := range ordered {
		statement, _ := doc.Statement(id)
		rows = append(rows, LedgerRow{
			Statement: *statement, Depth: depths[id],
			Derivations: incomingLedgerSupports(doc, id, included, root.ID, selectedJunctorID),
		})
	}
	return root.ID, rows, nil
}

func LedgerEvaluated(doc *argument.Document, reference string, evaluated evaluation.Result) (string, []LedgerRow, error) {
	root, rows, err := ledgerRows(doc, reference, "")
	if err != nil {
		return "", nil, err
	}
	applyLedgerEvaluation(rows, evaluated)
	return root, rows, nil
}

// LedgerInferenceEvaluated returns the complete upstream closure of one
// explicitly selected junctor at the ledger root. Deeper statements retain all
// of their authored justifications.
func LedgerInferenceEvaluated(doc *argument.Document, reference, junctorID string, evaluated evaluation.Result) (string, []LedgerRow, LedgerInference, error) {
	root, rows, err := ledgerRows(doc, reference, junctorID)
	if err != nil {
		return "", nil, LedgerInference{}, err
	}
	applyLedgerEvaluation(rows, evaluated)
	junctor, _ := doc.Junctor(junctorID)
	junctorValue, _ := evaluated.Junctor(junctor.ID)
	undercut := false
	for _, edge := range evaluated.DisabledInferenceEdges {
		if edge.JunctorID == junctor.ID && edge.AtTarget == root {
			undercut = true
			break
		}
	}
	omitted := false
	for _, candidate := range doc.Junctors {
		if candidate.Target == root && candidate.ID != junctor.ID {
			omitted = true
			break
		}
	}
	if !omitted {
		for _, support := range doc.DirectSupports {
			if support.Target == root {
				omitted = true
				break
			}
		}
	}
	selectedOnlyTruth := junctorValue.EffectiveTruth
	if undercut {
		selectedOnlyTruth = argument.TruthFalse
	}
	rootValue, _ := evaluated.Statement(root)
	selection := LedgerInference{
		Junctor: *junctor, EffectiveTruth: junctorValue.EffectiveTruth,
		Undercut: undercut, OtherJustificationsOmitted: omitted,
		OtherRoutesAffectTruth: omitted && selectedOnlyTruth != rootValue.EffectiveTruth,
	}
	selection.Junctor.Sources = append([]string(nil), junctor.Sources...)
	return root, rows, selection, nil
}

func applyLedgerEvaluation(rows []LedgerRow, evaluated evaluation.Result) {
	for index := range rows {
		value, _ := evaluated.Statement(rows[index].Statement.ID)
		rows[index].Challenged = evaluated.TruthChangedByDefeat(rows[index].Statement.ID)
		rows[index].EffectiveTruth = value.EffectiveTruth
		rows[index].TruthSource = value.TruthSource
		rows[index].Acceptance = value.Acceptance
	}
}

// StatementDirectlyChallenged reports structural defeat attached to a
// statement or one of its incoming junctors. It intentionally does not apply
// grounded acceptance and is for local inspection rather than the compact !
// marker shown by evaluated overview surfaces.
func StatementDirectlyChallenged(doc *argument.Document, id string) bool {
	if doc == nil {
		return false
	}
	incomingJunctors := make(map[string]bool)
	for _, junctor := range doc.Junctors {
		if junctor.Target == id {
			incomingJunctors[junctor.ID] = true
		}
	}
	for _, defeat := range doc.Defeats {
		if defeat.To == id || defeat.AtTarget == id || (defeat.Scope == argument.DefeatInference && incomingJunctors[defeat.JunctorID]) {
			return true
		}
	}
	return false
}

func incomingLedgerSupports(doc *argument.Document, target string, included map[string]bool, root, selectedJunctorID string) []Support {
	result := make([]Support, 0)
	for _, junctor := range doc.Junctors {
		if junctor.Target != target {
			continue
		}
		if selectedJunctorID != "" && target == root && junctor.ID != selectedJunctorID {
			continue
		}
		allIncluded := true
		for _, source := range junctor.Sources {
			if !included[source] {
				allIncluded = false
				break
			}
		}
		if allIncluded {
			result = append(result, junctorSupport(junctor))
		}
	}
	for _, support := range doc.DirectSupports {
		if selectedJunctorID != "" && target == root {
			continue
		}
		if support.Target == target && included[support.Source] {
			result = append(result, directSupport(support))
		}
	}
	return result
}

func ledgerSupportDepths(doc *argument.Document, root, selectedJunctorID string) map[string]int {
	incoming := make(map[string][][]string, len(doc.Statements))
	for _, statement := range doc.Statements {
		incoming[statement.ID] = [][]string{}
	}
	for _, junctor := range doc.Junctors {
		if selectedJunctorID != "" && junctor.Target == root && junctor.ID != selectedJunctorID {
			continue
		}
		incoming[junctor.Target] = append(incoming[junctor.Target], append([]string(nil), junctor.Sources...))
	}
	for _, support := range doc.DirectSupports {
		if selectedJunctorID != "" && support.Target == root {
			continue
		}
		incoming[support.Target] = append(incoming[support.Target], []string{support.Source})
	}
	return calculateSupportDepths(doc, incoming)
}

func supportDepths(doc *argument.Document) map[string]int {
	incoming := make(map[string][][]string, len(doc.Statements))
	for _, statement := range doc.Statements {
		incoming[statement.ID] = [][]string{}
	}
	for _, junctor := range doc.Junctors {
		incoming[junctor.Target] = append(incoming[junctor.Target], append([]string(nil), junctor.Sources...))
	}
	for _, support := range doc.DirectSupports {
		incoming[support.Target] = append(incoming[support.Target], []string{support.Source})
	}
	return calculateSupportDepths(doc, incoming)
}

func calculateSupportDepths(doc *argument.Document, incoming map[string][][]string) map[string]int {
	memo := make(map[string]int, len(doc.Statements))
	visiting := make(map[string]bool, len(doc.Statements))
	var depth func(string) int
	depth = func(id string) int {
		if value, ok := memo[id]; ok {
			return value
		}
		if visiting[id] {
			return 0
		}
		visiting[id] = true
		best := 0
		for _, sources := range incoming[id] {
			for _, source := range sources {
				if candidate := depth(source) + 1; candidate > best {
					best = candidate
				}
			}
		}
		visiting[id] = false
		memo[id] = best
		return best
	}
	for _, statement := range doc.Statements {
		depth(statement.ID)
	}
	return memo
}

func sortIDsForReverseLedger(ids []string, index, depths map[string]int) {
	sort.SliceStable(ids, func(i, j int) bool {
		if depths[ids[i]] != depths[ids[j]] {
			return depths[ids[i]] < depths[ids[j]]
		}
		return index[ids[i]] > index[ids[j]]
	})
}
