package httpapi

import (
	"context"
	"math"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/units"
)

func symptomBody(over map[string]any) map[string]any {
	return withOverrides(map[string]any{"at": ts(t0.Add(-time.Hour)), "symptom": "cough", "severity": "mild", "note": "first"}, over)
}

func feverBody(over map[string]any) map[string]any {
	return withOverrides(map[string]any{"symptom": "fever", "temperature": amountBody("c", 3850), "method": "oral"}, over)
}

func (k *kit) createSymptom(token, kid string, body map[string]any) map[string]any {
	k.t.Helper()
	rec := k.do(http.MethodPost, "/v1/kids/"+kid+"/symptoms", token, body)
	if rec.Code != http.StatusCreated {
		k.t.Fatalf("create symptom: %d %q", rec.Code, rec.Body.String())
	}
	return bodyMap(k.t, rec)
}

func TestSymptomRoutes(t *testing.T) {
	k := newKit(t)
	kid, ana, bo := k.trackerKid()
	other := k.createKid(ana, map[string]any{"name": "Second"})
	xavi, admin := k.asParent("xavi"), k.asAdmin()
	base := "/v1/kids/" + kid + "/symptoms"

	for name, token := range map[string]string{"guardian": ana, "caregiver": bo} {
		rec := k.do(http.MethodPost, base, token, symptomBody(feverBody(map[string]any{"temperature": amountBody("f", 10150)})))
		if rec.Code != http.StatusCreated {
			t.Fatalf("%s create: %d %q", name, rec.Code, rec.Body.String())
		}
		m := bodyMap(t, rec)
		sameKeys(t, m, "id", "at", "symptom", "severity", "other_text", "temperature", "method", "note")
		wantTemp := map[string]any{"c_hundredths": float64(3861), "f_hundredths": float64(10150), "typed_unit": "f"}
		if m["at"] != ts(t0.Add(-time.Hour)) || m["symptom"] != "fever" || m["severity"] != "mild" || m["other_text"] != nil ||
			m["method"] != "oral" || m["note"] != "first" || !equalJSON(m["temperature"], wantTemp) {
			t.Errorf("%s view = %v", name, m)
		}
		one := base + "/" + m["id"].(string)
		if rec := k.do(http.MethodGet, one, token, nil); rec.Code != http.StatusOK || !equalJSON(bodyMap(t, rec), m) {
			t.Errorf("%s get: %d %q", name, rec.Code, rec.Body.String())
		}
		rec = k.do(http.MethodPatch, one, token, map[string]any{"severity": "severe", "note": nil})
		if p := bodyMap(t, rec); rec.Code != http.StatusOK || p["severity"] != "severe" || p["note"] != nil || p["at"] != m["at"] ||
			!equalJSON(p["temperature"], wantTemp) {
			t.Errorf("%s patch: %d %v", name, rec.Code, p)
		}
		if rec := k.do(http.MethodDelete, one, token, nil); rec.Code != http.StatusNoContent {
			t.Errorf("%s delete: %d %q", name, rec.Code, rec.Body.String())
		}
		wantErr(t, k.do(http.MethodGet, one, token, nil), http.StatusNotFound, "not_found")
	}

	id := k.createSymptom(ana, kid, symptomBody(nil))["id"].(string)
	otherID := k.createSymptom(ana, other, symptomBody(nil))["id"].(string)
	for _, c := range []struct {
		method, path string
		body         any
	}{
		{http.MethodGet, base, nil},
		{http.MethodPost, base, symptomBody(nil)},
		{http.MethodGet, base + "/" + id, nil},
		{http.MethodPatch, base + "/" + id, map[string]any{"severity": "severe"}},
		{http.MethodDelete, base + "/" + id, nil},
	} {
		unknown := strings.Replace(c.path, kid, model.NewID(), 1)
		want := k.do(c.method, unknown, xavi, c.body).Body.String()
		if rec := k.do(c.method, c.path, xavi, c.body); rec.Code != http.StatusNotFound || rec.Body.String() != want {
			t.Errorf("no grant %s %s: %d %q, want %q", c.method, c.path, rec.Code, rec.Body.String(), want)
		}
		wantErr(t, k.do(c.method, c.path, admin, c.body), http.StatusForbidden, "forbidden")
	}
	for _, method := range []string{http.MethodGet, http.MethodPatch, http.MethodDelete} {
		wantErr(t, k.do(method, base+"/"+otherID, ana, map[string]any{"severity": "severe"}), http.StatusNotFound, "not_found")
	}
	if rec := k.do(http.MethodGet, "/v1/kids/"+other+"/symptoms/"+otherID, ana, nil); rec.Code != http.StatusOK || bodyMap(t, rec)["severity"] != "mild" {
		t.Errorf("the other kid's symptom changed: %d %q", rec.Code, rec.Body.String())
	}
}

