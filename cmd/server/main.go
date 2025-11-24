package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/AmmannChristian/nist-800-90b/internal/config"
	"github.com/AmmannChristian/nist-800-90b/internal/middleware"
	"github.com/AmmannChristian/nist-800-90b/internal/service"
	pb "github.com/AmmannChristian/nist-800-90b/pkg/pb"
)

const (
	version = "1.0.0"
)

type server struct {
	config *config.Config
	mux    *http.ServeMux
}

func main() {
	if err := run(); err != nil {
		log.Fatal().Err(err).Msg("server error")
	}
}

func run() error {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	setupLogging(cfg.LogLevel)

	log.Info().
		Str("version", version).
		Int("http_port", cfg.ServerPort).
		Int("grpc_port", cfg.GRPCPort).
		Bool("grpc_enabled", cfg.GRPCEnabled).
		Int64("max_upload_bytes", cfg.MaxUploadSize).
		Msg("starting SP800-90B entropy assessment server")

	// Create server
	srv := &server{
		config: cfg,
		mux:    http.NewServeMux(),
	}

	// Channel to listen for errors coming from the listener
	serverErrors := make(chan error, 2)

	// Optional gRPC server
	var grpcServer *grpc.Server
	var grpcListener net.Listener
	if cfg.GRPCEnabled {
		grpcListener, err = net.Listen("tcp", fmt.Sprintf("%s:%d", cfg.ServerHost, cfg.GRPCPort))
		if err != nil {
			return fmt.Errorf("failed to create gRPC listener: %w", err)
		}

		grpcServer = grpc.NewServer(
			grpc.ChainUnaryInterceptor(
				middleware.UnaryRequestIDInterceptor(),
				loggingInterceptor,
			),
		)

		pb.RegisterEntropyServiceServer(grpcServer, service.NewGRPCServer(service.NewService()))
		reflection.Register(grpcServer)

		go func() {
			log.Info().Str("addr", grpcListener.Addr().String()).Msg("gRPC server listening")
			if err := grpcServer.Serve(grpcListener); err != nil && err != grpc.ErrServerStopped {
				serverErrors <- err
			}
		}()
	}

	// Channel to listen for interrupt signal
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Optional metrics/health server on HTTP
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

	// Block until we receive a signal or error
	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)
	case sig := <-shutdown:
		log.Info().Str("signal", sig.String()).Msg("shutdown requested")

		// Give outstanding requests a deadline for completion
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Gracefully shutdown the server
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

func (s *server) registerRoutes() {
	// Health check
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

	// Metrics endpoint
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
