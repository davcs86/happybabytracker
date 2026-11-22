package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/auth"
	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/units"
)

type actor int

const (
	aNoToken   actor = iota // valid g session, no internal token
	aNoSession              // internal token, no session
	aAdmin
	aX // parent without a grant on the kid
	aC // caregiver
	aG // guardian
	actorCount
)

var actorNames = [actorCount]string{"no-token", "no-session", "admin", "x", "c", "g"}

func (a actor) String() string { return actorNames[a] }

// want is 0 for "reaches the handler" or the exact refusal status.
type want int

const (
	wantOK want = 0
	s401   want = http.StatusUnauthorized
	s403   want = http.StatusForbidden
	s404   want = http.StatusNotFound
)

// matrixWant is columns no-token, no-session, admin, x, c, g.
var matrixWant = map[Access][actorCount]want{
	Open:      {wantOK, wantOK, wantOK, wantOK, wantOK, wantOK},
	Internal:  {s401, wantOK, wantOK, wantOK, wantOK, wantOK},
	Session:   {s401, s401, wantOK, wantOK, wantOK, wantOK},
	Admin:     {s401, s401, wantOK, s403, s403, s403},
	Parent:    {s401, s401, s403, wantOK, wantOK, wantOK},
	Caregiver: {s401, s401, s403, s404, wantOK, wantOK},
	Guardian:  {s401, s401, s403, s404, s403, wantOK},
}

