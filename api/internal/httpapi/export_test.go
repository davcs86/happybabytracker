package httpapi

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/davcs86/happybabytracker/api/internal/constants"
	"github.com/davcs86/happybabytracker/api/internal/export"
	"github.com/davcs86/happybabytracker/api/internal/export/exporttest"
	"github.com/davcs86/happybabytracker/api/internal/model"
	"github.com/davcs86/happybabytracker/api/internal/units"
)

const (
	exportFrom = "2026-11-01"
	exportTo   = "2026-11-20"
	overflowOf = " more entries in this range are not shown. Export a shorter range to see them."
	noEntries  = "No entries in this range."
	notice     = "Some characters cannot be shown in this PDF and appear as ?"
)

var sectionTitles = []string{"Bottles", "Nursing", "Pumping", "Diapers", "Sleep", "Feeding", "Milestones", "Growth", "Vaccination", "Sickness", "Medication"}

func exportBody(from, to string, modules ...string) map[string]any {
	return map[string]any{"from": from, "to": to, "modules": modules}
}

func (k *kit) exportReq(token, kid string, body any) *httptest.ResponseRecorder {
	k.t.Helper()
	return k.do(http.MethodPost, "/v1/kids/"+kid+"/export", token, body)
}

// exportPDF posts the default range with modules and returns the PDF; the status must be 200.
func (k *kit) exportPDF(token, kid string, modules ...string) []byte {
	k.t.Helper()
	return k.exportRange(token, kid, exportFrom, exportTo, modules...)
}

func (k *kit) exportRange(token, kid, from, to string, modules ...string) []byte {
	k.t.Helper()
	rec := k.exportReq(token, kid, exportBody(from, to, modules...))
	if rec.Code != http.StatusOK {
		k.t.Fatalf("export %v: %d %q", modules, rec.Code, rec.Body.String())
	}
	return rec.Body.Bytes()
}

// pdfStrings is every drawn string, page after page, and the page count.
func pdfStrings(t *testing.T, pdf []byte) ([]string, int) {
	t.Helper()
	var out []string
	pages := exporttest.PageTexts(t, pdf)
	for _, p := range pages {
		out = append(out, exporttest.Strings(p)...)
	}
	return out, len(pages)
}

// headings is the section titles in order; the profile's "Modules" value is not one.
func headings(strs []string) []string {
	var out []string
	for i, s := range strs {
		if slices.Contains(sectionTitles, s) && (i == 0 || strs[i-1] != "Modules") {
			out = append(out, s)
		}
	}
	return out
}

func countOf(strs []string, s string) int {
	n := 0
	for _, x := range strs {
		if x == s {
			n++
		}
	}
	return n
}

func withPrefix(strs []string, prefix string) []string {
	var out []string
	for _, s := range strs {
		if strings.HasPrefix(s, prefix) {
			out = append(out, s)
		}
	}
	return out
}

// wantStrings finds each want as a drawn string or, wrapped at spaces, across consecutive ones.
func wantStrings(t *testing.T, name string, strs []string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !slices.Contains(strs, w) && !wrappedIn(strs, w) {
			t.Errorf("%s: %q missing from %q", name, w, strs)
		}
	}
}

// wrappedIn reports whether consecutive strings, joined by spaces, equal w.
func wrappedIn(strs []string, w string) bool {
	for i := range strs {
		line := strs[i]
		for j := i + 1; len(line) < len(w) && j < len(strs); j++ {
			line += " " + strs[j]
		}
		if line == w {
			return true
		}
	}
	return false
}

func wantNoSubstring(t *testing.T, name string, strs []string, banned ...string) {
	t.Helper()
	all := strings.Join(strs, "\n")
	for _, b := range banned {
		if strings.Contains(all, b) {
			t.Errorf("%s: %q present", name, b)
		}
	}
}

