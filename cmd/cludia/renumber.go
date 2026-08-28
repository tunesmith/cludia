package main

import (
	"crypto/sha256"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/tunesmith/cludia/internal/argfile"
	"github.com/tunesmith/cludia/internal/argument"
	"github.com/tunesmith/cludia/internal/diagnostic"
	"github.com/tunesmith/cludia/internal/validation"
)

type statementIDMapping struct {
	PreviousID string        `json:"previous_id"`
	CurrentID  string        `json:"current_id"`
	Role       argument.Role `json:"role"`
	Slug       string        `json:"slug,omitempty"`
}

type junctorIDMapping struct {
	PreviousID string `json:"previous_id"`
	CurrentID  string `json:"current_id"`
}

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
	output, next, err := planRenumber(doc, profile, *dryRun)
	if err != nil {
		return err
	}
	if applying && strings.TrimSpace(*applyToken) != output.PlanToken {
		return writeMutationFailure(stdout, *jsonOutput, profile, "renumber_plan_stale", "renumber plan token does not match the current workspace; run --dry-run again", doc.ID)
	}
	if applying {
		if err := argfile.SaveAtomic(fs.Arg(0), next); err != nil {
			return err
		}
		output.DryRun = false
	}
	return writeRenumber(stdout, *jsonOutput, output)
}

