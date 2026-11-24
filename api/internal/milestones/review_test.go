package milestones

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const reviewDocsDir = "../../../docs/milestones"

const pdfURL = "https://www.cdc.gov/act-early/media/pdfs/2025/10/cdc-milestone-checklists-ltsae-english-508.pdf"

// pdfDifferences: items whose wording in the combined PDF (compared 2026-11-24) is not the web page's.
var pdfDifferences = []struct{ ID, PDF string }{
	{"4m-se-chuckles-not-yet", "Chuckles (not yet a full laugh) when you try to make her laugh"},
	{"4m-cg-if-hungry-opens", "If hungry, opens mouth when she sees breast or bottle"},
	{"4m-cg-looks-at-her", "Looks at his hands with interest"},
	{"4m-mp-holds-head-steady", "Holds head steady without support when you are holding her"},
	{"4m-mp-uses-his-arm", "Uses her arm to swing at toys"},
	{"6m-se-likes-to-look", "Likes to look at himself in a mirror"},
	{"6m-cg-reaches-to-grab", "Reaches to grab a toy he wants"},
	{"6m-mp-leans-on-hands", "Leans on hands to support himself when sitting"},
	{"9m-lc-makes-a-lot", "Makes different sounds like “mamamama” and “babababa”"},
	{"18m-mp-feeds-himself-with", "Feeds herself with her fingers"},
	{"48m-lc-talks-about-at", "Talks about at least one thing that happened during his day, like “I played soccer.”"},
	{"48m-mp-serves-herself-food", "Serves himself food or pours water, with adult supervision"},
}

var domainHeadings = map[Domain]string{
	DomainSocialEmotional: "Social/Emotional",
	DomainLanguage:        "Language/Communication",
	DomainCognitive:       "Cognitive",
	DomainMovement:        "Movement/Physical Development",
}

func ageTitle(m int) string {
	switch {
	case m == 12:
		return "1 year"
	case m%12 == 0:
		return fmt.Sprintf("%d years", m/12)
	}
	return fmt.Sprintf("%d months", m)
}

func cell(s string) string { return strings.ReplaceAll(s, "|", `\|`) }

// renderMarkdown is the review document of the catalog: provenance, one table per age, retired items, PDF wording.
func renderMarkdown(c Catalog) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Milestone catalog: %s (`%s`)\n\n", c.Name, c.ID)
	fmt.Fprintf(&b, "Generated from `api/internal/milestones/data/%s.csv`; do not edit. After changing the CSV, run\n", c.ID)
	b.WriteString("`UPDATE_DOCS=1 go test ./internal/milestones/` in `api/` and commit both files.\n\n")
	for _, kv := range [][2]string{
		{"Authority", c.Authority}, {"Edition", c.Edition}, {"Retrieved", c.Retrieved}, {"Source", c.OriginURL},
		{"Licence", c.Licence}, {"Attribution", c.Attribution}, {"Screening note", c.ScreeningNote},
	} {
		fmt.Fprintf(&b, "- **%s:** %s\n", kv[0], kv[1])
	}
	b.WriteString("\n## Transcription\n\n")
	b.WriteString("Each item is the text of one list item under the four milestone headings of its age page, in page\n")
	b.WriteString("order, with HTML entities decoded and the pages' wording kept, pronouns included. The only change is the\n")
	b.WriteString("removal of spaces at the start or end of a list item; no item had a run of spaces or a line break inside.\n")
	b.WriteString("The screening note is the \"Keep in mind\" sentence of the age pages without its italics and link. Ten\n")
	b.WriteString("pages word it as shown; the 6- and 9-month pages put the campaign name in straight double quotes\n")
	b.WriteString("(\"Learn the Signs. Act Early.\" materials …).\n")
	if len(c.Omitted) > 0 {
		b.WriteString("\n## Not transcribed from the source\n\n")
		for _, o := range c.Omitted {
			fmt.Fprintf(&b, "- %s\n", o)
		}
	}
	const head = "| Id | Domain | Text | Source page |\n|---|---|---|---|\n"
	var retired []Item
	last := 0
	for _, it := range c.Items {
		if it.Retired != "" {
			retired = append(retired, it)
			continue
		}
		if it.AgeMonths != last {
			fmt.Fprintf(&b, "\n## %s\n\n%s", ageTitle(it.AgeMonths), head)
			last = it.AgeMonths
		}
		fmt.Fprintf(&b, "| `%s` | %s | %s | %s |\n", it.ID, domainHeadings[it.Domain], cell(it.Text), it.SourceURL)
	}
	b.WriteString("\n## Retired items\n\n")
	if len(retired) == 0 {
		b.WriteString("None.\n")
	} else {
		b.WriteString("| Id | Age | Domain | Text | Retired |\n|---|---|---|---|---|\n")
		for _, it := range retired {
			fmt.Fprintf(&b, "| `%s` | %s | %s | %s | %s |\n", it.ID, ageTitle(it.AgeMonths), domainHeadings[it.Domain], cell(it.Text), it.Retired)
		}
	}
	b.WriteString("\n## Wording differences from the PDF\n\n")
	fmt.Fprintf(&b, "Compared on 2026-11-24 with CDC's combined English checklists PDF, %s,\n", pdfURL)
	b.WriteString("through its text (`pdftotext`, spaces collapsed); the per-age PDFs were not compared. Every other item\n")
	b.WriteString("appears in the PDF word for word. The reverse direction (items in the PDF but not on the web pages)\n")
	b.WriteString("was not checked item by item. The catalog keeps the web pages' wording.\n\n")
	b.WriteString("| Id | Web page (in the catalog) | PDF |\n|---|---|---|\n")
	text := map[string]string{}
	for _, it := range c.Items {
		text[it.ID] = it.Text
	}
	for _, d := range pdfDifferences {
		fmt.Fprintf(&b, "| `%s` | %s | %s |\n", d.ID, cell(text[d.ID]), cell(d.PDF))
	}
	return b.String()
}

