package httpapi

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

// createNursing posts a nursing session and returns its id; the status must be 201.
func (k *kit) createNursing(token, kid string, body map[string]any) string {
	k.t.Helper()
	rec := k.do(http.MethodPost, "/v1/kids/"+kid+"/nursing", token, body)
	if rec.Code != http.StatusCreated {
		k.t.Fatalf("create nursing: %d %q", rec.Code, rec.Body.String())
	}
	return bodyMap(k.t, rec)["id"].(string)
}

func finishedNursingBody(over map[string]any) map[string]any {
	return withOverrides(map[string]any{
		"started_at":    ts(t0.Add(-time.Hour)),
		"ended_at":      ts(t0.Add(-50 * time.Minute)), // 600 s
		"left_seconds":  300,
		"right_seconds": 240,
		"note":          "calm",
	}, over)
}

var nursingKeys = []string{"id", "started_at", "ended_at", "left_seconds", "right_seconds", "side", "side_since", "note"}

func TestNursingLifecycle(t *testing.T) {
	k := newKit(t)
	kid, ana, bo := k.trackerKid()
	base := "/v1/kids/" + kid + "/nursing"

	rec := k.do(http.MethodPost, base, ana, map[string]any{"side": "left"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("start: %d %q", rec.Code, rec.Body.String())
	}
	m := bodyMap(t, rec)
	sameKeys(t, m, nursingKeys...)
	if m["started_at"] != ts(t0) || m["ended_at"] != nil || m["side"] != "left" || m["side_since"] != ts(t0) ||
		m["left_seconds"] != 0.0 || m["right_seconds"] != 0.0 || m["note"] != nil {
		t.Fatalf("started = %v", m)
	}
	one := base + "/" + m["id"].(string)
	wantErr(t, k.do(http.MethodPost, base, bo, map[string]any{"side": "right"}), http.StatusConflict, "already_running")
	k.createNursing(bo, kid, finishedNursingBody(nil)) // a finished session may be logged while one runs

	k.now = t0.Add(10 * time.Minute)
	rec = k.do(http.MethodPost, one+"/side", bo, map[string]any{"side": "right"})
	if m := bodyMap(t, rec); rec.Code != http.StatusOK || m["left_seconds"] != 600.0 || m["right_seconds"] != 0.0 ||
		m["side"] != "right" || m["side_since"] != ts(k.now) {
		t.Fatalf("switch to right: %d %v", rec.Code, m)
	}
	k.now = k.now.Add(5 * time.Minute)
	rec = k.do(http.MethodPost, one+"/side", ana, map[string]any{"side": "left", "at": ts(k.now.Add(-time.Minute))})
	if m := bodyMap(t, rec); rec.Code != http.StatusOK || m["right_seconds"] != 240.0 || m["side"] != "left" || m["side_since"] != ts(k.now.Add(-time.Minute)) {
		t.Fatalf("switch back with at: %d %v", rec.Code, m)
	}
	wantInvalid2(t, "switch before side_since", k.do(http.MethodPost, one+"/side", ana, map[string]any{"side": "right", "at": ts(t0.Add(11 * time.Minute))}), "at")
	wantInvalid2(t, "switch ahead", k.do(http.MethodPost, one+"/side", ana, map[string]any{"side": "right", "at": ts(k.now.Add(constants.MaxFutureSkew + time.Second))}), "at")
	for name, body := range map[string]map[string]any{"both": {"side": "both"}, "empty": {"side": ""}, "wrong case": {"side": "Left"}, "absent": {}} {
		wantInvalid2(t, "side "+name, k.do(http.MethodPost, one+"/side", ana, body), "side")
	}
	wantUnknownField(t, k.do(http.MethodPost, one+"/side", ana, map[string]any{"side": "left", "seconds": 1}), "seconds")
	wantInvalid2(t, "end before side_since", k.do(http.MethodPost, one+"/end", ana, map[string]any{"at": ts(t0.Add(13 * time.Minute))}), "at")

	k.now = k.now.Add(10 * time.Minute)
	rec = k.do(http.MethodPost, one+"/end", bo, map[string]any{})
	ended := bodyMap(t, rec)
	if rec.Code != http.StatusOK || ended["left_seconds"] != 600.0+660 || ended["right_seconds"] != 240.0 ||
		ended["ended_at"] != ts(k.now) || ended["side"] != nil || ended["side_since"] != nil {
		t.Fatalf("end: %d %v", rec.Code, ended)
	}
	sameKeys(t, ended, nursingKeys...)
	wantErr(t, k.do(http.MethodPost, one+"/end", ana, map[string]any{}), http.StatusConflict, "not_running")
	wantErr(t, k.do(http.MethodPost, one+"/side", ana, map[string]any{"side": "left"}), http.StatusConflict, "not_running")
	if rec := k.do(http.MethodGet, one, ana, nil); !equalJSON(bodyMap(t, rec), ended) {
		t.Errorf("refused calls changed the session: %q", rec.Body.String())
	}
	list := bodyMap(t, k.do(http.MethodGet, base, ana, nil))
	sameKeys(t, list, "nursing", "next_cursor")
	if rows := list["nursing"].([]any); len(rows) != 2 || rows[0].(map[string]any)["id"] != ended["id"] {
		t.Errorf("list = %v", rows)
	}
	k.createNursing(ana, kid, map[string]any{"side": "right"}) // the next timer may start
}

func TestNursingCreateValidation(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	base := "/v1/kids/" + kid + "/nursing"
	start := t0.Add(-30 * time.Hour)
	for name, c := range map[string]struct {
		body  map[string]any
		field string
	}{
		"start ahead":             {map[string]any{"side": "left", "started_at": ts(t0.Add(constants.MaxFutureSkew + time.Second))}, "started_at"},
		"start before birth":      {map[string]any{"side": "left", "started_at": "2026-03-03T22:59:59Z"}, "started_at"},
		"running without a side":  {map[string]any{}, "side"},
		"running on side both":    {map[string]any{"side": "both"}, "side"},
		"running with left":       {map[string]any{"side": "left", "left_seconds": 0}, "left_seconds"},
		"running with right":      {map[string]any{"side": "left", "right_seconds": 0}, "right_seconds"},
		"note over the limit":     {map[string]any{"side": "left", "note": strings.Repeat("ñ", constants.MaxNoteLen+1)}, "note"},
		"finished without left":   {finishedNursingBody(map[string]any{"left_seconds": nil}), "left_seconds"},
		"finished without right":  {finishedNursingBody(map[string]any{"right_seconds": nil}), "right_seconds"},
		"finished with a side":    {finishedNursingBody(map[string]any{"side": "left"}), "side"},
		"finished left negative":  {finishedNursingBody(map[string]any{"left_seconds": -1}), "left_seconds"},
		"finished right negative": {finishedNursingBody(map[string]any{"right_seconds": -1}), "right_seconds"},
		"seconds over duration":   {finishedNursingBody(map[string]any{"left_seconds": 361}), "left_seconds"},
		"seconds sum wraps int64": {finishedNursingBody(map[string]any{"left_seconds": 1 << 62, "right_seconds": 1 << 62}), "left_seconds"},
		"end ahead":               {finishedNursingBody(map[string]any{"ended_at": ts(t0.Add(constants.MaxFutureSkew + time.Second))}), "ended_at"},
		"end before start":        {finishedNursingBody(map[string]any{"ended_at": ts(t0.Add(-time.Hour - time.Second))}), "ended_at"},
		"longer than the limit": {finishedNursingBody(map[string]any{
			"started_at": ts(start), "ended_at": ts(start.Add(constants.MaxSessionDuration + time.Second)),
		}), "ended_at"},
	} {
		wantInvalid2(t, name, k.do(http.MethodPost, base, ana, c.body), c.field)
	}
	// A number past int64 fails to decode: 400, never a 500.
	tooBig := `{"started_at":"2026-11-20T11:00:00Z","ended_at":"2026-11-20T11:10:00Z","left_seconds":9223372036854775808,"right_seconds":0}`
	wantErr(t, k.do(http.MethodPost, base, ana, tooBig), http.StatusBadRequest, "bad_request")
	for _, body := range []string{`{"side":"left","left_seconds":1.5}`, `{"side":"left","side_since":"2026-11-20T12:00:00Z"}`} {
		if rec := k.do(http.MethodPost, base, ana, body); rec.Code != http.StatusBadRequest && rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("%s: %d %q", body, rec.Code, rec.Body.String())
		}
	}
	if n := len(bodyMap(t, k.do(http.MethodGet, base, ana, nil))["nursing"].([]any)); n != 0 {
		t.Fatalf("%d sessions after refused creates", n)
	}
	rec := k.do(http.MethodPost, base, ana, finishedNursingBody(map[string]any{"left_seconds": 360}))
	if m := bodyMap(t, rec); rec.Code != http.StatusCreated || m["left_seconds"] != 360.0 || m["right_seconds"] != 240.0 ||
		m["side"] != nil || m["side_since"] != nil || m["note"] != "calm" {
		t.Errorf("finished at exactly the duration: %d %v", rec.Code, m)
	}
	rec = k.do(http.MethodPost, base, ana, map[string]any{"side": "right", "started_at": "2026-11-20T11:59:30.7Z"})
	if m := bodyMap(t, rec); rec.Code != http.StatusCreated || m["started_at"] != "2026-11-20T11:59:30Z" || m["side_since"] != "2026-11-20T11:59:30Z" {
		t.Errorf("start with a fractional time: %d %v", rec.Code, m)
	}
}

