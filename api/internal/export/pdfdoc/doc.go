// Package pdfdoc writes the export PDF; it is the only package that calls fpdf.
package pdfdoc

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"io"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"codeberg.org/go-pdf/fpdf"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/export/fonts"
)

// ErrTooLarge: the rendered document exceeds constants.ExportMaxBytes; nothing was written.
var ErrTooLarge = errors.New("pdfdoc: document too large")

var (
	maxBytes  = constants.ExportMaxBytes // lowered by tests only
	loadFonts = fonts.Load               // replaced by tests only
)

// ContentWidthMM is the page width inside the margins.
const ContentWidthMM = constants.ExportPageWidthMM - 2*constants.ExportMarginMM

const (
	family = "NotoSans"
	// nbAlias is a private-use rune fonts.Encode never lets through, so no user text matches it.
	nbAlias = "\uf8ff\uf8ff"

	mmPerPt = 25.4 / 72
	// Factors of the font size (line height, padding, point side) or of the content width (label column, plot height).
	lineFactor    = 1.4
	padFactor     = 0.25
	squareFactor  = 0.5
	labelShare    = 0.3
	chartAspect   = 0.55
	hairlineMM    = 0.1
	ruleMM        = 0.2
	greyLevel     = 150
	footerBaseMM  = constants.ExportPageHeightUsableMM - constants.ExportMarginMM
	contentBottom = footerBaseMM - 2*lineFactor*constants.ExportFontSizeSmall*mmPerPt
	// Curve labels' baselines stay at least one small-print line apart.
	labelGapMM = lineFactor * constants.ExportFontSizeSmall * mmPerPt
)

// Doc is one export document; methods after a failure do nothing, and WriteTo returns the failure.
type Doc struct {
	f           *fpdf.Fpdf
	err         error
	y           float64
	atTop       bool
	substituted bool
	written     bool
}

// Col is one table column.
type Col struct {
	Title   string
	WidthMM float64
}

// Chart is a plot of reference curves and a kid's points; only what lies inside the ranges is drawn.
type Chart struct {
	Title, XLabel, YLabel  string
	XMin, XMax, YMin, YMax float64
	Curves                 []Curve
	Points                 []Point
}

// Curve is one reference line; an empty Label draws no label.
type Curve struct {
	Label  string
	Points []Point
}

// Point is one chart coordinate.
type Point struct{ X, Y float64 }

// New starts an A4 document; footer gives each page's left footer text.
func New(created time.Time, footer func(page int) string) *Doc {
	d := &Doc{}
	if d.err = loadFonts(); d.err != nil {
		return d
	}
	f := fpdf.NewCustom(&fpdf.InitType{
		OrientationStr: "P",
		UnitStr:        "mm",
		Size:           fpdf.SizeType{Wd: constants.ExportPageWidthMM, Ht: constants.ExportPageHeightMM},
	})
	d.f = f
	f.SetCompression(false)
	f.SetCatalogSort(true)
	f.SetCreationDate(created.UTC())
	f.SetModificationDate(created.UTC())
	f.SetProducer(constants.ExportProducer, false)
	f.SetTitle(constants.ExportTitle, false)
	f.SetMargins(constants.ExportMarginMM, constants.ExportMarginMM, constants.ExportMarginMM)
	f.SetAutoPageBreak(false, 0)
	f.SetCellMargin(0)
	f.AliasNbPages(nbAlias)
	f.AddUTF8FontFromBytes(family, "", fonts.Regular())
	f.AddUTF8FontFromBytes(family, "B", fonts.Bold())
	f.SetFooterFunc(func() {
		f.SetFont(family, "", constants.ExportFontSizeSmall)
		page := f.PageNo()
		left := d.enc(footer(page)) + d.enc(" · Page "+strconv.Itoa(page)+" of ")
		f.Text(constants.ExportMarginMM, footerBaseMM, left+nbAlias)
	})
	d.newPage()
	return d
}

func (d *Doc) ok() bool { return d.err == nil && !d.f.Err() }

func (d *Doc) enc(s string) string {
	out, sub := fonts.Encode(s)
	d.substituted = d.substituted || sub
	return out
}

func (d *Doc) newPage() {
	d.f.AddPage()
	d.y = constants.ExportMarginMM
	d.atTop = true
}

// setFont skips an unchanged font: fpdf writes a font operator on every SetFont call.
func (d *Doc) setFont(style string, size float64) {
	if pt, _ := d.f.GetFontSize(); strings.EqualFold(d.f.GetFontFamily(), family) && d.f.GetFontStyle() == style && pt == size {
		return
	}
	d.f.SetFont(family, style, size)
}

func fontMM(size float64) float64 { return size * mmPerPt }

type cell struct {
	x, w  float64
	text  string
	style string
}

