package httpapi

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
)

// vaccineNow is 2025-03-15 at noon UTC: a kid born 2025-01-10 is two months and five days old.
var vaccineNow = time.Date(2025, 3, 15, 12, 0, 0, 0, time.UTC)

// vaccineKid creates a kid born 2025-01-10 in UTC on schema (nil drops the key) with ana as guardian and cara as
// caregiver.
func (k *kit) vaccineKid(schema any) (kid, ana, cara string) {
	k.t.Helper()
	k.now = vaccineNow
	ana, cara = k.asParent("ana"), k.asParent("cara")
	kid = k.createKid(ana, map[string]any{"birth_date": "2025-01-10", "time_zone": "UTC", "vaccine_schema": schema})
	k.grant(ana, kid, "cara", "caregiver")
	return kid, ana, cara
}

func (k *kit) recordDose(token, kid string, body map[string]any) string {
	k.t.Helper()
	rec := k.do(http.MethodPost, "/v1/kids/"+kid+"/vaccinations", token, body)
	if rec.Code != http.StatusCreated {
		k.t.Fatalf("record: %d %q", rec.Code, rec.Body.String())
	}
	return bodyMap(k.t, rec)["id"].(string)
}

func (k *kit) vaccines(token, kid string) map[string]any {
	k.t.Helper()
	rec := k.do(http.MethodGet, "/v1/kids/"+kid+"/vaccines", token, nil)
	if rec.Code != http.StatusOK {
		k.t.Fatalf("vaccines: %d %q", rec.Code, rec.Body.String())
	}
	return bodyMap(k.t, rec)
}

// allLines is every line of groups and also, by id.
func allLines(v map[string]any) map[string]map[string]any {
	out := map[string]map[string]any{}
	for _, g := range v["groups"].([]any) {
		for _, l := range g.(map[string]any)["lines"].([]any) {
			out[l.(map[string]any)["id"].(string)] = l.(map[string]any)
		}
	}
	for _, l := range v["also"].([]any) {
		out[l.(map[string]any)["id"].(string)] = l.(map[string]any)
	}
	return out
}

func groupFrom(t *testing.T, v map[string]any, fromDate string) map[string]any {
	t.Helper()
	for _, g := range v["groups"].([]any) {
		if g := g.(map[string]any); g["from_date"] == fromDate {
			return g
		}
	}
	t.Fatalf("no group from %s in %v", fromDate, v["groups"])
	return nil
}

func lineIDs(g map[string]any) []string {
	var ids []string
	for _, l := range g["lines"].([]any) {
		ids = append(ids, l.(map[string]any)["id"].(string))
	}
	return ids
}

