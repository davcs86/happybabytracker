# Vaccination schedule: United States (`us`)

Generated from `api/internal/vaccines/data/us.csv`; do not edit. After changing the CSV, run
`UPDATE_DOCS=1 go test ./internal/vaccines/` in `api/` and commit both files.

- **Authority:** Centers for Disease Control and Prevention (CDC)
- **Edition:** CDC Child and Adolescent Immunization Schedule by Age, 2025 (addendum updated July 2, 2025)
- **Retrieved:** 2026-11-22
- **Source:** https://www.cdc.gov/vaccines/hcp/imz-schedules/child-adolescent-age.html
- **Licence:** Public domain (a work of the United States government), reused on CDC's conditions at https://www.cdc.gov/other/agencymaterials.html: credit the agency, state that its use implies no endorsement, do not change the substantive content, and state that the material is available free of charge on the agency website.
- **Attribution:** Source: CDC, Recommended Child and Adolescent Immunization Schedule for Ages 18 Years or Younger, United States, 2025 (revised 07/02/2025). CDC does not endorse this application or its use of the material. The material is available free of charge on the CDC website at https://www.cdc.gov/vaccines/hcp/imz-schedules/child-adolescent-age.html. This transcription is simplified for display (one line per antigen and dose; risk-group and catch-up rules are left out) and must not be read as CDC's own text; every age and series is kept as CDC states it. Trade names are CDC's: “The use of trade names is for identification purposes only and does not imply endorsement by the ACIP or CDC.”

## Kinds

- `routine`: due inside its window, overdue after it.
- `may_not_be_needed`: depends on the product or history; due inside its window, never overdue.
- `supplementary`: shared decision-making or risk groups; listed under "Also available", never due or overdue.
- `recurring`: seasonal; listed with the doses given, never due or overdue.

A window runs from the age in `from` up to, not including, the age in `until`; without `until` it is one month.

## Birth

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| Hepatitis B | hepatitis_b | 1 | from Birth (default window) | `routine` | CDC: “Birth”. “Birth weight ≥2,000 grams: 1 dose within 24 hours of birth if medically stable”; “Birth weight <2,000 grams: 1 dose at chronological age 1 month or hospital discharge”. Monovalent HepB only. |
| RSV | rsv | 1 | from Birth until 8 months | `supplementary` | (verify) end of window derived: CDC prints no end age for this dose; 8 months is where the second-season bar “1 dose (8 through 19 months)” starts. Nirsevimab, a monoclonal antibody: “1 dose depending on maternal RSV vaccination status”. Born October–March: “administer 1 dose nirsevimab within 1 week of birth”; born April–September: “shortly before start of RSV season”; “Mother received RSV vaccine at least 14 days prior to delivery: nirsevimab not needed”. |

## 1 month

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| Hepatitis B | hepatitis_b | 2 | from 1 month until 3 months | `routine` | “3-dose series at age 0, 1–2, 6–18 months”. |

## 2 months

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| DTaP | diphtheria | 1 | from 2 months (default window) | `routine` | “5-dose series (3-dose primary series at age 2, 4, and 6 months, followed by booster doses at ages 15–18 months and 4–6 years)”. |
| DTaP | pertussis | 1 | from 2 months (default window) | `routine` | “5-dose series (3-dose primary series at age 2, 4, and 6 months, followed by booster doses at ages 15–18 months and 4–6 years)”. |
| DTaP | tetanus | 1 | from 2 months (default window) | `routine` | “5-dose series (3-dose primary series at age 2, 4, and 6 months, followed by booster doses at ages 15–18 months and 4–6 years)”. |
| Hib | hib | 1 | from 2 months (default window) | `routine` | “ActHIB, Hiberix, Pentacel, or Vaxelis: 4-dose series (3-dose primary series at age 2, 4, and 6 months, followed by a booster dose at age 12–15 months)”; “PedvaxHIB: 3-dose series (2-dose primary series at age 2 and 4 months, followed by a booster dose at age 12–15 months)”. |
| IPV | polio | 1 | from 2 months (default window) | `routine` | “4-dose series at ages 2, 4, 6–18 months, 4–6 years”. |
| PCV | pneumococcus | 1 | from 2 months (default window) | `routine` | PCV15 or PCV20: “4-dose series at 2, 4, 6, 12–15 months”. |
| Rotavirus | rotavirus | 1 | from 2 months (default window) | `routine` | “Rotarix: 2-dose series at age 2 and 4 months”; “RotaTeq: 3-dose series at age 2, 4, and 6 months”. “Maximum age for first dose is 14 weeks, 6 days.” |