// routeClasses pins the intended class of every route independently of routes.go, so a wrongly
// declared class cannot make its own expectations pass.
var routeClasses = map[string]Access{
	"GET /healthz":                                        Open,
	"POST /v1/session":                                    Internal,
	"GET /v1/session":                                     Session,
	"DELETE /v1/session":                                  Session,
	"POST /v1/admin/parents":                              Admin,
	"GET /v1/admin/parents":                               Admin,
	"POST /v1/admin/parents/{parent}/reset-link":          Admin,
	"POST /v1/admin/parents/{parent}/disable":             Admin,
	"POST /v1/admin/parents/{parent}/enable":              Admin,
	"GET /v1/tokens/{token}":                              Internal,
	"POST /v1/tokens/{token}/redeem":                      Internal,
	"PATCH /v1/me":                                        Parent,
	"POST /v1/me/password":                                Parent,
	"GET /v1/me/sessions":                                 Parent,
	"DELETE /v1/me/sessions/{session}":                    Parent,
	"GET /v1/growth-sources":                              Parent,
	"GET /v1/vaccine-schemas":                             Parent,
	"GET /v1/vaccine-catalogue":                           Parent,
	"GET /v1/kids":                                        Parent,
	"POST /v1/kids":                                       Parent,
	"GET /v1/kids/{kid}":                                  Caregiver,
	"PATCH /v1/kids/{kid}":                                Guardian,
	"DELETE /v1/kids/{kid}":                               Guardian,
	"GET /v1/kids/{kid}/access":                           Guardian,
	"PUT /v1/kids/{kid}/access/{username}":                Guardian,
	"DELETE /v1/kids/{kid}/access/{parent}":               Caregiver,
	"GET /v1/kids/{kid}/bottles":                          Caregiver,
	"POST /v1/kids/{kid}/bottles":                         Caregiver,
	"GET /v1/kids/{kid}/bottles/{entry}":                  Caregiver,
	"PATCH /v1/kids/{kid}/bottles/{entry}":                Caregiver,
	"DELETE /v1/kids/{kid}/bottles/{entry}":               Caregiver,
	"GET /v1/kids/{kid}/diapers":                          Caregiver,
	"POST /v1/kids/{kid}/diapers":                         Caregiver,
	"GET /v1/kids/{kid}/diapers/{entry}":                  Caregiver,
	"PATCH /v1/kids/{kid}/diapers/{entry}":                Caregiver,
	"DELETE /v1/kids/{kid}/diapers/{entry}":               Caregiver,
	"GET /v1/kids/{kid}/meals":                            Caregiver,
	"POST /v1/kids/{kid}/meals":                           Caregiver,
	"GET /v1/kids/{kid}/meals/{entry}":                    Caregiver,
	"PATCH /v1/kids/{kid}/meals/{entry}":                  Caregiver,
	"DELETE /v1/kids/{kid}/meals/{entry}":                 Caregiver,
	"GET /v1/kids/{kid}/pumping":                          Caregiver,
	"POST /v1/kids/{kid}/pumping":                         Caregiver,
	"GET /v1/kids/{kid}/pumping/{entry}":                  Caregiver,
	"PATCH /v1/kids/{kid}/pumping/{entry}":                Caregiver,
	"DELETE /v1/kids/{kid}/pumping/{entry}":               Caregiver,
	"GET /v1/kids/{kid}/sleep":                            Caregiver,
	"POST /v1/kids/{kid}/sleep":                           Caregiver,
	"GET /v1/kids/{kid}/sleep/{entry}":                    Caregiver,
	"PATCH /v1/kids/{kid}/sleep/{entry}":                  Caregiver,
	"DELETE /v1/kids/{kid}/sleep/{entry}":                 Caregiver,
	"POST /v1/kids/{kid}/sleep/{entry}/end":               Caregiver,
	"GET /v1/kids/{kid}/nursing":                          Caregiver,
	"POST /v1/kids/{kid}/nursing":                         Caregiver,
	"GET /v1/kids/{kid}/nursing/{entry}":                  Caregiver,
	"PATCH /v1/kids/{kid}/nursing/{entry}":                Caregiver,
	"DELETE /v1/kids/{kid}/nursing/{entry}":               Caregiver,
	"POST /v1/kids/{kid}/nursing/{entry}/end":             Caregiver,
	"POST /v1/kids/{kid}/nursing/{entry}/side":            Caregiver,
	"GET /v1/kids/{kid}/growth":                           Caregiver,
	"POST /v1/kids/{kid}/growth":                          Caregiver,
	"GET /v1/kids/{kid}/growth/curves":                    Caregiver,
	"GET /v1/kids/{kid}/growth/{entry}":                   Caregiver,
	"PATCH /v1/kids/{kid}/growth/{entry}":                 Caregiver,
	"DELETE /v1/kids/{kid}/growth/{entry}":                Caregiver,
	"GET /v1/kids/{kid}/vaccines":                         Caregiver,
	"GET /v1/kids/{kid}/vaccinations":                     Caregiver,
	"POST /v1/kids/{kid}/vaccinations":                    Caregiver,
	"GET /v1/kids/{kid}/vaccinations/{administration}":    Caregiver,
	"PATCH /v1/kids/{kid}/vaccinations/{administration}":  Caregiver,
	"DELETE /v1/kids/{kid}/vaccinations/{administration}": Caregiver,
	"PUT /v1/kids/{kid}/vaccine-dismissals/{line}":        Guardian,
	"DELETE /v1/kids/{kid}/vaccine-dismissals/{line}":     Guardian,
	"POST /v1/kids/{kid}/vaccine-dismissals":              Guardian,
	"GET /v1/kids/{kid}/timeline":                         Caregiver,
	"GET /v1/kids/{kid}/summary":                          Caregiver,
	"GET /v1/kids/{kid}/modules":                          Caregiver,
	"PUT /v1/kids/{kid}/modules/{module}":                 Guardian,
	"GET /v1/kids/{kid}/home":                             Caregiver,
}

var accessNames = map[Access]string{
	Open: "Open", Internal: "Internal", Session: "Session", Admin: "Admin",
	Parent: "Parent", Caregiver: "Caregiver", Guardian: "Guardian",
}

func (w want) matches(got int) bool {
	if w == wantOK {
		return got < 500 && got != 401 && got != 403 && got != 404
	}
	return got == int(w)
}

func (w want) String() string {
	if w == wantOK {
		return "ok (below 500, not 401/403/404)"
	}
	return fmt.Sprint(int(w))
}

// An unknown field makes every JSON handler answer 422 before its own 401/403 credential checks.
const matrixBody = `{"matrix_probe":true}`

type fixture struct {
	kit                                           *kit
	kid                                           string
	bottle, diaper, meal, pumping, sleep, nursing string
	growth, administration                        string
	invite                                        string
	parent                                        [actorCount]model.Parent
	token                                         [actorCount]string
	sid                                           [actorCount]string
}

