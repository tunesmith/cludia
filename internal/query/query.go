// SPDX-FileCopyrightText: 2026 KeenWorks
// SPDX-License-Identifier: GPL-3.0-or-later

package query

import "github.com/tunesmith/cludia/internal/argument"

type Support struct {
	Type      string             `json:"type"`
	ID        string             `json:"id,omitempty"`
	Connector argument.Connector `json:"connector"`
	Sources   []string           `json:"sources"`
	Target    string             `json:"target"`
}

type Relations struct {
	IncomingSupport    []Support         `json:"incoming_support"`
	OutgoingSupport    []Support         `json:"outgoing_support"`
	DefeatsTargeting   []argument.Defeat `json:"defeats_targeting"`
	DefeatsOriginating []argument.Defeat `json:"defeats_originating"`
}

func IsolatedStatementIDs(doc *argument.Document) map[string]bool {
	connected := make(map[string]bool, len(doc.Statements))
	for _, junctor := range doc.Junctors {
		connected[junctor.Target] = true
		for _, source := range junctor.Sources {
			connected[source] = true
		}
	}
	for _, support := range doc.DirectSupports {
		connected[support.Source] = true
		connected[support.Target] = true
	}
	for _, defeat := range doc.Defeats {
		connected[defeat.From] = true
		if defeat.To != "" {
			connected[defeat.To] = true
		}
		if defeat.AtTarget != "" {
			connected[defeat.AtTarget] = true
		}
	}
	isolated := make(map[string]bool)
	for _, statement := range doc.Statements {
		if !connected[statement.ID] {
			isolated[statement.ID] = true
		}
	}
	return isolated
}

func StatementRelations(doc *argument.Document, id string) Relations {
	result := emptyRelations()
	for _, junctor := range doc.Junctors {
		support := junctorSupport(junctor)
		if junctor.Target == id {
			result.IncomingSupport = append(result.IncomingSupport, support)
		}
		for _, source := range junctor.Sources {
			if source == id {
				result.OutgoingSupport = append(result.OutgoingSupport, support)
				break
			}
		}
	}
	for _, direct := range doc.DirectSupports {
		support := directSupport(direct)
		if direct.Target == id {
			result.IncomingSupport = append(result.IncomingSupport, support)
		}
		if direct.Source == id {
			result.OutgoingSupport = append(result.OutgoingSupport, support)
		}
	}
	for _, defeat := range doc.Defeats {
		if defeat.From == id {
			result.DefeatsOriginating = append(result.DefeatsOriginating, defeat)
		}
		if defeat.To == id || defeat.AtTarget == id {
			result.DefeatsTargeting = append(result.DefeatsTargeting, defeat)
		}
	}
	return result
}

func JunctorRelations(doc *argument.Document, id string) Relations {
	result := emptyRelations()
	for _, defeat := range doc.Defeats {
		if defeat.Scope == argument.DefeatInference && defeat.JunctorID == id {
			result.DefeatsTargeting = append(result.DefeatsTargeting, defeat)
		}
	}
	return result
}

func emptyRelations() Relations {
	return Relations{
		IncomingSupport: []Support{}, OutgoingSupport: []Support{},
		DefeatsTargeting: []argument.Defeat{}, DefeatsOriginating: []argument.Defeat{},
	}
}

func junctorSupport(junctor argument.Junctor) Support {
	return Support{Type: "junctor", ID: junctor.ID, Connector: junctor.Connector, Sources: append([]string(nil), junctor.Sources...), Target: junctor.Target}
}

func directSupport(support argument.DirectSupport) Support {
	return Support{Type: "direct", Connector: support.Connector, Sources: []string{support.Source}, Target: support.Target}
}
