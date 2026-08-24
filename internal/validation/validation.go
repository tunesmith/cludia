package validation

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tunesmith/cludia/internal/argument"
	"github.com/tunesmith/cludia/internal/diagnostic"
)

type Profile string

const (
	ProfileWorkspace Profile = "workspace"
	ProfileConcludia Profile = "concludia"
)

type Result struct {
	Profile     Profile                 `json:"profile"`
	Diagnostics []diagnostic.Diagnostic `json:"diagnostics"`
}

func (r Result) OK() bool {
	return !diagnostic.HasErrors(r.Diagnostics)
}

func Validate(doc *argument.Document, profile Profile) Result {
	result := Result{Profile: profile, Diagnostics: []diagnostic.Diagnostic{}}
	if profile != ProfileWorkspace && profile != ProfileConcludia {
		result.error("profile_unknown", fmt.Sprintf("unknown validation profile %q", profile), "")
		return result
	}
	if doc == nil {
		result.error("document_nil", "document is nil", "")
		return result
	}
	if strings.TrimSpace(doc.ID) == "" {
		result.error("document_id_required", "document id is required", "")
	}
	if strings.TrimSpace(doc.Title) == "" {
		result.error("document_title_required", "document title is required", doc.ID)
	}
	if len(doc.Statements) == 0 {
		result.error("statements_required", "document must contain at least one statement", doc.ID)
		return result
	}

	statementByID := make(map[string]argument.Statement, len(doc.Statements))
	slugOwner := make(map[string]string, len(doc.Statements))
	allIDs := make(map[string]string, len(doc.Statements)+len(doc.Junctors))
	incomingSupport := make(map[string]int, len(doc.Statements))
	supportSources := make(map[string]bool, len(doc.Statements))
	directed := make(map[string][]string)
	undirected := make(map[string]map[string]bool, len(doc.Statements))

	for _, statement := range doc.Statements {
		undirected[statement.ID] = make(map[string]bool)
		if !argument.ValidID(statement.ID) {
			result.error("statement_id_invalid", fmt.Sprintf("invalid statement id %q", statement.ID), statement.ID)
		}
		if owner, exists := allIDs[statement.ID]; exists {
			result.error("id_duplicate", fmt.Sprintf("id %q is already used by %s", statement.ID, owner), statement.ID)
		} else {
			allIDs[statement.ID] = "statement"
		}
		if _, exists := statementByID[statement.ID]; exists {
			result.error("statement_id_duplicate", fmt.Sprintf("duplicate statement id %q", statement.ID), statement.ID)
		} else {
			statementByID[statement.ID] = statement
		}
		if statement.Slug != "" {
			if !argument.ValidSlug(statement.Slug) {
				result.error("statement_slug_invalid", fmt.Sprintf("invalid statement slug %q", statement.Slug), statement.ID)
			}
			if owner, exists := slugOwner[statement.Slug]; exists {
				result.error("statement_slug_duplicate", fmt.Sprintf("slug %q is already used by %s", statement.Slug, owner), statement.ID)
			} else {
				slugOwner[statement.Slug] = statement.ID
			}
		}
		switch statement.Role {
		case argument.RolePremise, argument.RoleLemma, argument.RoleConclusion, argument.RoleCounterpoint:
		default:
			result.error("statement_role_invalid", fmt.Sprintf("invalid role %q", statement.Role), statement.ID)
		}
		switch statement.Kind {
		case argument.KindFact, argument.KindValue:
		default:
			result.error("statement_kind_invalid", fmt.Sprintf("invalid kind %q", statement.Kind), statement.ID)
		}
		switch statement.Truth {
		case argument.TruthTrue, argument.TruthFalse, argument.TruthUnknown:
		default:
			result.error("statement_truth_invalid", fmt.Sprintf("invalid truth %q", statement.Truth), statement.ID)
		}
		if statement.Role != argument.RolePremise && statement.Role != argument.RoleCounterpoint && statement.Truth != argument.TruthUnknown {
			result.error("statement_truth_role_invalid", "only premises and counterpoints may have non-unknown truth", statement.ID)
		}
		if strings.TrimSpace(statement.Text) == "" {
			result.error("statement_text_required", "statement text is required", statement.ID)
		}
	}

	junctorByID := make(map[string]argument.Junctor, len(doc.Junctors))
	for _, junctor := range doc.Junctors {
		if !argument.ValidID(junctor.ID) {
			result.error("junctor_id_invalid", fmt.Sprintf("invalid junctor id %q", junctor.ID), junctor.ID)
		}
		if owner, exists := allIDs[junctor.ID]; exists {
			result.error("id_duplicate", fmt.Sprintf("id %q is already used by %s", junctor.ID, owner), junctor.ID)
		} else {
			allIDs[junctor.ID] = "junctor"
		}
		if _, exists := junctorByID[junctor.ID]; exists {
			result.error("junctor_id_duplicate", fmt.Sprintf("duplicate junctor id %q", junctor.ID), junctor.ID)
		} else {
			junctorByID[junctor.ID] = junctor
		}
		if junctor.Connector != argument.ConnectorAND && junctor.Connector != argument.ConnectorOR {
			result.error("junctor_connector_invalid", fmt.Sprintf("invalid connector %q", junctor.Connector), junctor.ID)
		}
		if len(junctor.Sources) < 2 {
			result.error("junctor_sources_too_few", "junctor must have at least two sources", junctor.ID)
		}
		if profile == ProfileConcludia && len(junctor.Sources) > 3 {
			result.warning("concludia_junctor_sources_many", fmt.Sprintf("junctor has %d sources; prefer at most 3 and introduce lemmas for clarity", len(junctor.Sources)), junctor.ID)
		}
		if _, exists := statementByID[junctor.Target]; !exists {
			result.error("junctor_target_unknown", fmt.Sprintf("junctor target %q does not exist", junctor.Target), junctor.ID)
		} else {
			incomingSupport[junctor.Target]++
		}
		seenSources := make(map[string]bool, len(junctor.Sources))
		junctorNode := "junctor:" + junctor.ID
		for _, source := range junctor.Sources {
			if seenSources[source] {
				result.error("junctor_source_duplicate", fmt.Sprintf("source %q appears more than once", source), junctor.ID)
			}
			seenSources[source] = true
			if _, exists := statementByID[source]; !exists {
				result.error("junctor_source_unknown", fmt.Sprintf("junctor source %q does not exist", source), junctor.ID)
				continue
			}
			if source == junctor.Target {
				result.error("support_self", "a statement cannot support itself", junctor.ID)
			}
			supportSources[source] = true
			addDirected(directed, "statement:"+source, junctorNode)
			if _, targetExists := statementByID[junctor.Target]; targetExists {
				addUndirected(undirected, source, junctor.Target)
			}
		}
		if _, exists := statementByID[junctor.Target]; exists {
			addDirected(directed, junctorNode, "statement:"+junctor.Target)
		}
	}

	for _, support := range doc.DirectSupports {
		if support.Connector != argument.ConnectorAND && support.Connector != argument.ConnectorOR {
			result.error("direct_support_connector_invalid", fmt.Sprintf("invalid connector %q", support.Connector), support.Source+"->"+support.Target)
		}
		_, sourceExists := statementByID[support.Source]
		_, targetExists := statementByID[support.Target]
		if !sourceExists {
			result.error("direct_support_source_unknown", fmt.Sprintf("direct support source %q does not exist", support.Source), support.Source)
		}
		if !targetExists {
			result.error("direct_support_target_unknown", fmt.Sprintf("direct support target %q does not exist", support.Target), support.Target)
		}
		if support.Source == support.Target {
			result.error("support_self", "a statement cannot directly support itself", support.Source)
		}
		if sourceExists && targetExists {
			incomingSupport[support.Target]++
			supportSources[support.Source] = true
			addDirected(directed, "statement:"+support.Source, "statement:"+support.Target)
			addUndirected(undirected, support.Source, support.Target)
		}
	}

	defeatsBySource := make(map[string]bool, len(doc.Defeats))
	for _, defeat := range doc.Defeats {
		source, sourceExists := statementByID[defeat.From]
		if !sourceExists {
			result.error("defeat_source_unknown", fmt.Sprintf("defeat source %q does not exist", defeat.From), defeat.From)
		} else if source.Role != argument.RoleCounterpoint {
			result.error("defeat_source_role", "defeat source must be a counterpoint", defeat.From)
		}
		if defeatsBySource[defeat.From] {
			result.error("defeat_source_multiple", "a counterpoint may have only one defeat target in .arg syntax", defeat.From)
		}
		defeatsBySource[defeat.From] = true

		switch defeat.Scope {
		case argument.DefeatPremise:
			target, exists := statementByID[defeat.To]
			if !exists {
				result.error("defeat_target_unknown", fmt.Sprintf("defeat target %q does not exist", defeat.To), defeat.From)
			} else {
				if target.Role != argument.RolePremise {
					result.error("defeat_premise_target_role", "premise-scope defeat must target a premise", defeat.From)
				}
				if sourceExists {
					validateDefeatEdge(defeat.From, defeat.To, directed, undirected, &result)
				}
				if profile == ProfileConcludia && incomingSupport[defeat.To] > 0 {
					result.error("concludia_defeat_target_not_leaf", fmt.Sprintf("defeat target %q must have no incoming support", defeat.To), defeat.From)
				}
			}
		case argument.DefeatCounterpoint:
			target, exists := statementByID[defeat.To]
			if !exists {
				result.error("defeat_target_unknown", fmt.Sprintf("defeat target %q does not exist", defeat.To), defeat.From)
			} else {
				if target.Role != argument.RoleCounterpoint {
					result.error("defeat_counterpoint_target_role", "counterpoint-scope defeat must target a counterpoint", defeat.From)
				}
				if sourceExists {
					validateDefeatEdge(defeat.From, defeat.To, directed, undirected, &result)
				}
				if profile == ProfileConcludia && incomingSupport[defeat.To] > 0 {
					result.error("concludia_defeat_target_not_leaf", fmt.Sprintf("defeat target %q must have no incoming support in the current Concludia profile", defeat.To), defeat.From)
				}
			}
		case argument.DefeatInference:
			junctor, exists := junctorByID[defeat.JunctorID]
			if !exists {
				result.error("defeat_junctor_unknown", fmt.Sprintf("defeat junctor %q does not exist", defeat.JunctorID), defeat.From)
			}
			if _, exists := statementByID[defeat.AtTarget]; !exists {
				result.error("defeat_target_unknown", fmt.Sprintf("inference defeat target %q does not exist", defeat.AtTarget), defeat.From)
			} else {
				if defeat.From == defeat.AtTarget {
					result.error("defeat_self", "a counterpoint cannot defeat itself", defeat.From)
				}
				if sourceExists {
					addUndirected(undirected, defeat.From, defeat.AtTarget)
				}
			}
			if exists && junctor.Target != defeat.AtTarget {
				result.error("defeat_inference_target_mismatch", fmt.Sprintf("inference defeat target %q does not match junctor target %q", defeat.AtTarget, junctor.Target), defeat.From)
			}
			if sourceExists && exists {
				addDirected(directed, "statement:"+defeat.From, "junctor:"+defeat.JunctorID)
			}
		default:
			result.error("defeat_scope_invalid", fmt.Sprintf("invalid defeat scope %q", defeat.Scope), defeat.From)
		}
	}

	if cycle := findCycle(directed); len(cycle) > 0 {
		result.error("relation_cycle", "directed relation cycle: "+strings.Join(cycle, " -> "), cycle[0])
	}

	if profile == ProfileConcludia {
		validateConcludia(doc, statementByID, incomingSupport, supportSources, undirected, &result)
	}
	return result
}

