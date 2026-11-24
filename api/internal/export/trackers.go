package export

import (
	"strconv"
	"strings"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/export/pdfdoc"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/store"
	"github.com/davcs86/happybabytracker/api/internal/units"
)

// Shares of the content width per column.
const (
	contentWidth = constants.ExportPageWidthMM - 2*constants.ExportMarginMM
	timeShare    = 0.2
	entryShare   = 0.5
	noteShare    = 0.3
	dateShare    = 0.2
)

const (
	noEntries = "No entries in this range."
	overflow  = " more entries in this range are not shown. Export a shorter range to see them."
	running   = "running"
	dayTotal  = "Day total"
)

var entryCols = []pdfdoc.Col{
	{Title: "Time", WidthMM: contentWidth * timeShare},
	{Title: "Entry", WidthMM: contentWidth * entryShare},
	{Title: "Note", WidthMM: contentWidth * noteShare},
}

// contentLabel and sideInitial word the stored names as ui/src/lib/timeline.ts does.
var (
	contentLabel = map[string]string{
		model.ContentBreastMilk: "breast milk", model.ContentFormula: "formula", model.ContentCowMilk: "cow's milk", model.ContentOther: "other",
	}
	sideInitial = map[string]string{"left": "L", "right": "R"}
)

// tracker is one daily-tracker section: its entries in the range, then its daily summary.
type tracker struct {
	typ, title string
	line       func(it store.TimelineItem, req Request) string
	day        func(d store.DaySummary, u model.Units) string // "" for a day with nothing
}

func (tr tracker) items(snap store.ExportSnapshot, req Request, b *budget) []item {
	out := []item{{heading: tr.title}}
	entries, count := snap.Entries[tr.typ], snap.Counts[tr.typ]
	shown := min(b.take(count), len(entries))
	if count > 0 && shown == 0 {
		return append(out, item{paragraph: strconv.Itoa(count) + overflow})
	}
	if count == 0 {
		out = append(out, item{paragraph: noEntries})
	} else {
		rows := make([][]string, shown)
		for i, it := range entries[:shown] {
			rows[i] = []string{FormatInstant(it.At, req.Loc), tr.line(it, req), note(it.Entry)}
		}
		out = append(out, item{cols: entryCols, rows: rows})
		if count > shown {
			out = append(out, item{paragraph: strconv.Itoa(count-shown) + overflow})
		}
	}
	var days [][]string
	for _, d := range snap.Days {
		if v := tr.day(d, req.Units); v != "" {
			days = append(days, []string{FormatDate(d.Date), v})
		}
	}
	if len(days) > 0 {
		cols := []pdfdoc.Col{{Title: "Date", WidthMM: contentWidth * dateShare}, {Title: dayTotal, WidthMM: contentWidth * (1 - dateShare)}}
		out = append(out, item{cols: cols, rows: days})
	}
	return out
}

func note(entry any) string {
	var n *string
	switch e := entry.(type) {
	case model.BottleFeed:
		n = e.Note
	case model.NursingSession:
		n = e.Note
	case model.PumpingSession:
		n = e.Note
	case model.SleepSession:
		n = e.Note
	case model.Diaper:
		n = e.Note
	case model.Meal:
		n = e.Note
	}
	if n == nil {
		return ""
	}
	return Text(*n)
}

func countAnd(n int64, rest string) string {
	if n == 0 {
		return ""
	}
	return strconv.FormatInt(n, 10) + " · " + rest
}

var bottles = tracker{typ: "bottle", title: "Bottles",
	line: func(it store.TimelineItem, req Request) string {
		b := it.Entry.(model.BottleFeed)
		parts := make([]string, 0, len(b.Contents))
		if len(b.Sources) > 0 {
			for _, s := range b.Sources {
				parts = append(parts, contentLabel[s.Source]+" "+FormatVolume(s.Volume, req.Units))
			}
		} else {
			for _, c := range b.Contents {
				parts = append(parts, contentLabel[c])
			}
		}
		return "Bottle " + FormatVolume(b.Volume, req.Units) + ", " + strings.Join(parts, " + ")
	},
	day: func(d store.DaySummary, u model.Units) string {
		return countAnd(d.Bottles.Count, FormatVolume(units.Volume{ML: d.Bottles.ML, FlOzHundredths: d.Bottles.FlOzHundredths}, u))
	},
}

