# Brand preset: Happy Baby Tracker

The preset layer of the design system. Everything here is a *choice for this product*; the rules that use it are in `../DESIGN.md`. To make another product, copy this folder, change the values, regenerate `tokens/`, and keep `DESIGN.md` as is.

## Character
Warm, quiet, nocturnal-friendly. A cream-white surface with a rust-orange accent in light; a warm near-black with an amber accent in dark. Typeface: Geist (UI) and Geist Mono (usernames, ntfy topics, codes). One accent colour, used for the primary action, selection and focus only. Trackers get one hue each, and that is the only multi-colour in the product.

## Palette (values copied from the prototype; hex is the sRGB rendering of the oklch value)

| Token | Role | Light oklch | Light hex | Dark oklch | Dark hex |
|---|---|---|---|---|---|
| `--background` | Page and input surface | `oklch(0.982 0.008 70)` | `#fdf8f3` | `oklch(0.19 0.015 50)` | `#19120e` |
| `--card / --popover / --secondary` | Cards, sheets, tab bar, side nav | `oklch(0.998 0.003 70)` | `#fffefc` | `oklch(0.245 0.015 50)` | `#271e1a` |
| `--foreground` | Text, 1.5 px alert border, toast surface | `oklch(0.24 0.02 50)` | `#271d17` | `oklch(0.95 0.01 70)` | `#f3ede7` |
| `--muted-foreground` | Secondary text, badges, meta lines | `oklch(0.48 0.02 50)` | `#675b54` | `oklch(0.74 0.02 60)` | `#b5a89f` |
| `--border / --input` | 1 px hairlines, outlines | `oklch(0.9 0.012 60)` | `#e4dcd6` | `oklch(0.36 0.015 50)` | `#443b36` |
| `--primary / --ring` | Primary button fill, selected state, focus ring, links | `oklch(0.53 0.17 38)` | `#b83b07` | `oklch(0.78 0.13 55)` | `#f7a062` |
| `--primary-foreground` | Text on primary | `oklch(0.99 0 0)` | `#fcfcfc` | `oklch(0.2 0.02 50)` | `#1e130e` |
| `--muted` | Tab track (segmented control background) | `oklch(0.95 0.01 60)` | `#f4ede8` | `oklch(0.27 0.015 50)` | `#2d2420` |
| `--accent` | Pressed chip / selected row fill | `oklch(0.95 0.04 55)` | `#ffe8d6` | `oklch(0.3 0.05 55)` | `#412714` |
| `--primary-ink / --accent-foreground` | Text on --accent, tracker-neutral accents | `oklch(0.42 0.12 38)` | `#812f13` | `oklch(0.84 0.11 55)` | `#ffb784` |
| `--destructive` | Destructive text and 1.5 px border, row errors | `oklch(0.46 0.17 27)` | `#a21a1b` | `oklch(0.76 0.14 25)` | `#fe8b83` |
| `--toast-link` | Action text on the toast (inverse) surface | `oklch(0.8 0.12 55)` | `#f9a870` | `oklch(0.45 0.15 38)` | `#962a00` |

Overlay (sheet scrim): `rgba(20, 12, 8, 0.5)` in both themes. Dark theme uses *lighter* surfaces for raised things (card 0.245 over background 0.19), never shadows alone.

## Contrast (WCAG 2.x ratio, computed from the values above)

| Foreground on background | Light | Dark | Need |
|---|---|---|---|
| foreground on muted (tab track) | 14.22 | 13.06 | 4.5 |
| muted text on muted (tab track) | 5.66 | 6.55 | 4.5 |
| foreground on background | 15.61 | 15.94 | 4.5 |
| foreground on card | 16.35 | 14.05 | 4.5 |
| muted text on background | 6.22 | 7.99 | 4.5 |
| muted text on card | 6.51 | 7.05 | 4.5 |
| primary-foreground on primary | 5.58 | 8.8 | 4.5 |
| primary-ink on background | 8.47 | 10.9 | 4.5 |
| primary-ink on accent | 7.56 | 8.1 | 4.5 |
| primary on background (links, icons) | 5.43 | 8.96 | 4.5 |
| primary on card | 5.68 | 7.89 | 4.5 |
| destructive on background | 7.37 | 8.14 | 4.5 |
| destructive on card | 7.72 | 7.18 | 4.5 |
| toast-link on the toast surface (--foreground) | 8.52 | 6.88 | 4.5 |

Tracker chip text on chip fill: minimum 7.06 across all seven hues and both themes (need 4.5). The toast pair is the only one where the *same token name* sits on opposite surfaces in the two themes, so it is the one most likely to regress: re-check it whenever the palette changes (it measured 1.78 before it was fixed in this project).

## Typography

Geist variable (weights 400, 500, 600, 700 in use; 600 is the working weight for controls, 700 for badges/emphasis), Geist Mono for machine-readable strings. Install `@fontsource-variable/geist` and `@fontsource-variable/geist-mono` and import them once in the global CSS; `fonts/` in the export carries the Latin and Latin-extended woff2 files with the OFL licence for offline builds.

