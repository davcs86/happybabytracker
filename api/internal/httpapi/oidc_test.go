package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/auth"
	"github.com/davcs86/happybabytracker/api/internal/config"
	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
)

const testIssuer = "https://issuer.example"

// linkParent gives the parent an OIDC identity through an SSO redemption of a fresh invite.
func (k *kit) linkParent(parentID, sub string) {
	k.t.Helper()
	ctx := context.Background()
	_, inviteHash, err := auth.NewToken()
	_, sessionHash, err2 := auth.NewToken()
	if err = errors.Join(err, err2); err == nil {
		err = k.Store.IssueToken(ctx, model.TokenInvite, parentID, inviteHash, k.now.Add(constants.InviteTTL))
	}
	if err == nil {
		err = k.Store.RedeemInviteOIDC(ctx, inviteHash, testIssuer, sub, sessionHash, k.now, k.now.Add(time.Hour), "")
	}
	if err != nil {
		k.t.Fatal(err)
	}
}

// oidcAdminKit is a kit whose admin holds the OIDC identity (testIssuer, sub).
func oidcAdminKit(t *testing.T, sub string) *kit {
	t.Helper()
	k, db := fileKit(t)
	if _, err := db.Exec(`INSERT INTO admin_identities(admin_id, provider, subject) VALUES (?, ?, ?)`, k.adminID, testIssuer, sub); err != nil {
		t.Fatal(err)
	}
	return k
}

func (k *kit) oidcLogin(body any) *httptest.ResponseRecorder {
	k.t.Helper()
	return k.do(http.MethodPost, "/v1/session/oidc", "", body)
}

func (k *kit) redeemOIDC(token string, body any) *httptest.ResponseRecorder {
	k.t.Helper()
	return k.do(http.MethodPost, "/v1/tokens/"+token+"/redeem-oidc", "", body)
}

func oidcBody(sub string) map[string]string { return map[string]string{"iss": testIssuer, "sub": sub} }

func wantExpiry(t *testing.T, rec *httptest.ResponseRecorder, kind string, ttl time.Duration) string {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d %q, want 200", rec.Code, rec.Body.String())
	}
	m := bodyMap(t, rec)
	sameKeys(t, m, "token", "kind", "expires_at")
	if m["kind"] != kind || m["expires_at"] != stamp(t0.Add(ttl)) {
		t.Fatalf("got %v, want kind %s expiring %s", m, kind, stamp(t0.Add(ttl)))
	}
	return m["token"].(string)
}

func TestOIDCLoginModes(t *testing.T) {
	k := oidcAdminKit(t, "sub-admin")
	ana := k.addParent("ana")
	k.linkParent(ana.ID, "sub-ana")
	wantErr(t, k.oidcLogin(oidcBody("sub-ana")), http.StatusForbidden, "mode_refused")

	for _, m := range []config.AuthMode{config.AuthOIDC, config.AuthBoth} {
		k.setMode(m)
		parent := wantExpiry(t, k.oidcLogin(oidcBody("sub-ana")), "parent", 30*24*time.Hour)
		if rec := k.do(http.MethodGet, "/v1/session", parent, nil); rec.Code != http.StatusOK || bodyMap(t, rec)["parent"].(map[string]any)["id"] != ana.ID {
			t.Errorf("%s: parent session: %d %q", m, rec.Code, rec.Body.String())
		}
		admin := wantExpiry(t, k.oidcLogin(oidcBody("sub-admin")), "admin", 12*time.Hour)
		if rec := k.do(http.MethodGet, "/v1/session", admin, nil); rec.Code != http.StatusOK || bodyMap(t, rec)["kind"] != "admin" {
			t.Errorf("%s: admin session: %d %q", m, rec.Code, rec.Body.String())
		}
	}

	unknown := k.oidcLogin(oidcBody("sub-nobody"))
	wantErr(t, unknown, http.StatusForbidden, "not_linked")
	if err := k.Store.SetParentDisabled(context.Background(), ana.ID, true, k.now); err != nil {
		t.Fatal(err)
	}
	disabled := k.oidcLogin(oidcBody("sub-ana"))
	if disabled.Code != unknown.Code || disabled.Body.String() != unknown.Body.String() {
		t.Errorf("disabled parent %d %q differs from unknown pair %d %q", disabled.Code, disabled.Body.String(), unknown.Code, unknown.Body.String())
	}
}

