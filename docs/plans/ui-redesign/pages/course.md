# Medication: course (add and edit)

Prototype page: `Course.dc.html`  ·  baselines: `screenshots/course/<variant>/NN-<state>.png`  ·  layout data: `specs/course.<variant>.layout.json`

## Routes and files (legacy repo)

| Route | Status |
|---|---|
| `/kids/[kid]/medication/new` | exists |
| `/kids/[kid]/medication/[medication]` | exists |

Legacy files to restyle: `ui/src/routes/(app)/kids/[kid]/medication/new/+page.svelte`, `ui/src/routes/(app)/kids/[kid]/medication/[medication]/+page.svelte`

## Implementation notes carried over from the prototype

These are the design notes that sit in the prototype's own side panel (they are part of the design, the panel itself is not).

```text
shadcn-svelte parts
Field, Input, Label, RadioGroup, Checkbox, Collapsible, Button, Alert.

Assumptions (labelled)
Dose is required, 1 to 80 characters (the handoff gives only the message). The time grid is 2 columns at 390 px, 3 from 640 px.

Errors
Save an empty name, two equal times, or 169 hours.
```

## Layout (measured from the rendered prototype, Geist loaded)

Phone frame 390 px wide, content height 1013 px; side-nav frame 980 px wide, content height 944 px, two-column frame 1280 px wide, content height 780 px. All numbers are CSS px at 1×.

**Headings (phone)**

| Level | Text | Box x,y w×h | font |
|---|---|---|---|
| h1 |  | 16,90 358×33 | 30px/600 |

**Interactive elements (default state)**

| Name | Element | shadcn part | Phone x,y w×h | 768–1279 x,y w×h | ≥1280 x,y w×h | Radius | Font |
|---|---|---|---|---|---|---|---|
| Notification settings | link | Button | 248,16 48×48 | 835,16 48×48 | 1086,16 48×48 | 12px | 16px/400 |
| f-name | input |  | 16,212 358×48 | 241,212 448×48 | 292,212 448×48 | 8px | 15px/400 |
| f-dose | input |  | 16,297 358×48 | 241,297 448×48 | 292,297 448×48 | 8px | 15px/400 |
| f-kind | input |  | 16,392 20×20 | 241,392 20×20 | 292,392 20×20 | 50% | 13.3333px/400 |
| f-start | input |  | 16,553 358×48 | 241,553 448×48 | 764,212 448×48 | 8px | 15px/400 |
| f-last | input |  | 16,662 358×48 | 241,662 448×48 | 764,322 448×48 | 8px | 15px/400 |
| f-active | input |  | 16,736 20×20 | 241,736 20×20 | 764,396 20×20 | 4px | 13.3333px/400 |
| f-note | input |  | 16,806 358×48 | 241,806 448×48 | 764,464 448×48 | 8px | 15px/400 |
| Save | button | Button | 16,868 266×48 | 241,868 356×48 | 292,530 356×48 | 12px | 15px/600 |
| Cancel | link |  | 290,868 84×48 | 605,868 84×48 | 656,530 84×48 | 12px | 15px/600 |

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
|  | SCREEN Add | state | dup of 02 | ·/·/·/·/·/· | SCREEN |
| 06 | SCREEN Edit | state | 06-screen-edit.png | 0/0/0/0/0/0 | SCREEN |
| 07 | EDIT WHICH COURSE Calpol (fixed) | state | 07-edit-which-course-calpol-fixed.png | 0/0/0/0/0/0 | EDIT WHICH COURSE |
|  | EDIT WHICH COURSE Amoxil (hourly) | state | dup of 06 | ·/·/·/·/·/· | EDIT WHICH COURSE |
| 08 | EDIT WHICH COURSE Saline (0 doses) | state | 08-edit-which-course-saline-0-doses.png | 0/0/0/0/0/0 | EDIT WHICH COURSE |
|  | All 12 (as built) | state | dup of 02 | ·/·/·/·/·/· |  |
|  | 3 + “More times” | state | dup of 02 | ·/·/·/·/·/· |  |
|  | As built | state | dup of 02 | ·/·/·/·/·/· |  |
|  | Omit the line | state | dup of 02 | ·/·/·/·/·/· |  |
|  | ROLE Guardian | state | dup of 02 | ·/·/·/·/·/· | ROLE |
| 09 | ROLE Caregiver | state | 09-role-caregiver.png | 0/0/0/0/0/0 | ROLE |
| 10 | sheet More | sheet | 10-sheet-more.png | 0/0/·/·/·/· |  |

## Acceptance

- Every state above matches its baseline in all six variants (tolerance in `VERIFICATION.md`).
- axe clean per state; targets ≥ 44 px; no horizontal scroll at 390 px and 200 % zoom; at ≥ 1280 px the content block is 920 px wide and the left column precedes the right in the DOM (`DESIGN.md` §2.1).
- Accessible names equal the `Name` column (or are listed in `IMPLEMENTATION.md` section 6).
- The no-JS path in the notes above works with JavaScript disabled.
