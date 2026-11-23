package ntfy

import (
	"strings"
	"testing"
)

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestLoadConfigOff(t *testing.T) {
	for name, m := range map[string]map[string]string{
		"unset": nil,
		"empty": {"NTFY_URL": "", "NTFY_TOKEN": "", "NTFY_TOPIC_PREFIX": "", "NTFY_PUBLIC_URL": ""},
	} {
		cfg, on, err := LoadConfig(env(m))
		if err != nil || on || cfg != (Config{}) {
			t.Fatalf("%s: got %+v on=%v err=%v, want zero, off, no error", name, cfg, on, err)
		}
	}
}

func TestLoadConfigOn(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want Config
	}{
		{"http url and prefix only", map[string]string{"NTFY_URL": "http://ntfy:2586", "NTFY_TOPIC_PREFIX": "hbt"},
			Config{URL: "http://ntfy:2586", TopicPrefix: "hbt"}},
		{"https url", map[string]string{"NTFY_URL": "https://ntfy.example.org", "NTFY_TOPIC_PREFIX": "hbt-1"},
			Config{URL: "https://ntfy.example.org", TopicPrefix: "hbt-1"}},
		{"trailing slash is dropped", map[string]string{"NTFY_URL": "http://ntfy:2586/", "NTFY_TOPIC_PREFIX": "hbt"},
			Config{URL: "http://ntfy:2586", TopicPrefix: "hbt"}},
		{"uppercase scheme and host", map[string]string{"NTFY_URL": "HTTP://Ntfy:2586", "NTFY_TOPIC_PREFIX": "hbt"},
			Config{URL: "http://Ntfy:2586", TopicPrefix: "hbt"}},
		{"32 character prefix", map[string]string{"NTFY_URL": "http://ntfy:2586", "NTFY_TOPIC_PREFIX": strings.Repeat("a", 32)},
			Config{URL: "http://ntfy:2586", TopicPrefix: strings.Repeat("a", 32)}},
		{"1 character prefix", map[string]string{"NTFY_URL": "http://ntfy:2586", "NTFY_TOPIC_PREFIX": "-"},
			Config{URL: "http://ntfy:2586", TopicPrefix: "-"}},
		{"token", map[string]string{"NTFY_URL": "http://ntfy:2586", "NTFY_TOPIC_PREFIX": "hbt", "NTFY_TOKEN": "tk_abcdefghijklmnopqrstuvwxyz012"},
			Config{URL: "http://ntfy:2586", TopicPrefix: "hbt", Token: "tk_abcdefghijklmnopqrstuvwxyz012"}},
		{"public url", map[string]string{"NTFY_URL": "http://ntfy:2586", "NTFY_TOPIC_PREFIX": "hbt", "NTFY_PUBLIC_URL": "https://ntfy.example.org/"},
			Config{URL: "http://ntfy:2586", TopicPrefix: "hbt", PublicURL: "https://ntfy.example.org"}},
		{"everything", map[string]string{"NTFY_URL": "https://ntfy.example.org:8443", "NTFY_TOPIC_PREFIX": "hbt", "NTFY_TOKEN": "tk_x", "NTFY_PUBLIC_URL": "https://ntfy.example.org:8443"},
			Config{URL: "https://ntfy.example.org:8443", TopicPrefix: "hbt", Token: "tk_x", PublicURL: "https://ntfy.example.org:8443"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, on, err := LoadConfig(env(tc.env))
			if err != nil || !on {
				t.Fatalf("on=%v err=%v", on, err)
			}
			if cfg != tc.want {
				t.Fatalf("got %+v, want %+v", cfg, tc.want)
			}
		})
	}
}

