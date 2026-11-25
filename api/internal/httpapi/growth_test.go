package httpapi

import (
	"context"
	"math"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/growth"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/units"
)

func amountBody(unit string, amount int64) map[string]any {
	return map[string]any{"unit": unit, "amount": amount}
}

func growthBody(over map[string]any) map[string]any {
	return withOverrides(map[string]any{
		"measured_on": "2026-06-01",
		"weight":      amountBody("kg", 5000),
		"length":      amountBody("cm", 600),
		"head":        amountBody("cm", 400),
		"note":        "first",
	}, over)
}

// createGrowth posts a measurement and returns its view; the status must be 201.
func (k *kit) createGrowth(token, kid string, over map[string]any) map[string]any {
	k.t.Helper()
	rec := k.do(http.MethodPost, "/v1/kids/"+kid+"/growth", token, growthBody(over))
	if rec.Code != http.StatusCreated {
		k.t.Fatalf("create growth: %d %q", rec.Code, rec.Body.String())
	}
	return bodyMap(k.t, rec)
}

func (k *kit) getGrowth(token, kid, id string) map[string]any {
	k.t.Helper()
	rec := k.do(http.MethodGet, "/v1/kids/"+kid+"/growth/"+id, token, nil)
	if rec.Code != http.StatusOK {
		k.t.Fatalf("get growth: %d %q", rec.Code, rec.Body.String())
	}
	return bodyMap(k.t, rec)
}

func scoresOf(m map[string]any) map[string]any { return m["scores"].(map[string]any) }

func round(x, scale float64) float64 { return math.Round(x*scale) / scale }

