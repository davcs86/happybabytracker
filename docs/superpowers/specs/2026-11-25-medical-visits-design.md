# Medical visits: design

Date: 2026-11-25. Status: approved by the owner on 2026-11-25 with every **(call)** as written; plan 9
(`docs/superpowers/plans/2026-11-25-visits.md`) records the rulings taken since. Extends
`docs/superpowers/specs/2026-11-20-happybabytracker-design.md` ("the main spec"); section numbers
below without a prefix are this document's, "main N" is the main spec's.

Marks: **(owner)** is a decision the owner made in the design conversation; **(call)** is the
author's choice, open to correction.

## 1. Intent

Parents already record growth, vaccines and medication, but the event that produces most of that data,
a visit to a doctor, has no place in the app. Today a parent comes home from a well-child visit and
types the weight on one screen, each vaccine on another and the prescription on a third, and nothing
says they belong together. Upcoming appointments live only in someone's calendar.

This module gives a visit a record of its own: scheduled ahead or logged afterwards, editable, with
post-visit notes, and with a single "after the visit" form that writes the measurement, the vaccines
and the prescriptions into their own modules in one step. A scheduled visit can be put in the
family's calendar as an `.ics` file.

## 2. Goals, non-goals, anti-goals

Goals

- V1. Guardians and caregivers create, edit, cancel and delete visits for a kid, past or scheduled
  **(owner)**.
- V2. Post-visit notes on a visit **(owner)**.
- V3. One form records the visit's outcome and creates the Growth, Vaccination and Medication rows
  atomically, linked to the visit **(owner: approach A)**.
- V4. A visit's `.ics` file, with a stable UID, in a generic or a detailed form chosen at download
  **(owner)**.
- V5. Visits appear in the Home "due" list, the Timeline and the PDF export **(owner)**.
- V6. `visits` is a module: on by default at every age, switchable by a guardian **(owner)**.
- V7. Optional provider lookup against the US NPI registry, behind a flag, never required to save
  **(owner: approach A)**.

Non-goals

- Push or ntfy reminders for visits (main 2 keeps reminders to medication). The `.ics` alarm is the
  only reminder.
- A subscribable calendar feed; only single-event download **(owner)**.
- Recurring visits, attachments, linking or unlinking an existing record to a visit, a provider
  address book.
- Two-way sync: after the cascade, a linked row is edited only in its own module.
- Provider registries other than NPPES.

Anti-goals (a failure even if it works)

- Two copies of a weight, a dose or a prescription that can drift apart.
- A half-applied outcome: the note saved but a vaccine missing, or the reverse.
- A double-submitted outcome form that records every vaccine twice.
- A visit that cannot be saved because a third-party registry is slow or down.
- A provider name, a note, a location or a search term in a log line, an error string or a URL.
- A linked row deleted with its visit without the person having been told it would be.

## 3. Constraints

- The CLAUDE.md rules apply unchanged: 404 for no grant, byte-identical to an unknown kid; the admin
  is refused; writes re-assert the role with `actorRole` inside the store transaction; hard deletes;
  instants are unix seconds UTC in INTEGER columns and RFC3339 UTC on the wire; limits only in the
  two constants files; no PII in logs or errors; comments state constraints only.
- Every new page works without JavaScript and is sent `Cache-Control: no-store`.
- No new dependency in `api/` or `ui/` **(call)**: the `.ics` writer and the NPPES client are small
  enough to write against the standard library.
- Contract with `davcs86/homelab-edge` (CLAUDE.md rule 1): sections 4–9 change nothing in it.
  Section 10 adds one API env var and one outbound host; it ships as its own plan and the edge change
  is the owner's **(owner)**.

## 4. Approaches

### 4.1 Getting visit data into Growth, Vaccination and Medication

| | What it is | Trade-off |
|---|---|---|
| A. Linked rows, one transaction **(owner: chosen)** | The outcome form embeds the three modules' own field groups. Saving writes real rows in `growth_measurements`, `vaccine_doses` and `medications` in the same SQLite transaction as the visit's notes, each stamped with `visit_id`. | One source of truth, atomic, and the visit can show what it produced. Costs a tx-accepting variant of three store functions and a nullable column on three tables. |
| B. Copies on the visit | The visit stores its own weight, vaccine and prescription fields; a button copies them out. | Simplest API, but two sources of truth; an edit on either side silently diverges. Hits the first anti-goal. |
| C. Prefilled links | The visit page links to each module's "new" form with the date filled in. | Cheapest; not atomic, three round trips, nothing ties the rows to the visit. |

