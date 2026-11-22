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
