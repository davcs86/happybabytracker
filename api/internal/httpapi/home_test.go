package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/ntfy"
	"github.com/davcs86/happybabytracker/api/internal/schedule"
	"github.com/davcs86/happybabytracker/api/internal/store"
	"github.com/davcs86/happybabytracker/api/internal/units"
)

func (k *kit) home(token, kid string) map[string]any {
	k.t.Helper()
	rec := k.do(http.MethodGet, "/v1/kids/"+kid+"/home", token, nil)
	if rec.Code != http.StatusOK {
		k.t.Fatalf("home: %d %q", rec.Code, rec.Body.String())
	}
	return bodyMap(k.t, rec)
}

func TestHomeShape(t *testing.T) {
	k := newKit(t)
	kid, _, bob := k.trackerKid() // born 2026-03-04: 8 months at t0
	k.createBottle(bob, kid, nil)
	h := k.home(bob, kid)
	sameKeys(t, h, "age_months", "corrected_birth_date", "band", "modules", "running", "today", "due")
	if due, ok := h["due"].([]any); !ok || len(due) != 0 {
		t.Errorf("due of a kid with no schedule = %#v, want []", h["due"])
	}
	if h["corrected_birth_date"] != nil {
		t.Errorf("term kid corrected_birth_date = %v, want null", h["corrected_birth_date"])
	}
	if h["age_months"] != float64(8) || h["band"] != "0-11" {
		t.Errorf("age %v band %v, want 8 and 0-11", h["age_months"], h["band"])
	}
	modules := h["modules"].([]any)
	if len(modules) != len(model.Modules) {
		t.Fatalf("%d modules, want %d", len(modules), len(model.Modules))
	}
	for i, it := range modules {
		m := it.(map[string]any)
		sameKeys(t, m, "module", "enabled", "default", "overridden")
		if m["module"] != string(model.Modules[i]) {
			t.Errorf("modules[%d] = %v, want %s", i, m["module"], model.Modules[i])
		}
	}
	running := h["running"].(map[string]any)
	sameKeys(t, running, "sleep", "nursing")
	if running["sleep"] != nil || running["nursing"] != nil {
		t.Errorf("running = %v, want both null", running)
	}
	today := h["today"].(map[string]any)
	sameKeys(t, today, "date", "bottles", "nursing", "pumping", "diapers", "sleep", "meals")
	summary := k.summaryDays(bob, kid, "2026-11-20", "2026-11-20")
	if len(summary) != 1 || !equalJSON(today, summary[0]) {
		t.Errorf("today = %v, want the summary day %v", today, summary)
	}

	old := k.createKid(k.asParent("ana"), map[string]any{"birth_date": "2024-01-15"})
	if h := k.home(k.asParent("ana"), old); h["age_months"] != float64(34) || h["band"] != "24+" {
		t.Errorf("toddler age %v band %v, want 34 and 24+", h["age_months"], h["band"])
	}
	mid := k.createKid(k.asParent("ana"), map[string]any{"birth_date": "2025-06-01"})
	if h := k.home(k.asParent("ana"), mid); h["age_months"] != float64(17) || h["band"] != "12-23" {
		t.Errorf("age %v band %v, want 17 and 12-23", h["age_months"], h["band"])
	}
}

