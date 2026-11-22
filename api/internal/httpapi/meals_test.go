package httpapi

import (
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
)

func mealBody(over map[string]any) map[string]any {
	return withOverrides(map[string]any{
		"at": ts(t0.Add(-time.Hour)), "what": "banana", "reaction": "liked it", "note": "first",
	}, over)
}

func (k *kit) createMeal(token, kid string, over map[string]any) string {
	k.t.Helper()
	rec := k.do(http.MethodPost, "/v1/kids/"+kid+"/meals", token, mealBody(over))
	if rec.Code != http.StatusCreated {
		k.t.Fatalf("create meal: %d %q", rec.Code, rec.Body.String())
	}
	return bodyMap(k.t, rec)["id"].(string)
}

func TestMealRoutes(t *testing.T) {
	k := newKit(t)
	kid, ana, bo := k.trackerKid()
	other := k.createKid(ana, map[string]any{"name": "Second"})
	xavi, admin := k.asParent("xavi"), k.asAdmin()
	base := "/v1/kids/" + kid + "/meals"

	for name, token := range map[string]string{"guardian": ana, "caregiver": bo} {
		rec := k.do(http.MethodPost, base, token, mealBody(nil))
		if rec.Code != http.StatusCreated {
			t.Fatalf("%s create: %d %q", name, rec.Code, rec.Body.String())
		}
		m := bodyMap(t, rec)
		sameKeys(t, m, "id", "at", "what", "reaction", "note")
		if m["at"] != ts(t0.Add(-time.Hour)) || m["what"] != "banana" || m["reaction"] != "liked it" || m["note"] != "first" {
			t.Errorf("%s view = %v", name, m)
		}
		one := base + "/" + m["id"].(string)
		if rec := k.do(http.MethodGet, one, token, nil); rec.Code != http.StatusOK || !equalJSON(bodyMap(t, rec), m) {
			t.Errorf("%s get: %d %q", name, rec.Code, rec.Body.String())
		}
		rec = k.do(http.MethodPatch, one, token, map[string]any{"what": "pear", "reaction": nil})
		if p := bodyMap(t, rec); rec.Code != http.StatusOK || p["what"] != "pear" || p["reaction"] != nil || p["note"] != "first" || p["at"] != m["at"] {
			t.Errorf("%s patch: %d %v", name, rec.Code, p)
		}
		if rec := k.do(http.MethodDelete, one, token, nil); rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
			t.Errorf("%s delete: %d %q", name, rec.Code, rec.Body.String())
		}
		wantErr(t, k.do(http.MethodGet, one, token, nil), http.StatusNotFound, "not_found")
	}

	id, otherID := k.createMeal(ana, kid, nil), k.createMeal(ana, other, nil)
	for _, c := range []struct {
		method, path string
		body         any
	}{
		{http.MethodGet, base, nil},
		{http.MethodPost, base, mealBody(nil)},
		{http.MethodGet, base + "/" + id, nil},
		{http.MethodPatch, base + "/" + id, map[string]any{"what": "x"}},
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
			wantErr(t, k.do(method, base+"/"+url.PathEscape(entry), ana, map[string]any{"what": "x"}), http.StatusNotFound, "not_found")
		}
	}
	if rec := k.do(http.MethodGet, "/v1/kids/"+other+"/meals/"+otherID, ana, nil); rec.Code != http.StatusOK || bodyMap(t, rec)["what"] != "banana" {
		t.Errorf("the other kid's meal changed: %d %q", rec.Code, rec.Body.String())
	}
}

func TestMealCreateDefaultsAtToNow(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	rec := k.do(http.MethodPost, "/v1/kids/"+kid+"/meals", ana, map[string]any{"what": "toast"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("%d %q", rec.Code, rec.Body.String())
	}
	if m := bodyMap(t, rec); m["at"] != ts(t0) || m["reaction"] != nil || m["note"] != nil {
		t.Errorf("view = %v, want at now and null reaction and note", m)
	}
}

func TestMealWhatIsRequiredAndTrimmed(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	base := "/v1/kids/" + kid + "/meals"

	rec := k.do(http.MethodPost, base, ana, mealBody(map[string]any{"what": " \t banana\n "}))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %q", rec.Code, rec.Body.String())
	}
	m := bodyMap(t, rec)
	if m["what"] != "banana" {
		t.Errorf("created what = %q, want it trimmed", m["what"])
	}
	one := base + "/" + m["id"].(string)
	if got := bodyMap(t, k.do(http.MethodGet, one, ana, nil)); got["what"] != "banana" {
		t.Errorf("stored what = %q, want it trimmed", got["what"])
	}
	rec = k.do(http.MethodPatch, one, ana, map[string]any{"what": "  pear "})
	if p := bodyMap(t, rec); rec.Code != http.StatusOK || p["what"] != "pear" {
		t.Errorf("patched what = %v (%d), want it trimmed", p["what"], rec.Code)
	}

	for name, what := range map[string]any{"empty": "", "spaces": "   ", "whitespace": " \t\n", "none": nil} {
		wantInvalid2(t, name+" (create)", k.do(http.MethodPost, base, ana, mealBody(map[string]any{"what": what})), "what")
	}
	for name, what := range map[string]any{"empty": "", "spaces": "   "} {
		wantInvalid2(t, name+" (patch)", k.do(http.MethodPatch, one, ana, map[string]any{"what": what}), "what")
	}
	if got := bodyMap(t, k.do(http.MethodGet, one, ana, nil)); got["what"] != "pear" {
		t.Errorf("a refused patch changed what to %q", got["what"])
	}

	// The limit counts characters of the trimmed text.
	limit := strings.Repeat("ñ", constants.MaxMealTextLen)
	if rec := k.do(http.MethodPost, base, ana, mealBody(map[string]any{"what": "  " + limit + "  "})); rec.Code != http.StatusCreated {
		t.Errorf("what at the limit, padded: %d %q", rec.Code, rec.Body.String())
	}
	wantInvalid2(t, "one over", k.do(http.MethodPost, base, ana, mealBody(map[string]any{"what": limit + "ñ"})), "what")
}

