package vaccines

import (
	"bytes"
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"testing"
	"time"
)

// mandatoryAttribution is the wording each schedule's licence requires in its attribution.
var mandatoryAttribution = map[string][]string{
	"us": {"Source: CDC", "CDC does not endorse this application or its use of the material.",
		"available free of charge on the CDC website at https://", "must not be read as CDC's own text"},
	"uk": {"Source: UKHSA", "Contains public sector information licensed under the Open Government Licence v3.0",
		"https://www.gov.uk/government/publications/the-complete-routine-immunisation-schedule",
		"https://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/",
		"partial, simplified transcription", "not endorsed by UKHSA"},
}

func TestShippedSchedulesLoad(t *testing.T) {
	mustLoad(t)
	paths, err := fs.Glob(dataFS, "data/*.csv")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, p := range paths {
		if p == antigensFile || p == productsFile {
			continue
		}
		b, err := fs.ReadFile(dataFS, p)
		if err != nil {
			t.Fatal(err)
		}
		s, err := parse(p, bytes.NewReader(b))
		if err != nil {
			t.Errorf("parse(%s) = %v", p, err)
			continue
		}
		seen[s.ID] = true
		for k, v := range map[string]string{"name": s.Name, "authority": s.Authority, "edition": s.Edition,
			"retrieved": s.Retrieved, "origin_url": s.OriginURL, "licence": s.Licence, "attribution": s.Attribution} {
			if strings.TrimSpace(v) == "" {
				t.Errorf("%s: %s is empty", p, k)
			}
		}
		if !strings.HasPrefix(s.OriginURL, "https://") {
			t.Errorf("%s: origin_url does not start with https://", p)
		}
		if _, err := time.Parse(dateLayout, s.Retrieved); err != nil {
			t.Errorf("%s: retrieved %q is not a date", p, s.Retrieved)
		}
		phrases, ok := mandatoryAttribution[s.ID]
		if !ok {
			t.Errorf("%s: no mandatory attribution wording is pinned for this schedule", p)
		}
		for _, ph := range phrases {
			if !strings.Contains(s.Attribution, ph) {
				t.Errorf("%s: attribution lacks %q", p, ph)
			}
		}
		if g, ok := GetSchema(s.ID); !ok || len(g.Lines) != len(s.Lines) {
			t.Errorf("GetSchema(%s) does not serve the file", s.ID)
		}
	}
	for _, id := range []string{"us", "uk"} {
		if !seen[id] {
			t.Errorf("data/%s.csv is not shipped", id)
		}
	}
}

type pinned struct {
	id, group, from, until string
	kind                   Kind
}

