package httpapi

import (
	"context"
	"encoding/json"
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

func pumpingBody(over map[string]any) map[string]any {
	return withOverrides(map[string]any{
		"started_at":       ts(t0.Add(-time.Hour)),
		"duration_seconds": 900,
		"volume":           volumeBody("ml", 90),
		"side":             "both",
		"note":             "first",
	}, over)
}

func (k *kit) createPumping(token, kid string, over map[string]any) string {
	k.t.Helper()
	rec := k.do(http.MethodPost, "/v1/kids/"+kid+"/pumping", token, pumpingBody(over))
	if rec.Code != http.StatusCreated {
		k.t.Fatalf("create pumping: %d %q", rec.Code, rec.Body.String())
	}
	return bodyMap(k.t, rec)["id"].(string)
}

func maxSessionSeconds() int { return int(constants.MaxSessionDuration / time.Second) }

func TestPumpingRoutes(t *testing.T) {
	k := newKit(t)
	kid, ana, bo := k.trackerKid()
	other := k.createKid(ana, map[string]any{"name": "Second"})
	xavi, admin := k.asParent("xavi"), k.asAdmin()
	base := "/v1/kids/" + kid + "/pumping"

	for name, token := range map[string]string{"guardian": ana, "caregiver": bo} {
		rec := k.do(http.MethodPost, base, token, pumpingBody(nil))
		if rec.Code != http.StatusCreated {
			t.Fatalf("%s create: %d %q", name, rec.Code, rec.Body.String())
		}
		m := bodyMap(t, rec)
		sameKeys(t, m, "id", "started_at", "duration_seconds", "volume", "side", "sides", "note")
		if m["started_at"] != ts(t0.Add(-time.Hour)) || m["duration_seconds"] != 900.0 || m["side"] != "both" || m["note"] != "first" ||
			!equalJSON(m["volume"], map[string]any{"ml": 90.0, "floz_hundredths": 304.0, "typed_unit": "ml"}) {
			t.Errorf("%s view = %v", name, m)
		}
		one := base + "/" + m["id"].(string)
		if rec := k.do(http.MethodGet, one, token, nil); rec.Code != http.StatusOK || !equalJSON(bodyMap(t, rec), m) {
			t.Errorf("%s get: %d %q", name, rec.Code, rec.Body.String())
		}
		rec = k.do(http.MethodPatch, one, token, map[string]any{"side": "left", "duration_seconds": 60, "note": nil})
		if p := bodyMap(t, rec); rec.Code != http.StatusOK || p["side"] != "left" || p["duration_seconds"] != 60.0 || p["note"] != nil ||
			p["started_at"] != m["started_at"] || !equalJSON(p["volume"], m["volume"]) {
			t.Errorf("%s patch: %d %v", name, rec.Code, p)
		}
		if rec := k.do(http.MethodDelete, one, token, nil); rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
			t.Errorf("%s delete: %d %q", name, rec.Code, rec.Body.String())
		}
		wantErr(t, k.do(http.MethodGet, one, token, nil), http.StatusNotFound, "not_found")
	}

	id, otherID := k.createPumping(ana, kid, nil), k.createPumping(ana, other, nil)
	for _, c := range []struct {
		method, path string
		body         any
	}{
		{http.MethodGet, base, nil},
		{http.MethodPost, base, pumpingBody(nil)},
		{http.MethodGet, base + "/" + id, nil},
		{http.MethodPatch, base + "/" + id, map[string]any{"side": "right"}},
		{http.MethodDelete, base + "/" + id, nil},
	} {
		unknown := strings.Replace(c.path, kid, model.NewID(), 1)
		want := k.do(c.method, unknown, xavi, c.body).Body.String()
		if rec := k.do(c.method, c.path, xavi, c.body); rec.Code != http.StatusNotFound || rec.Body.String() != want {
			t.Errorf("no grant %s %s: %d %q, want %q", c.method, c.path, rec.Code, rec.Body.String(), want)
		}
		wantErr(t, k.do(c.method, c.path, admin, c.body), http.StatusForbidden, "forbidden")
	}
	for _, entry := range []string{otherID, model.NewID(), "not an id"} {
		for _, method := range []string{http.MethodGet, http.MethodPatch, http.MethodDelete} {
			wantErr(t, k.do(method, base+"/"+url.PathEscape(entry), ana, map[string]any{"side": "right"}), http.StatusNotFound, "not_found")
		}
	}
	// Recorded on the kid it is for: the other kid's session is neither listed here nor changed.
	if rec := k.do(http.MethodGet, "/v1/kids/"+other+"/pumping/"+otherID, ana, nil); rec.Code != http.StatusOK || bodyMap(t, rec)["side"] != "both" {
		t.Errorf("the other kid's session changed: %d %q", rec.Code, rec.Body.String())
	}
	if rows := bodyMap(t, k.do(http.MethodGet, base, ana, nil))["pumping"].([]any); len(rows) != 1 || rows[0].(map[string]any)["id"] != id {
		t.Errorf("list for the kid = %v, want only %s", rows, id)
	}
}

