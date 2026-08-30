package argument

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
