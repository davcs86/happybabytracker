# Plan 6 research: Web Push (Go API + SvelteKit 3 UI)

Read-only research. Nothing in any repository was changed. A working, tested stdlib prototype is kept next to this
note in `pushproto/` (`push.go.txt` 153 lines, `push_test.go.txt`, `go.mod.txt`; rename to `.go`/`go.mod` to run); it passes
`go test` on Go 1.27 (including the RFC 8291 Appendix A vector). It is evidence, not a deliverable to copy blindly.

## Rechecked 2026-11-23 (plan 6 refresh)

- The prototype's three tests (`TestRFC8291Vector`, `TestRoundTripAndLimits`, `TestVAPID`) pass on Go 1.27.0.
- `init_secrets.py::generate_vapid_private_key` in `homelab-edge` still writes 43 base64url characters, and the edge still
  does not render any `VAPID_*` line (`happybabytracker-config-test.py::test_vapid_is_not_rendered_in_plan_2`). Because
  plan 6 makes VAPID required at start, the edge must render it **before** the plan-6 images are deployed (section 4
  item 4 below; the order in `ntfy-and-edge.md` is corrected).
- `@sveltejs/kit` 3.1.1 in `node_modules` has `src/runtime/app/service-worker` and the `module_removed_service_worker`
  build error: section 3 still holds.
- xstockstrat `services/xstockstrat-notify/src/fanout/webPush.ts`: `WEBPUSH_TTL_SECONDS = 3600`, `WEBPUSH_SEND_TIMEOUT_MS =
  10000`, prune on 404/410 only: section 2 still holds.
- iOS (WebKit blog 13878, "Web Push for Web Apps on iOS and iPadOS"): iOS/iPadOS 16.4+, Home Screen web apps only (manifest
  `display` `standalone` or `fullscreen`), permission only on direct user interaction, and "allow URLs from
  `*.push.apple.com`" if outbound traffic is filtered.
- Apple's documentation (as quoted in pywebpush issue 175) lists `BadJwtToken` causes including "the JWT subject claim isn't a
  URL or mailto:"; one public report (a project's PR) saw Apple reject a reserved `.local` subject and fixed it with
  `mailto:`. The edge's subject would be the tailnet-only `https://happybaby.private.<domain>` (a real registrable domain,
  not reserved, but not publicly resolvable): acceptance is unverified and needs a real iPhone before relying on it.
- Status classes used by the plan: 201 sent; 404/410 gone; 429 (with `Retry-After`) and 5xx/timeout/network retry; 400, 401,
  403, 413 and other 4xx: fault, no retry, subscription kept.

## 0. Facts found in the repos that shape the answers

- The edge already generates the VAPID private key: `homelab-edge/scripts/init_secrets.py::generate_vapid_private_key` = raw P-256 scalar, `1 <= d < n`,
  32 bytes, **unpadded base64url, 43 chars** (charset `A-Za-z0-9_-`: no quote, backslash or line break). It is in SOPS but not yet rendered into `api.env`
  (`docs/runbooks/happybabytracker.md` s.7, `docs/decisions.md` L166): the Web Push release adds the env line + assert. So the API must parse exactly this form.
- `ui/src/hooks.server.ts`: `handle` redirects every non-public path to `/login`, returns 400 on a wrong origin, sets `cache-control: no-store` on everything not under `/_app/`.
  It sets **no CSP**; `Caddyfile.j2` (homelab-edge) has no CSP or security-header lines either. `e2e/pwa.spec.ts` already asserts the adapter-node static handler serves
  `static/` files (manifest, icons) **before** hooks run, with no cookie. So `/service-worker.js` and `/offline.html` are reachable signed out without touching hooks.
- `ui/static/manifest.webmanifest` already has `display: standalone`, `scope: /`, 192/512/maskable icons; `app.html` has apple-touch-icon and `apple-mobile-web-app-capable`.
- Installed: `@sveltejs/kit` 3.1.1, `adapter-node` 6.0.1. Kit 3 removed `$service-worker` (build error `module_removed_service_worker`): use `immutable`/`assets`/`prerendered`
  from `$app/manifest`, `version` from `$app/env`, `resolve` from `$app/paths`, and `self` from `$app/service-worker` (typed `ServiceWorkerGlobalScope`).
  `src/service-worker.ts` is the default entry (`kit.files.serviceWorker`); it is built by a separate Vite `serviceWorker` environment to `build/client/service-worker.js` (unhashed name).
