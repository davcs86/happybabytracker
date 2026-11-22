package httpapi

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
)

func (k *kit) modules(token, kid string) []map[string]any {
	k.t.Helper()
	rec := k.do(http.MethodGet, "/v1/kids/"+kid+"/modules", token, nil)
	if rec.Code != http.StatusOK {
		k.t.Fatalf("modules: %d %q", rec.Code, rec.Body.String())
	}
	m := bodyMap(k.t, rec)
	sameKeys(k.t, m, "modules")
	var out []map[string]any
	for _, it := range m["modules"].([]any) {
		out = append(out, it.(map[string]any))
	}
	return out
}

func (k *kit) setModule(token, kid, module string, enabled bool) {
	k.t.Helper()
	rec := k.do(http.MethodPut, "/v1/kids/"+kid+"/modules/"+module, token, map[string]any{"enabled": enabled})
	if rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		k.t.Fatalf("set module %s %v: %d %q", module, enabled, rec.Code, rec.Body.String())
	}
}

func moduleItem(t *testing.T, items []map[string]any, module string) map[string]any {
	t.Helper()
	for _, it := range items {
		if it["module"] == module {
			return it
		}
	}
	t.Fatalf("no module %q in %v", module, items)
	return nil
}

func TestModulesShape(t *testing.T) {
	k := newKit(t)
	kid, _, bob := k.trackerKid()
	items := k.modules(bob, kid)
	if len(items) != len(model.Modules) {
		t.Fatalf("%d modules, want %d", len(items), len(model.Modules))
	}
	for i, it := range items {
		sameKeys(t, it, "module", "enabled", "default", "overridden")
		if it["module"] != string(model.Modules[i]) || it["enabled"] != true || it["default"] != true || it["overridden"] != false {
			t.Errorf("modules[%d] = %v, want %s on by default", i, it, model.Modules[i])
		}
	}
}

func TestCaregiverCannotSetModule(t *testing.T) {
	k := newKit(t)
	kid, ana, bob := k.trackerKid()
	cy := k.asParent("cy")
	path := "/v1/kids/" + kid + "/modules/bottles"
	off := map[string]any{"enabled": false}

	wantErr(t, k.do(http.MethodPut, path, bob, off), http.StatusForbidden, "forbidden")
	wantErr(t, k.do(http.MethodPut, path, cy, off), http.StatusNotFound, "not_found")
	if it := moduleItem(t, k.modules(ana, kid), "bottles"); it["enabled"] != true || it["overridden"] != false {
		t.Fatalf("a refused PUT changed bottles: %v", it)
	}

	wantErr(t, k.do(http.MethodPut, "/v1/kids/"+kid+"/modules/nope", bob, off), http.StatusForbidden, "forbidden")
	wantErr(t, k.do(http.MethodPut, "/v1/kids/"+kid+"/modules/nope", ana, off), http.StatusNotFound, "not_found")
	wantUnknownField(t, k.do(http.MethodPut, "/v1/kids/"+kid+"/modules/nope", ana, map[string]any{"on": true}), "on")
	wantInvalid(t, k.do(http.MethodPut, path, ana, map[string]any{}), "enabled")
	wantErr(t, k.do(http.MethodPut, path, ana, `{"enabled":"no"}`), http.StatusBadRequest, "bad_request")

	k.setModule(ana, kid, "bottles", false)
	if it := moduleItem(t, k.modules(bob, kid), "bottles"); it["enabled"] != false || it["default"] != true || it["overridden"] != true {
		t.Errorf("bottles after the guardian's off = %v", it)
	}
}

func TestSwitchedOffTrackerKeepsDataAndReturns(t *testing.T) {
	k := newKit(t)
	kid, ana, bob := k.trackerKid()
	id := k.createBottle(bob, kid, nil)
	list := func() []any {
		rec := k.do(http.MethodGet, "/v1/kids/"+kid+"/bottles", bob, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("list bottles: %d %q", rec.Code, rec.Body.String())
		}
		return bodyMap(t, rec)["bottles"].([]any)
	}
	k.setModule(ana, kid, "bottles", false)
	if got := list(); len(got) != 1 || got[0].(map[string]any)["id"] != id {
		t.Fatalf("bottles while off = %v", got)
	}
	k.setModule(ana, kid, "bottles", true)
	if got := list(); len(got) != 1 || got[0].(map[string]any)["id"] != id {
		t.Fatalf("bottles on again = %v", got)
	}
	if it := moduleItem(t, k.modules(bob, kid), "bottles"); it["enabled"] != true || it["overridden"] != true {
		t.Errorf("bottles on again = %v", it)
	}
}

func TestTimelineDefaultFollowsModules(t *testing.T) {
	k := newKit(t)
	kid, ana, bob := k.trackerKid()
	bottle := k.seedEntry(kid, "ana", "bottle", t0.Add(-2*time.Hour))
	diaper := k.seedEntry(kid, "ana", "diaper", t0.Add(-time.Hour))
	k.setModule(ana, kid, "bottles", false)

	rows, _, _ := k.timelinePage(bob, kid, "")
	if len(rows) != 1 || rows[0].ID != diaper {
		t.Errorf("default timeline with bottles off = %v, want the diaper only", rows)
	}
	rows, _, _ = k.timelinePage(bob, kid, "?types=bottle,diaper")
	if len(rows) != 2 || rows[1].ID != bottle {
		t.Errorf("types=bottle,diaper = %v, want both", rows)
	}

	for _, m := range []string{"nursing", "pumping", "diapers", "sleep", "feeding"} {
		k.setModule(ana, kid, m, false)
	}
	rec := k.do(http.MethodGet, "/v1/kids/"+kid+"/timeline", bob, nil)
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != `{"items":[],"next_cursor":null}` {
		t.Errorf("every tracker off: %d %q", rec.Code, rec.Body.String())
	}
}

func TestModuleAndHomeRoutesLogNoValues(t *testing.T) {
	const note = "xyzzy-private-note"
	k := newKit(t)
	kid, ana, bob := k.trackerKid()
	k.createBottle(bob, kid, map[string]any{"note": note, "volume": volumeBody("ml", 639)})
	k.createSleep(bob, kid, map[string]any{"note": note})
	k.Log.Reset()

	k.modules(bob, kid)
	k.setModule(ana, kid, "diapers", false)
	k.do(http.MethodPut, "/v1/kids/"+kid+"/modules/nope", ana, map[string]any{"enabled": true})
	k.do(http.MethodGet, "/v1/kids/"+kid+"/home", bob, nil)
	k.do(http.MethodGet, "/v1/kids/"+kid+"/timeline", bob, nil)

	lines := strings.Split(strings.TrimSpace(k.Log.String()), "\n")
	if len(lines) != 5 {
		t.Fatalf("%d log lines, want 5:\n%s", len(lines), k.Log.String())
	}
	allowed := []string{"time", "level", "msg", "route", "status", "principal_kind", "request_id", "duration_ms", constants.LogFieldCause, constants.LogFieldSlide}
	for _, line := range lines {
		var fields map[string]any
		if err := json.Unmarshal([]byte(line), &fields); err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		for key, v := range fields {
			if !slices.Contains(allowed, key) {
				t.Errorf("unexpected log field %q in %s", key, line)
			}
			s, _ := json.Marshal(v)
			if key == "request_id" || key == "time" {
				continue
			}
			for _, banned := range []string{"xyzzy", "639", "diapers", "nope", "enabled", testKidName, kid} {
				if strings.Contains(string(s), banned) {
					t.Errorf("log field %s = %s carries %q", key, s, banned)
				}
			}
		}
	}
}