func TestRedeemOIDCModes(t *testing.T) {
	k := newKit(t)
	_, invite := k.createParent(k.asAdmin(), "ana")
	wantErr(t, k.redeemOIDC(invite, oidcBody("sub-ana")), http.StatusForbidden, "mode_refused")
	if rec := k.redeem(invite, "invite password 1"); rec.Code != http.StatusOK {
		t.Fatalf("password redemption in local after the refusal: %d %q", rec.Code, rec.Body.String())
	}

	for _, m := range []config.AuthMode{config.AuthOIDC, config.AuthBoth} {
		k.setMode(m)
		_, invite := k.createParent(k.asAdmin(), "bo-"+string(m))
		token := wantExpiry(t, k.redeemOIDC(invite, oidcBody("sub-"+string(m))), "parent", 30*24*time.Hour)
		if rec := k.do(http.MethodGet, "/v1/session", token, nil); rec.Code != http.StatusOK {
			t.Errorf("%s: session: %d %q", m, rec.Code, rec.Body.String())
		}
		wantErr(t, k.redeemOIDC(invite, oidcBody("sub-again")), http.StatusNotFound, "not_found")
		_, other := k.createParent(k.asAdmin(), "cy-"+string(m))
		wantErr(t, k.redeemOIDC(other, oidcBody("sub-"+string(m))), http.StatusConflict, "already_linked")
		if rec := k.redeemOIDC(other, oidcBody("sub-cy-"+string(m))); rec.Code != http.StatusOK {
			t.Errorf("%s: the token after already_linked: %d %q", m, rec.Code, rec.Body.String())
		}
	}
	wantErr(t, k.redeemOIDC(strings.Repeat("x", 43), oidcBody("sub-z")), http.StatusNotFound, "not_found")
}

func TestRedeemOIDCEndsOtherSessions(t *testing.T) {
	k := newKit(t)
	k.setMode(config.AuthBoth)
	id, invite := k.createParent(k.asAdmin(), "ana")
	before := k.seedSession("parent", id, constants.ParentSessionTTL)
	if rec := k.do(http.MethodGet, "/v1/session", before, nil); rec.Code != http.StatusOK {
		t.Fatalf("seeded session: %d", rec.Code)
	}
	token := wantExpiry(t, k.redeemOIDC(invite, oidcBody("sub-ana")), "parent", constants.ParentSessionTTL)
	wantErr(t, k.do(http.MethodGet, "/v1/session", before, nil), http.StatusUnauthorized, "no_session")
	if rec := k.do(http.MethodGet, "/v1/session", token, nil); rec.Code != http.StatusOK {
		t.Errorf("returned token: %d %q", rec.Code, rec.Body.String())
	}
}

