package httpapi

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

func TestParentCannotReachAdmin(t *testing.T) {
	k := newKit(t)
	parent := k.asParent("ana")
	for _, c := range []struct{ method, path string }{
		{http.MethodPost, "/v1/admin/parents"},
		{http.MethodGet, "/v1/admin/parents"},
		{http.MethodPost, "/v1/admin/parents/x/reset-link"},
		{http.MethodPost, "/v1/admin/parents/x/disable"},
		{http.MethodPost, "/v1/admin/parents/x/enable"},
	} {
		wantErr(t, k.do(c.method, c.path, parent, nil), http.StatusForbidden, "forbidden")
		wantErr(t, k.do(c.method, c.path, "", nil), http.StatusUnauthorized, "no_session")
	}
}

func TestAdminCannotReachMe(t *testing.T) {
	k := newKit(t)
	admin := k.asAdmin()
	for _, c := range []struct{ method, path string }{
		{http.MethodPatch, "/v1/me"},
		{http.MethodPost, "/v1/me/password"},
		{http.MethodGet, "/v1/me/sessions"},
		{http.MethodDelete, "/v1/me/sessions/x"},
	} {
		wantErr(t, k.do(c.method, c.path, admin, nil), http.StatusForbidden, "forbidden")
		wantErr(t, k.do(c.method, c.path, "", nil), http.StatusUnauthorized, "no_session")
	}
}

func parentObject(t *testing.T, m map[string]any, name, display string, units ...string) {
	t.Helper()
	sameKeys(t, m, "id", "username", "display_name", "units")
	u := m["units"].(map[string]any)
	sameKeys(t, u, "weight", "length", "volume", "temp")
	if m["username"] != name || m["display_name"] != display || m["id"] == "" ||
		u["weight"] != units[0] || u["length"] != units[1] || u["volume"] != units[2] || u["temp"] != units[3] {
		t.Fatalf("parent = %v, want %s %q %v", m, name, display, units)
	}
}

func TestUpdateProfile(t *testing.T) {
	k := newKit(t)
	tok := k.asParent("ana")

	rec := k.do(http.MethodPatch, "/v1/me", tok, map[string]any{"display_name": "  Ana Díaz  "})
	if rec.Code != http.StatusOK {
		t.Fatalf("patch: %d %q", rec.Code, rec.Body.String())
	}
	parentObject(t, bodyMap(t, rec), "ana", "Ana Díaz", "kg", "cm", "ml", "c")

	rec = k.do(http.MethodPatch, "/v1/me", tok, map[string]any{"units": map[string]string{"weight": "lb", "temp": "f"}})
	if rec.Code != http.StatusOK {
		t.Fatalf("patch units: %d %q", rec.Code, rec.Body.String())
	}
	parentObject(t, bodyMap(t, rec), "ana", "Ana Díaz", "lb", "cm", "ml", "f")

	rec = k.do(http.MethodPatch, "/v1/me", tok, map[string]any{"display_name": "Ana", "units": map[string]string{"length": "in", "volume": "floz"}})
	parentObject(t, bodyMap(t, rec), "ana", "Ana", "lb", "in", "floz", "f")

	got := bodyMap(t, k.do(http.MethodGet, "/v1/session", tok, nil))["parent"].(map[string]any)
	parentObject(t, got, "ana", "Ana", "lb", "in", "floz", "f")
}

func TestUpdateProfileEmptyBodyChangesNothing(t *testing.T) {
	k := newKit(t)
	tok := k.asParent("ana")
	k.do(http.MethodPatch, "/v1/me", tok, map[string]any{"units": map[string]string{"weight": "lb"}})
	rec := k.do(http.MethodPatch, "/v1/me", tok, map[string]any{})
	if rec.Code != http.StatusOK {
		t.Fatalf("empty patch: %d %q", rec.Code, rec.Body.String())
	}
	parentObject(t, bodyMap(t, rec), "ana", "Name of ana", "lb", "cm", "ml", "c")
}

