package argument

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var (
	idPattern   = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)
	slugPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
)

func ValidID(id string) bool {
	return idPattern.MatchString(id)
}

func ValidSlug(slug string) bool {
	return slugPattern.MatchString(slug)
}

func Slugify(text string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(text)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case unicode.IsSpace(r) || r == '-' || r == '_':
			if b.Len() > 0 && !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
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
