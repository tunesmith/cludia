package ui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/tunesmith/cludia/internal/query"
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true)
	mutedStyle    = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"})
	warningStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#B45309", Dark: "#FBBF24"})
	selectedStyle = lipgloss.NewStyle().Bold(true)
)

type renderedItem struct {
	id    string
	lines []string
}

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
	width := m.contentWidth()
	items := make([]renderedItem, 0, len(m.topItems))
	labelWidth := 5
	for _, item := range m.topItems {
		labelWidth = maxInt(labelWidth, runeCount(displayID(item.Statement.ID, item.Challenged)))
	}
	for i, item := range m.topItems {
		items = append(items, renderedItem{id: item.Statement.ID, lines: renderTopItem(item, width, labelWidth, i == m.topCursor)})
	}
	var body []string
	if len(items) == 0 {
		body = []string{mutedStyle.Render("  no top statements")}
	} else {
		header := "  " + pad("LABEL", labelWidth) + "  " + pad("DEPTH", 5) + "  STATEMENT"
		body = append([]string{mutedStyle.Render(header)}, visibleRenderedItems(items, m.topCursor, m.bodyHeight(5))...)
	}
	return m.frame(m.doc.Title+" — TOP", body, "j/k move  Enter inspect  f derivation  q quit")
}

func (m Model) viewLedger() string {
	width := m.contentWidth()
	items := make([]renderedItem, 0, len(m.ledgerRows))
	labelWidth := 5
	for _, row := range m.ledgerRows {
		labelWidth = maxInt(labelWidth, runeCount(displayID(row.Statement.ID, row.Challenged)))
	}
	for i, row := range m.ledgerRows {
		items = append(items, renderedItem{id: row.Statement.ID, lines: renderLedgerItem(row, width, labelWidth, i == m.ledgerCursor)})
	}
	derivationWidth := minInt(34, maxInt(20, width/4))
	statementWidth := maxInt(26, width-2-labelWidth-2-derivationWidth-2)
	header := "  " + pad("LABEL", labelWidth) + "  " + pad("STATEMENT", statementWidth) + "  DERIVATION"
	body := append([]string{mutedStyle.Render(header)}, visibleRenderedItems(items, m.ledgerCursor, m.bodyHeight(5))...)
	return m.frame("DERIVATION TO "+m.ledgerRoot, body, "j/k move  Enter inspect  h/Esc back  q quit")
}

