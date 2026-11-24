package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/kidtime"
	"github.com/davcs86/happybabytracker/api/internal/milestones"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

// loadMilestones is replaced by tests to simulate an embedded catalog that failed to load.
var loadMilestones = milestones.Load

// errUnknownMilestoneItem: a stored row names an item the catalog no longer holds; items are retired, never dropped.
var errUnknownMilestoneItem = errors.New("milestones: stored item not in the catalog")

// milestoneCatalog is the shipped catalog; it answers 503 when the catalog failed to load.
func milestoneCatalog(w http.ResponseWriter) (milestones.Catalog, bool) {
	err := loadMilestones()
	c, ok := milestones.Get()
	if err == nil && !ok {
		err = milestones.ErrCatalogUnavailable
	}
	if err != nil {
		noteCause(w, err)
		respondError(w, http.StatusServiceUnavailable, "unavailable")
		return milestones.Catalog{}, false
	}
	return c, true
}

func catalogItem(c milestones.Catalog, id string) (milestones.Item, bool) {
	for _, it := range c.Items {
		if it.ID == id {
			return it, true
		}
	}
	return milestones.Item{}, false
}

type milestoneCatalogView struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Authority     string `json:"authority"`
	Edition       string `json:"edition"`
	Retrieved     string `json:"retrieved"`
	OriginURL     string `json:"origin_url"`
	Licence       string `json:"licence"`
	Attribution   string `json:"attribution"`
	ScreeningNote string `json:"screening_note"`
}

type milestoneAgeView struct {
	ChronologicalMonths int  `json:"chronological_months"`
	CorrectedMonths     *int `json:"corrected_months"`
	BasisMonths         int  `json:"basis_months"`
}

type itemReachedView struct {
	ID        string  `json:"id"`
	ReachedOn string  `json:"reached_on"`
	Note      *string `json:"note"`
}

type checklistItemView struct {
	ID      string           `json:"id"`
	Domain  string           `json:"domain"`
	Text    string           `json:"text"`
	Reached *itemReachedView `json:"reached"`
}

type checklistView struct {
	AgeMonths int                 `json:"age_months"`
	Position  string              `json:"position"`
	Items     []checklistItemView `json:"items"`
}

type customMilestoneView struct {
	ID        string  `json:"id"`
	Text      string  `json:"text"`
	ReachedOn string  `json:"reached_on"`
	Note      *string `json:"note"`
}

type retiredView struct {
	ID        string  `json:"id"`
	Item      string  `json:"item"`
	Text      string  `json:"text"`
	ReachedOn string  `json:"reached_on"`
	Note      *string `json:"note"`
}

type reachedRowView struct {
	ID        string  `json:"id"`
	Item      string  `json:"item"`
	AgeMonths int     `json:"age_months"`
	Domain    string  `json:"domain"`
	Text      string  `json:"text"`
	ReachedOn string  `json:"reached_on"`
	Note      *string `json:"note"`
}

type milestonesView struct {
	Catalog      milestoneCatalogView  `json:"catalog"`
	Age          milestoneAgeView      `json:"age"`
	CatalogShown bool                  `json:"catalog_shown"`
	Checklists   []checklistView       `json:"checklists"`
	Custom       []customMilestoneView `json:"custom"`
	Retired      []retiredView         `json:"retired"`
	Reached      []reachedRowView      `json:"reached"`
}

