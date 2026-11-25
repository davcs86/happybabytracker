# Medication

Prototype page: `Medication.dc.html`  ·  baselines: `screenshots/medication/<variant>/NN-<state>.png`  ·  layout data: `specs/medication.<variant>.layout.json`

## Routes and files (legacy repo)

| Route | Status |
|---|---|
| `/kids/[kid]/medication` | exists |

Legacy files to restyle: `ui/src/routes/(app)/kids/[kid]/medication/+page.svelte`

## Implementation notes carried over from the prototype

These are the design notes that sit in the prototype's own side panel (they are part of the design, the panel itself is not).

```text
shadcn-svelte parts
Card, Button, Input, Label, Collapsible (<details>), Alert.

Try
Switch child to Ivy for the empty state. Type 81 characters in an Amount. Use “Course gone” or “Dose cap”. Turn on “Next save: Offline”.

No-JS
Every action is a POST that 303s back here; the filter is kept in the query string.

Labelled variant
“Larger, one-hand” is a UI-only proposal; it also needs a status line after the redirect (not drawn).
```

## Layout (measured from the rendered prototype, Geist loaded)

Phone frame 390 px wide, content height 1666 px; side-nav frame 980 px wide, content height 1597 px, two-column frame 1280 px wide, content height 1210 px. All numbers are CSS px at 1×.

**Headings (phone)**

| Level | Text | Box x,y w×h | font |
|---|---|---|---|
| h1 | Medication | 16,90 210×33 | 30px/600 |
| h2 | Courses | 16,191 358×23 | 18px/600 |
| h3 |  | 87,244 212×22 | 17px/600 |
| h3 |  | 87,548 212×22 | 17px/600 |
| h3 |  | 87,854 212×22 | 17px/600 |
| h3 |  | 100,1196 186×22 | 17px/600 |
| h3 |  | 100,1482 186×22 | 17px/600 |
| h2 |  | 16,1196 358×23 | 18px/600 |

**Interactive elements (default state)**

| Name | Element | shadcn part | Phone x,y w×h | 768–1279 x,y w×h | ≥1280 x,y w×h | Radius | Font |
|---|---|---|---|---|---|---|---|
| Child: Leo. Switch child | link | Button | 16,16 112×48 | 241,16 112×48 | 292,16 112×48 | 12px | 15px/600 |
| Notification settings | link | Button | 248,16 48×48 | 835,16 48×48 | 1086,16 48×48 | 12px | 16px/400 |
| Add a course | link |  | 238,90 136×48 | 826,90 136×48 | 1077,90 136×48 | 12px | 15px/600 |
| Edit Amoxil | link | Button | 311,241 48×48 | 898,241 48×48 | 678,243 48×48 | 12px | 16px/400 |
| Amount (optional) | label |  | 31,373 328×17 | 256,373 690×17 | 308,375 418×17 |  | 13px/400 |
| amt-c2 | input |  | 31,396 328×48 | 256,396 690×48 | 308,398 418×48 | 8px | 15px/400 |
| Log dose now of Amoxil | button | Button | 31,456 138×48 | 256,456 138×48 | 308,458 138×48 | 12px | 15px/600 |
| Show doses of Amoxil | link | Button | 178,456 121×48 | 402,456 121×48 | 454,458 121×48 | 12px | 15px/600 |
| Edit Calpol | link | Button | 311,546 48×48 | 898,546 48×48 | 678,548 48×48 | 12px | 16px/400 |
| amt-c1 | input |  | 31,702 328×48 | 256,702 690×48 | 308,704 418×48 | 8px | 15px/400 |
| Log dose now of Calpol | button | Button | 31,762 138×48 | 256,762 138×48 | 308,764 138×48 | 12px | 15px/600 |
| Show doses of Calpol | link | Button | 178,762 121×48 | 402,762 121×48 | 454,764 121×48 | 12px | 15px/600 |
| Edit Vitamin D drops | link | Button | 311,852 48×48 | 898,852 48×48 | 678,854 48×48 | 12px | 16px/400 |
| amt-c3 | input |  | 31,986 328×48 | 256,986 690×48 | 308,988 418×48 | 8px | 15px/400 |
| Log dose now of Vitamin D drops | button | Button | 31,1046 138×48 | 256,1046 138×48 | 308,1048 138×48 | 12px | 15px/600 |
| Show doses of Vitamin D drops | link | Button | 178,1046 121×48 | 402,1046 121×48 | 454,1048 121×48 | 12px | 15px/600 |
| Edit Nurofen | link | Button | 298,1194 48×48 | 885,1194 48×48 | 664,1196 48×48 | 12px | 16px/400 |
| amt-c4 | input |  | 44,1329 302×48 | 269,1329 664×48 | 320,1331 392×48 | 8px | 15px/400 |
| Log dose now of Nurofen | button | Button | 44,1389 138×48 | 269,1389 138×48 | 320,1391 138×48 | 12px | 15px/600 |
| Show doses of Nurofen | link | Button | 190,1389 121×48 | 416,1389 121×48 | 467,1391 121×48 | 12px | 15px/600 |
| Edit Saline spray | link | Button | 298,1479 48×48 | 885,1479 48×48 | 664,1481 48×48 | 12px | 16px/400 |
| amt-c5 | input |  | 44,1614 302×48 | 269,1614 664×48 | 320,1616 392×48 | 8px | 15px/400 |
| Log dose now of Saline spray | button | Button | 44,1674 138×48 | 269,1674 138×48 | 320,1676 138×48 | 12px | 15px/600 |
| Show doses of Saline spray | link | Button | 190,1674 121×48 | 416,1674 121×48 | 467,1676 121×48 | 12px | 15px/600 |
| Amoxil, 13:21 | link | Button | 16,1228 358×60 | n/a | n/a | 14px | 16px/400 |
| Calpol, 08:05 | link | Button | 16,1298 358×60 | 241,1298 720×60 | 764,296 448×60 | 14px | 16px/400 |
| Calpol, 2026-11-23 20:05 | link | Button | 16,1368 358×60 | 241,1368 720×60 | 764,366 448×60 | 14px | 16px/400 |
| Vitamin D drops, 2026-11-23 09:00 | link | Button | 16,1438 358×60 | 241,1438 720×60 | 764,436 448×60 | 14px | 16px/400 |
| Nurofen, 2026-11-19 09:00 | link | Button | 16,1508 358×60 | 241,1508 720×60 | 764,506 448×60 | 14px | 16px/400 |