// matrixParams is the only list of path parameters the matrix can fill.
var matrixParams = map[string]func(f *fixture, rt Route, a actor) string{
	"kid":            func(f *fixture, _ Route, _ actor) string { return f.kid },
	"username":       func(f *fixture, _ Route, _ actor) string { return f.parent[aX].Username },
	"token":          func(f *fixture, _ Route, _ actor) string { return f.invite },
	"session":        func(f *fixture, _ Route, a actor) string { return f.sid[a] },
	"module":         func(*fixture, Route, actor) string { return "bottles" },
	"administration": func(f *fixture, _ Route, _ actor) string { return f.administration },
	"line":           func(*fixture, Route, actor) string { return "hepatitis_b-1" },
	"parent": func(f *fixture, rt Route, a actor) string {
		if rt.Access == Caregiver && (a == aC || a == aG) {
			return f.parent[aC].ID
		}
		return f.parent[aX].ID
	},
	"entry": func(f *fixture, rt Route, _ actor) string {
		switch {
		case strings.Contains(rt.Pattern, "/bottles/"):
			return f.bottle
		case strings.Contains(rt.Pattern, "/diapers/"):
			return f.diaper
		case strings.Contains(rt.Pattern, "/meals/"):
			return f.meal
		case strings.Contains(rt.Pattern, "/pumping/"):
			return f.pumping
		case strings.Contains(rt.Pattern, "/sleep/"):
			return f.sleep
		case strings.Contains(rt.Pattern, "/nursing/"):
			return f.nursing
		case strings.Contains(rt.Pattern, "/growth/{entry}"):
			return f.growth
		}
		f.kit.t.Fatalf("%s: no seeded entry for this pattern; add one to newMatrixFixture", rt.Pattern)
		return ""
	},
}

var paramRE = regexp.MustCompile(`\{([^}]*)\}`)

type matrixCase struct {
	route Route
	actor actor
}

// matrixCases pairs every route with every actor and reports each path parameter the filler lacks.
func matrixCases(routes []Route) (cases []matrixCase, problems []string) {
	for _, rt := range routes {
		known := true
		for _, m := range paramRE.FindAllStringSubmatch(rt.Pattern, -1) {
			if _, found := matrixParams[m[1]]; !found {
				known = false
				problems = append(problems, fmt.Sprintf(
					"%s %s: path parameter {%s} is unknown to the matrix; add it to matrixParams", rt.Method, rt.Pattern, m[1]))
			}
		}
		if !known {
			continue
		}
		for a := actor(0); a < actorCount; a++ {
			cases = append(cases, matrixCase{rt, a})
		}
	}
	return cases, problems
}