## 4 months

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| DTaP | diphtheria | 2 | from 4 months (default window) | `routine` | Primary series dose 2 at 4 months. |
| DTaP | pertussis | 2 | from 4 months (default window) | `routine` | Primary series dose 2 at 4 months. |
| DTaP | tetanus | 2 | from 4 months (default window) | `routine` | Primary series dose 2 at 4 months. |
| Hib | hib | 2 | from 4 months (default window) | `routine` | Primary series dose 2 at 4 months (every product). |
| IPV | polio | 2 | from 4 months (default window) | `routine` | “4-dose series at ages 2, 4, 6–18 months, 4–6 years”. |
| PCV | pneumococcus | 2 | from 4 months (default window) | `routine` | “4-dose series at 2, 4, 6, 12–15 months”. |
| Rotavirus | rotavirus | 2 | from 4 months (default window) | `routine` | Dose 2 at 4 months (Rotarix and RotaTeq). |

## 6 months

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| COVID-19 | covid19 | 1 | from 6 months until 18 years | `supplementary` | “Shared clinical decision-making: Ages 6 months–17 years who are NOT moderately or severely immunocompromised.” The number of doses depends on age, product and earlier doses (for example, unvaccinated at 6 months–4 years: “2 doses 2024–25 Moderna at 0, 4–8 weeks” or “3 doses 2024–25 Pfizer-BioNTech”). The routine dose at age 18 is not listed. |
| DTaP | diphtheria | 3 | from 6 months (default window) | `routine` | Primary series dose 3 at 6 months. |
| DTaP | pertussis | 3 | from 6 months (default window) | `routine` | Primary series dose 3 at 6 months. |
| DTaP | tetanus | 3 | from 6 months (default window) | `routine` | Primary series dose 3 at 6 months. |
| Hepatitis B | hepatitis_b | 3 | from 6 months until 19 months | `routine` | “Final (3rd or 4th) dose: age 6–18 months (minimum age 24 weeks)”. “Administration of 4 doses is permitted when a combination vaccine containing HepB is used after the birth dose”; a 4th dose is listed under Other. |
| Hib | hib | 3 | from 6 months (default window) | `may_not_be_needed` | Table 1 shows “See Notes” at 6 months: given with ActHIB, Hiberix, Pentacel or Vaxelis; PedvaxHIB has no 6-month dose, so a PedvaxHIB booster fills this line by count. |
| IPV | polio | 3 | from 6 months until 19 months | `routine` | “4-dose series at ages 2, 4, 6–18 months, 4–6 years”. |
| Influenza | influenza | 1 | from 6 months (default window) | `recurring` | “Use any influenza vaccine appropriate for age and health status annually”. “Age 6 months–8 years who have received fewer than 2 influenza vaccine doses before July 1, 2024, or whose influenza vaccination history is unknown: 2 doses, separated by at least 4 weeks”; otherwise 1 dose; “Age 9 years or older: 1 dose”. |
| PCV | pneumococcus | 3 | from 6 months (default window) | `routine` | “4-dose series at 2, 4, 6, 12–15 months”. |
| Rotavirus | rotavirus | 3 | from 6 months (default window) | `may_not_be_needed` | Only for RotaTeq: “3-dose series at age 2, 4, and 6 months”; Rotarix is a 2-dose series. “If any dose in the series is either RotaTeq or unknown, default to 3-dose series.” “Maximum age for final dose is 8 months, 0 days.” |

## 8 months

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| RSV | rsv | 2 | from 8 months until 20 months | `supplementary` | High-risk groups only: “1 dose (8 through 19 months), See Notes”: chronic lung disease of prematurity, severe immunocompromise, cystic fibrosis with severity criteria, or American Indian or Alaska Native: “1 dose nirsevimab shortly before start of second RSV season”. |

## 12 months

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| Hepatitis A | hepatitis_a | 1 | from 12 months until 24 months | `routine` | “2-dose series (minimum interval: 6 months) at age 12–23 months”. A dose at 6–11 months before international travel does not count. |
| Hepatitis A | hepatitis_a | 2 | from 12 months until 24 months | `routine` | CDC gives one window for the series: “2-dose series (minimum interval: 6 months) at age 12–23 months”; dose 2 comes at least 6 months after dose 1. |
| Hib | hib | 4 | from 12 months until 16 months | `may_not_be_needed` | “3rd or 4th dose (See Notes)” at “12–15 months”, the booster for every product. A 4th dose is only needed after a 6-month dose (not with PedvaxHIB), so this line is never flagged overdue. “Vaxelis is not recommended for use as a booster dose.” |
| MMR | measles | 1 | from 12 months until 16 months | `routine` | “2-dose series at age 12–15 months, age 4–6 years”. A dose at 6–11 months before international travel does not count. |
| MMR | mumps | 1 | from 12 months until 16 months | `routine` | “2-dose series at age 12–15 months, age 4–6 years”. A dose at 6–11 months before international travel does not count. |
| MMR | rubella | 1 | from 12 months until 16 months | `routine` | “2-dose series at age 12–15 months, age 4–6 years”. A dose at 6–11 months before international travel does not count. |
| PCV | pneumococcus | 4 | from 12 months until 16 months | `routine` | “4-dose series at 2, 4, 6, 12–15 months”. |
| Varicella | varicella | 1 | from 12 months until 16 months | `routine` | “2-dose series at age 12–15 months, 4–6 years”. |

