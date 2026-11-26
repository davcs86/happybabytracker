package httpapi

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/schedule"
)

// The default kid is in Europe/Madrid (UTC+1 in November) and t0 is 13:00 there.
var madridEvening = ts(time.Date(2026, 11, 20, 19, 0, 0, 0, time.UTC))

func timesSchedule(times ...string) map[string]any {
	return map[string]any{"kind": "times", "times": times}
}

func everySchedule(hours any) map[string]any { return map[string]any{"kind": "every", "hours": hours} }

func medBody(over map[string]any) map[string]any {
	return withOverrides(map[string]any{"name": "Amoxicillin", "dose": "2.5 ml", "schedule": timesSchedule("20:00", "08:00"),
		"start_at": ts(t0.Add(-time.Hour)), "note": "with food"}, over)
}

func (k *kit) createMedication(token, kid string, body map[string]any) map[string]any {
	k.t.Helper()
	rec := k.do(http.MethodPost, "/v1/kids/"+kid+"/medications", token, body)
	if rec.Code != http.StatusCreated {
		k.t.Fatalf("create medication: %d %q", rec.Code, rec.Body.String())
	}
	return bodyMap(k.t, rec)
}

func TestMedicationRoutes(t *testing.T) {
	k := newKit(t)
	kid, ana, bo := k.trackerKid()
	base := "/v1/kids/" + kid + "/medications"

	for name, token := range map[string]string{"guardian": ana, "caregiver": bo} {
		rec := k.do(http.MethodPost, base, token, medBody(nil))
		if rec.Code != http.StatusCreated {
			t.Fatalf("%s create: %d %q", name, rec.Code, rec.Body.String())
		}
		m := bodyMap(t, rec)
		sameKeys(t, m, "id", "name", "dose", "schedule", "start_at", "end_on", "active", "note", "last_dose", "next_due", "dose_count", "visit_id")
		if m["name"] != "Amoxicillin" || m["dose"] != "2.5 ml" || m["start_at"] != ts(t0.Add(-time.Hour)) || m["end_on"] != nil ||
			m["active"] != true || m["note"] != "with food" || m["last_dose"] != nil || m["dose_count"] != float64(0) ||
			!equalJSON(m["schedule"], timesSchedule("08:00", "20:00")) || !equalJSON(m["next_due"], map[string]any{"due_at": madridEvening}) {
			t.Errorf("%s view = %v", name, m)
		}
		one := base + "/" + m["id"].(string)
		if rec := k.do(http.MethodGet, one, token, nil); rec.Code != http.StatusOK || !equalJSON(bodyMap(t, rec), m) {
			t.Errorf("%s get: %d %q", name, rec.Code, rec.Body.String())
		}
		rec = k.do(http.MethodPatch, one, token, map[string]any{"name": "Amoxicillin B", "note": nil, "end_on": "2026-11-30"})
		if p := bodyMap(t, rec); rec.Code != http.StatusOK || p["name"] != "Amoxicillin B" || p["note"] != nil || p["end_on"] != "2026-11-30" ||
			p["dose"] != m["dose"] || !equalJSON(p["schedule"], m["schedule"]) || p["start_at"] != m["start_at"] {
			t.Errorf("%s patch: %d %v", name, rec.Code, p)
		}
		if rec := k.do(http.MethodDelete, one, token, nil); rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
			t.Errorf("%s delete: %d %q", name, rec.Code, rec.Body.String())
		}
		wantErr(t, k.do(http.MethodGet, one, token, nil), http.StatusNotFound, "not_found")
	}
}

