package httpapi

import (
	"context"
	"errors"
	"maps"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

const testKidName = "Lía Wren"

func kidBody(over map[string]any) map[string]any {
	b := map[string]any{"name": testKidName, "birth_date": "2026-03-04", "sex": "female", "time_zone": "Europe/Madrid"}
	maps.Copy(b, over)
	for key, v := range over {
		if v == nil {
			delete(b, key)
		}
	}
	return b
}

// createKid posts a kid (the default body with over applied; a nil value drops the key) and returns its id.
func (k *kit) createKid(token string, over map[string]any) string {
	k.t.Helper()
	rec := k.do(http.MethodPost, "/v1/kids", token, kidBody(over))
	if rec.Code != http.StatusCreated {
		k.t.Fatalf("create kid: %d %q", rec.Code, rec.Body.String())
	}
	return bodyMap(k.t, rec)["id"].(string)
}

func (k *kit) grant(token, kid, username, role string) {
	k.t.Helper()
	rec := k.do(http.MethodPut, "/v1/kids/"+kid+"/access/"+username, token, map[string]string{"role": role})
	if rec.Code != http.StatusNoContent {
		k.t.Fatalf("grant %s %s: %d %q", username, role, rec.Code, rec.Body.String())
	}
}

func (k *kit) parentID(username string) string {
	k.t.Helper()
	p, _, err := k.Store.ParentByUsername(context.Background(), username)
	if err != nil {
		k.t.Fatal(err)
	}
	return p.ID
}

func (k *kit) kidsOf(token string) []any {
	k.t.Helper()
	rec := k.do(http.MethodGet, "/v1/kids", token, nil)
	if rec.Code != http.StatusOK {
		k.t.Fatalf("list kids: %d %q", rec.Code, rec.Body.String())
	}
	return bodyMap(k.t, rec)["kids"].([]any)
}

func wantInvalid(t *testing.T, rec *httptest.ResponseRecorder, field string) {
	t.Helper()
	want := `{"error":"invalid","field":"` + field + `"}`
	if rec.Code != http.StatusUnprocessableEntity || strings.TrimSpace(rec.Body.String()) != want {
		t.Fatalf("got %d %q, want 422 %s", rec.Code, rec.Body.String(), want)
	}
}

func TestCreateKidMakesGuardian(t *testing.T) {
	k := newKit(t)
	ana := k.asParent("ana")
	rec := k.do(http.MethodPost, "/v1/kids", ana, kidBody(map[string]any{"name": "  " + testKidName + " ", "gestation_weeks": 36, "gestation_days": 4}))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %q", rec.Code, rec.Body.String())
	}
	got := bodyMap(t, rec)
	sameKeys(t, got, "id", "name", "birth_date", "sex", "gestation_weeks", "gestation_days", "time_zone", "growth_source", "vaccine_schema", "reminder_overdue_minutes", "role")
	id, _ := got["id"].(string)
	want := map[string]any{"id": id, "name": testKidName, "birth_date": "2026-03-04", "sex": "female",
		"gestation_weeks": float64(36), "gestation_days": float64(4), "time_zone": "Europe/Madrid", "growth_source": "who", "vaccine_schema": nil, "reminder_overdue_minutes": float64(60), "role": "guardian"}
	if id == "" || !reflect.DeepEqual(got, want) {
		t.Fatalf("created = %v", got)
	}
	if role, err := k.Store.RoleFor(context.Background(), id, k.parentID("ana")); err != nil || role != model.RoleGuardian {
		t.Errorf("RoleFor = %q, %v", role, err)
	}

	rec = k.do(http.MethodGet, "/v1/kids/"+id, ana, nil)
	if rec.Code != http.StatusOK || !reflect.DeepEqual(bodyMap(t, rec), want) {
		t.Errorf("GET = %d %q", rec.Code, rec.Body.String())
	}

	plain := k.createKid(ana, nil)
	rec = k.do(http.MethodGet, "/v1/kids/"+plain, ana, nil)
	if b := bodyMap(t, rec); b["gestation_weeks"] != nil || b["gestation_days"] != nil || len(b) != 11 {
		t.Errorf("without gestation = %v", b)
	}
}

