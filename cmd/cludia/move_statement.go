package main

import (
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/tunesmith/cludia/internal/argfile"
	"github.com/tunesmith/cludia/internal/argument"
	"github.com/tunesmith/cludia/internal/diagnostic"
	"github.com/tunesmith/cludia/internal/validation"
)

type statementMoveOutput struct {
	SchemaVersion    int                     `json:"schema_version"`
	Action           string                  `json:"action"`
	Profile          validation.Profile      `json:"profile"`
	Document         documentOutput          `json:"document"`
	Statement        argument.Statement      `json:"statement"`
	Anchor           argument.Statement      `json:"anchor"`
	Placement        argument.MovePlacement  `json:"placement"`
	PreviousPosition int                     `json:"previous_position"`
	CurrentPosition  int                     `json:"current_position"`
	Changes          []changeOutput          `json:"changes"`
	Diagnostics      []diagnostic.Diagnostic `json:"diagnostics"`
}

func runMoveStatement(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("move-statement", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOutput := fs.Bool("json", false, "output versioned JSON")
	var before, after optionalStringFlag
	fs.Var(&before, "before", "move immediately before this anchor statement")
	fs.Var(&after, "after", "move immediately after this anchor statement")
	fs.Usage = func() { writeMoveStatementUsage(fs.Output()) }
	if err := fs.Parse(flagsFirst(args, map[string]bool{"before": true, "after": true})); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 2 {
		fs.Usage()
		return fmt.Errorf("move-statement expects a file and statement")
	}
	if before.set == after.set {
		fs.Usage()
		return fmt.Errorf("move-statement requires exactly one of --before or --after")
	}

	placement := argument.MoveBefore
	anchorRef := before.value
	if after.set {
		placement = argument.MoveAfter
		anchorRef = after.value
	}
	doc, profile, diagnostics := loadValidated(fs.Arg(0))
	if diagnostic.HasErrors(diagnostics) {
		if err := writeFailure(stdout, *jsonOutput, profile, diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	next, move, err := argument.MoveStatement(doc, fs.Arg(1), anchorRef, placement)
	if err != nil {
		if moveErr, ok := err.(*argument.StatementMoveError); ok {
			return writeMutationFailure(stdout, *jsonOutput, profile, moveErr.Code, moveErr.Message, moveErr.Element)
		}
		return err
	}
	validated := validation.Validate(next, profile)
	if !validated.OK() {
		if err := writeFailure(stdout, *jsonOutput, profile, validated.Diagnostics); err != nil {
			return err
		}
		return errValidationFailed
	}
	if move.Changed {
		if err := argfile.SaveAtomic(fs.Arg(0), next); err != nil {
			return err
		}
	}
	changes := []changeOutput{}
	if move.Changed {
		changes = append(changes, changeOutput{Operation: "reordered", ElementType: "statement", ID: move.Statement.ID})
	}
	diagnostics = validated.Diagnostics
	if diagnostics == nil {
		diagnostics = []diagnostic.Diagnostic{}
	}
	output := statementMoveOutput{
		SchemaVersion: outputSchemaVersion, Action: "move-statement", Profile: profile,
		Document: documentSummary(next), Statement: move.Statement, Anchor: move.Anchor, Placement: move.Placement,
		PreviousPosition: move.PreviousPosition, CurrentPosition: move.CurrentPosition,
		Changes: changes, Diagnostics: diagnostics,
	}
	return writeStatementMove(stdout, *jsonOutput, output)
}

func writeStatementMove(w io.Writer, jsonOutput bool, output statementMoveOutput) error {
	if jsonOutput {
		return writeIndentedJSON(w, output)
	}
	if len(output.Changes) == 0 {
		fmt.Fprintf(w, "%s is already immediately %s %s at statement position %d\n", output.Statement.ID, output.Placement, output.Anchor.ID, output.CurrentPosition)
	} else {
		fmt.Fprintf(w, "Moved %s %s %s (statement position %d -> %d)\n", output.Statement.ID, output.Placement, output.Anchor.ID, output.PreviousPosition, output.CurrentPosition)
	}
	for _, item := range output.Diagnostics {
		writeDiagnostic(w, item)
	}
	return nil
}

func writeMoveStatementUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cludia move-statement [--json] FILE STATEMENT (--before STATEMENT | --after STATEMENT)")
	fmt.Fprintln(w, "Move one statement in durable document order without changing its identity or relations.")
}
