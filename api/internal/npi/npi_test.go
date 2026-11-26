package npi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// fake answers every request with status and body, and records each request it sees.
type fake struct {
	mu     sync.Mutex
	reqs   []*http.Request
	status int
	body   string
	header http.Header
}

func (f *fake) RoundTrip(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reqs = append(f.reqs, r)
	h := f.header
	if h == nil {
		h = http.Header{"Content-Type": {"application/json"}}
	}
	return &http.Response{StatusCode: f.status, Header: h, Body: io.NopCloser(strings.NewReader(f.body)), Request: r}, nil
}

func (f *fake) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.reqs)
}

func okFake(body string) *fake { return &fake{status: http.StatusOK, body: body} }

var clock = time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)

func fixedNow() time.Time { return clock }

func waQuery() Query { return Query{Kind: Person, LastName: "quill", State: "WA"} }

// addr is one NPPES address; purpose is LOCATION or MAILING.
func addr(purpose, line1, city, state, postal string) map[string]any {
	return map[string]any{"address_purpose": purpose, "address_1": line1, "city": city, "state": state,
		"postal_code": postal, "country_code": "US", "telephone_number": "206-555-0100"}
}

// person is an active NPI-1 record with the given addresses.
func person(number, first, last string, addrs ...map[string]any) map[string]any {
	return map[string]any{"number": number, "enumeration_type": "NPI-1",
		"basic":       map[string]any{"first_name": first, "last_name": last, "credential": "MD", "status": "A"},
		"addresses":   addrs,
		"taxonomies":  []any{map[string]any{"code": "208000000X", "desc": "Pediatrics", "primary": true}},
		"identifiers": []any{}}
}

func with(rec map[string]any, key string, v any) map[string]any {
	out := maps.Clone(rec)
	out[key] = v
	return out
}

func withBasic(rec map[string]any, key string, v any) map[string]any {
	b := maps.Clone(rec["basic"].(map[string]any))
	b[key] = v
	return with(rec, "basic", b)
}

