// Package main bootstraps the NIST SP 800-90B entropy assessment gRPC server.
// It provides optional TLS/mTLS, OIDC authentication, Prometheus metrics, and
// an HTTP health endpoint. The server delegates entropy computations to a CGO
// bridge that invokes the upstream NIST C++ reference implementation.
package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AmmannChristian/go-authx/grpcserver"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/AmmannChristian/nist-800-90b/internal/config"
	"github.com/AmmannChristian/nist-800-90b/internal/middleware"
	"github.com/AmmannChristian/nist-800-90b/internal/service"
	pb "github.com/AmmannChristian/nist-800-90b/pkg/pb"
)

const (
	version = "1.0.0"
)

// server holds references to the loaded configuration and the HTTP multiplexer
// used for health and metrics endpoints.
type server struct {
	config *config.Config
	mux    *http.ServeMux
}

func main() {
	if err := run(); err != nil {
		log.Fatal().Err(err).Msg("server error")
	}
}

// run initializes the server, starts gRPC and HTTP listeners, and blocks until
// a termination signal is received or a fatal error occurs. It performs a
// graceful shutdown with a 30-second deadline.
func run() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	setupLogging(cfg.LogLevel)

	log.Info().
		Str("version", version).
		Int("metrics_port", cfg.ServerPort).
		Int("grpc_port", cfg.GRPCPort).
		Bool("grpc_enabled", cfg.GRPCEnabled).
		Bool("auth_enabled", cfg.AuthEnabled).
		Int64("max_upload_bytes", cfg.MaxUploadSize).
		Msg("starting SP800-90B entropy assessment server")

	srv := &server{
		config: cfg,
		mux:    http.NewServeMux(),
	}

	serverErrors := make(chan error, 2)

	var grpcServer *grpc.Server
	var grpcListener net.Listener
	if cfg.GRPCEnabled {
		grpcListener, err = net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.ServerHost, cfg.GRPCPort))
		if err != nil {
			return fmt.Errorf("failed to create gRPC listener: %w", err)
		}

		unaryInterceptors, err := buildUnaryInterceptors(cfg)
		if err != nil {
			return fmt.Errorf("failed to configure gRPC server: %w", err)
		}

		serverOpts, err := buildGRPCServerOptions(cfg, unaryInterceptors)
		if err != nil {
			return fmt.Errorf("failed to configure gRPC server: %w", err)
		}

		grpcServer = grpc.NewServer(serverOpts...)

		pb.RegisterSp80090BAssessmentServiceServer(grpcServer, service.NewGRPCServer(service.NewService()))
		healthServer := health.NewServer()
		healthpb.RegisterHealthServer(grpcServer, healthServer)
		healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
		healthServer.SetServingStatus("nist.sp800_90b.v1.Sp80090bAssessmentService", healthpb.HealthCheckResponse_SERVING)
		reflection.Register(grpcServer)

		go func() {
			log.Info().Str("addr", grpcListener.Addr().String()).Msg("gRPC server listening")
			if err := grpcServer.Serve(grpcListener); err != nil && err != grpc.ErrServerStopped {
				serverErrors <- err
			}
		}()
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	var httpServer *http.Server
	if cfg.MetricsEnabled {
		srv.registerRoutes()
		httpServer = &http.Server{
			Addr:         fmt.Sprintf("%s:%d", cfg.ServerHost, cfg.ServerPort),
			Handler:      srv.mux,
			ReadTimeout:  cfg.Timeout,
			WriteTimeout: cfg.Timeout,
		}

		go func() {
			log.Info().Str("addr", httpServer.Addr).Msg("HTTP metrics server listening")
			if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				serverErrors <- err
			}
		}()
	}

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)
	case sig := <-shutdown:
		log.Info().Str("signal", sig.String()).Msg("shutdown requested")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if httpServer != nil {
			if err := httpServer.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
				httpServer.Close()
				return fmt.Errorf("graceful shutdown failed: %w", err)
			}
		}

		if grpcServer != nil {
			grpcServer.GracefulStop()
			if grpcListener != nil {
				_ = grpcListener.Close()
			}
		}

		log.Info().Msg("server stopped gracefully")
	}

	return nil
}