func TestMedicationScheduleViews(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	for name, c := range map[string]struct {
		schedule map[string]any
		want     map[string]any
	}{
		"none":  {map[string]any{"kind": "none"}, map[string]any{"kind": "none"}},
		"times": {timesSchedule("21:30", "07:05", "13:00"), timesSchedule("07:05", "13:00", "21:30")},
		"every": {everySchedule(6), everySchedule(6)},
	} {
		m := k.createMedication(ana, kid, medBody(map[string]any{"schedule": c.schedule}))
		if !equalJSON(m["schedule"], c.want) {
			t.Errorf("%s: schedule = %v, want %v", name, m["schedule"], c.want)
		}
		if name == "none" && m["next_due"] != nil {
			t.Errorf("an unscheduled course has next due %v", m["next_due"])
		}
	}
}

func TestMedicationList(t *testing.T) {
	k := newKit(t)
	kid, ana, bo := k.trackerKid()
	base := "/v1/kids/" + kid + "/medications"
	rec := k.do(http.MethodGet, base, bo, nil)
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != `{"medications":[]}` {
		t.Fatalf("empty list: %d %q", rec.Code, rec.Body.String())
	}
	older := k.createMedication(ana, kid, medBody(map[string]any{"name": "Older", "start_at": ts(t0.Add(-48 * time.Hour))}))
	newer := k.createMedication(ana, kid, medBody(map[string]any{"name": "Newer"}))
	rec = k.do(http.MethodGet, base, bo, nil)
	list, _ := bodyMap(t, rec)["medications"].([]any)
	if rec.Code != http.StatusOK || len(list) != 2 || !equalJSON(list[0], newer) || !equalJSON(list[1], older) {
		t.Errorf("list: %d %q, want [newer, older]", rec.Code, rec.Body.String())
	}
}

