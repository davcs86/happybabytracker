package httpapi

import (
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/npi"
)

var (
	npiNameRE = regexp.MustCompile(constants.NPINamePattern)
	npiOrgRE  = regexp.MustCompile(constants.NPIOrgPattern)
)

type providerStateView struct {
	Search struct {
		Enabled bool `json:"enabled"`
	} `json:"search"`
}

// providerState says whether search is on, so the UI needs no flag of its own (visits spec 10).
func (d Deps) providerState(w http.ResponseWriter, _ *http.Request) {
	var v providerStateView
	v.Search.Enabled = d.NPI != nil
	respondJSON(w, http.StatusOK, v)
}

// providerSearchInput is the body of POST /v1/providers/search; the terms never travel in a URL.
type providerSearchInput struct {
	Kind         *string `json:"kind"`
	LastName     *string `json:"last_name"`
	FirstName    *string `json:"first_name"`
	Organization *string `json:"organization"`
	State        *string `json:"state"`
	City         *string `json:"city"`
	PostalCode   *string `json:"postal_code"`
}

// searchText is v trimmed: absent is fine unless required, sent is refused unless allowed, and it must match re.
func searchText(v *string, allowed, required bool, re *regexp.Regexp) (string, bool) {
	if v == nil {
		return "", !required
	}
	t := strings.TrimSpace(*v)
	return t, allowed && re.MatchString(t)
}

// query validates in (plan 10 B4) and names the first invalid field in body order; "" when valid.
func (in providerSearchInput) query() (npi.Query, string) {
	var q npi.Query
	if in.Kind == nil || (*in.Kind != string(npi.Person) && *in.Kind != string(npi.Organization)) {
		return q, "kind"
	}
	q.Kind = npi.Kind(*in.Kind)
	person := q.Kind == npi.Person
	var ok bool
	if q.LastName, ok = searchText(in.LastName, person, person, npiNameRE); !ok {
		return q, "last_name"
	}
	if q.FirstName, ok = searchText(in.FirstName, person, false, npiNameRE); !ok {
		return q, "first_name"
	}
	if q.Organization, ok = searchText(in.Organization, !person, !person, npiOrgRE); !ok {
		return q, "organization"
	}
	if in.State == nil {
		return q, "state"
	}
	if q.State = strings.ToUpper(strings.TrimSpace(*in.State)); !slices.Contains(constants.NPIStates, q.State) {
		return q, "state"
	}
	if q.City, ok = searchText(in.City, true, false, npiNameRE); !ok {
		return q, "city"
	}
	if in.PostalCode != nil {
		if q.PostalCode = *in.PostalCode; len(q.PostalCode) != constants.NPIPostalCodeLen || strings.Trim(q.PostalCode, "0123456789") != "" {
			return q, "postal_code"
		}
	}
	return q, ""
}

// noteNPI hands a search's outcome and result count to the request's one log line (plan 10 B8).
func noteNPI(w http.ResponseWriter, outcome npi.Outcome, results int) {
	if rec := recorder(w); rec != nil {
		rec.npiOutcome, rec.npiResults = string(outcome), results
	}
}

// searchProviders checks the body, then the per-parent limit (cache hits count), then searches.
func (d Deps) searchProviders(w http.ResponseWriter, r *http.Request) {
	var in providerSearchInput
	if !decodeJSON(w, r, &in) {
		return
	}
	q, field := in.query()
	if field != "" {
		respondInvalid(w, field)
		return
	}
	if !d.NPILimiter.Allow("npi:" + PrincipalFrom(r.Context()).ParentID) {
		noteNPI(w, npi.RateLimited, 0)
		respondRateLimited(w)
		return
	}
	page, outcome, err := d.NPI.Search(r.Context(), q)
	noteNPI(w, outcome, len(page.Results))
	if err != nil {
		respondError(w, http.StatusServiceUnavailable, "unavailable")
		return
	}
	respondJSON(w, http.StatusOK, page)
}
