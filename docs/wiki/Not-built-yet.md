# Not built yet

The redesign ([design export](../plans/ui-redesign/README.md)) shows a few things the app does not do. The screens in this wiki do not show them. Most of what pull request #16 left out was built afterwards (pull request #26): the Home Latest list, "x ago" and "Last:" lines, the running clock and side tiles; the Timeline strip, Since last, running cards, day filter and age line; Undo toasts; the Account password date, device names and "Sign out all other devices"; the Access "Account disabled" badge, the delete counts and "Export a PDF first"; Admin last-active and link lines; and the Growth units switch, selected row and readable chart labels. What is still absent is listed here, with the reason.

## Left out on purpose

- **Places on signed-in devices** (for example "Lisbon"). The app stores no IP address, and looking a place up would send one to another service.
- **"Started on Cleo's phone"** on a running timer. The API does not say who started an entry or on which device.
- **A notification bell for the admin.** An admin session cannot open any other page, and notifications belong to a parent.
- **A "Sign out my other devices" tick box** on the password form. Changing the password always signs the other devices out, and the form says so.
- **The remove-access confirmation sheet.** The design system says reversible changes use Undo, not a confirm (DESIGN 4.5, PATTERNS P2), and the Child profile page spec draws a confirm. Removing access has the Undo. The owner decides which one stands.
- **The edit-name and change-password sheets.** My profile stays a page (AGENTS.md rule 1), so both forms are on the page.
- **Tapping a point on the Growth chart.** The chart marks the selected measurement; choosing one is done from its row, because a link inside the chart image cannot be reached by keyboard or screen reader.
- **Undo for deleting a growth measurement, a dose, a vaccination or a medical visit.** Undo covers the seven daily trackers and milestones only.
- **A tick-by-tick count of sleeps** in the Timeline's week strip. The daily summary has the minutes slept, not the number of sleeps, so a day with sleep shows one sleep bar.

## Other gaps the screenshots show

- The **503 page**, and the **404 page a parent gets on `/admin`**, are an unstyled fallback page (`src/error.html`), not the designed error page: both errors are raised in the server hook, before the designed page can render ([Error pages](Error-pages.md)).
- With **Milestones switched off** for a child, the Milestones page still shows the lists under a note; the design says to hide the lists and the add controls ([Milestones](Milestones.md)). Medication does hide them.
- A **64-character child name** widens the page on a phone, because the header's child switcher does not truncate it. The design says it truncates ([Children and access](Children-and-access.md)).
- On a phone without JavaScript the **More** tab has only a plain list of links below the page; the design leaves that case undefined.
- On [Access](Children-and-access.md), after a person is added with JavaScript on, the Role choice resets to none instead of Caregiver (`kid-admin/14-access-caregiver-added`). The first load of the form does preselect Caregiver (`kid-admin/11-access-sole-guardian`).
- The sign-in page does not link to an invite page: an invite is only the one-time link a guardian or the admin sends.

Back to the [wiki home](Home.md).
