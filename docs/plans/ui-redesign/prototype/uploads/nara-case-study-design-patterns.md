# Reference: Nara baby tracker case study — reusable design patterns

Source: https://everydayindustries.com/casestudy/mobile-app-ui-design-case-study/ (Everyday Industries case study of the Nara Baby & Pregnancy Tracker, iOS/Android, launched early 2019). Read 2026-11-22: the article text, plus 9 screenshots of the page supplied by the user (section 5b). Paraphrased; the page is a marketing case study, so the text is thin on specifics. Section 6 lists what neither text nor screenshots show.

How to use this file: every future session on this artifact should read it before adding a screen. Each pattern has a **Status** (done / candidate / conflicts) against this project's rules (neutral copy, no diagnosis, no-JS baseline, shadcn-svelte first, 44px targets, axe-clean, colour never the only signal).

## 1. Research findings that shaped the product (reuse as design rationale)

| # | Finding | Design consequence | Status here |
|---|---|---|---|
| R1 | Parents focus on the present; what they track shifts as the baby grows (feeding and diapers early, sleep later). | Home leads with "now"; tracker set adapts to age. | **Done**: age bands change the Log tiles and order (Kid Home). Keep every new module band-aware. |
| R2 | Tracking is reassurance: logging is a grounding act, not just data capture. | Logging must feel finished and calm: instant confirmation, no friction, no guilt. | **Partly**: undo toast exists. Candidate: a quiet "Logged" state on the tile itself. Reassurance comes from the *act*, never from interpreting values (handoff rule: no judging). |
| R3 | Data is a long-term reference; parents return to spot patterns and decide. | History and trends are first-class, not an export. | **Candidate**: Timeline + Trends pages (Timeline page added; trends = section 3). |
| R4 | Competitors lose on clutter and complexity. | Simplicity over feature density; one primary action per screen. | Applies to every page; Vaccines page hides settled visits for this reason. |

## 2. Principles stated by the case study

1. Simplicity over feature density.
2. Design for tired eyes: low light, late night, one hand. → large targets, large inputs, real dark mode.
3. Scannability and fast retrieval of recent information.
4. Flexibility: the app adapts as the baby grows (switch trackers on/off, reorder).
5. Shared use: several caregivers, on different phone platforms, stay in sync.
6. Brand expression that is playful but adds no friction.

Mapping: 1–3, 5 are already project rules. 4's "reorder" is new (see P7). 6 is limited here by the neutral-tone rule: playfulness may live in colour, shape and small illustrations, never in copy about health values.

## 3. UI and interaction patterns (comprehensive list)

*Screenshot cross-reference: P2/P3/P9/P18 → V1; P10 and the per-card summary idea → V2; P11–P15 → V3; P16 → V4 (a time-of-day week grid, stronger than a count strip); P17 → V5; P18–P22 → V6. Read 5b for specifics before building any of these.*

**Navigation / information architecture**
- **P1 Three-part IA: Activity (today) · History · Trends.** Home answers "what do I need now"; History is the raw record; Trends is the pattern view. *Here:* Home / Timeline / Growth / More. Growth is the only trend view today. *Candidate:* add a Trends page (sleep totals, feeds per day) reachable from Timeline; keep the 4-tab bar.
- **P2 "Today" as the home anchor.** The home is scoped to today, with totals for the day. *Done:* Today grid on Kid Home.
- **P3 Most recent entry always visible, as elapsed time since the last sleep/diaper/feed.** *Done:* tile subline "45 min ago" (`since()` on Kid Home). *Candidate:* live elapsed on every tile, "time since last" repeated in the Timeline header.
- **P7 Customise trackers: on/off and reorder.** *Done:* on/off (Customize trackers sheet). *Candidate:* reorder. Reorder needs a no-JS baseline (up/down buttons in a form) and an API field for tile order (costly: new column, tag it).
- **P8 Multi-user sync across devices.** *Done:* shared running timers, "Cleo started a sleep" simulation, role chip. Keep showing *who* started a timer.

