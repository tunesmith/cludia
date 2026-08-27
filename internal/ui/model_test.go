package ui

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tunesmith/cludia/internal/argfile"
	"github.com/tunesmith/cludia/internal/argument"
)

func TestTopViewWrapsFullTextMarksChallengesAndNavigates(t *testing.T) {
	m := newModel("", testUIDocument(), diskVersion{})
	m.width, m.height = 90, 24
	view := m.View()
	flat := strings.Join(strings.Fields(view), " ")
	for _, want := range []string{"TOP", "LABEL", "DEPTH", "STATEMENT", "L2!", "A deliberately long final statement whose complete text must wrap without being summarized or omitted", "P5!"} {
		if !strings.Contains(flat, want) {
			t.Fatalf("top view missing %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "ROLE") || strings.Contains(view, "...") {
		t.Fatalf("top view contains forbidden content:\n%s", view)
	}
	m = pressKey(m, "j")
	if m.selectedTopID() != "P5" {
		t.Fatalf("selected top = %q", m.selectedTopID())
	}
	m = pressKey(m, "enter")
	if m.mode != modeDetail || m.current != "P5" {
		t.Fatalf("detail state: %s", m.debugState())
	}
}

func TestNarrowTopReflowsWithoutTruncating(t *testing.T) {
	m := newModel("", testUIDocument(), diskVersion{})
	m.width, m.height = 56, 30
	view := m.View()
	flat := strings.Join(strings.Fields(view), " ")
	if !strings.Contains(flat, "A deliberately long final statement whose complete text must wrap without being summarized or omitted") || strings.Contains(view, "...") {
		t.Fatalf("narrow view truncated statement:\n%s", view)
	}
}

func TestDetailScopesChallengesAndSupportsNavigationStack(t *testing.T) {
	m := newModel("", testUIDocument(), diskVersion{})
	m.width, m.height = 100, 32
	m = m.openDetail("L2")
	view := m.View()
	for _, want := range []string{"STATEMENT DETAIL", "L2!", "lemma[fact]", "JUSTIFICATIONS", "1 — AND", "UNDERCUTS", "CP3", "P2"} {
		if !strings.Contains(view, want) {
			t.Fatalf("detail view missing %q:\n%s", want, view)
		}
	}
	if got := m.detailSelectableIDs(); len(got) < 4 || got[0] != "L1" || got[2] != "CP3" {
		t.Fatalf("detail selectable ids = %#v", got)
	}
	m = pressKey(m, "enter")
	if m.current != "L1" || m.mode != modeDetail {
		t.Fatalf("followed detail = %s", m.debugState())
	}
	if view := m.View(); !strings.Contains(view, "USED BY") || !strings.Contains(view, "L2!") {
		t.Fatalf("downstream uses missing:\n%s", view)
	}
	m = pressKey(m, "esc")
	if m.current != "L2" || m.mode != modeDetail {
		t.Fatalf("back stack = %s", m.debugState())
	}

	m = m.openDetail("P5")
	view = m.View()
	for _, want := range []string{"CHALLENGES TO STATEMENT", "CP1", "CP2"} {
		if !strings.Contains(view, want) {
			t.Fatalf("counterpoint tree missing %q:\n%s", want, view)
		}
	}
}

func TestLedgerViewUsesCompactDerivationAndEnterNavigation(t *testing.T) {
	m := newModel("", testUIDocument(), diskVersion{})
	m.width, m.height = 110, 40
	m = pressKey(m, "f")
	if m.mode != modeLedger || m.ledgerCursor != 0 || len(m.ledgerRows) != 6 {
		t.Fatalf("ledger state = %s rows=%d", m.debugState(), len(m.ledgerRows))
	}
	view := m.View()
	flat := strings.Join(strings.Fields(view), " ")
	for _, want := range []string{"LABEL", "STATEMENT", "DERIVATION", "AND(P1, P2)", "OR(P3, P4)", "AND(P2) [direct]", "without being summarized or omitted"} {
		if !strings.Contains(flat, want) {
			t.Fatalf("ledger view missing %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "J1") || strings.Contains(view, "justified by") || strings.Contains(view, "...") {
		t.Fatalf("ledger view contains forbidden notation:\n%s", view)
	}
	m = pressKey(m, "enter")
	if m.mode != modeDetail || m.current != "P1" {
		t.Fatalf("ledger enter = %s", m.debugState())
	}
	m = pressKey(m, "h")
	if m.mode != modeLedger || m.ledgerCursor != 0 {
		t.Fatalf("ledger return = %s", m.debugState())
	}
}

func TestWindowSizeMessageReflowsView(t *testing.T) {
	m := newModel("", testUIDocument(), diskVersion{})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 56, Height: 14})
	m = updated.(Model)
	if m.width != 56 || m.height != 14 {
		t.Fatalf("window size = %dx%d", m.width, m.height)
	}
	if view := m.View(); !strings.Contains(view, "depth 2") {
		t.Fatalf("narrow reflow missing depth:\n%s", view)
	}
}

func TestDetailViewportFollowsLogicalSelection(t *testing.T) {
	m := newModel("", testUIDocument(), diskVersion{})
	m.width, m.height = 90, 12
	m = m.openDetail("L2")
	for range 3 {
		m = pressKey(m, "j")
	}
	view := m.View()
	ids := m.detailSelectableIDs()
	selected := ids[clampCursor(m.detailCursor, len(ids))]
	if !strings.Contains(view, selected) {
		t.Fatalf("selected detail %s not visible:\n%s", selected, view)
	}
}

func TestTallLedgerNeverScrollsWhileNavigating(t *testing.T) {
	m := newModel("", testUIDocument(), diskVersion{})
	m.width, m.height = 110, 20
	m = m.openLedger("L2").ensureSelectionVisible()
	for range len(m.ledgerRows) - 1 {
		m = pressKey(m, "j")
		if m.ledgerScroll != 0 {
			t.Fatalf("tall ledger scrolled down at cursor %d: %d", m.ledgerCursor, m.ledgerScroll)
		}
	}
	for range len(m.ledgerRows) - 1 {
		m = pressKey(m, "k")
		if m.ledgerScroll != 0 {
			t.Fatalf("tall ledger scrolled up at cursor %d: %d", m.ledgerCursor, m.ledgerScroll)
		}
	}
}

func TestShortLedgerScrollsOnlyAtViewportEdges(t *testing.T) {
	m := newModel("", testUIDocument(), diskVersion{})
	m.width, m.height = 110, 7 // four body lines: header plus three one-line rows
	m = m.openLedger("L2").ensureSelectionVisible()

	for m.ledgerCursor < len(m.ledgerRows)-1 {
		before := m
		next := pressKey(m, "j")
		_, start, end := next.renderedLedgerBody()
		wasVisible := start >= before.ledgerScroll && end <= before.ledgerScroll+before.viewportBudget()
		if wasVisible && next.ledgerScroll != before.ledgerScroll {
			t.Fatalf("downward navigation shifted an already-visible row: cursor=%d before=%d after=%d", next.ledgerCursor, before.ledgerScroll, next.ledgerScroll)
		}
		m = next
	}
	if m.ledgerScroll == 0 {
		t.Fatal("short ledger never scrolled")
	}

	stableReversals := 0
	for m.ledgerCursor > 0 {
		before := m
		next := pressKey(m, "k")
		_, start, end := next.renderedLedgerBody()
		wasVisible := start >= before.ledgerScroll && end <= before.ledgerScroll+before.viewportBudget()
		if wasVisible {
			stableReversals++
			if next.ledgerScroll != before.ledgerScroll {
				t.Fatalf("reverse navigation shifted an already-visible row: cursor=%d before=%d after=%d", next.ledgerCursor, before.ledgerScroll, next.ledgerScroll)
			}
		}
		m = next
	}
	if stableReversals == 0 {
		t.Fatal("test did not exercise a visible row while reversing")
	}
	if m.ledgerScroll != 0 {
		t.Fatalf("return to first row did not restore header: %d", m.ledgerScroll)
	}
}

func TestTopAndDetailUsePersistentViewportOffsets(t *testing.T) {
	m := newModel("", testUIDocument(), diskVersion{})
	m.width, m.height = 56, 7
	m = m.ensureSelectionVisible()
	m = pressKey(m, "j")
	if m.topScroll == 0 {
		t.Fatal("short wrapped Top did not scroll to reveal second item")
	}
	topScroll := m.topScroll
	m = pressKey(m, "k")
	if m.topScroll >= topScroll {
		t.Fatalf("Top did not move minimally toward first item: before=%d after=%d", topScroll, m.topScroll)
	}

	m = m.openDetail("L2").ensureSelectionVisible()
	for range 3 {
		m = pressKey(m, "j")
	}
	if m.detailScroll == 0 {
		t.Fatal("short Detail did not scroll to reveal later relation")
	}
	detailScroll := m.detailScroll
	m = pressKey(m, "k")
	if m.detailScroll != detailScroll {
		t.Fatalf("Detail shifted while reversed selection remained visible: before=%d after=%d", detailScroll, m.detailScroll)
	}
}

func TestNavigationStackRestoresCursorAndScroll(t *testing.T) {
	m := newModel("", testUIDocument(), diskVersion{})
	m.width, m.height = 110, 7
	m = m.openLedger("L2").ensureSelectionVisible()
	for range 4 {
		m = pressKey(m, "j")
	}
	wantCursor, wantScroll := m.ledgerCursor, m.ledgerScroll
	m = pressKey(m, "enter")
	if m.mode != modeDetail {
		t.Fatalf("did not enter detail: %s", m.debugState())
	}
	m = pressKey(m, "h")
	if m.mode != modeLedger || m.ledgerCursor != wantCursor || m.ledgerScroll != wantScroll {
		t.Fatalf("ledger viewport not restored: cursor=%d/%d scroll=%d/%d", m.ledgerCursor, wantCursor, m.ledgerScroll, wantScroll)
	}
}

func TestResizeToTallWindowClearsUnneededScroll(t *testing.T) {
	m := newModel("", testUIDocument(), diskVersion{})
	m.width, m.height = 110, 7
	m = m.openLedger("L2").ensureSelectionVisible()
	for range 5 {
		m = pressKey(m, "j")
	}
	if m.ledgerScroll == 0 {
		t.Fatal("precondition: short ledger did not scroll")
	}
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 110, Height: 30})
	m = updated.(Model)
	if m.ledgerScroll != 0 {
		t.Fatalf("tall resize retained unnecessary scroll: %d", m.ledgerScroll)
	}
}