func TestUpdateProfileRefusals(t *testing.T) {
	k := newKit(t)
	tok := k.asParent("ana")
	cases := []struct {
		name, body, code, field string
		status                  int
	}{
		{"unknown field", `{"username":"x"}`, "unknown_field", "username", 422},
		{"unknown unit field", `{"units":{"mass":"kg"}}`, "unknown_field", "mass", 422},
		{"blank name", `{"display_name":"   "}`, "invalid", "display_name", 422},
		{"empty name", `{"display_name":""}`, "invalid", "display_name", 422},
		{"long name", `{"display_name":"` + strings.Repeat("é", constants.DisplayNameMaxLen+1) + `"}`, "invalid", "display_name", 422},
		{"malformed", `{`, "bad_request", "", 400},
	}
	for _, c := range cases {
		rec := k.do(http.MethodPatch, "/v1/me", tok, c.body)
		want := `{"error":"` + c.code + `"}`
		if c.field != "" {
			want = `{"error":"` + c.code + `","field":"` + c.field + `"}`
		}
		if rec.Code != c.status || strings.TrimSpace(rec.Body.String()) != want {
			t.Errorf("%s: %d %q, want %d %s", c.name, rec.Code, rec.Body.String(), c.status, want)
		}
	}
	if rec := k.do(http.MethodPatch, "/v1/me", tok, `{"display_name":"`+strings.Repeat("é", constants.DisplayNameMaxLen)+`"}`); rec.Code != http.StatusOK {
		t.Errorf("64 multibyte characters: %d", rec.Code)
	}
}

func TestUnitsValidation(t *testing.T) {
	k := newKit(t)
	tok := k.asParent("ana")
	for _, c := range []struct{ key, value string }{
		{"weight", "stone"}, {"weight", ""}, {"weight", "KG"}, {"weight", "ml"},
		{"length", "m"}, {"volume", "cup"}, {"temp", "k"},
	} {
		body := map[string]any{"display_name": "Changed", "units": map[string]string{c.key: c.value}}
		rec := k.do(http.MethodPatch, "/v1/me", tok, body)
		if rec.Code != http.StatusUnprocessableEntity || bodyMap(t, rec)["field"] != "units."+c.key || bodyMap(t, rec)["error"] != "invalid" {
			t.Errorf("%s=%q: %d %q, want 422 invalid on units.%s", c.key, c.value, rec.Code, rec.Body.String(), c.key)
		}
	}
	got := bodyMap(t, k.do(http.MethodGet, "/v1/session", tok, nil))["parent"].(map[string]any)
	parentObject(t, got, "ana", "Name of ana", "kg", "cm", "ml", "c")

	for _, units := range []map[string]string{{"weight": "kg"}, {"weight": "lb"}, {"length": "cm"}, {"length": "in"}, {"volume": "ml"}, {"volume": "floz"}, {"temp": "c"}, {"temp": "f"}} {
		if rec := k.do(http.MethodPatch, "/v1/me", tok, map[string]any{"units": units}); rec.Code != http.StatusOK {
			t.Errorf("%v: %d %q", units, rec.Code, rec.Body.String())
		}
	}
}

func TestChangePassword(t *testing.T) {
	k := newKit(t)
	calling, other := k.asParent("ana"), k.asParent("ana")
	bob := k.asParent("bob")
	const newPass = "a brand new password"

	wantErr(t, k.do(http.MethodPost, "/v1/me/password", calling, map[string]string{"current": "not the password", "new": newPass}), http.StatusForbidden, "wrong_password")
	if got := k.do(http.MethodGet, "/v1/session", other, nil); got.Code != http.StatusOK {
		t.Fatalf("a refused change ended another session: %d", got.Code)
	}

	rec := k.do(http.MethodPost, "/v1/me/password", calling, map[string]string{"current": testParentPass, "new": newPass})
	if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Fatalf("change: %d %q", rec.Code, rec.Body.String())
	}
	if got := k.do(http.MethodGet, "/v1/session", calling, nil); got.Code != http.StatusOK {
		t.Errorf("the calling session ended: %d", got.Code)
	}
	wantErr(t, k.do(http.MethodGet, "/v1/session", other, nil), http.StatusUnauthorized, "no_session")
	if got := k.do(http.MethodGet, "/v1/session", bob, nil); got.Code != http.StatusOK {
		t.Errorf("another parent's session ended: %d", got.Code)
	}
	if rec := k.login("203.0.113.1", "ana", newPass); rec.Code != http.StatusOK {
		t.Errorf("new password: %d", rec.Code)
	}
	if rec := k.login("203.0.113.2", "ana", testParentPass); rec.Code != http.StatusUnauthorized {
		t.Errorf("old password still works: %d", rec.Code)
	}
}

func TestChangePasswordLength(t *testing.T) {
	k := newKit(t)
	tok := k.asParent("ana")
	for _, pw := range []string{"", strings.Repeat("a", constants.MinPasswordLen-1), strings.Repeat("a", constants.MaxPasswordLen+1)} {
		rec := k.do(http.MethodPost, "/v1/me/password", tok, map[string]string{"current": testParentPass, "new": pw})
		if rec.Code != http.StatusUnprocessableEntity || bodyMap(t, rec)["field"] != "new" || bodyMap(t, rec)["error"] != "invalid" {
			t.Errorf("%d characters: %d %q, want 422 invalid on new", len(pw), rec.Code, rec.Body.String())
		}
	}
	if rec := k.login("203.0.113.1", "ana", testParentPass); rec.Code != http.StatusOK {
		t.Errorf("the password changed despite the refusals: %d", rec.Code)
	}
	rec := k.do(http.MethodPost, "/v1/me/password", tok, `{"current":"x","new":"y","other":1}`)
	if rec.Code != http.StatusUnprocessableEntity || bodyMap(t, rec)["error"] != "unknown_field" {
		t.Errorf("unknown field: %d %q", rec.Code, rec.Body.String())
	}
}

