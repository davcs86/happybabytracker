# Vaccination schedule: United States (`us`)

Generated from `api/internal/vaccines/data/us.csv`; do not edit. After changing the CSV, run
`UPDATE_DOCS=1 go test ./internal/vaccines/` in `api/` and commit both files.

- **Authority:** Centers for Disease Control and Prevention (CDC)
- **Edition:** CDC Child and Adolescent Immunization Schedule by Age, 2025 (addendum updated July 2, 2025)
- **Retrieved:** 2026-11-22
- **Source:** https://www.cdc.gov/vaccines/hcp/imz-schedules/child-adolescent-age.html
- **Licence:** Public domain (a work of the United States government), reused on CDC's conditions at https://www.cdc.gov/other/agencymaterials.html: credit the agency, state that its use implies no endorsement, do not change the substantive content, and state that the material is available free of charge on the agency website.
- **Attribution:** Source: CDC, Recommended Child and Adolescent Immunization Schedule for Ages 18 Years or Younger, United States, 2025 (revised 07/02/2025). CDC does not endorse this application or its use of the material. The material is available free of charge on the CDC website at https://www.cdc.gov/vaccines/hcp/imz-schedules/child-adolescent-age.html. This is a partial, simplified transcription and must not be read as CDC's own text: risk-group, travel, pregnancy, catch-up and shared-decision dose rules and the items listed in the review document are not transcribed; see the CDC schedule for the full text. Trade names are CDC's: “The use of trade names is for identification purposes only and does not imply endorsement by the ACIP or CDC.”

## Kinds

- `routine`: due inside its window, overdue after it.
- `may_not_be_needed`: depends on the product or history; is due inside its window; afterwards it is listed as available and never overdue.
- `supplementary`: shared decision-making or risk groups; listed under "Also available", never due or overdue.
- `recurring`: seasonal; listed with the doses given, never due or overdue.

A window runs from the age in `from` up to, not including, the age in `until`; without `until` it is 1 month.

## Not transcribed from the source

- COVID-19: the routine dose at age 18 years and the additional doses for children who are moderately or severely immunocompromised
- HPV: the 3rd dose of the 3-dose series for children who start at age 15 years or older or have an immunocompromising condition
- MenACWY: the doses for high-risk groups from age 2 months (asplenia, HIV infection, complement deficiency or inhibitor use) and their boosters
- MenB: the 3-dose series for high-risk groups from age 10 years, its boosters, and the 3-dose option for rapid protection
- RSV monoclonal antibody: the additional dose after cardiac surgery with cardiopulmonary bypass
- IPV: the one lifetime booster at age 18 years for adolescents at increased risk of exposure to poliovirus
- Tdap: 1 dose during each pregnancy
- RSV vaccine (Abrysvo) during pregnancy
- Dengue vaccine: age 9–16 years in endemic areas with laboratory-confirmed previous dengue infection
- Mpox vaccine: age 18 years at risk for mpox infection
- Pneumococcal: PPSV23 and the extra PCV doses for children with risk conditions
- Hib: the doses for children with chemotherapy, stem cell transplant, asplenia, HIV infection or immunoglobulin or complement deficiency
- Hepatitis B: the 4-dose series and HBIG for infants of HBsAg-positive or HBsAg-unknown mothers, and post-vaccination testing
- MMR and hepatitis A: the dose at 6–11 months before international travel (it does not count) and the MMR dose in mumps outbreaks
- Influenza: the 2nd dose in a season for children aged 6 months–8 years with fewer than 2 earlier doses (one recurring line stands for every season)
- Td in place of DTaP for a pertussis-specific contraindication, and tetanus doses for wound management
- Catch-up schedules (Table 2) and the recommendations by medical condition (Table 3)

## Birth

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| Hepatitis B | Hepatitis B | 1 | from Birth until 1 month (default window) | `routine` | CDC: “Birth”. “Birth weight ≥2,000 grams: 1 dose within 24 hours of birth if medically stable”; “Birth weight <2,000 grams: 1 dose at chronological age 1 month or hospital discharge”. Monovalent HepB only. |
| RSV | Respiratory syncytial virus (RSV) | 1 | from Birth until 8 months | `may_not_be_needed` | (verify) end of window derived: CDC prints no end age for this dose; 8 months is where the second-season bar “1 dose (8 through 19 months)” starts. Table 1 shows it in yellow (all children): “1 dose depending on maternal RSV vaccination status”; “Mother received RSV vaccine at least 14 days prior to delivery: nirsevimab not needed”. Born October–March: “administer 1 dose nirsevimab within 1 week of birth”; born April–September: “1 dose nirsevimab shortly before start of RSV season”. Nirsevimab is a monoclonal antibody. |

