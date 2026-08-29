package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/tunesmith/cludia/internal/argfile"
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

	next := doc.Clone()
	allocator, err := argument.NewIDAllocator(next)
	if err != nil {
		return err
	}
	outputs := make([]batchAddStatementOutput, 0, len(input.Statements))
	changes := make([]changeOutput, 0, len(input.Statements))
	keys := make(map[string]bool, len(input.Statements))
	usedIDs := make(map[string]string, len(next.Statements)+len(next.Junctors)+len(input.Statements))
	usedSlugs := make(map[string]string, len(next.Statements)+len(input.Statements))
	for _, statement := range next.Statements {
		usedIDs[statement.ID] = statement.ID
		if statement.Slug != "" {
			usedSlugs[statement.Slug] = statement.ID
		}
	}
	for _, junctor := range next.Junctors {
		usedIDs[junctor.ID] = junctor.ID
	}
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
		id, allocationErr := allocator.Statement(argument.RolePremise, strings.TrimSpace(item.ID))
		if allocationErr != nil {
			return writeIDAllocationFailure(stdout, *jsonOutput, profile, allocationErr)
		}
		if !argument.ValidID(id) {
			return writeMutationFailure(stdout, *jsonOutput, profile, "statement_id_invalid", fmt.Sprintf("batch statement %q has invalid statement id %q", key, id), key)
		}
		if owner, exists := usedIDs[id]; exists {
			return writeMutationFailure(stdout, *jsonOutput, profile, "id_duplicate", fmt.Sprintf("batch statement %q uses id %q already owned by %s", key, id, owner), key)
		}
		slug := strings.TrimSpace(item.Slug)
		if slug == "" {
			slug = argument.UniqueSlug(next, text)
		}
		if !argument.ValidSlug(slug) {
			return writeMutationFailure(stdout, *jsonOutput, profile, "statement_slug_invalid", fmt.Sprintf("batch statement %q has invalid slug %q", key, slug), key)
		}
		if owner, exists := usedSlugs[slug]; exists {
			return writeMutationFailure(stdout, *jsonOutput, profile, "statement_slug_duplicate", fmt.Sprintf("batch statement %q uses slug %q already owned by %s", key, slug, owner), key)
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
			return writeMutationFailure(stdout, *jsonOutput, profile, "truth_invalid", fmt.Sprintf("invalid truth %q for batch statement %q; expected T, F, or U", item.Truth, key), id)
		}
		if !kindOK {
			return writeMutationFailure(stdout, *jsonOutput, profile, "kind_invalid", fmt.Sprintf("invalid kind %q for batch statement %q; expected fact or value", item.Kind, key), id)
		}
		statement := argument.Statement{ID: id, Slug: slug, Role: argument.RolePremise, Kind: kind, Truth: truth, Text: text}
		next.Statements = append(next.Statements, statement)
		usedIDs[id] = key
		usedSlugs[slug] = key
		outputs = append(outputs, batchAddStatementOutput{Key: key, Statement: statement})
		changes = append(changes, changeOutput{Operation: "added", ElementType: "statement", ID: id})
	}
	changes = appendMetadataChange(changes, persistIDAllocator(next, allocator))
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
