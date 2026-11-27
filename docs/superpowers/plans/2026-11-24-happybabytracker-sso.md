# Single Sign-On Implementation Plan (plan 8)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status: plan only, written against `main` fd9cf11 (plans 1–7 merged) on 2026-11-24, updated 2026-11-27 for the owner's decisions of that day (Open questions 1 and 4 resolved; re-authentication, callback state and signing keys added to Tasks 4, 7 and 9–12). Execution waits for the owner's review and the answers still open under "Open questions for the owner".**

**Goal:** The admin moves from env vars into its own tables with a break-glass CLI, and the app becomes a provider-neutral OIDC relying party (login, invite redemption and explicit linking by SSO) behind `AUTH_MODE` (`local` default, `oidc`, `both`), so the edge can put the UI on `happybaby.public.<domain>` behind Authelia.

**Architecture:** The API stays the authority. Migration `0008_sso.sql` adds `admins` and `admin_identities`, rebuilds `sessions` with `admin_id` and `reauth_at`, and drops the admin fingerprint. A start-up import copies the env admin once, and `happybabytracker-api admin <verb>` edits it afterwards. The API reads `AUTH_MODE`, gates every password and SSO route on it, and resolves `(iss, sub)` pairs into sessions. It never talks to the issuer. The UI server is the relying party (`openid-client`, code flow with PKCE, `state` and `nonce`). It keeps the flow in an AEAD-sealed `hbt_oidc` cookie, answers the callback with a 200 page (never a redirect), and asks the API to resolve, link or redeem.

**Tech Stack:** Go 1.27, `modernc.org/sqlite`, argon2id (`golang.org/x/crypto`); SvelteKit 3 / Svelte 5 (Node adapter), Playwright, Vitest. **New:** `openid-client` (exact pin, Open question 12). No new Go module.

**Spec:** `docs/superpowers/specs/2026-11-20-happybabytracker-design.md` § 5.5 (the whole section) and the 2026-11-24 and 2026-11-27 amendments in the header, § 2, § 5.1, § 5.3, § 5.4, § 6.1, § 6.3, § 13, § 14, § 15 and § 18. Cross-repo source of truth: `davcs86/homelab-edge` `docs/superpowers/specs/2026-11-23-authelia-sso-respec.md` § 3 (app contract and conformance cases, with the 2026-11-27 bullets "Callback state" and "Signing keys" and the amended "Linking" and "Conformance", commit `1d12328`), § 4.3, § 4.6 (the `happybabytracker` client), § 5 (happybabytracker row), § 8 and § 10 (P6), and `docs/runbooks/sso.md`. The edge's wiring (compose, Caddy zones and redaction, SOPS keys, `extra_hosts`, Gatus, the edge runbook) is a separate plan in that repo.

## Plan number and file name

The existing plans number themselves in their title (plan 1 … plan 7, spec § 16), not in the file name, and the file name is `YYYY-MM-DD-<topic>.md`. This plan is **plan 8**: § 16 lists seven plans, and SSO is the next one. The file uses the path the request named (`2026-11-24-happybabytracker-sso.md`). The repo name in it is redundant beside the older names but harmless.

## Facts on main fd9cf11 this plan relies on

| Artifact | Where | Used by |
|---|---|---|
| Migrations `0001`–`0007`; the runner wants a 1..n sequence; each runs in one transaction on the single writer (`_txlock=immediate`, `foreign_keys(1)`) | `api/internal/store/migrate.go`, `store.go:76` | 1 |
| Table rebuild precedent (`nursing_sessions_new`, `DROP`, `RENAME`) | `migrations/0005_advanced_logging.sql:19-54` | 1 |
| `sessions`: `CHECK ((kind = 'admin') = (parent_id IS NULL))`; nothing references `sessions` | `migrations/0001_foundation.sql` | 1 |
| `identities (parent_id, provider, subject, secret_hash)`, PK `(provider, subject)`; local rows are `provider = 'local'`, `subject = username` (`providerLocal`) | `0001`; `store/parents.go:15` | 1, 4 |
| `wantTables`; `schema_version` `"7"` at `store_test.go:36, 54, 400, 549, 749, 856, 975`; `TestMigration0007KeepsRows` (`:890`, applies `ms[:6]`); `TestAdminSessionShape` (`:91`) | `api/internal/store/store_test.go` | 1 |
| `CreateSession(ctx, kind, parentID, tokenHash, now, expires)`, `SessionByToken`, `DeleteAdminSessions` | `store/sessions.go` | 1, 5 |
| `CreateParent` (one `INSERT`, `ErrConflict` on the unique username) | `store/parents.go:51` | 1 |
| `setPassword` (upserts the local identity, deletes sessions except one), `SetParentDisabled` (sessions, `deleteChannels`, last-guardian check) | `store/parents.go:113`, `:139`; `store/push.go:209` | 2, 4, 6 |
| `IssueToken` (replaces unused tokens of a kind), `PeekToken`, `RedeemToken` | `store/tokens.go` | 4 |
| `syncAdminFingerprint`, `checkAdminUsername`, `fingerprintKey`; `prepare(ctx, st, cfg)` | `api/cmd/happybabytracker-api/main.go:119, 181, 197` | 1 |
| `auth.AdminFingerprint`, `auth.CheckAdmin` (the only importer of `config` in `auth`) | `api/internal/auth/admin.go` | 1 |
| `config.Load` validates `ADMIN_USERNAME` (pattern on the raw value) and `ADMIN_PASSWORD` (≥ `MinAdminPasswordLen` = 16) | `api/internal/config/config.go:48-53` | 1, 3 |
| `login` (decode, `addr:` then `user:` limiter keys, one verify) and `checkCredentials` (the admin success skips the verify today) | `httpapi/session.go:68`, `:114` | 1, 3 |
| `createParent` refuses `username == d.Cfg.AdminUsername` with 422 | `httpapi/admin.go:65` | 1 |
| `redeemToken`: decode, `addr:` limiter, `PeekToken`, hash, `RedeemToken`, parent session | `httpapi/admin.go:181` | 3 |
| `changePassword`: no limiter today | `httpapi/me.go:86` | 5 |
| `Access` classes; `Routes(d)`; `routeClasses` (115 keys) at `matrix_test.go:59`; `matrixWant` (`:47`); `matrixBody` (decode-first 422); `newMatrixFixture` (`:291`) seeds the admin with `f.seed("admin", "", …)` (`:297`) | `httpapi/routes.go`, `matrix_test.go` | 1, 3–6 |
| Test kit: `newKitWith`, `asAdmin`/`seedSession(kind, parentID, ttl)`, `login`, `do`, `wantErr`, `wantCleanLog`, `testlog.Leaks` | `httpapi/testkit_test.go`; `internal/testlog` | all API tasks |
| UI: `hooks.server.ts` (`isPublic`, admin redirect at `:53`, `cache-control: no-store`); `src/env.ts` (`defineEnvVars`, all optional); `loadEnv` | `ui/src/hooks.server.ts`; `ui/src/env.ts`; `ui/src/lib/server/env.ts` | 7, 10 |
| UI client: `createApi(env, fetch, session?, clientAddress?)`, `call`, `refused` (`code`, `field`), `logFailure` (fixed shape), `toSession`, `toParent`, `toAdminParent`, `tokenPath` | `ui/src/lib/server/api.ts` | 8 |
| `setPasswordRoute(kind)` serves invite and reset with one **default** action; `SetPasswordForm.svelte` posts with no `action` | `ui/src/lib/server/set-password.ts`; `ui/src/lib/components/SetPasswordForm.svelte:63` | 11 |
| `landing(kind, next)`, `safeNext` | `ui/src/lib/server/landing.ts`, `next.ts` | 10 |
| Cookie attributes `{ path: '/', httpOnly, secure, sameSite: 'strict' }` | `ui/src/lib/server/cookie.ts` | 7, 10 |
| Stub API: `route()` handles `/v1/session` and `/v1/tokens/*` by hand (`:4188`, `:4228`); `createParent` refuses `ADMIN.username` with 422 (`:729`); controls `/__scenario`, `/__reset`, `/__clock`; `workers: 1` | `ui/e2e/stub-api.ts`; `ui/playwright.config.ts` | 1, 8, 10–12 |
| Smoke: `redeem` posts the invite path (`:193`); `refused` (`:1045`); assertion 8 (`:1428-1442`); log list of assertion 9 (`:1501`) | `scripts/smoke.sh` | 1–3, 11 |
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
  - **Order in every handler:** decode the body → mode check → everything else (token lookup, limiter, hashing, re-auth window). A refused redemption therefore leaves the token usable.