func TestOIDCValidation(t *testing.T) {
	k := newKit(t)
	k.setMode(config.AuthOIDC)
	_, invite := k.createParent(k.asAdmin(), "ana")
	long := "https://issuer.example/" + strings.Repeat("a", constants.OIDCIssuerMaxLen-len("https://issuer.example/")+1)
	cases := []struct {
		field string
		body  map[string]string
	}{
		{"iss", map[string]string{"iss": "", "sub": "s"}},
		{"iss", map[string]string{"sub": "s"}},
		{"iss", map[string]string{"iss": "http://issuer.example", "sub": "s"}},
		{"iss", map[string]string{"iss": long, "sub": "s"}},
		{"iss", map[string]string{"iss": "local", "sub": "s"}},
		{"iss", map[string]string{"iss": "https://", "sub": "s"}},
		{"sub", map[string]string{"iss": testIssuer, "sub": ""}},
		{"sub", map[string]string{"iss": testIssuer, "sub": strings.Repeat("s", constants.OIDCSubjectMaxLen+1)}},
		{"device_label", map[string]string{"iss": testIssuer, "sub": "s", "device_label": "bad!"}},
		{"device_label", map[string]string{"iss": testIssuer, "sub": "s", "device_label": strings.Repeat("a", 49)}},
	}
	for name, send := range map[string]func(any) *httptest.ResponseRecorder{
		"session/oidc": k.oidcLogin,
		"redeem-oidc":  func(b any) *httptest.ResponseRecorder { return k.redeemOIDC(invite, b) },
	} {
		for _, c := range cases {
			rec := send(c.body)
			if m := bodyMap(t, rec); rec.Code != http.StatusUnprocessableEntity || m["error"] != "invalid" || m["field"] != c.field {
				t.Errorf("%s %v: %d %q, want 422 invalid %s", name, c.body, rec.Code, rec.Body.String(), c.field)
			}
		}
	}
	edge := "https://issuer.example/" + strings.Repeat("a", constants.OIDCIssuerMaxLen-len("https://issuer.example/"))
	wantErr(t, k.oidcLogin(map[string]string{"iss": edge, "sub": strings.Repeat("s", constants.OIDCSubjectMaxLen)}), http.StatusForbidden, "not_linked")
	if rec := k.redeemOIDC(invite, map[string]string{"iss": edge, "sub": strings.Repeat("s", constants.OIDCSubjectMaxLen), "device_label": strings.Repeat("a", 48)}); rec.Code != http.StatusOK {
		t.Errorf("redeem at the limits: %d %q", rec.Code, rec.Body.String())
	}
}

func TestOIDCValidationOrder(t *testing.T) {
	k := newKit(t)
	_, invite := k.createParent(k.asAdmin(), "ana")
	bad := map[string]string{"iss": "http://x", "sub": "", "device_label": "bad!"}
	goodLabel := map[string]string{"iss": "http://x", "sub": "", "device_label": "Firefox on Mac"}
	goodIss := map[string]string{"iss": testIssuer, "sub": "", "device_label": "Firefox on Mac"}
	for name, send := range map[string]func(any) *httptest.ResponseRecorder{
		"session/oidc": k.oidcLogin,
		"redeem-oidc":  func(b any) *httptest.ResponseRecorder { return k.redeemOIDC(invite, b) },
	} {
		k.setMode(config.AuthLocal)
		wantErr(t, send(bad), http.StatusForbidden, "mode_refused")
		if rec := send(`{"x":1}`); rec.Code != http.StatusUnprocessableEntity || bodyMap(t, rec)["error"] != "unknown_field" {
			t.Errorf("%s: unknown field in local: %d %q, want 422 unknown_field", name, rec.Code, rec.Body.String())
		}
		k.setMode(config.AuthOIDC)
		for field, body := range map[string]map[string]string{"device_label": bad, "iss": goodLabel, "sub": goodIss} {
			rec := send(body)
			if m := bodyMap(t, rec); rec.Code != http.StatusUnprocessableEntity || m["field"] != field {
				t.Errorf("%s %v: %d %q, want 422 %s", name, body, rec.Code, rec.Body.String(), field)
			}
		}
	}
	k.setMode(config.AuthOIDC)
	if rec := k.redeemOIDC(invite, oidcBody("sub-ana")); rec.Code != http.StatusOK {
		t.Errorf("the token after the refusals: %d %q", rec.Code, rec.Body.String())
	}
}