func planRenumber(doc *argument.Document, profile validation.Profile, dryRun bool) (renumberOutput, *argument.Document, error) {
	_, _, before, _, _, err := argument.InspectNextIDs(doc)
	if err != nil {
		return renumberOutput{}, nil, err
	}
	statementCounters := map[argument.Role]int{
		argument.RolePremise: 1, argument.RoleLemma: 1, argument.RoleConclusion: 1, argument.RoleCounterpoint: 1,
	}
	statementMappings := make([]statementIDMapping, 0, len(doc.Statements))
	statementMap := make(map[string]string, len(doc.Statements))
	for _, statement := range doc.Statements {
		number := statementCounters[statement.Role]
		currentID, ok := argument.CanonicalStatementID(statement.Role, number)
		if !ok {
			return renumberOutput{}, nil, fmt.Errorf("statement %s has unsupported role %q", statement.ID, statement.Role)
		}
		statementCounters[statement.Role] = number + 1
		statementMappings = append(statementMappings, statementIDMapping{
			PreviousID: statement.ID, CurrentID: currentID, Role: statement.Role, Slug: statement.Slug,
		})
		statementMap[statement.ID] = currentID
	}
	junctorMappings := make([]junctorIDMapping, 0, len(doc.Junctors))
	junctorMap := make(map[string]string, len(doc.Junctors))
	for index, junctor := range doc.Junctors {
		currentID := argument.CanonicalJunctorID(index + 1)
		junctorMappings = append(junctorMappings, junctorIDMapping{PreviousID: junctor.ID, CurrentID: currentID})
		junctorMap[junctor.ID] = currentID
	}

	next := doc.Clone()
	changes := make([]changeOutput, 0)
	idsChanged := false
	for index := range next.Statements {
		mapping := statementMappings[index]
		next.Statements[index].ID = mapping.CurrentID
		if mapping.PreviousID != mapping.CurrentID {
			idsChanged = true
			changes = append(changes, changeOutput{Operation: "renumbered", ElementType: "statement", ID: mapping.CurrentID})
		}
	}
	for index := range next.Junctors {
		mapping := junctorMappings[index]
		junctor := &next.Junctors[index]
		previousTarget := junctor.Target
		previousSources := append([]string(nil), junctor.Sources...)
		junctor.ID = mapping.CurrentID
		junctor.Target = mappedID(statementMap, junctor.Target)
		for sourceIndex := range junctor.Sources {
			junctor.Sources[sourceIndex] = mappedID(statementMap, junctor.Sources[sourceIndex])
		}
		if mapping.PreviousID != mapping.CurrentID {
			idsChanged = true
			changes = append(changes, changeOutput{Operation: "renumbered", ElementType: "junctor", ID: mapping.CurrentID})
		} else if previousTarget != junctor.Target || strings.Join(previousSources, "\x00") != strings.Join(junctor.Sources, "\x00") {
			changes = append(changes, changeOutput{Operation: "updated", ElementType: "junctor", ID: mapping.CurrentID})
		}
	}
	for index := range next.DirectSupports {
		support := &next.DirectSupports[index]
		previousSource, previousTarget := support.Source, support.Target
		support.Source = mappedID(statementMap, support.Source)
		support.Target = mappedID(statementMap, support.Target)
		if previousSource != support.Source || previousTarget != support.Target {
			changes = append(changes, changeOutput{Operation: "updated", ElementType: "direct_support", ID: support.Source + "->" + support.Target})
		}
	}
	for index := range next.Defeats {
		defeat := &next.Defeats[index]
		previous := *defeat
		defeat.From = mappedID(statementMap, defeat.From)
		defeat.To = mappedID(statementMap, defeat.To)
		defeat.AtTarget = mappedID(statementMap, defeat.AtTarget)
		defeat.JunctorID = mappedID(junctorMap, defeat.JunctorID)
		if previous != *defeat {
			changes = append(changes, changeOutput{Operation: "updated", ElementType: "defeat", ID: defeat.From})
		}
	}
	rootUpdated := false
	for index := range next.Metadata {
		metadata := &next.Metadata[index]
		if metadata.Key == "root" {
			if current, ok := statementMap[metadata.Value]; ok && current != metadata.Value {
				metadata.Value = current
				rootUpdated = true
			}
		}
	}
	if rootUpdated {
		changes = append(changes, changeOutput{Operation: "updated", ElementType: "metadata", ID: "root"})
	}
	after := argument.NextIDs{
		P: statementCounters[argument.RolePremise], L: statementCounters[argument.RoleLemma],
		C: statementCounters[argument.RoleConclusion], CP: statementCounters[argument.RoleCounterpoint],
		J: len(next.Junctors) + 1,
	}
	previousNextIDs, previousNextIDsPresent := next.MetadataValue(argument.NextIDsMetadataKey)
	argument.SetNextIDs(next, after)
	currentNextIDs, _ := next.MetadataValue(argument.NextIDsMetadataKey)
	if !previousNextIDsPresent || previousNextIDs != currentNextIDs {
		operation := "added"
		if previousNextIDsPresent {
			operation = "updated"
		}
		changes = append(changes, changeOutput{Operation: operation, ElementType: "metadata", ID: argument.NextIDsMetadataKey})
	}

	validated := validation.Validate(next, profile)
	if !validated.OK() {
		return renumberOutput{}, nil, fmt.Errorf("renumbered workspace failed validation: %v", validated.Diagnostics)
	}
	diagnostics := validated.Diagnostics
	if diagnostics == nil {
		diagnostics = []diagnostic.Diagnostic{}
	}
	if idsChanged {
		diagnostics = append(diagnostics, diagnostic.Diagnostic{
			Code:     "external_id_references_unchecked",
			Message:  "Cludia rewrites this workspace and recognized root metadata only. It cannot update ID references in Markdown, scripts, other workspaces, prior exports, or published graphs; use the complete mapping to review those references.",
			Severity: diagnostic.SeverityWarning,
			Element:  doc.ID,
		})
	}
	token, err := renumberPlanToken(doc, statementMappings, junctorMappings)
	if err != nil {
		return renumberOutput{}, nil, err
	}
	return renumberOutput{
		SchemaVersion: outputSchemaVersion, Action: "renumber", DryRun: dryRun, Applicable: true,
		Profile: profile, Document: documentSummary(next), StatementIDs: statementMappings, JunctorIDs: junctorMappings,
		RootMetadataUpdated: rootUpdated, NextIDsBefore: before, NextIDsAfter: after,
		PlanToken: token, Changes: changes, Diagnostics: diagnostics,
	}, next, nil
}

func mappedID(mapping map[string]string, id string) string {
	if current, ok := mapping[id]; ok {
		return current
	}
	return id
}

func renumberPlanToken(doc *argument.Document, statements []statementIDMapping, junctors []junctorIDMapping) (string, error) {
	serialized, err := argfile.Serialize(doc)
	if err != nil {
		return "", err
	}
	var mapping strings.Builder
	for _, item := range statements {
		fmt.Fprintf(&mapping, "\x00statement:%s=%s", item.PreviousID, item.CurrentID)
	}
	for _, item := range junctors {
		fmt.Fprintf(&mapping, "\x00junctor:%s=%s", item.PreviousID, item.CurrentID)
	}
	sum := sha256.Sum256([]byte("renumber-v1\x00" + serialized + mapping.String()))
	return fmt.Sprintf("%x", sum[:]), nil
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
	fmt.Fprintln(w, "Usage: cludia renumber [--json] FILE (--dry-run | --apply-token TOKEN)")
	fmt.Fprintln(w, "Plan or atomically apply a complete role-based statement and junctor ID rewrite.")
}
