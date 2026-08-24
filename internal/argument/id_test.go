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
