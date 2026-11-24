package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strings"
	"testing"
)

var kitTopicRE = regexp.MustCompile(`^hbt-test-[0-9A-Za-z]{22}$`)

func ntfyOptinPath(kid string) string { return "/v1/kids/" + kid + "/ntfy" }

func (k *kit) ntfyState(token string) map[string]any {
	k.t.Helper()
	rec := k.do(http.MethodGet, "/v1/me/ntfy", token, nil)
	if rec.Code != http.StatusOK {
		k.t.Fatalf("ntfy state: %d %q", rec.Code, rec.Body.String())
	}
	return bodyMap(k.t, rec)
}

// mustNtfy turns ntfy on for the token's parent and returns the topic.
func (k *kit) mustNtfy(token string) string {
	k.t.Helper()
	rec := k.do(http.MethodPut, "/v1/me/ntfy", token, nil)
	if rec.Code != http.StatusCreated {
		k.t.Fatalf("ntfy on: %d %q", rec.Code, rec.Body.String())
	}
	return bodyMap(k.t, rec)["topic"].(string)
}

func ntfyKids(m map[string]any) []string {
	var out []string
	for _, v := range m["kids"].([]any) {
		out = append(out, v.(string))
	}
	return out
}

func TestNtfyStateAndTurnOn(t *testing.T) {
	k := newKit(t)
	ana := k.asParent("ana")
	got := k.ntfyState(ana)
	sameKeys(t, got, "enabled", "has_topic", "server", "topic", "kids")
	if got["enabled"] != true || got["has_topic"] != false || got["server"] != testNtfyURL || got["topic"] != nil || len(got["kids"].([]any)) != 0 {
		t.Fatalf("before on: %v", got)
	}
	rec := k.do(http.MethodPut, "/v1/me/ntfy", ana, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("on: %d %q", rec.Code, rec.Body.String())
	}
	on := bodyMap(t, rec)
	sameKeys(t, on, "enabled", "has_topic", "server", "topic", "kids")
	topic, _ := on["topic"].(string)
	if on["has_topic"] != true {
		t.Fatalf("on: has_topic %v", on["has_topic"])
	}
	if !kitTopicRE.MatchString(topic) {
		t.Fatalf("topic %q is not <prefix>-<22 base62>", topic)
	}
	again := k.do(http.MethodPut, "/v1/me/ntfy", ana, nil)
	if again.Code != http.StatusOK || bodyMap(t, again)["topic"] != topic {
		t.Fatalf("on again: %d %q", again.Code, again.Body.String())
	}
	if got := k.ntfyState(ana); got["topic"] != topic || got["has_topic"] != true {
		t.Fatalf("state after on: %v", got)
	}
}

func TestNtfyServerPrefersThePublicURL(t *testing.T) {
	k := newKit(t)
	k.Cfg.Ntfy.PublicURL = "https://ntfy.example.org"
	k.Handler = NewHandler(k.deps())
	if got := k.ntfyState(k.asParent("ana")); got["server"] != "https://ntfy.example.org" {
		t.Fatalf("server = %v", got["server"])
	}
}

func TestNtfyRegenerateKeepsOptins(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	wantErr(t, k.do(http.MethodPost, "/v1/me/ntfy/regenerate", ana, nil), http.StatusNotFound, "not_found")
	first := k.mustNtfy(ana)
	if rec := k.do(http.MethodPut, ntfyOptinPath(kid), ana, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("opt in: %d %q", rec.Code, rec.Body.String())
	}
	rec := k.do(http.MethodPost, "/v1/me/ntfy/regenerate", ana, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("regenerate: %d %q", rec.Code, rec.Body.String())
	}
	got := bodyMap(t, rec)
	if got["topic"] == first || !kitTopicRE.MatchString(got["topic"].(string)) {
		t.Fatalf("regenerate kept or broke the topic")
	}
	if !slices.Equal(ntfyKids(got), []string{kid}) {
		t.Fatalf("kids after regenerate = %v", got["kids"])
	}
	if strings.Contains(k.ntfyState(ana)["topic"].(string), first) {
		t.Fatal("old topic still shown")
	}
}

