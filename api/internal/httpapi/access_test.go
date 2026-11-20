package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func (k *kit) disable(admin, username string) *httptest.ResponseRecorder {
	k.t.Helper()
	return k.do(http.MethodPost, "/v1/admin/parents/"+k.parentID(username)+"/disable", admin, nil)
}

func TestAccessListShape(t *testing.T) {
	k := newKit(t)
	ana := k.asParent("ana")
	k.asParent("cara")
	k.asParent("bob")
	kid := k.createKid(ana, nil)
	k.grant(ana, kid, "cara", "caregiver")
	k.grant(ana, kid, "bob", "guardian")
	rec := k.do(http.MethodGet, "/v1/kids/"+kid+"/access", ana, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list: %d %q", rec.Code, rec.Body.String())
	}
	rows := bodyMap(t, rec)["grants"].([]any)
	var order []string
	for _, r := range rows {
		row := r.(map[string]any)
		sameKeys(t, row, "parent_id", "username", "display_name", "role")
		if row["parent_id"] != k.parentID(row["username"].(string)) || row["display_name"] != "Name of "+row["username"].(string) {
			t.Errorf("row = %v", row)
		}
		order = append(order, row["username"].(string)+":"+row["role"].(string))
	}
	if strings.Join(order, " ") != "ana:guardian bob:guardian cara:caregiver" {
		t.Errorf("order = %v", order)
	}
}

func TestCaregiverLimits(t *testing.T) {
	k := newKit(t)
	ana, cara, cat := k.asParent("ana"), k.asParent("cara"), k.asParent("cat")
	k.asParent("bob")
	kid := k.createKid(ana, nil)
	k.grant(ana, kid, "cara", "caregiver")
	k.grant(ana, kid, "cat", "caregiver")

	if rec := k.do(http.MethodGet, "/v1/kids/"+kid, cara, nil); rec.Code != http.StatusOK {
		t.Errorf("caregiver GET: %d", rec.Code)
	}
	for _, rq := range []struct {
		method, path string
		body         any
	}{
		{http.MethodPatch, "", map[string]string{"name": "X"}},
		{http.MethodDelete, "", map[string]string{"confirm_name": testKidName}},
		{http.MethodGet, "/access", nil},
		{http.MethodPut, "/access/bob", map[string]string{"role": "caregiver"}},
		{http.MethodPut, "/access/cara", map[string]string{"role": "guardian"}},
		{http.MethodDelete, "/access/" + k.parentID("cat"), nil},
		{http.MethodDelete, "/access/" + k.parentID("ana"), nil},
	} {
		wantErr(t, k.do(rq.method, "/v1/kids/"+kid+rq.path, cara, rq.body), http.StatusForbidden, "forbidden")
	}
	if rec := k.do(http.MethodDelete, "/v1/kids/"+kid+"/access/"+k.parentID("cara"), cara, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("caregiver leaves: %d %q", rec.Code, rec.Body.String())
	}
	wantErr(t, k.do(http.MethodGet, "/v1/kids/"+kid, cara, nil), http.StatusNotFound, "not_found")
	if rec := k.do(http.MethodGet, "/v1/kids/"+kid, cat, nil); rec.Code != http.StatusOK {
		t.Errorf("the other caregiver lost access: %d", rec.Code)
	}
	// A guardian removes anyone; a parent with no grant on the kid is 404.
	if rec := k.do(http.MethodDelete, "/v1/kids/"+kid+"/access/"+k.parentID("cat"), ana, nil); rec.Code != http.StatusNoContent {
		t.Errorf("guardian removes a caregiver: %d", rec.Code)
	}
	wantErr(t, k.do(http.MethodDelete, "/v1/kids/"+kid+"/access/"+k.parentID("bob"), ana, nil), http.StatusNotFound, "not_found")
	wantErr(t, k.do(http.MethodDelete, "/v1/kids/"+kid+"/access/no-such-parent", ana, nil), http.StatusNotFound, "not_found")
}

func TestGrantByUsername(t *testing.T) {
	k := newKit(t)
	admin, ana, bo := k.asAdmin(), k.asParent("ana"), k.asParent("bob")
	k.asParent("off")
	kid := k.createKid(ana, nil)

	rec := k.do(http.MethodPut, "/v1/kids/"+kid+"/access/%20Bob%20", ana, map[string]string{"role": "caregiver"})
	if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Fatalf("grant \" Bob \": %d %q", rec.Code, rec.Body.String())
	}
	if rows := k.kidsOf(bo); len(rows) != 1 || rows[0].(map[string]any)["role"] != "caregiver" {
		t.Errorf("bo's kids = %v", rows)
	}
	k.grant(ana, kid, "BOB", "guardian")
	if rows := k.kidsOf(bo); rows[0].(map[string]any)["role"] != "guardian" {
		t.Errorf("bo after upgrade = %v", rows)
	}

	if rec := k.disable(admin, "off"); rec.Code != http.StatusNoContent {
		t.Fatalf("disable: %d", rec.Code)
	}
	unknown := k.do(http.MethodPut, "/v1/kids/"+kid+"/access/nobody", ana, map[string]string{"role": "caregiver"})
	disabled := k.do(http.MethodPut, "/v1/kids/"+kid+"/access/off", ana, map[string]string{"role": "caregiver"})
	invalid := k.do(http.MethodPut, "/v1/kids/"+kid+"/access/no%20such%20name", ana, map[string]string{"role": "caregiver"})
	admins := k.do(http.MethodPut, "/v1/kids/"+kid+"/access/"+testAdminUser, ana, map[string]string{"role": "guardian"})
	for _, rec := range []*httptest.ResponseRecorder{unknown, disabled, invalid, admins} {
		wantErr(t, rec, http.StatusNotFound, "parent_not_found")
	}

	for _, body := range []any{map[string]string{"role": "admin"}, map[string]string{"role": "Guardian"}, map[string]string{}} {
		wantInvalid(t, k.do(http.MethodPut, "/v1/kids/"+kid+"/access/bob", ana, body), "role")
	}
	if grants := k.do(http.MethodGet, "/v1/kids/"+kid+"/access", ana, nil); len(bodyMap(t, grants)["grants"].([]any)) != 2 {
		t.Errorf("grants = %s", grants.Body.String())
	}
}