func TestNursingPatch(t *testing.T) {
	k := newKit(t)
	kid, ana, bo := k.trackerKid()
	base := "/v1/kids/" + kid + "/nursing"
	finished := base + "/" + k.createNursing(ana, kid, finishedNursingBody(nil))

	rec := k.do(http.MethodPatch, finished, bo, map[string]any{"left_seconds": 0, "right_seconds": 600, "note": nil})
	if m := bodyMap(t, rec); rec.Code != http.StatusOK || m["left_seconds"] != 0.0 || m["right_seconds"] != 600.0 || m["note"] != nil {
		t.Fatalf("patch finished seconds: %d %v", rec.Code, m)
	}
	stored := k.do(http.MethodGet, finished, ana, nil).Body.String()
	for name, c := range map[string]struct {
		body  map[string]any
		field string
	}{
		"over the duration": {map[string]any{"left_seconds": 1}, "left_seconds"},
		"sum wraps int64":   {map[string]any{"left_seconds": 1 << 62, "right_seconds": 1 << 62}, "left_seconds"},
		"shorter session":   {map[string]any{"started_at": ts(t0.Add(-time.Hour + time.Second))}, "left_seconds"},
		"negative right":    {map[string]any{"right_seconds": -1}, "right_seconds"},
		"re-open":           {map[string]any{"ended_at": nil}, "ended_at"},
		"end before start":  {map[string]any{"ended_at": ts(t0.Add(-2 * time.Hour))}, "ended_at"},
		"end ahead":         {map[string]any{"ended_at": ts(t0.Add(time.Hour))}, "ended_at"},
		"a side":            {map[string]any{"side": "left"}, "side"},
	} {
		wantInvalid2(t, name, k.do(http.MethodPatch, finished, ana, c.body), c.field)
	}
	wantErr(t, k.do(http.MethodPatch, finished, ana, `{"right_seconds":9223372036854775808}`), http.StatusBadRequest, "bad_request")
	if got := k.do(http.MethodGet, finished, ana, nil).Body.String(); got != stored {
		t.Errorf("a refused patch changed the session:\n got %s\nwant %s", got, stored)
	}
	rec = k.do(http.MethodPatch, finished, ana, map[string]any{"ended_at": ts(t0), "left_seconds": 3000})
	if m := bodyMap(t, rec); rec.Code != http.StatusOK || m["ended_at"] != ts(t0) || m["left_seconds"] != 3000.0 {
		t.Errorf("correct a finished end and seconds: %d %v", rec.Code, m)
	}

	runningID := k.createNursing(ana, kid, map[string]any{"side": "left", "started_at": ts(t0.Add(-20 * time.Minute))})
	running := base + "/" + runningID
	if m := bodyMap(t, k.do(http.MethodPost, running+"/side", ana, map[string]any{"side": "right", "at": ts(t0.Add(-10 * time.Minute))})); m["left_seconds"] != 600.0 {
		t.Fatalf("switch: %v", m)
	}
	storedRunning := k.do(http.MethodGet, running, ana, nil).Body.String()
	for name, c := range map[string]struct {
		body  map[string]any
		field string
	}{
		"left while running":         {map[string]any{"left_seconds": 600}, "left_seconds"},
		"right while running":        {map[string]any{"right_seconds": 0}, "right_seconds"},
		"side through patch":         {map[string]any{"side": "left"}, "side"},
		"start after side_since":     {map[string]any{"started_at": ts(t0.Add(-9 * time.Minute))}, "started_at"},
		"start leaves too few":       {map[string]any{"started_at": ts(t0.Add(-11 * time.Minute))}, "started_at"},
		"end before side_since":      {map[string]any{"ended_at": ts(t0.Add(-11 * time.Minute))}, "ended_at"},
		"end and seconds at once":    {map[string]any{"ended_at": ts(t0), "left_seconds": 1}, "left_seconds"},
		"end ahead while running":    {map[string]any{"ended_at": ts(t0.Add(time.Hour))}, "ended_at"},
		"note over limit, running":   {map[string]any{"note": strings.Repeat("a", constants.MaxNoteLen+1)}, "note"},
		"start ahead while running":  {map[string]any{"started_at": ts(t0.Add(time.Hour))}, "started_at"},
		"start before birth running": {map[string]any{"started_at": "2026-03-03T22:59:59Z"}, "started_at"},
	} {
		wantInvalid2(t, name, k.do(http.MethodPatch, running, ana, c.body), c.field)
	}
	if got := k.do(http.MethodGet, running, ana, nil).Body.String(); got != storedRunning {
		t.Errorf("a refused patch changed the running session:\n got %s\nwant %s", got, storedRunning)
	}
	rec = k.do(http.MethodPatch, running, ana, map[string]any{"note": "ok", "ended_at": nil, "started_at": ts(t0.Add(-30 * time.Minute))})
	if m := bodyMap(t, rec); rec.Code != http.StatusOK || m["ended_at"] != nil || m["side"] != "right" || m["note"] != "ok" || m["left_seconds"] != 600.0 {
		t.Fatalf("patch running: %d %v", rec.Code, m)
	}
	rec = k.do(http.MethodPatch, running, bo, map[string]any{"ended_at": ts(t0.Add(-4 * time.Minute))})
	if m := bodyMap(t, rec); rec.Code != http.StatusOK || m["ended_at"] != ts(t0.Add(-4*time.Minute)) || m["right_seconds"] != 360.0 ||
		m["left_seconds"] != 600.0 || m["side"] != nil || m["side_since"] != nil {
		t.Fatalf("end through patch: %d %v", rec.Code, m)
	}
	wantErr(t, k.do(http.MethodPost, running+"/end", ana, map[string]any{}), http.StatusConflict, "not_running")
	k.createNursing(ana, kid, map[string]any{"side": "left"})
}

