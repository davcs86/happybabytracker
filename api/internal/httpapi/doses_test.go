package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
)

func doseBody(medication string, over map[string]any) map[string]any {
	return withOverrides(map[string]any{"medication": medication, "given_at": ts(t0.Add(-time.Hour)), "amount": "2.5 ml", "note": "after food"}, over)
}

func (k *kit) createDose(token, kid string, body map[string]any) map[string]any {
	k.t.Helper()
	rec := k.do(http.MethodPost, "/v1/kids/"+kid+"/medication-doses", token, body)
	if rec.Code != http.StatusCreated {
		k.t.Fatalf("create dose: %d %q", rec.Code, rec.Body.String())
	}
	return bodyMap(k.t, rec)
}

func TestDoseRoutes(t *testing.T) {
	k := newKit(t)
	kid, ana, bo := k.trackerKid()
	course := k.createMedication(ana, kid, medBody(nil))["id"].(string)
	base := "/v1/kids/" + kid + "/medication-doses"

	for name, token := range map[string]string{"guardian": ana, "caregiver": bo} {
		rec := k.do(http.MethodPost, base, token, doseBody(course, nil))
		if rec.Code != http.StatusCreated {
			t.Fatalf("%s create: %d %q", name, rec.Code, rec.Body.String())
		}
		d := bodyMap(t, rec)
		sameKeys(t, d, "id", "medication", "given_at", "amount", "note")
		if d["medication"] != course || d["given_at"] != ts(t0.Add(-time.Hour)) || d["amount"] != "2.5 ml" || d["note"] != "after food" {
			t.Errorf("%s view = %v", name, d)
		}
		one := base + "/" + d["id"].(string)
		if rec := k.do(http.MethodGet, one, token, nil); rec.Code != http.StatusOK || !equalJSON(bodyMap(t, rec), d) {
			t.Errorf("%s get: %d %q", name, rec.Code, rec.Body.String())
		}
		rec = k.do(http.MethodPatch, one, token, map[string]any{"amount": nil, "note": "late", "given_at": ts(t0.Add(-2 * time.Hour))})
		if p := bodyMap(t, rec); rec.Code != http.StatusOK || p["amount"] != nil || p["note"] != "late" || p["given_at"] != ts(t0.Add(-2*time.Hour)) ||
			p["medication"] != course || p["id"] != d["id"] {
			t.Errorf("%s patch: %d %v", name, rec.Code, p)
		}
		if rec := k.do(http.MethodDelete, one, token, nil); rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
			t.Errorf("%s delete: %d %q", name, rec.Code, rec.Body.String())
		}
		wantErr(t, k.do(http.MethodGet, one, token, nil), http.StatusNotFound, "not_found")
	}
	// A body with only the course logs a dose now, with no amount and no note.
	d := k.createDose(ana, kid, map[string]any{"medication": course})
	if d["given_at"] != ts(t0) || d["amount"] != nil || d["note"] != nil {
		t.Errorf("minimal dose = %v", d)
	}
}

