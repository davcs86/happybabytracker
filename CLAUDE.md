## How to Act

Read this first — it governs *how* you work here; everything below is the *what* you work with.
These four behaviors are the operating defaults; the rest of this file is context you load per task.

1. **Don't assume — ask, and surface tradeoffs.** Before acting, always list every assumption you
   are making on your own — scope, interpretation, defaults, a choice between options — each with
   its basis, or say "none". On ambiguity, a missing detail, or a design fork, stop and raise it.
   *Red flag:* "it's obvious" / "too small to ask" — that *is* the assumption; list it.
   *(Enforced by the contract with `davcs86/homelab-edge` in Rule 1, where a one-sided guess breaks
   the deployment, and by the design spec's **(owner)** items.)*
2. **Write the minimum that solves the stated problem.** Nothing speculative — no abstraction,
   option, or "while I'm here" scaffolding. *Red flag:* "might need it later", "for flexibility".
   *(Enforced by the plans in `docs/superpowers/plans/`: a later plan's feature does not land early.)*
3. **Touch only what the task requires.** Surgical, auditable diffs; clean up only orphans *you*
   introduced. *Red flag:* "cleaner if I also…" — propose it, don't do it.
   *(Enforced by the access matrix: `routes.go` and the pinned `routeClasses` in
   `api/internal/httpapi/matrix_test.go` must agree, so a route edit ripples into both.)*
4. **Define success up front, then loop until verified.** State the pass condition before you
   start; claim done only with the check's output from this session in hand.
   *Red flag:* "should work", "looks right", "probably passes".
   *(Enforced by the `Commands` pipelines and `scripts/smoke.sh`: the UI tests run against a stub
   API and stay green on a broken contract, so smoke is the real check.)*

No exceptions for small or obvious tasks — honoring the letter while dodging the intent is a violation.

> Litmus test for any future line in this file: *does it shape how the agent thinks (a behavior), or
> restate a fact the agent can read from the code?* If it's a fact already in the repo, leave it out.

# happybabytracker — CLAUDE.md

UI design rules are in `AGENTS.md` and bind as much as this file: read it before touching `ui/`.

Two containers built from one repo. `api/` (Go, module `github.com/davcs86/happybabytracker/api`, main
in `cmd/happybabytracker-api`): one listener on 9210, SQLite under `DATA_DIR` (`/data`). `ui/`
(SvelteKit, Node adapter): the only thing the browser talks to, on 9212; it calls the API server-side
with `INTERNAL_API_TOKEN` and the person's session token. Design:
`docs/superpowers/specs/2026-11-20-happybabytracker-design.md`; plans: `docs/superpowers/plans/`.

## Commands

- API: `cd api && gofmt -l . && go vet ./... && GOTOOLCHAIN=$(go env GOVERSION) go run
  honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./... && go test ./... -race` (`gofmt -l` must print nothing;
  staticcheck must run on the module's Go, not an older one)
- UI: `cd ui && pnpm install --frozen-lockfile && pnpm check && pnpm lint && pnpm test && pnpm test:e2e`
  (Node 24; e2e needs `pnpm exec playwright install chromium`, or `PW_CHROMIUM_PATH` pointing at an
  existing Chromium)
- Images and smoke (needs Docker): `docker build -t hbt-api:dev api && docker build -t hbt-ui:dev ui`
  then `scripts/smoke.sh hbt-api:dev hbt-ui:dev` (every line `PASS:`)

## Rules

1. **Contract with `davcs86/homelab-edge`**: API port 9210 (never published), UI port 9212, the env
   var names of spec 13.1/13.2, health paths `/healthz` and `/healthz/api`, data path `/data`, image
   names `ghcr.io/davcs86/happybabytracker-api` and `-ui`, uid 10001, read-only root filesystem. Never
   change one here alone.
2. **The admin never sees a kid.** The admin is not a row in `parents` and never holds a `kid_access`
   row; admin sessions are refused by every route outside `/v1/admin/*` and `/v1/session`.
3. **No grant is 404**, byte-identical to an unknown kid. A disabled actor counts as having no grant.
4. **A kid always has an active guardian.** The last-guardian check lives inside the store
   transaction that makes the change (revoke, downgrade, leave, the admin's disable), never in a
   handler's pre-check.
5. **No PII in logs or errors.** Logs are JSON lines of ids and enumerated facts: never a name,
   username, note, value, token or secret. No secret, name or username in an error string either.
   Invite and reset tokens travel in the URL path and must never reach OTEL (traces, metrics or logs):
   scrub `/invite/*` and `/reset/*` before any export. Caddy's own access log may keep them.
6. **Secrets are server-only and compared in constant time.** Read only in server-only code (`api/`,
   `ui/src/lib/server/`); never in a fixture.
7. **Limits live in two files only**: `api/internal/constants/constants.go` (API) and
   `ui/src/lib/server/constants.ts` (UI); `ui/src/lib/constants.ts` mirrors the client-safe ones. No
   literal elsewhere.
8. **Instants are unix seconds UTC** in INTEGER columns; RFC3339 UTC strings on the wire.
9. **Comments state constraints only, at most 2 lines.** How and why belong in the PR or the spec.
10. **Pins**: base images by digest, workflow actions by SHA; bump them deliberately, never invent a
    SHA.

## Release

Merging to `main` deploys nothing; CI publishes both images and prints their digests, and the edge
pins digests in `davcs86/homelab-edge`. When you finish a change, say in your report which image
content changed (`api/` or `ui/` sources, Dockerfile, dependencies) and that the edge still needs its
pins bumped and a deploy.
