# Provider Lookup (NPI) Implementation Plan (plan 10)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status: written 2026-11-26 against `main` effd0b2; owner rulings below; building on branch `ccr-406c8e35-e8hvag`.**
The UI panel (Task 6) starts only after the owner approves the "Find a US provider (plan 10)" section of
`docs/plans/ui-redesign/pages/visits.md`. The feature ships **off**: nothing changes for a user until the owner's
edge change (below) sets `NPI_SEARCH=1`.

**Goal:** While adding or editing a visit, a parent can search the US national provider registry (NPPES) by name and
state, pick a result, and have the visit's provider, place and NPI filled in. The search is optional, behind a flag,
and never in the way of saving a visit.

**Architecture:** A new stdlib-only package `api/internal/npi` calls NPPES with a fixed host, a 4 s timeout, no
redirects, a 1 MiB body cap and at most 4 calls in flight. It keeps only records whose practice-location address
matches the search, and caches normalised queries for 10 minutes. Two `Parent` routes front it:
`GET /v1/providers` (is search on?) and `POST /v1/providers/search`, which is registered only under
`NPI_SEARCH=1`. Migration 0009 adds `visits.provider_npi`, a snapshot set when a result is picked. In the UI the
"Find a US provider" panel is a `<details>` inside the existing Add and Edit visit sheets. Its Search and "Use this
provider" buttons are `formaction`s on the visit form, so it works without JavaScript and keeps what was typed.

**Tech Stack:** Go 1.27 (stdlib `net/http`, `encoding/json`), SQLite via `modernc.org/sqlite`; SvelteKit / Svelte 5
runes, Tailwind 4, shadcn-svelte parts. No new Go or npm dependency.

**Spec:** `docs/superpowers/specs/2026-11-25-medical-visits-design.md` §10, §11, §13 item 2 ("the visits spec").
"main N" means `2026-11-20-happybabytracker-design.md` (§13.1 is amended here). UI: the plan 10 section of
`docs/plans/ui-redesign/pages/visits.md` (draft for owner approval), `AGENTS.md`. CLAUDE.md rules 1, 5 and 7
govern the contract, the logs and the limits.

## Owner rulings (2026-11-26, do not reopen)

1. **We filter, not NPPES.** NPPES's `state` and `city` filters match *any* address of a record. In a live test on
   2026-11-25, a "WA, Seattle" person search returned records whose LOCATION address was in MN and ID, and
   `address_purpose=LOCATION` did not change that. So the API keeps only records whose practice-location address
   (`address_purpose` `LOCATION`) matches the searched state, and also the city and the postal code when given
   (postal code: first 5 digits). It still asks NPPES for `limit=20`, so fewer may show. A user never sees an
   out-of-state provider for a state search.
2. The panel's look is a page-spec addition to `docs/plans/ui-redesign/pages/visits.md`, built only from existing
   parts and words. The owner approves it before any UI code (AGENTS.md rules 5, 6).
3. Plan, then build, in this session, as one PR. The edge change (`NPI_SEARCH` on the API, outbound HTTPS to
   `npiregistry.cms.hhs.gov`) is the owner's, in `davcs86/homelab-edge`. This plan says exactly what the edge needs.
   The feature stays off (flag unset) until then.

## Builder rulings (this plan; open to owner correction)

- **B1. Result count.** The response is `{ "results": [...], "more": bool }`. `results` is what survives the
  filter (0–20). `more` is true when NPPES returned the full 20, so more matches may exist. NPPES's
  `result_count` is never shown, because it counts records the filter drops. The panel says "There may be more…"
  when `more` is set.
- **B2. Errors and limits.** All of these answer 503 `unavailable`: a `{"Errors":[…]}` body (even at HTTP 200), a
  non-200 status, a 3xx (redirects are not followed), a body over 1 MiB, unparseable JSON, a timeout, and a fifth
  concurrent call. The in-flight check does not wait: when 4 calls are out, the next one fails at once. Timeouts log
  `timeout`. Everything else logs `upstream_error`, the in-flight cap included, because §10 fixes the six outcome
  values.
- **B3. Cache.** The key is `kind` and the validated fields, in body order, each trimmed, with inner spaces
  collapsed and lowercased, joined by `\x1f`. The entry holds the filtered `{results, more}`. Successful and empty
  answers are cached. Errors are not. At most 256 entries for 10 minutes; when full, expired entries go first,
  then the oldest one inserted. Order of checks: body validation (422), then the per-parent limiter (429; cache
  hits count), then the cache, then the upstream call.
- **B4. Validation.** Body keys: `kind`, `last_name`, `first_name`, `organization`, `state`, `city`, `postal_code`.
  The first invalid field is named in that order. `kind` is `person` or `organization`. A person needs
  `last_name`, may have `first_name`, and must not send `organization`. An organization needs `organization` and
  must not send either name. Names and city match `constants.NPINamePattern` (a letter, then 1–59 more letters,
  spaces, hyphens, apostrophes or periods, after trimming). A `*` is refused, because the client adds the wildcard.
  `state` is one of `constants.NPIStates` (the 50 states, DC, AS, GU, MP, PR, VI), case-insensitive.
  `postal_code` is exactly 5 digits. The organization rule is the spec's ("2 to 60 letters…"): a name with digits
  or `&` cannot be searched. The person types the part before it.
