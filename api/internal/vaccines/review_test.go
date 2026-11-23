package vaccines

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/davcs86/happybabytracker/api/internal/constants"
)

const reviewDocsDir = "../../../docs/vaccine-schedules"

var unitNames = map[Unit][2]string{
	UnitDay:   {"day", "days"},
	UnitWeek:  {"week", "weeks"},
	UnitMonth: {"month", "months"},
	UnitYear:  {"year", "years"},
}

func ageLabel(a Age) string {
	if a.N == 0 {
		return "Birth"
	}
	names := unitNames[a.Unit]
	if a.N == 1 {
		return "1 " + names[0]
	}
	return fmt.Sprintf("%d %s", a.N, names[1])
}

// defaultEnd labels the end of a line without until, VaccineDefaultWindowMonths after its from age.
func defaultEnd(from Age) string {
	m := Age{constants.VaccineDefaultWindowMonths, UnitMonth}
	switch {
	case from.Unit == UnitMonth:
		return ageLabel(Age{from.N + m.N, UnitMonth})
	case from.N == 0:
		return ageLabel(m)
	}
	return ageLabel(from) + " and " + ageLabel(m)
}

func cell(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "|", `\|`), "\n", " ")
}

// renderMarkdown is the review document of a schedule: its provenance and one table per from age.
func renderMarkdown(s Schema) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Vaccination schedule: %s (`%s`)\n\n", s.Name, s.ID)
	fmt.Fprintf(&b, "Generated from `api/internal/vaccines/data/%s.csv`; do not edit. After changing the CSV, run\n", s.ID)
	b.WriteString("`UPDATE_DOCS=1 go test ./internal/vaccines/` in `api/` and commit both files.\n\n")
	for _, kv := range [][2]string{
		{"Authority", s.Authority}, {"Edition", s.Edition}, {"Retrieved", s.Retrieved},
		{"Source", s.OriginURL}, {"Licence", s.Licence}, {"Attribution", s.Attribution},
	} {
		fmt.Fprintf(&b, "- **%s:** %s\n", kv[0], kv[1])
	}
	b.WriteString("\n## Kinds\n\n")
	b.WriteString("- `routine`: due inside its window, overdue after it.\n")
	b.WriteString("- `may_not_be_needed`: depends on the product or history; is due inside its window; afterwards it is listed as available and never overdue.\n")
	b.WriteString("- `supplementary`: shared decision-making or risk groups; listed under \"Also available\", never due or overdue.\n")
	b.WriteString("- `recurring`: seasonal; listed with the doses given, never due or overdue.\n\n")
	fmt.Fprintf(&b, "A window runs from the age in `from` up to, not including, the age in `until`; without `until` it is %s.\n",
		ageLabel(Age{constants.VaccineDefaultWindowMonths, UnitMonth}))
	if len(s.Omitted) > 0 {
		b.WriteString("\n## Not transcribed from the source\n\n")
		for _, o := range s.Omitted {
			fmt.Fprintf(&b, "- %s\n", o)
		}
	}
	names := map[string]string{}
	for _, a := range Antigens() {
		names[a.ID] = a.Name
	}
	lines := slices.Clone(s.Lines)
	slices.SortStableFunc(lines, func(x, y Line) int {
		return cmp.Or(cmp.Compare(x.From.approxDays(), y.From.approxDays()), cmp.Compare(x.Group, y.Group),
			cmp.Compare(x.Antigen, y.Antigen), cmp.Compare(x.Series, y.Series))
	})
	const tableHead = "| Vaccine | Antigen | Dose | Window | Kind | Note |\n|---|---|---|---|---|---|\n"
	var verify []Line
	for _, l := range lines {
		if strings.HasPrefix(l.Note, "(verify)") {
			verify = append(verify, l)
		}
	}
	if len(verify) > 0 {
		b.WriteString("\n## Lines to verify\n\nThese lines carry `(verify)`: check them against the source first.\n\n" + tableHead)
		for _, l := range verify {
			b.WriteString(reviewRow(l, names))
		}
	}
	for i, l := range lines {
		if i == 0 || lines[i-1].From != l.From {
			fmt.Fprintf(&b, "\n## %s\n\n", ageLabel(l.From))
			b.WriteString(tableHead)
		}
		b.WriteString(reviewRow(l, names))
	}
	return b.String()
}

