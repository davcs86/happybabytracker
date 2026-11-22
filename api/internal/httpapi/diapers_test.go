package httpapi

import (
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
)

func diaperBody(over map[string]any) map[string]any {
	return withOverrides(map[string]any{"at": ts(t0.Add(-time.Hour)), "kind": "wet", "note": "first"}, over)
}

func (k *kit) createDiaper(token, kid string, over map[string]any) string {
	k.t.Helper()
	rec := k.do(http.MethodPost, "/v1/kids/"+kid+"/diapers", token, diaperBody(over))
	if rec.Code != http.StatusCreated {
		k.t.Fatalf("create diaper: %d %q", rec.Code, rec.Body.String())
	}
	return bodyMap(k.t, rec)["id"].(string)
}

func TestDiaperRoutes(t *testing.T) {
	k := newKit(t)
	kid, ana, bo := k.trackerKid()
	other := k.createKid(ana, map[string]any{"name": "Second"})
	xavi, admin := k.asParent("xavi"), k.asAdmin()
	base := "/v1/kids/" + kid + "/diapers"

	for name, token := range map[string]string{"guardian": ana, "caregiver": bo} {
		rec := k.do(http.MethodPost, base, token, diaperBody(nil))
		if rec.Code != http.StatusCreated {
			t.Fatalf("%s create: %d %q", name, rec.Code, rec.Body.String())
		}
		m := bodyMap(t, rec)
		sameKeys(t, m, "id", "at", "kind", "note")
		if m["at"] != ts(t0.Add(-time.Hour)) || m["kind"] != "wet" || m["note"] != "first" {
			t.Errorf("%s view = %v", name, m)
		}
		one := base + "/" + m["id"].(string)
		if rec := k.do(http.MethodGet, one, token, nil); rec.Code != http.StatusOK || !equalJSON(bodyMap(t, rec), m) {
			t.Errorf("%s get: %d %q", name, rec.Code, rec.Body.String())
		}
		rec = k.do(http.MethodPatch, one, token, map[string]any{"kind": "both", "note": nil})
		if p := bodyMap(t, rec); rec.Code != http.StatusOK || p["kind"] != "both" || p["note"] != nil || p["at"] != m["at"] {
			t.Errorf("%s patch: %d %v", name, rec.Code, p)
		}
		if rec := k.do(http.MethodDelete, one, token, nil); rec.Code != http.StatusNoContent {
			t.Errorf("%s delete: %d %q", name, rec.Code, rec.Body.String())
		}
		wantErr(t, k.do(http.MethodGet, one, token, nil), http.StatusNotFound, "not_found")
	}

	id, otherID := k.createDiaper(ana, kid, nil), k.createDiaper(ana, other, nil)
	for _, c := range []struct {
		method, path string
		body         any
	}{
		{http.MethodGet, base, nil},
		{http.MethodPost, base, diaperBody(nil)},
		{http.MethodGet, base + "/" + id, nil},
		{http.MethodPatch, base + "/" + id, map[string]any{"kind": "dry"}},
		{http.MethodDelete, base + "/" + id, nil},
	} {
		unknown := strings.Replace(c.path, kid, model.NewID(), 1)
		want := k.do(c.method, unknown, xavi, c.body).Body.String()
		if rec := k.do(c.method, c.path, xavi, c.body); rec.Code != http.StatusNotFound || rec.Body.String() != want {
			t.Errorf("no grant %s %s: %d %q, want %q", c.method, c.path, rec.Code, rec.Body.String(), want)
		}
		wantErr(t, k.do(c.method, c.path, admin, c.body), http.StatusForbidden, "forbidden")
	}
	for _, method := range []string{http.MethodGet, http.MethodPatch, http.MethodDelete} {
		wantErr(t, k.do(method, base+"/"+otherID, ana, map[string]any{"kind": "dry"}), http.StatusNotFound, "not_found")
	}
	if rec := k.do(http.MethodGet, "/v1/kids/"+other+"/diapers/"+otherID, ana, nil); rec.Code != http.StatusOK || bodyMap(t, rec)["kind"] != "wet" {
		t.Errorf("the other kid's diaper changed: %d %q", rec.Code, rec.Body.String())
	}
}

func TestDiaperValidation(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	base := "/v1/kids/" + kid + "/diapers"
	id := k.createDiaper(ana, kid, nil)

	for _, c := range []struct {
		name       string
		over       map[string]any
		field      string
		createOnly bool
	}{
		{"future", map[string]any{"at": ts(t0.Add(constants.MaxFutureSkew + time.Second))}, "at", false},
		{"before the birth date", map[string]any{"at": "2026-03-03T22:59:59Z"}, "at", false},
		{"unknown kind", map[string]any{"kind": "green"}, "kind", false},
		{"empty kind", map[string]any{"kind": ""}, "kind", false},
		{"no kind", map[string]any{"kind": nil}, "kind", true},
		{"note over the limit", map[string]any{"note": strings.Repeat("x", constants.MaxNoteLen+1)}, "note", false},
	} {
		wantInvalid2(t, c.name+" (create)", k.do(http.MethodPost, base, ana, diaperBody(c.over)), c.field)
		if !c.createOnly {
			wantInvalid2(t, c.name+" (patch)", k.do(http.MethodPatch, base+"/"+id, ana, c.over), c.field)
		}
	}
	for _, kind := range model.DiaperKinds {
		if rec := k.do(http.MethodPost, base, ana, diaperBody(map[string]any{"kind": kind})); rec.Code != http.StatusCreated {
			t.Errorf("kind %s: %d %q", kind, rec.Code, rec.Body.String())
		}
	}
	rec := k.do(http.MethodPost, base, ana, diaperBody(map[string]any{"colour": "red"}))
	if rec.Code != http.StatusUnprocessableEntity || strings.TrimSpace(rec.Body.String()) != `{"error":"unknown_field","field":"colour"}` {
		t.Errorf("unknown field: %d %q", rec.Code, rec.Body.String())
	}
}

func TestDiaperList(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	base := "/v1/kids/" + kid + "/diapers"
	var ids []string
	for _, m := range []time.Duration{3, 1, 2} {
		ids = append(ids, k.createDiaper(ana, kid, map[string]any{"at": ts(t0.Add(-m * time.Minute))}))
	}
	want := []string{ids[1], ids[2], ids[0]}

	rec := k.do(http.MethodGet, base+"?limit=2", ana, nil)
	m := bodyMap(t, rec)
	sameKeys(t, m, "diapers", "next_cursor")
	first := m["diapers"].([]any)
	cursor, ok := m["next_cursor"].(string)
	if rec.Code != http.StatusOK || len(first) != 2 || !ok {
		t.Fatalf("page 1: %d %q", rec.Code, rec.Body.String())
	}
	rec = k.do(http.MethodGet, base+"?limit=2&cursor="+url.QueryEscape(cursor), ana, nil)
	m = bodyMap(t, rec)
	rest := m["diapers"].([]any)
	if rec.Code != http.StatusOK || len(rest) != 1 || m["next_cursor"] != nil {
		t.Fatalf("page 2: %d %q", rec.Code, rec.Body.String())
	}
	var got []string
	for _, e := range append(first, rest...) {
		got = append(got, e.(map[string]any)["id"].(string))
	}
	if !slices.Equal(got, want) {
		t.Errorf("paged = %v, want %v", got, want)
	}
	wantInvalid(t, k.do(http.MethodGet, base+"?limit=0", ana, nil), "limit")
}
