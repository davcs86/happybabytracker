// Package vaccines loads the bundled vaccination schedules and the antigen and product catalogue.
// Line ids are <antigen>-<series>, plus a cohort suffix only where two lines share one (diphtheria-4-b2024).
package vaccines

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

// Kind says how a line may be flagged.
type Kind string

const (
	KindRoutine        Kind = "routine"
	KindMayNotBeNeeded Kind = "may_not_be_needed"
	KindSupplementary  Kind = "supplementary"
	KindRecurring      Kind = "recurring"
)

// Line is one recommended dose of one antigen. Until nil means the default window; BornFrom and
// BornTo are inclusive YYYY-MM-DD dates, "" for unbounded.
type Line struct {
	ID, Group, Antigen string
	Series             int
	From               Age
	Until              *Age
	Kind               Kind
	BornFrom, BornTo   string
	Note               string
}

// Schema is one schedule file: its provenance and its lines in file order.
type Schema struct {
	ID, Name, Authority, Edition, Retrieved, OriginURL, Licence, Attribution string
	Lines                                                                    []Line
}

// Every loader error has this shape and never carries a field's value.
const errFormat = "vaccines: %s:%d: column %s: %s"

const (
	reasonEmpty          = "is empty"
	reasonHeader         = "header is not "
	reasonNoLines        = "no lines"
	reasonDuplicateID    = "duplicate id"
	reasonUnknownAntigen = "unknown antigen"
	reasonDupAntigen     = "duplicate antigen"
	reasonProvLine       = "not a provenance line"
	reasonUnknownKey     = "unknown provenance key"
	reasonDuplicateKey   = "duplicate provenance key"
	reasonMissingKey     = "missing provenance key"
	reasonSchemaMismatch = "does not match the file name"
	reasonUnshipped      = "not a shipped schedule id"
	reasonDate           = "not a YYYY-MM-DD date"
	reasonURL            = "not an https URL"
	reasonDuplicateLine  = "duplicate line id"
	reasonKind           = "unknown kind"
	reasonSeries         = "not a positive integer"
	reasonAge            = "not an age (<n><unit>, unit d, w, m or y)"
	reasonUntil          = "not after from"
	reasonCohort         = "after born_to"
	reasonRecurring      = "a recurring antigen has more than one line"
	reasonSeriesStart    = "series does not start at 1 in its cohort"
	reasonSeriesGap      = "series has a gap in its cohort"
	reasonSeriesRepeat   = "series repeats in an overlapping cohort"
)

const dateLayout = "2006-01-02"

var provenanceKeys = []string{"schema", "name", "authority", "edition", "retrieved", "origin_url", "licence", "attribution"}

var schemaHeader = []string{"line", "group", "antigen", "series", "from", "until", "kind", "born_from", "born_to", "note"}

var kinds = []Kind{KindRoutine, KindMayNotBeNeeded, KindSupplementary, KindRecurring}

func fail(file string, line int, column, reason string) error {
	return fmt.Errorf(errFormat, file, line, column, reason)
}

// parse reads one schedule file; antigens are checked against the loaded catalogue.
func parse(name string, r io.Reader) (Schema, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return Schema{}, err
	}
	id := strings.TrimSuffix(path.Base(name), ".csv")
	meta := map[string]string{}
	n, header, found := 0, "", false
	for len(data) > 0 {
		var l string
		l, data = cutLine(data)
		n++
		if !strings.HasPrefix(l, "#") {
			header, found = l, true
			break
		}
		if err := provenance(name, n, l, id, meta); err != nil {
			return Schema{}, err
		}
	}
	if !found {
		n++
	}
	if err := checkHeader(name, n, header, found, schemaHeader); err != nil {
		return Schema{}, err
	}
	for _, k := range provenanceKeys {
		if _, ok := meta[k]; !ok {
			return Schema{}, fail(name, n, k, reasonMissingKey)
		}
	}
	rows, err := readRows(name, data, n, len(schemaHeader))
	if err != nil {
		return Schema{}, err
	}
	if len(rows) == 0 {
		return Schema{}, fail(name, n, schemaHeader[0], reasonNoLines)
	}
	s := Schema{
		ID: id, Name: meta["name"], Authority: meta["authority"], Edition: meta["edition"],
		Retrieved: meta["retrieved"], OriginURL: meta["origin_url"], Licence: meta["licence"],
		Attribution: meta["attribution"],
	}
	seen := map[string]bool{}
	for _, row := range rows {
		l, err := parseLine(name, row)
		if err != nil {
			return Schema{}, err
		}
		if seen[l.ID] {
			return Schema{}, fail(name, row.line, "line", reasonDuplicateLine)
		}
		seen[l.ID] = true
		s.Lines = append(s.Lines, l)
	}
	if err := checkSeries(name, s.Lines, rows); err != nil {
		return Schema{}, err
	}
	return s, nil
}

