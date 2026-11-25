# Medical visits (list, visit, add and edit, after the visit)

Status: **draft for owner approval**. Owner decision 2026-11-25: there is no prototype for this module; the
pages are composed from existing designed pages, parts and wording, and this file is the page spec. Items
marked **(confirm)** are the author's choices waiting on that approval.

Behaviour comes from `docs/superpowers/specs/2026-11-25-medical-visits-design.md` ("the visits spec"); this
file decides only look, words and the no-JS routes. No prototype, no `specs/*.layout.json` and no baselines
exist: the measured numbers below are borrowed from the named page, and the visual baselines for these pages
are created from the build (listed as a deviation in the PR).

## Routes and files

| Route | Kind | Modelled on |
|---|---|---|
| `/kids/[kid]/visits` | page | `medication.md` (Courses list) |
| `/kids/[kid]/visits/new` | form page, opens in a sheet (`data-drawer="Add a visit"`) | `course.md` |
| `/kids/[kid]/visits/[visit]` | page (the visit) | `medication.md` card + `course.md` destructive |
| `/kids/[kid]/visits/[visit]/edit` | form page, sheet (`data-drawer="Edit visit"`) | `course.md` (SCREEN Edit) |
| `/kids/[kid]/visits/[visit]/outcome` | form page, sheet (`data-drawer="After the visit"`) | `course.md` + the Growth, Vaccination and Course field sets |
| `/kids/[kid]/visits/[visit]/calendar` | `+server.ts`, a download (no page) | `pdf-export.md` (Download PDF) |

The visit itself is a page, not a sheet: it is the record's home, as the Medication page is, and holds the
calendar download, the outcome summary and the delete confirm (AGENTS.md rule 1 covers its two forms).

## Parts

Card, Button, Badge, Input, Label, RadioGroup, Checkbox, Collapsible (`<details>`), Alert, Separator; the
existing `EntryTimeField`, `growth/MeasurementFields`, `vaccines/DoseFields`, `medication/CourseFields`
(with `ScheduleFields`), `FormFrame`, `FormCancel`, `DueList`, `TimelineRow`. No new part.

Icon: lucide `Stethoscope` in a `tracker-chip`, hue `--h:235` (the hue the Due list already uses for vaccines)
**(confirm: the preset has no visits hue)**.

## Words (fixed; accessible names in quotes)

- Nav (More sheet and side nav, after Medication): "Visits". Page h1: "Medical visits".
- Kind labels: `well_child` "Well-child visit", `sick` "Sick visit", `specialist` "Specialist visit",
  `dental` "Dental visit", `emergency` "Emergency visit", `other` "Other visit".
- Status badges: "Upcoming", "Past", "Cancelled".
- List add action: "Add a visit" **(confirm: the visits spec says "New visit"; every designed list says "Add a …")**.
- List groups (h2): "Upcoming", "Past". Empty: "No visits yet." + "Add a visit".
- Row: kind label (h3), then "{EEE d MMM yyyy}, {HH:mm}" and, when set, "{provider}"; a "Cancelled" badge.
  The row is a link to the visit, name "{kind label}, {d MMM yyyy}".
- Form fields: "Date and time" (EntryTimeField), "Length (minutes)", "Kind" (RadioGroup), "Doctor or clinic
  (optional)", "Place (optional)", "Reason (optional)". Save: "Save"; Cancel: `FormCancel`.
- Toasts: "Visit added", "Visit saved", "Visit cancelled", "Visit restored", "Visit deleted",
  "Visit recorded" (the outcome).
- Visit page: h1 = kind label; badge; "{EEEE d MMMM yyyy}, {HH:mm} · {n} min"; "{provider} · {place}";
  "Reason: {note}"; "Times are in {zone}".
- Actions on the visit page: "Edit visit" (link, drawer), "Cancel visit" / "Restore visit" (form POST), "Delete visit"
  (P2 confirm, below).
- Calendar (upcoming and cancelled visits only): legend "Add to calendar", RadioGroup "Without details"
  (description "Shows only “Medical visit”") / "With details" (description "Includes the child’s name, the
  doctor, the place and the reason"), default "Without details" **(confirm: the visits spec draws "With
  details" selected; the private default is proposed)**; button "Download .ics". A GET form, so it works
  without JavaScript.
- Past visit, no outcome yet: link "Record the visit" (drawer "After the visit"). With an outcome: section
  h2 "After the visit" with the notes, then h2 "Recorded at this visit" with one row per linked record
  ("Growth", "Vaccines", "Medication"; each a link to the record), and the same link renamed "Add more from
  the visit".
- Refusals (fixed): `has_outcome` "This visit has notes or records from the visit. Remove them first, or delete
  the visit."; `not_past` "This visit hasn’t happened yet."; `stale` "Someone changed this visit on another
  phone. It’s shown now; your entries are kept."; `limit` "This child has the most visits the app keeps (2,000)."

## After the visit (outcome form)

One sheet, sections in this order, each a `Checkbox` that discloses its field set (a `<details>` without
JavaScript):

1. "Notes from the visit" (textarea, always shown).
2. "Add a growth measurement": `MeasurementFields`, date prefilled with the visit's date.
3. "Add vaccines given": `DoseFields`; with JavaScript "Add another vaccine" adds a block up to 10.
4. "Add a medication": `CourseFields`; with JavaScript "Add another medication" adds a block up to 5.

A section whose module is switched off is not offered. Without JavaScript the sheet's page offers one vaccine
block and one medication block per save; saving again adds more. A refused field is named in its block
("Vaccine 2: Enter a lot of 40 characters or fewer.") and the summary alert is the usual "Check the fields
with a message.". Save: "Save"; toast "Visit recorded".

## Delete (P2 / DESIGN 4.5)

Text button "Delete visit" reveals the inline confirm card (`<details>` without JavaScript). Consequence first:

```
This visit recorded:
  • 1 growth measurement
  • 2 vaccines
  • 1 medication, with its 14 logged doses
[x] Also delete these records
[ Yes, delete this visit ]  [ Cancel ]
```

With nothing linked: "This deletes the visit." and no checkbox. Unticked, the sentence under the box reads
"The records stay in Growth, Vaccines and Medication." Toast "Visit deleted"; back to the list.

## Elsewhere

- Home Due: one line per upcoming visit in the next 7 days, "{kind label}, {EEE d MMM}, {HH:mm}", linking to
  the visit; counted into "N more" as "N more on the Visits page".
- Timeline: type chip "Visits"; row "{kind label}" with "Cancelled" when it applies; opens the visit page.
- Growth, Vaccines and Medication edit sheets: a line "Recorded at a visit" linking to it when `visit_id` is set.
- Modules page and Export: a "Medical visits" row/chip, from `MODULE_LABEL`.

## States

Normal (upcoming and past), empty, module off (DESIGN 8.5: "Medical visits is switched off for {child}"),
not found (one page), can't reach the server, a cancelled visit, a past visit with and without outcome,
the outcome refused (field), stale, and the delete confirm with and without linked records. Roles: guardian
and caregiver see the same controls.

## Acceptance

- axe clean per state in light and dark at 390 and 1280 px; targets ≥ 44 px; no horizontal scroll at 390 px
  and 200 % zoom; at ≥ 1280 px the content block is 920 px wide (DESIGN 2.1), as on the Medication page.
- Every no-JS route above works with JavaScript disabled.
- Visual baselines are created from the build for: list (default, empty), visit (upcoming, past with outcome,
  cancelled), and the three sheets.
