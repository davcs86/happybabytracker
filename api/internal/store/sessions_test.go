package store

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
)

var (
	ctx = context.Background()
	t0  = time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
)

func openMem(t *testing.T) *Store {
	t.Helper()
	s, err := OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func mustParent(t *testing.T, s *Store, username string) model.Parent {
	t.Helper()
	p, err := s.CreateParent(ctx, username, "Name "+username, t0)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func mustSession(t *testing.T, s *Store, kind, parentID string, hash []byte, expires time.Time) string {
	t.Helper()
	id, err := s.CreateSession(ctx, kind, parentID, hash, t0, expires, "")
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestSessionRoundTrip(t *testing.T) {
	s := openMem(t)
	p := mustParent(t, s, "ana")
	exp := t0.Add(constants.ParentSessionTTL)
	id := mustSession(t, s, "parent", p.ID, []byte("h-parent"), exp)
	a := mustAdmin(t, s)
	adminID := mustSession(t, s, "admin", a.ID, []byte("h-admin"), t0.Add(constants.AdminSessionTTL))

	got, err := s.SessionByToken(ctx, []byte("h-parent"), t0)
	if err != nil {
		t.Fatal(err)
	}
	want := model.Principal{Kind: "parent", ParentID: p.ID, SessionID: id, ExpiresAt: exp}
	if got != want {
		t.Fatalf("parent principal = %+v, want %+v", got, want)
	}
	got, err = s.SessionByToken(ctx, []byte("h-admin"), t0)
	if err != nil || got.Kind != "admin" || got.ParentID != "" || got.AdminID != a.ID || got.SessionID != adminID {
		t.Fatalf("admin principal = %+v, %v", got, err)
	}
}

func TestSessionByTokenRefusals(t *testing.T) {
	s := openMem(t)
	p := mustParent(t, s, "ana")
	exp := t0.Add(time.Hour)
	mustSession(t, s, "parent", p.ID, []byte("h1"), exp)

	if _, err := s.SessionByToken(ctx, []byte("unknown"), t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown token: %v, want ErrNotFound", err)
	}
	if _, err := s.SessionByToken(ctx, []byte("h1"), exp.Add(-time.Second)); err != nil {
		t.Errorf("1s before expiry: %v", err)
	}
	if _, err := s.SessionByToken(ctx, []byte("h1"), exp); !errors.Is(err, ErrNotFound) {
		t.Errorf("at expiry: %v, want ErrNotFound", err)
	}
	// Disabling removes the sessions, but a row that survives must still be refused.
	if _, err := s.w.ExecContext(ctx, `UPDATE parents SET disabled = 1 WHERE id = ?`, p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SessionByToken(ctx, []byte("h1"), t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("disabled parent: %v, want ErrNotFound", err)
	}
}

func TestCreateSessionRefusesInconsistentKind(t *testing.T) {
	s := openMem(t)
	p := mustParent(t, s, "ana")
	if _, err := s.CreateSession(ctx, "admin", p.ID, []byte("a"), t0, t0.Add(time.Hour), ""); err == nil {
		t.Error("admin session with a parent id was accepted")
	}
	if _, err := s.CreateSession(ctx, "parent", "", []byte("b"), t0, t0.Add(time.Hour), ""); err == nil {
		t.Error("parent session without a parent id was accepted")
	}
	if _, err := s.CreateSession(ctx, "other", "", []byte("c"), t0, t0.Add(time.Hour), ""); err == nil {
		t.Error("unknown session kind was accepted")
	}
	if _, err := s.CreateSession(ctx, "admin", "", []byte("e"), t0, t0.Add(time.Hour), ""); err == nil {
		t.Error("admin session without an admin id was accepted")
	}
	a := mustAdmin(t, s)
	mustSession(t, s, "admin", a.ID, []byte("d"), t0.Add(time.Hour))
	if _, err := s.CreateSession(ctx, "admin", a.ID, []byte("d"), t0, t0.Add(time.Hour), ""); err == nil {
		t.Error("duplicate token hash was accepted")
	}
}

func TestTouchSession(t *testing.T) {
	s := openMem(t)
	p := mustParent(t, s, "ana")
	id := mustSession(t, s, "parent", p.ID, []byte("h"), t0.Add(time.Hour))
	later := t0.Add(2 * time.Hour)
	newExp := later.Add(constants.ParentSessionTTL)
	if err := s.TouchSession(ctx, id, later, newExp); err != nil {
		t.Fatal(err)
	}
	got, err := s.SessionByToken(ctx, []byte("h"), later)
	if err != nil || !got.ExpiresAt.Equal(newExp) {
		t.Fatalf("after touch: %+v, %v; want expiry %v", got, err, newExp)
	}
	var seen int64
	if err := s.r.QueryRow(`SELECT last_seen_at FROM sessions WHERE id = ?`, id).Scan(&seen); err != nil || seen != later.Unix() {
		t.Fatalf("last_seen_at = %d, %v; want %d", seen, err, later.Unix())
	}
	if err := s.TouchSession(ctx, "no-such-session", later, newExp); err != nil {
		t.Errorf("touching a vanished session: %v, want nil", err)
	}
}

func TestDeleteSessions(t *testing.T) {
	s := openMem(t)
	p := mustParent(t, s, "ana")
	exp := t0.Add(time.Hour)
	pid := mustSession(t, s, "parent", p.ID, []byte("p1"), exp)
	mustSession(t, s, "parent", p.ID, []byte("p2"), exp)

	if err := s.DeleteSession(ctx, pid); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SessionByToken(ctx, []byte("p1"), t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("deleted session still resolves: %v", err)
	}
	if err := s.DeleteSession(ctx, pid); err != nil {
		t.Errorf("deleting twice: %v, want nil", err)
	}

}

func TestCreateParent(t *testing.T) {
	s := openMem(t)
	p, err := s.CreateParent(ctx, "ana", "Ana Díaz", t0)
	if err != nil {
		t.Fatal(err)
	}
	want := model.Parent{
		ID: p.ID, Username: "ana", DisplayName: "Ana Díaz",
		Units:     model.Units{Weight: "kg", Length: "cm", Volume: "ml", Temp: "c"},
		CreatedAt: t0,
	}
	if p != want || p.ID == "" || p.Disabled || p.HasPassword {
		t.Fatalf("CreateParent = %+v, want %+v", p, want)
	}
	if _, err := s.CreateParent(ctx, "ana", "Other", t0); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate username: %v, want ErrConflict", err)
	}
}

func TestParentLookups(t *testing.T) {
	s := openMem(t)
	p := mustParent(t, s, "ana")

	got, hash, err := s.ParentByUsername(ctx, "ana")
	if err != nil || got != p || hash != "" || got.HasPassword {
		t.Fatalf("no password yet: %+v %q %v", got, hash, err)
	}
	if _, _, err := s.ParentByUsername(ctx, "nobody"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown username: %v, want ErrNotFound", err)
	}
	if _, err := s.ParentByID(ctx, "nobody"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown id: %v, want ErrNotFound", err)
	}

	if err := s.SetPassword(ctx, p.ID, "$hash$one", "", t0); err != nil {
		t.Fatal(err)
	}
	got, hash, err = s.ParentByUsername(ctx, "ana")
	if err != nil || hash != "$hash$one" || !got.HasPassword {
		t.Fatalf("with password: %+v %q %v", got, hash, err)
	}
	byID, err := s.ParentByID(ctx, p.ID)
	if err != nil || !reflect.DeepEqual(byID, got) {
		t.Fatalf("ParentByID = %+v, %v; want %+v", byID, err, got)
	}
}

func TestSessionDeviceLabel(t *testing.T) {
	s := openMem(t)
	p := mustParent(t, s, "ana")
	exp := t0.Add(time.Hour)
	if _, err := s.CreateSession(ctx, "parent", p.ID, []byte("a"), t0, exp, "Safari on iPhone"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateSession(ctx, "parent", p.ID, []byte("b"), t0.Add(time.Minute), exp, ""); err != nil {
		t.Fatal(err)
	}
	got, err := s.ListSessions(ctx, p.ID, t0)
	if err != nil || len(got) != 2 {
		t.Fatalf("ListSessions = %v, %v", got, err)
	}
	if got[0].DeviceLabel != "" || got[1].DeviceLabel != "Safari on iPhone" {
		t.Errorf("labels = %q, %q; want empty then Safari on iPhone", got[0].DeviceLabel, got[1].DeviceLabel)
	}
	var n int
	if err := s.r.QueryRow(`SELECT COUNT(*) FROM sessions WHERE device_label = ''`).Scan(&n); err != nil || n != 0 {
		t.Errorf("empty label stored as empty string (%d rows), want NULL; %v", n, err)
	}
}

func TestSetPassword(t *testing.T) {
	s := openMem(t)
	p := mustParent(t, s, "ana")
	other := mustParent(t, s, "bob")
	exp := t0.Add(time.Hour)
	keep := mustSession(t, s, "parent", p.ID, []byte("keep"), exp)
	mustSession(t, s, "parent", p.ID, []byte("drop"), exp)
	mustSession(t, s, "parent", other.ID, []byte("bobs"), exp)

	if err := s.SetPassword(ctx, p.ID, "$hash$one", keep, t0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SessionByToken(ctx, []byte("keep"), t0); err != nil {
		t.Errorf("kept session gone: %v", err)
	}
	if _, err := s.SessionByToken(ctx, []byte("drop"), t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("other session survived a password change: %v", err)
	}
	if _, err := s.SessionByToken(ctx, []byte("bobs"), t0); err != nil {
		t.Errorf("another parent's session was removed: %v", err)
	}

	// A second call replaces the one local identity rather than adding another.
	if err := s.SetPassword(ctx, p.ID, "$hash$two", "", t0); err != nil {
		t.Fatal(err)
	}
	_, hash, _ := s.ParentByUsername(ctx, "ana")
	var n int
	if err := s.r.QueryRow(`SELECT COUNT(*) FROM identities WHERE parent_id = ?`, p.ID).Scan(&n); err != nil || n != 1 || hash != "$hash$two" {
		t.Fatalf("identities = %d, hash %q, err %v", n, hash, err)
	}
	var provider, subject string
	if err := s.r.QueryRow(`SELECT provider, subject FROM identities WHERE parent_id = ?`, p.ID).Scan(&provider, &subject); err != nil || provider != "local" || subject != "ana" {
		t.Fatalf("identity = %q/%q, %v; want local/ana", provider, subject, err)
	}
	if _, err := s.SessionByToken(ctx, []byte("keep"), t0); !errors.Is(err, ErrNotFound) {
		t.Errorf(`keepSessionID "" must keep none; keep survived: %v`, err)
	}
	if err := s.SetPassword(ctx, "nobody", "$hash", "", t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown parent: %v, want ErrNotFound", err)
	}
}

func TestPasswordChangedAt(t *testing.T) {
	s := openMem(t)
	p := mustParent(t, s, "ana")
	if got, err := s.ParentByID(ctx, p.ID); err != nil || got.PasswordChangedAt != nil {
		t.Fatalf("before any password: %v, %v; want nil", got.PasswordChangedAt, err)
	}
	at := t0.Add(5 * time.Minute)
	if err := s.SetPassword(ctx, p.ID, "$hash$one", "", at); err != nil {
		t.Fatal(err)
	}
	got, err := s.ParentByID(ctx, p.ID)
	if err != nil || got.PasswordChangedAt == nil || !got.PasswordChangedAt.Equal(at) {
		t.Fatalf("after SetPassword: %v, %v; want %v", got.PasswordChangedAt, err, at)
	}
	mustIssue(t, s, model.TokenReset, p.ID, "tok", t0.Add(time.Hour))
	later := t0.Add(10 * time.Minute)
	if _, err := s.RedeemToken(ctx, []byte("tok"), "$hash$two", later); err != nil {
		t.Fatal(err)
	}
	got, _ = s.ParentByID(ctx, p.ID)
	if got.PasswordChangedAt == nil || !got.PasswordChangedAt.Equal(later) {
		t.Fatalf("after RedeemToken: %v; want %v", got.PasswordChangedAt, later)
	}
}

func TestSetParentDisabled(t *testing.T) {
	s := openMem(t)
	p := mustParent(t, s, "ana")
	other := mustParent(t, s, "bob")
	exp := t0.Add(time.Hour)
	mustSession(t, s, "parent", p.ID, []byte("a1"), exp)
	mustSession(t, s, "parent", other.ID, []byte("b1"), exp)

	later := t0.Add(time.Minute)
	if err := s.SetParentDisabled(ctx, p.ID, true, later); err != nil {
		t.Fatal(err)
	}
	got, _, _ := s.ParentByUsername(ctx, "ana")
	if !got.Disabled {
		t.Error("parent not disabled")
	}
	if _, err := s.SessionByToken(ctx, []byte("a1"), t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("disabled parent's session survived: %v", err)
	}
	if _, err := s.SessionByToken(ctx, []byte("b1"), t0); err != nil {
		t.Errorf("another parent's session was removed: %v", err)
	}
	var updated int64
	if err := s.r.QueryRow(`SELECT updated_at FROM parents WHERE id = ?`, p.ID).Scan(&updated); err != nil || updated != later.Unix() {
		t.Errorf("updated_at = %d, %v; want %d", updated, err, later.Unix())
	}

	if err := s.SetParentDisabled(ctx, p.ID, false, later); err != nil {
		t.Fatal(err)
	}
	got, _, _ = s.ParentByUsername(ctx, "ana")
	if got.Disabled {
		t.Error("parent still disabled after enable")
	}
	if err := s.SetParentDisabled(ctx, "nobody", true, later); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown parent: %v, want ErrNotFound", err)
	}
}

func TestInstantsAreStoredAsUnixSeconds(t *testing.T) {
	s := openMem(t)
	p := mustParent(t, s, "ana")
	exp := t0.Add(constants.ParentSessionTTL)
	id := mustSession(t, s, "parent", p.ID, []byte("h"), exp)

	var created, updated, sCreated, seen, expires int64
	if err := s.r.QueryRow(`SELECT created_at, updated_at FROM parents WHERE id = ?`, p.ID).Scan(&created, &updated); err != nil {
		t.Fatal(err)
	}
	if err := s.r.QueryRow(`SELECT created_at, last_seen_at, expires_at FROM sessions WHERE id = ?`, id).Scan(&sCreated, &seen, &expires); err != nil {
		t.Fatal(err)
	}
	for name, got := range map[string]int64{"parents.created_at": created, "parents.updated_at": updated, "sessions.created_at": sCreated, "sessions.last_seen_at": seen} {
		if got != t0.Unix() {
			t.Errorf("%s = %d, want %d (unix seconds)", name, got, t0.Unix())
		}
	}
	if expires != exp.Unix() {
		t.Errorf("sessions.expires_at = %d, want %d (unix seconds)", expires, exp.Unix())
	}
	if p.CreatedAt != time.Unix(created, 0).UTC() {
		t.Errorf("CreatedAt = %v, want %v", p.CreatedAt, time.Unix(created, 0).UTC())
	}
}

func TestListSessions(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	bob := mustParent(t, s, "bob")
	old := mustSession(t, s, "parent", ana.ID, []byte("old"), t0.Add(2*time.Hour))
	if err := s.TouchSession(ctx, old, t0.Add(30*time.Minute), t0.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateSession(ctx, "parent", ana.ID, []byte("new"), t0.Add(time.Hour), t0.Add(3*time.Hour), ""); err != nil {
		t.Fatal(err)
	}
	mustSession(t, s, "parent", ana.ID, []byte("gone"), t0.Add(time.Minute))
	mustSession(t, s, "parent", bob.ID, []byte("bobs"), t0.Add(2*time.Hour))
	mustSession(t, s, "admin", mustAdmin(t, s).ID, []byte("admin"), t0.Add(2*time.Hour))

	now := t0.Add(time.Hour)
	got, err := s.ListSessions(ctx, ana.ID, now)
	if err != nil || len(got) != 2 {
		t.Fatalf("ListSessions = %+v, %v; want ana's two live sessions", got, err)
	}
	if !got[0].CreatedAt.Equal(t0.Add(time.Hour)) || got[1].ID != old {
		t.Errorf("order = %+v; want newest first", got)
	}
	if !got[1].CreatedAt.Equal(t0) || !got[1].LastSeenAt.Equal(t0.Add(30*time.Minute)) || got[1].Current {
		t.Errorf("old session = %+v", got[1])
	}
	if got, err := s.ListSessions(ctx, "nobody", now); err != nil || got == nil || len(got) != 0 {
		t.Errorf("no sessions = %#v, %v; want a non-nil empty slice", got, err)
	}
}

func TestDeleteParentSession(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	bob := mustParent(t, s, "bob")
	exp := t0.Add(time.Hour)
	mine := mustSession(t, s, "parent", ana.ID, []byte("mine"), exp)
	theirs := mustSession(t, s, "parent", bob.ID, []byte("theirs"), exp)

	if err := s.DeleteParentSession(ctx, ana.ID, theirs); !errors.Is(err, ErrNotFound) {
		t.Errorf("another parent's session: %v, want ErrNotFound", err)
	}
	if _, err := s.SessionByToken(ctx, []byte("theirs"), t0); err != nil {
		t.Errorf("another parent's session was removed: %v", err)
	}
	if err := s.DeleteParentSession(ctx, ana.ID, "no-such-session"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown session: %v, want ErrNotFound", err)
	}
	if err := s.DeleteParentSession(ctx, ana.ID, mine); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SessionByToken(ctx, []byte("mine"), t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("deleted session still resolves: %v", err)
	}
}

func TestParentActivity(t *testing.T) {
	s := openMem(t)
	ana, bob := mustParent(t, s, "ana"), mustParent(t, s, "bob")
	mustParent(t, s, "cy")
	exp := t0.Add(time.Hour)
	mustSession(t, s, "parent", ana.ID, []byte("a1"), exp)
	second := mustSession(t, s, "parent", ana.ID, []byte("a2"), exp)
	mustSession(t, s, "parent", bob.ID, []byte("b1"), exp)
	if _, err := s.w.Exec(`UPDATE sessions SET last_seen_at = ? WHERE id = ?`, t0.Add(30*time.Minute).Unix(), second); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateSession(ctx, "admin", mustAdmin(t, s).ID, []byte("adm"), t0.Add(time.Hour), exp.Add(time.Hour), ""); err != nil {
		t.Fatal(err)
	}
	got, err := s.ParentActivity(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || !got[ana.ID].Equal(t0.Add(30*time.Minute)) || !got[bob.ID].Equal(t0) {
		t.Errorf("activity = %v; want ana at +30m, bob at t0, none for cy and none for the admin", got)
	}
	if err := s.SetParentDisabled(ctx, ana.ID, true, t0); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.ParentActivity(ctx); len(got) != 1 {
		t.Errorf("after disabling ana: %v", got)
	}
}

func TestStampReauth(t *testing.T) {
	s := openMem(t)
	p := mustParent(t, s, "ana")
	id, err := s.CreateSession(ctx, "parent", p.ID, []byte("tok"), t0, t0.Add(time.Hour), "")
	if err != nil {
		t.Fatal(err)
	}
	other, err := s.CreateSession(ctx, "parent", p.ID, []byte("tok-2"), t0, t0.Add(time.Hour), "")
	if err != nil {
		t.Fatal(err)
	}
	if got, err := s.SessionByToken(ctx, []byte("tok"), t0); err != nil || !got.ReauthAt.IsZero() {
		t.Fatalf("before: %+v, %v; want a zero ReauthAt", got, err)
	}
	at := t0.Add(90*time.Second + 500*time.Millisecond)
	if err := s.StampReauth(ctx, id, at); err != nil {
		t.Fatal(err)
	}
	if n := scalar(t, s, `SELECT reauth_at FROM sessions WHERE id = ?`, id); int64(n) != at.Unix() {
		t.Errorf("reauth_at = %d, want %d", n, at.Unix())
	}
	if got, err := s.SessionByToken(ctx, []byte("tok"), t0); err != nil || !got.ReauthAt.Equal(time.Unix(at.Unix(), 0)) || got.ReauthAt.Location() != time.UTC {
		t.Errorf("after: %+v, %v; want ReauthAt %v UTC", got, err, time.Unix(at.Unix(), 0).UTC())
	}
	if n := scalar(t, s, `SELECT COUNT(*) FROM sessions WHERE id = ? AND reauth_at IS NULL`, other); n != 1 {
		t.Error("the stamp reached another session")
	}
}
