package httpapi

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/auth"
	"github.com/davcs86/happybabytracker/api/internal/config"
	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

const headerClientAddress = "X-Client-Address"

var deviceLabelRE = regexp.MustCompile(constants.DeviceLabelPattern)

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
	Admin     *adminView  `json:"admin,omitempty"`
}

type adminView struct {
	Username  string `json:"username"`
	SSOLinked bool   `json:"sso_linked"`
}

type parentView struct {
	ID          string    `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	Units       unitsView `json:"units"`
	// PasswordChangedAt is null when the date is unknown.
	PasswordChangedAt *string `json:"password_changed_at"`
	SSOLinked         bool    `json:"sso_linked"`
}

func newParentView(p model.Parent) parentView {
	return parentView{
		ID: p.ID, Username: p.Username, DisplayName: p.DisplayName,
		Units:             unitsView{Weight: p.Units.Weight, Length: p.Units.Length, Volume: p.Units.Volume, Temp: p.Units.Temp},
		PasswordChangedAt: optionalInstant(p.PasswordChangedAt),
		SSOLinked:         p.SSOLinked,
	}
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
		Username    string `json:"username"`
		Password    string `json:"password"`
		DeviceLabel string `json:"device_label"`
	}
	if !decodeJSON(w, r, &in) || !d.allowMode(w, config.AuthLocal, config.AuthBoth) {
		return
	}
	if in.DeviceLabel != "" && !deviceLabelRE.MatchString(in.DeviceLabel) {
		respondInvalid(w, "device_label")
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

	kind, principalID, authed, err := d.checkCredentials(r, username, valid, in.Password)
	if err != nil {
		respondInternal(w, err)
		return
	}
	if !authed {
		respondError(w, http.StatusUnauthorized, "invalid_credentials")
		return
	}

	token, hash, err := auth.NewToken()
	if err != nil {
		respondInternal(w, err)
		return
	}
	now := d.Now()
	ttl := constants.ParentSessionTTL
	if kind == kindAdmin {
		ttl = constants.AdminSessionTTL
	}
	label := in.DeviceLabel
	if kind == kindAdmin {
		label = ""
	}
	if _, err := d.Store.CreateSession(r.Context(), kind, principalID, hash, now, now.Add(ttl), label); err != nil {
		respondInternal(w, err)
		return
	}
	respondJSON(w, http.StatusOK, loginResponse{Token: token, Kind: kind, ExpiresAt: formatTime(now.Add(ttl))})
}

// checkCredentials runs exactly one password verification on every path, against dummyHash when no hash was found.
func (d Deps) checkCredentials(r *http.Request, username string, valid bool, password string) (kind, principalID string, authed bool, err error) {
	hash, found := dummyHash(), false
	var disabled bool
	if valid {
		a, secret, err := d.Store.AdminByUsername(r.Context(), username)
		switch {
		case err == nil:
			kind, principalID = kindAdmin, a.ID
			if secret != "" {
				hash, found = secret, true
			}
		case errors.Is(err, store.ErrNotFound):
			p, secret, err := d.Store.ParentByUsername(r.Context(), username)
			switch {
			case err == nil:
				kind, principalID, disabled = kindParent, p.ID, p.Disabled
				if secret != "" {
					hash, found = secret, true
				}
			case !errors.Is(err, store.ErrNotFound):
				return "", "", false, err
			}
		default:
			return "", "", false, err
		}
	}
	verified := verifyPassword(hash, password)
	if !found || !verified || disabled {
		return "", "", false, nil
	}
	return kind, principalID, true, nil
}

// limitedSecret counts addr: then user: against the login limiter and returns the caller's local
// password hash, "" when none; on false it has answered.
func (d Deps) limitedSecret(w http.ResponseWriter, r *http.Request) (string, bool) {
	if !d.Limiter.Allow("addr:" + r.Header.Get(headerClientAddress)) {
		respondRateLimited(w)
		return "", false
	}
	ctx, p := r.Context(), PrincipalFrom(r.Context())
	var username, hash string
	if p.Kind == kindAdmin {
		a, secret, err := d.Store.AdminByID(ctx, p.AdminID)
		if err != nil {
			respondPrincipalErr(w, err)
			return "", false
		}
		username, hash = a.Username, secret
	} else {
		parent, err := d.Store.ParentByID(ctx, p.ParentID)
		if err != nil {
			respondPrincipalErr(w, err)
			return "", false
		}
		username = parent.Username
		if hash, err = d.Store.LocalSecret(ctx, p.ParentID); err != nil && !errors.Is(err, store.ErrNotFound) {
			respondInternal(w, err)
			return "", false
		}
	}
	if !d.Limiter.Allow("user:" + username) {
		respondRateLimited(w)
		return "", false
	}
	return hash, true
}

func respondPrincipalErr(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) {
		respondError(w, http.StatusUnauthorized, "no_session")
		return
	}
	respondInternal(w, err)
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
			respondInternal(w, err)
			return
		}
		pv := newParentView(parent)
		view.Parent = &pv
	} else {
		a, _, err := d.Store.AdminByID(r.Context(), p.AdminID)
		if errors.Is(err, store.ErrNotFound) {
			respondError(w, http.StatusUnauthorized, "no_session")
			return
		}
		if err != nil {
			respondInternal(w, err)
			return
		}
		view.Admin = &adminView{Username: a.Username, SSOLinked: a.SSOLinked}
	}
	respondJSON(w, http.StatusOK, view)
}

func (d Deps) logout(w http.ResponseWriter, r *http.Request) {
	if err := d.Store.DeleteSession(r.Context(), PrincipalFrom(r.Context()).SessionID); err != nil {
		respondInternal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
