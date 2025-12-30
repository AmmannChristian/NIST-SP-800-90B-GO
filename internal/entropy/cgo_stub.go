//go:build teststub

package entropy

import "math"

// Test stub for CGO-backed functions to allow fast unit testing without the C++ library.

// stubIIDEstimators returns mock IID estimator results
func stubIIDEstimators() []EstimatorResult {
	return []EstimatorResult{
		{Name: "Most Common Value", EntropyEstimate: 7.6, Passed: true, IsEntropyValid: true},
		{Name: "Chi-Square Tests", EntropyEstimate: -1.0, Passed: true, IsEntropyValid: false},
		{Name: "Length of Longest Repeated Substring Test", EntropyEstimate: -1.0, Passed: true, IsEntropyValid: false},
		{Name: "Permutation Tests", EntropyEstimate: -1.0, Passed: true, IsEntropyValid: false},
	}
}

// stubNonIIDEstimators returns mock Non-IID estimator results
func stubNonIIDEstimators() []EstimatorResult {
	return []EstimatorResult{
		{Name: "Most Common Value", EntropyEstimate: 6.8, Passed: true, IsEntropyValid: true},
		{Name: "Collision Test", EntropyEstimate: 6.9, Passed: true, IsEntropyValid: true},
		{Name: "Markov Test", EntropyEstimate: 6.7, Passed: true, IsEntropyValid: true},
		{Name: "Compression Test", EntropyEstimate: 6.5, Passed: true, IsEntropyValid: true},
		{Name: "t-Tuple Test", EntropyEstimate: 6.6, Passed: true, IsEntropyValid: true},
		{Name: "LRS Test", EntropyEstimate: 6.8, Passed: true, IsEntropyValid: true},
		{Name: "Multi Most Common in Window Test", EntropyEstimate: 6.7, Passed: true, IsEntropyValid: true},
		{Name: "Lag Prediction Test", EntropyEstimate: 6.9, Passed: true, IsEntropyValid: true},
		{Name: "Multi Markov Model with Counting Test", EntropyEstimate: 6.6, Passed: true, IsEntropyValid: true},
		{Name: "LZ78Y Test", EntropyEstimate: 6.5, Passed: true, IsEntropyValid: true},
	}
}

func calculateIIDEntropy(data []byte, bitsPerSymbol int, verbose int) (*Result, error) {
	if len(data) > 0 && data[0] == 0xFF {
		return nil, newError("calculateIIDEntropy", ErrInvalidData, "stub failure")
	}
	if len(data) > 0 && data[0] == 0xEE {
		return &Result{
			MinEntropy:   math.Inf(1),
			HOriginal:    math.Inf(1),
			HBitstring:   math.Inf(1),
			HAssessed:    math.Inf(1),
			DataWordSize: bitsPerSymbol,
			TestType:     IID,
			Estimators:   nil,
		}, nil
	}
	return &Result{
		MinEntropy:   7.5,
		HOriginal:    7.6,
		HBitstring:   7.1,
		HAssessed:    7.5,
		DataWordSize: bitsPerSymbol,
		TestType:     IID,
		Estimators:   stubIIDEstimators(),
	}, nil
}

func calculateNonIIDEntropy(data []byte, bitsPerSymbol int, verbose int) (*Result, error) {
	if len(data) > 0 && data[0] == 0xFF {
		return nil, newError("calculateNonIIDEntropy", ErrInvalidData, "stub failure")
	}
	if len(data) > 0 && data[0] == 0xEE {
		return &Result{
			MinEntropy:   math.Inf(1),
			HOriginal:    math.Inf(1),
			HBitstring:   math.Inf(1),
			HAssessed:    math.Inf(1),
			DataWordSize: bitsPerSymbol,
			TestType:     NonIID,
			Estimators:   nil,
		}, nil
	}
	return &Result{
		MinEntropy:   6.5,
		HOriginal:    6.6,
		HBitstring:   6.1,
		HAssessed:    6.5,
		DataWordSize: bitsPerSymbol,
		TestType:     NonIID,
		Estimators:   stubNonIIDEstimators(),
	}, nil
}
