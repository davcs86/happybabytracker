package httpapi

import (
	"bytes"
	"errors"
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

var visitKeys = []string{"id", "starts_at", "duration_min", "kind", "status", "cancelled_at", "provider", "location", "provider_npi",
	"note", "outcome_notes", "revision", "linked"}

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
		{"location", map[string]any{"location": long(constants.VisitLocationMaxLen + 1), "provider_npi": "1234567890"}},
		{"provider_npi", map[string]any{"provider_npi": "1234567890", "note": ""}},
		{"provider_npi", map[string]any{"provider_npi": "123456789"}},
		{"provider_npi", map[string]any{"provider_npi": "1234567893", "provider": nil}},
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
	if got, _ = listIDs("?order=asc&from=" + url.QueryEscape(ts(t0.Add(36*time.Hour)))); len(got) != 1 || got[0] != ids[3] {
		t.Errorf("from a day and a half ahead, no to: %v", got)
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

func (k *kit) pastVisit(token, kid string, at time.Time) (path string, revision float64) {
	k.t.Helper()
	m := k.createVisit(token, kid, visitBody(map[string]any{"starts_at": ts(at)}))
	return "/v1/kids/" + kid + "/visits/" + m["id"].(string), m["revision"].(float64)
}

func TestOutcomeRecordsLinkedRows(t *testing.T) {
	k := newKit(t)
	kid, ana, bo := k.trackerKid()
	one, rev := k.pastVisit(bo, kid, t0.Add(-2*time.Hour))
	body := map[string]any{"revision": rev, "outcome_notes": "Healthy",
		"growth":       map[string]any{"weight": amountBody("kg", 6400), "length": amountBody("cm", 630)},
		"vaccinations": []any{map[string]any{"antigens": []string{"polio", "tetanus"}}, map[string]any{"antigens": []string{"measles"}, "lot": "A1"}},
		"medications":  []any{map[string]any{"name": "Amoxicillin", "dose": "2.5 ml", "schedule": map[string]any{"kind": "times", "times": []string{"08:00", "20:00"}}}}}
	rec := k.do(http.MethodPost, one+"/outcome", bo, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("outcome: %d %q", rec.Code, rec.Body.String())
	}
	m := bodyMap(t, rec)
	sameKeys(t, m, visitKeys...)
	linked := m["linked"].(map[string]any)
	if m["outcome_notes"] != "Healthy" || m["revision"] != rev+1 || len(linked["growth"].([]any)) != 1 ||
		len(linked["vaccinations"].([]any)) != 2 || len(linked["medications"].([]any)) != 1 {
		t.Fatalf("visit after the outcome: %v", m)
	}
	g := linked["growth"].([]any)[0].(map[string]any)
	if g["measured_on"] != "2026-11-20" {
		t.Errorf("growth date defaulted to %v, want the visit's kid-local date", g["measured_on"])
	}
	rows := bodyMap(t, k.do(http.MethodGet, "/v1/kids/"+kid+"/growth/"+g["id"].(string), ana, nil))
	if rows["visit_id"] != m["id"] {
		t.Errorf("the growth row's visit_id = %v", rows["visit_id"])
	}
	course := bodyMap(t, k.do(http.MethodGet, "/v1/kids/"+kid+"/medications/"+linked["medications"].([]any)[0].(map[string]any)["id"].(string), ana, nil))
	if course["start_at"] != ts(t0.Add(-2*time.Hour)) || course["visit_id"] != m["id"] {
		t.Errorf("the course: start %v, visit %v; want the visit's start and id", course["start_at"], course["visit_id"])
	}
	wantErr(t, k.do(http.MethodPost, one+"/outcome", bo, body), http.StatusConflict, "stale")
	if n := len(bodyMap(t, k.do(http.MethodGet, "/v1/kids/"+kid+"/vaccinations", ana, nil))["vaccinations"].([]any)); n != 2 {
		t.Errorf("vaccinations after a double submit: %d, want 2", n)
	}
	logs := k.Log.String()
	if !strings.Contains(logs, `"`+constants.LogFieldOutcomeVaccinations+`":2`) || !strings.Contains(logs, `"`+constants.LogFieldOutcomeGrowth+`":1`) ||
		!strings.Contains(logs, `"`+constants.LogFieldOutcomeMedications+`":1`) {
		t.Errorf("the outcome's counts are not on its log line")
	}
	for _, line := range strings.Split(strings.TrimSpace(logs), "\n") {
		if leaks := logLeaks(line, "Healthy", "Amoxicillin", "Dr. Lee", "12 Main St", "4-month check", "A1"); len(leaks) > 0 {
			t.Errorf("log leaks %v: %s", leaks, line)
		}
	}
}

func TestOutcomeDefaultsUseKidDate(t *testing.T) {
	k := newKit(t)
	ana := k.asParent("ana")
	kid := k.createKid(ana, map[string]any{"time_zone": "America/Los_Angeles"})
	one, rev := k.pastVisit(ana, kid, time.Date(2026, 11, 20, 6, 30, 0, 0, time.UTC))
	rec := k.do(http.MethodPost, one+"/outcome", ana, map[string]any{"revision": rev, "growth": map[string]any{"weight": amountBody("kg", 6400)},
		"vaccinations": []any{map[string]any{"antigens": []string{"polio"}}}})
	m := bodyMap(t, rec)
	linked := m["linked"].(map[string]any)
	if rec.Code != http.StatusOK || linked["growth"].([]any)[0].(map[string]any)["measured_on"] != "2026-11-19" ||
		linked["vaccinations"].([]any)[0].(map[string]any)["given_on"] != "2026-11-19" {
		t.Errorf("06:30Z in Los Angeles: %d %v, want 2026-11-19", rec.Code, linked)
	}
}

func TestOutcomeFieldPaths(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	one, rev := k.pastVisit(ana, kid, t0.Add(-time.Hour))
	long := strings.Repeat("a", constants.VaccineLotMaxLen+1)
	for _, c := range []struct {
		field  string
		module string
		block  map[string]any
		body   func(map[string]any) map[string]any
	}{
		{"growth.weight", "growth", map[string]any{"measured_on": "2026-11-20", "weight": amountBody("kg", 1)},
			func(b map[string]any) map[string]any { return map[string]any{"growth": b} }},
		{"growth.measurements", "growth", map[string]any{"measured_on": "2026-11-20"},
			func(b map[string]any) map[string]any { return map[string]any{"growth": b} }},
		{"vaccinations[1].lot", "vaccinations", map[string]any{"given_on": "2026-11-20", "antigens": []string{"polio"}, "lot": long},
			func(b map[string]any) map[string]any {
				return map[string]any{"vaccinations": []any{map[string]any{"antigens": []string{"polio"}}, b}}
			}},
		{"medications[0].schedule", "medications", map[string]any{"name": "X", "dose": "1", "schedule": map[string]any{"kind": "every", "hours": 0}},
			func(b map[string]any) map[string]any { return map[string]any{"medications": []any{b}} }},
	} {
		own := k.do(http.MethodPost, "/v1/kids/"+kid+"/"+c.module, ana, c.block)
		ownField := bodyMap(t, own)["field"]
		rec := k.do(http.MethodPost, one+"/outcome", ana, withOverrides(c.body(c.block), map[string]any{"revision": rev}))
		b := bodyMap(t, rec)
		if rec.Code != http.StatusUnprocessableEntity || b["field"] != c.field || !strings.HasSuffix(c.field, "."+ownField.(string)) {
			t.Errorf("%s: outcome %d %v, module's own POST field %v", c.field, rec.Code, b, ownField)
		}
	}
	wantInvalid(t, k.do(http.MethodPost, one+"/outcome", ana, map[string]any{"outcome_notes": "x"}), "revision")
	wantInvalid(t, k.do(http.MethodPost, one+"/outcome", ana, map[string]any{"revision": rev}), "outcome")
	many := make([]any, constants.MaxOutcomeVaccinations+1)
	for i := range many {
		many[i] = map[string]any{"antigens": []string{"polio"}}
	}
	wantInvalid(t, k.do(http.MethodPost, one+"/outcome", ana, map[string]any{"revision": rev, "vaccinations": many}), "vaccinations")
	up := k.createVisit(ana, kid, visitBody(nil))
	wantErr(t, k.do(http.MethodPost, "/v1/kids/"+kid+"/visits/"+up["id"].(string)+"/outcome", ana,
		map[string]any{"revision": 0, "outcome_notes": "x"}), http.StatusConflict, "not_past")
	wantUnknownField(t, k.do(http.MethodPost, one+"/outcome", ana, map[string]any{"revision": rev, "growth": map[string]any{"colour": "red"}}), "colour")
}

func TestOutcomeWithoutSchedules(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	one, rev := k.pastVisit(ana, kid, t0.Add(-time.Hour))
	saved := loadVaccines
	loadVaccines = func() error { return errors.New("vaccines: schedules unavailable") }
	t.Cleanup(func() { loadVaccines = saved })
	rec := k.do(http.MethodPost, one+"/outcome", ana, map[string]any{"revision": rev, "vaccinations": []any{map[string]any{"antigens": []string{"polio"}}}})
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("vaccinations without schedules: %d %q", rec.Code, rec.Body.String())
	}
	if rec := k.do(http.MethodPost, one+"/outcome", ana, map[string]any{"revision": rev, "outcome_notes": "Healthy"}); rec.Code != http.StatusOK {
		t.Errorf("notes without schedules: %d %q", rec.Code, rec.Body.String())
	}
}

func TestVisitCalendar(t *testing.T) {
	k := newKit(t)
	kid, ana, bo := k.trackerKid()
	m := k.createVisit(ana, kid, visitBody(map[string]any{"starts_at": "2026-12-03T23:30:00Z"}))
	one := "/v1/kids/" + kid + "/visits/" + m["id"].(string)
	uid := "UID:visit-" + m["id"].(string) + "@example.test\r\n"
	rec := k.do(http.MethodGet, one+"/calendar", bo, nil)
	unfold := strings.NewReplacer("\r\n ", "").Replace
	h, body := rec.Header(), unfold(rec.Body.String())
	if rec.Code != http.StatusOK || h.Get("Content-Type") != "text/calendar; charset=utf-8" || h.Get("Cache-Control") != "no-store" ||
		h.Get("Content-Disposition") != `attachment; filename="visit-2026-12-04.ics"` || h.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("generic: %d %v", rec.Code, h)
	}
	if !strings.Contains(body, uid) || !strings.Contains(body, "\r\nSUMMARY:Medical visit\r\n") || !strings.Contains(body, "\r\nSEQUENCE:0\r\n") ||
		!strings.Contains(body, "\r\nURL:https://example.test/kids/"+kid+"/visits/"+m["id"].(string)+"\r\n") ||
		!strings.Contains(body, "TRIGGER:-PT24H") || strings.Contains(body, "LOCATION") || strings.Contains(body, "Dr. Lee") || strings.Contains(body, testKidName) {
		t.Errorf("generic body %q", body)
	}
	k.do(http.MethodPatch, one, ana, map[string]any{"cancelled": true})
	full := unfold(k.do(http.MethodGet, one+"/calendar?detail=full", ana, nil).Body.String())
	if !strings.Contains(full, uid) || !strings.Contains(full, "\r\nSEQUENCE:1\r\n") || !strings.Contains(full, "\r\nSTATUS:CANCELLED\r\n") ||
		!strings.Contains(full, "SUMMARY:"+testKidName+": Well-child visit with Dr. Lee\r\n") || !strings.Contains(full, "\r\nLOCATION:12 Main St\r\n") ||
		!strings.Contains(full, "\r\nDESCRIPTION:4-month check\r\n") || strings.Contains(full, "VALARM") {
		t.Errorf("full, cancelled: %q", full)
	}
	past, rev := k.pastVisit(ana, kid, t0.Add(-time.Hour))
	k.do(http.MethodPost, past+"/outcome", ana, map[string]any{"revision": rev, "outcome_notes": "Private notes"})
	if b := unfold(k.do(http.MethodGet, past+"/calendar?detail=full", ana, nil).Body.String()); strings.Contains(b, "Private notes") {
		t.Errorf("outcome notes in the calendar file: %q", b)
	}
	wantInvalid(t, k.do(http.MethodGet, one+"/calendar?detail=all", ana, nil), "detail")
	xavi := k.asParent("xavi")
	unknown := k.do(http.MethodGet, "/v1/kids/k-unknown/visits", ana, nil)
	if rec := k.do(http.MethodGet, one+"/calendar", xavi, nil); rec.Code != http.StatusNotFound || !bytes.Equal(rec.Body.Bytes(), unknown.Body.Bytes()) {
		t.Errorf("no grant: %d %q", rec.Code, rec.Body.String())
	}
}

