# Sickness, Medication and Reminders Implementation Plan (plan 6, app side)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Parents log symptoms (a fever with its temperature), keep medication courses with a schedule, log doses, and get a generic reminder ("Medication due for <kid>") on every device or ntfy topic they opted in for that kid. The home "Due" list and each course's "next due" read the same reminder rows the worker sends from.

**Architecture:** Migration 0006 adds the symptom, medication, dose, reminder, delivery, push and ntfy tables. A pure schedule engine (`api/internal/schedule`, imported by the store and the worker) lists a course's occurrences in the kid's zone. The store keeps, for every active scheduled course, a `reminders` row for each occurrence due in the last 30 minutes and for the next one ahead (unique on medication and due time); course, dose and kid-zone writes adjust those rows in their own transaction. A stdlib-only Web Push package (`api/internal/push`, from the tested prototype in the research notes) and a small ntfy publisher (`api/internal/ntfy`) deliver. One worker goroutine (`api/internal/reminders`) ticks every 30 s: it materialises rows, creates one delivery per allowed target when a reminder falls due, sends each target on its own and settles it. The UI adds the sickness tracker, the medication screen, an account notifications screen, a service worker (install, push, notification click and an offline page) and the medication entries of the home Due list.

**Tech Stack:** Go 1.27 (`crypto/ecdh`, `crypto/ecdsa`, `crypto/hkdf`, `crypto/aes`, `crypto/cipher`, `net/http`), SQLite via `modernc.org/sqlite`; SvelteKit 3.1.1 / Svelte 5 runes, Tailwind 4, a SvelteKit service worker. No new Go or npm dependency (Open value 6).

**Spec:** `docs/superpowers/specs/2026-11-20-happybabytracker-design.md` sections 2, 5.2, 6.1–6.4, 7.2–7.3, 10.2, 10.3, 11, 11.1, 13.1–13.3, 14, 16. Earlier plans: `docs/superpowers/plans/` (foundation, daily trackers, growth, vaccination, advanced logging). Research notes (2026-11-22, rechecked 2026-11-23): `docs/superpowers/research/reminders/{web-push,ntfy-and-edge}.md` and the stdlib prototype `docs/superpowers/research/reminders/pushproto/*.txt` (its three tests pass on Go 1.27.0 as of 2026-11-23; rename to `.go` when copied into `api/internal/push`).

**Not in this plan:** anything in `davcs86/homelab-edge`. That repository is a separate plan and PR whose `CLAUDE.md` binds it. **Deploy order (hard):** plan 6 makes `VAPID_PRIVATE_KEY` and `VAPID_SUBJECT` required at start, and the edge does not render them today (`scripts/happybabytracker-config-test.py::test_vapid_is_not_rendered_in_plan_2`, runbook `happybabytracker.md` section 7). The edge must render both into `api.env` **before** the `bump-happybabytracker` PR that pins plan-6 images is deployed, or the API refuses to start (Owner question 1). ntfy ships off (no `NTFY_*`); tests and smoke use `HBT_ALLOW_FAKE_PUSH=1`.

## Global Constraints

- Every limit, cap, duration and tick lives in `api/internal/constants/constants.go` (API) or `ui/src/lib/server/constants.ts`, with client-safe mirrors in `ui/src/lib/constants.ts` (CLAUDE.md rule 7). No literal elsewhere.
- Instants are unix seconds UTC in INTEGER columns and RFC3339 UTC on the wire (rule 8). Fixed times of day are `HH:MM` in the kid's zone. "Every N hours" is a fixed grid from the course's start instant and is **never re-anchored to the last dose logged** (spec 10.3, owner); the app never computes or suggests a safe interval.
- Go's `time.Date` picks either side of a daylight-saving transition without a guarantee (its documentation says so). The schedule engine resolves a wall time explicitly: a time in a spring-forward gap fires at the first valid instant after it, a time in a fall-back overlap fires once at the earlier instant. It never builds local midnight with `time.Date` (the HANDOFF's midnight-DST flaw in `kidtime.BandStart` and friends must not spread here).
- The notification text is generic and fixed: `<Category> due for <kid's name>` ("Medication due for Mia"); it **never names the medication or the dose** (owner). It is built once, in Go, and sent as the push payload's `title` and as the ntfy title and message. The push payload is encrypted (RFC 8291) and carries only `{v, category, kid, title, path, tag}`.
- Logs carry ids (kid, parent, medication, reminder, delivery, device) and enumerated facts (channel, outcome, status class, attempt, duration) only: never a name, medication name, dose, note, temperature, push endpoint, push key, ntfy topic, token or key (spec 13.3, rule 5). A push endpoint is a secret-like capability: never in a log, an error string, a URL or an API response.
- A caller with no grant, an unknown kid and an entry of another kid answer the same 404 body; the admin is refused; writes re-assert the role with `actorRole` inside `inTx`; hard deletes; no edit history. Symptoms, medications, doses and per-kid opt-ins are class `Caregiver`; devices and the ntfy topic are class `Parent`. No request creates or edits a reminder row directly.
- **Delivery is decided at send time:** a reminder goes only to targets whose parent is enabled, holds a grant on the kid, and opted that target in for that kid, while the kid's `medication` module is on and the course is active and within its dates. A row materialised earlier never overrides this (spec 11, 14).
- Reads that combine a row and its children, or a course and its reminders, use one `Store.inReadTx` snapshot through `querier` (plan 5.1). Lists load children with one extra query per page, never one per row.
- Measured values are stored three ways (rule of spec 6.2): a temperature as hundredths of °C, hundredths of °F and the unit typed, converted only in `api/internal/units`. A PATCH merges onto the stored row; an unchanged typed value in the same unit keeps the three stored columns, and that comparison runs **before** the range check, so an unchanged stored value never becomes invalid (`mergeVolume`, `mergeWeight`, `changedOK` are the pattern).
- UI form readers send only what changed, against hidden `*_original` fields, and fail closed on a missing original (plan 5.1). The UI converts nothing: it formats the API's columns. Units come from `formUnits(session)`.
- The stub API (`ui/e2e/stub-api.ts`) mirrors each Go handler's validation order, field names and 404 rules line by line; UI tests are green on a broken contract, so `scripts/smoke.sh` is the real check (CLAUDE.md "How to Act" 4).
- **Every Playwright form save waits for its response with `submit()` from `ui/e2e/helpers.ts`** (a click-then-assert race flaked CI in plan 5).
- `Cache-Control: no-store` on every page that shows kid data (already global in `hooks.server.ts`); every new page except the push enable button works without JavaScript and passes axe at 390×844 and 1280×800; comments state constraints only, at most 2 lines (rule 9); `gofmt -l` prints nothing; staticcheck clean; Prettier tabs.
- Tests that pin counts change with the code they pin: `wantTables` and every `schema_version` assertion (`"5"` → `"6"`, five places in `api/internal/store/store_test.go`), `routeClasses` and `matrixParams` and the fixture in `api/internal/httpapi/matrix_test.go`, `store.TimelineTypes` and `timelineModules`, the home `due` entry key sets (`home_test.go`), `TRACKER_KINDS`/`TRACKER_ROUTE`/`TIMELINE_TYPE`/`TRACKER_TITLE`/`TRACKER_LABEL` and `quick-log.test.ts`, the `tracker` param matcher test, the stub.
- Every place that starts the API (config tests, `cmd/happybabytracker-api/main_test.go`, the httpapi `kit`, `scripts/smoke.sh` `start_api`) gains the now-required VAPID variables in the same task that makes them required.

## Reference data (what the research found; verify by test, do not re-research)

- **Web Push in Go, stdlib only**: the prototype (`pushproto/push.go.txt`, 153 lines) passes the RFC 8291 Appendix A vector byte for byte, an independent decrypt round trip and a VAPID test; rerun on Go 1.27.0 on 2026-11-23: `PASS` (3 tests). Facts: P-256 ECDH, HKDF-SHA256 (`crypto/hkdf.Key`), AES-128-GCM, one record, `rs` 4096, plaintext cap 3993 bytes; headers `Content-Encoding: aes128gcm`, `TTL` (required), `Urgency`, optional `Topic` (≤ 32 of `[A-Za-z0-9_-]`, replaces a pending message with the same topic), `Authorization: vapid t=<jwt>, k=<65-byte public key b64url>`. The ES256 JWT signature is **raw 64-byte r||s** (not DER); `aud` is the endpoint's origin; `sub` the `https:` subject; `exp` ≤ 24 h (RFC 8292). The edge generates the private key as 43-character unpadded base64url (`init_secrets.py::generate_vapid_private_key`, checked 2026-11-23); parse with `ecdsa.ParseRawPrivateKey(elliptic.P256(), d)` after requiring 32 decoded bytes; the public key is `.ECDH()` → `.PublicKey().Bytes()` (65 bytes). `webpush-go` v1.4.0 adds `golang-jwt/jwt/v5` and pads every message to 4096 bytes: not chosen.
- **Statuses**: 201 accepted; 404/410 gone (delete that subscription); 429 (honour `Retry-After`) and 5xx/timeout/network retry; 400/401/403/413 and other 4xx: configuration or payload fault, never retried for that occurrence, subscription kept (a rotated VAPID key shows as 401/403 on every subscription).
- **xstockstrat `services/xstockstrat-notify/src/fanout/webPush.ts` (read 2026-11-23):** copy TTL 3600 s, 10 s timeout, prune only on 404/410, always show a notification on push, a deterministic `tag`, the public key served by the server. **Do not copy:** no retries, sequential sends, no endpoint validation (SSRF), an upsert that silently re-binds a subscription to another user and an unscoped unregister, a payload carrying the alert text, a loose `includes` click match.
- **iOS**: Web Push exists only for a web app added to the Home Screen (iOS/iPadOS 16.4+, manifest `display: standalone`, already set) and the permission request must follow a user gesture (WebKit blog 13878). Apple's push service answers 403 `BadJwtToken` when the JWT `sub` "isn't a URL or mailto:"; one public report saw Apple reject a reserved `.local` subject. The edge's subject is the tailnet-only `https://happybaby.private.<domain>`: Apple's acceptance is **not verified** (needs a real iPhone).
- **SvelteKit 3.1.1** (checked in `node_modules` 2026-11-23): `$service-worker` was removed (`module_removed_service_worker`); use `$app/manifest`, `$app/env`, `$app/service-worker`. `src/service-worker.ts` is auto-registered at scope `/` as a module worker; the Node adapter serves `/service-worker.js` and `static/offline.html` before the hooks (`e2e/pwa.spec.ts` proves the static path). There is no CSP. Playwright (Chromium) can cover a stubbed `PushManager` and a dispatched `PushEvent`; real delivery, iOS and Apple's acceptance of the subject need a real device.
- **ntfy**: JSON publish to the server root (`topic`, `message`, `title`, `click`) with `Authorization: Bearer tk_…`; 4096-byte message limit; per-visitor rate limit 60 burst, 1 per 5 s; a topic nobody subscribes to caches the message (72 h on the edge). On the edge the API cannot reach ntfy today (API on network `happybabytracker` only, ntfy on the default network), `NTFY_URL` would be `http://ntfy:2586` (the app must accept `http:`), and only `david` reads topics under `deny-all` (edge-plan matters, Open values E1–E2).

