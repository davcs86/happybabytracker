package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/davcs86/happybabytracker/api/internal/auth"
	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

type adminParentView struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Disabled    bool   `json:"disabled"`
	HasPassword bool   `json:"has_password"`
	CreatedAt   string `json:"created_at"`
}

func newAdminParentView(p model.Parent) adminParentView {
	return adminParentView{
		ID: p.ID, Username: p.Username, DisplayName: p.DisplayName,
		Disabled: p.Disabled, HasPassword: p.HasPassword, CreatedAt: formatTime(p.CreatedAt),
	}
}

// cleanDisplayName trims the name and counts code points, not bytes.
func cleanDisplayName(s string) (string, bool) {
	s = strings.TrimSpace(s)
	n := utf8.RuneCountInString(s)
	return s, n >= constants.DisplayNameMinLen && n <= constants.DisplayNameMaxLen
}

// issueLink stores a fresh one-time token and returns the link the admin hands to the parent.
func (d Deps) issueLink(r *http.Request, kind, parentID string, now time.Time) (link string, expires time.Time, err error) {
	token, hash, err := auth.NewToken()
	if err != nil {
		return "", time.Time{}, err
	}
	path, ttl := "/invite/", constants.InviteTTL
	if kind == model.TokenReset {
		path, ttl = "/reset/", constants.ResetTTL
	}
	expires = now.Add(ttl)
	if err := d.Store.IssueToken(r.Context(), kind, parentID, hash, expires); err != nil {
		return "", time.Time{}, err
	}
	return d.Cfg.PublicOrigin + path + token, expires, nil
}

func (d Deps) createParent(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	username, ok := auth.NormalizeUsername(in.Username)
	if !ok || username == d.Cfg.AdminUsername {
		respondInvalid(w, "username")
		return
	}
	name, ok := cleanDisplayName(in.DisplayName)
	if !ok {
		respondInvalid(w, "display_name")
		return
	}

	now := d.Now()
	parent, err := d.Store.CreateParent(r.Context(), username, name, now)
	if errors.Is(err, store.ErrConflict) {
		respondError(w, http.StatusConflict, "username_taken")
		return
	}
	if err != nil {
		respondInternal(w, err)
		return
	}
	link, expires, err := d.issueLink(r, model.TokenInvite, parent.ID, now)
	if err != nil {
		respondInternal(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, struct {
		Parent    adminParentView `json:"parent"`
		InviteURL string          `json:"invite_url"`
		ExpiresAt string          `json:"expires_at"`
	}{newAdminParentView(parent), link, formatTime(expires)})
}

func (d Deps) listParents(w http.ResponseWriter, r *http.Request) {
	parents, err := d.Store.ListParents(r.Context())
	if err != nil {
		respondInternal(w, err)
		return
	}
	rows := make([]adminParentView, len(parents))
	for i, p := range parents {
		rows[i] = newAdminParentView(p)
	}
	respondJSON(w, http.StatusOK, struct {
		Parents []adminParentView `json:"parents"`
	}{rows})
}

// resetLink issues an invite instead when the parent has never set a password; the response key says which.
func (d Deps) resetLink(w http.ResponseWriter, r *http.Request) {
	parent, err := d.Store.ParentByID(r.Context(), r.PathValue("parent"))
	if errors.Is(err, store.ErrNotFound) {
		respondError(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		respondInternal(w, err)
		return
	}
	kind := model.TokenReset
	if !parent.HasPassword {
		kind = model.TokenInvite
	}
	link, expires, err := d.issueLink(r, kind, parent.ID, d.Now())
	if err != nil {
		respondInternal(w, err)
		return
	}
	out := struct {
		ResetURL  string `json:"reset_url,omitempty"`
		InviteURL string `json:"invite_url,omitempty"`
		ExpiresAt string `json:"expires_at"`
	}{ExpiresAt: formatTime(expires)}
	if kind == model.TokenInvite {
		out.InviteURL = link
	} else {
		out.ResetURL = link
	}
	respondJSON(w, http.StatusOK, out)
}

func (d Deps) setDisabled(disabled bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := d.Store.SetParentDisabled(r.Context(), r.PathValue("parent"), disabled, d.Now())
		if errors.Is(err, store.ErrNotFound) {
			respondError(w, http.StatusNotFound, "not_found")
			return
		}
		if errors.Is(err, store.ErrLastGuardian) { // the admin learns nothing about which kid
			respondError(w, http.StatusConflict, "sole_guardian")
			return
		}
		if err != nil {
			respondInternal(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) peekToken(w http.ResponseWriter, r *http.Request) {
	kind, username, err := d.Store.PeekToken(r.Context(), auth.HashToken(r.PathValue("token")), d.Now())
	if errors.Is(err, store.ErrNotFound) {
		respondError(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		respondInternal(w, err)
		return
	}
	respondJSON(w, http.StatusOK, struct {
		Kind     string `json:"kind"`
		Username string `json:"username"`
	}{kind, username})
}

// redeemToken refuses an unusable token before it hashes anything, so a bad link costs no KDF run.
func (d Deps) redeemToken(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if !d.Limiter.Allow("addr:" + r.Header.Get(headerClientAddress)) {
		respondRateLimited(w)
		return
	}

	ctx, tokenHash, now := r.Context(), auth.HashToken(r.PathValue("token")), d.Now()
	if _, _, err := d.Store.PeekToken(ctx, tokenHash, now); errors.Is(err, store.ErrNotFound) {
		respondError(w, http.StatusNotFound, "not_found")
		return
	} else if err != nil {
		respondInternal(w, err)
		return
	}
	secret, err := auth.HashPassword(in.Password)
	if errors.Is(err, auth.ErrPasswordLength) {
		respondInvalid(w, "password")
		return
	}
	if err != nil {
		respondInternal(w, err)
		return
	}
	parentID, err := d.Store.RedeemToken(ctx, tokenHash, secret, now)
	if errors.Is(err, store.ErrNotFound) {
		respondError(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		respondInternal(w, err)
		return
	}

	session, sessionHash, err := auth.NewToken()
	if err != nil {
		respondInternal(w, err)
		return
	}
	expires := now.Add(constants.ParentSessionTTL)
	if _, err := d.Store.CreateSession(ctx, kindParent, parentID, sessionHash, now, expires); err != nil {
		respondInternal(w, err)
		return
	}
	respondJSON(w, http.StatusOK, loginResponse{Token: session, Kind: kindParent, ExpiresAt: formatTime(expires)})
}
