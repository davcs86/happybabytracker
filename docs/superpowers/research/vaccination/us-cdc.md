# US routine childhood immunization schedule: transcription research (2026-11-22)

Researcher notes. Read-only task, nothing in any repo was changed. Not legal advice.

## 0. Headline findings (read this first)

1. **The "2026" CDC schedule is NOT the one CDC publishes today.** A revised schedule was approved by the Acting CDC Director on 2026-01-05 without ACIP. On 2026-03-16 the D. Mass. court (AAP v. Kennedy, No. 1:25-cv-11916) stayed it; the stay "reverts the recommended CDC immunization schedules back to the version recommended as of May 2025" (CRS R48982, 2026-06-11). Government appealed 2026-04-29; First Circuit (No. 26-1503) heard argument 2026-10-06; no decision found as of 2026-11-22. EO 14407 (2026-05-29) tells CDC/ACIP to review the assessment. Sources: everycrsreport.com/reports/R48982.html; settlementinsight.com/cdc-lawsuit (page updated 2026-10-25, docket as of 2026-11-22); hpmmatters.publichealth.gwu.edu/american-academy-pediatrics-et-al-v-kennedy-et-al.
2. **What cdc.gov serves today (fetched 2026-11-22):** "Child and Adolescent Immunization Schedule by Age (Addendum updated July 2, 2025)", "Recommendations for Ages 18 Years or Younger, United States, **2025**". Page lines: Published August 14, 2024; Updated July 2, 2025; Reviewed July 2, 2025. Notes page (…/child-schedule-notes.html and …/child-adolescent-notes.html): Published Aug 14 2024, Updated July 1 2025, Reviewed July 2 2025. Printable PDF footer: "Revised 07/02/2025", doc id CS363989-B; PDF file creation date is 2026-04-13 (re-posted after the stay; the "restored in April 2026" claim is from one secondary source, not verified).
   - URLs: https://www.cdc.gov/vaccines/hcp/imz-schedules/child-adolescent-age.html ; notes https://www.cdc.gov/vaccines/hcp/imz-schedules/child-adolescent-notes.html ; PDF https://www.cdc.gov/vaccines/hcp/imz-schedules/downloads/child/0-18yrs-child-combined-schedule.pdf (also a B/W PDF `0-18yrs-combined-schedule-bw.pdf`).
3. **Everything in section 2 below is the 2025 edition (revised 2025-07-02)**, transcribed from the CDC PDF I downloaded (via WebFetch; copy at scratchpad `us/cdc-child.pdf`, text `us/cdc-child.txt`) and from the Notes HTML page. Bar placements in Table 1 were read from a rendered image of PDF page 2, not from extracted text (the HTML/PDF text extraction scrambles column alignment).
4. **Licence: public domain, with conditions** (attribution, non-endorsement disclaimer, do not change substantive content, state it is free on the agency site). See section 3. Redistribution of a faithful transcription in a public repo/container looks permitted; two points are unclear (what counts as "changing substantive content"; third-party content in CDC-hosted material). Flagged below.
5. **Owner decision needed (Rule 1 / "don't assume"):** which edition the app pins. Options: (a) CDC 2025 rev. 07/02/2025 = what CDC publishes now and the legally operative one under the stay; (b) AAP 2026 schedule (essentially the last ACIP-endorsed 2024-25 content; separate publisher, licence NOT checked); (c) the stayed Jan 2026 federal schedule (not retrievable from CDC; see 2.6). I recommend (a), labelled with the edition string and a "last verified" date, and a data-file field for edition so it can be swapped.

## 1. Fetch log / what I could and could not retrieve

| Item | Result |
|---|---|
| CDC schedule-by-age HTML (WebFetch) | OK, but tool returns a summary and mangled bar alignment; edition/dates reliable |
| CDC Notes HTML (…/child-schedule-notes.html) | OK (summarised by tool; used only to cross-check PDF) |
| CDC combined PDF (WebFetch saved binary) | OK. Full text via pdftotext, Table 1 bars via image render. Primary source for all rows below |
| curl to cdc.gov | 403 (bot block). Only WebFetch works for cdc.gov |
| cdc.gov/other/policies/index.html and /other/about-cdc-gov/index.html | 404 |
| https://www.cdc.gov/other/agencymaterials.html ("Use of Agency Materials") | OK, Last Reviewed October 28, 2026 |
| https://www.cdc.gov/other/link.html ("Linking to CDC.gov") | OK, last reviewed May 27, 2021 |
| CDC's January 2026 schedule | NOT retrievable (removed from CDC site 2026-03-16 per AAFP AFP Aug 2026). Only secondary descriptions (CRS, AAFP, CDC press release 2026-01-05) |
| WebFetch tool quotes | Tool limits verbatim quotes (~125 chars); term quotes in section 3 are in short pieces for that reason |

