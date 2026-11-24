package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/kidtime"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/store"
	"github.com/davcs86/happybabytracker/api/internal/vaccines"
)

// loadVaccines is replaced by tests to simulate embedded schedules that failed to load.
var loadVaccines = vaccines.Load

// antigenOther is the catalogue antigen whose administration must name its product.
const antigenOther = "other"

var dismissReasons = []string{"not_needed", "given_unknown"}

// statusRank orders a group's status: the lowest rank among its lines wins.
var statusRank = map[vaccines.Status]int{
	vaccines.StatusOverdue: 0, vaccines.StatusDue: 1, vaccines.StatusUpcoming: 2,
	vaccines.StatusAvailable: 3, vaccines.StatusDismissed: 4, vaccines.StatusGiven: 5,
}

// schedulesLoaded answers 503 when the embedded schedules failed to load; a stored schedule id is never judged then.
func schedulesLoaded(w http.ResponseWriter) bool {
	if err := loadVaccines(); err != nil {
		noteCause(w, err)
		respondError(w, http.StatusServiceUnavailable, "unavailable")
		return false
	}
	return true
}

// kidSchema is the kid's chosen schedule; false when none is chosen or it is no longer shipped.
func kidSchema(kid model.Kid) (vaccines.Schema, bool) {
	if kid.VaccineSchema == nil {
		return vaccines.Schema{}, false
	}
	return vaccines.GetSchema(*kid.VaccineSchema)
}

type schemaInfoView struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Authority   string `json:"authority"`
	Edition     string `json:"edition"`
	Retrieved   string `json:"retrieved"`
	OriginURL   string `json:"origin_url"`
	Licence     string `json:"licence"`
	Attribution string `json:"attribution"`
}

type schemaView struct {
	schemaInfoView
	Omitted []string `json:"omitted"`
}

func newSchemaInfoView(s vaccines.Schema) schemaInfoView {
	return schemaInfoView{ID: s.ID, Name: s.Name, Authority: s.Authority, Edition: s.Edition, Retrieved: s.Retrieved,
		OriginURL: s.OriginURL, Licence: s.Licence, Attribution: s.Attribution}
}

type ageJSON struct {
	N    int    `json:"n"`
	Unit string `json:"unit"`
}

type dismissalView struct {
	Reason string `json:"reason"`
}

type lineStateView struct {
	ID             string         `json:"id"`
	Group          string         `json:"group"`
	Antigen        string         `json:"antigen"`
	Series         int            `json:"series"`
	Kind           string         `json:"kind"`
	Status         string         `json:"status"`
	FromDate       string         `json:"from_date"`
	UntilDate      string         `json:"until_date"`
	GivenOn        *string        `json:"given_on"`
	Administration *string        `json:"administration"`
	Dismissal      *dismissalView `json:"dismissal"`
	Note           string         `json:"note"`
}

type lineGroupView struct {
	From     ageJSON         `json:"from"`
	FromDate string          `json:"from_date"`
	Status   string          `json:"status"`
	Lines    []lineStateView `json:"lines"`
}

type recurringDoseView struct {
	Administration string `json:"administration"`
	GivenOn        string `json:"given_on"`
}

type recurringView struct {
	Antigen string              `json:"antigen"`
	LineID  string              `json:"line_id"`
	Doses   []recurringDoseView `json:"doses"`
}

type otherDoseView struct {
	Administration string  `json:"administration"`
	Antigen        string  `json:"antigen"`
	GivenOn        string  `json:"given_on"`
	Product        *string `json:"product"`
}

type vaccinesView struct {
	Schema    *schemaView     `json:"schema"`
	Today     string          `json:"today"`
	Groups    []lineGroupView `json:"groups"`
	Also      []lineStateView `json:"also"`
	Recurring []recurringView `json:"recurring"`
	Other     []otherDoseView `json:"other"`
}

