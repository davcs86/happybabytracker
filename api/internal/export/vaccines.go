package export

import (
	"strconv"
	"strings"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/export/pdfdoc"
	"github.com/davcs86/happybabytracker/api/internal/kidtime"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/store"
	"github.com/davcs86/happybabytracker/api/internal/vaccines"
)

// The screen's words: vaccines/+page.svelte, VaccineGroup.svelte, ui/src/lib/vaccines.ts and VACCINE_STATUS_LABEL.
const (
	schedulePrefix = "Schedule: "
	alsoAvailable  = "Also available"
	otherTitle     = "Other recorded vaccinations"
	noneRecorded   = "None"
	givenPrefix    = "Given "
	birthGroup     = "Birth"
	noVaccinations = "No vaccinations recorded."
)

var (
	statusLabel = map[vaccines.Status]string{
		vaccines.StatusGiven: "Given", vaccines.StatusDue: "Due", vaccines.StatusUpcoming: "Upcoming",
		vaccines.StatusOverdue: "Overdue", vaccines.StatusDismissed: "Dismissed", vaccines.StatusAvailable: "Available",
	}
	dismissedText = map[string]string{"not_needed": "Dismissed: not needed", "given_unknown": "Dismissed: given, date unknown"}
	unitOne       = map[vaccines.Unit]string{vaccines.UnitDay: "day", vaccines.UnitWeek: "week", vaccines.UnitMonth: "month", vaccines.UnitYear: "year"}
)

// Shares of the content width per vaccination column.
const (
	lineGroupShare   = 0.2
	lineVaccineShare = 0.3
	lineStatusShare  = 0.12
	doseDateShare    = 0.14
	doseNamesShare   = 0.3
	doseProductShare = 0.2
	doseLotShare     = 0.12
	otherNameShare   = 0.5
)

var (
	lineCols = []pdfdoc.Col{
		{Title: "Group", WidthMM: contentWidth * lineGroupShare},
		{Title: "Vaccine", WidthMM: contentWidth * lineVaccineShare},
		{Title: "Status", WidthMM: contentWidth * lineStatusShare},
		{Title: "Detail", WidthMM: contentWidth * (1 - lineGroupShare - lineVaccineShare - lineStatusShare)},
	}
	doseCols = []pdfdoc.Col{
		{Title: "Date", WidthMM: contentWidth * doseDateShare},
		{Title: "Vaccines", WidthMM: contentWidth * doseNamesShare},
		{Title: "Product", WidthMM: contentWidth * doseProductShare},
		{Title: "Lot", WidthMM: contentWidth * doseLotShare},
		{Title: "Note", WidthMM: contentWidth * (1 - doseDateShare - doseNamesShare - doseProductShare - doseLotShare)},
	}
	otherCols = []pdfdoc.Col{
		{Title: "Date", WidthMM: contentWidth * doseDateShare},
		{Title: otherTitle, WidthMM: contentWidth * otherNameShare},
		{Title: "Product", WidthMM: contentWidth * (1 - doseDateShare - otherNameShare)},
	}
)

// groupTitle mirrors vaccines.ts: "Birth", or "<n>-<unit> vaccines".
func groupTitle(a vaccines.Age) string {
	if a.N == 0 {
		return birthGroup
	}
	return strconv.Itoa(a.N) + "-" + unitOne[a.Unit] + " vaccines"
}

// windowLabel mirrors vaccines.ts; recurring lines have no window.
func windowLabel(ls vaccines.LineState) string {
	switch ls.Line.Kind {
	case vaccines.KindRoutine:
		return "Due from " + ls.FromDate + " · overdue from " + ls.UntilDate
	case vaccines.KindMayNotBeNeeded:
		return "Due from " + ls.FromDate + " · no longer due from " + ls.UntilDate
	case vaccines.KindSupplementary:
		return "Available from " + ls.FromDate
	}
	return ""
}

func optional(s *string) string {
	if s == nil {
		return ""
	}
	return Text(*s)
}

// vaccinationItems evaluates the doses given up to req.To on the kid-local today, as the screen does on that day.
func vaccinationItems(snap store.ExportSnapshot, req Request, b *budget) ([]item, error) {
	names := map[string]string{}
	for _, a := range vaccines.Antigens() {
		names[a.ID] = a.Name
	}
	name := func(id string) string {
		if n, ok := names[id]; ok {
			return n
		}
		return id
	}
	var doses []model.VaccineDose
	products := map[string]string{}
	for _, d := range snap.Doses {
		if d.GivenOn <= req.To {
			doses = append(doses, d)
			products[d.ID] = optional(d.Product)
		}
	}
	out := []item{{heading: moduleTitle["vaccination"]}}
	s := newSection(b)
	var other [][]string
	schema, chosen := vaccines.Schema{}, false
	if v := snap.Kid.VaccineSchema; v != nil {
		schema, chosen = vaccines.GetSchema(*v)
	}
	if chosen {
		start, _ := kidtime.DayBounds(req.Loc, req.Now)
		e, err := vaccines.Evaluate(schema, vaccines.DosesFrom(doses), snap.Dismissals, snap.Kid.BirthDate, start.Format(time.DateOnly), req.Loc)
		if err != nil {
			return nil, err
		}
		out = append(out, item{body: schedulePrefix + schema.Authority + ", " + schema.Edition}, item{paragraph: schema.Attribution})
		var lines [][]string
		for _, ls := range e.Lines {
			detail := ""
			switch {
			case ls.Status == vaccines.StatusGiven:
				detail = givenPrefix + ls.GivenOn
			case ls.Status == vaccines.StatusDismissed:
				detail = dismissedText[ls.DismissReason]
			case ls.Status != vaccines.StatusAvailable:
				detail = windowLabel(ls)
			}
			lines = append(lines, lineRow(groupTitle(ls.Line.From), ls, name, detail))
		}
		for _, ls := range e.Also {
			detail := windowLabel(ls)
			if ls.Status == vaccines.StatusGiven {
				detail = givenPrefix + ls.GivenOn
			}
			lines = append(lines, lineRow(alsoAvailable, ls, name, detail))
		}
		if len(lines) > 0 {
			out = append(out, s.table(lineCols, lines)...)
		}
		for _, d := range e.Other {
			other = append(other, []string{FormatDate(d.GivenOn), name(d.Antigen), products[d.ID]})
		}
	} else {
		out = append(out, item{body: noSchedule})
	}
	var given [][]string
	for _, a := range snap.Administrations {
		if a.GivenOn > req.To {
			continue
		}
		antigens := make([]string, len(a.Antigens))
		for i, id := range a.Antigens {
			antigens[i] = name(id)
		}
		given = append(given, []string{FormatDate(a.GivenOn), strings.Join(antigens, ", "), optional(a.Product), optional(a.Lot), optional(a.Note)})
	}
	if len(given) == 0 {
		out = append(out, item{paragraph: noVaccinations})
	} else {
		out = append(out, s.table(doseCols, given)...)
	}
	switch {
	case !chosen:
	case len(other) == 0:
		out = append(out, item{label: otherTitle, value: noneRecorded})
	default:
		out = append(out, s.table(otherCols, other)...)
	}
	return out, nil
}

func lineRow(group string, ls vaccines.LineState, name func(string) string, detail string) []string {
	return []string{group, name(ls.Line.Antigen) + ", dose " + strconv.Itoa(ls.Line.Series), statusLabel[ls.Status], detail}
}
