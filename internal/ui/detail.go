// SPDX-FileCopyrightText: 2026 KeenWorks
// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"fmt"

	"github.com/tunesmith/cludia/internal/argument"
	"github.com/tunesmith/cludia/internal/query"
)

type detailLine struct {
	header     string
	id         string
	text       string
	indent     int
	selectable bool
	challenged bool
}

func (m Model) detailLines() []detailLine {
	statement, ok := m.doc.Statement(m.current)
	if !ok {
		return []detailLine{}
	}
	lines := make([]detailLine, 0)
	relations := query.StatementRelations(m.doc, statement.ID)
	if len(relations.IncomingSupport) > 0 {
		lines = append(lines, detailLine{header: "JUSTIFICATIONS"})
		for i, support := range relations.IncomingSupport {
			label := fmt.Sprintf("%d — %s", i+1, support.Connector)
			if support.Type == "direct" {
				label += " [direct]"
			}
			lines = append(lines, detailLine{header: label, indent: 1})
			for _, source := range support.Sources {
				lines = appendStatementDetailLine(lines, m.doc, source, 2)
			}
			if support.Type == "junctor" {
				undercuts := inferenceDefeats(m.doc, support.ID)
				if len(undercuts) > 0 {
					lines = append(lines, detailLine{header: "UNDERCUTS", indent: 2})
					for _, defeat := range undercuts {
						lines = appendCounterpointTree(lines, m.doc, defeat.From, 3, map[string]bool{})
					}
				}
			}
		}
	}

	directChallenges := directDefeats(m.doc, statement.ID)
	if len(directChallenges) > 0 {
		lines = append(lines, detailLine{header: "CHALLENGES TO STATEMENT"})
		for _, defeat := range directChallenges {
			lines = appendCounterpointTree(lines, m.doc, defeat.From, 1, map[string]bool{})
		}
	}

	if len(relations.OutgoingSupport) > 0 {
		lines = append(lines, detailLine{header: "USED BY"})
		for _, support := range relations.OutgoingSupport {
			lines = appendStatementDetailLine(lines, m.doc, support.Target, 1)
		}
	}

	if statement.Role == argument.RoleCounterpoint && len(relations.DefeatsOriginating) > 0 {
		lines = append(lines, detailLine{header: "TARGETS"})
		for _, defeat := range relations.DefeatsOriginating {
			target := defeat.To
			if defeat.Scope == argument.DefeatInference {
				target = defeat.AtTarget
			}
			lines = appendStatementDetailLine(lines, m.doc, target, 1)
		}
	}
	for index := range lines {
		if lines[index].id != "" {
			lines[index].challenged = m.evaluation.TruthChangedByDefeat(lines[index].id)
		}
	}
	return lines
}

func (m Model) detailSelectableIDs() []string {
	ids := make([]string, 0)
	for _, line := range m.detailLines() {
		if line.selectable {
			ids = append(ids, line.id)
		}
	}
	return ids
}

func appendStatementDetailLine(lines []detailLine, doc *argument.Document, id string, indent int) []detailLine {
	statement, ok := doc.Statement(id)
	if !ok {
		return lines
	}
	return append(lines, detailLine{
		id: statement.ID, text: statement.Text, indent: indent, selectable: true,
	})
}

func appendCounterpointTree(lines []detailLine, doc *argument.Document, id string, indent int, visiting map[string]bool) []detailLine {
	if visiting[id] {
		return lines
	}
	visiting[id] = true
	lines = appendStatementDetailLine(lines, doc, id, indent)
	for _, defeat := range doc.Defeats {
		if defeat.Scope == argument.DefeatCounterpoint && defeat.To == id {
			lines = appendCounterpointTree(lines, doc, defeat.From, indent+1, visiting)
		}
	}
	delete(visiting, id)
	return lines
}

func directDefeats(doc *argument.Document, id string) []argument.Defeat {
	result := make([]argument.Defeat, 0)
	for _, defeat := range doc.Defeats {
		if defeat.To == id {
			result = append(result, defeat)
		}
	}
	return result
}

func inferenceDefeats(doc *argument.Document, junctorID string) []argument.Defeat {
	result := make([]argument.Defeat, 0)
	for _, defeat := range doc.Defeats {
		if defeat.Scope == argument.DefeatInference && defeat.JunctorID == junctorID {
			result = append(result, defeat)
		}
	}
	return result
}