## 2. The schedule (2025 edition, rev. 07/02/2025)

### 2.1 Conventions stated by CDC (Notes p.5, verbatim in substance)
- "4 weeks = 28 days. Intervals of >=4 months are determined by calendar months."
- "Within a number range (e.g., 12-18), a dash (-) should be read as 'through.'"
- Grace: doses given <=4 days before the minimum age/interval are valid; >=5 days early are invalid and repeated.
- "A vaccine series does not need to be restarted, regardless of the time that has elapsed between doses." (Table 2)
- Table 1 colour key: yellow = range recommended for all children; green = catch-up range; purple = certain high-risk groups; checkered = can begin; light blue = shared clinical decision-making (SCDM); grey = no guidance.
- Table 1 columns: Birth, 1 mo, 2 mos, 4 mos, 6 mos, 9 mos, 12 mos, 15 mos, 18 mos, 19-23 mos, 2-3 yrs, 4-6 yrs, 7-10 yrs, 11-12 yrs, 13-15 yrs, 16 yrs, 17-18 yrs.

### 2.2 Routine dose lines, by antigen (all rows from the PDF; bar = Table 1, text = Notes pp.5-15 / Table 2 p.3)

| Antigen | Dose line as CDC prints it | Min age (CDC) | Product / combination notes |
|---|---|---|---|
| RSV-mAb (nirsevimab, Beyfortus) | Bar: "1 dose depending on maternal RSV vaccination status" Birth to ~7 mo (yellow); then "1 dose (8 through 19 months), See Notes" (purple, high-risk). Notes: born Oct-Mar: 1 dose within 1 week of birth, ideally in hospital; born Apr-Sep: 1 dose shortly before RSV season; not needed if mother had Abrysvo >=14 days before delivery. Second-season dose at 8-19 mo only for chronic lung disease of prematurity, severe immunocompromise, CF with severity criteria, or American Indian/Alaska Native. Extra dose after cardiac surgery with bypass | birth | A monoclonal antibody, not a vaccine. Only nirsevimab is named in this PDF (0 hits for clesrovimab/Enflonsia in the text); the Washington DOH 2026 (AAP-adapted) schedule names "nirsevimab/clesrovimab" |
| HepB | "1st dose" Birth; "2nd dose" 1-2 mo; "3rd dose" 6-18 mo. Notes: "3-dose series at age 0, 1-2, 6-18 months"; birth dose within 24 h if >=2,000 g and stable; <2,000 g: 1 dose at chronological age 1 month or hospital discharge. "Administration of 4 doses is permitted when a combination vaccine containing HepB is used after the birth dose." Final dose age 6-18 mo | birth; final dose 24 weeks. Table 2: dose1-2 >=4 wk, dose2-3 >=8 wk and >=16 wk after dose 1 | Birth dose monovalent only. Mother HBsAg-positive: final dose at 6 mo, HBIG, test at 9-12 mo. Pediarix (DTaP-HepB-IPV), Vaxelis contain HepB |
| RV | 1st dose 2 mo; 2nd dose 4 mo; 6 mo cell "See Notes". Rotarix 2 doses (2, 4 mo); RotaTeq 3 doses (2, 4, 6 mo). "If any dose in the series is either RotaTeq or unknown, default to 3-dose series" | 6 weeks. Do not start on/after 15 wk 0 d (Table 2: max age dose 1 = 14 wk 6 d); max age for final dose 8 mo 0 d | Product-dependent count (2 vs 3) |
| DTaP | 1st 2 mo; 2nd 4 mo; 3rd 6 mo; 4th 15-18 mo; 5th 4-6 yrs. "5-dose series (3-dose primary series at age 2, 4, and 6 months, followed by booster doses at ages 15-18 months and 4-6 years)" | 6 weeks (4 years for Kinrix/Quadracel). Dose 4 prospectively valid from 12 mo if >=6 months after dose 3 (retrospectively valid if >=4 months). Table 2 intervals 4 wk, 4 wk, 6 mo, 6 mo | "Dose 5 is not necessary if dose 4 was administered at age 4 years or older and at least 6 months after dose 3." Combos: Pediarix, Pentacel, Vaxelis, Kinrix, Quadracel. Td may replace DTaP only for the pertussis-specific contraindication |
| Hib | 1st 2 mo; 2nd 4 mo; 6 mo cell "See Notes"; "3rd or 4th dose (See Notes)" 12-15 mo. ActHIB/Hiberix/Pentacel/Vaxelis: 3-dose primary (2, 4, 6 mo) + booster 12-15 mo = 4 doses. PedvaxHIB: 2-dose primary (2, 4 mo) + booster 12-15 mo = 3 doses | 6 weeks | **Product-dependent (3 vs 4 doses).** "Vaxelis is not recommended for use as a booster dose." AI/AN infants: Vaxelis or PedvaxHIB preferred. Table 2 treats "unknown" product like PRP-T/Vaxelis. Hib catch-up not required for healthy children >=60 months |
| PCV (PCV15, PCV20) | 1st 2 mo; 2nd 4 mo; 3rd 6 mo; 4th 12-15 mo. "4-dose series at 2, 4, 6, 12-15 months" | 6 weeks | Healthy children 2-4 y with an incomplete series: 1 dose. PPSV23 (min age 2 y) only for risk conditions, not routine. PCV15 and PPSV23 not at the same visit |
| IPV | 1st 2 mo; 2nd 4 mo; 3rd 6-18 mo; 4th 4-6 yrs. "4-dose series at ages 2, 4, 6-18 months, 4-6 years" | 6 weeks. "Final dose on or after age 4 years and at least 6 months after the previous dose." Table 2: 4 wk, 4 wk, 6 mo | "4 or more doses of IPV can be administered before age 4 years when a combination vaccine containing IPV is used. However, a dose is still recommended on or after age 4 years..." (so count of doses is NOT a reliable completion test for IPV). Combos: Pediarix, Pentacel, Kinrix, Quadracel, Vaxelis |
| COVID-19 | Table 1: light blue (SCDM) from 6 mo through 17 yrs; yellow at age 18. Notes: "Shared clinical decision-making: Ages 6 months-17 years who are NOT moderately or severely immunocompromised." Page 4 note: "As of May 29, 2025, the schedule incorporates the HHS directive regarding COVID-19 vaccine recommendations" | 6 months (Moderna, Pfizer-BioNTech); 12 years (Novavax) | Dose counts depend on product and prior doses and name the "2024-25" formula (edition-specific). Unvaccinated 6 mo-4 y: use one manufacturer; Moderna 2 doses (0, 4-8 wk) or Pfizer 3 doses (0, 3-8 wk, >=8 wk after dose 2) per the Notes HTML summary; Table 1 itself says only "See Notes" |
| Influenza (IIV3/ccIIV3; LAIV3) | Bar: "1 or 2 doses annually" from 6 mo to ~9 y; "1 dose annually" from ~9 y. LAIV3 bar starts at 2 yrs. Notes: age 6 mo-8 y with fewer than 2 doses before July 1, 2024 (or unknown): 2 doses >=4 weeks apart (dose 2 even if child turns 9); with >=2 prior doses: 1 dose; >=9 y: 1 dose | 6 months (IIV3), 2 years (LAIV3), 18 years (RIV3) | **Recurring annually.** The "before July 1, 2024" cut-off is edition/season-specific and rolls yearly. Egg allergy: any age-appropriate vaccine |
| MMR | 6-11 mo cell "See Notes" (purple: travel dose, does not count); 1st dose 12-15 mo; 2nd dose 4-6 yrs. "2-dose series at age 12-15 months, age 4-6 years" | 12 months (routine). Catch-up interval 4 weeks | MMRV (ProQuad) allowed; max age for MMRV 12 y; MMRV minimum interval 3 months; dose 1 at 12-47 mo: separate MMR + VAR recommended unless parent prefers MMRV |
| VAR | 1st dose 12-15 mo; 2nd dose 4-6 yrs. "Dose 2 may be administered as early as 3 months after dose 1 (a dose inadvertently administered after at least 4 weeks may be counted as valid)" | 12 months | MMRV as above. Catch-up 7-18 y: interval 3 months (age 7-12), 4-8 wk (>=13; min 4 wk) |
| HepA | 6-11 mo cell "See Notes" (travel only); "2-dose series (See Notes)" across 12 mo to 19-23 mos. "2-dose series (minimum interval: 6 months) at age 12-23 months" | 12 months. Interval >=6 months | Travel infants 6-11 mo: 1 dose before departure (does not count), revaccinate with 2 doses at 12-23 mo. Unvaccinated through 18 y complete 2 doses |

