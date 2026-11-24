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
	Counts          map[string]int            // per requested type, "medication" and "medication_dose": every row in the range
	Days            []DaySummary              // nil when no type is requested
	Growth          []model.Growth
	Doses           []model.VaccineDose
	Administrations []model.Administration
	Dismissals      map[string]ExportDismissal // per dismissed line of the kid's schema
	Milestones      []model.Milestone
	Medications     []model.Medication     // Open value 7 (R1), oldest start first, at most rowCap+1; no last dose, next due or count
	MedicationDoses []model.MedicationDose // given in the range, newest first by (GivenAt, ID), at most rowCap+1
}

// ExportDismissal is a dismissal's reason and when it was made.
type ExportDismissal struct {
	Reason    string
	CreatedAt time.Time
}

// exportTables names each exportable type's table and time column, both code constants.
var exportTables = map[string]struct{ table, timeCol string }{
	"bottle": {"bottle_feeds", "at"}, "nursing": {"nursing_sessions", "started_at"}, "pumping": {"pumping_sessions", "started_at"},
	"sleep": {"sleep_sessions", "started_at"}, "diaper": {"diapers", "at"}, "meal": {"meals", "at"}, "symptom": {"symptoms", "at"},
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
	case "symptom":
		rows, err := listEntries(ctx, q, `SELECT `+symptomColumns+` FROM symptoms`, "at", kidID, lq, scanSymptom)
		return timelineItems(typ, rows, err, func(v model.Symptom) (time.Time, string) { return v.At, v.ID })
	}
	return nil, errUnknownTimelineType
}

// exportCoursesWhere is Open value 7 (R1): a course whose kid-local span touches the range, or with a dose in it.
const exportCoursesWhere = ` FROM medications m WHERE m.kid_id = ? AND ((m.start_at < ? AND (m.end_on IS NULL OR m.end_on >= ?))
	OR EXISTS (SELECT 1 FROM medication_doses d WHERE d.medication_id = m.id AND d.given_at >= ? AND d.given_at < ?))`

func exportMedication(ctx context.Context, q querier, snap *ExportSnapshot, kidID, fromDate string, lq ListQuery) error {
	args := []any{kidID, lq.To.Unix(), fromDate, lq.From.Unix(), lq.To.Unix()}
	snap.Medications = []model.Medication{}
	err := eachRow(ctx, q, `SELECT `+prefixed("m.", medicationColumns)+exportCoursesWhere+` ORDER BY m.start_at, m.id LIMIT ?`,
		append(slices.Clone(args), lq.Limit), func(row rowScanner) error {
			v, err := scanMedication(row)
			snap.Medications = append(snap.Medications, v)
			return err
		})
	if err != nil {
		return err
	}
	courses := len(snap.Medications)
	if courses >= lq.Limit {
		if err := q.QueryRowContext(ctx, `SELECT COUNT(*)`+exportCoursesWhere, args...).Scan(&courses); err != nil {
			return err
		}
	}
	snap.Counts["medication"] = courses
	if snap.MedicationDoses, err = listEntries(ctx, q, `SELECT `+medicationDoseColumns+` FROM medication_doses`, "given_at", kidID, lq, scanMedicationDose); err != nil {
		return err
	}
	doses := len(snap.MedicationDoses)
	if doses >= lq.Limit {
		err = q.QueryRowContext(ctx, `SELECT COUNT(*) FROM medication_doses WHERE kid_id = ? AND given_at >= ? AND given_at < ?`,
			kidID, lq.From.Unix(), lq.To.Unix()).Scan(&doses)
	}
	snap.Counts["medication_dose"] = doses
	return err
}

// ExportSnapshot reads, in one transaction, only the requested modules and types, from the start of from to the end of to
// in loc (growth, vaccination and milestones: the whole record); at most rowCap+1 rows per list show overflow.
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
			snap.Administrations, snap.Dismissals = groupDoses(admin), map[string]ExportDismissal{}
			if snap.Kid.VaccineSchema != nil {
				if snap.Dismissals, err = exportDismissals(ctx, q, kidID, *snap.Kid.VaccineSchema); err != nil {
					return err
				}
			}
		}
		if slices.Contains(modules, "milestones") {
			if snap.Milestones, err = listMilestones(ctx, q, kidID); err != nil {
				return err
			}
		}
		if slices.Contains(modules, "medication") {
			return exportMedication(ctx, q, &snap, kidID, from.In(loc).Format(time.DateOnly), lq)
		}
		return nil
	})
	if err != nil {
		return ExportSnapshot{}, err
	}
	return snap, nil
}

func exportDismissals(ctx context.Context, q querier, kidID, schema string) (map[string]ExportDismissal, error) {
	rows, err := q.QueryContext(ctx, `SELECT line, reason, created_at FROM vaccine_dismissals WHERE kid_id = ? AND schema = ?`, kidID, schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]ExportDismissal{}
	for rows.Next() {
		var line, reason string
		var at int64
		if err := rows.Scan(&line, &reason, &at); err != nil {
			return nil, err
		}
		out[line] = ExportDismissal{Reason: reason, CreatedAt: time.Unix(at, 0)}
	}
	return out, rows.Err()
}
