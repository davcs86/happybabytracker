package push

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

const testSubject = "https://happybaby.private.example.org"

// newKey returns a fresh VAPID private key in the edge's encoding and its public key.
func newKey(t *testing.T) (priv, pub string) {
	t.Helper()
	k, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return b64.EncodeToString(k.Bytes()), b64.EncodeToString(k.PublicKey().Bytes())
}

func TestParsePrivateKeyAndPublicKey(t *testing.T) {
	// The RFC 8291 vector's sender key pair: the public key is bytes 21..86 of the body.
	k, err := ParsePrivateKey(vecASPriv)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := PublicKey(k), b64.EncodeToString(mustDec(t, vecBody)[21:86]); got != want {
		t.Fatalf("PublicKey = %s, want %s", got, want)
	}
	priv, pub := newKey(t)
	if len(priv) != 43 {
		t.Fatalf("generated key is %d characters", len(priv))
	}
	k, err = ParsePrivateKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	if got := PublicKey(k); got != pub || len(mustDec(t, got)) != 65 {
		t.Fatalf("PublicKey = %s, want %s", got, pub)
	}
}

func TestParsePrivateKeyRefuses(t *testing.T) {
	raw := mustDec(t, vecASPriv)
	n := elliptic.P256().Params().N
	big32 := func(x *big.Int) []byte { return x.FillBytes(make([]byte, 32)) }
	std := base64.RawStdEncoding.EncodeToString(raw)
	if !strings.ContainsAny(std, "+/") {
		t.Fatal("fixture: the vector key has no standard-alphabet character")
	}
	cases := map[string]string{
		"padded":            base64.URLEncoding.EncodeToString(raw),
		"standard alphabet": std,
		"42 characters":     vecASPriv[:42],
		"44 characters":     vecASPriv + "A",
		"zero":              b64.EncodeToString(make([]byte, 32)),
		"n":                 b64.EncodeToString(big32(n)),
		"n+1":               b64.EncodeToString(big32(new(big.Int).Add(n, big.NewInt(1)))),
		"all ones":          b64.EncodeToString([]byte(strings.Repeat("\xff", 32))),
		"empty":             "",
		"non-canonical":     vecASPriv[:42] + "x", // trailing bits set
	}
	for name, v := range cases {
		_, err := ParsePrivateKey(v)
		if err == nil {
			t.Errorf("%s accepted", name)
			continue
		}
		if v != "" && strings.Contains(err.Error(), v) {
			t.Errorf("%s: error %q echoes the key", name, err)
		}
	}
}

func TestNewSignerRefuses(t *testing.T) {
	priv, _ := newKey(t)
	k, err := ParsePrivateKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewSigner(nil, testSubject); err == nil {
		t.Error("nil key accepted")
	}
	for _, s := range []string{
		"", "http://happybaby.example.org", "mailto:ops@example.org", "https://", "https:opaque",
		"https://user@happybaby.example.org", "https://user:pw@happybaby.example.org",
		"https://happybaby.example.org?q=1", "https://happybaby.example.org?",
		"https://happybaby.example.org#frag", "happybaby.example.org",
	} {
		_, err := NewSigner(k, s)
		if err == nil {
			t.Errorf("subject %q accepted", s)
			continue
		}
		if s != "" && strings.Contains(err.Error(), s) {
			t.Errorf("subject %q echoed in %q", s, err)
		}
	}
	for _, s := range []string{testSubject, "https://happybaby.example.org:8443", "https://happybaby.example.org/path"} {
		if _, err := NewSigner(k, s); err != nil {
			t.Errorf("subject %q refused: %v", s, err)
		}
	}
}

func TestAuthorization(t *testing.T) {
	if constants.PushJWTLifetime > 24*time.Hour || constants.PushJWTLifetime <= 0 {
		t.Fatalf("PushJWTLifetime %v outside (0, 24h]", constants.PushJWTLifetime)
	}
	priv, pub := newKey(t)
	k, err := ParsePrivateKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewSigner(k, testSubject)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_800_000_000, 0)
	for endpoint, aud := range map[string]string{
		"https://fcm.googleapis.com/fcm/send/abc":        "https://fcm.googleapis.com",
		"https://push.example.org:8443/wpush/v2/xyz?q=1": "https://push.example.org:8443",
	} {
		h, err := s.Authorization(endpoint, now)
		if err != nil {
			t.Fatal(err)
		}
		rest, ok := strings.CutPrefix(h, "vapid t=")
		jwt, kv, ok2 := strings.Cut(rest, ", k=")
		if !ok || !ok2 || kv != pub {
			t.Fatalf("header shape %q", h)
		}
		parts := strings.Split(jwt, ".")
		if len(parts) != 3 {
			t.Fatalf("jwt has %d parts", len(parts))
		}
		var hdr map[string]any
		if err := json.Unmarshal(mustDec(t, parts[0]), &hdr); err != nil || hdr["alg"] != "ES256" || hdr["typ"] != "JWT" {
			t.Fatalf("jwt header %v %v", hdr, err)
		}
		sig := mustDec(t, parts[2])
		if len(sig) != 64 {
			t.Fatalf("sig len %d (DER?)", len(sig))
		}
		var claims map[string]any
		if err := json.Unmarshal(mustDec(t, parts[1]), &claims); err != nil {
			t.Fatal(err)
		}
		exp := int64(claims["exp"].(float64))
		if claims["aud"] != aud || claims["sub"] != testSubject || exp != now.Add(constants.PushJWTLifetime).Unix() || len(claims) != 3 {
			t.Fatalf("claims %v", claims)
		}
		ecpub, err := ecdsa.ParseUncompressedPublicKey(elliptic.P256(), mustDec(t, pub))
		if err != nil {
			t.Fatal(err)
		}
		h2 := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
		if !ecdsa.Verify(ecpub, h2[:], new(big.Int).SetBytes(sig[:32]), new(big.Int).SetBytes(sig[32:])) {
			t.Fatal("signature does not verify")
		}
	}
	for _, bad := range []string{"http://push.example.org/x", "https:///x", "::", ""} {
		if _, err := s.Authorization(bad, now); err == nil {
			t.Errorf("endpoint %q accepted", bad)
		} else if bad != "" && strings.Contains(err.Error(), bad) {
			t.Errorf("endpoint %q echoed in %q", bad, err)
		}
	}
}
