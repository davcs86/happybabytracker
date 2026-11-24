package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/milestones"
	"github.com/davcs86/happybabytracker/api/internal/model"
)

// milestoneKid creates a kid of ana (the default body with over applied, in UTC unless over names a zone) and grants
// cara the caregiver role.
func (k *kit) milestoneKid(over map[string]any) (kid, ana, cara string) {
	k.t.Helper()
	ana, cara = k.asParent("ana"), k.asParent("cara")
	body := map[string]any{"time_zone": "UTC"}
	for key, v := range over {
		body[key] = v
	}
	kid = k.createKid(ana, body)
	k.grant(ana, kid, "cara", "caregiver")
	return kid, ana, cara
}

func (k *kit) milestones(token, kid string) map[string]any {
	k.t.Helper()
	rec := k.do(http.MethodGet, "/v1/kids/"+kid+"/milestones", token, nil)
	if rec.Code != http.StatusOK {
		k.t.Fatalf("milestones: %d %q", rec.Code, rec.Body.String())
	}
	return bodyMap(k.t, rec)
}

func (k *kit) reach(token, kid string, body map[string]any) map[string]any {
	k.t.Helper()
	rec := k.do(http.MethodPost, "/v1/kids/"+kid+"/milestones", token, body)
	if rec.Code != http.StatusCreated {
		k.t.Fatalf("reach %v: %d %q", body, rec.Code, rec.Body.String())
	}
	return bodyMap(k.t, rec)
}

func shippedCatalog(t *testing.T) milestones.Catalog {
	t.Helper()
	c, ok := milestones.Get()
	if !ok {
		t.Fatal("the shipped catalog did not load")
	}
	return c
}

// firstItem is the first active catalog item of the age, in file order.
func firstItem(t *testing.T, ageMonths int) milestones.Item {
	t.Helper()
	for _, it := range shippedCatalog(t).Items {
		if it.AgeMonths == ageMonths && it.Retired == "" {
			return it
		}
	}
	t.Fatalf("no item for %d months", ageMonths)
	return milestones.Item{}
}

func objects(v any) []map[string]any {
	var out []map[string]any
	for _, o := range v.([]any) {
		out = append(out, o.(map[string]any))
	}
	return out
}

// positionsOf maps each checklist's age to its position, in the order the view lists them.
func positionsOf(v map[string]any) (ages []int, positions []string) {
	for _, cl := range objects(v["checklists"]) {
		ages = append(ages, int(cl["age_months"].(float64)))
		positions = append(positions, cl["position"].(string))
	}
	return ages, positions
}

// checklistItem finds the catalog item id in the view's checklists.
func checklistItem(t *testing.T, v map[string]any, id string) map[string]any {
	t.Helper()
	for _, cl := range objects(v["checklists"]) {
		for _, it := range objects(cl["items"]) {
			if it["id"] == id {
				return it
			}
		}
	}
	t.Fatalf("item %s is in no checklist", id)
	return nil
}

