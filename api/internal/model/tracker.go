package model

import (
	"time"

	"github.com/davcs86/happybabytracker/api/internal/units"
)

// Entry holds the columns every tracker row shares.
type Entry struct {
	ID, KidID, CreatedBy, UpdatedBy string
	CreatedAt, UpdatedAt            time.Time
	Note                            *string
}

// BottleFeed contents are a non-empty subset of the Content* names, stored in that order.
type BottleFeed struct {
	Entry
	At       time.Time
	Contents []string
	Volume   units.Volume
}

type Diaper struct {
	Entry
	At   time.Time
	Kind string
}

type Meal struct {
	Entry
	At       time.Time
	What     string
	Reaction *string
}

type PumpingSession struct {
	Entry
	StartedAt       time.Time
	DurationSeconds int
	Volume          units.Volume
	Side            string
}

// SleepSession is running while EndedAt is nil.
type SleepSession struct {
	Entry
	StartedAt time.Time
	EndedAt   *time.Time
}

// NursingSession is running while EndedAt is nil; Side and SideSince name the side now being fed.
type NursingSession struct {
	Entry
	StartedAt                 time.Time
	EndedAt                   *time.Time
	LeftSeconds, RightSeconds int
	Side                      *string
	SideSince                 *time.Time
}

// Bottle content names, in storage order.
const (
	ContentBreastMilk = "breast_milk"
	ContentFormula    = "formula"
	ContentCowMilk    = "cow_milk"
	ContentOther      = "other"
)

// Module is a switchable tracker.
type Module string

// Modules lists every module id in display order.
var Modules = []Module{
	"bottles", "nursing", "pumping", "diapers", "sleep", "feeding",
	"milestones", "growth", "vaccination", "sickness", "medication",
}
