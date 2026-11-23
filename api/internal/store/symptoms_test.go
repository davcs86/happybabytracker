package store

import (
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/units"
)

func mustTemperature(t *testing.T, unit units.Unit, amount int64) *units.Temperature {
	t.Helper()
	v, err := units.NewTemperature(unit, amount)
	if err != nil {
		t.Fatal(err)
	}
	return &v
}

func newFever(t *testing.T, at time.Time) model.Symptom {
	t.Helper()
	return model.Symptom{At: at, Symptom: "fever", Temperature: mustTemperature(t, units.UnitF, 10150), Method: strp("rectal")}
}

func mustSymptom(t *testing.T, s *Store, kidID, actorID string, v model.Symptom) model.Symptom {
	t.Helper()
	got, err := s.CreateSymptom(ctx, kidID, actorID, v, t0)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestSymptomCRUD(t *testing.T) {
	s := openMem(t)
	ana, bo := mustParent(t, s, "ana"), mustParent(t, s, "bo")
	k := mustKid(t, s, ana.ID, "Lía")
	mustGrant(t, s, k.ID, ana.ID, "bo", model.RoleCaregiver)

	in := newFever(t, t0.Add(-time.Hour))
	in.Severity, in.Note = strp("severe"), strp("n")
	created, err := s.CreateSymptom(ctx, k.ID, ana.ID, in, t0)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.KidID != k.ID || created.CreatedBy != ana.ID || !created.CreatedAt.Equal(t0) || created.Symptom != "fever" {
		t.Fatalf("created = %+v", created)
	}
	got, err := s.Symptom(ctx, k.ID, created.ID)
	if err != nil || !reflect.DeepEqual(got, created) {
		t.Fatalf("Symptom = %+v, %v; want %+v", got, err, created)
	}
	if want := (units.Temperature{CHundredths: 3861, FHundredths: 10150, Typed: units.UnitF}); *got.Temperature != want {
		t.Errorf("temperature = %+v, want %+v", *got.Temperature, want)
	}

	edit := got
	edit.Symptom, edit.Severity, edit.Temperature, edit.Method, edit.At, edit.Note = "cough", nil, nil, nil, t0.Add(-2*time.Hour), nil
	later := t0.Add(time.Hour)
	if err := s.UpdateSymptom(ctx, k.ID, bo.ID, edit, later); err != nil {
		t.Fatal(err)
	}
	edit.UpdatedBy, edit.UpdatedAt = bo.ID, later
	if got, err = s.Symptom(ctx, k.ID, created.ID); err != nil || !reflect.DeepEqual(got, edit) {
		t.Fatalf("after update = %+v, %v; want %+v", got, err, edit)
	}

	other := mustSymptom(t, s, k.ID, ana.ID, model.Symptom{At: t0, Symptom: "other", OtherText: strp("pulling at the ear")})
	if got, err := s.Symptom(ctx, k.ID, other.ID); err != nil || got.OtherText == nil || *got.OtherText != "pulling at the ear" || got.Temperature != nil || got.Method != nil {
		t.Errorf("other = %+v, %v", got, err)
	}

	if err := s.DeleteSymptom(ctx, k.ID, bo.ID, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Symptom(ctx, k.ID, created.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("after delete: %v", err)
	}
	if err := s.DeleteSymptom(ctx, k.ID, bo.ID, created.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("second delete: %v", err)
	}
	if err := s.UpdateSymptom(ctx, k.ID, bo.ID, edit, later); !errors.Is(err, ErrNotFound) {
		t.Errorf("update after delete: %v", err)
	}
}

func TestSymptomRefusesRowsTheTableForbids(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	for name, v := range map[string]model.Symptom{
		"fever without a temperature":  {At: t0, Symptom: "fever", Method: strp("oral")},
		"fever without a method":       {At: t0, Symptom: "fever", Temperature: mustTemperature(t, units.UnitC, 3850)},
		"a temperature on a cough":     {At: t0, Symptom: "cough", Temperature: mustTemperature(t, units.UnitC, 3850), Method: strp("oral")},
		"other without text":           {At: t0, Symptom: "other"},
		"text on a cough":              {At: t0, Symptom: "cough", OtherText: strp("x")},
		"an unknown symptom":           {At: t0, Symptom: "sneeze"},
		"an unknown severity":          {At: t0, Symptom: "cough", Severity: strp("dire")},
		"a method without temperature": {At: t0, Symptom: "cough", Method: strp("oral")},
	} {
		if _, err := s.CreateSymptom(ctx, k.ID, ana.ID, v, t0); err == nil {
			t.Errorf("%s: stored", name)
		}
	}
	if list, err := s.ListSymptoms(ctx, k.ID, wideQuery()); err != nil || len(list) != 0 {
		t.Errorf("list = %+v, %v; want none", list, err)
	}
}

func TestSymptomBelongsToItsKid(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	mine, other := mustKid(t, s, ana.ID, "Mine"), mustKid(t, s, ana.ID, "Other")
	v := mustSymptom(t, s, mine.ID, ana.ID, newFever(t, t0))

	if _, err := s.Symptom(ctx, other.ID, v.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("read through another kid: %v", err)
	}
	edit := v
	edit.Symptom, edit.Temperature, edit.Method = "cough", nil, nil
	if err := s.UpdateSymptom(ctx, other.ID, ana.ID, edit, t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("update through another kid: %v", err)
	}
	if err := s.DeleteSymptom(ctx, other.ID, ana.ID, v.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("delete through another kid: %v", err)
	}
	if list, err := s.ListSymptoms(ctx, other.ID, wideQuery()); err != nil || len(list) != 0 {
		t.Errorf("other kid's list = %+v, %v", list, err)
	}
	if got, err := s.Symptom(ctx, mine.ID, v.ID); err != nil || !reflect.DeepEqual(got, v) {
		t.Errorf("entry changed: %+v, %v", got, err)
	}
}

func TestSymptomWriteByDisabledActor(t *testing.T) {
	s := openMem(t)
	ana, bo, cy := mustParent(t, s, "ana"), mustParent(t, s, "bo"), mustParent(t, s, "cy")
	k := mustKid(t, s, ana.ID, "Lía")
	mustGrant(t, s, k.ID, ana.ID, "bo", model.RoleCaregiver)
	v := mustSymptom(t, s, k.ID, ana.ID, newFever(t, t0))
	if err := s.SetParentDisabled(ctx, bo.ID, true, t0); err != nil {
		t.Fatal(err)
	}

	for name, actor := range map[string]string{"disabled caregiver": bo.ID, "no grant": cy.ID} {
		if _, err := s.CreateSymptom(ctx, k.ID, actor, newFever(t, t0), t0); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s create: %v", name, err)
		}
		if err := s.UpdateSymptom(ctx, k.ID, actor, v, t0); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s update: %v", name, err)
		}
		if err := s.DeleteSymptom(ctx, k.ID, actor, v.ID); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s delete: %v", name, err)
		}
	}
	if list, err := s.ListSymptoms(ctx, k.ID, wideQuery()); err != nil || len(list) != 1 || !reflect.DeepEqual(list[0], v) {
		t.Errorf("list = %+v, %v; want the one seeded entry unchanged", list, err)
	}
}

