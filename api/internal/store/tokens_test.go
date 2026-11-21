package store

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
)

func mustIssue(t *testing.T, s *Store, kind, parentID, hash string, expires time.Time) {
	t.Helper()
	if err := s.IssueToken(ctx, kind, parentID, []byte(hash), expires); err != nil {
		t.Fatal(err)
	}
}

func TestPeekToken(t *testing.T) {
	s := openMem(t)
	p := mustParent(t, s, "ana")
	exp := t0.Add(constants.InviteTTL)
	mustIssue(t, s, model.TokenInvite, p.ID, "tok", exp)

	kind, username, err := s.PeekToken(ctx, []byte("tok"), t0)
	if err != nil || kind != model.TokenInvite || username != "ana" {
		t.Fatalf("PeekToken = %q %q %v", kind, username, err)
	}
	if _, _, err := s.PeekToken(ctx, []byte("tok"), exp.Add(-time.Second)); err != nil {
		t.Errorf("1s before expiry: %v", err)
	}
	if _, _, err := s.PeekToken(ctx, []byte("tok"), exp); !errors.Is(err, ErrNotFound) {
		t.Errorf("at expiry: %v, want ErrNotFound", err)
	}
	if _, _, err := s.PeekToken(ctx, []byte("unknown"), t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown token: %v, want ErrNotFound", err)
	}

	if err := s.SetParentDisabled(ctx, p.ID, true, t0); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.PeekToken(ctx, []byte("tok"), t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("disabled parent: %v, want ErrNotFound", err)
	}
	if err := s.SetParentDisabled(ctx, p.ID, false, t0); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.PeekToken(ctx, []byte("tok"), t0); err != nil {
		t.Errorf("re-enabled parent: %v", err)
	}
	if _, err := s.RedeemToken(ctx, []byte("tok"), "$hash", t0); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.PeekToken(ctx, []byte("tok"), t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("used token: %v, want ErrNotFound", err)
	}
}

func TestIssueTokenReplacesUnusedOfTheSameKind(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	bob := mustParent(t, s, "bob")
	exp := t0.Add(time.Hour)
	mustIssue(t, s, model.TokenInvite, ana.ID, "ana-invite-1", exp)
	mustIssue(t, s, model.TokenReset, ana.ID, "ana-reset", exp)
	mustIssue(t, s, model.TokenInvite, bob.ID, "bob-invite", exp)
	mustIssue(t, s, model.TokenInvite, ana.ID, "ana-invite-2", exp)

	for hash, want := range map[string]bool{"ana-invite-1": false, "ana-invite-2": true, "ana-reset": true, "bob-invite": true} {
		_, _, err := s.PeekToken(ctx, []byte(hash), t0)
		if (err == nil) != want {
			t.Errorf("%s: err = %v, want usable = %v", hash, err, want)
		}
	}

	// A used token is a record, not a pending link: replacing leaves it alone.
	if _, err := s.RedeemToken(ctx, []byte("ana-invite-2"), "$h", t0); err != nil {
		t.Fatal(err)
	}
	mustIssue(t, s, model.TokenInvite, ana.ID, "ana-invite-3", exp)
	var n int
	if err := s.r.QueryRow(`SELECT COUNT(*) FROM one_time_tokens WHERE parent_id = ? AND kind = 'invite' AND used_at IS NOT NULL`, ana.ID).Scan(&n); err != nil || n != 1 {
		t.Fatalf("used invites kept = %d, %v; want 1", n, err)
	}
}

func TestIssueTokenStoresOnlyTheHash(t *testing.T) {
	s := openMem(t)
	p := mustParent(t, s, "ana")
	exp := t0.Add(constants.ResetTTL)
	mustIssue(t, s, model.TokenReset, p.ID, "the-hash", exp)
	var hash []byte
	var expires int64
	var used *int64
	if err := s.r.QueryRow(`SELECT token_hash, expires_at, used_at FROM one_time_tokens WHERE parent_id = ?`, p.ID).Scan(&hash, &expires, &used); err != nil {
		t.Fatal(err)
	}
	if string(hash) != "the-hash" || expires != exp.Unix() || used != nil {
		t.Fatalf("row = %q %d %v; want the-hash %d NULL (unix seconds)", hash, expires, used, exp.Unix())
	}
}

