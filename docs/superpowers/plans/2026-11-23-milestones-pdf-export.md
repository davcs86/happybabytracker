# Milestones and PDF Export Implementation Plan (plan 7)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status: draft, owner answers of 2026-11-23 applied.** Nothing here is built. No question is open; execution starts only after the owner approves the open values. Start implementation only when the owner asks for it.

## Owner answers (2026-11-23, do not reopen)

1. **PDF library:** add `codeberg.org/go-pdf/fpdf` v0.12.0, pinned in `go.mod` and `go.sum`. It is the maintained successor of the archived `github.com/go-pdf/fpdf`.
2. **Milestone catalog:** all twelve CDC "Learn the Signs. Act Early." checklists, 2 months to 5 years, transcribed verbatim from the HTML age pages under CDC's four reuse conditions.
3. **Export range and module filter:** at most 93 days per export. The export is **filterable by module**: the form has one checkbox per module, the API takes a module list, refuses an unknown id or an empty selection with 422 `modules`, and prints only the chosen modules. The modules switched on for the kid are ticked by default. A switched-off module is offered unticked, and its data is exported when the requester ticks it (rule and cost: Open value 21).
4. **Names in other scripts:** **embed a Unicode font** instead of substituting `?` for everything outside WinAnsi. The font is Noto Sans 2.015 (Regular and Bold), SIL Open Font License 1.1, embedded with `go:embed` under `api/internal/export/fonts/` with `LICENSE` and `NOTICE` (Task 8, Open value 6).

5. **The 24-months-and-over band and how much of the catalog to show**, verbatim: "Keep the module default visibility as it is now. Display the full list of milestones until 5yo. It's the parent/guardian decision to keep it on after the default visibility." In this plan that means:
   - The module default per band stays exactly as spec 7.2: on at 0–11 and 12–23 months, **off from 24 months**. `DefaultEnabled` is not changed.
   - Whether the module stays on past 24 months is the guardian's decision, made on the existing Modules page. Plan 3's band rule (an entry in the 60-day lookback turns it on once as a stored row) may keep it on. No new setting is added.
   - The Milestones page (reached from the quick-log tile while the module is on, and always from the kid links) shows **the full catalog, all twelve lists (2 months to 5 years), until the kid's 5th birthday**, ordered by list age. The kid's current list is marked in text and linked from the top of the page, and every item of every list can be marked reached with a date.
   - From the 5th birthday, only reached, custom and retired-item rows are shown (Open value 3).

## Defaults accepted unless the owner vetoes

- **Record sections:** growth, vaccination and milestones print the kid's whole record up to the range's end date. The daily trackers print only the range (Open value 5).
- **Page size:** A4 pages whose margins, content and footer stay within the top 279.4 mm, so the same file prints at 100 % on US Letter (Open value 8).
- **Sickness and medication:** added by Task 17 after plan 6 merges. Until then the API refuses those module ids (Open value 7).

---

**Goal:** Two parts. Milestones: a guardian or caregiver sees the full CDC developmental milestone catalog (all twelve lists) until the kid's 5th birthday, with the list for the kid's (corrected) age marked, marks a milestone as reached with a date, adds custom free-text milestones, and can change or delete any of them. The app never says a kid is late, behind or anything else that compares or judges. PDF export: a guardian or caregiver exports one kid for a date range of at most 93 days and a chosen set of modules, as a PDF the API renders itself in the requester's units and the kid's time zone. Names and notes in Latin, Greek and Cyrillic scripts print correctly. The PDF is generated on demand and never stored.

**Architecture:** A new package `api/internal/milestones` embeds one CSV (provenance header, stable item ids, as `api/internal/vaccines` does), loads it at start and computes each checklist's position against the kid's age basis. The next free migration adds `milestones_reached`. The routes follow the vaccination pattern: one kid read model and CRUD on reached rows.

The export lives under `api/internal/export`, in three parts:
- `export/fonts` embeds Noto Sans Regular and Bold, with their OFL `LICENSE` and a `NOTICE`, and knows the fonts' glyph coverage.
- `export/pdfdoc` wraps `codeberg.org/go-pdf/fpdf`. It takes the embedded TrueType fonts only (subset automatically into each PDF), and draws text, lines and rectangles: no images, no HTML, no external fetch.
- `export` reads one consistent snapshot of the kid's data (one read transaction) for the chosen modules and lays out the sections.

A new UI page `/kids/{kid}/export` holds a plain form that posts to a `+server.ts` route, which streams the API's response body to the browser as an attachment. The UI also adds a Milestones page, a quick-log tile and kid links.

**Tech Stack:** Go 1.27 (`embed`, `encoding/csv`), SQLite via `modernc.org/sqlite`. **New:** `codeberg.org/go-pdf/fpdf` v0.12.0 (MIT) and the Noto Sans 2.015 font files (SIL OFL 1.1, data, not code). SvelteKit 3 / Svelte 5 runes, Tailwind 4. No new npm dependency.

**Spec:** `docs/superpowers/specs/2026-11-20-happybabytracker-design.md` sections 2 (G6, G7, G8, G12, non-goals), 5.2, 6.1, 6.3 (`milestones_reached`), 6.4, 7.1, 7.2, 7.3, 10.1, 12, 13.3, 14, 16, 17 ("The PDF library and layout"; "The milestone catalog's exact items").

Earlier plans:
- `docs/superpowers/plans/2026-11-22-growth.md`: corrected age, source attribution.
- `2026-11-22-vaccination.md`: embedded data with provenance, generated review documents, the update runbook.
- `2026-11-23-advanced-logging.md`: per-source volumes, weigh-ins and the milk estimate.

Research notes, all retrieved 2026-11-23: `docs/superpowers/research/milestones/{cdc-milestone-checklists,pdf-libraries,noto-sans-font,pdf-text-and-verification}.md`. Part of `pdf-text-and-verification.md` is superseded by `noto-sans-font.md`.

**Dependency on plan 6:** plan 6 (sickness, medication, reminders) is being refreshed in parallel and is not merged. Its draft takes migration 0006, so this plan's migration is **"the next free number"**: 0006 if this plan merges first, 0007 if plan 6 merges first. The `schema_version` assertions change with it. Task 17 waits for plan 6.

## Global Constraints

- Every limit, cap and duration lives in `api/internal/constants/constants.go` (API) or `ui/src/lib/server/constants.ts`, with client-safe mirrors in `ui/src/lib/constants.ts` (CLAUDE.md rule 7). No literal anywhere else, including the PDF's page and margin sizes, font sizes and row caps.
- **No judgement, anywhere the app writes its own words** (spec 10.1, spec 2 non-goals).
  - This covers the Milestones page, the milestones API view, the PDF's profile, milestones and growth sections, and every empty-state or error text.
  - None of these words appear there: `late`, `behind`, `delayed`, `delay`, `normal`, `abnormal`, `typical`, `atypical`, `on track`, `ahead`, `missed`, `missing`, `concern`, `worry`, `should`, `red flag`, `warning`.
  - Nothing is coloured by state, and no list of "not reached" items appears in the PDF.
  - The only status words in the PDF are the vaccination statuses the owner approved in plan 5 (`given`, `due`, `upcoming`, `overdue`, `dismissed`), and only in the vaccination section.
  - CDC's own item text is reproduced verbatim (CDC's conditions forbid changing it). Task 2 checks it against the same list and stops for the owner if an item matches.
- **Milestones use corrected age** where spec 7.1 applies: born before 37 weeks 0 days, until the second birthday. This matches CDC's "born more than 3 weeks prematurely, use … corrected age". Both ages are shown wherever the corrected one is. Vaccination code stays chronological.
- Calendar arithmetic goes only through `kidtime` (`AgeMonths`, `CorrectedBirthDate`, `AddAge`). Dates are `YYYY-MM-DD` in the kid's zone, and instants are printed in the kid's zone.
- A caller with no grant, an unknown kid and a reached row of another kid answer the same 404 body. The admin is refused. Writes re-assert the role with `actorRole` inside `inTx`. Hard deletes, no edit history. Milestone writes and the export are `Caregiver` (spec 5.2: entries and export belong to both roles).
- **Privacy of the export:**
  - It is generated on demand, never written to disk, never cached (`Cache-Control: no-store` on the API and UI responses) and never logged.
  - Logs carry the route, kid id, status and duration only, never a module list, range, row count or byte count. The UI's `logFailure` keeps its fixed shape.
  - The range and the module ids travel in the POST body, never in a URL. The only exception is a validation redirect, which carries only `error`, `from`, `to` and `modules` (Open value 25).
  - The download's file name and the PDF's metadata hold no name.
- **The module filter is the only selector:** the PDF contains exactly the sections of the requested modules, plus the profile block, in a fixed order whatever order the request lists them in. A module that was not requested leaves no heading, no attribution and no row.
- **The PDF writer is closed:**
  - Its only fonts are the two embedded Noto Sans files from `export/fonts`.
  - It draws only text, lines and rectangles. No core font, image, HTML (`HTMLBasicNew`), link, attachment, JavaScript, encryption, file or network read.
  - Every string passes through one app-owned encoder, which keeps a rune only if the embedded fonts have a glyph for it and the rune is in the BMP. Every other rune and every control character becomes `?`, and the profile block then says once that some characters cannot be shown. fpdf panics on runes above U+FFFF, so nothing else may call fpdf with unfiltered text.
- **Determinism:**
  - The PDF's creation and modification dates are the request's `Deps.Now`, and the producer is a fixed string.
  - The catalog is sorted, sections come in a fixed order, and nothing ranges over a map into the output.
  - The font files are pinned by SHA-256 in a test.
  - With a fixed clock, a golden-file test is byte-stable. It changes only when the text, the layout or the fonts change.
