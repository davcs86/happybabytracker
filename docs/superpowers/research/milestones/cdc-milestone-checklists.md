# CDC "Learn the Signs. Act Early." milestone checklists: research for plan 7 (2026-11-23)

Researcher notes for the built-in milestone catalog (design 10.1). Read-only research; nothing was
transcribed into the repository. Not legal advice. Every source below was retrieved on **2026-11-23**.

## 1. Headline findings

1. **What cdc.gov serves.** Twelve checklists, at 2, 4, 6, 9 months, 1 year, 15, 18 months, 2 years,
   30 months, 3, 4 and 5 years. The milestones index page gives these ages and says it was
   "Updated: February 16, 2026" and "Reviewed: February 17, 2026". The age pages read for this note
   (2 months and 4 years) say "Updated: May 16, 2026".
2. **The four domains are the page headings**, exactly as printed: "Social/Emotional Milestones",
   "Language/Communication Milestones", "Cognitive Milestones (learning, thinking, problem-solving)",
   "Movement/Physical Development Milestones".
3. **Edition.** These are the checklists of the 2022 revision. The revision is described in Zubler JM,
   Wiggins LD, Macias MM, et al., "Evidence-Informed Milestones for Developmental Surveillance Tools",
   *Pediatrics* 149(3), 1 March 2022, DOI 10.1542/peds.2021-052138, PMCID PMC9680195. That article is
   AAP's publication, so its copyright was not checked and it is not a source for the catalog. The
   English PDFs CDC links today are named `Eng.-<age>-Milestone-Moments-Checklist-2021-P.pdf`, under
   a `2025/05` path. The current web pages differ from the PDF items in pronouns (point 7) and, at
   9 months, in one item's wording ("Makes a lot of different sounds" on the page, "Makes different
   sounds" in the PDF); `docs/milestones/cdc.md` lists all 12 differences.
4. **Licence: public domain, with CDC's reuse conditions.** CDC's general page "Use of Agency
   Materials" says most content "is not subject to copyright, is in the public domain". Reuse needs
   attribution to the agency, a prominent statement that the use implies no endorsement, no change
   to the substantive content, and a statement that the material is "otherwise available on the
   agency website for no charge". A CDC "Watch Me!" training page lists among the qualities of the
   Act Early materials: "Reproducible (no copyright)". The checklist pages themselves carry no
   licence statement.
5. **Exceptions that matter.** CDC says some resources and images "are restricted in their use",
   namely those developed by contractors or grantees and those that "have been licensed by a third
   party". The combined checklist PDF embeds photographs whose XMP metadata reads
   `dc:rights = "Scott Housley"` with `xmpRights:Marked="True"`. **The catalog must take only the
   milestone text, never an image.**
6. **Corrected age.** The online checklists say: "If your child was born more than 3 weeks
   prematurely, use his/her corrected age." and "If your child's age falls between 2 checklist ages,
   use the checklist for the younger age." More than 3 weeks premature is a birth before 37 weeks
   0 days. That is the app's existing threshold (`PretermBelowDays = 259`, plan 4). CDC states **no
   age at which correction stops**. The app's rule (until the second birthday, spec 7.1, owner
   decision in plan 4) is the app's own.
7. **Pronouns differ between formats.** The 4-year web page has "Serves herself food or pours
   water, with adult supervision", and the 2021-named PDF has "Serves himself food or pours water,
   with adult supervision". The 2-month web page has "Smiles when you talk to or smile at her" in its list and
   "… at him" in the image caption. The transcription has to pick one source and say which. The
   plan picks the HTML age pages, the newest dated text.
8. **Size.** The combined English PDF (`/act-early/media/pdfs/2025/10/cdc-milestone-checklists-ltsae-english-508.pdf`)
   has these item counts: 2m 11, 4m 13, 6m 12, 9m 13, 12m 10, 15m 13, 18m 15, 2y 13, 30m 15, 3y 12,
   4y 17, 5y 15, which is **159 items**. 100 of them are at 2 months to 2 years and 59 at 30 months to
   5 years. The counts come from `pdftotext -layout` and the bullet glyph and were not checked item by
   item. The 4-year web page has 17 items (6+4+3+4), the same as the PDF.
9. **CDC's own framing sentence**, printed on every checklist page of the PDF: "This milestone
   checklist is not a substitute for a standardized, validated developmental screening tool. These
   developmental milestones show what most children (75% or more) can do by each age. Subject
   matter experts selected these milestones based on available data and expert consensus."
10. **What the checklists carry that the app must not transcribe.** The "Other important things to
    share with the doctor…" questions, the "You know your baby best. Don't wait… act early" box, the
    "It's time for developmental screening!" call-outs at 9, 18 and 30 months and 2 years, the
    activity tips, and the photographs. These are advice and judgement, so they fall under the
    spec's non-goal of giving no advice. Only the milestone items are transcribed.

## 2. Examples to pin in tests (verbatim from cdc.gov, 2026-11-23)

