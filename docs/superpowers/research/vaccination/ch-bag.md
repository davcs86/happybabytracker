# Switzerland: Schweizerischer Impfplan, transcription research (read-only)

Researched 2026-11-22. Every number below was read from the BAG PDF itself (downloaded with curl, text via pdftotext, and Table 1 checked visually on a rendered image of p. 6). Nothing here is from memory.

## 1. Edition found

- Label: **"Schweizerischer Impfplan 2026", "Stand Februar 2026"**; series "Richtlinien und Empfehlungen"; German original; Artikelnummer 311.236.D.
- Publisher/authors (Impressum): "© Bundesamt für Gesundheit (BAG)". Authors: BAG, Abteilung Übertragbare Krankheiten, and Eidgenössische Kommission für Impffragen (EKIF) 2025 members. Cover: "Bundesamt für Gesundheit und Eidgenössische Kommission für Impffragen".
- BAG landing page (fetched 2026-11-22): "Publikationsdatum: 23.02.2026; Korrigierte Version: 16.04.2026". The PDF I used is the file currently linked there (982,758 bytes; PDF CreationDate 2026-04-09, ModDate 2026-04-20). I could not find what was corrected between 23.02 and 16.04.
- Landing page: https://www.bag.admin.ch/de/schweizerischer-impfplan (also https://www.bag.admin.ch/impfplan, as the Impressum says "Aktuelle Version im Internet").
- PDF: https://www.bag.admin.ch/dam/de/sd-web/Oj0BSN-w-Ggs/Schweizerischer%20Impfplan_2026_DE_Web_final%20(3).pdf
- The plan states: "Der Impfplan wird jeweils im ersten Quartal eines Jahres neu herausgegeben, unabhängig davon, ob Änderungen vorgenommen wurden oder nicht." So expect a "2027" edition in Q1 2027.
- Previous edition (seen only as a search hit, not read): "Schweizerischer Impfplan 2025, Stand März 2025".
- Related, separate and legally referenced document: "Art. 12a KLV Referenzdokument Prophylaktische Impfungen, Version vom 1. Juli 2026" (BAG, Direktionsbereich Kranken- und Unfallversicherung). I used it only as a cross-check; it agrees with the Impfplan on every childhood line I compared (DTP 2+1, MMR 9 months, preterm 3+1, etc.). https://www.bag.admin.ch/dam/de/sd-web/CZekGc9bJiz7/KLV_Art.%2012a_Referenzdokument%20Prophylaktische%20Impfungen%20vom%201.%20Juli%202026.pdf

### Corrections to the brief's assumptions (verified against the 2026 text)
- **MMR and varicella are at 9 and 12 months, not 12 and 15–24 months.** "Gegen Masern, Mumps, Röteln und Varizellen wird ein 2-Dosen-Schema für alle Säuglinge mit je einer Impfdosis im Alter von 9 und 12 Monaten empfohlen. Empfehlungskategorie: Basisimpfung" (ch. 1.1). MMR since 2019 on this schedule; varicella "Seit 2023 ... als Basisimpfung für alle Säuglinge (2 Impfdosen)", preferably as combined MMRV.
- Rotavirus: **ergänzende Impfung** (supplementary), since 2024, 2 doses at 2 and 4 months.
- MenB (3 doses) and MenACWY (12–18 months) for infants/toddlers: **ergänzende Impfung**, since 2024. Adolescent MenACWY + MenB at 11–15 years: ergänzende Impfung.
- HPV: Basisimpfung for girls and boys (11–14 years, 2 doses); 15–19 catch-up; 20–26 years ergänzend.
- RSV infants: a single dose of a long-acting monoclonal antibody (mAk) in the 1st year of life, treated "analog zu Basisimpfungen", i.e. in the Basis category. The 2026 plan names no product (the brief said nirsevimab; the plan only says "langwirkender mAk"). New since autumn 2025: maternal RSV vaccination (Abrysvo, SSW 32+0 to 36+0, birth due Oct–Mar) as an equivalent alternative.
- Also new in 2026: the plan no longer contains any cost-coverage statements.

