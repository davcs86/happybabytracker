package httpapi

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/units"
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

var nursingKeys = []string{"id", "started_at", "ended_at", "left_seconds", "right_seconds", "side", "side_since", "note",
	"weight_before", "weight_after", "milk"}

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

func (k *kit) nursingPath(kid string, id ...string) string {
	path := "/v1/kids/" + kid + "/nursing"
	if len(id) > 0 {
		path += "/" + id[0]
	}
	return path
}

func weighedBody(over map[string]any) map[string]any {
	m := finishedNursingBody(nil)
	m["weight_before"], m["weight_after"] = amountBody("kg", 3400), amountBody("kg", 3462)
	return withOverrides(m, over)
}

func TestNursingWeighIns(t *testing.T) {
	k := newKit(t)
	kid, ana, bo := k.trackerKid()
	base := k.nursingPath(kid)

	rec := k.do(http.MethodPost, base, ana, weighedBody(nil))
	m := bodyMap(t, rec)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create with both: %d %q", rec.Code, rec.Body.String())
	}
	sameKeys(t, m, nursingKeys...)
	want := map[string]any{
		"weight_before": map[string]any{"grams": 3400.0, "oz_hundredths": 11993.0, "typed_unit": "kg"},
		"weight_after":  map[string]any{"grams": 3462.0, "oz_hundredths": 12212.0, "typed_unit": "kg"},
		"milk": map[string]any{
			"gain":   map[string]any{"grams": 62.0, "oz_hundredths": 219.0},
			"volume": map[string]any{"ml": 60.0, "floz_hundredths": 204.0},
		},
	}
	for key, v := range want {
		if !equalJSON(m[key], v) {
			t.Errorf("%s = %v, want %v", key, m[key], v)
		}
	}
	finished := k.nursingPath(kid, m["id"].(string))
	if got := bodyMap(t, k.do(http.MethodGet, finished, bo, nil)); !equalJSON(got, m) {
		t.Errorf("GET = %v, want the created view %v", got, m)
	}

	runningID := k.createNursing(ana, kid, map[string]any{"side": "left", "weight_before": amountBody("lb", 12000)})
	running := k.nursingPath(kid, runningID)
	rec = k.do(http.MethodGet, running, ana, nil)
	if v := bodyMap(t, rec); v["weight_before"].(map[string]any)["typed_unit"] != "lb" || v["weight_after"] != nil || v["milk"] != nil {
		t.Errorf("running with before only: %v", v)
	}
	rec = k.do(http.MethodPatch, running, bo, map[string]any{"weight_after": amountBody("lb", 12220)})
	if v := bodyMap(t, rec); rec.Code != http.StatusOK || v["milk"] == nil || v["ended_at"] != nil || v["side"] != "left" {
		t.Fatalf("patch after on a running session: %d %v", rec.Code, v)
	}
	rec = k.do(http.MethodPatch, running, ana, map[string]any{"weight_before": nil})
	if v := bodyMap(t, rec); rec.Code != http.StatusOK || v["weight_before"] != nil || v["weight_after"] == nil || v["milk"] != nil {
		t.Errorf("patch null clears before: %d %v", rec.Code, v)
	}
	rec = k.do(http.MethodPatch, running, ana, map[string]any{"note": "kept"})
	if v := bodyMap(t, rec); rec.Code != http.StatusOK || v["weight_after"] == nil || v["note"] != "kept" {
		t.Errorf("absent keeps the weights: %d %v", rec.Code, v)
	}
	rec = k.do(http.MethodPatch, finished, ana, map[string]any{"weight_before": nil, "weight_after": nil})
	if v := bodyMap(t, rec); rec.Code != http.StatusOK || v["weight_before"] != nil || v["weight_after"] != nil || v["milk"] != nil {
		t.Errorf("patch null clears both on a finished session: %d %v", rec.Code, v)
	}
	// A weigh-in does not stop the one-tap End, and the end keeps the weights.
	rec = k.do(http.MethodPost, running+"/end", ana, map[string]any{})
	if v := bodyMap(t, rec); rec.Code != http.StatusOK || v["weight_after"] == nil || v["ended_at"] == nil {
		t.Errorf("end keeps the weights: %d %v", rec.Code, v)
	}
}

