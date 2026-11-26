package export

import (
	"github.com/davcs86/happybabytracker/api/internal/export/pdfdoc"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

// cancelledMark follows the kind of a cancelled visit, as the visit page's badge words it.
const cancelledMark = ", cancelled"

// Shares of the content width per visit column.
const (
	visitKindShare     = 0.16
	visitProviderShare = 0.16
	visitPlaceShare    = 0.16
	visitReasonShare   = 0.16
	visitNotesShare    = 1 - timeShare - visitKindShare - visitProviderShare - visitPlaceShare - visitReasonShare
)

var visitCols = []pdfdoc.Col{
	{Title: "Date and time", WidthMM: contentWidth * timeShare},
	{Title: "Kind", WidthMM: contentWidth * visitKindShare},
	{Title: "Doctor or clinic", WidthMM: contentWidth * visitProviderShare},
	{Title: "Place", WidthMM: contentWidth * visitPlaceShare},
	{Title: "Reason", WidthMM: contentWidth * visitReasonShare},
	{Title: "Notes from the visit", WidthMM: contentWidth * visitNotesShare},
}

// visitItems prints the visits starting in the range, oldest first; their linked rows print in their own sections.
func visitItems(snap store.ExportSnapshot, req Request, b *budget) ([]item, error) {
	out := []item{{heading: moduleTitle["visits"]}}
	total := snap.Counts["visits"]
	if total == 0 {
		return append(out, item{paragraph: noEntries}), nil
	}
	s := newSection(b)
	n := min(s.take(total), len(snap.Visits))
	rows := make([][]string, n)
	for i, v := range snap.Visits[:n] {
		kind := model.VisitKindLabel[v.Kind]
		if v.CancelledAt != nil {
			kind += cancelledMark
		}
		rows[i] = []string{FormatInstant(v.StartsAt, req.Loc), kind, optional(v.Provider), optional(v.Location), optional(v.Note),
			optional(v.OutcomeNotes)}
	}
	return append(out, s.shown(visitCols, rows, total)...), nil
}