func TestVaccinesShape(t *testing.T) {
	k := newKit(t)
	kid, ana, cara := k.vaccineKid("us")

	rec := k.do(http.MethodGet, "/v1/vaccine-schemas", cara, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("schemas: %d %q", rec.Code, rec.Body.String())
	}
	m := bodyMap(t, rec)
	sameKeys(t, m, "schemas")
	var ids []string
	for _, s := range m["schemas"].([]any) {
		s := s.(map[string]any)
		sameKeys(t, s, "id", "name", "authority", "edition", "retrieved", "origin_url", "licence", "attribution")
		ids = append(ids, s["id"].(string))
	}
	if !slices.Equal(ids, []string{"us", "uk", "mx"}) {
		t.Errorf("schema ids = %v", ids)
	}

	rec = k.do(http.MethodGet, "/v1/vaccine-catalogue", cara, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("catalogue: %d %q", rec.Code, rec.Body.String())
	}
	m = bodyMap(t, rec)
	sameKeys(t, m, "antigens", "products")
	sameKeys(t, m["antigens"].([]any)[0].(map[string]any), "id", "name")
	sameKeys(t, m["products"].([]any)[0].(map[string]any), "id", "name", "antigens")

	v := k.vaccines(cara, kid)
	sameKeys(t, v, "schema", "today", "groups", "also", "recurring", "other")
	sameKeys(t, v["schema"].(map[string]any), "id", "name", "authority", "edition", "retrieved", "origin_url", "licence",
		"attribution", "omitted")
	if v["schema"].(map[string]any)["id"] != "us" || v["today"] != "2025-03-15" {
		t.Errorf("schema %v today %v", v["schema"], v["today"])
	}
	g := groupFrom(t, v, "2025-03-10")
	sameKeys(t, g, "from", "from_date", "status", "lines")
	if from := g["from"].(map[string]any); from["n"] != float64(2) || from["unit"] != "m" || g["status"] != "due" {
		t.Errorf("2-month group = %v", g)
	}
	want := []string{"diphtheria-1", "pertussis-1", "tetanus-1", "hib-1", "polio-1", "pneumococcus-1", "rotavirus-1"}
	if got := lineIDs(g); !slices.Equal(got, want) {
		t.Errorf("2-month lines = %v, want %v", got, want)
	}
	line := g["lines"].([]any)[0].(map[string]any)
	sameKeys(t, line, "id", "group", "antigen", "series", "kind", "status", "from_date", "until_date", "given_on",
		"administration", "dismissal", "note")
	if line["group"] != "DTaP" || line["antigen"] != "diphtheria" || line["series"] != float64(1) || line["kind"] != "routine" ||
		line["status"] != "due" || line["until_date"] != "2025-04-10" || line["given_on"] != nil || line["administration"] != nil ||
		line["dismissal"] != nil || line["note"] == "" {
		t.Errorf("diphtheria-1 = %v", line)
	}
	if s := groupFrom(t, v, "2025-01-10")["status"]; s != "overdue" {
		t.Errorf("birth group status = %v, want overdue (hepatitis B birth dose missed)", s)
	}
	for _, l := range v["also"].([]any) {
		if s := l.(map[string]any)["status"]; s != "available" {
			t.Errorf("also line %v", l)
		}
	}
	recurring := v["recurring"].([]any)
	if len(recurring) != 1 {
		t.Fatalf("recurring = %v", recurring)
	}
	r := recurring[0].(map[string]any)
	sameKeys(t, r, "antigen", "line_id", "doses")
	if r["antigen"] != "influenza" || r["line_id"] != "influenza-1" || len(r["doses"].([]any)) != 0 {
		t.Errorf("recurring = %v", r)
	}
	if len(v["other"].([]any)) != 0 {
		t.Errorf("other = %v", v["other"])
	}

	hepB := k.recordDose(cara, kid, map[string]any{"given_on": "2025-01-10", "antigens": []string{"hepatitis_b"}, "lot": "L1"})
	flu := k.recordDose(cara, kid, map[string]any{"given_on": "2025-03-01", "antigens": []string{"influenza"}})
	typhoid := k.recordDose(ana, kid, map[string]any{"given_on": "2025-03-02", "antigens": []string{"other", "typhoid"},
		"product": "Travel shot"})

	rec = k.do(http.MethodGet, "/v1/kids/"+kid+"/vaccinations/"+typhoid, cara, nil)
	a := bodyMap(t, rec)
	sameKeys(t, a, "id", "given_on", "antigens", "lot", "product", "note", "visit_id")
	if rec.Code != http.StatusOK || a["id"] != typhoid || a["given_on"] != "2025-03-02" || a["lot"] != nil ||
		a["product"] != "Travel shot" || a["note"] != nil || !slices.Equal(stringsOf(a["antigens"]), []string{"other", "typhoid"}) {
		t.Errorf("administration = %d %v", rec.Code, a)
	}
	rec = k.do(http.MethodGet, "/v1/kids/"+kid+"/vaccinations", cara, nil)
	list := bodyMap(t, rec)
	sameKeys(t, list, "vaccinations")
	if rows := list["vaccinations"].([]any); len(rows) != 3 || rows[0].(map[string]any)["id"] != typhoid {
		t.Errorf("list = %v", rows)
	}

	v = k.vaccines(cara, kid)
	birth := allLines(v)["hepatitis_b-1"]
	if birth["status"] != "given" || birth["given_on"] != "2025-01-10" || birth["administration"] != hepB {
		t.Errorf("hepatitis_b-1 = %v", birth)
	}
	if s := groupFrom(t, v, "2025-01-10")["status"]; s != "due" {
		t.Errorf("birth group after the dose = %v, want due (RSV still in its window)", s)
	}
	doses := v["recurring"].([]any)[0].(map[string]any)["doses"].([]any)
	if len(doses) != 1 {
		t.Fatalf("flu doses = %v", doses)
	}
	sameKeys(t, doses[0].(map[string]any), "administration", "given_on")
	if doses[0].(map[string]any)["administration"] != flu {
		t.Errorf("flu dose = %v", doses[0])
	}
	other := v["other"].([]any)
	if len(other) != 2 {
		t.Fatalf("other = %v", other)
	}
	for _, o := range other {
		o := o.(map[string]any)
		sameKeys(t, o, "administration", "antigen", "given_on", "product")
		if o["administration"] != typhoid || o["product"] != "Travel shot" || o["given_on"] != "2025-03-02" {
			t.Errorf("other row = %v", o)
		}
	}

	uk := k.createKid(ana, map[string]any{"birth_date": "2025-01-10", "time_zone": "UTC", "vaccine_schema": "uk"})
	g = groupFrom(t, k.vaccines(ana, uk), "2025-03-07")
	if from := g["from"].(map[string]any); from["n"] != float64(8) || from["unit"] != "w" || g["status"] != "due" {
		t.Errorf("uk 8-week group = %v", g)
	}
	want = []string{"diphtheria-1", "hepatitis_b-1", "hib-1", "pertussis-1", "polio-1", "tetanus-1", "meningococcal_b-1", "rotavirus-1"}
	if got := lineIDs(g); !slices.Equal(got, want) {
		t.Errorf("uk 8-week lines = %v, want %v", got, want)
	}
}

func stringsOf(v any) []string {
	var out []string
	for _, s := range v.([]any) {
		out = append(out, s.(string))
	}
	return out
}

