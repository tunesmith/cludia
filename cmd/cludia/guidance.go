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
	ScriptedAuthoring   scriptedAuthoringGuidance   `json:"scripted_authoring"`
	Deletion            deletionGuidance            `json:"deletion"`
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

type scriptedAuthoringGuidance struct {
	PredictGeneratedIDs                   bool   `json:"predict_generated_ids"`
	ExplicitIDFlag                        string `json:"explicit_id_flag"`
	AddResultIDField                      string `json:"add_result_id_field"`
	SuccessfulMutationResultAuthoritative bool   `json:"successful_mutation_result_authoritative"`
	AtomicBatchCommand                    string `json:"atomic_batch_command"`
	BatchInputSchemaVersion               int    `json:"batch_input_schema_version"`
	BatchResultMappingField               string `json:"batch_result_mapping_field"`
	BatchDryRunMappingTentative           bool   `json:"batch_dry_run_mapping_tentative"`
}

type deletionGuidance struct {
	DryRunAvailable                  bool   `json:"dry_run_available"`
	DryRunFlag                       string `json:"dry_run_flag"`
	RemoveAttachedCounterpointsFirst bool   `json:"remove_attached_counterpoints_first"`
	CounterpointRemovalCommand       string `json:"counterpoint_removal_command"`
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
	fmt.Fprintln(stdout, "- Scripted capture must not predict generated IDs across mutations; pass --id or read statement.id from each successful add --json result.")
	fmt.Fprintln(stdout, "- Use add-batch with versioned JSON input for all-or-nothing multi-statement capture and caller-key-to-statement mappings.")
	fmt.Fprintln(stdout, "- A batch dry-run mapping is tentative; use IDs from the applied mutation result for later references.")
	fmt.Fprintln(stdout, "- Before deleting a challenged element, remove attached counterpoints with remove-counterpoint; use delete --dry-run to inspect structural effects.")
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
		ScriptedAuthoring: scriptedAuthoringGuidance{
			PredictGeneratedIDs: false, ExplicitIDFlag: "--id", AddResultIDField: "statement.id",
			SuccessfulMutationResultAuthoritative: true, AtomicBatchCommand: "add-batch",
			BatchInputSchemaVersion: 1, BatchResultMappingField: "statements", BatchDryRunMappingTentative: true,
		},
		Deletion: deletionGuidance{
			DryRunAvailable: true, DryRunFlag: "--dry-run", RemoveAttachedCounterpointsFirst: true,
			CounterpointRemovalCommand: "remove-counterpoint",
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
