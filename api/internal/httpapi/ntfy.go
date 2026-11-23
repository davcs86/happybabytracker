package httpapi

import (
	"errors"
	"net/http"

	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

// ntfyView is served only on the owner's /v1/me routes: the topic is a capability.
type ntfyView struct {
	Enabled bool     `json:"enabled"`
	Server  string   `json:"server"`
	Topic   *string  `json:"topic"`
	Kids    []string `json:"kids"`
}

// ntfyOn answers 409 ntfy_unavailable and is false while ntfy is off.
func (d Deps) ntfyOn(w http.ResponseWriter) bool {
	if !d.Cfg.NtfyEnabled {
		respondError(w, http.StatusConflict, "ntfy_unavailable")
		return false
	}
	return true
}

func (d Deps) respondNtfy(w http.ResponseWriter, status int, t *model.NtfyTopic) {
	v := ntfyView{Enabled: true, Server: d.Cfg.Ntfy.PublicURL, Kids: []string{}}
	if v.Server == "" {
		v.Server = d.Cfg.Ntfy.URL
	}
	if t != nil {
		v.Topic, v.Kids = &t.Topic, t.Kids
	}
	respondJSON(w, status, v)
}

func (d Deps) ntfyState(w http.ResponseWriter, r *http.Request) {
	if !d.Cfg.NtfyEnabled {
		respondJSON(w, http.StatusOK, struct {
			Enabled bool `json:"enabled"`
		}{false})
		return
	}
	t, err := d.Store.NtfyTopic(r.Context(), PrincipalFrom(r.Context()).ParentID)
	switch {
	case errors.Is(err, store.ErrNotFound):
		d.respondNtfy(w, http.StatusOK, nil)
	case err != nil:
		respondInternal(w, err)
	default:
		d.respondNtfy(w, http.StatusOK, &t)
	}
}

// setNtfyTopic serves PUT (turn on; an existing topic is kept) and POST regenerate (404 without a topic).
func (d Deps) setNtfyTopic(regenerate bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.ntfyOn(w) {
			return
		}
		t, created, err := d.Store.SetNtfyTopic(r.Context(), PrincipalFrom(r.Context()).ParentID, d.Cfg.Ntfy.TopicPrefix, regenerate, d.Now())
		switch {
		case errors.Is(err, store.ErrNotFound):
			respondError(w, http.StatusNotFound, "not_found")
		case err != nil:
			respondInternal(w, err)
		case created:
			d.respondNtfy(w, http.StatusCreated, &t)
		default:
			d.respondNtfy(w, http.StatusOK, &t)
		}
	}
}

func (d Deps) deleteNtfyTopic(w http.ResponseWriter, r *http.Request) {
	err := d.Store.DeleteNtfyTopic(r.Context(), PrincipalFrom(r.Context()).ParentID)
	if errors.Is(err, store.ErrNotFound) {
		respondError(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		respondInternal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// setNtfyOptin and clearNtfyOptin answer a caller without a topic with the unknown-kid 404.
func (d Deps) setNtfyOptin(w http.ResponseWriter, r *http.Request) {
	if !d.ntfyOn(w) {
		return
	}
	if kidWriteOK(w, d.Store.SetNtfyOptin(r.Context(), r.PathValue("kid"), PrincipalFrom(r.Context()).ParentID)) {
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) clearNtfyOptin(w http.ResponseWriter, r *http.Request) {
	if kidWriteOK(w, d.Store.ClearNtfyOptin(r.Context(), r.PathValue("kid"), PrincipalFrom(r.Context()).ParentID)) {
		w.WriteHeader(http.StatusNoContent)
	}
}
