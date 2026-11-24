package export

import (
	"bytes"
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/export/exporttest"
	"github.com/davcs86/happybabytracker/api/internal/export/pdfdoc"
	"github.com/davcs86/happybabytracker/api/internal/growth"
	"github.com/davcs86/happybabytracker/api/internal/kidtime"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/store"
	"github.com/davcs86/happybabytracker/api/internal/units"
)

// measured builds a stored row from typed metric values; 0 leaves a measure out.
func measured(t testing.TB, id, on string, grams, lengthMM, headMM int64) model.Growth {
	t.Helper()
	g := model.Growth{Entry: model.Entry{ID: id}, MeasuredOn: on}
	if grams > 0 {
		w, err := units.NewWeight(units.UnitKG, grams)
		if err != nil {
			t.Fatal(err)
		}
		g.Weight = &w
	}
	for _, c := range []struct {
		mm  int64
		dst **units.Length
	}{{lengthMM, &g.Length}, {headMM, &g.Head}} {
		if c.mm > 0 {
			l, err := units.NewLength(units.UnitCM, c.mm)
			if err != nil {
				t.Fatal(err)
			}
			*c.dst = &l
		}
	}
	return g
}

func growthKid(source, sex, birth string) model.Kid {
	return model.Kid{Name: "Ada", BirthDate: birth, Sex: sex, TimeZone: "UTC", GrowthSource: source}
}

func renderGrowth(t testing.TB, kid model.Kid, rows []model.Growth, u model.Units, now time.Time, to string) []byte {
	t.Helper()
	var buf bytes.Buffer
	req := Request{From: to, To: to, Modules: []model.Module{"growth"}, Units: u, Now: now, Loc: time.UTC}
	if err := Render(&buf, store.ExportSnapshot{Kid: kid, Growth: rows}, req); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func pdfText(t testing.TB, pdf []byte) []string {
	t.Helper()
	var out []string
	for _, p := range exporttest.PageTexts(t, pdf) {
		out = append(out, exporttest.Strings(p)...)
	}
	return out
}

// joined rebuilds wrapped lines: a cell or paragraph splits only at spaces.
func joined(strs []string) string { return strings.Join(strs, " ") }

func wantIn(t *testing.T, name, text string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(text, w) {
			t.Errorf("%s: %q missing", name, w)
		}
	}
}

func wantNotIn(t *testing.T, name, text string, banned ...string) {
	t.Helper()
	for _, b := range banned {
		if strings.Contains(text, b) {
			t.Errorf("%s: %q present", name, b)
		}
	}
}

func scoreText(s *growth.MeasureScore) string {
	return "z " + FormatZ(s.Z) + ", percentile " + FormatPercentile(s.Percentile)
}

func TestGrowthSectionTable(t *testing.T) {
	kid := growthKid("who", "female", "2026-08-01")
	rows := []model.Growth{measured(t, "b", "2026-10-29", 5000, 600, 400), measured(t, "a", "2026-08-11", 3400, 0, 0)}
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	text := joined(pdfText(t, renderGrowth(t, kid, rows, metric, now, "2026-11-20")))
	s, err := growth.ScoreRow(growth.SourceWHO, growth.Female, kidtime.AgeBasis{ChronologicalDays: 89, ScoringDays: 89}, &rows[0].Weight.Grams, &rows[0].Length.MM, &rows[0].Head.MM)
	if err != nil {
		t.Fatal(err)
	}
	wantIn(t, "who table", text, "Date Age Weight Length Head circumference BMI",
		"Thu 29 Oct 2026", "2 months 28 days", "5 kg: "+scoreText(s.Weight), "60 cm: "+scoreText(s.Length),
		"40 cm: "+scoreText(s.Head), "13.89: "+scoreText(s.BMI), "Tue 11 Aug 2026", "10 days", "3.4 kg: z ")
	if strings.Index(text, "Thu 29 Oct 2026") > strings.Index(text, "Tue 11 Aug 2026") {
		t.Error("rows are not newest first, as on screen")
	}

	imp := joined(pdfText(t, renderGrowth(t, kid, rows, imperial, now, "2026-11-20")))
	wantIn(t, "imperial", imp, "11 lb 0.4 oz: "+scoreText(s.Weight), "23.62 in: "+scoreText(s.Length), "15.75 in: ", "Weight (lb)", "Length (in)")
	wantNotIn(t, "imperial", imp, "5 kg", "60 cm", "Weight (kg)")

	upTo := joined(pdfText(t, renderGrowth(t, kid, rows, metric, now, "2026-10-28")))
	wantIn(t, "up to to", upTo, "Tue 11 Aug 2026")
	wantNotIn(t, "up to to", upTo, "Thu 29 Oct 2026", "5 kg")

	empty := joined(pdfText(t, renderGrowth(t, kid, nil, metric, now, "2026-11-20")))
	wantIn(t, "empty", empty, "Growth "+noMeasurements)
}

