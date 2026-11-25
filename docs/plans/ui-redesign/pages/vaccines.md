# Vaccines

Prototype page: `Vaccines.dc.html`  ·  baselines: `screenshots/vaccines/<variant>/NN-<state>.png`  ·  layout data: `specs/vaccines.<variant>.layout.json`

## Routes and files (legacy repo)

| Route | Status |
|---|---|
| `/kids/[kid]/vaccines` | exists |
| `/kids/[kid]/vaccines/new` | exists |
| `/kids/[kid]/vaccines/[administration]` | exists |

Legacy files to restyle: `ui/src/routes/(app)/kids/[kid]/vaccines/+page.svelte`, `ui/src/routes/(app)/kids/[kid]/vaccines/new/+page.svelte`, `ui/src/routes/(app)/kids/[kid]/vaccines/[administration]/+page.svelte`

## Implementation notes carried over from the prototype

These are the design notes that sit in the prototype's own side panel (they are part of the design, the panel itself is not).

```text
shadcn-svelte parts
Card, Badge, Button, Alert, Sheet (record, dismiss, schedule, edit), Checkbox, Select, Input, Label, Collapsible (visit rows, About), RadioGroup, Sonner (undo).

No-JS baseline
Every sheet is its own route (/vaccines/new?visit=…); each visit is a <details>; Dismiss and Undo are form POSTs; the status line is shown after the redirect.

Changed on purpose
Visit cards group antigens by schedule group (one 6-in-1 card, not six rows). Banner and button say “Dismiss older vaccines” (was “lines”). Window text uses ages and short dates, no ISO.

Sample data, simplified. Not a real schedule.
```

## Layout (measured from the rendered prototype, Geist loaded)

Phone frame 390 px wide, content height 2100 px; side-nav frame 980 px wide, content height 1997 px, two-column frame 1280 px wide, content height 1835 px. All numbers are CSS px at 1×.

**Headings (phone)**

| Level | Text | Box x,y w×h | font |
|---|---|---|---|
| h1 | Vaccines | 16,86 246×33 | 30px/600 |
| h2 |  | 16,265 358×16 | 12px/600 |
| h3 |  | 31,308 206×21 | 16px/600 |
| h3 |  | 31,822 232×21 | 16px/600 |
| h2 |  | 16,1208 358×16 | 12px/600 |
| h3 |  | 31,1250 197×21 | 16px/600 |
| h3 |  | 31,1689 197×42 | 16px/600 |
| h2 |  | 16,1759 358×16 | 12px/600 |
| h3 |  | 31,1802 222×21 | 16px/600 |
| h2 | Every season | 16,1859 358×16 | 12px/600 |

**Interactive elements (default state)**

| Name | Element | shadcn part | Phone x,y w×h | 768–1279 x,y w×h | ≥1280 x,y w×h | Radius | Font |
|---|---|---|---|---|---|---|---|
| Child: Leo. Switch child | link | Button | 16,16 112×48 | 241,16 112×48 | 292,16 112×48 | 12px | 15px/600 |
| Notification settings | link | Button | 248,16 48×48 | 835,16 48×48 | 1086,16 48×48 | 12px | 16px/400 |
| About this schedule | button | Button | 270,78 48×48 | 857,78 48×48 | 1108,78 48×48 | 12px | 13.3333px/400 |
| Schedule options | button | Button | 326,78 48×48 | 913,78 48×48 | 1164,78 48×48 | 12px | 13.3333px/400 |
| Record a vaccination | button |  | 16,191 358×48 | 241,191 720×48 | 292,191 448×48 | 12px | 15px/600 |
| Edit 6-in-1 · dose 2, given 2026-10-07 | button |  | 42,419 110×44 | 267,402 110×44 | 318,402 110×44 |  | 13.5px/400 |
| Edit Pneumococcal · dose 1, given 2026-10-07 | button |  | 42,540 110×44 | 267,523 110×44 | 318,523 110×44 |  | 13.5px/400 |
| Dismiss Rotavirus · dose 2 | button |  | 250,661 98×48 | 836,644 98×48 | 616,644 98×48 | 12px | 15px/600 |
| Record 12 weeks vaccines | button |  | 29,730 332×48 | 254,713 694×48 | 306,713 422×48 | 12px | 15px/600 |
| Dismiss 6-in-1 · dose 3 | button |  | 250,933 98×48 | 836,899 98×48 | 616,899 98×48 | 12px | 15px/600 |
| Dismiss MenB · dose 2 | button |  | 250,1058 98×48 | 836,1024 98×48 | 616,1024 98×48 | 12px | 15px/600 |
| Record 16 weeks vaccines | button |  | 29,1127 332×48 | 254,1093 694×48 | 306,1093 422×48 | 12px | 15px/600 |
| Dismiss MMR · dose 1 | button |  | 250,1345 98×48 | 836,1311 98×48 | 616,1311 98×48 | 12px | 15px/600 |
| Dismiss Pneumococcal · dose 2 | button |  | 250,1470 98×48 | 836,1436 98×48 | 616,1436 98×48 | 12px | 15px/600 |
| Dismiss MenB · dose 3 | button |  | 250,1595 98×48 | 836,1561 98×48 | 616,1561 98×48 | 12px | 15px/600 |
| Dismiss MMR · dose 2 | button |  | 250,1794 98×48 | 836,1760 98×48 | 616,1760 98×48 | 12px | 15px/600 |
| Dismiss 4-in-1 booster · dose 1 | button |  | 250,1919 98×48 | 836,1885 98×48 | 616,1885 98×48 | 12px | 15px/600 |
| Edit 6-in-1 · dose 1, given 2026-09-09 | button |  | 42,1913 114×44 | 267,1862 114×44 | 318,1862 114×44 |  | 13.5px/400 |
| Edit Rotavirus · dose 1, given 2026-09-09 | button |  | 42,2034 114×44 | 267,1983 114×44 | 318,1983 114×44 |  | 13.5px/400 |
| Edit MenB · dose 1, given 2026-09-09 | button |  | 42,2155 114×44 | 267,2104 114×44 | 318,2104 114×44 |  | 13.5px/400 |

