package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/tunesmith/cludia/internal/argfile"
	"github.com/tunesmith/cludia/internal/argument"
	"github.com/tunesmith/cludia/internal/diagnostic"
	"github.com/tunesmith/cludia/internal/validation"
)

type optionalStringFlag struct {
	value string
	set   bool
}

func (f *optionalStringFlag) String() string {
	return f.value
}

func (f *optionalStringFlag) Set(value string) error {
	f.value, f.set = value, true
	return nil
}

type changeOutput struct {
	Operation   string `json:"operation"`
	ElementType string `json:"element_type"`
	ID          string `json:"id"`
}

type mutationOutput struct {
	SchemaVersion     int                     `json:"schema_version"`
	Action            string                  `json:"action"`
	DryRun            bool                    `json:"dry_run"`
	Profile           validation.Profile      `json:"profile"`
	Document          documentOutput          `json:"document"`
	Statement         argument.Statement      `json:"statement"`
	PreviousStatement *argument.Statement     `json:"previous_statement,omitempty"`
	Changes           []changeOutput          `json:"changes"`
	Diagnostics       []diagnostic.Diagnostic `json:"diagnostics"`
}

type failureOutput struct {
	SchemaVersion int                     `json:"schema_version"`
	OK            bool                    `json:"ok"`
	Profile       validation.Profile      `json:"profile"`
	Diagnostics   []diagnostic.Diagnostic `json:"diagnostics"`
}

func loadValidated(path string) (*argument.Document, validation.Profile, []diagnostic.Diagnostic) {
	parsed := argfile.Load(path)
	profile := selectedProfile(parsed.Document, "")
	diagnostics := append([]diagnostic.Diagnostic(nil), parsed.Diagnostics...)
	if !diagnostic.HasErrors(diagnostics) {
		validated := validation.Validate(parsed.Document, profile)
		diagnostics = append(diagnostics, validated.Diagnostics...)
	}
	if diagnostics == nil {
		diagnostics = []diagnostic.Diagnostic{}
	}
	return parsed.Document, profile, diagnostics
}

func writeFailure(stdout io.Writer, jsonOutput bool, profile validation.Profile, diagnostics []diagnostic.Diagnostic) error {
	if diagnostics == nil {
		diagnostics = []diagnostic.Diagnostic{}
	}
	if jsonOutput {
		return writeIndentedJSON(stdout, failureOutput{SchemaVersion: outputSchemaVersion, OK: false, Profile: profile, Diagnostics: diagnostics})
	}
	for _, item := range diagnostics {
		writeDiagnostic(stdout, item)
	}
	return nil
}

func writeIndentedJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func writeDiagnostic(w io.Writer, item diagnostic.Diagnostic) {
	location := ""
	if item.Line > 0 {
		location = fmt.Sprintf(" line %d", item.Line)
	}
	if item.Element != "" {
		location += " " + item.Element
	}
	fmt.Fprintf(w, "%s [%s]%s: %s\n", strings.ToUpper(string(item.Severity)), item.Code, location, item.Message)
}

func documentSummary(doc *argument.Document) documentOutput {
	return documentOutput{ID: doc.ID, Title: doc.Title}
}

func diagnosticError(code, message, element string) []diagnostic.Diagnostic {
	return []diagnostic.Diagnostic{{Code: code, Message: message, Severity: diagnostic.SeverityError, Element: element}}
}
