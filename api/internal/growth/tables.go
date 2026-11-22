// Package growth scores measurements and draws percentile curves from the bundled WHO and CDC LMS tables.
package growth

import (
	"bufio"
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

// Source is a reference data set offered to a kid.
type Source string

// Measure is what a measurement records.
type Measure string

// Sex selects the table of a source.
type Sex string

const (
	SourceWHO Source = "who"
	SourceCDC Source = "cdc"

	Weight Measure = "weight"
	Length Measure = "length"
	Head   Measure = "head"
	BMI    Measure = "bmi"

	Female Sex = "female"
	Male   Sex = "male"
)

// Sources lists the offered sources, in order.
var Sources = []Source{SourceWHO, SourceCDC}

// ErrNotCovered: no table row covers this source, measure, sex and age.
var ErrNotCovered = errors.New("growth: not covered by the reference")

// whoLengthLastDay is the last row of the length region of lhfa-day and bfa-day; height starts the day after.
const whoLengthLastDay = 730

//go:embed data/*/*.csv data/*/NOTICE data/who/LICENSE
var dataFS embed.FS

type row struct{ age, l, m, s float64 }

type table struct {
	name string
	rows []row
}

type tableKey struct {
	src  Source
	name string
	sex  Sex
}

// tablePair names a measure's tables: WHO 2006 day and 2007 month, CDC infant and child; "" when absent.
type tablePair struct{ first, second string }

var tableNames = map[Source]map[Measure]tablePair{
	SourceWHO: {
		Weight: {"wfa-day", "wfa-month"},
		Length: {"lhfa-day", "hfa-month"},
		Head:   {"hcfa-day", ""},
		BMI:    {"bfa-day", "bfa-month"},
	},
	SourceCDC: {
		Weight: {"wtageinf", "wtage"},
		Length: {"lenageinf", "statage"},
		Head:   {"hcageinf", ""},
		BMI:    {"", "bmiagerev"},
	},
}

var (
	once   sync.Once
	tables map[tableKey]*table
	infos  map[Source]Info
)

func loadOnce() { once.Do(load) }

// load panics on a malformed embedded file: the data ships with the binary and data_test.go checks it.
func load() {
	tables = map[tableKey]*table{}
	infos = map[Source]Info{}
	for _, src := range Sources {
		paths, err := fs.Glob(dataFS, path.Join("data", string(src), "*.csv"))
		if err != nil || len(paths) == 0 {
			panic(fmt.Sprintf("growth: no tables for %s: %v", src, err))
		}
		sort.Strings(paths)
		for i, p := range paths {
			meta, t := parseTable(p)
			if src == SourceWHO && strings.HasSuffix(t.name, "-day") {
				for d, r := range t.rows {
					if r.age != float64(d) {
						panic(fmt.Sprintf("growth: %s: row %d is not day %d", p, d, d))
					}
				}
			}
			tables[tableKey{src, meta["table"], Sex(meta["sex"])}] = t
			if i == 0 {
				infos[src] = newInfo(src, meta)
			}
		}
	}
}

func parseTable(p string) (map[string]string, *table) {
	b, err := dataFS.ReadFile(p)
	if err != nil {
		panic(fmt.Sprintf("growth: %s: %v", p, err))
	}
	meta := map[string]string{}
	t := &table{}
	sc := bufio.NewScanner(bytes.NewReader(b))
	header := false
	for sc.Scan() {
		line := sc.Text()
		switch {
		case !header && strings.HasPrefix(line, "# "):
			k, v, _ := strings.Cut(strings.TrimPrefix(line, "# "), ": ")
			meta[k] = v
		case !header:
			header = line == "age,l,m,s"
		default:
			var v [4]float64
			cols := strings.Split(line, ",")
			if len(cols) != len(v) {
				panic(fmt.Sprintf("growth: %s: bad row", p))
			}
			for i, c := range cols {
				if v[i], err = strconv.ParseFloat(c, 64); err != nil {
					panic(fmt.Sprintf("growth: %s: %v", p, err))
				}
			}
			t.rows = append(t.rows, row{v[0], v[1], v[2], v[3]})
		}
	}
	if !header || len(t.rows) == 0 || meta["table"] == "" || meta["sex"] == "" {
		panic(fmt.Sprintf("growth: %s: no header or rows", p))
	}
	t.name = meta["table"]
	return meta, t
}

func tableFor(src Source, name string, sex Sex) *table {
	loadOnce()
	if name == "" {
		return nil
	}
	return tables[tableKey{src, name, sex}]
}

func pairFor(src Source, m Measure, sex Sex) (first, second *table) {
	p := tableNames[src][m]
	return tableFor(src, p.first, sex), tableFor(src, p.second, sex)
}

// hit is the L, M, S used for one age; region changes where the curve must not be joined.
type hit struct {
	l, m, s float64
	table   string
	region  string
}

// whoLastDay is the last day row of the WHO 2006 table of m, or -1.
func whoLastDay(m Measure, sex Sex) int {
	day, _ := pairFor(SourceWHO, m, sex)
	if day == nil {
		return -1
	}
	return int(day.rows[len(day.rows)-1].age)
}

// lookupDays: WHO 2006 rows by exact day, everything else by days / DaysPerMonth.
func lookupDays(src Source, m Measure, sex Sex, days int) (hit, error) {
	if days < 0 {
		return hit{}, ErrNotCovered
	}
	if src == SourceWHO && days <= whoLastDay(m, sex) {
		day, _ := pairFor(src, m, sex)
		r := day.rows[days]
		region := day.name
		if (m == Length || m == BMI) && days > whoLengthLastDay {
			region += "/height"
		}
		return hit{r.l, r.m, r.s, day.name, region}, nil
	}
	return lookupMonths(src, m, sex, float64(days)/constants.DaysPerMonth)
}

// lookupMonths reads the WHO 2007 or a CDC table, interpolating L, M, S between the bracketing rows.
func lookupMonths(src Source, m Measure, sex Sex, months float64) (hit, error) {
	first, second := pairFor(src, m, sex)
	var t *table
	switch src {
	case SourceWHO:
		t = second
	case SourceCDC:
		t = first
		if second != nil && (first == nil || months >= second.rows[0].age) {
			t = second
		}
	}
	if t == nil {
		return hit{}, ErrNotCovered
	}
	h, ok := interpolate(t, months)
	if !ok {
		return hit{}, ErrNotCovered
	}
	return h, nil
}

func interpolate(t *table, months float64) (hit, bool) {
	rows := t.rows
	if math.IsNaN(months) || months < rows[0].age || months > rows[len(rows)-1].age {
		return hit{}, false
	}
	i := sort.Search(len(rows), func(i int) bool { return rows[i].age >= months })
	if rows[i].age == months {
		r := rows[i]
		return hit{r.l, r.m, r.s, t.name, t.name}, true
	}
	lo, hi := rows[i-1], rows[i]
	f := (months - lo.age) / (hi.age - lo.age)
	lerp := func(a, b float64) float64 { return a + (b-a)*f }
	return hit{lerp(lo.l, hi.l), lerp(lo.m, hi.m), lerp(lo.s, hi.s), t.name, t.name}, true
}

// coverage is the age range in months a source covers for m and sex.
func coverage(src Source, m Measure, sex Sex) (from, to float64, ok bool) {
	first, second := pairFor(src, m, sex)
	scale := 1.0
	if src == SourceWHO {
		scale = 1 / constants.DaysPerMonth
	}
	from, to = math.Inf(1), math.Inf(-1)
	if first != nil {
		from, to = first.rows[0].age*scale, first.rows[len(first.rows)-1].age*scale
	}
	if second != nil {
		from, to = min(from, second.rows[0].age), max(to, second.rows[len(second.rows)-1].age)
	}
	return from, to, first != nil || second != nil
}
