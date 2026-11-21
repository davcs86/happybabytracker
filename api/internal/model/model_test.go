package model

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewIDIsUUIDv7(t *testing.T) {
	a, b := NewID(), NewID()
	if a == b {
		t.Fatal("NewID repeated")
	}
	u, err := uuid.Parse(a)
	if err != nil || u.Version() != 7 {
		t.Fatalf("NewID %q: version %v err %v", a, u.Version(), err)
	}
}
