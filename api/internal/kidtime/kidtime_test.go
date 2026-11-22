package kidtime

import (
	"strings"
	"testing"
	"time"
)

func mustLoc(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func TestDayBoundsDST(t *testing.T) {
	madrid := mustLoc(t, "Europe/Madrid")
	la := mustLoc(t, "America/Los_Angeles")
	cases := []struct {
		name    string
		loc     *time.Location
		instant time.Time
		hours   time.Duration
	}{
		{"madrid spring forward", madrid, time.Date(2026, 3, 29, 12, 0, 0, 0, madrid), 23},
		{"madrid fall back", madrid, time.Date(2026, 10, 25, 12, 0, 0, 0, madrid), 25},
		{"madrid ordinary", madrid, time.Date(2026, 6, 1, 12, 0, 0, 0, madrid), 24},
		{"la spring forward", la, time.Date(2026, 3, 8, 12, 0, 0, 0, la), 23},
		{"la fall back", la, time.Date(2026, 11, 1, 12, 0, 0, 0, la), 25},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			start, end := DayBounds(c.loc, c.instant)
			if got := end.Sub(start); got != c.hours*time.Hour {
				t.Fatalf("day length = %v, want %vh", got, int(c.hours))
			}
			if h, m, s := start.In(c.loc).Clock(); h != 0 || m != 0 || s != 0 {
				t.Fatalf("start %v is not local midnight", start)
			}
			if h, m, s := end.In(c.loc).Clock(); h != 0 || m != 0 || s != 0 {
				t.Fatalf("end %v is not local midnight", end)
			}
			if c.instant.Before(start) || !c.instant.Before(end) {
				t.Fatalf("instant %v outside [%v, %v)", c.instant, start, end)
			}
		})
	}
}

func TestDayBoundsMidnightEdge(t *testing.T) {
	madrid := mustLoc(t, "Europe/Madrid")
	// Madrid midnight 2026-06-02 is 2026-06-01T22:00Z.
	before := time.Date(2026, 6, 1, 21, 59, 59, 0, time.UTC)
	at := time.Date(2026, 6, 1, 22, 0, 0, 0, time.UTC)
	s1, e1 := DayBounds(madrid, before)
	s2, e2 := DayBounds(madrid, at)
	if !e1.Equal(at) || !s2.Equal(at) {
		t.Fatalf("boundary: day 1 ends %v, day 2 starts %v, want %v", e1, s2, at)
	}
	if s1.Equal(s2) || e1.Equal(e2) {
		t.Fatal("instants either side of local midnight share a day")
	}
}

func TestDayBoundsLaEvening(t *testing.T) {
	la := mustLoc(t, "America/Los_Angeles")
	// 2026-06-02T03:00Z is still 2026-06-01 20:00 in Los Angeles.
	start, end := DayBounds(la, time.Date(2026, 6, 2, 3, 0, 0, 0, time.UTC))
	if want := time.Date(2026, 6, 1, 0, 0, 0, 0, la); !start.Equal(want) {
		t.Fatalf("start = %v, want %v", start, want)
	}
	if want := time.Date(2026, 6, 2, 0, 0, 0, 0, la); !end.Equal(want) {
		t.Fatalf("end = %v, want %v", end, want)
	}
}

func TestParseDate(t *testing.T) {
	madrid := mustLoc(t, "Europe/Madrid")
	got, err := ParseDate("2026-03-29", madrid)
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, 3, 29, 0, 0, 0, 0, madrid); !got.Equal(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for _, bad := range []string{"", "2026-3-29", "2026-02-30", "29-03-2026", "2026-03-29T00:00:00Z", " 2026-03-29"} {
		if _, err := ParseDate(bad, madrid); err == nil {
			t.Errorf("ParseDate(%q) accepted", bad)
		}
	}
}

