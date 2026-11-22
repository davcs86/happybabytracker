# Handoff: happybabytracker

Updated 2026-11-22 at the end of plan 4 (growth). Read this first in a new session, then the
design and the plan of the next item.

## State

| Item | Where | Status |
|---|---|---|
| Design | `docs/superpowers/specs/2026-11-20-happybabytracker-design.md` | approved by the owner |
| Plan 1, foundation | `docs/superpowers/plans/2026-11-20-foundation.md` | done, merged to `main` (PR #3) |
| Plan 2, edge | `davcs86/homelab-edge` `docs/superpowers/plans/2026-11-21-happybabytracker-edge.md` | merged by the owner in `davcs86/homelab-edge` PR #100 on 2026-11-22 and deployed |
| Plan 3, daily trackers | `docs/superpowers/plans/2026-11-22-daily-trackers.md` | done, merged to `main` (PR #4) |
| Plan 4, growth | `docs/superpowers/plans/2026-11-22-growth.md` | done on branch `ccr-174d8c4e-jjczdt`, PR #6, not merged yet; after it merges the edge needs its `bump-happybabytracker` PR and a deploy |
| Plans 5–7 | not written | see "Order of work" below |

The design is the source of truth. Markers in it: **(owner)** = decided by the owner, **(call)** = the
designer's choice the owner has seen and may still correct. This file repeats nothing from it except
what a new session needs to start correctly.

## What to do next

1. After PR #6 is merged, CI publishes both images; the edge's `bump-happybabytracker` PR changes the
   two pins and a deploy makes plan 4 live. Nothing in plan 4 changes the edge otherwise; the API
   image's licence label is now `MIT AND CC-BY-NC-SA-3.0-IGO` and it carries the WHO and CDC notices
   under `/usr/share/doc/happybabytracker-api/`.
2. Write plan 5 (vaccination) with the `superpowers:writing-plans` skill, starting from the code on
   `main` after PR #6. Its age arithmetic starts from `api/internal/kidtime` and uses chronological age
   only (`AgeMonths`, `AgeDays`, `BandStart`); the corrected-age functions there (`CorrectionDays`,
   `CorrectedBirthDate`, `AgeBasisOn`) are for growth and must not be used for vaccination. The module
   table already knows `vaccination` (on in every band); the access matrix, the store/handler helpers
   (`api/internal/store/entries.go`, `api/internal/httpapi/entries.go`), the UI forms and the smoke
   script are what it extends. The home has no "Due" section until plans 5 and 6.
3. Start implementation only when the owner asks for it.

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

## Order of work (design section 16)

1. **Foundation** (this repo) — done.
2. **Edge** (`davcs86/homelab-edge`) — compose, the private name, secrets, backup, Gatus, bump workflow, runbook — done.
3. Daily trackers, timeline, modules, age-adaptive home, PWA install — done.
4. Growth — done.
5. Vaccination.
6. Sickness, medication, the reminder worker, Web Push, the ntfy channel; then the edge's ntfy switch.
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
- Vaccination schemas: United States, Mexico, United Kingdom, Switzerland. Canada is out for now.
  Doses are matched by count per antigen; the app never judges validity. Overdue is a soft flag only.
  The owner reviews the transcribed schedule files in the pull request.
- Reminders are for medication only in this phase. "Every N hours" is a fixed clock grid and is never
  recomputed from the last dose logged.
- Notification text is generic: "<Category> due for <kid's name>". It never names the medication.
- Web Push with VAPID is always required. The VAPID subject is an `https:` URL: the app's own origin.
- ntfy is an optional second channel, on only when `NTFY_URL` is set, one generated topic per parent.
  Nothing in this repository may depend on `homelab-edge`; a fork runs against any ntfy server or none.
  On the edge it is behind one inventory switch, `happybabytracker_ntfy_enabled`.
- Hard deletes, no edit history, English only, no photos or attachments. Everything stays editable.
- Every tracker can be turned off and on again per kid without losing data.

## Things found during design that are easy to trip on

- **Nothing reads the VAPID or `NTFY_*` variables yet.** Plan 6 adds them, and VAPID becomes
  required at start then. The edge already generates and stores `happybabytracker_vapid_private_key`
  but does not render it into the API's environment yet; plan 6 must add that before it deploys.
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
  plan 3 the smoke logs, times, edits and deletes an entry of every tracker through the UI forms, and
  since plan 4 it scores a measurement under both growth sources.

## Not yet verified (verify in the plan that needs it)

- The WHO licence reading: WHO's pages for the tables carry no licence and link to WHO's website
  terms (which ask for prior written authorization to reproduce substantial portions); applying CC
  BY-NC-SA 3.0 IGO to them is the owner's reading of WHO's publishing policy, not confirmed by WHO and
  not legal advice.
- The AAP guidance on corrected age: its page was blocked during plan 4's research, so the rule has not
  been checked against AAP's own text.
- Installing the app on a real phone (plan 3 checks the manifest, icons and head links only; there is
  no service worker until plan 6).
- The current published vaccination schedule of each of the four countries (plan 5).

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

> Start from `main` of `davcs86/happybabytracker` after PR #6 (plan 4) is merged. Read
> `docs/superpowers/HANDOFF.md`, then the design. Write plan 5 (vaccination) with the
> `superpowers:writing-plans` skill, settle its owner values first (the four countries' current
> schedules, their sources), and execute it with subagent-driven development on a new branch from `main`.
> `davcs86/commsgun` is the reference to mirror; attach it read-only. Do not change
> `davcs86/homelab-edge` in this plan beyond its bot's image bump.
