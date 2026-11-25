# Verification: how to prove an implementation matches the design

Three layers, cheapest first. All three run in CI on every pull request that touches UI.

## 1. Static: tokens and contrast
- `tokens/tokens.css` is generated; do not hand-edit. The repo's global stylesheet imports it unchanged.
- Contrast: `tokens/tokens.json` holds the pair table; re-run `gen_tokens.py` (or an equivalent script) after any palette change; every row ≥ 4.5 (UI boundaries ≥ 3). The toast pair is the one that regresses.
- Lint: no raw colour literals (`#`, `rgb(`, `oklch(`) outside `tokens.css` and the print stylesheet; no `text-[10px]`-style arbitrary sizes below 12 px except chart ticks.

## 2. Accessibility and layout, per state
For each documented state, in the six variants (phone-light, phone-dark = 390 px; desktop-light, desktop-dark = 980 px with the side nav; wide-light, wide-dark = 1280 px two-column; viewport 1700 px so the 920 px block is centred):
1. axe-core (`wcag2a`, `wcag2aa`, `wcag21aa`, `best-practice`): 0 violations.
2. Every interactive element ≥ 44 × 44 px (audit script: bounding boxes of `a, button, input, select, summary, [role=button]`), text ≥ 12 px.
3. No horizontal scroll at 390 px and at 200 % zoom (`document.scrollingElement.scrollWidth <= innerWidth`).
4. Accessible-name snapshot (`page.accessibility.snapshot()` or `getByRole` list) diffed against the stored one in `specs/`.
5. Load the product font before measuring (a fallback font hides overflow).
Playwright driver: `tools/capture.py` shows the pattern (states are reached by clicking the documented triggers, the failure injectors are test hooks, not product code).

## 3. Pixel baselines
`screenshots/<page>/<variant>/NN-<state>.png` are the reference. Implementation tests take the same shot and compare with a tolerance:
```ts
// playwright.config.ts: one project per variant
{ name: 'phone-light', use: { viewport: { width: 390, height: 800 }, colorScheme: 'light', deviceScaleFactor: 1 } },
{ name: 'phone-dark',  use: { viewport: { width: 390, height: 800 }, colorScheme: 'dark',  deviceScaleFactor: 1 } },
{ name: 'desktop-light', use: { viewport: { width: 980, height: 780 }, colorScheme: 'light', deviceScaleFactor: 1 } },
{ name: 'desktop-dark',  use: { viewport: { width: 980, height: 780 }, colorScheme: 'dark',  deviceScaleFactor: 1 } },
{ name: 'wide-light',  use: { viewport: { width: 1280, height: 780 }, colorScheme: 'light', deviceScaleFactor: 1 } },
{ name: 'wide-dark',   use: { viewport: { width: 1280, height: 780 }, colorScheme: 'dark',  deviceScaleFactor: 1 } },
// expect(page).toHaveScreenshot('01-default.png', { maxDiffPixelRatio: 0.002, animations: 'disabled', caret: 'hide' })
```
Notes: the baselines were captured with Geist loaded, animations and transitions off, the prototype's phone frame (rounded corners, outer shadow, simulation panel) removed, and a full-height capture of the scrolling content, so compare the **content area** (clip to the app frame or capture `main` + navigation). Expect sub-pixel text differences between rendering stacks; update baselines per environment with `--update-snapshots` only after reviewing the diff, and never to hide a layout difference.

## 4. Not covered (state this in release notes)
Screen readers (VoiceOver, TalkBack, NVDA), real devices and browsers other than Chromium, the no-JS render path (verify it by hand or with JS disabled in Playwright), motion preferences, and print rendering of the PDF in a real printer.

## 5. Running the capture on a new project
```
python3 tools/capture.py <project_dir> <out_dir> <fonts_dir> <axe.min.js> "<Page name>" ...
# writes out/screenshots/<page>/<variant>/NN-<state>.png
#        out/manifest-<page>.json   (state records + axe results + duplicate links)
#        out/specs/<page>.<variant>.layout.json (element boxes, computed styles, names)
```
Requires Python Playwright, Chrome, `axe-core`, and the font files in `fonts/`.