func TestNtfyOffDeletesTopicAndOptins(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	k.mustNtfy(ana)
	if rec := k.do(http.MethodPut, ntfyOptinPath(kid), ana, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("opt in: %d", rec.Code)
	}
	if rec := k.do(http.MethodDelete, "/v1/me/ntfy", ana, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("off: %d %q", rec.Code, rec.Body.String())
	}
	wantErr(t, k.do(http.MethodDelete, "/v1/me/ntfy", ana, nil), http.StatusNotFound, "not_found")
	got := k.ntfyState(ana)
	if got["topic"] != nil || len(got["kids"].([]any)) != 0 {
		t.Fatalf("after off: %v", got)
	}
	k.mustNtfy(ana)
	if kids := ntfyKids(k.ntfyState(ana)); len(kids) != 0 {
		t.Fatalf("a new topic brought back the opt-in: %v", kids)
	}
}

func TestNtfyOptins(t *testing.T) {
	k := newKit(t)
	kid, ana, bob := k.trackerKid()
	other := k.createKid(k.asParent("cid"), nil)
	k.mustNtfy(ana)

	unknownKid := k.do(http.MethodPut, ntfyOptinPath("0190a1b2-0000-7000-8000-000000000000"), ana, nil)
	wantErr(t, unknownKid, http.StatusNotFound, "not_found")
	for name, rec := range map[string]*httptest.ResponseRecorder{
		"kid without a grant":         k.do(http.MethodPut, ntfyOptinPath(other), ana, nil),
		"clear kid without a grant":   k.do(http.MethodDelete, ntfyOptinPath(other), ana, nil),
		"unknown kid, clear":          k.do(http.MethodDelete, ntfyOptinPath("0190a1b2-0000-7000-8000-000000000000"), ana, nil),
		"grant but no topic":          k.do(http.MethodPut, ntfyOptinPath(kid), bob, nil),
		"clear, grant but no topic":   k.do(http.MethodDelete, ntfyOptinPath(kid), bob, nil),
		"unknown kid (the reference)": unknownKid,
	} {
		sameResponse(t, name, rec, unknownKid)
	}
	for range 2 {
		if rec := k.do(http.MethodPut, ntfyOptinPath(kid), ana, nil); rec.Code != http.StatusNoContent {
			t.Fatalf("opt in: %d %q", rec.Code, rec.Body.String())
		}
	}
	k.mustNtfy(bob)
	if rec := k.do(http.MethodPut, ntfyOptinPath(kid), bob, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("caregiver opt in: %d %q", rec.Code, rec.Body.String())
	}
	if kids := ntfyKids(k.ntfyState(ana)); !slices.Equal(kids, []string{kid}) {
		t.Fatalf("kids = %v", kids)
	}
	for range 2 {
		if rec := k.do(http.MethodDelete, ntfyOptinPath(kid), ana, nil); rec.Code != http.StatusNoContent {
			t.Fatalf("opt out: %d %q", rec.Code, rec.Body.String())
		}
	}
	if kids := ntfyKids(k.ntfyState(ana)); len(kids) != 0 {
		t.Fatalf("after opt out kids = %v", kids)
	}
	if kids := ntfyKids(k.ntfyState(bob)); !slices.Equal(kids, []string{kid}) {
		t.Fatalf("ana's opt-out touched bob's: %v", kids)
	}
}

