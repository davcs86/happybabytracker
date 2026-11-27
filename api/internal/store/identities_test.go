package store

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
)

const testIss = "https://issuer.example"

func mustOIDCIdentity(t *testing.T, s *Store, table, column, principalID, iss, sub string) {
	t.Helper()
	if _, err := s.w.Exec(`INSERT INTO `+table+`(`+column+`, provider, subject) VALUES (?, ?, ?)`, principalID, iss, sub); err != nil {
		t.Fatal(err)
	}
}

func sessionLabel(t *testing.T, s *Store, hash string) (kind string, label sql.NullString) {
	t.Helper()
	if err := s.r.QueryRow(`SELECT kind, device_label FROM sessions WHERE token_hash = ?`, []byte(hash)).Scan(&kind, &label); err != nil {
		t.Fatal(err)
	}
	return kind, label
}

func TestOIDCLoginParentAndAdmin(t *testing.T) {
	s := openMem(t)
	p := mustParent(t, s, "ana")
	a := mustAdmin(t, s)
	mustOIDCIdentity(t, s, "identities", "parent_id", p.ID, testIss, "sub-ana")
	mustOIDCIdentity(t, s, "admin_identities", "admin_id", a.ID, testIss, "sub-admin")
	pExp, aExp := t0.Add(30*24*time.Hour), t0.Add(12*time.Hour)

	kind, exp, err := s.OIDCLogin(ctx, testIss, "sub-ana", []byte("p-tok"), t0, pExp, aExp, "Firefox on Mac")
	if err != nil || kind != "parent" || !exp.Equal(pExp) {
		t.Fatalf("parent: %q %v %v", kind, exp, err)
	}
	got, err := s.SessionByToken(ctx, []byte("p-tok"), t0)
	if err != nil || got.ParentID != p.ID || got.AdminID != "" {
		t.Fatalf("parent session = %+v, %v", got, err)
	}
	if _, label := sessionLabel(t, s, "p-tok"); label.String != "Firefox on Mac" {
		t.Errorf("parent label = %v", label)
	}

	kind, exp, err = s.OIDCLogin(ctx, testIss, "sub-admin", []byte("a-tok"), t0, pExp, aExp, "Firefox on Mac")
	if err != nil || kind != "admin" || !exp.Equal(aExp) {
		t.Fatalf("admin: %q %v %v", kind, exp, err)
	}
	got, err = s.SessionByToken(ctx, []byte("a-tok"), t0)
	if err != nil || got.AdminID != a.ID || got.ParentID != "" {
		t.Fatalf("admin session = %+v, %v", got, err)
	}
	if _, label := sessionLabel(t, s, "a-tok"); label.Valid {
		t.Errorf("admin label = %v, want NULL", label)
	}

	if _, _, err := s.OIDCLogin(ctx, testIss, "sub-unknown", []byte("u-tok"), t0, pExp, aExp, ""); !errors.Is(err, ErrNotLinked) {
		t.Errorf("unknown pair: %v, want ErrNotLinked", err)
	}
	if _, _, err := s.OIDCLogin(ctx, "https://other.example", "sub-ana", []byte("o-tok"), t0, pExp, aExp, ""); !errors.Is(err, ErrNotLinked) {
		t.Errorf("another issuer: %v, want ErrNotLinked", err)
	}
	if n := scalar(t, s, `SELECT COUNT(*) FROM sessions`); n != 2 {
		t.Errorf("%d sessions, want 2", n)
	}
}

func TestOIDCLoginNeverMatchesALocalRow(t *testing.T) {
	s := openMem(t)
	p := mustParent(t, s, "ana")
	if err := s.SetPassword(ctx, p.ID, "h", "", t0); err != nil {
		t.Fatal(err)
	}
	mustAdmin(t, s)
	for _, sub := range []string{"ana", "root-admin"} {
		if _, _, err := s.OIDCLogin(ctx, "local", sub, []byte("tok-"+sub), t0, t0.Add(time.Hour), t0.Add(time.Hour), ""); !errors.Is(err, ErrNotLinked) {
			t.Errorf("(local, %s): %v, want ErrNotLinked", sub, err)
		}
	}
	if n := scalar(t, s, `SELECT COUNT(*) FROM sessions`); n != 0 {
		t.Errorf("%d sessions, want 0", n)
	}
}

