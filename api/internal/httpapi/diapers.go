package httpapi

import (
	"net/http"
	"slices"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
)

type diaperView struct {
	ID   string  `json:"id"`
	At   string  `json:"at"`
	Kind string  `json:"kind"`
	Note *string `json:"note"`
}

func newDiaperView(d model.Diaper) diaperView {
	return diaperView{ID: d.ID, At: instantString(d.At), Kind: d.Kind, Note: d.Note}
}

// diaperInput is the body of POST and PATCH; on PATCH an absent key keeps the stored value.
type diaperInput struct {
	At   *time.Time     `json:"at"`
	Kind *string        `json:"kind"`
	Note optionalString `json:"note"`
}

// mergeOnto applies in to d and names the first invalid field in the order at, kind, note; "" when valid.
func (in diaperInput) mergeOnto(d model.Diaper, birth, now time.Time) (model.Diaper, string) {
	if in.At != nil {
		d.At = wholeSecond(*in.At)
	}
	if !instantOK(d.At, birth, now) {
		return d, "at"
	}
	if in.Kind != nil {
		d.Kind = *in.Kind
	}
	if !slices.Contains(model.DiaperKinds, d.Kind) {
		return d, "kind"
	}
	if in.Note.Set {
		d.Note = in.Note.V
	}
	if !noteOK(d.Note) {
		return d, "note"
	}
	return d, ""
}

func (d Deps) listDiapers(w http.ResponseWriter, r *http.Request) {
	q, bad := parseListQuery(r, d.Now())
	if bad != "" {
		respondInvalid(w, bad)
		return
	}
	rows, err := d.Store.ListDiapers(r.Context(), r.PathValue("kid"), pageQuery(q))
	if !entryOK(w, err) {
		return
	}
	respondPage(w, "diapers", rows, q.Limit, newDiaperView, func(x model.Diaper) cursor { return entryCursor(x.At, x.ID) })
}

func (d Deps) createDiaper(w http.ResponseWriter, r *http.Request) {
	var in diaperInput
	if !decodeJSON(w, r, &in) {
		return
	}
	birth, ok := d.birthStart(w, r)
	if !ok {
		return
	}
	now := d.Now()
	x, field := in.mergeOnto(model.Diaper{At: wholeSecond(now)}, birth, now)
	if field != "" {
		respondInvalid(w, field)
		return
	}
	x, err := d.Store.CreateDiaper(r.Context(), r.PathValue("kid"), PrincipalFrom(r.Context()).ParentID, x, now)
	if entryOK(w, err) {
		respondJSON(w, http.StatusCreated, newDiaperView(x))
	}
}

func (d Deps) getDiaper(w http.ResponseWriter, r *http.Request) {
	getEntry(w, r, d.Store.Diaper, newDiaperView)
}

// updateDiaper validates the stored diaper with the patch applied, never the patch alone.
func (d Deps) updateDiaper(w http.ResponseWriter, r *http.Request) {
	patchEntry(d, w, r, d.Store.Diaper, diaperInput.mergeOnto, d.Store.UpdateDiaper, newDiaperView)
}

func (d Deps) deleteDiaper(w http.ResponseWriter, r *http.Request) {
	deleteEntry(w, r, d.Store.DeleteDiaper)
}
