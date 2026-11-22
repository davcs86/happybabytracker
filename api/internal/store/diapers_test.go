package store

import (
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
)

func mustDiaper(t *testing.T, s *Store, kidID, actorID string, d model.Diaper) model.Diaper {
	t.Helper()
	got, err := s.CreateDiaper(ctx, kidID, actorID, d, t0)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestDiaperCRUD(t *testing.T) {
	s := openMem(t)
	ana, bo := mustParent(t, s, "ana"), mustParent(t, s, "bo")
	k := mustKid(t, s, ana.ID, "Lía")
	mustGrant(t, s, k.ID, ana.ID, "bo", model.RoleCaregiver)

	created, err := s.CreateDiaper(ctx, k.ID, ana.ID, model.Diaper{At: t0.Add(-time.Hour), Kind: "wet", Entry: model.Entry{Note: strp("n")}}, t0)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.KidID != k.ID || created.CreatedBy != ana.ID || !created.CreatedAt.Equal(t0) || created.Kind != "wet" {
		t.Fatalf("created = %+v", created)
	}
	got, err := s.Diaper(ctx, k.ID, created.ID)
	if err != nil || !reflect.DeepEqual(got, created) {
		t.Fatalf("Diaper = %+v, %v; want %+v", got, err, created)
	}

	edit := got
	edit.Kind, edit.At, edit.Note = "both", t0.Add(-2*time.Hour), nil
	later := t0.Add(time.Hour)
	if err := s.UpdateDiaper(ctx, k.ID, bo.ID, edit, later); err != nil {
		t.Fatal(err)
	}
	edit.UpdatedBy, edit.UpdatedAt = bo.ID, later
	if got, err = s.Diaper(ctx, k.ID, created.ID); err != nil || !reflect.DeepEqual(got, edit) {
		t.Fatalf("after update = %+v, %v; want %+v", got, err, edit)
	}
	if err := s.DeleteDiaper(ctx, k.ID, bo.ID, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Diaper(ctx, k.ID, created.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("after delete: %v", err)
	}
	if err := s.DeleteDiaper(ctx, k.ID, bo.ID, created.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("second delete: %v", err)
	}
}

func TestDiaperBelongsToItsKidAndItsWriter(t *testing.T) {
	s := openMem(t)
	ana, bo := mustParent(t, s, "ana"), mustParent(t, s, "bo")
	mine, other := mustKid(t, s, ana.ID, "Mine"), mustKid(t, s, ana.ID, "Other")
	d := mustDiaper(t, s, mine.ID, ana.ID, model.Diaper{At: t0, Kind: "dry"})

	if _, err := s.Diaper(ctx, other.ID, d.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("read through another kid: %v", err)
	}
	if err := s.UpdateDiaper(ctx, other.ID, ana.ID, d, t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("update through another kid: %v", err)
	}
	if err := s.DeleteDiaper(ctx, other.ID, ana.ID, d.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("delete through another kid: %v", err)
	}
	if err := s.UpdateDiaper(ctx, mine.ID, bo.ID, d, t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("update without a grant: %v", err)
	}
	if _, err := s.CreateDiaper(ctx, mine.ID, bo.ID, model.Diaper{At: t0, Kind: "wet"}, t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("create without a grant: %v", err)
	}
	if err := s.DeleteDiaper(ctx, mine.ID, bo.ID, d.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("delete without a grant: %v", err)
	}
	if got, err := s.Diaper(ctx, mine.ID, d.ID); err != nil || !reflect.DeepEqual(got, d) {
		t.Errorf("entry changed: %+v, %v", got, err)
	}
}

func TestDiaperListNewestFirstWithEqualTimestamps(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	same := t0.Add(-time.Hour)
	var want []string
	for _, at := range []time.Time{same, same, same, same.Add(time.Minute)} {
		want = append(want, mustDiaper(t, s, k.ID, ana.ID, model.Diaper{At: at, Kind: "wet"}).ID)
	}
	newest, ties := want[3], slices.Clone(want[:3])
	slices.Sort(ties)
	slices.Reverse(ties)
	want = append([]string{newest}, ties...)

	got, err := s.ListDiapers(ctx, k.ID, wideQuery())
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, len(got))
	for i, d := range got {
		ids[i] = d.ID
	}
	if !slices.Equal(ids, want) {
		t.Fatalf("order = %v, want %v", ids, want)
	}

	q := wideQuery()
	q.After = &Cursor{At: got[1].At.Unix(), ID: got[1].ID}
	rest, err := s.ListDiapers(ctx, k.ID, q)
	if err != nil || len(rest) != 2 || rest[0].ID != want[2] || rest[1].ID != want[3] {
		t.Errorf("after the second row = %+v, %v", rest, err)
	}
}
