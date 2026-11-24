package growth

import (
	"strings"
	"testing"
)

func TestAdaptationDisclaimers(t *testing.T) {
	all := []string{"wfa-day", "lhfa-day", "hcfa-day", "bfa-day", "wfa-month", "hfa-month", "bfa-month"}
	got := AdaptationDisclaimers(SourceWHO, all)
	if len(got) != 3 {
		t.Fatalf("who: %d paragraphs, want 3: %q", len(got), got)
	}
	info, _ := SourceInfo(SourceWHO)
	notice := strings.Join(strings.Fields(info.Notice), " ")
	for _, p := range got {
		if !strings.HasPrefix(p, `This is an adaptation of an original work "WHO`) ||
			!strings.HasSuffix(p, "The original edition shall be the binding and authentic edition.") {
			t.Errorf("not a disclaimer paragraph: %q", p)
		}
		if !strings.Contains(notice, p) {
			t.Errorf("not the NOTICE's wording: %q", p)
		}
		for _, banned := range []string{"NOT MIT", "scripts/", "directory", "\n", "  "} {
			if strings.Contains(p, banned) {
				t.Errorf("%q leaks %q", p, banned)
			}
		}
	}
	for _, c := range []struct {
		tables []string
		want   string
	}{
		{[]string{"wfa-day"}, "WHO Child Growth Standards: length/height-for-age"},
		{[]string{"lhfa-day", "bfa-day"}, "WHO Child Growth Standards: length/height-for-age"},
		{[]string{"hcfa-day"}, "WHO Child Growth Standards: head circumference-for-age"},
		{[]string{"bfa-month"}, "WHO Growth Reference 2007 for 5-19 years"},
	} {
		got := AdaptationDisclaimers(SourceWHO, c.tables)
		if len(got) != 1 || !strings.Contains(got[0], c.want) {
			t.Errorf("%v: %q, want the one paragraph naming %q", c.tables, got, c.want)
		}
	}
	if got := AdaptationDisclaimers(SourceWHO, []string{"hcfa-day", "wfa-day", "hcfa-day"}); len(got) != 2 ||
		!strings.Contains(got[0], "length/height-for-age") || !strings.Contains(got[1], "head circumference") {
		t.Errorf("two works: %q, want both once in NOTICE order", got)
	}
	for _, tables := range [][]string{nil, {"unknown"}} {
		if got := AdaptationDisclaimers(SourceWHO, tables); len(got) != 0 {
			t.Errorf("who %v: %q, want none", tables, got)
		}
	}
	cdc := []string{"wtageinf", "wtage", "lenageinf", "statage", "hcageinf", "bmiagerev"}
	if got := AdaptationDisclaimers(SourceCDC, cdc); len(got) != 0 {
		t.Errorf("cdc: %q, want none", got)
	}
}
