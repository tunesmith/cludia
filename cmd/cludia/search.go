package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/tunesmith/cludia/internal/diagnostic"
	"github.com/tunesmith/cludia/internal/query"
	"github.com/tunesmith/cludia/internal/validation"
)

type searchMatchOutput struct {
	evaluatedStatement
	Isolated      bool     `json:"isolated"`
	MatchedFields []string `json:"matched_fields"`
}

type searchOutput struct {
	SchemaVersion int                     `json:"schema_version"`
	Profile       validation.Profile      `json:"profile"`
	Evaluation    evaluationMetadata      `json:"evaluation"`
	Query         string                  `json:"query"`
	Matches       []searchMatchOutput     `json:"matches"`
	Diagnostics   []diagnostic.Diagnostic `json:"diagnostics"`
}

func runSearch(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOutput := fs.Bool("json", false, "output versioned JSON")
	fs.Usage = func() { writeSearchUsage(fs.Output()) }
	if err := fs.Parse(flagsFirst(args, map[string]bool{})); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 2 || strings.TrimSpace(fs.Arg(1)) == "" {
		fs.Usage()
		return fmt.Errorf("search expects a file and non-empty query")
	}
	doc, profile, diagnostics := loadValidated(fs.Arg(0))
	if diagnostic.HasErrors(diagnostics) {
		if err := writeFailure(stdout, *jsonOutput, profile, diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	evaluated, evaluationDiagnostics := evaluateDocument(doc)
	if diagnostic.HasErrors(evaluationDiagnostics) {
		if err := writeFailure(stdout, *jsonOutput, profile, evaluationDiagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	search := fs.Arg(1)
	isolated := query.IsolatedStatementIDs(doc)
	output := searchOutput{
		SchemaVersion: outputSchemaVersion, Profile: profile, Evaluation: evaluationMeta(evaluated), Query: search,
		Matches: []searchMatchOutput{}, Diagnostics: diagnostics,
	}
	for _, match := range query.SearchStatements(doc, search) {
		output.Matches = append(output.Matches, searchMatchOutput{
			evaluatedStatement: evaluatedStatementFor(match.Statement, evaluated), Isolated: isolated[match.Statement.ID],
			MatchedFields: append([]string(nil), match.Fields...),
		})
	}
	if *jsonOutput {
		return writeIndentedJSON(stdout, output)
	}
	if len(output.Matches) == 0 {
		fmt.Fprintf(stdout, "No statements matched %q.\n", search)
	}
	for _, match := range output.Matches {
		fmt.Fprintf(stdout, "%s[%s]\t%s", match.Role, match.Kind, match.ID)
		if match.Slug != "" {
			fmt.Fprintf(stdout, ":%s", match.Slug)
		}
		fmt.Fprintf(stdout, "\t%s\t[%s]\t%s\n", formatTruthStatus(match.evaluatedStatement), strings.Join(match.MatchedFields, ","), match.Text)
	}
	for _, item := range output.Diagnostics {
		writeDiagnostic(stdout, item)
	}
	return nil
}

func writeSearchUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cludia search [--json] FILE QUERY")
	fmt.Fprintln(w, "Search statement IDs, slugs, and text by case-insensitive substring.")
}
