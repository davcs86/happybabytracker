package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"
	"unicode/utf8"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/kidtime"
	"github.com/davcs86/happybabytracker/api/internal/store"
	"github.com/davcs86/happybabytracker/api/internal/units"
)

var errKidZone = errors.New("kid has no usable time zone")

// optionalString tells an absent key (Set false) from an explicit null (Set true, V nil).
type optionalString struct {
	Set bool
	V   *string
}

func (o *optionalString) UnmarshalJSON(b []byte) error {
	o.Set = true
	return json.Unmarshal(b, &o.V)
}

// optionalTime tells an absent key (Set false) from an explicit null (Set true, V nil).
type optionalTime struct {
	Set bool
	V   *time.Time
}

func (o *optionalTime) UnmarshalJSON(b []byte) error {
	o.Set = true
	return json.Unmarshal(b, &o.V)
}

// entryID reads {entry}; a value that cannot be an id answers the 404 an unknown entry gets.
func entryID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := r.PathValue("entry")
	if !cursorIDPattern.MatchString(id) {
		respondNoKid(w)
		return "", false
	}
	return id, true
}

// entryOK is true when err is nil; otherwise it answers the refused tracker read or write and is false.
func entryOK(w http.ResponseWriter, err error) bool {
	switch {
	case errors.Is(err, store.ErrAlreadyRunning):
		respondError(w, http.StatusConflict, "already_running")
	case errors.Is(err, store.ErrNotRunning):
		respondError(w, http.StatusConflict, "not_running")
	case errors.Is(err, store.ErrInvalidSession):
		respondInvalid(w, "at")
	case errors.Is(err, store.ErrInvalidContents):
		respondInvalid(w, "contents")
	default:
		return kidWriteOK(w, err)
	}
	return false
}

// birthStart is the earliest instant an entry of the kid in the path may carry; it answers for a failed lookup.
func (d Deps) birthStart(w http.ResponseWriter, r *http.Request) (time.Time, bool) {
	kid, ok := d.loadKid(w, r)
	if !ok {
		return time.Time{}, false
	}
	loc, zoneOK := kidLocation(kid.TimeZone)
	if !zoneOK {
		respondInternal(w, errKidZone)
		return time.Time{}, false
	}
	birth, err := kidtime.ParseDate(kid.BirthDate, loc)
	if err != nil {
		respondInternal(w, err)
		return time.Time{}, false
	}
	return birth, true
}

// wholeSecond drops the sub-second part, which the store does not keep.
func wholeSecond(t time.Time) time.Time { return time.Unix(t.Unix(), 0).UTC() }

func instantString(t time.Time) string { return t.UTC().Format(time.RFC3339) }

func optionalInstant(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := instantString(*t)
	return &s
}

// instantOK is false for a time later than now by more than MaxFutureSkew or before the kid's birth.
func instantOK(t, birth, now time.Time) bool {
	return !t.After(now.Add(constants.MaxFutureSkew)) && !t.Before(birth)
}

// changedOK is instantOK for a time an edit changed from stored; an unchanged one stays valid, so a later birth-date
// correction never locks an older entry.
func changedOK(t, stored, birth, now time.Time) bool {
	return t.Equal(stored) || instantOK(t, birth, now)
}

// startOK is changedOK for a timer's start; a running timer may not start after now, or it could not end yet.
func startOK(t, stored time.Time, running bool, birth, now time.Time) bool {
	return t.Equal(stored) || (instantOK(t, birth, now) && !(running && t.After(now)))
}

// endChanged is true when end differs from a stored end that may be absent.
func endChanged(end time.Time, stored *time.Time) bool { return stored == nil || !end.Equal(*stored) }

func noteOK(note *string) bool {
	return note == nil || utf8.RuneCountInString(*note) <= constants.MaxNoteLen
}

type volumeInput struct {
	Unit   units.Unit `json:"unit"`
	Amount int64      `json:"amount"`
}

type volumeView struct {
	ML             int64      `json:"ml"`
	FlOzHundredths int64      `json:"floz_hundredths"`
	TypedUnit      units.Unit `json:"typed_unit"`
}

func newVolumeView(v units.Volume) volumeView {
	return volumeView{ML: v.ML, FlOzHundredths: v.FlOzHundredths, TypedUnit: v.Typed}
}

// mergeVolume keeps stored when in is absent or re-sends the typed value, else converts in; false when it is refused.
func mergeVolume(stored units.Volume, in *volumeInput) (units.Volume, bool) {
	if in == nil {
		return stored, true
	}
	typed := stored.ML
	if stored.Typed == units.UnitFlOz {
		typed = stored.FlOzHundredths
	}
	if in.Unit == stored.Typed && in.Amount == typed {
		return stored, true
	}
	v, err := units.NewVolume(in.Unit, in.Amount)
	return v, err == nil
}

// pageQuery converts to the store's query, asking for one row past the page to learn whether another follows.
func pageQuery(q listQuery) store.ListQuery {
	out := store.ListQuery{From: q.From, To: q.To, Limit: q.Limit + 1}
	if q.After != nil {
		out.After = &store.Cursor{At: q.After.At, ID: q.After.ID}
	}
	return out
}