// An unswitched running session feeds its side from its start, so moving the start moves side_since with it.
func TestRunningNursingPatchMovesAnUnswitchedSideSince(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	base := "/v1/kids/" + kid + "/nursing"
	for name, start := range map[string]time.Time{"earlier": t0.Add(-30 * time.Minute), "later": t0.Add(-10 * time.Minute)} {
		one := base + "/" + k.createNursing(ana, kid, map[string]any{"side": "left", "started_at": ts(t0.Add(-20 * time.Minute))})
		rec := k.do(http.MethodPatch, one, ana, map[string]any{"started_at": ts(start)})
		if m := bodyMap(t, rec); rec.Code != http.StatusOK || m["started_at"] != ts(start) || m["side_since"] != ts(start) {
			t.Fatalf("%s: patch start: %d %v", name, rec.Code, m)
		}
		rec = k.do(http.MethodPost, one+"/end", ana, map[string]any{})
		if m := bodyMap(t, rec); rec.Code != http.StatusOK || m["left_seconds"] != t0.Sub(start).Seconds() || m["right_seconds"] != 0.0 {
			t.Errorf("%s: end: %d %v, want the whole shown duration on the left", name, rec.Code, m)
		}
	}

	one := base + "/" + k.createNursing(ana, kid, map[string]any{"side": "left", "started_at": ts(t0.Add(-20 * time.Minute))})
	k.do(http.MethodPost, one+"/side", ana, map[string]any{"side": "right", "at": ts(t0.Add(-10 * time.Minute))})
	rec := k.do(http.MethodPatch, one, ana, map[string]any{"started_at": ts(t0.Add(-30 * time.Minute))})
	if m := bodyMap(t, rec); rec.Code != http.StatusOK || m["side_since"] != ts(t0.Add(-10*time.Minute)) || m["left_seconds"] != 600.0 {
		t.Errorf("switched session: %d %v, want side_since and the credited seconds kept", rec.Code, m)
	}
}