func TestPDFDifferencesNameShippedItems(t *testing.T) {
	mustLoad(t)
	for _, d := range pdfDifferences {
		it, ok := Lookup(d.ID)
		if !ok || it.Text == d.PDF {
			t.Errorf("pdfDifferences %s: unknown id or same wording", d.ID)
		}
	}
}

func TestReviewDocIsCurrent(t *testing.T) {
	c := mustLoad(t)
	p := filepath.Join(reviewDocsDir, c.ID+".md")
	want := renderMarkdown(c)
	if os.Getenv("UPDATE_DOCS") == "1" {
		if err := os.MkdirAll(reviewDocsDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(want), 0o644); err != nil {
			t.Fatal(err)
		}
	} else if got, err := os.ReadFile(p); err != nil || string(got) != want {
		t.Errorf("docs/milestones/%s.md is absent or stale: run UPDATE_DOCS=1 go test ./internal/milestones/ in api/", c.ID)
	}
	docs, err := filepath.Glob(filepath.Join(reviewDocsDir, "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range docs {
		if base := filepath.Base(d); base != c.ID+".md" && base != "README.md" {
			t.Errorf("docs/milestones/%s has no shipped catalog: delete it", base)
		}
	}
}

func TestRenderMarkdown(t *testing.T) {
	c := Catalog{ID: "zz", Name: "Test", Omitted: []string{"Tips", "Images"}, Items: []Item{
		{"2m-se-calms-down", 2, DomainSocialEmotional, "Calms | down", "https://www.cdc.gov/2.html", ""},
		{"2m-mp-holds-head", 2, DomainMovement, "Holds head", "https://www.cdc.gov/2.html", "2027-01-01"},
		{"12m-cg-looks-for", 12, DomainCognitive, "Looks for", "https://www.cdc.gov/12.html", ""},
	}}
	got := renderMarkdown(c)
	for _, want := range []string{
		"# Milestone catalog: Test (`zz`)\n", "## Not transcribed from the source\n\n- Tips\n- Images\n",
		"## 2 months\n\n| Id | Domain | Text | Source page |\n|---|---|---|---|\n" +
			"| `2m-se-calms-down` | Social/Emotional | Calms \\| down | https://www.cdc.gov/2.html |\n\n## 1 year\n",
		"| `12m-cg-looks-for` | Cognitive | Looks for | https://www.cdc.gov/12.html |\n",
		"## Retired items\n\n| Id | Age | Domain | Text | Retired |\n|---|---|---|---|---|\n" +
			"| `2m-mp-holds-head` | 2 months | Movement/Physical Development | Holds head | 2027-01-01 |\n",
		"## Wording differences from the PDF\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("renderMarkdown lacks %q:\n%s", want, got)
		}
	}
	for m, want := range map[int]string{2: "2 months", 12: "1 year", 15: "15 months", 24: "2 years", 30: "30 months", 60: "5 years"} {
		if got := ageTitle(m); got != want {
			t.Errorf("ageTitle(%d) = %q, want %q", m, got, want)
		}
	}
}
