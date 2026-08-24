package argfile

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/tunesmith/cludia/internal/argument"
	"github.com/tunesmith/cludia/internal/diagnostic"
)

func TestWorkspaceExampleParsesAndSemanticallyRoundTrips(t *testing.T) {
	parsed := ParseFile(filepath.Join("..", "..", "examples", "broken-window-workspace.arg"))
	if diagnostic.HasErrors(parsed.Diagnostics) {
		t.Fatalf("parse diagnostics: %#v", parsed.Diagnostics)
	}
	if got := len(parsed.Document.Statements); got != 9 {
		t.Fatalf("statements = %d, want 9", got)
	}
	if got := len(parsed.Document.Junctors); got != 2 {
		t.Fatalf("junctors = %d, want 2", got)
	}
	if got := len(parsed.Document.Defeats); got != 4 {
		t.Fatalf("defeats = %d, want 4", got)
	}
	if parsed.Document.Junctors[0].ID != "J1" || parsed.Document.Junctors[1].ID != "J2" {
		t.Fatalf("junctor labels not preserved: %#v", parsed.Document.Junctors)
	}

	serialized, err := Serialize(parsed.Document)
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	roundTrip := Parse(serialized)
	if diagnostic.HasErrors(roundTrip.Diagnostics) {
		t.Fatalf("round-trip diagnostics: %#v\n%s", roundTrip.Diagnostics, serialized)
	}
	if !reflect.DeepEqual(parsed.Document, roundTrip.Document) {
		t.Fatalf("semantic round trip changed document\nbefore: %#v\nafter:  %#v\n%s", parsed.Document, roundTrip.Document, serialized)
	}
}

func TestConcludiaConstructsRoundTrip(t *testing.T) {
	input := `argument compatibility "Compatibility"
meta author="test", version="0.1.0", custom="preserved"

premise[fact] P1:first ::T "First"
premise[fact] P2:second ::F "Second"

lemma[fact] L1:either "Either source"
  <- OR#alternatives(P1, P2)

conclusion[value] C1:result "Result"
  <- AND(L1)

undermine CP1:challenge ::T "P1 is challenged" -> premise P1
rejoinder CP2:reply ::T "Reply" -> counterpoint CP1
undercut CP3:link-challenge ::T "The OR link is challenged" -> inference alternatives:target L1
`
	parsed := Parse(input)
	if diagnostic.HasErrors(parsed.Diagnostics) {
		t.Fatalf("parse diagnostics: %#v", parsed.Diagnostics)
	}
	if parsed.Document.Junctors[0].Connector != argument.ConnectorOR || parsed.Document.Junctors[0].ID != "alternatives" {
		t.Fatalf("OR junctor not preserved: %#v", parsed.Document.Junctors[0])
	}
	if len(parsed.Document.DirectSupports) != 1 || parsed.Document.DirectSupports[0].Connector != argument.ConnectorAND {
		t.Fatalf("direct support not preserved: %#v", parsed.Document.DirectSupports)
	}
	serialized, err := Serialize(parsed.Document)
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	roundTrip := Parse(serialized)
	if diagnostic.HasErrors(roundTrip.Diagnostics) {
		t.Fatalf("round-trip diagnostics: %#v\n%s", roundTrip.Diagnostics, serialized)
	}
	if !reflect.DeepEqual(parsed.Document, roundTrip.Document) {
		t.Fatalf("round trip changed compatibility document\nbefore: %#v\nafter: %#v", parsed.Document, roundTrip.Document)
	}
}

func TestCurrentConcludiaFixtureParsesValidatesAndRoundTrips(t *testing.T) {
	parsed := ParseFile(filepath.Join("testdata", "concludia-auto-backups.arg"))
	if diagnostic.HasErrors(parsed.Diagnostics) {
		t.Fatalf("parse diagnostics: %#v", parsed.Diagnostics)
	}
	serialized, err := Serialize(parsed.Document)
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	roundTrip := Parse(serialized)
	if diagnostic.HasErrors(roundTrip.Diagnostics) {
		t.Fatalf("round-trip diagnostics: %#v", roundTrip.Diagnostics)
	}
	if !reflect.DeepEqual(parsed.Document, roundTrip.Document) {
		t.Fatal("current Concludia fixture changed on semantic round trip")
	}
}

func TestMultilineStatementsAreDedented(t *testing.T) {
	input := "argument multi \"Multiline\"\n\n" +
		"premise[fact] P1:first ::T \"\"\"\n" +
		"    First line\n" +
		"      indented detail\n" +
		"    \"\"\"\n"
	parsed := Parse(input)
	if diagnostic.HasErrors(parsed.Diagnostics) {
		t.Fatalf("parse diagnostics: %#v", parsed.Diagnostics)
	}
	if got, want := parsed.Document.Statements[0].Text, "First line\n  indented detail"; got != want {
		t.Fatalf("text = %q, want %q", got, want)
	}
}

func TestUnknownSyntaxIsDiagnosedInsteadOfDropped(t *testing.T) {
	parsed := Parse("argument bad \"Bad\"\nmeta known=\"yes\", not-valid\npremise P1 ::T \"One\"\nunknown construct\n")
	assertDiagnosticCode(t, parsed.Diagnostics, "metadata_invalid")
	assertDiagnosticCode(t, parsed.Diagnostics, "statement_invalid")
}

func TestMetadataEntriesRequireCommas(t *testing.T) {
	parsed := Parse("argument bad \"Bad\"\nmeta first=\"one\" second=\"two\"\npremise P1 ::T \"One\"\n")
	assertDiagnosticCode(t, parsed.Diagnostics, "metadata_invalid")
}

func TestGeneratedJunctorIDDoesNotCollideWithStatementID(t *testing.T) {
	parsed := Parse("argument ids \"IDs\"\npremise J1 ::T \"Reserved\"\npremise P1 ::T \"One\"\nlemma L1 \"Result\"\n  <- AND(J1, P1)\n")
	if diagnostic.HasErrors(parsed.Diagnostics) {
		t.Fatalf("parse diagnostics: %#v", parsed.Diagnostics)
	}
	if got := parsed.Document.Junctors[0].ID; got != "J2" {
		t.Fatalf("generated junctor id = %q, want J2", got)
	}
}

func assertDiagnosticCode(t *testing.T, diagnostics []diagnostic.Diagnostic, code string) {
	t.Helper()
	for _, item := range diagnostics {
		if item.Code == code {
			return
		}
	}
	t.Fatalf("diagnostic %q not found in %#v", code, diagnostics)
}
