package evaluation

import (
	"fmt"
	"sort"

	"github.com/tunesmith/cludia/internal/argument"
)

const SchemaVersion = 1

type Mode string

const ModeGrounded Mode = "grounded"

type TruthSource string

const (
	TruthAsserted   TruthSource = "asserted"
	TruthDerived    TruthSource = "derived"
	TruthUnassigned TruthSource = "unassigned"
)

type Acceptance string

const (
	AcceptanceIn        Acceptance = "in"
	AcceptanceOut       Acceptance = "out"
	AcceptanceUndecided Acceptance = "undecided"
)

type StatementResult struct {
	ID             string         `json:"id"`
	StoredTruth    argument.Truth `json:"stored_truth"`
	EffectiveTruth argument.Truth `json:"effective_truth"`
	TruthSource    TruthSource    `json:"truth_source"`
	Acceptance     Acceptance     `json:"acceptance,omitempty"`
}

type JunctorResult struct {
	ID             string         `json:"id"`
	EffectiveTruth argument.Truth `json:"effective_truth"`
}

type InferenceEdge struct {
	JunctorID string `json:"junctor_id"`
	AtTarget  string `json:"at_target"`
}

type Result struct {
	SchemaVersion            int               `json:"schema_version"`
	Mode                     Mode              `json:"mode"`
	Statements               []StatementResult `json:"statements"`
	Junctors                 []JunctorResult   `json:"junctors"`
	AcceptedPremiseDefeats   []argument.Defeat `json:"accepted_premise_defeats"`
	AcceptedInferenceDefeats []argument.Defeat `json:"accepted_inference_defeats"`
	DisabledInferenceEdges   []InferenceEdge   `json:"disabled_inference_edges"`
	statementByID            map[string]StatementResult
	junctorByID              map[string]JunctorResult
	baseStatementTruthByID   map[string]argument.Truth
}

func (r Result) Statement(id string) (StatementResult, bool) {
	value, ok := r.statementByID[id]
	return value, ok
}

func (r Result) Junctor(id string) (JunctorResult, bool) {
	value, ok := r.junctorByID[id]
	return value, ok
}

// TruthChangedByDefeat reports whether the grounded defeat overlay changes a
// statement's truth from its support-propagated value. Because both values are
// propagated through the complete support graph, this includes material
// counterpoint effects anywhere upstream, not just defeats attached directly
// to the statement.
func (r Result) TruthChangedByDefeat(id string) bool {
	base, baseOK := r.baseStatementTruthByID[id]
	effective, effectiveOK := r.statementByID[id]
	return baseOK && effectiveOK && base != effective.EffectiveTruth
}

type Error struct {
	Code    string
	Message string
	Element string
}

func (e *Error) Error() string { return e.Message }

func Evaluate(doc *argument.Document) (Result, error) {
	statements, junctors, err := validateReferences(doc)
	if err != nil {
		return Result{}, err
	}
	baseStatements, _, err := propagate(doc, statements, junctors, nil, nil)
	if err != nil {
		return Result{}, err
	}
	acceptance := groundedAcceptance(doc, baseStatements)
	undermined := make(map[string]bool)
	disabled := make(map[InferenceEdge]bool)
	acceptedPremise := make([]argument.Defeat, 0)
	acceptedInference := make([]argument.Defeat, 0)
	for _, defeat := range doc.Defeats {
		if acceptance[defeat.From] != AcceptanceIn {
			continue
		}
		switch defeat.Scope {
		case argument.DefeatPremise:
			undermined[defeat.To] = true
			acceptedPremise = append(acceptedPremise, defeat)
		case argument.DefeatInference:
			edge := InferenceEdge{JunctorID: defeat.JunctorID, AtTarget: defeat.AtTarget}
			disabled[edge] = true
			acceptedInference = append(acceptedInference, defeat)
		}
	}
	effectiveStatements, effectiveJunctors, err := propagate(doc, statements, junctors, undermined, disabled)
	if err != nil {
		return Result{}, err
	}

	result := Result{
		SchemaVersion: SchemaVersion, Mode: ModeGrounded,
		Statements: []StatementResult{}, Junctors: []JunctorResult{},
		AcceptedPremiseDefeats: acceptedPremise, AcceptedInferenceDefeats: acceptedInference,
		DisabledInferenceEdges: []InferenceEdge{},
		statementByID:          make(map[string]StatementResult, len(doc.Statements)),
		junctorByID:            make(map[string]JunctorResult, len(doc.Junctors)),
		baseStatementTruthByID: baseStatements,
	}
	for edge := range disabled {
		result.DisabledInferenceEdges = append(result.DisabledInferenceEdges, edge)
	}
	sort.Slice(result.DisabledInferenceEdges, func(i, j int) bool {
		if result.DisabledInferenceEdges[i].JunctorID != result.DisabledInferenceEdges[j].JunctorID {
			return result.DisabledInferenceEdges[i].JunctorID < result.DisabledInferenceEdges[j].JunctorID
		}
		return result.DisabledInferenceEdges[i].AtTarget < result.DisabledInferenceEdges[j].AtTarget
	})
	incoming := incomingSupport(doc)
	for _, statement := range doc.Statements {
		source := TruthDerived
		if incoming[statement.ID] == 0 {
			if statement.Role == argument.RolePremise || statement.Role == argument.RoleCounterpoint {
				source = TruthAsserted
			} else {
				source = TruthUnassigned
			}
		}
		value := StatementResult{
			ID: statement.ID, StoredTruth: statement.Truth,
			EffectiveTruth: effectiveStatements[statement.ID], TruthSource: source,
		}
		if statement.Role == argument.RoleCounterpoint {
			value.Acceptance = acceptance[statement.ID]
		}
		result.Statements = append(result.Statements, value)
		result.statementByID[value.ID] = value
	}
	for _, junctor := range doc.Junctors {
		value := JunctorResult{ID: junctor.ID, EffectiveTruth: effectiveJunctors[junctor.ID]}
		result.Junctors = append(result.Junctors, value)
		result.junctorByID[value.ID] = value
	}
	return result, nil
}

