# Design system: core rules

Version 1.0 (2026-11-24). Source: the Happy Baby Tracker design prototype (20 pages, 4 render variants each). This file is the **core layer**: structure, interaction, accessibility, state and copy rules that hold in every project that adopts the system. Anything visual that is brand-specific (palette, typeface, tracker hues, radii usage) lives in the **preset layer**: `preset-happybaby/BRAND.md` and `tokens/tokens.css`. To start a new product, keep this file and replace the preset.

How to read the rules: **MUST** is checked in review and by the verification matrix (section 12). **SHOULD** is the default and needs a written reason to depart. **MAY** is optional. A rule marked *(owner decision)* was an explicit choice by the product owner, not an inference; do not reopen it without the owner.

## 0. Layers and files

| Layer | File | Changes when |
|---|---|---|
| Core rules | `DESIGN.md` (this file) | the product's interaction model changes |
| Core patterns | `patterns/PATTERNS.md` | a new state or flow pattern is added |
| Component contract | `components/COMPONENTS.md` | a part is added or its sizes change |
| Preset | `preset-happybaby/BRAND.md`, `tokens/tokens.css`, `tokens/tokens.json`, `icons/` | the brand changes; regenerate tokens, never hand-edit values |
| Verification | `VERIFICATION.md`, `tools/capture.py` | the page or state inventory changes |

Target stacks, equivalent by construction: **SvelteKit 5 + shadcn-svelte (bits-ui) + Tailwind 4 + lucide**, and **React + shadcn/ui + Tailwind 4 + lucide-react**. Both read the same `tokens.css` (shadcn variable names), so a token change is made once.

## 1. Principles

1. **Calm and neutral.** The product is used one-handed, tired, often at night. No celebratory motion, no streaks, no scores that grade a child or a parent. Copy states facts and the next action. Developmental content never judges (section 9.4).
2. **Colour is never the only signal.** Every state that colour marks also has a word, a shape or a position (pressed chip = fill + border + leading check; error = words + 1.5 px border; reached/unreached = a text line).
3. **Small, stable surface.** One primary action per screen or sheet. One pattern per job: one way to add, one way to filter, one way to switch child, one way to confirm a destructive action.
4. **Server is the source of truth, JavaScript is an enhancement.** Every interaction has a stated no-JS fallback that works by navigation or a form POST. The enhanced behaviour (Drawer, toast, live theme) must not change what the server accepts or refuses.
5. **One definition, many views.** Data, ages, copy, navigation lists and tokens each have one definition that every screen reads (one nav function feeds phone sheet and desktop side nav; one age function feeds every age line). Two hand-written copies of the same thing are a defect.
6. **Failure is a designed state.** Every write has an offline, server-error and conflict outcome with fixed wording (section 8). Every page has an empty state, a not-found state and a "can't reach the server" state.
7. **No hidden truth.** A fact is never available only through colour, hover, long-press or a tooltip.

## 2. Platform and layout

- **Three layouts.** Phone (< 768 px, designed at 390 px), side-nav (768 to 1279 px, designed at 980 px) and wide (≥ 1280 px, designed at 1280 px, §2.1). At ≥ 768 the primary navigation moves from a bottom tab bar to a left side nav; between 768 and 1279 content is one column, left-aligned in the content area: **720 px at most** for list and detail pages, **448 px** for full-page forms (the pages behind Drawers, Notifications, Export, Choose child). The side nav is 225 px wide (200 content + 12 px padding each side + 1 px border); the main area takes the rest. At ≥ 1280 the content becomes a fixed two-column block (§2.1); there are no fluid multi-column dashboards.
- **Frame.** `main` is the scroll container (`role="main"`); the tab bar or side nav is `role="navigation"` named "Primary"; only one of the two is in the accessibility tree at a time.
- **Safe areas.** The tab bar has 10 px bottom padding plus the platform inset; sheets use `padding: 10px 16px 20px` plus the platform inset.
- **Page padding.** `main` has padding `16 16 28` on phone and wide. Vertical rhythm: 18 px between sections on every page (the same value on Home as elsewhere), 12 px between items in a card, 6 px label-to-control. Pages outside the shell (sign in, set password, errors) use padding `24 20` and centre their card.
- **Text scales.** The layout MUST survive 200 % text zoom without horizontal scroll: containers are flex/grid with `min-width: 0`; names use `overflow-wrap: anywhere`; the only intentionally truncated text is the child name in the header switcher (ellipsis, full name in the accessible name).

