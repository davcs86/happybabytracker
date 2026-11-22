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

func mustVolume(t *testing.T, unit units.Unit, amount int64) units.Volume {
	t.Helper()
	v, err := units.NewVolume(unit, amount)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func strp(s string) *string { return &s }

func newBottle(t *testing.T, at time.Time, contents ...string) model.BottleFeed {
	t.Helper()
	return model.BottleFeed{At: at, Contents: contents, Volume: mustVolume(t, units.UnitML, 120)}
}

func mustBottle(t *testing.T, s *Store, kidID, actorID string, b model.BottleFeed) model.BottleFeed {
	t.Helper()
	got, err := s.CreateBottle(ctx, kidID, actorID, b, t0)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestBottleCRUD(t *testing.T) {
	s := openMem(t)
	ana, bo := mustParent(t, s, "ana"), mustParent(t, s, "bo")
	k := mustKid(t, s, ana.ID, "Lía")
	mustGrant(t, s, k.ID, ana.ID, "bo", model.RoleCaregiver)

	in := newBottle(t, t0.Add(-time.Hour), model.ContentFormula, model.ContentBreastMilk)
	in.Note = strp("first")
	created, err := s.CreateBottle(ctx, k.ID, ana.ID, in, t0)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.KidID != k.ID || created.CreatedBy != ana.ID || created.UpdatedBy != ana.ID ||
		!created.CreatedAt.Equal(t0) || !created.UpdatedAt.Equal(t0) || !created.At.Equal(in.At) {
		t.Fatalf("created = %+v", created)
	}
	if want := []string{model.ContentBreastMilk, model.ContentFormula}; !slices.Equal(created.Contents, want) {
		t.Errorf("contents = %v, want %v", created.Contents, want)
	}
	got, err := s.Bottle(ctx, k.ID, created.ID)
	if err != nil || !reflect.DeepEqual(got, created) {
		t.Fatalf("Bottle = %+v, %v; want %+v", got, err, created)
	}

	edit := got
	edit.At = t0.Add(-2 * time.Hour)
	edit.Contents = []string{model.ContentCowMilk}
	edit.Volume = mustVolume(t, units.UnitFlOz, 400)
	edit.Note = nil
	later := t0.Add(time.Hour)
	if err := s.UpdateBottle(ctx, k.ID, bo.ID, edit, later); err != nil {
		t.Fatal(err)
	}
	got, err = s.Bottle(ctx, k.ID, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	edit.UpdatedBy, edit.UpdatedAt = bo.ID, later
	if !reflect.DeepEqual(got, edit) {
		t.Fatalf("after update = %+v, want %+v", got, edit)
	}
	if got.CreatedBy != ana.ID || !got.CreatedAt.Equal(t0) {
		t.Errorf("update touched the creation stamps: %+v", got.Entry)
	}

	list, err := s.ListBottles(ctx, k.ID, wideQuery())
	if err != nil || len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("ListBottles = %+v, %v", list, err)
	}
	if err := s.DeleteBottle(ctx, k.ID, bo.ID, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Bottle(ctx, k.ID, created.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("Bottle after delete: %v", err)
	}
	if err := s.DeleteBottle(ctx, k.ID, bo.ID, created.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("second delete: %v", err)
	}
	if err := s.UpdateBottle(ctx, k.ID, bo.ID, edit, later); !errors.Is(err, ErrNotFound) {
		t.Errorf("update after delete: %v", err)
	}
}

func wideQuery() ListQuery {
	return ListQuery{From: time.Unix(0, 0), To: t0.Add(24 * time.Hour), Limit: 100}
}

func TestBottleBelongsToItsKid(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	mine, other := mustKid(t, s, ana.ID, "Mine"), mustKid(t, s, ana.ID, "Other")
	b := mustBottle(t, s, mine.ID, ana.ID, newBottle(t, t0, model.ContentFormula))

	if _, err := s.Bottle(ctx, other.ID, b.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("read through another kid: %v", err)
	}
	edit := b
	edit.Contents = []string{model.ContentOther}
	if err := s.UpdateBottle(ctx, other.ID, ana.ID, edit, t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("update through another kid: %v", err)
	}
	if err := s.DeleteBottle(ctx, other.ID, ana.ID, b.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("delete through another kid: %v", err)
	}
	if list, err := s.ListBottles(ctx, other.ID, wideQuery()); err != nil || len(list) != 0 {
		t.Errorf("other kid's list = %+v, %v", list, err)
	}
	got, err := s.Bottle(ctx, mine.ID, b.ID)
	if err != nil || !reflect.DeepEqual(got, b) {
		t.Errorf("entry changed: %+v, %v", got, err)
	}
}

func TestBottleWriteByDisabledActor(t *testing.T) {
	s := openMem(t)
	ana, bo, cy := mustParent(t, s, "ana"), mustParent(t, s, "bo"), mustParent(t, s, "cy")
	k := mustKid(t, s, ana.ID, "Lía")
	mustGrant(t, s, k.ID, ana.ID, "bo", model.RoleCaregiver)
	b := mustBottle(t, s, k.ID, ana.ID, newBottle(t, t0, model.ContentFormula))
	if err := s.SetParentDisabled(ctx, bo.ID, true, t0); err != nil {
		t.Fatal(err)
	}

	for name, actor := range map[string]string{"disabled caregiver": bo.ID, "no grant": cy.ID} {
		if _, err := s.CreateBottle(ctx, k.ID, actor, newBottle(t, t0, model.ContentFormula), t0); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s create: %v", name, err)
		}
		if err := s.UpdateBottle(ctx, k.ID, actor, b, t0); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s update: %v", name, err)
		}
		if err := s.DeleteBottle(ctx, k.ID, actor, b.ID); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s delete: %v", name, err)
		}
	}
	if list, err := s.ListBottles(ctx, k.ID, wideQuery()); err != nil || len(list) != 1 {
		t.Errorf("list = %d rows, %v; want the one seeded entry", len(list), err)
	}
}

