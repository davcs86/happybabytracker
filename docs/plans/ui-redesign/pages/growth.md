# Growth

Prototype page: `Growth.dc.html`  ·  baselines: `screenshots/growth/<variant>/NN-<state>.png`  ·  layout data: `specs/growth.<variant>.layout.json`

## Routes and files (legacy repo)

| Route | Status |
|---|---|
| `/kids/[kid]/growth` | exists |
| `/kids/[kid]/growth/new` | exists |
| `/kids/[kid]/growth/[entry]` | exists |

Legacy files to restyle: `ui/src/routes/(app)/kids/[kid]/growth/+page.svelte`, `ui/src/routes/(app)/kids/[kid]/growth/new/+page.svelte`, `ui/src/routes/(app)/kids/[kid]/growth/[entry]/+page.svelte`, `ui/src/lib/components/GrowthChart.svelte`, `ui/src/lib/components/GrowthTable.svelte`

## Implementation notes carried over from the prototype

These are the design notes that sit in the prototype's own side panel (they are part of the design, the panel itself is not).

```text
shadcn-svelte parts
Tabs (measure), Card, Badge, Button, Collapsible (help, chart data, reference data), Sheet (measurement form, source), ToggleGroup (units), Input, Label, Textarea, Alert, Sonner.

No-JS baseline
Measure tabs are links (?measure=); the chart is server-built SVG; tapping a row or point is a link to ?sel=<id>; the form is its own route.

New vs as-built
Row ↔ point sync; labels only on 3 / 50 / 97 (other lines live in the table); plain-language help; both ages shown wherever corrected age applies.

Curves are illustrative, not WHO or CDC tables.
```

## Layout (measured from the rendered prototype, Geist loaded)

Phone frame 390 px wide, content height 1806 px; side-nav frame 980 px wide, content height 1986 px, two-column frame 1280 px wide, content height 1246 px. All numbers are CSS px at 1×.

**Headings (phone)**

| Level | Text | Box x,y w×h | font |
|---|---|---|---|
| h1 | Growth | 16,78 177×33 | 30px/600 |
| h2 |  | 16,257 358×16 | 12px/600 |
| h2 | Measurements | 16,731 358×16 | 12px/600 |

**Interactive elements (default state)**

| Name | Element | shadcn part | Phone x,y w×h | 768–1279 x,y w×h | ≥1280 x,y w×h | Radius | Font |
|---|---|---|---|---|---|---|---|
| Child: Leo. Switch child | link | Button | 16,16 112×48 | 241,16 112×48 | 292,16 112×48 | 12px | 15px/600 |
| Notification settings | link | Button | 248,16 48×48 | 835,16 48×48 | 1086,16 48×48 | 12px | 16px/400 |
| Add measurement | button |  | 203,121 171×48 | 790,104 171×48 | 1042,104 171×48 | 12px | 15px/600 |
| Select Sep 11, 2026 measurement | button |  | 78,424 44×44 | 382,568 44×44 | 374,448 44×44 |  | 13.3333px/400 |
| Select Oct 14, 2026 measurement | button |  | 126,395 44×44 | 483,507 44×44 | 436,410 44×44 |  | 13.3333px/400 |
| Select Nov 17, 2026 measurement | button |  | 175,374 44×44 | 584,462 44×44 | 497,383 44×44 |  | 13.3333px/400 |
| Edit the measurement of Nov 17, 2026 | button |  | 313,768 48×48 | 900,1017 48×48 | 1152,277 48×48 | 12px | 13.3333px/400 |
| Edit the measurement of Oct 14, 2026 | button |  | 313,930 48×48 | 900,1179 48×48 | 1152,439 48×48 | 12px | 13.3333px/400 |
| Edit the measurement of Sep 11, 2026 | button |  | 313,1092 48×48 | 900,1341 48×48 | 1152,601 48×48 | 12px | 13.3333px/400 |
| Edit the measurement of Aug 11, 2026 | button |  | 313,1274 48×48 | 900,1523 48×48 | 1152,783 48×48 | 12px | 13.3333px/400 |
| Edit the measurement of Jul 14, 2026 | button |  | 313,1436 48×48 | 900,1685 48×48 | 1152,945 48×48 | 12px | 13.3333px/400 |
| Change the growth source | button |  | 154,1597 170×44 | 378,1846 170×44 | 902,1106 170×44 |  | 14px/400 |

## States captured

`dup` = pixel-identical to an earlier state of the same variant (the control changed nothing visible); `axe` = violations found (0 = clean).

| # | State | Kind | Baseline (phone-light) | axe (pl/pd/dl/dd/wl/wd) | Notes |
|---|---|---|---|---|---|
| 01 | default-frame | frame | 01-default-frame.png | 0/0/0/0/0/0 |  |
| 02 | default-full-height | full | 02-default-full-height.png | 0/0/0/0/0/0 |  |
| 03 | CHILD Ivy | state | 03-child-ivy.png | 0/0/0/0/0/0 | CHILD |
| 04 | CHILD Mia | state | 04-child-mia.png | 0/0/0/0/0/0 | CHILD |
|  | CHILD Leo · preterm | state | dup of 02 | ·/·/·/·/·/· | CHILD |
|  | NEXT SAVE Works | state | dup of 02 | ·/·/·/·/·/· | NEXT SAVE |
| 05 | NEXT SAVE Works after action | state-after-action | 05-next-save-works-after-action.png | 0/0/0/0/0/0 | NEXT SAVE |
|  | NEXT SAVE Offline | state | dup of 02 | ·/·/·/·/·/· | NEXT SAVE |
|  | NEXT SAVE Offline after action | state-after-action | dup of 05 | ·/·/·/·/·/· | NEXT SAVE |
|  | NEXT SAVE Error | state | dup of 02 | ·/·/·/·/·/· | NEXT SAVE |
|  | NEXT SAVE Error after action | state-after-action | dup of 05 | ·/·/·/·/·/· | NEXT SAVE |
|  | SOURCE WHO | state | dup of 02 | ·/·/·/·/·/· | SOURCE |
| 06 | SOURCE CDC | state | 06-source-cdc.png | 0/0/0/0/0/0 | SOURCE |
|  | DATA Rows | state | dup of 02 | ·/·/·/·/·/· | DATA |
| 07 | DATA Empty | state | 07-data-empty.png | 0/0/0/0/0/0 | DATA |
|  | UNITS ml · kg | state | dup of 02 | ·/·/·/·/·/· | UNITS |
| 08 | UNITS fl oz · lb | state | 08-units-fl-oz-lb.png | 0/0/0/0/0/0 | UNITS |
|  | ROLE Guardian | state | dup of 02 | ·/·/·/·/·/· | ROLE |
| 09 | ROLE Caregiver | state | 09-role-caregiver.png | 0/0/0/0/0/0 | ROLE |
| 10 | sheet Switch child | sheet | 10-sheet-switch-child.png | 0/0/0/0/0/0 |  |
| 11 | sheet More | sheet | 11-sheet-more.png | 0/0/·/·/·/· |  |

## Acceptance

- Every state above matches its baseline in all six variants (tolerance in `VERIFICATION.md`).
- axe clean per state; targets ≥ 44 px; no horizontal scroll at 390 px and 200 % zoom; at ≥ 1280 px the content block is 920 px wide and the left column precedes the right in the DOM (`DESIGN.md` §2.1).
- Accessible names equal the `Name` column (or are listed in `IMPLEMENTATION.md` section 6).
- The no-JS path in the notes above works with JavaScript disabled.
