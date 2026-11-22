# Updating a vaccination schedule

A schedule is one CSV file, `api/internal/vaccines/data/<id>.csv`. It starts with a provenance header,
and its generated review document is `docs/vaccine-schedules/<id>.md`. An update is a pull request that
changes the CSV, the review document and the tests that pin the data. The owner reviews the review
document's diff in that pull request. Schedules are embedded in the API image, so an update is live only
after CI publishes the image, the edge's `bump-happybabytracker` PR changes the pin and the edge is
deployed.

## When to check each source

| Id | Source | Reissued (observed unless the source says so) | Where to look |
|---|---|---|---|
| `us` | CDC, Child and Adolescent Immunization Schedule by Age | Observed: a new edition each year, plus addenda during the year (the shipped 2025 edition has an addendum updated July 2, 2025) | https://www.cdc.gov/vaccines/hcp/imz-schedules/child-adolescent-age.html; notes: https://www.cdc.gov/vaccines/hcp/imz-schedules/child-adolescent-notes.html; PDF: https://www.cdc.gov/vaccines/hcp/imz-schedules/downloads/child/0-18yrs-child-combined-schedule.pdf; reuse terms: https://www.cdc.gov/other/agencymaterials.html |
| `uk` | UKHSA, Complete routine immunisation schedule | Observed: whenever a programme changes; it changed several times in 2025–2026. Check the page's "Updated" date | https://www.gov.uk/government/publications/the-complete-routine-immunisation-schedule; the cohort rules are in UKHSA's MMRV guidance and the childhood schedule changes guidance, both linked from the research note |
| `mx` | Secretaría de Salud, CeNSIA, Lineamientos Generales del Programa de Vacunación Universal | Observed: a new edition for each year (the 2026 edition is dated 31 December 2025), sometimes amended during the year | https://www.gob.mx/salud/censia/documentos/lineamientos-generales-pvu-2026 (the year is in the URL); cross-check the PRONAM “Vacunación a lo largo de la vida” |
| `ch` (not shipped) | BAG with EKIF, Schweizerischer Impfplan | Every year in the first quarter (the plan itself says so) | https://www.bag.admin.ch/impfplan. A future option only: shipping it needs the BAG's written consent in advance |

The research notes in `docs/superpowers/research/vaccination/` record what was read, the traps and the
licence findings for each source; read the country's note before an update.

## Steps

1. Read the new edition and list what changed: ages, new or removed doses, cohorts, kinds.
2. Edit the CSV.
   - **Provenance keys** (all required except `omitted`): `schema` (the id, equal to the file name),
     `name`, `authority`, `edition` (the edition as the source names it, with its revision date),
     `retrieved` (YYYY-MM-DD, the day you read it), `origin_url` (https), `licence`, `attribution`, and
     `omitted`. The `omitted` key is one line of `;`-separated items: what the source has that the file
     leaves out. Do not put a `;` inside an item. Keep the licence's required wording in `attribution`:
     the test `mandatoryAttribution` pins it.
   - **Columns**: `line,group,antigen,series,from,until,kind,born_from,born_to,note`.
     - `antigen` is an id from `antigens.csv`.
     - `series` counts from 1 per antigen and cohort, with no gaps.
     - `from` and `until` are ages written `<n><unit>`, with unit `d`, `w`, `m` or `y`.
     - `kind` is `routine`, `may_not_be_needed`, `supplementary` or `recurring`.
     - `born_from` and `born_to` are inclusive birth dates.
     - `note` quotes the source where the encoding needs a reason; start it with `(verify)` when the
       row could not be checked against the primary source.
   - **Window**: a source range "12–15 months" means through, so write `from 12m`, `until 16m`; `until`
     is exclusive. Leave `until` empty for a single age: the line then gets the default window of one
     calendar month (`constants.VaccineDefaultWindowMonths`).
   - **Line ids** are `<antigen>-<series>`. A line limited to a cohort adds `-bYYYYMM` (from its
     `born_from`) or `-toYYYYMM` (from its `born_to`) where two cohorts share a series number; a few
     older UK ids with `born_from` have no suffix and stay as they are.
3. Run `cd api && UPDATE_DOCS=1 go test ./internal/vaccines/ -race`, then read the diff of
   `docs/vaccine-schedules/<id>.md`. It should show exactly the changes listed in step 1. The loader
   names the file, line and column of every error.
4. Update the tests that pin real data (they fail on any data change, on purpose):
   - `api/internal/vaccines/data_test.go`:
     - the line tables `usLines`, `ukLines` and `mxLines`, and `mandatoryAttribution`;
     - the visit and age tests (`TestUSEvaluatesTwoMonthVisit`, `TestUSAtFifteenYears`,
       `TestUKEvaluatesEightWeekVisit`, `TestUKAtFifteenYears`, `TestMXEvaluatesTwoMonthVisit`,
       `TestMXAtFifteenYears`, `TestMXHepatitisBNeverOverdue`);
     - the cohort boundary tests (`TestUKCohortBoundaries`, `TestMXCohortBoundaries`) and the
       brand-name tests.
   - `api/internal/vaccines/load_test.go`: `TestSchemasInShippedOrder`.
   - `api/internal/httpapi/vaccines_test.go` and `home_test.go`: they use real line ids (for example
     `hepatitis_b-1`, `rotavirus-1`), group counts and dismissed counts (`TestDismissBefore`,
     `TestSwitchSchema`, `TestDismissalOnlyForLinesTheKidSees`, `TestHomeDue` and others).
   - `scripts/smoke.sh` step 4i:
     - the US and UK captions, the CDC attribution and the omitted item "Tdap: 1 dose during each
       pregnancy";
     - the 6-in-1 lines `diphtheria-1 … hepatitis_b-1`;
     - the birth group overdue ("Birth · Overdue (1)"), the window text of `hepatitis_b-1` and the one
       line that dismiss-before dismisses. That count of 1 relies on `rsv-1` being Available (past its
       8-month window) and `hepatitis_b-1` being given by the 6-in-1; a data edit that changes either
       changes the count;
     - the settings options ` us uk mx `.
   - The UI tests run against made-up stub schedules (`ui/e2e/stub-api.ts`) and need no change.
