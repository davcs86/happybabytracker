package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/kidtime"
	"github.com/davcs86/happybabytracker/api/internal/model"
)

// toddler is born so that at t0 (Madrid's 2026-11-20) it is exactly 24 months old; bandStart is that local midnight.
func toddler(t *testing.T, s *Store, creator string) (kid model.Kid, loc *time.Location, bandStart time.Time) {
	t.Helper()
	k, err := s.CreateKid(ctx, model.Kid{Name: "Toddler", BirthDate: "2024-11-20", Sex: "male", TimeZone: "Europe/Madrid", GrowthSource: "who"}, creator, t0)
	if err != nil {
		t.Fatal(err)
	}
	loc = mustLoc(t, "Europe/Madrid")
	return k, loc, time.Date(2026, 11, 20, 0, 0, 0, 0, loc)
}

func mustModules(t *testing.T, s *Store, kidID string, loc *time.Location, now time.Time) map[model.Module]ModuleState {
	t.Helper()
	got, err := s.Modules(ctx, kidID, loc, now)
	if err != nil {
		t.Fatal(err)
	}
	states := got.States
	if len(states) != len(model.Modules) {
		t.Fatalf("%d states, want %d", len(states), len(model.Modules))
	}
	out := map[model.Module]ModuleState{}
	for i, st := range states {
		if st.Module != model.Modules[i] {
			t.Fatalf("state %d is %s, want %s (model.Modules order)", i, st.Module, model.Modules[i])
		}
		out[st.Module] = st
	}
	return out
}

func mustSetModule(t *testing.T, s *Store, kidID, actorID string, m model.Module, enabled bool) {
	t.Helper()
	if err := s.SetModule(ctx, kidID, actorID, m, enabled, t0); err != nil {
		t.Fatalf("SetModule %s %v: %v", m, enabled, err)
	}
}

type moduleRow struct {
	enabled   bool
	updatedAt int64
}