func And(values []argument.Truth) argument.Truth {
	for _, value := range values {
		if value == argument.TruthFalse {
			return argument.TruthFalse
		}
	}
	for _, value := range values {
		if value == argument.TruthUnknown {
			return argument.TruthUnknown
		}
	}
	return argument.TruthTrue
}

func Or(values []argument.Truth) argument.Truth {
	for _, value := range values {
		if value == argument.TruthTrue {
			return argument.TruthTrue
		}
	}
	for _, value := range values {
		if value == argument.TruthUnknown {
			return argument.TruthUnknown
		}
	}
	return argument.TruthFalse
}

func propagate(
	doc *argument.Document,
	statements map[string]argument.Statement,
	junctors map[string]argument.Junctor,
	undermined map[string]bool,
	disabled map[InferenceEdge]bool,
) (map[string]argument.Truth, map[string]argument.Truth, error) {
	statementTruths := make(map[string]argument.Truth, len(statements))
	junctorTruths := make(map[string]argument.Truth, len(junctors))
	incomingJunctors := make(map[string][]string, len(statements))
	incomingDirect := make(map[string][]string, len(statements))
	indegree := make(map[string]int, len(statements)+len(junctors))
	outgoing := make(map[string][]string, len(statements)+len(junctors))
	statementNode := func(id string) string { return "s:" + id }
	junctorNode := func(id string) string { return "j:" + id }
	for _, statement := range doc.Statements {
		indegree[statementNode(statement.ID)] = 0
	}
	for _, junctor := range doc.Junctors {
		jnode := junctorNode(junctor.ID)
		indegree[jnode] = 0
		for _, source := range junctor.Sources {
			snode := statementNode(source)
			outgoing[snode] = append(outgoing[snode], jnode)
			indegree[jnode]++
		}
		tnode := statementNode(junctor.Target)
		outgoing[jnode] = append(outgoing[jnode], tnode)
		indegree[tnode]++
		incomingJunctors[junctor.Target] = append(incomingJunctors[junctor.Target], junctor.ID)
	}
	for _, support := range doc.DirectSupports {
		snode, tnode := statementNode(support.Source), statementNode(support.Target)
		outgoing[snode] = append(outgoing[snode], tnode)
		indegree[tnode]++
		incomingDirect[support.Target] = append(incomingDirect[support.Target], support.Source)
	}
	queue := make([]string, 0)
	for _, statement := range doc.Statements {
		node := statementNode(statement.ID)
		if indegree[node] == 0 {
			queue = append(queue, node)
		}
	}
	for _, junctor := range doc.Junctors {
		node := junctorNode(junctor.ID)
		if indegree[node] == 0 {
			queue = append(queue, node)
		}
	}
	processed := 0
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		processed++
		switch node[:2] {
		case "s:":
			id := node[2:]
			statement := statements[id]
			if undermined[id] {
				statementTruths[id] = argument.TruthFalse
			} else if len(incomingJunctors[id])+len(incomingDirect[id]) == 0 {
				if statement.Role == argument.RolePremise || statement.Role == argument.RoleCounterpoint {
					statementTruths[id] = statement.Truth
				} else {
					statementTruths[id] = argument.TruthUnknown
				}
			} else {
				inputs := make([]argument.Truth, 0, len(incomingJunctors[id])+len(incomingDirect[id]))
				for _, junctorID := range incomingJunctors[id] {
					if !disabled[InferenceEdge{JunctorID: junctorID, AtTarget: id}] {
						inputs = append(inputs, junctorTruths[junctorID])
					}
				}
				for _, source := range incomingDirect[id] {
					inputs = append(inputs, statementTruths[source])
				}
				if len(inputs) == 0 {
					statementTruths[id] = argument.TruthFalse
				} else {
					statementTruths[id] = Or(inputs)
				}
			}
		case "j:":
			id := node[2:]
			junctor := junctors[id]
			inputs := make([]argument.Truth, 0, len(junctor.Sources))
			for _, source := range junctor.Sources {
				inputs = append(inputs, statementTruths[source])
			}
			if len(inputs) == 0 {
				junctorTruths[id] = argument.TruthUnknown
			} else if junctor.Connector == argument.ConnectorAND {
				junctorTruths[id] = And(inputs)
			} else {
				junctorTruths[id] = Or(inputs)
			}
		}
		for _, target := range outgoing[node] {
			indegree[target]--
			if indegree[target] == 0 {
				queue = append(queue, target)
			}
		}
	}
	if processed != len(indegree) {
		return nil, nil, &Error{Code: "evaluation_support_cycle", Message: "support graph contains a cycle", Element: doc.ID}
	}
	return statementTruths, junctorTruths, nil
}

