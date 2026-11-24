package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/export"
	"github.com/davcs86/happybabytracker/api/internal/export/fonts"
	"github.com/davcs86/happybabytracker/api/internal/kidtime"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

// loadFonts is replaced by tests to simulate embedded fonts that failed to load.
var loadFonts = fonts.Load

// exportSlots admits MaxConcurrentExports renders and exportWriteSlots MaxConcurrentExportWrites downloads; neither queues.
var (
	exportSlots      = make(chan struct{}, constants.MaxConcurrentExports)
	exportWriteSlots = make(chan struct{}, constants.MaxConcurrentExportWrites)
)

type exportInput struct {
	From    string          `json:"from"`
	To      string          `json:"to"`
	Modules json.RawMessage `json:"modules"`
}

// exportModules reads a non-empty JSON array of distinct ExportModules ids; false for any other value.
func exportModules(raw json.RawMessage) ([]model.Module, bool) {
	var elems []json.RawMessage
	if len(raw) == 0 || raw[0] != '[' || json.Unmarshal(raw, &elems) != nil || len(elems) == 0 {
		return nil, false
	}
	out := make([]model.Module, 0, len(elems))
	for _, e := range elems {
		var id string
		if len(e) == 0 || e[0] != '"' || json.Unmarshal(e, &id) != nil ||
			!slices.Contains(constants.ExportModules, id) || slices.Contains(out, model.Module(id)) {
			return nil, false
		}
		out = append(out, model.Module(id))
	}
	return out, true
}

// exportRange checks from, then to, as kid-local dates; the third result names the first invalid one, or "".
func exportRange(in exportInput, birthDate string, loc *time.Location, now time.Time) (from, to time.Time, bad string) {
	from, err := kidtime.ParseDate(in.From, loc)
	if err != nil || in.From < birthDate {
		return time.Time{}, time.Time{}, "from"
	}
	to, err = kidtime.ParseDate(in.To, loc)
	today, _ := kidtime.DayBounds(loc, now)
	if err != nil || in.To < in.From || in.To > today.Format(time.DateOnly) {
		return time.Time{}, time.Time{}, "to"
	}
	fy, fm, fd := from.Date()
	ty, tm, td := to.Date()
	days := int(time.Date(ty, tm, td, 0, 0, 0, 0, time.UTC).Sub(time.Date(fy, fm, fd, 0, 0, 0, 0, time.UTC))/(24*time.Hour)) + 1
	if days > constants.ExportMaxDays {
		return time.Time{}, time.Time{}, "to"
	}
	return from, to, ""
}

// exportDataLoaded answers 503 when the fonts, or the embedded data of a requested module, failed to load.
func exportDataLoaded(w http.ResponseWriter, modules []model.Module) bool {
	if err := loadFonts(); err != nil {
		noteCause(w, err)
		respondError(w, http.StatusServiceUnavailable, "unavailable")
		return false
	}
	if slices.Contains(modules, "vaccination") && !schedulesLoaded(w) {
		return false
	}
	if slices.Contains(modules, "milestones") {
		if _, ok := milestoneCatalog(w); !ok {
			return false
		}
	}
	return true
}

// exportTypes is the timeline types of the requested modules, in TimelineTypes order.
func exportTypes(modules []model.Module) []string {
	var out []string
	for _, typ := range store.TimelineTypes {
		if slices.Contains(modules, timelineModules[typ]) {
			out = append(out, typ)
		}
	}
	return out
}

// export renders the whole PDF before the first byte, so every refusal is a JSON error and nothing about it is logged.
func (d Deps) export(w http.ResponseWriter, r *http.Request) {
	kid, loc, ok := d.growthKid(w, r)
	if !ok {
		return
	}
	parent, err := d.Store.ParentByID(r.Context(), PrincipalFrom(r.Context()).ParentID)
	if errors.Is(err, store.ErrNotFound) {
		respondError(w, http.StatusUnauthorized, "no_session")
		return
	}
	if err != nil {
		respondInternal(w, err)
		return
	}
	var in exportInput
	if !decodeJSON(w, r, &in) {
		return
	}
	now := d.Now()
	from, to, bad := exportRange(in, kid.BirthDate, loc, now)
	if bad != "" {
		respondInvalid(w, bad)
		return
	}
	modules, ok := exportModules(in.Modules)
	if !ok {
		respondInvalid(w, "modules")
		return
	}
	if !exportDataLoaded(w, modules) {
		return
	}
	select {
	case exportSlots <- struct{}{}:
	default:
		respondError(w, http.StatusServiceUnavailable, "busy")
		return
	}
	releaseRender := sync.OnceFunc(func() { <-exportSlots })
	defer releaseRender()
	snap, err := d.Store.ExportSnapshot(r.Context(), kid.ID, modules, exportTypes(modules), loc, from, to, now, constants.ExportMaxRowsPerSection)
	if !entryOK(w, err) {
		return
	}
	var buf bytes.Buffer
	req := export.Request{From: in.From, To: in.To, Modules: modules, Units: parent.Units, Now: now, Loc: loc}
	if err := export.Render(&buf, snap, req); err != nil {
		respondInternal(w, err)
		return
	}
	// Tried before the render slot is freed, so a refused render never overlaps the next one.
	select {
	case exportWriteSlots <- struct{}{}:
		defer func() { <-exportWriteSlots }()
	default:
		respondError(w, http.StatusServiceUnavailable, "busy")
		return
	}
	releaseRender()
	h := w.Header()
	h.Set("Content-Type", "application/pdf")
	h.Set("Content-Disposition", `attachment; filename="happybabytracker-`+in.From+`-`+in.To+`.pdf"`)
	h.Set("Cache-Control", "no-store")
	h.Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = buf.WriteTo(w)
}
