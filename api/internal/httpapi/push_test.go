package httpapi

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync"
	"testing"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/push"
)

const testEndpoint = "https://push.example.com/send/abc123secretcapability"

var testKeyPair = sync.OnceValue(func() [2]string {
	k, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	auth := make([]byte, 16)
	if _, err := rand.Read(auth); err != nil {
		panic(err)
	}
	return [2]string{base64.RawURLEncoding.EncodeToString(k.PublicKey().Bytes()), base64.RawURLEncoding.EncodeToString(auth)}
})

// testSubscriptionKeys is a real P-256 public key and a 16-byte auth secret, canonical unpadded base64url.
func testSubscriptionKeys() (p256dh, auth string) {
	k := testKeyPair()
	return k[0], k[1]
}

func deviceBody(endpoint, label string) map[string]any {
	p, a := testSubscriptionKeys()
	return map[string]any{"endpoint": endpoint, "keys": map[string]any{"p256dh": p, "auth": a}, "label": label}
}

// fakeResolver answers from a map; a missing name is a lookup error. It never touches the network.
type fakeResolver map[string][]string

func (f fakeResolver) LookupNetIP(_ context.Context, _, host string) ([]netip.Addr, error) {
	ss, found := f[host]
	if !found {
		return nil, errors.New("no such host")
	}
	out := make([]netip.Addr, len(ss))
	for i, s := range ss {
		out[i] = netip.MustParseAddr(s)
	}
	return out, nil
}

func (k *kit) putDevice(token string, body any) *httptest.ResponseRecorder {
	k.t.Helper()
	return k.do(http.MethodPut, "/v1/me/push-devices", token, body)
}

func (k *kit) mustDevice(token, endpoint string) string {
	k.t.Helper()
	rec := k.putDevice(token, deviceBody(endpoint, "Phone"))
	if rec.Code != http.StatusCreated {
		k.t.Fatalf("register: %d %q", rec.Code, rec.Body.String())
	}
	return bodyMap(k.t, rec)["device"].(map[string]any)["id"].(string)
}

func (k *kit) deviceList(token string) []any {
	k.t.Helper()
	rec := k.do(http.MethodGet, "/v1/me/push-devices", token, nil)
	if rec.Code != http.StatusOK {
		k.t.Fatalf("list: %d %q", rec.Code, rec.Body.String())
	}
	return bodyMap(k.t, rec)["devices"].([]any)
}

func optinPath(kid, device string) string { return "/v1/kids/" + kid + "/push-devices/" + device }

func noSecrets(t *testing.T, what string, rec *httptest.ResponseRecorder) {
	t.Helper()
	p, a := testSubscriptionKeys()
	for _, s := range []string{testEndpoint, "push.example.com", p, a} {
		if strings.Contains(rec.Body.String(), s) {
			t.Errorf("%s: response carries %q: %s", what, s, rec.Body.String())
		}
	}
}

func TestPushState(t *testing.T) {
	k := newKit(t)
	rec := k.do(http.MethodGet, "/v1/push", k.asParent("ana"), nil)
	m := bodyMap(t, rec)
	sameKeys(t, m, "enabled", "public_key")
	if rec.Code != http.StatusOK || m["enabled"] != true || m["public_key"] != push.PublicKey(testVAPID()) {
		t.Fatalf("on: %d %q", rec.Code, rec.Body.String())
	}
	off := newKitNoPush(t)
	rec = off.do(http.MethodGet, "/v1/push", off.asParent("ana"), nil)
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != `{"enabled":false}` {
		t.Fatalf("off: %d %q", rec.Code, rec.Body.String())
	}
}

