package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/tunesmith/cludia/internal/argfile"
	"github.com/tunesmith/cludia/internal/argument"
)

func TestRenameSlugUpdatesStatementAndRootMetadata(t *testing.T) {
	path := twoPremiseWorkspace(t)
	doc := argfile.ParseFile(path).Document
	doc.Metadata = append(doc.Metadata, metadata("root", "first"))
	if err := argfile.SaveAtomic(path, doc); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err := run([]string{"rename-slug", path, "P1", "--slug", "renamed-first", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("rename-slug: %v\nstderr: %s", err, stderr.String())
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	assertExactKeys(t, raw, "schema_version", "action", "profile", "document", "statement", "previous_slug", "current_slug", "root_metadata_updated", "scope_checked", "changes", "diagnostics")
	var output slugMutationOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if output.PreviousSlug != "first" || output.CurrentSlug != "renamed-first" || !output.RootMetadataUpdated || len(output.ScopeChecked) != 1 || len(output.Diagnostics) != 1 || output.Diagnostics[0].Code != "external_slug_references_unchecked" {
		t.Fatalf("slug output = %#v", output)
	}
	parsed := argfile.ParseFile(path)
	statement, _ := parsed.Document.Statement("P1")
	root, _ := parsed.Document.MetadataValue("root")
	if statement.Slug != "renamed-first" || root != "renamed-first" {
		t.Fatalf("saved slug = %q root = %q", statement.Slug, root)
	}
}

func TestRenameSlugFromTextAndClear(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workspace.arg")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"init", path, "--title", "Slug test", "--text", "A clearer statement name"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"rename-slug", path, "P1", "--from-text", "--json"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	parsed := argfile.ParseFile(path)
	statement, _ := parsed.Document.Statement("P1")
	if statement.Slug != "clearer-statement-name" {
		t.Fatalf("generated slug = %q", statement.Slug)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"rename-slug", path, "P1", "--clear", "--json"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	parsed = argfile.ParseFile(path)
	statement, _ = parsed.Document.Statement("P1")
	if statement.Slug != "" {
		t.Fatalf("slug not cleared: %q", statement.Slug)
	}
}

func TestRenameSlugRejectsDuplicateWithoutWriting(t *testing.T) {
	path := twoPremiseWorkspace(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err = run([]string{"rename-slug", path, "P1", "--slug", "second", "--json"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("duplicate slug accepted")
	}
	after, readErr := os.ReadFile(path)
	if readErr != nil || !bytes.Equal(before, after) {
		t.Fatalf("workspace changed after duplicate slug: %v", readErr)
	}
}

func TestClearSlugUpdatesRootMetadataToID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workspace.arg")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"init", path, "--title", "Title", "--text", "First"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	doc := argfile.ParseFile(path).Document
	doc.Metadata = append(doc.Metadata, metadata("root", "first"))
	if err := argfile.SaveAtomic(path, doc); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"rename-slug", path, "P1", "--clear"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	parsed := argfile.ParseFile(path)
	root, _ := parsed.Document.MetadataValue("root")
	if root != "P1" {
		t.Fatalf("root metadata = %q, want P1", root)
	}
}

func metadata(key, value string) argument.Metadata {
	return argument.Metadata{Key: key, Value: value}
}
