// SPDX-FileCopyrightText: 2026 KeenWorks
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tunesmith/cludia/internal/argfile"
)

func TestLegacyWorkspaceProfileReadDryRunFailureAndSaveMigration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.arg")
	legacy := "argument legacy-profile \"Legacy profile\"\n" +
		"meta profile=\"workspace\", cludia-next-ids=\"v1;P=3;L=1;C=1;CP=1;J=1\"\n" +
		"premise P1 ::T \"One\"\n" +
		"premise P2 ::T \"Two\"\n"
	if err := os.WriteFile(path, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if err := run([]string{"validate", "--json", path}, &stdout, &stderr); err != nil {
		t.Fatalf("validate: %v\nstderr: %s", err, stderr.String())
	}
	var readOutput validateOutput
	if err := json.Unmarshal(stdout.Bytes(), &readOutput); err != nil {
		t.Fatal(err)
	}
	if readOutput.Profile != "cludia" {
		t.Fatalf("read profile = %q", readOutput.Profile)
	}
	assertFileContents(t, path, legacy)

	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"delete", "--dry-run", "--json", path, "P2"}, &stdout, &stderr); err != nil {
		t.Fatalf("delete dry-run: %v\nstderr: %s", err, stderr.String())
	}
	var dryRun statementDeletionOutput
	if err := json.Unmarshal(stdout.Bytes(), &dryRun); err != nil {
		t.Fatal(err)
	}
	if !hasProfileMigrationChange(dryRun.Changes) {
		t.Fatalf("dry-run changes = %#v", dryRun.Changes)
	}
	assertFileContents(t, path, legacy)

	stdout.Reset()
	stderr.Reset()
	err := run([]string{"delete", "--json", path, "missing"}, &stdout, &stderr)
	if !errors.Is(err, errValidationFailed) {
		t.Fatalf("failed mutation error = %v", err)
	}
	assertFileContents(t, path, legacy)

	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"add", "--json", path, "--text", "Three"}, &stdout, &stderr); err != nil {
		t.Fatalf("add: %v\nstderr: %s", err, stderr.String())
	}
	var saved mutationOutput
	if err := json.Unmarshal(stdout.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Profile != "cludia" || !hasProfileMigrationChange(saved.Changes) {
		t.Fatalf("saved output = %#v", saved)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(contents), "profile=\"workspace\"") || !strings.Contains(string(contents), "profile=\"cludia\"") {
		t.Fatalf("saved profile was not migrated:\n%s", contents)
	}
	parsed := argfile.Load(path)
	if profile, ok := parsed.Document.MetadataValue("profile"); !ok || profile != "cludia" {
		t.Fatalf("parsed profile = %q, %v", profile, ok)
	}
}

func TestWorkspaceIsNotAnAcceptedCLIProfileName(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "broken-window-workspace.arg")
	var stdout, stderr bytes.Buffer
	err := run([]string{"validate", "--json", "--profile", "workspace", path}, &stdout, &stderr)
	if !errors.Is(err, errValidationFailed) {
		t.Fatalf("error = %v", err)
	}
	var output validateOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if output.Profile != "workspace" || len(output.Diagnostics) != 1 || output.Diagnostics[0].Code != "profile_unknown" {
		t.Fatalf("output = %#v", output)
	}
}

func assertFileContents(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("file changed\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func hasProfileMigrationChange(changes []changeOutput) bool {
	for _, change := range changes {
		if change.Operation == "updated" && change.ElementType == "metadata" && change.ID == "profile" {
			return true
		}
	}
	return false
}
