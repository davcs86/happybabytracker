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
  Every entry can be edited or deleted by any parent with access.
- The kid's home: quick-log buttons ordered for the kid's age, the running timers and today's totals.
- A timeline per kid: entries newest first, grouped by the kid's days with each day's totals, filtered
  by date range and tracker. A day's sleep total counts the part of each sleep inside that day, so an
  overnight sleep listed under the evening it started also adds to the next morning's total.
- Modules: a guardian switches any module off or on per kid; its entries stay.
- Growth: measurements of weight, length/height and head circumference, each scored as a z-score and a
  percentile against the kid's growth source and drawn on charts (see "Growth" below).
- Installable from the browser (a web app manifest and icons); online only, no service worker.

Modules and their defaults by age (whole months from the birth date, in the kid's zone):

| Module | Tracker | 0–11 | 12–23 | 24+ |
|---|---|---|---|---|
| `bottles`, `nursing`, `pumping`, `diapers`, `sleep`, `feeding` (solids) | built | on | on | off |
| `milestones` | later plan | on | on | off |
| `growth` | built | on | on | on |
| `vaccination`, `sickness`, `medication` | later plans | on | on | on |

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

## Configuration

API (`ghcr.io/davcs86/happybabytracker-api`):

| Env var | Meaning |
|---|---|
| `ADMIN_USERNAME` | The admin's login name, `^[a-z0-9._-]{3,32}$` |
| `ADMIN_PASSWORD` | The admin's password, at least 16 characters (secret) |
| `INTERNAL_API_TOKEN` | Shared with the UI, at least 32 characters (secret) |
| `PUBLIC_ORIGIN` | The app's origin, `http(s)://host[:port]` with no path; used in invite and reset links |
| `DATA_DIR` | SQLite directory, default `/data` (the only writable path) |

The process refuses to start on a missing or invalid value (only `DATA_DIR` has a default) and names
the variable, never its value. It listens on `:9210`. The image also sets `GOMEMLIMIT=40MiB`, so that
two concurrent password hashes stay within the edge's 96m container limit and the 60 MiB memory ceiling.
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

The `VAPID_*` and `NTFY_*` variables of the design arrive with a later plan; nothing reads them yet.

Health, all through the UI: `GET /healthz` (the UI), `GET /healthz/api` (the API's `/healthz`, 200 while
its database answers).

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

## Release

Merging to `main` deploys nothing. CI builds both images, smoke-tests them together, pushes
`ghcr.io/davcs86/happybabytracker-api` and `ghcr.io/davcs86/happybabytracker-ui` (`:<sha>`, plus the
moving tag `main`, labelled with `org.opencontainers.image.revision`) and prints both `@sha256:…`
references in the `images` job summary. The edge runs the digests pinned in `davcs86/homelab-edge`; a
change is live only after those pins are changed there and the edge is deployed.

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
not covered by the MIT licence.

| File | Authority and edition | Licence | Attribution |
|---|---|---|---|
| `api/internal/vaccines/data/us.csv` | Centers for Disease Control and Prevention (CDC), Child and Adolescent Immunization Schedule by Age, 2025 (addendum updated July 2, 2025), retrieved 2026-11-22 | Public domain, on CDC's conditions (https://www.cdc.gov/other/agencymaterials.html): attribution; no implied endorsement; no change to the substantive content; a note that the material is free on the CDC website. This is a partial, simplified transcription: risk-group, travel, pregnancy, catch-up and shared-decision dose rules and the items listed in the review document (`docs/vaccine-schedules/us.md`) are not transcribed; see the CDC schedule for the full text | "Source: CDC. CDC does not endorse this application or its use of the material. The material is available free of charge at https://www.cdc.gov/vaccines/hcp/imz-schedules/child-adolescent-age.html." |
| `api/internal/vaccines/data/uk.csv` | UK Health Security Agency (UKHSA), Complete routine immunisation schedule from 1 September 2026 (updated 24 September 2026), England only, with the birth-date cohorts from UKHSA's MMRV and childhood schedule changes guidance (both updated 25 August 2026), retrieved 2026-11-22 | Open Government Licence v3.0 (https://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/), © Crown copyright 2026: attribution; no logos, trade marks or third-party rights (the file names no trade names); no implied endorsement. This is a partial, simplified transcription and adaptation with dose numbers derived by counting: selective, risk-group, pregnancy and catch-up programmes and the items listed in the review document (`docs/vaccine-schedules/uk.md`) are not transcribed; see the UKHSA schedule for the full text | "Source: UKHSA, Complete routine immunisation schedule from 1 September 2026, https://www.gov.uk/government/publications/the-complete-routine-immunisation-schedule. Contains public sector information licensed under the Open Government Licence v3.0. This is a partial, simplified transcription; it is not endorsed by UKHSA." |

The published API image, `ghcr.io/davcs86/happybabytracker-api`, contains the WHO tables, which are
licensed CC BY-NC-SA 3.0 IGO: non-commercial use only, share-alike, attribution. Bundling them is the
owner's decision. The image carries the notices under `/usr/share/doc/happybabytracker-api/` and the
licence label `MIT AND CC-BY-NC-SA-3.0-IGO`; the UI image holds no tables and stays `MIT`. The CC BY-NC-SA
3.0 IGO terms apply to those tables wherever the image is redistributed, and a commercial use needs
WHO's permission. WHO's pages for these files carry no licence statement of their own and link to WHO's
website terms of use, which say that reproducing substantial portions needs WHO's prior written
authorization. Applying CC BY-NC-SA 3.0 IGO to the tables is the owner's reading of WHO's publishing
policy, not confirmed by WHO. None of this is legal advice.