func TestHomeTodayUsesKidZone(t *testing.T) {
	k := newKit(t)
	ana := k.asParent("ana")
	kid := k.createKid(ana, map[string]any{"time_zone": "Pacific/Auckland"}) // UTC+13 in November
	vol, err := units.NewVolume(units.UnitML, 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, at := range []time.Time{
		time.Date(2026, 11, 20, 10, 30, 0, 0, time.UTC), // Auckland 2026-11-20 23:30
		time.Date(2026, 11, 20, 11, 10, 0, 0, time.UTC), // Auckland 2026-11-21 00:10
	} {
		if _, err := k.Store.CreateBottle(context.Background(), kid, k.parentID("ana"),
			model.BottleFeed{At: at, Contents: []string{model.ContentFormula}, Volume: vol}, k.now); err != nil {
			t.Fatal(err)
		}
	}
	k.now = time.Date(2026, 11, 20, 11, 30, 0, 0, time.UTC) // still the 20th in UTC
	today := k.home(ana, kid)["today"].(map[string]any)
	if today["date"] != "2026-11-21" || today["bottles"].(map[string]any)["count"] != float64(1) {
		t.Errorf("today = %v, want Auckland's 2026-11-21 with one bottle", today)
	}
}

func TestHomeShowsRunningTimers(t *testing.T) {
	k := newKit(t)
	kid, ana, bob := k.trackerKid()
	sleep := k.createSleep(ana, kid, map[string]any{"started_at": ts(t0.Add(-time.Hour))})
	nursing := k.createNursing(bob, kid, map[string]any{"started_at": ts(t0.Add(-10 * time.Minute)), "side": "left"})
	running := k.home(bob, kid)["running"].(map[string]any)
	for name, id := range map[string]string{"sleep": sleep, "nursing": nursing} {
		rec := k.do(http.MethodGet, "/v1/kids/"+kid+"/"+name+"/"+id, bob, nil)
		if rec.Code != http.StatusOK || !equalJSON(running[name], bodyMap(t, rec)) {
			t.Errorf("running %s = %v, want its view %q", name, running[name], rec.Body.String())
		}
	}
	if today := k.home(bob, kid)["today"].(map[string]any); today["sleep"].(map[string]any)["seconds"] != float64(3600) {
		t.Errorf("today sleep = %v, want the running hour", today["sleep"])
	}
}

func TestHomeReEnablesAnOffByDefaultModuleOnceAndKeepsTheGuardiansOff(t *testing.T) {
	k := newKit(t)
	ana, bob := k.asParent("ana"), k.asParent("bob")
	kid := k.createKid(ana, map[string]any{"birth_date": "2024-05-20"}) // 30 months at t0
	k.grant(ana, kid, "bob", "caregiver")
	diapers := func() map[string]any {
		h := k.home(bob, kid)
		if h["age_months"] != float64(30) || h["band"] != "24+" {
			t.Fatalf("age %v band %v, want 30 and 24+", h["age_months"], h["band"])
		}
		var items []map[string]any
		for _, it := range h["modules"].([]any) {
			items = append(items, it.(map[string]any))
		}
		return moduleItem(t, items, "diapers")
	}
	if it := diapers(); it["enabled"] != false || it["default"] != false || it["overridden"] != false {
		t.Fatalf("diapers with no entries = %v, want off by default", it)
	}
	k.createDiaper(bob, kid, nil)
	if it := diapers(); it["enabled"] != true || it["default"] != false || it["overridden"] != true {
		t.Errorf("diapers after a logged diaper = %v, want re-enabled", it)
	}
	k.setModule(ana, kid, "diapers", false)
	if it := diapers(); it["enabled"] != false || it["overridden"] != true {
		t.Errorf("diapers after the guardian's off = %v, want off", it)
	}
}

func TestHomeCorrectedBirthDate(t *testing.T) {
	k := newKit(t)
	ana := k.asParent("ana")
	// Born at 35 weeks 2 days: 33 days early, so due on 2025-07-13.
	kid := k.createKid(ana, map[string]any{"birth_date": "2025-06-10", "gestation_weeks": 35, "gestation_days": 2, "time_zone": "Pacific/Auckland"})
	for _, c := range []struct {
		name string
		now  time.Time
		want any
	}{
		{"at t0", t0, "2025-07-13"},
		{"the day before the second birthday", time.Date(2027, 6, 9, 11, 0, 0, 0, time.UTC), "2025-07-13"}, // Auckland 06-09 23:00
		{"the second birthday in the kid's zone", time.Date(2027, 6, 9, 12, 30, 0, 0, time.UTC), nil},      // Auckland 06-10 00:30
		{"after the second birthday", time.Date(2027, 9, 1, 0, 0, 0, 0, time.UTC), nil},
	} {
		k.now = c.now
		if got := k.home(k.asParent("ana"), kid)["corrected_birth_date"]; got != c.want {
			t.Errorf("%s: corrected_birth_date = %v, want %v", c.name, got, c.want)
		}
	}
}

// dueLines renders the home's due list, a vaccine entry as "<n><unit> <status> <count>", a medication entry as
// "<name> <due_at> <state>" and a visit as "visit <visit_kind> <starts_at>", after checking each entry's shape.
func dueLines(t *testing.T, h map[string]any) []string {
	t.Helper()
	list, ok := h["due"].([]any)
	if !ok {
		t.Fatalf("due = %#v, want a list", h["due"])
	}
	out := []string{}
	for _, it := range list {
		e := it.(map[string]any)
		if e["kind"] == "medication" {
			sameKeys(t, e, "kind", "medication", "due_at", "state")
			m := e["medication"].(map[string]any)
			sameKeys(t, m, "id", "name")
			out = append(out, fmt.Sprintf("%v %v %v", m["name"], e["due_at"], e["state"]))
			continue
		}
		if e["kind"] == "visit" {
			sameKeys(t, e, "kind", "id", "starts_at", "visit_kind")
			out = append(out, fmt.Sprintf("visit %v %v", e["visit_kind"], e["starts_at"]))
			continue
		}
		sameKeys(t, e, "kind", "from", "status", "count")
		sameKeys(t, e["from"].(map[string]any), "n", "unit")
		if e["kind"] != "vaccine" {
			t.Errorf("kind = %v, want vaccine", e["kind"])
		}
		from := e["from"].(map[string]any)
		out = append(out, fmt.Sprintf("%v%v %v %v", from["n"], from["unit"], e["status"], e["count"]))
	}
	return out
}

func (k *kit) wantDue(token, kid, name string, want ...string) {
	k.t.Helper()
	if want == nil {
		want = []string{}
	}
	if got := dueLines(k.t, k.home(token, kid)); !slices.Equal(got, want) {
		k.t.Errorf("%s: due = %q, want %q", name, got, want)
	}
}

// dueKid creates a kid born 2025-01-10 in UTC on schema (nil: none) with the clock at dueNow, and a token for its guardian.
func (k *kit) dueKid(schema any) (kid, ana string) {
	k.t.Helper()
	k.now = dueNow
	ana = k.asParent("ana")
	return k.createKid(ana, map[string]any{"birth_date": "2025-01-10", "time_zone": "UTC", "vaccine_schema": schema}), ana
}

// dueNow is 2025-05-12 at noon UTC: a kid born 2025-01-10 is four months and two days old.
var dueNow = time.Date(2025, 5, 12, 12, 0, 0, 0, time.UTC)

func TestHomeDue(t *testing.T) {
	k := newKit(t)
	kid, ana := k.dueKid("us")
	// us.csv at 4 months 2 days: 0d hepatitis_b-1 overdue, rsv-1 due (until 8m); 1m hepatitis_b-2 overdue; 2m seven
	// overdue, 4m seven due (default windows). Supplementary rsv-2 and covid19-1 are never listed.
	k.wantDue(ana, kid, "nothing given", "0d overdue 2", "1m overdue 1", "2m overdue 7", "4m due 7")

	k.recordDose(ana, kid, map[string]any{"given_on": "2025-01-10", "antigens": []string{"hepatitis_b", "rsv"}})
	k.wantDue(ana, kid, "the birth visit given", "1m overdue 1", "2m overdue 7", "4m due 7")

	k.recordDose(ana, kid, map[string]any{"given_on": "2025-03-12", "antigens": []string{"hepatitis_b"}})
	k.wantDue(ana, kid, "hepatitis B dose 2 given", "2m overdue 7", "4m due 7")

	visit2 := []string{"rotavirus", "diphtheria", "tetanus", "pertussis", "hib", "pneumococcus"}
	k.recordDose(ana, kid, map[string]any{"given_on": "2025-03-12", "antigens": visit2})
	k.wantDue(ana, kid, "the 2-month visit but polio given", "2m overdue 1", "4m due 7")
	k.recordDose(ana, kid, map[string]any{"given_on": "2025-03-13", "antigens": []string{"polio"}})
	k.wantDue(ana, kid, "the 2-month visit given", "4m due 7")

	if rec := k.do(http.MethodPut, "/v1/kids/"+kid+"/vaccine-dismissals/rotavirus-2", ana, map[string]any{"reason": "not_needed"}); rec.Code != http.StatusOK {
		t.Fatalf("dismiss: %d %q", rec.Code, rec.Body.String())
	}
	k.wantDue(ana, kid, "rotavirus-2 dismissed", "4m due 6")
	k.recordDose(ana, kid, map[string]any{"given_on": "2025-05-11", "antigens": []string{"diphtheria", "tetanus", "pertussis", "hib", "pneumococcus", "polio"}})
	k.wantDue(ana, kid, "the rest of the 4-month visit given")
}

func TestHomeDueIsEmptyWithoutAWorkingSchedule(t *testing.T) {
	k := newKit(t)
	kid, ana := k.dueKid("us")
	if got := dueLines(t, k.home(ana, kid)); len(got) != 4 {
		t.Fatalf("control: due = %q, want 4 groups", got)
	}
	k.setModule(ana, kid, "vaccination", false)
	k.wantDue(ana, kid, "module off")

	none, _ := k.dueKid(nil)
	k.recordDose(ana, none, map[string]any{"given_on": "2025-02-01", "antigens": []string{"polio"}})
	k.wantDue(ana, none, "no schema")

	stale, _ := k.dueKid("us")
	stored, err := k.Store.Kid(context.Background(), stale)
	if err != nil {
		t.Fatal(err)
	}
	retired := "ch"
	stored.VaccineSchema = &retired
	if err := k.Store.UpdateKid(context.Background(), stored, k.parentID("ana"), k.now); err != nil {
		t.Fatal(err)
	}
	k.wantDue(ana, stale, "a stored id no longer shipped")
}

func TestHomeDueNeverListsAvailableLines(t *testing.T) {
	k := newKit(t)
	k.now = dueNow
	ana := k.asParent("ana")
	kid := k.createKid(ana, map[string]any{"birth_date": "2010-01-10", "time_zone": "UTC", "vaccine_schema": "us"}) // 15 years
	if got := dueLines(t, k.home(ana, kid)); len(got) == 0 {
		t.Fatal("control: a 15-year-old with nothing given has overdue routine lines")
	}
	rec := k.do(http.MethodPost, "/v1/kids/"+kid+"/vaccine-dismissals", ana, map[string]any{"before": "2025-05-13", "reason": "given_unknown"})
	if rec.Code != http.StatusOK {
		t.Fatalf("bulk: %d %q", rec.Code, rec.Body.String())
	}
	v := k.vaccines(ana, kid)
	lines := allLines(v)
	for _, id := range []string{"rsv-1", "rotavirus-3", "rsv-2", "covid19-1"} {
		if lines[id]["status"] != "available" {
			t.Fatalf("%s = %v, want available", id, lines[id])
		}
	}
	k.wantDue(ana, kid, "only available lines left")
}

func TestHomeDueUsesKidZone(t *testing.T) {
	k := newKit(t)
	ana := k.asParent("ana")
	kid := k.createKid(ana, map[string]any{"birth_date": "2025-01-10", "time_zone": "Pacific/Auckland", "vaccine_schema": "us"})
	for _, c := range []struct {
		name string
		now  time.Time
		want []string
	}{
		// hepatitis_b-2 (1m, until 3m) and the 2-month lines (until 3m) turn overdue on 2025-04-10 in Auckland (UTC+12).
		{"Auckland 2025-04-09 23:00", time.Date(2025, 4, 9, 11, 0, 0, 0, time.UTC), []string{"0d overdue 2", "1m due 1", "2m due 7"}},
		{"Auckland 2025-04-10 00:00, still the 9th in UTC", time.Date(2025, 4, 9, 12, 0, 0, 0, time.UTC), []string{"0d overdue 2", "1m overdue 1", "2m overdue 7"}},
	} {
		k.now = c.now
		k.wantDue(k.asParent("ana"), kid, c.name, c.want...)
	}
}

func TestHomeDueFailedLoad(t *testing.T) {
	k := newKit(t)
	kid, ana := k.dueKid("us")
	off, _ := k.dueKid("us")
	k.setModule(ana, off, "vaccination", false)
	none, _ := k.dueKid(nil)
	saved := loadVaccines
	loadVaccines = func() error { return errors.New("vaccines: schedules unavailable") }
	t.Cleanup(func() { loadVaccines = saved })

	wantErr(t, k.do(http.MethodGet, "/v1/kids/"+kid+"/home", ana, nil), http.StatusServiceUnavailable, "unavailable")
	k.wantDue(ana, off, "module off")
	k.wantDue(ana, none, "no schema")
}

// madridDue is the trackerKid's 20:00 dose on t0's day (Europe/Madrid, UTC+1).
var madridDue = time.Date(2026, 11, 20, 19, 0, 0, 0, time.UTC)

// eveningCourse is a 20:00 course started an hour before t0; its first reminder is madridDue.
func (k *kit) eveningCourse(token, kid, name string) string {
	k.t.Helper()
	return k.createMedication(token, kid, medBody(map[string]any{"name": name, "schedule": timesSchedule("20:00")}))["id"].(string)
}

// tick moves the clock and runs the worker's store steps with the kit's channels; it returns the deliveries opened.
func (k *kit) tick(now time.Time) int {
	k.t.Helper()
	k.now = now
	ctx := context.Background()
	if err := k.Store.Materialise(ctx, now); err != nil {
		k.t.Fatal(err)
	}
	n, err := k.Store.OpenDue(ctx, now, store.Channels{Push: k.Push != nil, Ntfy: k.Cfg.NtfyEnabled})
	if err != nil {
		k.t.Fatal(err)
	}
	if err := k.Store.Expire(ctx, now); err != nil {
		k.t.Fatal(err)
	}
	return n
}

func (k *kit) setOverdueMinutes(token, kid string, minutes int) {
	k.t.Helper()
	if rec := k.do(http.MethodPatch, "/v1/kids/"+kid, token, map[string]any{"reminder_overdue_minutes": minutes}); rec.Code != http.StatusOK {
		k.t.Fatalf("overdue minutes: %d %q", rec.Code, rec.Body.String())
	}
}

func medLine(name string, due time.Time, state string) string {
	return name + " " + ts(due) + " " + state
}

func TestHomeDueMedicationShape(t *testing.T) {
	k := newKit(t)
	kid, _, bob := k.trackerKid()
	course := k.eveningCourse(bob, kid, "Amoxicillin")
	due := k.home(bob, kid)["due"].([]any)
	if len(due) != 1 {
		t.Fatalf("due = %v, want one medication entry", due)
	}
	e := due[0].(map[string]any)
	sameKeys(t, e, "kind", "medication", "due_at", "state")
	m := e["medication"].(map[string]any)
	sameKeys(t, m, "id", "name")
	if e["kind"] != "medication" || m["id"] != course || m["name"] != "Amoxicillin" || e["due_at"] != ts(madridDue) || e["state"] != "due" {
		t.Errorf("entry = %v", e)
	}
}

func TestHomeDueMedicationWindow(t *testing.T) {
	k := newKit(t)
	kid, ana, bob := k.trackerKid()
	k.eveningCourse(bob, kid, "Amoxicillin")
	k.wantDue(bob, kid, "before its time", medLine("Amoxicillin", madridDue, "due"))
	k.tick(madridDue)
	k.wantDue(bob, kid, "at its time", medLine("Amoxicillin", madridDue, "overdue"))
	k.tick(madridDue.Add(59 * time.Minute))
	k.wantDue(bob, kid, "default 60, due + 59 min", medLine("Amoxicillin", madridDue, "overdue"))
	k.tick(madridDue.Add(60 * time.Minute))
	k.wantDue(bob, kid, "default 60, due + 60 min")

	k.setOverdueMinutes(ana, kid, 5)
	k.tick(madridDue.Add(4 * time.Minute))
	k.wantDue(bob, kid, "5, due + 4 min", medLine("Amoxicillin", madridDue, "overdue"))
	k.tick(madridDue.Add(5 * time.Minute))
	k.wantDue(bob, kid, "5, due + 5 min")
}

func TestHomeDueMedicationOneOverduePerCourse(t *testing.T) {
	k := newKit(t)
	kid, ana, bob := k.trackerKid()
	k.setOverdueMinutes(ana, kid, constants.MaxReminderOverdueMinutes)
	k.createMedication(bob, kid, medBody(map[string]any{"schedule": everySchedule(1), "start_at": ts(t0.Add(-10 * time.Minute))}))
	first := t0.Add(50 * time.Minute)
	k.tick(first.Add(30 * time.Minute))
	k.wantDue(bob, kid, "the first missed", medLine("Amoxicillin", first, "overdue"), medLine("Amoxicillin", first.Add(time.Hour), "due"))
	k.tick(first.Add(time.Hour))
	k.wantDue(bob, kid, "the next one due", medLine("Amoxicillin", first.Add(time.Hour), "overdue"), medLine("Amoxicillin", first.Add(2*time.Hour), "due"))
	k.tick(first.Add(3*time.Hour + 10*time.Minute))
	k.wantDue(bob, kid, "hours later", medLine("Amoxicillin", first.Add(3*time.Hour), "overdue"), medLine("Amoxicillin", first.Add(4*time.Hour), "due"))
}

func TestHomeDueMedicationStates(t *testing.T) {
	k := newKit(t)
	kid, ana, bob := k.trackerKid()
	course := k.eveningCourse(bob, kid, "Amoxicillin")
	k.tick(madridDue.Add(40 * time.Minute))
	if got := k.dueEntries(kid); len(got) != 1 || got[0].State != "expired" {
		t.Fatalf("stored = %+v, want one expired reminder", got)
	}
	k.wantDue(bob, kid, "expired, within its window", medLine("Amoxicillin", madridDue, "overdue"))
	k.createDose(bob, kid, doseBody(course, map[string]any{"given_at": ts(k.now)}))
	k.wantDue(bob, kid, "done")

	other := k.eveningCourse(bob, kid, "Ibuprofen")
	k.now = madridDue.Add(24*time.Hour - 30*time.Minute)
	k.wantDue(bob, kid, "the next day, before its time", medLine("Amoxicillin", madridDue.Add(24*time.Hour), "due"),
		medLine("Ibuprofen", madridDue.Add(24*time.Hour), "due"))
	k.setModule(ana, kid, "medication", false)
	k.tick(madridDue.Add(24*time.Hour + time.Minute)) // the worker cancels both while the module is off
	k.wantDue(bob, kid, "module off")
	k.setModule(ana, kid, "medication", true)
	k.wantDue(bob, kid, "cancelled")
	if rec := k.do(http.MethodPatch, "/v1/kids/"+kid+"/medications/"+other, bob, map[string]any{"active": false}); rec.Code != http.StatusOK {
		t.Fatalf("deactivate: %d %q", rec.Code, rec.Body.String())
	}
	k.tick(madridDue.Add(48*time.Hour - 30*time.Minute))
	k.wantDue(bob, kid, "inactive course", medLine("Amoxicillin", madridDue.Add(48*time.Hour), "due"))
}

func TestHomeDueMedicationBeforeVaccines(t *testing.T) {
	k := newKit(t)
	kid, ana := k.dueKid("us")
	vaccines := []string{"0d overdue 2", "1m overdue 1", "2m overdue 7", "4m due 7"}
	evening := time.Date(2025, 5, 12, 18, 0, 0, 0, time.UTC)
	late := k.createMedication(ana, kid, medBody(map[string]any{"name": "Late", "schedule": timesSchedule("19:00"), "start_at": ts(dueNow)}))["id"].(string)
	var same []string
	for _, name := range []string{"Same A", "Same B"} {
		same = append(same, k.createMedication(ana, kid, medBody(map[string]any{"name": name, "schedule": timesSchedule("18:00"), "start_at": ts(dueNow)}))["id"].(string))
	}
	ids := map[string]string{same[0]: "Same A", same[1]: "Same B"}
	slices.Sort(same)
	want := append([]string{medLine(ids[same[0]], evening, "due"), medLine(ids[same[1]], evening, "due"), medLine("Late", evening.Add(time.Hour), "due")}, vaccines...)
	k.wantDue(ana, kid, "by due_at then id, before vaccines", want...)
	if due := k.home(ana, kid)["due"].([]any); len(due) < 3 || !equalJSON(due[2].(map[string]any)["medication"], map[string]any{"id": late, "name": "Late"}) {
		t.Errorf("due = %v, want the late course third", due)
	}

	k.setModule(ana, kid, "medication", false)
	k.wantDue(ana, kid, "medication off", vaccines...)
	k.setModule(ana, kid, "medication", true)
	k.setModule(ana, kid, "vaccination", false)
	k.wantDue(ana, kid, "vaccination off", want[:3]...)
}

func TestHomeDueMedicationOfAnotherKid(t *testing.T) {
	k := newKit(t)
	kid, ana, bob := k.trackerKid()
	sibling := k.createKid(ana, nil)
	k.eveningCourse(ana, sibling, "Amoxicillin")
	k.wantDue(ana, sibling, "control", medLine("Amoxicillin", madridDue, "due"))
	k.wantDue(ana, kid, "the other kid")
	k.wantDue(bob, kid, "the other kid's caregiver")
}

func TestHomeDueMedicationWithoutChannels(t *testing.T) {
	lines := func(k *kit) ([]string, int) {
		ctx := context.Background()
		kid, _, bob := k.trackerKid()
		k.eveningCourse(bob, kid, "Amoxicillin")
		p, a := testSubscriptionKeys()
		dev, _, err := k.Store.UpsertDevice(ctx, k.parentID("bob"), testEndpoint, p, a, "Phone", k.now)
		if err != nil {
			t.Fatal(err)
		}
		if err := k.Store.SetPushOptin(ctx, kid, k.parentID("bob"), dev.ID); err != nil {
			t.Fatal(err)
		}
		if _, _, err := k.Store.SetNtfyTopic(ctx, k.parentID("bob"), "hbt-test", false, k.now); err != nil {
			t.Fatal(err)
		}
		if err := k.Store.SetNtfyOptin(ctx, kid, k.parentID("bob")); err != nil {
			t.Fatal(err)
		}
		opened := k.tick(madridDue)
		k.tick(madridDue.Add(10 * time.Minute))
		return dueLines(t, k.home(bob, kid)), opened
	}
	off := newKitNoPush(t)
	off.Cfg.Ntfy, off.Cfg.NtfyEnabled = ntfy.Config{}, false
	off.Handler = NewHandler(off.deps())
	got, none := lines(off)
	want, both := lines(newKit(t))
	if none != 0 || both != 2 {
		t.Errorf("deliveries opened = %d without channels and %d with both, want 0 and 2", none, both)
	}
	if !slices.Equal(got, want) || !slices.Equal(want, []string{medLine("Amoxicillin", madridDue, "overdue")}) {
		t.Errorf("no channel: due = %q, with both channels %q", got, want)
	}
}

// dueEntries reads the store's Due rows of a Europe/Madrid kid at the kit's clock with the default window.
func (k *kit) dueEntries(kid string) []store.DueEntry {
	k.t.Helper()
	loc, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		k.t.Fatal(err)
	}
	got, err := k.Store.DueEntries(context.Background(), kid, constants.DefaultReminderOverdueMinutes*time.Minute, k.now, schedule.DayEnd(loc, k.now))
	if err != nil {
		k.t.Fatal(err)
	}
	return got
}

