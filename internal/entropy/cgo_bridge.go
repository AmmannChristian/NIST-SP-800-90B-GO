//go:build !teststub

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

// calculateIIDEntropy calls the C wrapper to perform IID entropy calculation
func calculateIIDEntropy(data []byte, bitsPerSymbol int, verbose int) (*Result, error) {
	if len(data) == 0 {
		return nil, newError("calculateIIDEntropy", ErrInvalidData, "data is empty")
	}

	// Convert Go slice to C array
	cData := (*C.uint8_t)(unsafe.Pointer(&data[0]))
	cLength := C.size_t(len(data))
	cBitsPerSymbol := C.int(bitsPerSymbol)
	cIsBinary := C.bool(bitsPerSymbol == 1)
	cVerbose := C.int(verbose)

	// Call C function
	cResult := C.calculate_iid_entropy(cData, cLength, cBitsPerSymbol, cIsBinary, cVerbose)
	if cResult == nil {
		return nil, newError("calculateIIDEntropy", ErrMemoryAllocation, "failed to allocate result structure")
	}
	defer C.free_entropy_result(cResult)

	// Check for errors
	if cResult.error_code != 0 {
		errMsg := C.GoString(&cResult.error_message[0])
		return nil, wrapCError("calculateIIDEntropy", int(cResult.error_code), errMsg)
	}

	// Convert C result to Go result
	result := &Result{
		MinEntropy:   float64(cResult.min_entropy),
		HOriginal:    float64(cResult.h_original),
		HBitstring:   float64(cResult.h_bitstring),
		HAssessed:    float64(cResult.h_assessed),
		DataWordSize: int(cResult.data_word_size),
		TestType:     IID,
	}

	return result, nil
}

// calculateNonIIDEntropy calls the C wrapper to perform Non-IID entropy calculation
func calculateNonIIDEntropy(data []byte, bitsPerSymbol int, verbose int) (*Result, error) {
	if len(data) == 0 {
		return nil, newError("calculateNonIIDEntropy", ErrInvalidData, "data is empty")
	}

	// Convert Go slice to C array
	cData := (*C.uint8_t)(unsafe.Pointer(&data[0]))
	cLength := C.size_t(len(data))
	cBitsPerSymbol := C.int(bitsPerSymbol)
	cIsBinary := C.bool(bitsPerSymbol == 1)
	cVerbose := C.int(verbose)

	// Call C function
	cResult := C.calculate_non_iid_entropy(cData, cLength, cBitsPerSymbol, cIsBinary, cVerbose)
	if cResult == nil {
		return nil, newError("calculateNonIIDEntropy", ErrMemoryAllocation, "failed to allocate result structure")
	}
	defer C.free_entropy_result(cResult)

	// Check for errors
	if cResult.error_code != 0 {
		errMsg := C.GoString(&cResult.error_message[0])
		return nil, wrapCError("calculateNonIIDEntropy", int(cResult.error_code), errMsg)
	}

	// Convert C result to Go result
	result := &Result{
		MinEntropy:   float64(cResult.min_entropy),
		HOriginal:    float64(cResult.h_original),
		HBitstring:   float64(cResult.h_bitstring),
		HAssessed:    float64(cResult.h_assessed),
		DataWordSize: int(cResult.data_word_size),
		TestType:     NonIID,
	}

	return result, nil
}