func TestExportValidation(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	path := "/v1/kids/" + kid + "/export"
	ok := `"modules":["bottles"]`
	for name, c := range map[string]struct{ body, field string }{
		"from missing":                 {`{"to":"2026-11-20",` + ok + `}`, "from"},
		"from malformed":               {`{"from":"2026-11-1","to":"2026-11-20",` + ok + `}`, "from"},
		"from no date":                 {`{"from":"2026-02-30","to":"2026-11-20",` + ok + `}`, "from"},
		"from before birth":            {`{"from":"2026-03-03","to":"2026-11-20",` + ok + `}`, "from"},
		"to missing":                   {`{"from":"2026-11-01",` + ok + `}`, "to"},
		"to malformed":                 {`{"from":"2026-11-01","to":"20-11-2026",` + ok + `}`, "to"},
		"to before from":               {`{"from":"2026-11-02","to":"2026-11-01",` + ok + `}`, "to"},
		"to after today":               {`{"from":"2026-11-01","to":"2026-11-21",` + ok + `}`, "to"},
		"94 days":                      {`{"from":"2026-08-19","to":"2026-11-20",` + ok + `}`, "to"},
		"from bad before to":           {`{"from":"x","to":"y","modules":[]}`, "from"},
		"to bad before mods":           {`{"from":"2026-11-01","to":"y","modules":[]}`, "to"},
		"modules missing":              {`{"from":"2026-11-01","to":"2026-11-20"}`, "modules"},
		"modules unknown":              {`{"from":"2026-11-01","to":"2026-11-20","modules":["weather"]}`, "modules"},
		"modules case":                 {`{"from":"2026-11-01","to":"2026-11-20","modules":["Bottles"]}`, "modules"},
		"modules path":                 {`{"from":"2026-11-01","to":"2026-11-20","modules":["../x"]}`, "modules"},
		"modules duplicate":            {`{"from":"2026-11-01","to":"2026-11-20","modules":["bottles","sleep","bottles"]}`, "modules"},
		"modules empty":                {`{"from":"2026-11-01","to":"2026-11-20","modules":[]}`, "modules"},
		"modules string":               {`{"from":"2026-11-01","to":"2026-11-20","modules":"bottles"}`, "modules"},
		"modules number":               {`{"from":"2026-11-01","to":"2026-11-20","modules":3}`, "modules"},
		"modules null":                 {`{"from":"2026-11-01","to":"2026-11-20","modules":null}`, "modules"},
		"modules object":               {`{"from":"2026-11-01","to":"2026-11-20","modules":{"bottles":true}}`, "modules"},
		"modules number el":            {`{"from":"2026-11-01","to":"2026-11-20","modules":[1]}`, "modules"},
		"modules null el":              {`{"from":"2026-11-01","to":"2026-11-20","modules":["bottles",null]}`, "modules"},
		"modules nested":               {`{"from":"2026-11-01","to":"2026-11-20","modules":[["bottles"]]}`, "modules"},
		"modules sick case":            {`{"from":"2026-11-01","to":"2026-11-20","modules":["Sickness"]}`, "modules"},
		"modules medication duplicate": {`{"from":"2026-11-01","to":"2026-11-20","modules":["medication","sickness","medication"]}`, "modules"},
	} {
		wantInvalid2(t, name, k.do(http.MethodPost, path, ana, c.body), c.field)
	}
	wantUnknownField(t, k.do(http.MethodPost, path, ana, `{"from":"nope","bogus":1}`), "bogus")
	wantErr(t, k.do(http.MethodPost, path, ana, `{"from":20261101,"to":"2026-11-20",`+ok+`}`), http.StatusBadRequest, "bad_request")
	wantInvalid2(t, "to null", k.do(http.MethodPost, path, ana, `{"from":"2026-11-01","to":null,`+ok+`}`), "to")
	wantErr(t, k.do(http.MethodPost, path, ana, `{"from":"2026-11-01","to":["2026-11-20"],`+ok+`}`), http.StatusBadRequest, "bad_request")
	wantErr(t, k.do(http.MethodPost, path, ana, `not json`), http.StatusBadRequest, "bad_request")

	for _, ids := range [][]string{{"bottles"}, {"sickness"}, {"bottles", "medication"},
		{"medication", "sickness", "vaccination", "growth", "milestones", "feeding", "sleep", "diapers", "pumping", "nursing", "bottles"}} {
		k.exportRange(ana, kid, "2026-08-20", "2026-11-20", ids...)
	}
	k.exportRange(ana, kid, "2026-03-04", "2026-03-04", "bottles")
}

func TestExportMatrixProbeIsUnprocessable(t *testing.T) {
	k := newKit(t)
	kid, ana, bob := k.trackerKid()
	for _, token := range []string{ana, bob} {
		wantUnknownField(t, k.exportReq(token, kid, matrixBody), "matrix_probe")
	}
}

func TestExportUnavailable(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	failing := func() error { return errors.New("embedded data unavailable") }

	saved := loadFonts
	loadFonts = failing
	for _, ids := range [][]string{{"bottles"}, {"growth"}, {"vaccination"}} {
		wantErr(t, k.exportReq(ana, kid, exportBody(exportFrom, exportTo, ids...)), http.StatusServiceUnavailable, "unavailable")
	}
	wantInvalid2(t, "validation first", k.exportReq(ana, kid, exportBody(exportFrom, exportTo)), "modules")
	loadFonts = saved

	savedV := loadVaccines
	loadVaccines = failing
	k.exportPDF(ana, kid, "bottles", "growth", "milestones")
	wantErr(t, k.exportReq(ana, kid, exportBody(exportFrom, exportTo, "bottles", "vaccination")), http.StatusServiceUnavailable, "unavailable")
	loadVaccines = savedV

	savedM := loadMilestones
	loadMilestones = failing
	k.exportPDF(ana, kid, "bottles", "growth", "vaccination")
	wantErr(t, k.exportReq(ana, kid, exportBody(exportFrom, exportTo, "milestones")), http.StatusServiceUnavailable, "unavailable")
	loadMilestones = savedM

	k.exportPDF(ana, kid, "milestones", "vaccination")
}

// filterKid holds an entry of every daily tracker, a measurement, a US schedule, a custom milestone, a symptom, a course
// and a dose.
func (k *kit) filterKid() (kid, ana string) {
	k.t.Helper()
	ana = k.asParent("ana")
	kid = k.createKid(ana, map[string]any{"vaccine_schema": "us"})
	at := time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC)
	for _, typ := range []string{"bottle", "nursing", "pumping", "sleep", "diaper", "meal"} {
		k.seedEntry(kid, "ana", typ, at)
	}
	k.createGrowth(ana, kid, nil)
	k.reach(ana, kid, map[string]any{"text": "Waves bye", "reached_on": "2026-10-01"})
	k.seedEntry(kid, "ana", "symptom", at)
	course := k.createMedication(ana, kid, medBody(map[string]any{"name": "Zentrolex"}))["id"].(string)
	k.createDose(ana, kid, doseBody(course, nil))
	return kid, ana
}

