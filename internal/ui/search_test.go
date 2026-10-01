// SPDX-FileCopyrightText: 2026 KeenWorks
// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"github.com/tunesmith/cludia/internal/argfile"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/tunesmith/cludia/internal/argument"
)

func typeSearch(m Model, text string) Model {
	for _, r := range text {
		m = pressKey(m, string(r))
	}
	return m
}

func TestSearchOpensEveryStatementAndRestoresNavigation(t *testing.T) {
	for _, reference := range []string{"P2", "two", "CP3", "L1"} {
		t.Run(reference, func(t *testing.T) {
			m := newModel("", testUIDocument(), diskVersion{})
			m.width, m.height = 100, 7
			m = pressKey(m, "j")
			cursor, scroll := m.topCursor, m.topScroll
			m = typeSearch(pressKey(m, "/"), reference)
			id := m.selectedSearchID()
			m = pressKey(m, "enter")
			if m.mode != modeDetail || m.current != id {
				t.Fatalf("did not open %s: %s", id, m.debugState())
			}
			if !strings.Contains(m.View(), "STATEMENT DETAIL") {
				t.Fatal(m.View())
			}
			m = pressKey(m, "esc")
			if m.mode != modeSearch || m.search.text != reference || m.selectedSearchID() != id {
				t.Fatalf("search not restored: %#v", m.search)
			}
			m = pressKey(m, "esc")
			if m.mode != modeTop || m.topCursor != cursor || m.topScroll != scroll {
				t.Fatalf("original position not restored: %s", m.debugState())
			}
		})
	}
}

func TestSearchExactReferencePrecedesMentionsAndSlugCollision(t *testing.T) {
	doc := testUIDocument()
	doc.Statements[0].Text = "Mentions P2 and two"
	doc.Statements[0].Slug = "P2" // imported collision must resolve by durable ID
	m := typeSearch(pressKey(newModel("", doc, diskVersion{}), "/"), "P2")
	if m.selectedSearchID() != "P2" {
		t.Fatalf("exact ID lost to mention/slug: %s", m.selectedSearchID())
	}
	m = typeSearch(pressKey(m, "ctrl+u"), "two")
	if m.selectedSearchID() != "P2" {
		t.Fatalf("exact slug lost to mention: %s", m.selectedSearchID())
	}
	m = typeSearch(pressKey(m, "ctrl+u"), "SOURCE")
	if len(m.searchMatches()) != 4 {
		t.Fatalf("case-insensitive text matches: %#v", m.searchMatches())
	}
}

func TestSearchInputDoesNotInvokeNavigationOrQuit(t *testing.T) {
	m := pressKey(newModel("", testUIDocument(), diskVersion{}), "/")
	for _, r := range "tqjkJKfl/" {
		next, cmd := updateWithKey(m, string(r))
		m = next
		if cmd != nil || m.mode != modeSearch {
			t.Fatalf("input %q invoked command", r)
		}
	}
	if m.search.text != "tqjkJKfl/" {
		t.Fatal(m.search.text)
	}
	m = pressKey(m, "ctrl+u")
	m = typeSearch(m, "P2")
	m = pressKey(m, "left")
	m = pressKey(m, "delete")
	m = typeSearch(m, "1")
	if m.search.text != "P1" || m.selectedSearchID() != "P1" {
		t.Fatalf("editing failed: %#v", m.search)
	}
	m = pressKey(m, "backspace")
	if m.search.text != "P" {
		t.Fatal(m.search.text)
	}
	_, cmd := updateWithKey(m, "ctrl+c")
	if cmd == nil {
		t.Fatal("Ctrl+C did not quit")
	}
}

func TestSearchEmptyAndNoMatchesStayInSearch(t *testing.T) {
	m := pressKey(newModel("", testUIDocument(), diskVersion{}), "/")
	for _, text := range []string{"", "nothing matches this", "   "} {
		m = typeSearch(pressKey(m, "ctrl+u"), text)
		m = pressKey(m, "enter")
		if m.mode != modeSearch || m.selectedSearchID() != "" {
			t.Fatalf("empty search opened a statement: %s", m.debugState())
		}
	}
}

