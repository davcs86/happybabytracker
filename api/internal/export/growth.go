package export

import (
	"errors"
	"math"
	"strconv"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/export/pdfdoc"
	"github.com/davcs86/happybabytracker/api/internal/growth"
	"github.com/davcs86/happybabytracker/api/internal/kidtime"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/store"
	"github.com/davcs86/happybabytracker/api/internal/units"
)

// Shares of the content width per growth column.
const (
	growthDateShare    = 0.14
	growthAgeShare     = 0.18
	growthMeasureShare = 0.17
)

// The screen's words: GrowthTable.svelte, growth/+page.svelte and growth-chart.ts.
const (
	noMeasurements = "No measurements recorded."
	noTableSuffix  = " has no table for this measure at this age"
	licencePrefix  = "Licence: "
	axisAge        = "Age (months)"
	axisCorrected  = "Corrected age (months)"
	axisMixed      = "Age (months; corrected before the second birthday)"
	bmiUnit        = "kg/m²"
)

var (
	growthMeasures = []growth.Measure{growth.Weight, growth.Length, growth.Head, growth.BMI}
	measureLabel   = map[growth.Measure]string{growth.Weight: "Weight", growth.Length: "Length", growth.Head: "Head circumference", growth.BMI: "BMI"}
	unscoredReason = map[string]string{"before_term": beforeTerm, "not_covered": "this source has no table for this age"}
	growthCols     = []pdfdoc.Col{
		{Title: "Date", WidthMM: contentWidth * growthDateShare},
		{Title: "Age", WidthMM: contentWidth * growthAgeShare},
		{Title: measureLabel[growth.Weight], WidthMM: contentWidth * growthMeasureShare},
		{Title: measureLabel[growth.Length], WidthMM: contentWidth * growthMeasureShare},
		{Title: measureLabel[growth.Head], WidthMM: contentWidth * growthMeasureShare},
		{Title: measureLabel[growth.BMI], WidthMM: contentWidth * growthMeasureShare},
	}
)

// scoredRow is a printed measurement with its age basis and its scores under the kid's current source.
type scoredRow struct {
	g      model.Growth
	basis  kidtime.AgeBasis
	scores growth.RowScores
}

func (r scoredRow) score(m growth.Measure) *growth.MeasureScore {
	switch m {
	case growth.Weight:
		return r.scores.Weight
	case growth.Length:
		return r.scores.Length
	case growth.Head:
		return r.scores.Head
	}
	return r.scores.BMI
}

// growthItems lists every measurement up to req.To, newest first, scored under the kid's current source.
func growthItems(snap store.ExportSnapshot, req Request, b *budget) ([]item, error) {
	kid := snap.Kid
	src, sex := growth.Source(kid.GrowthSource), growth.Sex(kid.Sex)
	info, ok := growth.SourceInfo(src)
	if !ok {
		return nil, errGrowthSource
	}
	out := []item{{heading: moduleTitle["growth"]}}
	var listed []model.Growth
	for _, g := range snap.Growth {
		if g.MeasuredOn <= req.To {
			listed = append(listed, g)
		}
	}
	if len(listed) == 0 {
		return append(out, item{paragraph: noMeasurements}), nil
	}
	shown := b.take(len(listed))
	if shown == 0 {
		return append(out, item{paragraph: strconv.Itoa(len(listed)) + overflow}), nil
	}
	rows := make([]scoredRow, shown)
	cells := make([][]string, shown)
	var tables []string
	for i, g := range listed[:shown] {
		basis, err := kidtime.AgeBasisOn(kid.BirthDate, kid.GestationWeeks, kid.GestationDays, g.MeasuredOn, req.Loc)
		if err != nil {
			return nil, err
		}
		var weightG, lengthMM, headMM *int64
		if g.Weight != nil {
			weightG = &g.Weight.Grams
		}
		if g.Length != nil {
			lengthMM = &g.Length.MM
		}
		if g.Head != nil {
			headMM = &g.Head.MM
		}
		s, err := growth.ScoreRow(src, sex, basis, weightG, lengthMM, headMM)
		if err != nil {
			return nil, err
		}
		rows[i] = scoredRow{g, basis, s}
		cells[i] = growthCells(rows[i], req.Units)
		for _, m := range growthMeasures {
			if ms := rows[i].score(m); ms != nil && ms.Scored {
				tables = append(tables, ms.Table)
			}
		}
	}
	out = append(out, item{cols: growthCols, rows: cells})
	if len(listed) > shown {
		out = append(out, item{paragraph: strconv.Itoa(len(listed)-shown) + overflow})
	}
	out = append(out, item{paragraph: info.Attribution}, item{paragraph: licencePrefix + info.Licence})
	for _, p := range growth.AdaptationDisclaimers(src, tables) {
		out = append(out, item{paragraph: p})
	}
	today := req.Now.In(req.Loc).Format(time.DateOnly)
	now, err := kidtime.AgeBasisOn(kid.BirthDate, kid.GestationWeeks, kid.GestationDays, today, req.Loc)
	if err != nil {
		return nil, err
	}
	for _, m := range growthMeasures {
		its, err := measureChart(m, rows, src, sex, info.Name, now.Corrected, req.Units)
		if err != nil {
			return nil, err
		}
		out = append(out, its...)
	}
	return out, nil
}