func TestVaccinesRoutesRoles(t *testing.T) {
	k := newKit(t)
	kid, ana, cara := k.vaccineKid("us")
	xavi, admin := k.asParent("xavi"), k.asAdmin()
	base := "/v1/kids/" + kid

	for _, token := range []string{ana, cara} {
		id := k.recordDose(token, kid, map[string]any{"given_on": "2025-02-01", "antigens": []string{"polio"}})
		if rec := k.do(http.MethodPatch, base+"/vaccinations/"+id, token, map[string]any{"note": "ok"}); rec.Code != http.StatusOK {
			t.Errorf("patch: %d %q", rec.Code, rec.Body.String())
		}
		if rec := k.do(http.MethodDelete, base+"/vaccinations/"+id, token, nil); rec.Code != http.StatusNoContent {
			t.Errorf("delete: %d %q", rec.Code, rec.Body.String())
		}
	}
	kept := k.recordDose(ana, kid, map[string]any{"given_on": "2025-02-01", "antigens": []string{"polio"}})

	for _, rq := range []struct {
		method, path string
		body         any
	}{
		{http.MethodPut, base + "/vaccine-dismissals/rsv-1", map[string]any{"reason": "not_needed"}},
		{http.MethodDelete, base + "/vaccine-dismissals/rsv-1", nil},
		{http.MethodPost, base + "/vaccine-dismissals", map[string]any{"before": "2025-03-01", "reason": "not_needed"}},
		{http.MethodPatch, base, map[string]any{"vaccine_schema": "uk"}},
	} {
		wantErr(t, k.do(rq.method, rq.path, cara, rq.body), http.StatusForbidden, "forbidden")
	}
	if len(k.vaccines(ana, kid)["groups"].([]any)) == 0 || bodyMap(t, k.do(http.MethodGet, base, ana, nil))["vaccine_schema"] != "us" {
		t.Error("a caregiver's refused write changed the kid")
	}

	paths := []string{base + "/vaccines", base + "/vaccinations", base + "/vaccinations/" + kept}
	noKid := ""
	for _, p := range paths {
		rec := k.do(http.MethodGet, p, xavi, nil)
		unknown := k.do(http.MethodGet, strings.Replace(p, kid, model.NewID(), 1), ana, nil)
		if rec.Code != http.StatusNotFound || rec.Body.String() != unknown.Body.String() || unknown.Code != http.StatusNotFound {
			t.Errorf("%s: no grant %d %q, unknown kid %d %q", p, rec.Code, rec.Body.String(), unknown.Code, unknown.Body.String())
		}
		noKid = rec.Body.String()
	}
	for _, rq := range []struct{ method, path string }{
		{http.MethodGet, base + "/vaccinations/" + model.NewID()},
		{http.MethodPatch, base + "/vaccinations/" + model.NewID()},
		{http.MethodDelete, base + "/vaccinations/" + model.NewID()},
		{http.MethodGet, base + "/vaccinations/not-an-id"},
	} {
		var body any
		if rq.method == http.MethodPatch {
			body = map[string]any{}
		}
		if rec := k.do(rq.method, rq.path, ana, body); rec.Code != http.StatusNotFound || rec.Body.String() != noKid {
			t.Errorf("%s %s: %d %q, want the no-kid 404 %q", rq.method, rq.path, rec.Code, rec.Body.String(), noKid)
		}
	}

	for _, p := range []string{"/v1/vaccine-schemas", "/v1/vaccine-catalogue", base + "/vaccines", base + "/vaccinations"} {
		wantErr(t, k.do(http.MethodGet, p, admin, nil), http.StatusForbidden, "forbidden")
	}
}

