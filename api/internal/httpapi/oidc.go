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
	switch {
	case in.DeviceLabel != "" && !deviceLabelRE.MatchString(in.DeviceLabel):
		respondInvalid(w, "device_label")
	case !validIssuer(in.Iss):
		respondInvalid(w, "iss")
	case in.Sub == "" || len(in.Sub) > constants.OIDCSubjectMaxLen:
		respondInvalid(w, "sub")
	default:
		return in, true
	}
	return in, false
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
