package entropy

import (
	"fmt"
	"io"
	"os"
)

const (
	// MinRecommendedSamples is the minimum number of samples recommended by SP800-90B
	MinRecommendedSamples = 1000000
)

// AssessFile reads a binary file and performs entropy assessment
func (a *Assessment) AssessFile(filename string, bitsPerSymbol int, testType TestType) (*Result, error) {
	// Open file
	file, err := os.Open(filename)
	if err != nil {
		return nil, newError("AssessFile", err, fmt.Sprintf("failed to open file: %s", filename))
	}
	defer file.Close()

	// Read file contents
	return a.AssessReader(file, bitsPerSymbol, testType)
}

// AssessReader reads from an io.Reader and performs entropy assessment
func (a *Assessment) AssessReader(r io.Reader, bitsPerSymbol int, testType TestType) (*Result, error) {
	// Read all data
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, newError("AssessReader", err, "failed to read data")
	}

	// Perform assessment based on test type
	switch testType {
	case IID:
		return a.AssessIID(data, bitsPerSymbol)
	case NonIID:
		return a.AssessNonIID(data, bitsPerSymbol)
	default:
		return nil, newError("AssessReader", ErrInvalidData, "invalid test type")
	}
}

// AssessIID performs IID (Independent and Identically Distributed) entropy assessment
func (a *Assessment) AssessIID(data []byte, bitsPerSymbol int) (*Result, error) {
	// Validate bits per symbol
	if bitsPerSymbol < 0 || bitsPerSymbol > 8 {
		return nil, newError("AssessIID", ErrInvalidBitsPerSymbol, fmt.Sprintf("got %d", bitsPerSymbol))
	}

	// Validate data
	if len(data) == 0 {
		return nil, newError("AssessIID", ErrInvalidData, "data is empty")
	}

	// Warn if data is below recommended size
	if len(data) < MinRecommendedSamples && a.verbose > 0 {
		fmt.Fprintf(os.Stderr, "Warning: data contains less than %d samples\n", MinRecommendedSamples)
	}

	// Call CGO bridge
	return calculateIIDEntropy(data, bitsPerSymbol, a.verbose)
}

// AssessNonIID performs Non-IID entropy assessment
func (a *Assessment) AssessNonIID(data []byte, bitsPerSymbol int) (*Result, error) {
	// Validate bits per symbol
	if bitsPerSymbol < 0 || bitsPerSymbol > 8 {
		return nil, newError("AssessNonIID", ErrInvalidBitsPerSymbol, fmt.Sprintf("got %d", bitsPerSymbol))
	}

	// Validate data
	if len(data) == 0 {
		return nil, newError("AssessNonIID", ErrInvalidData, "data is empty")
	}

	// Warn if data is below recommended size
	if len(data) < MinRecommendedSamples && a.verbose > 0 {
		fmt.Fprintf(os.Stderr, "Warning: data contains less than %d samples\n", MinRecommendedSamples)
	}

	// Call CGO bridge
	return calculateNonIIDEntropy(data, bitsPerSymbol, a.verbose)
}
