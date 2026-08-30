package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/tunesmith/cludia/internal/argfile"
)

func TestAddSourceUpdatesANDJunctor(t *testing.T) {
	path := repairWorkspace(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"add-source", path, "J1", "--source", "P3", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("add-source: %v\nstderr: %s", err, stderr.String())
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	assertExactKeys(t, raw, "schema_version", "action", "dry_run", "profile", "document", "junctor", "previous_junctor", "source_added", "changes", "diagnostics")
	var output junctorMutationOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode typed JSON: %v", err)
	}
	if output.SourceAdded != "P3" || len(output.PreviousJunctor.Sources) != 2 || output.Junctor == nil || len(output.Junctor.Sources) != 3 {
		t.Fatalf("add-source output = %#v", output)
	}
	parsed := argfile.ParseFile(path)
	if len(parsed.Document.Junctors[0].Sources) != 3 {
		t.Fatalf("saved junctor = %#v", parsed.Document.Junctors[0])
	}
}

func TestAddSourceWarnsOnlyWhenCrossingPreferredSourceCount(t *testing.T) {
	path := repairWorkspace(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"add-source", path, "J1", "--source", "P3", "--json"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	var output junctorMutationOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil || len(output.Diagnostics) != 0 {
		t.Fatalf("three-source output = %#v, err %v", output, err)
	}
	for _, text := range []string{"Fourth", "Fifth"} {
		stdout.Reset()
		stderr.Reset()
		if err := run([]string{"add", path, "--text", text}, &stdout, &stderr); err != nil {
			t.Fatal(err)
		}
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"add-source", path, "J1", "--source", "P4", "--json"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil || len(output.Diagnostics) != 1 || output.Diagnostics[0].Code != "concludia_junctor_sources_many" || output.Diagnostics[0].Element != "J1" {
		t.Fatalf("threshold output = %#v, err %v", output, err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"add-source", path, "J1", "--source", "P5", "--json"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil || len(output.Diagnostics) != 0 {
		t.Fatalf("post-threshold output = %#v, err %v", output, err)
	}
}

func TestAddSourceCycleFailureLeavesWorkspaceUnchanged(t *testing.T) {
	path := repairWorkspace(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err = run([]string{"add-source", path, "J1", "--source", "L1", "--json"}, &stdout, &stderr)
	if !errors.Is(err, errValidationFailed) {
		t.Fatalf("add-source error = %v", err)
	}
	after, readErr := os.ReadFile(path)
	if readErr != nil || !bytes.Equal(before, after) {
		t.Fatalf("workspace changed after failed add-source: %v", readErr)
	}
}

func TestRemoveSourceDryRunAndMutation(t *testing.T) {
	path := repairWorkspace(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"add-source", path, "J1", "--source", "P3"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"remove-source", "--json", path, "J1", "--source", "P2", "--dry-run"}, &stdout, &stderr); err != nil {
		t.Fatalf("remove-source dry-run: %v", err)
	}
	var dryRun junctorMutationOutput
	if err := json.Unmarshal(stdout.Bytes(), &dryRun); err != nil {
		t.Fatalf("decode dry-run: %v", err)
	}
	if !dryRun.DryRun || dryRun.SourceRemoved != "P2" || dryRun.Junctor == nil || len(dryRun.Junctor.Sources) != 2 {
		t.Fatalf("dry-run output = %#v", dryRun)
	}
	afterDryRun, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, afterDryRun) {
		t.Fatalf("dry-run changed file: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"remove-source", path, "J1", "--source", "P2", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("remove-source: %v", err)
	}
	parsed := argfile.ParseFile(path)
	if got := parsed.Document.Junctors[0].Sources; len(got) != 2 || got[0] != "P1" || got[1] != "P3" {
		t.Fatalf("sources after removal = %#v", got)
	}
}

func TestRemoveSourceRejectsSingleSourceRemainder(t *testing.T) {
	path := repairWorkspace(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err = run([]string{"remove-source", path, "J1", "--source", "P2", "--json"}, &stdout, &stderr)
	if !errors.Is(err, errValidationFailed) {
		t.Fatalf("remove-source error = %v", err)
	}
	after, readErr := os.ReadFile(path)
	if readErr != nil || !bytes.Equal(before, after) {
		t.Fatalf("workspace changed after invalid removal: %v", readErr)
	}
}

func TestReplaceSourceDryRunAndMutation(t *testing.T) {
	path := repairWorkspace(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err := run([]string{"replace-source", "--json", path, "J1", "--from", "P2", "--to", "P3", "--dry-run"}, &stdout, &stderr); err != nil {
		t.Fatalf("replace-source dry-run: %v\nstderr: %s", err, stderr.String())
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	assertExactKeys(t, raw, "schema_version", "action", "dry_run", "profile", "document", "junctor", "previous_junctor", "source_added", "source_removed", "changes", "diagnostics")
	var dryRun junctorMutationOutput
	if err := json.Unmarshal(stdout.Bytes(), &dryRun); err != nil {
		t.Fatal(err)
	}
	if !dryRun.DryRun || dryRun.SourceRemoved != "P2" || dryRun.SourceAdded != "P3" || dryRun.Junctor == nil {
		t.Fatalf("dry-run output = %#v", dryRun)
	}
	if got := dryRun.Junctor.Sources; len(got) != 2 || got[0] != "P1" || got[1] != "P3" {
		t.Fatalf("dry-run sources = %#v", got)
	}
	afterDryRun, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, afterDryRun) {
		t.Fatalf("dry-run changed file: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"replace-source", path, "J1", "--from", "P2", "--to", "P3", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("replace-source: %v\nstderr: %s", err, stderr.String())
	}
	parsed := argfile.ParseFile(path)
	if got := parsed.Document.Junctors[0].Sources; len(got) != 2 || got[0] != "P1" || got[1] != "P3" {
		t.Fatalf("saved sources = %#v", got)
	}
}

func TestReplaceSourceFailuresLeaveWorkspaceUnchanged(t *testing.T) {
	tests := []struct {
		name string
		from string
		to   string
		code string
	}{
		{name: "from is not a source", from: "P3", to: "P1", code: "junctor_source_not_found"},
		{name: "replacement already present", from: "P2", to: "P1", code: "junctor_source_duplicate"},
		{name: "replacement is unchanged", from: "P2", to: "P2", code: "source_replacement_same_statement"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := repairWorkspace(t)
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			err = run([]string{"replace-source", path, "J1", "--from", tt.from, "--to", tt.to, "--json"}, &stdout, &stderr)
			if !errors.Is(err, errValidationFailed) {
				t.Fatalf("replace-source error = %v", err)
			}
			var failure failureOutput
			if err := json.Unmarshal(stdout.Bytes(), &failure); err != nil {
				t.Fatal(err)
			}
			if len(failure.Diagnostics) != 1 || failure.Diagnostics[0].Code != tt.code {
				t.Fatalf("diagnostics = %#v", failure.Diagnostics)
			}
			after, readErr := os.ReadFile(path)
			if readErr != nil || !bytes.Equal(before, after) {
				t.Fatalf("workspace changed after failure: %v", readErr)
			}
		})
	}
}

func TestReplaceSourceCycleFailureLeavesWorkspaceUnchanged(t *testing.T) {
	path := repairWorkspace(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err = run([]string{"replace-source", path, "J1", "--from", "P2", "--to", "L1", "--json"}, &stdout, &stderr)
	if !errors.Is(err, errValidationFailed) {
		t.Fatalf("replace-source error = %v", err)
	}
	after, readErr := os.ReadFile(path)
	if readErr != nil || !bytes.Equal(before, after) {
		t.Fatalf("workspace changed after cycle failure: %v", readErr)
	}
}

func TestReplaceSourceRefusesORJunctorWithoutWriting(t *testing.T) {
	path := repairWorkspace(t)
	parsed := argfile.ParseFile(path)
	parsed.Document.Junctors[0].Connector = "OR"
	if err := argfile.SaveAtomic(path, parsed.Document); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err = run([]string{"replace-source", path, "J1", "--from", "P2", "--to", "P3", "--json"}, &stdout, &stderr)
	if !errors.Is(err, errValidationFailed) {
		t.Fatalf("replace-source error = %v", err)
	}
	var failure failureOutput
	if err := json.Unmarshal(stdout.Bytes(), &failure); err != nil {
		t.Fatal(err)
	}
	if len(failure.Diagnostics) != 1 || failure.Diagnostics[0].Code != "junctor_not_editable" {
		t.Fatalf("diagnostics = %#v", failure.Diagnostics)
	}
	after, readErr := os.ReadFile(path)
	if readErr != nil || !bytes.Equal(before, after) {
		t.Fatalf("OR workspace changed after refusal: %v", readErr)
	}
}

func TestRemoveJunctorDryRunAndMutation(t *testing.T) {
	path := repairWorkspace(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"remove-junctor", path, "J1", "--dry-run", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("remove-junctor dry-run: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	assertExactKeys(t, raw, "schema_version", "action", "dry_run", "profile", "document", "previous_junctor", "changes", "diagnostics")
	var output junctorMutationOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil || !output.DryRun || output.PreviousJunctor.ID != "J1" {
		t.Fatalf("dry-run output = %#v, err %v", output, err)
	}
	if parsed := argfile.ParseFile(path); len(parsed.Document.Junctors) != 1 {
		t.Fatal("dry-run removed junctor")
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"remove-junctor", "--json", path, "J1"}, &stdout, &stderr); err != nil {
		t.Fatalf("remove-junctor: %v", err)
	}
	if parsed := argfile.ParseFile(path); len(parsed.Document.Junctors) != 0 {
		t.Fatalf("junctor remains: %#v", parsed.Document.Junctors)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{
		"derive", path, "--source", "P1", "--source", "P2", "--target", "L1", "--json",
	}, &stdout, &stderr); err != nil {
		t.Fatalf("derive after junctor deletion: %v", err)
	}
	var derived deriveOutput
	if err := json.Unmarshal(stdout.Bytes(), &derived); err != nil || derived.Junctor.ID != "J2" {
		t.Fatalf("post-deletion junctor = %#v, decode %v", derived.Junctor, err)
	}
}

func TestRemoveJunctorRefusesDanglingUndercut(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "..", "examples", "broken-window-workspace.arg"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "undercut.arg")
	if err := os.WriteFile(path, source, 0o600); err != nil {
		t.Fatal(err)
	}
	before := append([]byte(nil), source...)
	var stdout, stderr bytes.Buffer
	err = run([]string{"remove-junctor", path, "J1", "--json"}, &stdout, &stderr)
	if !errors.Is(err, errValidationFailed) {
		t.Fatalf("remove-junctor error = %v", err)
	}
	var failure failureOutput
	if err := json.Unmarshal(stdout.Bytes(), &failure); err != nil {
		t.Fatalf("decode failure: %v", err)
	}
	if len(failure.Diagnostics) != 1 || failure.Diagnostics[0].Code != "junctor_has_undercuts" {
		t.Fatalf("diagnostics = %#v", failure.Diagnostics)
	}
	after, readErr := os.ReadFile(path)
	if readErr != nil || !bytes.Equal(before, after) {
		t.Fatalf("workspace changed after undercut refusal: %v", readErr)
	}
}

func repairWorkspace(t *testing.T) string {
	t.Helper()
	path := twoPremiseWorkspace(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"add", path, "--text", "Third"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{
		"derive", path, "--source", "P1", "--source", "P2",
		"--target-text", "Combined",
	}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	return path
}
