package vaccines

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

const testProvenance = `# schema: us
# name: Test schedule
# authority: Test authority
# edition: 2025
# retrieved: 2026-11-22
# origin_url: https://example.org/schedule
# licence: Public domain
# attribution: Test authority
`

const testHeader = "line,group,antigen,series,from,until,kind,born_from,born_to,note\n"

// Provenance is lines 1-8, the header line 9, so the first row is CSV line 10.
func testFile(rows ...string) string {
	return testProvenance + testHeader + strings.Join(rows, "\n") + "\n"
}

func parseString(t *testing.T, in string) (Schema, error) {
	t.Helper()
	mustLoad(t)
	return parse("us.csv", strings.NewReader(in))
}

func TestParseValidSchema(t *testing.T) {
	s, err := parseString(t, testFile(
		"measles-1,MMR,measles,1,12m,16m,routine,,,",
		`measles-2,MMR,measles,2,4y,7y,routine,,,"Dose 2, at school entry"`,
		"rotavirus-1,Rotavirus,rotavirus,1,8w,,may_not_be_needed,,,",
		"influenza-1,Flu,influenza,1,6m,,recurring,,,Every season",
		"hepatitis_a-1,Hep A,hepatitis_a,1,1y,,supplementary,,,",
		"diphtheria-1,6-in-1,diphtheria,1,0d,,routine,,2024-06-30,",
		"diphtheria-1-b2024,6-in-1,diphtheria,1,8w,,routine,2024-07-01,,",
	))
	if err != nil {
		t.Fatal(err)
	}
	if s.ID != "us" || s.Name != "Test schedule" || s.Authority != "Test authority" || s.Edition != "2025" ||
		s.Retrieved != "2026-11-22" || s.OriginURL != "https://example.org/schedule" ||
		s.Licence != "Public domain" || s.Attribution != "Test authority" {
		t.Errorf("provenance = %+v", s)
	}
	if len(s.Lines) != 7 {
		t.Fatalf("%d lines, want 7", len(s.Lines))
	}
	m1 := s.Lines[0]
	if m1.ID != "measles-1" || m1.Group != "MMR" || m1.Antigen != "measles" || m1.Series != 1 ||
		m1.From != (Age{12, UnitMonth}) || m1.Until == nil || *m1.Until != (Age{16, UnitMonth}) || m1.Kind != KindRoutine {
		t.Errorf("line 0 = %+v", m1)
	}
	if s.Lines[1].Note != "Dose 2, at school entry" {
		t.Errorf("quoted note = %q", s.Lines[1].Note)
	}
	if s.Lines[2].Until != nil || s.Lines[2].Kind != KindMayNotBeNeeded {
		t.Errorf("line 2 = %+v", s.Lines[2])
	}
	if s.Lines[3].Kind != KindRecurring || s.Lines[4].Kind != KindSupplementary {
		t.Errorf("kinds = %s, %s", s.Lines[3].Kind, s.Lines[4].Kind)
	}
	if s.Lines[5].BornFrom != "" || s.Lines[5].BornTo != "2024-06-30" || s.Lines[6].BornFrom != "2024-07-01" {
		t.Errorf("cohorts = %+v / %+v", s.Lines[5], s.Lines[6])
	}
}