func TestMilestonesShape(t *testing.T) {
	k := newKit(t)
	kid, ana, cara := k.milestoneKid(map[string]any{"birth_date": "2026-03-04"})
	c := shippedCatalog(t)

	v := k.milestones(cara, kid)
	sameKeys(t, v, "catalog", "age", "catalog_shown", "checklists", "custom", "retired", "reached")
	cat := v["catalog"].(map[string]any)
	sameKeys(t, cat, "id", "name", "authority", "edition", "retrieved", "origin_url", "licence", "attribution", "screening_note")
	if cat["id"] != c.ID || cat["name"] != c.Name || cat["authority"] != c.Authority || cat["edition"] != c.Edition ||
		cat["retrieved"] != c.Retrieved || cat["origin_url"] != c.OriginURL || cat["licence"] != c.Licence ||
		cat["attribution"] != c.Attribution || cat["screening_note"] != c.ScreeningNote {
		t.Errorf("catalog = %v", cat)
	}
	raw := k.do(http.MethodGet, "/v1/kids/"+kid+"/milestones", cara, nil).Body.String()
	for _, o := range c.Omitted {
		if strings.Contains(raw, o) {
			t.Errorf("the view carries the omitted provenance %q", o)
		}
	}
	age := v["age"].(map[string]any)
	sameKeys(t, age, "chronological_months", "corrected_months", "basis_months")
	if age["chronological_months"] != float64(8) || age["corrected_months"] != nil || age["basis_months"] != float64(8) ||
		v["catalog_shown"] != true {
		t.Errorf("age %v, catalog_shown %v", age, v["catalog_shown"])
	}
	ages, positions := positionsOf(v)
	if !slices.Equal(ages, constants.MilestoneCatalogAges) {
		t.Errorf("checklist ages = %v", ages)
	}
	want := []string{"earlier", "earlier", "current", "next", "later", "later", "later", "later", "later", "later", "later", "later"}
	if !slices.Equal(positions, want) {
		t.Errorf("positions = %v, want %v", positions, want)
	}
	for _, cl := range objects(v["checklists"]) {
		sameKeys(t, cl, "age_months", "position", "items")
		var got, wantIDs []string
		for _, it := range objects(cl["items"]) {
			sameKeys(t, it, "id", "domain", "text", "reached")
			got = append(got, it["id"].(string))
			if it["reached"] != nil {
				t.Errorf("unreached item %v", it)
			}
		}
		for _, it := range c.Items {
			if float64(it.AgeMonths) == cl["age_months"] && it.Retired == "" {
				wantIDs = append(wantIDs, it.ID)
			}
		}
		if !slices.Equal(got, wantIDs) {
			t.Errorf("%v-month items = %v, want file order %v", cl["age_months"], got, wantIDs)
		}
	}
	first := objects(objects(v["checklists"])[0]["items"])[0]
	if it := c.Items[0]; first["id"] != it.ID || first["domain"] != string(it.Domain) || first["text"] != it.Text {
		t.Errorf("first item = %v, want %+v", first, it)
	}
	for _, key := range []string{"custom", "retired", "reached"} {
		if l, ok := v[key].([]any); !ok || len(l) != 0 {
			t.Errorf("%s = %v, want []", key, v[key])
		}
	}

	item := firstItem(t, 6)
	e := k.reach(cara, kid, map[string]any{"item": item.ID, "reached_on": "2026-10-01", "note": "in the garden"})
	sameKeys(t, e, "id", "item", "text", "reached_on", "note")
	if e["item"] != item.ID || e["text"] != item.Text || e["reached_on"] != "2026-10-01" || e["note"] != "in the garden" {
		t.Errorf("item entry = %v", e)
	}
	custom := k.reach(ana, kid, map[string]any{"text": "  Waves bye  ", "reached_on": "2026-11-01"})
	sameKeys(t, custom, "id", "item", "text", "reached_on", "note")
	if custom["item"] != nil || custom["text"] != "Waves bye" || custom["note"] != nil {
		t.Errorf("custom entry = %v", custom)
	}
	rec := k.do(http.MethodGet, "/v1/kids/"+kid+"/milestones/"+e["id"].(string), ana, nil)
	if got := bodyMap(t, rec); rec.Code != http.StatusOK || got["id"] != e["id"] || got["text"] != item.Text {
		t.Errorf("get entry = %d %v", rec.Code, got)
	}

	v = k.milestones(ana, kid)
	reached := checklistItem(t, v, item.ID)["reached"].(map[string]any)
	sameKeys(t, reached, "id", "reached_on", "note")
	if reached["id"] != e["id"] || reached["reached_on"] != "2026-10-01" || reached["note"] != "in the garden" {
		t.Errorf("reached = %v", reached)
	}
	cs := objects(v["custom"])
	if len(cs) != 1 {
		t.Fatalf("custom = %v", v["custom"])
	}
	sameKeys(t, cs[0], "id", "text", "reached_on", "note")
	if cs[0]["id"] != custom["id"] || cs[0]["text"] != "Waves bye" || cs[0]["reached_on"] != "2026-11-01" || cs[0]["note"] != nil {
		t.Errorf("custom row = %v", cs[0])
	}
}

func TestMilestonesPositionsThroughAPI(t *testing.T) {
	k := newKit(t)
	preterm := map[string]any{"gestation_weeks": 32, "gestation_days": 0}
	for _, c := range []struct {
		name, birth             string
		chrono, basis           float64
		corrected               any
		currentIndex, nextIndex int
	}{
		// 32 weeks: the due date is 8 weeks after birth.
		{"3 months chronological", "2026-08-20", 3, 1, float64(1), -1, 0},
		{"5 months chronological", "2026-06-20", 5, 3, float64(3), 0, 1},
		{"2 years 1 month", "2024-10-20", 25, 25, nil, 7, 8},
	} {
		over := map[string]any{"birth_date": c.birth}
		for key, v := range preterm {
			over[key] = v
		}
		kid, ana, _ := k.milestoneKid(over)
		v := k.milestones(ana, kid)
		age := v["age"].(map[string]any)
		if age["chronological_months"] != c.chrono || age["corrected_months"] != c.corrected || age["basis_months"] != c.basis {
			t.Errorf("%s: age = %v", c.name, age)
		}
		ages, positions := positionsOf(v)
		if !slices.Equal(ages, constants.MilestoneCatalogAges) || v["catalog_shown"] != true {
			t.Errorf("%s: ages %v, catalog_shown %v", c.name, ages, v["catalog_shown"])
		}
		for i, p := range positions {
			want := "later"
			switch {
			case i == c.currentIndex:
				want = "current"
			case i == c.nextIndex:
				want = "next"
			case i < c.nextIndex:
				want = "earlier"
			}
			if p != want {
				t.Errorf("%s: %d-month list is %s, want %s", c.name, ages[i], p, want)
			}
		}
	}
}