// growthCells words a row as GrowthTable.svelte does: "<value>: z …, percentile …" or "<value>: <reason>".
func growthCells(r scoredRow, u model.Units) []string {
	var corrected *int
	if r.basis.Corrected {
		corrected = &r.basis.ScoringDays
	}
	cells := []string{FormatDayHeading(r.g.MeasuredOn), FormatAgeDays(r.basis.ChronologicalDays, corrected), "", "", "", ""}
	if w := r.g.Weight; w != nil {
		cells[2] = FormatWeight(*w, u) + scoreSuffix(r.scores.Weight)
	}
	if l := r.g.Length; l != nil {
		cells[3] = FormatLength(*l, u) + scoreSuffix(r.scores.Length)
	}
	if h := r.g.Head; h != nil {
		cells[4] = FormatLength(*h, u) + scoreSuffix(r.scores.Head)
	}
	if v := r.scores.BMIValue; v != nil {
		cells[5] = strconv.FormatFloat(*v, 'f', -1, 64) + scoreSuffix(r.scores.BMI)
	}
	return cells
}

func scoreSuffix(s *growth.MeasureScore) string {
	if !s.Scored {
		return ": " + unscoredReason[s.Reason]
	}
	return ": z " + FormatZ(s.Z) + ", percentile " + FormatPercentile(s.Percentile)
}

func imperialAxis(m growth.Measure, u model.Units) bool {
	if m == growth.Weight {
		return u.Weight == string(units.UnitLB)
	}
	return (m == growth.Length || m == growth.Head) && u.Length == string(units.UnitIN)
}

func unitLabel(m growth.Measure, u model.Units) string {
	switch m {
	case growth.BMI:
		return bmiUnit
	case growth.Weight:
		return u.Weight
	}
	return u.Length
}

// pointValue is the row's stored column of the requester's unit, as growth-chart.ts rowValue reads it.
func pointValue(r scoredRow, m growth.Measure, u model.Units) float64 {
	imperial := imperialAxis(m, u)
	switch m {
	case growth.Weight:
		if imperial {
			return float64(r.g.Weight.OzHundredths) / (hundredths * ozPerLB)
		}
		return float64(r.g.Weight.Grams) / gramsPerKG
	case growth.Length, growth.Head:
		l := r.g.Length
		if m == growth.Head {
			l = r.g.Head
		}
		if imperial {
			return float64(l.InHundredths) / hundredths
		}
		return float64(l.MM) / mmPerCM
	}
	return *r.scores.BMIValue
}

