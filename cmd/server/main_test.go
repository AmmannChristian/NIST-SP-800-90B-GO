package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/AmmannChristian/nist-800-90b/internal/config"
)

func TestSetupLogging(t *testing.T) {
	orig := zerolog.GlobalLevel()
	defer zerolog.SetGlobalLevel(orig)

	cases := []struct {
		level    string
		expected zerolog.Level
	}{
		{"debug", zerolog.DebugLevel},
		{"warn", zerolog.WarnLevel},
		{"error", zerolog.ErrorLevel},
		{"unknown", zerolog.InfoLevel},
	}

	for _, tc := range cases {
		t.Run(tc.level, func(t *testing.T) {
			setupLogging(tc.level)
			assert.Equal(t, tc.expected, zerolog.GlobalLevel())
		})
	}
}

func TestRegisterRoutesHealthAndMetrics(t *testing.T) {
	srv := &server{
		config: &config.Config{
			ServerHost:     "127.0.0.1",
			ServerPort:     0,
			MetricsEnabled: true,
		},
		mux: http.NewServeMux(),
	}

	srv.registerRoutes()

	// Health GET
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Health wrong method
	req = httptest.NewRequest(http.MethodPost, "/health", nil)
	w = httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)

	// Metrics endpoint should exist
	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w = httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLoggingInterceptor(t *testing.T) {
	setupLogging("debug")

	ctx := context.Background()
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}

	// Success case
	resp, err := loggingInterceptor(ctx, "req", info, func(ctx context.Context, req interface{}) (interface{}, error) {
		time.Sleep(1 * time.Millisecond)
		return "ok", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)

	// Error case
	_, err = loggingInterceptor(ctx, "req", info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, assert.AnError
	})
	assert.Error(t, err)
}

func TestBuildUnaryInterceptors_WithoutAuth(t *testing.T) {
	cfg := &config.Config{AuthEnabled: false}

	interceptors, err := buildUnaryInterceptors(cfg)
	require.NoError(t, err)
	assert.Len(t, interceptors, 2)
}

func TestBuildUnaryInterceptors_WithOpaqueAuth(t *testing.T) {
	cfg := &config.Config{
		AuthEnabled:                   true,
		AuthIssuer:                    "https://issuer.example.com",
		AuthAudience:                  "nist-entropy",
		AuthTokenType:                 "opaque",
		AuthIntrospectionURL:          "https://issuer.example.com/oauth2/introspect",
		AuthIntrospectionClientID:     "svc-client",
		AuthIntrospectionClientSecret: "svc-secret",
	}

	interceptors, err := buildUnaryInterceptors(cfg)
	require.NoError(t, err)
	assert.Len(t, interceptors, 3)
}

func TestBuildUnaryInterceptors_WithOpaqueAuthPrivateKeyJWTPEM(t *testing.T) {
	cfg := &config.Config{
		AuthEnabled:                             true,
		AuthIssuer:                              "https://issuer.example.com",
		AuthAudience:                            "nist-entropy",
		AuthTokenType:                           "opaque",
		AuthIntrospectionURL:                    "https://issuer.example.com/oauth2/introspect",
		AuthIntrospectionAuthMethod:             "private_key_jwt",
		AuthIntrospectionClientID:               "svc-client",
		AuthIntrospectionPrivateKey:             mustGenerateRSAPrivateKeyPEM(t),
		AuthIntrospectionPrivateKeyJWTKeyID:     "kid-1",
		AuthIntrospectionPrivateKeyJWTAlgorithm: "RS256",
	}

	interceptors, err := buildUnaryInterceptors(cfg)
	require.NoError(t, err)
	assert.Len(t, interceptors, 3)
}

func TestBuildUnaryInterceptors_WithOpaqueAuthPrivateKeyJWTZitadelJSON(t *testing.T) {
	privateKeyPEM := mustGenerateRSAPrivateKeyPEM(t)
	zitadelEnvelope := map[string]string{
		"keyId":    "zitadel-kid",
		"clientId": "svc-client",
		"key":      privateKeyPEM,
	}
	zitadelKeyJSONBytes, err := json.Marshal(zitadelEnvelope)
	require.NoError(t, err)

	cfg := &config.Config{
		AuthEnabled:                 true,
		AuthIssuer:                  "https://issuer.example.com",
		AuthAudience:                "nist-entropy",
		AuthTokenType:               "opaque",
		AuthIntrospectionURL:        "https://issuer.example.com/oauth2/introspect",
		AuthIntrospectionAuthMethod: "private_key_jwt",
		AuthIntrospectionPrivateKey: string(zitadelKeyJSONBytes),
	}

	interceptors, err := buildUnaryInterceptors(cfg)
	require.NoError(t, err)
	assert.Len(t, interceptors, 3)
}

func TestRunFailsOnBadConfig(t *testing.T) {
	// Invalid port should cause config validation failure
	os.Setenv("SERVER_PORT", "-1")
	t.Cleanup(func() {
		os.Unsetenv("SERVER_PORT")
	})

	err := run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load configuration")
}

func TestRunStartsAndStopsWithSignal(t *testing.T) {
	// Find free port
	ln := mustListen(t)
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	os.Setenv("SERVER_PORT", fmt.Sprintf("%d", port))
	os.Setenv("METRICS_ENABLED", "true")
	os.Setenv("GRPC_ENABLED", "false")
	t.Cleanup(func() {
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("METRICS_ENABLED")
		os.Unsetenv("GRPC_ENABLED")
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- run()
	}()

	// allow startup
	time.Sleep(200 * time.Millisecond)

	// Send SIGTERM to self
	p, err := os.FindProcess(os.Getpid())
	require.NoError(t, err)
	require.NoError(t, p.Signal(syscall.SIGTERM))

	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatalf("run did not return in time")
	}
}

func TestRunStartsGRPCAndStops(t *testing.T) {
	httpLn := mustListen(t)
	httpPort := httpLn.Addr().(*net.TCPAddr).Port
	httpLn.Close()

	grpcLn := mustListen(t)
	grpcPort := grpcLn.Addr().(*net.TCPAddr).Port
	grpcLn.Close()

	os.Setenv("SERVER_PORT", fmt.Sprintf("%d", httpPort))
	os.Setenv("GRPC_PORT", fmt.Sprintf("%d", grpcPort))
	os.Setenv("GRPC_ENABLED", "true")
	os.Setenv("METRICS_ENABLED", "false")
	t.Cleanup(func() {
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("GRPC_PORT")
		os.Unsetenv("GRPC_ENABLED")
		os.Unsetenv("METRICS_ENABLED")
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- run()
	}()

	time.Sleep(200 * time.Millisecond)

	p, err := os.FindProcess(os.Getpid())
	require.NoError(t, err)
	require.NoError(t, p.Signal(syscall.SIGTERM))

	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatalf("run did not return in time (grpc)")
	}
}

// mustListen gives an ephemeral TCP port.
func mustListen(t *testing.T) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("cannot listen on tcp :0: %v", err)
	}
	return ln
}

func mustGenerateRSAPrivateKeyPEM(t *testing.T) string {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	privateKeyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	require.NoError(t, err)

	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyDER,
	})

	return string(privateKeyPEM)
}
