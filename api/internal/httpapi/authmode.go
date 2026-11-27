package httpapi

import (
	"net/http"
	"slices"

	"github.com/davcs86/happybabytracker/api/internal/config"
)

// allowMode runs right after the body is decoded, before any lookup, hash or limiter count.
func (d Deps) allowMode(w http.ResponseWriter, allowed ...config.AuthMode) bool {
	if slices.Contains(allowed, d.Cfg.AuthMode) {
		return true
	}
	respondError(w, http.StatusForbidden, "mode_refused")
	return false
}

func (d Deps) authMode(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, struct {
		Mode config.AuthMode `json:"mode"`
	}{d.Cfg.AuthMode})
}
