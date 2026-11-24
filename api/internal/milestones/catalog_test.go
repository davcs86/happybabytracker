package milestones

import (
	"slices"
	"strings"
	"testing"
)

const testProvenance = `# catalog: cdc
# name: Test checklists
# authority: Test authority
# edition: 2022 revision
# retrieved: 2026-11-24
# origin_url: https://www.cdc.gov/act-early/milestones/index.html
# licence: Public domain
# attribution: Source: Test authority
# screening_note: Test note.
# omitted: tips; images
`

const testHeader = "id,age_months,domain,text,source_url,retired\n"

const (
	rowA = "2m-se-calms-down,2,social_emotional,Calms down when spoken to,https://www.cdc.gov/act-early/milestones/2-months.html,"
	rowB = "2m-mp-holds-head-up,2,movement_physical,Holds head up,https://www.cdc.gov/act-early/milestones/2-months.html,"
	rowC = "4m-lc-makes-sounds,4,language_communication,Makes sounds,https://www.cdc.gov/act-early/milestones/4-months.html,2026-11-24"
	rowD = "4m-cg-reaches-for-toy,4,cognitive,\"Reaches, then grasps\",https://www.cdc.gov/act-early/milestones/4-months.html,"
)

// Provenance is lines 1-10, the header line 11, so the first row is CSV line 12.
func testFile(rows ...string) string {
	return testProvenance + testHeader + strings.Join(rows, "\n") + "\n"
}

func TestParseValidCatalog(t *testing.T) {
	c, err := parse("cdc.csv", strings.NewReader(testFile(rowA, rowB, rowC, rowD)))
	if err != nil {
		t.Fatal(err)
	}
	if c.ID != "cdc" || c.Name != "Test checklists" || c.Authority != "Test authority" || c.Edition != "2022 revision" ||
		c.Retrieved != "2026-11-24" || c.OriginURL != "https://www.cdc.gov/act-early/milestones/index.html" ||
		c.Licence != "Public domain" || c.Attribution != "Source: Test authority" || c.ScreeningNote != "Test note." {
		t.Errorf("provenance = %+v", c)
	}
	if !slices.Equal(c.Omitted, []string{"tips", "images"}) {
		t.Errorf("omitted = %q", c.Omitted)
	}
	want := []Item{
		{"2m-se-calms-down", 2, DomainSocialEmotional, "Calms down when spoken to", "https://www.cdc.gov/act-early/milestones/2-months.html", ""},
		{"2m-mp-holds-head-up", 2, DomainMovement, "Holds head up", "https://www.cdc.gov/act-early/milestones/2-months.html", ""},
		{"4m-lc-makes-sounds", 4, DomainLanguage, "Makes sounds", "https://www.cdc.gov/act-early/milestones/4-months.html", "2026-11-24"},
		{"4m-cg-reaches-for-toy", 4, DomainCognitive, "Reaches, then grasps", "https://www.cdc.gov/act-early/milestones/4-months.html", ""},
	}
	if !slices.Equal(c.Items, want) {
		t.Errorf("items = %+v", c.Items)
	}
}

func TestParseWithoutOmitted(t *testing.T) {
	in := strings.Replace(testFile(rowA), "# omitted: tips; images\n", "", 1)
	c, err := parse("cdc.csv", strings.NewReader(in))
	if err != nil || c.Omitted != nil {
		t.Errorf("parse = %v, omitted %v", err, c.Omitted)
	}
}

func row(id, age, domain, text, url, retired string) string {
	return strings.Join([]string{id, age, domain, text, url, retired}, ",")
}

const goodURL = "https://www.cdc.gov/act-early/milestones/2-months.html"

