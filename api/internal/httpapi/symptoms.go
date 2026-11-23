package httpapi

import (
	"net/http"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/units"
)

const symptomFever = "fever"

type temperatureView struct {
	CHundredths int64      `json:"c_hundredths"`
	FHundredths int64      `json:"f_hundredths"`
	TypedUnit   units.Unit `json:"typed_unit"`
}

type symptomView struct {
	ID          string           `json:"id"`
	At          string           `json:"at"`
	Symptom     string           `json:"symptom"`
	Severity    *string          `json:"severity"`
	OtherText   *string          `json:"other_text"`
	Temperature *temperatureView `json:"temperature"`
	Method      *string          `json:"method"`
	Note        *string          `json:"note"`
}

func newSymptomView(v model.Symptom) symptomView {
	out := symptomView{ID: v.ID, At: instantString(v.At), Symptom: v.Symptom, Severity: v.Severity, OtherText: v.OtherText,
		Method: v.Method, Note: v.Note}
	if t := v.Temperature; t != nil {
		out.Temperature = &temperatureView{CHundredths: t.CHundredths, FHundredths: t.FHundredths, TypedUnit: t.Typed}
	}
	return out
}

// symptomInput is the body of POST and PATCH; on PATCH an absent key keeps the stored value and null clears one.
type symptomInput struct {
	At          *time.Time      `json:"at"`
	Symptom     *string         `json:"symptom"`
	Severity    optionalString  `json:"severity"`
	OtherText   optionalString  `json:"other_text"`
	Temperature optionalMeasure `json:"temperature"`
	Method      optionalString  `json:"method"`
	Note        optionalString  `json:"note"`
}

// mergeTemperature keeps stored when in is absent or re-sends the typed value, else converts in; false when it is refused.
func mergeTemperature(stored *units.Temperature, in optionalMeasure) (*units.Temperature, bool) {
	if !in.Set {
		return stored, true
	}
	if in.V == nil {
		return nil, true
	}
	if stored != nil {
		typed := stored.CHundredths
		if stored.Typed == units.UnitF {
			typed = stored.FHundredths
		}
		if in.V.Unit == stored.Typed && in.V.Amount == typed {
			return stored, true
		}
	}
	t, err := units.NewTemperature(in.V.Unit, in.V.Amount)
	return &t, err == nil
}

// mergeOnto applies in to v and names the first invalid field in the order at, symptom, severity, other_text,
// temperature, method, note; "" when valid. Leaving fever needs temperature and method cleared in the same body.
func (in symptomInput) mergeOnto(v model.Symptom, birth, now time.Time) (model.Symptom, string) {
	stored, storedSymptom := v.At, v.Symptom
	if in.At != nil {
		v.At = wholeSecond(*in.At)
	}
	if !changedOK(v.At, stored, birth, now) {
		return v, "at"
	}
	if in.Symptom != nil {
		v.Symptom = *in.Symptom
	}
	if !slices.Contains(model.Symptoms, v.Symptom) {
		return v, "symptom"
	}
	if in.Severity.Set {
		v.Severity = in.Severity.V
	}
	if v.Severity != nil && !slices.Contains(model.Severities, *v.Severity) {
		return v, "severity"
	}
	if in.OtherText.Set {
		v.OtherText = in.OtherText.V
	}
	if v.Symptom == "other" {
		if v.OtherText == nil {
			return v, "other_text"
		}
		text := strings.TrimSpace(*v.OtherText)
		if text == "" || utf8.RuneCountInString(text) > constants.SymptomOtherMaxLen {
			return v, "other_text"
		}
		v.OtherText = &text
	} else if v.OtherText != nil {
		return v, "other_text"
	}
	isFever := v.Symptom == symptomFever
	cleared := in.Temperature.Set && in.Temperature.V == nil && in.Method.Set && in.Method.V == nil
	if storedSymptom == symptomFever && !isFever && !cleared {
		return v, "temperature"
	}
	temperature, ok := mergeTemperature(v.Temperature, in.Temperature)
	if !ok || (temperature != nil) != isFever {
		return v, "temperature"
	}
	v.Temperature = temperature
	if in.Method.Set {
		v.Method = in.Method.V
	}
	if (v.Method != nil) != isFever || (v.Method != nil && !slices.Contains(model.TempMethods, *v.Method)) {
		return v, "method"
	}
	if in.Note.Set {
		v.Note = in.Note.V
	}
	if !noteOK(v.Note) {
		return v, "note"
	}
	return v, ""
}

func (d Deps) listSymptoms(w http.ResponseWriter, r *http.Request) {
	q, bad := parseListQuery(r, d.Now())
	if bad != "" {
		respondInvalid(w, bad)
		return
	}
	rows, err := d.Store.ListSymptoms(r.Context(), r.PathValue("kid"), pageQuery(q))
	if !entryOK(w, err) {
		return
	}
	respondPage(w, "symptoms", rows, q.Limit, newSymptomView, func(x model.Symptom) cursor { return entryCursor(x.At, x.ID) })
}

func (d Deps) createSymptom(w http.ResponseWriter, r *http.Request) {
	var in symptomInput
	if !decodeJSON(w, r, &in) {
		return
	}
	birth, ok := d.birthStart(w, r)
	if !ok {
		return
	}
	now := d.Now()
	x, field := in.mergeOnto(model.Symptom{At: wholeSecond(now)}, birth, now)
	if field != "" {
		respondInvalid(w, field)
		return
	}
	x, err := d.Store.CreateSymptom(r.Context(), r.PathValue("kid"), PrincipalFrom(r.Context()).ParentID, x, now)
	if entryOK(w, err) {
		respondJSON(w, http.StatusCreated, newSymptomView(x))
	}
}

func (d Deps) getSymptom(w http.ResponseWriter, r *http.Request) {
	getEntry(w, r, d.Store.Symptom, newSymptomView)
}

// updateSymptom validates the stored symptom with the patch applied, never the patch alone.
func (d Deps) updateSymptom(w http.ResponseWriter, r *http.Request) {
	patchEntry(d, w, r, d.Store.Symptom, symptomInput.mergeOnto, d.Store.UpdateSymptom, newSymptomView)
}

func (d Deps) deleteSymptom(w http.ResponseWriter, r *http.Request) {
	deleteEntry(w, r, d.Store.DeleteSymptom)
}
