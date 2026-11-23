# Mexico national vaccination schedule: research for embedding (read-only research, 2026-11-22)

Not legal advice. Every numeric row below was read from a page I fetched in this session (PDF pages rendered and read visually, because the official PDFs are scanned images with no text layer). Confidence flags say how legible the source was.

## 0. Sources actually fetched (edition / date / URL)

| # | Document | Label / date as shown | URL | Used for |
|---|---|---|---|---|
| S1 | **Lineamientos Generales PVU 2026** (Programa de Vacunación Universal), CeNSIA, Secretaría de Salud | gob.mx page dated "31 de diciembre de 2025"; file `LINEAMIENTOS_GENERALES_2026_rem.pdf` (attachment 1076033; PDF ModDate 2026-04-30; 64 pp; signed by CeNSIA + IMSS, ISSSTE, PEMEX, SEDENA, SEMAR). Text: "de aplicación obligatoria en todos los niveles técnico-administrativos del SNS". | page https://www.gob.mx/salud/censia/documentos/lineamientos-generales-pvu-2026 ; PDF https://www.gob.mx/cms/uploads/attachment/file/1076033/LINEAMIENTOS_GENERALES_2026_rem.pdf | PRIMARY source. Cuadro 1 (p.5), Cuadro 6 (p.35), vaccine text pp.36-46, Cuadro 8 (p.41-42), Anexo 1 (p.64, a rotated, blurry 72-ppi scan). |
| S2 | **Enmienda a los Lineamientos Generales PVU 2026** (measles only) | "Ciudad de México a, 18 de febrero de 2026" | https://www.gob.mx/cms/uploads/attachment/file/1058162/ENMIENDA_LINEAMIENTOS_2026.pdf | measles/SRP outbreak rules. Bears the legend "PENDIENTE FIRMA". |
| S3 | **PRONAM Vacunación a lo largo de la vida, 2a edición** (Protocolo Nacional de Atención Médica, Consejo de Salubridad General), published in the DOF | "Actualización aprobada por el Consejo de Salubridad General en su Primera Sesión Extraordinaria 2026, el día 30 de julio de 2026" (1a ed.: 19 marzo 2025). PDF metadata title "2026.07.16 PRONAMA ..." | https://dof.gob.mx/2026/CSG/PRONAM_Vacunacion_a_lo_largo_de_la_vida.pdf | Cross-check (it has a real text layer). Its tables agree with S1. It defers to "Lineamientos Generales de Vacunación vigentes" and cites S1 for the varicela note. |
| S4 | gob.mx "Términos y Condiciones" | no date shown | https://www.gob.mx/terminos | licence / reuse |
| S5 | Ley Federal del Derecho de Autor (LFDA), "Última Reforma DOF 14-05-2026" | | https://www.diputados.gob.mx/LeyesBiblio/pdf/LFDA.pdf | art. 14, 148 |
| S6 | DOF 20/02/2015, Decreto por el que se establece la regulación en materia de Datos Abiertos | | https://www.gob.mx/cms/uploads/attachment/file/400568/datos_abiertos.pdf | art. 5 fr. III |

