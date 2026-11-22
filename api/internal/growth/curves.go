package growth

import (
	"math"
	"slices"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

// Point is one curve value. Gap is true on the first point read from another table or table region than
// the point before it (WHO day 730/731, WHO 2006/2007, CDC infant/child): the line must not join the two.
type Point struct {
	AgeMonths, Value float64
	Gap              bool
}

// Curve is one percentile line.
type Curve struct {
	Percentile float64
	Points     []Point
}

var (
	whoLines    = []float64{3, 15, 50, 85, 97}
	cdcLines    = []float64{3, 5, 10, 25, 50, 75, 90, 95, 97}
	cdcBMILines = []float64{3, 5, 10, 25, 50, 75, 85, 90, 95, 97}
)

// Percentiles are the chart lines of a source and measure, ascending; nil when not offered.
func Percentiles(s Source, m Measure) []float64 {
	if _, ok := tableNames[s][m]; !ok {
		return nil
	}
	switch {
	case s == SourceWHO:
		return slices.Clone(whoLines)
	case m == BMI:
		return slices.Clone(cdcBMILines)
	}
	return slices.Clone(cdcLines)
}

// Curves draws every line of Percentiles(s, m) from fromMonths to toMonths.
// The age range is clamped to the source's coverage; ErrNotCovered when nothing remains.
func Curves(s Source, m Measure, sex Sex, fromMonths, toMonths float64) ([]Curve, error) {
	lo, hi, ok := coverage(s, m, sex)
	lines := Percentiles(s, m)
	from, to := max(fromMonths, lo), min(toMonths, hi)
	if !ok || lines == nil || !(from <= to) {
		return nil, ErrNotCovered
	}
	ages := curveAges(from, to)
	if seam := whoLengthLastDay / constants.DaysPerMonth; s == SourceWHO && (m == Length || m == BMI) && from < seam && seam < to {
		i, _ := slices.BinarySearch(ages, seam)
		ages = slices.Insert(ages, i, seam)
	}
	zs := make([]float64, len(lines))
	curves := make([]Curve, len(lines))
	for i, p := range lines {
		zs[i] = normalQuantile(p / 100)
		curves[i] = Curve{Percentile: p, Points: make([]Point, 0, len(ages))}
	}
	prevDay, prevRegion := -1, ""
	for n, a := range ages {
		h, day, err := curveHit(s, m, sex, a)
		if err != nil {
			return nil, err
		}
		if day >= 0 && day == prevDay {
			continue
		}
		gap := n > 0 && h.region != prevRegion
		for i := range curves {
			curves[i].Points = append(curves[i].Points, Point{a, valueAtZ(h, zs[i], restricted(s, m)), gap})
		}
		prevDay, prevRegion = day, h.region
	}
	return curves, nil
}

// curveHit reads WHO 2006 at day round(months × DaysPerMonth), everything else at months; day is -1 then.
func curveHit(s Source, m Measure, sex Sex, months float64) (hit, int, error) {
	if s == SourceWHO {
		if day := int(math.Round(months * constants.DaysPerMonth)); day <= whoLastDay(m, sex) {
			h, err := lookupDays(s, m, sex, day)
			return h, day, err
		}
		if _, month := pairFor(s, m, sex); month != nil {
			months = max(months, month.rows[0].age)
		}
	}
	h, err := lookupMonths(s, m, sex, months)
	return h, -1, err
}

// curveAges: from, the grid strictly between from and to (fine steps until CurveFineUntilMonths), then to.
func curveAges(from, to float64) []float64 {
	ages := []float64{from}
	step := func(a float64) float64 {
		if a < constants.CurveFineUntilMonths {
			return constants.CurveFineStepMonths
		}
		return constants.CurveCoarseStepMonths
	}
	for a := 0.0; a < to; a += step(a) {
		if a > from {
			ages = append(ages, a)
		}
	}
	if to > from {
		ages = append(ages, to)
	}
	return ages
}