func TestExportModuleFilter(t *testing.T) {
	k := newKit(t)
	kid, ana := k.filterKid()

	strs, _ := pdfStrings(t, k.exportPDF(ana, kid, "growth", "bottles"))
	if got := headings(strs); !slices.Equal(got, []string{"Bottles", "Growth"}) {
		t.Errorf("headings = %q, want only the requested sections", got)
	}
	wantStrings(t, "growth and bottles", strs, "Growth source", "Modules", "Bottles, Growth", "Bottle 120 ml, formula")
	wantNoSubstring(t, "growth and bottles", strs, "Vaccination schedule", "Centers for Disease Control", "Waves bye", "Learn the Signs",
		"Diaper wet", "Nursing", "Pumping", "Sleep", "Solids")

	strs, _ = pdfStrings(t, k.exportPDF(ana, kid, "sleep", "diapers", "bottles"))
	if got := headings(strs); !slices.Equal(got, []string{"Bottles", "Diapers", "Sleep"}) {
		t.Errorf("headings = %q, want model.Modules order whatever the request's", got)
	}
	wantNoSubstring(t, "trackers only", strs, "Growth source", "Vaccination schedule")

	strs, _ = pdfStrings(t, k.exportPDF(ana, kid, "medication", "bottles"))
	if got := headings(strs); !slices.Equal(got, []string{"Bottles", "Medication"}) {
		t.Errorf("headings = %q, want bottles then medication", got)
	}
	wantNoSubstring(t, "medication and bottles", strs, "Cough", "Diaper wet")

	strs, _ = pdfStrings(t, k.exportPDF(ana, kid, reversed(constants.ExportModules)...))
	if got := headings(strs); !slices.Equal(got, sectionTitles) {
		t.Errorf("every module in reverse: headings %q, want model.Modules order", got)
	}

	strs, _ = pdfStrings(t, k.exportPDF(ana, kid, "vaccination"))
	wantStrings(t, "vaccination", strs, "Vaccination schedule", "Centers for Disease Control and Prevention (CDC), CDC Child and Adolescent Immunization Schedule by Age, 2025 (addendum updated July 2, 2025)")
	wantNoSubstring(t, "vaccination", strs, "Growth source", "Bottle 120 ml")

	rows := map[string]string{
		"bottles": "Bottle 120 ml, formula", "nursing": "Nursing L 1 min · R 1 min", "pumping": "Pumping 10 min, 120 ml (both)",
		"diapers": "Diaper wet", "sleep": "Sleep until 2026-11-18 10:30 (30 min)", "feeding": "Solids: pear",
		"growth": "Mon 1 Jun 2026", "milestones": "Waves bye", "vaccination": "Hepatitis B, dose 1",
		"sickness": "Cough", "medication": "Zentrolex",
	}
	for _, m := range constants.ExportModules {
		strs, _ := pdfStrings(t, k.exportPDF(ana, kid, m))
		want := []string{sectionTitle(m)}
		wantStrings(t, m, strs, rows[m])
		for other, row := range rows {
			if other != m {
				wantNoSubstring(t, m, strs, row)
			}
		}
		if got := headings(strs); !slices.Equal(got, want) {
			t.Errorf("%s alone: headings %q, want %q", m, got, want)
		}
	}
}

// TestExportModulesMatchModel pins R3: both lists are kept, and an export accepts every module in model.Modules order.
func TestExportModulesMatchModel(t *testing.T) {
	ids := make([]string, len(model.Modules))
	for i, m := range model.Modules {
		ids[i] = string(m)
	}
	if !slices.Equal(constants.ExportModules, ids) {
		t.Errorf("ExportModules = %q, want model.Modules %q", constants.ExportModules, ids)
	}
	if len(sectionTitles) != len(ids) {
		t.Errorf("%d section titles for %d modules", len(sectionTitles), len(ids))
	}
}

func reversed(ids []string) []string {
	out := slices.Clone(ids)
	slices.Reverse(out)
	return out
}

func sectionTitle(m string) string {
	return sectionTitles[slices.Index(constants.ExportModules, m)]
}

func TestExportSwitchedOffModule(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	k.setModule(ana, kid, "pumping", false)
	k.seedEntry(kid, "ana", "pumping", time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC))
	strs, _ := pdfStrings(t, k.exportPDF(ana, kid, "pumping"))
	wantStrings(t, "switched off", strs, "Pumping", "Pumping 10 min, 120 ml (both)")
}

