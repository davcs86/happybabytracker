// Package units converts feed volumes between millilitres and US fluid ounces.
package units

import (
	"errors"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

type Unit string

const (
	UnitML   Unit = "ml"
	UnitFlOz Unit = "floz"
)

// Volume holds both columns; Typed names the one the person entered, which is exact.
type Volume struct {
	ML             int64
	FlOzHundredths int64
	Typed          Unit
}

var (
	ErrUnit  = errors.New("units: unknown volume unit")
	ErrRange = errors.New("units: volume out of range")
)

const hundredthsPerFlOz = 100

// NewVolume takes whole millilitres for UnitML, hundredths of a US fl oz for UnitFlOz.
func NewVolume(unit Unit, amount int64) (Volume, error) {
	if unit != UnitML && unit != UnitFlOz {
		return Volume{}, ErrUnit
	}
	if amount < 1 || amount > constants.MaxVolumeAmount {
		return Volume{}, ErrRange
	}
	const (
		num = constants.FlOzMicroMLNumerator
		den = constants.FlOzMicroMLDenominator * hundredthsPerFlOz
	)
	v := Volume{Typed: unit}
	if unit == UnitML {
		v.ML = amount
		v.FlOzHundredths = roundHalfUp(amount*den, num)
	} else {
		v.FlOzHundredths = amount
		v.ML = roundHalfUp(amount*num, den)
	}
	if v.ML < 1 || v.FlOzHundredths < 1 {
		return Volume{}, ErrRange
	}
	return v, nil
}

func roundHalfUp(n, d int64) int64 {
	return (2*n + d) / (2 * d)
}
