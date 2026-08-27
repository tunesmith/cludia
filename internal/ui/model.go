package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tunesmith/cludia/internal/argument"
	"github.com/tunesmith/cludia/internal/query"
)

type mode int

const (
	modeTop mode = iota
	modeDetail
	modeLedger
)

type screenState struct {
	mode         mode
	current      string
	topCursor    int
	detailCursor int
	ledgerCursor int
	ledgerRoot   string
}

// Model is the read-only Bubble Tea model.
type Model struct {
	path string
	doc  *argument.Document

	mode         mode
	current      string
	topItems     []query.TopItem
	topCursor    int
	detailCursor int
	ledgerRoot   string
	ledgerRows   []query.LedgerRow
	ledgerCursor int
	history      []screenState

	width   int
	height  int
	message string

	diskVersion      diskVersion
	seenDiskVersion  diskVersion
	diskVersionKnown bool
}

// Run opens a validated workspace in the terminal alternate screen.
func Run(path string) error {
	doc, version, err := loadDocument(path)
	if err != nil {
		return err
	}
	_, err = tea.NewProgram(newModel(path, doc, version), tea.WithAltScreen()).Run()
	return err
}

func newModel(path string, doc *argument.Document, version diskVersion) Model {
	m := Model{
		path: path, doc: doc, mode: modeTop, width: 100, height: 30,
		diskVersion: version, seenDiskVersion: version, diskVersionKnown: true,
	}
	m.refreshQueries("")
	return m
}

func (m Model) Init() tea.Cmd { return scheduleDiskCheck() }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case diskCheckMsg:
		m = m.refreshFromDisk(readDisk(m.path))
		return m, scheduleDiskCheck()
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		return m.updateKey(msg)
	}
	return m, nil
}

func (m Model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "ctrl+c" || key == "q" {
		return m, tea.Quit
	}
	switch m.mode {
	case modeTop:
		m = m.updateTop(key)
	case modeDetail:
		m = m.updateDetail(key)
	case modeLedger:
		m = m.updateLedger(key)
	}
	return m, nil
}

func (m Model) updateTop(key string) Model {
	switch key {
	case "j", "down":
		m.topCursor = moveCursor(m.topCursor, len(m.topItems), 1)
	case "k", "up":
		m.topCursor = moveCursor(m.topCursor, len(m.topItems), -1)
	case "pgdown":
		m.topCursor = moveCursor(m.topCursor, len(m.topItems), 5)
	case "pgup":
		m.topCursor = moveCursor(m.topCursor, len(m.topItems), -5)
	case "home":
		m.topCursor = moveCursor(0, len(m.topItems), 0)
	case "end":
		m.topCursor = moveCursor(len(m.topItems)-1, len(m.topItems), 0)
	case "enter", "l":
		if id := m.selectedTopID(); id != "" {
			m = m.openDetail(id)
		}
	case "f":
		if id := m.selectedTopID(); id != "" {
			m = m.openLedger(id)
		}
	}
	return m
}

func (m Model) updateDetail(key string) Model {
	ids := m.detailSelectableIDs()
	switch key {
	case "j", "down":
		m.detailCursor = moveCursor(m.detailCursor, len(ids), 1)
	case "k", "up":
		m.detailCursor = moveCursor(m.detailCursor, len(ids), -1)
	case "pgdown":
		m.detailCursor = moveCursor(m.detailCursor, len(ids), 5)
	case "pgup":
		m.detailCursor = moveCursor(m.detailCursor, len(ids), -5)
	case "home":
		m.detailCursor = moveCursor(0, len(ids), 0)
	case "end":
		m.detailCursor = moveCursor(len(ids)-1, len(ids), 0)
	case "enter", "l":
		if len(ids) > 0 {
			m = m.openDetail(ids[clampCursor(m.detailCursor, len(ids))])
		}
	case "f":
		if statement, ok := m.doc.Statement(m.current); ok && statement.Role != argument.RoleCounterpoint {
			m = m.openLedger(statement.ID)
		} else {
			m.message = "counterpoints cannot be ledger roots"
		}
	case "h", "esc":
		m = m.back()
	}
	return m
}

