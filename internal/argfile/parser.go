// SPDX-FileCopyrightText: 2026 KeenWorks
// SPDX-License-Identifier: GPL-3.0-or-later

package argfile

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/tunesmith/cludia/internal/argument"
	"github.com/tunesmith/cludia/internal/diagnostic"
)

type ParseResult struct {
	Document    *argument.Document
	Diagnostics []diagnostic.Diagnostic
}

type sourceLine struct {
	text string
	line int
}

type rawStatement struct {
	statement  argument.Statement
	sourceRole string
	line       int
	defeat     string
	supports   []rawSupport
}

type rawSupport struct {
	connector  string
	label      string
	references []string
	line       int
	order      int
}

var (
	headerPattern             = regexp.MustCompile(`^argument\s+([^\s]+)\s+"([^"]+)"\s*$`)
	statementPattern          = regexp.MustCompile(`^(premise|lemma|conclusion|counterpoint|undermine|undercut|rejoinder)(?:\[(fact|value)\])?\s+([A-Za-z][A-Za-z0-9_-]*)(?::([a-z][a-z0-9-]*))?\s*(?:::(T|F|U))?\s+"(.*)"\s*(?:->\s*(.*))?$`)
	multilinePattern          = regexp.MustCompile(`^(premise|lemma|conclusion|counterpoint|undermine|undercut|rejoinder)(?:\[(fact|value)\])?\s+([A-Za-z][A-Za-z0-9_-]*)(?::([a-z][a-z0-9-]*))?\s*(?:::(T|F|U))?\s+"""\s*$`)
	statementHeaderPattern    = regexp.MustCompile(`^(premise|lemma|conclusion|counterpoint|undermine|undercut|rejoinder)(?:\[(fact|value)\])?\s+([A-Za-z][A-Za-z0-9_-]*)(?::([a-z][a-z0-9-]*))?\s*(?:::(T|F|U))?\s*$`)
	clausePattern             = regexp.MustCompile(`^([A-Z]+)(?:#([A-Za-z][A-Za-z0-9_-]*))?\(([^)]*)\)$`)
	metaEntryPattern          = regexp.MustCompile(`([A-Za-z0-9_-]+)\s*=\s*"([^"]*)"`)
	premiseDefeatPattern      = regexp.MustCompile(`^premise\s+([A-Za-z][A-Za-z0-9_-]*)$`)
	counterpointDefeatPattern = regexp.MustCompile(`^counterpoint\s+([A-Za-z][A-Za-z0-9_-]*)$`)
	inferenceDefeatPattern    = regexp.MustCompile(`^inference\s+([A-Za-z][A-Za-z0-9_-]*)(?::target\s+([A-Za-z][A-Za-z0-9_-]*))?$`)
)

func ParseFile(path string) ParseResult {
	data, err := os.ReadFile(path)
	if err != nil {
		return ParseResult{Diagnostics: []diagnostic.Diagnostic{{
			Code: "file_read", Message: err.Error(), Severity: diagnostic.SeverityError,
		}}}
	}
	return Parse(string(data))
}

