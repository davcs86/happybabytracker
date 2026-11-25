# Mark as reached / add a milestone

Prototype page: `Milestone Form.dc.html`  ·  baselines: `screenshots/milestone-form/<variant>/NN-<state>.png`  ·  layout data: `specs/milestone-form.<variant>.layout.json`

## Routes and files (legacy repo)

| Route | Status |
|---|---|
| `/kids/[kid]/milestones/new[?item=]` | exists |
| `/kids/[kid]/milestones/[entry]` | exists |

Legacy files to restyle: `ui/src/routes/(app)/kids/[kid]/milestones/new/+page.svelte`, `ui/src/routes/(app)/kids/[kid]/milestones/[entry]/+page.svelte`

## Layout (measured from the rendered prototype, Geist loaded)

Phone frame 390 px wide, content height 800 px; side-nav frame 980 px wide, content height 780 px, two-column frame 1280 px wide, content height 780 px. All numbers are CSS px at 1×.

**Headings (phone)**

| Level | Text | Box x,y w×h | font |
|---|---|---|---|
| h1 |  | 16,90 358×33 | 30px/600 |

**Interactive elements (default state)**

| Name | Element | shadcn part | Phone x,y w×h | 768–1279 x,y w×h | ≥1280 x,y w×h | Radius | Font |
|---|---|---|---|---|---|---|---|
| Notification settings | link | Button | 248,16 48×48 | 835,16 48×48 | 1086,16 48×48 | 12px | 16px/400 |
| f-date | input |  | 16,224 358×48 | 241,224 448×48 | 292,224 448×48 | 8px | 15px/400 |
| f-note | input |  | 16,309 358×48 | 241,309 448×48 | 764,224 448×48 | 8px | 15px/400 |
| Save | button | Button | 16,371 266×48 | 241,371 356×48 | 292,286 356×48 | 12px | 15px/600 |
| Cancel | link |  | 290,371 84×48 | 605,371 84×48 | 656,286 84×48 | 12px | 15px/600 |

## States captured

`dup` = pixel-identical to an earlier state of the same variant (the control changed nothing visible); `axe` = violations found (0 = clean).

| # | State | Kind | Baseline (phone-light) | axe (pl/pd/dl/dd/wl/wd) | Notes |
|---|---|---|---|---|---|
| 01 | default-frame | frame | 01-default-frame.png | 0/0/0/0/0/0 |  |
|  | NEXT SAVE Works | state | dup of 01 | ·/·/·/·/·/· | NEXT SAVE |
| 02 | NEXT SAVE Works after action | state-after-action | 02-next-save-works-after-action.png | 0/0/0/0/0/0 | NEXT SAVE |
|  | NEXT SAVE Offline | state | dup of 01 | ·/·/·/·/·/· | NEXT SAVE |
| 03 | NEXT SAVE Offline after action | state-after-action | 03-next-save-offline-after-action.png | 0/0/0/0/0/0 | NEXT SAVE |
|  | NEXT SAVE Error | state | dup of 01 | ·/·/·/·/·/· | NEXT SAVE |
| 04 | NEXT SAVE Error after action | state-after-action | 04-next-save-error-after-action.png | 0/0/0/0/0/0 | NEXT SAVE |
|  | SCREEN Mark as reached | state | dup of 01 | ·/·/·/·/·/· | SCREEN |
| 05 | SCREEN Add a milestone | state | 05-screen-add-a-milestone.png | 0/0/0/0/0/0 | SCREEN |
| 06 | SCREEN Edit (catalog) | state | 06-screen-edit-catalog.png | 0/0/0/0/0/0 | SCREEN |
| 07 | SCREEN Edit (custom) | state | 07-screen-edit-custom.png | 0/0/0/0/0/0 | SCREEN |
|  | UNDO / DELETE CONFIRM Closed | state | dup of 01 | ·/·/·/·/·/· | UNDO / DELETE CONFIRM |
|  | UNDO / DELETE CONFIRM Open | state | dup of 01 | ·/·/·/·/·/· | UNDO / DELETE CONFIRM |
|  | UNDO / DELETE CONFIRM Failed | state | dup of 01 | ·/·/·/·/·/· | UNDO / DELETE CONFIRM |
|  | ROLE Guardian | state | dup of 01 | ·/·/·/·/·/· | ROLE |
| 08 | ROLE Caregiver | state | 08-role-caregiver.png | 0/0/0/0/0/0 | ROLE |
| 09 | sheet More | sheet | 09-sheet-more.png | 0/0/·/·/·/· |  |

## Acceptance

- Every state above matches its baseline in all six variants (tolerance in `VERIFICATION.md`).
- axe clean per state; targets ≥ 44 px; no horizontal scroll at 390 px and 200 % zoom; at ≥ 1280 px the content block is 920 px wide and the left column precedes the right in the DOM (`DESIGN.md` §2.1).
- Accessible names equal the `Name` column (or are listed in `IMPLEMENTATION.md` section 6).
- The no-JS path in the notes above works with JavaScript disabled.