A is possible because the whole dataset is one SQLite database behind one process (main 6.1).

### 4.2 Provider lookup

| | What it is | Trade-off |
|---|---|---|
| A. API-side proxy, off by default **(owner: chosen)** | A parent-only API route calls NPPES. Short timeout, per-parent rate limit, small cache, bounded concurrency. Any failure degrades to free text. | Keeps outbound traffic where Web Push and ntfy already are, inside the access matrix. Costs an env var and an egress rule on the edge. |
| B. The UI server calls NPPES | No API change. | Outbound logic in the container main 13.2 says holds no data; outside the access-matrix tests. Rejected. |
| C. The browser calls NPPES | No server change. | Sends each parent's IP and search terms to a third party. Rejected. |
| D. An NPI number field only | No live search. | No contract change, little value. Rejected. |

## 5. The visit

### 5.1 Data

Migration `0008_visits.sql`.

`visits`, with the common entry columns of main 6.1 (`id`, `kid_id`, `note`, `created_by`,
`updated_by`, `created_at`, `updated_at`) plus:

| Column | Type | Rule |
|---|---|---|
| `starts_at` | INTEGER | unix seconds UTC |
| `duration_min` | INTEGER | default 30, 5 to 720 |
| `kind` | TEXT | one of `well_child`, `sick`, `specialist`, `dental`, `emergency`, `other` **(owner)** |
| `provider` | TEXT, nullable | doctor or clinic, at most 120 characters |
| `location` | TEXT, nullable | at most 200 characters |
| `outcome_notes` | TEXT, nullable | post-visit notes, at most 4000 characters |
| `cancelled_at` | INTEGER, nullable | set while the visit is cancelled |
| `revision` | INTEGER | starts at 0; every write to the row adds 1 |

Indexed on `(kid_id, starts_at)`. At most 2000 visits per kid; the next create answers 409 `limit`,
as Growth does.

The visit's "reason" is the common `note` column: wire key `note`, screen label "Reason", the shared
500-character limit **(call: one free-text field before the visit and one after, rather than three)**.

The same migration adds `visit_id TEXT REFERENCES visits (id) ON DELETE SET NULL`, nullable, to
`growth_measurements`, `vaccine_doses` and `medications`, each with a partial index
`WHERE visit_id IS NOT NULL`. All rows of one vaccine administration carry the same `visit_id`.

`kid_modules.module` has a `CHECK (module IN (...))` list (migration 0002), and SQLite cannot alter a
CHECK, so the migration rebuilds `kid_modules` with `visits` added to the list and copies the rows.

Limits (all new, in `constants.go`, client-safe ones mirrored): default duration 30, duration 5–720,
provider 120, location 200, outcome notes 4000, visits per kid 2000, scheduling horizon 731 days,
due-list look-ahead 7 days, outcome vaccinations per save 10, outcome medications per save 5.

### 5.2 Status

Status is derived, never stored **(owner)**: `cancelled` when `cancelled_at` is set; otherwise
`upcoming` while now is before `starts_at`; otherwise `past`. The API computes it at read time and
sends it as `status`.

### 5.3 Rules

- `starts_at` is from the start of the kid's birth date (kid's zone) to 731 days ahead of now. An
  unchanged stored value stays valid on edit, as the other trackers treat dates.
- A visit **has an outcome** when `outcome_notes` is set or any row links to it.
- `outcome_notes` can be set only on a `past` visit. A visit that has an outcome cannot be cancelled
  and cannot have `starts_at` moved into the future: both answer 409 `has_outcome` **(call)**. The
  person removes the outcome first, or deletes the visit.
- Any visit without an outcome can be cancelled, upcoming or past (a past one that never happened),
  and a cancelled visit can be un-cancelled or deleted **(owner)**. A cancelled visit stays in the
  list, marked.
- Guardians and caregivers have the same rights on every visit route **(owner)**.
- Turning the module off hides the entry points and the due entries; the data and the routes stay, as
  for every module (main 7.2).

### 5.4 Routes

All `Caregiver` class in `routes.go`, pinned in `routeClasses` and `matrixParams`.