func TestOIDCLoginDisabledParentPurges(t *testing.T) {
	s := openMem(t)
	p := mustParent(t, s, "ana")
	other := mustParent(t, s, "bob")
	mustOIDCIdentity(t, s, "identities", "parent_id", p.ID, testIss, "sub-ana")
	if err := s.SetParentDisabled(ctx, p.ID, true, t0); err != nil {
		t.Fatal(err)
	}
	exp := t0.Add(time.Hour)
	mustSession(t, s, "parent", p.ID, []byte("a1"), exp)
	mustSession(t, s, "parent", p.ID, []byte("a2"), exp)
	mustSession(t, s, "parent", other.ID, []byte("b1"), exp)
	mustDevice(t, s, other.ID, "https://push.example.com/b")
	for q, args := range map[string][]any{
		`INSERT INTO push_subscriptions(id, parent_id, endpoint, p256dh, auth, label, created_at, updated_at) VALUES (?, ?, ?, 'k', 'a', 'Phone', 100, 100)`: {model.NewID(), p.ID, "https://push.example.com/a"},
		`INSERT INTO ntfy_topics(parent_id, topic, created_at) VALUES (?, 'hbt-anatopic', 0)`:                                                                {p.ID},
	} {
		if _, err := s.w.Exec(q, args...); err != nil {
			t.Fatal(err)
		}
	}

	for i := range 2 {
		if _, _, err := s.OIDCLogin(ctx, testIss, "sub-ana", []byte("new"), t0, exp, exp, ""); !errors.Is(err, ErrNotLinked) {
			t.Fatalf("call %d: %v, want ErrNotLinked", i, err)
		}
		for q, want := range map[string]int{
			`SELECT COUNT(*) FROM sessions WHERE parent_id = ?`:           0,
			`SELECT COUNT(*) FROM push_subscriptions WHERE parent_id = ?`: 0,
			`SELECT COUNT(*) FROM ntfy_topics WHERE parent_id = ?`:        0,
		} {
			if n := scalar(t, s, q, p.ID); n != want {
				t.Errorf("call %d: %s = %d, want %d", i, q, n, want)
			}
		}
	}
	if n := scalar(t, s, `SELECT COUNT(*) FROM sessions WHERE parent_id = ?`, other.ID); n != 1 {
		t.Errorf("the other parent's sessions = %d, want 1", n)
	}
	if n := scalar(t, s, `SELECT COUNT(*) FROM push_subscriptions WHERE parent_id = ?`, other.ID); n != 1 {
		t.Errorf("the other parent's devices = %d, want 1", n)
	}
}

func TestRedeemInviteOIDC(t *testing.T) {
	s := openMem(t)
	p := mustParent(t, s, "ana")
	mustIssue(t, s, model.TokenInvite, p.ID, "tok", t0.Add(time.Hour))
	later := t0.Add(time.Minute)
	exp := later.Add(30 * 24 * time.Hour)
	if err := s.RedeemInviteOIDC(ctx, []byte("tok"), testIss, "sub-ana", []byte("sess"), later, exp, "Firefox on Mac"); err != nil {
		t.Fatal(err)
	}
	got, err := s.SessionByToken(ctx, []byte("sess"), later)
	if err != nil || got.ParentID != p.ID || !got.ExpiresAt.Equal(exp) {
		t.Fatalf("session = %+v, %v", got, err)
	}
	if kind, label := sessionLabel(t, s, "sess"); kind != "parent" || label.String != "Firefox on Mac" {
		t.Errorf("session = %s %v", kind, label)
	}
	var hash sql.NullString
	if err := s.r.QueryRow(`SELECT secret_hash FROM identities WHERE parent_id = ? AND provider = ? AND subject = 'sub-ana'`, p.ID, testIss).Scan(&hash); err != nil || hash.Valid {
		t.Errorf("identity secret_hash = %v, %v; want a row with NULL", hash, err)
	}
	if n := scalar(t, s, `SELECT used_at FROM one_time_tokens WHERE token_hash = ?`, []byte("tok")); n != int(later.Unix()) {
		t.Errorf("used_at = %d, want %d", n, later.Unix())
	}
	if kind, _, err := s.OIDCLogin(ctx, testIss, "sub-ana", []byte("login"), later, exp, exp, ""); err != nil || kind != "parent" {
		t.Errorf("login after the redemption: %q %v", kind, err)
	}
	if err := s.RedeemInviteOIDC(ctx, []byte("tok"), "https://other.example", "sub-2", []byte("sess2"), later, exp, ""); !errors.Is(err, ErrNotFound) {
		t.Errorf("second redemption: %v, want ErrNotFound", err)
	}
}

