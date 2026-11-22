package httpapi

import (
	"net/http"
	"slices"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

type moduleView struct {
	Module     model.Module `json:"module"`
	Enabled    bool         `json:"enabled"`
	Default    bool         `json:"default"`
	Overridden bool         `json:"overridden"`
}

func newModuleViews(states []store.ModuleState) []moduleView {
	out := make([]moduleView, len(states))
	for i, st := range states {
		out[i] = moduleView(st)
	}
	return out
}

// kidModules loads the kid in the path and its module states at now; it answers itself on failure.
func (d Deps) kidModules(w http.ResponseWriter, r *http.Request, now time.Time) (model.Kid, *time.Location, []store.ModuleState, bool) {
	kid, ok := d.loadKid(w, r)
	if !ok {
		return model.Kid{}, nil, nil, false
	}
	loc, zoneOK := kidLocation(kid.TimeZone)
	if !zoneOK {
		respondInternal(w, errKidZone)
		return model.Kid{}, nil, nil, false
	}
	states, err := d.Store.Modules(r.Context(), kid.ID, loc, now)
	if !kidWriteOK(w, err) {
		return model.Kid{}, nil, nil, false
	}
	return kid, loc, states, true
}

func (d Deps) listModules(w http.ResponseWriter, r *http.Request) {
	_, _, states, ok := d.kidModules(w, r, d.Now())
	if !ok {
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"modules": newModuleViews(states)})
}

func (d Deps) setModule(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Enabled *bool `json:"enabled"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	m := model.Module(r.PathValue("module"))
	if !slices.Contains(model.Modules, m) {
		respondNoKid(w)
		return
	}
	if in.Enabled == nil {
		respondInvalid(w, "enabled")
		return
	}
	err := d.Store.SetModule(r.Context(), r.PathValue("kid"), PrincipalFrom(r.Context()).ParentID, m, *in.Enabled, d.Now())
	if kidWriteOK(w, err) {
		w.WriteHeader(http.StatusNoContent)
	}
}