func page(t *testing.T, records ...map[string]any) string {
	t.Helper()
	b, err := json.Marshal(map[string]any{"result_count": len(records), "results": records})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func search(t *testing.T, rt http.RoundTripper, q Query) (Page, Outcome, error) {
	t.Helper()
	return New(rt, fixedNow).Search(context.Background(), q)
}

func names(p Page) []string {
	out := []string{}
	for _, r := range p.Results {
		out = append(out, r.Name)
	}
	return out
}

func TestSearchRequest(t *testing.T) {
	f := okFake(page(t))
	if _, _, err := search(t, f, Query{Kind: Person, LastName: "quil", FirstName: "av", State: "WA", City: "Seattle", PostalCode: "98101"}); err != nil {
		t.Fatal(err)
	}
	r := f.reqs[0]
	want := "version=2.1&enumeration_type=NPI-1&last_name=quil*&first_name=av*&state=WA&city=Seattle&postal_code=98101&limit=20"
	if r.Method != http.MethodGet || r.URL.Scheme != "https" || r.URL.Host != "npiregistry.cms.hhs.gov" || r.URL.Path != "/api/" || r.URL.RawQuery != want {
		t.Errorf("person request %s %s, want GET https://npiregistry.cms.hhs.gov/api/?%s", r.Method, r.URL, want)
	}

	f = okFake(page(t))
	if _, _, err := search(t, f, Query{Kind: Organization, Organization: "quill", State: "WA"}); err != nil {
		t.Fatal(err)
	}
	want = "version=2.1&enumeration_type=NPI-2&organization_name=quill*&state=WA&limit=20"
	if got := f.reqs[0].URL.RawQuery; got != want {
		t.Errorf("organization query %s, want %s", got, want)
	}

	f = okFake(page(t))
	if _, _, err := search(t, f, Query{Kind: Organization, Organization: "A&B Kids 2", State: "WA", City: "Port Angeles"}); err != nil {
		t.Fatal(err)
	}
	q := f.reqs[0].URL.Query()
	if q.Get("organization_name") != "A&B Kids 2*" || q.Get("city") != "Port Angeles" || len(q) != 6 {
		t.Errorf("escaped organization query %s decodes to %v", f.reqs[0].URL.RawQuery, q)
	}
}

func TestSearchMapsShape(t *testing.T) {
	body := `{"result_count": 2, "results": [
	  {"number": "9123456789", "enumeration_type": "NPI-1", "created_epoch": "1180000000000",
	   "basic": {"first_name": "AVERY", "last_name": "QUILL", "middle_name": "J", "credential": "MD", "sex": "F", "status": "A"},
	   "addresses": [
	     {"country_code": "US", "address_purpose": "MAILING", "address_type": "DOM", "address_1": "PO BOX 9",
	      "city": "SEATTLE", "state": "WA", "postal_code": "981110009", "telephone_number": "206-555-0199"},
	     {"country_code": "US", "address_purpose": "LOCATION", "address_type": "DOM", "address_1": "12 PINE ST",
	      "address_2": "SUITE 4", "city": "SEATTLE", "state": "WA", "postal_code": "981012756", "telephone_number": "206-555-0100"}],
	   "taxonomies": [
	     {"code": "207Q00000X", "desc": "Family Medicine", "primary": false, "state": "WA", "license": "MD1"},
	     {"code": "208000000X", "desc": "Pediatrics", "primary": true, "state": "WA", "license": "MD2"}],
	   "identifiers": [], "endpoints": [], "other_names": [], "practiceLocations": []},
	  {"number": "9000000015", "enumeration_type": "NPI-2",
	   "basic": {"organization_name": "QUILL & SONS KIDS CLINIC 2", "organizational_subpart": "NO", "status": "A",
	             "authorized_official_first_name": "ROBIN", "authorized_official_last_name": "VALE", "authorized_official_credential": "MD"},
	   "addresses": [{"address_purpose": "LOCATION", "address_1": "1 MAIN ST", "city": "TACOMA", "state": "WA", "postal_code": "98402"}],
	   "taxonomies": []}
	]}`
	p, outcome, err := search(t, okFake(body), waQuery())
	if err != nil || outcome != OK || len(p.Results) != 2 || p.More {
		t.Fatalf("got %+v %s %v", p, outcome, err)
	}
	a := p.Results[0]
	if a.NPI != "9123456789" || a.Name != "AVERY QUILL" || a.Credential == nil || *a.Credential != "MD" || a.Taxonomy == nil ||
		*a.Taxonomy != "Pediatrics" || a.Address != "12 PINE ST, SUITE 4, SEATTLE, WA 98101" || a.Phone == nil || *a.Phone != "206-555-0100" {
		t.Errorf("person result %+v", a)
	}
	o := p.Results[1]
	if o.NPI != "9000000015" || o.Name != "QUILL & SONS KIDS CLINIC 2" || o.Credential != nil || o.Taxonomy != nil ||
		o.Address != "1 MAIN ST, TACOMA, WA 98402" || o.Phone != nil {
		t.Errorf("organization result %+v", o)
	}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"results":[{"npi":"9123456789","name":"AVERY QUILL","credential":"MD","taxonomy":"Pediatrics",` +
		`"address":"12 PINE ST, SUITE 4, SEATTLE, WA 98101","phone":"206-555-0100"},{"npi":"9000000015",` +
		`"name":"QUILL \u0026 SONS KIDS CLINIC 2","credential":null,"taxonomy":null,"address":"1 MAIN ST, TACOMA, WA 98402","phone":null}],"more":false}`
	if string(b) != want {
		t.Errorf("marshalled\n%s\nwant\n%s", b, want)
	}
}

func TestSearchKeepsOnlyPracticeLocationMatches(t *testing.T) {
	a := person("9123456789", "ANA", "QUILL", addr("LOCATION", "1 A ST", "SEATTLE", "WA", "981012756"), addr("MAILING", "PO BOX 1", "PORTLAND", "OR", "97201"))
	b := person("9000000015", "BEA", "QUILL", addr("MAILING", "PO BOX 2", "SEATTLE", "WA", "98101"), addr("LOCATION", "2 B ST", "PORTLAND", "OR", "97201"))
	c := person("1234567893", "CAL", "QUILL", addr("LOCATION", "3 C ST", "MINNEAPOLIS", "MN", "55401"))
	d := person("9000000023", "DEE", "QUILL", addr("LOCATION", "4 D ST", "TACOMA", "WA", "98402"))
	e := person("9000000031", "EVE", "QUILL", addr("LOCATION", "5 E ST", "SEATTLE", "WA", "981040000"))
	for _, n := range []string{"9000000023", "9000000031"} {
		if !Valid(n) {
			t.Fatalf("fixture %s fails the check digit", n)
		}
	}
	body := page(t, a, b, c, d, e)
	for _, tc := range []struct {
		name string
		q    Query
		want []string
	}{
		{"state", Query{Kind: Person, LastName: "quill", State: "WA"}, []string{"ANA QUILL", "DEE QUILL", "EVE QUILL"}},
		{"state in lower case", Query{Kind: Person, LastName: "quill", State: "wa"}, []string{"ANA QUILL", "DEE QUILL", "EVE QUILL"}},
		{"city", Query{Kind: Person, LastName: "quill", State: "WA", City: "Seattle"}, []string{"ANA QUILL", "EVE QUILL"}},
		{"postal code", Query{Kind: Person, LastName: "quill", State: "WA", PostalCode: "98101"}, []string{"ANA QUILL"}},
		{"another state", Query{Kind: Person, LastName: "quill", State: "OR"}, []string{"BEA QUILL"}},
	} {
		p, _, err := search(t, okFake(body), tc.q)
		if err != nil || strings.Join(names(p), ",") != strings.Join(tc.want, ",") {
			t.Errorf("%s: kept %v (err %v), want %v", tc.name, names(p), err, tc.want)
		}
	}
}

func TestAuthorizedOfficialNeverSurfaces(t *testing.T) {
	org := map[string]any{"number": "9000000015", "enumeration_type": "NPI-2",
		"basic": map[string]any{"organization_name": "QUILL CLINIC", "status": "A", "authorized_official_first_name": "Zebedee",
			"authorized_official_last_name": "Zebedee", "authorized_official_telephone_number": "206-555-0177"},
		"addresses": []any{addr("LOCATION", "1 MAIN ST", "SEATTLE", "WA", "98101")}}
	p, _, err := search(t, okFake(page(t, org)), Query{Kind: Organization, Organization: "quill", State: "WA"})
	if err != nil || len(p.Results) != 1 {
		t.Fatalf("got %+v %v", p, err)
	}
	b, _ := json.Marshal(p)
	if s := strings.ToLower(string(b)); strings.Contains(s, "zebedee") || strings.Contains(s, "0177") {
		t.Errorf("an authorized official surfaced: %s", b)
	}
}

func TestSearchDrops(t *testing.T) {
	loc := addr("LOCATION", "1 A ST", "SEATTLE", "WA", "98101")
	keep := person("9123456789", "ANA", "QUILL", loc)
	long := with(person("9000000015", "BEA", "QUILL", loc), "addresses",
		[]any{with(addr("LOCATION", strings.Repeat("A", constants.VisitLocationMaxLen-20), "SEATTLE", "WA", "98101"), "address_2", "SUITE 4000")})
	tooLong := person("9000000023", "CAL", "QUILL", addr("LOCATION", strings.Repeat("A", constants.VisitLocationMaxLen), "SEATTLE", "WA", "98101"))
	body := page(t, keep,
		withBasic(person("9000000031", "DEE", "QUILL", loc), "status", "D"),
		person("9123456788", "EVE", "QUILL", loc),
		person("1234567893", strings.Repeat("F", constants.VisitProviderMaxLen-5), "QUILL", loc),
		long, tooLong)
	p, _, err := search(t, okFake(body), waQuery())
	if err != nil || len(p.Results) != 2 || p.Results[0].Name != "ANA QUILL" || p.Results[1].Name != "BEA QUILL" {
		t.Fatalf("kept %v, %v; want ANA QUILL and BEA QUILL", names(p), err)
	}
	if got := p.Results[1].Address; strings.Contains(got, "SUITE") || got != strings.Repeat("A", constants.VisitLocationMaxLen-20)+", SEATTLE, WA 98101" {
		t.Errorf("a long address kept as %q, want it without address_2", got)
	}
}

func TestMoreWhenUpstreamFull(t *testing.T) {
	in := person("9123456789", "ANA", "QUILL", addr("LOCATION", "1 A ST", "SEATTLE", "WA", "98101"))
	out := person("9123456789", "BEA", "QUILL", addr("LOCATION", "1 A ST", "PORTLAND", "OR", "97201"))
	for _, tc := range []struct {
		upstream int
		more     bool
	}{{constants.NPISearchLimit, true}, {constants.NPISearchLimit - 1, false}} {
		recs := []map[string]any{in, in, in}
		for len(recs) < tc.upstream {
			recs = append(recs, out)
		}
		p, _, err := search(t, okFake(page(t, recs...)), waQuery())
		if err != nil || len(p.Results) != 3 || p.More != tc.more {
			t.Errorf("%d upstream: %d kept, more %v (err %v); want 3, %v", tc.upstream, len(p.Results), p.More, err, tc.more)
		}
	}
}

func TestErrorsBodyIsUpstreamError(t *testing.T) {
	body := `{"Errors":[{"description":"Field state requires additional search criteria","field":"state","number":"07"}]}`
	p, outcome, err := search(t, okFake(body), waQuery())
	if !errors.Is(err, ErrUnavailable) || outcome != UpstreamError || len(p.Results) != 0 {
		t.Fatalf("got %+v %s %v, want ErrUnavailable and upstream_error", p, outcome, err)
	}
	if strings.Contains(err.Error(), "state requires") {
		t.Errorf("the error carries the registry's text: %v", err)
	}
}

func TestUpstreamFailures(t *testing.T) {
	big := `{"results":[` + strings.Repeat(" ", constants.NPIMaxResponseBytes) + `]}`
	for _, tc := range []struct {
		name string
		f    *fake
	}{
		{"status 500", &fake{status: http.StatusInternalServerError, body: page(t)}},
		{"redirect", &fake{status: http.StatusFound, header: http.Header{"Location": {"https://npiregistry.cms.hhs.gov/elsewhere"}}}},
		{"too large", okFake(big)},
		{"unparseable", okFake(`{`)},
		{"a number as a number", okFake(`{"results":[{"number":9123456789}]}`)},
	} {
		p, outcome, err := search(t, tc.f, waQuery())
		if !errors.Is(err, ErrUnavailable) || outcome != UpstreamError || len(p.Results) != 0 {
			t.Errorf("%s: %+v %s %v, want upstream_error", tc.name, p, outcome, err)
		}
		if n := tc.f.count(); n != 1 {
			t.Errorf("%s: %d requests, want 1", tc.name, n)
		}
		if err != nil && strings.Contains(err.Error(), "quill") {
			t.Errorf("%s: the error carries the term: %v", tc.name, err)
		}
	}

	block := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		<-r.Context().Done()
		return nil, r.Context().Err()
	})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, outcome, err := New(block, fixedNow).Search(ctx, waQuery())
	if !errors.Is(err, ErrUnavailable) || outcome != Timeout {
		t.Errorf("a blocked transport: %s %v, want timeout", outcome, err)
	}
	if err != nil && strings.Contains(err.Error(), "quill") {
		t.Errorf("the timeout error carries the term: %v", err)
	}
	refused := roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("dial: refused") })
	if _, outcome, err := search(t, refused, waQuery()); !errors.Is(err, ErrUnavailable) || outcome != UpstreamError {
		t.Errorf("a refused dial: %s %v, want upstream_error", outcome, err)
	}
}

func TestInFlightCap(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	var mu sync.Mutex
	calls := 0
	rt := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		mu.Lock()
		calls++
		mu.Unlock()
		entered <- struct{}{}
		<-release
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"results":[]}`)), Request: r}, nil
	})
	s := New(rt, fixedNow)
	finished := make(chan error, constants.NPIMaxInFlight+1)
	n := 0
	// Each call is a distinct query, so none is answered from the cache.
	start := func() {
		n++
		q := Query{Kind: Person, LastName: "quill", State: "WA", PostalCode: strconv.Itoa(10000 + n)}
		go func() {
			_, _, err := s.Search(context.Background(), q)
			finished <- err
		}()
		<-entered
	}
	for range constants.NPIMaxInFlight {
		start()
	}
	if _, outcome, err := s.Search(context.Background(), Query{Kind: Person, LastName: "other", State: "WA"}); !errors.Is(err, ErrUnavailable) || outcome != UpstreamError {
		t.Errorf("the call over the cap: %s %v, want upstream_error at once", outcome, err)
	}
	mu.Lock()
	if calls != constants.NPIMaxInFlight {
		t.Errorf("transport saw %d calls, want %d", calls, constants.NPIMaxInFlight)
	}
	mu.Unlock()
	release <- struct{}{}
	if err := <-finished; err != nil {
		t.Errorf("a call under the cap: %v", err)
	}
	start()
	for range constants.NPIMaxInFlight {
		release <- struct{}{}
		if err := <-finished; err != nil {
			t.Errorf("a call after one was released: %v", err)
		}
	}
}

