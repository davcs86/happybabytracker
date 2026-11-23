# PDF text encoding, page size, determinism and test-time verification: notes for plan 7 (2026-11-23)

Researcher notes. The facts come from the PDF format as the libraries implement it and from local
experiments, run on **2026-11-23** in a scratch directory outside the repository with Go 1.27.4 and
poppler's `pdftotext` and `pdfinfo` from the session's environment (they are not in CI's
runner or the images).

> **Superseded in part, 2026-11-23:** the owner chose to embed a Unicode font, not to use WinAnsi with
> `?`. Sections 1 and 4 below describe the core-font option that was not chosen. The chosen design and
> its measurements are in `noto-sans-font.md`: text is written as 2-byte code units, and the smoke
> strips NULs before `grep`.

## 1. Fonts and characters

- The PDF standard 14 fonts (Times, Helvetica, Courier in four styles each, Symbol, ZapfDingbats)
  need no embedding. A font dictionary that names `/BaseFont /Helvetica` and
  `/Encoding /WinAnsiEncoding` is enough, and every conforming viewer supplies the glyphs. Verified
  with the standard-library prototype: `pdfinfo` reads it and `pdftotext` extracts the text.
- WinAnsiEncoding is Windows-1252: Latin-1 (U+00A0–U+00FF) plus 27 characters at 0x80–0x9F (€, ‚, ƒ,
  „, …, †, ‡, ˆ, ‰, Š, ‹, Œ, Ž, ‘, ’, “, ”, •, –, —, ˜, ™, š, ›, œ, ž, Ÿ). That covers the accented
  letters of English, Spanish, French, German, Portuguese, Italian and the Nordic languages, but
  not Polish (Ł), Czech (ř), Romanian (ș), Turkish (ı, ş), Greek, Cyrillic, CJK, Arabic, Hebrew,
  Devanagari or emoji.
- Measured with fpdf's translator: every rune outside cp1252 becomes `.` (section 2 of
  `pdf-libraries.md`). An app-owned translator can substitute any byte instead. The plan proposes
  `?` and a one-line notice on the profile page.
- Full Unicode needs an embedded TrueType font (fpdf `AddUTF8Font`, gopdf's only path) in the image.
  That means a font licence (SIL OFL for Noto Sans or DejaVu) and a few hundred KB per font, and no
  single font covers every script. Not measured.

## 2. Page size

- A4 is 210 × 297 mm and US Letter is 215.9 × 279.4 mm. Their intersection is **210 × 279.4 mm**.
  A layout whose margins, content and footer all sit inside the top 279.4 mm of an A4 page prints
  at 100 % on Letter with nothing clipped (the extra 17.6 mm of A4 is blank). With "fit to page" it
  prints on either paper anyway. This needs no setting and no zone table.
- The alternative, Letter or A4 by country, needs a time-zone-to-country table. Go's `time` package
  has none, and the zone names alone are not reliable (`America/*` covers Letter and A4 countries
  alike).

## 3. Determinism

- fpdf: with `SetCatalogSort(true)`, a fixed `SetCreationDate` and `SetModificationDate`, and a fixed
  `SetProducer`, two runs were byte-identical (`cmp`). The file has no `/ID` array. Fonts are core
  fonts, so no subset tags are generated.
- The creation date is the generation instant (`Deps.Now`), so tests inject a fixed clock, as every
  handler test already does (`kit.now`).
- Map iteration in our own code must not reach the output, for example by ranging over a map of
  modules. Sections are written in a fixed order.

## 4. Reading the text back without a PDF viewer

- With `SetCompression(false)`, every cell is a literal string followed by `Tj`, for example
  `BT 144.57 769.74 Td (3000 g)Tj ET`. The strings are cp1252 bytes with `\`, `(` and `)` escaped. A
  test helper of about 40 lines can scan each content stream for `\((?:\\.|[^\\)])*\)\s*Tj`,
  unescape `\\`, `\(`, `\)` and octal `\ddd`, decode cp1252 and return the strings in page order. Go
  tests then assert exact strings and the absence of words.
- In `scripts/smoke.sh` (bash, curl and awk only, no Python and no PDF tools), the same file can be
  checked with `grep -a` for structural markers and for ASCII strings: `%PDF-1.3` at byte 0,
  `%%EOF` at the end, `/Type /Pages`, `/Count N`, and the literal `(…)Tj` strings of chosen values
  (the kid's ASCII name, a distinctive volume, the attribution sentence's first words). A string
  longer than its cell wraps, so smoke asserts on short cells only.
- With compression on (Flate), the text is invisible to `grep` and the smoke would need `zlib`,
  meaning Python or another container. That is why the plan proposes uncompressed output (Open
  value 19): 6.3 times the size, measured (684 KB against 109 KB for 3000 table rows).

## 5. Not verified

- How viewers render `?` substitutions inside names in a printed record (cosmetic).
- Whether iOS Safari's download of an `attachment` PDF from an installed PWA opens the share sheet
  or a viewer. No device test was run.
- Whether `pdftotext` is on GitHub's `ubuntu-latest` image. It is not relied on.