func TestPumpingCreateDefaultsStartToNow(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	rec := k.do(http.MethodPost, "/v1/kids/"+kid+"/pumping", ana, pumpingBody(map[string]any{"started_at": nil, "note": nil}))
	if rec.Code != http.StatusCreated {
		t.Fatalf("%d %q", rec.Code, rec.Body.String())
	}
	if m := bodyMap(t, rec); m["started_at"] != ts(t0) || m["note"] != nil {
		t.Errorf("view = %v, want started_at now and a null note", m)
	}
}

func TestPumpingDurationBounds(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	base := "/v1/kids/" + kid + "/pumping"
	id := k.createPumping(ana, kid, nil)

	for _, seconds := range []int{1, maxSessionSeconds()} {
		rec := k.do(http.MethodPost, base, ana, pumpingBody(map[string]any{"duration_seconds": seconds}))
		if m := bodyMap(t, rec); rec.Code != http.StatusCreated || m["duration_seconds"] != float64(seconds) {
			t.Errorf("create %d s: %d %q", seconds, rec.Code, rec.Body.String())
		}
		rec = k.do(http.MethodPatch, base+"/"+id, ana, map[string]any{"duration_seconds": seconds})
		if m := bodyMap(t, rec); rec.Code != http.StatusOK || m["duration_seconds"] != float64(seconds) {
			t.Errorf("patch %d s: %d %q", seconds, rec.Code, rec.Body.String())
		}
	}
	for name, seconds := range map[string]int{"zero": 0, "negative": -1, "one over the limit": maxSessionSeconds() + 1, "huge": 1 << 40} {
		wantInvalid2(t, name+" (create)", k.do(http.MethodPost, base, ana, pumpingBody(map[string]any{"duration_seconds": seconds})), "duration_seconds")
		wantInvalid2(t, name+" (patch)", k.do(http.MethodPatch, base+"/"+id, ana, map[string]any{"duration_seconds": seconds}), "duration_seconds")
	}
	wantInvalid2(t, "no duration", k.do(http.MethodPost, base, ana, pumpingBody(map[string]any{"duration_seconds": nil})), "duration_seconds")
	for _, body := range []string{`{"duration_seconds":1.5}`, `{"duration_seconds":"60"}`} {
		wantErr(t, k.do(http.MethodPatch, base+"/"+id, ana, body), http.StatusBadRequest, "bad_request")
	}
	if m := bodyMap(t, k.do(http.MethodGet, base+"/"+id, ana, nil)); m["duration_seconds"] != float64(maxSessionSeconds()) {
		t.Errorf("a refused patch changed the duration to %v", m["duration_seconds"])
	}
}

func TestPumpingValidation(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	base := "/v1/kids/" + kid + "/pumping"
	id := k.createPumping(ana, kid, nil)
	stored := k.do(http.MethodGet, base+"/"+id, ana, nil).Body.String()

	cases := []struct {
		name       string
		over       map[string]any
		field      string
		createOnly bool // a missing key on PATCH keeps the stored value
	}{
		{"more than 5 minutes ahead", map[string]any{"started_at": ts(t0.Add(constants.MaxFutureSkew + time.Second))}, "started_at", false},
		{"before the birth date", map[string]any{"started_at": "2026-03-03T22:59:59Z"}, "started_at", false},
		{"volume 0", map[string]any{"volume": volumeBody("ml", 0)}, "volume", false},
		{"volume -1", map[string]any{"volume": volumeBody("ml", -1)}, "volume", false},
		{"over the cap", map[string]any{"volume": volumeBody("ml", constants.MaxPumpingVolumeML+1)}, "volume", false},
		{"over the cap in ounces", map[string]any{"volume": volumeBody("floz", 7000)}, "volume", false},
		{"ounces that round to 0 ml", map[string]any{"volume": volumeBody("floz", 1)}, "volume", false},
		{"unknown unit", map[string]any{"volume": volumeBody("cup", 1)}, "volume", false},
		{"no amount", map[string]any{"volume": map[string]any{"unit": "ml"}}, "volume", false},
		{"no volume", map[string]any{"volume": nil}, "volume", true},
		{"unknown side", map[string]any{"side": "middle"}, "side", false},
		{"empty side", map[string]any{"side": ""}, "side", false},
		{"wrong case side", map[string]any{"side": "Left"}, "side", false},
		{"no side", map[string]any{"side": nil}, "side", true},
		{"note over the limit", map[string]any{"note": strings.Repeat("ñ", constants.MaxNoteLen+1)}, "note", false},
	}
	for _, c := range cases {
		wantInvalid2(t, c.name+" (create)", k.do(http.MethodPost, base, ana, pumpingBody(c.over)), c.field)
		if !c.createOnly {
			wantInvalid2(t, c.name+" (patch)", k.do(http.MethodPatch, base+"/"+id, ana, c.over), c.field)
		}
	}
	if got := k.do(http.MethodGet, base+"/"+id, ana, nil).Body.String(); got != stored {
		t.Errorf("a refused patch changed the entry:\n got %s\nwant %s", got, stored)
	}
	if n := len(bodyMap(t, k.do(http.MethodGet, base, ana, nil))["pumping"].([]any)); n != 1 {
		t.Errorf("%d sessions after the refused creates, want 1", n)
	}

	for _, c := range []struct {
		name string
		over map[string]any
	}{
		{"exactly 5 minutes ahead", map[string]any{"started_at": ts(t0.Add(constants.MaxFutureSkew))}},
		{"the start of the birth date in the kid's zone", map[string]any{"started_at": "2026-03-03T23:00:00Z"}},
		{"the volume cap", map[string]any{"volume": volumeBody("ml", constants.MaxPumpingVolumeML)}},
		{"one millilitre", map[string]any{"volume": volumeBody("ml", 1)}},
		{"side left", map[string]any{"side": "left"}},
		{"side right", map[string]any{"side": "right"}},
		{"a note at the limit", map[string]any{"note": strings.Repeat("ñ", constants.MaxNoteLen)}},
	} {
		if rec := k.do(http.MethodPost, base, ana, pumpingBody(c.over)); rec.Code != http.StatusCreated {
			t.Errorf("%s: %d %q", c.name, rec.Code, rec.Body.String())
		}
	}

	for _, method := range []string{http.MethodPost, http.MethodPatch} {
		path := base
		if method == http.MethodPatch {
			path += "/" + id
		}
		rec := k.do(method, path, ana, pumpingBody(map[string]any{"colour": "red"}))
		if rec.Code != http.StatusUnprocessableEntity || strings.TrimSpace(rec.Body.String()) != `{"error":"unknown_field","field":"colour"}` {
			t.Errorf("%s unknown field: %d %q", method, rec.Code, rec.Body.String())
		}
		wantErr(t, k.do(method, path, ana, pumpingBody(map[string]any{"started_at": "yesterday"})), http.StatusBadRequest, "bad_request")
	}
}

