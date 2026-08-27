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

const outputSchemaVersion = 1

var (
	version             = "dev"
	errValidationFailed = errors.New("validation failed")
)

type validateOutput struct {
	SchemaVersion int                     `json:"schema_version"`
	OK            bool                    `json:"ok"`
	Profile       validation.Profile      `json:"profile"`
	Document      documentOutput          `json:"document"`
	Stats         statsOutput             `json:"stats"`
	Diagnostics   []diagnostic.Diagnostic `json:"diagnostics"`
}

type documentOutput struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type statsOutput struct {
	Statements     int `json:"statements"`
	Junctors       int `json:"junctors"`
	DirectSupports int `json:"direct_supports"`
	Defeats        int `json:"defeats"`
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		if !errors.Is(err, errValidationFailed) {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		writeTopLevelUsage(stderr)
		return fmt.Errorf("expected a command")
	}
	switch args[0] {
	case "init":
		return runInit(args[1:], stdout, stderr)
	case "add":
		return runAdd(args[1:], stdout, stderr)
	case "add-batch":
		return runAddBatch(args[1:], stdout, stderr)
	case "edit":
		return runEdit(args[1:], stdout, stderr)
	case "derive":
		return runDerive(args[1:], stdout, stderr)
	case "list":
		return runList(args[1:], stdout, stderr)
	case "show":
		return runShow(args[1:], stdout, stderr)
	case "components":
		return runComponents(args[1:], stdout, stderr)
	case "component":
		return runComponent(args[1:], stdout, stderr)
	case "search":
		return runSearch(args[1:], stdout, stderr)
	case "top":
		return runTop(args[1:], stdout, stderr)
	case "ledger":
		return runLedger(args[1:], stdout, stderr)
	case "add-source":
		return runAddSource(args[1:], stdout, stderr)
	case "remove-source":
		return runRemoveSource(args[1:], stdout, stderr)
	case "replace-source":
		return runReplaceSource(args[1:], stdout, stderr)
	case "remove-junctor":
		return runRemoveJunctor(args[1:], stdout, stderr)
	case "undermine":
		return runUndermine(args[1:], stdout, stderr)
	case "undercut":
		return runUndercut(args[1:], stdout, stderr)
	case "counterpoint":
		return runCounterpoint(args[1:], stdout, stderr)
	case "remove-counterpoint":
		return runRemoveCounterpoint(args[1:], stdout, stderr)
	case "delete":
		return runDelete(args[1:], stdout, stderr)
	case "replace":
		return runReplace(args[1:], stdout, stderr)
	case "root":
		return runRoot(args[1:], stdout, stderr)
	case "export":
		return runExport(args[1:], stdout, stderr)
	case "rename-slug":
		return runRenameSlug(args[1:], stdout, stderr)
	case "guidance":
		return runGuidance(args[1:], stdout, stderr)
	case "validate", "check":
		return runValidate(args[1:], stdout, stderr)
	case "help":
		writeTopLevelUsage(stdout)
		return nil
	case "version", "--version", "-version":
		fmt.Fprintf(stdout, "cludia %s\n", version)
		return nil
	default:
		writeTopLevelUsage(stderr)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runValidate(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOutput := fs.Bool("json", false, "output versioned JSON")
	profileName := fs.String("profile", "", "validation profile: workspace or concludia")
	fs.Usage = func() { writeValidateUsage(fs.Output()) }
	if err := fs.Parse(flagsFirst(args, map[string]bool{"profile": true})); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("validate expects exactly one file")
	}

	parsed := argfile.Load(fs.Arg(0))
	profile := selectedProfile(parsed.Document, *profileName)
	diagnostics := append([]diagnostic.Diagnostic(nil), parsed.Diagnostics...)
	if !diagnostic.HasErrors(diagnostics) {
		validated := validation.Validate(parsed.Document, profile)
		diagnostics = append(diagnostics, validated.Diagnostics...)
	}
	output := makeValidateOutput(parsed.Document, profile, diagnostics)
	if *jsonOutput {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(output); err != nil {
			return err
		}
	} else {
		writeHumanValidation(stdout, output)
	}
	if !output.OK {
		return errValidationFailed
	}
	return nil
}

func selectedProfile(doc *argument.Document, override string) validation.Profile {
	if override != "" {
		return validation.Profile(strings.ToLower(override))
	}
	if doc != nil {
		if profile, ok := doc.MetadataValue("profile"); ok && strings.EqualFold(profile, string(validation.ProfileWorkspace)) {
			return validation.ProfileWorkspace
		}
	}
	return validation.ProfileConcludia
}

func makeValidateOutput(doc *argument.Document, profile validation.Profile, diagnostics []diagnostic.Diagnostic) validateOutput {
	if diagnostics == nil {
		diagnostics = []diagnostic.Diagnostic{}
	}
	output := validateOutput{
		SchemaVersion: outputSchemaVersion,
		OK:            !diagnostic.HasErrors(diagnostics),
		Profile:       profile,
		Diagnostics:   diagnostics,
	}
	if doc != nil {
		output.Document = documentOutput{ID: doc.ID, Title: doc.Title}
		output.Stats = statsOutput{
			Statements: len(doc.Statements), Junctors: len(doc.Junctors),
			DirectSupports: len(doc.DirectSupports), Defeats: len(doc.Defeats),
		}
	}
	return output
}

func writeHumanValidation(w io.Writer, output validateOutput) {
	status := "OK"
	if !output.OK {
		status = "INVALID"
	}
	fmt.Fprintf(w, "%s (%s profile)\n", status, output.Profile)
	if output.Document.ID != "" {
		fmt.Fprintf(w, "document: %s — %s\n", output.Document.ID, output.Document.Title)
	}
	fmt.Fprintf(w, "statements: %d\n", output.Stats.Statements)
	fmt.Fprintf(w, "junctors: %d\n", output.Stats.Junctors)
	fmt.Fprintf(w, "direct_supports: %d\n", output.Stats.DirectSupports)
	fmt.Fprintf(w, "defeats: %d\n", output.Stats.Defeats)
	for _, item := range output.Diagnostics {
		location := ""
		if item.Line > 0 {
			location = fmt.Sprintf(" line %d", item.Line)
		}
		if item.Element != "" {
			location += " " + item.Element
		}
		fmt.Fprintf(w, "%s [%s]%s: %s\n", strings.ToUpper(string(item.Severity)), item.Code, location, item.Message)
	}
}

func writeTopLevelUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  cludia init [--json] FILE --title TITLE --text TEXT")
	fmt.Fprintln(w, "  cludia add [--json] FILE --text TEXT")
	fmt.Fprintln(w, "  cludia add-batch [--dry-run] [--json] FILE --input FILE")
	fmt.Fprintln(w, "  cludia edit [--json] FILE STATEMENT --text TEXT --same-proposition")
	fmt.Fprintln(w, "  cludia derive [--json] FILE --source STATEMENT --source STATEMENT (--target STATEMENT | --target-text TEXT)")
	fmt.Fprintln(w, "  cludia list [--state all|isolated] [--json] FILE")
	fmt.Fprintln(w, "  cludia show [--relations] [--json] FILE ELEMENT")
	fmt.Fprintln(w, "  cludia components [--json] FILE")
	fmt.Fprintln(w, "  cludia component [--json] FILE ELEMENT")
	fmt.Fprintln(w, "  cludia search [--json] FILE QUERY")
	fmt.Fprintln(w, "  cludia top [--json] FILE")
	fmt.Fprintln(w, "  cludia ledger [--json] FILE STATEMENT")
	fmt.Fprintln(w, "  cludia add-source [--json] FILE JUNCTOR --source STATEMENT")
	fmt.Fprintln(w, "  cludia remove-source [--dry-run] [--json] FILE JUNCTOR --source STATEMENT")
	fmt.Fprintln(w, "  cludia replace-source [--dry-run] [--json] FILE JUNCTOR --from STATEMENT --to STATEMENT")
	fmt.Fprintln(w, "  cludia remove-junctor [--dry-run] [--json] FILE JUNCTOR")
	fmt.Fprintln(w, "  cludia undermine [--json] FILE PREMISE --text TEXT")
	fmt.Fprintln(w, "  cludia undercut [--json] FILE JUNCTOR --text TEXT")
	fmt.Fprintln(w, "  cludia counterpoint [--json] FILE COUNTERPOINT --text TEXT")
	fmt.Fprintln(w, "  cludia remove-counterpoint [--dry-run] [--json] FILE COUNTERPOINT")
	fmt.Fprintln(w, "  cludia delete [--dry-run] [--json] FILE STATEMENT")
	fmt.Fprintln(w, "  cludia replace [--json] FILE OLD --with NEW [choices] (--dry-run | --apply-token TOKEN)")
	fmt.Fprintln(w, "  cludia root [--json] FILE STATEMENT")
	fmt.Fprintln(w, "  cludia export [--json] FILE --root STATEMENT --output FILE")
	fmt.Fprintln(w, "  cludia rename-slug [--json] FILE STATEMENT (--slug SLUG | --from-text | --clear)")
	fmt.Fprintln(w, "  cludia guidance [--json]")
	fmt.Fprintln(w, "  cludia validate [--profile PROFILE] [--json] FILE")
	fmt.Fprintln(w, "  cludia check [--profile PROFILE] [--json] FILE")
	fmt.Fprintln(w, "  cludia version")
}

func writeValidateUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cludia validate [--profile workspace|concludia] [--json] FILE")
	fmt.Fprintln(w, "Parse and structurally validate an .arg file.")
}

func flagsFirst(args []string, valueFlags map[string]bool) []string {
	flags := make([]string, 0, len(args))
	positionals := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positionals = append(positionals, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			positionals = append(positionals, arg)
			continue
		}
		flags = append(flags, arg)
		name := strings.TrimLeft(arg, "-")
		if before, _, found := strings.Cut(name, "="); found {
			name = before
		}
		if valueFlags[name] && !strings.Contains(arg, "=") && i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}
	return append(flags, positionals...)
}