func TestKidsForListsOnlyGranted(t *testing.T) {
	k := newKit(t)
	ana, bo, sam := k.asParent("ana"), k.asParent("bob"), k.asParent("sam")
	first := k.createKid(ana, map[string]any{"name": "First"})
	k.now = k.now.Add(time.Minute)
	shared := k.createKid(bo, map[string]any{"name": "Shared"})
	k.grant(bo, shared, "ana", "caregiver")
	k.now = k.now.Add(time.Minute)
	last := k.createKid(ana, map[string]any{"name": "Last"})
	k.createKid(bo, map[string]any{"name": "Bo only"})

	rows := k.kidsOf(ana)
	if len(rows) != 3 {
		t.Fatalf("ana's kids = %v", rows)
	}
	for i, want := range []struct{ id, role string }{{first, "guardian"}, {shared, "caregiver"}, {last, "guardian"}} {
		row := rows[i].(map[string]any)
		sameKeys(t, row, "id", "name", "birth_date", "sex", "gestation_weeks", "gestation_days", "time_zone", "growth_source", "vaccine_schema", "reminder_overdue_minutes", "role")
		if row["id"] != want.id || row["role"] != want.role {
			t.Errorf("row %d = %v, want %s %s", i, row, want.id, want.role)
		}
	}
	if len(k.kidsOf(bo)) != 2 {
		t.Errorf("bo's kids = %v", k.kidsOf(bo))
	}
	rec := k.do(http.MethodGet, "/v1/kids", sam, nil)
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != `{"kids":[]}` {
		t.Errorf("sam = %d %q", rec.Code, rec.Body.String())
	}
}

