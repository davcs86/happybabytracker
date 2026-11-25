# Notifications

Prototype page: `Notifications.dc.html`  ·  baselines: `screenshots/notifications/<variant>/NN-<state>.png`  ·  layout data: `specs/notifications.<variant>.layout.json`

## Routes and files (legacy repo)

| Route | Status |
|---|---|
| `/account/notifications` | exists |
| `/account/notifications/subscribe (POST)` | exists |

Legacy files to restyle: `ui/src/routes/(app)/account/notifications/+page.svelte`, `ui/src/lib/components/NotificationDevice.svelte`

## Implementation notes carried over from the prototype

These are the design notes that sit in the prototype's own side panel (they are part of the design, the panel itself is not).

```text
shadcn-svelte parts
Card, Button, Input, Label, Checkbox (24 px), Alert, Separator.

No-JS
Everything works except the one enable button. The help line stays visible.
```

## Layout (measured from the rendered prototype, Geist loaded)

Phone frame 390 px wide, content height 800 px; side-nav frame 980 px wide, content height 780 px, two-column frame 1280 px wide, content height 780 px. All numbers are CSS px at 1×.

**Headings (phone)**

| Level | Text | Box x,y w×h | font |
|---|---|---|---|
| h1 | Notifications | 16,90 358×33 | 30px/600 |
| h2 | This device | 33,165 324×23 | 18px/600 |
| h2 | Your other devices | 33,350 324×23 | 18px/600 |
| h2 | ntfy | 33,496 324×23 | 18px/600 |

**Interactive elements (default state)**

| Name | Element | shadcn part | Phone x,y w×h | 768–1279 x,y w×h | ≥1280 x,y w×h | Radius | Font |
|---|---|---|---|---|---|---|---|
| Notification settings | link | Button | 248,16 48×48 | 835,16 48×48 | 1086,16 48×48 | 12px | 16px/400 |
| Turn on notifications on this device | button | Button | 33,200 324×48 | 258,200 414×48 | 310,202 414×48 | 12px | 15px/600 |
| Remove Work laptop | button | Button | 257,392 100×48 | 572,392 100×48 | 624,396 100×48 | 12px | 15px/600 |
| Turn on ntfy | button | Button | 33,606 324×48 | 258,586 414×48 | 782,256 414×48 | 12px | 15px/600 |

## States captured

`dup` = pixel-identical to an earlier state of the same variant (the control changed nothing visible); `axe` = violations found (0 = clean).

| # | State | Kind | Baseline (phone-light) | axe (pl/pd/dl/dd/wl/wd) | Notes |
|---|---|---|---|---|---|
| 01 | default-frame | frame | 01-default-frame.png | 0/0/0/0/0/0 |  |
|  | NEXT SAVE Works | state | dup of 01 | ·/·/·/·/·/· | NEXT SAVE |
|  | NEXT SAVE Offline | state | dup of 01 | ·/·/·/·/·/· | NEXT SAVE |
|  | NEXT SAVE Error | state | dup of 01 | ·/·/·/·/·/· | NEXT SAVE |
| 02 | CHANNELS ON THIS SERVER None | state | 02-channels-on-this-server-none.png | 0/0/0/0/0/0 | CHANNELS ON THIS SERVER |
| 03 | CHANNELS ON THIS SERVER Web Push | state | 03-channels-on-this-server-web-push.png | 0/0/0/0/0/0 | CHANNELS ON THIS SERVER |
|  | CHANNELS ON THIS SERVER + ntfy | state | dup of 01 | ·/·/·/·/·/· | CHANNELS ON THIS SERVER |
|  | THIS BROWSER Push OK | state | dup of 01 | ·/·/·/·/·/· | THIS BROWSER |
| 04 | THIS BROWSER iPhone Safari | state | 04-this-browser-iphone-safari.png | 0/0/0/0/0/0 | THIS BROWSER |
| 05 | THIS BROWSER No push | state | 05-this-browser-no-push.png | 0/0/0/0/0/0 | THIS BROWSER |
| 06 | THIS BROWSER JS off | state | 06-this-browser-js-off.png | 0/0/0/0/0/0 | THIS BROWSER |
|  | BROWSER PROMPT ANSWER Allow | state | dup of 01 | ·/·/·/·/·/· | BROWSER PROMPT ANSWER |
|  | BROWSER PROMPT ANSWER Block | state | dup of 01 | ·/·/·/·/·/· | BROWSER PROMPT ANSWER |
|  | BROWSER PROMPT ANSWER Dismiss | state | dup of 01 | ·/·/·/·/·/· | BROWSER PROMPT ANSWER |
|  | BROWSER PROMPT ANSWER Error | state | dup of 01 | ·/·/·/·/·/· | BROWSER PROMPT ANSWER |
|  | As built | state | dup of 01 | ·/·/·/·/·/· |  |
|  | One-line instruction | state | dup of 01 | ·/·/·/·/·/· |  |
|  | SIMULATE Ana enables here | state | dup of 01 | ·/·/·/·/·/· | SIMULATE |
| 07 | SIMULATE Key rotated | state | 07-simulate-key-rotated.png | 0/0/0/0/0/0 | SIMULATE |
|  | ROLE Guardian | state | dup of 01 | ·/·/·/·/·/· | ROLE |
| 08 | ROLE Caregiver | state | 08-role-caregiver.png | 0/0/0/0/0/0 | ROLE |
| 09 | sheet More | sheet | 09-sheet-more.png | 0/0/·/·/·/· |  |

## Acceptance

- Every state above matches its baseline in all six variants (tolerance in `VERIFICATION.md`).
- axe clean per state; targets ≥ 44 px; no horizontal scroll at 390 px and 200 % zoom; at ≥ 1280 px the content block is 920 px wide and the left column precedes the right in the DOM (`DESIGN.md` §2.1).
- Accessible names equal the `Name` column (or are listed in `IMPLEMENTATION.md` section 6).
- The no-JS path in the notes above works with JavaScript disabled.
