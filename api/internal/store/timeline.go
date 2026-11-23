package store

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
)

// TimelineTypes names every tracker the timeline merges, as the wire spells them.
var TimelineTypes = []string{"bottle", "nursing", "pumping", "sleep", "diaper", "meal", "symptom"}

var errUnknownTimelineType = errors.New("store: unknown timeline type")

// TimelineItem is one tracker row; At is the row's time column and Entry the model value (model.BottleFeed, ...).
type TimelineItem struct {
	Type  string
	At    time.Time
	ID    string
	Entry any
}

func timelineItems[T any](typ string, rows []T, err error, key func(T) (time.Time, string)) ([]TimelineItem, error) {
	if err != nil {
		return nil, err
	}
	out := make([]TimelineItem, len(rows))
	for i, row := range rows {
		at, id := key(row)
		out[i] = TimelineItem{Type: typ, At: at, ID: id, Entry: row}
	}
	return out, nil
}

func (s *Store) timelineOf(ctx context.Context, typ, kidID string, q ListQuery) ([]TimelineItem, error) {
	switch typ {
	case "bottle":
		rows, err := s.ListBottles(ctx, kidID, q)
		return timelineItems(typ, rows, err, func(v model.BottleFeed) (time.Time, string) { return v.At, v.ID })
	case "nursing":
		rows, err := s.ListNursing(ctx, kidID, q)
		return timelineItems(typ, rows, err, func(v model.NursingSession) (time.Time, string) { return v.StartedAt, v.ID })
	case "pumping":
		rows, err := s.ListPumping(ctx, kidID, q)
		return timelineItems(typ, rows, err, func(v model.PumpingSession) (time.Time, string) { return v.StartedAt, v.ID })
	case "sleep":
		rows, err := s.ListSleep(ctx, kidID, q)
		return timelineItems(typ, rows, err, func(v model.SleepSession) (time.Time, string) { return v.StartedAt, v.ID })
	case "diaper":
		rows, err := s.ListDiapers(ctx, kidID, q)
		return timelineItems(typ, rows, err, func(v model.Diaper) (time.Time, string) { return v.At, v.ID })
	case "meal":
		rows, err := s.ListMeals(ctx, kidID, q)
		return timelineItems(typ, rows, err, func(v model.Meal) (time.Time, string) { return v.At, v.ID })
	case "symptom":
		rows, err := s.ListSymptoms(ctx, kidID, q)
		return timelineItems(typ, rows, err, func(v model.Symptom) (time.Time, string) { return v.At, v.ID })
	}
	return nil, errUnknownTimelineType
}

// Timeline merges the kid's rows of types (distinct TimelineTypes names) within q, newest first by (At, ID), and
// keeps q.Limit of them; each tracker contributes at most q.Limit rows, so the load is bounded.
func (s *Store) Timeline(ctx context.Context, kidID string, types []string, q ListQuery) ([]TimelineItem, error) {
	out := []TimelineItem{}
	for _, typ := range types {
		items, err := s.timelineOf(ctx, typ, kidID, q)
		if err != nil {
			return nil, err
		}
		out = append(out, items...)
	}
	slices.SortFunc(out, func(a, b TimelineItem) int {
		if c := b.At.Compare(a.At); c != 0 {
			return c
		}
		return cmp.Compare(b.ID, a.ID)
	})
	return out[:min(len(out), q.Limit)], nil
}
