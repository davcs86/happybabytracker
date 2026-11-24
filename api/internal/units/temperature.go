package units

import "github.com/davcs86/happybabytracker/api/internal/constants"

const (
	UnitC Unit = "c"
	UnitF Unit = "f"
)

const (
	fPerCNumerator   = 9
	fPerCDenominator = 5
	fAtZeroC         = 3200 // 32.00 °F, in hundredths
)

// Temperature holds both columns; Typed names the one the person entered, which is exact.
type Temperature struct {
	CHundredths int64
	FHundredths int64
	Typed       Unit
}

func cToF(c int64) int64 { return roundHalfUp(c*fPerCNumerator, fPerCDenominator) + fAtZeroC }

func fToC(f int64) int64 { return roundHalfUp((f-fAtZeroC)*fPerCDenominator, fPerCNumerator) }

// NewTemperature takes hundredths of a degree in the typed unit.
func NewTemperature(unit Unit, amount int64) (Temperature, error) {
	if unit != UnitC && unit != UnitF {
		return Temperature{}, ErrUnit
	}
	// Refuse far-out amounts before multiplying so nothing can overflow into range.
	lo, hi := int64(constants.MinTempCHundredths)-1, int64(constants.MaxTempCHundredths)+1
	if unit == UnitF {
		lo, hi = cToF(constants.MinTempCHundredths)-1, cToF(constants.MaxTempCHundredths)+1
	}
	if amount < lo || amount > hi {
		return Temperature{}, ErrRange
	}
	t := Temperature{Typed: unit}
	if unit == UnitC {
		t.CHundredths = amount
		t.FHundredths = cToF(amount)
	} else {
		t.FHundredths = amount
		t.CHundredths = fToC(amount)
	}
	if t.CHundredths < constants.MinTempCHundredths || t.CHundredths > constants.MaxTempCHundredths {
		return Temperature{}, ErrRange
	}
	return t, nil
}
