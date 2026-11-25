package httpapi

import (
	"bytes"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

func visitBody(over map[string]any) map[string]any {
	return withOverrides(map[string]any{"starts_at": ts(t0.Add(48 * time.Hour)), "kind": "well_child", "provider": "Dr. Lee",
		"location": "12 Main St", "note": "4-month check"}, over)
}

func (k *kit) createVisit(token, kid string, body map[string]any) map[string]any {
	k.t.Helper()
	rec := k.do(http.MethodPost, "/v1/kids/"+kid+"/visits", token, body)
	if rec.Code != http.StatusCreated {
		k.t.Fatalf("create visit: %d %q", rec.Code, rec.Body.String())
	}
	return bodyMap(k.t, rec)
}

var visitKeys = []string{"id", "starts_at", "duration_min", "kind", "status", "cancelled_at", "provider", "location", "note",
	"outcome_notes", "revision", "linked"}

func TestVisitRoutes(t *testing.T) {
	k := newKit(t)
	kid, ana, bo := k.trackerKid()
	base := "/v1/kids/" + kid + "/visits"
	for name, token := range map[string]string{"guardian": ana, "caregiver": bo} {
		m := k.createVisit(token, kid, visitBody(nil))
		sameKeys(t, m, visitKeys...)
		if m["status"] != "upcoming" || m["duration_min"] != float64(30) || m["revision"] != float64(0) || m["cancelled_at"] != nil ||
			m["provider"] != "Dr. Lee" || m["note"] != "4-month check" {
			t.Errorf("%s created %v", name, m)
		}
		sameKeys(t, m["linked"].(map[string]any), "growth", "vaccinations", "medications")
		one := base + "/" + m["id"].(string)
		rec := k.do(http.MethodPatch, one, token, map[string]any{"provider": nil, "duration_min": 45, "cancelled": true})
		p := bodyMap(t, rec)
		if rec.Code != http.StatusOK || p["provider"] != nil || p["duration_min"] != float64(45) || p["status"] != "cancelled" ||
			p["cancelled_at"] != ts(t0) || p["revision"] != float64(1) || p["location"] != "12 Main St" {
			t.Errorf("%s patch: %d %v", name, rec.Code, p)
		}
		rec = k.do(http.MethodPatch, one, token, map[string]any{"cancelled": false})
		if p := bodyMap(t, rec); p["status"] != "upcoming" || p["cancelled_at"] != nil {
			t.Errorf("%s un-cancel: %v", name, p)
		}
		if rec := k.do(http.MethodGet, one, token, nil); rec.Code != http.StatusOK {
			t.Errorf("%s get: %d", name, rec.Code)
		}
		if rec := k.do(http.MethodDelete, one+"?linked=keep", token, nil); rec.Code != http.StatusNoContent {
			t.Errorf("%s delete: %d %q", name, rec.Code, rec.Body.String())
		}
	}
	past := k.createVisit(ana, kid, visitBody(map[string]any{"starts_at": ts(t0.Add(-time.Hour))}))
	if past["status"] != "past" {
		t.Errorf("a visit an hour ago is %v", past["status"])
	}
	rec := k.do(http.MethodPatch, base+"/"+past["id"].(string), ana, map[string]any{"outcome_notes": "Healthy"})
	if p := bodyMap(t, rec); rec.Code != http.StatusOK || p["outcome_notes"] != "Healthy" {
		t.Fatalf("notes on a past visit: %d %v", rec.Code, p)
	}
	wantErr(t, k.do(http.MethodPatch, base+"/"+past["id"].(string), ana, map[string]any{"cancelled": true}), http.StatusConflict, "has_outcome")
	up := k.createVisit(ana, kid, visitBody(nil))
	wantErr(t, k.do(http.MethodPatch, base+"/"+up["id"].(string), ana, map[string]any{"outcome_notes": "x"}), http.StatusConflict, "not_past")
	wantUnknownField(t, k.do(http.MethodPatch, base+"/"+up["id"].(string), ana, map[string]any{"colour": "red"}), "colour")
	wantUnknownField(t, k.do(http.MethodPost, base, ana, visitBody(map[string]any{"outcome_notes": "x"})), "outcome_notes")
	wantInvalid(t, k.do(http.MethodDelete, base+"/"+up["id"].(string)+"?linked=maybe", ana, nil), "linked")
}

func TestVisitValidationOrder(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	base := "/v1/kids/" + kid + "/visits"
	long := func(n int) string { return strings.Repeat("a", n) }
	for _, c := range []struct {
		field string
		over  map[string]any
	}{
		{"starts_at", map[string]any{"starts_at": nil}},
		{"starts_at", map[string]any{"starts_at": "2026-03-02T12:00:00Z"}},
		{"starts_at", map[string]any{"starts_at": ts(t0.Add(constants.VisitHorizon + time.Minute)), "kind": "nope"}},
		{"duration_min", map[string]any{"duration_min": 4, "kind": "nope"}},
		{"duration_min", map[string]any{"duration_min": 721}},
		{"kind", map[string]any{"kind": "nope", "provider": ""}},
		{"kind", map[string]any{"kind": nil}},
		{"provider", map[string]any{"provider": " ", "location": ""}},
		{"provider", map[string]any{"provider": long(constants.VisitProviderMaxLen + 1)}},
		{"location", map[string]any{"location": long(constants.VisitLocationMaxLen + 1), "note": ""}},
		{"note", map[string]any{"note": long(constants.MaxNoteLen + 1)}},
		{"note", map[string]any{"note": ""}},
	} {
		rec := k.do(http.MethodPost, base, ana, visitBody(c.over))
		if b := bodyMap(t, rec); rec.Code != http.StatusUnprocessableEntity || b["field"] != c.field {
			t.Errorf("%v: %d %v, want field %s", c.over, rec.Code, b, c.field)
		}
	}
	if m := k.createVisit(ana, kid, visitBody(map[string]any{"starts_at": ts(t0.Add(constants.VisitHorizon))})); m["status"] != "upcoming" {
		t.Errorf("a visit at the horizon: %v", m)
	}
	m := k.createVisit(ana, kid, visitBody(map[string]any{"provider": nil, "location": nil, "note": nil}))
	one := base + "/" + m["id"].(string)
	for _, c := range []struct {
		field string
		body  map[string]any
	}{
		{"outcome_notes", map[string]any{"outcome_notes": long(constants.VisitOutcomeNotesMaxLen + 1)}},
		{"outcome_notes", map[string]any{"outcome_notes": " "}},
		{"cancelled", map[string]any{"cancelled": nil}},
	} {
		rec := k.do(http.MethodPatch, one, ana, c.body)
		if b := bodyMap(t, rec); rec.Code != http.StatusUnprocessableEntity || b["field"] != c.field {
			t.Errorf("patch %v: %d %v, want field %s", c.body, rec.Code, b, c.field)
		}
	}
}

func TestVisitListOrderAndRange(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	base := "/v1/kids/" + kid + "/visits"
	var ids []string
	for i := range 4 {
		ids = append(ids, k.createVisit(ana, kid, visitBody(map[string]any{"starts_at": ts(t0.Add(time.Duration(i-1) * 24 * time.Hour))}))["id"].(string))
	}
	listIDs := func(query string) ([]string, any) {
		rec := k.do(http.MethodGet, base+query, ana, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("list %s: %d %q", query, rec.Code, rec.Body.String())
		}
		b := bodyMap(t, rec)
		var out []string
		for _, v := range b["visits"].([]any) {
			m := v.(map[string]any)
			sameKeys(t, m, visitKeys...)
			sameKeys(t, m["linked"].(map[string]any), "growth", "vaccinations", "medications")
			out = append(out, m["id"].(string))
		}
		return out, b["next_cursor"]
	}
	if got, _ := listIDs(""); len(got) != 4 || got[0] != ids[3] {
		t.Errorf("default (newest first, into the future): %v", got)
	}
	got, next := listIDs("?order=asc&limit=2&from=" + url.QueryEscape(ts(t0)))
	if len(got) != 2 || got[0] != ids[1] || got[1] != ids[2] || next == nil {
		t.Fatalf("upcoming soonest first: %v %v", got, next)
	}
	if got, _ = listIDs("?order=asc&limit=2&from=" + url.QueryEscape(ts(t0)) + "&cursor=" + url.QueryEscape(next.(string))); len(got) != 1 || got[0] != ids[3] {
		t.Errorf("upcoming page 2: %v", got)
	}
	if got, _ = listIDs("?to=" + url.QueryEscape(ts(t0))); len(got) != 1 || got[0] != ids[0] {
		t.Errorf("past: %v", got)
	}
	wantInvalid(t, k.do(http.MethodGet, base+"?order=up", ana, nil), "order")
	wantInvalid(t, k.do(http.MethodGet, base+"?to="+url.QueryEscape(ts(t0.Add(constants.VisitHorizon+time.Hour))), ana, nil), "to")
}

func TestVisitNotFoundParity(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	other := k.createKid(ana, map[string]any{"name": "Second"})
	xavi, admin := k.asParent("xavi"), k.asAdmin()
	v := k.createVisit(ana, kid, visitBody(nil))
	theirs := k.createVisit(ana, other, visitBody(nil))
	unknown := k.do(http.MethodGet, "/v1/kids/k-unknown/visits", ana, nil)
	if unknown.Code != http.StatusNotFound {
		t.Fatalf("unknown kid: %d", unknown.Code)
	}
	for _, c := range []struct {
		method, path, token string
	}{
		{http.MethodGet, "/v1/kids/" + kid + "/visits", xavi},
		{http.MethodPost, "/v1/kids/" + kid + "/visits", xavi},
		{http.MethodGet, "/v1/kids/" + kid + "/visits/" + v["id"].(string), xavi},
		{http.MethodPatch, "/v1/kids/" + kid + "/visits/" + v["id"].(string), xavi},
		{http.MethodDelete, "/v1/kids/" + kid + "/visits/" + v["id"].(string), xavi},
		{http.MethodGet, "/v1/kids/" + kid + "/visits/" + theirs["id"].(string), ana},
		{http.MethodPatch, "/v1/kids/" + kid + "/visits/" + theirs["id"].(string), ana},
		{http.MethodDelete, "/v1/kids/" + kid + "/visits/" + theirs["id"].(string), ana},
		{http.MethodGet, "/v1/kids/" + kid + "/visits/not%20an%20id", ana},
	} {
		var body any
		if c.method != http.MethodGet && c.method != http.MethodDelete {
			body = visitBody(nil)
		}
		rec := k.do(c.method, c.path, c.token, body)
		if rec.Code != http.StatusNotFound || !bytes.Equal(rec.Body.Bytes(), unknown.Body.Bytes()) {
			t.Errorf("%s %s: %d %q, want the unknown kid's 404", c.method, c.path, rec.Code, rec.Body.String())
		}
	}
	wantErr(t, k.do(http.MethodGet, "/v1/kids/"+kid+"/visits", admin, nil), http.StatusForbidden, "forbidden")
}
