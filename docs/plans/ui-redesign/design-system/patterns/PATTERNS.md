# Patterns: flows and states (product-neutral)

A pattern is a combination of components with a fixed behaviour. Each pattern lists its states, the wording rules, the no-JS route and the checks. Components are in `../components/COMPONENTS.md`; wording in `../DESIGN.md` section 9.

## P1. Record form (add/edit) in a Drawer, page as the fallback
| Step | With JS | Without JS |
|---|---|---|
| Open | click `a[data-drawer]` → Sheet with the same page in embed mode | the link navigates to the page |
| Fill | fields validate on submit (never while typing); first refused field gets focus | same, server-rendered messages |
| Refused | field messages + a form summary "Check the fields with a message." | same, on the re-rendered page |
| Failed (offline/server) | Sheet stays open, values kept, Alert "Couldn't save. {reason}", same button retries | page re-renders with the Alert and the values |
| Saved | Sheet closes, list refreshes in place, toast that names the result in the action's own words ("Measurement added", "Entry deleted"; Undo where reversible) | 303 redirect to the list with the toast text carried as a flash message |
| Cancel | closes, no toast, focus returns to the trigger | link back to the list |
Edge states: the record was deleted by someone else (toast, row removed), the record changed (the conflict sentences in DESIGN 8.2), the date is in the future or before birth (field message).

## P2. Destructive action
Row action is a text button ("Remove", "Delete this dose"). It reveals an inline confirm card: consequence sentence first, then a destructive button (same word as the trigger) and Cancel. Irreversible and high-impact (delete a child) adds a second step: list consequences, then type the name and tick the box. Reversible changes use Undo in a toast instead of a confirm. After confirm: toast "Removed" (+ Undo if reversible). Last-guardian and access rules are enforced by the server and stated in the UI where they apply.

## P3. Toast outcomes
One toast at a time, bottom, above the tab bar; a new toast replaces the old one. Success 6 s; failure with "Try again" 9 s (`role="alert"`). Verbs are the same as the button that caused it. A toast never carries the only copy of an error that needs reading in a sheet (that is the Alert).

## P4. Filters and selection
ToggleGroup chips filter a list (Timeline types, Export modules, Admin status). Pressed chips are the selection; with nothing pressed, the list is unfiltered. Without JS each chip is a link or a checkbox in a GET form, so the filter lives in the URL. The list below the chips always says what it shows; an empty result is an invitation (one sentence + the action that fills it), never a blank area.

## P5. Lists and time
- Time-ordered lists (Timeline, doses, history) are grouped by day; each row is a link to the edit route and opens the Edit Drawer with JS.
- "Older" is a link at the end of the list (a plain navigation without JS). The newest page is always first.
- Grouped records (vaccines by visit, milestones by age) use disclosure cards; the current group is open.
- Times display in the viewer's zone; the data layer stores UTC.

## P6. Page states (every page)
Normal, empty (an invitation), module off, not found (one page for unknown/no access/wrong role), can't reach the server (503, session kept, "Try again"), loading (the shell and navigation render first; a page never blocks them). Pages outside the shell (sign in, set password, 404, 503) have their own copies of the states.

## P7. Authentication
Sign in: one username field, one password field, one message for any wrong credential, username kept, password cleared; rate limit message fixed; no show-password toggle, no forgot-password link. Set password (invite/reset): new + confirm, ≥ 12 characters, mismatch on the confirm field, expired/used/unknown link share one message with the validity period. Sign out is a POST that clears the local session even when the API is unreachable.

## P8. Theme
Change theme opens a RadioGroup sheet (Light, Dark, Auto); the choice applies immediately and persists (cookie + local store); Done closes. Auto follows the device live. Details: DESIGN 7.

## P9. Switch child
Header button → Sheet with one row per child; choosing one saves the choice and returns to the same module for that child (whitelisted return target, default Home). Many children: a "Find a child" field. No back links on module pages.

## P10. Access and roles
Roles: guardian (manage), caregiver (log), admin (separate surface). Controls that a role cannot use are not shown (not disabled), with a one-sentence explanation where the absence would confuse ("Ask a guardian to add a child."). The server enforces everything; the UI hides.

## P11. Reference data and charts
A chart always has its data table in a Collapsible, a one-line source with retrieval date, a neutral "not scored" reason for out-of-range values, and no judgement wording (denylist in the preset). Milestones, vaccines and growth never say a child is behind, late, normal or at risk.

## P12. Export (PDF)
Module picker (ToggleGroup chips: switched-on modules ticked, switched-off ones unticked in a separate disclosure "Switched off for this child"), a start and end date (default: the last 30 days), one primary "Download PDF". Refusals mirror the server: the start is before the birth date, the range is longer than 93 days, nothing is selected. A failure shows the Alert "The PDF cannot be made right now, try again in a moment" and keeps the selection. The PDF is A4, 12 mm margins, Noto Sans, greyscale, with no meaning carried by colour alone.

## P13. Notification settings
A page (not a Drawer) with one section per channel the server supports: Web Push (explains when it is not configured on the server) and ntfy (topic and server in monospace, "New topic" replaces the topic and says to subscribe again, a plain statement of how long ntfy keeps a message: 72 hours, including the child's name). When no channel is configured the page says reminders still appear on the Home Due list. There is no inbox and no unread dot.

## P14. Admin
A separate sign-in surface with no tabs, no child pages and no link from the parent app; it never shows children, their names or any count. List with a status filter (ToggleGroup chips, links without JS); "Create parent", "New link" and "Enable" are form POSTs (Drawer with JS). An invite or reset link is shown once in a selectable read-only field ("One-time link"), with a note to copy it now and hand it over by a trusted channel, and it states how long it lasts (reset 1 hour, invite 72 hours, single use). "Disable" is a destructive confirm (P2); a parent who is the only active guardian of any child cannot be disabled and the server says why.
