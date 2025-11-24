package entropy

import (
	"errors"
	"fmt"
)

var (
	// ErrInvalidData indicates the input data is invalid
	ErrInvalidData = errors.New("invalid input data")

	// ErrInvalidBitsPerSymbol indicates bits_per_symbol is not in valid range [1,8]
	ErrInvalidBitsPerSymbol = errors.New("bits_per_symbol must be between 1 and 8")

	// ErrInsufficientData indicates not enough data for reliable entropy estimate
	ErrInsufficientData = errors.New("insufficient data for entropy assessment")

	// ErrCFunction indicates an error occurred in the C library
	ErrCFunction = errors.New("c library function error")

	// ErrMemoryAllocation indicates a memory allocation failure
	ErrMemoryAllocation = errors.New("memory allocation failed")
)

// EntropyError wraps errors from the entropy assessment with additional context
type EntropyError struct {
	Op  string // Operation that failed
	Err error  // Underlying error
	Msg string // Additional message
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

// newError creates a new EntropyError
func newError(op string, err error, msg string) error {
	return &EntropyError{
		Op:  op,
		Err: err,
		Msg: msg,
	}
}

// wrapCError wraps a C library error with context
func wrapCError(op string, code int, message string) error {
	return newError(op, ErrCFunction, fmt.Sprintf("code=%d, message=%s", code, message))
}
