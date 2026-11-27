package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/growth"
	"github.com/davcs86/happybabytracker/api/internal/kidtime"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/store"
	"github.com/davcs86/happybabytracker/api/internal/units"
)

type weightView struct {
	Grams        int64      `json:"grams"`
	OzHundredths int64      `json:"oz_hundredths"`
	TypedUnit    units.Unit `json:"typed_unit"`
}

type lengthView struct {
	MM           int64      `json:"mm"`
	InHundredths int64      `json:"in_hundredths"`
	TypedUnit    units.Unit `json:"typed_unit"`
}

type ageView struct {
	Days          int  `json:"days"`
	CorrectedDays *int `json:"corrected_days"`
}

// scoreView is {scored:true, z, percentile} or {scored:false, reason}; Value is set for BMI only.
type scoreView struct {
	Scored     bool     `json:"scored"`
	Z          *float64 `json:"z,omitempty"`
	Percentile *float64 `json:"percentile,omitempty"`
	Reason     string   `json:"reason,omitempty"`
	Value      *float64 `json:"value,omitempty"`
}

type scoresView struct {
	Weight *scoreView `json:"weight"`
	Length *scoreView `json:"length"`
	Head   *scoreView `json:"head"`
	BMI    *scoreView `json:"bmi"`
}

type growthView struct {
	ID         string      `json:"id"`
	MeasuredOn string      `json:"measured_on"`
	Weight     *weightView `json:"weight"`
	Length     *lengthView `json:"length"`
	Head       *lengthView `json:"head"`
	Note       *string     `json:"note"`
	Age        ageView     `json:"age"`
	Scores     scoresView  `json:"scores"`
	VisitID    *string     `json:"visit_id"`
}

func newLengthView(l *units.Length) *lengthView {
	if l == nil {
		return nil
	}
	return &lengthView{MM: l.MM, InHundredths: l.InHundredths, TypedUnit: l.Typed}
}

// newGrowthView scores g's metric values against the kid's current source at the scoring age on g's date.
func newGrowthView(g model.Growth, kid model.Kid, loc *time.Location) (growthView, error) {
	basis, err := kidtime.AgeBasisOn(kid.BirthDate, kid.GestationWeeks, kid.GestationDays, g.MeasuredOn, loc)
	if err != nil {
		return growthView{}, err
	}
	v := growthView{ID: g.ID, VisitID: g.VisitID, MeasuredOn: g.MeasuredOn, Length: newLengthView(g.Length), Head: newLengthView(g.Head), Note: g.Note,
		Age: ageView{Days: basis.ChronologicalDays}}
	if basis.Corrected {
		v.Age.CorrectedDays = &basis.ScoringDays
	}
	var weightG, lengthMM, headMM *int64
	if w := g.Weight; w != nil {
		v.Weight = &weightView{Grams: w.Grams, OzHundredths: w.OzHundredths, TypedUnit: w.Typed}
		weightG = &w.Grams
	}
	if g.Length != nil {
		lengthMM = &g.Length.MM
	}
	if g.Head != nil {
		headMM = &g.Head.MM
	}
	s, err := growth.ScoreRow(growth.Source(kid.GrowthSource), growth.Sex(kid.Sex), basis, weightG, lengthMM, headMM)
	if err != nil {
		return growthView{}, err
	}
	v.Scores = scoresView{Weight: newScoreView(s.Weight), Length: newScoreView(s.Length), Head: newScoreView(s.Head), BMI: newScoreView(s.BMI)}
	if v.Scores.BMI != nil {
		v.Scores.BMI.Value = s.BMIValue
	}
	return v, nil
}

func newScoreView(s *growth.MeasureScore) *scoreView {
	if s == nil {
		return nil
	}
	if !s.Scored {
		return &scoreView{Reason: s.Reason}
	}
	z, p := s.Z, s.Percentile
	return &scoreView{Scored: true, Z: &z, Percentile: &p}
}