- **B5. Upstream query.** `version=2.1`, `enumeration_type` `NPI-1` (person) or `NPI-2` (organization),
  `last_name=<v>*` and `first_name=<v>*`, or `organization_name=<v>*`, `state`, `city` as given, `postal_code` as
  given (5 digits also match a 9-digit ZIP, as probed), `limit=20`. Terms go in the query string of the API's own
  outbound request. That request is never logged and never traced.
- **B6. Result mapping.** `npi` is `number`, a JSON string; a record whose number fails `npi.Valid` is dropped.
  `name` is `first_name last_name` for NPI-1 and `organization_name` for NPI-2. `credential` is `basic.credential`
  (NPI-1 only, else null). `taxonomy` is the `desc` of the taxonomy with `primary: true`, else null. `address` is
  the LOCATION address as `address_1[, address_2], City, ST 12345`. `phone` is that address's `telephone_number`,
  else null. NPPES sends names and addresses in capitals, so the API title-cases every word of the name and the
  address, splitting after a space, hyphen, apostrophe or period. State codes stay upper case. Records with
  `basic.status` other than `A` are dropped, and so are records whose `name` would exceed `VisitProviderMaxLen` or
  whose `address` would exceed `VisitLocationMaxLen` (with `address_2` removed first). The decoder's structs have no
  `authorized_official_*` fields, so those names cannot reach a result.
- **B7. `provider_npi` lives with its provider text, enforced by the API.** A POST or PATCH that changes `provider`
  and does not send `provider_npi` clears it. A body whose merged result has `provider_npi` but no `provider` is
  422 `provider_npi`. The UI's only part: the picked result posts `provider_npi` along with `provider_npi_for` (the
  provider text it filled in). The form reader sends `provider_npi` only while the provider text still equals
  `provider_npi_for` (trimmed, exact). The comparison is exact on purpose: any edit of the text drops the link,
  as §10 says.
- **B8. Logs.** The request's one log line (`guard`, as plan 9's B3) gains `npi_outcome` (one of the six values)
  and `npi_results` (the count sent, 0 on failure). The line's `duration_ms` is the duration. A 422 has no
  `npi_outcome`. Field names go in `constants.go`.
- **B9. Panel placement and offer.** The panel is a closed `<details>` placed right above "Doctor or clinic
  (optional)" in `VisitFields`, on the Add and Edit visit sheets only, not on the outcome sheet. It is rendered only
  when `GET /v1/providers` answers `{search:{enabled:true}}`. Any failure of that call counts as off.
- **B10. No-JS actions.** The panel's buttons are `formaction="?/find"` ("Search") and `formaction="?/pick"`
  ("Use this provider") on the visit form itself, so every typed visit value is posted with them. `find` returns
  `{ values, providers: { query, page | error } }` and opens the panel again. The result choice is a RadioGroup
  `npi_choice` (value = index). Each result row carries hidden fields `npi_r<i>_npi`, `_name`, `_credential` and
  `_address`. `pick` sets `provider` to `name` (plus `, <credential>` when that fits in 120 characters),
  `location` to `address`, `provider_npi`, and `provider_npi_for`, and closes the panel. Nothing is saved until
  Save. A tampered hidden NPI is refused on save (422 `provider_npi`).
- **B11. The sheet stays open, and Enter still saves.** `savedAs` gains an option `inPlace: string[]` of action
  names: their `success` results are applied like a failure (`update({ reset: false })`, no close, no toast).
  Both pages pass `inPlace: ['find', 'pick']`. The form's first submit button in tree order is a visually hidden
  copy of Save (`sr-only`, `tabindex="-1"`, `aria-hidden="true"`, `formaction="?/save"`), so implicit submission
  (Enter) saves the visit, as it does today. With JavaScript, Enter inside a panel field clicks Search instead.
- **B12. Stub.** The stub defaults to search **off**, as the edge does, so every existing visits e2e and baseline is
  unchanged. New scenarios: `npi_on` (enabled, fixtures), `npi_unavailable` (enabled, search 503) and
  `npi_rate_limited` (enabled, search 429). While off, the stub's `POST /v1/providers/search` answers its
  unknown-route 404. The fixtures are fictional people and clinics. Their NPIs start with `9`, so none can be a real
  provider's (issued NPIs start with 1 or 2). They pass the check digit, and one of them has a MAILING address in
  WA and a LOCATION in OR. The stub applies the same filter, so the UI can show the shapes. The Go tests are the
  filter's real proof.

## Live facts (NPPES, probed from this environment)

| Fact | Probed | Used by |
|---|---|---|
| `GET https://npiregistry.cms.hhs.gov/api/?version=2.1&…` → 200 `application/json` `{result_count, results:[{number, enumeration_type, basic{…}, addresses[{address_purpose, address_1, address_2?, city, state, postal_code, telephone_number}], taxonomies[{code, desc, primary, state, license}], …}]}` | 2026-11-25 | 3 |
| Query errors come back as **HTTP 200** `{"Errors":[{"description","field","number"}]}` (e.g. state alone: "Field state requires additional search criteria") | 2026-11-25 | 3 |
| Trailing wildcard works (`last_name=smi*`). `state`/`city` match any address of a record (ruling 1) | 2026-11-25 | 3 |
| `number` is a JSON **string**; names and cities are upper case (`NICOLE`, `SEATTLE`); `postal_code=98101` matches `981012756`; `basic.status` is `"A"` | 2026-11-26 | 3 |
| NPI-2 records carry `authorized_official_*` names: third-party PII, never surfaced | 2026-11-25 | 3 |

