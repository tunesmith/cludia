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

type junctorMutationOutput struct {
	SchemaVersion   int                     `json:"schema_version"`
	Action          string                  `json:"action"`
	DryRun          bool                    `json:"dry_run"`
	Profile         validation.Profile      `json:"profile"`
	Document        documentOutput          `json:"document"`
	Junctor         *argument.Junctor       `json:"junctor,omitempty"`
	PreviousJunctor argument.Junctor        `json:"previous_junctor"`
	SourceAdded     string                  `json:"source_added,omitempty"`
	SourceRemoved   string                  `json:"source_removed,omitempty"`
	Changes         []changeOutput          `json:"changes"`
	Diagnostics     []diagnostic.Diagnostic `json:"diagnostics"`
}

func runAddSource(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("add-source", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOutput := fs.Bool("json", false, "output versioned JSON")
	sourceRef := fs.String("source", "", "statement id or slug to append as a source")
	fs.Usage = func() { writeAddSourceUsage(fs.Output()) }
	if err := fs.Parse(flagsFirst(args, map[string]bool{"source": true})); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 2 || strings.TrimSpace(*sourceRef) == "" {
		fs.Usage()
		return fmt.Errorf("add-source expects a file and junctor and requires --source")
	}
	return mutateJunctorSource(fs.Arg(0), fs.Arg(1), *sourceRef, true, false, *jsonOutput, stdout)
}

func runRemoveSource(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("remove-source", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOutput := fs.Bool("json", false, "output versioned JSON")
	dryRun := fs.Bool("dry-run", false, "validate and report without saving")
	sourceRef := fs.String("source", "", "statement id or slug to remove as a source")
	fs.Usage = func() { writeRemoveSourceUsage(fs.Output()) }
	if err := fs.Parse(flagsFirst(args, map[string]bool{"source": true})); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 2 || strings.TrimSpace(*sourceRef) == "" {
		fs.Usage()
		return fmt.Errorf("remove-source expects a file and junctor and requires --source")
	}
	return mutateJunctorSource(fs.Arg(0), fs.Arg(1), *sourceRef, false, *dryRun, *jsonOutput, stdout)
}

func runReplaceSource(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("replace-source", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOutput := fs.Bool("json", false, "output versioned JSON")
	dryRun := fs.Bool("dry-run", false, "validate and report without saving")
	fromRef := fs.String("from", "", "current source statement id or slug")
	toRef := fs.String("to", "", "replacement source statement id or slug")
	fs.Usage = func() { writeReplaceSourceUsage(fs.Output()) }
	if err := fs.Parse(flagsFirst(args, map[string]bool{"from": true, "to": true})); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 2 || strings.TrimSpace(*fromRef) == "" || strings.TrimSpace(*toRef) == "" {
		fs.Usage()
		return fmt.Errorf("replace-source expects a file and junctor and requires --from and --to")
	}
	return replaceJunctorSource(fs.Arg(0), fs.Arg(1), *fromRef, *toRef, *dryRun, *jsonOutput, stdout)
}

func mutateJunctorSource(path, junctorID, sourceRef string, add, dryRun, jsonOutput bool, stdout io.Writer) error {
	doc, profile, diagnostics := loadValidated(path)
	if diagnostic.HasErrors(diagnostics) {
		if err := writeFailure(stdout, jsonOutput, profile, diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	next := doc.Clone()
	junctor, ok := next.Junctor(junctorID)
	if !ok {
		return writeMutationFailure(stdout, jsonOutput, profile, "junctor_not_found", fmt.Sprintf("junctor %q not found", junctorID), junctorID)
	}
	if junctor.Connector != argument.ConnectorAND {
		return writeMutationFailure(stdout, jsonOutput, profile, "junctor_not_editable", fmt.Sprintf("focused repair edits only AND junctors; %s uses %s", junctor.ID, junctor.Connector), junctor.ID)
	}
	source, ok := next.Statement(strings.TrimSpace(sourceRef))
	if !ok {
		return writeMutationFailure(stdout, jsonOutput, profile, "source_not_found", fmt.Sprintf("source statement %q not found", sourceRef), sourceRef)
	}
	previous := copyJunctor(*junctor)
	action := "remove-source"
	if add {
		action = "add-source"
		junctor.Sources = append(junctor.Sources, source.ID)
	} else {
		index := -1
		for i, id := range junctor.Sources {
			if id == source.ID {
				index = i
				break
			}
		}
		if index < 0 {
			return writeMutationFailure(stdout, jsonOutput, profile, "junctor_source_not_found", fmt.Sprintf("statement %s is not a source of junctor %s", source.ID, junctor.ID), source.ID)
		}
		junctor.Sources = append(junctor.Sources[:index], junctor.Sources[index+1:]...)
	}
	validated := validation.Validate(next, profile)
	if !validated.OK() {
		if err := writeFailure(stdout, jsonOutput, profile, validated.Diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	if !dryRun {
		if err := argfile.SaveAtomic(path, next); err != nil {
			return err
		}
	}
	diagnostics = validated.Diagnostics
	if diagnostics == nil {
		diagnostics = []diagnostic.Diagnostic{}
	}
	current := copyJunctor(*junctor)
	output := junctorMutationOutput{
		SchemaVersion: outputSchemaVersion, Action: action, DryRun: dryRun,
		Profile: profile, Document: documentSummary(next), Junctor: &current,
		PreviousJunctor: previous, Changes: []changeOutput{{Operation: "updated", ElementType: "junctor", ID: junctor.ID}},
		Diagnostics: diagnostics,
	}
	if add {
		output.SourceAdded = source.ID
	} else {
		output.SourceRemoved = source.ID
	}
	return writeJunctorMutation(stdout, jsonOutput, output)
}

func replaceJunctorSource(path, junctorID, fromRef, toRef string, dryRun, jsonOutput bool, stdout io.Writer) error {
	doc, profile, diagnostics := loadValidated(path)
	if diagnostic.HasErrors(diagnostics) {
		if err := writeFailure(stdout, jsonOutput, profile, diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	next := doc.Clone()
	junctor, ok := next.Junctor(junctorID)
	if !ok {
		return writeMutationFailure(stdout, jsonOutput, profile, "junctor_not_found", fmt.Sprintf("junctor %q not found", junctorID), junctorID)
	}
	if junctor.Connector != argument.ConnectorAND {
		return writeMutationFailure(stdout, jsonOutput, profile, "junctor_not_editable", fmt.Sprintf("focused repair edits only AND junctors; %s uses %s", junctor.ID, junctor.Connector), junctor.ID)
	}
	from, ok := next.Statement(strings.TrimSpace(fromRef))
	if !ok {
		return writeMutationFailure(stdout, jsonOutput, profile, "source_not_found", fmt.Sprintf("source statement %q not found", fromRef), fromRef)
	}
	to, ok := next.Statement(strings.TrimSpace(toRef))
	if !ok {
		return writeMutationFailure(stdout, jsonOutput, profile, "source_not_found", fmt.Sprintf("source statement %q not found", toRef), toRef)
	}
	index := -1
	for i, id := range junctor.Sources {
		if id == from.ID {
			index = i
			break
		}
	}
	if index < 0 {
		return writeMutationFailure(stdout, jsonOutput, profile, "junctor_source_not_found", fmt.Sprintf("statement %s is not a source of junctor %s", from.ID, junctor.ID), from.ID)
	}
	if from.ID == to.ID {
		return writeMutationFailure(stdout, jsonOutput, profile, "source_replacement_same_statement", fmt.Sprintf("replacement source for junctor %s must differ from %s", junctor.ID, from.ID), from.ID)
	}
	for _, id := range junctor.Sources {
		if id == to.ID {
			return writeMutationFailure(stdout, jsonOutput, profile, "junctor_source_duplicate", fmt.Sprintf("statement %s is already a source of junctor %s", to.ID, junctor.ID), to.ID)
		}
	}
	previous := copyJunctor(*junctor)
	junctor.Sources[index] = to.ID
	validated := validation.Validate(next, profile)
	if !validated.OK() {
		if err := writeFailure(stdout, jsonOutput, profile, validated.Diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	if !dryRun {
		if err := argfile.SaveAtomic(path, next); err != nil {
			return err
		}
	}
	diagnostics = validated.Diagnostics
	if diagnostics == nil {
		diagnostics = []diagnostic.Diagnostic{}
	}
	current := copyJunctor(*junctor)
	output := junctorMutationOutput{
		SchemaVersion: outputSchemaVersion, Action: "replace-source", DryRun: dryRun,
		Profile: profile, Document: documentSummary(next), Junctor: &current,
		PreviousJunctor: previous, SourceAdded: to.ID, SourceRemoved: from.ID,
		Changes:     []changeOutput{{Operation: "updated", ElementType: "junctor", ID: junctor.ID}},
		Diagnostics: diagnostics,
	}
	return writeJunctorMutation(stdout, jsonOutput, output)
}

func runRemoveJunctor(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("remove-junctor", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOutput := fs.Bool("json", false, "output versioned JSON")
	dryRun := fs.Bool("dry-run", false, "validate and report without saving")
	fs.Usage = func() { writeRemoveJunctorUsage(fs.Output()) }
	if err := fs.Parse(flagsFirst(args, map[string]bool{})); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 2 {
		fs.Usage()
		return fmt.Errorf("remove-junctor expects a file and junctor")
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
	junctor, ok := next.Junctor(fs.Arg(1))
	if !ok {
		return writeMutationFailure(stdout, *jsonOutput, profile, "junctor_not_found", fmt.Sprintf("junctor %q not found", fs.Arg(1)), fs.Arg(1))
	}
	for _, defeat := range next.Defeats {
		if defeat.Scope == argument.DefeatInference && defeat.JunctorID == junctor.ID {
			return writeMutationFailure(stdout, *jsonOutput, profile, "junctor_has_undercuts", fmt.Sprintf("junctor %s is targeted by undercut %s; remove the counterpoint first", junctor.ID, defeat.From), junctor.ID)
		}
	}
	previous := copyJunctor(*junctor)
	for i := range next.Junctors {
		if next.Junctors[i].ID == junctor.ID {
			next.Junctors = append(next.Junctors[:i], next.Junctors[i+1:]...)
			break
		}
	}
	validated := validation.Validate(next, profile)
	if !validated.OK() {
		if err := writeFailure(stdout, *jsonOutput, profile, validated.Diagnostics); err != nil {
			return err
		}
		return errValidationFailed
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
	output := junctorMutationOutput{
		SchemaVersion: outputSchemaVersion, Action: "remove-junctor", DryRun: *dryRun,
		Profile: profile, Document: documentSummary(next), PreviousJunctor: previous,
		Changes: appendMetadataChange(
			[]changeOutput{{Operation: "removed", ElementType: "junctor", ID: previous.ID}},
			metadataChange,
		),
		Diagnostics: diagnostics,
	}
	return writeJunctorMutation(stdout, *jsonOutput, output)
}

func writeMutationFailure(w io.Writer, jsonOutput bool, profile validation.Profile, code, message, element string) error {
	diagnostics := diagnosticError(code, message, element)
	if err := writeFailure(w, jsonOutput, profile, diagnostics); err != nil {
		return err
	}
	return errValidationFailed
}

func copyJunctor(junctor argument.Junctor) argument.Junctor {
	junctor.Sources = append([]string(nil), junctor.Sources...)
	return junctor
}

func writeJunctorMutation(w io.Writer, jsonOutput bool, output junctorMutationOutput) error {
	if jsonOutput {
		return writeIndentedJSON(w, output)
	}
	switch output.Action {
	case "add-source":
		fmt.Fprintf(w, "Added source %s to %s\n", output.SourceAdded, output.PreviousJunctor.ID)
	case "remove-source":
		fmt.Fprintf(w, "Removed source %s from %s\n", output.SourceRemoved, output.PreviousJunctor.ID)
	case "replace-source":
		fmt.Fprintf(w, "Replaced source %s with %s in %s\n", output.SourceRemoved, output.SourceAdded, output.PreviousJunctor.ID)
	case "remove-junctor":
		fmt.Fprintf(w, "Removed junctor %s\n", output.PreviousJunctor.ID)
	}
	if output.DryRun {
		fmt.Fprintln(w, "dry-run: no file changes written")
	}
	if output.Junctor != nil {
		fmt.Fprintf(w, "%s#%s(%s) -> %s\n", output.Junctor.Connector, output.Junctor.ID, strings.Join(output.Junctor.Sources, ", "), output.Junctor.Target)
	}
	for _, item := range output.Diagnostics {
		writeDiagnostic(w, item)
	}
	return nil
}

func writeAddSourceUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cludia add-source [--json] FILE JUNCTOR --source STATEMENT")
	fmt.Fprintln(w, "Append an existing statement to an AND junctor after validating the complete workspace.")
}

func writeRemoveSourceUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cludia remove-source [--dry-run] [--json] FILE JUNCTOR --source STATEMENT")
	fmt.Fprintln(w, "Remove a source when at least two distinct sources remain valid.")
}

func writeReplaceSourceUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cludia replace-source [--dry-run] [--json] FILE JUNCTOR --from STATEMENT --to STATEMENT")
	fmt.Fprintln(w, "Replace one source in place within an AND junctor after validating the complete workspace.")
}

func writeRemoveJunctorUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cludia remove-junctor [--dry-run] [--json] FILE JUNCTOR")
	fmt.Fprintln(w, "Remove an AND or OR junctor unless an inference undercut still targets it.")
}