func TestExportHeaders(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	rec := k.exportReq(ana, kid, exportBody(exportFrom, exportTo, "bottles"))
	h := rec.Header()
	if rec.Code != http.StatusOK || h.Get("Content-Type") != "application/pdf" ||
		h.Get("Content-Disposition") != `attachment; filename="happybabytracker-2026-11-01-2026-11-20.pdf"` ||
		h.Get("Cache-Control") != "no-store" || h.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("got %d %v", rec.Code, h)
	}
	if b := rec.Body.Bytes(); !bytes.HasPrefix(b, []byte("%PDF-1.")) || !bytes.HasSuffix(b, []byte("%%EOF\n")) {
		t.Errorf("body is not a whole PDF: %q…", b[:min(len(b), 16)])
	}
	if bytes.Contains(rec.Body.Bytes(), []byte(testKidName)) {
		t.Error("the kid's name appears in the raw bytes (metadata or an unencoded string)")
	}
}

func TestExportRoles(t *testing.T) {
	roleModules := []string{"bottles", "sickness", "medication"}
	k := newKit(t)
	kid, ana, bob := k.trackerKid()
	k.exportPDF(ana, kid, roleModules...)
	k.exportPDF(bob, kid, roleModules...)

	x := k.asParent("xena")
	noGrant := k.exportReq(x, kid, exportBody(exportFrom, exportTo, roleModules...))
	unknown := k.exportReq(x, model.NewID(), exportBody(exportFrom, exportTo, roleModules...))
	wantErr(t, noGrant, http.StatusNotFound, "not_found")
	if unknown.Code != noGrant.Code || unknown.Body.String() != noGrant.Body.String() {
		t.Errorf("unknown kid %d %q differs from no grant %d %q", unknown.Code, unknown.Body.String(), noGrant.Code, noGrant.Body.String())
	}
	wantErr(t, k.exportReq(k.asAdmin(), kid, exportBody(exportFrom, exportTo, roleModules...)), http.StatusForbidden, "forbidden")

	k.asParent("cara")
	k.grant(ana, kid, "cara", "caregiver")
	cara := k.asParent("cara")
	k.exportPDF(cara, kid, roleModules...)
	if rec := k.do(http.MethodDelete, "/v1/kids/"+kid+"/access/"+k.parentID("cara"), ana, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("revoke cara: %d %q", rec.Code, rec.Body.String())
	}
	sameResponse(t, "revoked caregiver", k.exportReq(cara, kid, exportBody(exportFrom, exportTo, roleModules...)), noGrant)

	if err := k.Store.SetParentDisabled(context.Background(), k.parentID("bob"), true, t0); err != nil {
		t.Fatal(err)
	}
	// Disabling ends the sessions, so a disabled caregiver never reaches the kid lookup.
	wantErr(t, k.exportReq(bob, kid, exportBody(exportFrom, exportTo, roleModules...)), http.StatusUnauthorized, "no_session")
	wantErr(t, k.exportReq(k.asParent("bob"), kid, exportBody(exportFrom, exportTo, roleModules...)), http.StatusUnauthorized, "no_session")
}

func TestExportBusy(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	exportSlots <- struct{}{}
	wantErr(t, k.exportReq(ana, kid, exportBody(exportFrom, exportTo, "bottles")), http.StatusServiceUnavailable, "busy")
	wantInvalid2(t, "validation before the slot", k.exportReq(ana, kid, exportBody(exportFrom, exportTo)), "modules")
	<-exportSlots
	k.exportPDF(ana, kid, "bottles")
	k.exportPDF(ana, kid, "bottles")
	if len(exportSlots) != 0 {
		t.Errorf("%d slots still held after the exports", len(exportSlots))
	}
}

func TestExportEmptyKid(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	pdf := k.exportPDF(ana, kid, constants.ExportModules...)
	strs, pages := pdfStrings(t, pdf)
	if pages != 2 {
		t.Errorf("an empty export of every module has %d pages, want 2", pages)
	}
	if got := headings(strs); !slices.Equal(got, sectionTitles) {
		t.Errorf("headings = %q", got)
	}
	for _, empty := range []string{"No milestones recorded", "No vaccinations recorded."} {
		if n := countOf(strs, empty); n != 1 {
			t.Errorf("%q appears %d times, want once", empty, n)
		}
	}
	if n := countOf(strs, noEntries); n != 8 {
		t.Errorf("%q appears %d times, want once per daily tracker, sickness and medication", noEntries, n)
	}
	if n := countOf(strs, "No measurements recorded."); n != 1 {
		t.Errorf("the empty growth text appears %d times, want once", n)
	}
	wantStrings(t, "profile", strs, "Profile", testKidName, "2026-03-04", "Female", "Europe/Madrid", "8 months 16 days",
		"2026-11-01 to 2026-11-20", "Bottles, Nursing, Pumping, Diapers, Sleep, Feeding, Milestones, Growth, Vaccination, Sickness, Medication",
		"No schedule chosen", "WHO Child Growth Standards (2006) and Growth Reference (2007)",
		"Generated 2026-11-20 13:00 (Europe/Madrid) by happybabytracker · Page 1 of 2",
		"Generated 2026-11-20 13:00 (Europe/Madrid) by happybabytracker · Page 2 of 2")
	wantNoSubstring(t, "empty", strs, notice, "Day total")
}

