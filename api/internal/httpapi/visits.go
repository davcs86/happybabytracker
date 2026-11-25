package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

type linkedGrowthView struct {
	ID         string `json:"id"`
	MeasuredOn string `json:"measured_on"`
}

type linkedVaccinationView struct {
	ID       string   `json:"id"`
	GivenOn  string   `json:"given_on"`
	Antigens []string `json:"antigens"`
}

type linkedMedicationView struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	DoseCount int    `json:"dose_count"`
}

type visitLinksView struct {
	Growth       []linkedGrowthView      `json:"growth"`
	Vaccinations []linkedVaccinationView `json:"vaccinations"`
	Medications  []linkedMedicationView  `json:"medications"`
}

type visitCountsView struct {
	Growth       int `json:"growth"`
	Vaccinations int `json:"vaccinations"`
	Medications  int `json:"medications"`
}

type visitView struct {
	ID           string  `json:"id"`
	StartsAt     string  `json:"starts_at"`
	DurationMin  int     `json:"duration_min"`
	Kind         string  `json:"kind"`
	Status       string  `json:"status"`
	CancelledAt  *string `json:"cancelled_at"`
	Provider     *string `json:"provider"`
	Location     *string `json:"location"`
	Note         *string `json:"note"`
	OutcomeNotes *string `json:"outcome_notes"`
	Revision     int     `json:"revision"`
	Linked       any     `json:"linked"`
}

// visitStatus is derived, never stored (visits spec 5.2).
func visitStatus(v model.Visit, now time.Time) string {
	switch {
	case v.CancelledAt != nil:
		return "cancelled"
	case now.Before(v.StartsAt):
		return "upcoming"
	default:
		return "past"
	}
}

func newVisitView(v model.Visit, linked any, now time.Time) visitView {
	return visitView{ID: v.ID, StartsAt: instantString(v.StartsAt), DurationMin: v.DurationMin, Kind: string(v.Kind),
		Status: visitStatus(v, now), CancelledAt: optionalInstant(v.CancelledAt), Provider: v.Provider, Location: v.Location,
		Note: v.Note, OutcomeNotes: v.OutcomeNotes, Revision: v.Revision, Linked: linked}
}

func newVisitDetailView(d store.VisitDetail, now time.Time) visitView {
	links := visitLinksView{Growth: []linkedGrowthView{}, Vaccinations: []linkedVaccinationView{}, Medications: []linkedMedicationView{}}
	for _, g := range d.Links.Growth {
		links.Growth = append(links.Growth, linkedGrowthView(g))
	}
	for _, a := range d.Links.Vaccinations {
		links.Vaccinations = append(links.Vaccinations, linkedVaccinationView(a))
	}
	for _, m := range d.Links.Medications {
		links.Medications = append(links.Medications, linkedMedicationView(m))
	}
	return newVisitView(d.Visit, links, now)
}

// visitFields are the keys POST takes; PATCH adds outcome_notes and cancelled.
type visitFields struct {
	StartsAt    *time.Time     `json:"starts_at"`
	DurationMin *int           `json:"duration_min"`
	Kind        *string        `json:"kind"`
	Provider    optionalString `json:"provider"`
	Location    optionalString `json:"location"`
	Note        optionalString `json:"note"`
}

// visitInput is the body of PATCH: an absent key keeps the stored value, null clears an optional one.
type visitInput struct {
	visitFields
	OutcomeNotes optionalString `json:"outcome_notes"`
	Cancelled    optionalBool   `json:"cancelled"`
}

// optionalBool tells an absent key (Set false) from an explicit null (Set true, V nil).
type optionalBool struct {
	Set bool
	V   *bool
}

func (o *optionalBool) UnmarshalJSON(b []byte) error {
	o.Set = true
	return json.Unmarshal(b, &o.V)
}

// textOK is an absent or set text of 1 to limit characters that is not only spaces; the table CHECKs refuse "".
func textOK(s *string, limit int) bool {
	return s == nil || (strings.TrimSpace(*s) != "" && utf8.RuneCountInString(*s) <= limit)
}

// mergeOnto applies in to v and names the first invalid field in the order starts_at, duration_min, kind, provider,
// location, note, outcome_notes, cancelled; "" when valid. An unchanged stored start stays valid.
func (in visitInput) mergeOnto(v model.Visit, birth, now time.Time) (model.Visit, string) {
	stored := v.StartsAt
	if in.StartsAt != nil {
		v.StartsAt = wholeSecond(*in.StartsAt)
	}
	if v.StartsAt.IsZero() || (!v.StartsAt.Equal(stored) && (v.StartsAt.Before(birth) || v.StartsAt.After(now.Add(constants.VisitHorizon)))) {
		return v, "starts_at"
	}
	if in.DurationMin != nil {
		v.DurationMin = *in.DurationMin
	}
	if v.DurationMin < constants.MinVisitDurationMin || v.DurationMin > constants.MaxVisitDurationMin {
		return v, "duration_min"
	}
	if in.Kind != nil {
		v.Kind = model.VisitKind(*in.Kind)
	}
	if !slices.Contains(model.VisitKinds, v.Kind) {
		return v, "kind"
	}
	for _, f := range []struct {
		name  string
		in    optionalString
		dst   **string
		limit int
	}{
		{"provider", in.Provider, &v.Provider, constants.VisitProviderMaxLen},
		{"location", in.Location, &v.Location, constants.VisitLocationMaxLen},
		{"note", in.Note, &v.Note, constants.MaxNoteLen},
		{"outcome_notes", in.OutcomeNotes, &v.OutcomeNotes, constants.VisitOutcomeNotesMaxLen},
	} {
		if f.in.Set {
			*f.dst = f.in.V
		}
		if !textOK(*f.dst, f.limit) {
			return v, f.name
		}
	}
	if in.Cancelled.Set {
		switch {
		case in.Cancelled.V == nil:
			return v, "cancelled"
		case !*in.Cancelled.V:
			v.CancelledAt = nil
		case v.CancelledAt == nil:
			at := wholeSecond(now)
			v.CancelledAt = &at
		}
	}
	return v, ""
}

