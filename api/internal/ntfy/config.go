// Package ntfy is the optional ntfy channel: its configuration and the publisher.
package ntfy

import (
	"errors"
	"net/url"
	"regexp"
	"strings"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

// Config is the loaded ntfy configuration; the zero value is off. PublicURL is empty when unset.
type Config struct{ URL, PublicURL, TopicPrefix, Token string }

var prefixRE = regexp.MustCompile(constants.NtfyPrefixPattern)

// LoadConfig: ntfy is on only when NTFY_URL is set. An empty value is unset, and errors name the variable
// and never echo a value.
func LoadConfig(getenv func(string) string) (cfg Config, enabled bool, err error) {
	u, token, prefix, public := getenv("NTFY_URL"), getenv("NTFY_TOKEN"), getenv("NTFY_TOPIC_PREFIX"), getenv("NTFY_PUBLIC_URL")
	if u == "" {
		for _, v := range []struct{ name, val string }{{"NTFY_TOKEN", token}, {"NTFY_TOPIC_PREFIX", prefix}, {"NTFY_PUBLIC_URL", public}} {
			if v.val != "" {
				return Config{}, false, errors.New(v.name + " requires NTFY_URL")
			}
		}
		return Config{}, false, nil
	}
	base, ok := serverURL(u, "http", "https")
	if !ok {
		return Config{}, false, errors.New("NTFY_URL must be an http: or https: URL with a host and no path, userinfo, query or fragment")
	}
	switch {
	case prefix == "":
		return Config{}, false, errors.New("NTFY_TOPIC_PREFIX is required when NTFY_URL is set")
	case !prefixRE.MatchString(prefix):
		return Config{}, false, errors.New("NTFY_TOPIC_PREFIX must match " + constants.NtfyPrefixPattern)
	case !validToken(token):
		return Config{}, false, errors.New("NTFY_TOKEN must be printable ASCII without spaces")
	}
	cfg = Config{URL: base, TopicPrefix: prefix, Token: token}
	if public != "" {
		if cfg.PublicURL, ok = serverURL(public, "https"); !ok {
			return Config{}, false, errors.New("NTFY_PUBLIC_URL must be an https: URL with a host and no path, userinfo, query or fragment")
		}
	}
	return cfg, true, nil
}

// serverURL returns scheme://host[:port] when raw is exactly that, optionally with one trailing slash.
func serverURL(raw string, schemes ...string) (string, bool) {
	if strings.ContainsAny(raw, "?#") {
		return "", false
	}
	u, err := url.Parse(raw)
	if err != nil || u.User != nil || u.Opaque != "" || u.Hostname() == "" || (u.Path != "" && u.Path != "/") {
		return "", false
	}
	for _, s := range schemes {
		if u.Scheme == s {
			return u.Scheme + "://" + u.Host, true
		}
	}
	return "", false
}

// validToken accepts an empty token (none); the rest must be safe in an HTTP header value.
func validToken(s string) bool {
	for _, c := range []byte(s) {
		if c <= ' ' || c >= 0x7f {
			return false
		}
	}
	return true
}
