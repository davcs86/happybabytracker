package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/davcs86/happybabytracker/api/internal/model"
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

// staleCall runs h as if the middleware had admitted username with role on kid before the state changed.
func (k *kit) staleCall(h http.HandlerFunc, method, path, username, kid string, role model.Role, pathValues map[string]string, body any) *httptest.ResponseRecorder {
	k.t.Helper()
	r := k.newReq(method, path, body)
	ctx := context.WithValue(r.Context(), principalKey, model.Principal{Kind: kindParent, ParentID: k.parentID(username)})
	if role != "" {
		ctx = context.WithValue(ctx, kidRoleKey, role)
	}
	r = r.WithContext(ctx)
	r.SetPathValue("kid", kid)
	for key, v := range pathValues {
		r.SetPathValue(key, v)
	}
	rec := httptest.NewRecorder()
	h(rec, r)
	return rec
}

func sameResponse(t *testing.T, what string, got, want *httptest.ResponseRecorder) {
	t.Helper()
	if got.Code != want.Code || !reflect.DeepEqual(got.Header(), want.Header()) || got.Body.String() != want.Body.String() {
		t.Errorf("%s: %d %v %q, want %d %v %q", what, got.Code, got.Header(), got.Body.String(), want.Code, want.Header(), want.Body.String())
	}
}

func TestCreateKidByCreatorDisabledMidRequest(t *testing.T) {
	k := newKit(t)
	admin := k.asAdmin()
	k.asParent("ana")
	if rec := k.disable(admin, "ana"); rec.Code != http.StatusNoContent {
		t.Fatalf("disable: %d", rec.Code)
	}
	d := k.deps()
	rec := k.staleCall(d.createKid, http.MethodPost, "/v1/kids", "ana", "", "", nil, kidBody(nil))
	wantErr(t, rec, http.StatusUnauthorized, "no_session")
	if kids, err := k.Store.KidsFor(context.Background(), k.parentID("ana")); err != nil || len(kids) != 0 {
		t.Errorf("kids = %v, %v", kids, err)
	}
}

func TestStaleAuthorizationIsRefused(t *testing.T) {
	k := newKit(t)
	ana, bob, sam := k.asParent("ana"), k.asParent("bob"), k.asParent("sam")
	k.asParent("cal")
	kid := k.createKid(ana, nil)
	k.grant(ana, kid, "bob", "guardian")
	d := k.deps()
	base := "/v1/kids/" + kid
	// What the middleware answers a parent with no grant: every stale refusal must match it byte for byte.
	noGrant := k.do(http.MethodGet, base, sam, nil)
	k.grant(ana, kid, "sam", "caregiver")
	forbidden := k.do(http.MethodPatch, base, sam, map[string]string{"name": "X"})
	wantErr(t, forbidden, http.StatusForbidden, "forbidden")

	calls := func(role model.Role) map[string]*httptest.ResponseRecorder {
		return map[string]*httptest.ResponseRecorder{
			"PATCH kid":      k.staleCall(d.updateKid, http.MethodPatch, base, "ana", kid, role, nil, map[string]string{"name": "Changed"}),
			"DELETE kid":     k.staleCall(d.deleteKid, http.MethodDelete, base, "ana", kid, role, nil, map[string]string{"confirm_name": testKidName}),
			"PUT self":       k.staleCall(d.setAccess, http.MethodPut, base+"/access/ana", "ana", kid, role, map[string]string{"username": "ana"}, map[string]string{"role": "guardian"}),
			"PUT other":      k.staleCall(d.setAccess, http.MethodPut, base+"/access/cal", "ana", kid, role, map[string]string{"username": "cal"}, map[string]string{"role": "guardian"}),
			"DELETE another": k.staleCall(d.removeAccess, http.MethodDelete, base+"/access/x", "ana", kid, role, map[string]string{"parent": k.parentID("sam")}, nil),
		}
	}

	// Downgraded to caregiver after the middleware saw a guardian.
	k.grant(bob, kid, "ana", "caregiver")
	for what, rec := range calls(model.RoleGuardian) {
		sameResponse(t, "downgraded "+what, rec, forbidden)
	}
	// Revoked after the middleware saw a guardian.
	if rec := k.do(http.MethodDelete, base+"/access/"+k.parentID("ana"), bob, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("revoke: %d", rec.Code)
	}
	for what, rec := range calls(model.RoleGuardian) {
		sameResponse(t, "revoked "+what, rec, noGrant)
	}

	grants := bodyMap(t, k.do(http.MethodGet, base+"/access", bob, nil))["grants"].([]any)
	if len(grants) != 2 {
		t.Errorf("grants changed: %v", grants)
	}
	if got := bodyMap(t, k.do(http.MethodGet, base, bob, nil)); got["name"] != testKidName {
		t.Errorf("kid changed: %v", got)
	}
	// The kid deleted under a guardian's PUT is the same 404, not a 500.
	if rec := k.do(http.MethodDelete, base, bob, map[string]string{"confirm_name": testKidName}); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d", rec.Code)
	}
	rec := k.staleCall(d.setAccess, http.MethodPut, base+"/access/cal", "bob", kid, model.RoleGuardian, map[string]string{"username": "cal"}, map[string]string{"role": "caregiver"})
	sameResponse(t, "PUT on a deleted kid", rec, noGrant)
}

func TestDisabledActorMidRequestGetsTheNoGrant404(t *testing.T) {
	k := newKit(t)
	admin, ana, sam := k.asAdmin(), k.asParent("ana"), k.asParent("sam")
	k.asParent("bob")
	k.asParent("cal")
	kid := k.createKid(ana, nil)
	k.grant(ana, kid, "bob", "guardian")
	base := "/v1/kids/" + kid
	noGrant := k.do(http.MethodGet, base, sam, nil)
	// The middleware admitted ana as a guardian; the admin disables her before her write runs.
	if rec := k.disable(admin, "ana"); rec.Code != http.StatusNoContent {
		t.Fatalf("disable: %d %q", rec.Code, rec.Body.String())
	}
	d := k.deps()
	for what, rec := range map[string]*httptest.ResponseRecorder{
		"DELETE kid":   k.staleCall(d.deleteKid, http.MethodDelete, base, "ana", kid, model.RoleGuardian, nil, map[string]string{"confirm_name": testKidName}),
		"PUT access":   k.staleCall(d.setAccess, http.MethodPut, base+"/access/cal", "ana", kid, model.RoleGuardian, map[string]string{"username": "cal"}, map[string]string{"role": "guardian"}),
		"PATCH kid":    k.staleCall(d.updateKid, http.MethodPatch, base, "ana", kid, model.RoleGuardian, nil, map[string]string{"name": "Changed"}),
		"DELETE other": k.staleCall(d.removeAccess, http.MethodDelete, base+"/access/x", "ana", kid, model.RoleGuardian, map[string]string{"parent": k.parentID("bob")}, nil),
		"DELETE self":  k.staleCall(d.removeAccess, http.MethodDelete, base+"/access/x", "ana", kid, model.RoleGuardian, map[string]string{"parent": k.parentID("ana")}, nil),
	} {
		sameResponse(t, what, rec, noGrant)
	}
	bob := k.asParent("bob")
	if got := bodyMap(t, k.do(http.MethodGet, base, bob, nil)); got["name"] != testKidName {
		t.Errorf("kid changed: %v", got)
	}
	if grants := bodyMap(t, k.do(http.MethodGet, base+"/access", bob, nil))["grants"].([]any); len(grants) != 2 {
		t.Errorf("grants changed: %v", grants)
	}
}
