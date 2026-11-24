# Pure-Go PDF libraries for the export (design 12): research for plan 7 (2026-11-23)

Researcher notes. Every source was retrieved on **2026-11-23**. Measurements were made in a
scratch directory outside the repository with Go 1.27.4, `CGO_ENABLED=0 go build -trimpath
-ldflags="-s -w"`, on linux/amd64. Nothing was added to `api/go.mod` on the branch. Not legal advice.

## 1. Candidates at a glance

| # | Module (latest tag, date) | Status | Licence | Runtime deps of the package used | CGO | Base-14 fonts | Measured |
|---|---|---|---|---|---|---|---|
| 1 | `github.com/go-pdf/fpdf` (`@latest` v0.9.0, 2023-09-13) | **archived** on 2025-03-04: "This repository was archived by the owner on Mar 4, 2025. It is now read-only." README: "**ARCHIVED**. Please use fpdf (codeberg.org/go-pdf/fpdf)" | MIT | n/a | n/a | n/a | not used (archived) |
| 1′ | **`codeberg.org/go-pdf/fpdf` v0.12.0 (2026-05-18)**, the successor | active: last commit 2026-09-14 ("all: bump x/image@v0.46.0, Go-1.26"), 660 commits, 37 open issues; go.mod `go 1.25.0` | MIT ("Copyright (c) 2021 The Go-PDF authors / 2020 David Barnes / 2017 Kurt Jung and contributors") | root package imports the **standard library only** (go list: bufio … unicode, including `crypto/rc4`, `image/*`, `encoding/xml`). The module's go.mod lists barcode, gofpdi, pdf417 and x/image, which only sub-packages use. `go mod tidy` on the API added only `codeberg.org/go-pdf/fpdf v0.12.0` to go.mod and two lines to go.sum | builds with `CGO_ENABLED=0` | yes (Courier, Helvetica, Times, Symbol, ZapfDingbats; widths embedded) | API binary 13,365,408 → 14,000,288 bytes (**+635 KB**, +4.7 %; gzip 5.51 → 5.78 MB); 3000-row table: 57 pages, 109 KB compressed, process VmHWM 11.5 MB; 8000 rows uncompressed: 151 pages, 1.82 MB, VmHWM 17.4 MB, 0.05 s |
| 2 | `github.com/signintech/gopdf` v0.38.6 (2026-11-21; v0.38.4 2026-10-31) | active, not archived, 120 open issues; v0.38.6 is younger than the two-week pin rule | MIT ("Copyright (c) 2015 signintech") | root package imports `github.com/phpdave11/gofpdi` and its own `fontmaker/core`; go.mod `go 1.13` | not stated; nothing found | **no**: every example calls `AddTTFFont` before `SetFont`; only the TTF loaders exist (`AddTTFFont`, `AddTTFFontData`, `AddTTFFontByReader`). A TrueType font (and its licence) must be shipped | not built |
| 3 | `github.com/jung-kurt/gofpdf` v1.16.2 (2019-11-19) | **archived** 2021-11-13; README: "This repository will not be maintained, at least for some unknown duration." | MIT | stdlib | builds with `CGO_ENABLED=0` | yes | demo 3.60 MB vs 1.61 MB empty program; 3000 rows VmHWM 11.0 MB, 109 KB |
| 4 | `github.com/johnfercher/maroto/v2` v2.4.4 (2026-10-22) | active | MIT | built on `github.com/phpdave11/gofpdf` v1.4.3; go.mod requires pdfcpu, go-tree, uuid, barcode, yaml.v3, testify, plus 11 indirect modules (runewidth, uax29, tiff, stringprep, x/crypto, x/image, x/text …) | builds with `CGO_ENABLED=0` | yes (through gofpdf) | demo binary **12.7 MB**; 3000 rows VmHWM **28.9 MB**, output **1.94 MB**; Producer "FPDF 1.7", creation date = wall clock by default |
| 5 | standard library only (own writer) | ours | MIT (ours) | none (a 32-entry Windows-1252 table replaces `golang.org/x/text`, which the prototype used) | n/a | yes, by name: `/Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding` needs no embedding | prototype (~90 lines, text only, no wrapping or tables): 3000 rows → 60 pages, VmHWM 8.2 MB; 8000 rows → 160 pages, 120 KB compressed, VmHWM 17.7 MB; `pdfinfo` reads it, `pdftotext` extracts "Zoë Müller-Señor – export" |