## 15 months

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| DTaP | diphtheria | 4 | from 15 months until 19 months | `routine` | Booster “at ages 15–18 months”. “Dose 4 may be administered as early as age 12 months if at least 6 months have elapsed since dose 3.” |
| DTaP | pertussis | 4 | from 15 months until 19 months | `routine` | Booster “at ages 15–18 months”. “Dose 4 may be administered as early as age 12 months if at least 6 months have elapsed since dose 3.” |
| DTaP | tetanus | 4 | from 15 months until 19 months | `routine` | Booster “at ages 15–18 months”. “Dose 4 may be administered as early as age 12 months if at least 6 months have elapsed since dose 3.” |

## 4 years

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| DTaP | diphtheria | 5 | from 4 years until 7 years | `routine` | Booster at “4–6 years”. “Dose 5 is not necessary if dose 4 was administered at age 4 years or older and at least 6 months after dose 3”; the next dose then fills this line. |
| DTaP | pertussis | 5 | from 4 years until 7 years | `routine` | Booster at “4–6 years”. “Dose 5 is not necessary if dose 4 was administered at age 4 years or older and at least 6 months after dose 3”; the next dose then fills this line. |
| DTaP | tetanus | 5 | from 4 years until 7 years | `routine` | Booster at “4–6 years”. “Dose 5 is not necessary if dose 4 was administered at age 4 years or older and at least 6 months after dose 3”; the next dose then fills this line. |
| IPV | polio | 4 | from 4 years until 7 years | `routine` | “administer the final dose on or after age 4 years and at least 6 months after the previous dose”. “4 or more doses of IPV can be administered before age 4 years when a combination vaccine containing IPV is used. However, a dose is still recommended on or after age 4 years”; by count an early 4th dose fills this line. |
| MMR | measles | 2 | from 4 years until 7 years | `routine` | “2-dose series at age 12–15 months, age 4–6 years”. |
| MMR | mumps | 2 | from 4 years until 7 years | `routine` | “2-dose series at age 12–15 months, age 4–6 years”. |
| MMR | rubella | 2 | from 4 years until 7 years | `routine` | “2-dose series at age 12–15 months, age 4–6 years”. |
| Varicella | varicella | 2 | from 4 years until 7 years | `routine` | “2-dose series at age 12–15 months, 4–6 years”. “Dose 2 may be administered as early as 3 months after dose 1”. |

## 11 years

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| HPV | hpv | 1 | from 11 years until 13 years | `routine` | “HPV vaccination routinely recommended at age 11–12 years (can start at age 9 years)”. “Age 9–14 years at initial vaccination: 2-dose series at 0, 6–12 months”; “Age 15 years or older at initial vaccination: 3-dose series” (the 3rd dose is not listed). |
| HPV | hpv | 2 | from 11 years until 13 years | `routine` | CDC states dose 2 as an interval, not an age: “2-dose series at 0, 6–12 months (minimum interval: 5 months)”; this line has the same window as dose 1. |
| MenACWY | meningococcal_acwy | 1 | from 11 years until 13 years | `routine` | “2-dose series at age 11–12 years; 16 years”. Earlier doses for high-risk groups are not listed. |
| Tdap | diphtheria | 6 | from 11 years until 13 years | `routine` | “Age 11–12 years: 1 dose Tdap (adolescent booster)”, counted as the next diphtheria, tetanus and pertussis dose. “Age 7–9 years who receive Tdap should receive the adolescent Tdap booster dose at age 11–12 years.” |
| Tdap | pertussis | 6 | from 11 years until 13 years | `routine` | “Age 11–12 years: 1 dose Tdap (adolescent booster)”, counted as the next diphtheria, tetanus and pertussis dose. “Age 7–9 years who receive Tdap should receive the adolescent Tdap booster dose at age 11–12 years.” |
| Tdap | tetanus | 6 | from 11 years until 13 years | `routine` | “Age 11–12 years: 1 dose Tdap (adolescent booster)”, counted as the next diphtheria, tetanus and pertussis dose. “Age 7–9 years who receive Tdap should receive the adolescent Tdap booster dose at age 11–12 years.” |

## 16 years

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| MenACWY | meningococcal_acwy | 2 | from 16 years until 17 years | `routine` | “2-dose series at age 11–12 years; 16 years”; Table 1 shows the 2nd dose in the “16 yrs” column, read here as age 16 up to 17. Catch-up “Age 13–15 years: 1 dose now and booster at age 16–18 years”. |
| MenB | meningococcal_b | 1 | from 16 years until 24 years | `supplementary` | “Shared clinical decision-making”: “Adolescents not at increased risk age 16–23 years (preferred age 16–18 years)”; “Bexsero or Trumenba (use same brand for all doses): 2-dose series at least 6 months apart”. High-risk 3-dose series from age 10 not listed. |
| MenB | meningococcal_b | 2 | from 16 years until 24 years | `supplementary` | “2-dose series at least 6 months apart (if dose 2 is administered earlier than 6 months, administer dose 3 at least 4 months after dose 2)”. |