- **OIDC identities:** `provider` = the issuer URL (exact string), `subject` = `sub`, `secret_hash` NULL. Every OIDC lookup adds `provider <> 'local'`, so a pair can never match a password row. A pair belongs to one principal across both tables, and each principal has at most one OIDC identity per issuer.
- **Callback state (SSO spec § 3, owner 2026-11-27):** `/auth/callback` never reads the session cookie or `locals.session`; what it shows or binds to (purpose, `next`, the username, the session hash, the start time) comes from the sealed `hbt_oidc` only. For `link` and `invite` it authorizes nothing: `POST /auth/confirm` does, and for `link` it re-checks the live session against the bound hash.
- **Re-authentication and signing keys (SSO spec § 3, owner 2026-11-27):** `link` and `invite` send `prompt=login` and `max_age=0` and refuse an ID token whose `auth_time` is missing or earlier than the start time minus `OIDC_AUTH_TIME_SKEW_S`. The relying party calls `enableNonRepudiationChecks`, so ID-token signatures are verified on a JWKS at most 300 s old, failing closed.
- **Every route change touches `routes.go` and `routeClasses` in the same task** (How to Act 3). The access matrix runs with `AuthMode = both`, and its fixture seeds an `admins` row whose id the admin session carries. Per-mode refusals are tested apart from the matrix.
- **Rule 5:** no `sub`, `preferred_username`, label, `code`, `state`, invite token, session token, password or client secret in any log line or error string, API or UI. UI SSO refusals log one fixed-shape line, `{"level":"warn","msg":"sso refused","code":<enumerated>}`. The callback's URL is never logged.
- **Rule 6:**
  - `OIDC_CLIENT_SECRET` and `SESSION_SEAL_KEY` are read only under `ui/src/lib/server/`.
  - The link binding is compared with `crypto.timingSafeEqual`.
  - Fixtures carry only fake values, as `STUB_TOKEN` does.
- **Rule 7:** every limit and duration lives in `api/internal/constants/constants.go` or `ui/src/lib/server/constants.ts`, with client-safe mirrors in `ui/src/lib/constants.ts`.
- **Rule 8:** `reauth_at` is unix seconds UTC in an INTEGER column.
- **Rule 9:** comments are constraints only, at most 2 lines.
- **Rule 10:**
  - `openid-client` is pinned exactly (no `^`) in `ui/package.json`, with `pnpm-lock.yaml` updated in the same commit.
  - It goes in `devDependencies` like every other UI package, so adapter-node bundles it and `pnpm prune --prod` leaves the runtime image as it is.
  - No base-image or action pin changes.
- **The stub mirrors the Go handlers** (`ui/e2e/stub-api.ts`): the same validation order, codes and fields. UI tests stay green on a broken contract, so `scripts/smoke.sh` remains the real check for everything it covers.
- **Playwright** (plan 6 and 7 lessons):
  - Form saves wait with `submit()`; a test that edits a prefilled field first waits for `untilHydrated`.
  - CI runs the headless shell.
  - Run one spec by its path (`pnpm test:e2e e2e/sso-login.spec.ts`).
  - Every new page passes axe at 390×844 and 1280×800 and works without JavaScript.
  - Every new page keeps `Cache-Control: no-store`.
- **Copy** the spec fixes is used verbatim:
  - "Single sign-on is unavailable" and "Single sign-on did not complete".
  - "Continue with single sign-on" and "Set a password".
  - "You are linking <label> to <username>", with Confirm and Cancel.
  - "the single sign-on account you just signed in with" (the label fallback).
  - Every other string in this plan is a proposal (Open question 11).

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
   - Each gives 404 `not_found` and "This link is no longer valid", and the reissued token stays usable.
5. **CLI password on stdin** (Task 2):
   - Exactly one trailing `\n` or `\r\n` is stripped. Leading and trailing spaces stay part of the password, and so does an inner `\r`.
   - No newline at all works.
   - Empty stdin is refused, and so are 15 code points of `é` (30 bytes) and 257 code points.
   - Input longer than `CLIPasswordMaxBytes` is refused without hashing.

---

## Task 1: The admin becomes a principal in the database

Migration, store, the start-up import, the login and the username namespace land together: an admin session cannot be created without an `admins` row, so no smaller slice stays green.

**Files:**
- Create: `api/internal/store/migrations/0008_sso.sql`, `api/internal/store/admins.go`, `api/internal/store/admins_test.go`, `api/cmd/happybabytracker-api/admin.go`, `api/cmd/happybabytracker-api/admin_test.go`
- Modify: `api/internal/store/{sessions.go,parents.go,store.go,store_test.go,sessions_test.go,parents_test.go}`, `api/internal/model/model.go`, `api/internal/auth/password.go` (+ `password_test.go`), `api/internal/config/{config.go,config_test.go}`, `api/cmd/happybabytracker-api/{main.go,main_test.go}`, `api/internal/httpapi/{session.go,admin.go,session_test.go,admin_test.go,testkit_test.go,matrix_test.go}`, `ui/e2e/stub-api.ts` (`:729`), `scripts/smoke.sh` (assertion 8, first half)
- Delete: `api/internal/auth/admin.go`, `api/internal/auth/admin_test.go`

**Interfaces:**
- Produces:
  - `model.Admin{ID, Username string; HasPassword, SSOLinked bool; CreatedAt time.Time}`; `model.Principal` gains `AdminID string`.
  - `store.ErrAdminExists`.
  - `(*Store).HasAdmin(ctx) (bool, error)`.
  - `(*Store).CreateFirstAdmin(ctx, username, secretHash string, now time.Time) (model.Admin, error)`: `ErrAdminExists` when `admins` holds a row, `ErrConflict` when a parent holds the name.
  - `(*Store).AdminByUsername(ctx, username string) (model.Admin, string, error)` and `(*Store).AdminByID(ctx, id string) (model.Admin, string, error)`: the second value is the local hash, `""` when none; `ErrNotFound`.
  - `(*Store).CreateSession(ctx, kind, principalID string, tokenHash []byte, now, expires time.Time) (string, error)`: `principalID` is the admin id when `kind == "admin"`.
  - `auth.HashAdminPassword(pw string) (string, error)`: `ErrPasswordLength` below `MinAdminPasswordLen` or above `MaxPasswordLen`.
  - `importAdmin(ctx context.Context, st *store.Store, cfg config.Config, log *slog.Logger, now time.Time) error`; `prepare(ctx, st, cfg, log)`.
  - Test kit: `kit.adminID`; `newKitWith` seeds the admin `testAdminUser`/`testAdminPass` when `HasAdmin` is false.
- Removes: `DeleteAdminSessions`, `syncAdminFingerprint`, `checkAdminUsername`, `fingerprintKey`, `auth.AdminFingerprint`, `auth.CheckAdmin`, and `config.Load`'s two `ADMIN_*` checks. `Config` keeps the raw `AdminUsername`/`AdminPassword` strings.

- [ ] **Step 1: Write the failing store tests**

In `store_test.go`:
- Add `"admins", "admin_identities"` to `wantTables`, and change the seven `schema_version` assertions to `"8"`.
- `TestAdminSessionShape` inserts an `admins` row `a`. It then asserts:
  - an admin session with `admin_id = 'a'` and `parent_id` NULL is accepted;
  - an admin session with no `admin_id`, or with a `parent_id`, is refused;
  - a parent session that carries an `admin_id` is refused.
- `TestMigration0008KeepsRows` is modelled on `TestMigration0007KeepsRows`. It applies `ms[:7]` and seeds a parent session `s1`, an admin session (`kind='admin'`, `parent_id` NULL), a local identity and the meta row `admin_fingerprint`. After `Open`:
  - `s1` is kept, with `admin_id` and `reauth_at` NULL;
  - no admin session is left, and the fingerprint row is gone;
  - every other table keeps its count, and `pragma_foreign_key_check` finds nothing.

In `admins_test.go`:
- `TestCreateFirstAdminOnce`: the second call answers `ErrAdminExists`, and one row is stored.
- `TestCreateFirstAdminRefusesAParentsName`: `ErrConflict`, and `admins` stays empty.
- `TestCreateParentRefusesTheAdminsName`: `ErrConflict`.
- `TestOIDCIdentityIndexes`: a second non-local identity for the same parent and provider is a constraint error, and two `local` rows are impossible already (PK).

- [ ] **Step 2: Run them to verify they fail**

Run: `cd api && go test ./internal/store/ -run 'TestAdminSessionShape|TestMigration0008|TestCreateFirstAdmin|TestCreateParentRefusesTheAdminsName|TestOIDCIdentityIndexes' -race`
Expected: FAIL (no table `admins`, schema version 7).

- [ ] **Step 3: Write `0008_sso.sql`**

The constraints are the decision here, so the SQL is given:

```sql
CREATE TABLE admins (
    id         TEXT PRIMARY KEY,
    username   TEXT NOT NULL UNIQUE,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
) STRICT;

CREATE TABLE admin_identities (
    admin_id    TEXT NOT NULL REFERENCES admins (id) ON DELETE CASCADE,
    provider    TEXT NOT NULL,
    subject     TEXT NOT NULL,
    secret_hash TEXT,
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
    reauth_at    INTEGER,
    CHECK ((kind = 'admin' AND admin_id IS NOT NULL AND parent_id IS NULL)
        OR (kind = 'parent' AND parent_id IS NOT NULL AND admin_id IS NULL))
) STRICT;
INSERT INTO sessions_new (id, token_hash, kind, parent_id, created_at, last_seen_at, expires_at)
    SELECT id, token_hash, kind, parent_id, created_at, last_seen_at, expires_at FROM sessions WHERE kind = 'parent';
DROP TABLE sessions;
ALTER TABLE sessions_new RENAME TO sessions;

DELETE FROM meta WHERE key = 'admin_fingerprint';
```

