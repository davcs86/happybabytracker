# Medication: edit dose

Prototype page: `Dose.dc.html`  ·  baselines: `screenshots/dose/<variant>/NN-<state>.png`  ·  layout data: `specs/dose.<variant>.layout.json`

## Routes and files (legacy repo)

| Route | Status |
|---|---|
| `/kids/[kid]/medication/doses/[dose]` | exists |

Legacy files to restyle: `ui/src/routes/(app)/kids/[kid]/medication/doses/[dose]/+page.svelte`

## Implementation notes carried over from the prototype

These are the design notes that sit in the prototype's own side panel (they are part of the design, the panel itself is not).

```text
shadcn-svelte parts
Field, Input, Label, Button, Collapsible, Alert.

Picks the dose you tapped on the Medication page, else the newest.
```

## Layout (measured from the rendered prototype, Geist loaded)

Phone frame 390 px wide, content height 800 px; side-nav frame 980 px wide, content height 780 px, two-column frame 1280 px wide, content height 780 px. All numbers are CSS px at 1×.

**Headings (phone)**

| Level | Text | Box x,y w×h | font |
|---|---|---|---|
| h1 | Edit dose | 16,90 358×33 | 30px/600 |

**Interactive elements (default state)**

| Name | Element | shadcn part | Phone x,y w×h | 768–1279 x,y w×h | ≥1280 x,y w×h | Radius | Font |
|---|---|---|---|---|---|---|---|
| Notification settings | link | Button | 248,16 48×48 | 835,16 48×48 | 1086,16 48×48 | 12px | 16px/400 |
| f-at | input |  | 16,234 358×48 | 241,234 448×48 | 292,234 448×48 | 8px | 15px/400 |
| f-amount | input |  | 16,319 358×48 | 241,319 448×48 | 292,319 448×48 | 8px | 15px/400 |
| f-note | input |  | 16,404 358×48 | 241,404 448×48 | 764,234 448×48 | 8px | 15px/400 |
| Save | button | Button | 16,466 266×48 | 241,466 356×48 | 292,381 356×48 | 12px | 15px/600 |
| Cancel | link |  | 290,466 84×48 | 605,466 84×48 | 656,381 84×48 | 12px | 15px/600 |
| Delete this dose | disclosure | Button | 16,545 358×48 | 241,545 448×48 | 292,460 448×48 | 12px | 15px/600 |
| Yes, delete this dose | button | Button | 16,628 358×48 | 241,628 448×48 | 292,542 448×48 | 12px | 15px/600 |

## States captured

`dup` = pixel-identical to an earlier state of the same variant (the control changed nothing visible); `axe` = violations found (0 = clean).

| # | State | Kind | Baseline (phone-light) | axe (pl/pd/dl/dd/wl/wd) | Notes |
|---|---|---|---|---|---|
| 01 | default-frame | frame | 01-default-frame.png | 0/0/0/0/0/0 |  |
|  | NEXT SAVE Works | state | dup of 01 | ·/·/0/·/·/· | NEXT SAVE |
| 02 | NEXT SAVE Works after action | state-after-action | 02-next-save-works-after-action.png | 0/0/0/0/0/0 | NEXT SAVE |
|  | NEXT SAVE Offline | state | dup of 01 | ·/·/·/·/0/· | NEXT SAVE |
| 03 | NEXT SAVE Offline after action | state-after-action | 03-next-save-offline-after-action.png | 0/0/0/0/0/0 | NEXT SAVE |
|  | NEXT SAVE Error | state | dup of 01 | ·/·/·/·/·/· | NEXT SAVE |
| 04 | NEXT SAVE Error after action | state-after-action | 04-next-save-error-after-action.png | 0/0/0/0/0/0 | NEXT SAVE |
|  | ROLE Guardian | state | dup of 01 | ·/·/·/·/·/· | ROLE |
| 05 | ROLE Caregiver | state | 05-role-caregiver.png | 0/0/0/0/0/0 | ROLE |
| 06 | sheet More | sheet | 06-sheet-more.png | 0/0/·/·/·/· |  |

## Acceptance

- Every state above matches its baseline in all six variants (tolerance in `VERIFICATION.md`).
- axe clean per state; targets ≥ 44 px; no horizontal scroll at 390 px and 200 % zoom; at ≥ 1280 px the content block is 920 px wide and the left column precedes the right in the DOM (`DESIGN.md` §2.1).
- Accessible names equal the `Name` column (or are listed in `IMPLEMENTATION.md` section 6).
- The no-JS path in the notes above works with JavaScript disabled.
