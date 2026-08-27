package main

import (
	"crypto/sha256"
	"errors"
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/tunesmith/cludia/internal/argfile"
	"github.com/tunesmith/cludia/internal/argument"
	"github.com/tunesmith/cludia/internal/diagnostic"
	"github.com/tunesmith/cludia/internal/query"
	"github.com/tunesmith/cludia/internal/validation"
)

type replacementIncidents struct {
	SourceJunctors []string                 `json:"source_junctors"`
	TargetJunctors []string                 `json:"target_junctors"`
	DirectSupports []argument.DirectSupport `json:"direct_supports"`
	Defeats        []argument.Defeat        `json:"defeats"`
	RootMetadata   bool                     `json:"root_metadata"`
}

type replacementSourceRetarget struct {
	PreviousJunctor argument.Junctor `json:"previous_junctor"`
	Junctor         argument.Junctor `json:"junctor"`
}

type replacementBlocker struct {
	Relation string `json:"relation"`
	ID       string `json:"id"`
	Message  string `json:"message"`
}

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
	old, ok := doc.Statement(fs.Arg(1))
	if !ok {
		return writeMutationFailure(stdout, *jsonOutput, profile, "statement_not_found", fmt.Sprintf("old statement %q not found", fs.Arg(1)), fs.Arg(1))
	}
	replacement, ok := doc.Statement(strings.TrimSpace(*replacementRef))
	if !ok {
		return writeMutationFailure(stdout, *jsonOutput, profile, "statement_not_found", fmt.Sprintf("replacement statement %q not found", *replacementRef), *replacementRef)
	}
	if old.ID == replacement.ID {
		return writeMutationFailure(stdout, *jsonOutput, profile, "replacement_same_statement", "old and replacement statements must differ", old.ID)
	}
	if old.Role == argument.RoleCounterpoint || replacement.Role == argument.RoleCounterpoint {
		return writeMutationFailure(stdout, *jsonOutput, profile, "replacement_counterpoint_unsupported", "focused material replacement does not replace counterpoint statements", old.ID)
	}
	if duplicate := duplicateSelection(sourceJunctors); duplicate != "" {
		return writeMutationFailure(stdout, *jsonOutput, profile, "replacement_selection_duplicate", fmt.Sprintf("retarget-source selection %q appears more than once", duplicate), duplicate)
	}
	if duplicate := duplicateSelection(removeJustifications); duplicate != "" {
		return writeMutationFailure(stdout, *jsonOutput, profile, "replacement_selection_duplicate", fmt.Sprintf("remove-justification selection %q appears more than once", duplicate), duplicate)
	}

	planToken, err := replacementPlanToken(doc, old.ID, replacement.ID, sourceJunctors, removeJustifications, *retargetRoot, *deleteOld)
	if err != nil {
		return err
	}
	if applying && strings.TrimSpace(*applyToken) != planToken {
		return writeMutationFailure(stdout, *jsonOutput, profile, "replacement_plan_stale", "replacement plan token does not match the current workspace and requested choices; run --dry-run again", old.ID)
	}

	next := doc.Clone()
	oldNext, _ := next.Statement(old.ID)
	replacementNext, _ := next.Statement(replacement.ID)
	beforeComponents := len(query.Components(next))
	beforeIsolated := query.IsolatedStatementIDs(next)
	incidentsBefore := collectReplacementIncidents(next, oldNext)
	sourceRetargets := make([]replacementSourceRetarget, 0, len(sourceJunctors))
	changes := make([]changeOutput, 0, len(sourceJunctors)+len(removeJustifications)+2)
	selectedJunctors := make(map[string]bool, len(sourceJunctors))
	for _, selected := range sourceJunctors {
		junctor, ok := next.Junctor(selected)
		if !ok {
			return writeMutationFailure(stdout, *jsonOutput, profile, "junctor_not_found", fmt.Sprintf("retarget-source junctor %q not found", selected), selected)
		}
		if junctor.Connector != argument.ConnectorAND {
			return writeMutationFailure(stdout, *jsonOutput, profile, "junctor_not_editable", fmt.Sprintf("replacement source retargeting edits only AND junctors; %s uses %s", junctor.ID, junctor.Connector), junctor.ID)
		}
		index := stringIndex(junctor.Sources, oldNext.ID)
		if index < 0 {
			return writeMutationFailure(stdout, *jsonOutput, profile, "junctor_source_not_found", fmt.Sprintf("old statement %s is not a source of junctor %s", oldNext.ID, junctor.ID), junctor.ID)
		}
		if containsString(junctor.Sources, replacementNext.ID) {
			return writeMutationFailure(stdout, *jsonOutput, profile, "junctor_source_duplicate", fmt.Sprintf("replacement statement %s is already a source of junctor %s", replacementNext.ID, junctor.ID), junctor.ID)
		}
		previous := copyJunctor(*junctor)
		junctor.Sources[index] = replacementNext.ID
		sourceRetargets = append(sourceRetargets, replacementSourceRetarget{PreviousJunctor: previous, Junctor: copyJunctor(*junctor)})
		selectedJunctors[junctor.ID] = true
		changes = append(changes, changeOutput{Operation: "updated", ElementType: "junctor", ID: junctor.ID})
	}

	removeSet := make(map[string]bool, len(removeJustifications))
	justificationsRemoved := make([]argument.Junctor, 0, len(removeJustifications))
	for _, selected := range removeJustifications {
		junctor, ok := next.Junctor(selected)
		if !ok {
			return writeMutationFailure(stdout, *jsonOutput, profile, "junctor_not_found", fmt.Sprintf("remove-justification junctor %q not found", selected), selected)
		}
		if junctor.Target != oldNext.ID {
			return writeMutationFailure(stdout, *jsonOutput, profile, "replacement_justification_target_mismatch", fmt.Sprintf("junctor %s targets %s rather than old statement %s", junctor.ID, junctor.Target, oldNext.ID), junctor.ID)
		}
		for _, defeat := range next.Defeats {
			if defeat.Scope == argument.DefeatInference && defeat.JunctorID == junctor.ID {
				return writeMutationFailure(stdout, *jsonOutput, profile, "junctor_has_undercuts", fmt.Sprintf("junctor %s is targeted by undercut %s; remove the counterpoint first", junctor.ID, defeat.From), junctor.ID)
			}
		}
		removeSet[junctor.ID] = true
		justificationsRemoved = append(justificationsRemoved, copyJunctor(*junctor))
		changes = append(changes, changeOutput{Operation: "removed", ElementType: "junctor", ID: junctor.ID})
	}
	if len(removeSet) > 0 {
		junctors := make([]argument.Junctor, 0, len(next.Junctors)-len(removeSet))
		for _, junctor := range next.Junctors {
			if !removeSet[junctor.ID] {
				junctors = append(junctors, junctor)
			}
		}
		next.Junctors = junctors
	}

	rootMatched := replacementRootMatches(next, oldNext)
	rootRetargeted := false
	if *retargetRoot {
		if !rootMatched {
			return writeMutationFailure(stdout, *jsonOutput, profile, "replacement_root_not_old", fmt.Sprintf("recognized root metadata does not reference old statement %s", oldNext.ID), oldNext.ID)
		}
		rootValue := replacementNext.ID
		if replacementNext.Slug != "" {
			rootValue = replacementNext.Slug
		}
		for i := range next.Metadata {
			metadata := &next.Metadata[i]
			if metadata.Key == "root" && (metadata.Value == oldNext.ID || (oldNext.Slug != "" && metadata.Value == oldNext.Slug)) {
				metadata.Value = rootValue
				rootRetargeted = true
			}
		}
		changes = append(changes, changeOutput{Operation: "updated", ElementType: "metadata", ID: "root"})
	}

	affectedDefeats := make([]argument.Defeat, 0)
	for _, defeat := range next.Defeats {
		if defeat.Scope == argument.DefeatInference && selectedJunctors[defeat.JunctorID] {
			affectedDefeats = append(affectedDefeats, defeat)
		}
	}
	remaining := collectReplacementIncidents(next, oldNext)
	blockers := []replacementBlocker{}
	if *deleteOld {
		blockers = replacementDeletionBlockers(remaining)
	}
	if len(blockers) == 0 && *deleteOld {
		statements := make([]argument.Statement, 0, len(next.Statements)-1)
		for _, statement := range next.Statements {
			if statement.ID != oldNext.ID {
				statements = append(statements, statement)
			}
		}
		next.Statements = statements
		changes = append(changes, changeOutput{Operation: "removed", ElementType: "statement", ID: oldNext.ID})
	}

	validated := validation.Validate(next, profile)
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
	if len(blockers) > 0 {
		diagnostics = append(diagnostics, diagnostic.Diagnostic{
			Code: "replacement_delete_blocked", Message: fmt.Sprintf("cannot delete %s while %d incident relations or metadata references remain", oldNext.ID, len(blockers)),
			Severity: diagnostic.SeverityError, Element: oldNext.ID,
		})
	}
	if *deleteOld || *retargetRoot {
		reference := oldNext.ID
		if oldNext.Slug != "" {
			reference = fmt.Sprintf("%s or slug %q", oldNext.ID, oldNext.Slug)
		}
		diagnostics = append(diagnostics, diagnostic.Diagnostic{
			Code: "external_statement_references_unchecked", Message: fmt.Sprintf("Cludia checked only this workspace and recognized root metadata. It cannot detect textual references to %s in Markdown, scripts, other workspaces, prior exports, or published graphs. If no such references exist, no action is needed.", reference),
			Severity: diagnostic.SeverityWarning, Element: oldNext.ID,
		})
	}
	afterIsolated := query.IsolatedStatementIDs(next)
	newlyIsolated := make([]string, 0)
	for _, statement := range next.Statements {
		if afterIsolated[statement.ID] && !beforeIsolated[statement.ID] {
			newlyIsolated = append(newlyIsolated, statement.ID)
		}
	}
	applicable := len(blockers) == 0
	outputToken := ""
	if applicable {
		outputToken = planToken
	}
	output := replacementOutput{
		SchemaVersion: outputSchemaVersion, Action: "replace", DryRun: *dryRun, Applicable: applicable,
		Profile: profile, Document: documentSummary(next), OldStatement: *old, ReplacementStatement: *replacement,
		SourceRetargets: sourceRetargets, JustificationsRemoved: justificationsRemoved,
		AffectedDefeats: affectedDefeats, IncidentsBefore: incidentsBefore, IncidentsRemaining: remaining,
		RootRetargetRequested: *retargetRoot, RootWillBeRetargeted: rootRetargeted,
		RootRetargeted: applying && rootRetargeted, DeleteOldRequested: *deleteOld,
		OldWillBeDeleted: *deleteOld && applicable, OldStatementDeleted: applying && *deleteOld && applicable, Blockers: blockers,
		ComponentsBefore: beforeComponents, ComponentsAfter: len(query.Components(next)), NewlyIsolated: newlyIsolated,
		PlanToken: outputToken, Changes: changes, Diagnostics: diagnostics,
	}
	if !applicable {
		if err := writeReplacement(stdout, *jsonOutput, output); err != nil {
			return err
		}
		return errValidationFailed
	}
	if applying {
		if err := argfile.SaveAtomic(fs.Arg(0), next); err != nil {
			return err
		}
	}
	return writeReplacement(stdout, *jsonOutput, output)
}