func TestKidValidation(t *testing.T) {
	k := newKit(t) // now is 2026-11-20 12:00 UTC
	ana := k.asParent("ana")
	cases := []struct {
		name  string
		over  map[string]any
		field string
	}{
		{"missing name", map[string]any{"name": nil}, "name"},
		{"blank name", map[string]any{"name": " \t "}, "name"},
		{"name of 65 code points", map[string]any{"name": strings.Repeat("é", 65)}, "name"},
		{"tomorrow", map[string]any{"birth_date": "2026-11-21"}, "birth_date"},
		{"tomorrow in the kid's zone", map[string]any{"birth_date": "2026-11-21", "time_zone": "America/Los_Angeles"}, "birth_date"},
		{"impossible date", map[string]any{"birth_date": "2025-02-30"}, "birth_date"},
		{"before 1900", map[string]any{"birth_date": "1899-12-31"}, "birth_date"},
		{"one-digit month", map[string]any{"birth_date": "2025-2-03"}, "birth_date"},
		{"no dashes", map[string]any{"birth_date": "20250203"}, "birth_date"},
		{"with a time", map[string]any{"birth_date": "2025-02-03T00:00:00Z"}, "birth_date"},
		{"missing date", map[string]any{"birth_date": nil}, "birth_date"},
		{"other sex", map[string]any{"sex": "other"}, "sex"},
		{"capitalised sex", map[string]any{"sex": "Female"}, "sex"},
		{"missing sex", map[string]any{"sex": nil}, "sex"},
		{"weeks 21", map[string]any{"gestation_weeks": 21, "gestation_days": 0}, "gestation_weeks"},
		{"weeks 45", map[string]any{"gestation_weeks": 45, "gestation_days": 0}, "gestation_weeks"},
		{"days without weeks", map[string]any{"gestation_days": 3}, "gestation_weeks"},
		{"weeks without days", map[string]any{"gestation_weeks": 38}, "gestation_days"},
		{"days 7", map[string]any{"gestation_weeks": 38, "gestation_days": 7}, "gestation_days"},
		{"days -1", map[string]any{"gestation_weeks": 38, "gestation_days": -1}, "gestation_days"},
		{"unknown zone", map[string]any{"time_zone": "Mars/Olympus"}, "time_zone"},
		{"empty zone", map[string]any{"time_zone": ""}, "time_zone"},
		{"Local zone", map[string]any{"time_zone": "Local"}, "time_zone"},
		{"missing zone", map[string]any{"time_zone": nil}, "time_zone"},
		{"path in zone", map[string]any{"time_zone": "../zoneinfo/UTC"}, "time_zone"},
		{"rooted zone", map[string]any{"time_zone": "/usr/share/zoneinfo/UTC"}, "time_zone"},
		{"first failing field wins", map[string]any{"name": "", "sex": "x", "time_zone": "Mars/Olympus"}, "name"},
		{"date before sex", map[string]any{"birth_date": "x", "sex": "x"}, "birth_date"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			wantInvalid(t, k.do(http.MethodPost, "/v1/kids", ana, kidBody(c.over)), c.field)
			if rows := k.kidsOf(ana); len(rows) != 0 {
				t.Fatalf("a kid was stored: %v", rows)
			}
		})
	}

	for _, good := range []map[string]any{
		{"birth_date": "1900-01-01"},
		{"birth_date": "2026-11-20"},
		{"birth_date": "2026-11-21", "time_zone": "Pacific/Kiritimati"}, // already the 21st there
		{"name": strings.Repeat("é", 64)},
		{"name": "A"},
		{"sex": "male", "time_zone": "UTC"},
		{"gestation_weeks": 22, "gestation_days": 0},
		{"gestation_weeks": 44, "gestation_days": 6},
		{"gestation_weeks": nil, "gestation_days": nil},
	} {
		if rec := k.do(http.MethodPost, "/v1/kids", ana, kidBody(good)); rec.Code != http.StatusCreated {
			t.Errorf("%v: %d %q, want 201", good, rec.Code, rec.Body.String())
		}
	}

	rec := k.do(http.MethodPost, "/v1/kids", ana, `{"name":"X","birth_date":"2026-01-01","sex":"male","time_zone":"UTC","colour":"red"}`)
	if rec.Code != http.StatusUnprocessableEntity || strings.TrimSpace(rec.Body.String()) != `{"error":"unknown_field","field":"colour"}` {
		t.Errorf("unknown field: %d %q", rec.Code, rec.Body.String())
	}
	wantErr(t, k.do(http.MethodPost, "/v1/kids", ana, `{"name":`), http.StatusBadRequest, "bad_request")
	wantErr(t, k.do(http.MethodPost, "/v1/kids", ana, kidBody(map[string]any{"gestation_weeks": "38"})), http.StatusBadRequest, "bad_request")
}

func TestGetKidRole(t *testing.T) {
	k := newKit(t)
	ana, cara := k.asParent("ana"), k.asParent("cara")
	kid := k.createKid(ana, nil)
	k.grant(ana, kid, "cara", "caregiver")
	rec := k.do(http.MethodGet, "/v1/kids/"+kid, cara, nil)
	if rec.Code != http.StatusOK || bodyMap(t, rec)["role"] != "caregiver" {
		t.Fatalf("caregiver GET: %d %q", rec.Code, rec.Body.String())
	}
}

