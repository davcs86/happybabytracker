package httpapi

import (
	"math"
	"net/http"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/growth"
	"github.com/davcs86/happybabytracker/api/internal/kidtime"
)

func (k *kit) curves(token, kid, query string) map[string]any {
	k.t.Helper()
	rec := k.do(http.MethodGet, "/v1/kids/"+kid+"/growth/curves?"+query, token, nil)
	if rec.Code != http.StatusOK {
		k.t.Fatalf("curves %s: %d %q", query, rec.Code, rec.Body.String())
	}
	return bodyMap(k.t, rec)
}

func linesOf(t *testing.T, c map[string]any) []map[string]any {
	t.Helper()
	raw, ok := c["lines"].([]any)
	if !ok {
		t.Fatalf("lines = %v, want an array", c["lines"])
	}
	lines := make([]map[string]any, len(raw))
	for i, l := range raw {
		lines[i] = l.(map[string]any)
	}
	return lines
}

func pointsOf(line map[string]any) []map[string]any {
	raw := line["points"].([]any)
	points := make([]map[string]any, len(raw))
	for i, p := range raw {
		points[i] = p.(map[string]any)
	}
	return points
}

func percentilesOf(t *testing.T, c map[string]any) []float64 {
	t.Helper()
	var ps []float64
	for _, l := range linesOf(t, c) {
		ps = append(ps, l["percentile"].(float64))
	}
	return ps
}

func near(a, b float64) bool { return math.Abs(a-b) <= 1e-9*math.Max(1, math.Abs(b)) }

// wantEngineCurves checks every point against growth.Curves for the same range, including the imperial column.
func wantEngineCurves(t *testing.T, c map[string]any, src growth.Source, m growth.Measure, sex growth.Sex, from, to float64) {
	t.Helper()
	want, err := growth.Curves(src, m, sex, from, to)
	if err != nil {
		t.Fatalf("engine: %v", err)
	}
	lines := linesOf(t, c)
	if len(lines) != len(want) {
		t.Fatalf("%d lines, want %d", len(lines), len(want))
	}
	for i, l := range lines {
		points := pointsOf(l)
		if l["percentile"] != want[i].Percentile || len(points) != len(want[i].Points) {
			t.Fatalf("line %d: percentile %v with %d points, want %v with %d", i, l["percentile"], len(points),
				want[i].Percentile, len(want[i].Points))
		}
		for j, p := range points {
			w := want[i].Points[j]
			if p["age_months"] != w.AgeMonths || !near(p["metric"].(float64), w.Value) {
				t.Fatalf("line %d point %d = %v, want %v", i, j, p, w)
			}
			var imperial any
			switch m {
			case growth.Weight:
				imperial = w.Value * constants.PoundsPerKG
			case growth.Length, growth.Head:
				imperial = w.Value / constants.CMPerInch
			}
			if got, ok := p["imperial"].(float64); imperial == nil && p["imperial"] != nil ||
				imperial != nil && (!ok || !near(got, imperial.(float64))) {
				t.Fatalf("line %d point %d imperial = %v, want %v", i, j, p["imperial"], imperial)
			}
			if gap, set := p["gap"]; w.Gap != set || set && gap != true {
				t.Fatalf("line %d point %d gap = %v (set %v), want %v", i, j, gap, set, w.Gap)
			}
		}
	}
}

func TestCurvesShape(t *testing.T) {
	k := newKit(t)
	ana := k.asParent("ana")
	kid := k.createKid(ana, map[string]any{"sex": "male"}) // born 2026-03-04: 8 months at t0
	c := k.curves(ana, kid, "measure=weight&to_months=36")
	sameKeys(t, c, "source", "measure", "sex", "covered", "reason", "from_months", "to_months", "lines", "corrected_age_axis")
	if c["source"] != "who" || c["measure"] != "weight" || c["sex"] != "male" || c["covered"] != true || c["reason"] != nil ||
		c["from_months"] != float64(0) || c["to_months"] != float64(36) || c["corrected_age_axis"] != false {
		t.Errorf("curves head = %v", c)
	}
	if ps := percentilesOf(t, c); !equalJSON(ps, []float64{3, 15, 50, 85, 97}) {
		t.Errorf("percentiles = %v", ps)
	}
	lines := linesOf(t, c)
	var ages []float64
	for a := 0.0; a < 24; a += 0.5 {
		ages = append(ages, a)
	}
	for a := 24.0; a <= 36; a++ {
		ages = append(ages, a)
	}
	for i, l := range lines {
		points := pointsOf(l)
		if len(points) != len(ages) {
			t.Fatalf("line %d: %d points, want %d", i, len(points), len(ages))
		}
		for j, p := range points {
			sameKeys(t, p, "age_months", "metric", "imperial")
			if p["age_months"] != ages[j] {
				t.Errorf("line %d point %d age %v, want %v", i, j, p["age_months"], ages[j])
			}
			if i > 0 && !(p["metric"].(float64) > pointsOf(lines[i-1])[j]["metric"].(float64)) {
				t.Errorf("age %v: line %d is not above line %d", ages[j], i, i-1)
			}
		}
	}
	wantEngineCurves(t, c, growth.SourceWHO, growth.Weight, growth.Male, 0, 36)
}

