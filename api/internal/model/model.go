// Package model holds the domain types shared by the store and the HTTP layer.
package model

import "github.com/google/uuid"

// NewID returns a new UUIDv7 string.
func NewID() string {
	return uuid.Must(uuid.NewV7()).String()
}
