package model

import (
	"slices"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/schedule"
	"github.com/davcs86/happybabytracker/api/internal/units"
)

// Entry holds the columns every tracker row shares.
type Entry struct {
	ID, KidID, CreatedBy, UpdatedBy string
	CreatedAt, UpdatedAt            time.Time
	Note                            *string
}

// BottleSource is one content's share of an advanced bottle.
type BottleSource struct {
	Source string
	Volume units.Volume
}

// BottleFeed contents are a non-empty subset of the Content* names, stored in that order.
// Sources is nil or empty for a simple bottle; otherwise one per Contents name, in that order.
type BottleFeed struct {
	Entry
	At       time.Time
	Contents []string
	Volume   units.Volume
	Sources  []BottleSource
}

type Diaper struct {
	Entry
	At   time.Time
	Kind string
}

// Symptom is one symptom at a time. Temperature and Method are both set exactly when Symptom is "fever";
// OtherText is set exactly when it is "other".
type Symptom struct {
	Entry
	At          time.Time
	Symptom     string
	Severity    *string
	OtherText   *string
	Temperature *units.Temperature
	Method      *string
}

type Meal struct {
	Entry
	At       time.Time
	What     string
	Reaction *string
}

// Medication is a course. Its Schedule's Times are sorted; EndOn is a kid-local date (inclusive). LastDose, NextDue
// and DoseCount are filled by the store's reads and ignored by its writes.
type Medication struct {
	Entry
	schedule.Schedule
	Name, Dose string
	StartAt    time.Time
	EndOn      *string
	Active     bool
	LastDose   *LastDose
	NextDue    *time.Time
	DoseCount  int
}

// LastDose is the course's most recent dose.
type LastDose struct {
	ID      string
	GivenAt time.Time
	Amount  *string
}

// MedicationDose is one dose given from the course MedicationID, which never changes after creation.
type MedicationDose struct {
	Entry
	MedicationID string
	GivenAt      time.Time
	Amount       *string
}

// PumpingSide is one side's share of an advanced pumping session.
type PumpingSide struct {
	Side   string
	Volume units.Volume
}

// PumpingSession is simple when Sides is nil or empty; otherwise Sides holds one per side pumped, in PumpingSideNames
// order, and Side is the one they imply (DerivedPumpingSide).
type PumpingSession struct {
	Entry
	StartedAt       time.Time
	DurationSeconds int
	Volume          units.Volume
	Side            string
	Sides           []PumpingSide
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
	WeightBefore, WeightAfter *units.Weight
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
	if n.EndedAt != nil {
		if n.Side != nil || n.SideSince != nil {
			return "side"
		}
		if f := durationField(n.StartedAt, *n.EndedAt); f != "" {
			return f
		}
		if !secondsFit(n.LeftSeconds, n.RightSeconds, n.EndedAt.Sub(n.StartedAt)) {
			return "left_seconds"
		}
		return ""
	}
	if n.Side == nil || n.SideSince == nil || !slices.Contains(NursingSides, *n.Side) {
		return "side"
	}
	if n.SideSince.Before(n.StartedAt) || !secondsFit(n.LeftSeconds, n.RightSeconds, n.SideSince.Sub(n.StartedAt)) {
		return "started_at"
	}
	return ""
}

// secondsFit is left + right <= span in whole seconds, for non-negative left and right; it never adds them,
// so huge client values cannot wrap past the check.
func secondsFit(left, right int, span time.Duration) bool {
	d := int64(span / time.Second)
	return int64(left) <= d && int64(right) <= d-int64(left)
}

// Restarted moves n's start to start; an unswitched running session (SideSince at the start) feeds from start.
func (n NursingSession) Restarted(start time.Time) NursingSession {
	if n.EndedAt == nil && n.SideSince != nil && n.SideSince.Equal(n.StartedAt) {
		n.SideSince = &start
	}
	n.StartedAt = start
	return n
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

// Growth is one dated measurement; at least one of Weight, Length and Head is set.
type Growth struct {
	Entry
	MeasuredOn   string // YYYY-MM-DD in the kid's zone
	Weight       *units.Weight
	Length, Head *units.Length
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

// Symptoms lists every symptom name.
var Symptoms = []string{"congestion", "cough", "fever", "vomiting", "diarrhea", "constipation", "rash", "ear_pain",
	"sore_throat", "poor_appetite", "unusually_sleepy", "other"}

// Severities lists every symptom severity.
var Severities = []string{"mild", "moderate", "severe"}

// TempMethods lists every temperature-taking method.
var TempMethods = []string{"oral", "rectal", "axillary", "ear", "forehead", "other"}

// NursingSides lists every nursing side.
var NursingSides = []string{"left", "right"}

// PumpingSides lists every pumping side.
var PumpingSides = []string{"left", "right", "both"}

// PumpingSideNames lists the sides that can carry a volume of their own, in storage order.
var PumpingSideNames = []string{"left", "right"}

// CanonicalPumpingSides returns in as a set in storage order; false for an empty set, a duplicate or an unknown name.
func CanonicalPumpingSides(in []string) ([]string, bool) {
	out := make([]string, 0, len(in))
	for _, name := range PumpingSideNames {
		if slices.Contains(in, name) {
			out = append(out, name)
		}
	}
	return out, len(out) > 0 && len(out) == len(in)
}

// DerivedPumpingSide is the Side that sides imply: both for two, else the one named; false unless they are a valid set.
func DerivedPumpingSide(sides []PumpingSide) (string, bool) {
	names := make([]string, len(sides))
	for i, s := range sides {
		names[i] = s.Side
	}
	set, ok := CanonicalPumpingSides(names)
	if !ok {
		return "", false
	}
	if len(set) == len(PumpingSideNames) {
		return "both", true
	}
	return set[0], true
}

// Module is a switchable tracker.
type Module string

// Modules lists every module id in display order.
var Modules = []Module{
	"bottles", "nursing", "pumping", "diapers", "sleep", "feeding",
	"milestones", "growth", "vaccination", "sickness", "medication",
}
