package vaccines

import (
	"errors"
	"io/fs"
	"path"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

func TestLoadIsIdempotent(t *testing.T) {
	for range 3 {
		if err := Load(); err != nil {
			t.Fatalf("Load() = %v, want nil", err)
		}
	}
}

func TestSchemasInShippedOrder(t *testing.T) {
	mustLoad(t)
	var got []string
	for _, s := range Schemas() {
		got = append(got, s.ID)
		g, ok := GetSchema(s.ID)
		if !ok || g.ID != s.ID || len(g.Lines) != len(s.Lines) {
			t.Errorf("GetSchema(%s) = %v, %v", s.ID, g.ID, ok)
		}
	}
	var want []string
	for _, id := range constants.VaccineSchedules {
		if _, err := fs.Stat(dataFS, path.Join("data", id+".csv")); err == nil {
			want = append(want, id)
		}
	}
	if !slices.Equal(got, want) {
		t.Errorf("Schemas() = %v, want %v", got, want)
	}
	if !slices.Equal(constants.VaccineSchedules, []string{"us", "uk", "mx"}) {
		t.Errorf("VaccineSchedules = %v", constants.VaccineSchedules)
	}
	if _, ok := GetSchema("ch"); ok {
		t.Error("GetSchema(ch) found a schema")
	}
}

func TestLoadTurnsAPanicIntoAnError(t *testing.T) {
	t.Cleanup(func() { resetLoad(load) })
	for _, l := range []func() error{
		func() error { panic("corrupt schedule") },
		func() error { return errors.New("vaccines: us.csv:10: column kind: unknown kind") },
	} {
		resetLoad(l)
		for range 3 {
			if err := Load(); !errors.Is(err, ErrSchedulesUnavailable) {
				t.Fatalf("Load() = %v, want ErrSchedulesUnavailable", err)
			}
		}
		if Schemas() != nil || Antigens() != nil || Products() != nil {
			t.Error("accessors answered after a failed load")
		}
		if _, ok := GetSchema("us"); ok {
			t.Error("GetSchema answered ok after a failed load")
		}
		if err := ValidAntigens([]string{"measles"}); !errors.Is(err, ErrSchedulesUnavailable) {
			t.Errorf("ValidAntigens = %v, want ErrSchedulesUnavailable", err)
		}
	}
}

func TestLoadRefusesAnUnshippedScheduleFile(t *testing.T) {
	fsys := fstest.MapFS{}
	for _, name := range []string{"data/antigens.csv", "data/products.csv"} {
		b, err := dataFS.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		fsys[name] = &fstest.MapFile{Data: b}
	}
	ch := strings.Replace(testFile("measles-1,MMR,measles,1,12m,16m,routine,,,"), "# schema: us", "# schema: ch", 1)
	fsys["data/ch.csv"] = &fstest.MapFile{Data: []byte(ch)}
	t.Cleanup(func() { resetLoad(load) })
	resetLoad(func() error { return loadFrom(fsys) })
	err := Load()
	want := "vaccines: data/ch.csv:1: column schema: not a shipped schedule id"
	if !errors.Is(err, ErrSchedulesUnavailable) || !strings.Contains(err.Error(), want) {
		t.Errorf("Load() = %v, want %q", err, want)
	}
	if Schemas() != nil {
		t.Error("Schemas() answered after a failed load")
	}
}

func resetLoad(l func() error) {
	loader, once, loadErr = l, sync.Once{}, nil
	antigens, antigenIdx, products, schemas = nil, nil, nil, nil
}