func TestEmpty(t *testing.T) {
	p, outcome, err := search(t, okFake(`{"result_count":0,"results":[]}`), waQuery())
	if err != nil || outcome != Empty || p.Results == nil || p.More {
		t.Fatalf("got %+v %s %v", p, outcome, err)
	}
	if b, _ := json.Marshal(p); string(b) != `{"results":[],"more":false}` {
		t.Errorf("marshalled %s", b)
	}
}

func TestSearchCachesNormalisedQuery(t *testing.T) {
	now := clock
	f := okFake(page(t, person("9123456789", "ANA", "QUILL", addr("LOCATION", "1 A ST", "PORT ANGELES", "WA", "98362"))))
	s := New(f, func() time.Time { return now })
	first, outcome, err := s.Search(context.Background(), Query{Kind: Person, LastName: " Quill ", State: "wa", City: "Port  Angeles"})
	if err != nil || outcome != OK {
		t.Fatalf("first: %s %v", outcome, err)
	}
	second, outcome, err := s.Search(context.Background(), Query{Kind: Person, LastName: "quill", State: "WA", City: "port angeles"})
	if err != nil || outcome != CacheHit || f.count() != 1 {
		t.Fatalf("second: %s %v, %d requests; want cache_hit after one", outcome, err, f.count())
	}
	if len(first.Results) != len(second.Results) || first.More != second.More {
		t.Errorf("cached page %+v differs from %+v", second, first)
	}
	if _, outcome, _ := s.Search(context.Background(), Query{Kind: Person, LastName: "quill", FirstName: "a", State: "WA", City: "port angeles"}); outcome == CacheHit {
		t.Error("another first name hit the cache")
	}
	if _, outcome, _ := s.Search(context.Background(), Query{Kind: Organization, Organization: "quill", State: "WA", City: "port angeles"}); outcome == CacheHit {
		t.Error("an organization search hit a person's cache entry")
	}
	reqs := f.count()
	now = now.Add(constants.NPICacheTTL - time.Second)
	if _, outcome, _ := s.Search(context.Background(), Query{Kind: Person, LastName: "quill", State: "WA", City: "Port Angeles"}); outcome != CacheHit {
		t.Errorf("just before the TTL: %s, want cache_hit", outcome)
	}
	now = clock.Add(constants.NPICacheTTL)
	if _, outcome, _ := s.Search(context.Background(), Query{Kind: Person, LastName: "quill", State: "WA", City: "Port Angeles"}); outcome != OK || f.count() != reqs+1 {
		t.Errorf("at the TTL: %s after %d requests, want a new request", outcome, f.count())
	}

	empty := okFake(page(t))
	s = New(empty, fixedNow)
	for range 2 {
		s.Search(context.Background(), waQuery())
	}
	if _, outcome, _ := s.Search(context.Background(), waQuery()); outcome != CacheHit || empty.count() != 1 {
		t.Errorf("an empty answer: %s after %d requests, want it cached", outcome, empty.count())
	}
}