func TestDoseRoutesRefuseOutsiders(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	other := k.createKid(ana, map[string]any{"name": "Second"})
	xavi, admin := k.asParent("xavi"), k.asAdmin()
	base := "/v1/kids/" + kid + "/medication-doses"
	course := k.createMedication(ana, kid, medBody(nil))["id"].(string)
	otherCourse := k.createMedication(ana, other, medBody(nil))["id"].(string)
	id := k.createDose(ana, kid, doseBody(course, nil))["id"].(string)
	otherID := k.createDose(ana, other, doseBody(otherCourse, nil))["id"].(string)

	for _, c := range []struct {
		method, path string
		body         any
	}{
		{http.MethodGet, base, nil},
		{http.MethodPost, base, doseBody(course, nil)},
		{http.MethodGet, base + "/" + id, nil},
		{http.MethodPatch, base + "/" + id, map[string]any{"note": "x"}},
		{http.MethodDelete, base + "/" + id, nil},
	} {
		unknown := strings.Replace(c.path, kid, model.NewID(), 1)
		want := k.do(c.method, unknown, xavi, c.body).Body.String()
		if rec := k.do(c.method, c.path, xavi, c.body); rec.Code != http.StatusNotFound || rec.Body.String() != want {
			t.Errorf("no grant %s %s: %d %q, want %q", c.method, c.path, rec.Code, rec.Body.String(), want)
		}
		wantErr(t, k.do(c.method, c.path, admin, c.body), http.StatusForbidden, "forbidden")
	}
	// Another kid's dose answers like an unknown one, and is left alone.
	unknownDose := k.do(http.MethodGet, base+"/"+model.NewID(), ana, nil).Body.String()
	for _, method := range []string{http.MethodGet, http.MethodPatch, http.MethodDelete} {
		if rec := k.do(method, base+"/"+otherID, ana, map[string]any{"note": "x"}); rec.Code != http.StatusNotFound || rec.Body.String() != unknownDose {
			t.Errorf("%s another kid's dose: %d %q, want %q", method, rec.Code, rec.Body.String(), unknownDose)
		}
	}
	wantErr(t, k.do(http.MethodGet, base+"/not-an-id!", ana, nil), http.StatusNotFound, "not_found")
	if rec := k.do(http.MethodGet, "/v1/kids/"+other+"/medication-doses/"+otherID, ana, nil); rec.Code != http.StatusOK || bodyMap(t, rec)["note"] != "after food" {
		t.Errorf("the other kid's dose changed: %d %q", rec.Code, rec.Body.String())
	}
	// The kid's list never shows the other kid's dose, whatever the filter says.
	for _, q := range []string{"", "?medication=" + otherCourse} {
		list := bodyMap(t, k.do(http.MethodGet, base+q, ana, nil))["doses"].([]any)
		for _, e := range list {
			if e.(map[string]any)["id"] == otherID {
				t.Errorf("list %q shows another kid's dose", q)
			}
		}
	}
}

func TestDoseValidation(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	other := k.createKid(ana, map[string]any{"name": "Second"})
	course := k.createMedication(ana, kid, medBody(nil))["id"].(string)
	otherCourse := k.createMedication(ana, other, medBody(nil))["id"].(string)
	base := "/v1/kids/" + kid + "/medication-doses"
	post := func(over map[string]any) *httptest.ResponseRecorder {
		return k.do(http.MethodPost, base, ana, doseBody(course, over))
	}

	for name, c := range map[string]struct {
		over  map[string]any
		field string
	}{
		"5 min + 1 s ahead":    {map[string]any{"given_at": ts(t0.Add(constants.MaxFutureSkew + time.Second))}, "given_at"},
		"before the birth":     {map[string]any{"given_at": "2026-03-03T22:59:59Z"}, "given_at"},
		"empty amount":         {map[string]any{"amount": ""}, "amount"},
		"blank amount":         {map[string]any{"amount": "  \t "}, "amount"},
		"81 character amount":  {map[string]any{"amount": strings.Repeat("a", constants.MedicationTextMaxLen+1)}, "amount"},
		"overlong note":        {map[string]any{"note": strings.Repeat("n", constants.MaxNoteLen+1)}, "note"},
		"given_at before note": {map[string]any{"given_at": ts(t0.Add(time.Hour)), "note": strings.Repeat("n", constants.MaxNoteLen+1)}, "given_at"},
	} {
		wantInvalid2(t, name, post(c.over), c.field)
	}
	for name, body := range map[string]map[string]any{
		"another kid's course": doseBody(otherCourse, nil),
		"an unknown course":    doseBody(model.NewID(), nil),
		"a malformed id":       doseBody("not-an-id!", nil),
		"an empty id":          doseBody("", nil),
		"no medication":        doseBody("", map[string]any{"medication": nil}),
		"a null medication":    withOverrides(doseBody(course, nil), map[string]any{"medication": nil}),
	} {
		wantInvalid2(t, name, k.do(http.MethodPost, base, ana, body), "medication")
	}
	// A refused body creates nothing; the boundaries are accepted.
	if n := len(bodyMap(t, k.do(http.MethodGet, base, ana, nil))["doses"].([]any)); n != 0 {
		t.Errorf("%d doses stored by refused requests", n)
	}
	for name, over := range map[string]map[string]any{
		"exactly 5 minutes ahead": {"given_at": ts(t0.Add(constants.MaxFutureSkew))},
		"the birth instant":       {"given_at": "2026-03-03T23:00:00Z"},
		"an 80 character amount":  {"amount": strings.Repeat("é", constants.MedicationTextMaxLen)},
		"an amount to trim":       {"amount": "  3 drops  "},
		"no amount, no note":      {"amount": nil, "note": nil},
	} {
		if rec := post(over); rec.Code != http.StatusCreated {
			t.Errorf("%s: %d %q", name, rec.Code, rec.Body.String())
		}
	}
	trimmed := k.createDose(ana, kid, doseBody(course, map[string]any{"amount": "  3 drops  "}))
	if trimmed["amount"] != "3 drops" {
		t.Errorf("amount = %q, want it trimmed", trimmed["amount"])
	}
	if rec := k.do(http.MethodPost, base, ana, `{"medication":`); rec.Code != http.StatusBadRequest {
		t.Errorf("malformed JSON: %d", rec.Code)
	}
	if rec := k.do(http.MethodPost, base, ana, map[string]any{"medication": course, "bogus": 1}); rec.Code != http.StatusUnprocessableEntity ||
		strings.TrimSpace(rec.Body.String()) != `{"error":"unknown_field","field":"bogus"}` {
		t.Errorf("unknown field: %d %q", rec.Code, rec.Body.String())
	}
}

