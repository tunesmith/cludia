package ui

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/tunesmith/cludia/internal/argfile"
	"github.com/tunesmith/cludia/internal/argument"
	"github.com/tunesmith/cludia/internal/query"
)

func TestTopViewWrapsFullTextMarksChallengesAndNavigates(t *testing.T) {
	m := newModel("", testUIDocument(), diskVersion{})
	m.width, m.height = 90, 24
	view := m.View()
	flat := strings.Join(strings.Fields(view), " ")
	for _, want := range []string{"TOP · 1 of 2", "LABEL", "TRUTH", "DEPTH", "STATEMENT", "L2!", "A deliberately long final statement whose complete text must wrap without being summarized or omitted", "P5!"} {
		if !strings.Contains(flat, want) {
			t.Fatalf("top view missing %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "ROLE") || strings.Contains(view, "derived") || strings.Contains(view, "...") {
		t.Fatalf("top view contains forbidden content:\n%s", view)
	}
	m = pressKey(m, "j")
	if m.selectedTopID() != "P5" {
		t.Fatalf("selected top = %q", m.selectedTopID())
	}
	if view := m.View(); !strings.Contains(view, "TOP · 2 of 2") {
		t.Fatalf("top view did not update position:\n%s", view)
	}
	m = pressKey(m, "enter")
	if m.mode != modeDetail || m.current != "P5" {
		t.Fatalf("detail state: %s", m.debugState())
	}
}

func TestSelectedChallengedTopRowKeepsOneStyleAcrossFirstLine(t *testing.T) {
	originalSelected, originalWarning := selectedStyle, warningStyle
	selectedStyle = lipgloss.NewStyle().Transform(func(value string) string { return "SELECT[" + value + "]" })
	warningStyle = lipgloss.NewStyle().Transform(func(value string) string { return "WARN[" + value + "]" })
	t.Cleanup(func() {
		selectedStyle, warningStyle = originalSelected, originalWarning
	})

	item := query.Top(testUIDocument())[0]
	lines := renderTopItem(item, 90, 5, true)
	if len(lines) < 2 {
		t.Fatalf("test row did not wrap: %#v", lines)
	}
	for _, line := range lines {
		if !strings.HasPrefix(line, "SELECT[") || !strings.HasSuffix(line, "]") || strings.Contains(line, "WARN[") {
			t.Fatalf("selected challenged row has interrupted selection styling: %q", line)
		}
	}
	unselected := renderTopItem(item, 90, 5, false)
	if !strings.Contains(unselected[0], "WARN[L2!]") {
		t.Fatalf("unselected challenged label lost warning styling: %q", unselected[0])
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

func TestUnicodeWrappingUsesDisplayCellsAndPreservesGraphemes(t *testing.T) {
	text := "界界 e\u0301lan 👩‍💻 family👨‍👩‍👧‍👦"
	lines := wrapWords(text, 4)
	for _, line := range lines {
		if width := displayWidth(line); width > 4 {
			t.Fatalf("wrapped line width = %d: %q (%#v)", width, line, lines)
		}
		if !utf8.ValidString(line) {
			t.Fatalf("invalid UTF-8 line: %q", line)
		}
	}
	joined := strings.ReplaceAll(strings.Join(lines, ""), " ", "")
	if want := strings.ReplaceAll(text, " ", ""); joined != want {
		t.Fatalf("wrapped text changed content: got %q want %q (%#v)", joined, want, lines)
	}
	parts := splitDisplayCells("e\u0301👩‍💻界", 2)
	if got, want := parts, []string{"e\u0301", "👩‍💻", "界"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("grapheme split = %#v, want %#v", got, want)
	}
}

func TestViewNeverExceedsTerminalDimensions(t *testing.T) {
	doc := testUIDocument().Clone()
	doc.Title = "非常に長い題名 👩‍💻 e\u0301vidence"
	doc.Statements[5].Text = "漢字 evidence e\u0301lan 👨‍👩‍👧‍👦 remains complete at usable widths"
	for _, size := range []struct{ width, height int }{
		{1, 1}, {8, 3}, {20, 5}, {56, 12}, {90, 30},
	} {
		base := newModel("", doc, diskVersion{})
		for _, screen := range []struct {
			name  string
			model Model
		}{
			{name: "top", model: base},
			{name: "detail", model: base.openDetail("L2")},
			{name: "ledger", model: base.openLedger("L2")},
		} {
			m := screen.model
			m.width, m.height = size.width, size.height
			view := m.View()
			lines := strings.Split(view, "\n")
			if len(lines) > size.height {
				t.Fatalf("%s %dx%d view has %d lines:\n%s", screen.name, size.width, size.height, len(lines), view)
			}
			for _, line := range lines {
				if width := ansi.StringWidth(line); width > size.width {
					t.Fatalf("%s %dx%d line width = %d: %q", screen.name, size.width, size.height, width, line)
				}
			}
		}
	}
}

func TestPageMovementUsesRenderedViewportAndKeepsSelectionVisible(t *testing.T) {
	top := newModel("", pageTestDocument(), diskVersion{})
	top.width, top.height = 100, 7 // four body rows including the header
	top = top.ensureSelectionVisible()
	top = pressKey(top, "pgdown")
	if top.topCursor != 3 {
		t.Fatalf("Top PgDown cursor = %d, want viewport-derived 3", top.topCursor)
	}
	_, start, end := top.renderedTopBody()
	if start < top.topScroll || end > top.topScroll+top.viewportBudget() {
		t.Fatalf("Top paged selection not visible: cursor=%d scroll=%d bounds=%d:%d", top.topCursor, top.topScroll, start, end)
	}
	top = pressKey(top, "pgup")
	if top.topCursor != 0 {
		t.Fatalf("Top PgUp cursor = %d", top.topCursor)
	}

	ledger := newModel("", testUIDocument(), diskVersion{})
	ledger.width, ledger.height = 110, 7
	ledger = ledger.openLedger("L2").ensureSelectionVisible()
	ledger = pressKey(ledger, "pgdown")
	if ledger.ledgerCursor != 3 {
		t.Fatalf("Ledger PgDown cursor = %d, want viewport-derived 3", ledger.ledgerCursor)
	}

	detail := newModel("", testUIDocument(), diskVersion{})
	detail.width, detail.height = 56, 8
	detail = detail.openDetail("L2").ensureSelectionVisible()
	detail = pressKey(detail, "pgdown")
	if detail.detailCursor == 0 {
		t.Fatal("Detail PgDown made no progress")
	}
	_, start, end = detail.renderedDetailBody()
	if start < detail.detailScroll || end > detail.detailScroll+detail.viewportBudget() {
		t.Fatalf("Detail paged selection not visible: cursor=%d scroll=%d bounds=%d:%d", detail.detailCursor, detail.detailScroll, start, end)
	}
}

func TestDetailScopesChallengesAndSupportsNavigationStack(t *testing.T) {
	m := newModel("", testUIDocument(), diskVersion{})
	m.width, m.height = 100, 32
	m = m.openDetail("L2")
	view := m.View()
	for _, want := range []string{"STATEMENT DETAIL", "L2!", "lemma[fact]  T", "JUSTIFICATIONS", "1 — AND", "L1   T", "P4   T", "UNDERCUTS", "CP3  T", "P2   T"} {
		if !strings.Contains(view, want) {
			t.Fatalf("detail view missing %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "derived") {
		t.Fatalf("detail view exposes repetitive provenance:\n%s", view)
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

func TestCounterpointDetailShowsGroundedAcceptance(t *testing.T) {
	m := newModel("", testUIDocument(), diskVersion{}).openDetail("CP1")
	if view := m.View(); !strings.Contains(view, "counterpoint[fact]  T  OUT") {
		t.Fatalf("counterpoint acceptance missing:\n%s", view)
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
	for _, want := range []string{"LABEL", "TRUTH", "STATEMENT", "DERIVATION", "AND(P1, P2)", "OR(P3, P4)", "AND(P2) [direct]", "without being summarized or omitted"} {
		if !strings.Contains(flat, want) {
			t.Fatalf("ledger view missing %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "J1") || strings.Contains(view, "derived") || strings.Contains(view, "justified by") || strings.Contains(view, "...") {
		t.Fatalf("ledger view contains forbidden notation:\n%s", view)
	}
	m = pressKey(m, "enter")
	if m.mode != modeDetail || m.current != "P1" {
		t.Fatalf("ledger enter = %s", m.debugState())
	}
	m = pressKey(m, "esc")
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
	m = pressKey(m, "esc")
	if m.mode != modeLedger || m.ledgerCursor != wantCursor || m.ledgerScroll != wantScroll {
		t.Fatalf("ledger viewport not restored: cursor=%d/%d scroll=%d/%d", m.ledgerCursor, wantCursor, m.ledgerScroll, wantScroll)
	}
}

func TestTopKeyReturnsDirectlyToTopAndClearsHistory(t *testing.T) {
	m := newModel("", testUIDocument(), diskVersion{})
	m = m.openLedger("L2").ensureSelectionVisible()
	m = pressKey(m, "enter")
	if m.mode != modeDetail || len(m.history) < 2 {
		t.Fatalf("nested navigation precondition failed: %s history=%d", m.debugState(), len(m.history))
	}
	m = pressKey(m, "t")
	if m.mode != modeTop || m.current != "" || len(m.history) != 0 {
		t.Fatalf("t did not return directly to Top: %s history=%d", m.debugState(), len(m.history))
	}
	m = pressKey(m, "esc")
	if m.mode != modeTop {
		t.Fatalf("Top retained a hidden back destination: %s", m.debugState())
	}
}

func TestSpatialAliasesWorkWithoutFooterAdvertising(t *testing.T) {
	m := newModel("", testUIDocument(), diskVersion{})
	m = pressKey(m, "l")
	if m.mode != modeDetail {
		t.Fatalf("l did not follow from Top: %s", m.debugState())
	}
	m = pressKey(m, "h")
	if m.mode != modeTop {
		t.Fatalf("h did not go back to Top: %s", m.debugState())
	}
	m = m.openLedger("L2").ensureSelectionVisible()
	view := m.View()
	if strings.Contains(view, "h/Esc") || strings.Contains(view, "Enter/l") || !strings.Contains(view, "Esc back  t Top") {
		t.Fatalf("ledger footer advertises hidden aliases or lacks visible guidance:\n%s", view)
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

func TestTopCapitalKeysPersistOrderAndKeepSelection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workspace.arg")
	doc := testUIDocument()
	if err := argfile.SaveAtomic(path, doc); err != nil {
		t.Fatal(err)
	}
	m := newModel(path, doc, readDisk(path).version)

	updated, cmd := updateWithKey(m, "J")
	if cmd == nil || !updated.topMovePending {
		t.Fatalf("J did not start reorder: pending=%t cmd=%v", updated.topMovePending, cmd)
	}
	ignored, ignoredCmd := updateWithKey(updated, "K")
	if ignoredCmd != nil || ignored.selectedTopID() != "L2" {
		t.Fatalf("pending reorder accepted another move: cmd=%v selected=%s", ignoredCmd, ignored.selectedTopID())
	}
	m = applyTeaCmd(ignored, cmd)
	if m.topMovePending || m.selectedTopID() != "L2" || !strings.Contains(m.message, "moved L2 after P5") {
		t.Fatalf("completed reorder state=%s pending=%t message=%q", m.debugState(), m.topMovePending, m.message)
	}
	if got := []string{m.topItems[0].Statement.ID, m.topItems[1].Statement.ID}; !reflect.DeepEqual(got, []string{"P5", "L2"}) {
		t.Fatalf("Top order = %v", got)
	}
	parsed := argfile.Load(path)
	if got := []string{query.Top(parsed.Document)[0].Statement.ID, query.Top(parsed.Document)[1].Statement.ID}; !reflect.DeepEqual(got, []string{"P5", "L2"}) {
		t.Fatalf("persisted Top order = %v", got)
	}

	beforeNavigation, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	m = pressKey(m, "k")
	if m.selectedTopID() != "P5" {
		t.Fatalf("lowercase k did not navigate: %s", m.selectedTopID())
	}
	afterNavigation, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(beforeNavigation, afterNavigation) {
		t.Fatalf("lowercase navigation wrote workspace: %v", err)
	}

	restarted, _, err := loadDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := query.Top(restarted); got[0].Statement.ID != "P5" || got[1].Statement.ID != "L2" {
		t.Fatalf("restart lost Top order: %#v", got)
	}
}

func TestTopReorderBoundariesDoNotWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workspace.arg")
	doc := testUIDocument()
	if err := argfile.SaveAtomic(path, doc); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	m := newModel(path, doc, readDisk(path).version)
	updated, cmd := updateWithKey(m, "K")
	if cmd != nil || updated.topMovePending {
		t.Fatalf("upper boundary started reorder: pending=%t cmd=%v", updated.topMovePending, cmd)
	}
	m.topCursor = len(m.topItems) - 1
	updated, cmd = updateWithKey(m, "J")
	if cmd != nil || updated.topMovePending {
		t.Fatalf("lower boundary started reorder: pending=%t cmd=%v", updated.topMovePending, cmd)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("boundary reorder rewrote workspace")
	}
}

func TestTopReorderRefreshesOnExternalOrderChange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workspace.arg")
	doc := testUIDocument()
	if err := argfile.SaveAtomic(path, doc); err != nil {
		t.Fatal(err)
	}
	m := newModel(path, doc, readDisk(path).version)
	pending, cmd := updateWithKey(m, "J")
	if cmd == nil {
		t.Fatal("J did not return reorder command")
	}
	external, _, err := argument.MoveStatement(doc, "P5", "L2", argument.MoveBefore)
	if err != nil {
		t.Fatal(err)
	}
	if err := argfile.SaveAtomic(path, external); err != nil {
		t.Fatal(err)
	}
	m = applyTeaCmd(pending, cmd)
	if !strings.Contains(m.message, "changed externally") || m.topItems[0].Statement.ID != "P5" || m.selectedTopID() != "L2" {
		t.Fatalf("stale reorder did not refresh safely: order=%s,%s selected=%s message=%q", m.topItems[0].Statement.ID, m.topItems[1].Statement.ID, m.selectedTopID(), m.message)
	}
}

func TestTopReorderRetainsLastValidDocumentWhenDiskBecomesInvalid(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workspace.arg")
	doc := testUIDocument()
	if err := argfile.SaveAtomic(path, doc); err != nil {
		t.Fatal(err)
	}
	m := newModel(path, doc, readDisk(path).version)
	pending, cmd := updateWithKey(m, "J")
	if err := os.WriteFile(path, []byte("invalid"), 0o600); err != nil {
		t.Fatal(err)
	}
	m = applyTeaCmd(pending, cmd)
	if !strings.Contains(m.message, "reorder failed") || m.topItems[0].Statement.ID != "L2" {
		t.Fatalf("invalid reorder replaced last valid state: first=%s message=%q", m.topItems[0].Statement.ID, m.message)
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

func TestNonMutationNavigationDoesNotWriteFile(t *testing.T) {
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
		t.Fatalf("navigation wrote workspace: %v", err)
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
	if view := m.View(); !strings.Contains(view, "TOP · 0 of 0") || !strings.Contains(view, "no top statements") {
		t.Fatalf("empty top view:\n%s", view)
	}
}

func pressKey(m Model, key string) Model {
	updated, _ := updateWithKey(m, key)
	return updated
}

func updateWithKey(m Model, key string) (Model, tea.Cmd) {
	var msg tea.KeyMsg
	switch key {
	case "enter":
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	default:
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
	updated, cmd := m.Update(msg)
	return updated.(Model), cmd
}

func applyTeaCmd(m Model, cmd tea.Cmd) Model {
	updated, _ := m.Update(cmd())
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

func pageTestDocument() *argument.Document {
	doc := &argument.Document{ID: "pages", Title: "Pages", Metadata: []argument.Metadata{{Key: "profile", Value: "workspace"}}}
	for index := 1; index <= 12; index++ {
		doc.Statements = append(doc.Statements, argument.Statement{
			ID: fmt.Sprintf("P%d", index), Role: argument.RolePremise, Kind: argument.KindFact,
			Truth: argument.TruthTrue, Text: fmt.Sprintf("Statement %d", index),
		})
	}
	return doc
}
