// SPDX-FileCopyrightText: 2026 KeenWorks
// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"fmt"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/tunesmith/cludia/internal/query"
)

type searchState struct {
	text        string
	inputCursor int
	cursor      int
	scroll      int
}

func (m Model) searchMatches() []query.StatementMatch {
	return query.SearchStatements(m.doc, m.search.text)
}

func (m Model) selectedSearchID() string {
	matches := m.searchMatches()
	if len(matches) == 0 {
		return ""
	}
	return matches[clampCursor(m.search.cursor, len(matches))].Statement.ID
}

func (m *Model) restoreSearchSelection(id string) {
	matches := m.searchMatches()
	m.search.cursor = clampCursor(m.search.cursor, len(matches))
	for i, match := range matches {
		if match.Statement.ID == id {
			m.search.cursor = i
			return
		}
	}
}

func (m Model) openSearch() Model {
	m.history = append(m.history, m.snapshot())
	m.mode, m.search = modeSearch, searchState{}
	m.clearMessage()
	return m
}

func (m Model) updateSearch(msg tea.KeyMsg) Model {
	text := []rune(m.search.text)
	pos := minInt(m.search.inputCursor, len(text))
	switch msg.String() {
	case "esc":
		return m.back()
	case "enter":
		if id := m.selectedSearchID(); id != "" {
			return m.openDetail(id)
		}
		return m
	case "up", "down":
		delta := 1
		if msg.String() == "up" {
			delta = -1
		}
		m.search.cursor = moveCursor(m.search.cursor, len(m.searchMatches()), delta)
		return m
	case "pgup", "pgdown":
		direction := 1
		if msg.String() == "pgup" {
			direction = -1
		}
		m.search.cursor = pageCursorByRenderedLines(m.search.cursor, len(m.searchMatches()), direction, m.searchBudget(), func(cursor int) (int, int) {
			candidate := m
			candidate.search.cursor = cursor
			_, start, end := candidate.renderedSearchBody()
			return start, end
		})
		return m
	case "left":
		m.search.inputCursor = maxInt(0, pos-1)
		return m
	case "right":
		m.search.inputCursor = minInt(len(text), pos+1)
		return m
	case "home", "ctrl+a":
		m.search.inputCursor = 0
		return m
	case "end", "ctrl+e":
		m.search.inputCursor = len(text)
		return m
	case "ctrl+u":
		text, pos = nil, 0
	case "backspace", "ctrl+h":
		if pos > 0 {
			text = append(text[:pos-1], text[pos:]...)
			pos--
		}
	case "delete":
		if pos < len(text) {
			text = append(text[:pos], text[pos+1:]...)
		}
	default:
		var inserted []rune
		if msg.Type == tea.KeyRunes || msg.Type == tea.KeySpace {
			for _, r := range msg.Runes {
				if !unicode.IsControl(r) {
					inserted = append(inserted, r)
				}
			}
			if msg.Type == tea.KeySpace {
				inserted = []rune{' '}
			}
		}
		if len(inserted) == 0 {
			return m
		}
		text = append(append(append([]rune{}, text[:pos]...), inserted...), text[pos:]...)
		pos += len(inserted)
	}
	m.search.inputCursor = pos
	if string(text) != m.search.text {
		m.search.text, m.search.cursor, m.search.scroll = string(text), 0, 0
		// Use the shared resolver: exact durable IDs take precedence over slugs.
		if statement, ok := m.doc.Statement(strings.TrimSpace(m.search.text)); ok && strings.TrimSpace(m.search.text) != "" {
			m.restoreSearchSelection(statement.ID)
		}
	}
	return m
}

func (m Model) searchBudget() int { return maxInt(1, m.viewportBudget()-1) }

func (m Model) renderedSearchBody() ([]string, int, int) {
	matches := m.searchMatches()
	if strings.TrimSpace(m.search.text) == "" {
		return []string{"Type a statement ID, slug, or text."}, -1, -1
	}
	if len(matches) == 0 {
		return []string{"No matching statements."}, -1, -1
	}
	lines := []string{}
	start, end := -1, -1
	for i, match := range matches {
		label := match.Statement.ID
		if match.Statement.Slug != "" {
			label += ":" + match.Statement.Slug
		}
		marker := "  "
		selected := i == clampCursor(m.search.cursor, len(matches))
		if selected {
			marker, start = "> ", len(lines)
		}
		for _, line := range wrapWords(marker+label+"  "+match.Statement.Text, m.contentWidth()) {
			if selected {
				line = selectedStyle.Render(line)
			}
			lines = append(lines, line)
		}
		if selected {
			end = len(lines)
		}
	}
	return lines, start, end
}

func (m Model) viewSearch() string {
	text := []rune(m.search.text)
	pos := minInt(m.search.inputCursor, len(text))
	before := string(text[:pos])
	// Keep the insertion point visible even for queries wider than the terminal.
	width := maxInt(1, m.contentWidth()-3)
	for ansi.StringWidth(before) > width {
		before = string([]rune(before)[1:])
	}
	prompt := "/ " + before + "▏" + string(text[pos:])
	lines, _, _ := m.renderedSearchBody()
	body := append([]string{prompt}, renderLineViewport(lines, m.search.scroll, m.searchBudget())...)
	count, position := len(m.searchMatches()), 0
	if count > 0 {
		position = clampCursor(m.search.cursor, count) + 1
	}
	return m.frame(fmt.Sprintf("SEARCH · %d of %d", position, count), body, "↑/↓ select  Enter inspect  Esc back  Ctrl+U clear")
}