func TestAgeMonths(t *testing.T) {
	madrid := mustLoc(t, "Europe/Madrid")
	cases := []struct {
		name  string
		birth string
		now   time.Time
		want  int
	}{
		{"day before month", "2026-08-31", time.Date(2026, 9, 30, 12, 0, 0, 0, madrid), 0},
		{"month turns", "2026-08-31", time.Date(2026, 10, 1, 12, 0, 0, 0, madrid), 1},
		{"birth day itself", "2026-08-31", time.Date(2026, 8, 31, 0, 0, 0, 0, madrid), 0},
		{"leap day, day before first birthday", "2024-02-29", time.Date(2025, 2, 28, 12, 0, 0, 0, madrid), 11},
		{"leap day, first birthday", "2024-02-29", time.Date(2025, 3, 1, 12, 0, 0, 0, madrid), 12},
		{"leap day, next leap day", "2024-02-29", time.Date(2028, 2, 29, 12, 0, 0, 0, madrid), 48},
		{"born tomorrow", "2026-06-02", time.Date(2026, 6, 1, 12, 0, 0, 0, madrid), 0},
		{"born next year", "2027-01-15", time.Date(2026, 6, 1, 12, 0, 0, 0, madrid), 0},
		{"one year", "2025-06-01", time.Date(2026, 6, 1, 8, 0, 0, 0, madrid), 12},
		// 2026-06-01T23:30Z is already 2026-06-02 in Madrid, where the kid is born "today".
		{"uses the kid zone", "2026-06-02", time.Date(2026, 6, 1, 23, 30, 0, 0, time.UTC), 0},
		{"zone decides the month turn", "2026-05-02", time.Date(2026, 6, 1, 23, 30, 0, 0, time.UTC), 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := AgeMonths(c.birth, madrid, c.now)
			if err != nil {
				t.Fatal(err)
			}
			if got != c.want {
				t.Fatalf("AgeMonths(%s) = %d, want %d", c.birth, got, c.want)
			}
		})
	}
	if _, err := AgeMonths("nope", madrid, time.Now()); err == nil {
		t.Fatal("bad birth date accepted")
	}
}

func TestBandFor(t *testing.T) {
	cases := map[int]Band{0: Band0to11, 11: Band0to11, 12: Band12to23, 23: Band12to23, 24: Band24Plus, 120: Band24Plus}
	for months, want := range cases {
		if got := BandFor(months); got != want {
			t.Errorf("BandFor(%d) = %v, want %v", months, got, want)
		}
	}
}

func TestBandStart(t *testing.T) {
	madrid := mustLoc(t, "Europe/Madrid")
	cases := []struct {
		birth string
		band  Band
		want  time.Time
	}{
		{"2026-03-04", Band0to11, time.Date(2026, 3, 4, 0, 0, 0, 0, madrid)},
		{"2026-03-04", Band12to23, time.Date(2027, 3, 4, 0, 0, 0, 0, madrid)},
		{"2026-03-04", Band24Plus, time.Date(2028, 3, 4, 0, 0, 0, 0, madrid)},
		{"2024-02-29", Band12to23, time.Date(2025, 3, 1, 0, 0, 0, 0, madrid)},
		{"2024-02-29", Band24Plus, time.Date(2026, 3, 1, 0, 0, 0, 0, madrid)},
	}
	for _, c := range cases {
		got, err := BandStart(c.birth, madrid, c.band)
		if err != nil {
			t.Fatal(err)
		}
		if !got.Equal(c.want) {
			t.Errorf("BandStart(%s, %v) = %v, want %v", c.birth, c.band, got, c.want)
		}
	}
	if _, err := BandStart("nope", madrid, Band12to23); err == nil {
		t.Fatal("bad birth date accepted")
	}
}

// BandStart is the first local day AgeMonths reports the band's first month, for every kind of birth day.
func TestBandStartAgreesWithAgeMonths(t *testing.T) {
	madrid := mustLoc(t, "Europe/Madrid")
	for _, birth := range []string{"2026-03-04", "2024-02-29", "2025-08-31", "2025-01-31", "2023-12-31"} {
		for b, months := range map[Band]int{Band12to23: 12, Band24Plus: 24} {
			start, err := BandStart(birth, madrid, b)
			if err != nil {
				t.Fatal(err)
			}
			at, _ := AgeMonths(birth, madrid, start)
			before, _ := AgeMonths(birth, madrid, start.AddDate(0, 0, -1))
			if at != months || before != months-1 {
				t.Errorf("%s band %v: months at start %d, day before %d; want %d and %d", birth, b, at, before, months, months-1)
			}
		}
	}
}

func ip(v int) *int { return &v }

