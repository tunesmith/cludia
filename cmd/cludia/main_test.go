package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestValidateJSONContractAndProfileDetection(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "broken-window-workspace.arg")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"validate", path, "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("run: %v\nstderr: %s", err, stderr.String())
	}
	var output map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, stdout.String())
	}
	assertExactKeys(t, output, "schema_version", "ok", "profile", "document", "stats", "diagnostics")
	var schemaVersion int
	if err := json.Unmarshal(output["schema_version"], &schemaVersion); err != nil || schemaVersion != 1 {
		t.Fatalf("schema_version = %d, err %v", schemaVersion, err)
	}
	var profile string
	if err := json.Unmarshal(output["profile"], &profile); err != nil || profile != "workspace" {
		t.Fatalf("profile = %q, err %v", profile, err)
	}
	var diagnostics []json.RawMessage
	if err := json.Unmarshal(output["diagnostics"], &diagnostics); err != nil || diagnostics == nil || len(diagnostics) != 0 {
		t.Fatalf("diagnostics must be an empty array, got %s, err %v", output["diagnostics"], err)
	}
}

func TestValidateInvalidProfileReturnsStructuredFailure(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "broken-window-workspace.arg")
	var stdout, stderr bytes.Buffer
	err := run([]string{"check", "--json", path, "--profile", "other"}, &stdout, &stderr)
	if !errors.Is(err, errValidationFailed) {
		t.Fatalf("error = %v, want validation failure", err)
	}
	var output validateOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if output.OK || len(output.Diagnostics) != 1 || output.Diagnostics[0].Code != "profile_unknown" {
		t.Fatalf("unexpected output: %#v", output)
	}
}

func assertExactKeys(t *testing.T, object map[string]json.RawMessage, want ...string) {
	t.Helper()
	got := make([]string, 0, len(object))
	for key := range object {
		got = append(got, key)
	}
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("JSON keys = %v, want %v", got, want)
	}
}
