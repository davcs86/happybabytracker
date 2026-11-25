# Sickness: log a symptom

Prototype page: `Symptom.dc.html`  ·  baselines: `screenshots/symptom/<variant>/NN-<state>.png`  ·  layout data: `specs/symptom.<variant>.layout.json`

## Routes and files (legacy repo)

| Route | Status |
|---|---|
| `/kids/[kid]/log/symptom` | exists (tracker route; confirm the symptom tracker id) |
| `/kids/[kid]/log/symptom/[entry]` | exists (tracker route; confirm the symptom tracker id) |

Legacy files to restyle: `ui/src/routes/(app)/kids/[kid]/log/[tracker=tracker]/+page.svelte`, `ui/src/routes/(app)/kids/[kid]/log/[tracker=tracker]/[entry]/+page.svelte`

## Implementation notes carried over from the prototype

These are the design notes that sit in the prototype's own side panel (they are part of the design, the panel itself is not).

```text
shadcn-svelte parts
Field + Label + Input, native Select (h-12 here, see notes), RadioGroup (fieldset), Button, Alert, Collapsible (<details>).

No-JS
Full POST, 303 to the timeline day; hidden fever/other fields are ignored on save.

Sample data
Switch °C/°F to see the same entry shown from the API’s other column. The UI never converts.
```

## Layout (measured from the rendered prototype, Geist loaded)

Phone frame 390 px wide, content height 808 px; side-nav frame 980 px wide, content height 780 px, two-column frame 1280 px wide, content height 780 px. All numbers are CSS px at 1×.

**Headings (phone)**

| Level | Text | Box x,y w×h | font |
|---|---|---|---|
| h1 |  | 16,90 358×33 | 30px/600 |

**Interactive elements (default state)**

| Name | Element | shadcn part | Phone x,y w×h | 768–1279 x,y w×h | ≥1280 x,y w×h | Radius | Font |
|---|---|---|---|---|---|---|---|
| Notification settings | link | Button | 248,16 48×48 | 835,16 48×48 | 1086,16 48×48 | 12px | 16px/400 |
| f-time | input |  | 16,214 358×48 | 241,214 448×48 | 292,214 448×48 | 8px | 15px/400 |
| f-sev | input |  | 16,394 20×20 | 241,394 20×20 | 292,394 20×20 | 50% | 13.3333px/400 |
| f-note | input |  | 16,601 358×48 | 241,601 448×48 | 764,214 448×48 | 8px | 15px/400 |
| Save | button | Button | 16,663 266×48 | 241,663 356×48 | 292,578 356×48 | 12px | 15px/600 |
| Cancel | link |  | 290,663 84×48 | 605,663 84×48 | 656,578 84×48 | 12px | 15px/600 |

## States captured

`dup` = pixel-identical to an earlier state of the same variant (the control changed nothing visible); `axe` = violations found (0 = clean).

| # | State | Kind | Baseline (phone-light) | axe (pl/pd/dl/dd/wl/wd) | Notes |
|---|---|---|---|---|---|
| 01 | default-frame | frame | 01-default-frame.png | 0/0/0/0/0/0 |  |
| 02 | default-full-height | full | 02-default-full-height.png | 0/0/·/·/·/· |  |
|  | NEXT SAVE Works | state | dup of 02 | ·/·/·/·/0/· | NEXT SAVE |
| 03 | NEXT SAVE Works after action | state-after-action | 03-next-save-works-after-action.png | 0/0/0/0/0/0 | NEXT SAVE |
|  | NEXT SAVE Offline | state | dup of 02 | ·/·/·/·/·/· | NEXT SAVE |
| 04 | NEXT SAVE Offline after action | state-after-action | 04-next-save-offline-after-action.png | 0/0/0/0/0/0 | NEXT SAVE |
|  | NEXT SAVE Error | state | dup of 02 | ·/·/·/·/·/· | NEXT SAVE |
| 05 | NEXT SAVE Error after action | state-after-action | 05-next-save-error-after-action.png | 0/0/0/0/0/0 | NEXT SAVE |
|  | SCREEN New | state | dup of 02 | ·/·/·/·/·/· | SCREEN |
| 06 | SCREEN Edit | state | 06-screen-edit.png | 0/0/0/0/0/0 | SCREEN |
|  | MY TEMPERATURE UNIT °C | state | dup of 02 | ·/·/·/·/·/0 | MY TEMPERATURE UNIT |
|  | MY TEMPERATURE UNIT °F | state | dup of 02 | ·/·/·/·/·/· | MY TEMPERATURE UNIT |
|  | ROLE Guardian | state | dup of 02 | ·/·/0/·/·/· | ROLE |
| 07 | ROLE Caregiver | state | 07-role-caregiver.png | 0/0/0/0/0/0 | ROLE |
| 08 | sheet More | sheet | 08-sheet-more.png | 0/0/·/·/·/· |  |

## Acceptance

- Every state above matches its baseline in all six variants (tolerance in `VERIFICATION.md`).
- axe clean per state; targets ≥ 44 px; no horizontal scroll at 390 px and 200 % zoom; at ≥ 1280 px the content block is 920 px wide and the left column precedes the right in the DOM (`DESIGN.md` §2.1).
- Accessible names equal the `Name` column (or are listed in `IMPLEMENTATION.md` section 6).
- The no-JS path in the notes above works with JavaScript disabled.
