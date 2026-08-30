package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestGuidanceJSONContractIsUseCaseNeutral(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"guidance", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("guidance: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	assertExactKeys(t, raw, "schema_version", "use_case_neutral", "statement_authoring", "truth_evaluation", "statement_identity", "text_edits", "slugs", "scripted_authoring", "deletion", "material_replacement", "renumbering")
	var output guidanceOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if !output.UseCaseNeutral || !output.StatementIdentity.IDRequired || !output.StatementIdentity.MateriallyDifferentGetsNewID || output.StatementIdentity.SemanticEquivalenceMechanical || !output.StatementIdentity.RolePrefixesCurrent || !output.StatementIdentity.PremisePromotionChangesID || !output.StatementIdentity.PremisePromotionExternalWarning || len(output.StatementIdentity.PremisePromotionMappingFields) != 2 || output.StatementIdentity.PremisePromotionMappingFields[0] != "previous_id" || output.StatementIdentity.PremisePromotionMappingFields[1] != "current_id" {
		t.Fatalf("identity guidance = %#v", output)
	}
	if !output.StatementAuthoring.TruthAptRequired || output.StatementAuthoring.QuestionsSupported || output.StatementAuthoring.QuestionAlternative != "conversation or adjacent notes" || !output.StatementAuthoring.HypothesesAsUnknownPropositions || output.StatementAuthoring.UnknownTruthFlag != "--truth U" || output.StatementAuthoring.DefaultTruth != "T" || output.StatementAuthoring.ConfidenceSupported {
		t.Fatalf("statement authoring guidance = %#v", output.StatementAuthoring)
	}
	if !output.TruthEvaluation.AuthoredTruthLeafOnly || !output.TruthEvaluation.EffectiveCalculated || output.TruthEvaluation.EffectivePersisted || output.TruthEvaluation.EvaluationVersion != 1 || output.TruthEvaluation.Mode != "grounded" || !output.TruthEvaluation.DefeatsIncluded || output.TruthEvaluation.EvaluateCommand != "evaluate" || output.TruthEvaluation.NormalizeCommand != "normalize-truth" {
		t.Fatalf("truth evaluation guidance = %#v", output.TruthEvaluation)
	}
	if !output.TextEdits.SamePropositionRequired || output.TextEdits.Flag != "--same-proposition" || output.TextEdits.TruthKindRequireFlag {
		t.Fatalf("edit guidance = %#v", output.TextEdits)
	}
	if !output.Slugs.Optional || !output.Slugs.Mutable || output.Slugs.OldAliasesRetained || !output.Slugs.RelationsUseIDs {
		t.Fatalf("slug guidance = %#v", output.Slugs)
	}
	if output.ScriptedAuthoring.PredictGeneratedIDs || output.ScriptedAuthoring.ExplicitIDFlag != "--id" || !output.ScriptedAuthoring.ExplicitIDMustEqualNext || output.ScriptedAuthoring.CustomIDsAuthored || output.ScriptedAuthoring.NextIDsMetadata != "cludia-next-ids" || output.ScriptedAuthoring.AddResultIDField != "statement.id" || !output.ScriptedAuthoring.SuccessfulMutationResultAuthoritative || output.ScriptedAuthoring.AtomicBatchCommand != "add-batch" || output.ScriptedAuthoring.BatchInputSchemaVersion != 2 || output.ScriptedAuthoring.BatchResultMappingField != "statements, derivations, and defeats" || !output.ScriptedAuthoring.BatchDryRunMappingTentative {
		t.Fatalf("scripted authoring guidance = %#v", output.ScriptedAuthoring)
	}
	if !output.Deletion.DryRunAvailable || output.Deletion.DryRunFlag != "--dry-run" || !output.Deletion.RemoveAttachedCounterpointsFirst || output.Deletion.CounterpointRemovalCommand != "remove-counterpoint" {
		t.Fatalf("deletion guidance = %#v", output.Deletion)
	}
	if output.MaterialReplacement.EditCommandAppropriate || output.MaterialReplacement.Command != "replace" || !output.MaterialReplacement.TwoPhase || !output.MaterialReplacement.DryRunRequired || !output.MaterialReplacement.ApplyTokenRequired || !output.MaterialReplacement.RelationChoicesExplicit || !output.MaterialReplacement.OldRetainedByDefault || output.MaterialReplacement.AutomaticRetargetAll {
		t.Fatalf("replacement guidance = %#v", output.MaterialReplacement)
	}
	if output.Renumbering.Command != "renumber" || !output.Renumbering.OrdinaryAllocationMonotonic || output.Renumbering.DeletedIDsReusedOrdinarily || !output.Renumbering.SoleNumberingReset || !output.Renumbering.TwoPhase || !output.Renumbering.ApplyTokenRequired || !output.Renumbering.CompleteMapping || !output.Renumbering.ExternalScopeWarning || output.Renumbering.TUIHotkey {
		t.Fatalf("renumbering guidance = %#v", output.Renumbering)
	}
}

func TestGuidanceHumanOutputExplainsTruthAptBoundary(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"guidance"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"truth-apt propositions", "not questions", "--truth U", "does not author confidence scores", "leaf premises and leaf counterpoints", "grounded three-valued", "normalize-truth"} {
		if !bytes.Contains(stdout.Bytes(), []byte(want)) {
			t.Fatalf("guidance missing %q:\n%s", want, stdout.String())
		}
	}
}