| Route | Does |
|---|---|
| `GET /v1/kids/{kid}/visits` | lists by `starts_at` with the trackers' `from`/`to`/cursor paging; unlike the trackers, `to` may lie in the future, up to the scheduling horizon (its default), and `order=asc` lists soonest first (plan 9 ruling B1; default `desc`) |
| `POST /v1/kids/{kid}/visits` | creates |
| `GET /v1/kids/{kid}/visits/{entry}` | one visit, with its linked rows (5.5) |
| `PATCH /v1/kids/{kid}/visits/{entry}` | edits; an absent key keeps the stored value, `null` clears an optional one |
| `DELETE /v1/kids/{kid}/visits/{entry}` | deletes; `?linked=delete` (default) or `keep` (7) |
| `POST /v1/kids/{kid}/visits/{entry}/outcome` | records the outcome (6) |
| `GET /v1/kids/{kid}/visits/{entry}/calendar` | the `.ics` file; `?detail=generic` (default) or `full` (8) |

PATCH keys: `starts_at`, `duration_min`, `kind`, `provider`, `location`, `note`, `outcome_notes`,
`cancelled` (boolean; `true` stamps `cancelled_at`, `false` clears it). Unknown keys are refused, as
elsewhere. Validation names the first invalid field in the order `starts_at`, `duration_min`, `kind`,
`provider`, `location`, `note`, `outcome_notes`.

### 5.5 Wire shape

```json
{
  "id": "…", "starts_at": "2026-12-03T09:30:00Z", "duration_min": 30,
  "kind": "well_child", "status": "upcoming", "cancelled_at": null,
  "provider": "Dr. Lee", "location": "12 Main St", "note": "4-month check",
  "outcome_notes": null, "revision": 3,
  "linked": {
    "growth": [{ "id": "…", "measured_on": "2026-12-03" }],
    "vaccinations": [{ "id": "…", "given_on": "2026-12-03", "antigens": ["dtap", "ipv"] }],
    "medications": [{ "id": "…", "name": "Amoxicillin", "dose_count": 14 }]
  }
}
```

The list sends the same object with `linked` reduced to three counts. The existing Growth,
Vaccination and Medication views each gain `visit_id` (nullable) **(call)**, so their own pages can
say "Recorded at a visit" with a link; without it, a person editing a weight has no way to know that
deleting the visit would take it too.

## 6. Recording the outcome

`POST …/visits/{entry}/outcome` with:

```json
{
  "revision": 3,
  "outcome_notes": "Healthy. Next check at 6 months.",
  "growth": { …the body of POST /growth… },
  "vaccinations": [{ …the body of POST /vaccinations… }],
  "medications": [{ …the body of POST /medications… }]
}
```

- `revision` is required. Every other key is optional; a body that sets none of them is `invalid`.
- `growth`, each element of `vaccinations` and each element of `medications` is exactly the body the
  module's own POST takes, validated by the module's own validator, so the cascade can never accept
  what the module would refuse. An absent `measured_on` or `given_on` defaults to the visit's date in
  the kid's zone; an absent medication `start_at` defaults to the visit's `starts_at` **(call)**.
- A validation failure names the field as `growth.weight`, `vaccinations[1].lot`,
  `medications[0].schedule`.
- The visit must be `past`; otherwise 409 `not_past`.
- **Atomic.** One transaction: re-assert the role; compare `revision` (mismatch: 409 `stale`); insert
  every row with `visit_id`; set `outcome_notes` when the key is present; add 1 to `revision`. A
  per-kid cap hit by any row (409 `limit`) rolls the whole call back. The medication rows materialise
  their reminders inside the same transaction, as the module's own create does.
- **Repeatable and safe to retry.** Each call adds rows; it never replaces earlier ones, which are
  edited in their own modules. Because every call bumps `revision`, the second of two identical
  submissions answers 409 `stale` and writes nothing **(call: optimistic check on this route only;
  the other visit routes are last-write-wins like every tracker)**.
- The response is the visit with its linked rows.

Store work: `CreateGrowth`, `CreateAdministration` and `CreateMedication` each open their own
transaction today. Each is split into a function that takes an open `*sql.Tx` and an optional visit
id, and the existing public function that wraps it; behaviour of the three existing routes is
unchanged. `store.ErrInvalidAntigens`, `ErrInvalidSchedule` and `ErrConflict` keep their meanings.

When the vaccine schedules failed to load, a call that carries `vaccinations` answers the 503 the
vaccination POST answers; a call without them is unaffected.

## 7. Deleting a visit

