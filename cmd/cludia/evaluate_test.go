// SPDX-FileCopyrightText: 2026 KeenWorks
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tunesmith/cludia/internal/argument"
	"github.com/tunesmith/cludia/internal/evaluation"
)

func TestEvaluateJSONAndHumanContract(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "broken-window-workspace.arg")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"evaluate", path, "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("evaluate: %v\n%s", err, stderr.String())
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	assertExactKeys(t, raw, "schema_version", "profile", "document", "evaluation", "diagnostics")
	var output evaluateOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if output.SchemaVersion != 2 || output.Evaluation.SchemaVersion != evaluation.SchemaVersion || output.Evaluation.Mode != evaluation.ModeGrounded {
		t.Fatalf("versions = %#v", output)
	}
	var lemmaTruth argument.Truth
	for _, statement := range output.Evaluation.Statements {
		if statement.ID == "L1" {
			lemmaTruth = statement.EffectiveTruth
		}
	}
	if lemmaTruth != argument.TruthTrue {
		t.Fatalf("L1 effective truth = %s", lemmaTruth)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"evaluate", path}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if human := stdout.String(); !strings.Contains(human, "Evaluation v1 · grounded") || !strings.Contains(human, "P1\tstored T\ttruth T\tasserted") || !strings.Contains(human, "L1\tstored U\tproof ⊢\tderived") {
		t.Fatalf("human evaluation:\n%s", human)
	}
}