func TestLoadConfigRefuses(t *testing.T) {
	const url, prefix = "http://ntfy:2586", "hbt"
	cases := []struct {
		name, variable string
		env            map[string]string
	}{
		{"token without url", "NTFY_TOKEN", map[string]string{"NTFY_TOKEN": "tk_zqsecret"}},
		{"prefix without url", "NTFY_TOPIC_PREFIX", map[string]string{"NTFY_TOPIC_PREFIX": "zqprefix"}},
		{"public url without url", "NTFY_PUBLIC_URL", map[string]string{"NTFY_PUBLIC_URL": "https://zq.example.org"}},
		{"prefix with a valid token but no url", "NTFY_TOKEN", map[string]string{"NTFY_TOKEN": "tk_zqsecret", "NTFY_TOPIC_PREFIX": "hbt"}},
		{"url without prefix", "NTFY_TOPIC_PREFIX", map[string]string{"NTFY_URL": url}},
		{"url with empty prefix", "NTFY_TOPIC_PREFIX", map[string]string{"NTFY_URL": url, "NTFY_TOPIC_PREFIX": ""}},
		{"prefix of 33", "NTFY_TOPIC_PREFIX", map[string]string{"NTFY_URL": url, "NTFY_TOPIC_PREFIX": strings.Repeat("z", 33)}},
		{"prefix with underscore", "NTFY_TOPIC_PREFIX", map[string]string{"NTFY_URL": url, "NTFY_TOPIC_PREFIX": "zq_pre"}},
		{"prefix with upper case", "NTFY_TOPIC_PREFIX", map[string]string{"NTFY_URL": url, "NTFY_TOPIC_PREFIX": "ZqPre"}},
		{"prefix with newline", "NTFY_TOPIC_PREFIX", map[string]string{"NTFY_URL": url, "NTFY_TOPIC_PREFIX": "zq\n"}},
		{"url only whitespace", "NTFY_URL", map[string]string{"NTFY_URL": "   ", "NTFY_TOPIC_PREFIX": prefix}},
		{"url no scheme", "NTFY_URL", map[string]string{"NTFY_URL": "zqhost:2586", "NTFY_TOPIC_PREFIX": prefix}},
		{"url bad scheme", "NTFY_URL", map[string]string{"NTFY_URL": "ftp://zqhost", "NTFY_TOPIC_PREFIX": prefix}},
		{"url no host", "NTFY_URL", map[string]string{"NTFY_URL": "http://", "NTFY_TOPIC_PREFIX": prefix}},
		{"url with path", "NTFY_URL", map[string]string{"NTFY_URL": "http://zqhost/zqpath", "NTFY_TOPIC_PREFIX": prefix}},
		{"url with query", "NTFY_URL", map[string]string{"NTFY_URL": "http://zqhost?zq=1", "NTFY_TOPIC_PREFIX": prefix}},
		{"url with empty query", "NTFY_URL", map[string]string{"NTFY_URL": "http://zqhost?", "NTFY_TOPIC_PREFIX": prefix}},
		{"url with fragment", "NTFY_URL", map[string]string{"NTFY_URL": "http://zqhost#zq", "NTFY_TOPIC_PREFIX": prefix}},
		{"url with userinfo", "NTFY_URL", map[string]string{"NTFY_URL": "http://zquser:zqpass@zqhost", "NTFY_TOPIC_PREFIX": prefix}},
		{"url bad port", "NTFY_URL", map[string]string{"NTFY_URL": "http://zqhost:zq", "NTFY_TOPIC_PREFIX": prefix}},
		{"public url only whitespace", "NTFY_PUBLIC_URL", map[string]string{"NTFY_URL": url, "NTFY_TOPIC_PREFIX": prefix, "NTFY_PUBLIC_URL": " \t"}},
		{"public url http", "NTFY_PUBLIC_URL", map[string]string{"NTFY_URL": url, "NTFY_TOPIC_PREFIX": prefix, "NTFY_PUBLIC_URL": "http://zqhost.example.org"}},
		{"public url no scheme", "NTFY_PUBLIC_URL", map[string]string{"NTFY_URL": url, "NTFY_TOPIC_PREFIX": prefix, "NTFY_PUBLIC_URL": "zqhost.example.org"}},
		{"public url with path", "NTFY_PUBLIC_URL", map[string]string{"NTFY_URL": url, "NTFY_TOPIC_PREFIX": prefix, "NTFY_PUBLIC_URL": "https://zqhost.example.org/zqpath"}},
		{"public url with userinfo", "NTFY_PUBLIC_URL", map[string]string{"NTFY_URL": url, "NTFY_TOPIC_PREFIX": prefix, "NTFY_PUBLIC_URL": "https://zquser@zqhost.example.org"}},
		{"public url with query", "NTFY_PUBLIC_URL", map[string]string{"NTFY_URL": url, "NTFY_TOPIC_PREFIX": prefix, "NTFY_PUBLIC_URL": "https://zqhost.example.org?zq=1"}},
		{"token with a space", "NTFY_TOKEN", map[string]string{"NTFY_URL": url, "NTFY_TOPIC_PREFIX": prefix, "NTFY_TOKEN": "tk_zq secret"}},
		{"token with a newline", "NTFY_TOKEN", map[string]string{"NTFY_URL": url, "NTFY_TOPIC_PREFIX": prefix, "NTFY_TOKEN": "tk_zq\nsecret"}},
		{"token with a non-ASCII rune", "NTFY_TOKEN", map[string]string{"NTFY_URL": url, "NTFY_TOPIC_PREFIX": prefix, "NTFY_TOKEN": "tk_zqé"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, on, err := LoadConfig(env(tc.env))
			if err == nil {
				t.Fatal("LoadConfig accepted an invalid configuration")
			}
			if on || cfg != (Config{}) {
				t.Errorf("got %+v on=%v with an error, want zero and off", cfg, on)
			}
			if !strings.Contains(err.Error(), tc.variable) {
				t.Errorf("error %q does not name %s", err, tc.variable)
			}
			for k, v := range tc.env {
				if tv := strings.TrimSpace(v); tv != "" && strings.Contains(err.Error(), tv) {
					t.Errorf("error %q contains the value of %s", err, k)
				}
			}
			if strings.Contains(err.Error(), "zq") {
				t.Errorf("error %q contains a marker from a value", err)
			}
		})
	}
}

func TestLoadConfigNamesTheFirstOffender(t *testing.T) {
	_, _, err := LoadConfig(env(map[string]string{"NTFY_TOKEN": "tk_a", "NTFY_TOPIC_PREFIX": "hbt", "NTFY_PUBLIC_URL": "https://a.example.org"}))
	if want := "NTFY_TOKEN requires NTFY_URL"; err == nil || err.Error() != want {
		t.Fatalf("got %v, want %q", err, want)
	}
	_, _, err = LoadConfig(env(map[string]string{"NTFY_URL": "http://ntfy:2586"}))
	if want := "NTFY_TOPIC_PREFIX is required when NTFY_URL is set"; err == nil || err.Error() != want {
		t.Fatalf("got %v, want %q", err, want)
	}
}
