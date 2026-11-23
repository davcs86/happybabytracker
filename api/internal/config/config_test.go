package config

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"strings"
	"testing"
)

const (
	okPassword = "correct-horse-16" // 16 runes
	okToken    = "0123456789abcdef0123456789abcdef"
)

func env(overrides map[string]string) func(string) string {
	m := map[string]string{
		"ADMIN_USERNAME":     "admin",
		"ADMIN_PASSWORD":     okPassword,
		"INTERNAL_API_TOKEN": okToken,
		"PUBLIC_ORIGIN":      "https://hbt.example.org",
		"DATA_DIR":           "/var/lib/hbt",
	}
	for k, v := range overrides {
		if v == "\x00unset" {
			delete(m, k)
		} else {
			m[k] = v
		}
	}
	return func(k string) string { return m[k] }
}

func TestLoadValid(t *testing.T) {
	cfg, err := Load(env(nil))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := Config{
		AdminUsername:    "admin",
		AdminPassword:    okPassword,
		InternalAPIToken: okToken,
		PublicOrigin:     "https://hbt.example.org",
		DataDir:          "/var/lib/hbt",
		Addr:             ":9210",
	}
	if cfg != want {
		t.Fatalf("got %+v, want %+v", cfg, want)
	}
}

func TestLoadDataDirDefault(t *testing.T) {
	cfg, err := Load(env(map[string]string{"DATA_DIR": "\x00unset"}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DataDir != "/data" {
		t.Fatalf("DataDir = %q, want /data", cfg.DataDir)
	}
}

func TestLoadAcceptsBoundaries(t *testing.T) {
	cases := map[string]map[string]string{
		"16 multibyte runes password": {"ADMIN_PASSWORD": strings.Repeat("é", 16)},
		"32 char token":               {"INTERNAL_API_TOKEN": strings.Repeat("t", 32)},
		"http origin with port":       {"PUBLIC_ORIGIN": "http://localhost:9212"},
		"3 char username":             {"ADMIN_USERNAME": "a.b"},
		"32 char username":            {"ADMIN_USERNAME": strings.Repeat("u", 32)},
	}
	for name, o := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Load(env(o)); err != nil {
				t.Fatalf("Load: %v", err)
			}
		})
	}
}

func TestLoadRefuses(t *testing.T) {
	cases := []struct {
		name     string
		variable string
		val      string
	}{
		{"username empty", "ADMIN_USERNAME", ""},
		{"username unset", "ADMIN_USERNAME", "\x00unset"},
		{"username upper case", "ADMIN_USERNAME", "Qzadmin"},
		{"username too short", "ADMIN_USERNAME", "qz"},
		{"username too long", "ADMIN_USERNAME", strings.Repeat("q", 33)},
		{"username bad char", "ADMIN_USERNAME", "qz admin"},
		{"password empty", "ADMIN_PASSWORD", ""},
		{"password 15 chars", "ADMIN_PASSWORD", "wxyzwxyzwxyzwxy"},
		{"password 15 multibyte runes", "ADMIN_PASSWORD", strings.Repeat("é", 15)},
		{"token empty", "INTERNAL_API_TOKEN", ""},
		{"token 31 chars", "INTERNAL_API_TOKEN", strings.Repeat("k", 31)},
		{"origin empty", "PUBLIC_ORIGIN", ""},
		{"origin no scheme", "PUBLIC_ORIGIN", "hbt.example.org"},
		{"origin bad scheme", "PUBLIC_ORIGIN", "ftp://hbt.example.org"},
		{"origin no host", "PUBLIC_ORIGIN", "https://"},
		{"origin with path", "PUBLIC_ORIGIN", "https://hbt.example.org/app"},
		{"origin trailing slash", "PUBLIC_ORIGIN", "https://hbt.example.org/"},
		{"origin with query", "PUBLIC_ORIGIN", "https://hbt.example.org?zq=1"},
		{"origin with userinfo", "PUBLIC_ORIGIN", "https://zquser:zqpass@hbt.example.org"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Load(env(map[string]string{tc.variable: tc.val}))
			if err == nil {
				t.Fatal("Load accepted an invalid value")
			}
			if !strings.Contains(err.Error(), tc.variable) {
				t.Errorf("error %q does not name %s", err, tc.variable)
			}
			if tc.val != "" && tc.val != "\x00unset" && strings.Contains(err.Error(), tc.val) {
				t.Errorf("error %q contains the value", err)
			}
		})
	}
}

// testVAPIDKey is generated per test run, never the edge's key.
func testVAPIDKey(t *testing.T) string {
	t.Helper()
	k, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(k.Bytes())
}

const okSubject = "https://happybaby.private.example.org"

func TestLoadVAPIDOff(t *testing.T) {
	for name, o := range map[string]map[string]string{
		"unset": nil,
		"empty": {"VAPID_PRIVATE_KEY": "", "VAPID_SUBJECT": "", "HBT_ALLOW_FAKE_PUSH": ""},
	} {
		cfg, err := Load(env(o))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if cfg.VAPIDKey != nil || cfg.VAPIDSubject != "" || cfg.AllowFakePush {
			t.Fatalf("%s: push config %+v, want off", name, cfg)
		}
	}
}

