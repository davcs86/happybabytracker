package model

import (
	"slices"
	"testing"

	"github.com/google/uuid"
)

func TestNewIDIsUUIDv7(t *testing.T) {
	a, b := NewID(), NewID()
	if a == b {
		t.Fatal("NewID repeated")
	}
	u, err := uuid.Parse(a)
	if err != nil || u.Version() != 7 {
		t.Fatalf("NewID %q: version %v err %v", a, u.Version(), err)
	}
}

func TestCanonicalContents(t *testing.T) {
	for _, c := range []struct {
		in, want []string
		ok       bool
	}{
		{[]string{ContentFormula, ContentBreastMilk}, []string{ContentBreastMilk, ContentFormula}, true},
		{[]string{ContentOther, ContentCowMilk, ContentFormula, ContentBreastMilk}, ContentNames, true},
		{[]string{ContentCowMilk}, []string{ContentCowMilk}, true},
		{nil, nil, false},
		{[]string{}, nil, false},
		{[]string{ContentFormula, ContentFormula}, nil, false},
		{[]string{ContentFormula, "goat_milk"}, nil, false},
		{[]string{"goat_milk"}, nil, false},
	} {
		got, ok := CanonicalContents(c.in)
		if ok != c.ok || (ok && !slices.Equal(got, c.want)) {
			t.Errorf("CanonicalContents(%v) = %v, %v; want %v, %v", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestPumpingSideSet(t *testing.T) {
	for _, c := range []struct {
		in, want []string
		ok       bool
	}{
		{[]string{"right", "left"}, []string{"left", "right"}, true},
		{[]string{"right"}, []string{"right"}, true},
		{nil, nil, false},
		{[]string{"left", "left"}, nil, false},
		{[]string{"both"}, nil, false},
		{[]string{"left", "middle"}, nil, false},
	} {
		got, ok := CanonicalPumpingSides(c.in)
		if ok != c.ok || (ok && !slices.Equal(got, c.want)) {
			t.Errorf("CanonicalPumpingSides(%v) = %v, %v; want %v, %v", c.in, got, ok, c.want, c.ok)
		}
	}
	side := func(names ...string) []PumpingSide {
		var out []PumpingSide
		for _, n := range names {
			out = append(out, PumpingSide{Side: n})
		}
		return out
	}
	for _, c := range []struct {
		in   []PumpingSide
		want string
		ok   bool
	}{
		{side("left"), "left", true},
		{side("right"), "right", true},
		{side("right", "left"), "both", true},
		{nil, "", false},
		{side("left", "left"), "", false},
		{side("both"), "", false},
	} {
		if got, ok := DerivedPumpingSide(c.in); got != c.want || ok != c.ok {
			t.Errorf("DerivedPumpingSide(%v) = %q, %v; want %q, %v", c.in, got, ok, c.want, c.ok)
		}
	}
}
