// Package kidtime does calendar arithmetic in a kid's own time zone.
package kidtime

import (
	"errors"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

// Band is an age band by whole calendar months.
type Band int

const (
	Band0to11 Band = iota
	Band12to23
	Band24Plus
)

// DayBounds returns [start, end) of the local calendar day containing instant; end is the next local midnight.
func DayBounds(loc *time.Location, instant time.Time) (start, end time.Time) {
	y, m, d := instant.In(loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, loc), time.Date(y, m, d+1, 0, 0, 0, 0, loc)
}

// ParseDate reads "YYYY-MM-DD" as local midnight in loc.
func ParseDate(s string, loc *time.Location) (time.Time, error) {
	return time.ParseInLocation(time.DateOnly, s, loc)
}

// AgeMonths is the whole calendar months from birthDate to now's local date, never negative.
func AgeMonths(birthDate string, loc *time.Location, now time.Time) (int, error) {
	birth, err := parseDay(birthDate, loc)
	if err != nil {
		return 0, err
	}
	by, bm, bd := birth.Date()
	ny, nm, nd := now.In(loc).Date()
	months := (ny-by)*12 + int(nm) - int(bm)
	if nd < bd {
		months--
	}
	return max(months, 0), nil
}

// BandFor maps whole months to a band.
func BandFor(months int) Band {
	switch {
	case months >= constants.BandThirdStartMonths:
		return Band24Plus
	case months >= constants.BandSecondStartMonths:
		return Band12to23
	}
	return Band0to11
}

// BandStart is the local midnight the kid enters b: the birth date for Band0to11, else the birth date plus the band's months.
func BandStart(birthDate string, loc *time.Location, b Band) (time.Time, error) {
	birth, err := parseDay(birthDate, loc)
	if err != nil {
		return time.Time{}, err
	}
	months := 0
	switch b {
	case Band12to23:
		months = constants.BandSecondStartMonths
	case Band24Plus:
		months = constants.BandThirdStartMonths
	}
	y, m, d := birth.Date()
	return time.Date(y, m+time.Month(months), d, 0, 0, 0, 0, loc), nil
}

// ErrBadDate reports a date that is not "YYYY-MM-DD"; it never carries the input.
var ErrBadDate = errors.New("kidtime: invalid date")

// AgeBasis is a kid's age in days on a date: as lived, and as used to score growth (corrected for prematurity).
type AgeBasis struct {
	ChronologicalDays, ScoringDays int
	Corrected                      bool
}

func parseDay(s string, loc *time.Location) (time.Time, error) {
	t, err := ParseDate(s, loc)
	if err != nil {
		return time.Time{}, ErrBadDate
	}
	return t, nil
}

// civilDays counts calendar days from the Unix epoch to t's local date; UTC midnights are never 23 or 25 hours apart.
func civilDays(t time.Time) int {
	y, m, d := t.Date()
	return int(time.Date(y, m, d, 0, 0, 0, 0, time.UTC).Unix() / 86400)
}

// AgeDays is the whole calendar days from birthDate to on in loc; negative when on is before the birth date.
func AgeDays(birthDate, on string, loc *time.Location) (int, error) {
	birth, err := parseDay(birthDate, loc)
	if err != nil {
		return 0, err
	}
	day, err := parseDay(on, loc)
	if err != nil {
		return 0, err
	}
	return civilDays(day) - civilDays(birth), nil
}

// CorrectionDays is how many days early a preterm kid was born; 0 when either part is nil or gestation reached PretermBelowDays.
func CorrectionDays(gestWeeks, gestDays *int) int {
	if gestWeeks == nil || gestDays == nil {
		return 0
	}
	gestation := *gestWeeks*7 + *gestDays
	if gestation >= constants.PretermBelowDays {
		return 0
	}
	return constants.TermGestationDays - gestation
}

// correctionOn is the correction in force on the date on: none from the second birthday (birth date plus CorrectionEndsYears).
func correctionOn(birth, day time.Time, gestWeeks, gestDays *int) int {
	by, bm, bd := birth.Date()
	if !day.Before(time.Date(by+constants.CorrectionEndsYears, bm, bd, 0, 0, 0, 0, birth.Location())) {
		return 0
	}
	return CorrectionDays(gestWeeks, gestDays)
}

// AgeBasisOn is the age on a date; ScoringDays may be negative before the expected term and is returned as is.
func AgeBasisOn(birthDate string, gestWeeks, gestDays *int, on string, loc *time.Location) (AgeBasis, error) {
	birth, err := parseDay(birthDate, loc)
	if err != nil {
		return AgeBasis{}, err
	}
	day, err := parseDay(on, loc)
	if err != nil {
		return AgeBasis{}, err
	}
	chrono := civilDays(day) - civilDays(birth)
	corr := correctionOn(birth, day, gestWeeks, gestDays)
	return AgeBasis{ChronologicalDays: chrono, ScoringDays: chrono - corr, Corrected: corr > 0}, nil
}

// CorrectedBirthDate is the due date, the birth date moved later by the correction; "" and false when none applies on on.
func CorrectedBirthDate(birthDate string, gestWeeks, gestDays *int, on string, loc *time.Location) (string, bool, error) {
	birth, err := parseDay(birthDate, loc)
	if err != nil {
		return "", false, err
	}
	day, err := parseDay(on, loc)
	if err != nil {
		return "", false, err
	}
	corr := correctionOn(birth, day, gestWeeks, gestDays)
	if corr == 0 {
		return "", false, nil
	}
	y, m, d := birth.Date()
	return time.Date(y, m, d+corr, 0, 0, 0, 0, loc).Format(time.DateOnly), true, nil
}