func TestPumpingEditingNoteKeepsTypedVolume(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	id := k.createPumping(ana, kid, map[string]any{"volume": volumeBody("floz", 400)})
	one := "/v1/kids/" + kid + "/pumping/" + id
	oz := map[string]any{"ml": 118.0, "floz_hundredths": 400.0, "typed_unit": "floz"}

	for _, patch := range []map[string]any{{"note": "only the note"}, {"volume": volumeBody("floz", 400)}} {
		rec := k.do(http.MethodPatch, one, ana, patch)
		if got := bodyMap(t, rec)["volume"]; rec.Code != http.StatusOK || !equalJSON(got, oz) {
			t.Errorf("after %v: %d volume = %v", patch, rec.Code, got)
		}
	}
	rec := k.do(http.MethodPatch, one, ana, map[string]any{"volume": volumeBody("ml", 120)})
	if got := bodyMap(t, rec)["volume"]; !equalJSON(got, map[string]any{"ml": 120.0, "floz_hundredths": 406.0, "typed_unit": "ml"}) {
		t.Errorf("after entering 120 ml = %v", got)
	}
}

func TestPumpingListRange(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	other := k.createKid(ana, map[string]any{"name": "Second"})
	k.createPumping(ana, other, nil)
	base := "/v1/kids/" + kid + "/pumping"

	at := func(m time.Duration) map[string]any {
		return map[string]any{"started_at": ts(t0.Add(-m * time.Minute))}
	}
	a0 := k.createPumping(ana, kid, at(1))
	a1 := k.createPumping(ana, kid, at(2))
	a2 := k.createPumping(ana, kid, at(2))
	a3 := k.createPumping(ana, kid, at(3))
	tied := []string{a1, a2}
	slices.Sort(tied)
	slices.Reverse(tied)
	all := append(append([]string{a0}, tied...), a3)

	page := func(query string) ([]string, any) {
		t.Helper()
		rec := k.do(http.MethodGet, base+query, ana, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s: %d %q", query, rec.Code, rec.Body.String())
		}
		m := bodyMap(t, rec)
		sameKeys(t, m, "pumping", "next_cursor")
		var ids []string
		for _, e := range m["pumping"].([]any) {
			ids = append(ids, e.(map[string]any)["id"].(string))
		}
		return ids, m["next_cursor"]
	}
	if ids, next := page(""); !slices.Equal(ids, all) || next != nil {
		t.Errorf("default = %v next %v, want %v and null", ids, next, all)
	}
	var paged []string
	query := "?limit=3"
	for i := range 2 {
		ids, next := page(query)
		paged = append(paged, ids...)
		if i == 0 {
			c, ok := next.(string)
			if !ok || len(ids) != 3 {
				t.Fatalf("page 1 = %v next %v, want 3 rows and a cursor", ids, next)
			}
			query = "?limit=3&cursor=" + url.QueryEscape(c)
		} else if next != nil || len(ids) != 1 {
			t.Fatalf("last page = %v next %v, want 1 row and null", ids, next)
		}
	}
	if !slices.Equal(paged, all) {
		t.Errorf("paged = %v, want %v", paged, all)
	}

	rangeQ := func(from, to time.Time) string {
		return "?from=" + url.QueryEscape(from.Format(time.RFC3339Nano)) + "&to=" + url.QueryEscape(to.Format(time.RFC3339Nano))
	}
	if ids, _ := page(rangeQ(t0.Add(-3*time.Minute), t0.Add(-time.Minute))); !slices.Equal(ids, append(slices.Clone(tied), a3)) {
		t.Errorf("[-3m, -1m) = %v, want the 2m pair then the 3m entry", ids)
	}
	if ids, _ := page(rangeQ(t0.Add(-3*time.Minute+900*time.Millisecond), t0.Add(-2*time.Minute+900*time.Millisecond))); !slices.Equal(ids, []string{a3}) {
		t.Errorf("fractional range = %v, want only %s", ids, a3)
	}
	for query, field := range map[string]string{
		"?from=yesterday": "from",
		"?to=2026":        "to",
		"?from=2026-11-20T00:00:00Z&to=2026-11-19T00:00:00Z": "from",
		"?limit=0":   "limit",
		"?limit=101": "limit",
		"?cursor=!!": "cursor",
	} {
		wantInvalid(t, k.do(http.MethodGet, base+query, ana, nil), field)
	}
}