## Review Focus

Failure modes no task's happy path exercises; each has a test in the named task.

1. **Clock edges.** A fixed time inside the spring-forward gap (`America/New_York` 02:30 on the change day fires 03:00 EDT), a fixed time in the fall-back overlap (01:30 fires once, at the EDT instant), a zone whose DST starts at midnight (`America/Havana`, `America/Santiago`: a 00:00 time and `end_on` both land correctly), a course starting mid-day, one ending today, one with `end_on` before today, a leap day and 31 Dec → 1 Jan; a kid's time-zone change re-materialises fixed-time rows (Tasks 4, 5).
2. **Restart and concurrency.** A restart or a second tick mid-send: no duplicate occurrence, no duplicate delivery to a target that already accepted, a row left `sending` becomes `unconfirmed` and is never resent; one hanging target does not delay another past the timeout, one failing channel never blocks the other (Tasks 11, 12).
3. **Access after materialisation.** Grant revoked, parent disabled, module switched off, course made inactive or deleted, the kid deleted, a dose logged 29 minutes early, 1 minute before the push, or for the previous occurrence: nothing is sent where nothing may be, and a dose marks exactly one occurrence done (Tasks 6, 9, 12).
4. **Hostile subscriptions.** `http:`, a userinfo URL, a non-443 port, an IP literal, `localhost`, a hostname resolving to a private, loopback, link-local or ULA address (at registration **and** at dial time), a redirect, a compressed or invalid `p256dh`, an `auth` not 16 bytes, an endpoint over 2048 bytes, the same endpoint registered by a second parent, 404/410 deleting only that subscription, 429/5xx retried but never past the 30-minute expiry (Tasks 8, 9, 12).
5. **Input edges.** Temperature 0, negative, below 30.00 °C or above 45.00 °C (and the °F equivalents ±1 hundredth), a decimal comma, three decimals; a name or dose text of 81 characters; every-N-hours 0 or 169; 13 fixed times, a duplicate time, `24:00`, `7:5`; `end_on` before the start; a dose more than 5 minutes in the future; a dose for another kid's course; a PATCH that only changes the note keeps every stored column (Tasks 2, 3, 5, 6).
6. **Due list caps.** Medication entries and vaccine groups share `HOME_DUE_MAX`; vaccine-only behaviour is byte-for-byte unchanged when a kid has no course; a medication entry never shows for a module that is off or an inactive course (Tasks 14, 21).

---

## Open values

Rule 1 (never assume an owner value). Rows marked **owner** are the owner's decisions from `docs/superpowers/HANDOFF.md` "Decisions a new session must not reopen" and are settled. Every other row says **(a)** a proposal the plan author can rule on, with its cost if wrong, or **(b)** it needs the owner (and is in the questions below).

### Questions for the owner

Prioritised; at most one sentence each.

