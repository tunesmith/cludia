package argfile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tunesmith/cludia/internal/argument"
	"github.com/tunesmith/cludia/internal/diagnostic"
)

func TestSaveAtomicWritesParseableDocumentAndPreservesMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "case.arg")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	doc := minimalDocument()
	if err := SaveAtomic(path, doc); err != nil {
		t.Fatalf("SaveAtomic: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("mode = %o, want 600", got)
	}
	parsed := Load(path)
	if diagnostic.HasErrors(parsed.Diagnostics) {
		t.Fatalf("saved file does not parse: %#v", parsed.Diagnostics)
	}
}

func TestSaveAtomicSerializationFailureLeavesExistingFileUntouched(t *testing.T) {
	path := filepath.Join(t.TempDir(), "case.arg")
	const original = "original contents\n"
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	doc := minimalDocument()
	doc.Statements[0].Text = "first line\n\"\"\"\nlast line"
	if err := SaveAtomic(path, doc); err == nil {
		t.Fatal("SaveAtomic succeeded for unencodable text")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != original {
		t.Fatalf("existing file changed after failed save: %q", data)
	}
}

func TestSerializeRejectsMultilineDefeatWithoutDroppingTarget(t *testing.T) {
	doc := minimalDocument()
	doc.Statements = append(doc.Statements, argument.Statement{
		ID: "CP1", Role: argument.RoleCounterpoint, Kind: argument.KindFact,
		Truth: argument.TruthTrue, Text: "first line\nsecond line",
	})
	doc.Defeats = []argument.Defeat{{From: "CP1", Scope: argument.DefeatPremise, To: "P1"}}
	if _, err := Serialize(doc); err == nil {
		t.Fatal("Serialize silently accepted a multiline counterpoint defeat that Concludia syntax cannot represent")
	}
}

func minimalDocument() *argument.Document {
	return &argument.Document{
		ID: "case", Title: "Case",
		Metadata: []argument.Metadata{{Key: "profile", Value: "workspace"}},
		Statements: []argument.Statement{{
			ID: "P1", Slug: "first", Role: argument.RolePremise,
			Kind: argument.KindFact, Truth: argument.TruthTrue, Text: "First",
		}},
	}
}