// leak is a cell value that must not appear in the error.
func TestParseRefusals(t *testing.T) {
	longText := strings.Repeat("a", 201)
	for _, tc := range []struct {
		name, in, want, leak string
	}{
		{"unknown provenance key", strings.Replace(testFile(rowA), "# licence:", "# licensee:", 1),
			"milestones: cdc.csv:7: column key: unknown provenance key", "licensee"},
		{"not a provenance line", strings.Replace(testFile(rowA), "# licence: Public domain", "#licence Public domain", 1),
			"milestones: cdc.csv:7: column key: not a provenance line", "Public domain"},
		{"duplicate provenance key", strings.Replace(testFile(rowA), "# licence: Public domain\n", "# licence: Public domain\n# licence: Other\n", 1),
			"milestones: cdc.csv:8: column licence: duplicate provenance key", "Other"},
		{"empty provenance value", strings.Replace(testFile(rowA), "# name: Test checklists", "# name: ", 1),
			"milestones: cdc.csv:2: column name: is empty", ""},
		{"provenance key not present", strings.Replace(testFile(rowA), "# screening_note: Test note.\n", "", 1),
			"milestones: cdc.csv:10: column screening_note: provenance key not present", ""},
		{"absent catalog key", strings.Replace(testFile(rowA), "# catalog: cdc\n", "", 1),
			"milestones: cdc.csv:10: column catalog: provenance key not present", ""},
		{"bad retrieved", strings.Replace(testFile(rowA), "2026-11-24", "yesterday", 1),
			"milestones: cdc.csv:5: column retrieved: not a YYYY-MM-DD date", "yesterday"},
		{"bad origin url", strings.Replace(testFile(rowA), "https://www.cdc.gov/act-early/milestones/index.html", "http://www.cdc.gov/x", 1),
			"milestones: cdc.csv:6: column origin_url: not an https URL", "http://www.cdc.gov/x"},
		{"empty omitted entry", strings.Replace(testFile(rowA), "tips; images", "tips; ", 1),
			"milestones: cdc.csv:10: column omitted: is empty", ""},
		{"empty file", "", "milestones: cdc.csv:1: column id: header is not id,age_months,domain,text,source_url,retired", ""},
		{"no header", testProvenance, "milestones: cdc.csv:11: column id: header is not id,age_months,domain,text,source_url,retired", ""},
		{"wrong header", testProvenance + "id,age,domain,text,source_url,retired\n" + rowA + "\n",
			"milestones: cdc.csv:11: column age_months: header is not id,age_months,domain,text,source_url,retired", ""},
		{"no items", testFile()[:len(testProvenance)+len(testHeader)],
			"milestones: cdc.csv:11: column id: no items", ""},
		{"short record", testFile(rowA, "2m-se-x,2,social_emotional"),
			"milestones: cdc.csv:13: column record: wrong number of fields", ""},
		{"empty id", testFile(row("", "2", "social_emotional", "Text", goodURL, "")),
			"milestones: cdc.csv:12: column id: is empty", ""},
		{"duplicate id", testFile(rowA, row("2m-se-calms-down", "2", "social_emotional", "Other text", goodURL, "")),
			"milestones: cdc.csv:13: column id: duplicate id", ""},
		{"id with a capital", testFile(row("2m-se-Calms", "2", "social_emotional", "Text", goodURL, "")),
			"milestones: cdc.csv:12: column id: not <n>m-<se|lc|cg|mp>-<slug of a-z, 0-9 and hyphens>", "Calms"},
		{"id with an unknown domain code", testFile(row("2m-xx-calms", "2", "social_emotional", "Text", goodURL, "")),
			"milestones: cdc.csv:12: column id: not <n>m-<se|lc|cg|mp>-<slug of a-z, 0-9 and hyphens>", ""},
		{"id with a three-digit age", testFile(row("120m-se-calms", "2", "social_emotional", "Text", goodURL, "")),
			"milestones: cdc.csv:12: column id: not <n>m-<se|lc|cg|mp>-<slug of a-z, 0-9 and hyphens>", ""},
		{"id with an over-long slug", testFile(row("2m-se-"+strings.Repeat("a", 49), "2", "social_emotional", "Text", goodURL, "")),
			"milestones: cdc.csv:12: column id: not <n>m-<se|lc|cg|mp>-<slug of a-z, 0-9 and hyphens>", ""},
		{"id with another age", testFile(row("4m-se-calms", "2", "social_emotional", "Text", goodURL, "")),
			"milestones: cdc.csv:12: column id: does not start with its age_months", ""},
		{"id prefix that only contains the age", testFile(row("12m-se-calms", "2", "social_emotional", "Text", goodURL, "")),
			"milestones: cdc.csv:12: column id: does not start with its age_months", ""},
		{"age outside the twelve", testFile(row("3m-se-calms", "3", "social_emotional", "Text", goodURL, "")),
			"milestones: cdc.csv:12: column age_months: not one of the catalog ages", ""},
		{"age with a leading zero", testFile(row("2m-se-calms", "02", "social_emotional", "Text", goodURL, "")),
			"milestones: cdc.csv:12: column age_months: not one of the catalog ages", ""},
		{"age that is not a number", testFile(row("2m-se-calms", "two", "social_emotional", "Text", goodURL, "")),
			"milestones: cdc.csv:12: column age_months: not one of the catalog ages", "two"},
		{"unknown domain", testFile(row("2m-se-calms", "2", "social", "Text", goodURL, "")),
			"milestones: cdc.csv:12: column domain: unknown domain", ""},
		{"id code of another domain", testFile(row("2m-cg-calms", "2", "social_emotional", "Text", goodURL, "")),
			"milestones: cdc.csv:12: column id: domain code does not match the domain column", "calms"},
		{"text that is not UTF-8", testFile(row("2m-se-calms", "2", "social_emotional", "Sees\xffa toy", goodURL, "")),
			"milestones: cdc.csv:12: column text: not valid UTF-8", "Sees"},
		{"empty text", testFile(row("2m-se-calms", "2", "social_emotional", "", goodURL, "")),
			"milestones: cdc.csv:12: column text: is empty", ""},
		{"text over the cap", testFile(row("2m-se-calms", "2", "social_emotional", longText, goodURL, "")),
			"milestones: cdc.csv:12: column text: longer than 200 characters", longText},
		{"text with a control character", testFile(row("2m-se-calms", "2", "social_emotional", "\"Sees\ta toy\"", goodURL, "")),
			"milestones: cdc.csv:12: column text: contains a control character", "Sees"},
		{"text with a line break", testFile(row("2m-se-calms", "2", "social_emotional", "\"Sees\na toy\"", goodURL, ""), rowB),
			"milestones: cdc.csv:12: column text: contains a control character", "Sees"},
		{"source outside cdc.gov", testFile(row("2m-se-calms", "2", "social_emotional", "Text", "https://example.org/2-months.html", "")),
			"milestones: cdc.csv:12: column source_url: not under https://www.cdc.gov/", "example.org"},
		{"source over http", testFile(row("2m-se-calms", "2", "social_emotional", "Text", "http://www.cdc.gov/2-months.html", "")),
			"milestones: cdc.csv:12: column source_url: not under https://www.cdc.gov/", ""},
		{"source on a lookalike host", testFile(row("2m-se-calms", "2", "social_emotional", "Text", "https://www.cdc.gov.example.org/x", "")),
			"milestones: cdc.csv:12: column source_url: not under https://www.cdc.gov/", ""},
		{"retired year only", testFile(row("2m-se-calms", "2", "social_emotional", "Text", goodURL, "2022")),
			"milestones: cdc.csv:12: column retired: not empty or a YYYY-MM-DD date", ""},
		{"retired month 13", testFile(row("2m-se-calms", "2", "social_emotional", "Text", goodURL, "2026-13-01")),
			"milestones: cdc.csv:12: column retired: not empty or a YYYY-MM-DD date", "2026-13-01"},
		{"retired free text", testFile(row("2m-se-calms", "2", "social_emotional", "Text", goodURL, "dropped in the update")),
			"milestones: cdc.csv:12: column retired: not empty or a YYYY-MM-DD date", "dropped"},
		{"retired without zero padding", testFile(row("2m-se-calms", "2", "social_emotional", "Text", goodURL, "2026-1-5")),
			"milestones: cdc.csv:12: column retired: not empty or a YYYY-MM-DD date", ""},
		{"second row reported at its own line", testFile(rowA, rowB, row("2m-se-calms", "2", "social_emotional", "", goodURL, "")),
			"milestones: cdc.csv:14: column text: is empty", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parse("cdc.csv", strings.NewReader(tc.in))
			if err == nil || err.Error() != tc.want {
				t.Fatalf("parse error = %v, want %q", err, tc.want)
			}
			if tc.leak != "" && strings.Contains(err.Error(), tc.leak) {
				t.Errorf("error carries the cell %q", tc.leak)
			}
		})
	}
}

func TestParseAcceptsTheTextCap(t *testing.T) {
	text := strings.Repeat("é", 200)
	if _, err := parse("cdc.csv", strings.NewReader(testFile(row("2m-se-calms", "2", "social_emotional", text, goodURL, "")))); err != nil {
		t.Errorf("200-character text refused: %v", err)
	}
}