func provenance(name string, n int, l, id string, meta map[string]string) error {
	k, v, ok := strings.Cut(strings.TrimPrefix(l, "# "), ": ")
	if !strings.HasPrefix(l, "# ") || !ok || k == "" {
		return fail(name, n, "key", reasonProvLine)
	}
	if !slices.Contains(provenanceKeys, k) {
		return fail(name, n, "key", reasonUnknownKey)
	}
	if _, dup := meta[k]; dup {
		return fail(name, n, k, reasonDuplicateKey)
	}
	v = strings.TrimSpace(v)
	switch {
	case v == "":
		return fail(name, n, k, reasonEmpty)
	case k == "schema" && v != id:
		return fail(name, n, k, reasonSchemaMismatch)
	case k == "schema" && !slices.Contains(constants.VaccineSchedules, id):
		return fail(name, n, k, reasonUnshipped)
	case k == "retrieved" && !isDate(v):
		return fail(name, n, k, reasonDate)
	case k == "origin_url" && !isHTTPS(v):
		return fail(name, n, k, reasonURL)
	}
	meta[k] = v
	return nil
}

func parseLine(name string, row csvRow) (Line, error) {
	f := row.fields
	col := func(i int, reason string) error { return fail(name, row.line, schemaHeader[i], reason) }
	l := Line{ID: f[0], Group: f[1], Antigen: f[2], Kind: Kind(f[6]), BornFrom: f[7], BornTo: f[8], Note: f[9]}
	switch {
	case l.ID == "":
		return Line{}, col(0, reasonEmpty)
	case l.Group == "":
		return Line{}, col(1, reasonEmpty)
	case !antigenIdx[l.Antigen]:
		return Line{}, col(2, reasonUnknownAntigen)
	}
	series, ok := positiveInt(f[3])
	if !ok {
		return Line{}, col(3, reasonSeries)
	}
	l.Series = series
	from, err := ParseAge(f[4])
	if err != nil {
		return Line{}, col(4, reasonAge)
	}
	l.From = from
	if f[5] != "" {
		until, err := ParseAge(f[5])
		if err != nil {
			return Line{}, col(5, reasonAge)
		}
		if !after(until, from) {
			return Line{}, col(5, reasonUntil)
		}
		l.Until = &until
	}
	if !slices.Contains(kinds, l.Kind) {
		return Line{}, col(6, reasonKind)
	}
	for i, d := range []string{l.BornFrom, l.BornTo} {
		if d != "" && !isDate(d) {
			return Line{}, col(7+i, reasonDate)
		}
	}
	if l.BornFrom != "" && l.BornTo != "" && l.BornFrom > l.BornTo {
		return Line{}, col(7, reasonCohort)
	}
	return l, nil
}