## 2. Machine-readable draft

Conventions
- Ages in **months**. `from` is inclusive. `to_exclusive` is exclusive. This follows the plan's own definition: "Alter 12 Monate bedeutet ab 1. Geburtstag bis einen Tag vor dem Alter von 13 Monaten. 4–7 Jahre bedeutet vom 4. Geburtstag bis zum Tag vor dem 8. Geburtstag" (Table 1 footnote *, repeated in ch. 1.1 "Präzisierung der Altersangaben"). Applying the same rule to other columns (2 months = [2,3), 11–15 years = [132,192), 12–18 months = [12,19)) is my extension (quote_confidence medium where I did that).
- `category`: `basic` = "Basisimpfung"; `supplementary` = "ergänzende Impfung"; `risk` = Risikogruppen (chapter 3). The app must never flag `supplementary` or `risk` lines as overdue.
- Dose numbers count per antigen. A hexavalent shot is 6 antigen rows.

```yaml
source:
  title: "Schweizerischer Impfplan 2026"
  stand: "Februar 2026"
  published: "2026-02-23"
  corrected_version: "2026-04-16"
  publisher: "Bundesamt für Gesundheit (BAG); Eidgenössische Kommission für Impffragen (EKIF)"
  url: "https://www.bag.admin.ch/impfplan"
  pdf: "https://www.bag.admin.ch/dam/de/sd-web/Oj0BSN-w-Ggs/Schweizerischer%20Impfplan_2026_DE_Web_final%20(3).pdf"
  language: de

# Product to antigen mapping used below
# hexavalent (DTPa-IPV-Hib-HBV): D, T, aP, IPV, Hib, HBV
# MMRV: M, M(umps), R, VZV    (MMR + VZV given separately is the permitted alternative)
# PCV (13 or 15; 3-dose "2+1"): PCV
# dTpa / dT / DTPa-IPV (boosters): D, T, aP (not in dT), IPV (only in DTPa-IPV / dTpa-IPV)

doses:
  # ---- Infant series, category basic -------------------------------------------------
  - {antigen: D,    series: 1, from: 2,  to_exclusive: 3,  min_age: null, category: basic, wording: "2, 4 und 12 Monaten (2+1)", conf: high, notes: "via DTPa-IPV-Hib-HBV"}
  - {antigen: D,    series: 2, from: 4,  to_exclusive: 5,  min_age: null, category: basic, wording: "2, 4 und 12 Monaten (2+1)", conf: high}
  - {antigen: D,    series: 3, from: 12, to_exclusive: 13, min_age: 11,   category: basic, wording: "Alter 12 Monate; vor dem Alter von 13 Monaten abgeschlossen; nicht vor dem Mindestalter von 11 Monaten (Fussnote ***)", conf: high, notes: "min 6 months after dose 2; dose 3 may be given before 12 months but not before 11"}
  # T, aP, IPV, Hib, HBV: identical rows to D for series 1-3 (same hexavalent product)
  - {antigen: T,    series: [1,2,3], from: [2,4,12], to_exclusive: [3,5,13], min_age: [null,null,11], category: basic, conf: high}
  - {antigen: aP,   series: [1,2,3], from: [2,4,12], to_exclusive: [3,5,13], min_age: [null,null,11], category: basic, conf: high, notes: "group-care infants <5 months: physician may choose accelerated 3+1 at 2-3-4-12 months (ch. 1.1 c)"}
  - {antigen: IPV,  series: [1,2,3], from: [2,4,12], to_exclusive: [3,5,13], min_age: [null,null,11], category: basic, conf: high}
  - {antigen: Hib,  series: [1,2,3], from: [2,4,12], to_exclusive: [3,5,13], min_age: [null,null,11], category: basic, conf: high, notes: "no 4th dose; catch-up only until 5th birthday"}
  - {antigen: HBV,  series: [1,2,3], from: [2,4,12], to_exclusive: [3,5,13], min_age: [null,null,11], category: basic, conf: high, notes: "if infant series not done: see adolescent HBV row"}
  - {antigen: PCV,  series: [1,2,3], from: [2,4,12], to_exclusive: [3,5,13], min_age: null, category: basic, conf: high, notes: "vaccine must be licensed for 2+1: PCV13 or PCV15; PCV20 NOT recommended for the 2+1 infant schedule (Table 1 fn 3). A PCV20 dose before 7 months forces a 3+1 (2-4-6-12). Count doses, not product."}
  # ---- Rotavirus, supplementary -------------------------------------------------------
  - {antigen: RV,   series: 1, from: 2, to_exclusive: 3, min_age_weeks: 6, max_age_weeks_exclusive: 16, category: supplementary, wording: "ab dem Alter von 6 Wochen (Höchstalter 15 Wochen und 6 Tagen)", conf: high, notes: "oral"}
  - {antigen: RV,   series: 2, from: 4, to_exclusive: 5, min_age_weeks: null, max_age_weeks_exclusive: 24, category: supplementary, wording: "spätestens bis zum Alter von 23 Wochen und 6 Tagen ... nach der vollendeten 24. Lebenswoche kontraindiziert", conf: high, notes: "min 4 weeks between doses. Contraindicated from 24 weeks: a missed dose can never be caught up (never show as 'due' after that)."}
  # ---- MenB / MenACWY infants, supplementary --------------------------------------------
  - {antigen: MenB, series: 1, from: 2,  to_exclusive: 4,  category: supplementary, wording: "(2–)3, (4–)5 und 12–18 Monaten", conf: high, notes: "Table 1 column '2-3'; 3 and 5 months are preferred to reduce fever; may be given at 2 and 4 with other infant shots"}
  - {antigen: MenB, series: 2, from: 4,  to_exclusive: 6,  category: supplementary, conf: high, notes: "min 2 months between doses 1 and 2 (start <12 months)"}
  - {antigen: MenB, series: 3, from: 12, to_exclusive: 19, category: supplementary, conf: medium, notes: "'12–18 Monate'; upper bound 19 follows the plan's inclusive-age rule; min 6 months after dose 2; catch-up <5 years"}
  - {antigen: MenACWY, series: 1, from: 12, to_exclusive: 19, category: supplementary, wording: "Alter 12–18 Monate: 1- oder 2-Dosenschema in Abhängigkeit vom verwendeten Impfstoff", conf: high, notes: "1 or 2 doses depending on vaccine (2-dose: min 2 months apart) so the expected count is vaccine-dependent; may be given with MenB dose 3 ('Meningokokken-Impfpaket')"}
  # ---- MMR + varicella, basic -----------------------------------------------------------
  - {antigen: [M, M2, R], series: 1, from: 9,  to_exclusive: 10, category: basic, wording: "im Alter von 9 und 12 Monaten", conf: high, notes: "(M=measles, M2=mumps, R=rubella). Preferred product MMRV. Exposure/outbreak/travel: dose from 6 months, but doses before 9 months are NOT counted (then 3 doses total)."}
  - {antigen: [M, M2, R], series: 2, from: 12, to_exclusive: 13, min_age: 12, category: basic, wording: "Alter 12 Monate; 'Gabe der zweiten MMRV-Dosis im Alter von 12(–15) Monaten möglich'", conf: high, notes: "min interval 4 weeks (catch-up Table 2 fn 14). Plan allows dose 2 up to 15 months depending on measles exposure risk; for a soft 'overdue' threshold use <=16 not 13."}
  - {antigen: VZV, series: 1, from: 9,  to_exclusive: 10, category: basic, conf: high, notes: "basic since 2023; preferably MMRV"}
  - {antigen: VZV, series: 2, from: 12, to_exclusive: 13, min_age: 12, category: basic, conf: high}
  # ---- Passive RSV, basic-analog --------------------------------------------------------
  - {antigen: RSV_mAb, series: 1, from: 0, to_exclusive: 12, category: basic, wording: "Einzeldosis eines langwirkenden mAk im 1. Lebensjahr (bis 1 Tag vor dem 1. Geburtstag)", conf: high, notes: "SEASONAL, not age-windowed: born Oct-Mar -> first week of life; born Apr-Sep -> October (or ASAP). Not needed if mother got maternal RSV vaccine >=14 days before birth (exceptions: preterm <37+0 weeks, maternal immunosuppression). Do not auto-flag overdue; the maternal route is recorded on the mother, so the baby's record cannot tell."}
  # ---- Age 4-7 booster, basic -----------------------------------------------------------
  - {antigen: [D, T, aP, IPV], series: 4, from: 48, to_exclusive: 96, category: basic, wording: "Im Alter von 4–7 Jahren wird eine Auffrischimpfung ... mit einem DTPa-IPV-Kombinationsimpfstoff empfohlen", conf: high, notes: "'möglichst mit Eintritt in die obligatorische Primarstufe'. For 2+1 infant series this is dose 4 (so 3 doses by age 4 counts as complete); after a 3+1 infant series it is dose 5 for polio. dTpa-IPV allowed in DTPa-IPV supply shortage."}
  # ---- Adolescent -----------------------------------------------------------------------
  - {antigen: [D, T, aP], series: 5, from: 132, to_exclusive: 192, category: basic, wording: "Alter 11–15 Jahre ... Auffrischimpfung ... dTpa", conf: high, notes: "[132,192) uses the plan's 4–7-year birthday rule by analogy"}
  - {antigen: HBV, series: "catch-up", from: 132, to_exclusive: 192, category: basic, wording: "ungeimpften Jugendlichen im Alter von 11–15 Jahren ausdrücklich empfohlen", conf: medium, notes: "only if not vaccinated in infancy: 2 doses adult-dose vaccine at 0 and 4–6 months, or 3 doses 0-1-6 months. Table 1 shows '(HBV)' in brackets for this column."}
  - {antigen: HPV, series: 1, from: 132, to_exclusive: 180, category: basic, wording: "Im Alter von 11–14 Jahren gilt ein 2-Dosen-Schema (Zeitpunkte 0, 6 Monate)", conf: high, notes: "girls and boys; first dose must be before the 15th birthday for the 2-dose schedule"}
  - {antigen: HPV, series: 2, from: 138, to_exclusive: null, category: basic, conf: high, notes: "6 months after dose 1; 3 doses (0-2-6) if started >=15 years or immunocompromised"}
  - {antigen: HPV, series: "catch-up 15-19 y", from: 180, to_exclusive: 240, category: basic, wording: "Nachholimpfung ... 15–19 Jahren (bis zum 20. Geburtstag), 3-Dosen-Schema", conf: high}
  - {antigen: HPV, series: "20-26 y", from: 240, to_exclusive: 324, category: supplementary, wording: "im Alter von 20–26 Jahren als ergänzende Impfung", conf: high}
  - {antigen: MenACWY, series: 1, from: 132, to_exclusive: 192, category: supplementary, wording: "allen Jugendlichen im Alter von 11–15 Jahren als ergänzende Impfung", conf: high, notes: "1 dose, regardless of earlier toddler dose; catch-up until 20th birthday"}
  - {antigen: MenB, series: [1,2], from: 132, to_exclusive: 192, category: supplementary, conf: high, notes: "2 doses, min interval 1 month; regardless of earlier infant doses"}
  # ---- Adults (summary) -----------------------------------------------------------------
  - {antigen: [D, T, aP], series: booster, from: 300, to_exclusive: 312, category: basic, wording: "im Alter von 25 Jahren mit einem dTpa-Kombinationsimpfstoff", conf: high}
  - {antigen: [D, T],     series: booster, from: 540, to_exclusive: 552, category: basic, wording: "im Alter von 45 und 65 Jahren (dT)", conf: high, notes: "45 and 65; then every 10 years from 65"}
  - {antigen: [PCV, HZ, influenza, Covid-19], series: ">=65 y", from: 780, category: supplementary, conf: high, notes: "PCV20/PCV21 single dose; HZ 2 doses 0 and 2 months; influenza and Covid-19 yearly"}

risk_group_lines_not_for_everyone:   # category: risk; never overdue
  - "HBV newborns of HBsAg-positive mothers: 4 doses at 0 (with HBIG), 1, 2, 12 months (Table 1 fn 2)"
  - "Preterm <32+0 weeks or <1500 g: 4 doses 3+1 for DTPa/IPV/Hib/HBV/PCV (Referenzdokument 4.1-4.5 confirms; Impfplan ch. 3.2 c / Table 6 not transcribed)"
  - "Influenza: only risk persons (children from 6 months to 8 years: 2 doses 4 weeks apart the first time) and everyone >=65 (supplementary)"
  - "RSV mAb for children 12-24 months with continuing high RSV risk (ch. 3.1 i; details not transcribed)"
  - "PCV for risk persons 5-17 y; extra MenACWY/MenB; hepatitis A; varicella outside 'all infants' (ch. 3.1)"
```