### 2.3 Ages 7-18 years (summary, same sources)
- **Tdap**: "Age 11-12 years: 1 dose Tdap (adolescent booster)"; catch-up 13-18 if none: 1 dose; age 7-18 not fully vaccinated with DTaP: 1 Tdap as part of catch-up (min age 7 y; 11 y for routine). Given at 7-9 y: still needs the 11-12 y booster; given at 10 y: no booster. Also 1 dose each pregnancy (27-36 wk).
- **HPV**: "routinely recommended at age 11-12 years (can start at age 9 years)", catch-up through 18. Start at 9-14: 2 doses at 0, 6-12 months (min interval 5 months). Start at >=15: 3 doses at 0, 1-2, 6 months. 3 doses for immunocompromising conditions regardless of age. Min age 9 y. (Bar: checkered "can begin" in the 7-10 yrs column, "See Notes" at 11-12.)
- **MenACWY**: "2-dose series at age 11-12 years; 16 years". Catch-up 13-15: 1 dose + booster at 16-18 (min 8 weeks); 16-18: 1 dose. Purple "See Notes" from 2 mo through 10 y for high-risk only (asplenia, HIV, complement deficiency/inhibitors; Menveo 4-dose series 2, 4, 6, 12 mo). Min age 2 mo (MenACWY-CRM Menveo), 2 y (MenACWY-TT MenQuadfi), 10 y (Penbraya).
- **MenB**: SCDM for ages 16-23 (preferred 16-18) "not at increased risk": 2 doses of same brand >=6 months apart (3 if dose 2 early). High-risk from 10 y: 3 doses 0, 1-2, 6 months. Min age 10 y. Penbraya (MenACWY-TT/MenB-FHbp) for same-day use; the addendum adds Penmenvy (MenABCWY, GSK), effective 2025-06-25.
- **Influenza**: 1 dose annually from age 9 (see above).
- **COVID-19**: SCDM through 17; recommended at 18 (Notes: "Age 18 years who are NOT moderately or severely immunocompromised... Unvaccinated: ..."; confidence medium because I read this from the yellow bar and the heading, not from a single clean sentence).
- **Not routine/outside 0-18 scope for this app**: Dengue (DEN4CYD, 9-16 y, endemic area AND lab-confirmed prior infection, 3 doses 0, 6, 12 months); Mpox (Jynneos, min 18 y, risk-based); RSV Abrysvo (pregnancy, 32 wk 0 d-36 wk 6 d, Sept-Jan); PPSV23 (risk conditions, min age 2 y).

