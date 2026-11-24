package export

import (
	"bytes"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/export/exporttest"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

var renderNow = time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)

func diaperItems(n int) []store.TimelineItem {
	out := make([]store.TimelineItem, n)
	for i := range out {
		at := renderNow.Add(-time.Duration(i+1) * time.Minute)
		out[i] = store.TimelineItem{Type: "diaper", At: at, ID: strconv.Itoa(i), Entry: model.Diaper{At: at, Kind: "wet"}}
	}
	return out
}

func renderStrings(t *testing.T, snap store.ExportSnapshot, modules ...model.Module) []string {
	t.Helper()
	var buf bytes.Buffer
	req := Request{From: "2026-11-01", To: "2026-11-20", Modules: modules, Units: metric, Now: renderNow, Loc: time.UTC}
	if err := Render(&buf, snap, req); err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, p := range exporttest.PageTexts(t, buf.Bytes()) {
		out = append(out, exporttest.Strings(p)...)
	}
	return out
}

func TestRowBudget(t *testing.T) {
	savedSection, savedAll := maxRowsPerSection, maxRows
	maxRowsPerSection, maxRows = 3, 5
	t.Cleanup(func() { maxRowsPerSection, maxRows = savedSection, savedAll })

	kid := model.Kid{Name: "Ada", BirthDate: "2026-03-04", Sex: "female", TimeZone: "UTC", GrowthSource: "who"}
	// A snapshot holds at most rowCap+1 rows of a longer list; Counts holds the whole count.
	snap := store.ExportSnapshot{Kid: kid,
		Entries: map[string][]store.TimelineItem{"nursing": nursingItems(4), "pumping": pumpingItems(4), "diaper": diaperItems(2)},
		Counts:  map[string]int{"nursing": 9, "pumping": 4, "diaper": 2},
	}
	strs := renderStrings(t, snap, "diapers", "pumping", "nursing")

	for prefix, want := range map[string]int{"Nursing L 10 min": 3, "Pumping 10 min": 2, "Diaper wet": 0} {
		if got := countOf(strs, prefix); got != want {
			t.Errorf("%q: %d rows, want %d", prefix, got, want)
		}
	}
	if n := countOf(strs, overflow(6)); n != 1 {
		t.Errorf("nursing's overflow line appears %d times, want once", n)
	}
	if n := countOf(strs, overflow(2)); n != 2 {
		t.Errorf("%q appears %d times, want once for pumping and once for diapers", overflow(2), n)
	}
	i := slices.Index(strs, "Diapers")
	if i < 0 || strs[i+1] != overflow(2) || slices.Contains(strs[i:], "Day total") || slices.Contains(strs[i:], "Time") {
		t.Errorf("after the budget a section prints only its heading and overflow line: %q", strs[max(0, i):])
	}
}

func countOf(strs []string, s string) int {
	n := 0
	for _, x := range strs {
		if x == s || strings.HasPrefix(x, s+",") || strings.HasPrefix(x, s+" (") {
			n++
		}
	}
	return n
}

func nursingItems(n int) []store.TimelineItem {
	out := make([]store.TimelineItem, n)
	for i := range out {
		at := renderNow.Add(-time.Duration(i+1) * time.Hour)
		end := at.Add(10 * time.Minute)
		out[i] = store.TimelineItem{Type: "nursing", At: at, ID: strconv.Itoa(i), Entry: model.NursingSession{StartedAt: at, EndedAt: &end, LeftSeconds: 600}}
	}
	return out
}

func pumpingItems(n int) []store.TimelineItem {
	out := make([]store.TimelineItem, n)
	for i := range out {
		at := renderNow.Add(-time.Duration(i+1) * time.Hour)
		out[i] = store.TimelineItem{Type: "pumping", At: at, ID: strconv.Itoa(i), Entry: model.PumpingSession{StartedAt: at, DurationSeconds: 600, Side: "left"}}
	}
	return out
}

func TestRenderSkipsModulesWithoutAWriter(t *testing.T) {
	kid := model.Kid{Name: "Ada", BirthDate: "2026-03-04", Sex: "male", TimeZone: "UTC", GrowthSource: "cdc"}
	strs := renderStrings(t, store.ExportSnapshot{Kid: kid}, "growth", "milestones")
	if i, j := slices.Index(strs, "Milestones"), slices.Index(strs, "Growth"); i < 0 || i > j || !slices.Contains(strs[i:j], noMilestones) {
		t.Errorf("milestones section missing or not before growth: %q", strs)
	}
	if i := slices.Index(strs, "Growth"); i < 0 || strs[i+1] != noMeasurements {
		t.Errorf("growth section missing or not empty: %q", strs)
	}
	for _, want := range []string{"Profile", "Male", "Growth source", "CDC 2000 Growth Charts", "Milestones, Growth"} {
		if !slices.Contains(strs, want) {
			t.Errorf("%q missing from %q", want, strs)
		}
	}
}

func TestOverflowLine(t *testing.T) {
	for _, c := range []struct{ got, want string }{
		{overflow(1), "1 more entry in this range is not shown. Export a shorter range to see it."},
		{overflow(2), "2 more entries in this range are not shown. Export a shorter range to see them."},
		{growthOverflow(1), "1 more entry up to the end of the range is not shown. Export a range that ends earlier to see it."},
		{growthOverflow(40), "40 more entries up to the end of the range are not shown. Export a range that ends earlier to see them."},
	} {
		if c.got != c.want {
			t.Errorf("got %q, want %q", c.got, c.want)
		}
	}
}
