package httpapi

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
)

// createSleep posts a sleep session and returns its id; the status must be 201.
func (k *kit) createSleep(token, kid string, body map[string]any) string {
	k.t.Helper()
	rec := k.do(http.MethodPost, "/v1/kids/"+kid+"/sleep", token, body)
	if rec.Code != http.StatusCreated {
		k.t.Fatalf("create sleep: %d %q", rec.Code, rec.Body.String())
	}
	return bodyMap(k.t, rec)["id"].(string)
}

func wantUnknownField(t *testing.T, rec *httptest.ResponseRecorder, field string) {
	t.Helper()
	if want := `{"error":"unknown_field","field":"` + field + `"}`; rec.Code != http.StatusUnprocessableEntity || strings.TrimSpace(rec.Body.String()) != want {
		t.Errorf("got %d %q, want 422 %s", rec.Code, rec.Body.String(), want)
	}
}

func finishedSleepBody() map[string]any {
	return map[string]any{"started_at": ts(t0.Add(-2 * time.Hour)), "ended_at": ts(t0.Add(-time.Hour)), "note": "nap"}
}

func TestSleepLifecycle(t *testing.T) {
	k := newKit(t)
	kid, ana, bo := k.trackerKid()
	base := "/v1/kids/" + kid + "/sleep"

	rec := k.do(http.MethodPost, base, ana, map[string]any{})
	if rec.Code != http.StatusCreated {
		t.Fatalf("start: %d %q", rec.Code, rec.Body.String())
	}
	started := bodyMap(t, rec)
	sameKeys(t, started, "id", "started_at", "ended_at", "note")
	if started["started_at"] != ts(t0) || started["ended_at"] != nil || started["note"] != nil {
		t.Fatalf("started = %v, want started now and running", started)
	}
	one := base + "/" + started["id"].(string)

	wantErr(t, k.do(http.MethodPost, base, bo, map[string]any{}), http.StatusConflict, "already_running")
	// A finished session may still be logged while one runs.
	k.createSleep(bo, kid, finishedSleepBody())

	if rec := k.do(http.MethodGet, one, bo, nil); rec.Code != http.StatusOK || !equalJSON(bodyMap(t, rec), started) {
		t.Errorf("the other parent sees %d %q", rec.Code, rec.Body.String())
	}

	k.now = t0.Add(40 * time.Minute)
	rec = k.do(http.MethodPost, one+"/end", bo, map[string]any{})
	ended := bodyMap(t, rec)
	if rec.Code != http.StatusOK || ended["ended_at"] != ts(k.now) || ended["started_at"] != ts(t0) || ended["id"] != started["id"] {
		t.Fatalf("end: %d %v", rec.Code, ended)
	}
	sameKeys(t, ended, "id", "started_at", "ended_at", "note")
	wantErr(t, k.do(http.MethodPost, one+"/end", ana, map[string]any{}), http.StatusConflict, "not_running")
	if rec := k.do(http.MethodGet, one, ana, nil); !equalJSON(bodyMap(t, rec), ended) {
		t.Errorf("the refused end changed the session: %q", rec.Body.String())
	}

	// Once ended, a new one may start; an explicit start time is kept, truncated to the second.
	rec = k.do(http.MethodPost, base, ana, map[string]any{"started_at": "2026-11-20T12:30:00.9Z"})
	if m := bodyMap(t, rec); rec.Code != http.StatusCreated || m["started_at"] != "2026-11-20T12:30:00Z" {
		t.Fatalf("restart: %d %v", rec.Code, m)
	}
	rec = k.do(http.MethodPost, base+"/"+bodyMap(t, rec)["id"].(string)+"/end", ana, map[string]any{"at": "2026-11-20T12:35:10.999Z"})
	if m := bodyMap(t, rec); rec.Code != http.StatusOK || m["ended_at"] != "2026-11-20T12:35:10Z" {
		t.Errorf("end with at: %d %v", rec.Code, m)
	}

	list := bodyMap(t, k.do(http.MethodGet, base, ana, nil))
	sameKeys(t, list, "sleep", "next_cursor")
	if rows := list["sleep"].([]any); len(rows) != 3 || rows[2].(map[string]any)["note"] != "nap" {
		t.Errorf("list = %v, want three sessions newest start first", rows)
	}
}

