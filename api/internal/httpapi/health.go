package httpapi

import (
	"net/http"

	"github.com/davcs86/happybabytracker/api/internal/store"
)

// Health reports 200 while the store answers and 503 otherwise.
func Health(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := s.Ping(r.Context()); err != nil {
			respondError(w, http.StatusServiceUnavailable, "unavailable")
			return
		}
		respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
