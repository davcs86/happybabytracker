package push

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/json"
	"errors"
	"math/big"
	"net/url"
	"strings"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

// vapidKeyLen is the unpadded base64url length of a 32-byte P-256 scalar.
const vapidKeyLen = 43

var (
	errPrivateKey = errors.New("VAPID private key must be 43 unpadded base64url characters encoding a P-256 scalar")
	errSubject    = errors.New("VAPID subject must be an https: URL with a host and no userinfo, query or fragment")
	errNoKey      = errors.New("VAPID private key is missing")
	errEndpoint   = errors.New("push endpoint is not an https URL with a host")
	errSignature  = errors.New("VAPID signature is malformed")
)

// ParsePrivateKey reads the edge's encoding; errors never echo the key.
func ParsePrivateKey(s string) (*ecdsa.PrivateKey, error) {
	if len(s) != vapidKeyLen {
		return nil, errPrivateKey
	}
	d, err := b64.Strict().DecodeString(s)
	if err != nil || len(d) != 32 {
		return nil, errPrivateKey
	}
	k, err := ecdsa.ParseRawPrivateKey(elliptic.P256(), d)
	if err != nil {
		return nil, errPrivateKey
	}
	return k, nil
}

// PublicKey is the applicationServerKey and the k= value; empty for a key not on P-256.
func PublicKey(k *ecdsa.PrivateKey) string {
	ek, err := k.ECDH()
	if err != nil {
		return ""
	}
	return b64.EncodeToString(ek.PublicKey().Bytes())
}

type Signer struct {
	key     *ecdsa.PrivateKey
	pub     string
	subject string
}

func NewSigner(k *ecdsa.PrivateKey, subject string) (*Signer, error) {
	if k == nil {
		return nil, errNoKey
	}
	if !validSubject(subject) {
		return nil, errSubject
	}
	pub := PublicKey(k)
	if pub == "" {
		return nil, errPrivateKey
	}
	return &Signer{key: k, pub: pub, subject: subject}, nil
}

func validSubject(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	return u.Scheme == "https" && u.Opaque == "" && u.Hostname() != "" && u.User == nil &&
		u.RawQuery == "" && !u.ForceQuery && u.Fragment == "" && u.RawFragment == ""
}

// Authorization is the Authorization header value for one endpoint; errors never echo it.
func (s *Signer) Authorization(endpoint string, now time.Time) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" {
		return "", errEndpoint
	}
	claims, err := json.Marshal(map[string]any{
		"aud": origin(u),
		"exp": now.Add(constants.PushJWTLifetime).Unix(),
		"sub": s.subject,
	})
	if err != nil {
		return "", err
	}
	signing := b64.EncodeToString([]byte(`{"typ":"JWT","alg":"ES256"}`)) + "." + b64.EncodeToString(claims)
	h := sha256.Sum256([]byte(signing))
	der, err := ecdsa.SignASN1(rand.Reader, s.key, h[:])
	if err != nil {
		return "", err
	}
	sig, err := rawRS(der)
	if err != nil {
		return "", err
	}
	return "vapid t=" + signing + "." + b64.EncodeToString(sig) + ", k=" + s.pub, nil
}

// origin is the ASCII serialisation: lowercase host, the port only when it is not 443.
func origin(u *url.URL) string {
	host := strings.ToLower(u.Hostname())
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	if p := u.Port(); p != "" && p != "443" {
		host += ":" + p
	}
	return "https://" + host
}

// rawRS converts a DER ECDSA signature to the 64-byte r||s that JWS ES256 requires.
func rawRS(der []byte) ([]byte, error) {
	var sig struct{ R, S *big.Int }
	if rest, err := asn1.Unmarshal(der, &sig); err != nil || len(rest) != 0 {
		return nil, errSignature
	}
	out := make([]byte, 64)
	sig.R.FillBytes(out[:32])
	sig.S.FillBytes(out[32:])
	return out, nil
}
