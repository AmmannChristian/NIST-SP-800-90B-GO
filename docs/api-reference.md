# NIST SP 800-90B Service API Reference

## 1. Overview

The NIST SP 800-90B service exposes its entropy assessment capabilities through two interfaces: a gRPC API for programmatic integration and a command-line interface for batch processing. Both interfaces delegate to the same underlying assessment engine and produce equivalent results for identical inputs.

This document specifies the gRPC service contract, HTTP endpoints, CLI usage, Prometheus metrics, and the internal Go package interfaces.

## 2. gRPC API

### 2.1 Service Definition

The service is defined in `api/nist/v1/nist_sp800_90b.proto` under the package `nist.sp800_90b.v1`.

```
service Sp80090bAssessmentService {
  rpc AssessEntropy(Sp80090bAssessmentRequest) returns (Sp80090bAssessmentResponse);
}
```

The service registers a single RPC method. When the gRPC listener is enabled (`GRPC_ENABLED=true`), the server also registers the standard gRPC health check service (`grpc.health.v1.Health`) and gRPC reflection for service discovery.

### 2.2 AssessEntropy

Performs an entropy assessment on the provided data samples according to NIST SP 800-90B.

**Full Method Name**: `/nist.sp800_90b.v1.Sp80090bAssessmentService/AssessEntropy`

#### 2.2.1 Request Message

```
message Sp80090bAssessmentRequest {
  bytes  data            = 1;
  uint32 bits_per_symbol = 2;
  bool   iid_mode        = 3;
  bool   non_iid_mode    = 4;
  uint32 verbosity       = 5;
}
```

| Field | Type | Required | Constraints | Description |
|---|---|---|---|---|
| `data` | `bytes` | Yes | Non-empty; max `MAX_UPLOAD_SIZE` (default 100 MB) | Raw entropy source samples packed as bytes |
| `bits_per_symbol` | `uint32` | Yes | 0-8 | Bits per symbol. A value of 0 triggers auto-detection based on the highest set bit across all samples |
| `iid_mode` | `bool` | Conditional | At least one of `iid_mode` or `non_iid_mode` must be true | Enable IID statistical tests (Most Common Value, Chi-Square, LRS, Permutation) |
| `non_iid_mode` | `bool` | Conditional | At least one of `iid_mode` or `non_iid_mode` must be true | Enable Non-IID estimators (10 estimators from Section 6.3) |
| `verbosity` | `uint32` | No | 0-3 | Controls logging verbosity: 0 = quiet, 1 = normal, 2 = verbose, 3 = debug |

#### 2.2.2 Response Message

```
message Sp80090bAssessmentResponse {
  double                          min_entropy        = 1;
  repeated Sp80090bEstimatorResult iid_results       = 2;
  repeated Sp80090bEstimatorResult non_iid_results   = 3;
  bool                            passed             = 4;
  string                          assessment_summary = 5;
  uint64                          sample_count       = 6;
  uint32                          bits_per_symbol    = 7;
}
```

| Field | Type | Description |
|---|---|---|
| `min_entropy` | `double` | Overall minimum entropy estimate in bits per sample. When both modes are enabled, this is the minimum across IID and Non-IID results. Falls back to 0.0 if all estimators produce infinity |
| `iid_results` | `repeated Sp80090bEstimatorResult` | Results from IID tests. Empty if `iid_mode` was false |
| `non_iid_results` | `repeated Sp80090bEstimatorResult` | Results from Non-IID estimators. Empty if `non_iid_mode` was false |
| `passed` | `bool` | Assessment completion status |
| `assessment_summary` | `string` | Human-readable summary |
| `sample_count` | `uint64` | Number of bytes in the assessed sample |
| `bits_per_symbol` | `uint32` | Actual bits per symbol used (may differ from request if auto-detected) |

#### 2.2.3 Estimator Result Message

```
message Sp80090bEstimatorResult {
  string              name             = 1;
  double              entropy_estimate = 2;
  bool                passed           = 3;
  map<string, double> details          = 4;
  string              description      = 5;
}
```

| Field | Type | Description |
|---|---|---|
| `name` | `string` | Estimator or test name |
| `entropy_estimate` | `double` | Entropy estimate in bits per sample. Set to -1.0 for statistical tests that produce pass/fail results without an entropy estimate |
| `passed` | `bool` | Whether the test or estimator passed |
| `details` | `map<string, double>` | Estimator-specific numeric details. For entropy estimators, includes `entropy_estimate` as a key-value pair |
| `description` | `string` | Human-readable description indicating whether the result is an "entropy estimator" or a "statistical test" |

#### 2.2.4 IID Estimators

When `iid_mode` is true, the following tests are executed:

| Name | Type | Description |
|---|---|---|
| Most Common Value | Entropy estimator | Estimates min-entropy from the frequency of the most common symbol |
| Chi-Square Tests | Statistical test | Independence test; pass/fail only, no entropy estimate |
| Length of Longest Repeated Substring Test | Statistical test | LRS-based independence test; pass/fail only |
| Permutation Tests | Statistical test | Tests for non-randomness via permutation analysis; pass/fail only |

#### 2.2.5 Non-IID Estimators

When `non_iid_mode` is true, the following ten estimators from NIST SP 800-90B Section 6.3 are executed:

| Name | Section | Description |
|---|---|---|
| Most Common Value | 6.3.1 | Frequency-based min-entropy estimate |
| Collision Test | 6.3.2 | Time-to-first-collision entropy estimate |
| Markov Test | 6.3.3 | First-order Markov chain entropy estimate |
| Compression Test | 6.3.4 | Maurer universal statistic entropy estimate |
| t-Tuple Test | 6.3.5 | Suffix array based tuple frequency estimate |
| LRS Test | 6.3.6 | Longest repeated substring entropy estimate |
| Multi Most Common in Window Test | 6.3.7 | Sliding window frequency prediction |
| Lag Prediction Test | 6.3.8 | Lag-based prediction entropy estimate |
| Multi Markov Model with Counting Test | 6.3.9 | Multi-order Markov model estimate |
| LZ78Y Test | 6.3.10 | LZ78 variant dictionary compression estimate |

For each estimator, when `bits_per_symbol` exceeds 2, both the literal (original alphabet) and bitstring (binary expansion) representations are assessed. The final assessed entropy is the conservative minimum: `min(H_original, H_bitstring * word_size)`.

#### 2.2.6 Error Responses

Errors are returned as standard gRPC status codes.

| Condition | gRPC Code | Message Pattern |
|---|---|---|
| Nil request | `INVALID_ARGUMENT` | `request cannot be nil` |
| Empty data | `INVALID_ARGUMENT` | `data cannot be empty` |
| `bits_per_symbol` > 8 | `INVALID_ARGUMENT` | `bits_per_symbol must be between 0 and 8, got N` |
| Neither mode selected | `INVALID_ARGUMENT` | `either iid_mode or non_iid_mode must be enabled` |
| IID assessment failure | `INVALID_ARGUMENT` | `IID assessment failed: ...` |
| Non-IID assessment failure | `INVALID_ARGUMENT` | `Non-IID assessment failed: ...` |

#### 2.2.7 Response Metadata

Each response includes the following gRPC metadata header:

| Header | Value | Description |
|---|---|---|
| `x-request-id` | UUID v4 | Unique identifier for request tracing |

#### 2.2.8 Example: grpcurl

```bash
DATA_BASE64=$(base64 -w0 data.bin)
grpcurl -plaintext \
  -d '{"data":"'"$DATA_BASE64"'","bits_per_symbol":8,"non_iid_mode":true}' \
  localhost:9090 nist.sp800_90b.v1.Sp80090bAssessmentService/AssessEntropy
```

## 3. HTTP Endpoints

The HTTP server is bound to `SERVER_HOST:SERVER_PORT` (default `0.0.0.0:9091`) when `METRICS_ENABLED=true`.

### 3.1 Health Check

| Property | Value |
|---|---|
| Path | `/health` |
| Method | `GET` |
| Content-Type | `application/json` |

**Response Body**:

```json
{
  "status": "healthy",
  "version": "1.0.0"
}
```

Non-GET requests return HTTP 405 Method Not Allowed.

### 3.2 Prometheus Metrics

| Property | Value |
|---|---|
| Path | `/metrics` |
| Method | `GET` |
| Content-Type | `text/plain; version=0.0.4` |
| Handler | `promhttp.Handler()` |

Returns all registered Prometheus metrics in the standard exposition format.

### 3.3 gRPC Health Check

The standard gRPC health check protocol is registered when `GRPC_ENABLED=true`.

| Service Name | Status |
|---|---|
| (empty string) | `SERVING` |
| `nist.sp800_90b.v1.Sp80090bAssessmentService` | `SERVING` |

```bash
grpcurl -plaintext localhost:9090 grpc.health.v1.Health/Check
```

## 4. Command-Line Interface

The `ea_tool` binary provides a batch-mode assessment interface.

### 4.1 Synopsis

```
ea_tool [options] <file>
```

When no file argument is provided, data is read from standard input.

### 4.2 Options

| Flag | Type | Default | Description |
|---|---|---|---|
| `-iid` | bool | `false` | Run IID tests |
| `-non-iid` | bool | `false` | Run Non-IID estimators |
| `-bits` | int | `0` | Bits per symbol (1-8); 0 for auto-detect |
| `-verbose` | int | `1` | Verbosity (0 = quiet, 1 = normal, 2 = verbose, 3 = very verbose) |
| `-output` | string | (empty) | JSON output file path |
| `-version` | bool | `false` | Print version and exit |