// usLines is every line of us.csv; a change to the file must change this table.
var usLines = []pinned{
	{"rsv-1", "RSV", "0d", "8m", KindMayNotBeNeeded},
	{"rsv-2", "RSV", "8m", "20m", KindSupplementary},
	{"hepatitis_b-1", "Hepatitis B", "0d", "", KindRoutine},
	{"hepatitis_b-2", "Hepatitis B", "1m", "3m", KindRoutine},
	{"hepatitis_b-3", "Hepatitis B", "6m", "19m", KindRoutine},
	{"rotavirus-1", "Rotavirus", "2m", "", KindRoutine},
	{"rotavirus-2", "Rotavirus", "4m", "", KindRoutine},
	{"rotavirus-3", "Rotavirus", "6m", "", KindMayNotBeNeeded},
	{"diphtheria-1", "DTaP", "2m", "", KindRoutine},
	{"tetanus-1", "DTaP", "2m", "", KindRoutine},
	{"pertussis-1", "DTaP", "2m", "", KindRoutine},
	{"diphtheria-2", "DTaP", "4m", "", KindRoutine},
	{"tetanus-2", "DTaP", "4m", "", KindRoutine},
	{"pertussis-2", "DTaP", "4m", "", KindRoutine},
	{"diphtheria-3", "DTaP", "6m", "", KindRoutine},
	{"tetanus-3", "DTaP", "6m", "", KindRoutine},
	{"pertussis-3", "DTaP", "6m", "", KindRoutine},
	{"diphtheria-4", "DTaP", "15m", "19m", KindRoutine},
	{"tetanus-4", "DTaP", "15m", "19m", KindRoutine},
	{"pertussis-4", "DTaP", "15m", "19m", KindRoutine},
	{"diphtheria-5", "DTaP", "4y", "7y", KindRoutine},
	{"tetanus-5", "DTaP", "4y", "7y", KindRoutine},
	{"pertussis-5", "DTaP", "4y", "7y", KindRoutine},
	{"hib-1", "Hib", "2m", "", KindRoutine},
	{"hib-2", "Hib", "4m", "", KindRoutine},
	{"hib-3", "Hib", "6m", "16m", KindRoutine},
	{"hib-4", "Hib", "12m", "16m", KindMayNotBeNeeded},
	{"pneumococcus-1", "PCV", "2m", "", KindRoutine},
	{"pneumococcus-2", "PCV", "4m", "", KindRoutine},
	{"pneumococcus-3", "PCV", "6m", "", KindRoutine},
	{"pneumococcus-4", "PCV", "12m", "16m", KindRoutine},
	{"polio-1", "IPV", "2m", "", KindRoutine},
	{"polio-2", "IPV", "4m", "", KindRoutine},
	{"polio-3", "IPV", "6m", "19m", KindRoutine},
	{"polio-4", "IPV", "4y", "7y", KindRoutine},
	{"covid19-1", "COVID-19", "6m", "18y", KindSupplementary},
	{"influenza-1", "Influenza", "6m", "", KindRecurring},
	{"measles-1", "MMR", "12m", "16m", KindRoutine},
	{"mumps-1", "MMR", "12m", "16m", KindRoutine},
	{"rubella-1", "MMR", "12m", "16m", KindRoutine},
	{"measles-2", "MMR", "4y", "7y", KindRoutine},
	{"mumps-2", "MMR", "4y", "7y", KindRoutine},
	{"rubella-2", "MMR", "4y", "7y", KindRoutine},
	{"varicella-1", "Varicella", "12m", "16m", KindRoutine},
	{"varicella-2", "Varicella", "4y", "7y", KindRoutine},
	{"hepatitis_a-1", "Hepatitis A", "12m", "24m", KindRoutine},
	{"hepatitis_a-2", "Hepatitis A", "18m", "24m", KindRoutine},
	{"diphtheria-6", "Tdap", "11y", "13y", KindRoutine},
	{"tetanus-6", "Tdap", "11y", "13y", KindRoutine},
	{"pertussis-6", "Tdap", "11y", "13y", KindRoutine},
	{"hpv-1", "HPV", "11y", "13y", KindRoutine},
	{"hpv-2", "HPV", "138m", "13y", KindRoutine},
	{"meningococcal_acwy-1", "MenACWY", "11y", "13y", KindRoutine},
	{"meningococcal_acwy-2", "MenACWY", "16y", "17y", KindRoutine},
	{"meningococcal_b-1", "MenB", "16y", "24y", KindSupplementary},
	{"meningococcal_b-2", "MenB", "16y", "24y", KindSupplementary},
}

func ageString(a Age) string { return fmt.Sprintf("%d%c", a.N, a.Unit) }

func TestUSKnownLines(t *testing.T) {
	mustLoad(t)
	s, ok := GetSchema("us")
	if !ok {
		t.Fatal("GetSchema(us) not found")
	}
	byID := map[string]Line{}
	for _, l := range s.Lines {
		byID[l.ID] = l
	}
	for _, p := range usLines {
		l, ok := byID[p.id]
		if !ok {
			t.Errorf("%s: missing", p.id)
			continue
		}
		u := ""
		if l.Until != nil {
			u = ageString(*l.Until)
		}
		if l.Group != p.group || ageString(l.From) != p.from || u != p.until || l.Kind != p.kind {
			t.Errorf("%s = {%s %s %s %s}, want {%s %s %s %s}", p.id, l.Group, ageString(l.From), u, l.Kind,
				p.group, p.from, p.until, p.kind)
		}
		if l.BornFrom != "" || l.BornTo != "" {
			t.Errorf("%s: has a cohort", p.id)
		}
		if l.ID != fmt.Sprintf("%s-%d", l.Antigen, l.Series) {
			t.Errorf("%s: id is not <antigen>-<series>", l.ID)
		}
	}
	if len(s.Lines) != len(usLines) {
		t.Errorf("us.csv has %d lines, the table pins %d", len(s.Lines), len(usLines))
	}
	omitted := strings.Join(s.Omitted, "\n")
	for _, want := range []string{"COVID-19", "HPV", "MenACWY", "MenB", "cardiac", "IPV", "Tdap", "Abrysvo", "Dengue", "Mpox"} {
		if !strings.Contains(omitted, want) {
			t.Errorf("omitted does not mention %s", want)
		}
	}
}

