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

type componentSummaryOutput struct {
	Anchor         string `json:"anchor"`
	Statements     int    `json:"statements"`
	Junctors       int    `json:"junctors"`
	DirectSupports int    `json:"direct_supports"`
	Defeats        int    `json:"defeats"`
	Isolated       bool   `json:"isolated"`
}

type componentsOutput struct {
	SchemaVersion int                      `json:"schema_version"`
	Profile       validation.Profile       `json:"profile"`
	Components    []componentSummaryOutput `json:"components"`
	Diagnostics   []diagnostic.Diagnostic  `json:"diagnostics"`
}

type componentOutput struct {
	SchemaVersion  int                      `json:"schema_version"`
	Profile        validation.Profile       `json:"profile"`
	Anchor         string                   `json:"anchor"`
	Isolated       bool                     `json:"isolated"`
	Statements     []argument.Statement     `json:"statements"`
	Junctors       []argument.Junctor       `json:"junctors"`
	DirectSupports []argument.DirectSupport `json:"direct_supports"`
	Defeats        []argument.Defeat        `json:"defeats"`
	Diagnostics    []diagnostic.Diagnostic  `json:"diagnostics"`
}

func runComponents(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("components", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOutput := fs.Bool("json", false, "output versioned JSON")
	fs.Usage = func() { writeComponentsUsage(fs.Output()) }
	if err := fs.Parse(flagsFirst(args, map[string]bool{})); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("components expects exactly one file")
	}
	doc, profile, diagnostics := loadValidated(fs.Arg(0))
	if diagnostic.HasErrors(diagnostics) {
		if err := writeFailure(stdout, *jsonOutput, profile, diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	output := componentsOutput{
		SchemaVersion: outputSchemaVersion, Profile: profile,
		Components: []componentSummaryOutput{}, Diagnostics: diagnostics,
	}
	for _, component := range query.Components(doc) {
		output.Components = append(output.Components, componentSummaryOutput{
			Anchor: component.Anchor, Statements: len(component.Statements),
			Junctors: len(component.Junctors), DirectSupports: len(component.DirectSupports),
			Defeats: len(component.Defeats), Isolated: component.Isolated,
		})
	}
	if *jsonOutput {
		return writeIndentedJSON(stdout, output)
	}
	for _, component := range output.Components {
		label := "component"
		if component.Isolated {
			label = "isolated"
		}
		fmt.Fprintf(stdout, "%s %s\t%d %s\t%d %s\t%d %s\t%d %s\n",
			label, component.Anchor,
			component.Statements, plural(component.Statements, "statement", "statements"),
			component.Junctors, plural(component.Junctors, "junctor", "junctors"),
			component.DirectSupports, plural(component.DirectSupports, "direct support", "direct supports"),
			component.Defeats, plural(component.Defeats, "defeat", "defeats"))
	}
	for _, item := range output.Diagnostics {
		writeDiagnostic(stdout, item)
	}
	return nil
}

func runComponent(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("component", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOutput := fs.Bool("json", false, "output versioned JSON")
	fs.Usage = func() { writeComponentUsage(fs.Output()) }
	if err := fs.Parse(flagsFirst(args, map[string]bool{})); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 2 {
		fs.Usage()
		return fmt.Errorf("component expects a file and statement id, slug, or junctor id")
	}
	doc, profile, diagnostics := loadValidated(fs.Arg(0))
	if diagnostic.HasErrors(diagnostics) {
		if err := writeFailure(stdout, *jsonOutput, profile, diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	component, ok := query.ComponentContaining(doc, fs.Arg(1))
	if !ok {
		diagnostics := diagnosticError("element_not_found", fmt.Sprintf("statement or junctor %q not found", fs.Arg(1)), fs.Arg(1))
		if err := writeFailure(stdout, *jsonOutput, profile, diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	output := componentOutput{
		SchemaVersion: outputSchemaVersion, Profile: profile,
		Anchor: component.Anchor, Isolated: component.Isolated,
		Statements: component.Statements, Junctors: component.Junctors,
		DirectSupports: component.DirectSupports, Defeats: component.Defeats,
		Diagnostics: diagnostics,
	}
	if *jsonOutput {
		return writeIndentedJSON(stdout, output)
	}
	writeHumanComponent(stdout, output)
	return nil
}

func writeHumanComponent(w io.Writer, output componentOutput) {
	label := "component"
	if output.Isolated {
		label = "isolated"
	}
	fmt.Fprintf(w, "%s %s\n", label, output.Anchor)
	for _, statement := range output.Statements {
		fmt.Fprintf(w, "statement %s", statement.ID)
		if statement.Slug != "" {
			fmt.Fprintf(w, ":%s", statement.Slug)
		}
		fmt.Fprintf(w, "\t%s\n", statement.Text)
	}
	for _, junctor := range output.Junctors {
		fmt.Fprintf(w, "junctor %s#%s(%s) -> %s\n", junctor.Connector, junctor.ID, strings.Join(junctor.Sources, ", "), junctor.Target)
	}
	for _, support := range output.DirectSupports {
		fmt.Fprintf(w, "direct %s(%s) -> %s\n", support.Connector, support.Source, support.Target)
	}
	for _, defeat := range output.Defeats {
		fmt.Fprintf(w, "defeat %s\n", formatDefeat(defeat))
	}
	for _, item := range output.Diagnostics {
		writeDiagnostic(w, item)
	}
}

func writeComponentsUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cludia components [--json] FILE")
	fmt.Fprintln(w, "List computed reasoning components in deterministic document order.")
}

func plural(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

func writeComponentUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cludia component [--json] FILE ELEMENT")
	fmt.Fprintln(w, "Show the complete component containing a statement, slug, or junctor.")
}
