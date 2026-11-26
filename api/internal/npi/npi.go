package npi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

// registryURL is fixed in code so no configuration can point the search elsewhere (visits spec 10).
const registryURL = "https://npiregistry.cms.hhs.gov/api/"

type Kind string

const (
	Person       Kind = "person"
	Organization Kind = "organization"
)

// Query is one search, already validated by the caller (plan 10 B4).
type Query struct {
	Kind                                                       Kind
	LastName, FirstName, Organization, State, City, PostalCode string
}

// Result is one registry record as a visit's provider, place and NPI would take it.
type Result struct {
	NPI        string  `json:"npi"`
	Name       string  `json:"name"`
	Credential *string `json:"credential"`
	Taxonomy   *string `json:"taxonomy"`
	Address    string  `json:"address"`
	Phone      *string `json:"phone"`
}

// Page is what survives the location filter; More is set when the registry returned a full page.
type Page struct {
	Results []Result `json:"results"`
	More    bool     `json:"more"`
}

// Outcome is the enumerated result of one search, as the request log carries it.
type Outcome string

const (
	OK            Outcome = "ok"
	Empty         Outcome = "empty"
	CacheHit      Outcome = "cache_hit"
	RateLimited   Outcome = "rate_limited"
	Timeout       Outcome = "timeout"
	UpstreamError Outcome = "upstream_error"
)

// ErrUnavailable is every failed search; the wrapped reason is fixed text, never a term or the registry's words.
var ErrUnavailable = errors.New("npi: unavailable")

// failure is an error that is ErrUnavailable without carrying the transport's text, which holds the URL.
type failure struct{ reason string }

func (f failure) Error() string        { return ErrUnavailable.Error() + ": " + f.reason }
func (f failure) Is(target error) bool { return target == ErrUnavailable }

// Searcher calls the registry with at most NPIMaxInFlight calls out at once.
type Searcher struct {
	client *http.Client
	now    func() time.Time
	slots  chan struct{}
}

// New builds a Searcher over rt; nil means a clone of http.DefaultTransport.
func New(rt http.RoundTripper, now func() time.Time) *Searcher {
	if rt == nil {
		rt = http.DefaultTransport.(*http.Transport).Clone()
	}
	return &Searcher{
		client: &http.Client{Transport: rt, Timeout: constants.NPITimeout,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
		now:   now,
		slots: make(chan struct{}, constants.NPIMaxInFlight),
	}
}

// Search asks the registry and keeps the records practising where q says; a fifth concurrent call fails at once.
func (s *Searcher) Search(ctx context.Context, q Query) (Page, Outcome, error) {
	select {
	case s.slots <- struct{}{}:
		defer func() { <-s.slots }()
	default:
		return Page{}, UpstreamError, failure{"in flight"}
	}
	recs, full, err := s.fetch(ctx, q)
	if err != nil {
		if isTimeout(err) {
			return Page{}, Timeout, failure{"timeout"}
		}
		var f failure
		if errors.As(err, &f) {
			return Page{}, UpstreamError, f
		}
		return Page{}, UpstreamError, failure{"transport"}
	}
	p := Page{Results: []Result{}, More: full}
	for _, rec := range recs {
		if r, ok := rec.result(q); ok {
			p.Results = append(p.Results, r)
		}
	}
	if len(p.Results) == 0 {
		return p, Empty, nil
	}
	return p, OK, nil
}

func isTimeout(err error) bool {
	var ne net.Error
	return errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &ne) && ne.Timeout())
}

// upstreamQuery is the registry's query string; the trailing * is the registry's wildcard, sent unescaped.
func upstreamQuery(q Query) string {
	var b strings.Builder
	add := func(key, value, suffix string) {
		if value = strings.TrimSpace(value); value != "" {
			b.WriteString("&" + key + "=" + url.QueryEscape(value) + suffix)
		}
	}
	b.WriteString("version=2.1")
	if q.Kind == Person {
		b.WriteString("&enumeration_type=NPI-1")
		add("last_name", q.LastName, "*")
		add("first_name", q.FirstName, "*")
	} else {
		b.WriteString("&enumeration_type=NPI-2")
		add("organization_name", q.Organization, "*")
	}
	add("state", strings.ToUpper(q.State), "")
	add("city", q.City, "")
	add("postal_code", q.PostalCode, "")
	b.WriteString("&limit=" + strconv.Itoa(constants.NPISearchLimit))
	return b.String()
}