// Every refusal below must be what the same route answers elsewhere in the kid API.
func TestMedicationRoutesRefuseOutsiders(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	other := k.createKid(ana, map[string]any{"name": "Second"})
	xavi, admin := k.asParent("xavi"), k.asAdmin()
	base := "/v1/kids/" + kid + "/medications"
	id := k.createMedication(ana, kid, medBody(nil))["id"].(string)
	otherID := k.createMedication(ana, other, medBody(nil))["id"].(string)

	for _, c := range []struct {
		method, path string
		body         any
	}{
		{http.MethodGet, base, nil},
		{http.MethodPost, base, medBody(nil)},
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
	for _, method := range []string{http.MethodGet, http.MethodPatch, http.MethodDelete} {
		wantErr(t, k.do(method, base+"/"+otherID, ana, map[string]any{"note": "x"}), http.StatusNotFound, "not_found")
	}
	wantErr(t, k.do(http.MethodGet, base+"/not-an-id!", ana, nil), http.StatusNotFound, "not_found")
	if rec := k.do(http.MethodGet, "/v1/kids/"+other+"/medications/"+otherID, ana, nil); rec.Code != http.StatusOK || bodyMap(t, rec)["note"] != "with food" {
		t.Errorf("the other kid's course changed: %d %q", rec.Code, rec.Body.String())
	}
}

func TestMedicationValidation(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	base := "/v1/kids/" + kid + "/medications"
	id := k.createMedication(ana, kid, medBody(nil))["id"].(string)
	long := strings.Repeat("x", constants.MaxNoteLen+1)
	tooLong := strings.Repeat("x", constants.MedicationTextMaxLen+1)
	thirteen := []string{"00:00", "01:00", "02:00", "03:00", "04:00", "05:00", "06:00", "07:00", "08:00", "09:00", "10:00", "11:00", "12:00"}

	for _, c := range []struct {
		name       string
		over       map[string]any
		field      string
		createOnly bool
	}{
		{"empty name", map[string]any{"name": ""}, "name", false},
		{"blank name", map[string]any{"name": "   "}, "name", false},
		{"name of 81 characters", map[string]any{"name": tooLong}, "name", false},
		{"no name", map[string]any{"name": nil}, "name", true},
		{"empty dose", map[string]any{"dose": ""}, "dose", false},
		{"blank dose", map[string]any{"dose": " \t "}, "dose", false},
		{"dose of 81 characters", map[string]any{"dose": tooLong}, "dose", false},
		{"no dose", map[string]any{"dose": nil}, "dose", true},
		{"no schedule", map[string]any{"schedule": nil}, "schedule", true},
		{"empty schedule", map[string]any{"schedule": map[string]any{}}, "schedule", false},
		{"unknown kind", map[string]any{"schedule": map[string]any{"kind": "weekly"}}, "schedule", false},
		{"times without a list", map[string]any{"schedule": map[string]any{"kind": "times"}}, "schedule", false},
		{"times empty", map[string]any{"schedule": timesSchedule()}, "schedule", false},
		{"13 times", map[string]any{"schedule": timesSchedule(thirteen...)}, "schedule", false},
		{"duplicate time", map[string]any{"schedule": timesSchedule("08:00", "08:00")}, "schedule", false},
		{"24:00", map[string]any{"schedule": timesSchedule("24:00")}, "schedule", false},
		{"7:5", map[string]any{"schedule": timesSchedule("7:5")}, "schedule", false},
		{"7:05", map[string]any{"schedule": timesSchedule("7:05")}, "schedule", false},
		{"08:60", map[string]any{"schedule": timesSchedule("08:60")}, "schedule", false},
		{"times with hours", map[string]any{"schedule": map[string]any{"kind": "times", "times": []string{"08:00"}, "hours": 6}}, "schedule", false},
		{"every without hours", map[string]any{"schedule": map[string]any{"kind": "every"}}, "schedule", false},
		{"every 0", map[string]any{"schedule": everySchedule(0)}, "schedule", false},
		{"every 169", map[string]any{"schedule": everySchedule(constants.MaxEveryHours + 1)}, "schedule", false},
		{"every negative", map[string]any{"schedule": everySchedule(-6)}, "schedule", false},
		{"every with times", map[string]any{"schedule": map[string]any{"kind": "every", "hours": 6, "times": []string{"08:00"}}}, "schedule", false},
		{"none with hours", map[string]any{"schedule": map[string]any{"kind": "none", "hours": 6}}, "schedule", false},
		{"none with times", map[string]any{"schedule": map[string]any{"kind": "none", "times": []string{"08:00"}}}, "schedule", false},
		{"before the birth date", map[string]any{"start_at": "2026-03-03T22:59:59Z"}, "start_at", false},
		{"end_on not a date", map[string]any{"end_on": "soon"}, "end_on", false},
		{"end_on empty", map[string]any{"end_on": ""}, "end_on", false},
		{"end_on month 13", map[string]any{"end_on": "2026-13-01"}, "end_on", false},
		{"end_on 30 February", map[string]any{"end_on": "2026-02-30"}, "end_on", false},
		{"end_on unpadded", map[string]any{"end_on": "2026-1-5"}, "end_on", false},
		{"end_on with a time", map[string]any{"end_on": "2026-11-30T10:00:00Z"}, "end_on", false},
		{"end_on before the start", map[string]any{"end_on": "2026-11-19"}, "end_on", false},
		{"note over the limit", map[string]any{"note": long}, "note", false},
		// The first invalid field wins, in the order name, dose, schedule, start_at, end_on, active, note.
		{"name before dose", map[string]any{"name": "", "dose": ""}, "name", false},
		{"dose before schedule", map[string]any{"dose": "", "schedule": everySchedule(0)}, "dose", false},
		{"schedule before start_at", map[string]any{"schedule": everySchedule(0), "start_at": "2026-03-03T22:59:59Z"}, "schedule", false},
		{"start_at before end_on", map[string]any{"start_at": "2026-03-03T22:59:59Z", "end_on": "soon"}, "start_at", false},
		{"end_on before note", map[string]any{"end_on": "soon", "note": long}, "end_on", false},
	} {
		wantInvalid2(t, c.name+" (create)", k.do(http.MethodPost, base, ana, medBody(c.over)), c.field)
		if !c.createOnly {
			wantInvalid2(t, c.name+" (patch)", k.do(http.MethodPatch, base+"/"+id, ana, c.over), c.field)
		}
	}

	for name, over := range map[string]map[string]any{
		"80 characters of two bytes":  {"name": strings.Repeat("é", constants.MedicationTextMaxLen), "dose": strings.Repeat("é", constants.MedicationTextMaxLen)},
		"a padded name is trimmed":    {"name": "  Ibuprofen  "},
		"none":                        {"schedule": map[string]any{"kind": "none"}},
		"12 times":                    {"schedule": timesSchedule(thirteen[:constants.MaxScheduleTimes]...)},
		"every 1":                     {"schedule": everySchedule(constants.MinEveryHours)},
		"every 168":                   {"schedule": everySchedule(constants.MaxEveryHours)},
		"start in the future":         {"start_at": ts(t0.Add(30 * 24 * time.Hour))},
		"start at the birth moment":   {"start_at": "2026-03-03T23:00:00Z"},
		"end_on the day of the start": {"end_on": "2026-11-20"},
		"inactive":                    {"active": false},
		"a null end_on and note":      {"end_on": nil, "note": nil},
		"fractional start_at seconds": {"start_at": "2026-11-20T11:00:00.75Z"},
	} {
		if rec := k.do(http.MethodPost, base, ana, medBody(over)); rec.Code != http.StatusCreated {
			t.Errorf("%s: %d %q", name, rec.Code, rec.Body.String())
		}
	}
	trimmed := k.createMedication(ana, kid, medBody(map[string]any{"name": "  Ibuprofen  "}))
	if trimmed["name"] != "Ibuprofen" {
		t.Errorf("name = %q, want it trimmed", trimmed["name"])
	}

	for name, body := range map[string]string{
		"active as a string":     `{"name":"A","dose":"1","schedule":{"kind":"none"},"active":"yes"}`,
		"hours as a fraction":    `{"name":"A","dose":"1","schedule":{"kind":"every","hours":6.5}}`,
		"hours as a string":      `{"name":"A","dose":"1","schedule":{"kind":"every","hours":"6"}}`,
		"start_at not a time":    `{"name":"A","dose":"1","schedule":{"kind":"none"},"start_at":"tomorrow"}`,
		"schedule not an object": `{"name":"A","dose":"1","schedule":"none"}`,
	} {
		rec := k.do(http.MethodPost, base, ana, body)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: %d %q, want 400", name, rec.Code, rec.Body.String())
		}
	}
	rec := k.do(http.MethodPost, base, ana, medBody(map[string]any{"colour": "red"}))
	if rec.Code != http.StatusUnprocessableEntity || strings.TrimSpace(rec.Body.String()) != `{"error":"unknown_field","field":"colour"}` {
		t.Errorf("unknown field: %d %q", rec.Code, rec.Body.String())
	}
}

