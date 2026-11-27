# Single Sign-On Implementation Plan (plan 8)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status: plan only. Written against `main` fd9cf11 on 2026-11-24; re-grounded on 2026-11-27 against `main` as merged into this branch at 19eecfa (plans 1–7, 9 and 10 and the redesign merged) and updated for the owner's answers of that day (Open questions 1–9, 11–17 resolved; 10 open; 18 and 19 new). Execution waits for the owner's review, the open questions, and sign-off of "Deviations from the design".**

**Goal:** The admin moves from env vars into its own tables with a break-glass CLI, and the app becomes a provider-neutral OIDC relying party (login, invite redemption and explicit linking by SSO) behind `AUTH_MODE` (`local` default, `oidc`, `both`), so the edge can put the UI on `happybaby.public.<domain>` behind Authelia.

**Architecture:** The API stays the authority. Migration `0011_sso.sql` adds `admins` and `admin_identities`, rebuilds `sessions` with `admin_id` and `reauth_at` (keeping `device_label`), and drops the admin fingerprint. A start-up import copies the env admin once, and `happybabytracker-api admin <verb>` edits it afterwards. The API reads `AUTH_MODE`, gates every password and SSO route on it, and resolves `(iss, sub)` pairs into sessions. It never talks to the issuer. The UI server is the relying party (`openid-client`, code flow with PKCE, `state` and `nonce`). It keeps the flow in an AEAD-sealed `hbt_oidc` cookie, answers the callback with a 200 page (never a redirect), and asks the API to resolve, link or redeem.

**Tech Stack:** Go 1.27, `modernc.org/sqlite`, argon2id (`golang.org/x/crypto`); SvelteKit 3 / Svelte 5 (Node adapter), shadcn-svelte parts under `ui/src/lib/components/ui/`, Playwright, Vitest. **New:** `openid-client` 6.8.9 (exact pin, Open question 12). No new Go module.

**Spec:** `docs/superpowers/specs/2026-11-20-happybabytracker-design.md` § 5.5 (the whole section) and the 2026-11-24 and 2026-11-27 amendments in the header, § 2, § 5.1, § 5.3, § 5.4, § 6.1, § 6.3, § 13, § 14, § 15 and § 18. **UI design (binding, `AGENTS.md`):** `docs/plans/ui-redesign/design-system/DESIGN.md`, `components/COMPONENTS.md`, `patterns/PATTERNS.md`, `IMPLEMENTATION.md`, and the page specs `pages/sign-in.md`, `pages/set-password.md`, `pages/error.md`, `pages/my-profile.md`, `pages/admin.md`. Cross-repo source of truth: `davcs86/homelab-edge` `docs/superpowers/specs/2026-11-23-authelia-sso-respec.md` § 3 (app contract and conformance cases, with the 2026-11-27 bullets "Callback state" and "Signing keys" and the amended "Linking" and "Conformance"; `max_age=1` from `0bbe7ca`; read at branch `ccr-589817f0-cyg7gy` head `77d9d99`), § 4.3, § 4.6 (the `happybabytracker` client, `client_secret_basic` + PKCE), § 5 (happybabytracker row), § 8 and § 10 (P6), and `docs/runbooks/sso.md`. The edge's wiring (compose, Caddy zones and redaction, SOPS keys, `extra_hosts`, Gatus, the edge runbook) is a separate plan in that repo.

## Plan number and file name

The existing plans number themselves in their title, not in the file name, and the file name is `YYYY-MM-DD-<topic>.md`. Number 8 was reserved for SSO when this plan was written (§ 16 listed seven plans); plans 9 (visits, redesign data) and 10 (provider lookup) merged before it, so plan 8 executes after them and keeps its number. The file uses the path the request named (`2026-11-24-happybabytracker-sso.md`).

## Facts on 19eecfa this plan relies on

| Artifact | Where | Used by |
|---|---|---|
| Migrations `0001`–`0010`; `loadMigrations` refuses a gap in the 1..n sequence, so this plan's is `0011`; each runs in one transaction on the single writer (`_txlock=immediate`, `foreign_keys(1)`) | `api/internal/store/migrate.go:44`, `store.go:84` | 1 |
| `0010_redesign_data.sql` added `identities.secret_changed_at` and `sessions.device_label`; no migration indexes `sessions` or references it | `migrations/0010_redesign_data.sql:1-2` | 1 |
| Table rebuild precedent (`nursing_sessions_new`, `DROP`, `RENAME`) | `migrations/0005_advanced_logging.sql:19-54` | 1 |
| `sessions`: `CHECK ((kind = 'admin') = (parent_id IS NULL))` | `migrations/0001_foundation.sql:27-36` | 1 |
| `identities (parent_id, provider, subject, secret_hash, secret_changed_at)`, PK `(provider, subject)`; local rows are `provider = 'local'`, `subject = username` (`providerLocal`) | `0001`, `0010`; `store/parents.go:15` | 1, 4 |
| `wantTables` (`:15`); `schema_version` `"10"` at ten sites `store_test.go:36, 54, 400, 549, 749, 856, 975, 1058, 1346, 1431`; `TestMigration0007KeepsRows` and `TestMigration0008KeepsRows` count every `wantTables` table in the pre-migration database (`:956`, `:1039`, already skipping `visits`); `TestMigration0008KeepsRows` (`:999`, visits), `0009` (`:1310`) and `0010` (`:1398`, applies `ms[:9]`) exist; `TestAdminSessionShape` (`:91`) | `api/internal/store/store_test.go` | 1 |
| `CreateSession(ctx, kind, parentID string, tokenHash []byte, now, expires time.Time, deviceLabel string)`; `SessionByToken`; `DeleteAdminSessions`; `ListSessions` reads `device_label` | `store/sessions.go:13, 33, 67, 94` | 1, 4, 5 |
| `CreateParent` (one `INSERT`, `ErrConflict` on the unique username) | `store/parents.go:57` | 1 |
| `setPassword(ctx, tx, parentID, secretHash, keepSessionID string, now time.Time)` (upserts the local identity with `secret_changed_at`, then `DELETE FROM sessions WHERE parent_id = ? AND id <> ?`), `SetParentDisabled` (sessions, `deleteChannels`, last-guardian check) | `store/parents.go:123, 133-138, 144`; `store/push.go:209` | 2, 4, 6 |
| `IssueToken` (replaces unused tokens of a kind), `PeekToken`, `RedeemToken` (→ `setPassword(…, "", now)`) | `store/tokens.go:13, 50, 63, 81` | 4 |
| `prepare(ctx, st, cfg)`, `syncAdminFingerprint`, `checkAdminUsername`, `fingerprintKey`, `errAdminCollision`; `main()` calls `run` | `api/cmd/happybabytracker-api/main.go:123, 185, 201, 33, 35, 37` | 1, 2 |
| Start-up tests `TestAdminFingerprintRotation`, `TestFirstStartRecordsTheFingerprint`, `TestAdminUsernameCollidingWithAParentIsRefused`, `TestConfigNeverHoldsAnUnnormalisedAdminUsername`, `TestRunRefusesToStartOnACollision`; `collisionMsg`; `baseEnv` | `main_test.go:62, 137, 162, 193, 205, 160, 238` | 1 |
| `auth.AdminFingerprint`, `auth.CheckAdmin` (the only importer of `config` in `auth`); `idKey` is unexported in `auth` | `api/internal/auth/admin.go`; `password.go:21` | 1, 2 |
| `config.Load` validates `ADMIN_USERNAME` (pattern on the raw value) and `ADMIN_PASSWORD` (≥ `MinAdminPasswordLen` = 16) | `api/internal/config/config.go:49-54` | 1, 3 |
| `login`: decode, `device_label` (422, `constants.DeviceLabelPattern`) **before** the limiter, `addr:` then `user:` limiter keys, one verify; an admin session stores no label. `checkCredentials` (the admin success skips the verify today) | `httpapi/session.go:74, 83-86, 117-120, 129` | 1, 3, 4 |
| The password login sends `deviceLabel(user-agent)`, a closed set of words | `ui/src/routes/login/+page.server.ts:26`; `ui/src/lib/server/device-label.ts:19` | 10, 11 |
| `parentView` keys `id, username, display_name, units, password_changed_at`; asserted at `session_test.go:278`, `me_test.go:43`; admin view keys `kind, expires_at` at `session_test.go:266` | `httpapi/session.go:36-43` | 5 |
| `adminParentView` keys asserted at `admin_test.go:75` (create) and `:315` (list, with `last_active_at`, `link`) | `httpapi/admin.go:16-23` | 6 |
| `createParent` refuses `username == d.Cfg.AdminUsername` with 422; its test `admin_test.go:87-92` | `httpapi/admin.go:65` | 1 |
| `redeemToken`: decode, `addr:` limiter, `PeekToken`, hash, `RedeemToken`, parent session with no label | `httpapi/admin.go:220` | 3 |
| `resetLink` issues an invite when the parent has no password | `httpapi/admin.go:152` | Open question 13 |
| `changePassword`: no limiter today | `httpapi/me.go:87` | 5 |
| `Access` classes; `Routes(d)`; `routeClasses` (**125 keys**) at `matrix_test.go:59-185`; `matrixWant` (`:47`); `matrixBody` (`:207`, decode-first 422); `newMatrixFixture` (`:303`) seeds the admin with `f.seed("admin", "", …)` (`:309`) | `httpapi/routes.go`, `matrix_test.go` | 1, 3–6 |
| Test kit: `newKitWith` (`:146`, sets `AdminUsername: testAdminUser`, `AdminPassword: testAdminPass`), `asAdmin` (`:219`), `seedSession(kind, parentID, ttl)` (`:236`), `do`, `login`, `wantErr`, `sameKeys`, `wantCleanLog`; `verifyPassword` is a replaceable var | `httpapi/testkit_test.go`; `session.go:22` | all API tasks |
| UI: `hooks.server.ts` (`isPublic` `:16`, admin redirect `:60`, `cache-control: no-store`); `src/env.ts` (`defineEnvVars`, every variable through `optional`, which turns `undefined` into `''`); `loadEnv` | `ui/src/hooks.server.ts`; `ui/src/env.ts`; `ui/src/lib/server/env.ts` | 7, 10 |
| UI client: `createApi(env, fetch, session?, clientAddress?)`, `refused`, `logFailure` (fixed shape), `toSession`, `toParent`, `toAdminParent`, `withId`; `login(username, password, deviceLabel?)` sends `device_label` only when given | `ui/src/lib/server/api.ts:1141, 1200, 211, 312, 301, 367, 1289, 1316` | 8 |
| `failAction` turns 401/403/404 into the 404 page: an action must map a known 403 code before calling it | `ui/src/lib/server/outcome.ts` | 10–12 |
| `setPasswordRoute(kind)` serves invite and reset with one **default** action; `SetPasswordForm.svelte` posts with no `action` (`:67`), shows the invalid-link Alert and "Go to sign in" | `ui/src/lib/server/set-password.ts`; `ui/src/lib/components/SetPasswordForm.svelte` | 11 |
| Login page: one card that is the form (`rounded-2xl border bg-card p-4`), primary `Button size="xl"` "Sign in"; outside the shell; the error page is the outside-shell frame with h1, lead and an xl `Button href` | `ui/src/routes/login/+page.svelte`; `ui/src/routes/+error.svelte` | 10 |
| Account page: `Card role="region"` sections (Details, Password, Signed-in devices, Children) in the 920 px two-column grid; the Password card's inline form posts `?/password`; the `password` action maps `wrong_password` and has no 429 case | `ui/src/routes/(app)/account/{+page.svelte,+page.server.ts}` | 12 |
| Admin: `routes/admin/+layout.svelte` holds a side `nav` "Primary" with only the brand line (hidden below `md`) and a header row with the "Admin" badge; no `navModel` call. The parents page rows use a status `Badge`, outline row actions and the `<details>` destructive confirm (Disable); its footnote copies the prototype: "The admin’s own username and password can’t be changed here. They come from the deployment; changing them signs the admin out." | `ui/src/routes/admin/+layout.svelte:8-24`; `admin/parents/+page.svelte:220-299` | 12 |
| `navModel(i: NavInput)` is the one nav list (AppShell side nav and More sheet); the parent header's bell is a 48 × 48 outline icon link with `aria-label` and `aria-current` | `ui/src/lib/nav.ts:56`; `components/AppShell.svelte:37, 63`; `components/PageHeader.svelte:39-48` | 12 |
| Stub API: `route()` handles `POST /v1/session` (`:5106`, `device_label` checked before the `rate_limited` scenario) and `/v1/tokens/*` (`:5150`) by hand; `createParent` refuses `ADMIN.username` with 422 (`:796`); controls `/__scenario`, `/__reset`, `/__clock`; helpers `resetStub`, `moveStubClock`, `openSession` | `ui/e2e/stub-api.ts` | 1, 8, 10–12 |
| e2e: `playwright.config.ts` has **no `webServer`**, runs `fullyParallel`, and `globalSetup: './e2e/stacks.ts'` starts one stub API and one UI per worker slot (`slotPorts(slot) = { stub: 4181 + 100·slot, ui: 4173 + 100·slot }`, `stub-api.ts:8`), polling readiness with `fetch`; projects `phone` 390, `desktop` 1280, `side-nav` 980; dark runs through `test.use({ colorScheme: 'dark' })` (`theme.spec.ts:18`); `expectAccessible` runs axe | `ui/playwright.config.ts`; `ui/e2e/stacks.ts:46-82`; `ui/e2e/helpers.ts:7` | 10–12 |
| `pnpm screenshots` (`playwright.screenshots.config.ts`: one stub and one UI as `webServer`, `workers: 1`, frozen clock) writes `docs/screenshots` and its README; `pnpm test:visual` reuses that `webServer` and compares the design baselines, matching only `sign-in`, `set-password` and `error` (`tests/visual/matched.ts`) | `ui/playwright.{screenshots,visual}.config.ts` | 10–12 |
| Wiki pages that describe what this plan changes: `Sign-in-and-invites.md`, `My-profile.md`, `Admin.md` (`:78` quotes the footnote), `Navigation-and-shell.md` (`:118` "an admin session only reaches `/admin/parents`"), `Not-built-yet.md` (`:9`), `How-these-screenshots-are-made.md` | `docs/wiki/` | 10–13 |
| Smoke: `redeem` posts the invite path (`:193`); `refused` (`:1051`); assertion 8 (`:1650-1664`); log list of assertion 9 (`:1723`) | `scripts/smoke.sh` | 1–3, 11 |
| No OTEL exporter exists in either image (no `otel` import anywhere) | `api/`, `ui/` | Open question 8 |
| No PR template in the repo | `.github/` | — |

## Global Constraints

