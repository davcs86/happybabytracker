// Package npi searches the US national provider registry (NPPES) and checks NPI numbers.
package npi

import "github.com/davcs86/happybabytracker/api/internal/constants"

// npiPrefix is the card-issuer prefix the NPI check digit is computed over (Luhn of 80840 + the first 9 digits).
const npiPrefix = "80840"

// Valid is whether s is ten ASCII digits whose last is the NPI check digit.
func Valid(s string) bool {
	if len(s) != constants.ProviderNPILen {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	digits := npiPrefix + s[:len(s)-1]
	sum := 0
	for i := len(digits) - 1; i >= 0; i-- {
		d := int(digits[i] - '0')
		if (len(digits)-1-i)%2 == 0 {
			if d *= 2; d > 9 {
				d -= 9
			}
		}
		sum += d
	}
	return (10-sum%10)%10 == int(s[len(s)-1]-'0')
}
