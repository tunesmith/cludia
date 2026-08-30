package argument

import "testing"

func TestStatementLifecycleOperationsCloneAndReportSemanticResults(t *testing.T) {
	doc, first, err := InitializeDocument(InitializeOptions{
		DocumentID: "lifecycle", Title: "Lifecycle",
		Statement: StatementInput{Text: "First", Truth: TruthTrue, Kind: KindFact},
	})
	if err != nil || first.ID != "P1" {
		t.Fatalf("initialize = %#v, %v", first, err)
	}
	added, second, err := AddStatement(doc, StatementInput{Text: "Second", Slug: "second", Truth: TruthUnknown, Kind: KindValue})
	if err != nil || second.ID != "P2" || len(doc.Statements) != 1 || len(added.Statements) != 2 {
		t.Fatalf("add = %#v, %v; original=%d next=%d", second, err, len(doc.Statements), len(added.Statements))
	}
	withThird, third, err := AddStatement(added, StatementInput{Text: "Third", Truth: TruthTrue, Kind: KindFact})
	if err != nil || third.ID != "P3" || len(added.Statements) != 2 {
		t.Fatalf("third = %#v, %v", third, err)
	}
	withFourth, fourth, err := AddStatement(withThird, StatementInput{Text: "Fourth", Truth: TruthFalse, Kind: KindValue})
	if err != nil || fourth.ID != "P4" || len(withThird.Statements) != 3 {
		t.Fatalf("fourth = %#v, %v", fourth, err)
	}
	text := "Reworded second"
	truth := TruthFalse
	edited, edit, err := EditStatement(withFourth, EditStatementOptions{Reference: "second", Text: &text, Truth: &truth})
	if err != nil || !edit.Changed || edit.Previous.Text != "Second" || edit.Current.Text != text || edit.Current.Truth != TruthFalse {
		t.Fatalf("edit = %#v, %v", edit, err)
	}
	renamed, slug, err := RenameStatementSlug(edited, RenameSlugOptions{Reference: "P2", Mode: SlugFromText})
	if err != nil || !slug.Changed || slug.PreviousSlug != "second" || slug.CurrentSlug != "reworded-second" {
		t.Fatalf("slug = %#v, %v", slug, err)
	}
	if current, _ := renamed.Statement("P2"); current.Slug != "reworded-second" {
		t.Fatalf("renamed statement = %#v", current)
	}
}

func TestRenameSlugRewritesRecognizedRootOnClone(t *testing.T) {
	doc, _, err := InitializeDocument(InitializeOptions{
		DocumentID: "root", Title: "Root",
		Statement: StatementInput{Text: "First", Slug: "first", Truth: TruthTrue, Kind: KindFact},
	})
	if err != nil {
		t.Fatal(err)
	}
	doc.Metadata = append(doc.Metadata, Metadata{Key: "root", Value: "first"})
	next, result, err := RenameStatementSlug(doc, RenameSlugOptions{Reference: "P1", Mode: SlugClear})
	if err != nil || !result.RootMetadataUpdated {
		t.Fatalf("rename = %#v, %v", result, err)
	}
	if root, _ := next.MetadataValue("root"); root != "P1" {
		t.Fatalf("next root = %q", root)
	}
	if root, _ := doc.MetadataValue("root"); root != "first" {
		t.Fatalf("caller root changed = %q", root)
	}
}