func TestUSLineIDsUnique(t *testing.T) {
	mustLoad(t)
	s, _ := GetSchema("us")
	seen := map[string]bool{}
	for _, l := range s.Lines {
		if seen[l.ID] {
			t.Errorf("duplicate line id %s", l.ID)
		}
		seen[l.ID] = true
	}
	if len(seen) == 0 {
		t.Error("us.csv has no lines")
	}
}

func TestUSEvaluatesTwoMonthVisit(t *testing.T) {
	mustLoad(t)
	s, _ := GetSchema("us")
	e := mustEval(t, s, nil, nil, "2025-01-10", "2025-03-15")
	due := map[string]bool{}
	for _, l := range e.Lines {
		if l.Status == StatusDue && l.Line.From == age(2, UnitMonth) {
			due[l.Line.ID] = true
		}
	}
	for _, id := range []string{"diphtheria-1", "tetanus-1", "pertussis-1", "hib-1", "pneumococcus-1", "polio-1", "rotavirus-1"} {
		if !due[id] {
			t.Errorf("%s is not due at 2 months", id)
		}
	}
	groups := e.Due()
	if len(groups) == 0 || !hasTwoMonthGroup(groups) {
		t.Errorf("Due() = %+v, want a 2-month group", groups)
	}
}

func hasTwoMonthGroup(groups []DueGroup) bool {
	for _, g := range groups {
		if g.From == age(2, UnitMonth) && g.FromDate == "2025-03-10" && g.Status == StatusDue {
			return true
		}
	}
	return false
}

func TestUSAtFifteenYears(t *testing.T) {
	mustLoad(t)
	s, _ := GetSchema("us")
	e := mustEval(t, s, nil, nil, "2010-01-10", "2025-01-10")
	byStatus := map[Status][]string{}
	for _, l := range e.Lines {
		byStatus[l.Status] = append(byStatus[l.Status], l.Line.ID)
	}
	var wantOverdue []string
	for _, p := range usLines {
		if p.kind == KindRoutine && p.id != "meningococcal_acwy-2" {
			wantOverdue = append(wantOverdue, p.id)
		}
	}
	want := map[Status][]string{
		StatusOverdue:   wantOverdue,
		StatusAvailable: {"hib-4", "rotavirus-3", "rsv-1"},
		StatusUpcoming:  {"meningococcal_acwy-2"},
	}
	for st, ids := range want {
		if !sameSet(byStatus[st], ids) {
			t.Errorf("%s = %v, want %v", st, byStatus[st], ids)
		}
	}
	if len(byStatus) != len(want) {
		t.Errorf("statuses = %v", byStatus)
	}
	var also []string
	for _, l := range e.Also {
		also = append(also, l.Line.ID)
		if l.Status != StatusAvailable {
			t.Errorf("%s = %s, want available", l.Line.ID, l.Status)
		}
	}
	if !sameSet(also, []string{"covid19-1", "meningococcal_b-1", "meningococcal_b-2", "rsv-2"}) {
		t.Errorf("Also = %v", also)
	}
	if len(e.Recurring) != 1 || e.Recurring[0].Line.ID != "influenza-1" {
		t.Errorf("Recurring = %+v, want influenza-1 only", e.Recurring)
	}
}

func sameSet(a, b []string) bool {
	a, b = slices.Clone(a), slices.Clone(b)
	slices.Sort(a)
	slices.Sort(b)
	return slices.Equal(a, b)
}

type cohortPinned struct {
	pinned
	bornFrom, bornTo string
}

// UKHSA cohorts: the 18-month appointment exists for born on or after 1 July 2024; MMR-containing
// doses split again at 1 January 2025 and 1 September 2022 (MMRV information, Table 1).
const (
	ukNew18m   = "2024-07-01"
	ukOld18m   = "2024-06-30"
	ukMMRVFrom = "2025-01-01"
	ukMMRTo    = "2024-12-31"
	ukOneMMRV  = "2022-09-01"
	ukNoMMRV   = "2022-08-31"
)

func ukLine(id, group, from, until string, kind Kind, bornFrom, bornTo string) cohortPinned {
	return cohortPinned{pinned{id, group, from, until, kind}, bornFrom, bornTo}
}

