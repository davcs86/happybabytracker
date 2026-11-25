import json
t=json.load(open('/home/claude/export/design-system/tokens/tokens.json'))
L=t['color']['light']; D=t['color']['dark']
names={'bg':('--background','Page and input surface'),'card':('--card / --popover / --secondary','Cards, sheets, tab bar, side nav'),'fg':('--foreground','Text, 1.5 px alert border, toast surface'),'muted':('--muted-foreground','Secondary text, badges, meta lines'),'border':('--border / --input','1 px hairlines, outlines'),'primary':('--primary / --ring','Primary button fill, selected state, focus ring, links'),'primaryFg':('--primary-foreground','Text on primary'),'soft2':('--muted','Tab track (segmented control background)'),'soft':('--accent','Pressed chip / selected row fill'),'ink':('--primary-ink / --accent-foreground','Text on --accent, tracker-neutral accents'),'destructive':('--destructive','Destructive text and 1.5 px border, row errors'),'toastLink':('--toast-link','Action text on the toast (inverse) surface')}
o=['# Brand preset: Happy Baby Tracker','',
'The preset layer of the design system. Everything here is a *choice for this product*; the rules that use it are in `../DESIGN.md`. To make another product, copy this folder, change the values, regenerate `tokens/`, and keep `DESIGN.md` as is.','',
'## Character','Warm, quiet, nocturnal-friendly. A cream-white surface with a rust-orange accent in light; a warm near-black with an amber accent in dark. Typeface: Geist (UI) and Geist Mono (usernames, ntfy topics, codes). One accent colour, used for the primary action, selection and focus only. Trackers get one hue each, and that is the only multi-colour in the product.','',
'## Palette (values copied from the prototype; hex is the sRGB rendering of the oklch value)','',
'| Token | Role | Light oklch | Light hex | Dark oklch | Dark hex |','|---|---|---|---|---|---|']
for k,(n,r) in names.items():
    o.append(f"| `{n}` | {r} | `{L[k]['oklch']}` | `{L[k]['hex']}` | `{D[k]['oklch']}` | `{D[k]['hex']}` |")
o+=['','Overlay (sheet scrim): `rgba(20, 12, 8, 0.5)` in both themes. Dark theme uses *lighter* surfaces for raised things (card 0.245 over background 0.19), never shadows alone.','',
'## Contrast (WCAG 2.x ratio, computed from the values above)','',
'| Foreground on background | Light | Dark | Need |','|---|---|---|---|']
lab={('fg','soft2'):'foreground on muted (tab track)',('muted','soft2'):'muted text on muted (tab track)',('fg','bg'):'foreground on background',('fg','card'):'foreground on card',('muted','bg'):'muted text on background',('muted','card'):'muted text on card',('primaryFg','primary'):'primary-foreground on primary',('ink','bg'):'primary-ink on background',('ink','soft'):'primary-ink on accent',('primary','bg'):'primary on background (links, icons)',('primary','card'):'primary on card',('destructive','bg'):'destructive on background',('destructive','card'):'destructive on card',('toastLink','fg'):'toast-link on the toast surface (--foreground)'}
for i,r in enumerate(t['contrast']['light']):
    d=t['contrast']['dark'][i]; key=(r['fg'],r['on'])
    need='4.5' 
    o.append(f"| {lab[key]} | {r['ratio']} | {d['ratio']} | {need} |")
mn=min(v[th]['chipContrast'] for v in t['tracker'].values() for th in('light','dark'))
o+=['',f'Tracker chip text on chip fill: minimum {mn} across all seven hues and both themes (need 4.5). The toast pair is the only one where the *same token name* sits on opposite surfaces in the two themes, so it is the one most likely to regress: re-check it whenever the palette changes (it measured 1.78 before it was fixed in this project).','',
'## Typography','',
'Geist variable (weights 400, 500, 600, 700 in use; 600 is the working weight for controls, 700 for badges/emphasis), Geist Mono for machine-readable strings. Install `@fontsource-variable/geist` and `@fontsource-variable/geist-mono` and import them once in the global CSS; `fonts/` in the export carries the Latin and Latin-extended woff2 files with the OFL licence for offline builds.','',
'| Role | Size / weight / line-height | Notes |','|---|---|---|',
'| Page title `h1` | 30 / 600 / 1.1, letter-spacing -0.02em | wraps anywhere |',
'| Sheet and section title | 18 / 600 | |',
'| Lead | 17 | rare |',
'| Body | 15 / 400 / 1.5 | default |',
'| Control label (button, summary) | 15 / 600 | |',
'| Input text | 16 / 400 | 16 px stops iOS focus zoom |',
'| Secondary, row detail | 13.5-14 / 400 / 1.5 | muted |',
'| Meta, badge, tab label | 12-12.5 | floor 12 px; badge 600 |',
'| Chart ticks | 11 | the only text below 12 |','',
f"Full list of sizes in use: {', '.join(str(v) for v in t['type']['scalePx'].values())} px. Do not add sizes: pick the nearest role.",'',
'## Radius usage','','| Value | Used for |','|---|---|']
for k,v in t['radius']['usage'].items(): o.append(f"| {v} px | {k} |")
o+=['','## Tracker hues','',
'One hue per tracker; lightness and chroma are fixed per theme so contrast is constant. Chip fill `oklch(L C h)` with (L,C) = (0.95, 0.04) light / (0.30, 0.05) dark; chip text (0.42, 0.12) light / (0.84, 0.11) dark; bar and chart mark (0.60, 0.14) light / (0.70, 0.12) dark.','',
'| Tracker | hue | icon (lucide) | Light chip fill / text / bar | Dark chip fill / text / bar |','|---|---|---|---|---|']
ic={'bottle':'milk','nursing':'heart','diaper':'droplet','sleep':'moon','pumping':'arrow-up','solids':'utensils','symptom':'notebook-pen'}
for k,v in t['tracker'].items():
    l=v['light']; d=v['dark']
    o.append(f"| {k} | {v['hue']} | `{ic[k]}` | `{l['bg']}` / `{l['fg']}` / `{l['bar']}` | `{d['bg']}` / `{d['fg']}` / `{d['bar']}` |")
o+=['','## Shadows','','- Sheet: `0 -8px 32px rgba(0,0,0,.2)`','- Toast and menus: `0 8px 24px rgba(0,0,0,.25)`','- Cards, buttons, chips: none (a 1 px border carries the edge)','','## Icons','','Lucide, 24 px viewBox, stroke 2, round caps and joins, `currentColor`. Inventory and files in `../icons/`. Sizes: 22 tab/side nav, 20 rows, 16 inline and chevrons, 18-24 inside tracker chips.','',
'## What to change for another brand','','1. Palette values in `tokens/gen` (or edit `tokens.css` and `tokens.json` together), then re-run the contrast table: every row above must still pass, including the toast row.','2. Typeface: keep the size/weight roles, swap the family; re-run the capture matrix because wrapping changes.','3. Tracker hues and icons for the domain; keep the (L, C) formulas.','4. Do **not** change: sizes, targets, radius usage, the 1.5 px alert border, the pressed-chip anatomy (fill + border + check). Those are structure, not brand.']
open('/home/claude/export/design-system/preset-happybaby/BRAND.md','w').write('\n'.join(o)+'\n')
print('ok',len(o))