func TestCatalogSpanThroughAPI(t *testing.T) {
	k := newKit(t)
	// Born 2021-11-21: the 5th birthday is 2026-11-21, which starts at 2026-11-20 11:00 UTC in Auckland (UTC+13).
	kid, ana, _ := k.milestoneKid(map[string]any{"birth_date": "2021-11-21", "time_zone": "Pacific/Auckland"})
	utcKid, _, _ := k.milestoneKid(map[string]any{"birth_date": "2021-11-21"})
	five, four := firstItem(t, 60), firstItem(t, 48)

	k.now = time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	v := k.milestones(ana, kid)
	ages, positions := positionsOf(v)
	if v["catalog_shown"] != true || len(ages) != 12 || positions[10] != "current" || positions[11] != "next" ||
		v["age"].(map[string]any)["chronological_months"] != float64(59) {
		t.Fatalf("4 years 11 months: shown %v, positions %v, age %v", v["catalog_shown"], positions, v["age"])
	}
	reached := k.reach(ana, kid, map[string]any{"item": five.ID, "reached_on": "2026-10-20", "note": "with a ball"})
	k.reach(ana, kid, map[string]any{"item": four.ID, "reached_on": "2026-10-21"})
	k.reach(ana, kid, map[string]any{"text": "Rides a bike", "reached_on": "2026-10-22"})

	k.now = time.Date(2026, 11, 20, 10, 59, 59, 0, time.UTC)
	if v := k.milestones(ana, kid); v["catalog_shown"] != true || len(objects(v["checklists"])) != 12 || len(v["reached"].([]any)) != 0 {
		t.Errorf("the day before the 5th birthday (kid-local): shown %v, %d lists", v["catalog_shown"], len(v["checklists"].([]any)))
	}

	k.now = time.Date(2026, 11, 20, 11, 0, 0, 0, time.UTC)
	v = k.milestones(ana, kid)
	if v["catalog_shown"] != false || len(v["checklists"].([]any)) != 0 {
		t.Fatalf("the 5th birthday in Auckland: shown %v, checklists %v", v["catalog_shown"], v["checklists"])
	}
	rows := objects(v["reached"])
	if len(rows) != 2 {
		t.Fatalf("reached = %v", v["reached"])
	}
	sameKeys(t, rows[1], "id", "item", "age_months", "domain", "text", "reached_on", "note")
	if r := rows[1]; r["id"] != reached["id"] || r["item"] != five.ID || r["age_months"] != float64(60) ||
		r["domain"] != string(five.Domain) || r["text"] != five.Text || r["reached_on"] != "2026-10-20" || r["note"] != "with a ball" {
		t.Errorf("reached 5-year row = %v", r)
	}
	if rows[0]["item"] != four.ID {
		t.Errorf("reached rows are not newest first: %v", rows)
	}
	if cs := objects(v["custom"]); len(cs) != 1 || cs[0]["text"] != "Rides a bike" || len(v["retired"].([]any)) != 0 {
		t.Errorf("custom %v retired %v", v["custom"], v["retired"])
	}
	if v := k.milestones(ana, utcKid); v["catalog_shown"] != true || len(objects(v["checklists"])) != 12 {
		t.Errorf("the same birth date in UTC is still the day before: shown %v", v["catalog_shown"])
	}
}

func TestMilestonesWithModuleOff(t *testing.T) {
	k := newKit(t)
	kid, ana, cara := k.milestoneKid(map[string]any{"birth_date": "2024-05-20"})
	m := moduleItem(t, k.modules(ana, kid), "milestones")
	if m["enabled"] != false || m["default"] != false || m["overridden"] != false {
		t.Fatalf("30 months: milestones module = %v, want off by default", m)
	}
	v := k.milestones(cara, kid)
	if v["catalog_shown"] != true || len(objects(v["checklists"])) != 12 || v["age"].(map[string]any)["basis_months"] != float64(30) {
		t.Errorf("module off: shown %v, age %v", v["catalog_shown"], v["age"])
	}
	k.reach(cara, kid, map[string]any{"item": firstItem(t, 30).ID, "reached_on": "2026-11-01"})
	if m := moduleItem(t, k.modules(ana, kid), "milestones"); m["enabled"] != true || m["default"] != false || m["overridden"] != true {
		t.Errorf("after the first mark at 24+: milestones module = %v, want on as a stored override", m)
	}
}