func TestSymptomValidation(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	base := "/v1/kids/" + kid + "/symptoms"
	coughID := k.createSymptom(ana, kid, symptomBody(nil))["id"].(string)
	feverID := k.createSymptom(ana, kid, symptomBody(feverBody(nil)))["id"].(string)
	long := strings.Repeat("x", constants.MaxNoteLen+1)

	for _, c := range []struct {
		name       string
		over       map[string]any
		field      string
		createOnly bool
	}{
		{"future", map[string]any{"at": ts(t0.Add(constants.MaxFutureSkew + time.Second))}, "at", false},
		{"before the birth date", map[string]any{"at": "2026-03-03T22:59:59Z"}, "at", false},
		{"unknown symptom", map[string]any{"symptom": "sneeze"}, "symptom", false},
		{"empty symptom", map[string]any{"symptom": ""}, "symptom", false},
		{"no symptom", map[string]any{"symptom": nil}, "symptom", true},
		{"unknown severity", map[string]any{"severity": "dire"}, "severity", false},
		{"empty severity", map[string]any{"severity": ""}, "severity", false},
		{"text on a cough", map[string]any{"other_text": "x"}, "other_text", false},
		{"empty text on a cough", map[string]any{"other_text": ""}, "other_text", false},
		{"other without text", map[string]any{"symptom": "other"}, "other_text", false},
		{"other with empty text", map[string]any{"symptom": "other", "other_text": ""}, "other_text", false},
		{"other with blank text", map[string]any{"symptom": "other", "other_text": "   "}, "other_text", false},
		{"other with text over the limit", map[string]any{"symptom": "other", "other_text": strings.Repeat("x", constants.SymptomOtherMaxLen+1)}, "other_text", false},
		{"temperature on a cough", map[string]any{"temperature": amountBody("c", 3850)}, "temperature", false},
		{"temperature and method on a cough", map[string]any{"temperature": amountBody("c", 3850), "method": "oral"}, "temperature", false},
		{"method on a cough", map[string]any{"method": "oral"}, "method", false},
		{"fever without temperature", map[string]any{"symptom": "fever", "method": "oral"}, "temperature", true},
		{"fever with null temperature", map[string]any{"symptom": "fever", "temperature": nil, "method": "oral"}, "temperature", false},
		{"fever without method", map[string]any{"symptom": "fever", "temperature": amountBody("c", 3850)}, "method", true},
		{"fever with unknown method", map[string]any{"symptom": "fever", "temperature": amountBody("c", 3850), "method": "tongue"}, "method", false},
		{"unknown temperature unit", map[string]any{"symptom": "fever", "temperature": amountBody("k", 3850), "method": "oral"}, "temperature", false},
		{"temperature without unit", map[string]any{"symptom": "fever", "temperature": map[string]any{"amount": 3850}, "method": "oral"}, "temperature", false},
		{"note over the limit", map[string]any{"note": long}, "note", false},
		// The first invalid field wins, in the order at, symptom, severity, other_text, temperature, method, note.
		{"at before symptom", map[string]any{"at": ts(t0.Add(time.Hour)), "symptom": "sneeze"}, "at", false},
		{"symptom before severity", map[string]any{"symptom": "sneeze", "severity": "dire"}, "symptom", false},
		{"severity before other_text", map[string]any{"severity": "dire", "other_text": "x"}, "severity", false},
		{"other_text before temperature", map[string]any{"other_text": "x", "temperature": amountBody("c", 3850)}, "other_text", false},
		{"temperature before method", map[string]any{"temperature": amountBody("c", 3850), "method": "oral"}, "temperature", false},
		{"method before note", map[string]any{"method": "oral", "note": long}, "method", false},
	} {
		wantInvalid2(t, c.name+" (create)", k.do(http.MethodPost, base, ana, symptomBody(c.over)), c.field)
		if c.createOnly {
			continue
		}
		target := coughID
		if c.over["symptom"] == "fever" {
			target = feverID
		}
		wantInvalid2(t, c.name+" (patch)", k.do(http.MethodPatch, base+"/"+target, ana, c.over), c.field)
	}

	for _, name := range model.Symptoms {
		body := symptomBody(map[string]any{"symptom": name, "severity": nil})
		switch name {
		case "fever":
			body = feverBody(body)
		case "other":
			body["other_text"] = strings.Repeat("é", constants.SymptomOtherMaxLen)
		}
		if rec := k.do(http.MethodPost, base, ana, body); rec.Code != http.StatusCreated {
			t.Errorf("symptom %s: %d %q", name, rec.Code, rec.Body.String())
		}
	}
	for _, severity := range model.Severities {
		if rec := k.do(http.MethodPost, base, ana, symptomBody(map[string]any{"severity": severity})); rec.Code != http.StatusCreated {
			t.Errorf("severity %s: %d %q", severity, rec.Code, rec.Body.String())
		}
	}
	for _, method := range model.TempMethods {
		if rec := k.do(http.MethodPost, base, ana, symptomBody(feverBody(map[string]any{"method": method}))); rec.Code != http.StatusCreated {
			t.Errorf("method %s: %d %q", method, rec.Code, rec.Body.String())
		}
	}
	rec := k.do(http.MethodPost, base, ana, symptomBody(map[string]any{"colour": "red"}))
	if rec.Code != http.StatusUnprocessableEntity || strings.TrimSpace(rec.Body.String()) != `{"error":"unknown_field","field":"colour"}` {
		t.Errorf("unknown field: %d %q", rec.Code, rec.Body.String())
	}
	rec = k.do(http.MethodPost, base, ana, symptomBody(feverBody(map[string]any{"temperature": map[string]any{"unit": "c", "amount": 3850, "scale": 1}})))
	if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), "unknown_field") {
		t.Errorf("unknown field inside the temperature: %d %q", rec.Code, rec.Body.String())
	}
}

