package httpapi

import (
	"errors"
	"net/http"
	"slices"

	"github.com/davcs86/happybabytracker/api/internal/auth"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

type sessionInfoView struct {
	ID          string  `json:"id"`
	CreatedAt   string  `json:"created_at"`
	LastSeenAt  string  `json:"last_seen_at"`
	Current     bool    `json:"current"`
	DeviceLabel *string `json:"device_label"`
}

// pickUnit returns the unit to store: the stored one when the patch omits it, else the patch if allowed.
func pickUnit(stored string, patch *string, allowed ...string) (string, bool) {
	if patch == nil {
		return stored, true
	}
	return *patch, slices.Contains(allowed, *patch)
}

func (d Deps) updateProfile(w http.ResponseWriter, r *http.Request) {
	var in struct {
		DisplayName *string `json:"display_name"`
		Units       *struct {
			Weight *string `json:"weight"`
			Length *string `json:"length"`
			Volume *string `json:"volume"`
			Temp   *string `json:"temp"`
		} `json:"units"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	ctx, id := r.Context(), PrincipalFrom(r.Context()).ParentID
	parent, err := d.Store.ParentByID(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		respondError(w, http.StatusUnauthorized, "no_session")
		return
	}
	if err != nil {
		respondInternal(w, err)
		return
	}

	name := parent.DisplayName
	if in.DisplayName != nil {
		var ok bool
		if name, ok = cleanDisplayName(*in.DisplayName); !ok {
			respondInvalid(w, "display_name")
			return
		}
	}
	units := parent.Units
	if u := in.Units; u != nil {
		var okW, okL, okV, okT bool
		units.Weight, okW = pickUnit(units.Weight, u.Weight, "kg", "lb")
		units.Length, okL = pickUnit(units.Length, u.Length, "cm", "in")
		units.Volume, okV = pickUnit(units.Volume, u.Volume, "ml", "floz")
		units.Temp, okT = pickUnit(units.Temp, u.Temp, "c", "f")
		for _, bad := range []struct {
			ok    bool
			field string
		}{{okW, "units.weight"}, {okL, "units.length"}, {okV, "units.volume"}, {okT, "units.temp"}} {
			if !bad.ok {
				respondInvalid(w, bad.field)
				return
			}
		}
	}

	if err := d.Store.UpdateParentProfile(ctx, id, name, units, d.Now()); err != nil {
		respondInternal(w, err)
		return
	}
	parent.DisplayName, parent.Units = name, units
	respondJSON(w, http.StatusOK, newParentView(parent))
}

// changePassword counts against the login limiter, keeps the calling session and ends the others.
func (d Deps) changePassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Current string `json:"current"`
		New     string `json:"new"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	stored, ok := d.limitedSecret(w, r)
	if !ok {
		return
	}
	ctx, p := r.Context(), PrincipalFrom(r.Context())
	if !verifyPassword(stored, in.Current) {
		respondError(w, http.StatusForbidden, "wrong_password")
		return
	}
	hash, err := auth.HashPassword(in.New)
	if errors.Is(err, auth.ErrPasswordLength) {
		respondInvalid(w, "new")
		return
	}
	if err != nil {
		respondInternal(w, err)
		return
	}
	if err := d.Store.SetPassword(ctx, p.ParentID, hash, p.SessionID, d.Now()); err != nil {
		respondInternal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (d Deps) listSessions(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFrom(r.Context())
	sessions, err := d.Store.ListSessions(r.Context(), p.ParentID, d.Now())
	if err != nil {
		respondInternal(w, err)
		return
	}
	rows := make([]sessionInfoView, len(sessions))
	for i, s := range sessions {
		rows[i] = sessionInfoView{ID: s.ID, CreatedAt: formatTime(s.CreatedAt), LastSeenAt: formatTime(s.LastSeenAt), Current: s.ID == p.SessionID}
		if s.DeviceLabel != "" {
			rows[i].DeviceLabel = &s.DeviceLabel
		}
	}
	respondJSON(w, http.StatusOK, struct {
		Sessions []sessionInfoView `json:"sessions"`
	}{rows})
}

// deleteSession answers 404 for any session that is not the caller's, whether or not it exists.
func (d Deps) deleteSession(w http.ResponseWriter, r *http.Request) {
	err := d.Store.DeleteParentSession(r.Context(), PrincipalFrom(r.Context()).ParentID, r.PathValue("session"))
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
