package httpapi

import (
	"net/http"
	"slices"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
)

type pumpingView struct {
	ID              string     `json:"id"`
	StartedAt       string     `json:"started_at"`
	DurationSeconds int        `json:"duration_seconds"`
	Volume          volumeView `json:"volume"`
	Side            string     `json:"side"`
	Note            *string    `json:"note"`
}

func newPumpingView(p model.PumpingSession) pumpingView {
	return pumpingView{
		ID: p.ID, StartedAt: instantString(p.StartedAt), DurationSeconds: p.DurationSeconds,
		Volume: newVolumeView(p.Volume), Side: p.Side, Note: p.Note,
	}
}

// pumpingInput is the body of POST and PATCH; on PATCH an absent key keeps the stored value.
type pumpingInput struct {
	StartedAt       *time.Time     `json:"started_at"`
	DurationSeconds *int           `json:"duration_seconds"`
	Volume          *volumeInput   `json:"volume"`
	Side            *string        `json:"side"`
	Note            optionalString `json:"note"`
}

// mergeOnto applies in to p and names the first invalid field in the order started_at, duration_seconds, volume, side, note; "" when valid.
func (in pumpingInput) mergeOnto(p model.PumpingSession, birth, now time.Time) (model.PumpingSession, string) {
	if in.StartedAt != nil {
		p.StartedAt = wholeSecond(*in.StartedAt)
	}
	if !instantOK(p.StartedAt, birth, now) {
		return p, "started_at"
	}
	if in.DurationSeconds != nil {
		p.DurationSeconds = *in.DurationSeconds
	}
	if p.DurationSeconds < int(constants.MinSessionDuration/time.Second) || p.DurationSeconds > int(constants.MaxSessionDuration/time.Second) {
		return p, "duration_seconds"
	}
	volume, ok := mergeVolume(p.Volume, in.Volume)
	if !ok || volume.ML < constants.MinPumpingVolumeML || volume.ML > constants.MaxPumpingVolumeML {
		return p, "volume"
	}
	p.Volume = volume
	if in.Side != nil {
		p.Side = *in.Side
	}
	if !slices.Contains(model.PumpingSides, p.Side) {
		return p, "side"
	}
	if in.Note.Set {
		p.Note = in.Note.V
	}
	if !noteOK(p.Note) {
		return p, "note"
	}
	return p, ""
}

func (d Deps) listPumping(w http.ResponseWriter, r *http.Request) {
	q, bad := parseListQuery(r, d.Now())
	if bad != "" {
		respondInvalid(w, bad)
		return
	}
	rows, err := d.Store.ListPumping(r.Context(), r.PathValue("kid"), pageQuery(q))
	if !entryOK(w, err) {
		return
	}
	respondPage(w, "pumping", rows, q.Limit, newPumpingView, func(p model.PumpingSession) cursor { return entryCursor(p.StartedAt, p.ID) })
}

func (d Deps) createPumping(w http.ResponseWriter, r *http.Request) {
	var in pumpingInput
	if !decodeJSON(w, r, &in) {
		return
	}
	birth, ok := d.birthStart(w, r)
	if !ok {
		return
	}
	now := d.Now()
	p, field := in.mergeOnto(model.PumpingSession{StartedAt: wholeSecond(now)}, birth, now)
	if field != "" {
		respondInvalid(w, field)
		return
	}
	p, err := d.Store.CreatePumping(r.Context(), r.PathValue("kid"), PrincipalFrom(r.Context()).ParentID, p, now)
	if entryOK(w, err) {
		respondJSON(w, http.StatusCreated, newPumpingView(p))
	}
}

func (d Deps) getPumping(w http.ResponseWriter, r *http.Request) {
	getEntry(w, r, d.Store.Pumping, newPumpingView)
}

// updatePumping validates the stored session with the patch applied, never the patch alone.
func (d Deps) updatePumping(w http.ResponseWriter, r *http.Request) {
	patchEntry(d, w, r, d.Store.Pumping, pumpingInput.mergeOnto, d.Store.UpdatePumping, newPumpingView)
}

func (d Deps) deletePumping(w http.ResponseWriter, r *http.Request) {
	deleteEntry(w, r, d.Store.DeletePumping)
}