// newMilestonesView sorts rows (in store order) into custom, retired and reached; lists follow
// constants.MilestoneCatalogAges and items the catalog's file order. The catalog's omitted provenance never appears.
func newMilestonesView(c milestones.Catalog, rows []model.Milestone, age milestones.AgeBasis, shown bool) (milestonesView, error) {
	v := milestonesView{
		Catalog: milestoneCatalogView{ID: c.ID, Name: c.Name, Authority: c.Authority, Edition: c.Edition, Retrieved: c.Retrieved,
			OriginURL: c.OriginURL, Licence: c.Licence, Attribution: c.Attribution, ScreeningNote: c.ScreeningNote},
		Age:          milestoneAgeView{ChronologicalMonths: age.ChronologicalMonths, CorrectedMonths: age.CorrectedMonths, BasisMonths: age.BasisMonths},
		CatalogShown: shown, Checklists: []checklistView{}, Custom: []customMilestoneView{}, Retired: []retiredView{},
		Reached: []reachedRowView{},
	}
	reached := map[string]*itemReachedView{}
	for _, m := range rows {
		if m.Text != nil {
			v.Custom = append(v.Custom, customMilestoneView{ID: m.ID, Text: *m.Text, ReachedOn: m.ReachedOn, Note: m.Note})
			continue
		}
		it, ok := catalogItem(c, *m.Item)
		switch {
		case !ok:
			return milestonesView{}, errUnknownMilestoneItem
		case it.Retired != "":
			v.Retired = append(v.Retired, retiredView{ID: m.ID, Item: it.ID, Text: it.Text, ReachedOn: m.ReachedOn, Note: m.Note})
		case shown:
			reached[it.ID] = &itemReachedView{ID: m.ID, ReachedOn: m.ReachedOn, Note: m.Note}
		default:
			v.Reached = append(v.Reached, reachedRowView{ID: m.ID, Item: it.ID, AgeMonths: it.AgeMonths, Domain: string(it.Domain),
				Text: it.Text, ReachedOn: m.ReachedOn, Note: m.Note})
		}
	}
	if !shown {
		return v, nil
	}
	positions := milestones.Positions(constants.MilestoneCatalogAges, age.BasisMonths)
	for _, a := range constants.MilestoneCatalogAges {
		cl := checklistView{AgeMonths: a, Position: string(positions[a]), Items: []checklistItemView{}}
		for _, it := range c.Items {
			if it.AgeMonths == a && it.Retired == "" {
				cl.Items = append(cl.Items, checklistItemView{ID: it.ID, Domain: string(it.Domain), Text: it.Text, Reached: reached[it.ID]})
			}
		}
		v.Checklists = append(v.Checklists, cl)
	}
	return v, nil
}

type milestoneView struct {
	ID        string  `json:"id"`
	Item      *string `json:"item"`
	Text      string  `json:"text"`
	ReachedOn string  `json:"reached_on"`
	Note      *string `json:"note"`
}

// newMilestoneView words an item row with the catalog's text.
func newMilestoneView(m model.Milestone, c milestones.Catalog) (milestoneView, error) {
	v := milestoneView{ID: m.ID, Item: m.Item, ReachedOn: m.ReachedOn, Note: m.Note}
	if m.Text != nil {
		v.Text = *m.Text
		return v, nil
	}
	it, ok := catalogItem(c, *m.Item)
	if !ok {
		return milestoneView{}, errUnknownMilestoneItem
	}
	v.Text = it.Text
	return v, nil
}

func respondMilestone(w http.ResponseWriter, status int, m model.Milestone, c milestones.Catalog) {
	v, err := newMilestoneView(m, c)
	if err != nil {
		respondInternal(w, err)
		return
	}
	respondJSON(w, status, v)
}

// milestoneInput is the body of POST and PATCH; on PATCH an absent key keeps the stored value and null clears the note.
type milestoneInput struct {
	Item      optionalString `json:"item"`
	Text      optionalString `json:"text"`
	ReachedOn optionalString `json:"reached_on"`
	Note      optionalString `json:"note"`
}

// mergeOnto applies in to m (zero on POST) and names the first invalid field in the order item/text, reached_on, note;
// "" when valid. An item row keeps Text nil; an unchanged stored date stays valid.
func (in milestoneInput) mergeOnto(m model.Milestone, c milestones.Catalog, birthDate string, loc *time.Location, now time.Time) (model.Milestone, string) {
	if m.ID == "" {
		if in.Item.Set == in.Text.Set {
			return m, "item"
		}
		if in.Item.Set {
			if in.Item.V == nil {
				return m, "item"
			}
			if it, ok := catalogItem(c, *in.Item.V); !ok || it.Retired != "" {
				return m, "item"
			}
			m.Item = in.Item.V
		}
	} else if in.Item.Set || (in.Text.Set && m.Text == nil) {
		return m, "item"
	}
	if in.Text.Set {
		if in.Text.V == nil {
			return m, "text"
		}
		text := strings.TrimSpace(*in.Text.V)
		if text == "" || utf8.RuneCountInString(text) > constants.MilestoneTextMaxLen {
			return m, "text"
		}
		m.Text = &text
	}
	stored := m.ReachedOn
	if in.ReachedOn.Set {
		if in.ReachedOn.V == nil {
			return m, "reached_on"
		}
		m.ReachedOn = *in.ReachedOn.V
	}
	if m.ReachedOn == "" || m.ReachedOn != stored {
		today, _ := kidtime.DayBounds(loc, now)
		if _, err := kidtime.ParseDate(m.ReachedOn, loc); err != nil ||
			m.ReachedOn > today.Format(time.DateOnly) || m.ReachedOn < birthDate {
			return m, "reached_on"
		}
	}
	if in.Note.Set {
		m.Note = in.Note.V
	}
	if blank(m.Note) || !noteOK(m.Note) {
		return m, "note"
	}
	return m, ""
}