func TestMilkEstimate(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	for name, c := range map[string]struct {
		before, after map[string]any
		gain          float64
		oz, ml, floz  float64
	}{
		"kg":      {amountBody("kg", 3400), amountBody("kg", 3462), 62, 219, 60, 204},
		"lb":      {amountBody("lb", 12000), amountBody("lb", 12220), 62, 220, 60, 204},
		"no gain": {amountBody("kg", 3400), amountBody("kg", 3400), 0, 0, 0, 0},
		"mixed":   {amountBody("kg", 3400), amountBody("lb", 12000), 2, 7, 2, 7},
	} {
		rec := k.do(http.MethodPost, k.nursingPath(kid), ana, weighedBody(map[string]any{"weight_before": c.before, "weight_after": c.after}))
		m, _ := bodyMap(t, rec)["milk"].(map[string]any)
		if rec.Code != http.StatusCreated || m == nil {
			t.Errorf("%s: %d %q", name, rec.Code, rec.Body.String())
			continue
		}
		gain, vol := m["gain"].(map[string]any), m["volume"].(map[string]any)
		if gain["grams"] != c.gain || gain["oz_hundredths"] != c.oz || vol["ml"] != c.ml || vol["floz_hundredths"] != c.floz {
			t.Errorf("%s: milk = %v, want gain %v g %v oz_h, %v ml %v floz_h", name, m, c.gain, c.oz, c.ml, c.floz)
		}
	}
}

func TestAfterBelowBefore(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	rec := k.do(http.MethodPost, k.nursingPath(kid), ana, weighedBody(map[string]any{"weight_before": amountBody("kg", 3500), "weight_after": amountBody("kg", 3450)}))
	m := bodyMap(t, rec)
	if rec.Code != http.StatusCreated || m["milk"] != nil || m["weight_before"].(map[string]any)["grams"] != 3500.0 || m["weight_after"].(map[string]any)["grams"] != 3450.0 {
		t.Fatalf("after below before: %d %v", rec.Code, m)
	}
	one := k.nursingPath(kid, m["id"].(string))
	rec = k.do(http.MethodPatch, one, ana, map[string]any{"weight_after": amountBody("kg", 3400)})
	if v := bodyMap(t, rec); rec.Code != http.StatusOK || v["milk"] != nil || v["weight_after"].(map[string]any)["grams"] != 3400.0 {
		t.Errorf("patch after below before: %d %v", rec.Code, v)
	}
	if v := bodyMap(t, k.do(http.MethodGet, one, ana, nil)); v["milk"] != nil || v["weight_before"] == nil || v["weight_after"] == nil {
		t.Errorf("stored: %v", v)
	}
}

