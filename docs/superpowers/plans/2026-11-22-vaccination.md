# Vaccination Implementation Plan (plan 5)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A guardian picks a national vaccination schedule for a kid (United States, United Kingdom, Mexico), anyone with access records administered doses (date required, lot and product optional), and the kid's vaccine screen shows every schedule line as given, due, upcoming, overdue or dismissed, with a soft "Due" list on the home screen. Nothing judges a dose's validity.

**Architecture:** A new Go package `api/internal/vaccines` embeds one CSV per schedule (per-antigen numbered series with age windows, provenance header) plus a small antigen and product catalogue, and a pure `Evaluate` function matches recorded doses to lines by count per antigen. Migration `0004` adds `vaccine_doses`, `vaccine_dismissals` and `kids.vaccine_schema`. The API exposes the evaluation, administrations (one administration writes one row per antigen) and dismissals; the home read model gains `due`. The UI adds one Vaccines page with record, edit and dismiss forms, a schedule select in kid settings, a Due section on the home and a quick-log tile.

**Tech Stack:** Go 1.27 (`encoding/csv`, `embed`), SQLite via `modernc.org/sqlite`; SvelteKit 3 / Svelte 5 runes, Tailwind 4. No new Go or npm dependency.

**Spec:** `docs/superpowers/specs/2026-11-20-happybabytracker-design.md` sections 2 (G5, G7, G9), 5.2, 6.1, 6.3, 7.1–7.3, 9, 13.3, 14, 16. Earlier plans: foundation, daily trackers, growth (`docs/superpowers/plans/`). Research notes (retrieved 2026-11-22, one per country, with draft tables and traps): `docs/superpowers/research/vaccination/{us-cdc,uk-ukhsa,mx-censia,ch-bag}.md`.

## Global Constraints

- Every limit, cap and duration lives in `api/internal/constants/constants.go` (API) or `ui/src/lib/server/constants.ts`, with client-safe mirrors in `ui/src/lib/constants.ts` (CLAUDE.md rule 7). No literal elsewhere.
- Vaccination always uses **chronological** age (spec 7.1): never call `kidtime.CorrectionDays`, `CorrectedBirthDate` or `AgeBasisOn` from vaccination code.
- Doses are stored per antigen: one administration (date, optional lot, optional product name) writes one row for each antigen it covers, all sharing one administration id (spec 9). Switching a kid's schema keeps every dose and changes only what is shown; dismissals are stored per kid, schema and line and survive only in the schema they were made in.
- Matching is by count only: the kid's Nth recorded dose of an antigen fills the Nth line of that antigen's series, whenever it was given. The app never judges whether a dose was valid, too early, too late or too close to another (spec 9, non-goals).
- **Overdue is a soft flag**: a label and a place in the home "Due" list. No push, no blocking, and it never applies to recurring, supplementary, risk-group or "may not be needed" lines.
- A schedule file records its issuing authority, edition, origin URL, retrieval date, licence and attribution; the screen shows "Schedule: <authority>, <edition>" (spec 9; anti-goal in spec 2).
- Dates are `YYYY-MM-DD` in the kid's zone; a dose has a date, not an instant. "Today" is the kid-local today.
- A caller with no grant, an unknown kid and an unknown administration answer the same 404 body; the admin is refused; hard deletes; writes re-assert the role with `actorRole` inside `inTx`. Recording, editing and deleting doses is `Caregiver`; the kid's schema, dismissals and the bulk dismissal are `Guardian`.
- Logs carry ids and enumerated facts only: never a date, a lot number, a product name, a note or a date of birth (rule 5, spec 13.3).
- Comments state constraints only, at most 2 lines (rule 9). `gofmt -l` prints nothing; staticcheck clean; Prettier tabs; every new page works without JavaScript, passes axe at 390×844 and 1280×800 and keeps `Cache-Control: no-store`. The page is reachable when the `vaccination` module is switched off; only its tile and the home Due section hide.
- Embedded data lives under `api/internal/vaccines/data/` (never `testdata`, never a `*.db*` name). Each schedule file carries its own provenance; the code stays MIT (the schedule files are facts transcribed with the licences in Open value 2).
- Tests that pin counts change together with the code they pin: `wantTables` and `schema_version` (`"4"`), the kid JSON key set (9 → 10 keys, `vaccine_schema`) in `kids_test.go`, the home key set (`due`), `routeClasses`, `matrixParams`, the stub's `KID_FIELDS`, `quick-log.test.ts` (a Vaccination tile exists now).

## Reference data (what the research found; verify by test, do not re-research)

