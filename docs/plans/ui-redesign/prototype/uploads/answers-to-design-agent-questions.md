# Answers to the Design agent's six questions

Status of each answer: **Fact** = how the product is specified and built today (owner decisions already made, verified against the design spec and the code on 2026-11-23). **Open** = the owner has not decided; the document gives the current behaviour, the options and what each costs, and a proposed default that is marked as a proposal. Where an answer is Open, keep building with the proposed default and mark it as an assumption in your spec.

| # | Question | Answer in one line | Status |
|---|---|---|---|
| 1 | Is admin a separate role? | Yes: a separate principal, not an account flag. One person cannot be both. | Fact (owner) |
| 2 | Last-guardian rule | Correct, with the exact refusals below. | Fact (owner) |
| 3 | What may a caregiver see of the child profile? | Today: nothing. Whether to add a read-only view is not decided. | Open |
| 4 | Deleting a child: grace period, export? | Today: immediate hard delete, confirmed by typing the child's name; no grace period; the PDF export is planned but not built. | Open |
| 5 | Reset links | Built values are 1 hour (reset) and 72 hours (invite), not 24 hours. No "send": the admin hands over a one-time link. | Fact for the numbers (spec "call"); open if you want other numbers |
| 6 | Child time zone | Already editable by guardians (not display-only). Effects listed below. | Fact; the warning on change is Open |

---

## 1. Admin is a separate role (not a flag on an account)

- The admin is a **separate principal**, configured from the server's environment (a username and password held in the deployment's secrets). It is **not a row in the parents table**, so it **cannot hold a child grant**: there is no way for the same login to be a guardian or caregiver. A person who wants to administer and also to track their own child has **two identities**: the admin credentials and a normal parent account (the admin creates that account for them).
- The admin's lack of child access is structural, not a hidden permission: every route outside the admin routes and the session route refuses an admin session. The admin **can never see, list, count or name any child**.
- What the admin can do: create a parent (username 3 to 32 characters of `a-z 0-9 . _ -`, permanent; display name), issue the one-time **invite link** and **password-reset link**, disable and enable parents. What it **cannot** do: see any child, change its own username or password in the app (they come from the deployment's secrets; changing them signs the admin out), be deleted or demoted.
- Design consequence: two separate sign-in surfaces and shells. The admin shell is small (a parents list: create, invite link, reset link, disable/enable) and shows no child names anywhere, not even a count. Do not design an "admin" toggle inside a parent's account.

## 2. Last-guardian rule: correct

A child always has at least one **active** guardian. The API refuses, in one transaction (so the UI must show the API's refusal and must not try to pre-check it):

1. revoking the last guardian's access,
2. changing the last guardian to caregiver,
3. the last guardian leaving the child,
4. the admin disabling a parent who is the **only active guardian of any child**.

Wording for the admin's refusal (fixed by the spec): "this parent is the sole guardian of at least one child". It must carry **no child name and no count**. A disabled parent is not "active", so disabling counts as removing a guardian for this rule. Design implication: on the access page the last guardian's remove, leave and change-role controls can be shown disabled with a one-line explanation, but the API remains the authority.

## 3. Caregivers and the child profile: Open

- **Built and specified (owner):** a caregiver may create, edit and delete entries and may view and export, but has **no access to the child profile and settings** (spec 5.2 table: "Kid profile and settings: no"), cannot grant or revoke access, and cannot delete the child. The settings, access and modules pages are guardian-only (a caregiver gets the same not-found page as for a child they have no grant on, and the links are not shown). A caregiver sees the child's name and age in the home header and can **leave** the child.
- **Not decided:** whether a caregiver should see a **read-only** child profile (name, birth date, sex, gestation, time zone, growth source, vaccination schedule). The data is already returned to any person with a grant, so a read-only view is a UI-only change.
- **Options:** (a) keep nothing (current, the spec's literal reading); (b) a read-only summary page for caregivers, with no edit controls and no danger zone, useful because a caregiver otherwise cannot see which growth chart source or vaccination schedule is in use.
- **Proposed default for your spec (an assumption, not an owner decision):** (a), with (b) drawn as an optional variant clearly labelled "needs owner decision".

## 4. Deleting a child: Open

- **Built and specified (owner):** deletes are **hard deletes** with no history. A guardian deletes a child by typing the child's name to confirm. Everything about the child (entries, growth, vaccinations, access grants, and from plan 6 medication courses, doses and reminders) is removed **immediately** for everyone. Only guardians may delete; a caregiver can only leave.
- **Not decided:** a grace period (soft delete with a visible "will be deleted on <date>" and an undo) and an **export before delete**.
- **Costs:** a grace period needs a "pending deletion" state, a purge job, rules for access during the grace period (do other guardians see it? do reminders stop?) and it interacts with the encrypted backups; it is a large change. Export before delete is cheap once the **PDF export** (planned, not built yet; guardians and caregivers will be able to export a date range, choosing modules) exists: the confirmation can link to it.
- **Proposed default for your spec:** the confirmation states plainly that it is permanent and immediate, names what is removed, and, as a **labelled placeholder**, offers "Export a PDF first" that the build team enables when the export exists. Mark the grace period as an unresolved owner question.

## 5. Reset links: the built numbers differ from your defaults

- **Reset links last 1 hour; invite links last 72 hours** (spec 5.3, both marked "call", so the owner may change them; they are constants). Links are 32 random bytes, stored hashed, **single use**; using a reset link also deletes that parent's sessions. The password minimum is 12 characters.
- There is **no emailing or sending**: the app has no email. The admin sees the link once after creating or resetting and hands it over by whatever channel they trust. So there is no "one send per minute" in the product: the admin screen should say the link is shown once and must be copied now, and what its lifetime is.
- Rate limiting: there is a shared address-keyed limiter (one-minute window) on sign-in; I found no per-parent limit on issuing links (not verified beyond the admin handler).
- **Proposed default for your spec:** use 1 hour and 72 hours as the numbers shown in the UI copy. If the owner wants 24 hours, it is a one-constant change in both halves of the app.

## 6. Child time zone: already editable

- It is **not display-only**: the child's IANA time zone is chosen when the child is added (the form defaults it from the device) and a **guardian can change it** in the child settings (the shared child form has a Time zone field and the API validates and accepts the change).
- **What changes when it changes:** stored entry times are absolute instants (UTC) and **never change**. What changes is every *display*: the clock times shown on each entry, which day an entry belongs to (day headings, "today", daily summaries, a sleep that crosses midnight), and the child's age rolling over at midnight. **Calendar dates are plain dates and do not move**: the birth date, measurement dates and vaccination dates. From plan 6, fixed-time medication reminders are re-computed in the new zone (a course "08:00" follows the new zone).
- **Not decided:** whether to warn or confirm on change. **Proposed default for your spec:** a one-line notice under the field in settings ("Times are shown in this time zone. Changing it moves how past entries are grouped by day; it does not change when they happened.") and no extra confirmation step. Mark as an assumption.