func TestHomeDueMedicationOneOverdueAfterAScheduleEdit(t *testing.T) {
	k := newKit(t)
	kid, ana, bob := k.trackerKid()
	k.setOverdueMinutes(ana, kid, constants.MaxReminderOverdueMinutes)
	course := k.createMedication(bob, kid, medBody(map[string]any{"schedule": everySchedule(1), "start_at": ts(t0.Add(-10 * time.Minute))}))["id"].(string)
	first := t0.Add(50 * time.Minute)
	for _, at := range []time.Duration{30 * time.Minute, 90 * time.Minute, 155 * time.Minute} {
		k.tick(first.Add(at))
	}
	// Rows at 12:50, 13:50 and 14:50 are expired; the edit keeps them, and each one's next occurrence is now 17:50.
	if rec := k.do(http.MethodPatch, "/v1/kids/"+kid+"/medications/"+course, bob, map[string]any{"schedule": everySchedule(6)}); rec.Code != http.StatusOK {
		t.Fatalf("edit: %d %q", rec.Code, rec.Body.String())
	}
	k.wantDue(bob, kid, "after the edit", medLine("Amoxicillin", first.Add(2*time.Hour), "overdue"), medLine("Amoxicillin", first.Add(5*time.Hour), "due"))
}

func TestHomeDueMedicationExpiredOfAnInactiveCourse(t *testing.T) {
	k := newKit(t)
	kid, _, bob := k.trackerKid()
	course := k.eveningCourse(bob, kid, "Amoxicillin")
	k.tick(madridDue.Add(40 * time.Minute))
	k.wantDue(bob, kid, "control: expired, within its window", medLine("Amoxicillin", madridDue, "overdue"))
	if rec := k.do(http.MethodPatch, "/v1/kids/"+kid+"/medications/"+course, bob, map[string]any{"active": false}); rec.Code != http.StatusOK {
		t.Fatalf("deactivate: %d %q", rec.Code, rec.Body.String())
	}
	k.wantDue(bob, kid, "inactive course")
}

