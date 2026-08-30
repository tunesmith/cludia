package main

import (
	"encoding/json"
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

type batchAddInput struct {
	SchemaVersion int                      `json:"schema_version"`
	Statements    []batchAddStatementInput `json:"statements"`
}

type batchAddStatementInput struct {
	Key   string `json:"key"`
	Text  string `json:"text"`
	ID    string `json:"id,omitempty"`
	Slug  string `json:"slug,omitempty"`
	Truth string `json:"truth,omitempty"`
	Kind  string `json:"kind,omitempty"`
}

type batchAddStatementOutput struct {
	Key       string             `json:"key"`
	Statement argument.Statement `json:"statement"`
}

type batchAddOutput struct {
	SchemaVersion int                       `json:"schema_version"`
	Action        string                    `json:"action"`
	DryRun        bool                      `json:"dry_run"`
	Profile       validation.Profile        `json:"profile"`
	Document      documentOutput            `json:"document"`
	Statements    []batchAddStatementOutput `json:"statements"`
	Changes       []changeOutput            `json:"changes"`
	Diagnostics   []diagnostic.Diagnostic   `json:"diagnostics"`
}

const batchAddExample = `{
  "schema_version": 1,
  "statements": [
    {
      "key": "observation-1",
      "text": "The window was open."
    }
  ]
}`

func runAddBatch(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("add-batch", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOutput := fs.Bool("json", false, "output versioned JSON")
	dryRun := fs.Bool("dry-run", false, "validate and report without saving")
	example := fs.Bool("example", false, "print a minimal input JSON example and exit")
	inputPath := fs.String("input", "", "versioned JSON batch input file")
	fs.Usage = func() { writeAddBatchUsage(fs.Output()) }
	if err := fs.Parse(flagsFirst(args, map[string]bool{"input": true})); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if *example {
		if fs.NArg() != 0 || strings.TrimSpace(*inputPath) != "" || *dryRun || *jsonOutput {
			fs.Usage()
			return fmt.Errorf("add-batch --example does not accept a workspace file or other options")
		}
		fmt.Fprintln(stdout, batchAddExample)
		return nil
	}
	if fs.NArg() != 1 || strings.TrimSpace(*inputPath) == "" {
		fs.Usage()
		return fmt.Errorf("add-batch expects one workspace file and requires --input")
	}
	doc, profile, diagnostics := loadValidated(fs.Arg(0))
	if diagnostic.HasErrors(diagnostics) {
		if err := writeFailure(stdout, *jsonOutput, profile, diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	input, err := readBatchAddInput(strings.TrimSpace(*inputPath))
	if err != nil {
		return writeMutationFailure(stdout, *jsonOutput, profile, "batch_input_invalid", err.Error(), strings.TrimSpace(*inputPath))
	}
	if input.SchemaVersion != 1 {
		return writeMutationFailure(stdout, *jsonOutput, profile, "batch_schema_version_unsupported", fmt.Sprintf("unsupported batch schema_version %d; expected 1", input.SchemaVersion), strings.TrimSpace(*inputPath))
	}
	if len(input.Statements) == 0 {
		return writeMutationFailure(stdout, *jsonOutput, profile, "batch_statements_required", "batch input must contain at least one statement", strings.TrimSpace(*inputPath))
	}

	domainInputs := make([]argument.StatementInput, 0, len(input.Statements))
	orderedKeys := make([]string, 0, len(input.Statements))
	keys := make(map[string]bool, len(input.Statements))
	for index, item := range input.Statements {
		key := strings.TrimSpace(item.Key)
		if key == "" {
			return writeMutationFailure(stdout, *jsonOutput, profile, "batch_key_required", fmt.Sprintf("statement at index %d requires a non-empty key", index), fmt.Sprintf("statements[%d]", index))
		}
		if keys[key] {
			return writeMutationFailure(stdout, *jsonOutput, profile, "batch_key_duplicate", fmt.Sprintf("batch key %q appears more than once", key), key)
		}
		keys[key] = true
		text := strings.TrimSpace(item.Text)
		if text == "" {
			return writeMutationFailure(stdout, *jsonOutput, profile, "batch_text_required", fmt.Sprintf("batch statement %q requires non-empty text", key), key)
		}
		slug := strings.TrimSpace(item.Slug)
		if slug != "" && !argument.ValidSlug(slug) {
			return writeMutationFailure(stdout, *jsonOutput, profile, "statement_slug_invalid", fmt.Sprintf("batch statement %q has invalid slug %q", key, slug), key)
		}
		truthName := strings.TrimSpace(item.Truth)
		if truthName == "" {
			truthName = "T"
		}
		kindName := strings.TrimSpace(item.Kind)
		if kindName == "" {
			kindName = "fact"
		}
		truth, truthOK := parseTruth(truthName)
		kind, kindOK := parseKind(kindName)
		if !truthOK {
			return writeMutationFailure(stdout, *jsonOutput, profile, "truth_invalid", fmt.Sprintf("invalid truth %q for batch statement %q; expected T, F, or U", item.Truth, key), key)
		}
		if !kindOK {
			return writeMutationFailure(stdout, *jsonOutput, profile, "kind_invalid", fmt.Sprintf("invalid kind %q for batch statement %q; expected fact or value", item.Kind, key), key)
		}
		orderedKeys = append(orderedKeys, key)
		domainInputs = append(domainInputs, argument.StatementInput{
			Text: text, RequestedID: strings.TrimSpace(item.ID), Slug: slug, Truth: truth, Kind: kind,
		})
	}
	next, statements, err := argument.AddStatements(doc, domainInputs)
	if err != nil {
		if batchErr, ok := err.(*argument.BatchStatementError); ok {
			key := orderedKeys[batchErr.Index]
			if mutationErr, ok := batchErr.Err.(*argument.MutationError); ok {
				return writeMutationFailure(stdout, *jsonOutput, profile, mutationErr.Code, mutationErr.Message, key)
			}
			return writeArgumentMutationFailure(stdout, *jsonOutput, profile, batchErr.Err)
		}
		return writeArgumentMutationFailure(stdout, *jsonOutput, profile, err)
	}
	outputs := make([]batchAddStatementOutput, 0, len(statements))
	changes := make([]changeOutput, 0, len(statements)+1)
	for index, statement := range statements {
		outputs = append(outputs, batchAddStatementOutput{Key: orderedKeys[index], Statement: statement})
		changes = append(changes, changeOutput{Operation: "added", ElementType: "statement", ID: statement.ID})
	}
	changes = appendMetadataChange(changes, nextIDsMetadataChange(doc, next))
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
	output := batchAddOutput{
		SchemaVersion: outputSchemaVersion, Action: "add-batch", DryRun: *dryRun,
		Profile: profile, Document: documentSummary(next), Statements: outputs,
		Changes: changes, Diagnostics: diagnostics,
	}
	return writeBatchAdd(stdout, *jsonOutput, output)
}

func readBatchAddInput(path string) (batchAddInput, error) {
	file, err := os.Open(path)
	if err != nil {
		return batchAddInput{}, fmt.Errorf("read batch input: %w", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var input batchAddInput
	if err := decoder.Decode(&input); err != nil {
		return batchAddInput{}, fmt.Errorf("decode batch input: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return batchAddInput{}, fmt.Errorf("decode batch input: unexpected trailing JSON value")
		}
		return batchAddInput{}, fmt.Errorf("decode batch input: %w", err)
	}
	return input, nil
}

func writeBatchAdd(w io.Writer, jsonOutput bool, output batchAddOutput) error {
	if jsonOutput {
		return writeIndentedJSON(w, output)
	}
	fmt.Fprintf(w, "Added %d statements\n", len(output.Statements))
	for _, item := range output.Statements {
		fmt.Fprintf(w, "%s -> %s:%s %s\n", item.Key, item.Statement.ID, item.Statement.Slug, item.Statement.Text)
	}
	if output.DryRun {
		fmt.Fprintln(w, "dry-run: no file changes written")
	}
	for _, item := range output.Diagnostics {
		writeDiagnostic(w, item)
	}
	return nil
}

func writeAddBatchUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cludia add-batch [--dry-run] [--json] FILE --input FILE")
	fmt.Fprintln(w, "       cludia add-batch --example")
	fmt.Fprintln(w, "Atomically add statements from a versioned JSON batch and return each caller key with its assigned statement.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Minimal input JSON:")
	fmt.Fprintln(w, batchAddExample)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Each statement requires unique key and non-empty text fields; id, slug, truth (T|F|U), and kind (fact|value) are optional.")
}
