package store

import (
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/units"
)

func mustWeight(t *testing.T, unit units.Unit, amount int64) *units.Weight {
	t.Helper()
	w, err := units.NewWeight(unit, amount)
	if err != nil {
		t.Fatal(err)
	}
	return &w
}

func mustLength(t *testing.T, unit units.Unit, amount int64) *units.Length {
	t.Helper()
	l, err := units.NewLength(unit, amount)
	if err != nil {
		t.Fatal(err)
	}
	return &l
}

func newGrowth(t *testing.T, on string) model.Growth {
	t.Helper()
	return model.Growth{MeasuredOn: on, Weight: mustWeight(t, units.UnitKG, 3400)}
}

func mustGrowth(t *testing.T, s *Store, kidID, actorID string, g model.Growth) model.Growth {
	t.Helper()
	got, err := s.CreateGrowth(ctx, kidID, actorID, g, t0)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestGrowthCRUD(t *testing.T) {
	s := openMem(t)
	ana, bo := mustParent(t, s, "ana"), mustParent(t, s, "bo")
	k := mustKid(t, s, ana.ID, "Lía")
	mustGrant(t, s, k.ID, ana.ID, "bo", model.RoleCaregiver)

	in := model.Growth{MeasuredOn: "2026-02-01", Weight: mustWeight(t, units.UnitKG, 4100),
		Length: mustLength(t, units.UnitCM, 545), Head: mustLength(t, units.UnitIN, 1500), Entry: model.Entry{Note: strp("visit")}}
	created, err := s.CreateGrowth(ctx, k.ID, ana.ID, in, t0)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.KidID != k.ID || created.CreatedBy != ana.ID || created.UpdatedBy != ana.ID ||
		!created.CreatedAt.Equal(t0) || !created.UpdatedAt.Equal(t0) || created.MeasuredOn != "2026-02-01" ||
		!reflect.DeepEqual(created.Weight, in.Weight) || !reflect.DeepEqual(created.Head, in.Head) {
		t.Fatalf("created = %+v", created)
	}
	got, err := s.Growth(ctx, k.ID, created.ID)
	if err != nil || !reflect.DeepEqual(got, created) {
		t.Fatalf("Growth = %+v, %v; want %+v", got, err, created)
	}

	edit := got
	edit.MeasuredOn, edit.Weight, edit.Head, edit.Note = "2026-02-02", nil, nil, nil
	edit.Length = mustLength(t, units.UnitIN, 2200)
	later := t0.Add(time.Hour)
	if err := s.UpdateGrowth(ctx, k.ID, bo.ID, edit, later); err != nil {
		t.Fatal(err)
	}
	edit.UpdatedBy, edit.UpdatedAt = bo.ID, later
	if got, err = s.Growth(ctx, k.ID, created.ID); err != nil || !reflect.DeepEqual(got, edit) {
		t.Fatalf("after update = %+v, %v; want %+v", got, err, edit)
	}
	if list, err := s.ListGrowth(ctx, k.ID); err != nil || len(list) != 1 || !reflect.DeepEqual(list[0], edit) {
		t.Fatalf("list = %+v, %v", list, err)
	}
	if err := s.DeleteGrowth(ctx, k.ID, bo.ID, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Growth(ctx, k.ID, created.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("after delete: %v", err)
	}
	if err := s.DeleteGrowth(ctx, k.ID, bo.ID, created.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("second delete: %v", err)
	}
	if list, err := s.ListGrowth(ctx, k.ID); err != nil || list == nil || len(list) != 0 {
		t.Errorf("empty list = %#v, %v; want an empty, non-nil slice", list, err)
	}
}

func TestGrowthBelongsToItsKid(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	mine, other := mustKid(t, s, ana.ID, "Mine"), mustKid(t, s, ana.ID, "Other")
	g := mustGrowth(t, s, mine.ID, ana.ID, newGrowth(t, "2026-02-01"))

	if _, err := s.Growth(ctx, other.ID, g.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("read through another kid: %v", err)
	}
	if err := s.UpdateGrowth(ctx, other.ID, ana.ID, g, t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("update through another kid: %v", err)
	}
	if err := s.DeleteGrowth(ctx, other.ID, ana.ID, g.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("delete through another kid: %v", err)
	}
	if list, err := s.ListGrowth(ctx, other.ID); err != nil || len(list) != 0 {
		t.Errorf("other kid's list = %+v, %v", list, err)
	}
	if got, err := s.Growth(ctx, mine.ID, g.ID); err != nil || !reflect.DeepEqual(got, g) {
		t.Errorf("entry changed: %+v, %v", got, err)
	}
}

func TestGrowthWriteByDisabledActor(t *testing.T) {
	s := openMem(t)
	ana, bo, cy := mustParent(t, s, "ana"), mustParent(t, s, "bo"), mustParent(t, s, "cy")
	k := mustKid(t, s, ana.ID, "Lía")
	mustGrant(t, s, k.ID, ana.ID, "bo", model.RoleCaregiver)
	g := mustGrowth(t, s, k.ID, ana.ID, newGrowth(t, "2026-02-01"))
	if err := s.SetParentDisabled(ctx, bo.ID, true, t0); err != nil {
		t.Fatal(err)
	}

	for name, actor := range map[string]string{"disabled caregiver": bo.ID, "no grant": cy.ID} {
		if _, err := s.CreateGrowth(ctx, k.ID, actor, newGrowth(t, "2026-02-02"), t0); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s create: %v", name, err)
		}
		if err := s.UpdateGrowth(ctx, k.ID, actor, g, t0); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s update: %v", name, err)
		}
		if err := s.DeleteGrowth(ctx, k.ID, actor, g.ID); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s delete: %v", name, err)
		}
	}
	if list, err := s.ListGrowth(ctx, k.ID); err != nil || len(list) != 1 || !reflect.DeepEqual(list[0], g) {
		t.Errorf("list = %+v, %v; want the one seeded entry unchanged", list, err)
	}
}

