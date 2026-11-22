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
	"github.com/davcs86/happybabytracker/api/internal/growth"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

type kidView struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	BirthDate      string `json:"birth_date"`
	Sex            string `json:"sex"`
	GestationWeeks *int   `json:"gestation_weeks"`
	GestationDays  *int   `json:"gestation_days"`
	TimeZone       string `json:"time_zone"`
	GrowthSource   string `json:"growth_source"`
	Role           string `json:"role"`
}

func newKidView(k model.Kid, role model.Role) kidView {
	return kidView{
		ID: k.ID, Name: k.Name, BirthDate: k.BirthDate, Sex: k.Sex,
		GestationWeeks: k.GestationWeeks, GestationDays: k.GestationDays, TimeZone: k.TimeZone,
		GrowthSource: k.GrowthSource, Role: string(role),
	}
}

// optionalInt tells an absent key (Set false) from an explicit null (Set true, V nil).
type optionalInt struct {
	Set bool
	V   *int
}

func (o *optionalInt) UnmarshalJSON(b []byte) error {
	o.Set = true
	return json.Unmarshal(b, &o.V)
}

// kidInput is the body of POST and PATCH; on PATCH an absent key keeps the stored value.
type kidInput struct {
	Name           *string     `json:"name"`
	BirthDate      *string     `json:"birth_date"`
	Sex            *string     `json:"sex"`
	GestationWeeks optionalInt `json:"gestation_weeks"`
	GestationDays  optionalInt `json:"gestation_days"`
	TimeZone       *string     `json:"time_zone"`
	GrowthSource   *string     `json:"growth_source"`
}

func (in kidInput) mergeOnto(k model.Kid) model.Kid {
	for _, f := range []struct{ dst, src *string }{
		{&k.Name, in.Name}, {&k.BirthDate, in.BirthDate}, {&k.Sex, in.Sex}, {&k.TimeZone, in.TimeZone},
		{&k.GrowthSource, in.GrowthSource},
	} {
		if f.src != nil {
			*f.dst = *f.src
		}
	}
	if in.GestationWeeks.Set {
		k.GestationWeeks = in.GestationWeeks.V
	}
	if in.GestationDays.Set {
		k.GestationDays = in.GestationDays.V
	}
	k.Name = strings.TrimSpace(k.Name)
	return k
}

func inRange(n *int, lo, hi int) bool { return n != nil && *n >= lo && *n <= hi }

// kidLocation refuses names LoadLocation maps to something other than a real zone ("" is UTC, "Local" the host's).
func kidLocation(name string) (*time.Location, bool) {
	if name == "" || name == "Local" {
		return nil, false
	}
	loc, err := time.LoadLocation(name)
	return loc, err == nil
}

// invalidKidField names the first failing field in the order name, birth_date, sex, gestation, time_zone, growth_source; "" when valid.
func invalidKidField(k model.Kid, now time.Time) string {
	if n := utf8.RuneCountInString(k.Name); n < constants.KidNameMinLen || n > constants.KidNameMaxLen {
		return "name"
	}
	loc, zoneOK := kidLocation(k.TimeZone)
	if _, err := time.Parse(time.DateOnly, k.BirthDate); err != nil || k.BirthDate < constants.KidMinBirthDate ||
		(zoneOK && k.BirthDate > now.In(loc).Format(time.DateOnly)) {
		return "birth_date"
	}
	if !slices.Contains([]string{"female", "male"}, k.Sex) {
		return "sex"
	}
	if (k.GestationWeeks == nil && k.GestationDays != nil) ||
		(k.GestationWeeks != nil && !inRange(k.GestationWeeks, constants.GestationWeeksMin, constants.GestationWeeksMax)) {
		return "gestation_weeks"
	}
	if (k.GestationDays == nil && k.GestationWeeks != nil) ||
		(k.GestationDays != nil && !inRange(k.GestationDays, constants.GestationDaysMin, constants.GestationDaysMax)) {
		return "gestation_days"
	}
	if !zoneOK {
		return "time_zone"
	}
	if !slices.Contains(growth.Sources, growth.Source(k.GrowthSource)) {
		return "growth_source"
	}
	return ""
}

func (d Deps) createKid(w http.ResponseWriter, r *http.Request) {
	var in kidInput
	if !decodeJSON(w, r, &in) {
		return
	}
	now := d.Now()
	kid := in.mergeOnto(model.Kid{GrowthSource: string(growth.SourceWHO)})
	if field := invalidKidField(kid, now); field != "" {
		respondInvalid(w, field)
		return
	}
	kid, err := d.Store.CreateKid(r.Context(), kid, PrincipalFrom(r.Context()).ParentID, now)
	if errors.Is(err, store.ErrNotFound) { // the creator was disabled since the middleware, ending this session
		respondError(w, http.StatusUnauthorized, "no_session")
		return
	}
	if err != nil {
		respondInternal(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, newKidView(kid, model.RoleGuardian))
}

func (d Deps) listKids(w http.ResponseWriter, r *http.Request) {
	kids, err := d.Store.KidsFor(r.Context(), PrincipalFrom(r.Context()).ParentID)
	if err != nil {
		respondInternal(w, err)
		return
	}
	rows := make([]kidView, len(kids))
	for i, k := range kids {
		rows[i] = newKidView(k.Kid, k.Role)
	}
	respondJSON(w, http.StatusOK, struct {
		Kids []kidView `json:"kids"`
	}{rows})
}

// loadKid writes the response itself when the kid is gone (deleted since the middleware's lookup) or the store fails.
func (d Deps) loadKid(w http.ResponseWriter, r *http.Request) (model.Kid, bool) {
	kid, err := d.Store.Kid(r.Context(), r.PathValue("kid"))
	if errors.Is(err, store.ErrNotFound) {
		respondNoKid(w)
		return model.Kid{}, false
	}
	if err != nil {
		respondInternal(w, err)
		return model.Kid{}, false
	}
	return kid, true
}

func (d Deps) getKid(w http.ResponseWriter, r *http.Request) {
	kid, ok := d.loadKid(w, r)
	if !ok {
		return
	}
	respondJSON(w, http.StatusOK, newKidView(kid, KidRoleFrom(r.Context())))
}

// updateKid validates the stored kid with the patch applied, never the patch alone.
func (d Deps) updateKid(w http.ResponseWriter, r *http.Request) {
	var in kidInput
	if !decodeJSON(w, r, &in) {
		return
	}
	stored, ok := d.loadKid(w, r)
	if !ok {
		return
	}
	now := d.Now()
	kid := in.mergeOnto(stored)
	if field := invalidKidField(kid, now); field != "" {
		respondInvalid(w, field)
		return
	}
	err := d.Store.UpdateKid(r.Context(), kid, PrincipalFrom(r.Context()).ParentID, now)
	if !kidWriteOK(w, err) {
		return
	}
	respondJSON(w, http.StatusOK, newKidView(kid, KidRoleFrom(r.Context())))
}

func (d Deps) deleteKid(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ConfirmName string `json:"confirm_name"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	kid, ok := d.loadKid(w, r)
	if !ok {
		return
	}
	if in.ConfirmName != kid.Name {
		respondInvalid(w, "confirm_name")
		return
	}
	err := d.Store.DeleteKid(r.Context(), kid.ID, PrincipalFrom(r.Context()).ParentID)
	if !kidWriteOK(w, err) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
