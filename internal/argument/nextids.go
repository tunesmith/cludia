package argument

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const NextIDsMetadataKey = "cludia-next-ids"

var canonicalNumericIDPattern = regexp.MustCompile(`^(CP|P|L|C|J)([1-9][0-9]*)$`)

// NextIDs records the exact next numeric suffix in each focused namespace.
type NextIDs struct {
	P  int `json:"P"`
	L  int `json:"L"`
	C  int `json:"C"`
	CP int `json:"CP"`
	J  int `json:"J"`
}

func DefaultNextIDs() NextIDs {
	return NextIDs{P: 1, L: 1, C: 1, CP: 1, J: 1}
}

func (ids NextIDs) MetadataValue() string {
	return fmt.Sprintf("v1;P=%d;L=%d;C=%d;CP=%d;J=%d", ids.P, ids.L, ids.C, ids.CP, ids.J)
}

func ParseNextIDs(value string) (NextIDs, error) {
	parts := strings.Split(value, ";")
	if len(parts) != 6 || parts[0] != "v1" {
		return NextIDs{}, fmt.Errorf("%s must use v1 with P, L, C, CP, and J values", NextIDsMetadataKey)
	}
	result := NextIDs{}
	seen := map[string]bool{}
	for _, part := range parts[1:] {
		name, raw, ok := strings.Cut(part, "=")
		if !ok || seen[name] || (len(raw) > 1 && raw[0] == '0') {
			return NextIDs{}, fmt.Errorf("%s contains invalid entry %q", NextIDsMetadataKey, part)
		}
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 {
			return NextIDs{}, fmt.Errorf("%s value %q must be a positive decimal integer", name, raw)
		}
		seen[name] = true
		switch name {
		case "P":
			result.P = value
		case "L":
			result.L = value
		case "C":
			result.C = value
		case "CP":
			result.CP = value
		case "J":
			result.J = value
		default:
			return NextIDs{}, fmt.Errorf("%s contains unknown namespace %q", NextIDsMetadataKey, name)
		}
	}
	if len(seen) != 5 || result.P == 0 || result.L == 0 || result.C == 0 || result.CP == 0 || result.J == 0 {
		return NextIDs{}, fmt.Errorf("%s must define P, L, C, CP, and J exactly once", NextIDsMetadataKey)
	}
	return result, nil
}

// InspectNextIDs returns stored, observed, and safe effective allocator state.
func InspectNextIDs(doc *Document) (stored, observed, effective NextIDs, present bool, stale []string, err error) {
	stored = DefaultNextIDs()
	observed, err = observedNextIDs(doc)
	if err != nil {
		return NextIDs{}, NextIDs{}, NextIDs{}, false, nil, err
	}
	metadataCount := 0
	if doc != nil {
		for _, entry := range doc.Metadata {
			if entry.Key == NextIDsMetadataKey {
				metadataCount++
				stored, err = ParseNextIDs(entry.Value)
				if err != nil {
					return NextIDs{}, observed, NextIDs{}, true, nil, err
				}
			}
		}
	}
	if metadataCount > 1 {
		return NextIDs{}, observed, NextIDs{}, true, nil, fmt.Errorf("%s metadata appears more than once", NextIDsMetadataKey)
	}
	present = metadataCount == 1
	effective = stored
	for _, namespace := range []string{"P", "L", "C", "CP", "J"} {
		if observed.value(namespace) > effective.value(namespace) {
			if present {
				stale = append(stale, namespace)
			}
			effective.set(namespace, observed.value(namespace))
		}
	}
	return stored, observed, effective, present, stale, nil
}

func observedNextIDs(doc *Document) (NextIDs, error) {
	result := DefaultNextIDs()
	if doc == nil {
		return result, nil
	}
	observe := func(id string) error {
		match := canonicalNumericIDPattern.FindStringSubmatch(id)
		if match == nil {
			return nil
		}
		value, err := strconv.Atoi(match[2])
		if err != nil || value == int(^uint(0)>>1) {
			return fmt.Errorf("canonical id %q exceeds the supported numeric range", id)
		}
		next := value + 1
		if next > result.value(match[1]) {
			result.set(match[1], next)
		}
		return nil
	}
	for _, statement := range doc.Statements {
		if err := observe(statement.ID); err != nil {
			return NextIDs{}, err
		}
	}
	for _, junctor := range doc.Junctors {
		if err := observe(junctor.ID); err != nil {
			return NextIDs{}, err
		}
	}
	return result, nil
}

