package httpapi

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/davcs86/happybabytracker/api/internal/auth"
	"github.com/davcs86/happybabytracker/api/internal/constants"
)

const searchPath = "/v1/providers/search"

// oneProvider is an NPPES page with one active NPI-1 record practising in Port Angeles, WA (fictional).
const oneProvider = `{"result_count":1,"results":[{"number":"9123456789","enumeration_type":"NPI-1",
	"basic":{"first_name":"AVERY","last_name":"QUILLFEATHER","credential":"MD","status":"A"},
	"addresses":[{"address_purpose":"LOCATION","address_1":"12 PINE ST","city":"PORT ANGELES","state":"WA",
	"postal_code":"983620000","telephone_number":"360-555-0100"}],
	"taxonomies":[{"desc":"Pediatrics","primary":true}]}]}`

func personSearch(over map[string]any) map[string]any {
	return withOverrides(map[string]any{"kind": "person", "last_name": "quillfeather", "state": "WA"}, over)
}

func TestProviderState(t *testing.T) {
	k := newKit(t)
	ana := k.asParent("ana")
	rec := k.do(http.MethodGet, "/v1/providers", ana, nil)
	if got := strings.TrimSpace(rec.Body.String()); rec.Code != http.StatusOK || got != `{"search":{"enabled":true}}` {
		t.Errorf("flag on: %d %s", rec.Code, got)
	}
	k.NPI = nil
	k.Handler = NewHandler(k.deps())
	rec = k.do(http.MethodGet, "/v1/providers", ana, nil)
	if got := strings.TrimSpace(rec.Body.String()); rec.Code != http.StatusOK || got != `{"search":{"enabled":false}}` {
		t.Errorf("flag off: %d %s", rec.Code, got)
	}
}

func TestProviderSearchAnswers(t *testing.T) {
	k := newKit(t)
	ana := k.asParent("ana")
	k.Registry.set(http.StatusOK, oneProvider)
	rec := k.do(http.MethodPost, searchPath, ana, personSearch(map[string]any{"first_name": "Avery", "city": "Port Angeles", "postal_code": "98362"}))
	want := `{"results":[{"npi":"9123456789","name":"AVERY QUILLFEATHER","credential":"MD","taxonomy":"Pediatrics",` +
		`"address":"12 PINE ST, PORT ANGELES, WA 98362","phone":"360-555-0100"}],"more":false}`
	if got := strings.TrimSpace(rec.Body.String()); rec.Code != http.StatusOK || got != want {
		t.Errorf("search: %d %s, want %s", rec.Code, got, want)
	}
	k.Registry.set(http.StatusOK, `{"result_count":0,"results":[]}`)
	rec = k.do(http.MethodPost, searchPath, ana, map[string]any{"kind": "organization", "organization": "A&B Kids 2", "state": "wa"})
	if got := strings.TrimSpace(rec.Body.String()); rec.Code != http.StatusOK || got != `{"results":[],"more":false}` {
		t.Errorf("empty organization search: %d %s", rec.Code, got)
	}
	k.Registry.set(http.StatusInternalServerError, "")
	wantErr(t, k.do(http.MethodPost, searchPath, ana, personSearch(map[string]any{"last_name": "other"})), http.StatusServiceUnavailable, "unavailable")
	k.Registry.set(http.StatusOK, `{"Errors":[{"description":"Field state requires additional search criteria","field":"state","number":"07"}]}`)
	wantErr(t, k.do(http.MethodPost, searchPath, ana, personSearch(map[string]any{"last_name": "another"})), http.StatusServiceUnavailable, "unavailable")
}