## States captured

`dup` = pixel-identical to an earlier state of the same variant (the control changed nothing visible); `axe` = violations found (0 = clean).

| # | State | Kind | Baseline (phone-light) | axe (pl/pd/dl/dd/wl/wd) | Notes |
|---|---|---|---|---|---|
| 01 | default-frame | frame | 01-default-frame.png | 0/0/0/0/0/0 |  |
| 02 | default-full-height | full | 02-default-full-height.png | 0/0/0/0/0/0 |  |
|  | NEXT SAVE Works | state | dup of 02 | ·/·/·/·/·/· | NEXT SAVE |
|  | NEXT SAVE Offline | state | dup of 02 | ·/·/·/·/·/· | NEXT SAVE |
|  | NEXT SAVE Error | state | dup of 02 | ·/·/·/·/·/· | NEXT SAVE |
|  | SCHEDULE England | state | dup of 02 | ·/·/·/·/·/· | SCHEDULE |
| 03 | SCHEDULE US | state | 03-schedule-us.png | 0/0/0/0/0/0 | SCHEDULE |
| 04 | SCHEDULE None | state | 04-schedule-none.png | 0/0/0/0/0/0 | SCHEDULE |
|  | AGE Stored | state | dup of 02 | ·/·/·/·/·/· | AGE |
| 05 | AGE 3 wk | state | 05-age-3-wk.png | 0/0/0/0/0/0 | AGE |
| 06 | AGE 5 mo | state | 06-age-5-mo.png | 0/0/0/0/0/0 | AGE |
| 07 | AGE 14 mo | state | 07-age-14-mo.png | 0/0/0/0/0/0 | AGE |
| 08 | AGE 4 y | state | 08-age-4-y.png | 0/0/0/0/0/0 | AGE |
|  | RECORDS Sample | state | dup of 02 | ·/·/·/·/·/· | RECORDS |
|  | RECORDS None | state | dup of 04 | ·/·/·/·/·/· | RECORDS |
| 09 | RECORDS Some | state | 09-records-some.png | 0/0/0/0/0/0 | RECORDS |
|  | RECORDS Caught up | state | dup of 09 | ·/·/·/·/·/· | RECORDS |
|  | ROLE Guardian | state | dup of 02 | ·/·/·/·/·/· | ROLE |
| 10 | ROLE Caregiver | state | 10-role-caregiver.png | 0/0/0/0/0/0 | ROLE |
| 11 | sheet Switch child | sheet | 11-sheet-switch-child.png | 0/0/0/0/0/0 |  |
| 12 | sheet More | sheet | 12-sheet-more.png | 0/0/·/·/·/· |  |

## Acceptance

- Every state above matches its baseline in all six variants (tolerance in `VERIFICATION.md`).
- axe clean per state; targets ≥ 44 px; no horizontal scroll at 390 px and 200 % zoom; at ≥ 1280 px the content block is 920 px wide and the left column precedes the right in the DOM (`DESIGN.md` §2.1).
- Accessible names equal the `Name` column (or are listed in `IMPLEMENTATION.md` section 6).
- The no-JS path in the notes above works with JavaScript disabled.
