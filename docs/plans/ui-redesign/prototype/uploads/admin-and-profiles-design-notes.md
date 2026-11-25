# Admin, My profile, Child profile — design notes (prototype)

Pages: `Admin.dc.html`, `My Profile.dc.html`, `Child Profile.dc.html`. Stores: `hbt-acct.js` (parents, access, sessions) next to `hbt-data.js` (children). Updated after the owner's answers (`answers-to-design-agent-questions.md`).

## Facts applied (owner)
- Admin is a **separate principal** (credentials from the deployment), not a flag. Own sign-in surface and shell: no tabs, no child pages, no link from the parent app. One person who needs both uses two identities. The admin can't change its own username or password in the app.
- Admin screen: create parent (permanent username 3–32 of `a-z 0-9 . _ -`, display name), one-time invite link (72 h), one-time reset link (1 h; also signs the parent out everywhere), disable/enable. Links are shown once, no email. It never shows a child, a child name or a count.
- Last-guardian rule enforced by the API; the UI shows its refusal. Disabling the only active guardian of any child: "This parent is the sole guardian of at least one child." (no name, no count). On Child profile the last guardian's own row says they can't leave or change role.
- Caregivers have no child profile (not found / not linked). Delete is permanent and immediate, confirmed by typing the name. The time zone is editable by guardians.
- No email in the product: My profile shows the permanent username and an editable display name; password change; devices.

## Assumptions (labelled)
1. Caregiver read-only child profile: drawn as a variant ("Caregiver view: Read-only" in the panel), needs an owner decision. Default is no page.
2. "Export a PDF first" is a labelled placeholder until PDF export exists. Grace period before delete is an open owner question.
3. Time-zone change: one-line notice under the field, no extra confirmation. The prototype keeps displaying times in the original zone.
4. A guardian adds a person by an existing parent's username (no email invites). Role is per child.
5. Enabling a parent who never set a password returns them to Invited. Refusal sentences other than the admin one are mine.
6. Caregivers get My profile. Password change can sign out other devices (ticked by default).

## shadcn-svelte mapping
Card, Badge, Button, Alert, Sheet (details, add person, create parent, one-time link), AlertDialog (disable, remove access, both delete steps, sign out all), Input, Label, Checkbox, Toggle group (admin filter), Sonner (undo for role change, remove, add).

## Accessible names (new)
"New reset link for {name}", "New invite link for {name}", "Disable {name}", "Enable {name}", "Sign out {device}, {place}", "Make {name} a caregiver/guardian", "Remove {name}'s access to {child}", field "One-time link", button "Copy link", delete step 2 field "Type the name: {child}", button "Delete all data for {child}".

## API changes
Cheap: parents list/create, link issue (hashed single-use tokens), disable/enable, profile patch, password change (+ revoke other sessions), session list/revoke, child details patch (incl. time zone), access grant by username, role patch/delete.
Costly: guardian-count invariant in one transaction; cascade hard delete of a child; time-zone change affecting day grouping everywhere; PDF export; grace period (pending-deletion state, purge job, backups).