5. Run the whole pipeline in `CLAUDE.md` (API, UI, images and smoke) and open the pull request (see
   "Pull request checklist").

## What happens to stored data

- **Doses never depend on a schedule.** They are stored per kid and antigen. An update changes only
  which line a dose fills: the kid's Nth dose of an antigen fills the Nth line. Extra doses list under
  "Other recorded vaccinations".
- **Dismissals** are stored per kid, schedule id and line id. A dismissal whose line was removed or
  renamed is ignored, never an error. So keep line ids stable: rename a line only when its meaning
  changed, and expect its dismissals to drop.
- **A removed schedule id**: a kid who keeps it shows "Unavailable schedule (<id>)" in the settings
  select, and saving the settings keeps it. The Vaccines page says no schedule is chosen and lists every
  dose under "Other recorded vaccinations". The home shows no Due list. Kid edits keep working. A
  guardian picks another schedule to see lines again.

## Traps

- **Count matching and an earlier optional dose.** A child who skips an optional or product-dependent
  dose has every later dose counted one line early. That can flag a later line overdue when nothing is
  missing. Mexico's hepatitis B (no birth dose for some children) is why every Mexican hepatitis B line
  is `may_not_be_needed` (ruling R17). Re-check this for any new optional first dose. The US birth dose
  of hepatitis B and the UK schedule were not re-checked for it.
- **A 4-dose combination series consumes a later line.** In the US, polio has 4 lines (2m, 4m, 6m, 4y).
  A combination product can give a 4th IPV dose before age 4 (for example at 15 months). By count, that
  dose fills `polio-4`, although CDC still wants a dose at 4 years or later. The row's note says so.
- **Hib in the US (ruling R13).** `hib-3` is `routine` from 6m until 16m: one window for the 6-month
  dose and the PedvaxHIB booster. `hib-4` is `may_not_be_needed` from 12m to 16m, because the number of
  doses depends on the product.
- **Mixed units.** A line whose `from` and `until` use different units must have a gap of at least 4
  days (`VaccineMinMixedUnitGapDays`; ruling R8), or the loader refuses it. For example, `from 52w`
  `until 1y` is refused; write both ages in one unit.
- **The default window** is one calendar month. A single age that the source means as a longer window
  needs an explicit `until`; for example Mexico's BCG is `until 12m` (ruling R18).
- **`may_not_be_needed`** lines are due only inside their window. After it they are `available`: never
  overdue and never on the home (ruling R15).
- **`supplementary` and `recurring`** lines are never due or overdue. Only one recurring line per
  antigen is allowed.
- **`(verify)` rows** need the owner's check against the primary source:
  - US: `rsv-1`, `hib-3`, `hib-4`, `hepatitis_a-2`, `hpv-2`;
  - UK: `influenza-1`;
  - Mexico: `influenza-1`, `covid19-1`, `hpv-1`.

  Mexico's Anexo 1 (minimum and maximum ages and intervals) is a low-resolution scan with an illegible
  digit. None of its ages is used, and it is listed under `omitted`.
- **Cohort boundaries.** The loader cannot catch a typo in `born_from` or `born_to` that leaves a hole
  or an overlap between cohorts. `TestUKCohortBoundaries` and `TestMXCohortBoundaries` check the day
  before and the day of each boundary; extend them for a new cohort.
- **Editions.** CDC pages still serve the 2025 edition. A January 2026 revision is reported, by
  secondary sources only, to be stayed by a court. UKHSA's Green Book chapter 11 predates its September
  2026 table. The Swiss edition is reissued each first quarter even without changes.

## Adding a country

1. Add its id to `constants.VaccineSchedules` in `api/internal/constants/constants.go`; the order there
   is the order in the settings select.
2. Add `api/internal/vaccines/data/<id>.csv`. The loader refuses a schedule file whose id is not listed.
3. Add its tests to `data_test.go`: a full line table, a visit, an age-15 check, cohort boundaries,
   brand names and its `mandatoryAttribution`. Update `TestSchemasInShippedOrder`.
4. Generate `docs/vaccine-schedules/<id>.md` (step 3 above) and add its row to
   `docs/vaccine-schedules/README.md`.
5. Add rows to the README: the "Vaccination" table, the licence list, and the "Third-party data" table.
   If the licence needs a notice file in the image, copy it in `api/Dockerfile` as the growth notices are.
6. Update `scripts/smoke.sh` step 4i's expected settings options.
7. Settle the licence with the owner first. A schedule ships under an open licence, with written
   permission, or (as for Mexico) on the owner's decision to ship a facts-only transcription with the
   required citation. Switzerland waits for the BAG's written consent.

## Pull request checklist

- [ ] The CSV's `edition`, `retrieved` and `origin_url` are new, and its `omitted` list is current.
- [ ] `docs/vaccine-schedules/<id>.md` was regenerated. Its diff shows only the intended changes, and
      the owner has reviewed it.
- [ ] Every new `(verify)` note is listed in the PR description.
- [ ] The pinned data tests and smoke step 4i were updated.
- [ ] The whole pipeline passed (API, UI, images and `scripts/smoke.sh`), with its output in the PR.
- [ ] The README rows, and the index in `docs/vaccine-schedules/README.md`, show the new edition.
- [ ] After the merge: the edge's `bump-happybabytracker` PR and a deploy.