func TestUpsertDevice(t *testing.T) {
	k, db := fileKit(t)
	ana := k.asParent("ana")
	k.Log.Reset()
	rec := k.putDevice(ana, deviceBody(testEndpoint, "  Phone  "))
	noSecrets(t, "create", rec)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %q", rec.Code, rec.Body.String())
	}
	dev := bodyMap(t, rec)["device"].(map[string]any)
	sameKeys(t, dev, "id", "label", "created_at", "kids")
	if dev["label"] != "Phone" || dev["created_at"] != ts(k.now) || len(dev["kids"].([]any)) != 0 {
		t.Fatalf("device = %v", dev)
	}

	p, a := testSubscriptionKeys()
	raw, _ := base64.RawURLEncoding.DecodeString(p)
	rawAuth, _ := base64.RawURLEncoding.DecodeString(a)
	padded := map[string]any{"endpoint": testEndpoint, "label": "Tablet",
		"keys": map[string]any{"p256dh": base64.StdEncoding.EncodeToString(raw), "auth": base64.StdEncoding.EncodeToString(rawAuth)}}
	rec = k.putDevice(ana, padded)
	noSecrets(t, "re-send", rec)
	again := bodyMap(t, rec)["device"].(map[string]any)
	if rec.Code != http.StatusOK || again["id"] != dev["id"] || again["label"] != "Tablet" {
		t.Fatalf("re-send: %d %q", rec.Code, rec.Body.String())
	}
	var gotP, gotA string
	if err := db.QueryRow(`SELECT p256dh, auth FROM push_subscriptions WHERE id = ?`, dev["id"]).Scan(&gotP, &gotA); err != nil {
		t.Fatal(err)
	}
	if gotP != p || gotA != a {
		t.Fatalf("stored keys %q %q are not canonical unpadded base64url", gotP, gotA)
	}
	list := k.do(http.MethodGet, "/v1/me/push-devices", ana, nil)
	noSecrets(t, "list", list)
	rename := k.do(http.MethodPatch, "/v1/me/push-devices/"+dev["id"].(string), ana, map[string]any{"label": "Laptop"})
	noSecrets(t, "rename", rename)
	if rename.Code != http.StatusOK || bodyMap(t, rename)["device"].(map[string]any)["label"] != "Laptop" {
		t.Fatalf("rename: %d %q", rename.Code, rename.Body.String())
	}
	wantCleanLog(t, k.Log.String(), 4, testEndpoint, "push.example.com", p, a, gotP, base64.StdEncoding.EncodeToString(raw))
}

func TestDeviceMovesToTheParentWhoRegistersIt(t *testing.T) {
	k := newKit(t)
	kid, ana, bob := k.trackerKid()
	old := k.mustDevice(ana, testEndpoint)
	if rec := k.do(http.MethodPut, optinPath(kid, old), ana, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("opt in: %d %q", rec.Code, rec.Body.String())
	}
	fresh := k.putDevice(bob, deviceBody("https://push.example.com/fresh", "Bob's"))
	rec := k.putDevice(bob, deviceBody(testEndpoint, "Bob's"))
	moved := bodyMap(t, rec)["device"].(map[string]any)
	freshID := bodyMap(t, fresh)["device"].(map[string]any)["id"].(string)
	if rec.Code != http.StatusCreated || fresh.Code != http.StatusCreated || moved["id"] == old {
		t.Fatalf("move: %d %q (old id %s)", rec.Code, rec.Body.String(), old)
	}
	if got, want := strings.ReplaceAll(rec.Body.String(), moved["id"].(string), "ID"), strings.ReplaceAll(fresh.Body.String(), freshID, "ID"); got != want {
		t.Fatalf("a move answers %s, a fresh endpoint %s", got, want)
	}
	if list := k.deviceList(ana); len(list) != 0 {
		t.Fatalf("old owner still lists %v", list)
	}
	if list := k.deviceList(bob); len(list) != 2 {
		t.Fatalf("new owner lists %v", list)
	}
	unknownKid := k.do(http.MethodPut, optinPath(model.NewID(), old), ana, nil)
	sameResponse(t, "old owner's opt-in with the old id", k.do(http.MethodPut, optinPath(kid, old), ana, nil), unknownKid)
	sameResponse(t, "old owner's opt-out with the old id", k.do(http.MethodDelete, optinPath(kid, old), ana, nil), unknownKid)
	sameResponse(t, "new owner's opt-in with the old id", k.do(http.MethodPut, optinPath(kid, old), bob, nil), unknownKid)
	wantErr(t, k.do(http.MethodDelete, "/v1/me/push-devices/"+old, ana, nil), http.StatusNotFound, "not_found")
}

