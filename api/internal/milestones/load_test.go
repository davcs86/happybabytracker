package milestones

import (
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
)

func resetLoad(l func() error) {
	loader, once, loadErr = l, sync.Once{}, nil
	catalog, byID = Catalog{}, nil
}

func loadTest(t *testing.T, in string) {
	t.Helper()
	t.Cleanup(func() { resetLoad(load) })
	fsys := fstest.MapFS{catalogFile: &fstest.MapFile{Data: []byte(in)}}
	resetLoad(func() error { return loadFrom(fsys) })
	if err := Load(); err != nil {
		t.Fatalf("Load() = %v", err)
	}
}

func TestLoadIsIdempotent(t *testing.T) {
	calls := 0
	fsys := fstest.MapFS{catalogFile: &fstest.MapFile{Data: []byte(testFile(rowA, rowB))}}
	t.Cleanup(func() { resetLoad(load) })
	resetLoad(func() error { calls++; return loadFrom(fsys) })
	for range 3 {
		if err := Load(); err != nil {
			t.Fatalf("Load() = %v, want nil", err)
		}
	}
	if calls != 1 {
		t.Errorf("loader ran %d times, want 1", calls)
	}
	c, ok := Get()
	if !ok || c.ID != "cdc" || len(c.Items) != 2 {
		t.Errorf("Get() = %+v, %v", c, ok)
	}
}

func TestLoadRemembersAFailure(t *testing.T) {
	t.Cleanup(func() { resetLoad(load) })
	for _, l := range []func() error{
		func() error { panic("corrupt catalog") },
		func() error { return errors.New("milestones: data/cdc.csv:12: column id: duplicate id") },
	} {
		calls := 0
		resetLoad(func() error { calls++; return l() })
		for range 3 {
			if err := Load(); !errors.Is(err, ErrCatalogUnavailable) {
				t.Fatalf("Load() = %v, want ErrCatalogUnavailable", err)
			}
		}
		if calls != 1 {
			t.Errorf("loader ran %d times, want 1", calls)
		}
		if _, ok := Get(); ok {
			t.Error("Get answered ok after a failed load")
		}
		if _, ok := Lookup("2m-se-calms-down"); ok {
			t.Error("Lookup answered ok after a failed load")
		}
		if Ages() != nil {
			t.Error("Ages answered after a failed load")
		}
	}
}

func TestLoadFromNamesAParseFailure(t *testing.T) {
	t.Cleanup(func() { resetLoad(load) })
	bad := strings.Replace(testFile(rowA), "2m-se-calms-down", "2M-se-calms-down", 1)
	fsys := fstest.MapFS{catalogFile: &fstest.MapFile{Data: []byte(bad)}}
	resetLoad(func() error { return loadFrom(fsys) })
	err := Load()
	want := "milestones: data/cdc.csv:12: column id: not <n>m-<se|lc|cg|mp>-<slug of a-z, 0-9 and hyphens>"
	if !errors.Is(err, ErrCatalogUnavailable) || !strings.Contains(err.Error(), want) {
		t.Errorf("Load() = %v, want %q", err, want)
	}
}

func TestLoadFromAMissingFile(t *testing.T) {
	t.Cleanup(func() { resetLoad(load) })
	resetLoad(func() error { return loadFrom(fstest.MapFS{}) })
	if err := Load(); !errors.Is(err, ErrCatalogUnavailable) {
		t.Errorf("Load() = %v, want ErrCatalogUnavailable", err)
	}
}

func TestLookupFindsRetiredItems(t *testing.T) {
	loadTest(t, testFile(rowA, rowC))
	it, ok := Lookup("4m-lc-makes-sounds")
	if !ok || it.Retired != "2026-11-24" || it.Text != "Makes sounds" || it.Domain != DomainLanguage {
		t.Errorf("Lookup(retired) = %+v, %v", it, ok)
	}
	if it, ok := Lookup("2m-se-calms-down"); !ok || it.Retired != "" {
		t.Errorf("Lookup(active) = %+v, %v", it, ok)
	}
	if _, ok := Lookup("2m-se-unknown"); ok {
		t.Error("Lookup found an unknown id")
	}
}

func TestAgesAreDistinctActiveAndAscending(t *testing.T) {
	loadTest(t, testFile(
		"12m-mp-walks,12,movement_physical,Walks,"+goodURL+",",
		rowA, rowB,
		"6m-cg-looks,6,cognitive,Looks,"+goodURL+",",
		"9m-cg-plays,9,cognitive,Plays,"+goodURL+",2026-11-24",
		"6m-se-smiles,6,social_emotional,Smiles,"+goodURL+",",
	))
	if got, want := Ages(), []int{2, 6, 12}; !slices.Equal(got, want) {
		t.Errorf("Ages() = %v, want %v", got, want)
	}
}

func TestAccessorsReturnCopies(t *testing.T) {
	loadTest(t, testFile(rowA, rowB))
	c, _ := Get()
	c.Items[0].Text = "changed"
	c.Omitted[0] = "changed"
	a := Ages()
	a[0] = 99
	again, _ := Get()
	if again.Items[0].Text != "Calms down when spoken to" || again.Omitted[0] != "tips" || Ages()[0] != 2 {
		t.Errorf("a caller's change reached package state: %+v", again)
	}
}
