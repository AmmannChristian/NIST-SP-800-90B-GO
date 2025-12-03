# NIST SP 800-90B Entropy Assessment - gRPC Microservice

![CI](https://github.com/AmmannChristian/NIST-SP-800-90B/actions/workflows/ci.yml/badge.svg)
![NIST Validation](https://github.com/AmmannChristian/NIST-SP-800-90B/actions/workflows/nist-validation.yml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/AmmannChristian/NIST-SP-800-90B)](https://goreportcard.com/report/github.com/AmmannChristian/NIST-SP-800-90B)
[![License](https://img.shields.io/github/license/AmmannChristian/NIST-SP-800-90B)](LICENSE)
[![codecov](https://codecov.io/gh/AmmannChristian/NIST-SP-800-90B/branch/main/graph/badge.svg)](https://app.codecov.io/gh/AmmannChristian/NIST-SP-800-90B)
[![Go Version](https://img.shields.io/github/go-mod/go-version/AmmannChristian/NIST-SP-800-90B)](go.mod)

A high-performance Go microservice wrapping the NIST SP 800-90B entropy assessment C++ toolkit via CGO. Provides a CLI, optional gRPC API, and Prometheus metrics with deterministic builds and reproducible outputs. CI enforces a 90% coverage threshold (`-tags=teststub`); current snapshot coverage is 86.2%.

## Validation Results

The service is validated against the upstream NIST reference tools ([SP800-90B_EntropyAssessment](https://github.com/usnistgov/SP800-90B_EntropyAssessment)) and the generated Go bindings:

| Scope | Status | Notes |
|-------|--------|-------|
| NIST C++ parity | Reference build + artifact comparison | `nist-validation.yml` builds the upstream C++ tools, runs them on bundled datasets, executes the Go wrappers, and uploads both outputs for manual diffing. Automated comparator script is pending. |
| CGO bridge | Unit/integration coverage | IID/Non-IID flows, error paths, and metrics hooks are exercised in unit tests; CGO happy paths are compiled in CI. |

## Quick Start

### Using Docker Compose (Recommended)

```bash
docker-compose up -d
```

- gRPC: enable with `GRPC_ENABLED=true` (default port `GRPC_PORT=50051`, mapped in `docker-compose.yml`).
- Metrics/health: `SERVER_PORT` (default 8080). Set `SERVER_PORT=9090` to match the compose port mapping.
- Prometheus: `localhost:9092`
- Grafana: `localhost:3000` (admin/admin)

### Local Build

```bash
# Install system deps and build the NIST C++ library
make deps
make build-nist

# Build Go binaries (CGO enabled)
make build

# Run the server with gRPC enabled
SERVER_PORT=8080 GRPC_ENABLED=true GRPC_PORT=50051 ./build/server
```

## Usage

### CLI

```bash
# Non-IID assessment (8 bits per symbol)
./build/ea_tool -non-iid -bits 8 data.bin

# IID assessment (auto-detect bit width)
./build/ea_tool -iid -bits 0 data.bin

# JSON output to file
./build/ea_tool -non-iid -bits 8 data.bin -output result.json
```

### gRPC API

Run the server with `GRPC_ENABLED=true`, then call via `grpcurl`:

```bash
DATA_BASE64=$(base64 -w0 data.bin)
grpcurl -plaintext \
  -d '{"data":"'"$DATA_BASE64"'","bits_per_symbol":8,"non_iid_mode":true}' \
  localhost:50051 nist.v1.EntropyService/AssessEntropy
```

## Implementation Guide

### Architecture Overview

```
nist-800-90b-test-suite/
├── api/nist/v1/          # Protobuf API definitions
├── cmd/
│   ├── ea_tool/          # CLI entry point
│   └── server/           # gRPC server + metrics/health
├── internal/
│   ├── config/           # Environment-driven configuration
│   ├── entropy/          # CGO bridge + result types
│   ├── middleware/       # Request-ID interceptor
│   ├── metrics/          # Prometheus instrumentation
│   └── nist/             # NIST C++ sources, wrapper, build assets
├── pkg/pb/               # Generated protobuf code
└── tools/                # CI utilities and scripts
```

### Core Components

- **Entropy Engine** (`internal/entropy/`): CGO bindings to the NIST SP 800-90B C++ implementation with Go-friendly result structures.
- **Service Layer** (`internal/service/`, `cmd/server`): gRPC handlers for IID and Non-IID estimators with request IDs and structured logging.
- **CLI Tool** (`cmd/ea_tool`): Batch processing with IID/Non-IID modes, JSON output, and verbosity controls.
- **Observability** (`internal/metrics/`, `internal/middleware/`): Prometheus counters/histograms, min-entropy gauges, and request ID propagation.
- **Configuration** (`internal/config/`): Environment-based configuration for ports, timeouts, log levels, and upload limits.

## Testing

### Run Unit Tests

```bash
make test
```

### Coverage Report

```bash
make coverage      # deterministic profile + threshold check (90%)
make cover-html    # open build/coverage.html
```

Coverage snapshot (`COVER_PKGS=internal/..., cmd/ea_tool`):

| Package | Coverage |
|---------|----------|
| internal/config | 100.0% |
| internal/metrics | 100.0% |
| internal/entropy | 86.7% |
| internal/middleware | 88.9% |
| internal/service | 84.8% |
| cmd/ea_tool | 78.9% |

### Race Detector

```bash
make test-race
```

### Scientific Validation

The `nist-validation.yml` workflow builds the upstream NIST C++ tools, runs them against bundled self-test datasets, executes the Go implementation, and uploads both outputs for comparison. An automated diff script (`tools/validate_nist_cpp_vs_go.sh`) is pending; manual review of artifacts is currently required.

## Performance

### Benchmarking

Benchmark targets run the CGO entropy assessments via `go test -bench` in `internal/entropy/`:

```bash
make bench          # quick run
make bench-all      # 10 iterations, writes build/bench-current.txt
make bench-baseline # save baseline to build/bench-baseline.txt
make bench-compare  # compare current vs baseline (requires benchstat)
```

### Constraints

- `bits_per_symbol` must be between 0 (auto-detect) and 8.
- Default upload cap: `MAX_UPLOAD_SIZE=100MB`.
- gRPC is optional; enable with `GRPC_ENABLED=true` and configure ports via `GRPC_PORT`/`SERVER_PORT`.

### Performance Profiling

Use Prometheus metrics to monitor latency and data sizes. For deeper investigation, run the CLI under `go test` profiles (e.g., `go test -c ./cmd/ea_tool` and execute with `-test.cpuprofile`) or wrap the service with standard `pprof` tooling during local runs.

## Monitoring and Observability

### Prometheus Metrics

Metrics are exposed at `/metrics` when `METRICS_ENABLED=true` (default):

- `entropy_requests_total` — total requests by test type
- `entropy_duration_seconds` — assessment duration histogram
- `entropy_errors_total` — error counts by type
- `entropy_data_size_bytes` — observed payload sizes
- `entropy_min_entropy_value` — distribution of computed min-entropy

Health endpoint: `/health` returns service status and version.

### Request Tracking

Each gRPC request receives a UUID `x-request-id`, is logged with duration, and is returned in response metadata for traceability.

### Structured Logging

Zerolog provides structured JSON logs with request IDs, methods, durations, and errors. Control verbosity via `LOG_LEVEL` (`debug`, `info`, `warn`, `error`).

## Attribution and License

- Licensed under the MIT License; see `LICENSE`.
- Implements the algorithms from NIST SP 800-90B.
- CGO bindings wrap the official NIST reference implementation from https://github.com/usnistgov/SP800-90B_EntropyAssessment (bundled in `internal/nist`).

## Development

### Prerequisites

- Go 1.25 or later with CGO enabled
- GCC toolchain: `g++`, `make`, `libbz2-dev`, `libdivsufsort-dev`, `libjsoncpp-dev`, `libmpfr-dev`, `libgmp-dev`, `libssl-dev`
- `protoc` for regenerating protobufs
- Docker and Docker Compose (for containerized deployment)

### Build Targets

```bash
make help          # Show all available targets
make proto         # Generate protobuf code
make build-nist    # Build NIST C++ library
make build         # Build CLI + gRPC server
make build-arm64   # Cross-compile for ARM64
make run           # Run server locally
make clean         # Remove build artifacts
make fmt           # Format code (gofumpt/gofmt/goimports)
make lint          # Static analysis (staticcheck, vet)
make gosec         # Security scan
make govulncheck   # Vulnerability check
make coverage      # Generate coverage + threshold check
make test-race     # Run tests with race detector
make docker        # Build Docker image
```

## CI/CD

- **CI (`ci.yml`)**: Protobuf generation, C++ build, Go build, static analysis, vulnerability scanning, unit tests (`-tags=teststub`), race detector, coverage threshold (90%), and artifact uploads.
- **NIST Validation (`nist-validation.yml`)**: Builds the upstream C++ reference, runs validation datasets through both C++ and Go binaries, and publishes outputs for manual comparison.

## Troubleshooting

- gRPC not reachable: ensure `GRPC_ENABLED=true` and the port in `GRPC_PORT` matches your ingress mapping.
- Build errors from CGO: install the required C++ dependencies (`make deps`) and rebuild the NIST library (`make build-nist`).
- Upload failures: adjust `MAX_UPLOAD_SIZE` (bytes) if your dataset exceeds the default 100MB limit.

## Contributing

1. Fork the repository and create a feature branch.
2. Make changes with tests; ensure `make test` and `make coverage` pass.
3. Open a pull request. Validation against the upstream NIST reference runs in CI.
