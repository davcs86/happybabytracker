package httpapi

import (
	"net/http"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
)

type bottleView struct {
	ID       string     `json:"id"`
	At       string     `json:"at"`
	Contents []string   `json:"contents"`
	Volume   volumeView `json:"volume"`
	Note     *string    `json:"note"`
}

func newBottleView(b model.BottleFeed) bottleView {
	return bottleView{ID: b.ID, At: instantString(b.At), Contents: b.Contents, Volume: newVolumeView(b.Volume), Note: b.Note}
}

// bottleInput is the body of POST and PATCH; on PATCH an absent key keeps the stored value.
type bottleInput struct {
	At       *time.Time     `json:"at"`
	Contents *[]string      `json:"contents"`
	Volume   *volumeInput   `json:"volume"`
	Note     optionalString `json:"note"`
}

// mergeOnto applies in to b and names the first invalid field in the order at, contents, volume, note; "" when valid.
func (in bottleInput) mergeOnto(b model.BottleFeed, birth, now time.Time) (model.BottleFeed, string) {
	if in.At != nil {
		b.At = wholeSecond(*in.At)
	}
	if !instantOK(b.At, birth, now) {
		return b, "at"
	}
	if in.Contents != nil {
		b.Contents = *in.Contents
	}
	contents, ok := model.CanonicalContents(b.Contents)
	if !ok {
		return b, "contents"
	}
	b.Contents = contents
	volume, ok := mergeVolume(b.Volume, in.Volume)
	if !ok || volume.ML < constants.MinBottleVolumeML || volume.ML > constants.MaxBottleVolumeML {
		return b, "volume"
	}
	b.Volume = volume
	if in.Note.Set {
		b.Note = in.Note.V
	}
	if !noteOK(b.Note) {
		return b, "note"
	}
	return b, ""
}

func (d Deps) listBottles(w http.ResponseWriter, r *http.Request) {
	q, bad := parseListQuery(r, d.Now())
	if bad != "" {
		respondInvalid(w, bad)
		return
	}
	rows, err := d.Store.ListBottles(r.Context(), r.PathValue("kid"), pageQuery(q))
	if !entryOK(w, err) {
		return
	}
	respondPage(w, "bottles", rows, q.Limit, newBottleView, func(b model.BottleFeed) cursor { return entryCursor(b.At, b.ID) })
}

func (d Deps) createBottle(w http.ResponseWriter, r *http.Request) {
	var in bottleInput
	if !decodeJSON(w, r, &in) {
		return
	}
	birth, ok := d.birthStart(w, r)
	if !ok {
		return
	}
	now := d.Now()
	b, field := in.mergeOnto(model.BottleFeed{At: wholeSecond(now)}, birth, now)
	if field != "" {
		respondInvalid(w, field)
		return
	}
	b, err := d.Store.CreateBottle(r.Context(), r.PathValue("kid"), PrincipalFrom(r.Context()).ParentID, b, now)
	if entryOK(w, err) {
		respondJSON(w, http.StatusCreated, newBottleView(b))
	}
}

func (d Deps) getBottle(w http.ResponseWriter, r *http.Request) {
	id, ok := entryID(w, r)
	if !ok {
		return
	}
	b, err := d.Store.Bottle(r.Context(), r.PathValue("kid"), id)
	if entryOK(w, err) {
		respondJSON(w, http.StatusOK, newBottleView(b))
	}
}

// updateBottle validates the stored bottle with the patch applied, never the patch alone.
func (d Deps) updateBottle(w http.ResponseWriter, r *http.Request) {
	var in bottleInput
	if !decodeJSON(w, r, &in) {
		return
	}
	id, ok := entryID(w, r)
	if !ok {
		return
	}
	birth, ok := d.birthStart(w, r)
	if !ok {
		return
	}
	kidID := r.PathValue("kid")
	stored, err := d.Store.Bottle(r.Context(), kidID, id)
	if !entryOK(w, err) {
		return
	}
	now := d.Now()
	b, field := in.mergeOnto(stored, birth, now)
	if field != "" {
		respondInvalid(w, field)
		return
	}
	if err := d.Store.UpdateBottle(r.Context(), kidID, PrincipalFrom(r.Context()).ParentID, b, now); entryOK(w, err) {
		respondJSON(w, http.StatusOK, newBottleView(b))
	}
}

func (d Deps) deleteBottle(w http.ResponseWriter, r *http.Request) {
	id, ok := entryID(w, r)
	if !ok {
		return
	}
	if entryOK(w, d.Store.DeleteBottle(r.Context(), r.PathValue("kid"), PrincipalFrom(r.Context()).ParentID, id)) {
		w.WriteHeader(http.StatusNoContent)
	}
}
