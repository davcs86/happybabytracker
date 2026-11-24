package store

import (
	"context"
	"slices"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/kidtime"
	"github.com/davcs86/happybabytracker/api/internal/model"
)

// ExportSnapshot is what one export prints, read in one transaction; only the requested modules and types are read.
type ExportSnapshot struct {
	Kid             model.Kid
	Entries         map[string][]TimelineItem // per requested type, newest first by (At, ID), at most rowCap+1
	Counts          map[string]int            // per requested type, every row in the range
	Days            []DaySummary              // nil when no type is requested
	Growth          []model.Growth
	Doses           []model.VaccineDose
	Administrations []model.Administration
	Dismissals      map[string]string
	Milestones      []model.Milestone
}

// exportTables names each exportable type's table and time column, both code constants.
var exportTables = map[string]struct{ table, timeCol string }{
	"bottle": {"bottle_feeds", "at"}, "nursing": {"nursing_sessions", "started_at"}, "pumping": {"pumping_sessions", "started_at"},
	"sleep": {"sleep_sessions", "started_at"}, "diaper": {"diapers", "at"}, "meal": {"meals", "at"},
}

func exportEntries(ctx context.Context, q querier, typ, kidID string, lq ListQuery) ([]TimelineItem, error) {
	switch typ {
	case "bottle":
		rows, err := listEntries(ctx, q, `SELECT `+bottleColumns+` FROM bottle_feeds`, "at", kidID, lq, scanBottle)
		if err == nil {
			rows, err = withSources(ctx, q, rows)
		}
		return timelineItems(typ, rows, err, func(v model.BottleFeed) (time.Time, string) { return v.At, v.ID })
	case "nursing":
		rows, err := listEntries(ctx, q, `SELECT `+nursingColumns+` FROM nursing_sessions`, "started_at", kidID, lq, scanNursing)
		return timelineItems(typ, rows, err, func(v model.NursingSession) (time.Time, string) { return v.StartedAt, v.ID })
	case "pumping":
		rows, err := listEntries(ctx, q, `SELECT `+pumpingColumns+` FROM pumping_sessions`, "started_at", kidID, lq, scanPumping)
		if err == nil {
			rows, err = withSides(ctx, q, rows)
		}
		return timelineItems(typ, rows, err, func(v model.PumpingSession) (time.Time, string) { return v.StartedAt, v.ID })
	case "sleep":
		rows, err := listEntries(ctx, q, `SELECT `+sleepColumns+` FROM sleep_sessions`, "started_at", kidID, lq, scanSleep)
		return timelineItems(typ, rows, err, func(v model.SleepSession) (time.Time, string) { return v.StartedAt, v.ID })
	case "diaper":
		rows, err := listEntries(ctx, q, `SELECT `+diaperColumns+` FROM diapers`, "at", kidID, lq, scanDiaper)
		return timelineItems(typ, rows, err, func(v model.Diaper) (time.Time, string) { return v.At, v.ID })
	case "meal":
		rows, err := listEntries(ctx, q, `SELECT `+mealColumns+` FROM meals`, "at", kidID, lq, scanMeal)
		return timelineItems(typ, rows, err, func(v model.Meal) (time.Time, string) { return v.At, v.ID })
	}
	return nil, errUnknownTimelineType
}

// ExportSnapshot reads types' entries from the start of from to the end of to in loc, their day summaries and the whole
// record of each requested growth, vaccination or milestones module; at most rowCap+1 entries per type show overflow.
func (s *Store) ExportSnapshot(ctx context.Context, kidID string, modules []model.Module, types []string, loc *time.Location, from, to, now time.Time, rowCap int) (ExportSnapshot, error) {
	_, end := kidtime.DayBounds(loc, to)
	lq := ListQuery{From: from, To: end, Limit: rowCap + 1}
	snap := ExportSnapshot{Entries: map[string][]TimelineItem{}, Counts: map[string]int{}}
	err := s.inReadTx(ctx, func(q querier) error {
		var err error
		if snap.Kid, err = scanKid(q.QueryRowContext(ctx, `SELECT `+kidColumns+` FROM kids k WHERE k.id = ?`, kidID)); err != nil {
			return err
		}
		for _, typ := range types {
			list, err := exportEntries(ctx, q, typ, kidID, lq)
			if err != nil {
				return err
			}
			snap.Entries[typ], snap.Counts[typ] = list, len(list)
			if len(list) > rowCap {
				t, total := exportTables[typ], 0
				err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+t.table+` WHERE kid_id = ? AND `+t.timeCol+` >= ? AND `+t.timeCol+` < ?`,
					kidID, lq.From.Unix(), lq.To.Unix()).Scan(&total)
				if err != nil {
					return err
				}
				snap.Counts[typ] = total
			}
		}
		if len(types) > 0 {
			if snap.Days, err = summaries(ctx, q, kidID, loc, from, to, now, types); err != nil {
				return err
			}
		}
		if slices.Contains(modules, "growth") {
			if snap.Growth, err = listGrowth(ctx, q, kidID); err != nil {
				return err
			}
		}
		if slices.Contains(modules, "vaccination") {
			if snap.Doses, err = queryDoses(ctx, q, `WHERE kid_id = ? ORDER BY given_on, administration, antigen`, kidID); err != nil {
				return err
			}
			admin, err := queryDoses(ctx, q, `WHERE kid_id = ? ORDER BY given_on DESC, administration DESC, antigen`, kidID)
			if err != nil {
				return err
			}
			snap.Administrations, snap.Dismissals = groupDoses(admin), map[string]string{}
			if snap.Kid.VaccineSchema != nil {
				if snap.Dismissals, err = dismissals(ctx, q, kidID, *snap.Kid.VaccineSchema); err != nil {
					return err
				}
			}
		}
		if slices.Contains(modules, "milestones") {
			if snap.Milestones, err = listMilestones(ctx, q, kidID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return ExportSnapshot{}, err
	}
	return snap, nil
}