// Meals and pumping carry free text and volumes: none may reach a log line.
func TestMealAndPumpingValuesStayOutOfLogs(t *testing.T) {
	const (
		what     = "zzwhat-mango"
		reaction = "zzreaction-rash"
		note     = "zznote-private"
		mlAmount = 1357
		ozAmount = 4619 // hundredths of an ounce
		seconds  = 777
	)
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	k.Log.Reset()

	m := k.do(http.MethodPost, "/v1/kids/"+kid+"/meals", ana, mealBody(map[string]any{"what": what, "reaction": reaction, "note": note}))
	mealPath := "/v1/kids/" + kid + "/meals/" + bodyMap(t, m)["id"].(string)
	k.do(http.MethodPatch, mealPath, ana, map[string]any{"what": what + "2", "reaction": reaction + "2", "note": note + "2"})
	k.do(http.MethodPatch, mealPath, ana, map[string]any{"what": " "}) // a refusal
	k.do(http.MethodDelete, mealPath, ana, nil)

	p := k.do(http.MethodPost, "/v1/kids/"+kid+"/pumping", ana, pumpingBody(map[string]any{"note": note, "duration_seconds": seconds, "volume": volumeBody("ml", mlAmount)}))
	pumpPath := "/v1/kids/" + kid + "/pumping/" + bodyMap(t, p)["id"].(string)
	k.do(http.MethodPatch, pumpPath, ana, map[string]any{"note": note + "3", "volume": volumeBody("floz", ozAmount), "side": "left"})
	k.do(http.MethodPatch, pumpPath, ana, map[string]any{"volume": volumeBody("ml", 0)}) // a refusal
	k.do(http.MethodDelete, pumpPath, ana, nil)

	wantCleanLog(t, k.Log.String(), 8, "zz", "1357", "4619", "777", "mango", "rash")
}

func advancedPumpingBody(over map[string]any) map[string]any {
	return pumpingBody(withOverrides(map[string]any{
		"volume": nil,
		"side":   nil,
		"sides":  sourcesBody("ml", map[string]int{"left": 60, "right": 50}),
	}, over))
}

func sideView(side string, ml, floz float64, typed string) map[string]any {
	return map[string]any{"side": side, "volume": map[string]any{"ml": ml, "floz_hundredths": floz, "typed_unit": typed}}
}

func (k *kit) pumpingPath(kid string, id ...string) string {
	return "/v1/kids/" + kid + "/pumping" + strings.Join(append([]string{""}, id...), "/")
}

func TestAdvancedPumpingCreate(t *testing.T) {
	k := newKit(t)
	kid, ana, bo := k.trackerKid()

	for name, token := range map[string]string{"guardian": ana, "caregiver": bo} {
		rec := k.do(http.MethodPost, k.pumpingPath(kid), token, advancedPumpingBody(nil))
		if rec.Code != http.StatusCreated {
			t.Fatalf("%s create: %d %q", name, rec.Code, rec.Body.String())
		}
		m := bodyMap(t, rec)
		sameKeys(t, m, "id", "started_at", "duration_seconds", "volume", "side", "sides", "note")
		if m["side"] != "both" || m["duration_seconds"] != 900.0 ||
			!equalJSON(m["volume"], map[string]any{"ml": 110.0, "floz_hundredths": 372.0, "typed_unit": "ml"}) ||
			!equalJSON(m["sides"], []any{sideView("left", 60, 203, "ml"), sideView("right", 50, 169, "ml")}) {
			t.Errorf("%s create view = %v", name, m)
		}
		if rec := k.do(http.MethodGet, k.pumpingPath(kid, m["id"].(string)), token, nil); rec.Code != http.StatusOK || !equalJSON(bodyMap(t, rec), m) {
			t.Errorf("%s get: %d %q", name, rec.Code, rec.Body.String())
		}
	}

	// One side: the derived side is that side, whichever key order the body used, with or without a matching side.
	for name, c := range map[string]struct {
		over map[string]any
		side string
		ml   float64
	}{
		"left only":                {map[string]any{"sides": sourcesBody("ml", map[string]int{"left": 70})}, "left", 70},
		"right only":               {map[string]any{"sides": sourcesBody("ml", map[string]int{"right": 80})}, "right", 80},
		"left only, side sent":     {map[string]any{"sides": sourcesBody("ml", map[string]int{"left": 70}), "side": "left"}, "left", 70},
		"both, side sent":          {map[string]any{"side": "both"}, "both", 110},
		"keys in the other order":  {map[string]any{"sides": sourcesBody("ml", map[string]int{"right": 5, "left": 6})}, "both", 11},
		"one millilitre each":      {map[string]any{"sides": sourcesBody("ml", map[string]int{"left": 1, "right": 1})}, "both", 2},
		"the cap exactly":          {map[string]any{"sides": sourcesBody("ml", map[string]int{"left": 1000, "right": 1000})}, "both", 2000},
		"a single side at the cap": {map[string]any{"sides": sourcesBody("ml", map[string]int{"right": 2000})}, "right", 2000},
	} {
		rec := k.do(http.MethodPost, k.pumpingPath(kid), ana, advancedPumpingBody(c.over))
		m := bodyMap(t, rec)
		if rec.Code != http.StatusCreated || m["side"] != c.side || m["volume"].(map[string]any)["ml"] != c.ml {
			t.Errorf("%s: %d %v", name, rec.Code, m)
		}
	}
	rec := k.do(http.MethodPost, k.pumpingPath(kid), ana, advancedPumpingBody(map[string]any{"sides": sourcesBody("ml", map[string]int{"right": 5, "left": 6})}))
	var order []string
	for _, s := range bodyMap(t, rec)["sides"].([]any) {
		order = append(order, s.(map[string]any)["side"].(string))
	}
	if !slices.Equal(order, []string{"left", "right"}) {
		t.Errorf("sides in view order %v", order)
	}
}

