package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tunesmith/cludia/internal/argument"
	"github.com/tunesmith/cludia/internal/query"
	"github.com/tunesmith/cludia/internal/validation"
	"github.com/tunesmith/cludia/internal/workspace"
)

type topMoveResultMsg struct {
	doc     *argument.Document
	version diskVersion
	move    argument.StatementMove
	err     error
}

func (m Model) beginTopMove(delta int) (Model, tea.Cmd) {
	if m.topMovePending || len(m.topItems) < 2 {
		return m, nil
	}
	selectedIndex := clampCursor(m.topCursor, len(m.topItems))
	anchorIndex := selectedIndex + delta
	if anchorIndex < 0 || anchorIndex >= len(m.topItems) {
		return m, nil
	}
	placement := argument.MoveBefore
	if delta > 0 {
		placement = argument.MoveAfter
	}
	statementID := m.topItems[selectedIndex].Statement.ID
	anchorID := m.topItems[anchorIndex].Statement.ID
	m.topMovePending = true
	m.setMessage("reordering "+statementID+"…", messageNone)
	return m, moveTopStatement(m.path, statementID, anchorID, placement)
}

func moveTopStatement(path, statementID, anchorID string, placement argument.MovePlacement) tea.Cmd {
	return func() tea.Msg {
		check := readDisk(path)
		if check.err != nil {
			return topMoveResultMsg{err: check.err}
		}
		if !check.version.exists {
			return topMoveResultMsg{err: fmt.Errorf("workspace no longer exists")}
		}
		doc, err := parseValidDocument(check.data)
		if err != nil {
			return topMoveResultMsg{err: err}
		}
		if !adjacentTopItems(doc, statementID, anchorID, placement) {
			return topMoveResultMsg{
				doc: doc, version: check.version,
				err: fmt.Errorf("Top order changed externally; review the refreshed list and retry"),
			}
		}
		next, move, err := argument.MoveStatement(doc, statementID, anchorID, placement)
		if err != nil {
			return topMoveResultMsg{doc: doc, version: check.version, err: err}
		}
		validated, err := workspace.ValidateAndPersist(path, next, validation.ProfileWorkspace, move.Changed)
		if err != nil {
			return topMoveResultMsg{doc: doc, version: check.version, err: err}
		}
		if !validated.OK() {
			return topMoveResultMsg{doc: doc, version: check.version, err: fmt.Errorf("reordered workspace is invalid: %v", validated.Diagnostics)}
		}
		saved := readDisk(path)
		if saved.err != nil || !saved.version.exists {
			if saved.err != nil {
				err = saved.err
			} else {
				err = fmt.Errorf("workspace disappeared after reorder")
			}
			return topMoveResultMsg{err: err}
		}
		savedDoc, err := parseValidDocument(saved.data)
		if err != nil {
			return topMoveResultMsg{err: fmt.Errorf("saved workspace could not be reloaded: %w", err)}
		}
		return topMoveResultMsg{doc: savedDoc, version: saved.version, move: move}
	}
}

func adjacentTopItems(doc *argument.Document, statementID, anchorID string, placement argument.MovePlacement) bool {
	items := query.Top(doc)
	statementIndex, anchorIndex := -1, -1
	for i, item := range items {
		switch item.Statement.ID {
		case statementID:
			statementIndex = i
		case anchorID:
			anchorIndex = i
		}
	}
	if placement == argument.MoveBefore {
		return anchorIndex >= 0 && statementIndex == anchorIndex+1
	}
	if placement == argument.MoveAfter {
		return statementIndex >= 0 && anchorIndex == statementIndex+1
	}
	return false
}

func (m Model) applyTopMoveResult(result topMoveResultMsg) Model {
	m.topMovePending = false
	preferredTop := m.selectedTopID()
	if result.move.Statement.ID != "" {
		preferredTop = result.move.Statement.ID
	}
	if result.doc != nil {
		m.doc = result.doc
		m.diskVersion, m.seenDiskVersion, m.diskVersionKnown = result.version, result.version, true
		m.refreshQueries(preferredTop)
		m = m.refreshOpenViewAfterMove()
	}
	if result.err != nil {
		m.setMessage("reorder failed: "+result.err.Error(), messageError)
		return m
	}
	m.setMessage(fmt.Sprintf("moved %s %s %s", result.move.Statement.ID, result.move.Placement, result.move.Anchor.ID), messageSuccess)
	return m
}

func (m Model) refreshOpenViewAfterMove() Model {
	if m.mode == modeDetail {
		if _, ok := m.doc.Statement(m.current); !ok {
			m.mode, m.current, m.history = modeTop, "", nil
			return m
		}
	}
	if m.mode == modeLedger {
		root, rows, err := query.Ledger(m.doc, m.ledgerRoot)
		if err != nil {
			m.mode, m.current, m.history = modeTop, "", nil
			return m
		}
		m.ledgerRoot, m.ledgerRows = root, rows
	}
	m.ensureSelections()
	return m
}
