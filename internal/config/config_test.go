package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_Defaults(t *testing.T) {
	// Clear environment
	clearEnv(t)

	cfg, err := LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, 9091, cfg.ServerPort)
	assert.Equal(t, "0.0.0.0", cfg.ServerHost)
	assert.False(t, cfg.GRPCEnabled)
	assert.Equal(t, 9090, cfg.GRPCPort)
	assert.False(t, cfg.TLSEnabled)
	assert.Empty(t, cfg.TLSCertFile)
	assert.Empty(t, cfg.TLSKeyFile)
	assert.Empty(t, cfg.TLSCAFile)
	assert.Equal(t, "none", cfg.TLSClientAuth)
	assert.Equal(t, "1.2", cfg.TLSMinVersion)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, int64(100*1024*1024), cfg.MaxUploadSize)
	assert.Equal(t, 5*time.Minute, cfg.Timeout)
	assert.True(t, cfg.MetricsEnabled)
	assert.False(t, cfg.AuthEnabled)
	assert.Empty(t, cfg.AuthIssuer)
	assert.Empty(t, cfg.AuthAudience)
	assert.Empty(t, cfg.AuthJWKSURL)
	assert.Equal(t, "jwt", cfg.AuthTokenType)
	assert.Empty(t, cfg.AuthIntrospectionURL)
	assert.Empty(t, cfg.AuthIntrospectionClientID)
	assert.Empty(t, cfg.AuthIntrospectionClientSecret)
}

func TestLoadConfig_EnvironmentVariables(t *testing.T) {
	clearEnv(t)

	os.Setenv("SERVER_PORT", "9000")
	os.Setenv("METRICS_PORT", "9100")
	os.Setenv("SERVER_HOST", "127.0.0.1")
	os.Setenv("GRPC_ENABLED", "true")
	os.Setenv("GRPC_PORT", "50051")
	os.Setenv("TLS_ENABLED", "true")
	os.Setenv("TLS_CERT_FILE", "/tmp/server.crt")
	os.Setenv("TLS_KEY_FILE", "/tmp/server.key")
	os.Setenv("TLS_CA_FILE", "/tmp/ca.crt")
	os.Setenv("TLS_CLIENT_AUTH", "requireandverify")
	os.Setenv("TLS_MIN_VERSION", "1.3")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("MAX_UPLOAD_SIZE", "52428800")
	os.Setenv("TIMEOUT", "10m")
	os.Setenv("METRICS_ENABLED", "false")
	os.Setenv("AUTH_ENABLED", "true")
	os.Setenv("AUTH_ISSUER", "https://issuer.example.com")
	os.Setenv("AUTH_AUDIENCE", "nist-entropy")
	os.Setenv("AUTH_JWKS_URL", "https://issuer.example.com/jwks.json")
	os.Setenv("AUTH_TOKEN_TYPE", "jwt")

	cfg, err := LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, 9100, cfg.ServerPort)
	assert.Equal(t, "127.0.0.1", cfg.ServerHost)
	assert.True(t, cfg.GRPCEnabled)
	assert.Equal(t, 50051, cfg.GRPCPort)
	assert.True(t, cfg.TLSEnabled)
	assert.Equal(t, "/tmp/server.crt", cfg.TLSCertFile)
	assert.Equal(t, "/tmp/server.key", cfg.TLSKeyFile)
	assert.Equal(t, "/tmp/ca.crt", cfg.TLSCAFile)
	assert.Equal(t, "requireandverify", cfg.TLSClientAuth)
	assert.Equal(t, "1.3", cfg.TLSMinVersion)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, int64(52428800), cfg.MaxUploadSize)
	assert.Equal(t, 10*time.Minute, cfg.Timeout)
	assert.False(t, cfg.MetricsEnabled)
	assert.True(t, cfg.AuthEnabled)
	assert.Equal(t, "https://issuer.example.com", cfg.AuthIssuer)
	assert.Equal(t, "nist-entropy", cfg.AuthAudience)
	assert.Equal(t, "https://issuer.example.com/jwks.json", cfg.AuthJWKSURL)
	assert.Equal(t, "jwt", cfg.AuthTokenType)
}

