# Advanced Bottle, Nursing and Pumping Logging Implementation Plan (plan 5.1)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A parent can log a bottle with a volume per source (breast milk, formula, cow's milk, other), and can record the baby's weight before and after a nursing session; the app shows the weight difference and an estimated milk volume. A pumping session can record a volume per side (left and right). All three are opt-in per form ("advanced"); the simple forms and every existing entry behave as before.

**Architecture:** One migration (0005) adds `bottle_feed_sources` (one row per source of an advanced bottle), `pumping_session_sides` (one row per side of an advanced pumping session) and six nullable weight columns on `nursing_sessions`. The bottle's and the pumping session's own volume columns stay and always equal the sum of their sources or sides, so the timeline, daily summary and home read model need no change. A small pure function in `api/internal/units` turns two weights into the estimate. The UI gets an "advanced" toggle on the bottle and nursing forms (a plain link, works without JavaScript), a weight field in the parent's unit, and a one-line estimate on the nursing edit page and the timeline.

**Tech Stack:** Go 1.27, `modernc.org/sqlite`, stdlib `net/http` mux; SvelteKit 3 / Svelte 5 runes, Tailwind 4. No new dependency, no new route.

**Spec:** `docs/superpowers/specs/2026-11-20-happybabytracker-design.md` sections 6.2 (units), 6.3 (trackers), 7. Earlier plans: `docs/superpowers/plans/2026-11-22-daily-trackers.md` (bottles, nursing), `docs/superpowers/plans/2026-11-22-growth.md` (weight units).

## Global Constraints

- Every limit, cap and constant lives in `api/internal/constants/constants.go` (API) or `ui/src/lib/server/constants.ts` with client-safe mirrors in `ui/src/lib/constants.ts`. No literal elsewhere (CLAUDE.md rule 7).
- No new route: the five bottle routes and the nursing routes keep their paths, classes and 404 behaviour; `matrix_test.go` needs no new row.
- A weigh-in is **not** a growth measurement: it creates no `growth_measurements` row, is never scored, never plotted and never appears on the Growth page or in the Growth list; it lives only on its nursing session.
- The app records and displays; it never interprets (spec 2). The estimate is arithmetic on two typed numbers. No wording such as "good", "low", "enough" or "concern" anywhere, and a lower "after" weight shows no estimate and no message about why.
- Weights are stored three ways like every measured value (grams, hundredths of an ounce, the unit typed); converted in `api/internal/units` only at save and edit. Volumes likewise (ml, hundredths of a US fl oz, the unit typed).
- A caller with no grant, an unknown kid and an entry of another kid answer the same 404 body; the admin is refused; writes re-assert the role with `actorRole` inside `inTx`; hard deletes; no edit history.
- Logs carry ids and enumerated facts only: never a weight, volume, note or name (rule 5, spec 13.3).
- Comments state constraints only, at most 2 lines (rule 9). `gofmt -l` prints nothing; staticcheck clean; Prettier tabs; every changed page works without JavaScript, passes axe at 390×844 and 1280×800 and keeps `Cache-Control: no-store`.
- Tests that pin counts change together with the code they pin: `wantTables` and `schema_version` (`"5"`), the bottle and nursing JSON key sets in `bottles_test.go`, `nursing_test.go`, `timeline_test.go` (`sameKeys`), the stub's views.
- Existing rows are untouched: a bottle with several contents and no sources stays a simple bottle; a nursing session without weights stays as it is. No backfill.

## Review Focus

Failure modes no task's happy path exercises; each has a test in the named task.

1. **Sum is the total.** An advanced bottle's total (`volume`) equals the sum of its sources' typed amounts in the unit typed, through create, a PATCH that changes one source, a PATCH that changes only the note (all stored columns untouched), and a PATCH that drops a source via `contents` (Tasks 2, 3).
2. **Switching modes loses nothing silently.** Advanced → simple through a `volume` body removes the sources and keeps the new total; simple → advanced through `sources` creates them and replaces the total; a body with both `volume` and `sources` is a 422 (Task 2).
3. **Source set equals contents.** `contents` and `sources` never disagree: a `sources` key outside the four names, a zero or negative amount, a `contents` that differs from the `sources` keys, an empty `sources` object are 422s naming the field, never a 500 (Task 2).
4. **No N+1.** A list of 100 bottles reads their sources with one extra query, and the timeline carries sources for bottle items (Task 2).
5. **Weights stand alone.** A nursing session with only a before weight, only an after weight, both, neither; a PATCH that clears one with `null`; a weight outside the range; a weight on a running session; a note-only PATCH rewrites no weight column (Task 3).
6. **Estimate edges.** Both present and after ≥ before: grams, ounces, ml and fl oz, rounded once from grams; after < before or either missing: `milk` is `null`, never negative, never an error; after == before: a zero estimate (Task 3).
7. **Kid delete.** `ON DELETE CASCADE` reaches the sources table; the migration upgrades a database at 0004 holding mixed bottles and nursing rows unchanged (Task 1).

