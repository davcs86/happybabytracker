# Updating the milestone catalog

The catalog is one CSV file, `api/internal/milestones/data/cdc.csv`: CDC's "Learn the Signs. Act Early."
milestone checklists, transcribed from the twelve HTML age pages. It starts with a provenance header,
and its generated review document is `docs/milestones/cdc.md`. An update is a pull request that changes
the CSV, the review document and the tests that pin the data; the owner reviews the review document's
diff in that pull request. The catalog is embedded in the API image, so an update is live only after CI
publishes the image, the edge's `bump-happybabytracker` PR changes the pin and the edge is deployed.

## Where to look

| What | URL |
|---|---|
| Index (ages, "Updated" date) | https://www.cdc.gov/act-early/milestones/index.html |
| Age pages (the source of every item) | `https://www.cdc.gov/act-early/milestones/<page>.html` with `<page>` = `2-months`, `4-months`, `6-months`, `9-months`, `1-year`, `15-months`, `18-months`, `2-years`, `30-months`, `3-years`, `4-years`, `5-years` |
| Combined English PDF (wording comparison only) | https://www.cdc.gov/act-early/media/pdfs/2025/10/cdc-milestone-checklists-ltsae-english-508.pdf |
| Reuse conditions | https://www.cdc.gov/other/agencymaterials.html |

CDC publishes no revision schedule. Check each age page's "Updated" date; the shipped file was read
when every age page said "Updated: May 16, 2026". The research note
`docs/superpowers/research/milestones/cdc-milestone-checklists.md` records what was read, the licence
findings and the traps.

## Steps

1. Fetch the twelve age pages with `curl` into an empty scratch directory outside the repository, on the
   day you transcribe. Do not transcribe from a summarising fetch tool: those shorten quotations.
2. Extract, with a script kept outside the repository, the text of each `<li>` under the four headings
   "Social/Emotional Milestones", "Language/Communication Milestones", "Cognitive Milestones (learning,
   thinking, problem-solving)" and "Movement/Physical Development Milestones", in page order. Decode
   HTML entities and remove spaces at the start and end of an item. Change nothing else: keep the
   pronouns, the curly quotes and the punctuation. Compare the result with the CSV and list every
   added, removed and reworded item.
3. Check each new or reworded item against the denylist in `TestNoJudgementWords`
   (`api/internal/milestones/data_test.go`). On a match, stop and ask the owner; never reword CDC's text.
4. Edit the CSV.
   - **Provenance keys** (all required except `omitted`): `catalog` (`cdc`), `name`, `authority`,
     `edition` (with the "Updated" date the pages show), `retrieved` (YYYY-MM-DD, the day you fetched
     the pages), `origin_url` (the index page), `licence`, `attribution`, `screening_note` (the "Keep in
     mind" sentence of the age pages, verbatim) and `omitted` (one line of `;`-separated items: what the
     pages have that the file leaves out; no `;` inside an item). The attribution keeps "Source: CDC",
     "does not endorse" and "free of charge": `TestShippedCatalogLoads` pins them.
   - **Columns**: `id,age_months,domain,text,source_url,retired`. `age_months` is one of
     `constants.MilestoneCatalogAges`; `domain` is `social_emotional`, `language_communication`,
     `cognitive` or `movement_physical`; `source_url` is the age page the text came from; `text` is at
     most `constants.MilestoneCatalogTextMaxLen` characters, with no control character.
   - **A new item** gets a new id: `<age_months>m-<se|lc|cg|mp>-<slug>`, the slug being the item's first
     three words in lowercase a-z and 0-9 joined by hyphens, punctuation dropped (for example
     `2m-se-calms-down-when`). Only when that id is already taken in the same age and domain, add one
     more word at a time until it is unique; the slug is at most 48 characters. The domain code must match the
     `domain` column. Add the row in page order.
   - **A removed or reworded item is retired, never deleted or edited**: set its `retired` to the
     `retrieved` date of the update and add the new wording as a new row with a new id. Stored reached
     rows reference ids, so a deleted row would lose its text and an edited one would change what was
     recorded. Never renumber or reuse an id.
   - **The released ids are guarded.** `api/internal/milestones/testdata/ids.txt` lists every id ever
     released, one per line, and is append-only. `TestShippedIDsStay` fails when a listed id is no longer
     in the CSV (deleted or renamed); `TestShippedCatalogLoads` accepts a retired item only when its id is
     listed. Append a new item's id to `ids.txt` once the update that adds it is released.
5. Run `cd api && UPDATE_DOCS=1 go test ./internal/milestones/ -race`, then read the diff of
   `docs/milestones/cdc.md` and check that it shows exactly the changes listed in step 2. The loader
   names the file, line and column of every error, never the cell's text.
6. Compare the items with the PDF's text (`pdftotext`, spaces collapsed) and update `pdfDifferences` in
   `api/internal/milestones/review_test.go`, with the comparison date in its comment and in
   `renderMarkdown`. Regenerate the review document again.
7. Update the tests that pin real data: the per-age counts and the 159 total in
   `TestShippedCatalogLoads`, `TestPinnedItems`, and any test or smoke step that names a catalog id.
8. Run the whole pipeline in `CLAUDE.md` (API, UI, images and smoke) and open the pull request.

## Traps

- **Pick one source.** The HTML age pages and the PDFs differ in pronouns and, at 9 months, in one
  item's wording; the review document lists each difference. The catalog follows the HTML age pages.
- **The PDF's two-column layout** splits items across lines in `pdftotext -layout` output. Never
  transcribe from the PDF.
- **Inner quotes.** Some items quote speech in curly quotes (“Can I play with Alex?”); fields with a
  comma or a quote must be quoted in the CSV.
- **The screening sentence** is not the PDF's checklist sentence. The 6- and 9-month pages put the
  campaign name in straight double quotes; ten pages do not. The file keeps the ten pages' form.
- **Images are out.** CDC's photographs may be licensed by a third party; take only the milestone text.
- **`curl` and the CDN.** On 2026-11-24 cdc.gov answered 200 to `curl` with its default User-Agent and
  403 ("Access Denied") to `curl` with a browser User-Agent. Check the status and the body of each page.
- **Corrected age** is the app's rule (spec 7.1), not CDC's: CDC names no age at which correction stops.

## Pull request checklist

- [ ] The CSV's `edition` and `retrieved` are new, and its `omitted` list is current.
- [ ] Removed or reworded items are retired, not deleted; no id was renumbered or reused.
- [ ] `docs/milestones/cdc.md` was regenerated, its diff shows only the intended changes, and the owner
      has reviewed it.
- [ ] No item matches the denylist, or the owner has ruled on each match.
- [ ] The pinned data tests were updated, and the whole pipeline passed, with its output in the PR.
- [ ] After the merge: the edge's `bump-happybabytracker` PR and a deploy.