func TestProviderSearchValidation(t *testing.T) {
	k := newKit(t)
	ana := k.asParent("ana")
	org := func(over map[string]any) map[string]any {
		return withOverrides(map[string]any{"kind": "organization", "organization": "Quill Clinic", "state": "WA"}, over)
	}
	for _, c := range []struct {
		field string
		body  map[string]any
	}{
		{"kind", personSearch(map[string]any{"kind": nil})},
		{"kind", personSearch(map[string]any{"kind": "clinic", "last_name": "J"})},
		{"last_name", personSearch(map[string]any{"last_name": nil})},
		{"last_name", personSearch(map[string]any{"last_name": "quil*"})},
		{"last_name", personSearch(map[string]any{"last_name": "J", "state": "XX"})},
		{"last_name", personSearch(map[string]any{"last_name": "ab$"})},
		{"last_name", personSearch(map[string]any{"last_name": "O'Brien 2"})},
		{"last_name", personSearch(map[string]any{"last_name": strings.Repeat("a", constants.NPINameMaxLen+1)})},
		{"last_name", personSearch(map[string]any{"last_name": " J "})},
		{"last_name", org(map[string]any{"last_name": "Quill"})},
		{"first_name", personSearch(map[string]any{"first_name": "a"})},
		{"first_name", org(map[string]any{"first_name": "Avery"})},
		{"organization", personSearch(map[string]any{"organization": "Quill Clinic"})},
		{"organization", org(map[string]any{"organization": nil})},
		{"organization", org(map[string]any{"organization": "Quill*"})},
		{"organization", org(map[string]any{"organization": "&B Kids"})},
		{"organization", org(map[string]any{"organization": "Quill $"})},
		{"state", personSearch(map[string]any{"state": "XX"})},
		{"state", personSearch(map[string]any{"state": nil})},
		{"state", personSearch(map[string]any{"state": "Washington", "city": "S"})},
		{"city", personSearch(map[string]any{"city": "S"})},
		{"city", personSearch(map[string]any{"city": "Seattle 1"})},
		{"postal_code", personSearch(map[string]any{"postal_code": "9810"})},
		{"postal_code", personSearch(map[string]any{"postal_code": "98101-1234"})},
		{"postal_code", personSearch(map[string]any{"postal_code": "9810a"})},
	} {
		k.Log.Reset()
		rec := k.do(http.MethodPost, searchPath, ana, c.body)
		if b := bodyMap(t, rec); rec.Code != http.StatusUnprocessableEntity || b["error"] != "invalid" || b["field"] != c.field {
			t.Errorf("%v: %d %v, want field %s", c.body, rec.Code, b, c.field)
		}
		if strings.Contains(k.Log.String(), constants.LogFieldNPIOutcome) {
			t.Errorf("%v: a 422 logged %s", c.body, constants.LogFieldNPIOutcome)
		}
	}
	wantUnknownField(t, k.do(http.MethodPost, searchPath, ana, personSearch(map[string]any{"zip": "98101"})), "zip")
	for _, body := range []map[string]any{
		personSearch(map[string]any{"first_name": "Mary-Jo", "city": "Coeur d'Alene", "state": "id", "postal_code": "83814"}),
		personSearch(map[string]any{"last_name": " St. James ", "state": " PR "}),
		org(map[string]any{"organization": "3 Rivers & Co. Pediatrics", "state": "VI"}),
	} {
		if rec := k.do(http.MethodPost, searchPath, ana, body); rec.Code != http.StatusOK {
			t.Errorf("%v: %d %s, want 200", body, rec.Code, rec.Body.String())
		}
	}
	if k.Registry.hits != 3 {
		t.Errorf("the registry saw %d requests, want only the 3 valid ones", k.Registry.hits)
	}
}

func TestProviderSearchRateLimit(t *testing.T) {
	k := newKit(t)
	ana, bo := k.asParent("ana"), k.asParent("bo")
	for i := range constants.NPISearchesPerMinute {
		if rec := k.do(http.MethodPost, searchPath, ana, personSearch(nil)); rec.Code != http.StatusOK {
			t.Fatalf("search %d: %d %s", i+1, rec.Code, rec.Body.String())
		}
	}
	rec := k.do(http.MethodPost, searchPath, ana, personSearch(nil))
	if rec.Header().Get("Retry-After") == "" {
		t.Error("429 without Retry-After")
	}
	wantErr(t, rec, http.StatusTooManyRequests, "rate_limited")
	if rec := k.do(http.MethodPost, searchPath, bo, personSearch(nil)); rec.Code != http.StatusOK {
		t.Errorf("another parent: %d %s, want 200", rec.Code, rec.Body.String())
	}
	if k.Registry.hits != 1 {
		t.Errorf("the registry saw %d requests, want 1 (the rest are cache hits)", k.Registry.hits)
	}
}

// npiLines are the search route's log lines, as their two npi fields.
func npiLines(t *testing.T, log string) (outcomes []string, results []float64) {
	t.Helper()
	for _, line := range strings.Split(strings.TrimSpace(log), "\n") {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("log line %q: %v", line, err)
		}
		if m["route"] != "POST "+searchPath {
			continue
		}
		o, _ := m[constants.LogFieldNPIOutcome].(string)
		n, _ := m[constants.LogFieldNPIResults].(float64)
		outcomes, results = append(outcomes, o), append(results, n)
	}
	return outcomes, results
}