Empty-program baseline for the demo sizes: 1,605,792 bytes. The meaningful size figure is the API
delta (+635 KB) for option 1′.

## 2. Findings that decide

- **Option 1 as named in the task (`github.com/go-pdf/fpdf`) is archived.** Its maintained
  continuation is `codeberg.org/go-pdf/fpdf`. The module path changes, and the code is hosted on
  Codeberg, not GitHub. Go's module proxy serves it (v0.12.0, `h1:g8E/1VqGqB2lZUUaqQrrTnA0IEJLPTTX1DZ0qS/ZmhU=`).
  The `github.com/go-pdf/fpdf` go.mod retracts v1.4.2 and v1.4.3 ("v1.4.2 was imported from
  phpdave11/gofpdf by mistake").
- **The successor links into the API with no transitive module** and no CGO, and adds 0.6 MB to a
  13.4 MB binary. Its roadmap line, quoted from the Codeberg page, is a risk to watch: "Remove all
  legacy code page font support; use UTF-8 exclusively". A future release may drop the core-font
  code-page path this plan uses. The pin and a golden test catch that at bump time.
- **Determinism is supported by API**, and the API was checked in the source: `SetCatalogSort(bool)`,
  `SetCreationDate(time.Time)`, `SetModificationDate(time.Time)`, `SetProducer(string, bool)` and
  `SetCompression(bool)`. Two runs with a fixed creation date produced byte-identical files (`cmp`).
  The output writes no `/ID` array. `time.Now()` appears once in `fpdf.go`, as the default when no
  creation date is set, and `math/rand` appears only in `protect.go` (encryption, unused).
- **No network or HTML path is reachable from our use.** In the root package's non-test files there
  is no `net/http`, `os/exec`, `unsafe` or `import "C"`; only `fpdf_example_test.go` imports
  `net/http`. The package does have `HTMLBasicNew` (a tiny tag writer) and image loaders. The plan
  forbids both: text, lines and rectangles only.
- **Non-Latin text with core fonts** goes through `UnicodeTranslatorFromDescriptor("")` (cp1252).
  Measured: `"Zoë Müller-Señor"` and `"Œuvre € — “quotes”"` render correctly. `"李小明"` → `"..."`,
  `"Ольга"` → `"....."`, `"Mia 😀"` → `"Mia ."`, `"Ł ł Ș ș"` → `". . . ."`, so **each unsupported
  rune becomes one `.`**. The plan replaces this with its own fallback character (see the plan's
  question 6).
- **A 500-character unbroken string** in `MultiCell` is force-wrapped across lines with no error
  (`p.Err()` false).
- **Uncompressed output** (`SetCompression(false)`) leaves every cell as a literal `(…)Tj` string, for
  example `BT 144.57 769.74 Td (3000 g)Tj ET`, so a smoke test can find text with `grep -a`. The
  cost is size: 3000 rows took 684 KB instead of 109 KB.
- **Memory.** fpdf builds the whole document in memory and writes it on `Output(w)`. True streaming
  is not available. At ~8000 rows the process high-water mark was 17.4 MB, against the API's 60 MiB
  smoke ceiling and `GOMEMLIMIT=40MiB`. The same ceiling covers two concurrent argon2 hashes
  (~38 MiB, per the Dockerfile comment). So the export needs a row cap and a concurrency limit of one.
- **maroto** is a layout layer over a different gofpdf fork, with about 17 modules, roughly 3.5 times
  the binary and 2.5 times the memory of fpdf for the same table. It is rejected for supply-chain
  weight.
- **gopdf** cannot write with the base-14 fonts, so even English-only text needs a TrueType file in
  the image: a font licence (for example SIL OFL for Noto or DejaVu), more bytes in every PDF and a
  font-embedding path. Its strength, Unicode subset embedding, answers a question the owner has not
  asked yet (question 6). Its newest tag is also too young to pin.
- **jung-kurt/gofpdf** has been archived since 2021 and gets no fixes. Rejected.
- **The standard-library writer** keeps the dependency count at zero. It costs our own text width
  tables for Helvetica and Helvetica-Bold (needed for wrapping and column fitting), wrapping,
  pagination, table drawing, string escaping, the xref table and chart lines. The estimate is
  600–900 lines plus tests: two more implementer tasks than option 1′, and our own PDF bugs to find.
  The prototype shows the file format side is small. Layout is the real work.

- **Later finding, with an embedded TrueType font** (`AddUTF8FontFromBytes`, after the owner's
  answer): a rune above U+FFFF (an emoji) **panics** in `Output`, so the app must filter such runes
  first. Subsetting is automatic. Details and measurements are in `noto-sans-font.md`.

## 3. Licence compatibility

| Item | Licence | Compatible with the repo's MIT code and public images |
|---|---|---|
| `codeberg.org/go-pdf/fpdf` | MIT | yes; keep its notice. The plan adds it to the README's third-party list and copies `LICENSE` into `/usr/share/doc/happybabytracker-api/fpdf/` in the image, the way the WHO and CDC notices are copied |
| Core font metrics inside fpdf (`font_embed/*.json`, `cp1252.map`) | part of the MIT module (width numbers are facts) | yes |
| WHO growth tables shown as percentiles in the PDF | CC BY-NC-SA 3.0 IGO (plan 4) | attribution is required wherever the data is shown, so the PDF prints the attribution and the adaptation disclaimer. Whether a PDF with derived percentiles is an "adaptation" that ShareAlike reaches is **not verified** |
| CDC growth tables, CDC milestones, CDC vaccine schedule | public domain with CDC's conditions | attribution and no-endorsement lines in the PDF |
| UKHSA schedule | OGL v3.0 | the OGL attribution line in the PDF |
| Mexico schedule | facts with a citation (plan 5) | the citation in the PDF |

## 4. Sources (retrieved 2026-11-23)

- https://github.com/go-pdf/fpdf (archive banner, README "ARCHIVED")
- https://codeberg.org/go-pdf/fpdf (status, last commit, roadmap, licence)
- https://proxy.golang.org/codeberg.org/go-pdf/fpdf/@latest, `/@v/v0.12.0.mod`, `/@v/list`
- https://proxy.golang.org/github.com/go-pdf/fpdf/@v/v0.9.0.mod (the retractions)
- https://github.com/signintech/gopdf and https://proxy.golang.org/github.com/signintech/gopdf/@v/v0.38.6.mod, `…/v0.38.4.info`
- https://github.com/jung-kurt/gofpdf (archive banner, README maintenance line, `SetCatalogSort` and `SetCreationDate` for comparisons)
- https://github.com/johnfercher/maroto and https://proxy.golang.org/github.com/johnfercher/maroto/v2/@v/v2.4.4.mod
- Module source read from the local module cache: `codeberg.org/go-pdf/fpdf@v0.12.0/{fpdf.go,embedded.go,protect.go,LICENSE,README.md,font_embed/}`, `github.com/signintech/gopdf@v0.38.6/{gopdf.go,pdf_protection.go,LICENSE}`

## 5. Not verified

- gopdf was not built or measured. Its CGO-free build is assumed from its imports, which include no
  `C` package.
- Whether a future `codeberg.org/go-pdf/fpdf` release drops core-font code pages (only the roadmap
  line was read).
- The supply-chain standing of Codeberg as a host (availability, account security). The module proxy
  cache and `go.sum` protect builds of a pinned version, but not a fresh `go get` of an unpinned one.
- The PDF/A or tagged-PDF (accessibility) status of the output: neither library produces tagged PDF
  through the API used here.
- The memory of a real export inside the 96 MB container with the API's own heap. The figures above
  come from standalone processes.