func TestMealValidation(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	base := "/v1/kids/" + kid + "/meals"
	id := k.createMeal(ana, kid, nil)
	stored := k.do(http.MethodGet, base+"/"+id, ana, nil).Body.String()

	for _, c := range []struct {
		name  string
		over  map[string]any
		field string
	}{
		{"more than 5 minutes ahead", map[string]any{"at": ts(t0.Add(constants.MaxFutureSkew + time.Second))}, "at"},
		{"before the birth date", map[string]any{"at": "2026-03-03T22:59:59Z"}, "at"},
		{"reaction over the limit", map[string]any{"reaction": strings.Repeat("ñ", constants.MaxMealTextLen+1)}, "reaction"},
		{"note over the limit", map[string]any{"note": strings.Repeat("ñ", constants.MaxNoteLen+1)}, "note"},
	} {
		wantInvalid2(t, c.name+" (create)", k.do(http.MethodPost, base, ana, mealBody(c.over)), c.field)
		wantInvalid2(t, c.name+" (patch)", k.do(http.MethodPatch, base+"/"+id, ana, c.over), c.field)
	}
	if got := k.do(http.MethodGet, base+"/"+id, ana, nil).Body.String(); got != stored {
		t.Errorf("a refused patch changed the entry:\n got %s\nwant %s", got, stored)
	}
	if n := len(bodyMap(t, k.do(http.MethodGet, base, ana, nil))["meals"].([]any)); n != 1 {
		t.Errorf("%d meals after the refused creates, want 1", n)
	}

	for _, c := range []struct {
		name string
		over map[string]any
	}{
		{"exactly 5 minutes ahead", map[string]any{"at": ts(t0.Add(constants.MaxFutureSkew))}},
		{"the start of the birth date in the kid's zone", map[string]any{"at": "2026-03-03T23:00:00Z"}},
		{"a reaction at the limit", map[string]any{"reaction": strings.Repeat("ñ", constants.MaxMealTextLen)}},
		{"no reaction", map[string]any{"reaction": nil}},
		{"a note at the limit", map[string]any{"note": strings.Repeat("ñ", constants.MaxNoteLen)}},
	} {
		if rec := k.do(http.MethodPost, base, ana, mealBody(c.over)); rec.Code != http.StatusCreated {
			t.Errorf("%s: %d %q", c.name, rec.Code, rec.Body.String())
		}
	}

	for _, method := range []string{http.MethodPost, http.MethodPatch} {
		path := base
		if method == http.MethodPatch {
			path += "/" + id
		}
		rec := k.do(method, path, ana, mealBody(map[string]any{"colour": "red"}))
		if rec.Code != http.StatusUnprocessableEntity || strings.TrimSpace(rec.Body.String()) != `{"error":"unknown_field","field":"colour"}` {
			t.Errorf("%s unknown field: %d %q", method, rec.Code, rec.Body.String())
		}
		wantErr(t, k.do(method, path, ana, mealBody(map[string]any{"at": "yesterday"})), http.StatusBadRequest, "bad_request")
	}
}

func TestMealList(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	other := k.createKid(ana, map[string]any{"name": "Second"})
	k.createMeal(ana, other, nil)
	base := "/v1/kids/" + kid + "/meals"

	at := func(m time.Duration) map[string]any { return map[string]any{"at": ts(t0.Add(-m * time.Minute))} }
	a0 := k.createMeal(ana, kid, at(1))
	a1 := k.createMeal(ana, kid, at(2))
	a2 := k.createMeal(ana, kid, at(2))
	a3 := k.createMeal(ana, kid, at(3))
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
		sameKeys(t, m, "meals", "next_cursor")
		var ids []string
		for _, e := range m["meals"].([]any) {
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
