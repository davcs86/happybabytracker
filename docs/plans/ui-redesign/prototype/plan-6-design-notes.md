# Plan 6 design notes: sickness, medication, reminders, notifications

Hand-back for `uploads/plan-6-design-handoff.md`. Everything here is a design assumption unless the handoff says otherwise. Verified in this prototype by logic tests only; nothing has been rendered in a browser, and axe has not been run.

## 1. Where each piece is

| Handoff | Prototype page | Notes |
|---|---|---|
| 4.1 Symptom form (new, edit, delete) | `Symptom.dc.html` | Panel: New/Edit, °C/°F preference. Fever and Other fields appear when chosen (CSS `:has()` reveal in the real app) |
| 4.2 Timeline rows, "Symptom" filter | `Timeline.dc.html` | Default filter follows the sickness module (off → unticked until touched). Row opens the symptom edit page |
| 4.3 Medication page | `Medication.dc.html` | Child switch (Ivy shows the empty state), simulated "Course gone" / "Dose cap", larger one-hand variant |
| 4.4 Course create / edit / delete | `Course.dc.html` | All three schedule kinds; 12-input vs 3 + "More times" variant; 0-dose wording variant |
| 4.5 Dose edit | `Dose.dc.html` | |
| 4.6 Notifications | `Notifications.dc.html` | Panel: channels (none / Web Push / + ntfy), browser (OK / iPhone Safari / no push / JS off), prompt answer, "Ana enables here", key rotated |
| 4.7 Overdue field | `Child Profile.dc.html` (Settings row, sheet) and `Kid Home.dc.html` (Settings sheet) | Guardian only; hidden for caregivers |
| 4.8 Due list, tiles, links | `Kid Home.dc.html` | Medication lines first, shared cap of 3, two "more" lines; More list has "Medication" for every role |
| 4.9 Account link | `My Profile.dc.html` | "Notifications" under the h1 |
| Store | `hbt-med.js` (new), `hbt-data.js`, `hbt-acct.js` | Occurrence, "done", due, validation and cleanup rules live in the store, standing in for the server |

## 2. shadcn-svelte mapping

| UI | Part |
|---|---|
| Cards on Medication, Notifications | `Card` |
| Fields | `Field` + `Label` + `Input`; native `<select>` (as built, `SELECT_CLASS`) |
| Severity, Schedule | `RadioGroup` in a `fieldset` with a `legend` |
| Active, children choices | `Checkbox` (24 px here; see §5) |
| Delete this entry / course / dose, Ended and inactive courses | `<details>` (`Collapsible` styling) |
| Alerts (`role="alert"`) and "Saved" (`role="status"`) | `Alert` |
| Buttons, one-tap dose | `Button` |

Components to add: none. A `Separator` is optional between the device form and "Turn off on this device".

## 3. Icons

Appendix C is followed: no thermometer, no pill, no bell, no device icons. The two new quick-log tiles and the symptom timeline chip use one neutral lucide glyph, `notebook-pen`, which encodes no value or state. The handoff describes the tiles as label only; if you prefer that, drop the glyph and the tile keeps its label.

## 4. Accessible names

No name from appendix B is renamed. The Child Profile settings row for the overdue window uses the same text as its field label.

One structural change to flag: on the Timeline a symptom row is an `<a>` to the edit page, where every other row is a `<button>` that opens a sheet. Its accessible name is unchanged ("{title} at {time}. Edit entry.").

## 5. Weaknesses (§6) and what the design does

| # | Treatment |
|---|---|
| 1 Small targets | Everything drawn here is 44 px or more, including inputs (48), the one-tap button, "Remove", "New topic", "Turn off ntfy" and the 24 px checkboxes inside 44 px rows. Cost: UI only (`size` default instead of `sm`) |
| 2 No feedback after a dose | Not changed; see §7.2-1 variant |
| 3 / 17 "Next due" vs "overdue" | See §7.2-4 |
| 4 Twelve time inputs | See §7.2-2 |
| 5 Enabling opts in no child | See §7.2-3 |
| 6, 7, 8, 9 | Drawn as built (open items 3 and 2). Suggestions below |
| 10 "Added … UTC" | Kept, labelled "UTC". Alternative: show relative age ("Added 3 days ago"). UI only |
| 11 Casing | Medication lines read "· overdue", vaccine badge reads "Overdue". Kept. A one-line change on either side aligns them |
| 12 Only 50 doses | Not designed. Needs an API page parameter (costly) |
| 13 Hydration | No autofocus into a prefilled field anywhere |
| 14 Medication link with module off | See §6 item 1 |
| 15 "0 doses" | See §6 item 7 |
| 16 No time-zone warning | See §6 item 6 |

## 6. Open items (§7.1): all labelled assumptions

