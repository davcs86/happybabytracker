package httpapi

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/config"
	"github.com/davcs86/happybabytracker/api/internal/constants"
)

const (
	origin       = "https://example.test"
	notFoundBody = `{"error":"not_found"}`
)

var redeemSeq int

// redeem posts the token from a fresh client address, so the limiter never interferes unless a test wants it to.
func (k *kit) redeem(token, password string) *httptest.ResponseRecorder {
	k.t.Helper()
	redeemSeq++
	return k.redeemFrom(fmt.Sprintf("test-addr-%d", redeemSeq), token, password)
}

func (k *kit) redeemFrom(addr, token, password string) *httptest.ResponseRecorder {
	k.t.Helper()
	r := k.newReq(http.MethodPost, "/v1/tokens/"+token+"/redeem", map[string]string{"password": password})
	r.Header.Set("X-Client-Address", addr)
	return k.serve(r)
}

func tokenOf(t *testing.T, link, kind string) string {
	t.Helper()
	prefix := origin + "/" + kind + "/"
	if !strings.HasPrefix(link, prefix) || len(link) != len(prefix)+43 {
		t.Fatalf("link %q is not %s<43 characters>", link, prefix)
	}
	return strings.TrimPrefix(link, prefix)
}

// createParent has the admin create the parent and returns the parent's id and invite token.
func (k *kit) createParent(admin, username string) (id, token string) {
	k.t.Helper()
	rec := k.do(http.MethodPost, "/v1/admin/parents", admin, map[string]string{"username": username, "display_name": "Name of " + username})
	if rec.Code != http.StatusCreated {
		k.t.Fatalf("create %s: %d %q", username, rec.Code, rec.Body.String())
	}
	m := bodyMap(k.t, rec)
	return m["parent"].(map[string]any)["id"].(string), tokenOf(k.t, m["invite_url"].(string), "invite")
}

func (k *kit) resetLink(admin, parentID string) map[string]any {
	k.t.Helper()
	rec := k.do(http.MethodPost, "/v1/admin/parents/"+parentID+"/reset-link", admin, nil)
	if rec.Code != http.StatusOK {
		k.t.Fatalf("reset-link: %d %q", rec.Code, rec.Body.String())
	}
	return bodyMap(k.t, rec)
}

func TestAdminCreatesParent(t *testing.T) {
	k := newKit(t)
	admin := k.asAdmin()

	rec := k.do(http.MethodPost, "/v1/admin/parents", admin, map[string]string{"username": " Ana ", "display_name": "  Ana Díaz "})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %q", rec.Code, rec.Body.String())
	}
	m := bodyMap(t, rec)
	sameKeys(t, m, "parent", "invite_url", "expires_at")
	parent := m["parent"].(map[string]any)
	sameKeys(t, parent, "id", "username", "display_name", "disabled", "has_password", "sso_linked", "created_at")
	if parent["username"] != "ana" || parent["display_name"] != "Ana Díaz" || parent["disabled"] != false ||
		parent["has_password"] != false || parent["sso_linked"] != false || parent["created_at"] != stamp(t0) || parent["id"] == "" {
		t.Fatalf("parent = %v", parent)
	}
	tokenOf(t, m["invite_url"].(string), "invite")
	if m["expires_at"] != stamp(t0.Add(constants.InviteTTL)) {
		t.Fatalf("expires_at = %v", m["expires_at"])
	}

	for _, name := range []string{"ana", "ANA", " Ana "} {
		wantErr(t, k.do(http.MethodPost, "/v1/admin/parents", admin, map[string]string{"username": name, "display_name": "Other"}), http.StatusConflict, "username_taken")
	}
	for _, name := range []string{testAdminUser, " Root-Admin "} {
		wantErr(t, k.do(http.MethodPost, "/v1/admin/parents", admin, map[string]string{"username": name, "display_name": "Admin"}), http.StatusConflict, "username_taken")
	}
}

