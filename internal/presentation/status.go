// SPDX-FileCopyrightText: 2026 KeenWorks
// SPDX-License-Identifier: GPL-3.0-or-later

// Package presentation contains shared human-facing formatting over domain
// facts. It does not change persisted truth or evaluation semantics.
package presentation

import (
	"fmt"

	"github.com/tunesmith/cludia/internal/argument"
	"github.com/tunesmith/cludia/internal/evaluation"
)

// StatementStatusWidth leaves room for an authored/effective leaf transition
// such as "T → F" while keeping ordinary truth and proof glyphs aligned.
const StatementStatusWidth = 5

// StatementStatusHeader labels the mixed truth/provability column without
// implying that every row contains the same kind of value.
const StatementStatusHeader = "∴"

const (
	ProofProven         = "⊢"
	ProofPossiblyProven = "◇"
	ProofNotProven      = "⊬"
)

// IsProofRole reports whether effective truth is presented as provability.
func IsProofRole(role argument.Role) bool {
	return role == argument.RoleLemma || role == argument.RoleConclusion
}

// EffectiveStatementStatus presents literal truth for premises and
// counterpoints, and Concludia-compatible provability for lemmas/conclusions.
func EffectiveStatementStatus(role argument.Role, truth argument.Truth) string {
	if !IsProofRole(role) {
		return string(truth)
	}
	switch truth {
	case argument.TruthTrue:
		return ProofProven
	case argument.TruthFalse:
		return ProofNotProven
	default:
		return ProofPossiblyProven
	}
}

// StatementStatus includes a stored-to-effective transition for authored leaf
// truth changed by grounded defeat. Derived statements show only provability;
// their stored U is a persistence placeholder rather than authored status.
func StatementStatus(statement argument.Statement, effective argument.Truth, source evaluation.TruthSource) string {
	if effective != argument.TruthTrue && effective != argument.TruthFalse && effective != argument.TruthUnknown {
		effective = statement.Truth
	}
	if !IsProofRole(statement.Role) && source == evaluation.TruthAsserted && statement.Truth != effective {
		return fmt.Sprintf("%s → %s", statement.Truth, effective)
	}
	return EffectiveStatementStatus(statement.Role, effective)
}

// StatusNoun names a standalone human-readable status value.
func StatusNoun(role argument.Role) string {
	if IsProofRole(role) {
		return "proof"
	}
	return "truth"
}