func TestRolesAndNotFound(t *testing.T) {
	k := newKit(t)
	kid, ana, cara := k.milestoneKid(map[string]any{"birth_date": "2026-03-04"})
	other := k.createKid(ana, map[string]any{"birth_date": "2026-03-04", "time_zone": "UTC"})
	xavi, admin := k.asParent("xavi"), k.asAdmin()
	base := "/v1/kids/" + kid + "/milestones"

	for i, token := range []string{ana, cara} {
		item := firstItem(t, 2+2*i)
		e := k.reach(token, kid, map[string]any{"item": item.ID, "reached_on": "2026-06-01"})
		one := base + "/" + e["id"].(string)
		if rec := k.do(http.MethodGet, one, token, nil); rec.Code != http.StatusOK {
			t.Errorf("get: %d %q", rec.Code, rec.Body.String())
		}
		if rec := k.do(http.MethodPatch, one, token, map[string]any{"note": "ok"}); rec.Code != http.StatusOK || bodyMap(t, rec)["note"] != "ok" {
			t.Errorf("patch: %d %q", rec.Code, rec.Body.String())
		}
		if rec := k.do(http.MethodGet, base, token, nil); rec.Code != http.StatusOK {
			t.Errorf("list: %d %q", rec.Code, rec.Body.String())
		}
		if rec := k.do(http.MethodDelete, one, token, nil); rec.Code != http.StatusNoContent {
			t.Errorf("delete: %d %q", rec.Code, rec.Body.String())
		}
	}
	kept := k.reach(ana, kid, map[string]any{"text": "Claps", "reached_on": "2026-06-01"})["id"].(string)
	foreign := k.reach(ana, other, map[string]any{"text": "Claps", "reached_on": "2026-06-01"})["id"].(string)

	type rq struct {
		method, path string
		body         any
	}
	all := []rq{
		{http.MethodGet, base, nil},
		{http.MethodPost, base, map[string]any{"text": "Claps", "reached_on": "2026-06-02"}},
		{http.MethodGet, base + "/" + kept, nil},
		{http.MethodPatch, base + "/" + kept, map[string]any{"note": "x"}},
		{http.MethodDelete, base + "/" + kept, nil},
	}
	noKid := `{"error":"not_found"}`
	for _, c := range all {
		rec := k.do(c.method, c.path, xavi, c.body)
		unknown := k.do(c.method, strings.Replace(c.path, kid, model.NewID(), 1), ana, c.body)
		if rec.Code != http.StatusNotFound || unknown.Code != http.StatusNotFound || rec.Body.String() != unknown.Body.String() ||
			strings.TrimSpace(rec.Body.String()) != noKid {
			t.Errorf("%s %s: no grant %d %q, unknown kid %d %q", c.method, c.path, rec.Code, rec.Body.String(), unknown.Code, unknown.Body.String())
		}
		wantErr(t, k.do(c.method, c.path, admin, c.body), http.StatusForbidden, "forbidden")
	}
	for _, c := range []rq{
		{http.MethodGet, base + "/" + foreign, nil},
		{http.MethodPatch, base + "/" + foreign, map[string]any{"note": "x"}},
		{http.MethodDelete, base + "/" + foreign, nil},
		{http.MethodGet, base + "/" + model.NewID(), nil},
		{http.MethodPatch, base + "/not-an-id", map[string]any{}},
		{http.MethodDelete, base + "/not-an-id", nil},
	} {
		if rec := k.do(c.method, c.path, ana, c.body); rec.Code != http.StatusNotFound || strings.TrimSpace(rec.Body.String()) != noKid {
			t.Errorf("%s %s: %d %q, want the no-kid 404", c.method, c.path, rec.Code, rec.Body.String())
		}
	}
	if rec := k.do(http.MethodGet, "/v1/kids/"+other+"/milestones/"+foreign, cara, nil); rec.Code != http.StatusNotFound {
		t.Errorf("a caregiver of another kid reads its entry: %d", rec.Code)
	}
	if rec := k.do(http.MethodGet, "/v1/kids/"+other+"/milestones/"+foreign, ana, nil); rec.Code != http.StatusOK {
		t.Errorf("the foreign entry under its own kid: %d", rec.Code)
	}
}

