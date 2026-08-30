package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tunesmith/cludia/internal/argument"
	"github.com/tunesmith/cludia/internal/validation"
)

func TestValidateAndPersistSeparatesDryRunInvalidAndAppliedWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workspace.arg")
	doc := &argument.Document{ID: "workspace", Title: "Workspace", Statements: []argument.Statement{{ID: "P1", Role: argument.RolePremise, Kind: argument.KindFact, Truth: argument.TruthTrue, Text: "One"}}}
	if result, err := ValidateAndPersist(path, doc, validation.ProfileWorkspace, false); err != nil || !result.OK() {
		t.Fatalf("dry run = %#v, %v", result, err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("dry run created file: %v", err)
	}
	if result, err := ValidateAndPersist(path, doc, validation.ProfileWorkspace, true); err != nil || !result.OK() {
		t.Fatalf("persist = %#v, %v", result, err)
	}
	invalid := doc.Clone()
	invalid.Statements[0].Text = ""
	if result, err := ValidateAndPersist(path, invalid, validation.ProfileWorkspace, true); err != nil || result.OK() {
		t.Fatalf("invalid = %#v, %v", result, err)
	}
}
