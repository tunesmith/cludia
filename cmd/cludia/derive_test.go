package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/tunesmith/cludia/internal/argfile"
	"github.com/tunesmith/cludia/internal/argument"
	"github.com/tunesmith/cludia/internal/diagnostic"
	"github.com/tunesmith/cludia/internal/validation"
)

func TestDeriveCreatesLemmaAndANDJunctor(t *testing.T) {
	path := twoPremiseWorkspace(t)
	var stdout, stderr bytes.Buffer
	err := run([]string{
		"derive", path, "--source", "P1", "--target-kind", "value",
		"--target-text", "The two premises jointly establish the target.",
		"--source", "second", "--json",
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("derive: %v\nstderr: %s", err, stderr.String())
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatalf("decode derive JSON: %v", err)
	}
	assertExactKeys(t, raw, "schema_version", "action", "dry_run", "profile", "document", "target", "junctor", "role_changes", "truth_changes", "changes", "diagnostics")
	var output deriveOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode typed derive JSON: %v", err)
	}
	if output.Target.ID != "L1" || output.Target.Role != argument.RoleLemma || output.Target.Kind != argument.KindValue {
		t.Fatalf("target = %#v", output.Target)
	}
	if output.Junctor.ID != "J1" || output.Junctor.Connector != argument.ConnectorAND || output.Junctor.Target != "L1" {
		t.Fatalf("junctor = %#v", output.Junctor)
	}
	if len(output.Junctor.Sources) != 2 || output.Junctor.Sources[0] != "P1" || output.Junctor.Sources[1] != "P2" {
		t.Fatalf("sources = %#v", output.Junctor.Sources)
	}
	if output.RoleChanges == nil || output.TruthChanges == nil || output.Diagnostics == nil {
		t.Fatalf("empty arrays encoded as null: %#v", output)
	}
	parsed := argfile.ParseFile(path)
	if diagnostic.HasErrors(parsed.Diagnostics) || len(parsed.Document.Statements) != 3 || len(parsed.Document.Junctors) != 1 {
		t.Fatalf("saved document = %#v, diagnostics %#v", parsed.Document, parsed.Diagnostics)
	}
}

