package growth

import (
	"errors"
	"math"
	"testing"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

func TestWHORestrictedZWorkedExamples(t *testing.T) {
	cases := []struct{ l, m, s, y, want float64 }{
		{-0.3067, 15.4013, 0.08115, 20.5, 3.40},
		{-0.4850, 15.8667, 0.07818, 12.0, -3.76},
		{-0.4488, 15.2759, 0.08380, 18.8, 2.37},
		{-1.7862, 16.9392, 0.11070, 30, 3.35},
		{-1.3529, 20.4951, 0.12579, 14, -3.80},
		{-1.6318, 16.0490, 0.10038, 19, 1.47},
	}
	// WHO prints SD3/SD2 rounded to 2 dp, so the exact z* can differ from its printed value by up to ~0.01 near |z*| 3.8.
	for _, c := range cases {
		if got := restrictedZ(c.l, c.m, c.s, c.y); math.Abs(got-c.want) > 0.01 {
			t.Errorf("restrictedZ(%v, %v, %v, %v) = %.4f, want %.2f", c.l, c.m, c.s, c.y, got, c.want)
		}
		if got := printedRestrictedZ(c.l, c.m, c.s, c.y); math.Abs(got-c.want) > 0.005 {
			t.Errorf("printed working (%v, %v, %v, %v) = %.4f, want %.2f", c.l, c.m, c.s, c.y, got, c.want)
		}
	}
}

// printedRestrictedZ reproduces WHO's printed working: z* from SD3 and SD2 rounded to 2 decimals.
func printedRestrictedZ(l, m, s, y float64) float64 {
	r := func(k float64) float64 { return math.Round(lmsValue(l, m, s, k)*100) / 100 }
	switch z := lmsZ(l, m, s, y); {
	case z > 3:
		return 3 + (y-r(3))/(r(3)-r(2))
	case z < -3:
		return -3 + (y-r(-3))/(r(-2)-r(-3))
	default:
		return z
	}
}

func TestWHOSDLines(t *testing.T) {
	cases := []struct {
		m    Measure
		sex  Sex
		day  int
		k    float64
		want float64
	}{
		{Weight, Male, 0, -3, 2.08},
		{Weight, Male, 0, 0, 3.346},
		{Weight, Male, 0, 3, 5.031},
		{Weight, Male, 0, 4, 5.642},
		{Weight, Male, 365, -2, 7.741},
		{Weight, Male, 365, 4, 14.7},
		{Weight, Male, 1856, -4, 10.742},
		{Weight, Male, 1856, 3, 28.171},
		{Length, Female, 730, 0, 86.401},
		{Length, Female, 730, 2, 92.851},
		{Length, Female, 731, 0, 85.73},
		{Length, Female, 731, 3, 95.411},
		{Length, Female, 0, -3, 43.56},
	}
	for _, c := range cases {
		if got := sdLine(SourceWHO, c.m, c.sex, c.day, c.k); math.Abs(got-c.want) > 0.0005 {
			t.Errorf("sdLine(who, %s, %s, %d, %v) = %.4f, want %v", c.m, c.sex, c.day, c.k, got, c.want)
		}
	}
}

func TestWHOPercentileAtExactDay(t *testing.T) {
	for _, c := range []struct{ p, want float64 }{{3, 13.473}, {97, 18.533}} {
		got, err := valueAtPercentile(SourceWHO, BMI, Female, 731, c.p)
		if err != nil {
			t.Fatal(err)
		}
		if math.Abs(got-c.want) > 0.001 {
			t.Errorf("bfa girls day 731 P%v = %.4f, want %v", c.p, got, c.want)
		}
	}
	if _, err := valueAtPercentile(SourceWHO, Head, Female, 1857, 50); !errors.Is(err, ErrNotCovered) {
		t.Errorf("head day 1857: err %v, want ErrNotCovered", err)
	}
}

func TestQuantileSign(t *testing.T) {
	if z := normalQuantile(0.97); math.Abs(z-1.880794) > 1e-6 {
		t.Errorf("quantile(0.97) = %v, want 1.880794", z)
	}
	if z := normalQuantile(0.03); math.Abs(z+1.880794) > 1e-6 {
		t.Errorf("quantile(0.03) = %v, want -1.880794", z)
	}
	if p := phi(1.880794); math.Abs(p-0.97) > 1e-6 {
		t.Errorf("phi(1.880794) = %v, want 0.97", p)
	}
}

func TestCDCWorkedExample(t *testing.T) {
	sc, err := scoreAtMonths(SourceCDC, Weight, Male, 9.5, 9.7)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(sc.Z-0.207) > 0.001 {
		t.Errorf("z = %.4f, want 0.207", sc.Z)
	}
	if math.Abs(sc.Percentile-58.2) > 0.1 {
		t.Errorf("percentile = %.3f, want 58.2", sc.Percentile)
	}
	if got := curveValueAt(t, SourceCDC, Weight, Male, 5, 9.5); math.Abs(got-7.90) > 0.005 {
		t.Errorf("P5 at 9.5 mo = %.4f, want 7.90", got)
	}
}

func TestWHOScoreUsesRestriction(t *testing.T) {
	// Boys day 365, weight above SD3: z* must follow the restricted line, length must not.
	sd3 := sdLine(SourceWHO, Weight, Male, 365, 3)
	sd4 := sdLine(SourceWHO, Weight, Male, 365, 4)
	sc, err := ScoreOf(SourceWHO, Weight, Male, 365, sd4)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(sc.Z-4) > 1e-9 {
		t.Errorf("weight at restricted SD4: z = %v, want 4", sc.Z)
	}
	if want := 100 * phi(4); math.Abs(sc.Percentile-want) > 1e-9 {
		t.Errorf("percentile = %v, want %v", sc.Percentile, want)
	}
	if sd4 <= sd3 {
		t.Fatal("SD4 not above SD3")
	}
	ln := sdLine(SourceWHO, Length, Male, 365, 4)
	sl, _ := ScoreOf(SourceWHO, Length, Male, 365, ln)
	if math.Abs(sl.Z-4) > 1e-9 {
		t.Errorf("length at SD4: z = %v, want 4", sl.Z)
	}
	lms, _ := lookupDays(SourceCDC, Weight, Male, 365)
	high := lms.m * 2
	sc2, _ := ScoreOf(SourceCDC, Weight, Male, 365, high)
	if want := lmsZ(lms.l, lms.m, lms.s, high); math.Abs(sc2.Z-want) > 1e-12 {
		t.Errorf("CDC z = %v, want the plain %v", sc2.Z, want)
	}
}

func TestScoreRejectsBadValues(t *testing.T) {
	for _, v := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		_, err := ScoreOf(SourceWHO, Weight, Male, 100, v)
		if err == nil || errors.Is(err, ErrNotCovered) {
			t.Errorf("value %v: err %v, want a value error", v, err)
		}
	}
}