func TestParseCohortSeries(t *testing.T) {
	disjoint := testFile(
		"diphtheria-1,6-in-1,diphtheria,1,8w,,routine,,,",
		"diphtheria-2,6-in-1,diphtheria,2,12w,,routine,,,",
		"diphtheria-3,6-in-1,diphtheria,3,16w,,routine,,2024-06-30,",
		"diphtheria-3-b2024,6-in-1,diphtheria,3,18m,,routine,2024-07-01,,",
		"diphtheria-4,Pre-school,diphtheria,4,3y,,routine,,2024-06-30,",
	)
	if _, err := parseString(t, disjoint); err != nil {
		t.Errorf("disjoint cohorts sharing a series: %v", err)
	}
	overlap := testFile(
		"diphtheria-1,6-in-1,diphtheria,1,8w,,routine,,,",
		"diphtheria-2,6-in-1,diphtheria,2,12w,,routine,,2024-07-01,",
		"diphtheria-2-b2024,6-in-1,diphtheria,2,18m,,routine,2024-07-01,,",
	)
	want := "vaccines: us.csv:12: column series: series repeats in an overlapping cohort"
	if _, err := parseString(t, overlap); err == nil || err.Error() != want {
		t.Errorf("overlapping cohorts: err = %v, want %q", err, want)
	}
	// Series 2 exists only for kids born from 2024-07-01, so series 3 has a gap for kids born earlier.
	gapInCohort := testFile(
		"diphtheria-1,6-in-1,diphtheria,1,8w,,routine,,,",
		"diphtheria-2,6-in-1,diphtheria,2,12w,,routine,2024-07-01,,",
		"diphtheria-3,6-in-1,diphtheria,3,16w,,routine,,,",
	)
	want = "vaccines: us.csv:12: column series: series has a gap in its cohort"
	if _, err := parseString(t, gapInCohort); err == nil || err.Error() != want {
		t.Errorf("gap within a cohort: err = %v, want %q", err, want)
	}
	sameUnbounded := testFile(
		"measles-1,MMR,measles,1,12m,,routine,,,",
		"measles-1b,MMR,measles,1,13m,,routine,,,",
	)
	want = "vaccines: us.csv:11: column series: series repeats in an overlapping cohort"
	if _, err := parseString(t, sameUnbounded); err == nil || err.Error() != want {
		t.Errorf("two unbounded lines with one series: err = %v, want %q", err, want)
	}
	sameBornFrom := testFile(
		"measles-1,MMR,measles,1,12m,,routine,2024-01-01,,",
		"measles-1-b2025,MMR,measles,1,13m,,routine,2025-01-01,,",
	)
	if _, err := parseString(t, sameBornFrom); err == nil || err.Error() != want {
		t.Errorf("two born_from-only lines with one series: err = %v, want %q", err, want)
	}
	ukShaped := testFile(
		"diphtheria-1,6-in-1,diphtheria,1,8w,,routine,,,",
		"diphtheria-2,6-in-1,diphtheria,2,12w,,routine,,,",
		"diphtheria-3,6-in-1,diphtheria,3,16w,,routine,,,",
		"diphtheria-4-b2024,6-in-1,diphtheria,4,18m,,routine,2024-07-01,,",
		"diphtheria-4,4-in-1,diphtheria,4,40m,,routine,,2024-06-30,",
		"diphtheria-5-b2024,4-in-1,diphtheria,5,40m,,routine,2024-07-01,,",
		"diphtheria-5,3-in-1,diphtheria,5,168m,,routine,,2024-06-30,",
		"diphtheria-6-b2024,3-in-1,diphtheria,6,168m,,routine,2024-07-01,,",
	)
	if _, err := parseString(t, ukShaped); err != nil {
		t.Errorf("UK-shaped cohorts: %v", err)
	}
	cohortOnly := testFile("mmr-1,MMRV,measles,1,3y4m,,routine,2024-01-01,,")
	if _, err := parseString(t, cohortOnly); err == nil {
		t.Error("a compound age parsed")
	}
	if _, err := parseString(t, testFile("measles-1,MMRV,measles,1,40m,,routine,2024-01-01,,")); err != nil {
		t.Errorf("a line whose cohort leaves earlier births with no line: %v", err)
	}
}