- [ ] **Step 4: Implement the store side**

- `admins.go`:
  - `CreateFirstAdmin` runs one `inTx` (an immediate transaction on the writer). It checks `SELECT 1 FROM admins` (giving `ErrAdminExists`) and then `parents.username`. It inserts the `admins` row and an `admin_identities` row `('local', username, hash)`.
  - `AdminBy*` select the local hash and `SSOLinked` as `EXISTS (… provider <> 'local')`.
- `CreateParent` becomes an `inTx` that checks `admins.username` first (giving `ErrConflict`).
- `CreateSession` writes `admin_id` for admin sessions and `parent_id` for parent sessions. `SessionByToken` scans `admin_id` into `Principal.AdminID`.
- Delete `DeleteAdminSessions`.

- [ ] **Step 5: Run the store tests to verify they pass**

Run: `cd api && go test ./internal/store/ -race`
Expected: PASS.

- [ ] **Step 6: Write the failing start-up and login tests**

`admin_test.go` (package `main`) replaces `TestAdminFingerprintRotation`, `TestFirstStartRecordsTheFingerprint`, `TestAdminUsernameCollidingWithAParentIsRefused` and `TestConfigNeverHoldsAnUnnormalisedAdminUsername`:
- `TestImportCreatesTheAdminOnce`: a fresh dir and `baseEnv`. After `start`, `HasAdmin` is true and `AdminByUsername("root-admin")` verifies `admin-password-AAAAAAAA`.
- `TestImportIsIgnoredOnceAnAdminExists`: a second `start` with a different `ADMIN_PASSWORD`:
  - the old password still verifies;
  - stdout holds exactly one line with `"level":"WARN"` and `"msg":"admin credentials in the environment are ignored"`;
  - neither value appears in it.
  - A third start with both variables unset logs no warning.
- `TestImportRequiresTheVariablesWhileEmpty`: each case fails `start` naming the variable, never echoing the value:
  - `ADMIN_USERNAME` unset, `"David"`, `" david "` or `"qz"`;
  - `ADMIN_PASSWORD` unset, 15 runes, or 15 runes of `é`.
- `TestImportRefusesAParentsName`: replaces `TestRunRefusesToStartOnACollision`. The error is `collisionMsg`, the name is not echoed, and `admins` stays empty.

`config_test.go`:
- `TestLoadRefuses` loses the seven `ADMIN_*` cases.
- `TestLoadValid` still carries the raw strings.
- New `TestLoadIgnoresAdminVariables`: `ADMIN_USERNAME` unset, or `"David"`, loads.

`session_test.go`:
- `TestAdminLogin`, `TestAdminLoginWithMixedCaseName` and the limiter tests keep passing against the seeded row.
- The admin success now runs **1** verification (`:136`, "want 0" becomes "want 1"). A new case checks that a parent whose name is not an admin's runs exactly one too.

`admin_test.go`:
- `:88-93`: the admin's own username answers **409 `username_taken`** (spec 5.5, "as today" for parents).

`matrix_test.go`:
- `newMatrixFixture` creates the admin with `CreateFirstAdmin` and seeds its session with that id.

- [ ] **Step 7: Run them to verify they fail**

Run: `cd api && go test ./cmd/... ./internal/config/ ./internal/httpapi/ -race`
Expected: FAIL (`importAdmin` undefined; the admin still verified against the env).

- [ ] **Step 8: Implement**

- `importAdmin`:
  - If `HasAdmin`: log the one warning when either variable is non-empty, and return nil.
  - Else: the raw `cfg.AdminUsername` must match `constants.UsernamePattern`, and `auth.HashAdminPassword(cfg.AdminPassword)` must succeed. Each error names the variable.
  - Then `CreateFirstAdmin(…, now)`: `ErrConflict` becomes `errAdminCollision`; `ErrAdminExists` (the CLI won the race) returns nil.
  - `prepare` calls it in place of the two removed functions.
- `checkCredentials(r, username, valid, password) (kind, principalID string, authed bool, err error)`: `AdminByUsername`, then `ParentByUsername`. It runs one `verifyPassword` on every path, against `dummyHash()` when no hash was found. A disabled parent fails after the verify.
- `createParent` drops the `d.Cfg.AdminUsername` comparison.
- `testkit_test.go` seeds the admin and keeps `asAdmin()` returning a valid admin token.
- Stub: `createParent` answers 409 `username_taken` for `ADMIN.username`.

- [ ] **Step 9: Run the API pipeline**

Run: `cd api && gofmt -l . && go vet ./... && GOTOOLCHAIN=$(go env GOVERSION) go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./... && go test ./... -race`
Expected: `gofmt -l` prints nothing; PASS.

- [ ] **Step 10: Smoke assertion 8, first half**

Replace assertion 8's body. After `start_api "$ADMIN_PASSWORD_2"`:
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
  - `(*Store).SetAdminPassword(ctx, username, secretHash string) error`: upserts the local identity and deletes that admin's sessions; `ErrNotFound`.
  - `(*Store).UnlinkAdminSSO(ctx, username string) error`: deletes that admin's non-local identities and sessions; `ErrNotFound`.
  - `constants.CLIPasswordMaxBytes = 4*MaxPasswordLen + 2`.
- `main()`: when `os.Args[1] == "admin"`, it calls `os.Exit(runAdmin(…))` and never reaches `run`.
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
- `TestResetPassword`: seed two admin sessions, run with a new password, exit 0. Both sessions are gone, the new password verifies and the old one does not. A parent session survives.
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
| `CLIPasswordMaxBytes+1` bytes | refused, 1, KDF never called |

The KDF check uses the existing `idKey` test seam.
- `TestCLIOutputCarriesNoName`: neither the username nor the password appears in stdout or stderr for any verb.
- `TestCreateRacesTheImport`: 20 rounds. Each round opens two `*store.Store` on one fresh dir, runs `runAdmin(create)` and `importAdmin` behind a barrier, and asserts:
  - exactly one `admins` row;
  - if the import won, `create` exited 1 with `an admin already exists`;
  - if `create` won, the import returned nil and logged one warning.

- [ ] **Step 2: Run them to verify they fail**

Run: `cd api && go test ./cmd/... -run 'TestCreate|TestReset|TestUnlink|TestUsage|TestPasswordOnStdin|TestCLIOutput' -race`
Expected: FAIL (`runAdmin` undefined).

- [ ] **Step 3: Implement `runAdmin` and the two store functions**

The password is read through `io.LimitReader(stdin, CLIPasswordMaxBytes+1)`. More than `CLIPasswordMaxBytes` is refused. One trailing `\r\n` or `\n` is stripped, then `auth.HashAdminPassword` runs. Both store functions are one `inTx` each.

- [ ] **Step 4: Run the API pipeline**

Run: the Task 1 Step 9 command. Expected: PASS.

- [ ] **Step 5: Smoke assertion 8, second half**

After the first half:
- `printf '%s\n' "$ADMIN_PASSWORD_2" | docker run --rm -i --network none --read-only --cap-drop ALL --security-opt no-new-privileges -v "$VOLUME":/data "$API_IMAGE" admin reset-password "$ADMIN_USERNAME" >"$WORK/cli.log" 2>&1` exits 0, while the API keeps running.
- The old admin cookie then gets `303 /login` with `hbt_session=;…Max-Age=0`, the existing check.
- `NEW_ADMIN="$(login admin-login-2 … "$ADMIN_PASSWORD_2")"` succeeds.
- Add `cli.log` to assertion 9's file list.
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
  - `routeClasses` gains `"GET /v1/auth": Internal` (116 keys).

- [ ] **Step 1: Write the failing tests**

- `TestLoadAuthMode`: `""` gives `local`; `local`, `oidc` and `both` load. `Local`, `OIDC`, `none` and `both ` are refused, each error naming `AUTH_MODE`.
- `TestAuthModeRoute`: each mode is returned as-is; without the internal token it is 401.
- `TestPasswordLoginRefusedInOIDC`: in `oidc`, `POST /v1/session` with valid parent credentials gives 403 `mode_refused`. It runs **zero** password verifications, and the limiter is not counted: six refused calls from one address, then the mode switched to `both`, and the next login succeeds.
- `TestPasswordRedeemRefusedInOIDCLeavesTheTokenUsable`: in `oidc`, `POST /v1/tokens/{invite}/redeem` gives 403 `mode_refused`. In `both`, the same token then redeems (200). The same holds for a reset token.
- A body error still wins over the mode: `{"x":1}` in `oidc` gives 422 `unknown_field`.
- The matrix runs in `both`.

- [ ] **Step 2: Run them to verify they fail**

Run: `cd api && go test ./internal/config/ ./internal/httpapi/ -run 'AuthMode|RefusedInOIDC|TestAccessMatrix' -race`
Expected: FAIL.

- [ ] **Step 3: Implement**

