package store

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/units"
)

func exportSnap(t *testing.T, s *Store, kidID string, modules []model.Module, types []string, rowCap int) ExportSnapshot {
	t.Helper()
	madrid := mustLoc(t, "Europe/Madrid")
	snap, err := s.ExportSnapshot(ctx, kidID, modules, types, madrid, mustDate(t, "2026-11-01", madrid), mustDate(t, "2026-11-20", madrid), t0, rowCap)
	if err != nil {
		t.Fatal(err)
	}
	return snap
}

func TestExportSnapshotIsOneTransaction(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	mustDiaper(t, s, k.ID, ana.ID, model.Diaper{At: t0.Add(-2 * time.Hour), Kind: "wet"})
	var wrote atomic.Bool
	countReads(t, s, func(n int64) {
		if n == 2 && wrote.CompareAndSwap(false, true) {
			mustDiaper(t, s, k.ID, ana.ID, model.Diaper{At: t0.Add(-time.Hour), Kind: "dry"})
			mustGrowth(t, s, k.ID, ana.ID, newGrowth(t, "2026-11-19"))
		}
	})
	snap := exportSnap(t, s, k.ID, []model.Module{"diapers", "growth"}, []string{"diaper"}, 10)
	if !wrote.Load() {
		t.Fatal("the write did not run between the reads")
	}
	if len(snap.Entries["diaper"]) != 1 || snap.Counts["diaper"] != 1 || len(snap.Growth) != 0 {
		t.Errorf("snapshot saw %d diapers (count %d) and %d measurements, want the state before the write",
			len(snap.Entries["diaper"]), snap.Counts["diaper"], len(snap.Growth))
	}
	var total int64
	for _, d := range snap.Days {
		total += d.Diapers.Total
	}
	if total != 1 {
		t.Errorf("summaries count %d diapers, want 1 from the same snapshot", total)
	}
	if after := exportSnap(t, s, k.ID, []model.Module{"growth"}, []string{"diaper"}, 10); len(after.Entries["diaper"]) != 2 || len(after.Growth) != 1 {
		t.Errorf("a later snapshot = %d diapers, %d measurements, want 2 and 1", len(after.Entries["diaper"]), len(after.Growth))
	}
}

func TestExportSnapshotReadsOnlyRequested(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	vol, err := units.NewVolume(units.UnitML, 90)
	if err != nil {
		t.Fatal(err)
	}
	mustBottle(t, s, k.ID, ana.ID, model.BottleFeed{At: t0.Add(-time.Hour), Contents: []string{model.ContentFormula}, Volume: vol})
	mustDiaper(t, s, k.ID, ana.ID, model.Diaper{At: t0.Add(-time.Hour), Kind: "wet"})
	n := countReads(t, s, nil)
	for _, c := range []struct {
		name    string
		modules []model.Module
		types   []string
		want    int64
	}{
		{"nothing", nil, nil, 1},
		{"diapers: list and one aggregate", []model.Module{"diapers"}, []string{"diaper"}, 3},
		{"bottles: list, sources and one aggregate", []model.Module{"bottles"}, []string{"bottle"}, 4},
		{"sleep with nothing to list", []model.Module{"sleep"}, []string{"sleep"}, 3},
		{"growth", []model.Module{"growth"}, nil, 2},
		{"milestones", []model.Module{"milestones"}, nil, 2},
		{"vaccination without a schedule", []model.Module{"vaccination"}, nil, 3},
		{"every daily tracker", nil, []string{"bottle", "nursing", "pumping", "sleep", "diaper", "meal"}, 1 + 6 + 1 + 6},
	} {
		n.Store(0)
		exportSnap(t, s, k.ID, c.modules, c.types, 10)
		if got := n.Load(); got != c.want {
			t.Errorf("%s: %d queries, want %d", c.name, got, c.want)
		}
	}
	schema := "us"
	kid := k
	kid.VaccineSchema = &schema
	if err := s.UpdateKid(ctx, kid, ana.ID, t0); err != nil {
		t.Fatal(err)
	}
	n.Store(0)
	exportSnap(t, s, k.ID, []model.Module{"vaccination"}, nil, 10)
	if got := n.Load(); got != 4 {
		t.Errorf("vaccination with a schedule: %d queries, want 4 (kid, doses, administrations, dismissals)", got)
	}
}

func TestExportSnapshotRowCap(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	for i := range 5 {
		mustDiaper(t, s, k.ID, ana.ID, model.Diaper{At: t0.Add(-time.Duration(i+1) * time.Hour), Kind: "wet"})
	}
	for _, c := range []struct{ rowCap, rows, count int }{{5, 5, 5}, {4, 5, 5}, {3, 4, 5}, {2, 3, 5}} {
		snap := exportSnap(t, s, k.ID, nil, []string{"diaper"}, c.rowCap)
		got := snap.Entries["diaper"]
		if len(got) != c.rows || snap.Counts["diaper"] != c.count {
			t.Errorf("cap %d: %d rows, count %d; want %d rows (cap+1 at most) and count %d", c.rowCap, len(got), snap.Counts["diaper"], c.rows, c.count)
		}
		if len(got) > 0 && !got[0].At.Equal(t0.Add(-time.Hour)) {
			t.Errorf("cap %d: first row at %v, want the newest", c.rowCap, got[0].At)
		}
	}
}

func TestExportSnapshotRangeIsKidLocalDays(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	madrid := mustLoc(t, "Europe/Madrid")
	in := []time.Time{time.Date(2026, 11, 1, 0, 0, 0, 0, madrid), time.Date(2026, 11, 20, 23, 59, 0, 0, madrid)}
	out := []time.Time{time.Date(2026, 10, 31, 23, 59, 0, 0, madrid), time.Date(2026, 11, 21, 0, 0, 0, 0, madrid)}
	for _, at := range append(in, out...) {
		mustDiaper(t, s, k.ID, ana.ID, model.Diaper{At: at, Kind: "wet"})
	}
	snap := exportSnap(t, s, k.ID, nil, []string{"diaper"}, 10)
	if len(snap.Entries["diaper"]) != 2 || len(snap.Days) != 20 || snap.Days[0].Date != "2026-11-01" || snap.Days[19].Date != "2026-11-20" {
		t.Errorf("entries %d, days %d, want 2 entries and the 20 days 2026-11-01..20", len(snap.Entries["diaper"]), len(snap.Days))
	}
}