## Main facts this plan relies on (checked at effd0b2)

| Artifact | Where | Used by |
|---|---|---|
| `Load(getenv)`; `loadPush`'s `HBT_ALLOW_FAKE_PUSH must be 1 or unset` (errors name the variable, never the value); `TestLoadVAPIDRefuses` table | `api/internal/config/config.go:35,71-74`; `config_test.go:169` | 4 |
| Migrations embedded, one tx each; 0008 creates `visits` `STRICT` with column `CHECK`s; `schema_version` `"8"` pins (:37 :55 :401 :550 :750 :857); `TestMigration0008KeepsRows` :999 | `store/migrations/0008_visits.sql`; `store/store_test.go` | 1 |
| `visitColumns`, `scanVisit`, INSERT (15 placeholders), `UpdateVisit(ctx, kidID, actorID, id, now, edit)` UPDATE | `store/visits.go:12,53,101,194,220` | 1 |
| `model.Visit{Entry; StartsAt; DurationMin; Kind; Provider, Location, OutcomeNotes *string; CancelledAt; Revision}` | `model/tracker.go:334` | 1 |
| `visitView`, `visitFields` (POST keys), `visitInput` (+`outcome_notes`, `cancelled`), `mergeOnto(v, birth, now)` order `starts_at, duration_min, kind, provider, location, note, outcome_notes, cancelled`; `textOK`; `visitKeys` pin; `TestVisitValidationOrder` | `httpapi/visits.go:51,99,109,133`; `visits_test.go:29,78` | 2 |
| `auth.NewLimiter(perMinute, now)`, `Allow(key)` counts refused calls too; `LimiterWindow`, `LimiterMaxKeys`; `respondRateLimited` (sets `Retry-After`) | `auth/limiter.go:23,28`; `constants.go:19-20`; `httpapi/session.go:63` | 4 |
| `Deps` (one `Limiter`, the login one, built in `main.go:81`); `Routes(d)`; conditional append `if d.PushRecorder != nil { … /v1/_test/pushes … }`; unmatched paths → `guard("unmatched", Internal, 404 not_found)` | `httpapi/routes.go:33-44,46,193-195,208-213`; `cmd/happybabytracker-api/main.go:80` | 4 |
| `routeClasses` (122 keys, `"GET /v1/_test/pushes": Internal` :83), `matrixParams` :220, `TestAccessMatrix` walks `Routes(newKit(t).deps())` and asserts admin 403 on `Parent` | `httpapi/matrix_test.go:59,498-584` | 4 |
| `kit.deps()` wires `PushRecorder` so the conditional route is walked | `httpapi/testkit_test.go:164` | 4 |
| One log line per request; `statusRecorder{status, cause, slide, outcome *[3]int}`; `recorder(w)`; `LogField*` names | `httpapi/middleware.go:51,66-93`; `cause.go:17`; `constants.go:176-181` | 4 |
| Outbound clients with `CheckRedirect: … http.ErrUseLastResponse` (precedent) | `ntfy/publish.go:50-62`; `push/send.go:89-91` | 3 |
| Visit limits `VisitProviderMaxLen = 120`, `VisitLocationMaxLen = 200` and their UI mirrors | `constants.go:218-219`; `ui/src/lib/constants.ts:197-198` | 1, 3, 6 |
| `VisitFields.svelte` (provider/location/note inputs, two column groups) | `ui/src/lib/components/visits/VisitFields.svelte` | 6 |
| `readVisitForm` / `build`: `FIELDS` list, edit sends only what differs from `<f>_original` | `ui/src/lib/server/visit-form.ts:17,76-125` | 5, 6 |
| Add/Edit sheet pages: `FormFrame`, `FormCancel`, `use:enhance={savedAs('Visit added')}`, action `save` | `routes/(app)/kids/[kid]/visits/new/+page.{svelte,server.ts}`, `[visit]/edit/…` | 6 |
| `savedAs` closes the sheet and toasts on **any** `success` result | `ui/src/lib/drawer.ts:127-147` | 6 (B11) |
| `ROUTES` already lists `visits/new` and `visits/[^/]+/edit` (no new sheet) | `ui/src/lib/drawer.ts:50-55` | 6 |
| `api.visits` client; `Endpoint` names; `reasonFor` maps 429 → `rate_limited`, 503 → `unavailable`; `API_TIMEOUT_MS = 5000` (> the API's 4 s upstream timeout) | `ui/src/lib/server/api.ts:138,207,1384`; `server/constants.ts:4` | 5 |
| `toVisit` parser | `ui/src/lib/server/api.ts:769` | 5 |
| Stub: `SCENARIOS` :43 (`push_off` precedent, `pushOn` :4079), `/__scenario` :4843, `/__reset`; visits store; `stub-visits.spec.ts` | `ui/e2e/stub-api.ts` | 5 |
| Smoke: `start_api` (no NPI env) :114, `api_call` :132, `refused` helper :1051 and 4l's list :1068-1073, 4o visits :1372-1528, step 9 log scan :1670, `API_MEM_LIMIT_MIB=64` :9 | `scripts/smoke.sh` | 7 |
| Main §13.1 env table and refusal paragraph | `docs/superpowers/specs/2026-11-20-happybabytracker-design.md:726-757` | 8 |
| Wiki: `Medical-visits.md:182` and `Not-built-yet.md:26` say the lookup is not built | `docs/wiki/` | 8 |

## Global Constraints

- CLAUDE.md rule 1: the edge contract changes by one API env var and one outbound host, and only in the owner's
  edge PR. Ports, health paths, `/data`, image names, uid 10001 and the read-only root do not change.
- Rule 5: no search term, name, address, phone or NPI in any log line or error string, in either container.
  Search terms travel only in the JSON body (API) and the POST form body (UI), never in a URL this app serves.
- Rule 7: new limits only in `api/internal/constants/constants.go`: `NPISearchLimit = 20`,
  `NPITimeout = 4 * time.Second`, `NPIMaxResponseBytes = 1 << 20`, `NPIMaxInFlight = 4`,
  `NPISearchesPerMinute = 20`, `NPICacheEntries = 256`, `NPICacheTTL = 10 * time.Minute`, `NPINameMinLen = 2`,
  `NPINameMaxLen = 60`, `NPINamePattern`, `NPIPostalCodeLen = 5`, `ProviderNPILen = 10`, `NPIStates`,
  `LogFieldNPIOutcome = "npi_outcome"`, `LogFieldNPIResults = "npi_results"`. Client-safe mirrors go in
  `ui/src/lib/constants.ts`: `NPI_NAME_MIN_LEN`, `NPI_NAME_MAX_LEN`, `NPI_POSTAL_CODE_LEN`, `NPI_STATES`. The UI
  also keeps the state names there (`NPI_STATE_LABEL`, words of the page spec). The registry host is a constant in
  `npi`, not configuration.
- No new dependency. Comments state constraints only, at most 2 lines. `gofmt -l` prints nothing; staticcheck clean.
- Every changed page works without JavaScript, is `no-store`, and is axe clean in light and dark at 390 and 1280 px.
  Words come from the page spec.
- Saving a visit never calls NPPES.
- Counts that move: `schema_version` `"9"`, `routeClasses` 122 → 124, `visitKeys` (+`provider_npi`; the list item,
  the detail and the timeline's visit view share it), `TestVisitValidationOrder` (+`provider_npi` after
  `location`), the stub's `SCENARIOS` (+3), the UI `Endpoint` union (+2).

## Review Focus

1. **A WA search must not show a provider who practises in OR or MN** even when the record's mailing address is in
   WA (Task 3 `TestSearchKeepsOnlyPracticeLocationMatches`).
2. **An `Errors` body at HTTP 200 is 503, not "no providers"**, and it is not cached (Task 3
   `TestErrorsBodyIsUpstreamError`; Task 4 `TestSearchErrorsAreNotCached`).
3. **No term, name or NPI in a log line**, and no `authorized_official_*` name in any result (Task 4
   `TestProviderSearchLogsNoTerms`; Task 3 `TestAuthorizedOfficialNeverSurfaces`).
4. **Flag off**: `POST /v1/providers/search` is a 404 byte-identical to an unknown route, and `GET /v1/providers`
   says disabled; the matrix walks both states (Task 4 `TestProviderSearchRouteFollowsFlag`; smoke 4p in Task 7).
5. **A changed provider text never keeps a stale NPI** (Task 2 `TestProviderChangeClearsNPI`; Task 6 e2e "editing
   the name after picking drops the registry link").

---

## API

### Task 1: Migration 0009, model, constants

**Files:**
- Create: `api/internal/store/migrations/0009_provider_npi.sql`
- Modify: `api/internal/model/tracker.go`, `api/internal/store/visits.go`, `api/internal/constants/constants.go`
- Test: `api/internal/store/store_test.go`, `api/internal/store/visits_test.go`

**Interfaces:**
- Produces: `model.Visit.ProviderNPI *string`; `visitColumns` ends `…, revision, provider_npi`; INSERT and the
  `UpdateVisit` UPDATE write it; the outcome UPDATE leaves it untouched.
- Constants of Global Constraints (all of them land here, so later tasks only read them).

- [ ] **Step 1: Failing tests.** Every `schema_version` pin becomes `"9"`. Add `TestMigration0009KeepsRows`: apply
  `ms[:8]`, seed a visit with a provider, `Open`, then assert the row is unchanged and `provider_npi IS NULL`.
  Add `TestProviderNPICheck`: a raw INSERT with `provider_npi` `'123456789'` (9 digits), `'12345678a3'`, or
  `'1234567893'` with `provider` NULL each fails the CHECK, and `'1234567893'` with a provider succeeds. Add a
  store round trip in `visits_test.go`: create with `ProviderNPI`, read it back, and update it to nil.
- [ ] **Step 2:** `cd api && go test ./internal/store/` → FAIL (no column, version 8).
- [ ] **Step 3: Migration.** `ALTER TABLE visits ADD COLUMN provider_npi TEXT CHECK (provider_npi IS NULL OR
  (provider IS NOT NULL AND length(provider_npi) = 10 AND provider_npi NOT GLOB '*[^0-9]*'));`. No rebuild is
  needed: a column CHECK may name another column, and the existing rows are NULL. The check digit is validated
  in Go (Task 2), not in SQL.
- [ ] **Step 4:** Add the model field, the columns, the scan and the constants.
- [ ] **Step 5:** Run the API pipeline (CLAUDE.md Commands) → PASS. Commit `api: migration 0009, visits.provider_npi`.

### Task 2: `provider_npi` on the visit routes

**Files:** Create `api/internal/npi/valid.go`, `valid_test.go`. Modify `api/internal/httpapi/visits.go`,
`visits_test.go`.

**Interfaces:**
- Produces: `func npi.Valid(s string) bool`: true for exactly 10 ASCII digits whose last digit is the Luhn check
  digit of `80840` + the first 9.
- `visitFields` gains `ProviderNPI optionalString \`json:"provider_npi"\``, so POST and PATCH both take it.
  `visitView` gains `ProviderNPI *string \`json:"provider_npi"\``.
- `mergeOnto` order becomes `starts_at, duration_min, kind, provider, location, provider_npi, note, outcome_notes,
  cancelled`. The `provider_npi` step runs: if `provider` was sent, differs from the stored value and
  `provider_npi` was not sent, clear it (B7). If sent, apply it. Then a non-null value must pass `npi.Valid` and the
  merged `provider` must be non-null, else 422 `provider_npi`.

- [ ] **Step 1: Failing tests.** `TestValid`: `1234567893` true; `1234567890`, `123456789`, `12345678931`,
  `123456789a` and `""` false, and a fixture of B12 (starts with 9) true. `visitKeys` gains `provider_npi`.
  `TestVisitValidationOrder` gains a row where `location` and `provider_npi` are both bad and the answer is
  `location`, and one where `provider_npi` and `note` are both bad and the answer is `provider_npi`.
  `TestProviderChangeClearsNPI`: create with provider "Avery Quill" and NPI. PATCH `{"location": "x"}` keeps the
  NPI. PATCH `{"provider": "Dr Q"}` sets it to null. A create, then PATCH `{"provider": "Dr Q", "provider_npi": "<npi>"}`,
  keeps it. PATCH `{"provider": null}` clears both. `{"provider_npi": "<npi>"}` on a visit with no provider is
  422 `provider_npi`. A bad check digit is 422 `provider_npi`, and the error body carries no digits of the value.
- [ ] **Step 2:** run → FAIL. **Step 3:** implement. **Step 4:** `go test ./internal/...` → PASS (the timeline and
  export tests that pin visit keys move with `visitKeys`).
- [ ] **Step 5:** Commit `api: visits take provider_npi`.

### Task 3: The `npi` client

**Files:** Create `api/internal/npi/npi.go`, `npi_test.go` (fixtures inline: JSON written by hand from the live
shape above, fictional names).

**Interfaces:**
- `type Kind string` (`Person = "person"`, `Organization = "organization"`).
- `type Query struct{ Kind Kind; LastName, FirstName, Organization, State, City, PostalCode string }` (validated by
  the caller, B4).
- `type Result struct{ NPI, Name string; Credential, Taxonomy *string; Address string; Phone *string }`.
- `type Page struct{ Results []Result; More bool }`.
- `type Outcome string` with the values `OK "ok"`, `Empty "empty"`, `CacheHit "cache_hit"`,
  `RateLimited "rate_limited"` (set by the handler), `Timeout "timeout"` and `UpstreamError "upstream_error"`.
- `var ErrUnavailable = errors.New("npi: unavailable")`. Every failure wraps it with a fixed reason, never a term.
- `func New(rt http.RoundTripper, now func() time.Time) *Searcher`. A nil `rt` means a clone of
  `http.DefaultTransport`. The client sets `Timeout: constants.NPITimeout` and `CheckRedirect` →
  `http.ErrUseLastResponse`.
- `func (s *Searcher) Search(ctx context.Context, q Query) (Page, Outcome, error)`. Task 3 builds the upstream call
  and the semaphore; Task 4 adds the cache in front.
- `const registryURL = "https://npiregistry.cms.hhs.gov/api/"`.

- [ ] **Step 1: Failing tests** (a `roundTripFunc` fake records each `*http.Request`):
  - `TestSearchRequest`: a person query sends `GET` to host `npiregistry.cms.hhs.gov` path `/api/`, with
    `version=2.1&enumeration_type=NPI-1&last_name=quil*&first_name=av*&state=WA&city=Seattle&postal_code=98101&limit=20`.
    An organization query sends `NPI-2` and `organization_name=quill*`, without the name keys.
  - `TestSearchMapsShape`: an NPI-1 record maps to `{npi, "Avery Quill", "MD", "Pediatrics",
    "12 Pine St, Suite 4, Seattle, WA 98101", "206-555-0100"}`, built from the LOCATION address, the primary
    taxonomy and title case. An NPI-2 record maps its `organization_name` and has a nil credential.
  - `TestSearchKeepsOnlyPracticeLocationMatches`: three records for a WA search. A (LOCATION WA) is kept. B (MAILING
    WA, LOCATION OR) is dropped, and so is C (LOCATION MN). With `city` Seattle, a LOCATION in WA Tacoma is dropped.
    With `postal_code` 98101, a LOCATION `981012756` is kept and `981040000` is dropped.
  - `TestAuthorizedOfficialNeverSurfaces`: an NPI-2 record with `authorized_official_first_name` "Zebedee". The
    marshalled `Page` does not contain "Zebedee" or "zebedee".
  - `TestSearchDrops`: an inactive status, a bad check digit, and a name over 120 characters are dropped. An
    address over 200 characters is retried without `address_2`.
  - `TestMoreWhenUpstreamFull`: 20 records upstream, 3 kept → `More` true. 19 upstream → false.
  - `TestErrorsBodyIsUpstreamError`: HTTP 200 `{"Errors":[{"description":"Field state requires additional search
    criteria","field":"state","number":"…"}]}` → `ErrUnavailable` and `UpstreamError`. The error string does not
    contain "state requires".
  - `TestUpstreamFailures` (table): status 500; 302 with a `Location` (the fake sees one request only); a body of
    `NPIMaxResponseBytes+1` bytes; `{` (unparseable). Each gives `UpstreamError`. A transport that blocks until
    the context ends gives `Timeout` (driven by a context deadline in the test, not 4 s of wall time).
  - `TestInFlightCap`: 4 calls block in the fake; a 5th returns `UpstreamError` at once without reaching the
    transport. Releasing one lets the next call through.
  - `TestEmpty`: `{"result_count":0,"results":[]}` → `Empty`, an empty slice (`[]`, not null, when marshalled).
- [ ] **Step 2:** `go test ./internal/npi/` → FAIL.
- [ ] **Step 3: Implement** with `io.LimitReader(body, max+1)`, `json.Decoder` into structs that list only the fields
  used, and `chan struct{}` of `NPIMaxInFlight` acquired with a non-blocking `select`.
- [ ] **Step 4:** `go test ./internal/npi/ -race` → PASS.
- [ ] **Step 5:** Commit `api: npi registry client`.

### Task 4: Cache, flag, limiter, routes, logs, matrix

**Files:** Modify `api/internal/npi/npi.go` (cache), `npi_test.go`, `api/internal/config/config.go`, `config_test.go`,
`api/internal/httpapi/routes.go`, `middleware.go`, `testkit_test.go`, `matrix_test.go`,
`api/cmd/happybabytracker-api/main.go`. Create `api/internal/httpapi/providers.go`, `providers_test.go`.

**Interfaces:**
- `config.Config.NPISearch bool`. `loadNPI(getenv)`: `""` → false, `"1"` → true, anything else →
  `errors.New("NPI_SEARCH must be 1 or unset")`.
- `Deps.NPI *npi.Searcher` (nil when off) and `Deps.NPILimiter *auth.Limiter`. `main.go` builds both only when
  `cfg.NPISearch` is set (`auth.NewLimiter(constants.NPISearchesPerMinute, time.Now)`).
- Routes: `{GET, "/v1/providers", Parent, d.providerState}` is always registered. After the `PushRecorder` block,
  `if d.NPI != nil { routes = append(routes, Route{http.MethodPost, "/v1/providers/search", Parent, d.searchProviders}) }`.
- `providerState` → `{"search":{"enabled":<d.NPI != nil>}}`.
- `searchProviders`: `decodeJSON` into `providerSearchInput` (unknown key 422 `unknown_field`), validated per B4
  (422 `invalid` + `field`). Then `d.NPILimiter.Allow("npi:"+parentID)`, else `respondRateLimited` (429
  `rate_limited`, `Retry-After`) with outcome `rate_limited`. Then `d.NPI.Search`: an `ErrUnavailable` → 503
  `unavailable`, and success → 200 `{"results":[{npi, name, credential, taxonomy, address, phone}], "more": bool}`.
- `statusRecorder` gains `npiOutcome string; npiResults int`. `noteNPI(w, outcome, n)`. `guard` appends
  `LogFieldNPIOutcome`/`LogFieldNPIResults` when `npiOutcome != ""` (B8).
- Cache in `Searcher` (B3): `map[string]cacheEntry{page Page; at time.Time}` plus insertion order, under a mutex.

- [ ] **Step 1: Failing tests.**
  - Config: `TestLoadNPISearch` (unset → false, `1` → true), and `TestLoadRefuses` rows `NPI_SEARCH=2`, `true`, `0`
    → error naming `NPI_SEARCH`, with the value not in the message.
  - Cache (`npi_test.go`): `TestSearchCachesNormalisedQuery`: `{" Quill ", "wa"}` then `{"quill", "WA"}` → one
    upstream request, second outcome `CacheHit`. After `NPICacheTTL` on the fake clock → a new request.
    `TestSearchErrorsAreNotCached`: a 500 then a 200 → two requests and the second succeeds. `TestCacheBound`:
    257 distinct queries → 256 entries, and the first is evicted.
  - HTTP (`providers_test.go`, the kit with a fake-transport `Searcher`): `GET /v1/providers` is `{search:{enabled:true}}`
    with the flag and `false` without it. `TestProviderSearchValidation` is a table over B4's order (person
    without `last_name` → `last_name`; organization with `first_name` → `first_name`; `state` "XX" → `state`;
    `postal_code` "9810" → `postal_code`; a `*` in `last_name` → `last_name`; "J" → `last_name`; `ab$` → `last_name`;
    unknown key → `unknown_field`). `TestProviderSearchRateLimit`: the 21st call in a minute → 429 `rate_limited`
    with `Retry-After`, and another parent still gets 200. Upstream failure → 503 `unavailable`.
    `TestProviderSearchLogsNoTerms`: after an ok, an empty, a cache hit, a 429 and a 503, the log lines carry
    `npi_outcome` `ok`, `empty`, `cache_hit`, `rate_limited` and `upstream_error` with `npi_results`, and the whole
    log buffer contains none of the terms, names, addresses, phones or NPIs used (`testlog`).
  - Matrix: `routeClasses` gains `"GET /v1/providers": Parent` and `"POST /v1/providers/search": Parent`, and
    `kit.deps()` sets `NPI` (fake transport) and `NPILimiter`, so the matrix walks both. The admin gets 403 on
    both (existing assertion). `TestProviderSearchRouteFollowsFlag`: `Routes(deps with NPI nil)` has no
    `POST /v1/providers/search`, and sending it gives a 404 whose body is byte-equal to that of
    `POST /v1/no-such-route`. `GET /v1/providers` then says `enabled:false`.
- [ ] **Step 2:** `go test ./internal/...` → FAIL.
- [ ] **Step 3:** Implement.
- [ ] **Step 4:** Run the API pipeline → PASS (gofmt, vet, staticcheck, `-race`).
- [ ] **Step 5:** Commit `api: provider search routes behind NPI_SEARCH`.

## UI

### Task 5: Types, constants, client, stub

**Files:** `ui/src/lib/types.ts`, `ui/src/lib/constants.ts`, `ui/src/lib/server/api.ts` (+ `api.test.ts`),
`ui/src/lib/server/visit-form.ts` (+ test), `ui/e2e/stub-api.ts`, `ui/e2e/stub-visits.spec.ts`.

- `Visit.providerNpi: string | null`; `toVisit` reads `provider_npi`. Types `ProviderQuery`
  (`kind, lastName?, firstName?, organization?, state, city?, postalCode?`), `ProviderResult`
  (`npi, name, credential, taxonomy, address, phone`) and `ProviderPage` (`results, more`).
- `api.providers.state()` → `{ enabled: boolean }` (endpoint `provider_state`), and `api.providers.search(q)` →
  `ProviderPage` (endpoint `provider_search`, POST JSON).
- Constants of Global Constraints (mirrors, and `NPI_STATE_LABEL` from the page spec).
- `readVisitForm`: `FIELDS` gains `provider_npi` (hidden input, with `provider_npi_original` on edit). Before `build`,
  the value is blanked unless `provider` equals the posted `provider_npi_for` (B7). Then the usual diff rules
  apply. `VISIT_FIELD_MESSAGES.provider_npi` comes from the page spec.
- Stub per B12: visits store and return `provider_npi` with the API's rules (B7). Add `GET /v1/providers`,
  `POST /v1/providers/search` and the three scenarios. `stub-visits.spec.ts` asserts the new key and the off-state
  404 body equal to an unknown route's.
- [ ] Tests first: `api.test.ts` (parse a visit with and without `provider_npi`; a search page; 429 → `rate_limited`;
  503 → `unavailable`); `visit-form.test.ts` (create with a matching `provider_npi_for` sends it, one that differs
  does not, edit after changing the name sends no `provider_npi`). Run → FAIL. Implement. Run
  `pnpm check && pnpm lint && pnpm test` → PASS. Commit `ui: provider search client, types and stub`.

### Task 6: The "Find a US provider" panel (after the page spec is approved)

**Files:** Create `ui/src/lib/components/visits/ProviderSearch.svelte`, `ui/src/lib/server/provider-form.ts`
(+ test), `ui/e2e/visits-provider.spec.ts`. Modify `VisitFields.svelte` (B9 slot), `visits/new/+page.{svelte,server.ts}`,
`visits/[visit]/edit/+page.{svelte,server.ts}`, `ui/src/lib/drawer.ts` (`inPlace`, B11).

**Interfaces:**
- `readProviderQuery(form) → { query: ProviderQueryValues; input: ProviderQuery | null; field: string | null }`.
  It reads the `npi_*` fields with the same rules as B4, and messages come from the page spec.
- `findAction(api, form, values)` → `{ values, providers: { query, page } | { query, field } | { query, error:
  'unavailable' | 'rate_limited' } }` (status 200, or `fail(422)` for a refused field).
- `pickResult(form, values)` → `values` with `provider`, `location`, `provider_npi` and `provider_npi_for` set
  (B10).
- `load` adds `providerSearch: boolean` from `api.providers.state()` (false on any failure). Both pages add actions
  `find` and `pick` and pass `savedAs('…', { inPlace: ['find', 'pick'] })`.

- [ ] **Step 1: Failing tests.** Unit: `provider-form.test.ts` covers the query rules, `pickResult` (credential
  appended when it fits, left off when it would pass 120), and a missing choice → field `npi_choice`.
  `drawer` unit/e2e: an `inPlace` success keeps the sheet open with no toast. E2e (`visits-provider.spec.ts`,
  scenario `npi_on`):
  - The panel is closed by default and absent under `normal` (search off).
  - Search by last name and state lists the stub's in-state fixtures only, each with its taxonomy and address, and
    never the OR-located one.
  - Picking fills "Doctor or clinic" and "Place" and shows the registry line. Saving gives the toast "Visit added",
    and the stub's visit has `provider_npi`.
  - Editing the name after picking, then saving, drops the registry link (the visit's `provider_npi` is null).
  - Empty search → the empty sentence. `npi_unavailable` → the unavailable alert, and the visit still saves.
    `npi_rate_limited` → the rate-limited alert.
  - Typed visit values (date, kind, reason) survive a search and a pick, with and without JavaScript
    (`javaScriptEnabled: false` project).
  - Enter in "Reason" saves the visit. With JavaScript, Enter in "Last name" searches.
  - The edit sheet shows the registry line for a visit with `provider_npi`.
  - axe clean in light and dark at 390 and 1280 with results showing.
- [ ] **Step 2:** `pnpm test && pnpm test:e2e` → FAIL.
- [ ] **Step 3:** Implement per the page spec (parts, words, order). The person/organization fields switch with CSS
  `:has(:checked)`, as the outcome sheet's sections do, so both work without JavaScript.
- [ ] **Step 4:** `pnpm check && pnpm lint && pnpm test && pnpm test:e2e` → PASS (existing visits e2e and baselines
  unchanged, since the stub is off by default).
- [ ] **Step 5:** Commit `ui: find a US provider panel in the visit sheets`.

## Smoke, docs

### Task 7: Smoke (off state and refusal)

**Files:** `scripts/smoke.sh`.

- [ ] **Step 1:** In 4l's refusal list, add `refused NPI_SEARCH - NPI_SEARCH=2`. Use `-`, because a single digit
  would match timestamps in the log. Add "NPI_SEARCH=2" to 4l's pass sentence.
- [ ] **Step 2:** New step 4p after 4o, with the API as `start_api` runs it (no `NPI_SEARCH`). `api_call` as
  `$USER_A`: `GET /v1/providers` → 200 `{"search":{"enabled":false}}`. `POST /v1/providers/search` with a valid
  body → 404, its body byte-equal to `POST /v1/no-such-route`'s. The Add a visit page and an Edit visit page (no
  JS, as 4o fetches them) do not contain "Find a US provider". The search body's terms go through `note`, so step 9
  scans for them. `pass "4p provider lookup is off without NPI_SEARCH: GET /v1/providers says disabled, the search
  route is the unknown-route 404, and neither visit sheet offers the panel"`.
- [ ] **Step 3:** Smoke never sets `NPI_SEARCH=1` and never calls the registry (§10 call). Run in CI (`images` job,
  no Docker here) → every line `PASS:`. Commit `ci: smoke for provider lookup off and NPI_SEARCH refusal`.

### Task 8: Docs

- [ ] Main spec §13.1: a row `NPI_SEARCH` | "optional; `1` turns on the US provider lookup (outbound HTTPS to
  `npiregistry.cms.hhs.gov`); unset leaves it off". Add "an `NPI_SEARCH` other than `1`" to the refusal paragraph.