**Layout / containers**
- **P9 Card as the universal container; one card per activity.** *Done:* tiles, visit cards, running-timer cards. In shadcn-svelte terms: `Card` for every "thing with a state", `Badge` for its status.
- **P10 Layered cards: enter and review several data types without leaving the home tab.** *Candidate:* tile → bottom sheet (done) → recent entries for that tracker shown inside the same sheet ("last 3 bottles") so a parent can sanity-check before logging, without opening Timeline. Cheap: UI-only from existing list endpoint.
- **P11 One input sheet per activity type, tailored.** Sleep, breastfeeding, solids each get their own sheet. *Done:* bottle and diaper sheets. *Candidate:* nursing (timer + side), solids (food tags), sleep (start/end with "ongoing"), pumping, plus Growth and Vaccine record sheets (Vaccines page added).
- **P12 Timer built into the sheet (breastfeeding).** *Done:* Nursing running card with Switch side. *Candidate:* start the timer from the sheet and keep it running if the sheet is dismissed (it becomes the Running now card).
- **P13 Solids sheet lets parents tag foods.** *Candidate:* tag chips (`Badge`-styled toggle group) from the parent's recent foods + free text; handoff model has `what` ≤200 and `reaction` ≤200, so tags need either a client convention (comma-joined) or a costly schema change. Tag as API-costly if structured.
- **P14 Large inputs that expand on tap.** The only motion the page names. *Candidate:* steppers (done for volume) plus a full-width numeric field that grows to a big numeral on focus; respect `prefers-reduced-motion`, and the no-JS baseline is the plain field.
- **P15 Easy-tap buttons for one-handed, low-light use.** *Done:* 44–56 px targets, primary action at thumb height inside sheets. Keep primary buttons in the lower third.

**Data visualisation**
- **P16 History tab with a colour-coded weekly chart of all activities.** A week strip where each tracker has its own colour. *Candidate (Timeline page):* 7-day strip of per-day dots/bars coloured by tracker hue, each also labelled by icon and count, so colour is never the only signal. Hand-written inline SVG (no chart library); table fallback in a `Collapsible`.
- **P17 Trends: three views of the same data — graph, calendar, list.** *Candidate:* a segmented control (`Tabs` / `ToggleGroup`) Graph | Calendar | List on the Trends page. Graph = daily totals (sleep hours, feeds); Calendar = month grid with per-day tracker dots; List = the existing timeline. Each view must work without JS (server-rendered; the control is links with `aria-current`).
- **P18 Colour coding to tell activity types apart throughout.** *Done:* HUE map (bottle 65, nursing 20, diaper 95, sleep 275, pumping 330, solids 140, growth 200; vaccines use 235 on the Vaccines page). Rule: hue + icon + label, never hue alone. Medication and sickness will need their own hues, kept at least ~30° apart in oklch.
- **P19 Light and dark modes with the same colour coding.** *Done:* both themes use the same hue per tracker with different lightness. Keep verifying 4.5:1 for text on tinted chips.

**Visual identity**
- **P20 Playful, distinctive brand with a vibrant palette.** *Done in moderation:* warm orange primary on warm neutrals. *Conflicts if pushed:* saturated "status" colours on health values; keep the palette for identity and tracker coding only.
- **P21 Playful illustrations per activity type to reinforce recognition.** *Candidate:* small original illustrations as empty-state art and on tile icons (the project forbids photos, not illustration). Must be original, simple, SVG, `aria-hidden`; icon + label remains the accessible signal. Do not copy any app's mascot or illustration style.
- **P22 Simple typographic hierarchy for fast scanning.** *Done:* Geist, a small set of sizes (30 / 20 / 15 / 12.5 / 12 caps labels), mono numerals for timers. No new sizes without need.
- **P23 Intentionally simple system so features can be added easily.** Rule for future pages: reuse card, badge, sheet, tile, segmented control; add a new primitive only when two screens need it.

**Tone**
- **P24 Reassuring, supportive voice.** *Adapt:* reassure about the *act of logging* ("Logged", "Saved for everyone with access to Leo") and about the product ("Doses stay if you change schedule"). Never reassure or alarm about *values* (no "normal", "on track", "concerning").

## 4. Accessibility takeaways

