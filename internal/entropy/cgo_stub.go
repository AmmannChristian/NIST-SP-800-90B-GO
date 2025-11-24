//go:build teststub

package entropy

import "math"

// Test stub for CGO-backed functions to allow fast unit testing without the C++ library.

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
		}, nil
	}
	return &Result{
		MinEntropy:   7.5,
		HOriginal:    7.6,
		HBitstring:   7.1,
		HAssessed:    7.5,
		DataWordSize: bitsPerSymbol,
		TestType:     IID,
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
		}, nil
	}
	return &Result{
		MinEntropy:   6.5,
		HOriginal:    6.6,
		HBitstring:   6.1,
		HAssessed:    6.5,
		DataWordSize: bitsPerSymbol,
		TestType:     NonIID,
	}, nil
}