func TestProviderSearchLogsNoTerms(t *testing.T) {
	k := newKit(t)
	ana, bo := k.asParent("ana"), k.asParent("bo")
	k.NPILimiter = auth.NewLimiter(3, k.clock)
	k.Handler = NewHandler(k.deps())
	k.Log.Reset()
	terms := map[string]any{"first_name": "Avery", "city": "Port Angeles", "postal_code": "98362"}
	k.Registry.set(http.StatusOK, oneProvider)
	k.do(http.MethodPost, searchPath, ana, personSearch(terms))
	k.Registry.set(http.StatusOK, `{"result_count":0,"results":[]}`)
	k.do(http.MethodPost, searchPath, ana, personSearch(map[string]any{"last_name": "Zanzibarr"}))
	k.do(http.MethodPost, searchPath, ana, personSearch(terms))
	k.do(http.MethodPost, searchPath, ana, personSearch(terms))
	k.Registry.set(http.StatusInternalServerError, "")
	k.do(http.MethodPost, searchPath, bo, personSearch(map[string]any{"last_name": "Marchbanks"}))

	outcomes, results := npiLines(t, k.Log.String())
	if want := []string{"ok", "empty", "cache_hit", "rate_limited", "upstream_error"}; !slices.Equal(outcomes, want) {
		t.Errorf("npi outcomes %v, want %v", outcomes, want)
	}
	if want := []float64{1, 0, 1, 0, 0}; !slices.Equal(results, want) {
		t.Errorf("npi results %v, want %v", results, want)
	}
	banned := []string{"quillfeather", "QUILLFEATHER", "Avery", "AVERY", "Port Angeles", "PORT ANGELES", "98362", "Zanzibarr",
		"Marchbanks", "12 PINE", "360-555-0100", "9123456789", "Pediatrics"}
	wantCleanLog(t, k.Log.String(), 5, banned...)
	for _, b := range banned {
		if strings.Contains(strings.ToLower(k.Log.String()), strings.ToLower(b)) {
			t.Errorf("the log carries %q", b)
		}
	}
}

func TestProviderSearchRouteFollowsFlag(t *testing.T) {
	k := newKit(t)
	on := Routes(k.deps())
	k.NPI, k.NPILimiter = nil, nil
	off := Routes(k.deps())
	keysOf := func(rs []Route) []string {
		var out []string
		for _, r := range rs {
			out = append(out, r.Method+" "+r.Pattern)
		}
		return out
	}
	onKeys, offKeys := keysOf(on), keysOf(off)
	if slices.Contains(offKeys, "POST "+searchPath) || !slices.Contains(offKeys, "GET /v1/providers") {
		t.Fatalf("flag off: routes %v", offKeys)
	}
	if want := slices.DeleteFunc(slices.Clone(onKeys), func(key string) bool { return key == "POST "+searchPath }); !slices.Equal(offKeys, want) {
		t.Errorf("flag off routes differ from flag on by more than the search route:\n%v\n%v", offKeys, want)
	}
	for _, r := range off {
		if class, ok := routeClasses[r.Method+" "+r.Pattern]; !ok || class != r.Access {
			t.Errorf("flag off: %s %s declares %s, routeClasses pins %s", r.Method, r.Pattern, accessNames[r.Access], accessNames[class])
		}
	}

	k.Handler = NewHandler(k.deps())
	ana := k.asParent("ana")
	search := k.do(http.MethodPost, searchPath, ana, personSearch(nil))
	unknown := k.do(http.MethodPost, "/v1/no-such-route", ana, personSearch(nil))
	if search.Code != http.StatusNotFound || search.Body.String() != unknown.Body.String() || unknown.Code != http.StatusNotFound {
		t.Errorf("flag off search: %d %q, unknown route: %d %q", search.Code, search.Body.String(), unknown.Code, unknown.Body.String())
	}
	if rec := k.do(http.MethodGet, "/v1/providers", ana, nil); strings.TrimSpace(rec.Body.String()) != `{"search":{"enabled":false}}` {
		t.Errorf("flag off state: %d %s", rec.Code, rec.Body.String())
	}
	if k.Registry.hits != 0 {
		t.Errorf("the registry saw %d requests with the flag off", k.Registry.hits)
	}
}