func TestLiveReloadPreservesSelectionRejectsInvalidAndHandlesDeletion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workspace.arg")
	doc := testUIDocument()
	if err := argfile.SaveAtomic(path, doc); err != nil {
		t.Fatal(err)
	}
	check := readDisk(path)
	m := newModel(path, doc, check.version)
	m.topCursor = 1

	next := doc.Clone()
	next.Statements = append(next.Statements, argument.Statement{ID: "P6", Role: argument.RolePremise, Kind: argument.KindFact, Truth: argument.TruthTrue, Text: "New top statement"})
	if err := argfile.SaveAtomic(path, next); err != nil {
		t.Fatal(err)
	}
	m = m.refreshFromDisk(readDisk(path))
	if m.selectedTopID() != "P5" || !strings.Contains(m.message, "reloaded") {
		t.Fatalf("reload did not preserve selection: %s message=%q", m.debugState(), m.message)
	}

	if err := os.WriteFile(path, []byte("invalid"), 0o600); err != nil {
		t.Fatal(err)
	}
	m = m.refreshFromDisk(readDisk(path))
	if !strings.Contains(m.message, "invalid") {
		t.Fatalf("invalid reload message = %q", m.message)
	}
	if _, ok := m.doc.Statement("P6"); !ok {
		t.Fatal("invalid reload replaced last valid document")
	}
	if err := argfile.SaveAtomic(path, next); err != nil {
		t.Fatal(err)
	}
	m = m.refreshFromDisk(readDisk(path))
	if !strings.Contains(m.message, "restored") {
		t.Fatalf("restored reload message = %q", m.message)
	}

	withoutP5 := next.Clone()
	statements := make([]argument.Statement, 0, len(withoutP5.Statements)-1)
	for _, statement := range withoutP5.Statements {
		if statement.ID != "P5" {
			statements = append(statements, statement)
		}
	}
	withoutP5.Statements = statements
	withoutP5.Defeats = nil
	if err := argfile.SaveAtomic(path, withoutP5); err != nil {
		t.Fatal(err)
	}
	m.mode, m.current = modeDetail, "P5"
	m = m.refreshFromDisk(readDisk(path))
	if m.mode != modeTop || !strings.Contains(m.message, "removed") {
		t.Fatalf("deleted selection reload = %s message=%q", m.debugState(), m.message)
	}
}

