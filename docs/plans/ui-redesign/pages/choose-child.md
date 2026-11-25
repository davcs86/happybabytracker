# Choose a child (no-JS fallback of the child switcher)

Prototype page: `Choose Child.dc.html`  ·  baselines: `screenshots/choose-child/<variant>/NN-<state>.png`  ·  layout data: `specs/choose-child.<variant>.layout.json`

## Routes and files (legacy repo)

| Route | Status |
|---|---|
| `/kids (the child switcher's no-JS page)` | exists as the switcher's no-JS page (`/kids`); if the legacy repo has no `/kids` list route, add it |

Legacy files to restyle: `ui/src/lib/components/KidSwitcher.svelte (replaced)`

## Layout (measured from the rendered prototype, Geist loaded)

Phone frame 390 px wide, content height 800 px; side-nav frame 980 px wide, content height 780 px, two-column frame 1280 px wide, content height 780 px. All numbers are CSS px at 1×.

**Headings (phone)**

| Level | Text | Box x,y w×h | font |
|---|---|---|---|
| h1 | Choose a child | 16,90 358×33 | 30px/600 |

**Interactive elements (default state)**

| Name | Element | shadcn part | Phone x,y w×h | 768–1279 x,y w×h | ≥1280 x,y w×h | Radius | Font |
|---|---|---|---|---|---|---|---|
| Notification settings | link | Button | 248,16 48×48 | 835,16 48×48 | 1086,16 48×48 | 12px | 16px/400 |

## States captured

`dup` = pixel-identical to an earlier state of the same variant (the control changed nothing visible); `axe` = violations found (0 = clean).

| # | State | Kind | Baseline (phone-light) | axe (pl/pd/dl/dd/wl/wd) | Notes |
|---|---|---|---|---|---|
| 01 | default-frame | frame | 01-default-frame.png | 0/0/0/0/0/0 |  |
|  | NEXT SAVE Works | state | dup of 01 | ·/·/·/·/·/· | NEXT SAVE |
|  | NEXT SAVE Offline | state | dup of 01 | ·/·/·/·/·/· | NEXT SAVE |
|  | NEXT SAVE Error | state | dup of 01 | ·/·/·/·/·/· | NEXT SAVE |
|  | CHILDREN Three | state | dup of 01 | ·/·/·/·/·/· | CHILDREN |
| 02 | CHILDREN Nine, long names | state | 02-children-nine-long-names.png | 0/0/0/0/0/0 | CHILDREN |
|  | ROLE Guardian | state | dup of 01 | ·/·/·/·/·/· | ROLE |
| 03 | ROLE Caregiver | state | 03-role-caregiver.png | 0/0/0/0/0/0 | ROLE |
| 04 | sheet More | sheet | 04-sheet-more.png | 0/0/·/·/·/· |  |

## Acceptance

- Every state above matches its baseline in all six variants (tolerance in `VERIFICATION.md`).
- axe clean per state; targets ≥ 44 px; no horizontal scroll at 390 px and 200 % zoom; at ≥ 1280 px the content block is 920 px wide and the left column precedes the right in the DOM (`DESIGN.md` §2.1).
- Accessible names equal the `Name` column (or are listed in `IMPLEMENTATION.md` section 6).
- The no-JS path in the notes above works with JavaScript disabled.