func TestOIDCSessionsListTheDeviceLabel(t *testing.T) {
	k := newKit(t)
	k.setMode(config.AuthBoth)
	_, invite := k.createParent(k.asAdmin(), "ana")
	current := func(token string) any {
		t.Helper()
		for _, row := range bodyMap(t, k.do(http.MethodGet, "/v1/me/sessions", token, nil))["sessions"].([]any) {
			if m := row.(map[string]any); m["current"] == true {
				return m["device_label"]
			}
		}
		t.Fatal("no current session listed")
		return nil
	}
	body := map[string]string{"iss": testIssuer, "sub": "sub-ana", "device_label": "Firefox on Mac"}
	redeemed := wantExpiry(t, k.redeemOIDC(invite, body), "parent", constants.ParentSessionTTL)
	if got := current(redeemed); got != "Firefox on Mac" {
		t.Errorf("after redeem-oidc: device_label = %v", got)
	}
	labelled := wantExpiry(t, k.oidcLogin(body), "parent", constants.ParentSessionTTL)
	if got := current(labelled); got != "Firefox on Mac" {
		t.Errorf("after session/oidc: device_label = %v", got)
	}
	plain := wantExpiry(t, k.oidcLogin(oidcBody("sub-ana")), "parent", constants.ParentSessionTTL)
	if got := current(plain); got != nil {
		t.Errorf("without a label: device_label = %v, want null", got)
	}

	_, other := k.createParent(k.asAdmin(), "bob")
	plain = wantExpiry(t, k.redeemOIDC(other, oidcBody("sub-bob")), "parent", constants.ParentSessionTTL)
	if got := current(plain); got != nil {
		t.Errorf("redeem-oidc without a label: device_label = %v, want null", got)
	}
}

func TestAdminAfterSSOLoginIsRefusedOnKidRoutes(t *testing.T) {
	k := oidcAdminKit(t, "sub-admin")
	k.setMode(config.AuthOIDC)
	kid := k.createKid(k.asParent("ana"), nil)
	admin := wantExpiry(t, k.oidcLogin(oidcBody("sub-admin")), "admin", constants.AdminSessionTTL)
	for _, path := range []string{"/v1/kids", "/v1/kids/" + kid} {
		if rec := k.do(http.MethodGet, path, admin, nil); rec.Code != http.StatusForbidden {
			t.Errorf("GET %s as the SSO admin: %d %q, want 403", path, rec.Code, rec.Body.String())
		}
	}
}

func TestOIDCLogCarriesNoSubject(t *testing.T) {
	k := newKit(t)
	k.setMode(config.AuthBoth)
	const sub, label = "sub-unique-8841", "Unique Label Phone"
	_, invite := k.createParent(k.asAdmin(), "ana")
	k.Log.Reset()
	body := map[string]string{"iss": testIssuer, "sub": sub, "device_label": label}
	redeemed := wantExpiry(t, k.redeemOIDC(invite, body), "parent", constants.ParentSessionTTL)
	login := wantExpiry(t, k.oidcLogin(body), "parent", constants.ParentSessionTTL)
	k.redeemOIDC(invite, body)
	k.oidcLogin(map[string]string{"iss": testIssuer, "sub": "sub-unknown-5530", "device_label": label})
	k.oidcLogin(map[string]string{"iss": "http://bad.example", "sub": sub})
	wantCleanLog(t, k.Log.String(), 5, sub, "sub-unknown-5530", testIssuer, "issuer.example", label, invite, redeemed, login)
}

const (
	reauthPath      = "/v1/me/reauth"
	adminReauthPath = "/v1/admin/me/reauth"
	linkPath        = "/v1/me/identities"
	adminLinkPath   = "/v1/admin/me/identities"
)

func (k *kit) reauthFrom(path, addr, token, password string) *httptest.ResponseRecorder {
	k.t.Helper()
	r := k.newReq(http.MethodPost, path, map[string]string{"password": password})
	r.Header.Set("X-Client-Address", addr)
	r.Header.Set("X-Session-Token", token)
	return k.serve(r)
}

func (k *kit) link(path, token, sub string) *httptest.ResponseRecorder {
	k.t.Helper()
	return k.do(http.MethodPost, path, token, map[string]string{"iss": testIssuer, "sub": sub})
}

