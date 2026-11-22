package units

import (
	"errors"
	"testing"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

func TestNewWeight(t *testing.T) {
	w, err := NewWeight(UnitKG, 3200)
	if err != nil {
		t.Fatal(err)
	}
	if w.Grams != 3200 || w.OzHundredths != 11288 || w.Typed != UnitKG {
		t.Fatalf("got %+v", w)
	}

	w, err = NewWeight(UnitLB, 13150)
	if err != nil {
		t.Fatal(err)
	}
	if w.Grams != 3728 || w.OzHundredths != 13150 || w.Typed != UnitLB {
		t.Fatalf("got %+v", w)
	}
}

func TestWeightRoundingIsHalfUp(t *testing.T) {
	cases := []struct {
		name      string
		unit      Unit
		amount    int64
		wantGrams int64
		wantH     int64
	}{
		{"1 g is 3.53 hundredths", UnitKG, 1, 1, 4},
		{"2 g is 7.05 hundredths", UnitKG, 2, 2, 7},
		{"14 g is 49.38 hundredths", UnitKG, 14, 14, 49},
		{"15 g is 52.91 hundredths", UnitKG, 15, 15, 53},
		{"28 g is 98.77 hundredths", UnitKG, 28, 28, 99},
		{"2 hundredths is 0.57 g", UnitLB, 2, 1, 2},
		{"4 hundredths is 1.13 g", UnitLB, 4, 1, 4},
		{"5 hundredths is 1.42 g", UnitLB, 5, 1, 5},
		{"100 hundredths is 28.35 g", UnitLB, 100, 28, 100},
		{"200 hundredths is 56.70 g", UnitLB, 200, 57, 200},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w, err := NewWeight(c.unit, c.amount)
			if err != nil {
				t.Fatal(err)
			}
			if w.Grams != c.wantGrams || w.OzHundredths != c.wantH {
				t.Fatalf("got %+v, want Grams %d OzHundredths %d", w, c.wantGrams, c.wantH)
			}
		})
	}
}

func TestWeightRejects(t *testing.T) {
	cases := []struct {
		name    string
		unit    Unit
		amount  int64
		wantErr error
	}{
		{"zero g", UnitKG, 0, ErrRange},
		{"negative g", UnitKG, -1, ErrRange},
		{"zero lb", UnitLB, 0, ErrRange},
		{"negative lb", UnitLB, -1, ErrRange},
		{"g above max", UnitKG, constants.MaxWeightAmount + 1, ErrRange},
		{"lb above max", UnitLB, constants.MaxWeightAmount + 1, ErrRange},
		{"one hundredth rounds to 0 g", UnitLB, 1, ErrRange},
		{"unknown unit", Unit("st"), 5, ErrUnit},
		{"empty unit", Unit(""), 5, ErrUnit},
		{"length unit", UnitCM, 5, ErrUnit},
		{"unknown unit wins over a bad amount", Unit("st"), 0, ErrUnit},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w, err := NewWeight(c.unit, c.amount)
			if !errors.Is(err, c.wantErr) {
				t.Fatalf("err = %v, want %v", err, c.wantErr)
			}
			if w != (Weight{}) {
				t.Fatalf("got %+v with an error, want zero Weight", w)
			}
		})
	}
}

func TestWeightAcceptsMax(t *testing.T) {
	for _, u := range []Unit{UnitKG, UnitLB} {
		if _, err := NewWeight(u, constants.MaxWeightAmount); err != nil {
			t.Fatalf("%s: %v", u, err)
		}
	}
}

func TestWeightTypedValueIsExact(t *testing.T) {
	const (
		num = constants.GramsPerOzNumerator
		den = constants.GramsPerOzDenominator * 100 // hundredths of an ounce
	)
	for _, amount := range []int64{1, 2, 3, 7, 15, 29, 100, 3200, 13150, 12345, constants.MaxWeightAmount} {
		kg, err := NewWeight(UnitKG, amount)
		if err != nil {
			t.Fatalf("kg %d: %v", amount, err)
		}
		if kg.Grams != amount || kg.Typed != UnitKG {
			t.Fatalf("kg %d: typed column changed: %+v", amount, kg)
		}
		// exact hundredths = amount*den/num; |h*num - amount*den| < num means within one unit.
		if d := abs(kg.OzHundredths*num - amount*den); d >= num {
			t.Fatalf("kg %d: hundredths %d off by %d/%d", amount, kg.OzHundredths, d, num)
		}

		lb, err := NewWeight(UnitLB, amount)
		if err != nil {
			continue // tiny amounts legitimately round to 0 g
		}
		if lb.OzHundredths != amount || lb.Typed != UnitLB {
			t.Fatalf("lb %d: typed column changed: %+v", amount, lb)
		}
		// exact grams = amount*num/den; |g*den - amount*num| < den.
		if d := abs(lb.Grams*den - amount*num); d >= den {
			t.Fatalf("lb %d: grams %d off by %d/%d", amount, lb.Grams, d, den)
		}
	}
}
