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
	action := "remove-source"
	mode := argument.SourceRemove
	if add {
		action = "add-source"
		mode = argument.SourceAdd
	}
	next, result, err := argument.RepairJunctor(doc, argument.RepairJunctorOptions{
		JunctorID: junctorID, Mode: mode, SourceRef: sourceRef,
	})
	if err != nil {
		return writeArgumentMutationFailure(stdout, jsonOutput, profile, err)
	}
	validated, err := validateAndPersistMutation(path, next, profile, !dryRun)
	if err != nil {
		return err
	}
	if !validated.OK() {
		if err := writeFailure(stdout, jsonOutput, profile, validated.Diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	diagnostics = validated.Diagnostics
	if diagnostics == nil {
		diagnostics = []diagnostic.Diagnostic{}
	}
	current := result.Current
	output := junctorMutationOutput{
		SchemaVersion: outputSchemaVersion, Action: action, DryRun: dryRun,
		Profile: profile, Document: documentSummary(next), Junctor: &current,
		PreviousJunctor: result.Previous, Changes: []changeOutput{{Operation: "updated", ElementType: "junctor", ID: result.Current.ID}},
		Diagnostics: diagnostics,
	}
	if add {
		output.SourceAdded = result.SourceAdded
	} else {
		output.SourceRemoved = result.SourceRemoved
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
	next, result, err := argument.RepairJunctor(doc, argument.RepairJunctorOptions{
		JunctorID: junctorID, Mode: argument.SourceReplace, FromRef: fromRef, ToRef: toRef,
	})
	if err != nil {
		return writeArgumentMutationFailure(stdout, jsonOutput, profile, err)
	}
	validated, err := validateAndPersistMutation(path, next, profile, !dryRun)
	if err != nil {
		return err
	}
	if !validated.OK() {
		if err := writeFailure(stdout, jsonOutput, profile, validated.Diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	diagnostics = validated.Diagnostics
	if diagnostics == nil {
		diagnostics = []diagnostic.Diagnostic{}
	}
	current := result.Current
	output := junctorMutationOutput{
		SchemaVersion: outputSchemaVersion, Action: "replace-source", DryRun: dryRun,
		Profile: profile, Document: documentSummary(next), Junctor: &current,
		PreviousJunctor: result.Previous, SourceAdded: result.SourceAdded, SourceRemoved: result.SourceRemoved,
		Changes:     []changeOutput{{Operation: "updated", ElementType: "junctor", ID: result.Current.ID}},
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
	next, result, err := argument.RemoveJunctor(doc, fs.Arg(1))
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
	diagnostics = validated.Diagnostics
	if diagnostics == nil {
		diagnostics = []diagnostic.Diagnostic{}
	}
	output := junctorMutationOutput{
		SchemaVersion: outputSchemaVersion, Action: "remove-junctor", DryRun: *dryRun,
		Profile: profile, Document: documentSummary(next), PreviousJunctor: result.Previous,
		Changes: appendMetadataChange(
			[]changeOutput{{Operation: "removed", ElementType: "junctor", ID: result.Previous.ID}},
			nextIDsMetadataChange(doc, next),
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
