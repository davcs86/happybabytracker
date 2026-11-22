package store

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/davcs86/happybabytracker/api/internal/model"
)

var wantTables = []string{"meta", "parents", "identities", "sessions", "one_time_tokens", "kids", "kid_access",
	"bottle_feeds", "diapers", "meals", "pumping_sessions", "sleep_sessions", "nursing_sessions", "kid_modules"}

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
	if v := version(t, s); v != "2" {
		t.Fatalf("schema_version = %q, want 2", v)
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
		if v := version(t, s); v != "2" {
			t.Fatalf("open %d: schema_version = %q, want 2", i, v)
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