func TestRedeemInviteOIDCRefusals(t *testing.T) {
	s := openMem(t)
	p := mustParent(t, s, "ana")
	exp := t0.Add(time.Hour)
	redeem := func(tok string, now time.Time) error {
		return s.RedeemInviteOIDC(ctx, []byte(tok), testIss, "sub-ana", []byte("sess-"+tok), now, now.Add(time.Hour), "")
	}

	mustIssue(t, s, model.TokenReset, p.ID, "reset", exp)
	if err := redeem("reset", t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("reset token: %v, want ErrNotFound", err)
	}
	if err := redeem("unknown", t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown token: %v, want ErrNotFound", err)
	}
	mustIssue(t, s, model.TokenInvite, p.ID, "old", exp)
	mustIssue(t, s, model.TokenInvite, p.ID, "tok", exp)
	if err := redeem("old", t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("replaced token: %v, want ErrNotFound", err)
	}
	if err := redeem("tok", exp); !errors.Is(err, ErrNotFound) {
		t.Errorf("expired token: %v, want ErrNotFound", err)
	}
	if err := s.SetParentDisabled(ctx, p.ID, true, t0); err != nil {
		t.Fatal(err)
	}
	if err := redeem("tok", t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("disabled parent: %v, want ErrNotFound", err)
	}
	if err := s.SetParentDisabled(ctx, p.ID, false, t0); err != nil {
		t.Fatal(err)
	}
	if n := scalar(t, s, `SELECT COUNT(*) FROM identities WHERE provider <> 'local'`) + scalar(t, s, `SELECT COUNT(*) FROM sessions`); n != 0 {
		t.Fatalf("a refusal wrote %d identity or session rows", n)
	}
	if err := redeem("tok", t0); err != nil {
		t.Fatalf("the replacement after the refusals: %v", err)
	}
	if err := redeem("tok", t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("used token: %v, want ErrNotFound", err)
	}
}

func TestRedeemInviteOIDCAlreadyLinked(t *testing.T) {
	for name, setup := range map[string]func(t *testing.T, s *Store, ana model.Parent){
		"another parent holds the pair": func(t *testing.T, s *Store, _ model.Parent) {
			mustOIDCIdentity(t, s, "identities", "parent_id", mustParent(t, s, "bob").ID, testIss, "sub-x")
		},
		"the admin holds the pair": func(t *testing.T, s *Store, _ model.Parent) {
			mustOIDCIdentity(t, s, "admin_identities", "admin_id", mustAdmin(t, s).ID, testIss, "sub-x")
		},
		"the parent already has this issuer": func(t *testing.T, s *Store, ana model.Parent) {
			mustOIDCIdentity(t, s, "identities", "parent_id", ana.ID, testIss, "sub-old")
		},
	} {
		t.Run(name, func(t *testing.T) {
			s := openMem(t)
			ana := mustParent(t, s, "ana")
			mustIssue(t, s, model.TokenInvite, ana.ID, "tok", t0.Add(time.Hour))
			setup(t, s, ana)
			if err := s.RedeemInviteOIDC(ctx, []byte("tok"), testIss, "sub-x", []byte("sess"), t0, t0.Add(time.Hour), ""); !errors.Is(err, ErrAlreadyLinked) {
				t.Fatalf("RedeemInviteOIDC: %v, want ErrAlreadyLinked", err)
			}
			if n := scalar(t, s, `SELECT COUNT(*) FROM sessions`); n != 0 {
				t.Errorf("%d sessions after the refusal", n)
			}
			if err := s.RedeemInviteOIDC(ctx, []byte("tok"), "https://other.example", "sub-x", []byte("sess"), t0, t0.Add(time.Hour), ""); err != nil {
				t.Errorf("the token after the refusal: %v", err)
			}
		})
	}
}

