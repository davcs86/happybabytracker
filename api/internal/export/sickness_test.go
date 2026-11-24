package export

import (
	"bytes"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/schedule"
	"github.com/davcs86/happybabytracker/api/internal/store"
	"github.com/davcs86/happybabytracker/api/internal/units"
)

func renderWith(t *testing.T, snap store.ExportSnapshot, u model.Units, loc *time.Location, modules ...model.Module) []string {
	t.Helper()
	var buf bytes.Buffer
	req := Request{From: "2026-11-01", To: "2026-11-20", Modules: modules, Units: u, Now: renderNow, Loc: loc}
	if err := Render(&buf, snap, req); err != nil {
		t.Fatal(err)
	}
	return pdfText(t, buf.Bytes())
}

func symptomItem(id string, at time.Time, v model.Symptom) store.TimelineItem {
	v.ID, v.At = id, at
	return store.TimelineItem{Type: "symptom", At: at, ID: id, Entry: v}
}

func temp(t *testing.T, unit units.Unit, hundredths int64) *units.Temperature {
	t.Helper()
	v, err := units.NewTemperature(unit, hundredths)
	if err != nil {
		t.Fatal(err)
	}
	return &v
}

func sicknessSnap(items ...store.TimelineItem) store.ExportSnapshot {
	snap := vaccineSnap(nil, nil)
	snap.Entries = map[string][]store.TimelineItem{"symptom": items}
	snap.Counts = map[string]int{"symptom": len(items)}
	return snap
}

// appWords is text with the user-entered strings removed, so only the app's own words are checked.
func appWords(text string, user ...string) string {
	for _, u := range user {
		text = strings.ReplaceAll(text, u, "")
	}
	return text
}

var sectionDeny = regexp.MustCompile(`(?i)\b(late|behind|delayed|delay|normal|abnormal|typical|atypical|on track|ahead|missed|missing|concern|worry|should|red flag|warning|due|overdue|reminder)\b`)

func TestExportSickness(t *testing.T) {
	day := func(h int) time.Time { return time.Date(2026, 11, 18, h, 0, 0, 0, time.UTC) }
	// The store returns rows newest first; the section prints them oldest first.
	snap := sicknessSnap(
		symptomItem("s4", day(12), model.Symptom{Symptom: "other", OtherText: ptr("hiccups"), Entry: model.Entry{Note: ptr("after\nlunch")}}),
		symptomItem("s3", day(11), model.Symptom{Symptom: "cough", Severity: ptr("moderate")}),
		symptomItem("s2", day(10), model.Symptom{Symptom: "fever", Temperature: temp(t, units.UnitF, 10130), Method: ptr("axillary"), Severity: ptr("severe")}),
		symptomItem("s1", day(9), model.Symptom{Symptom: "fever", Temperature: temp(t, units.UnitC, 3860), Method: ptr("oral"), Entry: model.Entry{Note: ptr("cool cloth")}}),
	)
	c := renderWith(t, snap, metric, time.UTC, "sickness")
	f := renderWith(t, snap, imperial, time.UTC, "sickness")
	wantIn(t, "°C", joined(c), "Sickness", "2026-11-18 09:00", "Fever 38.6 °C (oral)", "cool cloth", "2026-11-18 10:00", "Fever 38.5 °C (armpit), severe",
		"2026-11-18 11:00", "Cough, moderate", "2026-11-18 12:00", "Other: hiccups", "after lunch")
	wantNotIn(t, "°C", joined(c), "°F")
	wantIn(t, "°F", joined(f), "Fever 101.48 °F (oral)", "Fever 101.3 °F (armpit), severe")
	wantNotIn(t, "°F", joined(f), "°C")
	order := []int{slices.Index(c, "Fever 38.6 °C (oral)"), slices.Index(c, "Fever 38.5 °C (armpit), severe"), slices.Index(c, "Cough, moderate"),
		slices.Index(c, "Other: hiccups")}
	if slices.Contains(order, -1) || !slices.IsSorted(order) {
		t.Errorf("rows are not oldest first: %v in %q", order, c)
	}
	wantNotIn(t, "no summary", joined(c), dayTotal)
	if m := sectionDeny.FindString(appWords(joined(c), "hiccups", "after lunch", "cool cloth")); m != "" {
		t.Errorf("%q in the sickness section", m)
	}

	empty := renderWith(t, sicknessSnap(), metric, time.UTC, "sickness")
	if j := slices.Index(empty, noEntries); j < 2 || empty[j-1] != "Sickness" || empty[j-2] != "Sickness" {
		t.Errorf("empty sickness: %q", empty)
	}
}

func TestSicknessBudget(t *testing.T) {
	savedSection := maxRowsPerSection
	maxRowsPerSection = 2
	t.Cleanup(func() { maxRowsPerSection = savedSection })
	day := func(h int) time.Time { return time.Date(2026, 11, 18, h, 0, 0, 0, time.UTC) }
	snap := sicknessSnap(
		symptomItem("s3", day(11), model.Symptom{Symptom: "rash"}),
		symptomItem("s2", day(10), model.Symptom{Symptom: "cough"}),
		symptomItem("s1", day(9), model.Symptom{Symptom: "vomiting"}),
	)
	snap.Counts["symptom"] = 5
	text := joined(renderWith(t, snap, metric, time.UTC, "sickness"))
	wantIn(t, "budget", text, "Cough", "Rash", "3"+overflow)
	wantNotIn(t, "budget", text, "Vomiting")
	if strings.Index(text, "Cough") > strings.Index(text, "Rash") {
		t.Error("the shown rows are not oldest first")
	}
}

func TestExportSicknessMedicationDST(t *testing.T) {
	ny := mustZone(t, "America/New_York")
	edt, est := time.Date(2026, 11, 1, 5, 30, 0, 0, time.UTC), time.Date(2026, 11, 1, 6, 30, 0, 0, time.UTC)
	snap := sicknessSnap(symptomItem("s2", est, model.Symptom{Symptom: "rash"}), symptomItem("s1", edt, model.Symptom{Symptom: "cough"}))
	snap.Medications = []model.Medication{course("m1", "Zentrolex", schedule.Schedule{Kind: schedule.KindNone}, edt.Add(-time.Hour), nil, true)}
	snap.MedicationDoses = []model.MedicationDose{dose("d2", "m1", est, ptr("second"), nil), dose("d1", "m1", edt, ptr("first"), nil)}
	snap.Counts["medication"], snap.Counts["medication_dose"] = 1, 2
	strs := renderWith(t, snap, metric, ny, "sickness", "medication")
	for _, want := range []string{"2026-11-01 01:30 -04:00", "2026-11-01 01:30 -05:00"} {
		if n := countOf(strs, want); n != 2 {
			t.Errorf("%q appears %d times, want once for the symptom and once for the dose", want, n)
		}
	}
	text := joined(strs)
	wantIn(t, "dst", text, "Zentrolex dose text As needed 2026-11-01 00:30 Active", "2026-11-01 01:30 -04:00 Cough", "2026-11-01 01:30 -05:00 Rash",
		"2026-11-01 01:30 -04:00 Zentrolex first", "2026-11-01 01:30 -05:00 Zentrolex second")
}