func TestSymptomTemperatureEdges(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	base := "/v1/kids/" + kid + "/symptoms"
	minC, maxC := int64(constants.MinTempCHundredths), int64(constants.MaxTempCHundredths)
	// The °F amounts that convert to exactly the bounds, and one hundredth either side of them.
	minF, maxF := minC*9/5+3200, maxC*9/5+3200

	for name, c := range map[string]struct {
		temp   any
		status int
	}{
		"zero c":                {amountBody("c", 0), 422},
		"negative c":            {amountBody("c", -100), 422},
		"zero f":                {amountBody("f", 0), 422},
		"negative f":            {amountBody("f", -4000), 422},
		"below the minimum c":   {amountBody("c", minC-1), 422},
		"the minimum c":         {amountBody("c", minC), 201},
		"the maximum c":         {amountBody("c", maxC), 201},
		"above the maximum c":   {amountBody("c", maxC+1), 422},
		"below the minimum f":   {amountBody("f", minF-1), 422},
		"the minimum f":         {amountBody("f", minF), 201},
		"above the minimum f":   {amountBody("f", minF+1), 201},
		"below the maximum f":   {amountBody("f", maxF-1), 201},
		"the maximum f":         {amountBody("f", maxF), 201},
		"above the maximum f":   {amountBody("f", maxF+1), 422},
		"the largest int64 c":   {amountBody("c", math.MaxInt64), 422},
		"the largest int64 f":   {amountBody("f", math.MaxInt64), 422},
		"the smallest int64 c":  {amountBody("c", math.MinInt64), 422},
		"the smallest int64 f":  {amountBody("f", math.MinInt64), 422},
		"three decimals":        {map[string]any{"unit": "c", "amount": 3850.5}, 400},
		"a decimal comma":       {map[string]any{"unit": "c", "amount": "38,5"}, 400},
		"an amount as a string": {map[string]any{"unit": "c", "amount": "3850"}, 400},
		"a bare number":         {3850, 400},
	} {
		rec := k.do(http.MethodPost, base, ana, symptomBody(feverBody(map[string]any{"temperature": c.temp})))
		if rec.Code != c.status {
			t.Errorf("%s: %d %q, want %d", name, rec.Code, rec.Body.String(), c.status)
			continue
		}
		switch c.status {
		case 422:
			wantInvalid2(t, name, rec, "temperature")
		case 400:
			wantErr(t, rec, http.StatusBadRequest, "bad_request")
		}
	}
}

