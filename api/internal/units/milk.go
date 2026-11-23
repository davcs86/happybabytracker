package units

import "github.com/davcs86/happybabytracker/api/internal/constants"

// Milk is the estimated transfer between two weigh-ins, rounded once from the gram gain.
type Milk struct {
	GainGrams, GainOzHundredths, ML, FlOzHundredths int64
}

// MilkTransfer is false unless both weights exist and after is not lighter than before.
func MilkTransfer(before, after *Weight) (Milk, bool) {
	if before == nil || after == nil || after.Grams < before.Grams {
		return Milk{}, false
	}
	const (
		scale = 100
		num   = constants.FlOzMicroMLNumerator
		den   = constants.FlOzMicroMLDenominator * hundredthsPerFlOz
	)
	gain := after.Grams - before.Grams
	return Milk{
		GainGrams: gain,
		// Typed-in-pounds ounces differ from grams by rounding: never report a negative gain.
		GainOzHundredths: max(after.OzHundredths-before.OzHundredths, 0),
		ML:               roundHalfUp(gain*scale, constants.MilkDensityHundredths),
		FlOzHundredths:   roundHalfUp(gain*scale*den, constants.MilkDensityHundredths*num),
	}, true
}
