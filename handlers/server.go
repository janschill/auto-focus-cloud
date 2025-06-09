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
	mux.Handle("/api/v1/licenses/validate", s.withRateLimit(http.HandlerFunc(s.ValidateLicense)))
	// mux.Handle("/api/v1/webhooks/stripe", http.HandlerFunc(s.stripe))

	return s
}

func (s *Server) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) withRateLimit(handler http.HandlerFunc) http.HandlerFunc {
	// Rate limiting logic
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.RateLimitter.Allow(r.RemoteAddr) {
			log.Default().Printf("Rate limit reached for %s\n", r.RemoteAddr)
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		handler(w, r)
	}
}
