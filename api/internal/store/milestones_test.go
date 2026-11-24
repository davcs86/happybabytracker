package store

import (
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
)

const insertMilestone = `INSERT INTO milestones_reached(id, kid_id, catalog_item, custom_text, reached_on, note, created_by, updated_by, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, 100, 100)`

func itemMilestone(item, on string) model.Milestone {
	return model.Milestone{Item: strp(item), ReachedOn: on}
}

func textMilestone(text, on string) model.Milestone {
	return model.Milestone{Text: strp(text), ReachedOn: on}
}

func mustMilestone(t *testing.T, s *Store, kidID, actorID string, m model.Milestone) model.Milestone {
	t.Helper()
	got, err := s.CreateMilestone(ctx, kidID, actorID, m, t0)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestMilestoneCRUD(t *testing.T) {
	s := openMem(t)
	ana, bo := mustParent(t, s, "ana"), mustParent(t, s, "bo")
	k := mustKid(t, s, ana.ID, "Lía")
	mustGrant(t, s, k.ID, ana.ID, "bo", model.RoleCaregiver)

	for _, in := range []model.Milestone{
		{Item: strp("m2-social-1"), ReachedOn: "2026-02-01", Entry: model.Entry{Note: strp("at the park")}},
		{Text: strp("First word"), ReachedOn: "2026-03-01"},
	} {
		created, err := s.CreateMilestone(ctx, k.ID, ana.ID, in, t0)
		if err != nil {
			t.Fatal(err)
		}
		if created.ID == "" || created.KidID != k.ID || created.CreatedBy != ana.ID || created.UpdatedBy != ana.ID ||
			!created.CreatedAt.Equal(t0) || !created.UpdatedAt.Equal(t0) || created.ReachedOn != in.ReachedOn ||
			!reflect.DeepEqual(created.Item, in.Item) || !reflect.DeepEqual(created.Text, in.Text) || !reflect.DeepEqual(created.Note, in.Note) {
			t.Fatalf("created = %+v", created)
		}
		got, err := s.Milestone(ctx, k.ID, created.ID)
		if err != nil || !reflect.DeepEqual(got, created) {
			t.Fatalf("Milestone = %+v, %v; want %+v", got, err, created)
		}

		edit := got
		edit.ReachedOn, edit.Note = "2026-02-02", strp("edited")
		if edit.Text != nil {
			edit.Text = strp("First words")
		}
		later := t0.Add(time.Hour)
		if err := s.UpdateMilestone(ctx, k.ID, bo.ID, edit, later); err != nil {
			t.Fatal(err)
		}
		edit.UpdatedBy, edit.UpdatedAt = bo.ID, later
		if got, err = s.Milestone(ctx, k.ID, created.ID); err != nil || !reflect.DeepEqual(got, edit) {
			t.Fatalf("after update = %+v, %v; want %+v", got, err, edit)
		}
		if list, err := s.ListMilestones(ctx, k.ID); err != nil || len(list) != 1 || !reflect.DeepEqual(list[0], edit) {
			t.Fatalf("list = %+v, %v", list, err)
		}
		if err := s.DeleteMilestone(ctx, k.ID, bo.ID, created.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Milestone(ctx, k.ID, created.ID); !errors.Is(err, ErrNotFound) {
			t.Errorf("after delete: %v", err)
		}
		if err := s.DeleteMilestone(ctx, k.ID, bo.ID, created.ID); !errors.Is(err, ErrNotFound) {
			t.Errorf("second delete: %v", err)
		}
	}
	if list, err := s.ListMilestones(ctx, k.ID); err != nil || list == nil || len(list) != 0 {
		t.Errorf("empty list = %#v, %v; want an empty, non-nil slice", list, err)
	}
}

func TestMilestoneUpdateKeepsTheItem(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	m := mustMilestone(t, s, k.ID, ana.ID, itemMilestone("m2-social-1", "2026-02-01"))
	edit := m
	edit.Item = strp("m2-social-2")
	if err := s.UpdateMilestone(ctx, k.ID, ana.ID, edit, t0); err != nil {
		t.Fatal(err)
	}
	if got, err := s.Milestone(ctx, k.ID, m.ID); err != nil || got.Item == nil || *got.Item != "m2-social-1" {
		t.Errorf("item after update = %v, %v; want unchanged", got.Item, err)
	}
}

func TestMilestonesNewestFirstTieByID(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	var ids []string
	for i, on := range []string{"2026-02-01", "2026-02-01", "2026-02-01", "2026-03-01", "2026-01-15"} {
		ids = append(ids, mustMilestone(t, s, k.ID, ana.ID, textMilestone("custom "+string(rune('a'+i)), on)).ID)
	}
	ties := slices.Clone(ids[:3])
	slices.Sort(ties)
	slices.Reverse(ties)
	want := append(append([]string{ids[3]}, ties...), ids[4])

	list, err := s.ListMilestones(ctx, k.ID)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, len(list))
	for i, m := range list {
		got[i] = m.ID
	}
	if !slices.Equal(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
}

func TestMilestoneBelongsToItsKid(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	mine, other := mustKid(t, s, ana.ID, "Mine"), mustKid(t, s, ana.ID, "Other")
	m := mustMilestone(t, s, mine.ID, ana.ID, itemMilestone("m2-social-1", "2026-02-01"))

	if _, err := s.Milestone(ctx, other.ID, m.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("read through another kid: %v", err)
	}
	if err := s.UpdateMilestone(ctx, other.ID, ana.ID, m, t0); !errors.Is(err, ErrNotFound) {
		t.Errorf("update through another kid: %v", err)
	}
	if err := s.DeleteMilestone(ctx, other.ID, ana.ID, m.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("delete through another kid: %v", err)
	}
	if list, err := s.ListMilestones(ctx, other.ID); err != nil || len(list) != 0 {
		t.Errorf("other kid's list = %+v, %v", list, err)
	}
	if got, err := s.Milestone(ctx, mine.ID, m.ID); err != nil || !reflect.DeepEqual(got, m) {
		t.Errorf("entry changed: %+v, %v", got, err)
	}
}

func TestMilestoneWriteByDisabledActor(t *testing.T) {
	s := openMem(t)
	ana, bo, cy := mustParent(t, s, "ana"), mustParent(t, s, "bo"), mustParent(t, s, "cy")
	k := mustKid(t, s, ana.ID, "Lía")
	mustGrant(t, s, k.ID, ana.ID, "bo", model.RoleCaregiver)
	m := mustMilestone(t, s, k.ID, ana.ID, itemMilestone("m2-social-1", "2026-02-01"))
	if err := s.SetParentDisabled(ctx, bo.ID, true, t0); err != nil {
		t.Fatal(err)
	}

	for name, actor := range map[string]string{"disabled caregiver": bo.ID, "no grant": cy.ID} {
		if _, err := s.CreateMilestone(ctx, k.ID, actor, textMilestone("Claps", "2026-02-02"), t0); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s create: %v", name, err)
		}
		edit := m
		edit.ReachedOn = "2026-02-03"
		if err := s.UpdateMilestone(ctx, k.ID, actor, edit, t0); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s update: %v", name, err)
		}
		if err := s.DeleteMilestone(ctx, k.ID, actor, m.ID); !errors.Is(err, ErrNotFound) {
			t.Errorf("%s delete: %v", name, err)
		}
	}
	if list, err := s.ListMilestones(ctx, k.ID); err != nil || len(list) != 1 || !reflect.DeepEqual(list[0], m) {
		t.Errorf("list = %+v, %v; want the one seeded entry unchanged", list, err)
	}
}

