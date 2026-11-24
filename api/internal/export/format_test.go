package export

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"
	_ "time/tzdata"

	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/schedule"
	"github.com/davcs86/happybabytracker/api/internal/units"
)

var (
	metric   = model.Units{Weight: "kg", Length: "cm", Volume: "ml", Temp: "c"}
	imperial = model.Units{Weight: "lb", Length: "in", Volume: "floz", Temp: "f"}
)

func TestFormatWeight(t *testing.T) {
	for _, c := range []struct {
		grams int64
		want  string
	}{{3200, "3.2 kg"}, {3000, "3 kg"}, {3215, "3.215 kg"}, {3205, "3.205 kg"}, {250000, "250 kg"}, {5, "0.005 kg"}} {
		if got := FormatWeight(units.Weight{Grams: c.grams}, metric); got != c.want {
			t.Errorf("%d g = %q, want %q", c.grams, got, c.want)
		}
	}
	for _, c := range []struct {
		oz   int64
		want string
	}{
		{11300, "7 lb 1 oz"}, {13150, "8 lb 3.5 oz"}, {12800, "8 lb 0 oz"}, {800, "0 lb 8 oz"},
		{13155, "8 lb 3.6 oz"}, {13154, "8 lb 3.5 oz"}, {12796, "8 lb 0 oz"},
	} {
		if got := FormatWeight(units.Weight{OzHundredths: c.oz}, imperial); got != c.want {
			t.Errorf("%d hundredths oz = %q, want %q", c.oz, got, c.want)
		}
	}
}

func TestFormatWeightTakesTheStoredColumnOfTheUnit(t *testing.T) {
	w := units.Weight{Grams: 3200, OzHundredths: 11300}
	if got := FormatWeight(w, metric); got != "3.2 kg" {
		t.Errorf("metric = %q", got)
	}
	if got := FormatWeight(w, imperial); got != "7 lb 1 oz" {
		t.Errorf("imperial = %q", got)
	}
}

func TestFormatLength(t *testing.T) {
	for _, c := range []struct {
		mm, in int64
		unit   model.Units
		want   string
	}{
		{495, 1949, metric, "49.5 cm"}, {500, 1969, metric, "50 cm"},
		{495, 1950, imperial, "19.5 in"}, {508, 2000, imperial, "20 in"}, {496, 1953, imperial, "19.53 in"},
		{7, 0, metric, "0.7 cm"},
	} {
		if got := FormatLength(units.Length{MM: c.mm, InHundredths: c.in}, c.unit); got != c.want {
			t.Errorf("%d mm / %d in-hundredths = %q, want %q", c.mm, c.in, got, c.want)
		}
	}
}

func TestFormatVolume(t *testing.T) {
	bottle := units.Volume{ML: 120, FlOzHundredths: 406}
	for _, c := range []struct {
		v     units.Volume
		units model.Units
		want  string
	}{
		{bottle, metric, "120 ml"},
		{bottle, imperial, "4.06 fl oz"},
		{units.Volume{ML: 133, FlOzHundredths: 450}, imperial, "4.5 fl oz"},
		{units.Volume{ML: 118, FlOzHundredths: 400}, imperial, "4 fl oz"},
		{units.Volume{ML: 1, FlOzHundredths: 3}, imperial, "0.03 fl oz"},
	} {
		if got := FormatVolume(c.v, c.units); got != c.want {
			t.Errorf("%+v = %q, want %q", c.v, got, c.want)
		}
	}
}

