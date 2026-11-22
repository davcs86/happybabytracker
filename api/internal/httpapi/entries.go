package httpapi

import (
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
	return kidWriteOK(w, err)
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

// instantOK is false for a time later than now by more than MaxFutureSkew or before the kid's birth.
func instantOK(t, birth, now time.Time) bool {
	return !t.After(now.Add(constants.MaxFutureSkew)) && !t.Before(birth)
}

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
