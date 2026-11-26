package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/auth"
	"github.com/davcs86/happybabytracker/api/internal/constants"
)

const invalidCreds = `{"error":"invalid_credentials"}`

func stamp(t time.Time) string { return t.UTC().Format(time.RFC3339) }

func TestInternalTokenRequired(t *testing.T) {
	k := newKit(t)
	tok := k.asParent("ana")
	for _, hdr := range []string{"", "Bearer nope", testInternalToken, "Bearer "} {
		r := k.newReq(http.MethodPost, "/v1/session", map[string]string{"username": "ana", "password": testParentPass})
		r.Header.Del("Authorization")
		if hdr != "" {
			r.Header.Set("Authorization", hdr)
		}
		wantErr(t, k.serve(r), http.StatusUnauthorized, "unauthorized")

		// A valid session token never stands in for the internal token.
		r = k.newReq(http.MethodGet, "/v1/session", nil)
		r.Header.Del("Authorization")
		if hdr != "" {
			r.Header.Set("Authorization", hdr)
		}
		r.Header.Set("X-Session-Token", tok)
		wantErr(t, k.serve(r), http.StatusUnauthorized, "unauthorized")
	}
}

func TestOpenRouteNeedsNoToken(t *testing.T) {
	k := newKit(t)
	rec := k.serve(httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("/healthz without token: %d %q", rec.Code, rec.Body.String())
	}
}

func TestAdminLogin(t *testing.T) {
	k := newKit(t)
	rec := k.login("203.0.113.1", testAdminUser, testAdminPass)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin login: %d %q", rec.Code, rec.Body.String())
	}
	m := bodyMap(t, rec)
	sameKeys(t, m, "token", "kind", "expires_at")
	if m["kind"] != "admin" || m["expires_at"] != stamp(t0.Add(constants.AdminSessionTTL)) {
		t.Fatalf("body = %v", m)
	}
	token, _ := m["token"].(string)
	if len(token) != 43 {
		t.Fatalf("token %q is not 43 base64url characters", token)
	}
	got := k.do(http.MethodGet, "/v1/session", token, nil)
	if got.Code != http.StatusOK || bodyMap(t, got)["kind"] != "admin" {
		t.Fatalf("the issued token does not work: %d %q", got.Code, got.Body.String())
	}
}

func TestAdminLoginWithMixedCaseName(t *testing.T) {
	k := newKit(t)
	rec := k.login("203.0.113.1", " Root-Admin ", testAdminPass)
	if rec.Code != http.StatusOK || bodyMap(t, rec)["kind"] != "admin" {
		t.Fatalf("normalised admin name: %d %q", rec.Code, rec.Body.String())
	}
}

func TestParentLogin(t *testing.T) {
	k := newKit(t)
	k.addParent("ana")
	rec := k.login("203.0.113.1", "ana", testParentPass)
	if rec.Code != http.StatusOK {
		t.Fatalf("parent login: %d %q", rec.Code, rec.Body.String())
	}
	m := bodyMap(t, rec)
	sameKeys(t, m, "token", "kind", "expires_at")
	if m["kind"] != "parent" || m["expires_at"] != stamp(t0.Add(constants.ParentSessionTTL)) {
		t.Fatalf("body = %v", m)
	}
}

func TestLoginFailuresLookAlike(t *testing.T) {
	k := newKit(t)
	k.addParent("ana")
	dave := k.addParent("dave")
	if err := k.Store.SetParentDisabled(context.Background(), dave.ID, true, t0); err != nil {
		t.Fatal(err)
	}
	if _, err := k.Store.CreateParent(context.Background(), "pending", "Pending", t0); err != nil {
		t.Fatal(err)
	}

	calls := 0
	orig := verifyPassword
	verifyPassword = func(hash, pw string) bool { calls++; return auth.VerifyPassword(hash, pw) }
	t.Cleanup(func() { verifyPassword = orig })

	cases := []struct{ name, user, pass string }{
		{"unknown user", "nobody", testParentPass},
		{"wrong password", "ana", "not the password"},
		{"disabled parent, right password", "dave", testParentPass},
		{"parent without a password", "pending", testParentPass},
		{"admin, wrong password", testAdminUser, "wrong-admin-password"},
		{"admin, parent's password", testAdminUser, testParentPass},
		{"non-ASCII username", "añb", testParentPass},
		{"empty username", "", testParentPass},
		{"empty password", "ana", ""},
	}
	for i, c := range cases {
		calls = 0
		rec := k.login(fmt.Sprintf("198.51.100.%d", i), c.user, c.pass)
		if rec.Code != http.StatusUnauthorized || strings.TrimSpace(rec.Body.String()) != invalidCreds {
			t.Errorf("%s: %d %q, want 401 %s", c.name, rec.Code, rec.Body.String(), invalidCreds)
		}
		if calls != 1 {
			t.Errorf("%s: %d password verifications, want 1", c.name, calls)
		}
	}

	calls = 0
	if rec := k.login("192.0.2.1", testAdminUser, testAdminPass); rec.Code != http.StatusOK {
		t.Fatalf("admin success: %d", rec.Code)
	}
	if calls != 0 {
		t.Errorf("admin success ran %d password verifications, want 0", calls)
	}
}