func TestMedicationPatchMerges(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	one := "/v1/kids/" + kid + "/medications/" + k.createMedication(ana, kid, medBody(map[string]any{"end_on": "2026-12-01"}))["id"].(string)
	patch := func(body map[string]any) map[string]any {
		t.Helper()
		rec := k.do(http.MethodPatch, one, ana, body)
		if rec.Code != http.StatusOK {
			t.Fatalf("patch %v: %d %q", body, rec.Code, rec.Body.String())
		}
		return bodyMap(t, rec)
	}
	same := func(m map[string]any, keys ...string) {
		t.Helper()
		before := bodyMap(t, k.do(http.MethodGet, one, ana, nil))
		for _, key := range keys {
			if !equalJSON(m[key], before[key]) {
				t.Errorf("%s = %v, the stored %v", key, m[key], before[key])
			}
		}
	}

	m := patch(map[string]any{"note": "only the note"})
	if m["note"] != "only the note" || m["name"] != "Amoxicillin" || m["end_on"] != "2026-12-01" || m["start_at"] != ts(t0.Add(-time.Hour)) ||
		!equalJSON(m["schedule"], timesSchedule("08:00", "20:00")) || m["active"] != true {
		t.Errorf("note-only patch = %v", m)
	}
	same(patch(map[string]any{}), "name", "dose", "schedule", "start_at", "end_on", "active", "note", "next_due")

	m = patch(map[string]any{"schedule": everySchedule(8)})
	if !equalJSON(m["schedule"], everySchedule(8)) {
		t.Errorf("schedule = %v, want it replaced whole", m["schedule"])
	}
	m = patch(map[string]any{"schedule": timesSchedule("09:00")})
	if !equalJSON(m["schedule"], timesSchedule("09:00")) {
		t.Errorf("schedule = %v", m["schedule"])
	}
	if m = patch(map[string]any{"end_on": nil}); m["end_on"] != nil {
		t.Errorf("end_on = %v, want cleared", m["end_on"])
	}
	if m = patch(map[string]any{"note": nil, "dose": "5 ml"}); m["note"] != nil || m["dose"] != "5 ml" {
		t.Errorf("note and dose = %v %v", m["note"], m["dose"])
	}
	if m = patch(map[string]any{"active": false}); m["active"] != false || m["next_due"] != nil {
		t.Errorf("deactivated = %v", m)
	}
	if m = patch(map[string]any{"active": true}); m["active"] != true || m["next_due"] == nil {
		t.Errorf("activated = %v", m)
	}

	// start_at moving past the stored end_on names end_on.
	patch(map[string]any{"end_on": "2026-12-01"})
	wantInvalid2(t, "start after end_on", k.do(http.MethodPatch, one, ana, map[string]any{"start_at": "2026-12-02T10:00:00Z"}), "end_on")
	if m = patch(map[string]any{"start_at": "2026-12-02T10:00:00Z", "end_on": "2026-12-03"}); m["start_at"] != "2026-12-02T10:00:00Z" {
		t.Errorf("start_at = %v", m["start_at"])
	}
}

