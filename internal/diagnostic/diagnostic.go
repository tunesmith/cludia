// SPDX-FileCopyrightText: 2026 KeenWorks
// SPDX-License-Identifier: GPL-3.0-or-later

package diagnostic

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

type Diagnostic struct {
	Code     string   `json:"code"`
	Message  string   `json:"message"`
	Severity Severity `json:"severity"`
	Line     int      `json:"line,omitempty"`
	Element  string   `json:"element,omitempty"`
}

func HasErrors(diagnostics []Diagnostic) bool {
	for _, item := range diagnostics {
		if item.Severity == SeverityError {
			return true
		}
	}
	return false
}
