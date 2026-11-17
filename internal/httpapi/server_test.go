package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jho-on/moada/internal/health"
)

type pingerFunc func(context.Context) error

func (f pingerFunc) Ping(ctx context.Context) error { return f(ctx) }

type readyFunc func(context.Context) error

func (f readyFunc) Ready(ctx context.Context) error { return f(ctx) }

func newTestHandler(databaseErr, storageErr error) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	checker := health.Checker{
		Database: pingerFunc(func(context.Context) error { return databaseErr }),
		Storage:  readyFunc(func(context.Context) error { return storageErr }),
	}
	return New(logger, checker, "test")
}

func TestLiveness(t *testing.T) {
	handler := newTestHandler(nil, nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status = %q, want ok", body["status"])
	}
}

func TestReadinessWhenHealthy(t *testing.T) {
	handler := newTestHandler(nil, nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}

	var body struct {
		Status     string          `json:"status"`
		Components map[string]bool `json:"components"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status != "ok" || !body.Components["database"] || !body.Components["storage"] {
		t.Errorf("unexpected response: %+v", body)
	}
}

func TestReadinessWhenDependencyFails(t *testing.T) {
	handler := newTestHandler(nil, errors.New("garage unavailable"))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", recorder.Code)
	}
	if strings.Contains(recorder.Body.String(), "garage unavailable") {
		t.Error("response leaked the internal error")
	}
}

func TestHomePage(t *testing.T) {
	handler := newTestHandler(nil, nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "MOADA") {
		t.Error("home page does not contain the application name")
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", contentType)
	}
}

func TestStatusPartialReportsUnavailableComponent(t *testing.T) {
	handler := newTestHandler(errors.New("database unavailable"), nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/partials/status", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "Indisponível") {
		t.Error("status partial does not report the unavailable component")
	}
	if strings.Contains(recorder.Body.String(), "database unavailable") {
		t.Error("status partial leaked the internal error")
	}
}

func TestUnknownRouteReturnsNotFound(t *testing.T) {
	handler := newTestHandler(nil, nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/does-not-exist", nil))

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
}

func TestSecurityHeadersAndRequestID(t *testing.T) {
	handler := newTestHandler(nil, nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if got := recorder.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
	}
	if got := recorder.Header().Get("Content-Security-Policy"); !strings.Contains(got, "default-src 'self'") {
		t.Errorf("Content-Security-Policy = %q, want a restrictive policy", got)
	}
	if got := recorder.Header().Get("X-Request-ID"); got == "" {
		t.Error("X-Request-ID is empty")
	}
}

func TestStaticAssetsAreServed(t *testing.T) {
	handler := newTestHandler(nil, nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/static/css/app.css", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if !strings.Contains(recorder.Header().Get("Cache-Control"), "max-age") {
		t.Error("static asset does not have a cache policy")
	}
}