func TestAdminCreateParentValidation(t *testing.T) {
	k := newKit(t)
	admin := k.asAdmin()
	cases := []struct{ name, user, display, field string }{
		{"short username", "ab", "Ok", "username"},
		{"non-ASCII username", "añb", "Ok", "username"},
		{"empty username", "", "Ok", "username"},
		{"empty display name", "ana", "", "display_name"},
		{"blank display name", "ana", " \t ", "display_name"},
		{"65 characters", "ana", strings.Repeat("a", constants.DisplayNameMaxLen+1), "display_name"},
		{"65 multibyte characters", "ana", strings.Repeat("é", constants.DisplayNameMaxLen+1), "display_name"},
	}
	for _, c := range cases {
		rec := k.do(http.MethodPost, "/v1/admin/parents", admin, map[string]string{"username": c.user, "display_name": c.display})
		if rec.Code != http.StatusUnprocessableEntity || bodyMap(t, rec)["field"] != c.field || bodyMap(t, rec)["error"] != "invalid" {
			t.Errorf("%s: %d %q, want 422 invalid on %s", c.name, rec.Code, rec.Body.String(), c.field)
		}
	}
	if rec := k.do(http.MethodPost, "/v1/admin/parents", admin, map[string]string{"username": "ana", "display_name": strings.Repeat("é", constants.DisplayNameMaxLen)}); rec.Code != http.StatusCreated {
		t.Errorf("64 multibyte characters: %d %q, want 201", rec.Code, rec.Body.String())
	}
	rec := k.do(http.MethodPost, "/v1/admin/parents", admin, `{"username":"bob","display_name":"Bob","role":"admin"}`)
	if rec.Code != http.StatusUnprocessableEntity || bodyMap(t, rec)["error"] != "unknown_field" || bodyMap(t, rec)["field"] != "role" {
		t.Errorf("unknown field: %d %q", rec.Code, rec.Body.String())
	}
	if rec := k.do(http.MethodPost, "/v1/admin/parents", admin, "{"); rec.Code != http.StatusBadRequest {
		t.Errorf("malformed body: %d", rec.Code)
	}
	list := bodyMap(t, k.do(http.MethodGet, "/v1/admin/parents", admin, nil))["parents"].([]any)
	if len(list) != 1 {
		t.Errorf("%d parents exist after the refusals, want only the one valid create", len(list))
	}
}

func TestInviteRedeem(t *testing.T) {
	k := newKit(t)
	admin := k.asAdmin()
	_, token := k.createParent(admin, "ana")

	peek := k.do(http.MethodGet, "/v1/tokens/"+token, "", nil)
	if peek.Code != http.StatusOK {
		t.Fatalf("peek: %d %q", peek.Code, peek.Body.String())
	}
	m := bodyMap(t, peek)
	sameKeys(t, m, "kind", "username")
	if m["kind"] != "invite" || m["username"] != "ana" {
		t.Fatalf("peek = %v", m)
	}

	rec := k.redeem(token, "twelve chars")
	if rec.Code != http.StatusOK {
		t.Fatalf("redeem: %d %q", rec.Code, rec.Body.String())
	}
	m = bodyMap(t, rec)
	sameKeys(t, m, "token", "kind", "expires_at")
	if m["kind"] != "parent" || m["expires_at"] != stamp(t0.Add(constants.ParentSessionTTL)) {
		t.Fatalf("session = %v", m)
	}
	session := k.do(http.MethodGet, "/v1/session", m["token"].(string), nil)
	if session.Code != http.StatusOK || bodyMap(t, session)["kind"] != "parent" {
		t.Fatalf("GET /v1/session: %d %q", session.Code, session.Body.String())
	}
	if rec := k.login("203.0.113.9", "ana", "twelve chars"); rec.Code != http.StatusOK {
		t.Errorf("login with the chosen password: %d %q", rec.Code, rec.Body.String())
	}
	listed := bodyMap(t, k.do(http.MethodGet, "/v1/admin/parents", admin, nil))["parents"].([]any)
	if listed[0].(map[string]any)["has_password"] != true {
		t.Errorf("has_password after redemption = %v", listed[0])
	}
}

