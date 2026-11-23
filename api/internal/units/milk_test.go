package units

import "testing"

func w(t *testing.T, unit Unit, amount int64) *Weight {
	t.Helper()
	v, err := NewWeight(unit, amount)
	if err != nil {
		t.Fatal(err)
	}
	return &v
}

func TestMilkTransfer(t *testing.T) {
	m, ok := MilkTransfer(w(t, UnitKG, 3400), w(t, UnitKG, 3462))
	if !ok || m.GainGrams != 62 || m.ML != 60 || m.FlOzHundredths != 204 || m.GainOzHundredths != 219 {
		t.Fatalf("62 g gain = %+v, %v", m, ok)
	}

	m, ok = MilkTransfer(w(t, UnitKG, 3400), w(t, UnitKG, 3400))
	if !ok || m != (Milk{}) {
		t.Fatalf("equal weights = %+v, %v; want the zero estimate", m, ok)
	}

	if _, ok := MilkTransfer(w(t, UnitKG, 3400), w(t, UnitKG, 3399)); ok {
		t.Error("an after weight below before gave an estimate")
	}
	if _, ok := MilkTransfer(nil, w(t, UnitKG, 3400)); ok {
		t.Error("a missing before weight gave an estimate")
	}
	if _, ok := MilkTransfer(w(t, UnitKG, 3400), nil); ok {
		t.Error("a missing after weight gave an estimate")
	}
	if _, ok := MilkTransfer(nil, nil); ok {
		t.Error("no weights gave an estimate")
	}
}

func TestMilkRoundsHalfUpOnceFromGrams(t *testing.T) {
	cases := []struct {
		gain      int64
		ml, flozH int64
	}{
		{1, 1, 3},
		{5, 5, 16},
		{17, 17, 56},       // 16.50 ml rounds up
		{52, 50, 171},      // 50.49 ml rounds down
		{53, 51, 174},      // 51.46 ml; 173.99 hundredths rounds up
		{86, 83, 282},      // 83.495 ml rounds down
		{281, 273, 922},    // 922.49999 hundredths rounds down
		{843, 818, 2767},   // 2767.49996 rounds down
		{2886, 2802, 9475}, // 9474.50166 rounds up
		{103, 100, 338},
	}
	for _, c := range cases {
		m, ok := MilkTransfer(w(t, UnitKG, 1000), w(t, UnitKG, 1000+c.gain))
		if !ok || m.GainGrams != c.gain || m.ML != c.ml || m.FlOzHundredths != c.flozH {
			t.Errorf("gain %d g = %+v, %v; want ml %d, floz_h %d", c.gain, m, ok, c.ml, c.flozH)
		}
	}
}

func TestMilkOuncesComeFromTheColumns(t *testing.T) {
	// 7 lb 8 oz and 7 lb 9.5 oz: 1.5 oz = 150 hundredths, 42 g
	m, ok := MilkTransfer(w(t, UnitLB, 12000), w(t, UnitLB, 12150))
	if !ok || m.GainOzHundredths != 150 || m.GainGrams != 42 {
		t.Fatalf("lb gain = %+v, %v", m, ok)
	}

	// 706 hundredths is 200 g; 200 g is 705 hundredths: the ounce gain is clamped, never negative.
	m, ok = MilkTransfer(w(t, UnitLB, 706), w(t, UnitKG, 200))
	if !ok || m != (Milk{}) {
		t.Fatalf("rounding-only difference = %+v, %v; want the zero estimate", m, ok)
	}

	for before := int64(706); before < 4000; before++ {
		b := w(t, UnitLB, before)
		a := w(t, UnitKG, b.Grams)
		m, ok := MilkTransfer(b, a)
		if !ok || m.GainOzHundredths < 0 || m.GainGrams != 0 {
			t.Fatalf("lb %d then %d g = %+v, %v", before, b.Grams, m, ok)
		}
	}
}