func newMatrixFixture(t *testing.T) *fixture {
	t.Helper()
	k := newKit(t)
	ctx := context.Background()
	f := &fixture{kit: k}

	f.token[aAdmin], f.sid[aAdmin] = f.seed("admin", "", constants.AdminSessionTTL)
	for a, name := range map[actor]string{aG: "gwen", aC: "cora", aX: "xavi"} {
		f.parent[a] = k.addParent(name)
		f.token[a], f.sid[a] = f.seed("parent", f.parent[a].ID, constants.ParentSessionTTL)
	}
	f.token[aNoToken], f.sid[aNoToken] = f.token[aG], f.sid[aG]
	f.sid[aNoSession] = f.sid[aG]

	schema := "us"
	kid, err := k.Store.CreateKid(ctx, model.Kid{Name: "Kid K", BirthDate: "2026-03-04", Sex: "female", TimeZone: "UTC", GrowthSource: "who",
		VaccineSchema: &schema}, f.parent[aG].ID, k.now)
	if err != nil {
		t.Fatal(err)
	}
	f.kid = kid.ID
	if err := k.Store.SetAccess(ctx, kid.ID, f.parent[aG].ID, f.parent[aC].Username, model.RoleCaregiver, k.now); err != nil {
		t.Fatal(err)
	}
	bottleVolume, err := units.NewVolume(units.UnitML, 120)
	if err != nil {
		t.Fatal(err)
	}
	bottle, err := k.Store.CreateBottle(ctx, kid.ID, f.parent[aG].ID,
		model.BottleFeed{At: k.now, Contents: []string{model.ContentFormula}, Volume: bottleVolume}, k.now)
	if err != nil {
		t.Fatal(err)
	}
	diaper, err := k.Store.CreateDiaper(ctx, kid.ID, f.parent[aG].ID, model.Diaper{At: k.now, Kind: "wet"}, k.now)
	if err != nil {
		t.Fatal(err)
	}
	meal, err := k.Store.CreateMeal(ctx, kid.ID, f.parent[aG].ID, model.Meal{At: k.now, What: "banana"}, k.now)
	if err != nil {
		t.Fatal(err)
	}
	pumping, err := k.Store.CreatePumping(ctx, kid.ID, f.parent[aG].ID,
		model.PumpingSession{StartedAt: k.now, DurationSeconds: 900, Volume: bottleVolume, Side: "both"}, k.now)
	if err != nil {
		t.Fatal(err)
	}
	sleep, err := k.Store.CreateSleep(ctx, kid.ID, f.parent[aG].ID, model.SleepSession{StartedAt: k.now.Add(-time.Hour), EndedAt: &k.now}, k.now)
	if err != nil {
		t.Fatal(err)
	}
	nursing, err := k.Store.CreateNursing(ctx, kid.ID, f.parent[aG].ID,
		model.NursingSession{StartedAt: k.now.Add(-time.Hour), EndedAt: &k.now, LeftSeconds: 600, RightSeconds: 600}, k.now)
	if err != nil {
		t.Fatal(err)
	}
	weight, err := units.NewWeight(units.UnitKG, 5000)
	if err != nil {
		t.Fatal(err)
	}
	measurement, err := k.Store.CreateGrowth(ctx, kid.ID, f.parent[aG].ID, model.Growth{MeasuredOn: "2026-06-01", Weight: &weight}, k.now)
	if err != nil {
		t.Fatal(err)
	}
	f.bottle, f.diaper, f.meal, f.pumping, f.sleep, f.nursing = bottle.ID, diaper.ID, meal.ID, pumping.ID, sleep.ID, nursing.ID
	dose, err := k.Store.CreateAdministration(ctx, kid.ID, f.parent[aG].ID, model.Administration{GivenOn: "2026-03-04", Antigens: []string{"hepatitis_b"}}, k.now)
	if err != nil {
		t.Fatal(err)
	}
	f.growth, f.administration = measurement.ID, dose.ID

	invitee, err := k.Store.CreateParent(ctx, "newbie", "Newbie", k.now)
	if err != nil {
		t.Fatal(err)
	}
	plain, hash, err := auth.NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if err := k.Store.IssueToken(ctx, model.TokenInvite, invitee.ID, hash, k.now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	f.invite = plain
	return f
}

func (f *fixture) seed(kind, parentID string, ttl time.Duration) (token, id string) {
	f.kit.t.Helper()
	plain, hash, err := auth.NewToken()
	if err != nil {
		f.kit.t.Fatal(err)
	}
	id, err = f.kit.Store.CreateSession(context.Background(), kind, parentID, hash, f.kit.now, f.kit.now.Add(ttl))
	if err != nil {
		f.kit.t.Fatal(err)
	}
	return plain, id
}

// send fills the pattern (kidID overrides {kid} when not empty) and serves it as the actor.
func (f *fixture) send(rt Route, a actor, kidID string) (rec *responseCapture, path string) {
	f.kit.t.Helper()
	path = rt.Pattern
	for _, m := range paramRE.FindAllStringSubmatch(rt.Pattern, -1) {
		v := matrixParams[m[1]](f, rt, a)
		if m[1] == "kid" && kidID != "" {
			v = kidID
		}
		path = strings.ReplaceAll(path, m[0], v)
	}
	var body any
	if rt.Method == http.MethodPost || rt.Method == http.MethodPut || rt.Method == http.MethodPatch {
		body = matrixBody
	}
	r := f.kit.newReq(rt.Method, path, body)
	if body != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	if a == aNoToken {
		r.Header.Del("Authorization")
	}
	if f.token[a] != "" {
		r.Header.Set("X-Session-Token", f.token[a])
	}
	rr := f.kit.serve(r)
	return &responseCapture{status: rr.Code, body: rr.Body.String()}, path
}

type responseCapture struct {
	status int
	body   string
}

// loggedRoute is the route the middleware logged last, i.e. the one the mux actually dispatched to.
func (f *fixture) loggedRoute() string {
	lines := strings.Split(strings.TrimSpace(f.kit.Log.String()), "\n")
	var line struct{ Route string }
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &line); err != nil {
		f.kit.t.Fatalf("last log line %q: %v", lines[len(lines)-1], err)
	}
	return line.Route
}

type matrixResult struct {
	route  Route
	actor  actor
	status int
	class  Access
}