func TestHomeDueVisitWindowIsKidLocal(t *testing.T) {
	k := newKit(t)
	kid, ana, bo := k.trackerKid()
	// t0 is Friday 2026-11-20 13:00 in Madrid (UTC+1); the window ends at the start of Saturday 2026-11-28 there, 23:00Z.
	last := time.Date(2026, 11, 27, 22, 59, 0, 0, time.UTC)
	k.createVisit(ana, kid, visitBody(map[string]any{"starts_at": ts(last), "kind": "dental"}))
	k.createVisit(ana, kid, visitBody(map[string]any{"starts_at": ts(last.Add(time.Minute))}))
	k.createVisit(ana, kid, visitBody(map[string]any{"starts_at": ts(t0.Add(-time.Minute))}))
	k.createVisit(ana, kid, visitBody(map[string]any{"starts_at": ts(t0.Add(time.Hour)), "kind": "sick"}))
	cancelled := k.createVisit(ana, kid, visitBody(map[string]any{"starts_at": ts(t0.Add(2 * time.Hour))}))
	k.do(http.MethodPatch, "/v1/kids/"+kid+"/visits/"+cancelled["id"].(string), ana, map[string]any{"cancelled": true})
	want := []string{"visit sick " + ts(t0.Add(time.Hour)), "visit dental " + ts(last)}
	k.wantDue(bo, kid, "a caregiver", want...)
	if body := k.do(http.MethodGet, "/v1/kids/"+kid+"/home", ana, nil).Body.String(); strings.Contains(body, "Dr. Lee") ||
		strings.Contains(body, "4-month check") {
		t.Errorf("the home names a provider or a reason: %s", body)
	}
	k.setModule(ana, kid, "visits", false)
	k.wantDue(ana, kid, "with the module off")
}