// An unchanged start_at stays valid after the birth date moves past it; only a change is held to the birth date.
func TestMedicationKeepsAnOldStartAfterABirthCorrection(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	one := "/v1/kids/" + kid + "/medications/" + k.createMedication(ana, kid, medBody(map[string]any{"start_at": "2026-03-10T10:00:00Z"}))["id"].(string)
	if rec := k.do(http.MethodPatch, "/v1/kids/"+kid, ana, map[string]any{"birth_date": "2026-06-01"}); rec.Code != http.StatusOK {
		t.Fatalf("kid patch: %d %q", rec.Code, rec.Body.String())
	}
	rec := k.do(http.MethodPatch, one, ana, map[string]any{"note": "still editable"})
	if m := bodyMap(t, rec); rec.Code != http.StatusOK || m["note"] != "still editable" || m["start_at"] != "2026-03-10T10:00:00Z" {
		t.Errorf("note-only patch: %d %q", rec.Code, rec.Body.String())
	}
	wantInvalid2(t, "a changed start before the birth", k.do(http.MethodPatch, one, ana, map[string]any{"start_at": "2026-03-11T10:00:00Z"}), "start_at")
	if rec := k.do(http.MethodPatch, one, ana, map[string]any{"start_at": "2026-06-02T10:00:00Z"}); rec.Code != http.StatusOK {
		t.Errorf("a start after the birth: %d %q", rec.Code, rec.Body.String())
	}
}