`login` and `redeemToken` each call `allowMode(w, AuthLocal, AuthBoth)` right after `decodeJSON`.

- [ ] **Step 4: Run the API pipeline**

Run: the Task 1 Step 9 command. Expected: PASS.

- [ ] **Step 5: Smoke**

Add `refused AUTH_MODE bogus AUTH_MODE=bogus` beside the other `refused` lines (it fails in `config.Load`, before the store opens). Run the image builds and the smoke. Expected: every line `PASS:`.

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
- Consumes: `CreateSession` semantics, `deleteChannels`, the token columns.
- Produces:
  - `store.ErrNotLinked` and `store.ErrAlreadyLinked`.
  - `(*Store).OIDCLogin(ctx, iss, sub string, tokenHash []byte, now, parentExpires, adminExpires time.Time) (kind string, expires time.Time, err error)`, in one write transaction:
    - It looks the pair up in `identities` (`provider = iss AND provider <> 'local' AND subject = sub`), then in `admin_identities`.
    - It creates the session of the principal found.
    - For a **disabled** linked parent it deletes that parent's sessions, push subscriptions and ntfy topic, **commits**, and returns `ErrNotLinked`. An unknown pair also returns `ErrNotLinked`.
  - `(*Store).RedeemInviteOIDC(ctx, tokenHash []byte, iss, sub string, sessionHash []byte, now, expires time.Time) error`, in one transaction:
    1. Marks used an unused, unexpired `kind = 'invite'` token of an enabled parent; otherwise `ErrNotFound`. A reset token is `ErrNotFound`.
    2. `ErrAlreadyLinked` when the pair is in either table or the parent already has an identity for `iss`.
    3. Inserts the identity, deletes all that parent's existing sessions and inserts the new one (owner, 2026-11-27, Open question 4). Password redemption deletes them the same way: `RedeemToken` calls `setPassword(ctx, tx, parentID, secretHash, "")`, whose `DELETE FROM sessions WHERE parent_id = ? AND id <> ?` keeps none (`store/tokens.go`, `store/parents.go:133`).
    - Any error rolls back, so the token stays usable.
  - `pairTaken(ctx, tx *sql.Tx, iss, sub string) (bool, error)`, shared with Task 5.
  - `constants.OIDCIssuerMaxLen = 512` and `constants.OIDCSubjectMaxLen = 255` (Open question 6).
  - Routes:
    - `{POST, "/v1/session/oidc", Internal, d.oidcLogin}`: body `{iss, sub}`, answers `loginResponse`.
    - `{POST, "/v1/tokens/{token}/redeem-oidc", Internal, d.redeemTokenOIDC}`: same body, answers a parent `loginResponse`.
    - `routeClasses` 118 keys.
- Handler order: `decodeJSON` → `allowMode(AuthOIDC, AuthBoth)` → validation:
  - `iss` gives 422 field `iss` unless it is an absolute `https:` URL of at most 512 bytes;
  - `sub` gives 422 field `sub` unless it is 1–255 bytes.
  - Then the store. `ErrNotLinked` gives 403 `not_linked`, `ErrNotFound` gives 404 `not_found`, `ErrAlreadyLinked` gives 409 `already_linked`.
  - Session lifetimes: `ParentSessionTTL` and `AdminSessionTTL`.

- [ ] **Step 1: Write the failing tests**

Store tests (`identities_test.go`):
- `TestOIDCLoginParentAndAdmin`.
- `TestOIDCLoginNeverMatchesALocalRow`: `iss = "local"`, `sub = "ana"` gives `ErrNotLinked`.
- `TestOIDCLoginDisabledParentPurges`: seed sessions, a push device and an ntfy topic. The call gives `ErrNotLinked`, and all three are gone. A second call is idempotent.
- `TestRedeemInviteOIDC`.
- `TestRedeemInviteOIDCRefusals`: a reset token, a used token, an expired token, a disabled parent, and a token the admin replaced with `IssueToken` (Review Focus 4) each give `ErrNotFound`.
- `TestRedeemInviteOIDCAlreadyLinked`: the pair is held by another parent, or by the admin, or the parent already has this issuer. Each gives `ErrAlreadyLinked`, and the token stays redeemable afterwards.
- `TestRedeemInviteOIDCEndsOtherSessions`: seed two sessions for the invited parent and one for another parent. After the call the invited parent holds exactly one session, the one whose hash was passed, and the other parent's session is kept. A refused call (`ErrAlreadyLinked`) leaves both seeded sessions in place.
- `TestPairBelongsToOnePrincipal`: an index-level backstop. A pair inserted under a parent cannot be inserted under the admin through any store function.

Handler tests (`oidc_test.go`):
- `TestOIDCLoginModes`: `local` gives 403 `mode_refused`; `oidc` and `both` give 200 for a linked pair. The parent cookie lifetime is 30 days and the admin's 12 hours. An unknown pair and a disabled parent give the same 403 `not_linked` body, byte for byte.
- `TestRedeemOIDCModes`: in `local`, 403 `mode_refused`, and the token then redeems by password in `local`.
- `TestRedeemOIDCEndsOtherSessions`: a session token the parent held before the redemption then answers 401 `no_session` on `GET /v1/session`; the returned token answers 200.
- `TestOIDCValidation`: `iss` empty, `http://…`, 513 bytes or `local`; `sub` empty or 256 bytes. Each gives 422 with the field.
- `TestAdminAfterSSOLoginIsRefusedOnKidRoutes`: an admin session from `/v1/session/oidc` gets 403 on `GET /v1/kids` and on a kid route (Rule 2; the matrix covers the class).
- `TestOIDCLogCarriesNoSubject`: `wantCleanLog` with `sub`, `iss` and the token banned.

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
- Consumes: `pairTaken`, `ErrAlreadyLinked` (Task 4); `allowMode` (Task 3).
- Produces:
  - `model.Principal.ReauthAt time.Time` (zero when NULL); `model.Parent.SSOLinked bool`, computed in `parentSelect` as `EXISTS (… provider <> 'local')`.
  - `(*Store).StampReauth(ctx, sessionID string, now time.Time) error`.
  - `(*Store).LinkIdentity(ctx, p model.Principal, iss, sub string) error`: picks `identities` or `admin_identities` by `p.Kind`; `ErrAlreadyLinked`.
  - `constants.ReauthWindow = 5 * time.Minute`.
  - Routes, `routeClasses` 122 keys:
    - `{POST, "/v1/me/reauth", Parent, d.reauth}` and `{POST, "/v1/admin/me/reauth", Admin, d.reauth}`, body `{password}`, 204;
    - `{POST, "/v1/me/identities", Parent, d.linkIdentity}` and `{POST, "/v1/admin/me/identities", Admin, d.linkIdentity}`, body `{iss, sub}`, 204.
  - Session views:
    - `parentView` gains `sso_linked`, so `GET /v1/session` and `PATCH /v1/me` both carry it;
    - the admin view of `GET /v1/session` gains `"admin": {"username": …, "sso_linked": …}`.
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
- `TestSessionViewsCarrySSOLinked`: `sameKeys` on the parent view (`id`, `username`, `display_name`, `units`, `sso_linked`) and on the admin view (`kind`, `expires_at`, `admin`). The value flips after a link.
- The matrix: the four routes answer 422 to `matrixBody` before the mode check.

- [ ] **Step 2: Run them to verify they fail**

Run: `cd api && go test ./internal/httpapi/ ./internal/store/ -run 'Reauth|ChangePasswordCounts|Link|SessionViews|TestAccessMatrix' -race`
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
  - Route `{POST, "/v1/admin/parents/{parent}/unlink-sso", Admin, d.unlinkParentSSO}`, 204; 404 `not_found`. `routeClasses` 123 keys.
  - `adminParentView` gains `sso_linked`.

- [ ] **Step 1: Write the failing tests**

- `TestUnlinkParentSSO`: seed an OIDC identity, two sessions, a push device with a kid opt-in, and an ntfy topic with an opt-in. The answer is 204. The identity, sessions, device, topic and opt-ins are gone; the local identity stays and the password still logs in. `/v1/session/oidc` with the old pair then gives 403 `not_linked`.
- `TestUnlinkUnknownParent`: 404, byte-identical to `resetLink`'s 404.
- `TestAdminListCarriesSSOLinked`: the key set of a list row.
- The matrix: `{parent}` is filled as for the other admin parent routes.

- [ ] **Step 2: Run them to verify they fail**