// ukLines is every line of uk.csv, from the UKHSA schedule's own rows; a change to the file must change this table.
var ukLines = func() []cohortPinned {
	var ls []cohortPinned
	sixInOne := func(series int, from, suffix, bornFrom, bornTo string, antigens ...string) {
		for _, a := range antigens {
			ls = append(ls, ukLine(fmt.Sprintf("%s-%d%s", a, series, suffix), "6-in-1", from, "", KindRoutine, bornFrom, bornTo))
		}
	}
	all6 := []string{"diphtheria", "tetanus", "pertussis", "polio", "hib", "hepatitis_b"}
	for i, from := range []string{"8w", "12w", "16w"} {
		sixInOne(i+1, from, "", "", "", all6...)
	}
	ls = append(ls,
		ukLine("meningococcal_b-1", "MenB", "8w", "", KindRoutine, "", ""),
		ukLine("meningococcal_b-2", "MenB", "12w", "", KindRoutine, "", ""),
		ukLine("meningococcal_b-3", "MenB", "12m", "", KindRoutine, "", ""),
		ukLine("rotavirus-1", "Rotavirus", "8w", "", KindRoutine, "", ""),
		ukLine("rotavirus-2", "Rotavirus", "12w", "", KindRoutine, "", ""),
		ukLine("pneumococcus-1", "PCV", "16w", "", KindRoutine, "", ""),
		ukLine("pneumococcus-2", "PCV", "12m", "", KindRoutine, "", ""),
	)
	sixInOne(4, "18m", "-b202407", ukNew18m, "", "diphtheria", "tetanus", "pertussis", "polio")
	sixInOne(4, "18m", "", ukNew18m, "", "hib", "hepatitis_b")
	booster := func(group string, series int, from, suffix, bornFrom, bornTo string, antigens ...string) {
		for _, a := range antigens {
			ls = append(ls, ukLine(fmt.Sprintf("%s-%d%s", a, series, suffix), group, from, "", KindRoutine, bornFrom, bornTo))
		}
	}
	booster("4-in-1 pre-school booster", 5, "40m", "-b202407", ukNew18m, "", "diphtheria", "tetanus", "polio")
	booster("4-in-1 pre-school booster", 5, "40m", "", ukNew18m, "", "pertussis")
	booster("4-in-1 pre-school booster", 4, "40m", "-to202406", "", ukOld18m, "diphtheria", "tetanus", "pertussis", "polio")
	booster("3-in-1 teenage booster", 6, "14y", "", ukNew18m, "", "diphtheria", "tetanus", "polio")
	booster("3-in-1 teenage booster", 5, "14y", "-to202406", "", ukOld18m, "diphtheria", "tetanus", "polio")
	mmr := []string{"measles", "mumps", "rubella"}
	booster("MMRV", 1, "12m", "-b202501", ukMMRVFrom, "", append(mmr, "varicella")...)
	booster("MMR", 1, "12m", "-to202412", "", ukMMRTo, mmr...)
	booster("MMRV", 2, "18m", "-b202407", ukNew18m, "", mmr...)
	booster("MMRV", 2, "18m", "-b202501", ukMMRVFrom, "", "varicella")
	booster("MMRV", 1, "18m", "-b202407", ukNew18m, ukMMRTo, "varicella")
	booster("MMRV", 3, "40m", "", ukNew18m, ukMMRTo, mmr...)
	booster("MMRV", 2, "40m", "-b202407", ukNew18m, ukMMRTo, "varicella")
	booster("MMRV", 2, "40m", "-b202209", ukOneMMRV, ukOld18m, mmr...)
	booster("MMRV", 1, "40m", "-b202209", ukOneMMRV, ukOld18m, "varicella")
	booster("MMR", 2, "40m", "-to202208", "", ukNoMMRV, mmr...)
	return append(ls,
		ukLine("hpv-1", "HPV", "12y", "14y", KindRoutine, "", ""),
		ukLine("meningococcal_acwy-1", "MenACWY", "14y", "", KindRoutine, "", ""),
		ukLine("influenza-1", "Flu", "2y", "", KindRecurring, "", ""),
	)
}()

// cohortTag is the id suffix of a line that shares its antigen and series with another line.
func cohortTag(l Line) string {
	if l.BornFrom != "" {
		return "-b" + strings.ReplaceAll(l.BornFrom[:7], "-", "")
	}
	return "-to" + strings.ReplaceAll(l.BornTo[:7], "-", "")
}