| # | Assumption in this prototype |
|---|---|
| 1 | Link under the child's name stays shown. The Kid Home panel has a **needs owner decision** variant that hides it when the module is off |
| 2 | "New topic" drawn as built (no confirmation). UI-only option: a `<details>` "Replace the topic?" with a second button |
| 3 | Drawn as built. Proposal, not drawn: a muted line under Sign out, "This device keeps getting reminders until you turn them off on the Notifications page" (UI only) |
| 4 | Drawn as built |
| 5 | Drawn as built (no ntfy section while off) |
| 6 | Drawn as built. Optional variant, not drawn: one muted line under the Time zone field, "Medication times at fixed hours move with the time zone." (UI only) |
| 7 | As built, plus a panel variant that omits the line at 0 doses (UI only) |

## 7. Design questions (§7.2)

| # | Answer | API change |
|---|---|---|
| 1 | A larger one-hand "Log dose now" (64 px, full width) is in the panel. Proposal for confirmation: after the redirect, a `role="status"` line "Logged {course} at {HH:MM}" at the top of the page, using the dose the server just created. | Cheap if the redirect carries a query flag; no API change needed |
| 2 | Show three time inputs and put Time 4 to 12 in a `<details>` "More times" (opens itself when one of them has a value or error). Works without JS | Cheap |
| 3 | Keep opt-in empty (as built). Add a one-line instruction above the child checkboxes right after enabling: "Choose which children this device gets reminders for, then press Save. A device with no child receives nothing." Shown in the panel as "One-line instruction". Auto-opting in the current child reverses an owner decision | UI only: cheap. Auto opt-in: costly in rules, cheap in code |
| 4 | Keep the two words, in the same place, with different jobs: the Due line says what was missed, the card says what is next. Proposed card wording when a missed occurrence exists: "Next due 20:00" stays as built, and the card gets no overdue text; the home line is the only place "overdue" appears. No colour or icon difference between due and overdue | Cheap, UI only |
| 5 | Temperature is plain text: number, unit, method, as one title: "Fever 38.4 °C (ear)". No scale, no colour, no icon. Same in the form and the timeline | None |
| 6 | Keep the select. 12 options is well within a native select, works with the CSS reveal, and a radio grid adds 12 targets and a scrolling form at 390 px | None |
| 7 | iOS sentence: where the enable button would be, replacing it. ntfy privacy sentence: first line of the ntfy section, before the "Turn on ntfy" button, so it is read before the choice. Both as built | None |

## 8. API-change list

| Change | Cheap / costly | Why |
|---|---|---|
| Status line after a dose (7.2-1) | Cheap | Query flag on the 303 |
| "More times" disclosure | Cheap | UI |
| Instruction after enabling Web Push | Cheap | UI |
| Auto opt-in the current child | Costly in rules | Reverses an owner decision |
| Older doses beyond 50 | Costly | New paging parameter |
| Warning on time-zone change | Cheap | UI-only line |
| Hide "Medication" link with module off | Cheap | UI; needs owner decision |
| ntfy "New topic" confirmation | Cheap | UI only (`<details>` second step) |
| Sign-out hint for shared tablets | Cheap | UI text |
| Make overdue minutes editable by caregivers or on create | Costly | Owner reversal; not drawn |

## 9. States covered in the prototype

| State | Where |
|---|---|
| Empty / loaded | Medication (child switch), Timeline |
| Field error | Symptom, Course, Dose, Medication amount, overdue field, Notifications device name |
| Server error / offline | "Next save: Offline / Error" panel buttons, shown as the page-level `role="alert"` line; typed values stay |
| Module off | Kid Home Modules switches: tile and Due lines disappear; Timeline "Symptom" filter unticks; pages still open by URL |
| No channel configured | Notifications: channels "None" |
| Caregiver view | Role toggle: Settings overdue field hidden; every other new page unchanged |
| No JS | Notifications "JS off": only the enable button disappears. All other pages are native forms in the real app |
| 390 px / long text | Names, doses and amounts use `overflow-wrap:anywhere` |

## 10. Prototype limits (honest list)

- Not rendered in a browser; no axe run; dark mode uses the existing tokens, not re-measured.
- `select value` is used for native selects; if the Design runtime does not honour it, the select shows its first option while the stored value is still right.
- Sample medication data is re-seeded daily; the overdue line depends on the time of day you open it.
- Time-zone display stays Los Angeles; changing the child's zone does not move times in the prototype.
- "Done" within 30 minutes before an occurrence, the one-overdue-line rule and the cap of 3 are implemented in the store and unit-tested, not in a server.
- Timeline running cards have no edit link; Vaccines and Growth rows are shared across children; the static Access sheet in Kid Home is dead.
