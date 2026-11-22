package units

import (
	"errors"
	"testing"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

func TestNewLength(t *testing.T) {
	l, err := NewLength(UnitCM, 500)
	if err != nil {
		t.Fatal(err)
	}
	if l.MM != 500 || l.InHundredths != 1969 || l.Typed != UnitCM {
		t.Fatalf("got %+v", l)
	}

	l, err = NewLength(UnitIN, 2000)
	if err != nil {
		t.Fatal(err)
	}
	if l.MM != 508 || l.InHundredths != 2000 || l.Typed != UnitIN {
		t.Fatalf("got %+v", l)
	}
}

func TestLengthRoundingIsHalfUp(t *testing.T) {
	cases := []struct {
		name   string
		unit   Unit
		amount int64
		wantMM int64
		wantH  int64
	}{
		{"1 mm is 3.94 hundredths", UnitCM, 1, 1, 4},
		{"2 mm is 7.87 hundredths", UnitCM, 2, 2, 8},
		{"6 mm is 23.62 hundredths", UnitCM, 6, 6, 24},
		{"127 mm is exactly 500 hundredths", UnitCM, 127, 127, 500},
		{"500 mm is 1968.50 hundredths", UnitCM, 500, 500, 1969},
		{"2 hundredths is 0.51 mm", UnitIN, 2, 1, 2},
		{"3 hundredths is 0.76 mm", UnitIN, 3, 1, 3},
		{"exact tie of 63.5 mm rounds up", UnitIN, 250, 64, 250},
		{"exact tie of 190.5 mm rounds up", UnitIN, 750, 191, 750},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l, err := NewLength(c.unit, c.amount)
			if err != nil {
				t.Fatal(err)
			}
			if l.MM != c.wantMM || l.InHundredths != c.wantH {
				t.Fatalf("got %+v, want MM %d InHundredths %d", l, c.wantMM, c.wantH)
			}
		})
	}
}

func TestLengthRejects(t *testing.T) {
	cases := []struct {
		name    string
		unit    Unit
		amount  int64
		wantErr error
	}{
		{"zero mm", UnitCM, 0, ErrRange},
		{"negative mm", UnitCM, -1, ErrRange},
		{"zero in", UnitIN, 0, ErrRange},
		{"negative in", UnitIN, -1, ErrRange},
		{"mm above max", UnitCM, constants.MaxLengthAmount + 1, ErrRange},
		{"in above max", UnitIN, constants.MaxLengthAmount + 1, ErrRange},
		{"one hundredth rounds to 0 mm", UnitIN, 1, ErrRange},
		{"unknown unit", Unit("ft"), 5, ErrUnit},
		{"empty unit", Unit(""), 5, ErrUnit},
		{"weight unit", UnitKG, 5, ErrUnit},
		{"unknown unit wins over a bad amount", Unit("ft"), 0, ErrUnit},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l, err := NewLength(c.unit, c.amount)
			if !errors.Is(err, c.wantErr) {
				t.Fatalf("err = %v, want %v", err, c.wantErr)
			}
			if l != (Length{}) {
				t.Fatalf("got %+v with an error, want zero Length", l)
			}
		})
	}
}

func TestLengthAcceptsMax(t *testing.T) {
	for _, u := range []Unit{UnitCM, UnitIN} {
		if _, err := NewLength(u, constants.MaxLengthAmount); err != nil {
			t.Fatalf("%s: %v", u, err)
		}
	}
}

func TestLengthTypedValueIsExact(t *testing.T) {
	const (
		num = constants.MMPerInchNumerator
		den = constants.MMPerInchDenominator * 100 // hundredths of an inch
	)
	for _, amount := range []int64{1, 2, 3, 7, 15, 29, 127, 250, 500, 2000, 12345, constants.MaxLengthAmount} {
		cm, err := NewLength(UnitCM, amount)
		if err != nil {
			t.Fatalf("cm %d: %v", amount, err)
		}
		if cm.MM != amount || cm.Typed != UnitCM {
			t.Fatalf("cm %d: typed column changed: %+v", amount, cm)
		}
		// exact hundredths = amount*den/num; |h*num - amount*den| < num means within one unit.
		if d := abs(cm.InHundredths*num - amount*den); d >= num {
			t.Fatalf("cm %d: hundredths %d off by %d/%d", amount, cm.InHundredths, d, num)
		}

		in, err := NewLength(UnitIN, amount)
		if err != nil {
			continue // tiny amounts legitimately round to 0 mm
		}
		if in.InHundredths != amount || in.Typed != UnitIN {
			t.Fatalf("in %d: typed column changed: %+v", amount, in)
		}
		// exact mm = amount*num/den; |mm*den - amount*num| < den.
		if d := abs(in.MM*den - amount*num); d >= den {
			t.Fatalf("in %d: mm %d off by %d/%d", amount, in.MM, d, den)
		}
	}
}
