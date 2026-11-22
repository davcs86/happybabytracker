package units

import "github.com/davcs86/happybabytracker/api/internal/constants"

const (
	UnitKG Unit = "kg"
	UnitLB Unit = "lb"
)

const hundredthsPerOz = 100

// Weight holds both columns; Typed names the one the person entered, which is exact.
type Weight struct {
	Grams        int64
	OzHundredths int64
	Typed        Unit
}

// NewWeight takes whole grams for UnitKG, hundredths of an ounce for UnitLB.
func NewWeight(unit Unit, amount int64) (Weight, error) {
	if unit != UnitKG && unit != UnitLB {
		return Weight{}, ErrUnit
	}
	if amount < 1 || amount > constants.MaxWeightAmount {
		return Weight{}, ErrRange
	}
	const (
		num = constants.GramsPerOzNumerator
		den = constants.GramsPerOzDenominator * hundredthsPerOz
	)
	w := Weight{Typed: unit}
	if unit == UnitKG {
		w.Grams = amount
		w.OzHundredths = roundHalfUp(amount*den, num)
	} else {
		w.OzHundredths = amount
		w.Grams = roundHalfUp(amount*num, den)
	}
	if w.Grams < 1 || w.OzHundredths < 1 {
		return Weight{}, ErrRange
	}
	return w, nil
}