func TestBackStackFallsBackWhenPriorStatementWasRemoved(t *testing.T) {
	m := newModel("", testUIDocument(), diskVersion{})
	m = m.openDetail("L2")
	m = m.openDetail("L1")
	next := m.doc.Clone()
	statements := make([]argument.Statement, 0, len(next.Statements)-1)
	for _, statement := range next.Statements {
		if statement.ID != "L2" {
			statements = append(statements, statement)
		}
	}
	next.Statements = statements
	next.Junctors = []argument.Junctor{}
	next.DirectSupports = nil
	next.Defeats = nil
	m.doc = next
	m = pressKey(m, "esc")
	if m.mode != modeTop || !strings.Contains(m.message, "removed") {
		t.Fatalf("back after removal = %s message=%q", m.debugState(), m.message)
	}
}

func TestReadOnlyNavigationDoesNotWriteFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workspace.arg")
	doc := testUIDocument()
	if err := argfile.SaveAtomic(path, doc); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	m := newModel(path, doc, readDisk(path).version)
	m = pressKey(m, "enter")
	m = pressKey(m, "f")
	m = pressKey(m, "esc")
	m = pressKey(m, "esc")
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("read-only navigation wrote workspace: %v", err)
	}
}

func TestMissingFileReloadRetainsLastValidDocument(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workspace.arg")
	doc := testUIDocument()
	if err := argfile.SaveAtomic(path, doc); err != nil {
		t.Fatal(err)
	}
	m := newModel(path, doc, readDisk(path).version)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	m = m.refreshFromDisk(readDisk(path))
	if !strings.Contains(m.message, "no longer exists") {
		t.Fatalf("missing-file message = %q", m.message)
	}
	if _, ok := m.doc.Statement("L2"); !ok {
		t.Fatal("missing file discarded last valid document")
	}
}

