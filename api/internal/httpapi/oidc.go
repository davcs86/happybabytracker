package httpapi

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/davcs86/happybabytracker/api/internal/auth"
	"github.com/davcs86/happybabytracker/api/internal/config"
	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

type oidcRequest struct {
	Iss         string `json:"iss"`
	Sub         string `json:"sub"`
	DeviceLabel string `json:"device_label"`
}

// decodeOIDC checks the body, then the mode, then device_label, iss and sub, in that order.
func (d Deps) decodeOIDC(w http.ResponseWriter, r *http.Request) (oidcRequest, bool) {
	var in oidcRequest
	if !decodeJSON(w, r, &in) || !d.allowMode(w, config.AuthOIDC, config.AuthBoth) {
		return in, false
	}
	if in.DeviceLabel != "" && !deviceLabelRE.MatchString(in.DeviceLabel) {
		respondInvalid(w, "device_label")
		return in, false
	}
	return in, validPair(w, in.Iss, in.Sub)
}

// validPair answers 422 on iss, then on sub.
func validPair(w http.ResponseWriter, iss, sub string) bool {
	switch {
	case !validIssuer(iss):
		respondInvalid(w, "iss")
	case sub == "" || len(sub) > constants.OIDCSubjectMaxLen:
		respondInvalid(w, "sub")
	default:
		return true
	}
	return false
}

func validIssuer(iss string) bool {
	if len(iss) > constants.OIDCIssuerMaxLen {
		return false
	}
	u, err := url.Parse(iss)
	return err == nil && u.Scheme == "https" && u.Host != ""
}

// oidcLogin answers an unknown pair and a disabled parent alike.
func (d Deps) oidcLogin(w http.ResponseWriter, r *http.Request) {
	in, ok := d.decodeOIDC(w, r)
	if !ok {
		return
	}
	token, hash, err := auth.NewToken()
	if err != nil {
		respondInternal(w, err)
		return
	}
	now := d.Now()
	kind, expires, err := d.Store.OIDCLogin(r.Context(), in.Iss, in.Sub, hash, now,
		now.Add(constants.ParentSessionTTL), now.Add(constants.AdminSessionTTL), in.DeviceLabel)
	if errors.Is(err, store.ErrNotLinked) {
		respondError(w, http.StatusForbidden, "not_linked")
		return
	}
	if err != nil {
		respondInternal(w, err)
		return
	}
	respondJSON(w, http.StatusOK, loginResponse{Token: token, Kind: kind, ExpiresAt: formatTime(expires)})
}

func (d Deps) redeemTokenOIDC(w http.ResponseWriter, r *http.Request) {
	in, ok := d.decodeOIDC(w, r)
	if !ok {
		return
	}
	token, hash, err := auth.NewToken()
	if err != nil {
		respondInternal(w, err)
		return
	}
	now := d.Now()
	expires := now.Add(constants.ParentSessionTTL)
	err = d.Store.RedeemInviteOIDC(r.Context(), auth.HashToken(r.PathValue("token")), in.Iss, in.Sub, hash, now, expires, in.DeviceLabel)
	switch {
	case errors.Is(err, store.ErrNotFound):
		respondError(w, http.StatusNotFound, "not_found")
	case errors.Is(err, store.ErrAlreadyLinked):
		respondError(w, http.StatusConflict, "already_linked")
	case err != nil:
		respondInternal(w, err)
	default:
		respondJSON(w, http.StatusOK, loginResponse{Token: token, Kind: kindParent, ExpiresAt: formatTime(expires)})
	}
}

// reauth runs one password verification, against dummyHash when the principal has no local password.
func (d Deps) reauth(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &in) || !d.allowMode(w, config.AuthBoth) {
		return
	}
	hash, ok := d.limitedSecret(w, r)
	if !ok {
		return
	}
	found := hash != ""
	if !found {
		hash = dummyHash()
	}
	// No local password is a refusal even if the dummy verifies: its input is public.
	if verified := verifyPassword(hash, in.Password); !found || !verified {
		respondError(w, http.StatusForbidden, "wrong_password")
		return
	}
	if err := d.Store.StampReauth(r.Context(), PrincipalFrom(r.Context()).SessionID, d.Now()); err != nil {
		respondInternal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// linkIdentity needs a re-auth on this session within ReauthWindow; a link leaves reauth_at as it is.
func (d Deps) linkIdentity(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Iss string `json:"iss"`
		Sub string `json:"sub"`
	}
	if !decodeJSON(w, r, &in) || !d.allowMode(w, config.AuthBoth) {
		return
	}
	p := PrincipalFrom(r.Context())
	if p.ReauthAt.IsZero() || d.Now().Sub(p.ReauthAt) > constants.ReauthWindow {
		respondError(w, http.StatusForbidden, "reauth_required")
		return
	}
	if !validPair(w, in.Iss, in.Sub) {
		return
	}
	err := d.Store.LinkIdentity(r.Context(), p, in.Iss, in.Sub)
	switch {
	case errors.Is(err, store.ErrAlreadyLinked):
		respondError(w, http.StatusConflict, "already_linked")
	case err != nil:
		respondInternal(w, err)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}