## 1 month

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| Hepatitis B | Hepatitis B | 2 | from 1 month until 3 months | `routine` | “3-dose series at age 0, 1–2, 6–18 months”. |

## 2 months

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| DTaP | Diphtheria | 1 | from 2 months until 3 months (default window) | `routine` | “5-dose series (3-dose primary series at age 2, 4, and 6 months, followed by booster doses at ages 15–18 months and 4–6 years)”. |
| DTaP | Pertussis (whooping cough) | 1 | from 2 months until 3 months (default window) | `routine` | “5-dose series (3-dose primary series at age 2, 4, and 6 months, followed by booster doses at ages 15–18 months and 4–6 years)”. |
| DTaP | Tetanus | 1 | from 2 months until 3 months (default window) | `routine` | “5-dose series (3-dose primary series at age 2, 4, and 6 months, followed by booster doses at ages 15–18 months and 4–6 years)”. |
| Hib | Haemophilus influenzae type b (Hib) | 1 | from 2 months until 3 months (default window) | `routine` | “ActHIB, Hiberix, Pentacel, or Vaxelis: 4-dose series (3-dose primary series at age 2, 4, and 6 months, followed by a booster dose at age 12–15 months)”; “PedvaxHIB: 3-dose series (2-dose primary series at age 2 and 4 months, followed by a booster dose at age 12–15 months)”. |
| IPV | Polio | 1 | from 2 months until 3 months (default window) | `routine` | “4-dose series at ages 2, 4, 6–18 months, 4–6 years”. |
| PCV | Pneumococcal | 1 | from 2 months until 3 months (default window) | `routine` | PCV15 or PCV20: “4-dose series at 2, 4, 6, 12–15 months”. |
| Rotavirus | Rotavirus | 1 | from 2 months until 3 months (default window) | `routine` | “Rotarix: 2-dose series at age 2 and 4 months”; “RotaTeq: 3-dose series at age 2, 4, and 6 months”. “Maximum age for first dose is 14 weeks, 6 days.” |

## 4 months

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| DTaP | Diphtheria | 2 | from 4 months until 5 months (default window) | `routine` | Primary series dose 2 at 4 months. |
| DTaP | Pertussis (whooping cough) | 2 | from 4 months until 5 months (default window) | `routine` | Primary series dose 2 at 4 months. |
| DTaP | Tetanus | 2 | from 4 months until 5 months (default window) | `routine` | Primary series dose 2 at 4 months. |
| Hib | Haemophilus influenzae type b (Hib) | 2 | from 4 months until 5 months (default window) | `routine` | Primary series dose 2 at 4 months (every product). |
| IPV | Polio | 2 | from 4 months until 5 months (default window) | `routine` | “4-dose series at ages 2, 4, 6–18 months, 4–6 years”. |
| PCV | Pneumococcal | 2 | from 4 months until 5 months (default window) | `routine` | “4-dose series at 2, 4, 6, 12–15 months”. |
| Rotavirus | Rotavirus | 2 | from 4 months until 5 months (default window) | `routine` | Dose 2 at 4 months (Rotarix and RotaTeq). |

