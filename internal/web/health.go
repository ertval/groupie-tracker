package web

import (
	"encoding/json"
	"net/http"
	"time"
)

// Health provides a health check endpoint.
func (s *Server) Health(w http.ResponseWriter, r *http.Request) {
	response := map[string]any{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"stats":     s.svc.Stats(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
