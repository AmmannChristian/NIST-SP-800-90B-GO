// Package config provides environment-variable-based configuration for the
// NIST SP 800-90B entropy assessment server. All settings have sensible defaults
// and are validated at load time.
package config

import (
	"crypto/tls"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all runtime parameters for the server, including network
// addresses, TLS settings, authentication, logging, and resource limits.
type Config struct {
	// Server configuration (HTTP metrics/health)
	ServerPort  int
	ServerHost  string
	GRPCEnabled bool
	GRPCPort    int

	// TLS for gRPC
	TLSEnabled    bool
	TLSCertFile   string
	TLSKeyFile    string
	TLSCAFile     string
	TLSClientAuth string
	TLSMinVersion string

	// Logging
	LogLevel string

	// File upload limits
	MaxUploadSize int64 // in bytes

	// Request timeouts
	Timeout time.Duration

	// Metrics
	MetricsEnabled bool

	// Authentication
	AuthEnabled                             bool
	AuthIssuer                              string
	AuthAudience                            string
	AuthJWKSURL                             string
	AuthTokenType                           string
	AuthIntrospectionURL                    string
	AuthIntrospectionAuthMethod             string
	AuthIntrospectionClientID               string
	AuthIntrospectionClientSecret           string
	AuthIntrospectionPrivateKey             string
	AuthIntrospectionPrivateKeyFile         string
	AuthIntrospectionPrivateKeyJWTKeyID     string
	AuthIntrospectionPrivateKeyJWTAlgorithm string
	AuthzRequiredRoles                      []string
	AuthzRequiredScopes                     []string
	AuthzRoleMatchMode                      string
	AuthzScopeMatchMode                     string
	AuthzRoleClaimPaths                     []string
	AuthzScopeClaimPaths                    []string
}

// LoadConfig reads configuration from environment variables, applies default
// values for any unset variables, and validates the resulting configuration.
// It returns an error if validation fails.
func LoadConfig() (*Config, error) {
	config := &Config{
		// Defaults
		ServerPort:                              getEnvAsInt("METRICS_PORT", getEnvAsInt("SERVER_PORT", 9091)),
		ServerHost:                              getEnv("SERVER_HOST", "0.0.0.0"),
		GRPCEnabled:                             getEnvAsBool("GRPC_ENABLED", false),
		GRPCPort:                                getEnvAsInt("GRPC_PORT", 9090),
		TLSEnabled:                              getEnvAsBool("TLS_ENABLED", false),
		TLSCertFile:                             getEnv("TLS_CERT_FILE", ""),
		TLSKeyFile:                              getEnv("TLS_KEY_FILE", ""),
		TLSCAFile:                               getEnv("TLS_CA_FILE", ""),
		TLSClientAuth:                           getEnv("TLS_CLIENT_AUTH", "none"),
		TLSMinVersion:                           getEnv("TLS_MIN_VERSION", "1.2"),
		LogLevel:                                getEnv("LOG_LEVEL", "info"),
		MaxUploadSize:                           getEnvAsInt64("MAX_UPLOAD_SIZE", 100*1024*1024), // 100MB default
		Timeout:                                 getEnvAsDuration("TIMEOUT", 5*time.Minute),
		MetricsEnabled:                          getEnvAsBool("METRICS_ENABLED", true),
		AuthEnabled:                             getEnvAsBool("AUTH_ENABLED", false),
		AuthIssuer:                              getEnv("AUTH_ISSUER", ""),
		AuthAudience:                            getEnv("AUTH_AUDIENCE", ""),
		AuthJWKSURL:                             getEnv("AUTH_JWKS_URL", ""),
		AuthTokenType:                           getEnv("AUTH_TOKEN_TYPE", "jwt"),
		AuthIntrospectionURL:                    getEnv("AUTH_INTROSPECTION_URL", ""),
		AuthIntrospectionAuthMethod:             getEnv("AUTH_INTROSPECTION_AUTH_METHOD", "client_secret_basic"),
		AuthIntrospectionClientID:               getEnv("AUTH_INTROSPECTION_CLIENT_ID", ""),
		AuthIntrospectionClientSecret:           getEnv("AUTH_INTROSPECTION_CLIENT_SECRET", ""),
		AuthIntrospectionPrivateKey:             getEnv("AUTH_INTROSPECTION_PRIVATE_KEY", ""),
		AuthIntrospectionPrivateKeyFile:         getEnv("AUTH_INTROSPECTION_PRIVATE_KEY_FILE", ""),
		AuthIntrospectionPrivateKeyJWTKeyID:     getEnv("AUTH_INTROSPECTION_PRIVATE_KEY_JWT_KID", ""),
		AuthIntrospectionPrivateKeyJWTAlgorithm: getEnv("AUTH_INTROSPECTION_PRIVATE_KEY_JWT_ALG", ""),
		AuthzRequiredRoles:                      parseCSV(getEnv("AUTHZ_REQUIRED_ROLES", "")),
		AuthzRequiredScopes:                     parseCSV(getEnv("AUTHZ_REQUIRED_SCOPES", "")),
		AuthzRoleMatchMode:                      getEnv("AUTHZ_ROLE_MATCH_MODE", "any"),
		AuthzScopeMatchMode:                     getEnv("AUTHZ_SCOPE_MATCH_MODE", "any"),
		AuthzRoleClaimPaths:                     parseCSV(getEnv("AUTHZ_ROLE_CLAIM_PATHS", "")),
		AuthzScopeClaimPaths:                    parseCSV(getEnv("AUTHZ_SCOPE_CLAIM_PATHS", "")),
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate checks all configuration invariants, including port ranges, upload
// size limits, log level validity, and cross-field constraints such as TLS and
// authentication requiring gRPC to be enabled.
func (c *Config) Validate() error {
	if c.ServerPort < 1 || c.ServerPort > 65535 {
		return fmt.Errorf("invalid server port: %d (must be 1-65535)", c.ServerPort)
	}

	if c.GRPCEnabled && (c.GRPCPort < 1 || c.GRPCPort > 65535) {
		return fmt.Errorf("invalid gRPC port: %d (must be 1-65535)", c.GRPCPort)
	}

	if c.MaxUploadSize < 1024 {
		return fmt.Errorf("max upload size too small: %d (must be at least 1024 bytes)", c.MaxUploadSize)
	}

	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", c.LogLevel)
	}

	roleMatchMode, err := parseAuthzMatchMode(c.AuthzRoleMatchMode, "AUTHZ_ROLE_MATCH_MODE")
	if err != nil {
		return err
	}
	c.AuthzRoleMatchMode = roleMatchMode

	scopeMatchMode, err := parseAuthzMatchMode(c.AuthzScopeMatchMode, "AUTHZ_SCOPE_MATCH_MODE")
	if err != nil {
		return err
	}
	c.AuthzScopeMatchMode = scopeMatchMode

	c.AuthzRequiredRoles = normalizeCSVValues(c.AuthzRequiredRoles)
	c.AuthzRequiredScopes = normalizeCSVValues(c.AuthzRequiredScopes)
	c.AuthzRoleClaimPaths = normalizeCSVValues(c.AuthzRoleClaimPaths)
	c.AuthzScopeClaimPaths = normalizeCSVValues(c.AuthzScopeClaimPaths)

	if c.AuthEnabled {
		if !c.GRPCEnabled {
			return fmt.Errorf("authentication requires gRPC to be enabled")
		}
		if c.AuthIssuer == "" {
			return fmt.Errorf("invalid auth issuer: required when AUTH_ENABLED=true")
		}
		if c.AuthAudience == "" {
			return fmt.Errorf("invalid auth audience: required when AUTH_ENABLED=true")
		}
		tokenType, err := parseAuthTokenType(c.AuthTokenType)
		if err != nil {
			return err
		}
		c.AuthTokenType = tokenType

		if c.AuthTokenType == "opaque" {
			if c.AuthIntrospectionURL == "" {
				return fmt.Errorf("invalid auth introspection URL: required when AUTH_TOKEN_TYPE=opaque")
			}
			authMethod, err := parseAuthIntrospectionAuthMethod(c.AuthIntrospectionAuthMethod)
			if err != nil {
				return err
			}
			c.AuthIntrospectionAuthMethod = authMethod

			switch c.AuthIntrospectionAuthMethod {
			case "client_secret_basic":
				if c.AuthIntrospectionClientID == "" {
					return fmt.Errorf("invalid auth introspection client ID: required when AUTH_INTROSPECTION_AUTH_METHOD=client_secret_basic")
				}
				if c.AuthIntrospectionClientSecret == "" {
					return fmt.Errorf("invalid auth introspection client secret: required when AUTH_INTROSPECTION_AUTH_METHOD=client_secret_basic")
				}
			case "private_key_jwt":
				c.AuthIntrospectionPrivateKey = strings.TrimSpace(c.AuthIntrospectionPrivateKey)
				c.AuthIntrospectionPrivateKeyFile = strings.TrimSpace(c.AuthIntrospectionPrivateKeyFile)
				if c.AuthIntrospectionPrivateKey != "" && c.AuthIntrospectionPrivateKeyFile != "" {
					return fmt.Errorf("invalid auth introspection private key config: AUTH_INTROSPECTION_PRIVATE_KEY and AUTH_INTROSPECTION_PRIVATE_KEY_FILE are mutually exclusive")
				}
				if c.AuthIntrospectionPrivateKey == "" && c.AuthIntrospectionPrivateKeyFile == "" {
					return fmt.Errorf("invalid auth introspection private key: required when AUTH_INTROSPECTION_AUTH_METHOD=private_key_jwt")
				}
				if c.AuthIntrospectionPrivateKeyFile != "" {
					privateKeyBytes, readErr := os.ReadFile(c.AuthIntrospectionPrivateKeyFile)
					if readErr != nil {
						return fmt.Errorf("invalid auth introspection private key file: %w", readErr)
					}
					c.AuthIntrospectionPrivateKey = strings.TrimSpace(string(privateKeyBytes))
					if c.AuthIntrospectionPrivateKey == "" {
						return fmt.Errorf("invalid auth introspection private key file: empty file")
					}
				}
				privateKeyJWTAlgorithm, parseErr := parseAuthIntrospectionPrivateKeyJWTAlgorithm(c.AuthIntrospectionPrivateKeyJWTAlgorithm)
				if parseErr != nil {
					return parseErr
				}
				c.AuthIntrospectionPrivateKeyJWTAlgorithm = privateKeyJWTAlgorithm
			}
		}
	}

	if c.TLSEnabled {
		if !c.GRPCEnabled {
			return fmt.Errorf("TLS requires gRPC to be enabled")
		}
		if c.TLSCertFile == "" {
			return fmt.Errorf("invalid TLS_CERT_FILE: required when TLS_ENABLED=true")
		}
		if c.TLSKeyFile == "" {
			return fmt.Errorf("invalid TLS_KEY_FILE: required when TLS_ENABLED=true")
		}
		if _, err := parseTLSClientAuth(c.TLSClientAuth); err != nil {
			return err
		}
		if _, err := parseTLSMinVersion(c.TLSMinVersion); err != nil {
			return err
		}
	}

	return nil
}

// TLSClientAuthType returns the parsed tls.ClientAuthType from configuration.
func (c *Config) TLSClientAuthType() (tls.ClientAuthType, error) {
	return parseTLSClientAuth(c.TLSClientAuth)
}

// TLSMinVersionValue returns the configured minimum TLS version (defaults to TLS 1.2).
func (c *Config) TLSMinVersionValue() (uint16, error) {
	return parseTLSMinVersion(c.TLSMinVersion)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

// parseTLSClientAuth maps a human-readable string to a tls.ClientAuthType.
// Accepted values include "none", "request", "requireany", "verifyifgiven",
// "requireandverify", and "mtls".
func parseTLSClientAuth(mode string) (tls.ClientAuthType, error) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "none", "noclientcert":
		return tls.NoClientCert, nil
	case "request", "requestclientcert":
		return tls.RequestClientCert, nil
	case "requireany", "requireanyclientcert":
		return tls.RequireAnyClientCert, nil
	case "verifyifgiven", "verify_client_cert_if_given":
		return tls.VerifyClientCertIfGiven, nil
	case "requireandverify", "requireandverifyclientcert", "mtls":
		return tls.RequireAndVerifyClientCert, nil
	default:
		return tls.NoClientCert, fmt.Errorf("invalid TLS_CLIENT_AUTH: %s", mode)
	}
}

// parseTLSMinVersion converts a version string (e.g., "1.2", "1.3") into the
// corresponding crypto/tls constant. Only TLS 1.2 and 1.3 are supported.
func parseTLSMinVersion(version string) (uint16, error) {
	switch strings.ToLower(strings.TrimSpace(version)) {
	case "", "default", "1.2", "tls1.2", "tls12":
		return tls.VersionTLS12, nil
	case "1.3", "tls1.3", "tls13":
		return tls.VersionTLS13, nil
	default:
		return 0, fmt.Errorf("invalid TLS_MIN_VERSION: %s (use 1.2 or 1.3)", version)
	}
}

func parseAuthTokenType(tokenType string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(tokenType)) {
	case "", "jwt":
		return "jwt", nil
	case "opaque":
		return "opaque", nil
	default:
		return "", fmt.Errorf("invalid AUTH_TOKEN_TYPE: %s (use jwt or opaque)", tokenType)
	}
}

