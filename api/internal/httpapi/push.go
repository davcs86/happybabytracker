package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/push"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

// deviceView never carries the endpoint or the keys.
type deviceView struct {
	ID        string   `json:"id"`
	Label     string   `json:"label"`
	CreatedAt string   `json:"created_at"`
	Kids      []string `json:"kids"`
}

func newDeviceView(d model.PushDevice) deviceView {
	return deviceView{ID: d.ID, Label: d.Label, CreatedAt: formatTime(d.CreatedAt), Kids: d.Kids}
}

var errNoResolver = errors.New("web push registration has no resolver")

// pushEnabled is the one test of "Web Push is on": a sender and the public key it serves.
func (d Deps) pushEnabled() bool { return d.Push != nil && d.VAPIDPublicKey != "" }

// pushOn answers 409 push_unavailable and is false while Web Push is off.
func (d Deps) pushOn(w http.ResponseWriter) bool {
	if !d.pushEnabled() {
		respondError(w, http.StatusConflict, "push_unavailable")
		return false
	}
	return true
}

func cleanLabel(s *string) (string, bool) {
	if s == nil {
		return "", false
	}
	label := strings.TrimSpace(*s)
	n := utf8.RuneCountInString(label)
	return label, n >= 1 && n <= constants.PushLabelMaxLen
}

func respondDevice(w http.ResponseWriter, status int, dev model.PushDevice) {
	respondJSON(w, status, struct {
		Device deviceView `json:"device"`
	}{newDeviceView(dev)})
}

func (d Deps) pushState(w http.ResponseWriter, _ *http.Request) {
	if !d.pushEnabled() {
		respondJSON(w, http.StatusOK, struct {
			Enabled bool `json:"enabled"`
		}{false})
		return
	}
	respondJSON(w, http.StatusOK, struct {
		Enabled   bool   `json:"enabled"`
		PublicKey string `json:"public_key"`
	}{true, d.VAPIDPublicKey})
}

func (d Deps) listDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := d.Store.Devices(r.Context(), PrincipalFrom(r.Context()).ParentID)
	if err != nil {
		respondInternal(w, err)
		return
	}
	views := make([]deviceView, len(devices))
	for i, dev := range devices {
		views[i] = newDeviceView(dev)
	}
	respondJSON(w, http.StatusOK, struct {
		Devices []deviceView `json:"devices"`
	}{views})
}