var nursing = tracker{typ: "nursing", title: "Nursing",
	line: func(it store.TimelineItem, req Request) string {
		n := it.Entry.(model.NursingSession)
		var sides []string
		if n.LeftSeconds > 0 {
			sides = append(sides, sideInitial["left"]+" "+FormatDuration(n.LeftSeconds))
		}
		if n.RightSeconds > 0 {
			sides = append(sides, sideInitial["right"]+" "+FormatDuration(n.RightSeconds))
		}
		out := "Nursing"
		if len(sides) > 0 {
			out += " " + strings.Join(sides, " · ")
		}
		if n.EndedAt == nil {
			out += ", " + running
			if n.Side != nil {
				out += " on " + sideInitial[*n.Side]
			}
		}
		if n.WeightBefore != nil {
			out += ", weight before " + FormatWeight(*n.WeightBefore, req.Units)
		}
		if n.WeightAfter != nil {
			out += ", weight after " + FormatWeight(*n.WeightAfter, req.Units)
		}
		if m, ok := units.MilkTransfer(n.WeightBefore, n.WeightAfter); ok {
			out += ", " + FormatMilk(m, req.Units)
		}
		return out
	},
	day: func(d store.DaySummary, _ model.Units) string {
		return countAnd(d.Nursing.Count, FormatDuration(int(d.Nursing.Seconds)))
	},
}

var pumping = tracker{typ: "pumping", title: "Pumping",
	line: func(it store.TimelineItem, req Request) string {
		p := it.Entry.(model.PumpingSession)
		where := p.Side
		if len(p.Sides) > 0 {
			parts := make([]string, len(p.Sides))
			for i, s := range p.Sides {
				parts[i] = sideInitial[s.Side] + " " + FormatVolume(s.Volume, req.Units)
			}
			where = strings.Join(parts, " · ")
		}
		return "Pumping " + FormatDuration(p.DurationSeconds) + ", " + FormatVolume(p.Volume, req.Units) + " (" + where + ")"
	},
	day: func(d store.DaySummary, u model.Units) string {
		return countAnd(d.Pumping.Count, FormatVolume(units.Volume{ML: d.Pumping.ML, FlOzHundredths: d.Pumping.FlOzHundredths}, u))
	},
}

var diapers = tracker{typ: "diaper", title: "Diapers",
	line: func(it store.TimelineItem, _ Request) string { return "Diaper " + it.Entry.(model.Diaper).Kind },
	day: func(d store.DaySummary, _ model.Units) string {
		if d.Diapers.Total == 0 {
			return ""
		}
		var kinds []string
		for _, k := range []struct {
			name string
			n    int64
		}{{"wet", d.Diapers.Wet}, {"dirty", d.Diapers.Dirty}, {"both", d.Diapers.Both}, {"dry", d.Diapers.Dry}} {
			if k.n > 0 {
				kinds = append(kinds, strconv.FormatInt(k.n, 10)+" "+k.name)
			}
		}
		return strconv.FormatInt(d.Diapers.Total, 10) + " (" + strings.Join(kinds, ", ") + ")"
	},
}

var sleep = tracker{typ: "sleep", title: "Sleep",
	line: func(it store.TimelineItem, req Request) string {
		s := it.Entry.(model.SleepSession)
		if s.EndedAt == nil {
			return "Sleep, " + running
		}
		return "Sleep until " + FormatInstant(*s.EndedAt, req.Loc) + " (" + FormatDuration(int(s.EndedAt.Sub(s.StartedAt).Seconds())) + ")"
	},
	day: func(d store.DaySummary, _ model.Units) string {
		if d.Sleep.Seconds == 0 {
			return ""
		}
		return FormatDuration(int(d.Sleep.Seconds))
	},
}

var feeding = tracker{typ: "meal", title: "Feeding",
	line: func(it store.TimelineItem, _ Request) string {
		m := it.Entry.(model.Meal)
		out := "Solids: " + Text(m.What)
		if m.Reaction != nil && *m.Reaction != "" {
			out += " · " + Text(*m.Reaction)
		}
		return out
	},
	day: func(d store.DaySummary, _ model.Units) string {
		if d.Meals.Count == 0 {
			return ""
		}
		return strconv.FormatInt(d.Meals.Count, 10)
	},
}