func TestVaccinationValidation(t *testing.T) {
	k, db := fileKit(t)
	kid, ana, _ := k.vaccineKid("us")
	path := "/v1/kids/" + kid + "/vaccinations"
	long := func(n int) string { return strings.Repeat("x", n) }
	ok := map[string]any{"given_on": "2025-02-01", "antigens": []string{"polio"}}
	with := func(over map[string]any) map[string]any {
		b := map[string]any{}
		for key, v := range ok {
			b[key] = v
		}
		for key, v := range over {
			if v == "<drop>" {
				delete(b, key)
				continue
			}
			b[key] = v
		}
		return b
	}
	for name, c := range map[string]struct {
		body  map[string]any
		field string
	}{
		"future date":         {with(map[string]any{"given_on": "2025-03-16"}), "given_on"},
		"before birth":        {with(map[string]any{"given_on": "2025-01-09"}), "given_on"},
		"malformed date":      {with(map[string]any{"given_on": "2025-2-1"}), "given_on"},
		"impossible date":     {with(map[string]any{"given_on": "2025-02-30"}), "given_on"},
		"no date":             {with(map[string]any{"given_on": "<drop>"}), "given_on"},
		"null date":           {with(map[string]any{"given_on": nil}), "given_on"},
		"date before antigen": {with(map[string]any{"given_on": "2025-03-16", "antigens": []string{}}), "given_on"},
		"empty antigens":      {with(map[string]any{"antigens": []string{}}), "antigens"},
		"no antigens":         {with(map[string]any{"antigens": "<drop>"}), "antigens"},
		"null antigens":       {with(map[string]any{"antigens": nil}), "antigens"},
		"unknown antigen":     {with(map[string]any{"antigens": []string{"polio", "smallpox"}}), "antigens"},
		"duplicate antigen":   {with(map[string]any{"antigens": []string{"polio", "polio"}}), "antigens"},
		"other, no product":   {with(map[string]any{"antigens": []string{"other"}}), "product"},
		"other, blank":        {with(map[string]any{"antigens": []string{"other"}, "product": "  "}), "product"},
		"long lot":            {with(map[string]any{"lot": long(41)}), "lot"},
		"long product":        {with(map[string]any{"product": long(81)}), "product"},
		"long note":           {with(map[string]any{"note": long(501)}), "note"},
		"lot before product":  {with(map[string]any{"lot": long(41), "product": long(81)}), "lot"},
		"empty lot":           {with(map[string]any{"lot": ""}), "lot"},
		"blank lot":           {with(map[string]any{"lot": " \t"}), "lot"},
		"empty product":       {with(map[string]any{"product": ""}), "product"},
		"blank product":       {with(map[string]any{"product": "  "}), "product"},
		"empty note":          {with(map[string]any{"note": ""}), "note"},
		"blank note":          {with(map[string]any{"note": " \n"}), "note"},
	} {
		wantInvalid2(t, name, k.do(http.MethodPost, path, ana, c.body), c.field)
	}
	rec := k.do(http.MethodPost, path, ana, with(map[string]any{"dose": 1}))
	if rec.Code != http.StatusUnprocessableEntity || strings.TrimSpace(rec.Body.String()) != `{"error":"unknown_field","field":"dose"}` {
		t.Errorf("unknown field: %d %q", rec.Code, rec.Body.String())
	}
	edge := k.recordDose(ana, kid, with(map[string]any{"given_on": "2025-01-10", "lot": long(40), "product": long(80), "note": long(500)}))
	k.recordDose(ana, kid, with(map[string]any{"given_on": "2025-03-15"}))

	one := path + "/" + edge
	for name, c := range map[string]struct {
		body  map[string]any
		field string
	}{
		"null date":     {map[string]any{"given_on": nil}, "given_on"},
		"null antigens": {map[string]any{"antigens": nil}, "antigens"},
		"future date":   {map[string]any{"given_on": "2025-03-16"}, "given_on"},
		"other":         {map[string]any{"antigens": []string{"other"}, "product": nil}, "product"},
		"long note":     {map[string]any{"note": long(501)}, "note"},
		"empty lot":     {map[string]any{"lot": ""}, "lot"},
		"empty product": {map[string]any{"product": ""}, "product"},
		"empty note":    {map[string]any{"note": ""}, "note"},
		"blank note":    {map[string]any{"note": "   "}, "note"},
	} {
		wantInvalid2(t, "patch "+name, k.do(http.MethodPatch, one, ana, c.body), c.field)
	}
	rec = k.do(http.MethodPatch, one, ana, map[string]any{"lot": nil, "antigens": []string{"tetanus", "diphtheria"}})
	if m := bodyMap(t, rec); rec.Code != http.StatusOK || m["lot"] != nil || m["product"] != long(80) || m["given_on"] != "2025-01-10" ||
		!slices.Equal(stringsOf(m["antigens"]), []string{"diphtheria", "tetanus"}) {
		t.Errorf("patch merge: %d %v", rec.Code, m)
	}
	if m := bodyMap(t, k.do(http.MethodGet, one, ana, nil)); m["lot"] != nil || len(m["antigens"].([]any)) != 2 {
		t.Errorf("stored after patch = %v", m)
	}

	// An unchanged stored date stays valid after the birth date moves past it.
	if rec := k.do(http.MethodPatch, "/v1/kids/"+kid, ana, map[string]any{"birth_date": "2025-01-20"}); rec.Code != http.StatusOK {
		t.Fatalf("move birth date: %d %q", rec.Code, rec.Body.String())
	}
	if rec := k.do(http.MethodPatch, one, ana, map[string]any{"note": "kept", "given_on": "2025-01-10"}); rec.Code != http.StatusOK {
		t.Errorf("patch with the unchanged early date: %d %q", rec.Code, rec.Body.String())
	}
	wantInvalid2(t, "a changed early date", k.do(http.MethodPatch, one, ana, map[string]any{"given_on": "2025-01-11"}), "given_on")

	if _, err := db.Exec(`WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i + 1 FROM n WHERE i < ?)
		INSERT INTO vaccine_doses(id, kid_id, administration, given_on, antigen, created_by, updated_by, created_at, updated_at)
		SELECT 'dose-' || i, ?, 'administration-' || i, '2025-02-01', 'polio', ?, ?, 100, 100 FROM n`,
		constants.MaxVaccineAdministrationsPerKid-2, kid, k.parentID("ana"), k.parentID("ana")); err != nil {
		t.Fatal(err)
	}
	wantErr(t, k.do(http.MethodPost, path, ana, ok), http.StatusConflict, "limit")

	dismiss := "/v1/kids/" + kid + "/vaccine-dismissals"
	for name, c := range map[string]struct {
		method, path string
		body         map[string]any
		field        string
	}{
		"unknown reason":         {http.MethodPut, dismiss + "/rsv-1", map[string]any{"reason": "forgot"}, "reason"},
		"no reason":              {http.MethodPut, dismiss + "/rsv-1", map[string]any{}, "reason"},
		"bad before":             {http.MethodPost, dismiss, map[string]any{"before": "2025-13-01", "reason": "not_needed"}, "before"},
		"no before":              {http.MethodPost, dismiss, map[string]any{"reason": "not_needed"}, "before"},
		"bulk unknown reason":    {http.MethodPost, dismiss, map[string]any{"before": "2025-03-01", "reason": "x"}, "reason"},
		"before checked first":   {http.MethodPost, dismiss, map[string]any{"before": "x", "reason": "x"}, "before"},
		"bulk null reason":       {http.MethodPost, dismiss, map[string]any{"before": "2025-03-01", "reason": nil}, "reason"},
		"put given_unknown typo": {http.MethodPut, dismiss + "/rsv-1", map[string]any{"reason": "given-unknown"}, "reason"},
	} {
		wantInvalid2(t, name, k.do(c.method, c.path, ana, c.body), c.field)
	}
	wantErr(t, k.do(http.MethodPut, dismiss+"/no-such-line", ana, map[string]any{"reason": "not_needed"}), http.StatusNotFound, "not_found")
	wantErr(t, k.do(http.MethodDelete, dismiss+"/no-such-line", ana, nil), http.StatusNotFound, "not_found")
}

