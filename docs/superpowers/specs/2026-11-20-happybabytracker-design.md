# happybabytracker — health and growth tracker for babies and kids: design

Status: **approved by the owner** (2026-11-20). Plans: `docs/superpowers/plans/`.

Items marked **(owner)** were decided by the owner in the design conversation.
Items marked **(call)** are the designer's choices, not owner decisions — correct any of them.

## 1. Intent

A self-hosted family health record. Parents log a child's day (feeds, diapers, sleep), growth,
vaccinations, illness, medication and milestones from their phones, and see it as curves, schedules and
a timeline. The record belongs to the **child**: several parents and caregivers share one child's
record, and one parent can hold several children.

It is the third custom-code service on the homelab edge. It is built like `davcs86/commsgun` (Go API,
SvelteKit dashboard, SQLite, two images from one repo) and released exactly like it: images built by
CI here, pinned by digest in `davcs86/homelab-edge`, live after a manual deploy there.

## 2. Goals, non-goals, anti-goals

**Goals**

- G1. Multiple kids; all data hangs off a kid, never off a parent **(owner)**.
- G2. Access is granted per kid **(owner)**. A parent sees only the kids they were granted.
- G3. A seeded platform admin who manages parent accounts and can never see a kid or kid data
  **(owner)**.
- G4. Growth: weight, length/height, head circumference and BMI-for-age against selectable reference
  sources, by age and birth sex, with corrected age for preterm babies. The source can be changed at
  any time **(owner)**.
- G5. Vaccination: show the schedule of a selectable national schema, record administered doses
  (date required, lot optional), soft-flag overdue doses. The schema can be changed at any time
  **(owner)**.
- G6. Trackers: bottles, nursing sessions, pumping, sleep, diapers, feeding (solids), developmental
  milestones, sickness, medication with scheduled reminders **(owner)**.
- G7. Every tracker can be turned off and on again per kid; turning one off never deletes data
  **(owner)**.
- G8. All tracked data is queryable by time range, editable at any time and deletable **(owner)**.
- G9. The home screen adapts to the kid's age **(owner)**.
- G10. An installable PWA, mobile-first, with no feature that exists on only one device class
  **(owner)**.
- G11. Medication reminders by Web Push with VAPID, opted into per device **(owner)**.
- G15. The same reminders can also go to an ntfy server when the deployment configures one. ntfy is
  an addition, never a replacement: Web Push is always present **(owner)**. Nothing in this repository
  depends on `homelab-edge`; a fork runs against any ntfy server or none **(owner)**.
- G12. PDF export of a kid's record for a time range **(owner)**.
- G13. Each user picks display units; storage keeps metric and imperial values side by side,
  converted when a row is saved **(owner)**.
- G14. The database is backed up off the edge twice a day, encrypted **(owner: twice a day)**.

**Non-goals (stage one)**

- Offline use. The PWA is installable but needs a connection **(owner)**.
- Public ingress. The app is tailnet-only **(owner)**.
- Authelia login. The design leaves a seam for it (5.4) and nothing more **(owner)**.
- Canadian vaccination schedules (they are provincial) **(owner: left out for now)**.
- Photos, attachments **(owner)**. Edit history **(owner)**.
- Languages other than English **(owner)**.
- Email or SMS notifications, reminders for anything other than medication, push for overdue vaccines.
- Import from other apps, data sharing with clinicians other than the PDF.
- Diagnosis or advice of any kind: the app records and displays; it never interprets a symptom, a
  percentile or a missed vaccine beyond showing it.

**Anti-goals** — the design has failed if any of these happen, even if everything else works

- A parent reads or changes a kid they were not granted, or learns that such a kid exists.
- The admin reads any kid data, including a kid's name.
- A kid's record becomes unreachable because its last guardian was removed or disabled.
- A health value, a kid's or parent's name, a username or a secret appears in a log.
- The app is reachable from the internet.
- A percentile or schedule is computed from reference data whose origin and version are not recorded.
- The droplet is lost and more than half a day of entries cannot be restored.

## 3. Constraints found in the code

1. `happybabytracker` holds only `README.md`, `LICENSE` and `.gitignore`.
2. commsgun's shape (`commsgun/docs/superpowers/specs/2026-11-13-commsgun-design.md`): Go 1.27 binary
   on distroless, `modernc.org/sqlite` (pure Go, static binary), data on a named volume at `/data`,
   uid 10001, read-only root; SvelteKit 3 / Svelte 5 with the Node adapter, shadcn-svelte, Tailwind 4,
   layerchart, Node 24; the browser talks only to the SvelteKit server, which calls the API over the
   compose network with a shared token; limits and durations in one constants file.
3. commsgun's dashboard has exactly one user, `admin`, with the password in an env var. Nothing there
   supports several users, so identity and authorization are new work here.
4. commsgun's store is deliberately not backed up. The edge's nightly backup
   (`homelab-edge/roles/backup/templates/edge-backup.sh.j2`) archives headscale only: SQLite `.backup`,
   `PRAGMA integrity_check`, tar, `age`, rsync to the home backup host, then a restore drill and a
   Gatus heartbeat (`edge_nightly-backup`).
