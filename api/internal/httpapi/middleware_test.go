package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

func ok(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"kind": PrincipalFrom(r.Context()).Kind})
}

// serveRoutes runs a request through the real middleware over a custom route list.
func (k *kit) serveRoutes(routes []Route, r *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	newMux(k.deps(), routes).ServeHTTP(rec, r)
	return rec
}

func TestAccessClasses(t *testing.T) {
	k := newKit(t)
	admin, parent := k.asAdmin(), k.asParent("ana")
	routes := []Route{
		{http.MethodGet, "/open", Open, ok},
		{http.MethodGet, "/internal", Internal, ok},
		{http.MethodGet, "/session", Session, ok},
		{http.MethodGet, "/admin", Admin, ok},
		{http.MethodGet, "/parent", Parent, ok},
	}
	cases := []struct {
		path, token string
		status      int
		code        string
	}{
		{"/open", "", 200, ""},
		{"/internal", "", 200, ""},
		{"/internal", parent, 200, ""}, // a session token is ignored, not required, on Internal
		{"/internal", "garbage", 200, ""},
		{"/session", "", 401, "no_session"},
		{"/session", "garbage", 401, "no_session"},
		{"/session", parent, 200, ""},
		{"/session", admin, 200, ""},
		{"/admin", "", 401, "no_session"},
		{"/admin", admin, 200, ""},
		{"/admin", parent, 403, "forbidden"},
		{"/parent", "", 401, "no_session"},
		{"/parent", parent, 200, ""},
		{"/parent", admin, 403, "forbidden"},
	}
	for _, c := range cases {
		r := k.newReq(http.MethodGet, c.path, nil)
		if c.token != "" {
			r.Header.Set("X-Session-Token", c.token)
		}
		rec := k.serveRoutes(routes, r)
		if c.status == 200 {
			if rec.Code != 200 {
				t.Errorf("%s with %.6q: %d %q, want 200", c.path, c.token, rec.Code, rec.Body.String())
			}
			continue
		}
		if rec.Code != c.status || strings.TrimSpace(rec.Body.String()) != `{"error":"`+c.code+`"}` {
			t.Errorf("%s with %.6q: %d %q, want %d %s", c.path, c.token, rec.Code, rec.Body.String(), c.status, c.code)
		}
	}
}

func TestOnlyOpenSkipsTheInternalToken(t *testing.T) {
	k := newKit(t)
	parent := k.asParent("ana")
	routes := []Route{
		{http.MethodGet, "/open", Open, ok},
		{http.MethodGet, "/internal", Internal, ok},
		{http.MethodGet, "/session", Session, ok},
		{http.MethodGet, "/admin", Admin, ok},
		{http.MethodGet, "/parent", Parent, ok},
		{http.MethodGet, "/caregiver", Caregiver, ok},
		{http.MethodGet, "/guardian", Guardian, ok},
		{http.MethodGet, "/unknown", Access(99), ok},
	}
	for _, rt := range routes {
		r := httptest.NewRequest(http.MethodGet, rt.Pattern, nil)
		r.Header.Set("X-Session-Token", parent)
		rec := k.serveRoutes(routes, r)
		if rt.Access == Open {
			if rec.Code != 200 {
				t.Errorf("%s: %d, want 200", rt.Pattern, rec.Code)
			}
			continue
		}
		wantErr(t, rec, http.StatusUnauthorized, "unauthorized")
	}
}

func TestUnknownAccessClassesAreRefused(t *testing.T) {
	k := newKit(t)
	parent := k.asParent("ana")
	routes := []Route{
		{http.MethodGet, "/unknown", Access(99), ok},
		{http.MethodGet, "/negative", Access(-1), ok},
	}
	for _, rt := range routes {
		for _, token := range []string{"", parent} {
			r := k.newReq(http.MethodGet, rt.Pattern, nil)
			if token != "" {
				r.Header.Set("X-Session-Token", token)
			}
			wantErr(t, k.serveRoutes(routes, r), http.StatusInternalServerError, "internal")
		}
	}
	if KidRoleFrom(httptest.NewRequest(http.MethodGet, "/", nil).Context()) != "" {
		t.Error("KidRoleFrom on a bare context must be the zero Role")
	}
}

