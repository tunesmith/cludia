package argument

import "testing"

func TestNextIDsRoundTripAndAllocatorEnforcesExactNext(t *testing.T) {
	doc := &Document{
		Metadata:   []Metadata{{Key: NextIDsMetadataKey, Value: "v1;P=4;L=2;C=1;CP=3;J=8"}},
		Statements: []Statement{{ID: "P1"}, {ID: "P3"}, {ID: "custom-id"}},
		Junctors:   []Junctor{{ID: "J7"}},
	}
	allocator, err := NewIDAllocator(doc)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := allocator.Statement(RolePremise, "P2"); allocationCode(err) != "id_not_next" {
		t.Fatalf("retired id error = %v", err)
	}
	if _, err := allocator.Statement(RolePremise, "custom-next"); allocationCode(err) != "statement_id_not_canonical" {
		t.Fatalf("custom id error = %v", err)
	}
	if id, err := allocator.Statement(RolePremise, "P4"); err != nil || id != "P4" {
		t.Fatalf("exact next statement = %q, %v", id, err)
	}
	if id, err := allocator.Junctor(""); err != nil || id != "J8" {
		t.Fatalf("generated junctor = %q, %v", id, err)
	}
	allocator.Persist(doc)
	if value, _ := doc.MetadataValue(NextIDsMetadataKey); value != "v1;P=5;L=2;C=1;CP=3;J=9" {
		t.Fatalf("persisted next ids = %q", value)
	}
}

func TestNextIDsBootstrapConservativelyAndPreserveDeletionGap(t *testing.T) {
	doc := &Document{Statements: []Statement{{ID: "P1"}, {ID: "P3"}, {ID: "CP2"}}, Junctors: []Junctor{{ID: "J4"}}}
	_, observed, effective, present, stale, err := InspectNextIDs(doc)
	if err != nil || present || len(stale) != 0 {
		t.Fatalf("inspect legacy = observed %#v effective %#v present %t stale %v err %v", observed, effective, present, stale, err)
	}
	if effective.P != 4 || effective.CP != 3 || effective.J != 5 {
		t.Fatalf("legacy effective next ids = %#v", effective)
	}
	if err := EnsureNextIDs(doc); err != nil {
		t.Fatal(err)
	}
	doc.Statements = doc.Statements[:1]
	doc.Junctors = nil
	allocator, err := NewIDAllocator(doc)
	if err != nil {
		t.Fatal(err)
	}
	if id, err := allocator.Statement(RolePremise, ""); err != nil || id != "P4" {
		t.Fatalf("post-deletion id = %q, %v", id, err)
	}
}

func TestNextIDsDetectMalformedAndStaleMetadata(t *testing.T) {
	malformed := &Document{Metadata: []Metadata{{Key: NextIDsMetadataKey, Value: "P=2"}}}
	if _, _, _, _, _, err := InspectNextIDs(malformed); err == nil {
		t.Fatal("malformed metadata accepted")
	}
	staleDoc := &Document{
		Metadata:   []Metadata{{Key: NextIDsMetadataKey, Value: "v1;P=2;L=1;C=1;CP=1;J=1"}},
		Statements: []Statement{{ID: "P3"}},
	}
	_, _, effective, present, stale, err := InspectNextIDs(staleDoc)
	if err != nil || !present || len(stale) != 1 || stale[0] != "P" || effective.P != 4 {
		t.Fatalf("stale inspect = effective %#v present %t stale %v err %v", effective, present, stale, err)
	}
}

func allocationCode(err error) string {
	if allocationErr, ok := err.(*IDAllocationError); ok {
		return allocationErr.Code
	}
	return ""
}

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