func TestWeightValidation(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	finished := k.nursingPath(kid, k.createNursing(ana, kid, finishedNursingBody(nil)))
	running := k.nursingPath(kid, k.createNursing(ana, kid, map[string]any{"side": "left"}))
	for _, field := range []string{"weight_before", "weight_after"} {
		for name, v := range map[string]any{
			"below the minimum":  amountBody("kg", constants.MinWeightGrams-1),
			"above the maximum":  amountBody("kg", constants.MaxWeightGrams+1),
			"lb below":           amountBody("lb", 700),
			"lb above":           amountBody("lb", 882000),
			"zero":               amountBody("kg", 0),
			"negative":           amountBody("kg", -3400),
			"unknown unit":       amountBody("stone", 3400),
			"no amount":          map[string]any{"unit": "kg"},
			"a bare number":      3400,
			"a string":           "heavy",
			"amount is a string": map[string]any{"unit": "kg", "amount": "3400"},
			"amount too large":   amountBody("kg", constants.MaxWeightAmount+1),
		} {
			wantInvalid2(t, field+" "+name+" on create", k.do(http.MethodPost, k.nursingPath(kid), ana, weighedBody(map[string]any{field: v})), field)
			wantInvalid2(t, field+" "+name+" on a finished patch", k.do(http.MethodPatch, finished, ana, map[string]any{field: v}), field)
			wantInvalid2(t, field+" "+name+" on a running patch", k.do(http.MethodPatch, running, ana, map[string]any{field: v}), field)
		}
		for name, v := range map[string]any{
			"the minimum": amountBody("kg", constants.MinWeightGrams), "the maximum": amountBody("kg", constants.MaxWeightGrams),
			"lb minimum": amountBody("lb", 710), "lb maximum": amountBody("lb", 881800),
		} {
			if rec := k.do(http.MethodPatch, finished, ana, map[string]any{field: v}); rec.Code != http.StatusOK {
				t.Errorf("%s %s: %d %q", field, name, rec.Code, rec.Body.String())
			}
		}
	}
	// Order: side, then weight_before, weight_after, note.
	bad, noteBad := amountBody("kg", 1), strings.Repeat("a", constants.MaxNoteLen+1)
	wantInvalid2(t, "before before after", k.do(http.MethodPatch, finished, ana, map[string]any{"weight_after": bad, "weight_before": bad}), "weight_before")
	wantInvalid2(t, "after before note", k.do(http.MethodPatch, finished, ana, map[string]any{"weight_after": bad, "note": noteBad}), "weight_after")
	wantInvalid2(t, "side before weight", k.do(http.MethodPatch, finished, ana, map[string]any{"side": "left", "weight_before": bad}), "side")
	wantInvalid2(t, "create: side before weight", k.do(http.MethodPost, k.nursingPath(kid), ana, finishedNursingBody(map[string]any{"side": "left", "weight_before": bad})), "side")
	wantInvalid2(t, "create: weight before note", k.do(http.MethodPost, k.nursingPath(kid), ana, finishedNursingBody(map[string]any{"weight_after": bad, "note": noteBad})), "weight_after")
	wantInvalid2(t, "create: seconds before weight", k.do(http.MethodPost, k.nursingPath(kid), ana, finishedNursingBody(map[string]any{"weight_before": bad, "right_seconds": nil})), "right_seconds")
	// An unknown key inside a weight is the same refusal as in a growth measure.
	rec := k.do(http.MethodPost, k.nursingPath(kid), ana, `{"started_at":"2026-11-20T11:00:00Z","ended_at":"2026-11-20T11:10:00Z","left_seconds":1,"right_seconds":1,"weight_before":{"unit":"kg","amount":3400,"precise":true}}`)
	if rec.Code != http.StatusUnprocessableEntity || strings.TrimSpace(rec.Body.String()) != `{"error":"unknown_field","field":"precise"}` {
		t.Errorf("unknown key in a weight: %d %q", rec.Code, rec.Body.String())
	}
}

// A re-sent typed value keeps the stored three columns, as the growth patch does.
func TestUnchangedTypedWeightKeepsStoredColumns(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	end := t0.Add(-50 * time.Minute)
	odd := &units.Weight{Grams: 3400, OzHundredths: 12000, Typed: units.UnitKG} // oz column deliberately not the conversion
	n, err := k.Store.CreateNursing(context.Background(), kid, k.parentID("ana"),
		model.NursingSession{StartedAt: t0.Add(-time.Hour), EndedAt: &end, LeftSeconds: 1, RightSeconds: 1, WeightBefore: odd}, k.now)
	if err != nil {
		t.Fatal(err)
	}
	one := k.nursingPath(kid, n.ID)
	k.do(http.MethodPatch, one, ana, map[string]any{"weight_before": amountBody("kg", 3400), "note": "same"})
	if got, _ := k.Store.Nursing(context.Background(), kid, n.ID); got.WeightBefore == nil || *got.WeightBefore != *odd {
		t.Errorf("re-sent typed value rewrote the columns: %+v, want %+v", got.WeightBefore, odd)
	}
	k.do(http.MethodPatch, one, ana, map[string]any{"weight_before": amountBody("kg", 3401)})
	if got, _ := k.Store.Nursing(context.Background(), kid, n.ID); got.WeightBefore == nil || got.WeightBefore.Grams != 3401 || got.WeightBefore.OzHundredths == 12000 {
		t.Errorf("a changed value was not converted: %+v", got.WeightBefore)
	}
}