func TestSearchErrorsAreNotCached(t *testing.T) {
	for name, bad := range map[string]*fake{
		"status 500":  {status: http.StatusInternalServerError},
		"errors body": okFake(`{"Errors":[{"description":"x","field":"state","number":"07"}]}`),
	} {
		var calls int
		rt := roundTripFunc(func(r *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return bad.RoundTrip(r)
			}
			return okFake(page(t)).RoundTrip(r)
		})
		s := New(rt, fixedNow)
		if _, outcome, err := s.Search(context.Background(), waQuery()); err == nil || outcome != UpstreamError {
			t.Fatalf("%s: first: %s %v", name, outcome, err)
		}
		if _, outcome, err := s.Search(context.Background(), waQuery()); err != nil || outcome != Empty || calls != 2 {
			t.Errorf("%s: second: %s %v after %d calls, want a fresh empty answer", name, outcome, err, calls)
		}
	}
}

func TestCacheBound(t *testing.T) {
	f := okFake(page(t))
	s := New(f, fixedNow)
	q := func(i int) Query {
		return Query{Kind: Person, LastName: "quill", State: "WA", PostalCode: strconv.Itoa(10000 + i)}
	}
	for i := range constants.NPICacheEntries + 1 {
		if _, _, err := s.Search(context.Background(), q(i)); err != nil {
			t.Fatal(err)
		}
	}
	if n := s.cacheLen(); n != constants.NPICacheEntries {
		t.Errorf("%d cache entries, want %d", n, constants.NPICacheEntries)
	}
	reqs := f.count()
	if _, outcome, _ := s.Search(context.Background(), q(1)); outcome != CacheHit {
		t.Errorf("the second query: %s, want still cached", outcome)
	}
	if _, outcome, _ := s.Search(context.Background(), q(0)); outcome == CacheHit || f.count() != reqs+1 {
		t.Errorf("the first query: %s, want evicted", outcome)
	}
}

func TestCacheEvictsExpiredFirst(t *testing.T) {
	now := clock
	s := New(okFake(page(t)), func() time.Time { return now })
	q := func(i int) Query {
		return Query{Kind: Person, LastName: "quill", State: "WA", PostalCode: strconv.Itoa(10000 + i)}
	}
	for i := range constants.NPICacheEntries {
		if i == 2 {
			now = clock.Add(constants.NPICacheTTL / 2)
		}
		s.Search(context.Background(), q(i))
	}
	now = clock.Add(constants.NPICacheTTL)
	s.Search(context.Background(), q(constants.NPICacheEntries))
	if n := s.cacheLen(); n != constants.NPICacheEntries-1 {
		t.Errorf("%d entries after a full cache with two expired took one more, want %d", n, constants.NPICacheEntries-1)
	}
	if _, outcome, _ := s.Search(context.Background(), q(2)); outcome != CacheHit {
		t.Errorf("the oldest live entry: %s, want kept", outcome)
	}
}

func (s *Searcher) cacheLen() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.cache)
}