func TestChangePasswordNeverLogsOrEchoesPasswords(t *testing.T) {
	k := newKit(t)
	tok := k.asParent("ana")
	rec := k.do(http.MethodPost, "/v1/me/password", tok, map[string]string{"current": "wrong secret one", "new": "wrong secret two"})
	if strings.Contains(rec.Body.String(), "secret") || strings.Contains(k.Log.String(), "secret") {
		t.Errorf("a password reached the response or the log:\n%s\n%s", rec.Body.String(), k.Log.String())
	}
}

func TestListOwnSessions(t *testing.T) {
	k := newKit(t)
	first := k.asParent("ana")
	k.now = t0.Add(time.Minute)
	second := k.asParent("ana")
	k.asParent("bob")

	rec := k.do(http.MethodGet, "/v1/me/sessions", second, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list: %d %q", rec.Code, rec.Body.String())
	}
	m := bodyMap(t, rec)
	sameKeys(t, m, "sessions")
	rows := m["sessions"].([]any)
	if len(rows) != 2 {
		t.Fatalf("%d sessions, want ana's two", len(rows))
	}
	for _, row := range rows {
		sameKeys(t, row.(map[string]any), "id", "created_at", "last_seen_at", "current")
	}
	newest, oldest := rows[0].(map[string]any), rows[1].(map[string]any)
	if newest["current"] != true || oldest["current"] != false ||
		newest["created_at"] != stamp(t0.Add(time.Minute)) || oldest["created_at"] != stamp(t0) || oldest["last_seen_at"] != stamp(t0) {
		t.Errorf("rows = %v", rows)
	}
	again := bodyMap(t, k.do(http.MethodGet, "/v1/me/sessions", first, nil))["sessions"].([]any)
	if again[0].(map[string]any)["current"] != false || again[1].(map[string]any)["current"] != true {
		t.Errorf("current flag follows the caller: %v", again)
	}
}

func TestListOwnSessionsSkipsExpiredOnes(t *testing.T) {
	k := newKit(t)
	k.asParent("ana")
	k.now = t0.Add(constants.ParentSessionTTL)
	tok := k.asParent("ana")
	rows := bodyMap(t, k.do(http.MethodGet, "/v1/me/sessions", tok, nil))["sessions"].([]any)
	if len(rows) != 1 {
		t.Errorf("%d sessions, want only the live one", len(rows))
	}
}

func TestDeleteOwnSession(t *testing.T) {
	k := newKit(t)
	calling, other := k.asParent("ana"), k.asParent("ana")
	bob := k.asParent("bob")
	idOf := func(token string) string {
		rows := bodyMap(t, k.do(http.MethodGet, "/v1/me/sessions", token, nil))["sessions"].([]any)
		for _, r := range rows {
			if r.(map[string]any)["current"] == true {
				return r.(map[string]any)["id"].(string)
			}
		}
		t.Fatal("no current session")
		return ""
	}
	otherID, bobID, callingID := idOf(other), idOf(bob), idOf(calling)

	wantErr(t, k.do(http.MethodDelete, "/v1/me/sessions/"+bobID, calling, nil), http.StatusNotFound, "not_found")
	wantErr(t, k.do(http.MethodDelete, "/v1/me/sessions/no-such-session", calling, nil), http.StatusNotFound, "not_found")
	if got := k.do(http.MethodGet, "/v1/session", bob, nil); got.Code != http.StatusOK {
		t.Fatalf("another parent's session was deleted: %d", got.Code)
	}

	rec := k.do(http.MethodDelete, "/v1/me/sessions/"+otherID, calling, nil)
	if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Fatalf("delete other: %d %q", rec.Code, rec.Body.String())
	}
	wantErr(t, k.do(http.MethodGet, "/v1/session", other, nil), http.StatusUnauthorized, "no_session")

	if rec := k.do(http.MethodDelete, "/v1/me/sessions/"+callingID, calling, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("delete own: %d %q", rec.Code, rec.Body.String())
	}
	wantErr(t, k.do(http.MethodGet, "/v1/session", calling, nil), http.StatusUnauthorized, "no_session")
}