// A stored end_on that a kid-zone change left before the start's local date is refused only when start_at or end_on changes.
func TestMedicationEndOnIsCheckedOnlyWhenChanged(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	start := "2026-11-20T21:00:00Z" // 22:00 in Madrid on the 20th, 06:00 in Tokyo on the 21st
	one := "/v1/kids/" + kid + "/medications/" + k.createMedication(ana, kid, medBody(map[string]any{"start_at": start, "end_on": "2026-11-20"}))["id"].(string)
	if rec := k.do(http.MethodPatch, "/v1/kids/"+kid, ana, map[string]any{"time_zone": "Asia/Tokyo"}); rec.Code != http.StatusOK {
		t.Fatalf("kid patch: %d %q", rec.Code, rec.Body.String())
	}
	for name, body := range map[string]map[string]any{
		"a note only":                {"note": "x"},
		"the stored end_on re-sent":  {"end_on": "2026-11-20", "note": "y"},
		"the stored start re-sent":   {"start_at": start},
		"both stored values re-sent": {"start_at": start, "end_on": "2026-11-20"},
	} {
		if rec := k.do(http.MethodPatch, one, ana, body); rec.Code != http.StatusOK {
			t.Errorf("%s: %d %q, want 200 (nothing the check covers changed)", name, rec.Code, rec.Body.String())
		}
	}
	wantInvalid2(t, "a changed start_at", k.do(http.MethodPatch, one, ana, map[string]any{"start_at": "2026-11-20T21:00:01Z"}), "end_on")
	wantInvalid2(t, "a changed end_on still before the start", k.do(http.MethodPatch, one, ana, map[string]any{"end_on": "2026-11-19"}), "end_on")
	if rec := k.do(http.MethodPatch, one, ana, map[string]any{"end_on": "2026-11-21"}); rec.Code != http.StatusOK {
		t.Errorf("a changed end_on on the start's local date: %d %q", rec.Code, rec.Body.String())
	}
	if rec := k.do(http.MethodPatch, one, ana, map[string]any{"end_on": nil}); rec.Code != http.StatusOK {
		t.Errorf("clearing end_on: %d %q", rec.Code, rec.Body.String())
	}
}

func TestMedicationNextDue(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	base := "/v1/kids/" + kid + "/medications"

	at := func(m map[string]any) any {
		if m["next_due"] == nil {
			return nil
		}
		return m["next_due"].(map[string]any)["due_at"]
	}
	if m := k.createMedication(ana, kid, medBody(nil)); at(m) != madridEvening {
		t.Errorf("times next due = %v, want %v", at(m), madridEvening)
	}
	start := t0.Add(-20 * time.Minute)
	if m := k.createMedication(ana, kid, medBody(map[string]any{"schedule": everySchedule(6), "start_at": ts(start)})); at(m) != ts(start.Add(6*time.Hour)) {
		t.Errorf("every next due = %v, want %v", at(m), ts(start.Add(6*time.Hour)))
	}
	if m := k.createMedication(ana, kid, medBody(map[string]any{"active": false})); at(m) != nil {
		t.Errorf("inactive next due = %v", at(m))
	}
	ended := medBody(map[string]any{"schedule": timesSchedule("08:00"), "start_at": ts(t0.Add(-10 * 24 * time.Hour)), "end_on": "2026-11-20"})
	if m := k.createMedication(ana, kid, ended); at(m) != nil {
		t.Errorf("a course ended today has next due %v after its last occurrence", at(m))
	}
	older := medBody(map[string]any{"start_at": ts(t0.Add(-10 * 24 * time.Hour)), "end_on": "2026-11-19"})
	if m := k.createMedication(ana, kid, older); at(m) != nil || m["end_on"] != "2026-11-19" {
		t.Errorf("an older course ended before today: next due %v, end_on %v; want null and kept", at(m), m["end_on"])
	}
	future := ts(t0.Add(3 * 24 * time.Hour))
	m := k.createMedication(ana, kid, medBody(map[string]any{"schedule": everySchedule(12), "start_at": future}))
	if at(m) != future {
		t.Errorf("a course starting later has next due %v, want its start %v", at(m), future)
	}
	list := bodyMap(t, k.do(http.MethodGet, base, ana, nil))["medications"].([]any)
	if len(list) != 6 {
		t.Errorf("list holds %d courses, want 6", len(list))
	}
}

