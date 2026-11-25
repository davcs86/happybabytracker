# Timeline

Prototype page: `Timeline.dc.html`  ·  baselines: `screenshots/timeline/<variant>/NN-<state>.png`  ·  layout data: `specs/timeline.<variant>.layout.json`

## Routes and files (legacy repo)

| Route | Status |
|---|---|
| `/kids/[kid]/timeline` | exists |
| `/kids/[kid]/log/[tracker]/[entry] (row edit)` | exists |

Legacy files to restyle: `ui/src/routes/(app)/kids/[kid]/timeline/+page.svelte`, `ui/src/routes/(app)/kids/[kid]/log/[tracker=tracker]/[entry]/+page.svelte`

## Implementation notes carried over from the prototype

These are the design notes that sit in the prototype's own side panel (they are part of the design, the panel itself is not).

```text
shadcn-svelte parts
Card, Badge, Button, ToggleGroup (type filter), Collapsible, Sheet (edit), Input, Label, Alert, Sonner.

Patterns from the Nara case study
Week strip (P16), time since last (P3), colour + icon + label per tracker (P18). See nara-case-study-design-patterns.md.

No-JS baseline
Filters are a GET form with a Show button; day buttons are links (?day=); rows are links to the edit route; “Older” is a link.

Open question
Undo after delete needs a client-held payload (re-POST) or a soft delete (costly).
```

## Layout (measured from the rendered prototype, Geist loaded)

Phone frame 390 px wide, content height 3296 px; side-nav frame 980 px wide, content height 3071 px, two-column frame 1280 px wide, content height 2759 px. All numbers are CSS px at 1×.

**Headings (phone)**

| Level | Text | Box x,y w×h | font |
|---|---|---|---|
| h1 | Timeline | 16,78 210×33 | 30px/600 |
| h2 | Last 7 days | 16,175 358×16 | 12px/600 |
| h2 | Show | 16,383 358×16 | 12px/600 |
| h2 |  | 16,573 358×21 | 16px/600 |
| h2 |  | 16,1521 358×21 | 16px/600 |
| h2 |  | 16,2337 358×21 | 16px/600 |

**Interactive elements (default state)**

