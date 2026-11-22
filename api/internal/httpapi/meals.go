package httpapi

import (
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
)

type mealView struct {
	ID       string  `json:"id"`
	At       string  `json:"at"`
	What     string  `json:"what"`
	Reaction *string `json:"reaction"`
	Note     *string `json:"note"`
}

func newMealView(m model.Meal) mealView {
	return mealView{ID: m.ID, At: instantString(m.At), What: m.What, Reaction: m.Reaction, Note: m.Note}
}

// mealInput is the body of POST and PATCH; on PATCH an absent key keeps the stored value.
type mealInput struct {
	At       *time.Time     `json:"at"`
	What     *string        `json:"what"`
	Reaction optionalString `json:"reaction"`
	Note     optionalString `json:"note"`
}

// mergeOnto applies in to m and names the first invalid field in the order at, what, reaction, note; "" when valid.
func (in mealInput) mergeOnto(m model.Meal, birth, now time.Time) (model.Meal, string) {
	stored := m.At
	if in.At != nil {
		m.At = wholeSecond(*in.At)
	}
	if !changedOK(m.At, stored, birth, now) {
		return m, "at"
	}
	if in.What != nil {
		m.What = strings.TrimSpace(*in.What)
	}
	if m.What == "" || utf8.RuneCountInString(m.What) > constants.MaxMealTextLen {
		return m, "what"
	}
	if in.Reaction.Set {
		m.Reaction = in.Reaction.V
	}
	if m.Reaction != nil && utf8.RuneCountInString(*m.Reaction) > constants.MaxMealTextLen {
		return m, "reaction"
	}
	if in.Note.Set {
		m.Note = in.Note.V
	}
	if !noteOK(m.Note) {
		return m, "note"
	}
	return m, ""
}

func (d Deps) listMeals(w http.ResponseWriter, r *http.Request) {
	q, bad := parseListQuery(r, d.Now())
	if bad != "" {
		respondInvalid(w, bad)
		return
	}
	rows, err := d.Store.ListMeals(r.Context(), r.PathValue("kid"), pageQuery(q))
	if !entryOK(w, err) {
		return
	}
	respondPage(w, "meals", rows, q.Limit, newMealView, func(m model.Meal) cursor { return entryCursor(m.At, m.ID) })
}

func (d Deps) createMeal(w http.ResponseWriter, r *http.Request) {
	var in mealInput
	if !decodeJSON(w, r, &in) {
		return
	}
	birth, ok := d.birthStart(w, r)
	if !ok {
		return
	}
	now := d.Now()
	m, field := in.mergeOnto(model.Meal{At: wholeSecond(now)}, birth, now)
	if field != "" {
		respondInvalid(w, field)
		return
	}
	m, err := d.Store.CreateMeal(r.Context(), r.PathValue("kid"), PrincipalFrom(r.Context()).ParentID, m, now)
	if entryOK(w, err) {
		respondJSON(w, http.StatusCreated, newMealView(m))
	}
}

func (d Deps) getMeal(w http.ResponseWriter, r *http.Request) {
	getEntry(w, r, d.Store.Meal, newMealView)
}

// updateMeal validates the stored meal with the patch applied, never the patch alone.
func (d Deps) updateMeal(w http.ResponseWriter, r *http.Request) {
	patchEntry(d, w, r, d.Store.Meal, mealInput.mergeOnto, d.Store.UpdateMeal, newMealView)
}

func (d Deps) deleteMeal(w http.ResponseWriter, r *http.Request) {
	deleteEntry(w, r, d.Store.DeleteMeal)
}