func TestMilestoneValidation(t *testing.T) {
	k, db := fileKit(t)
	kid, ana, _ := k.milestoneKid(map[string]any{"birth_date": "2026-03-04"})
	path := "/v1/kids/" + kid + "/milestones"
	item := firstItem(t, 6)
	runes := func(n int) string { return strings.Repeat("é", n) }
	with := func(base, over map[string]any) map[string]any {
		b := map[string]any{}
		for key, v := range base {
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
	okItem := map[string]any{"item": item.ID, "reached_on": "2026-10-01"}
	okText := map[string]any{"text": "Claps", "reached_on": "2026-10-01"}
	for name, c := range map[string]struct {
		body  map[string]any
		field string
	}{
		"both item and text":     {with(okItem, map[string]any{"text": "Claps"}), "item"},
		"neither":                {with(okItem, map[string]any{"item": "<drop>"}), "item"},
		"null item":              {with(okItem, map[string]any{"item": nil}), "item"},
		"null item, text":        {with(okText, map[string]any{"item": nil}), "item"},
		"unknown item":           {with(okItem, map[string]any{"item": "2m-se-no-such-item"}), "item"},
		"empty item":             {with(okItem, map[string]any{"item": ""}), "item"},
		"null text":              {with(okText, map[string]any{"text": nil}), "text"},
		"text of 201":            {with(okText, map[string]any{"text": runes(201)}), "text"},
		"whitespace text":        {with(okText, map[string]any{"text": " \t\n "}), "text"},
		"empty text":             {with(okText, map[string]any{"text": ""}), "text"},
		"future date":            {with(okItem, map[string]any{"reached_on": "2026-11-21"}), "reached_on"},
		"before birth":           {with(okItem, map[string]any{"reached_on": "2026-03-03"}), "reached_on"},
		"malformed date":         {with(okItem, map[string]any{"reached_on": "2026-6-1"}), "reached_on"},
		"impossible date":        {with(okItem, map[string]any{"reached_on": "2026-06-31"}), "reached_on"},
		"no date":                {with(okItem, map[string]any{"reached_on": "<drop>"}), "reached_on"},
		"null date":              {with(okItem, map[string]any{"reached_on": nil}), "reached_on"},
		"long note":              {with(okItem, map[string]any{"note": strings.Repeat("x", constants.MaxNoteLen+1)}), "note"},
		"empty note":             {with(okItem, map[string]any{"note": ""}), "note"},
		"blank note":             {with(okItem, map[string]any{"note": "  "}), "note"},
		"item before date":       {with(okItem, map[string]any{"item": "nope", "reached_on": "x"}), "item"},
		"text before date":       {with(okText, map[string]any{"text": " ", "reached_on": "x"}), "text"},
		"date before note":       {with(okItem, map[string]any{"reached_on": "x", "note": ""}), "reached_on"},
		"text of 201 with note":  {with(okText, map[string]any{"text": runes(201), "note": ""}), "text"},
		"unknown item with note": {with(okItem, map[string]any{"item": "nope", "note": ""}), "item"},
	} {
		wantInvalid2(t, name, k.do(http.MethodPost, path, ana, c.body), c.field)
	}
	for _, body := range []map[string]any{with(okItem, map[string]any{"age": 6}), with(okText, map[string]any{"custom_text": "x"})} {
		rec := k.do(http.MethodPost, path, ana, body)
		if rec.Code != http.StatusUnprocessableEntity || !strings.HasPrefix(rec.Body.String(), `{"error":"unknown_field"`) {
			t.Errorf("unknown field %v: %d %q", body, rec.Code, rec.Body.String())
		}
	}

	if e := k.reach(ana, kid, with(okText, map[string]any{"text": runes(200), "reached_on": "2026-03-04"})); e["text"] != runes(200) {
		t.Errorf("200 characters on the birth date = %v", e)
	}
	e := k.reach(ana, kid, with(okItem, map[string]any{"reached_on": "2026-11-20", "note": strings.Repeat("x", constants.MaxNoteLen)}))
	wantErr(t, k.do(http.MethodPost, path, ana, with(okItem, map[string]any{"reached_on": "2026-10-02"})), http.StatusConflict, "already_reached")

	auckland, _, _ := k.milestoneKid(map[string]any{"birth_date": "2026-03-04", "time_zone": "Pacific/Auckland"})
	aPath := "/v1/kids/" + auckland + "/milestones"
	if rec := k.do(http.MethodPost, aPath, ana, with(okItem, map[string]any{"reached_on": "2026-11-21"})); rec.Code != http.StatusCreated {
		t.Errorf("the kid-local today in Auckland: %d %q", rec.Code, rec.Body.String())
	}
	wantInvalid2(t, "after the Auckland today", k.do(http.MethodPost, aPath, ana, with(okText, map[string]any{"reached_on": "2026-11-22"})), "reached_on")

	itemRow := path + "/" + e["id"].(string)
	customRow := path + "/" + k.reach(ana, kid, okText)["id"].(string)
	for name, c := range map[string]struct {
		path  string
		body  map[string]any
		field string
	}{
		"item":                 {itemRow, map[string]any{"item": item.ID}, "item"},
		"another item":         {itemRow, map[string]any{"item": firstItem(t, 9).ID}, "item"},
		"null item":            {itemRow, map[string]any{"item": nil}, "item"},
		"item on custom":       {customRow, map[string]any{"item": item.ID}, "item"},
		"text on item row":     {itemRow, map[string]any{"text": "Claps"}, "item"},
		"null text":            {customRow, map[string]any{"text": nil}, "text"},
		"text of 201":          {customRow, map[string]any{"text": runes(201)}, "text"},
		"whitespace text":      {customRow, map[string]any{"text": "   "}, "text"},
		"null date":            {customRow, map[string]any{"reached_on": nil}, "reached_on"},
		"future date":          {itemRow, map[string]any{"reached_on": "2026-11-21"}, "reached_on"},
		"before birth":         {itemRow, map[string]any{"reached_on": "2026-03-03"}, "reached_on"},
		"long note":            {customRow, map[string]any{"note": strings.Repeat("x", constants.MaxNoteLen+1)}, "note"},
		"empty note":           {itemRow, map[string]any{"note": ""}, "note"},
		"text before the date": {customRow, map[string]any{"text": "", "reached_on": "x"}, "text"},
	} {
		wantInvalid2(t, "patch "+name, k.do(http.MethodPatch, c.path, ana, c.body), c.field)
	}
	rec := k.do(http.MethodPatch, customRow, ana, map[string]any{"text": " Claps hands ", "note": "two"})
	if m := bodyMap(t, rec); rec.Code != http.StatusOK || m["text"] != "Claps hands" || m["note"] != "two" || m["reached_on"] != "2026-10-01" || m["item"] != nil {
		t.Errorf("patch custom: %d %v", rec.Code, m)
	}
	rec = k.do(http.MethodPatch, itemRow, ana, map[string]any{"note": nil, "reached_on": "2026-09-30"})
	if m := bodyMap(t, rec); rec.Code != http.StatusOK || m["note"] != nil || m["reached_on"] != "2026-09-30" || m["text"] != item.Text || m["item"] != item.ID {
		t.Errorf("patch item row: %d %v", rec.Code, m)
	}
	if m := bodyMap(t, k.do(http.MethodGet, itemRow, ana, nil)); m["note"] != nil || m["reached_on"] != "2026-09-30" {
		t.Errorf("stored after patch = %v", m)
	}

	// An unchanged stored date stays valid after the birth date moves past it.
	if rec := k.do(http.MethodPatch, "/v1/kids/"+kid, ana, map[string]any{"birth_date": "2026-03-10"}); rec.Code != http.StatusOK {
		t.Fatalf("move birth date: %d %q", rec.Code, rec.Body.String())
	}
	var early string
	for _, c := range objects(k.milestones(ana, kid)["custom"]) {
		if c["reached_on"] == "2026-03-04" {
			early = path + "/" + c["id"].(string)
		}
	}
	if rec := k.do(http.MethodPatch, early, ana, map[string]any{"note": "kept", "reached_on": "2026-03-04"}); rec.Code != http.StatusOK {
		t.Errorf("patch with the unchanged early date: %d %q", rec.Code, rec.Body.String())
	}
	wantInvalid2(t, "a changed early date", k.do(http.MethodPatch, early, ana, map[string]any{"reached_on": "2026-03-05"}), "reached_on")

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM milestones_reached WHERE kid_id = ?`, kid).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i + 1 FROM n WHERE i < ?)
		INSERT INTO milestones_reached(id, kid_id, custom_text, reached_on, created_by, updated_by, created_at, updated_at)
		SELECT 'milestone-' || i, ?, 'Row ' || i, '2026-06-01', ?, ?, 100, 100 FROM n`,
		constants.MaxMilestonesPerKid-n, kid, k.parentID("ana"), k.parentID("ana")); err != nil {
		t.Fatal(err)
	}
	wantErr(t, k.do(http.MethodPost, path, ana, okText), http.StatusConflict, "limit")
	wantErr(t, k.do(http.MethodPost, path, ana, with(okItem, map[string]any{"item": firstItem(t, 12).ID})), http.StatusConflict, "limit")

	saved := loadMilestones
	loadMilestones = func() error { return errors.New("milestones: catalog unavailable") }
	t.Cleanup(func() { loadMilestones = saved })
	for _, c := range []struct {
		method, path string
		body         any
	}{
		{http.MethodGet, path, nil},
		{http.MethodPost, aPath, okText},
		{http.MethodGet, itemRow, nil},
		{http.MethodPatch, itemRow, map[string]any{"note": "x"}},
	} {
		wantErr(t, k.do(c.method, c.path, ana, c.body), http.StatusServiceUnavailable, "unavailable")
	}
	if rec := k.do(http.MethodDelete, itemRow, ana, nil); rec.Code != http.StatusNoContent {
		t.Errorf("delete needs no catalog: %d %q", rec.Code, rec.Body.String())
	}
}

// testCatalog holds one active and one retired 2-month item.
func testCatalog() milestones.Catalog {
	return milestones.Catalog{ID: "test", Name: "Test", Authority: "A", Edition: "E", Retrieved: "2026-01-01",
		OriginURL: "https://www.cdc.gov/", Licence: "L", Attribution: "Source: test", ScreeningNote: "S",
		Omitted: []string{"Left out"},
		Items: []milestones.Item{
			{ID: "2m-se-active", AgeMonths: 2, Domain: milestones.DomainSocialEmotional, Text: "Active text", SourceURL: "https://www.cdc.gov/"},
			{ID: "2m-se-retired", AgeMonths: 2, Domain: milestones.DomainSocialEmotional, Text: "Retired text", SourceURL: "https://www.cdc.gov/", Retired: "2026-06-01"},
		}}
}

func TestRetiredRowListed(t *testing.T) {
	c := testCatalog()
	retired, note := "2m-se-retired", "kept"
	rows := []model.Milestone{{Entry: model.Entry{ID: "row-1", Note: &note}, Item: &retired, ReachedOn: "2026-05-01"}}
	for _, shown := range []bool{true, false} {
		v, err := newMilestonesView(c, rows, milestones.AgeBasis{ChronologicalMonths: 3, BasisMonths: 3}, shown)
		if err != nil {
			t.Fatal(err)
		}
		if len(v.Retired) != 1 || v.Retired[0] != (retiredView{ID: "row-1", Item: retired, Text: "Retired text", ReachedOn: "2026-05-01", Note: &note}) {
			t.Errorf("shown %v: retired = %+v", shown, v.Retired)
		}
		if len(v.Reached) != 0 || len(v.Custom) != 0 {
			t.Errorf("shown %v: reached %+v custom %+v", shown, v.Reached, v.Custom)
		}
		for _, cl := range v.Checklists {
			for _, it := range cl.Items {
				if it.ID == retired {
					t.Errorf("shown %v: the retired item is in the %d-month checklist", shown, cl.AgeMonths)
				}
			}
		}
	}
	e, err := newMilestoneView(rows[0], c)
	if err != nil || e.Text != "Retired text" || e.Item == nil || *e.Item != retired {
		t.Errorf("entry view = %+v, %v", e, err)
	}
	in := milestoneInput{Item: optionalString{Set: true, V: &retired}, ReachedOn: optionalString{Set: true, V: strPtr("2026-05-01")}}
	if _, field := in.mergeOnto(model.Milestone{}, c, "2026-01-01", time.UTC, t0); field != "item" {
		t.Errorf("reaching a retired item: field %q, want item", field)
	}
	edit := milestoneInput{Note: optionalString{Set: true, V: strPtr("edited")}}
	if m, field := edit.mergeOnto(rows[0], c, "2026-01-01", time.UTC, t0); field != "" || m.Text != nil || *m.Note != "edited" {
		t.Errorf("a note edit on a retired-item row: field %q, row %+v", field, m)
	}
	unknown := "2m-se-gone"
	if _, err := newMilestonesView(c, []model.Milestone{{Item: &unknown, ReachedOn: "2026-05-01"}}, milestones.AgeBasis{}, true); err == nil {
		t.Error("a stored item missing from the catalog was listed")
	}
}

func strPtr(s string) *string { return &s }

func TestUndoReturnsItemToChecklist(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.milestoneKid(map[string]any{"birth_date": "2026-03-04"})
	item := firstItem(t, 4)
	e := k.reach(ana, kid, map[string]any{"item": item.ID, "reached_on": "2026-07-01"})
	if checklistItem(t, k.milestones(ana, kid), item.ID)["reached"] == nil {
		t.Fatal("the reached item shows no reached row")
	}
	if rec := k.do(http.MethodDelete, "/v1/kids/"+kid+"/milestones/"+e["id"].(string), ana, nil); rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Fatalf("undo: %d %q", rec.Code, rec.Body.String())
	}
	if r := checklistItem(t, k.milestones(ana, kid), item.ID)["reached"]; r != nil {
		t.Errorf("after undo: reached = %v", r)
	}
	k.reach(ana, kid, map[string]any{"item": item.ID, "reached_on": "2026-07-02"})
}

