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

type replacementIncidents = argument.ReplacementIncidents
type replacementSourceRetarget = argument.ReplacementSourceRetarget
type replacementBlocker = argument.ReplacementBlocker

type replacementOutput struct {
	SchemaVersion         int                         `json:"schema_version"`
	Action                string                      `json:"action"`
	DryRun                bool                        `json:"dry_run"`
	Applicable            bool                        `json:"applicable"`
	Profile               validation.Profile          `json:"profile"`
	Document              documentOutput              `json:"document"`
	OldStatement          argument.Statement          `json:"old_statement"`
	ReplacementStatement  argument.Statement          `json:"replacement_statement"`
	SourceRetargets       []replacementSourceRetarget `json:"source_retargets"`
	JustificationsRemoved []argument.Junctor          `json:"justifications_removed"`
	AffectedDefeats       []argument.Defeat           `json:"affected_defeats"`
	IncidentsBefore       replacementIncidents        `json:"incidents_before"`
	IncidentsRemaining    replacementIncidents        `json:"incidents_remaining"`
	RootRetargetRequested bool                        `json:"root_retarget_requested"`
	RootWillBeRetargeted  bool                        `json:"root_will_be_retargeted"`
	RootRetargeted        bool                        `json:"root_retargeted"`
	DeleteOldRequested    bool                        `json:"delete_old_requested"`
	OldWillBeDeleted      bool                        `json:"old_statement_will_be_deleted"`
	OldStatementDeleted   bool                        `json:"old_statement_deleted"`
	Blockers              []replacementBlocker        `json:"blockers"`
	ComponentsBefore      int                         `json:"components_before"`
	ComponentsAfter       int                         `json:"components_after"`
	NewlyIsolated         []string                    `json:"newly_isolated"`
	PlanToken             string                      `json:"plan_token"`
	Changes               []changeOutput              `json:"changes"`
	Diagnostics           []diagnostic.Diagnostic     `json:"diagnostics"`
}