func TestGrowthAttribution(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	who, _ := growth.SourceInfo(growth.SourceWHO)
	cdc, _ := growth.SourceInfo(growth.SourceCDC)
	all := growth.AdaptationDisclaimers(growth.SourceWHO, []string{"wfa-day", "hcfa-day", "wfa-month"})
	if len(all) != 3 {
		t.Fatalf("disclaimers: %q", all)
	}

	// Weight, length and BMI from the 2006 standards and head from hcfa: not the 2007 reference.
	text := joined(pdfText(t, renderGrowth(t, growthKid("who", "female", "2026-08-01"),
		[]model.Growth{measured(t, "a", "2026-10-29", 5000, 600, 400)}, metric, now, "2026-11-20")))
	wantIn(t, "who", text, who.Attribution, "Licence: CC BY-NC-SA 3.0 IGO (https://creativecommons.org/licenses/by-nc-sa/3.0/igo/legalcode)", all[0], all[1])
	wantNotIn(t, "who", text, all[2], "convert.py", "Source files", cdc.Attribution)
	wantNoRepoText(t, "who", text)

	older := joined(pdfText(t, renderGrowth(t, growthKid("who", "male", "2020-06-01"),
		[]model.Growth{measured(t, "a", "2026-06-01", 20000, 1150, 510)}, metric, now, "2026-11-20")))
	wantIn(t, "who at 6 years", older, all[2])
	wantNotIn(t, "who at 6 years", older, all[0], all[1])
	wantNoRepoText(t, "who at 6 years", older)

	text = joined(pdfText(t, renderGrowth(t, growthKid("cdc", "female", "2026-08-01"),
		[]model.Growth{measured(t, "a", "2026-10-29", 5000, 600, 400)}, metric, now, "2026-11-20")))
	wantIn(t, "cdc", text, cdc.Attribution, "Licence: public domain (U.S. Government work, https://www.cdc.gov/other/agencymaterials.html)")
	wantNotIn(t, "cdc", text, "adaptation", who.Attribution)
	wantNoRepoText(t, "cdc", text)
}

// repoText is the licence lines' repository references as whole words, in any case; the CC URL's "licenses" is not one.
var repoText = regexp.MustCompile(`(?i)\b(not MIT|NOTICE|LICENSE|directory)\b`)

func wantNoRepoText(t *testing.T, name, text string) {
	t.Helper()
	if m := repoText.FindString(text); m != "" {
		t.Errorf("%s: %q present", name, m)
	}
}

func TestGrowthUnscoredWording(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	// Born at 35 weeks 2 days: 33 days early, so day 10 is before the expected term.
	pre := growthKid("who", "female", "2026-06-10")
	gw, gd := 35, 2
	pre.GestationWeeks, pre.GestationDays = &gw, &gd
	text := joined(pdfText(t, renderGrowth(t, pre, []model.Growth{measured(t, "a", "2026-06-20", 3000, 500, 340)}, metric, now, "2026-11-20")))
	wantIn(t, "before term", text, "10 days (before expected term)", "3 kg: before expected term", "50 cm: before expected term",
		"34 cm: before expected term", "12: before expected term")
	wantNotIn(t, "before term", text, "z ", "has no table")

	who, _ := growth.SourceInfo(growth.SourceWHO)
	cdc, _ := growth.SourceInfo(growth.SourceCDC)
	text = joined(pdfText(t, renderGrowth(t, growthKid("cdc", "female", "2024-11-20"),
		[]model.Growth{measured(t, "a", "2026-05-20", 11000, 810, 470)}, metric, now, "2026-11-20")))
	wantIn(t, "cdc bmi", text, "16.77: this source has no table for this age", "BMI "+cdc.Name+" has no table for this measure at this age")
	for _, m := range []string{"Weight", "Length", "Head circumference"} {
		wantNotIn(t, "cdc "+m, text, m+" "+cdc.Name+" has no table")
	}

	text = joined(pdfText(t, renderGrowth(t, growthKid("who", "male", "2020-06-01"),
		[]model.Growth{measured(t, "a", "2026-06-01", 20000, 1150, 510)}, metric, now, "2026-11-20")))
	wantIn(t, "who head", text, "51 cm: this source has no table for this age", "Head circumference "+who.Name+" has no table for this measure at this age")
}

