package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"

	"github.com/davcs86/happybabytracker/api/internal/config"
)

// AdminFingerprint is hex HMAC-SHA256(key=internalToken, username + "\x00" + password).
func AdminFingerprint(internalToken, username, password string) string {
	mac := hmac.New(sha256.New, []byte(internalToken))
	mac.Write([]byte(username))
	mac.Write([]byte{0})
	mac.Write([]byte(password))
	return hex.EncodeToString(mac.Sum(nil))
}

// CheckAdmin evaluates both comparisons before combining them; the caller normalises the username.
func CheckAdmin(cfg config.Config, username, password string) bool {
	gotUser, gotPass := sha256.Sum256([]byte(username)), sha256.Sum256([]byte(password))
	wantUser, wantPass := sha256.Sum256([]byte(cfg.AdminUsername)), sha256.Sum256([]byte(cfg.AdminPassword))
	userOK := subtle.ConstantTimeCompare(gotUser[:], wantUser[:])
	passOK := subtle.ConstantTimeCompare(gotPass[:], wantPass[:])
	return userOK&passOK == 1
}
