package ui

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tunesmith/cludia/internal/argument"
	"github.com/tunesmith/cludia/internal/diagnostic"
	"github.com/tunesmith/cludia/internal/query"
	"github.com/tunesmith/cludia/internal/validation"
	"github.com/tunesmith/cludia/internal/workspace"
)

const diskCheckInterval = 500 * time.Millisecond

type diskVersion struct {
	exists bool
	digest [sha256.Size]byte
}

type diskContents struct {
	data    []byte
	version diskVersion
	err     error
}

type diskCheckMsg struct{}

func readDisk(path string) diskContents {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return diskContents{version: diskVersion{}}
	}
	if err != nil {
		return diskContents{err: err}
	}
	return diskContents{data: data, version: diskVersion{exists: true, digest: sha256.Sum256(data)}}
}

func loadDocument(path string) (*argument.Document, diskVersion, error) {
	check := readDisk(path)
	if check.err != nil {
		return nil, diskVersion{}, check.err
	}
	if !check.version.exists {
		return nil, diskVersion{}, fmt.Errorf("workspace does not exist: %s", path)
	}
	doc, err := parseValidDocument(check.data)
	if err != nil {
		return nil, diskVersion{}, err
	}
	return doc, check.version, nil
}

func parseValidDocument(data []byte) (*argument.Document, error) {
	doc, diagnostics := workspace.ParseValidated(data, validation.ProfileWorkspace)
	if diagnostic.HasErrors(diagnostics) {
		messages := make([]string, 0)
		for _, item := range diagnostics {
			if item.Severity == diagnostic.SeverityError {
				messages = append(messages, item.Code+": "+item.Message)
			}
		}
		return nil, fmt.Errorf("invalid workspace: %s", strings.Join(messages, "; "))
	}
	return doc, nil
}

func scheduleDiskCheck() tea.Cmd {
	return tea.Tick(diskCheckInterval, func(time.Time) tea.Msg { return diskCheckMsg{} })
}

func (m Model) refreshFromDisk(check diskContents) Model {
	if check.err != nil {
		m.setMessage("reload failed: "+check.err.Error(), messageError)
		return m.ensureSelectionVisible()
	}
	if m.diskVersionKnown && check.version == m.seenDiskVersion {
		return m
	}
	m.seenDiskVersion = check.version
	if m.diskVersionKnown && check.version == m.diskVersion {
		m.setMessage("workspace restored to last valid contents", messageSuccess)
		return m.ensureSelectionVisible()
	}
	if !check.version.exists {
		m.setMessage("reload failed: workspace no longer exists", messageError)
		return m.ensureSelectionVisible()
	}
	doc, err := parseValidDocument(check.data)
	if err != nil {
		m.setMessage("external change is invalid: "+err.Error(), messageError)
		return m.ensureSelectionVisible()
	}
	preferredTop := m.selectedTopID()
	current := m.current
	ledgerRoot := m.ledgerRoot
	m.doc = doc
	m.diskVersion, m.seenDiskVersion, m.diskVersionKnown = check.version, check.version, true
	m.refreshQueries(preferredTop)
	if m.mode == modeDetail {
		if _, ok := m.doc.Statement(current); ok {
			m.current = current
		} else {
			m.mode, m.history = modeTop, nil
			m.setMessage("selected statement was removed; returned to Top", messageError)
			return m.ensureSelectionVisible()
		}
	}
	if m.mode == modeLedger {
		root, rows, ledgerErr := queryLedger(m.doc, ledgerRoot)
		if ledgerErr != nil {
			m.mode, m.history = modeTop, nil
			m.setMessage("ledger root changed; returned to Top", messageError)
			return m.ensureSelectionVisible()
		}
		m.ledgerRoot, m.ledgerRows = root, rows
	}
	m.ensureSelections()
	m.setMessage("reloaded changes from disk", messageSuccess)
	return m.ensureSelectionVisible()
}

func queryLedger(doc *argument.Document, root string) (string, []query.LedgerRow, error) {
	return query.Ledger(doc, root)
}