type measureInput struct {
	Unit   units.Unit `json:"unit"`
	Amount int64      `json:"amount"`
}

// optionalMeasure tells an absent key (Set false) from an explicit null (Set true, V nil).
type optionalMeasure struct {
	Set bool
	V   *measureInput
}

// UnmarshalJSON refuses unknown keys inside the measure, as the outer decoder does at the top level.
func (o *optionalMeasure) UnmarshalJSON(b []byte) error {
	o.Set = true
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	return dec.Decode(&o.V)
}

// growthInput is the body of POST and PATCH; on PATCH an absent key keeps the stored value and null clears a measure.
type growthInput struct {
	MeasuredOn *string         `json:"measured_on"`
	Weight     optionalMeasure `json:"weight"`
	Length     optionalMeasure `json:"length"`
	Head       optionalMeasure `json:"head"`
	Note       optionalString  `json:"note"`
}

// mergeWeight keeps stored when in is absent or re-sends the typed value, else converts in; false when it is refused.
func mergeWeight(stored *units.Weight, in optionalMeasure) (*units.Weight, bool) {
	if !in.Set {
		return stored, true
	}
	if in.V == nil {
		return nil, true
	}
	if stored != nil {
		typed := stored.Grams
		if stored.Typed == units.UnitLB {
			typed = stored.OzHundredths
		}
		if in.V.Unit == stored.Typed && in.V.Amount == typed {
			return stored, true
		}
	}
	w, err := units.NewWeight(in.V.Unit, in.V.Amount)
	return &w, err == nil
}

// mergeLength is mergeWeight for a length or head circumference.
func mergeLength(stored *units.Length, in optionalMeasure) (*units.Length, bool) {
	if !in.Set {
		return stored, true
	}
	if in.V == nil {
		return nil, true
	}
	if stored != nil {
		typed := stored.MM
		if stored.Typed == units.UnitIN {
			typed = stored.InHundredths
		}
		if in.V.Unit == stored.Typed && in.V.Amount == typed {
			return stored, true
		}
	}
	l, err := units.NewLength(in.V.Unit, in.V.Amount)
	return &l, err == nil
}

func lengthIn(l *units.Length, lo, hi int64) bool { return l == nil || (l.MM >= lo && l.MM <= hi) }

// mergeOnto applies in to g and names the first invalid field in the order measured_on, weight, length, head, note,
// then measurements when no measure is left; "" when valid. An unchanged stored date stays valid.
func (in growthInput) mergeOnto(g model.Growth, kid model.Kid, loc *time.Location, now time.Time) (model.Growth, string) {
	stored := g.MeasuredOn
	if in.MeasuredOn != nil {
		g.MeasuredOn = *in.MeasuredOn
	}
	if g.MeasuredOn == "" || g.MeasuredOn != stored {
		today, _ := kidtime.DayBounds(loc, now)
		if _, err := kidtime.ParseDate(g.MeasuredOn, loc); err != nil ||
			g.MeasuredOn > today.Format(time.DateOnly) || g.MeasuredOn < kid.BirthDate {
			return g, "measured_on"
		}
	}
	weight, ok := mergeWeight(g.Weight, in.Weight)
	if !ok || (weight != nil && (weight.Grams < constants.MinWeightGrams || weight.Grams > constants.MaxWeightGrams)) {
		return g, "weight"
	}
	g.Weight = weight
	length, ok := mergeLength(g.Length, in.Length)
	if !ok || !lengthIn(length, constants.MinLengthMM, constants.MaxLengthMM) {
		return g, "length"
	}
	g.Length = length
	head, ok := mergeLength(g.Head, in.Head)
	if !ok || !lengthIn(head, constants.MinHeadMM, constants.MaxHeadMM) {
		return g, "head"
	}
	g.Head = head
	if in.Note.Set {
		g.Note = in.Note.V
	}
	if !noteOK(g.Note) {
		return g, "note"
	}
	if g.Weight == nil && g.Length == nil && g.Head == nil {
		return g, "measurements"
	}
	return g, ""
}

