# Admin

Prototype page: `Admin.dc.html`  ·  baselines: `screenshots/admin/<variant>/NN-<state>.png`  ·  layout data: `specs/admin.<variant>.layout.json`

## Routes and files (legacy repo)

| Route | Status |
|---|---|
| `/admin/parents` | exists |

Legacy files to restyle: `ui/src/routes/admin/parents/+page.svelte`, `ui/src/routes/admin/+layout.svelte`

## Implementation notes carried over from the prototype

These are the design notes that sit in the prototype's own side panel (they are part of the design, the panel itself is not).

```text
shadcn-svelte parts
Card, Badge, Button, Alert, Sheet (create, link), AlertDialog (disable), Input, Label, Toggle group, Sonner.

No-JS baseline
The filter is a link with ?show=. “Create parent”, “New link” and “Enable” are form POSTs; the link page is the POST response and shows the link once. “Disable” opens its own confirm page.

Facts applied
No child names or counts anywhere. Reset links last 1 hour, invite links 72 hours, single use, shown once, no email. A parent who is the only active guardian of any child can’t be disabled; the API refuses.

Assumptions
Sentence case for the refusal text. Enabling a parent who never set a password returns them to Invited.

Sample data.
```

## Layout (measured from the rendered prototype, Geist loaded)

Phone frame 390 px wide, content height 1619 px; side-nav frame 980 px wide, content height 1543 px, two-column frame 1280 px wide, content height 1466 px. All numbers are CSS px at 1×.

**Headings (phone)**

| Level | Text | Box x,y w×h | font |
|---|---|---|---|
| h1 | Parents | 16,98 211×33 | 30px/600 |

**Interactive elements (default state)**

| Name | Element | shadcn part | Phone x,y w×h | 768–1279 x,y w×h | ≥1280 x,y w×h | Radius | Font |
|---|---|---|---|---|---|---|---|
| Notification settings | link | Button | 263,16 48×48 | 850,16 48×48 | 1102,16 48×48 | 12px | 16px/400 |
| Create parent | button | Button | 235,90 139×48 | 822,90 139×48 | 1074,90 139×48 | 12px | 15px/600 |
| New reset link for Dan | button | Button | 33,462 146×48 | 258,422 146×48 | 310,438 146×48 | 12px | 15px/600 |
| Disable Dan | button | Button | 186,462 96×48 | 412,422 96×48 | 463,438 96×48 | 12px | 15px/600 |
| New reset link for Ana | button | Button | 33,640 146×48 | 258,600 146×48 | 310,620 146×48 | 12px | 15px/600 |
| Disable Ana | button | Button | 186,640 96×48 | 412,600 96×48 | 463,620 96×48 | 12px | 15px/600 |
| New reset link for Rae | button | Button | 33,818 146×48 | 258,778 146×48 | 310,802 146×48 | 12px | 15px/600 |
| Disable Rae | button | Button | 186,818 96×48 | 412,778 96×48 | 463,802 96×48 | 12px | 15px/600 |
| New reset link for Cleo | button | Button | 33,996 146×48 | 258,956 146×48 | 310,984 146×48 | 12px | 15px/600 |
| Disable Cleo | button | Button | 186,996 96×48 | 412,956 96×48 | 463,984 96×48 | 12px | 15px/600 |
| Enable Sam | button | Button | 33,1174 91×48 | 258,1134 91×48 | 310,1166 91×48 | 12px | 15px/600 |
| New invite link for Nina | button | Button | 33,1394 148×48 | 258,1336 148×48 | 310,1372 148×48 | 12px | 15px/600 |
| Disable Nina | button | Button | 189,1394 96×48 | 414,1336 96×48 | 466,1372 96×48 | 12px | 15px/600 |
| Sign out of admin | button | Button | 16,1544 358×48 | 241,1467 720×48 | 764,332 448×48 | 12px | 15px/600 |

## States captured

`dup` = pixel-identical to an earlier state of the same variant (the control changed nothing visible); `axe` = violations found (0 = clean).

| # | State | Kind | Baseline (phone-light) | axe (pl/pd/dl/dd/wl/wd) | Notes |
|---|---|---|---|---|---|
| 01 | default-frame | frame | 01-default-frame.png | 0/0/0/0/0/0 |  |
| 02 | default-full-height | full | 02-default-full-height.png | 0/0/0/0/0/0 |  |
|  | NEXT SAVE Works | state | dup of 02 | ·/·/0/·/·/· | NEXT SAVE |
|  | NEXT SAVE Offline | state | dup of 02 | ·/·/·/·/·/· | NEXT SAVE |
|  | NEXT SAVE Error | state | dup of 02 | ·/·/·/·/·/· | NEXT SAVE |

## Acceptance

- Every state above matches its baseline in all six variants (tolerance in `VERIFICATION.md`).
- axe clean per state; targets ≥ 44 px; no horizontal scroll at 390 px and 200 % zoom; at ≥ 1280 px the content block is 920 px wide and the left column precedes the right in the DOM (`DESIGN.md` §2.1).
- Accessible names equal the `Name` column (or are listed in `IMPLEMENTATION.md` section 6).
- The no-JS path in the notes above works with JavaScript disabled.
