package main

import (
	"bytes"
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

type batchAuthorInput struct {
	SchemaVersion int                         `json:"schema_version"`
	Statements    []batchAuthorStatementInput `json:"statements"`
	Derivations   []batchDerivationInput      `json:"derivations"`
	Defeats       []batchDefeatInput          `json:"defeats"`
}

type batchAuthorStatementInput struct {
	Key   string `json:"key"`
	Text  string `json:"text"`
	ID    string `json:"id,omitempty"`
	Slug  string `json:"slug,omitempty"`
	Truth string `json:"truth,omitempty"`
	Kind  string `json:"kind,omitempty"`
	Role  string `json:"role,omitempty"`
}

type batchReferenceInput struct {
	Key string `json:"key,omitempty"`
	ID  string `json:"id,omitempty"`
}

type batchDerivationInput struct {
	Key     string                `json:"key"`
	Sources []batchReferenceInput `json:"sources"`
	Target  batchReferenceInput   `json:"target"`
}

type batchDefeatInput struct {
	From   batchReferenceInput `json:"from"`
	Scope  string              `json:"scope"`
	Target batchReferenceInput `json:"target"`
}

type batchAddStatementOutput struct {
	Key       string             `json:"key"`
	Statement argument.Statement `json:"statement"`
}

type batchDerivationOutput struct {
	Key     string             `json:"key"`
	Target  argument.Statement `json:"target"`
	Junctor argument.Junctor   `json:"junctor"`
}

type batchDefeatOutput struct {
	From   batchReferenceInput `json:"from"`
	Defeat argument.Defeat     `json:"defeat"`
}

type batchAuthorOutput struct {
	SchemaVersion int                       `json:"schema_version"`
	Action        string                    `json:"action"`
	DryRun        bool                      `json:"dry_run"`
	Profile       validation.Profile        `json:"profile"`
	Document      documentOutput            `json:"document"`
	Statements    []batchAddStatementOutput `json:"statements"`
	Derivations   []batchDerivationOutput   `json:"derivations"`
	Defeats       []batchDefeatOutput       `json:"defeats"`
	RoleChanges   []roleChangeOutput        `json:"role_changes"`
	TruthChanges  []truthChangeOutput       `json:"truth_changes"`
	Changes       []changeOutput            `json:"changes"`
	Diagnostics   []diagnostic.Diagnostic   `json:"diagnostics"`
}

const batchAuthorExample = `{
  "schema_version": 2,
  "statements": [
    {"key": "puncture", "text": "A puncture marked the victim's thumb."},
    {"key": "groove", "text": "A dyed groove was present in the clasp."},
    {"key": "needle-theory", "text": "The clasp caused the puncture."},
    {"key": "alternative-cause", "role": "counterpoint", "text": "Another sharp object could have caused the puncture."}
  ],
  "derivations": [
    {
      "key": "needle-inference",
      "sources": [{"key": "puncture"}, {"key": "groove"}],
      "target": {"key": "needle-theory"}
    }
  ],
  "defeats": [
    {
      "from": {"key": "alternative-cause"},
      "scope": "inference",
      "target": {"key": "needle-inference"}
    }
  ]
}`

func runAddBatch(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("add-batch", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOutput := fs.Bool("json", false, "output versioned JSON")
	dryRun := fs.Bool("dry-run", false, "validate and report without saving")
	example := fs.Bool("example", false, "print a schema 2 transaction example and exit")
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
		fmt.Fprintln(stdout, batchAuthorExample)
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
	input, err := readBatchInput(strings.TrimSpace(*inputPath))
	if err != nil {
		return writeMutationFailure(stdout, *jsonOutput, profile, "batch_input_invalid", err.Error(), strings.TrimSpace(*inputPath))
	}
	if input.SchemaVersion != 2 {
		return writeMutationFailure(stdout, *jsonOutput, profile, "batch_schema_version_unsupported", fmt.Sprintf("unsupported batch schema_version %d; expected 2", input.SchemaVersion), strings.TrimSpace(*inputPath))
	}
	return applyBatch(fs.Arg(0), *dryRun, *jsonOutput, doc, profile, diagnostics, input, stdout)
}

func applyBatch(path string, dryRun, jsonOutput bool, doc *argument.Document, profile validation.Profile, diagnostics []diagnostic.Diagnostic, input batchAuthorInput, stdout io.Writer) error {
	options, err := batchAuthorOptions(input)
	if err != nil {
		return writeArgumentMutationFailure(stdout, jsonOutput, profile, err)
	}
	next, result, err := argument.AuthorBatch(doc, options)
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
	diagnostics = nonNilDiagnostics(validated.Diagnostics)
	statements := make([]batchAddStatementOutput, 0, len(result.Statements))
	derivations := make([]batchDerivationOutput, 0, len(result.Derivations))
	defeats := make([]batchDefeatOutput, 0, len(result.Defeats))
	roleChanges := make([]roleChangeOutput, 0, len(result.RoleChanges))
	truthChanges := make([]truthChangeOutput, 0, len(result.TruthChanges))
	changes := make([]changeOutput, 0, len(result.Statements)+len(result.Derivations)+len(result.Defeats)+len(result.RoleChanges)+2)
	for _, item := range result.Statements {
		statements = append(statements, batchAddStatementOutput{Key: item.Key, Statement: item.Statement})
		changes = append(changes, changeOutput{Operation: "added", ElementType: "statement", ID: item.Statement.ID})
	}
	for _, item := range result.Derivations {
		derivations = append(derivations, batchDerivationOutput{Key: item.Key, Target: item.Target, Junctor: item.Junctor})
		changes = append(changes, changeOutput{Operation: "added", ElementType: "junctor", ID: item.Junctor.ID})
		diagnostics = appendJunctorSizeAdvisory(diagnostics, item.Junctor)
	}
	for _, item := range result.Defeats {
		from := batchReferenceInput{Key: item.From.Key, ID: item.From.ID}
		defeats = append(defeats, batchDefeatOutput{From: from, Defeat: item.Defeat})
		changes = append(changes, changeOutput{Operation: "added", ElementType: "defeat", ID: item.Defeat.From})
	}
	for _, change := range result.RoleChanges {
		roleChanges = append(roleChanges, roleChangeOutput{PreviousID: change.PreviousID, CurrentID: change.CurrentID, From: change.From, To: change.To})
		changes = append(changes, changeOutput{Operation: "reidentified", ElementType: "statement", ID: change.CurrentID})
		diagnostics = append(diagnostics, diagnostic.Diagnostic{
			Code:     "external_id_references_unchecked",
			Message:  fmt.Sprintf("promoting %s to lemma %s rewrote references in this workspace and recognized root metadata only; references outside this workspace may require review", change.PreviousID, change.CurrentID),
			Severity: diagnostic.SeverityWarning, Element: change.CurrentID,
		})
	}
	for _, change := range result.TruthChanges {
		truthChanges = append(truthChanges, truthChangeOutput{ID: change.ID, From: change.From, To: change.To})
	}
	if result.RootMetadataUpdated {
		changes = append(changes, changeOutput{Operation: "updated", ElementType: "metadata", ID: "root"})
	}
	changes = appendMetadataChange(changes, nextIDsMetadataChange(doc, next))
	output := batchAuthorOutput{
		SchemaVersion: outputSchemaVersion, Action: "add-batch", DryRun: dryRun,
		Profile: profile, Document: documentSummary(next), Statements: statements,
		Derivations: derivations, Defeats: defeats, RoleChanges: roleChanges,
		TruthChanges: truthChanges, Changes: changes, Diagnostics: diagnostics,
	}
	return writeBatchAuthor(stdout, jsonOutput, output)
}

func batchAuthorOptions(input batchAuthorInput) (argument.AuthorBatchOptions, error) {
	options := argument.AuthorBatchOptions{
		Statements: []argument.BatchStatementSpec{}, Derivations: []argument.BatchDerivationSpec{}, Defeats: []argument.BatchDefeatSpec{},
	}
	failures := make([]argument.MutationError, 0)
	for index, item := range input.Statements {
		element := strings.TrimSpace(item.Key)
		if element == "" {
			element = fmt.Sprintf("statements[%d]", index)
		}
		kind := argument.KindFact
		if strings.TrimSpace(item.Kind) != "" {
			parsed, ok := parseKind(item.Kind)
			if !ok {
				failures = append(failures, argument.MutationError{Code: "kind_invalid", Message: fmt.Sprintf("invalid kind %q; expected fact or value", item.Kind), Element: element})
			} else {
				kind = parsed
			}
		}
		var truth *argument.Truth
		if strings.TrimSpace(item.Truth) != "" {
			parsed, ok := parseTruth(item.Truth)
			if !ok {
				failures = append(failures, argument.MutationError{Code: "truth_invalid", Message: fmt.Sprintf("invalid truth %q; expected T, F, or U", item.Truth), Element: element})
			} else {
				truth = &parsed
			}
		}
		var role *argument.Role
		if strings.TrimSpace(item.Role) != "" {
			parsed, ok := parseBatchRole(item.Role)
			if !ok {
				failures = append(failures, argument.MutationError{Code: "statement_role_invalid", Message: fmt.Sprintf("invalid role %q; expected premise, lemma, conclusion, or counterpoint", item.Role), Element: element})
			} else {
				role = &parsed
			}
		}
		options.Statements = append(options.Statements, argument.BatchStatementSpec{
			Key: item.Key, Text: item.Text, RequestedID: item.ID, Slug: item.Slug,
			Truth: truth, Kind: kind, Role: role,
		})
	}
	for _, item := range input.Derivations {
		sources := make([]argument.BatchReference, 0, len(item.Sources))
		for _, source := range item.Sources {
			sources = append(sources, argument.BatchReference{Key: source.Key, ID: source.ID})
		}
		options.Derivations = append(options.Derivations, argument.BatchDerivationSpec{
			Key: item.Key, Sources: sources, Target: argument.BatchReference{Key: item.Target.Key, ID: item.Target.ID},
		})
	}
	for index, item := range input.Defeats {
		scope, ok := parseDefeatScope(item.Scope)
		if !ok {
			element := strings.TrimSpace(item.From.Key + item.From.ID)
			if element == "" {
				element = fmt.Sprintf("defeats[%d]", index)
			}
			failures = append(failures, argument.MutationError{Code: "defeat_scope_invalid", Message: fmt.Sprintf("invalid defeat scope %q; expected premise, inference, or counterpoint", item.Scope), Element: element})
		}
		options.Defeats = append(options.Defeats, argument.BatchDefeatSpec{
			From: argument.BatchReference{Key: item.From.Key, ID: item.From.ID}, Scope: scope,
			Target: argument.BatchReference{Key: item.Target.Key, ID: item.Target.ID},
		})
	}
	if len(failures) > 0 {
		return options, &argument.MutationErrors{Failures: failures}
	}
	return options, nil
}

func parseBatchRole(value string) (argument.Role, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "premise":
		return argument.RolePremise, true
	case "lemma":
		return argument.RoleLemma, true
	case "conclusion":
		return argument.RoleConclusion, true
	case "counterpoint":
		return argument.RoleCounterpoint, true
	default:
		return argument.Role(value), false
	}
}

func parseDefeatScope(value string) (argument.DefeatScope, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "premise":
		return argument.DefeatPremise, true
	case "inference":
		return argument.DefeatInference, true
	case "counterpoint":
		return argument.DefeatCounterpoint, true
	default:
		return argument.DefeatScope(value), false
	}
}

