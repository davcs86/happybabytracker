package vaccines

import (
	"bytes"
	"fmt"
	"io/fs"
	"strings"
	"testing"
	"time"
)

// mandatoryAttribution is the wording each schedule's licence requires in its attribution.
var mandatoryAttribution = map[string][]string{
	"us": {"Source: CDC", "CDC does not endorse this application or its use of the material."},
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
	if !seen["us"] {
		t.Error("data/us.csv is not shipped")
	}
}

type pinned struct {
	id, group, from, until string
	kind                   Kind
}

// usLines is every line of us.csv; a change to the file must change this table.
var usLines = []pinned{
	{"rsv-1", "RSV", "0d", "8m", KindSupplementary},
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
	{"hib-3", "Hib", "6m", "", KindMayNotBeNeeded},
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
	{"hepatitis_a-2", "Hepatitis A", "12m", "24m", KindRoutine},
	{"diphtheria-6", "Tdap", "11y", "13y", KindRoutine},
	{"tetanus-6", "Tdap", "11y", "13y", KindRoutine},
	{"pertussis-6", "Tdap", "11y", "13y", KindRoutine},
	{"hpv-1", "HPV", "11y", "13y", KindRoutine},
	{"hpv-2", "HPV", "11y", "13y", KindRoutine},
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
	}
	if len(s.Lines) != len(usLines) {
		t.Errorf("us.csv has %d lines, the table pins %d", len(s.Lines), len(usLines))
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

func TestUSNeverFlagsOptionalLinesOverdue(t *testing.T) {
	mustLoad(t)
	s, _ := GetSchema("us")
	e := mustEval(t, s, nil, nil, "2010-01-10", "2025-01-10")
	for _, l := range e.Lines {
		if l.Status == StatusOverdue && l.Line.Kind != KindRoutine {
			t.Errorf("%s (%s) is overdue at 15 years", l.Line.ID, l.Line.Kind)
		}
		if l.Line.Kind == KindSupplementary || l.Line.Kind == KindRecurring {
			t.Errorf("%s (%s) is listed with the routine lines", l.Line.ID, l.Line.Kind)
		}
	}
	if len(e.Also) == 0 {
		t.Error("no supplementary lines at 15 years")
	}
	for _, l := range e.Also {
		if l.Status != StatusAvailable {
			t.Errorf("%s = %s, want available", l.Line.ID, l.Status)
		}
	}
	if len(e.Recurring) != 1 || e.Recurring[0].Line.Antigen != "influenza" {
		t.Errorf("Recurring = %+v, want influenza only", e.Recurring)
	}
}
