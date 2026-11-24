# happybabytracker

A self-hosted health record for babies and kids. The record belongs to the child: an admin creates
parent accounts and sends each parent a one-time invite link; a parent creates a kid and shares it with
other parents as a guardian or a caregiver. The admin manages accounts only and never sees a kid.

What a parent can do so far:

- Accounts, invites and resets; kids and per-kid access (guardian or caregiver).
- Daily trackers: bottles (one or more of breast milk, formula, cow's milk, other, with one total volume),
  nursing (a timer that switches sides, or a finished session in minutes per side), pumping, diapers, solids and sleep.
  Times are read and shown in the kid's time zone; volumes in each parent's unit (ml or US fl oz).
  Sleep and nursing run as timers from the home; a timer over 24 hours is settled from its edit page.
  Every entry can be edited or deleted by any parent with access. Bottles, pumping and nursing also have
  opt-in advanced forms (see "Advanced logging" below).
- The kid's home: quick-log buttons ordered for the kid's age, the running timers and today's totals.
- A timeline per kid: entries newest first, grouped by the kid's days with each day's totals, filtered
  by date range and tracker. A day's sleep total counts the part of each sleep inside that day, so an
  overnight sleep listed under the evening it started also adds to the next morning's total.
- Modules: a guardian switches any module off or on per kid; its entries stay.
- Growth: measurements of weight, length/height and head circumference, each scored as a z-score and a
  percentile against the kid's growth source and drawn on charts (see "Growth" below).
- Vaccination: recorded doses shown against a national schedule a guardian chooses for the kid, with a
  soft "Due" list on the home (see "Vaccination" below).
- Sickness and medication: symptoms (a fever with its temperature) on the timeline, medication courses
  with a schedule, logged doses, and reminders on the home's Due list and, when the server has a channel
  configured, on opted-in devices (see "Sickness, medication and reminders" below).
- Installable from the browser (a web app manifest and icons); online only. The service worker handles
  install, push and notification clicks only: it caches nothing and there is no offline page.

Modules and their defaults by age (whole months from the birth date, in the kid's zone):

| Module | Tracker | 0–11 | 12–23 | 24+ |
|---|---|---|---|---|
| `bottles`, `nursing`, `pumping`, `diapers`, `sleep`, `feeding` (solids) | built | on | on | off |
| `milestones` | later plan | on | on | off |
| `growth` | built | on | on | on |
| `vaccination` | built | on | on | on |
| `sickness`, `medication` | built | on | on | on |

A guardian's switch is final. When a kid enters a band where a module is off by default, the module
stays on if it has an entry from 60 days before the band's start onwards, or a running timer.

Two images, built from this repo:

| Image | Source | Port |
|---|---|---|
| `ghcr.io/davcs86/happybabytracker-api` | `api/` (Go, SQLite) | 9210, internal only, never published |
| `ghcr.io/davcs86/happybabytracker-ui` | `ui/` (SvelteKit, Node adapter) | 9212 |

The browser talks only to the UI; the UI calls the API with a shared token. Both run as uid 10001 on a
read-only root filesystem; the API writes only to `/data`. Design:
`docs/superpowers/specs/2026-11-20-happybabytracker-design.md`; plans: `docs/superpowers/plans/`.

## Growth

A measurement is a date (in the kid's zone, not in the future and not before the birth date) with any
one or more of weight, length/height and head circumference, typed in kg or lb and oz, cm or in, and a
note; a kid holds at most 3000. Each value is stored in metric and imperial with the unit typed.
Measurements live on the Growth page (Home, Timeline, Growth, Account in the navigation), not in the
timeline; any parent with access adds, edits and deletes them.

What is scored, at read time and never stored: weight-for-age, length/height-for-age, head
circumference-for-age and, for a row with both a weight and a length, BMI-for-age, each as a z-score (2
decimals) and a percentile (1 decimal, shown as "<0.1" or ">99.9" beyond those). The charts are inline
SVG with the source's percentile lines and a text table of the same points; they work without
JavaScript.

Sources, chosen per kid by a guardian in the kid's settings; the default is `who`:

| Id | Tables | Not covered |
|---|---|---|
| `who` | WHO Child Growth Standards 2006 by day, 0 to 1856 days; WHO Growth Reference 2007 by month, 61 to 228 months (weight to 120) | head circumference after 1856 days, weight after 120 months, anything after 228 months |
| `cdc` | CDC 2000 growth charts: infant tables below 24 months, child tables 24 to 240 months; head circumference from the infant table to 36 months | BMI under 24 months, head circumference after 36 months, anything after 240 months |

WHO weight and BMI beyond ±3 SD use WHO's restricted z-score; CDC applies no restriction. An age outside
a source's tables is shown as "this source has no table for this age", never extrapolated. Changing the
source changes every score and curve and no stored row.

Corrected age is the app's own rule; WHO and CDC publish none for growth. A kid born before 37 weeks 0
days of gestation is early by 280 minus the gestation in days, and that many days are subtracted from
the age of every measurement dated before the second birthday (not on the birthday itself). Wherever a
corrected age is used both ages are shown; a measurement before the expected term is listed as "before
expected term", unscored and not plotted.

The app records and displays; it never interprets a percentile and says nothing like "normal" or "low".
The tables and their licences are under "Third-party data" below.

## Vaccination

A guardian chooses the kid's schedule in the kid's settings; a new kid has none, and the Vaccines page
asks for one. Shipped schedules:

| Id | Authority and edition |
|---|---|
| `us` | Centers for Disease Control and Prevention (CDC), Child and Adolescent Immunization Schedule by Age, 2025 (revised 2025-07-02) |
| `uk` | UK Health Security Agency (UKHSA), Complete routine immunisation schedule from 1 September 2026 (updated 2026-09-24), England only |
| `mx` | Secretaría de Salud, CeNSIA, Lineamientos Generales del Programa de Vacunación Universal 2026 |

Switzerland is left out: the BAG requires written consent in advance to reproduce its schedule, and none
has been asked for. Each schedule is a partial, simplified transcription; what is left out is listed under
"Not transcribed from the source" in its review document, `docs/vaccine-schedules/<id>.md`.

Any parent with access records a dose: the date it was given (in the kid's zone, not in the future and not
before the birth date), the vaccines it covered (a product shortcut such as "6-in-1" ticks them, with
JavaScript on), and an optional lot number, product name and note; a kid holds at most 500. One dose of a
combination vaccine counts once for each vaccine it covers. Doses are kept per kid and never depend on the schedule:
switching the schedule changes only what is shown.

The Vaccines page (`/kids/{kid}/vaccines`, reached from the home's Due list, the kid's links and the
quick-log tile) shows "Schedule: <authority>, <edition>" with the retrieval date, the licence and
attribution, what is not included, then the schedule by age, each line as given, due, upcoming,
overdue, dismissed or available. The kid's Nth recorded dose of a vaccine fills its Nth line, whenever it
was given. **The app never judges validity:** it does not say whether a dose was too early, too late or
too close to another. Doses that the schedule has no line for are listed under "Other recorded
vaccinations".

The window rule, in plain words: a line is upcoming before the age the schedule names and due from that
age. A range such as "12–15 months" runs through the last month named: the line is due until the kid
turns 16 months and overdue from that day. A line with a single age ("8 weeks", "2 months") is due for
one calendar month and overdue after it. Some lines exist only for kids born in a given range of dates (UK
and Mexico cohorts). **Overdue is a soft flag:** a label and a place in the home's Due list, never a push
or a block. It never applies to these lines:

- lines that depend on the product used or on the kid's history ("may not be needed": due inside their
  window, then listed as available);
- lines for risk groups or shared decisions (listed under "Also available");
- seasonal lines (influenza, COVID-19), listed with the doses given.

A guardian can dismiss a line ("not needed" or "given, date unknown") and undo it. For a kid added
part-way through a schedule, "Dismiss older lines" dismisses in one go every line whose window starts
before a date and that is not given or dismissed yet. A dismissal belongs to the schedule it was made in.
The home's Due section lists the ages with a due or overdue line, three at most, linking to the Vaccines
page; it hides when the module is off.

Licences, per schedule (the full text is under "Third-party data" below):

- `us`: public domain, on CDC's four conditions. These are: credit CDC; imply no endorsement; do not
  change the substance; say the material is free on the CDC website.
- `uk`: Open Government Licence v3.0, citing UKHSA. No brand names or logos are included.
- `mx`: no open licence was found. Only facts are transcribed (vaccine, dose number, age), with the
  citation the gob.mx terms ask for: the document's URL and the date it was consulted. Those terms also
  limit downloads to personal, non-commercial use. Shipping a facts-only transcription is the owner's
  decision, not legal advice. It is not endorsed by, or a copy of, the Secretaría de Salud's publication.

Updating a schedule: `docs/runbooks/updating-vaccine-schedules.md`.

## Advanced logging

Three opt-in forms. Every form starts simple; a link at the top of a bottle, pumping or nursing form
("Advanced: volume per source", "Advanced: volume per side", "Advanced: weigh before and after") opens the
advanced one for that entry, and nothing is remembered. The simple forms and every entry saved before this
are unchanged. An edit page opens advanced on its own when the entry already carries advanced data. Saving
the simple bottle or pumping form over such an entry ("Use one total") replaces its parts with the one total
typed; hiding the weigh-ins of a session keeps its weights.

- Bottle, a volume per source: each ticked source (breast milk, formula, cow's milk, other) has its own
  volume and the total is their sum; the total is not typed on its own. One unit select (ml or fl oz) applies
  to every source, and each amount reads back as typed in the unit it was last saved in (the parts are shown and re-sent in the viewing parent's unit, so editing one part re-types the others).
- Pumping, a volume per side: a left and a right volume, at least one; the total is their sum and the side
  (left, right or both) follows from which are filled. The duration stays one value for the session.
- Nursing, weigh-ins: one optional weight before and one after the whole session, both sides together, typed
  like a growth weight (kg, or lb and oz; 200 g to 250 kg), on a new or an edited session, running or
  finished. The home's End button takes no weight: end the session, then edit it to add the after weight.

With both weights and the after weight not below the before weight, the timeline line and the edit page show
the weight gain in the parent's weight unit and an estimated milk volume in the parent's volume unit. The
volume converts the gain at 1.03 g/ml: an approximation in the range commonly used (1.0–1.03 g/ml); not measured.
A single-feed reading from a home scale can be off by several grams.
An after weight below the before weight is stored as typed, with no estimate and no message.

Weigh-ins are not growth measurements: they stay with their session and never reach the Growth page, a
score or a chart. The app records and displays; it never interprets a weight, a gain or a loss.

## Sickness, medication and reminders

- Symptoms (module `sickness`): congestion, cough, fever, vomiting, diarrhea, constipation, rash, ear pain,
  sore throat, poor appetite, unusually sleepy, or other with a short text, each with an optional severity.
  A fever carries its temperature (°C or °F, the parent's preference, up to two decimals) and how it was
  taken. Symptoms are on the timeline like the other trackers.
- Medication (module `medication`): a course has a name, a dose text, a start, an optional last day and a
  schedule: as needed, fixed times of day, or every N hours on a fixed grid from the start. The grid is
  never moved to the last dose logged and the app never suggests an interval. "Log dose now" on the
  Medication page records a dose; doses are not on the timeline. With the module off the page stays
  reachable and says reminders are paused.
- Reminders: each due occurrence of an active scheduled course is listed on the kid's home Due list (first,
  before vaccines) and as "Next due" on the Medication page, read from the same rows the sender uses. A
  missed dose shows `overdue` until the kid's "Show a missed dose as overdue for (minutes)" setting runs
  out (guardians, on the kid's settings page; 5 to 1440, default 60) or the course's next occurrence is
  due, whichever comes first. A dose logged from 30 minutes before a due time marks that occurrence done.
- What a reminder sends: only "Medication due for <kid's name>", opening the kid's Medication page. It
  never names the medication, the dose or a note. It goes only to devices or ntfy topics a parent opted in
  for that kid, while that parent is enabled and has access, and never later than 30 minutes after the due
  time.
- Channels, both optional and independent: Web Push (on when both `VAPID_*` variables are set) and ntfy (on
  when `NTFY_URL` is set). With neither, reminders still show on the Due list and the Medication page. Each
  parent turns them on under Account, Notifications: Web Push per device, choosing the kids; ntfy as one
  generated topic per parent. On iPhone and iPad, Web Push needs the app added to the Home Screen first.
  Signing out does not stop either channel on that phone or browser: turn it off there first (both sections
  say so). A new ntfy topic is created only after a confirm; a topic stored earlier while ntfy is off can
  be removed there.
- ntfy keeps each message in its cache (72 hours on the edge's server), so the kid's name rests on the
  ntfy server in plain text for that long; the Notifications page says so at the top of its ntfy section.

Operating reminders (health answers, a missing reminder, turning Web Push on): `docs/runbooks/reminders.md`.

## Configuration

API (`ghcr.io/davcs86/happybabytracker-api`):

| Env var | Meaning |
|---|---|
| `ADMIN_USERNAME` | The admin's login name, `^[a-z0-9._-]{3,32}$` |
| `ADMIN_PASSWORD` | The admin's password, at least 16 characters (secret) |
| `INTERNAL_API_TOKEN` | Shared with the UI, at least 32 characters (secret) |
| `PUBLIC_ORIGIN` | The app's origin, `http(s)://host[:port]` with no path; used in invite and reset links |
| `DATA_DIR` | SQLite directory, default `/data` (the only writable path) |
| `VAPID_PRIVATE_KEY` | Optional; P-256 private key for Web Push, 43 characters of unpadded base64url (secret) |
| `VAPID_SUBJECT` | Optional; the `https:` URL sent to the push services (the app's origin). Both VAPID variables or neither: neither turns Web Push off |
| `HBT_ALLOW_FAKE_PUSH` | `1` (with both VAPID variables) records pushes in memory instead of sending them, served at `GET /v1/_test/pushes`; tests and smoke only, never in production |
| `NTFY_URL` | Optional; the ntfy server the API publishes to, `http(s)://host[:port]`. Unset turns ntfy off |
| `NTFY_TOPIC_PREFIX` | Required with `NTFY_URL`; 1–32 of `a-z0-9-` |
| `NTFY_TOKEN` | Optional ntfy access token (secret) |
| `NTFY_PUBLIC_URL` | Optional `https:` server address shown to parents when it differs from `NTFY_URL` |

The process refuses to start on a missing or invalid value and names the variable, never its value. Only
`DATA_DIR` has a default. The VAPID and `NTFY_*` variables are optional, but any value that is set must be
valid: the two VAPID variables come both or neither, `HBT_ALLOW_FAKE_PUSH` is `1` or unset and needs both
VAPID variables, and every other `NTFY_*` variable needs `NTFY_URL`. For example, it refuses only
`VAPID_PRIVATE_KEY` or only `VAPID_SUBJECT`, a malformed key, a subject that is not an `https:` URL,
`HBT_ALLOW_FAKE_PUSH=1` without VAPID, and `NTFY_TOKEN` without `NTFY_URL`.
It listens on `:9210`. The image also sets `GOMEMLIMIT=40MiB`, so that
two concurrent password hashes stay within the edge's 96m container limit and the 64 MiB memory ceiling.
Changing `ADMIN_USERNAME`, `ADMIN_PASSWORD` or `INTERNAL_API_TOKEN` ends every admin session at the next
start; parent sessions are kept.

UI (`ghcr.io/davcs86/happybabytracker-ui`):

| Env var | Meaning |
|---|---|
| `INTERNAL_API_TOKEN` | Same value as the API's |
| `API_BASE_URL` | The API, for example `http://happybabytracker-api:9210` |
| `ORIGIN` | The app's origin; a request whose derived origin differs gets 400, except `/healthz*` |
| `PORT` | 9212 (set by the image) |
| `PROTOCOL_HEADER`, `HOST_HEADER`, `ADDRESS_HEADER`, `XFF_DEPTH` | Set by the image to `x-forwarded-proto`, `x-forwarded-host`, `x-forwarded-for` and `1` |

Because of the last row, the origin and client address come from `X-Forwarded-Proto`,
`X-Forwarded-Host` and `X-Forwarded-For`: put the UI behind a reverse proxy that sends all three. The
session cookie is `Secure`, so the public origin must be `https`.

Health, all through the UI: `GET /healthz` (the UI), `GET /healthz/api` (the API's `/healthz`, 200 only
while its database answers and the reminder worker has ticked within the last 60 seconds) and
`GET /healthz/reminders` (the API's, 503 while a reminder delivery is stuck or one failed or expired in the
last 15 minutes; 200 with no channel configured). What each answer means: `docs/runbooks/reminders.md`.

## Run locally

With Docker, on a private network (the API port stays unpublished):

```sh
docker build -t hbt-api:dev api && docker build -t hbt-ui:dev ui
TOKEN="$(head -c 24 /dev/urandom | od -An -tx1 | tr -d ' \n')"
docker network create hbt
docker run -d --name hbt-api --network hbt --read-only -v hbt-data:/data \
  -e ADMIN_USERNAME=admin -e ADMIN_PASSWORD='choose-16-or-more' \
  -e INTERNAL_API_TOKEN="$TOKEN" -e PUBLIC_ORIGIN=https://localhost:9212 hbt-api:dev
docker run -d --name hbt-ui --network hbt --read-only -p 127.0.0.1:9212:9212 \
  -e API_BASE_URL=http://hbt-api:9210 -e ORIGIN=https://localhost:9212 \
  -e INTERNAL_API_TOKEN="$TOKEN" hbt-ui:dev
```

The UI then expects a TLS-terminating proxy in front of `127.0.0.1:9212` that sends the three
`X-Forwarded-*` headers. `scripts/smoke.sh hbt-api:dev hbt-ui:dev` runs both images the same way and
plays that proxy with `curl`.

Without Docker, the API with `go run` and the UI with the Vite dev server, each in its own shell
started at the repository root. Both shells need the same token; pick one first:

```sh
export TOKEN=0123456789abcdef0123456789abcdef   # any value of at least 32 characters
```

Shell 1, the API (stays in the foreground):

```sh
cd api && ADMIN_USERNAME=admin ADMIN_PASSWORD='choose-16-or-more' \
  INTERNAL_API_TOKEN="$TOKEN" PUBLIC_ORIGIN=http://localhost:5173 DATA_DIR="$(mktemp -d)" \
  go run ./cmd/happybabytracker-api
```

Shell 2, the UI:

```sh
cd ui && pnpm install && INTERNAL_API_TOKEN="$TOKEN" API_BASE_URL=http://localhost:9210 \
  ORIGIN=http://localhost:5173 PORT=5173 pnpm dev
```

and open `http://localhost:5173` (browsers accept the `Secure` cookie on `localhost`).

## Test

```sh
cd api && gofmt -l . && go vet ./... && \
  GOTOOLCHAIN=$(go env GOVERSION) go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./... && \
  go test ./... -race
cd ui && pnpm install --frozen-lockfile && pnpm check && pnpm lint && pnpm test && pnpm test:e2e
docker build -t hbt-api:dev api && docker build -t hbt-ui:dev ui && \
  scripts/smoke.sh hbt-api:dev hbt-ui:dev
```

`gofmt -l` must print nothing. The UI's e2e runs against a stub API (it needs
`pnpm exec playwright install chromium`, or `PW_CHROMIUM_PATH` pointing at a Chromium); the smoke script
runs the real API and the real UI together and prints one `PASS:` line per assertion. CI
(`.github/workflows/ci.yml`) runs all three on every pull request and on `main`.

Memory readings: the smoke's step 10 reads each container's cgroup high-water mark
(`memory.max_usage_in_bytes`), which includes page cache, so it varies between runs. Three runs on
2026-11-23 with all three vaccination schedules loaded read API 46, 46 and 48 MiB for the first process
(9–10 MiB after the restart, 46 MiB after the login burst) and UI 120, 55 and 55 MiB, against ceilings of
60 and 140 MiB; an earlier run read API 58 MiB. A plan-6 run read a 61 MiB transient for the first API
process, so the owner raised the API ceiling to 64 MiB on 2026-11-24 (the edge's container limit is 96m).

## Release

Merging to `main` deploys nothing. CI builds both images, smoke-tests them together, pushes
`ghcr.io/davcs86/happybabytracker-api` and `ghcr.io/davcs86/happybabytracker-ui` (`:<sha>`, plus the
moving tag `main`, labelled with `org.opencontainers.image.revision`) and prints both `@sha256:…`
references in the `images` job summary. The edge runs the digests pinned in `davcs86/homelab-edge`; a
change is live only after those pins are changed there and the edge is deployed.

Deploy and rollback: both images ship together (the bump PR pins both). Migration 0006 (sickness,
medication, reminders) is forward-only: an older API refuses a database whose schema is newer than its
own, and the previous UI rejects the home list's medication entries, so rolling back the UI alone breaks
the home page of a kid whose Due list has a medication entry. Take an `edge-backup` archive right before the deploy. A rollback
is both pins back plus a restore of the last `edge-backup` archive; anything written since it is lost.

## Third-party data

The API embeds growth reference tables from two sources, retrieved 2026-11-22 and converted by
`scripts/growth/convert.py` with their values unchanged. The repository's code stays under the MIT
licence in `LICENSE`; the tables are not covered by it.

| Directory | Source | Licence |
|---|---|---|
| `api/internal/growth/data/who/` | WHO Child Growth Standards (2006; head circumference 2007) and WHO Growth Reference 2007 for 5–19 years, from who.int | **CC BY-NC-SA 3.0 IGO, not MIT**: non-commercial use only, share-alike, attribution. See its `LICENSE` and `NOTICE` |
| `api/internal/growth/data/cdc/` | CDC, National Center for Health Statistics, 2000 CDC Growth Charts, free at https://www.cdc.gov/growthcharts/cdc-data-files.htm | Public domain, on CDC's conditions, which its `NOTICE` meets: attribution "Source: CDC"; no implication that CDC, ATSDR, HHS or the United States Government endorses this project or its use of the data; no change to the substance of the content (only the percentile columns were dropped and each file split by sex); and a note that the data are available free of charge on the CDC website |

The API also embeds national vaccination schedules, transcribed by hand from each authority's
published schedule: one CSV per schedule under `api/internal/vaccines/data/`, each with its own
provenance header, and a generated review table per schedule in `docs/vaccine-schedules/`. They are
not covered by the MIT licence. Switzerland's schedule (BAG) is not included: reproducing it needs the
BAG's written consent, which has not been asked for.

| File | Authority and edition | Licence | Attribution |
|---|---|---|---|
| `api/internal/vaccines/data/us.csv` | Centers for Disease Control and Prevention (CDC), Child and Adolescent Immunization Schedule by Age, 2025 (addendum updated July 2, 2025), retrieved 2026-11-22 | Public domain, on CDC's conditions (https://www.cdc.gov/other/agencymaterials.html): attribution; no implied endorsement; no change to the substantive content; a note that the material is free on the CDC website. This is a partial, simplified transcription: risk-group, travel, pregnancy, catch-up and shared-decision dose rules and the items listed in the review document (`docs/vaccine-schedules/us.md`) are not transcribed; see the CDC schedule for the full text | "Source: CDC, Recommended Child and Adolescent Immunization Schedule for Ages 18 Years or Younger, United States, 2025 (revised 07/02/2025). CDC does not endorse this application or its use of the material. The material is available free of charge on the CDC website at https://www.cdc.gov/vaccines/hcp/imz-schedules/child-adolescent-age.html. This is a partial, simplified transcription and must not be read as CDC's own text: risk-group, travel, pregnancy, catch-up and shared-decision dose rules and the items listed in the review document are not transcribed; see the CDC schedule for the full text. Trade names are CDC's: “The use of trade names is for identification purposes only and does not imply endorsement by the ACIP or CDC.”" |
| `api/internal/vaccines/data/uk.csv` | UK Health Security Agency (UKHSA), Complete routine immunisation schedule from 1 September 2026 (updated 24 September 2026), England only, with the birth-date cohorts from UKHSA's MMRV and childhood schedule changes guidance (both updated 25 August 2026), retrieved 2026-11-22 | Open Government Licence v3.0 (https://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/), © Crown copyright 2026: attribution; no logos, trade marks or third-party rights (the file names no trade names); no implied endorsement. This is a partial, simplified transcription and adaptation with dose numbers derived by counting: selective, risk-group, pregnancy and catch-up programmes and the items listed in the review document (`docs/vaccine-schedules/uk.md`) are not transcribed; see the UKHSA schedule for the full text | "Source: UKHSA, “Complete routine immunisation schedule from 1 September 2026” (updated 24 September 2026), https://www.gov.uk/government/publications/the-complete-routine-immunisation-schedule, with the birth-date cohorts from UKHSA's “MMRV vaccination: information for healthcare professionals” and “Childhood schedule changes 2025 and 2026: information for healthcare practitioners” (both updated 25 August 2026). Contains public sector information licensed under the Open Government Licence v3.0 (https://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/). This is a partial, simplified transcription and adaptation for England, with dose numbers derived by counting and without trade names: selective, risk-group, pregnancy and catch-up programmes and the items listed in the review document are not transcribed. It is not endorsed by UKHSA; see the UKHSA schedule for the full text." |
| `api/internal/vaccines/data/mx.csv` | Secretaría de Salud, CeNSIA, Lineamientos Generales del Programa de Vacunación Universal 2026 (published 31 December 2025; measles amendment 18 February 2026), cross-checked with the PRONAM “Vacunación a lo largo de la vida”, 2a ed. (30 July 2026), consulted 2026-11-22 | No open licence: gob.mx publishes none on these documents. The gob.mx terms (https://www.gob.mx/terminos) say the portal's information is public unless indicated otherwise and require citing its URL and the date it was consulted (the same page also limits downloads to personal, non-commercial use). Only facts are transcribed (antigen, dose number, age), which the Ley Federal del Derecho de Autor, art. 14, excludes from protection together with administrative texts; no logos or prose are reproduced. This is a partial, simplified transcription: risk-group, pregnancy and catch-up lines, adolescent and adult lines other than HPV (Td included) and the items listed in the review document (`docs/vaccine-schedules/mx.md`) are not transcribed. It is not endorsed by the Secretaría de Salud and is not a copy of its publication; the official text prevails. This is not legal advice | "Fuente: Secretaría de Salud, CeNSIA, “Lineamientos Generales del Programa de Vacunación Universal 2026” (31 de diciembre de 2025; enmienda del 18 de febrero de 2026), https://www.gob.mx/cms/uploads/attachment/file/1076033/LINEAMIENTOS_GENERALES_2026_rem.pdf (page: https://www.gob.mx/salud/censia/documentos/lineamientos-generales-pvu-2026), consultado el 22 de noviembre de 2026 (2026-11-22). This is a partial, simplified transcription of facts only (antigen, dose number and age), with dose numbers derived by counting and without brand names: selective, risk-group, pregnancy and catch-up lines, adolescent and adult lines other than HPV (Td included) and the items listed in the review document are not transcribed. It is not endorsed by the Secretaría de Salud and is not a copy of its publication; the official text prevails." |

The published API image, `ghcr.io/davcs86/happybabytracker-api`, contains the WHO tables, which are
licensed CC BY-NC-SA 3.0 IGO: non-commercial use only, share-alike, attribution. Bundling them is the
owner's decision. The image carries the notices under `/usr/share/doc/happybabytracker-api/` and the
licence label `MIT AND CC-BY-NC-SA-3.0-IGO`; the UI image holds no tables and stays `MIT`. The CC BY-NC-SA
3.0 IGO terms apply to those tables wherever the image is redistributed, and a commercial use needs
WHO's permission. WHO's pages for these files carry no licence statement of their own and link to WHO's
website terms of use, which say that reproducing substantial portions needs WHO's prior written
authorization. Applying CC BY-NC-SA 3.0 IGO to the tables is the owner's reading of WHO's publishing
policy, not confirmed by WHO. None of this is legal advice.
