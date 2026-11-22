package httpapi

import (
	"net/http"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/kidtime"
)

var bandNames = map[kidtime.Band]string{kidtime.Band0to11: "0-11", kidtime.Band12to23: "12-23", kidtime.Band24Plus: "24+"}

type runningView struct {
	Sleep   *sleepView   `json:"sleep"`
	Nursing *nursingView `json:"nursing"`
}

type homeView struct {
	AgeMonths          int            `json:"age_months"`
	CorrectedBirthDate *string        `json:"corrected_birth_date"`
	Band               string         `json:"band"`
	Modules            []moduleView   `json:"modules"`
	Running            runningView    `json:"running"`
	Today              daySummaryView `json:"today"`
}

func (d Deps) home(w http.ResponseWriter, r *http.Request) {
	now := d.Now()
	kid, loc, km, ok := d.kidModules(w, r, now)
	if !ok {
		return
	}
	v := homeView{AgeMonths: km.AgeMonths, Band: bandNames[km.Band], Modules: newModuleViews(km.States)}
	start, _ := kidtime.DayBounds(loc, now)
	due, corrected, err := kidtime.CorrectedBirthDate(kid.BirthDate, kid.GestationWeeks, kid.GestationDays, start.Format(time.DateOnly), loc)
	if err != nil {
		respondInternal(w, err)
		return
	}
	if corrected {
		v.CorrectedBirthDate = &due
	}
	sleep, err := d.Store.RunningSleep(r.Context(), kid.ID)
	if !entryOK(w, err) {
		return
	}
	if sleep != nil {
		view := newSleepView(*sleep)
		v.Running.Sleep = &view
	}
	nursing, err := d.Store.RunningNursing(r.Context(), kid.ID)
	if !entryOK(w, err) {
		return
	}
	if nursing != nil {
		view := newNursingView(*nursing)
		v.Running.Nursing = &view
	}
	days, err := d.Store.Summaries(r.Context(), kid.ID, loc, start, start, now)
	if !entryOK(w, err) {
		return
	}
	v.Today = newDaySummaryView(days[0])
	respondJSON(w, http.StatusOK, v)
}
