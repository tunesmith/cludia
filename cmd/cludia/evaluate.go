// SPDX-FileCopyrightText: 2026 KeenWorks
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/tunesmith/cludia/internal/diagnostic"
	"github.com/tunesmith/cludia/internal/evaluation"
	"github.com/tunesmith/cludia/internal/presentation"
	"github.com/tunesmith/cludia/internal/validation"
)

type evaluateOutput struct {
	SchemaVersion int                     `json:"schema_version"`
	Profile       validation.Profile      `json:"profile"`
	Document      documentOutput          `json:"document"`
	Evaluation    evaluation.Result       `json:"evaluation"`
	Diagnostics   []diagnostic.Diagnostic `json:"diagnostics"`
}

func runEvaluate(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("evaluate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOutput := fs.Bool("json", false, "output versioned JSON")
	fs.Usage = func() { writeEvaluateUsage(fs.Output()) }
	if err := fs.Parse(flagsFirst(args, map[string]bool{})); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("evaluate expects exactly one workspace file")
	}
	doc, profile, diagnostics := loadValidated(fs.Arg(0))
	if diagnostic.HasErrors(diagnostics) {
		if err := writeFailure(stdout, *jsonOutput, profile, diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	result, evaluationDiagnostics := evaluateDocument(doc)
	if diagnostic.HasErrors(evaluationDiagnostics) {
		if err := writeFailure(stdout, *jsonOutput, profile, evaluationDiagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	output := evaluateOutput{
		SchemaVersion: outputSchemaVersion, Profile: profile, Document: documentSummary(doc),
		Evaluation: result, Diagnostics: nonNilDiagnostics(diagnostics),
	}
	if *jsonOutput {
		return writeIndentedJSON(stdout, output)
	}
	fmt.Fprintf(stdout, "Evaluation v%d · %s\n", result.SchemaVersion, result.Mode)
	for _, value := range result.Statements {
		statement, _ := doc.Statement(value.ID)
		status := presentation.EffectiveStatementStatus(statement.Role, value.EffectiveTruth)
		fmt.Fprintf(stdout, "%s\tstored %s\t%s %s\t%s", value.ID, value.StoredTruth, presentation.StatusNoun(statement.Role), status, value.TruthSource)
		if value.Acceptance != "" {
			fmt.Fprintf(stdout, "\t%s", value.Acceptance)
		}
		fmt.Fprintln(stdout)
	}
	fmt.Fprintln(stdout, "Junctors:")
	for _, junctor := range result.Junctors {
		fmt.Fprintf(stdout, "%s\teffective %s\n", junctor.ID, junctor.EffectiveTruth)
	}
	return nil
}

func writeEvaluateUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cludia evaluate [--json] FILE")
	fmt.Fprintln(w, "Calculate versioned grounded leaf truth, derived provability, counterpoint acceptance, and active defeat effects without modifying the file.")
}
