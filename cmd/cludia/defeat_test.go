package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/tunesmith/cludia/internal/argfile"
	"github.com/tunesmith/cludia/internal/argument"
	"github.com/tunesmith/cludia/internal/diagnostic"
)

func TestAllDefeatScopesCanBeAuthored(t *testing.T) {
	path := repairWorkspace(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"undermine", path, "P1", "--truth", "U", "--kind", "value", "--text", "P1 is disputed", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("undermine: %v\nstderr: %s", err, stderr.String())
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatalf("decode undermine: %v", err)
	}
	assertExactKeys(t, raw, "schema_version", "action", "dry_run", "profile", "document", "counterpoint", "defeat", "changes", "diagnostics")
	var undermine defeatMutationOutput
	if err := json.Unmarshal(stdout.Bytes(), &undermine); err != nil {
		t.Fatalf("decode typed undermine: %v", err)
	}
	if undermine.Counterpoint.ID != "CP1" || undermine.Counterpoint.Kind != argument.KindValue || undermine.Counterpoint.Truth != argument.TruthUnknown || undermine.Defeat.Scope != argument.DefeatPremise || undermine.Defeat.To != "P1" {
		t.Fatalf("undermine output = %#v", undermine)
	}

	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"undercut", path, "J1", "--text", "The sources do not suffice", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("undercut: %v", err)
	}
	var undercut defeatMutationOutput
	if err := json.Unmarshal(stdout.Bytes(), &undercut); err != nil {
		t.Fatal(err)
	}
	if undercut.Counterpoint.ID != "CP2" || undercut.Defeat.Scope != argument.DefeatInference || undercut.Defeat.JunctorID != "J1" || undercut.Defeat.AtTarget != "L1" {
		t.Fatalf("undercut output = %#v", undercut)
	}

	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"counterpoint", path, "CP1", "--text", "The dispute is answered", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("counterpoint: %v", err)
	}
	var reply defeatMutationOutput
	if err := json.Unmarshal(stdout.Bytes(), &reply); err != nil {
		t.Fatal(err)
	}
	if reply.Counterpoint.ID != "CP3" || reply.Defeat.Scope != argument.DefeatCounterpoint || reply.Defeat.To != "CP1" {
		t.Fatalf("counterpoint output = %#v", reply)
	}
	parsed := argfile.ParseFile(path)
	if diagnostic.HasErrors(parsed.Diagnostics) || len(parsed.Document.Defeats) != 3 {
		t.Fatalf("saved defeats = %#v, diagnostics %#v", parsed.Document.Defeats, parsed.Diagnostics)
	}
}

func TestDefeatTargetRolesAreEnforcedWithoutWriting(t *testing.T) {
	path := repairWorkspace(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err = run([]string{"undermine", path, "L1", "--text", "Wrong scope", "--json"}, &stdout, &stderr)
	if !errors.Is(err, errValidationFailed) {
		t.Fatalf("undermine error = %v", err)
	}
	after, readErr := os.ReadFile(path)
	if readErr != nil || !bytes.Equal(before, after) {
		t.Fatalf("workspace changed after role failure: %v", readErr)
	}
}

func TestRemoveCounterpointRequiresLeafFirstAndSupportsDryRun(t *testing.T) {
	path := repairWorkspace(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"undermine", path, "P1", "--text", "Challenge"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"counterpoint", path, "CP1", "--text", "Reply"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	err := run([]string{"remove-counterpoint", path, "CP1", "--json"}, &stdout, &stderr)
	if !errors.Is(err, errValidationFailed) {
		t.Fatalf("parent removal error = %v", err)
	}
	var failure failureOutput
	if err := json.Unmarshal(stdout.Bytes(), &failure); err != nil || len(failure.Diagnostics) != 1 || failure.Diagnostics[0].Code != "counterpoint_has_dependents" {
		t.Fatalf("parent failure = %#v, err %v", failure, err)
	}

	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"remove-counterpoint", path, "CP2", "--dry-run", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	assertExactKeys(t, raw, "schema_version", "action", "dry_run", "profile", "document", "counterpoint", "defeats_removed", "components_before", "components_after", "newly_isolated", "changes", "diagnostics")
	var dryRun counterpointRemovalOutput
	if err := json.Unmarshal(stdout.Bytes(), &dryRun); err != nil || !dryRun.DryRun || dryRun.Counterpoint.ID != "CP2" || len(dryRun.DefeatsRemoved) != 1 {
		t.Fatalf("dry-run output = %#v, err %v", dryRun, err)
	}
	afterDryRun, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, afterDryRun) {
		t.Fatalf("dry-run changed file: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"remove-counterpoint", path, "CP2", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("remove leaf: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"remove-counterpoint", path, "CP1", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("remove parent after leaf: %v", err)
	}
	parsed := argfile.ParseFile(path)
	if len(parsed.Document.Defeats) != 0 {
		t.Fatalf("defeats remain: %#v", parsed.Document.Defeats)
	}
}

func TestRemovingUndercutAllowsJunctorRemoval(t *testing.T) {
	path := repairWorkspace(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"undercut", path, "J1", "--text", "Challenge"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"remove-counterpoint", path, "CP1"}, &stdout, &stderr); err != nil {
		t.Fatalf("remove undercut: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"remove-junctor", path, "J1"}, &stdout, &stderr); err != nil {
		t.Fatalf("remove junctor after undercut: %v", err)
	}
}
