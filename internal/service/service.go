// Package service implements the business and transport layers for the NIST
// SP 800-90B entropy assessment microservice. EntropyService encapsulates
// domain logic, while GRPCServer exposes it as a gRPC API.
package service

import (
	"fmt"

	"github.com/AmmannChristian/nist-800-90b/internal/entropy"
)

// EntropyService provides the business-logic layer for entropy assessment,
// wrapping the lower-level Assessment with input validation.
type EntropyService struct {
	assessment *entropy.Assessment
}

// NewService creates a new EntropyService with default assessment settings.
func NewService() *EntropyService {
	return &EntropyService{
		assessment: entropy.NewAssessment(),
	}
}

// SetVerbose sets the verbosity level for entropy calculations.
func (s *EntropyService) SetVerbose(level int) {
	s.assessment.SetVerbose(level)
}

// AssessIID validates inputs and performs an IID entropy assessment on the
// provided data. A bitsPerSymbol of 0 enables auto-detection.
func (s *EntropyService) AssessIID(data []byte, bitsPerSymbol int) (*entropy.Result, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("data cannot be empty")
	}

	if bitsPerSymbol < 0 || bitsPerSymbol > 8 {
		return nil, fmt.Errorf("bits_per_symbol must be between 0 (auto-detect) and 8, got %d", bitsPerSymbol)
	}

	result, err := s.assessment.AssessIID(data, bitsPerSymbol)
	if err != nil {
		return nil, fmt.Errorf("IID assessment failed: %w", err)
	}

	return result, nil
}

// AssessNonIID validates inputs and performs a Non-IID entropy assessment on
// the provided data. A bitsPerSymbol of 0 enables auto-detection.
func (s *EntropyService) AssessNonIID(data []byte, bitsPerSymbol int) (*entropy.Result, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("data cannot be empty")
	}

	if bitsPerSymbol < 0 || bitsPerSymbol > 8 {
		return nil, fmt.Errorf("bits_per_symbol must be between 0 (auto-detect) and 8, got %d", bitsPerSymbol)
	}

	result, err := s.assessment.AssessNonIID(data, bitsPerSymbol)
	if err != nil {
		return nil, fmt.Errorf("Non-IID assessment failed: %w", err)
	}

	return result, nil
}