func runReplace(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("replace", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOutput := fs.Bool("json", false, "output versioned JSON")
	dryRun := fs.Bool("dry-run", false, "build a state-bound replacement plan without saving")
	applyToken := fs.String("apply-token", "", "apply the exact reviewed dry-run plan token")
	replacementRef := fs.String("with", "", "existing replacement statement id or slug")
	var sourceJunctors stringListFlag
	var removeJustifications stringListFlag
	fs.Var(&sourceJunctors, "retarget-source", "AND junctor whose old source should be replaced (repeatable)")
	fs.Var(&removeJustifications, "remove-justification", "junctor targeting the old statement to remove (repeatable)")
	retargetRoot := fs.Bool("retarget-root", false, "retarget recognized root metadata from old to replacement")
	deleteOld := fs.Bool("delete-old", false, "delete the old statement if no incident relation remains")
	fs.Usage = func() { writeReplaceUsage(fs.Output()) }
	valueFlags := map[string]bool{"apply-token": true, "with": true, "retarget-source": true, "remove-justification": true}
	if err := fs.Parse(flagsFirst(args, valueFlags)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 2 || strings.TrimSpace(*replacementRef) == "" {
		fs.Usage()
		return fmt.Errorf("replace expects a workspace file and old statement and requires --with")
	}
	applying := strings.TrimSpace(*applyToken) != ""
	if *dryRun == applying {
		fs.Usage()
		return fmt.Errorf("replace requires exactly one of --dry-run or --apply-token")
	}
	if len(sourceJunctors) == 0 && len(removeJustifications) == 0 && !*retargetRoot && !*deleteOld {
		return fmt.Errorf("replace requires at least one explicit relation choice or --delete-old")
	}

	doc, profile, diagnostics := loadValidated(fs.Arg(0))
	if diagnostic.HasErrors(diagnostics) {
		if err := writeFailure(stdout, *jsonOutput, profile, diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	options := argument.ReplacementOptions{
		OldRef: fs.Arg(1), ReplacementRef: strings.TrimSpace(*replacementRef),
		SourceJunctors: append([]string(nil), sourceJunctors...), RemoveJustifications: append([]string(nil), removeJustifications...),
		RetargetRoot: *retargetRoot, DeleteOld: *deleteOld,
	}
	beforeComponents := len(query.Components(doc))
	beforeIsolated := query.IsolatedStatementIDs(doc)
	next, result, err := argument.ReplaceStatement(doc, options)
	if err != nil {
		return writeArgumentMutationFailure(stdout, *jsonOutput, profile, err)
	}
	if applying && strings.TrimSpace(*applyToken) != result.PlanToken {
		return writeMutationFailure(stdout, *jsonOutput, profile, "replacement_plan_stale", "replacement plan token does not match the current workspace and requested choices; run --dry-run again", result.OldStatement.ID)
	}
	changes := make([]changeOutput, 0, len(result.SourceRetargets)+len(result.JustificationsRemoved)+3)
	for _, retarget := range result.SourceRetargets {
		changes = append(changes, changeOutput{Operation: "updated", ElementType: "junctor", ID: retarget.Junctor.ID})
	}
	for _, removed := range result.JustificationsRemoved {
		changes = append(changes, changeOutput{Operation: "removed", ElementType: "junctor", ID: removed.ID})
	}
	if result.RootRetargeted {
		changes = append(changes, changeOutput{Operation: "updated", ElementType: "metadata", ID: "root"})
	}
	if result.OldDeleted {
		changes = append(changes, changeOutput{Operation: "removed", ElementType: "statement", ID: result.OldStatement.ID})
	}
	changes = appendMetadataChange(changes, nextIDsMetadataChange(doc, next))

	validated, err := validateAndPersistMutation(fs.Arg(0), next, profile, applying && result.Applicable)
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
	if len(result.Blockers) > 0 {
		diagnostics = append(diagnostics, diagnostic.Diagnostic{
			Code: "replacement_delete_blocked", Message: fmt.Sprintf("cannot delete %s while %d incident relations or metadata references remain", result.OldStatement.ID, len(result.Blockers)),
			Severity: diagnostic.SeverityError, Element: result.OldStatement.ID,
		})
	}
	if *deleteOld || *retargetRoot {
		reference := result.OldStatement.ID
		if result.OldStatement.Slug != "" {
			reference = fmt.Sprintf("%s or slug %q", result.OldStatement.ID, result.OldStatement.Slug)
		}
		diagnostics = append(diagnostics, diagnostic.Diagnostic{
			Code: "external_statement_references_unchecked", Message: fmt.Sprintf("Cludia checked only this workspace and recognized root metadata. It cannot detect textual references to %s in Markdown, scripts, other workspaces, prior exports, or published graphs. If no such references exist, no action is needed.", reference),
			Severity: diagnostic.SeverityWarning, Element: result.OldStatement.ID,
		})
	}
	afterIsolated := query.IsolatedStatementIDs(next)
	newlyIsolated := make([]string, 0)
	for _, statement := range next.Statements {
		if afterIsolated[statement.ID] && !beforeIsolated[statement.ID] {
			newlyIsolated = append(newlyIsolated, statement.ID)
		}
	}
	output := replacementOutput{
		SchemaVersion: outputSchemaVersion, Action: "replace", DryRun: *dryRun, Applicable: result.Applicable,
		Profile: profile, Document: documentSummary(next), OldStatement: result.OldStatement, ReplacementStatement: result.ReplacementStatement,
		SourceRetargets: result.SourceRetargets, JustificationsRemoved: result.JustificationsRemoved,
		AffectedDefeats: result.AffectedDefeats, IncidentsBefore: result.IncidentsBefore, IncidentsRemaining: result.IncidentsRemaining,
		RootRetargetRequested: *retargetRoot, RootWillBeRetargeted: result.RootRetargeted,
		RootRetargeted: applying && result.RootRetargeted, DeleteOldRequested: *deleteOld,
		OldWillBeDeleted: *deleteOld && result.Applicable, OldStatementDeleted: applying && result.OldDeleted, Blockers: result.Blockers,
		ComponentsBefore: beforeComponents, ComponentsAfter: len(query.Components(next)), NewlyIsolated: newlyIsolated,
		PlanToken: result.PlanToken, Changes: changes, Diagnostics: diagnostics,
	}
	if !result.Applicable {
		if err := writeReplacement(stdout, *jsonOutput, output); err != nil {
			return err
		}
		return errValidationFailed
	}
	return writeReplacement(stdout, *jsonOutput, output)
}

func writeReplacement(w io.Writer, jsonOutput bool, output replacementOutput) error {
	if jsonOutput {
		return writeIndentedJSON(w, output)
	}
	mode := "Replacement plan"
	if !output.DryRun {
		mode = "Applied replacement"
	}
	fmt.Fprintf(w, "%s: %s -> %s\n", mode, output.OldStatement.ID, output.ReplacementStatement.ID)
	fmt.Fprintf(w, "source retargets: %d; justifications removed: %d; affected defeats: %d\n", len(output.SourceRetargets), len(output.JustificationsRemoved), len(output.AffectedDefeats))
	if output.DryRun {
		fmt.Fprintf(w, "root retarget: requested=%t planned=%t; delete old: requested=%t planned=%t; applicable=%t\n", output.RootRetargetRequested, output.RootWillBeRetargeted, output.DeleteOldRequested, output.OldWillBeDeleted, output.Applicable)
	} else {
		fmt.Fprintf(w, "root retargeted=%t; old deleted=%t; applicable=%t\n", output.RootRetargeted, output.OldStatementDeleted, output.Applicable)
	}
	for _, blocker := range output.Blockers {
		fmt.Fprintf(w, "blocker %s %s: %s\n", blocker.Relation, blocker.ID, blocker.Message)
	}
	if output.PlanToken != "" {
		fmt.Fprintf(w, "plan token: %s\n", output.PlanToken)
	}
	if output.DryRun {
		fmt.Fprintln(w, "dry-run: no file changes written")
	}
	for _, item := range output.Diagnostics {
		writeDiagnostic(w, item)
	}
	return nil
}

func writeReplaceUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  cludia replace [--json] FILE OLD --with NEW [choices] --dry-run")
	fmt.Fprintln(w, "  cludia replace [--json] FILE OLD --with NEW [same choices] --apply-token TOKEN")
	fmt.Fprintln(w, "Choices: --retarget-source JUNCTOR (repeatable), --remove-justification JUNCTOR (repeatable), --retarget-root, --delete-old")
	fmt.Fprintln(w, "Plan and apply an explicit, state-bound material statement replacement without automatic retarget-all semantics.")
}