func storedModules(t *testing.T, s *Store, kidID string) map[model.Module]moduleRow {
	t.Helper()
	rows, err := s.w.Query(`SELECT module, enabled, updated_at FROM kid_modules WHERE kid_id = ?`, kidID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[model.Module]moduleRow{}
	for rows.Next() {
		var (
			m model.Module
			r moduleRow
		)
		if err := rows.Scan(&m, &r.enabled, &r.updatedAt); err != nil {
			t.Fatal(err)
		}
		out[m] = r
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestDefaultsFollowTheTable(t *testing.T) {
	table := map[model.Module][3]bool{ // 0–11, 12–23, 24+
		"bottles": {true, true, false}, "nursing": {true, true, false}, "pumping": {true, true, false},
		"diapers": {true, true, false}, "sleep": {true, true, false}, "feeding": {true, true, false},
		"milestones": {true, true, false}, "growth": {true, true, true}, "vaccination": {true, true, true},
		"sickness": {true, true, true}, "medication": {true, true, true},
	}
	if len(table) != len(model.Modules) {
		t.Fatalf("table covers %d modules, want %d", len(table), len(model.Modules))
	}
	for _, m := range model.Modules {
		for i, b := range []kidtime.Band{kidtime.Band0to11, kidtime.Band12to23, kidtime.Band24Plus} {
			if got := DefaultEnabled(m, b); got != table[m][i] {
				t.Errorf("DefaultEnabled(%s, band %d) = %v, want %v", m, b, got, table[m][i])
			}
		}
	}

	s := openMem(t)
	ana := mustParent(t, s, "ana")
	baby := mustKid(t, s, ana.ID, "Baby") // born 2026-01-02: 10 months at t0
	madrid := mustLoc(t, "Europe/Madrid")
	for m, st := range mustModules(t, s, baby.ID, madrid, t0) {
		if !st.Enabled || !st.Default || st.Overridden {
			t.Errorf("baby %s = %+v, want on by default", m, st)
		}
	}
	old, loc, _ := toddler(t, s, ana.ID)
	for m, st := range mustModules(t, s, old.ID, loc, t0) {
		if st.Enabled != table[m][2] || st.Default != table[m][2] || st.Overridden {
			t.Errorf("24 months %s = %+v, want the 24+ default %v", m, st, table[m][2])
		}
	}
}

func TestOverrideWinsEitherWay(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	baby := mustKid(t, s, ana.ID, "Baby")
	madrid := mustLoc(t, "Europe/Madrid")
	mustSetModule(t, s, baby.ID, ana.ID, "bottles", false)
	if st := mustModules(t, s, baby.ID, madrid, t0)["bottles"]; st != (ModuleState{Module: "bottles", Enabled: false, Default: true, Overridden: true}) {
		t.Errorf("baby bottles off = %+v", st)
	}
	mustSetModule(t, s, baby.ID, ana.ID, "bottles", true)
	if st := mustModules(t, s, baby.ID, madrid, t0)["bottles"]; st != (ModuleState{Module: "bottles", Enabled: true, Default: true, Overridden: true}) {
		t.Errorf("baby bottles on again = %+v", st)
	}

	old, loc, _ := toddler(t, s, ana.ID)
	mustSetModule(t, s, old.ID, ana.ID, "diapers", true)
	mustSetModule(t, s, old.ID, ana.ID, "growth", false)
	got := mustModules(t, s, old.ID, loc, t0)
	if st := got["diapers"]; st != (ModuleState{Module: "diapers", Enabled: true, Default: false, Overridden: true}) {
		t.Errorf("24 months diapers on = %+v", st)
	}
	if st := got["growth"]; st != (ModuleState{Module: "growth", Enabled: false, Default: true, Overridden: true}) {
		t.Errorf("24 months growth off = %+v", st)
	}
}

func TestSetModuleNeedsGuardian(t *testing.T) {
	s := openMem(t)
	ana, bob, cy := mustParent(t, s, "ana"), mustParent(t, s, "bob"), mustParent(t, s, "cy")
	baby := mustKid(t, s, ana.ID, "Baby")
	mustGrant(t, s, baby.ID, ana.ID, "bob", model.RoleCaregiver)
	if err := s.SetModule(ctx, baby.ID, bob.ID, "bottles", false, t0); !errors.Is(err, ErrForbidden) {
		t.Errorf("caregiver SetModule = %v, want ErrForbidden", err)
	}
	if err := s.SetModule(ctx, baby.ID, cy.ID, "bottles", false, t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("no-grant SetModule = %v, want ErrNotFound", err)
	}
	if rows := storedModules(t, s, baby.ID); len(rows) != 0 {
		t.Errorf("refused writes stored %v", rows)
	}
	if _, err := s.Modules(ctx, "no-such-kid", time.UTC, t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("Modules of an unknown kid = %v, want ErrNotFound", err)
	}
}

func TestBandCrossingKeepsModuleWithRecentEntry(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k, loc, start := toddler(t, s, ana.ID)
	mustBottle(t, s, k.ID, ana.ID, newBottle(t, start.Add(-45*24*time.Hour), model.ContentFormula))
	mustPumping(t, s, k.ID, ana.ID, newPumping(t, start.Add(-70*24*time.Hour)))
	// A finished sleep outside the window, and a running one started outside it too: only the timer keeps it.
	end := start.Add(-70*24*time.Hour + time.Hour)
	mustSleep(t, s, k.ID, ana.ID, model.SleepSession{StartedAt: start.Add(-70 * 24 * time.Hour), EndedAt: &end})
	mustSleep(t, s, k.ID, ana.ID, model.SleepSession{StartedAt: start.Add(-70*24*time.Hour + 2*time.Hour)})

	got := mustModules(t, s, k.ID, loc, t0)
	for _, m := range []model.Module{"bottles", "sleep"} {
		if st := got[m]; st != (ModuleState{Module: m, Enabled: true, Default: false, Overridden: true}) {
			t.Errorf("%s = %+v, want kept on by an override", m, st)
		}
	}
	for _, m := range []model.Module{"pumping", "diapers", "nursing", "feeding", "milestones"} {
		if st := got[m]; st != (ModuleState{Module: m, Enabled: false, Default: false, Overridden: false}) {
			t.Errorf("%s = %+v, want off by default", m, st)
		}
	}
	rows := storedModules(t, s, k.ID)
	if len(rows) != 2 || rows["bottles"] != (moduleRow{true, t0.Unix()}) || rows["sleep"] != (moduleRow{true, t0.Unix()}) {
		t.Errorf("stored rows = %v, want bottles and sleep on at t0", rows)
	}
}

func TestBandCrossingCountsEntriesSinceTheBandStarted(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k, loc, start := toddler(t, s, ana.ID)
	mustMeal(t, s, k.ID, ana.ID, model.Meal{At: start.Add(time.Hour), What: "pear"})
	mustNursing(t, s, k.ID, ana.ID, runningNursing(start.Add(-90*24*time.Hour), "left"))
	got := mustModules(t, s, k.ID, loc, t0)
	for _, m := range []model.Module{"feeding", "nursing"} {
		if !got[m].Enabled || !got[m].Overridden {
			t.Errorf("%s = %+v, want kept on", m, got[m])
		}
	}
}

func TestBandCrossingIsIdempotentAndNotRepeatedAfterGuardianTurnsItOff(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k, loc, start := toddler(t, s, ana.ID)
	mustBottle(t, s, k.ID, ana.ID, newBottle(t, start.Add(-time.Hour), model.ContentFormula))
	mustDiaper(t, s, k.ID, ana.ID, model.Diaper{At: start.Add(-time.Hour), Kind: "wet"})

	first := mustModules(t, s, k.ID, loc, t0)
	rows := storedModules(t, s, k.ID)
	second := mustModules(t, s, k.ID, loc, t0.Add(time.Hour))
	if len(first) != len(second) {
		t.Fatal("state count changed")
	}
	for m := range first {
		if first[m] != second[m] {
			t.Errorf("%s changed on a second read: %+v then %+v", m, first[m], second[m])
		}
	}
	if again := storedModules(t, s, k.ID); len(again) != 2 || again["bottles"] != rows["bottles"] || again["diapers"] != rows["diapers"] {
		t.Errorf("second read rewrote rows: %v then %v", rows, again)
	}

	mustSetModule(t, s, k.ID, ana.ID, "bottles", false)
	mustBottle(t, s, k.ID, ana.ID, newBottle(t, t0.Add(-time.Minute), model.ContentFormula))
	if st := mustModules(t, s, k.ID, loc, t0.Add(2*time.Hour))["bottles"]; st.Enabled || !st.Overridden {
		t.Errorf("bottles after the guardian's off = %+v, want off", st)
	}
	if got := storedModules(t, s, k.ID)["bottles"]; got.enabled {
		t.Errorf("stored bottles = %+v, want the guardian's off", got)
	}
}

func TestGuardiansOffBeforeTheCrossingBlocksTheOverride(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k, loc, start := toddler(t, s, ana.ID)
	mustBottle(t, s, k.ID, ana.ID, newBottle(t, start.Add(-10*24*time.Hour), model.ContentFormula))
	before := start.Add(-24 * time.Hour)
	if st := mustModules(t, s, k.ID, loc, before)["bottles"]; !st.Enabled || st.Overridden {
		t.Fatalf("bottles at 23 months = %+v, want on by default", st)
	}
	mustSetModule(t, s, k.ID, ana.ID, "bottles", false)
	if st := mustModules(t, s, k.ID, loc, t0)["bottles"]; st.Enabled || !st.Overridden {
		t.Errorf("bottles at 24 months = %+v, want the guardian's off", st)
	}
}

func TestOffByDefaultModuleIsReEnabledByALaterEntryOnceAndAnOffIsFinal(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k, err := s.CreateKid(ctx, model.Kid{Name: "Older", BirthDate: "2024-05-20", Sex: "male", TimeZone: "Europe/Madrid", GrowthSource: "who"}, ana.ID, t0)
	if err != nil {
		t.Fatal(err)
	}
	loc := mustLoc(t, "Europe/Madrid") // 30 months at t0; the 24+ band started 2026-05-20
	off := ModuleState{Module: "diapers", Enabled: false, Default: false, Overridden: false}
	if st := mustModules(t, s, k.ID, loc, t0)["diapers"]; st != off {
		t.Fatalf("diapers with no entries = %+v, want %+v", st, off)
	}
	if rows := storedModules(t, s, k.ID); len(rows) != 0 {
		t.Fatalf("a read with no entries stored %v", rows)
	}

	mustDiaper(t, s, k.ID, ana.ID, model.Diaper{At: t0.Add(-time.Hour), Kind: "wet"})
	if st := mustModules(t, s, k.ID, loc, t0)["diapers"]; st != (ModuleState{Module: "diapers", Enabled: true, Default: false, Overridden: true}) {
		t.Errorf("diapers after a logged diaper = %+v, want re-enabled", st)
	}
	if rows := storedModules(t, s, k.ID); len(rows) != 1 || rows["diapers"] != (moduleRow{true, t0.Unix()}) {
		t.Errorf("stored rows = %v, want diapers on at t0", rows)
	}

	mustSetModule(t, s, k.ID, ana.ID, "diapers", false)
	if st := mustModules(t, s, k.ID, loc, t0.Add(time.Hour))["diapers"]; st != (ModuleState{Module: "diapers", Enabled: false, Default: false, Overridden: true}) {
		t.Errorf("diapers after the guardian's off = %+v, want off", st)
	}
	if got := storedModules(t, s, k.ID)["diapers"]; got.enabled {
		t.Errorf("stored diapers = %+v, want the guardian's off", got)
	}
}

// A read that writes no override never waits for the writer: here a held write transaction would block it.
func TestModulesWithNothingToWriteStayOnTheReadPool(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	baby := mustKid(t, s, ana.ID, "Baby")
	old, loc, _ := toddler(t, s, ana.ID)
	mustDiaper(t, s, old.ID, ana.ID, model.Diaper{At: t0.Add(-400 * 24 * time.Hour), Kind: "wet"}) // outside the window
	tx, err := s.w.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	for _, kidID := range []string{baby.ID, old.ID} {
		short, cancel := context.WithTimeout(ctx, 2*time.Second)
		got, err := s.Modules(short, kidID, loc, t0)
		cancel()
		if err != nil {
			t.Errorf("Modules with the writer busy: %v", err)
		}
		if kidID == old.ID && (got.AgeMonths != 24 || got.Band != kidtime.Band24Plus) {
			t.Errorf("toddler age = %d months, band %d; want 24, 24+", got.AgeMonths, got.Band)
		}
	}
}
