package store

import (
	"errors"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
)

func mustAdmin(t *testing.T, s *Store) model.Admin {
	t.Helper()
	a, err := s.CreateFirstAdmin(ctx, "root-admin", "h-admin", t0)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestCreateFirstAdminOnce(t *testing.T) {
	s := openMem(t)
	if ok, err := s.HasAdmin(ctx); err != nil || ok {
		t.Fatalf("HasAdmin before = %v, %v; want false", ok, err)
	}
	a := mustAdmin(t, s)
	if a.ID == "" || a.Username != "root-admin" || !a.HasPassword || a.SSOLinked || !a.CreatedAt.Equal(t0) {
		t.Errorf("admin = %+v", a)
	}
	if _, err := s.CreateFirstAdmin(ctx, "other-admin", "h2", t0); !errors.Is(err, ErrAdminExists) {
		t.Errorf("second call: %v, want ErrAdminExists", err)
	}
	var n int
	if err := s.r.QueryRow(`SELECT COUNT(*) FROM admins`).Scan(&n); err != nil || n != 1 {
		t.Errorf("admins rows = %d, %v; want 1", n, err)
	}
	var changed int64
	if err := s.r.QueryRow(`SELECT secret_changed_at FROM admin_identities WHERE admin_id = ? AND provider = 'local' AND subject = 'root-admin'`, a.ID).Scan(&changed); err != nil || changed != t0.Unix() {
		t.Errorf("secret_changed_at = %d, %v; want %d", changed, err, t0.Unix())
	}
	if ok, err := s.HasAdmin(ctx); err != nil || !ok {
		t.Errorf("HasAdmin after = %v, %v; want true", ok, err)
	}
	got, hash, err := s.AdminByUsername(ctx, "root-admin")
	if err != nil || got != a || hash != "h-admin" {
		t.Errorf("AdminByUsername = %+v %q %v", got, hash, err)
	}
	got, hash, err = s.AdminByID(ctx, a.ID)
	if err != nil || got != a || hash != "h-admin" {
		t.Errorf("AdminByID = %+v %q %v", got, hash, err)
	}
	if _, _, err := s.AdminByUsername(ctx, "other-admin"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown username: %v, want ErrNotFound", err)
	}
	if _, _, err := s.AdminByID(ctx, "nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown id: %v, want ErrNotFound", err)
	}
}

func TestAdminSSOLinked(t *testing.T) {
	s := openMem(t)
	a := mustAdmin(t, s)
	if _, err := s.w.Exec(`INSERT INTO admin_identities(admin_id, provider, subject) VALUES (?, 'https://issuer.example', 'sub-1')`, a.ID); err != nil {
		t.Fatal(err)
	}
	got, _, err := s.AdminByID(ctx, a.ID)
	if err != nil || !got.SSOLinked || !got.HasPassword {
		t.Errorf("AdminByID = %+v, %v; want linked with a password", got, err)
	}
}

func TestCreateFirstAdminRefusesAParentsName(t *testing.T) {
	s := openMem(t)
	mustParent(t, s, "david")
	if _, err := s.CreateFirstAdmin(ctx, "david", "h", t0); !errors.Is(err, ErrConflict) {
		t.Fatalf("CreateFirstAdmin: %v, want ErrConflict", err)
	}
	if ok, err := s.HasAdmin(ctx); err != nil || ok {
		t.Errorf("HasAdmin = %v, %v; want false", ok, err)
	}
}

func TestCreateParentRefusesTheAdminsName(t *testing.T) {
	s := openMem(t)
	mustAdmin(t, s)
	if _, err := s.CreateParent(ctx, "root-admin", "Root", t0); !errors.Is(err, ErrConflict) {
		t.Fatalf("CreateParent: %v, want ErrConflict", err)
	}
}

func TestOIDCIdentityIndexes(t *testing.T) {
	s := openMem(t)
	p := mustParent(t, s, "ana")
	a := mustAdmin(t, s)
	const iss = "https://issuer.example"
	if _, err := s.w.Exec(`INSERT INTO identities(parent_id, provider, subject) VALUES (?, ?, 'sub-1')`, p.ID, iss); err != nil {
		t.Fatal(err)
	}
	if _, err := s.w.Exec(`INSERT INTO identities(parent_id, provider, subject) VALUES (?, ?, 'sub-2')`, p.ID, iss); err == nil {
		t.Error("a second identity for the same parent and issuer was accepted")
	}
	if _, err := s.w.Exec(`INSERT INTO admin_identities(admin_id, provider, subject) VALUES (?, ?, 'sub-3')`, a.ID, iss); err != nil {
		t.Fatal(err)
	}
	if _, err := s.w.Exec(`INSERT INTO admin_identities(admin_id, provider, subject) VALUES (?, ?, 'sub-4')`, a.ID, iss); err == nil {
		t.Error("a second identity for the admin and the same issuer was accepted")
	}
}