`DELETE …/visits/{entry}?linked=delete|keep`, default `delete` **(owner: cascade by default, with a
warning in the confirm dialog)**.

- `delete`: in one transaction, the visit's growth rows, vaccine doses and medications are deleted
  with the semantics of each module's own delete (a medication takes its doses and reminders with
  it), then the visit.
- `keep`: only the visit is deleted; `ON DELETE SET NULL` clears the links.

The confirm step names what is linked and what will happen, using the counts from 5.5:

```
Delete this visit?

This visit recorded:
  • 1 growth measurement
  • 2 vaccines
  • 1 medication, with its 14 logged doses

[x] Also delete these records

            [ Cancel ]   [ Delete visit ]
```

The checkbox is ticked by default and absent when nothing is linked. A row deleted in its own module
simply leaves the visit; a row edited there stays linked. That includes a vaccine administration whose
antigens change: `UpdateAdministration` deletes and inserts dose rows, and the inserted ones must
carry the administration's `visit_id`.

## 8. Calendar file

A new package `api/internal/ics` writes one RFC 5545 `VCALENDAR` holding one `VEVENT`; the UI passes
it through unread, as it does the PDF.

- `UID:visit-<visit id>@<host of PUBLIC_ORIGIN>`: never changes for a visit **(owner: required)**.
- `SEQUENCE:<revision>`, `DTSTAMP` and `LAST-MODIFIED` from `updated_at`, so re-importing after an
  edit updates the event instead of duplicating it.
- `DTSTART` and `DTEND` as UTC instants (`…Z`): no `VTIMEZONE` block, and correct across a
  daylight-saving change because the stored value is an instant.
- `STATUS:CONFIRMED`, or `STATUS:CANCELLED` for a cancelled visit, with the same UID.
- `URL:` the visit's page under `PUBLIC_ORIGIN` (ids only).
- One `VALARM`, display, one day before, on a visit that is not cancelled **(call; the owner rated an
  alarm nice-to-have)**.
- `METHOD:PUBLISH`, CRLF line ends, 75-octet folding, TEXT escaping.

| | `detail=generic` | `detail=full` **(owner: the person chooses)** |
|---|---|---|
| `SUMMARY` | `Medical visit` | `<kid's name>: <kind label>`, plus ` with <provider>` when set |
| `LOCATION` | absent | the location |
| `DESCRIPTION` | absent | the reason (`note`) |

`outcome_notes` never enters the file. Response: `text/calendar; charset=utf-8`,
`Content-Disposition: attachment; filename="visit-<kid-local date>.ics"`, no name in the filename.
The API serves the file for any visit; the UI offers it on upcoming and on cancelled visits.

Not verified from here: how Google Calendar and Apple Calendar treat a re-import (update in place,
honour `STATUS:CANCELLED`). The UID and SEQUENCE are what the standard provides; the plan includes a
manual check on a phone, and the wording on the screen does not promise more than "Add to calendar".

## 9. Where visits show up

- **Module.** `visits` is appended to `model.Modules` and is on by default in all three age bands
  (main 7.2's table gains a row). It gets a row on the modules page.
- **Navigation.** A "Visits" entry in the More sheet, hidden when the module is off.
- **Home.** The `due` list gains kind `visit`: `{ "kind": "visit", "id", "starts_at", "visit_kind" }`
  for each visit that is `upcoming` and starts before the end of the seventh kid-local day from today
  **(call)**. No provider or reason in it. Nothing for past visits.
- **Timeline.** A new timeline type `visit`, keyed on `starts_at`, mapped to the `visits` module. A
  cancelled visit shows as cancelled. The row opens the visit.
- **PDF export.** `visits` joins `ExportModules`. A "Medical visits" section lists the visits whose
  kid-local date is in the range: date and time, kind, provider, location, reason, post-visit notes,
  and "cancelled" where it applies. The linked rows are not repeated there; they are in their own
  sections. The existing row caps and font-substitution machinery apply.

### 9.1 Screens

`/kids/{kid}/visits`: two groups, Upcoming (soonest first) and Past (newest first), cancelled ones
marked; a "New visit" action.

`/kids/{kid}/visits/new` and the edit form: date and time in the kid's zone (the existing entry time
field), duration, kind, provider, location, reason.

`/kids/{kid}/visits/{visit}`:

```
Well-child visit                      upcoming
Thu 3 Dec 2026, 10:30 · 30 min
Dr. Lee · 12 Main St
Reason: 4-month check

Add to calendar:  ( ) Without details  (•) With details   [ Download .ics ]

[ Edit ]  [ Cancel visit ]  [ Delete ]
```

Once the visit is past, "After the visit" is its own sheet page, `/kids/{kid}/visits/{visit}/outcome`, opened
from the visit page (owner ruling of 2026-11-25, AGENTS.md rule 1); the visit page lists what it recorded:

```
After the visit
Notes   [ Healthy. Next check at 6 months.            ]

[x] Add a growth measurement      (the Growth form's fields; date prefilled)
[x] Add vaccines given            (the Vaccination form's fields; "Add another")
[ ] Add a medication              (the Medication form's fields; "Add another")

                                   [ Save ]

Recorded at this visit
  Growth       3 Dec 2026 · 6.4 kg, 63 cm        →
  Vaccines     DTaP, IPV (Pediarix)              →
```

A section whose module is off for the kid is not offered. The three field groups are the existing
components (`growth/MeasurementFields`, `vaccines/DoseFields`, `medication/CourseFields` and
`ScheduleFields`), not copies. Without JavaScript the form offers one vaccine block and one
medication block per save, and saving again adds more; with JavaScript, "Add another" adds blocks up
to the caps of 5.1.

A 409 `stale` re-renders the page with the current visit and the message that it changed, without
losing what was typed.

## 10. Provider lookup (NPI), a separate plan