func TestProviderChangeClearsNPI(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	base := "/v1/kids/" + kid + "/visits"
	const npi = "9123456789"
	patch := func(one string, body map[string]any) map[string]any {
		t.Helper()
		rec := k.do(http.MethodPatch, one, ana, body)
		if rec.Code != http.StatusOK {
			t.Fatalf("patch %v: %d %q", body, rec.Code, rec.Body.String())
		}
		return bodyMap(t, rec)
	}
	m := k.createVisit(ana, kid, visitBody(map[string]any{"provider": "Avery Quill", "provider_npi": npi}))
	if m["provider_npi"] != npi {
		t.Fatalf("created provider_npi %v", m["provider_npi"])
	}
	one := base + "/" + m["id"].(string)
	if p := patch(one, map[string]any{"location": "x"}); p["provider_npi"] != npi {
		t.Errorf("a location edit: provider_npi %v, want kept", p["provider_npi"])
	}
	if p := patch(one, map[string]any{"provider": "Avery Quill"}); p["provider_npi"] != npi {
		t.Errorf("the same provider text: provider_npi %v, want kept", p["provider_npi"])
	}
	if p := patch(one, map[string]any{"provider": "Dr Q"}); p["provider_npi"] != nil || p["provider"] != "Dr Q" {
		t.Errorf("a changed provider: provider_npi %v, provider %v; want null, Dr Q", p["provider_npi"], p["provider"])
	}
	if got := bodyMap(t, k.do(http.MethodGet, one, ana, nil)); got["provider_npi"] != nil {
		t.Errorf("stored provider_npi after the change: %v", got["provider_npi"])
	}

	m = k.createVisit(ana, kid, visitBody(map[string]any{"provider": "Avery Quill", "provider_npi": npi}))
	one = base + "/" + m["id"].(string)
	if p := patch(one, map[string]any{"provider": "Dr Q", "provider_npi": npi}); p["provider_npi"] != npi || p["provider"] != "Dr Q" {
		t.Errorf("a changed provider sent with its npi: %v %v", p["provider_npi"], p["provider"])
	}
	if p := patch(one, map[string]any{"provider": nil}); p["provider_npi"] != nil || p["provider"] != nil {
		t.Errorf("provider null: provider_npi %v, provider %v; want both null", p["provider_npi"], p["provider"])
	}
	wantInvalid(t, k.do(http.MethodPatch, one, ana, map[string]any{"provider_npi": npi}), "provider_npi")
	if p := patch(one, map[string]any{"provider": "Avery Quill", "provider_npi": npi}); p["provider_npi"] != npi {
		t.Errorf("provider and npi together: %v", p["provider_npi"])
	}
	wantInvalid(t, k.do(http.MethodPatch, one, ana, map[string]any{"provider": nil, "provider_npi": npi}), "provider_npi")
	if p := patch(one, map[string]any{"provider_npi": nil}); p["provider_npi"] != nil || p["provider"] != "Avery Quill" {
		t.Errorf("provider_npi null: %v %v", p["provider_npi"], p["provider"])
	}
	rec := k.do(http.MethodPatch, one, ana, map[string]any{"provider_npi": "9123456788"})
	wantInvalid(t, rec, "provider_npi")
	if strings.ContainsAny(rec.Body.String(), "0123456789") {
		t.Errorf("the refusal carries digits: %q", rec.Body.String())
	}
	wantInvalid(t, k.do(http.MethodPost, base, ana, visitBody(map[string]any{"provider_npi": "9123456788"})), "provider_npi")
}