func TestEndpointCanonicalised(t *testing.T) {
	for in, want := range map[string]string{
		"https://H.Example.org.:443/x?A=B%2F": "https://h.example.org/x?A=B%2F",
		"https://h.example.org/X/Y?q=Z#f":     "https://h.example.org/X/Y?q=Z#f",
		"https://PUSH.example.org:443":        "https://push.example.org",
		"https://push.example.org.?a":         "https://push.example.org?a",
	} {
		if got := canonicalEndpoint(in); got != want {
			t.Errorf("%s: %s, want %s", in, got, want)
		}
	}
	k, db := fileKit(t)
	ana, ben := k.asParent("ana"), k.asParent("ben")
	first := k.putDevice(ana, deviceBody("https://H.Example.org.:443/x?A=B%2F", "x"))
	id := bodyMap(t, first)["device"].(map[string]any)["id"]
	again := k.putDevice(ana, deviceBody("https://h.example.org/x?A=B%2F", "y"))
	if first.Code != http.StatusCreated || again.Code != http.StatusOK || bodyMap(t, again)["device"].(map[string]any)["id"] != id {
		t.Fatalf("same binding: %d %q / %d %q", first.Code, first.Body.String(), again.Code, again.Body.String())
	}
	var stored string
	if err := db.QueryRow(`SELECT endpoint FROM push_subscriptions WHERE id = ?`, id).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != "https://h.example.org/x?A=B%2F" {
		t.Fatalf("stored %q", stored)
	}
	if rec := k.putDevice(ana, deviceBody("https://h.example.org/X?A=B%2F", "z")); rec.Code != http.StatusCreated {
		t.Fatalf("the path is case-sensitive: %d %q", rec.Code, rec.Body.String())
	}
	if rec := k.putDevice(ben, deviceBody("https://h.EXAMPLE.org.:443/x?A=B%2F", "b")); rec.Code != http.StatusCreated {
		t.Fatalf("move by a variant: %d %q", rec.Code, rec.Body.String())
	}
	if list := k.deviceList(ana); len(list) != 1 {
		t.Fatalf("a variant made a second binding: %v", list)
	}
	wantInvalid2(t, "empty port", k.putDevice(ana, deviceBody("https://h.example.org:/x", "x")), "endpoint")
	wantInvalid2(t, "port 8443", k.putDevice(ana, deviceBody("https://h.example.org:8443/x", "x")), "endpoint")
}

func TestNewHandlerRefusesPushWithoutAResolver(t *testing.T) {
	k := newKit(t)
	d := k.deps()
	d.Cfg.AllowFakePush = false
	func() {
		defer func() {
			if r := recover(); r != "httpapi: Web Push needs a resolver" {
				t.Fatalf("recovered %v", r)
			}
		}()
		NewHandler(d)
	}()
	d.Resolver = fakeResolver{}
	NewHandler(d)
	d.Resolver, d.Cfg.AllowFakePush = nil, true
	NewHandler(d)
	d.Push, d.Cfg.AllowFakePush = nil, false
	NewHandler(d)
}

func TestUpsertWithoutAResolverIsAClean500(t *testing.T) {
	k := newKit(t)
	k.Cfg.AllowFakePush = false
	d := k.deps()
	k.Handler = newMux(d, Routes(d))
	wantErr(t, k.putDevice(k.asParent("ana"), deviceBody(testEndpoint, "x")), http.StatusInternalServerError, "internal")
}

func TestPushStateNeedsThePublicKey(t *testing.T) {
	k := newKit(t)
	k.Cfg.VAPIDKey = nil
	k.Handler = NewHandler(k.deps())
	ana := k.asParent("ana")
	rec := k.do(http.MethodGet, "/v1/push", ana, nil)
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != `{"enabled":false}` {
		t.Fatalf("sender without a key: %d %q", rec.Code, rec.Body.String())
	}
	wantErr(t, k.putDevice(ana, deviceBody(testEndpoint, "x")), http.StatusConflict, "push_unavailable")
}

