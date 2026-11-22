# Vaccination schedules: review documents

One review document per shipped schedule, generated from its CSV in `api/internal/vaccines/data/`.

| Id | Authority | Edition | Retrieved | Review document | Licence |
|---|---|---|---|---|---|
| `us` | Centers for Disease Control and Prevention (CDC) | Child and Adolescent Immunization Schedule by Age, 2025 (addendum updated July 2, 2025) | 2026-11-22 | [us.md](us.md) | Public domain, on CDC's four conditions |
| `uk` | UK Health Security Agency (UKHSA) | Complete routine immunisation schedule from 1 September 2026 (updated 24 September 2026), England | 2026-11-22 | [uk.md](uk.md) | Open Government Licence v3.0 |
| `mx` | Secretaría de Salud, CeNSIA | Lineamientos Generales del Programa de Vacunación Universal 2026, Cuadros 1 and 6 | 2026-11-22 | [mx.md](mx.md) | No open licence; facts only, with the citation the gob.mx terms ask for |

Switzerland (`ch`) is not shipped: the BAG requires written consent in advance, which has not been asked
for. Its research note is `docs/superpowers/research/vaccination/ch-bag.md`.

The `<id>.md` files are generated; do not edit them. After changing a CSV, run
`UPDATE_DOCS=1 go test ./internal/vaccines/ -race` in `api/` and commit the CSV and the document
together. `TestReviewDocsAreCurrent` fails when a document is missing or stale, and when a document has
no shipped schedule. This README is the one file it ignores, so update it by hand.

Each schedule is a partial, simplified transcription. Its "Not transcribed from the source" list comes
from the CSV's `omitted` key. The licence and attribution in full are in the CSV header, the review
document and the README's "Third-party data". The update procedure is
`docs/runbooks/updating-vaccine-schedules.md`.
