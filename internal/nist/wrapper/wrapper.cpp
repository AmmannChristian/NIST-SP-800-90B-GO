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

#include <array>

// RAII guard for data_t to prevent memory leaks on exceptions
class DataGuard {
public:
    explicit DataGuard(data_t* dp) : dp_(dp), released_(false) {}
    ~DataGuard() {
        if (!released_ && dp_) {
            free_data(dp_);
        }
    }
    void release() { released_ = true; }

    // Non-copyable
    DataGuard(const DataGuard&) = delete;
    DataGuard& operator=(const DataGuard&) = delete;
private:
    data_t* dp_;
    bool released_;
};

extern "C" {

// Helper function to create and initialize result structure
static EntropyResult* create_result() {
    EntropyResult* result = (EntropyResult*)malloc(sizeof(EntropyResult));
    if (result) {
        result->min_entropy = 0.0;
        result->h_original = 0.0;
        result->h_bitstring = 0.0;
        result->h_assessed = 0.0;
        result->data_word_size = 0;
        result->error_code = 0;
        result->error_message[0] = '\0';
        result->estimator_count = 0;
    }
    return result;
}

// Helper function to add an estimator result with entropy value
static void add_estimator(EntropyResult* result, const char* name, double entropy, bool passed) {
    if (result->estimator_count >= MAX_ESTIMATORS) return;
    EstimatorResult* est = &result->estimators[result->estimator_count++];
    strncpy(est->name, name, sizeof(est->name) - 1);
    est->name[sizeof(est->name) - 1] = '\0';
    est->entropy_estimate = entropy;
    est->passed = passed;
    est->is_entropy_valid = (entropy >= 0.0);
}

// Helper function to add a pass/fail test result (no entropy value)
static void add_test_result(EntropyResult* result, const char* name, bool passed) {
    if (result->estimator_count >= MAX_ESTIMATORS) return;
    EstimatorResult* est = &result->estimators[result->estimator_count++];
    strncpy(est->name, name, sizeof(est->name) - 1);
    est->name[sizeof(est->name) - 1] = '\0';
    est->entropy_estimate = -1.0;
    est->passed = passed;
    est->is_entropy_valid = false;
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

    // Validate symbol width (max 8 bits = 256 symbols)
    int max_symbols = 1 << dp->word_size;
    int mask = max_symbols - 1;

    // Process symbols and build alphabet
    // Using std::array instead of VLA for C++ standard compliance
    std::array<int, 256> symbol_map_down_table{};  // zero-initialized

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

    // Create bitstring from rawsymbols (not mapped symbols)
    // See NIST Issue #71: https://github.com/usnistgov/SP800-90B_EntropyAssessment/issues/71
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

        // Build bitstring from rawsymbols, not from mapped symbols
        for (long i = 0; i < dp->len; i++) {
            uint8_t raw = dp->rawsymbols[i] & mask;
            for (int j = 0; j < dp->word_size; j++) {
                dp->bsymbols[i * dp->word_size + j] =
                    (raw >> (dp->word_size - 1 - j)) & 0x1;
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
        DataGuard guard(&dp);  // RAII: ensures free_data() on any exit path

        // Check alphabet size
        if (dp.alph_size <= 1) {
            set_error(result, -1, "Symbol alphabet consists of 1 symbol. No entropy awarded.");
            return result;
        }

        // Calculate entropy estimates
        double H_original = dp.word_size;
        double H_bitstring = 1.0;

        // Most Common Value estimate
        H_original = most_common(dp.symbols, dp.len, dp.alph_size, verbose, "Literal");
        add_estimator(result, "Most Common Value", H_original, true);

        if (dp.alph_size > 2) {
            H_bitstring = most_common(dp.bsymbols, dp.blen, 2, verbose, "Bitstring");
        }

        // Chi-square tests
        bool chi_square_pass = chi_square_tests(dp.symbols, dp.len, dp.alph_size, verbose);
        add_test_result(result, "Chi-Square Tests", chi_square_pass);

        // LRS test
        bool lrs_pass = len_LRS_test(dp.symbols, dp.len, dp.alph_size, verbose, "Literal");
        add_test_result(result, "Length of Longest Repeated Substring Test", lrs_pass);

        // Permutation tests
        double rawmean, median;
        calc_stats(&dp, rawmean, median);
        IidTestCase tc;
        bool perm_pass = permutation_tests(&dp, rawmean, median, verbose, tc);
        add_test_result(result, "Permutation Tests", perm_pass);

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

        // guard destructor calls free_data(&dp) automatically

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
        DataGuard guard(&dp);  // RAII: ensures free_data() on any exit path

        // Check alphabet size
        if (dp.alph_size <= 1) {
            set_error(result, -1, "Symbol alphabet consists of 1 symbol. No entropy awarded.");
            return result;
        }

        // Initialize entropy estimates
        double H_original = dp.word_size;
        double H_bitstring = 1.0;
        double ret_min_entropy;

        // Section 6.3.1 - Most Common Value
        // Note: is_binary parameter represents initial_entropy mode (not whether data is binary)
        bool initial_entropy = is_binary;
        double mcv_entropy = -1.0;

        if ((dp.alph_size > 2) || !initial_entropy) {
            ret_min_entropy = most_common(dp.bsymbols, dp.blen, 2, verbose, "Bitstring");
            H_bitstring = std::min(ret_min_entropy, H_bitstring);
            mcv_entropy = ret_min_entropy;
        }
        if (initial_entropy) {
            ret_min_entropy = most_common(dp.symbols, dp.len, dp.alph_size, verbose, "Literal");
            H_original = std::min(ret_min_entropy, H_original);
            mcv_entropy = ret_min_entropy;
        }
        add_estimator(result, "Most Common Value", mcv_entropy, true);

        // Section 6.3.2 - Collision Test (bit strings only)
        double collision_entropy = -1.0;
        if ((dp.alph_size > 2) || !initial_entropy) {
            ret_min_entropy = collision_test(dp.bsymbols, dp.blen, verbose, "Bitstring");
            H_bitstring = std::min(ret_min_entropy, H_bitstring);
            collision_entropy = ret_min_entropy;
        }
        if (initial_entropy && (dp.alph_size == 2)) {
            ret_min_entropy = collision_test(dp.symbols, dp.len, verbose, "Literal");
            H_original = std::min(ret_min_entropy, H_original);
            collision_entropy = ret_min_entropy;
        }
        add_estimator(result, "Collision Test", collision_entropy, true);

        // Section 6.3.3 - Markov Test (bit strings only)
        double markov_entropy = -1.0;
        if ((dp.alph_size > 2) || !initial_entropy) {
            ret_min_entropy = markov_test(dp.bsymbols, dp.blen, verbose, "Bitstring");
            H_bitstring = std::min(ret_min_entropy, H_bitstring);
            markov_entropy = ret_min_entropy;
        }
        if (initial_entropy && (dp.alph_size == 2)) {
            ret_min_entropy = markov_test(dp.symbols, dp.len, verbose, "Literal");
            H_original = std::min(ret_min_entropy, H_original);
            markov_entropy = ret_min_entropy;
        }
        add_estimator(result, "Markov Test", markov_entropy, true);

        // Section 6.3.4 - Compression Test (bit strings only)
        double compression_entropy = -1.0;
        if ((dp.alph_size > 2) || !initial_entropy) {
            ret_min_entropy = compression_test(dp.bsymbols, dp.blen, verbose, "Bitstring");
            if (ret_min_entropy >= 0) {
                H_bitstring = std::min(ret_min_entropy, H_bitstring);
                compression_entropy = ret_min_entropy;
            }
        }
        if (initial_entropy && (dp.alph_size == 2)) {
            ret_min_entropy = compression_test(dp.symbols, dp.len, verbose, "Literal");
            if (ret_min_entropy >= 0) {
                H_original = std::min(ret_min_entropy, H_original);
                compression_entropy = ret_min_entropy;
            }
        }
        add_estimator(result, "Compression Test", compression_entropy, compression_entropy >= 0);

        // Section 6.3.5 - t-Tuple Test
        // Section 6.3.6 - LRS Test
        double bin_t_tuple_res = -1.0, bin_lrs_res = -1.0;
        double t_tuple_res = -1.0, lrs_res = -1.0;
        double t_tuple_entropy = -1.0, lrs_entropy = -1.0;

        if ((dp.alph_size > 2) || !initial_entropy) {
            SAalgs(dp.bsymbols, dp.blen, 2, bin_t_tuple_res, bin_lrs_res, verbose, "Bitstring");
            if (bin_t_tuple_res >= 0.0) {
                H_bitstring = std::min(bin_t_tuple_res, H_bitstring);
                t_tuple_entropy = bin_t_tuple_res;
            }
            if (bin_lrs_res >= 0.0) {
                H_bitstring = std::min(bin_lrs_res, H_bitstring);
                lrs_entropy = bin_lrs_res;
            }
        }

        if (initial_entropy) {
            SAalgs(dp.symbols, dp.len, dp.alph_size, t_tuple_res, lrs_res, verbose, "Literal");
            if (t_tuple_res >= 0.0) {
                H_original = std::min(t_tuple_res, H_original);
                t_tuple_entropy = t_tuple_res;
            }
            if (lrs_res >= 0.0) {
                H_original = std::min(lrs_res, H_original);
                lrs_entropy = lrs_res;
            }
        }
        add_estimator(result, "t-Tuple Test", t_tuple_entropy, t_tuple_entropy >= 0);
        add_estimator(result, "LRS Test", lrs_entropy, lrs_entropy >= 0);

        // Section 6.3.7 - MultiMCW Test
        double mcw_entropy = -1.0;
        if ((dp.alph_size > 2) || !initial_entropy) {
            ret_min_entropy = multi_mcw_test(dp.bsymbols, dp.blen, 2, verbose, "Bitstring");
            if (ret_min_entropy >= 0) {
                H_bitstring = std::min(ret_min_entropy, H_bitstring);
                mcw_entropy = ret_min_entropy;
            }
        }
        if (initial_entropy) {
            ret_min_entropy = multi_mcw_test(dp.symbols, dp.len, dp.alph_size, verbose, "Literal");
            if (ret_min_entropy >= 0) {
                H_original = std::min(ret_min_entropy, H_original);
                mcw_entropy = ret_min_entropy;
            }
        }
        add_estimator(result, "Multi Most Common in Window Test", mcw_entropy, mcw_entropy >= 0);

        // Section 6.3.8 - Lag Prediction Test
        double lag_entropy = -1.0;
        if ((dp.alph_size > 2) || !initial_entropy) {
            ret_min_entropy = lag_test(dp.bsymbols, dp.blen, 2, verbose, "Bitstring");
            if (ret_min_entropy >= 0) {
                H_bitstring = std::min(ret_min_entropy, H_bitstring);
                lag_entropy = ret_min_entropy;
            }
        }
        if (initial_entropy) {
            ret_min_entropy = lag_test(dp.symbols, dp.len, dp.alph_size, verbose, "Literal");
            if (ret_min_entropy >= 0) {
                H_original = std::min(ret_min_entropy, H_original);
                lag_entropy = ret_min_entropy;
            }
        }
        add_estimator(result, "Lag Prediction Test", lag_entropy, lag_entropy >= 0);

        // Section 6.3.9 - MultiMMC Test
        double mmc_entropy = -1.0;
        if ((dp.alph_size > 2) || !initial_entropy) {
            ret_min_entropy = multi_mmc_test(dp.bsymbols, dp.blen, 2, verbose, "Bitstring");
            if (ret_min_entropy >= 0) {
                H_bitstring = std::min(ret_min_entropy, H_bitstring);
                mmc_entropy = ret_min_entropy;
            }
        }
        if (initial_entropy) {
            ret_min_entropy = multi_mmc_test(dp.symbols, dp.len, dp.alph_size, verbose, "Literal");
            if (ret_min_entropy >= 0) {
                H_original = std::min(ret_min_entropy, H_original);
                mmc_entropy = ret_min_entropy;
            }
        }
        add_estimator(result, "Multi Markov Model with Counting Test", mmc_entropy, mmc_entropy >= 0);

        // Section 6.3.10 - LZ78Y Test
        double lz78y_entropy = -1.0;
        if ((dp.alph_size > 2) || !initial_entropy) {
            ret_min_entropy = LZ78Y_test(dp.bsymbols, dp.blen, 2, verbose, "Bitstring");
            if (ret_min_entropy >= 0) {
                H_bitstring = std::min(ret_min_entropy, H_bitstring);
                lz78y_entropy = ret_min_entropy;
            }
        }
        if (initial_entropy) {
            ret_min_entropy = LZ78Y_test(dp.symbols, dp.len, dp.alph_size, verbose, "Literal");
            if (ret_min_entropy >= 0) {
                H_original = std::min(ret_min_entropy, H_original);
                lz78y_entropy = ret_min_entropy;
            }
        }
        add_estimator(result, "LZ78Y Test", lz78y_entropy, lz78y_entropy >= 0);

        // Calculate assessed entropy
        // Following NIST SP800-90B Section 3.1.3 (non_iid_main.cpp lines 491-496)
        double h_assessed = dp.word_size;
        if ((dp.alph_size > 2) || !initial_entropy) {
            h_assessed = std::min(h_assessed, H_bitstring * dp.word_size);
        }
        if (initial_entropy) {
            h_assessed = std::min(h_assessed, H_original);
        }

        // Set results
        result->h_original = H_original;
        result->h_bitstring = H_bitstring;
        result->h_assessed = h_assessed;
        result->min_entropy = h_assessed;
        result->data_word_size = dp.word_size;
        result->error_code = 0;

        // guard destructor calls free_data(&dp) automatically

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
