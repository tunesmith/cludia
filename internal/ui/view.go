package ui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
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
	return m.frame(m.doc.Title+" — TOP", body, "j/k move  Enter inspect  f derivation  q quit")
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
	b.WriteString(strings.Repeat("─", maxInt(1, m.contentWidth())))
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
	return b.String()
}

func (m Model) renderedTopBody() ([]string, int, int) {
	width := m.contentWidth()
	labelWidth := 5
	for _, item := range m.topItems {
		labelWidth = maxInt(labelWidth, runeCount(displayID(item.Statement.ID, item.Challenged)))
	}
	if len(m.topItems) == 0 {
		return []string{mutedStyle.Render("  no top statements")}, -1, -1
	}
	lines := []string{mutedStyle.Render("  " + pad("LABEL", labelWidth) + "  " + pad("DEPTH", 5) + "  STATEMENT")}
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
	for _, row := range m.ledgerRows {
		labelWidth = maxInt(labelWidth, runeCount(displayID(row.Statement.ID, row.Challenged)))
	}
	derivationWidth := minInt(34, maxInt(20, width/4))
	statementWidth := maxInt(26, width-2-labelWidth-2-derivationWidth-2)
	lines := []string{mutedStyle.Render("  " + pad("LABEL", labelWidth) + "  " + pad("STATEMENT", statementWidth) + "  DERIVATION")}
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
	challenged := query.StatementChallenged(m.doc, statement.ID)
	header := renderID(displayID(statement.ID, challenged), challenged) + statementHeadStyle.Render(fmt.Sprintf("  %s[%s]  %s", statement.Role, statement.Kind, statement.Truth))
	body := []string{header}
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
