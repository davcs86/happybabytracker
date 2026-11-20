package auth

import (
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"testing"

	"golang.org/x/crypto/argon2"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

const goodPassword = "correct horse battery"

var sharedHash = sync.OnceValues(func() (string, error) { return HashPassword(goodPassword) })

func countingKDF(t *testing.T) *int {
	t.Helper()
	calls := 0
	orig := idKey
	idKey = func(_, _ []byte, _, _ uint32, _ uint8, keyLen uint32) []byte {
		calls++
		return make([]byte, keyLen)
	}
	t.Cleanup(func() { idKey = orig })
	return &calls
}

func TestHashVerifyRoundTrip(t *testing.T) {
	h, err := sharedHash()
	if err != nil {
		t.Fatal(err)
	}
	wantPrefix := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$", constants.Argon2MemoryKiB, constants.Argon2Iterations, constants.Argon2Parallelism)
	if !strings.HasPrefix(h, wantPrefix) {
		t.Fatalf("hash %q lacks PHC prefix %q", h, wantPrefix)
	}
	if strings.Count(h, "$") != 5 {
		t.Fatalf("hash %q is not 5-separator PHC", h)
	}
	if !VerifyPassword(h, goodPassword) {
		t.Fatal("correct password did not verify")
	}
}

func TestVerifyWrongPassword(t *testing.T) {
	h, err := sharedHash()
	if err != nil {
		t.Fatal(err)
	}
	if VerifyPassword(h, goodPassword+"x") {
		t.Fatal("wrong password verified")
	}
	if VerifyPassword(h, "") {
		t.Fatal("empty password verified")
	}
}

func TestHashesDiffer(t *testing.T) {
	a, err := sharedHash()
	if err != nil {
		t.Fatal(err)
	}
	b, err := HashPassword(goodPassword)
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatal("same password produced identical hashes")
	}
}

func TestVerifyUsesStoredParameters(t *testing.T) {
	salt := []byte("0123456789abcdef")
	key := argon2.IDKey([]byte(goodPassword), salt, 1, 64, 2, 16)
	enc := base64.RawStdEncoding
	h := fmt.Sprintf("$argon2id$v=19$m=64,t=1,p=2$%s$%s", enc.EncodeToString(salt), enc.EncodeToString(key))
	if !VerifyPassword(h, goodPassword) {
		t.Fatal("hash with non-default stored parameters did not verify")
	}
	if VerifyPassword(h, "other password!") {
		t.Fatal("wrong password verified against stored parameters")
	}
}

func TestVerifyMalformedNeverPanics(t *testing.T) {
	enc := base64.RawStdEncoding
	salt := enc.EncodeToString([]byte("0123456789abcdef"))
	key := enc.EncodeToString(make([]byte, 32))
	cases := map[string]string{
		"empty":         "",
		"garbage":       "not a hash",
		"too few parts": "$argon2id$v=19$m=19456,t=2,p=1$" + salt,
		"too many":      "$argon2id$v=19$m=19456,t=2,p=1$" + salt + "$" + key + "$extra",
		"wrong alg":     "$argon2i$v=19$m=19456,t=2,p=1$" + salt + "$" + key,
		"old version":   "$argon2id$v=16$m=19456,t=2,p=1$" + salt + "$" + key,
		"bad version":   "$argon2id$v=x$m=19456,t=2,p=1$" + salt + "$" + key,
		"zero time":     "$argon2id$v=19$m=19456,t=0,p=1$" + salt + "$" + key,
		"zero threads":  "$argon2id$v=19$m=19456,t=2,p=0$" + salt + "$" + key,
		"zero memory":   "$argon2id$v=19$m=0,t=2,p=1$" + salt + "$" + key,
		"non numeric":   "$argon2id$v=19$m=a,t=2,p=1$" + salt + "$" + key,
		"negative":      "$argon2id$v=19$m=-1,t=2,p=1$" + salt + "$" + key,
		"overflow":      "$argon2id$v=19$m=99999999999,t=2,p=1$" + salt + "$" + key,
		"threads 256":   "$argon2id$v=19$m=19456,t=2,p=256$" + salt + "$" + key,
		"extra param":   "$argon2id$v=19$m=19456,t=2,p=1,x=1$" + salt + "$" + key,
		"trailing text": "$argon2id$v=19$m=19456,t=2,p=1x$" + salt + "$" + key,
		"bad salt b64":  "$argon2id$v=19$m=19456,t=2,p=1$!!!$" + key,
		"bad hash b64":  "$argon2id$v=19$m=19456,t=2,p=1$" + salt + "$!!!",
		"empty salt":    "$argon2id$v=19$m=19456,t=2,p=1$$" + key,
		"empty hash":    "$argon2id$v=19$m=19456,t=2,p=1$" + salt + "$",
		"padded b64":    "$argon2id$v=19$m=19456,t=2,p=1$" + salt + "=$" + key,
	}
	calls := countingKDF(t)
	for name, h := range cases {
		if VerifyPassword(h, goodPassword) {
			t.Errorf("%s: malformed hash verified", name)
		}
	}
	if *calls != 0 {
		t.Fatalf("kdf ran %d times for malformed hashes", *calls)
	}
}

func TestPasswordLengthBounds(t *testing.T) {
	calls := countingKDF(t)
	for _, n := range []int{0, 11, 257, 300} {
		if _, err := HashPassword(strings.Repeat("a", n)); err != ErrPasswordLength {
			t.Errorf("len %d: err = %v, want ErrPasswordLength", n, err)
		}
	}
	if *calls != 0 {
		t.Fatalf("kdf ran %d times for out-of-range passwords", *calls)
	}
	for _, n := range []int{12, 256} {
		if _, err := HashPassword(strings.Repeat("a", n)); err != nil {
			t.Errorf("len %d: unexpected error %v", n, err)
		}
	}
	if *calls != 2 {
		t.Fatalf("kdf ran %d times, want 2", *calls)
	}
}

func TestPasswordLengthCountsCodePoints(t *testing.T) {
	calls := countingKDF(t)
	short := strings.Repeat("é", 11)
	if len(short) != 22 {
		t.Fatalf("fixture is %d bytes", len(short))
	}
	if _, err := HashPassword(short); err != ErrPasswordLength {
		t.Fatalf("11 runes (22 bytes): err = %v, want ErrPasswordLength", err)
	}
	if _, err := HashPassword(strings.Repeat("é", 12)); err != nil {
		t.Fatalf("12 runes: %v", err)
	}
	if _, err := HashPassword(strings.Repeat("é", 256)); err != nil {
		t.Fatalf("256 runes (512 bytes): %v", err)
	}
	if _, err := HashPassword(strings.Repeat("é", 257)); err != ErrPasswordLength {
		t.Fatalf("257 runes: err = %v, want ErrPasswordLength", err)
	}
	if *calls != 2 {
		t.Fatalf("kdf ran %d times, want 2", *calls)
	}
}

func TestErrorsCarryNoSecret(t *testing.T) {
	_, err := HashPassword("tiny-secret")
	if err == nil || strings.Contains(err.Error(), "tiny-secret") {
		t.Fatalf("error %v must not echo the password", err)
	}
}
