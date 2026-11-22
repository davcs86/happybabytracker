package vaccines

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/kidtime"
)

// Status is a line's state for one kid on one day.
type Status string

const (
	StatusGiven     Status = "given"
	StatusDue       Status = "due"
	StatusUpcoming  Status = "upcoming"
	StatusOverdue   Status = "overdue"
	StatusDismissed Status = "dismissed"
	// StatusAvailable is a supplementary line neither given nor dismissed; never due or overdue.
	StatusAvailable Status = ""
)

// Dose is one antigen of a recorded administration; GivenOn is a kid-local YYYY-MM-DD date.
type Dose struct{ ID, Administration, Antigen, GivenOn string }

// LineState is one schedule line evaluated for a kid; FromDate and UntilDate are kid-local, UntilDate exclusive.
type LineState struct {
	Line                                  Line
	Status                                Status
	FromDate, UntilDate, GivenOn          string
	DoseID, Administration, DismissReason string
}

// RecurringState is a recurring line with the doses given, oldest first.
type RecurringState struct {
	Line  Line
	Doses []Dose
}

// Evaluation is a kid's schedule: Lines can be due or overdue, Also are supplementary lines, Other the unmatched doses.
type Evaluation struct {
	Lines     []LineState
	Also      []LineState
	Recurring []RecurringState
	Other     []Dose
}

// DueGroup is the due or overdue lines sharing one from date.
type DueGroup struct {
	From     Age
	FromDate string
	Status   Status
	Count    int
}

var errEmptyWindow = errors.New("vaccines: empty window")

// Evaluate places each line applicable to birthDate on the calendar in loc and matches doses to it;
// birthDate and today are kid-local YYYY-MM-DD dates, dismissals map a line id to its reason.
func Evaluate(s Schema, doses []Dose, dismissals map[string]string, birthDate, today string, loc *time.Location) (Evaluation, error) {
	if _, err := kidtime.ParseDate(birthDate, loc); err != nil {
		return Evaluation{}, kidtime.ErrBadDate
	}
	if _, err := kidtime.ParseDate(today, loc); err != nil {
		return Evaluation{}, kidtime.ErrBadDate
	}
	sorted := slices.Clone(doses)
	slices.SortStableFunc(sorted, byDate)
	byAntigen := map[string][]Dose{}
	for _, d := range sorted {
		byAntigen[d.Antigen] = append(byAntigen[d.Antigen], d)
	}
	var antigens []string
	applicable := map[string][]Line{}
	for _, l := range s.Lines {
		if (l.BornFrom == "" || l.BornFrom <= birthDate) && (l.BornTo == "" || birthDate <= l.BornTo) {
			if applicable[l.Antigen] == nil {
				antigens = append(antigens, l.Antigen)
			}
			applicable[l.Antigen] = append(applicable[l.Antigen], l)
		}
	}
	var e Evaluation
	for _, d := range sorted {
		if len(applicable[d.Antigen]) == 0 {
			e.Other = append(e.Other, d)
		}
	}
	for _, antigen := range antigens {
		lines, given := applicable[antigen], byAntigen[antigen]
		if lines[0].Kind == KindRecurring {
			e.Recurring = append(e.Recurring, RecurringState{Line: lines[0], Doses: given})
			continue
		}
		slices.SortStableFunc(lines, func(a, b Line) int { return cmp.Compare(a.Series, b.Series) })
		for i, l := range lines {
			ls, err := state(l, birthDate, today, loc)
			if err != nil {
				return Evaluation{}, err
			}
			if i < len(given) {
				d := given[i]
				ls.Status, ls.GivenOn, ls.DoseID, ls.Administration = StatusGiven, d.GivenOn, d.ID, d.Administration
			} else if reason, ok := dismissals[l.ID]; ok {
				ls.Status, ls.DismissReason = StatusDismissed, reason
			}
			if l.Kind == KindSupplementary {
				e.Also = append(e.Also, ls)
			} else {
				e.Lines = append(e.Lines, ls)
			}
		}
		if len(given) > len(lines) {
			e.Other = append(e.Other, given[len(lines):]...)
		}
	}
	slices.SortStableFunc(e.Other, byDate)
	order := func(a, b LineState) int {
		return cmp.Or(cmp.Compare(a.FromDate, b.FromDate), cmp.Compare(a.Line.Group, b.Line.Group),
			cmp.Compare(a.Line.Antigen, b.Line.Antigen), cmp.Compare(a.Line.Series, b.Line.Series))
	}
	slices.SortStableFunc(e.Lines, order)
	slices.SortStableFunc(e.Also, order)
	return e, nil
}

func byDate(a, b Dose) int {
	return cmp.Or(cmp.Compare(a.GivenOn, b.GivenOn), cmp.Compare(a.ID, b.ID))
}

// state is a line's dates and its status before doses and dismissals.
func state(l Line, birthDate, today string, loc *time.Location) (LineState, error) {
	from, err := kidtime.AddAge(birthDate, l.From.N, byte(l.From.Unit), loc)
	if err != nil {
		return LineState{}, err
	}
	// No until: the default window runs VaccineDefaultWindowMonths from the from date, not from birth.
	until, err := kidtime.AddAge(from, constants.VaccineDefaultWindowMonths, byte(UnitMonth), loc)
	if l.Until != nil {
		until, err = kidtime.AddAge(birthDate, l.Until.N, byte(l.Until.Unit), loc)
	}
	if err != nil {
		return LineState{}, err
	}
	if until <= from {
		return LineState{}, fmt.Errorf("%w: line %s", errEmptyWindow, l.ID)
	}
	ls := LineState{Line: l, FromDate: from, UntilDate: until}
	switch {
	case l.Kind == KindSupplementary:
		ls.Status = StatusAvailable
	case today < from:
		ls.Status = StatusUpcoming
	case today >= until && l.Kind == KindRoutine:
		ls.Status = StatusOverdue
	default:
		ls.Status = StatusDue
	}
	return ls, nil
}

// Due groups the due and overdue lines by from date; a group is overdue if any of its lines is.
func (e Evaluation) Due() []DueGroup {
	var groups []DueGroup
	for _, l := range e.Lines {
		if l.Status != StatusDue && l.Status != StatusOverdue {
			continue
		}
		if len(groups) == 0 || groups[len(groups)-1].FromDate != l.FromDate {
			groups = append(groups, DueGroup{From: l.Line.From, FromDate: l.FromDate, Status: StatusDue})
		}
		g := &groups[len(groups)-1]
		g.Count++
		if l.Status == StatusOverdue {
			g.Status = StatusOverdue
		}
	}
	return groups
}
