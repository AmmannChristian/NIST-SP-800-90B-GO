#ifndef ENTROPY_WRAPPER_H
#define ENTROPY_WRAPPER_H

#ifdef __cplusplus
extern "C" {
#endif

#include <stdint.h>
#include <stddef.h>
#include <stdbool.h>

// Result structure for entropy calculations
typedef struct {
    double min_entropy;      // Minimum entropy estimate
    double h_original;       // Entropy from original symbols
    double h_bitstring;      // Entropy from bitstring
    double h_assessed;       // Assessed entropy value
    int data_word_size;      // Bits per symbol
    int error_code;          // 0 = success, negative = error
    char error_message[512]; // Error description
} EntropyResult;

/**
 * Calculate IID (Independent and Identically Distributed) entropy estimate
 * 
 * @param data Pointer to binary data
 * @param length Number of bytes in data
 * @param bits_per_symbol Number of bits per symbol (1-8), 0 for auto-detect
 * @param is_binary Whether data is binary (1-bit symbols)
 * @param verbose Verbosity level (0=quiet, 1=normal, 2=verbose, 3=very verbose)
 * @return Pointer to EntropyResult structure (must be freed with free_entropy_result)
 */
EntropyResult* calculate_iid_entropy(
    const uint8_t* data,
    size_t length,
    int bits_per_symbol,
    bool is_binary,
    int verbose
);

/**
 * Calculate Non-IID entropy estimate
 * 
 * @param data Pointer to binary data
 * @param length Number of bytes in data
 * @param bits_per_symbol Number of bits per symbol (1-8), 0 for auto-detect
 * @param is_binary Whether data is binary (1-bit symbols)
 * @param verbose Verbosity level (0=quiet, 1=normal, 2=verbose, 3=very verbose)
 * @return Pointer to EntropyResult structure (must be freed with free_entropy_result)
 */
EntropyResult* calculate_non_iid_entropy(
    const uint8_t* data,
    size_t length,
    int bits_per_symbol,
    bool is_binary,
    int verbose
);

/**
 * Free an EntropyResult structure
 * 
 * @param result Pointer to EntropyResult to free
 */
void free_entropy_result(EntropyResult* result);

#ifdef __cplusplus
}
#endif

#endif // ENTROPY_WRAPPER_H