func TestSwitchSchema(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.vaccineKid("us")
	k.recordDose(ana, kid, map[string]any{"given_on": "2025-01-10", "antigens": []string{"hepatitis_b"}})
	k.recordDose(ana, kid, map[string]any{"given_on": "2025-03-12", "antigens": []string{"diphtheria", "tetanus", "pertussis"}})
	rec := k.do(http.MethodPut, "/v1/kids/"+kid+"/vaccine-dismissals/rotavirus-1", ana, map[string]any{"reason": "given_unknown"})
	if m := bodyMap(t, rec); rec.Code != http.StatusOK || len(m) != 2 || m["line"] != "rotavirus-1" || m["reason"] != "given_unknown" {
		t.Fatalf("dismiss: %d %q", rec.Code, rec.Body.String())
	}
	if d := allLines(k.vaccines(ana, kid))["rotavirus-1"]; d["status"] != "dismissed" || d["dismissal"].(map[string]any)["reason"] != "given_unknown" {
		t.Errorf("us rotavirus-1 = %v", d)
	}

	rec = k.do(http.MethodPatch, "/v1/kids/"+kid, ana, map[string]any{"vaccine_schema": "uk"})
	if rec.Code != http.StatusOK || bodyMap(t, rec)["vaccine_schema"] != "uk" {
		t.Fatalf("switch: %d %q", rec.Code, rec.Body.String())
	}
	v := k.vaccines(ana, kid)
	lines := allLines(v)
	if v["schema"].(map[string]any)["id"] != "uk" {
		t.Errorf("schema = %v", v["schema"])
	}
	for _, id := range []string{"hepatitis_b-1", "diphtheria-1", "tetanus-1", "pertussis-1"} {
		if lines[id]["status"] != "given" {
			t.Errorf("uk %s = %v, want given by the kept dose", id, lines[id])
		}
	}
	if r := lines["rotavirus-1"]; r["status"] != "due" || r["dismissal"] != nil {
		t.Errorf("uk rotavirus-1 = %v, want the us dismissal not shown", r)
	}
	if n := len(v["other"].([]any)); n != 0 {
		t.Errorf("other = %v, want no dose shown twice", v["other"])
	}
	if rows := bodyMap(t, k.do(http.MethodGet, "/v1/kids/"+kid+"/vaccinations", ana, nil))["vaccinations"].([]any); len(rows) != 2 {
		t.Errorf("administrations after switch = %v", rows)
	}
	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		var body any
		if method == http.MethodPut {
			body = map[string]any{"reason": "not_needed"}
		}
		wantErr(t, k.do(method, "/v1/kids/"+kid+"/vaccine-dismissals/rsv-1", ana, body), http.StatusNotFound, "not_found")
	}

	k.do(http.MethodPatch, "/v1/kids/"+kid, ana, map[string]any{"vaccine_schema": "us"})
	if d := allLines(k.vaccines(ana, kid))["rotavirus-1"]; d["status"] != "dismissed" {
		t.Errorf("back on us, rotavirus-1 = %v, want its us dismissal again", d)
	}
}

func TestDismissBefore(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.vaccineKid("us")
	k.recordDose(ana, kid, map[string]any{"given_on": "2025-01-10", "antigens": []string{"hepatitis_b"}})
	bulk := func(kid, before string) *httpResult {
		t.Helper()
		rec := k.do(http.MethodPost, "/v1/kids/"+kid+"/vaccine-dismissals", ana, map[string]any{"before": before, "reason": "not_needed"})
		return &httpResult{rec.Code, rec.Body.String()}
	}

	if r := bulk(kid, "2025-01-10"); r.code != http.StatusOK || strings.TrimSpace(r.body) != `{"dismissed":0}` {
		t.Errorf("none eligible: %d %s", r.code, r.body)
	}
	if r := bulk(kid, "2025-03-11"); r.code != http.StatusOK || strings.TrimSpace(r.body) != `{"dismissed":9}` {
		t.Fatalf("before the 2-month visit's next day: %d %s", r.code, r.body)
	}
	lines := allLines(k.vaccines(ana, kid))
	for id, want := range map[string]string{
		"hepatitis_b-1": "given", "rsv-1": "dismissed", "hepatitis_b-2": "dismissed", "diphtheria-1": "dismissed",
		"rotavirus-1": "dismissed", "rotavirus-2": "upcoming", "rsv-2": "available", "covid19-1": "available",
	} {
		if got := lines[id]["status"]; got != want {
			t.Errorf("%s = %v, want %s", id, got, want)
		}
	}
	if r := bulk(kid, "2025-03-11"); strings.TrimSpace(r.body) != `{"dismissed":0}` {
		t.Errorf("repeat: %s", r.body)
	}

	undo := "/v1/kids/" + kid + "/vaccine-dismissals/rsv-1"
	for range 2 {
		if rec := k.do(http.MethodDelete, undo, ana, nil); rec.Code != http.StatusNoContent {
			t.Errorf("undo: %d %q", rec.Code, rec.Body.String())
		}
	}
	lines = allLines(k.vaccines(ana, kid))
	if lines["rsv-1"]["status"] != "due" || lines["hepatitis_b-2"]["status"] != "dismissed" {
		t.Errorf("after undo: rsv-1 %v, hepatitis_b-2 %v", lines["rsv-1"], lines["hepatitis_b-2"])
	}

	// rsv-1 and rotavirus-3 are past their windows (available): never dismissed in bulk.
	older := k.createKid(ana, map[string]any{"birth_date": "2024-01-10", "time_zone": "UTC", "vaccine_schema": "us"})
	before := allLines(k.vaccines(ana, older))
	if before["rsv-1"]["status"] != "available" || before["rotavirus-3"]["status"] != "available" {
		t.Fatalf("older kid: rsv-1 %v, rotavirus-3 %v", before["rsv-1"], before["rotavirus-3"])
	}
	if r := bulk(older, "2025-03-15"); r.code != http.StatusOK {
		t.Fatalf("older bulk: %d %s", r.code, r.body)
	}
	after := allLines(k.vaccines(ana, older))
	for id, l := range after {
		was := before[id]["status"]
		switch {
		case was == "available" || was == "given":
			if l["status"] != was {
				t.Errorf("%s: %v became %v", id, was, l["status"])
			}
		case l["from_date"].(string) < "2025-03-15":
			if l["status"] != "dismissed" {
				t.Errorf("%s: %v, want dismissed", id, l["status"])
			}
		default:
			if l["status"] != was {
				t.Errorf("%s from %v: %v became %v", id, l["from_date"], was, l["status"])
			}
		}
	}

	none := k.createKid(ana, map[string]any{"birth_date": "2025-01-10", "time_zone": "UTC"})
	wantErr(t, k.do(http.MethodPost, "/v1/kids/"+none+"/vaccine-dismissals", ana,
		map[string]any{"before": "2025-03-01", "reason": "not_needed"}), http.StatusNotFound, "not_found")
}

