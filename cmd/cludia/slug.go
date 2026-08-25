package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/tunesmith/cludia/internal/argfile"
	"github.com/tunesmith/cludia/internal/argument"
	"github.com/tunesmith/cludia/internal/diagnostic"
	"github.com/tunesmith/cludia/internal/validation"
)

type slugMutationOutput struct {
	SchemaVersion       int                     `json:"schema_version"`
	Action              string                  `json:"action"`
	Profile             validation.Profile      `json:"profile"`
	Document            documentOutput          `json:"document"`
	Statement           argument.Statement      `json:"statement"`
	PreviousSlug        string                  `json:"previous_slug"`
	CurrentSlug         string                  `json:"current_slug"`
	RootMetadataUpdated bool                    `json:"root_metadata_updated"`
	ScopeChecked        []string                `json:"scope_checked"`
	Changes             []changeOutput          `json:"changes"`
	Diagnostics         []diagnostic.Diagnostic `json:"diagnostics"`
}

func runRenameSlug(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("rename-slug", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOutput := fs.Bool("json", false, "output versioned JSON")
	var explicit optionalStringFlag
	fs.Var(&explicit, "slug", "new explicit slug")
	fromText := fs.Bool("from-text", false, "generate a unique slug from current statement text")
	clearSlug := fs.Bool("clear", false, "remove the optional slug")
	fs.Usage = func() { writeRenameSlugUsage(fs.Output()) }
	if err := fs.Parse(flagsFirst(args, map[string]bool{"slug": true})); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 2 {
		fs.Usage()
		return fmt.Errorf("rename-slug expects a file and statement")
	}
	choices := 0
	if explicit.set {
		choices++
	}
	if *fromText {
		choices++
	}
	if *clearSlug {
		choices++
	}
	if choices != 1 {
		fs.Usage()
		return fmt.Errorf("rename-slug requires exactly one of --slug, --from-text, or --clear")
	}
	doc, profile, diagnostics := loadValidated(fs.Arg(0))
	if diagnostic.HasErrors(diagnostics) {
		if err := writeFailure(stdout, *jsonOutput, profile, diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	next := doc.Clone()
	statement, ok := next.Statement(fs.Arg(1))
	if !ok {
		return writeMutationFailure(stdout, *jsonOutput, profile, "statement_not_found", fmt.Sprintf("statement %q not found", fs.Arg(1)), fs.Arg(1))
	}
	previous := statement.Slug
	current := ""
	switch {
	case explicit.set:
		current = strings.TrimSpace(explicit.value)
		if current == "" {
			return fmt.Errorf("--slug must not be empty; use --clear to remove a slug")
		}
	case *fromText:
		statement.Slug = ""
		current = argument.UniqueSlug(next, statement.Text)
	case *clearSlug:
		current = ""
	}
	statement.Slug = current
	rootUpdated := false
	if previous != "" && previous != current {
		for i := range next.Metadata {
			metadata := &next.Metadata[i]
			if metadata.Key == "root" && metadata.Value == previous {
				metadata.Value = current
				if current == "" {
					metadata.Value = statement.ID
				}
				rootUpdated = true
			}
		}
	}
	validated := validation.Validate(next, profile)
	if !validated.OK() {
		if err := writeFailure(stdout, *jsonOutput, profile, validated.Diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	changed := previous != current
	if changed {
		if err := argfile.SaveAtomic(fs.Arg(0), next); err != nil {
			return err
		}
	}
	diagnostics = validated.Diagnostics
	if diagnostics == nil {
		diagnostics = []diagnostic.Diagnostic{}
	}
	if changed {
		diagnostics = append(diagnostics, diagnostic.Diagnostic{
			Code:     "external_slug_references_unchecked",
			Message:  "only the workspace file and recognized metadata were checked; unknown external slug references may require updates",
			Severity: diagnostic.SeverityWarning, Element: statement.ID,
		})
	}
	changes := []changeOutput{}
	if changed {
		changes = append(changes, changeOutput{Operation: "updated", ElementType: "statement", ID: statement.ID})
	}
	if rootUpdated {
		changes = append(changes, changeOutput{Operation: "updated", ElementType: "metadata", ID: "root"})
	}
	output := slugMutationOutput{
		SchemaVersion: outputSchemaVersion, Action: "rename-slug", Profile: profile,
		Document: documentSummary(next), Statement: *statement,
		PreviousSlug: previous, CurrentSlug: current, RootMetadataUpdated: rootUpdated,
		ScopeChecked: []string{"workspace_file"}, Changes: changes, Diagnostics: diagnostics,
	}
	if *jsonOutput {
		return writeIndentedJSON(stdout, output)
	}
	fmt.Fprintf(stdout, "Renamed slug for %s: %q -> %q\n", statement.ID, previous, current)
	if rootUpdated {
		fmt.Fprintln(stdout, "Updated root metadata reference")
	}
	for _, item := range diagnostics {
		writeDiagnostic(stdout, item)
	}
	return nil
}

func writeRenameSlugUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cludia rename-slug [--json] FILE STATEMENT (--slug SLUG | --from-text | --clear)")
	fmt.Fprintln(w, "Refactor the optional current slug without changing statement identity or durable relations.")
}