---

## Open values (owner confirmation before execution)

Rule 1 (never assume an owner value). Rows 1–7 are the owner's answers of 2026-11-23; rows 8–16 are this plan's proposals with their basis and the cost if wrong.

| # | Value | Decision | Basis |
|---|---|---|---|
| 1 | Order | This plan ships first, as its own PR, before plan 6; plan 6's migration becomes 0006 | owner |
| 2 | Mode switch | A per-form toggle only: a link "Advanced: volume per source" / "Advanced: weigh before and after" opens the form with `?advanced=1`; nothing is remembered, every form starts simple. An edit page opens advanced on its own when the entry already carries advanced data | owner |
| 3 | Bottle total | In advanced mode every ticked source has its own volume and the total is their sum; the total is not editable on its own | owner |
| 4 | Nursing weigh-ins | One weight before and one after the whole session (both sides together); each optional | owner |
| 5 | Estimate shown | Both the weight difference (in the parent's weight unit: grams for kg users, ounces for lb users) and an estimated milk volume in the parent's volume unit (ml or fl oz), converted at 1.03 g/ml | owner |
| 6 | After below before | Both weights are stored as typed; no estimate is shown, no message | owner |
| 7 | Weight input | The parent's unit, the same input as the growth form: kg to whole grams, or lb + oz to hundredths of an ounce; range 200 g to 250 kg (`MinWeightGrams`, `MaxWeightGrams`) | owner |
| 8 | Density | `MilkDensityHundredths = 103` (1.03 g per ml) in `constants.go`; the app's own assumption like the corrected-age rule, stated in help text and README, never presented as measured | owner gave the number; the constant and the wording are mine |
| 9 | Per-source unit | One unit select applies to every source of a bottle; each source is typed in it. The total is `NewVolume(unit, Σ typed amounts)`, so the typed column of the total is exact; each source row also stores its own three columns, which can differ from the total's other column by rounding (at most one unit per source) | the existing invariant "typed exact, other within one unit" |
| 10 | Wire shape, bottle | Body: `sources: {"unit":"ml"\|"floz","amounts":{"breast_milk":60,"formula":90}}`; `volume` and `sources` are mutually exclusive; `contents` may be sent with `sources` only if it equals its keys. View: `volume` as today plus `sources: [{source, volume:{ml, floz_hundredths, typed_unit}}]` in `model.ContentNames` order, `[]` for a simple bottle. Absent `sources` and `volume` on PATCH keeps both | design |
| 11 | Wire shape, nursing | Body: `weight_before`, `weight_after`: `{"unit":"kg"\|"lb","amount":n}` or `null` (grams, or hundredths of an ounce); absent keeps. View: `weight_before`, `weight_after`: `{grams, oz_hundredths, typed_unit}` or `null`, and `milk`: `null` or `{gain:{grams, oz_hundredths}, volume:{ml, floz_hundredths}}` | design; same units as growth |
| 12 | Estimate arithmetic | `gain_grams = after.grams - before.grams`, `gain_oz_h = after.oz_h - before.oz_h`, `ml = round_half_up(gain_grams × 100 / 103)`, `floz_h = round_half_up(gain_grams × 100000000 / (103 × 295735))`, integers only, one rounding from grams. `milk` is `null` unless both weights exist and `gain_grams >= 0`. `gain_oz_h` is clamped at 0 (a pound-ounce weight against a kilogram weight can differ by rounding alone); the HTTP layer and the UI take the API's columns as they are | owner formula; integer arithmetic as everywhere; clamp ruled in Task 1 review |
| 13 | Where weigh-ins can be set | On create (before, after, or both) and on edit, for a running or a finished session. The home's one-tap **End** stays one tap and takes no weight: to record an "after" weight, end the session and edit it (cost: one extra step; a field on the home's End form would put a number input on the busiest screen) | design |
| 14 | Where the estimate shows | The nursing edit page (a status paragraph under the weights) and the timeline line (`… , weight gain 62 g, about 60 ml`). Not on the home Today list, not in daily summaries, no per-source daily totals for bottles | design (call) |
| 15 | Toggle without JavaScript | A plain link reloads the form page with `?advanced=1`; typed values are not carried across (cost: switch before typing). On a bottle edit, "Use one total" (`?advanced=0`) opens the simple form; saving it replaces the sources with the total, and the form says so | design |
| 16 | Limits | Bottle total 1–1000 ml unchanged (`MaxBottleVolumeML`); each source at least 1 ml (or the derived column not zero); at most four sources (the four names); weight range per row 7 | existing |
| 17 | Pumping, owner steer 2026-11-23 | An "advanced" pumping form records a volume per side (left, right); the total is their sum; duration stays one value for the whole session (no per-side duration was asked for). Applied by analogy with the bottle answers (rows 2, 3, 9): per-form toggle only, sum is the total, one unit select for both sides | owner steer; analogy is mine |
| 18 | Side in advanced pumping | Derived, never typed: left only → `left`, right only → `right`, both → `both`; at least one side must carry a volume (each at least 1 ml or a non-zero derived column). A simple pumping form is unchanged (a `side` radio and one volume) | design |
| 19 | Wire shape, pumping | Body: `sides: {"unit":"ml"\|"floz","amounts":{"left":60,"right":50}}`; `volume` and `sides` are mutually exclusive (422 `sides`); `side`, if sent beside `sides`, must equal the derived side (else 422 `side`). View: `volume` and `side` as today plus `sides: [{side, volume:{ml, floz_hundredths, typed_unit}}]` in left, right order, `[]` for a simple session. Absent `sides` and `volume` on PATCH keeps both; `volume` (with `side`) clears the sides; the total stays within 1–2000 ml (`MaxPumpingVolume`). Table `pumping_session_sides(session_id, side, volume_ml, volume_floz_h, volume_typed)`, PK (session_id, side), added to the still-unreleased 0005 | design; same shape as bottle sources |
| 20 | Where per-side shows | The pumping edit page and the timeline line (`… 110 ml (L 60 ml · R 50 ml)`); the daily summary and home keep the total only | design (call) |

