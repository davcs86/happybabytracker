package httpapi

import (
	"context"
	"net/http"
	"slices"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
)

type nursingView struct {
	ID           string  `json:"id"`
	StartedAt    string  `json:"started_at"`
	EndedAt      *string `json:"ended_at"`
	LeftSeconds  int     `json:"left_seconds"`
	RightSeconds int     `json:"right_seconds"`
	Side         *string `json:"side"`
	SideSince    *string `json:"side_since"`
	Note         *string `json:"note"`
}

func newNursingView(n model.NursingSession) nursingView {
	return nursingView{
		ID: n.ID, StartedAt: instantString(n.StartedAt), EndedAt: optionalInstant(n.EndedAt),
		LeftSeconds: n.LeftSeconds, RightSeconds: n.RightSeconds, Side: n.Side, SideSince: optionalInstant(n.SideSince), Note: n.Note,
	}
}

// nursingInput is the body of POST and PATCH; on PATCH an absent key keeps the stored value.
type nursingInput struct {
	StartedAt    *time.Time     `json:"started_at"`
	EndedAt      optionalTime   `json:"ended_at"`
	LeftSeconds  *int           `json:"left_seconds"`
	RightSeconds *int           `json:"right_seconds"`
	Side         *string        `json:"side"`
	Note         optionalString `json:"note"`
}

// newSession builds a POST's session and names the first invalid field in the order started_at, ended_at,
// left_seconds, right_seconds, side, note, then the session invariants. No ended_at starts a timer on side.
func (in nursingInput) newSession(birth, now time.Time) (model.NursingSession, string) {
	n := model.NursingSession{StartedAt: wholeSecond(now)}
	if in.StartedAt != nil {
		n.StartedAt = wholeSecond(*in.StartedAt)
	}
	if !instantOK(n.StartedAt, birth, now) || (in.EndedAt.V == nil && n.StartedAt.After(now)) {
		return n, "started_at"
	}
	if in.EndedAt.V != nil {
		end := wholeSecond(*in.EndedAt.V)
		if !instantOK(end, birth, now) {
			return n, "ended_at"
		}
		n.EndedAt = &end
	}
	running := n.EndedAt == nil
	switch {
	case (in.LeftSeconds != nil) == running:
		return n, "left_seconds"
	case (in.RightSeconds != nil) == running:
		return n, "right_seconds"
	case (in.Side != nil) != running:
		return n, "side"
	}
	if running {
		since := n.StartedAt
		n.Side, n.SideSince = in.Side, &since
	} else {
		n.LeftSeconds, n.RightSeconds = *in.LeftSeconds, *in.RightSeconds
	}
	n.Note = in.Note.V
	if !noteOK(n.Note) {
		return n, "note"
	}
	return n, n.InvalidField()
}

// mergeOnto applies a PATCH to n in newSession's field order. The seconds are editable only once the session is
// finished, the side only through the side route; an ended_at on a running session ends it there.
func (in nursingInput) mergeOnto(n model.NursingSession, birth, now time.Time) (model.NursingSession, string) {
	wasRunning, stored := n.EndedAt == nil, n
	if in.StartedAt != nil {
		n = n.Restarted(wholeSecond(*in.StartedAt))
	}
	if !startOK(n.StartedAt, stored.StartedAt, wasRunning && in.EndedAt.V == nil, birth, now) {
		return n, "started_at"
	}
	if in.EndedAt.Set {
		if in.EndedAt.V == nil {
			if !wasRunning {
				return n, "ended_at"
			}
		} else {
			end := wholeSecond(*in.EndedAt.V)
			if endChanged(end, stored.EndedAt) && !instantOK(end, birth, now) {
				return n, "ended_at"
			}
			if !wasRunning {
				n.EndedAt = &end
			} else if ended, ok := n.Ended(end); ok {
				n = ended
			} else {
				return n, "ended_at"
			}
		}
	}
	if in.LeftSeconds != nil {
		if wasRunning {
			return n, "left_seconds"
		}
		n.LeftSeconds = *in.LeftSeconds
	}
	if in.RightSeconds != nil {
		if wasRunning {
			return n, "right_seconds"
		}
		n.RightSeconds = *in.RightSeconds
	}
	if in.Side != nil {
		return n, "side"
	}
	if in.Note.Set {
		n.Note = in.Note.V
	}
	if !noteOK(n.Note) {
		return n, "note"
	}
	return n, n.InvalidField()
}

func (d Deps) listNursing(w http.ResponseWriter, r *http.Request) {
	q, bad := parseListQuery(r, d.Now())
	if bad != "" {
		respondInvalid(w, bad)
		return
	}
	rows, err := d.Store.ListNursing(r.Context(), r.PathValue("kid"), pageQuery(q))
	if !entryOK(w, err) {
		return
	}
	respondPage(w, "nursing", rows, q.Limit, newNursingView, func(n model.NursingSession) cursor { return entryCursor(n.StartedAt, n.ID) })
}

func (d Deps) createNursing(w http.ResponseWriter, r *http.Request) {
	var in nursingInput
	if !decodeJSON(w, r, &in) {
		return
	}
	birth, ok := d.birthStart(w, r)
	if !ok {
		return
	}
	now := d.Now()
	n, field := in.newSession(birth, now)
	if field != "" {
		respondInvalid(w, field)
		return
	}
	n, err := d.Store.CreateNursing(r.Context(), r.PathValue("kid"), PrincipalFrom(r.Context()).ParentID, n, now)
	if entryOK(w, err) {
		respondJSON(w, http.StatusCreated, newNursingView(n))
	}
}

func (d Deps) getNursing(w http.ResponseWriter, r *http.Request) {
	getEntry(w, r, d.Store.Nursing, newNursingView)
}

func (d Deps) updateNursing(w http.ResponseWriter, r *http.Request) {
	patchInTx(d, w, r, nursingInput.mergeOnto, d.Store.UpdateNursing, newNursingView)
}

func (d Deps) deleteNursing(w http.ResponseWriter, r *http.Request) {
	deleteEntry(w, r, d.Store.DeleteNursing)
}

func (d Deps) endNursing(w http.ResponseWriter, r *http.Request) {
	var in endInput
	if !decodeJSON(w, r, &in) {
		return
	}
	timerAction(d, w, r, in.At, "", d.Store.Nursing, d.Store.EndNursing, newNursingView)
}

type sideInput struct {
	Side *string    `json:"side"`
	At   *time.Time `json:"at"`
}

func (d Deps) switchNursingSide(w http.ResponseWriter, r *http.Request) {
	var in sideInput
	if !decodeJSON(w, r, &in) {
		return
	}
	invalid := ""
	if in.Side == nil || !slices.Contains(model.NursingSides, *in.Side) {
		invalid = "side"
	}
	switchTo := func(ctx context.Context, kidID, actorID, id string, at, now time.Time) (model.NursingSession, error) {
		return d.Store.SwitchNursingSide(ctx, kidID, actorID, id, *in.Side, at, now)
	}
	timerAction(d, w, r, in.At, invalid, d.Store.Nursing, switchTo, newNursingView)
}
