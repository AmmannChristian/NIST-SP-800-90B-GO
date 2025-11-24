package service

import (
	"fmt"

	"github.com/AmmannChristian/nist-800-90b/internal/entropy"
)

// EntropyService provides business logic for entropy assessment
type EntropyService struct {
	assessment *entropy.Assessment
}

// NewService creates a new EntropyService instance
func NewService() *EntropyService {
	return &EntropyService{
		assessment: entropy.NewAssessment(),
	}
}

// SetVerbose sets the verbosity level for entropy calculations
func (s *EntropyService) SetVerbose(level int) {
	s.assessment.SetVerbose(level)
}

// AssessIID performs IID entropy assessment on the provided data
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

// AssessNonIID performs Non-IID entropy assessment on the provided data
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
