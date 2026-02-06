//go:build !teststub

// This file provides the CGO bridge to the NIST SP 800-90B C++ reference
// implementation. It is excluded when the "teststub" build tag is active,
// allowing unit tests to run without the C++ toolchain.

package entropy

/*
#cgo CXXFLAGS: -std=c++11 -fopenmp -I${SRCDIR}/../../internal/nist/cpp -I${SRCDIR}/../../internal/nist/wrapper
#cgo LDFLAGS: -L${SRCDIR}/../../internal/nist/lib -lentropy90b -lbz2 -ldivsufsort -ldivsufsort64 -ljsoncpp -lmpfr -lgmp -lgomp -lstdc++ -lm -lcrypto
#include "../../internal/nist/wrapper/wrapper.h"
#include <stdlib.h>
*/
import "C"

import (
	"unsafe"
)

// calculateIIDEntropy invokes the C wrapper to run IID tests including
// Most Common Value, Chi-Square, LRS, and Permutation tests.
func calculateIIDEntropy(data []byte, bitsPerSymbol int, verbose int) (*Result, error) {
	if len(data) == 0 {
		return nil, newError("calculateIIDEntropy", ErrInvalidData, "data is empty")
	}

	cData := (*C.uint8_t)(unsafe.Pointer(&data[0]))
	cLength := C.size_t(len(data))
	cBitsPerSymbol := C.int(bitsPerSymbol)
	// Always use initial_entropy=true for IID tests (not conditioned mode)
	cInitialEntropy := C.bool(true)
	cVerbose := C.int(verbose)

	cResult := C.calculate_iid_entropy(cData, cLength, cBitsPerSymbol, cInitialEntropy, cVerbose)
	if cResult == nil {
		return nil, newError("calculateIIDEntropy", ErrMemoryAllocation, "failed to allocate result structure")
	}
	defer C.free_entropy_result(cResult)

	if cResult.error_code != 0 {
		errMsg := C.GoString(&cResult.error_message[0])
		return nil, wrapCError("calculateIIDEntropy", int(cResult.error_code), errMsg)
	}

	result := &Result{
		MinEntropy:   float64(cResult.min_entropy),
		HOriginal:    float64(cResult.h_original),
		HBitstring:   float64(cResult.h_bitstring),
		HAssessed:    float64(cResult.h_assessed),
		DataWordSize: int(cResult.data_word_size),
		TestType:     IID,
		Estimators:   convertEstimators(cResult),
	}

	return result, nil
}

// convertEstimators marshals the C-allocated estimator array from an
// EntropyResult into a Go slice of EstimatorResult values.
func convertEstimators(cResult *C.EntropyResult) []EstimatorResult {
	count := int(cResult.estimator_count)
	if count <= 0 {
		return nil
	}

	estimators := make([]EstimatorResult, count)
	for i := 0; i < count; i++ {
		cEst := cResult.estimators[i]
		estimators[i] = EstimatorResult{
			Name:            C.GoString(&cEst.name[0]),
			EntropyEstimate: float64(cEst.entropy_estimate),
			Passed:          bool(cEst.passed),
			IsEntropyValid:  bool(cEst.is_entropy_valid),
		}
	}
	return estimators
}

// calculateNonIIDEntropy invokes the C wrapper to run all ten Non-IID
// estimators defined in NIST SP 800-90B Section 6.3.
func calculateNonIIDEntropy(data []byte, bitsPerSymbol int, verbose int) (*Result, error) {
	if len(data) == 0 {
		return nil, newError("calculateNonIIDEntropy", ErrInvalidData, "data is empty")
	}

	cData := (*C.uint8_t)(unsafe.Pointer(&data[0]))
	cLength := C.size_t(len(data))
	cBitsPerSymbol := C.int(bitsPerSymbol)
	// Always use initial_entropy=true for Non-IID tests (not conditioned mode)
	// This matches the NIST CLI default behavior with -i flag
	cInitialEntropy := C.bool(true)
	cVerbose := C.int(verbose)

	cResult := C.calculate_non_iid_entropy(cData, cLength, cBitsPerSymbol, cInitialEntropy, cVerbose)
	if cResult == nil {
		return nil, newError("calculateNonIIDEntropy", ErrMemoryAllocation, "failed to allocate result structure")
	}
	defer C.free_entropy_result(cResult)

	if cResult.error_code != 0 {
		errMsg := C.GoString(&cResult.error_message[0])
		return nil, wrapCError("calculateNonIIDEntropy", int(cResult.error_code), errMsg)
	}

	result := &Result{
		MinEntropy:   float64(cResult.min_entropy),
		HOriginal:    float64(cResult.h_original),
		HBitstring:   float64(cResult.h_bitstring),
		HAssessed:    float64(cResult.h_assessed),
		DataWordSize: int(cResult.data_word_size),
		TestType:     NonIID,
		Estimators:   convertEstimators(cResult),
	}

	return result, nil
}