func TestAccessMatrix(t *testing.T) {
	routes := Routes(newKit(t).deps())
	cases, problems := matrixCases(routes)
	for _, p := range problems {
		t.Error(p)
	}

	pinned := map[string]Access{}
	seen := map[string]bool{}
	for _, rt := range routes {
		key := rt.Method + " " + rt.Pattern
		seen[key] = true
		class, found := routeClasses[key]
		if !found {
			t.Errorf("%s: route has no entry in routeClasses; pin its intended access class", key)
			class = rt.Access
		}
		if class != rt.Access {
			t.Errorf("%s: routes.go declares %s, routeClasses pins %s", key, accessNames[rt.Access], accessNames[class])
		}
		if _, found := matrixWant[class]; !found {
			t.Errorf("%s: access class %d has no row in matrixWant", key, class)
		}
		pinned[key] = class
	}
	for key := range routeClasses {
		if !seen[key] {
			t.Errorf("%s: pinned in routeClasses but not in Routes()", key)
		}
	}
	perClass := map[Access]int{}
	for _, class := range pinned {
		perClass[class]++
	}
	for a := Open; a <= Guardian; a++ {
		if perClass[a] == 0 {
			t.Errorf("no route has access class %s; the matrix would not exercise it", accessNames[a])
		}
	}

	var results []matrixResult
	for _, c := range cases {
		rt, a := c.route, c.actor
		t.Run(rt.Method+" "+rt.Pattern+" as "+a.String(), func(t *testing.T) {
			f := newMatrixFixture(t)
			rec, path := f.send(rt, a, "")
			label := fmt.Sprintf("%s %s (%s) as %s", rt.Method, rt.Pattern, path, a)
			if got := f.loggedRoute(); got != rt.Method+" "+rt.Pattern {
				t.Fatalf("%s: dispatched to %q, not the route under test", label, got)
			}
			results = append(results, matrixResult{rt, a, rec.status, pinned[rt.Method+" "+rt.Pattern]})

			class := pinned[rt.Method+" "+rt.Pattern]
			w := matrixWant[class][a]
			if !w.matches(rec.status) {
				t.Errorf("%s: got %d %q, want %s", label, rec.status, rec.body, w)
			}
			if w == s404 && (class == Caregiver || class == Guardian) {
				unknown, _ := f.send(rt, a, model.NewID())
				if unknown.status != http.StatusNotFound || unknown.body != rec.body {
					t.Errorf("%s: no-grant 404 %q differs from unknown-kid response %d %q", label, rec.body, unknown.status, unknown.body)
				}
			}
		})
	}

	if got, wantN := len(cases), len(routes)*int(actorCount); got != wantN || len(results) != wantN {
		t.Errorf("walked %d cases and recorded %d results, want %d (%d routes x %d actors)", got, len(results), wantN, len(routes), actorCount)
	}
	walked := map[string]bool{}
	for _, r := range results {
		walked[r.route.Method+" "+r.route.Pattern] = true
	}
	if len(walked) != len(routes) {
		t.Errorf("walked %d distinct routes, Routes() has %d", len(walked), len(routes))
	}

	for _, r := range results {
		if r.actor != aAdmin {
			continue
		}
		p := r.route.Pattern
		adminMayPass := strings.HasPrefix(p, "/v1/admin/") || p == "/v1/session" || p == "/healthz" ||
			r.class == Internal || r.class == Open
		if (r.class == Parent || !adminMayPass) && r.status != http.StatusForbidden {
			t.Errorf("%s %s: an admin session got %d, want 403 (admin sessions never reach parent data)", r.route.Method, p, r.status)
		}
	}
}

func TestMatrixCasesRejectsUnknownParam(t *testing.T) {
	_, problems := matrixCases([]Route{
		{http.MethodGet, "/v1/kids/{kid}/widgets/{widget}", Caregiver, nil},
		{http.MethodGet, "/v1/kids/{kid}", Caregiver, nil},
	})
	if len(problems) != 1 || !strings.Contains(problems[0], "{widget}") {
		t.Fatalf("problems = %q, want exactly one naming {widget}", problems)
	}
	cases, problems := matrixCases([]Route{{http.MethodGet, "/v1/kids/{kid}", Caregiver, nil}})
	if len(problems) != 0 || len(cases) != int(actorCount) {
		t.Fatalf("known route: %d cases, problems %q", len(cases), problems)
	}
}
