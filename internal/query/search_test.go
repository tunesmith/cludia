package query

import (
	"testing"

	"github.com/tunesmith/cludia/internal/argument"
)

func TestSearchStatementsMatchesIDSlugAndTextInDocumentOrder(t *testing.T) {
	doc := &argument.Document{Statements: []argument.Statement{
		{ID: "P1", Slug: "alpha-observation", Text: "First statement"},
		{ID: "P2", Slug: "second", Text: "Alpha appears in the text"},
		{ID: "ALPHA3", Slug: "third", Text: "Last statement"},
	}}
	matches := SearchStatements(doc, "ALPHA")
	if len(matches) != 3 {
		t.Fatalf("matches = %#v, want 3", matches)
	}
	if matches[0].Statement.ID != "P1" || len(matches[0].Fields) != 1 || matches[0].Fields[0] != "slug" {
		t.Fatalf("first match = %#v", matches[0])
	}
	if matches[1].Statement.ID != "P2" || len(matches[1].Fields) != 1 || matches[1].Fields[0] != "text" {
		t.Fatalf("second match = %#v", matches[1])
	}
	if matches[2].Statement.ID != "ALPHA3" || len(matches[2].Fields) != 1 || matches[2].Fields[0] != "id" {
		t.Fatalf("third match = %#v", matches[2])
	}
}

func TestSearchStatementsReportsEveryMatchingField(t *testing.T) {
	doc := &argument.Document{Statements: []argument.Statement{{ID: "Alpha", Slug: "alpha", Text: "Alpha"}}}
	matches := SearchStatements(doc, "alpha")
	if len(matches) != 1 || len(matches[0].Fields) != 3 {
		t.Fatalf("matches = %#v", matches)
	}
	want := []string{"id", "slug", "text"}
	for i := range want {
		if matches[0].Fields[i] != want[i] {
			t.Fatalf("fields = %#v, want %#v", matches[0].Fields, want)
		}
	}
}

func TestSearchStatementsRejectsBlankQueryAsNoMatches(t *testing.T) {
	doc := &argument.Document{Statements: []argument.Statement{{ID: "P1", Text: "First"}}}
	if matches := SearchStatements(doc, "  "); len(matches) != 0 || matches == nil {
		t.Fatalf("blank search matches = %#v, want empty non-nil slice", matches)
	}
}
