package query

import (
	"strings"

	"github.com/tunesmith/cludia/internal/argument"
)

type StatementMatch struct {
	Statement argument.Statement
	Fields    []string
}

// SearchStatements performs a case-insensitive substring search over statement
// IDs, slugs, and text while preserving document order.
func SearchStatements(doc *argument.Document, search string) []StatementMatch {
	search = strings.ToLower(strings.TrimSpace(search))
	if doc == nil || search == "" {
		return []StatementMatch{}
	}
	matches := make([]StatementMatch, 0)
	for _, statement := range doc.Statements {
		fields := make([]string, 0, 3)
		if strings.Contains(strings.ToLower(statement.ID), search) {
			fields = append(fields, "id")
		}
		if statement.Slug != "" && strings.Contains(strings.ToLower(statement.Slug), search) {
			fields = append(fields, "slug")
		}
		if strings.Contains(strings.ToLower(statement.Text), search) {
			fields = append(fields, "text")
		}
		if len(fields) > 0 {
			matches = append(matches, StatementMatch{Statement: statement, Fields: fields})
		}
	}
	return matches
}