func TestDosePatchMerges(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	other := k.createKid(ana, map[string]any{"name": "Second"})
	course := k.createMedication(ana, kid, medBody(nil))["id"].(string)
	second := k.createMedication(ana, kid, medBody(map[string]any{"name": "Second course"}))["id"].(string)
	otherCourse := k.createMedication(ana, other, medBody(nil))["id"].(string)
	d := k.createDose(ana, kid, doseBody(course, nil))
	one := "/v1/kids/" + kid + "/medication-doses/" + d["id"].(string)
	get := func() map[string]any { return bodyMap(t, k.do(http.MethodGet, one, ana, nil)) }

	if rec := k.do(http.MethodPatch, one, ana, map[string]any{"note": "only"}); rec.Code != http.StatusOK {
		t.Fatalf("note-only patch: %d %q", rec.Code, rec.Body.String())
	}
	if g := get(); g["note"] != "only" || g["amount"] != "2.5 ml" || g["given_at"] != d["given_at"] || g["medication"] != course {
		t.Errorf("a note-only patch changed more: %v", g)
	}
	if rec := k.do(http.MethodPatch, one, ana, map[string]any{"medication": course, "amount": "1 ml"}); rec.Code != http.StatusOK || get()["amount"] != "1 ml" {
		t.Errorf("repeating the stored course: %d %q", rec.Code, rec.Body.String())
	}
	for name, id := range map[string]string{"another course of the kid": second, "another kid's course": otherCourse, "a malformed id": "!", "an empty id": ""} {
		wantInvalid2(t, name, k.do(http.MethodPatch, one, ana, map[string]any{"medication": id}), "medication")
	}
	if g := get(); g["medication"] != course || g["note"] != "only" || g["amount"] != "1 ml" {
		t.Errorf("a refused patch changed the dose: %v", g)
	}
	// A null medication asks for no change, like a null in any other pointer field.
	if rec := k.do(http.MethodPatch, one, ana, `{"medication":null,"note":"x"}`); rec.Code != http.StatusOK || get()["medication"] != course {
		t.Errorf("null medication: %d %q", rec.Code, rec.Body.String())
	}
	wantInvalid2(t, "future given_at", k.do(http.MethodPatch, one, ana, map[string]any{"given_at": ts(t0.Add(constants.MaxFutureSkew + time.Second))}), "given_at")
	wantInvalid2(t, "blank amount", k.do(http.MethodPatch, one, ana, map[string]any{"amount": " "}), "amount")
	if rec := k.do(http.MethodPatch, one, ana, map[string]any{"amount": nil, "note": nil}); rec.Code != http.StatusOK {
		t.Fatalf("clear: %d", rec.Code)
	}
	if g := get(); g["amount"] != nil || g["note"] != nil {
		t.Errorf("null did not clear: %v", g)
	}
}

