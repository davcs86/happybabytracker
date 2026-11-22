package vaccines

import (
	"strconv"
	"testing"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

func TestParseAge(t *testing.T) {
	ok := map[string]Age{
		"8w":  {8, UnitWeek},
		"12m": {12, UnitMonth},
		"3y":  {3, UnitYear},
		"0d":  {0, UnitDay},
		"5d":  {5, UnitDay},
	}
	for s, want := range ok {
		got, err := ParseAge(s)
		if err != nil || got != want {
			t.Errorf("ParseAge(%q) = %v, %v; want %v", s, got, err, want)
		}
	}
	for _, s := range []string{"", "0m", "0w", "0y", "-1m", "+1m", "8", "8x", "m", "1.5m", "12 m", " 12m", "12m ", "012m", "00d", "１2m", "1e2d", "12M"} {
		if _, err := ParseAge(s); err == nil {
			t.Errorf("ParseAge(%q) succeeded, want an error", s)
		}
	}
}

func TestParseAgeCaps(t *testing.T) {
	caps := map[Unit]int{
		UnitDay:   constants.VaccineMaxAgeDays,
		UnitWeek:  constants.VaccineMaxAgeWeeks,
		UnitMonth: constants.VaccineMaxAgeMonths,
		UnitYear:  constants.VaccineMaxAgeYears,
	}
	for u, max := range caps {
		at := strconv.Itoa(max) + string(u)
		if _, err := ParseAge(at); err != nil {
			t.Errorf("ParseAge(%q) = %v, want ok at the cap", at, err)
		}
		above := strconv.Itoa(max+1) + string(u)
		if _, err := ParseAge(above); err == nil {
			t.Errorf("ParseAge(%q) succeeded above the cap", above)
		}
	}
	if _, err := ParseAge("99999999999999999999999d"); err == nil {
		t.Error("ParseAge accepted an overflowing amount")
	}
}

func TestAgeCapsAreAbout130Years(t *testing.T) {
	for _, a := range []Age{
		{constants.VaccineMaxAgeDays, UnitDay},
		{constants.VaccineMaxAgeWeeks, UnitWeek},
		{constants.VaccineMaxAgeMonths, UnitMonth},
		{constants.VaccineMaxAgeYears, UnitYear},
	} {
		years := a.approxDays() / (12 * constants.DaysPerMonth)
		if years > 130 || years < 129.9 {
			t.Errorf("cap %v is %.2f years, want just under or at 130", a, years)
		}
	}
}