func optionalText(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func newAgeJSON(a vaccines.Age) ageJSON { return ageJSON{N: a.N, Unit: string(rune(a.Unit))} }

func newLineStateView(ls vaccines.LineState) lineStateView {
	v := lineStateView{ID: ls.Line.ID, Group: ls.Line.Group, Antigen: ls.Line.Antigen, Series: ls.Line.Series,
		Kind: string(ls.Line.Kind), Status: string(ls.Status), FromDate: ls.FromDate, UntilDate: ls.UntilDate,
		GivenOn: optionalText(ls.GivenOn), Administration: optionalText(ls.Administration), Note: ls.Line.Note}
	if ls.DismissReason != "" {
		v.Dismissal = &dismissalView{Reason: ls.DismissReason}
	}
	return v
}

// newVaccinesView groups e.Lines, which Evaluate sorts by from date; products maps a dose id to its product.
func newVaccinesView(s *vaccines.Schema, e vaccines.Evaluation, today string, products map[string]*string) vaccinesView {
	v := vaccinesView{Today: today, Groups: []lineGroupView{}, Also: []lineStateView{}, Recurring: []recurringView{},
		Other: []otherDoseView{}}
	if s != nil {
		v.Schema = &schemaView{schemaInfoView: newSchemaInfoView(*s), Omitted: s.Omitted}
		if v.Schema.Omitted == nil {
			v.Schema.Omitted = []string{}
		}
	}
	for _, ls := range e.Lines {
		if n := len(v.Groups); n == 0 || v.Groups[n-1].FromDate != ls.FromDate {
			v.Groups = append(v.Groups, lineGroupView{From: newAgeJSON(ls.Line.From),
				FromDate: ls.FromDate, Status: string(ls.Status)})
		}
		g := &v.Groups[len(v.Groups)-1]
		if statusRank[ls.Status] < statusRank[vaccines.Status(g.Status)] {
			g.Status = string(ls.Status)
		}
		g.Lines = append(g.Lines, newLineStateView(ls))
	}
	for _, ls := range e.Also {
		v.Also = append(v.Also, newLineStateView(ls))
	}
	for _, r := range e.Recurring {
		rv := recurringView{Antigen: r.Line.Antigen, LineID: r.Line.ID, Doses: []recurringDoseView{}}
		for _, d := range r.Doses {
			rv.Doses = append(rv.Doses, recurringDoseView{Administration: d.Administration, GivenOn: d.GivenOn})
		}
		v.Recurring = append(v.Recurring, rv)
	}
	for _, d := range e.Other {
		v.Other = append(v.Other, otherDoseView{Administration: d.Administration, Antigen: d.Antigen, GivenOn: d.GivenOn,
			Product: products[d.ID]})
	}
	return v
}

// evaluateKid evaluates the kid's doses against s (nil: no schedule, every dose is Other) on the kid-local today of now.
func (d Deps) evaluateKid(ctx context.Context, kid model.Kid, s *vaccines.Schema, loc *time.Location, now time.Time) (vaccines.Evaluation, []model.VaccineDose, string, error) {
	rows, err := d.Store.Doses(ctx, kid.ID)
	if err != nil {
		return vaccines.Evaluation{}, nil, "", err
	}
	doses := vaccines.DosesFrom(rows)
	var (
		schema     vaccines.Schema
		dismissals map[string]string
	)
	if s != nil {
		schema = *s
		if dismissals, err = d.Store.Dismissals(ctx, kid.ID, s.ID); err != nil {
			return vaccines.Evaluation{}, nil, "", err
		}
	}
	start, _ := kidtime.DayBounds(loc, now)
	today := start.Format(time.DateOnly)
	e, err := vaccines.Evaluate(schema, doses, dismissals, kid.BirthDate, today, loc)
	return e, rows, today, err
}

func (d Deps) vaccineSchemas(w http.ResponseWriter, _ *http.Request) {
	if !schedulesLoaded(w) {
		return
	}
	views := []schemaInfoView{}
	for _, s := range vaccines.Schemas() {
		views = append(views, newSchemaInfoView(s))
	}
	respondJSON(w, http.StatusOK, map[string]any{"schemas": views})
}

type antigenView struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type productView struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Antigens []string `json:"antigens"`
}

