// SPDX-FileCopyrightText: 2026 KeenWorks
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/tunesmith/cludia/internal/argfile"
	"github.com/tunesmith/cludia/internal/argument"
	"github.com/tunesmith/cludia/internal/diagnostic"
	"github.com/tunesmith/cludia/internal/validation"
)

func TestRootJSONReturnsCompleteExportableClosure(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "broken-window-workspace.arg")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"root", path, "crossed-wall", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("root: %v\nstderr: %s", err, stderr.String())
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatalf("decode root JSON: %v", err)
	}
	assertExactKeys(t, raw, "schema_version", "profile", "root", "exportable", "document", "stats", "evaluation", "diagnostics")
	var output rootedOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode typed root JSON: %v", err)
	}
	if !output.Exportable || output.Root != "L1" || len(output.Document.Statements) != 8 || len(output.Document.Junctors) != 2 || len(output.Document.Defeats) != 4 {
		t.Fatalf("root output = %#v", output)
	}
	root, found := output.Document.Statement("L1")
	if !found || root.Role != argument.RoleConclusion {
		t.Fatalf("root statement = %#v, found %t", root, found)
	}
}

func TestRootJSONUsesEmptyArraysForAbsentRelationTypes(t *testing.T) {
	path := componentTestWorkspace(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"root", path, "L1", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("root: %v", err)
	}
	var output rootedOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if output.Document.DirectSupports == nil || output.Document.Defeats == nil {
		t.Fatalf("rooted relation arrays encoded as nil: %#v", output.Document)
	}
}

func TestExportWritesNewStrictArtifactWithoutChangingWorkspace(t *testing.T) {
	source := filepath.Join("..", "..", "examples", "broken-window-workspace.arg")
	workspaceBefore, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	outputPath := filepath.Join(t.TempDir(), "export.arg")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"export", source, "--output", outputPath, "--root", "L1", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("export: %v\nstderr: %s", err, stderr.String())
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatalf("decode export JSON: %v", err)
	}
	assertExactKeys(t, raw, "schema_version", "action", "profile", "root", "output", "written", "document", "stats", "diagnostics")
	var output exportOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil || !output.Written || output.Root != "L1" {
		t.Fatalf("export output = %#v, err %v", output, err)
	}
	parsed := argfile.ParseFile(outputPath)
	if diagnostic.HasErrors(parsed.Diagnostics) {
		t.Fatalf("export parse: %#v", parsed.Diagnostics)
	}
	if result := validation.Validate(parsed.Document, validation.ProfileConcludia); !result.OK() || len(result.Diagnostics) != 0 {
		t.Fatalf("export validation: %#v", result.Diagnostics)
	}
	if _, found := parsed.Document.Statement("P4"); found {
		t.Fatal("unrelated P4 exported")
	}
	workspaceAfter, err := os.ReadFile(source)
	if err != nil || !bytes.Equal(workspaceBefore, workspaceAfter) {
		t.Fatalf("workspace changed during export: %v", err)
	}
}

func TestExportOverridesIdentityForCuratedArtifact(t *testing.T) {
	source := filepath.Join("..", "..", "examples", "broken-window-workspace.arg")
	expectedPath := filepath.Join("..", "..", "examples", "broken-window-conclusion.arg")
	outputPath := filepath.Join(t.TempDir(), "curated.arg")
	var stdout, stderr bytes.Buffer
	if err := run([]string{
		"export", source, "--root", "L1", "--output", outputPath,
		"--id", "crossed-wall", "--title", "The Intruder Crossed the Garden Wall",
	}, &stdout, &stderr); err != nil {
		t.Fatalf("export curated: %v", err)
	}
	actual := argfile.ParseFile(outputPath)
	expected := argfile.ParseFile(expectedPath)
	if diagnostic.HasErrors(actual.Diagnostics) || diagnostic.HasErrors(expected.Diagnostics) {
		t.Fatalf("parse diagnostics: actual %#v expected %#v", actual.Diagnostics, expected.Diagnostics)
	}
	// The rooted fixture carries an authored Concludia graph revision. Cludia's
	// source workspace intentionally omits graph-version metadata, so export
	// neither invents nor copies that independently authored value.
	metadata := expected.Document.Metadata[:0]
	for _, item := range expected.Document.Metadata {
		if item.Key != "version" {
			metadata = append(metadata, item)
		}
	}
	expected.Document.Metadata = metadata
	if !reflect.DeepEqual(actual.Document, expected.Document) {
		t.Fatalf("curated export differs: actual %#v expected %#v", actual.Document, expected.Document)
	}
}

func TestExportInvalidRootLeavesNoOutput(t *testing.T) {
	source := filepath.Join("..", "..", "examples", "broken-window-workspace.arg")
	outputPath := filepath.Join(t.TempDir(), "invalid.arg")
	var stdout, stderr bytes.Buffer
	err := run([]string{"export", source, "--root", "P4", "--output", outputPath, "--json"}, &stdout, &stderr)
	if !errors.Is(err, errValidationFailed) {
		t.Fatalf("export error = %v", err)
	}
	var output exportOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil || output.Written || len(output.Diagnostics) == 0 {
		t.Fatalf("invalid export output = %#v, err %v", output, err)
	}
	if _, err := os.Stat(outputPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("invalid export left output: %v", err)
	}
}

func TestExportRefusesExistingOutput(t *testing.T) {
	source := filepath.Join("..", "..", "examples", "broken-window-workspace.arg")
	outputPath := filepath.Join(t.TempDir(), "existing.arg")
	const original = "existing\n"
	if err := os.WriteFile(outputPath, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err := run([]string{"export", source, "--root", "L1", "--output", outputPath, "--json"}, &stdout, &stderr)
	if !errors.Is(err, errValidationFailed) {
		t.Fatalf("export error = %v", err)
	}
	var output exportOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil || output.Written {
		t.Fatalf("existing output result = %#v, err %v", output, err)
	}
	data, err := os.ReadFile(outputPath)
	if err != nil || string(data) != original {
		t.Fatalf("existing output changed: %q, err %v", data, err)
	}
}
