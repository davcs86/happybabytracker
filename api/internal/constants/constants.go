// Package constants holds every limit, duration and default of the API; no other package spells one.
package constants

import "time"

const (
	MinPasswordLen      = 12
	MaxPasswordLen      = 256
	MinAdminPasswordLen = 16
	MinSecretLen        = 32

	InviteTTL            = 72 * time.Hour
	ResetTTL             = 1 * time.Hour
	ParentSessionTTL     = 720 * time.Hour
	SessionTouchInterval = 1 * time.Hour
	AdminSessionTTL      = 12 * time.Hour

	LoginAttemptsPerMinute = 5
	TokenBytes             = 32
	MaxBodyBytes           = 64 << 10

	SQLiteBusyTimeout = 5 * time.Second
	ReadPoolMax       = 4

	UsernamePattern = `^[a-z0-9._-]{3,32}$`

	DefaultAddr    = ":9210"
	DefaultDataDir = "/data"
)