- Embedded data lives under `api/internal/milestones/data/` and `api/internal/export/fonts/`, never under `testdata` (which `api/.dockerignore` drops), never with a `*.db*` name. Golden PDFs for tests live in `api/internal/export/testdata/`.
- Comments state constraints only, at most 2 lines (rule 9). `gofmt -l` prints nothing, staticcheck is clean, Prettier uses tabs. Every new page works without JavaScript, passes axe at 390×844 and 1280×800 and keeps `Cache-Control: no-store`. The Milestones page stays reachable with its module switched off; only its tile hides.
- Tests that pin counts change together with the code they pin: `wantTables` and `schema_version`, `routeClasses`, `matrixParams` (`entry` for `/milestones/`), the stub's routes, `quick-log.test.ts` (milestones gets a tile), and the log scans.
- New third-party items (rule 10):
  - fpdf is pinned in `go.mod`/`go.sum`, and the two font files are pinned by hash.
  - The fpdf `LICENSE`, and the fonts' `LICENSE` and `NOTICE`, are copied into the image beside the WHO and CDC notices.
  - The README's third-party list gains both.
  - The image's `org.opencontainers.image.licenses` label gains `OFL-1.1` (Task 16).

## Review Focus

Failure modes that no task's happy path exercises. Each has a test in the named task.

1. **A kid with no data:** every chosen module prints its heading and "No entries in this range." Record sections print "No measurements recorded.", "No doses recorded." or "No milestones recorded." The PDF is still valid and has one page (Tasks 11–13).
2. **Large data:** 3000 growth rows (the `MaxGrowthPerKid` cap) render within the row cap. 93 days of bottles at 20 a day (1860 rows) render, and the overflow line appears exactly once when the cap is passed. The worst case for every module stays under `ExportMaxRows`, `ExportMaxBytes` and the measured memory budget (Tasks 11, 12, 15).
3. **A 500-character unbroken note** wraps inside its column and never overflows the page width or the 279.4 mm usable height (Tasks 9, 11).
4. **Names in other scripts:**
   - "Zoë Müller-Señor", "Łucja Ștefan", "Ольга" and "Ελένη" print exactly and are found by text search.
   - "Mia 😀" prints "Mia ?" and never reaches fpdf (it would panic on a rune above U+FFFF).
   - "李小明" prints "???" and adds the one-line notice.
   - A control character in a note becomes `?`.
   - Tasks 8, 9, 11.
5. **Bad export requests** each answer a 422 naming the field, never a 500 and never a partial PDF (Task 11):
   - `modules`: an unknown or corrupt id, a duplicate id, an empty list, a non-array value, `sickness` or `medication` before Task 17.
   - Anywhere in the body: an unknown field, a malformed date.
   - The range: `to` before `from`, a 94-day range, `to` after the kid-local today, `from` before the birth date.
6. **Module filter:**
   - Requesting only `bottles` and `growth` gives a PDF with exactly those two section headings, and no vaccination caption, milestone attribution or diaper row.
   - Every subset gives the same section order.
   - A module the kid has switched off is exported when requested.
   - The default ticks on the form follow the kid's switches.
   - Tasks 11, 14, 15.
7. **Two parents with different units** export the same kid. The kg/ml/cm parent and the lb-oz/fl oz/in parent get files whose values are each exactly the stored columns of their unit (Tasks 10, 11, 12).
8. **DST days in the kid's zone:** a bottle at 01:30 on a fall-back day in `America/New_York` prints the right offset, and the daily summary counts it once. A spring-forward day sums 23 hours of sleep. Times are printed in the kid's zone, not the server's (Task 11).
9. **Caregiver access:** a caregiver exports and reaches every milestone write. A no-grant parent and an unknown kid get byte-identical 404 bodies on all six new routes. The admin is refused, with a 403 at the API and a redirect at the UI (Tasks 5, 11, 14, 15).
10. **Concurrency and memory:** a second export while one runs answers 503 `busy` and never queues. An export during the smoke's login burst keeps the API under 60 MiB with the fonts parsed (Tasks 11, 15).
11. **The catalog span** (owner 2026-11-23; Tasks 3, 5, 7, 15):
    - A kid at 4 years 11 months gets all twelve lists, with the 4-year list current and the 5-year list next and reachable.
    - On the day before the 5th birthday the full catalog is shown. On the 5th birthday itself (kid-local, civil date) only reached and custom rows are shown, and a reached 5-year item is still listed.
    - A 29 February birth: the 5th birthday is 1 March.
    - In Pacific/Auckland the boundary follows the kid's local date, not UTC.
    - A preterm kid (32 weeks) at 3 months chronological still gets all twelve lists, with the 2-month list current by corrected age and both ages shown. At 2 years 1 month it is chronological again.
    - A kid at 30 months whose module is switched off by default: the page is still reachable (by URL and kid links), shows the full catalog and accepts marks, while the quick-log tile is hidden. `DefaultEnabled` is unchanged (`modules_test.go` still pins off at 24+).
12. **Milestones** (Tasks 1, 3, 4, 5):
    - A catalog item reached twice is a 409, never a second row.
    - A date after the kid-local today or before the birth date is a 422.
    - A custom text of 201 characters, or of spaces only, is a 422.
    - Deleting a reached item puts it back in its checklist.
    - A stored row whose catalog item was retired is still listed with its text.
    - A preterm kid at 3 months chronological and 1 month corrected sees the 2-month list as "next", and both ages are shown.
13. **Band rule:** a kid crossing 24 months with a milestone recorded in the 60-day lookback keeps the module on as a stored override (Task 4).

---

## Open values (owner confirmation before execution)

Rule 1 (never assume an owner value). A row marked "owner" or "accepted" cites a decision already made. Every other row is this plan's proposal, with its basis and the cost if it is wrong.

