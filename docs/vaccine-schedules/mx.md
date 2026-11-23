# Vaccination schedule: Mexico (`mx`)

Generated from `api/internal/vaccines/data/mx.csv`; do not edit. After changing the CSV, run
`UPDATE_DOCS=1 go test ./internal/vaccines/` in `api/` and commit both files.

- **Authority:** Secretaría de Salud, Centro Nacional para la Salud de la Infancia y la Adolescencia (CeNSIA)
- **Edition:** Lineamientos Generales del Programa de Vacunación Universal 2026 (published 31 December 2025, PDF of 30 April 2026; measles amendment of 18 February 2026), Cuadros 1 and 6; cross-checked with the PRONAM “Vacunación a lo largo de la vida”, 2a ed. (Consejo de Salubridad General, 30 July 2026)
- **Retrieved:** 2026-11-22
- **Source:** https://www.gob.mx/salud/censia/documentos/lineamientos-generales-pvu-2026
- **Licence:** gob.mx publishes no open licence on these documents (no Creative Commons or Libre Uso MX notice was found on them). The gob.mx Términos y Condiciones (https://www.gob.mx/terminos) say that the portal's information is public unless indicated otherwise, and require anyone who reproduces or processes it to cite its electronic location and the date it was consulted; the same page also limits downloaded materials to personal, non-commercial use. Only facts are transcribed (antigen, dose number, age): the Ley Federal del Derecho de Autor, art. 14, excludes facts and calendars (fr. X) and administrative texts (fr. VIII) from copyright protection. No logo, emblem, page image or prose of the publication is reproduced beyond short quoted age wording. This is not legal advice.
- **Attribution:** Fuente: Secretaría de Salud, CeNSIA, “Lineamientos Generales del Programa de Vacunación Universal 2026” (31 de diciembre de 2025; enmienda del 18 de febrero de 2026), https://www.gob.mx/cms/uploads/attachment/file/1076033/LINEAMIENTOS_GENERALES_2026_rem.pdf (page: https://www.gob.mx/salud/censia/documentos/lineamientos-generales-pvu-2026), consultado el 22 de noviembre de 2026 (2026-11-22). This is a partial, simplified transcription of facts only (antigen, dose number and age), with dose numbers derived by counting and without brand names: selective, risk-group, pregnancy and catch-up lines, adolescent and adult lines other than HPV (Td included) and the items listed in the review document are not transcribed. It is not endorsed by the Secretaría de Salud and is not a copy of its publication; the official text prevails.

## Kinds

- `routine`: due inside its window, overdue after it.
- `may_not_be_needed`: depends on the product or history; is due inside its window; afterwards it is listed as available and never overdue.
- `supplementary`: shared decision-making or risk groups; listed under "Also available", never due or overdue.
- `recurring`: seasonal; listed with the doses given, never due or overdue.

A window runs from the age in `from` up to, not including, the age in `until`; without `until` it is 1 month.

## Not transcribed from the source

- Tdpa in pregnancy from week 20 and for close contacts of infants under 1 year
- RSV (VSR) vaccination in pregnancy at 32 to 36 weeks
- Influenza and COVID-19 from 5 years for children with comorbidities, from 10 to 59 years with risk factors, from 60 years and in pregnancy, and the second dose in the first season for children of 6 to 59 months (one recurring line per antigen stands for every season)
- Adolescent and adult lines, which the Lineamientos class as risk-based and not universal: Td from 15 years with a booster every 10 years (3 doses at 0, 1 and 12 months when incomplete or undocumented), SR or SRP from 10 to 49 years without a complete schedule, hepatitis B from 11 years without prior vaccination, pneumococcal conjugate from 60 years
- HPV for people of 11 to 49 years living with HIV (3 doses) and for 9 to 19 years in the sexual-violence protocol, and the PRONAM catch-up of girls of 12 to 16 years
- The RV5 rotavirus scheme of 3 doses at 2, 4 and 6 months (Cuadro 7, a third dose lists under Other)
- The measles dosis cero at 6 to 11 months in outbreaks (it does not count towards the series) and the measles catch-up of children of 1 to 12 years (amendment of 18 February 2026)
- SRP catch-up of 2 doses 4 weeks apart for children under 10 with no dose
- BCG catch-up up to 5 years (exceptionally 14)
- hepatitis B timing for preterm babies under 29 weeks or under 1,500 g (6 to 8 weeks of life)
- DPT from 3 years 6 months in daycare or emergencies
- The minimum and maximum permitted ages and the intervals of Anexo 1 (a low-resolution scan)
- Legacy schemes recorded in older cartillas (pentavalent DTP-HB-Hib, oral polio): doses are matched by count, so they fill the same lines
- Special situations in the Manual de Vacunación

## Lines to verify

These lines carry `(verify)`: check them against the source first.

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| COVID-19 | COVID-19 | 1 | from 6 months, every season | `recurring` | (verify) “A partir de los 6 meses”, in the winter season: a first season of 2 doses 4 weeks apart (6 and 7 months), then one dose each winter season up to 59 months; the source says the COVID-19 policy may change during 2026. |
| Influenza | Influenza | 1 | from 6 months, every season | `recurring` | (verify) “A partir de los 6 meses”, in the winter season: a first season of 2 doses 4 weeks apart (6 and 7 months), then one dose each winter season up to 59 months; from 5 years only with a comorbidity. |
| HPV | Human papillomavirus (HPV) | 1 | from 11 years | `supplementary` | (verify) Single dose for girls and boys in the 5th grade of primary school (“5º grado de primaria”, about 10 to 11 years) and at “11 años” for those not in school; delivered by school grade, not by age, so it is supplementary and never flagged by age; the age is derived. |

## Birth

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| BCG | Tuberculosis | 1 | from Birth until 12 months | `routine` | “Recién nacido”, one dose, for newborns of 2,000 g or more before leaving hospital or at the first contact with health services, “preferentemente antes del año de edad”. |
| Hepatitis B | Hepatitis B | 1 | from Birth until 7 days | `may_not_be_needed` | “Recién nacido”: in the first 24 hours of life, late up to 7 days of life. A baby without this dose starts hepatitis B with the hexavalent at 2 months and needs no extra dose; doses then fill one line earlier by count, so no hepatitis B line is ever flagged overdue (the other hexavalent antigens flag a missed visit). |

## 2 months

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| Hexavalent | Diphtheria | 1 | from 2 months until 3 months (default window) | `routine` | “2 meses”: hexavalente acelular (DPaT+VIP+HB+Hib), 1st dose. |
| Hexavalent | Hepatitis B | 2 | from 2 months until 3 months (default window) | `may_not_be_needed` | “2 meses”: hexavalente acelular, 1st dose; it is given whether or not the birth dose was given. |
| Hexavalent | Haemophilus influenzae type b (Hib) | 1 | from 2 months until 3 months (default window) | `routine` | “2 meses”: hexavalente acelular, 1st dose. |
| Hexavalent | Pertussis (whooping cough) | 1 | from 2 months until 3 months (default window) | `routine` | “2 meses”: hexavalente acelular (acellular pertussis), 1st dose. |
| Hexavalent | Polio | 1 | from 2 months until 3 months (default window) | `routine` | “2 meses”: hexavalente acelular (inactivated polio), 1st dose. |
| Hexavalent | Tetanus | 1 | from 2 months until 3 months (default window) | `routine` | “2 meses”: hexavalente acelular, 1st dose. |
| Pneumococcal | Pneumococcal | 1 | from 2 months until 3 months (default window) | `routine` | “2 meses”: anti neumocócica conjugada, 1st dose. |
| Rotavirus | Rotavirus | 1 | from 2 months until 3 months (default window) | `routine` | “2 meses”: anti-rotavirus (RV1), 1st dose, by mouth; no dose after 7 months 29 days of age. |

## 4 months

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| Hexavalent | Diphtheria | 2 | from 4 months until 5 months (default window) | `routine` | “4 meses”: hexavalente acelular, 2nd dose. |
| Hexavalent | Hepatitis B | 3 | from 4 months until 5 months (default window) | `may_not_be_needed` | “4 meses”: hexavalente acelular, 2nd dose. |
| Hexavalent | Haemophilus influenzae type b (Hib) | 2 | from 4 months until 5 months (default window) | `routine` | “4 meses”: hexavalente acelular, 2nd dose. |
| Hexavalent | Pertussis (whooping cough) | 2 | from 4 months until 5 months (default window) | `routine` | “4 meses”: hexavalente acelular, 2nd dose. |
| Hexavalent | Polio | 2 | from 4 months until 5 months (default window) | `routine` | “4 meses”: hexavalente acelular, 2nd dose. |
| Hexavalent | Tetanus | 2 | from 4 months until 5 months (default window) | `routine` | “4 meses”: hexavalente acelular, 2nd dose. |
| Pneumococcal | Pneumococcal | 2 | from 4 months until 5 months (default window) | `routine` | “4 meses”: anti neumocócica conjugada, 2nd dose. |
| Rotavirus | Rotavirus | 2 | from 4 months until 5 months (default window) | `routine` | “4 meses”: anti-rotavirus (RV1), 2nd dose; no dose after 7 months 29 days of age. In the 2026 text “pentavalente” means the RV5 rotavirus vaccine (3 doses), not a DTP combination. |

## 6 months

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| COVID-19 | COVID-19 | 1 | from 6 months, every season | `recurring` | (verify) “A partir de los 6 meses”, in the winter season: a first season of 2 doses 4 weeks apart (6 and 7 months), then one dose each winter season up to 59 months; the source says the COVID-19 policy may change during 2026. |
| Hexavalent | Diphtheria | 3 | from 6 months until 7 months (default window) | `routine` | “6 meses”: hexavalente acelular, 3rd dose. |
| Hexavalent | Hepatitis B | 4 | from 6 months until 7 months (default window) | `may_not_be_needed` | “6 meses”: hexavalente acelular, 3rd dose. |
| Hexavalent | Haemophilus influenzae type b (Hib) | 3 | from 6 months until 7 months (default window) | `routine` | “6 meses”: hexavalente acelular, 3rd dose. |
| Hexavalent | Pertussis (whooping cough) | 3 | from 6 months until 7 months (default window) | `routine` | “6 meses”: hexavalente acelular, 3rd dose. |
| Hexavalent | Polio | 3 | from 6 months until 7 months (default window) | `routine` | “6 meses”: hexavalente acelular, 3rd dose. |
| Hexavalent | Tetanus | 3 | from 6 months until 7 months (default window) | `routine` | “6 meses”: hexavalente acelular, 3rd dose. |
| Influenza | Influenza | 1 | from 6 months, every season | `recurring` | (verify) “A partir de los 6 meses”, in the winter season: a first season of 2 doses 4 weeks apart (6 and 7 months), then one dose each winter season up to 59 months; from 5 years only with a comorbidity. |

## 12 months

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| Pneumococcal | Pneumococcal | 3 | from 12 months until 13 months (default window) | `routine` | “Un año” / “12 meses”: anti neumocócica conjugada, 3rd dose. |
| SRP (MMR) | Measles | 1 | from 12 months until 13 months (default window) | `routine` | “12 meses”: triple viral (SRP), 1st dose. A dosis cero given at 6 to 11 months in an outbreak does not count. |
| SRP (MMR) | Mumps | 1 | from 12 months until 13 months (default window) | `routine` | “12 meses”: triple viral (SRP), 1st dose. |
| SRP (MMR) | Rubella | 1 | from 12 months until 13 months (default window) | `routine` | “12 meses”: triple viral (SRP), 1st dose. |
| Varicella | Varicella (chickenpox) | 1 | from 12 months | `supplementary` | “Un año”: one dose, listed in Cuadros 1 and 6 but not part of the basic schedule; given to children of 12 months who attend daycare. |

## 18 months

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| Hepatitis A | Hepatitis A | 1 | from 18 months until 19 months (default window) | `routine` | “18 meses”: one dose; new in the programme (CONAVA agreement 01-2025), whose footnote targets children of 1 year or without prior vaccination. |
| Hexavalent | Diphtheria | 4 | from 18 months until 19 months (default window) | `routine` | “18 meses”: hexavalente acelular, booster (dosis de refuerzo). |
| Hexavalent | Hepatitis B | 5 | from 18 months until 19 months (default window) | `may_not_be_needed` | “18 meses”: hexavalente acelular, booster. The 5th hepatitis B dose only for a child who had the birth dose; without it the booster is the 4th, and no extra hepatitis B dose is needed. |
| Hexavalent | Haemophilus influenzae type b (Hib) | 4 | from 18 months until 19 months (default window) | `routine` | “18 meses”: hexavalente acelular, booster. |
| Hexavalent | Pertussis (whooping cough) | 4 | from 18 months until 19 months (default window) | `routine` | “18 meses”: hexavalente acelular, booster. |
| Hexavalent | Polio | 4 | from 18 months until 19 months (default window) | `routine` | “18 meses”: hexavalente acelular, booster. The DPT at 4 years carries no polio. |
| Hexavalent | Tetanus | 4 | from 18 months until 19 months (default window) | `routine` | “18 meses”: hexavalente acelular, booster. |
| SRP (MMR) | Measles | 2 | from 18 months until 19 months (default window); born 2020-07-01 to any date | `routine` | “18 meses”: triple viral (SRP), 2nd dose, for children born from July 2020. |
| SRP (MMR) | Mumps | 2 | from 18 months until 19 months (default window); born 2020-07-01 to any date | `routine` | “18 meses”: triple viral (SRP), 2nd dose, for children born from July 2020. |
| SRP (MMR) | Rubella | 2 | from 18 months until 19 months (default window); born 2020-07-01 to any date | `routine` | “18 meses”: triple viral (SRP), 2nd dose, for children born from July 2020. |

## 4 years

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| DPT booster | Diphtheria | 5 | from 4 years until 7 years | `routine` | “4 años”: DPT booster (triple bacteriana), the 5th dose of the primary scheme; when not given at 4 years, “la edad de aplicación no debe sobrepasar los 6 años, 11 meses y 29 días”, so the window runs to 7 years. |
| DPT booster | Pertussis (whooping cough) | 5 | from 4 years until 7 years | `routine` | “4 años”: DPT booster, the 5th dose of the primary scheme (whole-cell pertussis); “la edad de aplicación no debe sobrepasar los 6 años, 11 meses y 29 días”. |
| DPT booster | Tetanus | 5 | from 4 years until 7 years | `routine` | “4 años”: DPT booster, the 5th dose of the primary scheme; “la edad de aplicación no debe sobrepasar los 6 años, 11 meses y 29 días”. |

## 6 years

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| SRP (MMR) | Measles | 2 | from 6 years until 6 years and 1 month (default window); born any date to 2020-06-30 | `routine` | “6 años”: triple viral (SRP), 2nd dose, for children born before July 2020; the source applies this age until 2026. |
| SRP (MMR) | Mumps | 2 | from 6 years until 6 years and 1 month (default window); born any date to 2020-06-30 | `routine` | “6 años”: triple viral (SRP), 2nd dose, for children born before July 2020. |
| SRP (MMR) | Rubella | 2 | from 6 years until 6 years and 1 month (default window); born any date to 2020-06-30 | `routine` | “6 años”: triple viral (SRP), 2nd dose, for children born before July 2020. |

## 11 years

| Vaccine | Antigen | Dose | Window | Kind | Note |
|---|---|---|---|---|---|
| HPV | Human papillomavirus (HPV) | 1 | from 11 years | `supplementary` | (verify) Single dose for girls and boys in the 5th grade of primary school (“5º grado de primaria”, about 10 to 11 years) and at “11 años” for those not in school; delivered by school grade, not by age, so it is supplementary and never flagged by age; the age is derived. |