func TestGrowth3000Rows(t *testing.T) {
	kid := growthKid("who", "female", "2018-01-01")
	birth := time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)
	rows := make([]model.Growth, constants.MaxGrowthPerKid)
	for i := range rows {
		day := len(rows) - i
		rows[i] = measured(t, strconv.Itoa(i), birth.AddDate(0, 0, day).Format(time.DateOnly), 3000+int64(day)*5, 500+int64(day)/4, 340+int64(day)/20)
	}
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	start := time.Now()
	pdf := renderGrowth(t, kid, rows, metric, now, "2026-11-20")
	elapsed := time.Since(start)
	t.Logf("%d rows: %d bytes, %v", len(rows), len(pdf), elapsed)
	if len(pdf) > constants.ExportMaxBytes || elapsed > constants.WriteTimeout {
		t.Errorf("%d bytes in %v, caps %d bytes and %v", len(pdf), elapsed, constants.ExportMaxBytes, constants.WriteTimeout)
	}
	strs := pdfText(t, pdf)
	dates := regexp.MustCompile(`^(Mon|Tue|Wed|Thu|Fri|Sat|Sun) \d{1,2} `)
	n := 0
	for _, s := range strs {
		if dates.MatchString(s) {
			n++
		}
	}
	if n != len(rows) {
		t.Errorf("%d dated rows listed, want %d", n, len(rows))
	}
	wantNotIn(t, "3000 rows", joined(strs), overflow, growthOverflow)
}

func TestGrowthRowBudget(t *testing.T) {
	savedSection, savedAll := maxRowsPerSection, maxRows
	maxRowsPerSection, maxRows = 3, 5
	t.Cleanup(func() { maxRowsPerSection, maxRows = savedSection, savedAll })
	kid := growthKid("who", "female", "2026-08-01")
	var rows []model.Growth
	for i := range 4 {
		rows = append(rows, measured(t, strconv.Itoa(i), "2026-10-2"+strconv.Itoa(4-i), 5000, 0, 0))
	}
	text := joined(pdfText(t, renderGrowth(t, kid, rows, metric, time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC), "2026-11-20")))
	wantIn(t, "budget", text, "Sat 24 Oct 2026", "Thu 22 Oct 2026",
		"1 more entries up to the end of the range are not shown. Export a range that ends earlier to see them.")
	wantNotIn(t, "budget", text, "Wed 21 Oct 2026", overflow)
}

func TestGrowthCorrectedAge(t *testing.T) {
	kid := growthKid("who", "female", "2024-06-10")
	gw, gd := 35, 2
	kid.GestationWeeks, kid.GestationDays = &gw, &gd
	rows := []model.Growth{measured(t, "a", "2024-09-10", 5000, 600, 0)}
	s, err := growth.ScoreRow(growth.SourceWHO, growth.Female, kidtime.AgeBasis{ChronologicalDays: 92, ScoringDays: 59, Corrected: true}, &rows[0].Weight.Grams, &rows[0].Length.MM, nil)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2024, 12, 1, 12, 0, 0, 0, time.UTC)
	text := joined(pdfText(t, renderGrowth(t, kid, rows, metric, now, "2024-12-01")))
	wantIn(t, "corrected", text, "3 months (corrected 1 month 28 days)", "5 kg: "+scoreText(s.Weight), "Corrected age (months)")

	// Past the second birthday the axis is chronological, and the earlier row still plots at corrected age.
	text = joined(pdfText(t, renderGrowth(t, kid, rows, metric, time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC), "2026-11-20")))
	wantIn(t, "later", text, "3 months (corrected 1 month 28 days)", "Age (months; corrected before the second birthday)")
}

func TestGrowthNoJudgement(t *testing.T) {
	deny := regexp.MustCompile(`(?i)\b(late|behind|delayed|delay|normal|abnormal|typical|atypical|on track|ahead|missed|missing|concern|worry|should|red flag|warning)\b`)
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	pre := growthKid("who", "female", "2026-06-10")
	gw, gd := 35, 2
	pre.GestationWeeks, pre.GestationDays = &gw, &gd
	rows := []model.Growth{measured(t, "c", "2026-10-29", 2000, 450, 300), measured(t, "b", "2026-09-29", 9000, 700, 460), measured(t, "a", "2026-06-20", 3000, 500, 340)}
	for _, kid := range []model.Kid{pre, growthKid("cdc", "male", "2024-11-20"), growthKid("who", "male", "2020-06-01")} {
		for _, u := range []model.Units{metric, imperial} {
			text := joined(pdfText(t, renderGrowth(t, kid, rows, u, now, "2026-11-20")))
			src := growth.Source(kid.GrowthSource)
			info, _ := growth.SourceInfo(src)
			exempt := append([]string{info.Attribution, licencePrefix + pdfLicence(info.Licence)}, growth.AdaptationDisclaimers(src, []string{"wfa-day", "hcfa-day", "wfa-month"})...)
			for _, e := range exempt {
				text = strings.ReplaceAll(text, e, "")
			}
			if m := deny.FindString(text); m != "" {
				t.Errorf("%s %v: %q judges", kid.GrowthSource, u.Weight, m)
			}
		}
	}
}

