// SPDX-FileCopyrightText: 2026 KeenWorks
// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/rivo/uniseg"
	"github.com/tunesmith/cludia/internal/argument"
	"github.com/tunesmith/cludia/internal/evaluation"
	"github.com/tunesmith/cludia/internal/query"
)

var (
	titleStyle         = lipgloss.NewStyle().Bold(true)
	statementHeadStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#1F2937", Dark: "#FFF7D6"})
	mutedStyle         = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"})
	warningStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#B45309", Dark: "#FBBF24"})
	selectedStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#9A6700", Dark: "#D7AF5F"})
	successStyle       = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#15803D", Dark: "#4ADE80"})
	errorStyle         = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#B91C1C", Dark: "#FB7185"})
)

func (m Model) View() string {
	switch m.mode {
	case modeDetail:
		return m.viewDetail()
	case modeLedger:
		return m.viewLedger()
	default:
		return m.viewTop()
	}
}

func (m Model) viewTop() string {
	lines, _, _ := m.renderedTopBody()
	body := renderLineViewport(lines, m.topScroll, m.viewportBudget())
	position, total := 0, len(m.topItems)
	if total > 0 {
		position = clampCursor(m.topCursor, total) + 1
	}
	title := fmt.Sprintf("%s — TOP · %d of %d", m.doc.Title, position, total)
	return m.frame(title, body, "j/k select  J/K reorder  Enter inspect  f derivation  q quit")
}

func (m Model) viewLedger() string {
	lines, _, _ := m.renderedLedgerBody()
	body := renderLineViewport(lines, m.ledgerScroll, m.viewportBudget())
	return m.frame("DERIVATION TO "+m.ledgerRoot, body, "j/k move  Enter inspect  Esc back  t Top  q quit")
}

func (m Model) viewDetail() string {
	statement, ok := m.doc.Statement(m.current)
	if !ok {
		return m.frame("STATEMENT", []string{mutedStyle.Render("statement missing")}, "Esc back  t Top  q quit")
	}
	body, _, _ := m.renderedDetailBody()
	footer := "j/k move  Enter follow  f derivation  Esc back  t Top  q quit"
	if statement.Role == "counterpoint" {
		footer = "j/k move  Enter follow  Esc back  t Top  q quit"
	}
	return m.frame("STATEMENT DETAIL", renderLineViewport(body, m.detailScroll, m.viewportBudget()), footer)
}

func (m Model) frame(title string, body []string, footer string) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(title))
	b.WriteByte('\n')
	b.WriteString(strings.Repeat("─", m.contentWidth()))
	b.WriteByte('\n')
	b.WriteString(strings.Join(body, "\n"))
	b.WriteByte('\n')
	if m.message != "" {
		style := mutedStyle
		switch m.messageKind {
		case messageSuccess:
			style = successStyle
		case messageError:
			style = errorStyle
		}
		b.WriteString(style.Render(m.message))
		b.WriteByte('\n')
	}
	b.WriteString(mutedStyle.Render(footer))
	return fitRenderedView(b.String(), m.width, m.height)
}

func (m Model) renderedTopBody() ([]string, int, int) {
	width := m.contentWidth()
	labelWidth := 5
	truthWidth := len("TRUTH")
	for _, item := range m.topItems {
		labelWidth = maxInt(labelWidth, displayWidth(topLabel(item, false)))
	}
	if len(m.topItems) == 0 {
		return []string{mutedStyle.Render("  no top statements")}, -1, -1
	}
	lines := []string{mutedStyle.Render("  " + pad("LABEL", labelWidth) + "  " + pad("TRUTH", truthWidth) + "  " + pad("DEPTH", 5) + "  STATEMENT")}
	selectedStart, selectedEnd := -1, -1
	for i, item := range m.topItems {
		if i == m.topCursor {
			selectedStart = len(lines)
		}
		lines = append(lines, renderTopItem(item, width, labelWidth, i == m.topCursor)...)
		if i == m.topCursor {
			selectedEnd = len(lines)
		}
	}
	return lines, selectedStart, selectedEnd
}