Rows whose exact wording I did not read as a table cell but inferred from the Table 1 column header: HBV adolescent, ages given as column headers (2, 4, 12 months). The rest were read in both the table image and the running text.

## 3. Licence / terms of use

Short version: **no open licence and no explicit reuse permission was found. The only stated rule is "written permission in advance" for reproduction. Whether a data transcription needs that permission is a legal question I cannot settle; ask the BAG if you want certainty.**

Findings, with sources.
1. The Impfplan itself says "© Bundesamt für Gesundheit (BAG)" in its Impressum (last page). No Creative Commons licence, no OGD "terms of use" symbol, no reuse notice and no "Quellenangabe" wording appear in the PDF (I grepped the full text for Urheber/Copyright/Nachdruck/Quellenangabe/Weiterverwendung: only the © line matches). It offers a suggested citation: "Bundesamt für Gesundheit, Eidgenössische Kommission für Impffragen. Schweizerischer Impfplan 2026. Richtlinien und Empfehlungen. Bern: Bundesamt für Gesundheit, 2026."
2. The BAG page links its "Rechtliches" footer to https://www.admin.ch/gov/de/start/rechtliches.html. Under "Copyright" (verbatim, fetched via curl): "Copyright, Bundesbehörden der Schweizerischen Eidgenossenschaft. Die auf den Websites der Bundesbehörden enthaltenen Informationen werden der Öffentlichkeit zugänglich gemacht. Durch das Herunterladen oder Kopieren von Inhalten, Bildern, Fotos oder anderen Dateien werden keinerlei Rechte bezüglich der Inhalte übertragen. Die Urheber- und alle anderen Rechte an Inhalten, Bildern, Fotos oder anderen Dateien auf den Websites der Bundesbehörden gehören ausschliesslich diesen oder den speziell genannten Rechtsinhabern. Für die Reproduktion jeglicher Elemente ist die schriftliche Zustimmung der Urheberrechtsträger im Voraus einzuholen."
   English paraphrase: downloading confers no rights; copyright stays with the federal authorities; reproducing any element needs prior written consent of the rightsholder. No attribution formula, no OGD carve-out in that text.
