package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jho-on/moada/internal/config"
	"github.com/jho-on/moada/internal/database"
	"github.com/jho-on/moada/internal/health"
	"github.com/jho-on/moada/internal/httpapi"
	"github.com/jho-on/moada/internal/logging"
	"github.com/jho-on/moada/internal/storage"
)

const (
	shutdownTimeout = 10 * time.Second
	bucketTimeout   = 15 * time.Second
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "moada: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := logging.New(cfg.Environment, cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.Open(ctx, cfg.Database)
	if err != nil {
		return err
	}
	defer pool.Close()

	store, err := storage.NewS3(storage.S3Options{
		Endpoint:  cfg.S3.Endpoint,
		AccessKey: cfg.S3.AccessKey,
		SecretKey: cfg.S3.SecretKey,
		Bucket:    cfg.S3.Bucket,
		Region:    cfg.S3.Region,
		UseSSL:    cfg.S3.UseSSL,
	})
	if err != nil {
		return err
	}

	if cfg.S3.CreateBucket {
		bucketCtx, cancel := context.WithTimeout(ctx, bucketTimeout)
		err = store.EnsureBucket(bucketCtx)
		cancel()
		if err != nil {
			return err
		}
	}

	handler := httpapi.New(logger, health.Checker{
		Database: pool,
		Storage:  store,
	}, cfg.Environment)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	logger.Info("starting MOADA",
		"environment", cfg.Environment,
		"address", cfg.HTTPAddr,
		"database_host", safeHost(cfg.Database.URL),
		"s3_endpoint", cfg.S3.Endpoint,
		"s3_bucket", cfg.S3.Bucket,
	)

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown requested")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown HTTP server: %w", err)
		}
		return nil
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	}
}

// safeHost returns only the host portion of the database URL so credentials
// never reach the logs.
func safeHost(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "invalid"
	}
	return parsed.Host
}
