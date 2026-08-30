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
	allocator, err := NewIDAllocator(doc)
	if err != nil {
		return ""
	}
	id, err := allocator.Statement(role, "")
	if err != nil {
		return ""
	}
	return id
}

func NextJunctorID(doc *Document) string {
	allocator, err := NewIDAllocator(doc)
	if err != nil {
		return ""
	}
	id, err := allocator.Junctor("")
	if err != nil {
		return ""
	}
	return id
}

func UniqueSlug(doc *Document, text string) string {
	base := Slugify(text)
	if base == "" {
		base = "statement"
	} else if base[0] >= '0' && base[0] <= '9' {
		base = compactSlug(append([]string{"statement"}, strings.Split(base, "-")...))
	}
	used := make(map[string]bool, len(doc.Statements))
	for _, statement := range doc.Statements {
		used[statement.ID] = true
		if statement.Slug != "" {
			used[statement.Slug] = true
		}
	}
	for _, junctor := range doc.Junctors {
		used[junctor.ID] = true
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

// SlugIDCollision reports whether a slug would shadow another element's
// durable ID. ownerID allows a statement whose custom ID equals its own slug to
// remain readable for compatibility.
func SlugIDCollision(doc *Document, slug, ownerID string) (ElementType, string, bool) {
	if doc == nil || slug == "" {
		return "", "", false
	}
	for _, statement := range doc.Statements {
		if statement.ID == slug && statement.ID != ownerID {
			return ElementStatement, statement.ID, true
		}
	}
	for _, junctor := range doc.Junctors {
		if junctor.ID == slug {
			return ElementJunctor, junctor.ID, true
		}
	}
	return "", "", false
}
