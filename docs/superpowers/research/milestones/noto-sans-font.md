# Embedding a Unicode font in the PDF export: research for plan 7 (2026-11-23)

The owner asked (2026-11-23) that names in other scripts render, by embedding a Unicode font rather
than substituting `?`. These are researcher notes. Every source was retrieved on **2026-11-23**.
Measurements were made in a scratch directory outside the repository with Go 1.27.4,
`codeberg.org/go-pdf/fpdf` v0.12.0 and `CGO_ENABLED=0 go build -trimpath -ldflags="-s -w"`. Nothing
was added to the repository except this note. Not legal advice.

## 1. The font

| Item | Value |
|---|---|
| Family | Noto Sans, version string "Version 2.015" (name ID 5), `OS/2.fsType` = 0 (installable embedding, no restriction) |
| Files | `NotoSans-Regular.ttf`, 431,364 bytes, SHA-256 `f3961a9cde016d41a4879aecda1474d3a36d6bf54fa0e4643de029cc2248b0e8`; `NotoSans-Bold.ttf`, 432,376 bytes, SHA-256 `87cb2d84472a7d66da659ee47b6cdb9552326e8c128245231f191b6ac72529d9` |
| Variant | **unhinted** static TTF. Hinting only matters for screen rasterisation at small sizes, not for a PDF, and the hinted files are 621,572 bytes each. Static files are used because fpdf's TrueType parser expects static fonts. The variable `NotoSans[wdth,wght].ttf` of google/fonts was not tried |
| Download URLs | https://raw.githubusercontent.com/notofonts/notofonts.github.io/main/fonts/NotoSans/unhinted/ttf/NotoSans-Regular.ttf and `…/NotoSans-Bold.ttf`, the Noto project's distribution site repository. This is a `main` branch, not a tag, so the implementer re-downloads and **must get these hashes**, or stops |
| Font's own copyright string | "Copyright 2022 The Noto Project Authors (https://github.com/notofonts/latin-greek-cyrillic)" |
| Licence file | `OFL.txt` from https://raw.githubusercontent.com/notofonts/latin-greek-cyrillic/main/OFL.txt (the source project the font names), SHA-256 `cee9892f9f0cc8fe882c9e9537ee6a89621d86ee7ceaf70b02e2b2b1c25c061a`; it begins "Copyright 2022 The Noto Project Authors (https://github.com/notofonts/latin-greek-cyrillic)" and "This Font Software is licensed under the SIL Open Font License, Version 1.1." **No Reserved Font Name** is declared |
| Repository path (plan) | `api/internal/export/fonts/{NotoSans-Regular.ttf,NotoSans-Bold.ttf,LICENSE,NOTICE}`. `LICENSE` is the OFL.txt verbatim. `NOTICE` names the files, version, hashes and source URLs, says the fonts are not MIT, and says embedded subsets appear in every exported PDF. `api/.dockerignore` drops only `Dockerfile`, `.dockerignore`, `**/*_test.go`, `**/testdata`, `*.db`, `*.db-*` and `.git`, so the directory reaches the build stage. The plan's Dockerfile step also copies `LICENSE` and `NOTICE` to `/usr/share/doc/happybabytracker-api/noto-sans/` |

## 2. Licence: the exact text that permits it (SIL OFL 1.1, from OFL.txt)

- Preamble: "The fonts, including any derivative works, can be bundled, embedded, redistributed and/or
  sold with any software provided that any reserved names are not used by derivative works. The
  fonts and derivatives, however, cannot be released under any other type of license. The
  requirement for fonts to remain under this license does not apply to any document created using
  the fonts or their derivatives."
- Condition 1: "Neither the Font Software nor any of its individual components, in Original or
  Modified Versions, may be sold by itself."
- Condition 2: "Original or Modified Versions of the Font Software may be bundled, redistributed
  and/or sold with any software, provided that each copy contains the above copyright notice and
  this license. These can be included either as stand-alone text files, human-readable headers or in
  the appropriate machine-readable metadata fields within text or binary files as long as those
  fields can be easily viewed by the user."
- Condition 5: "The Font Software, modified or unmodified, in part or in whole, must be distributed
  entirely under this license, and must not be distributed under any other license. The requirement
  for fonts to remain under this license does not apply to any document created using the Font
  Software."

