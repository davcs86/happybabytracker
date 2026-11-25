// Package ics writes one RFC 5545 calendar holding one event.
package ics

import (
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"
)

// lineOctets is RFC 5545's limit before folding, CRLF excluded.
const lineOctets = 75

// Event is one VEVENT; empty optional text is left out, and AlarmBefore 0 means no VALARM.
type Event struct {
	UID                            string
	Sequence                       int
	Stamp, Start, End              time.Time
	Cancelled                      bool
	Summary, Location, Description string
	URL                            string
	AlarmBefore                    time.Duration
}

// Write writes e as a whole VCALENDAR with CRLF line ends.
func Write(w io.Writer, e Event) error {
	status := "CONFIRMED"
	if e.Cancelled {
		status = "CANCELLED"
	}
	lines := []string{
		"BEGIN:VCALENDAR", "VERSION:2.0", "PRODID:-//happybabytracker//visits//EN", "CALSCALE:GREGORIAN", "METHOD:PUBLISH",
		"BEGIN:VEVENT",
		"UID:" + e.UID,
		fmt.Sprintf("SEQUENCE:%d", e.Sequence),
		"DTSTAMP:" + instant(e.Stamp),
		"LAST-MODIFIED:" + instant(e.Stamp),
		"DTSTART:" + instant(e.Start),
		"DTEND:" + instant(e.End),
		"STATUS:" + status,
		"SUMMARY:" + text(e.Summary),
	}
	if e.Location != "" {
		lines = append(lines, "LOCATION:"+text(e.Location))
	}
	if e.Description != "" {
		lines = append(lines, "DESCRIPTION:"+text(e.Description))
	}
	if e.URL != "" {
		lines = append(lines, "URL:"+e.URL)
	}
	if e.AlarmBefore > 0 {
		lines = append(lines, "BEGIN:VALARM", "ACTION:DISPLAY", "DESCRIPTION:"+text(e.Summary),
			"TRIGGER:"+beforeDuration(e.AlarmBefore), "END:VALARM")
	}
	lines = append(lines, "END:VEVENT", "END:VCALENDAR")
	var b strings.Builder
	for _, l := range lines {
		b.WriteString(fold(l))
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func instant(t time.Time) string { return t.UTC().Format("20060102T150405Z") }

// text escapes a TEXT value (RFC 5545 3.3.11).
func text(s string) string {
	return strings.NewReplacer(`\`, `\\`, ";", `\;`, ",", `\,`, "\r\n", `\n`, "\n", `\n`, "\r", `\n`).Replace(s)
}

// beforeDuration is a negative DURATION in whole hours when it divides, else whole minutes.
func beforeDuration(d time.Duration) string {
	if d%time.Hour == 0 {
		return fmt.Sprintf("-PT%dH", d/time.Hour)
	}
	return fmt.Sprintf("-PT%dM", d/time.Minute)
}

// fold ends l with CRLF, splitting it into lines of at most 75 octets at rune boundaries, continuations led by a space.
func fold(l string) string {
	var b strings.Builder
	limit := lineOctets
	for len(l) > limit {
		cut := limit
		for cut > 0 && !utf8.RuneStart(l[cut]) {
			cut--
		}
		b.WriteString(l[:cut])
		b.WriteString("\r\n ")
		l = l[cut:]
		limit = lineOctets - 1
	}
	b.WriteString(l)
	b.WriteString("\r\n")
	return b.String()
}
