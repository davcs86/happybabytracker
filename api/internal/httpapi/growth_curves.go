package httpapi

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/growth"
	"github.com/davcs86/happybabytracker/api/internal/kidtime"
)

// growthMeasures is the order measures are listed in a source's coverage.
var growthMeasures = []growth.Measure{growth.Weight, growth.Length, growth.Head, growth.BMI}

// pointView carries Gap only on the first point after a table seam: the line must not be joined across it.
type pointView struct {
	AgeMonths float64  `json:"age_months"`
	Metric    float64  `json:"metric"`
	Imperial  *float64 `json:"imperial"`
	Gap       bool     `json:"gap,omitempty"`
}

type lineView struct {
	Percentile float64     `json:"percentile"`
	Points     []pointView `json:"points"`
}

type curvesView struct {
	Source           string     `json:"source"`
	Measure          string     `json:"measure"`
	Sex              string     `json:"sex"`
	Covered          bool       `json:"covered"`
	Reason           *string    `json:"reason"`
	FromMonths       float64    `json:"from_months"`
	ToMonths         float64    `json:"to_months"`
	Lines            []lineView `json:"lines"`
	CorrectedAgeAxis bool       `json:"corrected_age_axis"`
}

// imperialOf is the display-only pounds or inches of a curve value; nil for BMI, which has no imperial unit.
func imperialOf(m growth.Measure, v float64) *float64 {
	switch m {
	case growth.Weight:
		v *= constants.PoundsPerKG
	case growth.Length, growth.Head:
		v /= constants.CMPerInch
	default:
		return nil
	}
	return &v
}

// monthsParam reads a finite month count in [0, CurveMaxMonths]; def when the key is absent.
func monthsParam(r *http.Request, key string, def float64) (float64, bool) {
	q := r.URL.Query()
	if !q.Has(key) {
		return def, true
	}
	v, err := strconv.ParseFloat(q.Get(key), 64)
	if err != nil || math.IsNaN(v) || v < 0 || v > constants.CurveMaxMonths {
		return 0, false
	}
	return v, true
}

func validMeasure(s string) (growth.Measure, bool) {
	for _, m := range growthMeasures {
		if string(m) == s {
			return m, true
		}
	}
	return "", false
}

func (d Deps) growthCurves(w http.ResponseWriter, r *http.Request) {
	kid, loc, ok := d.growthKid(w, r)
	if !ok {
		return
	}
	m, ok := validMeasure(r.URL.Query().Get("measure"))
	if !ok {
		respondInvalid(w, "measure")
		return
	}
	today, _ := kidtime.DayBounds(loc, d.Now())
	basis, err := kidtime.AgeBasisOn(kid.BirthDate, kid.GestationWeeks, kid.GestationDays, today.Format(time.DateOnly), loc)
	if err != nil {
		respondInternal(w, err)
		return
	}
	defTo := min(max(constants.CurveDefaultMinToMonths,
		float64(basis.ScoringDays)/constants.DaysPerMonth+constants.CurveDefaultAheadMonths), constants.CurveMaxMonths)
	from, ok := monthsParam(r, "from_months", 0)
	if !ok {
		respondInvalid(w, "from_months")
		return
	}
	to, ok := monthsParam(r, "to_months", defTo)
	if !ok {
		respondInvalid(w, "to_months")
		return
	}
	if from > to {
		respondInvalid(w, "from_months")
		return
	}
	src, sex := growth.Source(kid.GrowthSource), growth.Sex(kid.Sex)
	v := curvesView{Source: kid.GrowthSource, Measure: string(m), Sex: kid.Sex, FromMonths: from, ToMonths: to,
		Lines: []lineView{}, CorrectedAgeAxis: kidtime.CorrectionDays(kid.GestationWeeks, kid.GestationDays) > 0}
	curves, err := growth.Curves(src, m, sex, from, to)
	if errors.Is(err, growth.ErrNotCovered) {
		reason := "not_covered"
		v.Reason = &reason
		respondJSON(w, http.StatusOK, v)
		return
	}
	if err != nil {
		respondInternal(w, err)
		return
	}
	v.Covered = true
	for _, c := range curves {
		line := lineView{Percentile: c.Percentile, Points: make([]pointView, len(c.Points))}
		for i, p := range c.Points {
			line.Points[i] = pointView{AgeMonths: p.AgeMonths, Metric: p.Value, Imperial: imperialOf(m, p.Value), Gap: p.Gap}
		}
		v.Lines = append(v.Lines, line)
	}
	first := curves[0].Points
	v.FromMonths, v.ToMonths = first[0].AgeMonths, first[len(first)-1].AgeMonths
	respondJSON(w, http.StatusOK, v)
}

type coverageView struct {
	Measure    string  `json:"measure"`
	FromMonths float64 `json:"from_months"`
	ToMonths   float64 `json:"to_months"`
}

type sourceView struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Attribution string         `json:"attribution"`
	Licence     string         `json:"licence"`
	Version     string         `json:"version"`
	OriginURL   string         `json:"origin_url"`
	Retrieved   string         `json:"retrieved"`
	Coverage    []coverageView `json:"coverage"`
}

// newSourceView spans both sexes' tables, so a measure is listed when either sex has one.
func newSourceView(info growth.Info) sourceView {
	v := sourceView{ID: string(info.ID), Name: info.Name, Attribution: info.Attribution, Licence: info.Licence,
		Version: info.Version, OriginURL: info.OriginURL, Retrieved: info.Retrieved, Coverage: []coverageView{}}
	for _, m := range growthMeasures {
		c := coverageView{Measure: string(m), FromMonths: math.Inf(1), ToMonths: math.Inf(-1)}
		found := false
		for _, sex := range []growth.Sex{growth.Female, growth.Male} {
			if from, to, ok := growth.Coverage(info.ID, m, sex); ok {
				c.FromMonths, c.ToMonths, found = min(c.FromMonths, from), max(c.ToMonths, to), true
			}
		}
		if found {
			v.Coverage = append(v.Coverage, c)
		}
	}
	return v
}

func (d Deps) growthSources(w http.ResponseWriter, _ *http.Request) {
	views := []sourceView{}
	for _, s := range growth.Sources {
		if info, ok := growth.SourceInfo(s); ok {
			views = append(views, newSourceView(info))
		}
	}
	respondJSON(w, http.StatusOK, map[string]any{"sources": views})
}