func TestSymptomPatchMerges(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	base := "/v1/kids/" + kid + "/symptoms"
	patch := func(id string, body map[string]any) (int, map[string]any) {
		t.Helper()
		rec := k.do(http.MethodPatch, base+"/"+id, ana, body)
		var m map[string]any
		if rec.Code == http.StatusOK {
			m = bodyMap(t, rec)
		}
		return rec.Code, m
	}
	fever := func() map[string]any {
		return k.createSymptom(ana, kid, symptomBody(feverBody(map[string]any{"temperature": amountBody("f", 10150), "severity": "moderate"})))
	}
	wantF := map[string]any{"c_hundredths": float64(3861), "f_hundredths": float64(10150), "typed_unit": "f"}

	// An empty patch and a note-only patch keep every stored column.
	m := fever()
	id := m["id"].(string)
	for name, body := range map[string]map[string]any{
		"empty":                  {},
		"note only":              {"note": "only the note changed"},
		"same typed temperature": {"temperature": amountBody("f", 10150), "note": "x"},
		"same value and method":  {"method": "oral", "symptom": "fever", "severity": "moderate"},
	} {
		code, got := patch(id, body)
		if code != http.StatusOK || !equalJSON(got["temperature"], wantF) || got["method"] != "oral" || got["severity"] != "moderate" {
			t.Errorf("%s: %d %v", name, code, got)
		}
	}
	// The same number in the other unit is a new typed value, converted from what was typed.
	if code, got := patch(id, map[string]any{"temperature": amountBody("c", 3861)}); code != http.StatusOK ||
		!equalJSON(got["temperature"], map[string]any{"c_hundredths": float64(3861), "f_hundredths": float64(10150), "typed_unit": "c"}) {
		t.Errorf("typed in c: %d %v", code, got)
	}
	if code, got := patch(id, map[string]any{"temperature": amountBody("c", 4000), "method": "ear"}); code != http.StatusOK ||
		!equalJSON(got["temperature"], map[string]any{"c_hundredths": float64(4000), "f_hundredths": float64(10400), "typed_unit": "c"}) || got["method"] != "ear" {
		t.Errorf("changed temperature and method: %d %v", code, got)
	}
	// A refused edit changes nothing.
	if code, _ := patch(id, map[string]any{"temperature": amountBody("c", constants.MaxTempCHundredths+1), "note": "lost"}); code != http.StatusUnprocessableEntity {
		t.Errorf("out of range: %d", code)
	}
	if got := bodyMap(t, k.do(http.MethodGet, base+"/"+id, ana, nil)); got["note"] == "lost" {
		t.Errorf("a refused patch changed the note: %v", got)
	}

	// null clears severity and note; an absent key keeps them.
	if code, got := patch(id, map[string]any{"severity": nil, "note": nil}); code != http.StatusOK || got["severity"] != nil || got["note"] != nil {
		t.Errorf("null clears: %d %v", code, got)
	}
	if code, got := patch(id, map[string]any{"at": ts(t0.Add(-2 * time.Hour))}); code != http.StatusOK || got["at"] != ts(t0.Add(-2*time.Hour)) || got["severity"] != nil {
		t.Errorf("at only: %d %v", code, got)
	}

	// A stored temperature that is out of range today stays valid while it is re-sent unchanged.
	oral := "oral"
	stored, err := k.Store.CreateSymptom(context.Background(), kid, k.parentID("ana"), model.Symptom{
		At: t0.Add(-time.Hour), Symptom: "fever", Method: &oral,
		Temperature: &units.Temperature{CHundredths: constants.MaxTempCHundredths + 100, FHundredths: 11480, Typed: units.UnitC},
	}, t0)
	if err != nil {
		t.Fatal(err)
	}
	if code, got := patch(stored.ID, map[string]any{"temperature": amountBody("c", constants.MaxTempCHundredths+100), "note": "kept"}); code != http.StatusOK ||
		got["note"] != "kept" || !equalJSON(got["temperature"], map[string]any{"c_hundredths": float64(constants.MaxTempCHundredths + 100), "f_hundredths": float64(11480), "typed_unit": "c"}) {
		t.Errorf("unchanged out-of-range temperature: %d %v", code, got)
	}
	if code, _ := patch(stored.ID, map[string]any{"temperature": amountBody("c", constants.MaxTempCHundredths+101)}); code != http.StatusUnprocessableEntity {
		t.Errorf("changed out-of-range temperature: %d", code)
	}
}