func (ids NextIDs) value(namespace string) int {
	switch namespace {
	case "P":
		return ids.P
	case "L":
		return ids.L
	case "C":
		return ids.C
	case "CP":
		return ids.CP
	case "J":
		return ids.J
	default:
		return 0
	}
}

func (ids *NextIDs) set(namespace string, value int) {
	switch namespace {
	case "P":
		ids.P = value
	case "L":
		ids.L = value
	case "C":
		ids.C = value
	case "CP":
		ids.CP = value
	case "J":
		ids.J = value
	}
}

func namespaceForRole(role Role) (string, bool) {
	switch role {
	case RolePremise:
		return "P", true
	case RoleLemma:
		return "L", true
	case RoleConclusion:
		return "C", true
	case RoleCounterpoint:
		return "CP", true
	default:
		return "", false
	}
}

// IDAllocationError is a stable focused-authoring failure.
type IDAllocationError struct {
	Code    string
	Message string
	Element string
}

func (e *IDAllocationError) Error() string { return e.Message }

// IDAllocator advances temporary next-ID state and persists it only on request.
type IDAllocator struct {
	next NextIDs
}

func NewIDAllocator(doc *Document) (*IDAllocator, error) {
	_, _, effective, _, _, err := InspectNextIDs(doc)
	if err != nil {
		return nil, err
	}
	return &IDAllocator{next: effective}, nil
}

func (a *IDAllocator) NextIDs() NextIDs { return a.next }

func (a *IDAllocator) Statement(role Role, requested string) (string, error) {
	namespace, ok := namespaceForRole(role)
	if !ok {
		return "", &IDAllocationError{Code: "statement_role_invalid", Message: fmt.Sprintf("role %q has no focused ID namespace", role), Element: requested}
	}
	return a.allocate(namespace, requested, "statement")
}

func (a *IDAllocator) Junctor(requested string) (string, error) {
	return a.allocate("J", requested, "junctor")
}

func (a *IDAllocator) allocate(namespace, requested, elementType string) (string, error) {
	expected := fmt.Sprintf("%s%d", namespace, a.next.value(namespace))
	requested = strings.TrimSpace(requested)
	if requested == "" {
		requested = expected
	}
	if requested != expected {
		match := canonicalNumericIDPattern.FindStringSubmatch(requested)
		if match == nil || match[1] != namespace {
			return "", &IDAllocationError{
				Code:    fmt.Sprintf("%s_id_not_canonical", elementType),
				Message: fmt.Sprintf("focused %s IDs must use the %s namespace; expected %s", elementType, namespace, expected),
				Element: requested,
			}
		}
		return "", &IDAllocationError{
			Code:    "id_not_next",
			Message: fmt.Sprintf("requested id %s is not the exact next %s id; expected %s", requested, namespace, expected),
			Element: requested,
		}
	}
	a.next.set(namespace, a.next.value(namespace)+1)
	return requested, nil
}

func (a *IDAllocator) Persist(doc *Document) {
	SetNextIDs(doc, a.next)
}

func SetNextIDs(doc *Document, ids NextIDs) {
	for i := range doc.Metadata {
		if doc.Metadata[i].Key == NextIDsMetadataKey {
			doc.Metadata[i].Value = ids.MetadataValue()
			return
		}
	}
	doc.Metadata = append(doc.Metadata, Metadata{Key: NextIDsMetadataKey, Value: ids.MetadataValue()})
}

// EnsureNextIDs persists effective state before an ID-deleting mutation.
func EnsureNextIDs(doc *Document) error {
	allocator, err := NewIDAllocator(doc)
	if err != nil {
		return err
	}
	allocator.Persist(doc)
	return nil
}
