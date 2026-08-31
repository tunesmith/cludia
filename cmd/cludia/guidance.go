// SPDX-FileCopyrightText: 2026 KeenWorks
// SPDX-License-Identifier: GPL-3.0-or-later

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
	StatementAuthoring  statementAuthoringGuidance  `json:"statement_authoring"`
	TruthEvaluation     truthEvaluationGuidance     `json:"truth_evaluation"`
	DefeatAuthoring     defeatAuthoringGuidance     `json:"defeat_authoring"`
	StatementIdentity   statementIdentityGuidance   `json:"statement_identity"`
	TextEdits           textEditGuidance            `json:"text_edits"`
	Slugs               slugGuidance                `json:"slugs"`
	ScriptedAuthoring   scriptedAuthoringGuidance   `json:"scripted_authoring"`
	Deletion            deletionGuidance            `json:"deletion"`
	MaterialReplacement materialReplacementGuidance `json:"material_replacement"`
	Renumbering         renumberingGuidance         `json:"renumbering"`
}

type truthEvaluationGuidance struct {
	AuthoredTruthLeafOnly bool     `json:"authored_truth_leaf_only"`
	ManualRoles           []string `json:"manual_roles"`
	EffectiveCalculated   bool     `json:"effective_truth_calculated"`
	EffectivePersisted    bool     `json:"effective_truth_persisted"`
	EvaluationVersion     int      `json:"evaluation_version"`
	Mode                  string   `json:"mode"`
	DefeatsIncluded       bool     `json:"defeats_included"`
	EvaluateCommand       string   `json:"evaluate_command"`
	NormalizeCommand      string   `json:"normalize_command"`
}

type statementAuthoringGuidance struct {
	TruthAptRequired                bool   `json:"truth_apt_required"`
	QuestionsSupported              bool   `json:"questions_supported"`
	QuestionAlternative             string `json:"question_alternative"`
	HypothesesAsUnknownPropositions bool   `json:"hypotheses_as_unknown_propositions"`
	UnknownTruthFlag                string `json:"unknown_truth_flag"`
	DefaultTruth                    string `json:"default_truth"`
	ConfidenceSupported             bool   `json:"confidence_supported"`
}

type defeatAuthoringGuidance struct {
	ChangesEffectiveTruth      bool     `json:"changes_effective_truth"`
	GroundedAcceptanceApplies  bool     `json:"grounded_acceptance_applies"`
	UndermineWhen              string   `json:"undermine_when"`
	UndercutWhen               string   `json:"undercut_when"`
	CounterpointWhen           string   `json:"counterpoint_when"`
	CaveatAutomaticallyDefeats bool     `json:"caveat_automatically_defeats"`
	NonDefeatExamples          []string `json:"non_defeat_examples"`
	NonDefeatAlternatives      []string `json:"non_defeat_alternatives"`
	InspectionCommand          string   `json:"inspection_command"`
}

type statementIdentityGuidance struct {
	IDRequired                      bool     `json:"id_required"`
	IDMeaning                       string   `json:"id_meaning"`
	MateriallyDifferentGetsNewID    bool     `json:"materially_different_gets_new_id"`
	SemanticEquivalenceMechanical   bool     `json:"semantic_equivalence_mechanical"`
	RolePrefixesCurrent             bool     `json:"role_prefixes_current"`
	PremisePromotionChangesID       bool     `json:"premise_promotion_changes_id"`
	PremisePromotionMappingFields   []string `json:"premise_promotion_mapping_fields"`
	PremisePromotionExternalWarning bool     `json:"premise_promotion_external_warning"`
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
	ExplicitIDMustEqualNext               bool   `json:"explicit_id_must_equal_next"`
	CustomIDsAuthored                     bool   `json:"custom_ids_authored"`
	NextIDsMetadata                       string `json:"next_ids_metadata"`
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
	EditCommandAppropriate  bool     `json:"edit_command_appropriate"`
	CurrentPrimitiveSteps   []string `json:"current_primitive_steps"`
	Command                 string   `json:"command"`
	TwoPhase                bool     `json:"two_phase"`
	DryRunRequired          bool     `json:"dry_run_required"`
	ApplyTokenRequired      bool     `json:"apply_token_required"`
	RelationChoicesExplicit bool     `json:"relation_choices_explicit"`
	OldRetainedByDefault    bool     `json:"old_retained_by_default"`
	AutomaticRetargetAll    bool     `json:"automatic_retarget_all"`
}

