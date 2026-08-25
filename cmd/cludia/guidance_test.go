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
	assertExactKeys(t, raw, "schema_version", "use_case_neutral", "statement_identity", "text_edits", "slugs", "material_replacement")
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
	if output.MaterialReplacement.EditCommandAppropriate || !output.MaterialReplacement.FutureDryRunRequired || output.MaterialReplacement.AutomaticRetargetAll {
		t.Fatalf("replacement guidance = %#v", output.MaterialReplacement)
	}
}