func TestFormatMilk(t *testing.T) {
	milk := units.Milk{GainGrams: 62, GainOzHundredths: 219, ML: 60, FlOzHundredths: 204}
	zero := units.Milk{}
	odd := units.Milk{GainGrams: 100, GainOzHundredths: 1, ML: 7, FlOzHundredths: 90}
	for _, c := range []struct {
		name  string
		milk  units.Milk
		units model.Units
		want  string
	}{
		{"kg and ml", milk, metric, "weight gain 62 g, about 60 ml"},
		{"lb and floz", milk, imperial, "weight gain 2.19 oz, about 2.04 fl oz"},
		{"kg and floz", milk, model.Units{Weight: "kg", Volume: "floz"}, "weight gain 62 g, about 2.04 fl oz"},
		{"lb and ml", milk, model.Units{Weight: "lb", Volume: "ml"}, "weight gain 2.19 oz, about 60 ml"},
		{"zero metric", zero, metric, "weight gain 0 g, about 0 ml"},
		{"zero imperial", zero, imperial, "weight gain 0 oz, about 0 fl oz"},
		{"odd imperial", odd, imperial, "weight gain 0.01 oz, about 0.9 fl oz"},
		{"odd metric", odd, metric, "weight gain 100 g, about 7 ml"},
	} {
		if got := FormatMilk(c.milk, c.units); got != c.want {
			t.Errorf("%s = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	for _, c := range []struct {
		s    int
		want string
	}{
		{0, "0 s"}, {59, "59 s"}, {60, "1 min"}, {1023, "17 min"}, {3600, "1 h"},
		{3660, "1 h 1 min"}, {26400, "7 h 20 min"}, {86400, "24 h"},
		{-5, "0 s"}, {math.MinInt, "0 s"},
	} {
		if got := FormatDuration(c.s); got != c.want {
			t.Errorf("%d s = %q, want %q", c.s, got, c.want)
		}
	}
	if got := FormatDuration(math.MaxInt); got == "" {
		t.Error("MaxInt gave an empty string")
	}
}

func TestFormatAge(t *testing.T) {
	for _, c := range []struct{ rule, birth, today, want string }{
		{"months and days", "2026-07-08", "2026-11-20", "4 months 12 days"},
		{"same day", "2026-11-20", "2026-11-20", "0 days"},
		{"singular day", "2026-11-19", "2026-11-20", "1 day"},
		{"plural days", "2026-10-21", "2026-11-20", "30 days"},
		{"singular month", "2026-10-20", "2026-11-20", "1 month"},
		{"singular day after months", "2026-09-19", "2026-11-20", "2 months 1 day"},
		{"years and months", "2024-08-20", "2026-11-20", "2 years 3 months"},
		{"singular year", "2025-11-20", "2026-11-20", "1 year"},
		{"years never show days", "2025-11-01", "2026-11-20", "1 year"},
		{"singular month after years", "2024-10-01", "2026-11-20", "2 years 1 month"},
		{"31 Jan + 1 month clamps to 28 Feb", "2026-01-31", "2026-02-28", "1 month"},
		{"31 Jan clamped, then 1 day", "2026-01-31", "2026-03-01", "1 month 1 day"},
		{"birth after today", "2026-11-21", "2026-11-20", "0 days"},
		{"unparseable birth", "x", "2026-11-20", ""},
		{"unparseable today", "2026-11-20", "", ""},
		{"not a calendar date", "2026-02-30", "2026-11-20", ""},
	} {
		if got := FormatAge(c.birth, c.today); got != c.want {
			t.Errorf("%s: %s on %s = %q, want %q", c.rule, c.birth, c.today, got, c.want)
		}
	}
}

func TestFormatAgeLeapDayClamp(t *testing.T) {
	for _, c := range []struct{ today, want string }{
		{"2025-02-28", "1 year"},
		{"2025-02-27", "11 months 29 days"},
		{"2025-03-01", "1 year"},
		{"2026-02-28", "2 years"},
		{"2025-03-29", "1 year 1 month"},
		{"2025-03-28", "1 year"},
		{"2028-02-29", "4 years"},
		{"2024-03-29", "1 month"},
		{"2024-03-28", "28 days"},
	} {
		if got := FormatAge("2024-02-29", c.today); got != c.want {
			t.Errorf("2024-02-29 on %s = %q, want %q", c.today, got, c.want)
		}
	}
}

func TestFormatAgeHeader(t *testing.T) {
	due := func(s string) *string { return &s }
	for _, c := range []struct {
		name      string
		age       string
		corrected *string
		today     string
		want      string
	}{
		{"no correction", "3 months", nil, "2026-11-22", "3 months"},
		{"due date ahead", "20 days", due("2026-12-31"), "2026-11-22", "20 days (before expected term)"},
		{"due date today", "70 days", due("2026-11-22"), "2026-11-22", "70 days (corrected 0 days)"},
		{"after the due date", "4 months", due("2026-10-01"), "2026-11-22", "4 months (corrected 1 month 21 days)"},
	} {
		if got := FormatAgeHeader(c.age, c.corrected, c.today); got != c.want {
			t.Errorf("%s = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestFormatAgeDays(t *testing.T) {
	for _, c := range []struct {
		days int
		want string
	}{
		{0, "0 days"}, {1, "1 day"}, {30, "30 days"}, {31, "1 month"}, {45, "1 month 14 days"},
		{61, "2 months"}, {365, "11 months 30 days"}, {366, "1 year"}, {400, "1 year 1 month"},
		{1000, "2 years 8 months"}, {-3, "0 days"},
	} {
		if got := FormatAgeDays(c.days, nil); got != c.want {
			t.Errorf("%d days = %q, want %q", c.days, got, c.want)
		}
	}
	corrected := func(n int) *int { return &n }
	for _, c := range []struct {
		name      string
		days      int
		corrected *int
		want      string
	}{
		{"no correction", 45, nil, "1 month 14 days"},
		{"both ages", 45, corrected(12), "1 month 14 days (corrected 12 days)"},
		{"below zero", 10, corrected(-23), "10 days (before expected term)"},
		{"zero is a real age", 33, corrected(0), "1 month 2 days (corrected 0 days)"},
	} {
		if got := FormatAgeDays(c.days, c.corrected); got != c.want {
			t.Errorf("%s = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestFormattersNeverPanic(t *testing.T) {
	for _, n := range []int{math.MinInt, -1, 0, 1, math.MaxInt} {
		FormatDuration(n)
		FormatAgeDays(n, nil)
		FormatAgeDays(n, &n)
	}
	for _, n := range []int64{math.MinInt64, -1, 0, 1, math.MaxInt64} {
		FormatWeight(units.Weight{Grams: n, OzHundredths: n}, metric)
		FormatWeight(units.Weight{Grams: n, OzHundredths: n}, imperial)
		FormatLength(units.Length{MM: n, InHundredths: n}, metric)
		FormatLength(units.Length{MM: n, InHundredths: n}, imperial)
		FormatVolume(units.Volume{ML: n, FlOzHundredths: n}, imperial)
		FormatMilk(units.Milk{GainGrams: n, GainOzHundredths: n, ML: n, FlOzHundredths: n}, imperial)
	}
	FormatInstant(time.Time{}, nil)
	FormatInstant(time.Unix(math.MaxInt64/2, 0), time.UTC)
	FormatAge("9999-12-31", "0001-01-01")
	FormatAge("0001-01-01", "9999-12-31")
	FormatDate("")
}

func mustZone(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func TestFormatInstant(t *testing.T) {
	ny := mustZone(t, "America/New_York")
	lh := mustZone(t, "Australia/Lord_Howe")
	kolkata := mustZone(t, "Asia/Kolkata")
	for _, c := range []struct {
		name string
		utc  string
		loc  *time.Location
		want string
	}{
		{"ordinary day, always with the date", "2026-03-04T19:05:59Z", ny, "2026-03-04 14:05"},
		{"UTC", "2026-03-04T14:05:00Z", time.UTC, "2026-03-04 14:05"},
		{"nil zone reads as UTC", "2026-03-04T14:05:00Z", nil, "2026-03-04 14:05"},
		{"a zone without DST", "2026-11-01T01:30:00Z", kolkata, "2026-11-01 07:00"},
		{"fall-back day before the repeated hour", "2026-11-01T04:30:00Z", ny, "2026-11-01 00:30"},
		{"01:30 EDT", "2026-11-01T05:30:00Z", ny, "2026-11-01 01:30 -04:00"},
		{"01:30 EST", "2026-11-01T06:30:00Z", ny, "2026-11-01 01:30 -05:00"},
		{"01:00 EDT starts the repeat", "2026-11-01T05:00:00Z", ny, "2026-11-01 01:00 -04:00"},
		{"01:00 EST at the transition second", "2026-11-01T06:00:00Z", ny, "2026-11-01 01:00 -05:00"},
		{"01:59 EST ends the repeat", "2026-11-01T06:59:00Z", ny, "2026-11-01 01:59 -05:00"},
		{"02:00 EST is not repeated", "2026-11-01T07:00:00Z", ny, "2026-11-01 02:00"},
		{"fall-back day after the repeated hour", "2026-11-01T07:30:00Z", ny, "2026-11-01 02:30"},
		{"spring-forward day before the gap", "2026-03-08T06:59:00Z", ny, "2026-03-08 01:59"},
		{"spring-forward day after the gap", "2026-03-08T07:30:00Z", ny, "2026-03-08 03:30"},
		{"half-hour DST: before the repeated half hour", "2026-04-04T14:15:00Z", lh, "2026-04-05 01:15"},
		{"half-hour DST: first 01:45", "2026-04-04T14:45:00Z", lh, "2026-04-05 01:45 +11:00"},
		{"half-hour DST: second 01:45", "2026-04-04T15:15:00Z", lh, "2026-04-05 01:45 +10:30"},
		{"half-hour DST: after the repeat", "2026-04-04T15:45:00Z", lh, "2026-04-05 02:15"},
	} {
		at, err := time.Parse(time.RFC3339, c.utc)
		if err != nil {
			t.Fatal(err)
		}
		if got := FormatInstant(at, c.loc); got != c.want {
			t.Errorf("%s: %s = %q, want %q", c.name, c.utc, got, c.want)
		}
	}
}

func TestFormatDate(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"2026-03-04", "2026-03-04"},
		{"2024-02-29", "2024-02-29"},
		{"not a date", "not a date"},
		{"", ""},
	} {
		if got := FormatDate(c.in); got != c.want {
			t.Errorf("%q = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestText(t *testing.T) {
	for _, c := range []struct{ name, in, want string }{
		{"plain", "hello", "hello"},
		{"empty", "", ""},
		{"CRLF", "line1\r\nline2", "line1 line2"},
		{"LF", "a\nb", "a b"},
		{"CR", "a\rb", "a b"},
		{"tab", "a\tb", "a b"},
		{"a run of newlines is one space", "a\n\nb", "a b"},
		{"a mixed run of newlines and tabs is one space", "a\r\n\t\nb", "a b"},
		{"only runs of newline and tab collapse, spaces stay", "a \n b", "a   b"},
		{"leading and trailing runs become a space, nothing is trimmed", "\na\n", " a "},
		{"other control runes are untouched", "a\x00b\x07c\x0bd", "a\x00b\x07c\x0bd"},
		{"non-Latin text is untouched", "Ω\nЖ", "Ω Ж"},
		{"an invalid byte is kept for the encoder", "a\xffb\n\xc3", "a\xffb \xc3"},
	} {
		if got := Text(c.in); got != c.want {
			t.Errorf("%s: %q = %q, want %q", c.name, c.in, got, c.want)
		}
	}
}

// The examples of ui/src/lib/growth.test.ts and timeline.test.ts.
func TestFormatPercentile(t *testing.T) {
	for _, c := range []struct {
		p    float64
		want string
	}{{0, "<0.1"}, {0.05, "<0.1"}, {0.1, "0.1"}, {50, "50.0"}, {57.9, "57.9"}, {50.25, "50.3"}, {99.9, "99.9"}, {99.95, ">99.9"}, {100, ">99.9"}} {
		if got := FormatPercentile(c.p); got != c.want {
			t.Errorf("FormatPercentile(%v) = %q, want %q", c.p, got, c.want)
		}
	}
}

func TestFormatZ(t *testing.T) {
	for _, c := range []struct {
		z    float64
		want string
	}{{0.207, "0.21"}, {0.2, "0.20"}, {-1.5, "-1.50"}, {-0.001, "0.00"}, {0, "0.00"}, {3.456, "3.46"}, {-0.125, "-0.12"}} {
		if got := FormatZ(c.z); got != c.want {
			t.Errorf("FormatZ(%v) = %q, want %q", c.z, got, c.want)
		}
	}
}

func TestFormatDayHeading(t *testing.T) {
	for date, want := range map[string]string{"2026-03-29": "Sun 29 Mar 2026", "2026-10-14": "Wed 14 Oct 2026"} {
		if got := FormatDayHeading(date); got != want {
			t.Errorf("FormatDayHeading(%q) = %q, want %q", date, got, want)
		}
	}
}

// The examples of ui/src/lib/entry-edit.test.ts (temperatureText) and timeline.test.ts (TEMP_UNIT_LABEL).
func TestFormatTemperature(t *testing.T) {
	feverC := units.Temperature{CHundredths: 3840, FHundredths: 10112, Typed: units.UnitC}
	feverF := units.Temperature{CHundredths: 3850, FHundredths: 10130, Typed: units.UnitF}
	for _, c := range []struct {
		t    units.Temperature
		u    model.Units
		want string
	}{
		{feverC, metric, "38.4 °C"}, {feverC, imperial, "101.12 °F"}, {feverF, imperial, "101.3 °F"}, {feverF, metric, "38.5 °C"},
		{units.Temperature{CHundredths: 3800, FHundredths: 10040, Typed: units.UnitC}, metric, "38 °C"},
		{units.Temperature{CHundredths: 3805, FHundredths: 10049, Typed: units.UnitC}, metric, "38.05 °C"},
	} {
		if got := FormatTemperature(c.t, c.u); got != c.want {
			t.Errorf("%+v in %s = %q, want %q", c.t, c.u.Temp, got, c.want)
		}
	}
}

// The examples of ui/src/lib/medication.test.ts (scheduleLabel).
func TestFormatSchedule(t *testing.T) {
	start := time.Date(2026, 11, 20, 8, 0, 0, 0, time.UTC)
	twelve := make([]string, 12)
	for i := range twelve {
		twelve[i] = fmt.Sprintf("%02d:30", i*2)
	}
	ny, madrid, kolkata := mustZone(t, "America/New_York"), mustZone(t, "Europe/Madrid"), mustZone(t, "Asia/Kolkata")
	for _, c := range []struct {
		s     schedule.Schedule
		start time.Time
		loc   *time.Location
		want  string
	}{
		{schedule.Schedule{Kind: schedule.KindNone}, start, time.UTC, "As needed"},
		{schedule.Schedule{Kind: schedule.KindTimes, Times: []string{"08:00", "20:00"}}, start, time.UTC, "08:00, 20:00"},
		{schedule.Schedule{Kind: schedule.KindTimes, Times: []string{"08:00"}}, start, time.UTC, "08:00"},
		{schedule.Schedule{Kind: schedule.KindTimes, Times: []string{"20:00", "08:00"}}, start, time.UTC, "20:00, 08:00"},
		{schedule.Schedule{Kind: schedule.KindTimes, Times: twelve}, start, time.UTC, strings.Join(twelve, ", ")},
		{schedule.Schedule{Kind: schedule.KindEvery, EveryHours: 6}, start, time.UTC, "Every 6 hours from 08:00"},
		{schedule.Schedule{Kind: schedule.KindEvery, EveryHours: 1}, start, time.UTC, "Every hour from 08:00"},
		{schedule.Schedule{Kind: schedule.KindEvery, EveryHours: 168}, start, time.UTC, "Every 168 hours from 08:00"},
		{schedule.Schedule{Kind: schedule.KindEvery, EveryHours: 8}, start, madrid, "Every 8 hours from 09:00"},
		{schedule.Schedule{Kind: schedule.KindEvery, EveryHours: 8}, time.Date(2026, 7, 1, 8, 0, 0, 0, time.UTC), madrid, "Every 8 hours from 10:00"},
		{schedule.Schedule{Kind: schedule.KindEvery, EveryHours: 4}, time.Date(2026, 3, 8, 7, 30, 0, 0, time.UTC), ny, "Every 4 hours from 03:30"},
		{schedule.Schedule{Kind: schedule.KindEvery, EveryHours: 4}, time.Date(2026, 11, 20, 23, 30, 0, 0, time.UTC), kolkata, "Every 4 hours from 05:00"},
	} {
		if got := FormatSchedule(c.s, c.start, c.loc); got != c.want {
			t.Errorf("%+v from %v in %s = %q, want %q", c.s, c.start, c.loc, got, c.want)
		}
	}
}
