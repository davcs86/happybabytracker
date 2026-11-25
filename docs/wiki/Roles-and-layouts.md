# Roles, layouts and conventions

Facts that hold on every screen. The other pages assume them.

## Who can see what

| Role | What they can do | What they cannot |
|---|---|---|
| **Guardian** | Everything for a child: log, view, export, change settings and modules, add or remove people, delete the child. | Leave when they are the child's only guardian. |
| **Caregiver** | Log, view and export for a child. Leave a child they were added to. | Open guardian pages (Modules, Child profile, Edit details, Access). They get the same "Not found" page as for a child that does not exist. |
| **Admin** | Create parents, issue invite and reset links, disable and enable accounts, on `/admin/parents` only. | Everything else. The admin never sees a child, a child's name or a count of children. |

Three rules the screens show repeatedly:

- **A page you have no access to is a 404**, identical to a page that does not exist. The page never says "you do not have access", so it cannot reveal that a child exists ([Error pages](Error-pages.md)).
- **A child always keeps at least one active guardian.** The last guardian cannot leave, be demoted or be removed, and the admin cannot disable them ([Children and access](Children-and-access.md), [Admin](Admin.md)).
- **Controls a role cannot use are not shown**, rather than shown disabled. Where the absence would confuse, a sentence says so ("Ask a guardian to add a child.").

## Three layouts

The same pages are laid out three ways. The screenshots in this wiki show the first and the last.

| Layout | Width | Navigation | Content |
|---|---|---|---|
| Phone | under 768 px (captured at 390 px) | Tab bar with Home, Timeline, Growth and More; everything else is in the More sheet | One column |
| Side navigation | 768 to 1279 px | A 225 px column on the left with every entry, so there is no More | One column, at most 720 px (lists and detail pages) or 448 px (forms) |
| Wide | 1280 px and up (captured at 1280 px) | The same side navigation | A fixed 920 px block in two columns: the task on the left, supporting detail on the right. Forms put the main fields on the left and the optional ones on the right |

The header is the same everywhere: the child switcher on the left, the notification bell and your role on the right.

## Modules

Each child has trackers and tools that can be switched on or off (Bottles, Nursing, Pumping, Diapers, Sleep, Feeding, Milestones, Growth, Vaccination, Sickness, Medication). The defaults depend on the child's age: for a child past two years the baby trackers and Milestones start off. A guardian changes them on the Modules page ([Children and access](Children-and-access.md)).

- A switched-off module has no quick-log tile on Home and its entries are hidden from the Timeline unless you ask for them.
- Its entry in More and in the side navigation is hidden. Its page, opened by address, says it is switched off and that nothing was deleted.

## Conventions

- **Times** are shown in the child's time zone, and each page that shows times says which zone.
- **Ages** are counted from the birth date ("4 months 16 days"). For a child born before 37 weeks, a corrected age appears on the next line, with the weeks at birth, until the child's second birthday; after that only the age shows. A child still before their expected term is shown as "before expected term" rather than as a zero or negative corrected age.
- **Units** are a personal choice (kilograms or pounds, millilitres or fluid ounces, °C or °F) set on [My profile](My-profile.md). Entries are stored in one unit and shown in yours.
- **No judgement.** The app states facts and the next action. It never says a child is behind, late, ahead or typical.
- **Colour is never the only signal.** Every status has a word, and usually an icon or a border shape.
- **Themes.** The app has Light, Dark and Auto (follows the device). Every screenshot here is the light theme.

Back to the [wiki home](Home.md).
