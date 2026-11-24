package store

import (
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/schedule"
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
		{"sickness: symptoms and no aggregate", []model.Module{"sickness"}, []string{"symptom"}, 2},
		{"medication: courses and doses", []model.Module{"medication"}, nil, 3},
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

// TestExportSnapshotMedication pins Open value 7 (R1): every course whose kid-local span touches the range, active or
// not, and any course with a dose in the range; then the doses in the range, newest first.
func TestExportSnapshotMedication(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKidIn(t, s, ana.ID, "Lía", "Europe/Madrid")
	madrid := mustLoc(t, "Europe/Madrid")
	oct1 := time.Date(2026, 10, 1, 9, 0, 0, 0, madrid)
	add := func(name string, start time.Time, endOn *string, active bool) model.Medication {
		return mustCourse(t, s, k.ID, ana.ID, model.Medication{Name: name, Dose: "1 ml", Schedule: schedule.Schedule{Kind: schedule.KindNone},
			StartAt: start, EndOn: endOn, Active: active}, t0)
	}
	day := func(d string) *string { return &d }
	ended := add("ended before", oct1, day("2026-10-31"), true)
	lastDay := add("last day is from", oct1, day("2026-11-01"), false)
	lateStart := add("starts on to", time.Date(2026, 11, 20, 23, 30, 0, 0, madrid), nil, true)
	add("starts after to", time.Date(2026, 11, 21, 0, 0, 0, 0, madrid), nil, true)
	dosed := add("dose in range", oct1.Add(time.Hour), day("2026-10-15"), true)
	open := add("never ended, inactive", oct1.Add(2*time.Hour), nil, false)
	in := []time.Time{time.Date(2026, 11, 1, 0, 0, 0, 0, madrid), time.Date(2026, 11, 5, 10, 0, 0, 0, madrid), time.Date(2026, 11, 20, 23, 59, 0, 0, madrid)}
	for _, at := range in {
		mustDose(t, s, k.ID, ana.ID, model.MedicationDose{MedicationID: dosed.ID, GivenAt: at}, t0)
	}
	for _, at := range []time.Time{time.Date(2026, 10, 31, 23, 59, 0, 0, madrid), time.Date(2026, 11, 21, 0, 0, 0, 0, madrid)} {
		mustDose(t, s, k.ID, ana.ID, model.MedicationDose{MedicationID: ended.ID, GivenAt: at}, t0)
	}

	snap := exportSnap(t, s, k.ID, []model.Module{"medication"}, nil, 10)
	var names []string
	for _, m := range snap.Medications {
		names = append(names, m.Name)
	}
	want := []string{lastDay.Name, dosed.Name, open.Name, lateStart.Name}
	if !slices.Equal(names, want) || snap.Counts["medication"] != 4 {
		t.Errorf("courses = %q (count %d), want %q oldest start first", names, snap.Counts["medication"], want)
	}
	for _, m := range snap.Medications {
		if m.LastDose != nil || m.NextDue != nil || m.DoseCount != 0 {
			t.Errorf("%s carries last dose, next due or a count", m.Name)
		}
	}
	if len(snap.MedicationDoses) != 3 || snap.Counts["medication_dose"] != 3 || !snap.MedicationDoses[0].GivenAt.Equal(in[2]) || !snap.MedicationDoses[2].GivenAt.Equal(in[0]) {
		t.Errorf("doses = %+v (count %d), want the three in range, newest first", snap.MedicationDoses, snap.Counts["medication_dose"])
	}

	capped := exportSnap(t, s, k.ID, []model.Module{"medication"}, nil, 1)
	if len(capped.Medications) != 2 || capped.Counts["medication"] != 4 || len(capped.MedicationDoses) != 2 || capped.Counts["medication_dose"] != 3 {
		t.Errorf("cap 1: %d courses (count %d), %d doses (count %d); want cap+1 rows and the whole counts",
			len(capped.Medications), capped.Counts["medication"], len(capped.MedicationDoses), capped.Counts["medication_dose"])
	}
	if other := exportSnap(t, s, k.ID, []model.Module{"growth"}, nil, 10); other.Medications != nil || other.MedicationDoses != nil {
		t.Error("medication read without being requested")
	}
}

func TestExportSnapshotSymptoms(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	madrid := mustLoc(t, "Europe/Madrid")
	for _, at := range []time.Time{time.Date(2026, 10, 31, 23, 59, 0, 0, madrid), time.Date(2026, 11, 1, 0, 0, 0, 0, madrid),
		time.Date(2026, 11, 20, 23, 59, 0, 0, madrid), time.Date(2026, 11, 21, 0, 0, 0, 0, madrid)} {
		mustSymptom(t, s, k.ID, ana.ID, model.Symptom{At: at, Symptom: "cough"})
	}
	snap := exportSnap(t, s, k.ID, []model.Module{"sickness"}, []string{"symptom"}, 10)
	if got := snap.Entries["symptom"]; len(got) != 2 || snap.Counts["symptom"] != 2 || got[0].Entry.(model.Symptom).Symptom != "cough" {
		t.Errorf("symptoms = %d (count %d), want the 2 in range", len(got), snap.Counts["symptom"])
	}
	if capped := exportSnap(t, s, k.ID, []model.Module{"sickness"}, []string{"symptom"}, 0); len(capped.Entries["symptom"]) != 1 || capped.Counts["symptom"] != 2 {
		t.Errorf("cap 0: %d rows, count %d; want 1 and 2", len(capped.Entries["symptom"]), capped.Counts["symptom"])
	}
}
