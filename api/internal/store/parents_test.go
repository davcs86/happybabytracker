package store

import (
	"errors"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
)

func TestListParents(t *testing.T) {
	s := openMem(t)
	if got, err := s.ListParents(ctx); err != nil || got == nil || len(got) != 0 {
		t.Fatalf("empty list = %#v, %v; want a non-nil empty slice", got, err)
	}
	bob := mustParent(t, s, "bob")
	ana := mustParent(t, s, "ana")
	if err := s.SetPassword(ctx, ana.ID, "$hash", "", t0); err != nil {
		t.Fatal(err)
	}
	if err := s.SetParentDisabled(ctx, bob.ID, true, t0); err != nil {
		t.Fatal(err)
	}

	got, err := s.ListParents(ctx)
	if err != nil || len(got) != 2 {
		t.Fatalf("ListParents = %+v, %v", got, err)
	}
	if got[0].Username != "ana" || !got[0].HasPassword || got[0].Disabled {
		t.Errorf("first = %+v", got[0])
	}
	if got[1].Username != "bob" || got[1].HasPassword || !got[1].Disabled {
		t.Errorf("second = %+v", got[1])
	}
	if got[0].ID != ana.ID || got[0].DisplayName != "Name ana" || got[0].CreatedAt != t0 {
		t.Errorf("fields = %+v", got[0])
	}
}

func TestUpdateParentProfile(t *testing.T) {
	s := openMem(t)
	p := mustParent(t, s, "ana")
	later := t0.Add(time.Minute)
	units := model.Units{Weight: "lb", Length: "in", Volume: "floz", Temp: "f"}
	if err := s.UpdateParentProfile(ctx, p.ID, "Ana Díaz", units, later); err != nil {
		t.Fatal(err)
	}
	got, err := s.ParentByID(ctx, p.ID)
	if err != nil || got.DisplayName != "Ana Díaz" || got.Units != units {
		t.Fatalf("after update: %+v, %v", got, err)
	}
	var updated int64
	if err := s.r.QueryRow(`SELECT updated_at FROM parents WHERE id = ?`, p.ID).Scan(&updated); err != nil || updated != later.Unix() {
		t.Errorf("updated_at = %d, %v; want %d", updated, err, later.Unix())
	}
	if err := s.UpdateParentProfile(ctx, "nobody", "X", units, later); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown parent: %v, want ErrNotFound", err)
	}
	bad := units
	bad.Weight = "stone"
	if err := s.UpdateParentProfile(ctx, p.ID, "Ana", bad, later); err == nil {
		t.Error("a unit outside the schema's set was stored")
	}
}

func TestLocalSecret(t *testing.T) {
	s := openMem(t)
	p := mustParent(t, s, "ana")
	if _, err := s.LocalSecret(ctx, p.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("no password yet: %v, want ErrNotFound", err)
	}
	if err := s.SetPassword(ctx, p.ID, "$hash$one", "", t0); err != nil {
		t.Fatal(err)
	}
	if got, err := s.LocalSecret(ctx, p.ID); err != nil || got != "$hash$one" {
		t.Errorf("LocalSecret = %q, %v", got, err)
	}
	if _, err := s.LocalSecret(ctx, "nobody"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown parent: %v, want ErrNotFound", err)
	}
}
