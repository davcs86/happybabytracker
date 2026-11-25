# happybabytracker — health and growth tracker for babies and kids: design

Status: **approved by the owner** (2026-11-20). Plans: `docs/superpowers/plans/`.

Amended 2026-11-24 for single sign-on and public exposure (5.5, 15, 18): **specified, not built.** The shared design is
`davcs86/homelab-edge` `docs/superpowers/specs/2026-11-23-authelia-sso-respec.md` (§ 3 app contract, § 5, § 7, § 8;
phase P6). Once an image with 5.5 ships, and until the edge sets `AUTH_MODE`, the app runs in `local` and behaves as
before, with three exceptions from its first start: the env admin is copied into the `admins` table and its sessions
end once; `ADMIN_USERNAME`/`ADMIN_PASSWORD` are ignored from then on, so rotating the admin password in SOPS no longer
works and moves to the CLI's `reset-password` at that bump; and a password change counts against the login limiter
(5.3, 5.5). The edge retires both SOPS keys in a later change: the
bump that brings the image changes only its digest.

Items marked **(owner)** were decided by the owner in the design conversation.
Items marked **(call)** are the designer's choices, not owner decisions — correct any of them.

## 1. Intent

A self-hosted family health record. Parents log a child's day (feeds, diapers, sleep), growth,
vaccinations, illness, medication and milestones from their phones, and see it as curves, schedules and
a timeline. The record belongs to the **child**: several parents and caregivers share one child's
record, and one parent can hold several children.

It is the third custom-code service on the homelab edge. It is built like `davcs86/commsgun` (Go API,
SvelteKit dashboard, SQLite, two images from one repo) and released exactly like it: images built by
CI here, pinned by digest in `davcs86/homelab-edge`, live after a manual deploy there.

## 2. Goals, non-goals, anti-goals

**Goals**

- G1. Multiple kids; all data hangs off a kid, never off a parent **(owner)**.
- G2. Access is granted per kid **(owner)**. A parent sees only the kids they were granted.
- G3. A seeded platform admin who manages parent accounts and can never see a kid or kid data
  **(owner)**.
- G4. Growth: weight, length/height, head circumference and BMI-for-age against selectable reference
  sources, by age and birth sex, with corrected age for preterm babies. The source can be changed at
  any time **(owner)**.
- G5. Vaccination: show the schedule of a selectable national schema, record administered doses
  (date required, lot optional), soft-flag overdue doses. The schema can be changed at any time
  **(owner)**.
- G6. Trackers: bottles, nursing sessions, pumping, sleep, diapers, feeding (solids), developmental
  milestones, sickness, medication with scheduled reminders **(owner)**.
- G7. Every tracker can be turned off and on again per kid; turning one off never deletes data
  **(owner)**.
- G8. All tracked data is queryable by time range, editable at any time and deletable **(owner)**.
- G9. The home screen adapts to the kid's age **(owner)**.
- G10. An installable PWA, mobile-first, with no feature that exists on only one device class
  **(owner)**.
- G11. Medication reminders by Web Push with VAPID, opted into per device **(owner)**.
- G15. The same reminders can also go to an ntfy server when the deployment configures one. ntfy is
  an addition, never a replacement: Web Push is always present **(owner)**. Nothing in this repository
  depends on `homelab-edge`; a fork runs against any ntfy server or none **(owner)**.
- G12. PDF export of a kid's record for a time range **(owner)**.
- G16. Medical visits, scheduled or past, with post-visit notes, one form that records the visit's growth,
  vaccines and medication, and an `.ics` file per visit (amended 2026-11-25:
  `2026-11-25-medical-visits-design.md`) **(owner)**.
- G13. Each user picks display units; storage keeps metric and imperial values side by side,
  converted when a row is saved **(owner)**.
- G14. The database is backed up off the edge twice a day, encrypted **(owner: twice a day)**.

**Non-goals (stage one)**

- Offline use. The PWA is installable but needs a connection **(owner)**.
- ~~Public ingress. The app is tailnet-only **(owner)**.~~ Amended 2026-11-24: the UI is public at
  `happybaby.public.<domain>` behind single sign-on (5.5, 15); the API stays on the compose network.
- ~~Authelia login. The design leaves a seam for it (5.4) and nothing more **(owner)**.~~ Amended 2026-11-24: single
  sign-on through a provider-neutral OIDC client (5.5), per the homelab-edge SSO spec.
- Canadian vaccination schedules (they are provincial) **(owner: left out for now)**.
- Photos, attachments **(owner)**. Edit history **(owner)**.
- Languages other than English **(owner)**.
- Email or SMS notifications, reminders for anything other than medication, push for overdue vaccines.
- Import from other apps, data sharing with clinicians other than the PDF.
- Diagnosis or advice of any kind: the app records and displays; it never interprets a symptom, a
  percentile or a missed vaccine beyond showing it.

**Anti-goals** — the design has failed if any of these happen, even if everything else works

- A parent reads or changes a kid they were not granted, or learns that such a kid exists.
- The admin reads any kid data, including a kid's name.
- A kid's record becomes unreachable because its last guardian was removed or disabled.
- A health value, a kid's or parent's name, a username or a secret appears in a log.
- ~~The app is reachable from the internet.~~ Amended 2026-11-24: the API is reachable from the internet, or the UI
  is reachable from it other than through single sign-on or, in `local`/`both`, the rate-limited password login.
- A disabled parent, an unlinked identity or a person outside the IdP's `happybabytracker` group signs in. Sessions
  that exist when the IdP removes a person last until the admin disables or unlinks them (5.5; residual 2 in 18).
- A percentile or schedule is computed from reference data whose origin and version are not recorded.
- The droplet is lost and more than half a day of entries cannot be restored.

## 3. Constraints found in the code

1. `happybabytracker` holds only `README.md`, `LICENSE` and `.gitignore`.
2. commsgun's shape (`commsgun/docs/superpowers/specs/2026-11-13-commsgun-design.md`): Go 1.27 binary
   on distroless, `modernc.org/sqlite` (pure Go, static binary), data on a named volume at `/data`,
   uid 10001, read-only root; SvelteKit 3 / Svelte 5 with the Node adapter, shadcn-svelte, Tailwind 4,
   layerchart, Node 24; the browser talks only to the SvelteKit server, which calls the API over the
   compose network with a shared token; limits and durations in one constants file.
3. commsgun's dashboard has exactly one user, `admin`, with the password in an env var. Nothing there
   supports several users, so identity and authorization are new work here.
4. commsgun's store is deliberately not backed up. The edge's nightly backup
   (`homelab-edge/roles/backup/templates/edge-backup.sh.j2`) archives headscale only: SQLite `.backup`,
   `PRAGMA integrity_check`, tar, `age`, rsync to the home backup host, then a restore drill and a
   Gatus heartbeat (`edge_nightly-backup`).
5. The edge is one `s-1vcpu-2gb-amd` droplet. Containers use the `x-hardening` anchor, `read_only`,
   ports bound to `127.0.0.1`, memory caps of 96–192 MB. The owner confirmed there is room for two more
   containers.
6. Tailnet-only services live under `*.private.<domain>` behind Caddy with a wildcard certificate.
   A new name is wired in four places (EDGE-07): hostname var, `host` handle in `Caddyfile.j2`,
   `_names` in `roles/tailscale/tasks/main.yml`, a Gatus probe.
7. Tailnet grants are per port, not per name. `group:members` (today only `david@`) may reach
   `tag:edge:443`, which is every `*.private` service: Gatus (no login), ntfy, the commsgun dashboard.
   Every person who uses this app needs a tailnet device with that reach, or a narrower rule in Caddy.
8. The policy tests deny `tag:edge → tag:db:5432`. Using the shared PostgreSQL would mean inverting
   that deny for the internet-facing host.
9. The droplet's Cloud Firewall allows all outbound traffic, so the API can reach the browser push
   services (FCM, Mozilla autopush, Apple) directly.
10. Secrets reach containers through Ansible-rendered `0600` env files read from
    `secrets/edge.sops.yaml`; generated keys are declared in `scripts/init_secrets.py`; a secret value
    may not contain a quote, backslash or line break.
11. Web Push on iOS works only for a PWA added to the home screen (iOS 16.4 or later).

## 4. Approaches considered

### 4.1 Application shape

