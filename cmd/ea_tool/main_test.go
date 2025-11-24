package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteJSONCreatesFile(t *testing.T) {
	tmp := t.TempDir() + "/out.json"

	payload := JSONOutput{
		Version:       "test",
		Filename:      "file.bin",
		TestType:      "IID",
		BitsPerSymbol: 8,
		DataSize:      3,
		MinEntropy:    1.23,
		HAssessed:     1.23,
		ErrorCode:     0,
	}

	writeJSON(tmp, payload)

	raw, err := os.ReadFile(tmp)
	require.NoError(t, err)

	var got JSONOutput
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Equal(t, payload.Version, got.Version)
	assert.Equal(t, payload.MinEntropy, got.MinEntropy)
}

func TestRunCLI_Version(t *testing.T) {
	var out bytes.Buffer
	code := runCLI([]string{"-version"}, bytes.NewReader(nil), &out, &out)
	assert.Equal(t, 0, code)
	assert.Contains(t, out.String(), "ea_tool version")
}

func TestRunCLI_MissingTestType(t *testing.T) {
	var out bytes.Buffer
	code := runCLI([]string{}, bytes.NewReader(nil), &out, &out)
	assert.Equal(t, 2, code)
	assert.Contains(t, out.String(), "Must specify exactly one of -iid or -non-iid")
}

func TestRunCLI_InvalidBits(t *testing.T) {
	var out bytes.Buffer
	code := runCLI([]string{"-iid", "-bits", "9"}, bytes.NewReader(nil), &out, &out)
	assert.Equal(t, 2, code)
	assert.Contains(t, out.String(), "bits per symbol must be 0-8")
}

func TestRunCLI_FileNotFound(t *testing.T) {
	var out bytes.Buffer
	code := runCLI([]string{"-iid", "-bits", "8", "nope.bin"}, bytes.NewReader(nil), &out, &out)
	assert.Equal(t, 1, code)
	assert.Contains(t, out.String(), "Error reading file")
}

func TestRunCLI_StdinSuccessWithStub(t *testing.T) {
	var out bytes.Buffer
	data := []byte{1, 2, 3, 4}
	code := runCLI([]string{"-non-iid", "-bits", "8"}, bytes.NewReader(data), &out, &out)
	assert.Equal(t, 0, code)
	assert.Contains(t, out.String(), "Entropy Assessment Results")
}

func TestRunCLI_OutputFileSuccess(t *testing.T) {
	var out bytes.Buffer
	data := []byte{1, 2, 3, 4}
	tmpFile := filepath.Join(t.TempDir(), "result.json")

	code := runCLI([]string{"-non-iid", "-bits", "8", "-output", tmpFile}, bytes.NewReader(data), &out, &out)
	require.Equal(t, 0, code)
	assert.Contains(t, out.String(), "Results written to")

	raw, err := os.ReadFile(tmpFile)
	require.NoError(t, err)

	var got JSONOutput
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Equal(t, "Non-IID", got.TestType)
	assert.Equal(t, 0, got.ErrorCode)
	assert.Equal(t, len(data), got.DataSize)
}

func TestRunCLI_OutputFileErrorPath(t *testing.T) {
	var out bytes.Buffer
	tmpFile := filepath.Join(t.TempDir(), "error.json")

	code := runCLI([]string{"-non-iid", "-bits", "8", "-output", tmpFile}, bytes.NewReader(nil), &out, &out)
	require.Equal(t, 1, code)

	raw, err := os.ReadFile(tmpFile)
	require.NoError(t, err)

	var got JSONOutput
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Equal(t, 1, got.ErrorCode)
	assert.Contains(t, got.ErrorMessage, "data is empty")
}

func TestRunCLI_IIDModeSuccess(t *testing.T) {
	var out bytes.Buffer
	data := []byte{1, 2, 3, 4}

	code := runCLI([]string{"-iid", "-bits", "8"}, bytes.NewReader(data), &out, &out)
	require.Equal(t, 0, code)
	assert.Contains(t, out.String(), "Test Type:       IID")
	assert.Contains(t, out.String(), "H_bitstring")
}