### 2.4 Combination products (CDC page 1 table)
DTaP-HepB-IPV (Pediarix); DTaP-IPV/Hib (Pentacel); DTaP-IPV (Kinrix, Quadracel); DTaP-IPV-Hib-HepB (Vaxelis); MMRV (ProQuad). CDC line: "use combination vaccines instead of separate injections when appropriate". Not combinations but "or" products: influenza IIV3/ccIIV3 vs LAIV3; Penbraya/Penmenvy for meningococcal.

### 2.5 Machine-readable draft (ages 0-6 y in full; 7-18 y summary)

Conventions for this file:
- `recommended_from_months`/`recommended_to_months` are the numbers CDC prints in the Notes ("12-15 months", "4-6 years" -> 48-72). CDC says a dash means "through", so the end is probably inclusive of the whole last unit (e.g. "through 15 months" could mean up to 15 mo 30 d). **That boundary is unresolved**; leave the choice to the owner. For a dose printed as one age ("at 2 months") CDC prints no end; the Table 1 bar is a single column and the catch-up (green) bar begins at the next column. I set `to` = `from` and say so.
- Where Notes give no number and I derived a value, `quote_confidence` is medium/low and the comment says "derived".
- `minimum_age_days` is given where CDC states a minimum age (weeks converted at 7 days/week; years at 365.25 d/yr rounded). Subsequent-dose minimum ages are mostly defined by minimum INTERVAL (Table 2), not age; left blank rather than guessed.
- `count_rule` records how the app should count when product-dependent.
- `routine`: yes = yellow (all children); risk = purple; scdm = shared decision-making.