---

## Task 1: Migration 0005, models and the estimate function

**Files:** Create `api/internal/store/migrations/0005_advanced_logging.sql`, `api/internal/units/milk.go` (+ `milk_test.go`); modify `api/internal/model/tracker.go`, `api/internal/constants/constants.go`, `api/internal/store/store_test.go`.

**Interfaces — Produces:**
```go
// model
type BottleSource struct{ Source string; Volume units.Volume }
type BottleFeed struct{ …; Sources []BottleSource /* nil or empty = simple bottle; else one per Contents name, in ContentNames order */ }
type NursingSession struct{ …; WeightBefore, WeightAfter *units.Weight }
// units
type Milk struct{ GainGrams, GainOzHundredths, ML, FlOzHundredths int64 }
// ok is false unless both are non-nil and after.Grams >= before.Grams
func MilkTransfer(before, after *Weight) (Milk, bool)
```
Constants: `MilkDensityHundredths = 103`. Migration:
```sql
CREATE TABLE bottle_feed_sources (
  bottle_id TEXT NOT NULL REFERENCES bottle_feeds (id) ON DELETE CASCADE,
  source TEXT NOT NULL CHECK (source IN ('breast_milk','formula','cow_milk','other')),
  volume_ml INTEGER NOT NULL, volume_floz_h INTEGER NOT NULL,
  volume_typed TEXT NOT NULL CHECK (volume_typed IN ('ml','floz')),
  PRIMARY KEY (bottle_id, source)
) STRICT;
ALTER TABLE nursing_sessions ADD COLUMN weight_before_g INTEGER;   -- and weight_before_oz_h, weight_before_typed CHECK IN ('kg','lb')
-- the same three for weight_after; every triplet is all NULL or all set (CHECK)
```
(Verify SQLite accepts `ALTER TABLE … ADD COLUMN … CHECK` on a STRICT table, as plan 4 did for `growth_source`; if a triplet CHECK cannot be added by `ALTER`, rebuild the table with the usual copy-and-rename inside the migration.)