func (m Model) viewDetail() string {
	statement, ok := m.doc.Statement(m.current)
	if !ok {
		return m.frame("STATEMENT", []string{mutedStyle.Render("statement missing")}, "h/Esc back  q quit")
	}
	width := m.contentWidth()
	header := fmt.Sprintf("%s  %s[%s]  %s", displayID(statement.ID, query.StatementChallenged(m.doc, statement.ID)), statement.Role, statement.Kind, statement.Truth)
	body := []string{titleStyle.Render(header)}
	for _, line := range wrapWords(statement.Text, maxInt(10, width)) {
		body = append(body, line)
	}
	body = append(body, "")
	selectableIndex := 0
	selectedStart, selectedEnd := -1, -1
	for _, line := range m.detailLines() {
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
		textWidth := maxInt(10, width-runeCount(indent)-2-runeCount(label)-2)
		wrapped := wrapWords(line.text, textWidth)
		if selected {
			selectedStart = len(body)
		}
		for i, text := range wrapped {
			prefix := strings.Repeat(" ", runeCount(indent)+2+runeCount(label)+2)
			if i == 0 {
				prefix = indent + marker + renderID(label, line.challenged) + "  "
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
	footer := "j/k move  Enter follow  f derivation  h/Esc back  q quit"
	if statement.Role == "counterpoint" {
		footer = "j/k move  Enter follow  h/Esc back  q quit"
	}
	return m.frame("STATEMENT DETAIL", visibleLineWindow(body, selectedStart, selectedEnd, m.bodyHeight(4)), footer)
}

func (m Model) frame(title string, body []string, footer string) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(title))
	b.WriteByte('\n')
	b.WriteString(strings.Repeat("─", maxInt(1, m.contentWidth())))
	b.WriteByte('\n')
	b.WriteString(strings.Join(body, "\n"))
	b.WriteByte('\n')
	if m.message != "" {
		b.WriteString(mutedStyle.Render(m.message))
		b.WriteByte('\n')
	}
	b.WriteString(mutedStyle.Render(footer))
	return b.String()
}

func renderTopItem(item query.TopItem, width, labelWidth int, selected bool) []string {
	label := displayID(item.Statement.ID, item.Challenged)
	depth := ""
	if item.Depth > 0 {
		depth = fmt.Sprintf("%d", item.Depth)
	}
	marker := "  "
	if selected {
		marker = "> "
	}
	var lines []string
	if width < 80 {
		header := marker + renderID(label, item.Challenged)
		if depth != "" {
			header += "  depth " + depth
		}
		lines = append(lines, header)
		for _, text := range wrapWords(item.Statement.Text, maxInt(10, width-4)) {
			lines = append(lines, "    "+text)
		}
	} else {
		textWidth := maxInt(20, width-2-labelWidth-2-5-2)
		wrapped := wrapWords(item.Statement.Text, textWidth)
		for i, text := range wrapped {
			if i == 0 {
				lines = append(lines, marker+pad(renderID(label, item.Challenged), labelWidth)+"  "+pad(depth, 5)+"  "+text)
			} else {
				lines = append(lines, strings.Repeat(" ", 2+labelWidth+2+5+2)+text)
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
	label := displayID(row.Statement.ID, row.Challenged)
	marker := "  "
	if selected {
		marker = "> "
	}
	derivations := ledgerNotation(row)
	var lines []string
	if width < 80 {
		lines = append(lines, marker+renderID(label, row.Challenged))
		for _, text := range wrapWords(row.Statement.Text, maxInt(10, width-4)) {
			lines = append(lines, "    "+text)
		}
		for _, derivation := range derivations {
			lines = append(lines, "    "+derivation)
		}
	} else {
		derivationWidth := minInt(34, maxInt(20, width/4))
		textWidth := maxInt(26, width-2-labelWidth-2-derivationWidth-2)
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
				labelCell, lineMarker = renderID(label, row.Challenged), marker
			}
			if i < len(statementLines) {
				statementCell = statementLines[i]
			}
			if i < len(derivationLines) {
				derivationCell = derivationLines[i]
			}
			lines = append(lines, lineMarker+pad(labelCell, labelWidth)+"  "+pad(statementCell, textWidth)+"  "+derivationCell)
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

func visibleRenderedItems(items []renderedItem, cursor, budget int) []string {
	if len(items) == 0 {
		return []string{}
	}
	cursor = clampCursor(cursor, len(items))
	if budget < 1 {
		budget = 1
	}
	start, end := cursor, cursor+1
	used := len(items[cursor].lines)
	for start > 0 && used+len(items[start-1].lines) <= budget {
		start--
		used += len(items[start].lines)
	}
	for end < len(items) && used+len(items[end].lines) <= budget {
		used += len(items[end].lines)
		end++
	}
	lines := make([]string, 0, used)
	for i := start; i < end; i++ {
		lines = append(lines, items[i].lines...)
	}
	return lines
}

func visibleLineWindow(lines []string, selectedStart, selectedEnd, budget int) []string {
	if budget <= 0 || len(lines) <= budget {
		return lines
	}
	if selectedStart < 0 {
		return lines[:budget]
	}
	start := selectedStart
	if selectedEnd-selectedStart < budget {
		start = maxInt(0, selectedEnd-budget)
	}
	end := minInt(len(lines), start+budget)
	if selectedEnd > end {
		end = minInt(len(lines), selectedEnd)
		start = maxInt(0, end-budget)
	}
	return lines[start:end]
}

func displayID(id string, challenged bool) string {
	if challenged {
		return id + "!"
	}
	return id
}

func renderID(value string, challenged bool) string {
	if challenged {
		return warningStyle.Render(value)
	}
	return value
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
			if runeCount(word) > width {
				if line != "" {
					result = append(result, line)
					line = ""
				}
				parts := splitRunes(word, width)
				result = append(result, parts[:len(parts)-1]...)
				line = parts[len(parts)-1]
				continue
			}
			candidate := word
			if line != "" {
				candidate = line + " " + word
			}
			if runeCount(candidate) > width && line != "" {
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

func splitRunes(value string, width int) []string {
	runes := []rune(value)
	parts := make([]string, 0, (len(runes)+width-1)/width)
	for len(runes) > width {
		parts = append(parts, string(runes[:width]))
		runes = runes[width:]
	}
	parts = append(parts, string(runes))
	return parts
}

func pad(value string, width int) string {
	missing := width - lipgloss.Width(value)
	if missing > 0 {
		return value + strings.Repeat(" ", missing)
	}
	return value
}

func runeCount(value string) int { return utf8.RuneCountInString(value) }

func (m Model) contentWidth() int {
	if m.width < 20 {
		return 20
	}
	return m.width
}

func (m Model) bodyHeight(reserved int) int {
	if m.height <= reserved {
		return 1
	}
	return m.height - reserved
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
