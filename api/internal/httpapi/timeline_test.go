package httpapi

import (
	"cmp"
	"context"
	"net/http"
	"net/url"
	"slices"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/units"
)

// seedEntry stores one entry of typ for kid at at (a session's start) as the parent username and returns its id.
func (k *kit) seedEntry(kid, username, typ string, at time.Time) string {
	k.t.Helper()
	ctx, actor := context.Background(), k.parentID(username)
	vol, err := units.NewVolume(units.UnitML, 120)
	if err != nil {
		k.t.Fatal(err)
	}
	end := at.Add(30 * time.Minute)
	var id string
	switch typ {
	case "bottle":
		var v model.BottleFeed
		v, err = k.Store.CreateBottle(ctx, kid, actor, model.BottleFeed{At: at, Contents: []string{model.ContentFormula}, Volume: vol}, k.now)
		id = v.ID
	case "diaper":
		var v model.Diaper
		v, err = k.Store.CreateDiaper(ctx, kid, actor, model.Diaper{At: at, Kind: "wet"}, k.now)
		id = v.ID
	case "meal":
		var v model.Meal
		v, err = k.Store.CreateMeal(ctx, kid, actor, model.Meal{At: at, What: "pear"}, k.now)
		id = v.ID
	case "symptom":
		var v model.Symptom
		v, err = k.Store.CreateSymptom(ctx, kid, actor, model.Symptom{At: at, Symptom: "cough"}, k.now)
		id = v.ID
	case "pumping":
		var v model.PumpingSession
		v, err = k.Store.CreatePumping(ctx, kid, actor, model.PumpingSession{StartedAt: at, DurationSeconds: 600, Volume: vol, Side: "both"}, k.now)
		id = v.ID
	case "sleep":
		var v model.SleepSession
		v, err = k.Store.CreateSleep(ctx, kid, actor, model.SleepSession{StartedAt: at, EndedAt: &end}, k.now)
		id = v.ID
	case "nursing":
		var v model.NursingSession
		v, err = k.Store.CreateNursing(ctx, kid, actor, model.NursingSession{StartedAt: at, EndedAt: &end, LeftSeconds: 60, RightSeconds: 60}, k.now)
		id = v.ID
	default:
		k.t.Fatalf("unknown type %q", typ)
	}
	if err != nil {
		k.t.Fatalf("seed %s: %v", typ, err)
	}
	return id
}

type timelineRow struct{ Type, ID, At string }

// timelinePage fetches one page and returns its rows and next_cursor ("" when none).
func (k *kit) timelinePage(token, kid, query string) ([]timelineRow, []map[string]any, string) {
	k.t.Helper()
	rec := k.do(http.MethodGet, "/v1/kids/"+kid+"/timeline"+query, token, nil)
	if rec.Code != http.StatusOK {
		k.t.Fatalf("timeline%s: %d %q", query, rec.Code, rec.Body.String())
	}
	m := bodyMap(k.t, rec)
	sameKeys(k.t, m, "items", "next_cursor")
	var rows []timelineRow
	var raw []map[string]any
	for _, it := range m["items"].([]any) {
		item := it.(map[string]any)
		raw = append(raw, item)
		rows = append(rows, timelineRow{item["type"].(string), item["id"].(string), item["at"].(string)})
	}
	next, _ := m["next_cursor"].(string)
	return rows, raw, next
}

var timelineTypeNames = []string{"bottle", "nursing", "pumping", "sleep", "diaper", "meal", "symptom"}

func TestTimelineMergesInOrderAcrossTrackers(t *testing.T) {
	k := newKit(t)
	kid, ana, bob := k.trackerKid()
	order := []string{"sleep", "bottle", "symptom", "nursing", "diaper", "pumping", "meal"}
	var want []timelineRow
	for i, typ := range order {
		at := t0.Add(-time.Duration(i+1) * time.Hour)
		want = append(want, timelineRow{typ, k.seedEntry(kid, "ana", typ, at), ts(at)})
	}
	rows, raw, next := k.timelinePage(bob, kid, "")
	if !slices.Equal(rows, want) || next != "" {
		t.Fatalf("timeline = %v next %q, want %v", rows, next, want)
	}
	shapes := map[string][]string{
		"bottle":  {"type", "id", "at", "contents", "volume", "sources", "note"},
		"diaper":  {"type", "id", "at", "kind", "note"},
		"meal":    {"type", "id", "at", "what", "reaction", "note"},
		"symptom": {"type", "id", "at", "symptom", "severity", "other_text", "temperature", "method", "note"},
		"pumping": {"type", "id", "at", "started_at", "duration_seconds", "volume", "side", "sides", "note"},
		"sleep":   {"type", "id", "at", "started_at", "ended_at", "note"},
		"nursing": {"type", "id", "at", "started_at", "ended_at", "left_seconds", "right_seconds", "side", "side_since", "note", "weight_before", "weight_after", "milk"},
	}
	for _, item := range raw {
		typ := item["type"].(string)
		sameKeys(t, item, shapes[typ]...)
		if started, ok := item["started_at"]; ok && started != item["at"] {
			t.Errorf("%s at %v differs from started_at %v", typ, item["at"], started)
		}
		single := k.do(http.MethodGet, "/v1/kids/"+kid+"/"+map[string]string{
			"bottle": "bottles", "diaper": "diapers", "meal": "meals", "pumping": "pumping", "sleep": "sleep", "nursing": "nursing", "symptom": "symptoms",
		}[typ]+"/"+item["id"].(string), ana, nil)
		view := bodyMap(t, single)
		for key, v := range view {
			if !equalJSON(item[key], v) {
				t.Errorf("%s %s = %v, the tracker's view has %v", typ, key, item[key], v)
			}
		}
	}
}

