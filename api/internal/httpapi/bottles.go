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

type bottleSourceView struct {
	Source string     `json:"source"`
	Volume volumeView `json:"volume"`
}

type bottleView struct {
	ID       string             `json:"id"`
	At       string             `json:"at"`
	Contents []string           `json:"contents"`
	Volume   volumeView         `json:"volume"`
	Sources  []bottleSourceView `json:"sources"`
	Note     *string            `json:"note"`
}

func newBottleView(b model.BottleFeed) bottleView {
	sources := make([]bottleSourceView, len(b.Sources))
	for i, src := range b.Sources {
		sources[i] = bottleSourceView{Source: src.Source, Volume: newVolumeView(src.Volume)}
	}
	return bottleView{ID: b.ID, At: instantString(b.At), Contents: b.Contents, Volume: newVolumeView(b.Volume), Sources: sources, Note: b.Note}
}

// sourcesInput types every source of an advanced bottle in one unit; its keys are the bottle's contents.
type sourcesInput struct {
	Unit    units.Unit       `json:"unit"`
	Amounts map[string]int64 `json:"amounts"`
}

// names is the keys as a canonical contents set; false for none, or a name outside model.ContentNames.
func (in *sourcesInput) names() ([]string, bool) {
	if in == nil {
		return nil, false
	}
	return model.CanonicalContents(slices.Collect(maps.Keys(in.Amounts)))
}

// bottleInput is the body of POST and PATCH; on PATCH an absent key keeps the stored value. Volume and Sources
// exclude each other: Volume makes a simple bottle, Sources an advanced one.
type bottleInput struct {
	At       *time.Time     `json:"at"`
	Contents *[]string      `json:"contents"`
	Volume   *volumeInput   `json:"volume"`
	Sources  *sourcesInput  `json:"sources"`
	Note     optionalString `json:"note"`
}

// mergeSources types names (the keys of in) over stored, keeping each stored source and the stored total whose typed
// amount is unchanged; the total is the sum of the typed amounts. It names "sources" or "volume" when it refuses.
func mergeSources(stored []model.BottleSource, storedTotal units.Volume, in *sourcesInput, names []string) ([]model.BottleSource, units.Volume, string) {
	out := make([]model.BottleSource, 0, len(names))
	var sum int64
	for _, name := range names {
		var v units.Volume
		var ok bool
		if i := slices.IndexFunc(stored, func(s model.BottleSource) bool { return s.Source == name }); i >= 0 {
			v, ok = mergeVolume(stored[i].Volume, &volumeInput{Unit: in.Unit, Amount: in.Amounts[name]})
		} else {
			var err error
			v, err = units.NewVolume(in.Unit, in.Amounts[name])
			ok = err == nil
		}
		if !ok {
			return nil, units.Volume{}, "sources"
		}
		out = append(out, model.BottleSource{Source: name, Volume: v})
		sum += in.Amounts[name]
	}
	total, ok := mergeVolume(storedTotal, &volumeInput{Unit: in.Unit, Amount: sum})
	if !ok {
		return nil, units.Volume{}, "volume"
	}
	return out, total, ""
}

// mergeOnto applies in to b and names the first invalid field in the order at, contents, volume or sources, note;
// "" when valid.
func (in bottleInput) mergeOnto(b model.BottleFeed, birth, now time.Time) (model.BottleFeed, string) {
	stored := b.At
	if in.At != nil {
		b.At = wholeSecond(*in.At)
	}
	if !changedOK(b.At, stored, birth, now) {
		return b, "at"
	}
	names, namesOK := in.Sources.names()
	switch {
	case in.Contents != nil:
		contents, ok := model.CanonicalContents(*in.Contents)
		if !ok || (namesOK && !slices.Equal(contents, names)) {
			return b, "contents"
		}
		// Contents that drop or add a source would leave the stored sources disagreeing with them.
		if in.Sources == nil && in.Volume == nil && len(b.Sources) > 0 && !slices.Equal(contents, b.Contents) {
			return b, "contents"
		}
		b.Contents = contents
	case in.Sources != nil:
		if namesOK {
			b.Contents = names
		}
	default:
		contents, ok := model.CanonicalContents(b.Contents)
		if !ok {
			return b, "contents"
		}
		b.Contents = contents
	}
	var volume units.Volume
	switch {
	case in.Volume != nil && in.Sources != nil:
		return b, "sources"
	case in.Sources != nil:
		if !namesOK {
			return b, "sources"
		}
		var field string
		if b.Sources, volume, field = mergeSources(b.Sources, b.Volume, in.Sources, names); field != "" {
			return b, field
		}
	default:
		var ok bool
		if volume, ok = mergeVolume(b.Volume, in.Volume); !ok {
			return b, "volume"
		}
		if in.Volume != nil {
			b.Sources = nil
		}
	}
	if volume.ML < constants.MinBottleVolumeML || volume.ML > constants.MaxBottleVolumeML {
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
	getEntry(w, r, d.Store.Bottle, newBottleView)
}

// updateBottle validates the stored bottle with the patch applied, never the patch alone.
func (d Deps) updateBottle(w http.ResponseWriter, r *http.Request) {
	patchEntry(d, w, r, d.Store.Bottle, bottleInput.mergeOnto, d.Store.UpdateBottle, newBottleView)
}

func (d Deps) deleteBottle(w http.ResponseWriter, r *http.Request) {
	deleteEntry(w, r, d.Store.DeleteBottle)
}
