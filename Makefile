# Makefile for SP800-90B Go Microservice

.PHONY: all build build-arm64 run clean test test-ci tests test-cover test-race cover cover-html cover-threshold coverage-ci coverage deps dev fmt fmt-fix fmt-check lint staticcheck gosec govulncheck vet tools tools-update help docker-build build-nist build-go bench bench-baseline bench-compare

# ========================================
# Variables
# ========================================
BINARY_NAME=sp800-90b-entropy
BUILD_DIR=build
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
PROTO_DIR=api/nist/v1
PB_DIR=pkg/pb

TEST_TAGS ?= teststub
GOTESTFLAGS ?= -count=1 -timeout=15m -tags=$(TEST_TAGS)
RACE_TESTFLAGS ?= -count=1 -timeout=10m
UNIT_PKGS ?= ./internal/... ./cmd/...
# Coverage focuses on internal logic + CLI; gRPC server integration is covered separately.
COVER_PKGS ?= ./internal/... ./cmd/ea_tool
UNIT_SHUFFLE ?= on
RACE_SHUFFLE ?= on
COVER_SHUFFLE ?= off
JUNIT_FILE ?=
COVERAGE_MIN ?= 90
COVERMODE ?= atomic
COVERPROFILE ?= $(BUILD_DIR)/coverage.out

DEV_TOOLS=\
	github.com/golangci/golangci-lint/cmd/golangci-lint \
	honnef.co/go/tools/cmd/staticcheck \
	golang.org/x/vuln/cmd/govulncheck \
	golang.org/x/tools/cmd/goimports \
	mvdan.cc/gofumpt \
	github.com/securego/gosec/v2/cmd/gosec \
	gotest.tools/gotestsum@v1.13.0 \
	google.golang.org/protobuf/cmd/protoc-gen-go \
	google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

FMT_FIND := find . -type f -name '*.go' -not -path './pkg/pb/*' -not -path './$(BUILD_DIR)/*' -not -path './.cache/*'

# Auto-add GOPATH/bin to PATH
GOBIN := $(shell go env GOPATH)/bin
export PATH := $(GOBIN):$(PATH)

# ========================================
# Default target
# ========================================
all: proto build

# ========================================
# Build for local development
# ========================================
# ========================================
# Generate protobuf code
# ========================================
proto:
	@set -eu; \
	echo "Generating protobuf code..."; \
	mkdir -p $(PB_DIR); \
	protoc \
	  -I $(PROTO_DIR) \
	  --go_out=$(PB_DIR) --go_opt=paths=source_relative \
	  --go-grpc_out=$(PB_DIR) --go-grpc_opt=paths=source_relative \
	  $(PROTO_DIR)/service.proto; \
	echo "Protobuf generation complete"; \
	find $(PB_DIR) -maxdepth 5 -type f -name '*.pb.go' -print

# ========================================
# Build for local development
# ========================================
build: proto build-nist
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 go build -o $(BUILD_DIR)/ea_tool ./cmd/ea_tool
	CGO_ENABLED=1 go build -o $(BUILD_DIR)/server ./cmd/server
	@echo "Build complete: $(BUILD_DIR)/{ea_tool,server}"

build-go: build

# ========================================
# Build for ARM64
# ========================================
build-arm64: proto build-nist
	@echo "Building for ARM64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm64 CGO_ENABLED=1 go build -o $(BUILD_DIR)/ea_tool-arm64 ./cmd/ea_tool
	GOOS=linux GOARCH=arm64 CGO_ENABLED=1 go build -o $(BUILD_DIR)/server-arm64 ./cmd/server
	@echo "ARM64 build complete: $(BUILD_DIR)/{ea_tool-arm64,server-arm64}"

# ========================================
# Build NIST C++ library
# ========================================
build-nist:
	@echo "Building NIST C++ library..."
	$(MAKE) -C internal/nist

# ========================================
# Run locally
# ========================================
run: build
	@echo "Running $(BINARY_NAME) server..."
	./$(BUILD_DIR)/server

# ========================================
# Development mode
# ========================================
dev: build-nist
	@echo "Starting development mode..."
	CGO_ENABLED=1 go run ./cmd/server