5. The edge is one `s-1vcpu-2gb-amd` droplet. Containers use the `x-hardening` anchor, `read_only`,
   ports bound to `127.0.0.1`, memory caps of 96–192 MB. The owner confirmed there is room for two more
   containers.
6. Tailnet-only services live under `*.private.<domain>` behind Caddy with a wildcard certificate.
   A new name is wired in four places (EDGE-07): hostname var, `host` handle in `Caddyfile.j2`,
   `_names` in `roles/tailscale/tasks/main.yml`, a Gatus probe.
7. Tailnet grants are per port, not per name. `group:members` (today only `david@`) may reach
   `tag:edge:443`, which is every `*.private` service: Gatus (no login), ntfy, the commsgun dashboard.
   Every person who uses this app needs a tailnet device with that reach, or a narrower rule in Caddy.
8. The policy tests deny `tag:edge → tag:db:5432`. Using the shared PostgreSQL would mean inverting
   that deny for the internet-facing host.
9. The droplet's Cloud Firewall allows all outbound traffic, so the API can reach the browser push
   services (FCM, Mozilla autopush, Apple) directly.
10. Secrets reach containers through Ansible-rendered `0600` env files read from
    `secrets/edge.sops.yaml`; generated keys are declared in `scripts/init_secrets.py`; a secret value
    may not contain a quote, backslash or line break.
11. Web Push on iOS works only for a PWA added to the home screen (iOS 16.4 or later).

## 4. Approaches considered

### 4.1 Application shape

1. **commsgun mirror — chosen (owner: acceptable as long as it serves the PWA and Web Push).**
   Two images. The Go API owns users, sessions, authorization, data, PDF export and the reminder
   worker, on a listener that is never published. The SvelteKit server is the only published port; it
   holds the session cookie and calls the API server-side. Authorization is enforced in the API, so a
   bug in the UI cannot widen access. Reuses commsgun's CI, digest pins, bump workflow and env-file
   shapes unchanged.
2. **One Go binary serving a static SPA.** One image and about 190 MB less memory. Rejected: the
   browser would call the API directly (CORS and CSRF handling in Go), and it breaks the two-image
   release contract and the UI patterns already proven in commsgun.
3. **SvelteKit-only monolith.** Fewest parts. Rejected: drops Go, moves the scheduler and
   authorization into Node, and is the furthest from "based on commsgun".

### 4.2 Storage

1. **SQLite on the edge, with encrypted snapshots to the home backup host — chosen (owner asked for a
   recommendation on security and resilience; approved with a twice-a-day interval).** No new network
   path, available whenever the edge is up, backups `age`-encrypted at home with a restore drill.
   Cost: up to half a day of entries lost if the droplet dies; the live database is plaintext on the
   droplet disk, the same trust level as the SOPS-rendered env files there.
2. **Shared PostgreSQL on the home DB host.** Point-in-time recovery. Rejected: needs
   `tag:edge → tag:db:5432` (constraint 8), makes the app depend on the home uplink and the DB host,
   puts its backups in the pgBackRest repository on the droplet, for which no repository cipher is
   configured, and departs from commsgun's store.

## 5. People, access and login

### 5.1 Principals

| Principal | Exists as | Can do | Cannot do |
|---|---|---|---|
| **Admin** | env vars only; not a row in `parents` | create parents, issue invite and password-reset links, disable and enable parents | see, list, count or name any kid; hold a kid grant; change its own username or password in the app; be deleted or demoted **(owner)** |
| **Parent** | row in `parents` | create kids; everything their grants allow; change their own password, display name, units | see parents or kids they share nothing with |

The admin's lack of kid access is structural, not a permission check: `kid_access.parent_id` is a
foreign key to `parents`, the admin is not in that table, and the admin session type is refused by
every route outside `/v1/admin/*` and `/v1/session`.

The admin's username and password come from `ADMIN_USERNAME` and `ADMIN_PASSWORD` (SOPS). At start the
API stores a keyed fingerprint of both in a `meta` row; when it differs from the stored one, every
admin session is deleted. Changing either in SOPS and redeploying therefore signs the admin out.

### 5.2 Per-kid access

| Role | Entries (create, edit, delete) | View, export | Kid profile and settings | Grant and revoke access | Delete the kid |
|---|---|---|---|---|---|
| `guardian` | yes | yes | yes | yes | yes |
| `caregiver` | yes | yes | no | no | no |

Two levels **(owner)**. A kid has one or more guardians and zero or more caregivers.

- A parent who creates a kid becomes its guardian **(owner)**.
- A guardian grants access by typing another parent's exact username and choosing a role. There is no
  directory of parents **(call)**. An unknown username and a disabled parent get the same answer.
- A kid always has at least one **active** guardian. The API refuses, in one transaction: revoking or
  downgrading the last guardian, a guardian leaving as the last one, and the admin disabling a parent
  who is the only active guardian of any kid **(owner)**. The admin's refusal says only "this parent
  is the sole guardian of at least one child"; no name or count.
- A request for a kid the caller has no grant on answers 404, identical to a kid that does not exist.

### 5.3 Accounts and sessions