func TestEditingNoteKeepsTypedTemperature(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	v := mustSymptom(t, s, k.ID, ana.ID, newFever(t, t0))
	want := units.Temperature{CHundredths: 3861, FHundredths: 10150, Typed: units.UnitF}

	v.Note = strp("only the note changed")
	if err := s.UpdateSymptom(ctx, k.ID, ana.ID, v, t0.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	got, err := s.Symptom(ctx, k.ID, v.ID)
	if err != nil {
		t.Fatal(err)
	}
	if *got.Temperature != want {
		t.Errorf("temperature = %+v, want %+v", *got.Temperature, want)
	}
}

func TestSymptomListWindowAndCursor(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	same := t0.Add(-time.Hour)
	var ids []string
	for _, at := range []time.Time{same.Add(-time.Hour), same, same, same, same.Add(time.Minute)} {
		ids = append(ids, mustSymptom(t, s, k.ID, ana.ID, model.Symptom{At: at, Symptom: "cough"}).ID)
	}
	ties := slices.Clone(ids[1:4])
	slices.Sort(ties)
	slices.Reverse(ties)
	want := append([]string{ids[4]}, ties...)

	q := wideQuery()
	q.From = same
	q.To = same.Add(2 * time.Minute)
	got, err := s.ListSymptoms(ctx, k.ID, q)
	if err != nil {
		t.Fatal(err)
	}
	gotIDs := make([]string, len(got))
	for i, v := range got {
		gotIDs[i] = v.ID
	}
	if !slices.Equal(gotIDs, want) {
		t.Fatalf("window = %v, want %v", gotIDs, want)
	}

	q.To = same.Add(time.Minute) // half open: the row at To is out
	if got, err := s.ListSymptoms(ctx, k.ID, q); err != nil || len(got) != 3 {
		t.Errorf("[from, to) = %d rows, %v; want 3", len(got), err)
	}
	q.To = same.Add(2 * time.Minute)
	q.After = &Cursor{At: same.Unix(), ID: want[1]}
	rest, err := s.ListSymptoms(ctx, k.ID, q)
	if err != nil || len(rest) != 2 || rest[0].ID != want[2] || rest[1].ID != want[3] {
		t.Errorf("after the second row = %+v, %v", rest, err)
	}
	q = wideQuery()
	q.Limit = 2
	if got, err := s.ListSymptoms(ctx, k.ID, q); err != nil || len(got) != 2 {
		t.Errorf("limit 2 = %d rows, %v", len(got), err)
	}
}