type httpResult struct {
	code int
	body string
}

func TestNoSchema(t *testing.T) {
	k := newKit(t)
	kid, ana, cara := k.vaccineKid(nil)
	rec := k.do(http.MethodGet, "/v1/kids/"+kid+"/vaccines", cara, nil)
	want := `{"schema":null,"today":"2025-03-15","groups":[],"also":[],"recurring":[],"other":[]}`
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != want {
		t.Errorf("no schema: %d %s", rec.Code, rec.Body.String())
	}
	id := k.recordDose(cara, kid, map[string]any{"given_on": "2025-02-01", "antigens": []string{"polio", "tetanus"}, "product": "Combo"})
	other := k.vaccines(cara, kid)["other"].([]any)
	if len(other) != 2 || other[0].(map[string]any)["administration"] != id || other[0].(map[string]any)["product"] != "Combo" {
		t.Errorf("other = %v", other)
	}
	for _, rq := range []struct {
		method, path string
		body         any
	}{
		{http.MethodPut, "/vaccine-dismissals/polio-1", map[string]any{"reason": "not_needed"}},
		{http.MethodDelete, "/vaccine-dismissals/polio-1", nil},
		{http.MethodPost, "/vaccine-dismissals", map[string]any{"before": "2025-03-01", "reason": "not_needed"}},
	} {
		wantErr(t, k.do(rq.method, "/v1/kids/"+kid+rq.path, ana, rq.body), http.StatusNotFound, "not_found")
	}
}

func TestFailedLoadAnswers503(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.vaccineKid("us")
	id := k.recordDose(ana, kid, map[string]any{"given_on": "2025-02-01", "antigens": []string{"polio"}})
	plain := k.createKid(ana, map[string]any{"birth_date": "2025-01-10"})
	saved := loadVaccines
	loadVaccines = func() error { return errors.New("vaccines: schedules unavailable") }
	t.Cleanup(func() { loadVaccines = saved })

	base := "/v1/kids/" + kid
	for _, rq := range []struct {
		method, path string
		body         any
	}{
		{http.MethodGet, "/v1/vaccine-schemas", nil},
		{http.MethodGet, "/v1/vaccine-catalogue", nil},
		{http.MethodGet, base + "/vaccines", nil},
		{http.MethodGet, "/v1/kids/" + plain + "/vaccines", nil},
		{http.MethodPost, base + "/vaccinations", map[string]any{"given_on": "2025-02-01", "antigens": []string{"polio"}}},
		{http.MethodPatch, base + "/vaccinations/" + id, map[string]any{"antigens": []string{"tetanus"}}},
		{http.MethodPut, base + "/vaccine-dismissals/rsv-1", map[string]any{"reason": "not_needed"}},
		{http.MethodDelete, base + "/vaccine-dismissals/rsv-1", nil},
		{http.MethodPost, base + "/vaccine-dismissals", map[string]any{"before": "2025-03-01", "reason": "not_needed"}},
		{http.MethodPatch, base, map[string]any{"vaccine_schema": "uk"}},
		{http.MethodPost, "/v1/kids", kidBody(map[string]any{"birth_date": "2025-01-10", "vaccine_schema": "us"})},
	} {
		wantErr(t, k.do(rq.method, rq.path, ana, rq.body), http.StatusServiceUnavailable, "unavailable")
	}
	if rec := k.do(http.MethodGet, base+"/vaccinations", ana, nil); rec.Code != http.StatusOK {
		t.Errorf("list needs no schedule: %d", rec.Code)
	}
	for _, rq := range []struct {
		path string
		body map[string]any
	}{
		{"/v1/kids/" + plain, map[string]any{"name": "Nova"}},
		{base, map[string]any{"name": "Nova"}},
		{base, map[string]any{"vaccine_schema": "us"}},
	} {
		if rec := k.do(http.MethodPatch, rq.path, ana, rq.body); rec.Code != http.StatusOK {
			t.Errorf("patch %v leaving the schedule as stored: %d %q", rq.body, rec.Code, rec.Body.String())
		}
	}
}