// buildUnaryInterceptors assembles the chain of gRPC unary interceptors. It
// always includes request ID injection and structured logging. When authentication
// is enabled, an OIDC token validator is appended with health-check exemptions.
// Validation supports JWT (JWKS) and opaque tokens (introspection).
func buildUnaryInterceptors(cfg *config.Config) ([]grpc.UnaryServerInterceptor, error) {
	interceptors := []grpc.UnaryServerInterceptor{
		middleware.UnaryRequestIDInterceptor(),
		loggingInterceptor,
	}

	if !cfg.AuthEnabled {
		return interceptors, nil
	}

	validatorBuilder := grpcserver.NewValidatorBuilder(cfg.AuthIssuer, cfg.AuthAudience)
	if cfg.AuthTokenType == "opaque" {
		validatorBuilder = validatorBuilder.WithOpaqueTokenIntrospection(
			cfg.AuthIntrospectionURL,
			cfg.AuthIntrospectionClientID,
			cfg.AuthIntrospectionClientSecret,
		)
	} else if cfg.AuthJWKSURL != "" {
		validatorBuilder = validatorBuilder.WithJWKSURL(cfg.AuthJWKSURL)
	}

	validator, err := validatorBuilder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build auth validator: %w", err)
	}

	log.Info().
		Str("token_type", cfg.AuthTokenType).
		Str("issuer", cfg.AuthIssuer).
		Str("audience", cfg.AuthAudience).
		Str("jwks_url", cfg.AuthJWKSURL).
		Str("introspection_url", cfg.AuthIntrospectionURL).
		Msg("gRPC authentication enabled")

	return append(interceptors, grpcserver.UnaryServerInterceptor(
		validator,
		grpcserver.WithExemptMethods(
			"/grpc.health.v1.Health/Check",
			"/grpc.health.v1.Health/Watch",
		),
	)), nil
}

// buildGRPCServerOptions constructs gRPC server options from the provided
// configuration. When TLS is enabled, it loads certificates and configures
// client authentication and minimum protocol version.
func buildGRPCServerOptions(cfg *config.Config, unaryInterceptors []grpc.UnaryServerInterceptor) ([]grpc.ServerOption, error) {
	opts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(unaryInterceptors...),
	}

	if !cfg.TLSEnabled {
		return opts, nil
	}

	clientAuth, err := cfg.TLSClientAuthType()
	if err != nil {
		return nil, fmt.Errorf("invalid TLS client auth setting: %w", err)
	}

	minVersion, err := cfg.TLSMinVersionValue()
	if err != nil {
		return nil, fmt.Errorf("invalid TLS min version: %w", err)
	}

	tlsConfig := &grpcserver.TLSConfig{
		CertFile:   cfg.TLSCertFile,
		KeyFile:    cfg.TLSKeyFile,
		CAFile:     cfg.TLSCAFile,
		ClientAuth: clientAuth,
		MinVersion: minVersion,
	}

	tlsOpt, err := grpcserver.ServerOption(tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to configure TLS: %w", err)
	}

	log.Info().
		Bool("tls_enabled", true).
		Str("cert_file", cfg.TLSCertFile).
		Str("key_file", cfg.TLSKeyFile).
		Str("ca_file", cfg.TLSCAFile).
		Str("client_auth", cfg.TLSClientAuth).
		Str("min_version", tlsVersionString(minVersion)).
		Msg("gRPC TLS enabled")

	return append(opts, tlsOpt), nil
}

// tlsVersionString returns a human-readable string for a TLS version constant.
func tlsVersionString(version uint16) string {
	switch version {
	case tls.VersionTLS13:
		return "1.3"
	case tls.VersionTLS12:
		return "1.2"
	default:
		return fmt.Sprintf("0x%x", version)
	}
}

// registerRoutes configures HTTP handlers for the /health and /metrics endpoints.
func (s *server) registerRoutes() {
	s.mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		health := map[string]interface{}{
			"status":  "healthy",
			"version": version,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(health)
	})

	s.mux.Handle("/metrics", promhttp.Handler())
}

// setupLogging configures zerolog for structured output.
func setupLogging(level string) {
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339,
	})

	switch level {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

// loggingInterceptor logs gRPC requests with timing and request ID.
func loggingInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	duration := time.Since(start)

	requestID := middleware.GetRequestID(ctx)

	if err != nil {
		log.Error().
			Err(err).
			Str("request_id", requestID).
			Str("method", info.FullMethod).
			Dur("duration", duration).
			Msg("gRPC request failed")
		return resp, err
	}

	log.Info().
		Str("request_id", requestID).
		Str("method", info.FullMethod).
		Dur("duration", duration).
		Msg("gRPC request completed")

	return resp, nil
}
