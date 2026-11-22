package httpapi

import (
	"context"
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/units"
)

// trackerKid creates "ana" (guardian) and "bob" (caregiver) on a kid born 2026-03-04 in Europe/Madrid.
func (k *kit) trackerKid() (kid, guardian, caregiver string) {
	k.t.Helper()
	guardian, caregiver = k.asParent("ana"), k.asParent("bob")
	kid = k.createKid(guardian, nil)
	k.grant(guardian, kid, "bob", "caregiver")
	return kid, guardian, caregiver
}

func ts(t time.Time) string { return t.UTC().Format(time.RFC3339) }

// withOverrides copies base and applies over; a nil value drops the key.
func withOverrides(base, over map[string]any) map[string]any {
	b := maps.Clone(base)
	maps.Copy(b, over)
	for key, v := range over {
		if v == nil {
			delete(b, key)
		}
	}
	return b
}

func bottleBody(over map[string]any) map[string]any {
	return withOverrides(map[string]any{
		"at":       ts(t0.Add(-time.Hour)),
		"contents": []string{model.ContentFormula},
		"volume":   map[string]any{"unit": "ml", "amount": 120},
		"note":     "first",
	}, over)
}

func volumeBody(unit string, amount int) map[string]any {
	return map[string]any{"unit": unit, "amount": amount}
}

// createBottle posts a bottle and returns the view's id; the status must be 201.
func (k *kit) createBottle(token, kid string, over map[string]any) string {
	k.t.Helper()
	rec := k.do(http.MethodPost, "/v1/kids/"+kid+"/bottles", token, bottleBody(over))
	if rec.Code != http.StatusCreated {
		k.t.Fatalf("create bottle: %d %q", rec.Code, rec.Body.String())
	}
	return bodyMap(k.t, rec)["id"].(string)
}

func TestBottleRoutesRoles(t *testing.T) {
	k := newKit(t)
	kid, ana, bo := k.trackerKid()
	base := "/v1/kids/" + kid + "/bottles"

	for name, token := range map[string]string{"guardian": ana, "caregiver": bo} {
		rec := k.do(http.MethodPost, base, token, bottleBody(nil))
		if rec.Code != http.StatusCreated {
			t.Fatalf("%s create: %d %q", name, rec.Code, rec.Body.String())
		}
		m := bodyMap(t, rec)
		sameKeys(t, m, "id", "at", "contents", "volume", "note")
		id := m["id"].(string)
		if m["at"] != ts(t0.Add(-time.Hour)) || m["note"] != "first" || !equalJSON(m["contents"], []any{"formula"}) ||
			!equalJSON(m["volume"], map[string]any{"ml": 120.0, "floz_hundredths": 406.0, "typed_unit": "ml"}) {
			t.Errorf("%s create view = %v", name, m)
		}

		one := base + "/" + id
		if rec := k.do(http.MethodGet, one, token, nil); rec.Code != http.StatusOK || !equalJSON(bodyMap(t, rec), m) {
			t.Errorf("%s get: %d %q", name, rec.Code, rec.Body.String())
		}
		rec = k.do(http.MethodPatch, one, token, map[string]any{"note": "edited", "contents": []string{model.ContentCowMilk, model.ContentFormula}})
		if rec.Code != http.StatusOK {
			t.Fatalf("%s patch: %d %q", name, rec.Code, rec.Body.String())
		}
		if p := bodyMap(t, rec); p["note"] != "edited" || p["id"] != id || !equalJSON(p["contents"], []any{"formula", "cow_milk"}) {
			t.Errorf("%s patch view = %v", name, p)
		}
		if rec := k.do(http.MethodDelete, one, token, nil); rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
			t.Errorf("%s delete: %d %q", name, rec.Code, rec.Body.String())
		}
		wantErr(t, k.do(http.MethodGet, one, token, nil), http.StatusNotFound, "not_found")
	}
}

func equalJSON(a, b any) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}