// judgementRE is the plan's Global Constraints denylist, as whole words.
var judgementRE = regexp.MustCompile(`(?i)\b(late|behind|delayed|delay|normal|abnormal|typical|atypical|on\s+track|ahead|missed|missing|concern|worry|should|red\s+flag|warning)\b`)

// judged lists every string value under v that matches the denylist, except the catalog's own item text and
// attribution.
func judged(v any, key string, parent map[string]any) []string {
	switch x := v.(type) {
	case map[string]any:
		var out []string
		for k, c := range x {
			out = append(out, judged(c, k, x)...)
		}
		return out
	case []any:
		var out []string
		for _, c := range x {
			out = append(out, judged(c, key, parent)...)
		}
		return out
	case string:
		_, catalogText := parent["domain"]
		if key == "attribution" || (key == "text" && (catalogText || parent["item"] != nil)) {
			return nil
		}
		if judgementRE.MatchString(x) {
			return []string{key + ": " + x}
		}
	}
	return nil
}

func TestNoJudgementInViews(t *testing.T) {
	if got := judged(map[string]any{"note": "Reached late"}, "", nil); len(got) != 1 {
		t.Fatalf("the check misses a denylisted word: %v", got)
	}
	k := newKit(t)
	kid, ana, _ := k.milestoneKid(map[string]any{"birth_date": "2021-12-01", "gestation_weeks": 33, "gestation_days": 1})
	base := "/v1/kids/" + kid + "/milestones"
	k.now = time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	var bodies []string
	record := func(method, path string, body any) map[string]any {
		rec := k.do(method, path, ana, body)
		bodies = append(bodies, rec.Body.String())
		if rec.Body.Len() == 0 {
			return nil
		}
		return bodyMap(t, rec)
	}
	item := firstItem(t, 60)
	e := record(http.MethodPost, base, map[string]any{"item": item.ID, "reached_on": "2026-10-01", "note": "at home"})
	c := record(http.MethodPost, base, map[string]any{"text": "Says a long sentence", "reached_on": "2026-10-02"})
	record(http.MethodPost, base, map[string]any{"item": item.ID, "reached_on": "2026-10-03"})
	record(http.MethodPost, base, map[string]any{"item": "nope", "reached_on": "2026-10-03"})
	record(http.MethodPatch, base+"/"+e["id"].(string), map[string]any{"note": "in the park"})
	record(http.MethodGet, base+"/"+c["id"].(string), nil)
	record(http.MethodGet, base, nil)
	k.now = time.Date(2026, 12, 2, 12, 0, 0, 0, time.UTC)
	if v := record(http.MethodGet, base, nil); v["catalog_shown"] != false || len(v["reached"].([]any)) != 1 {
		t.Fatalf("after the 5th birthday: %v", v)
	}
	record(http.MethodGet, base+"/"+model.NewID(), nil)
	for _, b := range bodies {
		var v any
		if strings.TrimSpace(b) != "" {
			if err := json.Unmarshal([]byte(b), &v); err != nil {
				t.Fatalf("body %q: %v", b, err)
			}
		}
		for _, hit := range judged(v, "", nil) {
			t.Errorf("judgement wording in a view: %s", hit)
		}
	}
}