func TestPercentileMonotonic(t *testing.T) {
	for _, src := range Sources {
		for _, m := range []Measure{Weight, Length, Head, BMI} {
			for _, sex := range []Sex{Female, Male} {
				curves, err := Curves(src, m, sex, 0, 240)
				if err != nil {
					t.Fatalf("%s %s %s: %v", src, m, sex, err)
				}
				for i := 1; i < len(curves); i++ {
					if curves[i].Percentile <= curves[i-1].Percentile {
						t.Fatalf("%s %s: percentiles out of order", src, m)
					}
					for j := range curves[i].Points {
						if curves[i].Points[j].Value <= curves[i-1].Points[j].Value {
							t.Fatalf("%s %s %s at %v mo: P%v %v <= P%v %v", src, m, sex,
								curves[i].Points[j].AgeMonths, curves[i].Percentile, curves[i].Points[j].Value,
								curves[i-1].Percentile, curves[i-1].Points[j].Value)
						}
					}
				}
			}
		}
	}
	prev := -1.0
	for v := 2.0; v <= 20; v += 0.5 {
		sc, err := ScoreOf(SourceWHO, Weight, Female, 400, v)
		if err != nil {
			t.Fatal(err)
		}
		if sc.Percentile < prev || sc.Percentile < 0 || sc.Percentile > 100 {
			t.Fatalf("weight %v: percentile %v after %v", v, sc.Percentile, prev)
		}
		prev = sc.Percentile
	}
}

func TestScoreCurveRoundTrip(t *testing.T) {
	for _, src := range Sources {
		for _, m := range []Measure{Weight, Length, Head, BMI} {
			for _, sex := range []Sex{Female, Male} {
				curves, err := Curves(src, m, sex, 0, 240)
				if err != nil {
					t.Fatal(err)
				}
				for _, c := range curves {
					want := map[float64]float64{50: 0, 97: 1.881}[c.Percentile]
					if c.Percentile != 50 && c.Percentile != 97 {
						continue
					}
					for _, p := range c.Points {
						sc, err := scoreAtCurveAge(src, p.AgeMonths, m, sex, p.Value)
						if err != nil {
							t.Fatalf("%s %s %s %v mo: %v", src, m, sex, p.AgeMonths, err)
						}
						if math.Abs(sc.Z-want) > 0.01 {
							t.Errorf("%s %s %s %v mo P%v: z %v, want %v", src, m, sex, p.AgeMonths, c.Percentile, sc.Z, want)
						}
					}
				}
			}
		}
	}
}

// scoreAtCurveAge scores at the same table row the curve point was computed from.
func scoreAtCurveAge(src Source, months float64, m Measure, sex Sex, v float64) (Score, error) {
	if day := int(math.Round(months * constants.DaysPerMonth)); src == SourceWHO && day <= whoLastDay(m, sex) {
		return ScoreOf(src, m, sex, day, v)
	}
	return scoreAtMonths(src, m, sex, months, v)
}