func (d Deps) vaccineCatalogue(w http.ResponseWriter, _ *http.Request) {
	if !schedulesLoaded(w) {
		return
	}
	antigens, products := []antigenView{}, []productView{}
	for _, a := range vaccines.Antigens() {
		antigens = append(antigens, antigenView{ID: a.ID, Name: a.Name})
	}
	for _, p := range vaccines.Products() {
		products = append(products, productView{ID: p.ID, Name: p.Name, Antigens: p.Antigens})
	}
	respondJSON(w, http.StatusOK, map[string]any{"antigens": antigens, "products": products})
}

func (d Deps) kidVaccines(w http.ResponseWriter, r *http.Request) {
	if !schedulesLoaded(w) {
		return
	}
	kid, loc, ok := d.growthKid(w, r)
	if !ok {
		return
	}
	var schema *vaccines.Schema
	if s, ok := kidSchema(kid); ok {
		schema = &s
	}
	e, rows, today, err := d.evaluateKid(r.Context(), kid, schema, loc, d.Now())
	if err != nil {
		respondInternal(w, err)
		return
	}
	products := map[string]*string{}
	for _, row := range rows {
		products[row.ID] = row.Product
	}
	respondJSON(w, http.StatusOK, newVaccinesView(schema, e, today, products))
}

type administrationView struct {
	ID       string   `json:"id"`
	GivenOn  string   `json:"given_on"`
	Antigens []string `json:"antigens"`
	Lot      *string  `json:"lot"`
	Product  *string  `json:"product"`
	Note     *string  `json:"note"`
}

func newAdministrationView(a model.Administration) administrationView {
	return administrationView{ID: a.ID, GivenOn: a.GivenOn, Antigens: a.Antigens, Lot: a.Lot, Product: a.Product, Note: a.Note}
}

// optionalStrings tells an absent key (Set false) from an explicit null (Set true, V nil).
type optionalStrings struct {
	Set bool
	V   *[]string
}

func (o *optionalStrings) UnmarshalJSON(b []byte) error {
	o.Set = true
	return json.Unmarshal(b, &o.V)
}

// administrationInput is the body of POST and PATCH; on PATCH an absent key keeps the stored value and null clears
// lot, product or note.
type administrationInput struct {
	GivenOn  optionalString  `json:"given_on"`
	Antigens optionalStrings `json:"antigens"`
	Lot      optionalString  `json:"lot"`
	Product  optionalString  `json:"product"`
	Note     optionalString  `json:"note"`
}

func tooLong(s *string, limit int) bool { return s != nil && utf8.RuneCountInString(*s) > limit }

// blank is set text that is empty or whitespace-only; the table CHECKs refuse the empty string.
func blank(s *string) bool { return s != nil && strings.TrimSpace(*s) == "" }

// mergeOnto applies in to a and names the first invalid field in the order given_on, antigens, lot, product, note;
// "" when valid. An unchanged stored date stays valid.
func (in administrationInput) mergeOnto(a model.Administration, birthDate string, loc *time.Location, now time.Time) (model.Administration, string) {
	stored := a.GivenOn
	if in.GivenOn.Set {
		if in.GivenOn.V == nil {
			return a, "given_on"
		}
		a.GivenOn = *in.GivenOn.V
	}
	if a.GivenOn == "" || a.GivenOn != stored {
		today, _ := kidtime.DayBounds(loc, now)
		if _, err := kidtime.ParseDate(a.GivenOn, loc); err != nil ||
			a.GivenOn > today.Format(time.DateOnly) || a.GivenOn < birthDate {
			return a, "given_on"
		}
	}
	if in.Antigens.Set {
		if in.Antigens.V == nil || vaccines.ValidAntigens(*in.Antigens.V) != nil {
			return a, "antigens"
		}
		a.Antigens = slices.Sorted(slices.Values(*in.Antigens.V))
	}
	if len(a.Antigens) == 0 {
		return a, "antigens"
	}
	if in.Lot.Set {
		a.Lot = in.Lot.V
	}
	if blank(a.Lot) || tooLong(a.Lot, constants.VaccineLotMaxLen) {
		return a, "lot"
	}
	if in.Product.Set {
		a.Product = in.Product.V
	}
	if blank(a.Product) || tooLong(a.Product, constants.VaccineProductMaxLen) ||
		(slices.Contains(a.Antigens, antigenOther) && a.Product == nil) {
		return a, "product"
	}
	if in.Note.Set {
		a.Note = in.Note.V
	}
	if blank(a.Note) || !noteOK(a.Note) {
		return a, "note"
	}
	return a, ""
}

