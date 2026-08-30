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

func runChallenge(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("challenge", flag.ContinueOnError)
	fs.SetOutput(stderr)
	flags := addDefeatFlags(fs)
	inferenceRef := fs.String("inference", "", "incoming junctor to challenge when the element has multiple justifications")
	fs.Usage = func() { writeChallengeUsage(fs.Output()) }
	valueFlags := map[string]bool{
		"text": true, "id": true, "slug": true, "truth": true, "kind": true, "inference": true,
	}
	if err := fs.Parse(flagsFirst(args, valueFlags)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 2 || strings.TrimSpace(*flags.text) == "" {
		fs.Usage()
		return fmt.Errorf("challenge expects a file and element and requires --text")
	}

	path, targetRef := fs.Arg(0), fs.Arg(1)
	doc, profile, diagnostics := loadValidated(path)
	if diagnostic.HasErrors(diagnostics) {
		if err := writeFailure(stdout, *flags.jsonOutput, profile, diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	selectedInference := strings.TrimSpace(*inferenceRef)
	resolved, found := doc.ResolveElement(targetRef)
	if found && resolved.Type == argument.ElementJunctor {
		junctor, _ := doc.Junctor(resolved.ID)
		if selectedInference != "" {
			return writeMutationFailure(stdout, *flags.jsonOutput, profile, "challenge_inference_redundant", "--inference is not used when the challenged element is already a junctor", junctor.ID)
		}
		return createDefeatWithDocument(path, doc, profile, diagnostics, "challenge", argument.DefeatInference, junctor.ID, flags, stdout)
	}

	if !found || resolved.Type != argument.ElementStatement {
		return writeMutationFailure(stdout, *flags.jsonOutput, profile, "challenge_target_not_found", fmt.Sprintf("statement or junctor %q not found", targetRef), targetRef)
	}
	statement, _ := doc.Statement(resolved.ID)
	switch statement.Role {
	case argument.RolePremise:
		if selectedInference != "" {
			return writeMutationFailure(stdout, *flags.jsonOutput, profile, "challenge_inference_not_applicable", fmt.Sprintf("premise %s is challenged directly; omit --inference", statement.ID), statement.ID)
		}
		return createDefeatWithDocument(path, doc, profile, diagnostics, "challenge", argument.DefeatPremise, statement.ID, flags, stdout)
	case argument.RoleCounterpoint:
		if selectedInference != "" {
			return writeMutationFailure(stdout, *flags.jsonOutput, profile, "challenge_inference_not_applicable", fmt.Sprintf("counterpoint %s is challenged directly; omit --inference", statement.ID), statement.ID)
		}
		return createDefeatWithDocument(path, doc, profile, diagnostics, "challenge", argument.DefeatCounterpoint, statement.ID, flags, stdout)
	case argument.RoleLemma, argument.RoleConclusion:
		return challengeDerivedStatement(path, statement, selectedInference, doc, profile, flags, stdout)
	default:
		return writeMutationFailure(stdout, *flags.jsonOutput, profile, "challenge_target_role", fmt.Sprintf("statement %s has unsupported role %s", statement.ID, statement.Role), statement.ID)
	}
}

func challengeDerivedStatement(path string, statement *argument.Statement, inferenceRef string, doc *argument.Document, profile validation.Profile, flags defeatFlags, stdout io.Writer) error {
	incoming := make([]argument.Junctor, 0)
	for _, junctor := range doc.Junctors {
		if junctor.Target == statement.ID {
			incoming = append(incoming, junctor)
		}
	}
	directSources := make([]string, 0)
	for _, support := range doc.DirectSupports {
		if support.Target == statement.ID {
			directSources = append(directSources, support.Source)
		}
	}
	if inferenceRef != "" {
		junctor, ok := doc.Junctor(inferenceRef)
		if !ok {
			return writeMutationFailure(stdout, *flags.jsonOutput, profile, "challenge_inference_not_found", fmt.Sprintf("junctor %q not found", inferenceRef), inferenceRef)
		}
		if junctor.Target != statement.ID {
			return writeMutationFailure(stdout, *flags.jsonOutput, profile, "challenge_inference_target_mismatch", fmt.Sprintf("junctor %s targets %s rather than challenged statement %s", junctor.ID, junctor.Target, statement.ID), junctor.ID)
		}
		return createDefeatWithDocument(path, doc, profile, nil, "challenge", argument.DefeatInference, junctor.ID, flags, stdout)
	}
	if len(incoming) == 1 && len(directSources) == 0 {
		return createDefeatWithDocument(path, doc, profile, nil, "challenge", argument.DefeatInference, incoming[0].ID, flags, stdout)
	}
	if len(incoming) == 0 {
		message := fmt.Sprintf("statement %s has no incoming junctor to undercut", statement.ID)
		if len(directSources) > 0 {
			message += fmt.Sprintf("; its legacy direct support from %s has no undercuttable junctor", strings.Join(directSources, ", "))
		}
		return writeMutationFailure(stdout, *flags.jsonOutput, profile, "challenge_no_inference", message, statement.ID)
	}
	if len(incoming) == 1 {
		message := fmt.Sprintf("statement %s has incoming junctor %s plus legacy direct support from %s; rerun with --inference %s to undercut the junctor explicitly", statement.ID, incoming[0].ID, strings.Join(directSources, ", "), incoming[0].ID)
		return writeMutationFailure(stdout, *flags.jsonOutput, profile, "challenge_inference_ambiguous", message, statement.ID)
	}
	ids := make([]string, 0, len(incoming))
	for _, junctor := range incoming {
		ids = append(ids, junctor.ID)
	}
	message := fmt.Sprintf("statement %s has multiple incoming junctors (%s); rerun with --inference JUNCTOR", statement.ID, strings.Join(ids, ", "))
	if len(directSources) > 0 {
		message += fmt.Sprintf("; legacy direct support from %s cannot be undercut directly", strings.Join(directSources, ", "))
	}
	return writeMutationFailure(stdout, *flags.jsonOutput, profile, "challenge_inference_ambiguous", message, statement.ID)
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
	return createDefeatWithDocument(path, doc, profile, diagnostics, action, scope, targetRef, flags, stdout)
}

func createDefeatWithDocument(path string, doc *argument.Document, profile validation.Profile, diagnostics []diagnostic.Diagnostic, action string, scope argument.DefeatScope, targetRef string, flags defeatFlags, stdout io.Writer) error {
	if diagnostic.HasErrors(diagnostics) {
		if err := writeFailure(stdout, *flags.jsonOutput, profile, diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	truth, _ := parseTruth(*flags.truth)
	kind, _ := parseKind(*flags.kind)

	next, result, err := argument.AddDefeat(doc, argument.AddDefeatOptions{
		Scope: scope, TargetRef: targetRef, Text: *flags.text,
		RequestedID: strings.TrimSpace(*flags.id), Slug: strings.TrimSpace(*flags.slug),
		Truth: truth, Kind: kind,
	})
	if err != nil {
		if _, ok := err.(*argument.IDAllocationError); ok {
			return writeIDAllocationFailure(stdout, *flags.jsonOutput, profile, err)
		}
		if addErr, ok := err.(*argument.AddDefeatError); ok {
			return writeMutationFailure(stdout, *flags.jsonOutput, profile, addErr.Failure.Code, addErr.Failure.Message, addErr.Failure.Element)
		}
		return writeArgumentMutationFailure(stdout, *flags.jsonOutput, profile, err)
	}
	validated, err := validateAndPersistMutation(path, next, profile, true)
	if err != nil {
		return err
	}
	diagnostics = validated.Diagnostics
	if !validated.OK() {
		if err := writeFailure(stdout, *flags.jsonOutput, profile, diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	if diagnostics == nil {
		diagnostics = []diagnostic.Diagnostic{}
	}
	previousNextIDs, nextIDsExisted := doc.MetadataValue(argument.NextIDsMetadataKey)
	currentNextIDs, _ := next.MetadataValue(argument.NextIDsMetadataKey)
	var metadataChange *changeOutput
	if !nextIDsExisted || previousNextIDs != currentNextIDs {
		operation := "added"
		if nextIDsExisted {
			operation = "updated"
		}
		change := changeOutput{Operation: operation, ElementType: "metadata", ID: argument.NextIDsMetadataKey}
		metadataChange = &change
	}
	output := defeatMutationOutput{
		SchemaVersion: outputSchemaVersion, Action: action, DryRun: false,
		Profile: profile, Document: documentSummary(next), Counterpoint: result.Counterpoint, Defeat: result.Defeat,
		Changes: appendMetadataChange([]changeOutput{
			{Operation: "added", ElementType: "statement", ID: result.Counterpoint.ID},
			{Operation: "added", ElementType: "defeat", ID: result.Counterpoint.ID},
		}, metadataChange),
		Diagnostics: diagnostics,
	}
	if *flags.jsonOutput {
		return writeIndentedJSON(stdout, output)
	}
	fmt.Fprintf(stdout, "Added %s %s:%s\n%s\n", action, result.Counterpoint.ID, result.Counterpoint.Slug, result.Counterpoint.Text)
	fmt.Fprintf(stdout, "%s\n", formatDefeat(result.Defeat))
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
	beforeComponents := len(query.Components(doc))
	beforeIsolated := query.IsolatedStatementIDs(doc)
	next, result, err := argument.RemoveCounterpoint(doc, fs.Arg(1))
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
	changes := []changeOutput{{Operation: "removed", ElementType: "statement", ID: result.Counterpoint.ID}}
	for range result.DefeatsRemoved {
		changes = append(changes, changeOutput{Operation: "removed", ElementType: "defeat", ID: result.Counterpoint.ID})
	}
	changes = appendMetadataChange(changes, nextIDsMetadataChange(doc, next))
	output := counterpointRemovalOutput{
		SchemaVersion: outputSchemaVersion, Action: "remove-counterpoint", DryRun: *dryRun,
		Profile: profile, Document: documentSummary(next), Counterpoint: result.Counterpoint,
		DefeatsRemoved: result.DefeatsRemoved, ComponentsBefore: beforeComponents,
		ComponentsAfter: len(query.Components(next)), NewlyIsolated: newlyIsolated,
		Changes: changes, Diagnostics: diagnostics,
	}
	if *jsonOutput {
		return writeIndentedJSON(stdout, output)
	}
	fmt.Fprintf(stdout, "Removed counterpoint %s:%s\n", result.Counterpoint.ID, result.Counterpoint.Slug)
	for _, defeat := range result.DefeatsRemoved {
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

func writeUndermineUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cludia undermine [--json] FILE PREMISE --text TEXT")
	fmt.Fprintln(w, "Add a counterpoint challenging the truth or scope of a premise.")
}

func writeUndercutUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cludia undercut [--json] FILE JUNCTOR --text TEXT")
	fmt.Fprintln(w, "Add a counterpoint challenging whether a junctor's sources imply its target.")
}

func writeChallengeUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cludia challenge [--json] FILE ELEMENT --text TEXT [--inference JUNCTOR]")
	fmt.Fprintln(w, "Challenge a premise, counterpoint, junctor, or a derived statement's selected incoming junctor without changing defeat semantics.")
	fmt.Fprintln(w, "A derived statement with multiple incoming junctors requires --inference; legacy direct support cannot be undercut directly.")
}

func writeCounterpointUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cludia counterpoint [--json] FILE COUNTERPOINT --text TEXT")
	fmt.Fprintln(w, "Add a counterpoint targeting another counterpoint.")
}

func writeRemoveCounterpointUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cludia remove-counterpoint [--dry-run] [--json] FILE COUNTERPOINT")
	fmt.Fprintln(w, "Remove a leaf counterpoint and its defeat after reporting structural effects.")
}