- The admin creates a parent with a username (3–32 of `a-z0-9._-`, permanent) and a display name. The
  API returns a one-time invite link; the parent opens it and sets a password. Invite links last 72
  hours, reset links 1 hour **(call)**; tokens are 32 random bytes, stored hashed, single use.
- Passwords: at least 12 characters **(call)**, argon2id. A parent changes their own password in the
  app (current password required). A forgotten password is an admin-issued reset link **(owner)**;
  using it deletes that parent's sessions.
- Sessions are opaque 32-byte tokens, stored hashed in the API's `sessions` table. The UI keeps the
  token in a cookie (`HttpOnly`, `Secure`, `SameSite=Strict`, path `/`). Parent sessions slide for 30
  days, the admin session lasts 12 hours **(call)**. Logout deletes the row. A parent can see and end
  their other sessions.
- Failed logins are limited per client address and per username, in memory **(call: 5 per minute)**.
  The limiter counts admin and parent attempts together and answers both the same way.
- Disabling a parent deletes their sessions, push subscriptions and ntfy topic at once.

### 5.4 The Authelia seam

Authelia is approved for the edge but not deployed, and it is planned for **public** apps
(`homelab-edge/docs/runbooks/sso.md`). Nothing is built for it now. The seam is two things:

- A parent's credentials live in `identities (parent_id, provider, subject, secret_hash)`, unique on
  `(provider, subject)`. Today every row has `provider = 'local'`, `subject = username`. An Authelia
  login later adds rows with `provider = 'authelia'` and the Authelia `sub`; no table changes.
- One Go package turns a request into a principal. Adding a second way to authenticate changes that
  package and the UI's login route, and nothing that checks access.

## 6. Data

### 6.1 Storage rules

- SQLite, WAL mode, foreign keys on, one process. Ids are UUIDv7. Forward-only migrations embedded in
  the binary and applied at start.
- Every tracker has its own table with typed columns (6.3). Every entry row carries `kid_id`,
  `created_by`, `updated_by` (parent ids), `created_at`, `updated_at`, and an optional `note`.
- Instants are stored in UTC. Calendar dates (birth date, vaccination date, measurement date) are
  stored as dates. Each kid has an IANA time zone, set at creation from the creating device and
  editable; "today", day boundaries and reminder clock times use it **(call)**.
- Deletes are hard deletes **(owner)**. Deleting a kid (guardian only, confirmed by typing the kid's
  name) removes every row that references it. A deleted row stays in the backup archives until they
  rotate.
- Every store function that reads or writes kid data takes the kid id, and every such route passes
  one middleware that resolves the caller's role for that kid before the handler runs.
- All limits, caps and durations live in one constants file, as in commsgun.

### 6.2 Units

- A parent's profile holds four display preferences: weight (kg, or lb and oz), length (cm or in),
  volume (ml or US fl oz), temperature (°C or °F) **(owner: per user)**. Defaults are metric
  **(call)**.
- Every measured quantity is stored three ways in its row: the metric value, the imperial value, and
  which unit the user typed **(owner: both as columns)**. Both values are fixed-point integers
  (grams, millimetres, millilitres, hundredths of °C; hundredths of an ounce, of an inch, of a fluid
  ounce, of °F) **(call)**.
- One Go package converts, at save and at edit only **(owner)**. The typed value is stored exactly;
  the other is derived and rounded. Percentiles always read the metric columns.
- Ounces of volume are US fluid ounces (29.5735 ml) **(call)**.

### 6.3 What is stored

| Table | Purpose | Columns beyond the common ones |
|---|---|---|
| `parents` | accounts | username, display name, unit preferences, disabled flag |
| `identities` | credentials (5.4) | provider, subject, secret hash |
| `sessions`, `one_time_tokens` | login state (5.3) | token hash, kind, expiry |
| `kids` | profile and settings | name, birth date, birth sex (`female`/`male`), gestational age at birth (weeks, days; optional), time zone, growth source, vaccine schema (nullable) |
| `kid_access` | grants | kid, parent, role |
| `kid_modules` | per-kid overrides (7.2) | kid, module, enabled |
| `bottle_feeds` | bottles | time, contents (`breast_milk`/`formula`/`other`), volume |
| `nursing_sessions` | nursing | start, end (null while running), left seconds, right seconds, side in use and since when (while running) |
| `pumping_sessions` | pumping, recorded on the kid it is for **(call)** | start, duration, volume, side |
| `sleep_sessions` | sleep | start, end (null while asleep) |
| `diapers` | diapers | time, kind (`wet`/`dirty`/`both`/`dry`) |
| `meals` | feeding (solids) | time, what was eaten (text), reaction (text, optional) |
| `growth_measurements` | growth | date, any of weight, length/height, head circumference |
| `vaccine_doses` | administered doses (9) | administration id, date, antigen, lot (optional), product name (optional) |
| `vaccine_dismissals` | dismissed schedule lines (9) | schema, line, reason |
| `milestones_reached` | milestones (10.1) | catalog id or custom text, date |
| `symptoms` | sickness (10.2) | time, symptom, severity (optional), temperature and method (for `fever`) |
| `medications` | a medication course | name, dose text, schedule (10.3), start, end, active |
| `medication_doses` | doses given | medication, time, amount text |
| `reminders` | materialised reminder occurrences (11) | medication, due time, state, attempts |
| `push_subscriptions`, `push_kid_optins` | Web Push (11) | parent, device label, endpoint, keys; subscription, kid |
| `ntfy_topics`, `ntfy_kid_optins` | ntfy (11.1) | parent (one row each), topic; parent, kid |
| `meta` | schema version, admin fingerprint | key, value |

