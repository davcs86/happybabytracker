# Plan 7 design notes: milestones and PDF export

Hand-back for `uploads/plan-7-design-handoff.md` (section 14). A design spec, not code. Everything here is a design assumption unless the handoff says otherwise. Verified in this prototype by Node logic tests only (store rules, view models, copy denylist, no `svg`/`img`/`progress`/`meter` in the Milestones markup). **Nothing was rendered in a browser, axe was not run, no PDF was produced or printed.**

## 0. Read this first

1. **The CDC catalog text is not verbatim-verified.** The 159 items (12 lists, 4 domains, counts 11/13/12/13/10/13/15/13/15/12/17/15) were transcribed from the CDC checklist pages through a summarizing page fetch. Counts match the handoff. Punctuation (curly vs straight quotes), the example sentences of a few items and two merged long items ("Tells a story…", "Pays attention…") may differ from CDC. **The API's catalog CSV is the source of truth**; replace `CAT` in `hbt-ms.js` from it before anyone reads the copy as final.
2. **Sample children.** The Milestones, form and Export pages show a scenario child chosen in the panel (9 months, under 2 months, preterm, 4 years 11 months, after the 5th birthday, 30 months with the module off). Ages are computed from today, by the same rule the API uses (M4 to M6), so the scenarios stay valid on any date.
3. **The PDF page shows sample values.** Weights, vaccine rows and the growth curves are illustrative. Source attributions, the licence line and the WHO disclaimer are placeholders in square brackets: they stay verbatim from the API.

## 1. Where each piece is

| Handoff | Prototype page | Panel (what you can switch) |
|---|---|---|
| 4.1 Milestones page | `Milestones.dc.html` | Child scenario; module On/Off; next one-tap (works, already marked, cap); "Your milestones" filled/empty; one retired row; page state (normal, server down, not found); "Choose a date" (link, as built / reveal in the row, proposal); Role (guardian, caregiver); reset marks |
| 4.2 Mark as reached / Add a milestone, 4.3 Edit milestone | `Milestone Form.dc.html` | Screen (mark, add, edit catalog, edit custom); undo/delete confirm closed / open / failed |
| 4.4 Export page | `Export.dc.html` | Modules (all on / some switched off); after a refusal (none, From, To, Modules, Busy); page state |
| 4.5 Kid links, quick-log tile | `Kid Home.dc.html` | The link row under the name; the tile in each band (child switch); Modules sheet |
| 4.6, appendix C PDF | `PDF Export.dc.html` | As built (C.2) / proposed (C.4); zoom; guides (margins, 279.4 mm line, ≈260.5 mm break line) |
| Store | `hbt-ms.js` (new), `hbt-data.js` (`deleteKid` also drops milestone rows) | Catalog, age basis, current/next list, marks, caps, D3 rule, export checks stand in for the server |

The Milestones page contains no `svg`, `img`, `progress` or `meter`. The app chrome around it (bottom tabs) keeps its icons, as the handoff leaves navigation unchanged.

## 2. Annotated layout

### 2.1 Milestones page (390 px column; ≥ 1024 px the same column, max 672 px, left-aligned in the content area)

| # | Region | Notes |
|---|---|---|
| 1 | Back link "‹ {child}" | 44 px high |
| 2 | `h1` "Milestones for {child}" | 30 px, wraps anywhere |
| 3 | Age line, muted | "9 months 3 days", "3 months 4 days (corrected 1 month 12 days)", "20 days", "{age} (before expected term)" |
| 4 | Off note | plain 14 px paragraph; guardians also get the underlined link "Turn it on in Modules". Caregiver: note only |
| 5 | Page alert `role="alert"` | only for failures not tied to an item |
| 6 | Status line `role="status"`, muted | **Proposal 7.2-3.** "Marked as reached: {text}" after a one-tap |
| 7 | Jump link | "Go to the current list" / "Go to the next list"; anchor `#list-{age}`; absent after the 5th birthday |
| 8 | `h2` "Milestone checklists", then **checklist index** | **Proposal 7.2-2.** A wrapping row of 12 plain anchor links ("2 months" … "5 years", accessible name = list title), each 44 px high; no marker on the current list (the text in its summary says it) |
| 9 | Twelve `<details id="list-{age}">` | bordered, 12 px radius, 12 px padding; summary 44 px min height, 15 px medium: "{title}" or "{title} · Current list for this age" or "{title} · Next list". Inside: `h3` per domain, items, then the link "Back to the checklist index" |
| 10 | Item row `id="item-{id}"` | text (14 px); unreached: outline button "Reached" (44 px) + "Choose a date"; reached: muted line "Reached on {d Month yyyy}" + "Edit". Same size, weight and colour of text in both states; the only difference is that line |
| 11 | Row error | `role="alert"`, 13.5 px semibold, `--destructive` colour; the words carry the meaning |
| 12 | After the 5th birthday | `h2` "Reached" with bordered rows: text, muted "{list title} · {domain}", date line, Edit; "No milestones recorded" when empty |
| 13 | `h2` "Reached, no longer in the lists" | only when such rows exist |
| 14 | CDC block | 14 px, two paragraphs, never folded, after the last list (or the Reached section) and before "Your milestones" |
| 15 | `h2` "Your milestones" + link "Add a milestone" on the same row | bordered rows; "No milestones added" when empty |

Item spacing: 10 px vertical padding, 1 px top border. Targets: 44 px for Reached, Choose a date, Edit, the summaries and the index links (**answer to 6.1 and 7.2-1**).

### 2.2 Forms

