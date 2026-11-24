package export

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/schedule"
	"github.com/davcs86/happybabytracker/api/internal/units"
)

// Radices of the stored columns, as ui/src/lib/constants.ts spells them.
const (
	gramsPerKG       = 1000
	mmPerCM          = 10
	hundredths       = 100
	ozPerLB          = 16
	hundredthsPerTen = 10
	monthsPerYear    = 12
	secondsPerMinute = 60
	minutesPerHour   = 60
	secondsPerHour   = 3600
	secondsPerDay    = 86400
	dayLayout        = "2006-01-02"
	dayHeadingLayout = "Mon 2 Jan 2006"
	instantLayout    = "2006-01-02 15:04"
	clockLayout      = "15:04"
	beforeTerm       = "before expected term"
)

// decimal is n / scale (a power of ten) without trailing zeros and without float maths.
func decimal(n int64, scale uint64) string {
	m := uint64(n)
	sign := ""
	if n < 0 {
		m, sign = -m, "-"
	}
	frac := strconv.FormatUint(m%scale, 10)
	width := len(strconv.FormatUint(scale, 10)) - 1
	frac = strings.TrimRight(strings.Repeat("0", width-len(frac))+frac, "0")
	if frac == "" {
		return sign + strconv.FormatUint(m/scale, 10)
	}
	return sign + strconv.FormatUint(m/scale, 10) + "." + frac
}

// FormatWeight prints the stored column of the parent's unit: kg from grams, lb and oz from hundredths of an ounce.
func FormatWeight(w units.Weight, u model.Units) string {
	if u.Weight == string(units.UnitKG) {
		return decimal(w.Grams, gramsPerKG) + " kg"
	}
	m := uint64(w.OzHundredths)
	sign := ""
	if w.OzHundredths < 0 {
		m, sign = -m, "-"
	}
	tenths := m / hundredthsPerTen
	if m%hundredthsPerTen*2 >= hundredthsPerTen {
		tenths++
	}
	const tenthsPerLB = ozPerLB * hundredths / hundredthsPerTen
	return fmt.Sprintf("%s%d lb %s oz", sign, tenths/tenthsPerLB, decimal(int64(tenths%tenthsPerLB), hundredthsPerTen))
}

func FormatLength(l units.Length, u model.Units) string {
	if u.Length == string(units.UnitCM) {
		return decimal(l.MM, mmPerCM) + " cm"
	}
	return decimal(l.InHundredths, hundredths) + " in"
}

func FormatVolume(v units.Volume, u model.Units) string {
	if u.Volume == string(units.UnitML) {
		return strconv.FormatInt(v.ML, 10) + " ml"
	}
	return decimal(v.FlOzHundredths, hundredths) + " fl oz"
}

func FormatMilk(m units.Milk, u model.Units) string {
	gain := decimal(m.GainOzHundredths, hundredths) + " oz"
	if u.Weight == string(units.UnitKG) {
		gain = strconv.FormatInt(m.GainGrams, 10) + " g"
	}
	return "weight gain " + gain + ", about " + FormatVolume(units.Volume{ML: m.ML, FlOzHundredths: m.FlOzHundredths}, u)
}

// FormatTemperature prints the stored column of the parent's unit with its symbol, as temperatureText and TEMP_UNIT_LABEL do.
func FormatTemperature(t units.Temperature, u model.Units) string {
	if u.Temp == string(units.UnitC) {
		return decimal(t.CHundredths, hundredths) + " °C"
	}
	return decimal(t.FHundredths, hundredths) + " °F"
}

// FormatSchedule words s as scheduleLabel does; an every-N-hours grid names start's clock in loc.
func FormatSchedule(s schedule.Schedule, start time.Time, loc *time.Location) string {
	if loc == nil {
		loc = time.UTC
	}
	switch s.Kind {
	case schedule.KindNone:
		return "As needed"
	case schedule.KindTimes:
		return strings.Join(s.Times, ", ")
	case schedule.KindEvery:
		every := strconv.Itoa(s.EveryHours) + " hours"
		if s.EveryHours == 1 {
			every = "hour"
		}
		return "Every " + every + " from " + start.In(loc).Format(clockLayout)
	}
	return ""
}

func FormatDuration(seconds int) string {
	total := max(0, seconds)
	if total < secondsPerMinute {
		return strconv.Itoa(total) + " s"
	}
	minutes := total / secondsPerMinute
	hours, rest := minutes/minutesPerHour, minutes%minutesPerHour
	switch {
	case hours == 0:
		return strconv.Itoa(rest) + " min"
	case rest == 0:
		return strconv.Itoa(hours) + " h"
	}
	return strconv.Itoa(hours) + " h " + strconv.Itoa(rest) + " min"
}

func plural(n int, unit string) string {
	if n == 1 {
		return "1 " + unit
	}
	return strconv.Itoa(n) + " " + unit + "s"
}

// yearsAndMonths is the tail both age formats share: from 12 months on, days are not shown.
func yearsAndMonths(months int) string {
	years, rest := months/monthsPerYear, months%monthsPerYear
	if rest == 0 {
		return plural(years, "year")
	}
	return plural(years, "year") + " " + plural(rest, "month")
}

type ymd struct{ y, m, d int }

func parseDay(s string) (ymd, bool) {
	t, err := time.Parse(dayLayout, s)
	if err != nil {
		return ymd{}, false
	}
	return ymd{t.Year(), int(t.Month()), t.Day()}, true
}

func (v ymd) dayNumber() int64 {
	return time.Date(v.y, time.Month(v.m), v.d, 0, 0, 0, 0, time.UTC).Unix() / secondsPerDay
}