### 2.1 Wide two-column (≥ 1280 px) <!-- wide:v1 -->

*(owner decision, 2026-11-24: "Make 1024px the next breakpoint; anything further than that will have a fixed-width 2-column layout, including lists and forms.". Breakpoint set to **1280** after the owner asked whether it fits better and confirmed. Meaning of "two columns": a main column plus a context column.)*

- **Three layouts, not two.** Phone (< 768, designed at 390), side-nav (768 to 1279, designed at 980, one column exactly as before) and wide (≥ 1280, designed at 1280). Nothing changes below 1280.
- **Fixed block.** At ≥ 1280 the content block beside the 225 px side nav is **920 px** wide (448 + 24 gap + 448), centred in the main area. It does not grow above 1280; extra width is empty margin. Columns are 448 px; row gap 18, column gap 24. `main` padding is unchanged.
- **Left = the task, right = context.** The left column holds what the person came to do (the form, the list, the current state); the right column holds what supports it (optional fields, history, people, help). **Source order = reading order = tab order:** the left column is always earlier in the DOM; the layout never reorders with `order` or `grid-area` tricks. The one exception is Timeline (controls left, results right) because its DOM already has the controls first.
- **Full-width children.** A child that is not assigned a column spans both (alerts, banners, page-level empty states).
- **Forms.** Every form splits into two field groups: **left = primary fields** (what, when), **right = optional fields and notes**, and the Save/Cancel row sits **under the left column**. In the DOM and in tab order Save/Cancel comes **last**, after the optional fields (a keyboard user passes every field before Save), although it is drawn under the left column. No field is added or removed by the split; below 1280 it is the same single column in the same order. The same rule applies to the drawer's page fallback.
- **Drawers and sheets** keep their width (480 max); they are not split.
- **Pages outside the shell** (sign in, set password, errors) stay one centred card.
- **Implementation.** `@media (min-width: 1280px)` (Tailwind 4 default `xl:`; token `--bp-wide`). Wrap the content in a `grid xl:grid-cols-[448px_448px] xl:gap-x-6 xl:gap-y-[18px] xl:w-[920px] xl:mx-auto` container; mark column wrappers `display: contents` below 1280 so the DOM is one flat column there. Forms use `xl:grid-rows-[auto_1fr]` with the optional group spanning two rows. Never use a container-width media query that depends on the side nav being present.

Column map (left = main, right = context):

| Page | Left | Right |
|---|---|---|
| Home | quick log, latest entries | due list, today |
| Timeline | last-7-days summary and filters *(exception: controls first in the DOM)* | the day-by-day entries |
| Growth | weight-for-age chart, measure tabs | measurements table |
| Vaccines | status, banner, record a vaccination, visits | also available, every season, other |
| Milestones | the lists, over-5 note, retired | screening note, your milestones |
| Medication | courses (and the module-off card) | recent doses |
| Child profile | details, settings | people with access, delete |
| My profile | details, password | devices, children |
| Notifications | push on / off | ntfy, no-channel note |
| Choose child | search, list, none | add a child / caregiver note |
| Admin | filter, empty state, rows | footnote, sign out |
| Course | name, dose, schedule + Save | start, last day, active, note |
| Dose | given at, amount + Save | note |
| Milestone form | date reached + Save | note |
| Symptom | time, symptom, severity + Save | temperature, method, note |
| Export | date range + Download | modules to include |
| Add a child | name, birth date, time zone + Add child | sex, born early (weeks and days) |
| Child settings | name, birth date, time zone + Save | sex, born early, overdue threshold |
| Access | people with access | add a person |

Known imbalances (accepted, not bugs): Vaccines, Admin and Choose child have a much shorter right column; Notifications' right column is empty when the ntfy channel is hidden. Do not pad them.

## 3. Navigation

