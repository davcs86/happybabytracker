package store

import (
	"errors"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
)

const testPrefix = "hbt-test"

var topicRE = regexp.MustCompile(`^hbt-test-[0-9A-Za-z]{22}$`)

func mustTopic(t *testing.T, s *Store, parentID string) model.NtfyTopic {
	t.Helper()
	tp, created, err := s.SetNtfyTopic(ctx, parentID, testPrefix, false, t0)
	if err != nil || !created {
		t.Fatalf("set topic: created=%v err=%v", created, err)
	}
	return tp
}

func ntfyTargets(t *testing.T, s *Store, kidID string) []NtfyTarget {
	t.Helper()
	var out []NtfyTarget
	if err := s.inReadTx(ctx, func(q querier) error {
		var err error
		out, err = NtfyTargets(ctx, q, kidID)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestNtfyTopicShapeAndUniqueness(t *testing.T) {
	const draws = 5000
	seen := map[string]bool{}
	counts := map[rune]int{}
	for range draws {
		tp := newNtfyTopic(testPrefix)
		if !topicRE.MatchString(tp) {
			t.Fatalf("topic %q is not <prefix>-<22 base62>", tp)
		}
		if seen[tp] {
			t.Fatalf("topic drawn twice")
		}
		seen[tp] = true
		for _, r := range strings.TrimPrefix(tp, testPrefix+"-") {
			counts[r]++
		}
	}
	if len(counts) != len(base62) {
		t.Fatalf("%d distinct characters, want %d", len(counts), len(base62))
	}
	// A plain byte%62 gives the first 8 characters 25% more draws; 12% is about 5 sigma here.
	want := float64(draws*constants.NtfyTopicRandomLen) / float64(len(base62))
	for r, n := range counts {
		if f := float64(n); f < want*0.88 || f > want*1.12 {
			t.Errorf("character %q drawn %d times, want about %.0f", r, n, want)
		}
	}
}

func TestSetNtfyTopic(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	if _, err := s.NtfyTopic(ctx, ana.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("no topic yet: %v", err)
	}
	if _, _, err := s.SetNtfyTopic(ctx, ana.ID, testPrefix, true, t0); !errors.Is(err, ErrNotFound) {
		t.Fatalf("regenerate without a topic: %v", err)
	}
	first := mustTopic(t, s, ana.ID)
	if !topicRE.MatchString(first.Topic) || first.Kids == nil || len(first.Kids) != 0 {
		t.Fatalf("first = %+v", first)
	}
	again, created, err := s.SetNtfyTopic(ctx, ana.ID, testPrefix, false, t0)
	if err != nil || created || again.Topic != first.Topic {
		t.Fatalf("turning on twice changed the topic: created=%v err=%v", created, err)
	}
	got, err := s.NtfyTopic(ctx, ana.ID)
	if err != nil || got.Topic != first.Topic {
		t.Fatalf("read: %v", err)
	}
}

func TestSetNtfyTopicRefusesADisabledParent(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	if _, err := s.w.Exec(`UPDATE parents SET disabled = 1 WHERE id = ?`, ana.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.SetNtfyTopic(ctx, ana.ID, testPrefix, false, t0); !errors.Is(err, ErrNotFound) {
		t.Fatalf("disabled parent: %v", err)
	}
}

func TestRegenerateKeepsOptins(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	kid := mustKid(t, s, ana.ID, "Mia")
	first := mustTopic(t, s, ana.ID)
	if err := s.SetNtfyOptin(ctx, kid.ID, ana.ID); err != nil {
		t.Fatal(err)
	}
	next, created, err := s.SetNtfyTopic(ctx, ana.ID, testPrefix, true, t0)
	if err != nil || created {
		t.Fatalf("regenerate: created=%v err=%v", created, err)
	}
	if next.Topic == first.Topic || !topicRE.MatchString(next.Topic) {
		t.Fatalf("regenerate kept or broke the topic")
	}
	if !slices.Equal(next.Kids, []string{kid.ID}) {
		t.Fatalf("kids after regenerate = %v", next.Kids)
	}
	if n := scalar(t, s, `SELECT COUNT(*) FROM ntfy_topics WHERE topic = ?`, first.Topic); n != 0 {
		t.Fatalf("old topic still stored")
	}
}

func TestDeleteNtfyTopicDeletesOptins(t *testing.T) {
	s := openMem(t)
	ana, ben := mustParent(t, s, "ana"), mustParent(t, s, "ben")
	kid := mustKid(t, s, ana.ID, "Mia")
	mustGrant(t, s, kid.ID, ana.ID, "ben", model.RoleCaregiver)
	mustTopic(t, s, ana.ID)
	mustTopic(t, s, ben.ID)
	for _, p := range []string{ana.ID, ben.ID} {
		if err := s.SetNtfyOptin(ctx, kid.ID, p); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.DeleteNtfyTopic(ctx, ana.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteNtfyTopic(ctx, ana.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second delete: %v", err)
	}
	if n := scalar(t, s, `SELECT COUNT(*) FROM ntfy_kid_optins WHERE parent_id = ?`, ana.ID); n != 0 {
		t.Fatalf("opt-ins survived the topic")
	}
	if got, err := s.NtfyTopic(ctx, ben.ID); err != nil || !slices.Equal(got.Kids, []string{kid.ID}) {
		t.Fatalf("ben's topic touched: %+v %v", got, err)
	}
}

func TestNtfyOptins(t *testing.T) {
	s := openMem(t)
	ana, ben, cid := mustParent(t, s, "ana"), mustParent(t, s, "ben"), mustParent(t, s, "cid")
	kid := mustKid(t, s, ana.ID, "Mia")
	other := mustKid(t, s, cid.ID, "Leo")
	mustGrant(t, s, kid.ID, ana.ID, "ben", model.RoleCaregiver)
	mustTopic(t, s, ana.ID)

	if err := s.SetNtfyOptin(ctx, kid.ID, ben.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("opt in without a topic: %v", err)
	}
	if err := s.ClearNtfyOptin(ctx, kid.ID, ben.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("opt out without a topic: %v", err)
	}
	for _, id := range []string{other.ID, model.NewID()} {
		if err := s.SetNtfyOptin(ctx, id, ana.ID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("opt in without a grant: %v", err)
		}
		if err := s.ClearNtfyOptin(ctx, id, ana.ID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("opt out without a grant: %v", err)
		}
	}
	for range 2 {
		if err := s.SetNtfyOptin(ctx, kid.ID, ana.ID); err != nil {
			t.Fatal(err)
		}
	}
	mustTopic(t, s, ben.ID)
	if err := s.SetNtfyOptin(ctx, kid.ID, ben.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.NtfyTopic(ctx, ana.ID); !slices.Equal(got.Kids, []string{kid.ID}) {
		t.Fatalf("ana's kids = %v", got.Kids)
	}
	for range 2 {
		if err := s.ClearNtfyOptin(ctx, kid.ID, ana.ID); err != nil {
			t.Fatal(err)
		}
	}
	if got, _ := s.NtfyTopic(ctx, ana.ID); len(got.Kids) != 0 {
		t.Fatalf("after opt out kids = %v", got.Kids)
	}
	if got, _ := s.NtfyTopic(ctx, ben.ID); !slices.Equal(got.Kids, []string{kid.ID}) {
		t.Fatalf("ana's opt-out touched ben's: %v", got.Kids)
	}
}

func TestNtfyTargetsExcludeRevokedDisabledAndNotOptedIn(t *testing.T) {
	s := openMem(t)
	ana, ben, cid, dan := mustParent(t, s, "ana"), mustParent(t, s, "ben"), mustParent(t, s, "cid"), mustParent(t, s, "dan")
	kid := mustKid(t, s, ana.ID, "Mia")
	for _, name := range []string{"ben", "cid", "dan"} {
		mustGrant(t, s, kid.ID, ana.ID, name, model.RoleCaregiver)
	}
	topics := map[string]string{}
	for _, p := range []string{ana.ID, ben.ID, cid.ID, dan.ID} {
		topics[p] = mustTopic(t, s, p).Topic
	}
	for _, p := range []string{ana.ID, ben.ID, cid.ID} {
		if err := s.SetNtfyOptin(ctx, kid.ID, p); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.w.Exec(`UPDATE parents SET disabled = 1 WHERE id = ?`, ben.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.w.Exec(`DELETE FROM kid_access WHERE kid_id = ? AND parent_id = ?`, kid.ID, cid.ID); err != nil {
		t.Fatal(err)
	}
	got := ntfyTargets(t, s, kid.ID)
	if want := []NtfyTarget{{ParentID: ana.ID, Topic: topics[ana.ID]}}; !slices.Equal(got, want) {
		t.Fatalf("targets = %+v, want ana only", got)
	}
}

func TestRevokeRemovesOnlyThatKidsNtfyOptin(t *testing.T) {
	s := openMem(t)
	ana, ben := mustParent(t, s, "ana"), mustParent(t, s, "ben")
	kid, other := mustKid(t, s, ana.ID, "Mia"), mustKid(t, s, ana.ID, "Leo")
	mustGrant(t, s, kid.ID, ana.ID, "ben", model.RoleCaregiver)
	mustGrant(t, s, other.ID, ana.ID, "ben", model.RoleCaregiver)
	bt := mustTopic(t, s, ben.ID)
	mustTopic(t, s, ana.ID)
	for _, o := range []struct{ kid, parent string }{{kid.ID, ben.ID}, {other.ID, ben.ID}, {kid.ID, ana.ID}} {
		if err := s.SetNtfyOptin(ctx, o.kid, o.parent); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.RemoveAccess(ctx, kid.ID, ana.ID, ben.ID); err != nil {
		t.Fatal(err)
	}
	got, err := s.NtfyTopic(ctx, ben.ID)
	if err != nil || got.Topic != bt.Topic {
		t.Fatalf("revoke touched ben's topic: %v", err)
	}
	if n := scalar(t, s, `SELECT COUNT(*) FROM ntfy_kid_optins WHERE parent_id = ? AND kid_id = ?`, ben.ID, kid.ID); n != 0 {
		t.Fatalf("ben's opt-in for the revoked kid survived")
	}
	if !slices.Equal(got.Kids, []string{other.ID}) {
		t.Fatalf("ben's other opt-in: %v", got.Kids)
	}
	if n := scalar(t, s, `SELECT COUNT(*) FROM ntfy_kid_optins WHERE parent_id = ?`, ana.ID); n != 1 {
		t.Fatalf("ana's opt-in was removed")
	}
}

func TestLeaveRemovesOnlyThatKidsNtfyOptin(t *testing.T) {
	s := openMem(t)
	ana, ben := mustParent(t, s, "ana"), mustParent(t, s, "ben")
	kid, other := mustKid(t, s, ana.ID, "Mia"), mustKid(t, s, ana.ID, "Leo")
	mustGrant(t, s, kid.ID, ana.ID, "ben", model.RoleCaregiver)
	mustGrant(t, s, other.ID, ana.ID, "ben", model.RoleCaregiver)
	bt := mustTopic(t, s, ben.ID)
	for _, k := range []string{kid.ID, other.ID} {
		if err := s.SetNtfyOptin(ctx, k, ben.ID); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.RemoveAccess(ctx, kid.ID, ben.ID, ben.ID); err != nil {
		t.Fatal(err)
	}
	got, err := s.NtfyTopic(ctx, ben.ID)
	if err != nil || got.Topic != bt.Topic || !slices.Equal(got.Kids, []string{other.ID}) {
		t.Fatalf("after leave: %+v %v", got.Kids, err)
	}
	if n := scalar(t, s, `SELECT COUNT(*) FROM ntfy_kid_optins WHERE parent_id = ? AND kid_id = ?`, ben.ID, kid.ID); n != 0 {
		t.Fatalf("ben's opt-in for the kid he left survived")
	}
}

func TestDisableRemovesNtfyTopic(t *testing.T) {
	s := openMem(t)
	ana, ben := mustParent(t, s, "ana"), mustParent(t, s, "ben")
	kid := mustKid(t, s, ana.ID, "Mia")
	mustGrant(t, s, kid.ID, ana.ID, "ben", model.RoleCaregiver)
	mustTopic(t, s, ben.ID)
	mustTopic(t, s, ana.ID)
	for _, p := range []string{ana.ID, ben.ID} {
		if err := s.SetNtfyOptin(ctx, kid.ID, p); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.SetParentDisabled(ctx, ben.ID, true, t0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.NtfyTopic(ctx, ben.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ben's topic after disable: %v", err)
	}
	if n := scalar(t, s, `SELECT COUNT(*) FROM ntfy_kid_optins WHERE parent_id = ?`, ben.ID); n != 0 {
		t.Fatalf("ben's opt-ins survived the disable")
	}
	if got, err := s.NtfyTopic(ctx, ana.ID); err != nil || len(got.Kids) != 1 {
		t.Fatalf("ana's topic touched: %+v %v", got, err)
	}
}

func TestDeleteKidCascadesNtfyOptins(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	kid := mustKid(t, s, ana.ID, "Mia")
	mustTopic(t, s, ana.ID)
	if err := s.SetNtfyOptin(ctx, kid.ID, ana.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteKid(ctx, kid.ID, ana.ID); err != nil {
		t.Fatal(err)
	}
	if n := scalar(t, s, `SELECT COUNT(*) FROM ntfy_kid_optins`); n != 0 {
		t.Fatalf("opt-in survived the kid")
	}
	if _, err := s.NtfyTopic(ctx, ana.ID); err != nil {
		t.Fatalf("deleting the kid removed the topic: %v", err)
	}
}