func TestAgeDays(t *testing.T) {
	madrid := mustLoc(t, "Europe/Madrid")
	la := mustLoc(t, "America/Los_Angeles")
	cases := []struct {
		name      string
		loc       *time.Location
		birth, on string
		want      int
	}{
		{"same day", madrid, "2026-06-01", "2026-06-01", 0},
		{"next day", madrid, "2026-06-01", "2026-06-02", 1},
		{"madrid spring forward", madrid, "2026-03-28", "2026-03-30", 2},
		{"madrid fall back", madrid, "2026-10-24", "2026-10-26", 2},
		{"madrid fall back, the long day itself", madrid, "2026-10-25", "2026-10-26", 1},
		{"la spring forward", la, "2026-03-07", "2026-03-09", 2},
		{"la fall back", la, "2026-10-31", "2026-11-02", 2},
		{"la fall back, the long day itself", la, "2026-11-01", "2026-11-02", 1},
		{"madrid across both changes", madrid, "2026-01-01", "2027-01-01", 365},
		{"leap day crossed", madrid, "2024-02-28", "2024-03-01", 2},
		{"no leap day", madrid, "2023-02-28", "2023-03-01", 1},
		{"leap day born, one year later", madrid, "2024-02-29", "2025-02-28", 365},
		{"leap year length", la, "2024-01-01", "2025-01-01", 366},
		{"on before birth", madrid, "2026-06-02", "2026-06-01", -1},
		{"on well before birth", la, "2026-06-02", "2026-03-09", -85},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := AgeDays(c.birth, c.on, c.loc)
			if err != nil {
				t.Fatal(err)
			}
			if got != c.want {
				t.Fatalf("AgeDays(%s, %s) = %d, want %d", c.birth, c.on, got, c.want)
			}
		})
	}
	for _, bad := range [][2]string{{"secret-birth", "2026-06-01"}, {"2026-06-01", "secret-on"}, {"", "2026-06-01"}, {"2026-02-30", "2026-06-01"}} {
		_, err := AgeDays(bad[0], bad[1], madrid)
		if err == nil {
			t.Errorf("AgeDays(%q, %q) accepted", bad[0], bad[1])
			continue
		}
		for _, in := range bad {
			if in != "" && strings.Contains(err.Error(), in) {
				t.Errorf("error %q carries input %q", err, in)
			}
		}
	}
}

func TestCorrectionDays(t *testing.T) {
	cases := []struct {
		name        string
		weeks, days *int
		want        int
	}{
		{"36w6d", ip(36), ip(6), 22},
		{"37w0d", ip(37), ip(0), 0},
		{"22w0d", ip(22), ip(0), 126},
		{"28w3d", ip(28), ip(3), 81},
		{"28w0d", ip(28), ip(0), 84},
		{"35w2d", ip(35), ip(2), 33},
		{"40w0d", ip(40), ip(0), 0},
		{"44w6d", ip(44), ip(6), 0},
		{"nil weeks", nil, ip(3), 0},
		{"nil days", ip(30), nil, 0},
		{"both nil", nil, nil, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := CorrectionDays(c.weeks, c.days); got != c.want {
				t.Fatalf("CorrectionDays = %d, want %d", got, c.want)
			}
		})
	}
}