| Name | Element | shadcn part | Phone x,y w×h | 768–1279 x,y w×h | ≥1280 x,y w×h | Radius | Font |
|---|---|---|---|---|---|---|---|
| Child: Leo. Switch child | link | Button | 16,16 112×48 | 241,16 112×48 | 292,16 112×48 | 12px | 15px/600 |
| Notification settings | link | Button | 248,16 48×48 | 835,16 48×48 | 1086,16 48×48 | 12px | 16px/400 |
| Log something | link |  | 236,88 138×48 | 823,88 138×48 | 1074,88 138×48 | 12px | 15px/600 |
| Wed, Nov 18: 2 bottles (250 ml) · 2 nursing (26  | button |  | 23,206 48×108 | 248,206 99×108 | 300,264 60×108 | 10px | 13.3333px/400 |
| Thu, Nov 19: 2 bottles (320 ml) · 2 nursing (26  | button |  | 72,206 48×108 | 349,206 99×108 | 362,264 60×108 | 10px | 13.3333px/400 |
| Fri, Nov 20: 3 bottles (430 ml) · 2 nursing (26  | button |  | 122,206 48×108 | 450,206 99×108 | 424,264 60×108 | 10px | 13.3333px/400 |
| Sat, Nov 21: 3 bottles (400 ml) · 4 diapers · 3  | button |  | 172,206 48×108 | 552,206 99×108 | 486,264 60×108 | 10px | 13.3333px/400 |
| Sun, Nov 22: 3 bottles (460 ml) · 2 nursing (26  | button |  | 220,206 48×108 | 652,206 99×108 | 548,264 60×108 | 10px | 13.3333px/400 |
| Yesterday: 2 bottles (290 ml) · 2 nursing (26 mi | button |  | 270,206 48×108 | 754,206 99×108 | 611,264 60×108 | 10px | 13.3333px/400 |
| Today: 3 bottles (400 ml) · 2 nursing (26 min) · | button |  | 320,206 48×108 | 855,206 99×108 | 673,264 60×108 | 10px | 13.3333px/400 |
| Bottle, 130 ml, breast milk + formula at 17:50.  | button | Card | 16,638 358×60 | 241,549 720×60 | 764,220 448×60 | 14px | 13.3333px/400 |
| Diaper dirty at 17:00. Edit entry. | button | Card | 16,704 358×60 | 241,615 720×60 | 764,286 448×60 | 14px | 13.3333px/400 |
| Nursing, L 6 min · R 7 min at 16:20. Edit entry. | button | Card | 16,770 358×60 | 241,681 720×60 | 764,352 448×60 | 14px | 13.3333px/400 |
| Pumping, 15 min, 90 ml (L 50 ml · R 40 ml) at 14 | button | Card | 16,836 358×60 | 241,747 720×60 | 764,418 448×60 | 14px | 13.3333px/400 |
| Diaper wet at 14:10. Edit entry. | button | Card | 16,902 358×60 | 241,813 720×60 | 764,484 448×60 | 14px | 13.3333px/400 |
| Bottle, 150 ml, formula at 13:10. Edit entry. | button | Card | 16,968 358×60 | 241,879 720×60 | 764,550 448×60 | 14px | 13.3333px/400 |
| Sleep, 12:10–13:55 (1 h 45 min) at 12:10. Edit e | button | Card | 16,1034 358×60 | 241,945 720×60 | 764,616 448×60 | 14px | 13.3333px/400 |
| Solids, Banana · liked it at 11:20. Edit entry. | button | Card | 16,1100 358×60 | 241,1011 720×60 | 764,682 448×60 | 14px | 13.3333px/400 |
| Diaper both at 10:30. Edit entry. | button | Card | 16,1166 358×60 | 241,1077 720×60 | 764,748 448×60 | 14px | 13.3333px/400 |
| Nursing, L 8 min · R 5 min, weight gain 60 g, ab | button | Card | 16,1232 358×71 | 241,1143 720×60 | 764,814 448×60 | 14px | 13.3333px/400 |
| Sleep, 07:40–09:10 (1 h 30 min) at 07:40. Edit e | button | Card | 16,1309 358×60 | 241,1209 720×60 | 764,880 448×60 | 14px | 13.3333px/400 |
| Bottle, 120 ml, breast milk at 07:00. Edit entry | button | Card | 16,1375 358×60 | 241,1275 720×60 | 764,946 448×60 | 14px | 13.3333px/400 |
| Diaper wet at 06:35. Edit entry. | button | Card | 16,1441 358×60 | 241,1341 720×60 | 764,1012 448×60 | 14px | 13.3333px/400 |
| Diaper dirty at 17:07. Edit entry. | button | Card | 16,1586 358×60 | 241,1469 720×60 | 764,1140 448×60 | 14px | 13.3333px/400 |
| Nursing, L 6 min · R 7 min at 16:27. Edit entry. | button | Card | 16,1652 358×60 | 241,1535 720×60 | 764,1206 448×60 | 14px | 13.3333px/400 |
| Pumping, 15 min, 90 ml (L 50 ml · R 40 ml) at 14 | button | Card | 16,1718 358×60 | 241,1601 720×60 | 764,1272 448×60 | 14px | 13.3333px/400 |
| Diaper wet at 14:17. Edit entry. | button | Card | 16,1784 358×60 | 241,1667 720×60 | 764,1338 448×60 | 14px | 13.3333px/400 |
| Bottle, 160 ml, formula at 13:17. Edit entry. | button | Card | 16,1850 358×60 | 241,1733 720×60 | 764,1404 448×60 | 14px | 13.3333px/400 |
| Sleep, 12:17–14:17 (2 h) at 12:17. Edit entry. | button | Card | 16,1916 358×60 | 241,1799 720×60 | 764,1470 448×60 | 14px | 13.3333px/400 |
| Diaper both at 10:37. Edit entry. | button | Card | 16,1982 358×60 | 241,1865 720×60 | 764,1536 448×60 | 14px | 13.3333px/400 |
| Nursing, L 8 min · R 5 min, weight gain 40 g, ab | button | Card | 16,2048 358×71 | 241,1931 720×60 | 764,1602 448×60 | 14px | 13.3333px/400 |
| Sleep, 07:47–09:32 (1 h 45 min) at 07:47. Edit e | button | Card | 16,2125 358×60 | 241,1997 720×60 | 764,1668 448×60 | 14px | 13.3333px/400 |
| Bottle, 130 ml, breast milk at 07:07. Edit entry | button | Card | 16,2191 358×60 | 241,2063 720×60 | 764,1734 448×60 | 14px | 13.3333px/400 |
| Diaper wet at 06:42. Edit entry. | button | Card | 16,2257 358×60 | 241,2129 720×60 | 764,1800 448×60 | 14px | 13.3333px/400 |
| Bottle, 150 ml, breast milk + formula at 18:04.  | button | Card | 16,2402 358×60 | 241,2257 720×60 | 764,1944 448×60 | 14px | 13.3333px/400 |
| Nursing, L 6 min · R 7 min at 16:34. Edit entry. | button | Card | 16,2468 358×60 | 241,2323 720×60 | 764,2010 448×60 | 14px | 13.3333px/400 |
| Pumping, 15 min, 90 ml (L 50 ml · R 40 ml) at 15 | button | Card | 16,2534 358×60 | 241,2389 720×60 | 764,2076 448×60 | 14px | 13.3333px/400 |
| Diaper wet at 14:24. Edit entry. | button | Card | 16,2600 358×60 | 241,2455 720×60 | 764,2142 448×60 | 14px | 13.3333px/400 |
| Bottle, 170 ml, formula at 13:24. Edit entry. | button | Card | 16,2666 358×60 | 241,2521 720×60 | 764,2208 448×60 | 14px | 13.3333px/400 |
| Sleep, 12:24–14:09 (1 h 45 min) at 12:24. Edit e | button | Card | 16,2732 358×60 | 241,2587 720×60 | 764,2274 448×60 | 14px | 13.3333px/400 |
| Solids, Banana · liked it at 11:34. Edit entry. | button | Card | 16,2798 358×60 | 241,2653 720×60 | 764,2340 448×60 | 14px | 13.3333px/400 |
| Nursing, L 8 min · R 5 min, weight gain 64 g, ab | button | Card | 16,2864 358×71 | 241,2719 720×60 | 764,2406 448×60 | 14px | 13.3333px/400 |
| Sleep, 07:54–09:24 (1 h 30 min) at 07:54. Edit e | button | Card | 16,2941 358×60 | 241,2785 720×60 | 764,2472 448×60 | 14px | 13.3333px/400 |
| Bottle, 140 ml, breast milk at 07:14. Edit entry | button | Card | 16,3007 358×60 | 241,2851 720×60 | 764,2538 448×60 | 14px | 13.3333px/400 |
| Diaper wet at 06:49. Edit entry. | button | Card | 16,3073 358×60 | 241,2917 720×60 | 764,2604 448×60 | 14px | 13.3333px/400 |
| Older | button |  | 16,3151 358×48 | 241,2995 720×48 | 764,2682 448×48 | 12px | 15px/600 |

## States captured

`dup` = pixel-identical to an earlier state of the same variant (the control changed nothing visible); `axe` = violations found (0 = clean).

| # | State | Kind | Baseline (phone-light) | axe (pl/pd/dl/dd/wl/wd) | Notes |
|---|---|---|---|---|---|
| 01 | default-frame | frame | 01-default-frame.png | 0/0/0/0/0/0 |  |
| 02 | default-full-height | full | 02-default-full-height.png | 0/0/0/0/0/0 |  |
| 03 | CHILD Ivy | state | 03-child-ivy.png | 0/0/0/0/0/0 | CHILD |
| 04 | CHILD Mia | state | 04-child-mia.png | 0/0/0/0/0/0 | CHILD |
|  | CHILD Leo | state | dup of 02 | ·/·/·/·/·/· | CHILD |
|  | ROLE Guardian | state | dup of 02 | ·/·/·/·/·/· | ROLE |
| 05 | ROLE Caregiver | state | 05-role-caregiver.png | 0/0/0/0/0/0 | ROLE |
|  | NEXT SAVE Works | state | dup of 02 | ·/·/·/·/·/· | NEXT SAVE |
|  | NEXT SAVE Offline | state | dup of 02 | ·/·/·/·/·/· | NEXT SAVE |
|  | NEXT SAVE Error | state | dup of 02 | ·/·/·/·/·/· | NEXT SAVE |
|  | NEXT SAVE Clash | state | dup of 02 | ·/·/·/·/·/· | NEXT SAVE |
|  | UNITS ml | state | dup of 02 | ·/·/·/·/·/· | UNITS |
| 06 | UNITS fl oz | state | 06-units-fl-oz.png | 0/0/0/0/0/0 | UNITS |
| 07 | sheet Switch child | sheet | 07-sheet-switch-child.png | 0/0/0/0/0/0 |  |
| 08 | sheet More | sheet | 08-sheet-more.png | 0/0/·/·/·/· |  |

## Acceptance

- Every state above matches its baseline in all six variants (tolerance in `VERIFICATION.md`).
- axe clean per state; targets ≥ 44 px; no horizontal scroll at 390 px and 200 % zoom; at ≥ 1280 px the content block is 920 px wide and the left column precedes the right in the DOM (`DESIGN.md` §2.1).
- Accessible names equal the `Name` column (or are listed in `IMPLEMENTATION.md` section 6).
- The no-JS path in the notes above works with JavaScript disabled.