Large tap targets and inputs; light/dark; type distinguished by colour **and** illustration. The page cites no standards or testing, so do not claim conformance from it. This project's bar (axe-clean at 390 px and ≥768 px, light and dark, `role="alert"/"status"`) is higher and stays.

## 5. Concrete backlog this reference suggests (not yet built)

1. Recent-entries strip inside each log sheet (P10) — UI-only.
2. Week strip on Timeline (P16) — a count strip is built in the Timeline page; upgrade to the time-of-day grid (V4).
3. Trends page with Graph | Calendar | List (P17) — new page.
4. Solids food tags (P13) — API-costly if structured.
5. Tile reorder (P7) — API-costly (order column) plus a no-JS up/down baseline.
6. Expanding numeric input (P14) with reduced-motion fallback.
7. Original per-tracker illustrations for empty states (P21).
8. "Logged" confirmation state on tiles (R2/P24) in addition to the undo toast.

## 5b. Visual findings from the 9 screenshots (viewed 2026-11-22)

These come from screenshots of the case study page supplied by the user, so they confirm what the product looked like in the marketing images, not how it behaves. Where the article text and the images differ, the images win. "Ours" notes the comparison with this project's prototype.

### V1. Home = a stack of full-width tracker cards (confirms P2, P3, P9, P18)
- Order seen: Sleep, Feed, Pump (Diaper in the carousel shots). Each card has a **coloured header band with the tracker's name** (Sleep pale blue, Feed yellow, Pump dusty rose, Diaper warm beige), a **round dark-green "+" button overlapping the band's right edge**, then a body with an **illustration in a tinted blob**, a **label + time-since line** ("Last feeding · 10m ago", "Woke up · 2h 6m ago", "Last change · 1h 55m ago") and a **large serif value** on the right ("12 oz", "10.0 oz", "wet"). A **"Show More"** footer row sits under the body.
- The header shows a kid avatar with a name dropdown, today's date, a **Summary** pill, and an alarm-bell icon with a count badge.
- Bottom tab bar has four tabs with icons and labels: Activity, History, Trends, Family.
- **Ours:** 2-column tiles with a hue-coded icon chip and a "since" subline. Nara spends a whole card per tracker and puts the **latest value** (not just the time) on the card. *Adopt:* show the last value on the tile (e.g. "120 ml", "Wet") next to the time since; keep tiles compact. *Don't copy:* the full-width stack costs scroll on a phone with six trackers.

### V2. Each card has three faces, shown as a swipeable carousel with 3 dots (new)
- Face 1 is the "last entry" card (V1). Face 2/3 are **summary cards**: "Total Diapers · 6 diapers" with a small **delta badge "↑2"**; "Feed Sessions · 16 sessions" with a **breakdown list using colour swatches** (Breastfeed 8, Bottlefeed 7, Solids 1, in three shades of the feed yellow); "Amount Pumped · 38 fl oz" with badge "↑3oz".
- **Pattern: Last → Total today → Breakdown**, same card chrome, a delta against a comparison period, and a dot pager.
- **Ours:** the Today grid is a separate section. *Adopt:* a per-card "Show more" that expands inline (a `Collapsible`) into today's total, the breakdown (bottle vs nursing vs solids) and the change since yesterday. *No-JS baseline:* `<details>`; the dot carousel is a JS-only enhancement and is not needed.
- *Rule check:* a delta such as "↑2" or "1h 19m less than the previous 7 days" is a plain comparison, not a judgement, so it is allowed if the wording is neutral ("2 more than yesterday"), shown with an arrow **and** text, never red/green, and never with words like "good" or "low".

