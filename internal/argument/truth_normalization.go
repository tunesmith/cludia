// SPDX-FileCopyrightText: 2026 KeenWorks
// SPDX-License-Identifier: GPL-3.0-or-later

package argument

import (
	"crypto/sha256"
	"fmt"
)

type TruthNormalization struct {
	ID            string `json:"id"`
	PreviousTruth Truth  `json:"previous_truth"`
	CurrentTruth  Truth  `json:"current_truth"`
}

type NormalizeTruthResult struct {
	Statements []TruthNormalization
	PlanToken  string
	Changed    bool
}

func NormalizeNonLeafTruth(doc *Document) (*Document, NormalizeTruthResult, error) {
	if doc == nil {
		return nil, NormalizeTruthResult{}, mutationError("document_nil", "document is nil", "")
	}
	next := doc.Clone()
	changes := make([]TruthNormalization, 0)
	for index := range next.Statements {
		statement := &next.Statements[index]
		if !next.HasIncomingSupport(statement.ID) || statement.Truth == TruthUnknown {
			continue
		}
		changes = append(changes, TruthNormalization{ID: statement.ID, PreviousTruth: statement.Truth, CurrentTruth: TruthUnknown})
		statement.Truth = TruthUnknown
	}
	state, err := marshalDocumentState(doc)
	if err != nil {
		return nil, NormalizeTruthResult{}, err
	}
	sum := sha256.Sum256(append([]byte("normalize-truth-v1\x00"), state...))
	return next, NormalizeTruthResult{Statements: changes, PlanToken: fmt.Sprintf("%x", sum[:]), Changed: len(changes) > 0}, nil
}
