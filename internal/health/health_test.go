package health

import (
	"context"
	"errors"
	"testing"
)

type pingerFunc func(context.Context) error

func (f pingerFunc) Ping(ctx context.Context) error { return f(ctx) }

type readyFunc func(context.Context) error

func (f readyFunc) Ready(ctx context.Context) error { return f(ctx) }

func TestCheckerReportsHealthyDependencies(t *testing.T) {
	checker := Checker{
		Database: pingerFunc(func(context.Context) error { return nil }),
		Storage:  readyFunc(func(context.Context) error { return nil }),
	}

	report := checker.Check(context.Background())
	if !report.OK() {
		t.Fatalf("report.OK() = false, want true: %#v", report.Components)
	}
	if len(report.Components) != 2 {
		t.Fatalf("len(Components) = %d, want 2", len(report.Components))
	}
	if report.CheckedAt.IsZero() {
		t.Error("CheckedAt is zero")
	}
}

func TestCheckerReportsUnhealthyDependencies(t *testing.T) {
	checker := Checker{
		Database: pingerFunc(func(context.Context) error { return errors.New("database unavailable") }),
		Storage:  readyFunc(func(context.Context) error { return nil }),
	}

	report := checker.Check(context.Background())
	if report.OK() {
		t.Fatal("report.OK() = true, want false")
	}
	if report.Components[0].Name != "database" || report.Components[0].Healthy {
		t.Errorf("unexpected database component: %#v", report.Components[0])
	}
	if !report.Components[1].Healthy {
		t.Errorf("storage component is unhealthy: %#v", report.Components[1])
	}
}