| | `us` | `uk` (England) | `mx` | `ch` (left out) |
|---|---|---|---|---|
| Edition to pin | CDC "Child and Adolescent Immunization Schedule by Age", 2025 edition, revised 07/02/2025 (what cdc.gov serves; the January 2026 revision is reported stayed by a court, see Open value 3) | UKHSA "Complete routine immunisation schedule from 1 September 2026", updated 2026-09-24 | "Lineamientos Generales del Programa de Vacunación Universal 2026" (CeNSIA, Secretaría de Salud), amended 2026-02-18; cross-check PRONAM 2nd ed. 2026-07-30 | "Schweizerischer Impfplan 2026", Stand Februar 2026, corrected 2026-04-16 (BAG with EKIF) |
| Licence | Public domain; conditions: credit the agency, no implied endorsement, do not change the substance, say it is free on the CDC site | Open Government Licence v3.0 ("Contains public sector information licensed under the Open Government Licence v3.0"); cite UKHSA, not NHS; logos and brand names excluded | No open licence found; gob.mx terms say content is public unless stated, require citing the URL and consultation date, and also say "solo para uso personal y no comercial"; LFDA art. 14 VIII excludes administrative texts and facts | "© Bundesamt für Gesundheit (BAG)"; admin.ch: written consent needed in advance for reproduction of any element; no open licence; URG Art. 5 coverage unclear |
| Window facts | ranges stated ("12–15 months", dash means "through"); ages in weeks, months, years | only "from" ages ("Eight weeks old"); cohort rows (6-in-1 4th dose and MMRV by birth date) | point ages plus min/max permitted ages in Anexo 1 (a 72-ppi scan, medium confidence); SRP at 6 years only for children born before July 2020 | "Alter 12 Monate" means from the 12-month birthday to the day before 13 months; basic vs supplementary vs risk-group categories |
| Traps | Hib 3 vs 4 doses and rotavirus 2 vs 3 depend on the product; influenza and COVID are shared-decision or seasonal, never overdue; the 2026 changes exist only in secondary sources | Green Book ch. 11 predates the Sept 2026 table (no MMRV at 3y4m); schedule lists PCV13 while an update note says PCV20 from 2026-01-01; NHS page footer is stale | PDFs are scans; "pentavalente" in the 2026 text means the RV5 rotavirus vaccine; varicella is not basic | MMR and varicella are at 9 and 12 months (doses before 9 months do not count); rotavirus, MenB, MenACWY and RSV-adjacent lines are supplementary or seasonal; 2027 edition expected in Q1 2027 |

Source ages are given in days, weeks, months or years, so a schedule file stores each age as `<n><unit>` (`d`, `w`, `m`, `y`) and the app converts by calendar arithmetic from the birth date.

## Review Focus

Failure modes the spec implies that no task's happy path exercises; each has a test in the named task.

1. A kid added at age four, or a kid whose schema is switched: no wall of overdue lines for a guardian who has not yet dismissed anything is acceptable (that is what dismiss-before is for), but a switch must keep every dose, keep dismissals made in the old schema out of the new view, and never show a dose twice (Tasks 2, 6, 7).
2. Count matching edges: more doses than the series has lines (the extras list under "Other"), a combination vaccine recorded once that fills the 4th diphtheria line and the 3rd hepatitis B line at once, deleting the 2nd of three doses shifts the later doses up one line, and a dose dated before an earlier-numbered one (Tasks 2, 7).
3. Window edges in the kid's zone: the day a window opens, the day before it ends, the day it ends, birth dates on the 29th–31st of a month, and a leap-day birth (Tasks 2, 8).
4. Cohort lines: a UK kid born the day before and the day of 1 July 2024 sees different series; a kid born outside every cohort of an antigen sees no line for it rather than an error (Tasks 1, 2, 4, 5).
5. Dismissals: undo; dismiss-before with no eligible line; a line dismissed and also given (given wins); a dismissal for a line that is not in the kid's current schema (404); caregiver refused (Tasks 2, 7).

---

## Open values (owner confirmation before execution)

Rule 1: these are proposals; each is silent or marked (call) in the spec.

