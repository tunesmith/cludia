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

type statementOutput struct {
	evaluatedStatement
	Isolated bool `json:"isolated"`
}

type listOutput struct {
	SchemaVersion  int                      `json:"schema_version"`
	Profile        validation.Profile       `json:"profile"`
	Evaluation     evaluationMetadata       `json:"evaluation"`
	State          string                   `json:"state"`
	Statements     []statementOutput        `json:"statements"`
	Junctors       []evaluatedJunctor       `json:"junctors"`
	DirectSupports []argument.DirectSupport `json:"direct_supports"`
	Defeats        []argument.Defeat        `json:"defeats"`
	Diagnostics    []diagnostic.Diagnostic  `json:"diagnostics"`
}

type showOutput struct {
	SchemaVersion int                     `json:"schema_version"`
	Profile       validation.Profile      `json:"profile"`
	Evaluation    evaluationMetadata      `json:"evaluation"`
	ElementType   string                  `json:"element_type"`
	Statement     *statementOutput        `json:"statement,omitempty"`
	Junctor       *evaluatedJunctor       `json:"junctor,omitempty"`
	Relations     *query.Relations        `json:"relations,omitempty"`
	Diagnostics   []diagnostic.Diagnostic `json:"diagnostics"`
}

func runList(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOutput := fs.Bool("json", false, "output versioned JSON")
	state := fs.String("state", "all", "filter: all or isolated")
	fs.Usage = func() { writeListUsage(fs.Output()) }
	if err := fs.Parse(flagsFirst(args, map[string]bool{"state": true})); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("list expects exactly one file")
	}
	filter := strings.ToLower(strings.TrimSpace(*state))
	if filter != "all" && filter != "isolated" {
		return fmt.Errorf("unknown list state %q; expected all or isolated", *state)
	}
	doc, profile, diagnostics := loadValidated(fs.Arg(0))
	if diagnostic.HasErrors(diagnostics) {
		if err := writeFailure(stdout, *jsonOutput, profile, diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	evaluated, evaluationDiagnostics := evaluateDocument(doc)
	if diagnostic.HasErrors(evaluationDiagnostics) {
		if err := writeFailure(stdout, *jsonOutput, profile, evaluationDiagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}

	isolated := query.IsolatedStatementIDs(doc)
	output := listOutput{
		SchemaVersion: outputSchemaVersion, Profile: profile, Evaluation: evaluationMeta(evaluated), State: filter,
		Statements: []statementOutput{}, Junctors: []evaluatedJunctor{},
		DirectSupports: []argument.DirectSupport{}, Defeats: []argument.Defeat{}, Diagnostics: diagnostics,
	}
	for _, statement := range doc.Statements {
		if filter == "isolated" && !isolated[statement.ID] {
			continue
		}
		output.Statements = append(output.Statements, statementOutput{evaluatedStatement: evaluatedStatementFor(statement, evaluated), Isolated: isolated[statement.ID]})
	}
	if filter == "all" {
		for _, junctor := range doc.Junctors {
			output.Junctors = append(output.Junctors, evaluatedJunctorFor(junctor, evaluated))
		}
		output.DirectSupports = append(output.DirectSupports, doc.DirectSupports...)
		output.Defeats = append(output.Defeats, doc.Defeats...)
	}
	if *jsonOutput {
		return writeIndentedJSON(stdout, output)
	}
	writeHumanList(stdout, output)
	return nil
}

func runShow(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOutput := fs.Bool("json", false, "output versioned JSON")
	withRelations := fs.Bool("relations", false, "include incoming/outgoing support and defeats")
	fs.Usage = func() { writeShowUsage(fs.Output()) }
	if err := fs.Parse(flagsFirst(args, map[string]bool{})); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 2 {
		fs.Usage()
		return fmt.Errorf("show expects a file and statement id, slug, or junctor id")
	}
	doc, profile, diagnostics := loadValidated(fs.Arg(0))
	if diagnostic.HasErrors(diagnostics) {
		if err := writeFailure(stdout, *jsonOutput, profile, diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	evaluated, evaluationDiagnostics := evaluateDocument(doc)
	if diagnostic.HasErrors(evaluationDiagnostics) {
		if err := writeFailure(stdout, *jsonOutput, profile, evaluationDiagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	reference := fs.Arg(1)
	isolated := query.IsolatedStatementIDs(doc)
	output := showOutput{SchemaVersion: outputSchemaVersion, Profile: profile, Evaluation: evaluationMeta(evaluated), Diagnostics: diagnostics}
	resolved, found := doc.ResolveElement(reference)
	if found && resolved.Type == argument.ElementStatement {
		statement, _ := doc.Statement(resolved.ID)
		output.ElementType = "statement"
		view := statementOutput{evaluatedStatement: evaluatedStatementFor(*statement, evaluated), Isolated: isolated[statement.ID]}
		output.Statement = &view
		if *withRelations {
			relations := query.StatementRelations(doc, statement.ID)
			output.Relations = &relations
		}
	} else if found && resolved.Type == argument.ElementJunctor {
		junctor, _ := doc.Junctor(resolved.ID)
		output.ElementType = "junctor"
		copy := evaluatedJunctorFor(*junctor, evaluated)
		output.Junctor = &copy
		if *withRelations {
			relations := query.JunctorRelations(doc, junctor.ID)
			output.Relations = &relations
		}
	} else {
		diagnostics := diagnosticError("element_not_found", fmt.Sprintf("statement or junctor %q not found", reference), reference)
		if err := writeFailure(stdout, *jsonOutput, profile, diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	if *jsonOutput {
		return writeIndentedJSON(stdout, output)
	}
	writeHumanShow(stdout, output)
	return nil
}

func writeHumanList(w io.Writer, output listOutput) {
	for _, statement := range output.Statements {
		marker := ""
		if statement.Isolated {
			marker = "\tisolated"
		}
		fmt.Fprintf(w, "%s[%s]\t%s", statement.Role, statement.Kind, statement.ID)
		if statement.Slug != "" {
			fmt.Fprintf(w, ":%s", statement.Slug)
		}
		fmt.Fprintf(w, "\t%s\t%s%s\n", formatTruthStatus(statement.evaluatedStatement), statement.Text, marker)
	}
	for _, junctor := range output.Junctors {
		fmt.Fprintf(w, "%s#%s(%s) -> %s\t%s\n", junctor.Connector, junctor.ID, strings.Join(junctor.Sources, ", "), junctor.Target, junctor.EffectiveTruth)
	}
	for _, support := range output.DirectSupports {
		fmt.Fprintf(w, "%s(%s) -> %s\tdirect\n", support.Connector, support.Source, support.Target)
	}
	for _, defeat := range output.Defeats {
		fmt.Fprintln(w, formatDefeat(defeat))
	}
	for _, item := range output.Diagnostics {
		writeDiagnostic(w, item)
	}
}

func writeHumanShow(w io.Writer, output showOutput) {
	if output.Statement != nil {
		statement := output.Statement
		fmt.Fprintf(w, "%s[%s] %s", statement.Role, statement.Kind, statement.ID)
		if statement.Slug != "" {
			fmt.Fprintf(w, ":%s", statement.Slug)
		}
		fmt.Fprintf(w, " ::%s\n%s\n", formatTruthStatus(statement.evaluatedStatement), statement.Text)
		if statement.Acceptance != "" {
			fmt.Fprintf(w, "acceptance: %s\n", statement.Acceptance)
		}
		fmt.Fprintf(w, "isolated: %t\n", statement.Isolated)
	} else if output.Junctor != nil {
		junctor := output.Junctor
		fmt.Fprintf(w, "%s#%s(%s) -> %s\teffective %s\n", junctor.Connector, junctor.ID, strings.Join(junctor.Sources, ", "), junctor.Target, junctor.EffectiveTruth)
	}
	if output.Relations != nil {
		writeHumanRelations(w, *output.Relations)
	}
	for _, item := range output.Diagnostics {
		writeDiagnostic(w, item)
	}
}

func writeHumanRelations(w io.Writer, relations query.Relations) {
	for _, support := range relations.IncomingSupport {
		fmt.Fprintf(w, "incoming: %s\n", formatSupport(support))
	}
	for _, support := range relations.OutgoingSupport {
		fmt.Fprintf(w, "outgoing: %s\n", formatSupport(support))
	}
	for _, defeat := range relations.DefeatsTargeting {
		fmt.Fprintf(w, "defeated by: %s\n", formatDefeat(defeat))
	}
	for _, defeat := range relations.DefeatsOriginating {
		fmt.Fprintf(w, "defeats: %s\n", formatDefeat(defeat))
	}
}

func formatSupport(support query.Support) string {
	label := string(support.Connector)
	if support.ID != "" {
		label += "#" + support.ID
	}
	return fmt.Sprintf("%s(%s) -> %s", label, strings.Join(support.Sources, ", "), support.Target)
}

func formatDefeat(defeat argument.Defeat) string {
	switch defeat.Scope {
	case argument.DefeatInference:
		return fmt.Sprintf("%s -> inference %s:target %s", defeat.From, defeat.JunctorID, defeat.AtTarget)
	default:
		return fmt.Sprintf("%s -> %s %s", defeat.From, defeat.Scope, defeat.To)
	}
}

func writeListUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cludia list [--state all|isolated] [--json] FILE")
	fmt.Fprintln(w, "List statements and durable relations, optionally limited to isolated statements.")
}

func writeShowUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cludia show [--relations] [--json] FILE ELEMENT")
	fmt.Fprintln(w, "Show a statement or junctor by id or statement slug.")
}