func TestLastGuardian(t *testing.T) {
	k := newKit(t)
	ana, bo := k.asParent("ana"), k.asParent("bob")
	kid := k.createKid(ana, nil)
	self := "/v1/kids/" + kid + "/access/"

	wantErr(t, k.do(http.MethodPut, self+"ana", ana, map[string]string{"role": "caregiver"}), http.StatusConflict, "last_guardian")
	wantErr(t, k.do(http.MethodDelete, self+k.parentID("ana"), ana, nil), http.StatusConflict, "last_guardian")
	if rows := k.kidsOf(ana); len(rows) != 1 || rows[0].(map[string]any)["role"] != "guardian" {
		t.Fatalf("refusals changed the grant: %v", rows)
	}

	k.grant(ana, kid, "bob", "guardian")
	if rec := k.do(http.MethodPut, self+"ana", ana, map[string]string{"role": "caregiver"}); rec.Code != http.StatusNoContent {
		t.Fatalf("downgrade with a co-guardian: %d %q", rec.Code, rec.Body.String())
	}
	k.grant(bo, kid, "ana", "guardian")
	if rec := k.do(http.MethodDelete, self+k.parentID("ana"), ana, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("leave with a co-guardian: %d %q", rec.Code, rec.Body.String())
	}
	wantErr(t, k.do(http.MethodDelete, self+k.parentID("bob"), bo, nil), http.StatusConflict, "last_guardian")
}

func TestLastActiveGuardian(t *testing.T) {
	k := newKit(t)
	admin, ana, bo := k.asAdmin(), k.asParent("ana"), k.asParent("bob")
	kid := k.createKid(ana, nil)
	k.grant(ana, kid, "bob", "guardian")

	if rec := k.disable(admin, "ana"); rec.Code != http.StatusNoContent {
		t.Fatalf("disable ana: %d %q", rec.Code, rec.Body.String())
	}
	wantErr(t, k.do(http.MethodDelete, "/v1/kids/"+kid+"/access/"+k.parentID("bob"), bo, nil), http.StatusConflict, "last_guardian")
	wantErr(t, k.do(http.MethodPut, "/v1/kids/"+kid+"/access/bob", bo, map[string]string{"role": "caregiver"}), http.StatusConflict, "last_guardian")

	rec := k.disable(admin, "bob")
	if rec.Code != http.StatusConflict || rec.Body.String() != `{"error":"sole_guardian"}`+"\n" {
		t.Fatalf("disable the sole active guardian: %d %q", rec.Code, rec.Body.String())
	}
	if rows := k.kidsOf(bo); len(rows) != 1 {
		t.Errorf("bo lost the kid or the session: %v", rows)
	}
	if rec := k.do(http.MethodPost, "/v1/admin/parents/"+k.parentID("ana")+"/enable", admin, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("enable ana: %d", rec.Code)
	}
	if rec := k.disable(admin, "bob"); rec.Code != http.StatusNoContent {
		t.Errorf("disable bo once ana is back: %d %q", rec.Code, rec.Body.String())
	}
	wantErr(t, k.disable(admin, "ana"), http.StatusConflict, "sole_guardian")
}

func TestDisabledParentGrantStays(t *testing.T) {
	k := newKit(t)
	admin, ana, bo := k.asAdmin(), k.asParent("ana"), k.asParent("bob")
	kid := k.createKid(ana, nil)
	k.grant(ana, kid, "bob", "guardian")
	if rec := k.disable(admin, "bob"); rec.Code != http.StatusNoContent {
		t.Fatalf("disable: %d", rec.Code)
	}
	wantErr(t, k.do(http.MethodGet, "/v1/kids", bo, nil), http.StatusUnauthorized, "no_session")
	grants := bodyMap(t, k.do(http.MethodGet, "/v1/kids/"+kid+"/access", ana, nil))["grants"].([]any)
	if len(grants) != 2 {
		t.Fatalf("the disabled parent's grant went away: %v", grants)
	}
	if rec := k.do(http.MethodPost, "/v1/admin/parents/"+k.parentID("bob")+"/enable", admin, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("enable: %d", rec.Code)
	}
	rows := k.kidsOf(k.asParent("bob"))
	if len(rows) != 1 || rows[0].(map[string]any)["id"] != kid || rows[0].(map[string]any)["role"] != "guardian" {
		t.Errorf("re-enabled parent's kids = %v", rows)
	}
}