func TestChartGeometry(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	pre := growthKid("who", "female", "2024-06-10")
	gw, gd := 35, 2
	pre.GestationWeeks, pre.GestationDays = &gw, &gd
	// One row before term (unscored), one past the WHO length/height seam, one scored with every measure.
	rows := []model.Growth{measured(t, "c", "2026-08-01", 12000, 870, 470), measured(t, "b", "2025-01-10", 7000, 650, 420), measured(t, "a", "2024-06-20", 2500, 450, 320)}
	for _, u := range []model.Units{metric, imperial} {
		items, err := growthItems(store.ExportSnapshot{Kid: pre, Growth: rows}, Request{To: "2026-11-20", Units: u, Now: now, Loc: time.UTC}, &budget{left: maxRows})
		if err != nil {
			t.Fatal(err)
		}
		var charts []*pdfdoc.Chart
		for _, it := range items {
			if it.chart != nil {
				charts = append(charts, it.chart)
			}
		}
		if len(charts) != 4 {
			t.Fatalf("%v: %d charts, want one per measure", u.Weight, len(charts))
		}
		for _, c := range charts {
			box := func(what string, p pdfdoc.Point) {
				if math.IsNaN(p.X) || math.IsNaN(p.Y) || p.X < c.XMin || p.X > c.XMax || p.Y < c.YMin || p.Y > c.YMax {
					t.Errorf("%s %s %v outside [%v,%v]×[%v,%v]", c.Title, what, p, c.XMin, c.XMax, c.YMin, c.YMax)
				}
			}
			if len(c.Points) != 2 {
				t.Errorf("%s: %d points, want the two scored rows only", c.Title, len(c.Points))
			}
			for _, p := range c.Points {
				box("point", p)
				if p.X < 0 {
					t.Errorf("%s: point before term plotted at %v", c.Title, p.X)
				}
			}
			if len(c.Curves) == 0 {
				t.Errorf("%s: no curves", c.Title)
			}
			for _, cv := range c.Curves {
				if len(cv.Points) < 2 {
					t.Errorf("%s %q: a curve of %d points", c.Title, cv.Label, len(cv.Points))
				}
				for _, p := range cv.Points {
					box("curve "+cv.Label, p)
				}
			}
		}
	}

	pdf := renderGrowth(t, pre, rows, metric, now, "2026-11-20")
	if n := bytes.Count(pdf, []byte(" re f")); n != 4*2 {
		t.Errorf("%d squares drawn, want two per measure", n)
	}
	text := joined(pdfText(t, pdf))
	wantIn(t, "labels", text, "Weight (kg)", "Length (cm)", "Head circumference (cm)", "BMI (kg/m²)", "3rd", "15th", "50th", "85th", "97th")
	for _, op := range regexp.MustCompile(`([\d.]+) ([\d.]+) ([\d.]+) (RG|rg)\b`).FindAllSubmatch(pdf, -1) {
		if string(op[1]) != string(op[2]) || string(op[2]) != string(op[3]) {
			t.Errorf("colour %s is not grey or black", op[0])
		}
	}
}

func TestChartSinglePoint(t *testing.T) {
	for _, on := range []string{"2026-08-01", "2026-10-29"} {
		items, err := growthItems(store.ExportSnapshot{Kid: growthKid("who", "female", "2026-08-01"), Growth: []model.Growth{measured(t, "a", on, 5000, 0, 0)}},
			Request{To: "2026-11-20", Units: metric, Now: time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC), Loc: time.UTC}, &budget{left: maxRows})
		if err != nil {
			t.Fatal(err)
		}
		var charts int
		for _, it := range items {
			if c := it.chart; c != nil {
				charts++
				if !(c.XMax > c.XMin) || !(c.YMax > c.YMin) || c.XMin < 0 || len(c.Points) != 1 {
					t.Errorf("%s: range [%v,%v]×[%v,%v] with %d points", on, c.XMin, c.XMax, c.YMin, c.YMax, len(c.Points))
				}
			}
		}
		if charts != 1 {
			t.Errorf("%s: %d charts, want weight only", on, charts)
		}
	}
}