func (m Model) updateLedger(key string) Model {
	switch key {
	case "j", "down":
		m.ledgerCursor = moveCursor(m.ledgerCursor, len(m.ledgerRows), 1)
	case "k", "up":
		m.ledgerCursor = moveCursor(m.ledgerCursor, len(m.ledgerRows), -1)
	case "pgdown":
		m.ledgerCursor = moveCursor(m.ledgerCursor, len(m.ledgerRows), 5)
	case "pgup":
		m.ledgerCursor = moveCursor(m.ledgerCursor, len(m.ledgerRows), -5)
	case "home":
		m.ledgerCursor = moveCursor(0, len(m.ledgerRows), 0)
	case "end":
		m.ledgerCursor = moveCursor(len(m.ledgerRows)-1, len(m.ledgerRows), 0)
	case "enter", "l":
		if len(m.ledgerRows) > 0 {
			m = m.openDetail(m.ledgerRows[clampCursor(m.ledgerCursor, len(m.ledgerRows))].Statement.ID)
		}
	case "h", "esc":
		m = m.back()
	}
	return m
}

func (m Model) openDetail(id string) Model {
	if _, ok := m.doc.Statement(id); !ok {
		m.message = "statement missing: " + id
		return m
	}
	m.history = append(m.history, m.snapshot())
	m.mode, m.current, m.detailCursor = modeDetail, id, 0
	m.message = ""
	return m
}

func (m Model) openLedger(id string) Model {
	root, rows, err := query.Ledger(m.doc, id)
	if err != nil {
		m.message = err.Error()
		return m
	}
	m.history = append(m.history, m.snapshot())
	m.mode, m.ledgerRoot, m.ledgerRows, m.ledgerCursor = modeLedger, root, rows, 0
	m.message = ""
	return m
}

func (m Model) back() Model {
	if len(m.history) == 0 {
		m.mode = modeTop
		return m
	}
	state := m.history[len(m.history)-1]
	m.history = m.history[:len(m.history)-1]
	m.mode, m.current = state.mode, state.current
	m.topCursor, m.detailCursor, m.ledgerCursor = state.topCursor, state.detailCursor, state.ledgerCursor
	m.ledgerRoot = state.ledgerRoot
	if m.mode == modeLedger {
		root, rows, err := query.Ledger(m.doc, m.ledgerRoot)
		if err != nil {
			m.mode, m.current, m.history = modeTop, "", nil
			m.message = "prior ledger root was removed; returned to Top"
			m.ensureSelections()
			return m
		}
		m.ledgerRoot, m.ledgerRows = root, rows
	}
	if m.mode == modeDetail {
		if _, ok := m.doc.Statement(m.current); !ok {
			m.mode, m.current, m.history = modeTop, "", nil
			m.message = "prior statement was removed; returned to Top"
			m.ensureSelections()
			return m
		}
	}
	m.ensureSelections()
	return m
}

func (m Model) snapshot() screenState {
	return screenState{
		mode: m.mode, current: m.current, topCursor: m.topCursor,
		detailCursor: m.detailCursor, ledgerCursor: m.ledgerCursor, ledgerRoot: m.ledgerRoot,
	}
}

func (m *Model) refreshQueries(preferredTopID string) {
	m.topItems = query.Top(m.doc)
	if preferredTopID != "" {
		for i, item := range m.topItems {
			if item.Statement.ID == preferredTopID {
				m.topCursor = i
				break
			}
		}
	}
	m.ensureSelections()
}

func (m *Model) ensureSelections() {
	m.topCursor = clampCursor(m.topCursor, len(m.topItems))
	m.detailCursor = clampCursor(m.detailCursor, len(m.detailSelectableIDs()))
	m.ledgerCursor = clampCursor(m.ledgerCursor, len(m.ledgerRows))
}

func (m Model) selectedTopID() string {
	if len(m.topItems) == 0 {
		return ""
	}
	return m.topItems[clampCursor(m.topCursor, len(m.topItems))].Statement.ID
}

func moveCursor(cursor, length, delta int) int {
	if length <= 0 {
		return 0
	}
	cursor += delta
	if cursor < 0 {
		return 0
	}
	if cursor >= length {
		return length - 1
	}
	return cursor
}

func clampCursor(cursor, length int) int {
	if length <= 0 || cursor < 0 {
		return 0
	}
	if cursor >= length {
		return length - 1
	}
	return cursor
}

func (m Model) debugState() string {
	return fmt.Sprintf("mode=%d current=%s top=%d detail=%d ledger=%d", m.mode, m.current, m.topCursor, m.detailCursor, m.ledgerCursor)
}