### V3. Log sheets (confirms P11, P12, P14; adds detail)
- **Sheet chrome:** a coloured header band in the tracker's colour with **×** on the left, the tracker name centred in serif, **Save** on the right. The home dims behind it and stays visible (kid name and date still readable), so it is a partial-height sheet, not a new page.
- **Rows:** a simple list of label-left / value-right rows with hairline dividers: Start time "Today 12:34pm", Total time "0m", End time "Add", Notes, and a **toggle row** ("Dream feed"). An "Add photo" pill sits at the bottom (not applicable here: no photos).
- **Breastfeed sheet:** two **large light-weight timers side by side** (4M 02S and 3M 25S), with unit letters as small subscripts; a **"last side" tag** on the side used last time; under each, an action pill: outline **"Start Left"** and filled orange **"Stop Timer"**; a small pencil icon next to a timer edits it by hand.
- **Sleep sheet:** one big "Total Time 2H 12M 06S" counter with a **Stop Timer** pill, then Start time / End time "Add" / Notes; decorative stars around the counter.
- **Solids sheet:** a **meal-type icon row** (Breakfast, Lunch, Dinner, Snack; one is selected with a ring), a Time row, a **Food row with a chevron** that opens a picker, then **added foods as expandable cards** (Avocado: "Fruit • Disliked, tried grilling…" with a chevron; Blueberries: "Fruit", expanded). Expanded content is an **amount scale as four labelled icons "All / Some / A Little / None"**, then a row of **reaction faces**.
- **Ours:** bottle and diaper sheets exist with steppers and undo. *Adopt (new):*
  1. Nursing sheet with two timers (L/R), a "last side" tag from the previous session, one primary pill that flips Start/Stop, and a manual-edit affordance. Our API already stores `left_seconds`, `right_seconds`, `side`, `side_since`; "last side" is derivable from the previous session (UI-only).
  2. A **meal-type icon row** on the solids sheet. The model has no meal type; adding one is **API-costly** (new column). A cheap version is a time-of-day default shown as a label only.
  3. **Food as expandable cards with an amount scale and a reaction**. The model has `what` ≤200 and `reaction` ≤200 as free text; structured amount/reaction would be **API-costly**. Cheap version: chips for recent foods and a short fixed set of reaction chips written into the existing `reaction` text.
  4. A toggle row for a boolean such as "dream feed" is also **API-costly** (new field); only add if the owner wants it.
- *Contrast flags:* white text on the bright orange "Stop Timer" pill looks marginal for 4.5:1; our primary (oklch 0.53 0.17 38 on white) is darker on purpose. The rose and yellow header bands carry dark text, which is fine.