// row draws cells side by side, wrapped; a row that does not fit moves to a new page, and splits only from a page top.
func (d *Doc) row(cells []cell, size float64, repeat func()) {
	lh := fontMM(size) * lineFactor
	pad := fontMM(size) * padFactor
	lines := make([][]string, len(cells))
	n := 1
	for i, c := range cells {
		d.setFont(c.style, size)
		lines[i] = d.f.SplitText(d.enc(c.text), c.w-2*pad)
		n = max(n, len(lines[i]))
	}
	fresh := d.atTop
	for start := 0; start < n; {
		fit := int((contentBottom - d.y - 2*pad) / lh)
		if n-start > fit && !fresh {
			d.newPage()
			if repeat != nil {
				repeat()
			}
			fresh = true
			continue
		}
		k := min(n-start, max(fit, 1))
		for i, c := range cells {
			d.setFont(c.style, size)
			for j := start; j < start+k && j < len(lines[i]); j++ {
				d.f.Text(c.x+pad, d.y+pad+float64(j-start)*lh+fontMM(size), lines[i][j])
			}
		}
		d.y += float64(k)*lh + 2*pad
		d.atTop, fresh = false, false
		start += k
	}
}

// ensure starts a new page unless h millimetres fit below the current position.
func (d *Doc) ensure(h float64) {
	if d.y+h > contentBottom && !d.atTop {
		d.newPage()
	}
}

// Heading writes a bold section title, kept with at least one body line.
func (d *Doc) Heading(text string) {
	if !d.ok() {
		return
	}
	if !d.atTop {
		d.y += fontMM(constants.ExportFontSizeBody) * lineFactor
	}
	d.ensure(fontMM(constants.ExportFontSizeHeading)*(lineFactor+2*padFactor) + fontMM(constants.ExportFontSizeBody)*(lineFactor+2*padFactor))
	d.row([]cell{{x: constants.ExportMarginMM, w: ContentWidthMM, text: text, style: "B"}}, constants.ExportFontSizeHeading, nil)
}

// Line writes one key/value row of the profile block.
func (d *Doc) Line(label, value string) {
	if !d.ok() {
		return
	}
	lw := ContentWidthMM * labelShare
	d.row([]cell{
		{x: constants.ExportMarginMM, w: lw, text: label, style: "B"},
		{x: constants.ExportMarginMM + lw, w: ContentWidthMM - lw, text: value},
	}, constants.ExportFontSizeBody, nil)
}

// Paragraph writes wrapped small print.
func (d *Doc) Paragraph(text string) {
	if !d.ok() {
		return
	}
	d.row([]cell{{x: constants.ExportMarginMM, w: ContentWidthMM, text: text}}, constants.ExportFontSizeSmall, nil)
}

// Body writes wrapped text at the body size: for wording that must not read as small print.
func (d *Doc) Body(text string) {
	if !d.ok() {
		return
	}
	d.row([]cell{{x: constants.ExportMarginMM, w: ContentWidthMM, text: text}}, constants.ExportFontSizeBody, nil)
}

// Table writes a header row, repeated after every page break, then the rows; columns wider than the page are scaled down.
func (d *Doc) Table(cols []Col, rows [][]string) {
	if !d.ok() || len(cols) == 0 {
		return
	}
	total := 0.0
	for _, c := range cols {
		total += c.WidthMM
	}
	scale := min(1, ContentWidthMM/total)
	header := make([]cell, len(cols))
	x := float64(constants.ExportMarginMM)
	for i, c := range cols {
		header[i] = cell{x: x, w: c.WidthMM * scale, text: c.Title, style: "B"}
		x += header[i].w
	}
	drawHeader := func() {
		d.row(header, constants.ExportFontSizeBody, nil)
		d.f.SetDrawColor(0, 0, 0)
		d.f.SetLineWidth(ruleMM)
		d.f.Line(constants.ExportMarginMM, d.y, x, d.y)
	}
	d.ensure(2 * fontMM(constants.ExportFontSizeBody) * (lineFactor + 2*padFactor))
	drawHeader()
	for _, r := range rows {
		cells := make([]cell, len(cols))
		for i := range cols {
			cells[i] = header[i]
			cells[i].style, cells[i].text = "", ""
			if i < len(r) {
				cells[i].text = r[i]
			}
		}
		d.row(cells, constants.ExportFontSizeBody, drawHeader)
	}
}

func formatTick(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }

