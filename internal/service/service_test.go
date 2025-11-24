package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NOTE: Success tests with actual CGO calls are skipped because the NIST C++ library
// requires at least 1 million samples and crashes with smaller datasets.
// These tests would take several minutes to run with proper data sizes.

func TestNewService(t *testing.T) {
	svc := NewService()

	assert.NotNil(t, svc)
	assert.NotNil(t, svc.assessment)
}

func TestService_SetVerbose(t *testing.T) {
	svc := NewService()

	svc.SetVerbose(2)
	assert.Equal(t, 2, svc.assessment.GetVerbose())

	svc.SetVerbose(0)
	assert.Equal(t, 0, svc.assessment.GetVerbose())
}

func TestService_AssessIID_ValidationErrors(t *testing.T) {
	svc := NewService()

	// Empty data
	_, err := svc.AssessIID([]byte{}, 8)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")

	// Invalid bits_per_symbol - too low
	_, err = svc.AssessIID([]byte{1, 2, 3}, -1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bits_per_symbol")

	// Invalid bits_per_symbol - too high
	_, err = svc.AssessIID([]byte{1, 2, 3}, 9)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bits_per_symbol")
}

func TestService_AssessNonIID_ValidationErrors(t *testing.T) {
	svc := NewService()

	// Empty data
	_, err := svc.AssessNonIID([]byte{}, 8)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")

	// Invalid bits_per_symbol - too low
	_, err = svc.AssessNonIID([]byte{1, 2, 3}, -1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bits_per_symbol")

	// Invalid bits_per_symbol - too high
	_, err = svc.AssessNonIID([]byte{1, 2, 3}, 9)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bits_per_symbol")
}

// Success paths rely on the teststub build tag to avoid CGO.
func TestService_AssessIID_SuccessStub(t *testing.T) {
	svc := NewService()
	res, err := svc.AssessIID([]byte{1, 2, 3, 4}, 8)
	require.NoError(t, err)
	assert.Equal(t, 7.5, res.MinEntropy)
}

func TestService_AssessNonIID_SuccessStub(t *testing.T) {
	svc := NewService()
	res, err := svc.AssessNonIID([]byte{1, 2, 3, 4}, 8)
	require.NoError(t, err)
	assert.Equal(t, 6.5, res.MinEntropy)
}

func TestService_AssessIID_AssessmentError(t *testing.T) {
	svc := NewService()

	_, err := svc.AssessIID([]byte{0xFF, 1, 2, 3}, 8)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "IID assessment failed")
}

func TestService_AssessNonIID_AssessmentError(t *testing.T) {
	svc := NewService()

	_, err := svc.AssessNonIID([]byte{0xFF, 1, 2, 3}, 8)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Non-IID assessment failed")
}
