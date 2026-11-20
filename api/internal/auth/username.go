package auth

import (
	"regexp"
	"strings"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

var usernameRE = regexp.MustCompile(constants.UsernamePattern)

// NormalizeUsername rejects any non-ASCII input so Unicode case folding can never alias a name.
func NormalizeUsername(s string) (string, bool) {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return "", false
		}
	}
	b := []byte(strings.TrimSpace(s))
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 'a' - 'A'
		}
	}
	if !usernameRE.Match(b) {
		return "", false
	}
	return string(b), true
}