func TestTokenRefusals(t *testing.T) {
	k := newKit(t)
	admin := k.asAdmin()

	t.Run("redeemed twice", func(t *testing.T) {
		_, token := k.createParent(admin, "twice")
		first := k.redeem(token, "first password 1")
		if first.Code != http.StatusOK {
			t.Fatalf("first: %d %q", first.Code, first.Body.String())
		}
		wantErr(t, k.redeem(token, "second password"), http.StatusNotFound, "not_found")
		wantErr(t, k.do(http.MethodGet, "/v1/tokens/"+token, "", nil), http.StatusNotFound, "not_found")
		if got := k.do(http.MethodGet, "/v1/session", bodyMap(t, first)["token"].(string), nil); got.Code != http.StatusOK {
			t.Errorf("the first session was undone by the second redemption: %d", got.Code)
		}
		if rec := k.login("203.0.113.1", "twice", "second password"); rec.Code != http.StatusUnauthorized {
			t.Errorf("the second password was accepted: %d", rec.Code)
		}
	})

	t.Run("invite after 72h", func(t *testing.T) {
		_, token := k.createParent(admin, "lateinvite")
		k.now = t0.Add(constants.InviteTTL)
		wantErr(t, k.redeem(token, "late password 12"), http.StatusNotFound, "not_found")
		k.now = t0.Add(constants.InviteTTL + time.Second)
		wantErr(t, k.redeem(token, "late password 12"), http.StatusNotFound, "not_found")
		wantErr(t, k.do(http.MethodGet, "/v1/tokens/"+token, "", nil), http.StatusNotFound, "not_found")
		k.now = t0
	})

	t.Run("reset after 1h", func(t *testing.T) {
		p := k.addParent("lagging")
		link := k.resetLink(admin, p.ID)["reset_url"].(string)
		token := tokenOf(t, link, "reset")
		k.now = t0.Add(constants.ResetTTL + time.Second)
		wantErr(t, k.redeem(token, "late password 12"), http.StatusNotFound, "not_found")
		k.now = t0
	})

	t.Run("disabled parent", func(t *testing.T) {
		id, token := k.createParent(admin, "blocked")
		if rec := k.do(http.MethodPost, "/v1/admin/parents/"+id+"/disable", admin, nil); rec.Code != http.StatusNoContent {
			t.Fatalf("disable: %d", rec.Code)
		}
		wantErr(t, k.redeem(token, "blocked password"), http.StatusNotFound, "not_found")
		wantErr(t, k.do(http.MethodGet, "/v1/tokens/"+token, "", nil), http.StatusNotFound, "not_found")
		if rec := k.login("203.0.113.2", "blocked", "blocked password"); rec.Code != http.StatusUnauthorized {
			t.Errorf("a refused redemption set a usable password: %d", rec.Code)
		}
		k.do(http.MethodPost, "/v1/admin/parents/"+id+"/enable", admin, nil)
		if rec := k.redeem(token, "blocked password"); rec.Code != http.StatusOK {
			t.Errorf("the refusal burned the token: %d %q", rec.Code, rec.Body.String())
		}
	})

	t.Run("unknown token", func(t *testing.T) {
		wantErr(t, k.redeem("not-a-token", "unknown password"), http.StatusNotFound, "not_found")
		wantErr(t, k.do(http.MethodGet, "/v1/tokens/not-a-token", "", nil), http.StatusNotFound, "not_found")
	})
}

func TestResetEndsSessions(t *testing.T) {
	k := newKit(t)
	admin := k.asAdmin()
	p := k.addParent("ana")
	one, two := k.asParent("ana"), k.asParent("ana")
	other := k.asParent("bob")

	m := k.resetLink(admin, p.ID)
	sameKeys(t, m, "reset_url", "expires_at")
	if m["expires_at"] != stamp(t0.Add(constants.ResetTTL)) {
		t.Errorf("expires_at = %v", m["expires_at"])
	}
	token := tokenOf(t, m["reset_url"].(string), "reset")
	if peek := bodyMap(t, k.do(http.MethodGet, "/v1/tokens/"+token, "", nil)); peek["kind"] != "reset" || peek["username"] != "ana" {
		t.Fatalf("peek = %v", peek)
	}
	if rec := k.redeem(token, "brand new password"); rec.Code != http.StatusOK {
		t.Fatalf("redeem: %d %q", rec.Code, rec.Body.String())
	}
	for _, old := range []string{one, two} {
		wantErr(t, k.do(http.MethodGet, "/v1/session", old, nil), http.StatusUnauthorized, "no_session")
	}
	if got := k.do(http.MethodGet, "/v1/session", other, nil); got.Code != http.StatusOK {
		t.Errorf("another parent's session ended: %d", got.Code)
	}
	if rec := k.login("203.0.113.3", "ana", testParentPass); rec.Code != http.StatusUnauthorized {
		t.Errorf("the old password still works: %d", rec.Code)
	}
	if rec := k.login("203.0.113.4", "ana", "brand new password"); rec.Code != http.StatusOK {
		t.Errorf("the new password does not work: %d", rec.Code)
	}
}