func TestLoginNormalisesUsername(t *testing.T) {
	k := newKit(t)
	p := k.addParent("ana")
	rec := k.login("203.0.113.1", " Ana ", testParentPass)
	if rec.Code != http.StatusOK {
		t.Fatalf("login: %d %q", rec.Code, rec.Body.String())
	}
	token := bodyMap(t, rec)["token"].(string)
	got := k.do(http.MethodGet, "/v1/session", token, nil)
	parent, _ := bodyMap(t, got)["parent"].(map[string]any)
	if parent["id"] != p.ID || parent["username"] != "ana" {
		t.Fatalf("session belongs to %v, want ana %s", parent, p.ID)
	}
}

func TestLoginBodyErrors(t *testing.T) {
	k := newKit(t)
	cases := []struct {
		name   string
		body   any
		status int
		code   string
		field  string
	}{
		{"unknown field", `{"username":"ana","password":"x","remember":true}`, 422, "unknown_field", "remember"},
		{"malformed", `{"username":`, 400, "bad_request", ""},
		{"empty body", ``, 400, "bad_request", ""},
		{"too large", `{"username":"` + strings.Repeat("a", constants.MaxBodyBytes) + `"}`, 413, "body_too_large", ""},
	}
	for _, c := range cases {
		rec := k.serve(k.newReq(http.MethodPost, "/v1/session", c.body))
		m := bodyMap(t, rec)
		if rec.Code != c.status || m["error"] != c.code {
			t.Errorf("%s: %d %q, want %d %s", c.name, rec.Code, rec.Body.String(), c.status, c.code)
		}
		if f, _ := m["field"].(string); f != c.field {
			t.Errorf("%s: field %q, want %q", c.name, f, c.field)
		}
	}
}

func TestLoginRateLimit(t *testing.T) {
	k := newKit(t)
	k.addParent("ana")
	const addr = "203.0.113.5"
	for i := 1; i <= constants.LoginAttemptsPerMinute; i++ {
		if rec := k.login(addr, "ana", "wrong password"); rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: %d, want 401", i, rec.Code)
		}
	}
	parent := k.login(addr, "ana", "wrong password")
	wantErr(t, parent, http.StatusTooManyRequests, "rate_limited")
	if got := parent.Header().Get("Retry-After"); got != "60" {
		t.Fatalf("Retry-After = %q, want 60", got)
	}
	// The same address is refused for the admin name too, with the same body and header.
	admin := k.login(addr, testAdminUser, testAdminPass)
	wantErr(t, admin, http.StatusTooManyRequests, "rate_limited")
	if admin.Header().Get("Retry-After") != "60" || admin.Body.String() != parent.Body.String() {
		t.Fatalf("admin and parent refusals differ: %q vs %q", admin.Body.String(), parent.Body.String())
	}
	if rec := k.login("203.0.113.6", testAdminUser, testAdminPass); rec.Code != http.StatusOK {
		t.Fatalf("another address, another user: %d, want 200", rec.Code)
	}
	k.now = k.now.Add(constants.LimiterWindow)
	if rec := k.login(addr, "ana", testParentPass); rec.Code != http.StatusOK {
		t.Fatalf("after the window: %d, want 200", rec.Code)
	}
}

func TestLoginRateLimitPerUser(t *testing.T) {
	k := newKit(t)
	k.addParent("ana")
	for i := 1; i <= constants.LoginAttemptsPerMinute; i++ {
		if rec := k.login(fmt.Sprintf("198.51.100.%d", i), "ana", "wrong password"); rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: %d, want 401", i, rec.Code)
		}
	}
	// A guess from a fresh address is refused even with the right password.
	wantErr(t, k.login("198.51.100.99", "ana", testParentPass), http.StatusTooManyRequests, "rate_limited")
	// The admin name is limited per user the same way, with the same body.
	for i := 1; i <= constants.LoginAttemptsPerMinute; i++ {
		k.login(fmt.Sprintf("192.0.2.%d", i), testAdminUser, "wrong-admin-password")
	}
	wantErr(t, k.login("192.0.2.99", testAdminUser, testAdminPass), http.StatusTooManyRequests, "rate_limited")
	if rec := k.login("198.51.100.99", "other", "x"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("an unrelated user: %d, want 401", rec.Code)
	}
}

