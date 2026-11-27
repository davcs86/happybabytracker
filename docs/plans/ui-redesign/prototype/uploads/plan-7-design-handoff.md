# happybabytracker — design handoff: what plan 7 added (milestones, PDF export)

Audience: **Claude Design**, a design agent with **no access to this repository or to the session that built
it**. Treat this document as your only source. It is self-contained for plan 7.

**This file supersedes nothing.** It describes only the delta of plan 7 on top of the plans 3–5 handoff
(`plans-3-4-5-design-handoff.md`: trackers, timeline, growth, vaccination, navigation, tokens), the plan 5.1
handoff (`plan-5-1-design-handoff.md`: advanced bottle, nursing and pumping forms) and the plan 6 handoff
(`plan-6-design-handoff.md`: sickness, medication, reminders, notifications). Where this file says "the home",
"the kid links", "the quick-log tiles" or "the Modules page" it means those screens as the earlier handoffs
describe them, changed only as stated here. Section 0 repeats the ground rules you need.

| | |
|---|---|
| What | A **Milestones page** showing CDC's twelve "Learn the Signs. Act Early." checklists (2 months to 5 years, 159 items, CDC's words verbatim), each item markable as **reached** with a date (one tap = today), plus **custom milestones** (free text + date); and a **PDF export** of one child for up to 93 days and a chosen set of modules, rendered by the server, downloaded, never stored. **The PDF itself is a designed surface** (appendix C) |
| State (2026-11-24) | Built and merged to `main` (PR #10, merge commit `88654ec`). API and UI ship together. Migration `0007_milestones.sql` adds one table. The UI e2e suites were not run locally for this plan (CI ran them on the PR) |
| Scope of a redesign | The Milestones page (twelve lists, the reached state, the custom section, the off note, the after-5-years view), the mark/add/edit pages, the Export page, two new kid links, one new quick-log tile, and the **PDF layout** within the writer's capabilities (text, lines, rectangles; appendix C.4). CDC's item text, CDC's attribution and screening sentence are fixed copy, not UI |

One-line version: the app **lists every CDC milestone so a parent can mark it, and prints only what was
marked**; it never compares a child with an age or says anything that judges. The PDF **records and displays**
what the app holds, in the requesting parent's units and the child's time zone, and nothing else.

---

## 0. Ground rules (short form) and status legend

- **Product.** Private, self-hosted family health record for babies and kids; phones on the family's private
  network; installable PWA; online only (no offline page, owner); English only; no photos. It **records and
  displays; it never diagnoses, advises or judges.**
- **Stack.** SvelteKit 5 (runes), server-rendered, Node adapter; the browser talks only to the SvelteKit
  server. UI = **shadcn-svelte** (style `nova`, base colour `neutral`, lucide icons), bits-ui, Tailwind 4,
  Geist Variable. Installed shadcn parts: `alert`, `button`, `card`, `input`, `label`; everything else
  (native `<details>`, checkboxes, date inputs, text links) is plain Tailwind. Adding registry components is
  welcome. Buttons: default `h-8` (32 px), `sm` `h-7` (28 px); inputs `h-8`.
- **No-JS baseline.** Every form is a real `<form method="POST">`; **every action works with JavaScript
  disabled** (the one app-wide exception is plan 6's "Turn on notifications on this device"; plan 7 adds none).
  Show/hide without JS is native `<details>` or CSS `:has()`. For every interaction you propose, say what the
  no-JS fallback is.
- **Mobile first.** 390 px, no horizontal scroll even with a 200-character unbroken custom milestone or note;
  one nav (bottom bar < 768 px, left column ≥ 768 px: Home, Timeline, Growth, Account; **unchanged by plan 7**);
  touch targets ≥ 44 px is the goal (several plan 7 controls miss it: section 6).
- **Accessibility.** axe-clean at 390×844 and 1280×800 in light and dark; colour never the only signal; errors
  `role="alert"` with the field `aria-invalid` and `aria-describedby`; accessible names are matched by the e2e
  tests (appendix B). Change one only on purpose, and say so.
- **Never judge** (section 8): a fixed denylist of words, nothing coloured by state, no count of unreached
  items, no progress indicator. A test asserts that the Milestones page contains no `svg`, `img`, `progress` or
  `meter` element.
- **Tokens** are greyscale CSS variables in `oklch`; propose new values for the existing names. Light/dark
  follows the OS. The PDF is **greyscale only** (appendix C).

**Status legend used below**

| Tag | Meaning |
|---|---|
| **Built** | What the code at `88654ec` does; copy quoted from the code |
| **Owner** | An owner decision; do not reopen (you may propose, labelled as an owner reversal) |
| **(a)** | A rule the plan author or the build controller set; changing it has a stated cost |
| **Open** | Not decided; design with the stated assumption and label it |
| **Unverified** | Could not be checked from the code (real devices, OS rendering, PDF viewers, printers) |

---

## 1. Plan 7 at a glance

### 1.1 What it adds, in product terms

| Area | What a parent can do | Where |
|---|---|---|
| Milestone checklists | See all twelve CDC lists ("By 2 months" … "By 5 years"), each a collapsible section grouped by CDC's four domains; the list for the child's age is open and marked "Current list for this age" | `/kids/{kid}/milestones` (kid link "Milestones"; quick-log tile "Milestones" while the module is on) |
| Mark reached | One tap **Reached** records the child's local **today**; "Choose a date" opens a form for an earlier date and a note | Milestones page; `/kids/{kid}/milestones/new?item={id}` |
| Change a mark | Edit its date or note; **Undo reached** returns the item to its list | `/kids/{kid}/milestones/{entry}` |
| Custom milestones | Add free text (up to 200 characters) with a date and note; edit; delete | "Your milestones" → "Add a milestone" → `/kids/{kid}/milestones/new` |
| After the 5th birthday | The lists are gone; the page lists what was marked ("Reached"), retired items, and custom milestones | same page |
| PDF export | Choose a first and last date (≤ 93 days, last not in the future) and the modules to include (switched-on ones pre-ticked); press **Download PDF**; the browser downloads `happybabytracker-{from}-{to}.pdf` | `/kids/{kid}/export` (kid link "Export PDF") |
| The PDF | Profile, then one section per chosen module in a fixed order; footer on every page | Appendix C |

**Milestones are never on the timeline or the home's Due list** (Built, plan Open value 18): they are dates,
not instants, and nothing about a milestone is ever due.

### 1.2 The guiding principle

The app **shows every item so it can be marked, and shows a mark only as a date**. It:

- shows CDC's item text exactly as CDC wrote it (pronouns included: "by herself", "his spoon");
- distinguishes an unreached item from a reached one **only** by the absence of the line "Reached on {date}";
  no colour, icon, checkmark, count, percentage, progress bar, "not yet" text;
- marks the list for the child's age in **text** ("Current list for this age"), never with colour alone;
- never shows the child's age at the time a milestone was reached (an age beside a checklist age invites
  comparison; plan Open value 15);
- keeps CDC's screening sentence and attribution **visible next to the lists**, never folded away;
- in the PDF, prints **only what was marked**, never the unreached items or their number.

### 1.3 Roles (who can do what)

| Action | Guardian | Caregiver | Admin |
|---|---|---|---|
| See the Milestones page, every list | yes | yes | never sees any child |
| Mark, edit, undo; add, edit, delete custom milestones | yes | yes | — |
| See the off note "Milestones are off for {kid name}." | yes, **with** the link "Turn it on in Modules" | yes, sentence only | — |
| Switch the `milestones` module | yes (Modules page) | no | — |
| Open the Export page, download a PDF, tick any module (switched off included) | yes | yes | redirected to `/admin/parents` |

There is **no role difference** on the milestone and export controls. A request for a child with no grant, an
unknown child or another child's milestone row shows the **same "Not found" page** (section 2.6).

---

## 2. Data model and business rules

### 2.1 What is stored (Built)

```
kids ──< milestones_reached   catalog_item? | custom_text?  (exactly one), reached_on (YYYY-MM-DD, kid-local),
                              note?, created/updated by and at
```

One new table (migration `0007_milestones.sql`). One row per reached catalog item per child (unique), or one
row per custom milestone. Hard deletes, no edit history. The **catalog itself is not in the database**: it is a
CSV embedded in the API (159 items), with stable ids such as `2m-se-calms-down-when` and a provenance header.

**The PDF is never stored**: rendered on request, streamed to the browser, `Cache-Control: no-store`, never
logged beyond route, child id, status and duration.

### 2.2 What each screen can show (API views, abbreviated)

- **Milestones view:** `{catalog:{name, authority, edition, retrieved, origin_url, licence, attribution,
  screening_note}, age:{chronological_months, corrected_months|null, basis_months}, catalog_shown,
  checklists:[{age_months, position: earlier|current|next|later, items:[{id, domain, text,
  reached:{id, reached_on, note}|null}]}], custom:[{id, text, reached_on, note}], retired:[{id, item, text,
  reached_on, note}], reached:[…]}`.
  - `catalog_shown` true (before the 5th birthday): `checklists` holds **all twelve**, in age order, items in
    CDC page order; `reached` is `[]`.
  - `catalog_shown` false: `checklists` is `[]`; `reached` lists the reached catalog rows `{id, item,
    age_months, domain, text, reached_on, note}`.
  - The answer is the same whether the module is on or off.
- **Milestone entry:** `{id, item|null, text, reached_on, note}` (`text` is CDC's text for a catalog row).
- **Export:** `POST` body `{from, to, modules[]}` → `200 application/pdf`, `Content-Disposition: attachment;
  filename="happybabytracker-{from}-{to}.pdf"` (dates only, never a name), `no-store`, `nosniff`.

### 2.3 Milestone rules

| # | Rule | Source |
|---|---|---|
| M1 | **Catalog:** all twelve CDC lists, 2 months to 5 years, transcribed verbatim from CDC's HTML age pages (retrieved 2026-11-24, "2022 revision, web pages updated May 16, 2026"). 159 items: 11, 13, 12, 13, 10, 13, 15, 13, 15, 12, 17, 15 in age order. Shortest item "Runs" (4 characters); longest 143 characters ("Follows directions given with both a gesture and words. For example, he gives you a toy when you hold out your hand and say, “Give me the toy.”") | **Owner** 2026-11-23; `api/internal/milestones/data/cdc.csv` |
| M2 | **List titles** are CDC's age labels: "By 2 months", "By 4 months", "By 6 months", "By 9 months", "By 1 year", "By 15 months", "By 18 months", "By 2 years", "By 30 months", "By 3 years", "By 4 years", "By 5 years" | `ui/src/lib/milestones.ts` `CHECKLIST_TITLE` |
| M3 | **Domains** (CDC's headings without "Milestones"), in this order inside each list: "Social/Emotional", "Language/Communication", "Cognitive (learning, thinking, problem-solving)", "Movement/Physical Development". Every list has every domain (1 to 6 items each) | `DOMAIN_LABEL`, `MILESTONE_DOMAINS` |
| M4 | **Age basis:** the **corrected** age for a child born before 37 weeks 0 days, until the 2nd birthday; chronological after. Both ages are shown in the page header wherever the corrected one applies ("3 months 4 days (corrected 1 month 2 days)") | (a) spec 7.1, plan Open value 13; CDC's "more than 3 weeks prematurely" |
| M5 | **Current list:** the largest list age ≤ the basis in whole months (CDC: "use the checklist for the younger age"). **Next list:** the smallest list age > the basis. **Before 2 months** (basis 0 or 1, including a preterm child by corrected age) nothing is current; the 2-month list is open and marked "Next list", and the top link reads "Go to the next list" | (a) Open value 14; **Owner** (label "Go to the next list" before 2 months) |
| M6 | **Catalog span:** all twelve lists while the child's local today is before the **5th birthday** (chronological, civil date; a 29 February birth → 1 March). From the 5th birthday: no lists; a "Reached" section lists the reached catalog rows | **Owner** 2026-11-23 (answer 5); `MilestoneCatalogUntilYears` 5 |
| M7 | **Reached date:** kid-local `YYYY-MM-DD`, **not after today, not before the birth date**. One-tap "Reached" records today (the child's zone) | Open value 15 |
| M8 | An item can be reached **once** (a second mark: "Already marked as reached"). The item of a row never changes; only its date and note | Open value 15; API 409 `already_reached`, 422 `item` |
| M9 | **Custom milestone:** text 1–200 characters after trimming, a date, an optional note; there is no unreached custom milestone | Open value 16; `MILESTONE_TEXT_MAX` 200 |
| M10 | Note: up to 500 characters (shared limit) | `NOTE_MAX_LEN` |
| M11 | **Cap:** 1000 marks per child, catalog and custom together ("This child has the maximum number of recorded milestones") | Open value 17; `MaxMilestonesPerKid` |
| M12 | **Undo** (catalog row) and **delete** (custom row) are hard deletes; an undone item goes back to its list as unreached | Open value 15 |
| M13 | **Retired items:** a later catalog update may retire an item (never delete it); a reached row of a retired item stays listed with its text under "Reached, no longer in the lists". The shipped catalog has **no** retired item today | Open values 9, 10 |
| M14 | **No age at reaching** is shown, ever; only the date | Open value 15 |
| M15 | Not on the timeline, not on the home Due list | Open value 18 |

### 2.4 Module rules (Milestones)

| # | Rule | Source |
|---|---|---|
| D1 | Module default **unchanged**: on at 0–11 and 12–23 months, **off from 24 months** | **Owner** 2026-11-23 (answer 5) |
| D2 | Whether it stays on after 24 months is the **guardian's** decision on the existing Modules page; no new setting | **Owner** |
| D3 | Plan 3's band rule: a milestone **recorded** in the 60 days before the 24-month band starts keeps the module on. So **the first mark made for a child past 24 months turns the module on** (the off note disappears, the quick-log tile appears) | (a) Open value 27; ruling T5-1 |
| D4 | **Module off:** the page stays reachable (kid link always shown, every role), shows the full content under the one note "Milestones are off for {kid name}."; a guardian also gets "Turn it on in Modules". No reminders wording (milestones have none). The note is advisory: if the modules lookup fails, no note shows | **Owner** (ruling R2, owner may veto) |
| D5 | Quick-log tile "Milestones" shows only while the module is on | Open value 26 |

### 2.5 Export rules

| # | Rule | Source |
|---|---|---|
| E1 | **Range:** two kid-local dates, inclusive; `from` not before the birth date; `to` not before `from`, not after today, **at most 93 days** in all | **Owner** 2026-11-23; `EXPORT_MAX_DAYS` 93 |
| E2 | **Defaults:** `to` = today; `from` = today − 29 days (the last 30 days), but never before the birth date | (a) Open value 25, P-31; `EXPORT_DEFAULT_DAYS` 30 |
| E3 | **Module filter:** one box per module, all eleven (bottles, nursing, pumping, diapers, sleep, feeding, milestones, growth, vaccination, sickness, medication). Switched-on modules are ticked; switched-off ones are listed **unticked** under "Switched off for this child" and **exported when ticked** (a switch never hides or deletes data). None ticked, or an unknown id → refused | **Owner** 2026-11-23 |
| E4 | **Order:** the PDF holds the profile, then exactly the chosen sections, always in the module order above, whatever order they were sent in. An unchosen module leaves no heading, no attribution, no row | (a) Global Constraints |
| E5 | **Range applies to trackers only.** Bottles, nursing, pumping, diapers, sleep, feeding, sickness, medication doses: entries inside the range. Growth, vaccination, milestones are **records**: everything up to the range's last date | (a) accepted default, Open value 5 |
| E6 | **Vaccination statuses** are evaluated **as of the range's last date** (never later than today), printed as "Statuses as of {date}." | (a) ruling T13-2; owner may revert |
| E7 | Values in the **requesting** parent's units (kg/lb, cm/in, ml/fl oz, °C/°F), from the stored column, never converted; times in the **child's** zone | Open value 24 |
| E8 | **Size caps:** 3000 rows per section, 6000 in all; a section cut short ends with one overflow line; a file over 4 MiB is refused (never truncated) | (a) Open value 22 |
| E9 | **One render at a time**; a second request at the same moment is refused as busy (never queued); up to 2 finished files download at once, a third is refused before any byte | (a) Open value 22, item 29 |
| E10 | **Every non-field failure** (busy, server unavailable, timeout after 25 s, network) shows **one** message: "The PDF cannot be made right now, try again in a moment" | **Owner** item; Open: a separate "unavailable" message needs spec 13.3 to name it |
| E11 | **Privacy:** range and modules travel in the POST body, never a URL, except the validation redirect (`error`, `from`, `to`, `modules=a,b`: enumerated values only). The file name and PDF metadata hold no name | (a) P2 (spec 13.3 amendment; owner may veto, cost: the form forgets the ticked boxes after an error) |
| E12 | **Characters:** Latin (with extended forms), Greek and Cyrillic print as typed (embedded Noto Sans 2.015, OFL 1.1); anything else (CJK, Arabic, Hebrew, Indic, emoji) prints as `?`, and the profile then says so once | **Owner** 2026-11-23 |

### 2.6 Identity and privacy

- No grant, an unknown child, another child's milestone row and a malformed id all show the **same "Not
  found" page**. A direct export POST by a parent without a grant also ends on that page.
- The admin never sees a child: the export page and its POST redirect the admin to `/admin/parents`.
- Logs never carry a custom milestone text, a note, a date range or the module list.

---

## 3. Flows (step by step, with branches)

1. **See the current list.** Kid home → link **Milestones** (under the child's name) or quick-log tile
   **Milestones** → page "Milestones for {child}", the age line under it ("9 months 3 days"), the link
   **Go to the current list**, then twelve sections; "By 9 months · Current list for this age" is open, the
   other eleven closed.
   - No JS: identical (server-rendered `<details>`; the link is an in-page anchor `#list-9`).
2. **Open another list.** Tap a closed title (e.g. "By 3 years") → it opens (native `<details>`, no JS
   needed). Several lists can be open at once.
3. **One-tap reached.** In an open list, under an item's text: button **Reached** → the item is recorded for
   today (child's zone) → the page reloads at `?open={item}#item-{item}`; the row now reads "Reached on
   24 November 2026" and an **Edit** link; the button and "Choose a date" are gone.
   - JS: the form is enhanced; a list the reader opened stays open across the reload (its open state is set
     once). Where the viewport lands after the reload is **Unverified** (section 7.2).
   - No JS: a full POST, 303 to `?open={item}#item-{item}`; the server renders that item's list open and the
     browser jumps to the item.
   - Already marked meanwhile (another parent) → "Already marked as reached" **in that item's row**
     (`role="alert"`). Cap reached → "This child has the maximum number of recorded milestones" in the row.
     Server down → "The app cannot reach its server, try again soon" / "Something went wrong, try again" in
     the row.
4. **Reached on an earlier date.** Item → link **Choose a date** → page "Mark as reached": CDC's text, "Date
   reached" (prefilled today; min the birth date, max today), "Note" → **Save** → back to the Milestones page
   (the list the item is in is not forced open on this path) → "Reached on {date}".
   - Future date or before birth → under "Date reached": "Enter a date that is not in the future and not
     before the birth date"; typed values kept.
   - Item no longer in the lists (retired by a catalog update, or the 5th birthday passed) → the page shows the
     custom form instead (an unknown `?item=` falls back to "Add a milestone").
5. **Edit or undo a mark.** "Reached on …" → **Edit** → page "Edit milestone": CDC's text, "Date reached",
   "Note"; **Save** → back. `<details>` **Undo reached** → "This removes the date for everyone with access to
   {child}." → **Yes, undo reached** → back; the item is unreached in its list again.
6. **Add a custom milestone.** Section "Your milestones" → **Add a milestone** → page "Add a milestone":
   "Milestone" (text), "Date reached" (today), "Note" → Save → back; the row shows the text, "Reached on
   {date}", Edit.
   - Blank or over 200 characters → "Enter a milestone of up to 200 characters".
7. **Edit or delete a custom milestone.** Edit → the text is editable too → Save. `<details>` **Delete this
   milestone** → "This removes the milestone for everyone with access to {child}." → **Yes, delete this
   milestone**.
8. **A child younger than 2 months.** The top link reads **Go to the next list**; "By 2 months · Next list" is
   open; nothing is marked current. Every item can still be marked.
9. **A preterm child.** Born at 32 weeks, 3 months chronological, 1 month corrected: the header reads "3 months
   … (corrected 1 month …)"; "By 2 months · Next list" is open (flow 8 applies by corrected age). At 5 months
   chronological "By 2 months" is current. From the 2nd birthday the basis is chronological.
10. **Between list ages.** At 14 months "By 1 year" is current (CDC's "younger age" rule), "By 15 months" is
    the next.
11. **The 5th birthday.** From that kid-local date the lists disappear: no "Go to…" link, no list titles; a
    section **Reached** lists each reached catalog row (CDC text, "{list title} · {domain}" muted, "Reached on
    {date}", Edit) or "No milestones recorded"; then retired rows, CDC's sentence and attribution, and "Your
    milestones". Edits and undo still work.
12. **Module off (child past 24 months).** The kid link still leads to the page; under the age line: "Milestones
    are off for {child}." (guardian: + "Turn it on in Modules"). Full catalog below; marks work. The first mark
    turns the module on: on the next load the note is gone and the quick-log tile appears.
13. **Export a PDF.** Kid link **Export PDF** → page "Export PDF for {child}", the sentence "The file is made
    when you download it and is not kept.", **From** / **To** (last 30 days), fieldset **Modules to include**
    (switched-on modules ticked; switched-off ones unticked under "Switched off for this child") → adjust →
    **Download PDF** → the browser downloads `happybabytracker-{from}-{to}.pdf`; the page itself does not
    change.
    - JS and no JS behave the same: the form is **not** enhanced; it is a plain POST to `/kids/{kid}/export/pdf`
      answered either by the file (attachment) or by a 303 back to the page.
    - The render can take up to 25 s; nothing on the page shows progress (section 6).
14. **Export refused.** The route redirects to `/kids/{kid}/export?error={field|busy}&from=…&to=…&modules=a,b`;
    the page re-fills both dates and re-ticks exactly the boxes sent, and shows:
    - `from`: under From, "Enter a start date that is not before the birth date";
    - `to`: under To, "Enter an end date from the start date to today, at most 93 days in all";
    - `modules`: under the fieldset, "Choose at least one module" (every box gets `aria-invalid`);
    - `busy`: at the top, "The PDF cannot be made right now, try again in a moment".
    - The browser's own date validation may block the submit first (the form has `min`, `max`, `required` and
      no `novalidate`; how each browser shows that is **Unverified**).
15. **Tick a switched-off module.** Pumping switched off → "Pumping" unticked under "Switched off for this
    child" → tick it → its section is in the PDF.
16. **Caregiver.** Same page, same boxes, same result. **No grant**: the page and a direct POST end on "Not
    found". **Admin**: redirected to `/admin/parents`.
17. **Open the PDF on a phone.** What the phone does with the downloaded file (viewer, Files, share sheet, an
    installed PWA on iOS) is the browser's and OS's business: **Unverified** on every real device.

---

## 4. Screens as built

All new kid pages share the earlier shell: a plain underlined link back (the child's name, or "Milestones for
{child}" on the mark/add/edit pages), an `h1`, one narrow column (`max-w-md`) for forms. HTML titles follow
"{Page} · {child} · happybabytracker".

### 4.1 Milestones page (`/kids/{kid}/milestones`)

**Who:** guardian and caregiver. **Entry:** kid link "Milestones" (every role, module on or off), quick-log tile
"Milestones" (module on). **Title:** "Milestones · {child} · happybabytracker".

Layout, top to bottom:

| # | Region | Content (Built) |
|---|---|---|
| 1 | Back link | "{child}" (to the kid home) |
| 2 | `h1` | "Milestones for {child}" (wraps anywhere) |
| 3 | Age line (muted) | the home's age header: "{age}", "{age} (corrected {age})" or "{age} (before expected term)" |
| 4 | Off note (module off only) | plain `text-sm` paragraph: "Milestones are off for {child}." + guardian-only underlined link "Turn it on in Modules" |
| 5 | Page alert | `role="alert"`, only for a failure not tied to a listed item |
| 6 | Jump link (lists shown) | "Go to the current list" (anchor `#list-{age}`), or "Go to the next list" before the first list age |
| 7 | Section "Milestone checklists" (`h2`) | twelve `<details>` (bordered, rounded, padding 12 px), one per list (4.1.1) |
| 7' | Section "Reached" (`h2`; **after the 5th birthday, instead of 6–7**) | rows of reached catalog items or "No milestones recorded" |
| 8 | Section "Reached, no longer in the lists" (`h2`; only when such rows exist) | rows (4.1.2) |
| 9 | CDC block (always, `text-sm`, never folded) | the screening sentence, then the attribution (appendix A); a test checks both sit **below the last list title** |
| 10 | Section "Your milestones" (`h2`) with link "Add a milestone" on the same row (right, wraps) | rows, or "No milestones added" (muted) |

#### 4.1.1 One checklist (`Checklist.svelte`)

- `<details id="list-{age}">`. **Summary**: "{title}" or "{title} · Current list for this age" / "{title} · Next
  list" (medium weight, pointer cursor). Open when it is the marked list, or when it holds the item the page
  returns to (`?open=`). Its open state is set **once** per page load, so a data reload after a mark never
  closes a list the reader opened.
- Inside, per domain in M3 order: `h3` (small, semibold) with the domain label, then a list of items. Each item
  (`id="item-{item id}"`, `text-sm`, wraps anywhere):
  - **Unreached:** the CDC text; below it, in one wrapping row: outline button **Reached** (32 px; accessible
    name "Reached {item text}") and the underlined link **Choose a date** (accessible name "Choose a date for
    {item text}").
  - **Reached:** the CDC text; below it "Reached on {d Month yyyy}" (e.g. "Reached on 4 March 2026") and the
    underlined link **Edit** (accessible name "Edit {item text}").
  - **Row error** (a refused one-tap for this item): `role="alert"` line under the text, destructive colour.
- **Nothing else** distinguishes the two states: same type size, same weight, no icon, no tint, no checkmark.

#### 4.1.2 Reached / retired / custom rows (one snippet)

A bordered rounded box per row (padding 12 px, `text-sm`, wraps anywhere): the text; for the after-5 "Reached"
section only, a muted line "{list title} · {domain}" (e.g. "By 5 years · Cognitive (learning, thinking,
problem-solving)"); then "Reached on {date}" and **Edit** (accessible name "Edit {text}").

States:

| State | What shows |
|---|---|
| Child 9 months | link "Go to the current list"; "By 9 months · Current list for this age" open; 11 closed |
| Child < 2 months, or preterm with corrected basis < 2 | link "Go to the next list"; "By 2 months · Next list" open; the text "Current list for this age" appears nowhere |
| Child 4 years 11 months | "By 4 years · Current list for this age"; "By 5 years" closed and markable |
| From the 5th birthday | no jump link, no lists; "Reached" section; "No milestones recorded" when none |
| Reached item | "Reached on {date}" + Edit instead of the button and link |
| No custom row | "No milestones added" |
| Retired row exists | section "Reached, no longer in the lists" (none today) |
| Module off | note + (guardian) link; everything else unchanged and usable |
| Module on | no note |
| One-tap refused | message in the item's row; other failures at the top |
| Catalog failed to load on the server | the generic error page (503 "The app cannot reach its server right now", "Try again in a moment.") |
| No grant / unknown child | "Not found" page |
| No JS | identical; `<details>` native; one-tap is a full POST with a 303 back to the item |
| 390 px | every text wraps anywhere; a 200-character unbroken custom text causes no horizontal scroll (tested) |
| Loading | no spinner; plain page loads |

Accessibility: lists are named by their summaries; the domain headings are `h3`; sections are regions named by
their `h2` ("Milestone checklists", "Reached", "Reached, no longer in the lists", "Your milestones"); the button
and links carry the item text in a visually hidden suffix.

### 4.2 Mark as reached / Add a milestone (`/kids/{kid}/milestones/new[?item={id}]`)

`h1` "Mark as reached" (with `?item=` naming a listed item) or "Add a milestone". Back link "Milestones for
{child}". One form (`novalidate`, `max-w-md`):

| Region | Control | Label / text |
|---|---|---|
| Page alert | `role="alert"` | refusals not tied to a field (and an `item` refusal) |
| Item (mark only) | plain paragraph | CDC's text; a hidden field carries the id |
| Text (add only) | text, required | "Milestone" |
| Date | `date`, required, min birth date, max today, prefilled today | "Date reached" |
| Note | text | "Note" |
| Actions | button + link | "Save", "Cancel" (to the Milestones page) |

Save → 303 to the Milestones page. No JS: identical. Field errors under the field (appendix A).

### 4.3 Edit milestone (`/kids/{kid}/milestones/{entry}`)

`h1` "Edit milestone"; back link "Milestones for {child}". Catalog row: CDC's text as a paragraph, then "Date
reached", "Note". Custom row: "Milestone" (editable), "Date reached", "Note". Save / Cancel. Only changed fields
are sent. Then a bordered `<details>`:

| Row kind | Summary | Body | Button |
|---|---|---|---|
| Catalog | "Undo reached" | "This removes the date for everyone with access to {child}." | "Yes, undo reached" (outline, destructive border and text) |
| Custom | "Delete this milestone" | "This removes the milestone for everyone with access to {child}." | "Yes, delete this milestone" |

A failed delete reopens the details with its message inside.

### 4.4 Export page (`/kids/{kid}/export`)

**Who:** guardian and caregiver. **Entry:** kid link "Export PDF". **Title:** "Export PDF · {child} ·
happybabytracker".

| # | Region | Content (Built) |
|---|---|---|
| 1 | Back link | "{child}" |
| 2 | `h1` | "Export PDF for {child}" |
| 3 | Sentence (muted, small) | "The file is made when you download it and is not kept." |
| 4 | Busy alert (after a refusal) | `role="alert"`: "The PDF cannot be made right now, try again in a moment" |
| 5 | Dates (2-column grid, also at 390 px) | `date` "From" (min birth date, max today) and `date` "To" (same bounds); each with its error under it |
| 6 | Fieldset, legend "Modules to include" | 2-column grid of checkbox rows (each label row ≥ 44 px high, 16 px box, module label); switched-on modules, ticked |
| 7 | Inside the fieldset, only if any module is off | a group labelled by `h2` "Switched off for this child", the same 2-column grid, unticked |
| 8 | Modules error | under the fieldset, `role="alert"`: "Choose at least one module"; every box `aria-invalid` and described by it |
| 9 | Button | "Download PDF" (default, 32 px, left-aligned) |

Module labels (the app's `MODULE_LABEL`): "Bottle", "Nursing", "Pumping", "Diaper", "Sleep", "Feeding",
"Milestones", "Growth", "Vaccination", "Sickness", "Medication", in that order.

States:

| State | What shows |
|---|---|
| Default | last 30 days (from the birth date for a younger child); switched-on ticked; no off group when every module is on |
| Some modules off | off group with its heading, unticked |
| After a refusal | dates and ticks from the redirect; the field message (or the busy alert) |
| Download in progress | **nothing changes** on the page; the button stays enabled; the browser's own download UI only (Unverified per browser) |
| Success | file downloaded; page unchanged; no confirmation text |
| No grant | "Not found" page (page and POST) |
| Admin | redirected |
| No JS | identical (the form is never enhanced) |
| 390 px | no horizontal overflow (tested) |

### 4.5 Kid links and quick-log

**Under the child's name** (`KidLinks`, a wrapping row of small underlined links): "Vaccines", "Medication",
**"Milestones"**, **"Export PDF"** (both new, every role), then for guardians "Settings", "Access", "Modules".
Each carries " for {child}" visually hidden.

**Quick-log tile "Milestones"** (label only; opens the Milestones page), shown while the module is on, placed by
the existing rule (band-first modules, then module order):

| Band | Tiles with the default modules |
|---|---|
| 0–11 months | Bottle, Nursing, Diaper, Sleep, Pumping, Feeding, **Milestones**, Growth, Vaccination, Sickness, Medication |
| 12–23 months | Feeding, Bottle, Sleep, Diaper, Nursing, Pumping, **Milestones**, Growth, Vaccination, Sickness, Medication |
| 24 months + | Sickness, Medication, Growth, Vaccination (+ **Milestones** last when a guardian keeps it on or after a first mark) |

### 4.6 The PDF

A designed surface of its own: appendix C.

---

## 5. States matrix and what changed in existing screens

### 5.1 States across the new screens

| Condition | Milestones page | Mark / add / edit | Export page | PDF |
|---|---|---|---|---|
| Module off | note "Milestones are off for {child}." (+ guardian link); full content; no tile | unchanged | the module listed unticked under "Switched off for this child" | its section only when ticked |
| Empty (nothing marked) | lists with buttons; "No milestones added" | — | — | "No milestones recorded"; trackers "No entries in this range." |
| After the 5th birthday | "Reached" section, no lists | `?item=` falls back to "Add a milestone" | unchanged | unchanged (marked rows only) |
| Field error | in the item row (one-tap) | under the field | under From / To / the fieldset | — |
| Server failure | row or page alert; catalog failure: error page | page alert | the single busy message | — (never a partial file) |
| 404 (no grant, unknown child, other child's row) | "Not found" | "Not found" | "Not found" | not produced |
| Caregiver vs guardian | identical except the off note's link | identical | identical | identical |
| No JS | identical | identical | identical | — |

### 5.2 What changed in existing screens and rules

| Existing screen / rule | Change |
|---|---|
| Kid links under the name | Two new links, "Milestones" and "Export PDF", for every role, after "Medication" |
| Home quick-log | New tile "Milestones" (module on) |
| Modules page | `milestones` now has an entry point; default per band unchanged (off from 24 months); a first mark past 24 months stores it as on |
| Home Due list, timeline, daily totals | Unchanged: no milestone, no export entry |
| Navigation bar | Unchanged (Home, Timeline, Growth, Account) |
| Delete a child | Also removes its milestone rows (wording unchanged) |
| Spec 13.3 (what URLs may carry) | Gains the export redirect's `error` and `modules` (owner may veto) |

---

## 6. Known design gaps (honest audit; evaluate and add your own)

1. **Small targets.** "Reached" is 32 px high; "Choose a date" and "Edit" are inline text links (about one line
   of small text); a list summary is one line of medium text inside a 12 px-padded box (its tap height is
   **Unverified**, likely under 44 px); the kid links are small text links in a wrapping row; "Download PDF",
   "Save" and the date inputs are 32 px. The export checkbox rows are the exception: each label row is ≥ 44 px.
2. **A long page.** Twelve lists, 159 items, each unreached one with a button and a link (up to 159 forms).
   The only navigation is one jump link to the marked list; there is no list index, no "back to top", no way
   to collapse all.
3. **No feedback after a one-tap** beyond the row changing to "Reached on {date}"; no `role="status"` line.
   One tap records **today**; a parent marking something seen last week has to know to use "Choose a date".
4. **Viewport after a one-tap mark with JavaScript** (the enhanced redirect to `#item-{id}`): whether the page
   keeps the reader's position, jumps to the item, or jumps to the top is **Unverified**; without JS the
   browser jumps to the item.
5. **The jump link targets a `<details>`**: if the reader closed the marked list, following "Go to the current
   list" scrolls to a closed title (whether the browser opens it is **Unverified**).
6. **Three date styles**: the page says "Reached on 4 March 2026"; the date input shows the browser's own
   format; the PDF prints "2026-03-04" (and growth rows "Wed 4 Mar 2026").
7. **"Choose a date" path loses the place**: after Save it returns to the page top, not to the item's list
   (unlike the one-tap).
8. **After the 5th birthday** the custom rows stay under "Your milestones", below the CDC block, rather than
   inside "Reached" (the plan's text put them after the reached rows; the build keeps two sections).
9. **Export has no progress state.** The render can take up to 25 s; the button stays enabled and nothing on
   the page changes; a second tap during a render may come back as the busy message.
10. **No confirmation after a download**: the page looks the same before and after.
11. **Browser validation on the export dates.** The form has `min`/`max`/`required` and no `novalidate`, so a
    browser may show its own bubble before the app's message; appearance and wording are the browser's
    (**Unverified**). The Milestones forms do use `novalidate`.
12. **One message for every non-field failure** ("busy"), including a timeout or a server that is down (owner
    item; section 7.1).
13. **Singular vs plural:** the form says "Bottle", "Diaper"; the PDF headings say "Bottles", "Diapers".
14. **PDF order is mixed**: tracker entries, growth rows and vaccine doses newest first; day totals by date
    ascending; milestones, symptoms and medication doses oldest first.
15. **PDF small print**: "No entries in this range.", the overflow lines, growth and vaccination attributions
    and the character notice are 7 pt; CDC's two sentences are 9 pt by rule.
16. **"No milestones recorded"** has no full stop (as on screen), while "No measurements recorded." and "No
    vaccinations recorded." do.
17. **Footer zone** prints the IANA id ("(America/New_York)"), not a friendly name.
18. **Mobile download** (iOS Safari, an installed PWA, Android Chrome): **Unverified**.

---

## 7. Open questions

### 7.1 Owner items (record as decided or as owner-vetoable; do not reopen)

| Item | As built | Tag |
|---|---|---|
| PDF library | `codeberg.org/go-pdf/fpdf` v0.12.0, pinned | **Owner** |
| Catalog | all twelve CDC lists, 2 months to 5 years, verbatim | **Owner** |
| Export range and filter | ≤ 93 days; one box per module; empty or unknown selection refused (API 422 `modules`) | **Owner** |
| Font | embedded Noto Sans 2.015 Regular + Bold (OFL 1.1); other scripts and emoji as `?` | **Owner** |
| Module default and span | default unchanged (off from 24 months); full catalog until the 5th birthday; guardian decides to keep it on | **Owner** |
| Module-off note | "Milestones are off for {kid name}." + guardian link (ruling R2) | **Owner** (may veto) |
| Vaccination statuses in the PDF | as of the end of the range ("Statuses as of {date}.") | (a) T13-2; owner may revert (one line) |
| Growth licence line in the PDF | the licence name and link only (text before the first "; ") | (a) T12-6 |
| Non-field export failures | all show the busy message | **Owner** item; **Open**: an "unavailable" message needs the spec 13.3 clause to name `error=unavailable` |
| First-list label before 2 months | "Go to the next list" / "Next list" | **Owner** |
| Spec 13.3 amendment (redirect carries `error`, `modules`) | applied | (a) P2; owner may veto (cost: the form forgets ticks after an error) |
| CDC screening sentence wording | the HTML pages' wording (two pages use straight quotes; the app shows the unquoted form) | owner reads it in the catalog review document |

### 7.2 Design questions (answer or label; tag each API change cheap / costly)

1. **Touch targets** of the one-tap "Reached" button, "Choose a date", "Edit" and the list summaries at
   390 px, without making 159 rows a wall of buttons (UI-only: cheap).
2. **Long-list navigation** across twelve lists: an in-page index of list titles (anchors, no JS), a "back to
   the current list" link inside each list, or another pattern that works without JavaScript (UI-only: cheap).
3. **After a one-tap mark**: where the viewport lands (JS and no JS), and whether a `role="status"` line names
   what was recorded (UI-only: cheap). Keep the reader's open lists open.
4. **Marking a past date** in fewer steps (for example a date input revealed per item through `<details>`);
   the one-tap "today" stays (UI-only: cheap).
5. **The export date inputs**: two native date fields side by side at 390 px; whether to show the range length
   or the 93-day bound next to them; whether to add `novalidate` so only the app's messages show (UI-only:
   cheap).
6. **Download feedback**: a no-JS-safe way to say a file is being made (the form cannot know when a native
   download finishes); with JS an enhanced fetch-and-save is possible but must keep the plain POST as the
   baseline (UI-only: cheap to costly, label it).
7. **Mobile download UX** (iOS PWA share sheet, Android downloads): **Unverified**; design for "the browser
   takes over" and label any device-specific assumption.
8. **PDF visual hierarchy** within appendix C.4: heading weight and spacing, table rules, zebra rows in grey,
   the profile block as a box, section breaks, chart framing.
9. **Placement of the CDC block** so it reads as part of the lists at 390 px and ≥ 1024 px (it may move, it may
   not be folded, shrunk to small print or separated from the lists).

---

## 8. Never-interpret rules (what you must NOT do)

- **No judgement words** anywhere the app writes its own words (screens, empty states, errors, PDF profile,
  milestone and growth sections): `late`, `behind`, `delayed`, `delay`, `normal`, `abnormal`, `typical`,
  `atypical`, `on track`, `ahead`, `missed`, `missing`, `concern`, `worry`, `should`, `red flag`, `warning`.
  Tests check the Milestones page text and the PDF for them. In the PDF's sickness and medication sections
  "due", "overdue" and "reminder" are refused too. The only status words in the PDF are the vaccination
  statuses (Given, Due, Upcoming, Overdue, Dismissed, Available), in the vaccination section only.
- **Nothing coloured, iconised or counted by state**: no checkmark, tint, strike-through, badge, progress bar,
  ring, percentage, "3 of 11", "not yet", per-list or per-domain totals. No `svg`, `img`, `progress` or `meter`
  in the Milestones page (tested). An unreached item differs from a reached one **only** by the absence of
  the date line.
- **CDC's item text is verbatim**: never reword, shorten, truncate with an ellipsis, change a pronoun, or add a
  CDC logo, photo, video or illustration. Never show CDC's omitted material (intro, "what most children (75% or
  more) can do", doctor questions, "act early" boxes, tips).
- **CDC's attribution and screening sentence stay visible next to the lists**, full size, never inside a
  collapsed element, a tooltip or a footer link; in the PDF they print at body size under the Milestones
  heading.
- **No age at the time of reaching**, no comparison of a mark's date with a list age, no "expected by".
- **No list of unreached items in the PDF**, no count of them.
- **No JS-only path**; no new navigation tab; milestones never on the timeline or the Due list.
- **Never put a name in the PDF file name or metadata**; never put the range or module list in a URL beyond the
  validation redirect.
- Do not rename the accessible names in appendix B without listing the rename.

---

## 9. Appendix A — app copy, verbatim (screens)

Dynamic parts in `{braces}`.

**Milestones page** (`milestones/+page.svelte`, `Checklist.svelte`, `ui/src/lib/milestones.ts`)

| Element | Text |
|---|---|
| Title | Milestones · {child} · happybabytracker |
| h1 | Milestones for {child} |
| Off note | Milestones are off for {child}. · Turn it on in Modules (link, guardian only) |
| Jump link | Go to the current list · Go to the next list |
| Sections (h2) | Milestone checklists · Reached · Reached, no longer in the lists · Your milestones |
| List summary | {title} · {title} · Current list for this age · {title} · Next list |
| List titles | By 2 months · By 4 months · By 6 months · By 9 months · By 1 year · By 15 months · By 18 months · By 2 years · By 30 months · By 3 years · By 4 years · By 5 years |
| Domain headings (h3) | Social/Emotional · Language/Communication · Cognitive (learning, thinking, problem-solving) · Movement/Physical Development |
| Item controls | Reached (sr: " {item text}") · Choose a date (sr: " for {item text}") |
| Reached line | Reached on {d} {Month} {yyyy} (e.g. Reached on 4 March 2026) · Edit (sr: " {text}") |
| After-5 detail line | {list title} · {domain} |
| Empty | No milestones recorded (after 5, "Reached") · No milestones added ("Your milestones") |
| Add link | Add a milestone |
| CDC block (from the catalog, fixed) | Learn the Signs. Act Early. materials are not a substitute for standardized, validated developmental screening tools. · Source: CDC, Learn the Signs. Act Early. milestone checklists. CDC does not endorse this application or its use of the material. The material is available free of charge at https://www.cdc.gov/act-early/milestones/index.html. |
| Row / page errors | Already marked as reached · This child has the maximum number of recorded milestones · This milestone is no longer in the list · The app cannot reach its server, try again soon · Something went wrong, try again |

**Mark / add** (`milestones/new/+page.svelte`, `MilestoneFields.svelte`, `milestone-form.ts`)

| Element | Text |
|---|---|
| Title / h1 | Mark as reached · Add a milestone (title adds " · {child} · happybabytracker") |
| Back link | Milestones for {child} |
| Fields | Milestone (add only) · Date reached · Note |
| Buttons / links | Save · Cancel |
| Errors | This milestone is no longer in the list (item) · Enter a milestone of up to 200 characters · Enter a date that is not in the future and not before the birth date · The note is too long · Already marked as reached · This child has the maximum number of recorded milestones |

**Edit** (`milestones/[entry]/+page.svelte`)

| Element | Text |
|---|---|
| Title / h1 | Edit milestone · {child} · happybabytracker / Edit milestone |
| Catalog delete | Undo reached · This removes the date for everyone with access to {child}. · Yes, undo reached |
| Custom delete | Delete this milestone · This removes the milestone for everyone with access to {child}. · Yes, delete this milestone |
| Fields, buttons, errors | as Mark / add |

**Export page** (`export/+page.svelte`, `export-form.ts`)

| Element | Text |
|---|---|
| Title / h1 | Export PDF · {child} · happybabytracker / Export PDF for {child} |
| Sentence | The file is made when you download it and is not kept. |
| Fields | From · To |
| Fieldset / group | Modules to include · Switched off for this child |
| Box labels | Bottle · Nursing · Pumping · Diaper · Sleep · Feeding · Milestones · Growth · Vaccination · Sickness · Medication |
| Button | Download PDF |
| Errors | Enter a start date that is not before the birth date · Enter an end date from the start date to today, at most 93 days in all · Choose at least one module · The PDF cannot be made right now, try again in a moment |
| File name | happybabytracker-{from}-{to}.pdf |

**Kid links / tile**: Milestones (sr: " for {child}") · Export PDF (sr: " for {child}") · tile "Milestones".

**Error page** (shared, unchanged): Not found · The app cannot reach its server right now · Something went
wrong · Try again in a moment. · Error {status}. · Try again.

---

## 10. Appendix B — accessible names the e2e tests rely on

From `ui/e2e/milestones.spec.ts`, `ui/e2e/export.spec.ts` and `ui/e2e/helpers.ts`.

- **Headings:** h1 "Milestones for {child}", "Mark as reached", "Add a milestone", "Edit milestone", "Export
  PDF for {child}", "{child}" (kid home); h3 domain headings inside a list, in M3 order; heading "Switched off
  for this child"; "Not found" (error page).
- **Links:** "Go to the current list" (href `#list-{age}`), "Go to the next list", "Turn it on in Modules",
  "Add a milestone", "Choose a date for {item text}" (href `…/milestones/new?item={id}`), "Edit {text}" (exact),
  "Milestones for {child}" (kid link), "Export PDF for {child}" (kid link), quick-log "Milestones" (exact,
  inside the region "Quick log").
- **Buttons:** "Reached {item text}", "Save" (exact), "Yes, undo reached", "Yes, delete this milestone",
  "Download PDF".
- **Summaries (text):** "Undo reached", "Delete this milestone"; list summaries matched as `^{title}( ·|$)`, with
  the exact texts "By 9 months · Current list for this age", "By 4 years · Current list for this age", "By 5
  years", "By 2 months · Next list", "By 2 months · Current list for this age", "By 4 months"; twelve summaries
  in age order.
- **Fields (exact labels):** "Date reached", "Note", "Milestone", "From", "To"; Modules page checkbox
  "Milestones".
- **Groups / regions:** group "Modules to include" (all eleven checkboxes, named by `MODULE_LABEL`), group
  "Switched off for this child", regions "Your milestones", "Reached" (exact), "Reached, no longer in the
  lists", "Quick log".
- **Exact texts asserted:** "Current list for this age" (exactly once on a page that has a current list; zero
  before 2 months), "Next list", "Milestones are off for {child}.", "Already marked as reached", "No
  milestones added", "Reached on …" (via `reachedLabel`), "The file is made when you download it and is not
  kept.", "Choose at least one module", "The PDF cannot be made right now, try again in a moment"; alerts by
  `role="alert"`; field errors by `aria-invalid="true"` plus the accessible description.
- **Structural assertions:** the CDC sentence and attribution sit **below the last list summary**; the page
  never contains the word "omitted" nor any `svg`, `img`, `progress`, `meter`; the denylist words never appear
  outside item text; no "remind" text; no horizontal overflow at 390 px.

---

## 11. Appendix C — the PDF

### C.1 Fixed constraints (Built; do not change without labelling)

| Constraint | Value |
|---|---|
| Page | **A4 portrait** 210 × 297 mm, but everything (content and footer) within the top **279.4 mm** so it prints at 100 % on US Letter (`ExportPageHeightUsableMM`) |
| Margins | **12 mm** left, right, top (`ExportMarginMM`); content width **186 mm** |
| Fonts | **Noto Sans Regular and Noto Sans Bold only** (embedded, subset). No italic, no other weight, no other family (an italic would need another embedded file: costly) |
| Sizes | body **9 pt**, heading **12 pt**, small **7 pt** (`ExportFontSizeBody/Heading/Small`) |
| Colour | **greyscale only**: black text and rules; grey (level 150 of 255) curve hairlines; black filled squares for points |
| Primitives | **text, lines, rectangles** (stroked or filled). No image, logo, link, annotation, attachment, JavaScript, form field, encryption, HTML |
| Size | ≤ **4 MiB** per file (else refused, never truncated); compression off |
| Rows | ≤ 3000 per section, ≤ 6000 per file; then one overflow line |
| Range | ≤ 93 days |
| Metadata | Title "happybabytracker export", Producer "happybabytracker"; no author, subject, keywords; no name anywhere in metadata or file name |
| Characters | a rune the fonts lack (or above U+FFFF, or a control character) prints as `?`; newlines and tabs in notes become one space |
| Determinism | same data + same clock = byte-identical file |

### C.2 Layout as built (`pdfdoc/doc.go`)

| Element | How it is drawn |
|---|---|
| Line height | 1.4 × font size (9 pt → 4.45 mm; 7 pt → 3.46 mm; 12 pt → 5.93 mm); cell padding 0.25 × font size on every side |
| Heading | Noto Sans **Bold 12 pt**, full width; one 9 pt line of space above it (except at a page top); never left alone at a page bottom (kept with one body line) |
| Profile line | two cells: label **Bold 9 pt** in the left 30 % (55.8 mm), value Regular 9 pt in the rest; wraps |
| Body | Regular 9 pt, full width, wrapped (CDC sentences, vaccination caption, "Statuses as of …", "No schedule chosen") |
| Paragraph | Regular **7 pt**, full width, wrapped (attributions other than CDC's milestones, empty-state lines, overflow lines, the `?` notice) |
| Table | header row Bold 9 pt with a **0.2 mm black rule** under it; body rows Regular 9 pt; **no other borders, no fills, no zebra**; cells wrap; a row never splits unless it starts a page; the header **repeats** after each page break; columns are shares of 186 mm, scaled down if wider |
| Chart | title Bold 9 pt; y-axis label above the plot (7 pt); plot height 0.55 × 186 = **102.3 mm**; black 0.2 mm axes (left and bottom); tick labels only at the axis ends (min and max); x-axis label centred under; reference curves as **grey 0.1 mm hairlines**, each labelled with its percentile ("50th") **at the right edge**, labels pushed at least one small line apart; the child's points as **black filled squares** (side ≈ 1.2 mm); nothing outside the ranges drawn |
| Footer | every page, Regular **7 pt**, left-aligned at the left margin, baseline 267.4 mm: "Generated {YYYY-MM-DD HH:MM} ({IANA zone}) by happybabytracker · Page {n} of {N}" |
| Page break | content stops at ≈ 260.5 mm (two small lines above the footer baseline) |

### C.3 Content, in order

1. **Profile** (always): heading "Profile"; lines "Name", "Birth date", "Sex" ("Female"/"Male"), "Time zone",
   "Age today" ("{age}", "{age} (corrected {age})" or "{age} (before expected term)"); "Growth source" (only if
   growth is chosen; e.g. "WHO Child Growth Standards (2006) and Growth Reference (2007)" or "CDC 2000 Growth
   Charts"); "Vaccination schedule" (only if vaccination is chosen: "{authority}, {edition}" or "No schedule
   chosen"); "Range" ("{from} to {to}"); "Modules" (chosen section names, comma-separated, in order); then,
   only if a substitution happened anywhere, the small line "Some characters cannot be shown in this PDF and
   appear as ?".
2. **Daily trackers** (Bottles, Nursing, Pumping, Diapers, Sleep, Feeding, each if chosen): heading; table
   "Time | Entry | Note" (20 % | 50 % | 30 %), **newest first**, times "YYYY-MM-DD HH:MM" in the child's zone
   (plus a numeric offset such as " -04:00" only for a time inside a repeated fall-back hour); then a table
   "Date | Day total" (20 % | 80 %), days ascending, only days with something; or the small line "No entries
   in this range."
   - Entry wording: "Bottle {volume}, {content} + {content}" or with per-source volumes "Bottle 120 ml, breast
     milk 60 ml + formula 60 ml"; "Nursing L {duration} · R {duration}" (+ ", running", " on L|R", ", weight
     before {w}", ", weight after {w}", ", weight gain {g}, about {volume}"); "Pumping {duration}, {volume}
     ({side})" or "({L 60 ml · R 60 ml})"; "Diaper {kind}"; "Sleep until {date time} ({duration})" or "Sleep,
     running"; "Solids: {what}" (+ " · {reaction}").
   - Day totals: "{n} · {volume}" (bottles, pumping), "{n} · {duration}" (nursing), "{n} ({n} wet, {n} dirty,
     {n} both, {n} dry)" (diapers, zero kinds left out), "{duration}" (sleep), "{n}" (feeding).
   - Durations "45 s", "12 min", "2 h", "1 h 30 min".
3. **Milestones** (if chosen): heading "Milestones"; **CDC's screening sentence at body size**; **CDC's
   attribution at body size**; then the rows reached **up to the range's last date, oldest first**, in up to
   three tables: "Date | Checklist | Domain | Milestone | Note" (14 | 14 | 18 | 34 | 20 %), then "Date | Your
   milestones | Note" (14 | 66 | 20 %), then "Date | Reached, no longer in the lists | Note". None: small line
   "No milestones recorded". **No unreached item, no count.**
4. **Growth** (if chosen): heading "Growth"; table "Date | Age | Weight | Length | Head circumference | BMI"
   (14 | 18 | 17 | 17 | 17 | 17 %), every measurement up to the last date, **newest first**; date as "Wed 4 Mar
   2026"; age as "3 months 4 days" (+ " (corrected …)" / " (before expected term)"); each value as "{value}: z
   {z}, percentile {p}" or "{value}: before expected term" / "{value}: this source has no table for this age";
   percentiles beyond 0.1/99.9 print "<0.1" / ">99.9". Then small lines: the source attribution, "Licence: {name
   (link)}", and for WHO the adaptation disclaimer paragraph(s). Then per measure: either the line "{measure} |
   {source} has no table for this measure at this age" or a chart titled "{measure}" with y label "{measure}
   ({kg|lb|cm|in|kg/m²})" and x label "Age (months)", "Corrected age (months)" or "Age (months; corrected before
   the second birthday)". None: "No measurements recorded."
5. **Vaccination** (if chosen): heading "Vaccination"; body "Schedule: {authority}, {edition}"; body "Statuses
   as of {date}."; small attribution of the schedule; table "Group | Vaccine | Status | Detail" (20 | 30 | 12 |
   38 %): group "Birth" or "{n}-{unit} vaccines" (e.g. "2-month vaccines") or "Also available"; vaccine
   "{antigen}, dose {n}"; status one of Given, Due, Upcoming, Overdue, Dismissed, Available; detail "Given {date}",
   "Dismissed: not needed", "Dismissed: given, date unknown", "Due from {date} · overdue from {date}", "Due from
   {date} · no longer due from {date}", "Available from {date}". Then doses given up to the last date, newest
   first: "Date | Vaccines | Product | Lot | Note" (14 | 30 | 20 | 12 | 24 %), or "No vaccinations recorded.".
   Then "Date | Other recorded vaccinations | Product" or the line "Other recorded vaccinations | None". With no
   schedule chosen: body "No schedule chosen", then the doses table only.
6. **Sickness** (if chosen): heading "Sickness"; "Time | Entry | Note", **oldest first**; entry as the timeline
   words it without its clock: "Cough, moderate", "Fever 38.6 °C (oral), mild", "Other: {text}"; temperature in
   the requester's unit. None: "No entries in this range."
7. **Medication** (if chosen): heading "Medication"; courses touching the range (active or not, plus any course
   with a dose in the range), oldest start first: "Course | Dose | Schedule | Start | Last day | Active" (20 | 15
   | 20 | 17 | 13 | 15 %); schedule "As needed", "08:00, 20:00", "Every 6 hours from 08:00", "Every hour from
   08:00"; active "Active" / "Not active". Then "Given at | Course | Amount | Note" (20 | 25 | 20 | 35 %), oldest
   first, or "No doses in this range.". No course: "No entries in this range." **Nothing about reminders,
   next due, last dose or counts.**

Overflow lines (small): "{n} more entries in this range are not shown. Export a shorter range to see them." ·
"1 more entry in this range is not shown. Export a shorter range to see it." · growth: "{n} more entries up to
the end of the range are not shown. Export a range that ends earlier to see them." · "1 more entry up to the end
of the range is not shown. Export a range that ends earlier to see it." A section past the file budget prints
only its overflow line.

### C.4 What a designer MAY change, and what is fixed

| May change (propose new values; each is a constant or a layout choice) | Fixed |
|---|---|
| Font sizes (9/12/7 pt), line height (1.4), cell padding, heading spacing | Noto Sans Regular/Bold only; no italic, no colour |
| Column shares of every table; which columns wrap | The column **content** (which facts) and their wording |
| Table styling with rules and grey-filled rectangles (header band, zebra rows, section boxes), within greyscale | No images, logos, icons, links, colour |
| Profile block presentation (a boxed area, two columns) | Profile facts and their order of meaning; the `?` notice when substitution happened |
| Section separation (rules, space, page break before a section) | Section order (module order); no section for an unchosen module |
| Chart size, aspect, tick density, label placement, point marker shape (any black filled/stroked rectangle) | Grey reference curves with percentile labels; black points; nothing coloured by value; no shaded "zones" |
| Footer typography and placement (within 279.4 mm) | Footer facts: "Generated … ({zone}) by happybabytracker · Page n of N" |
| Ordering direction of rows (newest/oldest first), labelled as a proposal | Milestones: only reached rows; no count, no unreached list |
| Size of the empty-state and overflow lines | Their wording (appendix C.3) and the denylist |
| — | CDC's screening sentence and attribution **verbatim, at body size, directly under the Milestones heading**; CDC item text verbatim; growth attribution, licence line and WHO disclaimer; vaccination attribution and status words; A4 within 279.4 mm; ≤ 4 MiB |

How the PDF looks in a given viewer or printer (Preview, Chrome, iOS Files, a Letter printer) is
**Unverified**; no rendered page was viewed while writing this file.

### C.5 PDF copy inventory (app-written strings, `api/internal/export/*.go`)

| Group | Strings |
|---|---|
| Headings | Profile · Bottles · Nursing · Pumping · Diapers · Sleep · Feeding · Milestones · Growth · Vaccination · Sickness · Medication |
| Profile labels | Name · Birth date · Sex · Time zone · Age today · Growth source · Vaccination schedule · Range · Modules |
| Profile values | Female · Male · No schedule chosen · {from} to {to} · (corrected {age}) · (before expected term) |
| Notice | Some characters cannot be shown in this PDF and appear as ? |
| Footer | Generated {date time} ({zone}) by happybabytracker · Page {n} of {N} |
| Tracker columns | Time · Entry · Note · Date · Day total |
| Tracker words | Bottle · breast milk · formula · cow's milk · other · Nursing · L · R · running · on · weight before · weight after · weight gain · about · Pumping · Diaper · Sleep until · Sleep, running · Solids: |
| Empty | No entries in this range. · No milestones recorded · No measurements recorded. · No vaccinations recorded. · No doses in this range. · None |
| Milestone columns | Date · Checklist · Domain · Milestone · Note · Your milestones · Reached, no longer in the lists |
| Growth | Date · Age · Weight · Length · Head circumference · BMI · z · percentile · before expected term · this source has no table for this age · {source} has no table for this measure at this age · Licence: · Age (months) · Corrected age (months) · Age (months; corrected before the second birthday) · kg/m² · 1st/2nd/3rd/…th |
| Vaccination | Schedule: · Statuses as of {date}. · Group · Vaccine · Status · Detail · Birth · {n}-{day\|week\|month\|year} vaccines · Also available · , dose {n} · Given · Due · Upcoming · Overdue · Dismissed · Available · Given {date} · Dismissed: not needed · Dismissed: given, date unknown · Due from {date} · overdue from {date} · no longer due from {date} · Available from {date} · Vaccines · Product · Lot · Other recorded vaccinations |
| Sickness | Congestion · Cough · Fever · Vomiting · Diarrhea · Constipation · Rash · Ear pain · Sore throat · Poor appetite · Unusually sleepy · Other · oral · rectal · armpit · ear · forehead · other · mild · moderate · severe · °C · °F |
| Medication | Course · Dose · Schedule · Start · Last day · Active · Not active · Given at · Amount · As needed · Every hour from · Every {n} hours from |
| Overflow | see C.3 |
| Fixed third-party text | CDC screening sentence and attribution (appendix A); WHO/CDC growth attributions and licence names; vaccination schedule attributions (plan 5) |

---

## 12. Appendix D — icons and illustrations NOT needed

Not needed, and several are forbidden by section 8: checkmarks, ticks or ticked circles for reached items,
progress rings or bars, per-list or per-domain counts, trophy/star/badge graphics, a "milestone" flag icon that
changes with state, the CDC or "Learn the Signs. Act Early." logo, CDC photos or videos, child silhouettes or
age illustrations, a PDF file icon, a download spinner that implies a known duration. A neutral lucide glyph
beside "Export PDF" (for example `file-down`) is acceptable only if it encodes no state.

## 13. Appendix E — reference: verified against code at `88654ec`

| Screen / rule | Files read |
|---|---|
| Milestones page | `ui/src/routes/(app)/kids/[kid]/milestones/+page.svelte`, `+page.server.ts`; `ui/src/lib/components/milestones/{Checklist,MilestoneFields}.svelte`; `ui/src/lib/milestones.ts` |
| Mark / add / edit | `…/milestones/new/+page.svelte`, `+page.server.ts`; `…/milestones/[entry]/+page.svelte`, `+page.server.ts`; `ui/src/lib/server/milestone-form.ts`; `ui/src/lib/server/entry-failure.ts` (note message) |
| Export page and route | `…/export/+page.svelte`, `+page.server.ts`, `pdf/+server.ts`; `ui/src/lib/server/export-form.ts`; `ui/src/lib/server/api.ts` (`exportPdf`, `reasonFor`); `ui/src/lib/server/outcome.ts` (`failPage`); `ui/src/routes/+error.svelte` |
| Constants | `ui/src/lib/constants.ts` (`MILESTONE_*`, `MODULES`, `EXPORT_*`, `MODULE_LABEL`, `QUICK_LOG_FIRST`); `ui/src/lib/server/constants.ts`; `api/internal/constants/constants.go` |
| Kid links, tile | `ui/src/lib/components/KidLinks.svelte`; `ui/src/lib/quick-log.ts`, `quick-log.test.ts`; `ui/src/lib/growth.ts` (`ageHeader`) |
| Component sizes | `ui/src/lib/components/ui/button/button.svelte` (h-8 / h-7); `ui/input` (h-8); `ui/src/lib/components/entry/Field.svelte`; `field.ts` |
| PDF | `api/internal/export/pdfdoc/doc.go`; `api/internal/export/{export,profile,trackers,milestones,growth,vaccines,sickness,medication,format}.go`; `api/internal/store/export.go` (row order); `api/internal/growth/{source,tables}.go` and the shipped table headers (attribution, licence) |
| Catalog | `api/internal/milestones/data/cdc.csv` (header; counts and lengths computed by a read-only script); `docs/milestones/cdc.md` (skimmed) |
| Tests (names) | `ui/e2e/milestones.spec.ts`, `ui/e2e/export.spec.ts`, `ui/e2e/helpers.ts` |
| Rules and decisions | `docs/superpowers/plans/2026-11-23-milestones-pdf-export.md` (Global Constraints, Owner answers, Open values, rulings R1–R8, P-*, T*); `README.md` § Milestones, PDF export, Third-party data; `docs/superpowers/HANDOFF.md` (plan 7 owner items, decisions, not verified) |

**Unverified:** every rendering question (how a browser draws the native `<details>` summary and its tap height,
the date inputs and their validation bubbles, the download UI; how any PDF viewer or printer draws the file;
whether Letter printing at 100 % keeps the footer); the viewport position after an enhanced one-tap mark;
whether following the jump link opens a closed list; mobile download on iOS (Safari and the installed PWA) and
Android. The UI unit and e2e suites and the API tests were **not run** while writing this file; every string
was read from source.

## 14. What to hand back

A design spec, not code. Per screen at **390 px and ≥ 1024 px, light and dark**:

1. The **Milestones page**: a child at 9 months (current list open), under 2 months (next list), preterm (both
   ages), 4 years 11 months, after the 5th birthday ("Reached" section), module off for a guardian and for a
   caregiver; reached and unreached items side by side (differing only by the date line); the CDC block; "Your
   milestones" empty and filled; a row error.
2. **Mark as reached**, **Add a milestone**, **Edit milestone** (catalog and custom; undo/delete confirm closed,
   open, failed).
3. The **Export page**: default, with a switched-off group, each field error, the busy message, no JS.
4. The **kid links** and the **quick-log tile** in each band.
5. The **PDF**: a page-by-page layout at A4 with the 279.4 mm usable box marked; the profile; one daily tracker
   with its day totals; milestones (with the CDC lines); growth table, attribution and one chart; vaccination;
   sickness; medication; empty states; an overflow line; the `?` notice; the footer. Give values for the C.4
   constants you change.
6. Annotated layout, the **shadcn-svelte component mapping** (and components to add), **all copy** (CDC's
   text and attribution unchanged), lucide icon picks (appendix D lists what is not needed), token changes as
   new `oklch` values, the accessible names you change (appendix B), an **API-change list** tagged cheap /
   costly, and your treatment of each item in section 7 labelled as an assumption.
