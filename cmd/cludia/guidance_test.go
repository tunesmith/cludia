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
	assertExactKeys(t, raw, "schema_version", "use_case_neutral", "statement_identity", "text_edits", "slugs", "scripted_authoring", "deletion", "material_replacement", "renumbering")
	var output guidanceOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if !output.UseCaseNeutral || !output.StatementIdentity.IDRequired || !output.StatementIdentity.MateriallyDifferentGetsNewID || output.StatementIdentity.SemanticEquivalenceMechanical {
		t.Fatalf("identity guidance = %#v", output)
	}
	if !output.TextEdits.SamePropositionRequired || output.TextEdits.Flag != "--same-proposition" || output.TextEdits.TruthKindRequireFlag {
		t.Fatalf("edit guidance = %#v", output.TextEdits)
	}
	if !output.Slugs.Optional || !output.Slugs.Mutable || output.Slugs.OldAliasesRetained || !output.Slugs.RelationsUseIDs {
		t.Fatalf("slug guidance = %#v", output.Slugs)
	}
	if output.ScriptedAuthoring.PredictGeneratedIDs || output.ScriptedAuthoring.ExplicitIDFlag != "--id" || !output.ScriptedAuthoring.ExplicitIDMustEqualNext || output.ScriptedAuthoring.CustomIDsAuthored || output.ScriptedAuthoring.NextIDsMetadata != "cludia-next-ids" || output.ScriptedAuthoring.AddResultIDField != "statement.id" || !output.ScriptedAuthoring.SuccessfulMutationResultAuthoritative || output.ScriptedAuthoring.AtomicBatchCommand != "add-batch" || output.ScriptedAuthoring.BatchInputSchemaVersion != 1 || output.ScriptedAuthoring.BatchResultMappingField != "statements" || !output.ScriptedAuthoring.BatchDryRunMappingTentative {
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
