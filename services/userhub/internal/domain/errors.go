package domain

import "fmt"

// ErrorKind identifies a category of domain errors and provides its message prefix.
type ErrorKind string

func (kind ErrorKind) Error() string {
	return string(kind)
}

// NewError creates a domain error of this kind.
func (kind ErrorKind) NewError(message string) *Error {
	return &Error{
		Kind:    kind,
		Message: message,
	}
}

// NewErrorf creates a domain error of this kind with a formatted message.
func (kind ErrorKind) NewErrorf(format string, args ...any) *Error {
	return kind.NewError(fmt.Sprintf(format, args...))
}

// WrapError creates a domain error of this kind that retains the original error.
func (kind ErrorKind) WrapError(err error, message string) *Error {
	return &Error{
		Kind:    kind,
		Message: message,
		Err:     err,
	}
}

// WrapErrorf creates a domain error of this kind with a formatted message and retains the original error.
func (kind ErrorKind) WrapErrorf(err error, format string, args ...any) *Error {
	return kind.WrapError(err, fmt.Sprintf(format, args...))
}

// Error represents a categorized domain error.
type Error struct {
	Kind    ErrorKind
	Message string
	Err     error
}

func (e *Error) Error() string {
	switch {
	case e.Kind != "" && e.Message != "" && e.Err != nil:
		return fmt.Sprintf("%s: %s: %v", e.Kind, e.Message, e.Err)

	case e.Kind != "" && e.Message != "":
		return fmt.Sprintf("%s: %s", e.Kind, e.Message)

	case e.Kind != "" && e.Err != nil:
		return fmt.Sprintf("%s: %v", e.Kind, e.Err)

	case e.Kind != "":
		return e.Kind.Error()

	case e.Message != "" && e.Err != nil:
		return fmt.Sprintf("%s: %v", e.Message, e.Err)

	case e.Message != "":
		return e.Message

	case e.Err != nil:
		return e.Err.Error()

	default:
		return ""
	}
}

func (e *Error) Unwrap() error {
	return e.Err
}

// Is reports whether the target error has the same error kind.
func (e *Error) Is(target error) bool {
	if e.Kind == "" {
		return false
	}

	kind, ok := target.(ErrorKind)
	return ok && kind != "" && e.Kind == kind
}

// NewError creates a domain error without a specific kind.
func NewError(message string) *Error {
	return &Error{Message: message}
}

// NewErrorf creates a domain error without a specific kind using a formatted message.
func NewErrorf(format string, args ...any) *Error {
	return NewError(fmt.Sprintf(format, args...))
}

// WrapError creates a domain error without a specific kind that retains the original error.
func WrapError(err error, message string) *Error {
	return &Error{Message: message, Err: err}
}

// WrapErrorf creates a domain error without a specific kind using a formatted message and retains the original error.
func WrapErrorf(err error, format string, args ...any) *Error {
	return WrapError(err, fmt.Sprintf(format, args...))
}