// administrationID reads {administration}; a value that cannot be an id answers the 404 an unknown one gets.
func administrationID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := r.PathValue("administration")
	if !cursorIDPattern.MatchString(id) {
		respondNoKid(w)
		return "", false
	}
	return id, true
}

// administrationOK is kidWriteOK that also answers a refused antigen set and the per-kid cap.
func administrationOK(w http.ResponseWriter, err error) bool {
	switch {
	case errors.Is(err, store.ErrInvalidAntigens):
		respondInvalid(w, "antigens")
	case errors.Is(err, store.ErrConflict):
		respondError(w, http.StatusConflict, "limit")
	default:
		return kidWriteOK(w, err)
	}
	return false
}

func (d Deps) listVaccinations(w http.ResponseWriter, r *http.Request) {
	rows, err := d.Store.ListAdministrations(r.Context(), r.PathValue("kid"))
	if !kidWriteOK(w, err) {
		return
	}
	views := make([]administrationView, len(rows))
	for i, a := range rows {
		views[i] = newAdministrationView(a)
	}
	respondJSON(w, http.StatusOK, map[string]any{"vaccinations": views})
}

func (d Deps) createVaccination(w http.ResponseWriter, r *http.Request) {
	if !schedulesLoaded(w) {
		return
	}
	var in administrationInput
	if !decodeJSON(w, r, &in) {
		return
	}
	kid, loc, ok := d.growthKid(w, r)
	if !ok {
		return
	}
	now := d.Now()
	a, field := in.mergeOnto(model.Administration{}, kid.BirthDate, loc, now)
	if field != "" {
		respondInvalid(w, field)
		return
	}
	a, err := d.Store.CreateAdministration(r.Context(), kid.ID, PrincipalFrom(r.Context()).ParentID, a, now)
	if administrationOK(w, err) {
		respondJSON(w, http.StatusCreated, newAdministrationView(a))
	}
}

func (d Deps) getVaccination(w http.ResponseWriter, r *http.Request) {
	id, ok := administrationID(w, r)
	if !ok {
		return
	}
	a, err := d.Store.Administration(r.Context(), r.PathValue("kid"), id)
	if kidWriteOK(w, err) {
		respondJSON(w, http.StatusOK, newAdministrationView(a))
	}
}

// updateVaccination validates the stored administration with the patch applied, never the patch alone.
func (d Deps) updateVaccination(w http.ResponseWriter, r *http.Request) {
	if !schedulesLoaded(w) {
		return
	}
	var in administrationInput
	if !decodeJSON(w, r, &in) {
		return
	}
	id, ok := administrationID(w, r)
	if !ok {
		return
	}
	kid, loc, ok := d.growthKid(w, r)
	if !ok {
		return
	}
	stored, err := d.Store.Administration(r.Context(), kid.ID, id)
	if !kidWriteOK(w, err) {
		return
	}
	now := d.Now()
	a, field := in.mergeOnto(stored, kid.BirthDate, loc, now)
	if field != "" {
		respondInvalid(w, field)
		return
	}
	if err := d.Store.UpdateAdministration(r.Context(), kid.ID, PrincipalFrom(r.Context()).ParentID, a, now); administrationOK(w, err) {
		respondJSON(w, http.StatusOK, newAdministrationView(a))
	}
}

func (d Deps) deleteVaccination(w http.ResponseWriter, r *http.Request) {
	id, ok := administrationID(w, r)
	if !ok {
		return
	}
	if kidWriteOK(w, d.Store.DeleteAdministration(r.Context(), r.PathValue("kid"), PrincipalFrom(r.Context()).ParentID, id)) {
		w.WriteHeader(http.StatusNoContent)
	}
}

