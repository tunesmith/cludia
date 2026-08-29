package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/tunesmith/cludia/internal/diagnostic"
	"github.com/tunesmith/cludia/internal/query"
	"github.com/tunesmith/cludia/internal/validation"
)

type topOutput struct {
	SchemaVersion int                     `json:"schema_version"`
	Profile       validation.Profile      `json:"profile"`
	Document      documentOutput          `json:"document"`
	Statements    []query.TopItem         `json:"statements"`
	Diagnostics   []diagnostic.Diagnostic `json:"diagnostics"`
}

type ledgerOutput struct {
	SchemaVersion int                     `json:"schema_version"`
	Profile       validation.Profile      `json:"profile"`
	Document      documentOutput          `json:"document"`
	Root          string                  `json:"root"`
	Rows          []query.LedgerRow       `json:"rows"`
	Diagnostics   []diagnostic.Diagnostic `json:"diagnostics"`
}

func runTop(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("top", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOutput := fs.Bool("json", false, "output versioned JSON")
	challengedOnly := fs.Bool("challenged", false, "show only challenged top statements")
	limit := fs.Int("limit", 0, "maximum number of statements to return (0 means all)")
	offset := fs.Int("offset", 0, "number of matching statements to skip")
	fs.Usage = func() { writeTopUsage(fs.Output()) }
	if err := fs.Parse(flagsFirst(args, map[string]bool{"limit": true, "offset": true})); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("top expects exactly one file")
	}
	if *limit < 0 {
		return fmt.Errorf("top --limit must be zero or greater")
	}
	if *offset < 0 {
		return fmt.Errorf("top --offset must be zero or greater")
	}
	doc, profile, diagnostics := loadValidated(fs.Arg(0))
	if diagnostic.HasErrors(diagnostics) {
		if err := writeFailure(stdout, *jsonOutput, profile, diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	items := query.Top(doc)
	if *challengedOnly {
		filtered := make([]query.TopItem, 0, len(items))
		for _, item := range items {
			if item.Challenged {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	start := minInt(*offset, len(items))
	end := len(items)
	if *limit > 0 && *limit < end-start {
		end = start + *limit
	}
	items = items[start:end]
	output := topOutput{
		SchemaVersion: outputSchemaVersion, Profile: profile, Document: documentSummary(doc),
		Statements: items, Diagnostics: nonNilDiagnostics(diagnostics),
	}
	if *jsonOutput {
		return writeIndentedJSON(stdout, output)
	}
	writeHumanTop(stdout, output, textViewWidth())
	return nil
}

func runLedger(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("ledger", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOutput := fs.Bool("json", false, "output versioned JSON")
	fs.Usage = func() { writeLedgerUsage(fs.Output()) }
	if err := fs.Parse(flagsFirst(args, map[string]bool{})); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 2 {
		fs.Usage()
		return fmt.Errorf("ledger expects one file and a statement id or slug")
	}
	doc, profile, diagnostics := loadValidated(fs.Arg(0))
	if diagnostic.HasErrors(diagnostics) {
		if err := writeFailure(stdout, *jsonOutput, profile, diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	root, rows, err := query.Ledger(doc, fs.Arg(1))
	if err != nil {
		return writeMutationFailure(stdout, *jsonOutput, profile, "ledger_root_invalid", err.Error(), fs.Arg(1))
	}
	output := ledgerOutput{
		SchemaVersion: outputSchemaVersion, Profile: profile, Document: documentSummary(doc),
		Root: root, Rows: rows, Diagnostics: nonNilDiagnostics(diagnostics),
	}
	if *jsonOutput {
		return writeIndentedJSON(stdout, output)
	}
	writeHumanLedger(stdout, output, textViewWidth())
	return nil
}

func nonNilDiagnostics(items []diagnostic.Diagnostic) []diagnostic.Diagnostic {
	if items == nil {
		return []diagnostic.Diagnostic{}
	}
	return items
}

func writeHumanTop(w io.Writer, output topOutput, width int) {
	if width < 80 {
		for _, item := range output.Statements {
			label := displayLabel(item.Statement.ID, item.Challenged)
			depth := ""
			if item.Depth > 0 {
				depth = fmt.Sprintf("  depth %d", item.Depth)
			}
			fmt.Fprintf(w, "%s%s\n%s\n\n", label, depth, item.Statement.Text)
		}
		return
	}
	labelWidth := maxDisplayLabelWidthTop(output.Statements, len("LABEL"))
	depthWidth := len("DEPTH")
	statementWidth := maxInt(24, width-labelWidth-depthWidth-4)
	fmt.Fprintf(w, "%s  %s  %s\n", padText("LABEL", labelWidth), padText("DEPTH", depthWidth), "STATEMENT")
	for _, item := range output.Statements {
		lines := wrapFullText(item.Statement.Text, statementWidth)
		depth := ""
		if item.Depth > 0 {
			depth = strconv.Itoa(item.Depth)
		}
		for i, line := range lines {
			labelCell, depthCell := "", ""
			if i == 0 {
				labelCell = displayLabel(item.Statement.ID, item.Challenged)
				depthCell = depth
			}
			fmt.Fprintf(w, "%s  %s  %s\n", padText(labelCell, labelWidth), padText(depthCell, depthWidth), line)
		}
	}
}

func writeHumanLedger(w io.Writer, output ledgerOutput, width int) {
	if width < 80 {
		for _, row := range output.Rows {
			fmt.Fprintln(w, displayLabel(row.Statement.ID, row.Challenged))
			fmt.Fprintln(w, row.Statement.Text)
			for _, derivation := range ledgerDerivations(row) {
				fmt.Fprintln(w, derivation)
			}
			fmt.Fprintln(w)
		}
		return
	}
	labelWidth := maxDisplayLabelWidthLedger(output.Rows, len("LABEL"))
	derivationWidth := minInt(34, maxInt(20, width/4))
	statementWidth := maxInt(30, width-labelWidth-derivationWidth-4)
	fmt.Fprintf(w, "%s  %s  %s\n", padText("LABEL", labelWidth), padText("STATEMENT", statementWidth), "DERIVATION")
	for _, row := range output.Rows {
		statementLines := wrapFullText(row.Statement.Text, statementWidth)
		derivationLines := make([]string, 0)
		for _, value := range ledgerDerivations(row) {
			derivationLines = append(derivationLines, wrapFullText(value, derivationWidth)...)
		}
		lineCount := maxInt(len(statementLines), len(derivationLines))
		if lineCount == 0 {
			lineCount = 1
		}
		for i := 0; i < lineCount; i++ {
			labelCell := ""
			if i == 0 {
				labelCell = displayLabel(row.Statement.ID, row.Challenged)
			}
			statementCell, derivationCell := "", ""
			if i < len(statementLines) {
				statementCell = statementLines[i]
			}
			if i < len(derivationLines) {
				derivationCell = derivationLines[i]
			}
			fmt.Fprintf(w, "%s  %s  %s\n", padText(labelCell, labelWidth), padText(statementCell, statementWidth), derivationCell)
		}
	}
}

func ledgerDerivations(row query.LedgerRow) []string {
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

func displayLabel(id string, challenged bool) string {
	if challenged {
		return id + "!"
	}
	return id
}

func textViewWidth() int {
	if value, err := strconv.Atoi(strings.TrimSpace(os.Getenv("COLUMNS"))); err == nil && value > 0 {
		return value
	}
	return 120
}

func wrapFullText(text string, width int) []string {
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
			if runeCount(candidate) > width {
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

func runeCount(value string) int { return utf8.RuneCountInString(value) }

func padText(value string, width int) string {
	if missing := width - runeCount(value); missing > 0 {
		return value + strings.Repeat(" ", missing)
	}
	return value
}

func maxDisplayLabelWidthTop(items []query.TopItem, minimum int) int {
	result := minimum
	for _, item := range items {
		result = maxInt(result, runeCount(displayLabel(item.Statement.ID, item.Challenged)))
	}
	return result
}

func maxDisplayLabelWidthLedger(items []query.LedgerRow, minimum int) int {
	result := minimum
	for _, item := range items {
		result = maxInt(result, runeCount(displayLabel(item.Statement.ID, item.Challenged)))
	}
	return result
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

func writeTopUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cludia top [--challenged] [--limit N] [--offset N] [--json] FILE")
	fmt.Fprintln(w, "List non-counterpoint statements with no outgoing support, longest support depth, and challenge state.")
	fmt.Fprintln(w, "Filtering and pagination preserve document order; root remains the complete rooted-structure query.")
}

func writeLedgerUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cludia ledger [--json] FILE STATEMENT")
	fmt.Fprintln(w, "Show the complete support derivation to a statement in stable topological order.")
}
