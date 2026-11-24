package httpapi

import (
	"net/http"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

// Health reports 200 while the store answers and the worker has ticked within WorkerStaleAfter; the
// first WorkerStaleAfter after construction counts from construction, before any tick has finished.
func Health(s *store.Store, now, lastTick func() time.Time) http.HandlerFunc {
	started := now()
	return func(w http.ResponseWriter, r *http.Request) {
		if err := s.Ping(r.Context()); err != nil {
			respondError(w, http.StatusServiceUnavailable, "unavailable")
			return
		}
		last := lastTick()
		if last.Before(started) {
			last = started
		}
		if now().Sub(last) > constants.WorkerStaleAfter {
			respondError(w, http.StatusServiceUnavailable, "unavailable")
			return
		}
		respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// RemindersHealth is Open value 30: 503 while a delivery is stuck or one failed lately; the body names nothing.
func RemindersHealth(s *store.Store, now func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ok, err := s.RemindersHealthy(r.Context(), now())
		if err != nil || !ok {
			respondJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
			return
		}
		respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
