// Package model holds the domain types shared by the store and the HTTP layer.
package model

import (
	"time"

	"github.com/google/uuid"
)

// NewID returns a new UUIDv7 string.
func NewID() string {
	return uuid.Must(uuid.NewV7()).String()
}

// Role is a parent's access to one kid: "guardian" or "caregiver".
type Role string

const (
	RoleGuardian  Role = "guardian"
	RoleCaregiver Role = "caregiver"
)

// One-time token kinds.
const (
	TokenInvite = "invite"
	TokenReset  = "reset"
)

// Principal is who a request acts as; Kind is "admin" or "parent", and ParentID is empty for an admin.
type Principal struct {
	Kind      string
	ParentID  string
	SessionID string
	ExpiresAt time.Time
}

type Units struct {
	Weight, Length, Volume, Temp string
}

type Parent struct {
	ID, Username, DisplayName string
	Disabled, HasPassword     bool
	Units                     Units
	CreatedAt                 time.Time
}

type SessionInfo struct {
	ID                    string
	CreatedAt, LastSeenAt time.Time
	Current               bool
}

type Kid struct {
	ID, Name, BirthDate, Sex, TimeZone string
	GestationWeeks, GestationDays      *int
	GrowthSource                       string  // a growth.Source: "who" or "cdc"
	VaccineSchema                      *string // a vaccines schedule id; nil until a guardian chooses one
	ReminderOverdueMinutes             int     // minutes after a due time before a reminder counts as overdue
}

// Administration is one vaccination visit's dose: one stored row per antigen, sharing ID; Antigens are sorted.
type Administration struct {
	ID, KidID, GivenOn   string
	Antigens             []string
	Lot, Product, Note   *string
	CreatedBy, UpdatedBy string
	CreatedAt, UpdatedAt time.Time
}

// VaccineDose is one stored row: one antigen of the administration it belongs to.
type VaccineDose struct {
	ID, KidID, Administration, Antigen, GivenOn string
	Lot, Product, Note                          *string
	CreatedBy, UpdatedBy                        string
	CreatedAt, UpdatedAt                        time.Time
}

type KidWithRole struct {
	Kid
	Role Role
}

type Grant struct {
	ParentID, Username, DisplayName string
	Role                            Role
}
