package store

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
)

var wantTables = []string{"meta", "parents", "identities", "sessions", "one_time_tokens", "kids", "kid_access",
	"bottle_feeds", "diapers", "meals", "pumping_sessions", "sleep_sessions", "nursing_sessions", "kid_modules", "growth_measurements",
	"vaccine_doses", "vaccine_dismissals", "bottle_feed_sources", "pumping_session_sides",
	"symptoms", "medications", "medication_doses", "reminders", "reminder_deliveries", "push_subscriptions", "push_kid_optins",
	"ntfy_topics", "ntfy_kid_optins", "milestones_reached", "visits"}

func version(t *testing.T, s *Store) string {
	t.Helper()
	v, ok, err := s.MetaGet(context.Background(), "schema_version")
	if err != nil || !ok {
		t.Fatalf("schema_version: %q %v %v", v, ok, err)
	}
	return v
}

func TestOpenAppliesMigrations(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if v := version(t, s); v != "8" {
		t.Fatalf("schema_version = %q, want 8", v)
	}
	for _, name := range wantTables {
		var n int
		if err := s.r.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, name).Scan(&n); err != nil || n != 1 {
			t.Errorf("table %s: count=%d err=%v", name, n, err)
		}
	}
}

func TestOpenIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 2; i++ {
		s, err := Open(dir)
		if err != nil {
			t.Fatalf("open %d: %v", i, err)
		}
		if v := version(t, s); v != "8" {
			t.Fatalf("open %d: schema_version = %q, want 8", i, v)
		}
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestOpenRefusesNewerSchema(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.MetaSet(context.Background(), "schema_version", "99"); err != nil {
		t.Fatal(err)
	}
	s.Close()
	if s, err = Open(dir); err == nil {
		s.Close()
		t.Fatal("Open accepted a newer schema")
	}
}

func TestForeignKeysOn(t *testing.T) {
	s, err := OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	_, err = s.w.Exec(`INSERT INTO kid_access(kid_id, parent_id, role, created_at) VALUES ('k', 'missing', 'guardian', 0)`)
	if err == nil {
		t.Fatal("kid_access for a missing parent was accepted")
	}
}

func TestAdminSessionShape(t *testing.T) {
	s, err := OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err := s.w.Exec(`INSERT INTO parents(id, username, display_name, created_at, updated_at) VALUES ('p', 'parent1', 'P', 0, 0)`); err != nil {
		t.Fatal(err)
	}
	const ins = `INSERT INTO sessions(id, token_hash, kind, parent_id, created_at, last_seen_at, expires_at) VALUES (?, ?, ?, ?, 0, 0, 1)`
	if _, err := s.w.Exec(ins, "s1", []byte{1}, "admin", "p"); err == nil {
		t.Error("admin session with a parent_id was accepted")
	}
	if _, err := s.w.Exec(ins, "s2", []byte{2}, "parent", nil); err == nil {
		t.Error("parent session without a parent_id was accepted")
	}
	if _, err := s.w.Exec(ins, "s3", []byte{3}, "admin", nil); err != nil {
		t.Errorf("valid admin session rejected: %v", err)
	}
}

func TestOpenMemoryIsolatedAndRemoved(t *testing.T) {
	a, err := OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	b, err := OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	if err := a.MetaSet(context.Background(), "k", "v"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := b.MetaGet(context.Background(), "k"); ok {
		t.Error("memory stores share data")
	}
	dir := a.cleanup
	if dir == "" || !strings.Contains(dir, "happybabytracker") {
		t.Fatalf("cleanup dir = %q", dir)
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("temp dir survived Close: %v", err)
	}
}

func TestMetaRoundTrip(t *testing.T) {
	s, err := OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	if _, ok, err := s.MetaGet(ctx, "absent"); ok || err != nil {
		t.Fatalf("absent key: ok=%v err=%v", ok, err)
	}
	for _, v := range []string{"one", "two"} {
		if err := s.MetaSet(ctx, "k", v); err != nil {
			t.Fatal(err)
		}
		if got, ok, err := s.MetaGet(ctx, "k"); got != v || !ok || err != nil {
			t.Fatalf("got %q %v %v, want %q", got, ok, err, v)
		}
	}
}

func TestAutoVacuumIncremental(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var got int
	if err := s.w.QueryRow(`PRAGMA auto_vacuum`).Scan(&got); err != nil || got != 2 {
		t.Fatalf("auto_vacuum = %d (%v), want 2 (INCREMENTAL)", got, err)
	}
}

// trackerInserts has one valid insert per tracker table; ? is kid_id, then the creating parent.
var trackerInserts = map[string]string{
	"bottle_feeds": `INSERT INTO bottle_feeds(id, kid_id, at, has_breast_milk, has_formula, has_cow_milk, has_other, volume_ml, volume_floz_h, volume_typed, created_by, updated_by, created_at, updated_at)
		VALUES ('e1', ?, 100, 1, 1, 0, 0, 120, 406, 'ml', ?, ?, 100, 100)`,
	"diapers": `INSERT INTO diapers(id, kid_id, at, kind, created_by, updated_by, created_at, updated_at)
		VALUES ('e1', ?, 100, 'wet', ?, ?, 100, 100)`,
	"meals": `INSERT INTO meals(id, kid_id, at, what, created_by, updated_by, created_at, updated_at)
		VALUES ('e1', ?, 100, 'banana', ?, ?, 100, 100)`,
	"pumping_sessions": `INSERT INTO pumping_sessions(id, kid_id, started_at, duration_seconds, volume_ml, volume_floz_h, volume_typed, side, created_by, updated_by, created_at, updated_at)
		VALUES ('e1', ?, 100, 600, 90, 304, 'ml', 'both', ?, ?, 100, 100)`,
	"sleep_sessions": `INSERT INTO sleep_sessions(id, kid_id, started_at, ended_at, created_by, updated_by, created_at, updated_at)
		VALUES ('e1', ?, 100, 200, ?, ?, 100, 100)`,
	"nursing_sessions": `INSERT INTO nursing_sessions(id, kid_id, started_at, ended_at, left_seconds, right_seconds, side, side_since, created_by, updated_by, created_at, updated_at)
		VALUES ('e1', ?, 100, 400, 120, 180, 'left', 100, ?, ?, 100, 100)`,
	"kid_modules": `INSERT INTO kid_modules(kid_id, module, enabled, updated_at) VALUES (?, 'bottles', 1, 100)`,
}

func insertTracker(s *Store, table, kidID, parentID string) error {
	q := trackerInserts[table]
	if table == "kid_modules" {
		_, err := s.w.Exec(q, kidID)
		return err
	}
	_, err := s.w.Exec(q, kidID, parentID, parentID)
	return err
}

func TestTrackerTablesCascadeWithKid(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	for table := range trackerInserts {
		if err := insertTracker(s, table, k.ID, ana.ID); err != nil {
			t.Fatalf("insert into %s: %v", table, err)
		}
	}
	if err := s.DeleteKid(ctx, k.ID, ana.ID); err != nil {
		t.Fatal(err)
	}
	for table := range trackerInserts {
		var n int
		if err := s.w.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil || n != 0 {
			t.Errorf("%s: %d rows after DeleteKid (err %v)", table, n, err)
		}
	}
}

func TestOneRunningRowPerKid(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k1, k2 := mustKid(t, s, ana.ID, "Lía"), mustKid(t, s, ana.ID, "Noa")
	for table, q := range map[string]string{
		"sleep_sessions":   `INSERT INTO sleep_sessions(id, kid_id, started_at, created_by, updated_by, created_at, updated_at) VALUES (?, ?, 100, ?, ?, 100, 100)`,
		"nursing_sessions": `INSERT INTO nursing_sessions(id, kid_id, started_at, side, side_since, created_by, updated_by, created_at, updated_at) VALUES (?, ?, 100, 'left', 100, ?, ?, 100, 100)`,
	} {
		if _, err := s.w.Exec(q, "r1", k1.ID, ana.ID, ana.ID); err != nil {
			t.Fatalf("%s: first running row: %v", table, err)
		}
		if _, err := s.w.Exec(q, "r2", k1.ID, ana.ID, ana.ID); err == nil {
			t.Errorf("%s: a second running row for the same kid was accepted", table)
		}
		if _, err := s.w.Exec(q, "r3", k2.ID, ana.ID, ana.ID); err != nil {
			t.Errorf("%s: a running row for another kid was refused: %v", table, err)
		}
	}
}

func TestChecksRejectBadEnums(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	bad := map[string]string{
		"contents all zero": `INSERT INTO bottle_feeds(id, kid_id, at, has_breast_milk, has_formula, has_cow_milk, has_other, volume_ml, volume_floz_h, volume_typed, created_by, updated_by, created_at, updated_at)
			VALUES ('b1', ?, 100, 0, 0, 0, 0, 120, 406, 'ml', ?, ?, 100, 100)`,
		"contents flag of 2": `INSERT INTO bottle_feeds(id, kid_id, at, has_breast_milk, has_formula, has_cow_milk, has_other, volume_ml, volume_floz_h, volume_typed, created_by, updated_by, created_at, updated_at)
			VALUES ('b2', ?, 100, 2, 0, 0, 0, 120, 406, 'ml', ?, ?, 100, 100)`,
		"diaper kind": `INSERT INTO diapers(id, kid_id, at, kind, created_by, updated_by, created_at, updated_at)
			VALUES ('d1', ?, 100, 'soaked', ?, ?, 100, 100)`,
		"pumping side": `INSERT INTO pumping_sessions(id, kid_id, started_at, duration_seconds, volume_ml, volume_floz_h, volume_typed, side, created_by, updated_by, created_at, updated_at)
			VALUES ('p1', ?, 100, 600, 90, 304, 'ml', 'middle', ?, ?, 100, 100)`,
		"nursing side": `INSERT INTO nursing_sessions(id, kid_id, started_at, side, created_by, updated_by, created_at, updated_at)
			VALUES ('n1', ?, 100, 'both', ?, ?, 100, 100)`,
		"volume unit": `INSERT INTO bottle_feeds(id, kid_id, at, has_breast_milk, has_formula, has_cow_milk, has_other, volume_ml, volume_floz_h, volume_typed, created_by, updated_by, created_at, updated_at)
			VALUES ('b3', ?, 100, 0, 1, 0, 0, 120, 406, 'cup', ?, ?, 100, 100)`,
		"sleep ends before it starts": `INSERT INTO sleep_sessions(id, kid_id, started_at, ended_at, created_by, updated_by, created_at, updated_at)
			VALUES ('s1', ?, 100, 99, ?, ?, 100, 100)`,
	}
	for name, q := range bad {
		if _, err := s.w.Exec(q, k.ID, ana.ID, ana.ID); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, err := s.w.Exec(`INSERT INTO kid_modules(kid_id, module, enabled, updated_at) VALUES (?, 'astrology', 1, 100)`, k.ID); err == nil {
		t.Error("module: unknown id accepted")
	}
}

func TestEveryModelModuleIsAcceptedByKidModules(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	for _, m := range model.Modules {
		if _, err := s.w.Exec(`INSERT INTO kid_modules(kid_id, module, enabled, updated_at) VALUES (?, ?, 1, 100)`, k.ID, string(m)); err != nil {
			t.Errorf("module %s refused: %v", m, err)
		}
	}
	if len(model.Modules) != 12 {
		t.Errorf("%d modules, want 12", len(model.Modules))
	}
}

const insertGrowth = `INSERT INTO growth_measurements(id, kid_id, measured_on, weight_g, weight_oz_h, weight_typed,
	length_mm, length_in_h, length_typed, head_mm, head_in_h, head_typed, created_by, updated_by, created_at, updated_at)
	VALUES (?, ?, '2026-03-01', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 100, 100)`

func TestGrowthTableChecks(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	row := func(id string, v ...any) error {
		args := append([]any{id, k.ID}, v...)
		_, err := s.w.Exec(insertGrowth, append(args, ana.ID, ana.ID)...)
		return err
	}
	n := any(nil)
	for name, v := range map[string][]any{
		"no measure at all":       {n, n, n, n, n, n, n, n, n},
		"weight grams alone":      {3400, n, n, n, n, n, n, n, n},
		"weight without typed":    {3400, 120, n, n, n, n, n, n, n},
		"weight typed alone":      {n, n, "kg", 500, 197, "cm", n, n, n},
		"weight ounces alone":     {n, 120, n, 500, 197, "cm", n, n, n},
		"length half set":         {n, n, n, 500, n, "cm", n, n, n},
		"length without typed":    {n, n, n, 500, 197, n, n, n, n},
		"head half set":           {3400, 120, "kg", n, n, n, 350, n, "cm"},
		"head typed alone":        {3400, 120, "kg", n, n, n, n, n, "cm"},
		"weight unit":             {3400, 120, "stone", n, n, n, n, n, n},
		"weight in a length unit": {3400, 120, "cm", n, n, n, n, n, n},
		"length unit":             {n, n, n, 500, 197, "kg", n, n, n},
		"head unit":               {n, n, n, n, n, n, 350, 138, "ft"},
	} {
		if err := row("bad", v...); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	for name, v := range map[string][]any{
		"weight only": {3400, 120, "kg", n, n, n, n, n, n},
		"length only": {n, n, n, 500, 197, "in", n, n, n},
		"head only":   {n, n, n, n, n, n, 350, 138, "cm"},
		"all three":   {3400, 120, "lb", 500, 197, "cm", 350, 138, "in"},
	} {
		if err := row("ok-"+name, v...); err != nil {
			t.Errorf("%s: refused: %v", name, err)
		}
	}
	if _, err := s.w.Exec(insertGrowth, "orphan", "no-such-kid", 3400, 120, "kg", n, n, n, n, n, n, ana.ID, ana.ID); err == nil {
		t.Error("a measurement for a missing kid was accepted")
	}
	if _, err := s.w.Exec(insertGrowth, "nobody", k.ID, 3400, 120, "kg", n, n, n, n, n, n, "ghost", "ghost"); err == nil {
		t.Error("a measurement by a missing parent was accepted")
	}
}

func TestGrowthCascadesWithKid(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k1, k2 := mustKid(t, s, ana.ID, "Lía"), mustKid(t, s, ana.ID, "Noa")
	for i, kid := range []string{k1.ID, k1.ID, k2.ID} {
		if _, err := s.w.Exec(insertGrowth, "g"+string(rune('0'+i)), kid, 3400, 120, "kg", nil, nil, nil, nil, nil, nil, ana.ID, ana.ID); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.DeleteKid(ctx, k1.ID, ana.ID); err != nil {
		t.Fatal(err)
	}
	var gone, kept int
	if err := s.w.QueryRow(`SELECT COUNT(*) FROM growth_measurements WHERE kid_id = ?`, k1.ID).Scan(&gone); err != nil {
		t.Fatal(err)
	}
	if err := s.w.QueryRow(`SELECT COUNT(*) FROM growth_measurements WHERE kid_id = ?`, k2.ID).Scan(&kept); err != nil {
		t.Fatal(err)
	}
	if gone != 0 || kept != 1 {
		t.Errorf("after DeleteKid: %d rows of the deleted kid, %d of the other (want 0, 1)", gone, kept)
	}
}

func TestExistingKidsReadTheDefaultSource(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, dbFile)
	old, err := sql.Open("sqlite", "file:"+path+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	old.SetMaxOpenConns(1)
	ms, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range ms[:2] {
		if err := apply(ctx, old, m); err != nil {
			t.Fatal(err)
		}
	}
	for _, q := range []string{
		`INSERT INTO parents(id, username, display_name, created_at, updated_at) VALUES ('p1', 'ana', 'Ana', 100, 100)`,
		`INSERT INTO kids(id, name, birth_date, sex, time_zone, created_by, updated_by, created_at, updated_at)
			VALUES ('k1', 'Lía', '2026-01-02', 'female', 'Europe/Madrid', 'p1', 'p1', 100, 100)`,
		`INSERT INTO kid_access(kid_id, parent_id, role, created_at) VALUES ('k1', 'p1', 'guardian', 100)`,
	} {
		if _, err := old.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	before := &Store{w: old, r: old}
	for table := range trackerInserts {
		if err := insertTracker(before, table, "k1", "p1"); err != nil {
			t.Fatalf("insert into %s: %v", table, err)
		}
	}
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}

	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if v := version(t, s); v != "8" {
		t.Fatalf("schema_version = %q, want 8", v)
	}
	k, err := s.Kid(ctx, "k1")
	if err != nil || k.GrowthSource != "who" || k.Name != "Lía" || k.BirthDate != "2026-01-02" {
		t.Fatalf("kid after 0003 = %+v, %v", k, err)
	}
	for table := range trackerInserts {
		var n int
		if err := s.r.QueryRow(`SELECT COUNT(*) FROM ` + table + ` WHERE kid_id = 'k1'`).Scan(&n); err != nil || n != 1 {
			t.Errorf("%s: %d rows for the kid after 0003 (err %v)", table, n, err)
		}
	}
	if role, err := s.RoleFor(ctx, "k1", "p1"); err != nil || role != model.RoleGuardian {
		t.Errorf("grant after 0003 = %q, %v", role, err)
	}
	if err := s.DeleteKid(ctx, "k1", "p1"); err != nil {
		t.Fatal(err)
	}
	for table := range trackerInserts {
		var n int
		if err := s.w.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil || n != 0 {
			t.Errorf("%s: %d rows survive DeleteKid after 0003 (err %v)", table, n, err)
		}
	}
}

const insertDose = `INSERT INTO vaccine_doses(id, kid_id, administration, given_on, antigen, created_by, updated_by, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, 100, 100)`

func TestVaccineTableChecks(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	const full = `INSERT INTO vaccine_doses(id, kid_id, administration, given_on, antigen, lot, product, note, created_by, updated_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'dtap', ?, ?, ?, ?, ?, 100, 100)`
	n := any(nil)
	for name, v := range map[string][]any{
		"given_on with a time":   {"2026-03-01T10:00", n, n, n},
		"given_on not a date":    {"2026-02-30", n, n, n},
		"given_on short":         {"2026-3-01", n, n, n},
		"given_on words":         {"yesterday", n, n, n},
		"given_on empty":         {"", n, n, n},
		"empty lot":              {"2026-03-01", "", n, n},
		"empty product":          {"2026-03-01", n, "", n},
		"empty note":             {"2026-03-01", n, n, ""},
		"given_on as an integer": {20260301, n, n, n},
	} {
		if _, err := s.w.Exec(full, append(append([]any{"bad-" + name, k.ID, "adm-" + name}, v...), ana.ID, ana.ID)...); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, err := s.w.Exec(full, "ok", k.ID, "a1", "2028-02-29", "L1", "P", "N", ana.ID, ana.ID); err != nil {
		t.Errorf("a full row was refused: %v", err)
	}
	if _, err := s.w.Exec(insertDose, "twin", k.ID, "a1", "2028-02-29", "dtap", ana.ID, ana.ID); err == nil {
		t.Error("an antigen twice in one administration was accepted")
	}
	if _, err := s.w.Exec(insertDose, "orphan", "no-such-kid", "a2", "2026-03-01", "dtap", ana.ID, ana.ID); err == nil {
		t.Error("a dose for a missing kid was accepted")
	}
	const dismissal = `INSERT INTO vaccine_dismissals(kid_id, schema, line, reason, created_by, created_at) VALUES (?, 'us', ?, ?, ?, 100)`
	if _, err := s.w.Exec(dismissal, k.ID, "hepb-1", "forgot", ana.ID); err == nil {
		t.Error("dismissal reason: unknown value accepted")
	}
	if _, err := s.w.Exec(dismissal, "no-such-kid", "hepb-1", "not_needed", ana.ID); err == nil {
		t.Error("a dismissal for a missing kid was accepted")
	}
	if _, err := s.w.Exec(dismissal, k.ID, "hepb-1", "not_needed", ana.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.w.Exec(dismissal, k.ID, "hepb-1", "given_unknown", ana.ID); err == nil {
		t.Error("a second dismissal of the same line was accepted")
	}
}

func TestVaccineTablesCascadeWithKid(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k1, k2 := mustKid(t, s, ana.ID, "Lía"), mustKid(t, s, ana.ID, "Noa")
	for _, kid := range []string{k1.ID, k1.ID, k2.ID} {
		if _, err := s.CreateAdministration(ctx, kid, ana.ID, model.Administration{GivenOn: "2026-03-01", Antigens: []string{"dtap", "hib"}}, t0); err != nil {
			t.Fatal(err)
		}
		if err := s.SetDismissal(ctx, kid, ana.ID, "us", model.NewID(), "not_needed", t0); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.DeleteKid(ctx, k1.ID, ana.ID); err != nil {
		t.Fatal(err)
	}
	for table, want := range map[string]int{"vaccine_doses": 2, "vaccine_dismissals": 1} {
		var gone, kept int
		if err := s.w.QueryRow(`SELECT COUNT(*) FROM `+table+` WHERE kid_id = ?`, k1.ID).Scan(&gone); err != nil {
			t.Fatal(err)
		}
		if err := s.w.QueryRow(`SELECT COUNT(*) FROM `+table+` WHERE kid_id = ?`, k2.ID).Scan(&kept); err != nil {
			t.Fatal(err)
		}
		if gone != 0 || kept != want {
			t.Errorf("%s after DeleteKid: %d rows of the deleted kid, %d of the other (want 0, %d)", table, gone, kept, want)
		}
	}
}

func TestUpgradeFrom0003KeepsEveryRow(t *testing.T) {
	dir := t.TempDir()
	old, err := sql.Open("sqlite", "file:"+filepath.Join(dir, dbFile)+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	old.SetMaxOpenConns(1)
	ms, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range ms[:3] {
		if err := apply(ctx, old, m); err != nil {
			t.Fatal(err)
		}
	}
	for _, q := range []string{
		`INSERT INTO parents(id, username, display_name, created_at, updated_at) VALUES ('p1', 'ana', 'Ana', 100, 100)`,
		`INSERT INTO kids(id, name, birth_date, sex, time_zone, growth_source, created_by, updated_by, created_at, updated_at)
			VALUES ('k1', 'Lía', '2026-01-02', 'female', 'Europe/Madrid', 'cdc', 'p1', 'p1', 100, 100)`,
		`INSERT INTO kid_access(kid_id, parent_id, role, created_at) VALUES ('k1', 'p1', 'guardian', 100)`,
	} {
		if _, err := old.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	before := &Store{w: old, r: old}
	for table := range trackerInserts {
		if err := insertTracker(before, table, "k1", "p1"); err != nil {
			t.Fatalf("insert into %s: %v", table, err)
		}
	}
	if _, err := old.Exec(insertGrowth, "g1", "k1", 3400, 120, "kg", nil, nil, nil, nil, nil, nil, "p1", "p1"); err != nil {
		t.Fatal(err)
	}
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}

	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if v := version(t, s); v != "8" {
		t.Fatalf("schema_version = %q, want 8", v)
	}
	k, err := s.Kid(ctx, "k1")
	if err != nil || k.VaccineSchema != nil || k.GrowthSource != "cdc" || k.Name != "Lía" || k.BirthDate != "2026-01-02" {
		t.Fatalf("kid after 0004 = %+v, %v", k, err)
	}
	for table := range trackerInserts {
		var n int
		if err := s.r.QueryRow(`SELECT COUNT(*) FROM ` + table + ` WHERE kid_id = 'k1'`).Scan(&n); err != nil || n != 1 {
			t.Errorf("%s: %d rows for the kid after 0004 (err %v)", table, n, err)
		}
	}
	if g, err := s.ListGrowth(ctx, "k1"); err != nil || len(g) != 1 || g[0].ID != "g1" {
		t.Errorf("growth after 0004 = %+v, %v", g, err)
	}
	if role, err := s.RoleFor(ctx, "k1", "p1"); err != nil || role != model.RoleGuardian {
		t.Errorf("grant after 0004 = %q, %v", role, err)
	}
	if _, err := s.CreateAdministration(ctx, "k1", "p1", model.Administration{GivenOn: "2026-03-01", Antigens: []string{"dtap"}}, t0); err != nil {
		t.Errorf("a dose for an upgraded kid: %v", err)
	}
}

const (
	insertSource  = `INSERT INTO bottle_feed_sources(bottle_id, source, volume_ml, volume_floz_h, volume_typed) VALUES (?, ?, ?, ?, ?)`
	insertSide    = `INSERT INTO pumping_session_sides(session_id, side, volume_ml, volume_floz_h, volume_typed) VALUES (?, ?, ?, ?, ?)`
	insertWeighed = `INSERT INTO nursing_sessions(id, kid_id, started_at, ended_at, left_seconds, right_seconds,
		weight_before_g, weight_before_oz_h, weight_before_typed, weight_after_g, weight_after_oz_h, weight_after_typed,
		created_by, updated_by, created_at, updated_at)
		VALUES (?, ?, 100, 400, 120, 180, ?, ?, ?, ?, ?, ?, ?, ?, 100, 100)`
)

func TestAdvancedColumnsCascadeWithKid(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k1, k2 := mustKid(t, s, ana.ID, "Lía"), mustKid(t, s, ana.ID, "Noa")
	for i, kid := range []string{k1.ID, k2.ID} {
		id := "b" + string(rune('0'+i))
		if _, err := s.w.Exec(`INSERT INTO bottle_feeds(id, kid_id, at, has_breast_milk, has_formula, has_cow_milk, has_other, volume_ml, volume_floz_h, volume_typed, created_by, updated_by, created_at, updated_at)
			VALUES (?, ?, 100, 1, 1, 0, 0, 150, 507, 'ml', ?, ?, 100, 100)`, id, kid, ana.ID, ana.ID); err != nil {
			t.Fatal(err)
		}
		for _, src := range []string{"breast_milk", "formula"} {
			if _, err := s.w.Exec(insertSource, id, src, 75, 254, "ml"); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := s.w.Exec(insertWeighed, "n"+id, kid, 3400, 11993, "kg", 3462, 12212, "kg", ana.ID, ana.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := s.w.Exec(`INSERT INTO pumping_sessions(id, kid_id, started_at, duration_seconds, volume_ml, volume_floz_h, volume_typed, side, created_by, updated_by, created_at, updated_at)
			VALUES (?, ?, 100, 600, 110, 372, 'ml', 'both', ?, ?, 100, 100)`, "p"+id, kid, ana.ID, ana.ID); err != nil {
			t.Fatal(err)
		}
		for _, side := range []string{"left", "right"} {
			if _, err := s.w.Exec(insertSide, "p"+id, side, 55, 186, "ml"); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := s.DeleteKid(ctx, k1.ID, ana.ID); err != nil {
		t.Fatal(err)
	}
	for table, want := range map[string]int{"bottle_feed_sources": 2, "bottle_feeds": 1, "nursing_sessions": 1, "pumping_session_sides": 2, "pumping_sessions": 1} {
		var n int
		if err := s.w.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil || n != want {
			t.Errorf("%s: %d rows after DeleteKid (err %v), want %d", table, n, err, want)
		}
	}
}

func TestSourceChecks(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	if _, err := s.w.Exec(`INSERT INTO bottle_feeds(id, kid_id, at, has_breast_milk, has_formula, has_cow_milk, has_other, volume_ml, volume_floz_h, volume_typed, created_by, updated_by, created_at, updated_at)
		VALUES ('b1', ?, 100, 1, 1, 0, 0, 150, 507, 'ml', ?, ?, 100, 100)`, k.ID, ana.ID, ana.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.w.Exec(insertSource, "b1", "formula", 90, 304, "ml"); err != nil {
		t.Fatalf("a valid source row: %v", err)
	}
	for name, args := range map[string][]any{
		"unknown source name": {"b1", "juice", 60, 203, "ml"},
		"unknown typed unit":  {"b1", "breast_milk", 60, 203, "cup"},
		"duplicate source":    {"b1", "formula", 60, 203, "ml"},
		"missing bottle":      {"nope", "breast_milk", 60, 203, "ml"},
		"null volume":         {"b1", "breast_milk", nil, 203, "ml"},
	} {
		if _, err := s.w.Exec(insertSource, args...); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}

	n := any(nil)
	row := func(id string, v ...any) error {
		args := append([]any{id, k.ID}, v...)
		_, err := s.w.Exec(insertWeighed, append(args, ana.ID, ana.ID)...)
		return err
	}
	for name, v := range map[string][]any{
		"before grams alone":     {3400, n, n, n, n, n},
		"before without typed":   {3400, 11993, n, n, n, n},
		"before typed alone":     {n, n, "kg", n, n, n},
		"before ounces alone":    {n, 11993, n, n, n, n},
		"after grams alone":      {n, n, n, 3462, n, n},
		"after without typed":    {n, n, n, 3462, 12212, n},
		"after typed alone":      {n, n, n, n, n, "kg"},
		"after half set":         {3400, 11993, "kg", 3462, n, "kg"},
		"before unit":            {3400, 11993, "stone", n, n, n},
		"after in a length unit": {n, n, n, 3462, 12212, "cm"},
	} {
		if err := row("bad", v...); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	for name, v := range map[string][]any{
		"neither":     {n, n, n, n, n, n},
		"before only": {3400, 11993, "kg", n, n, n},
		"after only":  {n, n, n, 12212, 346, "lb"},
		"both":        {3400, 11993, "lb", 3462, 12212, "kg"},
	} {
		if err := row("ok-"+name, v...); err != nil {
			t.Errorf("%s: refused: %v", name, err)
		}
	}
}

func TestPumpingSideChecks(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	if _, err := s.w.Exec(trackerInserts["pumping_sessions"], k.ID, ana.ID, ana.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.w.Exec(insertSide, "e1", "left", 60, 203, "ml"); err != nil {
		t.Fatalf("a valid side row: %v", err)
	}
	for name, args := range map[string][]any{
		"the combined side":  {"e1", "both", 60, 203, "ml"},
		"unknown side name":  {"e1", "middle", 60, 203, "ml"},
		"wrong case":         {"e1", "Right", 60, 203, "ml"},
		"unknown typed unit": {"e1", "right", 60, 203, "cup"},
		"duplicate side":     {"e1", "left", 50, 169, "ml"},
		"missing session":    {"nope", "right", 60, 203, "ml"},
		"null volume":        {"e1", "right", nil, 203, "ml"},
	} {
		if _, err := s.w.Exec(insertSide, args...); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, err := s.w.Exec(insertSide, "e1", "right", 50, 169, "floz"); err != nil {
		t.Errorf("the other side: %v", err)
	}
}

func TestMigration0005KeepsRows(t *testing.T) {
	dir := t.TempDir()
	old, err := sql.Open("sqlite", "file:"+filepath.Join(dir, dbFile)+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	old.SetMaxOpenConns(1)
	ms, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range ms[:4] {
		if err := apply(ctx, old, m); err != nil {
			t.Fatal(err)
		}
	}
	for _, q := range []string{
		`INSERT INTO parents(id, username, display_name, created_at, updated_at) VALUES ('p1', 'ana', 'Ana', 100, 100)`,
		`INSERT INTO kids(id, name, birth_date, sex, time_zone, created_by, updated_by, created_at, updated_at)
			VALUES ('k1', 'Lía', '2026-01-02', 'female', 'Europe/Madrid', 'p1', 'p1', 100, 100)`,
		`INSERT INTO kid_access(kid_id, parent_id, role, created_at) VALUES ('k1', 'p1', 'guardian', 100)`,
		`INSERT INTO bottle_feeds(id, kid_id, note, at, has_breast_milk, has_formula, has_cow_milk, has_other, volume_ml, volume_floz_h, volume_typed, created_by, updated_by, created_at, updated_at)
			VALUES ('b1', 'k1', 'mixed', 100, 1, 1, 0, 0, 120, 406, 'ml', 'p1', 'p1', 100, 100)`,
		`INSERT INTO nursing_sessions(id, kid_id, note, started_at, ended_at, left_seconds, right_seconds, created_by, updated_by, created_at, updated_at)
			VALUES ('n1', 'k1', 'done', 100, 400, 120, 180, 'p1', 'p1', 101, 102)`,
		`INSERT INTO nursing_sessions(id, kid_id, started_at, left_seconds, right_seconds, side, side_since, created_by, updated_by, created_at, updated_at)
			VALUES ('n2', 'k1', 500, 0, 0, 'right', 500, 'p1', 'p1', 500, 500)`,
		`INSERT INTO pumping_sessions(id, kid_id, note, started_at, duration_seconds, volume_ml, volume_floz_h, volume_typed, side, created_by, updated_by, created_at, updated_at)
			VALUES ('u1', 'k1', 'pumped', 700, 900, 110, 372, 'ml', 'both', 'p1', 'p1', 700, 701)`,
	} {
		if _, err := old.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}

	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if v := version(t, s); v != "8" {
		t.Fatalf("schema_version = %q, want 8", v)
	}
	var contents, sources int
	var volume int64
	if err := s.r.QueryRow(`SELECT has_breast_milk + has_formula, volume_ml FROM bottle_feeds WHERE id = 'b1'`).Scan(&contents, &volume); err != nil || contents != 2 || volume != 120 {
		t.Errorf("the mixed bottle after 0005: contents %d, volume %d, err %v", contents, volume, err)
	}
	if err := s.r.QueryRow(`SELECT COUNT(*) FROM bottle_feed_sources`).Scan(&sources); err != nil || sources != 0 {
		t.Errorf("%d source rows after 0005 (err %v), want none", sources, err)
	}
	var pumpedSide string
	var pumpedML int64
	if err := s.r.QueryRow(`SELECT side, volume_ml FROM pumping_sessions WHERE id = 'u1'`).Scan(&pumpedSide, &pumpedML); err != nil || pumpedSide != "both" || pumpedML != 110 {
		t.Errorf("the pumping session after 0005: side %q, volume %d, err %v", pumpedSide, pumpedML, err)
	}
	var sides int
	if err := s.r.QueryRow(`SELECT COUNT(*) FROM pumping_session_sides`).Scan(&sides); err != nil || sides != 0 {
		t.Errorf("%d side rows after 0005 (err %v), want none", sides, err)
	}
	var note string
	var started, ended, left, right, created, updated int64
	var wb, wa any
	if err := s.r.QueryRow(`SELECT note, started_at, ended_at, left_seconds, right_seconds, created_at, updated_at, weight_before_g, weight_after_g
		FROM nursing_sessions WHERE id = 'n1'`).Scan(&note, &started, &ended, &left, &right, &created, &updated, &wb, &wa); err != nil {
		t.Fatal(err)
	}
	if note != "done" || started != 100 || ended != 400 || left != 120 || right != 180 || created != 101 || updated != 102 || wb != nil || wa != nil {
		t.Errorf("the finished session after 0005 = %q %d %d %d %d %d %d %v %v", note, started, ended, left, right, created, updated, wb, wa)
	}
	var side string
	if err := s.r.QueryRow(`SELECT side FROM nursing_sessions WHERE id = 'n2' AND ended_at IS NULL`).Scan(&side); err != nil || side != "right" {
		t.Errorf("the running session after 0005: side %q, err %v", side, err)
	}
	for _, idx := range []string{"nursing_sessions_kid_started", "nursing_one_running"} {
		var tbl string
		if err := s.r.QueryRow(`SELECT tbl_name FROM sqlite_master WHERE type = 'index' AND name = ?`, idx).Scan(&tbl); err != nil || tbl != "nursing_sessions" {
			t.Errorf("index %s after 0005: table %q, err %v", idx, tbl, err)
		}
	}
	if _, err := s.w.Exec(`INSERT INTO nursing_sessions(id, kid_id, started_at, created_by, updated_by, created_at, updated_at) VALUES ('n3', 'k1', 600, 'p1', 'p1', 600, 600)`); err == nil {
		t.Error("a second running session was accepted after 0005")
	}
	var bad int
	if err := s.r.QueryRow(`SELECT COUNT(*) FROM pragma_foreign_key_check`).Scan(&bad); err != nil || bad != 0 {
		t.Errorf("foreign_key_check after 0005: %d violations, err %v", bad, err)
	}
}

func TestMigration0006KeepsRows(t *testing.T) {
	dir := t.TempDir()
	old, err := sql.Open("sqlite", "file:"+filepath.Join(dir, dbFile)+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	old.SetMaxOpenConns(1)
	ms, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range ms[:5] {
		if err := apply(ctx, old, m); err != nil {
			t.Fatal(err)
		}
	}
	for _, q := range []string{
		`INSERT INTO parents(id, username, display_name, created_at, updated_at) VALUES ('p1', 'ana', 'Ana', 100, 100)`,
		`INSERT INTO kids(id, name, birth_date, sex, time_zone, growth_source, created_by, updated_by, created_at, updated_at)
			VALUES ('k1', 'Lía', '2026-01-02', 'female', 'Europe/Madrid', 'cdc', 'p1', 'p1', 100, 100)`,
		`INSERT INTO kid_access(kid_id, parent_id, role, created_at) VALUES ('k1', 'p1', 'guardian', 100)`,
		`INSERT INTO kid_modules(kid_id, module, enabled, updated_at) VALUES ('k1', 'sickness', 0, 100)`,
		`INSERT INTO bottle_feeds(id, kid_id, note, at, has_breast_milk, has_formula, has_cow_milk, has_other, volume_ml, volume_floz_h, volume_typed, created_by, updated_by, created_at, updated_at)
			VALUES ('b1', 'k1', 'mixed', 100, 1, 1, 0, 0, 120, 406, 'ml', 'p1', 'p1', 100, 100)`,
		`INSERT INTO bottle_feed_sources(bottle_id, source, volume_ml, volume_floz_h, volume_typed) VALUES ('b1', 'formula', 60, 203, 'ml')`,
		`INSERT INTO nursing_sessions(id, kid_id, started_at, left_seconds, right_seconds, side, side_since, created_by, updated_by, created_at, updated_at)
			VALUES ('n1', 'k1', 500, 0, 0, 'right', 500, 'p1', 'p1', 500, 500)`,
		`INSERT INTO pumping_sessions(id, kid_id, started_at, duration_seconds, volume_ml, volume_floz_h, volume_typed, side, created_by, updated_by, created_at, updated_at)
			VALUES ('u1', 'k1', 700, 900, 110, 372, 'ml', 'both', 'p1', 'p1', 700, 701)`,
		`INSERT INTO pumping_session_sides(session_id, side, volume_ml, volume_floz_h, volume_typed) VALUES ('u1', 'left', 55, 186, 'ml')`,
		`INSERT INTO growth_measurements(id, kid_id, measured_on, weight_g, weight_oz_h, weight_typed, created_by, updated_by, created_at, updated_at)
			VALUES ('g1', 'k1', '2026-03-01', 3400, 11993, 'kg', 'p1', 'p1', 100, 100)`,
		`INSERT INTO vaccine_doses(id, kid_id, administration, given_on, antigen, created_by, updated_by, created_at, updated_at)
			VALUES ('v1', 'k1', 'a1', '2026-03-01', 'dtap', 'p1', 'p1', 100, 100)`,
		`INSERT INTO vaccine_dismissals(kid_id, schema, line, reason, created_by, created_at) VALUES ('k1', 'us', 'hepb', 'not_needed', 'p1', 100)`,
	} {
		if _, err := old.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	before := &Store{w: old, r: old}
	for table := range trackerInserts {
		if table == "bottle_feeds" || table == "nursing_sessions" || table == "pumping_sessions" || table == "kid_modules" {
			continue
		}
		if err := insertTracker(before, table, "k1", "p1"); err != nil {
			t.Fatalf("insert into %s: %v", table, err)
		}
	}
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}

	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if v := version(t, s); v != "8" {
		t.Fatalf("schema_version = %q, want 8", v)
	}
	counts := map[string]int{"parents": 1, "kids": 1, "kid_access": 1, "kid_modules": 1, "bottle_feeds": 1, "bottle_feed_sources": 1,
		"nursing_sessions": 1, "pumping_sessions": 1, "pumping_session_sides": 1, "growth_measurements": 1, "vaccine_doses": 1, "vaccine_dismissals": 1}
	for table := range trackerInserts {
		if _, ok := counts[table]; !ok {
			counts[table] = 1
		}
	}
	for table, want := range counts {
		var n int
		if err := s.r.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil || n != want {
			t.Errorf("%s: %d rows after 0006 (err %v), want %d", table, n, err, want)
		}
	}
	var minutes int
	var source string
	if err := s.r.QueryRow(`SELECT reminder_overdue_minutes, growth_source FROM kids WHERE id = 'k1'`).Scan(&minutes, &source); err != nil ||
		minutes != constants.DefaultReminderOverdueMinutes || source != "cdc" {
		t.Errorf("the kid after 0006: overdue %d, source %q, err %v", minutes, source, err)
	}
	for _, idx := range []string{"reminder_deliveries_state_settled"} {
		var tbl string
		if err := s.r.QueryRow(`SELECT tbl_name FROM sqlite_master WHERE type = 'index' AND name = ?`, idx).Scan(&tbl); err != nil || tbl != "reminder_deliveries" {
			t.Errorf("index %s after 0006: table %q, err %v", idx, tbl, err)
		}
	}
	var bad int
	if err := s.r.QueryRow(`SELECT COUNT(*) FROM pragma_foreign_key_check`).Scan(&bad); err != nil || bad != 0 {
		t.Errorf("foreign_key_check after 0006: %d violations, err %v", bad, err)
	}
}

func TestMigration0007KeepsRows(t *testing.T) {
	dir := t.TempDir()
	old, err := sql.Open("sqlite", "file:"+filepath.Join(dir, dbFile)+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	old.SetMaxOpenConns(1)
	ms, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range ms[:6] {
		if err := apply(ctx, old, m); err != nil {
			t.Fatal(err)
		}
	}
	n := any(nil)
	for _, e := range []struct {
		q    string
		args []any
	}{
		{`INSERT INTO parents(id, username, display_name, created_at, updated_at) VALUES ('p1', 'ana', 'Ana', 100, 100)`, nil},
		{`INSERT INTO identities(parent_id, provider, subject, secret_hash) VALUES ('p1', 'password', 'ana', 'h')`, nil},
		{`INSERT INTO sessions(id, token_hash, kind, parent_id, created_at, last_seen_at, expires_at) VALUES ('s1', x'01', 'parent', 'p1', 100, 100, 200)`, nil},
		{`INSERT INTO one_time_tokens(token_hash, kind, parent_id, expires_at) VALUES (x'02', 'reset', 'p1', 200)`, nil},
		{`INSERT INTO kids(id, name, birth_date, sex, time_zone, growth_source, created_by, updated_by, created_at, updated_at)
			VALUES ('k1', 'Lía', '2026-01-02', 'female', 'Europe/Madrid', 'cdc', 'p1', 'p1', 100, 100)`, nil},
		{`INSERT INTO kid_access(kid_id, parent_id, role, created_at) VALUES ('k1', 'p1', 'guardian', 100)`, nil},
		{`INSERT INTO kid_modules(kid_id, module, enabled, updated_at) VALUES ('k1', 'milestones', 0, 100)`, nil},
		{`INSERT INTO bottle_feeds(id, kid_id, note, at, has_breast_milk, has_formula, has_cow_milk, has_other, volume_ml, volume_floz_h, volume_typed, created_by, updated_by, created_at, updated_at)
			VALUES ('b1', 'k1', 'mixed', 100, 1, 1, 0, 0, 120, 406, 'ml', 'p1', 'p1', 100, 100)`, nil},
		{`INSERT INTO bottle_feed_sources(bottle_id, source, volume_ml, volume_floz_h, volume_typed) VALUES ('b1', 'formula', 60, 203, 'ml')`, nil},
		{`INSERT INTO nursing_sessions(id, kid_id, started_at, left_seconds, right_seconds, side, side_since, created_by, updated_by, created_at, updated_at)
			VALUES ('n1', 'k1', 500, 0, 0, 'right', 500, 'p1', 'p1', 500, 500)`, nil},
		{`INSERT INTO pumping_sessions(id, kid_id, started_at, duration_seconds, volume_ml, volume_floz_h, volume_typed, side, created_by, updated_by, created_at, updated_at)
			VALUES ('u1', 'k1', 700, 900, 110, 372, 'ml', 'both', 'p1', 'p1', 700, 701)`, nil},
		{`INSERT INTO pumping_session_sides(session_id, side, volume_ml, volume_floz_h, volume_typed) VALUES ('u1', 'left', 55, 186, 'ml')`, nil},
		{`INSERT INTO growth_measurements(id, kid_id, measured_on, weight_g, weight_oz_h, weight_typed, created_by, updated_by, created_at, updated_at)
			VALUES ('g1', 'k1', '2026-03-01', 3400, 11993, 'kg', 'p1', 'p1', 100, 100)`, nil},
		{`INSERT INTO vaccine_doses(id, kid_id, administration, given_on, antigen, created_by, updated_by, created_at, updated_at)
			VALUES ('v1', 'k1', 'a1', '2026-03-01', 'dtap', 'p1', 'p1', 100, 100)`, nil},
		{`INSERT INTO vaccine_dismissals(kid_id, schema, line, reason, created_by, created_at) VALUES ('k1', 'us', 'hepb', 'not_needed', 'p1', 100)`, nil},
		{insertSymptom, []any{"sy1", "k1", "cough", n, n, n, n, n, n, "p1", "p1"}},
		{insertMedication, []any{"m1", "k1", "every", n, 8, "p1", "p1"}},
		{insertMedDose, []any{"d1", "k1", "m1", "p1", "p1"}},
		{insertReminder, []any{"r1", "k1", "m1", 1000, "pending"}},
		{insertSubscribed, []any{"ps1", "p1", "https://push.example/a"}},
		{insertDelivery, []any{"dl1", "r1", "push", "ps1", "p1", "sent"}},
		{`INSERT INTO push_kid_optins(subscription_id, kid_id) VALUES ('ps1', 'k1')`, nil},
		{`INSERT INTO ntfy_topics(parent_id, topic, created_at) VALUES ('p1', 'hbt-abc', 100)`, nil},
		{`INSERT INTO ntfy_kid_optins(parent_id, kid_id) VALUES ('p1', 'k1')`, nil},
	} {
		if _, err := old.Exec(e.q, e.args...); err != nil {
			t.Fatalf("seed %.60s: %v", e.q, err)
		}
	}
	before := &Store{w: old, r: old}
	for table := range trackerInserts {
		if table == "bottle_feeds" || table == "nursing_sessions" || table == "pumping_sessions" || table == "kid_modules" {
			continue
		}
		if err := insertTracker(before, table, "k1", "p1"); err != nil {
			t.Fatalf("insert into %s: %v", table, err)
		}
	}
	counts := map[string]int{}
	for _, table := range wantTables {
		if table == "meta" || table == "milestones_reached" || table == "visits" {
			continue
		}
		var n int
		if err := old.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil || n == 0 {
			t.Fatalf("%s before 0007: %d rows, err %v; want a seeded row", table, n, err)
		}
		counts[table] = n
	}
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}

	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if v := version(t, s); v != "8" {
		t.Fatalf("schema_version = %q, want 8", v)
	}
	for table, want := range counts {
		var n int
		if err := s.r.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil || n != want {
			t.Errorf("%s: %d rows after 0007 (err %v), want %d", table, n, err, want)
		}
	}
	for _, idx := range []string{"milestones_reached_kid_item", "milestones_reached_kid_date"} {
		var tbl string
		if err := s.r.QueryRow(`SELECT tbl_name FROM sqlite_master WHERE type = 'index' AND name = ?`, idx).Scan(&tbl); err != nil || tbl != "milestones_reached" {
			t.Errorf("index %s after 0007: table %q, err %v", idx, tbl, err)
		}
	}
	if _, err := s.CreateMilestone(ctx, "k1", "p1", model.Milestone{Text: strp("Claps"), ReachedOn: "2026-03-01"}, t0); err != nil {
		t.Errorf("a milestone for the kept kid after 0007: %v", err)
	}
	var bad int
	if err := s.r.QueryRow(`SELECT COUNT(*) FROM pragma_foreign_key_check`).Scan(&bad); err != nil || bad != 0 {
		t.Errorf("foreign_key_check after 0007: %d violations, err %v", bad, err)
	}
}

func TestMigration0008KeepsRows(t *testing.T) {
	dir := t.TempDir()
	old, err := sql.Open("sqlite", "file:"+filepath.Join(dir, dbFile)+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	old.SetMaxOpenConns(1)
	ms, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range ms[:7] {
		if err := apply(ctx, old, m); err != nil {
			t.Fatal(err)
		}
	}
	n := any(nil)
	for _, e := range []struct {
		q    string
		args []any
	}{
		{`INSERT INTO parents(id, username, display_name, created_at, updated_at) VALUES ('p1', 'ana', 'Ana', 100, 100)`, nil},
		{`INSERT INTO kids(id, name, birth_date, sex, time_zone, growth_source, created_by, updated_by, created_at, updated_at)
			VALUES ('k1', 'Lía', '2026-01-02', 'female', 'Europe/Madrid', 'cdc', 'p1', 'p1', 100, 100)`, nil},
		{`INSERT INTO kid_access(kid_id, parent_id, role, created_at) VALUES ('k1', 'p1', 'guardian', 100)`, nil},
		{`INSERT INTO kid_modules(kid_id, module, enabled, updated_at) VALUES ('k1', 'milestones', 0, 100)`, nil},
		{`INSERT INTO kid_modules(kid_id, module, enabled, updated_at) VALUES ('k1', 'growth', 1, 101)`, nil},
		{`INSERT INTO growth_measurements(id, kid_id, measured_on, weight_g, weight_oz_h, weight_typed, created_by, updated_by, created_at, updated_at)
			VALUES ('g1', 'k1', '2026-03-01', 3400, 11993, 'kg', 'p1', 'p1', 100, 100)`, nil},
		{`INSERT INTO vaccine_doses(id, kid_id, administration, given_on, antigen, created_by, updated_by, created_at, updated_at)
			VALUES ('v1', 'k1', 'a1', '2026-03-01', 'dtap', 'p1', 'p1', 100, 100)`, nil},
		{insertMedication, []any{"m1", "k1", "every", n, 8, "p1", "p1"}},
		{insertMedDose, []any{"d1", "k1", "m1", "p1", "p1"}},
		{insertReminder, []any{"r1", "k1", "m1", 1000, "pending"}},
	} {
		if _, err := old.Exec(e.q, e.args...); err != nil {
			t.Fatalf("seed %.60s: %v", e.q, err)
		}
	}
	counts := map[string]int{}
	for _, table := range wantTables {
		if table == "meta" || table == "visits" {
			continue
		}
		var n int
		if err := old.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil {
			t.Fatalf("%s before 0008: %v", table, err)
		}
		counts[table] = n
	}
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}

	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if v := version(t, s); v != "8" {
		t.Fatalf("schema_version = %q, want 8", v)
	}
	for table, want := range counts {
		var n int
		if err := s.r.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil || n != want {
			t.Errorf("%s: %d rows after 0008 (err %v), want %d", table, n, err, want)
		}
	}
	var enabled, updated int
	if err := s.r.QueryRow(`SELECT enabled, updated_at FROM kid_modules WHERE kid_id = 'k1' AND module = 'milestones'`).Scan(&enabled, &updated); err != nil || enabled != 0 || updated != 100 {
		t.Errorf("kid_modules milestones after 0008: enabled %d, updated %d, err %v", enabled, updated, err)
	}
	for _, table := range []string{"growth_measurements", "vaccine_doses", "medications"} {
		var linked int
		if err := s.r.QueryRow(`SELECT COUNT(*) FROM ` + table + ` WHERE visit_id IS NOT NULL`).Scan(&linked); err != nil || linked != 0 {
			t.Errorf("%s: %d linked rows after 0008, err %v", table, linked, err)
		}
	}
	for idx, tbl := range map[string]string{"visits_kid_starts": "visits", "growth_measurements_visit": "growth_measurements",
		"vaccine_doses_visit": "vaccine_doses", "medications_visit": "medications"} {
		var got string
		if err := s.r.QueryRow(`SELECT tbl_name FROM sqlite_master WHERE type = 'index' AND name = ?`, idx).Scan(&got); err != nil || got != tbl {
			t.Errorf("index %s after 0008: table %q, err %v", idx, got, err)
		}
	}
	if _, err := s.w.Exec(`INSERT INTO kid_modules(kid_id, module, enabled, updated_at) VALUES ('k1', 'visits', 1, 100)`); err != nil {
		t.Errorf("visits module after 0008: %v", err)
	}
	var bad int
	if err := s.r.QueryRow(`SELECT COUNT(*) FROM pragma_foreign_key_check`).Scan(&bad); err != nil || bad != 0 {
		t.Errorf("foreign_key_check after 0008: %d violations, err %v", bad, err)
	}
}

func TestVisitLinkIsSetNullOnDelete(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	for _, q := range []string{
		`INSERT INTO visits(id, kid_id, starts_at, kind, created_by, updated_by, created_at, updated_at) VALUES ('vi1', ?, 100, 'sick', ?, ?, 100, 100)`,
		`INSERT INTO growth_measurements(id, kid_id, measured_on, weight_g, weight_oz_h, weight_typed, created_by, updated_by, created_at, updated_at, visit_id)
			VALUES ('g1', ?, '2026-03-01', 3400, 11993, 'kg', ?, ?, 100, 100, 'vi1')`,
	} {
		if _, err := s.w.Exec(q, k.ID, ana.ID, ana.ID); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	if _, err := s.w.Exec(`DELETE FROM visits WHERE id = 'vi1'`); err != nil {
		t.Fatal(err)
	}
	var visit sql.NullString
	if err := s.r.QueryRow(`SELECT visit_id FROM growth_measurements WHERE id = 'g1'`).Scan(&visit); err != nil || visit.Valid {
		t.Errorf("growth row after its visit's delete: visit_id %v, err %v; want the row with NULL", visit, err)
	}
}

const (
	insertSymptom = `INSERT INTO symptoms(id, kid_id, at, symptom, severity, other_text, temp_c_h, temp_f_h, temp_typed, method, created_by, updated_by, created_at, updated_at)
		VALUES (?, ?, 100, ?, ?, ?, ?, ?, ?, ?, ?, ?, 100, 100)`
	insertMedication = `INSERT INTO medications(id, kid_id, name, dose, schedule_kind, schedule_times, every_hours, start_at, end_on, active, created_by, updated_by, created_at, updated_at)
		VALUES (?, ?, 'Paracetamol', '2.5 ml', ?, ?, ?, 100, NULL, 1, ?, ?, 100, 100)`
	insertMedDose = `INSERT INTO medication_doses(id, kid_id, medication_id, given_at, amount, created_by, updated_by, created_at, updated_at)
		VALUES (?, ?, ?, 100, NULL, ?, ?, 100, 100)`
	insertReminder   = `INSERT INTO reminders(id, kid_id, medication_id, due_at, state, created_at) VALUES (?, ?, ?, ?, ?, 100)`
	insertDelivery   = `INSERT INTO reminder_deliveries(id, reminder_id, channel, target_id, parent_id, state, attempts, next_attempt_at) VALUES (?, ?, ?, ?, ?, ?, 0, 100)`
	insertSubscribed = `INSERT INTO push_subscriptions(id, parent_id, endpoint, p256dh, auth, label, created_at, updated_at) VALUES (?, ?, ?, 'k', 'a', 'Phone', 100, 100)`
)

func TestReminderChecks(t *testing.T) {
	s := openMem(t)
	ana, bob := mustParent(t, s, "ana"), mustParent(t, s, "bob")
	k := mustKid(t, s, ana.ID, "Lía")
	n := any(nil)
	exec := func(q string, args ...any) error {
		_, err := s.w.Exec(q, args...)
		return err
	}
	check := func(name string, ok bool, err error) {
		t.Helper()
		if ok && err != nil {
			t.Errorf("%s: refused: %v", name, err)
		}
		if !ok && err == nil {
			t.Errorf("%s: accepted", name)
		}
	}

	sym := func(id string, symptom, severity, other, c, f, typed, method any) error {
		return exec(insertSymptom, id, k.ID, symptom, severity, other, c, f, typed, method, ana.ID, ana.ID)
	}
	for _, name := range model.Symptoms {
		switch name {
		case "fever":
			check("symptom "+name, true, sym("sy-"+name, name, n, n, 3850, 10130, "c", "oral"))
		case "other":
			check("symptom "+name, true, sym("sy-"+name, name, n, "teething", n, n, n, n))
		default:
			check("symptom "+name, true, sym("sy-"+name, name, n, n, n, n, n, n))
		}
	}
	for _, sev := range model.Severities {
		check("severity "+sev, true, sym("sv-"+sev, "cough", sev, n, n, n, n, n))
	}
	for _, m := range model.TempMethods {
		check("method "+m, true, sym("me-"+m, "fever", n, n, 3850, 10130, "f", m))
	}
	check("bad symptom", false, sym("sy-bad", "sneeze", n, n, n, n, n, n))
	check("bad severity", false, sym("sv-bad", "cough", "dire", n, n, n, n, n))
	check("bad method", false, sym("me-bad", "fever", n, n, 3850, 10130, "c", "toe"))
	check("bad temperature unit", false, sym("tu-bad", "fever", n, n, 3850, 10130, "k", "oral"))
	check("temperature without its Fahrenheit and unit", false, sym("tp-half", "fever", n, n, 3850, n, n, "oral"))
	check("Fahrenheit alone", false, sym("tp-f", "cough", n, n, n, 10130, n, n))
	check("unit alone", false, sym("tp-u", "cough", n, n, n, n, "c", n))
	check("fever without temperature", false, sym("fe-none", "fever", n, n, n, n, n, n))
	check("fever without method", false, sym("fe-nomethod", "fever", n, n, 3850, 10130, "c", n))
	check("temperature on another symptom", false, sym("fe-cough", "cough", n, n, 3850, 10130, "c", "oral"))
	check("method alone", false, sym("me-alone", "cough", n, n, n, n, n, "oral"))
	check("other without text", false, sym("ot-none", "other", n, n, n, n, n, n))
	check("text without other", false, sym("ot-cough", "cough", n, n, "teething", n, n, n))

	med := func(id, kind string, times, every any) error {
		return exec(insertMedication, id, k.ID, kind, times, every, ana.ID, ana.ID)
	}
	check("kind none", true, med("m-none", "none", n, n))
	check("kind times", true, med("m-times", "times", "08:00,20:00", n))
	check("kind every", true, med("m-every", "every", n, 8))
	check("every at the minimum", true, med("m-every-min", "every", n, constants.MinEveryHours))
	check("every at the maximum", true, med("m-every-max", "every", n, constants.MaxEveryHours))
	check("bad schedule kind", false, med("m-bad", "weekly", n, n))
	check("times kind with every_hours", false, med("m-t-e", "times", "08:00", 8))
	check("times kind without times", false, med("m-t-n", "times", n, n))
	check("every kind with times", false, med("m-e-t", "every", "08:00", 8))
	check("every kind without hours", false, med("m-e-n", "every", n, n))
	check("none kind with times", false, med("m-n-t", "none", "08:00", n))
	check("none kind with hours", false, med("m-n-e", "none", n, 8))
	check("every_hours below the minimum", false, med("m-low", "every", n, constants.MinEveryHours-1))
	check("every_hours above the maximum", false, med("m-high", "every", n, constants.MaxEveryHours+1))
	if _, err := s.w.Exec(`INSERT INTO medications(id, kid_id, name, dose, schedule_kind, start_at, active, created_by, updated_by, created_at, updated_at)
		VALUES ('m-act', ?, 'x', 'y', 'none', 100, 2, ?, ?, 100, 100)`, k.ID, ana.ID, ana.ID); err == nil {
		t.Error("active of 2: accepted")
	}

	check("dose", true, exec(insertMedDose, "d1", k.ID, "m-none", ana.ID, ana.ID))
	check("dose of a missing medication", false, exec(insertMedDose, "d2", k.ID, "nope", ana.ID, ana.ID))

	check("reminder", true, exec(insertReminder, "r1", k.ID, "m-every", 1000, "pending"))
	for i, state := range []string{"done", "expired", "cancelled"} {
		check("reminder "+state, true, exec(insertReminder, "r-"+state, k.ID, "m-every", 2000+i, state))
	}
	check("bad reminder state", false, exec(insertReminder, "r-bad", k.ID, "m-every", 3000, "snoozed"))
	check("duplicate (medication, due_at)", false, exec(insertReminder, "r-dup", k.ID, "m-every", 1000, "pending"))
	check("the same due_at on another course", true, exec(insertReminder, "r-other", k.ID, "m-times", 1000, "pending"))

	for _, state := range []string{"pending", "sending", "sent", "gone", "failed", "expired", "unconfirmed", "cancelled"} {
		check("delivery "+state, true, exec(insertDelivery, "dl-"+state, "r1", "push", "t"+state, bob.ID, state))
	}
	check("ntfy delivery", true, exec(insertDelivery, "dl-ntfy", "r1", "ntfy", "tn", bob.ID, "pending"))
	check("bad delivery channel", false, exec(insertDelivery, "dl-ch", "r1", "sms", "tx", bob.ID, "pending"))
	check("bad delivery state", false, exec(insertDelivery, "dl-st", "r1", "push", "ty", bob.ID, "queued"))
	check("duplicate delivery target", false, exec(insertDelivery, "dl-dup", "r1", "push", "tpending", bob.ID, "pending"))
	check("the same target on another channel", true, exec(insertDelivery, "dl-ch2", "r1", "ntfy", "tpending", bob.ID, "pending"))

	check("subscription", true, exec(insertSubscribed, "s1", bob.ID, "https://push.example/a"))
	check("duplicate endpoint", false, exec(insertSubscribed, "s2", ana.ID, "https://push.example/a"))
	check("opt-in", true, exec(`INSERT INTO push_kid_optins(subscription_id, kid_id) VALUES ('s1', ?)`, k.ID))
	check("duplicate opt-in", false, exec(`INSERT INTO push_kid_optins(subscription_id, kid_id) VALUES ('s1', ?)`, k.ID))
	check("ntfy topic", true, exec(`INSERT INTO ntfy_topics(parent_id, topic, created_at) VALUES (?, 'hbt-abc', 100)`, bob.ID))
	check("duplicate topic", false, exec(`INSERT INTO ntfy_topics(parent_id, topic, created_at) VALUES (?, 'hbt-abc', 100)`, ana.ID))
	check("second topic for a parent", false, exec(`INSERT INTO ntfy_topics(parent_id, topic, created_at) VALUES (?, 'hbt-def', 100)`, bob.ID))
	check("ntfy opt-in", true, exec(`INSERT INTO ntfy_kid_optins(parent_id, kid_id) VALUES (?, ?)`, bob.ID, k.ID))
	check("ntfy opt-in without a topic", false, exec(`INSERT INTO ntfy_kid_optins(parent_id, kid_id) VALUES (?, ?)`, ana.ID, k.ID))

	setMinutes := func(v int) error {
		return exec(`UPDATE kids SET reminder_overdue_minutes = ? WHERE id = ?`, v, k.ID)
	}
	check("overdue below the minimum", false, setMinutes(constants.MinReminderOverdueMinutes-1))
	check("overdue above the maximum", false, setMinutes(constants.MaxReminderOverdueMinutes+1))
	check("overdue at the minimum", true, setMinutes(constants.MinReminderOverdueMinutes))
	check("overdue at the maximum", true, setMinutes(constants.MaxReminderOverdueMinutes))
	check("overdue null", false, exec(`UPDATE kids SET reminder_overdue_minutes = NULL WHERE id = ?`, k.ID))
}

func TestCascades(t *testing.T) {
	seed := func(t *testing.T) (*Store, model.Parent, model.Parent, model.Kid) {
		s := openMem(t)
		ana, bob := mustParent(t, s, "ana"), mustParent(t, s, "bob")
		k := mustKid(t, s, ana.ID, "Lía")
		n := any(nil)
		for _, e := range []struct {
			q    string
			args []any
		}{
			{insertSymptom, []any{"sy1", k.ID, "cough", n, n, n, n, n, n, ana.ID, ana.ID}},
			{insertMedication, []any{"m1", k.ID, "every", n, 8, ana.ID, ana.ID}},
			{insertMedDose, []any{"d1", k.ID, "m1", ana.ID, ana.ID}},
			{insertReminder, []any{"r1", k.ID, "m1", 1000, "pending"}},
			{insertDelivery, []any{"dl1", "r1", "push", "s1", bob.ID, "sent"}},
			{insertDelivery, []any{"dl2", "r1", "ntfy", "bob", bob.ID, "pending"}},
			{insertSubscribed, []any{"s1", bob.ID, "https://push.example/a"}},
			{`INSERT INTO push_kid_optins(subscription_id, kid_id) VALUES ('s1', ?)`, []any{k.ID}},
			{`INSERT INTO ntfy_topics(parent_id, topic, created_at) VALUES (?, 'hbt-abc', 100)`, []any{bob.ID}},
			{`INSERT INTO ntfy_kid_optins(parent_id, kid_id) VALUES (?, ?)`, []any{bob.ID, k.ID}},
		} {
			if _, err := s.w.Exec(e.q, e.args...); err != nil {
				t.Fatalf("seed %.40s: %v", e.q, err)
			}
		}
		return s, ana, bob, k
	}
	count := func(t *testing.T, s *Store, table string) int {
		t.Helper()
		var n int
		if err := s.w.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	expect := func(t *testing.T, s *Store, want map[string]int) {
		t.Helper()
		for table, n := range want {
			if got := count(t, s, table); got != n {
				t.Errorf("%s: %d rows, want %d", table, got, n)
			}
		}
	}

	t.Run("kid", func(t *testing.T) {
		s, ana, _, k := seed(t)
		if err := s.DeleteKid(ctx, k.ID, ana.ID); err != nil {
			t.Fatal(err)
		}
		expect(t, s, map[string]int{"symptoms": 0, "medications": 0, "medication_doses": 0, "reminders": 0, "reminder_deliveries": 0,
			"push_kid_optins": 0, "ntfy_kid_optins": 0, "push_subscriptions": 1, "ntfy_topics": 1})
	})
	t.Run("medication", func(t *testing.T) {
		s, _, _, _ := seed(t)
		if _, err := s.w.Exec(`DELETE FROM medications WHERE id = 'm1'`); err != nil {
			t.Fatal(err)
		}
		expect(t, s, map[string]int{"medication_doses": 0, "reminders": 0, "reminder_deliveries": 0, "symptoms": 1, "push_kid_optins": 1})
	})
	t.Run("parent", func(t *testing.T) {
		s, _, bob, _ := seed(t)
		if _, err := s.w.Exec(`DELETE FROM parents WHERE id = ?`, bob.ID); err != nil {
			t.Fatal(err)
		}
		expect(t, s, map[string]int{"push_subscriptions": 0, "push_kid_optins": 0, "ntfy_topics": 0, "ntfy_kid_optins": 0,
			"reminder_deliveries": 0, "reminders": 1, "medications": 1, "kids": 1})
	})
}