// countVerifications replaces verifyPassword for the test and returns the call counter.
func countVerifications(t *testing.T) *int {
	t.Helper()
	calls := 0
	orig := verifyPassword
	verifyPassword = func(hash, pw string) bool { calls++; return auth.VerifyPassword(hash, pw) }
	t.Cleanup(func() { verifyPassword = orig })
	return &calls
}

func wantNoContent(t *testing.T, rec *httptest.ResponseRecorder, what string) {
	t.Helper()
	if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Fatalf("%s: %d %q, want 204", what, rec.Code, rec.Body.String())
	}
}

func TestReauthModes(t *testing.T) {
	k, db := fileKit(t)
	parent, admin := k.asParent("ana"), k.asAdmin()
	calls := countVerifications(t)
	for _, m := range []config.AuthMode{config.AuthLocal, config.AuthOIDC} {
		k.setMode(m)
		for i := 0; i <= constants.LoginAttemptsPerMinute; i++ {
			wantErr(t, k.reauthFrom(reauthPath, "203.0.113.30", parent, testParentPass), http.StatusForbidden, "mode_refused")
			wantErr(t, k.reauthFrom(adminReauthPath, "203.0.113.30", admin, testAdminPass), http.StatusForbidden, "mode_refused")
		}
	}
	if *calls != 0 {
		t.Fatalf("%d password verifications while refused, want 0", *calls)
	}

	k.setMode(config.AuthBoth)
	k.now = t0.Add(90 * time.Second)
	wantNoContent(t, k.reauthFrom(reauthPath, "203.0.113.30", parent, testParentPass), "parent re-auth")
	wantNoContent(t, k.reauthFrom(adminReauthPath, "203.0.113.30", admin, testAdminPass), "admin re-auth")
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE reauth_at = ?`, k.now.Unix()).Scan(&n); err != nil || n != 2 {
		t.Errorf("sessions stamped at %d: %d, %v; want 2", k.now.Unix(), n, err)
	}
}

func TestReauthWrongPassword(t *testing.T) {
	k := newKit(t)
	k.setMode(config.AuthBoth)
	wantErr(t, k.reauthFrom(reauthPath, "203.0.113.31", k.asParent("ana"), "not the password"), http.StatusForbidden, "wrong_password")
	wantErr(t, k.reauthFrom(adminReauthPath, "203.0.113.31", k.asAdmin(), testParentPass), http.StatusForbidden, "wrong_password")

	id, _ := k.createParent(k.asAdmin(), "bob")
	bob := k.seedSession("parent", id, constants.ParentSessionTTL)
	calls := countVerifications(t)
	wantErr(t, k.reauthFrom(reauthPath, "203.0.113.32", bob, ""), http.StatusForbidden, "wrong_password")
	if *calls != 1 {
		t.Errorf("a parent without a password: %d verifications, want 1", *calls)
	}
	wantErr(t, k.link(linkPath, bob, "sub-bob"), http.StatusForbidden, "reauth_required")
}

func TestReauthCountsAgainstTheLoginLimiter(t *testing.T) {
	k := newKit(t)
	k.setMode(config.AuthBoth)
	ana := k.asParent("ana")
	for i := 1; i <= constants.LoginAttemptsPerMinute; i++ {
		k.login("203.0.113.40", fmt.Sprintf("filler%d", i), "x")
	}
	rec := k.reauthFrom(reauthPath, "203.0.113.40", ana, testParentPass)
	wantErr(t, rec, http.StatusTooManyRequests, "rate_limited")
	if rec.Header().Get("Retry-After") != "60" {
		t.Errorf("Retry-After = %q", rec.Header().Get("Retry-After"))
	}
	admin := k.asAdmin()
	wantErr(t, k.reauthFrom(adminReauthPath, "203.0.113.40", admin, testAdminPass), http.StatusTooManyRequests, "rate_limited")

	for i := 0; i < constants.LoginAttemptsPerMinute; i++ {
		wantErr(t, k.reauthFrom(reauthPath, fmt.Sprintf("203.0.113.%d", 50+i), ana, "wrong"), http.StatusForbidden, "wrong_password")
		wantErr(t, k.reauthFrom(adminReauthPath, fmt.Sprintf("203.0.113.%d", 60+i), admin, "wrong"), http.StatusForbidden, "wrong_password")
	}
	wantErr(t, k.login("203.0.113.70", "ana", testParentPass), http.StatusTooManyRequests, "rate_limited")
	wantErr(t, k.login("203.0.113.71", testAdminUser, testAdminPass), http.StatusTooManyRequests, "rate_limited")
	wantErr(t, k.reauthFrom(reauthPath, "203.0.113.72", ana, testParentPass), http.StatusTooManyRequests, "rate_limited")
}

// reauthedParent returns a session of the parent (created when missing) re-authenticated at k.now.
func (k *kit) reauthedParent(username string) string {
	k.t.Helper()
	tok := k.asParent(username)
	wantNoContent(k.t, k.reauthFrom(reauthPath, "203.0.113.80", tok, testParentPass), "re-auth "+username)
	return tok
}

func TestLinkWindow(t *testing.T) {
	k := newKit(t)
	k.setMode(config.AuthBoth)
	wantErr(t, k.link(linkPath, k.asParent("ana"), "sub-never"), http.StatusForbidden, "reauth_required")
	wantErr(t, k.link(adminLinkPath, k.asAdmin(), "sub-never"), http.StatusForbidden, "reauth_required")

	start := k.now
	edge := k.reauthedParent("ana")
	late := k.reauthedParent("bob")
	admin := k.asAdmin()
	wantNoContent(t, k.reauthFrom(adminReauthPath, "203.0.113.81", admin, testAdminPass), "admin re-auth")
	k.now = start.Add(constants.ReauthWindow)
	wantNoContent(t, k.link(linkPath, edge, "sub-ana"), "link at 5 minutes")
	wantNoContent(t, k.link(adminLinkPath, admin, "sub-admin"), "admin link at 5 minutes")
	k.now = start.Add(constants.ReauthWindow + time.Second)
	wantErr(t, k.link(linkPath, late, "sub-bob"), http.StatusForbidden, "reauth_required")
	wantErr(t, k.link(linkPath, edge, "sub-ana-2"), http.StatusForbidden, "reauth_required")
}

func TestLinkKeepsTheReauthWindow(t *testing.T) {
	k := newKit(t)
	k.setMode(config.AuthBoth)
	ana := k.reauthedParent("ana")
	wantNoContent(t, k.link(linkPath, ana, "sub-ana"), "link")
	r := k.newReq(http.MethodPost, linkPath, map[string]string{"iss": "https://other.example", "sub": "sub-ana"})
	r.Header.Set("X-Session-Token", ana)
	wantNoContent(t, k.serve(r), "a second issuer in the same window")
}

func TestLinkAlreadyLinked(t *testing.T) {
	k := oidcAdminKit(t, "sub-admin")
	k.setMode(config.AuthBoth)
	held := k.addParent("held")
	k.linkParent(held.ID, "sub-held")
	ana := k.reauthedParent("ana")
	wantErr(t, k.link(linkPath, ana, "sub-held"), http.StatusConflict, "already_linked")
	wantErr(t, k.link(linkPath, ana, "sub-admin"), http.StatusConflict, "already_linked")
	wantNoContent(t, k.link(linkPath, ana, "sub-ana"), "first link")
	wantErr(t, k.link(linkPath, ana, "sub-ana-2"), http.StatusConflict, "already_linked")

	admin := k.asAdmin()
	wantNoContent(t, k.reauthFrom(adminReauthPath, "203.0.113.82", admin, testAdminPass), "admin re-auth")
	wantErr(t, k.link(adminLinkPath, admin, "sub-held"), http.StatusConflict, "already_linked")
	wantErr(t, k.link(adminLinkPath, admin, "sub-admin-2"), http.StatusConflict, "already_linked")
}

func TestLinkIsPerSession(t *testing.T) {
	k := newKit(t)
	k.setMode(config.AuthBoth)
	a := k.reauthedParent("ana")
	b := k.asParent("ana")
	wantErr(t, k.link(linkPath, b, "sub-ana"), http.StatusForbidden, "reauth_required")
	wantNoContent(t, k.link(linkPath, a, "sub-ana"), "the re-authenticated session")
}

func TestLinkModesAndValidation(t *testing.T) {
	k := newKit(t)
	k.setMode(config.AuthBoth)
	ana := k.reauthedParent("ana")
	for _, m := range []config.AuthMode{config.AuthLocal, config.AuthOIDC} {
		k.setMode(m)
		wantErr(t, k.link(linkPath, ana, "sub-ana"), http.StatusForbidden, "mode_refused")
		wantErr(t, k.do(http.MethodPost, linkPath, ana, map[string]string{"iss": "http://x", "sub": ""}), http.StatusForbidden, "mode_refused")
	}
	k.setMode(config.AuthBoth)
	never := k.asParent("bob")
	wantErr(t, k.do(http.MethodPost, linkPath, never, map[string]string{"iss": "http://x", "sub": ""}), http.StatusForbidden, "reauth_required")
	for field, body := range map[string]map[string]string{
		"iss": {"iss": "local", "sub": "s"},
		"sub": {"iss": testIssuer, "sub": strings.Repeat("s", constants.OIDCSubjectMaxLen+1)},
	} {
		rec := k.do(http.MethodPost, linkPath, ana, body)
		if m := bodyMap(t, rec); rec.Code != http.StatusUnprocessableEntity || m["error"] != "invalid" || m["field"] != field {
			t.Errorf("%v: %d %q, want 422 invalid %s", body, rec.Code, rec.Body.String(), field)
		}
	}
	rec := k.do(http.MethodPost, linkPath, ana, map[string]string{"iss": testIssuer, "sub": "s", "device_label": "Phone"})
	if rec.Code != http.StatusUnprocessableEntity || bodyMap(t, rec)["error"] != "unknown_field" {
		t.Errorf("device_label on a link: %d %q, want 422 unknown_field", rec.Code, rec.Body.String())
	}
}

func TestReauthAndLinkDecodeBeforeTheMode(t *testing.T) {
	k := newKit(t)
	parent, admin := k.asParent("ana"), k.asAdmin()
	for _, m := range []config.AuthMode{config.AuthLocal, config.AuthOIDC} {
		k.setMode(m)
		for path, tok := range map[string]string{reauthPath: parent, adminReauthPath: admin, linkPath: parent, adminLinkPath: admin} {
			rec := k.do(http.MethodPost, path, tok, matrixBody)
			if rec.Code != http.StatusUnprocessableEntity || bodyMap(t, rec)["error"] != "unknown_field" {
				t.Errorf("%s %s: %d %q, want 422 unknown_field", m, path, rec.Code, rec.Body.String())
			}
		}
	}
}

func TestAdminLinkUsesAdminIdentities(t *testing.T) {
	k, db := fileKit(t)
	k.setMode(config.AuthBoth)
	admin := k.asAdmin()
	wantNoContent(t, k.reauthFrom(adminReauthPath, "203.0.113.83", admin, testAdminPass), "admin re-auth")
	wantNoContent(t, k.link(adminLinkPath, admin, "sub-admin"), "admin link")
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM admin_identities WHERE admin_id = ? AND provider = ? AND subject = 'sub-admin'`, k.adminID, testIssuer).Scan(&n); err != nil || n != 1 {
		t.Fatalf("admin_identities rows = %d, %v", n, err)
	}
	wantExpiry(t, k.oidcLogin(oidcBody("sub-admin")), "admin", constants.AdminSessionTTL)
}