func TestDuplicateCatalogItem(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k, other := mustKid(t, s, ana.ID, "Lía"), mustKid(t, s, ana.ID, "Noa")
	mustMilestone(t, s, k.ID, ana.ID, itemMilestone("m2-social-1", "2026-02-01"))
	if _, err := s.CreateMilestone(ctx, k.ID, ana.ID, itemMilestone("m2-social-1", "2026-02-05"), t0); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("second row for the item: %v, want ErrDuplicate", err)
	}
	if list, err := s.ListMilestones(ctx, k.ID); err != nil || len(list) != 1 {
		t.Errorf("list = %d rows, %v; want 1", len(list), err)
	}
	mustMilestone(t, s, other.ID, ana.ID, itemMilestone("m2-social-1", "2026-02-01"))
	mustMilestone(t, s, k.ID, ana.ID, textMilestone("Claps", "2026-02-01"))
	mustMilestone(t, s, k.ID, ana.ID, textMilestone("Claps", "2026-02-01")) // custom texts may repeat
}

func TestMilestoneCap(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k, other := mustKid(t, s, ana.ID, "Lía"), mustKid(t, s, ana.ID, "Noa")
	tx, err := s.w.Begin()
	if err != nil {
		t.Fatal(err)
	}
	for range constants.MaxMilestonesPerKid - 1 {
		if _, err := tx.Exec(insertMilestone, model.NewID(), k.ID, nil, "custom", "2026-02-01", nil, ana.ID, ana.ID); err != nil {
			tx.Rollback()
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	last := mustMilestone(t, s, k.ID, ana.ID, itemMilestone("m2-social-1", "2026-02-01"))
	if _, err := s.CreateMilestone(ctx, k.ID, ana.ID, textMilestone("Claps", "2026-02-02"), t0); !errors.Is(err, ErrConflict) {
		t.Fatalf("row %d: %v, want ErrConflict", constants.MaxMilestonesPerKid+1, err)
	}
	if list, err := s.ListMilestones(ctx, k.ID); err != nil || len(list) != constants.MaxMilestonesPerKid {
		t.Errorf("list = %d rows, %v; want %d", len(list), err, constants.MaxMilestonesPerKid)
	}
	mustMilestone(t, s, other.ID, ana.ID, textMilestone("Claps", "2026-02-02")) // the cap is per kid
	if err := s.DeleteMilestone(ctx, k.ID, ana.ID, last.ID); err != nil {
		t.Fatal(err)
	}
	mustMilestone(t, s, k.ID, ana.ID, textMilestone("Claps", "2026-02-03"))
}

func TestMilestoneChecks(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	n := any(nil)
	for i, c := range []struct {
		name             string
		ok               bool
		item, text, note any
		on               string
	}{
		{"catalog item", true, "m2-social-1", n, n, "2026-02-01"},
		{"custom text with a note", true, n, "Claps", "x", "2026-02-01"},
		{"leap day", true, n, "Claps", n, "2028-02-29"},
		{"both item and text", false, "m2-social-2", "Claps", n, "2026-02-01"},
		{"neither item nor text", false, n, n, n, "2026-02-01"},
		{"empty item", false, "", n, n, "2026-02-01"},
		{"empty text", false, n, "", n, "2026-02-01"},
		{"empty note", false, n, "Claps", "", "2026-02-01"},
		{"impossible date", false, n, "Claps", n, "2026-02-30"},
		{"unpadded date", false, n, "Claps", n, "2026-2-01"},
		{"date with a time", false, n, "Claps", n, "2026-02-01T00:00"},
		{"not a date", false, n, "Claps", n, "yesterday"},
		{"empty date", false, n, "Claps", n, ""},
	} {
		_, err := s.w.Exec(insertMilestone, "c"+string(rune('a'+i)), k.ID, c.item, c.text, c.on, c.note, ana.ID, ana.ID)
		if c.ok && err != nil {
			t.Errorf("%s: refused: %v", c.name, err)
		}
		if !c.ok && err == nil {
			t.Errorf("%s: accepted", c.name)
		}
	}
	if _, err := s.w.Exec(`INSERT INTO milestones_reached(id, kid_id, custom_text, reached_on, created_by, updated_by, created_at, updated_at)
		VALUES ('cz', ?, 'Claps', '2026-02-01', ?, ?, 'soon', 100)`, k.ID, ana.ID, ana.ID); err == nil {
		t.Error("a text created_at was accepted (STRICT)")
	}
}

func TestMilestonesCascadeWithKid(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k, other := mustKid(t, s, ana.ID, "Lía"), mustKid(t, s, ana.ID, "Noa")
	mustMilestone(t, s, k.ID, ana.ID, itemMilestone("m2-social-1", "2026-02-01"))
	mustMilestone(t, s, k.ID, ana.ID, textMilestone("Claps", "2026-02-01"))
	kept := mustMilestone(t, s, other.ID, ana.ID, textMilestone("Claps", "2026-02-01"))
	if err := s.DeleteKid(ctx, k.ID, ana.ID); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := s.w.QueryRow(`SELECT COUNT(*) FROM milestones_reached WHERE kid_id = ?`, k.ID).Scan(&n); err != nil || n != 0 {
		t.Errorf("%d rows of the deleted kid (err %v)", n, err)
	}
	if got, err := s.Milestone(ctx, other.ID, kept.ID); err != nil || !reflect.DeepEqual(got, kept) {
		t.Errorf("the other kid's row = %+v, %v", got, err)
	}
}