type renumberingGuidance struct {
	Command                     string `json:"command"`
	OrdinaryAllocationMonotonic bool   `json:"ordinary_allocation_monotonic"`
	DeletedIDsReusedOrdinarily  bool   `json:"deleted_ids_reused_ordinarily"`
	SoleNumberingReset          bool   `json:"sole_numbering_reset"`
	TwoPhase                    bool   `json:"two_phase"`
	ApplyTokenRequired          bool   `json:"apply_token_required"`
	CompleteMapping             bool   `json:"complete_mapping"`
	ExternalScopeWarning        bool   `json:"external_scope_warning"`
	TUIHotkey                   bool   `json:"tui_hotkey"`
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
	fmt.Fprintln(stdout, "Statement authoring contract:")
	fmt.Fprintln(stdout, "- Capture accepted facts and values as truth-apt propositions, not questions or prompts.")
	fmt.Fprintln(stdout, "- Disconnected premises are ordinary investigation state: they may later prove useful, remain irrelevant, or turn out to be red herrings.")
	fmt.Fprintln(stdout, "- Keep questions in conversation or adjacent notes until they can be stated as propositions.")
	fmt.Fprintln(stdout, "- Keep speculative hypotheses, rival explanations, and brainstorming outside Cludia until explicit recorded premises are intended to prove them.")
	fmt.Fprintln(stdout, "- Use --truth U for a recorded leaf proposition whose truth is genuinely uncertain, not merely for a theory that may or may not be true; capture defaults to --truth T.")
	fmt.Fprintln(stdout, "- Cludia does not author confidence scores or probabilities.")
	fmt.Fprintln(stdout, "- Only leaf premises and leaf counterpoints carry authored truth; sourced statements store U.")
	fmt.Fprintln(stdout, "- Effective truth is calculated with grounded three-valued support and defeat semantics and is never persisted as cache state.")
	fmt.Fprintln(stdout, "- Human reads show premise/counterpoint truth as T/F/U and lemma/conclusion provability as ⊢/◇/⊬ (proven/possibly proven/not proven).")
	fmt.Fprintln(stdout, "- A derived ⊬ means the current argument does not prove the proposition; it does not prove the proposition false.")
	fmt.Fprintln(stdout, "- Use evaluate to inspect the complete overlay and normalize-truth to repair legacy sourced T/F tokens.")
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Defeat authoring contract:")
	fmt.Fprintln(stdout, "- A defeat is a semantic relation with grounded truth consequences, not a caution label or annotation.")
	fmt.Fprintln(stdout, "- Ground a counterpoint in case-specific record information that identifies an actual defect in the exact premise or inference it targets.")
	fmt.Fprintln(stdout, "- Undermine a premise only when accepting the counterpoint would make that premise false or materially out of scope.")
	fmt.Fprintln(stdout, "- Undercut an inference only when accepting the counterpoint means those sources do not suffice for that target.")
	fmt.Fprintln(stdout, "- Counterpoint a counterpoint only when the new statement defeats the earlier objection.")
	fmt.Fprintln(stdout, "- Bare possibility, an unsupported rival explanation, absence of direct proof, or residual uncertainty is not automatically a defeat.")
	fmt.Fprintln(stdout, "- Keep a mere qualification in conversation or an adjacent note; if it exposes a real defect, repair or narrow the exact inference.")
	fmt.Fprintln(stdout, "- Use evaluate to inspect which counterpoints are IN, OUT, or UNDECIDED and which truths they change.")
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Statement identity contract:")
	fmt.Fprintln(stdout, "- IDs are required durable proposition-record identities.")
	fmt.Fprintln(stdout, "- Text edits require --same-proposition; Cludia does not verify semantic equivalence.")
	fmt.Fprintln(stdout, "- Truth- and kind-only edits do not require --same-proposition.")
	fmt.Fprintln(stdout, "- Slugs are optional mutable aliases with one current value and no retained alias history.")
	fmt.Fprintln(stdout, "- Focused authoring uses monotonic canonical P/L/C/CP/J IDs; deleted numbers are not reused during ordinary mutation.")
	fmt.Fprintln(stdout, "- Promoting an existing premise to a lemma retires its P ID, assigns the next L ID, rewrites modeled references, and reports previous_id/current_id plus an external-reference warning.")
	fmt.Fprintln(stdout, "- Scripted capture must not predict generated IDs across mutations; omit --id and read statement.id from each successful add --json result.")
	fmt.Fprintln(stdout, "- An explicit ID is accepted only when it is the role-appropriate exact next ID recorded by cludia-next-ids.")
	fmt.Fprintln(stdout, "- Use add-batch schema 2 for all-or-nothing statement, AND-derivation, and typed-defeat authoring with caller-key mappings.")
	fmt.Fprintln(stdout, "- In schema 2 relations, {\"key\":\"...\"} names a new batch element and {\"id\":\"P1\"} names a pre-existing durable element; slugs and tentative generated IDs are not references.")
	fmt.Fprintln(stdout, "- New schema 2 derivation targets receive their final L IDs directly; leave derivations and defeats empty for statement-only capture.")
	fmt.Fprintln(stdout, "- A batch dry-run mapping is tentative; use IDs from the applied mutation result for later references.")
	fmt.Fprintln(stdout, "- Before deleting a challenged element, remove attached counterpoints with remove-counterpoint; use delete --dry-run to inspect structural effects.")
	fmt.Fprintln(stdout, "- Materially different propositions receive new IDs; audit each relation before retargeting.")
	fmt.Fprintln(stdout, "- Material replacement uses replace --dry-run followed by the same explicit relation choices and --apply-token; there is no retarget-all mode.")
	fmt.Fprintln(stdout, "- Identifier compaction uses renumber --dry-run followed by --apply-token; it is the sole numbering reset and reports a complete mapping plus external-reference warning.")
	fmt.Fprintln(stdout, "- These rules do not assume any particular workspace organization or use case.")
	return nil
}

