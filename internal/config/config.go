// Package config loads and validates the application configuration from
// environment variables.
package config

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	EnvironmentDevelopment = "development"
	EnvironmentTest        = "test"
	EnvironmentProduction  = "production"

	defaultHTTPAddr = ":8080"
)

// Config holds every runtime setting required by the application.
type Config struct {
	Environment string
	HTTPAddr    string
	LogLevel    slog.Level
	Database    DatabaseConfig
	S3          S3Config
}

// DatabaseConfig describes how the application connects to PostgreSQL.
type DatabaseConfig struct {
	URL            string
	MaxConns       int32
	ConnectTimeout time.Duration
}

// S3Config describes how the application connects to Garage through the
// S3-compatible API.
type S3Config struct {
	Endpoint     string
	AccessKey    string
	SecretKey    string
	Bucket       string
	Region       string
	UseSSL       bool
	CreateBucket bool
}

// Load reads the configuration from the process environment.
func Load() (Config, error) {
	return load(os.LookupEnv)
}

func load(lookup func(string) (string, bool)) (Config, error) {
	l := &loader{lookup: lookup}

	cfg := Config{
		Environment: l.string("MOADA_ENV", EnvironmentDevelopment),
		HTTPAddr:    l.string("MOADA_HTTP_ADDR", defaultHTTPAddr),
		LogLevel:    l.logLevel("MOADA_LOG_LEVEL", "info"),
		Database: DatabaseConfig{
			URL:            l.required("MOADA_DATABASE_URL"),
			MaxConns:       int32(l.integer("MOADA_DB_MAX_CONNS", 10, 1, 1000)),
			ConnectTimeout: l.duration("MOADA_DB_CONNECT_TIMEOUT", 10*time.Second),
		},
		S3: S3Config{
			Endpoint:     l.required("MOADA_S3_ENDPOINT"),
			AccessKey:    l.required("MOADA_S3_ACCESS_KEY"),
			SecretKey:    l.required("MOADA_S3_SECRET_KEY"),
			Bucket:       l.required("MOADA_S3_BUCKET"),
			Region:       l.string("MOADA_S3_REGION", "garage"),
			UseSSL:       l.boolean("MOADA_S3_USE_SSL", false),
			CreateBucket: l.boolean("MOADA_S3_CREATE_BUCKET", true),
		},
	}

	switch cfg.Environment {
	case EnvironmentDevelopment, EnvironmentTest, EnvironmentProduction:
	default:
		l.fail("MOADA_ENV must be one of: development, test, production")
	}

	if cfg.HTTPAddr == "" {
		l.fail("MOADA_HTTP_ADDR must not be empty")
	}

	if err := validateDatabaseURL(cfg.Database.URL); err != nil {
		l.fail(err.Error())
	}

	if strings.Contains(cfg.S3.Endpoint, "://") {
		l.fail("MOADA_S3_ENDPOINT must be a host and optional port, without a scheme")
	}

	if len(l.errs) > 0 {
		return Config{}, fmt.Errorf("invalid configuration:\n  - %s", strings.Join(l.errs, "\n  - "))
	}

	return cfg, nil
}

func validateDatabaseURL(raw string) error {
	if raw == "" {
		return nil // the required check already reports this field
	}

	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("MOADA_DATABASE_URL is not a valid URL")
	}

	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return fmt.Errorf("MOADA_DATABASE_URL must use the postgres or postgresql scheme")
	}

	if u.Host == "" {
		return fmt.Errorf("MOADA_DATABASE_URL must include a host")
	}

	return nil
}

type loader struct {
	lookup func(string) (string, bool)
	errs   []string
}

func (l *loader) fail(message string) {
	l.errs = append(l.errs, message)
}

func (l *loader) string(key, fallback string) string {
	if value, ok := l.lookup(key); ok {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return fallback
}

func (l *loader) required(key string) string {
	value := l.string(key, "")
	if value == "" {
		l.fail(key + " is required")
	}
	return value
}

func (l *loader) boolean(key string, fallback bool) bool {
	raw, ok := l.lookup(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return fallback
	}

	value, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err != nil {
		l.fail(key + " must be a boolean (true or false)")
		return fallback
	}
	return value
}

func (l *loader) integer(key string, fallback, minimum, maximum int) int {
	raw, ok := l.lookup(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return fallback
	}

	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < minimum || value > maximum {
		l.fail(fmt.Sprintf("%s must be an integer between %d and %d", key, minimum, maximum))
		return fallback
	}
	return value
}

func (l *loader) duration(key string, fallback time.Duration) time.Duration {
	raw, ok := l.lookup(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return fallback
	}

	value, err := time.ParseDuration(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		l.fail(key + " must be a positive duration (for example 10s)")
		return fallback
	}
	return value
}

func (l *loader) logLevel(key, fallback string) slog.Level {
	raw := strings.ToLower(l.string(key, fallback))

	var level slog.Level
	if err := level.UnmarshalText([]byte(raw)); err != nil {
		l.fail(key + " must be one of: debug, info, warn, error")
		return slog.LevelInfo
	}
	return level
}
