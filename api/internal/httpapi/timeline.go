package httpapi

import (
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

// timelineModules maps each TimelineTypes name to the module that switches it.
var timelineModules = map[string]model.Module{
	"bottle": "bottles", "nursing": "nursing", "pumping": "pumping", "sleep": "sleep", "diaper": "diapers", "meal": "feeding",
}

// defaultTimelineTypes is the trackers an absent types parameter selects: those whose module is enabled.
func defaultTimelineTypes(states []store.ModuleState) []string {
	out := []string{}
	for _, typ := range store.TimelineTypes {
		if slices.ContainsFunc(states, func(st store.ModuleState) bool { return st.Module == timelineModules[typ] && st.Enabled }) {
			out = append(out, typ)
		}
	}
	return out
}

// timelineTypes reads types, one comma list of distinct TimelineTypes names; nil when absent, false when invalid.
func timelineTypes(v url.Values) ([]string, bool) {
	given, ok := v["types"]
	if !ok {
		return nil, true
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
	if types == nil {
		_, _, states, ok := d.kidModules(w, r, d.Now())
		if !ok {
			return
		}
		types = defaultTimelineTypes(states)
	}
	items, err := d.Store.Timeline(r.Context(), r.PathValue("kid"), types, pageQuery(q))
	if !entryOK(w, err) {
		return
	}
	respondPage(w, "items", items, q.Limit, newTimelineView, func(it store.TimelineItem) cursor { return entryCursor(it.At, it.ID) })
}
