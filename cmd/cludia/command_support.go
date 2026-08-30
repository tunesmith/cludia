package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/tunesmith/cludia/internal/argument"
	"github.com/tunesmith/cludia/internal/diagnostic"
	"github.com/tunesmith/cludia/internal/evaluation"
	"github.com/tunesmith/cludia/internal/validation"
	"github.com/tunesmith/cludia/internal/workspace"
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

type evaluationMetadata struct {
	SchemaVersion int             `json:"schema_version"`
	Mode          evaluation.Mode `json:"mode"`
}

type evaluatedStatement struct {
	argument.Statement
	EffectiveTruth argument.Truth         `json:"effective_truth"`
	TruthSource    evaluation.TruthSource `json:"truth_source"`
	Acceptance     evaluation.Acceptance  `json:"acceptance,omitempty"`
}

type evaluatedJunctor struct {
	argument.Junctor
	EffectiveTruth argument.Truth `json:"effective_truth"`
}

func evaluationMeta(result evaluation.Result) evaluationMetadata {
	return evaluationMetadata{SchemaVersion: result.SchemaVersion, Mode: result.Mode}
}

func evaluatedStatementFor(statement argument.Statement, result evaluation.Result) evaluatedStatement {
	value, _ := result.Statement(statement.ID)
	return evaluatedStatement{
		Statement: statement, EffectiveTruth: value.EffectiveTruth,
		TruthSource: value.TruthSource, Acceptance: value.Acceptance,
	}
}

func evaluatedJunctorFor(junctor argument.Junctor, result evaluation.Result) evaluatedJunctor {
	value, _ := result.Junctor(junctor.ID)
	copy := junctor
	copy.Sources = append([]string(nil), junctor.Sources...)
	return evaluatedJunctor{Junctor: copy, EffectiveTruth: value.EffectiveTruth}
}

func formatTruthStatus(statement evaluatedStatement) string {
	if statement.TruthSource == evaluation.TruthDerived {
		return fmt.Sprintf("%s · derived", statement.EffectiveTruth)
	}
	if statement.TruthSource == evaluation.TruthUnassigned {
		return fmt.Sprintf("%s · unassigned", statement.EffectiveTruth)
	}
	if statement.Truth != statement.EffectiveTruth {
		return fmt.Sprintf("%s → %s", statement.Truth, statement.EffectiveTruth)
	}
	return string(statement.EffectiveTruth)
}

func evaluateDocument(doc *argument.Document) (evaluation.Result, []diagnostic.Diagnostic) {
	result, err := evaluation.Evaluate(doc)
	if err == nil {
		return result, []diagnostic.Diagnostic{}
	}
	if evaluationErr, ok := err.(*evaluation.Error); ok {
		return evaluation.Result{}, diagnosticError(evaluationErr.Code, evaluationErr.Message, evaluationErr.Element)
	}
	return evaluation.Result{}, diagnosticError("evaluation_failed", err.Error(), doc.ID)
}

type mutationOutput struct {
	SchemaVersion     int                     `json:"schema_version"`
	Action            string                  `json:"action"`
	DryRun            bool                    `json:"dry_run"`
	Profile           validation.Profile      `json:"profile"`
	Document          documentOutput          `json:"document"`
	Statement         argument.Statement      `json:"statement"`
	PreviousStatement *argument.Statement     `json:"previous_statement,omitempty"`
	SameProposition   *bool                   `json:"same_proposition,omitempty"`
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
	return workspace.LoadValidated(path, "")
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

func slugIDCollisionDiagnostic(doc *argument.Document, slug, ownerID string) []diagnostic.Diagnostic {
	elementType, id, collides := argument.SlugIDCollision(doc, slug, ownerID)
	if !collides {
		return nil
	}
	return diagnosticError(
		"statement_slug_id_collision",
		fmt.Sprintf("slug %q would be shadowed by %s id %s; choose a different slug", slug, elementType, id),
		ownerID,
	)
}

func writeIDAllocationFailure(stdout io.Writer, jsonOutput bool, profile validation.Profile, err error) error {
	if allocationErr, ok := err.(*argument.IDAllocationError); ok {
		return writeMutationFailure(stdout, jsonOutput, profile, allocationErr.Code, allocationErr.Message, allocationErr.Element)
	}
	return err
}

func writeArgumentMutationFailure(stdout io.Writer, jsonOutput bool, profile validation.Profile, err error) error {
	if _, ok := err.(*argument.IDAllocationError); ok {
		return writeIDAllocationFailure(stdout, jsonOutput, profile, err)
	}
	if mutationErr, ok := err.(*argument.MutationError); ok {
		return writeMutationFailure(stdout, jsonOutput, profile, mutationErr.Code, mutationErr.Message, mutationErr.Element)
	}
	if mutationErrs, ok := err.(*argument.MutationErrors); ok {
		diagnostics := make([]diagnostic.Diagnostic, 0, len(mutationErrs.Failures))
		for _, failure := range mutationErrs.Failures {
			diagnostics = append(diagnostics, diagnostic.Diagnostic{
				Code: failure.Code, Message: failure.Message, Severity: diagnostic.SeverityError, Element: failure.Element,
			})
		}
		if writeErr := writeFailure(stdout, jsonOutput, profile, diagnostics); writeErr != nil {
			return writeErr
		}
		return errValidationFailed
	}
	return err
}

func nextIDsMetadataChange(before, after *argument.Document) *changeOutput {
	previous, existed := before.MetadataValue(argument.NextIDsMetadataKey)
	current, _ := after.MetadataValue(argument.NextIDsMetadataKey)
	if existed && previous == current {
		return nil
	}
	operation := "added"
	if existed {
		operation = "updated"
	}
	return &changeOutput{Operation: operation, ElementType: "metadata", ID: argument.NextIDsMetadataKey}
}

func persistIDAllocator(doc *argument.Document, allocator *argument.IDAllocator) *changeOutput {
	previous, existed := doc.MetadataValue(argument.NextIDsMetadataKey)
	allocator.Persist(doc)
	current, _ := doc.MetadataValue(argument.NextIDsMetadataKey)
	if existed && previous == current {
		return nil
	}
	operation := "added"
	if existed {
		operation = "updated"
	}
	return &changeOutput{Operation: operation, ElementType: "metadata", ID: argument.NextIDsMetadataKey}
}

func ensureNextIDs(doc *argument.Document) (*changeOutput, error) {
	allocator, err := argument.NewIDAllocator(doc)
	if err != nil {
		return nil, err
	}
	return persistIDAllocator(doc, allocator), nil
}

func appendMetadataChange(changes []changeOutput, change *changeOutput) []changeOutput {
	if change != nil {
		return append(changes, *change)
	}
	return changes
}

func validateAndPersistMutation(path string, next *argument.Document, profile validation.Profile, persist bool) (validation.Result, error) {
	return workspace.ValidateAndPersist(path, next, profile, persist)
}

func validateAndCreateMutation(path string, next *argument.Document, profile validation.Profile) (validation.Result, error) {
	return workspace.ValidateAndCreate(path, next, profile)
}