// milestoneWriteOK is entryOK that also answers a second mark of one item and the per-kid cap.
func milestoneWriteOK(w http.ResponseWriter, err error) bool {
	switch {
	case errors.Is(err, store.ErrDuplicate):
		respondError(w, http.StatusConflict, "already_reached")
	case errors.Is(err, store.ErrConflict):
		respondError(w, http.StatusConflict, "limit")
	default:
		return entryOK(w, err)
	}
	return false
}

// kidMilestones reads the kid and its rows from one snapshot; the answer is the same whether the module is on or off.
func (d Deps) kidMilestones(w http.ResponseWriter, r *http.Request) {
	c, ok := milestoneCatalog(w)
	if !ok {
		return
	}
	kid, rows, err := d.Store.KidMilestones(r.Context(), r.PathValue("kid"))
	if !kidWriteOK(w, err) {
		return
	}
	loc, ok := kidLocation(kid.TimeZone)
	if !ok {
		respondInternal(w, errKidZone)
		return
	}
	now := d.Now()
	age, err := milestones.AgeBasisFor(kid.BirthDate, kid.GestationWeeks, kid.GestationDays, loc, now)
	if err != nil {
		respondInternal(w, err)
		return
	}
	start, _ := kidtime.DayBounds(loc, now)
	shown, err := milestones.CatalogShown(kid.BirthDate, start.Format(time.DateOnly))
	if err != nil {
		respondInternal(w, err)
		return
	}
	v, err := newMilestonesView(c, rows, age, shown)
	if err != nil {
		respondInternal(w, err)
		return
	}
	respondJSON(w, http.StatusOK, v)
}

func (d Deps) createMilestone(w http.ResponseWriter, r *http.Request) {
	c, ok := milestoneCatalog(w)
	if !ok {
		return
	}
	var in milestoneInput
	if !decodeJSON(w, r, &in) {
		return
	}
	kid, loc, ok := d.growthKid(w, r)
	if !ok {
		return
	}
	now := d.Now()
	m, field := in.mergeOnto(model.Milestone{}, c, kid.BirthDate, loc, now)
	if field != "" {
		respondInvalid(w, field)
		return
	}
	m, err := d.Store.CreateMilestone(r.Context(), kid.ID, PrincipalFrom(r.Context()).ParentID, m, now)
	if milestoneWriteOK(w, err) {
		respondMilestone(w, http.StatusCreated, m, c)
	}
}

func (d Deps) getMilestone(w http.ResponseWriter, r *http.Request) {
	c, ok := milestoneCatalog(w)
	if !ok {
		return
	}
	id, ok := entryID(w, r)
	if !ok {
		return
	}
	m, err := d.Store.Milestone(r.Context(), r.PathValue("kid"), id)
	if entryOK(w, err) {
		respondMilestone(w, http.StatusOK, m, c)
	}
}

// updateMilestone validates the stored row with the patch applied, never the patch alone.
func (d Deps) updateMilestone(w http.ResponseWriter, r *http.Request) {
	c, ok := milestoneCatalog(w)
	if !ok {
		return
	}
	var in milestoneInput
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
	stored, err := d.Store.Milestone(r.Context(), kid.ID, id)
	if !entryOK(w, err) {
		return
	}
	now := d.Now()
	m, field := in.mergeOnto(stored, c, kid.BirthDate, loc, now)
	if field != "" {
		respondInvalid(w, field)
		return
	}
	if err := d.Store.UpdateMilestone(r.Context(), kid.ID, PrincipalFrom(r.Context()).ParentID, m, now); milestoneWriteOK(w, err) {
		respondMilestone(w, http.StatusOK, m, c)
	}
}

func (d Deps) deleteMilestone(w http.ResponseWriter, r *http.Request) {
	deleteEntry(w, r, d.Store.DeleteMilestone)
}
