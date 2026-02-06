package entropy

import (
	"fmt"
	"io"
	"os"
)

const (
	// MinRecommendedSamples is the minimum number of samples recommended by
	// NIST SP 800-90B for reliable entropy estimation (1,000,000).
	MinRecommendedSamples = 1000000
)

// AssessFile reads a binary file from disk and delegates to AssessReader for
// entropy assessment using the specified test type and bits-per-symbol value.
func (a *Assessment) AssessFile(filename string, bitsPerSymbol int, testType TestType) (*Result, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, newError("AssessFile", err, fmt.Sprintf("failed to open file: %s", filename))
	}
	defer file.Close()

	return a.AssessReader(file, bitsPerSymbol, testType)
}

// AssessReader reads all data from the provided io.Reader and dispatches to
// AssessIID or AssessNonIID based on the given test type.
func (a *Assessment) AssessReader(r io.Reader, bitsPerSymbol int, testType TestType) (*Result, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, newError("AssessReader", err, "failed to read data")
	}

	switch testType {
	case IID:
		return a.AssessIID(data, bitsPerSymbol)
	case NonIID:
		return a.AssessNonIID(data, bitsPerSymbol)
	default:
		return nil, newError("AssessReader", ErrInvalidData, "invalid test type")
	}
}

// AssessIID performs an IID (Independent and Identically Distributed) entropy
// assessment. A bitsPerSymbol value of 0 triggers auto-detection; valid explicit
// values are 1 through 8. The data slice must be non-empty.
func (a *Assessment) AssessIID(data []byte, bitsPerSymbol int) (*Result, error) {
	if bitsPerSymbol < 0 || bitsPerSymbol > 8 {
		return nil, newError("AssessIID", ErrInvalidBitsPerSymbol, fmt.Sprintf("got %d", bitsPerSymbol))
	}

	if len(data) == 0 {
		return nil, newError("AssessIID", ErrInvalidData, "data is empty")
	}

	if len(data) < MinRecommendedSamples && a.verbose > 0 {
		fmt.Fprintf(os.Stderr, "Warning: data contains less than %d samples\n", MinRecommendedSamples)
	}

	return calculateIIDEntropy(data, bitsPerSymbol, a.verbose)
}

// AssessNonIID performs a Non-IID entropy assessment using the ten estimators
// defined in NIST SP 800-90B Section 6.3. A bitsPerSymbol value of 0 triggers
// auto-detection; valid explicit values are 1 through 8.
func (a *Assessment) AssessNonIID(data []byte, bitsPerSymbol int) (*Result, error) {
	if bitsPerSymbol < 0 || bitsPerSymbol > 8 {
		return nil, newError("AssessNonIID", ErrInvalidBitsPerSymbol, fmt.Sprintf("got %d", bitsPerSymbol))
	}

	if len(data) == 0 {
		return nil, newError("AssessNonIID", ErrInvalidData, "data is empty")
	}

	if len(data) < MinRecommendedSamples && a.verbose > 0 {
		fmt.Fprintf(os.Stderr, "Warning: data contains less than %d samples\n", MinRecommendedSamples)
	}

	return calculateNonIIDEntropy(data, bitsPerSymbol, a.verbose)
}
