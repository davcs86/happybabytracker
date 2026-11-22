package vaccines

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
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
	b.WriteString("- `may_not_be_needed`: depends on the product or history; due inside its window, never overdue.\n")
	b.WriteString("- `supplementary`: shared decision-making or risk groups; listed under \"Also available\", never due or overdue.\n")
	b.WriteString("- `recurring`: seasonal; listed with the doses given, never due or overdue.\n\n")
	b.WriteString("A window runs from the age in `from` up to, not including, the age in `until`; without `until` it is one month.\n")
	lines := slices.Clone(s.Lines)
	slices.SortStableFunc(lines, func(x, y Line) int {
		return cmp.Or(cmp.Compare(x.From.approxDays(), y.From.approxDays()), cmp.Compare(x.Group, y.Group),
			cmp.Compare(x.Antigen, y.Antigen), cmp.Compare(x.Series, y.Series))
	})
	for i, l := range lines {
		if i == 0 || lines[i-1].From != l.From {
			fmt.Fprintf(&b, "\n## %s\n\n", ageLabel(l.From))
			b.WriteString("| Vaccine | Antigen | Dose | Window | Kind | Note |\n|---|---|---|---|---|---|\n")
		}
		window := "from " + ageLabel(l.From) + " (default window)"
		if l.Until != nil {
			window = "from " + ageLabel(l.From) + " until " + ageLabel(*l.Until)
		}
		if l.BornFrom != "" || l.BornTo != "" {
			window += fmt.Sprintf("; born %s to %s", cmp.Or(l.BornFrom, "any date"), cmp.Or(l.BornTo, "any date"))
		}
		fmt.Fprintf(&b, "| %s | %s | %d | %s | `%s` | %s |\n", cell(l.Group), l.Antigen, l.Series, window, l.Kind, cell(l.Note))
	}
	return b.String()
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
}

func TestRenderMarkdown(t *testing.T) {
	s := Schema{ID: "zz", Name: "Test", Lines: []Line{
		line("b-1", "B|x", "rotavirus", 1, age(2, UnitMonth), nil, KindRoutine),
		line("a-1", "A", "hepatitis_b", 1, age(0, UnitDay), until(1, UnitMonth), KindSupplementary),
	}}
	s.Lines[0].BornFrom = "2024-07-01"
	got := renderMarkdown(s)
	for _, want := range []string{
		"## Birth\n", "| A | hepatitis_b | 1 | from Birth until 1 month | `supplementary` |  |\n",
		"## 2 months\n", `| B\|x | rotavirus | 1 | from 2 months (default window); born 2024-07-01 to any date | ` + "`routine`",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("renderMarkdown lacks %q:\n%s", want, got)
		}
	}
	if strings.Index(got, "## Birth") > strings.Index(got, "## 2 months") {
		t.Error("tables are not in age order")
	}
}