func TestKidAccessClasses(t *testing.T) {
	k := newKit(t)
	admin, guardian, carer, stranger := k.asAdmin(), k.asParent("ana"), k.asParent("cara"), k.asParent("sam")
	kid := k.createKid(guardian, nil)
	k.grant(guardian, kid, "cara", "caregiver")
	role := func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusOK, map[string]string{"role": string(KidRoleFrom(r.Context()))})
	}
	routes := []Route{
		{http.MethodGet, "/c/{kid}", Caregiver, role},
		{http.MethodGet, "/g/{kid}", Guardian, role},
		{http.MethodGet, "/nokid", Caregiver, role},
	}
	cases := []struct {
		path, token string
		status      int
		body        string
	}{
		{"/c/" + kid, guardian, 200, `{"role":"guardian"}`},
		{"/c/" + kid, carer, 200, `{"role":"caregiver"}`},
		{"/g/" + kid, guardian, 200, `{"role":"guardian"}`},
		{"/g/" + kid, carer, 403, `{"error":"forbidden"}`},
		{"/c/" + kid, stranger, 404, `{"error":"not_found"}`},
		{"/g/" + kid, stranger, 404, `{"error":"not_found"}`},
		{"/c/no-such-kid", guardian, 404, `{"error":"not_found"}`},
		{"/nokid", guardian, 404, `{"error":"not_found"}`},
		// The admin is refused before any lookup: a real and an unknown kid answer alike.
		{"/c/" + kid, admin, 403, `{"error":"forbidden"}`},
		{"/g/no-such-kid", admin, 403, `{"error":"forbidden"}`},
		{"/c/" + kid, "", 401, `{"error":"no_session"}`},
	}
	for _, c := range cases {
		r := k.newReq(http.MethodGet, c.path, nil)
		if c.token != "" {
			r.Header.Set("X-Session-Token", c.token)
		}
		rec := k.serveRoutes(routes, r)
		if rec.Code != c.status || strings.TrimSpace(rec.Body.String()) != c.body {
			t.Errorf("%s: %d %q, want %d %s", c.path, rec.Code, rec.Body.String(), c.status, c.body)
		}
	}
}

func TestEmptyConfiguredTokenFailsClosed(t *testing.T) {
	k := newKit(t)
	d := k.deps()
	d.Cfg.InternalAPIToken = ""
	r := httptest.NewRequest(http.MethodGet, "/internal", nil)
	r.Header.Set("Authorization", "Bearer ")
	rec := httptest.NewRecorder()
	newMux(d, []Route{{http.MethodGet, "/internal", Internal, ok}}).ServeHTTP(rec, r)
	wantErr(t, rec, http.StatusUnauthorized, "unauthorized")
}

func TestUnmatchedRequests(t *testing.T) {
	k := newKit(t)
	// Without the internal token nothing about the route table is revealed.
	r := httptest.NewRequest(http.MethodGet, "/v1/nothing-here", nil)
	wantErr(t, k.serve(r), http.StatusUnauthorized, "unauthorized")
	wantErr(t, k.serve(k.newReq(http.MethodGet, "/v1/nothing-here", nil)), http.StatusNotFound, "not_found")
	// A known path with another method is the same JSON 404, not the mux's plain-text 405.
	wantErr(t, k.serve(k.newReq(http.MethodPut, "/v1/session", nil)), http.StatusNotFound, "not_found")
	wantErr(t, k.serve(httptest.NewRequest(http.MethodPut, "/v1/session", nil)), http.StatusUnauthorized, "unauthorized")
}