func Parse(input string) ParseResult {
	rawLines := strings.Split(strings.ReplaceAll(input, "\r\n", "\n"), "\n")
	lines := make([]sourceLine, len(rawLines))
	for i, line := range rawLines {
		lines[i] = sourceLine{text: strings.TrimSuffix(line, "\r"), line: i + 1}
	}

	doc := &argument.Document{}
	result := ParseResult{Document: doc}
	index := skipIgnorable(lines, 0)
	if index >= len(lines) {
		result.addError("header_missing", 1, "empty argument source; expected header line", "")
		return result
	}

	header := headerPattern.FindStringSubmatch(strings.TrimSpace(lines[index].text))
	if header == nil {
		result.addError("header_invalid", lines[index].line, "expected `argument <id> \"Title\"` header", "")
		return result
	}
	doc.ID, doc.Title = header[1], header[2]
	index++

	index = skipIgnorable(lines, index)
	if index < len(lines) {
		trimmed := strings.TrimSpace(lines[index].text)
		if trimmed == "meta" || strings.HasPrefix(trimmed, "meta ") {
			doc.Metadata = parseMetadata(strings.TrimSpace(strings.TrimPrefix(trimmed, "meta")), lines[index].line, &result)
			index++
		}
	}

	var records []*rawStatement
	var last *rawStatement
	relationOrder := 0
	for index < len(lines) {
		line := lines[index]
		trimmed := strings.TrimSpace(line.text)
		if isIgnorable(trimmed) {
			index++
			continue
		}
		if strings.HasPrefix(trimmed, "<-") {
			if last == nil {
				result.addError("support_without_statement", line.line, "support clause appears before any statement", "")
			} else {
				supports := parseSupportLine(strings.TrimSpace(strings.TrimPrefix(trimmed, "<-")), line.line, &relationOrder, &result)
				last.supports = append(last.supports, supports...)
			}
			index++
			continue
		}

		if match := multilinePattern.FindStringSubmatch(trimmed); match != nil {
			text, next, ok := collectMultiline(lines, index+1)
			record := newRawStatement(match, text, line.line, "", &result)
			records = append(records, record)
			last = record
			if !ok {
				result.addError("statement_text_unterminated", line.line, "unterminated multi-line statement text; expected closing triple quote", match[3])
			}
			index = next
			continue
		}

		if match := statementHeaderPattern.FindStringSubmatch(trimmed); match != nil {
			opening := skipBlank(lines, index+1)
			if opening >= len(lines) || strings.TrimSpace(lines[opening].text) != `"""` {
				result.addError("statement_text_missing", line.line, "expected triple-quoted statement text on the next non-blank line", match[3])
				index++
				continue
			}
			text, next, ok := collectMultiline(lines, opening+1)
			record := newRawStatement(match, text, line.line, "", &result)
			records = append(records, record)
			last = record
			if !ok {
				result.addError("statement_text_unterminated", line.line, "unterminated multi-line statement text; expected closing triple quote", match[3])
			}
			index = next
			continue
		}

		if match := statementPattern.FindStringSubmatch(trimmed); match != nil {
			record := newRawStatement(match[:6], match[6], line.line, match[7], &result)
			records = append(records, record)
			last = record
			index++
			continue
		}

		result.addError("statement_invalid", line.line, fmt.Sprintf("unrecognized statement syntax: %q", line.text), "")
		index++
	}

	if len(records) == 0 {
		result.addError("statements_empty", 0, "no statements found; add at least one premise", doc.ID)
		return result
	}

	resolveRecords(doc, records, &result)
	return result
}

func newRawStatement(match []string, text string, line int, defeat string, result *ParseResult) *rawStatement {
	sourceRole := match[1]
	role := argument.Role(sourceRole)
	if sourceRole == "undermine" || sourceRole == "undercut" || sourceRole == "rejoinder" {
		role = argument.RoleCounterpoint
	}
	kind := argument.KindFact
	if match[2] != "" {
		kind = argument.Kind(match[2])
	}
	truth := argument.TruthUnknown
	if match[5] != "" {
		truth = argument.Truth(match[5])
	}
	if match[5] != "" && role != argument.RolePremise && role != argument.RoleCounterpoint {
		result.addError("truth_role_invalid", line, fmt.Sprintf("only premises and counterpoints may specify truth; remove ::%s", match[5]), match[3])
	}
	if defeat != "" && role != argument.RoleCounterpoint {
		result.addError("defeat_role_invalid", line, "only counterpoints may specify a defeat target", match[3])
	}
	return &rawStatement{
		statement:  argument.Statement{ID: match[3], Slug: match[4], Role: role, Kind: kind, Truth: truth, Text: text},
		sourceRole: sourceRole, line: line, defeat: strings.TrimSpace(defeat),
	}
}

