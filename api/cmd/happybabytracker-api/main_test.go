package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/auth"
	"github.com/davcs86/happybabytracker/api/internal/config"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

var now = time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)

func cfgWith(user, pass string) config.Config {
	return config.Config{
		AdminUsername: user, AdminPassword: pass,
		InternalAPIToken: "internal-token-0123456789abcdef0123456789abcdef",
	}
}

// restart closes and reopens the database, as a process restart does.
func restart(t *testing.T, s *store.Store, dir string) *store.Store {
	t.Helper()
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func has(t *testing.T, s *store.Store, hash []byte) bool {
	t.Helper()
	_, err := s.SessionByToken(context.Background(), hash, now)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		t.Fatal(err)
	}
	return err == nil
}

func TestAdminFingerprintRotation(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { s.Close() }()
	exp := now.Add(time.Hour)

	a := cfgWith("root-admin", "admin-password-AAAAAAAA")
	if err := syncAdminFingerprint(ctx, s, a); err != nil {
		t.Fatal(err)
	}
	p, err := s.CreateParent(ctx, "ana", "Ana", now)
	if err != nil {
		t.Fatal(err)
	}
	mustSession := func(kind, parentID, hash string) {
		t.Helper()
		if _, err := s.CreateSession(ctx, kind, parentID, []byte(hash), now, exp); err != nil {
			t.Fatal(err)
		}
	}
	mustSession("admin", "", "admin-1")
	mustSession("parent", p.ID, "parent-1")

	// Same credentials: nothing changes.
	s = restart(t, s, dir)
	if err := syncAdminFingerprint(ctx, s, a); err != nil {
		t.Fatal(err)
	}
	if !has(t, s, []byte("admin-1")) {
		t.Fatal("restart with the same credentials ended the admin session")
	}

	// New password: admin sessions go, parent sessions stay.
	b := cfgWith("root-admin", "admin-password-BBBBBBBB")
	s = restart(t, s, dir)
	if err := syncAdminFingerprint(ctx, s, b); err != nil {
		t.Fatal(err)
	}
	if has(t, s, []byte("admin-1")) {
		t.Fatal("admin session survived a password change")
	}
	if !has(t, s, []byte("parent-1")) {
		t.Fatal("a parent session was removed by an admin credential change")
	}

	// Restart again with the new password: sessions made since are kept.
	mustSession("admin", "", "admin-2")
	s = restart(t, s, dir)
	if err := syncAdminFingerprint(ctx, s, b); err != nil {
		t.Fatal(err)
	}
	if !has(t, s, []byte("admin-2")) {
		t.Fatal("restart with unchanged credentials ended the admin session")
	}

	// A changed username rotates too.
	c := cfgWith("other-admin", "admin-password-BBBBBBBB")
	s = restart(t, s, dir)
	if err := syncAdminFingerprint(ctx, s, c); err != nil {
		t.Fatal(err)
	}
	if has(t, s, []byte("admin-2")) {
		t.Fatal("admin session survived a username change")
	}

	stored, ok, err := s.MetaGet(ctx, "admin_fingerprint")
	if want := auth.AdminFingerprint(c.InternalAPIToken, c.AdminUsername, c.AdminPassword); err != nil || !ok || stored != want {
		t.Fatalf("stored fingerprint = %q (%v, %v), want %q", stored, ok, err, want)
	}
}

func TestFirstStartRecordsTheFingerprint(t *testing.T) {
	ctx := context.Background()
	s, err := store.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err := s.CreateSession(ctx, "admin", "", []byte("early"), now, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	cfg := cfgWith("root-admin", "admin-password-AAAAAAAA")
	if err := syncAdminFingerprint(ctx, s, cfg); err != nil {
		t.Fatal(err)
	}
	if got, ok, err := s.MetaGet(ctx, "admin_fingerprint"); err != nil || !ok || got == "" {
		t.Fatalf("fingerprint not stored: %q %v %v", got, ok, err)
	}
	// With no record, a session cannot be tied to the current credentials.
	if has(t, s, []byte("early")) {
		t.Fatal("an admin session of unknown origin survived the first fingerprint")
	}
}
