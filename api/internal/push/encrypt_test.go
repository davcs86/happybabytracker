package push

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"testing"
)

func mustDec(t *testing.T, s string) []byte {
	t.Helper()
	b, err := b64.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// RFC 8291 Appendix A.
const (
	vecASPriv = "yfWPiYE-n46HLnH0KqZOF1fJJU3MYrct3AELtAQ-oRw"
	vecUAPub  = "BCVxsr7N_eNgVRqvHtD0zTZsEc6-VV-JvLexhqUzORcxaOzi6-AYWXvTBHm4bjyPjs7Vd8pZGH6SRpkNtoIAiw4"
	vecAuth   = "BTBZMqHH6r4Tts7J_aSIgg"
	vecSalt   = "DGv6ra1nlYgDCS1FRnbzlw"
	vecBody   = "DGv6ra1nlYgDCS1FRnbzlwAAEABBBP4z9KsN6nGRTbVYI_c7VJSPQTBtkgcy27mlmlMoZIIgDll6e3vCYLocInmYWAmS6TlzAC8wEqKK6PBru3jl7A_yl95bQpu6cVPTpK4Mqgkf1CXztLVBSt2Ks3oZwbuwXPXLWyouBWLVWGNWQexSgSxsj_Qulcy4a-fN"
)

func TestRFC8291Vector(t *testing.T) {
	asPriv, err := ecdh.P256().NewPrivateKey(mustDec(t, vecASPriv))
	if err != nil {
		t.Fatal(err)
	}
	got, err := encrypt([]byte("When I grow up, I want to be a watermelon"), mustDec(t, vecUAPub),
		mustDec(t, vecAuth), asPriv, mustDec(t, vecSalt))
	if err != nil {
		t.Fatal(err)
	}
	if b64.EncodeToString(got) != vecBody {
		t.Fatalf("body mismatch\n got %s\nwant %s", b64.EncodeToString(got), vecBody)
	}
}

// decrypt is an independent receiver for a freshly generated subscription key pair.
func decrypt(t *testing.T, body []byte, uaPriv *ecdh.PrivateKey, auth []byte) []byte {
	t.Helper()
	salt, rs, idlen := body[:16], binary.BigEndian.Uint32(body[16:20]), int(body[20])
	asPub, ct := body[21:21+idlen], body[21+idlen:]
	if rs < 18 || idlen != 65 || len(ct) > int(rs) {
		t.Fatalf("bad header rs=%d idlen=%d ct=%d", rs, idlen, len(ct))
	}
	k, err := ecdh.P256().NewPublicKey(asPub)
	if err != nil {
		t.Fatal(err)
	}
	secret, _ := uaPriv.ECDH(k)
	info := append(append([]byte("WebPush: info\x00"), uaPriv.PublicKey().Bytes()...), asPub...)
	ikm, _ := hkdf.Key(sha256.New, secret, auth, string(info), 32)
	cek, _ := hkdf.Key(sha256.New, ikm, salt, "Content-Encoding: aes128gcm\x00", 16)
	nonce, _ := hkdf.Key(sha256.New, ikm, salt, "Content-Encoding: nonce\x00", 12)
	blk, _ := aes.NewCipher(cek)
	g, _ := cipher.NewGCM(blk)
	pt, err := g.Open(nil, nonce, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	pt = bytes.TrimRight(pt, "\x00")
	if pt[len(pt)-1] != 0x02 {
		t.Fatal("missing final-record delimiter")
	}
	return pt[:len(pt)-1]
}

type fixture struct {
	ua, as     *ecdh.PrivateKey
	auth, salt []byte
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	ua, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	as, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	f := fixture{ua: ua, as: as, auth: make([]byte, 16), salt: make([]byte, 16)}
	rand.Read(f.auth)
	rand.Read(f.salt)
	return f
}

func TestRoundTripAndLimits(t *testing.T) {
	f := newFixture(t)
	msg := []byte(`{"v":1,"category":"medication","kid":"k1","title":"Medication due for Mia"}`)
	body, err := encrypt(msg, f.ua.PublicKey().Bytes(), f.auth, f.as, f.salt)
	if err != nil {
		t.Fatal(err)
	}
	if got := decrypt(t, body, f.ua, f.auth); !bytes.Equal(got, msg) {
		t.Fatalf("got %q", got)
	}
	if MaxPlaintext != 3993 {
		t.Fatalf("MaxPlaintext = %d, want 3993", MaxPlaintext)
	}
	b, err := encrypt(make([]byte, 3993), f.ua.PublicKey().Bytes(), f.auth, f.as, f.salt)
	if err != nil || len(b) != 4096 {
		t.Fatalf("3993-byte plaintext: len %d, %v; want a 4096-byte body", len(b), err)
	}
	if got := decrypt(t, b, f.ua, f.auth); len(got) != 3993 {
		t.Fatalf("max plaintext decrypts to %d bytes", len(got))
	}
	if _, err := encrypt(make([]byte, 3994), f.ua.PublicKey().Bytes(), f.auth, f.as, f.salt); err == nil {
		t.Fatal("3994-byte plaintext accepted")
	}
}

func TestEncryptRefusesBadKeys(t *testing.T) {
	f := newFixture(t)
	pub := f.ua.PublicKey().Bytes()
	invalid := append([]byte{0x04}, bytes.Repeat([]byte{0x01}, 64)...) // not on the curve
	msg := []byte("x")
	cases := map[string]func() error{
		"compressed p256dh": func() error { _, err := encrypt(msg, pub[:33], f.auth, f.as, f.salt); return err },
		"invalid p256dh":    func() error { _, err := encrypt(msg, invalid, f.auth, f.as, f.salt); return err },
		"auth 15 bytes":     func() error { _, err := encrypt(msg, pub, f.auth[:15], f.as, f.salt); return err },
		"auth 17 bytes":     func() error { _, err := encrypt(msg, pub, append(f.auth, 0), f.as, f.salt); return err },
		"salt 15 bytes":     func() error { _, err := encrypt(msg, pub, f.auth, f.as, f.salt[:15]); return err },
	}
	for name, fn := range cases {
		if fn() == nil {
			t.Errorf("%s accepted", name)
		}
	}
}

func TestParseKeys(t *testing.T) {
	f := newFixture(t)
	pub := f.ua.PublicKey().Bytes()
	invalid := append([]byte{0x04}, bytes.Repeat([]byte{0x01}, 64)...)
	enc := func(e *base64.Encoding, b []byte) string { return e.EncodeToString(b) }

	for name, in := range map[string][2]string{
		"unpadded url":      {enc(base64.RawURLEncoding, pub), enc(base64.RawURLEncoding, f.auth)},
		"padded url":        {enc(base64.URLEncoding, pub), enc(base64.URLEncoding, f.auth)},
		"padded standard":   {enc(base64.StdEncoding, pub), enc(base64.StdEncoding, f.auth)},
		"unpadded standard": {enc(base64.RawStdEncoding, pub), enc(base64.RawStdEncoding, f.auth)},
	} {
		gotPub, gotAuth, err := ParseKeys(in[0], in[1])
		if err != nil || !bytes.Equal(gotPub, pub) || !bytes.Equal(gotAuth, f.auth) {
			t.Errorf("%s: %v", name, err)
		}
	}

	okPub, okAuth := enc(base64.RawURLEncoding, pub), enc(base64.RawURLEncoding, f.auth)
	for name, in := range map[string][2]string{
		"compressed p256dh": {enc(base64.RawURLEncoding, pub[:33]), okAuth},
		"invalid p256dh":    {enc(base64.RawURLEncoding, invalid), okAuth},
		"p256dh not base64": {"!!" + okPub, okAuth},
		"empty p256dh":      {"", okAuth},
		"auth 15 bytes":     {okPub, enc(base64.RawURLEncoding, f.auth[:15])},
		"auth 17 bytes":     {okPub, enc(base64.RawURLEncoding, append(f.auth, 0))},
		"auth not base64":   {okPub, "*" + okAuth},
	} {
		_, _, err := ParseKeys(in[0], in[1])
		if err == nil {
			t.Errorf("%s accepted", name)
			continue
		}
		for _, v := range in {
			if v != "" && bytes.Contains([]byte(err.Error()), []byte(v)) {
				t.Errorf("%s: error %q echoes a key", name, err)
			}
		}
	}
}

func TestEncryptRefusesNilEphemeralKey(t *testing.T) {
	f := newFixture(t)
	if _, err := encrypt([]byte("x"), f.ua.PublicKey().Bytes(), f.auth, nil, f.salt); err == nil {
		t.Fatal("nil ephemeral key accepted")
	}
}

// TestRFC8291VectorDecrypts opens vecBody with the RFC's receiver key, written out here, not through decrypt.
func TestRFC8291VectorDecrypts(t *testing.T) {
	ua, err := ecdh.P256().NewPrivateKey(mustDec(t, "q1dXpw3UpT5VOmu_cf_v6ih07Aems3njxI-JWgLcM94"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(ua.PublicKey().Bytes(), mustDec(t, vecUAPub)) {
		t.Fatal("ua_private does not match ua_public")
	}
	auth := mustDec(t, "BTBZMqHH6r4Tts7J_aSIgg")
	body := mustDec(t, vecBody)
	salt, keyid, ct := body[0:16], body[21:86], body[86:]
	if binary.BigEndian.Uint32(body[16:20]) != 4096 || body[20] != 65 {
		t.Fatalf("header rs/idlen %x", body[16:21])
	}
	asPub, err := ecdh.P256().NewPublicKey(keyid)
	if err != nil {
		t.Fatal(err)
	}
	shared, err := ua.ECDH(asPub)
	if err != nil {
		t.Fatal(err)
	}
	info := "WebPush: info\x00" + string(ua.PublicKey().Bytes()) + string(keyid)
	ikm, err := hkdf.Key(sha256.New, shared, auth, info, 32)
	if err != nil {
		t.Fatal(err)
	}
	cek, _ := hkdf.Key(sha256.New, ikm, salt, "Content-Encoding: aes128gcm\x00", 16)
	nonce, _ := hkdf.Key(sha256.New, ikm, salt, "Content-Encoding: nonce\x00", 12)
	blk, err := aes.NewCipher(cek)
	if err != nil {
		t.Fatal(err)
	}
	g, err := cipher.NewGCM(blk)
	if err != nil {
		t.Fatal(err)
	}
	pt, err := g.Open(nil, nonce, ct, nil)
	if err != nil {
		t.Fatalf("vector does not open: %v", err)
	}
	if string(pt) != "When I grow up, I want to be a watermelon\x02" {
		t.Fatalf("plaintext %q", pt)
	}
}
