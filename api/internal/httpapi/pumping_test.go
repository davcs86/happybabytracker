package httpapi

import (
	"encoding/json"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
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
		sameKeys(t, m, "id", "started_at", "duration_seconds", "volume", "side", "note")
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

	lines := strings.Split(strings.TrimSpace(k.Log.String()), "\n")
	if len(lines) != 8 {
		t.Fatalf("%d log lines, want 8:\n%s", len(lines), k.Log.String())
	}
	allowed := []string{"time", "level", "msg", "route", "status", "principal_kind", "request_id", "duration_ms", constants.LogFieldCause, constants.LogFieldSlide}
	for _, line := range lines {
		if strings.Contains(line, "zz") {
			t.Errorf("a text value reached the log: %s", line)
		}
		var fields map[string]any
		if err := json.Unmarshal([]byte(line), &fields); err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		for key, v := range fields {
			if !slices.Contains(allowed, key) {
				t.Errorf("unexpected log field %q in %s", key, line)
			}
			text, isText := v.(string)
			if !isText || key == "request_id" || key == "time" {
				continue
			}
			for _, banned := range []string{"1357", "4619", "777", "mango", "rash"} {
				if strings.Contains(text, banned) {
					t.Errorf("log field %s = %q carries %q", key, text, banned)
				}
			}
		}
	}
}