func TestAgeBasisOn(t *testing.T) {
	madrid := mustLoc(t, "Europe/Madrid")
	cases := []struct {
		name        string
		birth       string
		weeks, days *int
		on          string
		want        AgeBasis
	}{
		{"leo day before second birthday", "2024-08-20", ip(35), ip(2), "2026-08-19", AgeBasis{ChronologicalDays: 729, ScoringDays: 696, Corrected: true}},
		{"leo on second birthday", "2024-08-20", ip(35), ip(2), "2026-08-20", AgeBasis{ChronologicalDays: 730, ScoringDays: 730}},
		{"leo after second birthday", "2024-08-20", ip(35), ip(2), "2026-08-21", AgeBasis{ChronologicalDays: 731, ScoringDays: 731}},
		{"leo at 6 months", "2024-08-20", ip(35), ip(2), "2025-02-20", AgeBasis{ChronologicalDays: 184, ScoringDays: 151, Corrected: true}},
		{"leo on the birth date", "2024-08-20", ip(35), ip(2), "2024-08-20", AgeBasis{ChronologicalDays: 0, ScoringDays: -33, Corrected: true}},
		{"term kid", "2025-01-10", ip(40), ip(0), "2025-03-10", AgeBasis{ChronologicalDays: 59, ScoringDays: 59}},
		{"term kid at 37w0d", "2025-01-10", ip(37), ip(0), "2025-03-10", AgeBasis{ChronologicalDays: 59, ScoringDays: 59}},
		{"no gestation recorded", "2025-01-10", nil, nil, "2025-03-10", AgeBasis{ChronologicalDays: 59, ScoringDays: 59}},
		{"36w6d", "2025-01-10", ip(36), ip(6), "2025-03-10", AgeBasis{ChronologicalDays: 59, ScoringDays: 37, Corrected: true}},
		{"22w0d", "2025-01-10", ip(22), ip(0), "2025-03-10", AgeBasis{ChronologicalDays: 59, ScoringDays: -67, Corrected: true}},
		{"leap day second birthday is 1 March", "2024-02-29", ip(30), ip(0), "2026-02-28", AgeBasis{ChronologicalDays: 730, ScoringDays: 730 - 70, Corrected: true}},
		{"leap day second birthday not corrected", "2024-02-29", ip(30), ip(0), "2026-03-01", AgeBasis{ChronologicalDays: 731, ScoringDays: 731}},
		{"on before birth, term", "2025-01-10", ip(40), ip(0), "2025-01-09", AgeBasis{ChronologicalDays: -1, ScoringDays: -1}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := AgeBasisOn(c.birth, c.weeks, c.days, c.on, madrid)
			if err != nil {
				t.Fatal(err)
			}
			if got != c.want {
				t.Fatalf("AgeBasisOn = %+v, want %+v", got, c.want)
			}
		})
	}
	for _, bad := range [][2]string{{"secret-birth", "2026-06-01"}, {"2026-06-01", "secret-on"}} {
		_, err := AgeBasisOn(bad[0], ip(30), ip(0), bad[1], madrid)
		if err == nil {
			t.Errorf("AgeBasisOn(%q, %q) accepted", bad[0], bad[1])
		} else if strings.Contains(err.Error(), "secret") {
			t.Errorf("error %q carries input", err)
		}
	}
}

func TestCorrectedBirthDate(t *testing.T) {
	madrid := mustLoc(t, "Europe/Madrid")
	cases := []struct {
		name        string
		birth       string
		weeks, days *int
		on          string
		want        string
		ok          bool
	}{
		{"leo on the birth date", "2024-08-20", ip(35), ip(2), "2024-08-20", "2024-09-22", true},
		{"leo day before second birthday", "2024-08-20", ip(35), ip(2), "2026-08-19", "2024-09-22", true},
		{"leo on second birthday", "2024-08-20", ip(35), ip(2), "2026-08-20", "", false},
		{"leo later", "2024-08-20", ip(35), ip(2), "2027-01-01", "", false},
		{"month rollover", "2025-12-20", ip(35), ip(2), "2026-01-01", "2026-01-22", true},
		{"term kid", "2025-01-10", ip(40), ip(0), "2025-03-10", "", false},
		{"37w0d", "2025-01-10", ip(37), ip(0), "2025-03-10", "", false},
		{"no gestation", "2025-01-10", nil, nil, "2025-03-10", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok, err := CorrectedBirthDate(c.birth, c.weeks, c.days, c.on, madrid)
			if err != nil {
				t.Fatal(err)
			}
			if got != c.want || ok != c.ok {
				t.Fatalf("CorrectedBirthDate = (%q, %v), want (%q, %v)", got, ok, c.want, c.ok)
			}
		})
	}
	for _, bad := range [][2]string{{"secret-birth", "2026-06-01"}, {"2026-06-01", "secret-on"}} {
		got, ok, err := CorrectedBirthDate(bad[0], ip(30), ip(0), bad[1], madrid)
		if err == nil || got != "" || ok {
			t.Errorf("CorrectedBirthDate(%q, %q) = (%q, %v, %v)", bad[0], bad[1], got, ok, err)
		} else if strings.Contains(err.Error(), "secret") {
			t.Errorf("error %q carries input", err)
		}
	}
}