func TestPatchKid(t *testing.T) {
	k := newKit(t)
	k.now = time.Date(2026, 11, 20, 5, 0, 0, 0, time.UTC) // still the 19th in Los Angeles
	ana := k.asParent("ana")
	kid := k.createKid(ana, map[string]any{"birth_date": "2026-11-20", "time_zone": "UTC", "gestation_weeks": 35, "gestation_days": 2})
	get := func() map[string]any {
		t.Helper()
		return bodyMap(t, k.do(http.MethodGet, "/v1/kids/"+kid, ana, nil))
	}

	rec := k.do(http.MethodPatch, "/v1/kids/"+kid, ana, map[string]any{"name": " Nova "})
	if rec.Code != http.StatusOK {
		t.Fatalf("patch name: %d %q", rec.Code, rec.Body.String())
	}
	want := map[string]any{"id": kid, "name": "Nova", "birth_date": "2026-11-20", "sex": "female",
		"gestation_weeks": float64(35), "gestation_days": float64(2), "time_zone": "UTC", "growth_source": "who", "vaccine_schema": nil, "reminder_overdue_minutes": float64(60), "role": "guardian"}
	if got := bodyMap(t, rec); !reflect.DeepEqual(got, want) {
		t.Errorf("patch response = %v", got)
	}
	if got := get(); !reflect.DeepEqual(got, want) {
		t.Errorf("after patch = %v", got)
	}

	// The merged record is validated: each refusal leaves the kid as it was.
	for _, c := range []struct {
		body  any
		field string
	}{
		{map[string]any{"time_zone": "America/Los_Angeles"}, "birth_date"},
		{map[string]any{"gestation_weeks": nil}, "gestation_weeks"},
		{map[string]any{"gestation_days": nil}, "gestation_days"},
		{map[string]any{"gestation_weeks": 50}, "gestation_weeks"},
		{map[string]any{"name": ""}, "name"},
		{map[string]any{"sex": "x", "time_zone": "Local"}, "sex"},
		{map[string]any{"birth_date": "2025-02-30"}, "birth_date"},
	} {
		wantInvalid(t, k.do(http.MethodPatch, "/v1/kids/"+kid, ana, c.body), c.field)
		if got := get(); !reflect.DeepEqual(got, want) {
			t.Fatalf("after refused %v: %v", c.body, got)
		}
	}

	rec = k.do(http.MethodPatch, "/v1/kids/"+kid, ana, map[string]any{"gestation_weeks": nil, "gestation_days": nil, "sex": "male"})
	if rec.Code != http.StatusOK {
		t.Fatalf("clear gestation: %d %q", rec.Code, rec.Body.String())
	}
	if got := get(); got["gestation_weeks"] != nil || got["gestation_days"] != nil || got["sex"] != "male" || got["name"] != "Nova" {
		t.Errorf("after clearing = %v", got)
	}
	wantInvalid(t, k.do(http.MethodPatch, "/v1/kids/"+kid, ana, map[string]any{"gestation_days": 1}), "gestation_weeks")

	k.now = k.now.Add(10 * time.Hour) // the 20th in Los Angeles now
	if rec := k.do(http.MethodPatch, "/v1/kids/"+kid, ana, map[string]any{"time_zone": "America/Los_Angeles"}); rec.Code != http.StatusOK {
		t.Errorf("zone change once valid: %d %q", rec.Code, rec.Body.String())
	}
	if rec := k.do(http.MethodPatch, "/v1/kids/"+kid, ana, map[string]any{}); rec.Code != http.StatusOK {
		t.Errorf("empty patch: %d %q", rec.Code, rec.Body.String())
	}
	rec = k.do(http.MethodPatch, "/v1/kids/"+kid, ana, map[string]any{"id": "other"})
	if rec.Code != http.StatusUnprocessableEntity || strings.TrimSpace(rec.Body.String()) != `{"error":"unknown_field","field":"id"}` {
		t.Errorf("patch id: %d %q", rec.Code, rec.Body.String())
	}
}

