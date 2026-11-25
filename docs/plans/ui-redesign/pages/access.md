# Access (people with access, add a person)

Prototype page: `Access.dc.html`  ·  baselines: `screenshots/access/<variant>/NN-<state>.png`  ·  layout data: `specs/access.<variant>.layout.json`

## Routes and files (legacy repo)

| Route | Status |
|---|---|
| `/kids/[kid]/access` | exists (restyle). Designed 2026-11-24; the sole guardian cannot be removed or demoted |

Legacy files to restyle: `ui/src/routes/(app)/kids/[kid]/access/+page.svelte`

## Layout (measured from the rendered prototype, Geist loaded)

Phone frame 390 px wide, content height 1217 px; side-nav frame 980 px wide, content height 1109 px, two-column frame 1280 px wide, content height 780 px. All numbers are CSS px at 1×.

**Headings (phone)**

| Level | Text | Box x,y w×h | font |
|---|---|---|---|
| h1 | Access | 16,90 358×33 | 30px/600 |
| h2 | People with access | 33,184 324×21 | 16px/600 |
| h2 | Add a person | 33,745 324×21 | 16px/600 |

**Interactive elements (default state)**

| Name | Element | shadcn part | Phone x,y w×h | 768–1279 x,y w×h | ≥1280 x,y w×h | Radius | Font |
|---|---|---|---|---|---|---|---|
| Notification settings | link | Button | 248,16 48×48 | 835,16 48×48 | 1086,16 48×48 | 12px | 16px/400 |
| Make Ana a caregiver | button | Button | 33,337 152×48 | 258,318 152×48 | 310,318 152×48 | 12px | 15px/600 |
| Remove Ana’s access to Leo | button | Button | 194,337 154×48 | 418,318 154×48 | 470,318 154×48 | 12px | 15px/600 |
| Make Cleo a guardian | button | Button | 33,519 148×48 | 258,500 148×48 | 310,500 148×48 | 12px | 15px/600 |
| Remove Cleo’s access to Leo | button | Button | 189,519 154×48 | 414,500 154×48 | 466,500 154×48 | 12px | 15px/600 |
| Make Sam a guardian | button | Button | 33,637 148×48 | 258,618 148×48 | 310,618 148×48 | 12px | 15px/600 |
| Remove Sam’s access to Leo | button | Button | 189,637 154×48 | 414,618 154×48 | 466,618 154×48 | 12px | 15px/600 |
| f-un | input |  | 33,872 324×48 | 258,832 686×48 | 782,290 414×48 | 8px | 15px/400 |
| f-role | input |  | 33,964 20×20 | 258,926 20×20 | 782,384 20×20 | 50% | 13.3333px/400 |
| Give access | button | Button | 33,1054 324×48 | 258,1016 686×48 | 782,474 414×48 | 12px | 15px/600 |

## States captured

`dup` = pixel-identical to an earlier state of the same variant (the control changed nothing visible); `axe` = violations found (0 = clean).

| # | State | Kind | Baseline (phone-light) | axe (pl/pd/dl/dd/wl/wd) | Notes |
|---|---|---|---|---|---|
| 01 | default-frame | frame | 01-default-frame.png | 0/0/0/0/0/0 |  |
| 02 | default-full-height | full | 02-default-full-height.png | 0/0/0/0/·/· |  |
|  | NEXT SAVE Works | state | dup of 02 | ·/·/·/·/·/· | NEXT SAVE |
|  | NEXT SAVE Offline | state | dup of 02 | ·/·/·/·/·/· | NEXT SAVE |
|  | NEXT SAVE Error | state | dup of 02 | ·/·/·/·/·/· | NEXT SAVE |
|  | PEOPLE Two guardians | state | dup of 02 | ·/·/·/·/·/· | PEOPLE |
| 03 | PEOPLE Only you | state | 03-people-only-you.png | 0/0/0/0/0/0 | PEOPLE |
|  | ROLE Guardian | state | dup of 02 | ·/·/·/·/·/· | ROLE |
| 04 | ROLE Caregiver | state | 04-role-caregiver.png | 0/0/0/0/0/0 | ROLE |
| 05 | sheet More | sheet | 05-sheet-more.png | 0/0/·/·/·/· |  |

## Acceptance

- Every state above matches its baseline in all six variants (tolerance in `VERIFICATION.md`).
- axe clean per state; targets ≥ 44 px; no horizontal scroll at 390 px and 200 % zoom; at ≥ 1280 px the content block is 920 px wide and the left column precedes the right in the DOM (`DESIGN.md` §2.1).
- Accessible names equal the `Name` column (or are listed in `IMPLEMENTATION.md` section 6).
- The no-JS path in the notes above works with JavaScript disabled.
