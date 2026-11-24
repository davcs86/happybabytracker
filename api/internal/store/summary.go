package store

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/kidtime"
)

// DaySummary is one kid-local day's totals; both volume columns are summed independently.
type DaySummary struct {
	Date    string
	Bottles struct{ Count, ML, FlOzHundredths int64 }
	Nursing struct{ Count, Seconds int64 }
	Pumping struct{ Count, ML, FlOzHundredths int64 }
	Diapers struct{ Total, Wet, Dirty, Both, Dry int64 }
	Sleep   struct{ Seconds int64 }
	Meals   struct{ Count int64 }
}

// Summaries returns one DaySummary per local date from from to to inclusive in loc, ErrSummaryRange past SummaryMaxDays
// days. Sessions count on the day they start, except sleep, which is clipped to each day and a running one at now.
func (s *Store) Summaries(ctx context.Context, kidID string, loc *time.Location, from, to, now time.Time) ([]DaySummary, error) {
	return summaries(ctx, s.r, kidID, loc, from, to, now, TimelineTypes)
}

// summaries is Summaries on q, running only the aggregates of the trackers named in types (TimelineTypes names).
func summaries(ctx context.Context, q querier, kidID string, loc *time.Location, from, to, now time.Time, types []string) ([]DaySummary, error) {
	last := to.In(loc).Format(time.DateOnly)
	y, m, d := from.In(loc).Date()
	var out []DaySummary
	var values []string
	var dayArgs []any
	for i := 0; ; i++ {
		start, end := kidtime.DayBounds(loc, time.Date(y, m, d+i, 12, 0, 0, 0, loc))
		date := start.Format(time.DateOnly)
		if date > last {
			break
		}
		if i == constants.SummaryMaxDays {
			return nil, ErrSummaryRange
		}
		out = append(out, DaySummary{Date: date})
		values = append(values, "(?, ?, ?)")
		dayArgs = append(dayArgs, i, start.Unix(), end.Unix())
	}
	if len(out) == 0 {
		return out, nil
	}
	days := `WITH days(i, s, e) AS (VALUES ` + strings.Join(values, ", ") + `) `
	queries := []struct {
		typ, aggs, join string
		args            []any
		fill            func(*DaySummary, []int64)
	}{
		{"bottle", `COUNT(*), SUM(t.volume_ml), SUM(t.volume_floz_h)`, `bottle_feeds t ON t.kid_id = ? AND t.at >= d.s AND t.at < d.e`, nil,
			func(v *DaySummary, n []int64) {
				v.Bottles.Count, v.Bottles.ML, v.Bottles.FlOzHundredths = n[0], n[1], n[2]
			}},
		{"nursing", `COUNT(*), SUM(t.left_seconds + t.right_seconds)`, `nursing_sessions t ON t.kid_id = ? AND t.started_at >= d.s AND t.started_at < d.e`, nil,
			func(v *DaySummary, n []int64) { v.Nursing.Count, v.Nursing.Seconds = n[0], n[1] }},
		{"pumping", `COUNT(*), SUM(t.volume_ml), SUM(t.volume_floz_h)`, `pumping_sessions t ON t.kid_id = ? AND t.started_at >= d.s AND t.started_at < d.e`, nil,
			func(v *DaySummary, n []int64) {
				v.Pumping.Count, v.Pumping.ML, v.Pumping.FlOzHundredths = n[0], n[1], n[2]
			}},
		{"diaper", `COUNT(*), SUM(t.kind = 'wet'), SUM(t.kind = 'dirty'), SUM(t.kind = 'both'), SUM(t.kind = 'dry')`,
			`diapers t ON t.kid_id = ? AND t.at >= d.s AND t.at < d.e`, nil,
			func(v *DaySummary, n []int64) {
				v.Diapers.Total, v.Diapers.Wet, v.Diapers.Dirty, v.Diapers.Both, v.Diapers.Dry = n[0], n[1], n[2], n[3], n[4]
			}},
		{"sleep", `SUM(MAX(0, MIN(COALESCE(t.ended_at, ?), d.e) - MAX(t.started_at, d.s)))`,
			`sleep_sessions t ON t.kid_id = ? AND t.started_at < d.e AND (t.ended_at IS NULL OR t.ended_at > d.s)`, []any{now.Unix()},
			func(v *DaySummary, n []int64) { v.Sleep.Seconds = n[0] }},
		{"meal", `COUNT(*)`, `meals t ON t.kid_id = ? AND t.at >= d.s AND t.at < d.e`, nil,
			func(v *DaySummary, n []int64) { v.Meals.Count = n[0] }},
	}
	for _, agg := range queries {
		if !slices.Contains(types, agg.typ) {
			continue
		}
		args := append(append(append([]any{}, dayArgs...), agg.args...), kidID)
		if err := bucketed(ctx, q, days+`SELECT d.i, `+agg.aggs+` FROM days d JOIN `+agg.join+` GROUP BY d.i`, args, out, agg.fill); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// bucketed runs query, whose rows are a day index then int64 aggregates, and fills out[index] with them.
func bucketed(ctx context.Context, q querier, query string, args []any, out []DaySummary, fill func(*DaySummary, []int64)) error {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return err
	}
	vals := make([]int64, len(cols)-1)
	var i int
	dest := []any{&i}
	for k := range vals {
		dest = append(dest, &vals[k])
	}
	for rows.Next() {
		if err := rows.Scan(dest...); err != nil {
			return err
		}
		fill(&out[i], vals)
	}
	return rows.Err()
}