func validateDefeatEdge(from, to string, directed map[string][]string, undirected map[string]map[string]bool, result *Result) {
	if from == to {
		result.error("defeat_self", "a counterpoint cannot defeat itself", from)
		return
	}
	addDirected(directed, "statement:"+from, "statement:"+to)
	addUndirected(undirected, from, to)
}

func validateConcludia(doc *argument.Document, statements map[string]argument.Statement, incoming map[string]int, supportSources map[string]bool, undirected map[string]map[string]bool, result *Result) {
	if len(doc.Statements) == 1 {
		result.error("concludia_graph_too_small", "Concludia argument must contain more than one statement", doc.ID)
	}
	conclusionCount := 0
	for _, statement := range doc.Statements {
		if statement.Role == argument.RoleConclusion {
			conclusionCount++
		}
		if statement.Role == argument.RolePremise && incoming[statement.ID] > 0 {
			result.error("concludia_premise_supported", "premise must not be the target of support; promote it to a lemma", statement.ID)
		}
		if statement.Role == argument.RoleLemma && !supportSources[statement.ID] {
			result.warning("concludia_lemma_targetless", "lemma does not support another statement", statement.ID)
		}
		if len(undirected[statement.ID]) == 0 {
			result.error("concludia_statement_isolated", "statement is isolated", statement.ID)
		}
	}
	if conclusionCount == 0 {
		result.warning("concludia_conclusion_missing", "argument has no conclusion; treat it as a draft", doc.ID)
	}

	root := ""
	if declared, ok := doc.MetadataValue("root"); ok {
		if statement, exists := doc.Statement(declared); exists {
			root = statement.ID
		}
	}
	if root == "" {
		for _, statement := range doc.Statements {
			if statement.Role == argument.RoleConclusion {
				root = statement.ID
				break
			}
		}
	}
	if root == "" && len(doc.Statements) > 0 {
		root = doc.Statements[0].ID
	}
	visited := make(map[string]bool, len(statements))
	queue := []string{root}
	visited[root] = true
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for next := range undirected[current] {
			if !visited[next] {
				visited[next] = true
				queue = append(queue, next)
			}
		}
	}
	for _, statement := range doc.Statements {
		if !visited[statement.ID] {
			result.error("concludia_statement_disconnected", fmt.Sprintf("statement is disconnected from root %q", root), statement.ID)
		}
	}
}