// checkSeries requires, for every birth date, each antigen's applicable series to be exactly 1..k.
func checkSeries(name string, lines []Line, rows []csvRow) error {
	var order []string
	byAntigen := map[string][]int{}
	for i, l := range lines {
		if byAntigen[l.Antigen] == nil {
			order = append(order, l.Antigen)
		}
		byAntigen[l.Antigen] = append(byAntigen[l.Antigen], i)
	}
	for _, a := range order {
		idx := byAntigen[a]
		if len(idx) > 1 && slices.ContainsFunc(idx, func(i int) bool { return lines[i].Kind == KindRecurring }) {
			return fail(name, rows[idx[1]].line, "antigen", reasonRecurring)
		}
		for _, p := range cohortPoints(lines, idx) {
			var active []int
			for _, i := range idx {
				if inCohort(lines[i], p) {
					active = append(active, i)
				}
			}
			sort.SliceStable(active, func(x, y int) bool { return lines[active[x]].Series < lines[active[y]].Series })
			for k, i := range active {
				reason := ""
				switch {
				case k == 0 && lines[i].Series != 1:
					reason = reasonSeriesStart
				case k > 0 && lines[i].Series == lines[active[k-1]].Series:
					reason = reasonSeriesRepeat
				case k > 0 && lines[i].Series != lines[active[k-1]].Series+1:
					reason = reasonSeriesGap
				}
				if reason != "" {
					return fail(name, rows[i].line, "series", reason)
				}
			}
		}
	}
	return nil
}

// cohortPoints are the birth dates where an antigen's applicable lines can change; the zero time
// stands for every date before the first.
func cohortPoints(lines []Line, idx []int) []time.Time {
	points := []time.Time{{}}
	for _, i := range idx {
		if t, err := time.Parse(dateLayout, lines[i].BornFrom); err == nil {
			points = append(points, t)
		}
		if t, err := time.Parse(dateLayout, lines[i].BornTo); err == nil {
			points = append(points, t.AddDate(0, 0, 1))
		}
	}
	return points
}

func inCohort(l Line, born time.Time) bool {
	d := born.Format(dateLayout)
	if born.IsZero() {
		return l.BornFrom == ""
	}
	return (l.BornFrom == "" || l.BornFrom <= d) && (l.BornTo == "" || d <= l.BornTo)
}

func after(until, from Age) bool {
	if until.Unit == from.Unit {
		return until.N > from.N
	}
	return until.approxDays()-from.approxDays() >= constants.VaccineMinMixedUnitGapDays
}

func isDate(s string) bool {
	_, err := time.Parse(dateLayout, s)
	return len(s) == len(dateLayout) && err == nil
}

func isHTTPS(s string) bool {
	u, err := url.Parse(s)
	return err == nil && u.Scheme == "https" && u.Host != "" && strings.HasPrefix(s, "https://")
}

func positiveInt(s string) (int, bool) {
	if s == "" || s[0] == '0' {
		return 0, false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, false
		}
	}
	n, err := strconv.Atoi(s)
	return n, err == nil
}

func cutLine(data []byte) (string, []byte) {
	l, rest, _ := bytes.Cut(data, []byte("\n"))
	return string(l), rest
}

func checkHeader(name string, n int, got string, found bool, want []string) error {
	fields := strings.Split(got, ",")
	for i, w := range want {
		if !found || i >= len(fields) || fields[i] != w {
			return fail(name, n, w, reasonHeader+strings.Join(want, ","))
		}
	}
	if len(fields) != len(want) {
		return fail(name, n, want[len(want)-1], reasonHeader+strings.Join(want, ","))
	}
	return nil
}

type csvRow struct {
	line   int
	fields []string
}

// readRows reads the CSV records after line offset of the file, numbering lines from the file's start.
func readRows(name string, data []byte, offset, fields int) ([]csvRow, error) {
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = fields
	var rows []csvRow
	for {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			return rows, nil
		}
		var pe *csv.ParseError
		if errors.As(err, &pe) {
			column := "byte " + strconv.Itoa(pe.Column)
			if errors.Is(pe.Err, csv.ErrFieldCount) {
				column = "record"
			}
			return nil, fail(name, pe.Line+offset, column, pe.Err.Error())
		}
		if err != nil {
			return nil, err
		}
		line, _ := r.FieldPos(0)
		rows = append(rows, csvRow{line + offset, rec})
	}
}