- **Contract with `davcs86/homelab-edge` (Rule 1):** ports 9210/9212, health paths, `/data`, image names, uid 10001 and the read-only root stay as they are. New env names are exactly the spec's: API `AUTH_MODE`; UI `OIDC_ISSUER`, `OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET`, `OIDC_SCOPES`, `SESSION_SEAL_KEY`, and the refused `OIDC_AUTO_PROVISION`. The CLI is `happybabytracker-api admin <verb>` with the password on stdin.
- **Upgrade safety:** with `AUTH_MODE` unset, every image this plan produces behaves as `local` and as before, with exactly the header's three exceptions: the env admin is imported and its sessions end once; `ADMIN_USERNAME`/`ADMIN_PASSWORD` are then ignored with one warning per start; and the password change counts against the login limiter. Migrations stay forward-only.
- **Rule 2:** the admin is never a row in `parents` and never holds `kid_access`. Admin sessions stay refused outside `/v1/admin/*` and `/v1/session`, and the new session and token routes are `Internal`.
- **Mode table (spec 5.5), verbatim:**
  - `POST /v1/session`: allowed in `local` and `both`, 403 `mode_refused` in `oidc`.
  - `POST /v1/session/oidc`: allowed in `oidc` and `both`.
  - `POST /v1/tokens/{token}/redeem`: allowed in `local` and `both`.
  - `POST /v1/tokens/{token}/redeem-oidc`: allowed in `oidc` and `both`.
  - Re-auth and the identity routes: allowed in `both` only.
  - **Order in every handler (owner, 2026-11-27):** decode the body → mode check → `device_label` (the three session routes) → `iss`, then `sub` (the SSO routes) → everything else (token lookup, limiter, hashing, re-auth window). A refused redemption therefore leaves the token usable, and `POST /v1/session` in `oidc` answers `mode_refused` even with a bad `device_label`.
- **Device label (owner, 2026-11-27):** the SSO starts of `login` and `invite` derive the label with `deviceLabel(request.headers.get('user-agent') ?? '')`, as the password login does, seal it, and send it as `device_label`; the API validates it with `constants.DeviceLabelPattern` (optional, else 422 field `device_label`) and stores it on parent sessions only. No form field is added: the password form has none either.
- **OIDC identities:** `provider` = the issuer URL (exact string), `subject` = `sub`, `secret_hash` NULL. Every OIDC lookup adds `provider <> 'local'`, so a pair can never match a password row. A pair belongs to one principal across both tables, and each principal has at most one OIDC identity per issuer.
- **Callback state (SSO spec § 3, owner 2026-11-27):** `/auth/callback` never reads the session cookie or `locals.session`; what it shows or binds to (purpose, `next`, the username, the session hash, the start time, the device label) comes from the sealed `hbt_oidc` only. For `link` and `invite` it authorizes nothing: `POST /auth/confirm` does, and for `link` it re-checks the live session against the bound hash.
- **Re-authentication and signing keys (SSO spec § 3, owner 2026-11-27):** `link` and `invite` send `prompt=login` and `max_age=1` and refuse an ID token whose `auth_time` is missing or earlier than the start time minus `OIDC_AUTH_TIME_SKEW_S`. The relying party calls `enableNonRepudiationChecks`, so ID-token signatures are verified on a JWKS at most 300 s old, failing closed.
- **Every route change touches `routes.go` and `routeClasses` in the same task** (How to Act 3): 125 keys today, 126 after Task 3, 128 after Task 4, 132 after Task 5, 133 after Task 6. The access matrix runs with `AuthMode = both`, and its fixture seeds an `admins` row whose id the admin session carries. Per-mode refusals are tested apart from the matrix.
- **Rule 5:** no `sub`, `preferred_username`, label, `code`, `state`, invite token, session token, password or client secret in any log line or error string, API or UI. UI SSO refusals log one fixed-shape line, `{"level":"warn","msg":"sso refused","code":<enumerated>}`. The callback's URL is never logged.
- **Rule 6:**
  - `OIDC_CLIENT_SECRET` and `SESSION_SEAL_KEY` are read only under `ui/src/lib/server/`.
  - The link binding is compared with `crypto.timingSafeEqual`.
  - Fixtures carry only fake values, as `STUB_TOKEN` does.
- **Rule 7:** every limit and duration lives in `api/internal/constants/constants.go` or `ui/src/lib/server/constants.ts`, with client-safe mirrors in `ui/src/lib/constants.ts`.
- **Rule 8:** `reauth_at` and `admin_identities.secret_changed_at` are unix seconds UTC in INTEGER columns.
- **Rule 9:** comments are constraints only, at most 2 lines.
- **Rule 10:**
  - `openid-client` is pinned exactly (no `^`) in `ui/package.json`, with `pnpm-lock.yaml` updated in the same commit.
  - It goes in `devDependencies` like every other UI package, so adapter-node bundles it and `pnpm prune --prod` leaves the runtime image as it is.
  - No base-image or action pin changes.
- **The stub mirrors the Go handlers** (`ui/e2e/stub-api.ts`): the same validation order, codes and fields. UI tests stay green on a broken contract, so `scripts/smoke.sh` remains the real check for everything it covers.
- **UI design (`AGENTS.md`, owner 2026-11-27):** every SSO screen is composed from existing parts (`ui/src/lib/components/ui/*`, `FormSheet`, `AppShell`, `navModel`) and existing page patterns (the outside-shell frame of sign-in and the error page, the `Card role="region"` sections of My profile, the admin row actions and the P2 `<details>` confirm). Tokens and DESIGN 5 sizes only; no `dark:` utility, no new transition. `docs/plans/ui-redesign/**` is not edited (Rule 6). Every new element and every new sentence is listed under **Deviations from the design** below, which the execution PR's description carries for owner sign-off. None of these screens is a record form, so none is a sheet (Rule 1).
- **SSO start forms are plain form POSTs** without `use:enhance`: their success is a 303 to the issuer's origin, which an enhanced action would hand to `goto`.
- **Playwright** (plan 6 and 7 lessons):
  - Form saves wait with `submit()`; a test that edits a prefilled field first waits for `untilHydrated`.
  - CI runs the headless shell.
  - Run one spec by its path (`pnpm test:e2e e2e/sso-login.spec.ts`).
  - Every new page passes axe in light and dark (`test.use({ colorScheme })`) at 390×844 (`phone`) and 1280×800 (`desktop`), and works without JavaScript.
  - Every new page keeps `Cache-Control: no-store`.
  - A new field labelled "Password" on `/account` is found with `getByLabel('Password', { exact: true })`: "Current password", "New password" and "Confirm new password" share the substring.
- **Copy:** the spec's fixed sentences are used verbatim ("Single sign-on is unavailable", "Single sign-on did not complete", "Continue with single sign-on", "Set a password", "You are linking <label> to <username>" with Confirm and Cancel, and the label fallback "the single sign-on account you just signed in with"); DESIGN 8.2 and 8.4 sentences where they apply ("Too many attempts, try again in a minute", "This link is no longer valid. Ask the person who sent it for a new one. This kind of link lasts 72 hours."); every other sentence is the one listed under Deviations (Open question 11, resolved).

## Review Focus

Inputs and conditions the spec implies but no happy path exercises. Each one has a test in the task named.

1. **Double submit and replay** (Tasks 10, 11, 12). A second Confirm (double click, or back then resubmit) and a replayed callback URL each meet a cleared `hbt_oidc`. Each answers the "did not complete" page with 200. None issues a second session, adds a second identity or answers 500.
2. **A long or odd `preferred_username`** (Tasks 9, 11). A label of 300 characters, `<script>`, U+202E or a control character:
   - it is cut to `OIDC_LABEL_MAX_LEN` code points, and control and format characters (Cc, Cf) are removed;
   - it renders escaped;
   - the sealed cookie stays under 4096 bytes;
   - it never reaches a log.
   - An empty result falls back to the spec's label.
3. **The session changes between the link start and Confirm** (Task 12). The person signs out, or signs in as another account in another tab, before pressing Confirm. The binding no longer matches, so nothing is linked and the page says it did not complete. An admin session confirming a parent's pending link is refused the same way.
4. **The invite changes under an SSO redemption** (Tasks 4, 11):
   - the admin reissues the invite (`IssueToken` deletes the unused one) between the start and Confirm;
   - the invite expires;
   - the parent is disabled.
   - Each gives 404 `not_found` and the DESIGN 8.4 "This link is no longer valid" sentence, and the reissued token stays usable.
5. **CLI password on stdin** (Task 2):
   - Exactly one trailing `\n` or `\r\n` is stripped. Leading and trailing spaces stay part of the password, and so does an inner `\r`.
   - No newline at all works.
   - Empty stdin is refused, and so are 15 code points of `é` (30 bytes) and 257 code points.
   - Input longer than `CLIPasswordMaxBytes` is refused without hashing.

## Deviations from the design

The design (`docs/plans/ui-redesign/**`) has no SSO screen. Each item below is composed from existing parts; the execution PR's description copies this list under "Deviations from the design", and the owner signs it off (AGENTS.md Rule 5 and "Done means"). Every visible SSO element is absent in `local`, so the designed pages look as today on every deploy that leaves `AUTH_MODE` unset, except items 13, 14 and 17 (the admin nav, the admin Account page and the footnote), which show in every mode.

**New elements**