func TestWeighInIsNotGrowth(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	k.createGrowth(ana, kid, nil)
	growth := func() string { return k.do(http.MethodGet, "/v1/kids/"+kid+"/growth", ana, nil).Body.String() }
	before := growth()

	id := bodyMap(t, k.do(http.MethodPost, k.nursingPath(kid), ana, weighedBody(nil)))["id"].(string)
	k.do(http.MethodPatch, k.nursingPath(kid, id), ana, map[string]any{"weight_after": amountBody("kg", 3500)})
	k.createNursing(ana, kid, map[string]any{"side": "right", "weight_before": amountBody("kg", 3400)})
	if got := growth(); got != before {
		t.Errorf("growth list changed after weigh-ins:\n got %s\nwant %s", got, before)
	}
	rows, err := k.Store.ListGrowth(context.Background(), kid)
	if err != nil || len(rows) != 1 {
		t.Errorf("growth rows = %d, %v; want the one seeded", len(rows), err)
	}
}

func TestNursingViewKeys(t *testing.T) {
	k := newKit(t)
	kid, ana, bo := k.trackerKid()
	id := k.createNursing(ana, kid, map[string]any{"side": "left", "weight_before": amountBody("kg", 3400)})
	view := bodyMap(t, k.do(http.MethodGet, k.nursingPath(kid, id), ana, nil))
	sameKeys(t, view, nursingKeys...)

	running := k.home(bo, kid)["running"].(map[string]any)["nursing"].(map[string]any)
	sameKeys(t, running, nursingKeys...)
	if !equalJSON(running, view) {
		t.Errorf("home running nursing = %v, want %v", running, view)
	}

	k.do(http.MethodPost, k.nursingPath(kid, id)+"/end", ana, map[string]any{})
	k.do(http.MethodPatch, k.nursingPath(kid, id), ana, map[string]any{"weight_after": amountBody("kg", 3470)})
	view = bodyMap(t, k.do(http.MethodGet, k.nursingPath(kid, id), ana, nil))
	_, items, _ := k.timelinePage(bo, kid, "?types=nursing")
	if len(items) != 1 {
		t.Fatalf("timeline items = %v", items)
	}
	sameKeys(t, items[0], append([]string{"type", "at"}, nursingKeys...)...)
	for key, v := range view {
		if !equalJSON(items[0][key], v) {
			t.Errorf("timeline %s = %v, the tracker's view has %v", key, items[0][key], v)
		}
	}
	list := bodyMap(t, k.do(http.MethodGet, k.nursingPath(kid), ana, nil))["nursing"].([]any)
	if len(list) != 1 || !equalJSON(list[0], view) {
		t.Errorf("list = %v, want %v", list, view)
	}
}

func TestNursingWeightsNoValuesInLogs(t *testing.T) {
	const note = "xyzzy-private-note"
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	k.Log.Reset()

	id := bodyMap(t, k.do(http.MethodPost, k.nursingPath(kid), ana, weighedBody(map[string]any{
		"note": note, "weight_before": amountBody("kg", 4711), "weight_after": amountBody("kg", 4777)})))["id"].(string)
	k.do(http.MethodGet, k.nursingPath(kid, id), ana, nil)
	k.do(http.MethodPatch, k.nursingPath(kid, id), ana, map[string]any{"weight_after": amountBody("lb", 17391)})
	k.do(http.MethodPatch, k.nursingPath(kid, id), ana, map[string]any{"weight_before": amountBody("kg", 1)}) // a refusal
	k.do(http.MethodPatch, k.nursingPath(kid, id), ana, map[string]any{"weight_before": 4711})                // a refusal
	k.do(http.MethodDelete, k.nursingPath(kid, id), ana, nil)

	wantCleanLog(t, k.Log.String(), 6, "xyzzy", "4711", "4777", "17391", "weight_before", "weight_after")
}
