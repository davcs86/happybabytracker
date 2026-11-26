# Medical visits (list, visit, add and edit, after the visit)

Status: **approved by the owner on 2026-11-25**, including the three items marked **(confirm)**. Owner decision
2026-11-25: there is no prototype for this module; the pages are composed from existing designed pages, parts and
wording, and this file is the page spec.

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
  The row is a link to the visit; its accessible name is its visible text (axe label-content-name-mismatch, WCAG 2.5.3; the timeline rows' precedent).
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

## Find a US provider (plan 10)

Status: **approved by the owner on 2026-11-26**, including its five items marked **(confirm)** (plan 10 rulings 2
and 6). Behaviour is in the visits
spec §10 and `docs/superpowers/plans/2026-11-26-provider-lookup.md`. This section decides only look, words and
the no-JS route. No prototype and no baseline exist. The visual baselines come from the build, against the stub
(listed as a deviation in the PR).

### Where it is

- Inside the existing "Add a visit" and "Edit visit" sheets (and their pages), right above "Doctor or clinic
  (optional)". It is not on the "After the visit" sheet.
- Shown only while the server's lookup is on (`GET /v1/providers`). When it is off, the sheet is exactly what it is
  today: no panel, no hint.
- Closed by default. It opens again after a search, so the results are in view, and closes after a pick.

### Parts

Collapsible (`<details>`), Label, Input, RadioGroup, native Select (DESIGN 4.3 control layer), Button (secondary
for Search, primary stays Save), Alert, Separator. Nothing new; no icon **(confirm: no lucide icon in the summary,
because the other `<details>` disclosures in the sheets have none)**.

### Layout (390 px; at ≥ 1280 px it sits in the right-hand column of the sheet's page, as the provider field does)

```
▸ Find a US provider
  ┌──────────────────────────────────────────────┐
  │ Only US doctors and clinics are listed. Your │
  │ search is sent to the US national provider   │
  │ registry (NPPES).                            │
  │ Search for  (•) A doctor  ( ) A clinic or practice
  │ Last name                [            ]      │
  │ First name (optional)    [            ]      │   ← "A doctor"
  │ Clinic or practice name  [            ]      │   ← "A clinic or practice"
  │ State                    [ Choose a state ▾ ]│
  │ City (optional)          [            ]      │
  │ ZIP code (optional)      [     ]             │
  │                          [ Search ]          │
  │ ─────────────────────────────────────────────│
  │ Choose a provider                            │
  │ ( ) Avery Quill, MD                          │
  │     Pediatrics · 12 Pine St, Seattle, WA 98101 · 206-555-0100
  │ ( ) Quillfeather Pediatrics                  │
  │     Clinic/Center · 40 Elm Ave, Seattle, WA 98101
  │ There may be more. Add a first name, a city  │
  │ or a ZIP code to narrow the search.          │
  │                     [ Use this provider ]    │
  └──────────────────────────────────────────────┘
Doctor or clinic (optional)  [ Avery Quill, MD   ]
From the US provider registry, NPI 9…            ← after a pick
Place (optional)             [ 12 Pine St, …     ]
```

- The person fields and the clinic field switch with the "Search for" choice through CSS (`:has(:checked)`, as
  the outcome sheet's sections do), with and without JavaScript **(confirm: same mechanism as the outcome sheet,
  which the owner is also asked to confirm)**.
- Result rows are RadioGroup choices: the label is the name (with ", {credential}" when the registry has one), the
  description is "{specialty} · {address}" plus " · {phone}" when there is one. Rows are in the registry's order.
- "There may be more…" shows only when the registry sent its full 20.

### Words (fixed; accessible names in quotes)

- Summary: "Find a US provider".
- Note (always shown at the top of the open panel): "Only US doctors and clinics are listed. Your search is sent to
  the US national provider registry (NPPES)." **(confirm: the second sentence tells the person a third party sees
  what they type; the spec does not require it)**
- Legend "Search for"; options "A doctor" (default) and "A clinic or practice".
- Fields: "Last name", "First name (optional)", "Clinic or practice name", "State" (first option "Choose a state",
  then the state names in alphabetical order: the 50 states, "District of Columbia", "American Samoa", "Guam",
  "Northern Mariana Islands", "Puerto Rico", "U.S. Virgin Islands"), "City (optional)", "ZIP code (optional)".
- Buttons: "Search" (secondary), "Use this provider" (secondary). Save and Cancel stay as they are.
- Results legend: "Choose a provider".
- More: "There may be more. Add a first name, a city or a ZIP code to narrow the search."
- Empty: "No US providers match. Check the spelling, or type the doctor or clinic yourself."
- Unavailable (Alert, in the panel): "The provider search isn’t available right now. You can type the doctor or
  clinic yourself."
- Rate limited (Alert, in the panel): "Too many searches, try again in a minute. You can type the doctor or clinic
  yourself." (the sign-in page's "Too many attempts, try again in a minute" pattern)
- Field messages: "Enter a last name of 2 to 60 letters", "Enter a name of 2 to 60 letters", "Choose a state",
  "Enter a city of 2 to 60 letters", "Enter a 5-digit ZIP code", "Choose a provider" (Use this provider with no
  choice). A refused panel field does not raise the form's "Check the fields with a message." alert; its message
  sits on the field.
- After a pick, under "Doctor or clinic (optional)": "From the US provider registry, NPI {npi}". Changing the
  name removes it on save **(confirm: the only place the NPI is shown; the visit page does not show it)**.
- A refused `provider_npi` on save, on the "Doctor or clinic" field: "Search for the provider again, or type it
  yourself."

### Behaviour the look depends on

- Search and "Use this provider" submit the visit form to `?/find` and `?/pick`. Everything typed in the visit
  fields is kept, and nothing is saved until Save. With JavaScript the sheet stays open, with no toast.
- Enter in a visit field saves, as today. With JavaScript, Enter in a panel field searches **(confirm: without
  JavaScript, Enter anywhere saves; the person presses Search)**.
- Picking fills "Doctor or clinic" and "Place". Both stay editable.

### States

Panel closed; open and empty; a refused field; results (with and without "There may be more"); no results;
unavailable; rate limited; after a pick; the edit sheet of a visit with an NPI; search off (no panel).

### Acceptance

- axe clean per state in light and dark at 390 and 1280 px; targets ≥ 44 px; no horizontal scroll at 390 px and
  200 % zoom; long names and addresses wrap (`overflow-wrap:anywhere`).
- Every state above works with JavaScript disabled.
- Visual baselines created from the build: panel open with results, after a pick, unavailable.
