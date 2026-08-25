package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestSearchJSONContractAndDocumentOrder(t *testing.T) {
	path := searchTestWorkspace(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"search", path, "ALPHA", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("search: %v\nstderr: %s", err, stderr.String())
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatalf("decode search JSON: %v", err)
	}
	assertExactKeys(t, raw, "schema_version", "profile", "query", "matches", "diagnostics")
	var output searchOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode typed search JSON: %v", err)
	}
	if output.Query != "ALPHA" || len(output.Matches) != 2 {
		t.Fatalf("search output = %#v", output)
	}
	if output.Matches[0].ID != "P1" || output.Matches[1].ID != "P2" {
		t.Fatalf("match order = %#v", output.Matches)
	}
	if output.Matches[0].MatchedFields == nil || output.Diagnostics == nil {
		t.Fatalf("arrays encoded as null: %#v", output)
	}
}

func TestSearchMatchesStatementIDAndReportsIsolation(t *testing.T) {
	path := searchTestWorkspace(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"search", "--json", path, "p2"}, &stdout, &stderr); err != nil {
		t.Fatalf("search id: %v", err)
	}
	var output searchOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if len(output.Matches) != 1 || output.Matches[0].ID != "P2" || !output.Matches[0].Isolated {
		t.Fatalf("id match = %#v", output.Matches)
	}
}

func TestSearchNoMatchesHasEmptyArrayAndHumanMessage(t *testing.T) {
	path := searchTestWorkspace(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"search", path, "missing", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("search JSON: %v", err)
	}
	var output searchOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if output.Matches == nil || len(output.Matches) != 0 {
		t.Fatalf("matches = %#v", output.Matches)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"search", path, "missing"}, &stdout, &stderr); err != nil {
		t.Fatalf("search human: %v", err)
	}
	if got := stdout.String(); got != "No statements matched \"missing\".\n" {
		t.Fatalf("human no-match output = %q", got)
	}
}

func searchTestWorkspace(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "workspace.arg")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"init", path, "--title", "Search test", "--text", "Alpha appears here"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"add", path, "--text", "Second alpha observation"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	return path
}
