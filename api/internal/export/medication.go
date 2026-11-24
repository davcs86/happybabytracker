package export

import (
	"errors"

	"github.com/davcs86/happybabytracker/api/internal/export/pdfdoc"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

// errUnknownCourse: a dose's course is not in the snapshot, which R1 rules out; as the API's 500.
var errUnknownCourse = errors.New("export: a dose's course is not in the snapshot")

// The form's words: CourseFields.svelte and the dose page.
const (
	noDoses   = "No doses in this range."
	active    = "Active"
	notActive = "Not active"
)

// Shares of the content width per medication column.
const (
	courseNameShare     = 0.2
	courseDoseShare     = 0.15
	courseScheduleShare = 0.2
	courseStartShare    = 0.17
	courseEndShare      = 0.13
	courseActiveShare   = 1 - courseNameShare - courseDoseShare - courseScheduleShare - courseStartShare - courseEndShare
	doseCourseShare     = 0.25
	doseAmountShare     = 0.2
	doseNoteShare       = 1 - timeShare - doseCourseShare - doseAmountShare
)

var (
	courseCols = []pdfdoc.Col{
		{Title: "Course", WidthMM: contentWidth * courseNameShare},
		{Title: "Dose", WidthMM: contentWidth * courseDoseShare},
		{Title: "Schedule", WidthMM: contentWidth * courseScheduleShare},
		{Title: "Start", WidthMM: contentWidth * courseStartShare},
		{Title: "Last day", WidthMM: contentWidth * courseEndShare},
		{Title: "Active", WidthMM: contentWidth * courseActiveShare},
	}
	givenCols = []pdfdoc.Col{
		{Title: "Given at", WidthMM: contentWidth * timeShare},
		{Title: "Course", WidthMM: contentWidth * doseCourseShare},
		{Title: "Amount", WidthMM: contentWidth * doseAmountShare},
		{Title: "Note", WidthMM: contentWidth * doseNoteShare},
	}
)

// medicationItems prints the courses (R1) as recorded, then the shown doses oldest first; nothing about reminders.
func medicationItems(snap store.ExportSnapshot, req Request, b *budget) ([]item, error) {
	out := []item{{heading: moduleTitle["medication"]}}
	courses := snap.Counts["medication"]
	if courses == 0 {
		return append(out, item{paragraph: noEntries}), nil
	}
	s := newSection(b)
	names := make(map[string]string, len(snap.Medications))
	for _, m := range snap.Medications {
		names[m.ID] = Text(m.Name)
	}
	n := min(s.take(courses), len(snap.Medications))
	rows := make([][]string, n)
	for i, m := range snap.Medications[:n] {
		end, state := "", notActive
		if m.EndOn != nil {
			end = FormatDate(*m.EndOn)
		}
		if m.Active {
			state = active
		}
		rows[i] = []string{Text(m.Name), Text(m.Dose), FormatSchedule(m.Schedule, m.StartAt, req.Loc), FormatInstant(m.StartAt, req.Loc), end, state}
	}
	out = append(out, s.shown(courseCols, rows, courses)...)
	doses := snap.Counts["medication_dose"]
	if doses == 0 {
		return append(out, item{paragraph: noDoses}), nil
	}
	n = min(s.take(doses), len(snap.MedicationDoses))
	rows = make([][]string, n)
	for i, d := range snap.MedicationDoses[:n] {
		name, ok := names[d.MedicationID]
		if !ok {
			return nil, errUnknownCourse
		}
		amount := ""
		if d.Amount != nil {
			amount = Text(*d.Amount)
		}
		rows[n-1-i] = []string{FormatInstant(d.GivenAt, req.Loc), name, amount, note(d)}
	}
	return append(out, s.shown(givenCols, rows, doses)...), nil
}