| # | Value | Proposal | Basis |
|---|---|---|---|
| 1 | Schedules to ship | `us`, `uk` (England only) and `mx`. **Switzerland (`ch`) is left out** (owner, 2026-11-22: the BAG requires written consent in advance and none was asked for); its id is not offered, and the research note stays in the repository for a later plan that obtains consent | owner |
| 2 | Licences and permission | **us:** transcribe with CDC's four conditions in the file header and the README. **uk:** transcribe with the OGL line; cite UKHSA only; no brand names. **mx:** no open licence; owner decision (2026-11-22): ship a facts-only transcription (antigen, dose number, age) **with a citation** — the gob.mx URL of the Lineamientos and the consultation date, as gob.mx's terms ask — and say in the README that it is not endorsed by or a copy of the Secretaría de Salud | research notes; the owner reviews the files in the PR (spec 9) |
| 3 | US edition | Owner decision: the CDC **2025 edition (revised 2025-07-02)**, which is what cdc.gov serves now. The research found that a January 2026 revision was approved without ACIP and reported stayed by a court (HepA, HepB, MenACWY, rotavirus, influenza and COVID moved to shared decision-making or risk-based, HPV to one dose); CDC's own 2026 text could not be retrieved and this is not independently verified. The file records its edition so a later update can replace it | research note us-cdc |
| 4 | Window rule | Each line has `from` and an optional `until` (exclusive end). An age `N` covers `[N, N+1 unit)` when the source states it that way (Switzerland does); a source range such as "12–15 months" is `from 12m until 16m` ("through"). A line with no `until` (UK, Mexico) gets a default window of **one calendar month** after `from`. `upcoming` before `from`; `due` from `from` up to `until`; `overdue` from `until` on | spec 9 says "its recommended age window"; sources differ |
| 5 | Line kinds | `routine` (can be due or overdue), `may_not_be_needed` (product-dependent, spec 9: due inside its window, then available; never overdue), `supplementary` (Swiss "ergänzend", US shared decision-making, risk groups: listed under "Also available", never due or overdue), `recurring` (influenza, COVID-19 seasonal: listed with the doses given, never due or overdue) | spec 9 |
| 6 | Cohorts | A line may carry `born_from` and/or `born_to` (inclusive dates); it exists for a kid only if the birth date falls inside. The UK 6-in-1 booster at 18 months and the MMRV, and Mexico's SRP at 6 years, depend on it | research notes uk-ukhsa, mx-censia |
| 7 | Antigen granularity | One series per component antigen (diphtheria, tetanus, pertussis, polio, Hib, hepatitis B, pneumococcus, rotavirus, measles, mumps, rubella, varicella, hepatitis A, HPV, MenACWY, MenB, influenza, COVID-19, RSV, tuberculosis), so a combination product fills several series and a Tdap booster counts as the next diphtheria, tetanus and pertussis dose. Each line has a display `group` ("6-in-1", "MMR") so the screen shows vaccines, not 20 rows per visit | spec 9 "per antigen" |
| 8 | Catalogue | `antigens.csv` (id, English name) and `products.csv` (generic product id, name, antigen ids: for example "6-in-1 (DTaP-IPV-Hib-HepB)", "MMR", "Tdap"). No brand names (OGL excludes them). The record form offers a product shortcut that ticks its antigens, plus the antigens themselves | (call) |
| 9 | "Other" | A dose is listed under "Other" when its antigen is not in the kid's schema or when it exceeds the series length. The catalogue includes the usual travel antigens (hepatitis A, typhoid, yellow fever, rabies, Japanese encephalitis, cholera, TBE) and an `other` antigen with a required product name | spec 9 |
| 10 | Status precedence | `given` beats `dismissed` beats `overdue`/`due`/`upcoming` | (call) |
| 11 | Dismiss-before | Guardian-only: one action dismisses every line of the kid's current schema that is not given, not dismissed, of kind `routine` or `may_not_be_needed`, and whose `from` date is before the chosen date. It writes one dismissal row per line (reason `not_needed` or `given_unknown`, chosen once) so each can be undone alone | spec 9 |
| 12 | Kid's schema | `kids.vaccine_schema`, nullable, no default: a new kid has none and the Vaccines page asks the guardian to choose; the home shows no Due list until one is chosen | spec 6.3 "nullable" |
| 13 | Where it lives | `/kids/{kid}/vaccines` (not a nav slot: the bar stays Home, Timeline, Growth, Account), reached from the home Due section, the kid links and the quick-log tile; doses are not in the timeline (they are dates, not instants) | spec 7.3 order, plan 4 precedent |
| 14 | Home "Due" | The home read model gains `due`: one entry per `from` age group that has a `due` or `overdue` line (`kind:"vaccine"`, `from`, `status` = `overdue` if any line is, `count`; the `kind` discriminator lets plan 6 add medication entries to the same list without a breaking change), ordered by `from`; the UI shows "4-month vaccines · overdue". Empty when the module is off or the kid has no schema. Medication joins it in plan 6 | spec 7.3 wireframe |
| 15 | Limits | At most 500 administrations per kid (`MaxVaccineAdministrationsPerKid`); lot ≤ 40 and product name ≤ 80 characters; a note ≤ 500 (existing). A dose date may not be after the kid-local today nor before the birth date | (call) |
| 16 | Review aid | Each schedule file has a generated review document `docs/vaccine-schedules/<id>.md` (a table per visit) and a test that fails when it is out of date, so you review the PR's tables, not CSV rows | spec 9 "reviewed by the owner in the PR" |
| 17 | Easy to update (owner) | A schedule is one CSV plus a provenance header; an update edits that file, changes `edition` and `retrieved`, regenerates the review document (`UPDATE_DOCS=1 go test ./internal/vaccines/`) and opens a PR whose diff shows exactly which lines changed. The loader names the file, line and column of every error. Line ids are stable (`<antigen>-<series>`, plus a cohort tag only where two cohorts share a series number): an update that renames or removes a line only orphans stored dismissals, which are ignored, never an error; recorded doses never depend on a schedule. A runbook (`docs/runbooks/updating-vaccine-schedules.md`) lists the steps, each source's URL, when it is reissued and the traps from the research notes. The Vaccines page shows the edition and retrieval date so staleness is visible. Rejected: loading schedules from a directory at run time (a new env var, a volume and an edge change for something CI already ships) | owner |

---

## Task 1: Schedule format, loader and catalogue

**Files:** Create `api/internal/vaccines/{schema.go,age.go,catalogue.go,load.go}` (+ `_test.go` each) and `api/internal/vaccines/data/{antigens,products}.csv`; modify `api/internal/constants/constants.go`.