func (m Model) renderedLedgerBody() ([]string, int, int) {
	width := m.contentWidth()
	labelWidth := 5
	truthWidth := len("TRUTH")
	for _, row := range m.ledgerRows {
		labelWidth = maxInt(labelWidth, displayWidth(ledgerLabel(row, false)))
	}
	derivationWidth := minInt(34, maxInt(20, width/4))
	statementWidth := maxInt(26, width-2-labelWidth-2-truthWidth-2-derivationWidth-2)
	lines := []string{mutedStyle.Render("  " + pad("LABEL", labelWidth) + "  " + pad("TRUTH", truthWidth) + "  " + pad("STATEMENT", statementWidth) + "  DERIVATION")}
	selectedStart, selectedEnd := -1, -1
	for i, row := range m.ledgerRows {
		if i == m.ledgerCursor {
			selectedStart = len(lines)
		}
		lines = append(lines, renderLedgerItem(row, width, labelWidth, i == m.ledgerCursor)...)
		if i == m.ledgerCursor {
			selectedEnd = len(lines)
		}
	}
	return lines, selectedStart, selectedEnd
}

func (m Model) renderedDetailBody() ([]string, int, int) {
	statement, ok := m.doc.Statement(m.current)
	if !ok {
		return []string{mutedStyle.Render("statement missing")}, -1, -1
	}
	width := m.contentWidth()
	challenged := m.evaluation.TruthChangedByDefeat(statement.ID)
	value, _ := m.evaluation.Statement(statement.ID)
	status := truthStatus(statement.Truth, value.EffectiveTruth, value.TruthSource)
	header := renderID(displayID(statement.ID, challenged), challenged) + statementHeadStyle.Render(fmt.Sprintf("  %s[%s]  %s", statement.Role, statement.Kind, status))
	if value.Acceptance != "" {
		header += statementHeadStyle.Render("  " + strings.ToUpper(string(value.Acceptance)))
	}
	body := []string{header}
	for _, line := range wrapWords(statement.Text, width) {
		body = append(body, line)
	}
	body = append(body, "")
	selectableIndex := 0
	selectedStart, selectedEnd := -1, -1
	detailLines := m.detailLines()
	labelWidth := 1
	for _, line := range detailLines {
		if line.selectable {
			labelWidth = maxInt(labelWidth, displayWidth(displayID(line.id, line.challenged)))
		}
	}
	for _, line := range detailLines {
		indent := strings.Repeat("  ", line.indent)
		if line.header != "" {
			body = append(body, indent+mutedStyle.Render(line.header))
			continue
		}
		selected := line.selectable && selectableIndex == m.detailCursor
		marker := "  "
		if selected {
			marker = "> "
		}
		label := displayID(line.id, line.challenged)
		truth := ""
		if value, ok := m.evaluation.Statement(line.id); ok {
			truth = string(effectiveTruth(value.StoredTruth, value.EffectiveTruth))
		}
		prefixWidth := displayWidth(indent) + 2 + labelWidth + 2 + 1 + 2
		textWidth := maxInt(1, width-prefixWidth)
		wrapped := wrapWords(line.text, textWidth)
		if selected {
			selectedStart = len(body)
		}
		for i, text := range wrapped {
			prefix := strings.Repeat(" ", prefixWidth)
			if i == 0 {
				prefix = indent + marker + pad(renderSelectableID(label, line.challenged, selected), labelWidth) + "  " + truth + "  "
			}
			value := prefix + text
			if selected {
				value = selectedStyle.Render(value)
			}
			body = append(body, value)
		}
		if selected {
			selectedEnd = len(body)
		}
		if line.selectable {
			selectableIndex++
		}
	}
	return body, selectedStart, selectedEnd
}

