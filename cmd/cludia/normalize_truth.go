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
	"github.com/tunesmith/cludia/internal/validation"
)

type normalizeTruthOutput struct {
	SchemaVersion int                           `json:"schema_version"`
	Action        string                        `json:"action"`
	DryRun        bool                          `json:"dry_run"`
	Applicable    bool                          `json:"applicable"`
	Profile       validation.Profile            `json:"profile"`
	Document      documentOutput                `json:"document"`
	Statements    []argument.TruthNormalization `json:"statements"`
	PlanToken     string                        `json:"plan_token"`
	Changes       []changeOutput                `json:"changes"`
	Diagnostics   []diagnostic.Diagnostic       `json:"diagnostics"`
}

func runNormalizeTruth(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("normalize-truth", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOutput := fs.Bool("json", false, "output versioned JSON")
	dryRun := fs.Bool("dry-run", false, "build a state-bound normalization plan without saving")
	applyToken := fs.String("apply-token", "", "apply the exact reviewed normalization plan token")
	fs.Usage = func() { writeNormalizeTruthUsage(fs.Output()) }
	if err := fs.Parse(flagsFirst(args, map[string]bool{"apply-token": true})); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("normalize-truth expects exactly one workspace file")
	}
	applying := strings.TrimSpace(*applyToken) != ""
	if *dryRun == applying {
		fs.Usage()
		return fmt.Errorf("normalize-truth requires exactly one of --dry-run or --apply-token")
	}
	doc, profile, diagnostics := loadValidated(fs.Arg(0))
	if diagnostic.HasErrors(diagnostics) {
		if err := writeFailure(stdout, *jsonOutput, profile, diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	next, result, err := argument.NormalizeNonLeafTruth(doc)
	if err != nil {
		return writeArgumentMutationFailure(stdout, *jsonOutput, profile, err)
	}
	if applying && strings.TrimSpace(*applyToken) != result.PlanToken {
		return writeMutationFailure(stdout, *jsonOutput, profile, "normalize_truth_plan_stale", "normalization plan token does not match the current workspace; run --dry-run again", doc.ID)
	}
	validated, err := validateAndPersistMutation(fs.Arg(0), next, profile, applying && result.Changed)
	if err != nil {
		return err
	}
	if !validated.OK() {
		if err := writeFailure(stdout, *jsonOutput, profile, validated.Diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	changes := make([]changeOutput, 0, len(result.Statements))
	for _, statement := range result.Statements {
		changes = append(changes, changeOutput{Operation: "updated", ElementType: "statement", ID: statement.ID})
	}
	changes = appendProfileMigrationChange(changes, doc)
	output := normalizeTruthOutput{
		SchemaVersion: outputSchemaVersion, Action: "normalize-truth", DryRun: *dryRun, Applicable: true,
		Profile: profile, Document: documentSummary(next), Statements: result.Statements,
		PlanToken: result.PlanToken, Changes: changes, Diagnostics: nonNilDiagnostics(validated.Diagnostics),
	}
	if *jsonOutput {
		return writeIndentedJSON(stdout, output)
	}
	mode := "Truth normalization plan"
	if applying {
		mode = "Normalized truth"
	}
	fmt.Fprintf(stdout, "%s for %s — %s\n", mode, output.Document.ID, output.Document.Title)
	for _, statement := range output.Statements {
		fmt.Fprintf(stdout, "  %s: %s -> %s\n", statement.ID, statement.PreviousTruth, statement.CurrentTruth)
	}
	if len(output.Statements) == 0 {
		fmt.Fprintln(stdout, "  no sourced statements carry authored truth")
	}
	if output.DryRun {
		fmt.Fprintf(stdout, "plan token: %s\n", output.PlanToken)
		fmt.Fprintln(stdout, "dry-run: no file changes written")
	}
	return nil
}

func writeNormalizeTruthUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  cludia normalize-truth [--json] FILE --dry-run")
	fmt.Fprintln(w, "  cludia normalize-truth [--json] FILE --apply-token TOKEN")
	fmt.Fprintln(w, "Plan or apply state-bound normalization of stored T/F values on sourced statements to U.")
}