// visitOK is entryOK that also answers the visit rules and the per-kid cap.
func visitOK(w http.ResponseWriter, err error) bool {
	switch {
	case errors.Is(err, store.ErrHasOutcome):
		respondError(w, http.StatusConflict, "has_outcome")
	case errors.Is(err, store.ErrNotPast):
		respondError(w, http.StatusConflict, "not_past")
	case errors.Is(err, store.ErrStale):
		respondError(w, http.StatusConflict, "stale")
	case errors.Is(err, store.ErrConflict):
		respondError(w, http.StatusConflict, "limit")
	default:
		return entryOK(w, err)
	}
	return false
}

func (d Deps) listVisits(w http.ResponseWriter, r *http.Request) {
	now := d.Now()
	q, bad := parseListQuery(r, now)
	if bad != "" {
		respondInvalid(w, bad)
		return
	}
	v := r.URL.Query()
	horizon := now.Add(constants.VisitHorizon)
	if !v.Has("to") {
		q.To = horizon
	}
	if q.To.After(horizon) {
		respondInvalid(w, "to")
		return
	}
	order := v.Get("order")
	if v.Has("order") && order != "asc" && order != "desc" {
		respondInvalid(w, "order")
		return
	}
	rows, err := d.Store.ListVisits(r.Context(), r.PathValue("kid"), pageQuery(q), order == "asc")
	if !entryOK(w, err) {
		return
	}
	respondPage(w, "visits", rows, q.Limit, func(row store.VisitRow) visitView {
		return newVisitView(row.Visit, visitCountsView(row.Linked), now)
	}, func(row store.VisitRow) cursor { return entryCursor(row.StartsAt, row.ID) })
}

func (d Deps) createVisit(w http.ResponseWriter, r *http.Request) {
	var in visitFields
	if !decodeJSON(w, r, &in) {
		return
	}
	_, birth, ok := d.kidZoneBirth(w, r)
	if !ok {
		return
	}
	now := d.Now()
	v, field := visitInput{visitFields: in}.mergeOnto(model.Visit{DurationMin: constants.DefaultVisitDurationMin}, birth, now)
	if field != "" {
		respondInvalid(w, field)
		return
	}
	v, err := d.Store.CreateVisit(r.Context(), r.PathValue("kid"), PrincipalFrom(r.Context()).ParentID, v, now)
	if visitOK(w, err) {
		respondJSON(w, http.StatusCreated, newVisitDetailView(store.VisitDetail{Visit: v}, now))
	}
}

func (d Deps) getVisit(w http.ResponseWriter, r *http.Request) {
	id, ok := entryID(w, r)
	if !ok {
		return
	}
	v, err := d.Store.Visit(r.Context(), r.PathValue("kid"), id)
	if entryOK(w, err) {
		respondJSON(w, http.StatusOK, newVisitDetailView(v, d.Now()))
	}
}

func (d Deps) updateVisit(w http.ResponseWriter, r *http.Request) {
	id, ok := entryID(w, r)
	if !ok {
		return
	}
	var in visitInput
	if !decodeJSON(w, r, &in) {
		return
	}
	_, birth, ok := d.kidZoneBirth(w, r)
	if !ok {
		return
	}
	now := d.Now()
	var field string
	out, err := d.Store.UpdateVisit(r.Context(), r.PathValue("kid"), PrincipalFrom(r.Context()).ParentID, id, now,
		func(stored model.Visit) (model.Visit, error) {
			v, f := in.mergeOnto(stored, birth, now)
			if f != "" {
				field = f
				return v, errEditRefused
			}
			return v, nil
		})
	if errors.Is(err, errEditRefused) {
		respondInvalid(w, field)
		return
	}
	if visitOK(w, err) {
		respondJSON(w, http.StatusOK, newVisitDetailView(out, now))
	}
}

func (d Deps) deleteVisit(w http.ResponseWriter, r *http.Request) {
	id, ok := entryID(w, r)
	if !ok {
		return
	}
	linked := r.URL.Query()
	deleteLinked := true
	if linked.Has("linked") {
		switch linked.Get("linked") {
		case "delete":
		case "keep":
			deleteLinked = false
		default:
			respondInvalid(w, "linked")
			return
		}
	}
	err := d.Store.DeleteVisit(r.Context(), r.PathValue("kid"), PrincipalFrom(r.Context()).ParentID, id, deleteLinked)
	if entryOK(w, err) {
		w.WriteHeader(http.StatusNoContent)
	}
}