func entryCursor(at time.Time, id string) cursor { return cursor{At: at.Unix(), ID: id} }

// respondPage writes {key: views, "next_cursor": ...}; rows came from a pageQuery for limit, so a surplus row means another page.
func respondPage[T, V any](w http.ResponseWriter, key string, rows []T, limit int, view func(T) V, cursorOf func(T) cursor) {
	var next any
	if len(rows) > limit {
		rows = rows[:limit]
		next = encodeCursor(cursorOf(rows[limit-1]))
	}
	views := make([]V, len(rows))
	for i, row := range rows {
		views[i] = view(row)
	}
	respondJSON(w, http.StatusOK, map[string]any{key: views, "next_cursor": next})
}

// getEntry answers 200 with the view of the entry in the path; load is the tracker's store read.
func getEntry[T, V any](w http.ResponseWriter, r *http.Request, load func(ctx context.Context, kidID, id string) (T, error), view func(T) V) {
	id, ok := entryID(w, r)
	if !ok {
		return
	}
	v, err := load(r.Context(), r.PathValue("kid"), id)
	if entryOK(w, err) {
		respondJSON(w, http.StatusOK, view(v))
	}
}

// deleteEntry answers 204; del is the tracker's store delete.
func deleteEntry(w http.ResponseWriter, r *http.Request, del func(ctx context.Context, kidID, actorID, id string) error) {
	id, ok := entryID(w, r)
	if !ok {
		return
	}
	if entryOK(w, del(r.Context(), r.PathValue("kid"), PrincipalFrom(r.Context()).ParentID, id)) {
		w.WriteHeader(http.StatusNoContent)
	}
}

// patchEntry decodes the body, loads the stored entry, validates it with the patch merged in (merge names the first
// invalid field, "" when valid), saves it and answers 200 with the view.
func patchEntry[T, In, V any](
	d Deps, w http.ResponseWriter, r *http.Request,
	load func(ctx context.Context, kidID, id string) (T, error),
	merge func(in In, stored T, birth, now time.Time) (T, string),
	save func(ctx context.Context, kidID, actorID string, v T, now time.Time) error,
	view func(T) V,
) {
	var in In
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
	stored, err := load(r.Context(), kidID, id)
	if !entryOK(w, err) {
		return
	}
	now := d.Now()
	v, field := merge(in, stored, birth, now)
	if field != "" {
		respondInvalid(w, field)
		return
	}
	if err := save(r.Context(), kidID, PrincipalFrom(r.Context()).ParentID, v, now); entryOK(w, err) {
		respondJSON(w, http.StatusOK, view(v))
	}
}

// errEditRefused aborts a store transaction whose in-transaction merge named an invalid field.
var errEditRefused = errors.New("edit refused")

// patchInTx is patchEntry for timers: merge runs on the row read inside the store's write transaction, so a
// concurrent write is never lost and the running state it checks cannot change under it.
func patchInTx[T, In, V any](
	d Deps, w http.ResponseWriter, r *http.Request,
	merge func(in In, stored T, birth, now time.Time) (T, string),
	update func(ctx context.Context, kidID, actorID, id string, now time.Time, edit func(T) (T, error)) (T, error),
	view func(T) V,
) {
	var in In
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
	now := d.Now()
	var field string
	v, err := update(r.Context(), r.PathValue("kid"), PrincipalFrom(r.Context()).ParentID, id, now, func(stored T) (T, error) {
		v, f := merge(in, stored, birth, now)
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
	if entryOK(w, err) {
		respondJSON(w, http.StatusOK, view(v))
	}
}

// timerAction answers an end or side route: 404 for an unknown entry, then 422 for invalid (a body field the caller
// found invalid) or a bad at, then act at at (the body's, else now) and 200 with the view.
func timerAction[T, V any](
	d Deps, w http.ResponseWriter, r *http.Request, given *time.Time, invalid string,
	load func(ctx context.Context, kidID, id string) (T, error),
	act func(ctx context.Context, kidID, actorID, id string, at, now time.Time) (T, error),
	view func(T) V,
) {
	id, ok := entryID(w, r)
	if !ok {
		return
	}
	birth, ok := d.birthStart(w, r)
	if !ok {
		return
	}
	kidID := r.PathValue("kid")
	if _, err := load(r.Context(), kidID, id); !entryOK(w, err) {
		return
	}
	if invalid != "" {
		respondInvalid(w, invalid)
		return
	}
	now := d.Now()
	at := wholeSecond(now)
	if given != nil {
		at = wholeSecond(*given)
	}
	if !instantOK(at, birth, now) {
		respondInvalid(w, "at")
		return
	}
	v, err := act(r.Context(), kidID, PrincipalFrom(r.Context()).ParentID, id, at, now)
	if errors.Is(err, store.ErrInvalidSession) {
		respondInvalid(w, "at")
		return
	}
	if entryOK(w, err) {
		respondJSON(w, http.StatusOK, view(v))
	}
}

// endInput is the body of a timer's end route.
type endInput struct {
	At *time.Time `json:"at"`
}