`h1` "Mark as reached" / "Add a milestone" / "Edit milestone"; back link "‹ Milestones for {child}"; page alert; CDC text as a paragraph (mark, edit catalog); fields (`Milestone` for add and custom edit, `Date reached`, `Note`), Save (primary) and Cancel (link styled as a button), 48 px high. Edit adds a bordered `<details>`: "Undo reached" or "Delete this milestone", the consequence sentence, an outline button with `--destructive` border and text ("Yes, undo reached" / "Yes, delete this milestone"). A failed action reopens the details with the message inside.

### 2.3 Export page

`h1`; muted sentence "The file is made when you download it and is not kept."; busy alert; From and To side by side at 390 px (two `minmax(0,1fr)` columns, 48 px inputs, errors under each); fieldset "Modules to include" with a two-column checkbox grid (22 px box, label row ≥ 44 px); an inner group named by `h2` "Switched off for this child" only when a module is off; the modules error under the fieldset; "Download PDF" (48 px, left). A text link "See the PDF layout (design reference)" and a dashed "Prototype only" box exist only in the prototype.

### 2.4 Kid Home

A wrapping row of text links under the name (`nav`, accessible name "Links for {child}"): Vaccines, Medication, **Milestones**, **Export PDF**, then for guardians Settings, Access, Modules. Each link is 44 px high; its accessible name adds " for {child}". The quick-log tile "Milestones" is label only (no chip, no icon), placed: bands 0–11 and 12–23 months after Pumping, before Growth; 24 months and over last, only when the module is on. The More sheet also lists Milestones and Export PDF for every role.

## 3. shadcn-svelte mapping

| UI | Part |
|---|---|
| Checklists, "Undo reached", "Delete this milestone", the date reveal | `<details>` styled as `Collapsible` (no JS) |
| Reached, Save, Download PDF, Yes buttons | `Button` (outline for Reached and the Yes buttons; default for Save and Download) |
| Fields | `Field` + `Label` + `Input` (`type="date"` for dates) |
| Module boxes | `Checkbox` inside a native `fieldset` with `legend` |
| Alerts and the status line | `Alert` (`role="alert"` / `role="status"`) |
| Rows (reached, retired, custom) | plain bordered `div`, `Card` styling without header |
| Not found, server down | `Card` |

**Components to add: none.**

## 4. Copy

All strings are appendix A verbatim. New strings are proposals:

| String | Where | Status |
|---|---|---|
| "Marked as reached: {text}" | status line after a one-tap | Proposal (7.2-3) |
| "Checklist index" (nav name); link texts "2 months", "4 months", "6 months", "9 months", "1 year", "15 months", "18 months", "2 years", "30 months", "3 years", "4 years", "5 years"; their accessible names are the list titles | index | Proposal (7.2-2) |
| "Back to the checklist index" | end of each list | Proposal |
| "Date reached" + "Save" (name "Save the date for {text}") | date reveal | Proposal (7.2-4) |
| "The date was removed." / "The milestone was deleted." | prototype confirmation only (the real app answers with a 303 to the Milestones page) | Prototype only |
| "Prototype only, not part of the page." box | Export | Prototype only |

The CDC sentence and attribution are unchanged: "Learn the Signs. Act Early. materials are not a substitute for standardized, validated developmental screening tools." and "Source: CDC, Learn the Signs. Act Early. milestone checklists. CDC does not endorse this application or its use of the material. The material is available free of charge at https://www.cdc.gov/act-early/milestones/index.html." The prototype checks that no denylist word appears in the new copy, the catalog or the PDF sample.

## 5. Icons

None needed (appendix D followed). The Milestones tile and every kid link are text only. No icon, checkmark, ring or tint marks a reached item anywhere.

## 6. Tokens

One new token: `--destructive`, used for the row error text, the failed-delete message and the outline Yes buttons (always together with words and a 1.5 px border, never colour alone).

| Token | Light | Dark |
|---|---|---|
| `--destructive` | `oklch(0.46 0.17 27)` | `oklch(0.76 0.14 25)` |

Contrast against `--card` (light `oklch(0.998 0.003 70)`, dark `oklch(0.245 0.015 50)`) is expected to be above 4.5:1 for both; **not measured**. No other token changes: the "Reached on …" line uses `--muted` at the same size as the item text.

## 7. Accessible names (appendix B)

**None renamed.** Added names: the index nav "Checklist index" and its links (= list titles); the kid-links nav "Links for {child}" (each link "{label} for {child}"); the proposal's "Save the date for {text}". The date reveal summary keeps "Choose a date for {text}". The back link's visible "‹" is decorative; its name is "{child}" or "Milestones for {child}".

## 8. API changes (cheap / costly)

| Change | Tag |
|---|---|
| None for the as-built behaviour: every screen here works with the endpoints in section 2.2 of the handoff | n/a |
| Date reveal (7.2-4): the one-tap route accepts an optional `date` with the same validation as "Mark as reached"; a refusal shows in the item's row | **cheap** |
| Return to the item after "Choose a date" (gap 7): 303 to `?open={item}#item-{item}` like the one-tap | **cheap** |
| Status line after a one-tap (7.2-3): carry the item id in the redirect (`?open=` already does) and read its text server-side; no new parameter | **cheap** |
| `novalidate` on the export form (7.2-5) so only the app's messages show | **cheap** (UI-only) |
| Distinct "unavailable" message for timeouts and server down | **cheap** code, but needs the spec 13.3 clause to name `error=unavailable` (owner) |
| JS-enhanced download with progress (fetch and save), plain POST kept as the baseline | **costly**, not proposed |
| PDF changes in section 9 | **cheap** (constants plus two primitives that exist: filled rectangle, line); golden page-count tests change |

## 9. The PDF: C.4 constants I propose changing (labelled proposal; "As built" is the default view)