func TestAdvancedPumpingOunces(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	rec := k.do(http.MethodPost, k.pumpingPath(kid), ana, advancedPumpingBody(map[string]any{
		"sides": sourcesBody("floz", map[string]int{"left": 150, "right": 250}),
	}))
	if rec.Code != http.StatusCreated {
		t.Fatalf("%d %q", rec.Code, rec.Body.String())
	}
	m := bodyMap(t, rec)
	if m["side"] != "both" ||
		!equalJSON(m["volume"], map[string]any{"ml": 118.0, "floz_hundredths": 400.0, "typed_unit": "floz"}) ||
		!equalJSON(m["sides"], []any{sideView("left", 44, 150, "floz"), sideView("right", 74, 250, "floz")}) {
		t.Errorf("view = %v", m)
	}
}

func TestPumpingSumIsTheTotal(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	id := bodyMap(t, k.do(http.MethodPost, k.pumpingPath(kid), ana, advancedPumpingBody(nil)))["id"].(string)
	one := k.pumpingPath(kid, id)

	rec := k.do(http.MethodPatch, one, ana, map[string]any{"sides": sourcesBody("ml", map[string]int{"left": 60, "right": 100})})
	m := bodyMap(t, rec)
	if rec.Code != http.StatusOK || m["side"] != "both" ||
		!equalJSON(m["volume"], map[string]any{"ml": 160.0, "floz_hundredths": 541.0, "typed_unit": "ml"}) ||
		!equalJSON(m["sides"], []any{sideView("left", 60, 203, "ml"), sideView("right", 100, 338, "ml")}) {
		t.Fatalf("after changing one side: %d %v", rec.Code, m)
	}
	stored := k.do(http.MethodGet, one, ana, nil).Body.String()

	rec = k.do(http.MethodPatch, one, ana, map[string]any{"note": "only the note"})
	m = bodyMap(t, rec)
	if rec.Code != http.StatusOK || m["note"] != "only the note" {
		t.Fatalf("note patch: %d %q", rec.Code, rec.Body.String())
	}
	was := map[string]any{}
	if err := json.Unmarshal([]byte(stored), &was); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"volume", "sides", "side", "started_at", "duration_seconds"} {
		if !equalJSON(m[key], was[key]) {
			t.Errorf("a note-only patch changed %s: %v, was %v", key, m[key], was[key])
		}
	}

	// Typed in ounces, the sum of the typed amounts is exact whatever the ml columns round to.
	rec = k.do(http.MethodPatch, one, ana, map[string]any{"sides": sourcesBody("floz", map[string]int{"left": 101, "right": 102})})
	if m := bodyMap(t, rec); rec.Code != http.StatusOK || m["volume"].(map[string]any)["floz_hundredths"] != 203.0 || m["volume"].(map[string]any)["typed_unit"] != "floz" {
		t.Errorf("ounce sum: %d %v", rec.Code, m)
	}

	// Dropping a side re-derives the side; the duration and start are untouched.
	rec = k.do(http.MethodPatch, one, ana, map[string]any{"sides": sourcesBody("ml", map[string]int{"right": 90})})
	if m := bodyMap(t, rec); rec.Code != http.StatusOK || m["side"] != "right" || m["duration_seconds"] != 900.0 ||
		!equalJSON(m["sides"], []any{sideView("right", 90, 304, "ml")}) || m["volume"].(map[string]any)["ml"] != 90.0 {
		t.Errorf("dropping a side: %d %v", rec.Code, m)
	}
}