// Every refusal below must be what the same route answers elsewhere in the kid API.
func TestBottleRoutesRefuseOutsiders(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	other := k.createKid(ana, map[string]any{"name": "Second"})
	xavi, admin := k.asParent("xavi"), k.asAdmin()
	id := k.createBottle(ana, kid, nil)
	otherID := k.createBottle(ana, other, nil)

	type call struct {
		method, path string
		body         any
	}
	calls := func(kidPath, entry string) []call {
		return []call{
			{http.MethodGet, "/v1/kids/" + kidPath + "/bottles", nil},
			{http.MethodPost, "/v1/kids/" + kidPath + "/bottles", bottleBody(nil)},
			{http.MethodGet, "/v1/kids/" + kidPath + "/bottles/" + entry, nil},
			{http.MethodPatch, "/v1/kids/" + kidPath + "/bottles/" + entry, map[string]any{"note": "x"}},
			{http.MethodDelete, "/v1/kids/" + kidPath + "/bottles/" + entry, nil},
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
		if rec := k.do(http.MethodGet, "/v1/kids/"+kidID+"/bottles/"+entry, ana, nil); rec.Code != http.StatusOK || bodyMap(t, rec)["note"] != "first" {
			t.Errorf("entry %s changed or vanished: %d %q", entry, rec.Code, rec.Body.String())
		}
	}
}

func TestBottleCreateDefaultsAtToNow(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	rec := k.do(http.MethodPost, "/v1/kids/"+kid+"/bottles", ana, bottleBody(map[string]any{"at": nil, "note": nil}))
	if rec.Code != http.StatusCreated {
		t.Fatalf("%d %q", rec.Code, rec.Body.String())
	}
	if m := bodyMap(t, rec); m["at"] != ts(t0) || m["note"] != nil {
		t.Errorf("view = %v, want at now and a null note", m)
	}
}

func TestBottleValidation(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	base := "/v1/kids/" + kid + "/bottles"
	id := k.createBottle(ana, kid, nil)
	stored := k.do(http.MethodGet, base+"/"+id, ana, nil).Body.String()

	cases := []struct {
		name       string
		over       map[string]any
		field      string
		createOnly bool // a missing key on PATCH keeps the stored value
	}{
		{"more than 5 minutes ahead", map[string]any{"at": ts(t0.Add(constants.MaxFutureSkew + time.Second))}, "at", false},
		{"before the birth date", map[string]any{"at": "2026-03-03T22:59:59Z"}, "at", false},
		{"volume 0", map[string]any{"volume": volumeBody("ml", 0)}, "volume", false},
		{"volume -1", map[string]any{"volume": volumeBody("ml", -1)}, "volume", false},
		{"over the cap", map[string]any{"volume": volumeBody("ml", constants.MaxBottleVolumeML+1)}, "volume", false},
		{"over the cap in ounces", map[string]any{"volume": volumeBody("floz", 3500)}, "volume", false},
		{"ounces that round to 0 ml", map[string]any{"volume": volumeBody("floz", 1)}, "volume", false},
		{"unknown unit", map[string]any{"volume": volumeBody("cup", 1)}, "volume", false},
		{"no amount", map[string]any{"volume": map[string]any{"unit": "ml"}}, "volume", false},
		{"no volume", map[string]any{"volume": nil}, "volume", true},
		{"empty contents", map[string]any{"contents": []string{}}, "contents", false},
		{"no contents", map[string]any{"contents": nil}, "contents", true},
		{"unknown contents", map[string]any{"contents": []string{"goat_milk"}}, "contents", false},
		{"duplicate contents", map[string]any{"contents": []string{"formula", "formula"}}, "contents", false},
		{"note over the limit", map[string]any{"note": strings.Repeat("ñ", constants.MaxNoteLen+1)}, "note", false},
	}
	for _, c := range cases {
		wantInvalid2(t, c.name+" (create)", k.do(http.MethodPost, base, ana, bottleBody(c.over)), c.field)
		if !c.createOnly {
			wantInvalid2(t, c.name+" (patch)", k.do(http.MethodPatch, base+"/"+id, ana, c.over), c.field)
		}
	}
	if got := k.do(http.MethodGet, base+"/"+id, ana, nil).Body.String(); got != stored {
		t.Errorf("a refused patch changed the entry:\n got %s\nwant %s", got, stored)
	}
	if n := len(bodyMap(t, k.do(http.MethodGet, base, ana, nil))["bottles"].([]any)); n != 1 {
		t.Errorf("%d bottles after the refused creates, want 1", n)
	}

	for _, c := range []struct {
		name string
		over map[string]any
	}{
		{"exactly 5 minutes ahead", map[string]any{"at": ts(t0.Add(constants.MaxFutureSkew))}},
		{"the start of the birth date in the kid's zone", map[string]any{"at": "2026-03-03T23:00:00Z"}},
		{"the volume cap", map[string]any{"volume": volumeBody("ml", constants.MaxBottleVolumeML)}},
		{"one millilitre", map[string]any{"volume": volumeBody("ml", 1)}},
		{"a note at the limit", map[string]any{"note": strings.Repeat("ñ", constants.MaxNoteLen)}},
		{"all four contents", map[string]any{"contents": []string{"other", "cow_milk", "formula", "breast_milk"}}},
	} {
		if rec := k.do(http.MethodPost, base, ana, bottleBody(c.over)); rec.Code != http.StatusCreated {
			t.Errorf("%s: %d %q", c.name, rec.Code, rec.Body.String())
		}
	}

	for _, method := range []string{http.MethodPost, http.MethodPatch} {
		path := base
		if method == http.MethodPatch {
			path += "/" + id
		}
		rec := k.do(method, path, ana, bottleBody(map[string]any{"colour": "red"}))
		if rec.Code != http.StatusUnprocessableEntity || strings.TrimSpace(rec.Body.String()) != `{"error":"unknown_field","field":"colour"}` {
			t.Errorf("%s unknown field: %d %q", method, rec.Code, rec.Body.String())
		}
		wantErr(t, k.do(method, path, ana, bottleBody(map[string]any{"at": "yesterday"})), http.StatusBadRequest, "bad_request")
	}
}

func wantInvalid2(t *testing.T, name string, rec *httptest.ResponseRecorder, field string) {
	t.Helper()
	want := `{"error":"invalid","field":"` + field + `"}`
	if rec.Code != http.StatusUnprocessableEntity || strings.TrimSpace(rec.Body.String()) != want {
		t.Errorf("%s: got %d %q, want 422 %s", name, rec.Code, rec.Body.String(), want)
	}
}

func TestPatchMergesAndClears(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	id := k.createBottle(ana, kid, nil)
	one := "/v1/kids/" + kid + "/bottles/" + id

	rec := k.do(http.MethodPatch, one, ana, map[string]any{})
	if m := bodyMap(t, rec); rec.Code != http.StatusOK || m["note"] != "first" || m["at"] != ts(t0.Add(-time.Hour)) {
		t.Errorf("empty patch: %d %v", rec.Code, m)
	}
	rec = k.do(http.MethodPatch, one, ana, map[string]any{"note": nil, "at": ts(t0.Add(-2 * time.Hour))})
	if m := bodyMap(t, rec); rec.Code != http.StatusOK || m["note"] != nil || m["at"] != ts(t0.Add(-2*time.Hour)) {
		t.Errorf("clear note, move at: %d %v", rec.Code, m)
	}
	if m := bodyMap(t, k.do(http.MethodGet, one, ana, nil)); m["note"] != nil || !equalJSON(m["contents"], []any{"formula"}) {
		t.Errorf("stored after patch = %v", m)
	}
}

func TestEditingNoteKeepsTypedVolume(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	id := k.createBottle(ana, kid, map[string]any{"volume": volumeBody("floz", 400)})
	one := "/v1/kids/" + kid + "/bottles/" + id
	oz := map[string]any{"ml": 118.0, "floz_hundredths": 400.0, "typed_unit": "floz"}

	view := func(rec *httptest.ResponseRecorder) any {
		t.Helper()
		if rec.Code != http.StatusOK && rec.Code != http.StatusCreated {
			t.Fatalf("%d %q", rec.Code, rec.Body.String())
		}
		return bodyMap(t, rec)["volume"]
	}
	if got := view(k.do(http.MethodGet, one, ana, nil)); !equalJSON(got, oz) {
		t.Fatalf("created volume = %v", got)
	}
	if got := view(k.do(http.MethodPatch, one, ana, map[string]any{"note": "only the note"})); !equalJSON(got, oz) {
		t.Errorf("after a note edit = %v", got)
	}
	if got := view(k.do(http.MethodPatch, one, ana, map[string]any{"volume": volumeBody("floz", 400)})); !equalJSON(got, oz) {
		t.Errorf("after resending the same volume = %v", got)
	}
	if got := view(k.do(http.MethodPatch, one, ana, map[string]any{"volume": volumeBody("ml", 120)})); !equalJSON(got, map[string]any{"ml": 120.0, "floz_hundredths": 406.0, "typed_unit": "ml"}) {
		t.Errorf("after entering 120 ml = %v", got)
	}
	if got := view(k.do(http.MethodPatch, one, ana, map[string]any{"volume": volumeBody("floz", 450)})); !equalJSON(got, map[string]any{"ml": 133.0, "floz_hundredths": 450.0, "typed_unit": "floz"}) {
		t.Errorf("after entering 4.50 oz = %v", got)
	}
}

// A stored triple that NewVolume would not produce survives an edit that resends the typed value.
func TestUnchangedVolumeIsNotRecomputed(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	odd := model.BottleFeed{At: t0.Add(-time.Hour), Contents: []string{model.ContentFormula},
		Volume: units.Volume{ML: 119, FlOzHundredths: 400, Typed: units.UnitFlOz}}
	b, err := k.Store.CreateBottle(context.Background(), kid, k.parentID("ana"), odd, t0)
	if err != nil {
		t.Fatal(err)
	}
	rec := k.do(http.MethodPatch, "/v1/kids/"+kid+"/bottles/"+b.ID, ana, map[string]any{"volume": volumeBody("floz", 400), "note": "x"})
	if got := bodyMap(t, rec)["volume"]; rec.Code != http.StatusOK || !equalJSON(got, map[string]any{"ml": 119.0, "floz_hundredths": 400.0, "typed_unit": "floz"}) {
		t.Errorf("volume = %v (%d)", got, rec.Code)
	}
}

func TestBottleListRange(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	other := k.createKid(ana, map[string]any{"name": "Second"})
	k.createBottle(ana, other, nil)
	base := "/v1/kids/" + kid + "/bottles"

	at := func(m time.Duration) map[string]any { return map[string]any{"at": ts(t0.Add(-m * time.Minute))} }
	a0 := k.createBottle(ana, kid, at(1))
	a1 := k.createBottle(ana, kid, at(2))
	a2 := k.createBottle(ana, kid, at(2))
	a3 := k.createBottle(ana, kid, at(3))
	a4 := k.createBottle(ana, kid, at(4))
	tied := []string{a1, a2}
	slices.Sort(tied)
	slices.Reverse(tied)
	all := append(append([]string{a0}, tied...), a3, a4)

	page := func(query string) ([]string, any) {
		t.Helper()
		rec := k.do(http.MethodGet, base+query, ana, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s: %d %q", query, rec.Code, rec.Body.String())
		}
		m := bodyMap(t, rec)
		sameKeys(t, m, "bottles", "next_cursor")
		var ids []string
		for _, e := range m["bottles"].([]any) {
			ids = append(ids, e.(map[string]any)["id"].(string))
		}
		return ids, m["next_cursor"]
	}

	if ids, next := page(""); !slices.Equal(ids, all) || next != nil {
		t.Errorf("default = %v next %v, want %v and null", ids, next, all)
	}
	if ids, next := page("?limit=5"); !slices.Equal(ids, all) || next != nil {
		t.Errorf("limit equal to the count = %v next %v, want one page and null", ids, next)
	}
	var paged []string
	query := "?limit=2"
	for i := range 3 {
		ids, next := page(query)
		paged = append(paged, ids...)
		if i < 2 {
			c, ok := next.(string)
			if !ok || len(ids) != 2 {
				t.Fatalf("page %d = %v next %v, want 2 rows and a cursor", i, ids, next)
			}
			query = "?limit=2&cursor=" + url.QueryEscape(c)
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
		t.Errorf("[-3m, -1m) = %v, want the 2m pair then the 3m entry (the 1m entry is excluded)", ids)
	}
	// A fractional second truncates toward zero: [-3m+0.9s, -2m+0.9s) is [-3m, -2m).
	if ids, _ := page(rangeQ(t0.Add(-3*time.Minute+900*time.Millisecond), t0.Add(-2*time.Minute+900*time.Millisecond))); !slices.Equal(ids, []string{a3}) {
		t.Errorf("fractional range = %v, want only %s", ids, a3)
	}

	for query, field := range map[string]string{
		"?from=yesterday": "from",
		"?to=2026":        "to",
		"?from=2026-11-20T00:00:00Z&to=2026-11-19T00:00:00Z": "from",
		"?limit=0":   "limit",
		"?limit=101": "limit",
		"?limit=abc": "limit",
		"?cursor=!!": "cursor",
	} {
		wantInvalid(t, k.do(http.MethodGet, base+query, ana, nil), field)
	}
}

func TestNoValueInLogs(t *testing.T) {
	const (
		note     = "xyzzy-private-note"
		mlAmount = 639
		ozAmount = 2161 // hundredths of an ounce
	)
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	k.Log.Reset()

	rec := k.do(http.MethodPost, "/v1/kids/"+kid+"/bottles", ana, bottleBody(map[string]any{"note": note, "volume": volumeBody("ml", mlAmount)}))
	id := bodyMap(t, rec)["id"].(string)
	k.do(http.MethodPatch, "/v1/kids/"+kid+"/bottles/"+id, ana, map[string]any{"note": note + "2", "volume": volumeBody("floz", ozAmount), "contents": []string{"cow_milk"}})
	k.do(http.MethodPatch, "/v1/kids/"+kid+"/bottles/"+id, ana, map[string]any{"volume": volumeBody("ml", 0)}) // a refusal
	k.do(http.MethodDelete, "/v1/kids/"+kid+"/bottles/"+id, ana, nil)
	d := k.do(http.MethodPost, "/v1/kids/"+kid+"/diapers", ana, map[string]any{"kind": "wet", "note": note})
	k.do(http.MethodPatch, "/v1/kids/"+kid+"/diapers/"+bodyMap(t, d)["id"].(string), ana, map[string]any{"note": note + "3"})

	lines := strings.Split(strings.TrimSpace(k.Log.String()), "\n")
	if len(lines) != 6 {
		t.Fatalf("%d log lines, want 6:\n%s", len(lines), k.Log.String())
	}
	allowed := []string{"time", "level", "msg", "route", "status", "principal_kind", "request_id", "duration_ms", constants.LogFieldCause, constants.LogFieldSlide}
	for _, line := range lines {
		if strings.Contains(line, "xyzzy") {
			t.Errorf("a note reached the log: %s", line)
		}
		var fields map[string]any
		if err := json.Unmarshal([]byte(line), &fields); err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		for key, v := range fields {
			if !slices.Contains(allowed, key) {
				t.Errorf("unexpected log field %q in %s", key, line)
			}
			s, _ := json.Marshal(v)
			if key != "request_id" && key != "time" {
				for _, banned := range []string{"639", "2161", "cow_milk", "formula", "contents", "volume"} {
					if strings.Contains(string(s), banned) {
						t.Errorf("log field %s = %s carries %q", key, s, banned)
					}
				}
			}
		}
	}
}
