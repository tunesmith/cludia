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

type stringListFlag []string

func (f *stringListFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *stringListFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

type roleChangeOutput struct {
	PreviousID string        `json:"previous_id"`
	CurrentID  string        `json:"current_id"`
	From       argument.Role `json:"from"`
	To         argument.Role `json:"to"`
}

type deriveOutput struct {
	SchemaVersion int                     `json:"schema_version"`
	Action        string                  `json:"action"`
	DryRun        bool                    `json:"dry_run"`
	Profile       validation.Profile      `json:"profile"`
	Document      documentOutput          `json:"document"`
	Target        argument.Statement      `json:"target"`
	Junctor       argument.Junctor        `json:"junctor"`
	RoleChanges   []roleChangeOutput      `json:"role_changes"`
	Changes       []changeOutput          `json:"changes"`
	Diagnostics   []diagnostic.Diagnostic `json:"diagnostics"`
}

func runDerive(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("derive", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOutput := fs.Bool("json", false, "output versioned JSON")
	var sources stringListFlag
	fs.Var(&sources, "source", "source statement id or slug (repeat at least twice)")
	targetRef := fs.String("target", "", "existing target statement id or slug")
	targetText := fs.String("target-text", "", "text for a new lemma target")
	targetID := fs.String("target-id", "", "exact next role-appropriate id for a new target (generated if omitted)")
	targetSlug := fs.String("target-slug", "", "slug for a new target (generated if omitted)")
	targetKind := fs.String("target-kind", "fact", "kind for a new target: fact or value")
	targetRole := fs.String("target-role", "lemma", "role for a new target: lemma or conclusion")
	junctorID := fs.String("junctor-id", "", "exact next J id (generated if omitted)")
	fs.Usage = func() { writeDeriveUsage(fs.Output()) }
	valueFlags := map[string]bool{
		"source": true, "target": true, "target-text": true, "target-id": true,
		"target-slug": true, "target-kind": true, "target-role": true, "junctor-id": true,
	}
	if err := fs.Parse(flagsFirst(args, valueFlags)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("derive expects exactly one file")
	}
	if len(sources) < 2 {
		fs.Usage()
		return fmt.Errorf("derive requires at least two --source values")
	}
	cleanTargetRef := strings.TrimSpace(*targetRef)
	cleanTargetText := strings.TrimSpace(*targetText)
	if (cleanTargetRef == "") == (cleanTargetText == "") {
		fs.Usage()
		return fmt.Errorf("derive requires exactly one of --target or --target-text")
	}
	if cleanTargetRef != "" && (strings.TrimSpace(*targetID) != "" || strings.TrimSpace(*targetSlug) != "" || *targetKind != "fact" || *targetRole != "lemma") {
		return fmt.Errorf("--target-id, --target-slug, --target-kind, and --target-role apply only with --target-text")
	}

	doc, profile, diagnostics := loadValidated(fs.Arg(0))
	if diagnostic.HasErrors(diagnostics) {
		if err := writeFailure(stdout, *jsonOutput, profile, diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	options := argument.DeriveOptions{
		SourceRefs: append([]string(nil), sources...), ExistingTargetRef: cleanTargetRef,
		RequestedJunctorID: strings.TrimSpace(*junctorID),
	}
	if cleanTargetRef == "" {
		kind, ok := parseKind(*targetKind)
		if !ok {
			diagnostics := diagnosticError("kind_invalid", fmt.Sprintf("invalid target kind %q; expected fact or value", *targetKind), strings.TrimSpace(*targetID))
			if err := writeFailure(stdout, *jsonOutput, profile, diagnostics); err != nil {
				return err
			}
			return errValidationFailed
		}
		role, ok := parseDerivedRole(*targetRole)
		if !ok {
			diagnostics := diagnosticError("role_invalid", fmt.Sprintf("invalid target role %q; expected lemma or conclusion", *targetRole), strings.TrimSpace(*targetID))
			if err := writeFailure(stdout, *jsonOutput, profile, diagnostics); err != nil {
				return err
			}
			return errValidationFailed
		}
		options.NewTarget = &argument.NewDerivedTarget{
			Text: cleanTargetText, RequestedID: strings.TrimSpace(*targetID), Slug: strings.TrimSpace(*targetSlug), Kind: kind, Role: role,
		}
	}

	next, result, err := argument.Derive(doc, options)
	if err != nil {
		if _, ok := err.(*argument.IDAllocationError); ok {
			return writeIDAllocationFailure(stdout, *jsonOutput, profile, err)
		}
		if deriveErr, ok := err.(*argument.DeriveError); ok {
			failures := make([]diagnostic.Diagnostic, 0, len(deriveErr.Failures))
			for _, failure := range deriveErr.Failures {
				failures = append(failures, diagnostic.Diagnostic{
					Code: failure.Code, Message: failure.Message, Severity: diagnostic.SeverityError, Element: failure.Element,
				})
			}
			if writeErr := writeFailure(stdout, *jsonOutput, profile, failures); writeErr != nil {
				return writeErr
			}
			return errValidationFailed
		}
		return err
	}

	roleChanges := make([]roleChangeOutput, 0, len(result.RoleChanges))
	changes := make([]changeOutput, 0)
	if cleanTargetRef == "" {
		changes = append(changes, changeOutput{Operation: "added", ElementType: "statement", ID: result.Target.ID})
	}
	for _, change := range result.RoleChanges {
		roleChanges = append(roleChanges, roleChangeOutput{
			PreviousID: change.PreviousID, CurrentID: change.CurrentID, From: change.From, To: change.To,
		})
		changes = append(changes, changeOutput{Operation: "reidentified", ElementType: "statement", ID: change.CurrentID})
	}
	changes = append(changes, changeOutput{Operation: "added", ElementType: "junctor", ID: result.Junctor.ID})
	if result.RootMetadataUpdated {
		changes = append(changes, changeOutput{Operation: "updated", ElementType: "metadata", ID: "root"})
	}
	previousNextIDs, nextIDsExisted := doc.MetadataValue(argument.NextIDsMetadataKey)
	currentNextIDs, _ := next.MetadataValue(argument.NextIDsMetadataKey)
	if !nextIDsExisted || previousNextIDs != currentNextIDs {
		operation := "added"
		if nextIDsExisted {
			operation = "updated"
		}
		changes = append(changes, changeOutput{Operation: operation, ElementType: "metadata", ID: argument.NextIDsMetadataKey})
	}
	validated, err := validateAndPersistMutation(fs.Arg(0), next, profile, true)
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
	for _, change := range result.RoleChanges {
		diagnostics = append(diagnostics, diagnostic.Diagnostic{
			Code:     "external_id_references_unchecked",
			Message:  fmt.Sprintf("promoting %s to lemma %s rewrote references in this workspace and recognized root metadata only; references in conversations, Markdown, scripts, other workspaces, prior exports, or published graphs may require review", change.PreviousID, change.CurrentID),
			Severity: diagnostic.SeverityWarning, Element: change.CurrentID,
		})
	}
	output := deriveOutput{
		SchemaVersion: outputSchemaVersion, Action: "derive", DryRun: false,
		Profile: profile, Document: documentSummary(next), Target: result.Target, Junctor: result.Junctor,
		RoleChanges: roleChanges, Changes: changes, Diagnostics: diagnostics,
	}
	if *jsonOutput {
		return writeIndentedJSON(stdout, output)
	}
	writeHumanDerive(stdout, output)
	return nil
}

func parseDerivedRole(value string) (argument.Role, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "lemma":
		return argument.RoleLemma, true
	case "conclusion":
		return argument.RoleConclusion, true
	default:
		return argument.Role(value), false
	}
}

func writeHumanDerive(w io.Writer, output deriveOutput) {
	fmt.Fprintf(w, "Derived %s:%s\n", output.Target.ID, output.Target.Slug)
	fmt.Fprintf(w, "%s\n", output.Target.Text)
	fmt.Fprintf(w, "AND#%s(%s) -> %s\n", output.Junctor.ID, strings.Join(output.Junctor.Sources, ", "), output.Junctor.Target)
	for _, change := range output.RoleChanges {
		fmt.Fprintf(w, "Promoted %s -> %s: %s -> %s\n", change.PreviousID, change.CurrentID, change.From, change.To)
	}
	for _, item := range output.Diagnostics {
		writeDiagnostic(w, item)
	}
}

func writeDeriveUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cludia derive [--json] FILE --source STATEMENT --source STATEMENT (--target STATEMENT | --target-text TEXT)")
	fmt.Fprintln(w, "Create a multi-premise AND inference, optionally creating a lemma or conclusion target.")
	fmt.Fprintln(w, "An existing premise target is promoted with the next L ID; consume current_id from role_changes.")
}