func TestTimelinePagingWithEqualTimestamps(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	at := t0.Add(-time.Hour)
	k.seedEntry(kid, "ana", "bottle", at.Add(time.Second))
	k.seedEntry(kid, "ana", "meal", at.Add(-time.Second))
	want := map[string]bool{}
	for _, typ := range []string{"bottle", "diaper", "meal", "pumping", "bottle", "diaper", "meal", "pumping", "sleep", "nursing"} {
		want[k.seedEntry(kid, "ana", typ, at)] = true
	}
	from, to := url.QueryEscape(ts(at)), url.QueryEscape(ts(at.Add(time.Second)))
	base := "?limit=3&from=" + from + "&to=" + to
	got := map[string]bool{}
	var sizes []int
	var all []timelineRow
	query := base
	for range 10 {
		rows, _, next := k.timelinePage(ana, kid, query)
		sizes = append(sizes, len(rows))
		for _, r := range rows {
			if got[r.ID] {
				t.Fatalf("%s repeated", r.ID)
			}
			got[r.ID] = true
			all = append(all, r)
		}
		if next == "" {
			break
		}
		query = base + "&cursor=" + url.QueryEscape(next)
	}
	if len(got) != len(want) || !slices.Equal(sizes, []int{3, 3, 3, 1}) {
		t.Fatalf("pages %v hold %d entries, want 10", sizes, len(got))
	}
	for id := range want {
		if !got[id] {
			t.Errorf("%s never returned", id)
		}
	}
	if !slices.IsSortedFunc(all, func(a, b timelineRow) int { return cmp.Compare(b.ID, a.ID) }) {
		t.Errorf("equal timestamps not ordered by id descending: %v", all)
	}
}

func TestTimelineTypesFilter(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	ids := map[string]string{}
	for i, typ := range timelineTypeNames {
		ids[typ] = k.seedEntry(kid, "ana", typ, t0.Add(-time.Duration(i+1)*time.Minute))
	}
	rows, _, _ := k.timelinePage(ana, kid, "?types=sleep,bottle")
	if want := []timelineRow{{"bottle", ids["bottle"], ts(t0.Add(-time.Minute))}, {"sleep", ids["sleep"], ts(t0.Add(-4 * time.Minute))}}; !slices.Equal(rows, want) {
		t.Errorf("types=sleep,bottle = %v, want %v", rows, want)
	}
	if rows, _, _ := k.timelinePage(ana, kid, ""); len(rows) != len(timelineTypeNames) {
		t.Errorf("default types return %d entries, want all %d", len(rows), len(timelineTypeNames))
	}
	for query, field := range map[string]string{
		"?types=bottles":            "types",
		"?types=bottle,poop":        "types",
		"?types=":                   "types",
		"?types=bottle,,sleep":      "types",
		"?types=bottle,bottle":      "types",
		"?types=bottle&types=sleep": "types",
		"?types=Bottle":             "types",
		"?from=yesterday":           "from",
		"?to=2026-11-20":            "to",
		"?from=2026-11-20T12:00:00Z&to=2026-11-20T11:00:00Z": "from",
		"?limit=0":   "limit",
		"?limit=101": "limit",
		"?cursor=!!": "cursor",
	} {
		wantInvalid2(t, query, k.do(http.MethodGet, "/v1/kids/"+kid+"/timeline"+query, ana, nil), field)
	}
}

func TestTimelineRangeIsHalfOpen(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	a, b := t0.Add(-2*time.Hour), t0.Add(-time.Hour)
	inA := k.seedEntry(kid, "ana", "diaper", a)
	k.seedEntry(kid, "ana", "sleep", b)
	k.seedEntry(kid, "ana", "meal", a.Add(-time.Second))
	query := "?from=" + url.QueryEscape(ts(a)) + "&to=" + url.QueryEscape(ts(b))
	rows, _, _ := k.timelinePage(ana, kid, query)
	if len(rows) != 1 || rows[0].ID != inA {
		t.Errorf("[a, b) = %v, want only the diaper at a", rows)
	}
	// Fractional bounds truncate toward zero, as on the tracker lists.
	query = "?from=" + url.QueryEscape(a.Add(500*time.Millisecond).Format(time.RFC3339Nano)) + "&to=" + url.QueryEscape(b.Add(500*time.Millisecond).Format(time.RFC3339Nano))
	if rows, _, _ := k.timelinePage(ana, kid, query); len(rows) != 1 || rows[0].ID != inA {
		t.Errorf("fractional [a, b) = %v, want only the diaper at a", rows)
	}
}

