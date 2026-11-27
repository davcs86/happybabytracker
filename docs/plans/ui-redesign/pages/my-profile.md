# My profile

Prototype page: `My Profile.dc.html`  ·  baselines: `screenshots/my-profile/<variant>/NN-<state>.png`  ·  layout data: `specs/my-profile.<variant>.layout.json`

## Routes and files (legacy repo)

| Route | Status |
|---|---|
| `/account` | exists |

Legacy files to restyle: `ui/src/routes/(app)/account/+page.svelte`

## Implementation notes carried over from the prototype

These are the design notes that sit in the prototype's own side panel (they are part of the design, the panel itself is not).

```text
shadcn-svelte parts
Card, Badge, Button, Sheet, Input, Label, Checkbox, Sonner.

No-JS baseline
Each sheet is its own route (/profile/name, /profile/password, /profile/sessions). “Sign out” beside a device is a form POST with the session id; so is “Sign out all other devices”, after a confirm page.

Facts applied
No email in the product: the username is permanent and shown read-only. Admin is a separate principal, so there is no admin link here.

Assumptions
Caregivers get this page too. The current device has no sign-out button. Changing the password can sign out other devices (ticked by default).

Sample data. No real sign-in.
```

## Layout (measured from the rendered prototype, Geist loaded)

Phone frame 390 px wide, content height 1233 px; side-nav frame 980 px wide, content height 1143 px, two-column frame 1280 px wide, content height 822 px. All numbers are CSS px at 1×.

**Headings (phone)**

| Level | Text | Box x,y w×h | font |
|---|---|---|---|
| h1 | Your profile | 16,90 358×33 | 30px/600 |

**Interactive elements (default state)**

| Name | Element | shadcn part | Phone x,y w×h | 768–1279 x,y w×h | ≥1280 x,y w×h | Radius | Font |
|---|---|---|---|---|---|---|---|
| Notification settings | link | Button | 248,16 48×48 | 835,16 48×48 | 1086,16 48×48 | 12px | 16px/400 |
| Edit name | button | Button | 244,194 113×48 | 831,194 113×48 | 610,194 113×48 | 12px | 15px/600 |
| Change password | button | Button | 186,404 170×48 | 774,404 170×48 | 553,404 170×48 | 12px | 15px/600 |
| Sign out MacBook, Chrome, Oakland, CA | button | Button | 255,692 102×48 | 842,672 102×48 | 1094,352 102×48 | 12px | 15px/600 |
| Sign out iPad, Claude app, Hayward, CA | button | Button | 255,758 102×48 | 842,738 102×48 | 1094,417 102×48 | 12px | 15px/600 |
| Sign out all other devices | button | Button | 33,827 324×48 | 258,806 686×48 | 782,485 414×48 | 12px | 15px/600 |

## States captured

`dup` = pixel-identical to an earlier state of the same variant (the control changed nothing visible); `axe` = violations found (0 = clean).

| # | State | Kind | Baseline (phone-light) | axe (pl/pd/dl/dd/wl/wd) | Notes |
|---|---|---|---|---|---|
| 01 | default-frame | frame | 01-default-frame.png | 0/0/0/0/0/0 |  |
| 02 | default-full-height | full | 02-default-full-height.png | 0/0/0/0/0/0 |  |
|  | NEXT SAVE Works | state | dup of 02 | ·/·/·/·/·/· | NEXT SAVE |
|  | NEXT SAVE Offline | state | dup of 02 | ·/·/·/·/·/· | NEXT SAVE |
|  | NEXT SAVE Error | state | dup of 02 | ·/·/·/·/·/· | NEXT SAVE |
|  | ROLE Guardian | state | dup of 02 | ·/·/·/·/·/· | ROLE |
| 03 | ROLE Caregiver | state | 03-role-caregiver.png | 0/0/0/0/0/0 | ROLE |
| 04 | sheet More | sheet | 04-sheet-more.png | 0/0/·/·/·/· |  |

## Acceptance

- Every state above matches its baseline in all six variants (tolerance in `VERIFICATION.md`).
- axe clean per state; targets ≥ 44 px; no horizontal scroll at 390 px and 200 % zoom; at ≥ 1280 px the content block is 920 px wide and the left column precedes the right in the DOM (`DESIGN.md` §2.1).
- Accessible names equal the `Name` column (or are listed in `IMPLEMENTATION.md` section 6).
- The no-JS path in the notes above works with JavaScript disabled.
