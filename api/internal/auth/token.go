package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

func NewToken() (plain string, hash []byte, err error) {
	b := make([]byte, constants.TokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", nil, fmt.Errorf("read token bytes: %w", err)
	}
	plain = base64.RawURLEncoding.EncodeToString(b)
	return plain, HashToken(plain), nil
}

func HashToken(plain string) []byte {
	sum := sha256.Sum256([]byte(plain))
	return sum[:]
}