func replacementPlanToken(doc *argument.Document, oldID, replacementID string, sourceJunctors, removeJustifications []string, retargetRoot, deleteOld bool) (string, error) {
	serialized, err := argfile.Serialize(doc)
	if err != nil {
		return "", err
	}
	sources := append([]string(nil), sourceJunctors...)
	removals := append([]string(nil), removeJustifications...)
	sort.Strings(sources)
	sort.Strings(removals)
	payload := fmt.Sprintf("%s\x00old=%s\x00new=%s\x00sources=%s\x00remove=%s\x00root=%t\x00delete=%t", serialized, oldID, replacementID, strings.Join(sources, ","), strings.Join(removals, ","), retargetRoot, deleteOld)
	sum := sha256.Sum256([]byte(payload))
	return fmt.Sprintf("%x", sum[:]), nil
}

func duplicateSelection(values []string) string {
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if seen[value] {
			return value
		}
		seen[value] = true
	}
	return ""
}

func stringIndex(values []string, target string) int {
	for index, value := range values {
		if value == target {
			return index
		}
	}
	return -1
}

func collectReplacementIncidents(doc *argument.Document, statement *argument.Statement) replacementIncidents {
	incidents := replacementIncidents{
		SourceJunctors: []string{}, TargetJunctors: []string{},
		DirectSupports: []argument.DirectSupport{}, Defeats: []argument.Defeat{},
	}
	for _, junctor := range doc.Junctors {
		if containsString(junctor.Sources, statement.ID) {
			incidents.SourceJunctors = append(incidents.SourceJunctors, junctor.ID)
		}
		if junctor.Target == statement.ID {
			incidents.TargetJunctors = append(incidents.TargetJunctors, junctor.ID)
		}
	}
	for _, support := range doc.DirectSupports {
		if support.Source == statement.ID || support.Target == statement.ID {
			incidents.DirectSupports = append(incidents.DirectSupports, support)
		}
	}
	for _, defeat := range doc.Defeats {
		if defeat.From == statement.ID || defeat.To == statement.ID || defeat.AtTarget == statement.ID {
			incidents.Defeats = append(incidents.Defeats, defeat)
		}
	}
	incidents.RootMetadata = replacementRootMatches(doc, statement)
	return incidents
}