// trackersKid seeds one entry of each tracker with distinctive values through the API, as ana.
func (k *kit) trackersKid() (kid, ana, bob string) {
	k.t.Helper()
	kid, ana, bob = k.trackerKid()
	day := func(h, m int) string { return ts(time.Date(2026, 11, 18, h, m, 0, 0, time.UTC)) }
	k.createBottle(ana, kid, map[string]any{"at": day(7, 0), "volume": volumeBody("floz", 400), "note": "warm"})
	k.do(http.MethodPost, "/v1/kids/"+kid+"/bottles", ana, advancedBody(map[string]any{"at": day(7, 30),
		"sources": sourcesBody("ml", map[string]int{model.ContentBreastMilk: 80, model.ContentCowMilk: 45})}))
	k.createNursing(ana, kid, weighedBody(map[string]any{"started_at": day(8, 0), "ended_at": day(8, 20), "left_seconds": 600, "right_seconds": 0}))
	k.createPumping(ana, kid, map[string]any{"started_at": day(9, 0), "volume": nil, "side": nil,
		"sides": sourcesBody("ml", map[string]int{"left": 70, "right": 55})})
	k.createPumping(ana, kid, map[string]any{"started_at": day(9, 30), "duration_seconds": 600, "volume": volumeBody("ml", 40), "side": "left"})
	k.createSleep(ana, kid, map[string]any{"started_at": day(10, 0), "ended_at": day(11, 45)})
	k.createDiaper(ana, kid, map[string]any{"at": day(12, 0), "kind": "dirty"})
	k.createMeal(ana, kid, map[string]any{"at": day(13, 0), "what": "parsnip", "reaction": "spat it out"})
	if rec := k.do(http.MethodPost, "/v1/kids/"+kid+"/sleep", ana, map[string]any{}); rec.Code != http.StatusCreated {
		k.t.Fatalf("start sleep: %d %q", rec.Code, rec.Body.String())
	}
	if rec := k.do(http.MethodPost, "/v1/kids/"+kid+"/nursing", ana, map[string]any{"side": "right"}); rec.Code != http.StatusCreated {
		k.t.Fatalf("start nursing: %d %q", rec.Code, rec.Body.String())
	}
	return kid, ana, bob
}

func TestExportTrackers(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackersKid()
	strs, _ := pdfStrings(t, k.exportPDF(ana, kid, "bottles", "nursing", "pumping", "diapers", "sleep", "feeding"))
	wantStrings(t, "rows", strs,
		"2026-11-18 08:00", "Bottle 118 ml, formula", "warm",
		"2026-11-18 08:30", "Bottle 125 ml, breast milk 80 ml + cow's milk 45 ml",
		"2026-11-18 09:00", "Nursing L 10 min, weight before 3.4 kg, weight after 3.462 kg, weight gain 62 g, about 60 ml",
		"2026-11-20 13:00", "Nursing, running on R",
		"Pumping 15 min, 125 ml (L 70 ml · R 55 ml)", "Pumping 10 min, 40 ml (left)",
		"2026-11-18 11:00", "Sleep until 2026-11-18 12:45 (1 h 45 min)", "Sleep, running",
		"Diaper dirty", "Solids: parsnip · spat it out",
		"Day total", "2026-11-18", "2 · 243 ml", "1 · 10 min", "2 · 165 ml", "1 (1 dirty)", "1 h 45 min", "1",
	)
	for _, s := range []string{"Bottle ", "Nursing", "Pumping ", "Sleep ", "Diaper ", "Solids:"} {
		if n := len(withPrefix(strs, s)); n == 0 {
			t.Errorf("no row starts with %q", s)
		}
	}
}

func TestExportTwoUnits(t *testing.T) {
	k := newKit(t)
	kid, ana, bob := k.trackersKid()
	if rec := k.do(http.MethodPatch, "/v1/me", bob, map[string]any{"units": map[string]any{"weight": "lb", "volume": "floz", "length": "in"}}); rec.Code != http.StatusOK {
		t.Fatalf("units: %d %q", rec.Code, rec.Body.String())
	}
	before, _ := units.NewWeight(units.UnitKG, 3400)
	after, _ := units.NewWeight(units.UnitKG, 3462)
	milk, _ := units.MilkTransfer(&before, &after)
	imperial := model.Units{Weight: "lb", Length: "in", Volume: "floz", Temp: "c"}

	metric, _ := pdfStrings(t, k.exportPDF(ana, kid, "bottles", "nursing", "pumping"))
	wantStrings(t, "kg/ml", metric, "Bottle 118 ml, formula", "Nursing L 10 min, weight before 3.4 kg, weight after 3.462 kg, weight gain 62 g, about 60 ml")
	wantNoSubstring(t, "kg/ml", metric, " fl oz", " lb ", " oz,")

	lb, _ := pdfStrings(t, k.exportPDF(bob, kid, "bottles", "nursing", "pumping"))
	wantStrings(t, "lb/fl oz", lb, "Bottle 4 fl oz, formula", "Bottle 4.23 fl oz, breast milk 2.71 fl oz + cow's milk 1.52 fl oz",
		"Nursing L 10 min, weight before "+export.FormatWeight(before, imperial)+", weight after "+export.FormatWeight(after, imperial)+
			", "+export.FormatMilk(milk, imperial),
		"Pumping 15 min, 4.23 fl oz (L 2.37 fl oz · R 1.86 fl oz)")
	wantNoSubstring(t, "lb/fl oz", lb, " ml", " kg", " g,")
}