// upsertDevice checks endpoint, keys, label in that order and stores the keys as canonical unpadded base64url.
func (d Deps) upsertDevice(w http.ResponseWriter, r *http.Request) {
	if !d.pushOn(w) {
		return
	}
	var in struct {
		Endpoint *string `json:"endpoint"`
		Keys     *struct {
			P256dh *string `json:"p256dh"`
			Auth   *string `json:"auth"`
		} `json:"keys"`
		Label *string `json:"label"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if in.Endpoint == nil {
		respondInvalid(w, "endpoint")
		return
	}
	var err error
	switch {
	case d.Cfg.AllowFakePush:
		err = push.ValidateEndpointSyntax(*in.Endpoint)
	case d.Resolver == nil:
		respondInternal(w, errNoResolver)
		return
	default:
		err = push.ValidateEndpoint(r.Context(), d.Resolver, *in.Endpoint)
	}
	if err != nil {
		respondInvalid(w, "endpoint")
		return
	}
	if in.Keys == nil || in.Keys.P256dh == nil || in.Keys.Auth == nil {
		respondInvalid(w, "keys")
		return
	}
	p256dh, auth, err := push.ParseKeys(*in.Keys.P256dh, *in.Keys.Auth)
	if err != nil {
		respondInvalid(w, "keys")
		return
	}
	label, ok := cleanLabel(in.Label)
	if !ok {
		respondInvalid(w, "label")
		return
	}
	enc := base64.RawURLEncoding
	dev, created, err := d.Store.UpsertDevice(r.Context(), PrincipalFrom(r.Context()).ParentID, canonicalEndpoint(*in.Endpoint),
		enc.EncodeToString(p256dh), enc.EncodeToString(auth), label, d.Now())
	switch {
	case errors.Is(err, store.ErrConflict):
		respondError(w, http.StatusConflict, "limit")
	case errors.Is(err, store.ErrNotFound):
		respondError(w, http.StatusNotFound, "not_found")
	case err != nil:
		respondInternal(w, err)
	case created:
		respondDevice(w, http.StatusCreated, dev)
	default:
		respondDevice(w, http.StatusOK, dev)
	}
}

// canonicalEndpoint lowercases the host, drops a trailing dot and an explicit :443, and keeps the rest byte for byte.
// The endpoint must already have passed validation, which runs on the original input.
func canonicalEndpoint(endpoint string) string {
	u, err := url.Parse(endpoint)
	if err != nil {
		return endpoint
	}
	start := strings.Index(endpoint, "://") + len("://")
	end := len(endpoint)
	if i := strings.IndexAny(endpoint[start:], "/?#"); i >= 0 {
		end = start + i
	}
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if p := u.Port(); p != "" && p != "443" {
		host += ":" + p
	}
	return endpoint[:start] + host + endpoint[end:]
}

func (d Deps) renameDevice(w http.ResponseWriter, r *http.Request) {
	if !d.pushOn(w) {
		return
	}
	var in struct {
		Label *string `json:"label"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	label, ok := cleanLabel(in.Label)
	if !ok {
		respondInvalid(w, "label")
		return
	}
	dev, err := d.Store.RenameDevice(r.Context(), PrincipalFrom(r.Context()).ParentID, r.PathValue("device"), label, d.Now())
	if errors.Is(err, store.ErrNotFound) {
		respondError(w, http.StatusNotFound, "not_found")
		return
	}
	if err != nil {
		respondInternal(w, err)
		return
	}
	respondDevice(w, http.StatusOK, dev)
}

// deleteDevice answers 404 for any device that is not the caller's, whether or not it exists.
func (d Deps) deleteDevice(w http.ResponseWriter, r *http.Request) {
	err := d.Store.DeleteDevice(r.Context(), PrincipalFrom(r.Context()).ParentID, r.PathValue("device"))
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

// setOptin and clearOptin answer a device that is not the caller's with the unknown-kid 404.
func (d Deps) setOptin(w http.ResponseWriter, r *http.Request) {
	if !d.pushOn(w) {
		return
	}
	err := d.Store.SetPushOptin(r.Context(), r.PathValue("kid"), PrincipalFrom(r.Context()).ParentID, r.PathValue("device"))
	if kidWriteOK(w, err) {
		w.WriteHeader(http.StatusNoContent)
	}
}

func (d Deps) clearOptin(w http.ResponseWriter, r *http.Request) {
	err := d.Store.ClearPushOptin(r.Context(), r.PathValue("kid"), PrincipalFrom(r.Context()).ParentID, r.PathValue("device"))
	if kidWriteOK(w, err) {
		w.WriteHeader(http.StatusNoContent)
	}
}

type testPushView struct {
	Device     string `json:"device"`
	Title      string `json:"title"`
	Path       string `json:"path"`
	Tag        string `json:"tag"`
	TTLSeconds int    `json:"ttl_seconds"`
	Urgency    string `json:"urgency"`
	Topic      string `json:"topic"`
}

// testPushes lists the recorder's sends in order, decoded from the plaintext payload; never an endpoint or key.
func (d Deps) testPushes(w http.ResponseWriter, _ *http.Request) {
	sends := d.PushRecorder.Sends()
	views := make([]testPushView, len(sends))
	for i, s := range sends {
		var p struct{ Title, Path, Tag string }
		if err := json.Unmarshal(s.Payload, &p); err != nil {
			respondInternal(w, err)
			return
		}
		views[i] = testPushView{s.Device, p.Title, p.Path, p.Tag, s.TTLSeconds, s.Urgency, s.Topic}
	}
	respondJSON(w, http.StatusOK, struct {
		Pushes []testPushView `json:"pushes"`
	}{views})
}