func TestCurvesCDCLines(t *testing.T) {
	k := newKit(t)
	ana := k.asParent("ana")
	kid := k.createKid(ana, map[string]any{"growth_source": "cdc"})
	c := k.curves(ana, kid, "measure=weight")
	if c["source"] != "cdc" || c["sex"] != "female" {
		t.Errorf("source %v sex %v, want cdc and female", c["source"], c["sex"])
	}
	if ps := percentilesOf(t, c); !equalJSON(ps, []float64{3, 5, 10, 25, 50, 75, 90, 95, 97}) {
		t.Errorf("weight percentiles = %v", ps)
	}
	wantEngineCurves(t, c, growth.SourceCDC, growth.Weight, growth.Female, 0, 24)
	bmi := k.curves(ana, kid, "measure=bmi&from_months=24&to_months=60")
	if ps := percentilesOf(t, bmi); !equalJSON(ps, []float64{3, 5, 10, 25, 50, 75, 85, 90, 95, 97}) {
		t.Errorf("bmi percentiles = %v", ps)
	}
	wantEngineCurves(t, bmi, growth.SourceCDC, growth.BMI, growth.Female, 24, 60)
	length := k.curves(ana, kid, "measure=length&to_months=48")
	wantEngineCurves(t, length, growth.SourceCDC, growth.Length, growth.Female, 0, 48)
}

func TestCurvesMarkSeams(t *testing.T) {
	k := newKit(t)
	ana := k.asParent("ana")
	kid := k.createKid(ana, nil)
	c := k.curves(ana, kid, "measure=length&from_months=20&to_months=30")
	wantEngineCurves(t, c, growth.SourceWHO, growth.Length, growth.Female, 20, 30)
	gaps := 0
	for _, p := range pointsOf(linesOf(t, c)[0]) {
		if p["gap"] == true {
			gaps++
		}
	}
	if gaps != 1 {
		t.Errorf("%d gap points across the WHO length/height seam, want 1", gaps)
	}
}

func TestCurvesClampToCoverage(t *testing.T) {
	k := newKit(t)
	ana := k.asParent("ana")
	kid := k.createKid(ana, map[string]any{"growth_source": "cdc"})
	c := k.curves(ana, kid, "measure=head&to_months=120")
	if c["covered"] != true || c["from_months"] != float64(0) || c["to_months"] != float64(36) {
		t.Errorf("covered %v range %v–%v, want true 0–36", c["covered"], c["from_months"], c["to_months"])
	}
	for _, l := range linesOf(t, c) {
		points := pointsOf(l)
		if last := points[len(points)-1]["age_months"]; last != float64(36) {
			t.Errorf("last age %v, want 36", last)
		}
	}
	wantEngineCurves(t, c, growth.SourceCDC, growth.Head, growth.Female, 0, 120)
}

func TestCurvesNotCovered(t *testing.T) {
	k := newKit(t)
	ana := k.asParent("ana")
	cdc := k.createKid(ana, map[string]any{"growth_source": "cdc"})
	c := k.curves(ana, cdc, "measure=bmi&to_months=20")
	if c["covered"] != false || c["reason"] != "not_covered" || c["from_months"] != float64(0) || c["to_months"] != float64(20) {
		t.Errorf("CDC BMI under 24 months = %v", c)
	}
	if lines, ok := c["lines"].([]any); !ok || len(lines) != 0 {
		t.Errorf("lines = %v, want []", c["lines"])
	}
	who := k.createKid(ana, map[string]any{"name": "Second"})
	if c := k.curves(ana, who, "measure=head&from_months=72&to_months=96"); c["covered"] != false || c["reason"] != "not_covered" {
		t.Errorf("WHO head at 6 years = %v", c)
	}
}

