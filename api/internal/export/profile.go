package export

import (
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/growth"
	"github.com/davcs86/happybabytracker/api/internal/kidtime"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/vaccines"
)

var errGrowthSource = errors.New("export: growth source unavailable")

var sexLabel = map[string]string{"female": "Female", "male": "Male"}

const noSchedule = "No schedule chosen"

// profileItems is the block printed first; the growth source and the schedule appear only when their module is requested.
func profileItems(kid model.Kid, req Request) ([]item, error) {
	today := req.Now.In(req.Loc).Format(time.DateOnly)
	due, corrected, err := kidtime.CorrectedBirthDate(kid.BirthDate, kid.GestationWeeks, kid.GestationDays, today, req.Loc)
	if err != nil {
		return nil, err
	}
	var dueDate *string
	if corrected {
		dueDate = &due
	}
	out := []item{
		{heading: "Profile"},
		{label: "Name", value: Text(kid.Name)},
		{label: "Birth date", value: FormatDate(kid.BirthDate)},
		{label: "Sex", value: sexLabel[kid.Sex]},
		{label: "Time zone", value: kid.TimeZone},
		{label: "Age today", value: FormatAgeHeader(FormatAge(kid.BirthDate, today), dueDate, today)},
	}
	if slices.Contains(req.Modules, "growth") {
		info, ok := growth.SourceInfo(growth.Source(kid.GrowthSource))
		if !ok {
			return nil, errGrowthSource
		}
		out = append(out, item{label: "Growth source", value: info.Name})
	}
	if slices.Contains(req.Modules, "vaccination") {
		caption := noSchedule
		if kid.VaccineSchema != nil {
			if s, ok := vaccines.GetSchema(*kid.VaccineSchema); ok {
				caption = s.Authority + ", " + s.Edition
			}
		}
		out = append(out, item{label: "Vaccination schedule", value: caption})
	}
	var titles []string
	for _, m := range model.Modules {
		if slices.Contains(req.Modules, m) {
			titles = append(titles, moduleTitle[m])
		}
	}
	return append(out,
		item{label: "Range", value: FormatDate(req.From) + " to " + FormatDate(req.To)},
		item{label: "Modules", value: strings.Join(titles, ", ")},
	), nil
}
