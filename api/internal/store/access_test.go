package store

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
)

func intp(n int) *int { return &n }

func newKid(name string) model.Kid {
	return model.Kid{Name: name, BirthDate: "2026-01-02", Sex: "female", TimeZone: "Europe/Madrid"}
}

func mustKid(t *testing.T, s *Store, creator, name string) model.Kid {
	t.Helper()
	k, err := s.CreateKid(ctx, newKid(name), creator, t0)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func mustGrant(t *testing.T, s *Store, kidID, actorID, username string, role model.Role) {
	t.Helper()
	if err := s.SetAccess(ctx, kidID, actorID, username, role, t0); err != nil {
		t.Fatalf("SetAccess %s %s: %v", username, role, err)
	}
}

// activeGuardians counts on the writer so the answer reflects every committed write.
func activeGuardians(t *testing.T, s *Store, kidID string) int {
	t.Helper()
	var n int
	if err := s.w.QueryRow(`SELECT COUNT(*) FROM kid_access a JOIN parents p ON p.id = a.parent_id
		WHERE a.kid_id = ? AND a.role = 'guardian' AND p.disabled = 0`, kidID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestCreateKidStoresKidAndGuardianGrant(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	in := newKid("Lía")
	in.GestationWeeks, in.GestationDays = intp(36), intp(4)
	k, err := s.CreateKid(ctx, in, ana.ID, t0)
	if err != nil || k.ID == "" {
		t.Fatalf("CreateKid = %+v, %v", k, err)
	}
	got, err := s.Kid(ctx, k.ID)
	if err != nil || got.Name != "Lía" || got.BirthDate != "2026-01-02" || got.Sex != "female" || got.TimeZone != "Europe/Madrid" ||
		got.GestationWeeks == nil || *got.GestationWeeks != 36 || got.GestationDays == nil || *got.GestationDays != 4 {
		t.Fatalf("Kid = %+v, %v", got, err)
	}
	if role, err := s.RoleFor(ctx, k.ID, ana.ID); err != nil || role != model.RoleGuardian {
		t.Errorf("RoleFor = %q, %v", role, err)
	}
	var created, updated int64
	var by string
	if err := s.r.QueryRow(`SELECT created_at, updated_at, created_by FROM kids WHERE id = ?`, k.ID).Scan(&created, &updated, &by); err != nil ||
		created != t0.Unix() || updated != t0.Unix() || by != ana.ID {
		t.Errorf("stamps = %d %d %q, %v", created, updated, by, err)
	}
	if _, err := s.CreateKid(ctx, newKid("X"), "nobody", t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown creator: %v, want ErrNotFound", err)
	}
	if _, err := s.Kid(ctx, "nobody"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown kid: %v", err)
	}
}

func TestKidsForOrderAndRoles(t *testing.T) {
	s := openMem(t)
	ana, bo := mustParent(t, s, "ana"), mustParent(t, s, "bo")
	if got, err := s.KidsFor(ctx, ana.ID); err != nil || got == nil || len(got) != 0 {
		t.Fatalf("empty = %#v, %v", got, err)
	}
	late, err := s.CreateKid(ctx, newKid("Late"), ana.ID, t0.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	early := mustKid(t, s, bo.ID, "Early")
	mustGrant(t, s, early.ID, bo.ID, "ana", model.RoleCaregiver)
	mustKid(t, s, bo.ID, "Not Ana's")

	got, err := s.KidsFor(ctx, ana.ID)
	if err != nil || len(got) != 2 {
		t.Fatalf("KidsFor = %+v, %v", got, err)
	}
	if got[0].ID != early.ID || got[0].Role != model.RoleCaregiver || got[1].ID != late.ID || got[1].Role != model.RoleGuardian {
		t.Errorf("KidsFor = %+v", got)
	}
}

func TestUpdateKid(t *testing.T) {
	s := openMem(t)
	ana, bo := mustParent(t, s, "ana"), mustParent(t, s, "bo")
	k := mustKid(t, s, ana.ID, "Lía")
	mustGrant(t, s, k.ID, ana.ID, "bo", model.RoleGuardian)
	k.Name, k.TimeZone, k.GestationWeeks, k.GestationDays = "Lia", "UTC", intp(30), intp(0)
	later := t0.Add(time.Minute)
	if err := s.UpdateKid(ctx, k, bo.ID, later); err != nil {
		t.Fatal(err)
	}
	got, err := s.Kid(ctx, k.ID)
	if err != nil || got.Name != "Lia" || got.TimeZone != "UTC" || *got.GestationWeeks != 30 || *got.GestationDays != 0 {
		t.Fatalf("after update: %+v, %v", got, err)
	}
	var updated int64
	var by string
	if err := s.r.QueryRow(`SELECT updated_at, updated_by FROM kids WHERE id = ?`, k.ID).Scan(&updated, &by); err != nil || updated != later.Unix() || by != bo.ID {
		t.Errorf("updated = %d %q, %v", updated, by, err)
	}
	k.GestationWeeks, k.GestationDays = nil, nil
	if err := s.UpdateKid(ctx, k, ana.ID, later); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Kid(ctx, k.ID); got.GestationWeeks != nil || got.GestationDays != nil {
		t.Errorf("gestation not cleared: %+v", got)
	}
	k.ID = "nobody"
	if err := s.UpdateKid(ctx, k, ana.ID, later); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown kid: %v", err)
	}
}

func TestDeleteKidCascadesOnlyItsGrants(t *testing.T) {
	s := openMem(t)
	ana, bo := mustParent(t, s, "ana"), mustParent(t, s, "bo")
	gone, kept := mustKid(t, s, ana.ID, "Gone"), mustKid(t, s, ana.ID, "Kept")
	mustGrant(t, s, gone.ID, ana.ID, "bo", model.RoleCaregiver)
	mustGrant(t, s, kept.ID, ana.ID, "bo", model.RoleGuardian)

	if err := s.DeleteKid(ctx, gone.ID, ana.ID); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := s.w.QueryRow(`SELECT COUNT(*) FROM kid_access WHERE kid_id = ?`, gone.ID).Scan(&n); err != nil || n != 0 {
		t.Errorf("grants of the deleted kid = %d, %v; the cascade did not fire", n, err)
	}
	if _, err := s.Kid(ctx, gone.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("deleted kid: %v", err)
	}
	grants, err := s.AccessList(ctx, kept.ID)
	if err != nil || len(grants) != 2 {
		t.Fatalf("other kid's grants = %+v, %v", grants, err)
	}
	if _, err := s.Kid(ctx, kept.ID); err != nil {
		t.Errorf("other kid: %v", err)
	}
	if got, _ := s.KidsFor(ctx, bo.ID); len(got) != 1 || got[0].ID != kept.ID {
		t.Errorf("bo's kids = %+v", got)
	}
	if err := s.DeleteKid(ctx, gone.ID, ana.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("second delete: %v", err)
	}
}

func TestAccessListOrder(t *testing.T) {
	s := openMem(t)
	zed := mustParent(t, s, "zed")
	mustParent(t, s, "amy")
	mustParent(t, s, "bob")
	mustParent(t, s, "cal")
	k := mustKid(t, s, zed.ID, "Lía")
	mustGrant(t, s, k.ID, zed.ID, "cal", model.RoleCaregiver)
	mustGrant(t, s, k.ID, zed.ID, "bob", model.RoleGuardian)
	mustGrant(t, s, k.ID, zed.ID, "amy", model.RoleCaregiver)
	got, err := s.AccessList(ctx, k.ID)
	if err != nil {
		t.Fatal(err)
	}
	var order []string
	for _, g := range got {
		order = append(order, g.Username+":"+string(g.Role))
	}
	if fmt.Sprint(order) != "[bob:guardian zed:guardian amy:caregiver cal:caregiver]" {
		t.Errorf("order = %v", order)
	}
	if got[1].ParentID != zed.ID || got[1].DisplayName != "Name zed" {
		t.Errorf("grant = %+v", got[1])
	}
	if empty, err := s.AccessList(ctx, "nobody"); err != nil || empty == nil || len(empty) != 0 {
		t.Errorf("unknown kid = %#v, %v", empty, err)
	}
}

func TestSetAccess(t *testing.T) {
	s := openMem(t)
	ana, bo := mustParent(t, s, "ana"), mustParent(t, s, "bo")
	off := mustParent(t, s, "off")
	if err := s.SetParentDisabled(ctx, off.ID, true, t0); err != nil {
		t.Fatal(err)
	}
	k := mustKid(t, s, ana.ID, "Lía")

	mustGrant(t, s, k.ID, ana.ID, "bo", model.RoleCaregiver)
	if role, _ := s.RoleFor(ctx, k.ID, bo.ID); role != model.RoleCaregiver {
		t.Errorf("bo = %q", role)
	}
	mustGrant(t, s, k.ID, ana.ID, "bo", model.RoleGuardian)
	if role, _ := s.RoleFor(ctx, k.ID, bo.ID); role != model.RoleGuardian {
		t.Errorf("bo after upgrade = %q", role)
	}
	for _, name := range []string{"nobody", "off"} {
		if err := s.SetAccess(ctx, k.ID, ana.ID, name, model.RoleCaregiver, t0); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s: %v, want ErrNotFound", name, err)
		}
	}
	if _, err := s.RoleFor(ctx, k.ID, off.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("disabled parent got a grant: %v", err)
	}
	if _, err := s.RoleFor(ctx, "nobody", ana.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown kid: %v", err)
	}
}

func TestLastGuardianInTheStore(t *testing.T) {
	s := openMem(t)
	ana, bo := mustParent(t, s, "ana"), mustParent(t, s, "bo")
	k := mustKid(t, s, ana.ID, "Lía")

	if err := s.SetAccess(ctx, k.ID, ana.ID, "ana", model.RoleCaregiver, t0); !errors.Is(err, ErrLastGuardian) {
		t.Errorf("sole guardian downgrade: %v", err)
	}
	if err := s.RemoveAccess(ctx, k.ID, ana.ID, ana.ID); !errors.Is(err, ErrLastGuardian) {
		t.Errorf("sole guardian leaves: %v", err)
	}
	if role, _ := s.RoleFor(ctx, k.ID, ana.ID); role != model.RoleGuardian {
		t.Errorf("refusal was not rolled back: %q", role)
	}
	if err := s.RemoveAccess(ctx, k.ID, ana.ID, bo.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("no grant: %v", err)
	}

	mustGrant(t, s, k.ID, ana.ID, "bo", model.RoleGuardian)
	if err := s.SetParentDisabled(ctx, bo.ID, true, t0); err != nil {
		t.Fatal(err)
	}
	// A disabled guardian does not count.
	if err := s.RemoveAccess(ctx, k.ID, ana.ID, ana.ID); !errors.Is(err, ErrLastGuardian) {
		t.Errorf("leave with only a disabled co-guardian: %v", err)
	}
	if err := s.SetParentDisabled(ctx, bo.ID, false, t0); err != nil {
		t.Fatal(err)
	}
	if err := s.SetAccess(ctx, k.ID, ana.ID, "ana", model.RoleCaregiver, t0); err != nil {
		t.Errorf("downgrade with a co-guardian: %v", err)
	}
	if err := s.RemoveAccess(ctx, k.ID, ana.ID, ana.ID); err != nil {
		t.Errorf("caregiver leaves: %v", err)
	}
	if activeGuardians(t, s, k.ID) != 1 {
		t.Error("want one active guardian")
	}
}

func TestDisableChecksEveryKidTheParentGuards(t *testing.T) {
	s := openMem(t)
	ana, bo := mustParent(t, s, "ana"), mustParent(t, s, "bo")
	exp := t0.Add(time.Hour)
	mustSession(t, s, "parent", ana.ID, []byte("a1"), exp)
	var kids []model.Kid
	for i := range 5 {
		k := mustKid(t, s, ana.ID, fmt.Sprintf("Kid %d", i))
		kids = append(kids, k)
		if i != 3 {
			mustGrant(t, s, k.ID, ana.ID, "bo", model.RoleGuardian)
		}
	}
	if err := s.SetParentDisabled(ctx, ana.ID, true, t0); !errors.Is(err, ErrLastGuardian) {
		t.Fatalf("disable a sole guardian of one of five kids: %v", err)
	}
	if p, _ := s.ParentByID(ctx, ana.ID); p.Disabled {
		t.Error("refused disable left the parent disabled")
	}
	if _, err := s.SessionByToken(ctx, []byte("a1"), t0); err != nil {
		t.Errorf("refused disable ended the session: %v", err)
	}

	mustGrant(t, s, kids[3].ID, ana.ID, "bo", model.RoleCaregiver)
	if err := s.SetParentDisabled(ctx, ana.ID, true, t0); !errors.Is(err, ErrLastGuardian) {
		t.Errorf("a caregiver does not make a guardian: %v", err)
	}
	mustGrant(t, s, kids[3].ID, ana.ID, "bo", model.RoleGuardian)
	if err := s.SetParentDisabled(ctx, ana.ID, true, t0); err != nil {
		t.Fatalf("disable with a co-guardian everywhere: %v", err)
	}
	// Enabling never needs the check, and the grants stayed.
	if err := s.SetParentDisabled(ctx, bo.ID, false, t0); err != nil {
		t.Fatal(err)
	}
	if err := s.SetParentDisabled(ctx, ana.ID, false, t0); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.KidsFor(ctx, ana.ID); len(got) != 5 {
		t.Errorf("re-enabled parent's kids = %d", len(got))
	}
	// A parent who guards nothing alone, or only cares for kids, can be disabled.
	if err := s.SetParentDisabled(ctx, bo.ID, true, t0); err != nil {
		t.Errorf("disable a co-guardian: %v", err)
	}
}

// armPair creates fresh guardians A and B of one fresh kid, so earlier rounds never constrain this one.
func armPair(t *testing.T, s *Store, round int) (kid model.Kid, a, b model.Parent) {
	t.Helper()
	a, b = mustParent(t, s, fmt.Sprintf("a%d", round)), mustParent(t, s, fmt.Sprintf("b%d", round))
	kid = mustKid(t, s, a.ID, "Lía")
	mustGrant(t, s, kid.ID, a.ID, b.Username, model.RoleGuardian)
	return kid, a, b
}

// race runs both operations from two goroutines released together.
func race(f, g func() error) (error, error) {
	var (
		wg     sync.WaitGroup
		start  = make(chan struct{})
		e1, e2 error
	)
	wg.Add(2)
	go func() { defer wg.Done(); <-start; e1 = f() }()
	go func() { defer wg.Done(); <-start; e2 = g() }()
	close(start)
	wg.Wait()
	return e1, e2
}

func exactlyOneWins(t *testing.T, round int, e1, e2, wantLoss error) {
	t.Helper()
	ok1, ok2 := e1 == nil, e2 == nil
	if ok1 == ok2 {
		t.Fatalf("round %d: errors %v / %v, want exactly one success", round, e1, e2)
	}
	if lost := errors.Join(e1, e2); !errors.Is(lost, wantLoss) {
		t.Fatalf("round %d: loser got %v, want %v", round, lost, wantLoss)
	}
}

func TestConcurrentRevokes(t *testing.T) {
	s := openMem(t)
	for round := range 50 {
		kid, a, b := armPair(t, s, round)
		e1, e2 := race(
			func() error { return s.RemoveAccess(ctx, kid.ID, a.ID, b.ID) },
			func() error { return s.RemoveAccess(ctx, kid.ID, b.ID, a.ID) },
		)
		// The loser's own grant is already gone, so its actor check refuses it before the guardian count.
		exactlyOneWins(t, round, e1, e2, ErrNotFound)
		if n := activeGuardians(t, s, kid.ID); n != 1 {
			t.Fatalf("round %d: %d active guardians, want 1", round, n)
		}
	}
}

func TestConcurrentLeaveAndDisable(t *testing.T) {
	s := openMem(t)
	for round := range 50 {
		kid, a, b := armPair(t, s, round)
		e1, e2 := race(
			func() error { return s.RemoveAccess(ctx, kid.ID, b.ID, b.ID) },
			func() error { return s.SetParentDisabled(ctx, a.ID, true, t0) },
		)
		exactlyOneWins(t, round, e1, e2, ErrLastGuardian)
		if n := activeGuardians(t, s, kid.ID); n != 1 {
			t.Fatalf("round %d: %d active guardians, want 1", round, n)
		}
	}
}

func countKids(t *testing.T, s *Store) int {
	t.Helper()
	var n int
	if err := s.w.QueryRow(`SELECT COUNT(*) FROM kids`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestCreateKidByDisabledCreatorStoresNothing(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	if err := s.SetParentDisabled(ctx, ana.ID, true, t0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateKid(ctx, newKid("Lía"), ana.ID, t0); !errors.Is(err, ErrNotFound) {
		t.Fatalf("disabled creator: %v, want ErrNotFound", err)
	}
	if n := countKids(t, s); n != 0 {
		t.Fatalf("%d kids stored", n)
	}
	var grants int
	if err := s.w.QueryRow(`SELECT COUNT(*) FROM kid_access`).Scan(&grants); err != nil || grants != 0 {
		t.Errorf("grants = %d, %v", grants, err)
	}
}

func TestRevokedActorCannotRegrant(t *testing.T) {
	s := openMem(t)
	kid, a, b := armPair(t, s, 0)
	mustParent(t, s, "cal")
	if err := s.RemoveAccess(ctx, kid.ID, b.ID, a.ID); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{a.Username, "cal"} {
		if err := s.SetAccess(ctx, kid.ID, a.ID, target, model.RoleGuardian, t0); !errors.Is(err, ErrNotFound) {
			t.Errorf("revoked actor grants %s: %v, want ErrNotFound", target, err)
		}
	}
	if _, err := s.RoleFor(ctx, kid.ID, a.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("the revoke was undone: %v", err)
	}
	if grants, _ := s.AccessList(ctx, kid.ID); len(grants) != 1 || grants[0].ParentID != b.ID {
		t.Errorf("grants = %+v", grants)
	}
	k := kid
	k.Name = "Changed"
	if err := s.UpdateKid(ctx, k, a.ID, t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("revoked actor updates: %v", err)
	}
	if err := s.RemoveAccess(ctx, kid.ID, a.ID, b.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("revoked actor removes: %v", err)
	}
	if err := s.DeleteKid(ctx, kid.ID, a.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("revoked actor deletes: %v", err)
	}
	if got, err := s.Kid(ctx, kid.ID); err != nil || got.Name != "Lía" {
		t.Errorf("kid = %+v, %v", got, err)
	}
	// A vanished kid answers ErrNotFound, not a foreign-key failure.
	if err := s.DeleteKid(ctx, kid.ID, b.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.SetAccess(ctx, kid.ID, b.ID, "cal", model.RoleCaregiver, t0); !errors.Is(err, ErrNotFound) || errors.Is(err, ErrParentNotFound) {
		t.Errorf("grant on a deleted kid: %v, want ErrNotFound (not ErrParentNotFound)", err)
	}
}

func TestDowngradedActorIsForbidden(t *testing.T) {
	s := openMem(t)
	kid, a, b := armPair(t, s, 0)
	mustParent(t, s, "cal")
	mustGrant(t, s, kid.ID, b.ID, a.Username, model.RoleCaregiver)
	k := kid
	k.Name = "Changed"
	if err := s.UpdateKid(ctx, k, a.ID, t0); !errors.Is(err, ErrForbidden) {
		t.Errorf("caregiver updates: %v, want ErrForbidden", err)
	}
	if err := s.DeleteKid(ctx, kid.ID, a.ID); !errors.Is(err, ErrForbidden) {
		t.Errorf("caregiver deletes: %v, want ErrForbidden", err)
	}
	for _, target := range []string{a.Username, b.Username, "cal"} {
		if err := s.SetAccess(ctx, kid.ID, a.ID, target, model.RoleGuardian, t0); !errors.Is(err, ErrForbidden) {
			t.Errorf("caregiver grants %s: %v, want ErrForbidden", target, err)
		}
	}
	if err := s.RemoveAccess(ctx, kid.ID, a.ID, b.ID); !errors.Is(err, ErrForbidden) {
		t.Errorf("caregiver removes a guardian: %v, want ErrForbidden", err)
	}
	if got, _ := s.Kid(ctx, kid.ID); got.Name != "Lía" {
		t.Errorf("kid changed: %+v", got)
	}
	grants, _ := s.AccessList(ctx, kid.ID)
	if len(grants) != 2 || grants[0].ParentID != b.ID || grants[1].Role != model.RoleCaregiver {
		t.Errorf("grants changed: %+v", grants)
	}
	if err := s.SetAccess(ctx, kid.ID, b.ID, "nobody", model.RoleCaregiver, t0); !errors.Is(err, ErrParentNotFound) || !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown target: %v, want ErrParentNotFound (an ErrNotFound)", err)
	}
	// Leaving needs no role.
	if err := s.RemoveAccess(ctx, kid.ID, a.ID, a.ID); err != nil {
		t.Errorf("caregiver leaves: %v", err)
	}
}

func TestConcurrentRevokeAndRegrant(t *testing.T) {
	s := openMem(t)
	for round := range 50 {
		kid, a, b := armPair(t, s, round)
		removed, granted := race(
			func() error { return s.RemoveAccess(ctx, kid.ID, b.ID, a.ID) },
			func() error { return s.SetAccess(ctx, kid.ID, a.ID, a.Username, model.RoleGuardian, t0) },
		)
		if removed != nil {
			t.Fatalf("round %d: revoke: %v", round, removed)
		}
		if granted != nil && !errors.Is(granted, ErrNotFound) {
			t.Fatalf("round %d: regrant: %v", round, granted)
		}
		if _, err := s.RoleFor(ctx, kid.ID, a.ID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("round %d: the revoke was undone (regrant %v): %v", round, granted, err)
		}
		grants, err := s.AccessList(ctx, kid.ID)
		if err != nil || len(grants) != 1 || grants[0].ParentID != b.ID || activeGuardians(t, s, kid.ID) != 1 {
			t.Fatalf("round %d: grants = %+v, %v", round, grants, err)
		}
	}
}