| Role | Size / weight / line-height | Notes |
|---|---|---|
| Page title `h1` | 30 / 600 / 1.1, letter-spacing -0.02em | wraps anywhere |
| Sheet and section title | 18 / 600 | |
| Lead | 17 | rare |
| Body | 15 / 400 / 1.5 | default |
| Control label (button, summary) | 15 / 600 | |
| Input text | 16 / 400 | 16 px stops iOS focus zoom |
| Secondary, row detail | 13.5-14 / 400 / 1.5 | muted |
| Meta, badge, tab label | 12-12.5 | floor 12 px; badge 600 |
| Chart ticks | 11 | the only text below 12 |

Full list of sizes in use: 11, 11.5, 12, 12.5, 13, 13.5, 14, 14.5, 15, 16, 17, 18, 22, 30 px. Do not add sizes: pick the nearest role.

## Radius usage

| Value | Used for |
|---|---|
| 12 px | button, alert, tracker chip, radio row, tab track |
| 8 px | input, select, textarea, ToggleGroup chip |
| 10 px | unit segment, side-nav item, compact control |
| 9 px | measure tab (inside a 12 px track) |
| 4 px | checkbox |
| 14 px | card, collapsible |
| 16 px | section card, empty-state card |
| 14 px | large button (sheet Done/Cancel) |
| 20 px | sheet top corners |
| 99 px | badge, pill |
| 2 px | drag handle |

## Tracker hues

One hue per tracker; lightness and chroma are fixed per theme so contrast is constant. Chip fill `oklch(L C h)` with (L,C) = (0.95, 0.04) light / (0.30, 0.05) dark; chip text (0.42, 0.12) light / (0.84, 0.11) dark; bar and chart mark (0.60, 0.14) light / (0.70, 0.12) dark.

| Tracker | hue | icon (lucide) | Light chip fill / text / bar | Dark chip fill / text / bar |
|---|---|---|---|---|
| bottle | 65 | `milk` | `oklch(0.95 0.04 65)` / `oklch(0.42 0.12 65)` / `oklch(0.6 0.14 65)` | `oklch(0.3 0.05 65)` / `oklch(0.84 0.11 65)` / `oklch(0.7 0.12 65)` |
| nursing | 20 | `heart` | `oklch(0.95 0.04 20)` / `oklch(0.42 0.12 20)` / `oklch(0.6 0.14 20)` | `oklch(0.3 0.05 20)` / `oklch(0.84 0.11 20)` / `oklch(0.7 0.12 20)` |
| diaper | 95 | `droplet` | `oklch(0.95 0.04 95)` / `oklch(0.42 0.12 95)` / `oklch(0.6 0.14 95)` | `oklch(0.3 0.05 95)` / `oklch(0.84 0.11 95)` / `oklch(0.7 0.12 95)` |
| sleep | 275 | `moon` | `oklch(0.95 0.04 275)` / `oklch(0.42 0.12 275)` / `oklch(0.6 0.14 275)` | `oklch(0.3 0.05 275)` / `oklch(0.84 0.11 275)` / `oklch(0.7 0.12 275)` |
| pumping | 330 | `arrow-up` | `oklch(0.95 0.04 330)` / `oklch(0.42 0.12 330)` / `oklch(0.6 0.14 330)` | `oklch(0.3 0.05 330)` / `oklch(0.84 0.11 330)` / `oklch(0.7 0.12 330)` |
| solids | 140 | `utensils` | `oklch(0.95 0.04 140)` / `oklch(0.42 0.12 140)` / `oklch(0.6 0.14 140)` | `oklch(0.3 0.05 140)` / `oklch(0.84 0.11 140)` / `oklch(0.7 0.12 140)` |
| symptom | 60 | `notebook-pen` | `oklch(0.95 0.04 60)` / `oklch(0.42 0.12 60)` / `oklch(0.6 0.14 60)` | `oklch(0.3 0.05 60)` / `oklch(0.84 0.11 60)` / `oklch(0.7 0.12 60)` |

## Shadows

- Sheet: `0 -8px 32px rgba(0,0,0,.2)`
- Toast and menus: `0 8px 24px rgba(0,0,0,.25)`
- Cards, buttons, chips: none (a 1 px border carries the edge)

## Icons

Lucide, 24 px viewBox, stroke 2, round caps and joins, `currentColor`. Inventory and files in `../icons/`. Sizes: 22 tab/side nav, 20 rows, 16 inline and chevrons, 18-24 inside tracker chips.

## What to change for another brand

1. Palette values in `tokens/gen` (or edit `tokens.css` and `tokens.json` together), then re-run the contrast table: every row above must still pass, including the toast row.
2. Typeface: keep the size/weight roles, swap the family; re-run the capture matrix because wrapping changes.
3. Tracker hues and icons for the domain; keep the (L, C) formulas.
4. Do **not** change: sizes, targets, radius usage, the 1.5 px alert border, the pressed-chip anatomy (fill + border + check). Those are structure, not brand.
