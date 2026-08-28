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

type defeatMutationOutput struct {
	SchemaVersion int                     `json:"schema_version"`
	Action        string                  `json:"action"`
	DryRun        bool                    `json:"dry_run"`
	Profile       validation.Profile      `json:"profile"`
	Document      documentOutput          `json:"document"`
	Counterpoint  argument.Statement      `json:"counterpoint"`
	Defeat        argument.Defeat         `json:"defeat"`
	Changes       []changeOutput          `json:"changes"`
	Diagnostics   []diagnostic.Diagnostic `json:"diagnostics"`
}

type counterpointRemovalOutput struct {
	SchemaVersion    int                     `json:"schema_version"`
	Action           string                  `json:"action"`
	DryRun           bool                    `json:"dry_run"`
	Profile          validation.Profile      `json:"profile"`
	Document         documentOutput          `json:"document"`
	Counterpoint     argument.Statement      `json:"counterpoint"`
	DefeatsRemoved   []argument.Defeat       `json:"defeats_removed"`
	ComponentsBefore int                     `json:"components_before"`
	ComponentsAfter  int                     `json:"components_after"`
	NewlyIsolated    []string                `json:"newly_isolated"`
	Changes          []changeOutput          `json:"changes"`
	Diagnostics      []diagnostic.Diagnostic `json:"diagnostics"`
}

type defeatFlags struct {
	jsonOutput *bool
	text       *string
	id         *string
	slug       *string
	truth      *string
	kind       *string
}

func addDefeatFlags(fs *flag.FlagSet) defeatFlags {
	return defeatFlags{
		jsonOutput: fs.Bool("json", false, "output versioned JSON"),
		text:       fs.String("text", "", "counterpoint text"),
		id:         fs.String("id", "", "exact next CP statement id (generated if omitted)"),
		slug:       fs.String("slug", "", "counterpoint slug (generated if omitted)"),
		truth:      fs.String("truth", "T", "counterpoint truth: T, F, or U"),
		kind:       fs.String("kind", "fact", "counterpoint kind: fact or value"),
	}
}

var defeatValueFlags = map[string]bool{"text": true, "id": true, "slug": true, "truth": true, "kind": true}