- Go: toolchain 1.27.4 is available in `api/`; `crypto/hkdf` (1.24), `crypto/ecdh` (1.20) and `ecdsa.ParseRawPrivateKey` (1.25) are all in range. api/go.mod has only uuid, x/crypto, modernc.org/sqlite.

## 1. Go side

### 1a. Stdlib implementation: yes, reasonable (about 100 lines of crypto, ~150 with VAPID, plus tests)

Algorithm (all verified by the RFC 8291 App. A test passing byte for byte):
1. Per message: fresh ephemeral P-256 key (`ecdh.P256().GenerateKey`) and fresh 16-byte random salt.
2. `secret = ephemeral.ECDH(uaPublic)`; `key_info = "WebPush: info\0" || ua_public(65) || as_public(65)`.
3. `IKM = hkdf.Key(sha256.New, secret, salt=auth_secret, info=key_info, 32)`.
4. `CEK = hkdf.Key(sha256.New, IKM, salt, "Content-Encoding: aes128gcm\0", 16)`; `NONCE = hkdf.Key(..., "Content-Encoding: nonce\0", 12)`.
   Go's `hkdf.Key` does extract+expand in one call, which is exactly what RFC 8291/8188 specify.
5. Body = `salt(16) || rs(4, big-endian) || idlen(1)=65 || as_public(65) || AES-128-GCM(NONCE, plaintext || 0x02)`; no AAD. One record only.
6. Headers: `Content-Encoding: aes128gcm`, `TTL: <seconds>` (mandatory), optional `Urgency`, `Topic`, `Authorization: vapid t=<jwt>, k=<base64url 65-byte public key>` (RFC 8292).

