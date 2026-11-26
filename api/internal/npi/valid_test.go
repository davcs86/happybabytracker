package npi

import "testing"

func TestValid(t *testing.T) {
	for s, want := range map[string]bool{
		"1234567893":  true,
		"9123456789":  true,
		"1234567890":  false,
		"123456789":   false,
		"12345678931": false,
		"123456789a":  false,
		"12345678９3":  false,
		"":            false,
	} {
		if got := Valid(s); got != want {
			t.Errorf("Valid(%q) = %v, want %v", s, got, want)
		}
	}
}
