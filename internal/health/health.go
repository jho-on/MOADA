// Package health checks the availability of external dependencies.
package health

import (
	"context"
	"time"
)

// Pinger verifies a database connection. It is satisfied by *pgxpool.Pool.
type Pinger interface {
	Ping(ctx context.Context) error
}

// ReadyChecker verifies an object storage backend. It is satisfied by
// storage.Store.
type ReadyChecker interface {
	Ready(ctx context.Context) error
}

// Checker groups the dependency checks exposed by the readiness endpoint.
type Checker struct {
	Database Pinger
	Storage  ReadyChecker
}

// Component is the result of checking one dependency.
type Component struct {
	Name    string
	Healthy bool
	Err     error
}

// Report is the result of a health check run.
type Report struct {
	Components []Component
	CheckedAt  time.Time
}

// OK reports whether every component is healthy.
func (r Report) OK() bool {
	for _, component := range r.Components {
		if !component.Healthy {
			return false
		}
	}
	return true
}

// Check runs every component check using the provided context.
func (c Checker) Check(ctx context.Context) Report {
	report := Report{
		Components: make([]Component, 0, 2),
		CheckedAt:  time.Now().UTC(),
	}

	if c.Database != nil {
		err := c.Database.Ping(ctx)
		report.Components = append(report.Components, Component{
			Name:    "database",
			Healthy: err == nil,
			Err:     err,
		})
	}

	if c.Storage != nil {
		err := c.Storage.Ready(ctx)
		report.Components = append(report.Components, Component{
			Name:    "storage",
			Healthy: err == nil,
			Err:     err,
		})
	}

	return report
}
