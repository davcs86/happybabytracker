package auth

import (
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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
	key := argon2.IDKey([]byte(goodPassword), salt, 1, 64, 1, constants.Argon2KeyBytes)
	enc := base64.RawStdEncoding
	h := fmt.Sprintf("$argon2id$v=19$m=64,t=1,p=1$%s$%s", enc.EncodeToString(salt), enc.EncodeToString(key))
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
		"memory 4GiB":   "$argon2id$v=19$m=4194304,t=2,p=1$" + salt + "$" + key,
		"memory +1":     "$argon2id$v=19$m=19457,t=2,p=1$" + salt + "$" + key,
		"time max":      "$argon2id$v=19$m=19456,t=4294967295,p=1$" + salt + "$" + key,
		"time +1":       "$argon2id$v=19$m=19456,t=3,p=1$" + salt + "$" + key,
		"threads 255":   "$argon2id$v=19$m=19456,t=2,p=255$" + salt + "$" + key,
		"threads +1":    "$argon2id$v=19$m=19456,t=2,p=2$" + salt + "$" + key,
		"1-byte key":    "$argon2id$v=19$m=19456,t=2,p=1$" + salt + "$" + enc.EncodeToString([]byte{1}),
		"31-byte key":   "$argon2id$v=19$m=19456,t=2,p=1$" + salt + "$" + enc.EncodeToString(make([]byte, 31)),
		"33-byte key":   "$argon2id$v=19$m=19456,t=2,p=1$" + salt + "$" + enc.EncodeToString(make([]byte, 33)),
		"1-byte salt":   "$argon2id$v=19$m=19456,t=2,p=1$" + enc.EncodeToString([]byte{1}) + "$" + key,
		"15-byte salt":  "$argon2id$v=19$m=19456,t=2,p=1$" + enc.EncodeToString(make([]byte, 15)) + "$" + key,
		"17-byte salt":  "$argon2id$v=19$m=19456,t=2,p=1$" + enc.EncodeToString(make([]byte, 17)) + "$" + key,
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

func TestAdminPasswordLengthBounds(t *testing.T) {
	calls := countingKDF(t)
	for _, pw := range []string{"", strings.Repeat("a", 15), strings.Repeat("é", 15), strings.Repeat("a", 257)} {
		if _, err := HashAdminPassword(pw); err != ErrPasswordLength {
			t.Errorf("%d runes: err = %v, want ErrPasswordLength", len([]rune(pw)), err)
		}
	}
	if *calls != 0 {
		t.Fatalf("kdf ran %d times for out-of-range passwords", *calls)
	}
	for _, pw := range []string{strings.Repeat("a", 16), strings.Repeat("é", 16), strings.Repeat("a", 256)} {
		if _, err := HashAdminPassword(pw); err != nil {
			t.Errorf("%d runes: unexpected error %v", len([]rune(pw)), err)
		}
	}
	if *calls != 3 {
		t.Fatalf("kdf ran %d times, want 3", *calls)
	}
}

func TestErrorsCarryNoSecret(t *testing.T) {
	_, err := HashPassword("tiny-secret")
	if err == nil || strings.Contains(err.Error(), "tiny-secret") {
		t.Fatalf("error %v must not echo the password", err)
	}
}

func TestKDFConcurrencyIsBounded(t *testing.T) {
	var inFlight, peak atomic.Int64
	orig := idKey
	idKey = func(_, _ []byte, _, _ uint32, _ uint8, keyLen uint32) []byte {
		n := inFlight.Add(1)
		for {
			p := peak.Load()
			if n <= p || peak.CompareAndSwap(p, n) {
				break
			}
		}
		time.Sleep(5 * time.Millisecond)
		inFlight.Add(-1)
		return make([]byte, keyLen)
	}
	t.Cleanup(func() { idKey = orig })

	h, err := HashPassword(goodPassword)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				if _, err := HashPassword(goodPassword); err != nil {
					t.Error(err)
				}
			} else {
				VerifyPassword(h, goodPassword)
			}
		}(i)
	}
	wg.Wait()
	if got := peak.Load(); got > constants.MaxConcurrentHashes {
		t.Fatalf("%d KDF runs in flight at once, bound is %d", got, constants.MaxConcurrentHashes)
	}
	if got := peak.Load(); got < constants.MaxConcurrentHashes {
		t.Fatalf("peak %d: the semaphore admits fewer than %d concurrent runs", got, constants.MaxConcurrentHashes)
	}
}

func TestRejectedInputTakesNoKDFSlot(t *testing.T) {
	for i := 0; i < constants.MaxConcurrentHashes; i++ {
		kdfSlots <- struct{}{}
	}
	t.Cleanup(func() {
		for i := 0; i < constants.MaxConcurrentHashes; i++ {
			<-kdfSlots
		}
	})
	done := make(chan struct{})
	go func() {
		defer close(done)
		if _, err := HashPassword("short"); err != ErrPasswordLength {
			t.Errorf("err = %v", err)
		}
		if VerifyPassword("not a hash", goodPassword) {
			t.Error("malformed hash verified")
		}
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("rejected input blocked on the KDF semaphore")
	}
}
