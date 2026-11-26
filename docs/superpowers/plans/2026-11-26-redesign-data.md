# Redesign data Implementation Plan (plan 9)

Spec: `docs/superpowers/specs/2026-11-26-redesign-data-design.md` (approved 2026-11-26). Branch `api-redesign-data`, PR base `main`.
Rules: TDD (failing test first, run it, then the code), one commit per task, minimum code, comments are constraints only.

**Pass conditions (stated before starting; each is claimed only with this session's output):**
- API: `cd api && gofmt -l .` prints nothing; `go vet ./...`; staticcheck on the module's Go; `go test ./... -race` all green.
- UI: `pnpm check && pnpm lint && pnpm test && pnpm test:e2e` green (Node 24 in a scratch dir; the image has Node 22).
- `scripts/smoke.sh hbt-api:dev hbt-ui:dev`: every line `PASS:`; if Docker cannot be started here it is reported as **not run**.

## Part A: API (`api/`)

| Task | Test first | Change |
|---|---|---|
| A1 | `store_test.go`: the six `schema_version` assertions become `"8"`; `TestMigration0008KeepsRows` (applies `ms[:7]`, inserts rows, then 0008, rows intact, new columns NULL) | `migrations/0008_redesign_data.sql`: `identities.secret_changed_at INTEGER`, `sessions.device_label TEXT` |
| A2 | `parents_test`, `tokens_test`, `me_test`, `session_test`: set by `SetPassword` and `RedeemToken`; NULL before; `GET /v1/session` carries `password_changed_at` | `SetPassword(..., now)`; `setPassword` writes it; `parentSelect` reads it; `model.Parent.PasswordChangedAt *time.Time`; `parentView.password_changed_at` |
| A3 | `sessions_test`, `session_test`, `me_test`: stored and listed; invalid pattern is 400 on `device_label`; admin ignores it; never in the log | `constants.DeviceLabelPattern`; `CreateSession(..., deviceLabel)`; `login` reads it; `SessionInfo.DeviceLabel`; `sessionInfoView.device_label` |
| A4 | `access_test` (store and http): `disabled` true for a disabled grantee, false otherwise | `model.Grant.Disabled`; `AccessList` selects `p.disabled`; `grantView.disabled` |
| A5 | `kids_test`/`access_test`: counts across the seven tables; zero kid; non-guardian answers per the Guardian row; matrix | `Store.EntryCount`; `d.kidCounts`; route `GET /v1/kids/{kid}/counts` Guardian + `routeClasses` |
| A6 | `admin_test`, `sessions_test`, `tokens_test`: `last_active_at` max over sessions, null with none, null after disable; `link` for the unused token, expired token kept, used token absent, `issued_at = expires_at - TTL` | `Store.ParentActivity`, `Store.PendingLinks`; `adminParentView.last_active_at`, `.link` |
| A7 | `home_test`: `last` has a key per enabled module only; newest entry; `null` when empty; running sleep counts | `homeView.Last map[string]any`, built with `Store.Timeline(..., limit 1)` per enabled type and `newTimelineView` |
| A8 | `middleware_test`/`log` test: none of the new values appears in an access-log line; `matrix_test` green | spec 5; DESIGN 4.5 line amended (`docs/plans/ui-redesign/design-system/DESIGN.md`) |

## Part B: UI (`ui/`)

| Task | Scope |
|---|---|
| B1 | Types (`lib/types.ts`), client mapping (`lib/server/api.ts`), `ui/e2e/stub-api.ts` parity for every field in Part A; `lib/server/constants.ts` / `lib/constants.ts` (Undo 6 s, closed device-label set) |
| B2 | Home: Latest list, tile "x ago" and "Last:" lines, running clock, nursing side tiles |
| B3 | Timeline: week strip, since-last, running cards, day filter, age line |
| B4 | Undo: a toast helper plus Access (role, remove, add), milestones (mark/unmark), tracker-entry delete (spec 4.1) |
| B5 | Account: edit-name and password sheets with the disclaimer, device labels (UI builds them from a closed set at login), "Last changed", "Sign out all other devices" |
| B6 | Access: `disabled` badge, remove confirm; child delete: counts, "Export a PDF first" link; Admin: last active, link line (no bell) |
| B7 | Growth: units segmented control, `?sel=` and the Selected card, responsive tick labels |

Each UI task: unit tests first, then the page, then the e2e spec and the stub; `scripts/smoke.sh` markup scraping must keep passing, so every changed heading, link name and button name is checked against it before the task is committed.

## Part C: smoke and release

- `scripts/smoke.sh`: add steps for `home.last`, `counts` (guardian 200, caregiver 403, outsider 404) and `device_label` on a login.
- Release note: `api/` and `ui/` sources change; migration `0008` is forward-only and the API refuses to start on an older binary after it, so take a copy of `/data` before the edge bumps the pin; `davcs86/homelab-edge` still needs both pins bumped and a deploy.
