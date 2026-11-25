# Child profile (details, settings, people with access)

Prototype page: `Child Profile.dc.html`  ·  baselines: `screenshots/child-profile/<variant>/NN-<state>.png`  ·  layout data: `specs/child-profile.<variant>.layout.json`

## Routes and files (legacy repo)

| Route | Status |
|---|---|
| `/kids/[kid]/settings` | exists. The prototype merges details, settings and access into one profile page with sheets. The sheets' no-JS pages are designed separately (Child Settings, Access). **Modules has no designed page** (sheet only): confirm its no-JS route with the owner before building |
| `/kids/[kid]/access` | exists. The prototype merges details, settings and access into one profile page with sheets. The sheets' no-JS pages are designed separately (Child Settings, Access). **Modules has no designed page** (sheet only): confirm its no-JS route with the owner before building |
| `/kids/[kid]/modules` | exists. The prototype merges details, settings and access into one profile page with sheets. The sheets' no-JS pages are designed separately (Child Settings, Access). **Modules has no designed page** (sheet only): confirm its no-JS route with the owner before building |

Legacy files to restyle: `ui/src/routes/(app)/kids/[kid]/settings/+page.svelte`, `ui/src/routes/(app)/kids/[kid]/access/+page.svelte`, `ui/src/routes/(app)/kids/[kid]/modules/+page.svelte`

## Implementation notes carried over from the prototype

These are the design notes that sit in the prototype's own side panel (they are part of the design, the panel itself is not).

```text
shadcn-svelte parts
Card, Badge, Button, Alert, Sheet (details, add person), AlertDialog (remove access, both delete steps), Input, Label, Checkbox, Sonner (undo for role change, remove, add).

No-JS baseline
Every sheet is a route (/children/:id/edit, /access/:person/remove, /delete, /delete/confirm). Step 2 is a separate page; the server checks the typed name, so the disabled button is a convenience.

Facts applied
Caregivers get no child profile. Delete is permanent and immediate, confirmed by typing the name. The time zone is editable by guardians. The last active guardian’s role and remove controls show disabled with a line; the API stays the authority (a click still returns its refusal).

Assumptions
1) “Read-only for caregivers” is a drawn variant, needs an owner decision. 2) “Export a PDF first” is a placeholder until export exists; a grace period is an open owner question. 3) Adding a person is by an existing parent’s username (the app has no email). 4) The time-zone notice has no extra confirmation. 5) Refusal wording is mine.

Sample data. The prototype keeps showing times in the original zone after a change.
```

## Layout (measured from the rendered prototype, Geist loaded)

Phone frame 390 px wide, content height 1773 px; side-nav frame 980 px wide, content height 1646 px, two-column frame 1280 px wide, content height 991 px. All numbers are CSS px at 1×.

**Headings (phone)**

| Level | Text | Box x,y w×h | font |
|---|---|---|---|
| h1 |  | 16,90 358×33 | 30px/600 |

**Interactive elements (default state)**

| Name | Element | shadcn part | Phone x,y w×h | 768–1279 x,y w×h | ≥1280 x,y w×h | Radius | Font |
|---|---|---|---|---|---|---|---|
| Child: Leo. Switch child | link | Button | 16,16 112×48 | 241,16 112×48 | 292,16 112×48 | 12px | 15px/600 |
| Notification settings | link | Button | 248,16 48×48 | 835,16 48×48 | 1086,16 48×48 | 12px | 16px/400 |
| Edit details | button | Button | 234,210 122×48 | 822,210 122×48 | 601,210 122×48 | 12px | 15px/600 |
| Make Ana a caregiver | button | Button | 33,1038 152×48 | 258,1000 152×48 | 782,344 152×48 | 12px | 15px/600 |
| Remove Ana’s access to Leo | button | Button | 194,1038 154×48 | 418,1000 154×48 | 942,344 154×48 | 12px | 15px/600 |
| Make Cleo a guardian | button | Button | 33,1220 148×48 | 258,1182 148×48 | 782,526 148×48 | 12px | 15px/600 |
| Remove Cleo’s access to Leo | button | Button | 189,1220 154×48 | 414,1182 154×48 | 938,526 154×48 | 12px | 15px/600 |
| Make Sam a guardian | button | Button | 33,1338 148×48 | 258,1300 148×48 | 782,644 148×48 | 12px | 15px/600 |
| Remove Sam’s access to Leo | button | Button | 189,1338 154×48 | 414,1300 154×48 | 938,644 154×48 | 12px | 15px/600 |
| Add a person | button | Button | 33,1406 324×48 | 258,1368 686×48 | 782,712 414×48 | 12px | 15px/600 |

## States captured

`dup` = pixel-identical to an earlier state of the same variant (the control changed nothing visible); `axe` = violations found (0 = clean).

| # | State | Kind | Baseline (phone-light) | axe (pl/pd/dl/dd/wl/wd) | Notes |
|---|---|---|---|---|---|
| 01 | default-frame | frame | 01-default-frame.png | 0/0/0/0/0/0 |  |
| 02 | default-full-height | full | 02-default-full-height.png | 0/0/0/0/0/0 |  |
|  | NEXT SAVE Works | state | dup of 02 | ·/·/·/·/·/· | NEXT SAVE |
| 03 | NEXT SAVE Works after action | state-after-action | 03-next-save-works-after-action.png | 0/0/0/0/0/0 | NEXT SAVE |
|  | NEXT SAVE Offline | state | dup of 02 | ·/·/·/·/·/· | NEXT SAVE |
|  | NEXT SAVE Offline after action | state-after-action | dup of 03 | ·/·/·/·/·/· | NEXT SAVE |
|  | NEXT SAVE Error | state | dup of 02 | ·/·/·/·/·/· | NEXT SAVE |
|  | NEXT SAVE Error after action | state-after-action | dup of 03 | ·/·/·/·/·/· | NEXT SAVE |
|  | ROLE Guardian | state | dup of 02 | ·/·/·/·/·/· | ROLE |
| 04 | ROLE Caregiver | state | 04-role-caregiver.png | 0/0/0/0/0/0 | ROLE |
|  | CAREGIVER VIEW No page | state | dup of 02 | ·/·/·/·/·/· | CAREGIVER VIEW |
|  | CAREGIVER VIEW Read-only | state | dup of 02 | ·/·/·/·/·/· | CAREGIVER VIEW |
| 05 | sheet Switch child | sheet | 05-sheet-switch-child.png | 0/0/0/0/0/0 |  |
| 06 | sheet More | sheet | 06-sheet-more.png | 0/0/·/·/·/· |  |

## Acceptance

- Every state above matches its baseline in all six variants (tolerance in `VERIFICATION.md`).
- axe clean per state; targets ≥ 44 px; no horizontal scroll at 390 px and 200 % zoom; at ≥ 1280 px the content block is 920 px wide and the left column precedes the right in the DOM (`DESIGN.md` §2.1).
- Accessible names equal the `Name` column (or are listed in `IMPLEMENTATION.md` section 6).
- The no-JS path in the notes above works with JavaScript disabled.