- [ ] **Step 1: Failing tests.** `units`: `TestMilkTransfer` (before 3400 g, after 3462 g → gain 62, ml 60, floz_h 204 [62e8 / 30460705 = 203.55]; equal weights → zero estimate, ok true; after below before → ok false; either nil → ok false; table of ties for the half-up rounding; before/after typed in lb, whose ounce columns differ from grams by rounding: gain_oz_h = after.oz_h − before.oz_h, never negative when grams are not lower). Store: `wantTables` gains `bottle_feed_sources`; both `schema_version` assertions become `"5"`; `TestAdvancedColumnsCascadeWithKid` (a source row and a weighed session are gone after `DeleteKid`); `TestSourceChecks` (bad source name, bad typed unit, duplicate (bottle, source), a half-set weight triplet refused; a valid row accepted); `TestMigration0005KeepsRows` (a database at 0004 with a mixed bottle and a finished nursing session: both survive, no source rows, weights NULL).
- [ ] **Step 2:** `cd api && go test ./internal/units/ ./internal/store/ -race` → FAIL. **Step 3: Implement** (integer arithmetic only). **Step 4:** PASS; `gofmt -l .`, `go vet ./...`, staticcheck as in CLAUDE.md. **Step 5: Commit** `feat(api): advanced logging migration, models and the milk estimate`.

## Task 2: Bottles with a volume per source

**Files:** Modify `api/internal/store/bottles.go`, `api/internal/store/timeline.go` (bottle items), `api/internal/httpapi/bottles.go`, `api/internal/httpapi/entries.go` (shared helpers only if needed), `api/internal/httpapi/timeline.go` (+ their tests), `api/internal/constants/constants.go`.

**Interfaces — Consumes:** Task 1. **Produces:** store functions keep their signatures; `CreateBottle`/`UpdateBottle` write `b.Sources` (replacing all rows of the bottle in the same transaction; empty means none), `Bottle`/`ListBottles`/the timeline load `Sources` (one extra query per page: `WHERE bottle_id IN (…)`). The store refuses (`ErrInvalidContents`) a `Sources` set that differs from `Contents`. Handler: `bottleInput.Sources *sourcesInput` (`{Unit units.Unit; Amounts map[string]int64}`); `mergeOnto` order `at`, `contents`, `volume`/`sources`, `note`; rules of Open values 9, 10 and 16. A PATCH with `sources` replaces the set (the new keys become `contents`; a `contents` sent beside it must equal them); with `volume` it clears `Sources`; with neither it keeps both, and a `contents` change on an advanced bottle that no longer matches its sources is 422 `contents`. Unchanged per-source typed amounts in the same unit keep their stored columns (as `mergeVolume` does). View gains `sources`.

- [ ] **Step 1: Failing tests.** Store: `TestBottleSourcesRoundTrip`, `TestBottleSourcesReplacedOnUpdate`, `TestSourcesMustMatchContents`, `TestListLoadsSourcesWithOneExtraQuery` (count statements with a wrapped driver or `sqlite_stmt`-free approach the package already uses; else assert via a 100-row list within a time bound is NOT acceptable — use a statement counter), `TestTimelineBottleCarriesSources`. HTTP: `TestAdvancedBottleCreate` (breast milk 60 + formula 90 ml → total 150 ml, sources in fixed order, contents derived), `TestAdvancedBottleOunces` (typed exact, total typed = Σ), `TestSumIsTheTotal` (create, PATCH one source, PATCH note only: columns of total and each source untouched), `TestSwitchToSimpleAndBack` (Review Focus 2), `TestBothVolumeAndSources` (422 `sources`), `TestSourcesValidation` (unknown key, empty object, zero, negative, huge, unknown unit, `contents` mismatch → 422 `contents`, total over 1000 ml → 422 `volume`), `TestSimpleBottleViewHasEmptySources`, roles and 404 identity unchanged (existing tests stay green), `TestNoValuesInLogs` with distinctive per-source numbers.
- [ ] **Step 2:** `go test ./internal/store/ ./internal/httpapi/ -race` → FAIL. **Step 3: Implement.** **Step 4:** PASS including the matrix. **Step 5: Commit** `feat(api): bottles with a volume per source`.

