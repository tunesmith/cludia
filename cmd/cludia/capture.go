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
	"github.com/tunesmith/cludia/internal/validation"
)

func runInit(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOutput := fs.Bool("json", false, "output versioned JSON")
	documentID := fs.String("id", "", "stable document id (defaults from title)")
	title := fs.String("title", "", "workspace title")
	text := fs.String("text", "", "first statement text")
	statementID := fs.String("statement-id", "", "first statement id (must be exact next P id; default P1)")
	slug := fs.String("slug", "", "first statement slug (defaults from text)")
	truthName := fs.String("truth", "T", "first statement truth: T, F, or U")
	kindName := fs.String("kind", "fact", "first statement kind: fact or value")
	fs.Usage = func() { writeInitUsage(fs.Output()) }
	valueFlags := map[string]bool{"id": true, "title": true, "text": true, "statement-id": true, "slug": true, "truth": true, "kind": true}
	if err := fs.Parse(flagsFirst(args, valueFlags)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("init expects exactly one file")
	}
	cleanTitle, cleanText := strings.TrimSpace(*title), strings.TrimSpace(*text)
	if cleanTitle == "" || cleanText == "" {
		fs.Usage()
		return fmt.Errorf("--title and --text are required")
	}

	id := strings.TrimSpace(*documentID)
	if id == "" {
		id = argument.Slugify(cleanTitle)
		if !argument.ValidID(id) {
			id = "workspace-" + id
		}
		if !argument.ValidID(id) {
			id = "workspace"
		}
	}
	truth, _ := parseTruth(*truthName)
	kind, _ := parseKind(*kindName)
	doc, first, err := argument.InitializeDocument(argument.InitializeOptions{
		DocumentID: id, Title: cleanTitle,
		Statement: argument.StatementInput{
			Text: cleanText, RequestedID: strings.TrimSpace(*statementID), Slug: strings.TrimSpace(*slug), Truth: truth, Kind: kind,
		},
	})
	if err != nil {
		return writeArgumentMutationFailure(stdout, *jsonOutput, validation.ProfileWorkspace, err)
	}
	var diagnostics []diagnostic.Diagnostic
	validated, createErr := validateAndCreateMutation(fs.Arg(0), doc, validation.ProfileWorkspace)
	diagnostics = append(diagnostics, validated.Diagnostics...)
	if diagnostic.HasErrors(diagnostics) {
		if err := writeFailure(stdout, *jsonOutput, validation.ProfileWorkspace, diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	if createErr != nil {
		if errors.Is(createErr, os.ErrExist) {
			return fmt.Errorf("refusing to overwrite existing workspace %s", fs.Arg(0))
		}
		return createErr
	}
	output := mutationOutput{
		SchemaVersion: outputSchemaVersion, Action: "init", DryRun: false,
		Profile: validation.ProfileWorkspace, Document: documentSummary(doc), Statement: first,
		Changes: appendMetadataChange([]changeOutput{
			{Operation: "created", ElementType: "document", ID: doc.ID},
			{Operation: "added", ElementType: "statement", ID: first.ID},
		}, nextIDsMetadataChange(&argument.Document{}, doc)),
		Diagnostics: []diagnostic.Diagnostic{},
	}
	return writeMutation(stdout, *jsonOutput, output)
}

func runAdd(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOutput := fs.Bool("json", false, "output versioned JSON")
	text := fs.String("text", "", "statement text")
	idName := fs.String("id", "", "exact next P statement id (generated if omitted)")
	slugName := fs.String("slug", "", "statement slug (generated if omitted)")
	truthName := fs.String("truth", "T", "statement truth: T, F, or U")
	kindName := fs.String("kind", "fact", "statement kind: fact or value")
	fs.Usage = func() { writeAddUsage(fs.Output()) }
	valueFlags := map[string]bool{"text": true, "id": true, "slug": true, "truth": true, "kind": true}
	if err := fs.Parse(flagsFirst(args, valueFlags)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 1 || strings.TrimSpace(*text) == "" {
		fs.Usage()
		return fmt.Errorf("add expects one file and requires --text")
	}
	doc, profile, diagnostics := loadValidated(fs.Arg(0))
	if diagnostic.HasErrors(diagnostics) {
		if err := writeFailure(stdout, *jsonOutput, profile, diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}

	truth, _ := parseTruth(*truthName)
	kind, _ := parseKind(*kindName)
	next, statement, err := argument.AddStatement(doc, argument.StatementInput{
		Text: *text, RequestedID: strings.TrimSpace(*idName), Slug: strings.TrimSpace(*slugName), Truth: truth, Kind: kind,
	})
	if err != nil {
		return writeArgumentMutationFailure(stdout, *jsonOutput, profile, err)
	}
	validated, err := validateAndPersistMutation(fs.Arg(0), next, profile, true)
	if err != nil {
		return err
	}
	diagnostics = append([]diagnostic.Diagnostic(nil), validated.Diagnostics...)
	if diagnostic.HasErrors(diagnostics) {
		if err := writeFailure(stdout, *jsonOutput, profile, diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	if diagnostics == nil {
		diagnostics = []diagnostic.Diagnostic{}
	}
	output := mutationOutput{
		SchemaVersion: outputSchemaVersion, Action: "add", DryRun: false,
		Profile: profile, Document: documentSummary(next), Statement: statement,
		Changes: appendMetadataChange([]changeOutput{{Operation: "added", ElementType: "statement", ID: statement.ID}}, nextIDsMetadataChange(doc, next)), Diagnostics: diagnostics,
	}
	return writeMutation(stdout, *jsonOutput, output)
}

func writeMutation(w io.Writer, jsonOutput bool, output mutationOutput) error {
	if jsonOutput {
		return writeIndentedJSON(w, output)
	}
	if output.Action == "init" {
		fmt.Fprintf(w, "Created workspace %s — %s\n", output.Document.ID, output.Document.Title)
	}
	verb := "Added"
	if output.Action == "edit" {
		verb = "Updated"
	}
	fmt.Fprintf(w, "%s %s:%s\n", verb, output.Statement.ID, output.Statement.Slug)
	if output.PreviousStatement != nil && output.PreviousStatement.Text != output.Statement.Text {
		fmt.Fprintf(w, "Previous: %s\n", output.PreviousStatement.Text)
		fmt.Fprint(w, "Current: ")
	}
	fmt.Fprintf(w, "%s\n", output.Statement.Text)
	for _, item := range output.Diagnostics {
		writeDiagnostic(w, item)
	}
	return nil
}

func parseTruth(value string) (argument.Truth, bool) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "T", "TRUE":
		return argument.TruthTrue, true
	case "F", "FALSE":
		return argument.TruthFalse, true
	case "U", "UNKNOWN":
		return argument.TruthUnknown, true
	default:
		return argument.Truth(value), false
	}
}

func parseKind(value string) (argument.Kind, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "fact":
		return argument.KindFact, true
	case "value":
		return argument.KindValue, true
	default:
		return argument.Kind(value), false
	}
}

func writeInitUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cludia init [--json] FILE --title TITLE --text TEXT")
	fmt.Fprintln(w, "Create a workspace atomically with its first factual premise.")
}

func writeAddUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cludia add [--json] FILE --text TEXT [--truth T|F|U] [--kind fact|value]")
	fmt.Fprintln(w, "Capture an isolated premise in an existing workspace.")
}