// A stored side triple that NewVolume would not produce survives an edit that resends its typed amount.
func TestUnchangedSideIsNotRecomputed(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	odd := units.Volume{ML: 119, FlOzHundredths: 400, Typed: units.UnitFlOz}
	p, err := k.Store.CreatePumping(context.Background(), kid, k.parentID("ana"), model.PumpingSession{
		StartedAt: t0.Add(-time.Hour), DurationSeconds: 600, Side: "both", Volume: units.Volume{ML: 150, FlOzHundredths: 600, Typed: units.UnitFlOz},
		Sides: []model.PumpingSide{{Side: "left", Volume: odd}, {Side: "right", Volume: units.Volume{ML: 59, FlOzHundredths: 200, Typed: units.UnitFlOz}}},
	}, t0)
	if err != nil {
		t.Fatal(err)
	}
	rec := k.do(http.MethodPatch, k.pumpingPath(kid, p.ID), ana, map[string]any{"sides": sourcesBody("floz", map[string]int{"left": 400, "right": 300})})
	m := bodyMap(t, rec)
	if rec.Code != http.StatusOK ||
		!equalJSON(m["sides"], []any{sideView("left", 119, 400, "floz"), sideView("right", 89, 300, "floz")}) ||
		m["volume"].(map[string]any)["floz_hundredths"] != 700.0 {
		t.Errorf("after resending 4.00 and changing the other: %d %v", rec.Code, m)
	}
}

func TestPumpingSwitchToSimpleAndBack(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	id := bodyMap(t, k.do(http.MethodPost, k.pumpingPath(kid), ana, advancedPumpingBody(nil)))["id"].(string)
	one := k.pumpingPath(kid, id)

	// A side change that the sides contradict is refused; nothing is lost. The same side, resent, is no change.
	before := k.do(http.MethodGet, one, ana, nil).Body.String()
	wantInvalid2(t, "left over both sides", k.do(http.MethodPatch, one, ana, map[string]any{"side": "left"}), "side")
	wantInvalid2(t, "right over both sides", k.do(http.MethodPatch, one, ana, map[string]any{"side": "right"}), "side")
	wantInvalid2(t, "an unknown side", k.do(http.MethodPatch, one, ana, map[string]any{"side": "middle"}), "side")
	if got := k.do(http.MethodGet, one, ana, nil).Body.String(); got != before {
		t.Errorf("a refused patch changed the session:\n got %s\nwant %s", got, before)
	}
	if rec := k.do(http.MethodPatch, one, ana, map[string]any{"side": "both"}); rec.Code != http.StatusOK || rec.Body.String() != before {
		t.Errorf("resending the side: %d %q", rec.Code, rec.Body.String())
	}

	// Advanced to simple: the sides go, the new total stays, and the side comes from the body or stays.
	rec := k.do(http.MethodPatch, one, ana, map[string]any{"volume": volumeBody("ml", 200), "side": "left"})
	m := bodyMap(t, rec)
	if rec.Code != http.StatusOK || !equalJSON(m["sides"], []any{}) || m["side"] != "left" ||
		!equalJSON(m["volume"], map[string]any{"ml": 200.0, "floz_hundredths": 676.0, "typed_unit": "ml"}) {
		t.Fatalf("to simple: %d %v", rec.Code, m)
	}
	if got := bodyMap(t, k.do(http.MethodGet, one, ana, nil)); !equalJSON(got, m) {
		t.Errorf("stored after switching = %v, want %v", got, m)
	}
	// A simple session's side changes freely again.
	if rec := k.do(http.MethodPatch, one, ana, map[string]any{"side": "right"}); rec.Code != http.StatusOK || bodyMap(t, rec)["side"] != "right" {
		t.Errorf("side of a simple session: %d %q", rec.Code, rec.Body.String())
	}

	// Simple to advanced: the sides appear, replace the total, and the derived side replaces the stored one.
	rec = k.do(http.MethodPatch, one, ana, map[string]any{"sides": sourcesBody("ml", map[string]int{"left": 50, "right": 25})})
	m = bodyMap(t, rec)
	if rec.Code != http.StatusOK || m["side"] != "both" ||
		!equalJSON(m["volume"], map[string]any{"ml": 75.0, "floz_hundredths": 254.0, "typed_unit": "ml"}) ||
		!equalJSON(m["sides"], []any{sideView("left", 50, 169, "ml"), sideView("right", 25, 85, "ml")}) {
		t.Fatalf("to advanced: %d %v", rec.Code, m)
	}

	// A volume without a side on an advanced session clears the sides and keeps the derived side.
	rec = k.do(http.MethodPatch, one, ana, map[string]any{"volume": volumeBody("ml", 80)})
	if m := bodyMap(t, rec); rec.Code != http.StatusOK || m["side"] != "both" || !equalJSON(m["sides"], []any{}) || m["volume"].(map[string]any)["ml"] != 80.0 {
		t.Errorf("volume alone: %d %v", rec.Code, m)
	}
}

func TestBothVolumeAndSides(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	id := bodyMap(t, k.do(http.MethodPost, k.pumpingPath(kid), ana, advancedPumpingBody(nil)))["id"].(string)
	before := k.do(http.MethodGet, k.pumpingPath(kid, id), ana, nil).Body.String()

	both := map[string]any{"volume": volumeBody("ml", 100)}
	wantInvalid2(t, "create", k.do(http.MethodPost, k.pumpingPath(kid), ana, advancedPumpingBody(both)), "sides")
	wantInvalid2(t, "patch", k.do(http.MethodPatch, k.pumpingPath(kid, id), ana, withOverrides(both, map[string]any{
		"sides": sourcesBody("ml", map[string]int{"left": 10})})), "sides")
	if got := k.do(http.MethodGet, k.pumpingPath(kid, id), ana, nil).Body.String(); got != before {
		t.Errorf("a refused patch changed the session: %s", got)
	}
}

