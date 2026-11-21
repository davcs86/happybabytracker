package auth

import (
	"bytes"
	"crypto/sha256"
	"testing"
)

func TestTokenIsUnique(t *testing.T) {
	seen := make(map[string]bool, 1000)
	for i := 0; i < 1000; i++ {
		plain, hash, err := NewToken()
		if err != nil {
			t.Fatal(err)
		}
		if len(plain) != 43 {
			t.Fatalf("token length %d, want 43", len(plain))
		}
		if len(hash) != sha256.Size {
			t.Fatalf("hash length %d", len(hash))
		}
		if seen[plain] {
			t.Fatal("duplicate token")
		}
		seen[plain] = true
	}
}

func TestTokenAlphabetIsURLSafe(t *testing.T) {
	plain, _, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range plain {
		ok := c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_'
		if !ok {
			t.Fatalf("token %q has non-URL-safe rune %q", plain, c)
		}
	}
}

func TestHashTokenStable(t *testing.T) {
	plain, hash, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(HashToken(plain), hash) {
		t.Fatal("HashToken(plain) differs from the hash NewToken returned")
	}
	if !bytes.Equal(HashToken(plain), HashToken(plain)) {
		t.Fatal("HashToken is not deterministic")
	}
	if bytes.Equal(HashToken(plain), HashToken(plain+"x")) {
		t.Fatal("different tokens hashed alike")
	}
	want := sha256.Sum256([]byte("abc"))
	if !bytes.Equal(HashToken("abc"), want[:]) {
		t.Fatal("HashToken is not SHA-256 of the plain string")
	}
}