func replacementRootMatches(doc *argument.Document, statement *argument.Statement) bool {
	for _, metadata := range doc.Metadata {
		if metadata.Key == "root" && (metadata.Value == statement.ID || (statement.Slug != "" && metadata.Value == statement.Slug)) {
			return true
		}
	}
	return false
}

func replacementDeletionBlockers(incidents replacementIncidents) []replacementBlocker {
	blockers := make([]replacementBlocker, 0, len(incidents.SourceJunctors)+len(incidents.TargetJunctors)+len(incidents.DirectSupports)+len(incidents.Defeats)+1)
	for _, id := range incidents.SourceJunctors {
		blockers = append(blockers, replacementBlocker{Relation: "junctor_source", ID: id, Message: "old statement remains a source; select this junctor with --retarget-source or retain the old statement"})
	}
	for _, id := range incidents.TargetJunctors {
		blockers = append(blockers, replacementBlocker{Relation: "junctor_target", ID: id, Message: "old statement retains an incoming justification; select it with --remove-justification or retain the old statement"})
	}
	for _, support := range incidents.DirectSupports {
		id := support.Source + "->" + support.Target
		blockers = append(blockers, replacementBlocker{Relation: "direct_support", ID: id, Message: "legacy direct support requires separate relation review"})
	}
	for _, defeat := range incidents.Defeats {
		blockers = append(blockers, replacementBlocker{Relation: "defeat", ID: defeat.From, Message: "attached counterpoint requires explicit review and removal before deleting the old statement"})
	}
	if incidents.RootMetadata {
		blockers = append(blockers, replacementBlocker{Relation: "metadata_root", ID: "root", Message: "recognized root metadata still references the old statement; select --retarget-root or retain the old statement"})
	}
	return blockers
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