1. **commsgun mirror — chosen (owner: acceptable as long as it serves the PWA and Web Push).**
   Two images. The Go API owns users, sessions, authorization, data, PDF export and the reminder
   worker, on a listener that is never published. The SvelteKit server is the only published port; it
   holds the session cookie and calls the API server-side. Authorization is enforced in the API, so a
   bug in the UI cannot widen access. Reuses commsgun's CI, digest pins, bump workflow and env-file
   shapes unchanged.
2. **One Go binary serving a static SPA.** One image and about 190 MB less memory. Rejected: the
   browser would call the API directly (CORS and CSRF handling in Go), and it breaks the two-image
   release contract and the UI patterns already proven in commsgun.
3. **SvelteKit-only monolith.** Fewest parts. Rejected: drops Go, moves the scheduler and
   authorization into Node, and is the furthest from "based on commsgun".

### 4.2 Storage

1. **SQLite on the edge, with encrypted snapshots to the home backup host — chosen (owner asked for a
   recommendation on security and resilience; approved with a twice-a-day interval).** No new network
   path, available whenever the edge is up, backups `age`-encrypted at home with a restore drill.
   Cost: up to half a day of entries lost if the droplet dies; the live database is plaintext on the
   droplet disk, the same trust level as the SOPS-rendered env files there.
2. **Shared PostgreSQL on the home DB host.** Point-in-time recovery. Rejected: needs
   `tag:edge → tag:db:5432` (constraint 8), makes the app depend on the home uplink and the DB host,
   puts its backups in the pgBackRest repository on the droplet, for which no repository cipher is
   configured, and departs from commsgun's store.

## 5. People, access and login

### 5.1 Principals