| # | Where | Element, and the parts it is composed of | Task |
|---|---|---|---|
| 1 | Sign in, `oidc` with relying-party settings | The card holds one form posting to `/auth/sso` (hidden `next`) with the primary `Button size="xl"` "Continue with single sign-on", in place of the password fields | 10 |
| 2 | Sign in, `oidc` without relying-party settings | The card holds an `Alert` "Sign-in is unavailable" and no form | 10 |
| 3 | Sign in, `both` | Below the unchanged password card, a form posting to `/auth/sso` with a full-width outline `Button size="xl"` "Continue with single sign-on" (secondary, so "Sign in" stays the one primary action, DESIGN 1.3) | 10 |
| 4 | "Did not complete" and "unavailable" pages (`/auth/sso`, `/auth/callback`, `/auth/confirm`) | `SsoOutcome.svelte`: the outside-shell frame of the error page (brand line, h1, an optional `Alert` with the reason, one xl `Button href` back to where the flow started) | 10–12 |
| 5 | `/auth/callback` after a `login` | The same frame: h1 "Signing you in", a `<meta http-equiv="refresh">` and an xl `Button href` "Continue" | 10 |
| 6 | Invite, `oidc` | `SetPasswordForm`'s card keeps its read-only Username field; the password fields are replaced by the primary xl "Continue with single sign-on", posting `?/sso` | 11 |
| 7 | Invite, `both` | Below the unchanged card, a form posting `?/sso` with a full-width outline xl "Continue with single sign-on" | 11 |
| 8 | Reset, `oidc` | `SetPasswordForm`'s invalid-link layout: an `Alert` with the reset sentence and the outline xl "Go to sign in" | 11 |
| 9 | `/auth/callback` for `link` and `invite` (the confirmation) | The outside-shell frame: h1 "You are linking <label> to <username>" (both wrap anywhere), then a card form posting to `/auth/confirm` with two submit buttons `name="choice"`: primary xl "Confirm" (`confirm`) and outline xl "Cancel" (`cancel`) | 11 |
| 10 | My profile, `both` with settings and not linked | `SsoSection.svelte`: a `Card role="region"` "Single sign-on" placed in the left column after Password (DESIGN 2.1 column map: "details, password"), holding a form posting to `/account/link-sso`: a `Label` "Password" + `Input type="password" autocomplete="current-password"`, a field message under it, and an outline `Button` "Link single sign-on" | 12 |
| 11 | My profile, linked | The same card with the muted line "Single sign-on is linked." | 12 |
| 12 | `/account/link-sso` and `/admin/account/link-sso` (the no-JS and refusal page of item 10) | The account title line (h1 "Link single sign-on" + hairline, DESIGN 3.3) and the same `SsoSection` card with the refusal | 12 |
| 13 | Admin nav (DESIGN 3.4 and P14 say the admin has no navigation) | `navModel` gains an admin branch with two entries, "Parents" (`shield-check`) and "Account" (`user`). ≥ 768 px: the admin layout's existing side `nav` "Primary" lists them under the brand line with AppShell's side-entry classes. < 768 px: the admin header row shows them as 48 × 48 outline icon links (the bell's pattern, `aria-label`, `title`, `aria-current`), before the "Admin" badge | 12 |
| 14 | `/admin/account` | The account title line (h1 "Account" + hairline); a `Card role="region"` "Details" with `Label` "Username" and the read-only `Input` of `SetPasswordForm`; then `SsoSection` posting to `/admin/account/link-sso` | 12 |
| 15 | Admin parents row, linked | A second `Badge` "Single sign-on" beside the status badge | 12 |
| 16 | Admin parents row, linked | A row action as a P2 destructive confirm, built like Disable: `<details>` with the outline summary "Unlink single sign-on" (+ sr-only name), a card with the consequence list and the destructive `Button` "Unlink single sign-on"; the result in the page's existing action status | 12 |
| 17 | Admin parents footnote | The prototype's second sentence becomes false once the admin is in the database (Open question 19); proposed replacement below | 12 |

**New sentences** (each is new copy; the spec-fixed and DESIGN sentences above are not repeated)

| Sentence | Where | Task |
|---|---|---|
| "Sign-in is unavailable" | item 2 | 10 |
| "Password sign-in is turned off" | sign-in, invite and reset when a password post meets `mode_refused` | 10, 11 |
| "Signing you in" | item 5 heading | 10 |
| "Continue" | item 5 link | 10 |
| "Go to your invite link", "Go to My profile", "Go to Account" | item 4's back link for `invite`, a parent's `link`, the admin's `link` ("Go to sign in" exists in `SetPasswordForm`) | 10–12 |
| "Finish setting up your account" / "Continue with single sign-on to finish." | item 6 heading and lead | 11 |
| "This link works only while password sign-in is on. Ask the person who sent it." | item 8 | 11 |
| "Single sign-on is turned off" | confirm meets `mode_refused` | 11 |
| "That single sign-on account is already linked to an account" | 409 `already_linked` | 11, 12 |
| "Nothing to confirm" | `GET /auth/confirm` | 11 |
| "Single sign-on" | item 10 heading, item 15 badge | 12 |
| "Link single sign-on" | item 10 button, item 12 heading | 12 |
| "Single sign-on is linked." | item 11 | 12 |
| "The password is incorrect" | item 10 field message (403 `wrong_password`) | 12 |
| "Linking is turned off" | item 10, 403 `mode_refused` | 12 |
| "Confirm your password again, then link" | confirm meets 403 `reauth_required` | 12 |
| "Parents", "Account" | item 13 entries, item 14 heading | 12 |
| "Unlink single sign-on" | item 16 summary and button | 12 |
| "{name} is signed out everywhere and their notification devices are removed." / "Their password still works, if they have one." | item 16 consequence list | 12 |
| "Single sign-on unlinked. Their sessions and notification devices were removed." | item 16 result | 12 |
| "The admin’s own username can’t be changed. The operator resets the admin password with the admin command; that signs the admin out." | item 17, replacing "They come from the deployment; changing them signs the admin out." (Open question 19) | 12 |

---

## Task 1: The admin becomes a principal in the database

Migration, store, the start-up import, the login and the username namespace land together: an admin session cannot be created without an `admins` row, so no smaller slice stays green.

**Files:**
- Create: `api/internal/store/migrations/0011_sso.sql`, `api/internal/store/admins.go`, `api/internal/store/admins_test.go`, `api/cmd/happybabytracker-api/admin.go`, `api/cmd/happybabytracker-api/admin_test.go`
- Modify: `api/internal/store/{sessions.go,parents.go,store.go,store_test.go,sessions_test.go,parents_test.go}`, `api/internal/model/model.go`, `api/internal/auth/password.go` (+ `password_test.go`), `api/internal/config/{config.go,config_test.go}`, `api/cmd/happybabytracker-api/{main.go,main_test.go}`, `api/internal/httpapi/{session.go,admin.go,session_test.go,admin_test.go,testkit_test.go,matrix_test.go}`, `ui/e2e/stub-api.ts` (`:796`), `scripts/smoke.sh` (assertion 8, first half)
- Delete: `api/internal/auth/admin.go`, `api/internal/auth/admin_test.go`

**Interfaces:**
- Produces:
  - `model.Admin{ID, Username string; HasPassword, SSOLinked bool; CreatedAt time.Time}`; `model.Principal` gains `AdminID string`.
  - `store.ErrAdminExists`.
  - `(*Store).HasAdmin(ctx) (bool, error)`.
  - `(*Store).CreateFirstAdmin(ctx, username, secretHash string, now time.Time) (model.Admin, error)`: `ErrAdminExists` when `admins` holds a row, `ErrConflict` when a parent holds the name; the local row's `secret_changed_at` is `now`.
  - `(*Store).AdminByUsername(ctx, username string) (model.Admin, string, error)` and `(*Store).AdminByID(ctx, id string) (model.Admin, string, error)`: the second value is the local hash, `""` when none; `ErrNotFound`.
  - `(*Store).CreateSession(ctx, kind, principalID string, tokenHash []byte, now, expires time.Time, deviceLabel string) (string, error)`: `principalID` is the admin id when `kind == "admin"`; the label is stored as today (empty → NULL).
  - `auth.HashAdminPassword(pw string) (string, error)`: `ErrPasswordLength` below `MinAdminPasswordLen` or above `MaxPasswordLen`.
  - `importAdmin(ctx context.Context, st *store.Store, cfg config.Config, log *slog.Logger, now time.Time) error`; `prepare(ctx, st, cfg, log)`.
  - Test kit: `kit.adminID`; `newKitWith` seeds the admin `testAdminUser`/`testAdminPass` when `HasAdmin` is false (the hash computed once per test binary with `sync.OnceValue`); `seedSession(kind, principalID, ttl)`.
- Removes: `DeleteAdminSessions`, `syncAdminFingerprint`, `checkAdminUsername`, `fingerprintKey`, `auth.AdminFingerprint`, `auth.CheckAdmin`, and `config.Load`'s two `ADMIN_*` checks. `Config` keeps the raw `AdminUsername`/`AdminPassword` strings.

- [ ] **Step 1: Write the failing store tests**

In `store_test.go`:
- Add `"admins", "admin_identities"` to `wantTables`, and change the ten `schema_version` assertions (`:36, 54, 400, 549, 749, 856, 975, 1058, 1346, 1431`) from `"10"` to `"11"`.
- `TestMigration0007KeepsRows` (`:956`) and `TestMigration0008KeepsRows` (`:1039`) count every `wantTables` table in the database before the later migrations: both skip `admins` and `admin_identities` there, as they skip `visits`.
- `TestAdminSessionShape` inserts an `admins` row `a`. It then asserts:
  - an admin session with `admin_id = 'a'` and `parent_id` NULL is accepted;
  - an admin session with no `admin_id`, or with a `parent_id`, is refused;
  - a parent session that carries an `admin_id` is refused.
- New `TestMigration0011KeepsRows`, modelled on `TestMigration0010KeepsRows`. It applies `ms[:10]` and seeds a parent session `s1` with `device_label = 'Firefox on Mac'`, an admin session (`kind='admin'`, `parent_id` NULL), a local identity with `secret_changed_at = 100`, and the meta row `admin_fingerprint`. After `Open`:
  - `s1` is kept, with `device_label = 'Firefox on Mac'` and `admin_id` and `reauth_at` NULL;
  - the identity keeps `secret_changed_at = 100`;
  - no admin session is left, and the fingerprint row is gone;
  - every other table keeps its count, and `pragma_foreign_key_check` finds nothing.

In `admins_test.go`:
- `TestCreateFirstAdminOnce`: the second call answers `ErrAdminExists`, and one row is stored; the local `admin_identities` row has `secret_changed_at = now.Unix()`.
- `TestCreateFirstAdminRefusesAParentsName`: `ErrConflict`, and `admins` stays empty.
- `TestCreateParentRefusesTheAdminsName`: `ErrConflict`.
- `TestOIDCIdentityIndexes`: a second non-local identity for the same parent and provider is a constraint error, and the same for the admin in `admin_identities`; two `local` rows are impossible already (PK).

- [ ] **Step 2: Run them to verify they fail**

Run: `cd api && go test ./internal/store/ -run 'TestAdminSessionShape|TestMigration00|TestCreateFirstAdmin|TestCreateParentRefusesTheAdminsName|TestOIDCIdentityIndexes' -race`
Expected: FAIL (no table `admins`, schema version 10).

- [ ] **Step 3: Write `0011_sso.sql`**

The constraints are the decision here, so the SQL is given. `device_label` is the column `0010` added to `sessions`; `sessions` has no index and nothing references it, so the rebuild loses nothing else:

```sql
CREATE TABLE admins (
    id         TEXT PRIMARY KEY,
    username   TEXT NOT NULL UNIQUE,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
) STRICT;

CREATE TABLE admin_identities (
    admin_id          TEXT NOT NULL REFERENCES admins (id) ON DELETE CASCADE,
    provider          TEXT NOT NULL,
    subject           TEXT NOT NULL,
    secret_hash       TEXT,
    secret_changed_at INTEGER,
    PRIMARY KEY (provider, subject)
) STRICT;

CREATE UNIQUE INDEX identities_one_oidc_per_issuer ON identities (parent_id, provider) WHERE provider <> 'local';
CREATE UNIQUE INDEX admin_identities_one_oidc_per_issuer ON admin_identities (admin_id, provider) WHERE provider <> 'local';

CREATE TABLE sessions_new (
    id           TEXT PRIMARY KEY,
    token_hash   BLOB NOT NULL UNIQUE,
    kind         TEXT NOT NULL CHECK (kind IN ('admin', 'parent')),
    parent_id    TEXT REFERENCES parents (id) ON DELETE CASCADE,
    admin_id     TEXT REFERENCES admins (id) ON DELETE CASCADE,
    created_at   INTEGER NOT NULL,
    last_seen_at INTEGER NOT NULL,
    expires_at   INTEGER NOT NULL,
    device_label TEXT,
    reauth_at    INTEGER,
    CHECK ((kind = 'admin' AND admin_id IS NOT NULL AND parent_id IS NULL)
        OR (kind = 'parent' AND parent_id IS NOT NULL AND admin_id IS NULL))
) STRICT;
INSERT INTO sessions_new (id, token_hash, kind, parent_id, created_at, last_seen_at, expires_at, device_label)
    SELECT id, token_hash, kind, parent_id, created_at, last_seen_at, expires_at, device_label
    FROM sessions WHERE kind = 'parent';
DROP TABLE sessions;
ALTER TABLE sessions_new RENAME TO sessions;

DELETE FROM meta WHERE key = 'admin_fingerprint';
```

- [ ] **Step 4: Implement the store side**

- `admins.go`:
  - `CreateFirstAdmin` runs one `inTx` (an immediate transaction on the writer). It checks `SELECT 1 FROM admins` (giving `ErrAdminExists`) and then `parents.username`. It inserts the `admins` row and an `admin_identities` row `('local', username, hash, now)`.
  - `AdminBy*` select the local hash and `SSOLinked` as `EXISTS (… provider <> 'local')`.
- `CreateParent` becomes an `inTx` that checks `admins.username` first (giving `ErrConflict`).
- `CreateSession` writes `admin_id` for admin sessions and `parent_id` for parent sessions. `SessionByToken` scans `admin_id` into `Principal.AdminID`.
- Delete `DeleteAdminSessions`.

- [ ] **Step 5: Run the store tests to verify they pass**

Run: `cd api && go test ./internal/store/ -race`
Expected: PASS.

- [ ] **Step 6: Write the failing start-up and login tests**

`admin_test.go` (package `main`) replaces `TestAdminFingerprintRotation`, `TestFirstStartRecordsTheFingerprint`, `TestAdminUsernameCollidingWithAParentIsRefused` and `TestConfigNeverHoldsAnUnnormalisedAdminUsername`, which are deleted from `main_test.go`:
- `TestImportCreatesTheAdminOnce`: a fresh dir and `baseEnv`. After `start`, `HasAdmin` is true and `AdminByUsername("root-admin")` verifies `admin-password-AAAAAAAA`.
- `TestImportIsIgnoredOnceAnAdminExists`: a second `start` with a different `ADMIN_PASSWORD`:
  - the old password still verifies;
  - stdout holds exactly one line with `"level":"WARN"` and `"msg":"admin credentials in the environment are ignored"`;
  - neither value appears in it.
  - A third start with both variables unset logs no warning.
- `TestImportRequiresTheVariablesWhileEmpty`: each case fails `start` naming the variable, never echoing the value:
  - `ADMIN_USERNAME` unset, `"David"`, `" david "` or `"qz"`;
  - `ADMIN_PASSWORD` unset, 15 runes, or 15 runes of `é`.
- `TestImportRefusesAParentsName`: replaces `TestRunRefusesToStartOnACollision` (`main_test.go:205`). The error is `collisionMsg`, the name is not echoed, and `admins` stays empty.

`config_test.go`:
- `TestLoadRefuses` loses its `ADMIN_*` cases.
- `TestLoadValid` still carries the raw strings.
- New `TestLoadIgnoresAdminVariables`: `ADMIN_USERNAME` unset, or `"David"`, loads.

`session_test.go`:
- `TestAdminLogin`, `TestAdminLoginWithMixedCaseName` and the limiter tests keep passing against the seeded row.
- The admin success now runs **1** verification (`:136`, "want 0" becomes "want 1"). A new case checks that a parent whose name is not an admin's runs exactly one too.

`admin_test.go` (httpapi):
- `:87-92`: the admin's own username answers **409 `username_taken`** (spec 5.5, "as today" for parents).

`matrix_test.go`:
- `newMatrixFixture` creates the admin with `CreateFirstAdmin` and seeds its session with that id (`:309`).

- [ ] **Step 7: Run them to verify they fail**

Run: `cd api && go test ./cmd/... ./internal/config/ ./internal/httpapi/ -race`
Expected: FAIL (`importAdmin` undefined; the admin still verified against the env).

- [ ] **Step 8: Implement**

- `importAdmin`:
  - If `HasAdmin`: log the one warning when either variable is non-empty, and return nil.
  - Else: the raw `cfg.AdminUsername` must match `constants.UsernamePattern`, and `auth.HashAdminPassword(cfg.AdminPassword)` must succeed. Each error names the variable.
  - Then `CreateFirstAdmin(…, now)`: `ErrConflict` becomes `errAdminCollision`; `ErrAdminExists` (the CLI won the race) returns nil.
  - `prepare` calls it in place of the two removed functions.
- `checkCredentials(r, username, valid, password) (kind, principalID string, authed bool, err error)`: `AdminByUsername`, then `ParentByUsername`. It runs one `verifyPassword` on every path, against `dummyHash()` when no hash was found. A disabled parent fails after the verify. `login` keeps dropping the label for an admin session.
- `createParent` drops the `d.Cfg.AdminUsername` comparison (`admin.go:65`).
- `testkit_test.go` seeds the admin and keeps `asAdmin()` returning a valid admin token.
- Stub: `createParent` answers 409 `username_taken` for `ADMIN.username` (`:796`).

- [ ] **Step 9: Run the API pipeline**

Run: `cd api && gofmt -l . && go vet ./... && GOTOOLCHAIN=$(go env GOVERSION) go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./... && go test ./... -race`
Expected: `gofmt -l` prints nothing; PASS.

- [ ] **Step 10: Smoke assertion 8, first half**

Replace assertion 8's body (`:1650-1664`). After `start_api "$ADMIN_PASSWORD_2"`:
- the old admin cookie still gets 200 on `/admin/parents`;
- a login with `$ADMIN_PASSWORD_2` answers 400 (through `form`, not `login`);
- `docker logs` of this process holds exactly one line containing `admin credentials in the environment are ignored`.
- `pass "8 a restart with a new ADMIN_PASSWORD keeps the old admin password and session, with one warning"`.

Run: `docker build -t hbt-api:dev api && docker build -t hbt-ui:dev ui && scripts/smoke.sh hbt-api:dev hbt-ui:dev`
Expected: every line `PASS:`.

- [ ] **Step 11: Commit**

```bash
git add api ui/e2e/stub-api.ts scripts/smoke.sh
git commit -m "feat(api): admin principal in its own tables, imported once from the env"
```

## Task 2: The admin CLI

**Files:**
- Create: `api/cmd/happybabytracker-api/cli.go`, `api/cmd/happybabytracker-api/cli_test.go`
- Modify: `api/cmd/happybabytracker-api/main.go` (dispatch), `api/internal/store/admins.go` (+ test), `api/internal/constants/constants.go`, `scripts/smoke.sh` (assertion 8, second half; assertion 9's log list)

**Interfaces:**
- Consumes: `CreateFirstAdmin`, `AdminByUsername`, `auth.HashAdminPassword`, `importAdmin` (Task 1).
- Produces:
  - `runAdmin(ctx context.Context, args []string, getenv func(string) string, stdin io.Reader, stdout, stderr io.Writer, now time.Time) int`: exit 0 on success, 1 on a refusal or error, 2 on a usage error.
  - `var hashAdminPassword = auth.HashAdminPassword` in `cli.go`: the seam the tests replace to count KDF calls (`auth`'s `idKey` is unexported).
  - `(*Store).SetAdminPassword(ctx, username, secretHash string, now time.Time) error`: upserts the local identity with `secret_changed_at = now` and deletes that admin's sessions; `ErrNotFound`.
  - `(*Store).UnlinkAdminSSO(ctx, username string) error`: deletes that admin's non-local identities and sessions; `ErrNotFound`.
  - `constants.CLIPasswordMaxBytes = 4*MaxPasswordLen + 2`.
- `main()`: when `len(os.Args) > 1 && os.Args[1] == "admin"`, it calls `os.Exit(runAdmin(…))` and never reaches `run`.
- The CLI reads only `DATA_DIR` (default `constants.DefaultDataDir`). It opens the store with `store.Open`, so pending migrations apply and `create` works on a fresh volume. It never reads `ADMIN_*`, `AUTH_MODE` or any secret.
- Output: JSON lines through `slog`, carrying the verb and the outcome, never the username or the password (Rule 5). The messages are:
  - `admin created`;
  - `admin password reset; its sessions were ended`;
  - `admin single sign-on unlinked; its sessions were ended`;
  - refusals `an admin already exists`, `no admin with that username`, `that username belongs to a parent`, `the password must be 16 to 256 characters`;
  - and the usage line.

- [ ] **Step 1: Write the failing tests**

`cli_test.go`:
- `TestCreateOnAFreshVolume`: exit 0, one admin, and the password verifies.
- `TestCreateRefusedWhenAnAdminExists`: exit 1, and the row is unchanged.
- `TestCreateRefusesAParentsName`: exit 1.
- `TestResetPassword`: seed two admin sessions, run with a new password, exit 0. Both sessions are gone, the new password verifies and the old one does not, and the local row's `secret_changed_at` is `now`. A parent session survives.
- `TestResetPasswordUnknownAdmin`: exit 1.
- `TestUnlinkSSO`: seed one OIDC row in `admin_identities` and a session. Exit 0; the OIDC row and the session are gone, and the local row stays. An unknown name gives exit 1.
- `TestUsage`: no verb, an unknown verb, a missing or extra argument, or an invalid username pattern each give exit 2.
- `TestPasswordOnStdin`: Review Focus 5. A table of stdin values and the password stored:

| stdin | Stored or refused |
|---|---|
| `"pw…16\n"` | `pw…16` |
| `"pw…16\r\n"` | `pw…16` |
| `"pw…16"` | `pw…16` |
| `" pw…16 \n"` | `" pw…16 "` |
| `"pw\r…16\n"` | inner `\r` kept |
| `""` | refused, 1 |
| 15 × `é` + `"\n"` | refused, 1 |
| 257 runes | refused, 1 |
| `CLIPasswordMaxBytes+1` bytes | refused, 1, `hashAdminPassword` never called |

- `TestCLIOutputCarriesNoName`: neither the username nor the password appears in stdout or stderr for any verb.
- `TestCreateRacesTheImport`: 20 rounds. Each round opens two `*store.Store` on one fresh dir, runs `runAdmin(create)` and `importAdmin` behind a barrier, and asserts:
  - exactly one `admins` row;
  - if the import won, `create` exited 1 with `an admin already exists`;
  - if `create` won, the import returned nil and logged one warning.

- [ ] **Step 2: Run them to verify they fail**

Run: `cd api && go test ./cmd/... -run 'TestCreate|TestReset|TestUnlink|TestUsage|TestPasswordOnStdin|TestCLIOutput' -race`
Expected: FAIL (`runAdmin` undefined).

- [ ] **Step 3: Implement `runAdmin` and the two store functions**

The password is read through `io.LimitReader(stdin, CLIPasswordMaxBytes+1)`. More than `CLIPasswordMaxBytes` is refused. One trailing `\r\n` or `\n` is stripped, then `hashAdminPassword` runs. Both store functions are one `inTx` each.

- [ ] **Step 4: Run the API pipeline**

Run: the Task 1 Step 9 command. Expected: PASS.

- [ ] **Step 5: Smoke assertion 8, second half**

After the first half:
- `printf '%s\n' "$ADMIN_PASSWORD_2" | docker run --rm -i --network none --read-only --cap-drop ALL --security-opt no-new-privileges -v "$VOLUME":/data "$API_IMAGE" admin reset-password "$ADMIN_USERNAME" >"$WORK/cli.log" 2>&1` exits 0, while the API keeps running.
- The old admin cookie then gets `303 /login` with `hbt_session=;…Max-Age=0`, the existing check.
- `NEW_ADMIN="$(login admin-login-2 … "$ADMIN_PASSWORD_2")"` succeeds.
- Add `cli.log` to assertion 9's file list (`:1723`).
- `pass "8 the CLI's reset-password ends the old admin session and the new password logs in"`.

Run: the image builds and `scripts/smoke.sh hbt-api:dev hbt-ui:dev`. Expected: every line `PASS:`.

- [ ] **Step 6: Commit**

```bash
git add api scripts/smoke.sh
git commit -m "feat(api): admin CLI: create, reset-password, unlink-sso"
```

## Task 3: `AUTH_MODE`, `GET /v1/auth` and the mode gates on the password routes

**Files:**
- Modify: `api/internal/config/{config.go,config_test.go}`, `api/internal/httpapi/{routes.go,session.go,admin.go,session_test.go,admin_test.go,matrix_test.go}`, `api/cmd/happybabytracker-api/main_test.go`, `scripts/smoke.sh` (one `refused` line)
- Create: `api/internal/httpapi/authmode.go`, `api/internal/httpapi/authmode_test.go`

**Interfaces:**
- Produces:
  - `config.AuthMode` (string type), `config.AuthLocal = "local"`, `config.AuthOIDC = "oidc"`, `config.AuthBoth = "both"`; `Config.AuthMode`. Unset means `local`; any other value fails `Load` with `AUTH_MODE must be local, oidc or both`.
  - `func (d Deps) allowMode(w http.ResponseWriter, allowed ...config.AuthMode) bool`: writes 403 `mode_refused`.
  - Route `{GET, "/v1/auth", Internal, d.authMode}` answers `{"mode":"local"}`.
  - The matrix fixture sets `k.Cfg.AuthMode = config.AuthBoth` and rebuilds `k.Handler`.
  - `routeClasses` gains `"GET /v1/auth": Internal` (126 keys).

- [ ] **Step 1: Write the failing tests**

- `TestLoadAuthMode`: `""` gives `local`; `local`, `oidc` and `both` load. `Local`, `OIDC`, `none` and `both ` are refused, each error naming `AUTH_MODE`.
- `TestAuthModeRoute`: each mode is returned as-is; without the internal token it is 401.
- `TestPasswordLoginRefusedInOIDC`: in `oidc`, `POST /v1/session` with valid parent credentials gives 403 `mode_refused`. It runs **zero** password verifications, and the limiter is not counted: six refused calls from one address, then the mode switched to `both`, and the next login succeeds. With `device_label: "bad!"` in `oidc` it is still 403 `mode_refused` (the mode before the label); in `both` the same body gives 422 field `device_label`.
- `TestPasswordRedeemRefusedInOIDCLeavesTheTokenUsable`: in `oidc`, `POST /v1/tokens/{invite}/redeem` gives 403 `mode_refused`. In `both`, the same token then redeems (200). The same holds for a reset token.
- A body error still wins over the mode: `{"x":1}` in `oidc` gives 422 `unknown_field`.
- The matrix runs in `both`.

- [ ] **Step 2: Run them to verify they fail**

Run: `cd api && go test ./internal/config/ ./internal/httpapi/ -run 'AuthMode|RefusedInOIDC|TestAccessMatrix' -race`
Expected: FAIL.

- [ ] **Step 3: Implement**

`login` and `redeemToken` each call `allowMode(w, AuthLocal, AuthBoth)` right after `decodeJSON`; in `login` that moves the mode check ahead of the existing `device_label` check (`session.go:83`).

- [ ] **Step 4: Run the API pipeline**

Run: the Task 1 Step 9 command. Expected: PASS.

- [ ] **Step 5: Smoke**

Add `refused AUTH_MODE bogus AUTH_MODE=bogus` beside the other `refused` lines (`:1068-1074`; it fails in `config.Load`, before the store opens). Run the image builds and the smoke. Expected: every line `PASS:`.

- [ ] **Step 6: Commit**

```bash
git add api scripts/smoke.sh
git commit -m "feat(api): AUTH_MODE with GET /v1/auth; password routes refused in oidc"
```

## Task 4: SSO sessions and invite redemption by SSO

**Files:**
- Create: `api/internal/store/identities.go`, `api/internal/store/identities_test.go`, `api/internal/httpapi/oidc.go`, `api/internal/httpapi/oidc_test.go`
- Modify: `api/internal/store/store.go` (errors), `api/internal/constants/constants.go`, `api/internal/httpapi/{routes.go,matrix_test.go}`

**Interfaces:**
- Consumes: `CreateSession` semantics, `deleteChannels`, the token columns, `deviceLabelRE` (`session.go:19`).
- Produces:
  - `store.ErrNotLinked` and `store.ErrAlreadyLinked`.
  - `(*Store).OIDCLogin(ctx, iss, sub string, tokenHash []byte, now, parentExpires, adminExpires time.Time, deviceLabel string) (kind string, expires time.Time, err error)`, in one write transaction:
    - It looks the pair up in `identities` (`provider = iss AND provider <> 'local' AND subject = sub`), then in `admin_identities`.
    - It creates the session of the principal found; a parent session stores `deviceLabel`, an admin session none.
    - For a **disabled** linked parent it deletes that parent's sessions, push subscriptions and ntfy topic, **commits**, and returns `ErrNotLinked`. An unknown pair also returns `ErrNotLinked`.
  - `(*Store).RedeemInviteOIDC(ctx, tokenHash []byte, iss, sub string, sessionHash []byte, now, expires time.Time, deviceLabel string) error`, in one transaction:
    1. Marks used an unused, unexpired `kind = 'invite'` token of an enabled parent; otherwise `ErrNotFound`. A reset token is `ErrNotFound`.
    2. `ErrAlreadyLinked` when the pair is in either table or the parent already has an identity for `iss`.
    3. Inserts the identity, deletes all that parent's existing sessions and inserts the new one with `deviceLabel` (owner, 2026-11-27, Open question 4). Password redemption deletes them the same way: `RedeemToken` calls `setPassword(ctx, tx, parentID, secretHash, "", now)`, whose `DELETE FROM sessions WHERE parent_id = ? AND id <> ?` keeps none (`store/tokens.go:81`, `store/parents.go:138`).
    - Any error rolls back, so the token stays usable.
  - `pairTaken(ctx, tx *sql.Tx, iss, sub string) (bool, error)`, shared with Task 5.
  - `constants.OIDCIssuerMaxLen = 512` and `constants.OIDCSubjectMaxLen = 255` (Open question 6).
  - Routes:
    - `{POST, "/v1/session/oidc", Internal, d.oidcLogin}`: body `{iss, sub, device_label}` (`device_label` optional), answers `loginResponse`.
    - `{POST, "/v1/tokens/{token}/redeem-oidc", Internal, d.redeemTokenOIDC}`: same body, answers a parent `loginResponse`.
    - `routeClasses` 128 keys.
- Handler order (Global Constraints): `decodeJSON` → `allowMode(AuthOIDC, AuthBoth)` → validation:
  - `device_label` gives 422 field `device_label` unless empty or matching `deviceLabelRE`;
  - `iss` gives 422 field `iss` unless it is an absolute `https:` URL of at most 512 bytes;
  - `sub` gives 422 field `sub` unless it is 1–255 bytes.
  - Then the store. `ErrNotLinked` gives 403 `not_linked`, `ErrNotFound` gives 404 `not_found`, `ErrAlreadyLinked` gives 409 `already_linked`.
  - Session lifetimes: `ParentSessionTTL` and `AdminSessionTTL`.

- [ ] **Step 1: Write the failing tests**

Store tests (`identities_test.go`):
- `TestOIDCLoginParentAndAdmin`: the parent session stores the label; the admin session stores none.
- `TestOIDCLoginNeverMatchesALocalRow`: `iss = "local"`, `sub = "ana"` gives `ErrNotLinked`.
- `TestOIDCLoginDisabledParentPurges`: seed sessions, a push device and an ntfy topic. The call gives `ErrNotLinked`, and all three are gone. A second call is idempotent.
- `TestRedeemInviteOIDC`: the new session carries the label.
- `TestRedeemInviteOIDCRefusals`: a reset token, a used token, an expired token, a disabled parent, and a token the admin replaced with `IssueToken` (Review Focus 4) each give `ErrNotFound`.
- `TestRedeemInviteOIDCAlreadyLinked`: the pair is held by another parent, or by the admin, or the parent already has this issuer. Each gives `ErrAlreadyLinked`, and the token stays redeemable afterwards.
- `TestRedeemInviteOIDCEndsOtherSessions`: seed two sessions for the invited parent and one for another parent. After the call the invited parent holds exactly one session, the one whose hash was passed, and the other parent's session is kept. A refused call (`ErrAlreadyLinked`) leaves both seeded sessions in place.
- `TestPairBelongsToOnePrincipal`: an index-level backstop. A pair inserted under a parent cannot be inserted under the admin through any store function.

Handler tests (`oidc_test.go`):
- `TestOIDCLoginModes`: `local` gives 403 `mode_refused`; `oidc` and `both` give 200 for a linked pair. The parent cookie lifetime is 30 days and the admin's 12 hours. An unknown pair and a disabled parent give the same 403 `not_linked` body, byte for byte.
- `TestRedeemOIDCModes`: in `local`, 403 `mode_refused`, and the token then redeems by password in `local`.
- `TestRedeemOIDCEndsOtherSessions`: a session token the parent held before the redemption then answers 401 `no_session` on `GET /v1/session`; the returned token answers 200.
- `TestOIDCValidation`, for both routes: `iss` empty, `http://…`, 513 bytes or `local`; `sub` empty or 256 bytes; `device_label` `"bad!"` or 49 characters. Each gives 422 with the field.
- `TestOIDCValidationOrder`, for both routes: in `local`, a body with a bad `device_label`, `iss` and `sub` gives 403 `mode_refused`; in `oidc` it gives 422 `device_label`; with a good label, 422 `iss`; with a good `iss`, 422 `sub`.
- `TestOIDCSessionsListTheDeviceLabel`: after `/v1/session/oidc` and after `redeem-oidc` with `device_label: "Firefox on Mac"`, `GET /v1/me/sessions` lists that label on the current session; without `device_label` it is null.
- `TestAdminAfterSSOLoginIsRefusedOnKidRoutes`: an admin session from `/v1/session/oidc` gets 403 on `GET /v1/kids` and on a kid route (Rule 2; the matrix covers the class).
- `TestOIDCLogCarriesNoSubject`: `wantCleanLog` with `sub`, `iss`, the label and the token banned.

- [ ] **Step 2: Run them to verify they fail**

Run: `cd api && go test ./internal/store/ ./internal/httpapi/ -run 'OIDC|RedeemInviteOIDC|PairBelongs|AdminAfterSSO|TestAccessMatrix' -race`
Expected: FAIL.

- [ ] **Step 3: Implement the store functions, the handlers and the two routes, with their `routeClasses` rows**

`OIDCLogin` uses an explicit `BeginTx`, because the purge must commit before the error is returned; `inTx` would roll it back.

- [ ] **Step 4: Run the API pipeline**

Run: the Task 1 Step 9 command. Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add api
git commit -m "feat(api): SSO session and invite redemption by SSO"
```

## Task 5: Re-auth, linking, the password-change limiter and the session views

**Files:**
- Modify: `api/internal/store/{identities.go,sessions.go}` (+ tests), `api/internal/model/model.go`, `api/internal/constants/constants.go`, `api/internal/httpapi/{routes.go,me.go,session.go,oidc.go,me_test.go,session_test.go,oidc_test.go,matrix_test.go}`

**Interfaces:**
- Consumes: `pairTaken`, `ErrAlreadyLinked` (Task 4); `allowMode` (Task 3); `AdminByID` (Task 1).
- Produces:
  - `model.Principal.ReauthAt time.Time` (zero when NULL); `model.Parent.SSOLinked bool`, computed in `parentSelect` as `EXISTS (… provider <> 'local')`.
  - `(*Store).StampReauth(ctx, sessionID string, now time.Time) error`.
  - `(*Store).LinkIdentity(ctx, p model.Principal, iss, sub string) error`: picks `identities` or `admin_identities` by `p.Kind`; `ErrAlreadyLinked`.
  - `constants.ReauthWindow = 5 * time.Minute`.
  - Routes, `routeClasses` 132 keys:
    - `{POST, "/v1/me/reauth", Parent, d.reauth}` and `{POST, "/v1/admin/me/reauth", Admin, d.reauth}`, body `{password}`, 204;
    - `{POST, "/v1/me/identities", Parent, d.linkIdentity}` and `{POST, "/v1/admin/me/identities", Admin, d.linkIdentity}`, body `{iss, sub}`, 204.
  - Session views:
    - `parentView` gains `sso_linked`, so `GET /v1/session` and `PATCH /v1/me` both carry it;
    - the admin view of `GET /v1/session` gains `"admin": {"username": …, "sso_linked": …}`, read with `AdminByID(p.AdminID)`.
- `reauth`, in order:
  1. Decode.
  2. `allowMode(AuthBoth)`.
  3. The limiter: `addr:<X-Client-Address>`, then `user:<the principal's username>` (429 `rate_limited` with `Retry-After`).
  4. One `verifyPassword` against the principal's local hash, or `dummyHash()` when there is none: wrong is 403 `wrong_password`.
  5. `StampReauth`.
- `linkIdentity`, in order:
  1. Decode.
  2. `allowMode(AuthBoth)`.
  3. The window: `p.ReauthAt` zero or `now.Sub(p.ReauthAt) > ReauthWindow` gives 403 `reauth_required`.
  4. `iss`/`sub` validation as in Task 4.
  5. `LinkIdentity`: 409 `already_linked`.
  - A link does not clear `reauth_at` (Open question 15).
- `changePassword` counts `addr:` and `user:` after decode, before the verify: 429 `rate_limited`.

- [ ] **Step 1: Write the failing tests**

- `TestReauthModes`: `local` and `oidc` give 403 `mode_refused`, with no verification and no limiter count. `both` gives 204, and `reauth_at` is stored as unix seconds.
- `TestReauthWrongPassword`: 403 `wrong_password`. A parent without a password also gives 403 after exactly one verification.
- `TestReauthCountsAgainstTheLoginLimiter`: five failed logins from an address, then a re-auth from it, gives 429. Five failed re-auths for one username, then a login for that username from another address, gives 429.
- `TestChangePasswordCountsAgainstTheLimiter`: the same two directions, 429.
- `TestLinkWindow`: 204 when `reauth_at` is 5 minutes old; 403 `reauth_required` when it is 5 minutes and 1 second old, and when it was never set.
- `TestLinkAlreadyLinked`: the pair is held by another parent, or by the admin, or the caller already has this issuer. Each gives 409.
- `TestLinkIsPerSession`: a re-auth on session A does not let session B of the same parent link (403 `reauth_required`).
- `TestAdminLinkUsesAdminIdentities`: after an admin link, `/v1/session/oidc` with that pair gives an admin session.
- `TestSessionViewsCarrySSOLinked`, updating the existing key-set checks: the parent view at `session_test.go:278` (`TestSessionShape`) and `me_test.go:43` (the `parentObject` helper) becomes `id`, `username`, `display_name`, `units`, `password_changed_at`, `sso_linked`; the admin view at `session_test.go:266` becomes `kind`, `expires_at`, `admin`, and `admin` holds `username`, `sso_linked`. The value flips after a link.
- The matrix: the four routes answer 422 to `matrixBody` before the mode check.

- [ ] **Step 2: Run them to verify they fail**

Run: `cd api && go test ./internal/httpapi/ ./internal/store/ -run 'Reauth|ChangePasswordCounts|Link|SessionViews|TestSessionShape|TestUpdateProfile|TestAccessMatrix' -race`
Expected: FAIL.

- [ ] **Step 3: Implement the store functions, the handlers, the four routes with their `routeClasses` rows, and the views**

- [ ] **Step 4: Run the API pipeline**

Run: the Task 1 Step 9 command. Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add api
git commit -m "feat(api): re-auth, explicit SSO linking, limiter on password change"
```

## Task 6: The admin unlinks a parent's SSO

**Files:**
- Modify: `api/internal/store/identities.go` (+ test), `api/internal/httpapi/{routes.go,admin.go,admin_test.go,matrix_test.go}`

**Interfaces:**
- Produces:
  - `(*Store).UnlinkParentSSO(ctx, parentID string) error`, one transaction: `ErrNotFound` for an unknown parent; otherwise it deletes the parent's non-local identities, sessions, push subscriptions and ntfy topic (`deleteChannels`), even when none was linked (Open question 5).
  - Route `{POST, "/v1/admin/parents/{parent}/unlink-sso", Admin, d.unlinkParentSSO}`, 204; 404 `not_found`. `routeClasses` 133 keys.
  - `adminParentView` gains `sso_linked`.

- [ ] **Step 1: Write the failing tests**

- `TestUnlinkParentSSO`: seed an OIDC identity, two sessions, a push device with a kid opt-in, and an ntfy topic with an opt-in. The answer is 204. The identity, sessions, device, topic and opt-ins are gone; the local identity stays and the password still logs in. `/v1/session/oidc` with the old pair then gives 403 `not_linked`.
- `TestUnlinkUnknownParent`: 404, byte-identical to `resetLink`'s 404.
- The existing key-set checks at `admin_test.go:75` (`TestAdminCreatesParent`) and `:315` (`TestAdminListShowsNoKidData`) gain `sso_linked`.
- The matrix: `{parent}` is filled as for the other admin parent routes.

- [ ] **Step 2: Run them to verify they fail**

Run: `cd api && go test ./internal/httpapi/ ./internal/store/ -run 'Unlink|TestAdminCreatesParent|TestAdminListShowsNoKidData|TestAccessMatrix' -race`
Expected: FAIL.

- [ ] **Step 3: Implement**

- [ ] **Step 4: Run the API pipeline**

Run: the Task 1 Step 9 command. Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add api
git commit -m "feat(api): admin unlinks a parent's SSO, purging sessions and channels"
```

## Task 7: UI relying-party settings and the sealed transaction cookie

**Files:**
- Modify: `ui/src/env.ts`, `ui/src/lib/server/{env.ts,env.test.ts,constants.ts,cookie.ts,cookie.test.ts}`, `ui/src/lib/constants.ts`
- Create: `ui/src/lib/server/oidc/seal.ts`, `ui/src/lib/server/oidc/seal.test.ts`, `ui/src/lib/server/oidc/types.ts`

**Interfaces:**
- Produces:
  - `type OidcEnv = { issuer: string; clientId: string; clientSecret: string; scopes: string; sealKey: string }`; `Env` gains `oidc: OidcEnv | null`.
  - `src/env.ts` declares the six variables with a schema that keeps `undefined` (`{ schema: (v: string | undefined) => v }`), not through `optional`, so presence is detectable (Open question 7: present counts as set, even when empty).
  - `loadEnv` adds these to `bad` (variable names only), "present" meaning `!== undefined`:
    - one or two of `OIDC_ISSUER`, `OIDC_CLIENT_ID` and `OIDC_CLIENT_SECRET` present: the absent names (`OIDC_ISSUER=''` alone names the other two);
    - `OIDC_ISSUER` present and `SESSION_SEAL_KEY` absent or shorter than `MIN_SESSION_SEAL_KEY` code points (`''` included): `SESSION_SEAL_KEY`;
    - `OIDC_SCOPES` present and `''`: `OIDC_SCOPES`;
    - `OIDC_AUTO_PROVISION` present, `''` and `'false'` included: `OIDC_AUTO_PROVISION`.
    - `OIDC_SCOPES` absent gives `OIDC_DEFAULT_SCOPES`.
    - With none of the three present, `oidc` is null, and `SESSION_SEAL_KEY` and a non-empty `OIDC_SCOPES` are ignored (Open question 18).
  - `ui/src/lib/server/constants.ts`:
    - `OIDC_TXN_COOKIE = 'hbt_oidc'`, `OIDC_TXN_MAX_AGE_S = 600`, `OIDC_TXN_PATH = '/auth'`;
    - `MIN_SESSION_SEAL_KEY = 32`, `OIDC_DEFAULT_SCOPES = 'openid profile'`, `OIDC_CALLBACK_PATH = '/auth/callback'`;
    - `OIDC_TIMEOUT_MS = 5000` (a call), `OIDC_LABEL_MAX_LEN = 64` (a call), `OIDC_COOKIE_MAX_BYTES = 4096`;
    - `OIDC_AUTH_TIME_SKEW_S = 30` (owner, 2026-11-27: the `auth_time` skew of SSO spec § 3 "Linking").
  - `ui/src/lib/constants.ts`: `AUTH_MODES = ['local', 'oidc', 'both'] as const`.
  - `oidc/types.ts`:
    - `type OidcPurpose = 'login' | 'invite' | 'link'`;
    - `type OidcStart = { stage: 'start'; purpose: OidcPurpose; state: string; nonce: string; verifier: string; startedAt: number; next: string | null; binding: string | null; username: string | null; kind: SessionKind | null; deviceLabel: string | null; exp: number }`;
    - `type OidcPending = { stage: 'pending'; purpose: 'invite' | 'link'; binding: string; username: string; kind: SessionKind | null; iss: string; sub: string; label: string | null; deviceLabel: string | null; exp: number }`;
    - `type OidcCookie = OidcStart | OidcPending`.
    - `startedAt` is the flow's start time in unix seconds, for the `auth_time` check (Task 9).
    - `username` in `OidcStart` is filled for `link` with the account's username, read from the session at the start (owner, 2026-11-27, Open question 1), so the callback never needs the session cookie.
    - `deviceLabel` is set for `login` and `invite` (owner, 2026-11-27) and null for `link`.
    - `kind` is the linking principal's kind for `link` (null otherwise), so the callback and a confirm refusal can link back to `/account` or `/admin/account` without the session.
  - `oidc/seal.ts`:
    - `seal(sealKey: string, payload: OidcCookie): string`.
    - `open(sealKey: string, value: string | undefined, nowS: number): OidcCookie | null`.
    - The key is `hkdfSync('sha256', sealKey, '', 'hbt_oidc v1', 32)`. The cipher is AES-256-GCM with a random 12-byte IV and AAD `hbt_oidc`. The value is base64url of `iv ‖ ciphertext ‖ tag`.
    - `open` returns null on any decode, auth or JSON failure, on an unknown `stage`, and when `exp <= nowS`.
  - `cookie.ts`: `setOidcCookie(cookies, value, maxAgeS)` and `clearOidcCookie(cookies)` with `{ path: OIDC_TXN_PATH, httpOnly: true, secure: true, sameSite: 'lax' }`.

- [ ] **Step 1: Write the failing tests**

- `env.test.ts`: each refusal above names its variable and never its value. `OIDC_ISSUER` without a seal key is refused, and so is `SESSION_SEAL_KEY=''` with the three present. `OIDC_ISSUER=''` alone is refused naming `OIDC_CLIENT_ID` and `OIDC_CLIENT_SECRET`. `OIDC_SCOPES=''` is refused. `OIDC_AUTO_PROVISION` set to `''`, `'false'` or `'true'` is refused. No OIDC variable gives `oidc: null`. All five with `OIDC_SCOPES` absent give the default scopes.
- `seal.test.ts`:
  - a round trip;
  - a flipped byte, a value sealed with another key, a truncated value, `exp` in the past, and `stage: 'other'` each give null;
  - two seals of the same payload differ (random IV);
  - a pending payload with a 64-character label, a 48-character device label and 400-character `iss`/`sub` stays under `OIDC_COOKIE_MAX_BYTES` with its attributes;
  - a `link` start payload with a 32-character username, a 43-character binding and `startedAt` round-trips, and stays under `OIDC_COOKIE_MAX_BYTES`;
  - a start payload without a numeric `startedAt` gives null.
- `cookie.test.ts`: the attributes.

- [ ] **Step 2: Run them to verify they fail**

Run: `cd ui && pnpm test`
Expected: FAIL.

- [ ] **Step 3: Implement**

- [ ] **Step 4: Run the UI pipeline**

Run: `cd ui && pnpm install --frozen-lockfile && pnpm check && pnpm lint && pnpm test && pnpm test:e2e`
Expected: all pass (the e2e suite is unchanged; the e2e env sets no OIDC variable yet). No visible change, so no screenshots.

- [ ] **Step 5: Commit**

```bash
git add ui
git commit -m "feat(ui): relying-party settings and the sealed OIDC transaction cookie"
```

## Task 8: UI API client and stub parity for the new routes

**Files:**
- Modify: `ui/src/lib/server/{api.ts,api.test.ts}`, `ui/src/lib/types.ts`, `ui/e2e/stub-api.ts`
- Create: `ui/e2e/stub-sso.spec.ts`

**Interfaces:**
- Consumes: the API routes of Tasks 3–6.
- Produces:
  - Types: `type AuthMode = (typeof AUTH_MODES)[number]`; `Parent` gains `ssoLinked: boolean`; `AdminParent` gains `ssoLinked: boolean`; `Session = { kind: SessionKind; parent?: Parent; admin?: { username: string; ssoLinked: boolean } }`.
  - New `Endpoint` values: `auth_mode`, `oidc_session`, `redeem_token_oidc`, `reauth`, `admin_reauth`, `link_identity`, `admin_link_identity`, `unlink_parent_sso`.
  - Client methods:
    - `authMode(): Promise<ApiResult<AuthMode>>`;
    - `oidcSession(iss: string, sub: string, deviceLabel?: string): Promise<ApiResult<NewSession>>`;
    - `redeemTokenOidc(token: string, iss: string, sub: string, deviceLabel?: string): Promise<ApiResult<NewSession>>`, which is `NOT_FOUND` without a call when `tokenPath` refuses the token;
    - both send `device_label` only when given, as `login` does;
    - `reauth(password: string)` and `adminReauth(password: string)`, each `Promise<ApiResult<null>>`;
    - `linkIdentity(iss: string, sub: string)` and `adminLinkIdentity(iss: string, sub: string)`, each `Promise<ApiResult<null>>`;
    - `unlinkParentSso(parentId: string): Promise<ApiResult<null>>`, through `withId`.
  - Stub:
    - one `DEVICE_LABEL_RE = /^[A-Za-z0-9 ]{1,48}$/` shared by `POST /v1/session` and the two new session routes;
    - `state.authMode` (default `'local'`) and `state.identities` (`Map<string, { kind: 'admin' | 'parent'; id: string }>`, keyed by `iss + '\n' + sub`);
    - controls `POST /__auth-mode {mode}` and `POST /__identity {iss, sub, username}`;
    - helpers `setAuthMode(mode)` and `linkStubIdentity(iss, sub, username)`;
    - mode refusals on `POST /v1/session` (before its `device_label` check, `:5106`) and `…/redeem`;
    - every Task 3–6 route with the Go order (decode, mode, `device_label`, `iss`, `sub`, then the rest), its codes and fields, the label on the new parent session, and `sso_linked` in the views (the admin view of `GET /v1/session` gains `admin`); `redeem-oidc` ends the parent's other stub sessions, as Task 4 does;
    - `POST /v1/me/password` answers 429 `rate_limited` in the `rate_limited` scenario, as the Go handler now can;
    - `/__reset` restores `local` and clears the identities.

- [ ] **Step 1: Write the failing tests**

- `api.test.ts`: the paths, bodies and headers of each method (the client address is sent on `reauth` and `adminReauth` when given; `device_label` only when given); `toSession` for both kinds; `toAdminParent` with `sso_linked`. A response missing `sso_linked` gives reason `error`.
- `stub-sso.spec.ts`: for each new route, the answers of the spec 5.5 table in each mode, as `{ status, body }` pairs, mirroring `oidc_test.go` and `me_test.go`, including `TestOIDCValidationOrder`'s four cases. A password redemption refused in `oidc` and an SSO redemption refused in `local` leave the token usable. After `redeem-oidc`, a session the parent held before answers 401 `no_session`, as `TestRedeemOIDCEndsOtherSessions`, and `GET /v1/me/sessions` lists the label.

- [ ] **Step 2: Run them to verify they fail**

Run: `cd ui && pnpm test && pnpm test:e2e e2e/stub-sso.spec.ts`
Expected: FAIL.

- [ ] **Step 3: Implement**

- [ ] **Step 4: Run the UI pipeline**

Run: the Task 7 Step 4 command. Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add ui
git commit -m "feat(ui): API client and stub for the SSO routes"
```

## Task 9: The relying party and the conformance cases

**Files:**
- Modify: `ui/package.json`, `ui/pnpm-lock.yaml`
- Create: `ui/src/lib/server/oidc/relying-party.ts`, `ui/src/lib/server/oidc/relying-party.test.ts`, `ui/src/lib/server/oidc/log.ts`, `ui/e2e/stub-issuer.ts` (the issuer core; Task 10 adds its HTTPS listener and controls)

**Interfaces:**
- Consumes: `OidcEnv`, `OidcStart` (Task 7).
- Produces:
  - `type SsoCode = 'discovery' | 'idp_error' | 'state' | 'exchange' | 'id_token' | 'auth_time' | 'not_linked' | 'mode_refused' | 'already_linked' | 'reauth_required' | 'binding' | 'not_found' | 'oidc_not_configured'`.
  - `logSsoRefusal(code: SsoCode): void`: the fixed-shape line of Global Constraints.
  - `createRelyingParty(oidc: OidcEnv, origin: string, deps?: { fetch?: typeof fetch; jwksCache?: { jwks: { keys: object[] }; uat: number } }): RelyingParty`. `deps.jwksCache` exists only for the signing-key conformance case: after discovery it is handed to openid-client's exported `setJwksCache`.
  - `relyingParty(env: Env): RelyingParty | null`: one per process; null when `env.oidc` is null.
  - `type RelyingParty`:
    - `start(purpose: OidcPurpose): Promise<{ ok: true; url: string; state: string; nonce: string; verifier: string } | { ok: false }>`;
    - `finish(callback: URL, txn: OidcStart): Promise<{ ok: true; iss: string; sub: string; label: string | null } | { ok: false; code: SsoCode }>`.
- Behaviour pinned by the spec:
  - Discovery loads on the first `start` and is cached. A rejected discovery is not cached, so the next attempt retries. Nothing is fetched at start-up.
  - **Signing keys (owner, 2026-11-27; SSO spec § 3):** right after discovery the relying party calls openid-client's `enableNonRepudiationChecks(config)`. openid-client 6 verifies ID-token signatures only then (without it, oauth4webapi's `validateJwt` checks the header's `alg` and the claims, never the signature). The JWKS then loads at the first ID-token check through oauth4webapi's cache (`getPublicSigKeyFromIssuerJwksUri`), which refetches once the cached set is 300 s old and throws if that fetch fails: no stale-key grace window, inside the SSO spec's 3600 s ceiling. A bad signature or a failed fetch gives `id_token`.
  - The authorization URL carries `response_type=code`, `client_id`, `redirect_uri=${origin}/auth/callback`, `scope` (`oidc.scopes`), `state`, `nonce`, `code_challenge` (S256), `code_challenge_method=S256` and `response_mode=query`. It carries `prompt=login` and `max_age=1` for `link` and `invite` only (owner, 2026-11-27).
  - `finish` refuses with a code:
    - a query `error`: `idp_error`;
    - `state` ≠ `txn.state`: `state`;
    - a failed code exchange, including a replayed code: `exchange`;
    - an ID token check failure (the signature, `alg` other than RS256 through `id_token_signed_response_alg: 'RS256'`, `iss`, `aud`, `exp`, `nonce`): `id_token`;
    - for `link` and `invite` only, an ID token with no numeric `auth_time`, or with `auth_time < txn.startedAt - OIDC_AUTH_TIME_SKEW_S`: `auth_time`. The check is the relying party's own: it does not pass `maxAge` to openid-client's code grant, which compares `auth_time` with the callback's clock and its 30 s tolerance and would refuse every sign-in that took longer than that.
  - The label is `preferred_username` from the ID token, else from one userinfo call with the expected `sub`. A userinfo failure gives null. The label is cleaned as Review Focus 2 says, and an empty result gives null.
  - Client authentication at the token endpoint is `client_secret_basic` (Open question 3, resolved; the edge registers the client that way, SSO spec § 4.6 at `77d9d99`).
  - Calls are bounded by `OIDC_TIMEOUT_MS`.
- `e2e/stub-issuer.ts`:
  - `createIssuer(opts: { issuer: string; clientId: string; clientSecret: string })` returns `{ handle(req: Request): Promise<Response>; configure(b: Partial<IssuerBehaviour>): void; reset(): void; authorizeLog: Array<{ prompt: string | null; maxAge: string | null }> }`.
  - It serves discovery, `jwks.json`, authorization (a 302 to the redirect URI, or `error=access_denied`), the token endpoint (`client_secret_basic`, PKCE checked, single-use codes) and userinfo.
  - It signs RS256 with a `node:crypto` key pair; there is no new dependency.
  - `IssuerBehaviour = { user: { sub: string; preferredUsername?: string }; reauthUser?: { sub: string; preferredUsername?: string }; deny?: boolean; idToken?: Partial<{ iss: string; aud: string; expOffsetS: number; alg: string; nonce: string; authTime: number | null; badSignature: boolean }>; userinfoFails?: boolean; jwksFails?: boolean; down?: boolean }`.
  - The ID token carries `auth_time` = the authorize call's time unless `idToken.authTime` sets it (`null` omits the claim). `badSignature` signs with a second key pair under the published `kid`. `jwksFails` answers 503 on `jwks.json`; `down` answers 503 to everything but the controls (Task 10).
  - `reauthUser` answers when `prompt=login` was sent (the "another IdP session is live" case).

- [ ] **Step 1: Add the dependency**

Run: `cd ui && pnpm add -D --save-exact openid-client@6.8.9` (re-check at execution that 6.8.9 is still the newest release at least two weeks old; Open question 12)
Expected: `package.json` shows the exact version with no `^`, and `pnpm-lock.yaml` changes.

- [ ] **Step 2: Write the failing tests**

`relying-party.test.ts` uses the stub issuer through `deps.fetch = (input, init) => issuer.handle(new Request(input, init))` at `https://issuer.test`. It holds the SSO spec § 3 conformance cases and the spec 5.5 checks:

| Test | Assertion |
|---|---|
| `start builds the authorization URL` | every parameter above; `prompt=login` and `max_age=1` present for `link` and `invite`, both absent for `login` |
| `a full flow returns iss, sub and the label` | `iss === 'https://issuer.test'` |
| `bad state` | `{ ok: false, code: 'state' }`, and no token call is made |
| `wrong iss` / `wrong aud` / `expired token` / `wrong nonce` / `alg HS256 or none` | `code: 'id_token'` |
| `bad signature` (`badSignature`) | `code: 'id_token'` |
| `auth_time on a link` / `on an SSO invite` (`txn.startedAt = S`, `authTime` set) | `S - 30` passes; `S - 31` gives `code: 'auth_time'`; `null` (no claim) gives `code: 'auth_time'` |
| `login ignores auth_time` | `authTime: null` with purpose `login` passes |
| `key set older than 300 s, refetch fails` | `deps.jwksCache` holds the issuer's real key with `uat = now - 301` and `jwksFails` is set: `code: 'id_token'`. Control: the same with `uat = now - 10` passes, since the cached key is used. Each case builds its own `createRelyingParty`, so no earlier fetch is cached |
| `replayed code` | the second `finish` with the same callback gives `code: 'exchange'` |
| `IdP error` (`error=access_denied`) | `code: 'idp_error'` |
| `link while another IdP session is live` | with `reauthUser` set, the `link` flow returns `reauthUser.sub`; `authorizeLog` shows `prompt=login` |
| `label from userinfo` / `userinfo fails` / `no preferred_username` | the label, then null twice |
| `odd label` (Review Focus 2) | 300 characters gives 64; `<script>` is kept as text; U+202E and `\u0007` are removed; spaces only gives null |
| `discovery failure` | `start` gives `{ ok: false }`; the next `start` retries and succeeds |
| `nothing secret is logged` | with `console.*` spied over every case, no line contains the code, state, nonce, verifier, `sub`, label or client secret |

- [ ] **Step 3: Run them to verify they fail**

Run: `cd ui && pnpm test src/lib/server/oidc`
Expected: FAIL.

- [ ] **Step 4: Implement `relying-party.ts` and `log.ts` on openid-client's discovery, `enableNonRepudiationChecks`, authorization-URL, code-grant and userinfo functions**

Use its `customFetch` option for `deps.fetch`, and never its insecure-request option. Re-check at execution that the pinned version still verifies ID-token signatures only under `enableNonRepudiationChecks` and still refetches the JWKS at 300 s (`getPublicSigKeyFromIssuerJwksUri` in the resolved oauth4webapi's `build/index.js`, read for 6.8.9 with oauth4webapi 3.8.9).

- [ ] **Step 5: Run the UI pipeline**

Run: the Task 7 Step 4 command. Expected: all pass. No visible change, so no screenshots.

- [ ] **Step 6: Commit**

```bash
git add ui
git commit -m "feat(ui): OIDC relying party with the shared conformance cases"
```

## Task 10: SSO login: the e2e issuer per slot, hooks, the login page by mode, `/auth/sso` and `/auth/callback`

**Files:**
- Modify: `ui/src/hooks.server.ts`, `ui/src/routes/login/{+page.server.ts,+page.svelte}`, `ui/playwright.config.ts`, `ui/playwright.screenshots.config.ts`, `ui/.gitignore`, `ui/e2e/{stacks.ts,stub-api.ts,stub-issuer.ts,login.spec.ts}`, `docs/wiki/{Sign-in-and-invites.md,How-these-screenshots-are-made.md}`, `docs/screenshots/**` (regenerated)
- Create: `ui/e2e/tls.ts`, `ui/src/lib/server/oidc/{start.ts,start.test.ts}`, `ui/src/lib/components/SsoOutcome.svelte`, `ui/src/routes/auth/sso/{+page.server.ts,+page.svelte}`, `ui/src/routes/auth/callback/{+page.server.ts,+page.svelte}`, `ui/e2e/sso-login.spec.ts`, `ui/e2e/screenshots/sso.shot.ts`

**Interfaces:**
- Consumes: `relyingParty`, `seal`/`open`, `setOidcCookie`/`clearOidcCookie`, `api.authMode`, `api.oidcSession`, `deviceLabel`, `landing`, `safeNext`.
- Produces:
  - `startSso(event: RequestEvent, purpose: OidcPurpose, extra: { next?: string | null; binding?: string; username?: string; kind?: SessionKind }): Promise<{ ok: true; location: string } | { ok: false; message: string }>`. It seals an `OidcStart` with `startedAt = now` (unix seconds), `exp = now + OIDC_TXN_MAX_AGE_S` and, for `login` and `invite`, `deviceLabel = deviceLabel(event.request.headers.get('user-agent') ?? '')`; it sets the cookie and returns the IdP URL. A failure gives "Single sign-on is unavailable" and logs `discovery`.
  - `ssoOffered(mode: AuthMode, oidc: OidcEnv | null): boolean` in `oidc/start.ts`, used by the login, invite and account loads.
  - `SsoOutcome.svelte` props `{ heading: string; message?: string | null; back: { href: string; label: string } }` (Deviations item 4); `ssoBack(purpose: OidcPurpose | null, kind: SessionKind | null, binding: string | null): { href: string; label: string }` in `oidc/start.ts`: `/login` "Go to sign in" for `login` and an unreadable cookie, `/invite/{binding}` "Go to your invite link", `/account` "Go to My profile", `/admin/account` "Go to Account".
  - The callback load returns one of `{ outcome: 'continue'; to: string }`, `{ outcome: 'confirm'; label: string; username: string }` (Tasks 11, 12) or `{ outcome: 'failed'; back: { href: string; label: string } }`.
  - e2e: `slotPorts(slot)` gains `issuer: 4182 + 100 * slot`; `stub-api.ts` exports `ISSUER_URL` (`https://127.0.0.1:<issuer>`), `STUB_OIDC_CLIENT = { id: 'fake-client', secret: 'fake-client-secret-000000000000' }` and `STUB_SEAL_KEY` (32 fake characters); `e2e/tls.ts` exports `TLS_DIR` (`e2e/.tls`) and `ensureTls(): { cert: string; key: string }`; `stub-issuer.ts` exports `startIssuer(port)` (run when executed directly, as `stub-api.ts` does) and the worker helpers `configureIssuer(b)`, `resetIssuer()`, `issuerAuthorizeLog()` and `issuerDown()`, which call the slot's own issuer over `node:https` with `ca` read from `TLS_DIR`.
- `hooks.server.ts`:
  - `isPublic` adds `path.startsWith('/auth/')`.
  - The admin redirect (`:60`) skips `/auth/*`.
  - Responses for `/auth/*`, `/invite/*` and `/reset/*` get `referrer-policy: no-referrer`.
- Login page load returns `{ mode: AuthMode; sso: ssoOffered(mode, env.oidc) }`. If `authMode()` fails, it is `error(503, UNAVAILABLE_MESSAGE)`. `ssoOffered` logs `oidc_not_configured` once per process (Open question 14).
  - `local`, and `both` without settings: the password card only, as today (no markup change).
  - `oidc`: Deviations item 1 (item 2 without settings).
  - `both`: Deviations item 3.
  - The password action maps 403 `mode_refused` to "Password sign-in is turned off" before its other cases.
- `/auth/sso`:
  - The page always renders `SsoOutcome` with heading "Single sign-on is unavailable" and `ssoBack('login', …)`; a GET shows it too.
  - The default action re-reads `authMode()`. In `local` it fails (503) with that page. Otherwise it calls `startSso(event, 'login', { next: safeNext(form next) })` and `redirect(303, location)`.
- `/auth/callback` load (GET; answers **200** whatever happens). It never reads the session cookie or `locals.session` (Global Constraints, callback state):
  1. `open` the cookie. A null or `pending` cookie fails with `state`.
  2. `clearOidcCookie`.
  3. `finish`. A refusal, `auth_time` included, logs its code and renders `SsoOutcome` "Single sign-on did not complete" with `ssoBack` for the start's purpose (Open question 2, resolved).
  4. For `login`: `api.oidcSession(iss, sub, txn.deviceLabel ?? undefined)` (403 `not_linked` and `mode_refused` fail the same way, each logged), `setSessionCookie`, and render Deviations item 5 with `<meta http-equiv="refresh" content="0;url={to}">`, where `to = landing(kind, txn.next)`.
- e2e harness (per worker slot; nothing shared across workers):
  - `ensureTls()` creates `e2e/.tls/{cert,key}.pem` when missing, by `execFileSync('openssl', ['req', '-x509', '-newkey', 'ec', '-pkeyopt', 'ec_paramgen_curve:P-256', '-nodes', '-days', '7', '-subj', '/CN=127.0.0.1', '-addext', 'subjectAltName=IP:127.0.0.1', '-keyout', …, '-out', …])`. `stacks.ts`'s `globalSetup` calls it once, before the slots start.
  - For each slot `stacks.ts` starts `node e2e/stub-issuer.ts` (`TEST_PARALLEL_INDEX`) before the UI, and waits for `${ISSUER_URL}/.well-known/openid-configuration`: `answers(url, ca?)` polls with `node:https` `get` and `{ ca }` when a CA is given, `fetch` otherwise.
  - Each slot's UI env gains `OIDC_ISSUER` (that slot's issuer), `OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET`, `SESSION_SEAL_KEY` (the fake values above) and `NODE_EXTRA_CA_CERTS=<cert path>`.
  - `playwright.config.ts` `use.ignoreHTTPSErrors: true`, so the browser follows the authorize URL to the self-signed issuer.
  - The issuer's controls (`POST /__issuer/configure`, `/__issuer/reset`, `/__issuer/down`, `GET /__issuer/authorize-log`) live on its own HTTPS listener; each SSO spec calls `resetIssuer()` in `beforeEach`, after `resetStub()`.
  - `playwright.screenshots.config.ts` (one slot, `workers: 1`) calls `ensureTls()` at config load, adds the issuer as a third `webServer` (`url` its discovery document, `ignoreHTTPSErrors: true`), and gives `testEnv` the same OIDC variables and `NODE_EXTRA_CA_CERTS`; `pnpm test:visual` reuses that list. The stub's mode stays `local` unless a shot sets it, so the designed pages render as today.
  - `.gitignore` adds `/e2e/.tls`.

- [ ] **Step 1: Write the failing tests**

`sso-login.spec.ts`, each test calling `setAuthMode` and `linkStubIdentity` first:
- `oidc` shows only "Continue with single sign-on", and works without JavaScript.
- **The callback answers 200** (`response.status()` of the callback navigation is 200, with no `Location`). The session cookie arrives on the next navigation: after the meta refresh the page is the kid home (`expectLanding(page, { kid: MIA })`), with the 30-day hardened cookie.
- The session lists the device label the password login would have sent: after an SSO login with the browser's user agent, `/account`'s Signed-in devices shows the same label as after `signInWithForm`.
- `next=/kids/<id>/timeline` survives the round trip; `next=//evil.example` lands on `/`.
- The authorize log shows neither `prompt` nor `max_age` for a login.
- An admin linked by SSO lands on `/admin/parents`; then `/kids/<id>` gives a 303 to `/admin/parents` (Rule 2 at the UI).
- An unlinked `sub` and `deny` (`access_denied`, a person outside the group) each give "Single sign-on did not complete" and "Go to sign in", with no session cookie.
- **Replayed callback** (Review Focus 1): `page.goto` of the same callback URL again gives "did not complete" with 200, and the session is not replaced.
- A bad state (the cookie dropped before the callback) gives "did not complete".
- `both` shows the password card and the outline SSO button. With `issuerDown()`, the password login still works, and an SSO attempt ends without a session. The UI process may hold a cached discovery, so the e2e test does not pin which message; Task 9 pins "unavailable" for an uncached discovery failure.
- `local` shows no SSO button, and `POST /auth/sso` answers "Single sign-on is unavailable".
- The `hbt_oidc` cookie set by the start has `{ httpOnly: true, secure: true, sameSite: 'Lax', path: '/auth' }` and expires within 600 s.
- `referrer-policy: no-referrer` is on `/auth/callback`, `/invite/<token>` and `/reset/<token>`.
- Accessibility: in a `describe` per `colorScheme` (light, dark), `expectAccessible` on `/login` in `oidc` and in `both`, on `/auth/sso`, and on the callback's "did not complete" page; the `phone` and `desktop` projects give 390 and 1280 px. The "Signing you in" page refreshes at once, so its test asserts the 200 body (one h1, one "Continue" link to the landing) instead of running axe.

`login.spec.ts`: a password login in `oidc` (the form posted by hand without JavaScript, as in smoke) shows "Password sign-in is turned off".

`oidc/start.test.ts`: `ssoOffered(mode: AuthMode, oidc: OidcEnv | null): boolean` (the login, invite and account loads' one rule) is true only for `oidc` or `both` with settings; for `oidc` or `both` without settings it logs one `oidc_not_configured` line per process across calls. Deviations item 2 (`oidc` without settings) is not reachable in e2e, since every slot's UI has settings; its markup is the existing `Alert`, and its route is this unit test.

`e2e/screenshots/sso.shot.ts` (both sizes): `sso/01-login-oidc`, `sso/02-login-both`, `sso/03-did-not-complete`, `sso/04-sso-unavailable`.

- [ ] **Step 2: Run them to verify they fail**

Run: `cd ui && pnpm test src/lib/server/oidc && pnpm test:e2e e2e/sso-login.spec.ts`
Expected: FAIL.

- [ ] **Step 3: Implement**

- [ ] **Step 4: Run the UI pipeline, the screenshots and the visual baselines**

Run:
- `cd ui && pnpm install --frozen-lockfile && pnpm check && pnpm lint && pnpm test && pnpm test:e2e`
- `pnpm screenshots` then `node e2e/screenshots/index.ts --check`
- `pnpm test:visual`

Expected: all pass; the new `sso/*` shots exist in `docs/screenshots` and its README; `test:visual` passes unchanged (sign-in, set-password and error render as before in `local`).

- [ ] **Step 5: Wiki**

- `Sign-in-and-invites.md`: a "How do I sign in with single sign-on?" answer (the mode decides which buttons show, the round trip, "did not complete" and what to do), with the `sso/01`–`04` shots.
- `How-these-screenshots-are-made.md`: the stub issuer and its self-signed certificate (`openssl` is needed).

Run: `cd ui && node e2e/screenshots/index.ts --check`. Expected: every wiki link and image resolves.

- [ ] **Step 6: Commit**

```bash
git add ui docs/screenshots docs/wiki
git commit -m "feat(ui): SSO login with a 200 callback, hooks and mode-aware login page"
```

## Task 11: Invites and resets by mode, invite redemption by SSO, `/auth/confirm`

**Files:**
- Modify: `ui/src/lib/server/set-password.ts`, `ui/src/lib/components/SetPasswordForm.svelte`, `ui/src/routes/invite/[token]/{+page.server.ts,+page.svelte}`, `ui/src/routes/reset/[token]/+page.svelte`, `ui/src/routes/auth/callback/{+page.server.ts,+page.svelte}`, `ui/e2e/{invite.spec.ts,reset.spec.ts}`, `ui/e2e/screenshots/sso.shot.ts`, `scripts/smoke.sh` (`redeem`), `docs/wiki/Sign-in-and-invites.md`, `docs/screenshots/**`
- Create: `ui/src/routes/auth/confirm/{+page.server.ts,+page.svelte}`, `ui/e2e/sso-invite.spec.ts`

**Interfaces:**
- Consumes: `startSso`, `ssoBack`, `SsoOutcome`, `api.peekToken`, `api.redeemTokenOidc`, the callback outcomes (Task 10).
- Produces:
  - `setPasswordRoute(kind)`: `load` adds `{ mode, sso: ssoOffered(mode, env.oidc) }`. For `invite` it exports **named** actions `password` and `sso`, because SvelteKit refuses a default action beside named ones. `reset` keeps its default action.
  - `SetPasswordForm` takes `action?: string` and `mode: 'password' | 'sso' | 'both' | 'off'`; the invite page passes `'?/password'`. The paths (`POST /invite/{token}`, `POST /reset/{token}`) are unchanged for the edge's login zones.
  - The invite page by mode:
    - `local`: "Set a password" only (today's card);
    - `oidc`: Deviations item 6;
    - `both`: Deviations item 7.
  - The reset page in `oidc`: Deviations item 8.
  - Both pages map 403 `mode_refused` on submit to "Password sign-in is turned off".
  - `?/sso` re-reads the mode (it fails in `local`) and calls `startSso(event, 'invite', { binding: params.token })`.
  - The callback for `invite`, after `finish`:
    - `peekToken(binding)` must answer `kind === 'invite'`, else `SsoOutcome` with the DESIGN 8.4 sentence and "Go to sign in";
    - then it seals an `OidcPending` (`username` from the peek, the label, the start's `deviceLabel`, `exp` = the start's `exp`) and renders Deviations item 9.
  - `/auth/confirm`:
    - Its load returns `{}`, and the page shows `SsoOutcome` "Nothing to confirm" with "Go to sign in".
    - The default action reads `choice` (`confirm` or `cancel`), opens the cookie (it must be `pending`) and **always** clears it.
    - Cancel gives 303 to the start page (`/invite/{token}`).
    - For `invite`, Confirm calls `redeemTokenOidc(binding, iss, sub, deviceLabel ?? undefined)`. On success it sets the session cookie and gives 303 to `PARENT_HOME`. Refusals each log their code and render `SsoOutcome` "Single sign-on did not complete" with an `Alert`:
      - 404: the DESIGN 8.4 sentence with `INVITE_VALIDITY`, and "Go to sign in";
      - 409 `already_linked`: "That single sign-on account is already linked to an account";
      - 403 `mode_refused`: "Single sign-on is turned off";
      - a missing or invalid cookie: no `Alert`, "Go to sign in".

- [ ] **Step 1: Write the failing tests**

`sso-invite.spec.ts`:
- In `oidc` the invite page offers only SSO, with the read-only username. The flow sends `prompt=login` and `max_age=1` (`issuerAuthorizeLog`) and lands on the confirmation showing the stub user's `preferredUsername` and `bo`. Confirm lands on the welcome for `bo` with a 30-day cookie, the stub holds the identity, and `/account` lists the device label.
- In `local`, the invite page offers only "Set a password", and posting `?/sso` by hand fails.
- In `both`, both are offered; the password path still works.
- Cancel returns to the invite page, which is still valid.
- Review Focus 1: a double Confirm (two posts with the same cookie) gives one session; the second post renders "did not complete" (200).
- Review Focus 4: the token is replaced (`/__` reissue), used (redeemed by password in another context), or expired, or the parent disabled, between the start and Confirm. Each gives the DESIGN 8.4 sentence, and no session cookie.
- A reset token used for `?/sso` gives the DESIGN 8.4 sentence at the callback.
- Re-authentication (owner, 2026-11-27): the issuer configured with `authTime` 120 s before the start, or with `authTime: null`, gives "Single sign-on did not complete" with "Go to your invite link" at the callback, no confirmation and no session cookie; the invite page is still valid afterwards.
- Review Focus 2: `preferredUsername` set to 300 characters with `<b>` renders escaped and cut, and Confirm still works (the cookie survived).
- No `preferredUsername` gives the label "the single sign-on account you just signed in with".
- `already_linked` gives its message.
- The reset page in `oidc` shows the reset sentence and no form.
- Accessibility, light and dark: the invite page in `oidc` and `both`, the reset page in `oidc`, the confirmation, `GET /auth/confirm`, and a confirm refusal.

`invite.spec.ts` and `reset.spec.ts` keep passing: the invite form now posts `?/password`.

`sso.shot.ts` adds `sso/05-invite-oidc`, `sso/06-invite-both`, `sso/07-confirm-invite`, `sso/08-reset-oidc`, `sso/09-confirm-link-no-longer-valid`.

- [ ] **Step 2: Run them to verify they fail**

Run: `cd ui && pnpm test:e2e e2e/sso-invite.spec.ts`
Expected: FAIL.

- [ ] **Step 3: Implement**

- [ ] **Step 4: Smoke `redeem`**

`redeem` (`:193`) posts `"$path?/password"`.

- [ ] **Step 5: Run the UI pipeline, the screenshots, the visual baselines and the smoke**

Run: the Task 10 Step 4 commands, then the image builds and `scripts/smoke.sh hbt-api:dev hbt-ui:dev`.
Expected: all pass; `test:visual` unchanged for set-password in `local`; every smoke line `PASS:`.

- [ ] **Step 6: Wiki**

`Sign-in-and-invites.md`: the invite and reset answers say what each mode offers, with `sso/05`–`09`. Run `node e2e/screenshots/index.ts --check`. Expected: pass.

- [ ] **Step 7: Commit**

```bash
git add ui scripts/smoke.sh docs/screenshots docs/wiki
git commit -m "feat(ui): invite redemption by SSO with confirmation; invite and reset by mode"
```

## Task 12: Link SSO for parents and the admin; the admin nav, account page and Unlink

**Files:**
- Modify: `ui/src/lib/nav.ts` (+ `nav.test.ts`), `ui/src/lib/components/AppShell.svelte` (the side-entry classes move to `nav.ts`), `ui/src/routes/(app)/account/{+page.server.ts,+page.svelte}`, `ui/src/routes/admin/+layout.svelte`, `ui/src/routes/admin/parents/{+page.server.ts,+page.svelte}`, `ui/src/routes/auth/callback/+page.server.ts`, `ui/src/routes/auth/confirm/+page.server.ts`, `ui/e2e/{account.spec.ts,admin.spec.ts}`, `ui/e2e/screenshots/{sso.shot.ts,admin.shot.ts}`, `docs/wiki/{My-profile.md,Admin.md,Navigation-and-shell.md,Not-built-yet.md}`, `docs/screenshots/**`
- Create: `ui/src/lib/server/oidc/link.ts`, `ui/src/lib/components/SsoSection.svelte`, `ui/src/routes/(app)/account/link-sso/{+page.server.ts,+page.svelte}`, `ui/src/routes/admin/account/{+page.server.ts,+page.svelte}`, `ui/src/routes/admin/account/link-sso/{+page.server.ts,+page.svelte}`, `ui/e2e/sso-link.spec.ts`

**Interfaces:**
- Consumes: `startSso`, `ssoBack`, `SsoOutcome`, `api.reauth`/`adminReauth`, `api.linkIdentity`/`adminLinkIdentity`, `api.unlinkParentSso`, `Session.admin`, `Parent.ssoLinked`, `AdminParent.ssoLinked`.
- Produces:
  - `sessionBinding(token: string): string`: base64url SHA-256 of the session cookie value.
  - `linkSsoRoute(kind: SessionKind)`: the `load` (`{ mode, sso, linked }`) and default action shared by both `link-sso` routes. The action:
    1. Reads `password`.
    2. Calls `createApi(getEnv(), fetch, cookie, getClientAddress()).reauth` (`adminReauth` for the admin).
    3. Maps, before `failAction`: 403 `wrong_password` to the field message "The password is incorrect", 429 to "Too many attempts, try again in a minute", and 403 `mode_refused` to "Linking is turned off".
    4. Then `startSso(event, 'link', { binding: sessionBinding(cookie), username, kind })` and a 303 to the IdP (`username` from `locals.session`: `parent.username` or `admin.username`).
  - `SsoSection.svelte` props `{ action: string; state: 'linkable' | 'linked'; form: { field?: string; message?: string } | null }`: Deviations items 10 and 11; one component for `/account`, `/admin/account` and both `link-sso` pages (Rule 2).
  - The account page loads (`/account` and `/admin/account`) also return `{ mode, sso: ssoOffered(mode, env.oidc) }` (spec 5.5: the mode is read on each load of the account pages). The section renders `linkable` in `both` with settings and `!ssoLinked`, `linked` when `ssoLinked`, and nothing otherwise (parents cannot unlink, spec 5.5).
  - The `/account` `password` action maps 429 `rate_limited` to "Too many attempts, try again in a minute" (Task 5 added the 429).
  - `navModel(i: NavInput | AdminNavInput)` with `type AdminNavInput = { admin: true; pathname: string }`; for an `AdminNavInput` it returns `{ primary: [], more: [parents, account] }`, two `NavLink`s: `{ id: 'admin-parents', label: 'Parents', icon: ShieldCheckIcon, href: '/admin/parents', current: /^\/admin\/parents$/ }` and `{ id: 'admin-account', label: 'Account', icon: UserIcon, href: '/admin/account', current: /^\/admin\/account(\/|$)/ }`, note `'Open'`. `nav.ts` exports `SIDE_ENTRY_CLASS` (today's AppShell `SIDE` string), used by AppShell and the admin layout.
  - The admin layout renders those entries: Deviations item 13.
  - The admin account page `/admin/account`: Deviations item 14.
  - The callback for `link` seals an `OidcPending` with `binding`, `username` and `kind` (from the start cookie, never the session; Open question 1, resolved) and the label, and renders the confirmation.
  - `/auth/confirm` for `link`:
    - `timingSafeEqual(sessionBinding(current cookie), binding)`, else "did not complete" (logged as `binding`);
    - then `linkIdentity` or `adminLinkIdentity`, chosen by `locals.session.kind`;
    - on success a 303 to `/account` or `/admin/account`, where the section now reads "linked";
    - 403 `reauth_required` gives "Confirm your password again, then link"; 409 and `mode_refused` as in Task 11; the back link is `ssoBack('link', kind, …)`.
  - Admin parents page: Deviations items 15–17. The `unlink` action (`parent_id`) returns the result sentence; 404 gives `GONE`.

- [ ] **Step 1: Write the failing tests**

`nav.test.ts`: `navModel({ admin: true, pathname: '/admin/account' })` gives no primary entries, the ids `admin-parents`, `admin-account` in order, and `current` only on Account; the parent entries are unchanged.

`sso-link.spec.ts`, in `both`:
- A parent with a password links:
  - the authorize log shows `prompt=login` and `max_age=1`;
  - the confirmation names the stub user and `ana`;
  - Confirm returns to `/account` with "Single sign-on is linked.";
  - after signing out, SSO login lands on the kid home.
- **Link while another IdP session is live:** the issuer's current user is `david`, and `reauthUser` is `natalia` on `prompt=login`. The confirmation names `natalia`, and the identity linked is hers.
- **The callback without the session cookie** (callback state, Open question 1): the test captures the callback URL with `page.route` on `/auth/callback*` (aborting that first navigation), saves and removes `hbt_session` (`context.clearCookies({ name: 'hbt_session' })`), unroutes, and opens the captured URL. The confirmation names the stub user and `ana`. With the saved cookie restored (`context.addCookies`), Confirm links; without it, Confirm gives "did not complete" and nothing is linked.
- **Re-authentication** (owner, 2026-11-27): the issuer configured with `authTime` 120 s before the start, or with `authTime: null`, gives "Single sign-on did not complete" with "Go to My profile" at the callback, no confirmation, and nothing linked.
- A wrong password gives "The password is incorrect" under the field (`getByLabel('Password', { exact: true })`), and no redirect to the IdP.
- Review Focus 3:
  - between the start and Confirm, sign out and sign in as `cleo` in another page of the same context: Confirm gives "did not complete", and nothing is linked for either account;
  - the same with an admin session in place of `cleo`.
- Review Focus 1: a double Confirm links once.
- Re-auth expiry: `moveStubClock` past 5 minutes before Confirm gives "Confirm your password again, then link".
- In `local` and in `oidc`, the account page shows no Single sign-on section, and posting `/account/link-sso` by hand gives "Linking is turned off".
- The admin reaches `/admin/account` from the nav (the side nav at 1280 px, the header icon link at 390 px), links there, then SSO login lands on `/admin/parents`.
- The admin unlinks a parent: the confirm card shows the consequence list; after the destructive button the row loses the "Single sign-on" badge and the page shows the result sentence; that parent's open session is gone (the next navigation in their context goes to `/login`).
- The `/account` password change answers 429 with "Too many attempts, try again in a minute" in the `rate_limited` scenario.
- Accessibility, light and dark, at both widths: `/account` (linkable and linked), `/account/link-sso` with a refusal, `/admin/account`, `/admin/account/link-sso`, `/admin/parents` with a linked row and its confirm open, and the link confirmation; the forms work without JavaScript.

`admin.spec.ts`: the footnote reads the item 17 sentence. `account.spec.ts` keeps passing in `local` (no section).

`sso.shot.ts` adds `sso/10-account-linkable`, `sso/11-account-linked`, `sso/12-link-sso-wrong-password`, `sso/13-confirm-link`, `sso/14-admin-account`; `admin.shot.ts` re-takes the admin landing (the nav entries and the footnote change it) and adds `admin/NN-unlink-confirm` and `admin/NN-unlinked` with the next free numbers.

- [ ] **Step 2: Run them to verify they fail**

Run: `cd ui && pnpm test src/lib/nav && pnpm test:e2e e2e/sso-link.spec.ts`
Expected: FAIL.

- [ ] **Step 3: Implement**

- [ ] **Step 4: Run the UI pipeline, the screenshots and the visual baselines**

Run: the Task 10 Step 4 commands.
Expected: all pass. `test:visual` still passes: the admin baselines are not in `MATCHED_PAGES`, and the matched pages (sign-in, set-password, error) do not change in `local`.

- [ ] **Step 5: Wiki**

- `My-profile.md`: the Single sign-on section, with `sso/10`–`13`.
- `Admin.md`: the Account page and its nav entries, the "Single sign-on" badge and Unlink, the new footnote (`:78`), with `sso/14` and the new admin shots.
- `Navigation-and-shell.md` (`:118`): the admin now has Parents and Account.
- `Not-built-yet.md` (`:9`): an admin session now reaches `/admin/account` too.

Run: `node e2e/screenshots/index.ts --check`. Expected: pass.

- [ ] **Step 6: Commit**

```bash
git add ui docs/screenshots docs/wiki
git commit -m "feat(ui): Link SSO for parents and the admin; admin nav, account page and unlink"
```

## Task 13: Docs, HANDOFF and the final verification

**Files:**
- Modify: `README.md` (Configuration, Run locally, Test), `docs/superpowers/HANDOFF.md`

The spec is not edited: its "specified, not built" status is the owner's to change.

- [ ] **Step 1: README**

- **API table:** `AUTH_MODE`. The `ADMIN_*` rows read "used once, to create the admin when the database has none; then ignored with a warning". Replace the paragraph "Changing `ADMIN_USERNAME`, `ADMIN_PASSWORD` or `INTERNAL_API_TOKEN` ends every admin session…" with the CLI.
- **A new "Admin CLI" subsection:** the three verbs, the password on stdin, `DATA_DIR`, the exit codes, and the edge form `docker compose run --rm -T happybabytracker-api admin <verb>` (EDGE-03).
- **UI table:** the six `OIDC_*`/`SESSION_SEAL_KEY` rows, with the refusals of spec 13.2 (a present but empty variable counts as set).
- **A "Single sign-on" section:** the mode table, the redirect URI `${ORIGIN}/auth/callback`, `client_secret_basic`, the scopes, and the six password routes the edge must rate-limit (spec 5.5).
- **Test:** the e2e run and `pnpm screenshots` need `openssl` for the stub issuer's certificate.

- [ ] **Step 2: HANDOFF**

- **State table:** a row for plan 8.
- **"What to do next":**
  - this plan's image changes, and that the edge's `happybabytracker.md` runbook must move admin rotation to `admin reset-password` **before** the bump PR that brings this image is merged (SSO spec § 13);
  - the edge's `AUTH_MODE`, `OIDC_*` and `SESSION_SEAL_KEY` wiring is the edge plan's;
  - migration 0011 is forward-only: take an `edge-backup` archive before the deploy.

- [ ] **Step 3: Full verification**

Run:
- `cd api && gofmt -l . && go vet ./... && GOTOOLCHAIN=$(go env GOVERSION) go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./... && go test ./... -race`
- `cd ui && pnpm install --frozen-lockfile && pnpm check && pnpm lint && pnpm test && pnpm test:e2e && pnpm test:visual && node e2e/screenshots/index.ts --check`
- `docker build -t hbt-api:dev api && docker build -t hbt-ui:dev ui && scripts/smoke.sh hbt-api:dev hbt-ui:dev`

Expected: `gofmt -l` prints nothing, every suite passes, every smoke line `PASS:`, and the memory line stays within 64/140 MiB.

- [ ] **Step 4: Commit**

```bash
git add README.md docs/superpowers/HANDOFF.md
git commit -m "docs: single sign-on, admin CLI and AUTH_MODE"
```

## Done when

- Tasks 1–13 are committed, and the three pipelines of Task 13 Step 3 pass with their output in hand.
- `routeClasses` holds 133 keys, and the matrix runs in `both`.
- Every UI task's AGENTS.md "Done means" output is in hand: `pnpm check`, `pnpm lint`, `pnpm test`, `pnpm test:e2e` (axe clean in light and dark at 390 and 1280 px), `scripts/smoke.sh` where its markup changed, `pnpm screenshots`, `pnpm test:visual`, and the wiki pages of Tasks 10–12.
- The PR description has a **Deviations from the design** section that copies the list above, item by item, for the owner's sign-off.
- The PR states which image content changed:
  - `api/` sources: a migration, the CLI and the routes;
  - `ui/` sources;
  - UI dependencies: `openid-client`, in `package.json` and `pnpm-lock.yaml`;
  - no Dockerfile change.
  - It also states that the edge still needs its pins bumped and a deploy, after its runbook change. With `AUTH_MODE` unset the deploy behaves as `local`.

## Open questions for the owner

Each row has the plan's proposal. Rows 1–9 and 11–17 are resolved; 10 is open; 18 and 19 are new.

| # | Question | Plan's proposal | Where |
|---|---|---|---|
| 1 | **Resolved (owner, 2026-11-27; SSO spec § 3 "Callback state").** The link confirmation's `<username>`: a cross-site callback does not carry the `SameSite=Strict` session cookie. | Accepted: the link start writes the account's username into the sealed cookie, beside the session-binding hash, the purpose, `next` and the start time; the callback never reads the session; authorization happens only at `POST /auth/confirm`, which re-checks the live session against the bound hash (spec 5.5 "Callback state"). | Tasks 7, 10, 12 |
| 2 | **Resolved (owner, 2026-11-27):** a small "Single sign-on did not complete" page with a link back to where the flow started (spec 5.5 amended). **The callback's failure page.** The spec said the callback "answers the login page (or the invite or account page the flow started from)" with the message. | `SsoOutcome` (Deviations item 4) with `ssoBack`. | Tasks 10–12 |
| 3 | **Resolved (owner, 2026-11-27):** `client_secret_basic`. **Token-endpoint client authentication.** | `client_secret_basic`; the edge registers the client that way (SSO spec § 4.6 at `77d9d99`). | Task 9 |
| 4 | **Resolved (owner, 2026-11-27: "preferably yes").** Does `redeem-oidc` end the parent's other sessions? | Yes, in the same store transaction, as password redemption does (`RedeemToken` → `setPassword`, `store/parents.go:138`); tests `TestRedeemInviteOIDCEndsOtherSessions` and `TestRedeemOIDCEndsOtherSessions` (spec 5.5). | Tasks 4, 8 |
| 5 | **Resolved (owner, 2026-11-27):** as proposed. **Unlink on a parent with no SSO identity.** | 204, and still purges their sessions, push subscriptions and ntfy topic. The UI offers Unlink only on linked rows. | Task 6 |
| 6 | **Resolved (owner, 2026-11-27):** as proposed. **`iss`/`sub` validation at the API.** | 422 `iss` unless it is an absolute `https:` URL of at most 512 bytes; 422 `sub` unless 1–255 bytes; every OIDC lookup excludes `provider = 'local'`, since a pair `("local", <username>)` would otherwise match a password row (spec 5.5 route paragraph). | Task 4 |
| 7 | **Resolved (owner, 2026-11-27), changed from the proposal: align with webhooks-mcp, a variable that is present counts as set even when empty.** **Env edge cases.** | An empty `OIDC_ISSUER`/`OIDC_CLIENT_ID`/`OIDC_CLIENT_SECRET` counts as set, so the all-three-or-none check refuses a partial set; an empty `SESSION_SEAL_KEY` fails the ≥ 32 check; an empty `OIDC_SCOPES` is refused; `OIDC_AUTO_PROVISION` is refused when present (spec 13.2 amended). `OIDC_SCOPES` is not otherwise validated (a value without `openid` fails at the first SSO attempt). The cases the owner did not name are row 18. | Task 7 |
| 8 | **Resolved (owner, 2026-11-27):** as proposed. **OTEL.** Spec 5.5 and CLAUDE.md Rule 5 ask that tokens and the callback query never reach OTEL, but neither image exports OTEL today. | Nothing to scrub. The plan pins instead that no UI or API log line carries a URL, code, state or token (Tasks 4, 9, 10). | — |
| 9 | **Resolved (owner, 2026-11-27):** as proposed. **"Refusals log their code."** | The UI logs `{"msg":"sso refused","code":…}` with an enumerated code. The API's request line keeps its shape (route, status, principal kind), with no error code. | Tasks 9–12 |
| 10 | **The threat model (spec § 18).** It is a draft pending your review, and residuals 1–6 are for you to accept or reject. | The plan implements what § 18 states. Your acceptance gates P6 (going public), not this PR. | — |
| 11 | **Resolved (owner, 2026-11-27):** the drafted copy is used, each sentence listed under "Deviations from the design". **Copy the spec does not fix.** | The "New sentences" table. Composing the screens from existing parts (AGENTS.md Rule 5) added a few sentences the draft did not have ("Signing you in", "Continue", the back-link labels, the `oidc` invite heading and lead, "Parents", "Account", the unlink consequence list, and row 19's footnote); they are in the same table for the same sign-off. | Tasks 10–12 |
| 12 | **Resolved (owner, 2026-11-27):** as proposed. **`openid-client` version.** | **6.8.9** (2026-11-02), the newest release at least two weeks old on 2026-11-24; 6.9.0 (2026-11-13) and 6.9.1 (2026-11-20) were younger. Re-check at execution. | Task 9 |
| 13 | **Resolved (owner, 2026-11-27): the edge's; already fixed in homelab-edge `docs/runbooks/sso.md` ("Authelia down": "the admin presses Reset link, which for a parent without one issues an invite link (72 h)").** **Cross-repo drift.** The runbook said a parent who joined by SSO gets "a reset link, redeemable in `both`", while this app's `resetLink` (`api/internal/httpapi/admin.go:152`) issues an **invite** when the parent has no local password, redeemable by password in `local`/`both`. | Nothing to do here. | — |
| 14 | **Resolved (owner, 2026-11-27):** as proposed. **The `oidc_not_configured` log.** | Once per process. | Task 10 |
| 15 | **Resolved (owner, 2026-11-27):** as proposed. **Does a successful link clear `reauth_at`?** | No: a link needs a fresh re-auth window only, and each principal has at most one identity per issuer. | Task 5 |
| 16 | **Resolved (owner, 2026-11-27):** as proposed. **The smoke assertion 8 wording** "the CLI's `reset-password` refuses and clears the old admin cookie". | Read as: after `reset-password`, the UI refuses the old admin cookie and clears it (303 `/login`, `Max-Age=0`). | Task 2 |
| 17 | **Resolved (owner, 2026-11-27):** as proposed. **SSO in the smoke.** | The smoke gains no SSO step (it would need an issuer container). The real API's SSO routes are covered by Go tests, and the browser flow by Playwright against the stub API and the stub issuer. P6 on the edge is the first end-to-end check against Authelia. | — |
| 18 | **New: the env cases row 7 leaves open.** (a) `SESSION_SEAL_KEY`, or a non-empty `OIDC_SCOPES`, present while none of the three relying-party variables is: webhooks-mcp refuses a stray `OAUTH_*` in `static`; spec 13.2 checks the seal key only "while `OIDC_ISSUER` is set". (b) All three present and `OIDC_ISSUER` empty or not an `https:` URL: present counts as set, and the spec has no shape rule. | (a) Ignored, as the spec reads today. (b) Not refused at start; each SSO attempt then fails with "Single sign-on is unavailable" and the password login is unaffected. Say if either should be refused at start instead (one more `bad` entry each in Task 7). | Task 7 |
| 19 | **New: the admin parents footnote.** The prototype's "The admin’s own username and password can’t be changed here. They come from the deployment; changing them signs the admin out." (`Admin.dc.html`, `admin/parents/+page.svelte:293`, wiki `Admin.md:78`) is false once Task 1 lands: the env is read once, and rotation is the CLI's `reset-password`. AGENTS.md Rules 4 and 6 keep design sentences unless you decide. | "The admin’s own username can’t be changed. The operator resets the admin password with the admin command; that signs the admin out." (Deviations item 17). | Task 12 |
