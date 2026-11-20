// Package auth holds the pure credential primitives: passwords, tokens, the login limiter and the admin check.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

var ErrPasswordLength = errors.New("password length out of range")

// Replaced in tests to prove a refused password never reaches the KDF.
var idKey = argon2.IDKey

var b64 = base64.RawStdEncoding

// HashPassword counts code points and refuses out-of-range input before any hashing.
func HashPassword(pw string) (string, error) {
	n := utf8.RuneCountInString(pw)
	if n < constants.MinPasswordLen || n > constants.MaxPasswordLen {
		return "", ErrPasswordLength
	}
	salt := make([]byte, constants.Argon2SaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("read salt: %w", err)
	}
	key := idKey([]byte(pw), salt, constants.Argon2Iterations, constants.Argon2MemoryKiB, constants.Argon2Parallelism, constants.Argon2KeyBytes)
	return fmt.Sprintf("$argon2id$v=%d$%s$%s$%s", argon2.Version,
		formatParams(constants.Argon2MemoryKiB, constants.Argon2Iterations, constants.Argon2Parallelism),
		b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

// VerifyPassword takes its parameters from the stored hash and returns false on anything malformed.
func VerifyPassword(hash, pw string) bool {
	parts := strings.Split(hash, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return false
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version || parts[2] != fmt.Sprintf("v=%d", version) {
		return false
	}
	var m, t uint32
	var p uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil || parts[3] != formatParams(m, t, p) {
		return false
	}
	if m == 0 || t == 0 || p == 0 {
		return false
	}
	salt, err := b64.DecodeString(parts[4])
	if err != nil || len(salt) == 0 {
		return false
	}
	want, err := b64.DecodeString(parts[5])
	if err != nil || len(want) == 0 {
		return false
	}
	got := idKey([]byte(pw), salt, t, m, p, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1
}

func formatParams(m, t uint32, p uint8) string {
	return fmt.Sprintf("m=%d,t=%d,p=%d", m, t, p)
}
