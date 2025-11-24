package entropy

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NOTE: Success tests with actual CGO calls are skipped because the NIST C++ library
// requires at least 1 million samples and crashes with smaller datasets.
// These tests would take several minutes to run with proper data sizes.

// errorReader is a mock reader that always returns an error
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

func TestAssessFile_SuccessStub(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "data.bin")
	require.NoError(t, os.WriteFile(file, []byte{1, 2, 3, 4}, 0o644))

	assessment := NewAssessment()
	assessment.SetVerbose(0)

	res, err := assessment.AssessFile(file, 8, IID)
	require.NoError(t, err)
	assert.Equal(t, IID, res.TestType)
	assert.Equal(t, 8, res.DataWordSize)
	assert.Equal(t, 7.5, res.MinEntropy)
}

func TestAssessReader_SuccessPaths(t *testing.T) {
	data := []byte{1, 2, 3, 4}
	assessment := NewAssessment()
	assessment.SetVerbose(0)

	iidRes, err := assessment.AssessReader(bytes.NewReader(data), 8, IID)
	require.NoError(t, err)
	assert.Equal(t, IID, iidRes.TestType)

	nonIIDRes, err := assessment.AssessReader(bytes.NewReader(data), 8, NonIID)
	require.NoError(t, err)
	assert.Equal(t, NonIID, nonIIDRes.TestType)
}

func TestAssessIID_WarnsOnSmallDataset(t *testing.T) {
	assessment := NewAssessment()
	assessment.SetVerbose(1)

	output := captureStderr(t, func() {
		res, err := assessment.AssessIID([]byte{1, 2, 3, 4}, 8)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	assert.Contains(t, output, "Warning: data contains less than")
}

func TestAssessNonIID_WarnsOnSmallDataset(t *testing.T) {
	assessment := NewAssessment()
	assessment.SetVerbose(1)

	output := captureStderr(t, func() {
		res, err := assessment.AssessNonIID([]byte{1, 2, 3, 4}, 8)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	assert.Contains(t, output, "Warning: data contains less than")
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)

	os.Stderr = w
	fn()

	require.NoError(t, w.Close())
	os.Stderr = orig

	out, readErr := io.ReadAll(r)
	require.NoError(t, readErr)
	return string(out)
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

// The following success tests rely on the teststub build tag to avoid CGO calls.
func TestAssessIID_SuccessStub(t *testing.T) {
	assessment := NewAssessment()
	assessment.SetVerbose(0)

	res, err := assessment.AssessIID([]byte{1, 2, 3, 4}, 8)
	require.NoError(t, err)
	assert.Equal(t, 7.5, res.MinEntropy)
	assert.Equal(t, IID, res.TestType)
}

func TestAssessNonIID_SuccessStub(t *testing.T) {
	assessment := NewAssessment()
	assessment.SetVerbose(0)

	res, err := assessment.AssessNonIID([]byte{1, 2, 3, 4}, 8)
	require.NoError(t, err)
	assert.Equal(t, 6.5, res.MinEntropy)
	assert.Equal(t, NonIID, res.TestType)
}