func TestSymptomChangeAcrossFever(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	base := "/v1/kids/" + kid + "/symptoms"
	newFever := func() string { return k.createSymptom(ana, kid, symptomBody(feverBody(nil)))["id"].(string) }
	newCough := func() string { return k.createSymptom(ana, kid, symptomBody(nil))["id"].(string) }

	for name, over := range map[string]map[string]any{
		"neither cleared":        {"symptom": "cough"},
		"only the temperature":   {"symptom": "cough", "temperature": nil},
		"only the method":        {"symptom": "cough", "method": nil},
		"temperature re-sent":    {"symptom": "cough", "temperature": amountBody("c", 3850), "method": nil},
		"to a cleared other":     {"symptom": "other", "other_text": "x"},
		"to a cleared other too": {"symptom": "other", "other_text": "x", "temperature": nil},
	} {
		wantInvalid2(t, "fever to "+name, k.do(http.MethodPatch, base+"/"+newFever(), ana, over), "temperature")
	}
	rec := k.do(http.MethodPatch, base+"/"+newFever(), ana, map[string]any{"symptom": "cough", "temperature": nil, "method": nil})
	if m := bodyMap(t, rec); rec.Code != http.StatusOK || m["symptom"] != "cough" || m["temperature"] != nil || m["method"] != nil {
		t.Errorf("fever to cough: %d %v", rec.Code, m)
	}
	rec = k.do(http.MethodPatch, base+"/"+newFever(), ana, map[string]any{"symptom": "other", "other_text": "rash on the legs", "temperature": nil, "method": nil})
	if m := bodyMap(t, rec); rec.Code != http.StatusOK || m["symptom"] != "other" || m["other_text"] != "rash on the legs" || m["temperature"] != nil {
		t.Errorf("fever to other: %d %v", rec.Code, m)
	}
	// Re-sending fever is not a change.
	rec = k.do(http.MethodPatch, base+"/"+newFever(), ana, map[string]any{"symptom": "fever", "severity": "mild"})
	if m := bodyMap(t, rec); rec.Code != http.StatusOK || m["temperature"] == nil || m["method"] != "oral" {
		t.Errorf("fever to fever: %d %v", rec.Code, m)
	}

	wantInvalid2(t, "cough to fever, nothing", k.do(http.MethodPatch, base+"/"+newCough(), ana, map[string]any{"symptom": "fever"}), "temperature")
	wantInvalid2(t, "cough to fever, null temperature", k.do(http.MethodPatch, base+"/"+newCough(), ana,
		map[string]any{"symptom": "fever", "temperature": nil, "method": "oral"}), "temperature")
	wantInvalid2(t, "cough to fever, no method", k.do(http.MethodPatch, base+"/"+newCough(), ana,
		map[string]any{"symptom": "fever", "temperature": amountBody("c", 3850)}), "method")
	wantInvalid2(t, "cough to fever, null method", k.do(http.MethodPatch, base+"/"+newCough(), ana,
		map[string]any{"symptom": "fever", "temperature": amountBody("c", 3850), "method": nil}), "method")
	rec = k.do(http.MethodPatch, base+"/"+newCough(), ana, map[string]any{"symptom": "fever", "temperature": amountBody("f", 10200), "method": "forehead"})
	if m := bodyMap(t, rec); rec.Code != http.StatusOK || m["symptom"] != "fever" || m["method"] != "forehead" ||
		!equalJSON(m["temperature"], map[string]any{"c_hundredths": float64(3889), "f_hundredths": float64(10200), "typed_unit": "f"}) {
		t.Errorf("cough to fever: %d %v", rec.Code, m)
	}
}