### 3.1 Primary
- **Phone:** bottom tab bar, four items: Home, Timeline, Growth, More. 22 px icons, 12 px labels (floor). Current item: `aria-current="page"`, primary colour.
- **Side nav (≥ 768, including wide):** Home, Timeline, Growth, a hairline, then every More entry inline, so each is one click. "More" does not exist at ≥ 768. *(owner decision: two fewer clicks.)*
- **One source of both lists.** A single function returns `{ more[], side[] }` from one entry table, so the phone sheet and the side nav cannot drift. Entries, in order: Vaccines, Medication, Milestones, Export PDF, Modules*, Child profile*, Settings*, My profile, Change theme, **Sign out** (last). *Guardian only.
- **Module on/off.** An entry for a module that is switched off for the child is hidden only where the owner decided so (open question for Medication in this product; see the preset notes). A page reached directly for a switched-off module MUST show its off state (section 8.5), never an empty list.
- **Current page.** Child routes of a section count as the section (a Course/Dose form highlights Medication; the Milestone form highlights Milestones).
- **Sign out.** Last entry in More and in the side nav, note "This device". Production: a form POST to the sign-out route that clears the local session even if the API cannot be reached.

### 3.2 Global header (every screen except print layouts)
One 48 px row built once and shared. Left: **child switcher**. Right: **notification bell** (48 px icon button to notification settings; settings link, not an inbox; no unread dot; `aria-current="page"` on its own page) and the **role badge**.
- Child switcher: secondary button with the child's initial, name (one line, ellipsis) and chevron; accessible name "Child: {name}. Switch child". It replaces child chips and back links on module pages. Opens a Drawer (section 4.1) with one 64 px row per child (initial, name wraps, age, check + `aria-current="true"` on the current one). Above six children a "Find a child" field appears ("No child matches" empty state). Picking saves the choice and returns to the same module for that child; the return target is whitelisted to the child pages and defaults to Home. Guardians see "Add a child"; caregivers see "Ask a guardian to add a child."
- Back links exist only where there is no switcher: the full-page fallback forms ("‹ Medication for Leo") and account pages ("‹ Home", "‹ Account"). The "‹" is decorative (`aria-hidden`); the accessible name is the label alone.
- The Drawer-hosted copy of a page hides the header, tabs, back link, title and age line (section 4.1).

### 3.3 Title line
A page about a child: `h1` (30 px) then the child's age line (muted). Account pages (My profile, Notifications, Admin): `h1` then a 1 px hairline instead of an age line. Module titles are the module name only ("Medication", not "Medication for Leo"): the child is in the header. *(owner decision.)*

### 3.4 Pages outside the shell
Sign in, set password (invite and reset), 404 and 503 render with no tabs, bell or switcher, so a failing API cannot break them. Admin is a separate sign-in surface with no navigation.

## 4. Forms and sheets

### 4.1 Drawer first, page as the no-JS route *(owner decision)*
Every "Add" or "Edit" of a record is a link with a real `href` to a form page and a `data-drawer="Title"` attribute.
- **With JS:** a click opens a bottom Drawer (92 % max height, handle, close button, Escape, overlay tap, focus returns to the trigger) and renders the same page inside it in embed mode. Saving closes the Drawer, refreshes the list and shows the result as a toast; Cancel closes without a toast.
- **Without JS, or a modified click (new tab):** the link navigates to the full page. The form posts, and the response is a redirect with the message.
- Settings screens (Child profile, Notifications) are pages, not Drawers: they are not add/edit of a record.
- Production mapping: SvelteKit intercepted/parallel route with a Drawer layout; the direct visit renders the same route at full size. (The prototype uses an iframe because each page is static; do not copy that.)

