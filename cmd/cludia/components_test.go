// SPDX-FileCopyrightText: 2026 KeenWorks
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestComponentsListsDeterministicSummaries(t *testing.T) {
	path := componentTestWorkspace(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"components", path, "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("components: %v\nstderr: %s", err, stderr.String())
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatalf("decode components JSON: %v", err)
	}
	assertExactKeys(t, raw, "schema_version", "profile", "evaluation", "components", "diagnostics")
	var output componentsOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode typed components JSON: %v", err)
	}
	if len(output.Components) != 2 {
		t.Fatalf("components = %#v", output.Components)
	}
	if got := output.Components[0]; got.Anchor != "P1" || got.Statements != 3 || got.Junctors != 1 || got.Isolated {
		t.Fatalf("first component = %#v", got)
	}
	if got := output.Components[1]; got.Anchor != "P3" || got.Statements != 1 || !got.Isolated {
		t.Fatalf("second component = %#v", got)
	}
}

func TestComponentShowsCompleteIslandByJunctor(t *testing.T) {
	path := componentTestWorkspace(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"component", "--json", path, "J1"}, &stdout, &stderr); err != nil {
		t.Fatalf("component: %v\nstderr: %s", err, stderr.String())
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatalf("decode component JSON: %v", err)
	}
	assertExactKeys(t, raw, "schema_version", "profile", "evaluation", "anchor", "isolated", "statements", "junctors", "direct_supports", "defeats", "diagnostics")
	var output componentOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode typed component JSON: %v", err)
	}
	if output.Anchor != "P1" || output.Isolated || len(output.Statements) != 3 || len(output.Junctors) != 1 {
		t.Fatalf("component output = %#v", output)
	}
}

func TestComponentGivesJunctorIDPrecedenceOverStatementSlug(t *testing.T) {
	path := referenceCollisionWorkspace(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"component", path, "shared", "--json"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	var output componentOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatal(err)
	}
	if output.Anchor != "P2" || len(output.Junctors) != 1 || output.Junctors[0].ID != "shared" {
		t.Fatalf("component = %#v", output)
	}
}

func TestComponentShowsIsolatedStatementBySlug(t *testing.T) {
	path := componentTestWorkspace(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"component", path, "isolated", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("component isolated: %v", err)
	}
	var output componentOutput
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if output.Anchor != "P3" || !output.Isolated || len(output.Statements) != 1 {
		t.Fatalf("isolated output = %#v", output)
	}
}

func TestComponentsHumanOutputUsesSingularLabels(t *testing.T) {
	path := componentTestWorkspace(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"components", path}, &stdout, &stderr); err != nil {
		t.Fatalf("components: %v", err)
	}
	if got := stdout.String(); !bytes.Contains([]byte(got), []byte("isolated P3\t1 statement\t0 junctors")) {
		t.Fatalf("human output did not use singular statement:\n%s", got)
	}
}

func componentTestWorkspace(t *testing.T) string {
	t.Helper()
	path := twoPremiseWorkspace(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{
		"derive", path, "--source", "P1", "--source", "P2",
		"--target-text", "Combined",
	}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"add", path, "--text", "Isolated", "--slug", "isolated"}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	return path
}
