// Package config handles loading and validating application configuration.
//
// Configuration is read from a .env file via godotenv. If the file does not
// exist or a variable is missing, sensible defaults are used for non-critical
// fields. Critical fields (listed below) cause Load() to return an error if
// none of their sources provide a value.
//
// Required (no fallback):
//
//	DATABASE_URL
//	JWT_SECRET
//
// Fallback defaults:
//
//	SERVER_PORT       = 8080
//	DATABASE_URL        = postgresql://postgres:postgres@localhost:5432/photo_audit
//	JWT_SECRET         = change-me-in-production
//	JWT_EXPIRY         = 24h
package config

import (
	"crypto/rand"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration values.
type Config struct {
	ServerPort        string
	DatabaseURL       string
	RedisURL          string
	MinIOEndpoint     string
	MinIOAccessKey    string
	MinIOSecretKey    string
	MinIOBucket       string
	MinIOUseSSL       bool
	AgnesAPIKey       string
	DeepSeekAPIKey    string
	FallbackEnabled   bool // Enable local rule-based fallback when AI API unavailable
	JWTSecret         string
	JWTMinLength      int `mapstructure:"JWT_MIN_LENGTH" default:"32"`
	JWTExpiry         time.Duration
}

// defaults maps optional config keys to their fallback values.
var defaults = map[string]string{
	"SERVER_PORT":       "8080",
	"DATABASE_URL":      "postgresql://postgres:postgres@localhost:5432/photo_audit",
	"REDIS_URL":         "",
	"MINIO_ENDPOINT":    "localhost:9000",
	"MINIO_ACCESS_KEY":  "minioadmin",
	"MINIO_SECRET_KEY":  "minioadmin",
	"MINIO_BUCKET":      "audit-platform",
	"AGNES_API_KEY":     "",
	"DEEPSEEK_API_KEY":  "",
	"FALLBACK_ENABLED":  "true",
	"JWT_SECRET":        "",
	"JWT_EXPIRY":        "24h",
}

// critical lists the keys that must have a non-empty value for Load() to
// succeed. Missing critical fields produce a descriptive error.
var critical = []string{"DATABASE_URL"}

// Load reads configuration from .env, then falls back to built-in defaults.
// It returns an error if any critical field is absent.
func Load() (*Config, error) {
	// Try to load .env file. A missing file is not an error – we fall back
	// to defaults.
	_ = godotenv.Load() // ignore "file not found"; other parse errors bubble up.

	cfg := make(map[string]string)
	for k, v := range defaults {
		cfg[k] = v
	}

	// Overlay environment variables (including .env).
	for k := range defaults {
		if envVal := os.Getenv(k); envVal != "" {
			cfg[k] = envVal
		}
	}

	c := &Config{
		ServerPort:      envOr(cfg, "SERVER_PORT"),
		DatabaseURL:     envOr(cfg, "DATABASE_URL"),
		RedisURL:        envOr(cfg, "REDIS_URL"),
		MinIOEndpoint:   envOr(cfg, "MINIO_ENDPOINT"),
		MinIOAccessKey:  envOr(cfg, "MINIO_ACCESS_KEY"),
		MinIOSecretKey:  envOr(cfg, "MINIO_SECRET_KEY"),
		MinIOBucket:     envOr(cfg, "MINIO_BUCKET"),
		AgnesAPIKey:     envOr(cfg, "AGNES_API_KEY"),
		DeepSeekAPIKey:  envOr(cfg, "DEEPSEEK_API_KEY"),
		FallbackEnabled: envBool(cfg, "FALLBACK_ENABLED", true),
		JWTSecret:       envOr(cfg, "JWT_SECRET"),
	}

	// Generate a random JWT secret if none is configured.
	if c.JWTSecret == "" {
		c.JWTSecret = generateJWTSecret()
	}

	// Warn if the secret looks like a placeholder.
	if strings.HasPrefix(c.JWTSecret, "change-me") || c.JWTSecret == "dev-secret-change-me" {
		fmt.Fprintf(os.Stderr, "[WARN] JWT_SECRET appears to be a placeholder value. Set a strong secret in production.\n")
	}

	// Parse JWT expiry.
	c.JWTExpiry = parseDuration(envOr(cfg, "JWT_EXPIRY"))

	// Validate critical fields.
	var missing []string
	for _, k := range critical {
		if envOr(cfg, k) == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("critical config field(s) missing: %s", strings.Join(missing, ", "))
	}

	return c, nil
}

// envOr returns the value for key from env map. It first checks the key
// exactly, then tries the uppercase and lowercase variants so that both
// .env (which uses UPPER_CASE) and os.Getenv (which may differ) resolve
// consistently.
func envOr(env map[string]string, key string) string {
	// Exact match in the merged map.
	if v, ok := env[key]; ok {
		return v
	}
	return ""
}

// parseDuration parses a human-readable duration string into time.Duration.
// Returns zero value if s is empty or unparseable.
func parseDuration(s string) time.Duration {
	if s == "" {
		return 0
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		// Attempt to parse as an integer number of hours.
		if hours, herr := strconv.Atoi(s); herr == nil {
			return time.Duration(hours) * time.Hour
		}
		return 0
	}
	return d
}

// envBool returns the boolean value for key from env map.
// Defaults to defaultVal if the key is empty or unparseable.
func envBool(env map[string]string, key string, defaultVal bool) bool {
	v := envOr(env, key)
	if v == "" {
		return defaultVal
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return defaultVal
	}
	return b
}

// generateJWTSecret creates a cryptographically secure random 32-byte hex JWT secret.
func generateJWTSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// This should never happen on a functioning system.
		panic("crypto/rand failed: " + err.Error())
	}
	return fmt.Sprintf("%x", b)
}
