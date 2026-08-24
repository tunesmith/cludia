package argument

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	idPattern   = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)
	slugPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
)

const (
	maxSlugWords    = 8
	maxSlugLength   = 72
	maxSlugTokenLen = 24
)

var slugStopWords = map[string]bool{
	"a": true, "an": true, "and": true, "as": true, "at": true,
	"by": true, "for": true, "from": true, "in": true, "is": true,
	"of": true, "on": true, "or": true, "the": true, "to": true,
	"with": true,
}

func ValidID(id string) bool {
	return idPattern.MatchString(id)
}

func ValidSlug(slug string) bool {
	return slugPattern.MatchString(slug)
}

func Slugify(text string) string {
	text = strings.ReplaceAll(text, "’s", "")
	text = strings.ReplaceAll(text, "'s", "")
	var tokens []string
	var token strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(text)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			token.WriteRune(r)
		default:
			if token.Len() > 0 {
				tokens = append(tokens, token.String())
				token.Reset()
			}
		}
	}
	if token.Len() > 0 {
		tokens = append(tokens, token.String())
	}
	filtered := make([]string, 0, len(tokens))
	for _, value := range tokens {
		if !slugStopWords[value] {
			filtered = append(filtered, value)
		}
	}
	if len(filtered) > 0 {
		tokens = filtered
	}
	return compactSlug(tokens)
}

func compactSlug(tokens []string) string {
	result := make([]string, 0, min(len(tokens), maxSlugWords))
	length := 0
	for _, token := range tokens {
		if len(token) > maxSlugTokenLen {
			token = token[:maxSlugTokenLen]
		}
		additional := len(token)
		if len(result) > 0 {
			additional++
		}
		if len(result) >= maxSlugWords || length+additional > maxSlugLength {
			break
		}
		result = append(result, token)
		length += additional
	}
	return strings.Join(result, "-")
}

func NextStatementID(doc *Document, role Role) string {
	prefix := "P"
	switch role {
	case RoleLemma:
		prefix = "L"
	case RoleConclusion:
		prefix = "C"
	case RoleCounterpoint:
		prefix = "CP"
	}
	used := make(map[string]bool, len(doc.Statements)+len(doc.Junctors))
	for _, statement := range doc.Statements {
		used[statement.ID] = true
	}
	for _, junctor := range doc.Junctors {
		used[junctor.ID] = true
	}
	for n := 1; ; n++ {
		candidate := fmt.Sprintf("%s%d", prefix, n)
		if !used[candidate] {
			return candidate
		}
	}
}

func NextJunctorID(doc *Document) string {
	used := make(map[string]bool, len(doc.Junctors)+len(doc.Statements))
	for _, junctor := range doc.Junctors {
		used[junctor.ID] = true
	}
	for _, statement := range doc.Statements {
		used[statement.ID] = true
	}
	for n := 1; ; n++ {
		candidate := fmt.Sprintf("J%d", n)
		if !used[candidate] {
			return candidate
		}
	}
}

func UniqueSlug(doc *Document, text string) string {
	base := Slugify(text)
	if base == "" {
		base = "statement"
	}
	used := make(map[string]bool, len(doc.Statements))
	for _, statement := range doc.Statements {
		if statement.Slug != "" {
			used[statement.Slug] = true
		}
	}
	if !used[base] {
		return base
	}
	for n := 2; ; n++ {
		candidate := fmt.Sprintf("%s-%d", base, n)
		if !used[candidate] {
			return candidate
		}
	}
}