1. **Edge order:** may a small edge PR that only renders `VAPID_PRIVATE_KEY` and `VAPID_SUBJECT` (= `PUBLIC_ORIGIN`, no new owner value) merge and deploy before the plan-6 image bump, leaving ntfy for the later edge plan? *Default: yes.* If not, the plan-6 bump cannot be deployed until the full edge plan (rows E1–E2) is answered, because the API refuses to start without VAPID.
2. **Deleting a course:** may deleting a medication course also delete its logged doses (the form says so), or must a course with doses only be ended? *Default: delete cascades, behind a confirm that names the dose count.* If the owner differs, `DELETE` answers 409 `has_doses` for a course with doses and the form offers "End course" instead (Task 5).
3. **Doses on the timeline:** should medication doses appear on the kid's timeline next to symptoms? *Default: no; symptoms are on the timeline, doses are listed on the medication screen.* If yes, one more UI task splits the timeline's kind from `TrackerKind` (doses have no log route) and the API adds a `medication_dose` timeline type.
4. **Overdue on the home:** how long does a missed dose stay "overdue" in the home Due list? *Default: 30 minutes after its due time, the same window as the push; then it disappears.* If longer (for example until the next occurrence), only the home read model's window changes (Task 14).
5. **Disabled parent:** when the admin disables a parent, should their push devices, ntfy topic and opt-ins be deleted (re-enabled parents opt in again) rather than kept and skipped? *Default: deleted, in the same transaction as the disable.* If kept, sends still skip them (send-time check) and re-enabling resumes their reminders without new consent.
6. **One browser, two parents:** when a second parent enables notifications in a browser another parent already registered, should the device move to the new parent (the old parent's opt-ins for it are deleted) rather than be refused? *Default: it moves.* If refused, the second parent sees "This browser gets another parent's reminders" and cannot enable it until the first removes it.
7. **Offline page:** may the service worker add a navigation-only `fetch` handler that serves one static offline page (spec 11 says "install and push only" and also "shows an offline page", which is impossible without it)? *Default: yes; it caches only `/offline.html`, never kid data.* If no, there is no offline page and the browser shows its own error.

**Edge, later** (for the separate `homelab-edge` plan; nothing here depends on them):

- E1. The ntfy topic prefix (`NTFY_TOPIC_PREFIX`, 1–32 of `a-z0-9-`), a compose network shared by ntfy and the API only (ntfy on the `happybabytracker` network would break `happybabytracker-config-test.py` and EDGE-11), a write-only ntfy user with ACL `<prefix>-*:wo` and a generated token, publishing over `http://ntfy:2586`.
- E2. Who may read topics: the edge's ntfy denies by default and only `david` reads; per-parent random topics cannot be named in static ACLs. Grant a read-only family user `<prefix>-*:ro`, or leave readers to the operator as spec 11.1 says. Also: `init_secrets.py` generates keys unconditionally while the design says "only when enabled".

### Table

| # | Value | Decision or proposal | Kind |
|---|---|---|---|
| 1 | Scope | The app only; the edge is its own plan and PR. ntfy ships off. Web Push reaches no real device until the edge renders `VAPID_*`, which must precede the plan-6 deploy (question 1) | owner (scope); order is question 1 |
| 2 | Categories | Reminders for medication only; the payload carries `category` so others can follow | owner |
| 3 | Every N hours | A fixed clock grid from the course's start instant; never recomputed from the last dose | owner |
| 4 | Notification text | `"<Category> due for <kid's name>"`, never the medication or dose | owner |
| 5 | VAPID | Always required at start; `VAPID_SUBJECT` must be an `https:` URL; on the edge, the app's own origin. Start refuses an invalid key, a non-`https:` subject, a subject with userinfo, query or fragment | owner |
| 6 | Web Push library | Own stdlib code in `api/internal/push` from the prototype; no new dependency. Cost if wrong: the owner would prefer `webpush-go` v1.4.0 + `golang-jwt/jwt/v5` (a pin and go.sum change) | (a) |
| 7 | ntfy | Optional; on only when `NTFY_URL` is set; one generated topic per parent (`<prefix>-<22 base62>`); nothing here depends on `homelab-edge`; `NTFY_URL` may be `http:` or `https:`, `NTFY_PUBLIC_URL` must be `https:`; any other `NTFY_*` without `NTFY_URL` is a start error (spec 13.1) | owner (on/off, per parent); http acceptance (a): cost if wrong, the edge must publish through Caddy |
| 8 | Deletes and switches | Hard deletes; turning the `sickness` or `medication` module off hides entry points and reminders and keeps data | owner |
| 9 | Symptoms | Spec 10.2's list: `congestion, cough, fever, vomiting, diarrhea, constipation, rash, ear_pain, sore_throat, poor_appetite, unusually_sleepy, other`; `other` requires `other_text` (1–80). Optional severity `mild\|moderate\|severe`. One symptom per entry, no episode object | spec (call), kept |
| 10 | Temperature | Only `fever` carries one, and it is required there; stored as hundredths of °C, hundredths of °F and the unit typed (`c`\|`f`, the profile's existing `temp` values); valid range 30.00–45.00 °C (checked on the °C column after conversion); `method` required with it: `oral\|rectal\|axillary\|ear\|forehead\|other`. The UI accepts up to two decimals in either unit (the draft allowed one for °F). No threshold, colour or advice anywhere. Cost if wrong: one constant or list | (a) |
| 11 | Course | `name` (1–80), `dose` text (1–80, "2.5 ml"), `start_at` (instant; the grid anchor for `every`, the first moment for `times`), optional `end_on` (kid-local date, inclusive to the end of that day), `active`, `note`, and `schedule`: `{"kind":"none"}` \| `{"kind":"times","times":["08:00","20:00"]}` (1–12 distinct `HH:MM`, sorted on save) \| `{"kind":"every","hours":N}` (1–168). At most 50 courses per kid. The draft's separate `start_on` date and `start_at` instant became one `start_at`. Cost if wrong: a migration column | (a) |
| 12 | Doses | `medication` (a course of the same kid), `given_at` (not more than `MaxFutureSkew` = 5 min ahead), optional `amount` text (1–80), `note`; loggable for any course, scheduled or not, active or not; `medication` cannot change on PATCH (422 `medication`); at most 5000 per kid (409 `limit`) | (a) |
| 13 | Deleting a course | Deletes its doses and reminders (cascade) behind a confirm | question 2 |
| 14 | Materialisation | For each active course with a `times`/`every` schedule: rows for every occurrence with `due_at` in [now − 30 min, now], plus the first occurrence after now that is not `done` (so "next due" and the home always have a row). Inserted with `ON CONFLICT DO NOTHING`. An occurrence first seen more than 30 minutes late is never created. Creating or editing a course (schedule, `start_at`, `end_on`, `active`) and changing the kid's time zone delete that course's (the kid's) `pending` rows with `due_at` > now and no delivery, then re-materialise, in the same transaction. Cost if wrong: store-only | (a) |
| 15 | Done | A dose marks done exactly one occurrence: the latest materialised occurrence of that medication with `due_at − 30 min ≤ given_at` and `given_at` before the following occurrence; state `pending` or `expired` becomes `done`, inside the dose's transaction. Editing or deleting a dose does not undo `done` (no history). A done reminder sends nothing further. The draft's "window until the next occurrence" let one dose mark two occurrences done; this picks one. Cost if wrong: store-only | (a) |
| 16 | Delivery | When a pending reminder falls due, one delivery row per allowed target (row 22 rules) is created and claimed (`sending`) before its request and settled after. At-most-once: at start, rows left `sending` become `unconfirmed` and are never resent (a crash can lose one reminder, never double it). 429/5xx/timeout retry 30 s, 2 min, 10 min after the previous attempt (or `Retry-After`, whichever is later), never past `due_at + 30 min` → `expired`; 404/410 → `gone` and that subscription is deleted; other 4xx → `failed`, no retry; ntfy 401/403 → `failed` and one log line per tick without the topic. A reminder whose 30 minutes pass is `expired` unless done. Per-target requests run concurrently (at most `DeliveryConcurrency` = 8) | spec 11 (call), detail (a) |
| 17 | TTL, urgency, topic | `TTL = min(3600, seconds until due_at + 30 min)`, at least 1; `Urgency: high`; `Topic` = the reminder id without dashes (32 hex) so a retry replaces a pending copy | (a) |
| 18 | Payload | `{"v":1,"category":"medication","kid":"<id>","title":"Medication due for Mia","path":"/kids/<id>/medication","tag":"reminder-<reminder id>"}`; the service worker uses `title` as is (the draft built it in the worker, which would duplicate the text in TypeScript) | (a) |
| 19 | Subscription trust | Endpoint: `https`, no userinfo, port 443 or none, host not an IP literal, every resolved address public (no loopback, private, link-local, ULA, unspecified, multicast, CGNAT `100.64.0.0/10`) at registration and again at dial time (`net.Dialer.Control`); no redirects followed; at most 2048 bytes; no allowlist of push hosts (brittle; Apple's own advice is only to allow `*.push.apple.com` outbound). `p256dh` must parse with `ecdh.P256().NewPublicKey` (65 bytes), `auth` 16 bytes; stored canonical unpadded base64url. `HBT_ALLOW_FAKE_PUSH=1` swaps the sender for an in-memory recorder and skips the DNS and dial checks (syntax and key checks stay); never set on the edge | (a) |
| 20 | Shared browser | The endpoint is unique; a second parent's registration moves it | question 6 |
| 21 | Devices | Up to 10 per parent (409 `limit`); a label of 1–40 characters (default from the UI: "This device"); the UI remembers this browser's device id in an HttpOnly cookie `hbt_push_device` (an id, not a secret) so no-JS forms can target it; on each visit with permission granted the page re-sends the browser's subscription (an upsert), which covers expiry without a `pushsubscriptionchange` handler. Cost if wrong: UI-only | (a) |
| 22 | Who may opt in | Any parent with a grant (guardian or caregiver) on that kid; revoking the grant (remove, leave) deletes that parent's opt-ins for the kid in the same transaction; deleting the kid cascades | spec 11 |
| 23 | Disabled parent | Devices, topic and opt-ins deleted in the disable transaction | question 5 |
| 24 | Service worker | `install` (precache only `/offline.html`), `activate` (drop older offline caches, claim), `push` (always shows a notification, falls back to "Reminder due" on a bad payload), `notificationclick` (exact same-origin path starting with one `/`; focus a window on that path or open it), and a navigation-only `fetch` handler serving `/offline.html` on failure | question 7 |
| 25 | Home Due | Medication entries `{kind:"medication", medication:{id,name}, due_at, state}` for each `pending` reminder of an active course with `due_at` from now − 30 min to the end of the kid-local day, `state` `"overdue"` when `due_at` ≤ now else `"due"`; none when the module is off. Medication entries come first (by `due_at`, then medication id), then vaccine groups as today. `HOME_DUE_MAX = 3` caps the merged list; hidden entries show as one line per kind ("N more on the Medication page" / "N more on the Vaccines page"). Vaccine entries keep exactly their four keys. The home shows the medication **name** (on screen only; never in a notification). Cost if wrong: read model and one component | (a); the 30-minute window is question 4 |
| 26 | Course read model | `last_dose` (latest by `given_at`) and `next_due` (the earliest `pending` row with `due_at` ≥ now − 30 min, else `null`; `null` for `none`, inactive and ended courses) | (a) |
| 27 | Timeline | `symptom` joins `store.TimelineTypes` (module `sickness`); doses do not | question 3 |
| 28 | Module band rule | No change: `sickness` and `medication` are on by default in every band (`store.DefaultEnabled`), so the band-crossing lookback never applies to them and `moduleTables` gains nothing. The draft's "counts a symptom like any tracker entry" is dropped | fact |
| 29 | Notification screens | `/account/notifications`: this device (enable or disable Web Push, a label, which of the parent's kids), the parent's other devices (remove), the iOS add-to-home-screen sentence on iOS Safari outside standalone mode, and, only when ntfy is on, the topic, the server address, the 72-hour cache sentence, regenerate and off, with the same kid choices | spec 11, 11.1 |
| 30 | Health | `GET /healthz` (Open) is 200 while the store answers and the worker ticked within `WorkerStaleAfter` (60 s). `GET /healthz/reminders` (Open, like `/healthz`; the draft said internal-token) is 503 when a delivery is still `pending`/`sending` more than 15 minutes after its reminder's due time, or settled `failed`/`expired` within the last 15 minutes; 200 otherwise. The UI passes it through at `/healthz/reminders` (`isHealth` already admits the path) | spec 13.1 (call), detail (a) |
| 31 | Test-only route | `GET /v1/_test/pushes` (class `Internal`) exists in `Routes()` only when `HBT_ALLOW_FAKE_PUSH=1`; it returns the recorder's sends `{device, title, path, tag, ttl_seconds, urgency, topic}` in order (never an endpoint or key). The matrix kit runs with the fake on, so it is pinned and exercised | (a) |
| 32 | Limits | `MaxMedicationsPerKid` 50, `MaxMedicationDosesPerKid` 5000, `MaxScheduleTimes` 12, `MinEveryHours` 1, `MaxEveryHours` 168, `MedicationTextMaxLen` 80 (name, dose, amount), `SymptomOtherMaxLen` 80, `MinTempCHundredths` 3000, `MaxTempCHundredths` 4500, `PushDevicesPerParent` 10, `PushLabelMaxLen` 40, `PushEndpointMaxLen` 2048, `ReminderTick` 30 s, `ReminderExpiry` 30 min, `ReminderRetryDelays` 30 s / 2 min / 10 min, `ReminderHealthStale` 15 min, `WorkerStaleAfter` 60 s, `PushTTLMax` 3600 s, `PushTimeout` 10 s, `PushJWTLifetime` 12 h, `PushResponseMaxBytes` 4096, `DeliveryConcurrency` 8, `NtfyTimeout` 10 s, `NtfyTopicRandomLen` 22, `NtfyPrefixPattern` `^[a-z0-9-]{1,32}$` | (a) |

---

## Task 1: Migration 0006, models and constants

**Files:** Create `api/internal/store/migrations/0006_sickness_medication.sql`; modify `api/internal/model/tracker.go`, `api/internal/constants/constants.go`, `api/internal/store/store_test.go`.

**Interfaces — Produces:** tables (all `STRICT`, ids TEXT UUIDv7, instants INTEGER):
- `symptoms` (common entry columns as `diapers`: `id, kid_id → kids ON DELETE CASCADE, note, created_by, updated_by, created_at, updated_at`) + `at`, `symptom CHECK IN (…row 9)`, `severity CHECK IN ('mild','moderate','severe') NULL`, `other_text NULL`, `temp_c_h, temp_f_h, temp_typed CHECK IN ('c','f')` (all NULL or all set), `method CHECK IN (…row 10) NULL`; `CHECK` that `fever` ⇔ temperature and method set, `other` ⇔ `other_text` set; index `(kid_id, at)`.
- `medications` (common columns) + `name, dose, schedule_kind CHECK IN ('none','times','every'), schedule_times TEXT NULL` (comma-joined sorted `HH:MM`), `every_hours INTEGER NULL CHECK (1..168)`, `start_at, end_on TEXT NULL, active INTEGER CHECK IN (0,1)`; CHECKs tie kind to its column; index `(kid_id, start_at)`.
- `medication_doses` (common columns) + `medication_id → medications ON DELETE CASCADE, given_at, amount NULL`; index `(kid_id, given_at)`, `(medication_id, given_at)`.
- `reminders(id, kid_id → kids CASCADE, medication_id → medications CASCADE, due_at, state CHECK IN ('pending','done','expired','cancelled'), created_at, UNIQUE(medication_id, due_at))`, index `(state, due_at)`.
- `reminder_deliveries(id, reminder_id → reminders CASCADE, channel CHECK IN ('push','ntfy'), target_id, parent_id → parents CASCADE, state CHECK IN ('pending','sending','sent','gone','failed','expired','unconfirmed','cancelled'), attempts, next_attempt_at, settled_at NULL, UNIQUE(reminder_id, channel, target_id))`.
- `push_subscriptions(id, parent_id → parents CASCADE, endpoint UNIQUE, p256dh, auth, label, created_at, updated_at)`; `push_kid_optins(subscription_id → push_subscriptions CASCADE, kid_id → kids CASCADE, PRIMARY KEY (subscription_id, kid_id))`.
- `ntfy_topics(parent_id PRIMARY KEY → parents CASCADE, topic UNIQUE, created_at)`; `ntfy_kid_optins(parent_id → ntfy_topics(parent_id) CASCADE, kid_id → kids CASCADE, PRIMARY KEY (parent_id, kid_id))`.

Model structs `Symptom`, `Temperature` (in `units`, Task 2: model holds `*units.Temperature`), `Medication`, `Schedule`, `MedicationDose`, `Reminder`; every constant of Open value 32; `model.Symptoms`, `model.Severities`, `model.TempMethods` name lists.

- [ ] **Step 1: Failing tests** in `store_test.go`: `wantTables` gains the nine tables; the five `"5"` assertions (`TestOpenAppliesMigrations`, `TestOpenIsIdempotent`, and the three upgrade tests) become `"6"`; `TestMigration0006KeepsRows` (a database at 0005, built like `TestMigration0005KeepsRows`, with a row in every older table: all survive); `TestReminderChecks` table (one unique id per case: bad symptom, half-set temperature, fever without temperature, `other` without text, bad schedule kind, `times` kind with `every_hours`, `every_hours` 0 and 169, bad reminder state, duplicate `(medication_id, due_at)`, bad delivery channel/state, duplicate push endpoint; each refused, a valid row of each accepted); `TestCascades` (deleting a kid empties every new kid-owned table and its opt-ins; deleting a medication removes its doses, reminders and deliveries; deleting a parent removes its subscriptions, topic, opt-ins and deliveries).
- [ ] **Step 2:** `cd api && go test ./internal/store/ -race` → FAIL. **Step 3: Implement** (check each `CHECK` against SQLite's STRICT rules as plan 5.1 did). **Step 4:** PASS; `gofmt -l .`, `go vet ./...`, staticcheck as in CLAUDE.md. **Step 5: Commit** `feat(api): sickness, medication and reminder tables`.

## Task 2: Temperature units

**Files:** Create `api/internal/units/temperature.go`, `api/internal/units/temperature_test.go`.

**Interfaces — Produces:**
```go
const (UnitC Unit = "c"; UnitF Unit = "f")
type Temperature struct{ CHundredths, FHundredths int64; Typed Unit } // Typed is exact, the other derived
func NewTemperature(unit Unit, amount int64) (Temperature, error) // hundredths of the typed unit; ErrUnit, ErrRange
```
`f_h = roundHalfUp(c_h × 9, 5) + 3200`; `c_h = roundHalfUp((f_h − 3200) × 5, 9)`; range checked on `CHundredths` against `MinTempCHundredths`..`MaxTempCHundredths` after conversion; integers only (`roundHalfUp` of `volume.go`, valid for the non-negative numerators the range allows).

- [ ] **Step 1: Failing tests:** 3660 c → 9788 f; 10040 f → 3800 c exact typed; 3000 and 4500 c accepted, 2999 and 4501 refused; 8600 f (30.00 °C) and 11300 f (45.00 °C) accepted, 8599 and 11301 refused; 0, negative, `1<<40`; unknown unit; half-up ties table; round trip c → f → c within one hundredth.
- [ ] **Step 2:** `go test ./internal/units/ -race` → FAIL. **Step 3: Implement.** **Step 4:** PASS. **Step 5: Commit** `feat(api): temperatures in hundredths of both units`.

## Task 3: Symptoms (store, HTTP, timeline)

**Files:** Create `api/internal/store/symptoms.go`, `api/internal/store/symptoms_test.go`, `api/internal/httpapi/symptoms.go`, `api/internal/httpapi/symptoms_test.go`; modify `api/internal/httpapi/routes.go`, `api/internal/httpapi/matrix_test.go` (`routeClasses`, the `entry` filler, `newMatrixFixture`, `fixture`), `api/internal/store/timeline.go` (`TimelineTypes`, `timelineOf`), `api/internal/httpapi/timeline.go` (`timelineModules`, `newTimelineView`), `api/internal/httpapi/timeline_test.go`.

**Interfaces — Consumes:** Task 1, Task 2. **Produces:** store `CreateSymptom`, `Symptom`, `ListSymptoms`, `UpdateSymptom`, `DeleteSymptom` with the signatures of the diaper functions; routes (class `Caregiver`) `GET|POST /v1/kids/{kid}/symptoms`, `GET|PATCH|DELETE /v1/kids/{kid}/symptoms/{entry}`; view `{id, at, symptom, severity|null, other_text|null, temperature:{c_hundredths, f_hundredths, typed_unit}|null, method|null, note}`; body `temperature: {"unit":"c"|"f","amount":n}` (hundredths) or `null`; validation order `at`, `symptom`, `severity`, `other_text`, `temperature`, `method`, `note`; changing `symptom` away from `fever` on PATCH requires `temperature: null` and `method: null` in the same body (else 422 `temperature`), to `fever` requires both. Timeline type `symptom` (module `sickness`).

- [ ] **Step 1: Failing tests:** store round trip, `TestSymptomBelongsToItsKid`, `TestSymptomWriteByDisabledActor`, list with `from`/`to` and cursor; HTTP CRUD; fever requires temperature and method, other symptoms refuse both (422 `temperature` / `method`); `other` requires `other_text` ≤ 80; every Review Focus 5 temperature edge (422 `temperature`); PATCH merge (absent keeps, `null` clears severity, note, other_text where allowed; an unchanged typed temperature keeps all three columns and is not range-checked again); timeline `types=symptom` and default types follow the `sickness` module; roles and identical 404 bodies (no grant, unknown kid, another kid's entry), admin 403; `TestNoValuesInLogs` style test with `wantCleanLog` (a distinctive note, `other_text` and temperature); the matrix covers the five routes.
- [ ] **Step 2:** `cd api && go test ./internal/store/ ./internal/httpapi/ -race` → FAIL. **Step 3: Implement** by the pattern of `diapers` and `growth` (`mergeWeight`'s unchanged-first comparison). **Step 4:** full API pipeline PASS. **Step 5: Commit** `feat(api): symptoms with temperature`.

## Task 4: Schedule engine

**Files:** Create `api/internal/schedule/schedule.go`, `api/internal/schedule/schedule_test.go`.

**Interfaces — Produces** (pure; imports `time` and `constants` only, so both the store and the worker can import it):
```go
type Kind string // KindNone "none", KindTimes "times", KindEvery "every"
type Schedule struct{ Kind Kind; Times []string /* "HH:MM", sorted, distinct */; EveryHours int }
type Course struct{ Schedule Schedule; StartAt time.Time; EndOn string /* YYYY-MM-DD or "" */; Active bool }
func Valid(s Schedule) error                                               // names the bad part; never echoes a value
func Occurrences(c Course, loc *time.Location, from, to time.Time) []time.Time // ascending, from ≤ t ≤ to
func Next(c Course, loc *time.Location, after time.Time) (time.Time, bool)    // first t > after
```
`every`: `StartAt + k × hours` for k ≥ 0 (absolute durations, unaffected by DST). `times`: each kid-local date from `StartAt`'s date, each `HH:MM` resolved by the gap/overlap rule of Global Constraints, kept when ≥ `StartAt`. `EndOn`: occurrences strictly before the start of the kid-local day after `EndOn` (found without `time.Date` at midnight: step from noon, then resolve). An inactive course or `KindNone` yields none.

- [ ] **Step 1: Failing tests:** `every` 6 h from 08:00 local is 08:00, 14:00, 20:00, 02:00 and stays on the same absolute grid across a DST change; `times` 02:30 on the `America/New_York` spring-forward day → 03:00 EDT; 01:30 on the fall-back day once, at the EDT instant; `America/Havana` and `America/Santiago` midnight-DST days with a `00:00` time and `EndOn` on that day; a course starting at 13:00 skips that day's 08:00; `EndOn` inclusive to the end of the kid-local day, `EndOn` before `StartAt` yields none; `from`/`to` inclusive; `Next` across a day boundary and past `EndOn` (false); `Valid` refuses empty times, 13 times, a duplicate, `24:00`, `7:5`, `EveryHours` 0 and 169, `times` with hours; 29 Feb 2028 and 31 Dec → 1 Jan.
- [ ] **Step 2:** `go test ./internal/schedule/ -race` → FAIL. **Step 3: Implement.** **Step 4:** PASS. **Step 5: Commit** `feat(api): medication schedule engine`.

## Task 5: Medication courses

**Files:** Create `api/internal/store/medications.go`, `api/internal/store/medications_test.go`, `api/internal/httpapi/medications.go`, `api/internal/httpapi/medications_test.go`; modify `api/internal/store/kids.go` (`UpdateKid`: a time-zone change re-materialises), `api/internal/httpapi/routes.go`, `api/internal/httpapi/matrix_test.go`.

**Interfaces — Consumes:** Tasks 1, 4. **Produces:** store `CreateMedication`, `Medication`, `ListMedications`, `UpdateMedication`, `DeleteMedication`, and the shared helper `rematerialise(ctx, tx, kidID, medicationID string, loc, now)` (delete the course's `pending` rows with `due_at` > now and no delivery, then insert Open value 14's rows); `ListMedications` and `Medication` read course, `last_dose` and `next_due` in one `inReadTx`. Routes (class `Caregiver`): `GET|POST /v1/kids/{kid}/medications`, `GET|PATCH|DELETE /v1/kids/{kid}/medications/{entry}`. View `{id, name, dose, schedule:{kind, times?, hours?}, start_at, end_on|null, active, note, last_dose:{id, given_at, amount|null}|null, next_due:{due_at}|null, dose_count}`. Validation order `name`, `dose`, `schedule`, `start_at`, `end_on`, `active`, `note`; cap 50 → 409 `limit`. Delete per Open value 13 (question 2; the default cascades).

- [ ] **Step 1: Failing tests:** CRUD and every Review Focus 5 course edge; `TestCreateMaterialisesNext` (a `times` course created at 10:00 with 08:00 and 20:00 has `next_due` 20:00 at once; an `every` course whose start is 20 minutes ago has a row for it and one for the next); `TestEditReplacesPendingRows` (changing times, `start_at`, `end_on` or `active` replaces future pending rows; a row already `done`, or one with a delivery, stays); `TestInactiveHasNoNextDue`; `TestKidZoneChangeRematerialises` (a 08:00 course in `Europe/London` moves to the `America/New_York` 08:00 instant); `TestCourseReadIsOneSnapshot` (course, last dose and next due agree under a concurrent write; use the plan-5.1 pattern); delete removes doses and reminders; roles, 404 identity, admin 403; `wantCleanLog` with a distinctive name and dose text; matrix rows.
- [ ] **Step 2:** `go test ./internal/store/ ./internal/httpapi/ -race` → FAIL. **Step 3: Implement.** **Step 4:** full API pipeline PASS. **Step 5: Commit** `feat(api): medication courses with their next due`.

## Task 6: Doses

**Files:** Create `api/internal/store/doses.go`, `api/internal/store/doses_test.go`, `api/internal/httpapi/doses.go`, `api/internal/httpapi/doses_test.go`; modify `api/internal/httpapi/routes.go`, `api/internal/httpapi/matrix_test.go` (the `entry` filler must tell `/medications/` from `/medication-doses/`).

**Interfaces — Consumes:** Tasks 1, 4, 5. **Produces:** store `CreateDose`, `Dose`, `ListDoses(ctx, kidID, medicationID string /* "" = all */, q ListQuery)`, `UpdateDose`, `DeleteDose`; `markDone(ctx, tx, …)` per Open value 15 inside `CreateDose` and `UpdateDose` (when `given_at` changes). Routes (class `Caregiver`): `GET|POST /v1/kids/{kid}/medication-doses` (optional `medication`, `from`, `to`, cursor), `GET|PATCH|DELETE /v1/kids/{kid}/medication-doses/{entry}`. View `{id, medication, given_at, amount|null, note}`. A `medication` that is not this kid's → 422 `medication` (a malformed id too); cap 5000 → 409 `limit`.

- [ ] **Step 1: Failing tests:** CRUD; `given_at` 5 min + 1 s ahead → 422 `given_at`; another kid's course → 422 `medication`; PATCH `medication` → 422 `medication`; `TestDoseMarksOneReminderDone` (hand-inserted reminders at 08:00 and 14:00: a dose at 07:31 and one at 08:01 mark 08:00 done; 07:29 marks nothing; a dose at 13:40 marks only 14:00; another medication's dose marks nothing; an `expired` row becomes `done`); `TestDeleteDoseKeepsDone`; inactive course still accepts doses; list filter by medication; roles, 404 identity, `wantCleanLog` with a distinctive amount and note; matrix rows.
- [ ] **Step 2:** run → FAIL. **Step 3: Implement.** **Step 4:** full API pipeline PASS. **Step 5: Commit** `feat(api): medication doses mark their reminder done`.

## Task 7: Push crypto and the VAPID start checks

**Files:** Create `api/internal/push/encrypt.go`, `api/internal/push/vapid.go` (+ `_test.go` each, from `docs/superpowers/research/reminders/pushproto/`); modify `api/internal/config/config.go`, `api/internal/config/config_test.go`, `api/cmd/happybabytracker-api/main_test.go`, `api/internal/httpapi/testkit_test.go` (the kit's `Cfg`).

**Interfaces — Produces:**
```go
func ParsePrivateKey(b64 string) (*ecdsa.PrivateKey, error) // exactly 43 unpadded base64url chars, a valid P-256 scalar
func PublicKey(k *ecdsa.PrivateKey) string                  // 65-byte uncompressed point, unpadded base64url
func Encrypt(plaintext, uaPub, auth []byte, asPriv *ecdh.PrivateKey, salt []byte) ([]byte, error) // ≤ 3993 bytes plaintext
func ParseKeys(p256dh, auth string) (uaPub, authSecret []byte, err error) // tolerant decode, strict lengths
type Signer struct{ … }
func NewSigner(k *ecdsa.PrivateKey, subject string) (*Signer, error)
func (s *Signer) Authorization(endpoint string, now time.Time) (string, error) // "vapid t=…, k=…"
```
`config.Config` gains `VAPIDKey *ecdsa.PrivateKey`, `VAPIDSubject string`, `AllowFakePush bool`; `config.Load` refuses (naming the variable, never echoing it) a missing or malformed `VAPID_PRIVATE_KEY`, a `VAPID_SUBJECT` that is not `https:` with a host and without userinfo, query or fragment, and `HBT_ALLOW_FAKE_PUSH` set to anything but `1` or empty.

- [ ] **Step 1: Failing tests:** the RFC 8291 Appendix A vector; an independent decrypt round trip; 3994-byte plaintext refused; a compressed or invalid `p256dh` and a 15/17-byte `auth` refused; the signature is 64 bytes and verifies; `aud` is the endpoint origin (with a port when present), `exp` = now + `PushJWTLifetime` ≤ 24 h, `sub` the subject; `ParsePrivateKey` refuses padded, standard-alphabet, 42 and 44 characters, zero and ≥ n; `PublicKey` derives the prototype's key; config table cases for each refusal and the accepted form; no key in any error string; existing config, main and httpapi tests pass with the new required variables (a fixed test key in a test helper, generated for tests, never the edge's).
- [ ] **Step 2:** `go test ./internal/push/ ./internal/config/ ./cmd/... -race` → FAIL. **Step 3: Implement.** **Step 4:** full API pipeline PASS. **Step 5: Commit** `feat(api): Web Push encryption and VAPID, required at start`.

## Task 8: Push sender, endpoint validation and the recorder

**Files:** Create `api/internal/push/validate.go`, `api/internal/push/send.go`, `api/internal/push/recorder.go` (+ `_test.go` each).

**Interfaces — Consumes:** Task 7. **Produces:**
```go
type Resolver interface{ LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error) }
func ValidateEndpoint(ctx context.Context, r Resolver, endpoint string) error // Open value 19; never echoes the endpoint
type Subscription struct{ Endpoint, P256dh, Auth string }
type Message struct{ Payload []byte; TTL time.Duration; Topic string }
type Outcome int // Sent, Gone, Retry, Failed
type Result struct{ Outcome Outcome; Status int; RetryAfter time.Duration }
type Sender interface{ Send(ctx context.Context, s Subscription, m Message) Result }
func NewSender(sig *Signer, now func() time.Time) Sender // PushTimeout, no redirects, dial-time address check, reads ≤ PushResponseMaxBytes
type Recorder struct{ … }                                 // Sender; encrypts (so keys must be valid) and records plaintext and headers in order
func (r *Recorder) Sends() []RecordedSend
```

- [ ] **Step 1: Failing tests:** `ValidateEndpoint` refuses `http:`, userinfo, port 8443, `https://127.0.0.1/…`, `[::1]`, `localhost`, `10.0.0.1`, `169.254.169.254`, `fc00::1`, `100.64.0.1`, a 2049-byte URL, a name resolving (fake resolver) to any private address among public ones; accepts a public name. Sender against `httptest.NewTLSServer` with an injected transport and a test-only dial check bypass: headers (`TTL`, `Urgency: high`, `Topic`, `Content-Encoding`, `Authorization` parses and verifies), the body decrypts; 201 → `Sent`; 404/410 → `Gone`; 429 with `Retry-After: 120` → `Retry` and 2 min; 500 → `Retry`; a response slower than `PushTimeout` (shortened through an option in the test) → `Retry`; 302 → `Failed`, not followed; 400/401/403/413 → `Failed`; the dial check refuses a loopback address reached through DNS; no endpoint, key or payload in any error or `Result`; `Recorder` keeps sends in order and refuses an invalid key like the real path.
- [ ] **Step 2:** `go test ./internal/push/ -race` → FAIL. **Step 3: Implement.** **Step 4:** PASS. **Step 5: Commit** `feat(api): push sender with endpoint checks`.

## Task 9: Push devices and per-kid opt-ins

**Files:** Create `api/internal/store/push.go`, `api/internal/store/push_test.go`, `api/internal/httpapi/push.go`, `api/internal/httpapi/push_test.go`; modify `api/internal/store/access.go` (`RemoveAccess` deletes the target parent's opt-ins for the kid), `api/internal/store/parents.go` (`SetParentDisabled(…, true, …)` deletes devices, topic and opt-ins: question 5 default), `api/internal/httpapi/routes.go` (`Deps` gains `Push push.Sender`, `Resolver push.Resolver`, `VAPIDPublicKey string`), `api/internal/httpapi/testkit_test.go`, `api/internal/httpapi/matrix_test.go` (new param `device`, a device per actor in the fixture).

**Interfaces — Consumes:** Tasks 1, 7, 8. **Produces:** routes — class `Parent`: `GET /v1/push/public-key` → `{"public_key"}`; `GET /v1/me/push-devices` → `{"devices":[{id, label, created_at, kids:[kidId…]}]}` (kids filtered to current grants); `PUT /v1/me/push-devices` body `{endpoint, keys:{p256dh, auth}, label}` → 201 `{"device":{…}}` new, 200 existing (Open value 20 re-binding, question 6 default); `PATCH /v1/me/push-devices/{device}` `{label}`; `DELETE /v1/me/push-devices/{device}`; class `Caregiver`: `PUT|DELETE /v1/kids/{kid}/push-devices/{device}` (opt in or out; a device that is not the caller's → 404 identical). (The draft's `/v1/me/push-subscriptions/{id}/kids/{kid}` under class `Parent` could not answer the grant 404 through the middleware, so the opt-in moved under the kid.) Store: `UpsertDevice`, `RenameDevice`, `DeleteDevice`, `SetPushOptin`, `ClearPushOptin`, `Devices(parentID)`, `PushTargets(ctx, q querier, kidID) []PushTarget` (opted-in devices of enabled parents with a grant).

- [ ] **Step 1: Failing tests:** upsert creates (201), re-sends update the label and keys (200), another parent's re-registration moves the device and deletes the old parent's opt-ins for it; cap 10 → 409 `limit`; hostile endpoint → 422 `endpoint` (fake resolver; one case per Review Focus 4 item), bad keys → 422 `keys`, label 41 characters → 422 `label`; the endpoint and keys never appear in any response; opt-in for a kid without a grant → 404 identical to an unknown kid; `TestRevokeRemovesOptins` and `TestLeaveRemovesOptins` (store and HTTP); `TestDisableRemovesDevices`; `PushTargets` excludes revoked, disabled and not-opted-in; admin 403; another parent's device id → 404 identical; `wantCleanLog` with the endpoint and keys; matrix rows.
- [ ] **Step 2:** `go test ./internal/store/ ./internal/httpapi/ -race` → FAIL. **Step 3: Implement.** **Step 4:** full API pipeline PASS. **Step 5: Commit** `feat(api): push devices and per-kid opt-ins`.

## Task 10: The optional ntfy channel

**Files:** Create `api/internal/ntfy/config.go`, `api/internal/ntfy/publish.go` (+ `_test.go` each), `api/internal/store/ntfy.go`, `api/internal/store/ntfy_test.go`, `api/internal/httpapi/ntfy.go`, `api/internal/httpapi/ntfy_test.go`; modify `api/internal/config/config.go` (+ test: loads `ntfy.LoadConfig`), `api/internal/store/access.go`, `api/internal/store/parents.go` (the Task 9 hooks also cover ntfy rows), `api/internal/httpapi/routes.go`, `api/internal/httpapi/testkit_test.go` (the kit runs with ntfy on and a recording publisher), `api/internal/httpapi/matrix_test.go`.

**Interfaces — Produces:** `ntfy.Config{URL, PublicURL, TopicPrefix, Token string}`; `ntfy.LoadConfig(getenv) (cfg Config, enabled bool, err error)` (Open value 7; the error names the variable, never its value); `type Publisher interface{ Publish(ctx, topic, title, message, click string) Result }` with `Result{Outcome; Status; RetryAfter}` sharing push's `Outcome` meanings (401/403 → `Failed` with `ConfigFault: true`); `ntfy.NewPublisher(cfg)` (JSON to the server root, bearer token only when set, `NtfyTimeout`, no redirects). Routes (class `Parent`): `GET /v1/me/ntfy` → `{"enabled":false}` or `{"enabled":true,"server":"<NTFY_PUBLIC_URL or NTFY_URL>","topic":"…"|null,"kids":[…]}`; `PUT /v1/me/ntfy` (turn on, generates the topic; 200 when already on), `POST /v1/me/ntfy/regenerate`, `DELETE /v1/me/ntfy` (deletes topic and opt-ins); class `Caregiver`: `PUT|DELETE /v1/kids/{kid}/ntfy`. Every write answers 404 `not_found` while ntfy is off. Store: `NtfyTopic`, `SetNtfyTopic`, `DeleteNtfyTopic`, `SetNtfyOptin`, `ClearNtfyOptin`, `NtfyTargets(ctx, q, kidID)`.

- [ ] **Step 1: Failing tests:** config table (each refusal of spec 13.1 alone; `http:` and `https:` accepted for `NTFY_URL`, `http:` refused for `NTFY_PUBLIC_URL`; a prefix of 33 or with `_`); topic shape `<prefix>-` + 22 base62 from `crypto/rand`, 1000 draws distinct; regenerate replaces the topic and keeps opt-ins; off deletes both; opt-in needs a grant (404 identical); revoke, leave and disable remove opt-ins and the topic; publisher against `httptest` (JSON fields, `Authorization` only with a token, `click`, 401/403 → config fault, 429/5xx → retry, redirect not followed, the topic and token in no error); `GET` while off is `{"enabled":false}`; writes while off are 404; matrix rows.
- [ ] **Step 2:** `go test ./internal/ntfy/ ./internal/config/ ./internal/store/ ./internal/httpapi/ -race` → FAIL. **Step 3: Implement.** **Step 4:** full API pipeline PASS. **Step 5: Commit** `feat(api): optional ntfy channel`.

## Task 11: Reminder rows and deliveries (store)

**Files:** Create `api/internal/store/reminders.go`, `api/internal/store/reminders_test.go`.

**Interfaces — Consumes:** Tasks 1, 4, 5, 9, 10. **Produces** (every write in `inTx`, every multi-row read in `inReadTx`):
```go
func (s *Store) Materialise(ctx context.Context, now time.Time) error                  // Open value 14 for every kid with the module on
func (s *Store) OpenDue(ctx context.Context, now time.Time, ntfyOn bool) (int, error)   // pending reminders now due: gate (row 22, module, course) → deliveries or `cancelled`
func (s *Store) ClaimDeliveries(ctx context.Context, now time.Time, limit int) ([]Delivery, error) // pending with next_attempt_at ≤ now → sending, attempts+1
func (s *Store) SettleDelivery(ctx context.Context, id string, o Settle, now time.Time) error      // sent | gone (deletes the subscription) | failed | retry(next) | cancelled
func (s *Store) Expire(ctx context.Context, now time.Time) error                       // deliveries and reminders past due_at + ReminderExpiry
func (s *Store) MarkUnconfirmed(ctx context.Context) error                             // at start: sending → unconfirmed
func (s *Store) DeliveryTarget(ctx context.Context, d Delivery, now time.Time) (Target, bool, error) // re-checks access at send time
func (s *Store) RemindersHealthy(ctx context.Context, now time.Time) (bool, error)     // Open value 30
```
`Delivery` carries ids, channel, attempts and due time; `Target` carries what the sender needs (subscription or topic, the kid's name for the title, kid id) and is never logged.

- [ ] **Step 1: Failing tests (real SQLite, fixed clock):** `TestMaterialiseIsIdempotent` (two calls and a reopened store make one row per occurrence); a 31-minute-old occurrence is never created; module off → no rows; `TestOpenDueGates` (revoked, disabled, not opted in, module off, inactive course, kid deleted: no delivery or `cancelled`); `ClaimDeliveries` never returns a row twice across two concurrent callers; the retry schedule (30 s, 2 min, 10 min, `Retry-After` later wins) and nothing past `due_at + 30 min`; `gone` deletes exactly that subscription; `MarkUnconfirmed`; `DeliveryTarget` refuses after a revoke between claim and send; health true/false cases at 14:59 and 15:01 minutes.
- [ ] **Step 2:** `go test ./internal/store/ -race` → FAIL. **Step 3: Implement.** **Step 4:** PASS. **Step 5: Commit** `feat(api): reminder rows and deliveries`.

## Task 12: The reminder worker and its wiring

**Files:** Create `api/internal/reminders/worker.go`, `api/internal/reminders/worker_test.go`; modify `api/cmd/happybabytracker-api/main.go` (+ `main_test.go`), `api/internal/httpapi/routes.go` (the test-only route), `api/internal/httpapi/testpush.go` (new) and its test, `api/internal/httpapi/matrix_test.go`.

**Interfaces — Consumes:** Tasks 8, 10, 11. **Produces:** `reminders.New(Deps{Store *store.Store; Push push.Sender; Ntfy ntfy.Publisher /* nil when off */; Origin string; Now func() time.Time; Log *slog.Logger}) *Worker`; `(*Worker).Run(ctx)` ticks every `ReminderTick` until cancelled; `(*Worker).Tick(ctx) error` is one pass: `Materialise`, `OpenDue`, `ClaimDeliveries`, send each (bounded by `DeliveryConcurrency`, each with its own timeout), `SettleDelivery`, `Expire`; `(*Worker).LastTick() time.Time`. `main.go` calls `MarkUnconfirmed` before the first tick, runs the worker beside the server and stops it on shutdown; with `HBT_ALLOW_FAKE_PUSH=1` it uses `push.Recorder` and adds `GET /v1/_test/pushes` (class `Internal`, Open value 31).

- [ ] **Step 1: Failing tests (fake clock, fake sender and publisher, real SQLite):** one opted-in device and one opted-in topic receive one send each; the payload decrypts to exactly the six keys of Open value 18 with title `Medication due for <name>` and no medication name or dose anywhere in it; TTL = min(3600, seconds to expiry), `Urgency: high`, `Topic` the reminder id; `TestEachTargetOnItsOwn` (a target that hangs until its timeout does not delay another; a failing ntfy does not block push); `TestNoDuplicateAfterRestartMidSend` (a row left `sending` is never sent again; a second `Worker` on the same store sends nothing twice); `TestDoseBeforePushCancels`; a ntfy 401 logs one line per tick without the topic and does not retry; `wantCleanLog` over every worker log line (kid name, medication name, dose, endpoint, topic, token banned); `Run` returns on cancel with `-race`; main starts with the flag and refuses a bad VAPID key; `/v1/_test/pushes` absent without the flag, internal-token only with it, and never shows an endpoint; matrix row.
- [ ] **Step 2:** `go test ./internal/reminders/ ./internal/httpapi/ ./cmd/... -race` → FAIL. **Step 3: Implement.** **Step 4:** full API pipeline PASS. **Step 5: Commit** `feat(api): the reminder worker`.

## Task 13: Health

**Files:** Modify `api/internal/httpapi/health.go`, `api/internal/httpapi/health_test.go`, `api/internal/httpapi/routes.go` (`Deps` gains `LastTick func() time.Time`), `api/internal/httpapi/matrix_test.go`, `api/cmd/happybabytracker-api/main.go`.

**Interfaces — Produces:** `GET /healthz` 503 when the store does not answer or `LastTick` is older than `WorkerStaleAfter` (the first 60 s after start count from the process start); `GET /healthz/reminders` (class `Open`) 200 `{"status":"ok"}` or 503 `unavailable` per Open value 30; neither body names a kid, a count or a target.

- [ ] **Step 1: Failing tests:** both answers for each condition, with a fake clock and hand-made rows; matrix row for `/healthz/reminders`.
- [ ] **Step 2–4:** FAIL; implement; full API pipeline PASS. **Step 5: Commit** `feat(api): worker and reminder health`.

## Task 14: Home Due medication entries

**Files:** Modify `api/internal/httpapi/home.go`, `api/internal/httpapi/home_test.go`, `api/internal/store/reminders.go` (`DueToday(ctx, kidID, from, to)` in one `inReadTx`).

**Interfaces — Produces:** `homeView.Due` becomes `[]any` of `dueVaccineView` (today's `dueView`, four keys unchanged) and `dueMedicationView{Kind "medication"; Medication {id, name}; DueAt string; State "due"|"overdue"}`, medication entries first, per Open value 25; the `medication` module and the course's `active` flag gate them.

- [ ] **Step 1: Failing tests:** `sameKeys` for a medication entry (`kind`, `medication`, `due_at`, `state`) and its `medication` (`id`, `name`); the existing vaccine `sameKeys` test unchanged; `due` before its time, `overdue` after, gone when `done`, gone 30 minutes after due (question 4 default), gone with the module off (vaccine entries unaffected), another kid's course never appears, ordering by `due_at` then id, medication before vaccine.
- [ ] **Step 2–4:** FAIL; implement; full API pipeline PASS. **Step 5: Commit** `feat(api): medication reminders on the home Due list`.

## Task 15: UI plumbing — symptoms, courses and doses

**Files:** Modify `ui/src/lib/types.ts`, `ui/src/lib/constants.ts`, `ui/src/lib/server/constants.ts`, `ui/src/lib/server/api.ts`, `ui/src/lib/server/api.test.ts`, `ui/e2e/stub-api.ts`; create `ui/e2e/stub-medication.spec.ts`.

**Interfaces — Produces:** types `TrackerKind` gains `'symptoms'`, `TimelineType` gains `'symptom'`, `Symptom`, `Temperature` (`{cHundredths, fHundredths, typedUnit}`), `Medication`, `Schedule`, `MedicationDose`, `HomeDue` as a union on `kind` (`'vaccine' | 'medication'`); constants `TRACKER_KINDS`, `TRACKER_ROUTE` (`sickness: 'symptoms'`), `TIMELINE_TYPE`, `TRACKER_TITLE`, `TRACKER_LABEL`, `SYMPTOMS`, `SYMPTOM_LABEL`, `SEVERITIES`, `TEMP_METHODS` and their labels, the client-safe limits; the client (`locals.api.tracker('symptoms')` through the existing tracker table, plus `medications` and `doses` groups with the four-edit pattern; a malformed id answers not-found without a request); the stub mirrors Tasks 3, 5, 6 and 14 (validation order, field names, 404 rules, `next_due` from a stub clock, medication Due entries), with a self-test spec.

- [ ] **Step 1: Failing tests (RED):** `api.test.ts` (each new call, malformed bodies → `error`, the Due union parse); `stub-medication.spec.ts` self-tests; `ui/src/params/tracker.test.ts` accepts `symptoms`; `quick-log.test.ts` (the 24+ band leads with the Sickness log and the Medication page; both other bands list them after their first four).
- [ ] **Step 2:** `cd ui && pnpm test` → FAIL. **Step 3: Implement.** **Step 4:** `pnpm check && pnpm lint && pnpm test && pnpm test:e2e --trace off` PASS. **Step 5: Commit** `feat(ui): sickness and medication client, types and stub`.

## Task 16: UI plumbing — devices, ntfy and push helpers

**Files:** Modify `ui/src/lib/types.ts`, `ui/src/lib/server/api.ts`, `ui/src/lib/server/api.test.ts`, `ui/src/lib/server/constants.ts` (`PUSH_DEVICE_COOKIE`), `ui/e2e/stub-api.ts`; create `ui/src/lib/push.ts`, `ui/src/lib/push.test.ts`, `ui/src/lib/server/notification-form.ts`, `ui/src/lib/server/notification-form.test.ts`.

**Interfaces — Produces:** types `PushDevice`, `NtfyState`; client calls for every Task 9 and 10 route (the endpoint and keys only in the request body of the upsert, never logged or returned); `push.ts` (shared by page and service worker): `base64UrlToBytes`, `subscriptionBody(sub)`, `parsePayload(data): {title, path, tag} | null` (path must start with exactly one `/`), `isIosOutsideStandalone(ua, standalone)`; `readDeviceForm` (label 1–40, kid toggles; value-free messages); the stub mirrors both groups and the `/__requests` log covers them.

- [ ] **Step 1: Failing tests (RED):** each helper (padding, `//evil`, `https://x`, missing fields, iOS and iPadOS user agents, standalone true/false); form reader cases; `api.test.ts` for each call.
- [ ] **Step 2–4:** as Task 15. **Step 5: Commit** `feat(ui): notification client, push helpers and stub`.

## Task 17: UI — the sickness tracker

**Files:** Create `ui/src/lib/components/entry/SymptomFields.svelte`, `ui/src/lib/components/entry/TemperatureField.svelte`; modify `ui/src/lib/components/entry/TrackerFields.svelte`, `ui/src/lib/server/entry-form.ts` (+ test), `ui/src/lib/server/entry-failure.ts` (+ test), `ui/src/lib/entry-edit.ts` (+ test), `ui/src/lib/timeline.ts` (+ test), `ui/e2e/trackers.spec.ts`, `ui/e2e/timeline.spec.ts`.

**Interfaces:** the existing `log/[tracker=tracker]` and `[entry]` pages serve `symptoms` unchanged; `SymptomFields` has a symptom select, severity radios, the `other` text (shown by CSS `:has()` when `other` is chosen), and, for `fever`, `TemperatureField` (amount text and a unit select defaulting to `formUnits(session).temp`, hidden originals on edit) and a method select; `readEntryForm('symptoms', …)` parses up to two decimals with a point or comma, refuses hostile text (`1e3`, `0x10`, `--1`) with value-free messages, and sends only changed fields; `timelineLine` prints `"09:12 Fever 38.4 °C (ear)"` in the parent's unit from the API's columns, `"Cough, moderate"`, `"Other: <text>"`.

- [ ] **Step 1: Failing tests:** reader and formatter unit tests; Playwright (both projects, axe): log a fever with a temperature and method, read it back in °F for an imperial parent (exact for °F typed); a cough with severity; `other` requires its text; a range error keeps typed values with `aria-invalid` and `role="alert"`; edit only the note sends `{note}`; delete behind the confirm; caregiver can log; no-grant and unknown entry → the same 404 page; timeline filter `symptom`; the 24+ quick-log tile opens the form; 390 px no overflow with a long `other` text; no-JS create and edit. Every save uses `submit()`.
- [ ] **Step 2:** RED. **Step 3: Implement.** **Step 4:** full UI pipeline PASS. **Step 5: Commit** `feat(ui): the sickness tracker`.

## Task 18: UI — the medication screen

**Files:** Create `ui/src/routes/(app)/kids/[kid]/medication/{+page.server.ts,+page.svelte}`, `…/medication/new/{+page.server.ts,+page.svelte}`, `…/medication/[medication]/{+page.server.ts,+page.svelte}`, `…/medication/doses/[dose]/{+page.server.ts,+page.svelte}`, `ui/src/lib/components/medication/{CourseCard.svelte,ScheduleFields.svelte,DoseForm.svelte}`, `ui/src/lib/medication.ts` (+ test), `ui/src/lib/server/medication-form.ts` (+ test), `ui/e2e/medication.spec.ts`; modify `ui/src/lib/paths.ts` (`medicationPath`, `medicationNewPath`, `medicationEntryPath`, `doseEntryPath`), `ui/src/lib/quick-log.ts` (`PAGE_ROUTE.medication`), `ui/src/lib/components/KidLinks.svelte` (a Medication link for every role).

**Interfaces:** the page lists active courses (name, dose, `scheduleLabel` "Every 6 hours from 08:00" / "08:00, 20:00" / "As needed", "Last dose logged at 14:05" or "No dose logged yet", "Next due 20:00" when the API gives one, a one-tap "Log dose now" form with an optional amount that works without JavaScript), ended and inactive courses behind a `<details>`, and recent doses (edit links); create/edit (schedule kind radios with the matching fields shown by CSS `:has()`, up to 12 time inputs, an hours input, start date and time in the kid's zone, optional end date, active checkbox; delete behind `<details>` stating the dose count goes too: question 2 default). The page shows last and next only; it never computes or suggests an interval. Reachable with the module off (data stays).

- [ ] **Step 1: Failing tests:** `scheduleLabel`, `readMedicationForm`, `readDoseForm` units (every hostile input, edit sends only changes, clearing the end date sends `null`); Playwright (both projects, axe): create a fixed-times course and an every-N-hours course, log a dose with the one-tap form and see "Last dose logged at …", the next due line follows the stub clock, an inactive course shows no next due, edit only the note sends `{note}`, delete with the confirm, validation keeps values, caregiver, 404 identity, 390 px no overflow with a 200-character unbroken name, no-JS everything. Every save uses `submit()`.
- [ ] **Step 2–4:** RED; implement; full UI pipeline PASS. **Step 5: Commit** `feat(ui): the medication screen`.

## Task 19: UI — service worker and offline page

**Files:** Create `ui/src/service-worker.ts`, `ui/static/offline.html`, `ui/e2e/service-worker.spec.ts`; modify `ui/e2e/pwa.spec.ts`.

**Interfaces:** Open value 24 using `self` from `$app/service-worker` and `version` from `$app/env`, and `parsePayload` from `ui/src/lib/push.ts`; `notification` options `{body: '', tag, icon: '/icons/icon-192.png', data: {path}}`; `offline.html` is self-contained (inline CSS, no script, no external request).

- [ ] **Step 1: Failing tests:** no-cookie `GET /service-worker.js` → 200 JavaScript and `/offline.html` → 200 HTML without a redirect (`pwa.spec.ts` pattern); with the worker controlling the page, `context.setOffline(true)` and a navigation shows the offline page; a dispatched `PushEvent` with a valid payload shows a notification titled "Medication due for <name>" with the reminder's tag and path, a malformed one shows "Reminder due"; `caches.keys()` holds only the offline cache and it holds only `/offline.html`.
- [ ] **Step 2–4:** RED; implement; full UI pipeline PASS. **Step 5: Commit** `feat(ui): service worker and offline page`.

## Task 20: UI — the notifications page

**Files:** Create `ui/src/routes/(app)/account/notifications/{+page.server.ts,+page.svelte}`, `ui/src/routes/(app)/account/notifications/subscribe/+server.ts`, `ui/src/lib/components/NotificationDevice.svelte`, `ui/e2e/notifications.spec.ts`; modify `ui/src/routes/(app)/account/+page.svelte` (a link), `ui/src/lib/paths.ts`.

**Interfaces:** `load` fetches the public key, the devices, the parent's kids and the ntfy state; the enable button (needs JavaScript, says so) requests permission inside the click, subscribes with `applicationServerKey = base64UrlToBytes(key)`, and POSTs `subscriptionBody(sub)` to `subscribe/+server.ts`, which upserts and sets the `hbt_push_device` cookie; on load with permission granted the page re-sends the subscription; label, kid toggles, remove device, ntfy on/off/regenerate and ntfy kid toggles are form actions that work without JavaScript; the ntfy section (server, topic, the sentence "ntfy keeps each message for up to 72 hours on its server, so your child's name is stored there in plain text for that long") shows only when enabled; the iOS sentence ("On iPhone and iPad, add this app to your Home Screen first, then open it from there to turn on notifications") shows for iOS outside standalone mode.

- [ ] **Step 1: Failing tests (Playwright, Chromium, stubbed `PushManager` via `addInitScript`, `permissions: ['notifications']`):** enable posts the expected body to the stub (`/__requests`) and lists the device; kid toggles opt in and out; disable removes it; another parent's device id → 404 page; the iOS sentence only for an iOS user agent outside standalone; ntfy absent when off, present with topic and the 72-hour sentence when on, regenerate changes the topic, off removes it; no endpoint or key appears in any rendered page or page data; axe both widths; with JavaScript disabled the page renders and the non-push controls work. Every save uses `submit()`.
- [ ] **Step 2–4:** RED; implement; full UI pipeline PASS. **Step 5: Commit** `feat(ui): the notifications page`.

## Task 21: UI — home Due and the reminders health passthrough

**Files:** Create `ui/src/lib/components/DueList.svelte`, `ui/src/lib/due.ts` (+ test), `ui/src/routes/healthz/reminders/+server.ts`; modify `ui/src/routes/(app)/kids/[kid]/+page.svelte`, `ui/src/lib/server/api.ts` (`remindersHealth`), `ui/e2e/home.spec.ts`, `ui/e2e/pwa.spec.ts` or the existing health spec.

**Interfaces:** `dueLines(due, max)`: the first `HOME_DUE_MAX` entries in API order, then one "N more on the Medication page" and/or "N more on the Vaccines page" line per hidden kind; a medication entry reads `"<name> · 14:00"` (kid zone) plus `" · overdue"` and links to the medication page; vaccine entries keep `dueLabel` and their link. `GET /healthz/reminders` passes the API's status through like `/healthz/api` and never forwards a body.

- [ ] **Step 1: Failing tests:** `dueLines` units (0, 2 and 5 medication entries with 0 and 3 vaccine groups; vaccine-only output identical to today's); Playwright with the stub: the Due list shows a medication entry with its time and `overdue`, hides it with the module off, vaccine groups still render and the "more" lines link the right pages; the health route mirrors 200/503; axe both widths.
- [ ] **Step 2–4:** RED; implement; full UI pipeline PASS. **Step 5: Commit** `feat(ui): medication reminders on the home Due list`.

## Task 22: Smoke, docs and HANDOFF

**Files:** Modify `scripts/smoke.sh`, `README.md`, `docs/superpowers/HANDOFF.md`; create `docs/runbooks/reminders.md`.

- [ ] **Step 1: Extend `scripts/smoke.sh`** (failing first against images from `main`): `start_api` passes a smoke-only VAPID key (generated per run with `openssl ecparam -name prime256v1 -genkey`, converted to 43 base64url characters), `VAPID_SUBJECT="$ORIGIN"`, `HBT_ALLOW_FAKE_PUSH=1` and no `NTFY_*`, on every start (steps 3, 7, 8). New step **4k**: as the guardian log a fever and a cough through the real forms and see them on the timeline; create an every-1-hour course whose start makes an occurrence due within the run; register a fake push device for the guardian through the API (a real P-256 public key and 16-byte auth from `openssl`) and opt the kid in; register one for the caregiver without opt-in; wait (bounded, 90 s) and assert `/v1/_test/pushes` (internal token) holds exactly one send, to the guardian's device, titled "Medication due for <kid>" with no medication name; the home Due list shows the course; log a dose and assert the next occurrence is not sent early; revoke the caregiver and assert nothing changes for it; `/healthz/reminders` 200 through the UI; the API refuses to start with a malformed `VAPID_PRIVATE_KEY`, with an `http:` `VAPID_SUBJECT`, and with `NTFY_TOKEN` set but no `NTFY_URL` (each run exits non-zero, its log names the variable and not the value). Step 9's log scan gains the medication name, dose text, temperature figures, the fake endpoint, the keys and the notes. Step 10's ceilings stay (60 MiB API with the worker running, 140 MiB UI; report the readings).
- [ ] **Step 2:** `docker build -t hbt-api:dev api && docker build -t hbt-ui:dev ui && scripts/smoke.sh hbt-api:dev hbt-ui:dev` → every line `PASS:`; `shellcheck scripts/smoke.sh`.
- [ ] **Step 3: Docs:** README (what reminders send and never send, the new required variables and `HBT_ALLOW_FAKE_PUSH`, the ntfy 72-hour cache statement; replace "nothing reads them yet"); `docs/runbooks/reminders.md` (each health answer, what to check when a reminder is missing, rotating the VAPID key invalidates every device, iOS needs the Home Screen app); HANDOFF (plan 6 app done; the **edge** is next: the VAPID-only PR before the bump, then the ntfy edge plan with rows E1–E2 and `docs/superpowers/research/reminders/ntfy-and-edge.md`; then plan 7; the owner's answers to the questions above; the items under "Not verified").
- [ ] **Step 4:** the whole pipeline in CLAUDE.md green (API, UI, images, smoke). **Step 5: Commit** `test: smoke covers reminders; docs: plan 6 done`.

---

## Edge follow-up (separate PRs in `davcs86/homelab-edge`)

1. **Before the plan-6 bump is deployed (question 1):** render `VAPID_PRIVATE_KEY='{{ edge_secrets.happybabytracker_vapid_private_key }}'` and `VAPID_SUBJECT='https://{{ happybabytracker_ui_private_hostname }}'` into `api.env` with asserts (43 base64url characters; literal secret read, EDGE-08), flip `test_vapid_is_not_rendered_in_plan_2`, update the runbook section 7 and `docs/decisions.md`. Outbound 443 from the API container: the DigitalOcean firewall allows all outbound and the `happybabytracker` compose network is a plain bridge (checked 2026-11-23); host-level egress rules `TODO(verify)`.
2. **ntfy, after rows E1–E2 are answered:** a shared `ntfy` network for ntfy and the API only; `happybabytracker_ntfy_enabled` (ships `false`) and the prefix; a write-only ntfy user and generated token (`init_secrets.py`, SOPS); `NTFY_*` rendering with asserts; the Gatus `/healthz/reminders` probe (after plan 6 is deployed); tests (`happybabytracker-config-test.py`, `gatus-config-test.py`, `init-secrets-test.py`) and docs; binding rules EDGE-03, 04, 08, 11, 13 and CLAUDE.md rules 1, 3, 6, 7. Then a flip PR after the owner subscribes a phone.

## Done when

- API and UI pipelines and `scripts/smoke.sh` pass on the branch head and CI is green.
- The access matrix covers every new route (30 new rows: symptoms 5, medications 5, doses 5, push 7, ntfy 6, `/healthz/reminders`, `/v1/_test/pushes`) and fails if one is missing.
- A parent logs a symptom and a medication course; a fake push delivers "Medication due for <kid>" exactly once to an opted-in device and to nobody else; a logged dose cancels the next reminder; revoking access stops it.
- The final report states which image content changed (`api/` and `ui/` sources; no Dockerfile or dependency change expected), that the edge must render `VAPID_*` before the plan-6 images are deployed, the ntfy privacy statement, and the owner's answers.

## Not verified (state in the PR)

- Apple's acceptance of a tailnet-only `https:` VAPID subject, iOS delivery, the module service worker on iOS Safari 17+, notification click into the Home Screen app and its separate login state: only a real iPhone can show these.
- Real delivery through FCM, Mozilla autopush and Apple (tests use a fake push service).
- ntfy publishing from the API container's address with `behind-proxy: true` and no `X-Forwarded-For` (the edge's existing publishers work from `127.0.0.1`).
- Host-level egress from the API container on the edge beyond the DigitalOcean firewall rules.

## Changes from the draft

1. Migration `0005_sickness_medication.sql` → `0006_sickness_medication.sql`; `schema_version` `"5"` → `"6"` in five assertions (plan 5.1 took 0005).
2. **Deploy order:** the draft shipped the app first and the edge after; with VAPID required at start that crashes the API, so the edge must render `VAPID_*` first (question 1, Edge follow-up 1).
3. **Materialisation:** the draft materialised only `now − 30 min .. now + 2 min`, which left nothing for "next due" and the home Due list to read (spec 11 says both read the reminder rows); now the next occurrence is always materialised and course, dose and kid-zone writes adjust rows in their own transaction (Open value 14).
4. The schedule engine moved from `api/internal/reminders` to `api/internal/schedule`: the store needs it, and the worker imports the store (an import cycle otherwise).
5. **Done rule:** the draft's window let one dose mark two occurrences done; now it marks exactly one (Open value 15).
6. Per-kid opt-in routes moved under `/v1/kids/{kid}/…` with class `Caregiver` (the draft's `/v1/me/…/kids/{kid}` under `Parent` could not give the grant 404 through the middleware, and the matrix would fail); `/v1/me/push-subscriptions` renamed `/v1/me/push-devices` and gained a label PATCH.
7. `/healthz/reminders` is class `Open` like `/healthz` (the draft said internal-token, which Gatus through the UI could not call); its 503 condition is defined (Open value 30).
8. The band-crossing change in the draft's Task 9 is dropped: `sickness` and `medication` are on in every band (`store.DefaultEnabled`).
9. Medication doses are not a timeline type by default (question 3): the UI's timeline is keyed on `TrackerKind` and doses have no log route; symptoms join as `TrackerKind` `symptoms` / type `symptom`.
10. `units.NewTemperature` does not exist yet and is its own task; units `c`/`f` match the profile's existing `temp` preference; the UI accepts two decimals in both units.
11. A course has one `start_at` instant instead of a `start_on` date plus a separate grid start.
12. The payload carries a server-built `title` (one source of the generic text for push and ntfy) instead of the service worker composing it; delivery states gained `gone` and `cancelled`, and deliveries a `parent_id` and `settled_at`.
13. Home Due: `dueView` becomes a union so vaccine entries keep their four keys; the shared `HOME_DUE_MAX` cap and per-kind "more" lines are defined (Open value 25).
14. The draft's 15 tasks are 22: API and UI split further (temperature units, push crypto vs sender, reminder store vs worker, health, UI plumbing in two, service worker vs notifications page).
15. Existing tests that start the API (config, main, the httpapi kit, smoke's `start_api` on every restart) are named in the tasks that make VAPID required.
16. Global Constraints and Review Focus fold in plan 5.1's lessons (`inReadTx`, unchanged-before-range, PATCH merges, `*_original`, `formUnits`, `submit()`, the stub mirroring handlers) and the HANDOFF's midnight-DST flaw.
