package growth

import (
	"errors"
	"math"
	"testing"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/kidtime"
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

func TestWHORestrictionOnlyWeightAndBMI(t *testing.T) {
	plainZ := func(r row, y float64) float64 {
		if r.l == 0 {
			return math.Log(y/r.m) / r.s
		}
		return (math.Pow(y/r.m, r.l) - 1) / (r.l * r.s)
	}
	rowAt := func(name string, age float64) row {
		for _, r := range tableFor(SourceWHO, name, Male).rows {
			if r.age == age {
				return r
			}
		}
		t.Fatalf("%s: no row at %v", name, age)
		return row{}
	}
	cases := []struct {
		m          Measure
		table      string
		age        float64
		day        bool
		restricted bool
		factors    [2]float64
	}{
		{Length, "lhfa-day", 400, true, false, [2]float64{1.3, 0.7}},
		{Length, "lhfa-day", 1000, true, false, [2]float64{1.3, 0.7}},
		{Head, "hcfa-day", 400, true, false, [2]float64{1.3, 0.7}},
		{Length, "hfa-month", 100, false, false, [2]float64{1.3, 0.7}},
		{Weight, "wfa-day", 400, true, true, [2]float64{2, 0.5}},
		{BMI, "bfa-day", 400, true, true, [2]float64{2, 0.5}},
		{Weight, "wfa-month", 100, false, true, [2]float64{2, 0.5}},
		{BMI, "bfa-month", 100, false, true, [2]float64{2, 0.5}},
	}
	for _, c := range cases {
		r := rowAt(c.table, c.age)
		for _, f := range c.factors {
			y := r.m * f
			want := plainZ(r, y)
			if math.Abs(want) <= 3 {
				t.Fatalf("%s %v ×%v: plain z %v is within ±3", c.table, c.age, f, want)
			}
			var sc Score
			var err error
			if c.day {
				sc, err = ScoreOf(SourceWHO, c.m, Male, int(c.age), y)
			} else {
				sc, err = scoreAtMonths(SourceWHO, c.m, Male, c.age, y)
			}
			if err != nil {
				t.Fatalf("%s %v: %v", c.table, c.age, err)
			}
			if same := math.Abs(sc.Z-want) < 1e-9; same == c.restricted {
				t.Errorf("%s %v ×%v: z %v, plain z %v, restricted %v", c.table, c.age, f, sc.Z, want, c.restricted)
			}
		}
	}
}

func TestRestrictedPredicate(t *testing.T) {
	// Pinned directly: the bundled WHO length, height and head tables have L = 1, where z* equals the plain z,
	// so only the predicate can show the restriction reaching them. One predicate serves the 2006 and 2007 tables.
	want := map[Source]map[Measure]bool{
		SourceWHO: {Weight: true, BMI: true, Length: false, Head: false},
		SourceCDC: {Weight: false, BMI: false, Length: false, Head: false},
	}
	for src, ms := range want {
		for m, w := range ms {
			if got := restricted(src, m); got != w {
				t.Errorf("restricted(%s, %s) = %v, want %v", src, m, got, w)
			}
		}
	}
}

// TestScoreRow replays the rows of the httpapi growth tests: the same reasons, rounding and BMI.
func TestScoreRow(t *testing.T) {
	i := func(v int64) *int64 { return &v }
	r := func(x, scale float64) float64 { return math.Round(x*scale) / scale }
	bmi := func(g, mm float64) float64 { return g * 1000 / (mm * mm) }
	scored := func(src Source, m Measure, sex Sex, days int, value float64, table string) *MeasureScore {
		s, err := ScoreOf(src, m, sex, days, value)
		if err != nil {
			t.Fatalf("engine %s %s %d: %v", src, m, days, err)
		}
		return &MeasureScore{Scored: true, Z: r(s.Z, 100), Percentile: r(s.Percentile, 10), Table: table}
	}
	reason := func(why string) *MeasureScore { return &MeasureScore{Reason: why} }
	f := func(v float64) *float64 { return &v }
	term := func(days int) kidtime.AgeBasis { return kidtime.AgeBasis{ChronologicalDays: days, ScoringDays: days} }
	corrected := func(chrono, scoring int) kidtime.AgeBasis {
		return kidtime.AgeBasis{ChronologicalDays: chrono, ScoringDays: scoring, Corrected: true}
	}
	cases := []struct {
		name              string
		src               Source
		sex               Sex
		basis             kidtime.AgeBasis
		weight, len, head *int64
		want              RowScores
	}{
		{"who female at 89 days", SourceWHO, Female, term(89), i(5000), i(600), i(400), RowScores{
			Weight: scored(SourceWHO, Weight, Female, 89, 5, "wfa-day"), Length: scored(SourceWHO, Length, Female, 89, 60, "lhfa-day"),
			Head: scored(SourceWHO, Head, Female, 89, 40, "hcfa-day"), BMI: scored(SourceWHO, BMI, Female, 89, bmi(5000, 600), "bfa-day"),
			BMIValue: f(13.89)}},
		{"cdc female at 89 days", SourceCDC, Female, term(89), i(5000), i(600), i(400), RowScores{
			Weight: scored(SourceCDC, Weight, Female, 89, 5, "wtageinf"), Length: scored(SourceCDC, Length, Female, 89, 60, "lenageinf"),
			Head: scored(SourceCDC, Head, Female, 89, 40, "hcageinf"), BMI: reason("not_covered"), BMIValue: f(13.89)}},
		{"corrected at three months", SourceWHO, Female, corrected(92, 59), i(5000), i(600), nil, RowScores{
			Weight: scored(SourceWHO, Weight, Female, 59, 5, "wfa-day"), Length: scored(SourceWHO, Length, Female, 59, 60, "lhfa-day"),
			BMI: scored(SourceWHO, BMI, Female, 59, bmi(5000, 600), "bfa-day"), BMIValue: f(13.89)}},
		{"corrected the day before the second birthday", SourceWHO, Female, corrected(729, 696), i(5000), i(600), nil, RowScores{
			Weight: scored(SourceWHO, Weight, Female, 696, 5, "wfa-day"), Length: scored(SourceWHO, Length, Female, 696, 60, "lhfa-day"),
			BMI: scored(SourceWHO, BMI, Female, 696, bmi(5000, 600), "bfa-day"), BMIValue: f(13.89)}},
		{"the second birthday", SourceWHO, Female, term(730), i(5000), i(600), nil, RowScores{
			Weight: scored(SourceWHO, Weight, Female, 730, 5, "wfa-day"), Length: scored(SourceWHO, Length, Female, 730, 60, "lhfa-day"),
			BMI: scored(SourceWHO, BMI, Female, 730, bmi(5000, 600), "bfa-day"), BMIValue: f(13.89)}},
		{"before term", SourceWHO, Female, corrected(10, -23), i(5000), i(600), i(400), RowScores{
			Weight: reason("before_term"), Length: reason("before_term"), Head: reason("before_term"), BMI: reason("before_term"),
			BMIValue: f(13.89)}},
		{"cdc at 18 months", SourceCDC, Female, term(546), i(11000), i(810), i(470), RowScores{
			Weight: scored(SourceCDC, Weight, Female, 546, 11, "wtageinf"), Length: scored(SourceCDC, Length, Female, 546, 81, "lenageinf"),
			Head: scored(SourceCDC, Head, Female, 546, 47, "hcageinf"), BMI: reason("not_covered"), BMIValue: f(r(bmi(11000, 810), 100))}},
		{"who male at 6 years", SourceWHO, Male, term(2191), i(20000), i(1150), i(510), RowScores{
			Weight: scored(SourceWHO, Weight, Male, 2191, 20, "wfa-month"), Length: scored(SourceWHO, Length, Male, 2191, 115, "hfa-month"),
			Head: reason("not_covered"), BMI: scored(SourceWHO, BMI, Male, 2191, bmi(20000, 1150), "bfa-month"),
			BMIValue: f(r(bmi(20000, 1150), 100))}},
		{"weight only", SourceWHO, Female, term(89), i(5000), nil, nil, RowScores{Weight: scored(SourceWHO, Weight, Female, 89, 5, "wfa-day")}},
		{"length only", SourceWHO, Female, term(89), nil, i(600), nil, RowScores{Length: scored(SourceWHO, Length, Female, 89, 60, "lhfa-day")}},
		{"weight and head", SourceWHO, Female, term(89), i(5000), nil, i(400), RowScores{
			Weight: scored(SourceWHO, Weight, Female, 89, 5, "wfa-day"), Head: scored(SourceWHO, Head, Female, 89, 40, "hcfa-day")}},
	}
	for _, c := range cases {
		got, err := ScoreRow(c.src, c.sex, c.basis, c.weight, c.len, c.head)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		for _, m := range []struct {
			name      string
			got, want *MeasureScore
		}{{"weight", got.Weight, c.want.Weight}, {"length", got.Length, c.want.Length}, {"head", got.Head, c.want.Head}, {"bmi", got.BMI, c.want.BMI}} {
			if (m.got == nil) != (m.want == nil) || (m.got != nil && *m.got != *m.want) {
				t.Errorf("%s %s = %+v, want %+v", c.name, m.name, m.got, m.want)
			}
		}
		if (got.BMIValue == nil) != (c.want.BMIValue == nil) || (got.BMIValue != nil && *got.BMIValue != *c.want.BMIValue) {
			t.Errorf("%s bmi value = %v, want %v", c.name, got.BMIValue, c.want.BMIValue)
		}
	}
	if s, _ := ScoreRow(SourceWHO, Female, term(89), i(5000), nil, nil); s.Weight.Z == 0 && s.Weight.Percentile == 0 {
		t.Fatal("no score")
	}
	if got := roundTo(-0.001, constants.ZScoreDecimals); math.Signbit(got) {
		t.Errorf("roundTo(-0.001) = %v, want +0", got)
	}
}
