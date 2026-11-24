// Package export lays out one kid's export PDF from a store snapshot; it never calls httpapi.
package export

import (
	"errors"
	"io"
	"slices"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/export/fonts"
	"github.com/davcs86/happybabytracker/api/internal/export/pdfdoc"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/store"
)

// Request is what an export prints besides the snapshot: the kid-local range (YYYY-MM-DD), the modules, the
// requester's units, the clock and the kid's zone.
type Request struct {
	From, To string
	Modules  []model.Module
	Units    model.Units
	Now      time.Time
	Loc      *time.Location
}

var errNoZone = errors.New("export: no time zone")

var (
	maxRowsPerSection = constants.ExportMaxRowsPerSection // lowered by tests only
	maxRows           = constants.ExportMaxRows           // lowered by tests only
)

const (
	notice      = "Some characters cannot be shown in this PDF and appear as ?"
	footerLeft  = "Generated "
	footerRight = " by happybabytracker"
)

// moduleTitle is each module's section heading and its name in the profile.
var moduleTitle = map[model.Module]string{
	"bottles": "Bottles", "nursing": "Nursing", "pumping": "Pumping", "diapers": "Diapers", "sleep": "Sleep",
	"feeding": "Feeding", "milestones": "Milestones", "growth": "Growth", "vaccination": "Vaccination",
}

// item is one drawn element: a chart or a table when set, else a heading, a profile line or a paragraph.
type item struct {
	heading      string
	label, value string
	paragraph    string
	cols         []pdfdoc.Col
	rows         [][]string
	chart        *pdfdoc.Chart
}

func (it item) texts() []string {
	out := []string{it.heading, it.label, it.value, it.paragraph}
	for _, c := range it.cols {
		out = append(out, c.Title)
	}
	for _, r := range it.rows {
		out = append(out, r...)
	}
	if c := it.chart; c != nil {
		out = append(out, c.Title, c.XLabel, c.YLabel)
		for _, cv := range c.Curves {
			out = append(out, cv.Label)
		}
	}
	return out
}

func (it item) draw(d *pdfdoc.Doc) {
	switch {
	case it.chart != nil:
		d.Chart(*it.chart)
	case it.cols != nil:
		d.Table(it.cols, it.rows)
	case it.heading != "":
		d.Heading(it.heading)
	case it.label != "":
		d.Line(it.label, it.value)
	default:
		d.Paragraph(it.paragraph)
	}
}

// budget is what is left of maxRows for the sections still to come.
type budget struct{ left int }

// take is how many of n rows a section prints: at most maxRowsPerSection and what is left of maxRows.
func (b *budget) take(n int) int {
	k := max(0, min(n, maxRowsPerSection, b.left))
	b.left -= k
	return k
}

// sectionWriters builds each module's items; a requested module without a writer prints no section yet (Task 13).
var sectionWriters = map[model.Module]func(snap store.ExportSnapshot, req Request, b *budget) ([]item, error){
	"bottles": bottles.items, "nursing": nursing.items, "pumping": pumping.items,
	"diapers": diapers.items, "sleep": sleep.items, "feeding": feeding.items, "growth": growthItems,
}

// Render writes the PDF of req.Modules only, in model.Modules order, after the profile; nothing is written on failure.
func Render(w io.Writer, snap store.ExportSnapshot, req Request) error {
	if req.Loc == nil {
		return errNoZone
	}
	if err := fonts.Load(); err != nil {
		return err
	}
	b := &budget{left: maxRows}
	var body []item
	for _, m := range model.Modules {
		if write := sectionWriters[m]; write != nil && slices.Contains(req.Modules, m) {
			items, err := write(snap, req, b)
			if err != nil {
				return err
			}
			body = append(body, items...)
		}
	}
	profile, err := profileItems(snap.Kid, req)
	if err != nil {
		return err
	}
	if substituted(profile) || substituted(body) {
		profile = append(profile, item{paragraph: notice})
	}
	footer := footerLeft + FormatInstant(req.Now, req.Loc) + " (" + req.Loc.String() + ")" + footerRight
	d := pdfdoc.New(req.Now, func(int) string { return footer })
	for _, it := range append(profile, body...) {
		it.draw(d)
	}
	_, err = d.WriteTo(w)
	return err
}

func substituted(items []item) bool {
	for _, it := range items {
		for _, s := range it.texts() {
			if _, sub := fonts.Encode(s); sub {
				return true
			}
		}
	}
	return false
}
