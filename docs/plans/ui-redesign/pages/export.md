# Export PDF

Prototype page: `Export.dc.html`  ·  baselines: `screenshots/export/<variant>/NN-<state>.png`  ·  layout data: `specs/export.<variant>.layout.json`

## Routes and files (legacy repo)

| Route | Status |
|---|---|
| `/kids/[kid]/export` | exists |

Legacy files to restyle: `ui/src/routes/(app)/kids/[kid]/export/+page.svelte`

## Layout (measured from the rendered prototype, Geist loaded)

Phone frame 390 px wide, content height 800 px; side-nav frame 980 px wide, content height 780 px, two-column frame 1280 px wide, content height 780 px. All numbers are CSS px at 1×.

**Headings (phone)**

| Level | Text | Box x,y w×h | font |
|---|---|---|---|
| h1 | Export PDF | 16,90 358×33 | 30px/600 |

**Interactive elements (default state)**

| Name | Element | shadcn part | Phone x,y w×h | 768–1279 x,y w×h | ≥1280 x,y w×h | Radius | Font |
|---|---|---|---|---|---|---|---|
| Child: Leo. Switch child | link | Button | 16,16 112×48 | 241,16 112×48 | 292,16 112×48 | 12px | 15px/600 |
| Notification settings | link | Button | 248,16 48×48 | 835,16 48×48 | 1086,16 48×48 | 12px | 16px/400 |
| From | label |  | 16,191 173×17 | 241,191 218×17 | 292,191 218×17 |  | 13px/400 |
| from | input |  | 16,214 173×48 | 241,214 218×48 | 292,214 218×48 | 8px | 15px/400 |
| To | label |  | 201,191 173×17 | 471,191 218×17 | 522,191 218×17 |  | 13px/400 |
| to | input |  | 201,214 173×48 | 471,214 218×48 | 522,214 218×48 | 8px | 15px/400 |
| modules | input |  | 17,302 89×42 | 242,302 89×42 | 766,215 89×42 | 4px | 13.3333px/400 |
| Download PDF | button | Button | 16,569 144×48 | 241,465 144×48 | 292,276 144×48 | 12px | 15px/600 |

## States captured

`dup` = pixel-identical to an earlier state of the same variant (the control changed nothing visible); `axe` = violations found (0 = clean).

| # | State | Kind | Baseline (phone-light) | axe (pl/pd/dl/dd/wl/wd) | Notes |
|---|---|---|---|---|---|
| 01 | default-frame | frame | 01-default-frame.png | 0/0/0/0/0/0 |  |
|  | NEXT SAVE Works | state | dup of 01 | ·/·/·/·/·/· | NEXT SAVE |
| 02 | NEXT SAVE Works after action | state-after-action | 02-next-save-works-after-action.png | 0/0/0/0/0/0 | NEXT SAVE |
|  | NEXT SAVE Offline | state | dup of 01 | ·/·/·/·/·/· | NEXT SAVE |
|  | NEXT SAVE Offline after action | state-after-action | dup of 02 | ·/·/·/·/·/· | NEXT SAVE |
|  | NEXT SAVE Error | state | dup of 01 | ·/·/·/·/·/· | NEXT SAVE |
| 03 | NEXT SAVE Error after action | state-after-action | 03-next-save-error-after-action.png | 0/0/·/·/·/· | NEXT SAVE |
|  | MODULES All on | state | dup of 01 | ·/·/·/·/·/· | MODULES |
| 04 | MODULES Some switched off | state | 04-modules-some-switched-off.png | 0/0/0/0/0/0 | MODULES |
|  | AFTER A REFUSAL (REDIRECT) None | state | dup of 01 | ·/·/·/·/·/· | AFTER A REFUSAL (REDIRECT) |
|  | AFTER A REFUSAL (REDIRECT) None after action | state-after-action | dup of 02 | ·/·/·/·/·/· | AFTER A REFUSAL (REDIRECT) |
| 05 | AFTER A REFUSAL (REDIRECT) From | state | 05-after-a-refusal-redirect-from.png | 0/0/0/0/0/0 | AFTER A REFUSAL (REDIRECT) |
|  | AFTER A REFUSAL (REDIRECT) From after action | state-after-action | dup of 05 | ·/·/·/·/·/· | AFTER A REFUSAL (REDIRECT) |
| 06 | AFTER A REFUSAL (REDIRECT) To | state | 06-after-a-refusal-redirect-to.png | 0/0/0/0/0/0 | AFTER A REFUSAL (REDIRECT) |
|  | AFTER A REFUSAL (REDIRECT) To after action | state-after-action | dup of 06 | ·/·/·/·/·/· | AFTER A REFUSAL (REDIRECT) |
| 07 | AFTER A REFUSAL (REDIRECT) Modules | state | 07-after-a-refusal-redirect-modules.png | 0/0/0/0/0/0 | AFTER A REFUSAL (REDIRECT) |
|  | AFTER A REFUSAL (REDIRECT) Modules after action | state-after-action | dup of 07 | ·/·/·/·/·/· | AFTER A REFUSAL (REDIRECT) |
| 08 | AFTER A REFUSAL (REDIRECT) Busy | state | 08-after-a-refusal-redirect-busy.png | 0/0/0/0/0/0 | AFTER A REFUSAL (REDIRECT) |
| 09 | AFTER A REFUSAL (REDIRECT) Busy after action | state-after-action | 09-after-a-refusal-redirect-busy-after-action.png | 0/0/·/·/0/0 | AFTER A REFUSAL (REDIRECT) |
|  | PAGE STATE Normal | state | dup of 01 | ·/·/·/·/·/· | PAGE STATE |
| 10 | PAGE STATE Not found | state | 10-page-state-not-found.png | 0/0/0/0/0/0 | PAGE STATE |
|  | ROLE Guardian | state | dup of 01 | ·/·/·/·/·/· | ROLE |
| 11 | ROLE Caregiver | state | 11-role-caregiver.png | 0/0/0/0/0/0 | ROLE |
| 12 | sheet Switch child | sheet | 12-sheet-switch-child.png | 0/0/0/0/0/0 |  |
| 13 | sheet More | sheet | 13-sheet-more.png | 0/0/·/·/·/· |  |

## Acceptance

- Every state above matches its baseline in all six variants (tolerance in `VERIFICATION.md`).
- axe clean per state; targets ≥ 44 px; no horizontal scroll at 390 px and 200 % zoom; at ≥ 1280 px the content block is 920 px wide and the left column precedes the right in the DOM (`DESIGN.md` §2.1).
- Accessible names equal the `Name` column (or are listed in `IMPLEMENTATION.md` section 6).
- The no-JS path in the notes above works with JavaScript disabled.