func TestVaccinesNoValuesInLogs(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.vaccineKid("us")
	k.Log.Reset()
	const lot, product, note = "LOT-Q7731", "Prodname-Zebra", "note-Kumquat"
	id := k.recordDose(ana, kid, map[string]any{"given_on": "2025-02-03", "antigens": []string{"other"}, "lot": lot, "product": product, "note": note})
	base := "/v1/kids/" + kid
	k.do(http.MethodPatch, base+"/vaccinations/"+id, ana, map[string]any{"given_on": "2025-02-04"})
	k.do(http.MethodPost, base+"/vaccinations", ana, map[string]any{"given_on": "2025-02-05", "antigens": []string{"other"}, "lot": lot + "x" + strings.Repeat("9", 40)})
	k.do(http.MethodGet, base+"/vaccines", ana, nil)
	k.do(http.MethodPut, base+"/vaccine-dismissals/rsv-1", ana, map[string]any{"reason": "not_needed"})
	k.do(http.MethodPost, base+"/vaccine-dismissals", ana, map[string]any{"before": "2025-02-06", "reason": "given_unknown"})
	k.do(http.MethodDelete, base+"/vaccinations/"+id, ana, nil)
	wantCleanLog(t, k.Log.String(), 7, lot, product, note, "2025-02-03", "2025-02-04", "2025-02-05", "2025-02-06", "2025-01-10", "rsv-1")
}

func TestKidVaccineSchema(t *testing.T) {
	k := newKit(t)
	ana, cara := k.asParent("ana"), k.asParent("cara")
	schema := func(kid string) any {
		t.Helper()
		return bodyMap(t, k.do(http.MethodGet, "/v1/kids/"+kid, ana, nil))["vaccine_schema"]
	}
	if got := schema(k.createKid(ana, nil)); got != nil {
		t.Errorf("created without vaccine_schema = %v, want null", got)
	}
	kid := k.createKid(ana, map[string]any{"vaccine_schema": "uk"})
	if got := schema(kid); got != "uk" {
		t.Fatalf("created with uk = %v", got)
	}
	for _, body := range []map[string]any{{}, {"name": "Nova"}} {
		if rec := k.do(http.MethodPatch, "/v1/kids/"+kid, ana, body); rec.Code != http.StatusOK || schema(kid) != "uk" {
			t.Fatalf("patch %v: %d %q", body, rec.Code, rec.Body.String())
		}
	}
	for _, bad := range []any{"ch", "", "UK", 3} {
		rec := k.do(http.MethodPatch, "/v1/kids/"+kid, ana, map[string]any{"vaccine_schema": bad})
		if _, isString := bad.(string); isString {
			wantInvalid(t, rec, "vaccine_schema")
			wantInvalid(t, k.do(http.MethodPost, "/v1/kids", ana, kidBody(map[string]any{"vaccine_schema": bad})), "vaccine_schema")
		} else if rec.Code != http.StatusBadRequest {
			t.Errorf("a number: %d", rec.Code)
		}
	}
	k.grant(ana, kid, "cara", "caregiver")
	wantErr(t, k.do(http.MethodPatch, "/v1/kids/"+kid, cara, map[string]any{"vaccine_schema": "us"}), http.StatusForbidden, "forbidden")
	if got := schema(kid); got != "uk" {
		t.Errorf("after refused patches = %v", got)
	}
	rec := k.do(http.MethodPatch, "/v1/kids/"+kid, ana, map[string]any{"vaccine_schema": nil})
	if rec.Code != http.StatusOK || bodyMap(t, rec)["vaccine_schema"] != nil || schema(kid) != nil {
		t.Errorf("clear: %d %q", rec.Code, rec.Body.String())
	}
}

func TestRetiredSchemaNeverBlocksAKidEdit(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.vaccineKid(nil)
	k.recordDose(ana, kid, map[string]any{"given_on": "2025-02-01", "antigens": []string{"polio"}})
	stored, err := k.Store.Kid(context.Background(), kid)
	if err != nil {
		t.Fatal(err)
	}
	retired := "ch"
	stored.VaccineSchema = &retired
	if err := k.Store.UpdateKid(context.Background(), stored, k.parentID("ana"), k.now); err != nil {
		t.Fatal(err)
	}
	for _, body := range []map[string]any{{"name": "Nova"}, {"time_zone": "Europe/Madrid"}, {"vaccine_schema": "ch"}} {
		rec := k.do(http.MethodPatch, "/v1/kids/"+kid, ana, body)
		if rec.Code != http.StatusOK || bodyMap(t, rec)["vaccine_schema"] != "ch" {
			t.Errorf("patch %v: %d %q", body, rec.Code, rec.Body.String())
		}
	}
	wantInvalid(t, k.do(http.MethodPatch, "/v1/kids/"+kid, ana, map[string]any{"vaccine_schema": "zz"}), "vaccine_schema")
	v := k.vaccines(ana, kid)
	if v["schema"] != nil || len(v["groups"].([]any)) != 0 || len(v["other"].([]any)) != 1 {
		t.Errorf("vaccines of a retired schedule = %v", v)
	}
}