- **Standalone pages (owner decision, 2026-11-24).** Add a child (`/kids/new`), Child settings (`/kids/[kid]/settings`) and Access (`/kids/[kid]/access`) have their own pages, the no-JS routes of the sheets of the same name. Add a child and Child settings share one field set and one validation function with the sheet (name, birth date, sex optional, born early with weeks and days, time zone defaulting to the device's; Child settings adds the overdue threshold). Choose child (`/kids`) is the no-JS page of the header child switcher. The Modules sheet has **no designed page**; its no-JS route is not specified (flag it before building).

### 4.2 Sheet anatomy
Bottom sheet, max width 480 px, radius `20 20 0 0`, handle 36 × 4 px (radius 2), title 18/600, close button 44+ px with name "Close", content gap 14 px, **Done/Save 56 px** (large button), Cancel beside it. Overlay `rgba(20,12,8,.5)`. Opening a sheet moves focus into it; the trigger's name is stable.

### 4.3 Validation and failure
- Field-level messages sit under the field, `role="alert"` is not used per field; the field gets `aria-invalid="true"` and `aria-describedby` to its message; the message uses words (never colour alone), 12.5 px/600.
- A form-level summary alert ("Check the fields with a message.") appears above the fields when any field is refused.
- A **failed save** (offline/server) keeps the sheet open, keeps every value, and shows an alert "Couldn't save. {reason}" inside the sheet. A toast is used only when no sheet is open. Retry is the same button.
- **Conflicts** (another phone changed the thing) show the fixed sentences in 8.2 and refresh the list in place.
- Dates: never in the future beyond a 5-minute tolerance; never before the birth date; error words state the rule ("The date can't be in the future.").
- Passwords: at least 12 characters; mismatch is refused on the confirm field.
- `novalidate` on forms so only the product's messages show, never the browser's bubbles.
- Inputs, selects and textareas: 16 px text (prevents iOS focus zoom), 48 px high (44 px for the narrow inline weight fields), radius 8, `0 12px` padding, 1 px `--border`, `--background` fill, shadow `0 1px 2px rgb(0 0 0 / .05)`. Focus: border becomes `--primary` and a 3 px ring (`--primary` at 32 %) is added. `aria-invalid="true"` switches the border colour to `--destructive`; the message beside it carries the meaning.

### 4.4 Choices
- **Filters and multi-select = ToggleGroup chip** (44 px, radius 8, pressed = soft fill + primary border + leading check, `aria-pressed`). Used by Timeline filters, Export modules, Admin filters and every option group. The old pill is retired. *(owner decision.)*
- **Single choice in a sheet** (theme, role, units) = RadioGroup rows (20 px radio, label + one-line description). **Booleans** = Checkbox (20 px, accent = primary).
- **Native elements under the styling layer.** The control layer styles native `input/select/textarea`, so the no-JS pages look identical and no markup depends on JS.

### 4.5 Destructive actions
Destructive and irreversible actions keep a **text label** ("Remove", "Remove access", "Delete this dose") and reveal a confirm: card fill, 1.5 px destructive border and text, consequence sentence first ("Entries stay and keep their name."). Deleting a child is two steps (consequence list, then type the name + tick). Undo (toast action) is offered for reversible edits (role change, remove, add, mark/unmark), not for permanent deletion.

## 5. Components and sizes (summary; the contract is `components/COMPONENTS.md`)

| Thing | Size | Notes |
|---|---|---|
| Minimum target | 44 px | every tappable thing, rows, links, summaries, chips |
| Button default | 48 px, radius 12, 15/600 | `box-sizing: border-box`, border included in the 48 |
| Primary action | 52 px on standalone pages (auth), 48 elsewhere | one per screen/sheet |
| Large (sheet submit + Cancel) | 56 px, radius 14, 16/600 | |
| Icon button | 48 × 48, stable accessible name + `title` | repeated non-destructive row actions: Edit (pencil), Choose a date (calendar) |
| Chip (ToggleGroup) | 44 px, radius 8 | |
| Card | 1 px border, `--card`; radius is **tiered by role**: 12 for item cards in dense lists (Milestones, Vaccines), 14 for summary and row cards (Home, Timeline, Growth, Medication), 16 for form, settings and section cards and for disclosures | content gap 10-12 |
| Badge | 12/600, radius 99, `3px 8px`, 1 px border | |
| Tracker chip | 38 px by default (Timeline rows, Home tiles); 36 in Home latest and due lists, 40 on Timeline running cards, 44 on Home running cards; radius 12 always | tracker hue (preset) |
| Text floor | 12 px | only chart tick labels are 11 px |

**Button roles.** Primary: filled, one per screen. Secondary: card fill + 1 px border. Destructive confirm: card fill + 1.5 px destructive border and text. Page-level actions are text buttons; repeated row actions are icon buttons (the one exception to "text label").

## 6. Lists, cards and disclosure

- **Visit-style cards.** A list of grouped records (vaccines by visit, milestones by age) is one card per group with a chevron header; the current group opens by default; each item is an inner row/card. The chevron is the same everywhere and rotates when open (motion only if the user allows it). Disclosure summaries are ≥ 44 px.
- **Rows** (timeline, doses, history): tile + title and detail on the left, time on the right; a flat hairline between rows. A row opens an Edit Drawer.
- **Empty state = an invitation.** One sentence of what is empty and the action that fills it ("Nothing logged yet" + the add control). Never a blank panel.
- **Pagination.** "Older" link/button at the end of a time list; the first page is always the newest.

## 7. Theming

- One preference, `light | dark | auto`, default Auto. Auto follows `prefers-color-scheme` live (no reload) and across open tabs (`storage` event).
- Theme sheet: a RadioGroup (Light, Dark, Auto) with one-line descriptions and a decorative swatch; a choice applies at once (live preview); Done is 56 px.
- Mechanism (`tokens/tokens.css`): the `<html>` element carries `data-theme="light|dark|auto"` (server-rendered from a cookie, so it works with no JS) and `data-resolved="light|dark"` (set by an inline head script **before first paint**; for `auto` it reads `prefers-color-scheme`). Dark values apply for `[data-theme=dark]` or `[data-resolved=dark]`; a `prefers-color-scheme: dark` media block is the fallback when neither attribute is set or JS is off. There is no `.dark` class: the Tailwind `dark:` variant is remapped to these attributes with `@custom-variant dark`.
- Both themes MUST pass the same contrast bar (text ≥ 4.5:1, large text and UI boundaries ≥ 3:1); see the table in `preset-happybaby/BRAND.md`. A colour that is used on an inverted surface (toast) is checked on that surface.
- Print layouts (the PDF) are fixed light, greyscale, A4: they are print artefacts, not themes.

## 8. Feedback and states

### 8.1 Feedback components
| Kind | Anatomy | Use |
|---|---|---|
| **Toast** | inverse surface (`--foreground` background, `--background` text), radius 12, 14 px, shadow menu, `z` above sheets, auto-dismiss 6 s (9 s for a failure that offers Try again, announced as `role="alert"`; others `role="status"`); actions are 44 px: Undo, Try again, View, "Add weight after" | result of a write when no sheet is open |
| **Alert** (error) | `role="alert"`, 1.5 px solid `--foreground` border, `--background` fill, radius 12, 13.5 px, bold lead ("Couldn't save.") | a failure the user must read; inside the sheet or page where it happened |
| **Status** | `role="status"`, same box, muted | a neutral outcome ("Marked as reached: ...") |
| **Field message** | under the field, 12.5/600, words | a refused field |

Toast action colour is its own token (`--toast-link`) because the toast surface is inverted: on a light-theme toast (dark surface) it is light; on a dark-theme toast (light surface) it is dark. Never reuse `--primary` there.

### 8.2 Write outcomes (fixed wording)
| Outcome | Sentence |
|---|---|
| Offline | "No connection, so this wasn't saved. Try again when you're back online." |
| Server error | "Something went wrong on our side and this wasn't saved. Try again in a moment." |
| Already running (another phone started it) | "Someone else already started this on another phone. It's shown now." |
| Already ended | "Someone else already ended this on another phone. It's updated now." |
| Deleted by someone else | "Someone deleted this entry, so it can't be saved. It's been removed from your list." |
| Rate limited | "Too many attempts, try again in a minute" |

Offline and server error MUST be distinguishable only by cause and keep the typed values. Optimistic UI is not used for writes whose refusal is possible; the control shows its busy state until the server answers.

### 8.3 Page states
Every page defines: **normal**, **empty**, **module off**, **not found** (one page for unknown, no grant and wrong role, never revealing existence), **can't reach the server** (503: session kept, "Try again"). Pages outside the shell have their own copies.

### 8.4 Authentication outcomes
Wrong username or password is **one** message ("Incorrect username or password"), username kept, password cleared. Blank fields are refused on the field. Used, expired and unknown links are one message ("This link is no longer valid. Ask the person who sent it for a new one.") with how long that kind of link lasts. No "show password" toggle, no "forgot password" link (a guardian or admin issues reset links).

### 8.5 Module off
The page shows "{Module} is switched off for {child}", states that data is kept, hides add controls and lists, and offers "Open modules" to a guardian only (a caregiver reads that a guardian can switch it on).

### 8.6 Last-guardian and access rules
A child always has at least one active guardian. The check is made on the server inside the change; the UI states the rule where it would apply ("A child needs at least one guardian.") and does not offer the action to a caregiver. Leave is caregiver-only.

## 9. Copy and voice

1. Sentence case everywhere, plain verbs, no filler, no exclamation marks, no apologising ("Couldn't save." not "Sorry!").
2. Actions say what happens ("Save changes", "Log dose now"). **An action keeps one name through the whole flow** (a button "Publish" produces a toast "Published").
3. Name things by what the user knows, not how the system is built ("Notification settings", not "ntfy topic" except where the user must type it).
4. Errors say what happened and what to do, in the interface's voice.
5. **Developmental content never judges.** Banned in milestone/growth wording: late, behind, delayed, delay, normal, abnormal, typical, atypical, on track, ahead, missed, missing, concern, worry, should, red flag, warning. No colour, status icon, check, count or progress on a milestone. Reached and unreached differ only by the "Reached on {d Month yyyy}" line. Required reference sentences and attributions stay visible. A test asserts the denylist.
6. Typographic apostrophes (’) in prose; straight only in code.
7. Numbers: age as "4 months 13 days" (corrected age on the next line, "Corrected age 3 months 11 days · born 35w 2d"); times in the child's time zone, stated once on the page ("Times are in America/Los_Angeles").
8. Names of people and children are user data: they wrap anywhere, never truncate except the switcher button.

## 10. Accessibility contract

- **Targets** ≥ 44 px; buttons 48; sheet Done 56. Rows and summaries included.
- **Names are stable.** Accessible names are part of the test surface (e2e uses them). Rename only on purpose and list every rename in the release notes (the notes file's "Accessible names changed on purpose" section is the model).
- **Focus** is one rule: `:focus-visible { outline: 2px solid var(--primary); outline-offset: 2px }`; form controls instead turn the border `--primary` and add a 3 px ring (section 4.3). Focus moves into a sheet on open and returns to the trigger on close.
- **Landmarks** `main`, `navigation "Primary"`, one `h1` per page, headings in order, no duplicate ids.
- **Colour** is never the only signal; contrast per section 7.
- **State exposure:** toggles use `aria-pressed`; the current item `aria-current`; disclosures use native `details/summary`; errors `aria-invalid` + `aria-describedby`; alerts `role="alert"`, status `role="status"`.
- **Motion:** nothing moves on its own. User-triggered motion (chevron turn, sheet open) is allowed and MUST respect `prefers-reduced-motion`.
- **Zoom/reflow:** 200 % zoom, no horizontal scroll; 390, 980 and 1280 px all axe-clean.

## 11. Data, units and time (product-neutral rules)

- One store defines entries, ages, units and wording; screens never recompute an age by themselves.
- Ages count from the birth date; for children born before 37 weeks a corrected age is shown on the next line and used for growth scoring and milestone lists, with the source stated.
- Units are a per-account preference (kg·cm / lb·in; ml / oz) and the same value is shown in every place; entries store canonical units.
- Reference data (growth percentiles, vaccine schedule, milestone catalog) is labelled with its source and retrieval date and never claims to be a diagnosis. Values outside a reference's range are listed and not scored, with the neutral reason "this source has no table for this age".

## 12. Verification matrix (the definition of done)

Every page, in every documented state, in **six variants**: phone-light, phone-dark (390 px), desktop-light, desktop-dark (980 px, side nav), wide-light, wide-dark (1280 px, two-column).
1. **axe** (wcag2a, wcag2aa, wcag21aa, best-practice): zero violations, on every state that has a screenshot, not just the default.
2. **Targets** ≥ 44 px (audit script), text floor 12 px.
3. **Names**: the accessible-name list is diffed against the previous release; each difference is intentional and recorded.
4. **No-JS**: every interaction has its navigation/POST route and the same refusals.
5. **Fonts**: verification runs with the product font loaded (a fallback font changes wrapping and masks overflow).
6. **Copy**: the denylist test, the fixed failure sentences, the neutral-wording test.
7. Not covered by automation (state it in the release notes): screen readers, real devices, Safari/Firefox, the no-JS render.

`tools/capture.py` produces the state matrix and the per-state axe results; `VERIFICATION.md` explains how to run it on a new project.

## 13. Prototype-only things that MUST NOT ship

The prototype simulation panel (Next save, Theme, Layout, Role, scenario switchers), the "Prototype only" boxes, sample data, the iframe Drawer host, and the 34 px segment controls. They exist to demonstrate states.
