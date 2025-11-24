# NIST SP 800-90B Entropy Assessment - gRPC Microservice

![CI](https://github.com/AmmannChristian/nist-800-90b/actions/workflows/ci.yml/badge.svg)
![NIST Validation](https://github.com/AmmannChristian/nist-800-90b/actions/workflows/nist-validation.yml/badge.svg)
[![codecov](https://codecov.io/gh/AmmannChristian/nist-800-90b/branch/main/graph/badge.svg)](https://app.codecov.io/gh/AmmannChristian/nist-800-90b)

A high-performance Go service that wraps the NIST SP 800-90B entropy assessment C++ tools via CGO. Provides a gRPC API, CLI, and Prometheus metrics with deterministic builds and reproducible outputs.

CI enforces an 90% coverage threshold (internal + CLI with `-tags=teststub`); current coverage is 86.2%.

## Validation Status

| Scope | Status | Notes |
|-------|--------|-------|
| NIST C++ parity | Manual review | Validation workflow builds the original NIST tools and Go binaries, generates reference outputs, and uploads artifacts for comparison (see `.github/workflows/nist-validation.yml`). Automated diff script pending. |
| CGO bridge | Validated | Unit tests cover IID/Non-IID flows, error paths, and request instrumentation; CGO happy paths compiled in CI. |

## Quick Start

### Using Docker Compose (recommended)

```bash
# Start the service + Prometheus/Grafana stack
docker-compose up -d

# Endpoints:
# - gRPC: localhost:50051 (set GRPC_ENABLED=true in the service env to expose)
# - Metrics/health: localhost:9090
# - Prometheus: localhost:9092
# - Grafana: localhost:3000 (admin/admin)
```

### Local Build

```bash
# Install dependencies and build the NIST C++ library
make deps
make build-nist

# Build Go binaries (CGO enabled)
make build

# Run server with gRPC enabled
GRPC_ENABLED=true GRPC_PORT=50051 ./build/server
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
- **gRPC Service** (`internal/service/`, `cmd/server`): Exposes IID and Non-IID estimators with request IDs and Prometheus metrics.
- **CLI Tool** (`cmd/ea_tool`): Batch processing with IID/Non-IID modes, JSON output, and verbosity controls.
- **Observability** (`internal/metrics/`, `internal/middleware/`): Request counters, duration histograms, min-entropy gauges, and request ID propagation.
- **Configuration** (`internal/config/`): Environment-based configuration for ports, timeouts, log level, and upload limits.

## Testing

```bash
# Run unit tests (CGO enabled for integration paths)
make test

# Coverage with deterministic shuffle settings
make coverage

# Race detector
make test-race

# HTML coverage report
make cover-html
```

Coverage snapshot (COVER_PKGS=internal/..., cmd/ea_tool):

| Package | Coverage |
|---------|----------|
| internal/config | 100.0% |
| internal/metrics | 100.0% |
| internal/entropy | 86.7% |
| internal/middleware | 88.9% |
| internal/service | 84.8% |
| cmd/ea_tool | 78.9% |

## Scientific Validation

The `nist-validation.yml` workflow builds the original NIST C++ tools, runs them against bundled self-test datasets, executes the Go implementation, and uploads both outputs for comparison. An automated diff script (`tools/validate_nist_cpp_vs_go.sh`) is pending; manual review of artifacts is currently required.

## Monitoring and Observability

- Prometheus metrics at `/metrics` (enabled by default).
- Health endpoint at `/health`.
- Request IDs injected into gRPC responses via `x-request-id` metadata and logged with duration data.
- Docker Compose ships Prometheus and Grafana dashboards for immediate visibility.

## Development

### Prerequisites

- Go 1.25 or later with CGO enabled
- GCC toolchain (`g++`, `make`, `libbz2-dev`, `libdivsufsort-dev`, `libjsoncpp-dev`, `libmpfr-dev`, `libgmp-dev`, `libssl-dev`)
- `protoc` for regenerating protobufs

### Build Targets

```bash
make help          # Show all available targets
make proto         # Generate protobuf code
make build-nist    # Build NIST C++ library
make build         # Build CLI + gRPC server
make build-arm64   # Cross-compile for ARM64
make clean         # Remove build artifacts
make fmt           # Format code
make staticcheck   # Static analysis
make gosec         # Security scan
```

## License and Attribution

This project wraps the public-domain NIST SP 800-90B entropy assessment tools. See `LICENSE` for full details.