func TestDoseList(t *testing.T) {
	k := newKit(t)
	kid, ana, bo := k.trackerKid()
	base := "/v1/kids/" + kid + "/medication-doses"
	rec := k.do(http.MethodGet, base, bo, nil)
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != `{"doses":[],"next_cursor":null}` {
		t.Fatalf("empty list: %d %q", rec.Code, rec.Body.String())
	}
	a := k.createMedication(ana, kid, medBody(nil))["id"].(string)
	b := k.createMedication(ana, kid, medBody(map[string]any{"name": "Other"}))["id"].(string)
	var ids []string
	for i, m := range []time.Duration{3, 1, 2} {
		course := a
		if i == 1 {
			course = b
		}
		ids = append(ids, k.createDose(ana, kid, doseBody(course, map[string]any{"given_at": ts(t0.Add(-m * time.Minute))}))["id"].(string))
	}
	want := []string{ids[1], ids[2], ids[0]}

	rec = k.do(http.MethodGet, base+"?limit=2", ana, nil)
	m := bodyMap(t, rec)
	sameKeys(t, m, "doses", "next_cursor")
	first := m["doses"].([]any)
	cursor, ok := m["next_cursor"].(string)
	if rec.Code != http.StatusOK || len(first) != 2 || !ok {
		t.Fatalf("page 1: %d %q", rec.Code, rec.Body.String())
	}
	rec = k.do(http.MethodGet, base+"?limit=2&cursor="+url.QueryEscape(cursor), ana, nil)
	m = bodyMap(t, rec)
	rest := m["doses"].([]any)
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
	window := "?from=" + url.QueryEscape(ts(t0.Add(-2*time.Minute))) + "&to=" + url.QueryEscape(ts(t0.Add(-time.Minute)))
	if in := bodyMap(t, k.do(http.MethodGet, base+window, ana, nil))["doses"].([]any); len(in) != 1 || in[0].(map[string]any)["id"] != ids[2] {
		t.Errorf("[from, to) = %v, want only %s", in, ids[2])
	}
	for course, wantIDs := range map[string][]string{a: {ids[2], ids[0]}, b: {ids[1]}} {
		rec = k.do(http.MethodGet, base+"?medication="+course, ana, nil)
		var got []string
		for _, e := range bodyMap(t, rec)["doses"].([]any) {
			got = append(got, e.(map[string]any)["id"].(string))
		}
		if rec.Code != http.StatusOK || !slices.Equal(got, wantIDs) {
			t.Errorf("filter %s = %d %v, want %v", course, rec.Code, got, wantIDs)
		}
	}
	if rec := k.do(http.MethodGet, base+"?medication="+model.NewID(), ana, nil); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"doses":[]`) {
		t.Errorf("filter on an unknown course: %d %q", rec.Code, rec.Body.String())
	}
	wantInvalid(t, k.do(http.MethodGet, base+"?medication=not-an-id!", ana, nil), "medication")
	wantInvalid(t, k.do(http.MethodGet, base+"?medication=", ana, nil), "medication")
	wantInvalid(t, k.do(http.MethodGet, base+"?limit=0&medication=!", ana, nil), "limit")
}

func TestDoseForAnyCourse(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	for name, over := range map[string]map[string]any{
		"inactive":    {"active": false},
		"unscheduled": {"schedule": map[string]any{"kind": "none"}},
		"every":       {"schedule": everySchedule(6)},
		"ended":       {"end_on": "2026-11-20"},
	} {
		m := k.createMedication(ana, kid, medBody(over))
		d := k.createDose(ana, kid, doseBody(m["id"].(string), nil))
		got := bodyMap(t, k.do(http.MethodGet, "/v1/kids/"+kid+"/medications/"+m["id"].(string), ana, nil))
		if got["dose_count"] != float64(1) || !equalJSON(got["last_dose"], map[string]any{"id": d["id"], "given_at": d["given_at"], "amount": "2.5 ml"}) {
			t.Errorf("%s course after a dose: %v", name, got)
		}
	}
}

func TestDoseChangesTheCourseView(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	m := k.createMedication(ana, kid, medBody(nil))
	course, courseURL := m["id"].(string), "/v1/kids/"+kid+"/medications/"+m["id"].(string)
	if m["last_dose"] != nil || m["dose_count"] != float64(0) {
		t.Fatalf("a new course = %v", m)
	}
	older := k.createDose(ana, kid, doseBody(course, map[string]any{"given_at": ts(t0.Add(-3 * time.Hour)), "amount": nil}))
	newer := k.createDose(ana, kid, doseBody(course, map[string]any{"given_at": ts(t0.Add(-time.Hour)), "amount": "5 ml"}))
	got := bodyMap(t, k.do(http.MethodGet, courseURL, ana, nil))
	if got["dose_count"] != float64(2) || !equalJSON(got["last_dose"], map[string]any{"id": newer["id"], "given_at": newer["given_at"], "amount": "5 ml"}) {
		t.Errorf("course = %v", got)
	}
	k.do(http.MethodDelete, "/v1/kids/"+kid+"/medication-doses/"+newer["id"].(string), ana, nil)
	got = bodyMap(t, k.do(http.MethodGet, courseURL, ana, nil))
	if got["dose_count"] != float64(1) || !equalJSON(got["last_dose"], map[string]any{"id": older["id"], "given_at": older["given_at"], "amount": nil}) {
		t.Errorf("course after deleting the newest = %v", got)
	}
	if list := bodyMap(t, k.do(http.MethodGet, "/v1/kids/"+kid+"/medications", ana, nil))["medications"].([]any); len(list) != 1 ||
		list[0].(map[string]any)["dose_count"] != float64(1) {
		t.Errorf("listed course = %v", list)
	}
}

func TestDeletingACourseDeletesItsDoses(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	keep := k.createMedication(ana, kid, medBody(map[string]any{"name": "Keep"}))["id"].(string)
	gone := k.createMedication(ana, kid, medBody(nil))["id"].(string)
	var goneDoses []string
	for range 3 {
		goneDoses = append(goneDoses, k.createDose(ana, kid, doseBody(gone, nil))["id"].(string))
	}
	keepDose := k.createDose(ana, kid, doseBody(keep, nil))["id"].(string)
	base := "/v1/kids/" + kid + "/medication-doses"

	if got := bodyMap(t, k.do(http.MethodGet, "/v1/kids/"+kid+"/medications/"+gone, ana, nil)); got["dose_count"] != float64(3) {
		t.Fatalf("dose_count before = %v, want 3", got["dose_count"])
	}
	if rec := k.do(http.MethodDelete, "/v1/kids/"+kid+"/medications/"+gone, ana, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("delete course: %d %q", rec.Code, rec.Body.String())
	}
	wantErr(t, k.do(http.MethodGet, "/v1/kids/"+kid+"/medications/"+gone, ana, nil), http.StatusNotFound, "not_found")
	for _, id := range goneDoses {
		wantErr(t, k.do(http.MethodGet, base+"/"+id, ana, nil), http.StatusNotFound, "not_found")
	}
	if list := bodyMap(t, k.do(http.MethodGet, base+"?medication="+gone, ana, nil))["doses"].([]any); len(list) != 0 {
		t.Errorf("doses of the deleted course = %v", list)
	}
	all := bodyMap(t, k.do(http.MethodGet, base, ana, nil))["doses"].([]any)
	if len(all) != 1 || all[0].(map[string]any)["id"] != keepDose {
		t.Errorf("doses after the cascade = %v, want only the other course's", all)
	}
	wantInvalid2(t, "dose for the deleted course", k.do(http.MethodPost, base, ana, doseBody(gone, nil)), "medication")
	if got := bodyMap(t, k.do(http.MethodGet, "/v1/kids/"+kid+"/medications/"+keep, ana, nil)); got["dose_count"] != float64(1) {
		t.Errorf("the other course's dose_count = %v", got["dose_count"])
	}
}

func TestDoseCap(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	base := "/v1/kids/" + kid + "/medication-doses"
	course := k.createMedication(ana, kid, medBody(map[string]any{"schedule": map[string]any{"kind": "none"}}))["id"].(string)
	parent := k.parentID("ana")
	for i := range constants.MaxMedicationDosesPerKid {
		_, err := k.Store.CreateDose(context.Background(), kid, parent, model.MedicationDose{MedicationID: course, GivenAt: t0.Add(-time.Duration(i) * time.Second)}, t0)
		if err != nil {
			t.Fatal(err)
		}
	}
	wantErr(t, k.do(http.MethodPost, base, ana, doseBody(course, nil)), http.StatusConflict, "limit")
	// A refused body is a 422 before the cap is looked at.
	wantInvalid2(t, "invalid at the cap", k.do(http.MethodPost, base, ana, doseBody(course, map[string]any{"amount": ""})), "amount")
}

func TestDoseMarksTheCourseNextDue(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	m := k.createMedication(ana, kid, medBody(map[string]any{"schedule": timesSchedule("08:00", "14:00")}))
	course := m["id"].(string)
	due := ts(time.Date(2026, 11, 20, 13, 0, 0, 0, time.UTC)) // 14:00 in Europe/Madrid
	if !equalJSON(m["next_due"], map[string]any{"due_at": due}) {
		t.Fatalf("next due before = %v, want %v", m["next_due"], due)
	}
	nextDue := func() any {
		return bodyMap(t, k.do(http.MethodGet, "/v1/kids/"+kid+"/medications/"+course, ana, nil))["next_due"]
	}
	k.now = t0.Add(50 * time.Minute) // 12:50Z, ten minutes before the occurrence
	// 31 minutes before: not this occurrence's.
	k.createDose(ana, kid, doseBody(course, map[string]any{"given_at": ts(time.Date(2026, 11, 20, 12, 29, 0, 0, time.UTC))}))
	if got := nextDue(); !equalJSON(got, map[string]any{"due_at": due}) {
		t.Errorf("a dose 31 minutes early moved next due to %v", got)
	}
	// 15 minutes before: it is.
	k.createDose(ana, kid, doseBody(course, map[string]any{"given_at": ts(time.Date(2026, 11, 20, 12, 45, 0, 0, time.UTC))}))
	if got, want := nextDue(), map[string]any{"due_at": ts(time.Date(2026, 11, 21, 7, 0, 0, 0, time.UTC))}; !equalJSON(got, want) {
		t.Errorf("next due after the dose = %v, want %v (08:00 the next day, Madrid)", got, want)
	}
}

func TestDoseForAnUnscheduledCourseLeavesNextDueNull(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	course := k.createMedication(ana, kid, medBody(map[string]any{"schedule": map[string]any{"kind": "none"}}))["id"].(string)
	k.createDose(ana, kid, doseBody(course, nil))
	if got := bodyMap(t, k.do(http.MethodGet, "/v1/kids/"+kid+"/medications/"+course, ana, nil)); got["next_due"] != nil || got["dose_count"] != float64(1) {
		t.Errorf("course = %v", got)
	}
}

func TestDoseOnADSTDay(t *testing.T) {
	k := newKit(t)
	ana := k.asParent("ana")
	kid := k.createKid(ana, map[string]any{"time_zone": "America/New_York"})
	k.now = time.Date(2026, 3, 7, 13, 10, 0, 0, time.UTC) // 08:10 EST
	course := k.createMedication(ana, kid, medBody(map[string]any{"schedule": timesSchedule("08:00"), "start_at": ts(k.now.Add(-time.Hour))}))
	if !equalJSON(course["next_due"], map[string]any{"due_at": "2026-03-08T12:00:00Z"}) {
		t.Fatalf("created at 08:10 with 08:00 passed: next due %v", course["next_due"])
	}
	// The 08:00 EST row is in the lookback and pending; a dose at 08:05 marks it, and 08:00 EDT (12:00Z) stays next.
	k.createDose(ana, kid, doseBody(course["id"].(string), map[string]any{"given_at": ts(k.now.Add(-5 * time.Minute))}))
	got := bodyMap(t, k.do(http.MethodGet, "/v1/kids/"+kid+"/medications/"+course["id"].(string), ana, nil))
	if !equalJSON(got["next_due"], map[string]any{"due_at": "2026-03-08T12:00:00Z"}) || got["dose_count"] != float64(1) {
		t.Errorf("course = %v", got)
	}
}

func TestDoseByADisabledCaregiver(t *testing.T) {
	k := newKit(t)
	kid, ana, bo := k.trackerKid()
	course := k.createMedication(ana, kid, medBody(nil))["id"].(string)
	base := "/v1/kids/" + kid + "/medication-doses"
	id := k.createDose(ana, kid, doseBody(course, nil))["id"].(string)
	if err := k.Store.SetParentDisabled(context.Background(), k.parentID("bob"), true, t0); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		method, path string
		body         any
	}{
		{http.MethodPost, base, doseBody(course, nil)},
		{http.MethodPatch, base + "/" + id, map[string]any{"note": "x"}},
		{http.MethodDelete, base + "/" + id, nil},
	} {
		wantErr(t, k.do(c.method, c.path, bo, c.body), http.StatusUnauthorized, "no_session") // disabling ends the sessions
	}
	list := bodyMap(t, k.do(http.MethodGet, base, ana, nil))["doses"].([]any)
	if len(list) != 1 || list[0].(map[string]any)["note"] != "after food" {
		t.Errorf("doses after the refused writes = %v", list)
	}
}

func TestDoseByARevokedCaregiver(t *testing.T) {
	k := newKit(t)
	kid, ana, bo := k.trackerKid()
	course := k.createMedication(ana, kid, medBody(nil))["id"].(string)
	base := "/v1/kids/" + kid + "/medication-doses"
	id := k.createDose(ana, kid, doseBody(course, nil))["id"].(string)
	if rec := k.do(http.MethodDelete, "/v1/kids/"+kid+"/access/"+k.parentID("bob"), ana, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("revoke: %d %q", rec.Code, rec.Body.String())
	}
	for _, c := range []struct {
		method, path string
		body         any
	}{
		{http.MethodPost, base, doseBody(course, nil)},
		{http.MethodPatch, base + "/" + id, map[string]any{"note": "x"}},
		{http.MethodDelete, base + "/" + id, nil},
	} {
		wantErr(t, k.do(c.method, c.path, bo, c.body), http.StatusNotFound, "not_found")
	}
}

func TestDoseNoValuesInLogs(t *testing.T) {
	const (
		amount = "plugh-private-amount"
		note   = "quux-private-note"
	)
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	course := k.createMedication(ana, kid, medBody(nil))["id"].(string)
	base := "/v1/kids/" + kid + "/medication-doses"
	k.Log.Reset()

	id := k.createDose(ana, kid, doseBody(course, map[string]any{"amount": amount, "note": note}))["id"].(string)
	k.do(http.MethodGet, base+"/"+id, ana, nil)
	k.do(http.MethodGet, base+"?medication="+course, ana, nil)
	k.do(http.MethodPatch, base+"/"+id, ana, map[string]any{"amount": amount + "2", "note": note + "2"})
	k.do(http.MethodPatch, base+"/"+id, ana, map[string]any{"amount": "", "note": note}) // a refusal
	k.do(http.MethodPost, base, ana, doseBody(model.NewID(), map[string]any{"amount": amount, "note": note}))
	k.do(http.MethodDelete, base+"/"+id, ana, nil)

	wantCleanLog(t, k.Log.String(), 7, "plugh", "quux", "2.5 ml")
}