// A running timer cannot start in the future: it could be neither ended nor switched until the clock got there.
func TestRunningNursingRefusesAFutureStart(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	base := "/v1/kids/" + kid + "/nursing"
	wantInvalid2(t, "create", k.do(http.MethodPost, base, ana, map[string]any{"side": "left", "started_at": ts(t0.Add(time.Second))}), "started_at")
	one := base + "/" + k.createNursing(ana, kid, map[string]any{"side": "left", "started_at": ts(t0.Add(-time.Minute))})
	wantInvalid2(t, "patch", k.do(http.MethodPatch, one, ana, map[string]any{"started_at": ts(t0.Add(time.Second))}), "started_at")
	if rec := k.do(http.MethodPatch, one, ana, map[string]any{"started_at": ts(t0)}); rec.Code != http.StatusOK {
		t.Errorf("start at now: %d %q", rec.Code, rec.Body.String())
	}
	k.do(http.MethodPost, one+"/end", ana, map[string]any{})
	finished := finishedNursingBody(map[string]any{"started_at": ts(t0.Add(time.Minute)), "ended_at": ts(t0.Add(constants.MaxFutureSkew)), "left_seconds": 0, "right_seconds": 0})
	if rec := k.do(http.MethodPost, base, ana, finished); rec.Code != http.StatusCreated {
		t.Errorf("a finished session keeps the skew: %d %q", rec.Code, rec.Body.String())
	}
}
