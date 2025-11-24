package entropy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTestType_String(t *testing.T) {
	tests := []struct {
		name     string
		testType TestType
		want     string
	}{
		{"IID", IID, "IID"},
		{"NonIID", NonIID, "Non-IID"},
		{"Unknown", TestType(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.testType.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewAssessment(t *testing.T) {
	assessment := NewAssessment()

	assert.NotNil(t, assessment)
	assert.Equal(t, 1, assessment.GetVerbose())
}

func TestAssessment_SetVerbose(t *testing.T) {
	assessment := NewAssessment()

	// Test valid values
	assessment.SetVerbose(0)
	assert.Equal(t, 0, assessment.GetVerbose())

	assessment.SetVerbose(3)
	assert.Equal(t, 3, assessment.GetVerbose())

	// Test clamping negative
	assessment.SetVerbose(-1)
	assert.Equal(t, 0, assessment.GetVerbose())

	// Test clamping too high
	assessment.SetVerbose(10)
	assert.Equal(t, 3, assessment.GetVerbose())
}

func TestResult(t *testing.T) {
	result := &Result{
		MinEntropy:   7.5,
		HOriginal:    7.8,
		HBitstring:   0.95,
		HAssessed:    7.5,
		DataWordSize: 8,
		TestType:     NonIID,
	}

	assert.Equal(t, 7.5, result.MinEntropy)
	assert.Equal(t, 7.8, result.HOriginal)
	assert.Equal(t, 0.95, result.HBitstring)
	assert.Equal(t, 7.5, result.HAssessed)
	assert.Equal(t, 8, result.DataWordSize)
	assert.Equal(t, NonIID, result.TestType)
}
