# happybabytracker — data the redesigned UI still needs: design

Status: **approved by the owner** (2026-11-26; open questions answered in 8). Amends `2026-11-20-happybabytracker-design.md` (5.3, 7.3, 10.1).
Plans: written only after this spec is approved (`docs/superpowers/plans/`).

Items marked **(owner)** were decided by the owner in the design conversation of 2026-11-25/26.
Items marked **(call)** are the designer's choices, not owner decisions: correct any of them.

## 1. Intent

PR #16 restyled `ui/` to `docs/plans/ui-redesign/` and left elements unbuilt because the API or the loaders did not
supply the data. This spec covers the elements that need **new API surface** and fixes the contract of the UI-only ones
so the plan has one place to read. It adds no module, no route family and no env var.

## 2. Decisions

| # | Element | Decision |
|---|---|---|
| 1 | Home tile "x ago" and Today "Last:" | One call: a `last` field on `GET /v1/kids/{kid}/home` **(owner)** |
| 2 | "Started on Cleo's phone" | Dropped. No `created_by` is exposed **(owner)** |
| 3 | Device names on sessions | A coarse label stored with the session **(owner)**. Places dropped **(call)**: no IP is stored |
| 4 | "Sign out my other devices" checkbox | Dropped; the sheet states the behaviour. `POST /v1/me/password` is unchanged **(owner)** |
| 5 | Admin "Last active" | The newest `last_seen_at` among the parent's sessions **(owner)** |
| 6 | Undo for deleting a tracker entry | Client-held re-POST, seven daily trackers only **(owner)** |
| 7 | Admin notification bell | Dropped **(call)**: the admin has no kid and every notification route is `Parent` class |

## 3. API changes

Every change is additive. No existing field changes meaning, and no route is removed or renamed.

### 3.1 `GET /v1/kids/{kid}/home` (Caregiver): `last`

```json
"last": { "bottle": {…}, "nursing": null, "sleep": {…} }
```