func TestBodyIsCappedBeforeAnyHandler(t *testing.T) {
	k := newKit(t)
	read := func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.ReadAll(r.Body); err != nil {
			var tooBig *http.MaxBytesError
			if errors.As(err, &tooBig) {
				respondError(w, http.StatusRequestEntityTooLarge, "body_too_large")
				return
			}
			respondError(w, http.StatusBadRequest, "bad_request")
			return
		}
		respondJSON(w, http.StatusOK, map[string]string{})
	}
	routes := []Route{{http.MethodPost, "/internal", Internal, read}, {http.MethodPost, "/open", Open, read}}
	for _, path := range []string{"/internal", "/open"} {
		at := k.newReq(http.MethodPost, path, strings.Repeat("a", constants.MaxBodyBytes))
		if rec := k.serveRoutes(routes, at); rec.Code != http.StatusOK {
			t.Errorf("%s at the cap: %d, want 200", path, rec.Code)
		}
		over := k.newReq(http.MethodPost, path, strings.Repeat("a", constants.MaxBodyBytes+1))
		wantErr(t, k.serveRoutes(routes, over), http.StatusRequestEntityTooLarge, "body_too_large")
	}
}

func TestStoreFailureIs500WithoutDetail(t *testing.T) {
	k := newKit(t)
	token := k.asParent("ana")
	if err := k.Store.Close(); err != nil {
		t.Fatal(err)
	}
	rec := k.do(http.MethodGet, "/v1/session", token, nil)
	wantErr(t, rec, http.StatusInternalServerError, "internal")
	if logs := k.Log.String(); strings.Contains(logs, "closed") || strings.Contains(logs, "sql") {
		t.Errorf("log carries the store error text:\n%s", logs)
	}
	if !strings.Contains(k.Log.String(), `"status":500`) {
		t.Errorf("the 500 was not logged:\n%s", k.Log.String())
	}
}

func TestLogRouteIsThePatternNotThePath(t *testing.T) {
	k := newKit(t)
	routes := []Route{{http.MethodGet, "/v1/things/{id}", Internal, ok}}
	k.serveRoutes(routes, k.newReq(http.MethodGet, "/v1/things/secret-id-123", nil))
	var line map[string]any
	if err := json.Unmarshal(k.Log.Bytes(), &line); err != nil {
		t.Fatalf("log %q: %v", k.Log.String(), err)
	}
	if line["route"] != "GET /v1/things/{id}" || line["status"] != float64(200) || line["principal_kind"] != "none" {
		t.Fatalf("log line = %v", line)
	}
	if strings.Contains(k.Log.String(), "secret-id-123") {
		t.Errorf("log contains the path value: %s", k.Log.String())
	}
}

func TestRequestIDsAreFresh(t *testing.T) {
	k := newKit(t)
	k.serve(httptest.NewRequest(http.MethodGet, "/healthz", nil))
	k.serve(httptest.NewRequest(http.MethodGet, "/healthz", nil))
	dec := json.NewDecoder(k.Log)
	var ids []string
	for dec.More() {
		var line struct {
			ID string `json:"request_id"`
		}
		if err := dec.Decode(&line); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, line.ID)
	}
	if len(ids) != 2 || ids[0] == "" || ids[0] == ids[1] {
		t.Fatalf("request ids = %v", ids)
	}
}

func TestRoutesTable(t *testing.T) {
	k := newKit(t)
	want := map[string]Access{
		"GET /healthz":       Open,
		"POST /v1/session":   Internal,
		"GET /v1/session":    Session,
		"DELETE /v1/session": Session,

		"GET /v1/kids":                            Parent,
		"POST /v1/kids":                           Parent,
		"GET /v1/kids/{kid}":                      Caregiver,
		"PATCH /v1/kids/{kid}":                    Guardian,
		"DELETE /v1/kids/{kid}":                   Guardian,
		"GET /v1/kids/{kid}/access":               Guardian,
		"PUT /v1/kids/{kid}/access/{username}":    Guardian,
		"DELETE /v1/kids/{kid}/access/{parent}":   Caregiver,
		"POST /v1/admin/parents/{parent}/disable": Admin,
	}
	got := map[string]Access{}
	for _, rt := range Routes(k.deps()) {
		if rt.Handler == nil {
			t.Errorf("%s %s has no handler", rt.Method, rt.Pattern)
		}
		got[rt.Method+" "+rt.Pattern] = rt.Access
	}
	for key, a := range want {
		if g, found := got[key]; !found || g != a {
			t.Errorf("%s: access %v (present %v), want %v", key, g, found, a)
		}
	}
}