| Constant or choice | As built | Proposed |
|---|---|---|
| Table header | 0.2 mm rule only | grey fill level 235 behind the Bold 9 pt row; the 0.2 mm black rule stays |
| Row separators | none | 0.1 mm grey line, level 200, under every row except the last of a table |
| Cell padding | 0.25 × font size | 0.30 × font size; row height 9 pt × 1.4 + 2 × 0.30 × 9 pt ≈ 7.9 mm (was ≈ 7.7 mm) |
| Section separation | one 9 pt line of space | one 9 pt line of space, a 0.3 mm black rule, a 4 pt gap, then the heading (not at a page top) |
| Profile block | two cells per line, label 30 % | boxed (0.2 mm stroke), two columns of label over value |
| Chart plot height | 0.55 × 186 = 102.3 mm | 0.45 × 186 = 83.7 mm, so a table and one chart share a page |
| Footer | text only, baseline 267.4 mm | same text and baseline; a 0.1 mm grey hairline (level 150) at 263.4 mm |
| Unchanged | fonts, 9/12/7 pt, line height 1.4, 12 mm margins, 279.4 mm, greyscale, column content and wording, section order, CDC lines at body size under the Milestones heading |

Row order (gap 14) is not changed; each section's direction is as in appendix C.3. The page shows seven pages: profile and Bottles with day totals and Nursing; Pumping empty state, Diapers, Sleep and Feeding with an overflow line; Milestones with the CDC lines; Growth with table, source lines and one chart; Vaccination; Sickness and Medication; empty states. The `?` notice and the footer appear as in C.3. A height estimate in the tests puts the fullest page at about 212 mm from the top, under the ≈ 260.5 mm break.

## 10. Section 6 (known gaps), how each is treated

| Gap | Treatment |
|---|---|
| 1 Small targets | 44 px for Reached, Choose a date, Edit, summaries, index links, kid links, Save and Download (48 px). Assumption; with 159 rows the page grows by roughly 12 px per item |
| 2 Long page | Checklist index and "Back to the checklist index" (anchors, no JS). No "collapse all" |
| 3 No feedback | `role="status"` line; the one-tap stays "today"; the date reveal is the shorter path to a past date |
| 4 Viewport after one-tap | **Unverified**; the prototype keeps the page in place. Design intent: the row stays in view |
| 5 Jump link to a closed list | The jump link targets the marked list, which the server renders open; if the reader closed it, whether the browser opens it is **Unverified** |
| 6 Three date styles | Left as built; the PDF's ISO style is a print convention |
| 7 "Choose a date" loses the place | Return to the item (cheap, section 8) |
| 8 Custom rows after the 5th birthday | Left as built: two sections |
| 9, 10 No progress, no confirmation after export | Kept: the page does not change. The prototype shows a dashed note only to prove the click |
| 11 Browser validation | Proposal: `novalidate` on the export form |
| 12 One message for every failure | Kept (owner item) |
| 13, 14, 15, 16, 17 | Left as built; noted |
| 18 Mobile download | **Unverified** |

## 11. Section 7 items (labelled as assumptions)

**7.1 Owner items** are recorded as decided: catalog of twelve lists, ≤ 93 days, module default off from 24 months, the off note with its guardian link, "Go to the next list" / "Next list", the busy message for every non-field failure, the redirect carrying `error` and `modules`.

**7.2** — 1 Targets: 44 px (assumption). 2 Navigation: index of anchors plus a back link (assumption). 3 After a one-tap: status line, open lists stay open (assumption). 4 Past date: a per-row `<details>` reveal, offered as a switchable proposal beside the as-built link. 5 Export dates: two columns at 390 px, no range length shown, `novalidate` (assumption). 6 Download feedback: none, with no JS-safe way to know when a native download ends (assumption). 7 Mobile download: **Unverified**; the page promises nothing. 8 PDF hierarchy: section 9. 9 CDC block: after the last list and before "Your milestones", same width and size as the lists, never folded.

## 12. What the prototype simulates

The store (`hbt-ms.js`) stands in for the server: current/next list and the corrected-age rule, the one-tap "today" rule, caps (1000 marks, 200-character text, 500-character note), "Already marked as reached", and rule D3 (the first mark for a child whose module is off turns it on: the note disappears and the tile appears). The simulated refusals and the page states are panel switches, not real failures.


## 13. Design review changes (frontend-design pass, 2026-11-24)

Reason: the Milestones page buried the child's list under a title, a jump link, a 12-link index and three collapsed boxes; Kid Home's link row wrapped to two lines; Export used browser-blue checkboxes; the prototype panel clipped long scenario labels.

- **Milestones page:** the list for the child's age (or the next list before 2 months) now comes first, open, flat (no card), with the sentence "The list for {child}'s age" / "The next list". The other eleven sit under "Other checklists" as hairline rows with the native disclosure marker, ascending by age. **Removed on purpose:** the "Checklist index" nav (12 anchor links), the "Go to the current list" jump link, "Back to the checklist index", and the " · Current list for this age" suffix in list names. Renamed: list summaries are now just "By 4 months" etc. Reached / custom / retired rows are flat hairline rows, not bordered boxes. Unchanged: row content, "Reached on {date}" line, CDC sentences, no colour/icon/count.
- **Kid Home:** the link row is Vaccines, Medication, Milestones, Export PDF (one line). Settings, Access, Modules remain in More. Tile labels sit on the bottom of the tile so the label-only Milestones tile aligns with the others (still no icon, per handoff).
- **Export:** checkboxes use the app primary (`accent-color`); the link to the PDF design reference moved from the page into the prototype panel.
- **Prototype panel:** segmented controls wrap instead of clipping.
- Not changed: palette and type (shared by the already-accepted screens). Changing them is a separate decision.

