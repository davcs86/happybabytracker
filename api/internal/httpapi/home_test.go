package httpapi

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
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
	sameKeys(t, h, "age_months", "corrected_birth_date", "band", "modules", "running", "today")
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