func TestSleepEndAt(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	base := "/v1/kids/" + kid + "/sleep"
	start := t0.Add(-time.Hour)
	id := k.createSleep(ana, kid, map[string]any{"started_at": ts(start)})
	end := base + "/" + id + "/end"

	for name, at := range map[string]string{
		"before the start":              ts(start.Add(-time.Second)),
		"more than 5 minutes ahead":     ts(t0.Add(constants.MaxFutureSkew + time.Second)),
		"before the birth date":         "2026-03-03T22:59:59Z",
		"longer than the session limit": ts(start.Add(constants.MaxSessionDuration + time.Second)),
	} {
		wantInvalid2(t, name, k.do(http.MethodPost, end, ana, map[string]any{"at": at}), "at")
	}
	wantErr(t, k.do(http.MethodPost, end, ana, map[string]any{"at": "soon"}), http.StatusBadRequest, "bad_request")
	wantUnknownField(t, k.do(http.MethodPost, end, ana, map[string]any{"side": "left"}), "side")
	if m := bodyMap(t, k.do(http.MethodGet, base+"/"+id, ana, nil)); m["ended_at"] != nil {
		t.Fatalf("a refused end ended the session: %v", m)
	}
	if rec := k.do(http.MethodPost, end, ana, map[string]any{"at": ts(t0.Add(constants.MaxFutureSkew))}); rec.Code != http.StatusOK {
		t.Errorf("end 5 minutes ahead: %d %q", rec.Code, rec.Body.String())
	}
}

func TestSleepCreateValidation(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	base := "/v1/kids/" + kid + "/sleep"
	start := t0.Add(-30 * time.Hour)
	for name, c := range map[string]struct {
		body  map[string]any
		field string
	}{
		"start ahead":         {map[string]any{"started_at": ts(t0.Add(constants.MaxFutureSkew + time.Second))}, "started_at"},
		"start before birth":  {map[string]any{"started_at": "2026-03-03T22:59:59Z"}, "started_at"},
		"end ahead":           {map[string]any{"started_at": ts(start), "ended_at": ts(t0.Add(constants.MaxFutureSkew + time.Second))}, "ended_at"},
		"end before start":    {map[string]any{"started_at": ts(start), "ended_at": ts(start.Add(-time.Second))}, "ended_at"},
		"longer than a limit": {map[string]any{"started_at": ts(start), "ended_at": ts(start.Add(constants.MaxSessionDuration + time.Second))}, "ended_at"},
		"note over the limit": {map[string]any{"note": strings.Repeat("ñ", constants.MaxNoteLen+1)}, "note"},
	} {
		wantInvalid2(t, name, k.do(http.MethodPost, base, ana, c.body), c.field)
	}
	wantUnknownField(t, k.do(http.MethodPost, base, ana, map[string]any{"side": "left"}), "side")
	if n := len(bodyMap(t, k.do(http.MethodGet, base, ana, nil))["sleep"].([]any)); n != 0 {
		t.Errorf("%d sessions after refused creates", n)
	}
	k.createSleep(ana, kid, map[string]any{"started_at": ts(start), "ended_at": ts(start.Add(constants.MaxSessionDuration))})
	k.createSleep(ana, kid, map[string]any{"started_at": ts(start), "ended_at": nil}) // an explicit null starts a timer
	wantErr(t, k.do(http.MethodPost, base, ana, map[string]any{}), http.StatusConflict, "already_running")
}