Unreachable or not found:
- The URL you gave, `https://www.gob.mx/salud/acciones-y-programas/esquema-de-vacunacion`, returns **404**. The `.../salud/censia/acciones-y-programas/esquema-de-vacunacion` variant also returns 404. The CeNSIA "Programa de Vacunación Universal" page (https://www.gob.mx/salud/censia/acciones-y-programas/programa-de-vacunacion-universal) loads but has no schedule on it. The schedule lives in S1.
- Not fetched: the **Manual de Vacunación** (S1 says "consulte el Manual de Vacunación vigente" for details) and the **Cartilla Nacional de Salud** PDFs.
- datos.gob.mx "Libre Uso MX" page returned **403**. The Libre Uso MX text is NOT verified from a primary source. I only verified the Decreto (S6).
- A page-by-page copyright scan of S1 was not done. I viewed pp.1-5, 22-26, 30-46 and 61-64 and saw no copyright or licence notice. Pp.6-21, 27-29 and 47-60 were not viewed (governance, coverage, ESAVI, migration).
- Several secondary sites (Studocu, Scribd, news, IMSS bulletins) corroborate 2-4-6 / 18 months. They were NOT used for any row.

**Edition verdict.** The current official text is S1 (LG PVU 2026) plus its measles-only amendment S2. S3 (PRONAM 2a ed., 30 Jul 2026) is the newest dated document and is consistent with S1 on every row below. Nothing newer than S3 was found. Edition label to cite: "Lineamientos Generales del Programa de Vacunación Universal 2026, CeNSIA, Secretaría de Salud (publicado 31-dic-2025; enmienda 18-feb-2026)".

## 1. Verbatim schedule (S1)

**Cuadro 1. Esquema de vacunación 2026, Programa de Vacunación Universal (S1 p.5; read at high resolution)**

| Grupo de edad | Vacuna | Esquema |
|---|---|---|
| Recién nacido | BCG | Una dosis |
| Recién nacido | Anti hepatitis B | Una dosis |
| Menores de un año | Hexavalente acelular | 1ª, 2ª y 3ª dosis |
| Menores de un año | Anti neumocócica conjugada | 1ª y 2ª dosis |
| Menores de un año | Anti-rotavirus | 1ª y 2ª dosis (RV1) |
| Menores de un año | Anti influenza* | 1ª y 2ª dosis (esquema primario) |
| Menores de un año | Anti COVID-19* | 1ª y 2ª dosis (esquema primario) |
| Un año | Anti neumocócica conjugada | 3ª dosis |
| Un año | Triple viral (SRP) | 1ª dosis |
| Un año | Anti varicela*** | Una dosis |
| 18 meses | Triple viral (SRP) | 2ª dosis |
| 18 meses | Hexavalente acelular | Dosis de refuerzo |
| 18 meses | Anti hepatitis A**** | Una dosis |
| 4 años | Triple bacteriana (DPT) | Dosis de refuerzo |
| 6 años | Triple viral (SRP) | 2ª dosis** |

Footnotes, verbatim:
- `*El esquema primario anti influenza y contra la COVID-19 consiste en 2 dosis a partir de los 6 meses de edad con un intervalo de 4 semanas entre cada una, y una dosis anual en cada temporada invernal, hasta los 59 meses de edad.`
- `**La segunda dosis de SRP a los 6 años se aplicará a niñas y niños nacidos antes de julio de 2020. Dicha estrategia se implementará hasta 2026, posteriormente se dejará de aplicar a esta edad, pero se seguirá aplicando a niñas y niños de 18 meses de edad.`
- `*** A la fecha de publicación del presente lineamiento, la vacuna NO forma parte del esquema básico de vacunación. Actualmente se aplica en guarderías o centros de cuidados infantiles.`
- `****ACUERDO-01CONAVA-2025. Se aprueba la incorporación al PVU de la vacuna contra el virus de la hepatitis tipo A, con la estrategia de vacunación universal en niñas y niños de 1 año o bien aquellos que no cuenten con antecedente vacunal.`

**Cuadro 6 (S1 p.35), per-visit view, 0 to 9 years**

| Age | Vaccines |
|---|---|
| Nacimiento | BCG; Anti hepatitis B |
| 2 meses | Hexavalente acelular; Anti rotavirus; Anti neumocócica conjugada |
| 4 meses | Hexavalente acelular; Anti rotavirus; Anti neumocócica conjugada |
| 6 meses | Hexavalente acelular; Anti influenza estacional (1ª dosis), en temporada invernal; Anti COVID-19 (1ª dosis) en temporada invernal |
| 7 meses | Influenza (2ª dosis) "en la misma temporada invernal que la primera dosis, posteriormente una dosis anual hasta los 59 meses, en cada temporada invernal"; COVID-19 (2ª dosis), same wording |
| 12 meses | SRP; Anti neumocócica conjugada; Anti varicela^a |
| 18 meses | Hexavalente acelular; SRP (2ª dosis)^b; Anti hepatitis A^c |
| 48 meses (4 años) | DPT (refuerzo) |
| A partir de los 5 años | Anti influenza estacional^d; Anti COVID-19^e |
| (6 años) | SRP (2ª dosis)^f |

Cuadro 6 footnotes: ^b "La segunda dosis de SRP a los 18 meses aplica solo para niñas y niños nacidos a partir de julio de 2020."; ^d "Una dosis anual en cada temporada invernal, se aplica a partir de los 5 años en niñas y niños que vivan con alguna comorbilidad de riesgo."; ^f same text as Cuadro 1 `**`.

**Vaccine-specific text (S1 pp.36-40)**
- BCG: "dosis única de 0.1 mL ... a la población recién nacida con peso igual o mayor a 2,000 gramos, previo a su egreso del hospital, o en el primer contacto con los servicios de salud, preferentemente antes del año de edad. Se podrá vacunar a niñas y niños menores de 5 años y excepcionalmente a menores de 14 años que no hayan sido vacunados."
- Hepatitis B: "en las primeras 24 horas de vida, en recién nacidos con peso ≥2,000 gramos o <2,000 gramos clínicamente estables. Se podrá vacunar de forma extemporánea hasta los 7 días de vida." / "Las niñas y niños que no sean vacunados al nacimiento deberán iniciar su esquema a los 2 meses de edad con la vacuna hexavalente acelular" / "Los recién nacidos pretérmino (<29 semanas de edad gestacional) o con peso <1,500 gramos deberán vacunarse entre las 6-8 semanas de vida." / "No es necesario aplicar dosis adicionales de hepatitis B si se está utilizando la vacuna hexavalente, ya que ésta contiene el componente contra la hepatitis B."
- Hexavalente acelular (DPaT+VIP+HB+Hib): "contiene antígenos que protegen contra difteria, tos ferina (de componente acelular), tétanos, poliomielitis, hepatitis B y enfermedades invasivas por Haemophilus influenzae tipo b." / "El esquema primario de vacunación consiste en 3 dosis a los 2, 4 y 6 meses de edad y una dosis de refuerzo a los 18 meses de edad." / "Puede aplicarse si se ha administrado o no una dosis de vacuna contra la hepatitis B al nacimiento."
- Rotavirus RV1: "2 dosis de 1.5 mL cada una, vía oral a los 2 y 4 meses de edad. Ninguna persona deberá recibir esta vacuna después de los 7 meses con 29 días de edad." Cuadro 7: the RV5 (pentavalente) scheme is 3 doses at 2, 4 and 6 months, and mixed schemes are allowed.
- Neumocócica conjugada: "3 dosis de 0.5 mL ... a los 2, 4 y 12 meses de edad."
- Influenza pediátrica: "Niñas y niños de 6 a 59 meses de edad: 2 dosis de 0.5 mL cada una, con intervalo de 4 semanas entre ellas, si no tienen antecedente de haber recibido la vacuna o si no completaron el esquema de 2 dosis en una misma temporada invernal. Posteriormente una dosis anual de 0.5 mL hasta los 59 meses." Winter 2026-2027 campaign "iniciará en el último trimestre de 2026". For 5-9 years: comorbidity only.
- SRP: "Primera dosis al cumplir 12 meses de edad. Segunda dosis a los 18 meses de edad." / "dos dosis de SRP (0.5 mL) con un intervalo de 4 semanas, a menores de 10 años que no hayan recibido ninguna dosis" / outbreak: "podrá vacunarse a las niñas y niños de 6 a 11 meses ... (esta dosis se considera la dosis cero y se inicia el esquema a partir de los 12 meses de edad, con un intervalo mínimo de 4 semanas entre ambas dosis)."
- DPT: "Posterior a la aplicación del esquema primario con la vacuna hexavalente acelular, se aplica una dosis de 0.5 mL de la vacuna DPT ... a los 4 años de edad. En aquellos casos en los que no se reciba a los 4 años, la edad de aplicación no debe sobrepasar los 6 años, 11 meses y 29 días. El intervalo mínimo entre DPT y la cuarta dosis de hexavalente, es de 6 semanas." Footnote 9: "De manera excepcional ... podrá aplicar la vacuna DPT en niñas y niños que acudan a guarderías o estancias infantiles, así como en situaciones de emergencia a partir de los 3 años y 6 meses de edad."
- Hepatitis A: "Dosis única de 0.5 mL ... a niñas y niños de 18 meses de edad."
- Varicela: "1 dosis de 0.5 mL ... en niñas y niños de 12 meses de edad que acuden a guarderías o centros de cuidados infantiles." Footnote 11: "La vacuna contra varicela a la fecha de publicación no forma parte del PVU."
- COVID-19 (6-59 months): same 2-dose and annual pattern as influenza; footnote 10 says "La política de vacunación así como las vacunas disponibles contra COVID-19 podrían modificarse durante 2026".

**School-age / adolescent / adult lines (S1 Cuadro 8 pp.41-42 and text pp.44-46; S3 for cross-check)**
- Td: "Personas con esquema completo: Aplicar a partir de los 15 años de edad, con refuerzo cada 10 años." / "esquema incompleto o no documentado: Aplicar 3 dosis con intervalos 0, 1, 12 meses, con refuerzo cada 10 años." Definition (p.42): "El esquema primario completo para proteger contra tétanos y difteria consiste en: 4 dosis de vacuna hexavalente (DPaT+VIP+HB+Hib) a los 2, 4, 6 y 18 meses de edad, más una dosis de refuerzo con la vacuna DPT a los 4 años de edad. Este esquema debe estar cubierto máximo a los 6 años 11 meses de edad, con la dosis de DPT." Cuadro 8 footnote: "esquema completo ... 5 dosis del esquema primario (4 dosis de hexavalente acelular + una dosis de DPT) o 3 dosis de Td".
- SR / SRP: "Personas de 10 a 49 años de edad, en personas sin esquema previo, esquema incompleto o desconocido." (1 dose if one prior dose, 2 doses 4 weeks apart if none.)
- Anti hepatitis B: "A partir de los 11 años de edad en personas sin antecedente vacunal (al menos 3 dosis de Hb)" (2 x 20 ug, or 3 x 10 ug at 0, 1, 6 months).
- VPH: "Niñas y niños que cursan el 5º grado de primaria y de 11 años NO escolarizados" (single dose: "Se aplicará a dosis única de 0.5 mL"; "A partir del 2025 se incorporó como población objetivo a los niños de quinto grado de primaria"). Also 11-49 with VIH (3 doses 0-2-6), 9-19 years after sexual assault (1 dose). S3 Table 6 also lists "Grupo rezagado: Mujeres adolescentes de 12 a 16 años, sin antecedente vacunal, dosis única".
- Tdpa: pregnancy, each pregnancy, from week 20 (preferably 27-36). Also close contacts of infants under 1 year.
- Out of scope: influenza/COVID/pneumococcal/VSR for risk groups.
- Measles amendment S2 (18-feb-2026), "Población de 6 meses a 12 años": "Dosis Cero: Niñas y niños de 6 a 11 meses, aplicar una dosis. Esquema regular: primera dosis a los 12 meses y segunda dosis a los 18 meses. Población rezagada: Niñas y niños de 1 a 12 años que no tengan esquema o que cuenten con esquema incompleto, iniciar o completar esquema." Also "Si ya tiene 2 dosis, ¡YA NO REQUIERE VACUNARSE!".

**Anexo 1 (S1 p.64): min/max ages and intervals.** This is a rotated 72-ppi scan. Values below were read from a blown-up render and are **medium confidence**. Every "recommended age" value matches Cuadro 1/6 (high). Re-check the minimum ages and intervals against a better scan before shipping.

| Dose | Edad recomendada | Edad mínima permitida | Edad máxima permitida | Intervalo recomendado a la sig. dosis | Intervalo mínimo a la sig. dosis |
|---|---|---|---|---|---|
| BCG | Al nacer | Al nacer | <5 años; excepcionalmente <14 años | no aplica | no aplica |
| Hepatitis B | Al nacer y hasta los 7 días de vida | Al nacer | "Preferentemente no después de los 7 días de vida. Continuar esquema con vacuna hexavalente." | no aplica | no aplica |
| Hexavalente 1 | 2 meses | 6 semanas | 59 meses y 29 días (all four doses) | 8 semanas | 4 semanas |
| Hexavalente 2 | 4 meses | 10 semanas | | 8 semanas | 4 semanas |
| Hexavalente 3 | 6 meses | 14 semanas | | 12 meses | "8 semanas, si la 3ra dosis se aplicó después del año de edad" |
| Hexavalente refuerzo | 18 meses | 12 meses | | no aplica | no aplica |
| Neumocócica 1 | 2 meses | 6 semanas | 59 meses y 29 días | 8 semanas | 4 semanas en <12 meses; 8 semanas en >=12 meses |
| Neumocócica 2 | 4 meses | 10 semanas | | 8 meses | 8 semanas |
| Neumocócica 3 | 12 meses | 12 meses | | no aplica | no aplica |
| Rotavirus RV1 1 | 2 meses | 6 semanas | 7 meses y 29 días | 8 semanas | 4 semanas |
| Rotavirus RV1 2 | 4 meses | 10 semanas | | no aplica | no aplica |
| Influenza 1 | 6 meses | 6 meses | 59 meses y 29 días | 4 semanas | "4 semanas o mayor intervalo mientras se aplique en la misma temporada invernal" |
| Influenza 2 | 7 meses | 7 meses | | Anual | no aplica |
| Influenza dosis anual | "Una dosis en cada temporada invernal hasta los 59 meses" | | 59 meses y 29 días | Anual | |
| SRP 1 | 12 meses | 12 meses | <10 años | "6 meses* o 5 años**" | |
| SRP 2 | "18 meses* o 6 años**" | 18 meses | | no aplica | |
| Varicela | 12 meses | 12 meses | 59 meses y 29 días | no aplica | |
| Hepatitis A | 18 meses | 18 meses | 59 meses y 29 días | no aplica | |
| DPT | 4 años | 4 años (excepcionalmente 3 años y 6 meses) | <7 años | no aplica | |

(Annex footnotes `*`/`**`/`***` repeat the SRP born-before/after-July-2020 and varicela notes. The BCG, SRP and DPT max ages and the min intervals are the least legible cells.)

## 2. Proposed machine-readable draft (ages 0 to 6 years)

Design notes for this file:
- The Esquema gives **point ages** ("a los 2, 4 y 6 meses") and never states a window end for routine doses. It states only a **maximum permitted age**. So `recommended_to_months` is **null** wherever the source gives no end. Nothing was invented. If the app needs a soft "overdue" threshold, that is an owner decision (options: the next dose's recommended age, or `max_age`). Mark any default as an app policy, not as a source fact.
- Sub-month ages are given in days (`min_age_days`). Convert months to days with whatever rule the app uses and keep it consistent.
- `series_number` counts per antigen, across all products that carry it (so HepB birth = 1, hexavalent doses = 2..5). The source itself numbers doses per product ("1ª, 2ª, 3ª dosis" and "refuerzo"), so `dose_name_es` keeps the product-level label.
- Counts per antigen for a fully scheduled child at 6 years: DTP-containing (diphtheria, tetanus, pertussis) = 5 (4 hexavalente + 1 DPT; the source calls this "5 dosis del esquema primario"); polio (VIP) = 4; Hib = 4; hepatitis B = 5 if the birth dose was given (1 + 4 hexavalente), 4 if not; BCG/tuberculosis = 1; rotavirus = 2 (RV1) or 3 (RV5); pneumococcus = 3; measles, mumps, rubella = 2 each; hepatitis A = 1; influenza = 2 in the first season then 1 per season; COVID-19 = 2 then 1 per season.

```yaml
source:
  label: "Lineamientos Generales del Programa de Vacunación Universal 2026 (CeNSIA, Secretaría de Salud); Cuadros 1 y 6, Anexo 1; enmienda 18-feb-2026 (sarampión)"
  url: https://www.gob.mx/cms/uploads/attachment/file/1076033/LINEAMIENTOS_GENERALES_2026_rem.pdf
  published: "2025-12-31 (date shown on gob.mx page); pdf modified 2026-04-30"
  crosscheck: "PRONAM Vacunación a lo largo de la vida, 2a ed., CSG 2026-07-30, DOF"
  consulted: "2026-11-22"
doses:
# --- recien nacido ---
- {antigen: tuberculosis,   product: BCG,  dose_name_es: "BCG única",              series_number: 1, rec_from_months: 0,  rec_to_months: null, min_age: "al nacer", max_age: "<5 años (excepcionalmente <14 años)", age_es: "Recién nacido / 'preferentemente antes del año de edad'", notes: "Único; sin ventana final declarada (preferentemente antes de 12 meses). Peso >=2000 g.", quote_confidence: high}
- {antigen: hepatitis_b,    product: HB,   dose_name_es: "Anti hepatitis B (única, RN)", series_number: 1, rec_from_months: 0, rec_to_months: null, min_age: "al nacer", max_age: "7 días (extemporánea); luego continuar con hexavalente", age_es: "'primeras 24 horas de vida ... extemporánea hasta los 7 días de vida'", notes: "7 días = ventana del RN (rec_to ~0.23 meses; guardar en días). No contar como hexavalente.", quote_confidence: high}
# --- 2 meses ---
- {antigen: [diphtheria, tetanus, pertussis_acellular, polio_ipv, hepatitis_b, hib], product: hexavalente_acelular, dose_name_es: "Hexavalente acelular, 1ª dosis", series_number: {diphtheria:1, tetanus:1, pertussis:1, polio:1, hib:1, hepatitis_b: "2 (1 sin dosis RN)"}, rec_from_months: 2, rec_to_months: null, min_age_days: 42, max_age: "59 meses 29 días", age_es: "2 meses", notes: "HepB: si hubo dosis al nacer, esta es la 2ª dosis de HB; si no, es la 1ª. El texto no pide dosis extra de HB con hexavalente.", quote_confidence: high}
- {antigen: rotavirus,      product: RV1, dose_name_es: "Rotavirus RV1, 1ª dosis", series_number: 1, rec_from_months: 2, rec_to_months: null, min_age_days: 42, max_age: "7 meses 29 días", age_es: "2 meses", notes: "RV5 (pentavalente) usa 3 dosis 2-4-6 (Cuadro 7). 'Pentavalente' aquí = rotavirus, no DTP.", quote_confidence: high}
- {antigen: pneumococcus,   product: PCV,  dose_name_es: "Neumocócica conjugada, 1ª dosis", series_number: 1, rec_from_months: 2, rec_to_months: null, min_age_days: 42, max_age: "59 meses 29 días", age_es: "2 meses", notes: "Producto no fijado en LG ('conjugada'); PRONAM dice PCV20.", quote_confidence: high}
# --- 4 meses ---
- {antigen: [diphtheria, tetanus, pertussis_acellular, polio_ipv, hepatitis_b, hib], product: hexavalente_acelular, dose_name_es: "Hexavalente acelular, 2ª dosis", series_number: {diphtheria:2, tetanus:2, pertussis:2, polio:2, hib:2, hepatitis_b: "3 (o 2 sin dosis RN)"}, rec_from_months: 4, rec_to_months: null, min_age_days: 70, min_interval: "4 semanas", age_es: "4 meses", quote_confidence: high}
- {antigen: rotavirus,      product: RV1, dose_name_es: "Rotavirus RV1, 2ª dosis", series_number: 2, rec_from_months: 4, rec_to_months: null, min_age_days: 70, max_age: "7 meses 29 días", min_interval: "4 semanas", age_es: "4 meses", quote_confidence: high}
- {antigen: pneumococcus,   product: PCV,  dose_name_es: "Neumocócica conjugada, 2ª dosis", series_number: 2, rec_from_months: 4, rec_to_months: null, min_age_days: 70, min_interval: "4 semanas (<12 m)", age_es: "4 meses", quote_confidence: high}
# --- 6 meses ---
- {antigen: [diphtheria, tetanus, pertussis_acellular, polio_ipv, hepatitis_b, hib], product: hexavalente_acelular, dose_name_es: "Hexavalente acelular, 3ª dosis", series_number: {diphtheria:3, tetanus:3, pertussis:3, polio:3, hib:3, hepatitis_b: "4 (o 3 sin dosis RN)"}, rec_from_months: 6, rec_to_months: null, min_age_days: 98, age_es: "6 meses", quote_confidence: high}
- {antigen: influenza,      product: influenza_inactivada, dose_name_es: "Influenza 1ª dosis (esquema primario)", series_number: 1, rec_from_months: 6, rec_to_months: null, min_age_months: 6, max_age: "59 meses 29 días", age_es: "a partir de los 6 meses, en temporada invernal", recurring: true, notes: "Estacional (campaña oct-2026 en adelante). 2 dosis separadas 4 semanas si no hay antecedente o no completó en la misma temporada.", quote_confidence: high}
- {antigen: covid19,        product: covid19, dose_name_es: "COVID-19 1ª dosis (esquema primario)", series_number: 1, rec_from_months: 6, rec_to_months: null, min_age_months: 6, age_es: "6 meses, en temporada invernal", recurring: true, notes: "LG: 'política ... podrá modificarse durante 2026'. Probablemente excluir del conteo por defecto.", quote_confidence: medium}
# --- 7 meses ---
- {antigen: influenza,      product: influenza_inactivada, dose_name_es: "Influenza 2ª dosis", series_number: 2, rec_from_months: 7, rec_to_months: null, min_age_months: 7, age_es: "7 meses, misma temporada que la 1ª", notes: "Luego una dosis anual en cada temporada invernal hasta los 59 meses.", quote_confidence: high}
- {antigen: covid19,        product: covid19, dose_name_es: "COVID-19 2ª dosis", series_number: 2, rec_from_months: 7, rec_to_months: null, age_es: "7 meses, misma temporada", quote_confidence: medium}
# --- 12 meses ---
- {antigen: [measles, mumps, rubella], product: SRP, dose_name_es: "Triple viral (SRP), 1ª dosis", series_number: 1, rec_from_months: 12, rec_to_months: null, min_age_months: 12, max_age: "<10 años", age_es: "12 meses / 'al cumplir 12 meses'", notes: "Dosis cero 6-11 m en brote NO cuenta (S2, LG p.39).", quote_confidence: high}
- {antigen: pneumococcus,   product: PCV,  dose_name_es: "Neumocócica conjugada, 3ª dosis", series_number: 3, rec_from_months: 12, rec_to_months: null, min_age_months: 12, max_age: "59 meses 29 días", age_es: "12 meses", quote_confidence: high}
- {antigen: varicella,      product: varicela, dose_name_es: "Varicela (única)", series_number: 1, rec_from_months: 12, rec_to_months: null, min_age_months: 12, max_age: "59 meses 29 días", age_es: "'niños de 1 año que asisten a guarderías o centros de cuidado infantil'", optional: true, notes: "NO forma parte del esquema básico a la fecha. Marcar opcional; no usar para 'atrasado'.", quote_confidence: high}
# --- 18 meses ---
- {antigen: [diphtheria, tetanus, pertussis_acellular, polio_ipv, hepatitis_b, hib], product: hexavalente_acelular, dose_name_es: "Hexavalente acelular, dosis de refuerzo", series_number: {diphtheria:4, tetanus:4, pertussis:4, polio:4, hib:4, hepatitis_b: "5 (o 4 sin dosis RN)"}, rec_from_months: 18, rec_to_months: null, min_age_months: 12, max_age: "59 meses 29 días", age_es: "18 meses", notes: "El esquema primario son 3 dosis (2,4,6); el 'refuerzo' a los 18 m es la 4ª dosis. Intervalo mínimo desde la 3ª: 8 semanas si la 3ª se dio tras el año (Anexo 1, medium).", quote_confidence: high}
- {antigen: [measles, mumps, rubella], product: SRP, dose_name_es: "Triple viral (SRP), 2ª dosis", series_number: 2, rec_from_months: 18, rec_to_months: null, min_age_months: 18, age_es: "18 meses (nacidos a partir de julio de 2020)", notes: "Nacidos antes de jul-2020: 2ª dosis a los 6 años (fila siguiente).", quote_confidence: high}
- {antigen: hepatitis_a,    product: HepA, dose_name_es: "Anti hepatitis A (única)", series_number: 1, rec_from_months: 18, rec_to_months: null, min_age_months: 18, max_age: "59 meses 29 días", age_es: "18 meses ('niños de 1 año o ... sin antecedente')", notes: "Nueva en PVU (ACUERDO-01CONAVA-2025). Cuadro 6 la coloca a 18 m; el pie dice 'universal en niñas y niños de 1 año o ... sin antecedente vacunal'.", quote_confidence: high}
# --- 4 años ---
- {antigen: [diphtheria, tetanus, pertussis_whole_cell], product: DPT, dose_name_es: "Triple bacteriana (DPT), dosis de refuerzo", series_number: {diphtheria:5, tetanus:5, pertussis:5}, rec_from_months: 48, rec_to_months: null, min_age_months: 48, min_age_exception: "3 años 6 meses (guardería/emergencia)", max_age: "6 años 11 meses 29 días", age_es: "4 años / 'a los 4 años de edad'", notes: "Intervalo mínimo 6 semanas tras la 4ª hexavalente. DPT es de células enteras; no polio ni Hib ni HB.", quote_confidence: high}
# --- 6 años ---
- {antigen: [measles, mumps, rubella], product: SRP, dose_name_es: "Triple viral (SRP), 2ª dosis a los 6 años", series_number: 2, rec_from_months: 72, rec_to_months: null, age_es: "6 años (nacidos antes de julio de 2020)", applies_to: "birth_date < 2020-07-01", notes: "'Dicha estrategia se implementará hasta 2026'. Ya no aplica a nacidos desde 2020-07; a esos se les da a los 18 m.", quote_confidence: high}
```

Annual influenza (and COVID-19) after the primary series: one dose per winter season to 59 months. For ages 5 to 9 only with a comorbidity. The "rec_to" is a **season**, not an age window, so model it as `recurring: seasonal`, not as a dose with an age window.

### School / adolescent summary (not full rows)
| Line | Antigen(s) | As stated | Notes |
|---|---|---|---|
| SRP 6 años | measles/mumps/rubella | "6 años" for children born before July 2020 | see row above |
| SR/SRP rezagados | measles, rubella (+mumps) | "10 a 49 años ... sin esquema previo, esquema incompleto o desconocido" | catch-up, risk-based |
| Hepatitis B | HB | "A partir de los 11 años de edad en personas sin antecedente vacunal (al menos 3 dosis de Hb)" | 2 x 20 ug or 3 x 10 ug |
| VPH | HPV | "5º grado de primaria" or "11 años NO escolarizados"; single dose | now boys and girls; catch-up girls 12-16 (PRONAM) |
| Td | tetanus, diphtheria | "a partir de los 15 años", booster each 10 years; incomplete/undocumented: 0, 1, 12 months | school-age grade not stated |
| Tdpa | tetanus, diphtheria, pertussis | each pregnancy, from week 20 (preferably 27-36) | |
| Influenza / COVID / neumococo | | risk-based from 5 years (influenza, COVID) or 9 years (pneumococcus PRONAM) | skip for a baby app |

## 3. Licence and terms of reuse

### Findings (quotes in Spanish, English paraphrase after each)

1. **gob.mx "Términos y Condiciones"** (https://www.gob.mx/terminos), obligations of the user:
   - `Se autoriza al Usuario a visualizar y descargar los materiales contenidos en el sitio solamente para su uso personal y no para un uso comercial.`
   - `El Usuario deberá conservar en todas las copias de los materiales descargados, todos los avisos sobre derechos de autor y propiedad intelectual de los materiales, contenidos en los mismos;`
   - `El Usuario no deberá modificar, reproducir o mostrar pública o comercialmente los materiales, ni podrá distribuir o utilizarlos con algún propósito público o comercial;`
   - `El Usuario no deberá transferir los materiales a ningún tercero.`
   Paraphrase: a generic site clause allowing personal, non-commercial viewing and download only, with no public reproduction or redistribution. Read literally it would forbid redistributing the PDFs themselves in a public repo or image.

2. **Same page, section "Propiedad intelectual"**, the more specific clause:
   `La información del portal gob.mx es pública a menos que se indique lo contrario, en cuyo caso antes de reproducirla, deberás observar si tiene derechos reservados y respetarlos en términos de las normas relativas a derechos de autor y propiedad industrial. Adicionalmente el usuario que reproduzca o procese información contenida en gob.mx deberá referir la localización electrónica y fecha en que se realizó la consulta de la información. El usuario se compromete a respetar y dejar a salvo los derechos de terceros ...`
   Paraphrase: information on gob.mx is public unless marked otherwise; if a reservation exists, respect it. Anyone who reproduces or processes the information must cite the **electronic location (URL) and the date it was consulted**. This is the only attribution requirement the page states. The page shows no date of last update and does not mention a "Licencia de Uso Libre".

3. **LFDA art. 14 fr. VIII** (https://www.diputados.gob.mx/LeyesBiblio/pdf/LFDA.pdf, Última Reforma DOF 14-05-2026):
   `No son objeto de la protección como derecho de autor a que se refiere esta Ley: ... VIII. Los textos legislativos, reglamentarios, administrativos o judiciales, así como sus traducciones oficiales. En caso de ser publicados, deberán apegarse al texto oficial y no conferirán derecho exclusivo de edición; Sin embargo, serán objeto de protección las concordancias, interpretaciones, estudios comparativos, anotaciones, comentarios y demás trabajos similares que entrañen, por parte de su autor, la creación de una obra original;`
   Also fr. I: `Las ideas en sí mismas, las fórmulas, soluciones, conceptos, métodos, sistemas, principios ...` and fr. X: `La información de uso común tal como los refranes, dichos, leyendas, hechos, calendarios y las escalas métricas.`
   Paraphrase: administrative texts are not protected by copyright and confer no exclusive right of edition, but they must be reproduced faithfully to the official text, and original commentary or annotations remain protected. Facts, methods and calendars are also outside protection.
   Caveat: whether the Lineamientos (a "lineamiento" of the Sistema Nacional de Salud, signed by several institutions) or the PRONAM (published in the DOF) count as "textos administrativos" is a legal-classification question I did not resolve. Neither is "legislativo" or "judicial".

4. **Decreto de Datos Abiertos** (DOF 20/02/2015, https://www.gob.mx/cms/uploads/attachment/file/400568/datos_abiertos.pdf), art. quinto fr. III:
   `De libre uso: Citarán la fuente de origen como único requerimiento para ser utilizados libremente;`
   Paraphrase: datasets published as official open data are reusable freely with source citation as the only condition. This applies to datasets in the datos.gob.mx catalogue. The vaccination schedule is not published there. I did not find it in any dataset, so this decree is only an indication of policy, not a licence for the schedule PDF. The "Licencia de Uso Libre MX" itself could not be fetched (datos.gob.mx returned 403), so its text is unverified.

5. **LFDA art. 148** (quotation right): `... podrán utilizarse, siempre que no se afecte la explotación normal de la obra, sin autorización del titular del derecho patrimonial y sin remuneración, citando invariablemente la fuente y sin alterar la obra ...` fr. I `Cita de textos` and fr. III `Reproducción de partes de la obra, para la crítica e investigación científica ...`. This is a fallback only. It is not designed for embedding a full table in a product.

6. **No copyright, licence or "derechos reservados" notice** appeared in the pages of S1 or in S3 that I read. S3 carries the Gobierno de México / Salud / CSG logos. S1 carries the institutional logos and a "2026 año de Margarita Maza" banner.

### Conclusion
- There is **no explicit open licence** (no CC, no "Libre Uso MX" statement) on S1, S2 or S3. The gob.mx site terms have a contradictory pair: a personal-use-only clause and a "public unless reserved, cite URL + consultation date" clause.
- The practical position is that transcribing **facts** (antigen, dose number, age, interval) into a data file is low risk: facts and calendars are outside copyright (LFDA 14 I, X), the content is an official administrative text (14 VIII), and the site's own "Propiedad intelectual" clause treats the information as public and asks only for a citation. This is my reading, not a guaranteed legal clearance.
- **Do not** embed the PDFs or page images, the Escudo Nacional or the logos (LFDA 14 VII covers unauthorised reproduction of emblems), and avoid copying explanatory prose. Keep your own wording for notes and keep only short quoted age strings as `age_es`.
- **Attribution to include** (data file header, repo NOTICE/README, and an in-app "Fuente" line): document name and edition, issuer, URL, and the date consulted. Suggested text: `Fuente: Secretaría de Salud, CeNSIA, "Lineamientos Generales del Programa de Vacunación Universal 2026" (31-dic-2025) y su enmienda del 18-feb-2026; contrastado con el PRONAM "Vacunación a lo largo de la vida", 2a ed. (CSG, 30-jul-2026). https://www.gob.mx/salud/censia/documentos/lineamientos-generales-pvu-2026 . Consultado el 2026-11-22. Transcripción no oficial; no sustituye la Cartilla Nacional de Salud.` Add a disclaimer that the app is not an official source and is not endorsed by the Secretaría de Salud. MIT applies to your code. The data's provenance is the Secretaría de Salud text, and you cannot relicense the underlying content, so say so in the data file header.
- To remove all doubt, CeNSIA's published contact is `transparencia.censia@salud.gob.mx` (on the gob.mx page). A short written confirmation that transcribed schedule data may be redistributed with citation would settle it. I did not contact anyone.

## 4. Traps for the transcriber

1. **Point ages vs windows.** "2, 4 y 6 meses" are recommended ages, not ranges. The only ranges in the source are minimum ages and maximum ages (Anexo 1). A soft "overdue" flag needs an app-chosen grace rule.
2. **"Refuerzo" does not mean a separate antigen series.** Hexavalente "refuerzo" at 18 months is dose #4 of DTP/polio/Hib/HB. DPT "refuerzo" at 4 years is dose #5 of D/T/P only (whole-cell pertussis, no polio/Hib/HB). The source's own wording: the primary scheme is "3 dosis a los 2, 4 y 6 meses ... y una dosis de refuerzo a los 18 meses", while the Td section calls all four hexavalente plus DPT "las 5 dosis del esquema primario".
3. **Hepatitis B counting.** Birth dose is separate and "única"; hexavalente "contiene el componente contra la hepatitis B" and is given whether or not a birth dose was given. A child with no birth dose still reaches 4 HB doses from hexavalente. A count-by-antigen model should accept 4 or 5 and not flag a missing birth dose after day 7 (the source says to continue with hexavalente).
4. **"Pentavalente" is ambiguous.** In S1 it is the rotavirus RV5 vaccine (3 doses at 2/4/6 m). In S3 Td table it also appears as "pentavalente acelular + DPT" (older DTP-HB-Hib-IPV combo scheme counted as a complete series). I did not fetch any text describing the older pentavalente/Sabin infant scheme still used in some places, so I cannot confirm its schedule. Expect legacy cartillas with "Pentavalente" and "Sabin" entries and plan to map them manually.
5. **SRP 2nd dose moves with birth date.** 18 months for children born from July 2020; 6 years for children born before July 2020 ("estrategia ... hasta 2026"). A child aged 6+ in 2026 who got dose 2 at 6 years is complete. Do not hard-code either age.
6. **Dosis cero.** The 6-11 month measles dose in outbreak/blockade ("dosis cero") does not count towards the 2 doses. The S2 amendment lists affected states (Colima, Chiapas, CDMX, Jalisco, Nayarit, Sinaloa, Durango, Sonora, Tlaxcala, Puebla, Tabasco). The S2 text is marked "PENDIENTE FIRMA", so treat outbreak rules as campaign, not routine.
7. **Seasonal vs routine.** Influenza and COVID-19 are winter-campaign vaccines (2026-27 "iniciará en el último trimestre de 2026"). The ages 6 and 7 months are "in the same winter season", so a baby aged 6 m in spring has no due date until the campaign. COVID policy is explicitly "podrá modificarse durante 2026".
8. **Not basic schedule.** Varicela is "NO ... esquema básico", daycare only, state availability. Hepatitis A (18 m) is new in 2025-26 (ACUERDO-01CONAVA-2025), so older cartillas and many physicians' lists lack it. Cuadro 1 puts it at 18 m while the footnote says "niñas y niños de 1 año o ... sin antecedente vacunal".
9. **Pneumococcal product.** LG says only "conjugada" (2, 4, 12 m). PRONAM names PCV20. Mixed or earlier PCV13/PCV10 series count as the same antigen per the source ("deberán continuar con su esquema ...").
10. **Years vs months.** Cuadro 1 groups "Un año" (12-month visit) and then uses "18 meses", "4 años", "6 años". Hexavalente max age is "59 meses y 29 días" (not "5 años"); DPT max is "6 años, 11 meses y 29 días".
11. **Vaccine campaigns vs the esquema.** VPH is delivered by school campaign in "5º de primaria" (grade-based, not age-based; "11 años NO escolarizados" for out-of-school). Td, SR and HB for adolescents are risk/catch-up, not universal ("no son consideradas universales").
12. **Source quality.** Official PDFs are scans. Annex 1 intervals are blurry, and the high-confidence rows come from Cuadros 1/6 and the vaccine paragraphs. Re-verify Anexo 1 numbers (especially min intervals, BCG/SRP/DPT max ages) from the Manual de Vacunación or a clean copy before shipping.
13. **Incomplete schedules are never restarted.** S1 5.2.1.8: "los esquemas incompletos no deben reiniciarse", which fits count-by-antigen matching.

## 5. Could not retrieve / lower confidence
- gob.mx "esquema-de-vacunacion" URL (404 on both variants).
- Manual de Vacunación and Cartilla Nacional de Salud (not fetched).
- datos.gob.mx Libre Uso MX text (403).
- Anexo 1 min/max/interval cells (medium). Cuadros 1 and 6 and all vaccine text (high).
- S1 pp.6-21, 27-29, 47-60 not viewed.