Run: `cd api && go test ./internal/httpapi/ ./internal/store/ -run 'Unlink|AdminListCarries|TestAccessMatrix' -race`
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
  - `src/env.ts` declares the six variables. `OIDC_AUTO_PROVISION` uses a schema that keeps `undefined`, so presence is detectable.
  - `loadEnv` adds these to `bad` (variable names only):
    - one or two of `OIDC_ISSUER`, `OIDC_CLIENT_ID` and `OIDC_CLIENT_SECRET` non-empty: the missing names;
    - `OIDC_ISSUER` set and `SESSION_SEAL_KEY` shorter than `MIN_SESSION_SEAL_KEY` code points: `SESSION_SEAL_KEY`;
    - `OIDC_AUTO_PROVISION !== undefined`, including `''` and `'false'`.
    - `OIDC_SCOPES` defaults to `OIDC_DEFAULT_SCOPES`.
    - Without `OIDC_ISSUER`, `SESSION_SEAL_KEY` and `OIDC_SCOPES` are ignored (Open question 7).
  - `ui/src/lib/server/constants.ts`:
    - `OIDC_TXN_COOKIE = 'hbt_oidc'`, `OIDC_TXN_MAX_AGE_S = 600`, `OIDC_TXN_PATH = '/auth'`;
    - `MIN_SESSION_SEAL_KEY = 32`, `OIDC_DEFAULT_SCOPES = 'openid profile'`, `OIDC_CALLBACK_PATH = '/auth/callback'`;
    - `OIDC_TIMEOUT_MS = 5000` (a call), `OIDC_LABEL_MAX_LEN = 64` (a call), `OIDC_COOKIE_MAX_BYTES = 4096`;
    - `OIDC_AUTH_TIME_SKEW_S = 30` (owner, 2026-11-27: the `auth_time` skew of SSO spec § 3 "Linking").
  - `ui/src/lib/constants.ts`: `AUTH_MODES = ['local', 'oidc', 'both'] as const`.
  - `oidc/types.ts`:
    - `type OidcPurpose = 'login' | 'invite' | 'link'`;
    - `type OidcStart = { stage: 'start'; purpose: OidcPurpose; state: string; nonce: string; verifier: string; startedAt: number; next: string | null; binding: string | null; username: string | null; exp: number }`;
    - `type OidcPending = { stage: 'pending'; purpose: 'invite' | 'link'; binding: string; username: string; iss: string; sub: string; label: string | null; exp: number }`;
    - `type OidcCookie = OidcStart | OidcPending`.
    - `startedAt` is the flow's start time in unix seconds, for the `auth_time` check (Task 9).
    - `username` in `OidcStart` is filled for `link` with the account's username, read from the session at the start (owner, 2026-11-27, Open question 1), so the callback never needs the session cookie.
  - `oidc/seal.ts`:
    - `seal(sealKey: string, payload: OidcCookie): string`.
    - `open(sealKey: string, value: string | undefined, nowS: number): OidcCookie | null`.
    - The key is `hkdfSync('sha256', sealKey, '', 'hbt_oidc v1', 32)`. The cipher is AES-256-GCM with a random 12-byte IV and AAD `hbt_oidc`. The value is base64url of `iv ‖ ciphertext ‖ tag`.
    - `open` returns null on any decode, auth or JSON failure, on an unknown `stage`, and when `exp <= nowS`.
  - `cookie.ts`: `setOidcCookie(cookies, value, maxAgeS)` and `clearOidcCookie(cookies)` with `{ path: OIDC_TXN_PATH, httpOnly: true, secure: true, sameSite: 'lax' }`.

- [ ] **Step 1: Write the failing tests**

- `env.test.ts`: each refusal above names its variable and never its value. `OIDC_ISSUER` without a seal key is refused. No OIDC variable gives `oidc: null`. All five give the default scopes.
- `seal.test.ts`:
  - a round trip;
  - a flipped byte, a value sealed with another key, a truncated value, `exp` in the past, and `stage: 'other'` each give null;
  - two seals of the same payload differ (random IV);
  - a pending payload with a 64-character label and 400-character `iss`/`sub` stays under `OIDC_COOKIE_MAX_BYTES` with its attributes;
  - a `link` start payload with a 32-character username, a 43-character binding and `startedAt` round-trips, and stays under `OIDC_COOKIE_MAX_BYTES`;
  - a start payload without a numeric `startedAt` gives null.
- `cookie.test.ts`: the attributes.

- [ ] **Step 2: Run them to verify they fail**

Run: `cd ui && pnpm test`
Expected: FAIL.

- [ ] **Step 3: Implement**

- [ ] **Step 4: Run the UI pipeline**

Run: `cd ui && pnpm install --frozen-lockfile && pnpm check && pnpm lint && pnpm test && pnpm test:e2e`
Expected: all pass (the e2e suite is unchanged; the e2e env sets no OIDC variable yet).

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
    - `oidcSession(iss: string, sub: string): Promise<ApiResult<NewSession>>`;
    - `redeemTokenOidc(token: string, iss: string, sub: string): Promise<ApiResult<NewSession>>`, which is `NOT_FOUND` without a call when `tokenPath` refuses the token;
    - `reauth(password: string)` and `adminReauth(password: string)`, each `Promise<ApiResult<null>>`;
    - `linkIdentity(iss: string, sub: string)` and `adminLinkIdentity(iss: string, sub: string)`, each `Promise<ApiResult<null>>`;
    - `unlinkParentSso(parentId: string): Promise<ApiResult<null>>`, through `withId`.
  - Stub:
    - `state.authMode` (default `'local'`) and `state.identities` (`Map<string, { kind: 'admin' | 'parent'; id: string }>`, keyed by `iss + '\n' + sub`);
    - controls `POST /__auth-mode {mode}` and `POST /__identity {iss, sub, username}`;
    - helpers `setAuthMode(mode)` and `linkStubIdentity(iss, sub, username)`;
    - mode refusals on `POST /v1/session` and `…/redeem`;
    - every Task 3–6 route with the Go order (decode, mode, validation, then the rest), its codes and fields, and `sso_linked` in the views; `redeem-oidc` ends the parent's other stub sessions, as Task 4 does;
    - `/__reset` restores `local` and clears the identities.

- [ ] **Step 1: Write the failing tests**

- `api.test.ts`: the paths, bodies and headers of each method (the client address is sent on `reauth` and `adminReauth` when given); `toSession` for both kinds; `toAdminParent` with `sso_linked`. A response missing `sso_linked` gives reason `error`.
- `stub-sso.spec.ts`: for each new route, the answers of the spec 5.5 table in each mode, as `{ status, body }` pairs, mirroring `oidc_test.go` and `me_test.go`. A password redemption refused in `oidc` and an SSO redemption refused in `local` leave the token usable. After `redeem-oidc`, a session the parent held before answers 401 `no_session`, as `TestRedeemOIDCEndsOtherSessions`.

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
- Create: `ui/src/lib/server/oidc/relying-party.ts`, `ui/src/lib/server/oidc/relying-party.test.ts`, `ui/src/lib/server/oidc/log.ts`, `ui/e2e/stub-issuer.ts` (the issuer core; Task 10 adds its HTTPS listener)

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
  - The authorization URL carries `response_type=code`, `client_id`, `redirect_uri=${origin}/auth/callback`, `scope` (`oidc.scopes`), `state`, `nonce`, `code_challenge` (S256), `code_challenge_method=S256` and `response_mode=query`. It carries `prompt=login` and `max_age=0` for `link` and `invite` only (owner, 2026-11-27).
  - `finish` refuses with a code:
    - a query `error`: `idp_error`;
    - `state` ≠ `txn.state`: `state`;
    - a failed code exchange, including a replayed code: `exchange`;
    - an ID token check failure (the signature, `alg` other than RS256 through `id_token_signed_response_alg: 'RS256'`, `iss`, `aud`, `exp`, `nonce`): `id_token`;
    - for `link` and `invite` only, an ID token with no numeric `auth_time`, or with `auth_time < txn.startedAt - OIDC_AUTH_TIME_SKEW_S`: `auth_time`. The check is the relying party's own: it does not pass `maxAge` to openid-client's code grant, which compares `auth_time` with the callback's clock and its 30 s tolerance and would refuse every sign-in that took longer than that.
  - The label is `preferred_username` from the ID token, else from one userinfo call with the expected `sub`. A userinfo failure gives null. The label is cleaned as Review Focus 2 says, and an empty result gives null.
  - Client authentication at the token endpoint is `client_secret_basic` (Open question 3).
  - Calls are bounded by `OIDC_TIMEOUT_MS`.
- `e2e/stub-issuer.ts`:
  - `createIssuer(opts: { issuer: string; clientId: string; clientSecret: string })` returns `{ handle(req: Request): Promise<Response>; authorize(url: string): Promise<string>; configure(b: Partial<IssuerBehaviour>): void; authorizeLog: Array<{ prompt: string | null }> }`.
  - It serves discovery, `jwks.json`, authorization (a 302 to the redirect URI, or `error=access_denied`), the token endpoint (`client_secret_basic`, PKCE checked, single-use codes) and userinfo.
  - It signs RS256 with a `node:crypto` key pair; there is no new dependency.
  - `IssuerBehaviour = { user: { sub: string; preferredUsername?: string }; reauthUser?: { sub: string; preferredUsername?: string }; deny?: boolean; idToken?: Partial<{ iss: string; aud: string; expOffsetS: number; alg: string; nonce: string; authTime: number | null; badSignature: boolean }>; userinfoFails?: boolean; jwksFails?: boolean }`.
  - The ID token carries `auth_time` = the authorize call's time unless `idToken.authTime` sets it (`null` omits the claim). `badSignature` signs with a second key pair under the published `kid`. `jwksFails` answers 503 on `jwks.json`.
  - `authorizeLog` entries are `{ prompt: string | null; maxAge: string | null }`.
  - `reauthUser` answers when `prompt=login` was sent (the "another IdP session is live" case).