func adminSession(t *testing.T, s *Store, kind, principal, token string) {
	t.Helper()
	if _, err := s.CreateSession(ctx, kind, principal, []byte(token), t0, t0.Add(time.Hour), ""); err != nil {
		t.Fatal(err)
	}
}

func sessionAlive(t *testing.T, s *Store, token string) bool {
	t.Helper()
	_, err := s.SessionByToken(ctx, []byte(token), t0)
	if err != nil && !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	return err == nil
}

func TestSetAdminPassword(t *testing.T) {
	s := openMem(t)
	a := mustAdmin(t, s)
	p := mustParent(t, s, "ana")
	adminSession(t, s, "admin", a.ID, "admin-1")
	adminSession(t, s, "admin", a.ID, "admin-2")
	adminSession(t, s, "parent", p.ID, "parent-1")
	later := t0.Add(time.Hour)
	if err := s.SetAdminPassword(ctx, "root-admin", "h-new", later); err != nil {
		t.Fatal(err)
	}
	if _, hash, err := s.AdminByUsername(ctx, "root-admin"); err != nil || hash != "h-new" {
		t.Errorf("hash = %q, %v; want h-new", hash, err)
	}
	var changed int64
	if err := s.r.QueryRow(`SELECT secret_changed_at FROM admin_identities WHERE admin_id = ? AND provider = 'local'`, a.ID).Scan(&changed); err != nil || changed != later.Unix() {
		t.Errorf("secret_changed_at = %d, %v; want %d", changed, err, later.Unix())
	}
	if sessionAlive(t, s, "admin-1") || sessionAlive(t, s, "admin-2") {
		t.Error("an admin session survived")
	}
	if !sessionAlive(t, s, "parent-1") {
		t.Error("the parent session was deleted")
	}
	if err := s.SetAdminPassword(ctx, "nobody", "h", later); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown admin: %v, want ErrNotFound", err)
	}
}

func TestSetAdminPasswordAddsAMissingLocalRow(t *testing.T) {
	s := openMem(t)
	a := mustAdmin(t, s)
	if _, err := s.w.Exec(`DELETE FROM admin_identities WHERE admin_id = ?`, a.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.SetAdminPassword(ctx, "root-admin", "h-new", t0); err != nil {
		t.Fatal(err)
	}
	if got, hash, err := s.AdminByUsername(ctx, "root-admin"); err != nil || hash != "h-new" || !got.HasPassword {
		t.Errorf("AdminByUsername = %+v %q %v", got, hash, err)
	}
}

func TestUnlinkAdminSSO(t *testing.T) {
	s := openMem(t)
	a := mustAdmin(t, s)
	p := mustParent(t, s, "ana")
	if _, err := s.w.Exec(`INSERT INTO admin_identities(admin_id, provider, subject) VALUES (?, 'https://issuer.example', 'sub-1')`, a.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.w.Exec(`INSERT INTO identities(parent_id, provider, subject) VALUES (?, 'https://issuer.example', 'sub-2')`, p.ID); err != nil {
		t.Fatal(err)
	}
	adminSession(t, s, "admin", a.ID, "admin-1")
	adminSession(t, s, "parent", p.ID, "parent-1")
	if err := s.UnlinkAdminSSO(ctx, "root-admin"); err != nil {
		t.Fatal(err)
	}
	got, hash, err := s.AdminByUsername(ctx, "root-admin")
	if err != nil || got.SSOLinked || hash != "h-admin" {
		t.Errorf("AdminByUsername = %+v %q %v; want unlinked with its password", got, hash, err)
	}
	if sessionAlive(t, s, "admin-1") {
		t.Error("the admin session survived")
	}
	var n int
	if err := s.r.QueryRow(`SELECT COUNT(*) FROM identities WHERE parent_id = ? AND provider <> 'local'`, p.ID).Scan(&n); err != nil || n != 1 {
		t.Errorf("parent OIDC rows = %d, %v; want 1", n, err)
	}
	if !sessionAlive(t, s, "parent-1") {
		t.Error("the parent session was deleted")
	}
	if err := s.UnlinkAdminSSO(ctx, "nobody"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown admin: %v, want ErrNotFound", err)
	}
}
