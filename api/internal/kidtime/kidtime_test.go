package kidtime

import (
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
