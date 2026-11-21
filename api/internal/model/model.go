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
}

type KidWithRole struct {
	Kid
	Role Role
}

type Grant struct {
	ParentID, Username, DisplayName string
	Role                            Role
}
