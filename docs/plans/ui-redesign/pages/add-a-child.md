# Add a child (no-JS page of the Add child sheet)

Prototype page: `Add a Child.dc.html`  ·  baselines: `screenshots/add-a-child/<variant>/NN-<state>.png`  ·  layout data: `specs/add-a-child.<variant>.layout.json`

## Routes and files (legacy repo)

| Route | Status |
|---|---|
| `/kids/new` | exists (restyle). Designed 2026-11-24; same field set and validation as the sheet |

Legacy files to restyle: `ui/src/routes/(app)/kids/[kid]/../new/+page.svelte`

## Layout (measured from the rendered prototype, Geist loaded)

Phone frame 390 px wide, content height 1122 px; side-nav frame 980 px wide, content height 1053 px, two-column frame 1280 px wide, content height 780 px. All numbers are CSS px at 1×.

**Headings (phone)**

| Level | Text | Box x,y w×h | font |
|---|---|---|---|
| h1 |  | 16,90 358×33 | 30px/600 |

**Interactive elements (default state)**

| Name | Element | shadcn part | Phone x,y w×h | 768–1279 x,y w×h | ≥1280 x,y w×h | Radius | Font |
|---|---|---|---|---|---|---|---|
| Notification settings | link | Button | 248,16 48×48 | 835,16 48×48 | 1086,16 48×48 | 12px | 16px/400 |
| f-name | input |  | 16,227 358×48 | 241,227 448×48 | 292,204 448×48 | 8px | 15px/400 |
| f-born | input |  | 16,312 358×48 | 241,312 448×48 | 292,290 448×48 | 8px | 15px/400 |
| f-zone | input |  | 16,422 358×48 | 241,422 448×48 | 292,400 448×48 | 8px | 15px/400 |
| f-sex | input |  | 16,579 20×20 | 241,579 20×20 | 764,214 20×20 | 50% | 13.3333px/400 |
| f-gaw | input |  | 16,786 358×48 | 241,786 448×48 | 764,422 448×48 | 8px | 15px/400 |
| f-gad | input |  | 16,914 358×48 | 241,914 448×48 | 764,550 448×48 | 8px | 15px/400 |
| Cancel | link |  | 290,976 84×48 | 605,976 84×48 | 656,524 84×48 | 12px | 15px/600 |

## States captured

`dup` = pixel-identical to an earlier state of the same variant (the control changed nothing visible); `axe` = violations found (0 = clean).

| # | State | Kind | Baseline (phone-light) | axe (pl/pd/dl/dd/wl/wd) | Notes |
|---|---|---|---|---|---|
| 01 | default-frame | frame | 01-default-frame.png | 0/0/0/0/0/0 |  |
| 02 | default-full-height | full | 02-default-full-height.png | 0/0/0/0/·/· |  |
|  | NEXT SAVE Works | state | dup of 02 | ·/·/·/·/·/· | NEXT SAVE |
| 03 | NEXT SAVE Works after action | state-after-action | 03-next-save-works-after-action.png | 0/0/0/0/0/0 | NEXT SAVE |
|  | NEXT SAVE Offline | state | dup of 02 | ·/·/·/·/·/· | NEXT SAVE |
| 04 | NEXT SAVE Offline after action | state-after-action | 04-next-save-offline-after-action.png | 0/0/0/0/0/0 | NEXT SAVE |
|  | NEXT SAVE Error | state | dup of 02 | ·/·/·/·/·/· | NEXT SAVE |
| 05 | NEXT SAVE Error after action | state-after-action | 05-next-save-error-after-action.png | 0/0/0/0/0/0 | NEXT SAVE |
|  | SCREEN Empty form | state | dup of 02 | ·/·/·/·/·/· | SCREEN |
| 06 | SCREEN Field messages | state | 06-screen-field-messages.png | 0/0/0/0/0/0 | SCREEN |
|  | ROLE Guardian | state | dup of 02 | ·/·/·/·/·/· | ROLE |
| 07 | ROLE Caregiver | state | 07-role-caregiver.png | 0/0/0/0/0/0 | ROLE |
| 08 | sheet More | sheet | 08-sheet-more.png | 0/0/·/·/·/· |  |

## Acceptance

- Every state above matches its baseline in all six variants (tolerance in `VERIFICATION.md`).
- axe clean per state; targets ≥ 44 px; no horizontal scroll at 390 px and 200 % zoom; at ≥ 1280 px the content block is 920 px wide and the left column precedes the right in the DOM (`DESIGN.md` §2.1).
- Accessible names equal the `Name` column (or are listed in `IMPLEMENTATION.md` section 6).
- The no-JS path in the notes above works with JavaScript disabled.