- Keys are the timeline type names (`bottle nursing pumping sleep diaper meal symptom`). A key is present for every
  tracker whose module is enabled; the value is the newest entry of that tracker as the timeline row view
  (`newTimelineView`: `type`, `at`, and the tracker's own fields), or `null` when the kid has none.
- Newest means the greatest `(at, id)`, the order the timeline already uses. A running sleep or nursing session is
  the newest by its start.
- Cost: one indexed `LIMIT 1` per enabled tracker (`kid_id, at` indexes exist), inside the read the handler already
  does. No new table.
- The UI derives "1 h 34 min ago" and the "Last:" text from `at` and the entry's fields. The elapsed clock and the
  nursing side tiles need nothing new: `running.*.started_at`, `side`, `side_since`, `left_seconds`, `right_seconds`
  are already returned.

### 3.2 Password change date

- Migration `0010`: `ALTER TABLE identities ADD COLUMN secret_changed_at INTEGER` (nullable, unix seconds UTC).
  Existing rows stay NULL: their change date is unknown.
- `setPassword` writes `now` with the hash (invite and reset redemption and `POST /v1/me/password` all go through it).
  The store method takes `now time.Time` like its siblings.
- `GET /v1/session`, `parent` object: `password_changed_at` (RFC3339 UTC, or `null`). `parentSelect` already joins
  `identities`, so no extra query. The UI shows "Last changed N days ago" only when it is not null.

### 3.3 Session device label

- Migration `0010` (same file): `ALTER TABLE sessions ADD COLUMN device_label TEXT` (nullable).
- `POST /v1/session` (Internal) accepts an optional `device_label`. Rule: `^[A-Za-z0-9 ]{1,48}$` (constant
  `DeviceLabelPattern` in `constants.go`); a value that fails it is `422 invalid` on field `device_label`. Absent is
  stored as NULL. Admin sessions ignore it.
- The UI builds the label from a **closed set** (browser name + operating system name, for example "Safari on
  iPhone"; anything unrecognised is "Browser"), never from the raw user agent, so a valid request never fails the rule.
  The raw user agent, the IP and any place are neither sent nor stored. The IP still goes only to the login limiter.
- `GET /v1/me/sessions`: each row gains `device_label` (string or `null`). The UI shows "Browser" for null.
- "Sign out all other devices" is UI-only: it calls `DELETE /v1/me/sessions/{session}` for each non-current row.
  Not atomic (a session created in between survives); acceptable, the page reloads and shows it.

### 3.4 Access list: `disabled`

- `GET /v1/kids/{kid}/access` (Guardian): each grant gains `disabled` (bool), from `parents.disabled`, already
  joined by `AccessList`. A disabled parent already counts as having no grant (rule 3); this only labels the row for
  the guardians who already see their username and display name. No new information about the kid.

### 3.5 Delete-a-child counts: new route

- `GET /v1/kids/{kid}/counts` (**Guardian**): `{ "entries": N }`, the number of rows in the seven daily-tracker tables
  (`bottle_feeds`, `nursing_sessions`, `pumping_sessions`, `sleep_sessions`, `diapers`, `meals`, `symptoms`) for the
  kid, matching the design's "N logged entries: feeds, sleep, diapers, solids and pumping" **(call)**. Growth,
  vaccination, dose and milestone rows are named in the consequence list without a number.
- A separate route, not a field on `getKid`: `getKid` is Caregiver and runs on every page; seven `COUNT(*)` must not.
- The UI already knows the number of people from the access list it loads.
- "Export a PDF first" is a link to the existing export page: UI-only.

### 3.6 Admin list: `last_active_at`, `link`

`GET /v1/admin/parents` (Admin), each parent gains:

- `last_active_at`: RFC3339 UTC of `MAX(sessions.last_seen_at)` over the parent's session rows, or `null` when the
  parent has none **(owner)**. `last_seen_at` is touched at most once per `SessionTouchInterval` (1 h), so the value
  is accurate to an hour. Disabling a parent deletes their sessions (`SetParentDisabled`), so a disabled parent reads
  `null`: the UI says "No recent sign-in", never "Hasn't signed in yet", for a parent who has a password.
- `link`: `null`, or `{ "kind": "invite"|"reset", "issued_at", "expires_at" }` for the parent's unused one-time token
  (`IssueToken` keeps at most one per parent and kind; take the newest `expires_at`). `issued_at` is
  `expires_at − InviteTTL|ResetTTL` **(call)**: the table has no `created_at`, tokens live at most 72 h, and a column
  would be NULL for every live token at the upgrade. If a TTL constant ever changes, tokens issued before the change
  report a wrong `issued_at` until they expire; that is the price of not adding a column.
- Both are facts about the parent, never about a kid: the admin still sees no kid, no grant and no count.

## 4. UI-only contracts (no spec needed, listed so the plan covers them)

Home Latest list (`timeline?limit=3`); running clock; nursing side tiles; Timeline week strip (`summary`), since-last,
running cards, day filter, age line; Growth units control (`PATCH /v1/me`), `?sel=`, tick size; Account sheets and
the disclaimer on the password sheet ("Changing your password signs you out of every other device."); remove-access
and role-change sheets; Undo for access and milestone changes.

### 4.1 Undo for deleting a tracker entry (client-held re-POST)

- Scope: the seven daily trackers. Growth, doses, vaccinations and milestone deletes keep their confirm.
- The delete form (`?/delete`, a `use:enhance` form on the entry page) keeps the loaded entry in `sessionStorage` under
  a short-lived key, then redirects to the timeline day as today. The timeline shows a toast "Entry deleted · Undo" for
  6 s (the success-toast duration of DESIGN 4.5 / PATTERNS P3).
- **Amends DESIGN 4.5**, which offers Undo "not for permanent deletion" **(owner)**: for these seven trackers a delete
  stays a real delete on the server; Undo is a new create. The design system's line is updated with this change.
- Undo POSTs the stored fields to the tracker's create route, through the existing form action (never the API
  from the browser: the browser only talks to the UI, rule in CLAUDE.md). The entry returns with a **new id** and the
  restorer as `created_by`, `created_at` now. Its `at` is unchanged, so it lands where it was.
- Failure modes, each shown in the toast, none retried silently: the kid has since started another running timer
  (`already_running`, 409); the grant was revoked meanwhile (404, byte-identical to an unknown kid); the entry now
  fails validation (400).
- No JavaScript: the delete still works and there is no Undo. The payload never leaves the tab and is cleared when the
  toast closes, on sign-out and after 6 s. It holds a note, which is health data: it is never logged or sent to OTEL.
- Optimistic UI: the browser calls no API, so there is nothing to await. The row is removed by the redirect, and the
  Undo result is shown by invalidating the page.

## 5. Access matrix and rules

| CLAUDE.md rule | Effect |
|---|---|
| 2 Admin never sees a kid | `/home`, `/counts` and the access list stay Caregiver/Guardian. The admin list adds parent facts only. |
| 3 No grant is 404 | `GET /v1/kids/{kid}/counts` follows the Guardian row of `matrixWant`: 404 for a parent with no grant (byte-identical to an unknown kid), 403 for a caregiver, 403 for the admin, 401 without a session. |
| 4 A kid keeps a guardian | No change: no new route removes or downgrades. Undo of a role change or removal re-`PUT`s through `SetAccess`, whose last-guardian check lives in its transaction. A self-removal is not undoable (the actor is no longer a guardian). |
| 5 No PII in logs | `device_label`, `password_changed_at`, `last_active_at`, `link`, counts and the Undo payload are never log fields. Tests assert the access log lines contain none of them. |
| 8 Instants | Both new columns are unix seconds UTC; the wire is RFC3339 UTC. |
| 7 Limits | `DeviceLabelPattern` joins `api/internal/constants/constants.go`; the UI's closed set and the 6 s toast duration live in `ui/src/lib/server/constants.ts` / `ui/src/lib/constants.ts`. No literal elsewhere. |

`routes.go` and `routeClasses` change together: one added route, `GET /v1/kids/{kid}/counts` as `Guardian`. No other
route changes class. `matrix_test.go` must pass unchanged otherwise.

## 6. Migration, rollback and the edge contract

- One migration, `0010_redesign_data.sql`, two `ALTER TABLE … ADD COLUMN` (nullable). Forward-only like the rest.
- **Rollback is not free.** The binary refuses a database whose schema is newer than itself
  (`database schema version N is newer than this binary`). Redeploying the previous API digest after `0010` has
  run fails to start. Rolling back needs the `/data` volume restored from before the bump, or a hotfix image that
  carries `0010`. The release note must say so, and the edge's bump should be preceded by a copy of `/data`.
- No change to the contract with `davcs86/homelab-edge`: API port 9210 (never published), UI port 9212, env var names,
  health paths, `/data`, image names, uid 10001, read-only root. The migration writes only under `/data`.
- The UI image is rolled out together with the API: a new UI against an old API sees missing optional fields and must
  render without them (every new field is optional on the wire), but an old UI against the new API is also valid.

## 7. Tests and verification

- API: store tests for each new column (set, read, NULL), the home `last` for every module combination, the counts
  query, the admin aggregates (disabled parent, no sessions, an expired token), the matrix, and the log-scrub test.
- UI: unit tests for the label builder, the relative-time and "Last:" formatters, the Undo state machine (including
  the three failure modes); the e2e stub (`ui/e2e/stub-api.ts`) is updated to the API's real behaviour for every
  field above, and `scripts/smoke.sh` is extended for the `last`, `counts` and `device_label` paths.
- Commands as in CLAUDE.md. `scripts/smoke.sh` needs Docker; where the container has none, it is reported as not run.

## 8. Decisions on the open questions (owner, 2026-11-26)

1. **Rollback (6):** a `/data` copy before the bump is acceptable; `0010` stays.
2. **Counts (3.5):** the seven daily trackers only.
3. **`issued_at` (3.6):** derived from `expires_at`, no `created_at` column.