func parseMetadata(raw string, line int, result *ParseResult) []argument.Metadata {
	if raw == "" {
		return nil
	}
	matches := metaEntryPattern.FindAllStringSubmatchIndex(raw, -1)
	if len(matches) == 0 {
		result.addError("metadata_invalid", line, "metadata must use key=\"value\" entries", "")
		return nil
	}
	entries := make([]argument.Metadata, 0, len(matches))
	position := 0
	for i, match := range matches {
		separator := strings.TrimSpace(raw[position:match[0]])
		if i == 0 && separator != "" || i > 0 && separator != "," {
			result.addError("metadata_invalid", line, fmt.Sprintf("unparsed metadata fragment %q", strings.TrimSpace(raw[position:match[0]])), "")
		}
		entries = append(entries, argument.Metadata{Key: raw[match[2]:match[3]], Value: raw[match[4]:match[5]]})
		position = match[1]
	}
	if strings.TrimSpace(raw[position:]) != "" {
		result.addError("metadata_invalid", line, fmt.Sprintf("unparsed metadata fragment %q", strings.TrimSpace(raw[position:])), "")
	}
	return entries
}

func parseSupportLine(raw string, line int, order *int, result *ParseResult) []rawSupport {
	parts := strings.Split(raw, ";")
	supports := make([]rawSupport, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		match := clausePattern.FindStringSubmatch(part)
		if match == nil {
			result.addError("support_invalid", line, fmt.Sprintf("unable to parse support clause %q", part), "")
			continue
		}
		var references []string
		for _, reference := range strings.Split(match[3], ",") {
			if reference = strings.TrimSpace(reference); reference != "" {
				references = append(references, reference)
			}
		}
		if len(references) == 0 {
			result.addError("support_sources_empty", line, fmt.Sprintf("connector %s requires at least one reference", match[1]), match[2])
			continue
		}
		*order++
		supports = append(supports, rawSupport{connector: match[1], label: match[2], references: references, line: line, order: *order})
	}
	return supports
}

func resolveRecords(doc *argument.Document, records []*rawStatement, result *ParseResult) {
	idLines := make(map[string]int, len(records))
	slugs := make(map[string]string, len(records))
	for _, record := range records {
		statement := record.statement
		if previous, exists := idLines[statement.ID]; exists {
			result.addError("statement_id_duplicate", record.line, fmt.Sprintf("duplicate statement id %q (first defined at line %d)", statement.ID, previous), statement.ID)
		} else {
			idLines[statement.ID] = record.line
		}
		if statement.Slug != "" {
			if owner, exists := slugs[statement.Slug]; exists {
				result.addError("statement_slug_duplicate", record.line, fmt.Sprintf("duplicate slug %q already used by %s", statement.Slug, owner), statement.ID)
			} else {
				slugs[statement.Slug] = statement.ID
			}
		}
		doc.Statements = append(doc.Statements, statement)
	}

	resolveStatement := func(reference string, line int, context string) string {
		if statement, exists := doc.Statement(reference); exists {
			return statement.ID
		}
		result.addError("reference_unknown", line, fmt.Sprintf("reference %q in %s does not match any statement id or slug", reference, context), reference)
		return reference
	}

	explicitLabels := make(map[string]int)
	for _, record := range records {
		for _, support := range record.supports {
			if support.label == "" {
				continue
			}
			if prior, exists := explicitLabels[support.label]; exists {
				result.addError("junctor_id_duplicate", support.line, fmt.Sprintf("support label %q already defined at line %d", support.label, prior), support.label)
			} else {
				explicitLabels[support.label] = support.line
			}
		}
	}
	usedJunctors := make(map[string]bool, len(explicitLabels))
	reservedIDs := make(map[string]bool, len(explicitLabels)+len(idLines))
	for label := range explicitLabels {
		usedJunctors[label] = true
		reservedIDs[label] = true
	}
	for id := range idLines {
		reservedIDs[id] = true
	}
	labelsToIDs := make(map[string]string, len(explicitLabels))
	nextJunctor := 1
	for _, record := range records {
		for _, support := range record.supports {
			sources := make([]string, 0, len(support.references))
			for _, reference := range support.references {
				sources = append(sources, resolveStatement(reference, support.line, "support"))
			}
			connector := argument.Connector(support.connector)
			if len(sources) == 1 {
				if support.label != "" {
					result.addError("direct_support_labeled", support.line, "a one-source direct support cannot have a junctor label", support.label)
				}
				doc.DirectSupports = append(doc.DirectSupports, argument.DirectSupport{Source: sources[0], Target: record.statement.ID, Connector: connector, Order: support.order})
				continue
			}
			id := support.label
			if id == "" {
				for {
					id = fmt.Sprintf("J%d", nextJunctor)
					nextJunctor++
					if !reservedIDs[id] {
						break
					}
				}
			}
			usedJunctors[id] = true
			reservedIDs[id] = true
			if support.label != "" {
				labelsToIDs[support.label] = id
			}
			doc.Junctors = append(doc.Junctors, argument.Junctor{ID: id, Connector: connector, Sources: sources, Target: record.statement.ID, Order: support.order})
		}
	}

	for _, record := range records {
		if record.defeat == "" {
			continue
		}
		defeat, ok := parseDefeat(record, labelsToIDs, usedJunctors, resolveStatement, result)
		if ok {
			doc.Defeats = append(doc.Defeats, defeat)
		}
	}
}

