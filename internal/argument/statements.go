package argument

import (
	"fmt"
	"strings"
)

type StatementInput struct {
	Text        string
	RequestedID string
	Slug        string
	Truth       Truth
	Kind        Kind
}

type InitializeOptions struct {
	DocumentID string
	Title      string
	Statement  StatementInput
}

type DocumentIdentityOptions struct {
	ID    string
	Title string
}

type EditStatementOptions struct {
	Reference string
	Text      *string
	Truth     *Truth
	Kind      *Kind
}

type EditStatementResult struct {
	Previous Statement
	Current  Statement
	Changed  bool
}

type SlugMutationMode string

const (
	SlugExplicit SlugMutationMode = "explicit"
	SlugFromText SlugMutationMode = "from_text"
	SlugClear    SlugMutationMode = "clear"
)

type RenameSlugOptions struct {
	Reference string
	Mode      SlugMutationMode
	Slug      string
}

type RenameSlugResult struct {
	Statement           Statement
	PreviousSlug        string
	CurrentSlug         string
	RootMetadataUpdated bool
	Changed             bool
}

type BatchStatementError struct {
	Index int
	Err   error
}

func (e *BatchStatementError) Error() string { return e.Err.Error() }

func InitializeDocument(options InitializeOptions) (*Document, Statement, error) {
	documentID := strings.TrimSpace(options.DocumentID)
	title := strings.TrimSpace(options.Title)
	if !ValidID(documentID) {
		return nil, Statement{}, mutationError("document_id_invalid", fmt.Sprintf("invalid document id %q", documentID), documentID)
	}
	if title == "" {
		return nil, Statement{}, mutationError("document_title_required", "document title is required", documentID)
	}
	doc := &Document{
		ID: documentID, Title: title,
		Metadata:       []Metadata{{Key: "profile", Value: "workspace"}, {Key: "version", Value: "0.1.0"}},
		Statements:     []Statement{},
		Junctors:       []Junctor{},
		DirectSupports: []DirectSupport{},
		Defeats:        []Defeat{},
	}
	next, statement, err := addStatement(doc, options.Statement)
	return next, statement, err
}

func WithDocumentIdentity(doc *Document, options DocumentIdentityOptions) (*Document, error) {
	if doc == nil {
		return nil, mutationError("document_nil", "document is nil", "")
	}
	next := doc.Clone()
	if id := strings.TrimSpace(options.ID); id != "" {
		if !ValidID(id) {
			return nil, mutationError("document_id_invalid", fmt.Sprintf("invalid document id %q", id), id)
		}
		next.ID = id
	}
	if title := strings.TrimSpace(options.Title); title != "" {
		next.Title = title
	}
	return next, nil
}

func AddStatement(doc *Document, input StatementInput) (*Document, Statement, error) {
	if doc == nil {
		return nil, Statement{}, mutationError("document_nil", "document is nil", "")
	}
	return addStatement(doc.Clone(), input)
}

func AddStatements(doc *Document, inputs []StatementInput) (*Document, []Statement, error) {
	if doc == nil {
		return nil, nil, mutationError("document_nil", "document is nil", "")
	}
	if len(inputs) == 0 {
		return nil, nil, mutationError("batch_statements_required", "batch requires at least one statement", "")
	}
	next := doc.Clone()
	allocator, err := NewIDAllocator(next)
	if err != nil {
		return nil, nil, err
	}
	statements := make([]Statement, 0, len(inputs))
	for index, input := range inputs {
		statement, err := buildStatement(next, allocator, input)
		if err != nil {
			return nil, nil, &BatchStatementError{Index: index, Err: err}
		}
		next.Statements = append(next.Statements, statement)
		statements = append(statements, statement)
	}
	allocator.Persist(next)
	return next, statements, nil
}

func EditStatement(doc *Document, options EditStatementOptions) (*Document, EditStatementResult, error) {
	if doc == nil {
		return nil, EditStatementResult{}, mutationError("document_nil", "document is nil", "")
	}
	next := doc.Clone()
	statement, ok := next.Statement(strings.TrimSpace(options.Reference))
	if !ok {
		return nil, EditStatementResult{}, mutationError("statement_not_found", fmt.Sprintf("statement %q not found", options.Reference), options.Reference)
	}
	previous := *statement
	if options.Text != nil {
		text := strings.TrimSpace(*options.Text)
		if text == "" {
			return nil, EditStatementResult{}, mutationError("statement_text_required", "statement text is required", statement.ID)
		}
		statement.Text = text
	}
	failures := make([]MutationError, 0, 2)
	if options.Truth != nil {
		if !validTruth(*options.Truth) {
			failures = append(failures, MutationError{Code: "truth_invalid", Message: fmt.Sprintf("invalid truth %q; expected T, F, or U", *options.Truth), Element: statement.ID})
		} else {
			statement.Truth = *options.Truth
		}
	}
	if options.Kind != nil {
		if !validKind(*options.Kind) {
			failures = append(failures, MutationError{Code: "kind_invalid", Message: fmt.Sprintf("invalid kind %q; expected fact or value", *options.Kind), Element: statement.ID})
		} else {
			statement.Kind = *options.Kind
		}
	}
	if len(failures) > 0 {
		return nil, EditStatementResult{}, &MutationErrors{Failures: failures}
	}
	return next, EditStatementResult{Previous: previous, Current: *statement, Changed: previous != *statement}, nil
}