func TestResetLinkForAParentWithoutAPasswordIsAnInvite(t *testing.T) {
	k := newKit(t)
	admin := k.asAdmin()
	id, first := k.createParent(admin, "ana")

	m := k.resetLink(admin, id)
	sameKeys(t, m, "invite_url", "expires_at")
	if m["expires_at"] != stamp(t0.Add(constants.InviteTTL)) {
		t.Errorf("expires_at = %v, want the invite's 72h", m["expires_at"])
	}
	second := tokenOf(t, m["invite_url"].(string), "invite")
	if peek := bodyMap(t, k.do(http.MethodGet, "/v1/tokens/"+second, "", nil)); peek["kind"] != "invite" {
		t.Errorf("peek = %v", peek)
	}
	wantErr(t, k.do(http.MethodGet, "/v1/tokens/"+first, "", nil), http.StatusNotFound, "not_found")
}

func TestNewInviteReplacesOld(t *testing.T) {
	k := newKit(t)
	admin := k.asAdmin()
	id, first := k.createParent(admin, "ana")
	second := tokenOf(t, k.resetLink(admin, id)["invite_url"].(string), "invite")

	wantErr(t, k.redeem(first, "first password 1"), http.StatusNotFound, "not_found")
	if rec := k.redeem(second, "second password"); rec.Code != http.StatusOK {
		t.Fatalf("the newer link: %d %q", rec.Code, rec.Body.String())
	}

	// With a password set, two resets replace each other too.
	r1 := tokenOf(t, k.resetLink(admin, id)["reset_url"].(string), "reset")
	r2 := tokenOf(t, k.resetLink(admin, id)["reset_url"].(string), "reset")
	wantErr(t, k.redeem(r1, "reset password 1"), http.StatusNotFound, "not_found")
	if rec := k.redeem(r2, "reset password 2"); rec.Code != http.StatusOK {
		t.Fatalf("the newer reset: %d", rec.Code)
	}
}

func TestAdminListShowsNoKidData(t *testing.T) {
	k := newKit(t)
	admin := k.asAdmin()
	k.addParent("bob")
	k.createParent(admin, "ana")

	rec := k.do(http.MethodGet, "/v1/admin/parents", admin, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list: %d %q", rec.Code, rec.Body.String())
	}
	m := bodyMap(t, rec)
	sameKeys(t, m, "parents")
	rows := m["parents"].([]any)
	if len(rows) != 2 {
		t.Fatalf("%d rows, want 2", len(rows))
	}
	for _, row := range rows {
		sameKeys(t, row.(map[string]any), "id", "username", "display_name", "disabled", "has_password", "sso_linked", "created_at", "last_active_at", "link")
	}
	first, second := rows[0].(map[string]any), rows[1].(map[string]any)
	if first["username"] != "ana" || first["has_password"] != false || second["username"] != "bob" || second["has_password"] != true {
		t.Errorf("rows = %v", rows)
	}
}

func TestAdminListWhenEmpty(t *testing.T) {
	k := newKit(t)
	rec := k.do(http.MethodGet, "/v1/admin/parents", k.asAdmin(), nil)
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != `{"parents":[]}` {
		t.Fatalf("empty list: %d %q", rec.Code, rec.Body.String())
	}
}

func TestDisableEndsSessions(t *testing.T) {
	k := newKit(t)
	admin := k.asAdmin()
	p := k.addParent("ana")
	tok := k.asParent("ana")
	other := k.asParent("bob")

	rec := k.do(http.MethodPost, "/v1/admin/parents/"+p.ID+"/disable", admin, nil)
	if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Fatalf("disable: %d %q", rec.Code, rec.Body.String())
	}
	wantErr(t, k.do(http.MethodGet, "/v1/session", tok, nil), http.StatusUnauthorized, "no_session")
	wantErr(t, k.login("203.0.113.5", "ana", testParentPass), http.StatusUnauthorized, "invalid_credentials")
	if got := k.do(http.MethodGet, "/v1/session", other, nil); got.Code != http.StatusOK {
		t.Errorf("another parent's session ended: %d", got.Code)
	}
	listed := bodyMap(t, k.do(http.MethodGet, "/v1/admin/parents", admin, nil))["parents"].([]any)
	if listed[0].(map[string]any)["disabled"] != true {
		t.Errorf("list does not show the parent as disabled: %v", listed[0])
	}
}

