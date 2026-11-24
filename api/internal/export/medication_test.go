package export

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/schedule"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

func course(id, name string, sc schedule.Schedule, start time.Time, endOn *string, active bool) model.Medication {
	return model.Medication{Entry: model.Entry{ID: id}, Schedule: sc, Name: name, Dose: name + " dose text", StartAt: start, EndOn: endOn, Active: active}
}

func dose(id, course string, at time.Time, amount, note *string) model.MedicationDose {
	return model.MedicationDose{Entry: model.Entry{ID: id, Note: note}, MedicationID: course, GivenAt: at, Amount: amount}
}

func medicationSnap(courses []model.Medication, doses ...model.MedicationDose) store.ExportSnapshot {
	snap := vaccineSnap(nil, nil)
	snap.Medications, snap.MedicationDoses = courses, doses
	snap.Counts = map[string]int{"medication": len(courses), "medication_dose": len(doses)}
	return snap
}

func TestExportMedication(t *testing.T) {
	start := time.Date(2026, 10, 20, 8, 0, 0, 0, time.UTC)
	courses := []model.Medication{
		course("m1", "Zentrolex", schedule.Schedule{Kind: schedule.KindTimes, Times: []string{"08:00", "20:00"}}, start, ptr("2026-11-05"), false),
		course("m2", "Quorbitan", schedule.Schedule{Kind: schedule.KindEvery, EveryHours: 6}, start.Add(48*time.Hour), nil, true),
		course("m3", "Plixamol", schedule.Schedule{Kind: schedule.KindNone}, start.Add(72*time.Hour), nil, true),
		course("m4", "Vorrelin", schedule.Schedule{Kind: schedule.KindEvery, EveryHours: 1}, start.Add(96*time.Hour), nil, true),
	}
	at := func(d, h int) time.Time { return time.Date(2026, 11, d, h, 15, 0, 0, time.UTC) }
	// Newest first, as the store returns them.
	snap := medicationSnap(courses,
		dose("d3", "m3", at(18, 9), nil, nil),
		dose("d2", "m2", at(10, 14), ptr("1.5 ml"), ptr("with\tyogurt")),
		dose("d1", "m1", at(2, 8), ptr("2 drops"), nil),
	)
	strs := renderWith(t, snap, metric, time.UTC, "medication")
	text := joined(strs)
	wantIn(t, "courses", text, "Medication", "Zentrolex", "Zentrolex dose text", "08:00, 20:00", "2026-10-20 08:00", "2026-11-05", "Not active",
		"Quorbitan", "Every 6 hours from 08:00", "2026-10-22 08:00", "Plixamol", "As needed", "Every hour from 08:00")
	if a, n := countOf(strs, "Active"), countOf(strs, "Not active"); a != 4 || n != 1 {
		t.Errorf("%d Active (want the column and three courses) and %d Not active (want 1): %q", a, n, strs)
	}
	wantIn(t, "doses", text, "Given at", "2026-11-02 08:15 Zentrolex 2 drops", "2026-11-10 14:15 Quorbitan 1.5 ml with yogurt", "2026-11-18 09:15 Plixamol")
	if a, b, c := strings.Index(text, "2026-11-02 08:15"), strings.Index(text, "2026-11-10 14:15"), strings.Index(text, "2026-11-18 09:15"); !(a < b && b < c) {
		t.Errorf("doses are not oldest first: %q", text)
	}
	app := appWords(text, "Zentrolex", "Quorbitan", "Plixamol", "Vorrelin", " dose text", "1.5 ml", "2 drops", "with yogurt")
	wantNotIn(t, "no interpretation", strings.ToLower(app), "next", "last dose", "doses", "count", "expected", "missed")
	if m := sectionDeny.FindString(app); m != "" {
		t.Errorf("%q in the medication section", m)
	}

	empty := renderWith(t, medicationSnap(nil), metric, time.UTC, "medication")
	if j := slices.Index(empty, noEntries); j < 2 || empty[j-1] != "Medication" || empty[j-2] != "Medication" {
		t.Errorf("empty medication: %q", empty)
	}
	noneGiven := joined(renderWith(t, medicationSnap(courses[:1]), metric, time.UTC, "medication"))
	wantIn(t, "no doses", noneGiven, "Zentrolex", noDoses)

	if _, err := medicationItems(medicationSnap(courses[:1], dose("dx", "gone", at(2, 8), nil, nil)), Request{Loc: time.UTC}, &budget{left: maxRows}); err == nil {
		t.Error("a dose of a course not in the snapshot rendered")
	}
}

func TestMedicationBudget(t *testing.T) {
	savedSection := maxRowsPerSection
	maxRowsPerSection = 3
	t.Cleanup(func() { maxRowsPerSection = savedSection })
	start := time.Date(2026, 10, 20, 8, 0, 0, 0, time.UTC)
	courses := []model.Medication{course("m1", "Zentrolex", schedule.Schedule{Kind: schedule.KindNone}, start, nil, true)}
	at := func(d int) time.Time { return time.Date(2026, 11, d, 9, 0, 0, 0, time.UTC) }
	snap := medicationSnap(courses, dose("d4", "m1", at(4), ptr("four"), nil), dose("d3", "m1", at(3), ptr("three"), nil),
		dose("d2", "m1", at(2), ptr("two"), nil))
	snap.Counts["medication_dose"] = 7
	text := joined(renderWith(t, snap, metric, time.UTC, "medication"))
	wantIn(t, "budget", text, "three", "four", "5"+overflow)
	wantNotIn(t, "budget", text, " two")
	if strings.Index(text, "three") > strings.Index(text, "four") {
		t.Error("the shown doses are not oldest first")
	}
}