func identityGuidance() guidanceOutput {
	return guidanceOutput{
		SchemaVersion:  outputSchemaVersion,
		UseCaseNeutral: true,
		StatementAuthoring: statementAuthoringGuidance{
			TruthAptRequired: true, QuestionsSupported: false,
			QuestionAlternative:             "conversation or adjacent notes",
			HypothesesAsUnknownPropositions: false, UnknownTruthFlag: "--truth U",
			DefaultTruth: "T", ConfidenceSupported: false,
		},
		TruthEvaluation: truthEvaluationGuidance{
			AuthoredTruthLeafOnly: true, ManualRoles: []string{"premise", "counterpoint"},
			EffectiveCalculated: true, EffectivePersisted: false,
			EvaluationVersion: 1, Mode: "grounded", DefeatsIncluded: true,
			EvaluateCommand: "evaluate", NormalizeCommand: "normalize-truth",
		},
		DefeatAuthoring: defeatAuthoringGuidance{
			ChangesEffectiveTruth: true, GroundedAcceptanceApplies: true,
			UndermineWhen:              "accepting the counterpoint would make the premise false or materially out of scope",
			UndercutWhen:               "accepting the counterpoint means the stated sources do not suffice for that target",
			CounterpointWhen:           "the new counterpoint defeats the earlier counterpoint",
			CaveatAutomaticallyDefeats: false,
			NonDefeatExamples:          []string{"bare possibility", "absence of direct proof", "residual uncertainty"},
			NonDefeatAlternatives:      []string{"conversation", "adjacent note", "repair or narrow the exact inference"},
			InspectionCommand:          "evaluate",
		},
		StatementIdentity: statementIdentityGuidance{
			IDRequired: true, IDMeaning: "durable proposition record identity",
			MateriallyDifferentGetsNewID: true, SemanticEquivalenceMechanical: false,
			RolePrefixesCurrent: true, PremisePromotionChangesID: true,
			PremisePromotionMappingFields:   []string{"previous_id", "current_id"},
			PremisePromotionExternalWarning: true,
		},
		TextEdits: textEditGuidance{
			SamePropositionRequired: true, Flag: "--same-proposition", TruthKindRequireFlag: false,
		},
		Slugs: slugGuidance{
			Optional: true, Mutable: true, OldAliasesRetained: false, RelationsUseIDs: true,
			Operations: []string{"--slug", "--from-text", "--clear"}, ExternalScopeWarning: true,
		},
		ScriptedAuthoring: scriptedAuthoringGuidance{
			PredictGeneratedIDs: false, ExplicitIDFlag: "--id", ExplicitIDMustEqualNext: true,
			CustomIDsAuthored: false, NextIDsMetadata: "cludia-next-ids", AddResultIDField: "statement.id",
			SuccessfulMutationResultAuthoritative: true, AtomicBatchCommand: "add-batch",
			BatchInputSchemaVersion: 2, BatchResultMappingField: "statements, derivations, and defeats", BatchDryRunMappingTentative: true,
		},
		Deletion: deletionGuidance{
			DryRunAvailable: true, DryRunFlag: "--dry-run", RemoveAttachedCounterpointsFirst: true,
			CounterpointRemovalCommand: "remove-counterpoint",
		},
		MaterialReplacement: materialReplacementGuidance{
			EditCommandAppropriate: false,
			CurrentPrimitiveSteps:  []string{"add and justify new statement", "dry-run explicit relation choices", "apply the state-bound plan", "retain or explicitly delete old statement"},
			Command:                "replace", TwoPhase: true, DryRunRequired: true, ApplyTokenRequired: true,
			RelationChoicesExplicit: true, OldRetainedByDefault: true, AutomaticRetargetAll: false,
		},
		Renumbering: renumberingGuidance{
			Command: "renumber", OrdinaryAllocationMonotonic: true, DeletedIDsReusedOrdinarily: false,
			SoleNumberingReset: true, TwoPhase: true, ApplyTokenRequired: true,
			CompleteMapping: true, ExternalScopeWarning: true, TUIHotkey: false,
		},
	}
}

func writeGuidanceUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cludia guidance [--json]")
	fmt.Fprintln(w, "Explain evidence capture, leaf truth, derived provability, grounded defeats, and the use-case-neutral identity, allocation, edit, replacement, and renumbering contract.")
}
