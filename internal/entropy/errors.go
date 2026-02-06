package entropy

import (
	"errors"
	"fmt"
)

// Sentinel errors for entropy assessment failures. Use errors.Is to match
// against these when inspecting an EntropyError.
var (
	ErrInvalidData          = errors.New("invalid input data")
	ErrInvalidBitsPerSymbol = errors.New("bits_per_symbol must be between 1 and 8")
	ErrInsufficientData     = errors.New("insufficient data for entropy assessment")
	ErrCFunction            = errors.New("c library function error")
	ErrMemoryAllocation     = errors.New("memory allocation failed")
)

// EntropyError provides structured error context for entropy assessment failures.
// It records the operation name, the underlying cause, and an optional message.
// It implements the error and Unwrap interfaces.
type EntropyError struct {
	Op  string // Operation that failed
	Err error  // Underlying (sentinel) error
	Msg string // Additional context
}

func (e *EntropyError) Error() string {
	if e.Msg != "" {
		return fmt.Sprintf("%s: %s: %v", e.Op, e.Msg, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

func (e *EntropyError) Unwrap() error {
	return e.Err
}

// newError creates a new EntropyError with the given operation, sentinel, and message.
func newError(op string, err error, msg string) error {
	return &EntropyError{
		Op:  op,
		Err: err,
		Msg: msg,
	}
}

// wrapCError wraps a C library error code and message into an EntropyError
// with ErrCFunction as the underlying sentinel.
func wrapCError(op string, code int, message string) error {
	return newError(op, ErrCFunction, fmt.Sprintf("code=%d, message=%s", code, message))
}