## States captured

`dup` = pixel-identical to an earlier state of the same variant (the control changed nothing visible); `axe` = violations found (0 = clean).

| # | State | Kind | Baseline (phone-light) | axe (pl/pd/dl/dd/wl/wd) | Notes |
|---|---|---|---|---|---|
| 01 | default-frame | frame | 01-default-frame.png | 0/0/0/0/0/0 |  |
| 02 | default-full-height | full | 02-default-full-height.png | 0/0/0/0/0/0 |  |
|  | NEXT SAVE Works | state | dup of 02 | ·/·/·/·/·/· | NEXT SAVE |
|  | NEXT SAVE Offline | state | dup of 02 | ·/·/·/·/·/· | NEXT SAVE |
|  | NEXT SAVE Error | state | dup of 02 | ·/·/·/·/·/· | NEXT SAVE |
| 03 | CHILD Ivy | state | 03-child-ivy.png | 0/0/0/0/0/0 | CHILD |
| 04 | CHILD Mia | state | 04-child-mia.png | 0/0/0/0/0/0 | CHILD |
|  | CHILD Leo | state | dup of 02 | ·/·/·/0/·/· | CHILD |
|  | Works | state | dup of 02 | ·/·/·/·/·/· |  |
|  | Course gone | state | dup of 02 | ·/·/·/·/·/· |  |
|  | Dose cap | state | dup of 02 | ·/·/·/·/·/· |  |
|  | As built (32 px) | state | dup of 02 | ·/·/·/·/·/· |  |
| 05 | Larger, one-hand | state | 05-larger-one-hand.png | 0/0/0/0/0/0 |  |
|  | ROLE Guardian | state | dup of 02 | ·/·/·/·/·/0 | ROLE |
| 06 | ROLE Caregiver | state | 06-role-caregiver.png | 0/0/0/0/0/0 | ROLE |
| 07 | sheet Switch child | sheet | 07-sheet-switch-child.png | 0/0/0/0/0/0 |  |
| 08 | sheet Add a course | sheet | 08-sheet-add-a-course.png | 0/0/0/0/0/0 |  |
| 09 | sheet Edit course | sheet | 09-sheet-edit-course.png | 0/0/0/0/0/0 |  |
| 10 | sheet Edit dose | sheet | 10-sheet-edit-dose.png | 0/0/0/0/0/0 |  |
| 11 | sheet More | sheet | 11-sheet-more.png | 0/0/·/·/·/· |  |

## Acceptance

- Every state above matches its baseline in all six variants (tolerance in `VERIFICATION.md`).
- axe clean per state; targets ≥ 44 px; no horizontal scroll at 390 px and 200 % zoom; at ≥ 1280 px the content block is 920 px wide and the left column precedes the right in the DOM (`DESIGN.md` §2.1).
- Accessible names equal the `Name` column (or are listed in `IMPLEMENTATION.md` section 6).
- The no-JS path in the notes above works with JavaScript disabled.
