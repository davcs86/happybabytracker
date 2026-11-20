# Handoff: happybabytracker

Written 2026-11-20 at the end of the design session. Read this first in a new session, then the two
documents it points to. **No product code exists yet.**

## State

| Item | Where | Status |
|---|---|---|
| Design | `docs/superpowers/specs/2026-11-20-happybabytracker-design.md` | approved by the owner |
| Plan 1, foundation | `docs/superpowers/plans/2026-11-20-foundation.md` | approved by the owner, not started |
| Plans 2–7 | not written | see "Order of work" below |
| Design, plan 1 and this file | `main` of `davcs86/happybabytracker` | merged by PR #1 (docs only) |
| Implementation branch | none yet | create it from `main` when plan 1 starts |

The design is the source of truth. Markers in it: **(owner)** = decided by the owner, **(call)** = the
designer's choice the owner has seen and may still correct. This file repeats nothing from it except
what a new session needs to start correctly.

## What to do next

1. Execute plan 1, task by task, test first. The owner chose **subagent-driven execution** (a fresh
   agent per task, a fresh reviewer per task, then a whole-branch review); separate sessions per task
   are also acceptable. Use the `superpowers:subagent-driven-development` skill.
2. The owner said **do not start implementation** in the design session. Start only when the owner
   asks for it in the new session.
3. After plan 1 is merged and CI has published both images, write plan 2 (edge) with the
   `superpowers:writing-plans` skill, in `davcs86/homelab-edge` under `docs/superpowers/plans/`.

## Order of work (design section 16)

1. **Foundation** (this repo) — plan written.
2. **Edge** (`davcs86/homelab-edge`) — compose, the private name, secrets, backup, Gatus, bump workflow, runbook.
3. Daily trackers, timeline, modules, age-adaptive home, PWA install.
4. Growth.
5. Vaccination.
6. Sickness, medication, the reminder worker, Web Push, the ntfy channel; then the edge's ntfy switch.
7. Milestones and PDF export.

One plan per item, each its own PR, each written just before it is built.

## Repositories

| Repository | Role | Branch for this work |
|---|---|---|
| `davcs86/happybabytracker` | the app | a new branch from `main` |
| `davcs86/commsgun` | the architecture and tooling to mirror; read-only reference | none needed |
| `davcs86/homelab-edge` | deployment (Ansible); changed from plan 2 on | a new branch from its `main`, in plan 2 |
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

- **`.gitignore` in this repo is the Python template.** It ignores `lib/` and `build/`, which would
  silently drop `ui/src/lib`. Plan 1, task 1 replaces it. Do this before adding any UI file.
- **Plan 1 does not read the VAPID or `NTFY_*` variables.** Plan 6 adds them, and VAPID becomes
  required at start then. Plan 2 must therefore provision the VAPID key on the edge already.
- **Tailnet grants are per port, not per name.** Anyone allowed to reach `tag:edge:443` reaches every
  `*.private` service. This is why other parents are not given tailnet access now.
- **A `*.private` name is wired in four places** in `homelab-edge` (rule EDGE-07): the hostname
  variable, a `host` handle in `Caddyfile.j2`, `_names` in `roles/tailscale/tasks/main.yml`, and a
  Gatus probe.
- **The edge's nightly backup covers headscale only** (`roles/backup/templates/edge-backup.sh.j2`).
  Plan 2 adds this database to the same archive and its restore drill, and moves the timer to twice a
  day. The Gatus heartbeat endpoint name must not change (rule EDGE-10).
- **The edge's ntfy denies by default** and declares users, topic permissions and tokens in
  `roles/stack/templates/ntfy-server.yml.j2`. The app's publisher is a new write-only user there.
  ntfy's `base-url` name is the only one an iOS subscription works under.
- **A secret value on the edge may not contain a quote, a backslash or a line break.** The VAPID
  private key must be stored in a form that respects this.
- **xstockstrat's real VAPID subject is SOPS-encrypted** and was not read. The design uses this app's
  own origin instead; the owner was told and did not object.
- **commsgun's UI tests run against a stub API**, so they stay green on a broken contract. The smoke
  script, which runs both real images together, is the real check. Plan 1 keeps that pattern.

## Not yet verified (verify in the plan that needs it)

- Licence and availability of the UK-WHO and Euro-Growth tables, and that the WHO and CDC tables may
  be redistributed in a public repository (plan 4). If a required source cannot be bundled, stop and
  ask the owner.
- Actual resident memory of both containers against the 60 MiB and 140 MiB ceilings (plan 1 smoke).
- The corrected-age rule (until the second birthday) against the guidance published with each growth
  source (plan 4).
- The current published vaccination schedule of each of the four countries (plan 5).

## How to verify plan 1 is done

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

> Start from `main` of `davcs86/happybabytracker`. Read `docs/superpowers/HANDOFF.md`, then the
> design and plan 1 it points to. Execute plan 1 with subagent-driven development on a new branch
> created from `main`, and open a pull request against `main` when it is done. `davcs86/commsgun` is
> the reference to mirror; attach it read-only. Do not change `davcs86/homelab-edge` in this plan.