func TestCurvesDefaultRange(t *testing.T) {
	k := newKit(t)
	ana := k.asParent("ana")
	baby := k.createKid(ana, nil)
	if c := k.curves(ana, baby, "measure=weight"); c["from_months"] != float64(0) || c["to_months"] != float64(24) {
		t.Errorf("8-month-old default range %v–%v, want 0–24", c["from_months"], c["to_months"])
	}
	old := k.createKid(ana, map[string]any{"name": "Old", "birth_date": "2024-01-15", "time_zone": "UTC"})
	days, _ := kidtime.AgeDays("2024-01-15", "2026-11-20", time.UTC)
	want := float64(days)/constants.DaysPerMonth + constants.CurveDefaultAheadMonths
	if c := k.curves(ana, old, "measure=weight"); c["to_months"] != want {
		t.Errorf("34-month-old default to_months %v, want %v", c["to_months"], want)
	}
	// Born at 35 weeks 2 days, 33 days early: the default range follows the corrected age.
	pre := k.createKid(ana, map[string]any{"name": "Pre", "birth_date": "2025-01-10", "time_zone": "UTC",
		"gestation_weeks": 35, "gestation_days": 2})
	basis, _ := kidtime.AgeBasisOn("2025-01-10", ptr(35), ptr(2), "2026-11-20", time.UTC)
	c := k.curves(ana, pre, "measure=weight")
	if want := float64(basis.ScoringDays)/constants.DaysPerMonth + constants.CurveDefaultAheadMonths; c["to_months"] != want ||
		c["corrected_age_axis"] != true {
		t.Errorf("preterm default to_months %v axis %v, want %v and true", c["to_months"], c["corrected_age_axis"], want)
	}
	if c := k.curves(ana, old, "measure=weight&from_months=30"); c["from_months"] != float64(30) {
		t.Errorf("from_months = %v, want 30", c["from_months"])
	}
}

func ptr(n int) *int { return &n }

func TestCurvesCorrectedAxisEndsAtSecondBirthday(t *testing.T) {
	k := newKit(t)
	ana := k.asParent("ana")
	pre := k.createKid(ana, map[string]any{"name": "Pre", "birth_date": "2025-01-10", "time_zone": "UTC",
		"gestation_weeks": 35, "gestation_days": 2})
	term := k.createKid(ana, map[string]any{"name": "Term", "birth_date": "2025-01-10", "time_zone": "UTC"})
	for _, c := range []struct {
		name      string
		now       time.Time
		pre, term bool
	}{
		{"at 22 months", t0, true, false},
		{"the day before the second birthday", time.Date(2027, 1, 9, 12, 0, 0, 0, time.UTC), true, false},
		{"the second birthday", time.Date(2027, 1, 10, 0, 0, 0, 0, time.UTC), false, false},
		{"after the second birthday", time.Date(2028, 3, 1, 12, 0, 0, 0, time.UTC), false, false},
	} {
		k.now = c.now
		token := k.asParent("ana")
		for kid, want := range map[string]bool{pre: c.pre, term: c.term} {
			if got := k.curves(token, kid, "measure=weight")["corrected_age_axis"]; got != want {
				t.Errorf("%s: kid %s corrected_age_axis = %v, want %v", c.name, kid, got, want)
			}
		}
	}
}

func TestCurvesBadParams(t *testing.T) {
	k := newKit(t)
	ana := k.asParent("ana")
	kid := k.createKid(ana, nil)
	for query, field := range map[string]string{
		"":                                          "measure",
		"measure=":                                  "measure",
		"measure=height":                            "measure",
		"measure=WEIGHT":                            "measure",
		"measure=weight&from_months=-1":             "from_months",
		"measure=weight&from_months=241":            "from_months",
		"measure=weight&from_months=abc":            "from_months",
		"measure=weight&from_months=":               "from_months",
		"measure=weight&from_months=NaN":            "from_months",
		"measure=weight&to_months=Inf":              "to_months",
		"measure=weight&to_months=-0.5":             "to_months",
		"measure=weight&to_months=240.5":            "to_months",
		"measure=weight&to_months=1e400":            "to_months",
		"measure=weight&from_months=10&to_months=5": "from_months",
	} {
		rec := k.do(http.MethodGet, "/v1/kids/"+kid+"/growth/curves?"+query, ana, nil)
		if rec.Code != http.StatusUnprocessableEntity || !equalJSON(bodyMap(t, rec), map[string]any{"error": "invalid", "field": field}) {
			t.Errorf("%q: %d %q, want 422 invalid %s", query, rec.Code, rec.Body.String(), field)
		}
	}
	if c := k.curves(ana, kid, "measure=weight&from_months=240&to_months=240"); c["covered"] != false {
		t.Errorf("WHO weight at 240 months = %v, want not covered", c["covered"])
	}
}