// addMonths clamps the day to the target month's last day: 29 Feb + 12 months is 28 Feb.
func addMonths(from ymd, months int) ymd {
	index := from.y*monthsPerYear + from.m - 1 + months
	y, m := index/monthsPerYear, index%monthsPerYear+1
	last := time.Date(y, time.Month(m)+1, 0, 0, 0, 0, 0, time.UTC).Day()
	return ymd{y, m, min(from.d, last)}
}

// FormatAge reads both dates as YYYY-MM-DD in the kid's zone; "" when either is not a date.
func FormatAge(birthDate, today string) string {
	birth, okB := parseDay(birthDate)
	t, okT := parseDay(today)
	if !okB || !okT {
		return ""
	}
	end := t.dayNumber()
	if end <= birth.dayNumber() {
		return plural(0, "day")
	}
	months := (t.y-birth.y)*monthsPerYear + t.m - birth.m
	if addMonths(birth, months).dayNumber() > end {
		months--
	}
	days := int(end - addMonths(birth, months).dayNumber())
	switch {
	case months == 0:
		return plural(days, "day")
	case months >= monthsPerYear:
		return yearsAndMonths(months)
	case days == 0:
		return plural(months, "month")
	}
	return plural(months, "month") + " " + plural(days, "day")
}

// formatAgeDays counts months of constants.DaysPerMonth days (the curves' axis), so it may differ from FormatAge by a day.
func formatAgeDays(days int) string {
	total := max(0, days)
	months := int(math.Floor(float64(total) / constants.DaysPerMonth))
	if months == 0 {
		return plural(total, "day")
	}
	if months >= monthsPerYear {
		return yearsAndMonths(months)
	}
	rest := int(math.Floor(float64(total) - float64(months)*constants.DaysPerMonth))
	if rest == 0 {
		return plural(months, "month")
	}
	return plural(months, "month") + " " + plural(rest, "day")
}

// FormatAgeDays shows both ages when a correction applies; below zero there is no corrected age yet.
func FormatAgeDays(chronoDays int, correctedDays *int) string {
	lived := formatAgeDays(chronoDays)
	switch {
	case correctedDays == nil:
		return lived
	case *correctedDays < 0:
		return lived + " (" + beforeTerm + ")"
	}
	return lived + " (corrected " + formatAgeDays(*correctedDays) + ")"
}

// FormatAgeHeader takes the already formatted age; a due date still ahead (ISO dates compare as strings) has no corrected age yet.
func FormatAgeHeader(age string, correctedBirthDate *string, today string) string {
	switch {
	case correctedBirthDate == nil:
		return age
	case *correctedBirthDate > today:
		return age + " (" + beforeTerm + ")"
	}
	return age + " (corrected " + FormatAge(*correctedBirthDate, today) + ")"
}

// fixedHalfUp mirrors growth.ts toFixedHalfUp: Math.round rounds halves up, and -0 prints as 0.
func fixedHalfUp(x float64, places int) string {
	p := math.Pow(10, float64(places))
	r := math.Floor(x*p+0.5) / p
	if r == 0 {
		r = 0
	}
	return strconv.FormatFloat(r, 'f', places, 64)
}

// FormatPercentile mirrors growth.ts formatPercentile: beyond the shown range a bound, never 0 or 100.
func FormatPercentile(p float64) string {
	switch {
	case p < constants.PercentileShownMin:
		return "<" + strconv.FormatFloat(constants.PercentileShownMin, 'f', -1, 64)
	case p > constants.PercentileShownMax:
		return ">" + strconv.FormatFloat(constants.PercentileShownMax, 'f', -1, 64)
	}
	return fixedHalfUp(p, constants.PercentileDecimals)
}

func FormatZ(z float64) string { return fixedHalfUp(z, constants.ZScoreDecimals) }

// FormatDayHeading mirrors timeline.ts formatDayHeading ("Sun 29 Mar 2026"); a malformed date prints as stored.
func FormatDayHeading(date string) string {
	t, err := time.Parse(dayLayout, date)
	if err != nil {
		return date
	}
	return t.Format(dayHeadingLayout)
}

// FormatDate is the stored YYYY-MM-DD, as the UI prints plain dates.
func FormatDate(date string) string { return date }

// repeated reports whether t's wall clock also names another instant in loc (the fall-back hour).
func repeated(t time.Time, loc *time.Location) bool {
	_, off := t.In(loc).Zone()
	for _, probe := range []time.Duration{-24 * time.Hour, 24 * time.Hour} {
		_, other := t.Add(probe).In(loc).Zone()
		if other == off {
			continue
		}
		u := t.Add(time.Duration(off-other) * time.Second)
		if _, got := u.In(loc).Zone(); got == other {
			return true
		}
	}
	return false
}

// FormatInstant is the wall clock of loc, always with the date; only a repeated local time carries its numeric offset.
func FormatInstant(t time.Time, loc *time.Location) string {
	if loc == nil {
		loc = time.UTC
	}
	local := t.In(loc)
	out := local.Format(instantLayout)
	if !repeated(t, loc) {
		return out
	}
	_, off := local.Zone()
	sign := '+'
	if off < 0 {
		sign, off = '-', -off
	}
	return fmt.Sprintf("%s %c%02d:%02d", out, sign, off/secondsPerHour, off%secondsPerHour/secondsPerMinute)
}

// Text maps each run of newlines and tabs to one space; other control runes and invalid bytes are left for the font layer.
func Text(s string) string {
	var b strings.Builder
	inRun := false
	for i := 0; i < len(s); i++ {
		if c := s[i]; c == '\r' || c == '\n' || c == '\t' {
			if !inRun {
				b.WriteByte(' ')
			}
			inRun = true
			continue
		}
		inRun = false
		b.WriteByte(s[i])
	}
	return b.String()
}
