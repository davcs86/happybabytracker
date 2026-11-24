package export

import (
	"bytes"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/store"
	"github.com/davcs86/happybabytracker/api/internal/vaccines"
)

func ptr(s string) *string { return &s }

// administered builds an administration and its dose rows, one per antigen.
func administered(id, on string, antigens []string, lot, product, note *string) (model.Administration, []model.VaccineDose) {
	a := model.Administration{ID: id, GivenOn: on, Antigens: antigens, Lot: lot, Product: product, Note: note}
	var doses []model.VaccineDose
	for _, ag := range antigens {
		doses = append(doses, model.VaccineDose{ID: id + "-" + ag, Administration: id, Antigen: ag, GivenOn: on, Lot: lot, Product: product, Note: note})
	}
	return a, doses
}

// vaccineSnap is a kid born 2026-03-04 with the given administrations, newest first as the store returns them.
func vaccineSnap(schema *string, dismissals map[string]string, admins ...func() (model.Administration, []model.VaccineDose)) store.ExportSnapshot {
	snap := store.ExportSnapshot{Kid: model.Kid{Name: "Ada", BirthDate: "2026-03-04", Sex: "female", TimeZone: "UTC", GrowthSource: "who", VaccineSchema: schema},
		Dismissals: dismissals}
	for _, f := range admins {
		a, d := f()
		snap.Administrations = append([]model.Administration{a}, snap.Administrations...)
		snap.Doses = append(snap.Doses, d...)
	}
	return snap
}

func renderSnap(t *testing.T, snap store.ExportSnapshot, to string, modules ...model.Module) []string {
	t.Helper()
	var buf bytes.Buffer
	req := Request{From: "2026-11-01", To: to, Modules: modules, Units: metric, Now: renderNow, Loc: time.UTC}
	if err := Render(&buf, snap, req); err != nil {
		t.Fatal(err)
	}
	return pdfText(t, buf.Bytes())
}

func TestVaccinationSection(t *testing.T) {
	us, ok := vaccines.GetSchema("us")
	if !ok {
		t.Fatal("no us schedule")
	}
	hepB := func() (model.Administration, []model.VaccineDose) {
		return administered("a1", "2026-03-04", []string{"hepatitis_b"}, ptr("LOT-77"), ptr("Engerix-B"), ptr("left thigh"))
	}
	travel := func() (model.Administration, []model.VaccineDose) {
		return administered("a2", "2026-10-02", []string{"other"}, nil, ptr("Travel jab"), nil)
	}
	snap := vaccineSnap(ptr("us"), map[string]string{"rotavirus-1": "not_needed"}, hepB, travel)
	strs := renderSnap(t, snap, "2026-11-20", "vaccination")
	text := joined(strs)

	wantIn(t, "schedule", text, "Vaccination", "Schedule: "+us.Authority+", "+us.Edition, us.Attribution,
		"Birth", "Hepatitis B, dose 1", "Given 2026-03-04",
		"1-month vaccines", "Hepatitis B, dose 2", "Due from 2026-04-04 · overdue from 2026-06-04",
		"Rotavirus, dose 1", "Dismissed: not needed",
		"Also available", "Respiratory syncytial virus (RSV), dose 2", "Available from 2026-11-04",
		"2026-03-04", "Hepatitis B", "Engerix-B", "LOT-77", "left thigh",
		"Other recorded vaccinations", "2026-10-02", "Other (product name required)", "Travel jab")
	for _, w := range []string{"Given", "Due", "Upcoming", "Overdue", "Dismissed", "Available"} {
		if !slices.Contains(strs, w) {
			t.Errorf("status word %q missing", w)
		}
	}
	i := slices.Index(strs, "Hepatitis B, dose 2")
	if i < 0 || !slices.Contains(strs[i:i+3], "Overdue") {
		t.Errorf("the overdue line is not labelled %q: %q", "Overdue", strs[max(0, i):min(len(strs), i+4)])
	}
	wantNotIn(t, "schedule", text, "No schedule chosen", "omitted", us.Omitted[0])

	none := vaccineSnap(nil, map[string]string{}, hepB, travel)
	text = joined(renderSnap(t, none, "2026-11-20", "vaccination"))
	wantIn(t, "no schedule", text, "Vaccination", noSchedule, "2026-03-04", "Hepatitis B", "Engerix-B", "LOT-77", "left thigh",
		"2026-10-02", "Other (product name required)", "Travel jab")
	wantNotIn(t, "no schedule", text, "Schedule: ", "Overdue", "Given 2026-03-04", us.Attribution)

	empty := joined(renderSnap(t, vaccineSnap(nil, map[string]string{}), "2026-11-20", "vaccination"))
	wantIn(t, "empty", empty, "Vaccination "+noSchedule+" "+noVaccinations)
}

