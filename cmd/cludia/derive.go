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
	ID   string        `json:"id"`
	From argument.Role `json:"from"`
	To   argument.Role `json:"to"`
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
	next := doc.Clone()
	allocator, err := argument.NewIDAllocator(next)
	if err != nil {
		return err
	}
	resolvedSources := make([]string, 0, len(sources))
	for _, reference := range sources {
		statement, ok := next.Statement(strings.TrimSpace(reference))
		if !ok {
			diagnostics = append(diagnostics, diagnostic.Diagnostic{
				Code: "source_not_found", Message: fmt.Sprintf("source statement %q not found", reference),
				Severity: diagnostic.SeverityError, Element: reference,
			})
			continue
		}
		resolvedSources = append(resolvedSources, statement.ID)
	}
	if diagnostic.HasErrors(diagnostics) {
		if err := writeFailure(stdout, *jsonOutput, profile, diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}

	roleChanges := []roleChangeOutput{}
	changes := []changeOutput{}
	var target *argument.Statement
	if cleanTargetRef != "" {
		var ok bool
		target, ok = next.Statement(cleanTargetRef)
		if !ok {
			diagnostics := diagnosticError("target_not_found", fmt.Sprintf("target statement %q not found", cleanTargetRef), cleanTargetRef)
			if err := writeFailure(stdout, *jsonOutput, profile, diagnostics); err != nil {
				return err
			}
			return errValidationFailed
		}
		if target.Role == argument.RolePremise {
			roleChanges = append(roleChanges, roleChangeOutput{ID: target.ID, From: target.Role, To: argument.RoleLemma})
			target.Role = argument.RoleLemma
			target.Truth = argument.TruthUnknown
			changes = append(changes, changeOutput{Operation: "updated", ElementType: "statement", ID: target.ID})
		}
	} else {
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
		id, allocationErr := allocator.Statement(role, strings.TrimSpace(*targetID))
		if allocationErr != nil {
			return writeIDAllocationFailure(stdout, *jsonOutput, profile, allocationErr)
		}
		slug := strings.TrimSpace(*targetSlug)
		if slug == "" {
			slug = argument.UniqueSlug(next, cleanTargetText)
		}
		next.Statements = append(next.Statements, argument.Statement{
			ID: id, Slug: slug, Role: role, Kind: kind,
			Truth: argument.TruthUnknown, Text: cleanTargetText,
		})
		target = &next.Statements[len(next.Statements)-1]
		changes = append(changes, changeOutput{Operation: "added", ElementType: "statement", ID: target.ID})
	}

	id, allocationErr := allocator.Junctor(strings.TrimSpace(*junctorID))
	if allocationErr != nil {
		return writeIDAllocationFailure(stdout, *jsonOutput, profile, allocationErr)
	}
	junctor := argument.Junctor{
		ID: id, Connector: argument.ConnectorAND,
		Sources: append([]string(nil), resolvedSources...), Target: target.ID,
		Order: nextRelationOrder(next),
	}
	next.Junctors = append(next.Junctors, junctor)
	changes = append(changes, changeOutput{Operation: "added", ElementType: "junctor", ID: junctor.ID})
	changes = appendMetadataChange(changes, persistIDAllocator(next, allocator))
	validated := validation.Validate(next, profile)
	if !validated.OK() {
		if err := writeFailure(stdout, *jsonOutput, profile, validated.Diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	if err := argfile.SaveAtomic(fs.Arg(0), next); err != nil {
		return err
	}
	diagnostics = validated.Diagnostics
	if diagnostics == nil {
		diagnostics = []diagnostic.Diagnostic{}
	}
	output := deriveOutput{
		SchemaVersion: outputSchemaVersion, Action: "derive", DryRun: false,
		Profile: profile, Document: documentSummary(next), Target: *target, Junctor: junctor,
		RoleChanges: roleChanges, Changes: changes, Diagnostics: diagnostics,
	}
	if *jsonOutput {
		return writeIndentedJSON(stdout, output)
	}
	writeHumanDerive(stdout, output)
	return nil
}

func nextRelationOrder(doc *argument.Document) int {
	maximum := 0
	for _, junctor := range doc.Junctors {
		if junctor.Order > maximum {
			maximum = junctor.Order
		}
	}
	for _, support := range doc.DirectSupports {
		if support.Order > maximum {
			maximum = support.Order
		}
	}
	return maximum + 1
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
		fmt.Fprintf(w, "Promoted %s: %s -> %s\n", change.ID, change.From, change.To)
	}
	for _, item := range output.Diagnostics {
		writeDiagnostic(w, item)
	}
}

func writeDeriveUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cludia derive [--json] FILE --source STATEMENT --source STATEMENT (--target STATEMENT | --target-text TEXT)")
	fmt.Fprintln(w, "Create a multi-premise AND inference, optionally creating a lemma or conclusion target.")
}
