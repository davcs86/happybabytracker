package growth

import (
	"bufio"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

type tableFile struct {
	meta map[string]string
	ages []float64
	lms  [][3]float64
}

var provenanceKeys = []string{
	"source", "table", "sex", "origin_url", "source_sha256",
	"version", "retrieved", "licence", "attribution", "adaptation",
}

type tableSpec struct {
	rows        int
	first, last float64
}

var wantTables = map[string]map[string]tableSpec{
	"who": {
		"wfa-day":   {1857, 0, 1856},
		"lhfa-day":  {1857, 0, 1856},
		"hcfa-day":  {1857, 0, 1856},
		"bfa-day":   {1857, 0, 1856},
		"wfa-month": {60, 61, 120},
		"hfa-month": {168, 61, 228},
		"bfa-month": {168, 61, 228},
	},
	"cdc": {
		"wtageinf":  {38, 0, 36},
		"lenageinf": {37, 0, 35.5},
		"hcageinf":  {38, 0, 36},
		"wtage":     {218, 24, 240},
		"statage":   {218, 24, 240},
		"bmiagerev": {219, 24, 240.5},
	},
}

var sexes = []string{"female", "male"}

func readTable(t *testing.T, path string) tableFile {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	tf := tableFile{meta: map[string]string{}}
	sawHeader := false
	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		line := sc.Text()
		switch {
		case !sawHeader && strings.HasPrefix(line, "# "):
			k, v, ok := strings.Cut(strings.TrimPrefix(line, "# "), ": ")
			if !ok {
				t.Fatalf("%s:%d: comment is not `# key: value`", path, n)
			}
			tf.meta[k] = v
		case !sawHeader:
			if line != "age,l,m,s" {
				t.Fatalf("%s:%d: header %q, want age,l,m,s", path, n, line)
			}
			sawHeader = true
		default:
			cols := strings.Split(line, ",")
			if len(cols) != 4 {
				t.Fatalf("%s:%d: %d columns, want 4", path, n, len(cols))
			}
			var v [4]float64
			for i, c := range cols {
				if v[i], err = strconv.ParseFloat(c, 64); err != nil {
					t.Fatalf("%s:%d: column %d: %v", path, n, i, err)
				}
			}
			tf.ages = append(tf.ages, v[0])
			tf.lms = append(tf.lms, [3]float64{v[1], v[2], v[3]})
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !sawHeader {
		t.Fatalf("%s: no header row", path)
	}
	return tf
}

func allTables(t *testing.T) map[string]tableFile {
	t.Helper()
	out := map[string]tableFile{}
	for src := range wantTables {
		paths, err := filepath.Glob(filepath.Join("data", src, "*.csv"))
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range paths {
			out[filepath.ToSlash(p)] = readTable(t, p)
		}
	}
	if len(out) == 0 {
		t.Fatal("no table files under data/")
	}
	return out
}

func TestExpectedFilesExist(t *testing.T) {
	var want, got []string
	for src, tables := range wantTables {
		for table := range tables {
			for _, sex := range sexes {
				want = append(want, src+"/"+table+"-"+sex+".csv")
			}
		}
		paths, err := filepath.Glob(filepath.Join("data", src, "*.csv"))
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range paths {
			got = append(got, src+"/"+filepath.Base(p))
		}
	}
	sort.Strings(want)
	sort.Strings(got)
	if len(want) != 26 {
		t.Fatalf("test expects %d files, want 26", len(want))
	}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("files\n got %v\nwant %v", got, want)
	}
}

func TestEveryTableFileHasProvenance(t *testing.T) {
	hex64 := regexp.MustCompile(`^[0-9a-f]{64}$`)
	for path, tf := range allTables(t) {
		for _, k := range provenanceKeys {
			if strings.TrimSpace(tf.meta[k]) == "" {
				t.Errorf("%s: missing %q", path, k)
			}
		}
		if !strings.HasPrefix(tf.meta["origin_url"], "https://") {
			t.Errorf("%s: origin_url %q is not https", path, tf.meta["origin_url"])
		}
		if !hex64.MatchString(tf.meta["source_sha256"]) {
			t.Errorf("%s: source_sha256 %q is not 64 hex", path, tf.meta["source_sha256"])
		}
		base := strings.TrimSuffix(filepath.Base(path), ".csv")
		if want := tf.meta["table"] + "-" + tf.meta["sex"]; base != want {
			t.Errorf("%s: table/sex headers name %q", path, want)
		}
		if tf.meta["retrieved"] != "2026-11-22" {
			t.Errorf("%s: retrieved %q", path, tf.meta["retrieved"])
		}
	}
}

func TestRowCounts(t *testing.T) {
	for src, tables := range wantTables {
		for table, spec := range tables {
			for _, sex := range sexes {
				path := filepath.Join("data", src, table+"-"+sex+".csv")
				tf := readTable(t, path)
				if len(tf.ages) != spec.rows {
					t.Errorf("%s: %d rows, want %d", path, len(tf.ages), spec.rows)
					continue
				}
				if tf.ages[0] != spec.first || tf.ages[len(tf.ages)-1] != spec.last {
					t.Errorf("%s: ages %v..%v, want %v..%v", path,
						tf.ages[0], tf.ages[len(tf.ages)-1], spec.first, spec.last)
				}
			}
		}
	}
}

func TestAgesStrictlyIncreasing(t *testing.T) {
	for path, tf := range allTables(t) {
		for i := 1; i < len(tf.ages); i++ {
			if tf.ages[i] <= tf.ages[i-1] {
				t.Errorf("%s: row %d age %v after %v", path, i+1, tf.ages[i], tf.ages[i-1])
			}
		}
		if strings.HasPrefix(path, "data/who/") {
			for i, a := range tf.ages {
				if a != math.Trunc(a) {
					t.Errorf("%s: row %d age %v is not whole", path, i+1, a)
				}
			}
		}
	}
}

func TestLMSAreFinite(t *testing.T) {
	for path, tf := range allTables(t) {
		for i, r := range tf.lms {
			for _, v := range r {
				if math.IsNaN(v) || math.IsInf(v, 0) {
					t.Errorf("%s: row %d not finite: %v", path, i+1, r)
				}
			}
			if r[1] <= 0 || r[2] <= 0 {
				t.Errorf("%s: row %d needs m > 0 and s > 0: %v", path, i+1, r)
			}
		}
	}
}

func TestNoticeFilesNameTheirLicence(t *testing.T) {
	cases := []struct {
		path string
		want []string
	}{
		{"data/who/NOTICE", []string{"CC BY-NC-SA 3.0 IGO", "non-commercial"}},
		{"data/who/LICENSE", []string{"CC BY-NC-SA 3.0 IGO", "https://creativecommons.org/licenses/by-nc-sa/3.0/igo/legalcode"}},
		{"data/cdc/NOTICE", []string{"Source: CDC"}},
	}
	for _, c := range cases {
		b, err := os.ReadFile(c.path)
		if err != nil {
			t.Errorf("%s: %v", c.path, err)
			continue
		}
		for _, w := range c.want {
			if !strings.Contains(string(b), w) {
				t.Errorf("%s: does not contain %q", c.path, w)
			}
		}
	}
}
