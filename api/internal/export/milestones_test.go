package export

import (
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/davcs86/happybabytracker/api/internal/milestones"
	"github.com/davcs86/happybabytracker/api/internal/model"
)

// TestChecklistTitleAndDomainLabel pins the table of ui/src/lib/milestones.test.ts.
func TestChecklistTitleAndDomainLabel(t *testing.T) {
	titles := map[int]string{2: "By 2 months", 4: "By 4 months", 6: "By 6 months", 9: "By 9 months", 12: "By 1 year",
		15: "By 15 months", 18: "By 18 months", 24: "By 2 years", 30: "By 30 months", 36: "By 3 years", 48: "By 4 years", 60: "By 5 years"}
	if len(titles) != 12 {
		t.Fatal("the table lists every catalog age")
	}
	for age, want := range titles {
		if got := ChecklistTitle(age); got != want {
			t.Errorf("ChecklistTitle(%d) = %q, want %q", age, got, want)
		}
	}
	labels := map[milestones.Domain]string{
		milestones.DomainSocialEmotional: "Social/Emotional",
		milestones.DomainLanguage:        "Language/Communication",
		milestones.DomainCognitive:       "Cognitive (learning, thinking, problem-solving)",
		milestones.DomainMovement:        "Movement/Physical Development",
	}
	for d, want := range labels {
		if got := DomainLabel(d); got != want {
			t.Errorf("DomainLabel(%s) = %q, want %q", d, got, want)
		}
	}
}

func firstItem(t *testing.T, c milestones.Catalog, age int) milestones.Item {
	t.Helper()
	for _, it := range c.Items {
		if it.AgeMonths == age && it.Retired == "" {
			return it
		}
	}
	t.Fatalf("no item at %d months", age)
	return milestones.Item{}
}

func reachedRow(id, item, on string, note *string) model.Milestone {
	return model.Milestone{Entry: model.Entry{ID: id, Note: note}, Item: &item, ReachedOn: on}
}

func TestMilestonesSection(t *testing.T) {
	c, ok := milestones.Get()
	if !ok {
		t.Fatal("catalog did not load")
	}
	it2, it12 := firstItem(t, c, 2), firstItem(t, c, 12)
	snap := vaccineSnap(nil, nil)
	snap.Kid.BirthDate = "2025-11-01"
	// The store returns rows newest first; the section prints them oldest first.
	snap.Milestones = []model.Milestone{
		reachedRow("r2", it12.ID, "2026-11-02", nil),
		{Entry: model.Entry{ID: "c1", Note: ptr("at grandma's")}, Text: ptr("Waves bye"), ReachedOn: "2026-10-01"},
		reachedRow("r1", it2.ID, "2026-01-10", ptr("first smile")),
	}
	strs := renderSnap(t, snap, "2026-11-20", "milestones")
	text := joined(strs)
	wantIn(t, "milestones", text, c.ScreeningNote, c.Attribution, "2026-01-10", ChecklistTitle(2), DomainLabel(it2.Domain), it2.Text,
		"first smile", "2026-11-02", ChecklistTitle(12), DomainLabel(it12.Domain), it12.Text, "Your milestones", "Waves bye", "at grandma's")
	head := strings.Index(text, "Milestones "+c.ScreeningNote+" "+c.Attribution)
	if head < 0 || head > strings.Index(text, it2.Text) {
		t.Error("the sentence and the attribution do not follow the heading, before the rows")
	}
	if !(strings.Index(text, it2.Text) < strings.Index(text, it12.Text) && strings.Index(text, it12.Text) < strings.Index(text, "Waves bye")) {
		t.Error("catalog rows are not oldest first, or custom rows do not follow them")
	}

	rest := text
	for _, s := range []string{c.ScreeningNote, c.Attribution, it2.Text, it12.Text} {
		rest = strings.ReplaceAll(rest, s, "")
	}
	for _, it := range c.Items {
		if it.ID != it2.ID && it.ID != it12.ID && strings.Contains(rest, it.Text) {
			t.Errorf("unreached item %s appears", it.ID)
		}
	}
	for _, o := range c.Omitted {
		wantNotIn(t, "omitted", text, o)
	}
	wantNotIn(t, "no count", text, "not reached", "NOTICE", "LICENSE", "this directory")
	if m := regexp.MustCompile(`\b\d+ (items?|milestones?|reached)\b`).FindString(rest); m != "" {
		t.Errorf("a count appears: %q", m)
	}

	appText := text
	for _, it := range c.Items {
		appText = strings.ReplaceAll(appText, it.Text, "")
	}
	appText = strings.ReplaceAll(strings.ReplaceAll(appText, c.ScreeningNote, ""), c.Attribution, "")
	deny := regexp.MustCompile(`(?i)\b(late|behind|delayed|delay|normal|abnormal|typical|atypical|on track|ahead|missed|missing|concern|worry|should|red flag|warning)\b`)
	if m := deny.FindString(appText); m != "" {
		t.Errorf("%q judges", m)
	}

	empty := joined(renderSnap(t, vaccineSnap(nil, nil), "2026-11-20", "milestones"))
	wantIn(t, "empty", empty, c.Attribution+" "+noMilestones)
}

func TestMilestonesRetiredAndUnknown(t *testing.T) {
	c, _ := milestones.Get()
	it2 := firstItem(t, c, 2)
	for i := range c.Items {
		if c.Items[i].ID == it2.ID {
			c.Items[i].Retired = "2026-11-24"
		}
	}
	rows := []model.Milestone{reachedRow("r1", it2.ID, "2026-01-10", ptr("kept"))}
	items, err := milestoneSection(c, rows, "2026-11-20", &budget{left: maxRows})
	if err != nil {
		t.Fatal(err)
	}
	var texts []string
	for _, it := range items {
		texts = append(texts, it.texts()...)
	}
	text := strings.Join(texts, " ")
	wantIn(t, "retired", text, retiredTitle, it2.Text, "2026-01-10", "kept")
	wantNotIn(t, "retired", text, ChecklistTitle(2), noMilestones)

	rows = []model.Milestone{reachedRow("r1", "99m-se-gone", "2026-01-10", nil)}
	if _, err := milestoneSection(c, rows, "2026-11-20", &budget{left: maxRows}); !errors.Is(err, errUnknownMilestoneItem) {
		t.Errorf("an item missing from the catalog: %v, want errUnknownMilestoneItem", err)
	}
}
