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

type statementIDMapping = argument.StatementIDMapping
type junctorIDMapping = argument.JunctorIDMapping

type renumberOutput struct {
	SchemaVersion       int                     `json:"schema_version"`
	Action              string                  `json:"action"`
	DryRun              bool                    `json:"dry_run"`
	Applicable          bool                    `json:"applicable"`
	Profile             validation.Profile      `json:"profile"`
	Document            documentOutput          `json:"document"`
	StatementIDs        []statementIDMapping    `json:"statement_ids"`
	JunctorIDs          []junctorIDMapping      `json:"junctor_ids"`
	RootMetadataUpdated bool                    `json:"root_metadata_updated"`
	NextIDsBefore       argument.NextIDs        `json:"next_ids_before"`
	NextIDsAfter        argument.NextIDs        `json:"next_ids_after"`
	PlanToken           string                  `json:"plan_token"`
	Changes             []changeOutput          `json:"changes"`
	Diagnostics         []diagnostic.Diagnostic `json:"diagnostics"`
}

func runRenumber(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("renumber", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOutput := fs.Bool("json", false, "output versioned JSON")
	dryRun := fs.Bool("dry-run", false, "build a state-bound complete ID mapping without saving")
	applyToken := fs.String("apply-token", "", "apply the exact reviewed renumber plan token")
	fs.Usage = func() { writeRenumberUsage(fs.Output()) }
	if err := fs.Parse(flagsFirst(args, map[string]bool{"apply-token": true})); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("renumber expects exactly one workspace file")
	}
	applying := strings.TrimSpace(*applyToken) != ""
	if *dryRun == applying {
		fs.Usage()
		return fmt.Errorf("renumber requires exactly one of --dry-run or --apply-token")
	}

	doc, profile, diagnostics := loadValidated(fs.Arg(0))
	if diagnostic.HasErrors(diagnostics) {
		if err := writeFailure(stdout, *jsonOutput, profile, diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	next, result, err := argument.RenumberDocument(doc)
	if err != nil {
		return writeArgumentMutationFailure(stdout, *jsonOutput, profile, err)
	}
	if applying && strings.TrimSpace(*applyToken) != result.PlanToken {
		return writeMutationFailure(stdout, *jsonOutput, profile, "renumber_plan_stale", "renumber plan token does not match the current workspace; run --dry-run again", doc.ID)
	}
	validated, err := validateAndPersistMutation(fs.Arg(0), next, profile, applying)
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
	if result.IDsChanged {
		diagnostics = append(diagnostics, diagnostic.Diagnostic{
			Code:     "external_id_references_unchecked",
			Message:  "Cludia rewrites this workspace and recognized root metadata only. It cannot update ID references in Markdown, scripts, other workspaces, prior exports, or published graphs; use the complete mapping to review those references.",
			Severity: diagnostic.SeverityWarning,
			Element:  doc.ID,
		})
	}
	output := renumberOutput{
		SchemaVersion: outputSchemaVersion, Action: "renumber", DryRun: *dryRun, Applicable: true,
		Profile: profile, Document: documentSummary(next), StatementIDs: result.StatementIDs, JunctorIDs: result.JunctorIDs,
		RootMetadataUpdated: result.RootMetadataUpdated, NextIDsBefore: result.NextIDsBefore, NextIDsAfter: result.NextIDsAfter,
		PlanToken: result.PlanToken, Changes: renumberChanges(doc, next, result), Diagnostics: diagnostics,
	}
	if applying {
		output.DryRun = false
	}
	return writeRenumber(stdout, *jsonOutput, output)
}

func renumberChanges(before, after *argument.Document, result argument.RenumberResult) []changeOutput {
	changes := make([]changeOutput, 0)
	for _, mapping := range result.StatementIDs {
		if mapping.PreviousID != mapping.CurrentID {
			changes = append(changes, changeOutput{Operation: "renumbered", ElementType: "statement", ID: mapping.CurrentID})
		}
	}
	for index, mapping := range result.JunctorIDs {
		previous, current := before.Junctors[index], after.Junctors[index]
		if mapping.PreviousID != mapping.CurrentID {
			changes = append(changes, changeOutput{Operation: "renumbered", ElementType: "junctor", ID: mapping.CurrentID})
		} else if previous.Target != current.Target || strings.Join(previous.Sources, "\x00") != strings.Join(current.Sources, "\x00") {
			changes = append(changes, changeOutput{Operation: "updated", ElementType: "junctor", ID: mapping.CurrentID})
		}
	}
	for index, support := range after.DirectSupports {
		previous := before.DirectSupports[index]
		if previous.Source != support.Source || previous.Target != support.Target {
			changes = append(changes, changeOutput{Operation: "updated", ElementType: "direct_support", ID: support.Source + "->" + support.Target})
		}
	}
	for index, defeat := range after.Defeats {
		if before.Defeats[index] != defeat {
			changes = append(changes, changeOutput{Operation: "updated", ElementType: "defeat", ID: defeat.From})
		}
	}
	if result.RootMetadataUpdated {
		changes = append(changes, changeOutput{Operation: "updated", ElementType: "metadata", ID: "root"})
	}
	changes = appendMetadataChange(changes, nextIDsMetadataChange(before, after))
	return appendProfileMigrationChange(changes, before)
}

func writeRenumber(w io.Writer, jsonOutput bool, output renumberOutput) error {
	if jsonOutput {
		return writeIndentedJSON(w, output)
	}
	verb := "Renumber plan"
	if !output.DryRun {
		verb = "Renumbered"
	}
	fmt.Fprintf(w, "%s for %s — %s\n", verb, output.Document.ID, output.Document.Title)
	fmt.Fprintln(w, "Statements:")
	for _, mapping := range output.StatementIDs {
		fmt.Fprintf(w, "  %s -> %s  %s", mapping.PreviousID, mapping.CurrentID, mapping.Role)
		if mapping.Slug != "" {
			fmt.Fprintf(w, "  %s", mapping.Slug)
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w, "Junctors:")
	for _, mapping := range output.JunctorIDs {
		fmt.Fprintf(w, "  %s -> %s\n", mapping.PreviousID, mapping.CurrentID)
	}
	fmt.Fprintf(w, "next ids: %s\n", output.NextIDsAfter.MetadataValue())
	if output.RootMetadataUpdated {
		fmt.Fprintln(w, "Updated root metadata reference")
	}
	for _, item := range output.Diagnostics {
		writeDiagnostic(w, item)
	}
	if output.DryRun {
		fmt.Fprintf(w, "plan token: %s\n", output.PlanToken)
		fmt.Fprintln(w, "dry-run: no file changes written")
	}
	return nil
}

func writeRenumberUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  cludia renumber [--json] FILE --dry-run")
	fmt.Fprintln(w, "  cludia renumber [--json] FILE --apply-token TOKEN")
	fmt.Fprintln(w, "Plan or apply a state-bound whole-document canonical ID mapping; this is the sole numbering reset.")
}
