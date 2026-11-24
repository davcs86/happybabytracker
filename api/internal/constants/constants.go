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

	FlOzMicroMLNumerator   = 295735
	FlOzMicroMLDenominator = 10000
	MaxVolumeAmount        = 1 << 24

	GramsPerOzNumerator   = 28349523125
	GramsPerOzDenominator = 1000000000
	MaxWeightAmount       = 1 << 24

	// 1.03 g/ml: an approximation in the range commonly used (1.0–1.03 g/ml); not measured.
	MilkDensityHundredths = 103

	MMPerInchNumerator   = 254
	MMPerInchDenominator = 10
	MaxLengthAmount      = 1 << 24

	TrackerPageDefault   = 50
	TrackerPageMax       = 100
	MaxFutureSkew        = 5 * time.Minute
	MinSessionDuration   = time.Second
	MaxSessionDuration   = 24 * time.Hour
	MaxNoteLen           = 500
	SummaryMaxDays       = 93
	BandOverrideLookback = 60 * 24 * time.Hour
	CursorIDPattern      = `^[A-Za-z0-9-]{1,64}$`

	MinBottleVolumeML = 1
	MaxBottleVolumeML = 1000

	MinPumpingVolumeML = 1
	MaxPumpingVolumeML = 2000

	MaxMealTextLen = 200

	MinWeightGrams     = 200
	MaxWeightGrams     = 250000
	MinLengthMM        = 150
	MaxLengthMM        = 2500
	MinHeadMM          = 150
	MaxHeadMM          = 800
	MaxGrowthPerKid    = 3000
	ZScoreDecimals     = 2
	PercentileDecimals = 1
	BMIDecimals        = 2

	BandSecondStartMonths = 12
	BandThirdStartMonths  = 24

	TermGestationDays   = 280
	PretermBelowDays    = 259
	CorrectionEndsYears = 2

	DaysPerMonth          = 30.4375
	CurveFineStepMonths   = 0.5
	CurveCoarseStepMonths = 1
	CurveFineUntilMonths  = 24

	CurveMaxMonths          = 240
	CurveDefaultMinToMonths = 24
	CurveDefaultAheadMonths = 6
	// Display-only factors for the curves' imperial column; stored measurements use the exact ratios above.
	PoundsPerKG = 2.2046226218
	CMPerInch   = 2.54

	MaxVaccineAdministrationsPerKid = 500
	VaccineLotMaxLen                = 40
	VaccineProductMaxLen            = 80
	VaccineDefaultWindowMonths      = 1
	// Schedule ages above about 130 years are refused, per unit.
	VaccineMaxAgeDays   = 47482
	VaccineMaxAgeWeeks  = 6783
	VaccineMaxAgeMonths = 1560
	VaccineMaxAgeYears  = 130
	// A mixed-unit until must lead from by this many mean days, so the window is non-empty for every birth date.
	VaccineMinMixedUnitGapDays = 4

	MaxMedicationsPerKid     = 50
	MaxMedicationDosesPerKid = 5000
	MaxScheduleTimes         = 12
	MinEveryHours            = 1
	MaxEveryHours            = 168
	MaxScheduleOccurrences   = 1000 // Occurrences returns the earliest this many, truncated silently.
	MedicationTextMaxLen     = 80
	SymptomOtherMaxLen       = 80
	MinTempCHundredths       = 3000
	MaxTempCHundredths       = 4500

	DefaultReminderOverdueMinutes = 60
	MinReminderOverdueMinutes     = 5
	MaxReminderOverdueMinutes     = 1440

	PushDevicesPerParent = 10
	PushLabelMaxLen      = 40
	PushEndpointMaxLen   = 2048
	DefaultPushLabel     = "This device" // a new device's name when the upsert sends none

	ReminderTick        = 30 * time.Second
	ReminderExpiry      = 30 * time.Minute
	DoseDoneLead        = 30 * time.Minute
	ReminderHealthStale = 15 * time.Minute
	WorkerStaleAfter    = 60 * time.Second
	// A settle outlives the worker's cancelled context by at most this, so a send cut off by shutdown is still stored.
	ReminderSettleTimeout = 5 * time.Second

	PushTTLMax                 = 3600 * time.Second
	PushTimeout                = 10 * time.Second
	PushJWTLifetime            = 12 * time.Hour
	PushResponseMaxBytes       = 4096
	PushResponseHeaderMaxBytes = 8192
	PushIdleConnTimeout        = 30 * time.Second
	DeliveryConcurrency        = 8

	NtfyTimeout          = 10 * time.Second
	NtfyTopicRandomLen   = 22
	NtfyPrefixPattern    = `^[a-z0-9-]{1,32}$`
	NtfyMessageMaxBytes  = 4096
	NtfyResponseMaxBytes = 4096

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

// VaccineSchedules lists the schedule ids the API may ship, in display order.
var VaccineSchedules = []string{"us", "uk", "mx"}

// MilestoneCatalogTextMaxLen caps one catalog item's text, in characters.
const MilestoneCatalogTextMaxLen = 200

// MilestoneCatalogAges lists the checklist ages, in months, a catalog item may carry.
var MilestoneCatalogAges = []int{2, 4, 6, 9, 12, 15, 18, 24, 30, 36, 48, 60}

// MilestoneCatalogUntilYears is the age in whole years from which the checklists are no longer shown.
const MilestoneCatalogUntilYears = 5

// MilestoneTextMaxLen caps a custom milestone's text, in characters after trimming.
const MilestoneTextMaxLen = 200

// MaxMilestonesPerKid caps a kid's reached milestones, catalog and custom together.
const MaxMilestonesPerKid = 1000

// ReminderRetryDelays lists the wait after each failed send attempt, in order.
var ReminderRetryDelays = []time.Duration{30 * time.Second, 2 * time.Minute, 10 * time.Minute}
