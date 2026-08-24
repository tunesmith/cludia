package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/tunesmith/cludia/internal/argfile"
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
	next := doc.Clone()
	statement, ok := next.Statement(fs.Arg(1))
	if !ok {
		return writeMutationFailure(stdout, *jsonOutput, profile, "statement_not_found", fmt.Sprintf("statement %q not found", fs.Arg(1)), fs.Arg(1))
	}
	if statement.Role == argument.RoleCounterpoint {
		return writeMutationFailure(stdout, *jsonOutput, profile, "use_remove_counterpoint", fmt.Sprintf("statement %s is a counterpoint; use remove-counterpoint", statement.ID), statement.ID)
	}
	if len(next.Statements) == 1 {
		return writeMutationFailure(stdout, *jsonOutput, profile, "last_statement", "a workspace must retain at least one statement", statement.ID)
	}
	removed := *statement
	beforeComponents := len(query.Components(next))
	beforeIsolated := query.IsolatedStatementIDs(next)
	removedJunctorIDs := make(map[string]bool)
	junctorsRemoved := []argument.Junctor{}
	junctors := make([]argument.Junctor, 0, len(next.Junctors))
	for _, junctor := range next.Junctors {
		if junctor.Target == statement.ID || containsString(junctor.Sources, statement.ID) {
			junctorsRemoved = append(junctorsRemoved, copyJunctor(junctor))
			removedJunctorIDs[junctor.ID] = true
		} else {
			junctors = append(junctors, junctor)
		}
	}
	next.Junctors = junctors
	for _, defeat := range next.Defeats {
		if defeat.To == statement.ID || defeat.AtTarget == statement.ID || removedJunctorIDs[defeat.JunctorID] {
			return writeMutationFailure(stdout, *jsonOutput, profile, "statement_has_defeats", fmt.Sprintf("deleting %s would detach counterpoint %s; remove the counterpoint first", statement.ID, defeat.From), statement.ID)
		}
	}
	directRemoved := []argument.DirectSupport{}
	direct := make([]argument.DirectSupport, 0, len(next.DirectSupports))
	for _, support := range next.DirectSupports {
		if support.Source == statement.ID || support.Target == statement.ID {
			directRemoved = append(directRemoved, support)
		} else {
			direct = append(direct, support)
		}
	}
	next.DirectSupports = direct
	defeatsRemoved := []argument.Defeat{}
	defeats := make([]argument.Defeat, 0, len(next.Defeats))
	for _, defeat := range next.Defeats {
		if defeat.From == statement.ID || defeat.To == statement.ID || defeat.AtTarget == statement.ID || removedJunctorIDs[defeat.JunctorID] {
			defeatsRemoved = append(defeatsRemoved, defeat)
		} else {
			defeats = append(defeats, defeat)
		}
	}
	next.Defeats = defeats
	statements := make([]argument.Statement, 0, len(next.Statements)-1)
	for _, candidate := range next.Statements {
		if candidate.ID != statement.ID {
			statements = append(statements, candidate)
		}
	}
	next.Statements = statements
	validated := validation.Validate(next, profile)
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
	if !*dryRun {
		if err := argfile.SaveAtomic(fs.Arg(0), next); err != nil {
			return err
		}
	}
	diagnostics = validated.Diagnostics
	if diagnostics == nil {
		diagnostics = []diagnostic.Diagnostic{}
	}
	changes := []changeOutput{{Operation: "removed", ElementType: "statement", ID: removed.ID}}
	for _, junctor := range junctorsRemoved {
		changes = append(changes, changeOutput{Operation: "removed", ElementType: "junctor", ID: junctor.ID})
	}
	for range directRemoved {
		changes = append(changes, changeOutput{Operation: "removed", ElementType: "direct_support", ID: removed.ID})
	}
	for _, defeat := range defeatsRemoved {
		changes = append(changes, changeOutput{Operation: "removed", ElementType: "defeat", ID: defeat.From})
	}
	output := statementDeletionOutput{
		SchemaVersion: outputSchemaVersion, Action: "delete", DryRun: *dryRun,
		Profile: profile, Document: documentSummary(next), Statement: removed,
		JunctorsRemoved: junctorsRemoved, DirectSupportsRemoved: directRemoved,
		DefeatsRemoved: defeatsRemoved, ComponentsBefore: beforeComponents,
		ComponentsAfter: len(query.Components(next)), NewlyIsolated: newlyIsolated,
		Changes: changes, Diagnostics: diagnostics,
	}
	if *jsonOutput {
		return writeIndentedJSON(stdout, output)
	}
	fmt.Fprintf(stdout, "Deleted statement %s:%s\n", removed.ID, removed.Slug)
	fmt.Fprintf(stdout, "removed: %d junctors, %d direct supports, %d defeats\n", len(junctorsRemoved), len(directRemoved), len(defeatsRemoved))
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