func TestExportDST(t *testing.T) {
	k := newKit(t)
	ana := k.asParent("ana")
	kid := k.createKid(ana, map[string]any{"time_zone": "America/New_York"})
	ctx, actor := context.Background(), k.parentID("ana")
	vol, _ := units.NewVolume(units.UnitML, 100)
	for _, at := range []time.Time{
		time.Date(2026, 11, 1, 4, 30, 0, 0, time.UTC), // 00:30 EDT
		time.Date(2026, 11, 1, 5, 30, 0, 0, time.UTC), // 01:30 EDT
		time.Date(2026, 11, 1, 6, 30, 0, 0, time.UTC), // 01:30 EST
		time.Date(2026, 11, 1, 7, 30, 0, 0, time.UTC), // 02:30 EST
	} {
		if _, err := k.Store.CreateBottle(ctx, kid, actor, model.BottleFeed{At: at, Contents: []string{model.ContentFormula}, Volume: vol}, k.now); err != nil {
			t.Fatal(err)
		}
	}
	strs, _ := pdfStrings(t, k.exportRange(ana, kid, "2026-11-01", "2026-11-01", "bottles"))
	wantStrings(t, "fall back", strs, "2026-11-01 00:30", "2026-11-01 01:30 -04:00", "2026-11-01 01:30 -05:00", "2026-11-01 02:30",
		"4 · 400 ml", "Generated 2026-11-20 07:00 (America/New_York) by happybabytracker · Page 1 of 1")
	if got := withPrefix(strs, "2026-11-01 "); len(got) != 5 || got[0] != "2026-11-01 to 2026-11-01" {
		t.Errorf("times = %q, want the range and the four bottles once each", got)
	}

	ny, _ := time.LoadLocation("America/New_York")
	start, end := time.Date(2026, 3, 8, 0, 0, 0, 0, ny), time.Date(2026, 3, 9, 0, 0, 0, 0, ny)
	if _, err := k.Store.CreateSleep(ctx, kid, actor, model.SleepSession{StartedAt: start, EndedAt: &end}, k.now); err != nil {
		t.Fatal(err)
	}
	strs, _ = pdfStrings(t, k.exportRange(ana, kid, "2026-03-08", "2026-03-08", "sleep"))
	wantStrings(t, "spring forward", strs, "2026-03-08 00:00", "Sleep until 2026-03-09 00:00 (23 h)", "23 h")
}

func TestExportCaps(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	ctx, actor := context.Background(), k.parentID("ana")
	vol, _ := units.NewVolume(units.UnitML, 60)
	madrid, _ := time.LoadLocation("Europe/Madrid")
	first := time.Date(2026, 8, 20, 0, 0, 0, 0, madrid)
	add := func(at time.Time) {
		if _, err := k.Store.CreateBottle(ctx, kid, actor, model.BottleFeed{At: at, Contents: []string{model.ContentFormula}, Volume: vol}, k.now); err != nil {
			t.Fatal(err)
		}
	}
	for d := range 93 {
		for i := range 20 {
			add(first.AddDate(0, 0, d).Add(time.Duration(i) * time.Hour))
		}
	}
	strs, _ := pdfStrings(t, k.exportRange(ana, kid, "2026-08-20", "2026-11-20", "bottles"))
	if n := len(withPrefix(strs, "Bottle ")); n != 1860 {
		t.Errorf("1860 bottles printed %d rows", n)
	}
	wantNoSubstring(t, "1860", strs, overflowOf)
	if n := countOf(strs, "20 · 1200 ml"); n != 93 {
		t.Errorf("%d day totals of 20 bottles, want 93", n)
	}

	for i := range 3001 - 1860 {
		add(first.Add(time.Duration(i)*time.Minute + 30*time.Second))
	}
	strs, _ = pdfStrings(t, k.exportRange(ana, kid, "2026-08-20", "2026-11-20", "bottles", "diapers"))
	if n := len(withPrefix(strs, "Bottle ")); n != 3000 {
		t.Errorf("3001 bottles printed %d rows, want 3000", n)
	}
	if n := countOf(strs, "1"+overflowOf); n != 1 {
		t.Errorf("the overflow line appears %d times, want once", n)
	}
	wantStrings(t, "after the cap", strs, "Diapers", noEntries)
}