func TestLoginLimiterChecksAddressFirst(t *testing.T) {
	k := newKit(t)
	k.addParent("ana")
	// Five distinct users exhaust the address; the blocked sixth names ana.
	for i := 1; i <= constants.LoginAttemptsPerMinute; i++ {
		k.login("203.0.113.7", fmt.Sprintf("filler%d", i), "x")
	}
	wantErr(t, k.login("203.0.113.7", "ana", "x"), http.StatusTooManyRequests, "rate_limited")
	// The blocked attempt must not have spent ana's budget.
	for i := 1; i <= constants.LoginAttemptsPerMinute; i++ {
		if rec := k.login("203.0.113.8", "ana", "wrong password"); rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d from a clean address: %d, want 401", i, rec.Code)
		}
	}
}

func TestLoginWithoutClientAddressSharesOneBucket(t *testing.T) {
	k := newKit(t)
	for i := 1; i <= constants.LoginAttemptsPerMinute; i++ {
		k.login("", fmt.Sprintf("user%d", i), "x")
	}
	wantErr(t, k.login("", "user9", "x"), http.StatusTooManyRequests, "rate_limited")
	if rec := k.login("203.0.113.1", "user9", "x"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("with an address: %d, want 401", rec.Code)
	}
}

func TestSessionShape(t *testing.T) {
	k := newKit(t)
	p := k.addParent("ana")

	admin := k.do(http.MethodGet, "/v1/session", k.asAdmin(), nil)
	if admin.Code != http.StatusOK {
		t.Fatalf("admin: %d", admin.Code)
	}
	m := bodyMap(t, admin)
	sameKeys(t, m, "kind", "expires_at")
	if m["kind"] != "admin" || m["expires_at"] != stamp(t0.Add(constants.AdminSessionTTL)) {
		t.Fatalf("admin body = %v", m)
	}

	rec := k.do(http.MethodGet, "/v1/session", k.asParent("ana"), nil)
	m = bodyMap(t, rec)
	sameKeys(t, m, "kind", "expires_at", "parent")
	if m["kind"] != "parent" || m["expires_at"] != stamp(t0.Add(constants.ParentSessionTTL)) {
		t.Fatalf("parent body = %v", m)
	}
	parent := m["parent"].(map[string]any)
	sameKeys(t, parent, "id", "username", "display_name", "units", "password_changed_at")
	if parent["id"] != p.ID || parent["username"] != "ana" || parent["display_name"] != "Name of ana" {
		t.Fatalf("parent = %v", parent)
	}
	if parent["password_changed_at"] != stamp(k.now) {
		t.Fatalf("password_changed_at = %v, want %v", parent["password_changed_at"], stamp(k.now))
	}
	units := parent["units"].(map[string]any)
	sameKeys(t, units, "weight", "length", "volume", "temp")
	if units["weight"] != "kg" || units["length"] != "cm" || units["volume"] != "ml" || units["temp"] != "c" {
		t.Fatalf("units = %v", units)
	}
}

func TestSessionRequiresToken(t *testing.T) {
	k := newKit(t)
	wantErr(t, k.do(http.MethodGet, "/v1/session", "", nil), http.StatusUnauthorized, "no_session")
	wantErr(t, k.do(http.MethodGet, "/v1/session", "not-a-token", nil), http.StatusUnauthorized, "no_session")
	wantErr(t, k.do(http.MethodDelete, "/v1/session", "", nil), http.StatusUnauthorized, "no_session")
}

func TestSessionSlides(t *testing.T) {
	k := newKit(t)
	parent := k.asParent("ana")
	admin := k.asAdmin()
	expiry := func(token string) string {
		t.Helper()
		rec := k.do(http.MethodGet, "/v1/session", token, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("session at %v: %d %q", k.now, rec.Code, rec.Body.String())
		}
		return bodyMap(t, rec)["expires_at"].(string)
	}
	ttl, touch := constants.ParentSessionTTL, constants.SessionTouchInterval

	k.now = t0.Add(touch / 2)
	if got, want := expiry(parent), stamp(t0.Add(ttl)); got != want {
		t.Errorf("within the touch interval: %s, want unchanged %s", got, want)
	}
	k.now = t0.Add(touch)
	if got, want := expiry(parent), stamp(t0.Add(ttl)); got != want {
		t.Errorf("exactly one interval in: %s, want unchanged %s", got, want)
	}
	k.now = t0.Add(touch + time.Second)
	if got, want := expiry(parent), stamp(k.now.Add(ttl)); got != want {
		t.Errorf("just past the interval: %s, want %s", got, want)
	}

	k.now = t0.Add(touch + time.Second + 29*24*time.Hour)
	if got, want := expiry(parent), stamp(k.now.Add(ttl)); got != want {
		t.Errorf("day 29 of the last extension: %s, want %s", got, want)
	}

	k.now = t0.Add(11 * time.Hour)
	if got, want := expiry(admin), stamp(t0.Add(constants.AdminSessionTTL)); got != want {
		t.Errorf("admin at hour 11: %s, want unchanged %s", got, want)
	}
}