- [ ] **Step 1: Add the dependency**

Run: `cd ui && pnpm add -D --save-exact openid-client@<version from Open question 12>`
Expected: `package.json` shows the exact version with no `^`, and `pnpm-lock.yaml` changes.

- [ ] **Step 2: Write the failing tests**

`relying-party.test.ts` uses the stub issuer through `deps.fetch = (input, init) => issuer.handle(new Request(input, init))` at `https://issuer.test`. It holds the SSO spec § 3 conformance cases and the spec 5.5 checks:

| Test | Assertion |
|---|---|
| `start builds the authorization URL` | every parameter above; `prompt=login` and `max_age=0` present for `link` and `invite`, both absent for `login` |
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

Run: the Task 7 Step 4 command. Expected: all pass.

- [ ] **Step 6: Commit**

```bash
git add ui
git commit -m "feat(ui): OIDC relying party with the shared conformance cases"
```

## Task 10: SSO login: hooks, the login page by mode, `/auth/sso` and `/auth/callback`

**Files:**
- Modify: `ui/src/hooks.server.ts`, `ui/src/routes/login/{+page.server.ts,+page.svelte}`, `ui/playwright.config.ts`, `ui/.gitignore`, `ui/e2e/stub-issuer.ts`, `ui/e2e/login.spec.ts`
- Create: `ui/src/lib/server/oidc/start.ts`, `ui/src/routes/auth/sso/{+page.server.ts,+page.svelte}`, `ui/src/routes/auth/callback/{+page.server.ts,+page.svelte}`, `ui/e2e/sso-login.spec.ts`

**Interfaces:**
- Consumes: `relyingParty`, `seal`/`open`, `setOidcCookie`/`clearOidcCookie`, `api.authMode`, `api.oidcSession`, `landing`, `safeNext`.
- Produces:
  - `startSso(event: RequestEvent, purpose: OidcPurpose, extra: { next?: string | null; binding?: string; username?: string }): Promise<{ ok: true; location: string } | { ok: false; message: string }>`. It seals an `OidcStart` with `startedAt = now` (unix seconds) and `exp = now + OIDC_TXN_MAX_AGE_S`, sets the cookie and returns the IdP URL. A failure gives "Single sign-on is unavailable" and logs `discovery`.
  - The callback load returns one of `{ outcome: 'continue'; to: string }`, `{ outcome: 'confirm'; label: string; username: string }` (Tasks 11, 12) or `{ outcome: 'failed'; back: string }`.
- `hooks.server.ts`:
  - `isPublic` adds `path.startsWith('/auth/')`.
  - The admin redirect skips `/auth/*`.
  - Responses for `/auth/*`, `/invite/*` and `/reset/*` get `referrer-policy: no-referrer`.
- Login page load returns `{ mode: AuthMode; sso: boolean }`. If `authMode()` fails, it is `error(503, UNAVAILABLE_MESSAGE)`. In a mode that wants SSO without RP settings it logs `oidc_not_configured` once per process (a call).
  - `local`: the password form only, as today.
  - `oidc`: a form posting to `/auth/sso` with a hidden `next`, and no password form. Without RP settings: "Sign-in is unavailable" (copy proposal).
  - `both`: both forms.
  - The password action maps 403 `mode_refused` to "Password sign-in is turned off" (copy proposal).
- `/auth/sso`:
  - Its load returns `{}`, and the page shows a link back to `/login`.
  - The default action re-reads `authMode()`. In `local` it fails with the unavailable message. Otherwise it calls `startSso(event, 'login', { next: safeNext(form next) })` and `redirect(303, location)`.
  - On failure it renders "Single sign-on is unavailable" with a link to `/login`.
- `/auth/callback` load (GET; answers **200** whatever happens). It never reads the session cookie or `locals.session` (Global Constraints, callback state):
  1. `open` the cookie. A null or `pending` cookie fails with `state`.
  2. `clearOidcCookie`.
  3. `finish`. A refusal, `auth_time` included, logs its code and renders "Single sign-on did not complete" with a link back (`/login` for `login`; Open question 2).
  4. For `login`: `api.oidcSession(iss, sub)` (403 `not_linked` and `mode_refused` fail the same way, each logged), `setSessionCookie`, and render a page with `<meta http-equiv="refresh" content="0;url={to}">` and a Continue link, where `to = landing(kind, txn.next)`.
- e2e harness:
  - `playwright.config.ts` creates `e2e/.tls/{cert,key}.pem` once, when missing. It runs `openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes -days 7 -subj /CN=127.0.0.1 -addext subjectAltName=IP:127.0.0.1 -keyout … -out …` through `execFileSync`. The file is kept across config loads, because worker processes load the config again.
  - A third `webServer`, `node e2e/stub-issuer.ts`, listens on HTTPS port 4182 with `ignoreHTTPSErrors: true`.
  - The UI's env gains `OIDC_ISSUER=https://127.0.0.1:4182`, a fake client id and secret, a fake 32-character `SESSION_SEAL_KEY` and `NODE_EXTRA_CA_CERTS=<cert>`.
  - `use.ignoreHTTPSErrors: true`.
  - The issuer exposes `/__issuer` controls (`configure`, `reset`, `authorizeLog`, and `down`, which answers 503 to everything but the controls).
  - `.gitignore` adds `/e2e/.tls`.

- [ ] **Step 1: Write the failing tests**

`sso-login.spec.ts`, each test calling `setAuthMode` and `linkStubIdentity` first:
- `oidc` shows only "Continue with single sign-on" (copy proposal), passes axe, and works without JavaScript.
- **The callback answers 200** (`response.status()` of the callback navigation is 200, with no `Location`). The session cookie arrives on the next navigation: after the meta refresh the page is the kid home (`expectLanding(page, { kid: MIA })`), with the 30-day hardened cookie.
- `next=/kids/<id>/timeline` survives the round trip; `next=//evil.example` lands on `/`.
- The authorize log shows neither `prompt` nor `max_age` for a login.
- An admin linked by SSO lands on `/admin/parents`; then `/kids/<id>` gives a 303 to `/admin/parents` (Rule 2 at the UI).
- An unlinked `sub` and `deny` (`access_denied`, a person outside the group) each give "Single sign-on did not complete", with no session cookie.
- **Replayed callback** (Review Focus 1): `page.goto` of the same callback URL again gives "did not complete" with 200, and the session is not replaced.
- A bad state (the cookie dropped before the callback) gives "did not complete".
- `both` shows both forms. With the issuer stopped (`/__issuer` `down`), the password login still works, and an SSO attempt ends without a session. The UI process may hold a cached discovery, so the e2e test does not pin which message; Task 9 pins "unavailable" for an uncached discovery failure.
- `local` shows no SSO button, and `POST /auth/sso` fails.
- The `hbt_oidc` cookie set by the start has `{ httpOnly: true, secure: true, sameSite: 'Lax', path: '/auth' }` and expires within 600 s.
- `referrer-policy: no-referrer` is on `/auth/callback`, `/invite/<token>` and `/reset/<token>`.

`login.spec.ts`: a password login in `oidc` (the form posted by hand without JavaScript, as in smoke) shows "Password sign-in is turned off".

- [ ] **Step 2: Run them to verify they fail**

Run: `cd ui && pnpm test:e2e e2e/sso-login.spec.ts`
Expected: FAIL.

- [ ] **Step 3: Implement**

- [ ] **Step 4: Run the UI pipeline**

