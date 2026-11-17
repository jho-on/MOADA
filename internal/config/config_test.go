package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadUsesDefaults(t *testing.T) {
	env := map[string]string{
		"MOADA_DATABASE_URL":  "postgres://moada:moada@localhost:5432/moada?sslmode=disable",
		"MOADA_S3_ENDPOINT":   "localhost:3900",
		"MOADA_S3_ACCESS_KEY": "access",
		"MOADA_S3_SECRET_KEY": "secret",
		"MOADA_S3_BUCKET":     "moada",
	}

	cfg, err := load(mapLookup(env))
	if err != nil {
		t.Fatalf("load returned an error: %v", err)
	}

	if cfg.Environment != EnvironmentDevelopment {
		t.Errorf("Environment = %q, want %q", cfg.Environment, EnvironmentDevelopment)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q, want :8080", cfg.HTTPAddr)
	}
	if cfg.Database.MaxConns != 10 {
		t.Errorf("Database.MaxConns = %d, want 10", cfg.Database.MaxConns)
	}
	if cfg.Database.ConnectTimeout != 10*time.Second {
		t.Errorf("Database.ConnectTimeout = %s, want 10s", cfg.Database.ConnectTimeout)
	}
	if cfg.S3.Region != "garage" {
		t.Errorf("S3.Region = %q, want garage", cfg.S3.Region)
	}
	if !cfg.S3.CreateBucket {
		t.Error("S3.CreateBucket = false, want true by default")
	}
	if cfg.S3.UseSSL {
		t.Error("S3.UseSSL = true, want false by default")
	}
}

func TestLoadRequiresConnectionSettings(t *testing.T) {
	_, err := load(mapLookup(nil))
	if err == nil {
		t.Fatal("load returned no error, want a validation error")
	}

	for _, key := range []string{
		"MOADA_DATABASE_URL",
		"MOADA_S3_ENDPOINT",
		"MOADA_S3_ACCESS_KEY",
		"MOADA_S3_SECRET_KEY",
		"MOADA_S3_BUCKET",
	} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("error %q does not mention %s", err, key)
		}
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	env := map[string]string{
		"MOADA_ENV":           "staging",
		"MOADA_HTTP_ADDR":     "",
		"MOADA_LOG_LEVEL":     "verbose",
		"MOADA_DATABASE_URL":  "mysql://localhost/moada",
		"MOADA_DB_MAX_CONNS":  "0",
		"MOADA_S3_ENDPOINT":   "http://localhost:3900",
		"MOADA_S3_ACCESS_KEY": "access",
		"MOADA_S3_SECRET_KEY": "secret",
		"MOADA_S3_BUCKET":     "moada",
		"MOADA_S3_USE_SSL":    "maybe",
	}

	_, err := load(mapLookup(env))
	if err == nil {
		t.Fatal("load returned no error, want a validation error")
	}

	message := err.Error()
	for _, fragment := range []string{
		"MOADA_ENV",
		"MOADA_LOG_LEVEL",
		"MOADA_DATABASE_URL must use",
		"MOADA_DB_MAX_CONNS",
		"MOADA_S3_ENDPOINT must be",
		"MOADA_S3_USE_SSL",
	} {
		if !strings.Contains(message, fragment) {
			t.Errorf("error %q does not contain %q", message, fragment)
		}
	}
}

func TestLoadAcceptsProductionSettings(t *testing.T) {
	env := map[string]string{
		"MOADA_ENV":                "production",
		"MOADA_LOG_LEVEL":          "warn",
		"MOADA_DATABASE_URL":       "postgresql://moada:secret@db:5432/moada?sslmode=require",
		"MOADA_DB_MAX_CONNS":       "25",
		"MOADA_DB_CONNECT_TIMEOUT": "30s",
		"MOADA_S3_ENDPOINT":        "garage.internal:3900",
		"MOADA_S3_ACCESS_KEY":      "access",
		"MOADA_S3_SECRET_KEY":      "secret",
		"MOADA_S3_BUCKET":          "moada",
		"MOADA_S3_REGION":          "us-east-1",
		"MOADA_S3_USE_SSL":         "true",
		"MOADA_S3_CREATE_BUCKET":   "false",
	}

	cfg, err := load(mapLookup(env))
	if err != nil {
		t.Fatalf("load returned an error: %v", err)
	}

	if cfg.Environment != EnvironmentProduction {
		t.Errorf("Environment = %q, want production", cfg.Environment)
	}
	if cfg.Database.MaxConns != 25 {
		t.Errorf("Database.MaxConns = %d, want 25", cfg.Database.MaxConns)
	}
	if cfg.Database.ConnectTimeout != 30*time.Second {
		t.Errorf("Database.ConnectTimeout = %s, want 30s", cfg.Database.ConnectTimeout)
	}
	if !cfg.S3.UseSSL || cfg.S3.CreateBucket {
		t.Errorf("unexpected S3 booleans: UseSSL=%v CreateBucket=%v", cfg.S3.UseSSL, cfg.S3.CreateBucket)
	}
}

func mapLookup(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}
