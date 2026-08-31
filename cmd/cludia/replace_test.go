// SPDX-FileCopyrightText: 2026 KeenWorks
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/tunesmith/cludia/internal/argfile"
	"github.com/tunesmith/cludia/internal/argument"
)

func TestReplaceDryRunAndStateBoundApply(t *testing.T) {
	path := replacementWorkspace(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	choices := []string{"replace", path, "L1", "--with", "L2", "--retarget-source", "J3", "--remove-justification", "J1", "--delete-old"}
	var stdout, stderr bytes.Buffer
	dryArgs := append(append([]string(nil), choices...), "--dry-run", "--json")
	if err := run(dryArgs, &stdout, &stderr); err != nil {
		t.Fatalf("replace dry-run: %v\nstderr: %s", err, stderr.String())
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	assertExactKeys(t, raw,
		"schema_version", "action", "dry_run", "applicable", "profile", "document",
		"old_statement", "replacement_statement", "source_retargets", "justifications_removed",
		"affected_defeats", "incidents_before", "incidents_remaining", "root_retarget_requested",
		"root_will_be_retargeted", "root_retargeted", "delete_old_requested",
		"old_statement_will_be_deleted", "old_statement_deleted", "blockers", "components_before",
		"components_after", "newly_isolated", "plan_token", "changes", "diagnostics")
	var plan replacementOutput
	if err := json.Unmarshal(stdout.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	if !plan.DryRun || !plan.Applicable || plan.PlanToken == "" || !plan.OldWillBeDeleted || plan.OldStatementDeleted {
		t.Fatalf("replacement plan = %#v", plan)
	}
	if len(plan.SourceRetargets) != 1 || len(plan.JustificationsRemoved) != 1 || len(plan.Blockers) != 0 {
		t.Fatalf("replacement plan relations = %#v", plan)
	}
	if len(plan.Diagnostics) != 1 || plan.Diagnostics[0].Code != "external_statement_references_unchecked" || !strings.Contains(plan.Diagnostics[0].Message, "L1 or slug \"old-conclusion\"") || !strings.Contains(plan.Diagnostics[0].Message, "other workspaces") || !strings.Contains(plan.Diagnostics[0].Message, "no action is needed") {
		t.Fatalf("external-reference diagnostic = %#v", plan.Diagnostics)
	}
	if got := plan.IncidentsBefore.SourceJunctors; len(got) != 1 || got[0] != "J3" {
		t.Fatalf("source incidents = %#v", got)
	}
	if got := plan.IncidentsBefore.TargetJunctors; len(got) != 1 || got[0] != "J1" {
		t.Fatalf("target incidents = %#v", got)
	}
	afterDryRun, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, afterDryRun) {
		t.Fatalf("dry-run changed workspace: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	applyArgs := append(append([]string(nil), choices...), "--apply-token", plan.PlanToken, "--json")
	if err := run(applyArgs, &stdout, &stderr); err != nil {
		t.Fatalf("replace apply: %v\nstderr: %s", err, stderr.String())
	}
	var applied replacementOutput
	if err := json.Unmarshal(stdout.Bytes(), &applied); err != nil {
		t.Fatal(err)
	}
	if applied.DryRun || !applied.Applicable || !applied.OldStatementDeleted || applied.PlanToken != plan.PlanToken {
		t.Fatalf("applied replacement = %#v", applied)
	}
	parsed := argfile.ParseFile(path)
	if _, exists := parsed.Document.Statement("L1"); exists {
		t.Fatal("old statement L1 remains")
	}
	if _, exists := parsed.Document.Junctor("J1"); exists {
		t.Fatal("old justification J1 remains")
	}
	junctor, exists := parsed.Document.Junctor("J3")
	if !exists || len(junctor.Sources) != 2 || junctor.Sources[0] != "L2" || junctor.Sources[1] != "P5" {
		t.Fatalf("retargeted junctor = %#v", junctor)
	}
}

func TestReplaceDeleteBlockerReturnsCompletePlanWithoutWriting(t *testing.T) {
	path := replacementWorkspace(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err = run([]string{
		"replace", path, "L1", "--with", "L2", "--retarget-source", "J3", "--delete-old", "--dry-run", "--json",
	}, &stdout, &stderr)
	if !errors.Is(err, errValidationFailed) {
		t.Fatalf("replace blocker error = %v", err)
	}
	var plan replacementOutput
	if err := json.Unmarshal(stdout.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	if plan.Applicable || plan.PlanToken != "" || len(plan.Blockers) != 1 || plan.Blockers[0].Relation != "junctor_target" || plan.Blockers[0].ID != "J1" {
		t.Fatalf("blocked plan = %#v", plan)
	}
	after, readErr := os.ReadFile(path)
	if readErr != nil || !bytes.Equal(before, after) {
		t.Fatalf("blocked plan changed workspace: %v", readErr)
	}
}

func TestReplaceStalePlanRefusesAfterWorkspaceChange(t *testing.T) {
	path := replacementWorkspace(t)
	choices := []string{"replace", path, "L1", "--with", "L2", "--retarget-source", "J3", "--remove-justification", "J1", "--delete-old"}
	var stdout, stderr bytes.Buffer
	dryArgs := append(append([]string(nil), choices...), "--dry-run", "--json")
	if err := run(dryArgs, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	var plan replacementOutput
	if err := json.Unmarshal(stdout.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"add", path, "--id", "P6", "--text", "Workspace changed after planning"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	beforeApply, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	applyArgs := append(append([]string(nil), choices...), "--apply-token", plan.PlanToken, "--json")
	err = run(applyArgs, &stdout, &stderr)
	if !errors.Is(err, errValidationFailed) {
		t.Fatalf("stale apply error = %v", err)
	}
	var failure failureOutput
	if err := json.Unmarshal(stdout.Bytes(), &failure); err != nil {
		t.Fatal(err)
	}
	if len(failure.Diagnostics) != 1 || failure.Diagnostics[0].Code != "replacement_plan_stale" {
		t.Fatalf("stale diagnostics = %#v", failure.Diagnostics)
	}
	afterApply, readErr := os.ReadFile(path)
	if readErr != nil || !bytes.Equal(beforeApply, afterApply) {
		t.Fatalf("stale apply changed workspace: %v", readErr)
	}
}

func TestReplaceRootRequiresExplicitRetarget(t *testing.T) {
	path := replacementWorkspace(t)
	parsed := argfile.ParseFile(path)
	parsed.Document.Metadata = append(parsed.Document.Metadata, argument.Metadata{Key: "root", Value: "old-conclusion"})
	if err := argfile.SaveAtomic(path, parsed.Document); err != nil {
		t.Fatal(err)
	}
	choices := []string{"replace", path, "L1", "--with", "L2", "--retarget-source", "J3", "--remove-justification", "J1", "--delete-old"}
	var stdout, stderr bytes.Buffer
	dryArgs := append(append([]string(nil), choices...), "--dry-run", "--json")
	err := run(dryArgs, &stdout, &stderr)
	if !errors.Is(err, errValidationFailed) {
		t.Fatalf("root blocker error = %v", err)
	}
	var blocked replacementOutput
	if err := json.Unmarshal(stdout.Bytes(), &blocked); err != nil {
		t.Fatal(err)
	}
	if len(blocked.Blockers) != 1 || blocked.Blockers[0].Relation != "metadata_root" {
		t.Fatalf("root blockers = %#v", blocked.Blockers)
	}
	stdout.Reset()
	stderr.Reset()
	rootChoices := append(append([]string(nil), choices...), "--retarget-root")
	rootDryArgs := append(append([]string(nil), rootChoices...), "--dry-run", "--json")
	if err := run(rootDryArgs, &stdout, &stderr); err != nil {
		t.Fatalf("root retarget plan: %v", err)
	}
	var plan replacementOutput
	if err := json.Unmarshal(stdout.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	if !plan.RootRetargetRequested || !plan.RootWillBeRetargeted || plan.RootRetargeted || !plan.Applicable || plan.PlanToken == "" {
		t.Fatalf("root plan = %#v", plan)
	}
	stdout.Reset()
	stderr.Reset()
	applyArgs := append(append([]string(nil), rootChoices...), "--apply-token", plan.PlanToken, "--json")
	if err := run(applyArgs, &stdout, &stderr); err != nil {
		t.Fatalf("root retarget apply: %v", err)
	}
	updated := argfile.ParseFile(path).Document
	var applied replacementOutput
	if err := json.Unmarshal(stdout.Bytes(), &applied); err != nil {
		t.Fatal(err)
	}
	if !applied.RootRetargeted || !applied.OldStatementDeleted {
		t.Fatalf("applied root replacement = %#v", applied)
	}
	if root, _ := updated.MetadataValue("root"); root != "new-conclusion" {
		t.Fatalf("root metadata = %q", root)
	}
}

func TestReplaceReportsUndercutOnRetargetedJunctor(t *testing.T) {
	path := replacementWorkspace(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"undercut", path, "J3", "--text", "The downstream inference remains questionable"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{
		"replace", path, "L1", "--with", "L2", "--retarget-source", "J3", "--remove-justification", "J1", "--delete-old", "--dry-run", "--json",
	}, &stdout, &stderr); err != nil {
		t.Fatalf("replace plan with undercut: %v", err)
	}
	var plan replacementOutput
	if err := json.Unmarshal(stdout.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	if len(plan.AffectedDefeats) != 1 || plan.AffectedDefeats[0].JunctorID != "J3" {
		t.Fatalf("affected defeats = %#v", plan.AffectedDefeats)
	}
}

func TestReplaceDeleteBlockedByLegacyDirectSupport(t *testing.T) {
	path := replacementWorkspace(t)
	parsed := argfile.ParseFile(path)
	parsed.Document.DirectSupports = append(parsed.Document.DirectSupports, argument.DirectSupport{Source: "L1", Target: "L3", Connector: argument.ConnectorAND})
	if err := argfile.SaveAtomic(path, parsed.Document); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err := run([]string{
		"replace", path, "L1", "--with", "L2", "--retarget-source", "J3", "--remove-justification", "J1", "--delete-old", "--dry-run", "--json",
	}, &stdout, &stderr)
	if !errors.Is(err, errValidationFailed) {
		t.Fatalf("direct-support blocker error = %v", err)
	}
	var plan replacementOutput
	if err := json.Unmarshal(stdout.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	if len(plan.Blockers) != 1 || plan.Blockers[0].Relation != "direct_support" || len(plan.IncidentsRemaining.DirectSupports) != 1 {
		t.Fatalf("direct-support blockers = %#v", plan)
	}
}

func TestReplaceDeleteBlockedByAttachedCounterpoint(t *testing.T) {
	path := twoPremiseWorkspace(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"add", path, "--id", "P3", "--text", "Replacement premise"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"derive", path, "--source", "P1", "--source", "P2", "--target-text", "Derived", "--target-id", "L1", "--junctor-id", "J1"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"undermine", path, "P1", "--text", "The old premise is challenged"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	err := run([]string{"replace", path, "P1", "--with", "P3", "--retarget-source", "J1", "--delete-old", "--dry-run", "--json"}, &stdout, &stderr)
	if !errors.Is(err, errValidationFailed) {
		t.Fatalf("counterpoint blocker error = %v", err)
	}
	var plan replacementOutput
	if err := json.Unmarshal(stdout.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	if len(plan.Blockers) != 1 || plan.Blockers[0].Relation != "defeat" || len(plan.IncidentsRemaining.Defeats) != 1 {
		t.Fatalf("counterpoint blockers = %#v", plan)
	}
}

func replacementWorkspace(t *testing.T) string {
	t.Helper()
	path := twoPremiseWorkspace(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{
		"derive", path, "--source", "P1", "--source", "P2", "--target-text", "Old conclusion",
		"--target-id", "L1", "--target-slug", "old-conclusion", "--junctor-id", "J1",
	}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		id   string
		text string
	}{
		{id: "P3", text: "New evidence one"},
		{id: "P4", text: "New evidence two"},
		{id: "P5", text: "Shared downstream premise"},
	} {
		stdout.Reset()
		stderr.Reset()
		if err := run([]string{"add", path, "--id", item.id, "--text", item.text}, &stdout, &stderr); err != nil {
			t.Fatal(err)
		}
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{
		"derive", path, "--source", "P3", "--source", "P4", "--target-text", "New conclusion",
		"--target-id", "L2", "--target-slug", "new-conclusion", "--junctor-id", "J2",
	}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{
		"derive", path, "--source", "L1", "--source", "P5", "--target-text", "Downstream conclusion",
		"--target-id", "L3", "--target-slug", "downstream-conclusion", "--junctor-id", "J3",
	}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	return path
}
