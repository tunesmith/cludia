package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/tunesmith/cludia/internal/argument"
	"github.com/tunesmith/cludia/internal/diagnostic"
	"github.com/tunesmith/cludia/internal/evaluation"
	"github.com/tunesmith/cludia/internal/query"
	"github.com/tunesmith/cludia/internal/validation"
)

type rootedOutput struct {
	SchemaVersion int                     `json:"schema_version"`
	Profile       validation.Profile      `json:"profile"`
	Root          string                  `json:"root"`
	Exportable    bool                    `json:"exportable"`
	Document      *argument.Document      `json:"document"`
	Stats         statsOutput             `json:"stats"`
	Evaluation    evaluation.Result       `json:"evaluation"`
	Diagnostics   []diagnostic.Diagnostic `json:"diagnostics"`
}

type exportOutput struct {
	SchemaVersion int                     `json:"schema_version"`
	Action        string                  `json:"action"`
	Profile       validation.Profile      `json:"profile"`
	Root          string                  `json:"root"`
	Output        string                  `json:"output"`
	Written       bool                    `json:"written"`
	Document      documentOutput          `json:"document"`
	Stats         statsOutput             `json:"stats"`
	Diagnostics   []diagnostic.Diagnostic `json:"diagnostics"`
}

func runRoot(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("root", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOutput := fs.Bool("json", false, "output versioned JSON")
	fs.Usage = func() { writeRootUsage(fs.Output()) }
	if err := fs.Parse(flagsFirst(args, map[string]bool{})); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 2 {
		fs.Usage()
		return fmt.Errorf("root expects a file and root statement")
	}
	doc, profile, diagnostics := loadValidated(fs.Arg(0))
	if diagnostic.HasErrors(diagnostics) {
		if err := writeFailure(stdout, *jsonOutput, profile, diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	rooted, err := query.Rooted(doc, fs.Arg(1))
	if err != nil {
		return writeMutationFailure(stdout, *jsonOutput, profile, "root_invalid", err.Error(), fs.Arg(1))
	}
	validated, err := validateAndPersistMutation("", rooted, validation.ProfileConcludia, false)
	if err != nil {
		return err
	}
	diagnostics = validated.Diagnostics
	if diagnostics == nil {
		diagnostics = []diagnostic.Diagnostic{}
	}
	evaluated, evaluationDiagnostics := evaluateDocument(rooted)
	if diagnostic.HasErrors(evaluationDiagnostics) {
		if err := writeFailure(stdout, *jsonOutput, validation.ProfileConcludia, evaluationDiagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	root, _ := rooted.Statement(fs.Arg(1))
	rootID := fs.Arg(1)
	if root != nil {
		rootID = root.ID
	}
	output := rootedOutput{
		SchemaVersion: outputSchemaVersion, Profile: validation.ProfileConcludia,
		Root: rootID, Exportable: validated.OK(), Document: rooted,
		Stats: documentStats(rooted), Evaluation: evaluated, Diagnostics: diagnostics,
	}
	if *jsonOutput {
		return writeIndentedJSON(stdout, output)
	}
	writeHumanRooted(stdout, output)
	return nil
}

func runExport(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOutput := fs.Bool("json", false, "output versioned JSON")
	rootRef := fs.String("root", "", "root statement id or slug")
	outputPath := fs.String("output", "", "new Concludia .arg output path")
	documentID := fs.String("id", "", "optional exported document id")
	title := fs.String("title", "", "optional exported document title")
	fs.Usage = func() { writeExportUsage(fs.Output()) }
	if err := fs.Parse(flagsFirst(args, map[string]bool{"root": true, "output": true, "id": true, "title": true})); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 1 || strings.TrimSpace(*rootRef) == "" || strings.TrimSpace(*outputPath) == "" {
		fs.Usage()
		return fmt.Errorf("export expects one workspace and requires --root and --output")
	}
	doc, profile, diagnostics := loadValidated(fs.Arg(0))
	if diagnostic.HasErrors(diagnostics) {
		if err := writeFailure(stdout, *jsonOutput, profile, diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	rooted, err := query.Rooted(doc, *rootRef)
	if err != nil {
		return writeMutationFailure(stdout, *jsonOutput, profile, "root_invalid", err.Error(), *rootRef)
	}
	rooted, err = argument.WithDocumentIdentity(rooted, argument.DocumentIdentityOptions{ID: *documentID, Title: *title})
	if err != nil {
		return writeArgumentMutationFailure(stdout, *jsonOutput, profile, err)
	}
	root, _ := rooted.Statement(*rootRef)
	rootID := *rootRef
	if root != nil {
		rootID = root.ID
	}
	validated, err := validateAndPersistMutation("", rooted, validation.ProfileConcludia, false)
	if err != nil {
		return err
	}
	diagnostics = validated.Diagnostics
	if diagnostics == nil {
		diagnostics = []diagnostic.Diagnostic{}
	}
	output := exportOutput{
		SchemaVersion: outputSchemaVersion, Action: "export", Profile: validation.ProfileConcludia,
		Root: rootID, Output: *outputPath, Written: false,
		Document: documentSummary(rooted), Stats: documentStats(rooted), Diagnostics: diagnostics,
	}
	if !validated.OK() {
		if *jsonOutput {
			if err := writeIndentedJSON(stdout, output); err != nil {
				return err
			}
		} else {
			writeHumanExport(stdout, output)
		}
		return errValidationFailed
	}
	_, createErr := validateAndCreateMutation(*outputPath, rooted, validation.ProfileConcludia)
	if createErr != nil {
		if errors.Is(createErr, os.ErrExist) {
			output.Diagnostics = append(output.Diagnostics, diagnostic.Diagnostic{
				Code: "output_exists", Message: fmt.Sprintf("refusing to overwrite existing output %s", *outputPath),
				Severity: diagnostic.SeverityError, Element: *outputPath,
			})
			if *jsonOutput {
				if jsonErr := writeIndentedJSON(stdout, output); jsonErr != nil {
					return jsonErr
				}
			} else {
				writeHumanExport(stdout, output)
			}
			return errValidationFailed
		}
		return createErr
	}
	output.Written = true
	if *jsonOutput {
		return writeIndentedJSON(stdout, output)
	}
	writeHumanExport(stdout, output)
	return nil
}

func documentStats(doc *argument.Document) statsOutput {
	return statsOutput{
		Statements: len(doc.Statements), Junctors: len(doc.Junctors),
		DirectSupports: len(doc.DirectSupports), Defeats: len(doc.Defeats),
	}
}

func writeHumanRooted(w io.Writer, output rootedOutput) {
	fmt.Fprintf(w, "root %s\n", output.Root)
	fmt.Fprintf(w, "exportable: %t\n", output.Exportable)
	fmt.Fprintf(w, "statements: %d\njunctors: %d\ndirect_supports: %d\ndefeats: %d\n",
		output.Stats.Statements, output.Stats.Junctors, output.Stats.DirectSupports, output.Stats.Defeats)
	for _, statement := range output.Document.Statements {
		value, _ := output.Evaluation.Statement(statement.ID)
		formatted := formatTruthStatus(evaluatedStatement{Statement: statement, EffectiveTruth: value.EffectiveTruth, TruthSource: value.TruthSource, Acceptance: value.Acceptance})
		fmt.Fprintf(w, "%s[%s] %s\t%s\t%s\n", statement.Role, statement.Kind, statement.ID, formatted, statement.Text)
	}
	for _, junctor := range output.Document.Junctors {
		fmt.Fprintf(w, "%s#%s(%s) -> %s\n", junctor.Connector, junctor.ID, strings.Join(junctor.Sources, ", "), junctor.Target)
	}
	for _, defeat := range output.Document.Defeats {
		fmt.Fprintln(w, formatDefeat(defeat))
	}
	for _, item := range output.Diagnostics {
		writeDiagnostic(w, item)
	}
}

func writeHumanExport(w io.Writer, output exportOutput) {
	if output.Written {
		fmt.Fprintf(w, "Exported root %s to %s\n", output.Root, output.Output)
	} else {
		fmt.Fprintf(w, "Did not export root %s to %s\n", output.Root, output.Output)
	}
	fmt.Fprintf(w, "statements: %d\njunctors: %d\ndirect_supports: %d\ndefeats: %d\n",
		output.Stats.Statements, output.Stats.Junctors, output.Stats.DirectSupports, output.Stats.Defeats)
	for _, item := range output.Diagnostics {
		writeDiagnostic(w, item)
	}
}

func writeRootUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cludia root [--json] FILE STATEMENT")
	fmt.Fprintln(w, "Read the complete upstream support closure and attached recursive defeat chains.")
}

func writeExportUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cludia export [--json] FILE --root STATEMENT --output FILE [--id ID] [--title TITLE]")
	fmt.Fprintln(w, "Write a new strict-Concludia-valid rooted artifact atomically without overwriting an existing file.")
}
