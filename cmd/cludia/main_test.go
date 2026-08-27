package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
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

func TestSingleFileArgumentLaunchesTUIAndCommandsTakePrecedence(t *testing.T) {
	original := launchTUI
	t.Cleanup(func() { launchTUI = original })
	called := ""
	launchTUI = func(path string) error {
		called = path
		return nil
	}
	var stdout, stderr bytes.Buffer
	if err := run([]string{"case.arg"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if called != "case.arg" {
		t.Fatalf("TUI path = %q", called)
	}
	called = ""
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"help"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if called != "" || !strings.Contains(stdout.String(), "cludia top") {
		t.Fatalf("help dispatch called=%q output=%q", called, stdout.String())
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