func TestReminderOverdueMinutes(t *testing.T) {
	k := newKit(t)
	ana, cara, eve := k.asParent("ana"), k.asParent("cara"), k.asParent("eve")
	kid := k.createKid(ana, nil)
	minutes := func(id, token string) any {
		t.Helper()
		rec := k.do(http.MethodGet, "/v1/kids/"+id, token, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET: %d %q", rec.Code, rec.Body.String())
		}
		return bodyMap(t, rec)["reminder_overdue_minutes"]
	}
	if got := minutes(kid, ana); got != float64(60) {
		t.Errorf("a new kid reads %v, want 60", got)
	}
	if got := minutes(k.createKid(ana, map[string]any{"name": "Fifteen", "reminder_overdue_minutes": 15}), ana); got != float64(15) {
		t.Errorf("POST with 15 reads %v", got)
	}
	for _, bad := range []any{4, 1441, "60", 60.5, true} { // null: raw body below (kidBody drops nil)
		wantInvalid(t, k.do(http.MethodPost, "/v1/kids", ana, kidBody(map[string]any{"reminder_overdue_minutes": bad, "name": "Bad"})), "reminder_overdue_minutes")
	}
	wantInvalid(t, k.do(http.MethodPost, "/v1/kids", ana, `{"name":"X","birth_date":"2026-03-04","sex":"female","time_zone":"UTC","reminder_overdue_minutes":null}`), "reminder_overdue_minutes")

	for _, n := range []int{5, 1440} {
		rec := k.do(http.MethodPatch, "/v1/kids/"+kid, ana, map[string]any{"reminder_overdue_minutes": n})
		if rec.Code != http.StatusOK || bodyMap(t, rec)["reminder_overdue_minutes"] != float64(n) || minutes(kid, ana) != float64(n) {
			t.Errorf("PATCH %d: %d %q", n, rec.Code, rec.Body.String())
		}
	}
	for _, raw := range []string{`4`, `1441`, `null`, `"60"`, `60.5`, `1e2`, `true`, `[]`} {
		wantInvalid(t, k.do(http.MethodPatch, "/v1/kids/"+kid, ana, `{"reminder_overdue_minutes":`+raw+`}`), "reminder_overdue_minutes")
		if got := minutes(kid, ana); got != float64(1440) {
			t.Fatalf("after refused %s: %v", raw, got)
		}
	}
	if rec := k.do(http.MethodPatch, "/v1/kids/"+kid, ana, map[string]any{"name": "Renamed"}); rec.Code != http.StatusOK || minutes(kid, ana) != float64(1440) {
		t.Errorf("a patch without the key: %d %q, minutes %v", rec.Code, rec.Body.String(), minutes(kid, ana))
	}

	k.grant(ana, kid, "cara", "caregiver")
	rec := k.do(http.MethodPatch, "/v1/kids/"+kid, cara, map[string]any{"reminder_overdue_minutes": 30})
	if rec.Code != http.StatusForbidden || minutes(kid, ana) != float64(1440) {
		t.Errorf("caregiver PATCH: %d %q", rec.Code, rec.Body.String())
	}
	if rec := k.do(http.MethodPatch, "/v1/kids/"+kid, eve, map[string]any{"reminder_overdue_minutes": 30}); rec.Code != http.StatusNotFound {
		t.Errorf("no-grant PATCH: %d %q", rec.Code, rec.Body.String())
	}
}

func TestDeleteKid(t *testing.T) {
	k := newKit(t)
	ana, bo := k.asParent("ana"), k.asParent("bob")
	kid := k.createKid(ana, nil)
	other := k.createKid(ana, map[string]any{"name": "Other"})
	k.grant(ana, kid, "bob", "caregiver")
	k.grant(ana, other, "bob", "guardian")

	for _, body := range []any{
		map[string]string{"confirm_name": "lía wren"},
		map[string]string{"confirm_name": testKidName + " "},
		map[string]string{},
	} {
		wantInvalid(t, k.do(http.MethodDelete, "/v1/kids/"+kid, ana, body), "confirm_name")
	}
	if len(k.kidsOf(ana)) != 2 {
		t.Fatal("a refused delete removed a kid")
	}

	rec := k.do(http.MethodDelete, "/v1/kids/"+kid, ana, map[string]string{"confirm_name": testKidName})
	if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Fatalf("delete: %d %q", rec.Code, rec.Body.String())
	}
	wantErr(t, k.do(http.MethodGet, "/v1/kids/"+kid, ana, nil), http.StatusNotFound, "not_found")
	if _, err := k.Store.RoleFor(context.Background(), kid, k.parentID("bob")); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("bo's grant survived: %v", err)
	}
	if grants, err := k.Store.AccessList(context.Background(), kid); err != nil || len(grants) != 0 {
		t.Errorf("grants of the deleted kid = %v, %v", grants, err)
	}
	for _, token := range []string{ana, bo} {
		rows := k.kidsOf(token)
		if len(rows) != 1 || rows[0].(map[string]any)["id"] != other {
			t.Errorf("remaining kids = %v", rows)
		}
	}
	if grants, err := k.Store.AccessList(context.Background(), other); err != nil || len(grants) != 2 {
		t.Errorf("other kid's grants = %v, %v", grants, err)
	}
}

