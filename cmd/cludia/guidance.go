package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
)

type guidanceOutput struct {
	SchemaVersion       int                         `json:"schema_version"`
	UseCaseNeutral      bool                        `json:"use_case_neutral"`
	StatementIdentity   statementIdentityGuidance   `json:"statement_identity"`
	TextEdits           textEditGuidance            `json:"text_edits"`
	Slugs               slugGuidance                `json:"slugs"`
	MaterialReplacement materialReplacementGuidance `json:"material_replacement"`
}

type statementIdentityGuidance struct {
	IDRequired                    bool   `json:"id_required"`
	IDMeaning                     string `json:"id_meaning"`
	MateriallyDifferentGetsNewID  bool   `json:"materially_different_gets_new_id"`
	SemanticEquivalenceMechanical bool   `json:"semantic_equivalence_mechanical"`
}

type textEditGuidance struct {
	SamePropositionRequired bool   `json:"same_proposition_required"`
	Flag                    string `json:"flag"`
	TruthKindRequireFlag    bool   `json:"truth_kind_require_flag"`
}

type slugGuidance struct {
	Optional             bool     `json:"optional"`
	Mutable              bool     `json:"mutable"`
	OldAliasesRetained   bool     `json:"old_aliases_retained"`
	RelationsUseIDs      bool     `json:"relations_use_ids"`
	Operations           []string `json:"operations"`
	ExternalScopeWarning bool     `json:"external_scope_warning"`
}

type materialReplacementGuidance struct {
	EditCommandAppropriate bool     `json:"edit_command_appropriate"`
	CurrentPrimitiveSteps  []string `json:"current_primitive_steps"`
	FutureDryRunRequired   bool     `json:"future_dry_run_required"`
	AutomaticRetargetAll   bool     `json:"automatic_retarget_all"`
}

func runGuidance(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("guidance", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOutput := fs.Bool("json", false, "output versioned JSON")
	fs.Usage = func() { writeGuidanceUsage(fs.Output()) }
	if err := fs.Parse(flagsFirst(args, map[string]bool{})); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		fs.Usage()
		return fmt.Errorf("guidance does not accept positional arguments")
	}
	output := identityGuidance()
	if *jsonOutput {
		return writeIndentedJSON(stdout, output)
	}
	fmt.Fprintln(stdout, "Statement identity contract:")
	fmt.Fprintln(stdout, "- IDs are required durable proposition-record identities.")
	fmt.Fprintln(stdout, "- Text edits require --same-proposition; Cludia does not verify semantic equivalence.")
	fmt.Fprintln(stdout, "- Truth- and kind-only edits do not require --same-proposition.")
	fmt.Fprintln(stdout, "- Slugs are optional mutable aliases with one current value and no retained alias history.")
	fmt.Fprintln(stdout, "- Materially different propositions receive new IDs; audit each relation before retargeting.")
	fmt.Fprintln(stdout, "- These rules do not assume any particular workspace organization or use case.")
	return nil
}

func identityGuidance() guidanceOutput {
	return guidanceOutput{
		SchemaVersion:  outputSchemaVersion,
		UseCaseNeutral: true,
		StatementIdentity: statementIdentityGuidance{
			IDRequired: true, IDMeaning: "durable proposition record identity",
			MateriallyDifferentGetsNewID: true, SemanticEquivalenceMechanical: false,
		},
		TextEdits: textEditGuidance{
			SamePropositionRequired: true, Flag: "--same-proposition", TruthKindRequireFlag: false,
		},
		Slugs: slugGuidance{
			Optional: true, Mutable: true, OldAliasesRetained: false, RelationsUseIDs: true,
			Operations: []string{"--slug", "--from-text", "--clear"}, ExternalScopeWarning: true,
		},
		MaterialReplacement: materialReplacementGuidance{
			EditCommandAppropriate: false,
			CurrentPrimitiveSteps:  []string{"add new statement", "audit and repair relations", "retain or delete old statement"},
			FutureDryRunRequired:   true, AutomaticRetargetAll: false,
		},
	}
}

func writeGuidanceUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cludia guidance [--json]")
	fmt.Fprintln(w, "Explain the use-case-neutral statement identity, edit, slug, and replacement contract.")
}