func TestTimelineOtherKidsEntriesNeverAppear(t *testing.T) {
	k := newKit(t)
	kid, ana, bob := k.trackerKid()
	other := k.createKid(ana, map[string]any{"name": "Teo"})
	mine := map[string]bool{}
	for i, typ := range timelineTypeNames {
		at := t0.Add(-time.Duration(i+1) * time.Minute)
		mine[k.seedEntry(kid, "ana", typ, at)] = true
		k.seedEntry(other, "ana", typ, at)
	}
	for _, query := range []string{"", "?limit=2", "?types=" + "bottle,nursing,pumping,sleep,diaper,meal,symptom"} {
		rows, _, _ := k.timelinePage(ana, kid, query)
		for _, r := range rows {
			if !mine[r.ID] {
				t.Errorf("%q: %s %s belongs to another kid", query, r.Type, r.ID)
			}
		}
	}
	wantErr(t, k.do(http.MethodGet, "/v1/kids/"+other+"/timeline", bob, nil), http.StatusNotFound, "not_found")
	wantErr(t, k.do(http.MethodGet, "/v1/kids/"+other+"/summary?from=2026-11-20&to=2026-11-20", bob, nil), http.StatusNotFound, "not_found")
}

func TestTimelineSymptomType(t *testing.T) {
	k := newKit(t)
	kid, ana, bob := k.trackerKid()
	other := k.createKid(ana, map[string]any{"name": "Teo"})
	fever := k.createSymptom(ana, kid, symptomBody(feverBody(map[string]any{"at": ts(t0.Add(-2 * time.Minute))})))["id"].(string)
	k.createSymptom(ana, other, symptomBody(nil))
	k.seedEntry(kid, "ana", "diaper", t0.Add(-time.Minute))

	rows, raw, _ := k.timelinePage(bob, kid, "?types=symptom")
	if want := []timelineRow{{"symptom", fever, ts(t0.Add(-2 * time.Minute))}}; !slices.Equal(rows, want) {
		t.Fatalf("types=symptom = %v, want %v", rows, want)
	}
	if temp, _ := raw[0]["temperature"].(map[string]any); temp["typed_unit"] != "c" || temp["c_hundredths"] != float64(3850) || raw[0]["method"] != "oral" {
		t.Errorf("symptom item = %v", raw[0])
	}
	if rows, _, _ := k.timelinePage(bob, kid, ""); len(rows) != 2 || rows[1].Type != "symptom" {
		t.Errorf("default types = %v, want the diaper and the symptom", rows)
	}
}

func TestTimelineDefaultTypesFollowTheSicknessModule(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	k.seedEntry(kid, "ana", "symptom", t0.Add(-time.Minute))
	k.seedEntry(kid, "ana", "diaper", t0.Add(-2*time.Minute))

	k.setModule(ana, kid, "sickness", false)
	rows, _, _ := k.timelinePage(ana, kid, "")
	if len(rows) != 1 || rows[0].Type != "diaper" {
		t.Errorf("sickness off: default types = %v, want only the diaper", rows)
	}
	if rows, _, _ := k.timelinePage(ana, kid, "?types=symptom"); len(rows) != 1 || rows[0].Type != "symptom" {
		t.Errorf("sickness off: types=symptom = %v, an explicit type is still served", rows)
	}
	k.setModule(ana, kid, "sickness", true)
	if rows, _, _ := k.timelinePage(ana, kid, ""); len(rows) != 2 {
		t.Errorf("sickness on: default types = %v, want both", rows)
	}
}

func TestTimelineListsVisits(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	v := k.createVisit(ana, kid, visitBody(map[string]any{"starts_at": ts(t0.Add(-time.Hour))}))
	k.createVisit(ana, kid, visitBody(map[string]any{"starts_at": ts(t0.Add(time.Hour))}))
	items := bodyMap(t, k.do(http.MethodGet, "/v1/kids/"+kid+"/timeline?types=visit", ana, nil))["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("visit items: %v, want only the past one", items)
	}
	it := items[0].(map[string]any)
	sameKeys(t, it, append([]string{"type", "at"}, visitKeys...)...)
	if it["type"] != "visit" || it["id"] != v["id"] || it["at"] != ts(t0.Add(-time.Hour)) || it["status"] != "past" {
		t.Errorf("visit item %v", it)
	}
	k.setModule(ana, kid, "visits", false)
	for _, it := range bodyMap(t, k.do(http.MethodGet, "/v1/kids/"+kid+"/timeline", ana, nil))["items"].([]any) {
		if it.(map[string]any)["type"] == "visit" {
			t.Errorf("a visit in the default timeline with the module off")
		}
	}
}
