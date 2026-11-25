# happybabytracker — design handoff: what plan 5.1 added (advanced bottle, nursing and pumping logging)

Audience: **Claude Design**, a design agent with **no access to this repository or to the session that built
it**. Treat this document as your only source. It is self-contained for plan 5.1.

**This file supersedes nothing.** It describes only the delta of plan 5.1 on top of the plans 3–5 handoff
(`plans-3-4-5-design-handoff.md`: trackers, timeline, growth, vaccination, navigation, tokens). Where this
file says "the timeline" or "the home" it means those screens as that handoff describes them, changed only as
stated here. If you do not have that handoff, section 0 repeats the ground rules you need.

| | |
|---|---|
| What | Three opt-in "advanced" forms: a **volume per source** for a bottle, a **weight before and after** for a nursing session (with an estimated milk volume), a **volume per side** for a pumping session |
| State (2026-11-23) | Merged to `main` (PR #8). Plans 3, 4 and 5 are merged too. API and UI shipped together; no new route, no new dependency |
| Scope of a redesign | The three forms, the toggle that opens them, the nursing estimate paragraph, the timeline line of a weighed nursing session and of a per-side pumping session. Nothing else changed |

One-line version: every bottle, nursing and pumping form still starts **simple** and unchanged. A plain text
link at the top of the form reloads it as the **advanced** form. Nothing is remembered between visits.

---

## 0. Ground rules (short form)

- **Product.** Private, self-hosted family health record for babies and kids; phones on the family's tailnet;
  PWA; online only; English only. It **records and displays; it never diagnoses, advises or judges.**
- **Stack.** SvelteKit 5 (runes), server-rendered, Node adapter. UI = **shadcn-svelte** (style `nova`, base
  colour `neutral`, lucide icons), bits-ui, Tailwind 4, Geist Variable. Installed shadcn parts: `alert`,
  `button`, `card`, `input`, `label`; everything else (native `<select>`, `<details>`, checkboxes, text links)
  is plain Tailwind. Adding registry components is welcome.
- **No-JS baseline.** Every form is a real `<form method="POST">` with `use:enhance`; **every action works with
  JavaScript disabled** (there is a no-JS test run for each advanced form). Show/hide that today works
  without JS is done with CSS `:has()`. For every interaction you propose, say what the no-JS fallback is.
- **Mobile first.** 390 px, no horizontal scroll even with a 200-character note; one nav; touch targets
  ≥ 44 px is the goal (see weaknesses: the entry-form inputs and selects are 32 px today).
- **Accessibility.** axe-clean at 390 and desktop widths in light and dark; colour never the only signal;
  errors `role="alert"`; labels and accessible names are matched by the e2e tests (appendix A). Change one
  only on purpose, and say so.
- **Never interpret.** No judgement words, no judgement colours, no "normal", "enough", "low", "good",
  "concern" for a weight, a gain or an estimate (section 7).
- **Tokens** are greyscale CSS variables in `oklch` (`--primary`, `--muted`, `--destructive`, …); propose
  new values for the existing names. Light/dark follows the OS.

---

## 1. What changed, at a glance

| Tracker | Simple form (unchanged) | Advanced form (new) | Link that opens it | Link back | Where the result shows |
|---|---|---|---|---|---|
| Bottle | contents checkboxes + **one** volume + unit | a volume **per ticked source** (breast milk, formula, cow's milk, other), one unit select, read-only **Total** line | "Advanced: volume per source" | "Use one total" | Edit page (inputs + Total). Timeline shows only the total and the content names, as before |
| Nursing | start side / finished minutes, note | optional **Weight before** and **Weight after** for the whole session, plus (edit page) the estimate paragraph | "Advanced: weigh before and after" | "Hide weigh-ins" | Nursing edit page (weights + estimate paragraph) and the **timeline line** (`…, weight gain 60 g, about 58 ml`) |
| Pumping | one volume + unit + side radio (Left, Right, Both) | a volume **per side** (left, right; at least one), one unit select, read-only **Total** line; **no side radio** (the side is derived) | "Advanced: volume per side" | "Use one total" | Edit page and the **timeline line** (`… 110 ml (L 60 ml · R 50 ml)`) |

Mechanics common to the three:

- The link reloads the same page with `?advanced=1`; the way back is `?advanced=0`. **Typed values are not
  carried across** the link. Every new form starts simple; nothing is remembered (owner decision, do not
  reopen).
- An **edit page opens advanced on its own** when the entry already carries advanced data (a bottle with
  sources, a pumping session with sides, a nursing session with either weight). `?advanced=0` overrides that.
- No new route. The home, Today list, daily summaries, Growth page and quick-log tiles are **unchanged** and
  show no advanced data (the bottle and pumping totals they already show are the sums).
- A **weigh-in is not a growth measurement**: no growth row, no score, no chart point, never on the Growth
  page. It lives on its nursing session only.

---

## 2. Data model and the rules the UI must honour

### 2.1 What is stored (SQLite, migration 0005)

```
bottle_feeds ──< bottle_feed_sources      PK (bottle_id, source)
                  source         breast_milk | formula | cow_milk | other
                  volume_ml, volume_floz_h (hundredths of a US fl oz), volume_typed ml | floz
pumping_sessions ──< pumping_session_sides   PK (session_id, side)
                  side           left | right
                  volume_ml, volume_floz_h, volume_typed ml | floz
nursing_sessions  + weight_before_g, weight_before_oz_h, weight_before_typed  kg | lb
                  + weight_after_g,  weight_after_oz_h,  weight_after_typed   kg | lb
                  (each triplet is all NULL or all set)
```

- A **simple** bottle or pumping session has **no child rows**; an advanced one has one per source or side.
  The bottle's and the session's own `volume` columns stay and **always equal the sum** of the parts, so the
  timeline, Today and daily summaries needed no change.
- Rows deleted with the kid cascade. Existing entries are untouched: a bottle with several contents and no
  sources stays a simple bottle; a nursing session without weights stays as it is. No backfill.
- **Stored three ways.** Every volume is kept as ml, hundredths of a US fl oz, and the unit the person typed;
  every weight as grams, hundredths of an ounce, and the unit typed (`kg` | `lb`). The typed value is exact;
  the other column is rounded once. **The UI never converts**: the API returns both columns and the UI picks
  one for the viewing parent's preference.

### 2.2 API views (what each screen can show)

Bottle (`volume` as today, plus `sources`; `[]` for a simple bottle, else one per content in the order breast
milk, formula, cow's milk, other):

```
{ id, at, contents: ["breast_milk","formula"],
  volume:  { ml: 150, floz_hundredths: 507, typed_unit: "ml" },
  sources: [ { source: "breast_milk", volume: { ml: 60, floz_hundredths: 203, typed_unit: "ml" } },
             { source: "formula",     volume: { ml: 90, floz_hundredths: 304, typed_unit: "ml" } } ],
  note }
```

Pumping (`side` and `volume` as today, plus `sides`; `[]` for a simple session; order left, right):

```
{ id, started_at, duration_seconds, volume, side: "both",
  sides: [ { side: "left", volume: {…} }, { side: "right", volume: {…} } ], note }
```

Nursing (new keys last; the same view is used by the edit page, the home's running card and the timeline):

```
{ id, started_at, ended_at|null, left_seconds, right_seconds, side|null, side_since|null, note,
  weight_before: { grams, oz_hundredths, typed_unit: "kg"|"lb" } | null,
  weight_after:  { … } | null,
  milk: null | { gain: { grams, oz_hundredths }, volume: { ml, floz_hundredths } } }
```

The timeline items carry the same entry views, so a timeline row has `sources`, `sides`, weights and `milk`
available without a new call.

### 2.3 Write bodies (so you know what a form can say)

- Bottle: `sources: { unit: "ml"|"floz", amounts: { breast_milk: 60, formula: 90 } }` (ml whole, fl oz in
  hundredths). `volume` and `sources` exclude each other (422 `sources`). `contents`, if sent, must equal the
  keys of `sources` (422 `contents`); the keys become the contents. On update, `sources` replaces the whole
  set; `volume` clears the sources; neither keeps both.
- Pumping: `sides: { unit, amounts: { left: 60, right: 50 } }`; `volume` and `sides` exclude each other (422
  `sides`); a `side` sent beside `sides` must equal the derived side (422 `side`). `volume` + `side` (the simple
  form) clears the sides.
- Nursing: `weight_before` / `weight_after`: `{ unit: "kg"|"lb", amount }` (grams, or hundredths of an ounce),
  `null` clears, absent keeps; allowed on a **running or finished** session and on create. A note-only
  update rewrites no weight.
- Field names the API refuses with (422): `sources`, `sides`, `volume`, `side`, `contents`, `weight_before`,
  `weight_after`, plus the existing ones. Check order, bottle: `at`, `contents`, `volume`/`sources`, `note`;
  pumping: `started_at`, `duration_seconds`, `volume`/`sides`, `side`, `note`; nursing: `started_at`,
  `ended_at`, `left_seconds`, `right_seconds`, `side`, `weight_before`, `weight_after`, `note`.

### 2.4 Rules the UI must honour

1. **The sum is the total.** In advanced mode the total is never typed; it is the sum of the typed parts in the
   one unit chosen. The saved total's typed column is exact (the sum of typed amounts); each part's other column
   may differ from the total's other column by rounding (at most one unit per part).
2. **One unit per bottle, one per pumping session.** One "Volume unit" select applies to every part. A bottle
   or session is saved with the unit of its form; it can change on any edit.
3. **A ticked source needs a volume; an unticked one is ignored.** A source typed and then unticked is not
   sent. At most four sources, two sides. Each part is at least 1 ml (or its fl oz equivalent that converts to
   at least 1 ml and 0.01 fl oz, in practice 0.02 fl oz). The **total** must be 1–1000 ml (bottle) or 1–2000 ml
   (pumping).
4. **The side is derived, never typed**, in advanced pumping: left only → `left`, right only → `right`, both →
   `both`. At least one side must carry a volume. A blank side is simply absent. A pumping session that was
   "both" with no split can only be a simple session.
5. **Switching to one total replaces the parts.** Saving the simple bottle or pumping form over an entry that
   has parts replaces them with the one total typed, **even if only the note changed**, because the simple form
   always sends its total in that case. The simple form says so ("Saving one total replaces the volumes per
   source" / "…per side"). Nursing differs: **"Hide weigh-ins" never deletes a weight**; a form without the
   weigh-in fields sends no weight key.
6. **Weights.** Optional, independent (only before, only after, both, neither); typed in the same control as
   the growth form (kg to three decimals, or whole pounds plus ounces 0–15.99), **200 g to 250 kg**
   (0.2–250 kg; about 7 oz to about 551 lb). Each of the two weights has its **own** unit select. They apply
   to the **whole session** (both sides together). Clearing the text clears the weight (sent as `null`).
7. **The estimate** (`milk`) is arithmetic on two typed numbers, by the API:
   - `gain_grams = after.grams − before.grams`; `ml = round-half-up(gain_grams × 100 / 103)` (density 1.03
     g/ml); `fl oz (hundredths) = round-half-up(gain_grams × 10⁸ / (103 × 295735))`, one rounding from grams.
   - `gain` in ounces is `after.oz_hundredths − before.oz_hundredths`, clamped at 0 (a lb+oz weight against a
     kg weight can differ by rounding alone).
   - **`milk` is `null` unless both weights exist and `after ≥ before`.** Equal weights give a **zero**
     estimate (`weight gain 0 g, about 0 ml`), not null. After **below** before: both weights are stored as
     typed, `milk` is `null`, no error.
   - Worked examples: 3400 g → 3462 g = gain 62 g, 60 ml, 2.04 fl oz; 3.4 kg → 3.46 kg = gain 60 g, 58 ml;
     7 lb 8 oz → 7 lb 10.5 oz = gain 2.5 oz, 2.33 fl oz (the volume comes from the rounded grams, 71 g).
   - The estimate is shown in the viewing parent's units: weight kg → grams, lb → ounces (no pounds);
     volume ml → ml, fl oz → fl oz. The four combinations are independent (kg with fl oz is possible).
8. **Wording that is owner-fixed** (do not reword; see section 4.3): the density sentence and the accuracy
   sentence, both next to every place the estimate is explained.
9. **Roles.** Guardians and caregivers both log and edit all three; there is no role difference in these forms.
   A request for a kid with no grant is the same 404 as an unknown kid; the admin never sees any of this.
10. **Time.** Unchanged: times are in the kid's zone, at most 5 minutes in the future, a nursing or pumping
    session at most 24 h.

---

## 3. Flows with business rules

1. **Log an advanced bottle.** Quick-log "Bottle" → full-page form (simple) → tap "Advanced: volume per
   source" (the page reloads; the time is reset to now) → pick the unit (defaults to the parent's) → tick the
   sources (a volume field appears under each ticked box) → type a volume per source → Save → redirected to the
   timeline day; the row reads `13:05 Bottle 150 ml, breast milk + formula`. No ticked box → the boxes carry
   "Choose at least one". A ticked source with a blank volume → the volume inputs carry "Enter a volume for
   each source you ticked…" and every typed value is kept. The Total line after a failed save reflects the
   typed values (`Total: 60 ml`).
2. **Edit one source.** Timeline row → edit page opens advanced on its own (inputs prefilled in the **viewing
   parent's unit**, "Total: 150 ml" from the stored total) → change one volume → Save. All parts are re-sent;
   a part whose typed amount and unit are unchanged keeps its stored columns. Ticking another source requires
   its volume; **unticking a source removes it** from the bottle and from the total on save.
3. **Switch back to one total.** Edit page → "Use one total" (`?advanced=0`, typed values dropped) → the simple
   form opens with the **stored total** in the Volume field, the contents boxes, and the hint "Saving one
   total replaces the volumes per source". Saving (even with only a note change) replaces the sources with that
   total; the sources are gone for good (hard replace, no history). Reverse: "Advanced: volume per source"
   on a simple bottle's edit page keeps its contents ticked with **blank** source volumes; saving needs a volume
   for each and **replaces** the old total by their sum.
4. **Weigh-in across a running session.**
   a. Quick-log "Nursing" → "Advanced: weigh before and after" → choose the starting side (starts a timer)
      and type **Weight before** → Save. The timer exists with a before weight; **the home's running card is
      unchanged** (no weight, no link to edit).
   b. The home's **End** is still one tap and takes no weight. A second parent can end it from their phone.
   c. To add the **after** weight: Timeline → the row → edit page. It opens advanced on its own (a before
      weight exists), the Weight after field is empty, no estimate paragraph yet. Type it, Save. The timeline
      line now ends with `, weight gain 60 g, about 58 ml`.
   d. For a session with **no** weight yet, the edit page opens simple: tap "Advanced: weigh before and after"
      first.
   e. A finished session can also be created in one go ("Or record a finished session": ended at, minutes per
      side) with both weights.
   f. If the other parent ends the timer while you are typing on the running edit page and you Save, the page
      answers "This timer has ended; check the times and save again" and **reloads the finished form with the
      stored values** (what you typed in the weight fields is not kept), still advanced.
5. **After lower than before.** Both weights are saved exactly as typed. The **timeline line shows no weight
   text at all**. The edit page shows the single word "No estimate" in the place where the estimate paragraph
   would be. No alert, no reason, no colour.
6. **Only one weight.** Stored; the edit page shows no estimate paragraph and no "No estimate"; the timeline
   line shows nothing extra.
7. **Clear a weight.** Empty its text → Save → the weight is removed (and with it the estimate).
8. **Hide the weigh-ins.** On an edit page of a weighed session, "Hide weigh-ins" reloads it simple; saving
   changes the note and leaves both weights stored. The weight fields and the estimate paragraph reappear
   when the link "Advanced: weigh before and after" is tapped.
9. **Log advanced pumping.** Quick-log "Pumping" → "Advanced: volume per side" → minutes (required), unit,
   Left volume, Right volume (at least one; the other may stay blank) → Save → `13:05 Pumping 15 min, 110 ml
   (L 60 ml · R 50 ml)`. Left only → `(L 60 ml)` and the session is stored as `left`. Both blank → the field
   message "Enter a volume for at least one side…". Edit later: add the missing side → it becomes `both`;
   blank a side and Save → only the other side remains and the side becomes that one.
10. **Imperial parent.** A parent whose volume preference is fl oz sees the unit select default to "fl oz",
    types `2,5` or `3.25` (point or comma, up to two decimals), and reads the same numbers back. A parent whose
    weight preference is lb sees pounds and ounces and the estimate in ounces and fl oz.
11. **Two parents with different units.** Volumes of an edit page are shown in the *viewing* parent's unit
    even if the other parent typed in another; weights are shown in the unit they were *typed* in (see
    weakness 8). The estimate and the timeline always use the viewer's units.
12. **Caregiver.** Same forms, same flows, no extra or missing control. A caregiver can log all three advanced
    forms and add a weigh-in to a running session.

---

## 4. Current UX (as built)

All three advanced forms live on the existing log (`/kids/{kid}/log/{bottles|nursing|pumping}`) and edit
(`…/{entry}`) pages: one narrow column (`max-w-md`), `h1` "Log a bottle" / "Log nursing" / "Log pumping" (edit:
"Edit bottle" / "Edit nursing" / "Edit pumping"), a plain link back to the kid's home above it, and at the
bottom of an edit page the "Delete this entry" `<details>`.

### 4.1 The form shell and the toggle link (all three)

Top to bottom, inside the form:

1. **The toggle**: one small underlined text link, first thing in the form (`text-sm`, not a button, not a
   switch). Labels:
   - bottle: "Advanced: volume per source" ⇄ "Use one total"
   - nursing: "Advanced: weigh before and after" ⇄ "Hide weigh-ins"
   - pumping: "Advanced: volume per side" ⇄ "Use one total"
2. A hidden `advanced=1` field exists only in the advanced form (that is how the POST knows the mode).
3. A `role="alert"` line for a failure that belongs to no field (for example a second running timer).
4. Muted line "Times are in {zone}".
5. The tracker's fields (below), then **Note**, then **Save** and a "Cancel" text link to the kid's home.

Because the link is a normal navigation, it **reloads the page and drops everything typed**: a new form goes
back to its starting values (the time to now), an edit page back to the stored values; on a failed POST it also
drops the posted values. With and without JS it behaves the same.

### 4.2 Bottle, advanced

Order: **Time** (datetime-local, required) → **Volume unit** (select: `ml`, `fl oz`; default the parent's
preference) → fieldset **Contents** with the muted hint "Tick each source and enter its volume" → four rows,
each a 44 px checkbox label ("Breast milk", "Formula", "Cow's milk", "Other") and, **only while its box is
ticked**, an indented text input labelled "{Source} volume" (decimal keypad; required in markup, form is
`novalidate`, so the server decides) → the line **"Total: N ml"** (or "Total: N fl oz") in medium weight →
Note → Save / Cancel.

- The reveal is CSS only (`:has(:checked)`), so ticking works without JavaScript.
- **Total line**: rendered by the server. New form: `Total: 0 ml`. Edit page: the stored total. After a failed
  save: the sum of the typed texts. **It does not change while you type** (no JS); it updates only after a
  submit.
- Errors: no box ticked → "Choose at least one" on the boxes (every box `aria-invalid`); a blank or malformed
  source → "Enter a volume for each source you ticked, in whole millilitres or ounces with up to two decimals"
  (one `role="alert"`, every ticked input `aria-invalid` and described by it); total out of range → "The total
  must be from 1 to 1000 ml" under the Total line (in ml even for an fl oz parent).
- Edit page: prefilled inputs in the viewing parent's unit; unticked boxes have no input; hidden "original"
  fields make an untouched part keep its stored columns. A note-only save sends neither sources nor volume.
- **Simple form over a bottle that has sources** (`?advanced=0`): the usual Contents grid (2 columns), the
  Volume input + unit select prefilled with the stored total, and under the Volume field the hint **"Saving one
  total replaces the volumes per source"**. Nothing else warns.

### 4.3 Nursing, advanced

Simple form (unchanged): "Started at"; on a new session a "Side" radio group (Left, Right; hint "Choose the side
you start on to run a timer") and a bordered group "Or record a finished session" ("Ended at", "Left minutes",
"Right minutes"); on an edit page "Ended at" (running: hint "Set the end to end the timer"), minutes for a
finished one; Note.

Advanced adds, **after those fields and before the Note**, one block:

1. **Weight before** — a fieldset (accessible role group, legend "Weight before"). Inside: the input(s) of the
   selected unit and a unit select to their right (`kg` | `lb`, 96 px wide, accessible name "Weight before
   unit", default the parent's preference). Kg shows one field labelled "Kilograms"; lb shows two, "Pounds" and
   "Ounces" (CSS `:has()` on the select's checked option, so it flips without JS). Comma or point accepted.
2. **Weight after** — the same control, legend "Weight after", select "Weight after unit".
3. One muted paragraph under both: **"In your weight unit. These weights stay with this session; they are not
   added to growth. A single-feed reading from a home scale can be off by several grams."**
4. **Edit page only**, one plain paragraph (`text-sm`, not muted, no role) under the hint. It is computed when
   the page loads from the stored session:
   - `milk` present: **"Weight gain 60 g, about 58 ml. The volume assumes about 1.03 g per ml, an approximation
     in the range commonly used (1.0–1.03 g/ml); not measured. A single-feed reading from a home scale can be
     off by several grams."** (an lb / fl oz parent reads "Weight gain 2.5 oz, about 2.33 fl oz. The volume
     assumes …"; a zero gain reads "Weight gain 0 g, about 0 ml. …")
   - both weights present and `milk` null (after lower than before): the single paragraph **"No estimate"**;
   - a weight missing: nothing;
   - the new-session form never shows it.
- Errors: a malformed weight → "Enter a weight: kilograms with up to three decimals, or pounds and 0 to 15.99
  ounces"; a kilogram value outside the range → "Enter a weight from 0.2 to 250 kg" (a pound value outside the
  range is refused by the API and shows the first message, which does not state the range). The message sits
  under the one weight fieldset (`role="alert"`), its inputs and select `aria-invalid`; every typed value is
  kept.
- The home's running card ("Nursing, left since 13:05", **Switch side**, **End**) shows no weight and no link
  to edit (the only link on that card is "set the end time" after 24 h).

### 4.4 Pumping, advanced

Order: **Started at** → **Minutes** (required, whole minutes for the whole session) → **Volume unit** select →
fieldset legend **"Volume per side"** with the muted hint "Enter at least one side; leave the other blank" →
inputs "Left volume" and "Right volume" (stacked, not side by side) → **"Total: N ml"** → Note → Save / Cancel.
There is **no Side radio**. Errors: both blank or malformed → "Enter a volume for at least one side, in whole
millilitres or ounces with up to two decimals" (both inputs `aria-invalid`); total out of range → "The total
must be from 1 to 2000 ml". The Total line behaves exactly as the bottle's.

Simple pumping (unchanged): Started at, Minutes, **Volume** + unit select, a required "Side" radiogroup (Left,
Right, Both). Over a session that has sides (`?advanced=0`) the hint under Volume is **"Saving one total
replaces the volumes per side"**, the volume holds the stored total and the radio the derived side.

### 4.5 Timeline (the only place outside the edit pages that shows any of this)

A row is one link (whole row, min 44 px) with one line of text, then the note in muted text. Day headings are
"Mon 23 Nov 2026" style (the plans 3–5 handoff wrote "Mon, Nov 23, 2026"; the code prints no commas). Lines:

| Entry | Line |
|---|---|
| Simple bottle | `13:05 Bottle 120 ml, breast milk + formula` |
| **Advanced bottle** | **identical form**, the total only: `13:05 Bottle 150 ml, breast milk + formula` (the per-source volumes are **not** on the timeline) |
| Nursing, no weights, or weights without an estimate | `13:05 Nursing L 10 min · R 5 min` (running adds `, running on L`) |
| **Nursing with an estimate** | `13:05 Nursing L 10 min · R 5 min, weight gain 60 g, about 58 ml` (lb / fl oz parent: `…, weight gain 2.5 oz, about 2.33 fl oz`; zero gain: `…, weight gain 0 g, about 0 ml`). The raw weights are not shown on the timeline |
| Simple pumping | `13:05 Pumping 15 min, 110 ml (both)` |
| **Advanced pumping** | `13:05 Pumping 15 min, 110 ml (L 60 ml · R 50 ml)`; left only: `… 60 ml (L 60 ml)`; fl oz parent: `(L 2.03 fl oz · R 1.69 fl oz)` |

### 4.6 Where nothing shows

Home (Today list, running card, Due), daily summaries, the Growth page and chart, and notifications show no
part, weight or estimate. There is no per-source or per-side daily total. A weighed session adds nothing to
Growth.

### 4.7 No-JS baseline (what works without JavaScript, and how)

| Interaction | Without JS |
|---|---|
| Open advanced / go back | plain links with `?advanced=1` / `?advanced=0` (full reload) |
| Reveal a source's volume input | CSS `:has()` on the checked box, live, no JS |
| kg ⇄ lb+oz inputs | CSS `:has()` on the select's checked option, live, no JS |
| Total line | server-rendered; updates only after a submit |
| Estimate paragraph | server-rendered on load of the edit page; stale after a failed save (weakness 6) |
| Failed save | the form reloads with every typed value, field error and the form still advanced |
| Save | normal POST, 303 redirect to the entry's timeline day |

With JS the same markup is progressively enhanced (`use:enhance`: the form is not reloaded on a failed save).
No component here requires JS.

---

## 5. Weaknesses (honest audit; evaluate and add your own)

1. **The toggle is a link that loses typed values and resets the time.** Switch before typing or start over.
   A second tap per session for anyone who weighs every feed, because nothing is remembered. It is a 14 px
   text link, not a 44 px target, and it looks identical for two different consequences: "Hide weigh-ins" is
   harmless, "Use one total" **destroys the per-source split on the next Save**, even when only the note was
   edited. The only warning is a muted hint under the Volume field.
2. **The Total line is server-rendered**: it says `Total: 0 ml` while you type three volumes and updates only
   after a submit. It can be misread as the current sum. It has no live region.
3. **The after weight needs End, then Edit.** From the home, after End there is no link to the just-ended
   session; the path is Timeline → day → row → (if it had no weight) the advanced link → type → Save. A
   caregiver at 3 a.m. will not find it. The running card does not hint that a before weight exists.
4. **Weight fields come after "Or record a finished session"** on the new form, so a parent who wants "start a
   timer with a before weight" scrolls past an unrelated bordered group to reach it.
5. **Four per-source inputs on a phone**: the volume input appears indented under its box, so the form grows
   by up to four rows and pushes the Total and the Note below the fold. No inline layout (checkbox and input in
   one row), no unit label next to each input (the unit is the select at the top).
6. **The estimate paragraph lags**: it is computed from the stored session when the page loads. After a failed
   save the weight inputs show what you typed while the paragraph still describes the old stored weights. After
   a successful save you land on the timeline, so you see the estimate only there (one line) or by reopening the
   edit page.
7. **Wall of small print.** On an estimated session the edit page carries the accuracy sentence **twice** (in
   the hint under the weights and again at the end of the estimate paragraph) plus the density sentence: three
   sentences of muted text around a two-number result. The wording is owner-fixed; its layout is not.
8. **Unit inconsistency**: volumes of parts are shown on an edit page in the *viewing parent's* unit (even when
   the other parent typed in the other unit), but weights are shown in the unit they were *typed* in. An
   lb parent opening a kg session sees kilogram inputs and an estimate in ounces. The README says amounts "read
   back as typed in the unit last saved in"; the code does this for weights only.
9. **Imperial parents re-type other parts when editing one.** All parts are re-sent in the viewer's unit on
   any edit of one part. A part typed in ml and shown to an fl oz parent is re-saved as typed fl oz (the typed
   unit flips; the ml column is recomputed and, for these ranges, comes back to the same ml). Exactness of the
   untouched parts is not guaranteed across units.
10. **"No estimate" wording and the owner's rule.** The plan says an after weight below before shows "no
    estimate, no message"; the code shows the two-word paragraph "No estimate" and nothing else. It is neutral
    but it is a message; whether the owner wants it is open, and any reason ("after is lower") is forbidden.
11. **The range message for pounds**: out-of-range pounds show the format message (which does not mention
    200 g–250 kg); kilograms show "Enter a weight from 0.2 to 250 kg". Total-range messages are always in ml,
    even for an fl oz parent.
12. **Stale-timer reload discards a typed weight.** If the other parent ended the timer, your Save answers
    "This timer has ended; check the times and save again" and the form shows the stored values, not yours.
13. **Pumping "both, no split" is only possible in the simple form.** Advanced cannot say "both, unknown
    split"; a left-only entry becomes "both" the moment a right volume is added.
14. **Inconsistent timeline**: a pumping session shows its sides, a bottle does not show its sources, a weighed
    session shows its gain but never its weights.
15. **Inputs and selects in entry forms are 32 px high** (`h-8`), including the weight unit select and the four
    advanced inputs; the shared control, not specific to plan 5.1, but dense here.
16. **Two unit selects for one scale reading** (before and after), each defaulting to the parent's unit, can be
    set to different units; nothing explains that this is allowed.
17. **No confirmation after a save** (redirect only), as everywhere in the app; no undo for the destructive
    "Use one total".

---

## 6. Open design questions (answer, or label an assumption; tag each API change cheap / costly)

Where a question touches an owner decision, the decision is named; you may propose, the owner decides.

1. **Toggle that keeps typed values.** Replace the reload link by a control that swaps blocks in place without a
   round trip: for example a checkbox/disclosure ("Advanced") whose `:has(:checked)` shows the per-source or
   weigh-in fields and hides the simple volume. Works without JS and keeps typed values and the time.
   The form already reads `advanced=1`; two blocks would share the unit select name, so one must be shared.
   **UI-only: cheap.** (Owner: "nothing remembered, every form starts simple" stays true.)
2. **A one-tap way in for people who always weigh.** A second quick-log affordance ("Nursing, weighed") linking
   to `?advanced=1` without remembering anything. **UI-only: cheap.** A remembered preference is an owner
   reversal and would need a stored setting: **costly**.
3. **Adding the after weight.** Options: (a) after End on the home, a `role="status"` line "Nursing ended.
   Add a weight after" linking to that session's edit page (the End form already carries the session id):
   **cheap**; (b) a weight field on the home's End form: two API calls from the server action (end, then
   update), not atomic: **cheap in API, costly in busy-screen terms and the owner decided the opposite (row 13,
   "design call")**; (c) a single API call ending with a weight: **costly**.
4. **Live Total.** A small JS enhancement that sums the inputs as you type, with the server-rendered Total as
   the no-JS baseline; or remove the line from the form and show it only on edit/after save. **UI-only: cheap.**
5. **Layout of the four source rows on a phone**: one row per source (checkbox + input inline), a 2-column grid
   from 768 px, units shown beside the input, the Total sticky above Save. **UI-only: cheap.**
6. **Make the two toggle consequences different**: harmless ("Hide weigh-ins") versus destructive ("Use one
   total"): a distinct style, a stated consequence next to the link ("replaces the per-source volumes when you
   save"), or a `<details>` second step before the simple form saves. **UI-only: cheap** (a server-rendered
   confirmation would be a new step: also cheap).
7. **Present the estimate** without interpreting it: typographic treatment of "Weight gain 60 g, about 58 ml"
   (a result line, not a chart, not a badge), where the density and accuracy sentences sit (fixed wording, may
   be one `<details>` only if still adjacent to the estimate; ask the owner whether collapsing counts as
   "beside"), and removing the accuracy sentence's duplicate (hint + paragraph). **UI-only: cheap.**
8. **"No estimate"**: keep, or rename (for example "No estimate for these weights"), or show nothing like the
   one-weight case. No reason may be given. Owner call. **UI-only: cheap.**
9. **Show the raw weights on the timeline** (for example `3.40 → 3.46 kg`) or the sources of a bottle
   (`Bottle 150 ml (breast milk 60 · formula 90)`, like pumping). The API already returns both in the timeline
   items. **UI-only: cheap.** Owner: row 14 decided where the *estimate* shows; it did not decide raw weights or
   sources.
10. **Unit policy on an edit page**: show weights and parts in the viewer's unit (using the API's other column;
    `kg` weights up to 3 decimals, `lb` as pounds + ounces) or in the typed unit, and say which. **UI-only: cheap.**
11. **A single scale control** instead of two unit selects (before and after share one unit), so a weigh-in pair
    cannot mix units. **UI-only: cheap.** (Mixed units are valid for the API.)
12. **Move the weigh-in block above "Or record a finished session"** or fold the finished-session fields
    behind a disclosure on the new form. **UI-only: cheap.**
13. **Per-side advanced pumping**: how to say "both, split unknown" without the simple form (a third "Total
    only" field? a toggle?). **Costly** if it needs a new value; **cheap** if it is only copy ("use one total").
14. **Estimate on the home or in Today** (for example the last feed's estimate): owner row 14 excluded it.
    Only with the owner. **Costly** if aggregate (a summary field), **cheap** for the last session only.
15. **Visual identity of a weigh-in**: an icon (lucide `scale`) on the group and the timeline line; tracker
    accents. Neutral, no up/down glyphs (section 7). **UI-only: cheap.**
16. **Range copy**: state "200 g to 250 kg" (and the pound equivalent) in the hint, and make the pound message
    match the kilogram one. **UI-only: cheap.**

---

## 7. What you must NOT do

- **No judgement, in colour or words, for a weight, a gain, a loss or an estimate.** No green/red, no
  up/down/trend arrows or sparklines, no "good", "low", "enough", "concern", "normal", "healthy", "behind",
  "on track". The e2e tests assert that the nursing edit page's `<main>` text matches none of
  `/good|low|enough|concern|normal/i` (a case-insensitive substring match, so also "below", "allow", "follow",
  "slow"), on the pages with an estimate and with "No estimate". Check your copy against it.
- **No interpretation and no explanation of a lower after weight.** The app shows "No estimate" at most, never
  why, and never mentions latching, feeding problems or anything medical.
- **Do not change the two owner-fixed sentences**: the density wording ("The volume assumes about 1.03 g per
  ml, an approximation in the range commonly used (1.0–1.03 g/ml); not measured.") and the accuracy sentence
  ("A single-feed reading from a home scale can be off by several grams."). Layout is yours, wording is not.
- **A weigh-in is not growth.** Never place it on the Growth page, a chart, a percentile, a z-score or in the
  growth list; never label it "weight" in a way that reads as a measurement of the child (it is the weight
  before and after a feed); never offer to "save to growth".
- **No new JS-only path.** Everything must work with JavaScript off; JS may only enhance.
- **No chart library**, no animation that carries meaning, no colour as the only signal.
- Do not add a stored "advanced" preference (owner: per-form, nothing remembered) without labelling it as an
  owner reversal; do not put the estimate on the home or in daily summaries without labelling the same.
- Do not put any weight, volume or estimate in a notification text, and do not show any of it to the admin.
- Do not rename the accessible names in appendix A without listing the rename.

---

## 8. Appendix A — stable accessible names the e2e tests rely on

Links (exact text): **"Advanced: volume per source"**, **"Advanced: weigh before and after"**, **"Advanced:
volume per side"**, **"Use one total"**, **"Hide weigh-ins"**. Page headings (`h1`): "Log a bottle", "Log
nursing", "Log pumping", "Edit bottle", "Edit nursing", "Edit pumping". Buttons: "Save", "End nursing"
(the visible text is "End"), "Switch side", "Yes, delete this entry"; `<summary>` "Delete this entry"; region
"Running now"; link "Cancel".

Bottle: select **"Volume unit"** (options `ml`, `fl oz`); checkboxes **"Breast milk"**, **"Formula"**, **"Cow's
milk"**, **"Other"** (matched exactly, because the inputs below share the prefix); inputs **"Breast milk
volume"**, **"Formula volume"**, **"Cow's milk volume"**, **"Other volume"** (hidden until the box is ticked,
and they must stay hidden when unticked); text **"Total: 0 ml"**, **"Total: 150 ml"**, **"Total: 5.75 fl oz"**
(`Total:` appears only in the advanced form); simple form: input **"Volume"**, hint **"Saving one total
replaces the volumes per source"**; a hidden `advanced=1` input only in the advanced form.

Pumping: **"Started at"**, **"Minutes"**, **"Volume unit"**, **"Left volume"**, **"Right volume"**, **"Total:
…"**, the "Side" radiogroup with 3 radios in the simple form and **none** in the advanced one, hint **"Saving
one total replaces the volumes per side"**, error text starting **"Enter a volume for at least one side"**.

Nursing: groups (fieldset legends) **"Weight before"**, **"Weight after"**; inside each, labels
**"Kilograms"**, **"Pounds"**, **"Ounces"** (found within the group, so they repeat across the two groups);
selects **"Weight before unit"**, **"Weight after unit"** (values `kg`, `lb`); fields "Started at", "Ended at",
"Left minutes", "Right minutes", radios "Left"/"Right", "Note". Hint text exactly **"In your weight unit. These
weights stay with this session; they are not added to growth. A single-feed reading from a home scale can be
off by several grams."**; estimate paragraph exactly as in section 4.3 (matched with `exact: true`); the text
**"No estimate"** exactly; error text starting **"Enter a weight"**.

Errors: field errors are `role="alert"` and the field carries `aria-invalid="true"` and `aria-describedby`.
Messages: "Choose at least one"; "Enter a volume for each source you ticked…"; "Enter a volume for at least one
side…"; "The total must be from 1 to 1000 ml" / "…to 2000 ml"; "Enter a weight…".

Timeline row links (name = line text + note): `Bottle 150 ml, breast milk + formula`; `Nursing L 10 min · R 5
min, weight gain 60 g, about 58 ml`; `Pumping 15 min, 110 ml (L 60 ml · R 50 ml)`; a nursing line with a lower
after weight contains no word "weight".

## 9. Appendix B — limits

| What | Limit |
|---|---|
| Bottle total (sum of sources) | 1–1000 ml |
| Pumping total (sum of sides) | 1–2000 ml |
| Each source or side | at least 1 ml (fl oz: at least 0.02 fl oz); at most 4 sources, 2 sides |
| Volume input | ml: whole number; fl oz: up to 2 decimals (point or comma); nothing else (`1e3`, `0x10`, `--1` refused) |
| Weight (each of the two) | 200 g to 250 kg (0.2–250 kg); kg up to 3 decimals; lb whole pounds + ounces 0–15.99 |
| Density | 1.03 g per ml (owner's figure, the app's own constant) |
| Estimate | rounded once, from grams; shown only when both weights exist and after ≥ before |
| Note | 500 characters |
| Session length / entry time | ≤ 24 h; ≤ 5 min in the future; not before birth |

## 10. Appendix C — reference

- **Fixtures** (stub API and tests): kids Mia, Leo (and Ivy); parents Ana, Cleo, Dan. Ana guards Mia and Ivy
  and is a **caregiver** for Leo (Cleo guards Leo); Dan has no kids; admin `admin`. A test parent's units are
  metric (kg, ml) unless a test flips them to lb and fl oz.
- **Where things are** (for implementers; you need not read any of it): the forms are
  `ui/src/lib/components/entry/{BottleFields,SourceVolumes,NursingFields,WeightField,PumpingFields,SideVolumes,TrackerForm}.svelte`;
  the toggle labels and the estimate paragraph are in `ui/src/lib/entry-edit.ts`; the timeline lines in
  `ui/src/lib/timeline.ts`; the two owner-fixed sentences in `ui/src/lib/constants.ts`; the API in
  `api/internal/httpapi/{bottles,nursing,pumping}.go` and `api/internal/units/milk.go`; the plan is
  `docs/superpowers/plans/2026-11-23-advanced-logging.md`.

## 11. What to hand back

A design spec, not code. Per screen at **390 px and ≥ 1024 px, light and dark**:

1. The **three advanced forms** (bottle, nursing, pumping), each on a new entry and on an edit page, and the
   **simple form over an entry that has parts** (the destructive "Use one total" case).
2. The **toggle**: what it looks like, how the two consequences differ, and its no-JS form.
3. The **nursing estimate states**: estimate shown (kg/ml and lb/fl oz), zero gain, "No estimate" (or your
   replacement), one weight only, no weights, running session with a before weight.
4. The **timeline rows** for an advanced pumping session, a weighed nursing session (with and without an
   estimate) and an advanced bottle (and your answer to question 9).
5. **States** for each: empty, field error, server error, a failed save that keeps typed values, caregiver
   view, imperial parent, and the no-JS baseline.
6. Annotated layout, the **shadcn-svelte component mapping** (and any new components to add), **all copy**
   (changing nothing in the owner-fixed sentences), lucide icon picks, token changes as new `oklch` values for
   existing variable names, and the list of accessible names you change (appendix A).
7. An **API-change list** tagged **cheap / costly**, including your answers to section 6 (label each
   assumption), and a short **rationale** tying each change to a persona (parent at 3 a.m., second parent,
   caregiver, parent reviewing in the evening) and to a rule in sections 2.4 and 7.