func TestSymptomOtherText(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	base := "/v1/kids/" + kid + "/symptoms"
	other := k.createSymptom(ana, kid, symptomBody(map[string]any{"symptom": "other", "other_text": "  pulling at the ear  "}))
	if other["other_text"] != "pulling at the ear" {
		t.Errorf("other_text = %q, want it trimmed", other["other_text"])
	}
	id := other["id"].(string)

	wantInvalid2(t, "other to cough keeps its text", k.do(http.MethodPatch, base+"/"+id, ana, map[string]any{"symptom": "cough"}), "other_text")
	wantInvalid2(t, "other cannot lose its text", k.do(http.MethodPatch, base+"/"+id, ana, map[string]any{"other_text": nil}), "other_text")
	if rec := k.do(http.MethodPatch, base+"/"+id, ana, map[string]any{"other_text": "x"}); rec.Code != http.StatusOK || bodyMap(t, rec)["other_text"] != "x" {
		t.Errorf("edit the text: %d %q", rec.Code, rec.Body.String())
	}
	rec := k.do(http.MethodPatch, base+"/"+id, ana, map[string]any{"symptom": "cough", "other_text": nil})
	if m := bodyMap(t, rec); rec.Code != http.StatusOK || m["symptom"] != "cough" || m["other_text"] != nil {
		t.Errorf("other to cough: %d %v", rec.Code, m)
	}
	wantInvalid2(t, "cough to other needs text", k.do(http.MethodPatch, base+"/"+id, ana, map[string]any{"symptom": "other"}), "other_text")
	rec = k.do(http.MethodPatch, base+"/"+id, ana, map[string]any{"symptom": "other", "other_text": strings.Repeat("y", constants.SymptomOtherMaxLen)})
	if rec.Code != http.StatusOK {
		t.Errorf("cough to other: %d %q", rec.Code, rec.Body.String())
	}
}

func TestSymptomList(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	base := "/v1/kids/" + kid + "/symptoms"
	var ids []string
	for _, m := range []time.Duration{3, 1, 2} {
		ids = append(ids, k.createSymptom(ana, kid, symptomBody(map[string]any{"at": ts(t0.Add(-m * time.Minute))}))["id"].(string))
	}
	want := []string{ids[1], ids[2], ids[0]}

	rec := k.do(http.MethodGet, base+"?limit=2", ana, nil)
	m := bodyMap(t, rec)
	sameKeys(t, m, "symptoms", "next_cursor")
	first := m["symptoms"].([]any)
	cursor, ok := m["next_cursor"].(string)
	if rec.Code != http.StatusOK || len(first) != 2 || !ok {
		t.Fatalf("page 1: %d %q", rec.Code, rec.Body.String())
	}
	rec = k.do(http.MethodGet, base+"?limit=2&cursor="+url.QueryEscape(cursor), ana, nil)
	m = bodyMap(t, rec)
	rest := m["symptoms"].([]any)
	if rec.Code != http.StatusOK || len(rest) != 1 || m["next_cursor"] != nil {
		t.Fatalf("page 2: %d %q", rec.Code, rec.Body.String())
	}
	var got []string
	for _, e := range append(first, rest...) {
		got = append(got, e.(map[string]any)["id"].(string))
	}
	if !slices.Equal(got, want) {
		t.Errorf("paged = %v, want %v", got, want)
	}
	window := "?from=" + url.QueryEscape(ts(t0.Add(-2*time.Minute))) + "&to=" + url.QueryEscape(ts(t0.Add(-time.Minute)))
	rec = k.do(http.MethodGet, base+window, ana, nil)
	if in := bodyMap(t, rec)["symptoms"].([]any); rec.Code != http.StatusOK || len(in) != 1 || in[0].(map[string]any)["id"] != ids[2] {
		t.Errorf("[from, to) = %d %q, want only %s", rec.Code, rec.Body.String(), ids[2])
	}
	wantInvalid(t, k.do(http.MethodGet, base+"?limit=0", ana, nil), "limit")
}

func TestSymptomNoValuesInLogs(t *testing.T) {
	const (
		note  = "plugh-private-note"
		other = "xyzzy-private-text"
	)
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	base := "/v1/kids/" + kid + "/symptoms"
	k.Log.Reset()

	fever := k.createSymptom(ana, kid, symptomBody(feverBody(map[string]any{"note": note, "temperature": amountBody("c", 3917)})))["id"].(string)
	odd := k.createSymptom(ana, kid, symptomBody(map[string]any{"symptom": "other", "other_text": other, "note": note}))["id"].(string)
	k.do(http.MethodGet, base+"/"+fever, ana, nil)
	k.do(http.MethodGet, base, ana, nil)
	k.do(http.MethodPatch, base+"/"+fever, ana, map[string]any{"note": note + "2", "temperature": amountBody("f", 10321)})
	k.do(http.MethodPatch, base+"/"+odd, ana, map[string]any{"other_text": other + "2", "temperature": amountBody("c", 3917)}) // a refusal
	k.do(http.MethodDelete, base+"/"+fever, ana, nil)

	wantCleanLog(t, k.Log.String(), 7, "plugh", "xyzzy", "3917", "10321", "temperature", "other_text", "fever", "severe", "rectal")
}
