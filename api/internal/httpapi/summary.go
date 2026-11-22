package httpapi

import (
	"net/http"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/kidtime"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

type daySummaryView struct {
	Date    string        `json:"date"`
	Bottles volumeTotals  `json:"bottles"`
	Nursing nursingTotals `json:"nursing"`
	Pumping volumeTotals  `json:"pumping"`
	Diapers diaperTotals  `json:"diapers"`
	Sleep   sleepTotals   `json:"sleep"`
	Meals   mealTotals    `json:"meals"`
}

type volumeTotals struct {
	Count          int64 `json:"count"`
	ML             int64 `json:"ml"`
	FlOzHundredths int64 `json:"floz_hundredths"`
}

type nursingTotals struct {
	Count   int64 `json:"count"`
	Seconds int64 `json:"seconds"`
}

type diaperTotals struct {
	Total int64 `json:"total"`
	Wet   int64 `json:"wet"`
	Dirty int64 `json:"dirty"`
	Both  int64 `json:"both"`
	Dry   int64 `json:"dry"`
}

type sleepTotals struct {
	Seconds int64 `json:"seconds"`
}

type mealTotals struct {
	Count int64 `json:"count"`
}

func newDaySummaryView(s store.DaySummary) daySummaryView {
	return daySummaryView{
		Date:    s.Date,
		Bottles: volumeTotals(s.Bottles),
		Nursing: nursingTotals(s.Nursing),
		Pumping: volumeTotals(s.Pumping),
		Diapers: diaperTotals(s.Diapers),
		Sleep:   sleepTotals(s.Sleep),
		Meals:   mealTotals(s.Meals),
	}
}

// summaryRange reads from and to as kid-local dates; the second result names the first invalid one, or "".
// to must not precede from (else from), and the range spans at most SummaryMaxDays days (else to).
func summaryRange(r *http.Request, loc *time.Location) (from, to time.Time, bad string) {
	v := r.URL.Query()
	from, err := kidtime.ParseDate(v.Get("from"), loc)
	if err != nil {
		return time.Time{}, time.Time{}, "from"
	}
	to, err = kidtime.ParseDate(v.Get("to"), loc)
	if err != nil {
		return time.Time{}, time.Time{}, "to"
	}
	fy, fm, fd := from.Date()
	ty, tm, td := to.Date()
	days := int(time.Date(ty, tm, td, 0, 0, 0, 0, time.UTC).Sub(time.Date(fy, fm, fd, 0, 0, 0, 0, time.UTC))/(24*time.Hour)) + 1
	switch {
	case days < 1:
		return time.Time{}, time.Time{}, "from"
	case days > constants.SummaryMaxDays:
		return time.Time{}, time.Time{}, "to"
	}
	return from, to, ""
}

func (d Deps) summary(w http.ResponseWriter, r *http.Request) {
	kid, ok := d.loadKid(w, r)
	if !ok {
		return
	}
	loc, zoneOK := kidLocation(kid.TimeZone)
	if !zoneOK {
		respondInternal(w, errKidZone)
		return
	}
	from, to, bad := summaryRange(r, loc)
	if bad != "" {
		respondInvalid(w, bad)
		return
	}
	days, err := d.Store.Summaries(r.Context(), kid.ID, loc, from, to, d.Now())
	if !entryOK(w, err) {
		return
	}
	views := make([]daySummaryView, len(days))
	for i, day := range days {
		views[i] = newDaySummaryView(day)
	}
	respondJSON(w, http.StatusOK, map[string]any{"days": views})
}
