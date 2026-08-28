package validation

import (
	"path/filepath"
	"testing"

	"github.com/tunesmith/cludia/internal/argfile"
	"github.com/tunesmith/cludia/internal/argument"
	"github.com/tunesmith/cludia/internal/diagnostic"
)

func TestExamplesValidateUnderExpectedProfiles(t *testing.T) {
	workspace := argfile.ParseFile(filepath.Join("..", "..", "examples", "broken-window-workspace.arg"))
	if diagnostic.HasErrors(workspace.Diagnostics) {
		t.Fatalf("workspace parse: %#v", workspace.Diagnostics)
	}
	if result := Validate(workspace.Document, ProfileWorkspace); !result.OK() {
		t.Fatalf("workspace validation: %#v", result.Diagnostics)
	}
	strictWorkspace := Validate(workspace.Document, ProfileConcludia)
	if strictWorkspace.OK() {
		t.Fatal("disconnected workspace unexpectedly passed Concludia profile")
	}
	assertCode(t, strictWorkspace.Diagnostics, "concludia_statement_isolated")

	conclusion := argfile.ParseFile(filepath.Join("..", "..", "examples", "broken-window-conclusion.arg"))
	if diagnostic.HasErrors(conclusion.Diagnostics) {
		t.Fatalf("conclusion parse: %#v", conclusion.Diagnostics)
	}
	if result := Validate(conclusion.Document, ProfileConcludia); !result.OK() {
		t.Fatalf("Concludia validation: %#v", result.Diagnostics)
	}
}

func TestCurrentConcludiaFixtureValidates(t *testing.T) {
	parsed := argfile.ParseFile(filepath.Join("..", "argfile", "testdata", "concludia-auto-backups.arg"))
	if diagnostic.HasErrors(parsed.Diagnostics) {
		t.Fatalf("fixture parse: %#v", parsed.Diagnostics)
	}
	if result := Validate(parsed.Document, ProfileConcludia); !result.OK() {
		t.Fatalf("current Concludia fixture validation: %#v", result.Diagnostics)
	}
}

func TestNextIDMetadataValidation(t *testing.T) {
	doc := &argument.Document{
		ID: "next-ids", Title: "Next IDs",
		Metadata:   []argument.Metadata{{Key: argument.NextIDsMetadataKey, Value: "v1;P=2;L=1;C=1;CP=1;J=1"}},
		Statements: []argument.Statement{statement("P3", argument.RolePremise)},
	}
	result := Validate(doc, ProfileWorkspace)
	if !result.OK() {
		t.Fatalf("stale allocator metadata should warn rather than fail: %#v", result.Diagnostics)
	}
	assertCode(t, result.Diagnostics, "next_ids_stale")

	doc.Metadata[0].Value = "v2;P=4;L=1;C=1;CP=1;J=1"
	result = Validate(doc, ProfileWorkspace)
	if result.OK() {
		t.Fatal("unsupported next-id metadata version validated")
	}
	assertCode(t, result.Diagnostics, "next_ids_invalid")
}

func TestDirectedSupportCycleIsRejected(t *testing.T) {
	doc := &argument.Document{
		ID: "cycle", Title: "Cycle",
		Statements: []argument.Statement{
			statement("P1", argument.RolePremise),
			statement("P2", argument.RolePremise),
			statement("L1", argument.RoleLemma),
		},
		Junctors: []argument.Junctor{
			{ID: "J1", Connector: argument.ConnectorAND, Sources: []string{"P1", "P2"}, Target: "L1"},
			{ID: "J2", Connector: argument.ConnectorAND, Sources: []string{"L1", "P2"}, Target: "P1"},
		},
	}
	result := Validate(doc, ProfileWorkspace)
	if result.OK() {
		t.Fatal("cycle unexpectedly validated")
	}
	assertCode(t, result.Diagnostics, "relation_cycle")
}

func TestSupportedRecursiveCounterpointIsWorkspaceValidButDiagnosedForConcludia(t *testing.T) {
	doc := &argument.Document{
		ID: "supported-counterpoint", Title: "Supported counterpoint",
		Statements: []argument.Statement{
			statement("P1", argument.RolePremise),
			statement("P2", argument.RolePremise),
			statement("P3", argument.RolePremise),
			statement("CP1", argument.RoleCounterpoint),
			statement("CP2", argument.RoleCounterpoint),
		},
		Junctors: []argument.Junctor{{ID: "J1", Connector: argument.ConnectorAND, Sources: []string{"P1", "P2"}, Target: "CP1"}},
		Defeats: []argument.Defeat{
			{From: "CP1", Scope: argument.DefeatPremise, To: "P3"},
			{From: "CP2", Scope: argument.DefeatCounterpoint, To: "CP1"},
		},
	}
	if result := Validate(doc, ProfileWorkspace); !result.OK() {
		t.Fatalf("workspace rejected supported recursive counterpoint: %#v", result.Diagnostics)
	}
	strict := Validate(doc, ProfileConcludia)
	if strict.OK() {
		t.Fatal("current Concludia compatibility edge was not diagnosed")
	}
	assertCode(t, strict.Diagnostics, "concludia_defeat_target_not_leaf")
}

func TestConcludiaProfileAcceptsORDirectSupportAndDefeats(t *testing.T) {
	input := `argument compatibility "Compatibility"
premise P1:first ::T "First"
premise P2:second ::T "Second"
lemma L1:either "Either"
  <- OR#J1(P1, P2)
conclusion C1:done "Done"
  <- AND(L1)
undermine CP1:challenge ::T "Challenge" -> premise P1
counterpoint CP2:reply ::T "Reply" -> counterpoint CP1
undercut CP3:undercut ::T "Undercut" -> inference J1:target L1
`
	parsed := argfile.Parse(input)
	if diagnostic.HasErrors(parsed.Diagnostics) {
		t.Fatalf("parse diagnostics: %#v", parsed.Diagnostics)
	}
	if result := Validate(parsed.Document, ProfileConcludia); !result.OK() {
		t.Fatalf("Concludia compatibility fixture rejected: %#v", result.Diagnostics)
	}
}

func statement(id string, role argument.Role) argument.Statement {
	truth := argument.TruthUnknown
	if role == argument.RolePremise || role == argument.RoleCounterpoint {
		truth = argument.TruthTrue
	}
	return argument.Statement{ID: id, Role: role, Kind: argument.KindFact, Truth: truth, Text: id}
}

func assertCode(t *testing.T, diagnostics []diagnostic.Diagnostic, code string) {
	t.Helper()
	for _, item := range diagnostics {
		if item.Code == code {
			return
		}
	}
	t.Fatalf("diagnostic %q not found in %#v", code, diagnostics)
}