func groundedAcceptance(doc *argument.Document, baseTruths map[string]argument.Truth) map[string]Acceptance {
	active := make(map[string]bool)
	labels := make(map[string]Acceptance)
	attackers := make(map[string][]string)
	for _, statement := range doc.Statements {
		if statement.Role != argument.RoleCounterpoint {
			continue
		}
		active[statement.ID] = baseTruths[statement.ID] == argument.TruthTrue
		if active[statement.ID] {
			labels[statement.ID] = AcceptanceUndecided
		} else {
			labels[statement.ID] = AcceptanceOut
		}
	}
	for _, defeat := range doc.Defeats {
		if defeat.Scope == argument.DefeatCounterpoint && active[defeat.From] && active[defeat.To] {
			attackers[defeat.To] = append(attackers[defeat.To], defeat.From)
		}
	}
	for changed := true; changed; {
		changed = false
		next := make(map[string]Acceptance, len(labels))
		for id, label := range labels {
			next[id] = label
		}
		for _, statement := range doc.Statements {
			id := statement.ID
			if statement.Role != argument.RoleCounterpoint || !active[id] {
				continue
			}
			allOut, anyIn := true, false
			for _, attacker := range attackers[id] {
				allOut = allOut && labels[attacker] == AcceptanceOut
				anyIn = anyIn || labels[attacker] == AcceptanceIn
			}
			value := AcceptanceUndecided
			if allOut {
				value = AcceptanceIn
			} else if anyIn {
				value = AcceptanceOut
			}
			if value != labels[id] {
				next[id] = value
				changed = true
			}
		}
		labels = next
	}
	return labels
}

func incomingSupport(doc *argument.Document) map[string]int {
	result := make(map[string]int, len(doc.Statements))
	for _, junctor := range doc.Junctors {
		result[junctor.Target]++
	}
	for _, support := range doc.DirectSupports {
		result[support.Target]++
	}
	return result
}

func validateReferences(doc *argument.Document) (map[string]argument.Statement, map[string]argument.Junctor, error) {
	if doc == nil {
		return nil, nil, &Error{Code: "evaluation_document_nil", Message: "document is nil"}
	}
	statements := make(map[string]argument.Statement, len(doc.Statements))
	for _, statement := range doc.Statements {
		if _, exists := statements[statement.ID]; exists {
			return nil, nil, &Error{Code: "evaluation_statement_duplicate", Message: fmt.Sprintf("duplicate statement id %q", statement.ID), Element: statement.ID}
		}
		statements[statement.ID] = statement
	}
	junctors := make(map[string]argument.Junctor, len(doc.Junctors))
	for _, junctor := range doc.Junctors {
		if _, exists := junctors[junctor.ID]; exists {
			return nil, nil, &Error{Code: "evaluation_junctor_duplicate", Message: fmt.Sprintf("duplicate junctor id %q", junctor.ID), Element: junctor.ID}
		}
		if _, exists := statements[junctor.Target]; !exists {
			return nil, nil, &Error{Code: "evaluation_target_unknown", Message: fmt.Sprintf("junctor target %q does not exist", junctor.Target), Element: junctor.ID}
		}
		for _, source := range junctor.Sources {
			if _, exists := statements[source]; !exists {
				return nil, nil, &Error{Code: "evaluation_source_unknown", Message: fmt.Sprintf("junctor source %q does not exist", source), Element: junctor.ID}
			}
		}
		junctors[junctor.ID] = junctor
	}
	for _, support := range doc.DirectSupports {
		if _, exists := statements[support.Source]; !exists {
			return nil, nil, &Error{Code: "evaluation_source_unknown", Message: fmt.Sprintf("direct support source %q does not exist", support.Source), Element: support.Source}
		}
		if _, exists := statements[support.Target]; !exists {
			return nil, nil, &Error{Code: "evaluation_target_unknown", Message: fmt.Sprintf("direct support target %q does not exist", support.Target), Element: support.Target}
		}
	}
	return statements, junctors, nil
}