func mustUK(t *testing.T) Schema {
	t.Helper()
	mustLoad(t)
	s, ok := GetSchema("uk")
	if !ok {
		t.Fatal("GetSchema(uk) not found")
	}
	return s
}

func TestUKKnownLines(t *testing.T) {
	s := mustUK(t)
	byID := map[string]Line{}
	shared := map[string]int{}
	for _, l := range s.Lines {
		byID[l.ID] = l
		shared[fmt.Sprintf("%s-%d", l.Antigen, l.Series)]++
	}
	for _, p := range ukLines {
		l, ok := byID[p.id]
		if !ok {
			t.Errorf("%s: missing", p.id)
			continue
		}
		u := ""
		if l.Until != nil {
			u = ageString(*l.Until)
		}
		if l.Group != p.group || ageString(l.From) != p.from || u != p.until || l.Kind != p.kind ||
			l.BornFrom != p.bornFrom || l.BornTo != p.bornTo {
			t.Errorf("%s = {%s %s %s %s %s %s}, want {%s %s %s %s %s %s}", p.id, l.Group, ageString(l.From), u, l.Kind,
				l.BornFrom, l.BornTo, p.group, p.from, p.until, p.kind, p.bornFrom, p.bornTo)
		}
	}
	for _, l := range s.Lines {
		base := fmt.Sprintf("%s-%d", l.Antigen, l.Series)
		want := base
		if shared[base] > 1 {
			want += cohortTag(l)
		}
		if l.ID != want {
			t.Errorf("%s: id is not %s", l.ID, want)
		}
	}
	if len(s.Lines) != len(ukLines) {
		t.Errorf("uk.csv has %d lines, the table pins %d", len(s.Lines), len(ukLines))
	}
	omitted := strings.Join(s.Omitted, "\n")
	for _, want := range []string{"BCG", "hepatitis B infection", "clinical risk group", "pregnancy", "Hib/MenC",
		"catch-up", "underlying medical conditions", "Scotland"} {
		if !strings.Contains(omitted, want) {
			t.Errorf("omitted does not mention %s", want)
		}
	}
	if strings.Contains(s.Attribution, "NHS") {
		t.Error("attribution cites the NHS; OGL attribution must cite UKHSA only")
	}
}

// ukBrands are the trade names the UKHSA schedule and its supporting pages print.
var ukBrands = []string{
	"infanrix", "vaxelis", "bexsero", "rotarix", "prevenar", "proquad", "priorix", "repevax", "gardasil",
	"revaxis", "menquadfi", "engerix", "hbvaxpro", "menitorix", "pneumovax", "shingrix", "abrysvo", "adacel",
}

func TestUKHasNoBrandNames(t *testing.T) {
	s := mustUK(t)
	texts := append([]string{s.Name, s.Authority, s.Edition, s.Licence, s.Attribution}, s.Omitted...)
	for _, l := range s.Lines {
		texts = append(texts, l.ID, l.Group, l.Note)
	}
	for _, x := range texts {
		lower := strings.ToLower(x)
		for _, b := range ukBrands {
			if strings.Contains(lower, b) {
				t.Errorf("%q contains the brand name %q", x, b)
			}
		}
		// "fluenz" is inside "influenza", so it is matched as a word.
		if slices.Contains(strings.FieldsFunc(lower, func(r rune) bool { return r < 'a' || r > 'z' }), "fluenz") {
			t.Errorf("%q contains the brand name fluenz", x)
		}
	}
}

// ukSeries is what one birth cohort's kid must see: each antigen's from ages in series order.
type ukSeries map[string][]string

func ukCohort(dtp, pertussis, hibHepB, mmr, varicella []string) ukSeries {
	return ukSeries{
		"diphtheria": dtp, "tetanus": dtp, "polio": dtp, "pertussis": pertussis, "hib": hibHepB, "hepatitis_b": hibHepB,
		"measles": mmr, "mumps": mmr, "rubella": mmr, "varicella": varicella,
		"rotavirus": {"8w", "12w"}, "meningococcal_b": {"8w", "12w", "12m"}, "pneumococcus": {"16w", "12m"},
		"hpv": {"12y"}, "meningococcal_acwy": {"14y"},
	}
}