func (m Model) viewportBudget() int {
	reserved := 3
	if m.message != "" {
		reserved++
	}
	return maxInt(1, m.height-reserved)
}

func renderLineViewport(lines []string, scroll, budget int) []string {
	if len(lines) == 0 {
		return nil
	}
	budget = maxInt(1, budget)
	maxScroll := maxInt(0, len(lines)-budget)
	scroll = minInt(maxInt(0, scroll), maxScroll)
	end := minInt(len(lines), scroll+budget)
	return lines[scroll:end]
}

func renderTopItem(item query.TopItem, width, labelWidth int, selected bool) []string {
	label := topLabel(item, selected)
	truth := string(effectiveTruth(item.Statement.Truth, item.EffectiveTruth))
	truthWidth := len("TRUTH")
	depth := ""
	if item.Depth > 0 {
		depth = fmt.Sprintf("%d", item.Depth)
	}
	marker := "  "
	if selected {
		marker = "> "
	}
	var lines []string
	if width < 80 || 2+labelWidth+2+truthWidth+2+5+2+20 > width {
		header := marker + label + "  " + truth
		if depth != "" {
			header += "  depth " + depth
		}
		lines = append(lines, header)
		for _, text := range wrapWords(item.Statement.Text, maxInt(1, width-4)) {
			lines = append(lines, "    "+text)
		}
	} else {
		textWidth := maxInt(20, width-2-labelWidth-2-truthWidth-2-5-2)
		wrapped := wrapWords(item.Statement.Text, textWidth)
		for i, text := range wrapped {
			if i == 0 {
				lines = append(lines, marker+pad(label, labelWidth)+"  "+pad(truth, truthWidth)+"  "+pad(depth, 5)+"  "+text)
			} else {
				lines = append(lines, strings.Repeat(" ", 2+labelWidth+2+truthWidth+2+5+2)+text)
			}
		}
	}
	if selected {
		for i := range lines {
			lines[i] = selectedStyle.Render(lines[i])
		}
	}
	return lines
}

func renderLedgerItem(row query.LedgerRow, width, labelWidth int, selected bool) []string {
	label := ledgerLabel(row, selected)
	truth := string(effectiveTruth(row.Statement.Truth, row.EffectiveTruth))
	truthWidth := len("TRUTH")
	marker := "  "
	if selected {
		marker = "> "
	}
	derivations := ledgerNotation(row)
	var lines []string
	if width < 80 || 2+labelWidth+2+truthWidth+2+26+2+20 > width {
		lines = append(lines, marker+label+"  "+truth)
		for _, text := range wrapWords(row.Statement.Text, maxInt(1, width-4)) {
			lines = append(lines, "    "+text)
		}
		for _, derivation := range derivations {
			lines = append(lines, "    "+derivation)
		}
	} else {
		derivationWidth := minInt(34, maxInt(20, width/4))
		textWidth := maxInt(26, width-2-labelWidth-2-truthWidth-2-derivationWidth-2)
		statementLines := wrapWords(row.Statement.Text, textWidth)
		derivationLines := make([]string, 0)
		for _, derivation := range derivations {
			derivationLines = append(derivationLines, wrapWords(derivation, derivationWidth)...)
		}
		count := maxInt(len(statementLines), len(derivationLines))
		for i := 0; i < count; i++ {
			labelCell, statementCell, derivationCell := "", "", ""
			lineMarker := "  "
			if i == 0 {
				labelCell, lineMarker = label, marker
			}
			if i < len(statementLines) {
				statementCell = statementLines[i]
			}
			if i < len(derivationLines) {
				derivationCell = derivationLines[i]
			}
			truthCell := ""
			if i == 0 {
				truthCell = truth
			}
			lines = append(lines, lineMarker+pad(labelCell, labelWidth)+"  "+pad(truthCell, truthWidth)+"  "+pad(statementCell, textWidth)+"  "+derivationCell)
		}
	}
	if selected {
		for i := range lines {
			lines[i] = selectedStyle.Render(lines[i])
		}
	}
	return lines
}

