# happybabytracker

A self-hosted health record for babies and kids. The record belongs to the child: an admin creates
parent accounts and sends each parent a one-time invite link; a parent creates a kid and shares it with
other parents as a guardian or a caregiver. The admin manages accounts only and never sees a kid. This
first release is the foundation: accounts, invites and resets, kids and per-kid access.

Two images, built from this repo:

| Image | Source | Port |
|---|---|---|
| `ghcr.io/davcs86/happybabytracker-api` | `api/` (Go, SQLite) | 9210, internal only, never published |
| `ghcr.io/davcs86/happybabytracker-ui` | `ui/` (SvelteKit, Node adapter) | 9212 |

The browser talks only to the UI; the UI calls the API with a shared token. Both run as uid 10001 on a
read-only root filesystem; the API writes only to `/data`. Design:
`docs/superpowers/specs/2026-11-20-happybabytracker-design.md`; plans: `docs/superpowers/plans/`.

## Configuration

API (`ghcr.io/davcs86/happybabytracker-api`):

| Env var | Meaning |
|---|---|
| `ADMIN_USERNAME` | The admin's login name, `^[a-z0-9._-]{3,32}$` |
| `ADMIN_PASSWORD` | The admin's password, at least 16 characters (secret) |
| `INTERNAL_API_TOKEN` | Shared with the UI, at least 32 characters (secret) |
| `PUBLIC_ORIGIN` | The app's origin, `http(s)://host[:port]` with no path; used in invite and reset links |
| `DATA_DIR` | SQLite directory, default `/data` (the only writable path) |

The process refuses to start on a missing or invalid value (only `DATA_DIR` has a default) and names
the variable, never its value. It listens on `:9210`.
Changing `ADMIN_USERNAME`, `ADMIN_PASSWORD` or `INTERNAL_API_TOKEN` ends every admin session at the next
start; parent sessions are kept.

UI (`ghcr.io/davcs86/happybabytracker-ui`):

| Env var | Meaning |
|---|---|
| `INTERNAL_API_TOKEN` | Same value as the API's |
| `API_BASE_URL` | The API, for example `http://happybabytracker-api:9210` |
| `ORIGIN` | The app's origin; a request whose derived origin differs gets 400, except `/healthz*` |
| `PORT` | 9212 (set by the image) |
| `PROTOCOL_HEADER`, `HOST_HEADER`, `ADDRESS_HEADER`, `XFF_DEPTH` | Set by the image to `x-forwarded-proto`, `x-forwarded-host`, `x-forwarded-for` and `1` |

Because of the last row, the origin and client address come from `X-Forwarded-Proto`,
`X-Forwarded-Host` and `X-Forwarded-For`: put the UI behind a reverse proxy that sends all three. The
session cookie is `Secure`, so the public origin must be `https`.

The `VAPID_*` and `NTFY_*` variables of the design arrive with the reminders plan; nothing reads them yet.

Health, all through the UI: `GET /healthz` (the UI), `GET /healthz/api` (the API's `/healthz`, 200 while
its database answers).

## Run locally

With Docker, on a private network (the API port stays unpublished):

```sh
docker build -t hbt-api:dev api && docker build -t hbt-ui:dev ui
TOKEN="$(head -c 24 /dev/urandom | od -An -tx1 | tr -d ' \n')"
docker network create hbt
docker run -d --name hbt-api --network hbt --read-only -v hbt-data:/data \
  -e ADMIN_USERNAME=admin -e ADMIN_PASSWORD='choose-16-or-more' \
  -e INTERNAL_API_TOKEN="$TOKEN" -e PUBLIC_ORIGIN=https://localhost:9212 hbt-api:dev
docker run -d --name hbt-ui --network hbt --read-only -p 127.0.0.1:9212:9212 \
  -e API_BASE_URL=http://hbt-api:9210 -e ORIGIN=https://localhost:9212 \
  -e INTERNAL_API_TOKEN="$TOKEN" hbt-ui:dev
```

The UI then expects a TLS-terminating proxy in front of `127.0.0.1:9212` that sends the three
`X-Forwarded-*` headers. `scripts/smoke.sh hbt-api:dev hbt-ui:dev` runs both images the same way and
plays that proxy with `curl`.

Without Docker, the API with `go run` and the UI with the Vite dev server:

```sh
cd api && ADMIN_USERNAME=admin ADMIN_PASSWORD='choose-16-or-more' \
  INTERNAL_API_TOKEN="$TOKEN" PUBLIC_ORIGIN=http://localhost:5173 DATA_DIR="$(mktemp -d)" \
  go run ./cmd/happybabytracker-api
cd ui && pnpm install && INTERNAL_API_TOKEN="$TOKEN" API_BASE_URL=http://localhost:9210 \
  ORIGIN=http://localhost:5173 PORT=5173 pnpm dev
```

and open `http://localhost:5173` (browsers accept the `Secure` cookie on `localhost`).

## Test

```sh
cd api && gofmt -l . && go vet ./... && \
  GOTOOLCHAIN=$(go env GOVERSION) go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./... && \
  go test ./... -race
cd ui && pnpm install --frozen-lockfile && pnpm check && pnpm lint && pnpm test && pnpm test:e2e
docker build -t hbt-api:dev api && docker build -t hbt-ui:dev ui && \
  scripts/smoke.sh hbt-api:dev hbt-ui:dev
```

`gofmt -l` must print nothing. The UI's e2e runs against a stub API (it needs
`pnpm exec playwright install chromium`, or `PW_CHROMIUM_PATH` pointing at a Chromium); the smoke script
runs the real API and the real UI together and prints one `PASS:` line per assertion. CI
(`.github/workflows/ci.yml`) runs all three on every pull request and on `main`.

## Release

Merging to `main` deploys nothing. CI builds both images, smoke-tests them together, pushes
`ghcr.io/davcs86/happybabytracker-api` and `ghcr.io/davcs86/happybabytracker-ui` (`:<sha>`, plus the
moving tag `main`, labelled with `org.opencontainers.image.revision`) and prints both `@sha256:…`
references in the `images` job summary. The edge runs the digests pinned in `davcs86/homelab-edge`; a
change is live only after those pins are changed there and the edge is deployed.
