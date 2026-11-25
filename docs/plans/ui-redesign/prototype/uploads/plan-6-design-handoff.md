# happybabytracker — design handoff: what plan 6 added (sickness, medication, reminders, Web Push, ntfy)

Audience: **Claude Design**, a design agent with **no access to this repository or to the session that built
it**. Treat this document as your only source. It is self-contained for plan 6.

**This file supersedes nothing.** It describes only the delta of plan 6 on top of the plans 3–5 handoff
(`plans-3-4-5-design-handoff.md`: trackers, timeline, growth, vaccination, navigation, tokens) and the plan 5.1
handoff (`plan-5-1-design-handoff.md`: advanced bottle, nursing and pumping forms). Where this file says "the
timeline", "the home" or "the entry form" it means those screens as those handoffs describe them, changed only
as stated here. Section 0 repeats the ground rules you need.

| | |
|---|---|
| What | A **sickness tracker** (one symptom per entry; a fever carries a temperature and how it was taken), a **Medication page** (courses with a schedule, one-tap "Log dose now", dose edit), **reminders** for scheduled doses on the home **Due** list, and two **optional** delivery channels: **Web Push** (per device) and **ntfy** (per parent), managed on a new **Notifications** page under Account |
| State (2026-11-23) | Built on branch `ccr-174d8c4e-jjczdt` at commit `921c1a1` (draft PR #9 to `main`, merged only with the owner's go-ahead). API and UI ship together. The deployment ships with **both channels off** (no Web Push key, no ntfy) until separate infrastructure changes turn them on; every screen must therefore be designed for "no channel configured" as a normal, first-class state |
| Scope of a redesign | The symptom form and its timeline line, the four Medication pages, the Notifications page, the one new kid-settings field, the medication lines of the home Due list, the two new quick-log tiles, one new link on Account and one under the kid's name. The OS notification text is fixed copy, not UI |

One-line version: the app **records and displays** symptoms, temperatures, medication courses and doses; it
lists what is due and what was missed; it can send a **generic** "Medication due for <child's name>" to the
devices a parent chose. It **never interprets, advises or computes a dose or an interval**.

---

## 0. Ground rules (short form) and status legend

- **Product.** Private, self-hosted family health record for babies and kids; phones on the family's private
  network; installable PWA; online only (there is **no offline page**, by owner decision); English only.
  It **records and displays; it never diagnoses, advises or judges.**
- **Stack.** SvelteKit 5 (runes), server-rendered, Node adapter; the browser talks only to the SvelteKit
  server. UI = **shadcn-svelte** (style `nova`, base colour `neutral`, lucide icons), bits-ui, Tailwind 4,
  Geist Variable. Installed shadcn parts: `alert`, `button`, `card`, `input`, `label`; everything else
  (native `<select>`, `<details>`, checkboxes, radios, text links) is plain Tailwind. Adding registry
  components is welcome.
- **No-JS baseline.** Every form is a real `<form method="POST">` with `use:enhance`; **every action works with
  JavaScript disabled**, with **one exception in the whole app: the "Turn on notifications on this device"
  button**, which cannot exist without JavaScript (the browser's push API is JavaScript only) and says so.
  Show/hide that works without JS is done with CSS `:has()`. For every interaction you propose, say what the
  no-JS fallback is.
- **Mobile first.** 390 px, no horizontal scroll even with a 200-character unbroken name or note; one nav
  (bottom bar < 768 px, left column ≥ 768 px); touch targets ≥ 44 px is the goal (many plan 6 controls miss
  it today: section 6).
- **Accessibility.** axe-clean at 390×844 and 1280×800 in light and dark; colour never the only signal; errors
  `role="alert"` with the field `aria-invalid` and `aria-describedby`; status lines `role="status"`; accessible
  names are matched by the e2e tests (appendix B). Change one only on purpose, and say so.
- **Never interpret** (section 8). No threshold, no colour and no word that judges a temperature, a symptom, a
  dose, a schedule or a missed dose.
- **Tokens** are greyscale CSS variables in `oklch`; propose new values for the existing names. Light/dark
  follows the OS.

**Status legend used below**

| Tag | Meaning |
|---|---|
| **Built** | What the code at `921c1a1` does; copy quoted from the code |
| **Owner** | An owner decision; do not reopen (you may propose, labelled as an owner reversal) |
| **(a)** | A rule the plan author set; changing it has a stated cost |
| **Open** | Not decided; design with the stated assumption and label it |
| **Unverified** | Could not be checked from the code (real devices, OS rendering) |

---

## 1. Plan 6 at a glance

### 1.1 What it adds, in product terms

| Area | What a parent can do | Where |
|---|---|---|
| Sickness | Log one symptom per entry with an optional severity; "Other" with its own text; a **fever** with its **temperature** (°C or °F, up to two decimals) and **how it was taken** | Quick-log tile "Sickness" → `/kids/{kid}/log/symptoms`; edit from the timeline |
| Timeline | Symptoms appear as timeline rows and as a filter checkbox "Symptom" | `/kids/{kid}/timeline` |
| Medication courses | Keep a course: name, dose text, schedule (as needed / fixed times of day / every N hours), start, optional last day, active on/off, note | `/kids/{kid}/medication`, `/new`, `/{course}` |
| Doses | Log a dose now in one tap (optional amount text); correct its time, amount, note; delete | Medication page; `/kids/{kid}/medication/doses/{dose}` |
| Due list | See each scheduled dose that is due today or recently missed, by course name and time, before vaccine visits | Kid home, "Due" |
| Overdue window | A guardian sets how long a missed dose stays listed (5–1440 minutes, default 60) | Kid settings |
| Notifications | Turn Web Push on per device and choose which children; turn ntfy on and choose children; rename or remove devices | Account → Notifications (`/account/notifications`) |
| The notification itself | The phone shows "Medication due for <child's name>" (never the medication or the dose); tapping opens that child's Medication page | OS notification |

**No-channel state is normal.** With neither channel configured (how the app ships), the Medication page and
the Due list work fully; the Notifications page says "Web Push is not configured on this server." and has no
ntfy section.

### 1.2 The guiding principle

The app **records what a parent typed and displays it back, in the parent's units**. It:

- shows a temperature as a number with its unit and method, **never** with a label, colour, icon, threshold,
  range bar or comparison;
- shows a course's schedule exactly as typed ("08:00, 20:00", "Every 6 hours from 08:00", "As needed"), and
  "Last dose logged at …" / "Next due …" from the server; it **never** computes, suggests or warns about an
  interval, a dose or a total;
- lists a missed dose with the neutral word "overdue" for a bounded time, then drops it; no escalation, no
  alarm styling;
- sends a notification whose only text is "Medication due for <name>".

### 1.3 Roles (who can do what)

| Action | Guardian | Caregiver | Admin |
|---|---|---|---|
| Log, edit, delete symptoms | yes | yes | never sees any child |
| Create, edit, delete courses; log, edit, delete doses | yes | yes | — |
| See the Medication page and the Due list | yes | yes | — |
| Change "Show a missed dose as overdue for (minutes)" | **yes** (kid settings, guardian-only page) | no (no settings page; same 404 as an unknown child) | — |
| Turn Web Push on for a device, choose children, rename, remove | yes, for children they hold a grant on | yes, same | — |
| Turn ntfy on/off, new topic, choose children | yes | yes | — |
| Switch the `sickness` / `medication` modules | yes (Modules page) | no | — |

There is **no role difference** on any symptom or medication screen: a caregiver sees the same controls. A
request for a child with no grant, an unknown child, or another child's entry is the **same "Not found"
page** (section 2.10). The admin principal can never reach any of these pages.

---

## 2. Data model and business rules

### 2.1 What is stored (Built)

```
kids  + reminder_overdue_minutes   integer 5–1440, default 60 (existing children read 60)
kids ──< symptoms           at, symptom, severity?, other_text?, temperature? (c/f hundredths + unit typed), method?, note?
kids ──< medications        name, dose, schedule {none | times[HH:MM…] | every N hours}, start_at, end_on?, active, note?
          └──< medication_doses   given_at, amount?, note?
          └──< reminders          one row per occurrence: due_at, state pending|done|expired|cancelled
                 └──< reminder_deliveries   one per target and channel (server-side bookkeeping, never shown)
parents ──< push_subscriptions (devices: label, created_at) ──< push_kid_optins (device × child)
parents ──  ntfy_topics (one topic per parent) ──< ntfy_kid_optins (parent × child)
```

Hard deletes everywhere, no edit history (as in earlier plans). Ids are opaque.

### 2.2 What each screen can show (API views, abbreviated)

- **Symptom:** `{id, at, symptom, severity|null, other_text|null, temperature:{c_hundredths, f_hundredths,
  typed_unit}|null, method|null, note|null}`.
- **Course:** `{id, name, dose, schedule:{kind, times?, hours?}, start_at, end_on|null, active, note,
  last_dose:{id, given_at, amount}|null, next_due:{due_at}|null, dose_count}`.
- **Dose:** `{id, medication, given_at, amount|null, note|null}`.
- **Home due entry (medication):** `{kind:"medication", medication:{id, name}, due_at, state:"due"|"overdue"}`.
  (Vaccine entries keep `{kind:"vaccine", from, status, count}`. The field is `state` for medication and
  `status` for vaccines: an API detail with no UI consequence.)
- **Push state:** `{enabled:false}` or `{enabled:true, public_key}`. **Devices:** `[{id, label, created_at,
  kids:[ids]}]` (kids filtered to current grants). **ntfy:** `{enabled:false}` or `{enabled:true, server,
  topic|null, kids:[ids]}`.
- The UI **never receives** a push endpoint or key; it never shows one.

### 2.3 Symptoms and temperature

| # | Rule | Source |
|---|---|---|
| S1 | Symptom list (fixed order): Congestion, Cough, Fever, Vomiting, Diarrhea, Constipation, Rash, Ear pain, Sore throat, Poor appetite, Unusually sleepy, Other. **One symptom per entry**; there is no "episode" grouping | plan Open value 9; `ui/src/lib/constants.ts` `SYMPTOMS`, `SYMPTOM_LABEL` |
| S2 | Severity optional: mild, moderate, severe, or unset | Open value 9; `SEVERITIES` |
| S3 | "Other" requires its text, 1–80 characters | Open value 9; `SYMPTOM_OTHER_MAX_LEN` = `SymptomOtherMaxLen` 80 |
| S4 | **Only Fever** carries a temperature, and it is **required** there, together with a method: Oral, Rectal, Armpit (API value `axillary`), Ear, Forehead, Other. Every other symptom refuses both | Open value 10; `TEMP_METHODS`, `TEMP_METHOD_LABEL` |
| S5 | Temperature accepted range **30.00–45.00 °C** (86.00–113.00 °F), checked on the °C value after conversion; up to **two decimals**, point or comma; digits only (`1e3`, `0x10`, `--1` refused) | Open value 10; `api/internal/constants/constants.go` `MinTempCHundredths` 3000, `MaxTempCHundredths` 4500; `ui/src/lib/server/entry-form.ts` `degrees` |
| S6 | Stored three ways: hundredths of °C, hundredths of °F, and the unit typed (exact). **The UI never converts**: it shows the column of the **viewing parent's** temperature preference (Account → Temperature °C/°F). Trailing zeros are dropped (38.40 → "38.4", 38.00 → "38") | spec 6.2; `ui/src/lib/entry-edit.ts` `temperatureText`; `ui/src/lib/growth.ts` `decimal` |
| S7 | Changing the symptom on an edit **clears what the new symptom has no place for**: leaving Fever clears temperature and method; leaving Other clears its text. Severity and note are independent and kept | `ui/src/lib/server/entry-form.ts` `readSymptom` |
| S8 | Entry time: at most 5 minutes in the future, not before the birth date, in the **child's** time zone | earlier plans; `ENTRY_FIELD_MESSAGES.at` |
| S9 | An edit sends only changed fields; an unchanged temperature in the same unit is never re-checked against the range | Global Constraints; `readSymptom` |
| S10 | No cap on the number of symptom entries | `constants.go` (no symptom cap) |
| S11 | No threshold, colour, label or advice for any temperature or symptom, anywhere | Open value 10 |

### 2.4 Courses and schedules

| # | Rule | Source |
|---|---|---|
| C1 | Name and dose are free text, 1–80 characters each ("2.5 ml" is a dose text, not a number the app reads) | Open value 11; `MEDICATION_TEXT_MAX_LEN` = `MedicationTextMaxLen` 80 |
| C2 | Schedule kinds: **As needed** (`none`: no reminders, no "Next due"), **At fixed times** (`times`: 1–**12** distinct `HH:MM` wall-clock times in the child's zone, sorted on save), **Every few hours** (`every`: a whole number **1–168** hours) | Open value 11; `MAX_SCHEDULE_TIMES` 12, `MIN_EVERY_HOURS` 1, `MAX_EVERY_HOURS` 168 |
| C3 | **Every N hours is a fixed grid from the course's start instant** (start, start + N h, …) in absolute time, unaffected by daylight-saving changes, and **never re-anchored to the last dose logged** | Owner (Open value 3); `api/internal/schedule` |
| C4 | Fixed times are **wall-clock** times in the child's zone, DST-safe: a time inside a spring-forward gap fires at the first valid instant after it (02:30 → 03:00); a repeated fall-back time fires **once**, at the earlier instant. The first occurrence is the first one at or after the start | Global Constraints; `api/internal/schedule` |
| C5 | Start: an instant entered as date + time in the child's zone; blank on create means **now**; may be in the future; not before the birth date | `api/internal/httpapi/medications.go` `mergeOnto`; `CourseFields.svelte` hints |
| C6 | Last day (optional): a calendar date in the child's zone, **inclusive to the end of that day**; must not be before the start's date. A course whose last day is before today is "ended" | Open value 11; `ui/src/lib/medication.ts` `courseEnded` |
| C7 | Active (checkbox, on by default): an inactive course has no reminders, no "Next due" and no Due-list entries, but still accepts doses | Open values 11, 12; `TestInactiveHasNoNextDue` |
| C8 | At most **50 courses per child** ("This child has the maximum number of medication courses") | Open value 32; `MaxMedicationsPerKid` |
| C9 | Editing schedule, start, last day or active replaces the course's **future, not-yet-sent** reminder rows; rows already done or already being delivered stay | Open value 14 |
| C10 | **Deleting a course deletes its doses, reminders and deliveries**, behind a confirmation that names the dose count | Owner (Open value 13) |
| C11 | A changed child time zone re-computes fixed-time courses in the new zone ("08:00" follows the zone); every-N-hours courses keep their absolute grid (only the displayed clock changes) | Open value 14; `api/internal/store/kids.go` |
| C12 | The app shows last and next only; it never computes, suggests or warns about an interval, a total, or a maximum per day | Task 20; spec 10.3 |

### 2.5 Doses

| # | Rule | Source |
|---|---|---|
| D1 | A dose belongs to one course of the same child; its course **cannot change** after logging | Open value 12 |
| D2 | One-tap "Log dose now" records **now** (server time); optional amount text 1–80 characters | `doses.go` (default `given_at` = now); `readDoseForm` |
| D3 | Edited time: not more than **5 minutes ahead**, not before birth; blank time on edit keeps the stored one | Open value 12; `MaxFutureSkew`; `buildEdit` |
| D4 | Loggable for any course: scheduled or as needed, active or not, ended or not | Open value 12 |
| D5 | At most **5000 doses per child** ("This child has the maximum number of logged doses") | Open value 32 |
| D6 | **A dose marks exactly one occurrence done**: the latest occurrence whose due time is no more than **30 minutes after** the dose (`DoseDoneLead`), provided the dose is before the following occurrence. So a dose up to 30 minutes **early** marks the coming occurrence; a **late** dose marks the missed one; a dose given more than 30 minutes before the next occurrence marks nothing new | Open value 15; `api/internal/store/doses.go` `markDone` |
| D7 | A done occurrence sends nothing more and leaves the Due list. **Editing or deleting a dose never undoes "done"** (no history) | Open value 15 |
| D8 | Doses are **not** on the timeline; they are listed on the Medication page only | Owner (Open value 27) |
| D9 | The page lists the **50 newest** doses (all courses, or one course with "Show doses"); there is no "Older" link, so older doses are not reachable for edit or delete from the UI | `TrackerPageDefault` 50; HANDOFF PR notes |

### 2.6 Reminders: occurrence, delivery, Due list

| # | Rule | Source |
|---|---|---|
| R1 | For every **active** course with a `times` or `every` schedule, the server keeps a row for each occurrence due in the **last 30 minutes** and for the **next** one ahead; "Next due" and the Due list read these rows. An occurrence first noticed more than 30 minutes late is never created | Open value 14 |
| R2 | **Delivery window (30 minutes):** nothing is ever sent later than due time + 30 minutes (`ReminderExpiry`) | Open value 16; `ReminderExpiry` |
| R3 | **Retries:** a temporary failure (rate limit, server error, timeout) is retried **30 s, 2 min, 10 min** after the previous attempt, never past the 30-minute window. A "gone" device (push service 404/410) is deleted with its choices. Other refusals are not retried; the device is kept | Open value 16; `ReminderRetryDelays` |
| R4 | At-most-once: a crash mid-send can lose one reminder, never send it twice | Open value 16 |
| R5 | **Who receives:** each device (Web Push) and each ntfy topic that a parent **opted in for that child**, if the parent is **enabled** and **holds a grant** on the child, the child's **medication module is on**, the course is **active and within its dates**, and that channel is **configured**. Checked again at send time | Global Constraints; Open values 16, 22 |
| R6 | **What it says:** title "Medication due for <child's name>"; no body; never the medication, the dose, the amount, a note or a temperature. Tapping opens `/kids/<child>/medication` | Owner (Open value 4); Open value 18; `api/internal/reminders/worker.go`; `ui/src/service-worker/index.ts` |
| R7 | **Due list, state "due":** an occurrence later **today** (before the end of the child's day) that is not done | Open value 25; `api/internal/store/reminders.go` `DueEntries` |
| R8 | **Due list, state "overdue":** a missed occurrence stays listed until **due time + the child's overdue minutes (default 60, 5–1440) or the course's next occurrence, whichever comes first**, then it disappears. At most **one overdue entry per course** (a 1-hour course with a 1440-minute setting shows one, never 24) | Owner (Open value 25); `DueEntries` |
| R9 | The two windows are independent: the 30-minute delivery window (R2) never shortens or extends the overdue window (R8) | Global Constraints |
| R10 | Due list order: medication entries first (by due time, then course), then vaccine visits as before; **3 lines in total** across both kinds (`HOME_DUE_MAX`), then one "N more on the Medication page" and/or "N more on the Vaccines page" line per hidden kind | Open value 25; `ui/src/lib/due.ts` |
| R11 | Nothing on the Due list for a child whose **medication module is off** or for an **inactive** course; vaccine entries unaffected | Open value 25 |
| R12 | Reminders exist only for medication; the payload carries a category so others can follow later | Owner (Open value 2) |

### 2.7 Channels and devices

| # | Rule | Source |
|---|---|---|
| N1 | **Both channels optional and independent.** Neither configured is a normal state: the Due list and Medication page work; the worker still keeps and expires rows | Owner (Open value 5); Open value 7 |
| N2 | Web Push off: the page says "Web Push is not configured on this server.", shows no enable button, and lists any devices from an earlier configuration with **Remove** only | Owner (Open value 29) |
| N3 | ntfy off: **no ntfy section at all** | Open value 29 |
| N4 | Devices: up to **10 per parent**; label 1–40 characters; a newly enabled device is named "This device"; **no child is opted in automatically** | Open value 21; `PushDevicesPerParent`, `PUSH_LABEL_MAX_LEN`; `subscribe/+server.ts` |
| N5 | **One browser, two parents:** when a second parent turns notifications on in the same browser, the device **moves** to that parent; the first parent's choices for it are deleted | Owner (Open value 21) |
| N6 | Each visit to the Notifications page with permission granted **re-sends** the browser's subscription. A renewed subscription (new address) becomes a new device that inherits the old one's name and children, and the old one is removed | Open value 21; `push-renew.ts` |
| N7 | ntfy: one random topic per parent (prefix + 22 characters), on/off, regenerate, per-child choices. **ntfy keeps each message up to 72 hours on its server, so the child's name rests there in plain text** for that long; the page says so | Open value 7; owner (privacy statement) |
| N8 | Who may opt in a child: any parent with a grant (guardian or caregiver) | Open value 22 |
| N9 | Revoking a grant or leaving a child deletes that parent's opt-ins **for that child** (devices and ntfy topic stay). **Disabling a parent** (admin) deletes their devices, ntfy topic and all opt-ins in the same step. **Deleting a child** removes every opt-in for it | Owner (Open value 23); Open value 22; plan Task 11 F18 |
| N10 | **Rotating the server's Web Push key** stops delivery to every device; on such a device the page says "Turn notifications off and on again on this device." | `docs/runbooks/reminders.md`; `+page.svelte` `OTHER_KEY` |
| N11 | iPhone/iPad: Web Push works only for the app **added to the Home Screen** (iOS/iPadOS 16.4+) and opened from there; the Home Screen app has its own login | Open value 29; research notes |

### 2.8 Modules and age bands

| # | Rule | Source |
|---|---|---|
| M1 | `sickness` and `medication` are **on by default in every age band** (0–11, 12–23, 24+), so the band-crossing rule never applies to them | Open value 28 |
| M2 | Switching a module off **hides entry points** (quick-log tile; for medication also the Due entries and the sending) and **keeps all data**. The Medication page, the symptom log/edit pages and the timeline filter remain reachable by URL; the timeline's **default** filter drops "Symptom" when sickness is off | Owner (Open value 8); code: no module gate on those routes |

### 2.9 Caps and limits (Built, `constants.go` / `ui/src/lib/constants.ts`)

| What | Limit |
|---|---|
| Courses per child | 50 |
| Doses per child | 5000 |
| Devices per parent | 10 |
| Course name, dose text, dose amount | 80 characters |
| "Other" symptom text | 80 characters |
| Device name | 1–40 characters |
| Fixed times per course | 1–12, distinct |
| Every N hours | 1–168 |
| Temperature | 30.00–45.00 °C (86.00–113.00 °F), two decimals |
| Overdue minutes | 5–1440, default 60 |
| Note | 500 characters |
| Delivery window / done lead | 30 min / 30 min |
| Retries | 30 s, 2 min, 10 min |
| Home Due lines | 3 (+ one "more" line per hidden kind) |
| Doses listed | 50 newest |

### 2.10 Identity and privacy

- No grant, an unknown child, another child's symptom, course or dose, and a malformed id all show the **same
  "Not found" page**. Never hint that a child or record exists.
- The admin never sees a child, a symptom, a course, a dose, a device's children or a reminder.
- Notification text never carries a medication name, dose, amount, note or temperature. The Due list on the
  **home screen** does show the **course name** (on screen only).

---

## 3. Flows (step by step, with branches)

1. **Log a symptom (not fever).** Home → quick-log **Sickness** → page "Log a symptom", time prefilled to now
   in the child's zone → choose a Symptom (e.g. Cough) → optional Severity (Mild / Moderate / Severe / Not
   set) → Note → **Save** → redirected to the timeline day; row `09:12 Cough, moderate`.
   - No symptom chosen → "Choose a symptom" under the select.
   - Time in the future or before birth → "Enter a time that is not in the future and not before the birth date".
2. **Log a fever.** Choose **Fever** → the **Temperature** input with its unit select (default the parent's
   preference, °C or °F) and the **Method** select appear (CSS, no JS) → type `38.4` or `38,4` → choose Ear →
   Save → `09:12 Fever 38.4 °C (ear)`; with a severity `09:12 Fever 38.4 °C (ear), mild`.
   - Out of range or malformed → "Enter a temperature from 30 to 45 °C (86 to 113 °F), with up to two
     decimals"; the unit select is described by the same error; every typed value kept.
   - No method → "Choose how the temperature was taken".
   - Unit conversion is **shown, never computed by the UI**: a parent whose preference is °F sees that same
     entry as `101.12 °F` (the API's stored °F column), on the timeline and prefilled in the edit form.
3. **Log "Other".** Choose **Other** → an **Other symptom** text field appears → type up to 80 characters →
   `09:12 Other: <text>`. Blank → "Enter what the symptom is, up to 80 characters".
4. **Change a symptom's type on edit.** Timeline row → "Edit symptom" (prefilled) → switch Fever → Cough →
   Save: temperature and method are **cleared** for good; switch Other → Rash: its text is cleared. Switching
   to Fever requires a temperature and method in the same save. Without JS the fever/other fields appear and
   hide live (CSS); hidden fields are ignored on save.
5. **Delete a symptom.** Edit page → `<details>` "Delete this entry" → "This removes the entry for everyone
   with access to {child}." → **Yes, delete this entry** → back to the child's timeline.
6. **Create a fixed-times course.** Home → Medication link (or quick-log **Medication**) → "Medication for
   {child}" → **Add a course** → Name, Dose → Schedule radio **At fixed times** → the twelve "Time 1"…"Time 12"
   inputs appear (CSS) → fill 08:00 and 20:00 (blanks ignored) → Start (blank = now) → Last day optional →
   Active ticked → Save → back to the Medication page; the card reads `08:00, 20:00`, "No dose logged yet",
   "Next due 20:00" (or "Next due 2026-11-24 08:00" when not today).
   - A duplicate time, 13 times, or none → "Choose as needed, up to 12 different times of day, or a whole
     number of hours from 1 to 168".
   - 51st course → page-level "This child has the maximum number of medication courses".
7. **Create an every-N-hours course.** Schedule **Every few hours** → "Hours between doses" (hint "A whole
   number from 1 to 168, counted from the start below") → 6 → Start 2026-11-23 08:00 → card reads `Every 6
   hours from 08:00`; occurrences 08:00, 14:00, 20:00, 02:00 … on an absolute grid, never moved by a dose.
8. **One-tap dose.** Medication page → on the course card, optional **Amount (optional)** → **Log dose now**
   (disabled while the request runs, so one tap logs one dose) → the page reloads in place; the card reads
   "Last dose logged at 14:05"; the dose appears first under "Recent doses". If the list was filtered to that
   course ("Show doses"), it stays filtered.
   - Amount over 80 characters → message under that card's Amount: "Enter an amount of up to 80 characters".
   - Course deleted meanwhile → page-level "This course is no longer in the list".
   - 5001st dose → page-level "This child has the maximum number of logged doses".
9. **Dose given early.** A 14:00 occurrence; dose logged at 13:40 (within 30 minutes before) → 14:00 is
   marked done: no notification is sent for it, it never appears on the Due list, "Next due" moves on to the
   following occurrence.
   Dose at 13:20 (more than 30 minutes before) → 14:00 is **not** marked; it will still be sent and listed.
10. **Dose given late.** 08:00 missed; Due list shows `Calpol · 08:00 · overdue`; a dose at 08:40 marks 08:00
    done → the overdue line disappears on the next load.
11. **A reminder cycle, end to end.** Course with 14:00 today, overdue setting 60:
    a. In the morning the home Due list shows `Calpol · 14:00` (state due), and the card "Next due 14:00".
    b. At 14:00 the server sends "Medication due for Mia" to every opted-in device and ntfy topic (if a channel
       is configured); retries for up to 30 minutes on temporary failures.
    c. From 14:00 the line reads `Calpol · 14:00 · overdue`; the card's "Next due" already shows the
       following occurrence.
    d. The line leaves at 15:00 (14:00 + 60 min), or earlier if the course's next occurrence comes first. With
       the setting at 1440 and an every-1-hour course, the 14:00 line leaves at 15:00 because the 15:00
       occurrence is then due (and gets its own line): never more than one overdue line per course.
    e. A dose logged at any point from 13:30 removes it.
    With **no channel configured**, a–e happen identically, minus the sending in b.
12. **Enable Web Push on a phone.** Account → **Notifications** → section "This device" → **Turn on
    notifications on this device** (JS only) → the browser's own permission prompt (OS UI) →
    - **Allow** → the device appears as an editable form: "Device name" (`This device`), "Reminders on this
      device for" with one checkbox per child (**none ticked**) → tick children → **Save** → "Saved".
    - **Block** → "Notifications are blocked for this site. Allow them in the browser's settings, then try again."
    - **Dismiss** → "Notifications were not turned on. You can try again."
    - Error → "Notifications could not be turned on. Try again."; device cap → "You have the most devices
      allowed. Remove one, then try again."; server refuses the address → "The server refused this browser's
      notification address. Try again later."
    - **iPhone/iPad in Safari** (not opened from the Home Screen): no button; the sentence "On iPhone and iPad,
      add this app to your Home Screen first, then open it from there to turn on notifications." The parent adds
      to Home Screen (OS UI), opens it there, **signs in again** (separate login), returns to Notifications.
    - A browser without push support: "This browser cannot receive notifications."
    - JavaScript off: no button; the line "Turning on notifications needs JavaScript, and the browser asks for
      your permission." stays.
13. **A second parent in the same browser.** Parent B signs in on A's tablet and turns notifications on → the
    device **moves** to B (new id), named after B's earlier device in this browser if B's cookie names one,
    else "This device"; **A's children choices for it are deleted**. A, on another device, no longer sees it in
    "Your other devices". Nothing warns either parent (section 7).
14. **Subscription renewal.** The browser silently renews its push address. On the parent's next visit to the
    Notifications page, the page re-sends it: the server creates a new device, the page copies the old
    device's name and children onto it and removes the old one. If a reminder is sent to the dead address
    first, the server deletes that device and its choices; the next visit registers the browser again **with
    no children ticked** (by design; there is no background handler).
15. **Server key rotation.** Every device stops receiving. Opening Notifications on such a device shows "Turn
    notifications off and on again on this device." → **Turn off on this device** → **Turn on notifications on
    this device** → tick children again.
16. **Turn off on this device.** **Turn off on this device** → the server removes the device and its choices;
    the browser unsubscribes; the page shows the enable button again. A device in "Your other devices" is
    removed with **Remove**. Signing out of the app does **not** turn notifications off (section 7).
17. **ntfy.** (Only when the server has ntfy.) **Turn on ntfy** → the page shows Server and Topic, "Subscribe to
    this topic on this server in the ntfy app.", "Reminders through ntfy for" checkboxes (none ticked) and
    **Save** → "Saved". **New topic** → a new topic replaces the old one at once (**no confirmation**; children
    choices kept); the parent must subscribe to the new one in the ntfy app. **Turn off ntfy** → topic and
    choices deleted.
18. **Revoke, leave, disable.** A guardian removes a caregiver's access (or a caregiver leaves): that person's
    reminders for that child stop and their choices for that child are deleted; their devices and topic stay
    for other children. An admin disables a parent: all their devices, topic and choices are deleted; if
    re-enabled they start from nothing.
19. **Delete a course.** Course edit page → `<details>` "Delete this course" → "This removes the course for
    everyone with access to {child}." + "This also deletes the {N dose|N doses} logged for this course." →
    **Yes, delete this course** → doses, reminders and deliveries are gone for everyone.
20. **Delete a child.** Unchanged confirmation (type the name); it now also removes symptoms, courses, doses,
    reminders and every parent's notification choices for that child.
21. **Change the child's time zone.** Kid settings → Time zone → Save: fixed-time courses now fire at the same
    wall-clock times in the new zone; every-N-hours keep their absolute grid; every displayed clock follows the
    new zone. No warning is shown (section 7).
22. **Set the overdue window.** Kid settings (guardian) → "Show a missed dose as overdue for (minutes)" → 15 →
    Save → "Saved". 4 or 1441 or blank → "Enter minutes from 5 to 1440", typed value kept.

---

## 4. New screens as built

All new kid pages share the plans 3–5 shell: a plain underlined link back (the child's name, or "Medication for
{child}" on the course and dose pages), an `h1`, one narrow column (`max-w-md`) for forms, "Times are in
{zone}" as a muted line. HTML titles follow "{Page} · {child} · happybabytracker".

### 4.1 Sickness: log and edit (`/kids/{kid}/log/symptoms`, `/kids/{kid}/log/symptoms/{entry}`)

**Who:** guardian and caregiver. **Purpose:** record one symptom. **Entry:** quick-log "Sickness"; edit from a
timeline row. **h1:** "Log a symptom" / "Edit symptom". No advanced toggle (plan 5.1's toggle is **not** used
here).

Layout, top to bottom (inside one form, `novalidate`):

| Region | Control | Label / text | Notes |
|---|---|---|---|
| Page message | `role="alert"` line | (server failures only) | |
| Zone | muted text | "Times are in {zone}" | |
| Time | `datetime-local`, required | "Time" | prefilled now (new) or stored (edit) |
| Symptom | native select, required | "Symptom" | first option "Choose a symptom"; then the 12 labels of S1 |
| Severity | radiogroup (fieldset, legend) | "Severity" | radios "Mild", "Moderate", "Severe", "Not set" (each row ≥ 44 px); "Not set" checked on a new form |
| Other symptom | text, shown only while "Other" is selected (CSS) | "Other symptom" | |
| Temperature | text (`inputmode=decimal`) + select (96 px) | "Temperature"; select sr-only "Temperature unit" | shown only while "Fever" is selected; select options "°C", "°F", default the parent's preference |
| Method | native select | "Method" | first option "Choose a method"; "Oral", "Rectal", "Armpit", "Ear", "Forehead", "Other"; shown only with Fever |
| Note | text | "Note" | |
| Actions | button + link | "Save", "Cancel" (to the child's home) | |
| Edit only | `<details>` | "Delete this entry" → "This removes the entry for everyone with access to {child}." → "Yes, delete this entry" | |

States:

| State | What shows |
|---|---|
| New, empty | Time = now; Symptom "Choose a symptom"; Severity "Not set"; fever and other fields hidden |
| Fever chosen | Temperature + unit + Method visible |
| Other chosen | "Other symptom" visible |
| Edit, loaded | Every value prefilled; temperature in the **viewer's** unit (from the API's column), unit select = viewer's preference |
| Field error | message under the field (`role="alert"`), field `aria-invalid`; all typed values kept (table in appendix A) |
| API failure on save | page-level alert: "The app cannot reach its server, try again soon" or "Something went wrong, try again" |
| No grant / unknown entry | "Not found" page |
| Module off | page still works by URL; no quick-log tile |
| No JS | identical (CSS reveal; full POST; 303 to the timeline day) |
| JS, before hydration | **design constraint:** the server-rendered form is visible before Svelte takes over; hydration re-applies the loaded select/radio values, so an edit typed in the first instant can be overwritten (the tests wait for hydration). Do not design anything that invites typing during page load (e.g. autofocus into a prefilled field) |
| 390 px, long text | "Other" text up to 80 characters; the timeline row wraps |
| Loading | no spinner; a normal form post |

Accessibility: the temperature unit select is labelled "Temperature unit" (visually hidden) and is described by
the temperature error; the radiogroup is labelled by its legend; the hidden fever/other blocks are
`display:none` (not in the accessibility tree) until chosen.

### 4.2 Timeline: symptom rows and filter

- Filter: the "Show" checkbox group gains **"Symptom"** (last, after Solids). Default ticked when the
  sickness module is on.
- Row (one link to the edit page, then the note in muted text): `{HH:MM} {Symptom}{ reading}{, severity}`:
  - `09:12 Cough` · `09:12 Cough, moderate`
  - `09:12 Fever 38.4 °C (ear)` · `09:12 Fever 101.12 °F (armpit), mild` (method lower-case; "axillary" reads
    "armpit")
  - `09:12 Other: <text>` (the typed text; wraps at 390 px)
- No symptom count in the day's totals line (the totals are unchanged). Doses never appear.

### 4.3 Medication page (`/kids/{kid}/medication`)

**Who:** guardian and caregiver (the same page). **Purpose:** see courses, log a dose now, see recent doses.
**Entry:** the "Medication" link under the child's name (always shown, every role, even with the module off),
the quick-log tile "Medication" (module on), the home Due lines, and a tap on a notification.

Layout:

1. Back link "{child}". Header row: `h1` "Medication for {child}" and the link **"Add a course"**.
2. Muted "Times are in {zone}".
3. Page-level `role="alert"` (failures that are not about an amount).
4. **Section "Courses"** (`h2`):
   - No course at all → "No courses yet". Courses exist but none is active and current → "No active courses".
   - One **card** per active, not-ended course (newest start first). Card content:
     | Line | Text |
     |---|---|
     | `h3` link to edit | the course **name** |
     | dose | the dose text (e.g. "2.5 ml") |
     | schedule (muted) | "As needed" · "08:00, 20:00" · "Every 6 hours from 08:00" · "Every hour from 08:00" |
     | last | "Last dose logged at 14:05" (today) / "Last dose logged at 2026-11-22 21:40" (other day) / "No dose logged yet" |
     | next (only when the server has one) | "Next due 20:00" / "Next due 2026-11-24 08:00" |
     | one-tap form | input "Amount (optional)"; button **"Log dose now"** (accessible name "Log dose now of {name}"); form name "Log a dose of {name}" |
     | filter link | "Show doses" (accessible name "Show doses of {name}") |
   - `<details>` **"Ended and inactive courses ({n})"** with the same cards (an inactive course shows no "Next
     due"; it can still log a dose). It opens on its own when one of its cards has an error.
5. **Section "Recent doses"** (`h2`; with a filter: "Doses of {name}" and a link "Show all doses"):
   - Empty → "No doses logged yet".
   - Rows (newest first, up to 50): a link "{course name}, {time}" (time as "14:05" or "2026-11-22 21:40") to
     the dose edit page; the amount on its own line; the note muted.

States:

| State | What shows |
|---|---|
| Empty | "No courses yet"; "No doses logged yet" |
| Loaded | cards and doses as above |
| Logging (JS) | the tapped **Log dose now** is disabled (shadcn `disabled:opacity-50`) until the answer; no fade on re-enable (a fade let axe read an enabled button at 3.3:1, fixed in the last commit) |
| Logged | in place reload; amount cleared; "Last dose logged at …" updated; no confirmation text |
| Amount error | under that card's Amount field |
| Course gone / cap / server | page-level alert (copy in appendix A) |
| Filter `?medication=` unknown | "Not found" page |
| Module off | page works by URL; no quick-log tile; no Due entries; no reminders sent |
| No channel configured | identical page (nothing on this page depends on a channel) |
| No JS | every action works (full POST, 303 back to this page, filter kept) |
| 390 px | names, doses and notes up to 200 characters unbroken wrap anywhere (no horizontal scroll) |

### 4.4 Course create and edit (`/kids/{kid}/medication/new`, `/kids/{kid}/medication/{course}`)

**h1:** "Add a course" / "Edit course" (edit adds the course name as a muted line under the `h1`). Back link
"Medication for {child}".

| Region | Control | Label / text |
|---|---|---|
| Zone | muted | "Times are in {zone}" |
| Name | text | "Name" |
| Dose | text | "Dose" |
| Schedule | fieldset legend "Schedule" with a radiogroup | radios "As needed" (default on create), "At fixed times", "Every few hours" |
| Fixed times | shown only with "At fixed times" (CSS): **all twelve** `time` inputs at once, 2 columns (3 from 640 px) | "Time 1" … "Time 12" (blanks ignored) |
| Every | shown only with "Every few hours" (CSS) | "Hours between doses", hint "A whole number from 1 to 168, counted from the start below" |
| Start | `datetime-local` | "Start"; hint on create "Leave blank to start now", on edit "Leave blank to keep the current start" |
| Last day | `date` | "Last day (optional)" |
| Active | checkbox (20 px), ticked by default | "Active" |
| Note | text | "Note" |
| Actions | | "Save", "Cancel" (to the Medication page) |
| Edit only | `<details>` | "Delete this course" → "This removes the course for everyone with access to {child}." → "This also deletes the {n} dose(s) logged for this course." (templates: "1 dose", "N doses", including "0 doses") → "Yes, delete this course" |

Schedule errors land in one message under the radios ("Choose as needed, up to 12 different times of day, or
a whole number of hours from 1 to 168"); the time inputs or the hours input of the chosen kind carry
`aria-invalid` and point to it. Save redirects to the Medication page (no confirmation). An edit sends only what
changed. No JS: identical (CSS reveal; full POST).

### 4.5 Dose edit (`/kids/{kid}/medication/doses/{dose}`)

`h1` "Edit dose"; muted "Dose of {course name}"; "Times are in {zone}"; **"Given at"** (`datetime-local`),
**"Amount"**, **"Note"**; Save / Cancel; `<details>` "Delete this dose" → "This removes the dose for everyone
with access to {child}." → "Yes, delete this dose". The course cannot be changed. Errors per field (appendix
A). Deleting or editing never reverses a "done" reminder.

### 4.6 Notifications page (`/account/notifications`)

**Who:** every parent (it is about the parent and their devices, not one child). **Entry:** a new
"Notifications" link at the top of Account. **h1:** "Notifications". A `role="status"` "Saved" line after a
device or ntfy-children save; a page-level `role="alert"` for failures.

**A. Web Push on (server has a key)** — section `h2` "This device":

| Condition (in this order) | Shows |
|---|---|
| iPhone/iPad outside the Home Screen app | "On iPhone and iPad, add this app to your Home Screen first, then open it from there to turn on notifications." (no button, no help line) |
| This browser is a registered device of this parent | the **device form**: "Device name" (text, max 40, default "This device"); fieldset "Reminders on this device for" with one checkbox per child the parent has; "Save"; then a separate button **"Turn off on this device"** |
| Browser cannot do push | "This browser cannot receive notifications." |
| Otherwise | button **"Turn on notifications on this device"** (only after JS starts) and the help "Turning on notifications needs JavaScript, and the browser asks for your permission." |
| Problems (`role="alert"`) | "Notifications are blocked for this site. Allow them in the browser's settings, then try again." · "Notifications were not turned on. You can try again." · "Notifications could not be turned on. Try again." · "Turn notifications off and on again on this device." (key rotated) · "This device could not be checked with the server. Reload the page to try again." · server answers in appendix A |

Then, if any, section `h2` "Your other devices": one row each: the device name (bold), "Added 2026-11-23 18:04
UTC" (muted; this date is UTC, not the child's zone), and **"Remove"** (accessible name "Remove {device name}").
Other devices' children are **not** shown or editable here.

**B. Web Push off** — section `h2` "Web Push": "Web Push is not configured on this server."; if the parent has
devices from an earlier configuration, the list "Your devices" with **Remove** only. No enable button.

**C. ntfy (only when the server has ntfy)** — section `h2` "ntfy":

- Always first: "ntfy keeps each message for up to 72 hours on its server, so your child's name is stored there
  in plain text for that long."
- Off for this parent: button **"Turn on ntfy"**.
- On: a definition list **Server** / **Topic** (both break anywhere; topic in `<code>`); muted "Subscribe to
  this topic on this server in the ntfy app."; fieldset "Reminders through ntfy for" (one checkbox per child)
  + "Save"; buttons **"New topic"** and **"Turn off ntfy"**; muted help "A new topic replaces this one:
  subscribe to the new one in the ntfy app." (describes "New topic"). Neither button asks for confirmation.

States and JS:

| State | What shows |
|---|---|
| No channel at all | section B only; no ntfy section |
| No JS | everything renders; device form, Remove, all ntfy actions work; **only the enable button is absent** |
| Device removed elsewhere | the page simply shows the enable button again |
| Device form label error | "Enter a name of 1 to 40 characters" under Device name |
| A child id not in the form's list | page-level "Choose from your children"; a child the parent no longer has a grant on → "Not found" page (code-read) |
| ntfy turned off on the server meanwhile | "ntfy is not configured on this server." |
| Endpoint and keys | never in the page, its data or any message |

### 4.7 Kid settings: the overdue field (`/kids/{kid}/settings`, guardian only)

After "Vaccination schedule", before Save: label **"Show a missed dose as overdue for (minutes)"**, a number
input (min 5, max 1440, step 1), hint **"From 5 to 1440 (24 hours). A missed dose leaves the home list after
this time, or when the next dose of that course is due, whichever comes first."**, error "Enter minutes from 5
to 1440". Only a changed value is sent. **The create-child form does not show it** (new children get 60).
Caregivers have no settings page.

### 4.8 Home: Due list and quick-log tiles

**Due** (`h2` "Due", then underlined links; the whole section is absent when empty):

| Line | Template | Link |
|---|---|---|
| Medication, due later today | `{course name} · {HH:MM}` | Medication page |
| Medication, overdue | `{course name} · {HH:MM} · overdue`, or `{course name} · {YYYY-MM-DD HH:MM} · overdue` when it was before today | Medication page |
| Vaccine visit (unchanged) | `{visit} vaccines · Overdue` / `· Due` | Vaccines page |
| Hidden medication | `{n} more on the Medication page` | Medication page |
| Hidden vaccines | `{n} more on the Vaccines page` | Vaccines page |

Medication lines come first; at most 3 lines in total before the "more" lines. Long names wrap anywhere. Note
the casing difference as built: vaccine "· Overdue" (capital) versus medication "· overdue".

**Quick-log tiles** (2-column grid, label only; "Sickness" opens the symptom form, "Medication" opens the
Medication page). Order per band with the default modules:

| Band | Tiles |
|---|---|
| 0–11 months | Bottle, Nursing, Diaper, Sleep, Pumping, Feeding, Growth, Vaccination, **Sickness**, **Medication** |
| 12–23 months | Feeding, Bottle, Sleep, Diaper, Nursing, Pumping, Growth, Vaccination, **Sickness**, **Medication** |
| 24 months + | **Sickness**, **Medication**, Growth, Vaccination |

**Under the child's name** (`KidLinks`): "Vaccines", **"Medication"** (new; every role, not gated by the
module), then for guardians "Settings", "Access", "Modules".

### 4.9 Account: one new link

At the top of Account, under `h1` "Account", a plain link **"Notifications"** (before the Profile section).

### 4.10 The OS notification (copy, not UI)

| Part | Value |
|---|---|
| Title | "Medication due for {child's name}" |
| Fallback title (unreadable payload) | "Reminder due" (tap opens the app's start page) |
| Body | empty |
| Icon | the app's 192 px icon |
| Tag | one per occurrence, so a retry replaces rather than stacks |
| Tap | opens or focuses `/kids/{child}/medication` |
| ntfy message | title and message both "Medication due for {child's name}"; tapping opens the same page |

How it looks is the OS's and the ntfy app's business (**Unverified** on real devices).

### 4.11 Health (operators, not a designed surface)

`/healthz/api` now also means "the reminder worker ticked within 60 seconds"; new `/healthz/reminders` (200
`ok` / 503 `unavailable`, 503 while a delivery is stuck or failed in the last 15 minutes; always 200 with no
channel). Plain text bodies, nothing to design.

---

## 5. What changed in existing screens and rules

| Existing screen / rule | Change |
|---|---|
| Home Due list | Union of medication entries (first) and vaccine visits; one shared cap of 3; per-kind "more" lines. Vaccine-only behaviour is unchanged when a child has no course |
| Home quick-log | Two new tiles, "Sickness" and "Medication"; order per band in 4.8; Sickness/Medication lead the 24+ band |
| Today list, daily totals | Unchanged: no symptom or dose figures |
| Kid links under the name | New "Medication" link for every role |
| Timeline | New filter checkbox "Symptom"; symptom rows; default filter follows the sickness module |
| Entry form shell | Reused unchanged for symptoms (Times are in…, Note, Save, Cancel, delete `<details>`). Plan 5.1's **advanced toggle pattern is not reused** (no symptom toggle); its **CSS `:has()` reveal** is reused for Other and Fever, and for the course schedule kinds |
| Kid settings | One new field (4.7). The create form is unchanged |
| Kid JSON | Carries `reminder_overdue_minutes` (every kid view) |
| Modules page | `sickness` and `medication` now have entry points; both "On by default for this age" in every band |
| Delete a child | Now also removes symptoms, courses, doses, reminders and every notification choice for the child (wording unchanged: "every record") |
| Access: revoke / leave | Also deletes that parent's notification choices for that child |
| Admin: disable a parent | Also deletes the parent's devices, ntfy topic and choices |
| Account | New "Notifications" link |
| Health | `/healthz/api` includes the worker; new `/healthz/reminders` |
| PWA | A service worker now exists (install, push, notification click only). **No fetch handler, no caching, no offline page** (owner) |
| Child time zone change | Also re-computes fixed-time medication occurrences |

---

## 6. Weaknesses (honest audit; evaluate and add your own)

1. **Small targets.** "Log dose now" and "Save" are 32 px high; "Turn off on this device", "Remove", "New topic"
   and "Turn off ntfy" are 28 px (`size="sm"`); native selects and inputs are 32 px; the children checkboxes on
   the Notifications page are 16 px with no row padding.
2. **No feedback after a dose** beyond the "Last dose logged at" line changing; a parent at 3 a.m. may tap twice
   on a slow network (the button is disabled during the request only with JS).
3. **"Next due" and the Due list can disagree in the parent's eyes**: once a dose's time passes without a dose,
   the card already shows the following occurrence while the home says "… · overdue".
4. **Twelve time inputs** appear at once for "At fixed times"; most courses need two or three.
5. **Enabling Web Push opts in no child.** After the permission prompt the parent must also tick children and
   Save; nothing says so. A device with no child receives nothing.
6. **The device move is silent** (flow 13): neither parent is told; on a shared tablet the last parent to
   enable wins.
7. **Sign-out does not unsubscribe** the device: on a shared tablet the previous parent keeps receiving.
8. **"New topic" has no confirmation**, though it breaks every phone subscribed to the old topic.
9. **iOS path** is a sentence only; the Home Screen app needs its own sign-in; nothing guides the return trip.
10. **"Added … UTC"** on devices uses UTC while everything else uses the child's zone.
11. **Casing**: "· Overdue" (vaccine) vs "· overdue" (medication) on the same list.
12. **Only the 50 newest doses** are reachable for edit or delete; no "Older" link.
13. **Hydration window**: editing a prefilled field in the first instant after load can be overwritten (4.1).
14. **The Medication link stays visible with the module off**, while the tile disappears (open item 7.1).
15. **Course delete with "0 doses"** reads "This also deletes the 0 doses logged for this course" (open item).
16. **No time-zone-change warning**, though it moves every fixed-time reminder.
17. **No visual distinction** between due and overdue beyond the word.
18. **Known edge (Unverified; noted in an uncommitted HANDOFF edit after `921c1a1`, not in the commit):** when a
    course is edited minutes after an early dose, an overdue line of the new schedule can stay hidden from the
    Due list for up to the child's overdue minutes. Do not design around it; it is listed so a reviewer is not
    surprised.

---

## 7. Open items (mark each as an assumption in your design; do not decide them)

### 7.1 Owner questions

| # | Question | As built | Your assumption |
|---|---|---|---|
| 1 | Should the "Medication" link under the child's name hide when the medication module is off? | Always shown, every role | Keep shown; draw a variant hidden, labelled "needs owner decision" |
| 2 | Should "New topic" (ntfy) ask for confirmation? | No confirmation | Draw as built; a `<details>` second step is a UI-only option |
| 3 | Sign-out leaves this device subscribed (shared tablet) | Subscribed; reminders keep arriving | Draw as built; you may propose a hint near sign-out, labelled |
| 4 | Re-enabling after the device cookie is gone renames the device to "This device" | Renamed | Draw as built |
| 5 | A `has_topic` flag so a stale ntfy topic can be seen while ntfy is off | Not shown (invisible while ntfy is off) | Draw as built (no ntfy section) |
| 6 | A warning when a guardian changes the child's time zone | None | Draw as built; optional one-line notice as a labelled variant (see `answers-to-design-agent-questions.md` §6) |
| 7 | Course delete wording when the count is 0 ("0 doses") | "This also deletes the 0 doses logged for this course." | Draw as built; a variant that omits the line at 0, labelled |
| — | The API's memory ceiling | infrastructure only | Omit from the design |

### 7.2 Design questions (answer or label; tag each API change cheap / costly)

1. A one-hand dose control: a larger "Log dose now" per card, with a `role="status"` confirmation line after
   the redirect (UI-only: cheap).
2. Fixed times: show 2–3 time inputs with a no-JS "more times" disclosure (`<details>`) for the rest (UI-only:
   cheap).
3. After enabling Web Push, put the children checkboxes in front of the parent with a one-line instruction
   (UI-only: cheap); auto-opting in the current child is an owner reversal (costly in rules, cheap in code).
4. Presenting "Next due" vs "overdue" so a parent sees both without contradiction (UI-only: cheap).
5. A temperature presentation that is a plain record (number, unit, method) with no visual scale (UI-only).
6. Symptom picker: 12 options in a select vs a tappable grid of radios (CSS-only reveal must keep working).
7. Where the ntfy privacy sentence and the iOS sentence sit so they are read at the right moment.

---

## 8. What you must NOT do

- **No interpretation of a temperature, symptom, dose or schedule**: no "fever"/"high"/"normal"/"low" labels on
  a number, no colour scale, no thermometer graphic with zones, no icon that changes with the value, no "safe
  interval", "too soon", "max per day", "missed too many", no advice, no link to medical guidance.
- **No computation the server did not do**: no interval suggestion, no next-dose calculation in the UI, no
  unit conversion in the UI (show the API's column for the viewer's unit).
- **No escalation for "overdue"**: no red, no alarm icon, no repeated notification, no badge count on the app
  icon. "overdue" is a neutral word that disappears by rule.
- **Never put a medication name, dose, amount, note or temperature in a notification**, and never change "Medication
  due for {child's name}".
- **No JS-only path** except the existing enable button. No fetch handler, no offline page, no caching (owner).
- **Never show an endpoint, key or token**; never show the admin any of this.
- Do not put doses on the timeline (owner); do not make the overdue minutes editable by caregivers or on the
  create form without labelling it an owner reversal.
- Do not rename the accessible names in appendix B without listing the rename.

---

## 9. Appendix A — copy, verbatim (grouped by screen)

Dynamic parts in `{braces}`. Copy is from the files listed in section 13.

**Sickness form** (`TRACKER_TITLE`, `SymptomFields.svelte`, `TemperatureField.svelte`, `RadioField.svelte`,
`TrackerForm.svelte`, `entry-failure.ts`)

| Element | Text |
|---|---|
| h1 | Log a symptom / Edit symptom |
| Zone line | Times are in {zone} |
| Fields | Time · Symptom · Severity · Other symptom · Temperature · Temperature unit (sr-only) · Method · Note |
| Symptom options | Choose a symptom · Congestion · Cough · Fever · Vomiting · Diarrhea · Constipation · Rash · Ear pain · Sore throat · Poor appetite · Unusually sleepy · Other |
| Severity radios | Mild · Moderate · Severe · Not set |
| Unit options | °C · °F |
| Method options | Choose a method · Oral · Rectal · Armpit · Ear · Forehead · Other |
| Buttons / links | Save · Cancel |
| Delete | Delete this entry · This removes the entry for everyone with access to {child}. · Yes, delete this entry |
| Error: time | Enter a time that is not in the future and not before the birth date |
| Error: symptom | Choose a symptom |
| Error: severity | Choose mild, moderate or severe, or leave it unset |
| Error: other | Enter what the symptom is, up to 80 characters |
| Error: temperature | Enter a temperature from 30 to 45 °C (86 to 113 °F), with up to two decimals |
| Error: method | Choose how the temperature was taken |
| Error: note | The note is too long |

**Timeline** (`timeline.ts`, `constants.ts`)

| Element | Text |
|---|---|
| Filter checkbox | Symptom |
| Row templates | `{HH:MM} {Symptom}` · `{HH:MM} {Symptom}, {mild\|moderate\|severe}` · `{HH:MM} Fever {n} {°C\|°F} ({method lower-case})` (+ `, {severity}`) · `{HH:MM} Other: {text}` |

**Medication page** (`+page.svelte`, `CourseCard.svelte`, `DoseForm.svelte`, `medication.ts`, `medication-form.ts`)

| Element | Text |
|---|---|
| h1 / link | Medication for {child} · Add a course |
| Zone line | Times are in {zone} |
| Sections | Courses · Ended and inactive courses ({n}) · Recent doses · Doses of {course name} · Show all doses |
| Empty | No courses yet · No active courses · No doses logged yet |
| Schedule label | As needed · `{HH:MM}, {HH:MM}…` · Every hour from {HH:MM} · Every {n} hours from {HH:MM} |
| Last / next | Last dose logged at {HH:MM \| YYYY-MM-DD HH:MM} · No dose logged yet · Next due {HH:MM \| YYYY-MM-DD HH:MM} |
| One-tap | Amount (optional) · Log dose now (sr: " of {name}") · form name: Log a dose of {name} · Show doses (sr: " of {name}") |
| Dose row | {course name}, {HH:MM \| YYYY-MM-DD HH:MM} |
| Errors | Enter an amount of up to 80 characters · This course is no longer in the list · This child has the maximum number of logged doses · The app cannot reach its server, try again soon · Something went wrong, try again |

**Course create / edit** (`CourseFields.svelte`, `ScheduleFields.svelte`, `new/+page.svelte`, `[medication]/+page.svelte`, `medication-form.ts`)

| Element | Text |
|---|---|
| h1 | Add a course / Edit course (+ muted course name) |
| Back link | Medication for {child} |
| Fields | Name · Dose · Schedule · Time 1 … Time 12 · Hours between doses · Start · Last day (optional) · Active · Note |
| Schedule radios | As needed · At fixed times · Every few hours |
| Hints | A whole number from 1 to 168, counted from the start below · Leave blank to start now (create) · Leave blank to keep the current start (edit) |
| Buttons | Save · Cancel |
| Delete | Delete this course · This removes the course for everyone with access to {child}. · This also deletes the {n} {dose\|doses} logged for this course. · Yes, delete this course |
| Errors | Enter a name of up to 80 characters · Enter a dose of up to 80 characters · Choose as needed, up to 12 different times of day, or a whole number of hours from 1 to 168 · Enter a start that is not before the birth date · Enter a last day that is a real date and not before the start · Choose whether the course is active · The note is too long · This child has the maximum number of medication courses |

**Dose edit** (`doses/[dose]/+page.svelte`, `medication-form.ts`)

| Element | Text |
|---|---|
| h1 / sub | Edit dose · Dose of {course name} |
| Fields | Given at · Amount · Note |
| Delete | Delete this dose · This removes the dose for everyone with access to {child}. · Yes, delete this dose |
| Errors | Enter a time that is not in the future and not before the birth date · Enter an amount of up to 80 characters · The note is too long |

**Notifications** (`account/notifications/+page.svelte`, `NotificationDevice.svelte`, `notification-form.ts`, `subscribe/+server.ts`, `constants.ts`)

| Element | Text |
|---|---|
| h1 / sections | Notifications · This device · Your other devices · Web Push · Your devices (list name) · ntfy |
| Status | Saved |
| Enable | Turn on notifications on this device · Turning on notifications needs JavaScript, and the browser asks for your permission. |
| Unsupported | This browser cannot receive notifications. |
| iOS | On iPhone and iPad, add this app to your Home Screen first, then open it from there to turn on notifications. |
| Off | Web Push is not configured on this server. |
| Device form | Device name · Reminders on this device for · Save · Turn off on this device |
| Other device | {label} · Added {YYYY-MM-DD HH:MM} UTC · Remove (sr: " {label}") |
| Default device name | This device |
| Problems | Notifications are blocked for this site. Allow them in the browser's settings, then try again. · Notifications were not turned on. You can try again. · Notifications could not be turned on. Try again. · Turn notifications off and on again on this device. · This device could not be checked with the server. Reload the page to try again. |
| Server answers | You have the most devices allowed. Remove one, then try again. · The server refused this browser's notification address. Try again later. · The app cannot reach its server, try again soon · The app cannot reach its server right now · Something went wrong, try again |
| Device errors | Enter a name of 1 to 40 characters · Choose from your children |
| ntfy | ntfy keeps each message for up to 72 hours on its server, so your child's name is stored there in plain text for that long. · Turn on ntfy · Server · Topic · Subscribe to this topic on this server in the ntfy app. · Reminders through ntfy for · Save · New topic · Turn off ntfy · A new topic replaces this one: subscribe to the new one in the ntfy app. · ntfy is not configured on this server. |

**Kid settings** (`KidForm.svelte`, `kid-form.ts`)

| Element | Text |
|---|---|
| Label | Show a missed dose as overdue for (minutes) |
| Hint | From 5 to 1440 (24 hours). A missed dose leaves the home list after this time, or when the next dose of that course is due, whichever comes first. |
| Error | Enter minutes from 5 to 1440 |

**Home** (`due.ts`, `DueList.svelte`, `constants.ts`, `KidLinks.svelte`)

| Element | Text |
|---|---|
| Section | Due |
| Medication line | {course name} · {HH:MM \| YYYY-MM-DD HH:MM} (+ " · overdue") |
| More lines | {n} more on the Medication page · {n} more on the Vaccines page |
| Tiles | Sickness · Medication |
| Kid link | Medication (sr: " for {child}") |

**Account**: Notifications. **OS notification**: Medication due for {child's name} · Reminder due.

---

## 10. Appendix B — accessible names the e2e tests rely on

Headings: "Log a symptom", "Edit symptom", "Medication for {child}", "Add a course", "Edit course", "Edit
dose", "Notifications", "Due", "Doses of {course}". Links: "Add a course", "Medication", "Notifications",
"{course name}" (card heading link), "{n} more on the Medication page", "{n} more on the Vaccines page", "Show
all doses". Buttons: "Log dose now" (full name "Log dose now of {name}"), "Save", "Yes, delete this course",
"Yes, delete this dose", "Yes, delete this entry", "New topic", "Turn on ntfy", "Turn off ntfy", "Turn off on
this device", "Remove {label}", "Turn on notifications on this device". Summaries: "Delete this course",
"Delete this dose", "Delete this entry", "Ended and inactive courses ({n})". Fields: "Time", "Symptom",
"Severity" (radios "Mild", "Moderate", "Severe", "Not set"), "Other symptom", "Temperature", "Temperature
unit", "Method", "Name", "Dose", "Start", "Last day (optional)", "Active", "Hours between doses", "Time 1"…"Time
12", "Given at", "Amount", "Amount (optional)", "Device name", "Show a missed dose as overdue for (minutes)";
the children checkboxes are named by the child's name. Regions: "Your other devices", "Your devices",
"Entries". Exact texts asserted: the iOS, ntfy, off and overdue-hint sentences of appendix A; "Turning on
notifications needs JavaScript"; "This also deletes the {0|1|2|12} dose(s) logged for this course".

## 11. Appendix C — icons and illustrations NOT needed

The app is text-first. Not needed: a thermometer or temperature-level icon, symptom icons or body diagrams,
severity colour chips, pill/bottle/syringe illustrations, a dose counter graphic, a clock-face schedule
picker, empty-state illustrations for "No courses yet" / "No doses logged yet", a bell with a badge count,
device-type icons (phone/laptop), the ntfy logo or any third-party brand mark, platform badges (Apple, Google).
If you add a glyph (e.g. lucide `bell` beside "Notifications"), it must be neutral and never encode a value or a
state on its own.

## 12. Appendix D — reference

Fixtures used by the tests: children Mia, Leo, Ivy; parents Ana, Cleo, Dan; course names such as "Calpol" and
"Amoxil" (fixtures only, not product copy); admin `admin`. The UI unit and e2e suites were **not run** while
writing this file (the sandbox's Node is 22; the project needs 24); every string was read from source.

## 13. Verified against code at `921c1a1`

| Screen / rule | Files read |
|---|---|
| Sickness form | `ui/src/lib/components/entry/{SymptomFields,TemperatureField,RadioField,TrackerFields,TrackerForm,Field}.svelte`; `ui/src/routes/(app)/kids/[kid]/log/[tracker=tracker]/+page.svelte`, `…/[entry]/+page.svelte`; `ui/src/lib/server/entry-form.ts` (`readSymptom`, `degrees`); `ui/src/lib/server/entry-failure.ts`; `ui/src/lib/entry-edit.ts`; `ui/src/lib/constants.ts` |
| Timeline | `ui/src/lib/timeline.ts`; `ui/src/routes/(app)/kids/[kid]/timeline/+page.svelte`, `+page.server.ts`; `ui/src/lib/today.ts` |
| Medication page | `ui/src/routes/(app)/kids/[kid]/medication/+page.svelte`, `+page.server.ts`; `ui/src/lib/components/medication/{CourseCard,DoseForm}.svelte`; `ui/src/lib/medication.ts`; `ui/src/lib/server/medication-form.ts` |
| Course pages | `…/medication/new/+page.svelte`, `+page.server.ts`; `…/medication/[medication]/+page.svelte`, `+page.server.ts`; `ui/src/lib/components/medication/{CourseFields,ScheduleFields}.svelte` |
| Dose edit | `…/medication/doses/[dose]/+page.svelte`, `+page.server.ts` |
| Notifications | `ui/src/routes/(app)/account/notifications/+page.svelte`, `+page.server.ts`, `subscribe/+server.ts`; `ui/src/lib/components/NotificationDevice.svelte`; `ui/src/lib/server/{notification-form,push-renew,constants,outcome}.ts`; `ui/src/lib/push.ts`; `ui/src/lib/format.ts` (`formatInstant`) |
| Service worker / OS notification | `ui/src/service-worker/index.ts`; `ui/e2e/service-worker.spec.ts`; `api/internal/reminders/worker.go` |
| Kid settings | `ui/src/lib/components/KidForm.svelte`; `ui/src/lib/server/kid-form.ts`; `ui/src/routes/(app)/kids/[kid]/settings/+page.svelte` |
| Home Due, tiles, links | `ui/src/lib/{due,quick-log,vaccines,paths}.ts`; `ui/src/lib/quick-log.test.ts`; `ui/src/lib/components/{DueList,KidLinks,QuickLog}.svelte`; `ui/src/routes/(app)/kids/[kid]/+page.svelte`, `+page.server.ts` |
| Account link, health | `ui/src/routes/(app)/account/+page.svelte`; `ui/src/routes/healthz/reminders/+server.ts`; `api/internal/httpapi/health.go` |
| API rules | `api/internal/constants/constants.go`; `api/internal/httpapi/{medications,doses,symptoms,home}.go`; `api/internal/store/{doses,reminders,medications,kids,access,parents}.go` (`markDone`, `DueEntries`, `ListDoses`, `UpdateKid`, `RemoveAccess`, `SetParentDisabled`) |
| Business rules and owner decisions | `docs/superpowers/plans/2026-11-22-sickness-medication-reminders.md` (Global Constraints, Open values 1–32); `docs/runbooks/reminders.md`; `docs/superpowers/HANDOFF.md` (plan 6 decisions, gaps); `README.md` |
| Component sizes | `ui/src/lib/components/field.ts` (`SELECT_CLASS` h-8); `ui/src/lib/components/ui/{button,input}` |

**Unverified:** how the OS and the ntfy app render the notification; iOS delivery, Apple's acceptance of the
server's key subject, and the Home Screen app's sign-in; real delivery through the browser push services; the
exact behaviour when a device form opts in a child the parent has just lost (code-read as the "Not found" page,
not seen in a test).

## 14. What to hand back

A design spec, not code. Per screen at **390 px and ≥ 1024 px, light and dark**:

1. The **symptom form** (new and edit; fever, other, plain), its timeline rows and the "Symptom" filter.
2. The **Medication page** (empty, active cards, ended/inactive, filtered doses), **course create/edit/delete**
   (each schedule kind), **dose edit**.
3. The **Notifications page** in each state: Web Push on (enable, enabled device, other devices, problems, iOS,
   unsupported, no JS), Web Push off (with and without old devices), ntfy off for the parent, ntfy on.
4. The **home Due list** with medication and vaccine lines and "more" lines; the new tiles; the new links.
5. The **kid settings field**.
6. **States** for each: empty, loaded, field error, server error, module off, no channel, caregiver view,
   no-JS baseline, 390 px with long unbroken text.
7. Annotated layout, the **shadcn-svelte component mapping** (and components to add), **all copy** (changing
   nothing in the fixed notification text), lucide icon picks (section 11 lists what is not needed), token
   changes as new `oklch` values, the accessible names you change (appendix B), an **API-change list** tagged
   cheap / costly, and your treatment of each open item in section 7 labelled as an assumption.
