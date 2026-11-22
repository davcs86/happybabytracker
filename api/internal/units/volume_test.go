package units

import (
	"errors"
	"testing"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

func TestNewVolumeML(t *testing.T) {
	v, err := NewVolume(UnitML, 120)
	if err != nil {
		t.Fatal(err)
	}
	if v.ML != 120 || v.FlOzHundredths != 406 || v.Typed != UnitML {
		t.Fatalf("got %+v", v)
	}
}

func TestNewVolumeFlOz(t *testing.T) {
	v, err := NewVolume(UnitFlOz, 400)
	if err != nil {
		t.Fatal(err)
	}
	if v.ML != 118 || v.FlOzHundredths != 400 || v.Typed != UnitFlOz {
		t.Fatalf("got %+v", v)
	}
}

func TestRoundingIsHalfUp(t *testing.T) {
	cases := []struct {
		name   string
		unit   Unit
		amount int64
		wantML int64
		wantH  int64
	}{
		{"1 ml rounds up to 3 hundredths", UnitML, 1, 1, 3},
		{"2 ml is 6.76 hundredths", UnitML, 2, 2, 7},
		{"15 hundredths is 4.44 ml", UnitFlOz, 15, 4, 15},
		{"17 hundredths is 5.03 ml", UnitFlOz, 17, 5, 17},
		{"4 hundredths is 1.18 ml", UnitFlOz, 4, 1, 4},
		{"exact tie of 29573.5 ml rounds up", UnitFlOz, 100000, 29574, 100000},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v, err := NewVolume(c.unit, c.amount)
			if err != nil {
				t.Fatal(err)
			}
			if v.ML != c.wantML || v.FlOzHundredths != c.wantH {
				t.Fatalf("got %+v, want ML %d FlOzHundredths %d", v, c.wantML, c.wantH)
			}
		})
	}
}

func TestRejects(t *testing.T) {
	cases := []struct {
		name    string
		unit    Unit
		amount  int64
		wantErr error
	}{
		{"zero ml", UnitML, 0, ErrRange},
		{"negative ml", UnitML, -1, ErrRange},
		{"zero floz", UnitFlOz, 0, ErrRange},
		{"negative floz", UnitFlOz, -1, ErrRange},
		{"ml above max", UnitML, constants.MaxVolumeAmount + 1, ErrRange},
		{"floz above max", UnitFlOz, constants.MaxVolumeAmount + 1, ErrRange},
		{"one hundredth rounds to 0 ml", UnitFlOz, 1, ErrRange},
		{"unknown unit", Unit("cup"), 5, ErrUnit},
		{"empty unit", Unit(""), 5, ErrUnit},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v, err := NewVolume(c.unit, c.amount)
			if !errors.Is(err, c.wantErr) {
				t.Fatalf("err = %v, want %v", err, c.wantErr)
			}
			if v != (Volume{}) {
				t.Fatalf("got %+v with an error, want zero Volume", v)
			}
		})
	}
}

func TestAcceptsMax(t *testing.T) {
	for _, u := range []Unit{UnitML, UnitFlOz} {
		if _, err := NewVolume(u, constants.MaxVolumeAmount); err != nil {
			t.Fatalf("%s: %v", u, err)
		}
	}
}

func TestTypedValueIsExact(t *testing.T) {
	const (
		num = constants.FlOzMicroMLNumerator
		den = constants.FlOzMicroMLDenominator * 100 // hundredths of a fl oz
	)
	for _, amount := range []int64{1, 2, 3, 7, 15, 29, 30, 120, 355, 400, 1000, 12345, constants.MaxVolumeAmount} {
		ml, err := NewVolume(UnitML, amount)
		if err != nil {
			t.Fatalf("ml %d: %v", amount, err)
		}
		if ml.ML != amount || ml.Typed != UnitML {
			t.Fatalf("ml %d: typed column changed: %+v", amount, ml)
		}
		// exact hundredths = amount*den/num; |h*num - amount*den| < num means within one unit.
		if d := abs(ml.FlOzHundredths*num - amount*den); d >= num {
			t.Fatalf("ml %d: hundredths %d off by %d/%d", amount, ml.FlOzHundredths, d, num)
		}

		oz, err := NewVolume(UnitFlOz, amount)
		if err != nil {
			continue // tiny amounts legitimately round to 0 ml
		}
		if oz.FlOzHundredths != amount || oz.Typed != UnitFlOz {
			t.Fatalf("floz %d: typed column changed: %+v", amount, oz)
		}
		// exact ml = amount*num/den; |ml*den - amount*num| < den.
		if d := abs(oz.ML*den - amount*num); d >= den {
			t.Fatalf("floz %d: ml %d off by %d/%d", amount, oz.ML, d, den)
		}
	}
}

func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
