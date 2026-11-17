package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/jho-on/moada/internal/config"
	"github.com/jho-on/moada/internal/health"
	"github.com/jho-on/moada/internal/web"
	"github.com/jho-on/moada/internal/web/templates"
)

const (
	statusCheckTimeout = 3 * time.Second
	requestIDHeader    = "X-Request-ID"
)

// Server holds the HTTP dependencies shared by every handler.
type Server struct {
	logger       *slog.Logger
	checks       health.Checker
	isProduction bool
}

// New returns the application HTTP handler.
func New(logger *slog.Logger, checks health.Checker, environment string) http.Handler {
	server := &Server{
		logger:       logger,
		checks:       checks,
		isProduction: environment == config.EnvironmentProduction,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", server.handleHome)
	mux.HandleFunc("GET /healthz", server.handleLiveness)
	mux.HandleFunc("GET /readyz", server.handleReadiness)
	mux.HandleFunc("GET /partials/status", server.handleStatus)
	mux.Handle("GET /static/", http.StripPrefix("/static/", staticHandler()))
	mux.HandleFunc("/", server.handleNotFound)

	return server.withRequestID(
		server.securityHeaders(
			server.logRequests(
				server.recoverPanic(mux),
			),
		),
	)
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.HomePage().Render(r.Context(), w); err != nil {
		s.logger.Error("render home page", "error", err, "request_id", requestIDFrom(r.Context()))
	}
}

func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	if err := templates.NotFoundPage().Render(r.Context(), w); err != nil {
		s.logger.Error("render not found page", "error", err, "request_id", requestIDFrom(r.Context()))
	}
}

func (s *Server) handleLiveness(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), statusCheckTimeout)
	defer cancel()

	report := s.checks.Check(ctx)
	s.logUnhealthy(report, "readiness check failed")

	response := struct {
		Status     string          `json:"status"`
		Components map[string]bool `json:"components"`
		CheckedAt  time.Time       `json:"checked_at"`
	}{
		Status:     "ok",
		Components: make(map[string]bool, len(report.Components)),
		CheckedAt:  report.CheckedAt,
	}

	writeJSONStatus := http.StatusOK
	if !report.OK() {
		response.Status = "unavailable"
		writeJSONStatus = http.StatusServiceUnavailable
	}
	for _, component := range report.Components {
		response.Components[component.Name] = component.Healthy
	}

	s.writeJSON(w, writeJSONStatus, response)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), statusCheckTimeout)
	defer cancel()

	report := s.checks.Check(ctx)
	s.logUnhealthy(report, "status check failed")

	view := templates.StatusView{
		Healthy:   report.OK(),
		CheckedAt: report.CheckedAt,
	}
	for _, component := range report.Components {
		view.Components = append(view.Components, templates.StatusComponent{
			Name:    component.Name,
			Healthy: component.Healthy,
		})
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if err := templates.StatusPanel(view).Render(r.Context(), w); err != nil {
		s.logger.Error("render status panel", "error", err, "request_id", requestIDFrom(r.Context()))
	}
}

func (s *Server) logUnhealthy(report health.Report, message string) {
	for _, component := range report.Components {
		if !component.Healthy {
			s.logger.Warn(message,
				"component", component.Name,
				"error", component.Err,
			)
		}
	}
}

func staticHandler() http.Handler {
	files := http.FileServerFS(web.StaticFS())
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=3600")
		files.ServeHTTP(w, r)
	})
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		// The status line is already sent, so the error can only be logged.
		s.logger.Error("write JSON response", "error", err)
	}
}

type requestIDContextKey struct{}

func (s *Server) withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := newRequestID()
		w.Header().Set(requestIDHeader, id)

		ctx := context.WithValue(r.Context(), requestIDContextKey{}, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func requestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDContextKey{}).(string)
	return id
}

func newRequestID() string {
	var buffer [8]byte
	if _, err := rand.Read(buffer[:]); err != nil {
		// crypto/rand failing is exceptional; a timestamp still gives some
		// correlation value without affecting correctness.
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(buffer[:])
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(status int) {
	if r.status != 0 {
		return
	}
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.WriteHeader(http.StatusOK)
	}
	written, err := r.ResponseWriter.Write(data)
	r.bytes += written
	return written, err
}

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w}

		next.ServeHTTP(recorder, r)

		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}

		// Static assets are noisy and not useful for operational debugging.
		if strings.HasPrefix(r.URL.Path, "/static/") {
			return
		}

		s.logger.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
			"bytes", recorder.bytes,
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", requestIDFrom(r.Context()),
		)
	})
}

func (s *Server) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				s.logger.Error("panic recovered",
					"panic", recovered,
					"path", r.URL.Path,
					"request_id", requestIDFrom(r.Context()),
					"stack", string(debug.Stack()),
				)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header.Set("Content-Security-Policy", strings.Join([]string{
			"default-src 'self'",
			"script-src 'self'",
			"style-src 'self'",
			"img-src 'self' data:",
			"font-src 'self'",
			"connect-src 'self'",
			"object-src 'none'",
			"base-uri 'none'",
			"form-action 'self'",
			"frame-ancestors 'none'",
		}, "; "))
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("X-Frame-Options", "DENY")
		header.Set("Referrer-Policy", "no-referrer")
		header.Set("Permissions-Policy", "geolocation=(), camera=(), microphone=()")

		if s.isProduction {
			header.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		next.ServeHTTP(w, r)
	})
}