## Task 3: Nursing weigh-ins and the estimate

**Files:** Modify `api/internal/store/nursing.go` (all nursing selects and the insert/update), `api/internal/store/timeline.go`, `api/internal/httpapi/nursing.go` (+ tests), `api/internal/httpapi/timeline.go` if its nursing item builds its own view.

**Interfaces — Consumes:** Task 1 (`units.NewWeight`, `units.MilkTransfer`). **Produces:** `nursingInput.WeightBefore`, `.WeightAfter` as `optionalWeight` (absent keeps, `null` clears, `{unit, amount}` sets; an unchanged typed value keeps the stored three columns), validated in `newSession` after `side` and in `mergeOnto` after `side` (before `note`), range `MinWeightGrams`..`MaxWeightGrams` → 422 `weight_before` / `weight_after`; allowed on running and finished sessions; the existing end, side and seconds rules are unchanged. `nursingView` gains `weight_before`, `weight_after` and `milk` (Open values 11 and 12). The home's `running.nursing` view and the timeline item use the same view.

- [ ] **Step 1: Failing tests.** Store round trip with each combination of weights (Review Focus 5), `TestNoteOnlyUpdateKeepsWeights`. HTTP: `TestNursingWeighIns` (create finished with both → `milk` present; create running with before only → `milk` null; PATCH after on a running session; PATCH `null` clears), `TestMilkEstimate` (62 g gain → gain 62 g, ml 60, floz_h 204), `TestAfterBelowBefore` (both stored and returned, `milk` null, 200), `TestWeightValidation` (below 200 g, above 250 kg, zero, negative, unknown unit, a bare number → 422 naming the field), `TestWeighInIsNotGrowth` (a weighed session creates no growth row and `GET …/growth` is unchanged), `TestNursingViewKeys` (`sameKeys` with the three new keys in the nursing view, the home's running view and the timeline item), `TestNoValuesInLogs` with a distinctive weight.
- [ ] **Step 2:** run → FAIL. **Step 3: Implement.** **Step 4:** the full API pipeline from CLAUDE.md PASS. **Step 5: Commit** `feat(api): nursing weigh-ins and the milk estimate`.

## Task 4: UI plumbing — types, client, stub, form readers, formatters

**Files:** Modify `ui/src/lib/types.ts`, `ui/src/lib/constants.ts`, `ui/src/lib/server/constants.ts` if needed, `ui/src/lib/server/api.ts` (+ `api.test.ts`), `ui/src/lib/server/entry-form.ts` (+ test), `ui/src/lib/server/growth-form.ts` (export its weight parser only), `ui/src/lib/entry-edit.ts`, `ui/src/lib/timeline.ts` (+ test), `ui/src/lib/format.ts` (+ test), `ui/e2e/stub-api.ts`.

**Interfaces — Produces:** types `BottleSource`, `Bottle.sources`, `Nursing.weightBefore`, `.weightAfter`, `.milk` (`{gain:{grams, ozHundredths}, volume:{ml, flozHundredths}} | null`); parsers in `api.ts` (the four-edit pattern; a malformed body → `error`); `readEntryForm` gains advanced reading: `sources_{name}` amount texts and the hidden `*_original` fields for bottles (sends `sources` only when a text or the unit changed; sends `volume` when the form is the simple one over a bottle that had sources; sends neither when nothing changed), and `weight_before`/`weight_after` as kg or lb+oz texts for nursing (sends a weight only when its text changed, `null` when cleared, fails closed on a missing original like every edit); `formatMilk(milk, units): string` ("weight gain 62 g, about 60 ml" for kg and ml users, "weight gain 2.19 oz, about 2.04 fl oz" for lb and fl oz users; the UI converts nothing, it formats the API's columns); `timelineLine` appends `, ` + `formatMilk` for a nursing item whose `milk` is non-null. The stub implements the new request and view fields with the API's validation order and the same 422 fields.

- [ ] **Step 1: Failing tests (RED):** `readEntryForm` per advanced case (every Review Focus 1–3 parse: valid sources in ml and fl oz, a blank source text for a ticked source → field `sources`, a source typed for an unticked one is ignored or refused as decided in the test, hostile text `1e3`, `0x10`, `--1`; edit sends nothing when untouched, sends `volume` when the simple form is saved over a bottle with sources; nursing weights kg `3.4` → 3400, lb 7 + oz 8.5 → `lb` 12050, a blank clears with `null` only when the original was non-empty); `formatMilk` (both unit preferences, zero estimate "weight gain 0 g, about 0 ml"); `timelineLine` for a weighed and an unweighed session; `api.test.ts` (list/create/update with the new fields, a malformed `sources` body → `error`); the stub's self-test spec.
- [ ] **Step 2:** `cd ui && pnpm test` → FAIL. **Step 3: Implement.** **Step 4:** `pnpm check && pnpm lint && pnpm test && pnpm test:e2e --trace off` PASS. **Step 5: Commit** `feat(ui): advanced bottle and nursing client, forms and formatters`.

## Task 5: UI — the advanced bottle form

**Files:** Modify `ui/src/lib/components/entry/BottleFields.svelte`, `ui/src/lib/components/entry/TrackerFields.svelte`, `ui/src/lib/components/entry/TrackerForm.svelte` (the toggle link), the log and edit `+page.server.ts` and `+page.svelte` under `ui/src/routes/(app)/kids/[kid]/log/[tracker=tracker]/` (+ `[entry]/`), `ui/src/lib/entry-edit.ts`; create `ui/src/lib/components/entry/SourceVolumes.svelte`.

**Interfaces:** the create page reads `?advanced=1`; the edit page is advanced when `?advanced=1` or the bottle has sources, and `?advanced=0` opens the simple form over a bottle with sources (with the sentence "Saving one total replaces the volumes per source"). Advanced layout: Time; one **Volume unit** select; the four contents checkboxes each followed by its own volume input (labelled "Breast milk volume", "Formula volume", "Cow's milk volume", "Other volume"; an input is required when its box is ticked, ignored when not, with CSS `:has()` showing the input beside a ticked box so it works without JavaScript), a read-only line "Total: N ml" rendered by the server from the typed texts on a failed submit and from the stored total on edit; Note. The toggle is a link at the top of the form: "Advanced: volume per source" / "Use one total". Failed validation keeps every typed value and marks the field `aria-invalid` with `role="alert"`.

- [ ] **Step 1: Failing tests (Playwright, both projects, axe):** `trackers.spec.ts` additions — the simple bottle form is unchanged and has the toggle link; `?advanced=1` shows four volume inputs next to the four boxes; create a breast milk 60 + formula 90 bottle, see `Bottle 150 ml, breast milk + formula` on the timeline and the per-source lines on the edit page; edit one source and save, the other keeps its value; an imperial parent types fl oz per source and reads fl oz back exactly; a ticked source with a blank volume shows the field message with values kept; "Use one total" over an advanced bottle saves a plain bottle and the sources are gone; a bottle created simple opens simple; no horizontal overflow at 390 px with a 200-character note; no-JS create and edit.
- [ ] **Step 2:** run → FAIL. **Step 3: Implement.** **Step 4:** the full UI pipeline PASS. **Step 5: Commit** `feat(ui): advanced bottle form with a volume per source`.

## Task 6: UI — nursing weigh-ins and the estimate

**Files:** Modify `ui/src/lib/components/entry/NursingFields.svelte`, the nursing log and edit pages (shared with Task 5's routes), `ui/src/lib/components/entry/TrackerForm.svelte` if the toggle needs a label; create `ui/src/lib/components/entry/WeightField.svelte`.

**Interfaces:** `WeightField` is the growth form's weight control (kg input, or pounds and ounces inputs, shown by CSS `:has()` from a unit select defaulting to the parent's preference; comma or point as decimal separator) parametrised by a name prefix (`weight_before`, `weight_after`), with its hidden originals on edit; the growth form is **not** changed (note the duplication in the PR). Advanced layout: the nursing form plus "Weight before" and "Weight after" (each optional, with the hint "In your weight unit. These weights stay with this session; they are not added to growth."), and on the edit page a paragraph under them: "Weight gain 62 g, about 60 ml (estimate at 1.03 g per ml)" when `milk` is non-null, "No estimate" when both weights are present and `milk` is null, nothing when a weight is missing. Toggle link "Advanced: weigh before and after"; the edit page opens advanced when the session has a weight. The home's End button and running card are unchanged.

- [ ] **Step 1: Failing tests (Playwright, both projects, axe):** create a finished session with before 3.4 kg and after 3.46 kg → timeline `… weight gain 60 g, about 58 ml` and the edit page shows the same sentence; an imperial parent types lb + oz and reads lb + oz back exactly and sees ounces and fl oz in the estimate; after lower than before saves, shows "No estimate" and no judgement wording; only a before weight shows no estimate paragraph; clearing a weight removes it; start a running session with a before weight, end it from the home, edit and add the after weight; a weight outside the range shows the field message with values kept; a weighed session adds nothing to the Growth page; caregiver can record weigh-ins; axe, 390 px no overflow, no-JS create and edit.
- [ ] **Step 2:** run → FAIL. **Step 3: Implement.** **Step 4:** the full UI pipeline PASS. **Step 5: Commit** `feat(ui): nursing weigh-ins and the milk estimate`.

## Task 7: Pumping with a volume per side (API)

**Files:** Modify `api/internal/store/migrations/0005_advanced_logging.sql` (still unreleased: add the table here, not in a new migration), `api/internal/store/store_test.go`, `api/internal/model/tracker.go`, `api/internal/store/pumping.go`, `api/internal/store/timeline.go` (pumping items), `api/internal/httpapi/pumping.go` (+ tests).

**Interfaces — Produces:** `model.PumpingSide{Side string; Volume units.Volume}`, `PumpingSession.Sides []PumpingSide` (nil or empty = simple, else one per side in left, right order); table `pumping_session_sides` (Open value 19) with `ON DELETE CASCADE` to `pumping_sessions`; store functions keep their signatures and replace the sides in the same transaction as the session row; `Pumping`/`ListPumping`/the timeline load sides with one extra query per page; the store refuses (`ErrInvalidSides`, mapped like bottles' contents error) a `Sides` set whose derived side differs from `Side`. Handler: `pumpingInput.Sides *sidesInput`, order `started_at`, `duration_seconds`, `volume`/`sides`, `side`, `note`; the rules of Open values 17–20 (the total is `units.NewVolume(unit, Σ typed amounts)`, each side row has its own `NewVolume`; an unchanged per-side typed amount in the same unit keeps the stored columns).

- [ ] **Step 1: Failing tests.** Store: `wantTables` gains `pumping_session_sides`; cascade with the kid; `TestSideChecks` (bad side name, duplicate (session, side), bad typed unit); round trip, replace on update, derived-side mismatch refused, list and timeline load sides with one extra query (the Task 2 counting driver). HTTP: `TestAdvancedPumpingCreate` (left 60 + right 50 ml → total 110 ml, side `both`, sides in order), left only → side `left`, ounces typed exact, `TestSumIsTheTotal` (create, PATCH one side, PATCH note only leaves every column untouched), `TestSwitchToSimpleAndBack`, both `volume` and `sides` → 422 `sides`, `side` that disagrees with the derived side → 422 `side`, validation (unknown key, empty object, zero, negative, huge, unknown unit, total over 2000 ml → 422 `volume`), a simple session's view has `sides: []`, roles and 404 identity unchanged, no values in logs.
- [ ] **Step 2:** `cd api && go test ./internal/store/ ./internal/httpapi/ -race` → FAIL. **Step 3: Implement.** **Step 4:** the full API pipeline from CLAUDE.md PASS. **Step 5: Commit** `feat(api): pumping with a volume per side`.

## Task 8: UI — the advanced pumping form

**Files:** Modify `ui/src/lib/types.ts`, `ui/src/lib/server/api.ts` (+ test), `ui/src/lib/server/entry-form.ts` (+ test), `ui/src/lib/entry-edit.ts`, `ui/src/lib/timeline.ts` (+ test), `ui/e2e/stub-api.ts`, `ui/src/lib/components/entry/PumpingFields.svelte`, the pumping log and edit pages; create `ui/src/lib/components/entry/SideVolumes.svelte` (model it on Task 5's `SourceVolumes.svelte`).

**Interfaces:** types `PumpingSide`, `Pumping.sides`; `readEntryForm` reads `sides_left`, `sides_right` and the hidden originals (sends `sides` only when a text or the unit changed, `volume` + `side` when the simple form is saved over a session that had sides, nothing when untouched; a blank side is absent, both blank is the `sides` field error; hostile text refused); `timelineLine` appends ` (L 60 ml · R 50 ml)` in the parent's volume unit when `sides` is non-empty (`(left)` / `(right)` / `(both)` as before otherwise); the stub mirrors the API. Advanced layout: Time; Minutes; one **Volume unit** select; "Left volume" and "Right volume" inputs (at least one); a read-only "Total: N ml" line (server-rendered from the typed texts on a failed submit, from the stored total on edit); the `side` radio is not shown (derived); Note. Toggle link "Advanced: volume per side" / "Use one total" (`?advanced=1` / `?advanced=0`, a plain link; an edit page opens advanced on its own when the session has sides; saving the simple form over such a session replaces the sides, and the form says so).

- [ ] **Step 1: Failing tests (RED then Playwright, both projects, axe):** unit tests for the reader and formatter (valid left+right, left only, imperial fl oz exact, both blank, hostile text, untouched edit sends nothing, simple-over-advanced sends `volume` and `side`); `trackers.spec.ts` additions — the simple pumping form is unchanged and has the toggle link; `?advanced=1` shows two inputs and no side radio; create left 60 + right 50 → `Pumping … 110 ml (L 60 ml · R 50 ml)` on the timeline; left only saves as left; editing one side keeps the other; an imperial parent reads fl oz back exactly; both blank shows the field message with values kept; "Use one total" saves a plain session; caregiver can record; no horizontal overflow at 390 px; no-JS create and edit. Every submit waits for its response (`submit()` helper).
- [ ] **Step 2:** `cd ui && pnpm test` → FAIL. **Step 3: Implement.** **Step 4:** `pnpm check && pnpm lint && pnpm test && pnpm test:e2e --trace off` PASS. **Step 5: Commit** `feat(ui): pumping with a volume per side`.

## Task 9: Smoke, docs and HANDOFF

**Files:** Modify `scripts/smoke.sh`, `README.md`, `docs/superpowers/HANDOFF.md`.

- [ ] **Step 1: Extend `scripts/smoke.sh`** (failing first against images from `main`): in the tracker step, log an advanced bottle (two sources, distinctive amounts), an advanced pumping session (two sides, distinctive amounts) and a weighed nursing session (distinctive weights) through the real UI forms, assert the timeline shows the per-source total and the weight gain with the estimate, edit one source and a weight through the real API (scrape the hidden originals as steps 4h and 4i do), switch the bottle to one total and assert the sources are gone, assert a weighed session adds nothing to the Growth page, extend step 9's log scan with the distinctive volumes and weights, keep step 10's ceilings (60 MiB API, 140 MiB UI; report the readings).
- [ ] **Step 2:** `docker build -t hbt-api:dev api && docker build -t hbt-ui:dev ui && scripts/smoke.sh hbt-api:dev hbt-ui:dev` → every line `PASS:`; `shellcheck scripts/smoke.sh`.
- [ ] **Step 3: Docs:** README (the three advanced modes, the 1.03 g/ml assumption stated as the app's own, weigh-ins are not growth); HANDOFF (plan 5.1 done, plan 6's migration becomes 0006, the owner decisions of this plan, the unverified items below). **Step 4:** the whole pipeline in CLAUDE.md (API, UI, images, smoke) green. **Step 5: Commit** `test: smoke covers advanced logging; docs: plan 5.1 done`.

---

## Done when

- API and UI pipelines and `scripts/smoke.sh` pass on the branch head and CI is green.
- A parent logs a bottle with a volume per source and a pumping session with a volume per side, and reads the total and the parts back exactly, in the unit typed; logs a nursing session with a before and an after weight and reads the weight gain and the estimated milk volume in their own units; every simple form and every old entry is unchanged.
- The final report states which image content changed (`api/` and `ui/` sources; no Dockerfile or dependency change expected), that the edge still needs its `bump-happybabytracker` PR merged and a deploy, that nothing on the edge changes, and that plan 6's migration number moves to 0006.

## Not verified (state in the PR)

- The 1.03 g/ml density is the owner's figure, used as the app's own constant; no source was checked and a test-weighing estimate is not a clinical measure.
- The owner's observation that weight loss after nursing is associated with latching problems is not encoded anywhere: the app shows no estimate in that case and says nothing about it.