func TestMilestonesNoValuesInLogs(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.milestoneKid(map[string]any{"birth_date": "2026-03-04"})
	item := firstItem(t, 6)
	k.Log.Reset()
	const text, note = "Zebra-Kumquat dance", "note-Quince7"
	base := "/v1/kids/" + kid + "/milestones"
	id := k.reach(ana, kid, map[string]any{"text": text, "reached_on": "2026-10-03", "note": note})["id"].(string)
	k.reach(ana, kid, map[string]any{"item": item.ID, "reached_on": "2026-10-05"})
	k.do(http.MethodPost, base, ana, map[string]any{"item": item.ID, "reached_on": "2026-10-06"})
	k.do(http.MethodPost, base, ana, map[string]any{"text": text + strings.Repeat("9", 200), "reached_on": "2026-10-07"})
	k.do(http.MethodPatch, base+"/"+id, ana, map[string]any{"reached_on": "2026-10-04"})
	k.do(http.MethodGet, base, ana, nil)
	k.do(http.MethodGet, base+"/"+id, ana, nil)
	k.do(http.MethodDelete, base+"/"+id, ana, nil)
	wantCleanLog(t, k.Log.String(), 8, text, note, "2026-10-03", "2026-10-04", "2026-10-05", "2026-10-06", "2026-10-07",
		"2026-03-04", item.ID, "Kumquat", "Quince")
}