func TestEnableRestoresLogin(t *testing.T) {
	k := newKit(t)
	admin := k.asAdmin()
	p := k.addParent("ana")
	k.do(http.MethodPost, "/v1/admin/parents/"+p.ID+"/disable", admin, nil)

	rec := k.do(http.MethodPost, "/v1/admin/parents/"+p.ID+"/enable", admin, nil)
	if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Fatalf("enable: %d %q", rec.Code, rec.Body.String())
	}
	if rec := k.login("203.0.113.6", "ana", testParentPass); rec.Code != http.StatusOK {
		t.Errorf("login after enable: %d %q", rec.Code, rec.Body.String())
	}
}

func TestUnknownParentIs404(t *testing.T) {
	k := newKit(t)
	admin := k.asAdmin()
	for _, action := range []string{"reset-link", "disable", "enable"} {
		wantErr(t, k.do(http.MethodPost, "/v1/admin/parents/no-such-parent/"+action, admin, nil), http.StatusNotFound, "not_found")
	}
}

func TestUnlinkParentSSO(t *testing.T) {
	k, db := fileKit(t)
	admin := k.asAdmin()
	ana := k.addParent("ana")
	k.linkParent(ana.ID, "sub-ana")
	tok, tok2 := k.asParent("ana"), k.asParent("ana")
	kid := k.createKid(tok, nil)
	dev := k.mustDevice(tok, testEndpoint)
	if rec := k.do(http.MethodPut, optinPath(kid, dev), tok, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("push opt in: %d %q", rec.Code, rec.Body.String())
	}
	k.mustNtfy(tok)
	if rec := k.do(http.MethodPut, ntfyOptinPath(kid), tok, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("ntfy opt in: %d %q", rec.Code, rec.Body.String())
	}
	listed := bodyMap(t, k.do(http.MethodGet, "/v1/admin/parents", admin, nil))["parents"].([]any)
	if listed[0].(map[string]any)["sso_linked"] != true {
		t.Fatalf("before unlink: %v", listed[0])
	}

	rec := k.do(http.MethodPost, "/v1/admin/parents/"+ana.ID+"/unlink-sso", admin, nil)
	if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Fatalf("unlink: %d %q", rec.Code, rec.Body.String())
	}
	for q, want := range map[string]int{
		`SELECT COUNT(*) FROM identities WHERE parent_id = ? AND provider <> 'local'`: 0,
		`SELECT COUNT(*) FROM identities WHERE parent_id = ? AND provider = 'local'`:  1,
		`SELECT COUNT(*) FROM sessions WHERE parent_id = ?`:                           0,
		`SELECT COUNT(*) FROM push_subscriptions WHERE parent_id = ?`:                 0,
		`SELECT COUNT(*) FROM ntfy_topics WHERE parent_id = ?`:                        0,
		`SELECT COUNT(*) FROM ntfy_kid_optins WHERE parent_id = ?`:                    0,
	} {
		var n int
		if err := db.QueryRow(q, ana.ID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != want {
			t.Errorf("%s = %d, want %d", q, n, want)
		}
	}
	var optins int
	if err := db.QueryRow(`SELECT COUNT(*) FROM push_kid_optins WHERE kid_id = ?`, kid).Scan(&optins); err != nil || optins != 0 {
		t.Errorf("push opt-ins = %d, %v; want 0", optins, err)
	}
	for _, s := range []string{tok, tok2} {
		wantErr(t, k.do(http.MethodGet, "/v1/session", s, nil), http.StatusUnauthorized, "no_session")
	}
	listed = bodyMap(t, k.do(http.MethodGet, "/v1/admin/parents", admin, nil))["parents"].([]any)
	if listed[0].(map[string]any)["sso_linked"] != false {
		t.Errorf("after unlink: %v", listed[0])
	}
	if rec := k.login("203.0.113.10", "ana", testParentPass); rec.Code != http.StatusOK {
		t.Errorf("password login after unlink: %d %q", rec.Code, rec.Body.String())
	}
	k.setMode(config.AuthBoth)
	wantErr(t, k.oidcLogin(oidcBody("sub-ana")), http.StatusForbidden, "not_linked")
}

