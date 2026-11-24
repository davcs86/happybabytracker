package milestones

import (
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/kidtime"
)

// Position is where a checklist age sits against the kid's age basis.
type Position string

const (
	PositionEarlier Position = "earlier"
	PositionCurrent Position = "current"
	PositionNext    Position = "next"
	PositionLater   Position = "later"
)

// AgeBasis is whole months; CorrectedMonths is nil unless prematurity correction applies, and BasisMonths is the one to compare with list ages.
type AgeBasis struct {
	ChronologicalMonths int
	CorrectedMonths     *int
	BasisMonths         int
}

// AgeBasisFor reads the age on now's date in loc; the correction is the one kidtime applies (it ends on the second birthday).
func AgeBasisFor(birthDate string, gestWeeks, gestDays *int, loc *time.Location, now time.Time) (AgeBasis, error) {
	chrono, err := kidtime.AgeMonths(birthDate, loc, now)
	if err != nil {
		return AgeBasis{}, err
	}
	today := now.In(loc).Format(time.DateOnly)
	due, corrected, err := kidtime.CorrectedBirthDate(birthDate, gestWeeks, gestDays, today, loc)
	if err != nil {
		return AgeBasis{}, err
	}
	if !corrected {
		return AgeBasis{ChronologicalMonths: chrono, BasisMonths: chrono}, nil
	}
	months, err := kidtime.AgeMonths(due, loc, now)
	if err != nil {
		return AgeBasis{}, err
	}
	return AgeBasis{ChronologicalMonths: chrono, CorrectedMonths: &months, BasisMonths: months}, nil
}

// Positions places every age against basisMonths: current is the largest age at or below it, next the smallest above it.
// Callers range over ages, never the map.
func Positions(ages []int, basisMonths int) map[int]Position {
	current, next := -1, -1
	for _, a := range ages {
		if a <= basisMonths && a > current {
			current = a
		}
		if a > basisMonths && (next < 0 || a < next) {
			next = a
		}
	}
	out := make(map[int]Position, len(ages))
	for _, a := range ages {
		switch {
		case a == current:
			out[a] = PositionCurrent
		case a == next:
			out[a] = PositionNext
		case a < basisMonths:
			out[a] = PositionEarlier
		default:
			out[a] = PositionLater
		}
	}
	return out
}

// CatalogShown is true while today is before the 5th birthday, by chronological age.
func CatalogShown(birthDate, today string) (bool, error) {
	until, err := kidtime.AddAge(birthDate, constants.MilestoneCatalogUntilYears, 'y', time.UTC)
	if err != nil {
		return false, err
	}
	if _, err := kidtime.ParseDate(today, time.UTC); err != nil {
		return false, kidtime.ErrBadDate
	}
	return today < until, nil
}
