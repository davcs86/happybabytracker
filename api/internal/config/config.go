// Package config reads and validates the process environment.
package config

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"net/url"
	"unicode/utf8"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/ntfy"
	"github.com/davcs86/happybabytracker/api/internal/push"
)

// AuthMode is AUTH_MODE; unset loads as AuthLocal.
type AuthMode string

const (
	AuthLocal AuthMode = "local"
	AuthOIDC  AuthMode = "oidc"
	AuthBoth  AuthMode = "both"
)

type Config struct {
	AdminUsername    string
	AdminPassword    string
	InternalAPIToken string
	PublicOrigin     string
	DataDir          string
	Addr             string
	// VAPIDKey is nil when Web Push is off.
	VAPIDKey      *ecdsa.PrivateKey
	VAPIDSubject  string
	AllowFakePush bool
	// Ntfy is the zero value when ntfy is off.
	Ntfy        ntfy.Config
	NtfyEnabled bool
	NPISearch   bool
	AuthMode    AuthMode
}

// Load errors name the variable and never echo its value.
func Load(getenv func(string) string) (Config, error) {
	cfg := Config{
		AdminUsername:    getenv("ADMIN_USERNAME"),
		AdminPassword:    getenv("ADMIN_PASSWORD"),
		InternalAPIToken: getenv("INTERNAL_API_TOKEN"),
		PublicOrigin:     getenv("PUBLIC_ORIGIN"),
		DataDir:          getenv("DATA_DIR"),
		Addr:             constants.DefaultAddr,
	}
	if cfg.DataDir == "" {
		cfg.DataDir = constants.DefaultDataDir
	}
	if utf8.RuneCountInString(cfg.InternalAPIToken) < constants.MinSecretLen {
		return Config{}, fmt.Errorf("INTERNAL_API_TOKEN must be at least %d characters", constants.MinSecretLen)
	}
	if !validOrigin(cfg.PublicOrigin) {
		return Config{}, fmt.Errorf("PUBLIC_ORIGIN must be http(s)://host[:port] with no path")
	}
	if err := loadPush(&cfg, getenv); err != nil {
		return Config{}, err
	}
	var err error
	if cfg.Ntfy, cfg.NtfyEnabled, err = ntfy.LoadConfig(getenv); err != nil {
		return Config{}, err
	}
	if cfg.NPISearch, err = loadNPI(getenv); err != nil {
		return Config{}, err
	}
	if cfg.AuthMode, err = loadAuthMode(getenv); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// loadNPI is the provider lookup flag (visits spec 10): 1 or unset; the error never echoes the value.
func loadNPI(getenv func(string) string) (bool, error) {
	switch getenv("NPI_SEARCH") {
	case "":
		return false, nil
	case "1":
		return true, nil
	}
	return false, errors.New("NPI_SEARCH must be 1 or unset")
}

// loadAuthMode: unset is local; the error never echoes the value.
func loadAuthMode(getenv func(string) string) (AuthMode, error) {
	switch m := AuthMode(getenv("AUTH_MODE")); m {
	case "":
		return AuthLocal, nil
	case AuthLocal, AuthOIDC, AuthBoth:
		return m, nil
	}
	return "", errors.New("AUTH_MODE must be local, oidc or both")
}

// loadPush: both VAPID variables or neither (Web Push off); errors never echo a value.
func loadPush(cfg *Config, getenv func(string) string) error {
	key, subject, fake := getenv("VAPID_PRIVATE_KEY"), getenv("VAPID_SUBJECT"), getenv("HBT_ALLOW_FAKE_PUSH")
	if fake != "" && fake != "1" {
		return errors.New("HBT_ALLOW_FAKE_PUSH must be 1 or unset")
	}
	switch {
	case key == "" && subject == "":
		if fake == "1" {
			return errors.New("HBT_ALLOW_FAKE_PUSH=1 requires VAPID_PRIVATE_KEY and VAPID_SUBJECT")
		}
		return nil
	case subject == "":
		return errors.New("VAPID_SUBJECT is required when VAPID_PRIVATE_KEY is set")
	case key == "":
		return errors.New("VAPID_PRIVATE_KEY is required when VAPID_SUBJECT is set")
	}
	k, err := push.ParsePrivateKey(key)
	if err != nil {
		return errors.New("VAPID_PRIVATE_KEY must be a P-256 private key in unpadded base64url")
	}
	if _, err := push.NewSigner(k, subject); err != nil {
		return errors.New("VAPID_SUBJECT must be an https: URL with a host and no userinfo, query or fragment")
	}
	cfg.VAPIDKey, cfg.VAPIDSubject, cfg.AllowFakePush = k, subject, fake == "1"
	return nil
}

func validOrigin(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" && u.Hostname() != "" &&
		u.Path == "" && u.RawQuery == "" && u.Fragment == "" && u.User == nil && !u.ForceQuery
}
