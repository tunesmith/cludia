package argument

import "testing"

func TestGeneratedIDsAndSlugsAreStableAndUnique(t *testing.T) {
	doc := &Document{Statements: []Statement{
		{ID: "P1", Slug: "same-claim", Role: RolePremise},
		{ID: "CP1", Role: RoleCounterpoint},
		{ID: "J2", Role: RolePremise},
	}, Junctors: []Junctor{{ID: "J1"}, {ID: "P2"}}}

	if got := NextStatementID(doc, RolePremise); got != "P3" {
		t.Fatalf("NextStatementID premise = %q, want P3", got)
	}
	if got := NextStatementID(doc, RoleCounterpoint); got != "CP2" {
		t.Fatalf("NextStatementID counterpoint = %q, want CP2", got)
	}
	if got := NextJunctorID(doc); got != "J3" {
		t.Fatalf("NextJunctorID = %q, want J3", got)
	}
	if got := UniqueSlug(doc, "Same claim!"); got != "same-claim-2" {
		t.Fatalf("UniqueSlug = %q, want same-claim-2", got)
	}
}

func TestCloneDoesNotShareSlices(t *testing.T) {
	doc := &Document{
		Metadata:   []Metadata{{Key: "profile", Value: "workspace"}},
		Statements: []Statement{{ID: "P1"}},
		Junctors:   []Junctor{{ID: "J1", Sources: []string{"P1", "P2"}}},
	}
	clone := doc.Clone()
	clone.Metadata[0].Value = "concludia"
	clone.Statements[0].ID = "changed"
	clone.Junctors[0].Sources[0] = "changed"

	if doc.Metadata[0].Value != "workspace" || doc.Statements[0].ID != "P1" || doc.Junctors[0].Sources[0] != "P1" {
		t.Fatal("Clone shares mutable slices with original")
	}
}

func TestSlugifyProducesBoundedConversationalSlug(t *testing.T) {
	text := "Marlow, an LLM agent, completed Cludia's first implementation milestone on 2026-08-24."
	got := Slugify(text)
	want := "marlow-llm-agent-completed-cludia-first-implementation-milestone"
	if got != want {
		t.Fatalf("Slugify = %q, want %q", got, want)
	}
	if len(got) > maxSlugLength {
		t.Fatalf("slug length = %d, max %d", len(got), maxSlugLength)
	}
}

func TestUniqueSlugPrefixesDigitLeadingText(t *testing.T) {
	got := UniqueSlug(&Document{}, "42061 blocks the migration")
	want := "statement-42061-blocks-migration"
	if got != want {
		t.Fatalf("UniqueSlug = %q, want %q", got, want)
	}
	if !ValidSlug(got) {
		t.Fatalf("generated slug %q is invalid", got)
	}
	if len(got) > maxSlugLength {
		t.Fatalf("slug length = %d, max %d", len(got), maxSlugLength)
	}
}