func TestParseRefusals(t *testing.T) {
	ok := "measles-1,MMR,measles,1,12m,16m,routine,,,"
	cases := []struct{ name, in, want string }{
		{"missing provenance key", strings.Replace(testFile(ok), "# licence: Public domain\n", "", 1),
			"vaccines: us.csv:8: column licence: missing provenance key"},
		{"empty provenance value", strings.Replace(testFile(ok), "# edition: 2025", "# edition: ", 1),
			"vaccines: us.csv:4: column edition: is empty"},
		{"duplicate provenance key", strings.Replace(testFile(ok), "# edition: 2025\n", "# edition: 2025\n# edition: 2026\n", 1),
			"vaccines: us.csv:5: column edition: duplicate provenance key"},
		{"unknown provenance key", strings.Replace(testFile(ok), "# edition: 2025\n", "# edition: 2025\n# secret: x\n", 1),
			"vaccines: us.csv:5: column key: unknown provenance key"},
		{"malformed provenance line", strings.Replace(testFile(ok), "# edition: 2025", "# edition 2025", 1),
			"vaccines: us.csv:4: column key: not a provenance line"},
		{"schema not the file id", strings.Replace(testFile(ok), "# schema: us", "# schema: uk", 1),
			"vaccines: us.csv:1: column schema: does not match the file name"},
		{"malformed retrieved", strings.Replace(testFile(ok), "2026-11-22", "2026-13-01", 1),
			"vaccines: us.csv:5: column retrieved: not a YYYY-MM-DD date"},
		{"origin not https", strings.Replace(testFile(ok), "https://example.org", "http://example.org", 1),
			"vaccines: us.csv:6: column origin_url: not an https URL"},
		{"bad header", strings.Replace(testFile(ok), "kind,born_from", "type,born_from", 1),
			"vaccines: us.csv:9: column kind: header is not " + strings.TrimSuffix(testHeader, "\n")},
		{"no header", testProvenance,
			"vaccines: us.csv:9: column line: header is not " + strings.TrimSuffix(testHeader, "\n")},
		{"no lines", testProvenance + testHeader,
			"vaccines: us.csv:9: column line: no lines"},
		{"wrong field count", testFile("measles-1,MMR,measles,1,12m,16m,routine,,"),
			"vaccines: us.csv:10: column record: wrong number of fields"},
		{"empty line id", testFile(",MMR,measles,1,12m,16m,routine,,,"),
			"vaccines: us.csv:10: column line: is empty"},
		{"duplicate line id", testFile(ok, "measles-1,MMR,measles,2,4y,,routine,,,"),
			"vaccines: us.csv:11: column line: duplicate line id"},
		{"empty group", testFile("measles-1,,measles,1,12m,16m,routine,,,"),
			"vaccines: us.csv:10: column group: is empty"},
		{"unknown antigen", testFile("smallpox-1,X,smallpox,1,12m,,routine,,,"),
			"vaccines: us.csv:10: column antigen: unknown antigen"},
		{"unknown kind", testFile("measles-1,MMR,measles,1,12m,16m,optional,,,"),
			"vaccines: us.csv:10: column kind: unknown kind"},
		{"series not a number", testFile("measles-1,MMR,measles,one,12m,16m,routine,,,"),
			"vaccines: us.csv:10: column series: not a positive integer"},
		{"series zero", testFile("measles-1,MMR,measles,0,12m,16m,routine,,,"),
			"vaccines: us.csv:10: column series: not a positive integer"},
		{"series not starting at 1", testFile("measles-2,MMR,measles,2,12m,16m,routine,,,"),
			"vaccines: us.csv:10: column series: series does not start at 1 in its cohort"},
		{"series gap", testFile(ok, "measles-3,MMR,measles,3,4y,,routine,,,"),
			"vaccines: us.csv:11: column series: series has a gap in its cohort"},
		{"bad from", testFile("measles-1,MMR,measles,1,12,16m,routine,,,"),
			"vaccines: us.csv:10: column from: not an age (<n><unit>, unit d, w, m or y)"},
		{"non-ASCII digit in an age", testFile("measles-1,MMR,measles,1,１2m,16m,routine,,,"),
			"vaccines: us.csv:10: column from: not an age (<n><unit>, unit d, w, m or y)"},
		{"bad until", testFile("measles-1,MMR,measles,1,12m,16 m,routine,,,"),
			"vaccines: us.csv:10: column until: not an age (<n><unit>, unit d, w, m or y)"},
		{"until not after from", testFile("measles-1,MMR,measles,1,12m,1y,routine,,,"),
			"vaccines: us.csv:10: column until: not after from"},
		{"until same unit not after", testFile("measles-1,MMR,measles,1,12m,12m,routine,,,"),
			"vaccines: us.csv:10: column until: not after from"},
		{"until 4w to 1m", testFile("measles-1,MMR,measles,1,4w,1m,routine,,,"),
			"vaccines: us.csv:10: column until: not after from"},
		{"until 365d to 1y", testFile("measles-1,MMR,measles,1,365d,1y,routine,,,"),
			"vaccines: us.csv:10: column until: not after from"},
		{"until 30d to 1m", testFile("measles-1,MMR,measles,1,30d,1m,routine,,,"),
			"vaccines: us.csv:10: column until: not after from"},
		{"malformed born_from", testFile("measles-1,MMR,measles,1,12m,16m,routine,2024-02-30,,"),
			"vaccines: us.csv:10: column born_from: not a YYYY-MM-DD date"},
		{"malformed born_to", testFile("measles-1,MMR,measles,1,12m,16m,routine,,2024-7-1,"),
			"vaccines: us.csv:10: column born_to: not a YYYY-MM-DD date"},
		{"born_from after born_to", testFile("measles-1,MMR,measles,1,12m,16m,routine,2024-07-02,2024-07-01,"),
			"vaccines: us.csv:10: column born_from: after born_to"},
		{"recurring with two lines", testFile("influenza-1,Flu,influenza,1,6m,,recurring,,,", "influenza-2,Flu,influenza,2,7m,,recurring,,,"),
			"vaccines: us.csv:11: column antigen: a recurring antigen has more than one line"},
		{"bad quoting", testFile(`measles-1,MMR,measles,1,12m,16m,routine,,,"open`),
			"vaccines: us.csv:10: column byte 49: extraneous or missing \" in quoted-field"},
	}
	for _, c := range cases {
		_, err := parseString(t, c.in)
		if err == nil || err.Error() != c.want {
			t.Errorf("%s:\n got %v\nwant %s", c.name, err, c.want)
		}
	}
}

