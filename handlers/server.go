package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"auto-focus.app/cloud/internal/ratelimit"
	"auto-focus.app/cloud/storage"
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

	mux.Handle("/health", http.HandlerFunc(s.Health))
	// mux.Handle("/api/v1/licenses", http.HandlerFunc(db.list))
	// mux.Handle("/api/v1/licenses/validate", s.chain(s.withLogging, s.withRateLimit)(http.HandlerFunc(s.ValidateLicense)))
	mux.Handle("/api/v1/webhooks/stripe", s.chain(s.withLogging, s.withRateLimit)(http.HandlerFunc(s.stripe)))

	return s
}

func (s *Server) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
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

func (s *Server) withRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.RateLimitter.Allow(r.RemoteAddr) {
			log.Default().Printf("Rate limit reached for %s\n", r.RemoteAddr)
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		elapsedTime := time.Since(start)
		log.Printf("[%s] [%s] [%s]\n", r.Method, r.URL.Path, elapsedTime)
	})
}
