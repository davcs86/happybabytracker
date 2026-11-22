package store

import (
	"errors"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/kidtime"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/units"
)

func mustLoc(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func mustDate(t *testing.T, s string, loc *time.Location) time.Time {
	t.Helper()
	d, err := kidtime.ParseDate(s, loc)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func mustSummaries(t *testing.T, s *Store, kidID string, loc *time.Location, from, to string, now time.Time) []DaySummary {
	t.Helper()
	days, err := s.Summaries(ctx, kidID, loc, mustDate(t, from, loc), mustDate(t, to, loc), now)
	if err != nil {
		t.Fatal(err)
	}
	return days
}

func secs(d time.Duration) int64 { return int64(d / time.Second) }

func TestSummaryAcrossDSTDay(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	madrid := mustLoc(t, "Europe/Madrid")
	start := time.Date(2026, 3, 28, 22, 0, 0, 0, madrid)
	end := time.Date(2026, 3, 29, 8, 0, 0, 0, madrid)
	mustSleep(t, s, k.ID, ana.ID, model.SleepSession{StartedAt: start, EndedAt: &end})

	_, end28 := kidtime.DayBounds(madrid, start)
	start29, end29 := kidtime.DayBounds(madrid, end)
	if end29.Sub(start29) != 23*time.Hour {
		t.Fatalf("2026-03-29 in Madrid lasts %v, want the 23 h DST day", end29.Sub(start29))
	}
	days := mustSummaries(t, s, k.ID, madrid, "2026-03-28", "2026-03-29", t0)
	if len(days) != 2 || days[0].Date != "2026-03-28" || days[1].Date != "2026-03-29" {
		t.Fatalf("days = %+v", days)
	}
	if got, want := days[0].Sleep.Seconds, secs(end28.Sub(start)); got != want || want != secs(2*time.Hour) {
		t.Errorf("28th sleep = %d s, want %d", got, want)
	}
	if got, want := days[1].Sleep.Seconds, secs(end.Sub(start29)); got != want || want != secs(7*time.Hour) {
		t.Errorf("29th sleep = %d s, want %d", got, want)
	}
}

func TestSummarySleepClippedAtMidnightAndNow(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	madrid := mustLoc(t, "Europe/Madrid")
	start := time.Date(2026, 11, 18, 23, 0, 0, 0, madrid)
	end := time.Date(2026, 11, 19, 1, 30, 0, 0, madrid)
	mustSleep(t, s, k.ID, ana.ID, model.SleepSession{StartedAt: start, EndedAt: &end})
	runningFrom := time.Date(2026, 11, 20, 10, 0, 0, 0, madrid)
	mustSleep(t, s, k.ID, ana.ID, model.SleepSession{StartedAt: runningFrom})
	now := time.Date(2026, 11, 20, 13, 0, 0, 0, madrid)

	_, midnight := kidtime.DayBounds(madrid, start)
	days := mustSummaries(t, s, k.ID, madrid, "2026-11-18", "2026-11-21", now)
	want := []int64{secs(midnight.Sub(start)), secs(end.Sub(midnight)), secs(now.Sub(runningFrom)), 0}
	if len(days) != len(want) {
		t.Fatalf("days = %+v", days)
	}
	for i, w := range want {
		if days[i].Sleep.Seconds != w {
			t.Errorf("%s sleep = %d s, want %d", days[i].Date, days[i].Sleep.Seconds, w)
		}
	}
	if want[0] != secs(time.Hour) || want[1] != secs(90*time.Minute) || want[2] != secs(3*time.Hour) {
		t.Fatalf("fixture drifted: %v", want)
	}
	earlier := mustSummaries(t, s, k.ID, madrid, "2026-11-20", "2026-11-20", runningFrom.Add(-time.Minute))
	if earlier[0].Sleep.Seconds != 0 {
		t.Errorf("running sleep that starts after now counts %d s", earlier[0].Sleep.Seconds)
	}
}

func TestSummaryCountsAndSums(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	other := mustKid(t, s, ana.ID, "Teo")
	madrid := mustLoc(t, "Europe/Madrid")
	day := time.Date(2026, 11, 18, 9, 0, 0, 0, madrid)

	ml := mustBottle(t, s, k.ID, ana.ID, newBottle(t, day, model.ContentFormula))
	oz := newBottle(t, day.Add(time.Hour), model.ContentFormula)
	oz.Volume = mustVolume(t, units.UnitFlOz, 400)
	oz = mustBottle(t, s, k.ID, ana.ID, oz)
	for _, kind := range []string{"wet", "wet", "dirty", "both", "dry"} {
		mustDiaper(t, s, k.ID, ana.ID, model.Diaper{At: day, Kind: kind})
	}
	mustMeal(t, s, k.ID, ana.ID, model.Meal{At: day, What: "pear"})
	p1 := mustPumping(t, s, k.ID, ana.ID, newPumping(t, day))
	p2 := newPumping(t, day.Add(time.Hour))
	p2.Volume = mustVolume(t, units.UnitFlOz, 250)
	p2 = mustPumping(t, s, k.ID, ana.ID, p2)
	// A finished session that starts on the 18th and ends on the 19th counts on the 18th.
	mustNursing(t, s, k.ID, ana.ID, finishedNursing(day.Add(14*time.Hour+50*time.Minute), day.Add(15*time.Hour+20*time.Minute), 600, 900))
	running := mustNursing(t, s, k.ID, ana.ID, runningNursing(day.Add(48*time.Hour), "left"))
	running, err := s.SwitchNursingSide(ctx, k.ID, ana.ID, running.ID, "right", running.StartedAt.Add(5*time.Minute), t0)
	if err != nil {
		t.Fatal(err)
	}

	mustBottle(t, s, other.ID, ana.ID, newBottle(t, day, model.ContentFormula))
	mustDiaper(t, s, other.ID, ana.ID, model.Diaper{At: day, Kind: "wet"})
	mustMeal(t, s, other.ID, ana.ID, model.Meal{At: day, What: "apple"})
	mustPumping(t, s, other.ID, ana.ID, newPumping(t, day))
	mustNursing(t, s, other.ID, ana.ID, finishedNursing(day, day.Add(time.Hour), 60, 60))
	otherEnd := day.Add(time.Hour)
	mustSleep(t, s, other.ID, ana.ID, model.SleepSession{StartedAt: day, EndedAt: &otherEnd})

	days := mustSummaries(t, s, k.ID, madrid, "2026-11-17", "2026-11-20", day.Add(72*time.Hour))
	if len(days) != 4 {
		t.Fatalf("days = %+v", days)
	}
	var want [4]DaySummary
	for i, date := range []string{"2026-11-17", "2026-11-18", "2026-11-19", "2026-11-20"} {
		want[i].Date = date
	}
	d := &want[1]
	d.Bottles.Count, d.Bottles.ML, d.Bottles.FlOzHundredths = 2, ml.Volume.ML+oz.Volume.ML, ml.Volume.FlOzHundredths+oz.Volume.FlOzHundredths
	d.Pumping.Count, d.Pumping.ML, d.Pumping.FlOzHundredths = 2, p1.Volume.ML+p2.Volume.ML, p1.Volume.FlOzHundredths+p2.Volume.FlOzHundredths
	d.Diapers.Total, d.Diapers.Wet, d.Diapers.Dirty, d.Diapers.Both, d.Diapers.Dry = 5, 2, 1, 1, 1
	d.Meals.Count = 1
	d.Nursing.Count, d.Nursing.Seconds = 1, 1500
	want[3].Nursing.Count, want[3].Nursing.Seconds = 1, int64(running.LeftSeconds+running.RightSeconds)
	if running.LeftSeconds != 300 {
		t.Fatalf("running session credited %d s", running.LeftSeconds)
	}
	for i := range want {
		if days[i] != want[i] {
			t.Errorf("day %d = %+v\n want %+v", i, days[i], want[i])
		}
	}
	if ml.Volume.FlOzHundredths == 0 || oz.Volume.ML == 0 {
		t.Fatalf("volumes lack a converted column: %+v %+v", ml.Volume, oz.Volume)
	}
}

func TestSummariesBoundTheirRange(t *testing.T) {
	s := openMem(t)
	ana := mustParent(t, s, "ana")
	k := mustKid(t, s, ana.ID, "Lía")
	madrid := mustLoc(t, "Europe/Madrid")
	from := mustDate(t, "2026-08-01", madrid)
	if days, err := s.Summaries(ctx, k.ID, madrid, from, from.AddDate(0, 0, constants.SummaryMaxDays-1), t0); err != nil || len(days) != constants.SummaryMaxDays {
		t.Errorf("%d days: %d, %v", constants.SummaryMaxDays, len(days), err)
	}
	if _, err := s.Summaries(ctx, k.ID, madrid, from, from.AddDate(0, 0, constants.SummaryMaxDays), t0); !errors.Is(err, ErrSummaryRange) {
		t.Errorf("%d days: %v, want ErrSummaryRange", constants.SummaryMaxDays+1, err)
	}
	if days := mustSummaries(t, s, k.ID, madrid, "2026-11-20", "2026-11-20", t0); len(days) != 1 {
		t.Errorf("one day: %d", len(days))
	}
}