| Principal | Exists as | Can do | Cannot do |
|---|---|---|---|
| **Admin** | env vars only; not a row in `parents` (amended 2026-11-24: a row in `admins`, still not in `parents`, 5.5) | create parents, issue invite and password-reset links, disable and enable parents | see, list, count or name any kid; hold a kid grant; change its own username or password in the app (amended 2026-11-24: its password changes only through the CLI's `reset-password`, 5.5); be deleted or demoted **(owner)** |
| **Parent** | row in `parents` | create kids; everything their grants allow; change their own password, display name, units | see parents or kids they share nothing with |

The admin's lack of kid access is structural, not a permission check: `kid_access.parent_id` is a
foreign key to `parents`, the admin is not in that table, and the admin session type is refused by
every route outside `/v1/admin/*` and `/v1/session`.

The admin's username and password come from `ADMIN_USERNAME` and `ADMIN_PASSWORD` (SOPS). At start the
API stores a keyed fingerprint of both in a `meta` row; when it differs from the stored one, every
admin session is deleted. Changing either in SOPS and redeploying therefore signs the admin out.
Amended 2026-11-24 (not built): the two variables seed the `admins` table once and are then ignored, and the
fingerprint is dropped (5.5).

### 5.2 Per-kid access

| Role | Entries (create, edit, delete) | View, export | Kid profile and settings | Grant and revoke access | Delete the kid |
|---|---|---|---|---|---|
| `guardian` | yes | yes | yes | yes | yes |
| `caregiver` | yes | yes | no | no | no |

Two levels **(owner)**. A kid has one or more guardians and zero or more caregivers.

- A parent who creates a kid becomes its guardian **(owner)**.
- A guardian grants access by typing another parent's exact username and choosing a role. There is no
  directory of parents **(call)**. An unknown username and a disabled parent get the same answer.
- A kid always has at least one **active** guardian. The API refuses, in one transaction: revoking or
  downgrading the last guardian, a guardian leaving as the last one, and the admin disabling a parent
  who is the only active guardian of any kid **(owner)**. The admin's refusal says only "this parent
  is the sole guardian of at least one child"; no name or count.
- A request for a kid the caller has no grant on answers 404, identical to a kid that does not exist.

### 5.3 Accounts and sessions

- The admin creates a parent with a username (3–32 of `a-z0-9._-`, permanent) and a display name. The
  API returns a one-time invite link; the parent opens it and sets a password. Invite links last 72
  hours, reset links 1 hour **(call)**; tokens are 32 random bytes, stored hashed, single use.
- Passwords: at least 12 characters **(call)**, argon2id. A parent changes their own password in the
  app (current password required). A forgotten password is an admin-issued reset link **(owner)**;
  using it deletes that parent's sessions.
- Sessions are opaque 32-byte tokens, stored hashed in the API's `sessions` table. The UI keeps the
  token in a cookie (`HttpOnly`, `Secure`, `SameSite=Strict`, path `/`). Parent sessions slide for 30
  days, the admin session lasts 12 hours **(call)**. Logout deletes the row. A parent can see and end
  their other sessions.
- Failed logins are limited per client address and per username, in memory **(call: 5 per minute)**.
  The limiter counts admin and parent attempts together and answers both the same way. Amended 2026-11-24: re-auth
  and the password change count too (5.5).
- Disabling a parent deletes their sessions, push subscriptions and ntfy topic at once. Amended 2026-11-24: so does
  unlinking a parent's SSO identity, and a disabled linked parent's SSO attempt repeats it (5.5, owner).

### 5.4 The Authelia seam

Superseded by 5.5 on 2026-11-24 (the `provider` of an OIDC identity is the issuer URL, not `'authelia'`).

Authelia is approved for the edge but not deployed, and it is planned for **public** apps
(`homelab-edge/docs/runbooks/sso.md`). Nothing is built for it now. The seam is two things:

- A parent's credentials live in `identities (parent_id, provider, subject, secret_hash)`, unique on
  `(provider, subject)`. Today every row has `provider = 'local'`, `subject = username`. An Authelia
  login later adds rows with `provider = 'authelia'` and the Authelia `sub`; no table changes.
- One Go package turns a request into a principal. Adding a second way to authenticate changes that
  package and the UI's login route, and nothing that checks access.

### 5.5 Single sign-on (amended 2026-11-24, not built)

Implements the shared app contract (homelab-edge SSO spec § 3) and replaces the seam of 5.4. The app stays a
provider-neutral OIDC client: nothing here names Authelia, and a fork runs with `AUTH_MODE=local` and no IdP.

**Principals.** The admin moves from env vars into its own tables, so it is still never a row in `parents` and still
never holds a `kid_access` row (Rule 2):

| Table | Columns |
|---|---|
| `admins` | id, username (5.3 rules, permanent, never reused), created/updated. No disabled flag: the admin is never deleted or demoted (5.1), and the CLI's `unlink-sso` stops its SSO **(call)** |
| `admin_identities` | admin, provider, subject, secret hash; primary key `(provider, subject)`, as `identities` |
| `sessions` | gains `admin_id` (references `admins`) and `reauth_at` (unix seconds, nullable; set by re-auth below); `kind = 'admin'` exactly when `admin_id` is set and `parent_id` is not |

`identities` keeps its shape. An OIDC row has `provider` = the issuer URL (exact string), `subject` = `sub`,
`secret_hash` NULL. At most one OIDC identity per principal and issuer, in both identity tables (a partial unique index)
**(call)**. One `(provider, subject)` pair belongs to a parent or an admin, never both: every store transaction that adds
an OIDC identity (link, SSO redemption) checks both tables **(call)**. The migration rebuilds `sessions` for the new
columns and deletes admin sessions, so the admin signs in once more after the upgrade.

**One username namespace.** A username belongs to at most one row across `parents` and `admins`. `createParent`, the
CLI's `create` and the start-up import each refuse a name held in either table inside their own transaction
(`createParent` answers 409 `username_taken`, as today; the import's refusal stops the start, as `checkAdminUsername`
does now) **(call)**. The password login looks the name up in `admins`, then in `parents`, and runs exactly one password
verification on every path, against the dummy hash when neither holds it, as today.

**Import.** After migrations, in one transaction that first checks that `admins` is empty, the API creates the admin
from `ADMIN_USERNAME`/`ADMIN_PASSWORD` with a local identity (argon2id), once **(call: a start-up step, since SQL cannot
hash)**. The two variables are validated there, after the store opens, no longer in `config.Load`: while `admins` is
empty they are required (the username pattern, a password of at least 16 characters, `MinAdminPasswordLen`) and the
start fails without them; once it holds a row they are ignored, with one warning at each start that sees either set.
The 5.1 fingerprint and its `meta` row are dropped. From the first start of this image, changing the admin password in
SOPS changes nothing: rotation is the CLI's `reset-password`.

**CLI.** The API binary also runs `admin <verb>`, on the edge as `docker compose -f {{ edge_root }}/compose.yml run --rm
-T happybabytracker-api admin <verb>` (homelab-edge EDGE-03), any password on stdin and at least 16 characters. A verb
opens the store as the API does (`store.Open`, which applies pending migrations), so `create` works on a fresh volume:

- `create <username>`: only while `admins` is empty;
- `reset-password <username>`: sets the admin's local password and deletes that admin's sessions;
- `unlink-sso <username>`: deletes that admin's OIDC identities and sessions, so the admin can link again, for an IdP
  account that was recreated or the edge's "both databases lost" recovery (SSO spec § 8) **(call)**.

The CLI is a second, short-lived SQLite writer beside the running API (6.1): WAL and the busy timeout serialise the two,
and `create` and the import each check that `admins` is empty inside their own immediate transaction, so exactly one of
them creates the admin and the other changes nothing (`create` exits with an error).

**Modes.** `AUTH_MODE` is read by the **API**, which is the authority **(call)**: `local` (default), `oidc`, `both`; any
other value stops the start. The UI reads the mode from `GET /v1/auth` on each load of the login, invite, reset and
account pages. Switching mode needs no login: the API's env on the edge. In `oidc`, password redemption of invite and
reset links is refused, and in `local`, SSO redemption is refused **(owner, 2026-11-24)**. What each mode allows:

| | `local` | `oidc` | `both` |
|---|---|---|---|
| Password login, `POST /v1/session` | yes | 403 `mode_refused` | yes |
| SSO login, `POST /v1/session/oidc` | 403 `mode_refused` | yes | yes |
| Invite or reset by password, `POST /v1/tokens/{token}/redeem` | yes | 403 `mode_refused` | yes |
| Invite by SSO, `POST /v1/tokens/{token}/redeem-oidc` | 403 `mode_refused` | yes | yes |
| Re-auth and Link SSO | 403 `mode_refused` | 403 `mode_refused` | yes |

The mode check runs first, before a token is looked up, a password hashed or the limiter counted, so a refused
redemption leaves the token usable. In `oidc` the invite page offers only "Continue with single sign-on", and the reset
page says the link works only once the operator switches the app to `both`: a reset link is break-glass, and only in
`both`. In `local` the invite page offers only "Set a password". In `both` it offers both, and the password login keeps
working while the issuer is unreachable.

**Relying party.** The UI holds the settings: `OIDC_ISSUER`, `OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET` (all three or none),
`OIDC_SCOPES` (default `openid profile` **(call)**: the app reads no email; the edge's client may stay registered with
`openid profile email`, SSO spec § 4.6), and `SESSION_SEAL_KEY`, required with them. The UI refuses to start on a
partial set and on `OIDC_AUTO_PROVISION` set to anything: it is **not** implemented, parents are created by the admin
(5.3) **(call: a deliberate deviation from the shared config list)**.
Discovery and the JWKS are fetched on first use and cached; a failure fails that SSO attempt only ("Single sign-on is
unavailable"), never the start or the password login. When the API reports `oidc` or `both` and the UI has no
relying-party settings, the UI shows no SSO button, its login page in `oidc` says sign-in is unavailable, and it logs
`oidc_not_configured` **(call)**. Library: `openid-client`, exact pin **(call)**.

**UI routes.** Every path under `/auth/` is public (no session needed) and exempt from the admin redirect in
`hooks.server.ts`; `/auth/*`, `/invite/*` and `/reset/*` send `Referrer-Policy: no-referrer` **(call)**.

- `POST /auth/sso`: the login page's SSO button; purpose `login`, with the validated `next` path.
- `POST /invite/{token}?/sso`: the invite page's SSO button; purpose `invite`. Starting from a form POST keeps the token
  out of any new URL.
- `POST /account/link-sso` (parent) and `POST /admin/account/link-sso` (admin; a new admin account page): Link SSO's
  password field; calls the re-auth route, then starts purpose `link`.
- `GET /auth/callback`: the redirect URI, `${ORIGIN}/auth/callback`, with `response_mode` `query`.
- `POST /auth/confirm`: Confirm or Cancel on the link and invite confirmation.

**The flow.** The start writes a transaction cookie `hbt_oidc` (`HttpOnly; Secure; SameSite=Lax; Path=/auth`; 10
minutes, in `ui/src/lib/server/constants.ts`), AEAD-encrypted with a key derived from `SESSION_SEAL_KEY` **(call)**. It
holds the PKCE (S256) verifier, `state`, `nonce`, the purpose, `next` for `login`, and the purpose's binding: for `link` the session token's hash, for
`invite` the raw invite token. `link` and `invite` send `prompt=login` (SSO spec § 3, S17); `login` does not. The
callback checks `state` against the cookie, exchanges the code, and validates the ID token (`alg` RS256 from the
issuer's JWKS, `iss`, `aud`, `exp`, `nonce`). It then answers **200**, never a redirect, so the next step is a
same-origin navigation that carries the `SameSite=Strict` session cookie, whatever site the IdP is on **(call)**:

- an IdP error in the callback (`error=access_denied` for a person outside the group, or any other) or a failed check
  clears `hbt_oidc` and answers the login page (or the invite or account page the flow started from) with "Single
  sign-on did not complete"; nothing reaches the API **(call)**.
- `login`: calls `POST /v1/session/oidc {iss, sub}`, sets the session cookie, and answers a page that continues to
  `next` (the admin home for an admin) by meta refresh, with a Continue link.
- `link`, `invite`: rewrites the cookie as the pending `{purpose, binding, iss, sub, label}` (same attributes, the
  remaining lifetime) and answers "You are linking <label> to <username>" with Confirm and Cancel; for `invite` the
  username comes from `GET /v1/tokens/{token}`.

`POST /auth/confirm` is same-origin, so it carries the session cookie. It clears `hbt_oidc`; on Confirm, for `link` it
checks that the session cookie's token hashes to the bound hash and calls `POST /v1/me/identities` or
`POST /v1/admin/me/identities {iss, sub}` with that session; for `invite` it calls
`POST /v1/tokens/{token}/redeem-oidc {iss, sub}` and sets the returned session's cookie.

**The label.** `preferred_username` comes with the `profile` scope, which the edge's `happybabytracker` client is
registered with (SSO spec § 4.6). The UI reads it from the ID token, else from one userinfo call with the access token;
it lives only inside the encrypted `hbt_oidc` cookie until the confirmation, and is never stored by the API, sent
to it or logged. Without it the label is "the single
sign-on account you just signed in with" **(call)**.

**API side.** `POST /v1/session/oidc` looks `(iss, sub)` up in `identities`, then in `admin_identities`, and issues a
parent or admin session with the lifetimes of 5.3; an unknown pair or a disabled parent → 403 `not_linked`, one answer.
The IdP gates who may open the app (group `happybabytracker`); the app reads no groups, email or name.

- Re-auth (`POST /v1/me/reauth`, `POST /v1/admin/me/reauth`, body `{password}`) verifies the principal's local password,
  counts against the 5.3 limiter per address and per username like a login, and stamps `sessions.reauth_at`; a wrong
  password → 403 `wrong_password`.
- The identity routes require `reauth_at` within 5 minutes (`ReauthWindow` in `constants.go`), else 403
  `reauth_required`. Both identity routes and `redeem-oidc` answer 409 `already_linked` when the pair belongs to anyone
  or the principal already has an OIDC identity for that issuer.
- `redeem-oidc` does, in one transaction, what `redeem` does without the password: it marks an unused, unexpired invite
  of an enabled parent used, adds the identity and issues a parent session. A reset token, like an unknown, used or
  expired one, → 404 `not_found` **(call)**.
- `POST /v1/me/password` now counts against the same limiter, so a stolen session cannot guess the current password
  from the internet without limit **(call)**.

A parent who joined by SSO has no password. The admin's link for them is an invite (`resetLink` issues one when there
is no password), redeemable by password in `local` or `both`.

**Unlink.** The admin removes a parent's OIDC identities (`POST /v1/admin/parents/{parent}/unlink-sso`, 204; 404
`not_found` for an unknown parent), for an IdP account that was recreated or a wrong link; the admin's own go through
the CLI's `unlink-sso`. Parents cannot unlink themselves **(call)**. The parent view of `GET /v1/session` and the
admin's parent list gain `sso_linked` (a boolean), so the UI shows Link SSO or Unlink; the admin view of
`GET /v1/session` gains `username` and `sso_linked` for the admin's own Link SSO page **(call)**.

**Offboarding and notifications (owner, 2026-11-24).** Signing out does not stop reminders (11), and the IdP's removal
of a person is invisible to the app: it stops new SSO logins only, and a live parent session slides for 30 days (5.3).
So the app's own steps are what stop them, each in the store transaction that makes the change: disabling a parent
(5.3: sessions, push subscriptions, ntfy topic); **unlinking** a parent's OIDC identities deletes the same three; and
an OIDC session request for a linked parent who is disabled repeats that purge (idempotent) before it answers
`not_linked`. Until the admin disables or unlinks an offboarded parent, their sessions keep working and their devices
keep receiving "<Category> due for <kid's name>": homelab-edge `sso.md`'s offboarding list names the disable step for
this app.

**Routes and the access matrix.** Each new route goes into `routes.go` and the pinned `routeClasses` together
(`CLAUDE.md` How to Act 3):

| Route | Class | Answers beyond 2xx |
|---|---|---|
| `GET /v1/auth` | `Internal` | — (`{mode}`) |
| `POST /v1/session/oidc` | `Internal` | 403 `mode_refused`, 403 `not_linked` |
| `POST /v1/tokens/{token}/redeem-oidc` | `Internal` | 403 `mode_refused`, 404 `not_found`, 409 `already_linked` |
| `POST /v1/me/reauth` | `Parent` | 403 `mode_refused`, 403 `wrong_password`, 429 `rate_limited` |
| `POST /v1/admin/me/reauth` | `Admin` | as above |
| `POST /v1/me/identities` | `Parent` | 403 `mode_refused`, 403 `reauth_required`, 409 `already_linked` |
| `POST /v1/admin/me/identities` | `Admin` | as above |
| `POST /v1/admin/parents/{parent}/unlink-sso` | `Admin` | 404 `not_found` |

Existing routes gain: `POST /v1/session` and `POST /v1/tokens/{token}/redeem` 403 `mode_refused`; `POST /v1/me/password`
429 `rate_limited`. Admin sessions stay refused outside `/v1/admin/*` and `/v1/session` (Rule 2): the new session and
token routes are `Internal` and take no session.

**Routes that accept a password**, for the edge's `<site>_login` and `<site>_login_hourly` Caddy zones (SSO spec § 4.3):
`POST /login`, `POST /invite/{token}`, `POST /reset/{token}`, `POST /account` (its `?/password` action; the page's
other actions share the path), `POST /account/link-sso` and `POST /admin/account/link-sso`. The invite page's SSO
start shares the `/invite/{token}` path and falls in the same zones.

**Logs and OTEL.** No `sub` or `preferred_username` in a log or an error (Rule 5) **(call)**; refusals log their code.
The callback's whole query (`code`, `state`, `error`, `error_description`) is dropped from OTEL like `/invite/*` and
`/reset/*`.

**Tests the builder adds.** Every new route and its refusal in each mode, per the table above, including that a
password redemption refused in `oidc` and an SSO redemption refused in `local` leave the token usable; the import
(once, then ignored with a warning; a second start changes nothing; a parent holding the name stops the start); the
username namespace across both tables; the three CLI verbs, and `create` racing the import; the admin still refused on
every kid route after an SSO login (the matrix test); the cross-table uniqueness of a pair; re-auth counting against
the limiter, and the password change too; the link flow's re-auth window, session binding and `already_linked`;
invite redemption by SSO with `prompt=login` and the confirmation, including a used, expired or reset token; the
callback answering 200 and the session cookie arriving on the next navigation; unlink and the disabled-parent OIDC
request each deleting sessions, push subscriptions and the ntfy topic; the UI's refusals of a partial relying-party
set and of `OIDC_AUTO_PROVISION`; the conformance cases of the SSO spec § 3 (bad `state`, wrong `iss`/`aud`, expired
token, unlinked `sub`, replayed code, link while another IdP session is live) against a stub issuer; no `sub`, label or
token in captured logs.

The access matrix (`matrix_test.go`) runs with `AUTH_MODE=both`, so no route answers `mode_refused` there; its fixture
seeds an `admins` row and gives the admin session that `admin_id`. Every new JSON handler decodes its body before the
mode and re-auth checks (the matrix's decode-first expectation), and the per-mode refusals are tested apart from the
matrix. Smoke assertion 8 changes: restarting the API with a different `ADMIN_PASSWORD` keeps the old password working
and logs one warning; then the CLI's `reset-password` refuses and clears the old admin cookie, and the new password
logs in.

## 6. Data

### 6.1 Storage rules

- SQLite, WAL mode, foreign keys on, one process (amended 2026-11-24: plus the admin CLI's short-lived runs, 5.5).
  Ids are UUIDv7. Forward-only migrations embedded in
  the binary and applied at start.
- Every tracker has its own table with typed columns (6.3). Every entry row carries `kid_id`,
  `created_by`, `updated_by` (parent ids), `created_at`, `updated_at`, and an optional `note`.
- Instants are stored in UTC. Calendar dates (birth date, vaccination date, measurement date) are
  stored as dates. Each kid has an IANA time zone, set at creation from the creating device and
  editable; "today", day boundaries and reminder clock times use it **(call)**.
- Deletes are hard deletes **(owner)**. Deleting a kid (guardian only, confirmed by typing the kid's
  name) removes every row that references it. A deleted row stays in the backup archives until they
  rotate.
- Every store function that reads or writes kid data takes the kid id, and every such route passes
  one middleware that resolves the caller's role for that kid before the handler runs.
- All limits, caps and durations live in one constants file, as in commsgun.

### 6.2 Units

- A parent's profile holds four display preferences: weight (kg, or lb and oz), length (cm or in),
  volume (ml or US fl oz), temperature (°C or °F) **(owner: per user)**. Defaults are metric
  **(call)**.
- Every measured quantity is stored three ways in its row: the metric value, the imperial value, and
  which unit the user typed **(owner: both as columns)**. Both values are fixed-point integers
  (grams, millimetres, millilitres, hundredths of °C; hundredths of an ounce, of an inch, of a fluid
  ounce, of °F) **(call)**.
- One Go package converts, at save and at edit only **(owner)**. The typed value is stored exactly;
  the other is derived and rounded. Percentiles always read the metric columns.
- Ounces of volume are US fluid ounces (29.5735 ml) **(call)**.

### 6.3 What is stored

| Table | Purpose | Columns beyond the common ones |
|---|---|---|
| `parents` | accounts | username, display name, unit preferences, disabled flag |
| `identities` | credentials (5.4, 5.5) | provider, subject, secret hash |
| `admins`, `admin_identities` | the admin principal and its credentials (5.5, amended 2026-11-24) | username; provider, subject, secret hash |
| `sessions`, `one_time_tokens` | login state (5.3) | token hash, kind, expiry; sessions gain `admin_id` and `reauth_at` (5.5) |
| `kids` | profile and settings | name, birth date, birth sex (`female`/`male`), gestational age at birth (weeks, days; optional), time zone, growth source, vaccine schema (nullable) |
| `kid_access` | grants | kid, parent, role |
| `kid_modules` | per-kid overrides (7.2) | kid, module, enabled |
| `bottle_feeds` | bottles | time, contents (`breast_milk`/`formula`/`other`), volume |
| `nursing_sessions` | nursing | start, end (null while running), left seconds, right seconds, side in use and since when (while running) |
| `pumping_sessions` | pumping, recorded on the kid it is for **(call)** | start, duration, volume, side |
| `sleep_sessions` | sleep | start, end (null while asleep) |
| `diapers` | diapers | time, kind (`wet`/`dirty`/`both`/`dry`) |
| `meals` | feeding (solids) | time, what was eaten (text), reaction (text, optional) |
| `growth_measurements` | growth | date, any of weight, length/height, head circumference, visit (optional, amended 2026-11-25) |
| `vaccine_doses` | administered doses (9) | administration id, date, antigen, lot (optional), product name (optional), visit (optional, amended 2026-11-25) |
| `vaccine_dismissals` | dismissed schedule lines (9) | schema, line, reason |
| `milestones_reached` | milestones (10.1) | catalog id or custom text, date |
| `symptoms` | sickness (10.2) | time, symptom, severity (optional), temperature and method (for `fever`) |
| `medications` | a medication course | name, dose text, schedule (10.3), start, end, active, visit (optional, amended 2026-11-25) |
| `visits` | medical visits (amended 2026-11-25, visits spec 5.1) | start, length, kind, provider, place, reason (the common note), post-visit notes, cancelled at, revision |
| `medication_doses` | doses given | medication, time, amount text |
| `reminders` | materialised reminder occurrences (11) | medication, due time, state, attempts |
| `push_subscriptions`, `push_kid_optins` | Web Push (11) | parent, device label, endpoint, keys; subscription, kid |
| `ntfy_topics`, `ntfy_kid_optins` | ntfy (11.1) | parent (one row each), topic; parent, kid |
| `meta` | schema version, admin fingerprint (amended 2026-11-24: no fingerprint, 5.5) | key, value |

BMI is never stored; it is computed from a weight and a length/height measured on the same date.

A running sleep or nursing timer is a row with a null end, so it is shared: every parent with access
sees it, can switch the nursing side and can end it, from any device **(call)**. A kid has at most one
running row per table.

### 6.4 Querying by time range

- `GET /v1/kids/{kid}/timeline?from=&to=&types=` returns entries of the chosen trackers merged in
  time order, paged by cursor. Each tracker also has its own list endpoint with the same `from`/`to`.
- Each tracker table is indexed on `(kid_id, <its time column>)`.
- Daily summaries (bottle volume, nursing minutes, sleep hours, diaper counts per day) are computed
  by the API over a range, using the kid's time zone for day boundaries.

## 7. Age-adaptive experience

### 7.1 Age

Age is computed from the birth date in the kid's time zone. For a kid born before 37 weeks'
gestation, growth and milestones use **corrected age** (age minus the weeks short of 40) until the
second birthday **(call; the builder checks this against the guidance published with each growth
source and follows the source where it differs)**. Vaccination always uses chronological age. Wherever
corrected age is used the screen shows both ages.

### 7.2 Modules

A module is one tracker. Each has a default by age band, and a per-kid override that a guardian can
set either way at any time **(owner)**. Turning a module off hides its entry points and its
reminders; its data stays and reappears when it is turned on.

| Module | 0–11 months | 12–23 months | 24 months and over |
|---|---|---|---|
| Bottles, nursing, pumping | on | on | off |
| Diapers | on | on | off |
| Sleep | on | on | off |
| Feeding (solids) | on | on | off |
| Milestones | on | on | off |
| Growth | on | on | on |
| Vaccination | on | on | on |
| Sickness | on | on | on |
| Medication | on | on | on |
| Medical visits (amended 2026-11-25) | on | on | on |

The 24-months column is the owner's decision; the split of the first two is the home-screen order
below, not a difference in what is on.

Defaults are evaluated live from the kid's age, with one exception: when a kid crosses into a band
where a module's default is off, a module that has an entry in the previous 30 days, or a running
timer, gets an explicit "on" override instead of disappearing **(call)**. A guardian turns it off
when it is no longer wanted.

### 7.3 Home screen

One home screen per kid, with a kid switcher at the top. It shows, in order: quick-log buttons, what
is running now (a sleep or nursing timer), what is due (medication, vaccines), and today's summary.
The quick-log buttons are the enabled modules in a fixed order per age band **(call)**:

- 0–11 months: bottle, nursing, diaper, sleep, then the rest.
- 12–23 months: feeding, bottle, sleep, diaper, then the rest.
- 24 months and over: sickness, medication, growth, vaccination.

Phone width (the layout every page is designed at first):

```
┌──────────────────────────────┐
│ Mia ▾        4 months 12 days│
├──────────────────────────────┤
│ [ Bottle ] [ Nursing ]       │
│ [ Diaper ] [ Sleep   ]       │
│ [ More…                    ] │
├──────────────────────────────┤
│ ● Asleep since 13:05   [End] │
├──────────────────────────────┤
│ Due                          │
│  Vitamin D drops     14:00   │
│  4-month vaccines   overdue  │
├──────────────────────────────┤
│ Today                        │
│  Bottles   5 · 22 oz         │
│  Nursing   3 · 41 min        │
│  Diapers   6 (4 wet, 2 dirty)│
│  Sleep     9 h 20 min        │
├──────────────────────────────┤
│ Home  Timeline  Growth  More │
└──────────────────────────────┘
```

On a wide screen the bottom bar becomes a side navigation and the sections sit in two columns. Both
layouts are the same routes and the same components; a page is not done until it has been checked at
phone and desktop width **(owner: no functionality gaps)**.

## 8. Growth

### 8.1 Reference sources

| Source id | Covers | Status |
|---|---|---|
| `who` | WHO Child Growth Standards 2006 (0–5 years) and WHO Reference 2007 (5–19 years) | required |
| `cdc` | CDC 2000 growth charts (0–20 years) | required |
| `uk-who` | UK-WHO charts (RCPCH) | include if the tables can be obtained and their licence permits bundling **(owner: not a hard requirement)** |
| `euro-growth` | Euro-Growth references | same condition **(owner)** |

- Each source is a set of LMS tables by sex and age, for weight-for-age, length/height-for-age, head
  circumference-for-age and BMI-for-age, embedded in the API binary. Every table file records its
  origin URL, version, retrieval date and licence. A test checks each table against worked examples
  published by its source.
- A source that cannot be bundled is left out; its id is not offered. The builder reports which of
  the two optional sources made it in. The licence check applies to `who` and `cdc` too, against the
  licence and visibility of this repository and its images (both public, 15), so bundling a table is redistribution; if a required source
  cannot be bundled the builder stops and asks the owner.

### 8.2 Behaviour

- The kid's source is a setting. Nothing stored depends on it, so changing it re-draws every curve
  and percentile at once **(owner)**.
- For each measurement the API returns the z-score and percentile under the current source, using
  corrected age where 7.1 applies. For charts it returns the source's percentile curves for the kid's
  sex over the visible age range.
- Where a source has no data for a measure or an age (for example head circumference beyond its
  range), the chart shows the kid's points without curves and says the source does not cover it.
- Charts: weight, length/height, head circumference, BMI; each with the kid's points over the
  percentile curves.

## 9. Vaccination

- Schemas at launch: United States, Mexico, United Kingdom, Switzerland **(owner)**. Each is a data
  file embedded in the API: for each recommended dose, the antigens it covers, its dose number and
  its recommended age window. Each file records the issuing authority, the schedule's edition and the
  date it was transcribed; the screen shows "Schedule: <authority>, <edition>".
- The schedule files are transcribed by the builder from the authorities' current published
  schedules and **reviewed by the owner in the pull request** before release **(call)**.
- An administered dose is stored per antigen: one administration (date, optional lot, optional
  product name) writes one row for each antigen the vaccine covers. Because doses are counted per
  antigen and not per schedule line, switching the kid's schema keeps every recorded dose and only
  changes what is shown as due **(owner: changeable at any time)**.
- A schema file lists, per antigen, a numbered series of doses with windows. Matching is by count
  only: the kid's Nth recorded dose of an antigen fills the Nth line of that antigen's series, whenever
  it was given. The app never judges whether a dose was valid, too early or too late **(call)**.
- Where a series length depends on the product (rotavirus, Hib), the file lists the longest series and
  marks the product-dependent lines "may not be needed". Recurring doses (seasonal influenza) are
  listed as recurring and are never due or overdue **(call)**.
- For each schedule dose the screen shows one of: given (with date), due (inside its window), upcoming,
  **overdue** (past its window and not given), or dismissed. Overdue is a soft flag: a label and a
  place in the home screen's "Due" list. No push, no blocking **(owner: soft flag)**.
- A guardian can dismiss any schedule line ("not needed" or "given, date unknown") and undo it, and
  can dismiss every line before a chosen date in one action, so a kid added at age four does not start
  with a wall of overdue doses. Dismissals are stored per kid and schema line (`vaccine_dismissals`)
  and survive a schema switch only for the schema they were made in **(call)**.
- A dose not in the kid's schema (travel vaccine, another country's product) can be recorded and is
  listed under "Other".
- The vaccination module can be turned off per kid like any other **(owner)**.

## 10. Sickness, medication, milestones

### 10.1 Milestones

A built-in catalog of common developmental milestones by age, taken from the CDC's public-domain
milestone lists **(call)**, plus free-text custom milestones. A guardian or caregiver marks one as
reached with a date. The screen shows the milestones around the kid's (corrected) age; it never says
a kid is late.

### 10.2 Sickness

Each entry is one symptom at a time: congestion, cough, fever, vomiting, diarrhea, constipation,
rash, ear pain, sore throat, poor appetite, unusually sleepy, other **(call beyond the owner's
"congestion, temperature, diarrhea, constipation, etc.")**. Optional severity (mild, moderate,
severe). `fever` carries a temperature (6.2) and how it was taken. There is no "illness episode"
object; the timeline filtered to sickness shows the course **(call)**.

### 10.3 Medication

- A medication is a course: name, dose as text ("2.5 ml"), start, optional end, and a schedule that
  is one of: none (as needed), fixed times of day, or every N hours from a start time **(call)**.
- "Every N hours" is a fixed grid from the start time. It is not re-anchored to the dose actually
  logged: the app does not compute a safe interval (non-goals). Instead the "next due" line shows
  when the last dose was logged, so the parent decides **(owner: do not recompute)**.
- A dose given is logged against the medication with a time. Logging is always possible, scheduled or
  not.
- The medication screen shows each active course with its last dose and its next due time.

## 11. Reminders, Web Push and ntfy

- The API holds a VAPID key pair (the private key from SOPS; the public key is derived and served to
  the UI) and a VAPID subject, which is an `https:` URL **(owner)**: the app's own origin **(call)**.
  Web Push is optional: both VAPID variables set, or neither (Web Push off). When they are set, the API
  refuses to start unless the subject is an `https:` URL **(owner, 2026-11-23)**. `davcs86/xstockstrat` already
  sends Web Push this way (`services/xstockstrat-notify/src/fanout/webPush.ts`); its push TTL of one
  hour and 10-second send timeout are the starting values here **(call)**.
- A parent turns notifications on **per device**, and on that device chooses the kids to be reminded
  about **(owner)**. Only a parent with a grant on the kid can opt in; revoking the grant removes the
  opt-in.
- The worker ticks every 30 seconds. For each active scheduled medication of a kid whose medication
  module is on, it materialises the next occurrence as a `reminders` row, unique on (medication, due
  time), so a restart or a second tick never creates or sends a duplicate.
- A due reminder is sent to every opted-in Web Push subscription for that kid, and to every
  opted-in ntfy topic when ntfy is configured (11.1). Each target is delivered and retried on its own;
  a failing channel never delays or blocks the other. The Web Push payload is encrypted to the
  device (RFC 8291). The notification is generic: "<Category> due for <kid's name>", for example
  "Medication due for Mia", and never names the medication or the dose **(owner)**. Medication is the
  only category in stage one; the payload carries the category so others can follow. It opens the
  medication screen.
- Delivery: a 404 or 410 from the push service deletes that subscription; a 429 or 5xx is retried
  with backoff; a reminder not sent within 30 minutes of its due time is marked `expired` and never
  sent late **(call)**. A dose logged for that medication from 30 minutes before the due time until
  the next occurrence marks that occurrence done; a dose logged before the push goes out cancels it
  **(call)**. The home screen's "Due" list and each course's "next due" read the same `reminders`
  rows, so the list and the pushes never disagree. A missed dose stays `overdue` on the Due list until
  its due time plus the kid's `reminder_overdue_minutes` (a guardian's kid setting, 5 to 1440, default
  60) or the course's next occurrence, whichever comes first **(owner, 2026-11-23)**.
- On iOS the settings screen explains that notifications need the app added to the home screen.
- The service worker handles install, push and notificationclick only, with no fetch handler **(owner,
  2026-11-23)**. It caches no kid data **(owner: online-only)**. The offline page planned here is
  deliberately not built (owner, 2026-11-23).

### 11.1 ntfy (optional second channel)

- ntfy is on when `NTFY_URL` is set, and absent from the UI and the worker when it is not. Web Push
  does not depend on it, and is on only when both VAPID variables are set **(owner, 2026-11-23,
  superseding "always on")**.
- One topic **per parent** **(owner)**. When a parent turns ntfy on in account settings, the API
  generates `<NTFY_TOPIC_PREFIX>-<22 random base62 characters>` **(call)**, stores it, and the screen
  shows the server address (`NTFY_PUBLIC_URL`, or `NTFY_URL` when that is unset) and the topic to
  subscribe to in the ntfy app. The parent can regenerate the topic (the old one stops receiving) or
  turn ntfy off (the topic is deleted).
- The parent chooses the kids to be reminded about on that topic, as for a Web Push device. Only a
  parent with a grant on the kid can opt in; revoking the grant removes the opt-in. So a reminder for
  a kid only ever goes to topics of parents who can see that kid.
- The message is the same generic text as the push, "<Category> due for <kid's name>", with the app's
  medication screen as its click address. The API publishes with `Authorization: Bearer <NTFY_TOKEN>`
  when a token is set and with no credentials when it is not.
- Delivery: a 429 or 5xx is retried with the same backoff and the same 30-minute expiry as a push. A
  401 or 403 is logged once per tick as a configuration fault (no topic in the log) and is not
  retried for that occurrence. The topic is a secret and is never logged.
- The app does not manage who may **read** a topic. On an open ntfy server the unguessable topic is
  the protection; on a server that denies by default, read access is the server operator's to grant.
- ntfy keeps a message in its cache (72 hours on the edge's server), so the kid's name rests there in
  plaintext for that long. The settings screen says so at the top of its ntfy section **(call)**.
- `GET /healthz/reminders` covers both channels: a delivery of a due reminder on either one still
  pending or sending 15 minutes after its due time turns it 503 (13.1).

## 12. PDF export

A guardian or caregiver exports one kid for a date range, choosing which modules to include. The API
renders the PDF itself with a pure-Go library (no browser, no CGO) and streams it through the UI. It
contains the kid's profile, growth table and charts with percentiles under the current source, the
vaccination record, and each chosen tracker's entries and daily summaries, in the requesting
parent's units **(call on contents; the owner asked for a PDF export)**.

## 13. Processes and configuration

### 13.1 API (`api/`, image `ghcr.io/davcs86/happybabytracker-api`)

One Go binary, one process, **one listener** on container port 9210, never published to the host. The same binary's
`admin <verb>` runs as a second, short-lived process against the same database (amended 2026-11-24, 5.5). It
serves `/v1/*` to the UI container only and `GET /healthz`. Every `/v1/*` request must carry the
shared `INTERNAL_API_TOKEN`; requests acting for a person also carry that person's session token.

| Env var | Meaning |
|---|---|
| `ADMIN_USERNAME`, `ADMIN_PASSWORD` | the seeded admin (secret; password at least 16 characters). Amended 2026-11-24: validated after the store opens, required only while `admins` is empty, then ignored with a warning (5.5) |
| `AUTH_MODE` | amended 2026-11-24: `local` (default) \| `oidc` \| `both` (5.5) |
| `INTERNAL_API_TOKEN` | shared with the UI container (secret, at least 32 characters) |
| `VAPID_PRIVATE_KEY` | optional; P-256 private key for Web Push (secret) |
| `VAPID_SUBJECT` | optional; the `https:` subject sent to the push services; on the edge, the app's origin |
| `PUBLIC_ORIGIN` | the app's origin, used in invite and reset links and as the push click target |
| `DATA_DIR` | SQLite directory, default `/data`, the only writable path |
| `HBT_ALLOW_FAKE_PUSH` | `1` sends pushes to an in-memory recorder (tests and smoke only) |
| `NTFY_URL` | optional; the ntfy server the API publishes to. Unset turns ntfy off |
| `NTFY_TOKEN` | optional ntfy access token (secret) |
| `NTFY_TOPIC_PREFIX` | required with `NTFY_URL`; 1–32 of `a-z0-9-` |
| `NTFY_PUBLIC_URL` | optional; the server address shown to parents when it differs from `NTFY_URL` |

The process refuses to start on a missing or too-short secret, an invalid VAPID key or an invalid
origin. The VAPID variables are optional, both or neither (neither turns Web Push off) **(owner,
2026-11-23, superseding "the VAPID variables are always required")**; one of them without the other,
an `HBT_ALLOW_FAKE_PUSH` other than `1`, or `HBT_ALLOW_FAKE_PUSH=1` without them, is refused. It also refuses `NTFY_URL` without a
prefix, a prefix outside 1–32 of `a-z0-9-`, a malformed `NTFY_URL` or `NTFY_PUBLIC_URL`, an `NTFY_TOKEN`
with a space, control or non-ASCII character, and any other `NTFY_*` variable set while `NTFY_URL` is not.
Amended 2026-11-24 (5.5): it also refuses an `AUTH_MODE` other than `local`, `oidc` or `both`, and, while `admins` is
empty, a missing or invalid `ADMIN_USERNAME` or `ADMIN_PASSWORD`, checked after migrations rather than with the other
variables.

Health: `GET /healthz` is 200 while the database answers and the worker has ticked within 60 seconds.
`GET /healthz/reminders` is 503 when a delivery of a due reminder is still pending or sending 15 minutes
after its due time, or a delivery failed or expired within the last 15 minutes **(call)**; with no channel
configured it is 200.

### 13.2 UI (`ui/`, image `ghcr.io/davcs86/happybabytracker-ui`)

SvelteKit with the Node adapter on container port 9212, the same stack and versions policy as
commsgun's dashboard. The browser talks only to this server. It holds no data and no secret other
than `INTERNAL_API_TOKEN` (amended 2026-11-24: and the OIDC client secret and transaction-cookie key, 5.5).

| Env var | Meaning |
|---|---|
| `INTERNAL_API_TOKEN` | same value as the API's |
| `API_BASE_URL` | `http://happybabytracker-api:9210` on the edge |
| `ORIGIN` | the app's origin; a request whose derived origin differs gets 400, except the health paths. Amended 2026-11-24: `https://happybaby.public.<domain>`, and `PUBLIC_ORIGIN` follows |
| `OIDC_ISSUER`, `OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET` (secret) | amended 2026-11-24: the relying party (5.5); all three or none |
| `OIDC_SCOPES` | amended 2026-11-24: optional, default `openid profile` (5.5) |
| `SESSION_SEAL_KEY` | amended 2026-11-24: the key the OIDC transaction cookie is encrypted with (secret, at least 32 characters); required with `OIDC_ISSUER` (5.5) |
| `OIDC_AUTO_PROVISION` | amended 2026-11-24: not implemented; any value is refused (5.5) |
| `PORT` | 9212 |

The image sets `PROTOCOL_HEADER`, `HOST_HEADER`, `ADDRESS_HEADER` and `XFF_DEPTH` as commsgun's does.
Amended 2026-11-24 (5.5): the UI refuses to start on a partial relying-party set, on `SESSION_SEAL_KEY` missing or
shorter than 32 characters while `OIDC_ISSUER` is set, and on `OIDC_AUTO_PROVISION` set. It never contacts the issuer at start.
Health: `GET /healthz` (the UI itself), `GET /healthz/api` and `GET /healthz/reminders` (the API's
two answers passed through), so Gatus can probe everything through the one published name.

Every page that shows kid data is sent `Cache-Control: no-store`.

### 13.3 Logs

JSON lines. They carry ids (kid, parent, entry, request, reminder) and enumerated facts (route,
tracker, role, status, outcome, durations). Never a name, a username, a note, a health value, a
medication name, a push endpoint or a secret. URLs carry ids and `from`/`to`/`types` only, plus, on the
export page's validation redirect, `error` (a field name or `busy`) and `modules` (enumerated module ids,
never a value or a name), so the edge's Caddy access log holds nothing else either. Amended 2026-11-24: except the
invite and reset tokens in their paths and the callback's query, which the edge redacts and drops (5.5, 18).

## 14. Build and release (this repo)

Layout as commsgun: `api/`, `ui/`, one `Dockerfile` in each, `scripts/smoke.sh`,
`.github/workflows/ci.yml`, `CLAUDE.md`, `README.md`. Base images pinned by digest, actions pinned by
SHA, uid 10001, read-only root.

CI on every PR and on `main`: the API pipeline (`gofmt`, `go vet`, staticcheck, `go test -race`), the
UI pipeline (type-check, lint, unit tests, Playwright with axe at phone and desktop widths), both
image builds, then `scripts/smoke.sh`. On `main` only: push both images as `:<sha>` and `main`, label
both with `org.opencontainers.image.revision`, print both digests. Merging deploys nothing.

Tests the builder must include:

- **The access matrix.** For every kid-scoped route, generated from the route table so a new route
  without a row fails: guardian, caregiver, a parent with no grant (404), the admin (refused), no
  session (401).
- The last-guardian rules of 5.2, each inside its transaction, including the admin's disable.
- ~~The admin fingerprint: a changed username or password at start ends admin sessions.~~ Amended 2026-11-24: the
  admin import and the rest of 5.5's tests list replace it.
- Unit conversion round trips, and that an edit recomputes the derived column.
- Each growth table against its source's published examples; corrected age on and off; a source
  change altering percentiles without touching stored rows.
- Vaccination: a schema switch keeps every dose; a combination vaccine counts for each antigen;
  overdue at the window edge.
- Reminders: no duplicate after a restart mid-send; subscription removal on 410; expiry after 30
  minutes; a disabled module or a revoked grant stops reminders.
- Day boundaries and reminder times across a daylight-saving change in the kid's time zone.
- Logs contain no name, username, note or value.
- Smoke: create a parent through an invite, create a kid, log one entry of each tracker, read the
  timeline, export a PDF, deliver a fake push, check memory ceilings (60 MB API, 140 MB UI **(call)**).

## 15. Deployment (`davcs86/homelab-edge`)

Everything follows that repo's `CLAUDE.md`.

**Compose**: two services under the hardening anchor, both `read_only`, on their own network.

| Service | Image var | Ports | Volume | `mem_limit` |
|---|---|---|---|---|
| `happybabytracker-api` | `happybabytracker_api_image` | none published | `happybabytracker-data:/data` | 96m **(call)** |
| `happybabytracker-ui` | `happybabytracker_ui_image` | `127.0.0.1:9212:9212` | none | 192m **(call)** |

Amended 2026-11-24 (5.5, not deployed): the API's env gains `AUTH_MODE`; the UI's gains the `OIDC_*` settings and
`SESSION_SEAL_KEY` of 13.2, and the UI service gains `extra_hosts: ["auth.public.<domain>:host-gateway"]` (SSO spec
§ 4.2), its only path to the issuer.

**Image visibility**: the repository is public **(owner)** and both GHCR packages are made public
once, as commsgun's were, so the edge and the bump workflow pull without a credential.

**Name** (amended 2026-11-24, not deployed: the UI moves to `happybaby.public.<domain>`, the private name is removed
in its four EDGE-07 places, and the reachability paragraph below no longer applies; homelab-edge SSO spec § 5): one
`*.private` name for the UI, wired in the four places of EDGE-07, inheriting
`tailnet_only` and `deny_staging_vm`: `happybaby.private.<domain>` **(owner)**. Nothing under
`*.public`, nothing in `edge_sites`.

**Who can reach it** (amended 2026-11-24, not deployed: anyone on the internet reaches the login, and only members of
the IdP's `happybabytracker` group, or password holders in `local`/`both`, get past it; 18): only the tailnet users
that reach `*.private` today, `david@` and `david_admin@` **(owner)**. No policy or Caddy change. Other parents and caregivers can have accounts
but cannot reach the app until the Authelia work changes how it is exposed (5.4); that work decides
their network path.

**Secrets** (`secrets/edge.sops.yaml`, declared in `scripts/init_secrets.py`, documented in
`secrets/README.md`):

| Key | Source |
|---|---|
| `happybabytracker_admin_username` | owner-provided **(owner: only in SOPS)**. Amended 2026-11-24: ignored once the image with 5.5 has imported it; retired in a later edge change |
| `happybabytracker_admin_password` | generated; the owner may replace it with `sops set`. Amended 2026-11-24: as the username; from the bump on, rotation is the CLI's `reset-password` (5.5) |
| `happybabytracker_internal_api_token` | generated |
| `happybabytracker_vapid_private_key` | generated once (a P-256 key in a form with no quote, backslash or line break). Never rotated casually: a new key invalidates every push subscription |
| `happybabytracker_ntfy_token`, `happybabytracker_ntfy_password_bcrypt` | generated, only when ntfy is enabled for the app (below) |
| `happybabytracker_oidc_client_secret`, `happybabytracker_oidc_client_secret_pbkdf2` | amended 2026-11-24: generated once, the hash for Authelia's client config (SSO spec § 4.7); the secret goes to the UI as `OIDC_CLIENT_SECRET` **(call: key names)** |
| `happybabytracker_session_seal_key` | amended 2026-11-24: generated, at least 32 characters; the UI's `SESSION_SEAL_KEY`. Rotating it only fails SSO attempts in flight **(call: key name)** |

The deploy asserts presence, shape and minimum length before it writes any file, as it does for
commsgun.

**Backup**: the existing `edge-backup` script also snapshots this database (`.backup`, then
`PRAGMA integrity_check`) into the same `age`-encrypted archive, the restore drill opens it, and the
timer runs at 00:00 and 12:00 UTC **(owner)**, which also moves the headscale snapshot from its
single 03:17 UTC run. The
existing heartbeat endpoint keeps its name (EDGE-10) and its interval is shortened to match
**(call: one archive and one timer, rather than a second unit)**. `docs/runbooks/recovery.md` gains
the restore steps for this database.

**Monitoring**: Gatus probes for `/healthz`, `/healthz/api` and `/healthz/reminders` through the
private name, in the `edge` group, with ntfy alerts. Amended 2026-11-24: through the public name (SSO spec § 4.9).

**Pins and bumps**: both image vars in `versions.yml` with rows in `docs/versions.md`, and a
`bump-happybabytracker` workflow modelled on `bump-commsgun` that pins both in one PR and refuses a
pair whose revision labels differ.

**ntfy (the platform admin's choice)**: one inventory switch, `happybabytracker_ntfy_enabled`
**(owner: configurable on the edge)**. Off: no `NTFY_*` variable is rendered and the app runs on Web
Push alone. On, the edge PR adds:

- an ntfy user `happybabytracker` with write-only access to `<prefix>-*` and a token, declared in
  `ntfy-server.yml.j2` beside `alerts`; the prefix is an inventory value;
- `NTFY_URL` pointing at the ntfy container over a compose network the API joins, `NTFY_PUBLIC_URL` =
  `https://{{ ntfy_private_hostname }}` (ntfy's `base-url`, the only name an iOS subscription works
  under), `NTFY_TOKEN` and `NTFY_TOPIC_PREFIX` in the API's env file;
- nothing for readers: `david` already reads every topic as ntfy admin. Any other parent needs an ntfy
  user with read access to their own topic, added by hand in this repo when that day comes.

**Policy**: no change.

**Docs**: a runbook (first deploy, add a parent, reset a password, rotate the internal token, restore
the database, what a VAPID key change costs), the Context Guide row in `CLAUDE.md`, one entry in
`docs/decisions.md`, `secrets/README.md`. Amended 2026-11-24 (5.5): before the bump that brings the SSO image is
merged, the runbook's admin rotation moves to the CLI's `reset-password`; it gains the CLI verbs, switching
`AUTH_MODE`, linking and unlinking (the admin's own through `unlink-sso`), and the cutover step below.

**Origin move** (amended 2026-11-24, not deployed): installed PWAs, push subscriptions and the push-device cookie
belong to the old origin. Old subscriptions keep delivering, and enabling notifications on the new origin adds a second
device row, so the same device would get each reminder twice. Cutover step **(call)**: each parent opens Account →
Notifications on the new origin, enables push, deletes the old device's row, and removes the old installed app.

## 16. Order of work

One design, several plans, each its own PR, each leaving something that works:

1. **Foundation** (this repo): repo skeleton, CI, both images, admin and parent login, invites, kids,
   per-kid access, units preferences, the access-matrix test, smoke.
2. **Edge** (`homelab-edge`): compose, name, secrets, backup, Gatus, bump workflow, runbook. The app
   is live and backed up before it holds anything that matters.
3. **Daily trackers**: bottles, nursing, pumping, sleep, diapers, feeding; timeline; modules; the
   age-adaptive home screen; PWA manifest and install.
4. **Growth**: measurements, reference tables, charts, corrected age.
5. **Vaccination**: schedule files, doses, due and overdue.
6. **Sickness and medication**: symptoms, courses, doses, the reminder worker, Web Push, the ntfy
   channel; then the edge's ntfy switch.
7. **Milestones and PDF export.**

Each plan after the second ends with: merge, the bump PR, a deploy, and a check on the edge.

## 17. Left to the builder

- Go package layout, router, migration mechanism, lint configuration; exact SQL and indexes beyond
  6.4.
- The JSON shapes between the UI and the API. They ship from one commit and are not a contract with
  another repository. The contract with `homelab-edge` is section 13 (ports, env var names, health
  paths, the data path) and the image names.
- Component composition and visual detail within shadcn-svelte's defaults, provided each page works
  at phone and desktop width and the home screen keeps the content and order of 7.3.
- The PDF library and layout; the Web Push library; the argon2id parameters (current OWASP guidance).
- Exact pinned versions (newest stable, at least two weeks old, pinned by digest or lockfile),
  matching commsgun's where the same tool is used.
- The percentile lines each chart draws, following each source's own convention.
- The milestone catalog's exact items and the vaccine schedule files, from the named authorities,
  subject to the owner's review in 9.

## 18. Threat model (draft, pending owner review before P6)

Required by the homelab-edge SSO spec § 3 before the app goes public at `happybaby.public.<domain>`.

| Surface | Exposure | Auth | Limits | Secrets in URLs and logs | Session | CSRF |
|---|---|---|---|---|---|---|
| UI pages, form actions and server endpoints (`/logout`, `/account/notifications/subscribe`) | internet, through Caddy | the session cookie, reached by SSO or, in `local`/`both`, a password | Caddy per-IP zone for the site | ids only (13.3) | opaque, server-side; parent 30 days sliding, admin 12 h; deleted on logout, disable, reset, unlink | SvelteKit origin check (`ORIGIN`, 13.2); `SameSite=Strict` session cookie |
| Password routes (5.5): `POST /login`, `/invite/{token}`, `/reset/{token}`, `/account` (`?/password`), `/account/link-sso`, `/admin/account/link-sso` | internet | a password; refused by mode as 5.5's table says: in `oidc` no invite or reset link is redeemed by password (403 `mode_refused`), so none yields a session without SSO **(owner, 2026-11-24)** | Caddy `<site>_login` and `<site>_login_hourly`; the 5.3 limiter per address and per username (login, re-auth, password change) or per address (redemption) | invite and reset tokens in the path: Caddy redacts them on the public site (SSO spec § 4.3), OTEL scrubs them (Rule 5); query, `Referer` and `Location` dropped; `Referrer-Policy: no-referrer` | a redemption issues a parent session | origin check; `SameSite=Strict` |
| SSO starts: `POST /auth/sso`, `POST /invite/{token}?/sso` | internet | none, or the invite token | per-IP zone; the invite start also in the login zones | the token stays in its path; the transaction cookie is encrypted | `hbt_oidc`: 10 min, `Lax`, `Path=/auth` | origin check (form POST) |
| `GET /auth/callback` | internet | `state` bound to the transaction cookie; ID-token checks (`alg`, `iss`, `aud`, `exp`, `nonce`) | per-IP zone | `code`/`state` in the query: dropped from Caddy logs, scrubbed from OTEL | answers 200, never a redirect | `state` + PKCE + `nonce`; purpose and binding in the cookie; `prompt=login` for link and invite |
| `POST /auth/confirm` | internet | the pending cookie; for a link, the session whose hash it binds plus a re-auth within 5 minutes (API); for an invite, the token | per-IP zone | none | an invite issues a parent session | origin check; same-origin POST |
| API 9210 | compose network only, never published | `INTERNAL_API_TOKEN` + session token | the 5.3 limiter | `/v1/tokens/{token}`, `…/redeem` and `…/redeem-oidc` carry the token in the path; the API logs route patterns only (`middleware.go`) | — | not browser-reachable |
| Health paths | internet, unauthenticated | none | per-IP zone | none | — | — |

Residuals for the owner to accept or reject:

1. While `both` is on, a password login is reachable from the internet (SSO spec decision 18).
2. An offboarded parent keeps their live sessions (sliding for 30 days, 5.3), any password login in `both`, and their
   medication reminders (a kid's name and a category) until the admin disables or unlinks them (5.5; homelab-edge
   `sso.md` offboarding); ntfy keeps a sent message in its cache for 72 hours (11.1).
3. A parent who joined by SSO has no password, so while the issuer is down they cannot sign in until the operator
   switches the app to `both` and the admin sends them a link (an invite, since they have no password) that they redeem
   with a password. In `oidc` no link can be redeemed by password (5.5).
4. The `authelia_session` cookie is scoped to `public.<domain>`; the edge strips it before this upstream (SSO spec
   § 4.3). If that strip is missing, the app still ignores it.
5. Kid data now crosses the internet to the browser, over TLS only; `Cache-Control: no-store` (13.2) stays on every
   such page.
6. In `oidc` with the issuer down nobody signs in, the admin included, until the operator switches the app to `both`
   (the API's env and a deploy) **(call)**.