| # | Value | Decision or proposal | Basis | Cost if wrong |
|---|---|---|---|---|
| 1 | PDF library | `codeberg.org/go-pdf/fpdf` v0.12.0 (2026-05-18), pinned | **owner** 2026-11-23; research `pdf-libraries.md` | — |
| 2 | Catalog edition and scope | CDC LTSAE checklists, all 12 ages (2 months to 5 years), from the HTML pages retrieved on the transcription day, verbatim | **owner** 2026-11-23; research `cdc-milestone-checklists.md` | — |
| 3 | 24+ band and catalog span | **Owner** 2026-11-23. The module default is unchanged (spec 7.2: off from 24 months; `DefaultEnabled` untouched). Keeping it on is the guardian's choice on the existing Modules page; plan 3's 60-day band rule may keep it on, and there is no new setting. The page shows **all twelve lists while today < the 5th birthday**. The 5th birthday is `kidtime.AddAge(birth_date, 5, 'y')`: civil-date arithmetic, compared with the kid-local today as `YYYY-MM-DD` strings, chronological age (never corrected), and a 29 February birth gives 1 March (Go's `AddDate` normalisation). On and after that date, `catalog_shown` is false: no lists are shown, only reached, custom and retired-item rows. The API still accepts a catalog `item` after the 5th birthday (no extra rule), but the page no longer offers unreached items. `MilestoneCatalogUntilYears = 5` | owner; CDC's lists stop at 5 years | — |
| 4 | Range cap | `ExportMaxDays = 93`, inclusive kid-local dates | **owner** 2026-11-23; equal to `SummaryMaxDays` | — |
| 5 | Record sections | growth, vaccination, milestones: everything up to `to`; trackers: `[from, to]` | **accepted** default | three filters |
| 6 | Characters and font | Noto Sans **2.015**, unhinted static TTF: `NotoSans-Regular.ttf` (431,364 B, SHA-256 `f3961a9cde016d41a4879aecda1474d3a36d6bf54fa0e4643de029cc2248b0e8`) and `NotoSans-Bold.ttf` (432,376 B, SHA-256 `87cb2d84472a7d66da659ee47b6cdb9552326e8c128245231f191b6ac72529d9`), from `notofonts/notofonts.github.io` `fonts/NotoSans/unhinted/ttf/`; licence `api/internal/export/fonts/LICENSE` = the OFL 1.1 text of `notofonts/latin-greek-cyrillic` `OFL.txt` (SHA-256 `cee9892f…c061a`), with no Reserved Font Name; `NOTICE` beside it. Coverage: Basic Latin, Latin-1, Latin Extended-A/B and Additional, Greek, Cyrillic (2,965 code points). CJK, Arabic, Hebrew, Indic and emoji are **not** covered. Each rune the fonts lack (or above U+FFFF) prints as `?`, and the profile block adds "Some characters cannot be shown in this PDF and appear as ?" once. The owner can extend coverage later by adding another OFL font family as a fallback (CJK fonts are many MB per weight; not measured) | **owner** 2026-11-23 (embed a Unicode font); research `noto-sans-font.md` (licence text, coverage, measurements) | — |
| 7 | Plan 6 modules | Task 17 after plan 6 merges; `sickness` and `medication` are a 422 `modules` until then | **accepted** default | none |
| 8 | Page | A4 portrait, content within 210 × 279.4 mm (`ExportPageHeightUsableMM = 279.4`, margins `ExportMarginMM = 12`) | **accepted** default; research `pdf-text-and-verification.md` §2 | a zone table |
| 9 | Catalog file | `api/internal/milestones/data/cdc.csv`. Provenance lines `# key: value` with the keys `catalog`, `name`, `authority`, `edition`, `retrieved`, `origin_url`, `licence`, `attribution`, `omitted`, then the header `id,age_months,domain,text,source_url,retired`. `domain` ∈ `social_emotional`, `language_communication`, `cognitive`, `movement_physical`. `age_months` ∈ {2,4,6,9,12,15,18,24,30,36,48,60}. `source_url` is the age page the text came from. `retired` is empty, or the edition that removed the item | the plan 5 pattern | — |
| 10 | Stable ids | `<age_months>m-<domain short>-<slug of the first words>`, e.g. `2m-se-calms-down`, `48m-lc-four-word-sentences` (years are written in months), never renumbered. An update that drops an item sets `retired` instead of deleting the row, so stored rows keep their text | plan 5 Open value 17 | orphaned rows lose text |
| 11 | Verbatim text | The HTML pages' wording, pronouns included ("by herself", "his spoon"). The review document lists where the 2021-named PDF differs | CDC: "You may not change the substantive content" | a neutral-pronoun edit needs an owner reading of CDC's terms |
| 12 | What is shown with the catalog | Under the list: CDC's sentence "This milestone checklist is not a substitute for a standardized, validated developmental screening tool." and the attribution "Source: CDC, Learn the Signs. Act Early. milestone checklists. CDC does not endorse this application or its use of the material. The material is available free of charge at https://www.cdc.gov/act-early/milestones/index.html." **Not** transcribed: CDC's "what most children (75% or more) can do" sentence, the doctor questions, the "act early" box, the screening call-outs, the tips and all images | CDC's conditions (attribution, no endorsement, free availability); spec non-goal "advice of any kind"; the images are third-party (research §1.5) | — |
| 13 | Age basis | `basis_months = AgeMonths(CorrectedBirthDate or BirthDate, today)`. The corrected birth date is used only while `CorrectedBirthDate` applies today, that is, before the second birthday | spec 7.1; plan 4's rule; CDC's ">3 weeks" equals the app's `PretermBelowDays` | — |
| 14 | The current list | A list age A is `current` when it is the largest A ≤ basis. CDC: "If your child's age falls between 2 checklist ages, use the checklist for the younger age". The basis is the corrected age until the second birthday and chronological after (Open value 13). `next` is the smallest A > basis, `earlier` is below current, `later` is above next. The page shows **every list**, in age order, each a `<details>` section with its title; all twelve are reachable on the same page. The current list is open and its title carries the text "Current list for this age" (text, never colour alone), and a "Go to the current list" link at the top jumps to it (an anchor, works without JavaScript). Before 2 months nothing is current: the 2-month list is open, with the text "Next list". Other lists start closed | owner 2026-11-23 (full list until 5 years); CDC wording; calendar months from `kidtime` | — |
| 15 | Reached | One row per catalog item (unique), with a date (`reached_on`: kid-local, not after today, not before the birth date) and an optional note (≤ `MaxNoteLen`). "Undo" is a hard delete. The date and the note can be edited. Only the date is shown, never the age at the time of reaching (an age beside a checklist age invites comparison) | spec 10.1, 6.1 | a computed field later |
| 16 | Custom milestones | A custom milestone is a reached row with text (1–200 characters after trimming, `MilestoneTextMaxLen = 200`) and a date. There are no unreached custom items | spec 10.1: "free-text custom milestones … marks one as reached with a date" | a second table |
| 17 | Caps | `MaxMilestonesPerKid = 1000` reached rows, catalog and custom together (the catalog has about 159) → 409 `limit` | the cap pattern of plans 4 and 5 | — |
| 18 | Timeline and home | Milestones are **not** in the timeline: they are dates, not instants, and spec 6.4 lists tracker entries (plans 4 and 5 did the same for growth and doses). They are **not** in the home's "Due" either: nothing about a milestone is ever due | spec 6.4, 7.3; plans 4 and 5 | — |
| 19 | PDF compression | **Off** (`SetCompression(false)`). With the embedded font, strings are 2-byte code units (`(\0Z\0o…)Tj`). Smoke runs `tr -d '\000'` and then `grep -a -F`, and Go tests decode UTF-16BE in a small extractor. Measured cost: 802 KB instead of 125 KB for 3000 rows | research `noto-sans-font.md` §4 | the smoke would need a zlib tool |
| 20 | Route | `POST /v1/kids/{kid}/export`, JSON `{from, to, modules}`, class Caregiver. The response is `200 application/pdf` with `Content-Disposition: attachment; filename="happybabytracker-<from>-<to>.pdf"`, `Cache-Control: no-store` and `X-Content-Type-Options: nosniff`. A `GET … ?from&to&modules` was rejected: it is cacheable by default and puts the module list in the URL, which spec 13.3 limits to ids and `from`/`to`/`types` | spec 13.3; the matrix needs no special case (its probe body is a 422) | — |
| 21 | Validation and the module rule | Order `from`, `to`, `modules`. `from` ≥ birth date. `to` ≥ `from`, ≤ kid-local today, with a span of at most 93 days. `modules` is a non-empty JSON array of distinct ids from `ExportModules` (`bottles, nursing, pumping, diapers, sleep, feeding, milestones, growth, vaccination`); anything else is 422 `modules`. **Rule:** the API exports every requested module whether or not it is switched on for the kid. The form ticks the switched-on modules by default and lists the switched-off ones, unticked, under "Switched off for this child". **Cost of this rule:** a caregiver can export data of a module a guardian switched off. The switch is a display choice that never deletes or hides data (G7), and caregivers can already open such pages (e.g. Vaccines with its module off), so no access widens. The alternative, refusing switched-off modules, would mean a guardian has to switch a module on to export it, and a caregiver (who cannot switch modules) could never export it | **owner** 2026-11-23 (filter, 422 on unknown or empty); spec 12, G7 | refusing switched-off ids is one check plus tests |
| 22 | Limits | `ExportMaxRowsPerSection = 3000`, with the overflow line "N more entries in this range are not shown. Export a shorter range to see them." `ExportMaxRows = 6000` across sections; sections after the budget print only their overflow line. `ExportMaxBytes = 4 << 20`: a defensive 500 if exceeded, never a truncated file. `MaxConcurrentExports = 1`: a second request answers 503 `busy` and is not queued | measured with the embedded font, uncompressed, standalone process: 8000 rows = 2.1 MB and a 22.5 MB peak (core fonts: 1.8 MB, 17.4 MB). The API's ceiling is 60 MiB with ~38 MiB of argon2 at peak | caps tuned in Task 15 from the smoke readings |
| 23 | Contents | See the list below this table | spec 12 (call on contents), spec 6.2 | sections added or removed |
| 24 | Units | The **requesting** parent's four preferences: weight, length, volume, and temperature from Task 17. They are read from the session's parent row and formatted from the stored column of that unit, never re-converted. Go formatters mirror `ui/src/lib/format.ts` and use its test examples | spec 6.2, 12 | — |
| 25 | UI flow | `/kids/{kid}/export` holds a plain `<form method="post" action="/kids/{kid}/export/pdf">` with `from` and `to` date inputs (default: the last `EXPORT_DEFAULT_DAYS = 30` kid-local days, ending today) and one checkbox `modules` per id of `EXPORT_MODULES` (Open value 21's defaults). The POST route validates, calls the API with `EXPORT_TIMEOUT_MS = 25000` and returns `new Response(apiResponse.body, …)`, streamed, not buffered. On a 422 or 503 it answers 303 to `/kids/{kid}/export?error=<field\|busy>&from=…&to=…&modules=a,b`, and the page shows the field message and re-fills the form from these enumerated values. `modules` in that URL is one enumerated parameter beyond spec 13.3's `from`/`to`/`types` (proposal; carrying nothing would lose what was ticked) | SvelteKit actions cannot return a binary body; no-JS baseline | — |
| 26 | Entry points | Milestones get a quick-log tile through `PAGE_ROUTE.milestones` (in MODULES order after feeding in the 0–11 and 12–23 bands). `KidLinks` gains "Milestones" and "Export PDF". The navigation bar is unchanged (HANDOFF: Home, Timeline, Growth, Account) | plans 4 and 5 | — |
| 27 | Band lookback | Milestones join `moduleTables` as `{"milestones_reached", "created_at", false}`: a milestone **recorded** (not reached) inside the 60-day lookback keeps the module on at 24 months. This uses the instant column the mechanism already compares | HANDOFF, plan-3 owner decision (60 days); spec 7.2 | a date-column variant |
| 28 | Catalog review | `docs/milestones/cdc.md`, generated by `UPDATE_DOCS=1 go test ./internal/milestones/`: a table per age with one row per item (id, domain, text, source page), then the retired items, then the PDF-wording differences. A test fails when it is stale. The owner reads that file's diff in the PR | plan 5 Open value 16; spec 17 "subject to the owner's review" | — |
| 29 | Update runbook | `docs/runbooks/updating-milestone-catalog.md`: where to look, how to change `edition` and `retrieved`, how to add rows, retiring rather than deleting, regenerating the review document, and the traps from the research note | plan 5 Open value 17 | — |
| 30 | PDF metadata | Title "happybabytracker export" (no name), Producer "happybabytracker"; no Author, Subject or Keywords | privacy of a forwarded file | — |

**Open value 23, the contents of the PDF.** The profile block comes first, then the chosen sections in `model.Modules` order.

- **Profile block** (always printed): name, birth date, sex, time zone, age today, and the corrected age when it applies ("4 months 12 days (corrected 2 months 26 days)"). Then the growth source name, the vaccination schedule caption or "No schedule chosen", the range, the modules included, and the character notice of Open value 6 when it applies.
- **Each tracker:** its entries (time, the tracker's fields as the timeline line words them, note) and its daily summaries (the `/summary` figures per day).
  - Bottles show per-source volumes and pumping shows per-side volumes (plan 5.1).
  - Nursing shows both weigh-ins and the milk estimate exactly as the timeline line words it, with no other interpretation.
- **Growth:** every measurement with its age (both ages when corrected), weight, length/height, head, BMI, and the z-score and percentile under the current source, or the unscored reason as the screen words it. Then the source's attribution and licence lines (and WHO's adaptation disclaimer), and one chart per measure with points.
- **Vaccination:** "Schedule: <authority>, <edition>", the schedule's attribution, each line with its status as text, then the doses given (date, antigens, product, lot, note) and "Other".
- **Milestones:** the CDC attribution and the reached rows (date, checklist age, domain, text, note), then the custom rows. Nothing about unreached items.
- **Footer** on every page: "Generated YYYY-MM-DD HH:MM (<zone>) by happybabytracker · Page n of N".

---

## Task 1: Milestone catalog format and loader

**Files:** Create `api/internal/milestones/{catalog.go,load.go}` (+ `_test.go` each); modify `api/internal/constants/constants.go`.

**Interfaces — Produces:**
```go
type Domain string // DomainSocialEmotional, DomainLanguage, DomainCognitive, DomainMovement
type Item struct{ ID string; AgeMonths int; Domain Domain; Text, SourceURL, Retired string }
type Catalog struct{ ID, Name, Authority, Edition, Retrieved, OriginURL, Licence, Attribution string; Omitted []string; Items []Item }
func Load() error                         // idempotent; a failure is remembered (the vaccines.Load pattern); main calls it before listening
func Get() (Catalog, bool)
func Lookup(id string) (Item, bool)       // retired items included
func Ages() []int                         // distinct age_months of active items, ascending
```
- [ ] **Step 1: Failing tests** on in-memory files, through an unexported `parse(name string, r io.Reader) (Catalog, error)`. Each error names the file, line and column, never the cell's text, and a test pins one message per refusal. Cases:
  - a valid file parses;
  - a missing provenance key;
  - a duplicate id;
  - an id that does not match `^[0-9]{1,2}m-(se|lc|cg|mp)-[a-z0-9-]{1,48}$` or does not start with its own `age_months`;
  - an unknown domain;
  - an `age_months` outside the twelve;
  - empty or over-long text (`MilestoneCatalogTextMaxLen = 200`);
  - text with a control character;
  - a `source_url` that does not start with `https://www.cdc.gov/`;
  - a `retired` value that is neither empty nor an edition-like string;
  - `Load()` is idempotent, and a failing injected loader is remembered.
- [ ] **Step 2:** `cd api && go test ./internal/milestones/ -race` → FAIL.
- [ ] **Step 3: Implement.** Constants: `MilestoneCatalogTextMaxLen = 200`, and `MilestoneCatalogAges = []int{2,4,6,9,12,15,18,24,30,36,48,60}` (a `var` beside `VaccineSchedules`).
- [ ] **Step 4:** PASS; then `gofmt -l .`, `go vet ./...` and staticcheck as in CLAUDE.md.
- [ ] **Step 5: Commit** `feat(api): milestone catalog format and loader`.

## Task 2: Transcribe the CDC catalog (owner reviews)

**Files:** Create `api/internal/milestones/data/cdc.csv`, `api/internal/milestones/review_test.go`, `api/internal/milestones/data_test.go`, `docs/milestones/cdc.md` (generated), `docs/runbooks/updating-milestone-catalog.md`; modify `api/internal/milestones/load.go` (embed), `README.md` (§ Third-party data: one row).

- [ ] **Step 1: Failing tests:**
  - `TestShippedCatalogLoads`: provenance keys are non-empty; `origin_url` is `https://www.cdc.gov/act-early/milestones/index.html`; `retrieved` is a date; the attribution contains "Source: CDC", "does not endorse" and "free of charge".
  - `TestPinnedItems`: the seven examples in `docs/superpowers/research/milestones/cdc-milestone-checklists.md` §2, by age, domain and exact text.
  - `TestEveryAgeHasEveryDomain`: 12 ages × 4 domains, each non-empty.
  - `TestNoJudgementWords`: no item text matches the Global Constraints denylist as a whole word, case-insensitive. **On a match, stop and ask the owner**; never reword.
  - `TestReviewDocIsCurrent`: renders the Markdown of Open value 28 and compares it with `docs/milestones/cdc.md`; `UPDATE_DOCS=1` rewrites it.
- [ ] **Step 2:** FAIL.
- [ ] **Step 3: Transcribe** from the twelve HTML age pages through the WebFetch tool (cdc.gov refuses curl with a 403), on the day of the task, in page order.
  - Set `retrieved` to that day, and `edition` to "Learn the Signs. Act Early. milestone checklists (2022 revision), web pages updated <the dates the pages show>".
  - Record in `omitted` what is left out (Open value 12).
  - Generate the review document; it lists any PDF wording difference found (pronouns).
  - Write the runbook.
- [ ] **Step 4:** PASS.
- [ ] **Step 5: Commit** `feat(api): CDC milestone catalog (LTSAE checklists)`. The PR description asks the owner to read `docs/milestones/cdc.md`.

## Task 3: Age basis and checklist positions

**Files:** Create `api/internal/milestones/window.go` (+ test); modify `api/internal/kidtime/kidtime.go` only if a helper is missing (+ test).

**Interfaces — Produces:**
```go
type Position string // PositionEarlier, PositionCurrent, PositionNext, PositionLater
type AgeBasis struct{ ChronologicalMonths int; CorrectedMonths *int; BasisMonths int }
func AgeBasisFor(birthDate string, gestWeeks, gestDays *int, today string, loc *time.Location) (AgeBasis, error) // Open value 13
func Positions(ages []int, basisMonths int) map[int]Position                                                    // Open value 14; every age gets a position
func CatalogShown(birthDate, today string) (bool, error)                                                          // Open value 3: today < AddAge(birthDate, MilestoneCatalogUntilYears, 'y'); chronological
```
Constant: `MilestoneCatalogUntilYears = 5`.
- [ ] **Step 1: Failing tests:**
  - `TestAgeBasis`:
    - a term kid;
    - a 32-week kid at 3 months chronological is 1 month corrected, and its basis is 1;
    - the same kid on the day before the second birthday and on it (correction ends on the birthday, as in plan 4);
    - a 36w6d kid is corrected and a 37w0d kid is not;
    - birth dates on the 29th–31st;
    - a Pacific/Auckland zone.
  - `TestPositions`:
    - basis 0 and 1: no current, next 2;
    - basis 2: current 2, next 4;
    - basis 3: current 2, next 4 (CDC's "younger age");
    - basis 14: current 12, next 15;
    - basis 59: current 48, next 60;
    - basis 60: current 60, no next;
    - only the ages passed in appear.
  - `TestCatalogShown`:
    - a kid at 4 years 11 months → true;
    - the day before the 5th birthday → true; the 5th birthday → false; the day after → false;
    - born 2020-02-29: 2025-02-28 → true, 2025-03-01 → false;
    - a preterm kid's 5th birthday is chronological (correction plays no part);
    - a malformed date → `kidtime.ErrBadDate`.
- [ ] **Step 2:** `go test ./internal/milestones/ ./internal/kidtime/ -race` → FAIL.
- [ ] **Step 3: Implement.**
- [ ] **Step 4:** PASS.
- [ ] **Step 5: Commit** `feat(api): milestone age basis, checklist positions and the 5-year span`.

## Task 4: Migration and store

**Files:** Create `api/internal/store/migrations/<next>_milestones.sql` (the next free number, see "Dependency on plan 6") and `api/internal/store/milestones.go` (+ test); modify `api/internal/model/model.go` (or `tracker.go`), `api/internal/store/store_test.go`, `api/internal/store/modules.go` (+ `modules_test.go`), `api/internal/constants/constants.go`.

**Interfaces — Produces:**
```go
type model.Milestone struct{ ID, KidID string; Item, Text *string /* exactly one */; ReachedOn string; Note *string; CreatedBy, UpdatedBy string; CreatedAt, UpdatedAt time.Time }
func (s *Store) CreateMilestone(ctx, kidID, actorID string, m model.Milestone, now time.Time) (model.Milestone, error) // ErrConflict at the cap; ErrDuplicate for a reached item
func (s *Store) Milestone(ctx, kidID, id string) (model.Milestone, error)
func (s *Store) ListMilestones(ctx, kidID string) ([]model.Milestone, error)          // reached_on DESC, id DESC
func (s *Store) UpdateMilestone(ctx, kidID, actorID string, m model.Milestone, now time.Time) error
func (s *Store) DeleteMilestone(ctx, kidID, actorID, id string) error
```
`ErrDuplicate` is new in `store.go`. Migration:
```sql
CREATE TABLE milestones_reached (
    id           TEXT PRIMARY KEY,
    kid_id       TEXT NOT NULL REFERENCES kids (id) ON DELETE CASCADE,
    catalog_item TEXT CHECK (catalog_item <> ''),
    custom_text  TEXT CHECK (custom_text <> ''),
    reached_on   TEXT NOT NULL CHECK (reached_on GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]' AND date(reached_on) = reached_on),
    note         TEXT CHECK (note <> ''),
    created_by   TEXT NOT NULL REFERENCES parents (id),
    updated_by   TEXT NOT NULL REFERENCES parents (id),
    created_at   INTEGER NOT NULL,
    updated_at   INTEGER NOT NULL,
    CHECK ((catalog_item IS NULL) <> (custom_text IS NULL))
) STRICT;
CREATE UNIQUE INDEX milestones_reached_kid_item ON milestones_reached (kid_id, catalog_item) WHERE catalog_item IS NOT NULL;
CREATE INDEX milestones_reached_kid_date ON milestones_reached (kid_id, reached_on, id);
```
`moduleTables` gains `"milestones": {"milestones_reached", "created_at", false}` (Open value 27).

- [ ] **Step 1: Failing tests:**
  - `wantTables` gains `milestones_reached`, and both `schema_version` assertions become the new number.
  - The upgrade test from the previous version: every table's rows survive.
  - A CRUD round trip.
  - `TestMilestoneBelongsToItsKid`: another kid's id → `ErrNotFound` on read, update and delete.
  - `TestMilestoneWriteByDisabledActor`.
  - `TestDuplicateCatalogItem`: `ErrDuplicate`; the same item for another kid is fine.
  - `TestMilestoneCap`: the 1001st → `ErrConflict`; a delete frees a slot.
  - The CHECKs refuse both item and text, neither, an empty string and a malformed date.
  - Cascade on kid delete.
  - `TestMilestonesKeepModuleOnAt24Months`: a row created 10 days before the 24-month band start → a stored override, on; one created 61 days before → none.
- [ ] **Step 2:** `go test ./internal/store/ -race` → FAIL.
- [ ] **Step 3: Implement.**
- [ ] **Step 4:** PASS.
- [ ] **Step 5: Commit** `feat(api): milestones migration and store`.

## Task 5: Milestones API

**Files:** Create `api/internal/httpapi/milestones.go` (+ test); modify `routes.go`, `matrix_test.go`, `cmd/happybabytracker-api/main.go` (`milestones.Load()` before listening), `api/internal/constants/constants.go`.

**Interfaces — Produces** (class in brackets):
- `GET /v1/kids/{kid}/milestones` [Caregiver]
- `POST /v1/kids/{kid}/milestones` [Caregiver] (201)
- `GET|PATCH|DELETE /v1/kids/{kid}/milestones/{entry}` [Caregiver]

`GET …/milestones` answers:
```json
{"catalog":{id,name,authority,edition,retrieved,origin_url,licence,attribution},"age":{"chronological_months","corrected_months"|null,"basis_months"},"catalog_shown":true,"checklists":[{"age_months","position","items":[{"id","domain","text","reached":{"id","reached_on","note"}|null}]}],"custom":[{"id","text","reached_on","note"}],"retired":[{"id","item","text","reached_on","note"}],"reached":[…only when catalog_shown is false, else []…]}
```
- When `catalog_shown` is true (before the 5th birthday), `checklists` holds **all twelve lists** in age order, each with its `position`, and items in file order. When it is false, `checklists` is `[]`, while `custom` and `retired` are unchanged and a new `reached` array lists the reached catalog rows (`{id, item, age_months, domain, text, reached_on, note}`). The API answers the same way whether the module is on or off.
- Entry view: `{id, item|null, text, reached_on, note}`; `text` is the catalog text for an item row.
- Input: `{item}` or `{text}`, plus `reached_on` and `note`. Validation order: `item`/`text`, `reached_on`, `note`.
- PATCH may change `reached_on`, `note` and, for custom rows only, `text`. `item` cannot change (→ 422 `item`).
- Errors:
  - an unknown or retired item → 422 `item`;
  - both or neither of `item` and `text` → 422 `item`;
  - an item reached twice → 409 `already_reached`;
  - the cap → 409 `limit`;
  - a date after the kid-local today or before the birth date → 422 `reached_on`;
  - 503 when the catalog failed to load (as `schedulesLoaded` does).

- [ ] **Step 1: Failing tests:**
  - `TestMilestonesShape`: `sameKeys` on every object, using a small test catalog.
  - `TestMilestonesPositionsThroughAPI`: a preterm kid shows both ages and the corrected-basis positions, with all twelve lists present.
  - `TestCatalogSpanThroughAPI`: the kid at 4 years 11 months gets 12 lists; on the 5th birthday `catalog_shown` is false, `checklists` is `[]`, and a reached 5-year item appears in `reached`; the kid's zone decides the day (Pacific/Auckland).
  - `TestMilestonesWithModuleOff`: a 30-month kid with the module off by default still gets 200 and the full catalog, and `GET …/modules` still reports milestones off (no default changed).
  - `TestRolesAndNotFound`: guardian and caregiver can do everything; the no-grant, unknown-kid and other-kid-entry 404 bodies are byte-identical; the admin gets 403.
  - `TestMilestoneValidation`: every refusal above, plus unknown fields, `null` on required fields, a text of 201 runes, a whitespace-only text, a note over `MaxNoteLen`.
  - `TestUndoReturnsItemToChecklist`.
  - `TestRetiredRowListed`.
  - `TestNoJudgementInViews`: every string field of every view, except the catalog's own `text` and `attribution`, avoids the denylist.
  - `TestNoValuesInLogs`: `wantCleanLog` with a distinctive custom text, note and date.
  - Matrix: the five routes in `routeClasses`; `matrixParams["entry"]` gains the `/milestones/` case, with a fixture row.
- [ ] **Step 2:** `go test ./... -race` → FAIL.
- [ ] **Step 3: Implement.**
- [ ] **Step 4:** PASS, including the matrix.
- [ ] **Step 5: Commit** `feat(api): milestones read model and reached entries`.

## Task 6: UI plumbing for milestones

**Files:** Modify `ui/src/lib/types.ts`, `ui/src/lib/constants.ts`, `ui/src/lib/server/constants.ts`, `ui/src/lib/server/api.ts` (+ `api.test.ts`), `ui/src/lib/paths.ts`, `ui/e2e/stub-api.ts`; create `ui/src/lib/server/milestone-form.ts` (+ test), `ui/src/lib/milestones.ts` (+ test), `ui/e2e/stub-milestones.spec.ts`.

**Interfaces — Produces:**
- Types: `MilestoneDomain`, `MilestonePosition`, `MilestoneItem`, `MilestoneChecklist`, `Milestones`, `MilestoneEntry`.
- Client: `api.milestones(kidId)`, `.getMilestone`, `.createMilestone`, `.updateMilestone`, `.deleteMilestone`. They follow the four-edit pattern: the `Endpoint` union, parsers with `ShapeError`, methods through `withId`/`withKidEntry`, and types. A malformed id answers not-found without a request.
- `readMilestoneForm(form, mode, zone)`: the date defaults to the kid-local today; an `item` hidden field or `text`; `note`. An edit sends only the changed fields, and missing hidden originals fail closed.
- Formatters:
  - `checklistTitle(ageMonths)`: "By 2 months", "By 1 year", "By 15 months", "By 2 years", "By 30 months", "By 3 years" (CDC's own age labels);
  - `domainLabel(domain)`: CDC's headings without "Milestones";
  - `reachedLabel(date)`: "Reached on 4 March 2026".
- Paths: `milestonesPath`, `milestoneNewPath`, `milestoneEntryPath`.
- Constants: `MILESTONE_TEXT_MAX = 200` (a client mirror).
- The stub implements the five routes with the API's validation order, statuses and 404 rules, over a **clearly marked stub catalog** that no test reads numbers from.

- [ ] **Step 1: Failing tests (RED):**
  - formatters for every CDC age;
  - `readMilestoneForm`: no date → today; a future date and hostile text refused with value-free messages; a text over 200 or blank → field `text`; an edit omits untouched fields; missing originals fail closed;
  - `api.test.ts`: list, create 422 with `field`, 409 `already_reached`, update, delete, a bad id → not-found without a fetch, malformed bodies → `error`;
  - the stub's self-tests.
- [ ] **Step 2:** `cd ui && pnpm test` → FAIL.
- [ ] **Step 3: Implement.**
- [ ] **Step 4:** `pnpm check && pnpm lint && pnpm test && pnpm test:e2e --trace off` PASS.
- [ ] **Step 5: Commit** `feat(ui): milestones client, forms, formatters and stub routes`.

## Task 7: UI: the Milestones page, its forms and entry points

**Files:** Create `ui/src/routes/(app)/kids/[kid]/milestones/{+page.server.ts,+page.svelte}`, `…/milestones/new/{+page.server.ts,+page.svelte}`, `…/milestones/[entry]/{+page.server.ts,+page.svelte}`, `ui/src/lib/components/milestones/Checklist.svelte`, `ui/e2e/milestones.spec.ts`; modify `ui/src/lib/quick-log.ts` (`PAGE_ROUTE.milestones`) (+ test), `ui/src/lib/components/KidLinks.svelte`, `ui/e2e/{home,kids}.spec.ts`.

**Interfaces:**
- The page header shows the age as the home does (`ageHeader`: both ages when corrected).
- At the top, a "Go to the current list" link (an anchor; Open value 14).
- Before the 5th birthday, the page shows **all twelve lists** in age order, each a `<details>` section titled `checklistTitle` and grouped by domain. The current list is open, and its title carries "Current list for this age". Before 2 months, the 2-month list is open and carries "Next list". Every item of every list shows its text, and either:
  - a "Reached" form: one tap submits and records today, and "Choose a date" links to `new?item=<id>`; or
  - when reached, "Reached on <date>" with an edit link.
- From the 5th birthday (`catalog_shown` false), there are no lists. A "Reached" section lists the reached catalog rows (date, list title, domain, text), followed by the custom rows.
- "Your milestones" lists the custom rows, with "Add a milestone".
- Rows whose item was retired are listed with their text.
- The CDC sentence and attribution (Open value 12) sit under the lists.
- There is no colour, icon, count, progress bar or "not yet" text. Nothing distinguishes an unreached item except the absence of a date.
- The page is reachable with the module off, and shows the same content. The module's default and the guardian's switch live only on the existing Modules page; this page adds no setting and no prompt to turn the module on.
- Create and edit follow the plan-5 pattern:
  - `failAction` and `fail(422, {form, values, field, message})`;
  - no-JS, with delete behind `<details>`;
  - 303 back to the page;
  - the same 404 page for no grant, an unknown kid and another kid's entry;
  - 409 `already_reached` → "Already marked as reached", and 409 `limit` → "This child has the maximum number of recorded milestones".

- [ ] **Step 1: Failing tests (Playwright, both projects, axe):**
  - a stub kid at 9 months shows all twelve list titles in age order: "By 9 months" is open and carries "Current list for this age", the others are closed but open on click (and without JavaScript, as native `<details>`), and "Go to the current list" jumps to it;
  - an item in a later list ("By 3 years") and one in an earlier list ("By 2 months") can each be marked reached;
  - a stub kid at 4 years 11 months: the 4-year list is current and the 5-year list is reachable;
  - a stub kid on its 5th birthday: no list titles, the "Reached" section lists a reached 5-year item, the custom rows are shown;
  - a preterm stub kid shows both ages, and the list current by corrected age;
  - a 30-month stub kid with the module off: the page loads from the kid links with the full catalog, the quick-log tile is absent, and the Modules page still shows milestones off;
  - one-tap reached records today and shows the date (`submit()` waits for the response);
  - choose a past date, read it back and edit it;
  - a future date shows the field message, with the values kept;
  - undo (delete) returns the item to its list;
  - add, edit and delete a custom milestone; 201 characters is refused;
  - a caregiver can do all of it;
  - a no-grant parent and another kid's entry id get the same 404 page;
  - the rendered page text avoids the denylist outside the catalog items;
  - no horizontal overflow at 390 px with a 200-character unbroken custom text;
  - no-JS reached and create;
  - the quick-log tile appears with the module on and hides with it off; `KidLinks` shows Milestones; `quick-log.test.ts` is updated.
- [ ] **Step 2:** RED.
- [ ] **Step 3: Implement.**
- [ ] **Step 4:** The full UI pipeline PASS.
- [ ] **Step 5: Commit** `feat(ui): milestones page, reached and custom milestones`.

## Task 8: The PDF dependency and the embedded Unicode font

**Files:** Modify `api/go.mod` and `api/go.sum` (`go get codeberg.org/go-pdf/fpdf@v0.12.0`, then `go mod tidy`; the diff must add exactly that one module), `api/internal/constants/constants.go`. Create:
- `api/internal/export/fonts/NotoSans-Regular.ttf` and `NotoSans-Bold.ttf`, downloaded from the URLs in `docs/superpowers/research/milestones/noto-sans-font.md` §1;
- `api/internal/export/fonts/LICENSE` (the OFL.txt, verbatim);
- `api/internal/export/fonts/NOTICE`: the family, version 2.015, file names, SHA-256s, source URLs, "These fonts are not covered by the MIT licence of this repository; they are licensed under the SIL Open Font License 1.1 (see LICENSE). Subsets of them are embedded in every exported PDF; the OFL does not apply to those documents.";
- `api/internal/export/fonts/{fonts.go,coverage.go}` (+ `fonts_test.go`, `coverage_test.go`).

**Interfaces — Produces:**
```go
package fonts
func Regular() []byte; func Bold() []byte          // go:embed; never written anywhere
func Covers(r rune) bool                            // in the BMP, not a control character, and in both fonts' cmap (format 4/12, parsed once at init)
func Encode(s string) (out string, substituted bool) // every rune with !Covers(r) → ExportFallbackRune ('?')
```
- [ ] **Step 0 (gate):** download both files and the OFL. Check their SHA-256 against Open value 6. If any hash differs, **stop and ask the owner**; do not take another version silently.
- [ ] **Step 1: Failing tests:**
  - `TestFontHashes`: the embedded bytes' SHA-256 equal the two pinned values.
  - `TestLicenceFiles`: `LICENSE` starts "Copyright 2022 The Noto Project Authors" and contains "SIL Open Font License, Version 1.1"; `NOTICE` names both files and their hashes.
  - `TestCoverage`: all of Basic Latin, Latin-1, Latin Extended-A and Greek "Ελένη", Cyrillic "Ольга", "Łucja Ștefan", € and curly quotes are covered; 李, U+1F600 (😀), U+0000–U+001F, U+007F, U+2028 and an unpaired surrogate (from invalid UTF-8) are not; Regular and Bold cover the same set.
  - `TestEncode`: "Mia 😀" → "Mia ?", substituted; "李小明" → "???"; "Zoë Müller-Señor" unchanged, not substituted; invalid UTF-8 bytes → `?`.
- [ ] **Step 2:** `cd api && go test ./internal/export/fonts/ -race` → FAIL.
- [ ] **Step 3: Implement.** The cmap reader is a small parser of formats 4 and 12 that refuses a malformed table at init (a failure is remembered, as `Load` does). Constants: `ExportFallbackRune = '?'`.
- [ ] **Step 4:** PASS; `CGO_ENABLED=0 go build ./...` passes; record the stripped binary size before and after in the PR (research: 13,365,408 → 14,889,120 bytes with fpdf and both fonts).
- [ ] **Step 5: Commit** `feat(api): embed Noto Sans (OFL 1.1) and add the PDF dependency`.

## Task 9: The PDF document writer

**Files:** Modify `api/internal/constants/constants.go`; create `api/internal/export/pdfdoc/doc.go` (+ `doc_test.go`, `textdump_test.go`).

**Interfaces — Produces:**
```go
type Doc struct{ /* wraps *fpdf.Fpdf */ }
func New(created time.Time, footer func(page, pages int) string) *Doc // A4, Open value 8 geometry, compression off, catalog sort, fixed producer and title, no author; AddUTF8FontFromBytes for both fonts
func (d *Doc) Heading(text string)
func (d *Doc) Line(label, value string)                               // a key/value row of the profile block
func (d *Doc) Paragraph(text string)                                  // wrapped, small print (attributions)
func (d *Doc) Table(cols []Col, rows [][]string)                      // header repeated after each page break; cells wrap; no cell leaves the page
type Col struct{ Title string; WidthMM float64 }
func (d *Doc) Chart(c Chart)                                          // axes, curves as polylines, points as small squares; grey and black only
func (d *Doc) Substituted() bool                                      // any rune replaced by fonts.Encode in this document
func (d *Doc) WriteTo(w io.Writer) (int64, error)                     // refuses when the document exceeds ExportMaxBytes
```
Constants: `ExportPageWidthMM = 210`, `ExportPageHeightMM = 297`, `ExportPageHeightUsableMM = 279.4`, `ExportMarginMM = 12`, `ExportFontSizeBody = 9`, `ExportFontSizeHeading = 12`, `ExportFontSizeSmall = 7`, `ExportMaxBytes = 4 << 20`, `ExportProducer = "happybabytracker"`, `ExportTitle = "happybabytracker export"`.

- [ ] **Step 1: Failing tests:**
  - `textdump_test.go`: the test-only extractor scans each uncompressed content stream's `(…)Tj` strings, unescapes `\\`, `\(`, `\)` and octal, decodes the 2-byte code units as UTF-16BE, and returns `[]string` per page. It is tested on a document written by `Doc`.
  - `TestDeterministic`: two documents with the same clock and text are byte-identical; a different clock changes only the date bytes.
  - `TestGolden`: a fixed small document with Latin, Greek and Cyrillic names against `testdata/doc.golden.pdf`; `UPDATE_GOLDEN=1` rewrites it.
  - `TestSubsetOnly`: a one-page document is below 64 KiB, proving the fonts are subset (each full font is ~430 KB).
  - `TestEmojiNeverReachesFpdf`: a document with "😀" in every method renders and does not panic.
  - `TestUnbrokenNoteWraps`: 500 × "x" in a 60 mm column; every string's x + width ≤ the page width − margin, and every y is within the 279.4 mm usable height.
  - `TestTableRepeatsHeader`: 200 rows → several pages, each starting with the header row.
  - `TestFooterOnEveryPage`: "Page n of N", with the right N.
  - `TestNoForbiddenObjects`: the raw bytes contain no `/JavaScript`, `/JS`, `/URI`, `/Launch`, `/EmbeddedFile`, `/XObject` or `/Encrypt`, and no `/BaseFont` other than the two embedded Noto Sans subsets (no core font).
  - `TestStructure`: the file starts `%PDF-1.` and ends `%%EOF\n`.
  - `TestSizeCap`: an oversized document → an error, and nothing is written.
- [ ] **Step 2:** `go test ./internal/export/... -race` → FAIL.
- [ ] **Step 3: Implement.**
  - Call no fpdf API outside `pdfdoc`.
  - Never call `HTMLBasicNew`, `Image*`, `Link*`, `AddFont` (core or file), `SetProtection` or `SetXmpMetadata`.
  - Every string goes through `fonts.Encode`, and no other text reaches fpdf.
- [ ] **Step 4:** PASS, plus the full API pipeline.
- [ ] **Step 5: Commit** `feat(api): PDF document writer with the embedded font`.

## Task 10: Display formatters for the PDF

**Files:** Create `api/internal/export/format.go` (+ test).

**Interfaces — Produces:** `FormatWeight(units.Weight, model.Units)`, `FormatLength(units.Length, model.Units)`, `FormatVolume(units.Volume, model.Units)`, `FormatMilk(units.Milk, model.Units)`, `FormatDuration(seconds int)`, `FormatAge(birthDate, today string)`, `FormatAgeWithCorrection(...)`, `FormatInstant(t time.Time, loc *time.Location)` ("2026-03-04 14:05"), `FormatDate(date string)`. Each takes the value from the stored column of the requester's unit (kg from grams, lb + oz from hundredths of an ounce, and so on) and never converts.

- [ ] **Step 1: Failing tests:**
  - the examples of `ui/src/lib/format.test.ts` and `growth.test.ts`, copied: the same inputs give the same strings;
  - `FormatMilk` for both unit sets: "weight gain 62 g, about 60 ml" and "weight gain 2.19 oz, about 2.04 fl oz";
  - DST: an instant at 01:30 EST and one at 01:30 EDT on the fall-back day print distinctly, with their offsets.
- [ ] **Step 2:** FAIL.
- [ ] **Step 3: Implement.**
- [ ] **Step 4:** PASS.
- [ ] **Step 5: Commit** `feat(api): export display formatters mirroring the UI`.

## Task 11: Export route, module filter, snapshot, profile and daily-tracker sections

**Files:** Create `api/internal/export/{export.go,profile.go,trackers.go}` (+ tests), `api/internal/store/export.go` (+ test), `api/internal/httpapi/export.go` (+ test); modify `routes.go`, `matrix_test.go`, `api/internal/constants/constants.go`.

**Interfaces — Produces:**
```go
// store: one read-only transaction, so every section sees one state; only the requested modules are read
type ExportSnapshot struct{ Kid model.Kid; Entries map[string][]TimelineItem /* per type, ≤ cap+1 */; Days []DaySummary; Growth []model.Growth; Doses []model.VaccineDose; Administrations []model.Administration; Dismissals map[string]string; Milestones []model.Milestone }
func (s *Store) ExportSnapshot(ctx context.Context, kidID string, modules []model.Module, loc *time.Location, from, to, now time.Time, rowCap int) (ExportSnapshot, error)
// export
type Request struct{ From, To string; Modules []model.Module; Units model.Units; Now time.Time; Loc *time.Location }
func Render(w io.Writer, snap store.ExportSnapshot, req Request) error // sections of req.Modules only, in model.Modules order
// httpapi
// POST /v1/kids/{kid}/export [Caregiver]
```
The handler does the following, in order:
1. Resolves the kid and the caller's units.
2. Validates the body (Open value 21: the module rule is "any id in `ExportModules`, distinct, non-empty, switched off or not").
3. Takes the export semaphore (`MaxConcurrentExports`; when busy → 503 `busy`).
4. Reads the snapshot of the requested modules.
5. Renders into the `pdfdoc` buffer.
6. Only then writes the headers of Open value 20 and the bytes.

Every error happens before the first byte, and nothing about the request is logged beyond the middleware's line. Constants: `ExportMaxDays = 93`, `ExportMaxRowsPerSection = 3000`, `ExportMaxRows = 6000`, `MaxConcurrentExports = 1`, and `ExportModules = []string{"bottles","nursing","pumping","diapers","sleep","feeding","milestones","growth","vaccination"}` (a `var`).

- [ ] **Step 1: Failing tests:**
  - Store:
    - `TestExportSnapshotIsOneTransaction`: a write between two section reads is not seen.
    - `TestExportSnapshotReadsOnlyRequested`: no query for an unrequested module (a statement counter, as in plan 5.1).
    - The row cap returns cap+1, so overflow is detectable.
  - httpapi:
    - `TestExportValidation`: Review Focus 5, one case per field and per bad `modules` value (unknown, `"Bottles"`, `"../x"`, a duplicate, `[]`, a string instead of an array, `null`, `sickness`, `medication`).
    - `TestExportModuleFilter`: Review Focus 6. `["growth","bottles"]` gives exactly two section headings, in the order bottles then growth; the vaccination caption, the milestone attribution and every diaper row are absent; one test per single module checks that only its section appears.
    - `TestExportSwitchedOffModule`: pumping switched off, then requested → its rows are present.
    - `TestExportHeaders`: the status, `Content-Type`, the attachment file name with only dates, `no-store`, `nosniff`.
    - `TestExportRoles`: guardian and caregiver 200; no grant and an unknown kid → an identical 404; the admin 403; a disabled caregiver 404.
    - `TestExportBusy`: with the semaphore held → 503 `busy`; released afterwards.
    - `TestExportEmptyKid`: Review Focus 1, read through the text extractor.
    - `TestExportTrackers`: one entry of each tracker with distinctive values. Each value appears as the timeline words it, in the requester's units, including advanced bottle sources and pumping sides, weigh-ins and the estimate; a running timer prints "running".
    - `TestExportTwoUnits`: Review Focus 7.
    - `TestExportDST`: Review Focus 8, in `America/New_York`.
    - `TestExportCaps`: 93 days × 20 bottles = 1860 rows, no overflow; 3001 rows → the overflow line once; the `ExportMaxRows` budget.
    - `TestExportNames`: Review Focus 4 and the notice line, present only when a substitution happened.
    - `TestExportNoJudgement`: the profile and tracker text avoids the denylist.
    - `TestNoValuesInLogs`: `wantCleanLog`; the kid name, a note, a volume, the range dates and the module names never appear.
  - Matrix: `"POST /v1/kids/{kid}/export": Caregiver`.
- [ ] **Step 2:** `go test ./... -race` → FAIL.
- [ ] **Step 3: Implement.**
- [ ] **Step 4:** PASS.
- [ ] **Step 5: Commit** `feat(api): PDF export route with the module filter, profile and daily trackers`.

## Task 12: Growth section, with percentiles, attribution and charts

**Files:** Create `api/internal/export/growth.go` (+ test); modify `export.go`.

**Interfaces:**
- A table of every measurement up to `to` (Open value 5): the date, the age (`FormatAgeWithCorrection`; "before expected term" rows are unscored, as on screen), weight, length/height, head and BMI. Then z and percentile per measure under the kid's **current** source, or the unscored reason in the screen's words. Scoring goes through the same function as `newGrowthView`, shared, not copied.
- Under the table: the source's `Attribution` and `Licence` lines and, for `who`, WHO's adaptation disclaimer from `growth.Info.Notice`.
- Then one chart per measure with at least one scored point: the source's percentile curves (`growth.Curves`, the same lines the UI draws) over the points' age range, and the kid's points.
  - Axes are labelled in the requester's unit.
  - Curves are grey hairlines, labelled at the right edge with the source's own percentile names.
  - Points are black squares. Nothing is filled or coloured.
- A measure the source does not cover prints "<source> does not cover this measure at this age", the screen's wording.

- [ ] **Step 1: Failing tests:**
  - `TestGrowthSectionScores`: the same z and percentile strings as `GET …/growth` for the same rows under `who` and `cdc`; switching the source changes the PDF and no stored row.
  - `TestGrowthAttribution`: the WHO attribution, licence and disclaimer are present with `who`, and the CDC lines with `cdc`.
  - `TestGrowth3000Rows`: 3000 measurements are all listed, with size and time within the caps, measured in the test.
  - `TestGrowthCorrectedAge`.
  - `TestGrowthNoJudgement`.
  - `TestChartGeometry`: every curve and point lies inside the chart box, and no point is drawn for an unscored row.
- [ ] **Step 2:** FAIL.
- [ ] **Step 3: Implement.**
- [ ] **Step 4:** PASS.
- [ ] **Step 5: Commit** `feat(api): export growth table, percentiles and charts`.

## Task 13: Vaccination and milestones sections

**Files:** Create `api/internal/export/{vaccines.go,milestones.go}` (+ tests); modify `export.go`.

**Interfaces:**
- Vaccination:
  - The evaluation is made on the kid-local **today**, through the same `evaluateKid` code path (moved to a shared function if needed, not copied).
  - It prints "Schedule: <authority>, <edition>", the schedule's `attribution`, each group's lines with their status words, the doses given up to `to` (date, antigens by name, product, lot, note), and "Other".
  - With no schedule: "No schedule chosen", then the doses.
- Milestones:
  - The catalog attribution and CDC's sentence (Open value 12).
  - The reached catalog rows up to `to` (date, checklist title, domain label, text, note) in date order, then the custom rows, then the retired rows.
  - No list of unreached items, and no count of them.

- [ ] **Step 1: Failing tests:**
  - `TestVaccinationSection`: a US kid shows the caption, the attribution, a given line with its date, and an overdue line labelled `overdue`; with no schedule, the doses are still listed.
  - `TestMilestonesSection`: the reached and custom rows appear; no unreached item text appears in the PDF; the attribution is present; no denylist word appears outside catalog text.
  - `TestRecordSectionsUpToTo`: a dose and a milestone dated after `to` are absent.
- [ ] **Step 2:** FAIL.
- [ ] **Step 3: Implement.**
- [ ] **Step 4:** PASS.
- [ ] **Step 5: Commit** `feat(api): export vaccination and milestones sections`.

## Task 14: UI: the export page with the module filter, and the streaming route

**Files:** Create `ui/src/routes/(app)/kids/[kid]/export/{+page.server.ts,+page.svelte}`, `ui/src/routes/(app)/kids/[kid]/export/pdf/+server.ts`, `ui/src/lib/server/export-form.ts` (+ test), `ui/e2e/export.spec.ts`. Modify:
- `ui/src/lib/server/api.ts` (+ test): the endpoint `export_pdf` returns `{ok:true, body: ReadableStream, contentType, disposition}` and never parses JSON on a 200;
- `ui/src/lib/server/constants.ts`: `EXPORT_TIMEOUT_MS = 25000`, `EXPORT_DEFAULT_DAYS = 30`;
- `ui/src/lib/constants.ts`: `EXPORT_MAX_DAYS = 93` (a mirror) and `EXPORT_MODULES`, the same list as the API's `ExportModules`;
- `ui/src/lib/paths.ts`, `ui/src/lib/components/KidLinks.svelte`;
- `ui/e2e/stub-api.ts`: the route returns a fixed, clearly marked stub PDF of a few hundred bytes **whose text lists the received modules**, so the UI tests can see which modules were sent, and mirrors the API's 422 fields, 404 and 503 `busy`.

**Interfaces:**
- Open value 25 and Open value 21's form rule: one `<fieldset>` "Modules to include" with a checkbox per `EXPORT_MODULES` id. The kid's switched-on modules (from `api.modules(kid)`) are ticked. The switched-off ones are listed, unticked, under the sub-heading "Switched off for this child".
- `readExportForm(form, zone)` returns `{from, to, modules}` or `{field}`. Modules come in `EXPORT_MODULES` order whatever order the form sent; none → `modules`; an id not in `EXPORT_MODULES` → `modules` (it is never sent to the API).
- The POST route reads the form, calls `api.exportPdf(kidId, input)`, and on 200 returns `new Response(body, {status: 200, headers: {content-type, content-disposition}})`. The hook adds `no-store`.
- 404 → the same 404 page as every kid route. 422 or 503 → a 303 to the page with `error`, `from`, `to` and `modules`, and the page re-ticks exactly those modules.
- The page explains in one sentence that the file is made now and not kept.

- [ ] **Step 1: Failing tests:**
  - `export-form.test.ts`: defaults; a 94-day range → field `to`; no module → `modules`; an unknown or hostile module id → `modules`; a duplicate is de-duplicated; the order is normalised; hostile dates.
  - `api.test.ts`: `exportPdf` streams the body without reading it; a 422 → `invalid` with field; a timeout → `unavailable`; the log line keeps its fixed shape, with no range or modules.
  - Playwright, both projects, axe:
    - the page lists every module, with the kid's switched-on modules ticked and a switched-off one (pumping turned off in the test) unticked under "Switched off for this child";
    - submitting starts a download (`page.waitForEvent('download')`, which waits for the response) named `happybabytracker-<from>-<to>.pdf`, and the stub's text shows exactly the ticked modules;
    - unticking all and submitting shows the `modules` field message, with the typed dates kept;
    - ticking the switched-off pumping sends it;
    - a direct POST with `modules=bogus` lands back on the page with the message;
    - a caregiver can export;
    - a no-grant parent gets the 404 page, on the page and on a direct POST; an admin is redirected;
    - it works without JavaScript;
    - no horizontal overflow at 390 px.
- [ ] **Step 2:** RED.
- [ ] **Step 3: Implement.**
- [ ] **Step 4:** The full UI pipeline PASS.
- [ ] **Step 5: Commit** `feat(ui): export page with the module filter, streaming the PDF`.

## Task 15: Smoke

**Files:** Modify `scripts/smoke.sh`.

- [ ] **Step 1: Extend `scripts/smoke.sh`**, failing first against the images from `main`. The PDF text is checked with `tr -d '\000' < "$pdf" | grep -a -F …` (Open value 19).
  - **Step 4k, milestones:**
    - As the guardian, open the smoke kid's Milestones page. Assert:
      - the CDC attribution text;
      - **all twelve list titles** ("By 2 months" … "By 5 years") in age order;
      - "Current list for this age" next to the title for the kid's age;
      - none of the denylist words outside item text.
    - Mark one item reached in a later list (e.g. "By 3 years").
    - Mark one item reached today through the real one-tap form, and one with an earlier date through the new form.
    - Add a custom milestone with a distinctive text, and give it a name-like word in Cyrillic, to exercise the font in 4l.
    - Edit its date and delete one through the real forms (scrape the hidden originals as 4h does).
    - The caregiver marks one.
    - The no-grant parent and the admin get the 404 page or a redirect on `/kids/{id}/milestones`, `/milestones/new` and `/milestones/{entry}`.
  - **Step 4l, export:** as the caregiver, POST `/kids/{id}/export/pdf` without JavaScript (the smoke's form helper).
    - **(a) All modules,** for the 4a–4j range. Assert:
      - 200, `content-type: application/pdf`, `content-disposition: attachment; filename="happybabytracker-…"` with no name in it, `cache-control: no-store`;
      - the body starts with `%PDF-1.` and ends with `%%EOF`; `/Type /Pages` and a `/Count` ≥ 1 are present; `/FontFile2` is present (the embedded font) and no `/BaseFont /Helvetica`;
      - these strings, after `tr -d '\000'`: the smoke kid's name, one 4a bottle volume as printed, the growth source's attribution opening words, "Schedule: Centers for Disease Control and Prevention", and the custom milestone's ASCII words;
      - the milestones and growth sections contain no denylist word.
    - **(b) Module filter:** only `bottles` and `growth`. The bottle volume and the growth attribution are present; "Schedule:", the CDC milestone attribution and the 4a diaper kind are absent.
    - **(c) A switched-off module:** switch pumping off as the guardian (as 4f does), export with pumping ticked, assert the 4a pumping volume is present, then switch it back on.
    - **(d) Refusals:** `modules` empty and `modules=bogus` each answer a 303 to `/kids/{id}/export?error=modules…`.
    - **(e) Concurrency:** a second export started while one is running is either a 200 or a 303 `error=busy`, never a 5xx.
    - **(f) Access:** the no-grant parent's POST gets the 404 page, and the admin's is redirected.
  - **Step 9:** the log scan gains the custom milestone text, the export's range dates and the module names, for the API and UI logs.
  - **Step 10:** keeps the ceilings (60 MiB API, 140 MiB UI). Run one all-module export inside the login-burst window and report the readings.
- [ ] **Step 2:** `docker build -t hbt-api:dev api && docker build -t hbt-ui:dev ui && scripts/smoke.sh hbt-api:dev hbt-ui:dev` → every line `PASS:`; `shellcheck scripts/smoke.sh`. If a ceiling fails, tune `ExportMaxRows` from the measured readings and record the numbers in the PR. Never raise the ceilings.
- [ ] **Step 3: Commit** `test: smoke covers milestones, the PDF export and its module filter`.

## Task 16: Docs, image notices and HANDOFF

**Files:** Modify `README.md`, `api/Dockerfile`, `docs/superpowers/HANDOFF.md`, `docs/milestones/README.md` (an index).

- [ ] **Step 1: Docs.**
  - `api/Dockerfile`:
    - copy fpdf's `LICENSE` to `/usr/share/doc/happybabytracker-api/fpdf/`;
    - copy `api/internal/export/fonts/{LICENSE,NOTICE}` to `/usr/share/doc/happybabytracker-api/noto-sans/`;
    - the `org.opencontainers.image.licenses` label becomes `MIT AND CC-BY-NC-SA-3.0-IGO AND OFL-1.1`.
  - README:
    - "Milestones": what is shown, corrected age, never a judgement, the CDC conditions, the update runbook.
    - "PDF export": who can export, choosing modules (switched-off modules can be ticked), what is in it, 93 days, made on demand and never stored, Latin, Greek and Cyrillic print and other scripts and emoji appear as `?`, A4 that prints on Letter.
    - Third-party data and code: the CDC catalog, `codeberg.org/go-pdf/fpdf` (MIT), and "Noto Sans 2.015, Copyright 2022 The Noto Project Authors, SIL Open Font License 1.1; subsets are embedded in exported PDFs; licence in `api/internal/export/fonts/LICENSE` and in the image under `/usr/share/doc/happybabytracker-api/noto-sans/`".
  - HANDOFF: plan 7 done, the owner's answers, the defaults accepted, the unverified items below, and which migration number was taken.
- [ ] **Step 2:** The whole pipeline in CLAUDE.md (API, UI, images, smoke) green.
- [ ] **Step 3: Commit** `docs: plan 7 done (milestones and PDF export)`.

## Task 17 (after plan 6 merges): sickness and medication sections

**Files:** Create `api/internal/export/{sickness.go,medication.go}` (+ tests); modify `export.go`, `api/internal/constants/constants.go` (`ExportModules` gains both), `ui/src/lib/constants.ts` (`EXPORT_MODULES`), `ui/e2e/stub-api.ts`, `scripts/smoke.sh`.

**Interfaces:**
- Sickness: each symptom entry in the range: time, symptom, severity, the temperature in the requester's temperature unit with the method, and the note.
- Medication: each course active in the range (name, dose text, the schedule as the screen words it, start, end) and the doses given in the range.
- Nothing about reminders, and no wording about doses being due or missed. The denylist applies.
- Both modules join the form's checkboxes and the module-filter tests.

- [ ] **Step 1: Failing tests** as in Tasks 11 and 14: units for temperature, DST, roles, the module filter including the two new ids, no values in logs, no judgement.
- [ ] **Step 2:** FAIL.
- [ ] **Step 3: Implement.**
- [ ] **Step 4:** The full pipeline and smoke green.
- [ ] **Step 5: Commit** `feat(api): export sickness and medication`.

---

## Done when

- API and UI pipelines and `scripts/smoke.sh` pass on the branch head, and CI is green.
- The access matrix covers all six new routes and fails if one is missing.
- A guardian and a caregiver see all twelve CDC lists until the kid's 5th birthday, with the list for the kid's corrected age marked, mark any item reached and unmark it, and add custom ones, and no screen or PDF text judges. The owner has reviewed `docs/milestones/cdc.md` in the PR.
- A caregiver downloads a PDF of one kid for a range of at most 93 days and the chosen modules. It contains exactly those modules, in their own units and the kid's time zone, with every source's attribution where its data is shown. Latin, Greek and Cyrillic names print correctly. Smoke reads its text without a PDF viewer, including the module filter.
- The final report states:
  - which image content changed: `api/` sources, embedded catalog data, **a new Go module** `codeberg.org/go-pdf/fpdf`, **two embedded font files** (~0.86 MB), Dockerfile `COPY` lines and the licence label; `ui/` sources; no npm change;
  - that the edge still needs its `bump-happybabytracker` PR merged and a deploy, and that nothing else on the edge changes;
  - which migration number this plan took.

## Not verified (state in the PR)

- **CDC reuse:** "Use of Agency Materials" and a training page ("Reproducible (no copyright)") support public-domain reuse. The checklist pages themselves carry no licence statement, and the 2022 revision was written with AAP, whose article was not read (403). This is not legal advice, and CDC was not contacted.
- **Pronouns:** whether a pronoun is "substantive content" is avoided by transcribing verbatim.
- **2026 page updates:** that they changed nothing but pronouns was checked for the 2-month and 4-year pages only.
- **Item count:** the 159-item count comes from the PDF's bullet glyphs and is confirmed only by the transcription.
- **Corrected age:** CDC states no age at which it stops for milestones. The second birthday is the app's rule (spec 7.1, plan 4).
- **WHO ShareAlike:** the WHO licence reading of plan 4 now also covers derived percentiles printed in a PDF. Whether ShareAlike reaches such a PDF was not checked.
- **OFL reading:** the OFL 1.1 text and FAQ (1.12, 1.13) say a document made with the font is not bound by the OFL and that embedding a subset is allowed. Whether an fpdf subset is a "Modified Version" of the font is not settled by the FAQ, which classifies subsetting as modification only for webfonts. The document exception (condition 5) applies either way. This is not legal advice.
- **Font source:** the font files come from a `main` branch (`notofonts/notofonts.github.io`), not a tagged release. Their bytes are pinned by SHA-256, and Task 8 stops if they differ.
- **Shaping:** fpdf ignores GSUB and GPOS, so combining-mark sequences (a letter plus a separate accent) may be badly positioned. Precomposed letters print correctly. CJK, Arabic, Hebrew, Indic and emoji print as `?`, and coverage can be extended later.
- **fpdf panic:** a rune above U+FFFF panics in fpdf v0.12.0. It was found by experiment; no upstream issue was checked. The encoder prevents it.
- **fpdf roadmap:** "use UTF-8 exclusively" now matches this plan's use (TrueType fonts only), so it is no longer a risk to the core-font path.
- **Memory:** the figures (22.5 MB peak at 8000 rows with the fonts, standalone) were not measured in the 96 MB container with the API's own heap. Task 15 measures the real one.
- **gopdf** was not built or measured.
- **iOS:** the download behaviour of an installed PWA (share sheet or viewer) was not tested on a device.
- **Midnight-DST flaw:** the `kidtime` flaw noted in HANDOFF also reaches `AgeBasisFor` through `AgeMonths` and `CorrectedBirthDate` in zones whose DST starts at midnight. It is not fixed here (minimum change). The follow-up already listed in HANDOFF covers it.