func TestDismissalOnlyForLinesTheKidSees(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.vaccineKid("uk")
	dismiss := "/v1/kids/" + kid + "/vaccine-dismissals/"
	put := map[string]any{"reason": "not_needed"}
	// influenza-1 is recurring; measles-1-to202412 is the cohort born on or before 2024-12-31.
	for _, line := range []string{"influenza-1", "measles-1-to202412", "no-such-line"} {
		wantErr(t, k.do(http.MethodPut, dismiss+line, ana, put), http.StatusNotFound, "not_found")
		wantErr(t, k.do(http.MethodDelete, dismiss+line, ana, nil), http.StatusNotFound, "not_found")
	}
	if rec := k.do(http.MethodPut, dismiss+"measles-1-b202501", ana, put); rec.Code != http.StatusOK {
		t.Errorf("applicable cohort line: %d %q", rec.Code, rec.Body.String())
	}
	if rec := k.do(http.MethodDelete, dismiss+"rotavirus-1", ana, nil); rec.Code != http.StatusNoContent {
		t.Errorf("undo of a line never dismissed: %d %q", rec.Code, rec.Body.String())
	}

	us, _, _ := k.vaccineKid("us")
	for _, line := range []string{"covid19-1", "rsv-2", "influenza-1"} {
		wantErr(t, k.do(http.MethodPut, "/v1/kids/"+us+"/vaccine-dismissals/"+line, ana, put), http.StatusNotFound, "not_found")
	}
	rec := k.do(http.MethodPost, "/v1/kids/"+us+"/vaccine-dismissals", ana, map[string]any{"before": "2030-01-01", "reason": "not_needed"})
	if rec.Code != http.StatusOK {
		t.Fatalf("bulk: %d %q", rec.Code, rec.Body.String())
	}
	n := bodyMap(t, rec)["dismissed"].(float64)
	d, err := k.Store.Dismissals(context.Background(), us, "us")
	if err != nil || int(n) != len(d) || n == 0 {
		t.Errorf("dismissed %v, stored %d rows (%v)", n, len(d), err)
	}
	for _, line := range []string{"covid19-1", "rsv-2", "influenza-1"} {
		if _, found := d[line]; found {
			t.Errorf("bulk dismissed %s, which the kid never sees as due", line)
		}
	}
}

func TestVaccinationEdgesOverHTTP(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.vaccineKid("us")
	base := "/v1/kids/" + kid + "/vaccinations/"
	id := k.recordDose(ana, kid, map[string]any{"given_on": "2025-02-01", "antigens": []string{"polio"}, "lot": "A1"})
	wantInvalid2(t, "empty antigens", k.do(http.MethodPatch, base+id, ana, map[string]any{"antigens": []string{}}), "antigens")
	rec := k.do(http.MethodPatch, base+id, ana, map[string]any{"lots": "x"})
	if rec.Code != http.StatusUnprocessableEntity || strings.TrimSpace(rec.Body.String()) != `{"error":"unknown_field","field":"lots"}` {
		t.Errorf("unknown field: %d %q", rec.Code, rec.Body.String())
	}
	wantInvalid2(t, "long lot and product", k.do(http.MethodPatch, base+id, ana,
		map[string]any{"lot": strings.Repeat("x", 41), "product": strings.Repeat("x", 81)}), "lot")
	wantInvalid2(t, "long product", k.do(http.MethodPatch, base+id, ana, map[string]any{"product": strings.Repeat("x", 81)}), "product")

	other := k.createKid(ana, map[string]any{"birth_date": "2025-01-10", "time_zone": "UTC"})
	foreign := k.recordDose(ana, other, map[string]any{"given_on": "2025-02-01", "antigens": []string{"tetanus"}, "note": "theirs"})
	noKid := k.do(http.MethodGet, base+model.NewID(), ana, nil).Body.String()
	for _, method := range []string{http.MethodGet, http.MethodPatch, http.MethodDelete} {
		var body any
		if method == http.MethodPatch {
			body = map[string]any{"note": "mine"}
		}
		if rec := k.do(method, base+foreign, ana, body); rec.Code != http.StatusNotFound || rec.Body.String() != noKid {
			t.Errorf("%s another kid's administration: %d %q, want %q", method, rec.Code, rec.Body.String(), noKid)
		}
	}
	if m := bodyMap(t, k.do(http.MethodGet, "/v1/kids/"+other+"/vaccinations/"+foreign, ana, nil)); m["note"] != "theirs" {
		t.Errorf("the other kid's administration changed: %v", m)
	}

	hep := func(date string) string {
		return k.recordDose(ana, kid, map[string]any{"given_on": date, "antigens": []string{"hepatitis_b"}})
	}
	first, second, third := hep("2025-01-10"), hep("2025-02-10"), hep("2025-03-01")
	given := func() (map[string]any, []any) {
		v := k.vaccines(ana, kid)
		return map[string]any{"1": allLines(v)["hepatitis_b-1"]["administration"], "2": allLines(v)["hepatitis_b-2"]["administration"],
			"3": allLines(v)["hepatitis_b-3"]["administration"]}, v["other"].([]any)
	}
	if g, _ := given(); g["1"] != first || g["2"] != second || g["3"] != third {
		t.Fatalf("three doses = %v", g)
	}
	if rec := k.do(http.MethodDelete, base+second, ana, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d", rec.Code)
	}
	if g, _ := given(); g["1"] != first || g["2"] != third || g["3"] != nil {
		t.Errorf("after deleting the 2nd = %v, want the 3rd moved up one line", g)
	}
	hep("2025-03-05")
	extra := hep("2025-03-06")
	g, others := given()
	if g["3"] == nil || len(others) != 1 || others[0].(map[string]any)["administration"] != extra {
		t.Errorf("a 4th dose of a 3-line series: lines %v, other %v", g, others)
	}
}