func reviewRow(l Line, names map[string]string) string {
	window := "from " + ageLabel(l.From) + " until " + defaultEnd(l.From) + " (default window)"
	switch {
	case l.Kind == KindRecurring:
		window = "from " + ageLabel(l.From) + ", every season"
	case l.Until != nil:
		window = "from " + ageLabel(l.From) + " until " + ageLabel(*l.Until)
	case l.Kind == KindSupplementary:
		window = "from " + ageLabel(l.From)
	}
	if l.BornFrom != "" || l.BornTo != "" {
		window += fmt.Sprintf("; born %s to %s", cmp.Or(l.BornFrom, "any date"), cmp.Or(l.BornTo, "any date"))
	}
	return fmt.Sprintf("| %s | %s | %d | %s | `%s` | %s |\n", cell(l.Group), cell(names[l.Antigen]), l.Series, window, l.Kind, cell(l.Note))
}

func TestReviewDocsAreCurrent(t *testing.T) {
	mustLoad(t)
	update := os.Getenv("UPDATE_DOCS") == "1"
	for _, s := range Schemas() {
		p := filepath.Join(reviewDocsDir, s.ID+".md")
		want := renderMarkdown(s)
		if update {
			if err := os.MkdirAll(reviewDocsDir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte(want), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		got, err := os.ReadFile(p)
		if err != nil || string(got) != want {
			t.Errorf("docs/vaccine-schedules/%s.md is missing or stale: run UPDATE_DOCS=1 go test ./internal/vaccines/ in api/", s.ID)
		}
	}
	docs, err := filepath.Glob(filepath.Join(reviewDocsDir, "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range docs {
		base := filepath.Base(d)
		if _, ok := GetSchema(strings.TrimSuffix(base, ".md")); !ok && base != "README.md" {
			t.Errorf("docs/vaccine-schedules/%s has no shipped schedule: delete it", base)
		}
	}
}

func TestRenderMarkdown(t *testing.T) {
	mustLoad(t)
	s := Schema{ID: "zz", Name: "Test", Omitted: []string{"Mpox", "Dengue"}, Lines: []Line{
		line("b-1", "B|x", "rotavirus", 1, age(2, UnitMonth), nil, KindRoutine),
		line("a-1", "A", "hepatitis_b", 1, age(0, UnitDay), until(1, UnitMonth), KindSupplementary),
		line("influenza-1", "Flu", "influenza", 1, age(2, UnitYear), nil, KindRecurring),
		line("varicella-1", "Var", "varicella", 1, age(12, UnitMonth), nil, KindSupplementary),
	}}
	s.Lines[0].BornFrom = "2024-07-01"
	s.Lines[2].Note = "(verify) season"
	got := renderMarkdown(s)
	for _, want := range []string{
		"## Birth\n", "| A | Hepatitis B | 1 | from Birth until 1 month | `supplementary` |  |\n",
		"## 2 months\n", `| B\|x | Rotavirus | 1 | from 2 months until 3 months (default window); born 2024-07-01 to any date | ` + "`routine`",
		"## Not transcribed from the source\n\n- Mpox\n- Dengue\n", "without `until` it is 1 month.",
		"is due inside its window; afterwards it is listed as available and never overdue",
		"| Flu | Influenza | 1 | from 2 years, every season | `recurring` |",
		"| Var | Varicella (chickenpox) | 1 | from 12 months | `supplementary` |  |\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("renderMarkdown lacks %q:\n%s", want, got)
		}
	}
	verify := "## Lines to verify\n\nThese lines carry `(verify)`: check them against the source first.\n\n" +
		"| Vaccine | Antigen | Dose | Window | Kind | Note |\n|---|---|---|---|---|---|\n" +
		"| Flu | Influenza | 1 | from 2 years, every season | `recurring` | (verify) season |\n"
	if i := strings.Index(got, verify); i < 0 || i > strings.Index(got, "## Birth") {
		t.Errorf("renderMarkdown lacks a verify section before the age tables:\n%s", got)
	}
	if strings.Index(got, "## Birth") > strings.Index(got, "## 2 months") {
		t.Error("tables are not in age order")
	}
	for from, want := range map[Age]string{
		age(0, UnitDay): "1 month", age(2, UnitMonth): "3 months", age(8, UnitWeek): "8 weeks and 1 month",
		age(16, UnitYear): "16 years and 1 month",
	} {
		if got := defaultEnd(from); got != want {
			t.Errorf("defaultEnd(%v) = %q, want %q", from, got, want)
		}
	}
}
