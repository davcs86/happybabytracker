package schedule

import (
	"errors"
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

const instantLayout = "2006-01-02 15:04 -0700"

func mustLoc(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func at(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse(instantLayout, s)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func times(hhmm ...string) Schedule { return Schedule{Kind: KindTimes, Times: hhmm} }

func every(h int) Schedule { return Schedule{Kind: KindEvery, EveryHours: h} }

func course(s Schedule, start time.Time, endOn string) Course {
	return Course{Schedule: s, StartAt: start, EndOn: endOn, Active: true}
}

func local(loc *time.Location, ts []time.Time) []string {
	out := make([]string, len(ts))
	for i, v := range ts {
		out[i] = v.In(loc).Format(instantLayout)
	}
	return out
}

func wantLocal(t *testing.T, loc *time.Location, got []time.Time, want ...string) {
	t.Helper()
	if g := local(loc, got); !reflect.DeepEqual(g, want) && !(len(g) == 0 && len(want) == 0) {
		t.Fatalf("occurrences:\n got %q\nwant %q", g, want)
	}
}

func TestEveryOrdinaryDay(t *testing.T) {
	ny := mustLoc(t, "America/New_York")
	c := course(every(6), at(t, "2026-01-10 08:00 -0500"), "")
	got := Occurrences(c, ny, at(t, "2026-01-10 00:00 -0500"), at(t, "2026-01-11 07:59 -0500"))
	wantLocal(t, ny, got, "2026-01-10 08:00 -0500", "2026-01-10 14:00 -0500", "2026-01-10 20:00 -0500", "2026-01-11 02:00 -0500")
}

func TestEveryKeepsAbsoluteGridAcrossDST(t *testing.T) {
	ny := mustLoc(t, "America/New_York")
	c := course(every(6), at(t, "2026-03-07 08:00 -0500"), "")
	got := Occurrences(c, ny, at(t, "2026-03-07 00:00 -0500"), at(t, "2026-03-08 23:59 -0400"))
	wantLocal(t, ny, got,
		"2026-03-07 08:00 -0500", "2026-03-07 14:00 -0500", "2026-03-07 20:00 -0500",
		"2026-03-08 03:00 -0400", "2026-03-08 09:00 -0400", "2026-03-08 15:00 -0400", "2026-03-08 21:00 -0400")
	for i := 1; i < len(got); i++ {
		if d := got[i].Sub(got[i-1]); d != 6*time.Hour {
			t.Fatalf("gap %d = %v, want 6h", i, d)
		}
	}
	fall := course(every(6), at(t, "2026-10-31 08:00 -0400"), "")
	got = Occurrences(fall, ny, at(t, "2026-10-31 00:00 -0400"), at(t, "2026-11-01 23:59 -0500"))
	wantLocal(t, ny, got,
		"2026-10-31 08:00 -0400", "2026-10-31 14:00 -0400", "2026-10-31 20:00 -0400",
		"2026-11-01 01:00 -0500", "2026-11-01 07:00 -0500", "2026-11-01 13:00 -0500", "2026-11-01 19:00 -0500")
}

// Each case lists every occurrence of a one-time course over the three days around a transition.
func TestTimesDSTResolution(t *testing.T) {
	cases := []struct {
		zone, hhmm, start, from, to string
		want                        []string
	}{
		{"America/New_York", "02:30", "2026-03-07 00:00 -0500", "2026-03-07 00:00 -0500", "2026-03-09 23:59 -0400",
			[]string{"2026-03-07 02:30 -0500", "2026-03-08 03:00 -0400", "2026-03-09 02:30 -0400"}},
		{"America/New_York", "01:30", "2026-10-31 00:00 -0400", "2026-10-31 00:00 -0400", "2026-11-02 23:59 -0500",
			[]string{"2026-10-31 01:30 -0400", "2026-11-01 01:30 -0400", "2026-11-02 01:30 -0500"}},
		{"Europe/Madrid", "02:30", "2026-03-28 00:00 +0100", "2026-03-28 00:00 +0100", "2026-03-30 23:59 +0200",
			[]string{"2026-03-28 02:30 +0100", "2026-03-29 03:00 +0200", "2026-03-30 02:30 +0200"}},
		{"Europe/Madrid", "02:30", "2026-10-24 00:00 +0200", "2026-10-24 00:00 +0200", "2026-10-26 23:59 +0100",
			[]string{"2026-10-24 02:30 +0200", "2026-10-25 02:30 +0200", "2026-10-26 02:30 +0100"}},
		{"Pacific/Auckland", "02:30", "2026-09-26 00:00 +1200", "2026-09-26 00:00 +1200", "2026-09-28 23:59 +1300",
			[]string{"2026-09-26 02:30 +1200", "2026-09-27 03:00 +1300", "2026-09-28 02:30 +1300"}},
		{"Pacific/Auckland", "02:30", "2026-04-04 00:00 +1300", "2026-04-04 00:00 +1300", "2026-04-06 23:59 +1200",
			[]string{"2026-04-04 02:30 +1300", "2026-04-05 02:30 +1300", "2026-04-06 02:30 +1200"}},
		{"Australia/Lord_Howe", "02:15", "2026-10-03 00:00 +1030", "2026-10-03 00:00 +1030", "2026-10-05 23:59 +1100",
			[]string{"2026-10-03 02:15 +1030", "2026-10-04 02:30 +1100", "2026-10-05 02:15 +1100"}},
		{"Australia/Lord_Howe", "01:45", "2026-04-04 00:00 +1100", "2026-04-04 00:00 +1100", "2026-04-06 23:59 +1030",
			[]string{"2026-04-04 01:45 +1100", "2026-04-05 01:45 +1100", "2026-04-06 01:45 +1030"}},
		{"America/Havana", "00:00", "2026-03-07 00:00 -0500", "2026-03-07 00:00 -0500", "2026-03-09 23:59 -0400",
			[]string{"2026-03-07 00:00 -0500", "2026-03-08 01:00 -0400", "2026-03-09 00:00 -0400"}},
		{"America/Havana", "00:30", "2026-10-31 00:00 -0400", "2026-10-31 00:00 -0400", "2026-11-02 23:59 -0500",
			[]string{"2026-10-31 00:30 -0400", "2026-11-01 00:30 -0400", "2026-11-02 00:30 -0500"}},
		{"America/Santiago", "00:00", "2026-09-05 00:00 -0400", "2026-09-05 00:00 -0400", "2026-09-07 23:59 -0300",
			[]string{"2026-09-05 00:00 -0400", "2026-09-06 01:00 -0300", "2026-09-07 00:00 -0300"}},
		{"America/Santiago", "23:30", "2026-04-03 00:00 -0300", "2026-04-03 00:00 -0300", "2026-04-05 23:59 -0400",
			[]string{"2026-04-03 23:30 -0300", "2026-04-04 23:30 -0300", "2026-04-05 23:30 -0400"}},
		{"America/Santiago", "00:00", "2026-04-04 00:00 -0300", "2026-04-04 00:00 -0300", "2026-04-06 23:59 -0400",
			[]string{"2026-04-04 00:00 -0300", "2026-04-05 00:00 -0400", "2026-04-06 00:00 -0400"}},
	}
	for _, c := range cases {
		t.Run(c.zone+" "+c.hhmm+" "+c.start[:10], func(t *testing.T) {
			loc := mustLoc(t, c.zone)
			got := Occurrences(course(times(c.hhmm), at(t, c.start), ""), loc, at(t, c.from), at(t, c.to))
			wantLocal(t, loc, got, c.want...)
		})
	}
}

func TestTimesGapCollapsesToOneOccurrence(t *testing.T) {
	ny := mustLoc(t, "America/New_York")
	c := course(times("02:15", "02:45", "03:00", "03:30"), at(t, "2026-03-08 00:00 -0500"), "2026-03-08")
	got := Occurrences(c, ny, at(t, "2026-03-08 00:00 -0500"), at(t, "2026-03-09 00:00 -0400"))
	wantLocal(t, ny, got, "2026-03-08 03:00 -0400", "2026-03-08 03:30 -0400")
}

func TestMidnightDSTEndOn(t *testing.T) {
	cases := []struct {
		zone, start, endOn, to string
		want                   []string
	}{
		{"America/Havana", "2026-03-06 00:00 -0500", "2026-03-08", "2026-03-12 00:00 -0400",
			[]string{"2026-03-06 00:00 -0500", "2026-03-07 00:00 -0500", "2026-03-08 01:00 -0400"}},
		{"America/Havana", "2026-03-06 00:00 -0500", "2026-03-07", "2026-03-12 00:00 -0400",
			[]string{"2026-03-06 00:00 -0500", "2026-03-07 00:00 -0500"}},
		{"America/Santiago", "2026-09-04 00:00 -0400", "2026-09-06", "2026-09-10 00:00 -0300",
			[]string{"2026-09-04 00:00 -0400", "2026-09-05 00:00 -0400", "2026-09-06 01:00 -0300"}},
		{"America/Santiago", "2026-09-04 00:00 -0400", "2026-09-05", "2026-09-10 00:00 -0300",
			[]string{"2026-09-04 00:00 -0400", "2026-09-05 00:00 -0400"}},
		{"America/Santiago", "2026-04-03 00:00 -0300", "2026-04-04", "2026-04-10 00:00 -0400",
			[]string{"2026-04-03 00:00 -0300", "2026-04-04 00:00 -0300"}},
	}
	for _, c := range cases {
		t.Run(c.zone+" "+c.endOn, func(t *testing.T) {
			loc := mustLoc(t, c.zone)
			cr := course(times("00:00"), at(t, c.start), c.endOn)
			got := Occurrences(cr, loc, at(t, c.start), at(t, c.to))
			wantLocal(t, loc, got, c.want...)
		})
	}
}

func TestEndOnIsInclusive(t *testing.T) {
	cases := []struct {
		name, zone, hhmm, start, endOn string
		want                           []string
	}{
		{"santiago fall-back day keeps the earlier 23:59", "America/Santiago", "23:59", "2026-04-03 12:00 -0300", "2026-04-04",
			[]string{"2026-04-03 23:59 -0300", "2026-04-04 23:59 -0300"}},
		{"new york ordinary day", "America/New_York", "23:59", "2026-01-10 12:00 -0500", "2026-01-11",
			[]string{"2026-01-10 23:59 -0500", "2026-01-11 23:59 -0500"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			loc := mustLoc(t, c.zone)
			start := at(t, c.start)
			got := Occurrences(course(times(c.hhmm), start, c.endOn), loc, start, start.Add(30*24*time.Hour))
			wantLocal(t, loc, got, c.want...)
		})
	}
}

func TestCourseStartingMidDaySkipsEarlierTime(t *testing.T) {
	madrid := mustLoc(t, "Europe/Madrid")
	c := course(times("08:00", "20:00"), at(t, "2026-06-01 13:00 +0200"), "")
	got := Occurrences(c, madrid, at(t, "2026-06-01 00:00 +0200"), at(t, "2026-06-02 23:59 +0200"))
	wantLocal(t, madrid, got, "2026-06-01 20:00 +0200", "2026-06-02 08:00 +0200", "2026-06-02 20:00 +0200")
	exact := course(times("08:00"), at(t, "2026-06-01 08:00 +0200"), "")
	got = Occurrences(exact, madrid, at(t, "2026-06-01 00:00 +0200"), at(t, "2026-06-01 23:59 +0200"))
	wantLocal(t, madrid, got, "2026-06-01 08:00 +0200")
}

func TestCourseEndingTodayAndEndOnBeforeStart(t *testing.T) {
	madrid := mustLoc(t, "Europe/Madrid")
	now := at(t, "2026-06-10 12:00 +0200")
	today := course(times("08:00", "20:00"), at(t, "2026-06-01 09:00 +0200"), "2026-06-10")
	got := Occurrences(today, madrid, now.Add(-24*time.Hour), now.Add(72*time.Hour))
	wantLocal(t, madrid, got, "2026-06-09 20:00 +0200", "2026-06-10 08:00 +0200", "2026-06-10 20:00 +0200")
	if n, ok := Next(today, madrid, now); !ok || !n.Equal(at(t, "2026-06-10 20:00 +0200")) {
		t.Fatalf("Next = %v %v", n, ok)
	}
	if n, ok := Next(today, madrid, at(t, "2026-06-10 20:00 +0200")); ok {
		t.Fatalf("Next past end_on = %v, want none", n)
	}
	for _, endOn := range []string{"2026-05-31", "2026-06-09"} {
		past := course(times("08:00"), at(t, "2026-06-10 09:00 +0200"), endOn)
		if got := Occurrences(past, madrid, now.Add(-30*24*time.Hour), now.Add(30*24*time.Hour)); len(got) != 0 {
			t.Fatalf("end_on %s before start: %v", endOn, got)
		}
		if _, ok := Next(past, madrid, now.Add(-30*24*time.Hour)); ok {
			t.Fatalf("end_on %s before start: Next found one", endOn)
		}
		ev := course(every(4), at(t, "2026-06-10 09:00 +0200"), endOn)
		if got := Occurrences(ev, madrid, now.Add(-30*24*time.Hour), now.Add(30*24*time.Hour)); len(got) != 0 {
			t.Fatalf("every with end_on %s before start: %v", endOn, got)
		}
	}
	ev := course(every(8), at(t, "2026-06-10 06:00 +0200"), "2026-06-10")
	wantLocal(t, madrid, Occurrences(ev, madrid, now.Add(-48*time.Hour), now.Add(48*time.Hour)),
		"2026-06-10 06:00 +0200", "2026-06-10 14:00 +0200", "2026-06-10 22:00 +0200")
}

func TestFromToInclusive(t *testing.T) {
	ny := mustLoc(t, "America/New_York")
	c := course(times("08:00", "12:00", "20:00"), at(t, "2026-01-01 00:00 -0500"), "")
	from, to := at(t, "2026-01-05 08:00 -0500"), at(t, "2026-01-05 20:00 -0500")
	wantLocal(t, ny, Occurrences(c, ny, from, to), "2026-01-05 08:00 -0500", "2026-01-05 12:00 -0500", "2026-01-05 20:00 -0500")
	wantLocal(t, ny, Occurrences(c, ny, from.Add(time.Nanosecond), to.Add(-time.Nanosecond)), "2026-01-05 12:00 -0500")
	wantLocal(t, ny, Occurrences(c, ny, from, from), "2026-01-05 08:00 -0500")
	if got := Occurrences(c, ny, to, from); len(got) != 0 {
		t.Fatalf("from after to: %v", got)
	}
	ev := course(every(4), at(t, "2026-01-05 08:00 -0500"), "")
	wantLocal(t, ny, Occurrences(ev, ny, from, at(t, "2026-01-05 16:00 -0500")),
		"2026-01-05 08:00 -0500", "2026-01-05 12:00 -0500", "2026-01-05 16:00 -0500")
}

func TestNext(t *testing.T) {
	ny := mustLoc(t, "America/New_York")
	c := course(times("08:00", "20:00"), at(t, "2026-01-01 00:00 -0500"), "2026-01-06")
	steps := []struct{ after, want string }{
		{"2025-12-31 23:00 -0500", "2026-01-01 08:00 -0500"},
		{"2026-01-05 07:59 -0500", "2026-01-05 08:00 -0500"},
		{"2026-01-05 08:00 -0500", "2026-01-05 20:00 -0500"},
		{"2026-01-05 20:00 -0500", "2026-01-06 08:00 -0500"},
		{"2026-01-05 23:59 -0500", "2026-01-06 08:00 -0500"},
	}
	for _, s := range steps {
		n, ok := Next(c, ny, at(t, s.after))
		if !ok || !n.Equal(at(t, s.want)) {
			t.Fatalf("Next(%s) = %v %v, want %s", s.after, n.In(ny), ok, s.want)
		}
	}
	if n, ok := Next(c, ny, at(t, "2026-01-06 20:00 -0500")); ok {
		t.Fatalf("Next past end_on = %v", n.In(ny))
	}
	ev := course(every(6), at(t, "2026-01-01 08:00 -0500"), "")
	if n, ok := Next(ev, ny, at(t, "2026-01-03 21:00 -0500")); !ok || !n.Equal(at(t, "2026-01-04 02:00 -0500")) {
		t.Fatalf("every Next = %v %v", n.In(ny), ok)
	}
	if n, ok := Next(ev, ny, at(t, "2026-01-04 02:00 -0500")); !ok || !n.Equal(at(t, "2026-01-04 08:00 -0500")) {
		t.Fatalf("every Next on the grid = %v %v", n.In(ny), ok)
	}
	if n, ok := Next(ev, ny, at(t, "2025-06-01 00:00 -0400")); !ok || !n.Equal(at(t, "2026-01-01 08:00 -0500")) {
		t.Fatalf("every Next before start = %v %v", n.In(ny), ok)
	}
	far := course(times("09:00"), at(t, "2026-01-01 00:00 -0500"), "")
	if n, ok := Next(far, ny, at(t, "2031-07-04 10:00 -0400")); !ok || !n.Equal(at(t, "2031-07-05 09:00 -0400")) {
		t.Fatalf("Next years after start = %v %v", n.In(ny), ok)
	}
}

func TestInactiveAndNoneYieldNothing(t *testing.T) {
	ny := mustLoc(t, "America/New_York")
	start := at(t, "2026-01-01 00:00 -0500")
	inactive := course(times("08:00"), start, "")
	inactive.Active = false
	none := course(Schedule{Kind: KindNone}, start, "")
	invalid := course(every(0), start, "")
	badEnd := course(times("08:00"), start, "2026-1-5")
	for name, c := range map[string]Course{"inactive": inactive, "none": none, "invalid": invalid, "bad end_on": badEnd} {
		if got := Occurrences(c, ny, start, start.Add(10*24*time.Hour)); len(got) != 0 {
			t.Fatalf("%s: %v", name, got)
		}
		if _, ok := Next(c, ny, start); ok {
			t.Fatalf("%s: Next found one", name)
		}
	}
}

func TestLeapDayAndYearEnd(t *testing.T) {
	madrid := mustLoc(t, "Europe/Madrid")
	leap := course(times("09:00"), at(t, "2028-02-28 00:00 +0100"), "2028-02-29")
	wantLocal(t, madrid, Occurrences(leap, madrid, at(t, "2028-02-27 00:00 +0100"), at(t, "2028-03-05 00:00 +0100")),
		"2028-02-28 09:00 +0100", "2028-02-29 09:00 +0100")
	open := course(times("09:00"), at(t, "2028-02-28 00:00 +0100"), "")
	if n, ok := Next(open, madrid, at(t, "2028-02-29 09:00 +0100")); !ok || !n.Equal(at(t, "2028-03-01 09:00 +0100")) {
		t.Fatalf("Next after leap day = %v %v", n, ok)
	}
	auckland := mustLoc(t, "Pacific/Auckland")
	ye := course(times("00:15", "23:30"), at(t, "2026-12-31 00:00 +1300"), "2027-01-01")
	wantLocal(t, auckland, Occurrences(ye, auckland, at(t, "2026-12-30 00:00 +1300"), at(t, "2027-01-03 00:00 +1300")),
		"2026-12-31 00:15 +1300", "2026-12-31 23:30 +1300", "2027-01-01 00:15 +1300", "2027-01-01 23:30 +1300")
	if n, ok := Next(ye, auckland, at(t, "2026-12-31 23:30 +1300")); !ok || !n.Equal(at(t, "2027-01-01 00:15 +1300")) {
		t.Fatalf("Next over the year end = %v %v", n, ok)
	}
}

func TestDayEnd(t *testing.T) {
	cases := []struct{ zone, t, want string }{
		{"Europe/Madrid", "2026-06-01 12:00 +0200", "2026-06-02 00:00 +0200"},
		{"Europe/Madrid", "2026-06-01 00:00 +0200", "2026-06-02 00:00 +0200"},
		{"America/New_York", "2026-03-08 12:00 -0400", "2026-03-09 00:00 -0400"},
		{"America/Havana", "2026-03-07 12:00 -0500", "2026-03-08 01:00 -0400"},
		{"America/Havana", "2026-03-07 23:59 -0500", "2026-03-08 01:00 -0400"},
		{"America/Havana", "2026-10-31 18:00 -0400", "2026-11-01 00:00 -0400"},
		{"America/Santiago", "2026-09-05 12:00 -0400", "2026-09-06 01:00 -0300"},
		{"America/Santiago", "2026-04-04 23:30 -0400", "2026-04-05 00:00 -0400"},
		{"America/Santiago", "2026-04-04 23:30 -0300", "2026-04-05 00:00 -0400"},
		{"Australia/Lord_Howe", "2026-10-03 23:00 +1030", "2026-10-04 00:00 +1030"},
		{"Pacific/Auckland", "2026-12-31 23:59 +1300", "2027-01-01 00:00 +1300"},
		{"America/New_York", "2026-12-31 23:59 -0500", "2027-01-01 00:00 -0500"},
		{"UTC", "2028-02-28 23:59 +0000", "2028-02-29 00:00 +0000"},
	}
	for _, c := range cases {
		t.Run(c.zone+" "+c.t, func(t *testing.T) {
			loc := mustLoc(t, c.zone)
			got := DayEnd(loc, at(t, c.t))
			if !got.Equal(at(t, c.want)) {
				t.Fatalf("DayEnd = %s, want %s", got.In(loc).Format(instantLayout), c.want)
			}
		})
	}
}

func TestValid(t *testing.T) {
	thirteen := make([]string, constants.MaxScheduleTimes+1)
	for i := range thirteen {
		thirteen[i] = time.Date(2026, 1, 1, i, 0, 0, 0, time.UTC).Format("15:04")
	}
	ok := []Schedule{
		{Kind: KindNone},
		times("08:00"),
		times("00:00", "23:59"),
		times("20:00", "08:00"),
		times(thirteen[:constants.MaxScheduleTimes]...),
		every(constants.MinEveryHours),
		every(constants.MaxEveryHours),
	}
	for _, s := range ok {
		if err := Valid(s); err != nil {
			t.Fatalf("Valid(%+v) = %v", s, err)
		}
	}
	bad := []struct {
		s    Schedule
		want error
	}{
		{Schedule{}, ErrKind},
		{Schedule{Kind: "weekly"}, ErrKind},
		{times(), ErrTimes},
		{times(thirteen...), ErrTimes},
		{times("08:00", "08:00"), ErrTimes},
		{times("24:00"), ErrTimes},
		{times("7:5"), ErrTimes},
		{times("7:05"), ErrTimes},
		{times("07:60"), ErrTimes},
		{times("07-05"), ErrTimes},
		{times("07:05 "), ErrTimes},
		{times("０7:05"), ErrTimes},
		{times(""), ErrTimes},
		{every(constants.MinEveryHours - 1), ErrHours},
		{every(constants.MaxEveryHours + 1), ErrHours},
		{every(-6), ErrHours},
		{Schedule{Kind: KindTimes, Times: []string{"08:00"}, EveryHours: 6}, ErrHours},
		{Schedule{Kind: KindEvery, Times: []string{"08:00"}, EveryHours: 6}, ErrTimes},
		{Schedule{Kind: KindNone, Times: []string{"08:00"}}, ErrTimes},
		{Schedule{Kind: KindNone, EveryHours: 6}, ErrHours},
	}
	for _, b := range bad {
		err := Valid(b.s)
		if !errors.Is(err, b.want) {
			t.Fatalf("Valid(%+v) = %v, want %v", b.s, err, b.want)
		}
		for _, v := range b.s.Times {
			if v != "" && strings.Contains(err.Error(), v) {
				t.Fatalf("error %q echoes the value %q", err, v)
			}
		}
	}
}

func TestOccurrencesAreBounded(t *testing.T) {
	utc := time.UTC
	start := at(t, "2026-01-01 00:00 +0000")
	for name, c := range map[string]Course{
		"every": course(every(constants.MinEveryHours), start, ""),
		"times": course(times("00:00", "00:01", "00:02", "00:03", "00:04", "00:05", "00:06", "00:07", "00:08", "00:09", "00:10", "00:11"), start, ""),
	} {
		got := Occurrences(c, utc, start, start.AddDate(50, 0, 0))
		if len(got) != constants.MaxScheduleOccurrences {
			t.Fatalf("%s: %d occurrences, want the cap %d", name, len(got), constants.MaxScheduleOccurrences)
		}
		if !got[0].Equal(start) {
			t.Fatalf("%s: the capped list does not start at from", name)
		}
	}
}

// Properties over seeded random courses: ascending without duplicates, inside every bound, deterministic, agreeing with Next.
func TestOccurrenceProperties(t *testing.T) {
	zones := []string{"America/New_York", "Europe/Madrid", "Pacific/Auckland", "Australia/Lord_Howe", "America/Havana", "America/Santiago", "UTC"}
	r := rand.New(rand.NewPCG(6, 5))
	base := at(t, "2026-01-01 00:00 +0000")
	for i := range 400 {
		loc := mustLoc(t, zones[i%len(zones)])
		var s Schedule
		if r.IntN(2) == 0 {
			s = every(constants.MinEveryHours + r.IntN(constants.MaxEveryHours))
		} else {
			seen := map[string]bool{}
			for range 1 + r.IntN(constants.MaxScheduleTimes) {
				v := time.Date(2026, 1, 1, r.IntN(24), r.IntN(4)*15, 0, 0, time.UTC).Format("15:04")
				if !seen[v] {
					seen[v] = true
					s.Times = append(s.Times, v)
				}
			}
			s.Kind = KindTimes
		}
		start := base.Add(time.Duration(r.IntN(365*24*60)) * time.Minute)
		endOn := ""
		var endBound time.Time
		if r.IntN(2) == 0 {
			endDay := start.In(loc).AddDate(0, 0, r.IntN(20)-2)
			endOn = endDay.Format("2006-01-02")
			endBound = DayEnd(loc, endDay)
		}
		c := course(s, start, endOn)
		from := start.Add(time.Duration(r.IntN(96)-48) * time.Hour)
		to := from.Add(time.Duration(r.IntN(30*24)) * time.Hour)
		got := Occurrences(c, loc, from, to)
		if again := Occurrences(c, loc, from, to); !reflect.DeepEqual(got, again) {
			t.Fatalf("case %d: not deterministic", i)
		}
		for j, v := range got {
			if v.Before(from) || v.After(to) || v.Before(start) || (endOn != "" && !v.Before(endBound)) {
				t.Fatalf("case %d: %v outside its window", i, v)
			}
			if j > 0 && !v.After(got[j-1]) {
				t.Fatalf("case %d: not strictly increasing at %d", i, j)
			}
			if s.Kind == KindEvery && v.Sub(start)%(time.Duration(s.EveryHours)*time.Hour) != 0 {
				t.Fatalf("case %d: %v off the grid", i, v)
			}
			if s.Kind == KindTimes {
				if lv := v.In(loc); !hasClockAtOrJustAfter(s.Times, lv) {
					t.Fatalf("case %d: %s matches no time", i, lv.Format(instantLayout))
				}
			}
		}
		prev := from.Add(-time.Nanosecond)
		for j := 0; ; j++ {
			n, ok := Next(c, loc, prev)
			if !ok || n.After(to) {
				if j != len(got) {
					t.Fatalf("case %d: Next walked %d, Occurrences %d", i, j, len(got))
				}
				break
			}
			if j >= len(got) || !n.Equal(got[j]) {
				t.Fatalf("case %d: Next step %d = %v, Occurrences disagree", i, j, n)
			}
			prev = n
		}
	}
}

// A resolved time is the wall time itself, or the end of the gap that swallowed it (a gap is at most an hour).
func hasClockAtOrJustAfter(hhmm []string, lv time.Time) bool {
	got := lv.Hour()*60 + lv.Minute()
	for _, v := range hhmm {
		want := int(v[0]-'0')*600 + int(v[1]-'0')*60 + int(v[3]-'0')*10 + int(v[4]-'0')
		if d := (got - want + 1440) % 1440; d <= 60 {
			return true
		}
	}
	return false
}
