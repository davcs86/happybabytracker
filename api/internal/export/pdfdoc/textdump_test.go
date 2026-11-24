package pdfdoc

import (
	"fmt"
	"testing"

	"github.com/davcs86/happybabytracker/api/internal/export/exporttest"
)

// Text and pageTexts are the shared extractor (P-06).
type Text = exporttest.Text

var pageTexts = exporttest.PageTexts

const ptPerMM = exporttest.PtPerMM

func TestExtractor(t *testing.T) {
	d := New(fixedTime, func(int) string { return "foot" })
	d.Heading("Zoë (a\\b) Ольга Ελένη")
	d.Line("Ċ name", "ĉč value")
	pdf := render(t, d)
	pages := pageTexts(t, pdf)
	if len(pages) != 1 {
		t.Fatalf("pages = %d, want 1", len(pages))
	}
	want := []string{"Zoë (a\\b) Ольга Ελένη", "Ċ name", "ĉč value", "foot · Page 1 of 1"}
	got := strs(pages[0])
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("texts = %q, want %q", got, want)
	}
	h := pages[0][0]
	if h.X < 12 || h.X > 14 || h.Y < 12 || h.Y > 20 {
		t.Fatalf("heading at (%.2f, %.2f), want near the top-left margin", h.X, h.Y)
	}
}

var strs = exporttest.Strings