## 6 months

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| COVID-19 | COVID-19 | 1 | from 6 months until 18 years | `supplementary` | “Shared clinical decision-making: Ages 6 months–17 years who are NOT moderately or severely immunocompromised.” The number of doses depends on age, product and earlier doses (for example, unvaccinated at 6 months–4 years: “2 doses 2024–25 Moderna at 0, 4–8 weeks” or “3 doses 2024–25 Pfizer-BioNTech”). The routine dose at age 18 is not listed. |
| DTaP | Diphtheria | 3 | from 6 months until 7 months (default window) | `routine` | Primary series dose 3 at 6 months. |
| DTaP | Pertussis (whooping cough) | 3 | from 6 months until 7 months (default window) | `routine` | Primary series dose 3 at 6 months. |
| DTaP | Tetanus | 3 | from 6 months until 7 months (default window) | `routine` | Primary series dose 3 at 6 months. |
| Hepatitis B | Hepatitis B | 3 | from 6 months until 19 months | `routine` | “Final (3rd or 4th) dose: age 6–18 months (minimum age 24 weeks)”. “Administration of 4 doses is permitted when a combination vaccine containing HepB is used after the birth dose”; a 4th dose is listed under Other. |
| Hib | Haemophilus influenzae type b (Hib) | 3 | from 6 months until 16 months | `routine` | (verify) window is the 6-month primary dose and the PedvaxHIB booster “at age 12–15 months” together, because doses are matched by count: ActHIB, Hiberix, Pentacel or Vaxelis give this dose at 6 months; with PedvaxHIB (“3-dose series (2-dose primary series at age 2 and 4 months, followed by a booster dose at age 12–15 months)”) the booster fills it. Not flagged overdue at 7 months; overdue from 16 months for a child with fewer than 3 doses. |
| IPV | Polio | 3 | from 6 months until 19 months | `routine` | “4-dose series at ages 2, 4, 6–18 months, 4–6 years”. |
| Influenza | Influenza | 1 | from 6 months, every season | `recurring` | “Use any influenza vaccine appropriate for age and health status annually”. “Age 6 months–8 years who have received fewer than 2 influenza vaccine doses before July 1, 2024, or whose influenza vaccination history is unknown: 2 doses, separated by at least 4 weeks”; otherwise 1 dose; “Age 9 years or older: 1 dose”. |
| PCV | Pneumococcal | 3 | from 6 months until 7 months (default window) | `routine` | “4-dose series at 2, 4, 6, 12–15 months”. |
| Rotavirus | Rotavirus | 3 | from 6 months until 7 months (default window) | `may_not_be_needed` | Only for RotaTeq: “3-dose series at age 2, 4, and 6 months”; Rotarix is a 2-dose series. “If any dose in the series is either RotaTeq or unknown, default to 3-dose series.” “Maximum age for final dose is 8 months, 0 days.” |

## 8 months

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| RSV | Respiratory syncytial virus (RSV) | 2 | from 8 months until 20 months | `supplementary` | High-risk groups only: “1 dose (8 through 19 months), See Notes”: chronic lung disease of prematurity, severe immunocompromise, cystic fibrosis with severity criteria, or American Indian or Alaska Native: “1 dose nirsevimab shortly before start of second RSV season”. |

## 12 months

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| Hepatitis A | Hepatitis A | 1 | from 12 months until 24 months | `routine` | “2-dose series (minimum interval: 6 months) at age 12–23 months”. A dose at 6–11 months before international travel does not count. |
| Hib | Haemophilus influenzae type b (Hib) | 4 | from 12 months until 16 months | `may_not_be_needed` | (verify) “3rd or 4th dose (See Notes)” at “12–15 months”: a 4th dose only after ActHIB, Hiberix, Pentacel or Vaxelis primary doses. A child on a 3-dose product fills lines 1–3 and is never flagged overdue here. “Vaxelis is not recommended for use as a booster dose.” |
| MMR | Measles | 1 | from 12 months until 16 months | `routine` | “2-dose series at age 12–15 months, age 4–6 years”. A dose at 6–11 months before international travel does not count. |
| MMR | Mumps | 1 | from 12 months until 16 months | `routine` | “2-dose series at age 12–15 months, age 4–6 years”. A dose at 6–11 months before international travel does not count. |
| MMR | Rubella | 1 | from 12 months until 16 months | `routine` | “2-dose series at age 12–15 months, age 4–6 years”. A dose at 6–11 months before international travel does not count. |
| PCV | Pneumococcal | 4 | from 12 months until 16 months | `routine` | “4-dose series at 2, 4, 6, 12–15 months”. |
| Varicella | Varicella (chickenpox) | 1 | from 12 months until 16 months | `routine` | “2-dose series at age 12–15 months, 4–6 years”. |

## 15 months

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| DTaP | Diphtheria | 4 | from 15 months until 19 months | `routine` | Booster “at ages 15–18 months”. “Dose 4 may be administered as early as age 12 months if at least 6 months have elapsed since dose 3.” |
| DTaP | Pertussis (whooping cough) | 4 | from 15 months until 19 months | `routine` | Booster “at ages 15–18 months”. “Dose 4 may be administered as early as age 12 months if at least 6 months have elapsed since dose 3.” |
| DTaP | Tetanus | 4 | from 15 months until 19 months | `routine` | Booster “at ages 15–18 months”. “Dose 4 may be administered as early as age 12 months if at least 6 months have elapsed since dose 3.” |

## 18 months

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| Hepatitis A | Hepatitis A | 2 | from 18 months until 24 months | `routine` | (verify) start of window derived: 12 months plus the 6-month minimum interval. CDC gives one window for the series: “2-dose series (minimum interval: 6 months) at age 12–23 months”. |

