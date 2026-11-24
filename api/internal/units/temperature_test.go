package units

import (
	"errors"
	"math"
	"testing"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

func TestNewTemperatureC(t *testing.T) {
	v, err := NewTemperature(UnitC, 3660)
	if err != nil {
		t.Fatal(err)
	}
	if v.CHundredths != 3660 || v.FHundredths != 9788 || v.Typed != UnitC {
		t.Fatalf("got %+v", v)
	}
}

func TestNewTemperatureF(t *testing.T) {
	v, err := NewTemperature(UnitF, 10040)
	if err != nil {
		t.Fatal(err)
	}
	if v.CHundredths != 3800 || v.FHundredths != 10040 || v.Typed != UnitF {
		t.Fatalf("got %+v", v)
	}
}

func TestTemperatureBounds(t *testing.T) {
	cases := []struct {
		name    string
		unit    Unit
		amount  int64
		wantErr error
	}{
		{"min c", UnitC, constants.MinTempCHundredths, nil},
		{"max c", UnitC, constants.MaxTempCHundredths, nil},
		{"below min c", UnitC, constants.MinTempCHundredths - 1, ErrRange},
		{"above max c", UnitC, constants.MaxTempCHundredths + 1, ErrRange},
		{"30.00 c in f", UnitF, 8600, nil},
		{"45.00 c in f", UnitF, 11300, nil},
		{"just below 30.00 c in f", UnitF, 8599, ErrRange},
		{"just above 45.00 c in f", UnitF, 11301, ErrRange},
		{"zero c", UnitC, 0, ErrRange},
		{"negative c", UnitC, -1, ErrRange},
		{"zero f", UnitF, 0, ErrRange},
		{"negative f", UnitF, -1, ErrRange},
		{"1<<40 c", UnitC, 1 << 40, ErrRange},
		{"1<<40 f", UnitF, 1 << 40, ErrRange},
		{"max int64 c", UnitC, math.MaxInt64, ErrRange},
		{"max int64 f", UnitF, math.MaxInt64, ErrRange},
		{"min int64 c", UnitC, math.MinInt64, ErrRange},
		{"min int64 f", UnitF, math.MinInt64, ErrRange},
		{"unknown unit", Unit("k"), 3660, ErrUnit},
		{"empty unit", Unit(""), 3660, ErrUnit},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v, err := NewTemperature(c.unit, c.amount)
			if !errors.Is(err, c.wantErr) {
				t.Fatalf("err = %v, want %v", err, c.wantErr)
			}
			if err != nil && v != (Temperature{}) {
				t.Fatalf("got %+v with error", v)
			}
		})
	}
}

// Neither conversion has an exact tie (fifths and ninths), so the table pins the rounding either side of one.
func TestTemperatureRounding(t *testing.T) {
	cases := []struct {
		name   string
		unit   Unit
		amount int64
		wantC  int64
		wantF  int64
	}{
		{"c .8 rounds up", UnitC, 3001, 3001, 8602},
		{"c .6 rounds up", UnitC, 3002, 3002, 8604},
		{"c .4 rounds down", UnitC, 3003, 3003, 8605},
		{"c .2 rounds down", UnitC, 3004, 3004, 8607},
		{"f .56 rounds up", UnitF, 8601, 3001, 8601},
		{"f .11 rounds down", UnitF, 8602, 3001, 8602},
		{"f .67 rounds up", UnitF, 8603, 3002, 8603},
		{"f .22 rounds down", UnitF, 8604, 3002, 8604},
		{"f .78 rounds up", UnitF, 8605, 3003, 8605},
		{"f .33 rounds down", UnitF, 8606, 3003, 8606},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v, err := NewTemperature(c.unit, c.amount)
			if err != nil {
				t.Fatal(err)
			}
			if v.CHundredths != c.wantC || v.FHundredths != c.wantF {
				t.Fatalf("got %+v, want C %d F %d", v, c.wantC, c.wantF)
			}
		})
	}
}

func TestRoundHalfUpTies(t *testing.T) {
	cases := []struct{ n, d, want int64 }{
		{5, 2, 3}, {1, 2, 1}, {3, 2, 2}, {7, 5, 1}, {8, 5, 2}, {5, 10, 1}, {4, 10, 0},
	}
	for _, c := range cases {
		if got := roundHalfUp(c.n, c.d); got != c.want {
			t.Fatalf("roundHalfUp(%d, %d) = %d, want %d", c.n, c.d, got, c.want)
		}
	}
}

func TestTemperatureRoundTripWithinOneHundredth(t *testing.T) {
	for c := int64(constants.MinTempCHundredths); c <= constants.MaxTempCHundredths; c++ {
		tc, err := NewTemperature(UnitC, c)
		if err != nil {
			t.Fatalf("c %d: %v", c, err)
		}
		back, err := NewTemperature(UnitF, tc.FHundredths)
		if err != nil {
			t.Fatalf("c %d via f %d: %v", c, tc.FHundredths, err)
		}
		if d := back.CHundredths - c; d < -1 || d > 1 {
			t.Fatalf("c %d -> f %d -> c %d", c, tc.FHundredths, back.CHundredths)
		}
	}
}