func RenameStatementSlug(doc *Document, options RenameSlugOptions) (*Document, RenameSlugResult, error) {
	if doc == nil {
		return nil, RenameSlugResult{}, mutationError("document_nil", "document is nil", "")
	}
	next := doc.Clone()
	statement, ok := next.Statement(strings.TrimSpace(options.Reference))
	if !ok {
		return nil, RenameSlugResult{}, mutationError("statement_not_found", fmt.Sprintf("statement %q not found", options.Reference), options.Reference)
	}
	previous := statement.Slug
	current := ""
	switch options.Mode {
	case SlugExplicit:
		current = strings.TrimSpace(options.Slug)
		if current == "" {
			return nil, RenameSlugResult{}, mutationError("statement_slug_required", "explicit slug must not be empty", statement.ID)
		}
	case SlugFromText:
		statement.Slug = ""
		current = UniqueSlug(next, statement.Text)
	case SlugClear:
		current = ""
	default:
		return nil, RenameSlugResult{}, mutationError("slug_mode_invalid", fmt.Sprintf("invalid slug mutation mode %q", options.Mode), string(options.Mode))
	}
	if current != "" {
		if !ValidSlug(current) {
			return nil, RenameSlugResult{}, mutationError("statement_slug_invalid", fmt.Sprintf("invalid statement slug %q", current), statement.ID)
		}
		if owner := slugOwner(next, current, statement.ID); owner != "" {
			return nil, RenameSlugResult{}, mutationError("statement_slug_duplicate", fmt.Sprintf("slug %q is already used by %s", current, owner), statement.ID)
		}
		if elementType, id, collides := SlugIDCollision(next, current, statement.ID); collides {
			return nil, RenameSlugResult{}, mutationError("statement_slug_id_collision", fmt.Sprintf("slug %q would be shadowed by %s id %s; choose a different slug", current, elementType, id), statement.ID)
		}
	}
	statement.Slug = current
	rootUpdated := false
	if previous != "" && previous != current {
		for index := range next.Metadata {
			metadata := &next.Metadata[index]
			if metadata.Key == "root" && metadata.Value == previous {
				metadata.Value = current
				if current == "" {
					metadata.Value = statement.ID
				}
				rootUpdated = true
			}
		}
	}
	return next, RenameSlugResult{
		Statement: *statement, PreviousSlug: previous, CurrentSlug: current,
		RootMetadataUpdated: rootUpdated, Changed: previous != current,
	}, nil
}

func addStatement(doc *Document, input StatementInput) (*Document, Statement, error) {
	allocator, err := NewIDAllocator(doc)
	if err != nil {
		return nil, Statement{}, err
	}
	statement, err := buildStatement(doc, allocator, input)
	if err != nil {
		return nil, Statement{}, err
	}
	doc.Statements = append(doc.Statements, statement)
	allocator.Persist(doc)
	return doc, statement, nil
}

func buildStatement(doc *Document, allocator *IDAllocator, input StatementInput) (Statement, error) {
	text := strings.TrimSpace(input.Text)
	id, err := allocator.Statement(RolePremise, strings.TrimSpace(input.RequestedID))
	if err != nil {
		return Statement{}, err
	}
	if text == "" {
		return Statement{}, mutationError("statement_text_required", "statement text is required", id)
	}
	slug := strings.TrimSpace(input.Slug)
	if slug == "" {
		slug = UniqueSlug(doc, text)
	} else {
		if !ValidSlug(slug) {
			return Statement{}, mutationError("statement_slug_invalid", fmt.Sprintf("invalid statement slug %q", slug), id)
		}
		if owner := slugOwner(doc, slug, id); owner != "" {
			return Statement{}, mutationError("statement_slug_duplicate", fmt.Sprintf("slug %q is already used by %s", slug, owner), id)
		}
		if elementType, existingID, collides := SlugIDCollision(doc, slug, id); collides {
			return Statement{}, mutationError("statement_slug_id_collision", fmt.Sprintf("slug %q would be shadowed by %s id %s; choose a different slug", slug, elementType, existingID), id)
		}
	}
	failures := make([]MutationError, 0, 2)
	if !validTruth(input.Truth) {
		failures = append(failures, MutationError{Code: "truth_invalid", Message: fmt.Sprintf("invalid truth %q; expected T, F, or U", input.Truth), Element: id})
	}
	if !validKind(input.Kind) {
		failures = append(failures, MutationError{Code: "kind_invalid", Message: fmt.Sprintf("invalid kind %q; expected fact or value", input.Kind), Element: id})
	}
	if len(failures) > 0 {
		return Statement{}, &MutationErrors{Failures: failures}
	}
	return Statement{ID: id, Slug: slug, Role: RolePremise, Kind: input.Kind, Truth: input.Truth, Text: text}, nil
}

func slugOwner(doc *Document, slug, exceptID string) string {
	for _, statement := range doc.Statements {
		if statement.ID != exceptID && statement.Slug == slug {
			return statement.ID
		}
	}
	return ""
}

func validTruth(value Truth) bool {
	return value == TruthTrue || value == TruthFalse || value == TruthUnknown
}

func validKind(value Kind) bool { return value == KindFact || value == KindValue }