func ledgerNotation(row query.LedgerRow) []string {
	result := make([]string, 0, len(row.Derivations))
	for _, support := range row.Derivations {
		value := fmt.Sprintf("%s(%s)", support.Connector, strings.Join(support.Sources, ", "))
		if support.Type == "direct" {
			value += " [direct]"
		}
		result = append(result, value)
	}
	return result
}

func displayID(id string, challenged bool) string {
	if challenged {
		return id + "!"
	}
	return id
}

func topLabel(item query.TopItem, selected bool) string {
	id := displayID(item.Statement.ID, item.Challenged)
	return renderSelectableID(id, item.Challenged, selected)
}

func ledgerLabel(row query.LedgerRow, selected bool) string {
	id := displayID(row.Statement.ID, row.Challenged)
	return renderSelectableID(id, row.Challenged, selected)
}

func effectiveTruth(stored, effective argument.Truth) argument.Truth {
	if effective == argument.TruthTrue || effective == argument.TruthFalse || effective == argument.TruthUnknown {
		return effective
	}
	return stored
}

func truthStatus(stored, effective argument.Truth, source evaluation.TruthSource) string {
	if effective != argument.TruthTrue && effective != argument.TruthFalse && effective != argument.TruthUnknown {
		effective = stored
	}
	if source == evaluation.TruthAsserted && stored != effective {
		return fmt.Sprintf("%s → %s", stored, effective)
	}
	return string(effective)
}

func renderID(value string, challenged bool) string {
	if challenged {
		return warningStyle.Render(value)
	}
	return value
}

// A nested challenge style resets the outer selection style after the label.
// The exclamation mark still communicates challenge state while selection owns
// the complete highlighted row.
func renderSelectableID(value string, challenged, selected bool) string {
	if selected {
		return value
	}
	return renderID(value, challenged)
}

func wrapWords(text string, width int) []string {
	if width < 1 {
		width = 1
	}
	result := make([]string, 0)
	for _, paragraph := range strings.Split(text, "\n") {
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			result = append(result, "")
			continue
		}
		line := ""
		for _, word := range words {
			if displayWidth(word) > width {
				if line != "" {
					result = append(result, line)
					line = ""
				}
				parts := splitDisplayCells(word, width)
				result = append(result, parts[:len(parts)-1]...)
				line = parts[len(parts)-1]
				continue
			}
			candidate := word
			if line != "" {
				candidate = line + " " + word
			}
			if displayWidth(candidate) > width && line != "" {
				result = append(result, line)
				line = word
			} else {
				line = candidate
			}
		}
		result = append(result, line)
	}
	return result
}

func splitDisplayCells(value string, width int) []string {
	width = maxInt(1, width)
	parts := make([]string, 0)
	var part strings.Builder
	partWidth := 0
	graphemes := uniseg.NewGraphemes(value)
	for graphemes.Next() {
		cluster, clusterWidth := graphemes.Str(), graphemes.Width()
		if partWidth > 0 && partWidth+clusterWidth > width {
			parts = append(parts, part.String())
			part.Reset()
			partWidth = 0
		}
		part.WriteString(cluster)
		partWidth += clusterWidth
		if partWidth >= width {
			parts = append(parts, part.String())
			part.Reset()
			partWidth = 0
		}
	}
	if part.Len() > 0 || len(parts) == 0 {
		parts = append(parts, part.String())
	}
	return parts
}

func pad(value string, width int) string {
	missing := width - displayWidth(value)
	if missing > 0 {
		return value + strings.Repeat(" ", missing)
	}
	return value
}

func displayWidth(value string) int { return ansi.StringWidth(value) }

func fitRenderedView(value string, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	lines := strings.Split(value, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for index := range lines {
		lines[index] = ansi.Truncate(lines[index], width, "")
	}
	return strings.Join(lines, "\n")
}

func (m Model) contentWidth() int {
	return maxInt(1, m.width)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
