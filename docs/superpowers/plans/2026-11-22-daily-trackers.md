# Daily Trackers Implementation Plan (plan 3)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A parent logs bottles, nursing, pumping, sleep, diapers and solid meals for a kid, reads them as a timeline and as daily summaries, turns each tracker off and on per kid, and uses an age-adaptive home screen from an installable PWA.

**Architecture:** One migration adds six tracker tables and `kid_modules`. Each tracker has explicit store functions and a handler file behind the existing `Caregiver` access class; shared plumbing (volume conversion, day boundaries in the kid's time zone, range and cursor parsing, age bands) is small, pure and tested first. The UI gets one server-side tracker client, log and edit pages under `/kids/[kid]/log/...`, a timeline page, a module settings page, a kid home that replaces the placeholder, and a static manifest. No service worker (plan 6 adds one for Web Push).

**Tech Stack:** Go 1.27, `modernc.org/sqlite`, stdlib `net/http` mux (as plan 1); SvelteKit 3 / Svelte 5 runes, Tailwind 4, shadcn-svelte components already present (no new dependency in either image).

**Spec:** `docs/superpowers/specs/2026-11-20-happybabytracker-design.md` sections 2 (G6–G9, G13), 5.2, 6.1–6.4, 7, 13.3, 14, 16. Plan 1: `docs/superpowers/plans/2026-11-20-foundation.md`.

## Global Constraints

- Every limit, cap and duration lives in `api/internal/constants/constants.go` (API) or `ui/src/lib/server/constants.ts` (UI), with client-safe mirrors in `ui/src/lib/constants.ts`. No literal elsewhere (CLAUDE.md rule 7).
- Instants are unix seconds UTC in INTEGER columns; RFC3339 UTC strings on the wire; calendar dates are `YYYY-MM-DD` (rule 8; spec 6.1).
- Every tracker table: `kid_id TEXT NOT NULL REFERENCES kids (id) ON DELETE CASCADE`, `created_by`/`updated_by` referencing `parents (id)`, `created_at`, `updated_at`, optional `note`; `STRICT`; index on `(kid_id, <time column>)` (spec 6.1, 6.4).
- Every store function that touches kid data takes the kid id and filters by it; an entry id that belongs to another kid answers `ErrNotFound`. Writes re-assert the caller's role with `actorRole` inside `inTx` (as `access.go` does).
- A caller with no grant, an unknown kid and an entry of another kid all answer the same 404 body (`respondNoKid` / `not_found`). The admin is refused (spec 5.1).
- Hard deletes; no edit history; a running timer is a row with a null end; at most one running row per kid per table (spec 6.3).
- Measured volumes are stored three ways: millilitres, hundredths of a US fluid ounce, and the unit typed; US fl oz = 29.5735 ml; conversion happens in one Go package, only at save and at edit (spec 6.2).
- Logs carry ids and enumerated facts only: never a note, meal text, volume or name (rule 5, spec 13.3). URLs carry ids and `from`/`to`/`types`/`cursor`/`limit` only.
- Comments state constraints only, at most 2 lines (rule 9). `gofmt -l` prints nothing; staticcheck clean; Prettier tabs.
- Every new UI page works without JavaScript (plain form POSTs), is checked at phone (390x844) and desktop (1280x800) width, and passes axe. Pages that show kid data keep `Cache-Control: no-store`.
- Tests that pin counts change together with the code they pin (`wantTables`, `schema_version`, `routeClasses`, kid JSON key sets, the shell link list, the Welcome heading specs).
- Nothing in this plan reads or requires `VAPID_*` or `NTFY_*`; no change in `homelab-edge` except the digest bump after merge.

## Review Focus

Failure modes the spec implies that no single task's happy path exercises; each has a test in the named task.

1. An entry id from another kid, or a kid the caller has no grant on, in the path: identical 404, nothing read or changed (Tasks 4–6, matrix).
2. A day that is 23 or 25 hours long, a kid whose zone differs from the server's UTC and from the device's: "today", daily summaries and a sleep that crosses midnight split on the kid's local boundaries (Tasks 2, 7).
3. Two devices start the same timer at once, end a timer twice, or switch the nursing side on a finished session: exactly one running row, `already_running` / `not_running` 409s, no negative durations (Task 6).
4. Ounces: 4.00 oz typed, then only the note edited, keeps the typed value; editing the volume recomputes the other column; 0, negative and huge amounts are refused (Tasks 1, 4).
5. Timeline paging with equal timestamps across trackers: no entry is repeated or skipped across pages; a bad cursor, `from` after `to`, an unknown type and a huge limit are 400s, never a 500 or an unbounded query (Task 7).

---

## Open values (owner confirmation before execution)

Rule 1 (never assume an owner value): these are proposals. Each is marked **(call)** in the spec or is silent.

| # | Value | Proposal | Basis |
|---|---|---|---|
| 1 | Module ids | Eleven, one per tracker: `bottles`, `nursing`, `pumping`, `diapers`, `sleep`, `feeding`, `milestones`, `growth`, `vaccination`, `sickness`, `medication`. The API knows all eleven now; the UI shows entry points only for built ones | G7 says every tracker is switchable; the 7.2 table groups three rows |
| 2 | Age bands | 0–11, 12–23, 24+ whole calendar months from the birth date in the kid's zone | 7.2 |
| 3 | Band-crossing rule | Persisted on the first home or modules read inside the new band: a module whose default is off there, with no override, that has an entry in the 60 days before the band's start date (or since) or a running timer, gets an explicit `enabled = 1` row | 7.2 (call: 30 days); **owner chose 60 days**; no worker exists before plan 6 |
| 4 | Timeline | Newest first; default `types` = enabled trackers; an explicit `types=` returns those trackers even when switched off | 6.4 silent on order and on switched-off trackers |
| 5 | Entry-time validation | `at`/`started_at` at most 5 minutes after now and not before the kid's birth date (start of day, kid zone); a session lasts at most 24 hours; `ended_at` not before `started_at` | spec silent; typo guard |
| 6 | Other limits | Bottle volume 1–1000 ml; pumping 1–2000 ml; note 500 characters; meal text 200 and reaction 200; list/timeline page default 50, max 100; summary range at most 93 days | spec silent |
| 7 | Volume input | Integer millilitres, or hundredths of a US fl oz; half-up rounding of the derived column | 6.2 (fixed-point) |
| 8 | Quick-log | Sleep is one tap (POST starts it now). Nursing, bottle, diaper, pumping and solids open a short form with the time prefilled to now. Ending any timer uses server time | 7.3 mock-up shows buttons only |
| 9 | Home route | `/kids/[kid]` becomes the home screen; `/` redirects to the current kid (cookie, else first); with no kid `/` keeps the Welcome and add-a-child state. Bottom bar: Home, Timeline, Account. No "Due" section until plans 5–6 | 7.3 shows `Home Timeline Growth More`; Growth and More arrive later |
| 10 | Editing a timer | A finished session cannot be re-opened (PATCH may not null `ended_at`); a running one may be ended through PATCH or the end route | spec silent |
| 11 | Date-time input | `datetime-local` read in the kid's zone, not the device's; a time in a DST gap moves forward by the gap, a repeated time takes the earlier instant | 6.1 (kid zone) |
| 12 | Summaries | Per kid-local day: counts and sums; both volume columns summed independently; nursing = left + right seconds of sessions that start that day (a running one counts what is stored); sleep = seconds clipped to the day, a running one clipped at now | 6.4 |
| 13 | PWA | `static/manifest.webmanifest` (name "Happy Baby Tracker", short name "HappyBaby", `start_url "/"`, `display standalone`), three PNG icons (192, 512, 512 maskable) rendered from one plain placeholder SVG, `theme-color`, apple touch icon; no service worker. Installability without a service worker is believed fine in current Chrome and is checked by manifest tests only, not on a device | 14 (installable) ; 4.1 |
| 14 | Enumerations | bottle contents are a **non-empty set** of `breast_milk`, `formula`, `cow_milk`, `other` (**owner: mixed bottles allowed, cow's milk added**; one total volume, no per-component split); diaper `wet`/`dirty`/`both`/`dry`; nursing side `left`/`right`; pumping side `left`/`right`/`both` | 6.3 (pumping side values are a call) |

---

## Task 1: Volume conversion

**Files:** Create `api/internal/units/volume.go`, `api/internal/units/volume_test.go`; modify `api/internal/constants/constants.go`.

**Interfaces — Produces:**
```go
package units
type Unit string // "ml" | "floz"
const (UnitML Unit = "ml"; UnitFlOz Unit = "floz")
type Volume struct{ ML, FlOzHundredths int64; Typed Unit }
var ErrUnit, ErrRange error
// amount is whole millilitres for UnitML, hundredths of a US fl oz for UnitFlOz.
func NewVolume(unit Unit, amount int64) (Volume, error) // ErrUnit for any other unit; ErrRange if amount < 1 or the derived column rounds to 0
```
Constants: `FlOzMicroMLNumerator = 295735` and `FlOzMicroMLDenominator = 10000` (29.5735 ml per fl oz); `MaxVolumeAmount = 1 << 24`.

- [ ] **Step 1: Failing tests** in `volume_test.go`: `TestNewVolumeML` (120 ml → `FlOzHundredths 406`), `TestNewVolumeFlOz` (400 → `ML 118`, Typed `floz`), `TestRoundingIsHalfUp` (table incl. 1 ml → 3 hundredths; 15 hundredths → 4 ml), `TestRejects` (0, -1, `MaxVolumeAmount+1`, 1 hundredth of an ounce (0.3 ml rounds to 0), unit `"cup"`), `TestTypedValueIsExact` (the typed column equals the input for both units, and the other column is within one unit of the exact rational value).
- [ ] **Step 2:** `go test ./internal/units/ -race` → FAIL (package missing).
- [ ] **Step 3: Implement** `NewVolume` with integer arithmetic only: `floz_h = round_half_up(ml * 1_000_000 / 295_735)`, `ml = round_half_up(h * 295_735 / 1_000_000)`.
- [ ] **Step 4:** the same command → PASS. `gofmt -l .`, `go vet ./...`, staticcheck as in CLAUDE.md.
- [ ] **Step 5: Commit** `feat: convert volumes between millilitres and US fluid ounces`.

## Task 2: Plumbing — limits, kid-zone days, age bands, ranges, cursors

**Files:** Create `api/internal/kidtime/kidtime.go` (+ `_test.go`), `api/internal/httpapi/query.go` (+ `query_test.go`); modify `api/internal/constants/constants.go`.

**Interfaces — Produces:**
```go
package kidtime
func DayBounds(loc *time.Location, instant time.Time) (start, end time.Time)       // [start,end) of the local calendar day containing instant
func ParseDate(s string, loc *time.Location) (time.Time, error)                    // "YYYY-MM-DD" → local midnight
func AgeMonths(birthDate string, loc *time.Location, now time.Time) (int, error)   // whole calendar months, never negative
type Band int; const (Band0to11 Band = iota; Band12to23; Band24Plus)
func BandFor(months int) Band
func BandStart(birthDate string, loc *time.Location, b Band) (time.Time, error)    // local midnight when the kid enters b (birth date for Band0to11)
```
```go
package httpapi // query.go
type listQuery struct{ From, To time.Time; Limit int; After *cursor }
type cursor struct{ At int64; ID string }
func parseListQuery(r *http.Request, now time.Time) (listQuery, string /* invalid field name or "" */)
func encodeCursor(c cursor) string; func decodeCursor(s string) (cursor, error)
```
Constants: `TrackerPageDefault = 50`, `TrackerPageMax = 100`, `MaxFutureSkew = 5 * time.Minute`, `MaxSessionDuration = 24 * time.Hour`, `MaxNoteLen = 500`, `SummaryMaxDays = 93`, `BandOverrideLookback = 60 * 24 * time.Hour`.
Defaults: `from` absent = the Unix epoch, `to` absent = `now + MaxFutureSkew`; range is half-open `[from, to)`; the cursor is base64url of `At:ID` and names the last returned entry (next page = strictly older).

- [ ] **Step 1: Failing tests.** `kidtime`: `TestDayBoundsDST` (Europe/Madrid 2026-03-29 is 23 h, 2026-10-25 is 25 h; America/Los_Angeles; a UTC instant just before and after local midnight lands in different days), `TestAgeMonths` (birth 2026-08-31 on 2026-09-30 is 0; on 2026-10-01 is 1; leap day birth; a kid born "tomorrow" in its zone is 0), `TestBandStart`. `query`: `TestParseListQuery` (defaults, `limit=0`, `limit=101`, `limit=abc`, `from` after `to`, non-RFC3339 `from` all give the offending field name), `TestCursorRoundTrip`, `TestBadCursor` (not base64, wrong shape, negative `At`, an `ID` that is not `ID_PATTERN`-safe).
- [ ] **Step 2:** run both packages → FAIL.
- [ ] **Step 3: Implement.** Use `time.Date(y, m, d, 0, 0, 0, 0, loc)` for day starts and the next day's start as the end (never add 24 h). `parseListQuery` returns the first invalid field in the order `from`, `to`, `limit`, `cursor`.
- [ ] **Step 4:** both packages PASS under `-race`; `go vet`, `gofmt`, staticcheck clean.
- [ ] **Step 5: Commit** `feat: kid-zone day bounds, age bands and list queries`.

## Task 3: Migration 0002 and the entry model

**Files:** Create `api/internal/store/migrations/0002_trackers.sql`, `api/internal/model/tracker.go`; modify `api/internal/store/store_test.go`.

**Interfaces — Produces** (`model`):
```go
type Entry struct{ ID, KidID, CreatedBy, UpdatedBy string; CreatedAt, UpdatedAt time.Time; Note *string }
type BottleFeed struct{ Entry; At time.Time; Contents []string /* non-empty subset of breast_milk, formula, cow_milk, other, in that order */; Volume units.Volume }
type Diaper struct{ Entry; At time.Time; Kind string }
type Meal struct{ Entry; At time.Time; What string; Reaction *string }
type PumpingSession struct{ Entry; StartedAt time.Time; DurationSeconds int; Volume units.Volume; Side string }
type SleepSession struct{ Entry; StartedAt time.Time; EndedAt *time.Time }
type NursingSession struct{ Entry; StartedAt time.Time; EndedAt *time.Time; LeftSeconds, RightSeconds int; Side *string; SideSince *time.Time }
type Module string // the eleven ids of Open value 1
var Modules []Module // fixed order as listed in Open value 1
```
Tables (all `STRICT`, common columns as in Global Constraints): `bottle_feeds(at, has_breast_milk, has_formula, has_cow_milk, has_other (each 0 or 1, CHECK sum >= 1), volume_ml, volume_floz_h, volume_typed CHECK)`, `diapers(at, kind CHECK)`, `meals(at, what, reaction)`, `pumping_sessions(started_at, duration_seconds, volume_ml, volume_floz_h, volume_typed, side CHECK)`, `sleep_sessions(started_at, ended_at, CHECK (ended_at IS NULL OR ended_at >= started_at))`, `nursing_sessions(started_at, ended_at, left_seconds, right_seconds, side CHECK, side_since, with the same ended CHECK)`, `kid_modules(kid_id, module CHECK, enabled, updated_at, PRIMARY KEY (kid_id, module))`. Partial unique indexes `sleep_one_running ON sleep_sessions (kid_id) WHERE ended_at IS NULL` and the same for nursing. Time indexes on `(kid_id, <time column>)`.

- [ ] **Step 1: Failing tests** in `store_test.go`: extend `wantTables` with the seven tables; `schema_version` now `"2"` in both assertions; `TestTrackerTablesCascadeWithKid` (insert one row of every table through plain SQL on `s.w`, `DeleteKid`, every table empty); `TestOneRunningRowPerKid` (second `ended_at IS NULL` insert for the same kid fails for sleep and for nursing; a different kid succeeds); `TestChecksRejectBadEnums` (one bad value each for contents, kind, side, module).
- [ ] **Step 2:** `go test ./internal/store/ -race` → FAIL.
- [ ] **Step 3: Write** the migration and `model/tracker.go`.
- [ ] **Step 4:** PASS; the existing "database newer than the binary" test still uses 99.
- [ ] **Step 5: Commit** `feat: tracker tables, running-row indexes and kid modules`.

## Task 4: Bottles and diapers (store, API, matrix)

**Files:** Create `api/internal/store/bottles.go`, `api/internal/store/diapers.go`, `api/internal/httpapi/bottles.go`, `api/internal/httpapi/diapers.go` (+ a `_test.go` each); modify `api/internal/store/store.go` (shared `ListQuery`, see below), `api/internal/httpapi/routes.go`, `api/internal/httpapi/matrix_test.go`, `api/internal/constants/constants.go`.

**Interfaces — Produces** (store; the same five-function shape is repeated by Tasks 5 and 6 with their own types):
```go
type ListQuery struct{ From, To time.Time; Limit int; After *Cursor } // Cursor{At int64; ID string}
func (s *Store) CreateBottle(ctx context.Context, kidID, actorID string, b model.BottleFeed, now time.Time) (model.BottleFeed, error)
func (s *Store) Bottle(ctx context.Context, kidID, id string) (model.BottleFeed, error)
func (s *Store) ListBottles(ctx context.Context, kidID string, q ListQuery) ([]model.BottleFeed, error)   // newest first, (at DESC, id DESC)
func (s *Store) UpdateBottle(ctx context.Context, kidID, actorID string, b model.BottleFeed, now time.Time) error
func (s *Store) DeleteBottle(ctx context.Context, kidID, actorID, id string) error
// the same set for Diaper
```
Routes, all class `Caregiver`: `GET|POST /v1/kids/{kid}/bottles`, `GET|PATCH|DELETE /v1/kids/{kid}/bottles/{entry}`, and the same five under `/diapers`. Wire: bottle `{id, at, contents:["breast_milk","formula"], volume:{ml, floz_hundredths, typed_unit}, note}`; create/patch body `{at?, contents:[…], volume:{unit, amount}, note?}` (contents: 1 to 4 distinct values from the four names, any order; stored and returned in the fixed order; empty, duplicate or unknown values are 422 `contents`) (`at` defaults to server now on create); diaper `{id, at, kind, note}`. Lists return `{"bottles":[...],"next_cursor":"..."|null}`. Create answers 201 with the view, PATCH 200 with the view, DELETE 204. Error codes: `invalid` with `field` (422) for validation, `unknown_field`, and the existing set.
Handlers use `decodeJSON` first, then `parseListQuery`/validation, then the store; PATCH merges the stored row with the patch (pointer and `optionalInt`-style presence for `note`) and validates the merged result before writing; a changed `volume` recomputes through `units.NewVolume`, an unchanged one keeps the stored three columns untouched.

- [ ] **Step 1: Failing tests.** Store: `TestBottleCRUD`, `TestBottleBelongsToItsKid` (another kid's id → `ErrNotFound` for read, update and delete), `TestBottleWriteByDisabledActor`, `TestEditingNoteKeepsTypedVolume`, `TestMixedBottleContents` (breast milk + formula, cow milk alone, all four; empty and duplicate sets refused; order canonicalised), `TestListNewestFirstWithEqualTimestamps` (same `at`, ids decide), same for diapers. HTTP (`testkit_test.go` helpers): `TestBottleRoutesRoles` (guardian and caregiver may create, edit, delete; a no-grant parent and the admin get the identical 404 and 403 as other kid routes), `TestBottleValidation` (every row of Open values 5–7: future by more than 5 minutes, before the birth date, volume 0, -1, over the cap, unknown or empty or duplicate contents, note over `MaxNoteLen`, unknown field), `TestBottleListRange` (`from`/`to`/`limit`/`cursor`; `next_cursor` null on the last page), `TestDiaperRoutes`, `TestNoValueInLogs` (capture the logger: after create/patch/delete with a distinctive note, the note and the volume number are absent from every line).
- [ ] **Step 2:** `go test ./internal/store/ ./internal/httpapi/ -race` → FAIL.
- [ ] **Step 3: Implement** store and handlers; add the ten routes to `Routes`; extend `matrix_test.go`: ten `routeClasses` rows, `matrixParams["entry"]` returns the seeded id of the table named by the route's second path segment (`switch` on `rt.Pattern`), the fixture seeds one bottle and one diaper for "Kid K".
- [ ] **Step 4:** the same command PASS, including the matrix; `gofmt`, `vet`, staticcheck.
- [ ] **Step 5: Commit** `feat(api): bottles and diapers`.

## Task 5: Solid meals and pumping

**Files:** Create `api/internal/store/meals.go`, `api/internal/store/pumping.go`, `api/internal/httpapi/meals.go`, `api/internal/httpapi/pumping.go` (+ tests); modify `routes.go`, `matrix_test.go`, `constants.go`.

**Interfaces — Consumes:** Task 4's `ListQuery`, `Cursor`, handler helpers. **Produces:** the five-function store shape for `Meal` and `PumpingSession` (`CreateMeal`, `Meal`, `ListMeals`, `UpdateMeal`, `DeleteMeal`; `CreatePumping`, `Pumping`, `ListPumping`, `UpdatePumping`, `DeletePumping`); routes under `/meals` (module `feeding`) and `/pumping`, class `Caregiver`. Wire: meal `{id, at, what, reaction, note}`; pumping `{id, started_at, duration_seconds, volume:{…}, side, note}`. Pumping is recorded on the kid it is for (spec 6.3). List keys `{"meals":…}` and `{"pumping":…}`. Limits: `MaxMealTextLen = 200`, `MaxPumpingVolume = 2000` ml, duration 1 second to `MaxSessionDuration`.

- [ ] **Step 1: Failing tests:** the Task 4 set for each (CRUD, belongs-to-kid, roles, validation of every field and enum, range and cursor, no values in logs), plus `TestMealWhatIsRequiredAndTrimmed` and `TestPumpingDurationBounds`.
- [ ] **Step 2:** run → FAIL. **Step 3:** implement; ten more routes, matrix rows and seeds. **Step 4:** PASS with the matrix. **Step 5: Commit** `feat(api): solid meals and pumping`.

## Task 6: Sleep and nursing timers

**Files:** Create `api/internal/store/sleep.go`, `api/internal/store/nursing.go`, `api/internal/httpapi/sleep.go`, `api/internal/httpapi/nursing.go` (+ tests); modify `store.go` (sentinels), `respond.go` (error mapping), `routes.go`, `matrix_test.go`.

**Interfaces — Produces:**
```go
var ErrAlreadyRunning, ErrNotRunning error // 409 "already_running" / "not_running"
func (s *Store) CreateSleep(ctx, kidID, actorID string, v model.SleepSession, now time.Time) (model.SleepSession, error)   // EndedAt nil starts the timer
func (s *Store) EndSleep(ctx, kidID, actorID, id string, at, now time.Time) (model.SleepSession, error)
func (s *Store) CreateNursing(ctx, kidID, actorID string, v model.NursingSession, now time.Time) (model.NursingSession, error) // Side non-nil and EndedAt nil starts
func (s *Store) SwitchNursingSide(ctx, kidID, actorID, id, side string, at, now time.Time) (model.NursingSession, error)
func (s *Store) EndNursing(ctx, kidID, actorID, id string, at, now time.Time) (model.NursingSession, error)
func (s *Store) RunningSleep(ctx, kidID string) (*model.SleepSession, error); func (s *Store) RunningNursing(ctx, kidID string) (*model.NursingSession, error)
// plus Sleep/ListSleep/UpdateSleep/DeleteSleep and Nursing/ListNursing/UpdateNursing/DeleteNursing as in Task 4
```
Routes (class `Caregiver`): the five CRUD routes under `/sleep` and `/nursing`, `POST /v1/kids/{kid}/sleep/{entry}/end`, `POST /v1/kids/{kid}/nursing/{entry}/end`, `POST /v1/kids/{kid}/nursing/{entry}/side` (body `{side, at?}`); end bodies `{at?}`. Creating with no `ended_at` starts a timer (`started_at` defaults to server now); creating with `ended_at` records a finished session (nursing then needs `left_seconds`/`right_seconds` with `left + right <= ended_at - started_at`). A running nursing row shows `side` and `side_since`; switching adds `at - side_since` to the side being left. PATCH may change `started_at`, `note`, and for a running row set `ended_at`; it never nulls `ended_at` (Open value 10). All `at` values pass the Task 2 validation.

- [ ] **Step 1: Failing tests.** Store: `TestSecondStartIsAlreadyRunning`, `TestSecondKidMayRunOneToo`, `TestEndNotRunningIsNotRunning`, `TestEndBeforeStartRefused`, `TestNursingSwitchAccumulatesSeconds` (left 0:10, right 0:05, left again, end: totals match to the second), `TestSwitchOnFinishedSession`, `TestConcurrentStarts` (two goroutines, one succeeds, one `ErrAlreadyRunning`), `TestFinishedNursingSecondsCannotExceedDuration`, `TestMaxSessionDuration`. HTTP: `TestSleepLifecycle` (start → 409 on second start → end → 409 `not_running` on second end), `TestNursingLifecycle`, roles and 404 identity as in Task 4, `TestNoValueInLogs` for sleep and nursing notes.
- [ ] **Step 2:** run → FAIL. **Step 3:** implement in single transactions on the writer with `actorRole`; the partial unique index is the backstop (map its constraint error to `ErrAlreadyRunning`). 14 routes and matrix rows; fixture seeds one finished and one running row of each (the running rows belong to a second fixture kid so the matrix's create probes do not trip over them).
- [ ] **Step 4:** PASS with `-race`. **Step 5: Commit** `feat(api): sleep and nursing timers`.

## Task 7: Timeline and daily summaries

**Files:** Create `api/internal/store/timeline.go`, `api/internal/store/summary.go`, `api/internal/httpapi/timeline.go`, `api/internal/httpapi/summary.go` (+ tests); modify `routes.go`, `matrix_test.go`.

**Interfaces — Produces:**
```go
type TimelineItem struct{ Type string; At time.Time; ID string; Entry any } // Type: "bottle","nursing","pumping","sleep","diaper","meal"
func (s *Store) Timeline(ctx context.Context, kidID string, types []string, q ListQuery) ([]TimelineItem, error)
type DaySummary struct{ Date string; Bottles struct{Count, ML, FlOzHundredths int64}; Nursing struct{Count, Seconds int64}; Pumping struct{Count, ML, FlOzHundredths int64}; Diapers struct{Total, Wet, Dirty, Both, Dry int64}; Sleep struct{Seconds int64}; Meals struct{Count int64} }
func (s *Store) Summaries(ctx context.Context, kidID string, loc *time.Location, from, to time.Time, now time.Time) ([]DaySummary, error) // days [from,to] inclusive, kid-local
```
Routes (class `Caregiver`): `GET /v1/kids/{kid}/timeline?from=&to=&types=&limit=&cursor=` → `{"items":[{"type","id","at", …the tracker's own view fields…}],"next_cursor":…}`; `GET /v1/kids/{kid}/summary?from=YYYY-MM-DD&to=YYYY-MM-DD` → `{"days":[…]}`. `types` is a comma list of the six type names; absent = the enabled trackers of Task 8's module state (until Task 8 lands the default is all six). Merge order is `(at DESC, id DESC)` over `at` = the time column of each table; each tracker query asks for `limit + 1` rows after the cursor and the merge keeps `limit`. Summary range is at most `SummaryMaxDays` days and `to` not before `from`; days with no entries are present with zeros.

- [ ] **Step 1: Failing tests.** `TestTimelineMergesInOrderAcrossTrackers`, `TestTimelinePagingWithEqualTimestamps` (ten entries sharing one second across all six tables, page size 3: union of pages equals the set, no repeats), `TestTimelineTypesFilter` (unknown type → 422 `types`), `TestTimelineRangeIsHalfOpen`, `TestSummaryAcrossDSTDay` (Europe/Madrid, a sleep from 22:00 on 2026-03-28 to 08:00 on 2026-03-29 local: the 28th gets 2 h and the 29th gets 7 h of real elapsed time, because the 29th has 23 hours; assert against `DayBounds` results, not hand-typed instants), `TestSummarySleepClippedAtMidnightAndNow`, `TestSummaryKidZoneNotServerZone`, `TestSummaryRangeLimit`, `TestTimelineOtherKidsEntriesNeverAppear`. Matrix rows for both routes.
- [ ] **Step 2:** run → FAIL. **Step 3:** implement. Summaries use SQL aggregates per tracker over `[firstDayStart, lastDayEnd)` and bucket by day in Go using `DayBounds`; no per-row unbounded load. **Step 4:** PASS. **Step 5: Commit** `feat(api): timeline and daily summaries`.

## Task 8: Modules and the home read model

**Files:** Create `api/internal/store/modules.go`, `api/internal/httpapi/modules.go`, `api/internal/httpapi/home.go` (+ tests); modify `routes.go`, `matrix_test.go`, `timeline.go` (default types), `constants.go`.

**Interfaces — Produces:**
```go
type ModuleState struct{ Module model.Module; Enabled, Default, Overridden bool }
func (s *Store) Modules(ctx context.Context, kidID string, loc *time.Location, now time.Time) ([]ModuleState, error) // runs the band-crossing rule (Open value 3) first, in one writer transaction
func (s *Store) SetModule(ctx context.Context, kidID, actorID string, m model.Module, enabled bool, now time.Time) error // guardian only (actorIsGuardian)
func DefaultEnabled(m model.Module, b kidtime.Band) bool // the 7.2 table
```
Routes: `GET /v1/kids/{kid}/modules` (`Caregiver`) → `{"modules":[{"module","enabled","default","overridden"}]}` in `model.Modules` order; `PUT /v1/kids/{kid}/modules/{module}` (`Guardian`, body `{"enabled":bool}`, 204, unknown module 404 `not_found` after the access check); `GET /v1/kids/{kid}/home` (`Caregiver`) → `{"age_months":n,"band":"0-11"|"12-23"|"24+","modules":[…],"running":{"sleep":view|null,"nursing":view|null},"today":{"date":"YYYY-MM-DD", …the Task 7 DaySummary…}}`. The timeline default `types` becomes the enabled trackers among the six.

- [ ] **Step 1: Failing tests.** `TestDefaultsFollowTheTable` (each module × each band against the 7.2 table, including growth, vaccination, sickness and medication on at 24+), `TestOverrideWinsEitherWay`, `TestBandCrossingKeepsModuleWithRecentEntry` (kid turns 24 months: bottles with an entry 45 days before the boundary get an `enabled` row; one 70 days before does not; diapers with none do not; a running sleep counts), `TestBandCrossingIsIdempotentAndNotRepeatedAfterGuardianTurnsItOff`, `TestCaregiverCannotSetModule` (403), `TestSwitchedOffTrackerKeepsDataAndReturns` (entries survive off and on), `TestHomeShape` (`sameKeys`), `TestHomeTodayUsesKidZone`, `TestHomeShowsRunningTimers`. Matrix rows: `GET modules` Caregiver, `PUT modules/{module}` Guardian with `matrixParams["module"]` returning `"bottles"`, `GET home` Caregiver.
- [ ] **Step 2:** run → FAIL. **Step 3:** implement. **Step 4:** PASS. **Step 5: Commit** `feat(api): per-kid modules and the home read model`.

## Task 9: UI plumbing — client, types, stub, zone-aware times

**Files:** Modify `ui/src/lib/server/api.ts` (+ `api.test.ts`), `ui/src/lib/types.ts`, `ui/src/lib/constants.ts`, `ui/src/lib/server/constants.ts`, `ui/src/lib/format.ts` (+ test), `ui/src/lib/time-zone.ts` (+ test), `ui/e2e/stub-api.ts`; create `ui/src/params/tracker.ts` (SvelteKit param matcher), `ui/src/lib/server/entry-form.ts` (+ test).

**Interfaces — Produces:**
```ts
export type TrackerKind = 'bottles' | 'nursing' | 'pumping' | 'sleep' | 'diapers' | 'meals';
export const TRACKER_KINDS: readonly TrackerKind[];  // params/tracker.ts matcher accepts exactly these
// api.ts
api.tracker(kind).list(kidId, query) / .get(kidId, id) / .create(kidId, input) / .update(kidId, id, input) / .remove(kidId, id)
api.endTimer(kind: 'sleep'|'nursing', kidId, id, at?) ; api.switchNursingSide(kidId, id, side, at?)
api.timeline(kidId, query) ; api.summary(kidId, from, to) ; api.modules(kidId) ; api.setModule(kidId, module, enabled) ; api.home(kidId)
// time-zone.ts
export function zonedLocalToInstant(local: string /* 'YYYY-MM-DDTHH:mm' */, zone: string): string | null // RFC3339 UTC; null if not a valid local time
export function instantToLocalInput(instant: string, zone: string): string
// format.ts
export function formatVolume(v: { ml: number; floz_hundredths: number }, unit: 'ml' | 'floz'): string
export function formatDuration(seconds: number): string
export function ageMonths(birthDate: string, today: string): number
```
Rules: the client follows the four-edit pattern of `api.ts` (Endpoint union, parsers with `ShapeError`, methods through `withId`, types); one parser per tracker kind in a `Record<TrackerKind, Parser>`; `logFailure` keeps its fixed shape. `entry-form.ts` exports `readEntryForm(kind, FormData, zone): { values, input: object | null, field: string | null }`. The stub implements every new route in memory (Go's validation and 404/409 behaviour mirrored closely enough for the UI tests), with the new state in `State`/`freshState()`.

- [ ] **Step 1: Failing tests:** `zonedLocalToInstant` (Europe/Madrid 2026-03-29T02:30 → moves forward to 03:30 local = 01:30Z; 2026-10-25T02:30 → the earlier instant 00:30Z; America/Los_Angeles 2026-03-08T02:30; garbage and 24:00 → null), `instantToLocalInput` round trip, `formatVolume`, `formatDuration`, `ageMonths` against the Go `AgeMonths` table (same cases), `readEntryForm` per kind (valid, missing field, bad enum, volume in ounces → hundredths; bottle contents from several checkbox values, none ticked is the `contents` field error), `api.test.ts` cases for one list, one create failing with 422 `field`, `endTimer` 409 → reason `conflict`, a malformed list body → `error`.
- [ ] **Step 2:** `cd ui && pnpm test` → FAIL. **Step 3:** implement. **Step 4:** `pnpm check && pnpm lint && pnpm test` PASS. **Step 5: Commit** `feat(ui): tracker client, zone-aware times and stub routes`.

## Task 10: UI — kid home, navigation, module settings

**Files:** Modify `ui/src/routes/(app)/+page.server.ts`, `+page.svelte`, `(app)/+layout.svelte`, `(app)/kids/[kid]/+page.server.ts`, `+page.svelte`, `ui/src/lib/components/KidLinks.svelte`, `AppShell.svelte` (only if needed for the link list), the e2e specs that pin the old home; create `ui/src/routes/(app)/timeline/+page.server.ts` (redirect), `ui/src/routes/(app)/kids/[kid]/modules/+page.server.ts` and `+page.svelte`, `ui/src/lib/components/{QuickLog,RunningTimer,TodaySummary}.svelte`.

**Interfaces — Consumes:** Task 9's client and stub. **Produces:** the kid home at `/kids/[kid]` per spec 7.3 in order: kid switcher with age (`formatAge`), quick-log links for enabled and built modules in the per-band order of 7.3 (the rest in `model.Modules` order; links go to `/kids/{kid}/log/{tracker}`, sleep is a form action `?/startSleep`), the running timers (sleep: `Asleep since 13:05` with an End form; nursing: side, since, Switch side and End forms), `Today` from the summary in the parent's display units; `/` redirects 303 to `/kids/{current}`; `/timeline` redirects to `/kids/{current}/timeline`; bottom bar `Home /`, `Timeline /timeline`, `Account /account`; guardian-only `/kids/[kid]/modules` page (a checkbox per module with its default shown, POST saves).

- [ ] **Step 1: Failing tests (Playwright, both projects, axe):** `home.spec.ts` — a kid under 12 months shows Bottle, Nursing, Diaper, Sleep in that order; a 13-month kid starts with feeding; a 30-month kid shows no timer trackers and no empty "Due" heading; one tap on Sleep starts a timer that the home then shows with End; End returns to the normal buttons; a caregiver sees no Modules link and `/kids/k-mia/modules` is the 404 page; switching a module off hides its button and keeps the timeline entry; `/` with no kids still shows Welcome and "Add your first child". Update `shell.spec.ts` (new link list) and every spec that asserted `Welcome, …` or the old card list (`account`, `invite`, `kids`, `login`, `reset`, `unavailable`) to assert the new signed-in landing.
- [ ] **Step 2:** `pnpm test:e2e --trace off` → FAIL (new specs) and the updated specs fail until the pages exist. **Step 3:** implement; `Cache-Control: no-store` stays. **Step 4:** the full UI pipeline from CLAUDE.md PASS. **Step 5: Commit** `feat(ui): kid home, navigation and module settings`.

## Task 11: UI — log and edit pages for the six trackers

**Files:** Create `ui/src/routes/(app)/kids/[kid]/log/[tracker=tracker]/+page.server.ts`, `+page.svelte`, `.../log/[tracker=tracker]/[entry]/+page.server.ts`, `+page.svelte`, `ui/src/lib/components/{EntryTimeField,VolumeField,TrackerForm}.svelte` (names are a suggestion); modify `ui/e2e/stub-api.ts` only if Task 9 left a gap.

**Interfaces:** `…/log/{tracker}` creates (time prefilled to now in the kid's zone, all fields, volume with a unit select defaulting to the parent's preference); `…/log/{tracker}/{entry}` edits and deletes (delete asks for a second confirming POST without JavaScript via `<details>`, as the kid-leave form does); timers: the nursing form starts with a side choice or records a finished session; running sleep/nursing entries show End; every failed field returns `fail(422, {form, values, field, message})` with `aria-invalid` and `role="alert"`; success redirects 303 to the timeline row's day.

- [ ] **Step 1: Failing tests:** `trackers.spec.ts` — for each of the six: create with a note (the bottle with two contents ticked), see it on the timeline, edit one field, delete it; a volume typed in ounces shows in ounces for a parent with that preference and in millilitres for one without; a time in the future by an hour shows the field error without losing the other values; a second nursing start while one runs shows the conflict message; DST: a Europe/Madrid kid, 2026-03-29 02:30 saves as 03:30 local; `readEntryForm` unit tests already cover parsing. All at phone and desktop widths with axe.
- [ ] **Step 2:** run → FAIL. **Step 3:** implement. **Step 4:** the full UI pipeline PASS. **Step 5: Commit** `feat(ui): log and edit pages for the daily trackers`.

## Task 12: UI — timeline

**Files:** Create `ui/src/routes/(app)/kids/[kid]/timeline/+page.server.ts`, `+page.svelte`, `ui/src/lib/components/TimelineRow.svelte`.

**Interfaces:** `GET /kids/{kid}/timeline?from=YYYY-MM-DD&to=YYYY-MM-DD&types=a,b&cursor=…` — a GET filter form (two dates in the kid's zone converted to UTC bounds by the page server, a checkbox per built tracker), entries grouped under kid-local day headings with one line each (`13:05 Bottle 120 ml, breast milk + formula`, `Sleep 13:05–14:40`, `Nursing L 8 min R 5 min`), each linking to its edit page, an "Older" link carrying `cursor` (works without JavaScript), and the day's summary under the heading using Task 7's endpoint.

- [ ] **Step 1: Failing tests:** `timeline.spec.ts` — six entries across two local days show under two headings; the type filter hides a tracker; "Older" reaches the second page and shows no repeats; a switched-off module's entries still show when its box is ticked; an invalid `from` shows the field error, not a crash; phone and desktop with axe.
- [ ] **Step 2:** run → FAIL. **Step 3:** implement. **Step 4:** pipeline PASS. **Step 5: Commit** `feat(ui): timeline page`.

## Task 13: PWA manifest and install metadata

**Files:** Create `ui/static/manifest.webmanifest`, `ui/static/icons/icon.svg`, `ui/static/icons/icon-192.png`, `ui/static/icons/icon-512.png`, `ui/static/icons/icon-512-maskable.png`, `ui/src/lib/manifest.test.ts`; modify `ui/src/app.html` (and `routes/+layout.svelte` only if the head needs it); render the PNGs once from the SVG with the Chromium that Playwright already installs and commit them (no build-time dependency).

- [ ] **Step 1: Failing test** `manifest.test.ts` (vitest, node): the manifest parses; has `name`, `short_name`, `start_url "/"`, `scope "/"`, `display "standalone"`, `theme_color`, `background_color`; icons list 192, 512 and a `maskable` 512, and each file exists with the declared size (read the PNG header); `app.html` links `rel="manifest"`, sets `theme-color` and an apple touch icon.
- [ ] **Step 2:** `pnpm test` → FAIL. **Step 3:** implement. Static files are served by the Node adapter before the SvelteKit `handle`, so the manifest needs no session; confirm it in Task 14's smoke rather than assuming it. **Step 4:** pipeline PASS. **Step 5: Commit** `feat(ui): installable PWA manifest and icons`.

## Task 14: Smoke, CI, documentation

**Files:** Modify `scripts/smoke.sh`, `.github/workflows/ci.yml` (only if a step must change), `README.md`, `docs/superpowers/HANDOFF.md`, `CLAUDE.md` only if a rule changed (none expected).

- [ ] **Step 1: Extend `scripts/smoke.sh`** (failing first against the current images): after step 3 create one entry of each of the six trackers through the UI forms with distinctive notes (`remember` each note and each volume figure), start and end a sleep and a nursing timer, read the timeline page and assert every note appears, read the summary line on the home page, fetch `/manifest.webmanifest` and `/icons/icon-192.png` with no cookie (200, manifest content type), assert a caregiver can edit another parent's entry, assert the admin session gets no tracker route, delete an entry and see it gone, re-run the log check (step 9) so no note or value is in either container's logs, and keep the memory ceilings (60 MiB API, 140 MiB UI).
- [ ] **Step 2:** `docker build -t hbt-api:dev api && docker build -t hbt-ui:dev ui && scripts/smoke.sh hbt-api:dev hbt-ui:dev` → every line `PASS:` (run the failing form first against images from `main` to see the new checks fail).
- [ ] **Step 3: Docs:** README feature list and the module table; `HANDOFF.md` brought up to date (plans 1–3 done, what plan 4 starts from, the stale "no product code exists" lines removed).
- [ ] **Step 4:** the whole pipeline in CLAUDE.md (API, UI, images, smoke) green on the branch; `shellcheck scripts/smoke.sh` clean.
- [ ] **Step 5: Commit** `test: smoke covers the daily trackers; docs: plan 3 done`.

---

## Done when

- API and UI pipelines and `scripts/smoke.sh` pass on the branch head, and CI is green.
- The access matrix covers every new route and fails if one is missing.
- A parent can sign in on a phone-width browser, log each tracker, see the timeline and today's summary in their units, switch a module off and on, and install the app from the browser menu or the install prompt.
- The final report states which image content changed (`api/` and `ui/` sources; no Dockerfile or dependency change expected), that the edge still needs its two pins bumped (the `bump-happybabytracker` PR) and a deploy, and that the home screen shows no "Due" section until plans 5 and 6.
