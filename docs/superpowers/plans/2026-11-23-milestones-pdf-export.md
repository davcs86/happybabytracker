# Milestones and PDF Export Implementation Plan (plan 7)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status: draft for the owner.** Nothing here is built. Execution starts only after the owner answers the questions below and approves the open values. Tasks 8 and 10–15 depend on question 1; Task 2 depends on question 2.

## Questions for the owner

Each question has a proposed default and says what changes if your answer differs.

1. **PDF library (a new dependency):** the default is `codeberg.org/go-pdf/fpdf` v0.12.0, the maintained successor of the archived `github.com/go-pdf/fpdf`. It would be the API's first new runtime module since `modernc.org/sqlite`: MIT, pure Go, it adds no other module, adds 0.6 MB to the binary, and is pinned in `go.sum`. If you say no, Task 8 becomes a standard-library-only PDF writer with its own Helvetica width tables, which is about two more tasks and our own PDF bugs.
2. **Milestone catalog:** the default is all twelve CDC "Learn the Signs. Act Early." checklists (2 months to 5 years, the 2022 revision as cdc.gov serves it today, about 159 items), transcribed verbatim from the HTML age pages under CDC's four reuse conditions. If you limit it to 2 months to 2 years, the 30-month to 5-year rows (about 59 items) are left out and question 3 mostly goes away.
3. **The 24-months-and-over band:** the default is that the module stays off by default from 24 months (spec 7.2, yours). If a guardian turns it on, the page shows the checklists around the kid's age up to the 5-year list, and from 6 years only the reached and custom milestones. If you would rather hide the catalog from 24 months, the page shows only reached and custom milestones there.
4. **Export range cap:** the default is at most 93 days per export, the same as the daily summaries, and the export is built in memory with row caps. A larger cap needs a page-by-page streaming design and a new memory test, about one more task.
5. **Record sections:** the default is that growth, vaccination and milestones print the kid's whole record up to the range's end date, because they are sparse histories a clinician reads whole, while the daily trackers print only the range. If you would rather they print only what falls inside the range, three filters change and nothing else.
6. **Names in other scripts:** the default uses the PDF's built-in fonts with WinAnsi (Latin-1 plus €, Œ, curly quotes, dashes), so "Zoë", "Müller" and "Señor" print correctly. Every other character (Ł, ș, Cyrillic, CJK, emoji) prints as `?`, and the profile block says once "Some characters cannot be shown in this PDF". If they must render, the image embeds a Unicode TrueType font (for example Noto Sans, SIL OFL): a font licence, a few hundred KB more and one more task.
7. **Sickness and medication in the export:** the default is Task 16, which adds them in a short follow-up after plan 6 merges. Until then the API refuses those module ids. If you say no, Task 16 is dropped and plan 6's data is never exported.
8. **Page size:** the default is A4 pages whose margins, content and footer all stay within the top 279.4 mm, so the same file prints at 100 % on US Letter, with no setting and no locale guess. If you want Letter for some kids, the API needs a time-zone-to-country table to pick the size.

---

**Goal:** Two parts. Milestones: a guardian or caregiver sees the CDC developmental milestones around the kid's (corrected) age, marks a milestone as reached with a date, adds custom free-text milestones, and can change or delete any of them. The app never says a kid is late, behind or anything else that compares or judges. PDF export: a guardian or caregiver exports one kid for a date range and a chosen set of modules as a PDF that the API renders itself, in the requester's units and the kid's time zone, generated on demand and never stored.

**Architecture:** A new package `api/internal/milestones` embeds one CSV (provenance header, stable item ids, as `api/internal/vaccines` does), loads it at start and computes each checklist's position against the kid's age basis. The next free migration adds `milestones_reached`. Routes follow the vaccination pattern: one kid read model and CRUD on reached rows. A new package `api/internal/pdfexport` reads one consistent snapshot of the kid's data (one read transaction), lays it out with `codeberg.org/go-pdf/fpdf` using built-in Helvetica only (no images, no HTML, no external fetch), and writes the finished document to the response. A new UI page `/kids/{kid}/export` holds a plain form that posts to a `+server.ts` route, which streams the API's response body to the browser as an attachment. The UI adds a Milestones page, a quick-log tile and kid links.

**Tech Stack:** Go 1.27 (`embed`, `encoding/csv`), SQLite via `modernc.org/sqlite`, **new: `codeberg.org/go-pdf/fpdf` v0.12.0** (question 1); SvelteKit 3 / Svelte 5 runes, Tailwind 4. No new npm dependency.

**Spec:** `docs/superpowers/specs/2026-11-20-happybabytracker-design.md` sections 2 (G6, G7, G8, G12, non-goals), 5.2, 6.1, 6.3 (`milestones_reached`), 6.4, 7.1, 7.2, 7.3, 10.1, 12, 13.3, 14, 16, 17 ("The PDF library and layout"; "The milestone catalog's exact items"). Earlier plans: `docs/superpowers/plans/2026-11-22-growth.md` (corrected age, source attribution), `2026-11-22-vaccination.md` (embedded data with provenance, generated review documents, update runbook), `2026-11-23-advanced-logging.md` (per-source volumes, weigh-ins and the milk estimate). Research notes, all retrieved 2026-11-23: `docs/superpowers/research/milestones/{cdc-milestone-checklists,pdf-libraries,pdf-text-and-verification}.md`.

**Dependency on plan 6:** plan 6 (sickness, medication, reminders) is being refreshed in parallel and is not merged. Its draft takes migration 0006. This plan's migration is **"the next free number"**: 0006 if this plan merges first, 0007 if plan 6 merges first. The `schema_version` assertions change with it. Task 16 waits for plan 6.

## Global Constraints