func parseDefeat(record *rawStatement, labels map[string]string, junctors map[string]bool, resolveStatement func(string, int, string) string, result *ParseResult) (argument.Defeat, bool) {
	if match := premiseDefeatPattern.FindStringSubmatch(record.defeat); match != nil {
		return argument.Defeat{From: record.statement.ID, Scope: argument.DefeatPremise, To: resolveStatement(match[1], record.line, "premise defeat")}, true
	}
	if match := counterpointDefeatPattern.FindStringSubmatch(record.defeat); match != nil {
		return argument.Defeat{From: record.statement.ID, Scope: argument.DefeatCounterpoint, To: resolveStatement(match[1], record.line, "counterpoint defeat")}, true
	}
	if match := inferenceDefeatPattern.FindStringSubmatch(record.defeat); match != nil {
		if match[2] == "" {
			result.addError("defeat_inference_target_missing", record.line, "inference defeat must specify `:target <statement>`", record.statement.ID)
			return argument.Defeat{}, false
		}
		junctorID := match[1]
		if id, exists := labels[junctorID]; exists {
			junctorID = id
		} else if !junctors[junctorID] {
			result.addError("defeat_junctor_unknown", record.line, fmt.Sprintf("inference defeat references unknown junctor %q", match[1]), record.statement.ID)
		}
		return argument.Defeat{From: record.statement.ID, Scope: argument.DefeatInference, JunctorID: junctorID, AtTarget: resolveStatement(match[2], record.line, "inference defeat")}, true
	}
	result.addError("defeat_target_invalid", record.line, fmt.Sprintf("unable to parse counterpoint target %q", record.defeat), record.statement.ID)
	return argument.Defeat{}, false
}

func collectMultiline(lines []sourceLine, start int) (string, int, bool) {
	var collected []string
	for index := start; index < len(lines); index++ {
		if strings.TrimSpace(lines[index].text) == `"""` {
			return dedent(collected), index + 1, true
		}
		collected = append(collected, lines[index].text)
	}
	return dedent(collected), len(lines), false
}

func dedent(lines []string) string {
	minimum := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if minimum < 0 || indent < minimum {
			minimum = indent
		}
	}
	if minimum < 0 {
		minimum = 0
	}
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			lines[i] = ""
		} else if len(line) >= minimum {
			lines[i] = line[minimum:]
		}
	}
	return strings.Join(lines, "\n")
}

func skipBlank(lines []sourceLine, index int) int {
	for index < len(lines) && strings.TrimSpace(lines[index].text) == "" {
		index++
	}
	return index
}

func skipIgnorable(lines []sourceLine, index int) int {
	for index < len(lines) && isIgnorable(strings.TrimSpace(lines[index].text)) {
		index++
	}
	return index
}

func isIgnorable(trimmed string) bool {
	return trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//")
}

func (r *ParseResult) addError(code string, line int, message, element string) {
	r.Diagnostics = append(r.Diagnostics, diagnostic.Diagnostic{Code: code, Message: message, Severity: diagnostic.SeverityError, Line: line, Element: element})
}
