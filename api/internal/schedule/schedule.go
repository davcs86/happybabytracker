// Package schedule lists a medication course's occurrences in the kid's zone; it is pure and never builds local
// midnight with time.Date: a wall time in a gap fires at the gap's end, one in an overlap at the earlier instant.
package schedule

import (
	"errors"
	"slices"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

type Kind string

const (
	KindNone  Kind = "none"
	KindTimes Kind = "times"
	KindEvery Kind = "every"
)

// Each error names the part that is wrong and never echoes the value.
var (
	ErrKind  = errors.New("schedule: kind")
	ErrTimes = errors.New("schedule: times")
	ErrHours = errors.New("schedule: hours")
)

type Schedule struct {
	Kind       Kind
	Times      []string
	EveryHours int
}

type Course struct {
	Schedule Schedule
	StartAt  time.Time
	EndOn    string
	Active   bool
}

const dateLayout = "2006-01-02"

// UTC offsets span -12h to +14h, so every instant showing a wall time lies within this of it read as UTC.
const offsetReach = 15 * time.Hour

// Bounds the zone walk over 2 × offsetReach; real zones change at most twice in that span.
const maxZonesInReach = 8

func Valid(s Schedule) error {
	switch s.Kind {
	case KindNone:
		if len(s.Times) != 0 {
			return ErrTimes
		}
		if s.EveryHours != 0 {
			return ErrHours
		}
	case KindTimes:
		if s.EveryHours != 0 {
			return ErrHours
		}
		if len(s.Times) == 0 || len(s.Times) > constants.MaxScheduleTimes {
			return ErrTimes
		}
		seen := make(map[int]bool, len(s.Times))
		for _, v := range s.Times {
			m, ok := clockMinutes(v)
			if !ok || seen[m] {
				return ErrTimes
			}
			seen[m] = true
		}
	case KindEvery:
		if len(s.Times) != 0 {
			return ErrTimes
		}
		if s.EveryHours < constants.MinEveryHours || s.EveryHours > constants.MaxEveryHours {
			return ErrHours
		}
	default:
		return ErrKind
	}
	return nil
}

// clockMinutes reads strict "HH:MM" (two ASCII digits each, 00-23 and 00-59); time.Parse accepts "7:05".
func clockMinutes(v string) (int, bool) {
	if len(v) != 5 || v[2] != ':' {
		return 0, false
	}
	for _, i := range []int{0, 1, 3, 4} {
		if v[i] < '0' || v[i] > '9' {
			return 0, false
		}
	}
	h := int(v[0]-'0')*10 + int(v[1]-'0')
	m := int(v[3]-'0')*10 + int(v[4]-'0')
	if h > 23 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}

// Occurrences lists the course's occurrences with from ≤ t ≤ to, ascending, at most MaxScheduleOccurrences.
func Occurrences(c Course, loc *time.Location, from, to time.Time) []time.Time {
	var out []time.Time
	if to.Before(from) {
		return nil
	}
	walk(c, loc, from, func(t time.Time) bool {
		if t.After(to) {
			return false
		}
		out = append(out, t)
		return len(out) < constants.MaxScheduleOccurrences
	})
	return out
}

// Next is the course's first occurrence strictly after after.
func Next(c Course, loc *time.Location, after time.Time) (time.Time, bool) {
	var next time.Time
	found := false
	walk(c, loc, after.Add(time.Nanosecond), func(t time.Time) bool {
		next, found = t, true
		return false
	})
	return next, found
}

// DayEnd is the first valid instant of the kid-local day after t's.
func DayEnd(loc *time.Location, t time.Time) time.Time {
	y, m, d := t.In(loc).Date()
	return resolve(loc, time.Date(y, m, d+1, 0, 0, 0, 0, time.UTC))
}

// walk yields, ascending and without repeats, each occurrence ≥ from until yield returns false.
func walk(c Course, loc *time.Location, from time.Time, yield func(time.Time) bool) {
	if !c.Active || Valid(c.Schedule) != nil {
		return
	}
	end, ok := endBound(c, loc)
	if !ok {
		return
	}
	if from.Before(c.StartAt) {
		from = c.StartAt
	}
	switch c.Schedule.Kind {
	case KindEvery:
		step := time.Duration(c.Schedule.EveryHours) * time.Hour
		k := (from.Sub(c.StartAt) + step - 1) / step
		for i := 0; i < constants.MaxScheduleOccurrences; i++ {
			t := c.StartAt.Add((k + time.Duration(i)) * step).UTC()
			if (!end.IsZero() && !t.Before(end)) || !yield(t) {
				return
			}
		}
	case KindTimes:
		clocks := make([]int, len(c.Schedule.Times))
		for i, v := range c.Schedule.Times {
			clocks[i], _ = clockMinutes(v)
		}
		slices.Sort(clocks)
		// The day before from's: a time late in it can resolve into from's day.
		y, m, d := from.In(loc).Date()
		day := time.Date(y, m, d-1, 0, 0, 0, 0, time.UTC)
		var last time.Time
		for i := 0; i < constants.MaxScheduleOccurrences+2; i++ {
			for _, cm := range clocks {
				t := resolve(loc, day.Add(time.Duration(cm)*time.Minute))
				if !t.After(last) || t.Before(from) {
					continue
				}
				if (!end.IsZero() && !t.Before(end)) || !yield(t) {
					return
				}
				last = t
			}
			day = day.AddDate(0, 0, 1)
		}
	}
}

// endBound is the start of the kid-local day after EndOn (zero when open-ended); false for a malformed EndOn.
func endBound(c Course, loc *time.Location) (time.Time, bool) {
	if c.EndOn == "" {
		return time.Time{}, true
	}
	d, err := time.Parse(dateLayout, c.EndOn)
	if err != nil || d.Format(dateLayout) != c.EndOn {
		return time.Time{}, false
	}
	return resolve(loc, d.AddDate(0, 0, 1)), true
}

type zone struct {
	start, end time.Time
	offset     time.Duration
}

// resolve turns a wall time (its fields read in UTC) into the instant it names in loc: the earlier of two
// in an overlap, the gap's end in a gap.
func resolve(loc *time.Location, wall time.Time) time.Time {
	zones := zonesAround(loc, wall)
	var best time.Time
	for _, z := range zones {
		t := wall.Add(-z.offset)
		if inZone(z, t) && (best.IsZero() || t.Before(best)) {
			best = t
		}
	}
	if !best.IsZero() {
		return best
	}
	for i := 1; i < len(zones); i++ {
		at := zones[i].start
		if !at.Add(zones[i-1].offset).After(wall) && at.Add(zones[i].offset).After(wall) {
			return at.UTC()
		}
	}
	return wall.Add(-zones[0].offset)
}

func inZone(z zone, t time.Time) bool {
	return (z.start.IsZero() || !t.Before(z.start)) && (z.end.IsZero() || t.Before(z.end))
}

// zonesAround lists, in order, every zone loc is in at some instant within offsetReach of wall.
func zonesAround(loc *time.Location, wall time.Time) []zone {
	limit := wall.Add(offsetReach)
	t := wall.Add(-offsetReach).In(loc)
	var zones []zone
	for range maxZonesInReach {
		_, off := t.Zone()
		start, end := t.ZoneBounds()
		zones = append(zones, zone{start: start, end: end, offset: time.Duration(off) * time.Second})
		if end.IsZero() || end.After(limit) {
			break
		}
		t = end.In(loc)
	}
	return zones
}