func TestMedicationCap(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	base := "/v1/kids/" + kid + "/medications"
	parent := k.parentID("ana")
	for range constants.MaxMedicationsPerKid {
		_, err := k.Store.CreateMedication(context.Background(), kid, parent, model.Medication{Name: "M", Dose: "1", Active: true, StartAt: t0,
			Schedule: schedule.Schedule{Kind: schedule.KindNone}}, t0)
		if err != nil {
			t.Fatal(err)
		}
	}
	wantErr(t, k.do(http.MethodPost, base, ana, medBody(nil)), http.StatusConflict, "limit")
	// A refused body is a 422 before the cap is looked at.
	wantInvalid2(t, "invalid at the cap", k.do(http.MethodPost, base, ana, medBody(map[string]any{"name": ""})), "name")
}

func TestKidZoneChangeMovesCourseNextDue(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	m := k.createMedication(ana, kid, medBody(map[string]any{"schedule": timesSchedule("08:00"), "start_at": ts(t0.Add(-10 * 24 * time.Hour))}))
	one := "/v1/kids/" + kid + "/medications/" + m["id"].(string)
	if got := m["next_due"].(map[string]any)["due_at"]; got != ts(time.Date(2026, 11, 21, 7, 0, 0, 0, time.UTC)) {
		t.Fatalf("Madrid next due = %v", got)
	}
	if rec := k.do(http.MethodPatch, "/v1/kids/"+kid, ana, map[string]any{"time_zone": "America/New_York"}); rec.Code != http.StatusOK {
		t.Fatalf("kid patch: %d %q", rec.Code, rec.Body.String())
	}
	rec := k.do(http.MethodGet, one, ana, nil)
	if got := bodyMap(t, rec)["next_due"].(map[string]any)["due_at"]; rec.Code != http.StatusOK || got != ts(time.Date(2026, 11, 20, 13, 0, 0, 0, time.UTC)) {
		t.Errorf("New York next due = %v (%d), want 13:00Z today", got, rec.Code)
	}
}

func TestMedicationWithAnUnusableKidZone(t *testing.T) {
	k := newKit(t)
	ana := k.asParent("ana")
	kid, err := k.Store.CreateKid(context.Background(), model.Kid{Name: "Zed", BirthDate: "2026-01-02", Sex: "female", TimeZone: "Local",
		GrowthSource: "who", ReminderOverdueMinutes: constants.DefaultReminderOverdueMinutes}, k.parentID("ana"), t0)
	if err != nil {
		t.Fatal(err)
	}
	base := "/v1/kids/" + kid.ID + "/medications"
	wantErr(t, k.do(http.MethodPost, base, ana, medBody(nil)), http.StatusInternalServerError, "internal")
	if rec := k.do(http.MethodGet, base, ana, nil); rec.Code != http.StatusOK {
		t.Errorf("list: %d %q", rec.Code, rec.Body.String())
	}
	if n := len(bodyMap(t, k.do(http.MethodGet, base, ana, nil))["medications"].([]any)); n != 0 {
		t.Errorf("%d courses stored for a kid with no usable zone", n)
	}
}

func TestMedicationNoValuesInLogs(t *testing.T) {
	const (
		name = "plugh-private-name"
		dose = "xyzzy-private-dose"
		note = "quux-private-note"
	)
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	base := "/v1/kids/" + kid + "/medications"
	k.Log.Reset()

	id := k.createMedication(ana, kid, medBody(map[string]any{"name": name, "dose": dose, "note": note}))["id"].(string)
	k.do(http.MethodGet, base+"/"+id, ana, nil)
	k.do(http.MethodGet, base, ana, nil)
	k.do(http.MethodPatch, base+"/"+id, ana, map[string]any{"dose": dose + "2", "note": note + "2"})
	k.do(http.MethodPatch, base+"/"+id, ana, map[string]any{"name": name, "dose": "", "note": note}) // a refusal
	k.do(http.MethodDelete, base+"/"+id, ana, nil)

	wantCleanLog(t, k.Log.String(), 6, "plugh", "xyzzy", "quux", "Amoxicillin", "2.5 ml", "08:00", "20:00")
}
