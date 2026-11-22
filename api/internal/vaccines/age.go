package vaccines

import (
	"errors"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

// Unit is the unit of a schedule age.
type Unit byte

const (
	UnitDay   Unit = 'd'
	UnitWeek  Unit = 'w'
	UnitMonth Unit = 'm'
	UnitYear  Unit = 'y'
)

// Age is a schedule age as the source states it; it becomes a date only by calendar arithmetic.
type Age struct {
	N    int
	Unit Unit
}

// ErrBadAge: not <n><unit> with ASCII digits, no leading zero, 0 only as 0d, and at most the unit's cap.
var ErrBadAge = errors.New("vaccines: not an age")

var ageCaps = map[Unit]int{
	UnitDay:   constants.VaccineMaxAgeDays,
	UnitWeek:  constants.VaccineMaxAgeWeeks,
	UnitMonth: constants.VaccineMaxAgeMonths,
	UnitYear:  constants.VaccineMaxAgeYears,
}

// ParseAge reads "8w", "12m", "4y" or "0d".
func ParseAge(s string) (Age, error) {
	if len(s) < 2 {
		return Age{}, ErrBadAge
	}
	u := Unit(s[len(s)-1])
	limit, ok := ageCaps[u]
	digits := s[:len(s)-1]
	if !ok || digits[0] == '0' && (len(digits) > 1 || u != UnitDay) {
		return Age{}, ErrBadAge
	}
	n := 0
	for i := 0; i < len(digits); i++ {
		c := digits[i]
		if c < '0' || c > '9' {
			return Age{}, ErrBadAge
		}
		n = n*10 + int(c-'0')
		if n > limit {
			return Age{}, ErrBadAge
		}
	}
	return Age{N: n, Unit: u}, nil
}

// approxDays orders ages of different units; never use it to place a date.
func (a Age) approxDays() float64 {
	switch a.Unit {
	case UnitWeek:
		return float64(a.N) * 7
	case UnitMonth:
		return float64(a.N) * constants.DaysPerMonth
	case UnitYear:
		return float64(a.N) * 12 * constants.DaysPerMonth
	}
	return float64(a.N)
}