func parseAuthIntrospectionAuthMethod(method string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(method)) {
	case "", "client_secret_basic":
		return "client_secret_basic", nil
	case "private_key_jwt":
		return "private_key_jwt", nil
	default:
		return "", fmt.Errorf("invalid AUTH_INTROSPECTION_AUTH_METHOD: %s (use client_secret_basic or private_key_jwt)", method)
	}
}

func parseAuthIntrospectionPrivateKeyJWTAlgorithm(algorithm string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(algorithm)) {
	case "", "RS256", "ES256":
		return strings.ToUpper(strings.TrimSpace(algorithm)), nil
	default:
		return "", fmt.Errorf("invalid AUTH_INTROSPECTION_PRIVATE_KEY_JWT_ALG: %s (use RS256 or ES256)", algorithm)
	}
}

func parseAuthzMatchMode(mode string, envName string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "any":
		return "any", nil
	case "all":
		return "all", nil
	default:
		return "", fmt.Errorf("invalid %s: %s (use any or all)", envName, mode)
	}
}

func parseCSV(value string) []string {
	return normalizeCSVValues(strings.Split(value, ","))
}

func normalizeCSVValues(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	normalizedValues := make([]string, 0, len(values))
	for _, value := range values {
		normalizedValue := strings.TrimSpace(value)
		if normalizedValue == "" {
			continue
		}
		normalizedValues = append(normalizedValues, normalizedValue)
	}

	if len(normalizedValues) == 0 {
		return nil
	}

	return normalizedValues
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}
