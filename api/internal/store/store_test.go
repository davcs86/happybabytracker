package store

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/davcs86/happybabytracker/api/internal/model"
)

var wantTables = []string{"meta", "parents", "identities", "sessions", "one_time_tokens", "kids", "kid_access",
	"bottle_feeds", "diapers", "meals", "pumping_sessions", "sleep_sessions", "nursing_sessions", "kid_modules", "growth_measurements",
	"vaccine_doses", "vaccine_dismissals", "bottle_feed_sources", "pumping_session_sides"}

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
	if v := version(t, s); v != "5" {
		t.Fatalf("schema_version = %q, want 5", v)
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
		if v := version(t, s); v != "5" {
			t.Fatalf("open %d: schema_version = %q, want 5", i, v)
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
	if len(model.Modules) != 11 {
		t.Errorf("%d modules, want 11", len(model.Modules))
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
	if v := version(t, s); v != "5" {
		t.Fatalf("schema_version = %q, want 5", v)
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
	if v := version(t, s); v != "5" {
		t.Fatalf("schema_version = %q, want 5", v)
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
	if v := version(t, s); v != "5" {
		t.Fatalf("schema_version = %q, want 5", v)
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
