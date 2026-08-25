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
	sameProposition := fs.Bool("same-proposition", false, "assert that replacement text expresses the same proposition")
	var text, truthName, kindName optionalStringFlag
	fs.Var(&text, "text", "replacement statement text")
	fs.Var(&truthName, "truth", "replacement truth: T, F, or U")
	fs.Var(&kindName, "kind", "replacement kind: fact or value")
	fs.Usage = func() { writeEditUsage(fs.Output()) }
	if err := fs.Parse(flagsFirst(args, map[string]bool{"text": true, "truth": true, "kind": true})); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 2 || !text.set && !truthName.set && !kindName.set {
		fs.Usage()
		return fmt.Errorf("edit expects a file and statement and at least one of --text, --truth, or --kind")
	}
	if text.set && strings.TrimSpace(text.value) == "" {
		return fmt.Errorf("--text must not be empty")
	}
	if text.set && !*sameProposition {
		return fmt.Errorf("--same-proposition is required with --text; run 'cludia guidance' for the statement identity contract")
	}
	if !text.set && *sameProposition {
		return fmt.Errorf("--same-proposition applies only with --text")
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
	if text.set {
		edited.Text = strings.TrimSpace(text.value)
	}
	if truthName.set {
		truth, ok := parseTruth(truthName.value)
		if !ok {
			diagnostics := diagnosticError("truth_invalid", fmt.Sprintf("invalid truth %q; expected T, F, or U", truthName.value), edited.ID)
			if err := writeFailure(stdout, *jsonOutput, profile, diagnostics); err != nil {
				return err
			}
			return errValidationFailed
		}
		edited.Truth = truth
	}
	if kindName.set {
		kind, ok := parseKind(kindName.value)
		if !ok {
			diagnostics := diagnosticError("kind_invalid", fmt.Sprintf("invalid kind %q; expected fact or value", kindName.value), edited.ID)
			if err := writeFailure(stdout, *jsonOutput, profile, diagnostics); err != nil {
				return err
			}
			return errValidationFailed
		}
		edited.Kind = kind
	}
	validated := validation.Validate(next, profile)
	if !validated.OK() {
		if err := writeFailure(stdout, *jsonOutput, profile, validated.Diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	changed := *edited != previous
	if changed {
		if err := argfile.SaveAtomic(fs.Arg(0), next); err != nil {
			return err
		}
	}
	diagnostics = validated.Diagnostics
	if diagnostics == nil {
		diagnostics = []diagnostic.Diagnostic{}
	}
	changes := []changeOutput{}
	if changed {
		changes = append(changes, changeOutput{Operation: "updated", ElementType: "statement", ID: edited.ID})
	}
	output := mutationOutput{
		SchemaVersion: outputSchemaVersion, Action: "edit", DryRun: false,
		Profile: profile, Document: documentSummary(next), Statement: *edited,
		PreviousStatement: &previous, Changes: changes, Diagnostics: diagnostics,
	}
	if text.set {
		asserted := true
		output.SameProposition = &asserted
	}
	return writeMutation(stdout, *jsonOutput, output)
}

func writeEditUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cludia edit [--json] FILE STATEMENT [--text TEXT --same-proposition] [--truth T|F|U] [--kind fact|value]")
	fmt.Fprintln(w, "Change statement text, truth, or kind without changing its stable id or slug.")
}
