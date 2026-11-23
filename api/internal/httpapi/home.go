package httpapi

import (
	"net/http"
	"slices"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/kidtime"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

var bandNames = map[kidtime.Band]string{kidtime.Band0to11: "0-11", kidtime.Band12to23: "12-23", kidtime.Band24Plus: "24+"}

type runningView struct {
	Sleep   *sleepView   `json:"sleep"`
	Nursing *nursingView `json:"nursing"`
}

const vaccinationModule model.Module = "vaccination"

// dueKindVaccine is the Kind of a due entry that stands for a vaccine visit.
const dueKindVaccine = "vaccine"

type dueView struct {
	Kind   string  `json:"kind"`
	From   ageJSON `json:"from"`
	Status string  `json:"status"`
	Count  int     `json:"count"`
}

type homeView struct {
	AgeMonths          int            `json:"age_months"`
	CorrectedBirthDate *string        `json:"corrected_birth_date"`
	Band               string         `json:"band"`
	Modules            []moduleView   `json:"modules"`
	Running            runningView    `json:"running"`
	Today              daySummaryView `json:"today"`
	Due                []dueView      `json:"due"`
}

// vaccineDue lists the vaccine visits that are due or overdue on today; empty when the module is off, no schedule
// is chosen or the chosen one is no longer shipped. It answers 503 only when a schedule would be judged.
func (d Deps) vaccineDue(w http.ResponseWriter, r *http.Request, kid model.Kid, states []store.ModuleState, loc *time.Location, now time.Time) ([]dueView, bool) {
	out := []dueView{}
	on := slices.ContainsFunc(states, func(s store.ModuleState) bool { return s.Module == vaccinationModule && s.Enabled })
	if !on || kid.VaccineSchema == nil {
		return out, true
	}
	if !schedulesLoaded(w) {
		return nil, false
	}
	s, ok := kidSchema(kid)
	if !ok {
		return out, true
	}
	e, _, _, err := d.evaluateKid(r.Context(), kid, &s, loc, now)
	if err != nil {
		respondInternal(w, err)
		return nil, false
	}
	for _, g := range e.Due() {
		out = append(out, dueView{Kind: dueKindVaccine, From: newAgeJSON(g.From), Status: string(g.Status), Count: g.Count})
	}
	return out, true
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
	if v.Due, ok = d.vaccineDue(w, r, kid, km.States, loc, now); !ok {
		return
	}
	respondJSON(w, http.StatusOK, v)
}