func TestDerivePromotesExistingPremise(t *testing.T) {
	path := twoPremiseWorkspace(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"add", path, "--text", "Existing target", "--json"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	parsed := argfile.ParseFile(path)
	before := parsed.Document.Statements[2]
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"derive", path, "--source", "P1", "--source", "P2", "--target", before.Slug, "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("derive existing: %v\nstderr: %s", err, stderr.String())
	}
	var output deriveOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	var raw struct {
		RoleChanges []map[string]json.RawMessage `json:"role_changes"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil || len(raw.RoleChanges) != 1 {
		t.Fatalf("decode role change contract: %v, %#v", err, raw.RoleChanges)
	}
	assertExactKeys(t, raw.RoleChanges[0], "previous_id", "current_id", "from", "to")
	if len(output.RoleChanges) != 1 || output.RoleChanges[0].PreviousID != before.ID || output.RoleChanges[0].CurrentID != "L1" || output.RoleChanges[0].From != argument.RolePremise || output.RoleChanges[0].To != argument.RoleLemma {
		t.Fatalf("role changes = %#v", output.RoleChanges)
	}
	if output.Target.ID != "L1" || output.Target.Slug != before.Slug || output.Target.Role != argument.RoleLemma || output.Target.Truth != argument.TruthUnknown {
		t.Fatalf("promoted target = %#v, before %#v", output.Target, before)
	}
	if output.Junctor.Target != "L1" {
		t.Fatalf("junctor target = %q", output.Junctor.Target)
	}
	if len(output.Diagnostics) != 1 || output.Diagnostics[0].Code != "external_id_references_unchecked" {
		t.Fatalf("promotion diagnostics = %#v", output.Diagnostics)
	}
	saved := argfile.ParseFile(path)
	if _, exists := saved.Document.Statement(before.ID); exists {
		t.Fatalf("retired premise ID %s remains", before.ID)
	}
	if promoted, exists := saved.Document.Statement("L1"); !exists || promoted.Role != argument.RoleLemma {
		t.Fatalf("saved promoted target = %#v, exists %t", promoted, exists)
	}
}

func TestDerivePromotionRewritesExistingReferencesAndRootMetadata(t *testing.T) {
	path := twoPremiseWorkspace(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"add", path, "--text", "Candidate", "--slug", "candidate"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"add", path, "--text", "Existing target", "--slug", "existing-target"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{
		"derive", path, "--source", "candidate", "--source", "P1", "--target", "existing-target", "--json",
	}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	parsed := argfile.ParseFile(path)
	parsed.Document.Metadata = append(parsed.Document.Metadata, argument.Metadata{Key: "root", Value: "P3"})
	if err := argfile.SaveAtomic(path, parsed.Document); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	if err := run([]string{
		"derive", path, "--source", "P1", "--source", "P2", "--target", "candidate", "--json",
	}, &stdout, &stderr); err != nil {
		t.Fatalf("promote candidate: %v\n%s", err, stderr.String())
	}
	var output deriveOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if output.Target.ID != "L2" || !hasChange(output.Changes, "updated", "metadata", "root") {
		t.Fatalf("promotion output = %#v", output)
	}
	saved := argfile.ParseFile(path).Document
	existing, exists := saved.Junctor("J1")
	if !exists || existing.Sources[0] != "L2" {
		t.Fatalf("existing reference was not rewritten: %#v, exists %t", existing, exists)
	}
	if root, _ := saved.MetadataValue("root"); root != "L2" {
		t.Fatalf("root metadata = %q", root)
	}
}

func TestDeriveCanCreateClosedConclusion(t *testing.T) {
	path := twoPremiseWorkspace(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{
		"derive", path, "--source", "P1", "--source", "P2",
		"--target-role", "conclusion", "--target-kind", "value",
		"--target-text", "The decision follows.", "--json",
	}, &stdout, &stderr); err != nil {
		t.Fatalf("derive conclusion: %v\nstderr: %s", err, stderr.String())
	}
	var output deriveOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if output.Target.ID != "C1" || output.Target.Role != argument.RoleConclusion || output.Target.Kind != argument.KindValue {
		t.Fatalf("conclusion target = %#v", output.Target)
	}
	parsed := argfile.ParseFile(path)
	if diagnostic.HasErrors(parsed.Diagnostics) {
		t.Fatalf("parse diagnostics: %#v", parsed.Diagnostics)
	}
	if result := validation.Validate(parsed.Document, validation.ProfileConcludia); !result.OK() || len(result.Diagnostics) != 0 {
		t.Fatalf("closed conclusion did not validate cleanly: %#v", result.Diagnostics)
	}
}

func TestFailedDeriveLeavesWorkspaceUnchanged(t *testing.T) {
	path := twoPremiseWorkspace(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err = run([]string{
		"derive", path, "--source", "P1", "--source", "P1",
		"--target-text", "Invalid duplicate-source inference", "--json",
	}, &stdout, &stderr)
	if !errors.Is(err, errValidationFailed) {
		t.Fatalf("derive error = %v, want validation failure", err)
	}
	after, readErr := os.ReadFile(path)
	if readErr != nil || !bytes.Equal(before, after) {
		t.Fatalf("workspace changed after failed derive: %v", readErr)
	}
	var failure failureOutput
	if err := json.Unmarshal(stdout.Bytes(), &failure); err != nil {
		t.Fatalf("decode failure JSON: %v", err)
	}
	found := false
	for _, item := range failure.Diagnostics {
		if item.Code == "junctor_source_duplicate" {
			found = true
		}
	}
	if !found {
		t.Fatalf("duplicate-source diagnostic missing: %#v", failure.Diagnostics)
	}
}

func TestFailedExplicitDeriveDoesNotConsumeTargetOrJunctorIDs(t *testing.T) {
	path := twoPremiseWorkspace(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err = run([]string{
		"derive", path, "--source", "P1", "--source", "P2",
		"--target-text", "Rejected", "--target-id", "L1", "--junctor-id", "J2", "--json",
	}, &stdout, &stderr)
	if !errors.Is(err, errValidationFailed) {
		t.Fatalf("derive error = %v", err)
	}
	var failure failureOutput
	if err := json.Unmarshal(stdout.Bytes(), &failure); err != nil || len(failure.Diagnostics) != 1 || failure.Diagnostics[0].Code != "id_not_next" {
		t.Fatalf("failure = %#v, decode %v", failure, err)
	}
	after, readErr := os.ReadFile(path)
	if readErr != nil || !bytes.Equal(before, after) {
		t.Fatalf("failed explicit derive changed workspace: %v", readErr)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{
		"derive", path, "--source", "P1", "--source", "P2", "--target-text", "Accepted", "--json",
	}, &stdout, &stderr); err != nil {
		t.Fatalf("derive after failure: %v", err)
	}
	var output deriveOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil || output.Target.ID != "L1" || output.Junctor.ID != "J1" {
		t.Fatalf("derive after failure = %#v, decode %v", output, err)
	}
}

func TestCycleCreatingDeriveLeavesWorkspaceUnchanged(t *testing.T) {
	path := twoPremiseWorkspace(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{
		"derive", path, "--source", "P1", "--source", "P2",
		"--target-text", "First lemma",
	}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	err = run([]string{
		"derive", path, "--source", "L1", "--source", "P2",
		"--target", "P1", "--json",
	}, &stdout, &stderr)
	if !errors.Is(err, errValidationFailed) {
		t.Fatalf("cycle derive error = %v, want validation failure", err)
	}
	after, readErr := os.ReadFile(path)
	if readErr != nil || !bytes.Equal(before, after) {
		t.Fatalf("workspace changed after cycle failure: %v", readErr)
	}
}

func twoPremiseWorkspace(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "workspace.arg")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"init", path, "--title", "Derive test", "--text", "First"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"add", path, "--text", "Second"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	return path
}

func hasChange(changes []changeOutput, operation, elementType, id string) bool {
	for _, change := range changes {
		if change.Operation == operation && change.ElementType == elementType && change.ID == id {
			return true
		}
	}
	return false
}