**Interfaces — Produces:**
```go
type Unit byte // 'd','w','m','y'
type Age struct{ N int; Unit Unit }
func ParseAge(s string) (Age, error)            // "8w", "12m", "4y", "0d"; refuses empty, 0 units other than d, negatives, extra text
type Kind string                                // KindRoutine, KindMayNotBeNeeded, KindSupplementary, KindRecurring
type Line struct{ ID, Group, Antigen string; Series int; From Age; Until *Age; Kind Kind; BornFrom, BornTo string; Note string }
type Schema struct{ ID, Name, Authority, Edition, Retrieved, OriginURL, Licence, Attribution string; Lines []Line }
func Load() error                               // idempotent; a failure is remembered and returned on every call; main calls it before listening
func Schemas() []Schema                         // shipped order: us, uk, mx
func GetSchema(id string) (Schema, bool)
type Antigen struct{ ID, Name string }
type Product struct{ ID, Name string; Antigens []string }
func Antigens() []Antigen; func Products() []Product
func ValidAntigens(ids []string) error          // non-empty, known, no duplicates
```
File format: `data/<id>.csv`, `# key: value` provenance lines (`schema`, `name`, `authority`, `edition`, `retrieved`, `origin_url`, `licence`, `attribution`), then the header `line,group,antigen,series,from,until,kind,born_from,born_to,note`.

