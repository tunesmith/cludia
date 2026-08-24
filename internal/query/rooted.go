package query

import (
	"fmt"

	"github.com/tunesmith/cludia/internal/argument"
)

// Rooted returns the complete upstream support closure and attached recursive
// defeat chains for a selected non-counterpoint statement. The returned
// document is role-reconciled and omits workspace profile metadata so it can be
// validated and serialized as a Concludia artifact.
func Rooted(doc *argument.Document, reference string) (*argument.Document, error) {
	if doc == nil {
		return nil, fmt.Errorf("document is nil")
	}
	root, ok := doc.Statement(reference)
	if !ok {
		return nil, fmt.Errorf("root statement %q not found", reference)
	}
	if root.Role == argument.RoleCounterpoint {
		return nil, fmt.Errorf("root statement %s is a counterpoint", root.ID)
	}
	includedStatements := map[string]bool{root.ID: true}
	includedJunctors := make(map[string]bool)
	includedDirect := make(map[int]bool)
	includedDefeats := make(map[int]bool)
	for changed := true; changed; {
		changed = false
		for _, junctor := range doc.Junctors {
			if !includedStatements[junctor.Target] {
				continue
			}
			if !includedJunctors[junctor.ID] {
				includedJunctors[junctor.ID] = true
				changed = true
			}
			for _, source := range junctor.Sources {
				if !includedStatements[source] {
					includedStatements[source] = true
					changed = true
				}
			}
		}
		for i, support := range doc.DirectSupports {
			if !includedStatements[support.Target] {
				continue
			}
			if !includedDirect[i] {
				includedDirect[i] = true
				changed = true
			}
			if !includedStatements[support.Source] {
				includedStatements[support.Source] = true
				changed = true
			}
		}
		for i, defeat := range doc.Defeats {
			include := false
			switch defeat.Scope {
			case argument.DefeatPremise, argument.DefeatCounterpoint:
				include = includedStatements[defeat.To]
			case argument.DefeatInference:
				include = includedJunctors[defeat.JunctorID]
			}
			if !include {
				continue
			}
			if !includedDefeats[i] {
				includedDefeats[i] = true
				changed = true
			}
			if !includedStatements[defeat.From] {
				includedStatements[defeat.From] = true
				changed = true
			}
		}
	}

	result := &argument.Document{
		ID: doc.ID, Title: doc.Title,
		Metadata: []argument.Metadata{}, Statements: []argument.Statement{},
		Junctors: []argument.Junctor{}, DirectSupports: []argument.DirectSupport{},
		Defeats: []argument.Defeat{},
	}
	for _, metadata := range doc.Metadata {
		if metadata.Key != "profile" && metadata.Key != "root" {
			result.Metadata = append(result.Metadata, metadata)
		}
	}
	rootValue := root.ID
	if root.Slug != "" {
		rootValue = root.Slug
	}
	result.Metadata = append(result.Metadata, argument.Metadata{Key: "root", Value: rootValue})
	for _, statement := range doc.Statements {
		if includedStatements[statement.ID] {
			result.Statements = append(result.Statements, statement)
		}
	}
	for _, junctor := range doc.Junctors {
		if includedJunctors[junctor.ID] {
			result.Junctors = append(result.Junctors, copyRootedJunctor(junctor))
		}
	}
	for i, support := range doc.DirectSupports {
		if includedDirect[i] {
			result.DirectSupports = append(result.DirectSupports, support)
		}
	}
	for i, defeat := range doc.Defeats {
		if includedDefeats[i] {
			result.Defeats = append(result.Defeats, defeat)
		}
	}
	reconcileRootedRoles(result, root.ID)
	return result, nil
}

func reconcileRootedRoles(doc *argument.Document, rootID string) {
	incoming := make(map[string]bool)
	for _, junctor := range doc.Junctors {
		incoming[junctor.Target] = true
	}
	for _, support := range doc.DirectSupports {
		incoming[support.Target] = true
	}
	for i := range doc.Statements {
		statement := &doc.Statements[i]
		if statement.Role == argument.RoleCounterpoint {
			continue
		}
		switch {
		case statement.ID == rootID:
			statement.Role = argument.RoleConclusion
			statement.Truth = argument.TruthUnknown
		case incoming[statement.ID]:
			statement.Role = argument.RoleLemma
			statement.Truth = argument.TruthUnknown
		default:
			statement.Role = argument.RolePremise
		}
	}
}

func copyRootedJunctor(junctor argument.Junctor) argument.Junctor {
	junctor.Sources = append([]string(nil), junctor.Sources...)
	return junctor
}