func TestLoadVAPIDOn(t *testing.T) {
	key := testVAPIDKey(t)
	for _, fake := range []string{"", "1"} {
		cfg, err := Load(env(map[string]string{"VAPID_PRIVATE_KEY": key, "VAPID_SUBJECT": okSubject, "HBT_ALLOW_FAKE_PUSH": fake}))
		if err != nil {
			t.Fatalf("fake %q: %v", fake, err)
		}
		if cfg.VAPIDKey == nil {
			t.Fatalf("fake %q: key not loaded", fake)
		}
		if b, err := cfg.VAPIDKey.Bytes(); err != nil || base64.RawURLEncoding.EncodeToString(b) != key {
			t.Fatalf("fake %q: key differs: %v", fake, err)
		}
		if cfg.VAPIDSubject != okSubject || cfg.AllowFakePush != (fake == "1") {
			t.Fatalf("fake %q: got subject %q fake %v", fake, cfg.VAPIDSubject, cfg.AllowFakePush)
		}
	}
}

func TestLoadVAPIDRefuses(t *testing.T) {
	key := testVAPIDKey(t)
	cases := []struct {
		name, variable string
		env            map[string]string
	}{
		{"only the key", "VAPID_SUBJECT", map[string]string{"VAPID_PRIVATE_KEY": key}},
		{"only the subject", "VAPID_PRIVATE_KEY", map[string]string{"VAPID_SUBJECT": okSubject}},
		{"malformed key", "VAPID_PRIVATE_KEY", map[string]string{"VAPID_PRIVATE_KEY": key[:42], "VAPID_SUBJECT": okSubject}},
		{"padded key", "VAPID_PRIVATE_KEY", map[string]string{"VAPID_PRIVATE_KEY": key + "=", "VAPID_SUBJECT": okSubject}},
		{"zero key", "VAPID_PRIVATE_KEY", map[string]string{"VAPID_PRIVATE_KEY": strings.Repeat("A", 43), "VAPID_SUBJECT": okSubject}},
		{"http subject", "VAPID_SUBJECT", map[string]string{"VAPID_PRIVATE_KEY": key, "VAPID_SUBJECT": "http://happybaby.example.org"}},
		{"mailto subject", "VAPID_SUBJECT", map[string]string{"VAPID_PRIVATE_KEY": key, "VAPID_SUBJECT": "mailto:zq@example.org"}},
		{"subject userinfo", "VAPID_SUBJECT", map[string]string{"VAPID_PRIVATE_KEY": key, "VAPID_SUBJECT": "https://zquser@happybaby.example.org"}},
		{"subject query", "VAPID_SUBJECT", map[string]string{"VAPID_PRIVATE_KEY": key, "VAPID_SUBJECT": "https://happybaby.example.org?zq=1"}},
		{"subject fragment", "VAPID_SUBJECT", map[string]string{"VAPID_PRIVATE_KEY": key, "VAPID_SUBJECT": "https://happybaby.example.org#zq"}},
		{"subject no host", "VAPID_SUBJECT", map[string]string{"VAPID_PRIVATE_KEY": key, "VAPID_SUBJECT": "https://"}},
		{"fake without VAPID", "HBT_ALLOW_FAKE_PUSH", map[string]string{"HBT_ALLOW_FAKE_PUSH": "1"}},
		{"fake with only the key", "VAPID_SUBJECT", map[string]string{"HBT_ALLOW_FAKE_PUSH": "1", "VAPID_PRIVATE_KEY": key}},
		{"fake true", "HBT_ALLOW_FAKE_PUSH", map[string]string{"HBT_ALLOW_FAKE_PUSH": "true", "VAPID_PRIVATE_KEY": key, "VAPID_SUBJECT": okSubject}},
		{"fake 0", "HBT_ALLOW_FAKE_PUSH", map[string]string{"HBT_ALLOW_FAKE_PUSH": "0"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Load(env(tc.env))
			if err == nil {
				t.Fatal("Load accepted an invalid push configuration")
			}
			if !strings.Contains(err.Error(), tc.variable) {
				t.Errorf("error %q does not name %s", err, tc.variable)
			}
			for k, v := range tc.env {
				if k != "HBT_ALLOW_FAKE_PUSH" && strings.Contains(err.Error(), v) {
					t.Errorf("error %q contains the value of %s", err, k)
				}
			}
			if strings.Contains(err.Error(), "true") || strings.Contains(err.Error(), key[:8]) {
				t.Errorf("error %q contains a value", err)
			}
		})
	}
	_, err := Load(env(map[string]string{"VAPID_PRIVATE_KEY": key}))
	if want := "VAPID_SUBJECT is required when VAPID_PRIVATE_KEY is set"; err == nil || err.Error() != want {
		t.Fatalf("only the key: %v, want %q", err, want)
	}
}
