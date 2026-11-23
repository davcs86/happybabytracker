package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/kidtime"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/schedule"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

type scheduleView struct {
	Kind  schedule.Kind `json:"kind"`
	Times []string      `json:"times,omitempty"`
	Hours int           `json:"hours,omitempty"`
}

type lastDoseView struct {
	ID      string  `json:"id"`
	GivenAt string  `json:"given_at"`
	Amount  *string `json:"amount"`
}

type nextDueView struct {
	DueAt string `json:"due_at"`
}

type medicationView struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Dose      string        `json:"dose"`
	Schedule  scheduleView  `json:"schedule"`
	StartAt   string        `json:"start_at"`
	EndOn     *string       `json:"end_on"`
	Active    bool          `json:"active"`
	Note      *string       `json:"note"`
	LastDose  *lastDoseView `json:"last_dose"`
	NextDue   *nextDueView  `json:"next_due"`
	DoseCount int           `json:"dose_count"`
}

func newMedicationView(v model.Medication) medicationView {
	out := medicationView{ID: v.ID, Name: v.Name, Dose: v.Dose, StartAt: instantString(v.StartAt), EndOn: v.EndOn, Active: v.Active,
		Note: v.Note, DoseCount: v.DoseCount, Schedule: scheduleView{Kind: v.Kind, Times: v.Times, Hours: v.EveryHours}}
	if d := v.LastDose; d != nil {
		out.LastDose = &lastDoseView{ID: d.ID, GivenAt: instantString(d.GivenAt), Amount: d.Amount}
	}
	if v.NextDue != nil {
		out.NextDue = &nextDueView{DueAt: instantString(*v.NextDue)}
	}
	return out
}

type scheduleInput struct {
	Kind  string   `json:"kind"`
	Times []string `json:"times"`
	Hours *int     `json:"hours"`
}

// medicationInput is the body of POST and PATCH; on PATCH an absent key keeps the stored value and a schedule replaces it whole.
type medicationInput struct {
	Name     *string        `json:"name"`
	Dose     *string        `json:"dose"`
	Schedule *scheduleInput `json:"schedule"`
	StartAt  *time.Time     `json:"start_at"`
	EndOn    optionalString `json:"end_on"`
	Active   *bool          `json:"active"`
	Note     optionalString `json:"note"`
}

func medicationTextOK(s string) bool {
	n := utf8.RuneCountInString(s)
	return n >= 1 && n <= constants.MedicationTextMaxLen
}

// endOnOK is true for a real YYYY-MM-DD that is not before the kid-local date of start.
func endOnOK(endOn string, start time.Time, loc *time.Location) bool {
	d, err := time.Parse(time.DateOnly, endOn)
	return err == nil && d.Format(time.DateOnly) == endOn && endOn >= start.In(loc).Format(time.DateOnly)
}

// mergeOnto applies in to v and names the first invalid field in the order name, dose, schedule, start_at, end_on, active,
// note; "" when valid. start_at and end_on are checked only when changed, so a stored course stays editable.
func (in medicationInput) mergeOnto(v model.Medication, loc *time.Location, birth, now time.Time) (model.Medication, string) {
	if in.Name != nil {
		v.Name = strings.TrimSpace(*in.Name)
	}
	if !medicationTextOK(v.Name) {
		return v, "name"
	}
	if in.Dose != nil {
		v.Dose = strings.TrimSpace(*in.Dose)
	}
	if !medicationTextOK(v.Dose) {
		return v, "dose"
	}
	if s := in.Schedule; s != nil {
		v.Schedule = schedule.Schedule{Kind: schedule.Kind(s.Kind), Times: s.Times}
		if s.Hours != nil {
			v.EveryHours = *s.Hours
		}
	}
	if schedule.Valid(v.Schedule) != nil {
		return v, "schedule"
	}
	stored := v.StartAt
	if in.StartAt != nil {
		v.StartAt = wholeSecond(*in.StartAt)
	}
	startChanged := !v.StartAt.Equal(stored)
	if startChanged && v.StartAt.Before(birth) {
		return v, "start_at"
	}
	if in.EndOn.Set {
		v.EndOn = in.EndOn.V
	}
	if v.EndOn != nil && (in.EndOn.Set || startChanged) && !endOnOK(*v.EndOn, v.StartAt, loc) {
		return v, "end_on"
	}
	if in.Active != nil {
		v.Active = *in.Active
	}
	if in.Note.Set {
		v.Note = in.Note.V
	}
	if !noteOK(v.Note) {
		return v, "note"
	}
	return v, ""
}

