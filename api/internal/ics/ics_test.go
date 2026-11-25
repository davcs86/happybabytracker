package ics

import (
	"strings"
	"testing"
	"time"
)

var (
	stamp = time.Date(2026, 11, 20, 12, 0, 5, 0, time.UTC)
	start = time.Date(2026, 12, 3, 9, 30, 0, 0, time.UTC)
)

func write(t *testing.T, e Event) string {
	t.Helper()
	var b strings.Builder
	if err := Write(&b, e); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func crlf(lines ...string) string { return strings.Join(lines, "\r\n") + "\r\n" }

const head = "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//happybabytracker//visits//EN\r\nCALSCALE:GREGORIAN\r\nMETHOD:PUBLISH\r\nBEGIN:VEVENT\r\n"

func TestGeneric(t *testing.T) {
	got := write(t, Event{UID: "visit-v1@hbt.example", Sequence: 3, Stamp: stamp, Start: start, End: start.Add(30 * time.Minute),
		Summary: "Medical visit", URL: "https://hbt.example/kids/k1/visits/v1", AlarmBefore: 24 * time.Hour})
	want := head + crlf("UID:visit-v1@hbt.example", "SEQUENCE:3", "DTSTAMP:20261120T120005Z", "LAST-MODIFIED:20261120T120005Z",
		"DTSTART:20261203T093000Z", "DTEND:20261203T100000Z", "STATUS:CONFIRMED", "SUMMARY:Medical visit",
		"URL:https://hbt.example/kids/k1/visits/v1", "BEGIN:VALARM", "ACTION:DISPLAY", "DESCRIPTION:Medical visit", "TRIGGER:-PT24H",
		"END:VALARM", "END:VEVENT", "END:VCALENDAR")
	if got != want {
		t.Errorf("generic:\n%q\nwant\n%q", got, want)
	}
}

func TestFullAndEscaping(t *testing.T) {
	got := write(t, Event{UID: "visit-v1@hbt.example", Stamp: stamp, Start: start, End: start.Add(time.Hour),
		Summary: "Lía: Well-child visit with Dr. Lee", Location: "12 Main St, Suite 4; back door", Description: "a\\b\nnext line"})
	want := head + crlf("UID:visit-v1@hbt.example", "SEQUENCE:0", "DTSTAMP:20261120T120005Z", "LAST-MODIFIED:20261120T120005Z",
		"DTSTART:20261203T093000Z", "DTEND:20261203T103000Z", "STATUS:CONFIRMED", "SUMMARY:Lía: Well-child visit with Dr. Lee",
		`LOCATION:12 Main St\, Suite 4\; back door`, `DESCRIPTION:a\\b\nnext line`, "END:VEVENT", "END:VCALENDAR")
	if got != want {
		t.Errorf("full:\n%q\nwant\n%q", got, want)
	}
}

func TestCancelledHasNoAlarm(t *testing.T) {
	got := write(t, Event{UID: "u", Stamp: stamp, Start: start, End: start, Cancelled: true, Summary: "Medical visit"})
	if !strings.Contains(got, "\r\nSTATUS:CANCELLED\r\n") || strings.Contains(got, "VALARM") {
		t.Errorf("cancelled: %q", got)
	}
}

func TestFoldingKeepsRunesWhole(t *testing.T) {
	summary := strings.Repeat("é", 70) // 140 octets
	got := write(t, Event{UID: "u", Stamp: stamp, Start: start, End: start, Summary: summary})
	var unfolded strings.Builder
	for i, line := range strings.Split(strings.TrimSuffix(got, "\r\n"), "\r\n") {
		if len(line) > lineOctets+1 || (len(line) > lineOctets && !strings.HasPrefix(line, " ")) {
			t.Errorf("line %d is %d octets: %q", i, len(line), line)
		}
		if !utf8Valid(line) {
			t.Errorf("line %d splits a rune: %q", i, line)
		}
		if strings.HasPrefix(line, " ") {
			unfolded.WriteString(line[1:])
		} else {
			unfolded.WriteString("\n" + line)
		}
	}
	if !strings.Contains(unfolded.String(), "\nSUMMARY:"+summary+"\n") {
		t.Errorf("unfolding does not give the summary back")
	}
}

func utf8Valid(s string) bool { return strings.ToValidUTF8(s, "�") == s }

func TestInstantsAreUTCAcrossDST(t *testing.T) {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	before := time.Date(2026, 3, 7, 9, 30, 0, 0, ny) // EST, the day before the change
	after := time.Date(2026, 3, 8, 9, 30, 0, 0, ny)  // EDT
	got := write(t, Event{UID: "u", Stamp: stamp, Start: before, End: after, Summary: "x"})
	if !strings.Contains(got, "\r\nDTSTART:20260307T143000Z\r\n") || !strings.Contains(got, "\r\nDTEND:20260308T133000Z\r\n") {
		t.Errorf("DST: %q", got)
	}
}