// wantScored checks a score view against the engine's score at days, rounded as the API rounds.
func wantScored(t *testing.T, name string, got any, src growth.Source, m growth.Measure, sex growth.Sex, days int, value float64) {
	t.Helper()
	s, err := growth.ScoreOf(src, m, sex, days, value)
	if err != nil {
		t.Fatalf("%s: engine: %v", name, err)
	}
	want := map[string]any{"scored": true, "z": round(s.Z, 100), "percentile": round(s.Percentile, 10)}
	if m == growth.BMI {
		want["value"] = round(value, 100)
	}
	if !equalJSON(got, want) {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

func wantUnscored(t *testing.T, name string, got any, reason string, extra ...float64) {
	t.Helper()
	want := map[string]any{"scored": false, "reason": reason}
	if len(extra) > 0 {
		want["value"] = round(extra[0], 100)
	}
	if !equalJSON(got, want) {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

func bmiOf(grams, mm float64) float64 { return grams * 1000 / (mm * mm) }

func TestGrowthRoutesRoles(t *testing.T) {
	k := newKit(t)
	kid, ana, bo := k.trackerKid()
	base := "/v1/kids/" + kid + "/growth"
	w, _ := units.NewWeight(units.UnitKG, 5000)
	l, _ := units.NewLength(units.UnitCM, 600)
	h, _ := units.NewLength(units.UnitCM, 400)

	for name, token := range map[string]string{"guardian": ana, "caregiver": bo} {
		m := k.createGrowth(token, kid, nil)
		sameKeys(t, m, "id", "measured_on", "weight", "length", "head", "note", "age", "scores", "visit_id")
		id := m["id"].(string)
		if m["measured_on"] != "2026-06-01" || m["note"] != "first" ||
			!equalJSON(m["weight"], map[string]any{"grams": w.Grams, "oz_hundredths": w.OzHundredths, "typed_unit": "kg"}) ||
			!equalJSON(m["length"], map[string]any{"mm": l.MM, "in_hundredths": l.InHundredths, "typed_unit": "cm"}) ||
			!equalJSON(m["head"], map[string]any{"mm": h.MM, "in_hundredths": h.InHundredths, "typed_unit": "cm"}) ||
			!equalJSON(m["age"], map[string]any{"days": 89, "corrected_days": nil}) {
			t.Errorf("%s create view = %v", name, m)
		}
		sc := scoresOf(m)
		sameKeys(t, sc, "weight", "length", "head", "bmi")
		wantScored(t, name+" weight", sc["weight"], growth.SourceWHO, growth.Weight, growth.Female, 89, 5)
		wantScored(t, name+" length", sc["length"], growth.SourceWHO, growth.Length, growth.Female, 89, 60)
		wantScored(t, name+" head", sc["head"], growth.SourceWHO, growth.Head, growth.Female, 89, 40)
		wantScored(t, name+" bmi", sc["bmi"], growth.SourceWHO, growth.BMI, growth.Female, 89, bmiOf(5000, 600))

		one := base + "/" + id
		if got := k.getGrowth(token, kid, id); !equalJSON(got, m) {
			t.Errorf("%s get = %v, want %v", name, got, m)
		}
		rec := k.do(http.MethodPatch, one, token, map[string]any{"note": "edited", "head": nil})
		if rec.Code != http.StatusOK {
			t.Fatalf("%s patch: %d %q", name, rec.Code, rec.Body.String())
		}
		if p := bodyMap(t, rec); p["note"] != "edited" || p["id"] != id || p["head"] != nil || scoresOf(p)["head"] != nil ||
			!equalJSON(p["weight"], m["weight"]) {
			t.Errorf("%s patch view = %v", name, p)
		}
		rec = k.do(http.MethodGet, base, token, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s list: %d %q", name, rec.Code, rec.Body.String())
		}
		list := bodyMap(t, rec)
		sameKeys(t, list, "growth")
		if rows := list["growth"].([]any); len(rows) != 1 || !equalJSON(rows[0], k.getGrowth(token, kid, id)) {
			t.Errorf("%s list = %v", name, rows)
		}
		if rec := k.do(http.MethodDelete, one, token, nil); rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
			t.Errorf("%s delete: %d %q", name, rec.Code, rec.Body.String())
		}
		wantErr(t, k.do(http.MethodGet, one, token, nil), http.StatusNotFound, "not_found")
		if rows := bodyMap(t, k.do(http.MethodGet, base, token, nil))["growth"].([]any); len(rows) != 0 {
			t.Errorf("%s list after delete = %v", name, rows)
		}
	}
}

func TestGrowthRoutesRefuseOutsiders(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	other := k.createKid(ana, map[string]any{"name": "Second"})
	xavi, admin := k.asParent("xavi"), k.asAdmin()
	id := k.createGrowth(ana, kid, nil)["id"].(string)
	otherID := k.createGrowth(ana, other, nil)["id"].(string)

	type call struct {
		method, path string
		body         any
	}
	calls := func(kidPath, entry string) []call {
		return []call{
			{http.MethodGet, "/v1/kids/" + kidPath + "/growth", nil},
			{http.MethodPost, "/v1/kids/" + kidPath + "/growth", growthBody(nil)},
			{http.MethodGet, "/v1/kids/" + kidPath + "/growth/" + entry, nil},
			{http.MethodPatch, "/v1/kids/" + kidPath + "/growth/" + entry, map[string]any{"note": "x"}},
			{http.MethodDelete, "/v1/kids/" + kidPath + "/growth/" + entry, nil},
		}
	}
	for i, c := range calls(kid, id) {
		unknown := calls(model.NewID(), id)[i]
		wantBody := k.do(unknown.method, unknown.path, xavi, unknown.body).Body.String()
		if rec := k.do(c.method, c.path, xavi, c.body); rec.Code != http.StatusNotFound || rec.Body.String() != wantBody {
			t.Errorf("no grant %s %s: %d %q, want the unknown-kid 404 %q", c.method, c.path, rec.Code, rec.Body.String(), wantBody)
		}
		wantErr(t, k.do(c.method, c.path, admin, c.body), http.StatusForbidden, "forbidden")
	}

	// An entry of the caller's other kid, and a malformed id, are the same 404 as an unknown entry.
	for _, entry := range []string{otherID, model.NewID(), "not an id", strings.Repeat("a", 65)} {
		for _, c := range calls(kid, url.PathEscape(entry))[2:] {
			wantErr(t, k.do(c.method, c.path, ana, c.body), http.StatusNotFound, "not_found")
		}
	}
	for kidID, entry := range map[string]string{kid: id, other: otherID} {
		if m := k.getGrowth(ana, kidID, entry); m["note"] != "first" {
			t.Errorf("entry %s changed: %v", entry, m)
		}
	}
}

func TestGrowthValidation(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	base := "/v1/kids/" + kid + "/growth"
	id := k.createGrowth(ana, kid, nil)["id"].(string)
	stored := k.do(http.MethodGet, base+"/"+id, ana, nil).Body.String()

	cases := []struct {
		name       string
		over       map[string]any
		field      string
		createOnly bool // a missing key on PATCH keeps the stored value
	}{
		{"after the kid's today", map[string]any{"measured_on": "2026-11-21"}, "measured_on", false},
		{"before the birth date", map[string]any{"measured_on": "2026-03-03"}, "measured_on", false},
		{"not a date", map[string]any{"measured_on": "yesterday"}, "measured_on", false},
		{"month 13", map[string]any{"measured_on": "2026-13-01"}, "measured_on", false},
		{"February 30", map[string]any{"measured_on": "2026-02-30"}, "measured_on", false},
		{"one-digit day", map[string]any{"measured_on": "2026-06-1"}, "measured_on", false},
		{"an instant", map[string]any{"measured_on": "2026-06-01T00:00:00Z"}, "measured_on", false},
		{"empty date", map[string]any{"measured_on": ""}, "measured_on", false},
		{"no date", map[string]any{"measured_on": nil}, "measured_on", true},
		{"date first, then weight", map[string]any{"measured_on": "x", "weight": amountBody("kg", 0)}, "measured_on", false},
		{"weight below the range", map[string]any{"weight": amountBody("kg", constants.MinWeightGrams-1)}, "weight", false},
		{"weight above the range", map[string]any{"weight": amountBody("kg", constants.MaxWeightGrams+1)}, "weight", false},
		{"weight in pounds above the range", map[string]any{"weight": amountBody("lb", 900000)}, "weight", false},
		{"weight 0", map[string]any{"weight": amountBody("kg", 0)}, "weight", false},
		{"weight negative", map[string]any{"weight": amountBody("kg", -1)}, "weight", false},
		{"weight huge", map[string]any{"weight": amountBody("lb", constants.MaxWeightAmount+1)}, "weight", false},
		{"weight unit unknown", map[string]any{"weight": amountBody("stone", 5000)}, "weight", false},
		{"weight in a length unit", map[string]any{"weight": amountBody("cm", 5000)}, "weight", false},
		{"weight without an amount", map[string]any{"weight": map[string]any{"unit": "kg"}}, "weight", false},
		{"weight first, then length", map[string]any{"weight": amountBody("kg", 0), "length": amountBody("cm", 0)}, "weight", false},
		{"length below the range", map[string]any{"length": amountBody("cm", constants.MinLengthMM-1)}, "length", false},
		{"length above the range", map[string]any{"length": amountBody("cm", constants.MaxLengthMM+1)}, "length", false},
		{"length 0", map[string]any{"length": amountBody("in", 0)}, "length", false},
		{"length negative", map[string]any{"length": amountBody("cm", -5)}, "length", false},
		{"length huge", map[string]any{"length": amountBody("in", constants.MaxLengthAmount+1)}, "length", false},
		{"length in a weight unit", map[string]any{"length": amountBody("kg", 600)}, "length", false},
		{"length first, then head", map[string]any{"length": amountBody("cm", 0), "head": amountBody("cm", 0)}, "length", false},
		{"head below the range", map[string]any{"head": amountBody("cm", constants.MinHeadMM-1)}, "head", false},
		{"head above the range", map[string]any{"head": amountBody("cm", constants.MaxHeadMM+1)}, "head", false},
		{"head 0", map[string]any{"head": amountBody("cm", 0)}, "head", false},
		{"head unit unknown", map[string]any{"head": amountBody("ft", 400)}, "head", false},
		{"head first, then note", map[string]any{"head": amountBody("cm", 0), "note": strings.Repeat("ñ", constants.MaxNoteLen+1)}, "head", false},
		{"note over the limit", map[string]any{"note": strings.Repeat("ñ", constants.MaxNoteLen+1)}, "note", false},
		{"note before measurements", map[string]any{"note": strings.Repeat("ñ", constants.MaxNoteLen+1), "weight": nil, "length": nil, "head": nil}, "note", true},
		{"no measure at all", map[string]any{"weight": nil, "length": nil, "head": nil}, "measurements", true},
	}
	for _, c := range cases {
		wantInvalid2(t, c.name+" (create)", k.do(http.MethodPost, base, ana, growthBody(c.over)), c.field)
		if !c.createOnly {
			wantInvalid2(t, c.name+" (patch)", k.do(http.MethodPatch, base+"/"+id, ana, c.over), c.field)
		}
	}
	wantInvalid2(t, "every measure null (create)",
		k.do(http.MethodPost, base, ana, `{"measured_on":"2026-06-01","weight":null,"length":null,"head":null}`), "measurements")
	wantInvalid2(t, "every measure null (patch)",
		k.do(http.MethodPatch, base+"/"+id, ana, `{"weight":null,"length":null,"head":null}`), "measurements")
	if got := k.do(http.MethodGet, base+"/"+id, ana, nil).Body.String(); got != stored {
		t.Errorf("a refused patch changed the entry:\n got %s\nwant %s", got, stored)
	}

	// Clearing the last remaining measure is refused.
	only := k.createGrowth(ana, kid, map[string]any{"length": nil, "head": nil})["id"].(string)
	wantInvalid2(t, "null clearing the last measure", k.do(http.MethodPatch, base+"/"+only, ana, `{"weight":null}`), "measurements")
	if m := k.getGrowth(ana, kid, only); m["weight"] == nil {
		t.Errorf("the refused clear removed the weight: %v", m)
	}
	if n := len(bodyMap(t, k.do(http.MethodGet, base, ana, nil))["growth"].([]any)); n != 2 {
		t.Errorf("%d measurements after the refused creates, want 2", n)
	}

	for _, c := range []struct {
		name string
		over map[string]any
	}{
		{"the kid's today", map[string]any{"measured_on": "2026-11-20"}},
		{"the birth date", map[string]any{"measured_on": "2026-03-04"}},
		{"the lowest weight", map[string]any{"weight": amountBody("kg", constants.MinWeightGrams)}},
		{"the highest weight", map[string]any{"weight": amountBody("kg", constants.MaxWeightGrams)}},
		{"the lowest length", map[string]any{"length": amountBody("cm", constants.MinLengthMM)}},
		{"the highest length", map[string]any{"length": amountBody("cm", constants.MaxLengthMM)}},
		{"the lowest head", map[string]any{"head": amountBody("cm", constants.MinHeadMM)}},
		{"the highest head", map[string]any{"head": amountBody("cm", constants.MaxHeadMM)}},
		{"a note at the limit", map[string]any{"note": strings.Repeat("ñ", constants.MaxNoteLen)}},
		{"only a length", map[string]any{"weight": nil, "head": nil}},
		{"only a head, in inches", map[string]any{"weight": nil, "length": nil, "head": amountBody("in", 1575)}},
		{"no note", map[string]any{"note": nil}},
	} {
		if rec := k.do(http.MethodPost, base, ana, growthBody(c.over)); rec.Code != http.StatusCreated {
			t.Errorf("%s: %d %q", c.name, rec.Code, rec.Body.String())
		}
	}

	// The kid's today is its local date: 23:30 UTC is already the next day in Madrid.
	k.now = time.Date(2026, 11, 20, 23, 30, 0, 0, time.UTC)
	if rec := k.do(http.MethodPost, base, ana, growthBody(map[string]any{"measured_on": "2026-11-21"})); rec.Code != http.StatusCreated {
		t.Errorf("the kid's local today: %d %q", rec.Code, rec.Body.String())
	}
	wantInvalid2(t, "the kid's local tomorrow", k.do(http.MethodPost, base, ana, growthBody(map[string]any{"measured_on": "2026-11-22"})), "measured_on")
	k.now = t0

	for _, method := range []string{http.MethodPost, http.MethodPatch} {
		path := base
		if method == http.MethodPatch {
			path += "/" + id
		}
		for body, field := range map[string]string{
			`{"measured_on":"2026-06-01","weight":{"unit":"kg","amount":5000},"colour":"red"}`: "colour",
			`{"measured_on":"2026-06-01","weight":{"unit":"kg","amount":5000,"precise":true}}`: "precise",
		} {
			rec := k.do(method, path, ana, body)
			if rec.Code != http.StatusUnprocessableEntity || strings.TrimSpace(rec.Body.String()) != `{"error":"unknown_field","field":"`+field+`"}` {
				t.Errorf("%s unknown field %s: %d %q", method, field, rec.Code, rec.Body.String())
			}
		}
		for _, over := range []map[string]any{{"weight": "heavy"}, {"measured_on": 20260601}, {"head": "cm"}} {
			wantErr(t, k.do(method, path, ana, growthBody(over)), http.StatusBadRequest, "bad_request")
		}
	}
}

func TestGrowthTypedValuesStayExact(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	lb := map[string]any{"grams": 3728.0, "oz_hundredths": 13150.0, "typed_unit": "lb"}
	m := k.createGrowth(ana, kid, map[string]any{"weight": amountBody("lb", 13150)}) // 8 lb 3.5 oz
	one := "/v1/kids/" + kid + "/growth/" + m["id"].(string)
	if !equalJSON(m["weight"], lb) {
		t.Fatalf("created weight = %v", m["weight"])
	}
	for name, body := range map[string]map[string]any{
		"a note edit":            {"note": "only the note"},
		"the same typed amount":  {"weight": amountBody("lb", 13150)},
		"the same date resent":   {"measured_on": "2026-06-01"},
		"an empty patch":         {},
		"another measure edited": {"length": amountBody("cm", 610)},
	} {
		rec := k.do(http.MethodPatch, one, ana, body)
		if got := bodyMap(t, rec)["weight"]; rec.Code != http.StatusOK || !equalJSON(got, lb) {
			t.Errorf("after %s: %d weight %v", name, rec.Code, got)
		}
	}
	rec := k.do(http.MethodPatch, one, ana, map[string]any{"weight": amountBody("kg", 3728)})
	if got := bodyMap(t, rec)["weight"]; !equalJSON(got, map[string]any{"grams": 3728.0, "oz_hundredths": 13150.0, "typed_unit": "kg"}) {
		t.Errorf("after entering 3728 g: %v", got)
	}

	// A stored triple NewWeight would not produce survives an edit that resends the typed value.
	odd := model.Growth{MeasuredOn: "2026-06-01", Weight: &units.Weight{Grams: 3729, OzHundredths: 13150, Typed: units.UnitLB}}
	g, err := k.Store.CreateGrowth(context.Background(), kid, k.parentID("ana"), odd, t0)
	if err != nil {
		t.Fatal(err)
	}
	rec = k.do(http.MethodPatch, "/v1/kids/"+kid+"/growth/"+g.ID, ana, map[string]any{"weight": amountBody("lb", 13150), "note": "x"})
	if got := bodyMap(t, rec)["weight"]; rec.Code != http.StatusOK || !equalJSON(got, map[string]any{"grams": 3729.0, "oz_hundredths": 13150.0, "typed_unit": "lb"}) {
		t.Errorf("weight = %v (%d)", got, rec.Code)
	}
}

func TestScoresUseTheKidsSource(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	before := k.createGrowth(ana, kid, nil)
	id := before["id"].(string)
	wantScored(t, "who weight", scoresOf(before)["weight"], growth.SourceWHO, growth.Weight, growth.Female, 89, 5)

	if rec := k.do(http.MethodPatch, "/v1/kids/"+kid, ana, map[string]any{"growth_source": "cdc"}); rec.Code != http.StatusOK {
		t.Fatalf("switch source: %d %q", rec.Code, rec.Body.String())
	}
	after := k.getGrowth(ana, kid, id)
	sc := scoresOf(after)
	wantScored(t, "cdc weight", sc["weight"], growth.SourceCDC, growth.Weight, growth.Female, 89, 5)
	wantScored(t, "cdc length", sc["length"], growth.SourceCDC, growth.Length, growth.Female, 89, 60)
	wantScored(t, "cdc head", sc["head"], growth.SourceCDC, growth.Head, growth.Female, 89, 40)
	wantUnscored(t, "cdc bmi under 24 months", sc["bmi"], "not_covered", bmiOf(5000, 600))
	if zw, zc := scoresOf(before)["weight"].(map[string]any)["z"], sc["weight"].(map[string]any)["z"]; zw == zc {
		t.Errorf("weight z %v under both sources", zw)
	}
	for _, key := range []string{"id", "measured_on", "weight", "length", "head", "note", "age"} {
		if !equalJSON(after[key], before[key]) {
			t.Errorf("%s changed with the source: %v -> %v", key, before[key], after[key])
		}
	}
}

func TestCorrectedScoring(t *testing.T) {
	k := newKit(t)
	ana := k.asParent("ana")
	// Born at 35 weeks 2 days: 33 days early.
	kid := k.createKid(ana, map[string]any{"birth_date": "2024-06-10", "gestation_weeks": 35, "gestation_days": 2})

	for _, c := range []struct {
		name, on        string
		chrono, scoring int
		corrected       bool
	}{
		{"three months", "2024-09-10", 92, 59, true},
		{"the day before the second birthday", "2026-06-09", 729, 696, true},
		{"the second birthday", "2026-06-10", 730, 730, false},
	} {
		m := k.createGrowth(ana, kid, map[string]any{"measured_on": c.on, "head": nil})
		var corrected any
		if c.corrected {
			corrected = c.scoring
		}
		if !equalJSON(m["age"], map[string]any{"days": c.chrono, "corrected_days": corrected}) {
			t.Errorf("%s: age = %v", c.name, m["age"])
		}
		sc := scoresOf(m)
		wantScored(t, c.name+" weight", sc["weight"], growth.SourceWHO, growth.Weight, growth.Female, c.scoring, 5)
		wantScored(t, c.name+" length", sc["length"], growth.SourceWHO, growth.Length, growth.Female, c.scoring, 60)
		wantScored(t, c.name+" bmi", sc["bmi"], growth.SourceWHO, growth.BMI, growth.Female, c.scoring, bmiOf(5000, 600))
		if sc["head"] != nil {
			t.Errorf("%s: head score %v for a row without a head", c.name, sc["head"])
		}
	}

	m := k.createGrowth(ana, kid, map[string]any{"measured_on": "2024-06-20"})
	if !equalJSON(m["age"], map[string]any{"days": 10, "corrected_days": -23}) {
		t.Errorf("before term: age = %v", m["age"])
	}
	sc := scoresOf(m)
	for _, key := range []string{"weight", "length", "head"} {
		wantUnscored(t, "before term "+key, sc[key], "before_term")
	}
	wantUnscored(t, "before term bmi", sc["bmi"], "before_term", bmiOf(5000, 600))
}

func TestNotCoveredCells(t *testing.T) {
	k := newKit(t)
	ana := k.asParent("ana")

	cdc := k.createKid(ana, map[string]any{"birth_date": "2024-11-20", "growth_source": "cdc"})
	if growth.Covered(growth.SourceCDC, growth.BMI, growth.Female, 546) {
		t.Fatal("CDC covers BMI at 18 months; the case below proves nothing")
	}
	m := k.createGrowth(ana, cdc, map[string]any{"measured_on": "2026-05-20",
		"weight": amountBody("kg", 11000), "length": amountBody("cm", 810), "head": amountBody("cm", 470)})
	sc := scoresOf(m)
	wantUnscored(t, "cdc bmi at 18 months", sc["bmi"], "not_covered", bmiOf(11000, 810))
	wantScored(t, "cdc weight at 18 months", sc["weight"], growth.SourceCDC, growth.Weight, growth.Female, 546, 11)
	wantScored(t, "cdc length at 18 months", sc["length"], growth.SourceCDC, growth.Length, growth.Female, 546, 81)
	wantScored(t, "cdc head at 18 months", sc["head"], growth.SourceCDC, growth.Head, growth.Female, 546, 47)

	who := k.createKid(ana, map[string]any{"name": "Older", "birth_date": "2020-06-01", "sex": "male"})
	if growth.Covered(growth.SourceWHO, growth.Head, growth.Male, 2191) {
		t.Fatal("WHO covers head at 6 years; the case below proves nothing")
	}
	m = k.createGrowth(ana, who, map[string]any{"measured_on": "2026-06-01",
		"weight": amountBody("kg", 20000), "length": amountBody("cm", 1150), "head": amountBody("cm", 510)})
	sc = scoresOf(m)
	wantUnscored(t, "who head at 6 years", sc["head"], "not_covered")
	wantScored(t, "who weight at 6 years", sc["weight"], growth.SourceWHO, growth.Weight, growth.Male, 2191, 20)
	wantScored(t, "who height at 6 years", sc["length"], growth.SourceWHO, growth.Length, growth.Male, 2191, 115)
	wantScored(t, "who bmi at 6 years", sc["bmi"], growth.SourceWHO, growth.BMI, growth.Male, 2191, bmiOf(20000, 1150))
}

func TestBMIOnlyWithBoth(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	for name, over := range map[string]map[string]any{
		"weight only":     {"length": nil, "head": nil},
		"length only":     {"weight": nil, "head": nil},
		"weight and head": {"length": nil},
		"length and head": {"weight": nil},
	} {
		if bmi := scoresOf(k.createGrowth(ana, kid, over))["bmi"]; bmi != nil {
			t.Errorf("%s: bmi = %v, want null", name, bmi)
		}
	}
	m := k.createGrowth(ana, kid, map[string]any{"head": nil})
	bmi := scoresOf(m)["bmi"]
	wantScored(t, "weight and length", bmi, growth.SourceWHO, growth.BMI, growth.Female, 89, bmiOf(5000, 600))
	if v := bmi.(map[string]any)["value"]; v != 13.89 {
		t.Errorf("bmi value = %v, want 13.89", v)
	}
}

func TestGrowthRowCapAnswersLimit(t *testing.T) {
	k, db := fileKit(t)
	kid, ana, _ := k.trackerKid()
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	for range constants.MaxGrowthPerKid - 1 {
		if _, err := tx.Exec(`INSERT INTO growth_measurements(id, kid_id, measured_on, weight_g, weight_oz_h, weight_typed,
			created_by, updated_by, created_at, updated_at) VALUES (?, ?, '2026-06-01', 5000, 17637, 'kg', ?, ?, 100, 100)`,
			model.NewID(), kid, k.parentID("ana"), k.parentID("ana")); err != nil {
			tx.Rollback()
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	base := "/v1/kids/" + kid + "/growth"
	last := k.createGrowth(ana, kid, nil)["id"].(string)
	rec := k.do(http.MethodPost, base, ana, growthBody(nil))
	if rec.Code != http.StatusConflict || strings.TrimSpace(rec.Body.String()) != `{"error":"limit"}` {
		t.Errorf("row over the cap: %d %q, want 409 limit", rec.Code, rec.Body.String())
	}
	if rec := k.do(http.MethodDelete, base+"/"+last, ana, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d %q", rec.Code, rec.Body.String())
	}
	k.createGrowth(ana, kid, nil)
}

func TestNoValuesInLogs(t *testing.T) {
	const (
		note  = "plugh-private-note"
		grams = 4321
		date  = "2026-05-17"
	)
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	base := "/v1/kids/" + kid + "/growth"
	k.Log.Reset()

	id := k.createGrowth(ana, kid, map[string]any{"note": note, "measured_on": date, "weight": amountBody("kg", grams)})["id"].(string)
	k.do(http.MethodGet, base+"/"+id, ana, nil)
	k.do(http.MethodGet, base, ana, nil)
	k.do(http.MethodPatch, base+"/"+id, ana, map[string]any{"note": note + "2", "head": amountBody("in", 1531)})
	k.do(http.MethodPatch, base+"/"+id, ana, map[string]any{"measured_on": "2026-11-30"}) // a refusal
	k.do(http.MethodDelete, base+"/"+id, ana, nil)

	wantCleanLog(t, k.Log.String(), 6, "plugh", "4321", "1531", date, "2026-11-30", "measured_on", "weight", "head")
}
