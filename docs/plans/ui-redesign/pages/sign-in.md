# Sign in

Prototype page: `Sign In.dc.html`  ·  baselines: `screenshots/sign-in/<variant>/NN-<state>.png`  ·  layout data: `specs/sign-in.<variant>.layout.json`

## Routes and files (legacy repo)

| Route | Status |
|---|---|
| `/login` | exists |

Legacy files to restyle: `ui/src/routes/login/+page.svelte`, `ui/src/routes/login/+page.server.ts`

## Implementation notes carried over from the prototype

These are the design notes that sit in the prototype's own side panel (they are part of the design, the panel itself is not).

```text
shadcn-svelte parts
Card, Input, Label, Alert, Button.

Failure copy
One generic message for a wrong username or password, so it never says which was wrong. The username stays; the password clears. Rate limit and server-down have their own sentence.

No-JS baseline
A form POST to /login. A refusal comes back as the same page with the message and the username kept.

Assumptions
No “show password” toggle and no “forgot password” link: a guardian or the administrator issues a reset link (1 hour).
```

## Layout (measured from the rendered prototype, Geist loaded)

Phone frame 390 px wide, content height 800 px; side-nav frame 980 px wide, content height 780 px, two-column frame 1280 px wide, content height 780 px. All numbers are CSS px at 1×.

**Headings (phone)**

| Level | Text | Box x,y w×h | font |
|---|---|---|---|
| h1 | Sign in | 20,209 350×33 | 30px/600 |

**Interactive elements (default state)**

| Name | Element | shadcn part | Phone x,y w×h | 768–1279 x,y w×h | ≥1280 x,y w×h | Radius | Font |
|---|---|---|---|---|---|---|---|
| Username | label |  | 37,312 316×16 | 307,302 366×16 | 457,302 366×16 |  | 12px/400 |
| username | input |  | 37,334 316×48 | 307,324 366×48 | 457,324 366×48 | 8px | 16px/400 |
| Password | label |  | 37,396 316×16 | 307,386 366×16 | 457,386 366×16 |  | 12px/400 |
| password | input |  | 37,418 316×48 | 307,408 366×48 | 457,408 366×48 | 8px | 16px/400 |
| Sign in | button | Button | 37,480 316×52 | 307,470 366×52 | 457,470 366×52 | 12px | 15px/600 |
| Open an invite link | link |  | 102,584 106×44 | 345,574 106×44 | 495,574 106×44 |  | 12.5px/400 |

## States captured

`dup` = pixel-identical to an earlier state of the same variant (the control changed nothing visible); `axe` = violations found (0 = clean).

| # | State | Kind | Baseline (phone-light) | axe (pl/pd/dl/dd/wl/wd) | Notes |
|---|---|---|---|---|---|
| 01 | default-frame | frame | 01-default-frame.png | 0/0/0/0/0/0 |  |
|  | NEXT SIGN-IN Wrong password | state | dup of 01 | ·/·/·/·/·/· | NEXT SIGN-IN |
| 02 | NEXT SIGN-IN Wrong password after action | state-after-action | 02-next-sign-in-wrong-password-after-action.png | 0/0/0/0/0/0 | NEXT SIGN-IN |
|  | NEXT SIGN-IN Rate limited | state | dup of 01 | ·/·/·/·/·/· | NEXT SIGN-IN |
| 03 | NEXT SIGN-IN Rate limited after action | state-after-action | 03-next-sign-in-rate-limited-after-action.png | 0/0/0/0/0/0 | NEXT SIGN-IN |
|  | NEXT SIGN-IN API down | state | dup of 01 | ·/·/·/·/·/· | NEXT SIGN-IN |
| 04 | NEXT SIGN-IN API down after action | state-after-action | 04-next-sign-in-api-down-after-action.png | 0/0/0/0/0/0 | NEXT SIGN-IN |
| 05 | blank submit | flow | 05-blank-submit.png | 0/0/0/0/0/0 |  |
| 06 | signed in | flow | 06-signed-in.png | 0/0/0/0/0/0 |  |

## Acceptance

- Every state above matches its baseline in all six variants (tolerance in `VERIFICATION.md`).
- axe clean per state; targets ≥ 44 px; no horizontal scroll at 390 px and 200 % zoom; at ≥ 1280 px the content block is 920 px wide and the left column precedes the right in the DOM (`DESIGN.md` §2.1).
- Accessible names equal the `Name` column (or are listed in `IMPLEMENTATION.md` section 6).
- The no-JS path in the notes above works with JavaScript disabled.