## 4 years

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| DTaP | Diphtheria | 5 | from 4 years until 7 years | `routine` | Booster at “4–6 years”. “Dose 5 is not necessary if dose 4 was administered at age 4 years or older and at least 6 months after dose 3”; the next dose then fills this line. |
| DTaP | Pertussis (whooping cough) | 5 | from 4 years until 7 years | `routine` | Booster at “4–6 years”. “Dose 5 is not necessary if dose 4 was administered at age 4 years or older and at least 6 months after dose 3”; the next dose then fills this line. |
| DTaP | Tetanus | 5 | from 4 years until 7 years | `routine` | Booster at “4–6 years”. “Dose 5 is not necessary if dose 4 was administered at age 4 years or older and at least 6 months after dose 3”; the next dose then fills this line. |
| IPV | Polio | 4 | from 4 years until 7 years | `routine` | “administer the final dose on or after age 4 years and at least 6 months after the previous dose”. “4 or more doses of IPV can be administered before age 4 years when a combination vaccine containing IPV is used. However, a dose is still recommended on or after age 4 years”; by count an early 4th dose fills this line. |
| MMR | Measles | 2 | from 4 years until 7 years | `routine` | “2-dose series at age 12–15 months, age 4–6 years”. |
| MMR | Mumps | 2 | from 4 years until 7 years | `routine` | “2-dose series at age 12–15 months, age 4–6 years”. |
| MMR | Rubella | 2 | from 4 years until 7 years | `routine` | “2-dose series at age 12–15 months, age 4–6 years”. |
| Varicella | Varicella (chickenpox) | 2 | from 4 years until 7 years | `routine` | “2-dose series at age 12–15 months, 4–6 years”. “Dose 2 may be administered as early as 3 months after dose 1”. |

## 11 years

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| HPV | Human papillomavirus (HPV) | 1 | from 11 years until 13 years | `routine` | “HPV vaccination routinely recommended at age 11–12 years (can start at age 9 years)”. “Age 9–14 years at initial vaccination: 2-dose series at 0, 6–12 months”; “Age 15 years or older at initial vaccination: 3-dose series” (the 3rd dose is not listed). |
| MenACWY | Meningococcal ACWY | 1 | from 11 years until 13 years | `routine` | “2-dose series at age 11–12 years; 16 years”. Earlier doses for high-risk groups are not listed. |
| Tdap | Diphtheria | 6 | from 11 years until 13 years | `routine` | “Age 11–12 years: 1 dose Tdap (adolescent booster)”, counted as the next diphtheria, tetanus and pertussis dose. “Age 7–9 years who receive Tdap should receive the adolescent Tdap booster dose at age 11–12 years.” |
| Tdap | Pertussis (whooping cough) | 6 | from 11 years until 13 years | `routine` | “Age 11–12 years: 1 dose Tdap (adolescent booster)”, counted as the next diphtheria, tetanus and pertussis dose. “Age 7–9 years who receive Tdap should receive the adolescent Tdap booster dose at age 11–12 years.” |
| Tdap | Tetanus | 6 | from 11 years until 13 years | `routine` | “Age 11–12 years: 1 dose Tdap (adolescent booster)”, counted as the next diphtheria, tetanus and pertussis dose. “Age 7–9 years who receive Tdap should receive the adolescent Tdap booster dose at age 11–12 years.” |

## 138 months

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| HPV | Human papillomavirus (HPV) | 2 | from 138 months until 13 years | `routine` | (verify) start of window derived: dose 1 at 11 years plus 6 months. CDC states dose 2 as an interval, not an age: “Age 9–14 years at initial vaccination: 2-dose series at 0, 6–12 months (minimum interval: 5 months; repeat dose if administered too soon)”. |

## 16 years

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| MenACWY | Meningococcal ACWY | 2 | from 16 years until 17 years | `routine` | “2-dose series at age 11–12 years; 16 years”; Table 1 shows the 2nd dose in the “16 yrs” column, read here as age 16 up to 17. Catch-up “Age 13–15 years: 1 dose now and booster at age 16–18 years”. |
| MenB | Meningococcal B | 1 | from 16 years until 24 years | `supplementary` | “Shared clinical decision-making”: “Adolescents not at increased risk age 16–23 years (preferred age 16–18 years)”; “Bexsero or Trumenba (use same brand for all doses): 2-dose series at least 6 months apart”. High-risk 3-dose series from age 10 not listed. |
| MenB | Meningococcal B | 2 | from 16 years until 24 years | `supplementary` | “2-dose series at least 6 months apart (if dose 2 is administered earlier than 6 months, administer dose 3 at least 4 months after dose 2)”. |
