package argument

import "encoding/json"

// MutationError is a stable semantic operation failure suitable for
// interface-specific diagnostics.
type MutationError struct {
	Code    string
	Message string
	Element string
}

func (e *MutationError) Error() string { return e.Message }

func mutationError(code, message, element string) *MutationError {
	return &MutationError{Code: code, Message: message, Element: element}
}

// MutationErrors reports independent semantic input failures discovered after
// resolving the operation's canonical target and tentative generated ID.
type MutationErrors struct {
	Failures []MutationError
}

func (e *MutationErrors) Error() string {
	if len(e.Failures) == 0 {
		return "mutation failed"
	}
	return e.Failures[0].Message
}

func marshalDocumentState(doc *Document) ([]byte, error) {
	junctorOrders := make([]int, len(doc.Junctors))
	for index, junctor := range doc.Junctors {
		junctorOrders[index] = junctor.Order
	}
	directSupportOrders := make([]int, len(doc.DirectSupports))
	for index, support := range doc.DirectSupports {
		directSupportOrders[index] = support.Order
	}
	return json.Marshal(struct {
		Document            *Document `json:"document"`
		JunctorOrders       []int     `json:"junctor_orders"`
		DirectSupportOrders []int     `json:"direct_support_orders"`
	}{Document: doc, JunctorOrders: junctorOrders, DirectSupportOrders: directSupportOrders})
}