func TestDeviceCap(t *testing.T) {
	k := newKit(t)
	ana := k.asParent("ana")
	for i := range constants.PushDevicesPerParent {
		k.mustDevice(ana, fmt.Sprintf("https://push.example.com/%d", i))
	}
	wantErr(t, k.putDevice(ana, deviceBody("https://push.example.com/eleventh", "x")), http.StatusConflict, "limit")
	if rec := k.putDevice(ana, deviceBody("https://push.example.com/0", "y")); rec.Code != http.StatusOK {
		t.Fatalf("re-send at the cap: %d %q", rec.Code, rec.Body.String())
	}
}

func TestDeviceHostileEndpoints(t *testing.T) {
	k := newKit(t)
	k.Cfg.AllowFakePush = false
	k.Resolver = fakeResolver{
		"push.example.org": {"8.8.8.8"},
		"private.test":     {"10.0.0.1"},
		"loopback.test":    {"127.0.0.1"},
		"linklocal.test":   {"169.254.1.1"},
		"ula.test":         {"fd00::1"},
		"cgnat.test":       {"100.64.0.1"},
		"mixed.test":       {"8.8.8.8", "192.168.1.1"},
		"localhost":        {"8.8.8.8"},
	}
	k.Handler = NewHandler(k.deps())
	ana := k.asParent("ana")
	for name, endpoint := range map[string]string{
		"http":             "http://push.example.org/x",
		"userinfo":         "https://u:p@push.example.org/x",
		"port 8443":        "https://push.example.org:8443/x",
		"ip literal":       "https://8.8.8.8/x",
		"ipv6 literal":     "https://[2001:4860:4860::8888]/x",
		"localhost":        "https://localhost/x",
		"private":          "https://private.test/x",
		"loopback":         "https://loopback.test/x",
		"link-local":       "https://linklocal.test/x",
		"ula":              "https://ula.test/x",
		"cgnat":            "https://cgnat.test/x",
		"mixed answer":     "https://mixed.test/x",
		"no such host":     "https://nowhere.test/x",
		"over 2048 bytes":  "https://push.example.org/" + strings.Repeat("a", constants.PushEndpointMaxLen),
		"empty":            "",
		"not a url at all": "push.example.org",
	} {
		rec := k.putDevice(ana, deviceBody(endpoint, "x"))
		if rec.Code != http.StatusUnprocessableEntity || strings.TrimSpace(rec.Body.String()) != `{"error":"invalid","field":"endpoint"}` {
			t.Errorf("%s: %d %q", name, rec.Code, rec.Body.String())
		}
	}
	body := deviceBody("", "x")
	delete(body, "endpoint")
	wantInvalid2(t, "missing", k.putDevice(ana, body), "endpoint")
	if rec := k.putDevice(ana, deviceBody("https://push.example.org/ok", "x")); rec.Code != http.StatusCreated {
		t.Fatalf("public endpoint: %d %q", rec.Code, rec.Body.String())
	}
	if rec := k.putDevice(ana, deviceBody("https://push.example.org:443/ok", "x")); rec.Code != http.StatusOK {
		t.Fatalf("port 443 is the same binding: %d %q", rec.Code, rec.Body.String())
	}
}

func TestDeviceFakeModeSkipsDNSButKeepsSyntax(t *testing.T) {
	k := newKit(t)
	ana := k.asParent("ana")
	if rec := k.putDevice(ana, deviceBody("https://unresolvable.invalid/x", "x")); rec.Code != http.StatusCreated {
		t.Fatalf("fake mode resolved: %d %q", rec.Code, rec.Body.String())
	}
	wantInvalid2(t, "http", k.putDevice(ana, deviceBody("http://push.example.org/x", "x")), "endpoint")
	wantInvalid2(t, "ip", k.putDevice(ana, deviceBody("https://127.0.0.1/x", "x")), "endpoint")
}

