# Home

Prototype page: `Kid Home.dc.html`  ·  baselines: `screenshots/kid-home/<variant>/NN-<state>.png`  ·  layout data: `specs/kid-home.<variant>.layout.json`

## Routes and files (legacy repo)

| Route | Status |
|---|---|
| `/kids/[kid]` | exists |
| `/ (no children yet: Welcome state)` | exists |
| `/kids/[kid]/log/[tracker] (tile forms)` | exists |

Legacy files to restyle: `ui/src/routes/(app)/kids/[kid]/+page.svelte`, `ui/src/routes/(app)/+page.svelte`, `ui/src/lib/components/QuickLog.svelte`, `ui/src/lib/components/RunningTimer.svelte`, `ui/src/lib/components/DueList.svelte`, `ui/src/lib/components/KidLinks.svelte`

## Implementation notes carried over from the prototype

These are the design notes that sit in the prototype's own side panel (they are part of the design, the panel itself is not).

```text
shadcn-svelte parts
Card (tiles, running, Due, Today), Badge, Button, Sheet (forms), Alert (overrun, errors), Input, Label, Switch (Modules), Toggle (pills), Sonner (undo).

No-JS baseline
Each tile is a link to its form route; Sleep / End / Switch side are form POSTs. Sheets, live timers and the undo toast are enhancements.
```

## Layout (measured from the rendered prototype, Geist loaded)

Phone frame 390 px wide, content height 1902 px; side-nav frame 980 px wide, content height 1212 px, two-column frame 1280 px wide, content height 1212 px. All numbers are CSS px at 1×.

**Headings (phone)**

| Level | Text | Box x,y w×h | font |
|---|---|---|---|
| h1 |  | 16,84 358×33 | 30px/600 |
| h2 | Quick log | 16,180 358×16 | 12px/600 |
| h2 | Latest | 16,964 252×16 | 12px/600 |
| h2 | Due | 16,1206 358×16 | 12px/600 |
| h2 | Today | 16,1526 358×16 | 12px/600 |

**Interactive elements (default state)**

| Name | Element | shadcn part | Phone x,y w×h | 768–1279 x,y w×h | ≥1280 x,y w×h | Radius | Font |
|---|---|---|---|---|---|---|---|
| Child: Leo. Switch child | link | Button | 16,16 112×48 | 241,16 112×48 | 292,16 112×48 | 12px | 15px/600 |
| Notification settings | link | Button | 248,16 48×48 | 835,16 48×48 | 1086,16 48×48 | 12px | 16px/400 |
| See the Timeline | link |  | 268,950 106×44 | 484,950 106×44 | 634,950 106×44 |  | 14px/400 |
| Bottle, 130 ml, breast milk + formula at 17:50. | link | Card | 16,1004 358×56 | 241,1004 349×56 | 292,1004 448×56 | 14px | 16px/400 |
| Diaper dirty at 17:00. | link | Card | 16,1066 358×56 | 241,1066 349×56 | 292,1066 448×56 | 14px | 16px/400 |
| Nursing, L 6 min · R 7 min at 16:20. | link | Card | 16,1128 358×56 | 241,1128 349×56 | 292,1128 448×56 | 14px | 16px/400 |
| Add a child | button |  | 16,1761 83×44 | 612,735 83×44 | 764,712 83×44 |  | 14px/400 |

## States captured

`dup` = pixel-identical to an earlier state of the same variant (the control changed nothing visible); `axe` = violations found (0 = clean).

| # | State | Kind | Baseline (phone-light) | axe (pl/pd/dl/dd/wl/wd) | Notes |
|---|---|---|---|---|---|
| 01 | default-frame | frame | 01-default-frame.png | 0/0/0/0/0/0 |  |
| 02 | default-full-height | full | 02-default-full-height.png | 0/0/0/0/0/0 |  |
|  | AGE BAND 0–11 mo | state | dup of 02 | ·/·/·/·/·/· | AGE BAND |
| 03 | AGE BAND 12–23 mo | state | 03-age-band-12-23-mo.png | 0/0/0/0/0/0 | AGE BAND |
| 04 | AGE BAND 24 mo + | state | 04-age-band-24-mo.png | 0/0/0/0/0/0 | AGE BAND |
|  | UNITS ml | state | dup of 02 | ·/·/·/·/·/· | UNITS |
| 05 | UNITS fl oz | state | 05-units-fl-oz.png | 0/0/0/0/0/0 | UNITS |
|  | ROLE Guardian | state | dup of 02 | ·/·/·/·/·/· | ROLE |
| 06 | ROLE Caregiver | state | 06-role-caregiver.png | 0/0/0/0/0/0 | ROLE |
|  | NEXT SAVE Works | state | dup of 02 | ·/·/·/0/·/· | NEXT SAVE |
| 07 | NEXT SAVE Works after action | state-after-action | 07-next-save-works-after-action.png | 0/0/0/0/0/0 | NEXT SAVE |
|  | NEXT SAVE Offline | state | dup of 02 | ·/·/·/·/·/· | NEXT SAVE |
|  | NEXT SAVE Offline after action | state-after-action | dup of 07 | ·/·/·/·/·/· | NEXT SAVE |
|  | NEXT SAVE Error | state | dup of 02 | ·/·/0/·/·/· | NEXT SAVE |
|  | NEXT SAVE Error after action | state-after-action | dup of 07 | ·/·/0/·/·/· | NEXT SAVE |
|  | NEXT SAVE Clash | state | dup of 02 | ·/·/0/·/·/· | NEXT SAVE |
|  | NEXT SAVE Clash after action | state-after-action | dup of 07 | ·/·/0/·/·/· | NEXT SAVE |
|  | CHILDREN Has children | state | dup of 02 | ·/·/·/·/·/· | CHILDREN |
| 08 | CHILDREN None yet | state | 08-children-none-yet.png | 0/0/0/0/0/0 | CHILDREN |
| 09 | Simulate: Cleo starts a sleep on her phone | state | 09-simulate-cleo-starts-a-sleep-on-her-phone.png | 0/0/0/0/0/0 |  |
| 10 | Simulate: Cleo starts a sleep on her phone after action | state-after-action | 10-simulate-cleo-starts-a-sleep-on-her-phone-after-.png | 0/0/0/0/0/0 |  |
| 11 | Simulate: a sleep left running for 26 h | state | 11-simulate-a-sleep-left-running-for-26-h.png | 0/0/0/0/0/0 |  |
| 12 | Simulate: a sleep left running for 26 h after action | state-after-action | 12-simulate-a-sleep-left-running-for-26-h-after-act.png | 0/0/0/0/0/0 |  |
|  | Medication link: always shown (as built) | state | dup of 02 | ·/0/0/0/·/0 |  |
|  | Medication link: always shown (as built) after action | state-after-action | dup of 07 | ·/0/0/0/·/0 |  |
| 13 | sheet Switch child | sheet | 13-sheet-switch-child.png | 0/0/0/0/0/0 |  |
| 14 | sheet More | sheet | 14-sheet-more.png | 0/0/·/·/·/· |  |

## Acceptance

- Every state above matches its baseline in all six variants (tolerance in `VERIFICATION.md`).
- axe clean per state; targets ≥ 44 px; no horizontal scroll at 390 px and 200 % zoom; at ≥ 1280 px the content block is 920 px wide and the left column precedes the right in the DOM (`DESIGN.md` §2.1).
- Accessible names equal the `Name` column (or are listed in `IMPLEMENTATION.md` section 6).
- The no-JS path in the notes above works with JavaScript disabled.
