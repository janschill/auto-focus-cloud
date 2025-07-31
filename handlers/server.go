package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"auto-focus.app/cloud/internal/logger"
	"auto-focus.app/cloud/internal/ratelimit"
	"auto-focus.app/cloud/storage"
	"github.com/getsentry/sentry-go"
)

type Server struct {
	Mux          *http.ServeMux
	Storage      storage.Storage
	RateLimitter ratelimit.RateLimit
}

func NewHttpServer(db storage.Storage) *Server {
	mux := http.NewServeMux()

	s := &Server{
		Mux:          mux,
		Storage:      db,
		RateLimitter: ratelimit.New(10, time.Minute),
	}

	mux.Handle("/v1/health", http.HandlerFunc(s.Health))
	// mux.Handle("/v1/licenses", http.HandlerFunc(db.list))
	mux.Handle("/v1/licenses/validate", s.chain(s.withCORS, s.withLogging, s.withRateLimit)(http.HandlerFunc(s.ValidateLicense)))
	mux.Handle("/v1/webhooks/stripe", s.chain(s.withCORS, s.withLogging, s.withRateLimit)(http.HandlerFunc(s.Stripe)))

	return s
}

type HealthResponse struct {
	Status      string `json:"status"`
	Timestamp   string `json:"timestamp"`
	Environment string `json:"environment"`
	Database    string `json:"database"`
}

func (s *Server) Health(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	logger.Debug("Health check requested", map[string]interface{}{
		"remote_addr": r.RemoteAddr,
	})

	// Test database connectivity
	dbStatus := "ok"
	_, err := s.Storage.GetCustomer(ctx, "health-check-test")
	if err != nil {
		sentry.CaptureException(err)
		logger.Error("Database health check failed", map[string]interface{}{
			"error": err.Error(),
		})
		dbStatus = "error"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	environment := os.Getenv("ENVIRONMENT")
	if environment == "" {
		environment = "production"
	}

	response := HealthResponse{
		Status:      dbStatus,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Environment: environment,
		Database:    dbStatus,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		sentry.CaptureException(err)
		logger.Error("Failed to encode health response", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

type Middleware func(next http.Handler) http.Handler

func (s *Server) chain(middleware ...Middleware) func(http.Handler) http.Handler {
	return func(final http.Handler) http.Handler {
		handler := final
		for m := range middleware {
			handler = middleware[len(middleware)-1-m](handler)
		}
		return handler
	}
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) withRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.RateLimitter.Allow(r.RemoteAddr) {
			logger.Warn("Rate limit exceeded", map[string]interface{}{
				"remote_addr": r.RemoteAddr,
				"path":        r.URL.Path,
				"method":      r.Method,
				"user_agent":  r.Header.Get("User-Agent"),
			})
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a custom ResponseWriter to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		logger.Info("HTTP request", map[string]interface{}{
			"method":       r.Method,
			"path":         r.URL.Path,
			"remote_addr":  r.RemoteAddr,
			"user_agent":   r.Header.Get("User-Agent"),
			"status_code":  rw.statusCode,
			"duration_ms":  duration.Milliseconds(),
			"content_type": r.Header.Get("Content-Type"),
		})
	})
}

// Custom ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
