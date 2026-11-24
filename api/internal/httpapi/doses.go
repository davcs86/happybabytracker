package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

type doseView struct {
	ID         string  `json:"id"`
	Medication string  `json:"medication"`
	GivenAt    string  `json:"given_at"`
	Amount     *string `json:"amount"`
	Note       *string `json:"note"`
}

func newDoseView(v model.MedicationDose) doseView {
	return doseView{ID: v.ID, Medication: v.MedicationID, GivenAt: instantString(v.GivenAt), Amount: v.Amount, Note: v.Note}
}

// doseInput is the body of POST and PATCH; on PATCH an absent key keeps the stored value, null clears amount and note,
// and medication may only repeat the stored course.
type doseInput struct {
	Medication *string        `json:"medication"`
	GivenAt    *time.Time     `json:"given_at"`
	Amount     optionalString `json:"amount"`
	Note       optionalString `json:"note"`
}

// mergeOnto applies in to v and names the first invalid field in the order given_at, amount, note; "" when valid.
func (in doseInput) mergeOnto(v model.MedicationDose, birth, now time.Time) (model.MedicationDose, string) {
	stored := v.GivenAt
	if in.GivenAt != nil {
		v.GivenAt = wholeSecond(*in.GivenAt)
	}
	if !changedOK(v.GivenAt, stored, birth, now) {
		return v, "given_at"
	}
	if in.Amount.Set {
		v.Amount = in.Amount.V
	}
	if v.Amount != nil {
		text := strings.TrimSpace(*v.Amount)
		if !medicationTextOK(text) {
			return v, "amount"
		}
		v.Amount = &text
	}
	if in.Note.Set {
		v.Note = in.Note.V
	}
	if !noteOK(v.Note) {
		return v, "note"
	}
	return v, ""
}

// mergePatch is mergeOnto, then refuses a medication other than the stored one.
func (in doseInput) mergePatch(stored model.MedicationDose, birth, now time.Time) (model.MedicationDose, string) {
	v, field := in.mergeOnto(stored, birth, now)
	if field == "" && in.Medication != nil && *in.Medication != stored.MedicationID {
		return v, "medication"
	}
	return v, field
}

// doseOK is entryOK that also answers a course that is not the kid's and the per-kid cap.
func doseOK(w http.ResponseWriter, err error) bool {
	switch {
	case errors.Is(err, store.ErrInvalidMedication):
		respondInvalid(w, "medication")
	case errors.Is(err, store.ErrConflict):
		respondError(w, http.StatusConflict, "limit")
	default:
		return entryOK(w, err)
	}
	return false
}

func (d Deps) listDoses(w http.ResponseWriter, r *http.Request) {
	q, bad := parseListQuery(r, d.Now())
	if bad != "" {
		respondInvalid(w, bad)
		return
	}
	medication := r.URL.Query().Get("medication")
	if r.URL.Query().Has("medication") && !cursorIDPattern.MatchString(medication) {
		respondInvalid(w, "medication")
		return
	}
	rows, err := d.Store.ListDoses(r.Context(), r.PathValue("kid"), medication, pageQuery(q))
	if !entryOK(w, err) {
		return
	}
	respondPage(w, "doses", rows, q.Limit, newDoseView, func(x model.MedicationDose) cursor { return entryCursor(x.GivenAt, x.ID) })
}

// createDose validates given_at, amount and note, then the store refuses a medication that is not the kid's.
func (d Deps) createDose(w http.ResponseWriter, r *http.Request) {
	var in doseInput
	if !decodeJSON(w, r, &in) {
		return
	}
	birth, ok := d.birthStart(w, r)
	if !ok {
		return
	}
	now := d.Now()
	v, field := in.mergeOnto(model.MedicationDose{GivenAt: wholeSecond(now)}, birth, now)
	if field != "" {
		respondInvalid(w, field)
		return
	}
	if in.Medication != nil {
		v.MedicationID = *in.Medication
	}
	v, err := d.Store.CreateDose(r.Context(), r.PathValue("kid"), PrincipalFrom(r.Context()).ParentID, v, now)
	if doseOK(w, err) {
		respondJSON(w, http.StatusCreated, newDoseView(v))
	}
}

func (d Deps) getDose(w http.ResponseWriter, r *http.Request) {
	getEntry(w, r, d.Store.Dose, newDoseView)
}

// updateDose validates the stored dose with the patch applied, never the patch alone.
func (d Deps) updateDose(w http.ResponseWriter, r *http.Request) {
	patchEntry(d, w, r, d.Store.Dose, doseInput.mergePatch, d.Store.UpdateDose, newDoseView)
}

func (d Deps) deleteDose(w http.ResponseWriter, r *http.Request) {
	deleteEntry(w, r, d.Store.DeleteDose)
}
