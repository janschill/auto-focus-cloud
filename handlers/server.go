package handlers

import (
	"encoding/json"
	"net/http"

	"auto-focus.app/cloud/storage"
)

type Server struct {
	Mux     *http.ServeMux
	Storage storage.Database
}

func NewHttpServer(db storage.Database) *Server {
	mux := http.NewServeMux()

	s := &Server{
		Mux:     mux,
		Storage: db,
	}

	mux.Handle("/health", http.HandlerFunc(s.Health))
	// mux.Handle("/api/v1/licenses", http.HandlerFunc(db.list))
	mux.Handle("/api/v1/licenses/validate", http.HandlerFunc(s.ValidateLicense))
	// mux.Handle("/api/v1/webhooks/stripe", http.HandlerFunc(s.stripe))

	return s
}

func (s *Server) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
