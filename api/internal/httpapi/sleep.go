package httpapi

import (
	"net/http"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
)

type sleepView struct {
	ID        string  `json:"id"`
	StartedAt string  `json:"started_at"`
	EndedAt   *string `json:"ended_at"`
	Note      *string `json:"note"`
}

func newSleepView(v model.SleepSession) sleepView {
	return sleepView{ID: v.ID, StartedAt: instantString(v.StartedAt), EndedAt: optionalInstant(v.EndedAt), Note: v.Note}
}

// sleepInput is the body of POST and PATCH; on PATCH an absent key keeps the stored value.
type sleepInput struct {
	StartedAt *time.Time     `json:"started_at"`
	EndedAt   optionalTime   `json:"ended_at"`
	Note      optionalString `json:"note"`
}

// mergeOnto applies in to v and names the first invalid field in the order started_at, ended_at, note, then the
// session invariants; "" when valid. A null ended_at keeps a running session running and never re-opens one.
func (in sleepInput) mergeOnto(v model.SleepSession, birth, now time.Time) (model.SleepSession, string) {
	stored := v
	if in.StartedAt != nil {
		v.StartedAt = wholeSecond(*in.StartedAt)
	}
	if !startOK(v.StartedAt, stored.StartedAt, v.EndedAt == nil && in.EndedAt.V == nil, birth, now) {
		return v, "started_at"
	}
	if in.EndedAt.Set {
		if in.EndedAt.V == nil {
			if v.EndedAt != nil {
				return v, "ended_at"
			}
		} else {
			end := wholeSecond(*in.EndedAt.V)
			if endChanged(end, stored.EndedAt) && !instantOK(end, birth, now) {
				return v, "ended_at"
			}
			v.EndedAt = &end
		}
	}
	if in.Note.Set {
		v.Note = in.Note.V
	}
	if !noteOK(v.Note) {
		return v, "note"
	}
	return v, v.InvalidField()
}

func (d Deps) listSleep(w http.ResponseWriter, r *http.Request) {
	q, bad := parseListQuery(r, d.Now())
	if bad != "" {
		respondInvalid(w, bad)
		return
	}
	rows, err := d.Store.ListSleep(r.Context(), r.PathValue("kid"), pageQuery(q))
	if !entryOK(w, err) {
		return
	}
	respondPage(w, "sleep", rows, q.Limit, newSleepView, func(v model.SleepSession) cursor { return entryCursor(v.StartedAt, v.ID) })
}

// createSleep starts a timer when ended_at is absent or null.
func (d Deps) createSleep(w http.ResponseWriter, r *http.Request) {
	var in sleepInput
	if !decodeJSON(w, r, &in) {
		return
	}
	birth, ok := d.birthStart(w, r)
	if !ok {
		return
	}
	now := d.Now()
	v, field := in.mergeOnto(model.SleepSession{StartedAt: wholeSecond(now)}, birth, now)
	if field != "" {
		respondInvalid(w, field)
		return
	}
	v, err := d.Store.CreateSleep(r.Context(), r.PathValue("kid"), PrincipalFrom(r.Context()).ParentID, v, now)
	if entryOK(w, err) {
		respondJSON(w, http.StatusCreated, newSleepView(v))
	}
}

func (d Deps) getSleep(w http.ResponseWriter, r *http.Request) {
	getEntry(w, r, d.Store.Sleep, newSleepView)
}

func (d Deps) updateSleep(w http.ResponseWriter, r *http.Request) {
	patchInTx(d, w, r, sleepInput.mergeOnto, d.Store.UpdateSleep, newSleepView)
}

func (d Deps) deleteSleep(w http.ResponseWriter, r *http.Request) {
	deleteEntry(w, r, d.Store.DeleteSleep)
}

func (d Deps) endSleep(w http.ResponseWriter, r *http.Request) {
	var in endInput
	if !decodeJSON(w, r, &in) {
		return
	}
	timerAction(d, w, r, in.At, "", d.Store.Sleep, d.Store.EndSleep, newSleepView)
}
