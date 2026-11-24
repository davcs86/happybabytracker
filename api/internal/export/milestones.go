package export

import (
	"cmp"
	"errors"
	"slices"

	"github.com/davcs86/happybabytracker/api/internal/export/pdfdoc"
	"github.com/davcs86/happybabytracker/api/internal/milestones"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

// errUnknownMilestoneItem: a stored row names an item the catalog no longer holds, as the API's 500.
var errUnknownMilestoneItem = errors.New("export: stored milestone item not in the catalog")

// The screen's words: milestones/+page.svelte.
const (
	noMilestones = "No milestones recorded"
	customTitle  = "Your milestones"
	retiredTitle = "Reached, no longer in the lists"
)

// Shares of the content width per milestone column.
const (
	milestoneDateShare      = 0.14
	milestoneChecklistShare = 0.14
	milestoneDomainShare    = 0.18
	milestoneTextShare      = 0.34
	milestoneNoteShare      = 0.2
	milestoneOwnTextShare   = 1 - milestoneDateShare - milestoneNoteShare
)

var (
	reachedCols = []pdfdoc.Col{
		{Title: "Date", WidthMM: contentWidth * milestoneDateShare},
		{Title: "Checklist", WidthMM: contentWidth * milestoneChecklistShare},
		{Title: "Domain", WidthMM: contentWidth * milestoneDomainShare},
		{Title: "Milestone", WidthMM: contentWidth * milestoneTextShare},
		{Title: "Note", WidthMM: contentWidth * milestoneNoteShare},
	}
	customCols  = ownCols(customTitle)
	retiredCols = ownCols(retiredTitle)
)

func ownCols(title string) []pdfdoc.Col {
	return []pdfdoc.Col{
		{Title: "Date", WidthMM: contentWidth * milestoneDateShare},
		{Title: title, WidthMM: contentWidth * milestoneOwnTextShare},
		{Title: "Note", WidthMM: contentWidth * milestoneNoteShare},
	}
}

// checklistTitle and domainLabel mirror ui/src/lib/milestones.ts: CDC's own labels.
var (
	checklistTitle = map[int]string{2: "By 2 months", 4: "By 4 months", 6: "By 6 months", 9: "By 9 months", 12: "By 1 year",
		15: "By 15 months", 18: "By 18 months", 24: "By 2 years", 30: "By 30 months", 36: "By 3 years", 48: "By 4 years", 60: "By 5 years"}
	domainLabel = map[milestones.Domain]string{
		milestones.DomainSocialEmotional: "Social/Emotional",
		milestones.DomainLanguage:        "Language/Communication",
		milestones.DomainCognitive:       "Cognitive (learning, thinking, problem-solving)",
		milestones.DomainMovement:        "Movement/Physical Development",
	}
)

// ChecklistTitle is CDC's label of the checklist for ageMonths, as checklistTitle words it.
func ChecklistTitle(ageMonths int) string { return checklistTitle[ageMonths] }

// DomainLabel is CDC's domain heading without "Milestones", as domainLabel words it.
func DomainLabel(d milestones.Domain) string { return domainLabel[d] }

func milestoneItems(snap store.ExportSnapshot, req Request, b *budget) ([]item, error) {
	c, ok := milestones.Get()
	if !ok {
		return nil, milestones.ErrCatalogUnavailable
	}
	return milestoneSection(c, milestones.Lookup, snap.Milestones, req.To, b)
}

// milestoneSection prints CDC's sentence and attribution at body size, then the rows reached up to to, oldest first:
// catalog items, custom rows, retired items. Unreached items never appear, nor their count.
func milestoneSection(c milestones.Catalog, lookup func(string) (milestones.Item, bool), rows []model.Milestone, to string,
	b *budget) ([]item, error) {
	out := []item{{heading: moduleTitle["milestones"]}, {body: c.ScreeningNote}, {body: c.Attribution}}
	var listed []model.Milestone
	for _, m := range rows {
		if m.ReachedOn <= to {
			listed = append(listed, m)
		}
	}
	slices.SortFunc(listed, func(a, b model.Milestone) int {
		return cmp.Or(cmp.Compare(a.ReachedOn, b.ReachedOn), cmp.Compare(a.ID, b.ID))
	})
	var reached, custom, retired [][]string
	for _, m := range listed {
		note := ""
		if m.Note != nil {
			note = Text(*m.Note)
		}
		if m.Text != nil {
			custom = append(custom, []string{FormatDate(m.ReachedOn), Text(*m.Text), note})
			continue
		}
		it, ok := lookup(*m.Item)
		switch {
		case !ok:
			return nil, errUnknownMilestoneItem
		case it.Retired != "":
			retired = append(retired, []string{FormatDate(m.ReachedOn), it.Text, note})
		default:
			reached = append(reached, []string{FormatDate(m.ReachedOn), ChecklistTitle(it.AgeMonths), DomainLabel(it.Domain), it.Text, note})
		}
	}
	if len(listed) == 0 {
		return append(out, item{paragraph: noMilestones}), nil
	}
	s := newSection(b)
	for _, t := range []struct {
		cols []pdfdoc.Col
		rows [][]string
	}{{reachedCols, reached}, {customCols, custom}, {retiredCols, retired}} {
		if len(t.rows) > 0 {
			out = append(out, s.table(t.cols, t.rows)...)
		}
	}
	return out, nil
}