func TestEmptyTopView(t *testing.T) {
	doc := &argument.Document{ID: "empty-top", Title: "Empty Top", Statements: []argument.Statement{{ID: "CP1", Role: argument.RoleCounterpoint, Kind: argument.KindFact, Truth: argument.TruthTrue, Text: "Challenge"}}}
	m := newModel("", doc, diskVersion{})
	if view := m.View(); !strings.Contains(view, "no top statements") {
		t.Fatalf("empty top view:\n%s", view)
	}
}

func pressKey(m Model, key string) Model {
	var msg tea.KeyMsg
	switch key {
	case "enter":
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	default:
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
	updated, _ := m.Update(msg)
	return updated.(Model)
}

func testUIDocument() *argument.Document {
	statement := func(id, slug, text string, role argument.Role) argument.Statement {
		truth := argument.TruthUnknown
		if role == argument.RolePremise || role == argument.RoleCounterpoint {
			truth = argument.TruthTrue
		}
		return argument.Statement{ID: id, Slug: slug, Text: text, Role: role, Kind: argument.KindFact, Truth: truth}
	}
	return &argument.Document{
		ID: "tui", Title: "TUI Test", Metadata: []argument.Metadata{{Key: "profile", Value: "workspace"}},
		Statements: []argument.Statement{
			statement("P1", "one", "First source statement", argument.RolePremise),
			statement("P2", "two", "Second source statement", argument.RolePremise),
			statement("P3", "three", "Third source statement", argument.RolePremise),
			statement("P4", "four", "Fourth source statement", argument.RolePremise),
			statement("L1", "middle", "Intermediate statement", argument.RoleLemma),
			statement("L2", "final", "A deliberately long final statement whose complete text must wrap without being summarized or omitted", argument.RoleLemma),
			statement("P5", "isolated", "Isolated challenged statement", argument.RolePremise),
			statement("CP1", "challenge", "Challenge to isolated statement", argument.RoleCounterpoint),
			statement("CP2", "answer", "Counterpoint to the challenge", argument.RoleCounterpoint),
			statement("CP3", "undercut", "Challenge to final inference", argument.RoleCounterpoint),
		},
		Junctors: []argument.Junctor{
			{ID: "J1", Connector: argument.ConnectorAND, Sources: []string{"P1", "P2"}, Target: "L1"},
			{ID: "J2", Connector: argument.ConnectorOR, Sources: []string{"P3", "P4"}, Target: "L1"},
			{ID: "J3", Connector: argument.ConnectorAND, Sources: []string{"L1", "P4"}, Target: "L2"},
		},
		DirectSupports: []argument.DirectSupport{{Source: "P2", Target: "L2", Connector: argument.ConnectorAND}},
		Defeats: []argument.Defeat{
			{From: "CP1", Scope: argument.DefeatPremise, To: "P5"},
			{From: "CP2", Scope: argument.DefeatCounterpoint, To: "CP1"},
			{From: "CP3", Scope: argument.DefeatInference, JunctorID: "J3", AtTarget: "L2"},
		},
	}
}
