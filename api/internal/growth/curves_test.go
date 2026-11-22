package growth

import (
	"errors"
	"math"
	"slices"
	"testing"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

// curveValueAt reads percentile p at age `months` from Curves; the age must be a point of the curve.
func curveValueAt(t *testing.T, s Source, m Measure, sex Sex, p, months float64) float64 {
	t.Helper()
	curves, err := Curves(s, m, sex, months, months)
	if err != nil {
		t.Fatalf("Curves(%s, %s, %s, %v): %v", s, m, sex, months, err)
	}
	for _, c := range curves {
		if c.Percentile != p {
			continue
		}
		for _, pt := range c.Points {
			if pt.AgeMonths == months {
				return pt.Value
			}
		}
	}
	t.Fatalf("no P%v point at %v months", p, months)
	return 0
}

func TestCDCPercentileColumns(t *testing.T) {
	cases := []struct {
		m        Measure
		sex      Sex
		months   float64
		p, want  float64
		fromWide bool
	}{
		{Weight, Male, 9.5, 5, 7.90076, false},
		{Weight, Male, 9.5, 97, 11.74538, false},
		{Length, Male, 240, 3, 163.33333, false},
		{BMI, Female, 240.5, 85, 26.49502, false},
		{BMI, Female, 240.5, 97, 35.10556, false},
		{Weight, Male, 9.5, 5, 7.90076, true},
		{BMI, Female, 240.5, 97, 35.10556, true},
	}
	for _, c := range cases {
		var got float64
		if c.fromWide {
			got = pointOf(t, SourceCDC, c.m, c.sex, 0, 300, c.p, c.months)
		} else {
			got = curveValueAt(t, SourceCDC, c.m, c.sex, c.p, c.months)
		}
		if math.Abs(got-c.want) > 1e-4 {
			t.Errorf("CDC %s %s %v mo P%v = %.6f, want %v", c.m, c.sex, c.months, c.p, got, c.want)
		}
	}
}

func pointOf(t *testing.T, s Source, m Measure, sex Sex, from, to, p, months float64) float64 {
	t.Helper()
	curves, err := Curves(s, m, sex, from, to)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range curves {
		if c.Percentile == p {
			for _, pt := range c.Points {
				if pt.AgeMonths == months {
					return pt.Value
				}
			}
		}
	}
	t.Fatalf("no P%v point at %v months in %v..%v", p, months, from, to)
	return 0
}

func TestWHOCurveAtDay731(t *testing.T) {
	// The 24-month grid point reads WHO day round(24 × 30.4375) = 731.
	for _, c := range []struct{ p, want float64 }{{3, 13.473}, {97, 18.533}} {
		if got := pointOf(t, SourceWHO, BMI, Female, 0, 60, c.p, 24); math.Abs(got-c.want) > 0.001 {
			t.Errorf("bfa girls 24 mo P%v = %.4f, want %v", c.p, got, c.want)
		}
	}
}

func TestCurvesGrid(t *testing.T) {
	curves, err := Curves(SourceCDC, Weight, Male, 0, 30)
	if err != nil {
		t.Fatal(err)
	}
	var ages []float64
	for _, p := range curves[0].Points {
		ages = append(ages, p.AgeMonths)
	}
	var want []float64
	for a := 0.0; a <= 24; a += 0.5 {
		want = append(want, a)
	}
	for a := 25.0; a <= 30; a++ {
		want = append(want, a)
	}
	if !slices.Equal(ages, want) {
		t.Errorf("ages %v\nwant %v", ages, want)
	}
	for _, c := range curves {
		if len(c.Points) != len(want) {
			t.Errorf("P%v: %d points, want %d", c.Percentile, len(c.Points), len(want))
		}
	}

	// Endpoints off the grid are kept; the range is clamped to coverage.
	cc, err := Curves(SourceCDC, BMI, Female, 10, 1000)
	if err != nil {
		t.Fatal(err)
	}
	pts := cc[0].Points
	if pts[0].AgeMonths != 24 || pts[len(pts)-1].AgeMonths != 240.5 || pts[len(pts)-2].AgeMonths != 240 {
		t.Errorf("CDC BMI clamped ends %v..%v", pts[0].AgeMonths, pts[len(pts)-1].AgeMonths)
	}
	wh, err := Curves(SourceWHO, Head, Male, 0, 200)
	if err != nil {
		t.Fatal(err)
	}
	last := wh[0].Points[len(wh[0].Points)-1].AgeMonths
	if want := 1856 / constants.DaysPerMonth; math.Abs(last-want) > 1e-12 {
		t.Errorf("WHO head ends at %v mo, want %v", last, want)
	}
}

func TestCurvesSeamGap(t *testing.T) {
	seam := 730 / constants.DaysPerMonth
	for _, m := range []Measure{Length, BMI} {
		curves, err := Curves(SourceWHO, m, Female, 20, 28)
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range curves {
			gaps := 0
			for i, p := range c.Points {
				if p.Gap {
					gaps++
					prev := c.Points[i-1]
					if prev.AgeMonths != seam || p.AgeMonths != 24 {
						t.Errorf("%s P%v: gap between %v and %v, want %v and 24", m, c.Percentile, prev.AgeMonths, p.AgeMonths, seam)
					}
				}
			}
			if gaps != 1 {
				t.Errorf("%s P%v: %d gaps, want 1", m, c.Percentile, gaps)
			}
		}
		// The last length point is day 730 of the length region, not a blend with day 731.
		want, _ := valueAtPercentile(SourceWHO, m, Female, 730, 50)
		if got := pointOf(t, SourceWHO, m, Female, 20, 28, 50, seam); got != want {
			t.Errorf("%s P50 at the seam = %v, want day 730 value %v", m, got, want)
		}
	}
	w, err := Curves(SourceWHO, Weight, Female, 20, 28)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range w[0].Points {
		if p.Gap || p.AgeMonths == seam {
			t.Errorf("WHO weight has a seam point at %v mo", p.AgeMonths)
		}
	}
	// A change of table (WHO 2006 to 2007, CDC infant to child) also breaks the line.
	for _, c := range []struct {
		src    Source
		m      Measure
		at     float64
		from   float64
		to     float64
		before float64
	}{
		{SourceWHO, Weight, 61, 55, 65, 60},
		{SourceCDC, Length, 24, 20, 28, 23.5},
		{SourceCDC, Weight, 24, 20, 28, 23.5},
	} {
		curves, err := Curves(c.src, c.m, Male, c.from, c.to)
		if err != nil {
			t.Fatal(err)
		}
		pts := curves[0].Points
		for i, p := range pts {
			if p.Gap != (p.AgeMonths == c.at) {
				t.Errorf("%s %s at %v mo: gap %v", c.src, c.m, p.AgeMonths, p.Gap)
			}
			if p.Gap && pts[i-1].AgeMonths != c.before {
				t.Errorf("%s %s: gap after %v, want after %v", c.src, c.m, pts[i-1].AgeMonths, c.before)
			}
		}
	}
}

func TestCurvesNotCovered(t *testing.T) {
	for _, c := range []struct {
		src      Source
		m        Measure
		from, to float64
	}{
		{SourceWHO, Head, 61, 100},
		{SourceWHO, Weight, 121, 200},
		{SourceCDC, BMI, 0, 23.5},
		{SourceCDC, Head, 37, 60},
		{SourceWHO, Weight, 10, 5},
		{Source("x"), Weight, 0, 10},
		{SourceWHO, Weight, math.NaN(), 10},
	} {
		if _, err := Curves(c.src, c.m, Male, c.from, c.to); !errors.Is(err, ErrNotCovered) {
			t.Errorf("Curves(%s, %s, %v..%v): err %v, want ErrNotCovered", c.src, c.m, c.from, c.to, err)
		}
	}
}

func TestPercentiles(t *testing.T) {
	who := []float64{3, 15, 50, 85, 97}
	cdc := []float64{3, 5, 10, 25, 50, 75, 90, 95, 97}
	cdcBMI := []float64{3, 5, 10, 25, 50, 75, 85, 90, 95, 97}
	for _, m := range []Measure{Weight, Length, Head, BMI} {
		if got := Percentiles(SourceWHO, m); !slices.Equal(got, who) {
			t.Errorf("WHO %s: %v", m, got)
		}
		want := cdc
		if m == BMI {
			want = cdcBMI
		}
		if got := Percentiles(SourceCDC, m); !slices.Equal(got, want) {
			t.Errorf("CDC %s: %v", m, got)
		}
	}
	Percentiles(SourceWHO, Weight)[0] = 99
	if Percentiles(SourceWHO, Weight)[0] != 3 {
		t.Error("Percentiles returns shared storage")
	}
	if Percentiles(Source("x"), Weight) != nil {
		t.Error("unknown source has percentiles")
	}
}