### V4. History = a week-by-time-of-day grid (strong match for P16, better than the article text implied)
- **Layout:** seven day columns (Fr 22 … Th 28, with the month label "Apr" and a back chevron for earlier weeks; today's column is highlighted) against **hours on the vertical axis (12 AM, 3, 6 AM, 9, 12 PM, 3, 6 PM, 9, 12 AM)**. Each entry is a **coloured block placed at its time**; **sleep is a tall blue block spanning its duration**, feeds yellow, pump rose, diaper small thin marks. A **"now" marker** (dot + line) shows the current time on today's column. A **Filters** button sits top right. Nights are shaded slightly differently from days.
- It makes sleep/wake rhythm and feed spacing visible at a glance without reading any numbers. This answers handoff question "how to read a day at a glance".
- **Ours:** the Timeline has a 7-day *count* strip (bars by tracker). *Adopt (high value):* replace or complement it with a **time-of-day week grid** drawn as inline SVG (no chart library), 7 columns × 24 h, blocks coloured by tracker hue **plus a distinct shape/pattern per tracker** (colour is never the only signal), the sleep spans as tall blocks, the now-marker, tap a block → the edit sheet. Needs a table fallback in a `Collapsible` (the list below it already is one), keyboard focus order by day then time, and a dark variant. Entries needed: the existing timeline range endpoint (≤93 days) is enough, so **cheap**.
- *Caution:* very thin blocks for brief events (a diaper) have tiny touch targets; make the grid a *read* view and route edits through the list below it.

### V5. Trends = Calendar | Graph | Entries, with an average line (confirms P17)
- **Segmented control with exactly three segments: Calendar, Graph, Entries.** Title row "Nara's Total Sleep" with a back chevron; headline "**8h 12m daily average**" in serif; a 7-day date strip (Fr 22 … Th 28) under the control; then a **bar chart, one pale-blue bar per day, y-axis 2h–16h in 2 h steps with dashed gridlines, and a dotted horizontal "AVG" line with a dark label tab at the right edge**; below it a **comparison line "1h 19m less time as compared to the previous 7 days"** with a small down-arrow badge.
- **Ours:** no Trends page yet. *Adopt:* Trends page for one tracker at a time (sleep total, feeds per day, diapers per day) with that control (`Tabs`), the headline average, the dotted average line, and the comparison line. The Growth page already has the Tabs-over-chart pattern, so the system is consistent.
- *Contrast flags:* pale-blue bars on a cream background look under 3:1 for non-text contrast; use a darker bar fill or an outline, and label bar values for screen readers (table fallback). "AVG" is a text label as well as a line, which is good.
- *Rule check:* "less time than the previous 7 days" is a neutral comparison; avoid colouring it or adding any "enough sleep" language.

### V6. Visual system (answers the gaps listed in section 6)
- **Palette:** a warm cream page background (about #F3F0E3) with white cards; **deep forest green** for primary actions, "+" buttons, links and illustration strokes; pastel tracker colours: **sleep periwinkle-blue, feed mustard-yellow, pump dusty rose, diaper warm beige/sand, solids in the feed yellow family**; a bright orange used only for the timer's stop action; the site chrome around the screenshots is navy and is not part of the app.
- **Typography:** **serif (a soft, high-contrast display serif) for card titles, values and big numerals**, a clean sans for labels and body text; light-weight large numerals for timers with tiny subscript unit letters (M, S, H). The pairing gives the app its personality while keeping data legible.
- **Illustration:** **single-weight dark-green outline drawings with a coloured organic blob offset behind** (bottle with a yellow fill, diaper with a sand blob, pump with a rose blob, moon with a star on a blue blob, a food item per food, baby faces for reactions). The blob colour = the tracker colour, so the colour system and the drawing system are one thing. A tiny version of the same drawing is used inside the summary cards.
- **Cards:** about 14-16 px radius, soft low shadow, header band in tracker colour, hairline dividers inside.
- **Controls:** pill buttons (outline for secondary, filled for primary), circular "+" for add, a round on/off switch, ring-selection for chosen icon options.
- **Ours vs this:** our warm-orange palette, Geist sans and hue-coded icon chips are a deliberate, different identity (and the handoff forbids photos, not illustration). What transfers is the *structure*: one colour per tracker used consistently on header, icon blob, chart marks and legend; a display face for numbers; and illustration that shares its colour with the tracker.

### V7. What the screenshots add to the backlog (section 5)
1. Time-of-day week grid on Timeline (V4): cheap, high value.
2. Per-tracker "Show more" with total, breakdown and a neutral comparison (V2): cheap (the daily-summary endpoint exists).
3. Trends page with Calendar | Graph | Entries and an average line (V5): cheap.
4. Nursing sheet with dual L/R timers and a "last side" tag (V3): cheap.
5. Latest value on each tile, not just the time since (V1): cheap, the home read model already has the last values or the list endpoint does.
6. Meal type, structured food amounts/reactions, boolean toggles (V3): API-costly; only if the owner asks.

### V8. Things in the screenshots NOT to copy
- Photos and the "Add photo" control (product rule: no photos).
- Any alarm-bell badge that implies reminders with content; handoff rule says no kid name or health value in a notification.
- Colour-only distinctions between trackers; add shape or pattern as well.
- The orange-on-white contrast and pale-blue-on-cream chart fills noted above.
- Marketing-device frames; they are presentation, not UI.

## 6. Still not known (neither the text nor the screenshots show it)

Onboarding, empty/error/loading states, motion beyond input expansion, gamification, exact typefaces and sizes (a serif plus a sans are visible but unnamed), exact hex values (colours above are read by eye from screenshots), dark-mode appearance of the app (the case study text claims it; no dark screenshot was supplied), the Family tab, the Summary screen, the Filters sheet, and the Calendar and Entries views of Trends. Outcome claims (high retention, ~20 minutes of daily use, 4.9 stars over ~25,000 App Store reviews) are the studio's reported marketing figures. Note the tension: 20 minutes of daily use is a vanity goal for a tool meant to be quick, so do not design for engagement time.