// measureChart is the screen's no-table sentence when a row is not covered, then a chart when a point is scored.
func measureChart(m growth.Measure, rows []scoredRow, src growth.Source, sex growth.Sex, sourceName string, correctedAxis bool, u model.Units) ([]item, error) {
	var out []item
	var points []pdfdoc.Point
	mixed := false
	for _, r := range rows {
		s := r.score(m)
		if s == nil {
			continue
		}
		if !s.Scored && s.Reason == "not_covered" && out == nil {
			out = append(out, item{label: measureLabel[m], value: sourceName + noTableSuffix})
		}
		if s.Scored {
			points = append(points, pdfdoc.Point{X: float64(r.basis.ScoringDays) / constants.DaysPerMonth, Y: pointValue(r, m, u)})
			mixed = mixed || r.basis.Corrected
		}
	}
	if len(points) == 0 {
		return out, nil
	}
	xs := make([]float64, len(points))
	ys := make([]float64, 0, len(points))
	for i, p := range points {
		xs[i] = p.X
		ys = append(ys, p.Y)
	}
	xMin, xMax := niceDomain(xs, constants.ExportChartXTicks)
	xMin = max(0, xMin)
	lines, err := growth.Curves(src, m, sex, xMin, xMax)
	if err != nil && !errors.Is(err, growth.ErrNotCovered) {
		return nil, err
	}
	imperial := imperialAxis(m, u)
	var curves []pdfdoc.Curve
	for _, line := range lines {
		var segments [][]pdfdoc.Point
		for _, p := range line.Points {
			v := p.Value
			if imperial {
				v = *growth.Imperial(m, p.Value)
			}
			if p.Gap || segments == nil {
				segments = append(segments, nil)
			}
			segments[len(segments)-1] = append(segments[len(segments)-1], pdfdoc.Point{X: p.AgeMonths, Y: v})
			ys = append(ys, v)
		}
		var kept []pdfdoc.Curve
		for _, seg := range segments {
			if len(seg) >= 2 {
				kept = append(kept, pdfdoc.Curve{Points: seg})
			}
		}
		if len(kept) > 0 {
			kept[len(kept)-1].Label = ordinal(int(line.Percentile))
		}
		curves = append(curves, kept...)
	}
	yMin, yMax := niceDomain(ys, constants.ExportChartYTicks)
	axis := axisAge
	switch {
	case correctedAxis:
		axis = axisCorrected
	case mixed:
		axis = axisMixed
	}
	return append(out, item{chart: &pdfdoc.Chart{
		Title: measureLabel[m], XLabel: axis, YLabel: measureLabel[m] + " (" + unitLabel(m, u) + ")",
		XMin: xMin, XMax: xMax, YMin: yMin, YMax: yMax, Curves: curves, Points: points,
	}}), nil
}

// ordinal mirrors growth-chart.ts: 1st, 2nd, 3rd, 11th–13th, 15th.
func ordinal(n int) string {
	if tens := n % 100; tens >= 11 && tens <= 13 {
		return strconv.Itoa(n) + "th"
	}
	switch n % 10 {
	case 1:
		return strconv.Itoa(n) + "st"
	case 2:
		return strconv.Itoa(n) + "nd"
	case 3:
		return strconv.Itoa(n) + "rd"
	}
	return strconv.Itoa(n) + "th"
}

// niceStep, stepDecimals and niceDomain mirror growth-chart.ts: equal values open by one step each side.
func niceStep(span float64, target int) float64 {
	raw := span / float64(target)
	mag := math.Pow(10, math.Floor(math.Log10(raw)))
	switch norm := raw / mag; {
	case norm <= 1:
		return mag
	case norm <= 2:
		return 2 * mag
	case norm <= 5:
		return 5 * mag
	}
	return 10 * mag
}

func stepDecimals(step float64) int { return max(0, int(-math.Floor(math.Log10(step)))) }

func toFixed(v float64, decimals int) float64 {
	r, _ := strconv.ParseFloat(strconv.FormatFloat(v, 'f', decimals, 64), 64)
	return r
}

func niceDomain(values []float64, target int) (float64, float64) {
	lo, hi := values[0], values[0]
	for _, v := range values {
		lo, hi = min(lo, v), max(hi, v)
	}
	if lo == hi {
		abs := math.Abs(lo)
		if abs == 0 {
			abs = 1
		}
		pad := niceStep(abs, target)
		lo, hi = lo-pad, hi+pad
	}
	step := niceStep(hi-lo, target)
	d := stepDecimals(step)
	return toFixed(math.Floor(lo/step)*step, d), toFixed(math.Ceil(hi/step)*step, d)
}