```csv
antigen,series_number,recommended_from_months,recommended_to_months,minimum_age_days,routine,recurring,quote_confidence,notes
RSV_mAb,1,0,8,0,yes,no,medium,"Nirsevimab. Birth 'within 1 week of birth' if born Oct-Mar; 'shortly before start of RSV season' if born Apr-Sep. Skip if mother had Abrysvo >=14 d before delivery. 'to'=8 is where the 8-19 mo bar starts; CDC prints no number for the first-season window"
RSV_mAb,2,8,19,,risk,no,high,"'1 dose (8 through 19 months), See Notes': only chronic lung disease of prematurity / severe immunocompromise / CF-severe / AI-AN. Not routine"
HepB,1,0,0,0,yes,no,high,"'1 dose within 24 hours of birth if medically stable' and >=2000 g; monovalent only"
HepB,2,1,2,,yes,no,high,"'3-dose series at age 0, 1-2, 6-18 months'; min interval 4 wk after dose 1 (Table 2)"
HepB,3,6,18,168,yes,no,high,"'Final (3rd or 4th) dose: age 6-18 months (minimum age 24 weeks)'; 4 doses allowed when combination vaccine used; dose-3 interval >=8 wk after dose 2 and >=16 wk after dose 1"
RV,1,2,2,42,yes,no,high,"Min 6 wk; do not start on/after 15 wk 0 d (Table 2: max dose-1 age 14 wk 6 d)"
RV,2,4,4,,yes,no,high,"Rotarix: 2 doses (2, 4 mo)"
RV,3,6,6,,yes,no,high,"RV5/RotaTeq or unknown product only ('default to 3-dose series'); max age for final dose 8 mo 0 d. count_rule: required_doses=2 if all Rotarix else 3"
DTaP,1,2,2,42,yes,no,high,"Min 6 wk (4 y for Kinrix/Quadracel)"
DTaP,2,4,4,,yes,no,high,"min interval 4 wk"
DTaP,3,6,6,,yes,no,high,"min interval 4 wk"
DTaP,4,15,18,365,yes,no,high,"'booster doses at ages 15-18 months'; dose 4 'may be administered as early as age 12 months if at least 6 months have elapsed since dose 3'; 365 d is the prospective minimum age"
DTaP,5,48,72,,yes,no,high,"'4-6 years'. 'Dose 5 is not necessary if dose 4 was administered at age 4 years or older and at least 6 months after dose 3'"
Hib,1,2,2,42,yes,no,high,"All products. Min 6 wk"
Hib,2,4,4,,yes,no,high,"All products"
Hib,3,6,6,,yes,no,medium,"ONLY ActHIB/Hiberix/Pentacel/Vaxelis (or unknown product). PedvaxHIB has no 6-mo dose. Table 1 cell reads 'See Notes'"
Hib,4,12,15,,yes,no,high,"Booster 'age 12-15 months'. This is dose 4 for non-PedvaxHIB, dose 3 for PedvaxHIB ('3rd or 4th dose'); Vaxelis not for the booster. count_rule: required_doses=4 (PRP-T/Vaxelis/unknown) or 3 (PedvaxHIB only)"
PCV,1,2,2,42,yes,no,high,"PCV15/PCV20. Min 6 wk"
PCV,2,4,4,,yes,no,high,""
PCV,3,6,6,,yes,no,high,""
PCV,4,12,15,,yes,no,high,"'4-dose series at 2, 4, 6, 12-15 months'"
IPV,1,2,2,42,yes,no,high,"Min 6 wk"
IPV,2,4,4,,yes,no,high,""
IPV,3,6,18,,yes,no,high,"'4-dose series at ages 2, 4, 6-18 months, 4-6 years'"
IPV,4,48,72,1461,yes,no,high,"'final dose on or after age 4 years and at least 6 months after the previous dose'. >=4 doses can be given before age 4 via combination vaccines, but a dose at >=4 y is still needed"
COVID19,1,6,215,183,scdm,no,medium,"6 mo-17 y not immunocompromised: 'Shared clinical decision-making' only; no routine dose. from=6 (min age 6 mo Moderna/Pfizer; 12 y Novavax). Dose count depends on product/prior doses/'2024-25' formula (see notes). 'to' = 17 y 11 mo (215) is the end of the SCDM band, not a due date. DO NOT flag overdue"
Influenza,1,6,null,183,yes,yes,high,"Annual from 6 mo. 6 mo-8 y: 2 doses >=4 wk apart in first season (or if <2 prior doses before July 1 of the reference year); else 1. >=9 y: 1 dose. LAIV3 min age 2 y. No age end; recurring each season: do not use 'overdue by age window'"
MMR,1,12,15,365,yes,no,high,"'2-dose series at age 12-15 months, age 4-6 years'. Dose before 12 mo (travel, 6-11 mo) does not count"
MMR,2,48,72,,yes,no,high,"Min interval 4 wk after dose 1. MMRV allowed (max age 12 y)"
VAR,1,12,15,365,yes,no,high,""
VAR,2,48,72,,yes,no,high,"'Dose 2 may be administered as early as 3 months after dose 1'"
HepA,1,12,23,365,yes,no,medium,"'2-dose series (minimum interval: 6 months) at age 12-23 months': CDC gives one window for the series"
HepA,2,18,23,,yes,no,low,"DERIVED: 12 mo + 6 mo min interval = 18 mo. CDC does not print a separate window for dose 2"
```

