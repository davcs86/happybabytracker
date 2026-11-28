# How these screenshots are made

Every picture in this wiki is produced by code, not by hand, so it can be regenerated whenever the app changes and checked for drift.

## What is captured

- **270 screens and states**, each at two sizes: **mobile** 390 × 844 and **desktop** 1280 × 800 (540 images). Light theme, JavaScript on (except `visits/24-provider-no-js`, captured without it), full-page captures.
- A form that opens in a sheet is captured as the open sheet over the page it is opened from, as a person with JavaScript sees it. Only a form that no link opens in a sheet (Log sleep) is captured as a page.
- The real user interface (the SvelteKit app) running against `ui/e2e/stub-api.ts`, a stub of the API that holds fake people, children and entries. No real person or child appears.
- The images are in `docs/screenshots/<area>/<NN-name>-<mobile|desktop>.png`, and `docs/screenshots/README.md` is the generated catalogue: every shot, its caption, the route it shows, and what is not captured.
- The flows live in `ui/e2e/screenshots/*.shot.ts`. They are never part of `pnpm test:e2e`.

## Regenerate

From `ui/`, with Node 24, a Chromium, `openssl` and ports 4173, 4181 and 4182 free:

```sh
PW_CHROMIUM_PATH=/path/to/chrome pnpm screenshots
node e2e/screenshots/index.ts --check
```

`pnpm screenshots` builds the UI, starts it with the stub, runs every flow in both sizes, then rewrites the catalogue. The `--check` command confirms that the catalogue lists exactly the PNGs on disk and that **every link and image in `docs/wiki` points at a file that exists**, so a renamed or removed shot fails the check instead of leaving a broken picture.

## The stub single sign-on issuer

The single sign-on shots (`sso/*`) need an identity provider. The run starts `ui/e2e/stub-issuer.ts`, a stub issuer with one fake user, on `https://127.0.0.1:4182`; no real provider is contacted. An issuer must speak HTTPS, so the run first makes a self-signed certificate for `127.0.0.1` with `openssl` (`ui/e2e/tls.ts`), valid for 7 days and kept in `ui/e2e/.tls`, which is never committed. A later run reuses it until it is within an hour of expiring, then makes a new one. The UI trusts that one certificate, and the browser is told to accept it. Without `openssl` on the `PATH` the run stops before it starts.

The stub's sign-in mode stays passwords-only unless a shot sets it, so every other page looks as it does on a deployment without single sign-on.

## Why two runs give the same pictures

The stub, the UI server, the test workers and the browser all read one frozen instant, **2026-11-24T10:30:00Z**, in the UTC time zone. Ages, dates, "today" and every relative time are therefore the same whatever day the flow runs. Animations are off, the text caret is hidden, the pointer is parked and the page waits for the network and the fonts before each capture. A page taller than the screen is captured whole: the capture grows the window until the page fits, so the fixed tab bar sits at the bottom of the picture.

Two consecutive complete runs were compared file by file: all 508 images were byte-identical.

## The people and children on the screens

The catalogue's "Fixtures" section lists them. In short: Ana (guardian of Mia and Ivy, caregiver of Leo), Cleo (guardian of Leo, Nell, Otto, Pia, Remy and Sami), Dan (no children), Bo (invited, no password yet), Dee (disabled) and the admin. Children range from a 10-day-old to a 7-year-old, so every age band, a preterm child and a child with Milestones switched off are all on screen.

## Changing a screen or adding one

1. Edit or add a shot in the matching `ui/e2e/screenshots/<area>.shot.ts`. A name is `NN-slug` and the caption is the last argument of `shot(page, area, name, caption)`.
2. Run `pnpm screenshots`. Every shot must exist at both sizes; a shot that is renamed or removed leaves its old PNGs behind, and the check fails until you delete them.
3. Update the wiki page that shows the screen, and run `node e2e/screenshots/index.ts --check`.

A caption describes what the picture shows. Where a shot still shows an earlier error message because the flow typed over it without saving, the caption says so.

## The first-pass set

The screenshots of the design before the redesign are kept unchanged in `docs/screenshots-legacy`. See [Before and after](Before-and-after.md).

Back to the [wiki home](Home.md).