3. **Not open data.** CKAN search of opendata.swiss (https://opendata.swiss/api/3/action/package_search) for "Impfplan", "Impfkalender", "Impfschema" returned 0 datasets; "Impfempfehlung" returned only the BAG FSME map dataset. So the OGD terms (https://opendata.swiss/de/terms-of-use: "Freie Nutzung", "Quellenangabe ist Pflicht" etc.) are not attached to this schedule. Do not cite them for it. Those terms say they follow "die geltende Gesetzgebung" and that raw data is "in der Regel nicht urheberrechtlich geschützt", which is general context, not a licence for this document. The federal OGD obligation (Art. 10 EMBAG, SR 172.019) concerns "Verwaltungsdaten" published as datasets; I did not establish that it covers a guidance PDF.
4. **URG Art. 5** (unofficial copy at https://www.droit-bilingue.ch/en-de/2/23/231.1-5-8.html, page dated 2021; fedlex.admin.ch itself returned only a JavaScript shell to me): "1 Durch das Urheberrecht nicht geschützt sind: a. Gesetze, Verordnungen, völkerrechtliche Verträge und andere amtliche Erlasse; b. Zahlungsmittel; c. Entscheidungen, Protokolle und Berichte von Behörden und öffentlichen Verwaltungen; d. Patentschriften und veröffentlichte Patentgesuche. 2 Ebenfalls nicht geschützt sind amtliche oder gesetzlich geforderte Sammlungen und Übersetzungen der Werke nach Absatz 1."
   - The text says "Berichte" (reports), not "amtliche Berichte" as in the brief. Whether "Richtlinien und Empfehlungen" co-authored with an expert commission fall under "Entscheidungen, Protokolle und Berichte" or "andere amtliche Erlasse" is **unclear**; I found no case law or commentary and the BAG itself asserts © on the document.
   - Possible supporting argument, unverified: since 1 Jan 2026 Art. 12a KLV refers to the Referenzdokument (not the Impfplan) for schedules and indications ("Sein Absatz 1 verweist ... seit dem 1. Januar 2026 auf das vorliegende Referenzdokument. Bis dahin hatte er auf den Schweizerischen Impfplan ... verwiesen"). That makes the Referenzdokument the better candidate for "amtlicher Erlass" treatment and the Impfplan a weaker one than before. The Referenzdokument covers reimbursable indications only.
   - Separate point: a table of dose numbers and ages is mostly facts and short phrases. Copyright (URG Art. 2) protects only intellectual creations with individual character, so a re-expressed data file with no verbatim prose is lower risk than copying text or the table layout. That is my reading, not legal advice.
5. Conclusion for a public repo plus public container image: not clearly permitted, not clearly forbidden. Practical options, cheapest first: (a) transcribe only facts (antigen, dose number, age window, category), put short wording quotes in a separate field only if needed, and attribute; (b) additionally ask the BAG in writing (contact in the Impressum: info-mt@bag.admin.ch, Abteilung Übertragbare Krankheiten, 3003 Bern, 058 463 87 06) for consent to redistribute the transcription under MIT; the admin.ch notice says to get consent in writing in advance. I did not contact them.
6. Attribution to use regardless (suggested, not mandated by any text I found): "Source: Bundesamt für Gesundheit BAG and Eidgenössische Kommission für Impffragen EKIF, Schweizerischer Impfplan 2026 (Stand Februar 2026, corrected version 16.04.2026), https://www.bag.admin.ch/impfplan. Transcribed by <project>; not an official publication; the original prevails."
   Keep the MIT licence on the code but mark the data file as "© BAG, used with attribution", not MIT, until consent exists.

## 4. Traps for the transcriber

- **Age convention**: "12 Monate" means the whole month from the 1st birthday-month start to the day before 13 months; "4–7 Jahre" means 4th birthday to the day before the 8th. Do not read "12 months" as a point; do not read "4–7" as ending at 7.0.
- **Months vs weeks**: only rotavirus is in weeks (from 6 weeks; dose 1 max 15 w 6 d; dose 2 max 23 w 6 d; contraindicated from 24 weeks). Everything else is months/years.
- **MMR/VZV at 9 and 12 months** (the brief and older material say 12 and 15–24 months, which was the pre-2019 schedule). Dose before 9 months does not count. Footnotes also allow the second MMRV dose at "12(–15)" months.
- **Hexavalent dose 3**: nominally "12 months" but allowed from 11 months and "soll vor dem Alter von 13 Monaten abgeschlossen sein". A dose 2 before 4 months changes the series to 3+1 (3rd dose 1 month later, 4th at 12 months).
- **2+1 vs 3+1**: the infant series are 2+1 (3 doses) by default. The accelerated 3+1 (2-3-4-12) is allowed for group-care infants and preterm infants. An older "3rd dose at 6 months" (pre-2019) series still needs a 4th dose from 12 months. So dose counts per antigen vary: D/T/aP/Hib/HBV/PCV = 3, then 4th at 4–7 years for D/T/aP/IPV if it was 2+1 (IPV total 4; 5 if 3+1).
- **PCV product**: PCV20 is not recommended for the 2+1 infant schedule (and a PCV20 dose before 7 months forces 3+1). Count doses, never judge the product.
- **Supplementary vs basic**: rotavirus, MenB, MenACWY, HPV at 20–26 y, and all >=65 shots (except dT) are supplementary. RSV mAb is "basic-analog". HBV is basic for infants and for unvaccinated 11–15-year-olds, risk-category otherwise. Some vaccines span several categories (the plan says so itself: HBV categories 1 and 3, pneumococcal 1, 2 and 3).
- **Vaccine-dependent counts**: MenACWY 1 or 2 doses; MenB doses depend on the start age (3 doses from <12 months, 3 from 12–23, 2 from 24–59 months). HBV adolescent 2 or 3 doses. HPV 2 or 3 doses by age at first dose or immunodeficiency. Do not hard-code one expected count.
- **Internal inconsistency in the PDF**: the RSV mAk is described as recommended "seit Januar 2024" (p. 1) and "Seit Herbst 2024" (ch. 1.1 a). Not used by any row, but do not cite a start date without checking.
- **Authorship**: publisher is the BAG ("Herausgeber © BAG"); the EKIF is co-author (cover: "Bundesamt für Gesundheit und Eidgenössische Kommission für Impffragen"; Impfplan text: BAG "in Zusammenarbeit mit der EKIF"). The KLV Referenzdokument is BAG only. Cite both organisations for the Impfplan.
- **Editions**: annual in Q1; this one was silently re-issued ("korrigierte Version 16.04.2026", reason not stated) and the file name carries "(3)". Store the PDF's sha256 and the Publikationsdatum with the data file. Language: German original; French and Italian exist ("Diese Publikation erscheint auch in französischer und italienischer Sprache") but I did not fetch them.
- **Catch-up wording** (Table 2, not transcribed in full): Hib/PCV catch-up only until the 5th birthday ("bis 1 Tag vor dem 5. Geburtstag"); MenB/MenACWY catch-up until 5th birthday and again for adolescents until the 20th birthday; HPV catch-up until the 20th birthday; varicella catch-up to age 39.
- **The Table 1 "✓" cells** mean "Impfstatus kontrollieren und nachimpfen, falls indiziert", not a scheduled dose. Do not turn them into rows.
- **Table 1 colours**: dark/black = Basis, lighter blue = ergänzend (RV, MenB, MenACWY, HPV 20–26, PCV/HZ/influenza/Covid >=65). The text layer loses this; confirm each row against ch. 1.1 text (I did).

## 5. Could not retrieve / lower confidence

- opendata.swiss/en terms page via WebFetch gave 403 (read with curl instead). admin.ch French "conditions-utilisation" URL gave 403 (not needed; German page read in full).
- fedlex.admin.ch URG text is JavaScript-only; Art. 5 quoted from droit-bilingue.ch's unofficial copy (generated 2021-01-16, says "Dies ist keine amtliche Veröffentlichung"). Re-check on fedlex before relying on it.
- Not read: French/Italian editions; the 2025 edition (so no diff of what changed); infovac.ch (secondary source not needed); EKIF's separate PCV Stellungnahme (ekif.ch); Impfplan ch. 3 (risk groups) in full and Tables 3–10; what was corrected on 16.04.2026.
- No legal advice sought or given; no contact made with BAG.

Local working files (untrusted downloads): /tmp/claude-0/-home-user/3718f41d-6166-56bb-a748-b5887a89ca86/scratchpad/dl/impfplan2026.pdf and /tmp/claude-0/-home-user/3718f41d-6166-56bb-a748-b5887a89ca86/scratchpad/ref/ref.pdf