Run: the Task 7 Step 4 command. Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add ui
git commit -m "feat(ui): SSO login with a 200 callback, hooks and mode-aware login page"
```

## Task 11: Invites and resets by mode, invite redemption by SSO, `/auth/confirm`

**Files:**
- Modify: `ui/src/lib/server/set-password.ts`, `ui/src/lib/components/SetPasswordForm.svelte`, `ui/src/routes/invite/[token]/{+page.server.ts,+page.svelte}`, `ui/src/routes/reset/[token]/+page.svelte`, `ui/src/routes/auth/callback/{+page.server.ts,+page.svelte}`, `ui/e2e/{invite.spec.ts,reset.spec.ts}`, `scripts/smoke.sh` (`redeem`)
- Create: `ui/src/routes/auth/confirm/{+page.server.ts,+page.svelte}`, `ui/e2e/sso-invite.spec.ts`

**Interfaces:**
- Consumes: `startSso`, `api.peekToken`, `api.redeemTokenOidc`, the callback outcomes (Task 10).
- Produces:
  - `setPasswordRoute(kind)`: `load` adds `{ mode, sso }`. For `invite` it exports **named** actions `password` and `sso`, because SvelteKit refuses a default action beside named ones. `reset` keeps its default action.
  - `SetPasswordForm` takes `action?: string`; the invite page passes `'?/password'`. The paths (`POST /invite/{token}`, `POST /reset/{token}`) are unchanged for the edge's login zones.
  - The invite page by mode:
    - `local`: "Set a password" only;
    - `oidc`: "Continue with single sign-on" only (a form posting `?/sso`);
    - `both`: both.
  - The reset page in `oidc` shows no form: "This link works only while password sign-in is on. Ask the person who sent it." (copy proposal for the spec's "works only once the operator switches the app to `both`").
  - Both pages map 403 `mode_refused` on submit to "Password sign-in is turned off".
  - `?/sso` re-reads the mode (it fails in `local`) and calls `startSso(event, 'invite', { binding: params.token })`.
  - The callback for `invite`, after `finish`:
    - `peekToken(binding)` must answer `kind === 'invite'`, else "This link is no longer valid";
    - then it seals an `OidcPending` (`username` from the peek, the label, `exp` = the start's `exp`) and renders the confirmation: "You are linking <label or fallback> to <username>", with Confirm and Cancel posting to `/auth/confirm`.
  - `/auth/confirm`:
    - Its load returns `{}`, and the page shows "Nothing to confirm" with a link to `/login`.
    - The default action reads `choice` (`confirm` or `cancel`), opens the cookie (it must be `pending`) and **always** clears it.
    - Cancel gives 303 to the start page (`/invite/{token}`).
    - For `invite`, Confirm calls `redeemTokenOidc(binding, iss, sub)`. On success it sets the session cookie and gives 303 to `PARENT_HOME`. Refusals each log their code:
      - 404 gives "This link is no longer valid";
      - 409 `already_linked` gives "That single sign-on account is already linked to an account";
      - 403 `mode_refused` gives "Single sign-on is turned off";
      - a missing or invalid cookie gives "Single sign-on did not complete".

- [ ] **Step 1: Write the failing tests**

`sso-invite.spec.ts`:
- In `oidc` the invite page offers only SSO. The flow sends `prompt=login` and `max_age=0` (`authorizeLog`) and lands on the confirmation showing the stub user's `preferredUsername` and `bo`. Confirm lands on the welcome for `bo` with a 30-day cookie, and the stub holds the identity.
- In `local`, the invite page offers only "Set a password", and posting `?/sso` by hand fails.
- In `both`, both are offered; the password path still works.
- Cancel returns to the invite page, which is still valid.
- Review Focus 1: a double Confirm (two posts with the same cookie) gives one session; the second post renders "did not complete" (200).
- Review Focus 4: the token is replaced (`/__` reissue), used (redeemed by password in another context), or expired, or the parent disabled, between the start and Confirm. Each gives "This link is no longer valid", and no session cookie.
- A reset token used for `?/sso` gives "This link is no longer valid" at the callback.
- Re-authentication (owner, 2026-11-27): the issuer configured with `authTime` 120 s before the start, or with `authTime: null`, gives "Single sign-on did not complete" at the callback, no confirmation and no session cookie; the invite page is still valid afterwards.
- Review Focus 2: `preferredUsername` set to 300 characters with `<b>` renders escaped and cut, and Confirm still works (the cookie survived).
- No `preferredUsername` gives the label "the single sign-on account you just signed in with".
- `already_linked` gives its message.
- The reset page in `oidc` shows the proposed copy and no form.

`invite.spec.ts` and `reset.spec.ts` keep passing: the invite form now posts `?/password`.

- [ ] **Step 2: Run them to verify they fail**

Run: `cd ui && pnpm test:e2e e2e/sso-invite.spec.ts`
Expected: FAIL.

- [ ] **Step 3: Implement**

- [ ] **Step 4: Smoke `redeem`**

`redeem` posts `"$path?/password"`.

Run: the UI pipeline (Task 7 Step 4), then the image builds and `scripts/smoke.sh hbt-api:dev hbt-ui:dev`.
Expected: all pass; every smoke line `PASS:`.

- [ ] **Step 5: Commit**

```bash
git add ui scripts/smoke.sh
git commit -m "feat(ui): invite redemption by SSO with confirmation; invite and reset by mode"
```

## Task 12: Link SSO for parents and the admin; the admin account page and Unlink

**Files:**
- Modify: `ui/src/routes/(app)/account/{+page.server.ts,+page.svelte}`, `ui/src/routes/admin/+layout.svelte`, `ui/src/routes/admin/parents/{+page.server.ts,+page.svelte}`, `ui/src/routes/auth/callback/+page.server.ts`, `ui/src/routes/auth/confirm/+page.server.ts`, `ui/e2e/{account.spec.ts,admin.spec.ts}`
- Create: `ui/src/lib/server/oidc/link.ts`, `ui/src/routes/(app)/account/link-sso/{+page.server.ts,+page.svelte}`, `ui/src/routes/admin/account/{+page.server.ts,+page.svelte}`, `ui/src/routes/admin/account/link-sso/{+page.server.ts,+page.svelte}`, `ui/e2e/sso-link.spec.ts`

**Interfaces:**
- Consumes: `startSso`, `api.reauth`/`adminReauth`, `api.linkIdentity`/`adminLinkIdentity`, `api.unlinkParentSso`, `Session.admin`, `Parent.ssoLinked`.
- Produces:
  - `sessionBinding(token: string): string`: base64url SHA-256 of the session cookie value.
  - `linkSsoRoute(kind: SessionKind)`: the `load` and default action shared by both `link-sso` routes. The action:
    1. Reads `password`.
    2. Calls `createApi(getEnv(), fetch, cookie, getClientAddress()).reauth` (`adminReauth` for the admin).
    3. Maps 403 `wrong_password` to "The password is incorrect", 429 to "Too many attempts, try again in a minute", and 403 `mode_refused` to "Linking is turned off".
    4. Then `startSso(event, 'link', { binding: sessionBinding(cookie), username })` and a 303 to the IdP.
  - The account page loads (`/account` and `/admin/account`) also return `{ mode, sso }` from `authMode()` and `env.oidc` (spec 5.5: the mode is read on each load of the account pages).
  - The account page shows a "Single sign-on" section:
    - `both` and RP settings and `!ssoLinked`: a password field and "Link single sign-on", posting to `/account/link-sso` (the admin's to `/admin/account/link-sso`);
    - `ssoLinked`: "Single sign-on is linked." (parents cannot unlink, spec 5.5);
    - otherwise nothing.
  - The admin account page `/admin/account`: the username and the same section. The admin nav adds "Account".
  - The callback for `link` seals an `OidcPending` with `binding`, `username` (from the start cookie, never the session; Open question 1, resolved) and the label, and renders the confirmation.
  - `/auth/confirm` for `link`:
    - `timingSafeEqual(sessionBinding(current cookie), binding)`, else "did not complete" (logged as `binding`);
    - then `linkIdentity` or `adminLinkIdentity`, chosen by `locals.session.kind`;
    - on success a 303 to `/account` or `/admin/account`, where the section now reads "linked";
    - 403 `reauth_required` gives "Confirm your password again, then link" (copy proposal); 409 and `mode_refused` as in Task 11.
  - Admin parents page: an `unlink` action (`parent_id`) shown on rows with `ssoLinked` ("Unlink single sign-on"). It gives "Single sign-on unlinked. Their sessions and notification devices were removed." (copy proposal), and 404 gives `GONE`.

- [ ] **Step 1: Write the failing tests**

`sso-link.spec.ts`, in `both`:
- A parent with a password links:
  - the authorize log shows `prompt=login` and `max_age=0`;
  - the confirmation names the stub user and `ana`;
  - Confirm returns to `/account` with "linked";
  - after signing out, SSO login lands on the kid home.
- **Link while another IdP session is live:** the issuer's current user is `david`, and `reauthUser` is `natalia` on `prompt=login`. The confirmation names `natalia`, and the identity linked is hers.
- **The callback without the session cookie** (callback state, Open question 1): the test captures the callback URL with `page.route` on `/auth/callback*` (aborting that first navigation), saves and removes `hbt_session` (`context.clearCookies({ name: 'hbt_session' })`), unroutes, and opens the captured URL. The confirmation names the stub user and `ana`. With the saved cookie restored (`context.addCookies`), Confirm links; without it, Confirm gives "did not complete" and nothing is linked.
- **Re-authentication** (owner, 2026-11-27): the issuer configured with `authTime` 120 s before the start, or with `authTime: null`, gives "Single sign-on did not complete" at the callback, no confirmation, and nothing linked.
- A wrong password gives "The password is incorrect", and no redirect to the IdP.
- Review Focus 3:
  - between the start and Confirm, sign out and sign in as `cleo` in another page of the same context: Confirm gives "did not complete", and nothing is linked for either account;
  - the same with an admin session in place of `cleo`.
- Review Focus 1: a double Confirm links once.
- Re-auth expiry: `moveStubClock` past 5 minutes before Confirm gives the `reauth_required` message.
- In `local` and in `oidc`, the account page shows no Link section, and posting `/account/link-sso` by hand gives "Linking is turned off".
- The admin links from `/admin/account`, then SSO login lands on `/admin/parents`.
- The admin unlinks a parent: the row loses "linked", and that parent's open session is gone (the next navigation in their context goes to `/login`).
- axe on `/account`, `/admin/account` and both `link-sso` pages, at both widths; the forms work without JavaScript.

- [ ] **Step 2: Run them to verify they fail**

Run: `cd ui && pnpm test:e2e e2e/sso-link.spec.ts`
Expected: FAIL.

- [ ] **Step 3: Implement**

- [ ] **Step 4: Run the UI pipeline**

Run: the Task 7 Step 4 command. Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add ui
git commit -m "feat(ui): Link SSO for parents and the admin; admin account page and unlink"
```

