# Handoff: happybabytracker

Updated 2026-11-23 at the end of plan 6, app side (sickness, medication, reminders, Web Push, ntfy). Read this first
in a new session, then the design and the plan of the next item.

## State

| Item | Where | Status |
|---|---|---|
| Design | `docs/superpowers/specs/2026-11-20-happybabytracker-design.md` | approved by the owner |
| Plan 1, foundation | `docs/superpowers/plans/2026-11-20-foundation.md` | done, merged to `main` (PR #3) |
| Plan 2, edge | `davcs86/homelab-edge` `docs/superpowers/plans/2026-11-21-happybabytracker-edge.md` | merged by the owner in `davcs86/homelab-edge` PR #100 on 2026-11-22 and deployed |
| Plan 3, daily trackers | `docs/superpowers/plans/2026-11-22-daily-trackers.md` | done, merged to `main` (PR #4) |
| Plan 4, growth | `docs/superpowers/plans/2026-11-22-growth.md` | done, merged to `main` (PR #6) |
| Plan 5, vaccination | `docs/superpowers/plans/2026-11-22-vaccination.md` | done, merged to `main` (PR #7) |
| Plan 5.1, advanced logging | `docs/superpowers/plans/2026-11-23-advanced-logging.md` | done, merged to `main` (PR #8) |
| Plan 6, sickness, medication, reminders, Web Push, ntfy (app side) | `docs/superpowers/plans/2026-11-22-sickness-medication-reminders.md` | done on branch `ccr-174d8c4e-jjczdt` (draft PR #9 to `main`, merged only with the owner's go-ahead) |
| Plan 6, edge side | `davcs86/homelab-edge`, not written | the VAPID PR, then the ntfy plan; see "What to do next" |
| Plan 7, milestones and PDF export | not written | see "Order of work" below |

The design is the source of truth. Markers in it: **(owner)** = decided by the owner, **(call)** = the
designer's choice the owner has seen and may still correct. This file repeats nothing from it except
what a new session needs to start correctly.

## What to do next

1. The owner reviews plan 6's PR #9 (from `ccr-174d8c4e-jjczdt` to `main`, merged only with the owner's
   go-ahead). After the merge, CI publishes both images; the edge's `bump-happybabytracker` PR changes the
   two pins and a deploy makes plan 6 live. Plan 6 changes `api/` and `ui/` sources only (no Dockerfile or
   dependency change; migration 0006 runs at the API's start) and deploys with the edge unchanged: no
   `VAPID_*` and no `NTFY_*`, so both channels are off and reminders show only in the app (the home Due
   list and the Medication page).
2. **Edge, Web Push:** whenever the owner wants Web Push on (before or after the plan-6 bump), a PR in
   `davcs86/homelab-edge` renders `VAPID_PRIVATE_KEY` (the stored `happybabytracker_vapid_private_key`, a
   literal secret read, EDGE-08) and `VAPID_SUBJECT` (`https://` + the app's private hostname) into
   `api.env`, both or neither, with asserts (43 base64url characters), flips its
   `test_vapid_is_not_rendered_in_plan_2` and updates the edge runbook's section 7 and `docs/decisions.md`.
   Outbound 443 from the API container was checked at the DigitalOcean firewall and the compose network
   on 2026-11-23; host-level egress is not.
3. **Edge, ntfy:** after the owner answers E1 and E2 (plan 6, "Questions for the owner"), an edge plan
   following `docs/superpowers/research/reminders/ntfy-and-edge.md`: a network shared by ntfy and the API
   only, `happybabytracker_ntfy_enabled` (shipping `false`) and the prefix, a write-only ntfy user and
   token, the `NTFY_*` rendering, the Gatus `/healthz/reminders` probe (after plan 6 is deployed); then a
   flip PR once the owner subscribes a phone. Operating notes for both channels:
   `docs/runbooks/reminders.md`.
4. Then plan 7 (milestones and PDF export): write it just before it is built, when the owner asks.

## Owner decisions in plan 3 (do not reopen)

- Band lookback: a module that is off by default in a kid's new band stays on when it has an entry from
  60 days before the band's start date onwards, or a running timer (the design said 30 days).
- Mixed bottles: contents are a non-empty set of breast milk, formula, cow's milk and other, with one
  total volume and no per-component split.
- R12, re-evaluation: the band rule is re-checked on every home or modules read, with no "already
  evaluated" marker. Such an entry at any time from 60 days before the band's start turns the module
  on once, as a stored row; a guardian's stored switch is never overridden.
- R13, nursing seconds: a finished session is created with minutes x 60 per side. Its edit form shows
  floor(seconds / 60) and sends a side's seconds only when its minutes changed. A running session's
  edit never sends seconds.
- R11, 24-hour timers: a timer that ran over 24 hours cannot be ended through the end route (422 `at`);
  the home says so and links to the edit page, where an end within 24 hours of the start, or a delete,
  settles it.

## Owner decisions in plan 4 (do not reopen)

- Licence: the code stays MIT; the WHO tables are bundled under their own licence, CC BY-NC-SA 3.0
  IGO, with a `LICENSE` and `NOTICE` in `api/internal/growth/data/who/`; the README says the published
  API image contains them.
- Sources: `who` and `cdc` only (UK-WHO and Euro-Growth are left out); the default is `who`, for new
  kids and, through migration 0003, for every existing kid.
- Corrected age is the app's own rule (WHO and CDC publish none for growth): a kid born before 37 weeks
  0 days is early by 280 minus the gestation in days, and that is subtracted from the age of a
  measurement dated before the second birthday, not on the birthday itself. Measurements before the
  expected term are listed unscored ("before expected term") and are not plotted.
- Measurements are not in the timeline; they live on the Growth page.
- Limits: at most 3000 measurements per kid (`MaxGrowthPerKid`); a row may hold any one or more of
  weight, length and head circumference.
- Charts are hand-written inline SVG; no chart dependency.
- Navigation: Home, Timeline, Growth, Account.

## Owner decisions in plan 5 (do not reopen)

- Schedules: `us`, `uk` (England only) and `mx`. Switzerland is left out pending the BAG's written
  consent, which reproduction needs in advance and which was not asked for; its research note stays in
  `docs/superpowers/research/vaccination/ch-bag.md`.
- Mexico: no open licence, so the file is a facts-only transcription (antigen, dose number, age) with
  the citation the gob.mx terms ask for (the URL and the consultation date). The README says it is not
  endorsed by, or a copy of, the Secretaría de Salud's publication.
- US edition: the CDC 2025 edition, revised 2025-07-02, which is what cdc.gov serves.
- Window rule: a source range's dash means "through" (`12–15 months` is `from 12m until 16m`, with
  `until` exclusive). A line without `until` gets a one-month default window, and overdue starts at
  `until`. `supplementary`, `recurring` and `may_not_be_needed` lines are never overdue. Cohorts are
  inclusive `born_from` and `born_to` dates.
- Navigation unchanged (Home, Timeline, Growth, Account). The Vaccines page is reached from the home's
  Due section, the kid links and the quick-log tile.
- A new kid has no schedule until a guardian chooses one. The home shows no Due list until then.
- Updating a schedule is easy by design: one CSV per schedule, a generated review document checked by a
  test, and stable line ids. The procedure is `docs/runbooks/updating-vaccine-schedules.md`.

## Owner decisions in plan 5.1 (do not reopen)

Plan 5.1 supersedes plan 3's "no per-component split" for bottles, only in an opt-in advanced form; the
simple forms and every older entry are unchanged. Its open values are in the plan, rows 1 to 21.

- Order: plan 5.1 ships first, as its own PR, before plan 6; plan 6's migration becomes 0006 (row 1).
- Mode switch: a per-form toggle only ("Advanced: volume per source", "Advanced: weigh before and
  after", opening the form with `?advanced=1`). Nothing is remembered; every form starts simple. An edit
  page opens advanced on its own when the entry already carries advanced data.
- Bottle total: every ticked source has its own volume and the total is their sum; the total is not
  editable on its own.
- Nursing weigh-ins: one weight before and one after the whole session (both sides together), each
  optional, in the parent's weight unit, the same input as the growth form, 200 g to 250 kg.
- Estimate: both the weight gain (grams for kg users, ounces for lb users) and an estimated milk volume in
  the parent's volume unit, converted at 1.03 g/ml. An after weight below the before weight is stored as
  typed, with no estimate and no message.
- "Do both" wording (rows 8 and 21): wherever the density is described (the constant's comment, the
  README, the form's help text) it reads
  "an approximation in the range commonly used (1.0–1.03 g/ml); not measured"; beside every place the
  estimate is explained, the sentence
  "A single-feed reading from a home scale can be off by several grams."
- Pumping per side (rows 17 to 20) is the owner's steer, applied by the designer by analogy with the
  bottle answers: a per-form toggle, the sum is the total, one unit select, the side is derived (left,
  right or both), the duration stays one value. The owner may overturn any of it.
- Design calls (rows 13 and 14; the owner may still correct them): the home's End button stays one tap and takes no
  weight (end the session, then edit it); the estimate shows on the nursing edit page and the timeline
  line only, not on the home, the daily summaries or per-source daily totals.
- Weigh-ins are not growth measurements: no growth row, no score, no chart.

## Owner decisions in plan 6 (do not reopen)

The owner's answers of 2026-11-23, recorded verbatim in the plan ("Questions for the owner: answered"):

- Web Push with VAPID is optional, both variables or neither; this supersedes "Web Push with VAPID is always
  required" (see the decision below). A small edge PR rendering the two VAPID variables may merge before or
  after the plan-6 bump.
- Deleting a course deletes its doses, behind a confirm that names the dose count.
- Doses are not on the timeline; symptoms are.
- Overdue window: a guardian-only kid setting, `reminder_overdue_minutes`, default 60, range 5 to 1440; a
  missed dose stays `overdue` on the home Due list until its due time plus that many minutes or the course's
  next occurrence, whichever comes first.
- A disabled parent's push devices, ntfy topic and opt-ins are deleted in the same transaction as the
  disable.
- One browser, two parents: the device moves to the parent who registers it last; the other parent's
  opt-ins for it are deleted.
- No offline page and no fetch handler: the service worker handles install, push and notificationclick
  only. Spec 11's offline page is deliberately not built.

## Order of work (design section 16)

1. **Foundation** (this repo) — done.
2. **Edge** (`davcs86/homelab-edge`) — compose, the private name, secrets, backup, Gatus, bump workflow, runbook — done.
3. Daily trackers, timeline, modules, age-adaptive home, PWA install — done.
4. Growth — done.
5. Vaccination — done (PR #7, merged).
5.1. Advanced bottle, nursing and pumping logging — done (PR #8, merged).
6. Sickness, medication, the reminder worker, Web Push, the ntfy channel — app side done on its branch (PR #9);
   then the edge's VAPID PR and the edge's ntfy plan and switch.
7. Milestones and PDF export.

One plan per item, each its own PR, each written just before it is built.

## Repositories

| Repository | Role | Branch for this work |
|---|---|---|
| `davcs86/happybabytracker` | the app | a new branch from `main` per plan |
| `davcs86/commsgun` | the architecture and tooling to mirror; read-only reference | none needed |
| `davcs86/homelab-edge` | deployment (Ansible); image pins change through its `bump-happybabytracker` PR | a new branch from its `main` only when a plan changes the edge (plan 6) |
| `davcs86/xstockstrat` | public; reference for Web Push with VAPID (`services/xstockstrat-notify/src/fanout/webPush.ts`, `services/xstockstrat-ui/docs/pwa-push.md`) | read-only |

Both `commsgun` and `homelab-edge` have a `CLAUDE.md` whose rules bind any work there: list
assumptions, minimum change, surgical diffs, show the check's output. `homelab-edge` also requires:
never assume a value the owner has not stated, Ansible only, pin everything, no secret in the repo or a
log.

## Decisions a new session must not reopen

- Two images mirroring commsgun: Go API on an unpublished listener (9210), SvelteKit Node server as
  the only published port (9212). Authorization is enforced in the API.
- SQLite on the edge. Backed up inside the existing `age`-encrypted edge archive, at 00:00 and 12:00
  UTC. Not the shared PostgreSQL.
- Tailnet-only at `happybaby.private.<domain>`. Only `david@` and `david_admin@` reach it for now;
  no policy or Caddy change. Authelia is a seam only (design 5.4).
- The repository and both GHCR images are public.
- The seeded admin: username and password only from SOPS; cannot be deleted, demoted or changed in the
  app; may create parents, issue invite and reset links, disable and enable parents; can never see a
  kid, a kid's name or a count of kids.
- Per-kid access with two roles, guardian and caregiver. A kid has one or more guardians and zero or
  more caregivers, and always at least one guardian whose account is active.
- Online-only PWA. No offline data.
- Units: per-user display preference; every measured value stored as metric and imperial columns plus
  the unit typed; converted only when a row is saved or edited. Volume ounces are US fluid ounces.
- Growth sources: WHO and CDC required; UK-WHO and Euro-Growth included only if their tables can be
  obtained and their licence permits bundling in a public repository. BMI-for-age and corrected age
  for preterm babies are in.
- Vaccination schemas: United States, Mexico, United Kingdom; Switzerland waits for the BAG's written
  consent (plan 5). Canada is out for now. Doses are matched by count per antigen; the app never judges
  validity. Overdue is a soft flag only. The owner reviews the transcribed schedule files in the pull
  request.
- Reminders are for medication only in this phase. "Every N hours" is a fixed clock grid and is never
  recomputed from the last dose logged.
- Notification text is generic: "<Category> due for <kid's name>". It never names the medication.
- Web Push with VAPID is **optional** (owner, 2026-11-23; this supersedes the earlier "Web Push with VAPID is always
  required"). With `VAPID_PRIVATE_KEY` and `VAPID_SUBJECT` unset the API starts with Web Push off; setting only one of
  them is a start error. When set, the VAPID subject is an `https:` URL: the app's own origin.
- ntfy is an optional second channel, on only when `NTFY_URL` is set, one generated topic per parent.
  Nothing in this repository may depend on `homelab-edge`; a fork runs against any ntfy server or none.
  On the edge it is behind one inventory switch, `happybabytracker_ntfy_enabled`.
- Hard deletes, no edit history, English only, no photos or attachments. Everything stays editable.
- Every tracker can be turned off and on again per kid without losing data.

## Things found during design that are easy to trip on

- **The edge does not render the VAPID or `NTFY_*` variables yet.** Plan 6 reads them, both channels
  optional (VAPID both or neither; owner, 2026-11-23). The edge already generates and stores
  `happybabytracker_vapid_private_key`; an edge PR that renders it turns Web Push on, and plan 6 can deploy
  before it. Rotating that key later silently stops every registered device until each parent turns
  notifications off and on again on it (`docs/runbooks/reminders.md`).
- **CI runs Playwright's headless shell; local runs use full Chromium** (`PW_CHROMIUM_PATH`). Reproduce a
  CI-only e2e failure locally with `PW_CHROMIUM_PATH` pointing at the headless shell under
  `/opt/pw-browsers/chromium_headless_shell-*`. A Playwright filter such as `home` matches the whole repo
  path (it also picks specs in `home/` directories or with `home` anywhere in the path): pass file paths.
- **Tailnet grants are per port, not per name.** Anyone allowed to reach `tag:edge:443` reaches every
  `*.private` service. This is why other parents are not given tailnet access now.
- **A `*.private` name is wired in four places** in `homelab-edge` (rule EDGE-07): the hostname
  variable, a `host` handle in `Caddyfile.j2`, `_names` in `roles/tailscale/tasks/main.yml`, and a
  Gatus probe.
- **The edge's backup** (`roles/backup/templates/edge-backup.sh.j2`) snapshots this database with
  headscale since plan 2. The Gatus heartbeat endpoint name must not change (rule EDGE-10).
- **The edge's ntfy denies by default** and declares users, topic permissions and tokens in
  `roles/stack/templates/ntfy-server.yml.j2`. The app's publisher is a new write-only user there.
  ntfy's `base-url` name is the only one an iOS subscription works under.
- **A secret value on the edge may not contain a quote, a backslash or a line break.** The VAPID
  private key must be stored in a form that respects this.
- **xstockstrat's real VAPID subject is SOPS-encrypted** and was not read. The design uses this app's
  own origin instead; the owner was told and did not object.
- **commsgun's UI tests run against a stub API**, so they stay green on a broken contract. The smoke
  script, which runs both real images together, is the real check. This repo keeps that pattern; since
  plan 3 the smoke logs, times, edits and deletes an entry of every tracker through the UI forms,
  since plan 4 it scores a measurement under both growth sources, and since plan 5 it records, edits
  and deletes a 6-in-1 dose against the real US schedule and switches the kid to the UK schedule, and
  since plan 5.1 it logs and edits an advanced bottle, pumping session and weighed nursing session through
  the real forms and checks that none adds to the Growth page. Since plan 6 it logs symptoms and a
  medication course with no channel configured (the Due list turns `overdue`), then restarts the API with
  the fake push sender and a per-run VAPID key and checks one generic push to the one opted-in device; its
  steps 4k and 4l are the first to call the unpublished API directly, from a container on the smoke
  network.

## Not yet verified (verify in the plan that needs it)

- The WHO licence reading: WHO's pages for the tables carry no licence and link to WHO's website
  terms (which ask for prior written authorization to reproduce substantial portions); applying CC
  BY-NC-SA 3.0 IGO to them is the owner's reading of WHO's publishing policy, not confirmed by WHO and
  not legal advice.
- The AAP guidance on corrected age: its page was blocked during plan 4's research, so the rule has not
  been checked against AAP's own text.
- Installing the app on a real phone (plan 3 checks the manifest, icons and head links only).
- Plan 6, on a real phone only: a displayed notification. The e2e service-worker test checks what the
  worker passes to `showNotification` and what a `notificationclick` does with synthetic events, not a
  displayed notification (Playwright's headless shell never displays one, whatever the grant). After the
  first deploy with Web Push on: a due reminder shows "Medication due for <name>" and tapping it opens or
  focuses the kid's Medication page (`docs/runbooks/reminders.md`, "Manual check").
- Plan 6, iOS: Apple's acceptance of the tailnet-only `https:` VAPID subject, delivery, the module service
  worker on iOS Safari 17+, a notification click into the Home Screen app and that app's separate login.
  iOS needs the Home Screen app; only a real iPhone can show these.
- Plan 6: real delivery through FCM, Mozilla autopush and Apple (tests and smoke use a fake push sender).
- Plan 6: ntfy publishing from the API container's address with `behind-proxy: true` and no
  `X-Forwarded-For` (the edge's existing publishers publish from `127.0.0.1`), and host-level egress from
  the API container on the edge beyond the DigitalOcean firewall rules.
- Plan 6, gaps by design: a renewed browser subscription (endpoint change) keeps its kids and label only if
  the Notifications page is visited before the next reminder is sent; if a send reaches the dead endpoint
  first (push service 404 or 410), the worker deletes that device and its opt-ins and the new subscription
  registers with no kids (there is no `pushsubscriptionchange` handler). After a VAPID key rotation every
  device stops delivering (401 or 403 are kept as `failed`, not deleted) and the page does not prompt: the
  parent uses "Turn off on this device" and turns it on again, which resubscribes under the new key. A
  stale ntfy topic cannot be seen while ntfy is off (a direct `DELETE /v1/me/ntfy` removes it).
- Plan 6, smoke: the push assertion sees only the title, path and tag (the test route decodes only those);
  the payload's other fields are covered by the Go tests, so the smoke is not proof that nothing else
  leaks. The API's memory ceiling has thin headroom: one smoke run on 2026-11-23 read a 61 MiB peak for
  the first API process (a transient during password hashing; other runs 50–51 MiB) against the 60 MiB
  ceiling, so step 10 can flake. The ceiling is the owner's call.
- Plan 6, PR notes: medication entries of the home `due` list carry `state`, vaccine entries `status`;
  medication entries come before vaccine ones; only the first page of doses is reachable from the UI for
  edit and delete.
- Mexico's reuse terms: shipping a facts-only transcription is the owner's reading of the gob.mx terms
  and of the Ley Federal del Derecho de Autor, art. 14. Not confirmed by the Secretaría de Salud.
- The US January 2026 revision: it is reported stayed by a court, by secondary sources only. CDC's own
  2026 text could not be retrieved.
- Mexico's Anexo 1 ages (minimum and maximum ages and intervals): a low-resolution scan with an
  illegible digit. No age in `mx.csv` comes from it.
- The rows marked `(verify)` in the schedule files, listed in the update runbook.
- The licence readings for all three schedules are not legal advice, and no authority (CDC, UKHSA, the
  Secretaría de Salud, the BAG) was contacted.
- Midnight DST starts: `kidtime.BandStart`, `correctionOn` and `CorrectedBirthDate` (plans 3 and 4) build
  local midnight in the kid's zone, so in zones whose DST starts at midnight (America/Havana, Santiago,
  Asunción) they may land a day early. `kidtime.AddAge` (plan 5) does civil-date arithmetic in UTC and
  does not have this flaw. A follow-up should fix the older three, with a Havana/Santiago test.
- The 1.03 g/ml density is the owner's figure, used as the app's own constant. A check of public sources on
  2026-11-23 found no measured human-milk mean; one study's working range was 0.9975–1.0566 g/ml, and
  hospital test-weighing protocols use 1 g as 1 ml. The app calls it an approximation, not measured.
- The accuracy sentence rests on validation studies of test weighing, whose published single-feed
  differences ranged from about −12 to +15 ml. A test-weighing estimate is not a clinical measure.
- The owner's observation that weight loss after nursing is associated with latching problems is not
  encoded anywhere: the app shows no estimate in that case and says nothing about it.

## How to verify a plan is done

```sh
cd api && gofmt -l . && go vet ./... \
  && GOTOOLCHAIN=$(go env GOVERSION) go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./... \
  && go test ./... -race
cd ui && pnpm install --frozen-lockfile && pnpm check && pnpm lint && pnpm test && pnpm test:e2e --trace off
docker build -t hbt-api:dev api && docker build -t hbt-ui:dev ui && scripts/smoke.sh hbt-api:dev hbt-ui:dev
```

`gofmt -l` prints nothing, the smoke script prints only `PASS` lines, and CI is green on the pull
request. Report the output; do not report "should pass".

## Prompt to start the next session

> Start from `main` of `davcs86/happybabytracker` after plan 6's PR #9 is merged. Read
> `docs/superpowers/HANDOFF.md`, then the design. For the edge: in `davcs86/homelab-edge`, under its
> `CLAUDE.md`, write the VAPID PR when the owner wants Web Push on, and the ntfy plan once the owner has
> answered E1 and E2 (plan 6), from `docs/superpowers/research/reminders/ntfy-and-edge.md`. For the app:
> ask the owner before writing plan 7 (milestones and PDF export), settle its open values with the owner,
> and execute it with subagent-driven development on a new branch from `main`. `davcs86/commsgun` is the
> reference to mirror; attach it read-only.
