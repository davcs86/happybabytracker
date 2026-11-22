package httpapi

import (
	"net/http"
	"testing"
	"time"
)

func (k *kit) summaryDays(token, kid, from, to string) []map[string]any {
	k.t.Helper()
	rec := k.do(http.MethodGet, "/v1/kids/"+kid+"/summary?from="+from+"&to="+to, token, nil)
	if rec.Code != http.StatusOK {
		k.t.Fatalf("summary %s..%s: %d %q", from, to, rec.Code, rec.Body.String())
	}
	m := bodyMap(k.t, rec)
	sameKeys(k.t, m, "days")
	var days []map[string]any
	for _, d := range m["days"].([]any) {
		days = append(days, d.(map[string]any))
	}
	return days
}

func TestSummaryShapeAndZeroDays(t *testing.T) {
	k := newKit(t)
	kid, _, bob := k.trackerKid()
	k.seedEntry(kid, "ana", "bottle", time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC))
	days := k.summaryDays(bob, kid, "2026-11-17", "2026-11-19")
	if len(days) != 3 {
		t.Fatalf("days = %v", days)
	}
	zero := map[string]any{
		"date":    "2026-11-17",
		"bottles": map[string]any{"count": 0, "ml": 0, "floz_hundredths": 0},
		"nursing": map[string]any{"count": 0, "seconds": 0},
		"pumping": map[string]any{"count": 0, "ml": 0, "floz_hundredths": 0},
		"diapers": map[string]any{"total": 0, "wet": 0, "dirty": 0, "both": 0, "dry": 0},
		"sleep":   map[string]any{"seconds": 0},
		"meals":   map[string]any{"count": 0},
	}
	if !equalJSON(days[0], zero) {
		t.Errorf("empty day = %v, want %v", days[0], zero)
	}
	if b := days[1]["bottles"]; days[1]["date"] != "2026-11-18" || !equalJSON(b, map[string]any{"count": 1, "ml": 120, "floz_hundredths": 406}) {
		t.Errorf("18th = %v", days[1])
	}
	if days[2]["date"] != "2026-11-19" {
		t.Errorf("third day = %v", days[2]["date"])
	}
}

func TestSummaryKidZoneNotServerZone(t *testing.T) {
	saved := time.Local
	la, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}
	time.Local = la
	t.Cleanup(func() { time.Local = saved })

	k := newKit(t)
	ana := k.asParent("ana")
	kid := k.createKid(ana, map[string]any{"time_zone": "Pacific/Auckland"})
	// 2026-11-19 12:00 UTC is 04:00 on the 19th in Los Angeles and 01:00 on the 20th in Auckland.
	k.seedEntry(kid, "ana", "diaper", time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC))
	days := k.summaryDays(ana, kid, "2026-11-19", "2026-11-20")
	got := []any{days[0]["diapers"].(map[string]any)["total"], days[1]["diapers"].(map[string]any)["total"]}
	if got[0] != 0.0 || got[1] != 1.0 {
		t.Errorf("diapers per day = %v, want [0 1] on Auckland's days", got)
	}
}

func TestSummaryRangeLimit(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	last := from.AddDate(0, 0, 92).Format(time.DateOnly)
	if days := k.summaryDays(ana, kid, "2026-01-01", last); len(days) != 93 || days[92]["date"] != last {
		t.Fatalf("93-day range = %d days ending %v", len(days), days[len(days)-1]["date"])
	}
	if days := k.summaryDays(ana, kid, "2026-03-04", "2026-03-04"); len(days) != 1 {
		t.Errorf("one-day range = %d days", len(days))
	}
	base := "/v1/kids/" + kid + "/summary"
	for query, field := range map[string]string{
		"?from=2026-01-01&to=" + from.AddDate(0, 0, 93).Format(time.DateOnly): "to",
		"?from=2026-03-05&to=2026-03-04":                                      "from",
		"?to=2026-03-04":                                                      "from",
		"?from=2026-03-04":                                                    "to",
		"?from=2026-3-4&to=2026-03-04":                                        "from",
		"?from=2026-03-04&to=2026-03-04T00:00:00Z":                            "to",
		"?from=2026-02-30&to=2026-03-04":                                      "from",
	} {
		wantInvalid2(t, query, k.do(http.MethodGet, base+query, ana, nil), field)
	}
}