func TestSleepPatch(t *testing.T) {
	k := newKit(t)
	kid, ana, bo := k.trackerKid()
	base := "/v1/kids/" + kid + "/sleep"
	runningID := k.createSleep(ana, kid, map[string]any{"started_at": ts(t0.Add(-time.Hour))})
	running := base + "/" + runningID

	rec := k.do(http.MethodPatch, running, bo, map[string]any{"note": "fussy", "ended_at": nil, "started_at": ts(t0.Add(-2 * time.Hour))})
	if m := bodyMap(t, rec); rec.Code != http.StatusOK || m["ended_at"] != nil || m["note"] != "fussy" || m["started_at"] != ts(t0.Add(-2*time.Hour)) {
		t.Fatalf("patch running: %d %v", rec.Code, m)
	}
	wantInvalid2(t, "running end before start", k.do(http.MethodPatch, running, bo, map[string]any{"ended_at": ts(t0.Add(-3 * time.Hour))}), "ended_at")
	rec = k.do(http.MethodPatch, running, bo, map[string]any{"ended_at": ts(t0.Add(-time.Minute))})
	if m := bodyMap(t, rec); rec.Code != http.StatusOK || m["ended_at"] != ts(t0.Add(-time.Minute)) || m["note"] != "fussy" {
		t.Fatalf("end through patch: %d %v", rec.Code, m)
	}
	wantErr(t, k.do(http.MethodPost, running+"/end", ana, map[string]any{}), http.StatusConflict, "not_running")

	// A finished session is never re-opened, but its times stay correctable.
	stored := k.do(http.MethodGet, running, ana, nil).Body.String()
	wantInvalid2(t, "re-open", k.do(http.MethodPatch, running, ana, map[string]any{"ended_at": nil}), "ended_at")
	wantInvalid2(t, "end before start", k.do(http.MethodPatch, running, ana, map[string]any{"ended_at": ts(t0.Add(-3 * time.Hour))}), "ended_at")
	wantInvalid2(t, "start after end", k.do(http.MethodPatch, running, ana, map[string]any{"started_at": ts(t0)}), "ended_at")
	wantInvalid2(t, "too long", k.do(http.MethodPatch, running, ana, map[string]any{"started_at": ts(t0.Add(-constants.MaxSessionDuration - 2*time.Minute))}), "ended_at")
	wantInvalid2(t, "start ahead", k.do(http.MethodPatch, running, ana, map[string]any{"started_at": ts(t0.Add(time.Hour))}), "started_at")
	wantInvalid2(t, "note", k.do(http.MethodPatch, running, ana, map[string]any{"note": strings.Repeat("a", constants.MaxNoteLen+1)}), "note")
	if got := k.do(http.MethodGet, running, ana, nil).Body.String(); got != stored {
		t.Errorf("a refused patch changed the session:\n got %s\nwant %s", got, stored)
	}
	rec = k.do(http.MethodPatch, running, ana, map[string]any{"ended_at": ts(t0)})
	if m := bodyMap(t, rec); rec.Code != http.StatusOK || m["ended_at"] != ts(t0) {
		t.Errorf("correct a finished end: %d %v", rec.Code, m)
	}
	if rec := k.do(http.MethodGet, base, ana, nil); len(bodyMap(t, rec)["sleep"].([]any)) != 1 {
		t.Errorf("list: %q", rec.Body.String())
	}
	// Ended through PATCH, so a new timer may start.
	k.createSleep(ana, kid, map[string]any{})
}

type timerCall struct {
	method, path string
	body         any
}

// timerCalls lists every route of the tracker (sleep or nursing) for entry on kidPath.
func timerCalls(tracker, kidPath, entry string, create any) []timerCall {
	base := "/v1/kids/" + kidPath + "/" + tracker
	calls := []timerCall{
		{http.MethodGet, base, nil},
		{http.MethodPost, base, create},
		{http.MethodGet, base + "/" + entry, nil},
		{http.MethodPatch, base + "/" + entry, map[string]any{"note": "x"}},
		{http.MethodDelete, base + "/" + entry, nil},
		{http.MethodPost, base + "/" + entry + "/end", map[string]any{}},
	}
	if tracker == "nursing" {
		calls = append(calls, timerCall{http.MethodPost, base + "/" + entry + "/side", map[string]any{"side": "right"}})
	}
	return calls
}

