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

	assert.Equal(t, 8080, cfg.ServerPort)
	assert.Equal(t, "0.0.0.0", cfg.ServerHost)
	assert.False(t, cfg.GRPCEnabled)
	assert.Equal(t, 9090, cfg.GRPCPort)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, int64(100*1024*1024), cfg.MaxUploadSize)
	assert.Equal(t, 5*time.Minute, cfg.Timeout)
	assert.True(t, cfg.MetricsEnabled)
}

func TestLoadConfig_EnvironmentVariables(t *testing.T) {
	clearEnv(t)

	os.Setenv("SERVER_PORT", "9000")
	os.Setenv("SERVER_HOST", "127.0.0.1")
	os.Setenv("GRPC_ENABLED", "true")
	os.Setenv("GRPC_PORT", "50051")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("MAX_UPLOAD_SIZE", "52428800")
	os.Setenv("TIMEOUT", "10m")
	os.Setenv("METRICS_ENABLED", "false")

	cfg, err := LoadConfig()
	require.NoError(t, err)

	assert.Equal(t, 9000, cfg.ServerPort)
	assert.Equal(t, "127.0.0.1", cfg.ServerHost)
	assert.True(t, cfg.GRPCEnabled)
	assert.Equal(t, 50051, cfg.GRPCPort)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, int64(52428800), cfg.MaxUploadSize)
	assert.Equal(t, 10*time.Minute, cfg.Timeout)
	assert.False(t, cfg.MetricsEnabled)
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
	assert.Equal(t, 8080, cfg.ServerPort) // Should fall back to default

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
		"SERVER_PORT", "SERVER_HOST", "GRPC_ENABLED", "GRPC_PORT",
		"LOG_LEVEL", "MAX_UPLOAD_SIZE", "TIMEOUT", "METRICS_ENABLED",
	}
	for _, v := range envVars {
		os.Unsetenv(v)
	}
}