These are enough to pin the loader and the first transcription. Nothing else is transcribed here.

| Age | Domain heading | Item |
|---|---|---|
| 2 months | Social/Emotional | "Calms down when spoken to or picked up" |
| 2 months | Language/Communication | "Makes sounds other than crying" |
| 2 months | Movement/Physical Development | "Holds head up when on tummy" |
| 9 months | Cognitive | "Looks for objects when dropped out of sight (like his spoon or toy)" |
| 9 months | Movement/Physical Development | "Gets to a sitting position by herself" |
| 4 years | Language/Communication | "Says sentences with four or more words" |
| 4 years | Cognitive | "Draws a person with three or more body parts" |

## 3. Sources (retrieved 2026-11-23)

| Source | URL | What was read |
|---|---|---|
| CDC, Milestones index | https://www.cdc.gov/act-early/milestones/index.html | ages; "Updated: February 16, 2026", "Reviewed: February 17, 2026"; no licence text |
| CDC, Milestones by 2 Months | https://www.cdc.gov/act-early/milestones/2-months.html | domain headings, items, "Updated: May 16, 2026"; no licence text, no AAP mention |
| CDC, Milestones by 4 Years | https://www.cdc.gov/act-early/milestones/4-years.html | items, "Updated: May 16, 2026" |
| CDC, 9 Month Online Milestone Checklist | https://www.cdc.gov/act-early/digital-online-checklist/9-months.html | the corrected-age and "younger age" sentences |
| CDC, Milestone Checklists by Age | https://www.cdc.gov/act-early/resources/milestones-checklist-by-age.html | the per-age English PDF paths (`…/2025/05/Eng.-2-months-Milestone-Moments-Checklist-2021-P.pdf` and so on); "Updated: August 18, 2025" |
| CDC, combined English checklists PDF | https://www.cdc.gov/act-early/media/pdfs/2025/10/cdc-milestone-checklists-ltsae-english-508.pdf | item counts, the framing sentence, the photo rights metadata (read with `pdftotext` locally) |
| CDC, Use of Agency Materials | https://www.cdc.gov/other/agencymaterials.html | the public-domain statement and the four conditions, exceptions for contractor, grantee and third-party material, logos |
| CDC, Watch Me! training module 3 | https://www.cdc.gov/ncbddd/watchmetraining/module3.html | "Reproducible (no copyright)" |
| CDC Stacks record of Zubler et al. 2022 | https://stacks.cdc.gov/view/cdc/122643 | citation, DOI, PMCID; no licence statement on the record |
| AAFP editorial, "CDC's Revised Developmental Milestone Checklists" (search result only) | https://www.aafp.org/pubs/afp/issues/2022/1000/editorial-cdc-developmental-milestone-checklist.html | not opened; the search summary says 15- and 30-month checklists were added and the threshold moved from 50 % to 75 % |

Exact reuse text, as the fetch tool returned it from "Use of Agency Materials". The tool shortens
quotations, so these are fragments:

- "is not subject to copyright, is in the public domain"
- "are restricted in their use" (contractor-, grantee- or third-party-licensed material)
- "Attribution to the agency that developed the material must be provided."
- the disclaimer must be displayed "prominently and unambiguously"
- "You may not change the substantive content of the materials."
- material is "otherwise available on the agency website for no charge"
- logos "may not be used without express written permission"

## 4. Traps

- `curl` to cdc.gov answered 403 during plan 5's research. Found on 2026-11-24: curl with its default
  User-Agent gets 200 and a browser User-Agent gets 403. Fetch the HTML pages with curl; the WebFetch
  tool shortens quotations, so use it only to cross-check.
- The PDF's two-column layout splits items across lines in `pdftotext -layout` output (for example
  "Changes behavior based on where she is (place of" continues lower down). Do not transcribe from
  the PDF text. Use the HTML pages.
- Some items contain inner quotes, for example `Asks to go play with children if none are around,
  like “Can I play with Alex?”`. CDC's pages and the CSV use curly quotes (U+201C, U+201D); the CSV
  must quote such fields properly.
- The checklists go up to 5 years, but the module is off by default from 24 months (spec 7.2,
  owner). See the plan's question 3.

## 5. Not verified

- That the 2026-05-16 page updates changed nothing beyond what was compared: the 2-month and 4-year
  items were compared with the PDF, the other ten ages were not.
- Whether "Reproducible (no copyright)" and the general agency-materials page cover the milestone
  wording itself. The checklist pages carry no statement, and the revision was a CDC and AAP
  collaboration (the article is AAP's). CDC publishes the checklists as its own campaign material,
  and nothing found restricts the text.
- Whether changing a pronoun counts as a change of "substantive content". The plan avoids the
  question by transcribing verbatim.
- The AAP article's own text, including anything it says about prematurity. The AAP site answered
  403, and PubMed needs cookies.
- When CDC will next revise the checklists. No schedule was found.
