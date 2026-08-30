package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type processResult struct {
	exitCode int
	stdout   []byte
	stderr   []byte
}

func TestCLIProcessContract(t *testing.T) {
	binary := buildProcessTestBinary(t)

	t.Run("successful read is JSON on stdout only", func(t *testing.T) {
		path, err := filepath.Abs(filepath.Join("..", "..", "examples", "broken-window-workspace.arg"))
		if err != nil {
			t.Fatal(err)
		}
		result := runCLIProcess(t, binary, "validate", path, "--json")
		if result.exitCode != 0 || len(result.stderr) != 0 {
			t.Fatalf("exit=%d stdout=%q stderr=%q", result.exitCode, result.stdout, result.stderr)
		}
		var output map[string]json.RawMessage
		if err := json.Unmarshal(result.stdout, &output); err != nil {
			t.Fatal(err)
		}
		var schemaVersion int
		if err := json.Unmarshal(output["schema_version"], &schemaVersion); err != nil || schemaVersion != 2 {
			t.Fatalf("schema version=%d, err=%v", schemaVersion, err)
		}
		if got := strings.TrimSpace(string(output["diagnostics"])); got != "[]" {
			t.Fatalf("empty diagnostics = %s", got)
		}
	})

	t.Run("JSON validation failure uses stdout and nonzero exit", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "invalid.arg")
		writeProcessFixture(t, path, "argument invalid \"Invalid\"\npremise P1 ::T \"One\"\nlemma L1 \"Broken\"\n  <- AND(P1, missing)\n")
		result := runCLIProcess(t, binary, "validate", path, "--json")
		if result.exitCode == 0 || len(result.stderr) != 0 {
			t.Fatalf("exit=%d stdout=%q stderr=%q", result.exitCode, result.stdout, result.stderr)
		}
		var output failureOutput
		if err := json.Unmarshal(result.stdout, &output); err != nil || output.OK || len(output.Diagnostics) == 0 {
			t.Fatalf("failure=%#v, err=%v", output, err)
		}
	})

	t.Run("human validation failure also keeps diagnostics on stdout", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "invalid.arg")
		writeProcessFixture(t, path, "argument invalid \"Invalid\"\npremise P1 ::T \"One\"\nlemma L1 \"Broken\"\n  <- AND(P1, missing)\n")
		result := runCLIProcess(t, binary, "validate", path)
		if result.exitCode == 0 || len(result.stdout) == 0 || len(result.stderr) != 0 || !bytes.Contains(result.stdout, []byte("reference_unknown")) {
			t.Fatalf("exit=%d stdout=%q stderr=%q", result.exitCode, result.stdout, result.stderr)
		}
	})

	t.Run("usage failure uses stderr and no JSON", func(t *testing.T) {
		result := runCLIProcess(t, binary, "validate", "--json")
		if result.exitCode == 0 || len(result.stdout) != 0 || !bytes.Contains(result.stderr, []byte("Usage:")) || !bytes.Contains(result.stderr, []byte("validate expects exactly one file")) {
			t.Fatalf("exit=%d stdout=%q stderr=%q", result.exitCode, result.stdout, result.stderr)
		}
	})

	t.Run("warnings remain successful JSON diagnostics", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "warning.arg")
		writeProcessFixture(t, path, "argument warning \"Warning\"\nmeta profile=\"workspace\", cludia-next-ids=\"v1;P=1;L=1;C=1;CP=1;J=1\"\npremise P2 ::T \"Two\"\n")
		result := runCLIProcess(t, binary, "validate", path, "--json")
		if result.exitCode != 0 || len(result.stderr) != 0 {
			t.Fatalf("exit=%d stdout=%q stderr=%q", result.exitCode, result.stdout, result.stderr)
		}
		var output validateOutput
		if err := json.Unmarshal(result.stdout, &output); err != nil || !output.OK || len(output.Diagnostics) != 1 || output.Diagnostics[0].Code != "next_ids_stale" {
			t.Fatalf("warning output=%#v, err=%v", output, err)
		}
	})

	t.Run("dry run succeeds without writing", func(t *testing.T) {
		path := newProcessWorkspace(t, binary)
		before, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		result := runCLIProcess(t, binary, "delete", path, "P2", "--dry-run", "--json")
		if result.exitCode != 0 || len(result.stderr) != 0 {
			t.Fatalf("exit=%d stdout=%q stderr=%q", result.exitCode, result.stdout, result.stderr)
		}
		var output statementDeletionOutput
		if err := json.Unmarshal(result.stdout, &output); err != nil || !output.DryRun {
			t.Fatalf("dry run=%#v, err=%v", output, err)
		}
		after, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(before, after) {
			t.Fatalf("dry run changed file: %v", err)
		}
	})

	t.Run("stale two-phase apply is structured and does not write", func(t *testing.T) {
		path := newProcessWorkspace(t, binary)
		planResult := runCLIProcess(t, binary, "renumber", path, "--dry-run", "--json")
		if planResult.exitCode != 0 {
			t.Fatalf("plan exit=%d stdout=%q stderr=%q", planResult.exitCode, planResult.stdout, planResult.stderr)
		}
		var plan renumberOutput
		if err := json.Unmarshal(planResult.stdout, &plan); err != nil || plan.PlanToken == "" {
			t.Fatalf("plan=%#v, err=%v", plan, err)
		}
		added := runCLIProcess(t, binary, "add", path, "--text", "State change", "--json")
		if added.exitCode != 0 {
			t.Fatalf("add exit=%d stdout=%q stderr=%q", added.exitCode, added.stdout, added.stderr)
		}
		beforeApply, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		result := runCLIProcess(t, binary, "renumber", path, "--apply-token", plan.PlanToken, "--json")
		if result.exitCode == 0 || len(result.stderr) != 0 {
			t.Fatalf("exit=%d stdout=%q stderr=%q", result.exitCode, result.stdout, result.stderr)
		}
		var failure failureOutput
		if err := json.Unmarshal(result.stdout, &failure); err != nil || len(failure.Diagnostics) != 1 || failure.Diagnostics[0].Code != "renumber_plan_stale" {
			t.Fatalf("failure=%#v, err=%v", failure, err)
		}
		afterApply, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(beforeApply, afterApply) {
			t.Fatalf("stale apply changed file: %v", err)
		}
	})
}

func buildProcessTestBinary(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	binary := filepath.Join(directory, "cludia")
	command := exec.Command("go", "build", "-o", binary, ".")
	command.Dir = "."
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("build process-test binary: %v\n%s", err, output)
	}
	return binary
}

func runCLIProcess(t *testing.T, binary string, args ...string) processResult {
	t.Helper()
	command := exec.Command(binary, args...)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	result := processResult{stdout: stdout.Bytes(), stderr: stderr.Bytes()}
	if err == nil {
		return result
	}
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) {
		t.Fatalf("run %v: %v", args, err)
	}
	result.exitCode = exitError.ExitCode()
	return result
}

func newProcessWorkspace(t *testing.T, binary string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "workspace.arg")
	for _, args := range [][]string{
		{"init", path, "--title", "Process", "--text", "First", "--json"},
		{"add", path, "--text", "Second", "--json"},
	} {
		result := runCLIProcess(t, binary, args...)
		if result.exitCode != 0 || len(result.stderr) != 0 {
			t.Fatalf("setup %v exit=%d stdout=%q stderr=%q", args, result.exitCode, result.stdout, result.stderr)
		}
	}
	return path
}

func writeProcessFixture(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}
