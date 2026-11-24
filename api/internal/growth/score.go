package growth

import (
	"errors"
	"math"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/kidtime"
)

// Score is a z-score and its percentile (0–100).
type Score struct{ Z, Percentile float64 }

var errValue = errors.New("growth: value must be positive and finite")

// restricted: WHO weight-for-age and BMI-for-age use z* beyond ±3 SD; never length or head.
func restricted(src Source, m Measure) bool {
	return src == SourceWHO && (m == Weight || m == BMI)
}

// ScoreOf scores value (kg, cm, or kg/m² for bmi) at ageDays whole days of the scoring age.
func ScoreOf(s Source, m Measure, sex Sex, ageDays int, value float64) (Score, error) {
	sc, _, err := scoreOf(s, m, sex, ageDays, value)
	return sc, err
}

// scoreOf is ScoreOf plus the name of the table that scored the value.
func scoreOf(s Source, m Measure, sex Sex, ageDays int, value float64) (Score, string, error) {
	h, err := lookupDays(s, m, sex, ageDays)
	if err != nil {
		return Score{}, "", err
	}
	sc, err := scoreHit(s, m, h, value)
	return sc, h.table, err
}

// MeasureScore is a score rounded as the API shows it, or the reason ("before_term", "not_covered") there is none.
type MeasureScore struct {
	Scored        bool
	Z, Percentile float64
	Reason        string
	Table         string
}

// RowScores has a score per measure the row holds; BMI and BMIValue only with both weight and length.
type RowScores struct {
	Weight, Length, Head, BMI *MeasureScore
	BMIValue                  *float64
}

// roundTo rounds x to decimals places; never -0.
func roundTo(x float64, decimals int) float64 {
	p := math.Pow(10, float64(decimals))
	if r := math.Round(x*p) / p; r != 0 {
		return r
	}
	return 0
}

// ScoreRow scores a measurement's metric columns at basis against src; an error is never a per-measure verdict.
func ScoreRow(src Source, sex Sex, basis kidtime.AgeBasis, weightG *int64, lengthMM, headMM *int64) (RowScores, error) {
	var out RowScores
	score := func(m Measure, value float64) (*MeasureScore, error) {
		if basis.ScoringDays < 0 {
			return &MeasureScore{Reason: "before_term"}, nil
		}
		s, table, err := scoreOf(src, m, sex, basis.ScoringDays, value)
		if errors.Is(err, ErrNotCovered) {
			return &MeasureScore{Reason: "not_covered"}, nil
		}
		if err != nil {
			return nil, err
		}
		return &MeasureScore{Scored: true, Z: roundTo(s.Z, constants.ZScoreDecimals),
			Percentile: roundTo(s.Percentile, constants.PercentileDecimals), Table: table}, nil
	}
	var err error
	if weightG != nil {
		if out.Weight, err = score(Weight, float64(*weightG)/1000); err != nil {
			return RowScores{}, err
		}
	}
	if lengthMM != nil {
		if out.Length, err = score(Length, float64(*lengthMM)/10); err != nil {
			return RowScores{}, err
		}
	}
	if headMM != nil {
		if out.Head, err = score(Head, float64(*headMM)/10); err != nil {
			return RowScores{}, err
		}
	}
	if weightG != nil && lengthMM != nil {
		bmi := float64(*weightG) * 1000 / (float64(*lengthMM) * float64(*lengthMM))
		if out.BMI, err = score(BMI, bmi); err != nil {
			return RowScores{}, err
		}
		value := roundTo(bmi, constants.BMIDecimals)
		out.BMIValue = &value
	}
	return out, nil
}

// Covered reports whether a table row covers the source, measure, sex and age.
func Covered(s Source, m Measure, sex Sex, ageDays int) bool {
	_, err := lookupDays(s, m, sex, ageDays)
	return err == nil
}

func scoreAtMonths(s Source, m Measure, sex Sex, months float64, value float64) (Score, error) {
	h, err := lookupMonths(s, m, sex, months)
	if err != nil {
		return Score{}, err
	}
	return scoreHit(s, m, h, value)
}

func scoreHit(s Source, m Measure, h hit, value float64) (Score, error) {
	if !(value > 0) || math.IsInf(value, 0) {
		return Score{}, errValue
	}
	z := lmsZ(h.l, h.m, h.s, value)
	if restricted(s, m) {
		z = restrictedZ(h.l, h.m, h.s, value)
	}
	return Score{Z: z, Percentile: 100 * phi(z)}, nil
}

func lmsZ(l, m, s, y float64) float64 {
	if l == 0 {
		return math.Log(y/m) / s
	}
	return (math.Pow(y/m, l) - 1) / (l * s)
}

// lmsValue is the plain inverse, M(1+L·S·z)^(1/L).
func lmsValue(l, m, s, z float64) float64 {
	if l == 0 {
		return m * math.Exp(s*z)
	}
	return m * math.Pow(1+l*s*z, 1/l)
}

func restrictedZ(l, m, s, y float64) float64 {
	z := lmsZ(l, m, s, y)
	switch {
	case z > 3:
		sd3, sd2 := lmsValue(l, m, s, 3), lmsValue(l, m, s, 2)
		return 3 + (y-sd3)/(sd3-sd2)
	case z < -3:
		sd3, sd2 := lmsValue(l, m, s, -3), lmsValue(l, m, s, -2)
		return -3 + (y-sd3)/(sd2-sd3)
	}
	return z
}

// valueAtZ inverts z (or z* when restricted): beyond ±3 SD the line steps by the 2-to-3 SD distance.
func valueAtZ(h hit, z float64, restrict bool) float64 {
	if !restrict || math.Abs(z) <= 3 {
		return lmsValue(h.l, h.m, h.s, z)
	}
	k := math.Copysign(3, z)
	sd3, sd2 := lmsValue(h.l, h.m, h.s, k), lmsValue(h.l, h.m, h.s, k*2/3)
	return sd3 + (math.Abs(z)-3)*(sd3-sd2)
}

// sdLine is the value at k SD at ageDays; NaN when not covered.
func sdLine(s Source, m Measure, sex Sex, ageDays int, k float64) float64 {
	h, err := lookupDays(s, m, sex, ageDays)
	if err != nil {
		return math.NaN()
	}
	return valueAtZ(h, k, restricted(s, m))
}

// valueAtPercentile is the value at percentile p (0–100) at ageDays.
func valueAtPercentile(s Source, m Measure, sex Sex, ageDays int, p float64) (float64, error) {
	h, err := lookupDays(s, m, sex, ageDays)
	if err != nil {
		return 0, err
	}
	return valueAtZ(h, normalQuantile(p/100), restricted(s, m)), nil
}

// phi is the standard normal CDF.
func phi(z float64) float64 { return 0.5 * math.Erfc(-z/math.Sqrt2) }

// normalQuantile is the inverse of phi, for 0 < p < 1.
func normalQuantile(p float64) float64 { return -math.Sqrt2 * math.Erfcinv(2*p) }
