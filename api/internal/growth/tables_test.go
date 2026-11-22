package growth

import (
	"errors"
	"testing"
)

func TestTablesMatchPublishedLMS(t *testing.T) {
	cases := []struct {
		src     Source
		table   string
		sex     Sex
		age     float64
		l, m, s float64
	}{
		{SourceWHO, "bfa-month", Male, 132, -1.7862, 16.9392, 0.11070},
		{SourceWHO, "bfa-month", Male, 192, -1.3529, 20.4951, 0.12579},
		{SourceWHO, "bfa-month", Male, 108, -1.6318, 16.0490, 0.10038},
		{SourceCDC, "wtageinf", Male, 9.5, -0.1600954, 9.476500305, 0.11218624},
	}
	for _, c := range cases {
		tb := tableFor(c.src, c.table, c.sex)
		if tb == nil {
			t.Fatalf("%s %s %s: no table", c.src, c.table, c.sex)
		}
		found := false
		for _, r := range tb.rows {
			if r.age == c.age {
				found = true
				if r.l != c.l || r.m != c.m || r.s != c.s {
					t.Errorf("%s %s %s age %v: got (%v, %v, %v), want (%v, %v, %v)",
						c.src, c.table, c.sex, c.age, r.l, r.m, r.s, c.l, c.m, c.s)
				}
			}
		}
		if !found {
			t.Errorf("%s %s %s: no row at %v", c.src, c.table, c.sex, c.age)
		}
	}
}

func TestSeams(t *testing.T) {
	h730, err := lookupDays(SourceWHO, Length, Male, 730)
	if err != nil {
		t.Fatal(err)
	}
	h731, err := lookupDays(SourceWHO, Length, Male, 731)
	if err != nil {
		t.Fatal(err)
	}
	if h730.m != 87.8018 || h731.m != 87.1303 {
		t.Errorf("WHO length M at 730/731 = %v/%v, want 87.8018/87.1303", h730.m, h731.m)
	}
	if h730.region == h731.region {
		t.Errorf("WHO length 730 and 731 share region %q", h730.region)
	}
	b730, _ := lookupDays(SourceWHO, BMI, Male, 730)
	b731, _ := lookupDays(SourceWHO, BMI, Male, 731)
	if b730.region == b731.region {
		t.Errorf("WHO BMI 730 and 731 share region %q", b730.region)
	}

	for _, m := range []Measure{Weight, Length, BMI} {
		d, err := lookupDays(SourceWHO, m, Male, 1856)
		if err != nil || d.table != "wfa-day" && d.table != "lhfa-day" && d.table != "bfa-day" {
			t.Errorf("WHO %s day 1856: table %q err %v, want the 2006 day table", m, d.table, err)
		}
		mo, err := lookupDays(SourceWHO, m, Male, 1857)
		if err != nil || mo.table != "wfa-month" && mo.table != "hfa-month" && mo.table != "bfa-month" {
			t.Errorf("WHO %s day 1857: table %q err %v, want the 2007 month table", m, mo.table, err)
		}
	}

	inf, err := lookupMonths(SourceCDC, Weight, Male, 23.99)
	if err != nil || inf.table != "wtageinf" {
		t.Errorf("CDC weight 23.99 mo: table %q err %v, want wtageinf", inf.table, err)
	}
	child, err := lookupMonths(SourceCDC, Weight, Male, 24)
	if err != nil || child.table != "wtage" {
		t.Errorf("CDC weight 24 mo: table %q err %v, want wtage", child.table, err)
	}
	if child.m != 12.6707633 {
		t.Errorf("CDC weight 24 mo M = %v, want the wtage row 12.6707633", child.m)
	}

	if _, ok := interpolate(tableFor(SourceCDC, "lenageinf", Male), 36); ok {
		t.Error("lenageinf covers 36 months, want no row")
	}
	if _, ok := interpolate(tableFor(SourceCDC, "lenageinf", Male), 35.5); !ok {
		t.Error("lenageinf does not cover its last row 35.5")
	}
}

func TestCDCInterpolatesBetweenRows(t *testing.T) {
	h, err := lookupMonths(SourceCDC, Weight, Male, 9.75)
	if err != nil {
		t.Fatal(err)
	}
	// Rows 9.5 and 10.5 of wtageinf boys; 9.75 is a quarter of the way.
	lo, hi := 9.476500305, 9.835307701
	if want := lo + (hi-lo)*0.25; abs(h.m-want) > 1e-9 {
		t.Errorf("M at 9.75 = %v, want %v", h.m, want)
	}
}

func TestNotCovered(t *testing.T) {
	cases := []struct {
		name string
		src  Source
		m    Measure
		days int
	}{
		{"WHO head at 6 years", SourceWHO, Head, 6 * 365},
		{"WHO weight at 12 years", SourceWHO, Weight, 12 * 365},
		{"CDC BMI at 18 months", SourceCDC, BMI, 548},
		{"CDC head at 40 months", SourceCDC, Head, 1218},
		{"WHO below 0 days", SourceWHO, Weight, -1},
		{"CDC below 0 days", SourceCDC, Weight, -1},
		{"WHO beyond 19 years", SourceWHO, Length, 229 * 31},
		{"unknown source", Source("uk-who"), Weight, 100},
		{"unknown measure", SourceWHO, Measure("arm"), 100},
	}
	for _, c := range cases {
		for _, sex := range []Sex{Female, Male} {
			if _, err := ScoreOf(c.src, c.m, sex, c.days, 10); !errors.Is(err, ErrNotCovered) {
				t.Errorf("%s %s: err %v, want ErrNotCovered", c.name, sex, err)
			}
			if Covered(c.src, c.m, sex, c.days) {
				t.Errorf("%s %s: Covered true", c.name, sex)
			}
		}
	}
	if _, err := ScoreOf(SourceWHO, Weight, Sex("x"), 100, 5); !errors.Is(err, ErrNotCovered) {
		t.Errorf("unknown sex: err %v, want ErrNotCovered", err)
	}
	for _, ok := range []struct {
		src  Source
		m    Measure
		days int
	}{
		{SourceWHO, Head, 1856}, {SourceWHO, Weight, 3652}, {SourceCDC, BMI, 731}, {SourceCDC, Head, 1095},
		{SourceWHO, Weight, 0}, {SourceCDC, Length, 0},
	} {
		if !Covered(ok.src, ok.m, Male, ok.days) {
			t.Errorf("%s %s day %d: Covered false", ok.src, ok.m, ok.days)
		}
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
