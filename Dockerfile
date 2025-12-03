# Multi-stage Dockerfile for SP800-90B Go Microservice with CGO

# Stage 1: Builder
FROM golang:1.25-bookworm AS builder

# Install C++ build dependencies
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    g++ \
    libbz2-dev \
    libdivsufsort-dev \
    libjsoncpp-dev \
    libmpfr-dev \
    libgmp-dev \
    libssl-dev \
    make \
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /build

# Copy source code
COPY . .

# Build NIST C++ library
RUN make -C internal/nist clean && make -C internal/nist

# Download Go dependencies
RUN go mod download

# Build Go binaries with CGO enabled
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o /build/bin/ea_tool ./cmd/ea_tool
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o /build/bin/server ./cmd/server

# Stage 2: Runtime
FROM debian:bookworm-slim

# Install runtime dependencies only
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    libbz2-1.0 \
    libdivsufsort3 \
    libjsoncpp25 \
    libmpfr6 \
    libgmp10 \
    libssl3 \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN useradd -m -u 1000 entropy

# Set working directory
WORKDIR /app

# Copy binaries from builder
COPY --from=builder /build/bin/ea_tool /app/
COPY --from=builder /build/bin/server /app/

# Change ownership
RUN chown -R entropy:entropy /app

# Switch to non-root user
USER entropy

# Expose HTTP port
EXPOSE 8080

# Default command: run server
CMD ["/app/server"]
