package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/ics"
	"github.com/davcs86/happybabytracker/api/internal/kidtime"
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
	v := r.URL.Query()
	horizon := now.Add(constants.VisitHorizon)
	if !v.Has("to") {
		// The horizon is the default to before from is checked against it, so a future from needs no to.
		v.Set("to", horizon.UTC().Format(time.RFC3339))
		r = r.Clone(r.Context())
		r.URL.RawQuery = v.Encode()
	}
	q, bad := parseListQuery(r, now)
	if bad != "" {
		respondInvalid(w, bad)
		return
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

// outcomeInput is the body of POST …/outcome; each block is the body its module's own POST takes.
type outcomeInput struct {
	Revision     *int                  `json:"revision"`
	OutcomeNotes optionalString        `json:"outcome_notes"`
	Growth       *growthInput          `json:"growth"`
	Vaccinations []administrationInput `json:"vaccinations"`
	Medications  []medicationInput     `json:"medications"`
}

// outcome validates in through the modules' own mergeOnto, dates defaulting to the visit's; the refused field is
// named as growth.f, vaccinations[i].f or medications[i].f.
func (in outcomeInput) outcome(v model.Visit, kid model.Kid, loc *time.Location, birth, now time.Time) (store.VisitOutcome, string) {
	out := store.VisitOutcome{NotesSet: in.OutcomeNotes.Set, Notes: in.OutcomeNotes.V}
	if out.NotesSet && (out.Notes == nil || !textOK(out.Notes, constants.VisitOutcomeNotesMaxLen)) {
		return out, "outcome_notes"
	}
	day := v.StartsAt.In(loc).Format(time.DateOnly)
	if in.Growth != nil {
		g, f := in.Growth.mergeOnto(model.Growth{MeasuredOn: day}, kid, loc, now)
		if f != "" {
			return out, "growth." + f
		}
		out.Growth = &g
	}
	for i, a := range in.Vaccinations {
		admin, f := a.mergeOnto(model.Administration{GivenOn: day}, kid.BirthDate, loc, now)
		if f != "" {
			return out, rowField("vaccinations", i, f)
		}
		out.Vaccinations = append(out.Vaccinations, admin)
	}
	for i, m := range in.Medications {
		course, f := m.mergeOnto(model.Medication{StartAt: v.StartsAt, Active: true}, loc, birth, now)
		if f != "" {
			return out, rowField("medications", i, f)
		}
		out.Medications = append(out.Medications, course)
	}
	return out, ""
}

func rowField(section string, i int, field string) string {
	return section + "[" + strconv.Itoa(i) + "]." + field
}

// noteOutcomeCounts hands the rows one outcome created to the request's one log line.
func noteOutcomeCounts(w http.ResponseWriter, o store.VisitOutcome) {
	if rec := recorder(w); rec != nil {
		growth := 0
		if o.Growth != nil {
			growth = 1
		}
		rec.outcome = &[3]int{growth, len(o.Vaccinations), len(o.Medications)}
	}
}

func (d Deps) recordOutcome(w http.ResponseWriter, r *http.Request) {
	id, ok := entryID(w, r)
	if !ok {
		return
	}
	var in outcomeInput
	if !decodeJSON(w, r, &in) {
		return
	}
	switch {
	case in.Revision == nil:
		respondInvalid(w, "revision")
		return
	case !in.OutcomeNotes.Set && in.Growth == nil && len(in.Vaccinations) == 0 && len(in.Medications) == 0:
		respondInvalid(w, "outcome")
		return
	case len(in.Vaccinations) > constants.MaxOutcomeVaccinations:
		respondInvalid(w, "vaccinations")
		return
	case len(in.Medications) > constants.MaxOutcomeMedications:
		respondInvalid(w, "medications")
		return
	}
	if len(in.Vaccinations) > 0 && !schedulesLoaded(w) {
		return
	}
	kid, loc, ok := d.growthKid(w, r)
	if !ok {
		return
	}
	birth, err := kidtime.ParseDate(kid.BirthDate, loc)
	if err != nil {
		respondInternal(w, err)
		return
	}
	visit, err := d.Store.Visit(r.Context(), kid.ID, id)
	if !entryOK(w, err) {
		return
	}
	now := d.Now()
	o, field := in.outcome(visit.Visit, kid, loc, birth, now)
	if field != "" {
		respondInvalid(w, field)
		return
	}
	out, err := d.Store.RecordOutcome(r.Context(), kid.ID, PrincipalFrom(r.Context()).ParentID, id, *in.Revision, o, now)
	var row *store.OutcomeRowError
	switch {
	case errors.As(err, &row) && errors.Is(err, store.ErrInvalidAntigens):
		respondInvalid(w, rowField(row.Section, row.Index, "antigens"))
	case errors.As(err, &row) && errors.Is(err, store.ErrInvalidSchedule):
		respondInvalid(w, rowField(row.Section, row.Index, "schedule"))
	case visitOK(w, err):
		noteOutcomeCounts(w, o)
		respondJSON(w, http.StatusOK, newVisitDetailView(out, now))
	}
}

// visitCalendar serves the visit's .ics; generic leaves out the kid, the provider, the place and the reason, and
// the outcome notes never enter the file (visits spec 8).
func (d Deps) visitCalendar(w http.ResponseWriter, r *http.Request) {
	id, ok := entryID(w, r)
	if !ok {
		return
	}
	detail := r.URL.Query().Get("detail")
	if r.URL.Query().Has("detail") && detail != "generic" && detail != "full" {
		respondInvalid(w, "detail")
		return
	}
	kid, loc, ok := d.growthKid(w, r)
	if !ok {
		return
	}
	v, err := d.Store.Visit(r.Context(), kid.ID, id)
	if !entryOK(w, err) {
		return
	}
	origin, err := url.Parse(d.Cfg.PublicOrigin)
	if err != nil {
		respondInternal(w, err)
		return
	}
	e := ics.Event{UID: "visit-" + v.ID + "@" + origin.Hostname(), Sequence: v.Revision, Stamp: v.UpdatedAt, Start: v.StartsAt,
		End: v.StartsAt.Add(time.Duration(v.DurationMin) * time.Minute), Cancelled: v.CancelledAt != nil, Summary: "Medical visit",
		URL: d.Cfg.PublicOrigin + "/kids/" + kid.ID + "/visits/" + v.ID}
	if v.CancelledAt == nil {
		e.AlarmBefore = constants.VisitAlarmBefore
	}
	if detail == "full" {
		e.Summary = kid.Name + ": " + model.VisitKindLabel[v.Kind]
		if v.Provider != nil {
			e.Summary += " with " + *v.Provider
		}
		if v.Location != nil {
			e.Location = *v.Location
		}
		if v.Note != nil {
			e.Description = *v.Note
		}
	}
	var buf bytes.Buffer
	if err := ics.Write(&buf, e); err != nil {
		respondInternal(w, err)
		return
	}
	h := w.Header()
	h.Set("Content-Type", "text/calendar; charset=utf-8")
	h.Set("Content-Disposition", `attachment; filename="visit-`+v.StartsAt.In(loc).Format(time.DateOnly)+`.ics"`)
	h.Set("Cache-Control", "no-store")
	h.Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = buf.WriteTo(w)
}
