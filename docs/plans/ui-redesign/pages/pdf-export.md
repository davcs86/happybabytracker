# PDF layout

Prototype page: `PDF Export.dc.html`  ·  baselines: `screenshots/pdf-export/<variant>/NN-<state>.png`  ·  layout data: `specs/pdf-export.<variant>.layout.json`

## Routes and files (legacy repo)

| Route | Status |
|---|---|
| `/kids/[kid]/export/pdf` | exists (server-generated; design is the print layout) |

Legacy files to restyle: `ui/src/routes/(app)/kids/[kid]/export/pdf/+server.ts`

## States captured

`dup` = pixel-identical to an earlier state of the same variant (the control changed nothing visible); `axe` = violations found (0 = clean).

| # | State | Kind | Baseline (phone-light) | axe (pl/pd/dl/dd/wl/wd) | Notes |
|---|---|---|---|---|---|
| 01 | sheet-1 | paper | 01-sheet-1.png | ·/·/·/·/·/· |  |
| 02 | sheet-2 | paper | 02-sheet-2.png | ·/·/·/·/·/· |  |
| 03 | sheet-3 | paper | 03-sheet-3.png | ·/·/·/·/·/· |  |
| 04 | sheet-4 | paper | 04-sheet-4.png | ·/·/·/·/·/· |  |
| 05 | sheet-5 | paper | 05-sheet-5.png | ·/·/·/·/·/· |  |
| 06 | sheet-6 | paper | 06-sheet-6.png | ·/·/·/·/·/· |  |
| 07 | sheet-7 | paper | 07-sheet-7.png | ·/·/·/·/·/· |  |

## Acceptance

- Every state above matches its baseline in all six variants (tolerance in `VERIFICATION.md`).
- axe clean per state; targets ≥ 44 px; no horizontal scroll at 390 px and 200 % zoom; at ≥ 1280 px the content block is 920 px wide and the left column precedes the right in the DOM (`DESIGN.md` §2.1).
- Accessible names equal the `Name` column (or are listed in `IMPLEMENTATION.md` section 6).
- The no-JS path in the notes above works with JavaScript disabled.