func TestNtfyTopicOfAnotherParent(t *testing.T) {
	k := newKit(t)
	kid, ana, bob := k.trackerKid()
	topic := k.mustNtfy(ana)
	if rec := k.do(http.MethodPut, ntfyOptinPath(kid), ana, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("opt in: %d", rec.Code)
	}
	if got := k.ntfyState(bob); got["topic"] != nil || len(got["kids"].([]any)) != 0 {
		t.Fatalf("bob sees: %v", got)
	}
	missing := k.do(http.MethodDelete, "/v1/me/ntfy", bob, nil)
	wantErr(t, missing, http.StatusNotFound, "not_found")
	sameResponse(t, "regenerate without a topic", k.do(http.MethodPost, "/v1/me/ntfy/regenerate", bob, nil), missing)
	bobTopic := k.mustNtfy(bob)
	if bobTopic == topic {
		t.Fatal("two parents share a topic")
	}
	if rec := k.do(http.MethodPost, "/v1/me/ntfy/regenerate", bob, nil); rec.Code != http.StatusOK {
		t.Fatalf("bob regenerates: %d", rec.Code)
	}
	if rec := k.do(http.MethodDelete, "/v1/me/ntfy", bob, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("bob off: %d", rec.Code)
	}
	got := k.ntfyState(ana)
	if got["topic"] != topic || !slices.Equal(ntfyKids(got), []string{kid}) {
		t.Fatalf("bob's writes touched ana's topic: %v", got)
	}
	for _, path := range []string{"/v1/kids/" + kid, "/v1/kids/" + kid + "/access", "/v1/me/push-devices", "/v1/kids"} {
		if rec := k.do(http.MethodGet, path, bob, nil); strings.Contains(rec.Body.String(), topic) {
			t.Errorf("%s shows ana's topic to bob", path)
		}
	}
}

func TestRevokeAndLeaveRemoveOnlyThatKidsNtfyOptin(t *testing.T) {
	k := newKit(t)
	kid, ana, bob := k.trackerKid()
	second := k.createKid(ana, nil)
	k.grant(ana, second, "bob", "caregiver")
	topic := k.mustNtfy(bob)
	for _, id := range []string{kid, second} {
		if rec := k.do(http.MethodPut, ntfyOptinPath(id), bob, nil); rec.Code != http.StatusNoContent {
			t.Fatalf("opt in: %d", rec.Code)
		}
	}
	if rec := k.do(http.MethodDelete, "/v1/kids/"+kid+"/access/"+k.parentID("bob"), ana, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("revoke: %d %q", rec.Code, rec.Body.String())
	}
	got := k.ntfyState(bob)
	if got["topic"] != topic || !slices.Equal(ntfyKids(got), []string{second}) {
		t.Fatalf("after revoke: %v", got)
	}
	k.grant(ana, kid, "bob", "caregiver")
	if kids := ntfyKids(k.ntfyState(bob)); !slices.Equal(kids, []string{second}) {
		t.Fatalf("a re-grant restored the opt-in: %v", kids)
	}
	if rec := k.do(http.MethodDelete, "/v1/kids/"+second+"/access/"+k.parentID("bob"), bob, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("leave: %d %q", rec.Code, rec.Body.String())
	}
	got = k.ntfyState(bob)
	if got["topic"] != topic || len(got["kids"].([]any)) != 0 {
		t.Fatalf("after leave: %v", got)
	}
}

func TestDisableRemovesNtfyTopicHTTP(t *testing.T) {
	k := newKit(t)
	kid, _, bob := k.trackerKid()
	k.mustNtfy(bob)
	if rec := k.do(http.MethodPut, ntfyOptinPath(kid), bob, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("opt in: %d", rec.Code)
	}
	if rec := k.disable(k.asAdmin(), "bob"); rec.Code != http.StatusNoContent {
		t.Fatalf("disable: %d %q", rec.Code, rec.Body.String())
	}
	if _, err := k.Store.NtfyTopic(context.Background(), k.parentID("bob")); err == nil {
		t.Fatal("topic survived the disable")
	}
}

func TestNtfyRoutesRefuseTheAdmin(t *testing.T) {
	k := newKit(t)
	admin := k.asAdmin()
	for _, rt := range []struct{ method, path string }{
		{http.MethodGet, "/v1/me/ntfy"}, {http.MethodPut, "/v1/me/ntfy"},
		{http.MethodPost, "/v1/me/ntfy/regenerate"}, {http.MethodDelete, "/v1/me/ntfy"},
	} {
		wantErr(t, k.do(rt.method, rt.path, admin, nil), http.StatusForbidden, "forbidden")
	}
}

