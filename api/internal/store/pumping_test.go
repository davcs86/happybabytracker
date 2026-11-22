package store

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/units"
)

func newPumping(t *testing.T, startedAt time.Time) model.PumpingSession {
	t.Helper()
	return model.PumpingSession{StartedAt: startedAt, DurationSeconds: 900, Volume: mustVolume(t, units.UnitML, 90), Side: "both"}
}

func mustPumping(t *testing.T, s *Store, kidID, actorID string, p model.PumpingSession) model.PumpingSession {
	t.Helper()
	got, err := s.CreatePumping(ctx, kidID, actorID, p, t0)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestPumpingCRUD(t *testing.T) {
	s := openMem(t)
	ana, bo := mustParent(t, s, "ana"), mustParent(t, s, "bo")
	k := mustKid(t, s, ana.ID, "Lía")
	mustGrant(t, s, k.ID, ana.ID, "bo", model.RoleCaregiver)

	in := newPumping(t, t0.Add(-time.Hour))
	in.Note = strp("first")
	created, err := s.CreatePumping(ctx, k.ID, ana.ID, in, t0)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.KidID != k.ID || created.CreatedBy != ana.ID || created.UpdatedBy != ana.ID ||
		!created.CreatedAt.Equal(t0) || !created.UpdatedAt.Equal(t0) || !created.StartedAt.Equal(in.StartedAt) {
		t.Fatalf("created = %+v", created)
	}
	got, err := s.Pumping(ctx, k.ID, created.ID)
	if err != nil || !reflect.DeepEqual(got, created) {
		t.Fatalf("Pumping = %+v, %v; want %+v", got, err, created)
	}

	edit := got
	edit.StartedAt, edit.DurationSeconds, edit.Side = t0.Add(-2*time.Hour), 1, "left"
	edit.Volume = mustVolume(t, units.UnitFlOz, 400)
	edit.Note = nil
	later := t0.Add(time.Hour)
	if err := s.UpdatePumping(ctx, k.ID, bo.ID, edit, later); err != nil {
		t.Fatal(err)
	}
	edit.UpdatedBy, edit.UpdatedAt = bo.ID, later
	if got, err = s.Pumping(ctx, k.ID, created.ID); err != nil || !reflect.DeepEqual(got, edit) {
		t.Fatalf("after update = %+v, %v; want %+v", got, err, edit)
	}
	if list, err := s.ListPumping(ctx, k.ID, wideQuery()); err != nil || len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("ListPumping = %+v, %v", list, err)
	}
	if err := s.DeletePumping(ctx, k.ID, bo.ID, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Pumping(ctx, k.ID, created.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("after delete: %v", err)
	}
	if err := s.DeletePumping(ctx, k.ID, bo.ID, created.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("second delete: %v", err)
	}
	if err := s.UpdatePumping(ctx, k.ID, bo.ID, edit, later); !errors.Is(err, ErrNotFound) {
		t.Errorf("update after delete: %v", err)
	}
}

// A session is recorded on the kid it is for (spec 6.3), whoever pumped.
func TestPumpingBelongsToItsKidAndItsWriter(t *testing.T) {
	s := openMem(t)
	ana, bo, cy := mustParent(t, s, "ana"), mustParent(t, s, "bo"), mustParent(t, s, "cy")
	mine, other := mustKid(t, s, ana.ID, "Mine"), mustKid(t, s, ana.ID, "Other")
	mustGrant(t, s, mine.ID, ana.ID, "bo", model.RoleCaregiver)
	p := mustPumping(t, s, mine.ID, ana.ID, newPumping(t, t0))

	if _, err := s.Pumping(ctx, other.ID, p.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("read through another kid: %v", err)
	}
	if err := s.UpdatePumping(ctx, other.ID, ana.ID, p, t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("update through another kid: %v", err)
	}
	if err := s.DeletePumping(ctx, other.ID, ana.ID, p.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("delete through another kid: %v", err)
	}
	if list, err := s.ListPumping(ctx, other.ID, wideQuery()); err != nil || len(list) != 0 {
		t.Errorf("other kid's list = %+v, %v", list, err)
	}

	if err := s.SetParentDisabled(ctx, bo.ID, true, t0); err != nil {
		t.Fatal(err)
	}
	for name, actor := range map[string]string{"no grant": cy.ID, "disabled caregiver": bo.ID} {
		if _, err := s.CreatePumping(ctx, mine.ID, actor, newPumping(t, t0), t0); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s create: %v", name, err)
		}
		if err := s.UpdatePumping(ctx, mine.ID, actor, p, t0); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s update: %v", name, err)
		}
		if err := s.DeletePumping(ctx, mine.ID, actor, p.ID); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s delete: %v", name, err)
		}
	}
	if got, err := s.Pumping(ctx, mine.ID, p.ID); err != nil || !reflect.DeepEqual(got, p) {
		t.Errorf("entry changed: %+v, %v", got, err)
	}
}

func TestPumpingListNewestFirstByStartWithEqualTimestamps(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	same := t0.Add(-time.Hour)
	var ids []string
	starts := []time.Time{same, same.Add(-time.Minute), same, same.Add(time.Minute), same}
	for _, at := range starts {
		ids = append(ids, mustPumping(t, s, k.ID, ana.ID, newPumping(t, at)).ID)
	}
	startOf := map[string]int64{}
	for i, id := range ids {
		startOf[id] = starts[i].Unix()
	}
	want := slices.Clone(ids)
	slices.SortFunc(want, func(a, b string) int {
		if startOf[a] != startOf[b] {
			return int(startOf[b] - startOf[a])
		}
		return -strings.Compare(a, b)
	})

	got, err := s.ListPumping(ctx, k.ID, wideQuery())
	if err != nil {
		t.Fatal(err)
	}
	gotIDs := make([]string, len(got))
	for i, p := range got {
		gotIDs[i] = p.ID
	}
	if !slices.Equal(gotIDs, want) {
		t.Fatalf("order = %v, want %v", gotIDs, want)
	}

	var paged []string
	q := wideQuery()
	q.Limit = 2
	for range ids {
		page, err := s.ListPumping(ctx, k.ID, q)
		if err != nil {
			t.Fatal(err)
		}
		if len(page) == 0 {
			break
		}
		for _, p := range page {
			paged = append(paged, p.ID)
		}
		last := page[len(page)-1]
		q.After = &Cursor{At: last.StartedAt.Unix(), ID: last.ID}
	}
	if !slices.Equal(paged, want) {
		t.Errorf("paged = %v, want %v", paged, want)
	}

	// The range is half-open on started_at.
	r, err := s.ListPumping(ctx, k.ID, ListQuery{From: same.Add(-time.Minute), To: same.Add(time.Minute), Limit: 10})
	if err != nil || len(r) != 4 {
		t.Errorf("[-1m, +1m) = %d rows, %v; want the 4 that start before +1m", len(r), err)
	}
}
