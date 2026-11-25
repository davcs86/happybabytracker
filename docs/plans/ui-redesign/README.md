# Happy Baby Tracker: design export

Everything needed to restyle the SvelteKit app (`davcs86/happybabytracker`, `ui/`) to the new design, pixel for pixel, and to reuse the same design decisions in other projects.

Date: 2026-11-24. Source of truth: the Claude Design prototype (`prototype/`, also published as the "Happy Baby Tracker" artifact). Legacy = the app as implemented in PR 13. New = the prototype.

## Start here

1. `IMPLEMENTATION.md`: the ordered build plan, assumptions A1-A8, the shadcn install command, theme wiring, the accessible-name rename table, tests, definition of done. Read it fully before touching code.
2. `design-system/DESIGN.md`: the core rules (layout, navigation, forms, feedback, copy, accessibility).
3. `design-system/tokens/tokens.css`: drop-in Tailwind 4 theme (shadcn variable names, light and dark, tracker hues). Fonts are in `fonts/`.
4. `route-map.md`, then one `pages/<page>.md` per screen: routes, legacy files to change, shadcn parts, no-JS baseline, measured layout of every interactive element, every captured state.
5. `screenshots/INDEX.md`: the state matrix (1,077 unique PNGs for 1,992 states: light and dark in three layouts: phone 390 px, side-nav 980 px, wide two-column 1280 px). Per-state axe results are in the page specs.

## Contents

| Path | What |
|---|---|
| `IMPLEMENTATION.md` | Build plan and rules for the implementer |
| `design-system/` | Reusable system. Core layer: `DESIGN.md`, `components/COMPONENTS.md`, `patterns/PATTERNS.md`, `VERIFICATION.md`. Brand preset: `preset-happybaby/BRAND.md`. Generated: `tokens/tokens.css`, `tokens/tokens.json`. `icons/`: the 32 Lucide SVGs in use |
| `fonts/` | Geist and Geist Mono (variable, latin and latin-ext) with their OFL licences |
| `route-map.md`, `pages/` | Per-screen specs (23 pages) |
| `specs/` | Measured layout per page and variant (`<page>.<variant>.layout.json`): box, radius, font, accessible name, shadcn part |
| `screenshots/` | Baseline images `<page>/<variant>/NN-<state>.png` and `INDEX.md` (shipped as a second zip) |
| `tests/visual/` | Generated Playwright visual harness (`pages.visual.spec.ts`, `index.json`, `states.ts`). `states.ts` lists the states that still need an interaction hook in the real app |
| `prototype/` | The prototype source. `./make-runtime.sh` builds `runtime/` that opens in Chrome |
| `tools/` | `capture.py` (re-capture the matrix), `gen_pages.py`, `gen_visual.py`, `gen_tokens.py`, `gen_brand.py`, `install-baselines.sh` |

## Reusing the system in another project

Copy `design-system/`, keep the core files unchanged, replace `preset-happybaby/BRAND.md` values and regenerate tokens with `tools/gen_tokens.py` (it recomputes every contrast pair and reports failures below 4.5:1). The same system is published as a Design System artifact ("Happy Baby Tracker design system") for browsing and for other design sessions.

## What was verified, and what was not

Verified (headless Chromium with the real Geist fonts, animations off):
- axe (wcag2a, wcag2aa, wcag21aa, best-practice): 0 violations in every captured state, 6 variants (3 layouts × light and dark), 23 pages.
- Interactive targets at least 44 px (native checkbox and radio inputs are 20 px inside label rows of 44 px or more) and no horizontal overflow, across all 66 layout files (`tools/audit_layout.py`); text contrast at least 4.5:1 in both themes (the token generator checks each pair).
- Re-running `tools/capture.py` on two pages reproduced 31 of 32 images byte for byte; the 32nd differs in a 3 × 100 px area. Set `maxDiffPixelRatio` accordingly (see `VERIFICATION.md`).
- Numbers in `DESIGN.md` and `COMPONENTS.md` were checked against the prototype source and the measured layout; discrepancies found were fixed in the documents (for example card radius is tiered 12 / 14 / 16 by role).

Not verified: real devices, Safari and Firefox, screen readers, the no-JavaScript render (specified per page, not demonstrable in the prototype), the real POST endpoints.

## Changes made to the prototype while writing this export

Fixed in the prototype and recorded in `prototype/plan-7-design-notes.md` section 20: dark-theme toast action contrast (was 1.78:1), input text 15 → 16 px everywhere, invalid-field border unified (1 px destructive), content column unified at 720 px (Milestones was 672) and Home padding and gap aligned.

## Open decisions for the owner

- Resolved 2026-11-24: from 1280 px the content is a fixed 920 px two-column block (main + context); 768 to 1279 keeps one column. The owner asked for 1024 and confirmed 1280 after asking whether it fits better (at 1024 the 225 px side nav leaves 799 px, less than the 920 px block). Rules and the per-page column map: `design-system/DESIGN.md` §2.1.
- Pages with no prototype screen: only the Modules page (a sheet in the prototype; its no-JS route is not designed, the owner did not select it). Add a child (`/kids/new`), Child settings (`/kids/[kid]/settings`), Access (`/kids/[kid]/access`) and Choose child (`/kids`, the switcher's no-JS page) are designed.
- Whether the legacy docs and screenshots in the repo are regenerated as part of the change.
