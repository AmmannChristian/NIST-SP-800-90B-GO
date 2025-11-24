package entropy

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEntropyError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *EntropyError
		want string
	}{
		{
			name: "with message",
			err: &EntropyError{
				Op:  "AssessIID",
				Err: ErrInvalidData,
				Msg: "empty dataset",
			},
			want: "AssessIID: empty dataset: invalid input data",
		},
		{
			name: "without message",
			err: &EntropyError{
				Op:  "AssessNonIID",
				Err: ErrInsufficientData,
				Msg: "",
			},
			want: "AssessNonIID: insufficient data for entropy assessment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEntropyError_Unwrap(t *testing.T) {
	baseErr := ErrInvalidBitsPerSymbol
	err := &EntropyError{
		Op:  "test",
		Err: baseErr,
		Msg: "test message",
	}

	unwrapped := errors.Unwrap(err)
	assert.Equal(t, baseErr, unwrapped)
}

func TestNewError(t *testing.T) {
	err := newError("TestOp", ErrCFunction, "test message")

	entropyErr, ok := err.(*EntropyError)
	assert.True(t, ok)
	assert.Equal(t, "TestOp", entropyErr.Op)
	assert.Equal(t, ErrCFunction, entropyErr.Err)
	assert.Equal(t, "test message", entropyErr.Msg)
}

func TestWrapCError(t *testing.T) {
	err := wrapCError("calculate_iid_entropy", -1, "memory allocation failed")

	entropyErr, ok := err.(*EntropyError)
	assert.True(t, ok)
	assert.Equal(t, "calculate_iid_entropy", entropyErr.Op)
	assert.Equal(t, ErrCFunction, entropyErr.Err)
	assert.Contains(t, entropyErr.Msg, "code=-1")
	assert.Contains(t, entropyErr.Msg, "memory allocation failed")
}

func TestPredefinedErrors(t *testing.T) {
	// Test that predefined errors exist and have correct messages
	assert.NotNil(t, ErrInvalidData)
	assert.Equal(t, "invalid input data", ErrInvalidData.Error())

	assert.NotNil(t, ErrInvalidBitsPerSymbol)
	assert.Equal(t, "bits_per_symbol must be between 1 and 8", ErrInvalidBitsPerSymbol.Error())

	assert.NotNil(t, ErrInsufficientData)
	assert.Equal(t, "insufficient data for entropy assessment", ErrInsufficientData.Error())

	assert.NotNil(t, ErrCFunction)
	assert.Equal(t, "c library function error", ErrCFunction.Error())

	assert.NotNil(t, ErrMemoryAllocation)
	assert.Equal(t, "memory allocation failed", ErrMemoryAllocation.Error())
}
