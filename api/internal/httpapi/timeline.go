package httpapi

import (
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

// defaultTimelineTypes is the trackers an absent types parameter selects.
func defaultTimelineTypes() []string { return slices.Clone(store.TimelineTypes) }

// timelineTypes reads types, one comma list of distinct TimelineTypes names; false when it is invalid.
func timelineTypes(v url.Values) ([]string, bool) {
	given, ok := v["types"]
	if !ok {
		return defaultTimelineTypes(), true
	}
	if len(given) != 1 {
		return nil, false
	}
	names := strings.Split(given[0], ",")
	for i, name := range names {
		if !slices.Contains(store.TimelineTypes, name) || slices.Contains(names[:i], name) {
			return nil, false
		}
	}
	return names, true
}

// newTimelineView is the tracker's own view with type and at (the row's time column) added.
func newTimelineView(it store.TimelineItem) any {
	at := instantString(it.At)
	switch e := it.Entry.(type) {
	case model.BottleFeed:
		return struct {
			Type string `json:"type"`
			At   string `json:"at"`
			bottleView
		}{it.Type, at, newBottleView(e)}
	case model.NursingSession:
		return struct {
			Type string `json:"type"`
			At   string `json:"at"`
			nursingView
		}{it.Type, at, newNursingView(e)}
	case model.PumpingSession:
		return struct {
			Type string `json:"type"`
			At   string `json:"at"`
			pumpingView
		}{it.Type, at, newPumpingView(e)}
	case model.SleepSession:
		return struct {
			Type string `json:"type"`
			At   string `json:"at"`
			sleepView
		}{it.Type, at, newSleepView(e)}
	case model.Diaper:
		return struct {
			Type string `json:"type"`
			At   string `json:"at"`
			diaperView
		}{it.Type, at, newDiaperView(e)}
	case model.Meal:
		return struct {
			Type string `json:"type"`
			At   string `json:"at"`
			mealView
		}{it.Type, at, newMealView(e)}
	}
	return struct {
		Type string `json:"type"`
		ID   string `json:"id"`
		At   string `json:"at"`
	}{it.Type, it.ID, at}
}

func (d Deps) timeline(w http.ResponseWriter, r *http.Request) {
	q, bad := parseListQuery(r, d.Now())
	if bad != "" {
		respondInvalid(w, bad)
		return
	}
	types, ok := timelineTypes(r.URL.Query())
	if !ok {
		respondInvalid(w, "types")
		return
	}
	items, err := d.Store.Timeline(r.Context(), r.PathValue("kid"), types, pageQuery(q))
	if !entryOK(w, err) {
		return
	}
	respondPage(w, "items", items, q.Limit, newTimelineView, func(it store.TimelineItem) cursor { return entryCursor(it.At, it.ID) })
}
