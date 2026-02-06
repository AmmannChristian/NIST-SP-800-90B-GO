/**
 * @file wrapper.h
 * @brief C-linkage API for NIST SP 800-90B entropy assessment.
 *
 * Declares the IID and Non-IID assessment entry points, the result structures
 * returned to the caller, and the corresponding free function. This header is
 * designed for consumption by CGO.
 */

#ifndef ENTROPY_WRAPPER_H
#define ENTROPY_WRAPPER_H

#ifdef __cplusplus
extern "C" {
#endif

#include <stdint.h>
#include <stddef.h>
#include <stdbool.h>

// Maximum number of estimators per assessment
#define MAX_ESTIMATORS 16

// EstimatorResult holds the output of a single entropy estimator or statistical test.
typedef struct {
    char name[64];           // Estimator name (e.g., "Most Common Value")
    double entropy_estimate; // Entropy estimate (-1.0 if not applicable)
    bool passed;             // Whether the test passed
    bool is_entropy_valid;   // true if entropy_estimate is valid
} EstimatorResult;

// EntropyResult holds the aggregate output of an IID or Non-IID assessment.
typedef struct {
    double min_entropy;      // Minimum entropy estimate
    double h_original;       // Entropy from original symbols
    double h_bitstring;      // Entropy from bitstring
    double h_assessed;       // Assessed entropy value
    int data_word_size;      // Bits per symbol
    int error_code;          // 0 = success, negative = error
    char error_message[512]; // Error description

    // Individual estimator results
    EstimatorResult estimators[MAX_ESTIMATORS];
    int estimator_count;     // Number of valid entries in estimators array
} EntropyResult;

/**
 * Calculate IID (Independent and Identically Distributed) entropy estimate.
 *
 * @param data Pointer to raw sample bytes.
 * @param length Number of bytes in data.
 * @param bits_per_symbol Number of bits per symbol (1-8), 0 for auto-detect.
 * @param is_binary If true, run in initial-entropy mode (unconditioned source).
 * @param verbose Verbosity level (0=quiet, 1=normal, 2=verbose, 3=very verbose).
 * @return Pointer to EntropyResult (caller must free with free_entropy_result).
 */
EntropyResult* calculate_iid_entropy(
    const uint8_t* data,
    size_t length,
    int bits_per_symbol,
    bool is_binary,
    int verbose
);

/**
 * Calculate Non-IID entropy estimate using all ten SP 800-90B Section 6.3
 * estimators.
 *
 * @param data Pointer to raw sample bytes.
 * @param length Number of bytes in data.
 * @param bits_per_symbol Number of bits per symbol (1-8), 0 for auto-detect.
 * @param is_binary If true, run in initial-entropy mode (unconditioned source).
 * @param verbose Verbosity level (0=quiet, 1=normal, 2=verbose, 3=very verbose).
 * @return Pointer to EntropyResult (caller must free with free_entropy_result).
 */
EntropyResult* calculate_non_iid_entropy(
    const uint8_t* data,
    size_t length,
    int bits_per_symbol,
    bool is_binary,
    int verbose
);

/**
 * Free an EntropyResult structure allocated by a calculate_* function.
 *
 * @param result Pointer to EntropyResult to free (NULL-safe).
 */
void free_entropy_result(EntropyResult* result);

#ifdef __cplusplus
}
#endif

#endif // ENTROPY_WRAPPER_H