func TestEditingNoteKeepsTypedVolume(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	in := newBottle(t, t0, model.ContentFormula)
	in.Volume = mustVolume(t, units.UnitFlOz, 400)
	b := mustBottle(t, s, k.ID, ana.ID, in)

	b.Note = strp("only the note changed")
	if err := s.UpdateBottle(ctx, k.ID, ana.ID, b, t0.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	got, err := s.Bottle(ctx, k.ID, b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if want := (units.Volume{ML: 118, FlOzHundredths: 400, Typed: units.UnitFlOz}); got.Volume != want {
		t.Errorf("volume = %+v, want %+v", got.Volume, want)
	}
}

func TestMixedBottleContents(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")

	for _, c := range []struct {
		name     string
		in, want []string
	}{
		{"breast milk and formula", []string{model.ContentFormula, model.ContentBreastMilk}, []string{model.ContentBreastMilk, model.ContentFormula}},
		{"cow milk alone", []string{model.ContentCowMilk}, []string{model.ContentCowMilk}},
		{"all four, scrambled", []string{model.ContentOther, model.ContentCowMilk, model.ContentFormula, model.ContentBreastMilk},
			[]string{model.ContentBreastMilk, model.ContentFormula, model.ContentCowMilk, model.ContentOther}},
	} {
		b, err := s.CreateBottle(ctx, k.ID, ana.ID, newBottle(t, t0, c.in...), t0)
		if err != nil || !slices.Equal(b.Contents, c.want) {
			t.Fatalf("%s: created %v, %v; want %v", c.name, b.Contents, err, c.want)
		}
		got, err := s.Bottle(ctx, k.ID, b.ID)
		if err != nil || !slices.Equal(got.Contents, c.want) {
			t.Errorf("%s: read back %v, %v; want %v", c.name, got.Contents, err, c.want)
		}
	}

	valid := mustBottle(t, s, k.ID, ana.ID, newBottle(t, t0, model.ContentFormula))
	for name, contents := range map[string][]string{
		"empty":     {},
		"nil":       nil,
		"duplicate": {model.ContentFormula, model.ContentFormula},
		"unknown":   {"goat_milk"},
	} {
		if _, err := s.CreateBottle(ctx, k.ID, ana.ID, newBottle(t, t0, contents...), t0); !errors.Is(err, ErrInvalidContents) {
			t.Errorf("create %s: %v, want ErrInvalidContents", name, err)
		}
		bad := valid
		bad.Contents = contents
		if err := s.UpdateBottle(ctx, k.ID, ana.ID, bad, t0); !errors.Is(err, ErrInvalidContents) {
			t.Errorf("update %s: %v, want ErrInvalidContents", name, err)
		}
	}
	got, err := s.Bottle(ctx, k.ID, valid.ID)
	if err != nil || !slices.Equal(got.Contents, []string{model.ContentFormula}) {
		t.Errorf("refused updates changed the row: %v, %v", got.Contents, err)
	}
}

func TestListNewestFirstWithEqualTimestamps(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	same := t0.Add(-time.Hour)
	var ids []string
	for _, at := range []time.Time{same, same.Add(-time.Minute), same, same.Add(time.Minute), same} {
		ids = append(ids, mustBottle(t, s, k.ID, ana.ID, newBottle(t, at, model.ContentFormula)).ID)
	}
	atOf := map[string]int64{ids[0]: same.Unix(), ids[1]: same.Unix() - 60, ids[2]: same.Unix(), ids[3]: same.Unix() + 60, ids[4]: same.Unix()}
	want := slices.Clone(ids)
	slices.SortFunc(want, func(a, b string) int {
		if atOf[a] != atOf[b] {
			return int(atOf[b] - atOf[a])
		}
		return -strings.Compare(a, b)
	})

	got, err := s.ListBottles(ctx, k.ID, wideQuery())
	if err != nil {
		t.Fatal(err)
	}
	if gotIDs := bottleIDs(got); !slices.Equal(gotIDs, want) {
		t.Fatalf("order = %v, want %v", gotIDs, want)
	}

	// Paging by cursor visits every row once, in the same order.
	var paged []string
	q := wideQuery()
	q.Limit = 2
	for range ids {
		page, err := s.ListBottles(ctx, k.ID, q)
		if err != nil {
			t.Fatal(err)
		}
		if len(page) == 0 {
			break
		}
		paged = append(paged, bottleIDs(page)...)
		last := page[len(page)-1]
		q.After = &Cursor{At: last.At.Unix(), ID: last.ID}
	}
	if !slices.Equal(paged, want) {
		t.Errorf("paged = %v, want %v", paged, want)
	}
}

func TestListBottlesRangeIsHalfOpenAndLimited(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	var ats []time.Time
	for i := range 4 {
		at := t0.Add(-time.Duration(i) * time.Hour)
		ats = append(ats, at)
		mustBottle(t, s, k.ID, ana.ID, newBottle(t, at, model.ContentFormula))
	}
	got, err := s.ListBottles(ctx, k.ID, ListQuery{From: ats[2], To: ats[0], Limit: 10})
	if err != nil || len(got) != 2 || !got[0].At.Equal(ats[1]) || !got[1].At.Equal(ats[2]) {
		t.Fatalf("[ats[2], ats[0]) = %+v, %v", got, err)
	}
	got, err = s.ListBottles(ctx, k.ID, ListQuery{From: ats[3], To: t0.Add(time.Second), Limit: 1})
	if err != nil || len(got) != 1 || !got[0].At.Equal(ats[0]) {
		t.Fatalf("limit 1 = %+v, %v", got, err)
	}
}

func bottleIDs(bs []model.BottleFeed) []string {
	out := make([]string, len(bs))
	for i, b := range bs {
		out[i] = b.ID
	}
	return out
}