// Chart draws axes, grey hairline curves labelled at the right edge and the points as black squares.
func (d *Doc) Chart(c Chart) {
	if !d.ok() {
		return
	}
	if !(c.XMax > c.XMin) || !(c.YMax > c.YMin) || math.IsInf(c.XMax-c.XMin, 0) || math.IsInf(c.YMax-c.YMin, 0) {
		d.err = fmt.Errorf("pdfdoc: empty chart range")
		return
	}
	small := float64(constants.ExportFontSizeSmall)
	slh := fontMM(small) * lineFactor
	pad := fontMM(small) * padFactor
	plotH := ContentWidthMM * chartAspect
	d.ensure(fontMM(constants.ExportFontSizeBody)*(lineFactor+2*padFactor) + 2*slh + plotH + 2*slh)
	d.row([]cell{{x: constants.ExportMarginMM, w: ContentWidthMM, text: c.Title, style: "B"}}, constants.ExportFontSizeBody, nil)

	f := d.f
	f.SetFont(family, "", small)
	ylabel := d.enc(c.YLabel)
	xlabel := d.enc(c.XLabel)
	yticks := [2]string{d.enc(formatTick(c.YMin)), d.enc(formatTick(c.YMax))}
	xticks := [2]string{d.enc(formatTick(c.XMin)), d.enc(formatTick(c.XMax))}
	labels := make([]string, len(c.Curves))
	labelW := 0.0
	for i, cv := range c.Curves {
		labels[i] = d.enc(cv.Label)
		labelW = max(labelW, f.GetStringWidth(labels[i]))
	}
	left := constants.ExportMarginMM + max(f.GetStringWidth(yticks[0]), f.GetStringWidth(yticks[1])) + pad
	right := constants.ExportMarginMM + ContentWidthMM - labelW - pad
	f.Text(constants.ExportMarginMM, d.y+fontMM(small), ylabel)
	top := d.y + slh
	bottom := top + plotH
	px := func(x float64) float64 { return left + (x-c.XMin)/(c.XMax-c.XMin)*(right-left) }
	py := func(y float64) float64 { return bottom - (y-c.YMin)/(c.YMax-c.YMin)*plotH }
	in := func(p Point) bool { return p.X >= c.XMin && p.X <= c.XMax && p.Y >= c.YMin && p.Y <= c.YMax }

	f.SetDrawColor(0, 0, 0)
	f.SetLineWidth(ruleMM)
	f.Line(left, top, left, bottom)
	f.Line(left, bottom, right, bottom)
	f.Text(constants.ExportMarginMM, top+fontMM(small), yticks[1])
	f.Text(constants.ExportMarginMM, bottom, yticks[0])
	f.Text(left, bottom+fontMM(small)+pad, xticks[0])
	f.Text(right-f.GetStringWidth(xticks[1]), bottom+fontMM(small)+pad, xticks[1])
	f.Text(left+(right-left-f.GetStringWidth(xlabel))/2, bottom+slh+fontMM(small)+pad, xlabel)

	labelY := make([]float64, len(c.Curves))
	var labelled []int
	for i, cv := range c.Curves {
		for j := len(cv.Points) - 1; j >= 0; j-- {
			if in(cv.Points[j]) {
				if labels[i] != "" {
					labelY[i] = py(cv.Points[j].Y) + fontMM(small)/2
					labelled = append(labelled, i)
				}
				break
			}
		}
	}
	spreadLabels(labelY, labelled, top+fontMM(small), bottom)

	f.SetDrawColor(greyLevel, greyLevel, greyLevel)
	f.SetLineWidth(hairlineMM)
	for i, cv := range c.Curves {
		for j := 1; j < len(cv.Points); j++ {
			if p, q := cv.Points[j], cv.Points[j-1]; in(p) && in(q) {
				f.Line(px(q.X), py(q.Y), px(p.X), py(p.Y))
			}
		}
		if slices.Contains(labelled, i) {
			f.Text(right+pad, labelY[i], labels[i])
		}
	}
	f.SetFillColor(0, 0, 0)
	s := fontMM(small) * squareFactor
	for _, p := range c.Points {
		if in(p) {
			f.Rect(px(p.X)-s/2, py(p.Y)-s/2, s, s, "F")
		}
	}
	d.y = bottom + 2*slh + pad
	d.atTop = false
}

// spreadLabels moves the baselines y[idx] apart by labelGapMM in their order, keeping each within [lo, hi] while they fit.
func spreadLabels(y []float64, idx []int, lo, hi float64) {
	slices.SortStableFunc(idx, func(a, b int) int { return cmp.Compare(y[a], y[b]) })
	for k, i := range idx {
		y[i] = max(y[i], lo)
		if k > 0 {
			y[i] = max(y[i], y[idx[k-1]]+labelGapMM)
		}
	}
	for k := len(idx) - 1; k >= 0; k-- {
		i := idx[k]
		y[i] = min(y[i], hi)
		if k < len(idx)-1 {
			y[i] = min(y[i], y[idx[k+1]]-labelGapMM)
		}
		y[i] = max(y[i], lo)
	}
}

// Substituted reports whether any rune of this document was replaced by fonts.Encode.
func (d *Doc) Substituted() bool { return d.substituted }

// WriteTo renders the document once; it writes nothing when rendering fails or exceeds the size cap.
func (d *Doc) WriteTo(w io.Writer) (int64, error) {
	if d.err != nil {
		return 0, d.err
	}
	if d.written {
		return 0, errors.New("pdfdoc: document already written")
	}
	d.written = true
	var b bytes.Buffer
	if err := d.f.Output(&b); err != nil {
		return 0, fmt.Errorf("pdfdoc: render: %w", err)
	}
	if b.Len() > maxBytes {
		return 0, ErrTooLarge
	}
	return b.WriteTo(w)
}
