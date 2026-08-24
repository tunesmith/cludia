package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/tunesmith/cludia/internal/argfile"
	"github.com/tunesmith/cludia/internal/diagnostic"
	"github.com/tunesmith/cludia/internal/validation"
)

func runEdit(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("edit", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOutput := fs.Bool("json", false, "output versioned JSON")
	text := fs.String("text", "", "replacement statement text")
	fs.Usage = func() { writeEditUsage(fs.Output()) }
	if err := fs.Parse(flagsFirst(args, map[string]bool{"text": true})); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 2 || strings.TrimSpace(*text) == "" {
		fs.Usage()
		return fmt.Errorf("edit expects a file and statement and requires --text")
	}

	doc, profile, diagnostics := loadValidated(fs.Arg(0))
	if diagnostic.HasErrors(diagnostics) {
		if err := writeFailure(stdout, *jsonOutput, profile, diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	statement, ok := doc.Statement(fs.Arg(1))
	if !ok {
		diagnostics := diagnosticError("statement_not_found", fmt.Sprintf("statement %q not found", fs.Arg(1)), fs.Arg(1))
		if err := writeFailure(stdout, *jsonOutput, profile, diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}

	next := doc.Clone()
	edited, _ := next.Statement(statement.ID)
	previous := *edited
	edited.Text = strings.TrimSpace(*text)
	validated := validation.Validate(next, profile)
	if !validated.OK() {
		if err := writeFailure(stdout, *jsonOutput, profile, validated.Diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	if edited.Text != previous.Text {
		if err := argfile.SaveAtomic(fs.Arg(0), next); err != nil {
			return err
		}
	}
	diagnostics = validated.Diagnostics
	if diagnostics == nil {
		diagnostics = []diagnostic.Diagnostic{}
	}
	changes := []changeOutput{}
	if edited.Text != previous.Text {
		changes = append(changes, changeOutput{Operation: "updated", ElementType: "statement", ID: edited.ID})
	}
	output := mutationOutput{
		SchemaVersion: outputSchemaVersion, Action: "edit", DryRun: false,
		Profile: profile, Document: documentSummary(next), Statement: *edited,
		PreviousStatement: &previous, Changes: changes, Diagnostics: diagnostics,
	}
	return writeMutation(stdout, *jsonOutput, output)
}

func writeEditUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cludia edit [--json] FILE STATEMENT --text TEXT")
	fmt.Fprintln(w, "Replace statement text without changing its stable id or slug.")
}