func TestExportNames(t *testing.T) {
	k := newKit(t)
	ana := k.asParent("ana")
	for _, c := range []struct{ name, printed string }{
		{"Zoë Müller-Señor", "Zoë Müller-Señor"}, {"Łucja Ștefan", "Łucja Ștefan"}, {"Ольга", "Ольга"}, {"Ελένη", "Ελένη"},
	} {
		kid := k.createKid(ana, map[string]any{"name": c.name})
		strs, _ := pdfStrings(t, k.exportPDF(ana, kid, "bottles"))
		wantStrings(t, c.name, strs, c.printed)
		wantNoSubstring(t, c.name, strs, notice)
	}
	for _, c := range []struct{ name, printed string }{{"Mia 😀", "Mia ?"}, {"李小明", "???"}} {
		kid := k.createKid(ana, map[string]any{"name": c.name})
		strs, _ := pdfStrings(t, k.exportPDF(ana, kid, "bottles"))
		wantStrings(t, c.name, strs, c.printed, notice)
		if n := countOf(strs, notice); n != 1 {
			t.Errorf("%s: the notice appears %d times, want once", c.name, n)
		}
	}

	kid := k.createKid(ana, nil)
	ctx, actor := context.Background(), k.parentID("ana")
	long := strings.Repeat("x", 500)
	for i, n := range []string{"a\x01b", "line sep end", "bad\xffbyte", "two\nlines\tand tab", long} {
		note := n
		if _, err := k.Store.CreateDiaper(ctx, kid, actor, model.Diaper{Entry: model.Entry{Note: &note}, At: t0.Add(-time.Duration(i+1) * time.Hour), Kind: "wet"}, k.now); err != nil {
			t.Fatal(err)
		}
	}
	pdf := k.exportPDF(ana, kid, "diapers")
	strs, _ := pdfStrings(t, pdf)
	wantStrings(t, "notes", strs, "a?b", "line?sep?end", "bad?byte", "two lines and tab", notice)
	for _, p := range exporttest.PageTexts(t, pdf) {
		for _, s := range p {
			if s.X < constants.ExportMarginMM-0.01 || s.Y > constants.ExportPageHeightUsableMM {
				t.Errorf("%q at (%.1f, %.1f) leaves the printable area", s.S, s.X, s.Y)
			}
			if strings.Contains(s.S, "xxxx") && len(s.S) > 100 {
				t.Errorf("an unbroken note line of %d characters did not wrap", len(s.S))
			}
		}
	}
}

func TestExportNoJudgement(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackersKid()
	deny := regexp.MustCompile(`(?i)\b(late|behind|delayed|delay|normal|abnormal|typical|atypical|on track|ahead|missed|missing|concern|worry|should|red flag|warning)\b`)
	for _, ids := range [][]string{constants.ExportModules, {"bottles"}} {
		strs, _ := pdfStrings(t, k.exportPDF(ana, kid, ids...))
		for _, s := range strs {
			if deny.MatchString(s) {
				t.Errorf("%q judges", s)
			}
		}
	}
	empty := k.createKid(ana, nil)
	strs, _ := pdfStrings(t, k.exportPDF(ana, empty, constants.ExportModules...))
	for _, s := range strs {
		if deny.MatchString(s) {
			t.Errorf("%q judges", s)
		}
	}
}

func TestExportNoValuesInLogs(t *testing.T) {
	k := newKit(t)
	ana := k.asParent("ana")
	kid := k.createKid(ana, map[string]any{"name": "Quillon"})
	k.createBottle(ana, kid, map[string]any{"at": ts(time.Date(2026, 11, 18, 7, 0, 0, 0, time.UTC)), "volume": volumeBody("ml", 639), "note": "xyzzy"})
	k.createSymptom(ana, kid, symptomBody(map[string]any{"symptom": "other", "other_text": "Grommetfizz", "severity": nil}))
	k.createSymptom(ana, kid, symptomBody(feverBody(map[string]any{"temperature": amountBody("c", 3917)})))
	course := k.createMedication(ana, kid, medBody(map[string]any{"name": "Blorptazine", "dose": "Kwazzle drops"}))["id"].(string)
	k.createDose(ana, kid, doseBody(course, map[string]any{"amount": "Snorkwhistle"}))
	k.Log.Reset()
	k.exportPDF(ana, kid, "bottles", "diapers", "sickness", "medication")
	k.exportReq(ana, kid, exportBody(exportFrom, "2026-11-21", "bottles"))
	k.exportReq(ana, kid, exportBody(exportFrom, exportTo, "bottles", "weather"))
	wantCleanLog(t, k.Log.String(), 3, "Quillon", "xyzzy", "639", exportFrom, exportTo, "2026-11-21", "bottles", "diapers", "weather", "modules",
		"Grommetfizz", "39.17", "Blorptazine", "Kwazzle", "Snorkwhistle", "sickness", "medication")
}

// TestGrowthSectionScores: the PDF words each score as the growth screen does from GET …/growth, under the current source.
func TestGrowthSectionScores(t *testing.T) {
	k := newKit(t)
	kid, ana, _ := k.trackerKid()
	k.createGrowth(ana, kid, nil)
	k.createGrowth(ana, kid, map[string]any{"measured_on": "2026-10-01", "weight": amountBody("kg", 7400), "length": amountBody("cm", 680), "head": nil})
	reasons := map[string]string{"before_term": "before expected term", "not_covered": "this source has no table for this age"}
	list := func() []any {
		rec := k.do(http.MethodGet, "/v1/kids/"+kid+"/growth", ana, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("list growth: %d", rec.Code)
		}
		return bodyMap(t, rec)["growth"].([]any)
	}
	check := func(name string) (string, []any) {
		rows := list()
		strs, _ := pdfStrings(t, k.exportPDF(ana, kid, "growth"))
		text := strings.Join(strs, " ")
		n := 0
		for _, r := range rows {
			for _, m := range []string{"weight", "length", "head", "bmi"} {
				sc, ok := r.(map[string]any)["scores"].(map[string]any)[m].(map[string]any)
				if !ok {
					continue
				}
				reason, _ := sc["reason"].(string)
				want := reasons[reason]
				if sc["scored"] == true {
					want = "z " + export.FormatZ(sc["z"].(float64)) + ", percentile " + export.FormatPercentile(sc["percentile"].(float64))
				}
				if !strings.Contains(text, ": "+want) {
					t.Errorf("%s %s: %q missing", name, m, want)
				}
				n++
			}
		}
		if n != 7 {
			t.Fatalf("%s: %d scores checked, want 7", name, n)
		}
		return text, rows
	}
	whoText, whoRows := check("who")
	if rec := k.do(http.MethodPatch, "/v1/kids/"+kid, ana, map[string]any{"growth_source": "cdc"}); rec.Code != http.StatusOK {
		t.Fatalf("switch source: %d %q", rec.Code, rec.Body.String())
	}
	cdcText, cdcRows := check("cdc")
	if whoText == cdcText {
		t.Error("switching the source left the PDF unchanged")
	}
	for i := range whoRows {
		for _, key := range []string{"id", "measured_on", "weight", "length", "head", "note", "age"} {
			if a, b := whoRows[i].(map[string]any)[key], cdcRows[i].(map[string]any)[key]; !equalJSON(a, b) {
				t.Errorf("row %d %s changed with the source: %v -> %v", i, key, a, b)
			}
		}
	}
}