// growthKid loads the kid in the path and its zone; it answers for a failed lookup.
func (d Deps) growthKid(w http.ResponseWriter, r *http.Request) (model.Kid, *time.Location, bool) {
	kid, ok := d.loadKid(w, r)
	if !ok {
		return model.Kid{}, nil, false
	}
	loc, ok := kidLocation(kid.TimeZone)
	if !ok {
		respondInternal(w, errKidZone)
		return model.Kid{}, nil, false
	}
	return kid, loc, true
}

func respondGrowth(w http.ResponseWriter, status int, g model.Growth, kid model.Kid, loc *time.Location) {
	v, err := newGrowthView(g, kid, loc)
	if err != nil {
		respondInternal(w, err)
		return
	}
	respondJSON(w, status, v)
}

func (d Deps) listGrowth(w http.ResponseWriter, r *http.Request) {
	kid, loc, ok := d.growthKid(w, r)
	if !ok {
		return
	}
	rows, err := d.Store.ListGrowth(r.Context(), kid.ID)
	if !entryOK(w, err) {
		return
	}
	views := make([]growthView, len(rows))
	for i, g := range rows {
		if views[i], err = newGrowthView(g, kid, loc); err != nil {
			respondInternal(w, err)
			return
		}
	}
	respondJSON(w, http.StatusOK, map[string]any{"growth": views})
}

func (d Deps) createGrowth(w http.ResponseWriter, r *http.Request) {
	var in growthInput
	if !decodeJSON(w, r, &in) {
		return
	}
	kid, loc, ok := d.growthKid(w, r)
	if !ok {
		return
	}
	now := d.Now()
	g, field := in.mergeOnto(model.Growth{}, kid, loc, now)
	if field != "" {
		respondInvalid(w, field)
		return
	}
	g, err := d.Store.CreateGrowth(r.Context(), kid.ID, PrincipalFrom(r.Context()).ParentID, g, now)
	if errors.Is(err, store.ErrConflict) {
		respondError(w, http.StatusConflict, "limit")
		return
	}
	if entryOK(w, err) {
		respondGrowth(w, http.StatusCreated, g, kid, loc)
	}
}

func (d Deps) getGrowth(w http.ResponseWriter, r *http.Request) {
	id, ok := entryID(w, r)
	if !ok {
		return
	}
	kid, loc, ok := d.growthKid(w, r)
	if !ok {
		return
	}
	g, err := d.Store.Growth(r.Context(), kid.ID, id)
	if entryOK(w, err) {
		respondGrowth(w, http.StatusOK, g, kid, loc)
	}
}

// updateGrowth validates the stored measurement with the patch applied, never the patch alone.
func (d Deps) updateGrowth(w http.ResponseWriter, r *http.Request) {
	var in growthInput
	if !decodeJSON(w, r, &in) {
		return
	}
	id, ok := entryID(w, r)
	if !ok {
		return
	}
	kid, loc, ok := d.growthKid(w, r)
	if !ok {
		return
	}
	stored, err := d.Store.Growth(r.Context(), kid.ID, id)
	if !entryOK(w, err) {
		return
	}
	now := d.Now()
	g, field := in.mergeOnto(stored, kid, loc, now)
	if field != "" {
		respondInvalid(w, field)
		return
	}
	if err := d.Store.UpdateGrowth(r.Context(), kid.ID, PrincipalFrom(r.Context()).ParentID, g, now); entryOK(w, err) {
		respondGrowth(w, http.StatusOK, g, kid, loc)
	}
}

func (d Deps) deleteGrowth(w http.ResponseWriter, r *http.Request) {
	deleteEntry(w, r, d.Store.DeleteGrowth)
}