## 14. Consistency pass: tiles, buttons, row actions (2026-11-24)

Owner feedback: tiles without icons or sublabels on Kid Home; CTA colours not uniform; text buttons on some screens and icon buttons on others.

**Kid Home tiles.** Every tile now has a coloured chip, an icon and a sublabel. New: Milestones (milestone icon, sublabel is the child's list, for example "By 4 months", or "Add a milestone" after 5 years), Sickness ("Log a symptom"), Medication ("Log a dose"). This reverses the plan 7 rule "tile is label only"; the no-colour, no-icon rule still holds inside the Milestones page for status. The selected child pill uses the same primary fill as every other selected pill (it was the neutral ink fill on Kid Home).

**Button spec (applied to all pages by `normalize.py`).** Primary: primary fill, one per screen or sheet. Secondary: card fill, 1 px border. Destructive confirm: card fill, 1.5 px destructive border and text (Dose, Course and Medication delete confirms were black-filled, Milestone form was already this). Default size 48 px, radius 12, 15 px/600; large (sheet submit and its Cancel) 56 px, radius 14, 16 px/600. Vaccines: the banner's "Dismiss older vaccines" is secondary so the page has one primary ("Record a vaccination"); the outlined primary-colour "Record doses" buttons are secondary.

**Row actions.** A non-destructive action repeated on every row is a 48 px icon button with a stable accessible name and a title: Edit (pencil) on Growth and Milestones, Choose a date (calendar) on Milestones. Destructive or irreversible row actions keep a text label (Remove, Remove access, Dismiss). Page-level actions stay text. **Milestones page now contains inline svg (pencil, calendar) on action buttons only; there is still no status icon, tint, count or check.** If a test asserts no svg on that page, scope it to status marks.

**Back links.** "‹" is decorative (aria-hidden) with a gap, which fixes "forLeo" and "‹Leo" spacing; the link's accessible name is now the label without the chevron.

## 15. Consistency pass 2: forms, filters, title line, Milestones (2026-11-24)

Owner feedback: Vaccines has collapsible cards but Milestones is text only; Timeline and Export filters differ; some titles have a line under them and others do not; Add/Edit forms open as a Drawer in some places and as a page in others; form controls do not use shadcn-svelte uniformly.

**Rule 1. Drawer first, page as the no-JS route.** Every Add or Edit link to a form page carries `data-drawer="Title"` and keeps its `href`. With JS, a click opens the shadcn Drawer (bottom sheet, 90% height, close button, Escape, overlay tap, focus returns to the link) and the form page loads inside it with `?embed=1`. Without JS, or with a modified click (new tab), the link navigates to the page as before. Titles: Add a course, Edit course, Edit dose, Edit symptom, Choose a date, Edit milestone, Add a milestone. Native sheets on Kid Home, Growth, Vaccines, My Profile and Admin were already Drawers and stay. Child profile and Notifications stay pages: they are settings screens, not add/edit of a record.
- Embed mode hides the page chrome (back link, role badge, tabs, title, age line, prototype panel). Role and theme are passed as `&cg=1` and `&dark=1`.
- On save, the form page's status message is posted to the list page, which closes the Drawer, reloads its data and shows the message as a toast. Cancel and any link back to a list close the Drawer without a toast.
- Prototype caveat: the Drawer hosts the form page in an iframe because every form is its own static page. In the product this is a route with a Drawer layout (SvelteKit parallel or intercepted route), and the same page is rendered at full size on a direct visit.

**Rule 2. Title line.** A page about a child shows the child's age line under the title (Timeline, Medication, Course, Dose, Symptom, Milestone form, Export, Child profile; Growth, Vaccines and Milestones already did). Account pages (My profile, Notifications, Admin) show a 1 px hairline under the title instead. The Admin counts line moved below the hairline so the rule spans the full width.

**Rule 3. Filters and multi-select are the ToggleGroup chip** (`.hbt-chip`, 44 px, radius 8, pressed = soft fill, primary border and a leading check, so colour is not the only signal). Timeline filters, Export "Modules to include" and "Switched off for this child" (checkboxes inside chips), Admin filters and all form option groups use it. The old pill is gone.

**Rule 4. Form controls** come from one layer, `hbt-ui.js`: Input, Textarea, Select (48 px, radius 8, 3 px focus ring, destructive border for `aria-invalid`), Checkbox (20 px), RadioGroup (20 px). The layer styles native elements, so the no-JS pages look identical and no markup depends on it.

**Milestones** now match Vaccines: one card per age list with a chevron header (title, plus "The list for {child}'s age" or "The next list" on one card), the child's list open, and an inner card per item. Still no colour, status icon, count or progress. Reached and unreached items differ only by the "Reached on {date}" line, and the CDC sentence and attribution stay visible below the lists. List ids (`list-{age}`) and domain headings are unchanged.

**Accessible names.** Unchanged except: Admin filter buttons and form option buttons now expose their state through `aria-pressed` only (as before) with a decorative check.

**Files.** New: `hbt-ui.js` (load it after `support.js` on every page). Changed: every `*.dc.html`.

## 16. Global header, switcher, notifications, Vaccines (2026-11-24, third pass)

Owner feedback: quick links can go straight to More; Vaccines wastes vertical space and shows "Dismiss older vaccines" twice; there is no way to change child from inside a module, and the child pills do not scale past about four children or long names.

**Global header (every screen except the PDF layout).** One row, 48 px high: child switcher on the left, notifications bell and role badge on the right. Built once by `chrome.py`, so a page cannot drift. The Drawer-hosted copy of a page hides it.
- **Child switcher** (Home, Timeline, Growth, Vaccines, Medication, Milestones, Export, Child profile): a secondary button with the child's initial, the name (one line, truncated with an ellipsis) and a chevron. Accessible name: "Child: Leo. Switch child". It replaces the child pills on Kid Home and the "‹ Leo" back link on module pages. Module pages are reached from More and left with the bottom bar or the side nav.
- **Back links stay only where there is no switcher:** the four full-page fallback forms (Course, Dose, Symptom, Milestone form: "‹ Medication for Leo", "‹ Home", "‹ Milestones for Leo"), and the account pages (My profile "‹ Home", Notifications "‹ Account"; Admin has none).
- **Bell** (all screens): 48 px icon button to Notifications, accessible name "Notification settings". It is a settings link, not an inbox, so there is no unread dot. On the Notifications page it has `aria-current="page"` and a primary border.

**Switch child.** Drawer first, page as the no-JS route (same rule as the forms). The button is a link to `/children?from=<page>`; with JS the page opens in a Drawer sized to its content (scrolls above 90% height).
- List of all children, one 64 px row each: initial, name (wraps, never truncates), age, and a check on the current child (`aria-current="true"`). Above six children a "Find a child" field appears, with "No child matches" as the empty state.
- Picking a child saves it and returns to the same module for that child (`from`, whitelisted to the eight child pages, defaults to Home). With JS the page reloads under the closed Drawer.
- "Add a child" (guardians) closes the Drawer and opens Kid Home's Add a child sheet (`Kid Home.dc.html?add=1`). Caregivers see "Ask a guardian to add a child."
- Prototype panel on the Choose a child page: Children: Three / Nine, long names, to exercise the scaling case.

**Kid Home.** The Vaccines / Medication / Milestones / Export PDF link row is removed; all four are already in More. The Notifications link under "Your profile" is removed in favour of the bell.

**Vaccines.** The header now takes one title row instead of eight lines:
- Title row: "Vaccines", an Info button ("About this schedule", opens a sheet with the retrieved date, licence text and "What is not included") and a ⋯ button ("Schedule options").
- One age line, "14 months · from birth, not corrected age", and one schedule line, "Schedule: UKHSA, Sept 2026".
- The ⋯ sheet holds "Change schedule" and "Dismiss older vaccines". The second appears only when the earlier-vaccines banner is not showing, so there is exactly one Dismiss-older entry point at any time (the banner's button, or the sheet's). Caregivers see neither button.
- No-JS: Info and ⋯ are links to `/vaccines/schedule` and `/vaccines/options` (own routes, like the other Vaccines sheets).

**Accessible names changed on purpose.** Kid Home child pills (name, `aria-pressed`) → "Child: {name}. Switch child". Module pages "‹ {name}" → none (the switcher). Kid Home nav "Links for {name}" → removed. Vaccines disclosures "About this schedule" and "What is not included" → one Info button named "About this schedule"; "Change schedule" and "Dismiss older vaccines" keep their names but now live in the ⋯ sheet.

**Files.** New: `Choose Child.dc.html`. Changed: `hbt-ui.js` (header data, fit-to-content Drawer, switch and go-to messages), every page that shows the header, `Vaccines`, `Kid Home` (`--soft` colour added, `?add=1`), `My Profile`.

## 17. Final sweep (2026-11-24, fourth pass)

Method: a scripted audit of all 16 app pages × phone / ≥768 side-nav × light / dark (overflow, 44 px targets, headings, duplicate ids, accessible names, landmarks, button inventory, phone-vs-desktop control names), then a screenshot review of every page in all four states, then fixes, then axe (0 violations on the 16 app pages; PDF layout has no app frame by design) and the logic tests.

**Found and fixed**
- **No landmarks.** No page had a `main` or a `navigation`. The scroll area is now `role="main"`; the side nav and the tab bar are `role="navigation"` named "Primary" (only one is in the tree at a time). Kid Home's tab bar also gets the `data-hbt="tabs"` marker so it hides in the Drawer like the others.
- **Primary and secondary buttons differed by 2 px** (48 vs 50, the border sat outside the min-height). Buttons, button-like links and summaries are now `box-sizing:border-box`, so every 48 px button is 48 px; the icon buttons too (50 → 48).
- **Text under 12 px** (tab labels, role badge, section labels, chips, legends): floor is now 12 px; only chart tick labels are 11 px.
- **Button and link type scale.** 14 / 14.5 px on 48 px buttons (Cancel, Log dose now, Log something, Delete all data) → 15 px. Text links of 13 / 13.5 px (See the Timeline, N more on …, Add a child, Leave this child, Change the growth source) → 14 px.
- **Medication was the last old-style page** (underlined names as edit links at 20 px high, plain text rows). Rebuilt on the Timeline / Vaccines card language: icon tile, name and dose, 48 px pencil button, schedule / last / next lines, amount field, Log dose now + Show doses; "Ended and inactive courses" is a card disclosure with a chevron; Recent doses are rows like Timeline rows (tile, course and amount, time on the right). Desktop column widened from 448 to 720 px.
- **Child profile still had a child chip row**, a second switcher the header switcher replaced (and the exact thing that did not scale). Removed.
- **Kid Home "Due"** rows were bare text; now the same tile + title + time row as "Latest", with the "N more" links as 44 px text links.
- **Disclosures** had no affordance or a different one each (Growth "Chart data", "About the reference data", Medication ended courses, Course "More times"). All now show the same chevron, which turns when open (`data-chev`, motion only if the user allows it).
- **"Delete this dose / course / entry"** were bold plain text. Now destructive outlined 48 px buttons that reveal the confirm.
- **Vaccines banner:** "Dismiss older vaccines" wrapped to two lines beside "Record doses"; the two buttons now wrap to full-width rows when they do not fit side by side.
- **Titles.** "Medication for Leo", "Milestones for Leo", "Export PDF for Leo" → "Medication", "Milestones", "Export PDF" like Timeline, Growth, Vaccines (the child is in the header switcher; the age line stays).
- **Ages.** Kid Home showed 4 months 12 days (hard-coded, a day behind) and a corrected age of 2 months 18 days; Timeline and Medication showed 4 months 13 days. Kid Home (0–11 mo band) now reads the child's birth date and weeks at birth like the other pages ("4 months 13 days", "Corrected age 3 months 11 days · born 35w 2d"). Growth's frozen "14 months 3 days" and "13 months 2 weeks" now count from its own sample birth date.
- **Focus ring** is one rule for links, buttons and summaries (2 px primary, 2 px offset).

**Accessible names changed on purpose**
- Page headings: "Medication for {child}" → "Medication"; "Milestones for {child}" → "Milestones"; "Export PDF for {child}" → "Export PDF".
- Medication: the course name link (name "{course}") → icon button "Edit {course}". Dose rows keep the name "{course}, {time}".
- Child profile: group "Child" (the chip row) removed.
- Dose / Course / Symptom delete reveal: same text, now a button-styled summary.
- New: landmarks "Main" (implicit) and "Primary" navigation.

**Not changed, on purpose or out of scope**
- (Resolved in section 18.) Growth, Vaccines and Milestones are built on their own sample data (a 14-month-old and a 9-month-old, from the age-band samples), so their age lines are correct for their data but differ from Leo on Home (4 months). The prototype's Age band and Milestones age simulators exist to show those states; making all pages share one child's data means regenerating the Growth rows and the Vaccines windows from that birth date.
- (Resolved in section 18.) Timeline's prototype panel has a Child toggle and no Role toggle, and its role badge is fixed to "Guardian". Prototype panel only; not part of the product screens.
- The Drawer hosts the form and Choose child pages in an iframe (prototype mechanism, see section 14).
- Admin has no navigation by design (separate sign-in surface).
- Screenshots were taken with a headless browser; the audit and axe numbers above are the evidence for target sizes and contrast, not the screenshots.

**Files.** Changed: `hbt-ui.js` (box-sizing, chevron, focus ring, `ageFrom`, `corrected`), `Kid Home`, `Growth`, `Vaccines`, and the generated `Medication`, `Milestones`, `Export`, `Child Profile`, `Course`, `Dose`, `Symptom`. New generator pass: `gen/sweep.py` (landmarks, chevron marks, text floor, button font), run last by `build_all.sh`.

## 18. Theme sheet, desktop side nav, one set of sample data (2026-11-24, fifth pass)

Requests: (1) fix the two items left open in section 17, (2) add "Change theme" to More with a Light / Dark / Auto sheet, (3) on ≥768 show the More entries in the side nav instead of one "More" item.

**Theme.** One preference, `theme` = light | dark | auto, stored with the other prefs (`HBT.pref('theme')`, default Auto). Every page derives its palette from `HBTUI.isDark(this)`; Auto follows `prefers-color-scheme` and re-renders live when the device setting changes (no reload), and a change on one page re-renders the others open in other tabs (`storage` event). Before this pass each page kept its own `dark` state, so a choice on one page was lost on the next; the Drawer pages were handed `&dark=1` and clicked the prototype panel. The hand-off is gone: the Drawer reads the same pref.
- "Change theme" is the last row of More on the phone (the row's note shows the current mode) and the last item in the side nav on ≥768. It opens a Sheet named "Change theme" with a RadioGroup (fieldset legend "Choose how Happy baby looks. It applies to every screen."): Light, Dark, Auto, each with a one-line explanation and a small swatch (decorative; the label is the signal, not the colour). A choice applies at once (live preview); "Done" (56 px) closes.
- Prototype panel "Theme" now has Light / Dark / Auto and writes the same pref, so the panel and the sheet stay in sync.
- No-JS: the theme control is a progressive enhancement. The row's link is `#theme`; without JS nothing can be stored, so the server-rendered pages follow the device setting (Auto) through `prefers-color-scheme` CSS. The prototype cannot show a no-JS render; this is the production requirement. For production also set `color-scheme` and the theme cookie server-side to avoid a flash of the wrong theme on first paint.
- `PDF Export` keeps its fixed light, greyscale print layout (a print artefact, not a screen).

**Desktop side nav.** ≥768 shows Home, Timeline, Growth, a hairline, then Vaccines, Medication, Milestones, Export PDF, Modules, Child profile, Settings, My profile, Change theme: every entry is one click (the "More" item and its sheet are no longer in the path, two fewer clicks). The phone keeps the tab bar with More and the sheet. The current page is marked (`aria-current="page"`, primary colour and a soft background) on the side nav and the phone tab bar; Course / Dose count as Medication and Milestone form counts as Milestones. Guardian-only entries (Modules, Child profile, Settings) are hidden for a caregiver, as before. Modules and Settings open their Sheet in place on Kid Home and link to `Kid Home?open=modules|settings` from other pages (Kid Home reads `open`). Both lists come from one function (`HBTUI.nav`), so the phone sheet and the side nav cannot drift.
- **Found while doing this:** the More list differed per page. Timeline, Growth and Vaccines listed "Medication (Soon)", "Settings (Soon)", "Customize trackers" and had no Milestones or Export PDF, while Kid Home and the generated pages had the full list. All pages now show the same list.

**Section 17 items.**
- *One set of sample data.* Growth, Vaccines and Milestones now read the child's stored birth date and weeks at birth, like Home, Timeline, Medication and Child profile. Leo (born 2026-07-11, 35w 2d) reads 4 months 13 days, corrected 3 months 11 days, on every page.
  - Growth: five measurements from birth to 4 months, dated from the birth date; before-term rows are listed but not scored, as before. The chart axis is now 0–6, 0–12 or 0–24 months by the latest point, so a young baby's points are no longer squeezed into the left fifth of a 24-month chart. The corrected-age line and "born 35w 2d" come from the profile.
  - Vaccines: the England schedule table lives once in `hbt-ui.js`. The default sample records: every dose whose window has ended is given, except Rotavirus dose 2, which stays overdue; 6-in-1 dose 3 and MenB dose 2 are due. Home's Due rows now read the same function; they were four hard-coded lines in mixed wording ("4-month vaccines"). The prototype Age and Records toggles gain "Stored" and "Sample" (default); the old demo ages still work.
  - Milestones: the default scenario is "Stored birth date" (before: "9 months", a different baby). The other scenarios stay as demo age bands. Sample marks moved to the 2- and 4-month lists (a 9-month mark on a 4-month-old read as a mistake). The non-judging rules and the word list are unchanged (test t7: no hits).
- *Timeline role.* The prototype panel has a Role toggle and the badge follows it (Guardian / Caregiver); the More list and side nav follow too.

**Accessible names changed on purpose**
- Side nav (≥768): "More" removed; new links Vaccines, Medication, Milestones, Export PDF, Modules, Child profile, Settings, My profile, Change theme.
- Phone More sheet: new row "Change theme" (with the current mode as its note). Timeline / Growth / Vaccines lose "Customize trackers", "Medication (Soon)", "Settings (Soon)" and gain the shared list.
- New dialog "Change theme" (group "Choose how Happy baby looks…", radios Light / Dark / Auto, button "Done").
- Primary tabs and side items carry `aria-current="page"` on the current page.

**Verification (what was and was not checked).** Headless Chromium only, light and dark, phone and ≥768: axe (wcag2a/aa/21aa + best-practice) clean on all 16 app pages × 4 states and on the open More and theme sheets of 9 pages; the layout audit (targets, names, overflow, type floor) flags only Admin (no navigation, by design) and chart ticks at 11 px; the theme choice was exercised on all 15 pages that have navigation (select, persist across reload, Auto with a dark and then a light emulated device setting, a change in one tab re-rendering a second tab, Drawer in Auto + dark); logic tests t1–t8 pass (t2 keeps its older "cards 3" failure). Not checked: real devices, Safari and Firefox, screen readers, the no-JS render (stated above, not demonstrable here).

**Files.** `hbt-ui.js` (theme, nav model, vaccine table and sample records, radio row, Drawer no longer passes `dark`), `hbt-ms.js` (default scenario, seed marks), `Kid Home`, `Timeline`, `Growth`, `Vaccines`, every generated page via `gen/common.py` and the new idempotent pass `gen/theme.py` (run before `sweep.py` in `build_all.sh`).

## 19. Legacy parity pass (2026-11-24, sixth pass)

Source: the legacy SvelteKit/Go app as implemented in PR 13 (418 screenshots, published under “Legacy design” in the uploads). Every legacy route, flow and refusal string was compared with the prototype; the table of 27 differences was delivered first, then the recommended rows were applied. “Legacy” means the implemented app, “new” means this prototype.

**New pages (outside the app shell: no tabs, bell or switcher, so a failing API cannot break them).** Generated by `gen/auth.py`, not touched by the normalize, chrome, theme or sweep passes.
- `Sign In.dc.html`. Username and password, 48 px inputs, 52 px button. One generic “Incorrect username or password” for both cases, so it never says which was wrong; the username stays and the password clears. Blank fields are refused on the field (“Enter your username”, “Enter your password”). Rate limit and server-down have their own sentence. No “show password” toggle and no “forgot password” link (a guardian or the administrator issues a reset link). No-JS baseline: a form POST to `/login`; a refusal returns the same page with the message and the username kept.
- `Set Password.dc.html`. One page for the invite link (72 hours) and the reset link (1 hour). Username shown read-only so a password manager saves the right account. Under 12 characters is refused on the field; a mismatch says so on the confirm field; a rate limit shows an alert above the form. Used, expired and unknown links all read the same (“This link is no longer valid”) and show no form. Invite signs the parent in and lands on the welcome state; reset signs the other devices out. No-JS baseline: a form POST to `/invite/[token]` or `/reset/[token]`.
- `Error.dc.html`, 404 and 503. 404 is one page for an unknown child, a child without access, a path that does not exist and a parent opening an admin address, so it never reveals that a child exists. 503 keeps the session and offers Try again.

**Changed.**
- Sign out. A last entry in the shared nav model (`hbt-ui.js`): in the phone More sheet (note “This device”) and at the bottom of the desktop side nav. It links to Sign In. Legacy had a global Sign out; the prototype had none for the current device. Real build: a form POST to `/logout` that clears the local session even when the API cannot be reached.
- Kid Home, Leave this child. A failure (offline or server error) now shows “Couldn’t leave.” with the reason inside the sheet, and the sheet stays open. Leave stays caregiver-only, as in legacy, so the last-guardian refusal belongs to Access, not here.
- Growth. The save refuses an empty date, a date in the future and a date before the birth date (the old code clamped silently to age 0). Head circumference is unscored after 60 months with WHO and after 36 months with CDC, with the existing neutral reason “this source has no table for this age”.
- Medication. When the module is switched off for the child, the page shows “Medication is switched off for {child}”, hides Add a course and the course list, keeps data, and offers Open modules to a guardian only (a caregiver reads “A guardian can switch it on”). Sim panel gets a Medication module On/Off toggle.

**Checked and already covered (no change).** Unknown username, duplicate username, sole guardian, wrong current password and mismatched passwords (`hbt-acct.js`); Export range limit (93 days), start before birth and “Choose at least one module” (`hbt-ms.js`); Milestones already-marked and cap-reached; nursing and sleep second-timer race (`already_running`); a stale ntfy topic (the New topic action replaces it). Timeline has no free From and To inputs (a 7-day strip with Older paging), so a reversed range cannot occur.

**Not done, on purpose.** Legacy cursor links and the legacy date-range inputs; per-child measurement and vaccination caps beyond the dose cap already shown; dark mode and JS-off screenshots do not exist in legacy, so those states have no legacy counterpart to compare.

**Verification.** Headless Chromium: axe (wcag2a/aa/21aa + best-practice) clean on 19 app pages × light/dark × phone/≥768 (the PDF page is print-only and has no app frame); new pages clean in every state (blank, wrong password, rate limited, API down, success; short, mismatch, rate limited, invalid link, success; 404, 503); targets ≥ 44 px; logic tests t1 to t8 pass; side nav shows Sign out on all 12 pages with the Sign out row at 44 px. Not checked: real screen readers, real devices, the real POST endpoints.

**Files.** `Sign In`, `Set Password`, `Error` (new), `hbt-ui.js`, `Kid Home`, `Growth`, `Medication`, `plan-7-design-notes.md`; generator `gen/auth.py`.

## 20. Export pass (2026-11-24, seventh pass): corrections found while writing the system

Writing the reusable design system and the repo export meant reading every control back out of the prototype's source and measuring it. That surfaced one real defect and several inconsistencies; all were fixed in the prototype, not only in the documents.

**Fixed.**
- Toast action contrast. In dark, the toast surface is the light foreground, but the action colour (`--toast-link`) was still the light amber: 1.78:1. Dark now uses `oklch(0.45 0.15 38)` (a fixed dark rust). All 16 pages that define the token and `gen/common.py` carry it.
- Input text size. Inputs, selects and textareas were 15 px on some pages and 16 px on others; all are 16 px now (mobile browsers zoom a focused input below 16 px). `hbt-ui.js` control layer sets it as well.
- Invalid-field border. Sign In, Set Password and Error used a 2 px foreground border; everything else used the destructive colour. All use 1 px destructive now (`gen/auth.py` too). The message in words under the field is unchanged.
- Content column. List and detail pages are 720 px, forms 448 px (`hbt-ui.js` caps direct children of `[role=main]` that set no max-width at 720). Milestones was 672 and is 720. Kid Home uses the common padding `16 16 28` and gap 18.

**Deliverables.** `design-system/` (core rules, components contract, patterns, verification, brand preset, tokens as Tailwind 4 CSS and JSON, 32 icons, fonts) as plain files, and the same content as a Design System artifact. The export adds per-page specs, a route map, the full state matrix of screenshots with per-state layout data, a generated Playwright visual harness and `IMPLEMENTATION.md`.

**Verification.** Headless Chromium with the real Geist fonts (earlier passes measured with the fallback font; sizes and contrast were re-checked). Not checked: real devices, Safari and Firefox, screen readers, the no-JS render.

**Addendum (same pass).** Choose Child: the back link was 43.5 px wide; it now has `min-width:44px`. Found by the layout audit of the exported specs (`tools/audit_layout.py`).

## 21. Wide two-column layout and three standalone pages (2026-11-24, eighth pass)

Requests: (1) "make the next breakpoint 1024 px; anything further than that has a fixed-width two-column layout, including lists and forms", (2) add the screens that are missing, checking that they really are missing ("/kids" is the no-JS fallback of the Children switcher in the floating panel).

**Decisions (owner, through questions).**
- Two columns means a **main column plus a context column**: left = the task, right = what supports it.
- **Breakpoint 1280, not 1024.** The owner asked whether 1280 fits better; it does (at 1024 the 225 px side nav leaves 799 px, less than the 920 px block of two 448 columns and a 24 px gap). Confirmed. 768 to 1279 keeps today's single column (720 lists, 448 forms).
- From 1280 the content block is a fixed **920 px** (448 + 24 + 448), centred beside the side nav; wider screens get empty margin.
- Forms: **two groups of fields.** Left = primary fields (what, when); right = optional fields and notes; Save/Cancel under the left column. No content was invented.
- New standalone pages: **Add a child** (`/kids/new`), **Edit details** (`/kids/[kid]/settings`), **Access** (`/kids/[kid]/access`). Add a child and Edit details share one field set (name, birth date, sex optional, born early with weeks and days, time zone defaulting to the device's) with the sheet. **Choose child** (`/kids`) already existed as the switcher's no-JS page. **Modules** was not selected: the nav entry still opens the sheet and its no-JS page is not designed.

**Implementation in the prototype.** `data-w` marks (`grid`, `l`, `r`, `grid2` with `fa`, `fb`, `ff`, `form`) are `display: contents` below 1280, so phone and 768 to 1279 are byte-for-byte the previous layout; at ≥ 1280 they form the grid. Source order is reading order and tab order (the left column is always earlier in the DOM). Prototype panel, Layout group: a third button, "≥1280 two-column" (`?wide=1` starts there; the drawer iframes stay on the phone layout).

**Known limits.** Vaccines, Admin and Choose child have a much shorter right column; Notifications' right column is empty when ntfy is hidden. Timeline puts its controls on the left and the day list on the right because that is the DOM order. The sheet shows the Sex group first while the standalone pages put it in the right column.

**Verification.** Headless Chromium, real Geist, every documented state in six variants (phone, 768 to 1279, ≥ 1280, each light and dark): axe (wcag2a/aa/21aa + best-practice) and the layout audit (targets ≥ 44 px, overflow). Not checked: real devices, Safari and Firefox, screen readers, the no-JS render.
