// Package push encrypts Web Push payloads (RFC 8291, aes128gcm) and signs VAPID JWTs (RFC 8292), stdlib only.
package push

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/hkdf"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"strings"
)

var b64 = base64.RawURLEncoding

// Protocol constants of RFC 8291/8188, not tunable limits: one 4096-byte record, an 86-byte header.
const (
	recordSize   = 4096
	MaxPlaintext = recordSize - 16 - 1 - 86
)

var (
	errTooLarge = errors.New("push payload too large")
	errLengths  = errors.New("push auth secret or salt has the wrong length")
	errP256dh   = errors.New("push p256dh is not an uncompressed P-256 point")
	errAuth     = errors.New("push auth is not 16 bytes")
)

// encrypt builds the aes128gcm body; asPriv and salt are injected only for the RFC vector: senders use encryptFresh.
func encrypt(plaintext, uaPub, auth []byte, asPriv *ecdh.PrivateKey, salt []byte) ([]byte, error) {
	if len(plaintext) > MaxPlaintext {
		return nil, errTooLarge
	}
	if len(auth) != 16 || len(salt) != 16 {
		return nil, errLengths
	}
	if asPriv == nil {
		return nil, errNoEphemeral
	}
	uaKey, err := ecdh.P256().NewPublicKey(uaPub)
	if err != nil {
		return nil, errP256dh
	}
	secret, err := asPriv.ECDH(uaKey)
	if err != nil {
		return nil, err
	}
	asPub := asPriv.PublicKey().Bytes()
	keyInfo := append(append([]byte("WebPush: info\x00"), uaPub...), asPub...)
	ikm, err := hkdf.Key(sha256.New, secret, auth, string(keyInfo), 32)
	if err != nil {
		return nil, err
	}
	cek, err := hkdf.Key(sha256.New, ikm, salt, "Content-Encoding: aes128gcm\x00", 16)
	if err != nil {
		return nil, err
	}
	nonce, err := hkdf.Key(sha256.New, ikm, salt, "Content-Encoding: nonce\x00", 12)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(cek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	out.Write(salt)
	_ = binary.Write(&out, binary.BigEndian, uint32(recordSize))
	out.WriteByte(byte(len(asPub)))
	out.Write(asPub)
	// One final record: plaintext || 0x02, no padding.
	return gcm.Seal(out.Bytes(), nonce, append(append([]byte{}, plaintext...), 0x02), nil), nil
}

// ParseKeys decodes a subscription's keys in either base64 alphabet, padded or not; errors never echo a key.
func ParseKeys(p256dh, auth string) (uaPub, authSecret []byte, err error) {
	uaPub, err = decodeTolerant(p256dh)
	if err != nil {
		return nil, nil, errP256dh
	}
	if _, err := ecdh.P256().NewPublicKey(uaPub); err != nil {
		return nil, nil, errP256dh
	}
	authSecret, err = decodeTolerant(auth)
	if err != nil || len(authSecret) != 16 {
		return nil, nil, errAuth
	}
	return uaPub, authSecret, nil
}

func decodeTolerant(s string) ([]byte, error) {
	s = strings.TrimRight(s, "=")
	s = strings.NewReplacer("+", "-", "/", "_").Replace(s)
	return b64.DecodeString(s)
}