func runUndermine(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("undermine", flag.ContinueOnError)
	fs.SetOutput(stderr)
	flags := addDefeatFlags(fs)
	fs.Usage = func() { writeUndermineUsage(fs.Output()) }
	if err := fs.Parse(flagsFirst(args, defeatValueFlags)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 2 || strings.TrimSpace(*flags.text) == "" {
		fs.Usage()
		return fmt.Errorf("undermine expects a file and premise and requires --text")
	}
	return createDefeat(fs.Arg(0), "undermine", argument.DefeatPremise, fs.Arg(1), flags, stdout)
}

func runUndercut(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("undercut", flag.ContinueOnError)
	fs.SetOutput(stderr)
	flags := addDefeatFlags(fs)
	fs.Usage = func() { writeUndercutUsage(fs.Output()) }
	if err := fs.Parse(flagsFirst(args, defeatValueFlags)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 2 || strings.TrimSpace(*flags.text) == "" {
		fs.Usage()
		return fmt.Errorf("undercut expects a file and junctor and requires --text")
	}
	return createDefeat(fs.Arg(0), "undercut", argument.DefeatInference, fs.Arg(1), flags, stdout)
}

func runCounterpoint(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("counterpoint", flag.ContinueOnError)
	fs.SetOutput(stderr)
	flags := addDefeatFlags(fs)
	fs.Usage = func() { writeCounterpointUsage(fs.Output()) }
	if err := fs.Parse(flagsFirst(args, defeatValueFlags)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 2 || strings.TrimSpace(*flags.text) == "" {
		fs.Usage()
		return fmt.Errorf("counterpoint expects a file and counterpoint and requires --text")
	}
	return createDefeat(fs.Arg(0), "counterpoint", argument.DefeatCounterpoint, fs.Arg(1), flags, stdout)
}

func createDefeat(path, action string, scope argument.DefeatScope, targetRef string, flags defeatFlags, stdout io.Writer) error {
	doc, profile, diagnostics := loadValidated(path)
	if diagnostic.HasErrors(diagnostics) {
		if err := writeFailure(stdout, *flags.jsonOutput, profile, diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	next := doc.Clone()
	allocator, err := argument.NewIDAllocator(next)
	if err != nil {
		return err
	}
	defeat := argument.Defeat{Scope: scope}
	switch scope {
	case argument.DefeatPremise:
		target, ok := next.Statement(targetRef)
		if !ok {
			return writeMutationFailure(stdout, *flags.jsonOutput, profile, "target_not_found", fmt.Sprintf("premise %q not found", targetRef), targetRef)
		}
		if target.Role != argument.RolePremise {
			return writeMutationFailure(stdout, *flags.jsonOutput, profile, "undermine_target_role", fmt.Sprintf("undermine target %s has role %s, expected premise", target.ID, target.Role), target.ID)
		}
		defeat.To = target.ID
	case argument.DefeatInference:
		junctor, ok := next.Junctor(targetRef)
		if !ok {
			return writeMutationFailure(stdout, *flags.jsonOutput, profile, "junctor_not_found", fmt.Sprintf("junctor %q not found", targetRef), targetRef)
		}
		defeat.JunctorID, defeat.AtTarget = junctor.ID, junctor.Target
	case argument.DefeatCounterpoint:
		target, ok := next.Statement(targetRef)
		if !ok {
			return writeMutationFailure(stdout, *flags.jsonOutput, profile, "target_not_found", fmt.Sprintf("counterpoint %q not found", targetRef), targetRef)
		}
		if target.Role != argument.RoleCounterpoint {
			return writeMutationFailure(stdout, *flags.jsonOutput, profile, "counterpoint_target_role", fmt.Sprintf("counterpoint target %s has role %s, expected counterpoint", target.ID, target.Role), target.ID)
		}
		defeat.To = target.ID
	}
	truth, truthOK := parseTruth(*flags.truth)
	kind, kindOK := parseKind(*flags.kind)
	id, allocationErr := allocator.Statement(argument.RoleCounterpoint, strings.TrimSpace(*flags.id))
	if allocationErr != nil {
		return writeIDAllocationFailure(stdout, *flags.jsonOutput, profile, allocationErr)
	}
	slug := strings.TrimSpace(*flags.slug)
	if slug == "" {
		slug = argument.UniqueSlug(next, *flags.text)
	}
	statement := argument.Statement{
		ID: id, Slug: slug, Role: argument.RoleCounterpoint, Kind: kind,
		Truth: truth, Text: strings.TrimSpace(*flags.text),
	}
	defeat.From = id
	next.Statements = append(next.Statements, statement)
	next.Defeats = append(next.Defeats, defeat)
	metadataChange := persistIDAllocator(next, allocator)
	if !truthOK {
		diagnostics = append(diagnostics, diagnostic.Diagnostic{Code: "truth_invalid", Message: fmt.Sprintf("invalid truth %q; expected T, F, or U", *flags.truth), Severity: diagnostic.SeverityError, Element: id})
	}
	if !kindOK {
		diagnostics = append(diagnostics, diagnostic.Diagnostic{Code: "kind_invalid", Message: fmt.Sprintf("invalid kind %q; expected fact or value", *flags.kind), Severity: diagnostic.SeverityError, Element: id})
	}
	if !diagnostic.HasErrors(diagnostics) {
		validated := validation.Validate(next, profile)
		diagnostics = validated.Diagnostics
	}
	if diagnostic.HasErrors(diagnostics) {
		if err := writeFailure(stdout, *flags.jsonOutput, profile, diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	if err := argfile.SaveAtomic(path, next); err != nil {
		return err
	}
	if diagnostics == nil {
		diagnostics = []diagnostic.Diagnostic{}
	}
	output := defeatMutationOutput{
		SchemaVersion: outputSchemaVersion, Action: action, DryRun: false,
		Profile: profile, Document: documentSummary(next), Counterpoint: statement, Defeat: defeat,
		Changes: appendMetadataChange([]changeOutput{
			{Operation: "added", ElementType: "statement", ID: statement.ID},
			{Operation: "added", ElementType: "defeat", ID: statement.ID},
		}, metadataChange),
		Diagnostics: diagnostics,
	}
	if *flags.jsonOutput {
		return writeIndentedJSON(stdout, output)
	}
	fmt.Fprintf(stdout, "Added %s %s:%s\n%s\n", action, statement.ID, statement.Slug, statement.Text)
	fmt.Fprintf(stdout, "%s\n", formatDefeat(defeat))
	for _, item := range output.Diagnostics {
		writeDiagnostic(stdout, item)
	}
	return nil
}

func runRemoveCounterpoint(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("remove-counterpoint", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOutput := fs.Bool("json", false, "output versioned JSON")
	dryRun := fs.Bool("dry-run", false, "validate and report without saving")
	fs.Usage = func() { writeRemoveCounterpointUsage(fs.Output()) }
	if err := fs.Parse(flagsFirst(args, map[string]bool{})); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 2 {
		fs.Usage()
		return fmt.Errorf("remove-counterpoint expects a file and counterpoint")
	}
	doc, profile, diagnostics := loadValidated(fs.Arg(0))
	if diagnostic.HasErrors(diagnostics) {
		if err := writeFailure(stdout, *jsonOutput, profile, diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	next := doc.Clone()
	metadataChange, err := ensureNextIDs(next)
	if err != nil {
		return err
	}
	statement, ok := next.Statement(fs.Arg(1))
	if !ok {
		return writeMutationFailure(stdout, *jsonOutput, profile, "counterpoint_not_found", fmt.Sprintf("counterpoint %q not found", fs.Arg(1)), fs.Arg(1))
	}
	if statement.Role != argument.RoleCounterpoint {
		return writeMutationFailure(stdout, *jsonOutput, profile, "counterpoint_role_required", fmt.Sprintf("statement %s has role %s, expected counterpoint", statement.ID, statement.Role), statement.ID)
	}
	for _, defeat := range next.Defeats {
		if defeat.Scope == argument.DefeatCounterpoint && defeat.To == statement.ID {
			return writeMutationFailure(stdout, *jsonOutput, profile, "counterpoint_has_dependents", fmt.Sprintf("counterpoint %s is targeted by %s; remove dependent counterpoints first", statement.ID, defeat.From), statement.ID)
		}
	}
	for _, junctor := range next.Junctors {
		if junctor.Target == statement.ID || containsString(junctor.Sources, statement.ID) {
			return writeMutationFailure(stdout, *jsonOutput, profile, "counterpoint_has_support", fmt.Sprintf("counterpoint %s participates in junctor %s; remove its support relation first", statement.ID, junctor.ID), statement.ID)
		}
	}
	for _, support := range next.DirectSupports {
		if support.Source == statement.ID || support.Target == statement.ID {
			return writeMutationFailure(stdout, *jsonOutput, profile, "counterpoint_has_support", fmt.Sprintf("counterpoint %s participates in direct support", statement.ID), statement.ID)
		}
	}
	beforeComponents := len(query.Components(next))
	beforeIsolated := query.IsolatedStatementIDs(next)
	removed := *statement
	removedDefeats := []argument.Defeat{}
	statements := make([]argument.Statement, 0, len(next.Statements)-1)
	for _, candidate := range next.Statements {
		if candidate.ID != statement.ID {
			statements = append(statements, candidate)
		}
	}
	next.Statements = statements
	defeats := make([]argument.Defeat, 0, len(next.Defeats))
	for _, defeat := range next.Defeats {
		if defeat.From == statement.ID {
			removedDefeats = append(removedDefeats, defeat)
		} else {
			defeats = append(defeats, defeat)
		}
	}
	next.Defeats = defeats
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
	for range removedDefeats {
		changes = append(changes, changeOutput{Operation: "removed", ElementType: "defeat", ID: removed.ID})
	}
	changes = appendMetadataChange(changes, metadataChange)
	output := counterpointRemovalOutput{
		SchemaVersion: outputSchemaVersion, Action: "remove-counterpoint", DryRun: *dryRun,
		Profile: profile, Document: documentSummary(next), Counterpoint: removed,
		DefeatsRemoved: removedDefeats, ComponentsBefore: beforeComponents,
		ComponentsAfter: len(query.Components(next)), NewlyIsolated: newlyIsolated,
		Changes: changes, Diagnostics: diagnostics,
	}
	if *jsonOutput {
		return writeIndentedJSON(stdout, output)
	}
	fmt.Fprintf(stdout, "Removed counterpoint %s:%s\n", removed.ID, removed.Slug)
	for _, defeat := range removedDefeats {
		fmt.Fprintf(stdout, "Removed defeat %s\n", formatDefeat(defeat))
	}
	fmt.Fprintf(stdout, "components: %d -> %d\n", output.ComponentsBefore, output.ComponentsAfter)
	if len(output.NewlyIsolated) > 0 {
		fmt.Fprintf(stdout, "newly isolated: %s\n", strings.Join(output.NewlyIsolated, ", "))
	}
	if output.DryRun {
		fmt.Fprintln(stdout, "dry-run: no file changes written")
	}
	return nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func writeUndermineUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cludia undermine [--json] FILE PREMISE --text TEXT")
	fmt.Fprintln(w, "Add a counterpoint challenging the truth or scope of a premise.")
}

func writeUndercutUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cludia undercut [--json] FILE JUNCTOR --text TEXT")
	fmt.Fprintln(w, "Add a counterpoint challenging whether a junctor's sources imply its target.")
}

func writeCounterpointUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cludia counterpoint [--json] FILE COUNTERPOINT --text TEXT")
	fmt.Fprintln(w, "Add a counterpoint targeting another counterpoint.")
}

func writeRemoveCounterpointUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cludia remove-counterpoint [--dry-run] [--json] FILE COUNTERPOINT")
	fmt.Fprintln(w, "Remove a leaf counterpoint and its defeat after reporting structural effects.")
}
