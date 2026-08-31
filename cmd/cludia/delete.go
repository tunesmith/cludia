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
	"github.com/tunesmith/cludia/internal/query"
	"github.com/tunesmith/cludia/internal/validation"
)

type statementDeletionOutput struct {
	SchemaVersion         int                      `json:"schema_version"`
	Action                string                   `json:"action"`
	DryRun                bool                     `json:"dry_run"`
	Profile               validation.Profile       `json:"profile"`
	Document              documentOutput           `json:"document"`
	Statement             argument.Statement       `json:"statement"`
	JunctorsRemoved       []argument.Junctor       `json:"junctors_removed"`
	DirectSupportsRemoved []argument.DirectSupport `json:"direct_supports_removed"`
	DefeatsRemoved        []argument.Defeat        `json:"defeats_removed"`
	ComponentsBefore      int                      `json:"components_before"`
	ComponentsAfter       int                      `json:"components_after"`
	NewlyIsolated         []string                 `json:"newly_isolated"`
	Changes               []changeOutput           `json:"changes"`
	Diagnostics           []diagnostic.Diagnostic  `json:"diagnostics"`
}

func runDelete(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOutput := fs.Bool("json", false, "output versioned JSON")
	dryRun := fs.Bool("dry-run", false, "validate and report without saving")
	fs.Usage = func() { writeDeleteUsage(fs.Output()) }
	if err := fs.Parse(flagsFirst(args, map[string]bool{})); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 2 {
		fs.Usage()
		return fmt.Errorf("delete expects a file and statement")
	}
	doc, profile, diagnostics := loadValidated(fs.Arg(0))
	if diagnostic.HasErrors(diagnostics) {
		if err := writeFailure(stdout, *jsonOutput, profile, diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	beforeComponents := len(query.Components(doc))
	beforeIsolated := query.IsolatedStatementIDs(doc)
	next, result, err := argument.DeleteStatement(doc, fs.Arg(1))
	if err != nil {
		return writeArgumentMutationFailure(stdout, *jsonOutput, profile, err)
	}
	validated, err := validateAndPersistMutation(fs.Arg(0), next, profile, !*dryRun)
	if err != nil {
		return err
	}
	if !validated.OK() {
		if err := writeFailure(stdout, *jsonOutput, profile, validated.Diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	afterIsolated := query.IsolatedStatementIDs(next)
	newlyIsolated := []string{}
	for _, candidate := range next.Statements {
		if afterIsolated[candidate.ID] && !beforeIsolated[candidate.ID] {
			newlyIsolated = append(newlyIsolated, candidate.ID)
		}
	}
	diagnostics = validated.Diagnostics
	if diagnostics == nil {
		diagnostics = []diagnostic.Diagnostic{}
	}
	changes := []changeOutput{{Operation: "removed", ElementType: "statement", ID: result.Statement.ID}}
	for _, junctor := range result.JunctorsRemoved {
		changes = append(changes, changeOutput{Operation: "removed", ElementType: "junctor", ID: junctor.ID})
	}
	for range result.DirectSupportsRemoved {
		changes = append(changes, changeOutput{Operation: "removed", ElementType: "direct_support", ID: result.Statement.ID})
	}
	for _, defeat := range result.DefeatsRemoved {
		changes = append(changes, changeOutput{Operation: "removed", ElementType: "defeat", ID: defeat.From})
	}
	changes = appendMetadataChange(changes, nextIDsMetadataChange(doc, next))
	changes = appendProfileMigrationChange(changes, doc)
	output := statementDeletionOutput{
		SchemaVersion: outputSchemaVersion, Action: "delete", DryRun: *dryRun,
		Profile: profile, Document: documentSummary(next), Statement: result.Statement,
		JunctorsRemoved: result.JunctorsRemoved, DirectSupportsRemoved: result.DirectSupportsRemoved,
		DefeatsRemoved: result.DefeatsRemoved, ComponentsBefore: beforeComponents,
		ComponentsAfter: len(query.Components(next)), NewlyIsolated: newlyIsolated,
		Changes: changes, Diagnostics: diagnostics,
	}
	if *jsonOutput {
		return writeIndentedJSON(stdout, output)
	}
	fmt.Fprintf(stdout, "Deleted statement %s:%s\n", result.Statement.ID, result.Statement.Slug)
	fmt.Fprintf(stdout, "removed: %d junctors, %d direct supports, %d defeats\n", len(result.JunctorsRemoved), len(result.DirectSupportsRemoved), len(result.DefeatsRemoved))
	fmt.Fprintf(stdout, "components: %d -> %d\n", output.ComponentsBefore, output.ComponentsAfter)
	if len(newlyIsolated) > 0 {
		fmt.Fprintf(stdout, "newly isolated: %s\n", strings.Join(newlyIsolated, ", "))
	}
	if output.DryRun {
		fmt.Fprintln(stdout, "dry-run: no file changes written")
	}
	return nil
}

func writeDeleteUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cludia delete [--dry-run] [--json] FILE STATEMENT")
	fmt.Fprintln(w, "Delete a non-counterpoint statement and its incident relations after reporting structural effects.")
}