func addDirected(graph map[string][]string, from, to string) {
	graph[from] = append(graph[from], to)
	if _, exists := graph[to]; !exists {
		graph[to] = nil
	}
}

func addUndirected(graph map[string]map[string]bool, left, right string) {
	if graph[left] == nil {
		graph[left] = make(map[string]bool)
	}
	if graph[right] == nil {
		graph[right] = make(map[string]bool)
	}
	graph[left][right] = true
	graph[right][left] = true
}

func findCycle(graph map[string][]string) []string {
	const (
		unseen = iota
		visiting
		done
	)
	state := make(map[string]int, len(graph))
	stack := make([]string, 0, len(graph))
	stackIndex := make(map[string]int, len(graph))
	var cycle []string
	var visit func(string) bool
	visit = func(node string) bool {
		state[node] = visiting
		stackIndex[node] = len(stack)
		stack = append(stack, node)
		for _, next := range graph[node] {
			switch state[next] {
			case unseen:
				if visit(next) {
					return true
				}
			case visiting:
				start := stackIndex[next]
				cycle = append(append([]string(nil), stack[start:]...), next)
				return true
			}
		}
		stack = stack[:len(stack)-1]
		delete(stackIndex, node)
		state[node] = done
		return false
	}
	nodes := make([]string, 0, len(graph))
	for node := range graph {
		nodes = append(nodes, node)
		sort.Strings(graph[node])
	}
	sort.Strings(nodes)
	for _, node := range nodes {
		if state[node] == unseen && visit(node) {
			return cycle
		}
	}
	return nil
}

func (r *Result) error(code, message, element string) {
	r.Diagnostics = append(r.Diagnostics, diagnostic.Diagnostic{Code: code, Message: message, Severity: diagnostic.SeverityError, Element: element})
}

func (r *Result) warning(code, message, element string) {
	r.Diagnostics = append(r.Diagnostics, diagnostic.Diagnostic{Code: code, Message: message, Severity: diagnostic.SeverityWarning, Element: element})
}
