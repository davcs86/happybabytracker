package store

import (
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
)

func mustMeal(t *testing.T, s *Store, kidID, actorID string, m model.Meal) model.Meal {
	t.Helper()
	got, err := s.CreateMeal(ctx, kidID, actorID, m, t0)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestMealCRUD(t *testing.T) {
	s := openMem(t)
	ana, bo := mustParent(t, s, "ana"), mustParent(t, s, "bo")
	k := mustKid(t, s, ana.ID, "Lía")
	mustGrant(t, s, k.ID, ana.ID, "bo", model.RoleCaregiver)

	created, err := s.CreateMeal(ctx, k.ID, ana.ID, model.Meal{At: t0.Add(-time.Hour), What: "banana", Reaction: strp("liked it"), Entry: model.Entry{Note: strp("n")}}, t0)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.KidID != k.ID || created.CreatedBy != ana.ID || created.UpdatedBy != ana.ID ||
		!created.CreatedAt.Equal(t0) || !created.UpdatedAt.Equal(t0) || created.What != "banana" {
		t.Fatalf("created = %+v", created)
	}
	got, err := s.Meal(ctx, k.ID, created.ID)
	if err != nil || !reflect.DeepEqual(got, created) {
		t.Fatalf("Meal = %+v, %v; want %+v", got, err, created)
	}

	edit := got
	edit.What, edit.At, edit.Reaction, edit.Note = "oatmeal", t0.Add(-2*time.Hour), nil, nil
	later := t0.Add(time.Hour)
	if err := s.UpdateMeal(ctx, k.ID, bo.ID, edit, later); err != nil {
		t.Fatal(err)
	}
	edit.UpdatedBy, edit.UpdatedAt = bo.ID, later
	if got, err = s.Meal(ctx, k.ID, created.ID); err != nil || !reflect.DeepEqual(got, edit) {
		t.Fatalf("after update = %+v, %v; want %+v", got, err, edit)
	}
	if list, err := s.ListMeals(ctx, k.ID, wideQuery()); err != nil || len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("ListMeals = %+v, %v", list, err)
	}
	if err := s.DeleteMeal(ctx, k.ID, bo.ID, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Meal(ctx, k.ID, created.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("after delete: %v", err)
	}
	if err := s.DeleteMeal(ctx, k.ID, bo.ID, created.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("second delete: %v", err)
	}
	if err := s.UpdateMeal(ctx, k.ID, bo.ID, edit, later); !errors.Is(err, ErrNotFound) {
		t.Errorf("update after delete: %v", err)
	}
}

func TestMealBelongsToItsKidAndItsWriter(t *testing.T) {
	s := openMem(t)
	ana, bo, cy := mustParent(t, s, "ana"), mustParent(t, s, "bo"), mustParent(t, s, "cy")
	mine, other := mustKid(t, s, ana.ID, "Mine"), mustKid(t, s, ana.ID, "Other")
	mustGrant(t, s, mine.ID, ana.ID, "bo", model.RoleCaregiver)
	m := mustMeal(t, s, mine.ID, ana.ID, model.Meal{At: t0, What: "pear"})

	if _, err := s.Meal(ctx, other.ID, m.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("read through another kid: %v", err)
	}
	if err := s.UpdateMeal(ctx, other.ID, ana.ID, m, t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("update through another kid: %v", err)
	}
	if err := s.DeleteMeal(ctx, other.ID, ana.ID, m.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("delete through another kid: %v", err)
	}
	if list, err := s.ListMeals(ctx, other.ID, wideQuery()); err != nil || len(list) != 0 {
		t.Errorf("other kid's list = %+v, %v", list, err)
	}

	if err := s.SetParentDisabled(ctx, bo.ID, true, t0); err != nil {
		t.Fatal(err)
	}
	for name, actor := range map[string]string{"no grant": cy.ID, "disabled caregiver": bo.ID} {
		if _, err := s.CreateMeal(ctx, mine.ID, actor, model.Meal{At: t0, What: "pear"}, t0); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s create: %v", name, err)
		}
		if err := s.UpdateMeal(ctx, mine.ID, actor, m, t0); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s update: %v", name, err)
		}
		if err := s.DeleteMeal(ctx, mine.ID, actor, m.ID); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s delete: %v", name, err)
		}
	}
	if got, err := s.Meal(ctx, mine.ID, m.ID); err != nil || !reflect.DeepEqual(got, m) {
		t.Errorf("entry changed: %+v, %v", got, err)
	}
}

func TestMealListNewestFirstWithEqualTimestamps(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	same := t0.Add(-time.Hour)
	var want []string
	for _, at := range []time.Time{same, same, same, same.Add(time.Minute)} {
		want = append(want, mustMeal(t, s, k.ID, ana.ID, model.Meal{At: at, What: "rice"}).ID)
	}
	newest, ties := want[3], slices.Clone(want[:3])
	slices.Sort(ties)
	slices.Reverse(ties)
	want = append([]string{newest}, ties...)

	got, err := s.ListMeals(ctx, k.ID, wideQuery())
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, len(got))
	for i, m := range got {
		ids[i] = m.ID
	}
	if !slices.Equal(ids, want) {
		t.Fatalf("order = %v, want %v", ids, want)
	}

	q := wideQuery()
	q.After = &Cursor{At: got[1].At.Unix(), ID: got[1].ID}
	rest, err := s.ListMeals(ctx, k.ID, q)
	if err != nil || len(rest) != 2 || rest[0].ID != want[2] || rest[1].ID != want[3] {
		t.Errorf("after the second row = %+v, %v", rest, err)
	}
}