func TestDeviceBadKeysAndLabel(t *testing.T) {
	k := newKit(t)
	ana := k.asParent("ana")
	p, a := testSubscriptionKeys()
	raw, _ := base64.RawURLEncoding.DecodeString(p)
	compressed := append([]byte{2 + raw[64]&1}, raw[1:33]...)
	invalid := append([]byte{4}, make([]byte, 64)...)
	for name, keys := range map[string]any{
		"compressed p256dh": map[string]any{"p256dh": base64.RawURLEncoding.EncodeToString(compressed), "auth": a},
		"invalid point":     map[string]any{"p256dh": base64.RawURLEncoding.EncodeToString(invalid), "auth": a},
		"not base64":        map[string]any{"p256dh": "!!!", "auth": a},
		"auth 15 bytes":     map[string]any{"p256dh": p, "auth": base64.RawURLEncoding.EncodeToString(make([]byte, 15))},
		"auth 17 bytes":     map[string]any{"p256dh": p, "auth": base64.RawURLEncoding.EncodeToString(make([]byte, 17))},
		"no auth":           map[string]any{"p256dh": p},
		"no p256dh":         map[string]any{"auth": a},
		"null":              nil,
	} {
		body := map[string]any{"endpoint": testEndpoint, "keys": keys, "label": "x"}
		rec := k.putDevice(ana, body)
		wantInvalid2(t, name, rec, "keys")
		noSecrets(t, name, rec)
	}
	for name, label := range map[string]any{
		"41": strings.Repeat("é", constants.PushLabelMaxLen+1), "empty": "", "blank": "   ", "missing": nil,
	} {
		body := deviceBody(testEndpoint, "")
		if label == nil {
			delete(body, "label")
		} else {
			body["label"] = label
		}
		wantInvalid2(t, name, k.putDevice(ana, body), "label")
	}
	if rec := k.putDevice(ana, deviceBody(testEndpoint, strings.Repeat("é", constants.PushLabelMaxLen))); rec.Code != http.StatusCreated {
		t.Fatalf("40 characters: %d %q", rec.Code, rec.Body.String())
	}
	id := bodyMap(t, k.putDevice(ana, deviceBody(testEndpoint, "x")))["device"].(map[string]any)["id"].(string)
	wantInvalid2(t, "rename 41", k.do(http.MethodPatch, "/v1/me/push-devices/"+id, ana,
		map[string]any{"label": strings.Repeat("a", constants.PushLabelMaxLen+1)}), "label")
	wantInvalid2(t, "rename missing", k.do(http.MethodPatch, "/v1/me/push-devices/"+id, ana, map[string]any{}), "label")
}

func TestOptins(t *testing.T) {
	k := newKit(t)
	kid, ana, bob := k.trackerKid()
	cid := k.asParent("cid")
	other := k.createKid(cid, nil)
	id := k.mustDevice(ana, testEndpoint)
	bobDev := k.mustDevice(bob, "https://push.example.com/bob")

	unknownKid := k.do(http.MethodPut, optinPath("0190a1b2-0000-7000-8000-000000000000", id), ana, nil)
	for name, rec := range map[string]*httptest.ResponseRecorder{
		"kid without a grant":         k.do(http.MethodPut, optinPath(other, id), ana, nil),
		"clear kid without a grant":   k.do(http.MethodDelete, optinPath(other, id), ana, nil),
		"another parent's device":     k.do(http.MethodPut, optinPath(kid, bobDev), ana, nil),
		"clear another parent's":      k.do(http.MethodDelete, optinPath(kid, bobDev), ana, nil),
		"missing device":              k.do(http.MethodPut, optinPath(kid, "0190a1b2-0000-7000-8000-000000000001"), ana, nil),
		"clear a missing device":      k.do(http.MethodDelete, optinPath(kid, "nope"), ana, nil),
		"unknown kid, clear":          k.do(http.MethodDelete, optinPath("0190a1b2-0000-7000-8000-000000000000", id), ana, nil),
		"unknown kid (the reference)": unknownKid,
	} {
		sameResponse(t, name, rec, unknownKid)
	}
	wantErr(t, unknownKid, http.StatusNotFound, "not_found")

	if rec := k.do(http.MethodPut, optinPath(kid, id), ana, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("opt in: %d %q", rec.Code, rec.Body.String())
	}
	if rec := k.do(http.MethodPut, optinPath(kid, bobDev), bob, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("caregiver opt in: %d %q", rec.Code, rec.Body.String())
	}
	kids := k.deviceList(ana)[0].(map[string]any)["kids"].([]any)
	if len(kids) != 1 || kids[0] != kid {
		t.Fatalf("kids = %v", kids)
	}
	for range 2 {
		if rec := k.do(http.MethodDelete, optinPath(kid, id), ana, nil); rec.Code != http.StatusNoContent {
			t.Fatalf("opt out: %d %q", rec.Code, rec.Body.String())
		}
	}
	if kids := k.deviceList(ana)[0].(map[string]any)["kids"].([]any); len(kids) != 0 {
		t.Fatalf("after opt out kids = %v", kids)
	}
}