7-18 y (summary rows, same CSV columns; `to` in months):

```csv
antigen,series_number,recommended_from_months,recommended_to_months,minimum_age_days,routine,recurring,quote_confidence,notes
Tdap,1,132,144,2557,yes,no,high,"'Age 11-12 years: 1 dose Tdap (adolescent booster)'; min age 11 y routine, 7 y (2557 d) catch-up. Catch-up 13-18 if none. Tdap at 7-9 y still needs 11-12 y dose"
HPV,1,132,144,3288,yes,no,high,"'routinely recommended at age 11-12 years (can start at age 9 years)'; min age 9 y (3288 d); catch-up through 18"
HPV,2,138,156,,yes,no,medium,"Start 9-14 y: 2 doses at 0, 6-12 months (min interval 5 mo). Start >=15 y: 3 doses (0, 1-2, 6 mo). 3 doses if immunocompromised. 'from/to' for dose 2 is DERIVED (dose 1 at 11-12 + 6-12 mo)"
MenACWY,1,132,144,,yes,no,high,"'2-dose series at age 11-12 years; 16 years'. Min age 2 mo (CRM)/2 y (TT)/10 y (Penbraya)"
MenACWY,2,192,192,,yes,no,high,"Booster at 16 y. Catch-up 13-15: 1 dose now + booster 16-18 (>=8 wk); 16-18: 1 dose"
MenB,1,192,276,3653,scdm,no,high,"SCDM only: age 16-23 y (preferred 16-18 y), 2 doses same brand >=6 mo apart; high-risk 3 doses from 10 y. Not routine; DO NOT flag overdue"
Influenza,1,108,null,,yes,yes,high,"Age >=9 y: 1 dose annually"
COVID19,1,216,null,,yes,no,medium,"Age 18: recommended (yellow); 6 mo-17 y SCDM"
```

Out of scope (rows deliberately omitted): Dengue, Mpox, RSV (Abrysvo), PPSV23, Td, MenABCWY.

