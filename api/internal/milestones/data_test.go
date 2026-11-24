package milestones

import (
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

func mustLoad(t *testing.T) Catalog {
	t.Helper()
	if err := Load(); err != nil {
		t.Fatalf("Load() = %v", err)
	}
	c, ok := Get()
	if !ok {
		t.Fatal("Get() not ok")
	}
	return c
}

// agePages maps each checklist age to the CDC page its items come from.
var agePages = map[int]string{
	2: "2-months", 4: "4-months", 6: "6-months", 9: "9-months", 12: "1-year", 15: "15-months",
	18: "18-months", 24: "2-years", 30: "30-months", 36: "3-years", 48: "4-years", 60: "5-years",
}

func agePageURL(age int) string {
	return "https://www.cdc.gov/act-early/milestones/" + agePages[age] + ".html"
}

const screeningNote = "Learn the Signs. Act Early. materials are not a substitute for standardized, validated developmental screening tools."

func TestShippedCatalogLoads(t *testing.T) {
	c := mustLoad(t)
	for k, v := range map[string]string{"catalog": c.ID, "name": c.Name, "authority": c.Authority, "edition": c.Edition,
		"retrieved": c.Retrieved, "origin_url": c.OriginURL, "licence": c.Licence, "attribution": c.Attribution,
		"screening_note": c.ScreeningNote} {
		if strings.TrimSpace(v) == "" {
			t.Errorf("%s is empty", k)
		}
	}
	if len(c.Omitted) == 0 {
		t.Error("omitted is empty")
	}
	if c.ID != "cdc" || c.OriginURL != "https://www.cdc.gov/act-early/milestones/index.html" {
		t.Errorf("catalog %q, origin_url %q", c.ID, c.OriginURL)
	}
	if _, err := time.Parse(dateLayout, c.Retrieved); err != nil {
		t.Errorf("retrieved %q is not a date", c.Retrieved)
	}
	for _, want := range []string{"Source: CDC", "does not endorse", "free of charge"} {
		if !strings.Contains(c.Attribution, want) {
			t.Errorf("attribution lacks %q", want)
		}
	}
	if c.ScreeningNote != screeningNote {
		t.Errorf("screening_note = %q, want %q", c.ScreeningNote, screeningNote)
	}
	shipped, counts, active := shippedIDs(t), map[int]int{}, 0
	for _, it := range c.Items {
		if it.Retired == "" {
			counts[it.AgeMonths]++
		}
		if it.SourceURL != agePageURL(it.AgeMonths) {
			t.Errorf("%s: source_url %q, want its age page", it.ID, it.SourceURL)
		}
		if it.Retired != "" {
			if !shipped[it.ID] {
				t.Errorf("%s is retired but was never released (not in %s)", it.ID, shippedIDsFile)
			}
			continue
		}
		active++
	}
	want := map[int]int{2: 11, 4: 13, 6: 12, 9: 13, 12: 10, 15: 13, 18: 15, 24: 13, 30: 15, 36: 12, 48: 17, 60: 15}
	for age, n := range want {
		if counts[age] != n {
			t.Errorf("age %d: %d items, want %d", age, counts[age], n)
		}
	}
	if active != 159 {
		t.Errorf("%d active items, want 159", active)
	}
}

// shippedIDsFile lists every item id ever released, one per line: append-only, and a new id is appended once its
// catalog update is released.
const shippedIDsFile = "testdata/ids.txt"

func shippedIDs(t *testing.T) map[string]bool {
	t.Helper()
	data, err := os.ReadFile(shippedIDsFile)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]bool{}
	for i, id := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
		if !idPattern.MatchString(id) || out[id] {
			t.Fatalf("%s:%d: not an item id, or a duplicate", shippedIDsFile, i+1)
		}
		out[id] = true
	}
	return out
}

// TestShippedIDsStay keeps every released id in the catalog, so stored reached rows never lose their item.
func TestShippedIDsStay(t *testing.T) {
	mustLoad(t)
	for id := range shippedIDs(t) {
		if _, ok := Lookup(id); !ok {
			t.Errorf("%s was released (%s) but is not in the catalog: retire it, never delete or rename it", id, shippedIDsFile)
		}
	}
}

func TestPinnedItems(t *testing.T) {
	c := mustLoad(t)
	for _, p := range []struct {
		age    int
		domain Domain
		text   string
	}{
		{2, DomainSocialEmotional, "Calms down when spoken to or picked up"},
		{2, DomainLanguage, "Makes sounds other than crying"},
		{2, DomainMovement, "Holds head up when on tummy"},
		{9, DomainCognitive, "Looks for objects when dropped out of sight (like his spoon or toy)"},
		{9, DomainMovement, "Gets to a sitting position by herself"},
		{48, DomainLanguage, "Says sentences with four or more words"},
		{48, DomainCognitive, "Draws a person with three or more body parts"},
	} {
		found := false
		for _, it := range c.Items {
			found = found || (it.AgeMonths == p.age && it.Domain == p.domain && it.Text == p.text)
		}
		if !found {
			t.Errorf("no %d-month %s item %q", p.age, p.domain, p.text)
		}
	}
	if it, ok := Lookup("2m-se-calms-down-when"); !ok || it.Text != "Calms down when spoken to or picked up" {
		t.Errorf("Lookup(2m-se-calms-down-when) = %+v, %v", it, ok)
	}
}

func TestEveryAgeHasEveryDomain(t *testing.T) {
	c := mustLoad(t)
	seen := map[int]map[Domain]bool{}
	for _, it := range c.Items {
		if seen[it.AgeMonths] == nil {
			seen[it.AgeMonths] = map[Domain]bool{}
		}
		seen[it.AgeMonths][it.Domain] = true
	}
	for age := range agePages {
		for _, d := range domains {
			if !seen[age][d] {
				t.Errorf("age %d has no %s item", age, d)
			}
		}
	}
}

// judgementWords is the plan's Global Constraints denylist.
var judgementWords = []string{"late", "behind", "delayed", "delay", "normal", "abnormal", "typical", "atypical",
	"on track", "ahead", "missed", "missing", "concern", "worry", "should", "red flag", "warning"}

func TestNoJudgementWords(t *testing.T) {
	c := mustLoad(t)
	for _, w := range judgementWords {
		re := regexp.MustCompile(`(?i)\b` + strings.ReplaceAll(regexp.QuoteMeta(w), " ", `\s+`) + `\b`)
		for _, it := range c.Items {
			if re.MatchString(it.Text) {
				t.Errorf("%s matches %q: STOP and ask the owner; never reword", it.ID, w)
			}
		}
	}
}