// response lists only the fields a Result uses, so nothing else a record holds can reach one.
type response struct {
	Errors  []json.RawMessage `json:"Errors"`
	Results []record          `json:"results"`
}

type record struct {
	Number          string `json:"number"`
	EnumerationType string `json:"enumeration_type"`
	Basic           struct {
		FirstName        string `json:"first_name"`
		LastName         string `json:"last_name"`
		OrganizationName string `json:"organization_name"`
		Credential       string `json:"credential"`
		Status           string `json:"status"`
	} `json:"basic"`
	Addresses  []address `json:"addresses"`
	Taxonomies []struct {
		Desc    string `json:"desc"`
		Primary bool   `json:"primary"`
	} `json:"taxonomies"`
}

type address struct {
	Purpose    string `json:"address_purpose"`
	Line1      string `json:"address_1"`
	Line2      string `json:"address_2"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Phone      string `json:"telephone_number"`
}

func (s *Searcher) fetch(ctx context.Context, q Query) ([]record, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, registryURL+"?"+upstreamQuery(q), nil)
	if err != nil {
		return nil, false, failure{"request"}
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, false, failure{"status"}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, constants.NPIMaxResponseBytes+1))
	if err != nil {
		return nil, false, err
	}
	if len(body) > constants.NPIMaxResponseBytes {
		return nil, false, failure{"too large"}
	}
	var out response
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, false, failure{"unparseable"}
	}
	if len(out.Errors) > 0 {
		return nil, false, failure{"errors body"}
	}
	return out.Results, len(out.Results) >= constants.NPISearchLimit, nil
}

// result maps rec when it is active, valid, fits a visit's fields and practises where q says (owner ruling 1).
func (rec record) result(q Query) (Result, bool) {
	if rec.Basic.Status != "A" || !Valid(rec.Number) {
		return Result{}, false
	}
	loc, ok := rec.location(q)
	if !ok {
		return Result{}, false
	}
	r := Result{NPI: rec.Number, Phone: optional(loc.Phone)}
	switch rec.EnumerationType {
	case "NPI-1":
		r.Name = strings.TrimSpace(rec.Basic.FirstName + " " + rec.Basic.LastName)
		r.Credential = optional(rec.Basic.Credential)
	case "NPI-2":
		r.Name = strings.TrimSpace(rec.Basic.OrganizationName)
	default:
		return Result{}, false
	}
	if r.Name == "" || utf8.RuneCountInString(r.Name) > constants.VisitProviderMaxLen {
		return Result{}, false
	}
	for _, t := range rec.Taxonomies {
		if t.Primary {
			r.Taxonomy = optional(t.Desc)
			break
		}
	}
	r.Address = loc.format(true)
	if utf8.RuneCountInString(r.Address) > constants.VisitLocationMaxLen {
		r.Address = loc.format(false)
	}
	if utf8.RuneCountInString(r.Address) > constants.VisitLocationMaxLen {
		return Result{}, false
	}
	return r, true
}

// location is rec's practice-location address when it matches q's state, city and postal code.
func (rec record) location(q Query) (address, bool) {
	for _, a := range rec.Addresses {
		if !strings.EqualFold(a.Purpose, "LOCATION") {
			continue
		}
		ok := strings.EqualFold(a.State, strings.TrimSpace(q.State)) &&
			(q.City == "" || strings.EqualFold(collapse(a.City), collapse(q.City))) &&
			(q.PostalCode == "" || zip5(a.PostalCode) == strings.TrimSpace(q.PostalCode))
		return a, ok
	}
	return address{}, false
}

func (a address) format(withLine2 bool) string {
	parts := []string{strings.TrimSpace(a.Line1)}
	if l2 := strings.TrimSpace(a.Line2); withLine2 && l2 != "" {
		parts = append(parts, l2)
	}
	parts = append(parts, strings.TrimSpace(a.City), strings.TrimSpace(a.State)+" "+zip5(a.PostalCode))
	return strings.Join(parts, ", ")
}

func zip5(s string) string {
	if s = strings.TrimSpace(s); len(s) > constants.NPIPostalCodeLen {
		return s[:constants.NPIPostalCodeLen]
	}
	return s
}

// collapse trims s and folds each run of spaces to one.
func collapse(s string) string { return strings.Join(strings.Fields(s), " ") }

func optional(s string) *string {
	if s = strings.TrimSpace(s); s != "" {
		return &s
	}
	return nil
}