// Every refusal below must be what the same route answers elsewhere in the kid API.
func TestTimerRoutesRefuseOutsiders(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	other := k.createKid(ana, map[string]any{"name": "Second"})
	xavi, admin := k.asParent("xavi"), k.asAdmin()

	for tracker, create := range map[string]map[string]any{"sleep": {}, "nursing": {"side": "left"}} {
		id := bodyMap(t, k.do(http.MethodPost, "/v1/kids/"+kid+"/"+tracker, ana, create))["id"].(string)
		otherID := bodyMap(t, k.do(http.MethodPost, "/v1/kids/"+other+"/"+tracker, ana, create))["id"].(string)
		stored := k.do(http.MethodGet, "/v1/kids/"+kid+"/"+tracker+"/"+id, ana, nil).Body.String()
		for i, c := range timerCalls(tracker, kid, id, create) {
			unknown := timerCalls(tracker, model.NewID(), id, create)[i]
			wantBody := k.do(unknown.method, unknown.path, xavi, unknown.body).Body.String()
			if rec := k.do(c.method, c.path, xavi, c.body); rec.Code != http.StatusNotFound || rec.Body.String() != wantBody {
				t.Errorf("no grant %s %s: %d %q, want the unknown-kid 404 %q", c.method, c.path, rec.Code, rec.Body.String(), wantBody)
			}
			wantErr(t, k.do(c.method, c.path, admin, c.body), http.StatusForbidden, "forbidden")
		}
		for _, entry := range []string{otherID, model.NewID(), "not an id", strings.Repeat("a", 65)} {
			for _, c := range timerCalls(tracker, kid, url.PathEscape(entry), create)[2:] {
				wantErr(t, k.do(c.method, c.path, ana, c.body), http.StatusNotFound, "not_found")
			}
		}
		// A bad body on a missing entry is still the 404.
		missing := "/v1/kids/" + kid + "/" + tracker + "/" + model.NewID()
		wantErr(t, k.do(http.MethodPost, missing+"/end", ana, map[string]any{"at": ts(t0.Add(time.Hour))}), http.StatusNotFound, "not_found")
		for kidID, entry := range map[string]string{kid: id, other: otherID} {
			if rec := k.do(http.MethodGet, "/v1/kids/"+kidID+"/"+tracker+"/"+entry, ana, nil); rec.Code != http.StatusOK || bodyMap(t, rec)["ended_at"] != nil {
				t.Errorf("%s %s changed or vanished: %d %q", tracker, entry, rec.Code, rec.Body.String())
			}
		}
		if got := k.do(http.MethodGet, "/v1/kids/"+kid+"/"+tracker+"/"+id, ana, nil).Body.String(); got != stored {
			t.Errorf("%s: a refused call changed the session:\n got %s\nwant %s", tracker, got, stored)
		}
	}
}

func TestTimerRoutesRoles(t *testing.T) {
	k := newKit(t)
	kid, ana, bo := k.trackerKid()
	for tracker, create := range map[string]map[string]any{"sleep": {}, "nursing": {"side": "left"}} {
		for name, token := range map[string]string{"guardian": ana, "caregiver": bo} {
			base := "/v1/kids/" + kid + "/" + tracker
			rec := k.do(http.MethodPost, base, token, create)
			if rec.Code != http.StatusCreated {
				t.Fatalf("%s %s start: %d %q", name, tracker, rec.Code, rec.Body.String())
			}
			one := base + "/" + bodyMap(t, rec)["id"].(string)
			if tracker == "nursing" {
				if rec := k.do(http.MethodPost, one+"/side", token, map[string]any{"side": "right"}); rec.Code != http.StatusOK {
					t.Errorf("%s side: %d %q", name, rec.Code, rec.Body.String())
				}
			}
			if rec := k.do(http.MethodPatch, one, token, map[string]any{"note": "n"}); rec.Code != http.StatusOK {
				t.Errorf("%s %s patch: %d %q", name, tracker, rec.Code, rec.Body.String())
			}
			if rec := k.do(http.MethodPost, one+"/end", token, map[string]any{}); rec.Code != http.StatusOK {
				t.Errorf("%s %s end: %d %q", name, tracker, rec.Code, rec.Body.String())
			}
			if rec := k.do(http.MethodGet, one, token, nil); rec.Code != http.StatusOK {
				t.Errorf("%s %s get: %d", name, tracker, rec.Code)
			}
			if rec := k.do(http.MethodDelete, one, token, nil); rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
				t.Errorf("%s %s delete: %d %q", name, tracker, rec.Code, rec.Body.String())
			}
			wantErr(t, k.do(http.MethodGet, one, token, nil), http.StatusNotFound, "not_found")
		}
	}
}