Exactly one of `-iid` or `-non-iid` must be specified. Specifying both or neither produces an error.

### 4.3 Exit Codes

| Code | Meaning |
|---|---|
| 0 | Successful assessment |
| 1 | Assessment error (data processing failure, C++ error) |
| 2 | Argument validation error |

### 4.4 JSON Output Format

When `-output` is specified, results are written as indented JSON.

```json
{
  "version": "1.0.0",
  "filename": "data.bin",
  "test_type": "Non-IID",
  "bits_per_symbol": 8,
  "data_size": 1000000,
  "min_entropy": 6.5,
  "h_original": 6.6,
  "h_bitstring": 6.1,
  "h_assessed": 6.5,
  "error_code": 0
}
```

| Field | Type | Description |
|---|---|---|
| `version` | string | Tool version |
| `filename` | string | Input filename or `"stdin"` |
| `test_type` | string | `"IID"` or `"Non-IID"` |
| `bits_per_symbol` | int | Requested bits per symbol |
| `data_size` | int | Input data size in bytes |
| `min_entropy` | float | Minimum entropy estimate |
| `h_original` | float | Original-alphabet entropy (omitted if zero) |
| `h_bitstring` | float | Bitstring entropy (omitted if zero) |
| `h_assessed` | float | Assessed (final) entropy |
| `error_code` | int | 0 for success, 1 for error |
| `error_message` | string | Error description (present only on error) |

### 4.5 Examples

```bash
# Non-IID assessment with 8 bits per symbol
./build/ea_tool -non-iid -bits 8 data.bin

# IID assessment with auto-detect, JSON output
./build/ea_tool -iid -bits 0 data.bin -output result.json

# Read from stdin
cat data.bin | ./build/ea_tool -non-iid -bits 8
```

## 5. Prometheus Metrics Reference

All metrics use the `entropy_` prefix and are automatically registered via `promauto`.

### 5.1 entropy_requests_total

| Property | Value |
|---|---|
| Type | Counter |
| Labels | `test_type` (IID, Non-IID, mixed) |
| Description | Total number of entropy assessment requests received |

### 5.2 entropy_duration_seconds

| Property | Value |
|---|---|
| Type | Histogram |
| Labels | `test_type` |
| Buckets | Exponential: 0.01, 0.02, 0.04, 0.08, 0.16, 0.32, 0.64, 1.28, 2.56, 5.12 |
| Description | Duration of entropy assessments in seconds |

### 5.3 entropy_errors_total

| Property | Value |
|---|---|
| Type | Counter |
| Labels | `test_type`, `error_type` |
| Description | Total number of assessment errors by type |

### 5.4 entropy_data_size_bytes

| Property | Value |
|---|---|
| Type | Histogram |
| Labels | `test_type` |
| Buckets | Exponential: 1024, 10240, 102400, 1024000, 10240000, 102400000 |
| Description | Size of assessed data payloads in bytes |

### 5.5 entropy_min_entropy_value

| Property | Value |
|---|---|
| Type | Histogram |
| Labels | `test_type` |
| Buckets | Linear: 0.0, 0.5, 1.0, 1.5, 2.0, 2.5, 3.0, 3.5, 4.0, 4.5, 5.0, 5.5, 6.0, 6.5, 7.0, 7.5, 8.0 |
| Description | Distribution of computed min-entropy values |

## 6. Go Package Interface

### 6.1 entropy Package

The `internal/entropy` package provides the core assessment types and functions.

#### Assessment

```go
type Assessment struct { /* unexported fields */ }

func NewAssessment() *Assessment
func (a *Assessment) SetVerbose(level int)
func (a *Assessment) GetVerbose() int
func (a *Assessment) AssessIID(data []byte, bitsPerSymbol int) (*Result, error)
func (a *Assessment) AssessNonIID(data []byte, bitsPerSymbol int) (*Result, error)
func (a *Assessment) AssessFile(filename string, bitsPerSymbol int, testType TestType) (*Result, error)
func (a *Assessment) AssessReader(r io.Reader, bitsPerSymbol int, testType TestType) (*Result, error)
```

#### Result

```go
type Result struct {
    MinEntropy   float64           // Final min-entropy (= HAssessed)
    HOriginal    float64           // Original-alphabet entropy
    HBitstring   float64           // Bitstring entropy
    HAssessed    float64           // Assessed entropy: min(HOriginal, HBitstring * word_size)
    DataWordSize int               // Bits per symbol used
    TestType     TestType          // IID or NonIID
    Estimators   []EstimatorResult // Per-estimator results
}
```