BMI is never stored; it is computed from a weight and a length/height measured on the same date.

A running sleep or nursing timer is a row with a null end, so it is shared: every parent with access
sees it, can switch the nursing side and can end it, from any device **(call)**. A kid has at most one
running row per table.

### 6.4 Querying by time range

- `GET /v1/kids/{kid}/timeline?from=&to=&types=` returns entries of the chosen trackers merged in
  time order, paged by cursor. Each tracker also has its own list endpoint with the same `from`/`to`.
- Each tracker table is indexed on `(kid_id, <its time column>)`.
- Daily summaries (bottle volume, nursing minutes, sleep hours, diaper counts per day) are computed
  by the API over a range, using the kid's time zone for day boundaries.

## 7. Age-adaptive experience

### 7.1 Age

Age is computed from the birth date in the kid's time zone. For a kid born before 37 weeks'
gestation, growth and milestones use **corrected age** (age minus the weeks short of 40) until the
second birthday **(call; the builder checks this against the guidance published with each growth
source and follows the source where it differs)**. Vaccination always uses chronological age. Wherever
corrected age is used the screen shows both ages.

### 7.2 Modules

A module is one tracker. Each has a default by age band, and a per-kid override that a guardian can
set either way at any time **(owner)**. Turning a module off hides its entry points and its
reminders; its data stays and reappears when it is turned on.

| Module | 0–11 months | 12–23 months | 24 months and over |
|---|---|---|---|
| Bottles, nursing, pumping | on | on | off |
| Diapers | on | on | off |
| Sleep | on | on | off |
| Feeding (solids) | on | on | off |
| Milestones | on | on | off |
| Growth | on | on | on |
| Vaccination | on | on | on |
| Sickness | on | on | on |
| Medication | on | on | on |

The 24-months column is the owner's decision; the split of the first two is the home-screen order
below, not a difference in what is on.

Defaults are evaluated live from the kid's age, with one exception: when a kid crosses into a band
where a module's default is off, a module that has an entry in the previous 30 days, or a running
timer, gets an explicit "on" override instead of disappearing **(call)**. A guardian turns it off
when it is no longer wanted.

### 7.3 Home screen

One home screen per kid, with a kid switcher at the top. It shows, in order: quick-log buttons, what
is running now (a sleep or nursing timer), what is due (medication, vaccines), and today's summary.
The quick-log buttons are the enabled modules in a fixed order per age band **(call)**:

- 0–11 months: bottle, nursing, diaper, sleep, then the rest.
- 12–23 months: feeding, bottle, sleep, diaper, then the rest.
- 24 months and over: sickness, medication, growth, vaccination.

Phone width (the layout every page is designed at first):

```
┌──────────────────────────────┐
│ Mia ▾        4 months 12 days│
├──────────────────────────────┤
│ [ Bottle ] [ Nursing ]       │
│ [ Diaper ] [ Sleep   ]       │
│ [ More…                    ] │
├──────────────────────────────┤
│ ● Asleep since 13:05   [End] │
├──────────────────────────────┤
│ Due                          │
│  Vitamin D drops     14:00   │
│  4-month vaccines   overdue  │
├──────────────────────────────┤
│ Today                        │
│  Bottles   5 · 22 oz         │
│  Nursing   3 · 41 min        │
│  Diapers   6 (4 wet, 2 dirty)│
│  Sleep     9 h 20 min        │
├──────────────────────────────┤
│ Home  Timeline  Growth  More │
└──────────────────────────────┘
```

On a wide screen the bottom bar becomes a side navigation and the sections sit in two columns. Both
layouts are the same routes and the same components; a page is not done until it has been checked at
phone and desktop width **(owner: no functionality gaps)**.

## 8. Growth

### 8.1 Reference sources

| Source id | Covers | Status |
|---|---|---|
| `who` | WHO Child Growth Standards 2006 (0–5 years) and WHO Reference 2007 (5–19 years) | required |
| `cdc` | CDC 2000 growth charts (0–20 years) | required |
| `uk-who` | UK-WHO charts (RCPCH) | include if the tables can be obtained and their licence permits bundling **(owner: not a hard requirement)** |
| `euro-growth` | Euro-Growth references | same condition **(owner)** |

- Each source is a set of LMS tables by sex and age, for weight-for-age, length/height-for-age, head
  circumference-for-age and BMI-for-age, embedded in the API binary. Every table file records its
  origin URL, version, retrieval date and licence. A test checks each table against worked examples
  published by its source.
- A source that cannot be bundled is left out; its id is not offered. The builder reports which of
  the two optional sources made it in. The licence check applies to `who` and `cdc` too, against the
  licence and visibility of this repository and its images (both public, 15), so bundling a table is redistribution; if a required source
  cannot be bundled the builder stops and asks the owner.