func TestUKCohortBoundaries(t *testing.T) {
	s := mustUK(t)
	old := ukCohort([]string{"8w", "12w", "16w", "40m", "14y"}, []string{"8w", "12w", "16w", "40m"},
		[]string{"8w", "12w", "16w"}, []string{"12m", "40m"}, nil)
	oneMMRV := ukCohort(old["diphtheria"], old["pertussis"], old["hib"], old["measles"], []string{"40m"})
	threeMMR := ukCohort([]string{"8w", "12w", "16w", "18m", "40m", "14y"}, []string{"8w", "12w", "16w", "18m", "40m"},
		[]string{"8w", "12w", "16w", "18m"}, []string{"12m", "18m", "40m"}, []string{"18m", "40m"})
	mmrv := ukCohort(threeMMR["diphtheria"], threeMMR["pertussis"], threeMMR["hib"], []string{"12m", "18m"},
		[]string{"12m", "18m"})
	for born, want := range map[string]ukSeries{
		"2019-12-31": old, ukNoMMRV: old, ukOneMMRV: oneMMRV, ukOld18m: oneMMRV,
		ukNew18m: threeMMR, ukMMRTo: threeMMR, ukMMRVFrom: mmrv, "2026-09-01": mmrv,
	} {
		e := mustEval(t, s, nil, nil, born, born)
		got := map[string][]Line{}
		for _, l := range append(slices.Clone(e.Lines), e.Also...) {
			got[l.Line.Antigen] = append(got[l.Line.Antigen], l.Line)
		}
		for antigen, ages := range want {
			lines := got[antigen]
			slices.SortFunc(lines, func(a, b Line) int { return a.Series - b.Series })
			var from []string
			for i, l := range lines {
				from = append(from, ageString(l.From))
				if l.Series != i+1 {
					t.Errorf("born %s: %s series %v is not 1..%d", born, antigen, lines, len(lines))
				}
			}
			if !slices.Equal(from, ages) {
				t.Errorf("born %s: %s from ages = %v, want %v", born, antigen, from, ages)
			}
		}
		for antigen := range got {
			if _, ok := want[antigen]; !ok {
				t.Errorf("born %s: unexpected antigen %s", born, antigen)
			}
		}
		if len(e.Recurring) != 1 || e.Recurring[0].Line.ID != "influenza-1" {
			t.Errorf("born %s: Recurring = %+v, want influenza-1 only", born, e.Recurring)
		}
	}
}

func TestUKEvaluatesEightWeekVisit(t *testing.T) {
	s := mustUK(t)
	e := mustEval(t, s, nil, nil, "2026-09-01", "2026-10-27")
	var due []string
	for _, l := range e.Lines {
		if l.Status == StatusDue {
			due = append(due, l.Line.ID)
		}
	}
	want := []string{"diphtheria-1", "tetanus-1", "pertussis-1", "polio-1", "hib-1", "hepatitis_b-1",
		"meningococcal_b-1", "rotavirus-1"}
	if !sameSet(due, want) {
		t.Errorf("due = %v, want %v", due, want)
	}
	groups := e.Due()
	if len(groups) != 1 || groups[0].From != age(8, UnitWeek) || groups[0].FromDate != "2026-10-27" ||
		groups[0].Status != StatusDue || groups[0].Count != len(want) {
		t.Errorf("Due() = %+v, want one 8-week group of %d on 2026-10-27", groups, len(want))
	}
}

func TestUKAtFifteenYears(t *testing.T) {
	s := mustUK(t)
	for _, born := range []string{"2010-01-10", ukOld18m, ukNew18m, ukMMRVFrom} {
		b, _ := time.Parse(dateLayout, born)
		e := mustEval(t, s, nil, nil, born, b.AddDate(15, 0, 0).Format(dateLayout))
		for _, l := range e.Lines {
			if l.Status != StatusOverdue {
				t.Errorf("born %s: %s = %s, want overdue", born, l.Line.ID, l.Status)
			}
		}
		if len(e.Also) != 0 {
			t.Errorf("born %s: Also = %+v, want none", born, e.Also)
		}
		if len(e.Recurring) != 1 || e.Recurring[0].Line.ID != "influenza-1" {
			t.Errorf("born %s: Recurring = %+v, want influenza-1 only", born, e.Recurring)
		}
		n := 0
		for _, g := range e.Due() {
			n += g.Count
			if g.Status != StatusOverdue {
				t.Errorf("born %s: Due group %+v is not overdue", born, g)
			}
		}
		if n != len(e.Lines) {
			t.Errorf("born %s: Due() counts %d lines, want %d", born, n, len(e.Lines))
		}
	}
}