#### TestType

```go
type TestType int
const (
    IID    TestType = iota  // Independent and Identically Distributed
    NonIID                  // Non-IID
)
```

#### Sentinel Errors

| Error | Description |
|---|---|
| `ErrInvalidData` | Input data is nil, empty, or malformed |
| `ErrInvalidBitsPerSymbol` | `bits_per_symbol` is outside the valid range (1-8) |
| `ErrInsufficientData` | Sample size is below the minimum for reliable estimation |
| `ErrCFunction` | The underlying C library returned an error |
| `ErrMemoryAllocation` | Memory allocation failed in the C layer |

All errors are wrapped in `EntropyError`, which implements `Unwrap()` for use with `errors.Is()`.

### 6.2 service Package

```go
type EntropyService struct { /* unexported fields */ }

func NewService() *EntropyService
func (s *EntropyService) SetVerbose(level int)
func (s *EntropyService) AssessIID(data []byte, bitsPerSymbol int) (*entropy.Result, error)
func (s *EntropyService) AssessNonIID(data []byte, bitsPerSymbol int) (*entropy.Result, error)
```

```go
type GRPCServer struct { /* embeds UnimplementedSp80090BAssessmentServiceServer */ }

func NewGRPCServer(svc *EntropyService) *GRPCServer
func (s *GRPCServer) AssessEntropy(ctx context.Context, req *pb.Sp80090BAssessmentRequest) (*pb.Sp80090BAssessmentResponse, error)
```

### 6.3 config Package

```go
type Config struct {
    ServerPort     int
    ServerHost     string
    GRPCEnabled    bool
    GRPCPort       int
    TLSEnabled     bool
    TLSCertFile    string
    TLSKeyFile     string
    TLSCAFile      string
    TLSClientAuth  string
    TLSMinVersion  string
    LogLevel       string
    MaxUploadSize  int64
    Timeout        time.Duration
    MetricsEnabled bool
    AuthEnabled    bool
    AuthIssuer     string
    AuthAudience   string
    AuthJWKSURL    string
}

func LoadConfig() (*Config, error)
func (c *Config) Validate() error
func (c *Config) TLSClientAuthType() (tls.ClientAuthType, error)
func (c *Config) TLSMinVersionValue() (uint16, error)
```

### 6.4 metrics Package

```go
func RecordRequest(testType string)
func RecordDuration(testType string, duration float64)
func RecordError(testType, errorType string)
func RecordDataSize(testType string, sizeBytes int)
func RecordMinEntropy(testType string, value float64)
```

### 6.5 middleware Package

```go
func UnaryRequestIDInterceptor() grpc.UnaryServerInterceptor
func GetRequestID(ctx context.Context) string
```

## 7. C API Reference

The C-linkage API defined in `internal/nist/wrapper/wrapper.h` is consumed exclusively by the CGO bridge. It is documented here for completeness.

### 7.1 Data Structures

```c
#define MAX_ESTIMATORS 16

typedef struct {
    char   name[64];
    double entropy_estimate;  // -1.0 if not applicable
    bool   passed;
    bool   is_entropy_valid;
} EstimatorResult;

typedef struct {
    double          min_entropy;
    double          h_original;
    double          h_bitstring;
    double          h_assessed;
    int             data_word_size;
    int             error_code;       // 0 = success, -1 = validation, -2 = exception
    char            error_message[512];
    EstimatorResult estimators[MAX_ESTIMATORS];
    int             estimator_count;
} EntropyResult;
```

### 7.2 Functions

```c
EntropyResult* calculate_iid_entropy(
    const uint8_t* data, size_t length,
    int bits_per_symbol, bool is_binary, int verbose
);

EntropyResult* calculate_non_iid_entropy(
    const uint8_t* data, size_t length,
    int bits_per_symbol, bool is_binary, int verbose
);

void free_entropy_result(EntropyResult* result);
```

**Parameters**:
- `data`: Pointer to raw sample bytes.
- `length`: Number of bytes in the data array.
- `bits_per_symbol`: Symbol width in bits (1-8), or 0 for auto-detection.
- `is_binary`: When true, operate in initial-entropy mode (unconditioned source). This parameter controls whether estimators run on the literal symbol alphabet, the bitstring representation, or both.
- `verbose`: Logging verbosity level (0-3).

**Return Value**: Heap-allocated `EntropyResult` pointer. The caller must invoke `free_entropy_result` to release the memory. Returns `NULL` only on malloc failure.

**Error Codes**:
- `0`: Success
- `-1`: Input validation failure (empty data, invalid parameters, single-symbol alphabet)
- `-2`: C++ exception caught at the wrapper boundary