func TestNtfyOff(t *testing.T) {
	k := newKitNoNtfy(t)
	kid, ana, _ := k.trackerKid()
	if rec := k.do(http.MethodGet, "/v1/me/ntfy", ana, nil); rec.Code != http.StatusOK ||
		strings.TrimSpace(rec.Body.String()) != `{"enabled":false,"has_topic":false}` {
		t.Fatalf("state while off without a topic: %d %q", rec.Code, rec.Body.String())
	}
	if _, _, err := k.Store.SetNtfyTopic(context.Background(), k.parentID("ana"), "hbt-old", false, k.now); err != nil {
		t.Fatal(err)
	}
	if err := k.Store.SetNtfyOptin(context.Background(), kid, k.parentID("ana")); err != nil {
		t.Fatal(err)
	}
	rec := k.do(http.MethodGet, "/v1/me/ntfy", ana, nil)
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != `{"enabled":false,"has_topic":true}` {
		t.Fatalf("state while off with a topic: %d %q", rec.Code, rec.Body.String())
	}
	wantErr(t, k.do(http.MethodPut, "/v1/me/ntfy", ana, nil), http.StatusConflict, "ntfy_unavailable")
	wantErr(t, k.do(http.MethodPost, "/v1/me/ntfy/regenerate", ana, nil), http.StatusConflict, "ntfy_unavailable")
	wantErr(t, k.do(http.MethodPut, ntfyOptinPath(kid), ana, nil), http.StatusConflict, "ntfy_unavailable")
	if rec := k.do(http.MethodDelete, ntfyOptinPath(kid), ana, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("opt out while off: %d %q", rec.Code, rec.Body.String())
	}
	if rec := k.do(http.MethodDelete, "/v1/me/ntfy", ana, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("off while off: %d %q", rec.Code, rec.Body.String())
	}
	if _, err := k.Store.NtfyTopic(context.Background(), k.parentID("ana")); err == nil {
		t.Fatal("topic survived the delete")
	}
	if rec := k.do(http.MethodGet, "/v1/me/ntfy", ana, nil); strings.TrimSpace(rec.Body.String()) != `{"enabled":false,"has_topic":false}` {
		t.Fatalf("state while off after the delete: %d %q", rec.Code, rec.Body.String())
	}
}

func TestNtfyIndependentOfWebPush(t *testing.T) {
	k := newKitNoPush(t)
	if got := k.ntfyState(k.asParent("ana")); got["enabled"] != true {
		t.Fatalf("ntfy off with Web Push off: %v", got)
	}
	k = newKitNoNtfy(t)
	if rec := k.do(http.MethodGet, "/v1/push", k.asParent("ana"), nil); bodyMap(t, rec)["enabled"] != true {
		t.Fatalf("Web Push off with ntfy off: %q", rec.Body.String())
	}
}

func TestNtfyTopicNeverLogged(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	k.Log.Reset()
	topic := k.mustNtfy(ana)
	reqs := []struct{ method, path string }{
		{http.MethodGet, "/v1/me/ntfy"},
		{http.MethodPut, "/v1/me/ntfy"},
		{http.MethodPut, ntfyOptinPath(kid)},
		{http.MethodDelete, ntfyOptinPath(kid)},
		{http.MethodPost, "/v1/me/ntfy/regenerate"},
	}
	for _, r := range reqs {
		if rec := k.do(r.method, r.path, ana, nil); rec.Code >= 300 {
			t.Fatalf("%s %s: %d", r.method, r.path, rec.Code)
		}
	}
	next := k.ntfyState(ana)["topic"].(string)
	if rec := k.do(http.MethodDelete, "/v1/me/ntfy", ana, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("off: %d", rec.Code)
	}
	random := func(s string) string { return strings.TrimPrefix(s, "hbt-test-") }
	wantCleanLog(t, k.Log.String(), len(reqs)+3, topic, random(topic), next, random(next))
}
