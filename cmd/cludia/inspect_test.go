package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestListIsolatedFindsDisconnectedExampleStatement(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "broken-window-workspace.arg")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"list", path, "--state", "isolated", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("list: %v\nstderr: %s", err, stderr.String())
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatalf("decode list JSON: %v", err)
	}
	assertExactKeys(t, raw, "schema_version", "profile", "state", "statements", "junctors", "direct_supports", "defeats", "diagnostics")
	var output listOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode typed list JSON: %v", err)
	}
	if len(output.Statements) != 1 || output.Statements[0].ID != "P4" || !output.Statements[0].Isolated {
		t.Fatalf("isolated statements = %#v", output.Statements)
	}
	if output.Junctors == nil || output.DirectSupports == nil || output.Defeats == nil {
		t.Fatalf("filtered relation arrays must be empty, not null: %#v", output)
	}
}

func TestShowBySlugIncludesStatementRelations(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "broken-window-workspace.arg")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"show", "--json", path, "no-returning-prints", "--relations"}, &stdout, &stderr); err != nil {
		t.Fatalf("show: %v\nstderr: %s", err, stderr.String())
	}
	var output showOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode show JSON: %v", err)
	}
	if output.ElementType != "statement" || output.Statement == nil || output.Statement.ID != "P2" {
		t.Fatalf("unexpected statement output: %#v", output)
	}
	if output.Relations == nil || len(output.Relations.OutgoingSupport) != 2 || len(output.Relations.DefeatsTargeting) != 1 {
		t.Fatalf("unexpected relations: %#v", output.Relations)
	}
}

func TestShowJunctorIncludesUndercut(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "broken-window-workspace.arg")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"show", path, "J1", "--relations", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("show junctor: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatalf("decode show JSON: %v", err)
	}
	assertExactKeys(t, raw, "schema_version", "profile", "element_type", "junctor", "relations", "diagnostics")
	var output showOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode typed show JSON: %v", err)
	}
	if output.ElementType != "junctor" || output.Junctor == nil || output.Relations == nil || len(output.Relations.DefeatsTargeting) != 1 {
		t.Fatalf("unexpected junctor output: %#v", output)
	}
}