// TestExportSicknessMedication: each parent's file holds the stored temperature column of their unit, and the courses of
// Open value 7 (R1) with the doses in the range.
func TestExportSicknessMedication(t *testing.T) {
	k := newKit(t)
	kid, ana, bob := k.trackerKid()
	if rec := k.do(http.MethodPatch, "/v1/me", bob, map[string]any{"units": map[string]any{"temp": "f"}}); rec.Code != http.StatusOK {
		t.Fatalf("units: %d %q", rec.Code, rec.Body.String())
	}
	day := func(d, h int) string { return ts(time.Date(2026, 11, d, h, 0, 0, 0, time.UTC)) }
	plain := map[string]any{"severity": nil, "note": nil}
	k.createSymptom(ana, kid, withOverrides(symptomBody(feverBody(map[string]any{"at": day(18, 8), "temperature": amountBody("c", 3860)})),
		map[string]any{"severity": nil, "note": "cool cloth"}))
	k.createSymptom(ana, kid, withOverrides(symptomBody(feverBody(map[string]any{"at": day(18, 9), "temperature": amountBody("f", 10130), "method": "ear"})), plain))
	k.createSymptom(ana, kid, symptomBody(map[string]any{"at": day(18, 10), "severity": "moderate", "note": nil}))
	k.createSymptom(ana, kid, symptomBody(map[string]any{"at": day(18, 11), "symptom": "other", "other_text": "hiccups", "severity": nil, "note": nil}))
	k.createSymptom(ana, kid, symptomBody(map[string]any{"at": day(10, 23), "symptom": "rash", "severity": nil, "note": nil}))
	k.createSymptom(ana, kid, symptomBody(map[string]any{"at": ts(time.Date(2026, 10, 31, 9, 0, 0, 0, time.UTC)), "symptom": "vomiting"}))

	c, _ := pdfStrings(t, k.exportPDF(ana, kid, "sickness"))
	f, _ := pdfStrings(t, k.exportPDF(bob, kid, "sickness"))
	wantStrings(t, "°C", c, "2026-11-18 09:00", "Fever 38.6 °C (oral)", "cool cloth", "Fever 38.5 °C (ear)", "Cough, moderate", "Other: hiccups", "Rash")
	wantNoSubstring(t, "°C", c, "°F", "Vomiting")
	wantStrings(t, "°F", f, "Fever 101.48 °F (oral)", "Fever 101.3 °F (ear)")
	wantNoSubstring(t, "°F", f, "°C", "Vomiting")
	if i, j := slices.Index(c, "Rash"), slices.Index(c, "Other: hiccups"); i < 0 || i > j {
		t.Errorf("symptoms are not oldest first: %q", c)
	}

	start := ts(time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC))
	inactive := k.createMedication(ana, kid, medBody(map[string]any{"name": "Zentrolex", "start_at": start, "end_on": "2026-11-02", "active": false}))
	outside := k.createMedication(ana, kid, medBody(map[string]any{"name": "Quorbitan", "start_at": start, "end_on": "2026-10-20", "schedule": everySchedule(6)}))
	k.createMedication(ana, kid, medBody(map[string]any{"name": "Plixamol", "start_at": start, "end_on": "2026-10-20", "schedule": map[string]any{"kind": "none"}}))
	k.createDose(ana, kid, doseBody(outside["id"].(string), map[string]any{"given_at": day(5, 14), "amount": "1.5 ml", "note": "with yogurt"}))
	k.createDose(ana, kid, doseBody(inactive["id"].(string), map[string]any{"given_at": ts(time.Date(2026, 10, 31, 9, 0, 0, 0, time.UTC)), "amount": "9 drops"}))
	strs, _ := pdfStrings(t, k.exportPDF(bob, kid, "medication"))
	wantStrings(t, "medication", strs, "Zentrolex", "08:00, 20:00", "2026-11-02", "Not active", "Quorbitan", "Every 6 hours from 10:00",
		"2026-10-01 10:00", "Given at", "2026-11-05 15:00", "1.5 ml", "with yogurt")
	wantNoSubstring(t, "medication", strs, "Plixamol", "9 drops", "Next", "Last dose")
}