- [ ] **Step 1: Failing tests** on small in-memory schemas through an unexported `parse(name string, io.Reader) (Schema, error)` (every error names the file, the CSV line and the column, and never a free-form value; a test pins one message per refusal): `ParseAge` table (`8w`, `12m`, `3y`, `0d` ok; `""`, `0m`, `-1m`, `8`, `8x`, `1.5m`, `12 m` refused); a valid file parses; every refusal is its own case — missing provenance key, duplicate line id, unknown kind, unknown antigen, series not starting at 1 or with a gap **within one cohort**, `until` not after `from`, `born_from` after `born_to`, a malformed date, a non-ASCII digit in an age; two lines of one antigen with the same series number and overlapping cohorts refused, with disjoint cohorts accepted; `Load()` idempotent and a panicking/failing injected loader is remembered (the growth package's `Load` is the pattern); catalogue: every product's antigens exist, no brand names list (a test pins the product ids).
- [ ] **Step 2:** `cd api && go test ./internal/vaccines/ -race` → FAIL. **Step 3: Implement.** Constants: `MaxVaccineAdministrationsPerKid = 500`, `VaccineLotMaxLen = 40`, `VaccineProductMaxLen = 80`, `VaccineDefaultWindowMonths = 1`. **Step 4:** PASS (a stub schema file is only in test code). **Step 5: Commit** `feat(api): vaccination schedule format, loader and catalogue`.

## Task 2: Age arithmetic and the evaluation engine

**Files:** Create `api/internal/vaccines/evaluate.go` (+ test); modify `api/internal/kidtime/kidtime.go` (+ test).

**Interfaces — Consumes:** Task 1 types. **Produces:**
```go
// kidtime
func AddAge(date string, n int, unit byte, loc *time.Location) (string, error) // calendar arithmetic as BandStart does (Go AddDate; a 31st or 29 Feb birth normalises forward); ErrBadDate without the input
// vaccines
type Status string // StatusGiven, StatusDue, StatusUpcoming, StatusOverdue, StatusDismissed
type Dose struct{ ID, Administration, Antigen, GivenOn string }
type LineState struct{ Line Line; Status Status; FromDate, UntilDate, GivenOn string; DoseID, Administration, DismissReason string }
type Evaluation struct{ Lines []LineState; Also []LineState; Recurring []RecurringState; Other []Dose }   // Also = supplementary lines; Recurring = per antigen the doses given
func Evaluate(s Schema, doses []Dose, dismissals map[string]string, birthDate, today string, loc *time.Location) (Evaluation, error)
func (e Evaluation) Due() []DueGroup                // DueGroup{From Age; FromDate string; Status Status; Count int}, ordered by FromDate
```
`Evaluate` keeps only lines whose cohort admits the birth date; sorts each antigen's doses by `GivenOn` then id and fills series 1..n in order; excess doses and doses of antigens absent from the schema go to `Other`; statuses by Open values 4, 5, 10; `until` empty means `from` + `VaccineDefaultWindowMonths`; `Also` and `Recurring` never carry due/overdue.

- [ ] **Step 1: Failing tests:** `AddAge` (8w, 12m, 4y, 0d; a 31 Jan birth + 1m; 29 Feb 2024 + 1y; unparseable date → `ErrBadDate` without the date in the message); `TestEvaluateStatuses` (a US-shaped schema: before `from` upcoming, on `from` due, the day before `until` due, on `until` overdue, given beats overdue, dismissed beats due, given beats dismissed, `may_not_be_needed` due inside its window, then available, supplementary in `Also` without a status of due/overdue, recurring never due); `TestCountMatching` (a combination dose fills two antigens' lines; 6 doses against a 5-line series → one in `Other`; deleting the middle dose shifts later doses up; an earlier-dated dose recorded later takes the earlier line; a dose of an antigen not in the schema → `Other`); `TestOrphanDismissals` (a dismissal for a line the schedule no longer has is ignored, never an error); `TestCohorts` (born 2024-06-30 vs 2024-07-01 see different series for the same antigen; a kid in no cohort of an antigen has no lines for it); `TestWindowsInTheKidsZone` (window edges in Pacific/Auckland vs UTC); `TestDueGroups` (one group per `from` age, `overdue` if any line overdue, none when all are given or dismissed, counts).
- [ ] **Step 2:** FAIL. **Step 3: Implement.** **Step 4:** `go test ./internal/vaccines/ ./internal/kidtime/ -race` PASS. **Step 5: Commit** `feat(api): vaccination status engine and age arithmetic`.

## Task 3: United States schedule data

**Files:** Create `api/internal/vaccines/data/us.csv`, `docs/vaccine-schedules/us.md`, `api/internal/vaccines/data_test.go`, `api/internal/vaccines/review_test.go`; modify `api/internal/vaccines/load.go` (embed), `README.md` (§ Third-party data: the vaccination schedules and their licences, one row per shipped schedule).

- [ ] **Step 1: Failing tests:** `TestShippedSchedulesLoad` (every embedded file parses; provenance keys non-empty; `origin_url` starts `https://`; `retrieved` is a date; the attribution text contains each licence's mandatory phrase — for `us`: "Source: CDC" and the no-endorsement sentence); `TestUSKnownLines` (from the research note, 12 pinned lines: HepB 1 at `0d`; RV 1 at `2m`, RV 3 is `may_not_be_needed`; DTaP-component series 1–5 at 2m, 4m, 6m, 15m, 4y with the 4th `until 19m`; Hib 4 is `may_not_be_needed`; PCV 4 at 12m `until 16m`; MMR 1 at 12m and 2 at 4y; VAR; HepA 1 at 12m; MenACWY 1 at 11y; HPV 1 at 11y; influenza is `recurring`; COVID-19 is `supplementary`); `TestReviewDocsAreCurrent` (renders each schema to Markdown — a table per `from` group with antigens, series, window, kind and the source's own wording in `note` — and compares with `docs/vaccine-schedules/<id>.md`; `UPDATE_DOCS=1` rewrites them).
- [ ] **Step 2:** FAIL. **Step 3: Transcribe** `us.csv` from the CDC PDF and the research note (read both; every row's `note` quotes the source wording where it is not obvious; combination products appear as one line per component antigen sharing a `group`), write the generated review document. **Step 4:** PASS. **Step 5: Commit** `feat(api): United States vaccination schedule (CDC 2025)`.

## Task 4: United Kingdom schedule data

**Files:** Create `api/internal/vaccines/data/uk.csv`, `docs/vaccine-schedules/uk.md`; modify `data_test.go`, `README.md`.

- [ ] **Step 1: Failing tests:** `TestUKKnownLines` (6-in-1 components at `8w`, `12w`, `16w`; the 4th 6-in-1 at `18m` with `born_from 2024-07-01`; MenB at `8w`, `12w`, `1y`... as the note states; MMRV components at `12m` and `18m` with `born_from 2025-01-01` and the older cohort's MMR at `12m` and `3y4m` expressed in months as `40m` with `born_to 2024-12-31`; HPV at `12y until 14y`; influenza `recurring`); the attribution contains "Contains public sector information licensed under the Open Government Licence v3.0" and no line mentions a brand name (a list of the brand names in the research note is the denylist).
- [ ] **Step 2–5:** as Task 3, commit `feat(api): United Kingdom vaccination schedule (UKHSA, Sept 2026)`.

## Task 5: Mexico schedule data

**Files:** Create `api/internal/vaccines/data/mx.csv`, `docs/vaccine-schedules/mx.md`; modify `data_test.go`, `README.md`.

- [ ] **Step 1: Failing tests:** `TestMXKnownLines` (BCG and hepatitis B at `0d`; hexavalent components at `2m`, `4m`, `6m`, `18m`; DPT booster at `4y` counting as diphtheria, tetanus and pertussis series 5; rotavirus at `2m`, `4m`; pneumococcus `2m`, `4m`, `12m`; SRP at `12m`, `18m`, and the 6-year dose with `born_to 2020-06-30`; hepatitis A at `18m`; influenza `recurring`); the attribution contains the gob.mx URL, "consulta" date and the not-endorsed sentence; no line has an `until` earlier than its `from`. Rows whose ages came from the low-confidence Anexo 1 scan carry `note: "(verify)"` and the review document lists them first.
- [ ] **Step 2–5:** as Task 3, commit `feat(api): Mexico vaccination schedule (Lineamientos PVU 2026)`.

## Task 6: Migration 0004 and the store

**Files:** Create `api/internal/store/migrations/0004_vaccination.sql`, `api/internal/store/vaccines.go` (+ test); modify `api/internal/store/kids.go`, `api/internal/model/model.go`, `api/internal/store/store_test.go`.

**Interfaces — Produces:**
```go
type model.Kid { …; VaccineSchema *string }                     // nullable
type model.VaccineDose struct{ ID, KidID, Administration, Antigen, GivenOn string; Lot, Product, Note *string; CreatedBy, UpdatedBy string; CreatedAt, UpdatedAt time.Time }
func (s *Store) CreateAdministration(ctx, kidID, actorID string, a model.Administration, now time.Time) (model.Administration, error) // one row per antigen, one transaction; ErrConflict at 500 administrations
func (s *Store) Administration(ctx, kidID, id string) (model.Administration, error)
func (s *Store) ListAdministrations(ctx, kidID string) ([]model.Administration, error)               // newest first (given_on DESC, id DESC)
func (s *Store) UpdateAdministration(ctx, kidID, actorID string, a model.Administration, now time.Time) error // replaces the antigen set
func (s *Store) DeleteAdministration(ctx, kidID, actorID, id string) error
func (s *Store) Doses(ctx, kidID string) ([]model.VaccineDose, error)
func (s *Store) SetDismissal(ctx, kidID, actorID, schema, line, reason string, now time.Time) error  // upsert
func (s *Store) ClearDismissal(ctx, kidID, actorID, schema, line string) error
func (s *Store) Dismissals(ctx, kidID, schema string) (map[string]string, error)
func (s *Store) DismissLines(ctx, kidID, actorID, schema string, lines []string, reason string, now time.Time) error // one transaction
```
`model.Administration` = the shared fields plus `Antigens []string`. Migration: `ALTER TABLE kids ADD COLUMN vaccine_schema TEXT` (no CHECK: the handler validates against the loaded ids); `vaccine_doses` (`id`, `kid_id` FK cascade, `administration`, `given_on`, `antigen`, `lot`, `product`, `note`, common columns, UNIQUE (`administration`, `antigen`), index (`kid_id`, `given_on`)); `vaccine_dismissals` (`kid_id` FK cascade, `schema`, `line`, `reason` CHECK `not_needed|given_unknown`, `created_by`, `created_at`, PRIMARY KEY (`kid_id`, `schema`, `line`)).

- [ ] **Step 1: Failing tests:** migration on a database at 0003 with kids and every child table (the 0003 upgrade test is the pattern): all rows survive, `vaccine_schema` reads NULL, version `"4"`; CRUD of an administration (3 antigens → 3 rows with one administration id; update replaces the antigen set; delete removes all rows; a lone-antigen delete is not possible through the API); `TestAdministrationBelongsToItsKid` (another kid's id → `ErrNotFound` on read, update, delete), `TestVaccineWriteByDisabledActor`, `TestAdministrationCap` (the 501st is refused with `ErrConflict`; deleting one frees a slot), dismissals upsert/clear/list per schema, `DismissLines` is atomic (a failing line rolls all back), cascade on kid delete empties both tables.
- [ ] **Step 2:** FAIL. **Step 3: Implement** by the pattern of `growth.go` and `entries.go`. **Step 4:** `go test ./internal/store/ -race` PASS. **Step 5: Commit** `feat(api): vaccination migration and store`.

## Task 7: Vaccination API

**Files:** Create `api/internal/httpapi/vaccines.go` (+ test); modify `routes.go`, `matrix_test.go`, `kids.go` (+ `kids_test.go`), `cmd/happybabytracker-api/main.go` (call `vaccines.Load()` before listening).

**Interfaces — Produces:** routes (class in brackets):
`GET /v1/vaccine-schemas` [Parent] → `{"schemas":[{id,name,authority,edition,retrieved,origin_url,licence,attribution}]}`; `GET /v1/vaccine-catalogue` [Parent] → `{"antigens":[{id,name}],"products":[{id,name,antigens}]}`; `GET /v1/kids/{kid}/vaccines` [Caregiver]; `GET|POST /v1/kids/{kid}/vaccinations` [Caregiver] (list, record: 201); `GET|PATCH|DELETE /v1/kids/{kid}/vaccinations/{administration}` [Caregiver]; `PUT|DELETE /v1/kids/{kid}/vaccine-dismissals/{line}` [Guardian] (`{reason}`); `POST /v1/kids/{kid}/vaccine-dismissals` [Guardian] (`{before:"YYYY-MM-DD", reason}`); the kid view and `PATCH /v1/kids/{kid}` gain `vaccine_schema` (`null` or a shipped id; unknown → 422 `vaccine_schema`).
`GET …/vaccines` → `{"schema":{…info}|null,"today":"YYYY-MM-DD","groups":[{"from":{"n","unit"},"from_date","status","lines":[{"id","group","antigen","series","kind","status","from_date","until_date","given_on"|null,"administration"|null,"dismissal":{"reason"}|null,"note"}]}],"also":[…lines…],"recurring":[{"antigen","doses":[{"administration","given_on"}]}],"other":[{"administration","antigen","given_on","product"|null}]}`; with no schema: `schema:null` and empty arrays. Administration view: `{id, given_on, antigens:[…], lot, product, note}`; input `{given_on, antigens, lot, product, note}` (PATCH merges: absent keeps, `null` clears lot/product/note, `antigens` replaces the set); validation order `given_on`, `antigens`, `lot`, `product`, `note`; the date must parse, not be after the kid-local today nor before the birth date; an antigen `other` requires `product`; the 501st administration → 409 `limit`. Chronological age only.

- [ ] **Step 1: Failing tests:** `TestVaccinesShape` (key sets via `sameKeys`; a fixed kid and a small shipped-shape schema, so tests do not depend on the real files); `TestRoutesRoles` (guardian and caregiver record/edit/delete; caregiver refused on dismissals and the kid's schema → 403; no-grant and unknown kid and unknown administration 404 bodies byte-identical; admin 403); `TestValidation` (future date, before birth, malformed date, empty antigens, unknown or duplicate antigen, `other` without product, lot/product/note over their limits, unknown field, `null` on required fields); `TestSwitchSchema` (doses kept, dismissals of the old schema not shown, a dismissal for a line of another schema → 404); `TestDismissBefore` (only unsettled routine/`may_not_be_needed` lines before the date; none eligible → 200 with `dismissed: 0`; undo of one line); `TestNoSchema` (empty arrays, doses still recorded and listed under `other`); `TestNoValuesInLogs` with `wantCleanLog` (a distinctive lot, product, note, date); matrix: every new route classed and `matrixParams` entries for `{administration}` and `{line}`.
- [ ] **Step 2:** FAIL. **Step 3: Implement** by the pattern of `growth.go` (`optionalString`, `mergeOnto`, `entryOK`). **Step 4:** `go test ./... -race` PASS. **Step 5: Commit** `feat(api): vaccination schedule, doses and dismissals`.

## Task 8: Home "Due"

**Files:** Modify `api/internal/httpapi/home.go`, `home_test.go`.

**Interfaces — Produces:** `homeView.Due []dueView` (`{kind:"vaccine", from:{n,unit}, status, count}`), empty (never null) when the `vaccination` module is off, the kid has no schema or nothing is due.

- [ ] **Step 1: Failing tests:** `sameKeys` gains `due`; `TestHomeDue` (a kid with a US-shaped schema at 4 months: one group, `overdue` when a line is past its `until`, `due` otherwise; module switched off → `[]`; no schema → `[]`; all lines given or dismissed → `[]`; the window edges in a non-UTC zone from Task 2 are exercised through the API once).
- [ ] **Step 2–5:** `go test ./internal/httpapi/ -race`; commit `feat(api): the home read model lists what is due`.

## Task 9: UI plumbing — client, types, forms, stub

**Files:** Modify `ui/src/lib/server/api.ts` (+ test), `ui/src/lib/types.ts`, `ui/src/lib/constants.ts`, `ui/src/lib/server/constants.ts` if needed, `ui/src/lib/server/kid-form.ts`, `ui/e2e/stub-api.ts`; create `ui/src/lib/server/vaccine-form.ts` (+ test), `ui/src/lib/vaccines.ts` (+ test).

**Interfaces — Produces:** types `VaccineSchemaInfo`, `VaccineCatalogue`, `VaccineStatus` (`given|due|upcoming|overdue|dismissed`), `VaccineLine`, `VaccineGroup`, `Vaccines`, `Administration`, `Home.due`; `Kid.vaccineSchema: string | null`; client `api.vaccines(kidId)`, `.vaccineSchemas()`, `.vaccineCatalogue()`, `.listVaccinations/.getVaccination/.createVaccination/.updateVaccination/.deleteVaccination`, `.setDismissal/.clearDismissal/.dismissBefore` (ids through `withId`/`withKidEntry`; a malformed id answers not-found without a request); `readVaccineForm(form, mode)` (date input kid-local default today, antigen checkboxes, `lot`, `product`, `note`; edit mode sends a field only when changed, hidden originals fail closed, R14-style); formatters `ageLabel({n,unit}): string` (`"8 weeks"`, `"4 months"`, `"11 years"`), `groupTitle(group)` (`"4-month vaccines"`), `statusLabel`; the stub implements every route with the API's statuses, shapes and 404 rules, with a **clearly marked stub** schedule used for no numeric assertion.

- [ ] **Step 1: Failing tests (RED):** formatters and `readVaccineForm` (no antigen → field `antigens`; hostile date text; value-free messages; edit omits untouched fields and sends `null` for a cleared lot; missing originals fail closed), `api.test.ts` (list, create 422 with `field`, update, delete, dismissals, a bad id → not-found without fetch, malformed bodies → `error`), `e2e/stub-vaccines.spec.ts` self-tests.
- [ ] **Step 2–5:** `cd ui && pnpm test` → FAIL; implement; `pnpm check && pnpm lint && pnpm test && pnpm test:e2e --trace off`; commit `feat(ui): vaccination client, forms, formatters and stub routes`.

## Task 10: UI — the Vaccines page and the dose forms

**Files:** Create `ui/src/routes/(app)/kids/[kid]/vaccines/{+page.server.ts,+page.svelte}`, `…/vaccines/new/{+page.server.ts,+page.svelte}`, `…/vaccines/[administration]/{+page.server.ts,+page.svelte}`, `ui/src/lib/components/vaccines/{VaccineGroup.svelte,DoseFields.svelte}`; modify `ui/src/lib/paths.ts`.

**Interfaces:** the page shows "Schedule: {authority}, {edition}" with the schedule's licence/attribution text, the lines grouped by age (`groupTitle`, each line: antigen, series, status badge by **text** not colour alone, the date given or the window dates, a Record shortcut for due/overdue lines that opens the form with its antigens ticked), "Also available" (supplementary), recurring doses, "Other", and for guardians the per-line Dismiss/Undo forms and the dismiss-before form (date + reason); with no schema: a sentence and, for guardians, a link to settings. Create/edit follow the plan-3 and plan-4 patterns (`failAction`, `fail(422, {form, values, field, message})`, no-JS, delete behind `<details>`, 303 to the page, the same 404 page for no-grant/unknown/another kid's administration, 409 `limit` → "This child has the maximum number of recorded vaccinations"); the product shortcut is a native `<select>` that ticks antigens with a tiny enhancement and works without JavaScript through a "Use these antigens" submit-less fallback (the antigens are plain checkboxes).

- [ ] **Step 1: Failing tests (Playwright, both projects, axe):** `vaccines.spec.ts` — a kid with the stub schedule shows groups in age order with the right statuses; no schema shows the prompt (guardian link vs caregiver text); record a 6-in-1 through the product shortcut and see the lines given; record with a lot and product and read them back on the edit page; edit only the note sends `{note}`; clear the lot sends `null`; delete with the confirm; a future date, a date before birth and no antigen show field messages with values kept; dismiss a line and undo it; dismiss-before; caregiver can record and cannot see dismiss controls; no-grant parent and another kid's administration id → the same 404 page; "Other" lists an extra dose; no horizontal overflow at 390 px with a 200-character unbroken note; no-JS record.
- [ ] **Step 2–5:** RED; implement; full UI pipeline; commit `feat(ui): the vaccines page and dose forms`.

## Task 11: UI — schedule setting, home Due, quick log

**Files:** Modify `ui/src/lib/components/KidForm.svelte`, `ui/src/lib/server/kid-form.ts`, `ui/src/routes/(app)/kids/[kid]/settings/+page.server.ts`, `…/kids/new/+page.server.ts`, `…/kids/[kid]/+page.server.ts` and `+page.svelte`, `ui/src/lib/components/{KidLinks.svelte,QuickLog.svelte}`, `ui/src/lib/quick-log.ts` (+ test), `ui/e2e/{home,kids}.spec.ts`.

**Interfaces:** kid settings and `/kids/new` get a native `<select>` "Vaccination schedule" listing "None yet" plus exactly the ids from `api.vaccineSchemas()` (name, authority, edition as the option hint); the home shows a "Due" section (after Running now, before Today) from `home.due` — `groupTitle` and `overdue` as **text**, each linking to the Vaccines page — only when non-empty; a Vaccination tile in quick log links to `/kids/{kid}/vaccines` when the module is enabled (`PAGE_ROUTE` in `quick-log.ts`); `KidLinks` gains Vaccines.

- [ ] **Step 1: Failing tests (Playwright, both projects, axe):** select lists exactly the shipped ids plus "None yet", saving one changes the Vaccines page's caption, a new kid defaults to none; home shows Due with an overdue group and hides it with the module off; the Vaccination tile appears in the 24+ band after growth, hidden when the module is off; `quick-log.test.ts` updated (vaccination now has a tile); no-JS settings save.
- [ ] **Step 2–5:** RED; implement; full UI pipeline; commit `feat(ui): vaccination schedule setting, due list and quick-log tile`.

## Task 12: Smoke, docs and HANDOFF

**Files:** Modify `scripts/smoke.sh`, `README.md`, `docs/superpowers/HANDOFF.md`, `api/Dockerfile` only if a new notice file is needed.

- [ ] **Step 1: Extend `scripts/smoke.sh`** (failing first against the images from `main`): step `4i` — as the guardian choose the US schedule for the smoke kid in settings, read the Vaccines page and assert the caption and the attribution text, the lines in age order and a due/overdue line for the kid's age; record a 6-in-1 (or the US equivalent) through the real form and assert the matching lines turn given and the same dose is not listed twice; edit its lot and delete it through the real API (scrape the hidden originals as step 4h does); dismiss a line and undo it; switch to the UK schedule and assert the doses are still there and the US dismissal is gone; a caregiver records one; the no-grant parent and the admin get 404/redirect on `/kids/{id}/vaccines`, `/vaccines/new`, `/vaccines/{administration}`; assert the settings select lists exactly the shipped schedule ids; the home shows Due for the kid; extend step 9's log scan with the distinctive lot, product name and note; keep step 10's ceilings (the API must stay under 60 MiB with all schedules loaded: report the measured numbers).
- [ ] **Step 2:** build both images and run `scripts/smoke.sh hbt-api:dev hbt-ui:dev` → every line `PASS:`; `shellcheck scripts/smoke.sh`. **Step 3: Docs:** README Vaccination section (what is shown, the three schedules and editions, "never judges validity", "overdue is a soft flag", the licence position per schedule); `docs/vaccine-schedules/` index; the update runbook `docs/runbooks/updating-vaccine-schedules.md` (Open value 17); HANDOFF: plans 1–5 done, plan 6 (sickness, medication, reminders) starts from the `vaccination` Due group in the home read model, the owner decisions of plan 5, the unverified items (Mexico reuse, the US 2026 revision, Anexo 1 ages) and that Switzerland is left out pending the BAG's consent. **Step 4:** the whole pipeline in CLAUDE.md (API, UI, images, smoke) green. **Step 5: Commit** `test: smoke covers vaccination; docs: plan 5 done`.

---

## Done when

- API and UI pipelines and `scripts/smoke.sh` pass on the branch head and CI is green.
- The access matrix covers every new route and fails if one is missing.
- A guardian chooses a schedule, a caregiver records a combination vaccine, the lines it covers turn given, a dose due or overdue by the kid's age shows on the home, switching the schedule keeps every dose, and the owner has reviewed `docs/vaccine-schedules/*.md` in the PR.
- The final report states which image content changed (`api/` sources and embedded data, `ui/` sources; no Dockerfile or dependency change expected), that the edge still needs its `bump-happybabytracker` PR merged and a deploy, the licence position per schedule, and which schedules were left out.
