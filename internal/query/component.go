// SPDX-FileCopyrightText: 2026 KeenWorks
// SPDX-License-Identifier: GPL-3.0-or-later

package query

import "github.com/tunesmith/cludia/internal/argument"

// Component is a computed reasoning island. Anchor is the first statement in
// document order and is only a deterministic reference for this view; it is not
// a durable component identity.
type Component struct {
	Anchor         string                   `json:"anchor"`
	Statements     []argument.Statement     `json:"statements"`
	Junctors       []argument.Junctor       `json:"junctors"`
	DirectSupports []argument.DirectSupport `json:"direct_supports"`
	Defeats        []argument.Defeat        `json:"defeats"`
	Isolated       bool                     `json:"isolated"`
}

func Components(doc *argument.Document) []Component {
	if doc == nil || len(doc.Statements) == 0 {
		return []Component{}
	}
	sets := newDisjointSet(doc)
	for _, junctor := range doc.Junctors {
		for _, source := range junctor.Sources {
			sets.union(source, junctor.Target)
		}
	}
	for _, support := range doc.DirectSupports {
		sets.union(support.Source, support.Target)
	}
	for _, defeat := range doc.Defeats {
		switch defeat.Scope {
		case argument.DefeatPremise, argument.DefeatCounterpoint:
			sets.union(defeat.From, defeat.To)
		case argument.DefeatInference:
			target := defeat.AtTarget
			if target == "" {
				if junctor, ok := doc.Junctor(defeat.JunctorID); ok {
					target = junctor.Target
				}
			}
			sets.union(defeat.From, target)
		}
	}

	components := make([]Component, 0)
	indexByRoot := make(map[string]int)
	for _, statement := range doc.Statements {
		root := sets.find(statement.ID)
		index, exists := indexByRoot[root]
		if !exists {
			index = len(components)
			indexByRoot[root] = index
			components = append(components, Component{
				Anchor: statement.ID, Statements: []argument.Statement{},
				Junctors: []argument.Junctor{}, DirectSupports: []argument.DirectSupport{},
				Defeats: []argument.Defeat{},
			})
		}
		components[index].Statements = append(components[index].Statements, statement)
	}
	for _, junctor := range doc.Junctors {
		if index, ok := indexByRoot[sets.find(junctor.Target)]; ok {
			copy := junctor
			copy.Sources = append([]string(nil), junctor.Sources...)
			components[index].Junctors = append(components[index].Junctors, copy)
		}
	}
	for _, support := range doc.DirectSupports {
		if index, ok := indexByRoot[sets.find(support.Target)]; ok {
			components[index].DirectSupports = append(components[index].DirectSupports, support)
		}
	}
	for _, defeat := range doc.Defeats {
		if index, ok := indexByRoot[sets.find(defeat.From)]; ok {
			components[index].Defeats = append(components[index].Defeats, defeat)
		}
	}
	for i := range components {
		component := &components[i]
		component.Isolated = len(component.Statements) == 1 && len(component.Junctors) == 0 && len(component.DirectSupports) == 0 && len(component.Defeats) == 0
	}
	return components
}

func ComponentContaining(doc *argument.Document, reference string) (Component, bool) {
	resolved, ok := doc.ResolveElement(reference)
	if !ok {
		return Component{}, false
	}
	for _, component := range Components(doc) {
		for _, statement := range component.Statements {
			if resolved.Type == argument.ElementStatement && statement.ID == resolved.ID {
				return component, true
			}
		}
		for _, junctor := range component.Junctors {
			if resolved.Type == argument.ElementJunctor && junctor.ID == resolved.ID {
				return component, true
			}
		}
	}
	return Component{}, false
}

type disjointSet struct {
	parent map[string]string
	rank   map[string]int
}

func newDisjointSet(doc *argument.Document) *disjointSet {
	sets := &disjointSet{parent: make(map[string]string, len(doc.Statements)), rank: make(map[string]int, len(doc.Statements))}
	for _, statement := range doc.Statements {
		sets.parent[statement.ID] = statement.ID
	}
	return sets
}

func (s *disjointSet) find(id string) string {
	parent, exists := s.parent[id]
	if !exists || id == "" {
		return ""
	}
	if parent != id {
		s.parent[id] = s.find(parent)
	}
	return s.parent[id]
}

func (s *disjointSet) union(left, right string) {
	leftRoot, rightRoot := s.find(left), s.find(right)
	if leftRoot == "" || rightRoot == "" || leftRoot == rightRoot {
		return
	}
	if s.rank[leftRoot] < s.rank[rightRoot] {
		leftRoot, rightRoot = rightRoot, leftRoot
	}
	s.parent[rightRoot] = leftRoot
	if s.rank[leftRoot] == s.rank[rightRoot] {
		s.rank[leftRoot]++
	}
}