- Every limit, cap and duration lives in `api/internal/constants/constants.go` (API) or `ui/src/lib/server/constants.ts`, with client-safe mirrors in `ui/src/lib/constants.ts` (CLAUDE.md rule 7). No literal elsewhere, including the PDF's page and margin sizes, font sizes and row caps.
- **No judgement, anywhere the app writes its own words** (spec 10.1, spec 2 non-goals). On the Milestones page, in the milestones API view, in the PDF's profile, milestones and growth sections, and in every empty-state or error text, none of these words appear: `late`, `behind`, `delayed`, `delay`, `normal`, `abnormal`, `typical`, `atypical`, `on track`, `ahead`, `missed`, `missing`, `concern`, `worry`, `should`, `red flag`, `warning`. Nothing is coloured by state, and no list of "not reached" items appears in the PDF. The only status words in the PDF are the vaccination statuses the owner approved in plan 5 (`given`, `due`, `upcoming`, `overdue`, `dismissed`), and only in the vaccination section. CDC's own item text is reproduced verbatim (CDC's conditions forbid changing it). Task 2 checks it against the same list and stops for the owner if an item matches.
- **Milestones use corrected age** where spec 7.1 applies (born before 37 weeks 0 days, until the second birthday; this matches CDC's "born more than 3 weeks prematurely, use … corrected age"), and both ages are shown wherever the corrected one is. Vaccination code stays chronological.
- Calendar arithmetic only through `kidtime` (`AgeMonths`, `CorrectedBirthDate`, `AddAge`). Dates are `YYYY-MM-DD` in the kid's zone, and instants are printed in the kid's zone.
- A caller with no grant, an unknown kid and a reached row of another kid answer the same 404 body. The admin is refused. Writes re-assert the role with `actorRole` inside `inTx`. Hard deletes, no edit history. Milestone writes and the export are `Caregiver` (spec 5.2: entries and export belong to both roles).
- **Privacy of the export:** generated on demand, never written to disk, never cached (`Cache-Control: no-store` on the API and UI responses), never logged. Logs carry the route, kid id, status and duration only, never a module list, range, row count or byte count. The UI's `logFailure` keeps its fixed shape. The range and module ids travel in the POST body, never in a URL. A validation redirect carries only `error`, `from`, `to` and `modules` (Open value 25). The download's file name and the PDF's metadata hold no name.
- **The PDF writer is closed:** built-in Helvetica only, text, lines and rectangles. No image, HTML (`HTMLBasicNew`), link, attachment, JavaScript, encryption or font file. The writer never reads a file or the network. Every string passes through one app-owned encoder (cp1252, fallback `?`).
- **Determinism:** the PDF's creation and modification dates are the request's `Deps.Now`, the producer is a fixed string, the catalog is sorted, sections come in a fixed order, and nothing ranges over a map into the output. With a fixed clock, a golden-file test is byte-stable.
- Embedded data lives under `api/internal/milestones/data/` (never `testdata`, never a `*.db*` name). Golden PDFs for tests live in `api/internal/pdfexport/testdata/`.
- Comments state constraints only, at most 2 lines (rule 9). `gofmt -l` prints nothing, staticcheck is clean, Prettier uses tabs. Every new page works without JavaScript, passes axe at 390×844 and 1280×800 and keeps `Cache-Control: no-store`. The Milestones page stays reachable with its module switched off; only its tile hides.
- Tests that pin counts change together with the code they pin: `wantTables` and `schema_version`, `routeClasses`, `matrixParams` (`entry` for `/milestones/`), the stub's routes, `quick-log.test.ts` (milestones gets a tile), and the log scans.
- New dependency hygiene (rule 10): the version is pinned in `go.mod`/`go.sum`, its `LICENSE` is copied into the image beside the WHO and CDC notices, and the README's third-party list gains it.

## Review Focus

Failure modes that no task's happy path exercises. Each has a test in the named task.

1. **A kid with no data:** every chosen module prints its heading and "No entries in this range." (record sections: "No measurements recorded.", "No doses recorded.", "No milestones recorded."); the PDF is still valid and has one page (Tasks 10–12).
2. **3000 growth rows** (the `MaxGrowthPerKid` cap) render within the row cap, and **93 days of bottles** at 20 a day (1860 rows) render with the overflow line exactly once. The worst case for every chosen module stays under `ExportMaxRows`, `ExportMaxBytes` and the measured memory budget (Tasks 10, 11).
3. **A 500-character unbroken note** wraps inside its column and never overflows the page width or the 279.4 mm usable height (Tasks 8, 10).
4. **A name with accents or emoji:** "Zoë Müller-Señor" prints exactly, "Mia 😀" prints "Mia ?", "李小明" prints "???" and adds the one-line notice, and no byte outside cp1252 reaches a content stream (Tasks 8, 10).
5. **A corrupt or unknown module id**, a duplicate id, an empty list, `sickness` or `medication` before Task 16, a non-array `modules`, an unknown body field, a malformed date, `to` before `from`, a 94-day range, `to` after the kid-local today and `from` before the birth date: each is a 422 naming the field, never a 500 and never a partial PDF (Task 10).
6. **Two parents with different units** export the same kid: the kg/ml/cm parent and the lb-oz/fl oz/in parent get byte-different files whose values are each exactly the stored columns of their unit (Tasks 9, 10, 11).
7. **DST days in the kid's zone:** a bottle at 01:30 on a fall-back day in `America/New_York` prints the right offset and the daily summary counts it once. A spring-forward day sums 23 hours of sleep. Times are printed in the kid's zone, not the server's (Task 10).
8. **Caregiver access:** a caregiver exports and reaches every milestone write. A no-grant parent and an unknown kid get byte-identical 404 bodies on all six new routes. The admin is refused (403 at the API, a redirect at the UI) (Tasks 5, 10, 13, 14).
9. **Concurrency and memory:** a second export while one runs answers 503 `busy` and never queues. An export during the smoke's login burst keeps the API under 60 MiB (Tasks 10, 14).
10. **Milestones:** a catalog item reached twice is a 409 and never a second row; a date after the kid-local today or before the birth date is a 422; a custom text of 201 characters or only spaces is a 422; deleting a reached item puts it back in its checklist; a stored row whose catalog item was retired is still listed with its text; a preterm kid at 3 months chronological and 1 month corrected sees the 2-month list as "next", and both ages are shown (Tasks 1, 3, 4, 5).
11. **Band rule:** a kid crossing 24 months with a milestone recorded in the 60-day lookback keeps the module on as a stored override (Task 5).

---

## Open values (owner confirmation before execution)

Rule 1 (never assume an owner value). "Spec"/"HANDOFF" rows cite a decision already made. Every other row is this plan's proposal, with its basis and the cost if it is wrong. Rows 1–8 are the questions above.

| # | Value | Decision or proposal | Basis | Cost if wrong |
|---|---|---|---|---|
| 1 | PDF library | `codeberg.org/go-pdf/fpdf` v0.12.0 (2026-05-18) | research `pdf-libraries.md`: maintained, MIT, standard library only in the root package, +635 KB, deterministic; alternatives rejected there | Task 8 rewritten for the stdlib writer |
| 2 | Catalog edition and scope | CDC LTSAE checklists, all 12 ages, HTML pages retrieved on the transcription day, verbatim | research `cdc-milestone-checklists.md`; spec 10.1 "CDC's public-domain milestone lists" (call) | ~59 rows dropped |
| 3 | 24+ band | Default off (spec 7.2, **owner**). When on: checklists up to 60 months; from 72 months only reached and custom rows | spec 7.2; CDC lists stop at 5 years | one condition in `Positions` |
| 4 | Range cap | `ExportMaxDays = 93`, inclusive kid-local dates | = `SummaryMaxDays`; the daily summaries the PDF prints cannot exceed it | streaming redesign |
| 5 | Record sections | growth, vaccination, milestones: everything up to `to`; trackers: `[from, to]` | clinical usefulness | three filters |
| 6 | Character set | cp1252 via an app encoder; fallback `?` (`ExportFallbackRune`); one notice line when any substitution happened anywhere in the document | research `pdf-text-and-verification.md` §1 | a TTF task and a font licence |
| 7 | Plan 6 modules | Task 16 after plan 6 merges; `sickness` and `medication` are a 422 `modules` until then | spec 12 "each chosen tracker" | none |
| 8 | Page | A4 portrait, content within 210 × 279.4 mm (`ExportPageHeightUsableMM = 279.4`, margins `ExportMarginMM = 12`) | research §2 | a zone table |
| 9 | Catalog file | `api/internal/milestones/data/cdc.csv`: `# key: value` lines `catalog`, `name`, `authority`, `edition`, `retrieved`, `origin_url`, `licence`, `attribution`, `omitted`, then the header `id,age_months,domain,text,source_url,retired`. `domain` ∈ `social_emotional`, `language_communication`, `cognitive`, `movement_physical`; `age_months` ∈ {2,4,6,9,12,15,18,24,30,36,48,60}; `source_url` is the age page the text came from; `retired` is empty or the edition that removed it | the plan 5 pattern | — |
| 10 | Stable ids | `<age_months>m-<domain short>-<slug of the first words>`, e.g. `2m-se-calms-down`, `48m-lc-four-word-sentences` (years are written in months); never renumbered. An update that drops an item sets `retired` instead of deleting the row, so stored rows keep their text | plan 5 Open value 17 | orphaned rows lose text |
| 11 | Verbatim text | The HTML pages' wording, pronouns included ("by herself", "his spoon"); the review document lists where the 2021-named PDF differs | CDC: "You may not change the substantive content" | a neutral-pronoun edit needs an owner reading of CDC's terms |
| 12 | What is shown with the catalog | Under the list: CDC's sentence "This milestone checklist is not a substitute for a standardized, validated developmental screening tool." and the attribution "Source: CDC, Learn the Signs. Act Early. milestone checklists. CDC does not endorse this application or its use of the material. The material is available free of charge at https://www.cdc.gov/act-early/milestones/index.html." CDC's "what most children (75% or more) can do" sentence, the doctor questions, the "act early" box, the screening call-outs, the tips and all images are **not** transcribed | CDC conditions (attribution, no endorsement, free availability); spec non-goal "advice of any kind"; images are third-party (research §1.5) | — |
| 13 | Age basis | `basis_months = AgeMonths(CorrectedBirthDate or BirthDate, today)`, using the corrected birth date only while `CorrectedBirthDate` applies today (before the second birthday) | spec 7.1; plan 4's rule; CDC's ">3 weeks" equals the app's `PretermBelowDays` | — |
| 14 | "Around the age" | A checklist age A is `current` when it is the largest A ≤ basis (CDC: "If your child's age falls between 2 checklist ages, use the checklist for the younger age"), `next` the smallest A > basis, `earlier` below current, `later` above next. The page shows current and next open, earlier inside a closed `<details>` "Earlier ages", and later not at all. Before 2 months nothing is current and 2 months is next | CDC wording; calendar months from `kidtime` | — |
| 15 | Reached | One row per catalog item (unique), with a date (`reached_on`, kid-local, not after today, not before the birth date) and an optional note (≤ `MaxNoteLen`). "Undo" is a hard delete. The date and note are editable. No age at the time of reaching is shown, only the date (an age beside a checklist age invites comparison) | spec 10.1, 6.1 | a computed field later |
| 16 | Custom milestones | A custom milestone is a reached row with text (1–200 characters after trimming, `MilestoneTextMaxLen = 200`) and a date. There are no unreached custom items | spec 10.1 "free-text custom milestones … marks one as reached with a date" | a second table |
| 17 | Caps | `MaxMilestonesPerKid = 1000` reached rows (catalog and custom together; the catalog has about 159) → 409 `limit` | plan 4 and 5 caps pattern | — |
| 18 | Timeline and home | Milestones are **not** in the timeline (they are dates, not instants; spec 6.4 lists tracker entries, as plans 4 and 5 did for growth and doses) and **not** in the home's "Due" (nothing about a milestone is ever due) | spec 6.4, 7.3; plan 4/5 precedent | — |
| 19 | PDF compression | **Off** (`SetCompression(false)`), so smoke and tests read the text with `grep -a` and a small Go extractor; files are ~6× larger (684 KB instead of 109 KB for 3000 rows) | research §4 | smoke needs a zlib tool |
| 20 | Route | `POST /v1/kids/{kid}/export` with JSON `{from, to, modules}`, class Caregiver; `200 application/pdf`, `Content-Disposition: attachment; filename="happybabytracker-<from>-<to>.pdf"`, `Cache-Control: no-store`, `X-Content-Type-Options: nosniff`. Not a `GET … ?from&to&modules`: a GET is cacheable by default and puts the module list into the URL, which spec 13.3 limits to ids and `from`/`to`/`types` | spec 13.3; the matrix needs no special case (its probe body is a 422) | — |
| 21 | Validation | Order `from`, `to`, `modules`. `from` ≥ birth date; `to` ≥ `from`, ≤ kid-local today, span ≤ 93 days; `modules` a non-empty array of distinct ids from `bottles, nursing, pumping, diapers, sleep, feeding, milestones, growth, vaccination`. A module switched off for the kid **is** exportable (turning off never hides data from its owner, G7), and the UI ticks only the enabled ones by default | spec 12, G7 | — |
| 22 | Limits | `ExportMaxRowsPerSection = 3000` (overflow line: "N more entries in this range are not shown. Export a shorter range to see them."), `ExportMaxRows = 6000` across sections (sections after the budget print only their overflow line), `ExportMaxBytes = 4 << 20` (a defensive 500 if exceeded, never a truncated file), `MaxConcurrentExports = 1` (a second request → 503 `busy`, not queued) | measured: 8000 rows uncompressed = 1.8 MB and 17.4 MB peak in a standalone process; the API's ceiling is 60 MiB with ~38 MiB of argon2 at peak | caps tuned in Task 14 from smoke readings |
| 23 | Contents | Profile block (always): name, birth date, sex, time zone, age today and corrected age when it applies ("4 months 12 days (corrected 2 months 26 days)"), growth source name, vaccination schedule caption or "No schedule chosen", the range and the modules included. Then the chosen sections in `model.Modules` order. Per tracker: entries (time, the tracker's fields as on the timeline line, note) and daily summaries (the `/summary` figures per day). Bottles show per-source volumes and pumping shows per-side volumes (plan 5.1); nursing shows both weigh-ins and the estimate exactly as the timeline line words it, with no other interpretation. Growth: every measurement with age (both ages when corrected), weight, length/height, head, BMI, z-score and percentile under the current source or the unscored reason as on screen; the source's attribution and licence lines (and WHO's adaptation disclaimer); one chart per measure with points. Vaccination: "Schedule: <authority>, <edition>", the schedule's attribution, each line with its status as text, then the doses given (date, antigens, product, lot, note) and "Other". Milestones: the CDC attribution and the reached rows (date, checklist age, domain, text, note) and custom rows; nothing about unreached items. Footer on every page: "Generated YYYY-MM-DD HH:MM (<zone>) by happybabytracker · Page n of N" | spec 12 (call on contents), spec 6.2 | sections added or removed |
| 24 | Units | The **requesting** parent's four preferences (weight, length, volume; temperature in Task 16), read from the session's parent row and formatted from the stored column of that unit, never re-converted; Go formatters mirror `ui/src/lib/format.ts` with the same test examples | spec 6.2, 12 | — |
| 25 | UI flow | `/kids/{kid}/export`: a plain `<form method="post" action="/kids/{kid}/export/pdf">` with `from`/`to` date inputs (default the last `EXPORT_DEFAULT_DAYS = 30` kid-local days ending today) and one checkbox per module (enabled modules ticked). The POST route validates, calls the API with `EXPORT_TIMEOUT_MS = 25000`, and returns `new Response(apiResponse.body, …)` (streamed, not buffered). On a 422 or 503 it answers 303 to `/kids/{kid}/export?error=<field|busy>&from=…&to=…&modules=a,b`; the page shows the field message and re-fills the form from these enumerated values. `modules` in that URL is one enumerated parameter beyond spec 13.3's `from`/`to`/`types` (proposal; carrying nothing would lose the typed values) | SvelteKit actions cannot return a binary body; no-JS baseline | — |
| 26 | Entry points | Milestones: a quick-log tile through `PAGE_ROUTE.milestones` (it sits in MODULES order after feeding in the 0–11 and 12–23 bands), `KidLinks` gains "Milestones" and "Export PDF". The navigation bar is unchanged (HANDOFF: Home, Timeline, Growth, Account) | plan 4/5 precedent | — |
| 27 | Band lookback | Milestones join `moduleTables` with `{"milestones_reached", "created_at", false}`: a milestone **recorded** (not reached) inside the 60-day lookback keeps the module on at 24 months. It uses the instant column the mechanism already compares | HANDOFF plan-3 owner decision (60 days); spec 7.2 | a date-column variant |
| 28 | Catalog review | `docs/milestones/cdc.md`, generated by `UPDATE_DOCS=1 go test ./internal/milestones/` (a table per age, one row per item: id, domain, text, source page; then retired items; then the PDF-wording differences), and a test that fails when it is stale. The owner reads that file's diff in the PR | plan 5 Open value 16; spec 17 "subject to the owner's review" | — |
| 29 | Update runbook | `docs/runbooks/updating-milestone-catalog.md`: where to look, how to change `edition`/`retrieved`, add rows, retire rather than delete, regenerate the review document, the traps from the research note | plan 5 Open value 17 | — |
| 30 | PDF metadata | Title "happybabytracker export" (no name), Producer "happybabytracker", no Author, Subject or Keywords | privacy of a forwarded file | — |

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
- [ ] **Step 1: Failing tests** on in-memory files through an unexported `parse(name string, r io.Reader) (Catalog, error)`. Each error names the file, line and column and never the cell's text, and a test pins one message per refusal. Cases: a valid file parses; a missing provenance key; a duplicate id; an id not matching `^[0-9]{1,2}m-(se|lc|cg|mp)-[a-z0-9-]{1,48}$` or not starting with its own `age_months`; an unknown domain; an `age_months` outside the twelve; empty or over-long text (`MilestoneCatalogTextMaxLen = 200`); text with a control character; a `source_url` not `https://www.cdc.gov/`; `retired` that is not empty and not equal to an edition-like string; `Load()` idempotent and a failing injected loader remembered.
- [ ] **Step 2:** `cd api && go test ./internal/milestones/ -race` → FAIL. **Step 3: Implement.** Constants: `MilestoneCatalogTextMaxLen = 200`, `MilestoneCatalogAges = []int{2,4,6,9,12,15,18,24,30,36,48,60}` (a `var` beside `VaccineSchedules`). **Step 4:** PASS; `gofmt -l .`, `go vet ./...` and staticcheck as in CLAUDE.md. **Step 5: Commit** `feat(api): milestone catalog format and loader`.

## Task 2: Transcribe the CDC catalog (owner reviews)

**Files:** Create `api/internal/milestones/data/cdc.csv`, `api/internal/milestones/review_test.go`, `api/internal/milestones/data_test.go`, `docs/milestones/cdc.md` (generated), `docs/runbooks/updating-milestone-catalog.md`; modify `api/internal/milestones/load.go` (embed), `README.md` (§ Third-party data: one row).

- [ ] **Step 1: Failing tests:** `TestShippedCatalogLoads` (provenance keys non-empty; `origin_url` is `https://www.cdc.gov/act-early/milestones/index.html`; `retrieved` is a date; the attribution contains "Source: CDC", "does not endorse" and "free of charge"); `TestPinnedItems` (the seven examples in `docs/superpowers/research/milestones/cdc-milestone-checklists.md` §2, by age, domain and exact text); `TestEveryAgeHasEveryDomain` (12 ages × 4 domains, each non-empty); `TestNoJudgementWords` (no item text matches the Global Constraints denylist as a whole word, case-insensitive; **on a match, stop and ask the owner**, never reword); `TestReviewDocIsCurrent` (renders the Markdown of Open value 28 and compares it with `docs/milestones/cdc.md`; `UPDATE_DOCS=1` rewrites it).
- [ ] **Step 2:** FAIL. **Step 3: Transcribe** from the twelve HTML age pages through the WebFetch tool (curl gets a 403 from cdc.gov), on the day of the task, in page order. Set `retrieved` to that day and `edition` to "Learn the Signs. Act Early. milestone checklists (2022 revision), web pages updated <the dates the pages show>". Record in `omitted` what is left out (Open value 12). Generate the review document, which lists any PDF wording difference found (pronouns). Write the runbook. **Step 4:** PASS. **Step 5: Commit** `feat(api): CDC milestone catalog (LTSAE checklists)`. The PR description asks the owner to read `docs/milestones/cdc.md`.

## Task 3: Age basis and checklist positions

**Files:** Create `api/internal/milestones/window.go` (+ test); modify `api/internal/kidtime/kidtime.go` only if a helper is missing (+ test).

**Interfaces — Produces:**
```go
type Position string // PositionEarlier, PositionCurrent, PositionNext, PositionLater
type AgeBasis struct{ ChronologicalMonths int; CorrectedMonths *int; BasisMonths int }
func AgeBasisFor(birthDate string, gestWeeks, gestDays *int, today string, loc *time.Location) (AgeBasis, error) // Open value 13
func Positions(ages []int, basisMonths int) map[int]Position                                                    // Open values 3, 14; ages ≥ 72 months: every age is earlier
```
- [ ] **Step 1: Failing tests:** `TestAgeBasis` (term kid; a 32-week kid at 3 months chronological is 1 month corrected and its basis is 1; the same kid on the day before and on the second birthday, where correction ends on the birthday as in plan 4; a 36w6d kid is corrected and a 37w0d kid is not; birth dates on the 29th–31st; a Pacific/Auckland zone); `TestPositions` (basis 0 and 1: no current, next 2; basis 2: current 2, next 4; basis 3: current 2, next 4, CDC's "younger age"; basis 14: current 12, next 15; basis 60: current 60, no next; basis 71: current 60; basis 72: all earlier; ages are only those passed in).
- [ ] **Step 2–4:** `go test ./internal/milestones/ ./internal/kidtime/ -race`, FAIL then PASS. **Step 5: Commit** `feat(api): milestone age basis and checklist positions`.

## Task 4: Migration and store

**Files:** Create `api/internal/store/migrations/<next>_milestones.sql` (the next free number, see "Dependency on plan 6"), `api/internal/store/milestones.go` (+ test); modify `api/internal/model/model.go` (or `tracker.go`), `api/internal/store/store_test.go`, `api/internal/store/modules.go` (+ `modules_test.go`), `api/internal/constants/constants.go`.

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

- [ ] **Step 1: Failing tests:** `wantTables` gains `milestones_reached`; both `schema_version` assertions become the new number; the previous-version upgrade test (every table's rows survive); CRUD round trip; `TestMilestoneBelongsToItsKid` (another kid's id → `ErrNotFound` on read, update and delete); `TestMilestoneWriteByDisabledActor`; `TestDuplicateCatalogItem` (`ErrDuplicate`, and the same item for another kid is fine); `TestMilestoneCap` (the 1001st → `ErrConflict`; a delete frees a slot); the CHECKs refuse both item and text, neither, an empty string and a malformed date; cascade on kid delete; `TestMilestonesKeepModuleOnAt24Months` (a row created 10 days before the 24-month band start → stored override on; one created 61 days before → none).
- [ ] **Step 2–4:** `go test ./internal/store/ -race`, FAIL then PASS. **Step 5: Commit** `feat(api): milestones migration and store`.

## Task 5: Milestones API

**Files:** Create `api/internal/httpapi/milestones.go` (+ test); modify `routes.go`, `matrix_test.go`, `cmd/happybabytracker-api/main.go` (`milestones.Load()` before listening), `api/internal/constants/constants.go`.

**Interfaces — Produces** (class in brackets): `GET /v1/kids/{kid}/milestones` [Caregiver]; `POST /v1/kids/{kid}/milestones` [Caregiver] (201); `GET|PATCH|DELETE /v1/kids/{kid}/milestones/{entry}` [Caregiver].
`GET …/milestones` → `{"catalog":{id,name,authority,edition,retrieved,origin_url,licence,attribution},"age":{"chronological_months","corrected_months"|null,"basis_months"},"checklists":[{"age_months","position","items":[{"id","domain","text","reached":{"id","reached_on","note"}|null}]}],"custom":[{"id","text","reached_on","note"}],"retired":[{"id","item","text","reached_on","note"}]}`. Checklists are in age order, items in file order, and `later` checklists are included for the UI to hide. Entry view: `{id, item|null, text, reached_on, note}`, where `text` is the catalog text for an item row. Input: `{item}` or `{text}`, plus `reached_on` and `note`. Validation order `item`/`text`, `reached_on`, `note`. PATCH may change `reached_on`, `note` and (custom only) `text`; `item` is immutable (→ 422 `item`). Errors: unknown or retired item → 422 `item`; both or neither → 422 `item`; reached twice → 409 `already_reached`; cap → 409 `limit`; a date after the kid-local today or before the birth date → 422 `reached_on`. 503 when the catalog failed to load (as `schedulesLoaded`).

- [ ] **Step 1: Failing tests:** `TestMilestonesShape` (`sameKeys` on every object, using a small test catalog); `TestMilestonesPositionsThroughAPI` (a preterm kid shows both ages and the corrected-basis positions); `TestRolesAndNotFound` (guardian and caregiver can do everything; the no-grant, unknown-kid and other-kid-entry 404 bodies are byte-identical; admin 403); `TestMilestoneValidation` (every refusal above, plus unknown fields, `null` on required fields, text of 201 runes, whitespace-only text, a note over `MaxNoteLen`); `TestUndoReturnsItemToChecklist`; `TestRetiredRowListed`; `TestNoJudgementInViews` (every string field of every view, except the catalog's own `text` and `attribution`, avoids the denylist); `TestNoValuesInLogs` (`wantCleanLog` with a distinctive custom text, note and date). Matrix: the five routes in `routeClasses`; `matrixParams["entry"]` gains the `/milestones/` case with a fixture row.
- [ ] **Step 2–4:** `go test ./... -race` FAIL then PASS, including the matrix. **Step 5: Commit** `feat(api): milestones read model and reached entries`.

## Task 6: UI plumbing for milestones

**Files:** Modify `ui/src/lib/types.ts`, `ui/src/lib/constants.ts`, `ui/src/lib/server/constants.ts`, `ui/src/lib/server/api.ts` (+ `api.test.ts`), `ui/src/lib/paths.ts`, `ui/e2e/stub-api.ts`; create `ui/src/lib/server/milestone-form.ts` (+ test), `ui/src/lib/milestones.ts` (+ test), `ui/e2e/stub-milestones.spec.ts`.

**Interfaces — Produces:** types `MilestoneDomain`, `MilestonePosition`, `MilestoneItem`, `MilestoneChecklist`, `Milestones`, `MilestoneEntry`; client `api.milestones(kidId)`, `.getMilestone`, `.createMilestone`, `.updateMilestone`, `.deleteMilestone` (the four-edit pattern: `Endpoint` union, parsers with `ShapeError`, methods through `withId`/`withKidEntry`, types; a malformed id answers not-found without a request); `readMilestoneForm(form, mode, zone)` (kid-local date default today, `item` hidden field or `text`, `note`; edit sends only changed fields, hidden originals fail closed); formatters `checklistTitle(ageMonths)` ("By 2 months", "By 1 year", "By 15 months", "By 2 years", "By 30 months", "By 3 years"; CDC's own age labels), `domainLabel(domain)` (CDC's headings without "Milestones"), `reachedLabel(date)` ("Reached on 4 March 2026"); paths `milestonesPath`, `milestoneNewPath`, `milestoneEntryPath`; constants `MILESTONE_TEXT_MAX = 200` (client mirror). The stub implements the five routes with the API's validation order, statuses and 404 rules over a **clearly marked stub catalog** that no test asserts numbers from.

- [ ] **Step 1: Failing tests (RED):** formatters for every CDC age; `readMilestoneForm` (no date → today; a future date and hostile text refused with value-free messages; text over 200 or blank → field `text`; edit omits untouched fields; missing originals fail closed); `api.test.ts` (list, create 422 with `field`, 409 `already_reached`, update, delete, bad id → not-found without fetch, malformed bodies → `error`); stub self-tests.
- [ ] **Step 2–5:** `cd ui && pnpm test` → FAIL; implement; `pnpm check && pnpm lint && pnpm test && pnpm test:e2e --trace off`; commit `feat(ui): milestones client, forms, formatters and stub routes`.

## Task 7: UI: the Milestones page, its forms and entry points

**Files:** Create `ui/src/routes/(app)/kids/[kid]/milestones/{+page.server.ts,+page.svelte}`, `…/milestones/new/{+page.server.ts,+page.svelte}`, `…/milestones/[entry]/{+page.server.ts,+page.svelte}`, `ui/src/lib/components/milestones/Checklist.svelte`; modify `ui/src/lib/quick-log.ts` (`PAGE_ROUTE.milestones`) (+ test), `ui/src/lib/components/KidLinks.svelte`, `ui/e2e/{home,kids}.spec.ts`; create `ui/e2e/milestones.spec.ts`.

**Interfaces:** the page header shows the age as the home does (`ageHeader`: both ages when corrected). Then the `current` and `next` checklists, each grouped by domain. Every item shows its text and either a "Reached" form (one tap: a submit that records today, with "Choose a date" linking to `new?item=<id>`) or, when reached, "Reached on <date>" with an edit link. A `<details>` "Earlier ages" holds the earlier checklists. "Your milestones" lists the custom rows, with "Add a milestone". Rows whose item was retired are listed with their text. The CDC sentence and attribution (Open value 12) sit under the lists. There is no colour, icon, count, progress bar or "not yet" text, and nothing distinguishes an unreached item except the absence of a date. The page is reachable with the module off. Create and edit follow the plan-5 pattern (`failAction`, `fail(422, {form, values, field, message})`, no-JS, delete behind `<details>`, 303 back to the page, the same 404 page for no grant, unknown kid and another kid's entry; 409 `already_reached` → "Already marked as reached"; 409 `limit` → "This child has the maximum number of recorded milestones").

- [ ] **Step 1: Failing tests (Playwright, both projects, axe):** a stub kid at 9 months shows "By 9 months" and "By 1 year" open and "Earlier ages" closed; a preterm stub kid shows both ages and the corrected checklist; one-tap reached records today and shows the date (`submit()` waits for the response); choose a date in the past, read it back and edit it; a future date shows the field message with values kept; undo (delete) returns the item to its list; add, edit and delete a custom milestone; 201 characters is refused; a caregiver can do all of it; a no-grant parent and another kid's entry id get the same 404 page; `TestNoJudgementOnPage`, an e2e check that the rendered text of the page avoids the denylist outside the catalog items; no horizontal overflow at 390 px with a 200-character unbroken custom text; no-JS reached and create; the quick-log tile appears with the module on and hides with it off; `KidLinks` shows Milestones; `quick-log.test.ts` updated.
- [ ] **Step 2–5:** RED; implement; the full UI pipeline; commit `feat(ui): milestones page, reached and custom milestones`.

## Task 8: The PDF dependency and the document writer

Do this only after question 1 is answered. If the answer is "no", replace this task with a stdlib writer that has the same interface and add a task for the Helvetica width tables.

**Files:** Modify `api/go.mod`, `api/go.sum` (`go get codeberg.org/go-pdf/fpdf@v0.12.0`, then `go mod tidy`; the diff must add exactly that one module), `api/internal/constants/constants.go`; create `api/internal/pdfdoc/{doc.go,encode.go}` (+ `doc_test.go`, `encode_test.go`, `textdump_test.go`).

**Interfaces — Produces:**
```go
type Doc struct{ /* wraps *fpdf.Fpdf */ }
func New(created time.Time, footer func(page, pages int) string) *Doc // A4, Open value 8 geometry, compression off, catalog sort, fixed producer and title, no author
func (d *Doc) Heading(text string)
func (d *Doc) Line(label, value string)                               // a key/value row of the profile block
func (d *Doc) Paragraph(text string)                                  // wrapped, small print (attributions)
func (d *Doc) Table(cols []Col, rows [][]string)                      // header repeated after each page break; cells wrap; no cell leaves the page
type Col struct{ Title string; WidthMM float64 }
func (d *Doc) Chart(c Chart)                                          // axes, curves as polylines, points as small squares; grey and black only
func (d *Doc) Substituted() bool                                      // any rune replaced by ExportFallbackRune
func (d *Doc) WriteTo(w io.Writer) (int64, error)                     // refuses when the document exceeds ExportMaxBytes
// encode.go
func Encode(s string) (b []byte, substituted bool)                    // cp1252; every other rune (and every control character) → '?'
```
Constants: `ExportPageWidthMM = 210`, `ExportPageHeightMM = 297`, `ExportPageHeightUsableMM = 279.4`, `ExportMarginMM = 12`, `ExportFontSizeBody = 9`, `ExportFontSizeHeading = 12`, `ExportFontSizeSmall = 7`, `ExportFallbackRune = '?'`, `ExportMaxBytes = 4 << 20`, `ExportProducer = "happybabytracker"`, `ExportTitle = "happybabytracker export"`.

- [ ] **Step 1: Failing tests:** `encode_test.go` (Latin-1 identity; the 27 cp1252 extras; "Zoë Müller-Señor" exact; "Mia 😀" → "Mia ?"; "李小明" → "???"; a combining sequence; NUL, BEL and U+2028 → "?"; `substituted` true or false); `textdump_test.go` (the test-only extractor of `pdf-text-and-verification.md` §4: it scans each content stream's `(…)Tj` strings, unescapes and decodes cp1252, and returns `[]string` per page; tested on a document written by `Doc`); `doc_test.go`: `TestDeterministic` (two documents with the same clock are byte-identical; a different clock changes only the date bytes); `TestGolden` (a fixed small document against `testdata/doc.golden.pdf`; `UPDATE_GOLDEN=1` rewrites it); `TestUnbrokenNoteWraps` (500 × "x" in a 60 mm column: every string's x + width ≤ the page width − margin, and every y within the 279.4 mm usable height); `TestTableRepeatsHeader` (200 rows → several pages, each starting with the header row); `TestFooterOnEveryPage` ("Page n of N" with the right N); `TestNoForbiddenObjects` (the raw bytes contain no `/JavaScript`, `/JS`, `/URI`, `/Launch`, `/EmbeddedFile`, `/XObject`, `/FontFile`, `/Encrypt`; there is exactly one `/BaseFont /Helvetica` and at most one `/Helvetica-Bold`); `TestStructure` (starts `%PDF-1.`, ends `%%EOF\n`); `TestSizeCap` (an oversized document → error, nothing written).
- [ ] **Step 2:** `go test ./internal/pdfdoc/ -race` → FAIL. **Step 3: Implement.** Call no fpdf API outside `pdfdoc`; no `HTMLBasicNew`, `Image*`, `Link*`, `AddFont*`, `SetProtection` or `SetXmpMetadata`. Every string goes through `Encode` and fpdf's own translator is not used. **Step 4:** PASS, plus the full API pipeline; `CGO_ENABLED=0 go build ./...` passes; record the stripped binary size before and after in the PR. **Step 5: Commit** `feat(api): PDF document writer on go-pdf/fpdf (core fonts only)`.

## Task 9: Display formatters for the PDF

**Files:** Create `api/internal/pdfexport/format.go` (+ test).

**Interfaces — Produces:** `FormatWeight(units.Weight, model.Units)`, `FormatLength(units.Length, model.Units)`, `FormatVolume(units.Volume, model.Units)`, `FormatMilk(units.Milk, model.Units)`, `FormatDuration(seconds int)`, `FormatAge(birthDate, today string)`, `FormatAgeWithCorrection(...)`, `FormatInstant(t time.Time, loc *time.Location)` ("2026-03-04 14:05"), `FormatDate(date string)`. They take each value from the stored column of the requester's unit (kg from grams, lb + oz from hundredths of an ounce, and so on) and never convert.

- [ ] **Step 1: Failing tests** that copy the examples of `ui/src/lib/format.test.ts` and `growth.test.ts` (the same inputs give the same strings) plus `FormatMilk` for both unit sets ("weight gain 62 g, about 60 ml"; "weight gain 2.19 oz, about 2.04 fl oz"); DST: an instant at 01:30 EST and one at 01:30 EDT on the fall-back day print distinctly with their offsets.
- [ ] **Step 2–5:** FAIL; implement; PASS; commit `feat(api): export display formatters mirroring the UI`.

## Task 10: Export route, snapshot, profile and daily-tracker sections

**Files:** Create `api/internal/pdfexport/{export.go,profile.go,trackers.go}` (+ tests), `api/internal/store/export.go` (+ test), `api/internal/httpapi/export.go` (+ test); modify `routes.go`, `matrix_test.go`, `api/internal/constants/constants.go`.

**Interfaces — Produces:**
```go
// store: one read-only transaction, so every section sees one state
type ExportSnapshot struct{ Kid model.Kid; Entries map[string][]TimelineItem /* per type, ≤ cap+1 */; Days []DaySummary; Growth []model.Growth; Doses []model.VaccineDose; Administrations []model.Administration; Dismissals map[string]string; Milestones []model.Milestone }
func (s *Store) ExportSnapshot(ctx context.Context, kidID string, modules []model.Module, loc *time.Location, from, to, now time.Time, rowCap int) (ExportSnapshot, error)
// pdfexport
type Request struct{ From, To string; Modules []model.Module; Units model.Units; Now time.Time; Loc *time.Location }
func Render(w io.Writer, snap store.ExportSnapshot, req Request) error // sections in model.Modules order
// httpapi
// POST /v1/kids/{kid}/export [Caregiver]
```
The handler resolves the kid and the caller's units, validates (Open value 21), takes the export semaphore (`MaxConcurrentExports`; busy → 503 `busy`), reads the snapshot, renders into the `pdfdoc` buffer and only then writes the headers of Open value 20 and the bytes. Every error happens before the first byte, and nothing about the request is logged beyond the middleware's line. Constants: `ExportMaxDays = 93`, `ExportMaxRowsPerSection = 3000`, `ExportMaxRows = 6000`, `MaxConcurrentExports = 1`, `ExportModules = []string{...}` (a `var`).

- [ ] **Step 1: Failing tests:** store: `TestExportSnapshotIsOneTransaction` (a write between two section reads is not seen); the row cap returns cap+1 so overflow is detectable. httpapi: `TestExportValidation` (Review Focus 5, each field); `TestExportHeaders` (status, `Content-Type`, the attachment file name with only dates, `no-store`, `nosniff`); `TestExportRoles` (guardian and caregiver 200; no grant, unknown kid → identical 404; admin 403; a disabled caregiver 404); `TestExportBusy` (the semaphore held → 503 `busy`, released after); `TestExportEmptyKid` (Review Focus 1, through the text extractor); `TestExportTrackers` (one entry of each tracker with distinctive values: each value appears as the timeline would word it, in the requester's units; advanced bottle sources and pumping sides; weigh-ins and the estimate; a running timer prints "running"); `TestExportTwoUnits` (Review Focus 6); `TestExportDST` (Review Focus 7, `America/New_York`); `TestExportCaps` (93 days × 20 bottles: 1860 rows, no overflow; 3001 → the overflow line once; the `ExportMaxRows` budget); `TestExportNameEncoding` (Review Focus 4 and the notice line); `TestExportNoJudgement` (profile and tracker text avoids the denylist); `TestNoValuesInLogs` (`wantCleanLog`: the kid name, a note, a volume, the range dates and the module names never appear); matrix: `"POST /v1/kids/{kid}/export": Caregiver`.
- [ ] **Step 2–4:** `go test ./... -race` FAIL then PASS. **Step 5: Commit** `feat(api): PDF export route with profile and daily trackers`.

## Task 11: Growth section, with percentiles, attribution and charts

**Files:** Create `api/internal/pdfexport/growth.go` (+ test); modify `export.go`.

**Interfaces:** a table of every measurement up to `to` (Open value 5), dated, with the age (`FormatAgeWithCorrection`, "before expected term" rows unscored as on screen), weight, length/height, head, BMI, then z and percentile per measure under the kid's **current** source (the same `newGrowthView` scoring through one shared function, not a copy), or the unscored reason in the screen's words. Under the table come the source's `Attribution` and `Licence` lines and, for `who`, WHO's adaptation disclaimer from `growth.Info.Notice`. Then one chart per measure with at least one scored point: the source's percentile curves (`growth.Curves`, the same lines the UI draws) over the points' age range, and the kid's points. Axes are labelled in the requester's unit, curves are grey hairlines labelled at the right edge with the source's own percentile names, points are black squares, and nothing is filled or coloured. A measure the source does not cover prints "<source> does not cover this measure at this age" (the screen's wording).

- [ ] **Step 1: Failing tests:** `TestGrowthSectionScores` (the same z and percentile strings as `GET …/growth` for the same rows under `who` and `cdc`; switching the source changes the PDF and no stored row); `TestGrowthAttribution` (the WHO attribution, licence and disclaimer present when `who`; the CDC lines when `cdc`); `TestGrowth3000Rows` (3000 measurements: all listed, size and time within the caps, measured in the test); `TestGrowthCorrectedAge`; `TestGrowthNoJudgement`; `TestChartGeometry` (every curve and point inside the chart box; no point drawn for an unscored row).
- [ ] **Step 2–5:** FAIL; implement; PASS; commit `feat(api): export growth table, percentiles and charts`.

## Task 12: Vaccination and milestones sections

**Files:** Create `api/internal/pdfexport/{vaccines.go,milestones.go}` (+ tests); modify `export.go`.

**Interfaces:** vaccination: the evaluation on the kid-local **today** through the same `evaluateKid` code path (moved to a shared function if needed, not copied), "Schedule: <authority>, <edition>", the schedule's `attribution`, each group's lines with status words, the doses given up to `to` (date, antigens by name, product, lot, note) and "Other". With no schedule: "No schedule chosen", then the doses. Milestones: the catalog attribution and CDC's sentence (Open value 12), then the reached catalog rows up to `to` (date, checklist title, domain label, text, note) in date order, then custom rows, then retired rows. There is no list of unreached items and no count of them.

- [ ] **Step 1: Failing tests:** `TestVaccinationSection` (a US kid: caption, attribution, a given line with its date, an overdue line labelled `overdue`; no schedule: doses still listed); `TestMilestonesSection` (reached and custom rows, no unreached item text appears in the PDF, attribution present, no denylist word outside catalog text); `TestRecordSectionsUpToTo` (a dose and a milestone dated after `to` are absent).
- [ ] **Step 2–5:** FAIL; implement; PASS; commit `feat(api): export vaccination and milestones sections`.

## Task 13: UI: the export page and the streaming route

**Files:** Create `ui/src/routes/(app)/kids/[kid]/export/{+page.server.ts,+page.svelte}`, `ui/src/routes/(app)/kids/[kid]/export/pdf/+server.ts`, `ui/src/lib/server/export-form.ts` (+ test), `ui/e2e/export.spec.ts`; modify `ui/src/lib/server/api.ts` (+ test; endpoint `export_pdf` returns `{ok:true, body: ReadableStream, contentType, disposition}` and never parses JSON on 200), `ui/src/lib/server/constants.ts` (`EXPORT_TIMEOUT_MS = 25000`, `EXPORT_DEFAULT_DAYS = 30`), `ui/src/lib/constants.ts` (`EXPORT_MAX_DAYS = 93` mirror, `EXPORT_MODULES`), `ui/src/lib/paths.ts`, `ui/src/lib/components/KidLinks.svelte`, `ui/e2e/stub-api.ts` (the route returns a fixed, clearly marked stub PDF of a few hundred bytes and mirrors the API's 422 fields, 404 and 503 `busy`).

**Interfaces:** Open value 25. `readExportForm(form, zone)` returns `{from, to, modules}` or `{field}`. The POST route reads the form, calls `api.exportPdf(kidId, input)`, and on 200 returns `new Response(body, {status: 200, headers: {content-type, content-disposition}})`. The hook adds `no-store`. 404 → the same 404 page as every kid route; 422/503 → 303 to the page with `error`. The page explains in one sentence that the file is made now and not kept.

- [ ] **Step 1: Failing tests:** `export-form.test.ts` (defaults, a 94-day range → field `to`, no module → `modules`, an unknown module refused, hostile dates); `api.test.ts` (`exportPdf` streams the body without reading it; a 422 → `invalid` with field; a timeout → `unavailable`; the log line keeps its fixed shape with no range or modules); Playwright, both projects, axe: the page lists the modules with the enabled ones ticked; submitting starts a download (`page.waitForEvent('download')`, which waits for the response) named `happybabytracker-<from>-<to>.pdf` whose bytes are the stub's; an empty module set shows the field message with the typed dates kept; a caregiver can export; a no-grant parent gets the 404 page on both the page and a direct POST; an admin is redirected; it works without JavaScript; there is no horizontal overflow at 390 px.
- [ ] **Step 2–5:** RED; implement; the full UI pipeline; commit `feat(ui): export page streaming the PDF`.

## Task 14: Smoke

**Files:** Modify `scripts/smoke.sh`.

- [ ] **Step 1: Extend `scripts/smoke.sh`** (failing first against the images from `main`). Step **4k, milestones**: as the guardian, open the smoke kid's Milestones page and assert the CDC attribution text, a checklist title for the kid's age, and that the page contains none of the denylist words outside item text. Mark one item reached today through the real one-tap form, then one with an earlier date through the new form; add a custom milestone with a distinctive text; edit its date and delete one through the real forms (scrape the hidden originals as 4h does). The caregiver marks one. The no-grant parent and the admin get the 404 page or a redirect on `/kids/{id}/milestones`, `/milestones/new` and `/milestones/{entry}`. Step **4l, export**: as the caregiver, `POST /kids/{id}/export/pdf` without JavaScript (the smoke's form helper), for the 4a–4j range and every module. Assert 200, `content-type: application/pdf`, `content-disposition: attachment; filename="happybabytracker-…"` with no name in it, `cache-control: no-store`; that the body starts with `%PDF-1.` and ends with `%%EOF`; that `/Type /Pages` and a `/Count` ≥ 1 are present. Assert with `grep -a` the literal cell strings `(<smoke kid name>)Tj`, one 4a bottle volume as printed, the growth source's attribution opening words, `(Schedule: Centers for Disease Control and Prevention (CDC), …)` as escaped, the custom milestone's text, and the WHO or CDC licence line. Assert that the strings of the milestones and growth sections contain no denylist word. A second export started while the first is running is either 200 or 303 `busy`, never 5xx. The no-grant parent's POST is the 404 page, and the admin's is a redirect. **Step 9** gains the custom milestone text in the log scan, and the export's range dates and module names join the scan for the API and UI logs. **Step 10** keeps the ceilings (60 MiB API, 140 MiB UI): run one export inside the login-burst window and report the readings.
- [ ] **Step 2:** `docker build -t hbt-api:dev api && docker build -t hbt-ui:dev ui && scripts/smoke.sh hbt-api:dev hbt-ui:dev` → every line `PASS:`; `shellcheck scripts/smoke.sh`. If a ceiling fails, tune `ExportMaxRows` from the measured readings and record the numbers in the PR. Never raise the ceilings.
- [ ] **Step 3: Commit** `test: smoke covers milestones and the PDF export`.

## Task 15: Docs, image notice and HANDOFF

**Files:** Modify `README.md`, `api/Dockerfile` (copy fpdf's `LICENSE` to `/usr/share/doc/happybabytracker-api/fpdf/`; the `org.opencontainers.image.licenses` label stays as is, since fpdf is MIT), `docs/superpowers/HANDOFF.md`, `docs/milestones/README.md` (index).

- [ ] **Step 1: Docs:** README sections "Milestones" (what is shown, corrected age, never a judgement, the CDC conditions, the update runbook) and "PDF export" (who can export, what is in it, 93 days, on demand and never stored, characters outside WinAnsi print as `?`, A4 that prints on Letter); Third-party data: the CDC catalog row and `codeberg.org/go-pdf/fpdf` (MIT). HANDOFF: plan 7 done, the owner's answers to the eight questions, the unverified items below, and which migration number was taken. **Step 2:** the whole pipeline in CLAUDE.md (API, UI, images, smoke) green. **Step 3: Commit** `docs: plan 7 done (milestones and PDF export)`.

## Task 16 (after plan 6 merges): sickness and medication sections

**Files:** Create `api/internal/pdfexport/{sickness.go,medication.go}` (+ tests); modify `export.go`, `api/internal/constants/constants.go` (`ExportModules` gains both), `ui/src/lib/constants.ts` (`EXPORT_MODULES`), `ui/e2e/stub-api.ts`, `scripts/smoke.sh`.

**Interfaces:** sickness: each symptom entry (time, symptom, severity, temperature in the requester's temperature unit with the method, note) in the range. Medication: each course active in the range (name, dose text, schedule as worded on screen, start, end) and the doses given in the range. Nothing about reminders, and no wording about doses being due or missed. The denylist applies.

- [ ] **Step 1–5:** failing tests as Task 10 (units for temperature, DST, roles, no values in logs, no judgement); implement; the full pipeline and smoke green; commit `feat(api): export sickness and medication`.

---

## Done when

- API and UI pipelines and `scripts/smoke.sh` pass on the branch head, and CI is green.
- The access matrix covers all six new routes and fails if one is missing.
- A guardian and a caregiver see the milestones around the kid's corrected age, mark and unmark them, add custom ones, and no screen or PDF text judges. The owner has reviewed `docs/milestones/cdc.md` in the PR.
- A caregiver downloads a PDF of one kid for a range and the chosen modules, in their own units and the kid's time zone, with every source's attribution where its data is shown. Smoke reads its text without a PDF viewer.
- The final report states which image content changed (`api/` sources, embedded catalog data, **a new Go module** `codeberg.org/go-pdf/fpdf`, a Dockerfile `COPY` of its licence; `ui/` sources; no npm change), that the edge still needs its `bump-happybabytracker` PR merged and a deploy, that nothing else on the edge changes, and which migration number this plan took.

## Not verified (state in the PR)

- CDC reuse: "Use of Agency Materials" and a training page ("Reproducible (no copyright)") support public-domain reuse. The checklist pages themselves carry no licence statement, and the 2022 revision was written with AAP, whose article was not read (403). This is not legal advice, and CDC was not contacted.
- Whether a pronoun is "substantive content": avoided by transcribing verbatim.
- That the 2026 page updates changed nothing except pronouns: checked for the 2-month and 4-year pages only.
- The 159-item count comes from the PDF's bullet glyphs and is confirmed only by the transcription.
- CDC states no age at which corrected age stops for milestones. The second birthday is the app's rule (spec 7.1, plan 4).
- The WHO licence reading of plan 4 now also covers derived percentiles printed in a PDF. Whether ShareAlike reaches such a PDF was not checked.
- `codeberg.org/go-pdf/fpdf`'s roadmap ("use UTF-8 exclusively") could remove the core-font path in a later release. v0.12.0 is pinned, and the golden test catches a change at bump time.
- Memory figures (17.4 MB peak at 8000 rows) come from standalone processes, not from the 96 MB container with the API's own heap; Task 14 measures the real one.
- gopdf was not built or measured. Its exclusion rests on its need for a TrueType font.
- Download behaviour of an installed PWA on iOS (share sheet or viewer) was not tested on a device.
- The midnight-DST flaw in `kidtime` noted in HANDOFF also reaches `AgeBasisFor` through `AgeMonths` and `CorrectedBirthDate` in zones whose DST starts at midnight. It is not fixed here (minimum change). The follow-up already listed in HANDOFF covers it.
