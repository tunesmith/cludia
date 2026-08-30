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
	batch, statements, err := AddStatements(added, []StatementInput{
		{Text: "Third", Truth: TruthTrue, Kind: KindFact},
		{Text: "Fourth", Truth: TruthFalse, Kind: KindValue},
	})
	if err != nil || len(statements) != 2 || statements[0].ID != "P3" || statements[1].ID != "P4" || len(added.Statements) != 2 {
		t.Fatalf("batch = %#v, %v", statements, err)
	}
	text := "Reworded second"
	truth := TruthFalse
	edited, edit, err := EditStatement(batch, EditStatementOptions{Reference: "second", Text: &text, Truth: &truth})
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

func TestBatchStatementFailureIsAtomicAndIdentifiesInput(t *testing.T) {
	doc, _, err := InitializeDocument(InitializeOptions{
		DocumentID: "batch", Title: "Batch",
		Statement: StatementInput{Text: "First", Slug: "first", Truth: TruthTrue, Kind: KindFact},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = AddStatements(doc, []StatementInput{
		{Text: "Valid", Slug: "valid", Truth: TruthTrue, Kind: KindFact},
		{Text: "Duplicate", Slug: "first", Truth: TruthTrue, Kind: KindFact},
	})
	batchErr, ok := err.(*BatchStatementError)
	if !ok || batchErr.Index != 1 {
		t.Fatalf("batch error = %#v", err)
	}
	if mutationErr, ok := batchErr.Err.(*MutationError); !ok || mutationErr.Code != "statement_slug_duplicate" {
		t.Fatalf("nested error = %#v", batchErr.Err)
	}
	if len(doc.Statements) != 1 {
		t.Fatal("failed batch mutated caller")
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