### 8.2 Behaviour

- The kid's source is a setting. Nothing stored depends on it, so changing it re-draws every curve
  and percentile at once **(owner)**.
- For each measurement the API returns the z-score and percentile under the current source, using
  corrected age where 7.1 applies. For charts it returns the source's percentile curves for the kid's
  sex over the visible age range.
- Where a source has no data for a measure or an age (for example head circumference beyond its
  range), the chart shows the kid's points without curves and says the source does not cover it.
- Charts: weight, length/height, head circumference, BMI; each with the kid's points over the
  percentile curves.

## 9. Vaccination

- Schemas at launch: United States, Mexico, United Kingdom, Switzerland **(owner)**. Each is a data
  file embedded in the API: for each recommended dose, the antigens it covers, its dose number and
  its recommended age window. Each file records the issuing authority, the schedule's edition and the
  date it was transcribed; the screen shows "Schedule: <authority>, <edition>".
- The schedule files are transcribed by the builder from the authorities' current published
  schedules and **reviewed by the owner in the pull request** before release **(call)**.
- An administered dose is stored per antigen: one administration (date, optional lot, optional
  product name) writes one row for each antigen the vaccine covers. Because doses are counted per
  antigen and not per schedule line, switching the kid's schema keeps every recorded dose and only
  changes what is shown as due **(owner: changeable at any time)**.
- A schema file lists, per antigen, a numbered series of doses with windows. Matching is by count
  only: the kid's Nth recorded dose of an antigen fills the Nth line of that antigen's series, whenever
  it was given. The app never judges whether a dose was valid, too early or too late **(call)**.
- Where a series length depends on the product (rotavirus, Hib), the file lists the longest series and
  marks the product-dependent lines "may not be needed". Recurring doses (seasonal influenza) are
  listed as recurring and are never due or overdue **(call)**.
- For each schedule dose the screen shows one of: given (with date), due (inside its window), upcoming,
  **overdue** (past its window and not given), or dismissed. Overdue is a soft flag: a label and a
  place in the home screen's "Due" list. No push, no blocking **(owner: soft flag)**.
- A guardian can dismiss any schedule line ("not needed" or "given, date unknown") and undo it, and
  can dismiss every line before a chosen date in one action, so a kid added at age four does not start
  with a wall of overdue doses. Dismissals are stored per kid and schema line (`vaccine_dismissals`)
  and survive a schema switch only for the schema they were made in **(call)**.
- A dose not in the kid's schema (travel vaccine, another country's product) can be recorded and is
  listed under "Other".
- The vaccination module can be turned off per kid like any other **(owner)**.

## 10. Sickness, medication, milestones

### 10.1 Milestones

A built-in catalog of common developmental milestones by age, taken from the CDC's public-domain
milestone lists **(call)**, plus free-text custom milestones. A guardian or caregiver marks one as
reached with a date. The screen shows the milestones around the kid's (corrected) age; it never says
a kid is late.

### 10.2 Sickness

Each entry is one symptom at a time: congestion, cough, fever, vomiting, diarrhea, constipation,
rash, ear pain, sore throat, poor appetite, unusually sleepy, other **(call beyond the owner's
"congestion, temperature, diarrhea, constipation, etc.")**. Optional severity (mild, moderate,
severe). `fever` carries a temperature (6.2) and how it was taken. There is no "illness episode"
object; the timeline filtered to sickness shows the course **(call)**.

### 10.3 Medication

- A medication is a course: name, dose as text ("2.5 ml"), start, optional end, and a schedule that
  is one of: none (as needed), fixed times of day, or every N hours from a start time **(call)**.
- "Every N hours" is a fixed grid from the start time. It is not re-anchored to the dose actually
  logged: the app does not compute a safe interval (non-goals). Instead the "next due" line shows
  when the last dose was logged, so the parent decides **(owner: do not recompute)**.
- A dose given is logged against the medication with a time. Logging is always possible, scheduled or
  not.
- The medication screen shows each active course with its last dose and its next due time.

## 11. Reminders, Web Push and ntfy

- The API holds a VAPID key pair (the private key from SOPS; the public key is derived and served to
  the UI) and a VAPID subject, which is an `https:` URL **(owner)**: the app's own origin **(call)**.
  Web Push is optional: both VAPID variables set, or neither (Web Push off). When they are set, the API
  refuses to start unless the subject is an `https:` URL **(owner, 2026-11-23)**. `davcs86/xstockstrat` already
  sends Web Push this way (`services/xstockstrat-notify/src/fanout/webPush.ts`); its push TTL of one
  hour and 10-second send timeout are the starting values here **(call)**.
- A parent turns notifications on **per device**, and on that device chooses the kids to be reminded
  about **(owner)**. Only a parent with a grant on the kid can opt in; revoking the grant removes the
  opt-in.
- The worker ticks every 30 seconds. For each active scheduled medication of a kid whose medication
  module is on, it materialises the next occurrence as a `reminders` row, unique on (medication, due
  time), so a restart or a second tick never creates or sends a duplicate.
