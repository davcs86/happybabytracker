package httpapi

import (
	"maps"
	"net/http"
	"slices"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/units"
)

type pumpingSideView struct {
	Side   string     `json:"side"`
	Volume volumeView `json:"volume"`
}

type pumpingView struct {
	ID              string            `json:"id"`
	StartedAt       string            `json:"started_at"`
	DurationSeconds int               `json:"duration_seconds"`
	Volume          volumeView        `json:"volume"`
	Side            string            `json:"side"`
	Sides           []pumpingSideView `json:"sides"`
	Note            *string           `json:"note"`
}

func newPumpingView(p model.PumpingSession) pumpingView {
	sides := make([]pumpingSideView, len(p.Sides))
	for i, side := range p.Sides {
		sides[i] = pumpingSideView{Side: side.Side, Volume: newVolumeView(side.Volume)}
	}
	return pumpingView{
		ID: p.ID, StartedAt: instantString(p.StartedAt), DurationSeconds: p.DurationSeconds,
		Volume: newVolumeView(p.Volume), Side: p.Side, Sides: sides, Note: p.Note,
	}
}

// sidesInput types every side of an advanced session in one unit; its keys are the sides pumped.
type sidesInput struct {
	Unit    units.Unit       `json:"unit"`
	Amounts map[string]int64 `json:"amounts"`
}

// names is the keys as a canonical set; false for none, or a name outside model.PumpingSideNames.
func (in *sidesInput) names() ([]string, bool) {
	if in == nil {
		return nil, false
	}
	return model.CanonicalPumpingSides(slices.Collect(maps.Keys(in.Amounts)))
}

// pumpingInput is the body of POST and PATCH; on PATCH an absent key keeps the stored value. Volume and Sides
// exclude each other: Volume makes a simple session, Sides an advanced one whose Side is derived.
type pumpingInput struct {
	StartedAt       *time.Time     `json:"started_at"`
	DurationSeconds *int           `json:"duration_seconds"`
	Volume          *volumeInput   `json:"volume"`
	Sides           *sidesInput    `json:"sides"`
	Side            *string        `json:"side"`
	Note            optionalString `json:"note"`
}

// mergeSides types names (the keys of in) over stored, keeping each stored side and the stored total whose typed
// amount is unchanged; the total is the sum of the typed amounts. It names "sides" or "volume" when it refuses.
func mergeSides(stored []model.PumpingSide, storedTotal units.Volume, in *sidesInput, names []string) ([]model.PumpingSide, units.Volume, string) {
	out := make([]model.PumpingSide, 0, len(names))
	var sum int64
	for _, name := range names {
		var v units.Volume
		var ok bool
		if i := slices.IndexFunc(stored, func(s model.PumpingSide) bool { return s.Side == name }); i >= 0 {
			v, ok = mergeVolume(stored[i].Volume, &volumeInput{Unit: in.Unit, Amount: in.Amounts[name]})
		} else {
			var err error
			v, err = units.NewVolume(in.Unit, in.Amounts[name])
			ok = err == nil
		}
		if !ok {
			return nil, units.Volume{}, "sides"
		}
		out = append(out, model.PumpingSide{Side: name, Volume: v})
		sum += in.Amounts[name]
	}
	total, ok := mergeVolume(storedTotal, &volumeInput{Unit: in.Unit, Amount: sum})
	if !ok {
		return nil, units.Volume{}, "volume"
	}
	return out, total, ""
}

// mergeOnto applies in to p and names the first invalid field in the order started_at, duration_seconds, volume or
// sides, side, note; "" when valid.
func (in pumpingInput) mergeOnto(p model.PumpingSession, birth, now time.Time) (model.PumpingSession, string) {
	stored := p.StartedAt
	if in.StartedAt != nil {
		p.StartedAt = wholeSecond(*in.StartedAt)
	}
	if !changedOK(p.StartedAt, stored, birth, now) {
		return p, "started_at"
	}
	if in.DurationSeconds != nil {
		p.DurationSeconds = *in.DurationSeconds
	}
	if p.DurationSeconds < int(constants.MinSessionDuration/time.Second) || p.DurationSeconds > int(constants.MaxSessionDuration/time.Second) {
		return p, "duration_seconds"
	}
	var volume units.Volume
	switch {
	case in.Volume != nil && in.Sides != nil:
		return p, "sides"
	case in.Sides != nil:
		names, ok := in.Sides.names()
		if !ok {
			return p, "sides"
		}
		var field string
		if p.Sides, volume, field = mergeSides(p.Sides, p.Volume, in.Sides, names); field != "" {
			return p, field
		}
	default:
		var ok bool
		if volume, ok = mergeVolume(p.Volume, in.Volume); !ok {
			return p, "volume"
		}
		if in.Volume != nil {
			p.Sides = nil
		}
	}
	if volume.ML < constants.MinPumpingVolumeML || volume.ML > constants.MaxPumpingVolumeML {
		return p, "volume"
	}
	p.Volume = volume
	derived, advanced := model.DerivedPumpingSide(p.Sides)
	switch {
	case in.Sides != nil:
		if in.Side != nil && *in.Side != derived {
			return p, "side"
		}
		p.Side = derived
	case in.Side != nil:
		p.Side = *in.Side
	}
	if !slices.Contains(model.PumpingSides, p.Side) || (advanced && p.Side != derived) {
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