func TestCurvesRouteIsNotAnEntry(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	if c := k.curves(ana, kid, "measure=head"); c["measure"] != "head" {
		t.Errorf("curves = %v", c)
	}
	id := k.createGrowth(ana, kid, nil)["id"].(string)
	if got := k.getGrowth(ana, kid, id); got["id"] != id {
		t.Errorf("entry = %v, want %s", got, id)
	}
	if rec := k.do(http.MethodDelete, "/v1/kids/"+kid+"/growth/curves", ana, nil); rec.Code != http.StatusNotFound {
		t.Errorf("DELETE curves: %d, want 404 as for any unknown entry", rec.Code)
	}
}

func (k *kit) sources(token string) []map[string]any {
	k.t.Helper()
	rec := k.do(http.MethodGet, "/v1/growth-sources", token, nil)
	if rec.Code != http.StatusOK {
		k.t.Fatalf("growth sources: %d %q", rec.Code, rec.Body.String())
	}
	b := bodyMap(k.t, rec)
	sameKeys(k.t, b, "sources")
	var out []map[string]any
	for _, s := range b["sources"].([]any) {
		out = append(out, s.(map[string]any))
	}
	return out
}

func TestSourcesList(t *testing.T) {
	k := newKit(t)
	sources := k.sources(k.asParent("ana"))
	if len(sources) != 2 {
		t.Fatalf("%d sources, want 2", len(sources))
	}
	whoHead := 1856 / constants.DaysPerMonth
	wantCoverage := map[string][]map[string]any{
		"who": {
			{"measure": "weight", "from_months": 0, "to_months": 120},
			{"measure": "length", "from_months": 0, "to_months": 228},
			{"measure": "head", "from_months": 0, "to_months": whoHead},
			{"measure": "bmi", "from_months": 0, "to_months": 228},
		},
		"cdc": {
			{"measure": "weight", "from_months": 0, "to_months": 240},
			{"measure": "length", "from_months": 0, "to_months": 240},
			{"measure": "head", "from_months": 0, "to_months": 36},
			{"measure": "bmi", "from_months": 24, "to_months": 240.5},
		},
	}
	for i, id := range []growth.Source{growth.SourceWHO, growth.SourceCDC} {
		s := sources[i]
		sameKeys(t, s, "id", "name", "attribution", "licence", "version", "origin_url", "retrieved", "coverage")
		info, _ := growth.SourceInfo(id)
		if s["id"] != string(id) || s["name"] != info.Name || s["attribution"] != info.Attribution || s["licence"] != info.Licence ||
			s["version"] != info.Version || s["origin_url"] != info.OriginURL || s["retrieved"] != info.Retrieved {
			t.Errorf("source %d = %v, want %+v", i, s, info)
		}
		if s["licence"] == "" || s["version"] == "" {
			t.Errorf("%s: licence %q version %q, want both", id, s["licence"], s["version"])
		}
		if !equalJSON(s["coverage"], wantCoverage[string(id)]) {
			t.Errorf("%s coverage = %v, want %v", id, s["coverage"], wantCoverage[string(id)])
		}
	}
}

func TestSourcesNeverOffersUKWHO(t *testing.T) {
	k := newKit(t)
	var ids []any
	for _, s := range k.sources(k.asParent("ana")) {
		ids = append(ids, s["id"])
	}
	if !equalJSON(ids, []string{"who", "cdc"}) {
		t.Errorf("ids = %v, want exactly who and cdc", ids)
	}
	if rec := k.do(http.MethodGet, "/v1/growth-sources", k.asAdmin(), nil); rec.Code != http.StatusForbidden {
		t.Errorf("admin: %d, want 403", rec.Code)
	}
}