func TestNoGrantIs404(t *testing.T) {
	k := newKit(t)
	ana, sam := k.asParent("ana"), k.asParent("sam")
	k.asParent("bob")
	kid := k.createKid(ana, nil)
	unknown := model.NewID()
	samID := k.parentID("sam")
	requests := []struct {
		method, suffix string
		body           any
	}{
		{http.MethodGet, "", nil},
		{http.MethodPatch, "", map[string]any{"name": "X"}},
		{http.MethodDelete, "", map[string]string{"confirm_name": testKidName}},
		{http.MethodGet, "/access", nil},
		{http.MethodPut, "/access/bob", map[string]string{"role": "guardian"}},
		{http.MethodDelete, "/access/" + samID, nil},
		{http.MethodDelete, "/access/" + k.parentID("ana"), nil},
	}
	for _, rq := range requests {
		noGrant := k.do(rq.method, "/v1/kids/"+kid+rq.suffix, sam, rq.body)
		noKid := k.do(rq.method, "/v1/kids/"+unknown+rq.suffix, sam, rq.body)
		wantErr(t, noGrant, http.StatusNotFound, "not_found")
		if noGrant.Code != noKid.Code || !reflect.DeepEqual(noGrant.Header(), noKid.Header()) || !bytesEqual(noGrant.Body.Bytes(), noKid.Body.Bytes()) {
			t.Errorf("%s %s: no grant %d %v %q, unknown kid %d %v %q", rq.method, rq.suffix,
				noGrant.Code, noGrant.Header(), noGrant.Body.String(), noKid.Code, noKid.Header(), noKid.Body.String())
		}
	}
	if len(k.kidsOf(ana)) != 1 {
		t.Error("the kid changed")
	}
}

func bytesEqual(a, b []byte) bool { return string(a) == string(b) }

func TestAdminCannotReachKids(t *testing.T) {
	k := newKit(t)
	admin, ana := k.asAdmin(), k.asParent("ana")
	kid := k.createKid(ana, nil)
	wantErr(t, k.do(http.MethodGet, "/v1/kids", admin, nil), http.StatusForbidden, "forbidden")
	wantErr(t, k.do(http.MethodPost, "/v1/kids", admin, kidBody(nil)), http.StatusForbidden, "forbidden")
	wantErr(t, k.do(http.MethodGet, "/v1/kids/"+kid, admin, nil), http.StatusForbidden, "forbidden")
	wantErr(t, k.do(http.MethodGet, "/v1/kids/"+kid+"/access", admin, nil), http.StatusForbidden, "forbidden")
}

func TestKidNamesNeverReachTheLog(t *testing.T) {
	k := newKit(t)
	ana := k.asParent("ana")
	kid := k.createKid(ana, nil)
	k.do(http.MethodGet, "/v1/kids/"+kid, ana, nil)
	k.do(http.MethodGet, "/v1/kids", ana, nil)
	k.do(http.MethodPatch, "/v1/kids/"+kid, ana, map[string]string{"name": "Second Secretname"})
	k.do(http.MethodPatch, "/v1/kids/"+kid, ana, map[string]any{"name": "Third Secretname", "sex": "x"})
	k.do(http.MethodDelete, "/v1/kids/"+kid, ana, map[string]string{"confirm_name": "Wrong Secretname"})
	k.do(http.MethodDelete, "/v1/kids/"+kid, ana, map[string]string{"confirm_name": "Second Secretname"})
	logs := k.Log.String()
	for _, secret := range []string{"Lía", "Wren", "Secretname", "ana", "Europe/Madrid", "2026-03-04"} {
		if strings.Contains(logs, secret) {
			t.Errorf("log contains %q:\n%s", secret, logs)
		}
	}
	if strings.Count(logs, "\n") != 7 {
		t.Errorf("want one line per request:\n%s", logs)
	}
}

