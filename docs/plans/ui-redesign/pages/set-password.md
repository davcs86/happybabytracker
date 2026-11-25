# Set password (invite and reset)

Prototype page: `Set Password.dc.html`  ·  baselines: `screenshots/set-password/<variant>/NN-<state>.png`  ·  layout data: `specs/set-password.<variant>.layout.json`

## Routes and files (legacy repo)

| Route | Status |
|---|---|
| `/invite/[token]` | exists |
| `/reset/[token]` | exists |

Legacy files to restyle: `ui/src/routes/invite/[token]/+page.svelte`, `ui/src/routes/reset/[token]/+page.svelte`, `ui/src/lib/components/SetPasswordForm.svelte`

## Layout (measured from the rendered prototype, Geist loaded)

Phone frame 390 px wide, content height 800 px; side-nav frame 980 px wide, content height 780 px, two-column frame 1280 px wide, content height 780 px. All numbers are CSS px at 1×.

**Headings (phone)**

| Level | Text | Box x,y w×h | font |
|---|---|---|---|
| h1 |  | 20,194 350×33 | 30px/600 |

**Interactive elements (default state)**

| Name | Element | shadcn part | Phone x,y w×h | 768–1279 x,y w×h | ≥1280 x,y w×h | Radius | Font |
|---|---|---|---|---|---|---|---|
| Username | label |  | 37,298 316×16 | 307,288 366×16 | 457,288 366×16 |  | 12px/400 |
| username | input |  | 37,320 316×48 | 307,310 366×48 | 457,310 366×48 | 8px | 16px/400 |
| New password | label |  | 37,382 316×16 | 307,372 366×16 | 457,372 366×16 |  | 12px/400 |
| password | input |  | 37,404 316×48 | 307,394 366×48 | 457,394 366×48 | 8px | 16px/400 |
| Confirm password | label |  | 37,490 316×16 | 307,480 366×16 | 457,480 366×16 |  | 12px/400 |
| confirm | input |  | 37,512 316×48 | 307,502 366×48 | 457,502 366×48 | 8px | 16px/400 |
| Set password | button | Button | 37,574 316×52 | 307,564 366×52 | 457,564 366×52 | 12px | 15px/600 |

## States captured

`dup` = pixel-identical to an earlier state of the same variant (the control changed nothing visible); `axe` = violations found (0 = clean).

| # | State | Kind | Baseline (phone-light) | axe (pl/pd/dl/dd/wl/wd) | Notes |
|---|---|---|---|---|---|
| 01 | default-frame | frame | 01-default-frame.png | 0/0/0/0/0/0 |  |
| 02 | LINK Reset | state | 02-link-reset.png | 0/0/0/0/0/0 | LINK |
| 03 | LINK Reset after action | state-after-action | 03-link-reset-after-action.png | 0/0/0/0/0/0 | LINK |
| 04 | LINK STATE Used or expired | state | 04-link-state-used-or-expired.png | 0/0/0/0/0/0 | LINK STATE |
|  | NEXT SAVE Rate limited | state | dup of 01 | ·/·/·/·/·/· | NEXT SAVE |
| 05 | NEXT SAVE Rate limited after action | state-after-action | 05-next-save-rate-limited-after-action.png | 0/0/0/0/0/0 | NEXT SAVE |
| 06 | short password | flow | 06-short-password.png | 0/0/0/0/0/0 |  |
| 07 | mismatch | flow | 07-mismatch.png | 0/0/0/0/0/0 |  |
| 08 | password set | flow | 08-password-set.png | 0/0/0/0/0/0 |  |

## Acceptance

- Every state above matches its baseline in all six variants (tolerance in `VERIFICATION.md`).
- axe clean per state; targets ≥ 44 px; no horizontal scroll at 390 px and 200 % zoom; at ≥ 1280 px the content block is 920 px wide and the left column precedes the right in the DOM (`DESIGN.md` §2.1).
- Accessible names equal the `Name` column (or are listed in `IMPLEMENTATION.md` section 6).
- The no-JS path in the notes above works with JavaScript disabled.