func TestTimerListQuery(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	for _, tracker := range []string{"sleep", "nursing"} {
		base := "/v1/kids/" + kid + "/" + tracker
		var ids []string
		for _, m := range []time.Duration{3, 1, 2} {
			body := map[string]any{"started_at": ts(t0.Add(-m * time.Hour)), "ended_at": ts(t0.Add(-m*time.Hour + time.Minute))}
			if tracker == "nursing" {
				body["left_seconds"], body["right_seconds"] = 30, 30
			}
			rec := k.do(http.MethodPost, base, ana, body)
			if rec.Code != http.StatusCreated {
				t.Fatalf("%s create: %d %q", tracker, rec.Code, rec.Body.String())
			}
			ids = append(ids, bodyMap(t, rec)["id"].(string))
		}
		want := []string{ids[1], ids[2], ids[0]}
		var got []string
		query := "?limit=2"
		for range 3 {
			rec := k.do(http.MethodGet, base+query, ana, nil)
			m := bodyMap(t, rec)
			sameKeys(t, m, tracker, "next_cursor")
			for _, row := range m[tracker].([]any) {
				got = append(got, row.(map[string]any)["id"].(string))
			}
			if m["next_cursor"] == nil {
				break
			}
			query = "?limit=2&cursor=" + url.QueryEscape(m["next_cursor"].(string))
		}
		if !slices.Equal(got, want) {
			t.Errorf("%s paged = %v, want %v (newest start first)", tracker, got, want)
		}
		rec := k.do(http.MethodGet, base+"?from="+url.QueryEscape(ts(t0.Add(-150*time.Minute)))+"&to="+url.QueryEscape(ts(t0.Add(-90*time.Minute))), ana, nil)
		if rows := bodyMap(t, rec)[tracker].([]any); len(rows) != 1 || rows[0].(map[string]any)["id"] != ids[2] {
			t.Errorf("%s range = %v", tracker, rows)
		}
		for query, field := range map[string]string{"?limit=abc": "limit", "?cursor=!!": "cursor", "?from=yesterday": "from"} {
			wantInvalid(t, k.do(http.MethodGet, base+query, ana, nil), field)
		}
	}
}

func TestTimersNoValueInLogs(t *testing.T) {
	const note = "xyzzy-private-note"
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	k.Log.Reset()

	s := bodyMap(t, k.do(http.MethodPost, "/v1/kids/"+kid+"/sleep", ana, map[string]any{"note": note}))["id"].(string)
	k.do(http.MethodPatch, "/v1/kids/"+kid+"/sleep/"+s, ana, map[string]any{"note": note + "2"})
	k.do(http.MethodPost, "/v1/kids/"+kid+"/sleep/"+s+"/end", ana, map[string]any{})
	k.do(http.MethodPost, "/v1/kids/"+kid+"/sleep/"+s+"/end", ana, map[string]any{}) // a 409
	n := bodyMap(t, k.do(http.MethodPost, "/v1/kids/"+kid+"/nursing", ana, map[string]any{"side": "left", "note": note}))["id"].(string)
	k.do(http.MethodPost, "/v1/kids/"+kid+"/nursing/"+n+"/side", ana, map[string]any{"side": "right"})
	k.do(http.MethodPatch, "/v1/kids/"+kid+"/nursing/"+n, ana, map[string]any{"note": note + "3", "left_seconds": 4321}) // a 422
	k.do(http.MethodPost, "/v1/kids/"+kid+"/nursing/"+n+"/end", ana, map[string]any{})

	wantCleanLog(t, k.Log.String(), 8, "xyzzy", "4321")
}

// A running timer cannot start in the future: it could not be ended until the clock got there.
func TestRunningSleepRefusesAFutureStart(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	base := "/v1/kids/" + kid + "/sleep"
	wantInvalid2(t, "create", k.do(http.MethodPost, base, ana, map[string]any{"started_at": ts(t0.Add(time.Second))}), "started_at")
	one := base + "/" + k.createSleep(ana, kid, map[string]any{"started_at": ts(t0.Add(-time.Minute))})
	wantInvalid2(t, "patch", k.do(http.MethodPatch, one, ana, map[string]any{"started_at": ts(t0.Add(time.Second))}), "started_at")
	if rec := k.do(http.MethodPatch, one, ana, map[string]any{"started_at": ts(t0)}); rec.Code != http.StatusOK {
		t.Errorf("start at now: %d %q", rec.Code, rec.Body.String())
	}
	k.do(http.MethodPost, one+"/end", ana, map[string]any{})
	finished := map[string]any{"started_at": ts(t0.Add(time.Minute)), "ended_at": ts(t0.Add(constants.MaxFutureSkew))}
	if rec := k.do(http.MethodPost, base, ana, finished); rec.Code != http.StatusCreated {
		t.Errorf("a finished session keeps the skew: %d %q", rec.Code, rec.Body.String())
	}
}
