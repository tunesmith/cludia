package argfile

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tunesmith/cludia/internal/argument"
)

type orderedSupport struct {
	order    int
	direct   *argument.DirectSupport
	junctor  *argument.Junctor
	sequence int
}

func Serialize(doc *argument.Document) (string, error) {
	if doc == nil {
		return "", fmt.Errorf("document is nil")
	}
	if strings.ContainsAny(doc.ID, " \t\r\n") || doc.ID == "" {
		return "", fmt.Errorf("document id %q cannot be serialized", doc.ID)
	}
	if doc.Title == "" || strings.ContainsAny(doc.Title, "\r\n\"") {
		return "", fmt.Errorf("document title cannot be empty or contain quotes/newlines")
	}

	statementIDs := make(map[string]bool, len(doc.Statements))
	for _, statement := range doc.Statements {
		statementIDs[statement.ID] = true
	}
	supportsByTarget := make(map[string][]orderedSupport)
	sequence := 0
	for i := range doc.DirectSupports {
		support := &doc.DirectSupports[i]
		if !statementIDs[support.Source] || !statementIDs[support.Target] {
			return "", fmt.Errorf("direct support %s -> %s references an unknown statement", support.Source, support.Target)
		}
		sequence++
		supportsByTarget[support.Target] = append(supportsByTarget[support.Target], orderedSupport{order: support.Order, direct: support, sequence: sequence})
	}
	for i := range doc.Junctors {
		junctor := &doc.Junctors[i]
		if !statementIDs[junctor.Target] {
			return "", fmt.Errorf("junctor %s target %s is unknown", junctor.ID, junctor.Target)
		}
		for _, source := range junctor.Sources {
			if !statementIDs[source] {
				return "", fmt.Errorf("junctor %s source %s is unknown", junctor.ID, source)
			}
		}
		sequence++
		supportsByTarget[junctor.Target] = append(supportsByTarget[junctor.Target], orderedSupport{order: junctor.Order, junctor: junctor, sequence: sequence})
	}
	for target := range supportsByTarget {
		sort.SliceStable(supportsByTarget[target], func(i, j int) bool {
			left, right := supportsByTarget[target][i], supportsByTarget[target][j]
			if left.order == right.order {
				return left.sequence < right.sequence
			}
			if left.order == 0 {
				return false
			}
			if right.order == 0 {
				return true
			}
			return left.order < right.order
		})
	}

	defeatsBySource := make(map[string]argument.Defeat, len(doc.Defeats))
	for _, defeat := range doc.Defeats {
		if !statementIDs[defeat.From] {
			return "", fmt.Errorf("defeat source %s is unknown", defeat.From)
		}
		if _, exists := defeatsBySource[defeat.From]; exists {
			return "", fmt.Errorf("counterpoint %s has more than one defeat target", defeat.From)
		}
		defeatsBySource[defeat.From] = defeat
	}

	var b strings.Builder
	fmt.Fprintf(&b, "argument %s \"%s\"\n", doc.ID, doc.Title)
	if len(doc.Metadata) > 0 {
		b.WriteString("meta ")
		for i, entry := range doc.Metadata {
			if entry.Key == "" || strings.ContainsAny(entry.Key, " \t\r\n=,\"") {
				return "", fmt.Errorf("metadata key %q cannot be serialized", entry.Key)
			}
			if strings.ContainsAny(entry.Value, "\r\n\"") {
				return "", fmt.Errorf("metadata value for %s cannot contain quotes/newlines", entry.Key)
			}
			if i > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(&b, "%s=\"%s\"", entry.Key, entry.Value)
		}
		b.WriteByte('\n')
	}

	for _, statement := range doc.Statements {
		b.WriteByte('\n')
		defeat, hasDefeat := defeatsBySource[statement.ID]
		role := statement.Role
		if hasDefeat && statement.Role != argument.RoleCounterpoint {
			return "", fmt.Errorf("statement %s has a defeat target but is not a counterpoint", statement.ID)
		}
		if statement.Role == argument.RoleCounterpoint && hasDefeat {
			switch defeat.Scope {
			case argument.DefeatPremise:
				role = "undermine"
			case argument.DefeatInference:
				role = "undercut"
			}
		}
		fmt.Fprintf(&b, "%s[%s] %s", role, statement.Kind, statement.ID)
		if statement.Slug != "" {
			b.WriteByte(':')
			b.WriteString(statement.Slug)
		}
		if statement.Role == argument.RolePremise || statement.Role == argument.RoleCounterpoint {
			fmt.Fprintf(&b, " ::%s", statement.Truth)
		}
		if hasDefeat && strings.Contains(statement.Text, "\n") {
			return "", fmt.Errorf("statement %s: current Concludia syntax cannot attach a defeat target to multi-line text", statement.ID)
		}
		if err := writeStatementText(&b, statement.Text); err != nil {
			return "", fmt.Errorf("statement %s: %w", statement.ID, err)
		}
		if hasDefeat {
			clause, err := serializeDefeat(defeat)
			if err != nil {
				return "", err
			}
			b.WriteString(" -> ")
			b.WriteString(clause)
		}
		b.WriteByte('\n')

		for _, support := range supportsByTarget[statement.ID] {
			b.WriteString("  <- ")
			if support.direct != nil {
				connector := support.direct.Connector
				if connector == "" {
					connector = argument.ConnectorAND
				}
				fmt.Fprintf(&b, "%s(%s)\n", connector, support.direct.Source)
				continue
			}
			fmt.Fprintf(&b, "%s#%s(%s)\n", support.junctor.Connector, support.junctor.ID, strings.Join(support.junctor.Sources, ", "))
		}
	}
	return b.String(), nil
}

func writeStatementText(b *strings.Builder, text string) error {
	if strings.Contains(text, "\r") {
		return fmt.Errorf("text contains a carriage return")
	}
	if !strings.Contains(text, "\n") {
		b.WriteString(" \"")
		b.WriteString(text)
		b.WriteByte('"')
		return nil
	}
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == `"""` {
			return fmt.Errorf("multi-line text contains an unsupported closing triple-quote line")
		}
	}
	b.WriteString(" \"\"\"\n")
	for _, line := range strings.Split(text, "\n") {
		b.WriteString("  ")
		b.WriteString(line)
		b.WriteByte('\n')
	}
	b.WriteString("  \"\"\"")
	return nil
}

func serializeDefeat(defeat argument.Defeat) (string, error) {
	switch defeat.Scope {
	case argument.DefeatPremise:
		if defeat.To == "" {
			return "", fmt.Errorf("premise defeat from %s has no target", defeat.From)
		}
		return "premise " + defeat.To, nil
	case argument.DefeatCounterpoint:
		if defeat.To == "" {
			return "", fmt.Errorf("counterpoint defeat from %s has no target", defeat.From)
		}
		return "counterpoint " + defeat.To, nil
	case argument.DefeatInference:
		if defeat.JunctorID == "" || defeat.AtTarget == "" {
			return "", fmt.Errorf("inference defeat from %s requires junctor and target", defeat.From)
		}
		return fmt.Sprintf("inference %s:target %s", defeat.JunctorID, defeat.AtTarget), nil
	default:
		return "", fmt.Errorf("defeat from %s has unsupported scope %q", defeat.From, defeat.Scope)
	}
}
