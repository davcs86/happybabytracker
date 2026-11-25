# Route map: prototype page → SvelteKit route

Prototype route names inside the side-panel notes (e.g. `/children/:id/edit`, `/profile/name`) are illustrative; **use the legacy routes below** unless "NEW".

| Prototype page | Spec | Legacy route(s) | Status |
|---|---|---|---|
| Sign In | [pages/sign-in.md](pages/sign-in.md) | `/login` | exists |
| Set Password | [pages/set-password.md](pages/set-password.md) | `/invite/[token]`<br>`/reset/[token]` | exists |
| Error | [pages/error.md](pages/error.md) | `(any failing route)` | exists |
| Kid Home | [pages/kid-home.md](pages/kid-home.md) | `/kids/[kid]`<br>`/ (no children yet: Welcome state)`<br>`/kids/[kid]/log/[tracker] (tile forms)` | exists |
| Timeline | [pages/timeline.md](pages/timeline.md) | `/kids/[kid]/timeline`<br>`/kids/[kid]/log/[tracker]/[entry] (row edit)` | exists |
| Growth | [pages/growth.md](pages/growth.md) | `/kids/[kid]/growth`<br>`/kids/[kid]/growth/new`<br>`/kids/[kid]/growth/[entry]` | exists |
| Vaccines | [pages/vaccines.md](pages/vaccines.md) | `/kids/[kid]/vaccines`<br>`/kids/[kid]/vaccines/new`<br>`/kids/[kid]/vaccines/[administration]` | exists |
| Milestones | [pages/milestones.md](pages/milestones.md) | `/kids/[kid]/milestones` | exists |
| Milestone Form | [pages/milestone-form.md](pages/milestone-form.md) | `/kids/[kid]/milestones/new[?item=]`<br>`/kids/[kid]/milestones/[entry]` | exists |
| Medication | [pages/medication.md](pages/medication.md) | `/kids/[kid]/medication` | exists |
| Course | [pages/course.md](pages/course.md) | `/kids/[kid]/medication/new`<br>`/kids/[kid]/medication/[medication]` | exists |
| Dose | [pages/dose.md](pages/dose.md) | `/kids/[kid]/medication/doses/[dose]` | exists |
| Symptom | [pages/symptom.md](pages/symptom.md) | `/kids/[kid]/log/symptom`<br>`/kids/[kid]/log/symptom/[entry]` | exists (tracker route; confirm the symptom tracker id) |
| Export | [pages/export.md](pages/export.md) | `/kids/[kid]/export` | exists |
| PDF Export | [pages/pdf-export.md](pages/pdf-export.md) | `/kids/[kid]/export/pdf` | exists (server-generated; design is the print layout) |
| Child Profile | [pages/child-profile.md](pages/child-profile.md) | `/kids/[kid]/settings`<br>`/kids/[kid]/access`<br>`/kids/[kid]/modules` | exists. The prototype merges details, settings and access into one profile page with sheets. The sheets' no-JS pages are designed separately (Child Settings, Access). **Modules has no designed page** (sheet only): confirm its no-JS route with the owner before building |
| Add a Child | [pages/add-a-child.md](pages/add-a-child.md) | `/kids/new` | exists (restyle). Designed 2026-11-24; same field set and validation as the sheet |
| Child Settings | [pages/child-settings.md](pages/child-settings.md) | `/kids/[kid]/settings` | exists (restyle). Designed 2026-11-24; caregivers see the not-found card |
| Access | [pages/access.md](pages/access.md) | `/kids/[kid]/access` | exists (restyle). Designed 2026-11-24; the sole guardian cannot be removed or demoted |
| My Profile | [pages/my-profile.md](pages/my-profile.md) | `/account` | exists |
| Notifications | [pages/notifications.md](pages/notifications.md) | `/account/notifications`<br>`/account/notifications/subscribe (POST)` | exists |
| Choose Child | [pages/choose-child.md](pages/choose-child.md) | `/kids (the child switcher's no-JS page)` | exists as the switcher's no-JS page (`/kids`); if the legacy repo has no `/kids` list route, add it |
| Admin | [pages/admin.md](pages/admin.md) | `/admin/parents` | exists |

Legacy routes with **no prototype page** (restyle with the same components; not pixel-specified, see assumption A7 in `IMPLEMENTATION.md`): the Modules page (`/kids/[kid]/modules`, sheet only in the prototype), `/healthz*`, `/logout` (POST only).