func TestRedeemToken(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	bob := mustParent(t, s, "bob")
	exp := t0.Add(time.Hour)
	mustSession(t, s, "parent", ana.ID, []byte("a1"), exp)
	mustSession(t, s, "parent", ana.ID, []byte("a2"), exp)
	mustSession(t, s, "parent", bob.ID, []byte("b1"), exp)
	mustIssue(t, s, model.TokenReset, ana.ID, "tok", exp)

	later := t0.Add(time.Minute)
	id, err := s.RedeemToken(ctx, []byte("tok"), "$hash$one", later)
	if err != nil || id != ana.ID {
		t.Fatalf("RedeemToken = %q, %v; want %q", id, err, ana.ID)
	}
	if _, hash, _ := s.ParentByUsername(ctx, "ana"); hash != "$hash$one" {
		t.Errorf("stored hash = %q", hash)
	}
	for _, h := range []string{"a1", "a2"} {
		if _, err := s.SessionByToken(ctx, []byte(h), t0); !errors.Is(err, ErrNotFound) {
			t.Errorf("session %s survived the redemption: %v", h, err)
		}
	}
	if _, err := s.SessionByToken(ctx, []byte("b1"), t0); err != nil {
		t.Errorf("another parent's session was removed: %v", err)
	}
	var used int64
	if err := s.r.QueryRow(`SELECT used_at FROM one_time_tokens WHERE token_hash = ?`, []byte("tok")).Scan(&used); err != nil || used != later.Unix() {
		t.Errorf("used_at = %d, %v; want %d", used, err, later.Unix())
	}
	if _, err := s.RedeemToken(ctx, []byte("tok"), "$hash$two", later); !errors.Is(err, ErrNotFound) {
		t.Errorf("second redemption: %v, want ErrNotFound", err)
	}
	if _, hash, _ := s.ParentByUsername(ctx, "ana"); hash != "$hash$one" {
		t.Errorf("second redemption changed the hash to %q", hash)
	}
}

func TestRedeemTokenRefusalsLeaveTheTokenUsable(t *testing.T) {
	s := openMem(t)
	p := mustParent(t, s, "ana")
	exp := t0.Add(time.Hour)
	mustIssue(t, s, model.TokenInvite, p.ID, "tok", exp)

	if _, err := s.RedeemToken(ctx, []byte("unknown"), "$h", t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown: %v", err)
	}
	if _, err := s.RedeemToken(ctx, []byte("tok"), "$h", exp); !errors.Is(err, ErrNotFound) {
		t.Errorf("at expiry: %v", err)
	}
	if err := s.SetParentDisabled(ctx, p.ID, true, t0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RedeemToken(ctx, []byte("tok"), "$h", t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("disabled parent: %v", err)
	}
	if _, hash, _ := s.ParentByUsername(ctx, "ana"); hash != "" {
		t.Errorf("a refused redemption set a password: %q", hash)
	}
	if err := s.SetParentDisabled(ctx, p.ID, false, t0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RedeemToken(ctx, []byte("tok"), "$h", exp.Add(-time.Second)); err != nil {
		t.Errorf("refusals burned the token: %v", err)
	}
}

func TestRedeemTokenRace(t *testing.T) {
	s := openMem(t)
	const rounds, workers = 40, 8
	exp := t0.Add(time.Hour)
	for round := 0; round < rounds; round++ {
		p := mustParent(t, s, fmt.Sprintf("user%d", round))
		mustSession(t, s, "parent", p.ID, []byte(fmt.Sprintf("sess%d", round)), exp)
		tok := []byte(fmt.Sprintf("tok%d", round))
		mustIssue(t, s, model.TokenInvite, p.ID, string(tok), exp)

		var wg sync.WaitGroup
		results := make([]error, workers)
		start := make(chan struct{})
		for w := 0; w < workers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				_, results[w] = s.RedeemToken(ctx, tok, fmt.Sprintf("$hash$%d", w), t0)
			}()
		}
		close(start)
		wg.Wait()

		winner := -1
		for w, err := range results {
			switch {
			case err == nil && winner == -1:
				winner = w
			case err == nil:
				t.Fatalf("round %d: workers %d and %d both redeemed", round, winner, w)
			case !errors.Is(err, ErrNotFound):
				t.Fatalf("round %d worker %d: %v, want ErrNotFound", round, w, err)
			}
		}
		if winner == -1 {
			t.Fatalf("round %d: nobody redeemed", round)
		}
		if hash := localHash(t, s, p.ID); hash != fmt.Sprintf("$hash$%d", winner) {
			t.Fatalf("round %d: stored hash %q is not the winner's (%d)", round, hash, winner)
		}
	}
}

func localHash(t *testing.T, s *Store, parentID string) string {
	t.Helper()
	var hash string
	if err := s.r.QueryRow(`SELECT secret_hash FROM identities WHERE parent_id = ? AND provider = 'local'`, parentID).Scan(&hash); err != nil {
		t.Fatal(err)
	}
	return hash
}