## Task 13: Docs, HANDOFF and the final verification

**Files:**
- Modify: `README.md` (Configuration, Run locally, Test), `docs/superpowers/HANDOFF.md`

The spec is not edited: its "specified, not built" status is the owner's to change.

- [ ] **Step 1: README**

- **API table:** `AUTH_MODE`. The `ADMIN_*` rows read "used once, to create the admin when the database has none; then ignored with a warning". Replace the paragraph "Changing `ADMIN_USERNAME`, `ADMIN_PASSWORD` or `INTERNAL_API_TOKEN` ends every admin session…" with the CLI.
- **A new "Admin CLI" subsection:** the three verbs, the password on stdin, `DATA_DIR`, the exit codes, and the edge form `docker compose run --rm -T happybabytracker-api admin <verb>` (EDGE-03).
- **UI table:** the six `OIDC_*`/`SESSION_SEAL_KEY` rows, with the refusals of spec 13.2.
- **A "Single sign-on" section:** the mode table, the redirect URI `${ORIGIN}/auth/callback`, the scopes, and the six password routes the edge must rate-limit (spec 5.5).
- **Test:** the e2e run needs `openssl` for its stub issuer's certificate.

- [ ] **Step 2: HANDOFF**

- **State table:** a row for plan 8.
- **"What to do next":**
  - this plan's image changes, and that the edge's `happybabytracker.md` runbook must move admin rotation to `admin reset-password` **before** the bump PR that brings this image is merged (SSO spec § 13);
  - the edge's `AUTH_MODE`, `OIDC_*` and `SESSION_SEAL_KEY` wiring is the edge plan's;
  - migration 0008 is forward-only: take an `edge-backup` archive before the deploy;
  - the cross-repo drift of Open question 13.

- [ ] **Step 3: Full verification**

Run:
- `cd api && gofmt -l . && go vet ./... && GOTOOLCHAIN=$(go env GOVERSION) go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./... && go test ./... -race`
- `cd ui && pnpm install --frozen-lockfile && pnpm check && pnpm lint && pnpm test && pnpm test:e2e`
- `docker build -t hbt-api:dev api && docker build -t hbt-ui:dev ui && scripts/smoke.sh hbt-api:dev hbt-ui:dev`

Expected: `gofmt -l` prints nothing, every suite passes, every smoke line `PASS:`, and the memory line stays within 64/140 MiB.

- [ ] **Step 4: Commit**

```bash
git add README.md docs/superpowers/HANDOFF.md
git commit -m "docs: single sign-on, admin CLI and AUTH_MODE"
```

## Done when

- Tasks 1–13 are committed, and the three pipelines of Task 13 Step 3 pass with their output in hand.
- `routeClasses` holds 123 keys, and the matrix runs in `both`.
- The PR states which image content changed:
  - `api/` sources: a migration, the CLI and the routes;
  - `ui/` sources;
  - UI dependencies: `openid-client`, in `package.json` and `pnpm-lock.yaml`;
  - no Dockerfile change.
  - It also states that the edge still needs its bump PR merged and a deploy, after its runbook change. With `AUTH_MODE` unset the deploy behaves as `local`.

## Open questions for the owner

Each row has the plan's proposal. Rows 1 and 4 are resolved (owner, 2026-11-27). Rows 2, 3 and 5–7 change code if answered differently; the rest are confirmations.

| # | Question | Plan's proposal | Where |
|---|---|---|---|
| 1 | **Resolved (owner, 2026-11-27; SSO spec § 3 "Callback state").** The link confirmation's `<username>`: a cross-site callback does not carry the `SameSite=Strict` session cookie. | Accepted: the link start writes the account's username into the sealed cookie, beside the session-binding hash, the purpose, `next` and the start time; the callback never reads the session; authorization happens only at `POST /auth/confirm`, which re-checks the live session against the bound hash (spec 5.5 "Callback state"). | Tasks 7, 10, 12 |
| 2 | **The callback's failure page.** The spec says the callback "answers the login page (or the invite or account page the flow started from)" with the message. Re-rendering those pages at `/auth/callback` needs their data (mode, token peek, sessions). | A small page with "Single sign-on did not complete" and a link back to where the flow started. | Tasks 10–12 |
| 3 | **Token-endpoint client authentication.** Neither spec states it for the `happybabytracker` client, and the edge registration must match. | `client_secret_basic`, as the xstockstrat web client and Authelia's default use. | Task 9 |
| 4 | **Resolved (owner, 2026-11-27: "preferably yes").** Does `redeem-oidc` end the parent's other sessions? | Yes, in the same store transaction, as password redemption does (`RedeemToken` → `setPassword`, `store/parents.go:133`); tests `TestRedeemInviteOIDCEndsOtherSessions` and `TestRedeemOIDCEndsOtherSessions` (spec 5.5). | Tasks 4, 8 |
| 5 | **Unlink on a parent with no SSO identity.** | 204, and still purges their sessions, push subscriptions and ntfy topic. The UI offers Unlink only on linked rows. | Task 6 |
| 6 | **`iss`/`sub` validation at the API.** Not in the spec. | 422 `iss` unless it is an absolute `https:` URL of at most 512 bytes; 422 `sub` unless 1–255 bytes. Separately, every OIDC lookup excludes `provider = 'local'`; that is not optional, since a pair `("local", <username>)` would otherwise match a password row. | Task 4 |
| 7 | **Env edge cases.** | An empty string counts as unset for the three relying-party variables, `OIDC_SCOPES` and `SESSION_SEAL_KEY`. `OIDC_AUTO_PROVISION` is refused even when empty. `SESSION_SEAL_KEY` and `OIDC_SCOPES` are ignored without `OIDC_ISSUER`. `OIDC_SCOPES` is not validated (a value without `openid` fails at the first SSO attempt). | Task 7 |
| 8 | **OTEL.** Spec 5.5 and CLAUDE.md Rule 5 ask that tokens and the callback query never reach OTEL, but neither image exports OTEL today. | Nothing to scrub. The plan pins instead that no UI or API log line carries a URL, code, state or token (Tasks 4, 9, 10). | — |
| 9 | **"Refusals log their code."** | The UI logs `{"msg":"sso refused","code":…}` with an enumerated code. The API's request line keeps its shape (route, status, principal kind), with no error code. | Tasks 9–12 |
| 10 | **The threat model (spec § 18).** It is a draft pending your review, and residuals 1–6 are for you to accept or reject. | The plan implements what § 18 states. Your acceptance gates P6 (going public), not this PR. | — |
| 11 | **Copy the spec does not fix:** "Sign-in is unavailable"; "Password sign-in is turned off"; the reset page in `oidc`; "Linking is turned off"; "The password is incorrect"; "Confirm your password again, then link"; "That single sign-on account is already linked to an account"; "Single sign-on is linked."; the unlink confirmation; "Nothing to confirm". | As written in Tasks 10–12. | Tasks 10–12 |
| 12 | **`openid-client` version.** | **6.8.9** (2026-11-02), the newest release at least two weeks old on 2026-11-24; 6.9.0 (2026-11-13) and 6.9.1 (2026-11-20) are younger. Re-check at execution. | Task 9 |
| 13 | **Cross-repo drift (edge, not this plan).** `homelab-edge` `docs/runbooks/sso.md` § 8 ("Authelia down") says a parent who joined by SSO gets "a reset link, redeemable in `both`". This app's `resetLink` (`api/internal/httpapi/admin.go:113`) issues an **invite** when the parent has no local password (`HasPassword`, a `provider = 'local'` identity), and that invite is redeemable by password in `local`/`both` (spec 5.5). The admin presses the row's "Reset link" button and the page shows "Invite link for <username>", valid 72 hours, at `/invite/{token}`. | The edge plan's runbook says: "the admin presses Reset link, which for a parent with no password issues an invite link (72 h); with the app in `local` or `both` the parent opens it and sets a password (the SSO choice on that page fails while Authelia is down), then signs in with it". | — |
| 14 | **The `oidc_not_configured` log.** | Once per process. | Task 10 |
| 15 | **Does a successful link clear `reauth_at`?** | No: a link needs a fresh re-auth window only, and each principal has at most one identity per issuer. | Task 5 |
| 16 | **The smoke assertion 8 wording** "the CLI's `reset-password` refuses and clears the old admin cookie". | Read as: after `reset-password`, the UI refuses the old admin cookie and clears it (303 `/login`, `Max-Age=0`). | Task 2 |
| 17 | **SSO in the smoke.** The smoke gains no SSO step (it would need an issuer container). The real API's SSO routes are covered by Go tests, and the browser flow by Playwright against the stub API and the stub issuer. P6 on the edge is the first end-to-end check against Authelia. | Accept. | — |