func TestTypedValuesStayExact(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	in := model.Growth{MeasuredOn: "2026-01-03", Weight: mustWeight(t, units.UnitLB, 13150)} // 8 lb 3.5 oz
	g := mustGrowth(t, s, k.ID, ana.ID, in)
	want := units.Weight{Grams: 3728, OzHundredths: 13150, Typed: units.UnitLB}

	g.Note = strp("only the note changed")
	if err := s.UpdateGrowth(ctx, k.ID, ana.ID, g, t0.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	g.Weight = mustWeight(t, units.UnitLB, 13150)
	if err := s.UpdateGrowth(ctx, k.ID, ana.ID, g, t0.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	got, err := s.Growth(ctx, k.ID, g.ID)
	if err != nil || got.Weight == nil || *got.Weight != want || got.Length != nil || got.Head != nil {
		t.Fatalf("weight = %+v (length %v, head %v), %v; want %+v", got.Weight, got.Length, got.Head, err, want)
	}
}

func TestListNewestFirstTieByID(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	var ids []string
	for _, on := range []string{"2026-02-01", "2026-02-01", "2026-02-01", "2026-03-01", "2026-01-15"} {
		ids = append(ids, mustGrowth(t, s, k.ID, ana.ID, newGrowth(t, on)).ID)
	}
	ties := slices.Clone(ids[:3])
	slices.Sort(ties)
	slices.Reverse(ties)
	want := append(append([]string{ids[3]}, ties...), ids[4])

	list, err := s.ListGrowth(ctx, k.ID)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, len(list))
	for i, g := range list {
		got[i] = g.ID
	}
	if !slices.Equal(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
}

func TestRowCap(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k, other := mustKid(t, s, ana.ID, "Lía"), mustKid(t, s, ana.ID, "Noa")
	tx, err := s.w.Begin()
	if err != nil {
		t.Fatal(err)
	}
	for i := range constants.MaxGrowthPerKid - 1 {
		if _, err := tx.Exec(insertGrowth, model.NewID(), k.ID, 3400+i, 120, "kg", nil, nil, nil, nil, nil, nil, ana.ID, ana.ID); err != nil {
			tx.Rollback()
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	last := mustGrowth(t, s, k.ID, ana.ID, newGrowth(t, "2026-02-01"))
	if _, err := s.CreateGrowth(ctx, k.ID, ana.ID, newGrowth(t, "2026-02-02"), t0); !errors.Is(err, ErrConflict) {
		t.Fatalf("row %d: %v, want ErrConflict", constants.MaxGrowthPerKid+1, err)
	}
	if list, err := s.ListGrowth(ctx, k.ID); err != nil || len(list) != constants.MaxGrowthPerKid {
		t.Errorf("list = %d rows, %v; want %d", len(list), err, constants.MaxGrowthPerKid)
	}
	mustGrowth(t, s, other.ID, ana.ID, newGrowth(t, "2026-02-02")) // the cap is per kid
	if err := s.DeleteGrowth(ctx, k.ID, ana.ID, last.ID); err != nil {
		t.Fatal(err)
	}
	mustGrowth(t, s, k.ID, ana.ID, newGrowth(t, "2026-02-03"))
}