func TestSideBesideSidesMustMatch(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	id := bodyMap(t, k.do(http.MethodPost, k.pumpingPath(kid), ana, advancedPumpingBody(nil)))["id"].(string)
	before := k.do(http.MethodGet, k.pumpingPath(kid, id), ana, nil).Body.String()

	for name, c := range map[string]struct {
		sides map[string]int
		side  string
	}{
		"both beside the left only":  {map[string]int{"left": 10}, "both"},
		"right beside the left only": {map[string]int{"left": 10}, "right"},
		"left beside the right only": {map[string]int{"right": 10}, "left"},
		"left beside two sides":      {map[string]int{"left": 10, "right": 10}, "left"},
		"right beside two sides":     {map[string]int{"left": 10, "right": 10}, "right"},
		"an unknown side":            {map[string]int{"left": 10}, "middle"},
		"an empty side":              {map[string]int{"left": 10}, ""},
	} {
		over := map[string]any{"sides": sourcesBody("ml", c.sides), "side": c.side}
		wantInvalid2(t, name+" (create)", k.do(http.MethodPost, k.pumpingPath(kid), ana, advancedPumpingBody(over)), "side")
		wantInvalid2(t, name+" (patch)", k.do(http.MethodPatch, k.pumpingPath(kid, id), ana, over), "side")
	}
	if got := k.do(http.MethodGet, k.pumpingPath(kid, id), ana, nil).Body.String(); got != before {
		t.Errorf("a refused patch changed the session:\n got %s\nwant %s", got, before)
	}
	if n := len(bodyMap(t, k.do(http.MethodGet, k.pumpingPath(kid), ana, nil))["pumping"].([]any)); n != 1 {
		t.Errorf("%d sessions after the refused creates, want 1", n)
	}
}

func TestSidesValidation(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	id := bodyMap(t, k.do(http.MethodPost, k.pumpingPath(kid), ana, advancedPumpingBody(nil)))["id"].(string)
	stored := k.do(http.MethodGet, k.pumpingPath(kid, id), ana, nil).Body.String()
	two := func(unit string, a, b int) map[string]any {
		return sourcesBody(unit, map[string]int{"left": a, "right": b})
	}

	cases := []struct {
		name  string
		over  map[string]any
		field string
	}{
		{"unknown key", map[string]any{"sides": sourcesBody("ml", map[string]int{"middle": 10})}, "sides"},
		{"the combined side as a key", map[string]any{"sides": sourcesBody("ml", map[string]int{"both": 10})}, "sides"},
		{"an unknown key beside a good one", map[string]any{"sides": sourcesBody("ml", map[string]int{"left": 10, "middle": 10})}, "sides"},
		{"wrong case", map[string]any{"sides": sourcesBody("ml", map[string]int{"Left": 10})}, "sides"},
		{"empty amounts", map[string]any{"sides": sourcesBody("ml", map[string]int{})}, "sides"},
		{"no amounts", map[string]any{"sides": map[string]any{"unit": "ml"}}, "sides"},
		{"no unit", map[string]any{"sides": map[string]any{"amounts": map[string]int{"left": 10}}}, "sides"},
		{"empty object", map[string]any{"sides": map[string]any{}}, "sides"},
		{"zero", map[string]any{"sides": two("ml", 0, 10)}, "sides"},
		{"negative", map[string]any{"sides": two("ml", -5, 10)}, "sides"},
		{"huge", map[string]any{"sides": two("ml", 1<<40, 10)}, "sides"},
		{"unknown unit", map[string]any{"sides": two("cup", 10, 10)}, "sides"},
		{"no unit and zero for a side not stored", map[string]any{"sides": map[string]any{"amounts": map[string]int{"left": 0}}}, "sides"},
		{"ounces that round to 0 ml", map[string]any{"sides": two("floz", 1, 100)}, "sides"},
		{"total over the cap", map[string]any{"sides": two("ml", 1200, 801)}, "volume"},
		{"total over the cap in ounces", map[string]any{"sides": two("floz", 4000, 3000)}, "volume"},
		{"one side over the cap", map[string]any{"sides": sourcesBody("ml", map[string]int{"right": 2001})}, "volume"},
		{"duration out of range", map[string]any{"duration_seconds": 0}, "duration_seconds"},
		{"a note over the limit", map[string]any{"note": strings.Repeat("ñ", constants.MaxNoteLen+1)}, "note"},
	}
	for _, c := range cases {
		wantInvalid2(t, c.name+" (create)", k.do(http.MethodPost, k.pumpingPath(kid), ana, advancedPumpingBody(c.over)), c.field)
		wantInvalid2(t, c.name+" (patch)", k.do(http.MethodPatch, k.pumpingPath(kid, id), ana, withOverrides(map[string]any{"sides": nil}, c.over)), c.field)
	}
	if got := k.do(http.MethodGet, k.pumpingPath(kid, id), ana, nil).Body.String(); got != stored {
		t.Errorf("a refused patch changed the session:\n got %s\nwant %s", got, stored)
	}
	if n := len(bodyMap(t, k.do(http.MethodGet, k.pumpingPath(kid), ana, nil))["pumping"].([]any)); n != 1 {
		t.Errorf("%d sessions after the refused creates, want 1", n)
	}

	for _, method := range []string{http.MethodPost, http.MethodPatch} {
		path := k.pumpingPath(kid)
		if method == http.MethodPatch {
			path = k.pumpingPath(kid, id)
		}
		rec := k.do(method, path, ana, advancedPumpingBody(map[string]any{"sides": map[string]any{"unit": "ml", "amounts": map[string]int{"left": 5}, "colour": "red"}}))
		if rec.Code != http.StatusUnprocessableEntity || strings.TrimSpace(rec.Body.String()) != `{"error":"unknown_field","field":"colour"}` {
			t.Errorf("%s unknown key inside sides: %d %q", method, rec.Code, rec.Body.String())
		}
	}
}

