package entropy

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NOTE: Success tests with actual CGO calls are skipped because the NIST C++ library
// requires at least 1 million samples and crashes with smaller datasets.
// These tests would take several minutes to run with proper data sizes.

// errorReader is a mock io.Reader that always returns an error on Read.
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("mock read error")
}

func TestAssessFile_FileNotFound(t *testing.T) {
	assessment := NewAssessment()

	_, err := assessment.AssessFile("/nonexistent/file.bin", 8, IID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open file")
}

func TestAssessReader_ReadError(t *testing.T) {
	assessment := NewAssessment()
	reader := &errorReader{}

	_, err := assessment.AssessReader(reader, 8, IID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read data")
}

func TestAssessReader_InvalidTestType(t *testing.T) {
	testData := []byte{1, 2, 3, 4, 5}
	reader := bytes.NewReader(testData)
	assessment := NewAssessment()

	_, err := assessment.AssessReader(reader, 8, TestType(99))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid test type")
}

func TestAssessIID_InvalidBitsPerSymbol(t *testing.T) {
	testData := []byte{1, 2, 3, 4, 5}
	assessment := NewAssessment()

	// Test bits_per_symbol < 0
	_, err := assessment.AssessIID(testData, -1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bits_per_symbol")

	// Test bits_per_symbol > 8
	_, err = assessment.AssessIID(testData, 9)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bits_per_symbol")
}

func TestAssessIID_EmptyData(t *testing.T) {
	assessment := NewAssessment()

	_, err := assessment.AssessIID([]byte{}, 8)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "data is empty")
}

func TestAssessNonIID_InvalidBitsPerSymbol(t *testing.T) {
	testData := []byte{1, 2, 3, 4, 5}
	assessment := NewAssessment()

	// Test bits_per_symbol < 0
	_, err := assessment.AssessNonIID(testData, -1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bits_per_symbol")

	// Test bits_per_symbol > 8
	_, err = assessment.AssessNonIID(testData, 9)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bits_per_symbol")
}

func TestAssessNonIID_EmptyData(t *testing.T) {
	assessment := NewAssessment()

	_, err := assessment.AssessNonIID([]byte{}, 8)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "data is empty")
}
