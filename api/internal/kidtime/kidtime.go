// Package kidtime does calendar arithmetic in a kid's own time zone.
package kidtime

import (
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
	birth, err := ParseDate(birthDate, loc)
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
	birth, err := ParseDate(birthDate, loc)
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