func TestLoadConfig_OpaqueAuthEnvironmentVariables(t *testing.T) {
	clearEnv(t)

	os.Setenv("GRPC_ENABLED", "true")
	os.Setenv("AUTH_ENABLED", "true")
	os.Setenv("AUTH_ISSUER", "https://issuer.example.com")
	os.Setenv("AUTH_AUDIENCE", "nist-entropy")
	os.Setenv("AUTH_TOKEN_TYPE", "opaque")
	os.Setenv("AUTH_INTROSPECTION_URL", "https://issuer.example.com/oauth2/introspect")
	os.Setenv("AUTH_INTROSPECTION_CLIENT_ID", "svc-client")
	os.Setenv("AUTH_INTROSPECTION_CLIENT_SECRET", "svc-secret")

	cfg, err := LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, "opaque", cfg.AuthTokenType)
	assert.Equal(t, "https://issuer.example.com/oauth2/introspect", cfg.AuthIntrospectionURL)
	assert.Equal(t, "svc-client", cfg.AuthIntrospectionClientID)
	assert.Equal(t, "svc-secret", cfg.AuthIntrospectionClientSecret)
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			cfg: &Config{
				ServerPort:    8080,
				GRPCPort:      9090,
				MaxUploadSize: 1024,
				LogLevel:      "info",
			},
			wantErr: false,
		},
		{
			name: "invalid server port - too low",
			cfg: &Config{
				ServerPort:    0,
				GRPCPort:      9090,
				MaxUploadSize: 1024,
				LogLevel:      "info",
			},
			wantErr: true,
			errMsg:  "invalid server port",
		},
		{
			name: "invalid server port - too high",
			cfg: &Config{
				ServerPort:    70000,
				GRPCPort:      9090,
				MaxUploadSize: 1024,
				LogLevel:      "info",
			},
			wantErr: true,
			errMsg:  "invalid server port",
		},
		{
			name: "invalid gRPC port when enabled",
			cfg: &Config{
				ServerPort:    8080,
				GRPCEnabled:   true,
				GRPCPort:      0,
				MaxUploadSize: 1024,
				LogLevel:      "info",
			},
			wantErr: true,
			errMsg:  "invalid gRPC port",
		},
		{
			name: "invalid max upload size",
			cfg: &Config{
				ServerPort:    8080,
				GRPCPort:      9090,
				MaxUploadSize: 512,
				LogLevel:      "info",
			},
			wantErr: true,
			errMsg:  "max upload size too small",
		},
		{
			name: "invalid log level",
			cfg: &Config{
				ServerPort:    8080,
				GRPCPort:      9090,
				MaxUploadSize: 1024,
				LogLevel:      "invalid",
			},
			wantErr: true,
			errMsg:  "invalid log level",
		},
		{
			name: "auth enabled but grpc disabled",
			cfg: &Config{
				ServerPort:    8080,
				GRPCEnabled:   false,
				GRPCPort:      9090,
				MaxUploadSize: 1024,
				LogLevel:      "info",
				AuthEnabled:   true,
				AuthIssuer:    "issuer",
				AuthAudience:  "aud",
			},
			wantErr: true,
			errMsg:  "authentication requires gRPC to be enabled",
		},
		{
			name: "auth enabled missing issuer",
			cfg: &Config{
				ServerPort:    8080,
				GRPCEnabled:   true,
				GRPCPort:      9090,
				MaxUploadSize: 1024,
				LogLevel:      "info",
				AuthEnabled:   true,
				AuthAudience:  "aud",
			},
			wantErr: true,
			errMsg:  "auth issuer",
		},
		{
			name: "auth enabled missing audience",
			cfg: &Config{
				ServerPort:    8080,
				GRPCEnabled:   true,
				GRPCPort:      9090,
				MaxUploadSize: 1024,
				LogLevel:      "info",
				AuthEnabled:   true,
				AuthIssuer:    "issuer",
			},
			wantErr: true,
			errMsg:  "auth audience",
		},
		{
			name: "auth enabled invalid token type",
			cfg: &Config{
				ServerPort:    8080,
				GRPCEnabled:   true,
				GRPCPort:      9090,
				MaxUploadSize: 1024,
				LogLevel:      "info",
				AuthEnabled:   true,
				AuthIssuer:    "issuer",
				AuthAudience:  "aud",
				AuthTokenType: "paseto",
			},
			wantErr: true,
			errMsg:  "AUTH_TOKEN_TYPE",
		},
		{
			name: "auth opaque missing introspection URL",
			cfg: &Config{
				ServerPort:                    8080,
				GRPCEnabled:                   true,
				GRPCPort:                      9090,
				MaxUploadSize:                 1024,
				LogLevel:                      "info",
				AuthEnabled:                   true,
				AuthIssuer:                    "issuer",
				AuthAudience:                  "aud",
				AuthTokenType:                 "opaque",
				AuthIntrospectionClientID:     "client",
				AuthIntrospectionClientSecret: "secret",
			},
			wantErr: true,
			errMsg:  "auth introspection URL",
		},
		{
			name: "auth opaque missing introspection client ID",
			cfg: &Config{
				ServerPort:                    8080,
				GRPCEnabled:                   true,
				GRPCPort:                      9090,
				MaxUploadSize:                 1024,
				LogLevel:                      "info",
				AuthEnabled:                   true,
				AuthIssuer:                    "issuer",
				AuthAudience:                  "aud",
				AuthTokenType:                 "opaque",
				AuthIntrospectionURL:          "https://issuer.example.com/oauth2/introspect",
				AuthIntrospectionClientSecret: "secret",
			},
			wantErr: true,
			errMsg:  "auth introspection client ID",
		},
		{
			name: "auth opaque missing introspection client secret",
			cfg: &Config{
				ServerPort:                8080,
				GRPCEnabled:               true,
				GRPCPort:                  9090,
				MaxUploadSize:             1024,
				LogLevel:                  "info",
				AuthEnabled:               true,
				AuthIssuer:                "issuer",
				AuthAudience:              "aud",
				AuthTokenType:             "opaque",
				AuthIntrospectionURL:      "https://issuer.example.com/oauth2/introspect",
				AuthIntrospectionClientID: "client",
			},
			wantErr: true,
			errMsg:  "auth introspection client secret",
		},
		{
			name: "auth opaque valid introspection config",
			cfg: &Config{
				ServerPort:                    8080,
				GRPCEnabled:                   true,
				GRPCPort:                      9090,
				MaxUploadSize:                 1024,
				LogLevel:                      "info",
				AuthEnabled:                   true,
				AuthIssuer:                    "issuer",
				AuthAudience:                  "aud",
				AuthTokenType:                 "opaque",
				AuthIntrospectionURL:          "https://issuer.example.com/oauth2/introspect",
				AuthIntrospectionClientID:     "client",
				AuthIntrospectionClientSecret: "secret",
			},
			wantErr: false,
		},
		{
			name: "tls enabled without grpc",
			cfg: &Config{
				ServerPort:    8080,
				GRPCEnabled:   false,
				GRPCPort:      9090,
				LogLevel:      "info",
				MaxUploadSize: 1024,
				TLSEnabled:    true,
				TLSCertFile:   "/tmp/cert.pem",
				TLSKeyFile:    "/tmp/key.pem",
			},
			wantErr: true,
			errMsg:  "TLS requires gRPC",
		},
		{
			name: "tls enabled missing cert",
			cfg: &Config{
				ServerPort:    8080,
				GRPCEnabled:   true,
				GRPCPort:      9090,
				LogLevel:      "info",
				MaxUploadSize: 1024,
				TLSEnabled:    true,
				TLSKeyFile:    "/tmp/key.pem",
			},
			wantErr: true,
			errMsg:  "TLS_CERT_FILE",
		},
		{
			name: "tls enabled missing key",
			cfg: &Config{
				ServerPort:    8080,
				GRPCEnabled:   true,
				GRPCPort:      9090,
				LogLevel:      "info",
				MaxUploadSize: 1024,
				TLSEnabled:    true,
				TLSCertFile:   "/tmp/cert.pem",
			},
			wantErr: true,
			errMsg:  "TLS_KEY_FILE",
		},
		{
			name: "tls enabled invalid client auth",
			cfg: &Config{
				ServerPort:    8080,
				GRPCEnabled:   true,
				GRPCPort:      9090,
				LogLevel:      "info",
				MaxUploadSize: 1024,
				TLSEnabled:    true,
				TLSCertFile:   "/tmp/cert.pem",
				TLSKeyFile:    "/tmp/key.pem",
				TLSClientAuth: "broken",
			},
			wantErr: true,
			errMsg:  "TLS_CLIENT_AUTH",
		},
		{
			name: "tls enabled invalid min version",
			cfg: &Config{
				ServerPort:    8080,
				GRPCEnabled:   true,
				GRPCPort:      9090,
				LogLevel:      "info",
				MaxUploadSize: 1024,
				TLSEnabled:    true,
				TLSCertFile:   "/tmp/cert.pem",
				TLSKeyFile:    "/tmp/key.pem",
				TLSMinVersion: "1.1",
			},
			wantErr: true,
			errMsg:  "TLS_MIN_VERSION",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoadConfig_InvalidEnvironmentVariables(t *testing.T) {
	clearEnv(t)

	// Test invalid integer parsing
	os.Setenv("SERVER_PORT", "invalid")
	cfg, err := LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, 9091, cfg.ServerPort) // Should fall back to default

	clearEnv(t)

	// Test invalid metrics port falls back to server port
	os.Setenv("METRICS_PORT", "not-a-number")
	os.Setenv("SERVER_PORT", "9101")
	cfg, err = LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, 9101, cfg.ServerPort)

	clearEnv(t)

	// Test invalid int64 parsing
	os.Setenv("MAX_UPLOAD_SIZE", "not-a-number")
	cfg, err = LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, int64(100*1024*1024), cfg.MaxUploadSize) // Should fall back to default

	clearEnv(t)

	// Test invalid bool parsing
	os.Setenv("GRPC_ENABLED", "maybe")
	cfg, err = LoadConfig()
	require.NoError(t, err)
	assert.False(t, cfg.GRPCEnabled) // Should fall back to default

	clearEnv(t)

	// Test invalid duration parsing
	os.Setenv("TIMEOUT", "invalid-duration")
	cfg, err = LoadConfig()
	require.NoError(t, err)
	assert.Equal(t, 5*time.Minute, cfg.Timeout) // Should fall back to default
}

func TestLoadConfig_ValidationFailure(t *testing.T) {
	clearEnv(t)

	// Set invalid port that will fail validation
	os.Setenv("SERVER_PORT", "99999")
	_, err := LoadConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid server port")
}

func clearEnv(t *testing.T) {
	t.Helper()
	envVars := []string{
		"SERVER_PORT", "SERVER_HOST", "GRPC_ENABLED", "GRPC_PORT", "METRICS_PORT",
		"TLS_ENABLED", "TLS_CERT_FILE", "TLS_KEY_FILE", "TLS_CA_FILE", "TLS_CLIENT_AUTH", "TLS_MIN_VERSION",
		"LOG_LEVEL", "MAX_UPLOAD_SIZE", "TIMEOUT", "METRICS_ENABLED",
		"AUTH_ENABLED", "AUTH_ISSUER", "AUTH_AUDIENCE", "AUTH_JWKS_URL",
		"AUTH_TOKEN_TYPE", "AUTH_INTROSPECTION_URL",
		"AUTH_INTROSPECTION_CLIENT_ID", "AUTH_INTROSPECTION_CLIENT_SECRET",
	}
	for _, v := range envVars {
		os.Unsetenv(v)
	}
}
