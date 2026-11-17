// Package logging builds the application logger.
package logging

import (
	"log/slog"
	"os"

	"github.com/jho-on/moada/internal/config"
)

// New returns a structured logger suited to the configured environment.
// Development uses readable text output; other environments use JSON so logs
// can be collected by external tooling.
func New(environment string, level slog.Level) *slog.Logger {
	options := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if environment == config.EnvironmentDevelopment {
		handler = slog.NewTextHandler(os.Stdout, options)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, options)
	}

	return slog.New(handler)
}
