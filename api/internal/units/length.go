package units

import "github.com/davcs86/happybabytracker/api/internal/constants"

const (
	UnitCM Unit = "cm"
	UnitIN Unit = "in"
)

const hundredthsPerInch = 100

// Length holds both columns; Typed names the one the person entered, which is exact.
type Length struct {
	MM           int64
	InHundredths int64
	Typed        Unit
}

// NewLength takes whole millimetres for UnitCM, hundredths of an inch for UnitIN.
func NewLength(unit Unit, amount int64) (Length, error) {
	if unit != UnitCM && unit != UnitIN {
		return Length{}, ErrUnit
	}
	if amount < 1 || amount > constants.MaxLengthAmount {
		return Length{}, ErrRange
	}
	const (
		num = constants.MMPerInchNumerator
		den = constants.MMPerInchDenominator * hundredthsPerInch
	)
	l := Length{Typed: unit}
	if unit == UnitCM {
		l.MM = amount
		l.InHundredths = roundHalfUp(amount*den, num)
	} else {
		l.InHundredths = amount
		l.MM = roundHalfUp(amount*num, den)
	}
	if l.MM < 1 || l.InHundredths < 1 {
		return Length{}, ErrRange
	}
	return l, nil
}