// dismissalLine is the kid's schedule and its {line}; it answers 404 unless the line is one of the kid's
// Evaluation.Lines (applicable routine or may_not_be_needed), so recurring, supplementary and other cohorts' are refused.
func (d Deps) dismissalLine(w http.ResponseWriter, r *http.Request) (model.Kid, vaccines.Schema, string, bool) {
	kid, loc, ok := d.growthKid(w, r)
	if !ok {
		return model.Kid{}, vaccines.Schema{}, "", false
	}
	s, ok := kidSchema(kid)
	if !ok {
		respondNoKid(w)
		return model.Kid{}, vaccines.Schema{}, "", false
	}
	e, _, _, err := d.evaluateKid(r.Context(), kid, &s, loc, d.Now())
	if err != nil {
		respondInternal(w, err)
		return model.Kid{}, vaccines.Schema{}, "", false
	}
	line := r.PathValue("line")
	if !slices.ContainsFunc(e.Lines, func(l vaccines.LineState) bool { return l.Line.ID == line }) {
		respondNoKid(w)
		return model.Kid{}, vaccines.Schema{}, "", false
	}
	return kid, s, line, true
}

func (d Deps) setDismissal(w http.ResponseWriter, r *http.Request) {
	if !schedulesLoaded(w) {
		return
	}
	var in struct {
		Reason string `json:"reason"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	kid, s, line, ok := d.dismissalLine(w, r)
	if !ok {
		return
	}
	if !slices.Contains(dismissReasons, in.Reason) {
		respondInvalid(w, "reason")
		return
	}
	err := d.Store.SetDismissal(r.Context(), kid.ID, PrincipalFrom(r.Context()).ParentID, s.ID, line, in.Reason, d.Now())
	if kidWriteOK(w, err) {
		respondJSON(w, http.StatusOK, map[string]string{"line": line, "reason": in.Reason})
	}
}

func (d Deps) clearDismissal(w http.ResponseWriter, r *http.Request) {
	if !schedulesLoaded(w) {
		return
	}
	kid, s, line, ok := d.dismissalLine(w, r)
	if !ok {
		return
	}
	if kidWriteOK(w, d.Store.ClearDismissal(r.Context(), kid.ID, PrincipalFrom(r.Context()).ParentID, s.ID, line)) {
		w.WriteHeader(http.StatusNoContent)
	}
}

// dismissBefore dismisses, in one transaction, every unsettled routine or may_not_be_needed line of the kid's
// schedule whose window opens before the given date.
func (d Deps) dismissBefore(w http.ResponseWriter, r *http.Request) {
	if !schedulesLoaded(w) {
		return
	}
	var in struct {
		Before string `json:"before"`
		Reason string `json:"reason"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	kid, loc, ok := d.growthKid(w, r)
	if !ok {
		return
	}
	s, ok := kidSchema(kid)
	if !ok {
		respondNoKid(w)
		return
	}
	if _, err := time.Parse(time.DateOnly, in.Before); err != nil {
		respondInvalid(w, "before")
		return
	}
	if !slices.Contains(dismissReasons, in.Reason) {
		respondInvalid(w, "reason")
		return
	}
	e, _, _, err := d.evaluateKid(r.Context(), kid, &s, loc, d.Now())
	if err != nil {
		respondInternal(w, err)
		return
	}
	lines := []string{}
	for _, l := range e.Lines {
		settled := l.Status == vaccines.StatusGiven || l.Status == vaccines.StatusDismissed || l.Status == vaccines.StatusAvailable
		kindOK := l.Line.Kind == vaccines.KindRoutine || l.Line.Kind == vaccines.KindMayNotBeNeeded
		if kindOK && !settled && l.FromDate < in.Before {
			lines = append(lines, l.Line.ID)
		}
	}
	n, err := d.Store.DismissLines(r.Context(), kid.ID, PrincipalFrom(r.Context()).ParentID, s.ID, lines, in.Reason, d.Now())
	if kidWriteOK(w, err) {
		respondJSON(w, http.StatusOK, map[string]int{"dismissed": n})
	}
}
