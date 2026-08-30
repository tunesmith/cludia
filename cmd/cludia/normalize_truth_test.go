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
)

func TestNormalizeTruthPlanApplyAndLeafOnlyEditing(t *testing.T) {
	path := sourcedCounterpointWorkspace(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err = run([]string{"edit", path, "supported", "--truth", "F", "--json"}, &stdout, &stderr)
	if !errors.Is(err, errValidationFailed) {
		t.Fatalf("edit error = %v", err)
	}
	var failure failureOutput
	if err := json.Unmarshal(stdout.Bytes(), &failure); err != nil || len(failure.Diagnostics) != 1 || failure.Diagnostics[0].Code != "truth_assignment_nonleaf" || failure.Diagnostics[0].Element != "CP1" {
		t.Fatalf("edit failure = %#v, %v", failure, err)
	}

	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"normalize-truth", path, "--dry-run", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("dry run: %v\n%s", err, stderr.String())
	}
	var plan normalizeTruthOutput
	if err := json.Unmarshal(stdout.Bytes(), &plan); err != nil || !plan.DryRun || len(plan.Statements) != 1 || plan.Statements[0].ID != "CP1" || plan.PlanToken == "" {
		t.Fatalf("plan = %#v, %v", plan, err)
	}
	afterDryRun, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, afterDryRun) {
		t.Fatalf("dry run changed workspace: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"normalize-truth", path, "--apply-token", plan.PlanToken, "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("apply: %v\n%s", err, stderr.String())
	}
	parsed := argfile.ParseFile(path)
	counterpoint, _ := parsed.Document.Statement("CP1")
	if counterpoint.Truth != argument.TruthUnknown {
		t.Fatalf("normalized counterpoint = %#v", counterpoint)
	}
}

func TestNormalizeTruthRejectsStalePlanWithoutWriting(t *testing.T) {
	path := sourcedCounterpointWorkspace(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"normalize-truth", path, "--dry-run", "--json"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	var plan normalizeTruthOutput
	if err := json.Unmarshal(stdout.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"edit", path, "P1", "--truth", "F", "--json"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	beforeApply, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	err = run([]string{"normalize-truth", path, "--apply-token", plan.PlanToken, "--json"}, &stdout, &stderr)
	if !errors.Is(err, errValidationFailed) {
		t.Fatalf("stale apply error = %v", err)
	}
	var failure failureOutput
	if err := json.Unmarshal(stdout.Bytes(), &failure); err != nil || len(failure.Diagnostics) != 1 || failure.Diagnostics[0].Code != "normalize_truth_plan_stale" {
		t.Fatalf("failure = %#v, %v", failure, err)
	}
	afterApply, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(beforeApply, afterApply) {
		t.Fatalf("stale apply changed workspace: %v", err)
	}
}

func sourcedCounterpointWorkspace(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "workspace.arg")
	doc := &argument.Document{
		ID: "truth", Title: "Truth",
		Metadata: []argument.Metadata{{Key: "profile", Value: "workspace"}, {Key: argument.NextIDsMetadataKey, Value: "v1;P=3;L=1;C=1;CP=2;J=2"}},
		Statements: []argument.Statement{
			{ID: "P1", Role: argument.RolePremise, Kind: argument.KindFact, Truth: argument.TruthTrue, Text: "One"},
			{ID: "P2", Role: argument.RolePremise, Kind: argument.KindFact, Truth: argument.TruthTrue, Text: "Two"},
			{ID: "CP1", Slug: "supported", Role: argument.RoleCounterpoint, Kind: argument.KindFact, Truth: argument.TruthTrue, Text: "Supported"},
		},
		Junctors: []argument.Junctor{{ID: "J1", Connector: argument.ConnectorAND, Sources: []string{"P1", "P2"}, Target: "CP1"}},
	}
	if err := argfile.SaveAtomic(path, doc); err != nil {
		t.Fatal(err)
	}
	return path
}