func TestUnlinkUnknownParent(t *testing.T) {
	k := newKit(t)
	admin := k.asAdmin()
	sameResponse(t, "unlink-sso",
		k.do(http.MethodPost, "/v1/admin/parents/no-such-parent/unlink-sso", admin, nil),
		k.do(http.MethodPost, "/v1/admin/parents/no-such-parent/reset-link", admin, nil))
	wantErr(t, k.do(http.MethodPost, "/v1/admin/parents/no-such-parent/unlink-sso", admin, nil), http.StatusNotFound, "not_found")
}

func TestRedeemIsRateLimitedPerAddress(t *testing.T) {
	k := newKit(t)
	for i := 0; i < constants.LoginAttemptsPerMinute; i++ {
		wantErr(t, k.redeemFrom("203.0.113.7", "not-a-token", "some password 1"), http.StatusNotFound, "not_found")
	}
	rec := k.redeemFrom("203.0.113.7", "not-a-token", "some password 1")
	wantErr(t, rec, http.StatusTooManyRequests, "rate_limited")
	if rec.Header().Get("Retry-After") != "60" {
		t.Errorf("Retry-After = %q", rec.Header().Get("Retry-After"))
	}
	wantErr(t, k.login("203.0.113.8", "nobody", "x"), http.StatusUnauthorized, "invalid_credentials")
	if rec := k.redeemFrom("203.0.113.9", "not-a-token", "some password 1"); rec.Code != http.StatusNotFound {
		t.Errorf("another address: %d, want 404", rec.Code)
	}
	for i := 0; i < 3*constants.LoginAttemptsPerMinute; i++ {
		r := k.newReq(http.MethodGet, "/v1/tokens/not-a-token", nil)
		r.Header.Set("X-Client-Address", "203.0.113.7")
		wantErr(t, k.serve(r), http.StatusNotFound, "not_found")
	}
}

func TestRedeemPasswordLength(t *testing.T) {
	k := newKit(t)
	_, token := k.createParent(k.asAdmin(), "ana")
	for _, pw := range []string{"", strings.Repeat("a", constants.MinPasswordLen-1), strings.Repeat("é", constants.MinPasswordLen-1), strings.Repeat("a", constants.MaxPasswordLen+1)} {
		rec := k.redeem(token, pw)
		if rec.Code != http.StatusUnprocessableEntity || bodyMap(t, rec)["field"] != "password" || bodyMap(t, rec)["error"] != "invalid" {
			t.Errorf("%d characters: %d %q, want 422 invalid on password", len([]rune(pw)), rec.Code, rec.Body.String())
		}
	}
	if rec := k.redeem(token, strings.Repeat("é", constants.MinPasswordLen)); rec.Code != http.StatusOK {
		t.Errorf("12 multibyte characters: %d %q (the refusals must not burn the token)", rec.Code, rec.Body.String())
	}
}

func TestRedeemBodyErrors(t *testing.T) {
	k := newKit(t)
	_, token := k.createParent(k.asAdmin(), "ana")
	r := k.newReq(http.MethodPost, "/v1/tokens/"+token+"/redeem", `{"password":"twelve chars","extra":1}`)
	rec := k.serve(r)
	if rec.Code != http.StatusUnprocessableEntity || bodyMap(t, rec)["error"] != "unknown_field" || bodyMap(t, rec)["field"] != "extra" {
		t.Errorf("unknown field: %d %q", rec.Code, rec.Body.String())
	}
	if rec := k.serve(k.newReq(http.MethodPost, "/v1/tokens/"+token+"/redeem", "nope")); rec.Code != http.StatusBadRequest {
		t.Errorf("malformed: %d", rec.Code)
	}
}

func TestRedeemNeedsTheInternalToken(t *testing.T) {
	k := newKit(t)
	_, token := k.createParent(k.asAdmin(), "ana")
	for _, path := range []string{"/v1/tokens/" + token, "/v1/tokens/" + token + "/redeem"} {
		method := http.MethodGet
		if strings.HasSuffix(path, "/redeem") {
			method = http.MethodPost
		}
		r := k.newReq(method, path, `{"password":"twelve chars"}`)
		r.Header.Del("Authorization")
		wantErr(t, k.serve(r), http.StatusUnauthorized, "unauthorized")
	}
}

func TestTokenAndPasswordNeverReachTheLog(t *testing.T) {
	k := newKit(t)
	_, token := k.createParent(k.asAdmin(), "ana")
	k.do(http.MethodGet, "/v1/tokens/"+token, "", nil)
	k.redeem(token, "a secret password")
	if log := k.Log.String(); strings.Contains(log, token) || strings.Contains(log, "a secret password") || !strings.Contains(log, "/v1/tokens/{token}/redeem") {
		t.Errorf("log leaks or lost the route pattern:\n%s", log)
	}
}

