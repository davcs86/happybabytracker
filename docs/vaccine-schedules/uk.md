# Vaccination schedule: United Kingdom (England) (`uk`)

Generated from `api/internal/vaccines/data/uk.csv`; do not edit. After changing the CSV, run
`UPDATE_DOCS=1 go test ./internal/vaccines/` in `api/` and commit both files.

- **Authority:** UK Health Security Agency (UKHSA)
- **Edition:** Complete routine immunisation schedule from 1 September 2026 (updated 24 September 2026), England
- **Retrieved:** 2026-11-22
- **Source:** https://www.gov.uk/government/publications/the-complete-routine-immunisation-schedule
- **Licence:** Open Government Licence v3.0 (https://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/), © Crown copyright 2026. The licence does not cover logos, crests, trade marks or third-party rights, and its use implies no official status or endorsement; the information is licensed as is, without warranty.
- **Attribution:** Source: UKHSA, “Complete routine immunisation schedule from 1 September 2026” (updated 24 September 2026), https://www.gov.uk/government/publications/the-complete-routine-immunisation-schedule, with the birth-date cohorts from UKHSA's “MMRV vaccination: information for healthcare professionals” and “Childhood schedule changes 2025 and 2026: information for healthcare practitioners” (both updated 25 August 2026). Contains public sector information licensed under the Open Government Licence v3.0 (https://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/). This is a partial, simplified transcription and adaptation for England, with dose numbers derived by counting and without trade names: selective, risk-group, pregnancy and catch-up programmes and the items listed in the review document are not transcribed. It is not endorsed by UKHSA; see the UKHSA schedule for the full text.

## Kinds

- `routine`: due inside its window, overdue after it.
- `may_not_be_needed`: depends on the product or history; is due inside its window; afterwards it is listed as available and never overdue.
- `supplementary`: shared decision-making or risk groups; listed under "Also available", never due or overdue.
- `recurring`: seasonal; listed with the doses given, never due or overdue.

A window runs from the age in `from` up to, not including, the age in `until`; without `until` it is 1 month.

## Not transcribed from the source

- Hepatitis B for babies born to women with hepatitis B infection: monovalent doses at birth and 4 weeks (and at 12 months for those born on or before 30 June 2024) and the blood test for HBsAg between 12 and 18 months
- BCG within 28 days of birth for infants in areas with TB incidence of 40 per 100,000 or more or with a parent or grandparent born in a high-incidence country
- Influenza for children in a clinical risk group from 6 months, and the second dose 4 weeks after a first-ever flu vaccine under 9 years (one recurring line stands for every season)
- Vaccination in pregnancy: influenza, pertussis from 16 weeks gestation and RSV from 28 weeks gestation
- Additional vaccines for individuals with underlying medical conditions (asplenia, complement disorders, cochlear implants, chronic conditions, immunosuppression, homelessness from 16 years): MenACWY, MenB, PCV20, influenza, hepatitis A and B, RSV and shingles
- MMRV selective catch-up from 1 November 2026 to 31 March 2028 for children born 1 January 2020 to 31 August 2022 who have not had chickenpox or 2 varicella doses
- The MenC part of the Hib/MenC dose at 12 months for children born on or before 30 June 2024 (MenC-containing vaccine is no longer recommended for young children who missed it), and the catch-up of a missed Hib/MenC dose with the 6-in-1
- The 6-in-1 in place of the 4-in-1 at 3 years 4 months for a child with no Hib dose after the first birthday
- The infant timing before 1 July 2025 (second MenB at 16 weeks and first PCV at 12 weeks) for children who had their 12-week PCV before that date: doses are matched by count, so they fill the same lines
- Catch-up and uncertain or incomplete immunisation status (UKHSA algorithm and Green Book chapter 11), and MMR or MMRV doses given before the first birthday, which are discounted
- HPV and MenACWY up to the 25th birthday, and MenB offers to young people
- Adult programmes: pneumococcal, influenza and shingles from 65 years and RSV at 75 years
- The schedules of Scotland, Wales and Northern Ireland (this file is England only)

## 8 weeks

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| 6-in-1 | Diphtheria | 1 | from 8 weeks until 8 weeks and 1 month (default window) | `routine` | “Eight weeks old”: DTaP/IPV/Hib/HepB (6 in 1 vaccine). “The first set of primary immunisations can be given from six weeks of age if required in certain circumstances, such as travel to an endemic country” (Green Book chapter 11). |
| 6-in-1 | Hepatitis B | 1 | from 8 weeks until 8 weeks and 1 month (default window) | `routine` | “Eight weeks old”: DTaP/IPV/Hib/HepB (6 in 1 vaccine). Babies born to women with hepatitis B infection also get monovalent doses at birth and 4 weeks (selective, not transcribed). |
| 6-in-1 | Haemophilus influenzae type b (Hib) | 1 | from 8 weeks until 8 weeks and 1 month (default window) | `routine` | “Eight weeks old”: DTaP/IPV/Hib/HepB (6 in 1 vaccine). |
| 6-in-1 | Pertussis (whooping cough) | 1 | from 8 weeks until 8 weeks and 1 month (default window) | `routine` | “Eight weeks old”: DTaP/IPV/Hib/HepB (6 in 1 vaccine). |
| 6-in-1 | Polio | 1 | from 8 weeks until 8 weeks and 1 month (default window) | `routine` | “Eight weeks old”: DTaP/IPV/Hib/HepB (6 in 1 vaccine). |
| 6-in-1 | Tetanus | 1 | from 8 weeks until 8 weeks and 1 month (default window) | `routine` | “Eight weeks old”: DTaP/IPV/Hib/HepB (6 in 1 vaccine). |
| MenB | Meningococcal B | 1 | from 8 weeks until 8 weeks and 1 month (default window) | `routine` | “Eight weeks old”: MenB. |
| Rotavirus | Rotavirus | 1 | from 8 weeks until 8 weeks and 1 month (default window) | `routine` | “Eight weeks old”: rotavirus, by mouth. “Rotavirus vaccine should only be given after checking for SCID screening result.” An upper age limit exists (Green Book) and is not transcribed. |

## 12 weeks

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| 6-in-1 | Diphtheria | 2 | from 12 weeks until 12 weeks and 1 month (default window) | `routine` | “Twelve weeks old”: DTaP/IPV/Hib/HepB (6 in 1 vaccine). |
| 6-in-1 | Hepatitis B | 2 | from 12 weeks until 12 weeks and 1 month (default window) | `routine` | “Twelve weeks old”: DTaP/IPV/Hib/HepB (6 in 1 vaccine). |
| 6-in-1 | Haemophilus influenzae type b (Hib) | 2 | from 12 weeks until 12 weeks and 1 month (default window) | `routine` | “Twelve weeks old”: DTaP/IPV/Hib/HepB (6 in 1 vaccine). |
| 6-in-1 | Pertussis (whooping cough) | 2 | from 12 weeks until 12 weeks and 1 month (default window) | `routine` | “Twelve weeks old”: DTaP/IPV/Hib/HepB (6 in 1 vaccine). |
| 6-in-1 | Polio | 2 | from 12 weeks until 12 weeks and 1 month (default window) | `routine` | “Twelve weeks old”: DTaP/IPV/Hib/HepB (6 in 1 vaccine). |
| 6-in-1 | Tetanus | 2 | from 12 weeks until 12 weeks and 1 month (default window) | `routine` | “Twelve weeks old”: DTaP/IPV/Hib/HepB (6 in 1 vaccine). |
| MenB | Meningococcal B | 2 | from 12 weeks until 12 weeks and 1 month (default window) | `routine` | “Twelve weeks old”: MenB (moved from 16 weeks on 1 July 2025; children who had their 12-week PCV before then stayed on 8 and 16 weeks). |
| Rotavirus | Rotavirus | 2 | from 12 weeks until 12 weeks and 1 month (default window) | `routine` | “Twelve weeks old”: rotavirus, after checking the SCID screening result. |

## 16 weeks

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| 6-in-1 | Diphtheria | 3 | from 16 weeks until 16 weeks and 1 month (default window) | `routine` | “Sixteen weeks old”: DTaP/IPV/Hib/HepB (6 in 1 vaccine). |
| 6-in-1 | Hepatitis B | 3 | from 16 weeks until 16 weeks and 1 month (default window) | `routine` | “Sixteen weeks old”: DTaP/IPV/Hib/HepB (6 in 1 vaccine). |
| 6-in-1 | Haemophilus influenzae type b (Hib) | 3 | from 16 weeks until 16 weeks and 1 month (default window) | `routine` | “Sixteen weeks old”: DTaP/IPV/Hib/HepB (6 in 1 vaccine). |
| 6-in-1 | Pertussis (whooping cough) | 3 | from 16 weeks until 16 weeks and 1 month (default window) | `routine` | “Sixteen weeks old”: DTaP/IPV/Hib/HepB (6 in 1 vaccine). |
| 6-in-1 | Polio | 3 | from 16 weeks until 16 weeks and 1 month (default window) | `routine` | “Sixteen weeks old”: DTaP/IPV/Hib/HepB (6 in 1 vaccine). |
| 6-in-1 | Tetanus | 3 | from 16 weeks until 16 weeks and 1 month (default window) | `routine` | “Sixteen weeks old”: DTaP/IPV/Hib/HepB (6 in 1 vaccine). |
| PCV | Pneumococcal | 1 | from 16 weeks until 16 weeks and 1 month (default window) | `routine` | “Sixteen weeks old”: “Pneumococcal (13 serotypes)”, PCV (moved from 12 weeks on 1 July 2025; a dose given at 12 weeks “will still count as a valid dose”). The schedule's update log says PCV20 from 1 January 2026, but the routine rows still name the 13-serotype vaccine. |

## 12 months

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| Hib/MenC (previous schedule) | Haemophilus influenzae type b (Hib) | 4 | from 12 months until 13 months (default window); born any date to 2024-06-30 | `routine` | Born on or before 30 June 2024: “should have been offered Hib/MenC … at 12 months of age as per the previous schedule” (childhood schedule changes information). That vaccine is no longer available: “any child under 10 years of age who was eligible for Hib/MenC and did not receive it, should be caught up using the hexavalent DTaP/IPV/Hib/HepB vaccine”. Only the Hib part is transcribed. |
| MMR | Measles | 1 | from 12 months until 13 months (default window); born any date to 2024-12-31 | `routine` | Born on or before 31 December 2024: “12 months: MMR” (MMRV information, Table 1). The complete schedule's one-year row says MMRV with no cohort split; children born before 1 January 2025 had MMR at one year. |
| MMR | Mumps | 1 | from 12 months until 13 months (default window); born any date to 2024-12-31 | `routine` | Born on or before 31 December 2024: “12 months: MMR” (MMRV information, Table 1). |
| MMR | Rubella | 1 | from 12 months until 13 months (default window); born any date to 2024-12-31 | `routine` | Born on or before 31 December 2024: “12 months: MMR” (MMRV information, Table 1). |
| MMRV | Measles | 1 | from 12 months until 13 months (default window); born 2025-01-01 to any date | `routine` | “One year old (on or after the child’s first birthday”: MMRV. Born on or after 1 January 2025: “2 doses of MMRV at 12 and 18 months” (MMRV information, Table 1). A dose before the first birthday is discounted. |
| MMRV | Mumps | 1 | from 12 months until 13 months (default window); born 2025-01-01 to any date | `routine` | MMRV at one year for children born on or after 1 January 2025 (MMRV information, Table 1). |
| MMRV | Rubella | 1 | from 12 months until 13 months (default window); born 2025-01-01 to any date | `routine` | MMRV at one year for children born on or after 1 January 2025 (MMRV information, Table 1). |
| MMRV | Varicella (chickenpox) | 1 | from 12 months until 13 months (default window); born 2025-01-01 to any date | `routine` | MMRV at one year for children born on or after 1 January 2025 (MMRV information, Table 1). |
| MenB | Meningococcal B | 3 | from 12 months until 13 months (default window) | `routine` | “One year old (on or after the child’s first birthday”: MenB booster. |
| PCV | Pneumococcal | 2 | from 12 months until 13 months (default window) | `routine` | “One year old (on or after the child’s first birthday”: PCV booster. |

## 18 months

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| 6-in-1 | Diphtheria | 4 | from 18 months until 19 months (default window); born 2024-07-01 to any date | `routine` | “Eighteen months old”, “Born on or after 1 July 2024”: DTaP/IPV/Hib/HepB. Born on or before 30 June 2024: “No appointment”. |
| 6-in-1 | Hepatitis B | 4 | from 18 months until 19 months (default window); born 2024-07-01 to any date | `routine` | “Eighteen months old”, born on or after 1 July 2024: DTaP/IPV/Hib/HepB. |
| 6-in-1 | Haemophilus influenzae type b (Hib) | 4 | from 18 months until 19 months (default window); born 2024-07-01 to any date | `routine` | “Eighteen months old”, born on or after 1 July 2024: DTaP/IPV/Hib/HepB, replacing the Hib part of the Hib/MenC dose that children born on or before 30 June 2024 had at 12 months. |
| 6-in-1 | Pertussis (whooping cough) | 4 | from 18 months until 19 months (default window); born 2024-07-01 to any date | `routine` | “Eighteen months old”, born on or after 1 July 2024: DTaP/IPV/Hib/HepB. |
| 6-in-1 | Polio | 4 | from 18 months until 19 months (default window); born 2024-07-01 to any date | `routine` | “Eighteen months old”, born on or after 1 July 2024: DTaP/IPV/Hib/HepB. |
| 6-in-1 | Tetanus | 4 | from 18 months until 19 months (default window); born 2024-07-01 to any date | `routine` | “Eighteen months old”, born on or after 1 July 2024: DTaP/IPV/Hib/HepB. |
| MMRV | Measles | 2 | from 18 months until 19 months (default window); born 2024-07-01 to any date | `routine` | “Eighteen months old”, born on or after 1 July 2024: MMRV. For children born 1 July to 31 December 2024 this is the second of three MMR-containing doses (MMR at 12 months, MMRV at 18 months and at 3 years 4 months). |
| MMRV | Mumps | 2 | from 18 months until 19 months (default window); born 2024-07-01 to any date | `routine` | “Eighteen months old”, born on or after 1 July 2024: MMRV. |
| MMRV | Rubella | 2 | from 18 months until 19 months (default window); born 2024-07-01 to any date | `routine` | “Eighteen months old”, born on or after 1 July 2024: MMRV. |
| MMRV | Varicella (chickenpox) | 1 | from 18 months until 19 months (default window); born 2024-07-01 to 2024-12-31 | `routine` | Born 1 July 2024 to 31 December 2024: “2 doses of MMRV at 18 months and 3 years 4 months” (MMRV information, Table 1); this is their first varicella dose. |
| MMRV | Varicella (chickenpox) | 2 | from 18 months until 19 months (default window); born 2025-01-01 to any date | `routine` | “Eighteen months old”: MMRV, the second varicella dose for children born on or after 1 January 2025. “Where 2 doses are indicated, these should be given a minimum of 4 weeks apart.” |

## 2 years

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| Flu | Influenza | 1 | from 2 years, every season | `recurring` | (verify) Eligibility is set each season by the annual flu letter: “Eligible paediatric age group, See annual flu letter”, “each year from September”, nasal (LAIV), or inactivated if LAIV is unsuitable. The 2026 to 2027 letter covers “all children aged 2 or 3 years on 31 August 2026” and school children “from reception to year 11”; a child who turns 2 during the season is not eligible unless in a clinical risk group. |

## 40 months

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| 4-in-1 pre-school booster | Diphtheria | 4 | from 40 months until 41 months (default window); born any date to 2024-06-30 | `routine` | “Three years four months old or soon after”: dTaP/IPV. Dose number derived: the fourth for children born on or before 30 June 2024, who had no 18-month appointment. |
| 4-in-1 pre-school booster | Diphtheria | 5 | from 40 months until 41 months (default window); born 2024-07-01 to any date | `routine` | “Three years four months old or soon after”: dTaP/IPV. Dose number derived: the fifth for children born on or after 1 July 2024, who had the 18-month 6-in-1. |
| 4-in-1 pre-school booster | Pertussis (whooping cough) | 4 | from 40 months until 41 months (default window); born any date to 2024-06-30 | `routine` | “Three years four months old or soon after”: dTaP/IPV; the fourth dose for children born on or before 30 June 2024 (derived). |
| 4-in-1 pre-school booster | Pertussis (whooping cough) | 5 | from 40 months until 41 months (default window); born 2024-07-01 to any date | `routine` | “Three years four months old or soon after”: dTaP/IPV; the fifth dose for children born on or after 1 July 2024 (derived). |
| 4-in-1 pre-school booster | Polio | 4 | from 40 months until 41 months (default window); born any date to 2024-06-30 | `routine` | “Three years four months old or soon after”: dTaP/IPV; the fourth dose for children born on or before 30 June 2024 (derived). |
| 4-in-1 pre-school booster | Polio | 5 | from 40 months until 41 months (default window); born 2024-07-01 to any date | `routine` | “Three years four months old or soon after”: dTaP/IPV; the fifth dose for children born on or after 1 July 2024 (derived). |
| 4-in-1 pre-school booster | Tetanus | 4 | from 40 months until 41 months (default window); born any date to 2024-06-30 | `routine` | “Three years four months old or soon after”: dTaP/IPV; the fourth dose for children born on or before 30 June 2024 (derived). |
| 4-in-1 pre-school booster | Tetanus | 5 | from 40 months until 41 months (default window); born 2024-07-01 to any date | `routine` | “Three years four months old or soon after”: dTaP/IPV; the fifth dose for children born on or after 1 July 2024 (derived). |
| MMR | Measles | 2 | from 40 months until 41 months (default window); born any date to 2022-08-31 | `routine` | Born on or before 31 August 2022: “12 months: MMR dose 1, 3 years 4 months: MMR dose 2” (MMRV information, Table 1). Those born from 1 January 2020 are offered one MMRV dose in a selective catch-up (not transcribed). |
| MMR | Mumps | 2 | from 40 months until 41 months (default window); born any date to 2022-08-31 | `routine` | Born on or before 31 August 2022: MMR dose 2 at 3 years 4 months (MMRV information, Table 1). |
| MMR | Rubella | 2 | from 40 months until 41 months (default window); born any date to 2022-08-31 | `routine` | Born on or before 31 August 2022: MMR dose 2 at 3 years 4 months (MMRV information, Table 1). |
| MMRV | Measles | 2 | from 40 months until 41 months (default window); born 2022-09-01 to 2024-06-30 | `routine` | Born 1 September 2022 to 30 June 2024: “12 months: MMR, 3 years 4 months: MMRV” (MMRV information, Table 1). |
| MMRV | Measles | 3 | from 40 months until 41 months (default window); born 2024-07-01 to 2024-12-31 | `routine` | Born 1 July 2024 to 31 December 2024: “12 months: MMR, 18 months: MMRV, 3 years 4 months: MMRV” (MMRV information, Table 1). “There are no safety concerns with giving 3 MMR-containing vaccines.” (Table 1 Note 1: not required if MMRV was given in place of the 12-month dose; dismiss it.) |
| MMRV | Mumps | 2 | from 40 months until 41 months (default window); born 2022-09-01 to 2024-06-30 | `routine` | Born 1 September 2022 to 30 June 2024: MMRV at 3 years 4 months (MMRV information, Table 1). |
| MMRV | Mumps | 3 | from 40 months until 41 months (default window); born 2024-07-01 to 2024-12-31 | `routine` | Born 1 July 2024 to 31 December 2024: MMRV at 3 years 4 months, their third MMR-containing dose (MMRV information, Table 1). (Table 1 Note 1: not required if MMRV was given in place of the 12-month dose; dismiss it.) |
| MMRV | Rubella | 2 | from 40 months until 41 months (default window); born 2022-09-01 to 2024-06-30 | `routine` | Born 1 September 2022 to 30 June 2024: MMRV at 3 years 4 months (MMRV information, Table 1). |
| MMRV | Rubella | 3 | from 40 months until 41 months (default window); born 2024-07-01 to 2024-12-31 | `routine` | Born 1 July 2024 to 31 December 2024: MMRV at 3 years 4 months, their third MMR-containing dose (MMRV information, Table 1). (Table 1 Note 1: not required if MMRV was given in place of the 12-month dose; dismiss it.) |
| MMRV | Varicella (chickenpox) | 1 | from 40 months until 41 months (default window); born 2022-09-01 to 2024-06-30 | `routine` | Born 1 September 2022 to 30 June 2024: “1 dose of MMRV at 3 years 4 months” (MMRV information, Table 1); their only routine varicella dose. |
| MMRV | Varicella (chickenpox) | 2 | from 40 months until 41 months (default window); born 2024-07-01 to 2024-12-31 | `routine` | Born 1 July 2024 to 31 December 2024: MMRV at 3 years 4 months, their second varicella dose (MMRV information, Table 1). (Table 1 Note 1: not required if MMRV was given in place of the 12-month dose; dismiss it.) |

## 12 years

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| HPV | Human papillomavirus (HPV) | 1 | from 12 years until 14 years | `routine` | “Boys and girls aged twelve to thirteen years”: one dose (“a single dose of HPV vaccine” before leaving school, Green Book chapter 11). The window runs through age 13. |

## 14 years

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| 3-in-1 teenage booster | Diphtheria | 5 | from 14 years until 14 years and 1 month (default window); born any date to 2024-06-30 | `routine` | “Fourteen years old (school year 9)”: Td/IPV, low-dose diphtheria; check MMR status. Dose number derived: the fifth for children born on or before 30 June 2024 (“at least five doses” before leaving school, Green Book chapter 11). Given in school year 9, so some children have it at 13. |
| 3-in-1 teenage booster | Diphtheria | 6 | from 14 years until 14 years and 1 month (default window); born 2024-07-01 to any date | `routine` | “Fourteen years old (school year 9)”: Td/IPV, low-dose diphtheria; check MMR status. Dose number derived: the sixth for children born on or after 1 July 2024. Given in school year 9, so some children have it at 13. |
| 3-in-1 teenage booster | Polio | 5 | from 14 years until 14 years and 1 month (default window); born any date to 2024-06-30 | `routine` | “Fourteen years old (school year 9)”: Td/IPV; the fifth dose for children born on or before 30 June 2024 (derived). |
| 3-in-1 teenage booster | Polio | 6 | from 14 years until 14 years and 1 month (default window); born 2024-07-01 to any date | `routine` | “Fourteen years old (school year 9)”: Td/IPV; the sixth dose for children born on or after 1 July 2024 (derived). |
| 3-in-1 teenage booster | Tetanus | 5 | from 14 years until 14 years and 1 month (default window); born any date to 2024-06-30 | `routine` | “Fourteen years old (school year 9)”: Td/IPV; the fifth dose for children born on or before 30 June 2024 (derived). |
| 3-in-1 teenage booster | Tetanus | 6 | from 14 years until 14 years and 1 month (default window); born 2024-07-01 to any date | `routine` | “Fourteen years old (school year 9)”: Td/IPV; the sixth dose for children born on or after 1 July 2024 (derived). |
| MenACWY | Meningococcal ACWY | 1 | from 14 years until 14 years and 1 month (default window) | `routine` | “Fourteen years old (school year 9)”: MenACWY, “a single dose of MenACWY vaccine after the age of 10 years” (Green Book chapter 11). Given in school year 9, so some children have it at 13. |
