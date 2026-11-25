# Not built yet

The redesign ([design export](../plans/ui-redesign/README.md)) shows a few things the app cannot do yet, because the data or the route they need does not exist. The screens in this wiki do not show them. This is the list, as it stood when the redesign was merged (pull request #16).

A separate piece of work is deciding, for each item, whether it needs a new API endpoint, can be built from data the app already loads, or should be dropped. Until that is decided, treat every item below as absent.

## Home

- The **Latest** list (the three newest entries) and its "See the Timeline" link.
- "**1 h 34 min ago**" subtitles on the tracker tiles, and the "Last: ..." lines in the Today cards.
- The running timer's elapsed clock and the "Started on another phone" line.
- The Left and Right side tiles for nursing.

## Timeline

- The "Last 7 days" week strip, "Since last", running-timer cards, the day filter ("Show all days") and the age line under the title.

## Growth

- The segmented units control. Units are chosen with plain selects, as before.
- Selecting a point on the chart to highlight its row.
- The chart's tick labels are 11 units in a fixed drawing, so they read at about 9 px on a phone.

## Medical visits

- **Looking up a doctor** in the US provider registry (NPI) while adding a visit. It is a separate, later plan (the visits design, section 10), to sit behind a server switch; today **Doctor or clinic** is typed by hand ([Medical visits](Medical-visits.md)).

## Access and children

- Undo toasts after a role change, removing access, adding a person or marking a milestone.
- The remove-access confirmation sheet, an "Account disabled" badge, the counts in the delete-a-child list and "Export a PDF first".

## My profile

- The edit-name and change-password sheets (the forms are on the page), "last changed N days ago", device names and places, "sign out all other devices".

## Admin

- "Last active" and the link-issued line. The admin has no notification bell, because an admin session cannot open any other page.

## Other gaps the screenshots show

- The **503 page**, and the **404 page a parent gets on `/admin`**, are an unstyled fallback page (`src/error.html`), not the designed error page: both errors are raised in the server hook, before the designed page can render ([Error pages](Error-pages.md)).
- With **Milestones switched off** for a child, the Milestones page still shows the lists under a note; the design says to hide the lists and the add controls ([Milestones](Milestones.md)). Medication does hide them.
- A **64-character child name** widens the page on a phone, because the header's child switcher does not truncate it. The design says it truncates ([Children and access](Children-and-access.md)).
- On a phone without JavaScript the **More** tab has only a plain list of links below the page; the design leaves that case undefined.
- On [Access](Children-and-access.md), after a person is added with JavaScript on, the Role choice resets to none instead of Caregiver (`kid-admin/14-access-caregiver-added`). The first load of the form does preselect Caregiver (`kid-admin/11-access-sole-guardian`).
- The sign-in page does not link to an invite page: an invite is only the one-time link a guardian or the admin sends.

Back to the [wiki home](Home.md).