# ========================================
# Clean build artifacts
# ========================================
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)/
	rm -f $(PB_DIR)/*.pb.go
	$(MAKE) -C internal/nist clean
	rm -f coverage.out coverage.html
	find . -name "*.test" -type f -delete
	@echo "Clean complete"

# ========================================
# Run tests
# ========================================
test: build-nist
	@echo "Running tests..."
	CGO_ENABLED=1 go test $(GOTESTFLAGS) -shuffle=$(UNIT_SHUFFLE) ./...

# Deterministic test run for CI
test-ci: build-nist
	@echo "Running CI unit tests (packages: $(UNIT_PKGS))..."
	@echo "Shuffle: $(UNIT_SHUFFLE)"
	@if [ -n "$(JUNIT_FILE)" ] && command -v gotestsum >/dev/null; then \
		mkdir -p $(dir $(JUNIT_FILE)); \
		CGO_ENABLED=1 gotestsum --junitfile $(JUNIT_FILE) --format testname -- \
			$(GOTESTFLAGS) -shuffle=$(UNIT_SHUFFLE) $(UNIT_PKGS); \
	else \
		CGO_ENABLED=1 go test $(GOTESTFLAGS) -shuffle=$(UNIT_SHUFFLE) $(UNIT_PKGS); \
	fi

# Alias for convenience
tests: test

# Run tests with coverage
test-cover:
	@$(MAKE) coverage

# Run tests with race detector
test-race: build-nist
	@echo "Running tests with race detector..."
	CGO_ENABLED=1 go test $(RACE_TESTFLAGS) -race -shuffle=$(RACE_SHUFFLE) $(UNIT_PKGS)

# ========================================
# Coverage
# ========================================
cover:
	@$(MAKE) coverage-ci
	@$(MAKE) cover-threshold

cover-html:
	@go tool cover -html=$(COVERPROFILE) -o $(BUILD_DIR)/coverage.html
	@echo "Coverage HTML report: $(BUILD_DIR)/coverage.html"

cover-threshold:
	@echo "Checking total coverage ≥ $(COVERAGE_MIN)%..."
	@test -f $(COVERPROFILE) || { echo "Coverage profile $(COVERPROFILE) not found; run 'make coverage-ci' first."; exit 1; }
	@total=$$(go tool cover -func=$(COVERPROFILE) | awk '/^total:/ {print $$3}'); \
	awk -v cov=$$total -v min=$(COVERAGE_MIN) 'BEGIN { cov+=0; if (cov < min) { printf "Coverage %.2f%% < %.0f%%\n", cov, min; exit 1 } else { printf "Coverage %.2f%% ≥ %.0f%%\n", cov, min } }'

coverage-ci: build-nist
	@echo "Generating deterministic coverage profile..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 GOCACHE=$(abspath $(BUILD_DIR)/gocache) XDG_CACHE_HOME=$(abspath $(BUILD_DIR)/.cache) \
		go test $(GOTESTFLAGS) -shuffle=$(COVER_SHUFFLE) -covermode=$(COVERMODE) -coverprofile=$(COVERPROFILE) $(COVER_PKGS)
	@CGO_ENABLED=1 GOCACHE=$(abspath $(BUILD_DIR)/gocache) go tool cover -func=$(COVERPROFILE) | tail -n 1

coverage:
	@$(MAKE) coverage-ci
	@$(MAKE) cover-threshold

# ========================================
# Benchmarking
# ========================================
bench: build-nist
	@echo "Running benchmarks..."
	CGO_ENABLED=1 go test -bench=. -benchmem -benchtime=10s ./internal/entropy/

bench-all: build-nist
	@echo "Running all benchmarks with 10 iterations..."
	CGO_ENABLED=1 go test -bench=. -benchmem -benchtime=10s -count=10 ./internal/entropy/ | tee $(BUILD_DIR)/bench-current.txt

bench-compare: build-nist
	@echo "Comparing benchmarks with baseline..."
	@test -f $(BUILD_DIR)/bench-baseline.txt || { echo "No baseline found. Run 'make bench-baseline' first."; exit 1; }
	@echo "Running current benchmarks..."
	@CGO_ENABLED=1 go test -bench=. -benchmem -count=5 ./internal/entropy/ | tee $(BUILD_DIR)/bench-current.txt
	@if command -v benchstat >/dev/null 2>&1; then \
		echo ""; \
		echo "Benchmark comparison:"; \
		benchstat $(BUILD_DIR)/bench-baseline.txt $(BUILD_DIR)/bench-current.txt; \
	else \
		echo ""; \
		echo "Install benchstat for statistical comparison:"; \
		echo "  go install golang.org/x/perf/cmd/benchstat@latest"; \
	fi

bench-baseline: build-nist
	@echo "Capturing baseline benchmarks..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 go test -bench=. -benchmem -count=10 ./internal/entropy/ | tee $(BUILD_DIR)/bench-baseline.txt
	@echo ""
	@echo "Baseline saved to $(BUILD_DIR)/bench-baseline.txt"

# ========================================
# Install dependencies
# ========================================
deps:
	@echo "Installing system dependencies..."
	@if command -v apt-get > /dev/null; then \
		sudo apt-get update && \
		sudo apt-get install -y \
			g++ \
			libbz2-dev \
			libdivsufsort-dev \
			libjsoncpp-dev \
			libmpfr-dev \
			libgmp-dev \
			libssl-dev \
			make; \
	else \
		echo "Warning: apt-get not found. Please install dependencies manually:"; \
		echo "  - g++, libbz2-dev, libdivsufsort-dev, libjsoncpp-dev"; \
		echo "  - libmpfr-dev, libgmp-dev, libssl-dev, make"; \
	fi
	@echo "Installing Go dependencies..."
	go mod download
	@echo "Adding tool dependencies to go.mod..."
	@set -e; for tool in $(DEV_TOOLS); do \
		go get $$tool; \
	done
	@echo "Tidying modules..."
	go mod tidy
	@$(MAKE) tools
	@echo "Dependencies installed"

tools:
	@echo "Installing developer tools..."
	@set -e; for tool in $(DEV_TOOLS); do \
		echo "  $$tool"; \
		go install $$tool; \
	done

tools-update:
	@echo "Checking for newer tool versions..."
	@for tool in $(DEV_TOOLS); do \
		name=$${tool%@*}; \
		current=$${tool#*@}; \
		module=$$(echo $$name | sed -E 's|/cmd/.*$$||'); \
		[ -n "$$module" ] || module=$$name; \
		latest=$$(go list -m -versions $$module 2>/dev/null | tr ' ' '\n' | tail -n 1); \
		printf "%s: current=%s latest=%s\n" "$$name" "$$current" "$$latest"; \
	done

# ========================================
# Docker build
# ========================================
docker-build:
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):$(VERSION) .
	@echo "Docker build complete"

docker: docker-build

# ========================================
# Format code
# ========================================
fmt: fmt-fix

fmt-fix:
	@echo "Running gofumpt..."
	@$(FMT_FIND) -print0 | xargs -0 gofumpt -w
	@echo "Running gofmt -s..."
	@$(FMT_FIND) -print0 | xargs -0 gofmt -s -w
	@echo "Running goimports..."
	@$(FMT_FIND) -print0 | xargs -0 goimports -w
	@echo "Formatting complete"

fmt-check:
	@echo "Checking code format..."
	@if ! $(FMT_FIND) -print0 | xargs -0 gofumpt -d | grep .; then \
		echo "✓ Code is properly formatted"; \
	else \
		echo "✗ Code needs formatting. Run 'make fmt-fix'"; \
		exit 1; \
	fi

# ========================================
# Lint code
# ========================================
lint: build-nist
	@echo "Running linters..."
	@$(MAKE) staticcheck
	@$(MAKE) vet

staticcheck: build-nist
	@echo "Running staticcheck..."
	GOCACHE=$(abspath $(BUILD_DIR)/gocache) \
	XDG_CACHE_HOME=$(abspath $(BUILD_DIR)/.cache) \
	staticcheck $(UNIT_PKGS)

gosec: build-nist
	@echo "Running gosec..."
	@mkdir -p tools/ci
	GOCACHE=$(BUILD_DIR)/gocache gosec -exclude-dir=.cache -exclude-generated ./...

govulncheck: build-nist
	@echo "Running govulncheck..."
	govulncheck ./...

vet: build-nist
	@echo "Running go vet..."
	GOCACHE=$(abspath $(BUILD_DIR)/gocache) XDG_CACHE_HOME=$(abspath $(BUILD_DIR)/.cache) go vet ./...

# ========================================
# Help
# ========================================
help:
	@echo "SP800-90B Go Microservice - Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  make proto           - Generate protobuf code (outputs to $(PB_DIR))"
	@echo "  make build           - Build for local development (CGO)"
	@echo "  make build-arm64     - Build for ARM64"
	@echo "  make build-nist      - Build NIST C++ library"
	@echo "  make run             - Build and run server locally"
	@echo "  make dev             - Run in development mode"
	@echo "  make clean           - Remove build artifacts"
	@echo "  make test            - Run tests"
	@echo "  make tests           - Alias for 'make test'"
	@echo "  make test-ci         - Run tests for CI (deterministic)"
	@echo "  make test-cover      - Run tests with coverage"
	@echo "  make test-race       - Run tests with race detector"
	@echo "  make cover           - Generate coverage with threshold check"
	@echo "  make cover-html      - Render coverage HTML report"
	@echo "  make coverage        - Alias for 'make cover'"
	@echo "  make deps            - Install dependencies"
	@echo "  make fmt             - Format code"
	@echo "  make fmt-fix         - Apply gofumpt/gofmt/goimports"
	@echo "  make fmt-check       - Check code format"
	@echo "  make lint            - Run linters"
	@echo "  make vet             - Run go vet"
	@echo "  make staticcheck     - Run staticcheck"
	@echo "  make gosec           - Run security scanner"
	@echo "  make govulncheck     - Check for vulnerabilities"
	@echo "  make tools           - Install developer tools"
	@echo "  make bench           - Run benchmarks"
	@echo "  make bench-baseline  - Capture baseline benchmarks"
	@echo "  make bench-compare   - Compare with baseline"
	@echo "  make docker          - Build Docker image"
	@echo ""