func readBatchInput(path string) (batchAuthorInput, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return batchAuthorInput{}, fmt.Errorf("read batch input: %w", err)
	}
	var input batchAuthorInput
	if err := decodeBatchJSON(data, &input); err != nil {
		return batchAuthorInput{}, err
	}
	return input, nil
}

func decodeBatchJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode batch input: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("decode batch input: unexpected trailing JSON value")
		}
		return fmt.Errorf("decode batch input: %w", err)
	}
	return nil
}

func writeBatchAuthor(w io.Writer, jsonOutput bool, output batchAuthorOutput) error {
	if jsonOutput {
		return writeIndentedJSON(w, output)
	}
	fmt.Fprintf(w, "Authored %d statements, %d derivations, and %d defeats atomically\n", len(output.Statements), len(output.Derivations), len(output.Defeats))
	for _, item := range output.Statements {
		fmt.Fprintf(w, "statement %s -> %s:%s [%s]\n", item.Key, item.Statement.ID, item.Statement.Slug, item.Statement.Role)
	}
	for _, item := range output.Derivations {
		fmt.Fprintf(w, "derivation %s -> %s: AND(%s) -> %s\n", item.Key, item.Junctor.ID, strings.Join(item.Junctor.Sources, ", "), item.Junctor.Target)
	}
	for _, item := range output.Defeats {
		fmt.Fprintf(w, "defeat %s\n", formatDefeat(item.Defeat))
	}
	for _, change := range output.RoleChanges {
		fmt.Fprintf(w, "Promoted %s -> %s: %s -> %s\n", change.PreviousID, change.CurrentID, change.From, change.To)
	}
	if output.DryRun {
		fmt.Fprintln(w, "dry-run: mappings are tentative; no file changes written")
	}
	for _, item := range output.Diagnostics {
		writeDiagnostic(w, item)
	}
	return nil
}

func writeAddBatchUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cludia add-batch [--dry-run] [--json] FILE --input FILE")
	fmt.Fprintln(w, "       cludia add-batch --example")
	fmt.Fprintln(w, "Schema 2 atomically creates statements, AND derivations, and typed defeats.")
	fmt.Fprintln(w, "References name new elements as {\"key\":\"...\"} and pre-existing durable elements as {\"id\":\"P1\"}.")
	fmt.Fprintln(w, "Statement id, slug, truth (T|F|U), kind (fact|value), and role fields are optional.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Transaction example (`add-batch --example` prints this form):")
	fmt.Fprintln(w, batchAuthorExample)
}