func TestSimplePumpingViewHasEmptySides(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	id := k.createPumping(ana, kid, nil)

	for name, path := range map[string]string{"get": k.pumpingPath(kid, id), "list": k.pumpingPath(kid), "timeline": "/v1/kids/" + kid + "/timeline"} {
		if body := k.do(http.MethodGet, path, ana, nil).Body.String(); !strings.Contains(body, `"sides":[]`) {
			t.Errorf("%s: a simple session's view has no empty sides array: %s", name, body)
		}
	}
	rec := k.do(http.MethodPost, k.pumpingPath(kid), ana, advancedPumpingBody(nil))
	if !strings.Contains(rec.Body.String(), `"sides":[{"side":"left"`) {
		t.Errorf("an advanced session's view: %s", rec.Body.String())
	}
}

func TestAdvancedPumpingOnTheTimelineAndList(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	created := bodyMap(t, k.do(http.MethodPost, k.pumpingPath(kid), ana, advancedPumpingBody(nil)))

	list := bodyMap(t, k.do(http.MethodGet, k.pumpingPath(kid), ana, nil))["pumping"].([]any)
	if len(list) != 1 || !equalJSON(list[0], created) {
		t.Errorf("list = %v, want %v", list, created)
	}
	items := bodyMap(t, k.do(http.MethodGet, "/v1/kids/"+kid+"/timeline?types=pumping", ana, nil))["items"].([]any)
	if len(items) != 1 || !equalJSON(items[0].(map[string]any)["sides"], created["sides"]) {
		t.Errorf("timeline = %v", items)
	}
}

func TestAdvancedPumpingRolesAndIdentity(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	xavi, admin := k.asParent("xavi"), k.asAdmin()
	id := bodyMap(t, k.do(http.MethodPost, k.pumpingPath(kid), ana, advancedPumpingBody(nil)))["id"].(string)
	body := advancedPumpingBody(nil)
	for name, token := range map[string]string{"a parent without a grant": xavi, "the admin": admin} {
		for _, c := range []struct {
			method, path string
			body         any
		}{
			{http.MethodPost, k.pumpingPath(kid), body},
			{http.MethodPatch, k.pumpingPath(kid, id), map[string]any{"sides": body["sides"]}},
			{http.MethodGet, k.pumpingPath(kid, id), nil},
		} {
			rec := k.do(c.method, c.path, token, c.body)
			unknown := k.do(c.method, strings.Replace(c.path, kid, model.NewID(), 1), token, c.body)
			if rec.Code != unknown.Code || rec.Body.String() != unknown.Body.String() {
				t.Errorf("%s %s %s: %d %q differs from an unknown kid's %d %q", name, c.method, c.path, rec.Code, rec.Body.String(), unknown.Code, unknown.Body.String())
			}
		}
	}
	if got := bodyMap(t, k.do(http.MethodGet, k.pumpingPath(kid, id), ana, nil)); len(got["sides"].([]any)) != 2 {
		t.Errorf("a refused write changed the session: %v", got)
	}
}

func TestNoSideValuesInLogs(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	k.Log.Reset()

	rec := k.do(http.MethodPost, k.pumpingPath(kid), ana, advancedPumpingBody(map[string]any{"sides": sourcesBody("ml", map[string]int{"left": 613, "right": 287})}))
	id := bodyMap(t, rec)["id"].(string)
	k.do(http.MethodPatch, k.pumpingPath(kid, id), ana, map[string]any{"sides": sourcesBody("floz", map[string]int{"left": 1733, "right": 1127})})
	k.do(http.MethodPatch, k.pumpingPath(kid, id), ana, map[string]any{"volume": volumeBody("ml", 100), "sides": sourcesBody("ml", map[string]int{"left": 441})}) // a refusal
	k.do(http.MethodPatch, k.pumpingPath(kid, id), ana, map[string]any{"volume": volumeBody("ml", 100)})
	k.do(http.MethodDelete, k.pumpingPath(kid, id), ana, nil)

	wantCleanLog(t, k.Log.String(), 5, "613", "287", "900", "1733", "1127", "2860", "441", "left", "right", "sides", "amounts")
}