- A due reminder is sent to every opted-in Web Push subscription for that kid, and to every
  opted-in ntfy topic when ntfy is configured (11.1). Each target is delivered and retried on its own;
  a failing channel never delays or blocks the other. The Web Push payload is encrypted to the
  device (RFC 8291). The notification is generic: "<Category> due for <kid's name>", for example
  "Medication due for Mia", and never names the medication or the dose **(owner)**. Medication is the
  only category in stage one; the payload carries the category so others can follow. It opens the
  medication screen.
- Delivery: a 404 or 410 from the push service deletes that subscription; a 429 or 5xx is retried
  with backoff; a reminder not sent within 30 minutes of its due time is marked `expired` and never
  sent late **(call)**. A dose logged for that medication from 30 minutes before the due time until
  the next occurrence marks that occurrence done; a dose logged before the push goes out cancels it
  **(call)**. The home screen's "Due" list and each course's "next due" read the same `reminders`
  rows, so the list and the pushes never disagree. A missed dose stays `overdue` on the Due list until
  its due time plus the kid's `reminder_overdue_minutes` (a guardian's kid setting, 5 to 1440, default
  60) or the course's next occurrence, whichever comes first **(owner, 2026-11-23)**.
- On iOS the settings screen explains that notifications need the app added to the home screen.
- The service worker handles install, push and notificationclick only, with no fetch handler **(owner,
  2026-11-23)**. It caches no kid data **(owner: online-only)**. The offline page planned here is
  deliberately not built (owner, 2026-11-23).

### 11.1 ntfy (optional second channel)

- ntfy is on when `NTFY_URL` is set, and absent from the UI and the worker when it is not. Web Push
  does not depend on it, and is on only when both VAPID variables are set **(owner, 2026-11-23,
  superseding "always on")**.
- One topic **per parent** **(owner)**. When a parent turns ntfy on in account settings, the API
  generates `<NTFY_TOPIC_PREFIX>-<22 random base62 characters>` **(call)**, stores it, and the screen
  shows the server address (`NTFY_PUBLIC_URL`, or `NTFY_URL` when that is unset) and the topic to
  subscribe to in the ntfy app. The parent can regenerate the topic (the old one stops receiving) or
  turn ntfy off (the topic is deleted).
- The parent chooses the kids to be reminded about on that topic, as for a Web Push device. Only a
  parent with a grant on the kid can opt in; revoking the grant removes the opt-in. So a reminder for
  a kid only ever goes to topics of parents who can see that kid.
- The message is the same generic text as the push, "<Category> due for <kid's name>", with the app's
  medication screen as its click address. The API publishes with `Authorization: Bearer <NTFY_TOKEN>`
  when a token is set and with no credentials when it is not.
- Delivery: a 429 or 5xx is retried with the same backoff and the same 30-minute expiry as a push. A
  401 or 403 is logged once per tick as a configuration fault (no topic in the log) and is not
  retried for that occurrence. The topic is a secret and is never logged.
- The app does not manage who may **read** a topic. On an open ntfy server the unguessable topic is
  the protection; on a server that denies by default, read access is the server operator's to grant.