- [ ] Visits spec §10: status line "built in plan 10", rulings 1 and B1–B12 summarised where they settle a point
  (filter, `more`, the outcome mapping of the in-flight cap, B7's API rule).
- [ ] Wiki: a "How do I find a doctor in the US registry?" section in `Medical-visits.md` with panel screenshots
  (`pnpm screenshots` against the stub's `npi_on`: closed, results, picked, unavailable). Replace line 182. Remove
  the Medical visits item from `Not-built-yet.md`. Say that the panel appears only once the server switch is on.
- [ ] `docs/superpowers/HANDOFF.md`: a plan 10 row and the edge item below as open.
- [ ] Commit `docs: provider lookup in the main spec, the wiki and the handoff`.

## Edge change (owner, in `davcs86/homelab-edge`)

Until this lands, the deployed app behaves exactly as today: `GET /v1/providers` says disabled and no sheet shows
the panel.

1. The API's env file gains `NPI_SEARCH=1`, rendered only when an inventory switch is on (name the owner's, e.g.
   `happybabytracker_npi_search_enabled`, default off). Any other value stops the API at start, and the edge's
   health check shows it.
2. The API container may open outbound HTTPS (TCP 443) to `npiregistry.cms.hhs.gov` and resolve that name. The UI
   container needs nothing new.
3. Nothing else changes: no port, no secret, no UI env var, the same health paths, `/data`, uid 10001 and read-only root.
4. The image pins are bumped to this PR's digests (both images changed) and deployed.
5. A manual check once after deploy: a person search for a common last name in one state lists only providers
   practising in that state, a search shortly after is fast (cache), and the API log line shows
   `npi_outcome` with no term in it.

## Done when

The API and UI pipelines of CLAUDE.md pass in this session, CI's `images` job prints every smoke line `PASS:`, and
the page spec section is approved. The PR lists the image content changed: `api/` sources (new package `npi`,
migration 0009 forward-only, config, routes) and `ui/` sources; no Dockerfile or dependency change. It also says
that the edge needs its pins bumped and a deploy, plus the edge change above before the feature shows.

## Not verified from here

- The live path from production: the edge's egress to `npiregistry.cms.hhs.gov` and NPPES's latency from there,
  checked by hand once after deploy (edge step 5). Smoke never calls the registry.
- How the edge restricts egress (a firewall, a proxy, a compose network). The edge PR decides how to express step 2.
- NPPES has no SLA and no documented rate limit. Behaviour under its throttling is only what B2 maps (non-200 →
  503).
- Smoke itself (CI only; no Docker in this container).
