// SPDX-FileCopyrightText: 2026 KeenWorks
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/tunesmith/cludia/internal/argument"
	"github.com/tunesmith/cludia/internal/diagnostic"
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
	options := argument.EditStatementOptions{Reference: fs.Arg(1)}
	if text.set {
		value := strings.TrimSpace(text.value)
		options.Text = &value
	}
	if truthName.set {
		truth, _ := parseTruth(truthName.value)
		options.Truth = &truth
	}
	if kindName.set {
		kind, _ := parseKind(kindName.value)
		options.Kind = &kind
	}
	next, result, err := argument.EditStatement(doc, options)
	if err != nil {
		return writeArgumentMutationFailure(stdout, *jsonOutput, profile, err)
	}
	validated, err := validateAndPersistMutation(fs.Arg(0), next, profile, result.Changed)
	if err != nil {
		return err
	}
	if !validated.OK() {
		if err := writeFailure(stdout, *jsonOutput, profile, validated.Diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	diagnostics = validated.Diagnostics
	if diagnostics == nil {
		diagnostics = []diagnostic.Diagnostic{}
	}
	changes := []changeOutput{}
	if result.Changed {
		changes = append(changes, changeOutput{Operation: "updated", ElementType: "statement", ID: result.Current.ID})
		changes = appendProfileMigrationChange(changes, doc)
	}
	output := mutationOutput{
		SchemaVersion: outputSchemaVersion, Action: "edit", DryRun: false,
		Profile: profile, Document: documentSummary(next), Statement: result.Current,
		PreviousStatement: &result.Previous, Changes: changes, Diagnostics: diagnostics,
	}
	if text.set {
		asserted := true
		output.SameProposition = &asserted
	}
	return writeMutation(stdout, *jsonOutput, output)
}

func writeEditUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cludia edit [--json] FILE STATEMENT [--text TEXT --same-proposition] [--truth T|F|U] [--kind fact|value]")
	fmt.Fprintln(w, "Change statement text or kind without changing its stable id or slug; truth may be assigned only to leaf premises and leaf counterpoints.")
}
