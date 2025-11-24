/* C wrapper for NIST SP800-90B Entropy Assessment C++ library */

#include "wrapper.h"

#include <cstring> // memcpy, strcpy
#include <cstdlib> // malloc, free
#include <exception>

// Include NIST C++ headers
#include "../cpp/shared/utils.h"
#include "../cpp/shared/most_common.h"
#include "../cpp/shared/lrs_test.h"
#include "../cpp/iid/iid_test_run.h"
#include "../cpp/iid/permutation_tests.h"
#include "../cpp/iid/chi_square_tests.h"
#include "../cpp/non_iid/non_iid_test_run.h"
#include "../cpp/non_iid/collision_test.h"
#include "../cpp/non_iid/lz78y_test.h"
#include "../cpp/non_iid/multi_mmc_test.h"
#include "../cpp/non_iid/lag_test.h"
#include "../cpp/non_iid/multi_mcw_test.h"
#include "../cpp/non_iid/compression_test.h"
#include "../cpp/non_iid/markov_test.h"

extern "C" {

// Helper function to create and initialize result structure
static EntropyResult*create_result() {
    EntropyResult* result = (EntropyResult*)malloc(sizeof(EntropyResult));
    if (result) {
        result->min_entropy = 0.0;
        result->h_original = 0.0;
        result->h_bitstring = 0.0;
        result->h_assessed = 0.0;
        result->data_word_size = 0;
        result->error_code = 0;
        result->error_message[0] = '\0';
    }
    return result;
}

// Helper function to set error in result
static void set_error(EntropyResult* result, int code, const char* message) {
    result->error_code = code;
    strncpy(result->error_message, message, sizeof(result->error_message) - 1);
    result->error_message[sizeof(result->error_message) - 1] = '\0';
}

// Helper function to prepare data_t structure from raw bytes
static bool prepare_data(data_t* dp, const uint8_t* data, size_t length, int bits_per_symbol, EntropyResult* result) {
    dp->word_size = bits_per_symbol;
    dp->len = (long)length;
    dp->symbols = NULL;
    dp->rawsymbols = NULL;
    dp->bsymbols = NULL;
    dp->alph_size = 0;
    dp->maxsymbol = 0;
    dp->blen = 0;
    
    // Allocate memory for symbols
    dp->symbols = (uint8_t*)malloc(sizeof(uint8_t) * dp->len);
    dp->rawsymbols = (uint8_t*)malloc(sizeof(uint8_t) * dp->len);
    
    if (!dp->symbols || !dp->rawsymbols) {
        set_error(result, -1, "Failed to allocate memory for symbols");
        if (dp->symbols) free(dp->symbols);
        if (dp->rawsymbols) free(dp->rawsymbols);
        return false;
    }
    
    // Copy data
    memcpy(dp->symbols, data, dp->len);
    memcpy(dp->rawsymbols, data, dp->len);
    
    // Auto-detect word size if needed
    if (dp->word_size == 0) {
        uint8_t datamask = 0;
        for (long i = 0; i < dp->len; i++) {
            datamask |= dp->symbols[i];
        }
        
        uint8_t curbit = 0x80;
        int detected_size = 8;
        for (int i = 8; i > 0 && (datamask & curbit) == 0; i--) {
            curbit >>= 1;
            detected_size = i - 1;
        }
        dp->word_size = (detected_size == -1) ? 1 : detected_size + 1;
    }
    
    // Validate symbol width
    int max_symbols = 1 << dp->word_size;
    int mask = max_symbols - 1;
    
    // Process symbols and build alphabet
    int symbol_map_down_table[max_symbols];
    memset(symbol_map_down_table, 0, max_symbols * sizeof(int));
    
    dp->alph_size = 0;
    dp->maxsymbol = 0;
    
    for (long i = 0; i < dp->len; i++) {
        dp->symbols[i] &= mask;
        if (dp->symbols[i] > dp->maxsymbol) {
            dp->maxsymbol = dp->symbols[i];
        }
        if (symbol_map_down_table[dp->symbols[i]] == 0) {
            symbol_map_down_table[dp->symbols[i]] = 1;
        }
    }
    
    // Create symbol mapping
    for (int i = 0; i < max_symbols; i++) {
        if (symbol_map_down_table[i] != 0) {
            symbol_map_down_table[i] = dp->alph_size++;
        }
    }
    
    // Create bitstring
    dp->blen = dp->len * dp->word_size;
    if (dp->word_size == 1) {
        dp->bsymbols = dp->symbols;
    } else {
        dp->bsymbols = (uint8_t*)malloc(dp->blen);
        if (!dp->bsymbols) {
            set_error(result, -1, "Failed to allocate memory for bitstring");
            free(dp->symbols);
            free(dp->rawsymbols);
            return false;
        }
        
        for (long i = 0; i < dp->len; i++) {
            for (int j = 0; j < dp->word_size; j++) {
                dp->bsymbols[i * dp->word_size + j] = 
                    (dp->symbols[i] >> (dp->word_size - 1 - j)) & 0x1;
            }
        }
    }
    
    // Map down symbols
    if (dp->alph_size < dp->maxsymbol + 1) {
        for (long i = 0; i < dp->len; i++) {
            dp->symbols[i] = (uint8_t)symbol_map_down_table[dp->symbols[i]];
        }
    }
    
    return true;
}

EntropyResult* calculate_iid_entropy(
    const uint8_t* data,
    size_t length,
    int bits_per_symbol,
    bool is_binary,
    int verbose
) {
    EntropyResult* result = create_result();
    if (!result) {
        return NULL;
    }
    
    try {
        // Validate input
        if (!data || length == 0) {
            set_error(result, -1, "Invalid input: data is NULL or empty");
            return result;
        }
        
        if (bits_per_symbol < 0 || bits_per_symbol > 8) {
            set_error(result, -1, "Invalid bits_per_symbol: must be 0-8");
            return result;
        }
        
        // Prepare data structure
        data_t dp;
        if (!prepare_data(&dp, data, length, bits_per_symbol, result)) {
            return result;
        }
        
        // Check alphabet size
        if (dp.alph_size <= 1) {
            set_error(result, -1, "Symbol alphabet consists of 1 symbol. No entropy awarded.");
            free_data(&dp);
            return result;
        }
        
        // Calculate entropy estimates
        double H_original = dp.word_size;
        double H_bitstring = 1.0;
        
        // Most Common Value estimate
        H_original = most_common(dp.symbols, dp.len, dp.alph_size, verbose, "Literal");
        
        if (dp.alph_size > 2) {
            H_bitstring = most_common(dp.bsymbols, dp.blen, 2, verbose, "Bitstring");
        }
        
        // Chi-square tests
        bool chi_square_pass = chi_square_tests(dp.symbols, dp.len, dp.alph_size, verbose);
        
        // LRS test
        bool lrs_pass = len_LRS_test(dp.symbols, dp.len, dp.alph_size, verbose, "Literal");
        
        // Permutation tests
        double rawmean, median;
        calc_stats(&dp, rawmean, median);
        IidTestCase tc;
        bool perm_pass = permutation_tests(&dp, rawmean, median, verbose, tc);
        
        // Calculate assessed entropy
        double h_assessed = dp.word_size;
        if (dp.alph_size > 2) {
            h_assessed = std::min(h_assessed, H_bitstring * dp.word_size);
        }
        h_assessed = std::min(h_assessed, H_original);
        
        // Set results
        result->h_original = H_original;
        result->h_bitstring = H_bitstring;
        result->h_assessed = h_assessed;
        result->min_entropy = h_assessed;
        result->data_word_size = dp.word_size;
        result->error_code = 0;
        
        // Clean up
        free_data(&dp);
        
    } catch (const std::exception& e) {
        set_error(result, -2, e.what());
    } catch (...) {
        set_error(result, -2, "Unknown exception occurred");
    }
    
    return result;
}

EntropyResult* calculate_non_iid_entropy(
    const uint8_t* data,
    size_t length,
    int bits_per_symbol,
    bool is_binary,
    int verbose
) {
    EntropyResult* result = create_result();
    if (!result) {
        return NULL;
    }
    
    try {
        // Validate input
        if (!data || length == 0) {
            set_error(result, -1, "Invalid input: data is NULL or empty");
            return result;
        }
        
        if (bits_per_symbol < 0 || bits_per_symbol > 8) {
            set_error(result, -1, "Invalid bits_per_symbol: must be 0-8");
            return result;
        }
        
        // Prepare data structure
        data_t dp;
        if (!prepare_data(&dp, data, length, bits_per_symbol, result)) {
            return result;
        }
        
        // Check alphabet size
        if (dp.alph_size <= 1) {
            set_error(result, -1, "Symbol alphabet consists of 1 symbol. No entropy awarded.");
            free_data(&dp);
            return result;
        }
        
        // Initialize entropy estimates
        double H_original = dp.word_size;
        double H_bitstring = 1.0;
        double ret_min_entropy;
        
        // Section 6.3.1 - Most Common Value
        if (dp.alph_size > 2 || !is_binary) {
            ret_min_entropy = most_common(dp.bsymbols, dp.blen, 2, verbose, "Bitstring");
            H_bitstring = std::min(ret_min_entropy, H_bitstring);
        }
        ret_min_entropy = most_common(dp.symbols, dp.len, dp.alph_size, verbose, "Literal");
        H_original = std::min(ret_min_entropy, H_original);
        
        // Section 6.3.2 - Collision Test (bit strings only)
        if (dp.alph_size > 2 || !is_binary) {
            ret_min_entropy = collision_test(dp.bsymbols, dp.blen, verbose, "Bitstring");
            H_bitstring = std::min(ret_min_entropy, H_bitstring);
        }
        if (is_binary && dp.alph_size == 2) {
            ret_min_entropy = collision_test(dp.symbols, dp.len, verbose, "Literal");
            H_original = std::min(ret_min_entropy, H_original);
        }
        
        // Section 6.3.3 - Markov Test (bit strings only)
        if (dp.alph_size > 2 || !is_binary) {
            ret_min_entropy = markov_test(dp.bsymbols, dp.blen, verbose, "Bitstring");
            H_bitstring = std::min(ret_min_entropy, H_bitstring);
        }
        if (is_binary && dp.alph_size == 2) {
            ret_min_entropy = markov_test(dp.symbols, dp.len, verbose, "Literal");
            H_original = std::min(ret_min_entropy, H_original);
        }
        
        // Section 6.3.4 - Compression Test (bit strings only)
        if (dp.alph_size > 2 || !is_binary) {
            ret_min_entropy = compression_test(dp.bsymbols, dp.blen, verbose, "Bitstring");
            if (ret_min_entropy >= 0) {
                H_bitstring = std::min(ret_min_entropy, H_bitstring);
            }
        }
        if (is_binary && dp.alph_size == 2) {
            ret_min_entropy = compression_test(dp.symbols, dp.len, verbose, "Literal");
            if (ret_min_entropy >= 0) {
                H_original = std::min(ret_min_entropy, H_original);
            }
        }
        
        // Section 6.3.5 - t-Tuple Test
        // Section 6.3.6 - LRS Test
        double bin_t_tuple_res = -1.0, bin_lrs_res = -1.0;
        double t_tuple_res = -1.0, lrs_res = -1.0;
        
        if (dp.alph_size > 2 || !is_binary) {
            SAalgs(dp.bsymbols, dp.blen, 2, bin_t_tuple_res, bin_lrs_res, verbose, "Bitstring");
            if (bin_t_tuple_res >= 0.0) {
                H_bitstring = std::min(bin_t_tuple_res, H_bitstring);
            }
            if (bin_lrs_res >= 0.0) {
                H_bitstring = std::min(bin_lrs_res, H_bitstring);
            }
        }
        
        SAalgs(dp.symbols, dp.len, dp.alph_size, t_tuple_res, lrs_res, verbose, "Literal");
        if (t_tuple_res >= 0.0) {
            H_original = std::min(t_tuple_res, H_original);
        }
        if (lrs_res >= 0.0) {
            H_original = std::min(lrs_res, H_original);
        }
        
        // Section 6.3.7 - MultiMCW Test
        if (dp.alph_size > 2 || !is_binary) {
            ret_min_entropy = multi_mcw_test(dp.bsymbols, dp.blen, 2, verbose, "Bitstring");
            if (ret_min_entropy >= 0) {
                H_bitstring = std::min(ret_min_entropy, H_bitstring);
            }
        }
        ret_min_entropy = multi_mcw_test(dp.symbols, dp.len, dp.alph_size, verbose, "Literal");
        if (ret_min_entropy >= 0) {
            H_original = std::min(ret_min_entropy, H_original);
        }
        
        // Section 6.3.8 - Lag Prediction Test
        if (dp.alph_size > 2 || !is_binary) {
            ret_min_entropy = lag_test(dp.bsymbols, dp.blen, 2, verbose, "Bitstring");
            if (ret_min_entropy >= 0) {
                H_bitstring = std::min(ret_min_entropy, H_bitstring);
            }
        }
        ret_min_entropy = lag_test(dp.symbols, dp.len, dp.alph_size, verbose, "Literal");
        if (ret_min_entropy >= 0) {
            H_original = std::min(ret_min_entropy, H_original);
        }
        
        // Section 6.3.9 - MultiMMC Test
        if (dp.alph_size > 2 || !is_binary) {
            ret_min_entropy = multi_mmc_test(dp.bsymbols, dp.blen, 2, verbose, "Bitstring");
            if (ret_min_entropy >= 0) {
                H_bitstring = std::min(ret_min_entropy, H_bitstring);
            }
        }
        ret_min_entropy = multi_mmc_test(dp.symbols, dp.len, dp.alph_size, verbose, "Literal");
        if (ret_min_entropy >= 0) {
            H_original = std::min(ret_min_entropy, H_original);
        }
        
        // Section 6.3.10 - LZ78Y Test
        if (dp.alph_size > 2 || !is_binary) {
            ret_min_entropy = LZ78Y_test(dp.bsymbols, dp.blen, 2, verbose, "Bitstring");
            if (ret_min_entropy >= 0) {
                H_bitstring = std::min(ret_min_entropy, H_bitstring);
            }
        }
        ret_min_entropy = LZ78Y_test(dp.symbols, dp.len, dp.alph_size, verbose, "Literal");
        if (ret_min_entropy >= 0) {
            H_original = std::min(ret_min_entropy, H_original);
        }
        
        // Calculate assessed entropy
        double h_assessed = dp.word_size;
        if (dp.alph_size > 2 || !is_binary) {
            h_assessed = std::min(h_assessed, H_bitstring * dp.word_size);
        }
        h_assessed = std::min(h_assessed, H_original);
        
        // Set results
        result->h_original = H_original;
        result->h_bitstring = H_bitstring;
        result->h_assessed = h_assessed;
        result->min_entropy = h_assessed;
        result->data_word_size = dp.word_size;
        result->error_code = 0;
        
        // Clean up
        free_data(&dp);
        
    } catch (const std::exception& e) {
        set_error(result, -2, e.what());
    } catch (...) {
        set_error(result, -2, "Unknown exception occurred");
    }
    
    return result;
}

void free_entropy_result(EntropyResult* result) {
    if (result) {
        free(result);
    }
}

} // extern "C"
