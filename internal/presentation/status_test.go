// SPDX-FileCopyrightText: 2026 KeenWorks
// SPDX-License-Identifier: GPL-3.0-or-later

package presentation

import (
	"testing"

	"github.com/tunesmith/cludia/internal/argument"
	"github.com/tunesmith/cludia/internal/evaluation"
)

func TestEffectiveStatementStatusUsesTruthForLeavesAndProvabilityForDerivedRoles(t *testing.T) {
	for _, test := range []struct {
		role  argument.Role
		truth argument.Truth
		want  string
	}{
		{argument.RolePremise, argument.TruthTrue, "T"},
		{argument.RolePremise, argument.TruthUnknown, "U"},
		{argument.RoleCounterpoint, argument.TruthFalse, "F"},
		{argument.RoleLemma, argument.TruthTrue, ProofProven},
		{argument.RoleLemma, argument.TruthUnknown, ProofPossiblyProven},
		{argument.RoleLemma, argument.TruthFalse, ProofNotProven},
		{argument.RoleConclusion, argument.TruthTrue, ProofProven},
	} {
		if got := EffectiveStatementStatus(test.role, test.truth); got != test.want {
			t.Fatalf("%s %s status = %q, want %q", test.role, test.truth, got, test.want)
		}
	}
}

func TestStatementStatusShowsLeafTransitionButNotStoredDerivedPlaceholder(t *testing.T) {
	premise := argument.Statement{Role: argument.RolePremise, Truth: argument.TruthTrue}
	if got := StatementStatus(premise, argument.TruthFalse, evaluation.TruthAsserted); got != "T → F" {
		t.Fatalf("premise status = %q", got)
	}
	lemma := argument.Statement{Role: argument.RoleLemma, Truth: argument.TruthUnknown}
	if got := StatementStatus(lemma, argument.TruthTrue, evaluation.TruthDerived); got != ProofProven {
		t.Fatalf("lemma status = %q", got)
	}
}