**Status: built in plan 10** (`docs/superpowers/plans/2026-11-26-provider-lookup.md`, branch `ccr-406c8e35-e8hvag`,
PR #25); off until the owner's edge change sets `NPI_SEARCH=1`. Where the plan settles a point below:

- **Filter (owner ruling 1).** NPPES's `state` and `city` match any address of a record, so the API keeps only
  records whose practice-location (`LOCATION`) address matches the state, and the city and the first 5 digits of
  the postal code when given. It still asks for 20, so fewer may show.
- **`more` (B1).** The response is `{ "results": [...], "more": bool }`; `more` is true when NPPES sent its full
  20. NPPES's `result_count` is never shown.
- **Outcomes (B2, B8).** A `{"Errors":[…]}` body (even at HTTP 200), a 3xx, an oversized or unparseable body and
  the in-flight cap (which never waits) all answer 503 `unavailable` and log `upstream_error`; a timeout logs
  `timeout`. The request's one log line carries `npi_outcome` and `npi_results`.
- **`provider_npi` (B7).** The API enforces it: a POST or PATCH that changes `provider` without sending
  `provider_npi` clears it, and a merged result with `provider_npi` but no `provider` is 422 `provider_npi`.
- **Registry text (owner ruling 4).** A picked name and address are kept as NPPES sends them, in capitals.

NPPES (`https://npiregistry.cms.hhs.gov/api/?version=2.1`) is public, needs no key, and covers US
providers only. It was queried from the build environment on 2026-11-25 and answered. It has no SLA.

- **Flag.** New API env var `NPI_SEARCH` **(owner)**: `1` turns the lookup on, unset leaves it off,
  any other value is refused at start. Main 13.1's table gains the row; the edge needs the variable
  and outbound HTTPS to `npiregistry.cms.hhs.gov` **(owner makes the edge change)**.
- **Routes.** Both `Parent` class.
  - `GET /v1/providers` → `{ "search": { "enabled": true } }`, so the UI needs no env var of its own.
  - `POST /v1/providers/search`. The terms travel in the JSON body, never in a URL, so no access log
    on any hop can hold them. Body: `kind` (`person` or `organization`), `last_name` and optional
    `first_name` (person) or `organization`, `state` (required, two letters), optional `city` and
    `postal_code`. Names are 2 to 60 letters, spaces, hyphens, apostrophes and periods; an organization name may also hold digits
    and `&` (owner, 2026-11-26). While the
    flag is off the route is not registered, as `/v1/_test/pushes` is not without its flag, so it
    answers what an unknown route answers.
- **Upstream call.** Package `api/internal/npi`. The host is a code constant, not configuration, so
  there is nothing to point elsewhere. Trailing-wildcard match on names, `limit=20`. Client timeout
  4 seconds, redirects not followed, response read up to 1 MiB, at most 4 calls in flight.
- **Result.** At most 20 of `{ npi, name, credential, taxonomy, address, phone }`, taken from each
  record's practice-location address and primary taxonomy.
- **Protection.** Per-parent limit of 20 searches a minute with the existing `auth.Limiter` (429
  `rate_limited`). An in-memory cache of up to 256 normalised queries for 10 minutes; errors are not
  cached. The API runs in 64 MiB in smoke, which is why the response and the cache are bounded.
- **Failure.** Timeout, non-200, unparseable body, the in-flight cap: all answer 503 `unavailable`.
  The form says the search is unavailable and the text fields keep working. Saving a visit never
  calls NPPES.
- **Storage.** Migration `0009` adds `visits.provider_npi` (nullable, ten digits, NPI check digit).
  Picking a result fills `provider`, `location` and `provider_npi` as a snapshot; nothing is fetched
  again later. Editing the provider text in the form clears `provider_npi`. PATCH and POST accept
  `provider_npi`; validation order puts it after `location`.
- **Logs.** The outcome (`ok`, `empty`, `cache_hit`, `rate_limited`, `timeout`, `upstream_error`),
  the duration and the result count. Never a term, a name or an NPI.
- **Screen.** A "Find a US provider" panel in the visit form, shown only when the flag is on. It
  works without JavaScript: the search is a form action that re-renders the form with the results as
  choices, keeping everything typed so far.
- **Tests.** The client is tested against a fake transport, the route in the access matrix, the panel
  against the stub API. Smoke asserts the off state (no panel, 404) and that `NPI_SEARCH=2` refuses
  to start. Smoke does **not** call the real registry **(call: a third-party call would make smoke
  flaky; the live path is checked by hand once after deploy)**.

## 11. Errors

| Status | Code | When |
|---|---|---|
| 404 | `not_found` | no grant, unknown kid, unknown visit, a visit of another kid: identical bodies |
| 422 | `invalid` + `field` | validation, with the field paths of 5.4 and 6 |
| 409 | `limit` | a per-kid cap |
| 409 | `stale` | outcome `revision` mismatch |
| 409 | `has_outcome` | cancel, or move to the future, a visit that has an outcome |
| 409 | `not_past` | outcome on a visit that is not `past` |
| 429 | `rate_limited` | provider search limit |
| 503 | `unavailable` | schedules not loaded (outcome with vaccinations); provider search upstream |

Logs carry the visit id, the route, the status and, for the outcome, three counts of rows created.

## 12. Testing

- Store: create, list, edit, cancel, the `has_outcome` refusals; the outcome transaction rolls back
  entirely on a failing row; a second call with the old revision writes nothing; delete with
  `linked=delete` and `keep`; deleting a kid removes its visits.
- HTTP: every visit route in the access matrix; the 404 body identical for no grant and unknown kid;
  the outcome validators are the modules' own (one table of cases run through both paths).
- `ics`: golden files for generic, full and cancelled; folding of a long line; escaping; UID stable
  and SEQUENCE rising across an edit; an instant either side of a daylight-saving change.
- UI: unit tests for the visit and outcome form readers; stub-API e2e for list, create, edit, cancel,
  outcome, delete dialog and download; the no-JavaScript path.
- Smoke (the real check, CLAUDE.md): a scheduled visit appears in Home's due list and downloads an
  `.ics` with the expected UID; a past visit's outcome creates a growth row, a vaccination and a
  medication that the three module pages show; deleting it removes them; a caregiver can do all of
  it; a parent with no grant gets the 404 page on every new route; no visit text in either
  container's logs.

## 13. Delivery

Two plans, in order **(owner agreed)**:

1. **Visits**: sections 5 to 9, 11, 12. Changes `api/` and `ui/` image content; no contract change.
2. **Provider lookup**: section 10. Changes `api/` and `ui/` image content and the edge contract.

Each plan also amends the main spec where this document changes it: the goals list, main 6.3 (the
`visits` table and the three `visit_id` columns), main 7.2 (the module row), main 13.1 (plan 2 only).

## 14. Left to the builder

- Go and TypeScript names, file layout, and how the three store functions are split.
- The visits page's exact layout, icons and copy, within the existing design tokens and parts.
- How the delete confirmation and the "Add another" blocks degrade without JavaScript, within the
  rules of 7 and 9.1.
- The cache's data structure and the in-flight limiter in `npi`.
- The pinned-count tests that move with this change (`wantTables`, `schema_version`, JSON key sets,
  `routeClasses`, `matrixParams`, nav links, the stub API).