func TestAdminListLastActiveAndLink(t *testing.T) {
	k := newKit(t)
	admin := k.asAdmin()
	k.addParent("bob") // signs in below
	k.createParent(admin, "ana")
	k.asParent("bob")
	k.now = t0.Add(2 * time.Hour)
	k.asParent("bob")

	rows := bodyMap(t, k.do(http.MethodGet, "/v1/admin/parents", admin, nil))["parents"].([]any)
	ana, bob := rows[0].(map[string]any), rows[1].(map[string]any)
	if ana["last_active_at"] != nil || bob["last_active_at"] != stamp(t0.Add(2*time.Hour)) {
		t.Errorf("last_active_at: ana %v, bob %v; want null and the newest sign-in", ana["last_active_at"], bob["last_active_at"])
	}
	link := ana["link"].(map[string]any)
	sameKeys(t, link, "kind", "issued_at", "expires_at")
	if link["kind"] != "invite" || link["issued_at"] != stamp(t0) || link["expires_at"] != stamp(t0.Add(constants.InviteTTL)) {
		t.Errorf("ana link = %v", link)
	}
	if bob["link"] != nil {
		t.Errorf("bob link = %v, want null", bob["link"])
	}

	k.now = t0.Add(constants.InviteTTL + time.Hour) // an expired link is still listed, so the line can say so
	rows = bodyMap(t, k.do(http.MethodGet, "/v1/admin/parents", k.asAdmin(), nil))["parents"].([]any)
	if rows[0].(map[string]any)["link"] == nil {
		t.Error("an expired unused link was dropped")
	}
}

func TestAdminListLastActiveNullAfterDisable(t *testing.T) {
	k := newKit(t)
	admin := k.asAdmin()
	p := k.addParent("bob")
	k.asParent("bob")
	if rec := k.do(http.MethodPost, "/v1/admin/parents/"+p.ID+"/disable", admin, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("disable: %d %q", rec.Code, rec.Body.String())
	}
	rows := bodyMap(t, k.do(http.MethodGet, "/v1/admin/parents", admin, nil))["parents"].([]any)
	if rows[0].(map[string]any)["last_active_at"] != nil {
		t.Errorf("a disabled parent has no sessions, got %v", rows[0])
	}
}

func TestPasswordRedeemRefusedInOIDCLeavesTheTokenUsable(t *testing.T) {
	k := newKit(t)
	admin := k.asAdmin()
	id, invite := k.createParent(admin, "ana")
	k.setMode(config.AuthOIDC)
	wantErr(t, k.redeem(invite, "invite password 1"), http.StatusForbidden, "mode_refused")
	rec := k.do(http.MethodPost, "/v1/tokens/"+invite+"/redeem", "", `{"x":1}`)
	if rec.Code != http.StatusUnprocessableEntity || bodyMap(t, rec)["error"] != "unknown_field" {
		t.Errorf("unknown field in oidc: %d %q, want 422 unknown_field", rec.Code, rec.Body.String())
	}
	wantErr(t, k.redeem("not-a-token", "invite password 1"), http.StatusForbidden, "mode_refused")
	for i := 0; i <= constants.LoginAttemptsPerMinute; i++ {
		wantErr(t, k.redeemFrom("203.0.113.20", invite, "invite password 1"), http.StatusForbidden, "mode_refused")
	}
	k.setMode(config.AuthBoth)
	if rec := k.redeemFrom("203.0.113.20", invite, "invite password 1"); rec.Code != http.StatusOK {
		t.Fatalf("invite in both after the refusals, from the refused address: %d %q", rec.Code, rec.Body.String())
	}

	reset := tokenOf(t, k.resetLink(admin, id)["reset_url"].(string), "reset")
	k.setMode(config.AuthOIDC)
	wantErr(t, k.redeem(reset, "reset password 12"), http.StatusForbidden, "mode_refused")
	k.setMode(config.AuthBoth)
	if rec := k.redeem(reset, "reset password 12"); rec.Code != http.StatusOK {
		t.Fatalf("reset in both after the refusal: %d %q", rec.Code, rec.Body.String())
	}
}
