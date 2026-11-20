// Package config reads and validates the process environment.
package config

import (
	"fmt"
	"net/url"
	"regexp"
	"unicode/utf8"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

type Config struct {
	AdminUsername    string
	AdminPassword    string
	InternalAPIToken string
	PublicOrigin     string
	DataDir          string
	Addr             string
}

var usernameRE = regexp.MustCompile(constants.UsernamePattern)

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
	if !usernameRE.MatchString(cfg.AdminUsername) {
		return Config{}, fmt.Errorf("ADMIN_USERNAME must match %s", constants.UsernamePattern)
	}
	if utf8.RuneCountInString(cfg.AdminPassword) < constants.MinAdminPasswordLen {
		return Config{}, fmt.Errorf("ADMIN_PASSWORD must be at least %d characters", constants.MinAdminPasswordLen)
	}
	if utf8.RuneCountInString(cfg.InternalAPIToken) < constants.MinSecretLen {
		return Config{}, fmt.Errorf("INTERNAL_API_TOKEN must be at least %d characters", constants.MinSecretLen)
	}
	if !validOrigin(cfg.PublicOrigin) {
		return Config{}, fmt.Errorf("PUBLIC_ORIGIN must be http(s)://host[:port] with no path")
	}
	return cfg, nil
}

func validOrigin(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" && u.Hostname() != "" &&
		u.Path == "" && u.RawQuery == "" && u.Fragment == "" && u.User == nil && !u.ForceQuery
}
