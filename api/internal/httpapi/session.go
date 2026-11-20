package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/auth"
	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

const headerClientAddress = "X-Client-Address"

// Replaced in tests to count verifications.
var verifyPassword = auth.VerifyPassword

// A failed hash leaves "", which VerifyPassword rejects without running the KDF.
var dummyHash = sync.OnceValue(func() string {
	h, _ := auth.HashPassword("dummy password for timing")
	return h
})

type sessionView struct {
	Kind      string      `json:"kind"`
	ExpiresAt string      `json:"expires_at"`
	Parent    *parentView `json:"parent,omitempty"`
}

type parentView struct {
	ID          string    `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	Units       unitsView `json:"units"`
}

type unitsView struct {
	Weight string `json:"weight"`
	Length string `json:"length"`
	Volume string `json:"volume"`
	Temp   string `json:"temp"`
}

type loginResponse struct {
	Token     string `json:"token"`
	Kind      string `json:"kind"`
	ExpiresAt string `json:"expires_at"`
}

func formatTime(t time.Time) string { return t.UTC().Format(time.RFC3339) }

func respondRateLimited(w http.ResponseWriter) {
	w.Header().Set("Retry-After", strconv.Itoa(int(constants.LimiterWindow/time.Second)))
	respondError(w, http.StatusTooManyRequests, "rate_limited")
}

// login answers every credential failure alike, after one password verification.
func (d Deps) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if !d.Limiter.Allow("addr:" + r.Header.Get(headerClientAddress)) {
		respondRateLimited(w)
		return
	}
	username, valid := auth.NormalizeUsername(in.Username)
	if valid && !d.Limiter.Allow("user:"+username) {
		respondRateLimited(w)
		return
	}

	kind, parentID, authed, err := d.checkCredentials(r, username, valid, in.Password)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal")
		return
	}
	if !authed {
		respondError(w, http.StatusUnauthorized, "invalid_credentials")
		return
	}

	token, hash, err := auth.NewToken()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal")
		return
	}
	now := d.Now()
	ttl := constants.ParentSessionTTL
	if kind == kindAdmin {
		ttl = constants.AdminSessionTTL
	}
	if _, err := d.Store.CreateSession(r.Context(), kind, parentID, hash, now, now.Add(ttl)); err != nil {
		respondError(w, http.StatusInternalServerError, "internal")
		return
	}
	respondJSON(w, http.StatusOK, loginResponse{Token: token, Kind: kind, ExpiresAt: formatTime(now.Add(ttl))})
}

// checkCredentials runs one password verification on every failing path; only the admin success skips it.
func (d Deps) checkCredentials(r *http.Request, username string, valid bool, password string) (kind, parentID string, authed bool, err error) {
	if valid && username == d.Cfg.AdminUsername {
		if auth.CheckAdmin(d.Cfg, username, password) {
			return kindAdmin, "", true, nil
		}
		verifyPassword(dummyHash(), password)
		return "", "", false, nil
	}

	hash, found := dummyHash(), false
	var parent struct {
		id       string
		disabled bool
	}
	if valid {
		p, secret, err := d.Store.ParentByUsername(r.Context(), username)
		switch {
		case err == nil:
			parent.id, parent.disabled = p.ID, p.Disabled
			if secret != "" {
				hash, found = secret, true
			}
		case !errors.Is(err, store.ErrNotFound):
			return "", "", false, err
		}
	}
	verified := verifyPassword(hash, password)
	if !found || !verified || parent.disabled {
		return "", "", false, nil
	}
	return kindParent, parent.id, true, nil
}

func (d Deps) currentSession(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFrom(r.Context())
	view := sessionView{Kind: p.Kind, ExpiresAt: formatTime(p.ExpiresAt)}
	if p.Kind == kindParent {
		parent, err := d.Store.ParentByID(r.Context(), p.ParentID)
		if errors.Is(err, store.ErrNotFound) {
			respondError(w, http.StatusUnauthorized, "no_session")
			return
		}
		if err != nil {
			respondError(w, http.StatusInternalServerError, "internal")
			return
		}
		view.Parent = &parentView{
			ID: parent.ID, Username: parent.Username, DisplayName: parent.DisplayName,
			Units: unitsView{Weight: parent.Units.Weight, Length: parent.Units.Length, Volume: parent.Units.Volume, Temp: parent.Units.Temp},
		}
	}
	respondJSON(w, http.StatusOK, view)
}

func (d Deps) logout(w http.ResponseWriter, r *http.Request) {
	if err := d.Store.DeleteSession(r.Context(), PrincipalFrom(r.Context()).SessionID); err != nil {
		respondError(w, http.StatusInternalServerError, "internal")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
