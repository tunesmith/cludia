// SPDX-FileCopyrightText: 2026 KeenWorks
// SPDX-License-Identifier: GPL-3.0-or-later

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

func TestDeleteIsolatedStatementDryRunAndMutation(t *testing.T) {
	path := twoPremiseWorkspace(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err := run([]string{"delete", path, "P2", "--dry-run", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("delete dry-run: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	assertExactKeys(t, raw, "schema_version", "action", "dry_run", "profile", "document", "statement", "junctors_removed", "direct_supports_removed", "defeats_removed", "components_before", "components_after", "newly_isolated", "changes", "diagnostics")
	var output statementDeletionOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil || !output.DryRun || output.Statement.ID != "P2" {
		t.Fatalf("dry-run output = %#v, err %v", output, err)
	}
	afterDryRun, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, afterDryRun) {
		t.Fatalf("dry-run changed file: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"delete", "--json", path, "P2"}, &stdout, &stderr); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if parsed := argfile.ParseFile(path); len(parsed.Document.Statements) != 1 || parsed.Document.Statements[0].ID != "P1" {
		t.Fatalf("statements after delete = %#v", parsed.Document.Statements)
	}
}

func TestDeleteRequiresAttachedUndercutRemovalFirst(t *testing.T) {
	path := repairWorkspace(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"undercut", path, "J1", "--text", "Challenge"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	err := run([]string{"delete", path, "P1", "--dry-run", "--json"}, &stdout, &stderr)
	if !errors.Is(err, errValidationFailed) {
		t.Fatalf("delete error = %v", err)
	}
	var failure failureOutput
	if err := json.Unmarshal(stdout.Bytes(), &failure); err != nil {
		t.Fatal(err)
	}
	if len(failure.Diagnostics) != 1 || failure.Diagnostics[0].Code != "statement_has_defeats" {
		t.Fatalf("delete diagnostics = %#v", failure.Diagnostics)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"remove-counterpoint", path, "CP1"}, &stdout, &stderr); err != nil {
		t.Fatalf("remove undercut: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"delete", path, "P1", "--dry-run", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("delete dry-run after counterpoint removal: %v", err)
	}
	var output statementDeletionOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if len(output.JunctorsRemoved) != 1 || output.JunctorsRemoved[0].ID != "J1" || len(output.DefeatsRemoved) != 0 || len(output.NewlyIsolated) == 0 {
		t.Fatalf("incident removal plan = %#v", output)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"delete", path, "P1", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("delete: %v", err)
	}
	parsed := argfile.ParseFile(path)
	if len(parsed.Document.Junctors) != 0 || len(parsed.Document.Defeats) != 0 {
		t.Fatalf("incident relations remain: junctors %#v defeats %#v", parsed.Document.Junctors, parsed.Document.Defeats)
	}
}

func TestDeleteRejectsCounterpointAndLastStatement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workspace.arg")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"init", path, "--title", "Title", "--text", "Only"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	err := run([]string{"delete", path, "P1", "--json"}, &stdout, &stderr)
	if !errors.Is(err, errValidationFailed) {
		t.Fatalf("last statement error = %v", err)
	}
	if err := run([]string{"add", path, "--text", "Second"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"undermine", path, "P1", "--text", "Challenge"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	err = run([]string{"delete", path, "CP1", "--json"}, &stdout, &stderr)
	if !errors.Is(err, errValidationFailed) {
		t.Fatalf("counterpoint delete error = %v", err)
	}
	var failure failureOutput
	if err := json.Unmarshal(stdout.Bytes(), &failure); err != nil || len(failure.Diagnostics) != 1 || failure.Diagnostics[0].Code != "use_remove_counterpoint" {
		t.Fatalf("counterpoint failure = %#v, err %v", failure, err)
	}
}