func optedKids(t *testing.T, k *kit, token string) []any {
	t.Helper()
	return k.deviceList(token)[0].(map[string]any)["kids"].([]any)
}

func TestRevokeRemovesOptinsHTTP(t *testing.T) {
	k := newKit(t)
	kid, ana, bob := k.trackerKid()
	dev := k.mustDevice(bob, testEndpoint)
	if rec := k.do(http.MethodPut, optinPath(kid, dev), bob, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("opt in: %d", rec.Code)
	}
	if rec := k.do(http.MethodDelete, "/v1/kids/"+kid+"/access/"+k.parentID("bob"), ana, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("revoke: %d %q", rec.Code, rec.Body.String())
	}
	if kids := optedKids(t, k, bob); len(kids) != 0 {
		t.Fatalf("after revoke kids = %v", kids)
	}
	k.grant(ana, kid, "bob", "caregiver")
	if kids := optedKids(t, k, bob); len(kids) != 0 {
		t.Fatalf("a re-grant restored the opt-in: %v", kids)
	}
}

func TestLeaveRemovesOptinsHTTP(t *testing.T) {
	k := newKit(t)
	kid, _, bob := k.trackerKid()
	dev := k.mustDevice(bob, testEndpoint)
	if rec := k.do(http.MethodPut, optinPath(kid, dev), bob, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("opt in: %d", rec.Code)
	}
	if rec := k.do(http.MethodDelete, "/v1/kids/"+kid+"/access/"+k.parentID("bob"), bob, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("leave: %d %q", rec.Code, rec.Body.String())
	}
	if kids := optedKids(t, k, bob); len(kids) != 0 {
		t.Fatalf("after leave kids = %v", kids)
	}
}

func TestDisableRemovesDevicesHTTP(t *testing.T) {
	k := newKit(t)
	kid, _, bob := k.trackerKid()
	dev := k.mustDevice(bob, testEndpoint)
	if rec := k.do(http.MethodPut, optinPath(kid, dev), bob, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("opt in: %d", rec.Code)
	}
	if rec := k.disable(k.asAdmin(), "bob"); rec.Code != http.StatusNoContent {
		t.Fatalf("disable: %d %q", rec.Code, rec.Body.String())
	}
	if list, err := k.Store.Devices(context.Background(), k.parentID("bob")); err != nil || len(list) != 0 {
		t.Fatalf("devices after disable: %v %v", list, err)
	}
}

func TestDeviceOfAnotherParentIs404(t *testing.T) {
	k := newKit(t)
	ana, ben := k.asParent("ana"), k.asParent("ben")
	theirs := k.mustDevice(ben, testEndpoint)
	missing := k.do(http.MethodDelete, "/v1/me/push-devices/"+"0190a1b2-0000-7000-8000-000000000000", ana, nil)
	wantErr(t, missing, http.StatusNotFound, "not_found")
	sameResponse(t, "delete theirs", k.do(http.MethodDelete, "/v1/me/push-devices/"+theirs, ana, nil), missing)
	sameResponse(t, "rename theirs", k.do(http.MethodPatch, "/v1/me/push-devices/"+theirs, ana, map[string]any{"label": "mine"}), missing)
	sameResponse(t, "rename missing", k.do(http.MethodPatch, "/v1/me/push-devices/nope", ana, map[string]any{"label": "mine"}), missing)
	if list := k.deviceList(ben); len(list) != 1 || list[0].(map[string]any)["label"] != "Phone" {
		t.Fatalf("their device changed: %v", list)
	}
}