### 2.6 What the stayed January 2026 schedule changed (SECONDARY sources, lower confidence, for owner awareness only)
From CRS R48982 (2026-06-11) and AAFP AFP Aug 2026 (aafp.org/afp/2026/0800/...); CDC's own 2026 text could not be retrieved:
- No vaccines removed. Three tiers: routine, high-risk, SCDM.
- Moved to SCDM / risk-based: HepA, HepB (still recommended for infants of HBsAg-positive/unknown mothers), MenACWY, RV, influenza, COVID-19. RSV monoclonal recategorised (same population). Dengue and MenB: specific high-risk groups.
- HPV: two or three doses -> one dose (AFP notes FDA has not approved single-dose HPV).
- CDC release (2026-01-05) lists "recommended for all children": measles, mumps, rubella, polio, pertussis, tetanus, diphtheria, Hib, pneumococcal, HPV, varicella.
- CRS also says the Sept 2025 COVID and Dec 2025 HepB ACIP actions were incorporated; the stay also suspended 13 ACIP appointments and their votes.
- If the app pinned the 2026 federal schedule it would contradict what CDC publishes today and what AAP (and the courts' reversion) treat as current. Do not use it without an explicit owner decision.

## 3. Licence / terms of reuse

Primary text: "Use of Agency Materials", https://www.cdc.gov/other/agencymaterials.html, Last Reviewed October 28, 2026 (fetched 2026-11-22). Quotes below are exact fragments (the fetch tool returns short quoted pieces):

- Status: "Most of the information on the CDC and ATSDR websites is not subject to copyright, is in the public domain," and "may be freely used or reproduced without obtaining copyright permission."
- Conditions (paraphrase of four requirements, with exact fragments):
  1. Credit: "Attribution to the agency that developed the material must be provided in your use of the materials."
  2. Disclaimer: "You must utilize a disclaimer which clearly indicates that your use of the material, including any links to the materials on the CDC, ATSDR or HHS websites, does not imply endorsement by CDC, ATSDR, HHS or the United States Government of you, your company, product, facility, service or enterprise."
  3. No alteration: "You may not change the substantive content of the materials".
  4. Free availability: "You must state that the material is otherwise available on the agency website for no charge."
- Exceptions: "some resources, as well as images, on the CDC and ATSDR websites are restricted in their use because they were developed by government contractors or grantees, or have been licensed by a third party."; "the U.S. government work designation does not apply to works of state and local governments"; "Copyright-protected materials featured on the CDC and ATSDR websites should include a copyright statement."
- Logos: "The CDC and/or ATSDR logo and other related CDC, ATSDR or Department of Health and Human Services (HHS) logos may not be used without express written permission from CDC, ATSDR and/or HHS." and they "should never be used to promote or suggest endorsement of non-CDC, ATSDR or HHS companies".
- Re-posting: "we urge you to link directly to our resource documents rather than re-posting." and "If you do re-post, please check back periodically to see if there are revisions." (urging, not a prohibition).
- Linking page (https://www.cdc.gov/other/link.html, reviewed 2021-05-27): CDC.gov is "a public domain website, which means you may link to CDC.gov at no cost and without specific permission"; "A link does not indicate any form of endorsement or approval from CDC, or the U.S. Department of Health and Human Services."

The schedule document itself:
- The PDF has no copyright notice and no rights/licence field in its visible text. Its only raster image is a QR code. Visible text includes: "The use of trade names is for identification purposes only and does not imply endorsement by the ACIP or CDC." (this is the CDC's own disclaimer for the product names, which the data file should carry if it keeps trade names).
- PDF metadata "Subject" says: "Recommended by the Advisory Committee on Immunization Practices (www.cdc.gov/vaccines/acip) and approved by the Centers for Disease Control and Prevention, American Academy of Pediatrics, American Academy of Family Physicians, American College of Obstetricians and Gynecologists, American College of Nurse-Midwives, American Academy of Physician Associates, and National Association of Pediatric Nurse Practitioners." Author field: "CDC/NCIRD". I found no endorsement logos in the PDF text/images (vector logos not ruled out; I read pages 1-3 as text and page 2 as an image only). The Notes cite the AAP Red Book (a citation, not reproduced text).
- Did not find a statement that the schedule is third-party copyrighted. I did not check whether the HTML pages carry per-page copyright footers.

Conclusions (my reading, not legal advice):
- A transcription of CDC-authored schedule data into a data file in a public repo and a public container image appears permitted without permission, because the content is stated to be public domain and may be "freely used or reproduced". No licence is imposed (MIT does not apply to the data; mark the data file "US government work / public domain; transcribed from CDC").
- Conditions to meet: (1) attribute CDC/NCIRD, edition ("Recommended Child and Adolescent Immunization Schedule for Ages 18 Years or Younger, United States, 2025, revised 07/02/2025") and source URL; (2) carry a non-endorsement disclaimer in the data file/README/UI (CDC, HHS, ACIP, AAP etc. do not endorse the app); (3) state that the material is available from CDC free of charge (link to the CDC page); (4) do not alter substance: transcribe faithfully, keep CDC's wording for conditions, and keep any app-side simplification (e.g. counting by antigen, dropping risk-group lines) clearly separate from the transcribed table. No CDC/HHS logos.
- Unclear points: (a) what counts as "changing the substantive content" for a reformatted/derived table (the app deliberately omits risk-group rules and does not judge validity) - safest is to label the file "excerpt/summary of" and link to the authoritative source, and to display "not medical advice"; (b) whether any third-party-licensed content sits inside the specific PDF (no marking found); (c) the rules for the 2026 federal edition are the same site policy, but its text was unavailable; (d) AAP's own schedule is a separate publication, copyright/licence NOT checked; (e) the earlier-cited path cdc.gov/other/policies/index.html returned 404, so the "Website policies" index was not reached. The "Use of Agency Materials" page is the operative CDC policy page I found.

## 4. Traps for a transcriber

1. **Edition trap**: CDC pages say "2025" and "Addendum updated July 2, 2025" in November 2026 because the 2026 revision was stayed. Search results for "2026 CDC schedule" mostly describe the stayed version, state-health-department copies, or AAP's schedule. Check the edition string on the page.
2. **Bars vs Notes**: Table 1 columns are non-uniform (6 mos, 9 mos, 12 mos, 15 mos, 18 mos, 19-23 mos, 2-3 yrs...). The bar spans reflect column boundaries, not exact ages; use Notes ranges ("12-15 months", "15-18 months", "6-18 months", "1-2 months"). "Dash means through" (Notes p.5) leaves the end-of-range boundary ambiguous (through the start or the end of the last month).
3. **Weeks vs months**: min ages are in weeks (6 weeks; 24 weeks for final HepB dose; RV "15 weeks 0 days", "14 weeks 6 days", "8 months 0 days"). Intervals are weeks (4 wk = 28 d) or calendar months (>=4 months). Doses <=4 days early are valid (grace); >=5 days early are invalid.
4. **"1-2 months" HepB dose 2** and "6-18 months" dose 3/IPV dose 3 are wide windows; the bar for HepB shows a green catch-up cell at 4 mos between them.
5. **Birth doses**: HepB conditional on weight/maternal status; nirsevimab depends on birth month (Oct-Mar vs Apr-Sep), maternal Abrysvo timing (>=14 days before delivery), and a second-season dose for high-risk only.
6. **Product-dependent counts**: Hib 3 vs 4 (PedvaxHIB vs others; Vaxelis not as booster); RV 2 vs 3 (Rotarix vs RotaTeq/unknown); HepB 3 or 4 when combos used; IPV can be >=4 doses via combos while a dose at >=4 y is still required.
7. **"Not needed" lines**: DTaP dose 5 unneeded if dose 4 at >=4 y and >=6 mo after dose 3; IPV 4th dose not needed (catch-up) if 3rd at >=4 y and >=6 mo after previous; Hib: no catch-up for healthy >=60 months, one dose at >=15 months suffices; PCV: one dose if >=24 months healthy.
8. **Early-dose traps**: MMR/HepA dose at 6-11 months (travel) does not count; DTaP dose 4 valid from 12 months; VAR dose 2 can be at 3 months after dose 1 (4 weeks inadvertent).
9. **Recurring / SCDM**: Influenza (annual; first-season 2 doses under 9 y) and COVID-19/MenB (SCDM) must not drive an "overdue" flag. Flu "fewer than 2 doses before July 1, 2024" is edition/season-specific; COVID text names the "2024-25" formula.
10. **COVID row**: the Table 1 cell reads only "See Notes" on light blue (SCDM); the dosing text is long, product- and history-dependent; the July 2025 revision incorporates the 2025-05-29 HHS directive. Record "shared clinical decision-making", no routine age window.
11. **Combination products**: schedule is by antigen; products (Pediarix, Pentacel, Vaxelis, Kinrix, Quadracel, ProQuad) contribute doses to several antigens. Kinrix/Quadracel minimum age is 4 years. Vaxelis is not a Hib booster. HepB birth dose monovalent only.
12. **Ranges beyond 6 years that matter**: HPV 2 vs 3 doses depends on age at dose 1 (9-14 vs >=15) and immunocompromise; MenACWY 2 doses (11-12 and 16) but 1 dose if started 16-18; MenB SCDM 16-23; Tdap given at 7-9 y still needs the 11-12 y dose.
13. **Text extraction**: cdc.gov HTML and PDF text extraction (and the fetch tool's summaries) lose bar alignment; I verified bar positions against a rendered image of PDF page 2. A future refresh should do the same (render, don't trust text).
14. **Clesrovimab** is not in the CDC 2025 PDF; whether the app should include it needs the owner's call (it appears in AAP-adapted 2026 schedules).

## 5. Sources (all fetched 2026-11-22 unless noted)
- CDC schedule by age (HTML, edition 2025 rev. July 2, 2025): https://www.cdc.gov/vaccines/hcp/imz-schedules/child-adolescent-age.html
- CDC Notes: https://www.cdc.gov/vaccines/hcp/imz-schedules/child-adolescent-notes.html and https://www.cdc.gov/vaccines/schedules/hcp/imz/child-schedule-notes.html
- CDC PDF (primary for all rows): https://www.cdc.gov/vaccines/hcp/imz-schedules/downloads/child/0-18yrs-child-combined-schedule.pdf (local copy: /tmp/claude-0/-home-user/3718f41d-6166-56bb-a748-b5887a89ca86/scratchpad/us/cdc-child.pdf)
- CDC reuse policy: https://www.cdc.gov/other/agencymaterials.html ; linking: https://www.cdc.gov/other/link.html
- Status/litigation: https://www.everycrsreport.com/reports/R48982.html ; https://settlementinsight.com/cdc-lawsuit ; https://hpmmatters.publichealth.gwu.edu/american-academy-pediatrics-et-al-v-kennedy-et-al
- 2026 schedule changes (secondary): https://www.aafp.org/afp/2026/0800/editorials-practice-alert-child-adolescent-immunization-schedule ; https://www.cdc.gov/media/releases/2026/2026-cdc-acts-on-presidential-memorandum-to-update-childhood-immunization-schedule.html
- Washington DOH 2026 AAP-adapted schedule (clesrovimab mention only): https://doh.wa.gov/sites/default/files/2026-09/348-1164-ChildImmunizationSchedule.pdf
