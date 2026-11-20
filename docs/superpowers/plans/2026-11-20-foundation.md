# Foundation Implementation Plan (plan 1 of 7)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Two images that let the seeded admin create parents by invite, parents log in, create kids and
share them per kid as guardian or caregiver — with CI, smoke and the access-matrix test in place.

**Architecture:** A Go API (`api/`) on one unpublished listener owns SQLite, credentials, sessions and
every access decision. A SvelteKit Node server (`ui/`) is the only thing a browser talks to; it keeps
the session token in a cookie and calls the API with a shared internal token. Every API route is one
row in a route table that names its access class; one middleware enforces it, and one test walks the
table.

**Tech Stack:** Go 1.27 (stdlib `net/http` mux, `modernc.org/sqlite`, `golang.org/x/crypto/argon2`,
`github.com/google/uuid` v7), distroless; SvelteKit 3 / Svelte 5, Node adapter, shadcn-svelte,
Tailwind 4, Vitest, Playwright + axe, pnpm, Node 24. Versions and config files start from
`davcs86/commsgun` at the same pins.

**Spec:** `docs/superpowers/specs/2026-11-20-happybabytracker-design.md` (sections 5, 6.1, 6.3, 13, 14).
Reference implementation for shapes and tooling: `davcs86/commsgun` (`api/`, `ui/`, `scripts/smoke.sh`,
`.github/workflows/ci.yml`).

## Global Constraints

- The admin is not a row in `parents` and can never hold a `kid_access` row. Admin sessions are refused
  by every route outside `/v1/admin/*` and `/v1/session`.
- A request for a kid the caller has no grant on answers **404**, byte-identical to an unknown kid.
- A kid always has at least one guardian whose account is not disabled.
- Logs are JSON lines with ids and enumerated facts only: never a name, username, note, value, token
  or secret. No secret, name or username in an error string either.
- Secrets are read only in server-only code and compared in constant time. No secret in a fixture.
- Every limit, cap and duration lives in `api/internal/constants/constants.go` (API) or
  `ui/src/lib/server/constants.ts` (UI). No literal elsewhere.
- Contract with `homelab-edge` (do not change without changing the spec): API container port **9210**,
  UI port **9212**, env var names of spec 13.1/13.2, health paths `/healthz`, `/healthz/api`, data
  path `/data`, image names `ghcr.io/davcs86/happybabytracker-api` and `-ui`, uid **10001**, read-only
  root filesystem.
- Values from the spec: username `^[a-z0-9._-]{3,32}$`; password ≥ 12 characters (admin password
  ≥ 16, `INTERNAL_API_TOKEN` ≥ 32); invite link 72 h, reset link 1 h; parent session 30 days sliding,
  admin session 12 h; login limiter 5 per minute per client address and per username; tokens are 32
  random bytes, stored as SHA-256.