func TestKidGrowthSource(t *testing.T) {
	k := newKit(t)
	ana, cara := k.asParent("ana"), k.asParent("cara")
	source := func(kid string) any {
		t.Helper()
		return bodyMap(t, k.do(http.MethodGet, "/v1/kids/"+kid, ana, nil))["growth_source"]
	}

	if got := source(k.createKid(ana, nil)); got != "who" {
		t.Errorf("created without growth_source = %v, want who", got)
	}
	kid := k.createKid(ana, map[string]any{"growth_source": "cdc"})
	if got := source(kid); got != "cdc" {
		t.Fatalf("created with cdc = %v", got)
	}
	for _, row := range k.kidsOf(ana) {
		if row := row.(map[string]any); row["id"] == kid && row["growth_source"] != "cdc" {
			t.Errorf("list row = %v", row)
		}
	}

	// An omitted key keeps the stored value, across a PATCH of another field too.
	for _, body := range []map[string]any{{}, {"name": "Nova"}} {
		rec := k.do(http.MethodPatch, "/v1/kids/"+kid, ana, body)
		if rec.Code != http.StatusOK || bodyMap(t, rec)["growth_source"] != "cdc" || source(kid) != "cdc" {
			t.Fatalf("patch %v: %d %q, stored %v", body, rec.Code, rec.Body.String(), source(kid))
		}
	}

	for _, bad := range []string{"uk-who", "", "WHO"} {
		wantInvalid(t, k.do(http.MethodPatch, "/v1/kids/"+kid, ana, map[string]any{"growth_source": bad}), "growth_source")
		wantInvalid(t, k.do(http.MethodPost, "/v1/kids", ana, kidBody(map[string]any{"growth_source": bad})), "growth_source")
		if got := source(kid); got != "cdc" {
			t.Fatalf("after refused %q: %v", bad, got)
		}
	}

	k.grant(ana, kid, "cara", "caregiver")
	wantErr(t, k.do(http.MethodPatch, "/v1/kids/"+kid, cara, map[string]any{"growth_source": "who"}), http.StatusForbidden, "forbidden")
	if got := source(kid); got != "cdc" {
		t.Errorf("a caregiver's refused patch changed it: %v", got)
	}

	rec := k.do(http.MethodPatch, "/v1/kids/"+kid, ana, map[string]any{"growth_source": "who"})
	if rec.Code != http.StatusOK || bodyMap(t, rec)["growth_source"] != "who" || source(kid) != "who" {
		t.Errorf("patch to who: %d %q, stored %v", rec.Code, rec.Body.String(), source(kid))
	}
}

func TestKidCounts(t *testing.T) {
	k := newKit(t)
	ana, bo, cy := k.asParent("ana"), k.asParent("bob"), k.asParent("cy")
	kid := k.createKid(ana, nil)
	k.grant(ana, kid, "bob", "caregiver")
	k.createDiaper(ana, kid, nil)
	k.createDiaper(bo, kid, map[string]any{"kind": "dirty"})

	rec := k.do(http.MethodGet, "/v1/kids/"+kid+"/counts", ana, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("guardian: %d %q", rec.Code, rec.Body.String())
	}
	m := bodyMap(t, rec)
	sameKeys(t, m, "entries")
	if m["entries"] != float64(2) {
		t.Errorf("entries = %v, want 2", m["entries"])
	}
	wantErr(t, k.do(http.MethodGet, "/v1/kids/"+kid+"/counts", bo, nil), http.StatusForbidden, "forbidden")
	stranger := k.do(http.MethodGet, "/v1/kids/"+kid+"/counts", cy, nil)
	unknown := k.do(http.MethodGet, "/v1/kids/no-such-kid/counts", cy, nil)
	if stranger.Code != http.StatusNotFound || stranger.Body.String() != unknown.Body.String() {
		t.Errorf("no grant: %d %q, unknown kid: %d %q; want identical 404s", stranger.Code, stranger.Body.String(), unknown.Code, unknown.Body.String())
	}
}