- ntfy keeps a message in its cache (72 hours on the edge's server), so the kid's name rests there in
  plaintext for that long. The settings screen says so at the top of its ntfy section **(call)**.
- `GET /healthz/reminders` covers both channels: a delivery of a due reminder on either one still
  pending or sending 15 minutes after its due time turns it 503 (13.1).

## 12. PDF export

A guardian or caregiver exports one kid for a date range, choosing which modules to include. The API
renders the PDF itself with a pure-Go library (no browser, no CGO) and streams it through the UI. It
contains the kid's profile, growth table and charts with percentiles under the current source, the
vaccination record, and each chosen tracker's entries and daily summaries, in the requesting
parent's units **(call on contents; the owner asked for a PDF export)**.

## 13. Processes and configuration

### 13.1 API (`api/`, image `ghcr.io/davcs86/happybabytracker-api`)

One Go binary, one process, **one listener** on container port 9210, never published to the host. It
serves `/v1/*` to the UI container only and `GET /healthz`. Every `/v1/*` request must carry the
shared `INTERNAL_API_TOKEN`; requests acting for a person also carry that person's session token.

| Env var | Meaning |
|---|---|
| `ADMIN_USERNAME`, `ADMIN_PASSWORD` | the seeded admin (secret; password at least 16 characters) |
| `INTERNAL_API_TOKEN` | shared with the UI container (secret, at least 32 characters) |
| `VAPID_PRIVATE_KEY` | optional; P-256 private key for Web Push (secret) |
| `VAPID_SUBJECT` | optional; the `https:` subject sent to the push services; on the edge, the app's origin |
| `PUBLIC_ORIGIN` | the app's origin, used in invite and reset links and as the push click target |
| `DATA_DIR` | SQLite directory, default `/data`, the only writable path |
| `HBT_ALLOW_FAKE_PUSH` | `1` sends pushes to an in-memory recorder (tests and smoke only) |
| `NTFY_URL` | optional; the ntfy server the API publishes to. Unset turns ntfy off |
| `NTFY_TOKEN` | optional ntfy access token (secret) |
| `NTFY_TOPIC_PREFIX` | required with `NTFY_URL`; 1–32 of `a-z0-9-` |
| `NTFY_PUBLIC_URL` | optional; the server address shown to parents when it differs from `NTFY_URL` |

The process refuses to start on a missing or too-short secret, an invalid VAPID key or an invalid
origin. The VAPID variables are optional, both or neither (neither turns Web Push off) **(owner,
2026-11-23, superseding "the VAPID variables are always required")**; one of them without the other,
an `HBT_ALLOW_FAKE_PUSH` other than `1`, or `HBT_ALLOW_FAKE_PUSH=1` without them, is refused. It also refuses `NTFY_URL` without a
prefix, a prefix outside 1–32 of `a-z0-9-`, a malformed `NTFY_URL` or `NTFY_PUBLIC_URL`, an `NTFY_TOKEN`
with a space or a non-ASCII character, and any other `NTFY_*` variable set while `NTFY_URL` is not.

Health: `GET /healthz` is 200 while the database answers and the worker has ticked within 60 seconds.
`GET /healthz/reminders` is 503 when a delivery of a due reminder is still pending or sending 15 minutes
after its due time, or a delivery failed or expired within the last 15 minutes **(call)**; with no channel
configured it is 200.

### 13.2 UI (`ui/`, image `ghcr.io/davcs86/happybabytracker-ui`)

SvelteKit with the Node adapter on container port 9212, the same stack and versions policy as
commsgun's dashboard. The browser talks only to this server. It holds no data and no secret other
than `INTERNAL_API_TOKEN`.

| Env var | Meaning |
|---|---|
| `INTERNAL_API_TOKEN` | same value as the API's |
| `API_BASE_URL` | `http://happybabytracker-api:9210` on the edge |
| `ORIGIN` | the app's origin; a request whose derived origin differs gets 400, except the health paths |
| `PORT` | 9212 |

The image sets `PROTOCOL_HEADER`, `HOST_HEADER`, `ADDRESS_HEADER` and `XFF_DEPTH` as commsgun's does.
Health: `GET /healthz` (the UI itself), `GET /healthz/api` and `GET /healthz/reminders` (the API's
two answers passed through), so Gatus can probe everything through the one published name.

Every page that shows kid data is sent `Cache-Control: no-store`.

### 13.3 Logs

JSON lines. They carry ids (kid, parent, entry, request, reminder) and enumerated facts (route,
tracker, role, status, outcome, durations). Never a name, a username, a note, a health value, a
medication name, a push endpoint or a secret. URLs carry ids and `from`/`to`/`types` only, so the
edge's Caddy access log holds nothing else either.

## 14. Build and release (this repo)

Layout as commsgun: `api/`, `ui/`, one `Dockerfile` in each, `scripts/smoke.sh`,
`.github/workflows/ci.yml`, `CLAUDE.md`, `README.md`. Base images pinned by digest, actions pinned by
SHA, uid 10001, read-only root.

CI on every PR and on `main`: the API pipeline (`gofmt`, `go vet`, staticcheck, `go test -race`), the
UI pipeline (type-check, lint, unit tests, Playwright with axe at phone and desktop widths), both
image builds, then `scripts/smoke.sh`. On `main` only: push both images as `:<sha>` and `main`, label
both with `org.opencontainers.image.revision`, print both digests. Merging deploys nothing.

Tests the builder must include:

- **The access matrix.** For every kid-scoped route, generated from the route table so a new route
  without a row fails: guardian, caregiver, a parent with no grant (404), the admin (refused), no
  session (401).
- The last-guardian rules of 5.2, each inside its transaction, including the admin's disable.
- The admin fingerprint: a changed username or password at start ends admin sessions.
- Unit conversion round trips, and that an edit recomputes the derived column.
- Each growth table against its source's published examples; corrected age on and off; a source
  change altering percentiles without touching stored rows.
- Vaccination: a schema switch keeps every dose; a combination vaccine counts for each antigen;
  overdue at the window edge.
- Reminders: no duplicate after a restart mid-send; subscription removal on 410; expiry after 30
  minutes; a disabled module or a revoked grant stops reminders.
- Day boundaries and reminder times across a daylight-saving change in the kid's time zone.
- Logs contain no name, username, note or value.
- Smoke: create a parent through an invite, create a kid, log one entry of each tracker, read the
  timeline, export a PDF, deliver a fake push, check memory ceilings (60 MB API, 140 MB UI **(call)**).

## 15. Deployment (`davcs86/homelab-edge`)

Everything follows that repo's `CLAUDE.md`.

**Compose**: two services under the hardening anchor, both `read_only`, on their own network.

| Service | Image var | Ports | Volume | `mem_limit` |
|---|---|---|---|---|
| `happybabytracker-api` | `happybabytracker_api_image` | none published | `happybabytracker-data:/data` | 96m **(call)** |
| `happybabytracker-ui` | `happybabytracker_ui_image` | `127.0.0.1:9212:9212` | none | 192m **(call)** |

**Image visibility**: the repository is public **(owner)** and both GHCR packages are made public
once, as commsgun's were, so the edge and the bump workflow pull without a credential.

**Name**: one `*.private` name for the UI, wired in the four places of EDGE-07, inheriting
`tailnet_only` and `deny_staging_vm`: `happybaby.private.<domain>` **(owner)**. Nothing under
`*.public`, nothing in `edge_sites`.

**Who can reach it**: only the tailnet users that reach `*.private` today, `david@` and
`david_admin@` **(owner)**. No policy or Caddy change. Other parents and caregivers can have accounts
but cannot reach the app until the Authelia work changes how it is exposed (5.4); that work decides
their network path.

**Secrets** (`secrets/edge.sops.yaml`, declared in `scripts/init_secrets.py`, documented in
`secrets/README.md`):

| Key | Source |
|---|---|
| `happybabytracker_admin_username` | owner-provided **(owner: only in SOPS)** |
| `happybabytracker_admin_password` | generated; the owner may replace it with `sops set` |
| `happybabytracker_internal_api_token` | generated |
| `happybabytracker_vapid_private_key` | generated once (a P-256 key in a form with no quote, backslash or line break). Never rotated casually: a new key invalidates every push subscription |
| `happybabytracker_ntfy_token`, `happybabytracker_ntfy_password_bcrypt` | generated, only when ntfy is enabled for the app (below) |

The deploy asserts presence, shape and minimum length before it writes any file, as it does for
commsgun.

**Backup**: the existing `edge-backup` script also snapshots this database (`.backup`, then
`PRAGMA integrity_check`) into the same `age`-encrypted archive, the restore drill opens it, and the
timer runs at 00:00 and 12:00 UTC **(owner)**, which also moves the headscale snapshot from its
single 03:17 UTC run. The
existing heartbeat endpoint keeps its name (EDGE-10) and its interval is shortened to match
**(call: one archive and one timer, rather than a second unit)**. `docs/runbooks/recovery.md` gains
the restore steps for this database.

**Monitoring**: Gatus probes for `/healthz`, `/healthz/api` and `/healthz/reminders` through the
private name, in the `edge` group, with ntfy alerts.

**Pins and bumps**: both image vars in `versions.yml` with rows in `docs/versions.md`, and a
`bump-happybabytracker` workflow modelled on `bump-commsgun` that pins both in one PR and refuses a
pair whose revision labels differ.

**ntfy (the platform admin's choice)**: one inventory switch, `happybabytracker_ntfy_enabled`
**(owner: configurable on the edge)**. Off: no `NTFY_*` variable is rendered and the app runs on Web
Push alone. On, the edge PR adds:

- an ntfy user `happybabytracker` with write-only access to `<prefix>-*` and a token, declared in
  `ntfy-server.yml.j2` beside `alerts`; the prefix is an inventory value;
- `NTFY_URL` pointing at the ntfy container over a compose network the API joins, `NTFY_PUBLIC_URL` =
  `https://{{ ntfy_private_hostname }}` (ntfy's `base-url`, the only name an iOS subscription works
  under), `NTFY_TOKEN` and `NTFY_TOPIC_PREFIX` in the API's env file;
- nothing for readers: `david` already reads every topic as ntfy admin. Any other parent needs an ntfy
  user with read access to their own topic, added by hand in this repo when that day comes.

**Policy**: no change.

**Docs**: a runbook (first deploy, add a parent, reset a password, rotate the internal token, restore
the database, what a VAPID key change costs), the Context Guide row in `CLAUDE.md`, one entry in
`docs/decisions.md`, `secrets/README.md`.

## 16. Order of work

One design, several plans, each its own PR, each leaving something that works:

1. **Foundation** (this repo): repo skeleton, CI, both images, admin and parent login, invites, kids,
   per-kid access, units preferences, the access-matrix test, smoke.
2. **Edge** (`homelab-edge`): compose, name, secrets, backup, Gatus, bump workflow, runbook. The app
   is live and backed up before it holds anything that matters.
3. **Daily trackers**: bottles, nursing, pumping, sleep, diapers, feeding; timeline; modules; the
   age-adaptive home screen; PWA manifest and install.
4. **Growth**: measurements, reference tables, charts, corrected age.
5. **Vaccination**: schedule files, doses, due and overdue.
6. **Sickness and medication**: symptoms, courses, doses, the reminder worker, Web Push, the ntfy
   channel; then the edge's ntfy switch.
7. **Milestones and PDF export.**

Each plan after the second ends with: merge, the bump PR, a deploy, and a check on the edge.

## 17. Left to the builder

- Go package layout, router, migration mechanism, lint configuration; exact SQL and indexes beyond
  6.4.
- The JSON shapes between the UI and the API. They ship from one commit and are not a contract with
  another repository. The contract with `homelab-edge` is section 13 (ports, env var names, health
  paths, the data path) and the image names.
- Component composition and visual detail within shadcn-svelte's defaults, provided each page works
  at phone and desktop width and the home screen keeps the content and order of 7.3.
- The PDF library and layout; the Web Push library; the argon2id parameters (current OWASP guidance).
- Exact pinned versions (newest stable, at least two weeks old, pinned by digest or lockfile),
  matching commsgun's where the same tool is used.
- The percentile lines each chart draws, following each source's own convention.
- The milestone catalog's exact items and the vaccine schedule files, from the named authorities,
  subject to the owner's review in 9.