Traps (each is a way to get a silent 400/401/403 or a body the browser cannot decrypt):
- **ES256 signature is raw `r||s`, 64 bytes (each padded to 32), not ASN.1 DER.** `ecdsa.SignASN1` returns DER: convert (prototype `rawRS`, via `encoding/asn1`). This is the most common bug.
- JWT claims: `aud` = origin of the *endpoint* (scheme://host[:port]) per push service, not the app's origin; `exp` <= 24 h ahead (use 12 h); `sub` = `https:` URL. Header `{"typ":"JWT","alg":"ES256"}`. base64url **without padding** everywhere.
- Keys from the browser: `p256dh` must be a 65-byte uncompressed point (0x04 first byte); `auth` exactly 16 bytes. `ecdh.P256().NewPublicKey` rejects compressed/invalid points, so use it for validation at registration time. Browsers return base64url unpadded via `toJSON()`; decode tolerantly (accept padded and standard alphabet) but store one canonical form.
- Record size: `rs` is the plaintext-record limit, not the actual length; 4096 is conventional. RFC 8291 s.4: push services must accept a 4096-octet body, so max plaintext = 4096 - 86 (header: 16+4+1+65) - 16 (tag) - 1 (delimiter) = **3993 bytes**; the generic reminder payload is ~100 bytes. Padding is optional (length hiding); `0x02` delimiter is mandatory on the last record (`0x01` is for non-final records, unused here).
- `idlen`/keyid carry the *application server's ephemeral* public key (not the VAPID key). A different key in the header than in `key_info` gives an undecryptable message with a 201 response (invisible failure: only the RFC vector or a decrypting test catches it).
- `TTL` is required (missing -> 400). `Topic` <= 32 chars from `[A-Za-z0-9_-]`; a same-Topic message replaces a pending one (useful for retries, so a re-sent reminder does not pile up). `Urgency: high` for reminders (Android Doze).
- Do not follow redirects on the push POST (`CheckRedirect` returns `http.ErrUseLastResponse`); set the 10 s timeout on the client; read at most a few KB of the response and always close the body.
- Treat the endpoint as a capability and as attacker-supplied input (see SSRF in section 4); never log it (design 13.3).
- Statuses: 201 = accepted. 404/410 -> delete subscription. 429 (honour `Retry-After`) and 5xx/timeout/network -> retry with backoff until the 30-minute expiry. 413 -> bug (payload too large), no retry. 400/401/403 -> configuration fault (bad JWT, VAPID key does not match the one the subscription was created with, FCM "sender mismatch"): do **not** delete the subscription, log once per tick as a fault, no retry for that occurrence. A rotated VAPID key shows up as 401/403 on every subscription, which is why the spec says it invalidates them all.

### 1b. webpush-go comparison

`github.com/SherClockHolmes/webpush-go` **v1.4.0**, tagged 2025-01-02 (Go proxy; the GitHub API was not reachable from this session, so the latest commit date and open issues are unverified). Licence MIT. go.mod declares `go 1.13`; dependencies `github.com/golang-jwt/jwt/v5 v5.2.1` and `golang.org/x/crypto v0.31.0` (x/crypto is already in api/go.mod; jwt/v5 would be the one new module). About 625 lines in total, read in full.

What it does: RFC 8291 encrypt, VAPID JWT via jwt/v5, `SendNotificationWithContext`, `Options{TTL, Urgency, Topic, HTTPClient, RecordSize, Subscriber, VAPIDPublicKey, VAPIDPrivateKey}`, `GenerateVAPIDKeys`.
Observed shortcomings (from the source): always pads every message to a full 4096-byte record; ignores `Options.VapidExpiration` (JWT `exp` hard-coded to now+12 h); uses `elliptic.Marshal/ScalarMult/Unmarshal` (deprecated since Go 1.21, still works); the default `http.Client` follows redirects and has no timeout unless you pass `HTTPClient`; returns a raw `*http.Response` (caller must close and classify); no endpoint validation; any `Subscriber` not starting with `https:` gets `mailto:` prepended; no retry logic (its tests, 93 lines, were not checked for the RFC vector).

Trade-offs:
- Stdlib: no new module (the project's stated preference), no deprecated APIs, 0 padding waste, full control of redirect/timeout/validation, one auditable file; cost: the project owns ~100 lines of crypto, mitigated by the RFC vector + round-trip tests below.
- webpush-go: less code to write, battle-tested interop, but adds jwt/v5 (+ go.sum entries), still needs a wrapper for validation, retry classification, timeout, no-redirect and header-less handling, so the saving is small.
- **Recommendation: stdlib in `api/internal/push`** (two files: encrypt+VAPID, sender with retry classification). If the owner prefers a dependency: pin `github.com/SherClockHolmes/webpush-go v1.4.0` (go.sum hashes committed, record why in the PR/decisions), pass an `HTTPClient` with a 10 s timeout and `CheckRedirect: ErrUseLastResponse`, and keep VAPID key parsing/validation in our own startup code.

### 1c. Testing without a push service

1. **RFC 8291 Appendix A vector** (done in the prototype, `TestRFC8291Vector`): plaintext `When I grow up, I want to be a watermelon`, as_private `yfWPiYE-n46HLnH0KqZOF1fJJU3MYrct3AELtAQ-oRw`, ua_public `BCVxsr7N_eNgVRqvHtD0zTZsEc6-VV-JvLexhqUzORcxaOzi6-AYWXvTBHm4bjyPjs7Vd8pZGH6SRpkNtoIAiw4`,
   auth `BTBZMqHH6r4Tts7J_aSIgg`, salt `DGv6ra1nlYgDCS1FRnbzlw`; the full expected body (from the RFC) is asserted byte for byte in `push_test.go`; it passes. This requires the encrypt function to take injectable ephemeral key and salt (production wrapper supplies random ones).
2. **Independent decrypt round trip** (`TestRoundTripAndLimits`): generate a subscription key pair + auth in the test, encrypt, decrypt with a separate receiver routine written from the RFC (not sharing code), assert plaintext, `len(body) <= 4096`, rejection of compressed keys and oversize payloads.
3. **Fake push service** with `httptest.NewTLSServer` (the sender must accept an injected `*http.Client`/transport and an endpoint-validator bypass only under `HBT_ALLOW_FAKE_PUSH`): assert headers (`TTL`, `Urgency`, `Topic`, `Content-Encoding`, `Authorization` parses, `aud` equals the server origin), verify the JWT signature against the derived public key, decrypt the body, and drive 201/404/410/429(+Retry-After)/500/slow(>10 s)/302 responses to check delete-vs-retry-vs-fault classification and that redirects are not followed.
4. VAPID unit test (done): generate -> 32/65 bytes, no `" \ \n = + /`; `NewSigner` derives the same public key; JWT claims; 64-byte signature verifies with `ecdsa.Verify`.

### 1d. VAPID key generation and storage

- Form: base64url **raw 32-byte scalar, no padding, 43 chars** (what `init_secrets.py` already writes). Rejected alternatives: PEM/DER (line breaks), standard base64 (`+ / =` are legal in a systemd/env file but are not what the ecosystem uses).
- Parse: `d, err := base64.RawURLEncoding.DecodeString(v)`; require `len(d)==32` (also reject padded input); `ecdsa.ParseRawPrivateKey(elliptic.P256(), d)` (Go 1.25+) rejects 0 and `>= n`. Refuse to start on any failure (spec 13.1).
- Derive public key: `k.ECDH()` then `.PublicKey().Bytes()` = 65 bytes uncompressed; serve `base64.RawURLEncoding` of it. Alternative without ecdsa: `ecdh.P256().NewPrivateKey(d)` -> `.PublicKey().Bytes()` (also validates the scalar) and a second parse for signing.
- Generation in Go (if ever needed, e.g. a `cmd` tool or test): `ecdh.P256().GenerateKey(rand.Reader)` -> `k.Bytes()` is the 32-byte scalar. The edge's Python generator is already correct (rejection-sampled `1 <= d < n`).
- `VAPID_SUBJECT` validation: starts with `https://`, parses as URL, no userinfo; the design says refuse to start otherwise.
- Cache the signed JWT per `aud` for up to ~1 h (optional, tiny optimisation; skip unless profiling shows need).

## 2. Reference implementation (xstockstrat) summary

Files read: `services/xstockstrat-notify/src/fanout/webPush.ts` (210 lines), `services/xstockstrat-ui/docs/pwa-push.md`, `public/sw.js`, `notify/migrations/002_push_subscriptions.up.sql`, `notifyServiceImpl.ts` register/unregister, notify `docs/context-constitution.md` (NOTIFY-5/6/8), notify CLAUDE.md.

Copy these:
- Node `web-push` with VAPID from three env vars; **channel disabled (not crashed) when any is missing, and an invalid `VAPID_SUBJECT` (must be `mailto:`/`https:`) is logged as an ERROR once**. In HBT the VAPID variables are mandatory and the process refuses to start, which is stricter and what the design wants.
- TTL 3600 s and 10 s send timeout as fixed constants (not config); distinct concerns (retention at push service vs socket timeout).
- Always show a notification on `push` (userVisibleOnly), even on missing or unparseable payload (fallback title/body); deterministic `tag` for OS-level coalescing.
- Prune on 404/410 only; any other error is logged and the subscription kept.
- Subscription row: id, user, `endpoint UNIQUE`, `p256dh`, `auth`, `user_agent`, `created_at`; upsert on endpoint.
- Metrics by outcome (sent/pruned/error/no_audience/disabled), log subscriptionId, never the endpoint.
- Subscribe flow: Push API with the VAPID **public** key, key crosses server->client by a server-side prop (`VapidKeyContext`), never `NEXT_PUBLIC_*`.
- Payload JSON `{title, body, icon, tag, url}` with a `url` the SW opens on click.
- iOS: only documents "add to Home Screen first"; not engineered separately.

Do NOT copy:
- **No retries**: a failed send (429/5xx/timeout) is logged and dropped; HBT needs backoff and a 30-minute expiry.
- Sequential `await` per subscription in one loop: one slow endpoint delays the rest (HBT: per-target delivery with a bounded concurrency; one failing channel must not block another).
- No validation of the subscription `endpoint` (any URL is POSTed by the server: SSRF). notify has an https/allowlist rule for webhooks but not for push endpoints.
- Upsert `ON CONFLICT (endpoint) DO UPDATE SET user_id = EXCLUDED.user_id` and **unregister by endpoint only, unscoped** (NOTIFY-6). HBT binds a device to a parent plus per-kid opt-ins; rebinding must clear the previous parent's opt-ins, and unregister must be scoped to the acting parent.
- Payload carries the alert's title/body verbatim and a fixed `/accounts/...` URL; HBT payload must be generic and carry only category + kid id + an in-app path.
- `sw.js` `notificationclick` matches windows with `u.includes(targetUrl)` (loose) and `skipWaiting`/`claim` without a version; use an exact same-origin path comparison and accept only paths starting with a single `/`.
- `sw.js` is hand-copied logic mirrored from a TS helper because Next cannot bundle it; SvelteKit bundles `src/service-worker.ts`, so import the helpers directly.
- Severity-gate and admin-audience logic: not applicable.
- Logs `subscriptionId` only (fine) but the unscoped `DELETE` and the endpoint-as-capability reasoning do not hold if the endpoint can be re-registered by another parent.

## 3. UI side (SvelteKit 3.1.1, adapter-node)

**Service worker file and build.** Create `ui/src/service-worker.ts`. Kit builds it to `build/client/service-worker.js` (no hash) and the Node adapter serves it like `static/` files: no hooks, no auth redirect, public. In dev Kit serves it at `/service-worker.js`. Scope is `/` automatically (script at root); no `Service-Worker-Allowed` header needed. Import `self` from `$app/service-worker` for typing (the project's tsconfig includes the DOM lib; avoid `/// <reference lib="webworker" />` clashes). `static/` stays for the manifest, icons and `offline.html`.

**Registration.** Kit auto-registers when the file exists and `kit.serviceWorker.register` is not false: it injects an inline script into the page that calls `navigator.serviceWorker.register('/service-worker.js', { type: 'module' })` on `load`. Nothing to write in the app. Alternative: `serviceWorker: { register: false }` and register only from the notifications screen. Recommendation: auto-register (every installed parent device gets the offline page; the SW holds no data). With JavaScript disabled nothing registers and nothing breaks. Subscribe code waits on `navigator.serviceWorker.ready`. The SW must be a module worker (Kit sets `type: 'module'`); Firefox and Safari support module service workers in current versions (Safari 17+ in the PWA); this should be verified on the real iOS device because Kit 3 forces `type: 'module'`.

**Handlers (minimal).**
- `install`: `skipWaiting()` and `caches.open('hbt-offline-' + version).then(c => c.add('/offline.html'))` (only one static file; `version` from `$app/env`).
- `activate`: `clients.claim()`; delete older `hbt-offline-*` caches.
- `push`: `event.waitUntil(showNotification(title, { body, tag, icon: '/icons/icon-192.png', data: { url } }))` always, even with a missing/bad payload (userVisibleOnly; Safari revokes subscriptions that receive silent pushes). `tag` = per-reminder id so a retried send replaces rather than duplicates. Parse payload in try/catch; fallback "Reminder due".
- `notificationclick`: `close()`; `clients.matchAll({ type: 'window', includeUncontrolled: true })`; if a window exists, `focus()` and `navigate(path)`, else `openWindow(path)`; accept only same-origin paths (`/` first, not `//`). On iOS `openWindow` works only inside the click handler.
- `fetch` (see open question): only for `request.mode === 'navigate'`: `fetch(request).catch(() => caches.match('/offline.html'))`. No other caching, no kid data in any cache.

**Offline page.** `ui/static/offline.html`: self-contained (inline CSS, no JS, no external requests), uses the same icon; precached at install. Because the design says the SW "handles install and push only", the `fetch` handler is a deviation: without it the browser shows its own error page and the offline fallback is impossible. Pages remain `no-store`; the Cache API holds only `offline.html`.

**VAPID public key to the browser.** The browser only talks to the SvelteKit server: add an API route `GET /v1/push/public-key` (any parent session) returning the base64url key; the notifications page's `+page.server.ts` `load` calls it through `locals.api` and returns it in `data`. No `$env/static/public`, no `PUBLIC_` variable, nothing else exposed. Client: convert base64url to a `Uint8Array` before `applicationServerKey` (some Safari versions reject the string form).

**Subscribe flow (account/notifications screen).**
1. Feature detect: `'serviceWorker' in navigator && 'PushManager' in window && 'Notification' in window`. On iOS Safari tabs `PushManager` is absent; it appears only when launched from the home screen (iOS 16.4+, manifest `display: standalone`). Detect `matchMedia('(display-mode: standalone)').matches || navigator.standalone` and show "add to the home screen" help when iOS and not standalone (spec 11).
2. Button handler (user gesture required): `Notification.requestPermission()` then `reg.pushManager.subscribe({ userVisibleOnly: true, applicationServerKey })`. Resolve `navigator.serviceWorker.ready` before the click (on mount) so the gesture is not consumed by awaits; Safari is strict about this.
3. `POST` `sub.toJSON()` (endpoint, keys) to a `+server.ts` / form action in the UI, which forwards to the API with `INTERNAL_API_TOKEN` and the session token. On page load, if permission is `granted`, call `getSubscription()` and re-sync with the server (covers rotation/expiry without a `pushsubscriptionchange` handler, which the SW scope excludes).
4. Kid selection per device: a normal `<form>` action using `use:enhance`, keyed by a device id the server returns at subscribe time (cookie or hidden field), so the selection works without JS once the device is subscribed. Subscribing itself needs JS by nature.
5. Denied permission is permanent in most browsers: show instructions instead of a button.

**Headers/CSP.** Nothing to change today: no CSP exists in `hooks.server.ts` or the edge Caddyfile. `hooks.server.ts` needs no change for the SW (static handler runs first). The `no-store` on all non-`/_app/` responses does not affect the SW script. Two future-proofing notes: (a) if a CSP is ever added with `kit.csp`, Kit adds nonce/hash to its inline registration script automatically, and `worker-src 'self'` (or `script-src 'self'`) must allow the worker; (b) the edge must not add long `Cache-Control` to `/service-worker.js` (browsers revalidate the main SW script regardless of max-age by default, but keep it uncached to be safe). The 400-on-wrong-origin rule applies to the POSTs to our own endpoints and is fine.

**Testing with Playwright (Chromium only; the project already runs phone/desktop Chromium projects).**
- `http://127.0.0.1:4173` is a secure context, so a service worker registers in the existing harness. Grant `permissions: ['notifications']` in `test.use`/context.
- Real `pushManager.subscribe()` in headless Chromium needs the FCM connection and normally fails: stub it with `page.addInitScript` (override `PushManager.prototype.subscribe`/`getSubscription` returning a fake object with `endpoint`, `toJSON()`, `getKey`) and assert via the stub API (`/__requests`) that the UI POSTed the expected body and kid opt-ins.
- SW behaviour: `context.serviceWorkers()` -> `sw.evaluate(() => self.dispatchEvent(new PushEvent('push', { data: ... })))` then `self.registration.getNotifications()` to assert title, tag and `data.url`; same pattern for `NotificationEvent('notificationclick')`. (Chromium-specific; mark as "to try"; fallback is CDP `ServiceWorker.deliverPushMessage`.)
- Static/public checks following `e2e/pwa.spec.ts`: no-cookie GET `/service-worker.js` -> 200 JS, `/offline.html` -> 200 HTML, no redirect.
- Offline page: after the SW controls the page, `context.setOffline(true)` and navigate: expect the offline text. Whether `setOffline` also applies to the SW's own fetches should be confirmed when written.
- Unit tests (vitest `server` project): the payload parser/path validation as pure functions imported by both the SW and tests (Kit bundles the SW so imports are fine).
- **Only verifiable on a real device**: the permission prompt UX, iOS standalone detection and `PushManager` appearing only after install, delivery through APNs/FCM/Mozilla autopush (including Apple accepting the VAPID `sub`), lock-screen rendering, tag coalescing, notification click opening the PWA and the login state in the standalone container (iOS PWA has separate cookies from Safari), Doze behaviour with `Urgency: high`, rotation/revocation, and tailnet reachability when the notification is tapped.

## 4. Risks and open questions for the owner

1. Offline page needs a `fetch` handler (navigations only, caches just `offline.html`), which departs from "install and push only"; accept, or drop the offline page?
2. Push endpoint trust: allowlist known push-service hosts (FCM, Mozilla autopush, Apple `web.push.apple.com`, WNS; brittle) or accept any `https` host on port 443 that resolves to a public IP (dial-time check; no redirects)? Either is needed to avoid server-side request forgery from a registered "endpoint".
3. VAPID `sub` is the app's own `https:` origin, which on the edge is a tailnet-only name; Apple/FCM acceptance of an unreachable `sub` is unverified: confirm on a real iPhone before shipping (the owner insisted on `https:`; `mailto:` is the fallback).
4. The API container needs outbound 443 to the push services; confirm the edge compose network/hardening allows it (edge PR), and that the `api.env` line + assert for `VAPID_PRIVATE_KEY` is part of the edge change.
5. TTL: spec starts at 3600 s, expiry at 30 min; proposal is `TTL = min(3600, seconds until due+30 min)` (at least 1) so a late-waking phone never shows a reminder the server already expired. Confirm.
6. If a second parent signs in on a browser already subscribed by another parent, should the endpoint be re-bound (clearing the first parent's kid opt-ins) or refused? Reference upserts and re-binds silently.
7. No `pushsubscriptionchange` handler (SW scope): recovery is page-load resync only; a device that never opens the app after a rotation silently stops getting reminders. Accept?
8. Own stdlib crypto (~100 lines, RFC-vector tested) vs `webpush-go v1.4.0` plus the new `golang-jwt/jwt/v5` dependency; recommendation is stdlib.
