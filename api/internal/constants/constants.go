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
	LimiterWindow          = time.Minute
	LimiterMaxKeys         = 10000
	TokenBytes             = 32
	MaxBodyBytes           = 64 << 10

	Argon2MemoryKiB   = 19456
	Argon2Iterations  = 2
	Argon2Parallelism = 1
	Argon2SaltBytes   = 16
	Argon2KeyBytes    = 32

	MaxConcurrentHashes = 2

	SQLiteBusyTimeout = 5 * time.Second
	ReadPoolMax       = 4

	UsernamePattern = `^[a-z0-9._-]{3,32}$`

	DisplayNameMinLen = 1
	DisplayNameMaxLen = 64

	KidNameMinLen     = 1
	KidNameMaxLen     = 64
	KidMinBirthDate   = "1900-01-01"
	GestationWeeksMin = 22
	GestationWeeksMax = 44
	GestationDaysMin  = 0
	GestationDaysMax  = 6

	DefaultAddr    = ":9210"
	DefaultDataDir = "/data"

	ReadHeaderTimeout = 5 * time.Second
	ReadTimeout       = 15 * time.Second
	WriteTimeout      = 30 * time.Second
	ShutdownTimeout   = 10 * time.Second

	LogFieldCause    = "cause"
	LogFieldSlide    = "slide"
	SlideFailed      = "failed"
	CauseSQLite      = "sqlite"
	CauseOtherPrefix = "other:"
)
