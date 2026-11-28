package config

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/davcs86/happybabytracker/api/internal/ntfy"
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
		AuthMode:         AuthLocal,
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
		{"npi search 2", "NPI_SEARCH", "2"},
		{"npi search true", "NPI_SEARCH", "true"},
		{"npi search 0", "NPI_SEARCH", "0"},
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

func TestLoadIgnoresAdminVariables(t *testing.T) {
	for name, o := range map[string]map[string]string{
		"unset":      {"ADMIN_USERNAME": "\x00unset", "ADMIN_PASSWORD": "\x00unset"},
		"upper case": {"ADMIN_USERNAME": "David"},
	} {
		cfg, err := Load(env(o))
		if err != nil {
			t.Errorf("%s: Load: %v", name, err)
		}
		if name == "upper case" && cfg.AdminUsername != "David" {
			t.Errorf("AdminUsername = %q, want the raw value", cfg.AdminUsername)
		}
	}
}

func TestLoadNPISearch(t *testing.T) {
	for val, want := range map[string]bool{"\x00unset": false, "": false, "1": true} {
		cfg, err := Load(env(map[string]string{"NPI_SEARCH": val}))
		if err != nil || cfg.NPISearch != want {
			t.Errorf("NPI_SEARCH %q: %v %v, want %v", val, cfg.NPISearch, err, want)
		}
	}
}

func TestLoadAuthMode(t *testing.T) {
	for val, want := range map[string]AuthMode{"\x00unset": AuthLocal, "": AuthLocal, "local": AuthLocal, "oidc": AuthOIDC, "both": AuthBoth} {
		cfg, err := Load(env(map[string]string{"AUTH_MODE": val}))
		if err != nil || cfg.AuthMode != want {
			t.Errorf("AUTH_MODE %q: %q %v, want %q", val, cfg.AuthMode, err, want)
		}
	}
	for _, val := range []string{"Local", "OIDC", "none", "both "} {
		_, err := Load(env(map[string]string{"AUTH_MODE": val}))
		if err == nil || err.Error() != "AUTH_MODE must be local, oidc or both" {
			t.Errorf("AUTH_MODE %q: %v, want the AUTH_MODE refusal", val, err)
		}
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

func TestLoadNtfyOff(t *testing.T) {
	cfg, err := Load(env(nil))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.NtfyEnabled || cfg.Ntfy != (ntfy.Config{}) {
		t.Fatalf("ntfy %+v enabled=%v, want off", cfg.Ntfy, cfg.NtfyEnabled)
	}
}

func TestLoadNtfyOn(t *testing.T) {
	cfg, err := Load(env(map[string]string{
		"NTFY_URL": "http://ntfy:2586", "NTFY_TOPIC_PREFIX": "hbt", "NTFY_TOKEN": "tk_x", "NTFY_PUBLIC_URL": "https://ntfy.example.org",
	}))
	if err != nil {
		t.Fatal(err)
	}
	want := ntfy.Config{URL: "http://ntfy:2586", TopicPrefix: "hbt", Token: "tk_x", PublicURL: "https://ntfy.example.org"}
	if !cfg.NtfyEnabled || cfg.Ntfy != want {
		t.Fatalf("ntfy %+v enabled=%v, want %+v on", cfg.Ntfy, cfg.NtfyEnabled, want)
	}
}

func TestLoadNtfyIsIndependentOfWebPush(t *testing.T) {
	key := testVAPIDKey(t)
	cfg, err := Load(env(map[string]string{"VAPID_PRIVATE_KEY": key, "VAPID_SUBJECT": okSubject}))
	if err != nil || cfg.VAPIDKey == nil || cfg.NtfyEnabled {
		t.Fatalf("Web Push only: key=%v ntfy=%v err=%v", cfg.VAPIDKey != nil, cfg.NtfyEnabled, err)
	}
	cfg, err = Load(env(map[string]string{"NTFY_URL": "http://ntfy:2586", "NTFY_TOPIC_PREFIX": "hbt"}))
	if err != nil || cfg.VAPIDKey != nil || !cfg.NtfyEnabled {
		t.Fatalf("ntfy only: key=%v ntfy=%v err=%v", cfg.VAPIDKey != nil, cfg.NtfyEnabled, err)
	}
	cfg, err = Load(env(map[string]string{"VAPID_PRIVATE_KEY": key, "VAPID_SUBJECT": okSubject, "NTFY_URL": "http://ntfy:2586", "NTFY_TOPIC_PREFIX": "hbt"}))
	if err != nil || cfg.VAPIDKey == nil || !cfg.NtfyEnabled {
		t.Fatalf("both: key=%v ntfy=%v err=%v", cfg.VAPIDKey != nil, cfg.NtfyEnabled, err)
	}
}

func TestLoadNtfyRefuses(t *testing.T) {
	for name, tc := range map[string]struct {
		variable string
		env      map[string]string
	}{
		"token without url":  {"NTFY_TOKEN", map[string]string{"NTFY_TOKEN": "tk_zqsecret"}},
		"url without prefix": {"NTFY_TOPIC_PREFIX", map[string]string{"NTFY_URL": "http://ntfy:2586"}},
		"public url http":    {"NTFY_PUBLIC_URL", map[string]string{"NTFY_URL": "http://ntfy:2586", "NTFY_TOPIC_PREFIX": "hbt", "NTFY_PUBLIC_URL": "http://zq.example.org"}},
	} {
		_, err := Load(env(tc.env))
		if err == nil || !strings.Contains(err.Error(), tc.variable) || strings.Contains(err.Error(), "zq") {
			t.Errorf("%s: error %v, want one naming %s and no value", name, err, tc.variable)
		}
	}
}