The OFL FAQ (https://openfontlicense.org/ofl-faq/), quoted in fragments by the fetch tool:
- 1.12: embedding in a document is allowed "either in full or a subset"
- 1.13: "Referencing or embedding an OFL font in any document does not change the license of the document itself."
- 1.3: "The intent of the license is to allow aggregation or bundling with software under restricted licensing as well."
- 1.20: "At a minimum you must include the copyright statement, the license notice and the license text."

Reading: the repository's code stays MIT. The two font files are distributed under the OFL beside
it, in the repository and in the public API image, with `LICENSE` (the OFL text, which carries the
copyright line) next to them. A PDF made with the font is a "document created using the Font
Software", so the OFL does not reach the export.

## 3. Glyph coverage (the Regular file's `cmap`, parsed locally)

2,965 code points, the highest U+1DF1E.

| Block | Covered |
|---|---|
| Basic Latin | 95 / 95 |
| Latin-1 Supplement | 96 / 96 |
| Latin Extended-A | 128 / 128 (Ł, ő, ș …) |
| Latin Extended-B | 208 / 208 |
| Latin Extended Additional (Vietnamese) | 256 / 256 |
| Greek and Coptic | 121 / 144 (the block has unassigned code points) |
| Cyrillic | 256 / 256 |
| CJK Unified Ideographs | 0 / 20,992 |
| Arabic, Hebrew, Devanagari | 0 |
| Emoji (e.g. U+1F600) | not covered |

Also covered: €, curly quotes, em dash, ½, µ, °, `?` and U+FFFD. The Bold file's `cmap` was parsed too and is identical (2,965 code points).

## 4. Behaviour measured with fpdf v0.12.0 (`AddUTF8FontFromBytes`)

- **Subsetting is automatic.** A one-page document with both weights and a few dozen glyphs is
  11,753 bytes, against 863,740 bytes of font files. `pdffonts` reports `CID TrueType`,
  `Identity-H`, `emb yes`, `uni yes` (a ToUnicode map, so text can be copied and searched) and
  `sub no`. The "no" only means fpdf does not add the `ABCDEF+` subset tag to the font name. The
  size shows that it subsets.
- **A rune above U+FFFF panics** inside `Output` (`index out of range [65536]` in
  `generateCIDFontMap`), so one emoji in a name or note would crash the export. **The app must
  replace such runes before they reach fpdf.**
- **A rune in the BMP that the font lacks** (for example 李) does not panic. The font's `.notdef`
  glyph is drawn, while `pdftotext` still extracts 李 through ToUnicode. The plan replaces every rune
  the font lacks with `?` before rendering, so what prints and what a search finds agree, and adds
  the one-line notice.
- **Rendered and extracted correctly:** "Zoë Müller-Señor", "Łucja Ștefan", "Ольга Ελένη",
  "Œuvre € — “quotes”".
- **Text encoding in content streams:** each string is written as 2-byte code units, for example
  `(\0Z\0o\0\xeb …)Tj` for "Zoë". With compression off, `tr -d '\000' < file | grep -a -F 'Zo'`
  finds ASCII and Latin-1 text, which is what the smoke uses. Go tests decode UTF-16BE in the
  extractor.
- **Determinism:** two runs with a fixed creation date gave byte-identical files, with 0 rows and
  with 3000 rows (`cmp`). The subset is the same for the same text, so a golden file changes only
  when the text or the font files change. The font hashes are pinned by a test.

| Rows (one table, 3 columns) | Pages | Uncompressed | Compressed | Peak RSS (VmHWM), uncompressed / compressed |
|---|---|---|---|---|
| 0 (6 name lines) | 1 | 11,753 B | 11,590 B | 9.9 MB / 10.0 MB |
| 3000 | 57 | 802,128 B (core Helvetica: 683,833) | 125,351 B (core: 109,375) | 13.2 MB / 14.2 MB (core: 11.1 / 11.5) |
| 8000 | 152 | 2,119,724 B (core: 1,822,849) | 313,448 B | 22.5 MB / 18.5 MB (core: 17.4 uncompressed) |

The font costs each export about 2–5 MB more peak memory than core fonts. fpdf parses both font
files on every `New` document, roughly 0.9 MB of input.

**Binary and image:** API binary 13,365,408 bytes → 14,000,288 with fpdf → **14,889,120** with fpdf
and both fonts embedded (+1,523,712 bytes in total; the fonts alone add 888,832). Gzip: 5.51 →
6.24 MB.

## 5. Not verified

- That fpdf's TrueType parser handles every Noto Sans table correctly. GSUB and GPOS (ligatures,
  kerning, mark positioning) are **ignored** by fpdf, so combining marks are not positioned. Text
  that needs shaping (Arabic, Indic) is out of scope anyway, since the font lacks those scripts.
  Precomposed Latin, Greek and Cyrillic letters print correctly.
- That the `notofonts.github.io` main branch keeps serving these exact bytes. The hashes are
  recorded, and the implementer verifies them.
- The 60 MiB ceiling inside the real container. The numbers above come from standalone processes,
  and the smoke measures the real one.
- Extending to CJK later: Noto Sans CJK is a different family, much larger per weight. Its size and
  memory were not measured.