func TestRedeemInviteOIDCEndsOtherSessions(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	bob := mustParent(t, s, "bob")
	mustOIDCIdentity(t, s, "identities", "parent_id", bob.ID, testIss, "sub-bob")
	exp := t0.Add(time.Hour)
	mustSession(t, s, "parent", ana.ID, []byte("a1"), exp)
	mustSession(t, s, "parent", ana.ID, []byte("a2"), exp)
	mustSession(t, s, "parent", bob.ID, []byte("b1"), exp)
	mustIssue(t, s, model.TokenInvite, ana.ID, "tok", exp)

	if err := s.RedeemInviteOIDC(ctx, []byte("tok"), testIss, "sub-bob", []byte("new"), t0, exp, ""); !errors.Is(err, ErrAlreadyLinked) {
		t.Fatalf("refused call: %v, want ErrAlreadyLinked", err)
	}
	for _, h := range []string{"a1", "a2", "b1"} {
		if !sessionAlive(t, s, h) {
			t.Errorf("the refused call ended session %s", h)
		}
	}

	if err := s.RedeemInviteOIDC(ctx, []byte("tok"), testIss, "sub-ana", []byte("new"), t0, exp, ""); err != nil {
		t.Fatal(err)
	}
	if n := scalar(t, s, `SELECT COUNT(*) FROM sessions WHERE parent_id = ?`, ana.ID); n != 1 || !sessionAlive(t, s, "new") {
		t.Errorf("ana holds %d sessions, want exactly the new one", n)
	}
	if !sessionAlive(t, s, "b1") {
		t.Error("the other parent's session was ended")
	}
}

func TestPairBelongsToOnePrincipal(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	bob := mustParent(t, s, "bob")
	a := mustAdmin(t, s)
	mustIssue(t, s, model.TokenInvite, ana.ID, "ana-tok", t0.Add(time.Hour))
	mustIssue(t, s, model.TokenInvite, bob.ID, "bob-tok", t0.Add(time.Hour))
	if err := s.RedeemInviteOIDC(ctx, []byte("ana-tok"), testIss, "sub-x", []byte("s1"), t0, t0.Add(time.Hour), ""); err != nil {
		t.Fatal(err)
	}
	if err := s.RedeemInviteOIDC(ctx, []byte("bob-tok"), testIss, "sub-x", []byte("s2"), t0, t0.Add(time.Hour), ""); !errors.Is(err, ErrAlreadyLinked) {
		t.Errorf("second parent: %v, want ErrAlreadyLinked", err)
	}
	if _, err := s.w.Exec(`INSERT INTO identities(parent_id, provider, subject) VALUES (?, ?, 'sub-x')`, bob.ID, testIss); err == nil {
		t.Error("the identities primary key accepted the pair twice")
	}
	if _, err := s.w.Exec(`INSERT INTO admin_identities(admin_id, provider, subject) VALUES (?, ?, 'sub-y')`, a.ID, testIss); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		iss, sub string
		want     bool
	}{{testIss, "sub-x", true}, {testIss, "sub-y", true}, {testIss, "sub-z", false}, {"local", "ana", false}, {"local", "root-admin", false}} {
		var got bool
		err := s.inTx(ctx, func(tx *sql.Tx) (err error) {
			got, err = pairTaken(ctx, tx, c.iss, c.sub)
			return err
		})
		if err != nil || got != c.want {
			t.Errorf("pairTaken(%s, %s) = %v, %v; want %v", c.iss, c.sub, got, err, c.want)
		}
	}
	if err := s.RedeemInviteOIDC(ctx, []byte("bob-tok"), testIss, "sub-y", []byte("s2"), t0, t0.Add(time.Hour), ""); !errors.Is(err, ErrAlreadyLinked) {
		t.Errorf("the admin's pair for a parent: %v, want ErrAlreadyLinked", err)
	}
}