func TestParseRefusesUnshippedID(t *testing.T) {
	mustLoad(t)
	in := strings.Replace(testFile("measles-1,MMR,measles,1,12m,16m,routine,,,"), "# schema: us", "# schema: ch", 1)
	_, err := parse("data/ch.csv", strings.NewReader(in))
	want := "vaccines: data/ch.csv:1: column schema: not a shipped schedule id"
	if err == nil || err.Error() != want {
		t.Errorf("err = %v, want %q", err, want)
	}
}

func TestParseErrorsNeverEchoValues(t *testing.T) {
	secret := "zz-SECRET-zz"
	for _, row := range []string{
		secret + ",MMR,measles,1,12m,16m,routine,,,",
		"measles-1,MMR," + secret + ",1,12m,16m,routine,,,",
		"measles-1,MMR,measles," + secret + ",12m,16m,routine,,,",
		"measles-1,MMR,measles,1," + secret + ",16m,routine,,,",
		"measles-1,MMR,measles,1,12m," + secret + ",routine,,,",
		"measles-1,MMR,measles,1,12m,16m," + secret + ",,,",
		"measles-1,MMR,measles,1,12m,16m,routine," + secret + ",,",
		"measles-1,MMR,measles,1,12m,16m,routine,,," + secret + ",extra",
		"measles-2," + secret + ",measles,1,12m,16m,routine,,,",
		"measles-1,MMR,measles,1,12m,16m,routine,,," + secret + ",",
		"measles-2,MMR,measles,1,12m,16m,routine,," + secret + ",",
	} {
		in := testFile("measles-1,MMR,measles,1,12m,16m,routine,,,", row)
		_, err := parseString(t, in)
		if err == nil {
			t.Errorf("row %q parsed", row)
			continue
		}
		if strings.Contains(err.Error(), "SECRET") {
			t.Errorf("error echoes a value: %v", err)
		}
	}
	for _, key := range []string{"name", "authority", "edition", "retrieved", "origin_url", "licence", "attribution", "schema"} {
		in := regexp.MustCompile(`(?m)^# `+key+`: .*$`).ReplaceAllString(testProvenance, "# "+key+": "+secret) + testHeader
		if _, err := parseString(t, in); err == nil || strings.Contains(err.Error(), "SECRET") {
			t.Errorf("provenance %s: err = %v", key, err)
		}
	}
	for _, in := range []string{
		testProvenance + "# " + secret + ": x\n" + testHeader,
		testProvenance + "line,group," + secret + ",series,from,until,kind,born_from,born_to,note\n",
	} {
		if _, err := parseString(t, in); err == nil || strings.Contains(err.Error(), "SECRET") {
			t.Errorf("err = %v", err)
		}
	}
}

func TestMixedUnitWindowsAreNonEmptyForEveryBirthDate(t *testing.T) {
	pairs := [][2]string{{"8w", "3m"}, {"6w", "2m"}, {"2m", "10w"}, {"11m", "1y"}, {"26d", "1m"}, {"1m", "35d"}, {"360d", "1y"}, {"1y", "53w"}, {"4y", "1470d"}}
	for _, p := range pairs {
		if _, err := parseString(t, testFile("measles-1,MMR,measles,1,"+p[0]+","+p[1]+",routine,,,")); err != nil {
			t.Fatalf("%s/%s refused: %v", p[0], p[1], err)
		}
		from, _ := ParseAge(p[0])
		until, _ := ParseAge(p[1])
		for born := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC); born.Year() <= 2028; born = born.AddDate(0, 0, 1) {
			if !addAge(born, until).After(addAge(born, from)) {
				t.Errorf("%s/%s born %s: until is not after from", p[0], p[1], born.Format(dateLayout))
			}
		}
	}
}

// addAge mirrors the calendar arithmetic of kidtime.AddAge (Go's AddDate).
func addAge(born time.Time, a Age) time.Time {
	switch a.Unit {
	case UnitWeek:
		return born.AddDate(0, 0, 7*a.N)
	case UnitMonth:
		return born.AddDate(0, a.N, 0)
	case UnitYear:
		return born.AddDate(a.N, 0, 0)
	}
	return born.AddDate(0, 0, a.N)
}
