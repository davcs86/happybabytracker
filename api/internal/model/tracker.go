package model

import (
	"slices"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
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

// InvalidField names the field that breaks a sleep invariant, "" when none does.
func (v SleepSession) InvalidField() string {
	if v.EndedAt != nil {
		return durationField(v.StartedAt, *v.EndedAt)
	}
	return ""
}

func durationField(start, end time.Time) string {
	if end.Before(start) || end.Sub(start) > constants.MaxSessionDuration {
		return "ended_at"
	}
	return ""
}

// InvalidField names the field that breaks a nursing invariant, "" when none does: Side and SideSince are both set
// exactly while the session runs, and the seconds credited never exceed the time they cover.
func (n NursingSession) InvalidField() string {
	if n.LeftSeconds < 0 {
		return "left_seconds"
	}
	if n.RightSeconds < 0 {
		return "right_seconds"
	}
	credited := int64(n.LeftSeconds) + int64(n.RightSeconds)
	if n.EndedAt != nil {
		if n.Side != nil || n.SideSince != nil {
			return "side"
		}
		if f := durationField(n.StartedAt, *n.EndedAt); f != "" {
			return f
		}
		if credited > int64(n.EndedAt.Sub(n.StartedAt)/time.Second) {
			return "left_seconds"
		}
		return ""
	}
	if n.Side == nil || n.SideSince == nil || !slices.Contains(NursingSides, *n.Side) {
		return "side"
	}
	if n.SideSince.Before(n.StartedAt) || credited > int64(n.SideSince.Sub(n.StartedAt)/time.Second) {
		return "started_at"
	}
	return ""
}

// Ended closes the running n at at, crediting the side in use; false when n is not running or at precedes SideSince.
func (n NursingSession) Ended(at time.Time) (NursingSession, bool) {
	n, ok := n.credit(at)
	if ok {
		n.EndedAt, n.Side, n.SideSince = &at, nil, nil
	}
	return n, ok
}

// Switched credits the side in use up to at and feeds side from at; false as for Ended.
func (n NursingSession) Switched(side string, at time.Time) (NursingSession, bool) {
	n, ok := n.credit(at)
	if ok {
		n.Side, n.SideSince = &side, &at
	}
	return n, ok
}

func (n NursingSession) credit(at time.Time) (NursingSession, bool) {
	if n.EndedAt != nil || n.Side == nil || n.SideSince == nil || at.Before(*n.SideSince) {
		return n, false
	}
	seconds := int(at.Sub(*n.SideSince) / time.Second)
	switch *n.Side {
	case "left":
		n.LeftSeconds += seconds
	case "right":
		n.RightSeconds += seconds
	default:
		return n, false
	}
	return n, true
}

// Bottle content names, in storage order.
const (
	ContentBreastMilk = "breast_milk"
	ContentFormula    = "formula"
	ContentCowMilk    = "cow_milk"
	ContentOther      = "other"
)

// ContentNames lists every bottle content name in storage order.
var ContentNames = []string{ContentBreastMilk, ContentFormula, ContentCowMilk, ContentOther}

// CanonicalContents returns in as a set in storage order; false for an empty set, a duplicate or an unknown name.
func CanonicalContents(in []string) ([]string, bool) {
	out := make([]string, 0, len(in))
	for _, name := range ContentNames {
		if slices.Contains(in, name) {
			out = append(out, name)
		}
	}
	return out, len(out) > 0 && len(out) == len(in) // a duplicate or unknown name leaves out shorter than in
}

// DiaperKinds lists every diaper kind.
var DiaperKinds = []string{"wet", "dirty", "both", "dry"}

// NursingSides lists every nursing side.
var NursingSides = []string{"left", "right"}

// PumpingSides lists every pumping side.
var PumpingSides = []string{"left", "right", "both"}

// Module is a switchable tracker.
type Module string

// Modules lists every module id in display order.
var Modules = []Module{
	"bottles", "nursing", "pumping", "diapers", "sleep", "feeding",
	"milestones", "growth", "vaccination", "sickness", "medication",
}