// kidZoneBirth is the kid's time zone and the start of its birth date; it answers for a failed lookup, and an unusable
// zone is a 500 (the schedule engine must never get a nil zone).
func (d Deps) kidZoneBirth(w http.ResponseWriter, r *http.Request) (*time.Location, time.Time, bool) {
	kid, ok := d.loadKid(w, r)
	if !ok {
		return nil, time.Time{}, false
	}
	loc, zoneOK := kidLocation(kid.TimeZone)
	if !zoneOK {
		respondInternal(w, errKidZone)
		return nil, time.Time{}, false
	}
	birth, err := kidtime.ParseDate(kid.BirthDate, loc)
	if err != nil {
		respondInternal(w, err)
		return nil, time.Time{}, false
	}
	return loc, birth, true
}

// medicationOK is entryOK that also answers a refused schedule and the per-kid cap.
func medicationOK(w http.ResponseWriter, err error) bool {
	switch {
	case errors.Is(err, store.ErrInvalidSchedule):
		respondInvalid(w, "schedule")
	case errors.Is(err, store.ErrConflict):
		respondError(w, http.StatusConflict, "limit")
	default:
		return entryOK(w, err)
	}
	return false
}

func (d Deps) listMedications(w http.ResponseWriter, r *http.Request) {
	rows, err := d.Store.ListMedications(r.Context(), r.PathValue("kid"), d.Now())
	if !entryOK(w, err) {
		return
	}
	views := make([]medicationView, len(rows))
	for i, row := range rows {
		views[i] = newMedicationView(row)
	}
	respondJSON(w, http.StatusOK, struct {
		Medications []medicationView `json:"medications"`
	}{views})
}

func (d Deps) createMedication(w http.ResponseWriter, r *http.Request) {
	var in medicationInput
	if !decodeJSON(w, r, &in) {
		return
	}
	loc, birth, ok := d.kidZoneBirth(w, r)
	if !ok {
		return
	}
	now := d.Now()
	v, field := in.mergeOnto(model.Medication{StartAt: wholeSecond(now), Active: true}, loc, birth, now)
	if field != "" {
		respondInvalid(w, field)
		return
	}
	v, err := d.Store.CreateMedication(r.Context(), r.PathValue("kid"), PrincipalFrom(r.Context()).ParentID, v, now)
	if medicationOK(w, err) {
		respondJSON(w, http.StatusCreated, newMedicationView(v))
	}
}

func (d Deps) getMedication(w http.ResponseWriter, r *http.Request) {
	getEntry(w, r, func(ctx context.Context, kidID, id string) (model.Medication, error) {
		return d.Store.Medication(ctx, kidID, id, d.Now())
	}, newMedicationView)
}

// updateMedication validates the stored course with the patch applied, never the patch alone.
func (d Deps) updateMedication(w http.ResponseWriter, r *http.Request) {
	var in medicationInput
	if !decodeJSON(w, r, &in) {
		return
	}
	id, ok := entryID(w, r)
	if !ok {
		return
	}
	loc, birth, ok := d.kidZoneBirth(w, r)
	if !ok {
		return
	}
	kidID, now := r.PathValue("kid"), d.Now()
	stored, err := d.Store.Medication(r.Context(), kidID, id, now)
	if !entryOK(w, err) {
		return
	}
	v, field := in.mergeOnto(stored, loc, birth, now)
	if field != "" {
		respondInvalid(w, field)
		return
	}
	v, err = d.Store.UpdateMedication(r.Context(), kidID, PrincipalFrom(r.Context()).ParentID, v, now)
	if medicationOK(w, err) {
		respondJSON(w, http.StatusOK, newMedicationView(v))
	}
}

func (d Deps) deleteMedication(w http.ResponseWriter, r *http.Request) {
	deleteEntry(w, r, d.Store.DeleteMedication)
}