func TestPushRoutesRefuseTheAdmin(t *testing.T) {
	k := newKit(t)
	admin := k.asAdmin()
	for _, rt := range []struct{ method, path string }{
		{http.MethodGet, "/v1/push"}, {http.MethodGet, "/v1/me/push-devices"}, {http.MethodPut, "/v1/me/push-devices"},
		{http.MethodDelete, "/v1/me/push-devices/x"},
	} {
		wantErr(t, k.do(rt.method, rt.path, admin, deviceBody(testEndpoint, "x")), http.StatusForbidden, "forbidden")
	}
}

func TestPushOff(t *testing.T) {
	k := newKitNoPush(t)
	kid, ana, _ := k.trackerKid()
	p, a := testSubscriptionKeys()
	dev, _, err := k.Store.UpsertDevice(context.Background(), k.parentID("ana"), testEndpoint, p, a, "Old", k.now)
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := k.Store.UpsertDevice(context.Background(), k.parentID("ana"), "https://push.example.com/2", p, a, "Old 2", k.now)
	if err != nil {
		t.Fatal(err)
	}
	if err := k.Store.SetPushOptin(context.Background(), kid, k.parentID("ana"), dev.ID); err != nil {
		t.Fatal(err)
	}
	wantErr(t, k.putDevice(ana, deviceBody(testEndpoint, "x")), http.StatusConflict, "push_unavailable")
	wantErr(t, k.do(http.MethodPatch, "/v1/me/push-devices/"+dev.ID, ana, map[string]any{"label": "y"}), http.StatusConflict, "push_unavailable")
	wantErr(t, k.do(http.MethodPut, optinPath(kid, second.ID), ana, nil), http.StatusConflict, "push_unavailable")
	if list := k.deviceList(ana); len(list) != 2 {
		t.Fatalf("list while off: %v", list)
	}
	if rec := k.do(http.MethodDelete, optinPath(kid, dev.ID), ana, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("opt out while off: %d %q", rec.Code, rec.Body.String())
	}
	if rec := k.do(http.MethodDelete, "/v1/me/push-devices/"+dev.ID, ana, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("delete while off: %d %q", rec.Code, rec.Body.String())
	}
	if list := k.deviceList(ana); len(list) != 1 || list[0].(map[string]any)["id"] != second.ID {
		t.Fatalf("after delete: %v", list)
	}
	wantErr(t, k.do(http.MethodGet, "/v1/_test/pushes", "", nil), http.StatusNotFound, "not_found")
	for _, rt := range Routes(k.deps()) {
		if rt.Pattern == "/v1/_test/pushes" {
			t.Fatal("the test route is declared while the fake is off")
		}
	}
}

func TestTestPushesReportsTheRecorder(t *testing.T) {
	k := newKit(t)
	p, a := testSubscriptionKeys()
	payload := []byte(`{"v":1,"category":"medication","kid":"k1","title":"Medication due for Mia","path":"/kids/k1/medication","tag":"reminder-r1"}`)
	res := k.Recorder.Send(context.Background(), push.Subscription{ID: "dev-1", Endpoint: testEndpoint, P256dh: p, Auth: a},
		push.Message{Payload: payload, TTL: constants.PushTTLMax, Topic: "r1"})
	if res.Outcome != push.Sent {
		t.Fatalf("recorder: %+v", res)
	}
	rec := k.do(http.MethodGet, "/v1/_test/pushes", "", nil)
	want := `{"pushes":[{"device":"dev-1","title":"Medication due for Mia","path":"/kids/k1/medication","tag":"reminder-r1","ttl_seconds":3600,"urgency":"high","topic":"r1"}]}`
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != want {
		t.Fatalf("got %d %s\nwant %s", rec.Code, rec.Body.String(), want)
	}
	noSecrets(t, "test pushes", rec)
	r := k.newReq(http.MethodGet, "/v1/_test/pushes", nil)
	r.Header.Del("Authorization")
	wantErr(t, k.serve(r), http.StatusUnauthorized, "unauthorized")
}