- Comments state constraints only, at most 2 lines.
- Base images pinned by digest, workflow actions pinned by SHA (copy commsgun's pins).
- TDD: every behaviour lands test first. Commit after each task.

## Review Focus

1. **Two guardians act at once** (each revokes the other, or one leaves while the other is disabled by
   the admin): the kid must end with one active guardian, the loser gets a refusal. → Task 6.
2. **An invite or reset link used twice, after expiry, or after the parent was disabled**: refused with
   the same answer in every case; the first use is not undone. → Task 5.
3. **Usernames typed with capitals or spaces** (`" Ana"`, at login, at grant): treated as `ana`
   everywhere; never two accounts that differ by case. → Tasks 4, 6.
4. **A kid with a birth date in the future, an impossible date, or an unknown time zone**: 422 naming
   the field; nothing stored. → Task 6.
5. **The API is down or slow while the browser holds a valid cookie**: the UI shows an "unavailable"
   page and keeps the cookie; it does not log the user out or loop to `/login`. → Task 8.

Also pinned: a password longer than 256 characters is refused before hashing (Task 3).

---

## File Structure

```
.gitignore                       replaced (the Python template ignores lib/ and build/)
README.md  CLAUDE.md
.github/workflows/ci.yml
scripts/smoke.sh
api/
  Dockerfile  go.mod  go.sum
  cmd/happybabytracker-api/main.go        wiring only
  internal/constants/constants.go         every limit and duration
  internal/config/config.go               env → Config, refuses bad values
  internal/model/model.go                 Parent, Kid, Role, Units, Principal
  internal/store/                         SQLite; one file per area
    store.go  migrate.go  migrations/0001_foundation.sql
    parents.go  sessions.go  tokens.go  kids.go  access.go  meta.go
  internal/auth/                          no HTTP, no SQL
    password.go  token.go  limiter.go  admin.go
  internal/httpapi/                       routes and handlers
    routes.go      the route table (single source of truth)
    middleware.go  internal token, principal, access class, kid role
    respond.go     JSON and error helpers, request log line
    session.go  admin.go  me.go  kids.go  access.go  health.go
    matrix_test.go the access matrix
ui/
  (config files copied from commsgun/ui and renamed)
  src/hooks.server.ts                     origin check, cookie → locals
  src/lib/server/{env,constants,api,cookie}.ts
  src/lib/{types,format}.ts
  src/lib/components/                     AppShell (bottom bar / side nav), KidSwitcher, forms
  src/routes/
    healthz/  healthz/api/
    login/  logout/  invite/[token]/  reset/[token]/
    admin/parents/
    (app)/+layout  (app)/+page            kid list / home placeholder
    (app)/kids/new/  (app)/kids/[kid]/settings/  (app)/kids/[kid]/access/
    (app)/account/
  e2e/  stub-api.ts + specs
```

## API route table (the contract every task below shares)

Access classes: `Open` (no token), `Internal` (internal token only), `Session` (any session),
`Admin`, `Parent`, `Caregiver` (parent with any role on `{kid}`), `Guardian`.

| Method and path | Access | Task |
|---|---|---|
| `GET /healthz` | Open | 2 |
| `POST /v1/session` | Internal | 4 |
| `GET /v1/session`, `DELETE /v1/session` | Session | 4 |
| `GET /v1/tokens/{token}`, `POST /v1/tokens/{token}/redeem` | Internal | 5 |
| `GET /v1/admin/parents`, `POST /v1/admin/parents` | Admin | 5 |
| `POST /v1/admin/parents/{parent}/reset-link` | Admin | 5 |
| `POST /v1/admin/parents/{parent}/disable`, `.../enable` | Admin | 5 (disable rule: 6) |
| `PATCH /v1/me`, `POST /v1/me/password` | Parent | 5 |
| `GET /v1/me/sessions`, `DELETE /v1/me/sessions/{session}` | Parent | 5 |
| `GET /v1/kids`, `POST /v1/kids` | Parent | 6 |
| `GET /v1/kids/{kid}` | Caregiver | 6 |
| `PATCH /v1/kids/{kid}`, `DELETE /v1/kids/{kid}` | Guardian | 6 |
| `GET /v1/kids/{kid}/access` | Guardian | 6 |
| `PUT /v1/kids/{kid}/access/{username}` | Guardian | 6 |
| `DELETE /v1/kids/{kid}/access/{parent}` | Caregiver (self) or Guardian (anyone) | 6 |

Headers from the UI: `Authorization: Bearer <INTERNAL_API_TOKEN>` on every `/v1/*` call;
`X-Session-Token: <token>` when acting for a person; `X-Client-Address: <ip>` on `POST /v1/session`
and token redemption (the limiter's key). Errors are `{"error":"<snake_case_code>","field":"<name>"}`
(`field` only on 422). JSON bodies reject unknown fields with 422 `unknown_field`.

---

### Task 1: Repository skeleton, constants, config

**Files:**
- Create: `.gitignore`, `api/go.mod`, `api/internal/constants/constants.go`,
  `api/internal/config/config.go`, `api/internal/config/config_test.go`

**Interfaces:**
- Produces: `config.Load(getenv func(string) string) (config.Config, error)` with fields
  `AdminUsername, AdminPassword, InternalAPIToken, PublicOrigin, DataDir string`, `Addr string`
  (`":9210"`). `constants`: `MinPasswordLen=12`, `MaxPasswordLen=256`, `MinAdminPasswordLen=16`,
  `MinSecretLen=32`, `InviteTTL=72h`, `ResetTTL=1h`, `ParentSessionTTL=720h`,
  `SessionTouchInterval=1h`, `AdminSessionTTL=12h`, `LoginAttemptsPerMinute=5`, `TokenBytes=32`,
  `MaxBodyBytes=64<<10`, `SQLiteBusyTimeout=5s`, `ReadPoolMax=4`, `UsernamePattern`.

- [ ] **Step 1:** Replace `.gitignore` with: `node_modules/`, `ui/build/`, `ui/.svelte-kit/`,
  `ui/test-results/`, `ui/playwright-report/`, `*.db`, `*.db-wal`, `*.db-shm`, `.env*`.
- [ ] **Step 2:** `go mod init github.com/davcs86/happybabytracker/api` with the `go` and `toolchain`
  lines of `commsgun/api/go.mod`.
- [ ] **Step 3: Failing tests** in `config_test.go` (table-driven, `getenv` from a map):
  - `TestLoadValid`: all set → fields populated, `DataDir` defaults to `/data` when unset.
  - `TestLoadRefuses`: each of — `ADMIN_USERNAME` empty or not matching `UsernamePattern`;
    `ADMIN_PASSWORD` of 15 chars; `INTERNAL_API_TOKEN` of 31 chars; `PUBLIC_ORIGIN` not `http(s)://host`
    or with a path — returns an error whose text names the variable and never contains its value.
  - `VAPID_PRIVATE_KEY`, `VAPID_SUBJECT`, `HBT_ALLOW_FAKE_PUSH` and the `NTFY_*` variables are **not**
    read in this plan; plan 6 adds them (VAPID then becomes required at start, so the edge provisions
    it in plan 2).
- [ ] **Step 4:** Run `cd api && go test ./internal/config/` → FAIL (package missing).
- [ ] **Step 5:** Implement `constants.go` and `config.Load`.
- [ ] **Step 6:** Run again → PASS. `gofmt -l .` prints nothing; `go vet ./...` clean.
- [ ] **Step 7:** Commit `chore: api module, constants and config`.

### Task 2: Store, migrations, health

**Files:**
- Create: `api/internal/store/{store.go,migrate.go,meta.go,store_test.go}`,
  `api/internal/store/migrations/0001_foundation.sql`, `api/internal/model/model.go`,
  `api/internal/httpapi/{health.go,respond.go,health_test.go}`

**Interfaces:**
- Produces: `store.Open(dataDir string) (*Store, error)`, `store.OpenMemory() (*Store, error)`,
  `(*Store).Close() error`, `(*Store).Ping(ctx) error`, `(*Store).MetaGet(ctx, key) (string, bool, error)`,
  `(*Store).MetaSet(ctx, key, value) error`. Sentinel errors `store.ErrNotFound`, `store.ErrConflict`,
  `store.ErrLastGuardian`. `model.NewID() string` (UUIDv7).
- Open as commsgun does (`store.go`): one writer connection with `_txlock=immediate`, WAL,
  `foreign_keys(1)`, `secure_delete(ON)`, busy timeout; a read-only pool. Database file
  `happybabytracker.db`.
- Migrations: files `migrations/NNNN_name.sql` embedded with `embed.FS`, applied in order inside one
  transaction each, version recorded in `meta('schema_version')`. A database whose version is higher
  than the binary's newest migration refuses to open.

Schema `0001` (all ids `TEXT PRIMARY KEY`, times `INTEGER` unix seconds UTC, `STRICT` tables):

```sql
meta(key TEXT PRIMARY KEY, value TEXT NOT NULL)
parents(id, username UNIQUE, display_name, disabled INTEGER NOT NULL DEFAULT 0,
        unit_weight CHECK IN ('kg','lb'), unit_length CHECK IN ('cm','in'),
        unit_volume CHECK IN ('ml','floz'), unit_temp CHECK IN ('c','f'),
        created_at, updated_at)
identities(parent_id REFERENCES parents ON DELETE CASCADE, provider, subject, secret_hash,
           PRIMARY KEY(provider, subject))
sessions(id, token_hash UNIQUE, kind CHECK IN ('admin','parent'),
         parent_id REFERENCES parents ON DELETE CASCADE,   -- NULL iff kind='admin'
         created_at, last_seen_at, expires_at,
         CHECK ((kind='admin') = (parent_id IS NULL)))
one_time_tokens(token_hash PRIMARY KEY, kind CHECK IN ('invite','reset'),
                parent_id REFERENCES parents ON DELETE CASCADE, expires_at, used_at)
kids(id, name, birth_date TEXT, sex CHECK IN ('female','male'),
     gestation_weeks INTEGER, gestation_days INTEGER, time_zone,
     created_by REFERENCES parents, updated_by REFERENCES parents, created_at, updated_at)
kid_access(kid_id REFERENCES kids ON DELETE CASCADE, parent_id REFERENCES parents ON DELETE CASCADE,
           role CHECK IN ('guardian','caregiver'), created_at, PRIMARY KEY(kid_id, parent_id))
INDEX kid_access(parent_id)
```

`growth_source`, `vaccine_schema` and `kid_modules` arrive with the plans that use them.

- [ ] **Step 1: Failing tests** (`store_test.go`): `TestOpenAppliesMigrations` (fresh dir → schema
  version `1`, all seven tables exist); `TestOpenIsIdempotent` (open twice, version still `1`);
  `TestOpenRefusesNewerSchema` (set version `99` → error); `TestForeignKeysOn`
  (insert `kid_access` for a missing parent → error); `TestAdminSessionShape` (a `kind='admin'` row
  with a `parent_id` → constraint error).
- [ ] **Step 2:** Run `go test ./internal/store/` → FAIL.
- [ ] **Step 3:** Implement store, migration runner and `0001`.
- [ ] **Step 4:** Run → PASS.
- [ ] **Step 5: Failing test** `health_test.go`: `GET /healthz` → 200 `{"status":"ok"}` with an open
  store; 503 after `Close()`.
- [ ] **Step 6:** Implement `httpapi.Health(s *store.Store) http.HandlerFunc` and
  `respond.JSON(w, status, v)`, `respond.Error(w, status, code string, field ...string)`.
- [ ] **Step 7:** Run `go test ./...` → PASS. Commit `feat(api): store, migrations and health`.

### Task 3: Auth primitives

**Files:**
- Create: `api/internal/auth/{password.go,token.go,limiter.go,admin.go}` and `_test.go` for each

**Interfaces:**
- Produces:
  - `auth.HashPassword(pw string) (string, error)`; `auth.VerifyPassword(hash, pw string) bool` —
    argon2id, PHC string format, parameters from current OWASP guidance held in `constants`.
    `auth.ErrPasswordLength` when `len(pw)` is outside `[MinPasswordLen, MaxPasswordLen]`.
  - `auth.NewToken() (plain string, hash []byte, err error)` — 32 random bytes, base64url without
    padding; `auth.HashToken(plain string) []byte` (SHA-256).
  - `auth.NewLimiter(perMinute int, now func() time.Time) *Limiter`; `(*Limiter).Allow(key string) bool`
    (fixed one-minute window, in memory, bounded map).
  - `auth.NormalizeUsername(s string) (string, bool)` — trims, lower-cases, then matches
    `UsernamePattern`.
  - `auth.AdminFingerprint(internalToken, username, password string) string` —
    hex `HMAC-SHA256(key=internalToken, username + "\x00" + password)`.
  - `auth.CheckAdmin(cfg config.Config, username, password string) bool` — constant time on both.

- [ ] **Step 1: Failing tests:**
  - `TestHashVerifyRoundTrip`; `TestVerifyWrongPassword`; `TestHashesDiffer` (same password, two
    hashes differ); `TestPasswordLengthBounds` (11 and 257 chars → `ErrPasswordLength`, 12 and 256 ok).
  - `TestTokenIsUnique` (1000 tokens distinct, 43 chars); `TestHashTokenStable`.
  - `TestLimiterWindow`: 5 allowed, 6th refused, allowed again after the clock passes one minute;
    keys independent.
  - `TestNormalizeUsername`: `" Ana "`→`("ana",true)`, `"ab"`→false, `"a b"`→false, `"Ünï"`→false.
  - `TestAdminFingerprintChanges`: differs when either username or password differs.
  - `TestCheckAdmin`: right pair true; right username + wrong password false; wrong username false.
- [ ] **Step 2:** Run `go test ./internal/auth/` → FAIL.
- [ ] **Step 3:** Implement.
- [ ] **Step 4:** Run → PASS. Commit `feat(api): password, token, limiter and admin primitives`.

### Task 4: Route table, middleware, login

**Files:**
- Create: `api/internal/httpapi/{routes.go,middleware.go,session.go,session_test.go,testkit_test.go}`,
  `api/internal/store/{sessions.go,sessions_test.go,parents.go}`, `api/cmd/happybabytracker-api/main.go`
- Modify: `api/internal/model/model.go`

**Interfaces:**
- Produces:
  ```go
  // model
  type Role string  // "guardian" | "caregiver"
  type Principal struct { Kind string /* "admin"|"parent" */; ParentID, SessionID string }
  type Units struct { Weight, Length, Volume, Temp string }
  type Parent struct { ID, Username, DisplayName string; Disabled, HasPassword bool; Units Units; CreatedAt time.Time }
  type SessionInfo struct { ID string; CreatedAt, LastSeenAt time.Time; Current bool }
  type Kid struct { ID, Name, BirthDate, Sex, TimeZone string; GestationWeeks, GestationDays *int }
  type KidWithRole struct { Kid; Role Role }
  type Grant struct { ParentID, Username, DisplayName string; Role Role }
  // httpapi
  type Access int   // Open, Internal, Session, Admin, Parent, Caregiver, Guardian
  type Route struct { Method, Pattern string; Access Access; Handler http.HandlerFunc }
  func Routes(d Deps) []Route              // the only place routes are declared
  func NewHandler(d Deps) http.Handler     // builds the mux from Routes + middleware
  type Deps struct { Store *store.Store; Cfg config.Config; Limiter *auth.Limiter; Now func() time.Time; Log *slog.Logger }
  func PrincipalFrom(ctx) model.Principal
  func KidRoleFrom(ctx) model.Role         // set for Caregiver/Guardian routes
  // store
  func (s *Store) CreateSession(ctx, kind, parentID string, tokenHash []byte, now, expires time.Time) (id string, err error)
  func (s *Store) SessionByToken(ctx, tokenHash []byte, now time.Time) (model.Principal, error) // ErrNotFound if missing, expired, or parent disabled
  func (s *Store) TouchSession(ctx, id string, now, expires time.Time) error
  func (s *Store) DeleteSession(ctx, id string) error
  func (s *Store) DeleteAdminSessions(ctx) error
  func (s *Store) ParentByUsername(ctx, username string) (model.Parent, secretHash string, err error)
  ```
- Middleware order: body cap → internal token (constant time; 401 `unauthorized`) → principal from
  `X-Session-Token` (401 `no_session`) → access class (an admin on a non-admin class, or a parent on
  `Admin`: 403 `forbidden`) → (added in Task 6, with `RoleFor`) for `Caregiver`/`Guardian`, `Store.RoleFor(ctx, kid, parent)`; no grant
  → **404 `not_found`**; caregiver on a `Guardian` route → 403 `forbidden`. One log line per request:
  `route` (the pattern, not the path), `status`, `principal_kind`, `request_id`, `duration_ms`.
- `POST /v1/session` body `{"username","password"}` → 200
  `{"token","kind","expires_at"}`. Normalise the username. If it equals the admin username, check the
  admin pair only (a parent can never be created with that name — Task 5). Otherwise verify the parent's
  `local` identity. Every failure (unknown user, wrong password, disabled) → 401 `invalid_credentials`
  after a password hash has been verified against a real or dummy hash. Limiter keys
  `addr:<X-Client-Address>` and `user:<normalised username>`; over the limit → 429 `rate_limited`
  with `Retry-After: 60`.
- A parent session used with less than `ParentSessionTTL - SessionTouchInterval` remaining is extended
  to `now + ParentSessionTTL`. Admin sessions never slide.
- `main.go`: load config, open store, compare `auth.AdminFingerprint` with `meta('admin_fingerprint')`;
  when different call `DeleteAdminSessions` and store the new one; serve; graceful shutdown on SIGTERM.

- [ ] **Step 1: Failing tests** (`session_test.go`, using a `testkit` that builds `NewHandler` over
  `OpenMemory` with a fixed clock and helpers `asAdmin`, `asParent(username)`):
  - `TestInternalTokenRequired`: no or wrong bearer on `POST /v1/session` → 401 `unauthorized`.
  - `TestAdminLogin`: right pair → 200, `kind=admin`, expiry `now+12h`.
  - `TestLoginFailuresLookAlike`: unknown user, wrong password, disabled parent → identical status
    and body.
  - `TestLoginNormalisesUsername`: `" Ana "` logs in as `ana`.
  - `TestLoginRateLimit`: 6th attempt in a minute from one address → 429 with `Retry-After`.
  - `TestSessionSlides`: parent session at day 29 used → expiry moves to day 59; admin session at
    hour 11 used → expiry unchanged; expired session → 401 `no_session`.
  - `TestLogout`: `DELETE /v1/session` then `GET /v1/session` → 401.
  - `TestAdminFingerprintRotation` (in `cmd` test): start with password A and an admin session, restart
    with password B → the session is gone; restart again with B → sessions kept.
  - `TestLogHasNoIdentifyingText`: capture the logger over a login and a failure; no username, token
    or password substring.
- [ ] **Step 2:** Run `go test ./...` → FAIL.
- [ ] **Step 3:** Implement store functions, route table, middleware, session handlers, `main.go`.
- [ ] **Step 4:** Run `go test ./... -race` → PASS. Commit `feat(api): route table, access middleware and login`.

### Task 5: Parents, invites, resets, own account

**Files:**
- Create: `api/internal/httpapi/{admin.go,me.go,admin_test.go,me_test.go}`,
  `api/internal/store/{tokens.go,tokens_test.go}`
- Modify: `api/internal/store/parents.go`, `api/internal/httpapi/routes.go`

**Interfaces:**
- Consumes: Task 3 (`NewToken`, `HashPassword`), Task 4 (`Routes`, `PrincipalFrom`).
- Produces:
  ```go
  func (s *Store) CreateParent(ctx, username, displayName string, now time.Time) (model.Parent, error) // ErrConflict on duplicate
  func (s *Store) ListParents(ctx) ([]model.Parent, error)        // id, username, display name, disabled, has_password; nothing about kids
  func (s *Store) SetParentDisabled(ctx, id string, disabled bool, now time.Time) error // deletes sessions on disable; ErrLastGuardian wired in Task 6
  func (s *Store) IssueToken(ctx, kind, parentID string, hash []byte, expires time.Time) error // replaces unused tokens of that kind for that parent
  func (s *Store) PeekToken(ctx, hash []byte, now time.Time) (kind, username string, err error) // ErrNotFound if unknown, used, expired, or parent disabled
  func (s *Store) RedeemToken(ctx, hash []byte, secretHash string, now time.Time) (parentID string, err error) // one transaction: mark used, upsert local identity, delete that parent's sessions
  func (s *Store) UpdateParentProfile(ctx, id, displayName string, u model.Units, now time.Time) error
  func (s *Store) SetPassword(ctx, parentID, secretHash string, keepSessionID string) error // deletes the parent's other sessions
  func (s *Store) ListSessions(ctx, parentID string) ([]model.SessionInfo, error)
  ```
- `POST /v1/admin/parents` `{"username","display_name"}` → 201 `{"parent":{…},"invite_url"}`.
  `invite_url` = `PUBLIC_ORIGIN + "/invite/" + token`. Username equal to the admin's → 422 field
  `username`. Display name 1–64 characters after trimming.
- `POST …/reset-link` → `{"reset_url"}` (`/reset/<token>`). For a parent who never set a password it
  issues an invite instead.
- `POST /v1/tokens/{token}/redeem` `{"password"}` → a session, as `POST /v1/session` returns.
  `GET /v1/tokens/{token}` → `{"kind","username"}` or 404.
- `PATCH /v1/me` `{"display_name"?,"units"?:{"weight","length","volume","temp"}}`;
  `POST /v1/me/password` `{"current","new"}` (wrong current → 403 `wrong_password`).
- Defaults for a new parent: `kg`, `cm`, `ml`, `c`.

- [ ] **Step 1: Failing tests:**
  - `TestAdminCreatesParent`: 201, invite URL starts with the origin; a second create with the same
    username (any case) → 409 `username_taken`; the admin's username → 422.
  - `TestInviteRedeem`: redeem with a 12-char password → session works; `GET /v1/session` shows
    `kind=parent`.
  - `TestTokenRefusals` (Review Focus 2): redeem twice; redeem after `72h+1s`; reset after `1h+1s`;
    redeem after the admin disabled the parent; an unknown token — all 404 `not_found`, same body;
    after the double redeem the first session still works.
  - `TestResetEndsSessions`: parent with two sessions redeems a reset link → both old sessions 401.
  - `TestNewInviteReplacesOld`: issue twice → first link 404.
  - `TestAdminListShowsNoKidData`: the JSON of `GET /v1/admin/parents` has exactly the keys
    `id, username, display_name, disabled, has_password, created_at`.
  - `TestDisableEndsSessions`; `TestEnableRestoresLogin`.
  - `TestChangePassword`: wrong current → 403; right → other sessions end, the calling one stays;
    a 257-char new password → 422 field `new`.
  - `TestUnitsValidation`: `"weight":"stone"` → 422 field `units.weight`.
  - `TestParentCannotReachAdmin` / `TestAdminCannotReachMe`: 403.
- [ ] **Step 2:** Run → FAIL. **Step 3:** Implement. **Step 4:** `go test ./... -race` → PASS.
- [ ] **Step 5:** Commit `feat(api): parent accounts, invites, resets and own profile`.

### Task 6: Kids and per-kid access

**Files:**
- Create: `api/internal/httpapi/{kids.go,access.go,kids_test.go,access_test.go}`,
  `api/internal/store/{kids.go,access.go,access_test.go}`
- Modify: `api/internal/store/parents.go` (`SetParentDisabled`), `api/internal/httpapi/routes.go`

**Interfaces:**
- Produces:
  ```go
  func (s *Store) CreateKid(ctx, k model.Kid, creator string, now time.Time) (model.Kid, error) // kid + guardian grant in one tx
  func (s *Store) KidsFor(ctx, parentID string) ([]model.KidWithRole, error)
  func (s *Store) Kid(ctx, kidID string) (model.Kid, error)
  func (s *Store) UpdateKid(ctx, k model.Kid, by string, now time.Time) error
  func (s *Store) DeleteKid(ctx, kidID string) error
  func (s *Store) RoleFor(ctx, kidID, parentID string) (model.Role, error)  // ErrNotFound when no grant
  func (s *Store) AccessList(ctx, kidID string) ([]model.Grant, error)      // parent id, username, display name, role
  func (s *Store) SetAccess(ctx, kidID, username string, role model.Role, now time.Time) error // ErrNotFound unknown/disabled; ErrLastGuardian
  func (s *Store) RemoveAccess(ctx, kidID, parentID string) error           // ErrLastGuardian
  ```
- **The last-guardian rule lives in the store, in the same immediate transaction as the write.** After
  the change, the kid must have ≥ 1 `guardian` grant whose parent has `disabled = 0`; otherwise roll
  back and return `ErrLastGuardian`. `SetParentDisabled(…, true)` applies the same check to every kid
  that parent guards.
- HTTP: `ErrLastGuardian` → 409 `last_guardian`. For the admin's disable the body is exactly
  `{"error":"sole_guardian"}` — no kid id, name or count.
- `POST /v1/kids` `{"name","birth_date","sex","gestation_weeks"?,"gestation_days"?,"time_zone"}`.
  Validation (422 with `field`): name 1–64 chars; `birth_date` a real `YYYY-MM-DD`, not after today in
  `time_zone`, not before 1900-01-01; `sex` `female|male`; gestation weeks 22–44 with days 0–6, both
  or neither; `time_zone` loadable with `time.LoadLocation` and not empty or `Local`.
- `DELETE /v1/kids/{kid}` body `{"confirm_name"}` must equal the kid's name exactly → 204; otherwise
  422 field `confirm_name`.
- `PUT /v1/kids/{kid}/access/{username}` `{"role"}`: username normalised; unknown or disabled parent →
  404 `parent_not_found` (same body for both).
- `DELETE /v1/kids/{kid}/access/{parent}`: a caregiver may remove only themselves; a guardian anyone.

- [ ] **Step 1: Failing tests:**
  - `TestCreateKidMakesGuardian`; `TestKidsForListsOnlyGranted`.
  - `TestKidValidation` (Review Focus 4): tomorrow's date; `2025-02-30`; `1899-12-31`; `time_zone`
    `"Mars/Olympus"`, `""`, `"Local"`; weeks 21; days without weeks — each 422 naming the field, and
    `KidsFor` stays empty.
  - `TestNoGrantIs404`: a parent without a grant and a random kid id get the same status and body on
    `GET`, `PATCH`, `DELETE` and `…/access`.
  - `TestCaregiverLimits`: can `GET` the kid; `PATCH`, `DELETE`, `GET …/access`, `PUT …/access` → 403;
    can remove themselves; cannot remove someone else.
  - `TestGrantByUsername`: `" Bo "` grants `bo` (Review Focus 3); unknown and disabled → identical 404.
  - `TestLastGuardian`: sole guardian downgrades self → 409; removes self → 409; with a second
    guardian both succeed.
  - `TestLastActiveGuardian`: guardians A and B; admin disables A (ok); B removes self → 409; admin
    disables B → 409 body exactly `{"error":"sole_guardian"}`.
  - `TestConcurrentRevokes` (Review Focus 1): guardians A and B each `RemoveAccess` the other from two
    goroutines, 50 rounds → exactly one succeeds each round and one guardian remains.
  - `TestDeleteKid`: wrong `confirm_name` → 422; right → 204, grants gone, other kids untouched.
  - `TestDisabledParentGrantStays`: re-enabling a parent restores their kids.
- [ ] **Step 2:** Run → FAIL. **Step 3:** Implement. **Step 4:** `go test ./... -race` → PASS.
- [ ] **Step 5:** Commit `feat(api): kids and per-kid access with the last-guardian rule`.

### Task 7: The access matrix

**Files:**
- Create: `api/internal/httpapi/matrix_test.go`

**Interfaces:**
- Consumes: `Routes(d)` and the testkit.

- [ ] **Step 1: Write the test.** `TestAccessMatrix` builds one fixture (admin; parents `g` guardian,
  `c` caregiver and `x` with no grant on kid `K`; one unused invite token) and, **for every entry of
  `Routes()`**, sends the request as each of: no internal token, no session, admin, `x`, `c`, `g`.
  Expected status class per access class:

  | Access | no internal token | no session | admin | x (no grant) | c | g |
  |---|---|---|---|---|---|---|
  | Open | ok | ok | ok | ok | ok | ok |
  | Internal | 401 | ok | ok | ok | ok | ok |
  | Session | 401 | 401 | ok | ok | ok | ok |
  | Admin | 401 | 401 | ok | 403 | 403 | 403 |
  | Parent | 401 | 401 | 403 | ok | ok | ok |
  | Caregiver | 401 | 401 | 403 | 404 | ok | ok |
  | Guardian | 401 | 401 | 403 | 404 | 403 | ok |

  "ok" means "not 401, 403 or 404-for-access" (a 422 from an empty body counts). Path parameters are
  filled from the fixture by name (`{kid}`, `{parent}`, `{username}`, `{token}`, `{session}`). The test
  **fails if a route's pattern has a parameter the filler does not know**, so a new route cannot skip
  the matrix. `DELETE /v1/kids/{kid}/access/{parent}` is declared `Caregiver` and the matrix targets
  the caller's own id for `c`.
- [ ] **Step 2:** Run `go test ./internal/httpapi/ -run TestAccessMatrix -v` → PASS for every route
  (fix any route whose access class is wrong; that is the point of the task).
- [ ] **Step 3:** Prove the guard: temporarily register a route with an unknown parameter, see the
  test fail, remove it.
- [ ] **Step 4:** Commit `test(api): access matrix over the route table`.

### Task 8: UI foundation — shell, session, login, invite

**Files:**
- Create: `ui/` scaffold; `ui/src/hooks.server.ts`, `ui/src/app.d.ts`,
  `ui/src/lib/server/{env,constants,api,cookie}.ts` + tests, `ui/src/lib/types.ts`,
  `ui/src/lib/components/AppShell.svelte`, routes `healthz`, `healthz/api`, `login`, `logout`,
  `invite/[token]`, `reset/[token]`, `+layout.server.ts`, `+layout.svelte`, `+error.svelte`,
  `ui/e2e/stub-api.ts`, `ui/e2e/{login,invite,unavailable,shell}.spec.ts`

**Interfaces:**
- Consumes: the API route table above.
- Produces:
  - `getEnv(): { internalApiToken, apiBaseUrl, origin, port }` — refuses to start on a token under 32
    characters or a non-`http(s)` URL (as commsgun's `env.ts`).
  - `createApi(env, fetch, session?: string, clientAddress?: string)` returning typed methods, each
    `Promise<ApiResult<T>>` where
    `ApiResult<T> = {ok:true,data:T} | {ok:false,status:number,reason:'unavailable'|'unauthorized'|'forbidden'|'not_found'|'invalid'|'conflict'|'rate_limited'|'error',code?:string,field?:string}`.
    Timeout `API_TIMEOUT_MS = 5000`. One fixed-shape log line per failure: endpoint name and status only.
  - `event.locals.session: { kind:'admin'|'parent', parent?: {id,username,displayName,units} } | null`
    and `event.locals.api`.
  - Cookie `hbt_session`: `HttpOnly`, `Secure`, `SameSite=Strict`, `Path=/`, `Max-Age` 30 days
    (12 hours for the admin).
- Start by copying from `commsgun/ui`: `package.json` (rename, same versions), `pnpm-lock.yaml`
  regenerated, `svelte.config`/`vite.config.ts`, `tsconfig.json`, `eslint.config.js`,
  `prettier.config.js`, `components.json`, `playwright.config.ts`, `src/app.html`,
  `src/routes/layout.css`, `e2e/stub-api.ts` (the pattern: Playwright starts a stub of the API).
- `hooks.server.ts`: origin check exactly as commsgun (400 `Bad origin` except `/healthz*`); read the
  cookie; `GET /v1/session`; on `unauthorized` clear the cookie; on `unavailable` render the
  unavailable page with status 503 and **keep the cookie** (Review Focus 5). Unauthenticated requests
  outside `/login`, `/logout`, `/invite/*`, `/reset/*`, `/healthz*` → 303 to `/login?next=…`. An admin
  session is sent to `/admin/parents` from anything outside `/admin/*`; a parent session gets 404 on
  `/admin/*`.
- Every response under the authenticated layouts sets `Cache-Control: no-store`.
- `AppShell`: bottom tab bar below 768 px, side navigation from 768 px, same links in both. Parent
  links in this plan: Home, Account. Admin links: Parents.
- `/healthz` → 200 `ok` with no API call; `/healthz/api` → the API's `/healthz` status passed through
  (503 on timeout).

- [ ] **Step 1:** Scaffold and copy the config files; `pnpm install`; `pnpm check && pnpm lint` clean
  on the empty app.
- [ ] **Step 2: Failing unit tests** (Vitest): `env.test.ts` (refusals, as commsgun's);
  `api.test.ts` — sends `authorization`, `x-session-token` and `x-client-address` headers; maps 401,
  403, 404, 409, 422 (with `field`), 429, 5xx, network error and timeout to the reasons above; never
  logs a token, a body or a query; `cookie.test.ts` — attributes for both kinds.
- [ ] **Step 3:** Run `pnpm test` → FAIL. Implement. Run → PASS.
- [ ] **Step 4: Failing e2e** against the stub API, each at 390×844 and 1280×800 with an axe check:
  - `login.spec.ts`: parent logs in and lands on `/`; admin lands on `/admin/parents`; wrong password
    shows one generic message; 429 shows "Too many attempts, try again in a minute"; `next=` is
    honoured only for same-origin paths.
  - `invite.spec.ts`: valid link shows the username and a set-password form; mismatched confirmation
    and an 11-character password are refused in the form; success logs in; an invalid link shows
    "This link is no longer valid" for every failure cause.
  - `unavailable.spec.ts`: stub returns 503 → the unavailable page, the cookie is still present, a
    reload after the stub recovers shows the app.
  - `shell.spec.ts`: bottom bar at phone width, side navigation at desktop width, same link set;
    `Cache-Control: no-store` on `/`; a foreign `Host` gets 400.
- [ ] **Step 5:** Run `pnpm test:e2e --trace off` → FAIL. Implement routes and components. Run → PASS.
- [ ] **Step 6:** `pnpm check && pnpm lint && pnpm test` → clean. Commit `feat(ui): shell, session, login and invite`.

### Task 9: UI — admin, kids, access, account

**Files:**
- Create: `ui/src/routes/admin/parents/`, `ui/src/routes/(app)/{+layout.server.ts,+layout.svelte,+page.*}`,
  `(app)/kids/new/`, `(app)/kids/[kid]/settings/`, `(app)/kids/[kid]/access/`, `(app)/account/`,
  `ui/src/lib/components/KidSwitcher.svelte`, `ui/src/lib/format.ts` + test,
  `ui/e2e/{admin,kids,access,account}.spec.ts`

**Interfaces:**
- Consumes: Task 8 (`locals.api`, `locals.session`, `AppShell`).
- Produces: `formatAge(birthDate: string, today: string): string` ("4 months 12 days", "2 years 3
  months"; days only under 1 month; months and days under 1 year; years and months after) — used by
  later plans; `KidSwitcher` (props `kids`, `currentId`) — the selected kid is the `[kid]` route
  parameter, with the last one remembered in a `hbt_kid` cookie for `/`.
- All mutations are SvelteKit form actions (they work without client JavaScript and keep the token
  server-side).
- Pages:
  - `/admin/parents`: list (username, display name, status); "Add parent" form; after create or
    "Reset link", the link is shown once with a copy button and the text "Send this link to the parent.
    It works once and expires in 72 hours" (1 hour for a reset); Disable / Enable; the `sole_guardian`
    refusal reads "This parent is the only guardian of at least one child. Another guardian must be
    added first."
  - `/`: the kid switcher and, per kid, name and age, with links to settings and access. No kids →
    "Add your first child". (The real home screen is plan 3.)
  - `/kids/new` and `/kids/[kid]/settings`: name, birth date, sex, "Born early?" revealing weeks and
    days, time zone (defaulted from `Intl.DateTimeFormat().resolvedOptions().timeZone`, editable).
    Settings shows the delete section to guardians only; delete requires typing the name.
  - `/kids/[kid]/access` (guardians): current grants with role; add by exact username with a role
    picker; change role; remove; "Leave" for oneself. `parent_not_found` reads "No active account has
    that username." `last_guardian` reads "A child needs at least one guardian."
  - `/account`: display name, the four unit pickers, change password, list of sessions with "Sign out"
    for each other one.
  - A caregiver sees no settings-edit or access links for that kid; a direct visit shows the 404 page.

- [ ] **Step 1: Failing unit test** `format.test.ts`: `formatAge('2026-07-08','2026-11-20')` →
  `"4 months 12 days"`; same day → `"0 days"`; `('2024-08-20','2026-11-20')` → `"2 years 3 months"`;
  29 Feb birth dates on non-leap years.
- [ ] **Step 2: Failing e2e** (stub API; each flow at both widths, axe on every page):
  `admin.spec.ts` (create → link shown once; duplicate username message; disable refusal text);
  `kids.spec.ts` (create with and without "born early"; field errors from 422 land on the right field;
  delete needs the exact name); `access.spec.ts` (grant, change role, remove, both refusal texts;
  caregiver view has no admin controls); `account.spec.ts` (units persist; wrong current password
  message; sign out another session).
- [ ] **Step 3:** Run → FAIL. Implement. Run `pnpm check && pnpm lint && pnpm test && pnpm test:e2e --trace off` → PASS.
- [ ] **Step 4:** Commit `feat(ui): admin, kids, access and account pages`.

### Task 10: Images, smoke, CI, docs

**Files:**
- Create: `api/Dockerfile`, `ui/Dockerfile`, `ui/.dockerignore`, `scripts/smoke.sh`,
  `.github/workflows/ci.yml`, `CLAUDE.md`
- Modify: `README.md`

**Interfaces:**
- Consumes: both apps.
- Dockerfiles: commsgun's, renamed; same base digests; API `EXPOSE 9210`, `ENV DATA_DIR=/data`,
  `VOLUME /data`, uid 10001; UI `PORT=9212` and the four proxy-header env vars.
- `ci.yml`: commsgun's three jobs (`api`, `ui`, `images`) renamed, same action SHAs, same
  push-on-`main` and digest summary.

- [ ] **Step 1: Write `scripts/smoke.sh <api-image> <ui-image>`** (structure and helpers of commsgun's:
  random names, private network, volume, `--read-only`, cleanup trap, `pass`/`fail`). It asserts, all
  through the UI on `127.0.0.1:9212` with `X-Forwarded-*` headers except where noted:
  1. `/healthz` and `/healthz/api` are 200; the API port is **not** published.
  2. A wrong `Host` gets 400.
  3. Admin logs in; creates parent `smoke-a`; the invite link is redeemed; `smoke-a` creates a kid.
  4. Admin creates `smoke-b`; `smoke-a` grants `smoke-b` caregiver; `smoke-b` sees the kid and gets a
     404 page on its access screen.
  5. Admin disabling `smoke-a` is refused; after `smoke-a` makes `smoke-b` a guardian it succeeds.
  6. The admin's pages and API answers never contain the kid's name (grep the captured bodies).
  7. Restart the API container: `smoke-b`'s session still works (data on the volume).
  8. Restart the API with a different `ADMIN_PASSWORD`: the old admin cookie is refused.
  9. `docker logs` of both containers contain none of: the usernames, the kid's name, any password or
     token used.
  10. Resident memory ≤ 60 MiB (API) and ≤ 140 MiB (UI).
- [ ] **Step 2:** `shellcheck scripts/smoke.sh` clean; `docker build -t hbt-api:dev api && docker build -t hbt-ui:dev ui && scripts/smoke.sh hbt-api:dev hbt-ui:dev` → every line `PASS`.
- [ ] **Step 3:** Add `ci.yml`. Push the branch; the three jobs are green on the PR.
- [ ] **Step 4:** `README.md`: what it is, the two images and ports, the env var tables of spec 13,
  run locally, tests, release (merging deploys nothing; the edge pins digests). `CLAUDE.md`: the
  "How to Act" block and structure of commsgun's, with this repo's commands, the Global Constraints
  above as rules, and a pointer to the spec and plans.
- [ ] **Step 5:** Commit `build: images, smoke, CI and docs`.

---

## Done when

- `cd api && gofmt -l . && go vet ./... && GOTOOLCHAIN=$(go env GOVERSION) go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./... && go test ./... -race` is clean.
- `cd ui && pnpm install --frozen-lockfile && pnpm check && pnpm lint && pnpm test && pnpm test:e2e --trace off` is clean.
- `scripts/smoke.sh` prints only `PASS` lines; CI is green on the PR.

The report for this plan says which image content changed (both, new) and that nothing is deployed:
the edge work is plan 2.