func TestSessionSlideMovesTheStoredExpiry(t *testing.T) {
	k := newKit(t)
	token := k.asParent("ana")
	k.now = t0.Add(29 * 24 * time.Hour)
	k.do(http.MethodGet, "/v1/session", token, nil)
	p, err := k.Store.SessionByToken(context.Background(), auth.HashToken(token), k.now)
	want := t0.Add(59 * 24 * time.Hour)
	if err != nil || !p.ExpiresAt.Equal(want) {
		t.Fatalf("stored expiry = %v, %v; want %v", p.ExpiresAt, err, want)
	}
}

func TestExpiredSessionIsRefused(t *testing.T) {
	k := newKit(t)
	admin := k.asAdmin()
	parent := k.asParent("ana")

	k.now = t0.Add(constants.AdminSessionTTL - time.Second)
	if rec := k.do(http.MethodGet, "/v1/session", admin, nil); rec.Code != http.StatusOK {
		t.Fatalf("admin 1s before expiry: %d", rec.Code)
	}
	k.now = t0.Add(constants.AdminSessionTTL)
	wantErr(t, k.do(http.MethodGet, "/v1/session", admin, nil), http.StatusUnauthorized, "no_session")

	k.now = t0.Add(constants.ParentSessionTTL)
	wantErr(t, k.do(http.MethodGet, "/v1/session", parent, nil), http.StatusUnauthorized, "no_session")
}

func TestDisabledParentSessionIsRefused(t *testing.T) {
	k := newKit(t)
	token := k.asParent("ana")
	p, _, _ := k.Store.ParentByUsername(context.Background(), "ana")
	if err := k.Store.SetParentDisabled(context.Background(), p.ID, true, t0); err != nil {
		t.Fatal(err)
	}
	wantErr(t, k.do(http.MethodGet, "/v1/session", token, nil), http.StatusUnauthorized, "no_session")
}

func TestLogout(t *testing.T) {
	k := newKit(t)
	first, second := k.asParent("ana"), k.asParent("ana")
	rec := k.do(http.MethodDelete, "/v1/session", first, nil)
	if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Fatalf("logout: %d %q, want 204 and no body", rec.Code, rec.Body.String())
	}
	wantErr(t, k.do(http.MethodGet, "/v1/session", first, nil), http.StatusUnauthorized, "no_session")
	if rec := k.do(http.MethodGet, "/v1/session", second, nil); rec.Code != http.StatusOK {
		t.Fatalf("the parent's other session: %d, want 200", rec.Code)
	}
}

func TestLogHasNoIdentifyingText(t *testing.T) {
	k := newKit(t)
	const user, addr = "zelda-unique-731", "203.0.113.77"
	k.addParent(user)

	ok := k.login(addr, user, testParentPass)
	token := bodyMap(t, ok)["token"].(string)
	k.login(addr, user, "wrong-password-unique-5521")
	k.login(addr, testAdminUser, testAdminPass)
	k.do(http.MethodGet, "/v1/session", token, nil)
	k.do(http.MethodDelete, "/v1/session", token, nil)
	k.do(http.MethodGet, "/v1/session", token, nil)

	logs := k.Log.String()
	for _, secret := range []string{user, addr, testParentPass, "wrong-password-unique-5521", testAdminUser, testAdminPass, testInternalToken, token} {
		if strings.Contains(logs, secret) {
			t.Errorf("log contains %q:\n%s", secret, logs)
		}
	}
	if n := strings.Count(logs, "\n"); n != 6 {
		t.Errorf("%d log lines for 6 requests:\n%s", n, logs)
	}
	for _, field := range []string{`"route":"POST /v1/session"`, `"route":"GET /v1/session"`, `"status":200`, `"status":401`, `"principal_kind":"parent"`, `"principal_kind":"none"`, `"request_id":"`, `"duration_ms":`} {
		if !strings.Contains(logs, field) {
			t.Errorf("log lacks %s:\n%s", field, logs)
		}
	}
}