func TestRecordSectionsUpToTo(t *testing.T) {
	early := func() (model.Administration, []model.VaccineDose) {
		return administered("a1", "2026-03-04", []string{"hepatitis_b"}, nil, ptr("Early product"), nil)
	}
	late := func() (model.Administration, []model.VaccineDose) {
		return administered("a2", "2026-11-15", []string{"hepatitis_b", "other"}, ptr("LOT-LATE"), ptr("Later product"), ptr("after the range"))
	}
	snap := vaccineSnap(ptr("us"), map[string]string{}, early, late)
	snap.Milestones = []model.Milestone{
		{Entry: model.Entry{ID: "m2"}, Text: ptr("Claps after the range"), ReachedOn: "2026-11-15"},
		{Entry: model.Entry{ID: "m1"}, Text: ptr("Waves before the range"), ReachedOn: "2026-11-01"},
	}
	text := joined(renderSnap(t, snap, "2026-11-10", "milestones", "vaccination"))
	wantIn(t, "up to to", text, "Waves before the range", "Early product", "Given 2026-03-04")
	wantNotIn(t, "up to to", text, "2026-11-15", "Claps after the range", "Later product", "LOT-LATE", "after the range")
	if strings.Contains(text, "Other (product name required)") {
		t.Error("a dose after to appears under Other")
	}
}

func renderAt(t *testing.T, snap store.ExportSnapshot, from, to string, now time.Time) string {
	t.Helper()
	var buf bytes.Buffer
	req := Request{From: from, To: to, Modules: []model.Module{"vaccination"}, Units: metric, Now: now, Loc: time.UTC}
	if err := Render(&buf, snap, req); err != nil {
		t.Fatal(err)
	}
	return joined(pdfText(t, buf.Bytes()))
}

func TestVaccinationStatusesAsOfTo(t *testing.T) {
	today := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	dose1 := func() (model.Administration, []model.VaccineDose) {
		return administered("a1", "2026-03-04", []string{"hepatitis_b"}, nil, nil, nil)
	}
	dose2 := func() (model.Administration, []model.VaccineDose) {
		return administered("a2", "2026-11-15", []string{"hepatitis_b"}, nil, ptr("Given after to"), nil)
	}
	snap := vaccineSnap(ptr("us"), map[string]string{}, dose1, dose2)

	// As of 2026-11-10 dose 2 (window 2026-04-04 to 2026-06-04) is not given yet: the line reads as it did on that day.
	text := renderAt(t, snap, "2026-11-01", "2026-11-10", today)
	wantIn(t, "as of to", text, "Statuses as of 2026-11-10.",
		"Hepatitis B, dose 2 Overdue Due from 2026-04-04 · overdue from 2026-06-04", "Hepatitis B, dose 1 Given Given 2026-03-04")
	wantNotIn(t, "as of to", text, "2026-11-15", "Given after to", "Statuses as of 2026-11-24")

	// A line due at the end of the range prints Due, though it is overdue today.
	early := renderAt(t, vaccineSnap(ptr("us"), map[string]string{}), "2026-05-01", "2026-05-10", today)
	wantIn(t, "early range", early, "Statuses as of 2026-05-10.", "Diphtheria, dose 1 Due Due from 2026-05-04")
	wantNotIn(t, "early range", early, "Diphtheria, dose 1 Overdue")

	none := renderAt(t, vaccineSnap(nil, map[string]string{}, dose1), "2026-11-01", "2026-11-10", today)
	wantNotIn(t, "no schedule", none, "Statuses as of")
}

func TestVaccinationRangeEndingToday(t *testing.T) {
	us, _ := vaccines.GetSchema("us")
	snap := vaccineSnap(ptr("us"), map[string]string{"rotavirus-1": "not_needed"}, func() (model.Administration, []model.VaccineDose) {
		return administered("a1", "2026-03-04", []string{"hepatitis_b"}, nil, nil, nil)
	})
	text := renderAt(t, snap, "2026-11-01", "2026-11-20", renderNow)
	wantIn(t, "caption", text, "Schedule: "+us.Authority+", "+us.Edition+" Statuses as of 2026-11-20. "+us.Attribution)
	e, err := vaccines.Evaluate(us, vaccines.DosesFrom(snap.Doses), snap.Dismissals, snap.Kid.BirthDate, "2026-11-20", time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]string{}
	for _, a := range vaccines.Antigens() {
		names[a.ID] = a.Name
	}
	for _, ls := range append(e.Lines, e.Also...) {
		want := names[ls.Line.Antigen] + ", dose " + strconv.Itoa(ls.Line.Series) + " " + statusLabel[ls.Status]
		if !strings.Contains(text, want) {
			t.Errorf("%q missing: a range ending today prints today's statuses", want)
		}
	}
}
