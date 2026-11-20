package store

import (
	"context"
	"os"
	"strings"
	"testing"
)

var wantTables = []string{"meta", "parents", "identities", "sessions", "one_time_tokens", "kids", "kid_access"}

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
	if v := version(t, s); v != "1" {
		t.Fatalf("schema_version = %q, want 1", v)
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
		if v := version(t, s); v != "1" {
			t.Fatalf("open %d: schema_version = %q", i, v)
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