func TestSearchNestedHistoryFromDetailAndLedger(t *testing.T) {
	for _, key := range []string{"enter", "f"} {
		m := pressKey(newModel("", testUIDocument(), diskVersion{}), key)
		original := m.snapshot()
		m = typeSearch(pressKey(m, "/"), "CP")
		m = pressKey(m, "down")
		chosen := m.selectedSearchID()
		m = pressKey(m, "enter")
		m = typeSearch(pressKey(m, "/"), "P2")
		m = pressKey(m, "enter")
		m = pressKey(m, "esc")
		if m.search.text != "P2" {
			t.Fatal("inner search lost")
		}
		m = pressKey(m, "esc")
		if m.current != chosen {
			t.Fatal("inner search did not return to detail")
		}
		m = pressKey(m, "esc")
		if m.search.text != "CP" || m.selectedSearchID() != chosen {
			t.Fatal("outer search lost")
		}
		m = pressKey(m, "esc")
		if m.mode != original.mode || m.current != original.current || m.ledgerRoot != original.ledgerRoot {
			t.Fatalf("original screen lost: %s", m.debugState())
		}
	}
}

func TestSearchViewportPagingAndDimensions(t *testing.T) {
	m := typeSearch(pressKey(newModel("", pageTestDocument(), diskVersion{}), "/"), "Statement")
	m.width, m.height = 100, 8
	m = pressKey(pressKey(m, "pgdown"), "pgdown")
	if m.search.cursor == 0 || m.search.scroll == 0 {
		t.Fatalf("search did not page: %#v", m.search)
	}
	chosen, scroll := m.selectedSearchID(), m.search.scroll
	m = pressKey(pressKey(m, "enter"), "esc")
	if m.selectedSearchID() != chosen || m.search.scroll != scroll {
		t.Fatal("search viewport lost on return")
	}
	for _, size := range [][2]int{{100, 8}, {30, 10}, {8, 5}, {1, 1}, {0, 0}} {
		next, _ := m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		resized := next.(Model)
		view := resized.View()
		if view == "" {
			continue
		}
		if len(strings.Split(view, "\n")) > size[1] {
			t.Fatalf("too many lines at %v: %q", size, view)
		}
		for _, line := range strings.Split(view, "\n") {
			if ansi.StringWidth(line) > size[0] {
				t.Fatalf("too wide at %v: %q", size, line)
			}
		}
	}
}

func TestSearchSelectionFollowsIDAcrossRefresh(t *testing.T) {
	path := filepath.Join(t.TempDir(), "search.arg")
	doc := pageTestDocument()
	save := func() {
		t.Helper()
		if err := argfile.SaveAtomic(path, doc); err != nil {
			t.Fatal(err)
		}
	}
	save()
	m := typeSearch(pressKey(newModel(path, doc, readDisk(path).version), "/"), "Statement")
	m = pressKey(m, "down")
	chosen := m.selectedSearchID()
	doc = doc.Clone()
	doc.Statements[0], doc.Statements[1] = doc.Statements[1], doc.Statements[0]
	save()
	m = m.refreshFromDisk(readDisk(path))
	if m.selectedSearchID() != chosen {
		t.Fatal("reorder changed selected identity")
	}
	if err := os.WriteFile(path, []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	m = m.refreshFromDisk(readDisk(path))
	if m.selectedSearchID() != chosen || !strings.Contains(m.message, "invalid") {
		t.Fatal("invalid reload lost search")
	}
	doc.Statements = append([]argument.Statement{}, doc.Statements[1:]...)
	save()
	m = m.refreshFromDisk(readDisk(path))
	if m.selectedSearchID() == chosen || m.mode != modeSearch || m.search.text != "Statement" {
		t.Fatal("deletion left stale search")
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	m = pressKey(pressKey(m, "enter"), "esc")
	after, err := os.ReadFile(path)
	if err != nil || string(before) != string(after) {
		t.Fatal("search navigation wrote workspace")
	}
}
