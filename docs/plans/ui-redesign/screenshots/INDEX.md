# Screenshot index

Baselines are PNGs under `screenshots/<page>/<variant>/NN-<state>.png`. Variants: `phone-light`, `phone-dark` (390 px wide frame), `desktop-light`, `desktop-dark` (980 px wide frame, side nav, one column), `wide-light`, `wide-dark` (1280 px wide frame, side nav, two-column block). `NN` is the capture order within one variant (numbering can differ between variants when a state is a duplicate in one of them, so always follow the links below). A state marked **dup** produced a picture identical to an earlier state in that variant (the control changed nothing visible), so no extra file is stored for it; the earlier file is the baseline.
1077 images, 915 duplicate state records.


Images were captured with Geist loaded, animations off, the prototype frame chrome (corner radius, outer shadow, simulation panel) removed.

## access

Route in the app: `/kids/{kid}/access`

| State | Kind | phone-light | phone-dark | desktop-light | desktop-dark | wide-light | wide-dark |
|---|---|---|---|---|---|---|---|
| default-frame | frame | [01](screenshots/access/phone-light/01-default-frame.png) | [01](screenshots/access/phone-dark/01-default-frame.png) | [01](screenshots/access/desktop-light/01-default-frame.png) | [01](screenshots/access/desktop-dark/01-default-frame.png) | [01](screenshots/access/wide-light/01-default-frame.png) | [01](screenshots/access/wide-dark/01-default-frame.png) |
| default-full-height | full | [02](screenshots/access/phone-light/02-default-full-height.png) | [02](screenshots/access/phone-dark/02-default-full-height.png) | [02](screenshots/access/desktop-light/02-default-full-height.png) | [02](screenshots/access/desktop-dark/02-default-full-height.png) | · | · |
| NEXT SAVE Works | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Offline | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Error | state | dup | dup | dup | dup | dup | dup |
| PEOPLE Two guardians | state | dup | dup | dup | dup | dup | dup |
| PEOPLE Only you | state | [03](screenshots/access/phone-light/03-people-only-you.png) | [03](screenshots/access/phone-dark/03-people-only-you.png) | [03](screenshots/access/desktop-light/03-people-only-you.png) | [03](screenshots/access/desktop-dark/03-people-only-you.png) | [02](screenshots/access/wide-light/02-people-only-you.png) | [02](screenshots/access/wide-dark/02-people-only-you.png) |
| ROLE Guardian | state | dup | dup | dup | dup | dup | dup |
| ROLE Caregiver | state | [04](screenshots/access/phone-light/04-role-caregiver.png) | [04](screenshots/access/phone-dark/04-role-caregiver.png) | [04](screenshots/access/desktop-light/04-role-caregiver.png) | [04](screenshots/access/desktop-dark/04-role-caregiver.png) | [03](screenshots/access/wide-light/03-role-caregiver.png) | [03](screenshots/access/wide-dark/03-role-caregiver.png) |
| sheet More | sheet | [05](screenshots/access/phone-light/05-sheet-more.png) | [05](screenshots/access/phone-dark/05-sheet-more.png) | · | · | · | · |

## add-a-child

Route in the app: `/kids/new`

| State | Kind | phone-light | phone-dark | desktop-light | desktop-dark | wide-light | wide-dark |
|---|---|---|---|---|---|---|---|
| default-frame | frame | [01](screenshots/add-a-child/phone-light/01-default-frame.png) | [01](screenshots/add-a-child/phone-dark/01-default-frame.png) | [01](screenshots/add-a-child/desktop-light/01-default-frame.png) | [01](screenshots/add-a-child/desktop-dark/01-default-frame.png) | [01](screenshots/add-a-child/wide-light/01-default-frame.png) | [01](screenshots/add-a-child/wide-dark/01-default-frame.png) |
| default-full-height | full | [02](screenshots/add-a-child/phone-light/02-default-full-height.png) | [02](screenshots/add-a-child/phone-dark/02-default-full-height.png) | [02](screenshots/add-a-child/desktop-light/02-default-full-height.png) | [02](screenshots/add-a-child/desktop-dark/02-default-full-height.png) | · | · |
| NEXT SAVE Works | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Works after action | state-after-action | [03](screenshots/add-a-child/phone-light/03-next-save-works-after-action.png) | [03](screenshots/add-a-child/phone-dark/03-next-save-works-after-action.png) | [03](screenshots/add-a-child/desktop-light/03-next-save-works-after-action.png) | [03](screenshots/add-a-child/desktop-dark/03-next-save-works-after-action.png) | [02](screenshots/add-a-child/wide-light/02-next-save-works-after-action.png) | [02](screenshots/add-a-child/wide-dark/02-next-save-works-after-action.png) |
| NEXT SAVE Offline | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Offline after action | state-after-action | [04](screenshots/add-a-child/phone-light/04-next-save-offline-after-action.png) | [04](screenshots/add-a-child/phone-dark/04-next-save-offline-after-action.png) | [04](screenshots/add-a-child/desktop-light/04-next-save-offline-after-action.png) | [04](screenshots/add-a-child/desktop-dark/04-next-save-offline-after-action.png) | [03](screenshots/add-a-child/wide-light/03-next-save-offline-after-action.png) | [03](screenshots/add-a-child/wide-dark/03-next-save-offline-after-action.png) |
| NEXT SAVE Error | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Error after action | state-after-action | [05](screenshots/add-a-child/phone-light/05-next-save-error-after-action.png) | [05](screenshots/add-a-child/phone-dark/05-next-save-error-after-action.png) | [05](screenshots/add-a-child/desktop-light/05-next-save-error-after-action.png) | [05](screenshots/add-a-child/desktop-dark/05-next-save-error-after-action.png) | [04](screenshots/add-a-child/wide-light/04-next-save-error-after-action.png) | [04](screenshots/add-a-child/wide-dark/04-next-save-error-after-action.png) |
| SCREEN Empty form | state | dup | dup | dup | dup | dup | dup |
| SCREEN Field messages | state | [06](screenshots/add-a-child/phone-light/06-screen-field-messages.png) | [06](screenshots/add-a-child/phone-dark/06-screen-field-messages.png) | [06](screenshots/add-a-child/desktop-light/06-screen-field-messages.png) | [06](screenshots/add-a-child/desktop-dark/06-screen-field-messages.png) | [05](screenshots/add-a-child/wide-light/05-screen-field-messages.png) | [05](screenshots/add-a-child/wide-dark/05-screen-field-messages.png) |
| ROLE Guardian | state | dup | dup | dup | dup | dup | dup |
| ROLE Caregiver | state | [07](screenshots/add-a-child/phone-light/07-role-caregiver.png) | [07](screenshots/add-a-child/phone-dark/07-role-caregiver.png) | [07](screenshots/add-a-child/desktop-light/07-role-caregiver.png) | [07](screenshots/add-a-child/desktop-dark/07-role-caregiver.png) | [06](screenshots/add-a-child/wide-light/06-role-caregiver.png) | [06](screenshots/add-a-child/wide-dark/06-role-caregiver.png) |
| sheet More | sheet | [08](screenshots/add-a-child/phone-light/08-sheet-more.png) | [08](screenshots/add-a-child/phone-dark/08-sheet-more.png) | · | · | · | · |

## admin

Route in the app: `/admin/parents`

| State | Kind | phone-light | phone-dark | desktop-light | desktop-dark | wide-light | wide-dark |
|---|---|---|---|---|---|---|---|
| default-frame | frame | [01](screenshots/admin/phone-light/01-default-frame.png) | [01](screenshots/admin/phone-dark/01-default-frame.png) | [01](screenshots/admin/desktop-light/01-default-frame.png) | [01](screenshots/admin/desktop-dark/01-default-frame.png) | [01](screenshots/admin/wide-light/01-default-frame.png) | [01](screenshots/admin/wide-dark/01-default-frame.png) |
| default-full-height | full | [02](screenshots/admin/phone-light/02-default-full-height.png) | [02](screenshots/admin/phone-dark/02-default-full-height.png) | [02](screenshots/admin/desktop-light/02-default-full-height.png) | [02](screenshots/admin/desktop-dark/02-default-full-height.png) | [02](screenshots/admin/wide-light/02-default-full-height.png) | [02](screenshots/admin/wide-dark/02-default-full-height.png) |
| NEXT SAVE Works | state | dup | dup | [03](screenshots/admin/desktop-light/03-next-save-works.png) | dup | dup | dup |
| NEXT SAVE Offline | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Error | state | dup | dup | dup | dup | dup | dup |

## child-profile

Route in the app: `/kids/{kid}/settings`

| State | Kind | phone-light | phone-dark | desktop-light | desktop-dark | wide-light | wide-dark |
|---|---|---|---|---|---|---|---|
| default-frame | frame | [01](screenshots/child-profile/phone-light/01-default-frame.png) | [01](screenshots/child-profile/phone-dark/01-default-frame.png) | [01](screenshots/child-profile/desktop-light/01-default-frame.png) | [01](screenshots/child-profile/desktop-dark/01-default-frame.png) | [01](screenshots/child-profile/wide-light/01-default-frame.png) | [01](screenshots/child-profile/wide-dark/01-default-frame.png) |
| default-full-height | full | [02](screenshots/child-profile/phone-light/02-default-full-height.png) | [02](screenshots/child-profile/phone-dark/02-default-full-height.png) | [02](screenshots/child-profile/desktop-light/02-default-full-height.png) | [02](screenshots/child-profile/desktop-dark/02-default-full-height.png) | [02](screenshots/child-profile/wide-light/02-default-full-height.png) | [02](screenshots/child-profile/wide-dark/02-default-full-height.png) |
| NEXT SAVE Works | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Works after action | state-after-action | [03](screenshots/child-profile/phone-light/03-next-save-works-after-action.png) | [03](screenshots/child-profile/phone-dark/03-next-save-works-after-action.png) | [03](screenshots/child-profile/desktop-light/03-next-save-works-after-action.png) | [03](screenshots/child-profile/desktop-dark/03-next-save-works-after-action.png) | [03](screenshots/child-profile/wide-light/03-next-save-works-after-action.png) | [03](screenshots/child-profile/wide-dark/03-next-save-works-after-action.png) |
| NEXT SAVE Offline | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Offline after action | state-after-action | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Error | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Error after action | state-after-action | dup | dup | dup | dup | dup | dup |
| ROLE Guardian | state | dup | dup | dup | dup | dup | dup |
| ROLE Caregiver | state | [04](screenshots/child-profile/phone-light/04-role-caregiver.png) | [04](screenshots/child-profile/phone-dark/04-role-caregiver.png) | [04](screenshots/child-profile/desktop-light/04-role-caregiver.png) | [04](screenshots/child-profile/desktop-dark/04-role-caregiver.png) | [04](screenshots/child-profile/wide-light/04-role-caregiver.png) | [04](screenshots/child-profile/wide-dark/04-role-caregiver.png) |
| CAREGIVER VIEW No page | state | dup | dup | dup | dup | dup | dup |
| CAREGIVER VIEW Read-only | state | dup | dup | dup | dup | dup | dup |
| sheet Switch child | sheet | [05](screenshots/child-profile/phone-light/05-sheet-switch-child.png) | [05](screenshots/child-profile/phone-dark/05-sheet-switch-child.png) | [05](screenshots/child-profile/desktop-light/05-sheet-switch-child.png) | [05](screenshots/child-profile/desktop-dark/05-sheet-switch-child.png) | [05](screenshots/child-profile/wide-light/05-sheet-switch-child.png) | [05](screenshots/child-profile/wide-dark/05-sheet-switch-child.png) |
| sheet More | sheet | [06](screenshots/child-profile/phone-light/06-sheet-more.png) | [06](screenshots/child-profile/phone-dark/06-sheet-more.png) | · | · | · | · |

## child-settings

Route in the app: `/kids/{kid}/settings`

| State | Kind | phone-light | phone-dark | desktop-light | desktop-dark | wide-light | wide-dark |
|---|---|---|---|---|---|---|---|
| default-frame | frame | [01](screenshots/child-settings/phone-light/01-default-frame.png) | [01](screenshots/child-settings/phone-dark/01-default-frame.png) | [01](screenshots/child-settings/desktop-light/01-default-frame.png) | [01](screenshots/child-settings/desktop-dark/01-default-frame.png) | [01](screenshots/child-settings/wide-light/01-default-frame.png) | [01](screenshots/child-settings/wide-dark/01-default-frame.png) |
| default-full-height | full | [02](screenshots/child-settings/phone-light/02-default-full-height.png) | [02](screenshots/child-settings/phone-dark/02-default-full-height.png) | [02](screenshots/child-settings/desktop-light/02-default-full-height.png) | [02](screenshots/child-settings/desktop-dark/02-default-full-height.png) | · | · |
| NEXT SAVE Works | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Works after action | state-after-action | [03](screenshots/child-settings/phone-light/03-next-save-works-after-action.png) | [03](screenshots/child-settings/phone-dark/03-next-save-works-after-action.png) | [03](screenshots/child-settings/desktop-light/03-next-save-works-after-action.png) | [03](screenshots/child-settings/desktop-dark/03-next-save-works-after-action.png) | [02](screenshots/child-settings/wide-light/02-next-save-works-after-action.png) | [02](screenshots/child-settings/wide-dark/02-next-save-works-after-action.png) |
| NEXT SAVE Offline | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Offline after action | state-after-action | [04](screenshots/child-settings/phone-light/04-next-save-offline-after-action.png) | [04](screenshots/child-settings/phone-dark/04-next-save-offline-after-action.png) | [04](screenshots/child-settings/desktop-light/04-next-save-offline-after-action.png) | [04](screenshots/child-settings/desktop-dark/04-next-save-offline-after-action.png) | [03](screenshots/child-settings/wide-light/03-next-save-offline-after-action.png) | [03](screenshots/child-settings/wide-dark/03-next-save-offline-after-action.png) |
| NEXT SAVE Error | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Error after action | state-after-action | [05](screenshots/child-settings/phone-light/05-next-save-error-after-action.png) | [05](screenshots/child-settings/phone-dark/05-next-save-error-after-action.png) | [05](screenshots/child-settings/desktop-light/05-next-save-error-after-action.png) | [05](screenshots/child-settings/desktop-dark/05-next-save-error-after-action.png) | [04](screenshots/child-settings/wide-light/04-next-save-error-after-action.png) | [04](screenshots/child-settings/wide-dark/04-next-save-error-after-action.png) |
| SCREEN Filled form | state | dup | dup | dup | dup | dup | dup |
| SCREEN Field messages | state | [06](screenshots/child-settings/phone-light/06-screen-field-messages.png) | [06](screenshots/child-settings/phone-dark/06-screen-field-messages.png) | [06](screenshots/child-settings/desktop-light/06-screen-field-messages.png) | [06](screenshots/child-settings/desktop-dark/06-screen-field-messages.png) | [05](screenshots/child-settings/wide-light/05-screen-field-messages.png) | [05](screenshots/child-settings/wide-dark/05-screen-field-messages.png) |
| ROLE Guardian | state | dup | dup | dup | dup | dup | dup |
| ROLE Caregiver | state | [07](screenshots/child-settings/phone-light/07-role-caregiver.png) | [07](screenshots/child-settings/phone-dark/07-role-caregiver.png) | [07](screenshots/child-settings/desktop-light/07-role-caregiver.png) | [07](screenshots/child-settings/desktop-dark/07-role-caregiver.png) | [06](screenshots/child-settings/wide-light/06-role-caregiver.png) | [06](screenshots/child-settings/wide-dark/06-role-caregiver.png) |
| sheet More | sheet | [08](screenshots/child-settings/phone-light/08-sheet-more.png) | [08](screenshots/child-settings/phone-dark/08-sheet-more.png) | · | · | · | · |

## choose-child

Route in the app: `/kids`

| State | Kind | phone-light | phone-dark | desktop-light | desktop-dark | wide-light | wide-dark |
|---|---|---|---|---|---|---|---|
| default-frame | frame | [01](screenshots/choose-child/phone-light/01-default-frame.png) | [01](screenshots/choose-child/phone-dark/01-default-frame.png) | [01](screenshots/choose-child/desktop-light/01-default-frame.png) | [01](screenshots/choose-child/desktop-dark/01-default-frame.png) | [01](screenshots/choose-child/wide-light/01-default-frame.png) | [01](screenshots/choose-child/wide-dark/01-default-frame.png) |
| NEXT SAVE Works | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Offline | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Error | state | dup | dup | dup | dup | dup | dup |
| CHILDREN Three | state | dup | dup | dup | dup | dup | dup |
| CHILDREN Nine, long names | state | [02](screenshots/choose-child/phone-light/02-children-nine-long-names.png) | [02](screenshots/choose-child/phone-dark/02-children-nine-long-names.png) | [02](screenshots/choose-child/desktop-light/02-children-nine-long-names.png) | [02](screenshots/choose-child/desktop-dark/02-children-nine-long-names.png) | [02](screenshots/choose-child/wide-light/02-children-nine-long-names.png) | [02](screenshots/choose-child/wide-dark/02-children-nine-long-names.png) |
| ROLE Guardian | state | dup | dup | dup | dup | dup | dup |
| ROLE Caregiver | state | [03](screenshots/choose-child/phone-light/03-role-caregiver.png) | [03](screenshots/choose-child/phone-dark/03-role-caregiver.png) | [03](screenshots/choose-child/desktop-light/03-role-caregiver.png) | [03](screenshots/choose-child/desktop-dark/03-role-caregiver.png) | [03](screenshots/choose-child/wide-light/03-role-caregiver.png) | [03](screenshots/choose-child/wide-dark/03-role-caregiver.png) |
| sheet More | sheet | [04](screenshots/choose-child/phone-light/04-sheet-more.png) | [04](screenshots/choose-child/phone-dark/04-sheet-more.png) | · | · | · | · |

## course

Route in the app: `/kids/{kid}/medication/new`

| State | Kind | phone-light | phone-dark | desktop-light | desktop-dark | wide-light | wide-dark |
|---|---|---|---|---|---|---|---|
| default-frame | frame | [01](screenshots/course/phone-light/01-default-frame.png) | [01](screenshots/course/phone-dark/01-default-frame.png) | [01](screenshots/course/desktop-light/01-default-frame.png) | [01](screenshots/course/desktop-dark/01-default-frame.png) | [01](screenshots/course/wide-light/01-default-frame.png) | [01](screenshots/course/wide-dark/01-default-frame.png) |
| default-full-height | full | [02](screenshots/course/phone-light/02-default-full-height.png) | [02](screenshots/course/phone-dark/02-default-full-height.png) | [02](screenshots/course/desktop-light/02-default-full-height.png) | [02](screenshots/course/desktop-dark/02-default-full-height.png) | · | · |
| NEXT SAVE Works | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Works after action | state-after-action | [03](screenshots/course/phone-light/03-next-save-works-after-action.png) | [03](screenshots/course/phone-dark/03-next-save-works-after-action.png) | [03](screenshots/course/desktop-light/03-next-save-works-after-action.png) | [03](screenshots/course/desktop-dark/03-next-save-works-after-action.png) | [02](screenshots/course/wide-light/02-next-save-works-after-action.png) | [02](screenshots/course/wide-dark/02-next-save-works-after-action.png) |
| NEXT SAVE Offline | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Offline after action | state-after-action | [04](screenshots/course/phone-light/04-next-save-offline-after-action.png) | [04](screenshots/course/phone-dark/04-next-save-offline-after-action.png) | [04](screenshots/course/desktop-light/04-next-save-offline-after-action.png) | [04](screenshots/course/desktop-dark/04-next-save-offline-after-action.png) | [03](screenshots/course/wide-light/03-next-save-offline-after-action.png) | [03](screenshots/course/wide-dark/03-next-save-offline-after-action.png) |
| NEXT SAVE Error | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Error after action | state-after-action | [05](screenshots/course/phone-light/05-next-save-error-after-action.png) | [05](screenshots/course/phone-dark/05-next-save-error-after-action.png) | [05](screenshots/course/desktop-light/05-next-save-error-after-action.png) | [05](screenshots/course/desktop-dark/05-next-save-error-after-action.png) | [04](screenshots/course/wide-light/04-next-save-error-after-action.png) | [04](screenshots/course/wide-dark/04-next-save-error-after-action.png) |
| SCREEN Add | state | dup | dup | dup | dup | dup | dup |
| SCREEN Edit | state | [06](screenshots/course/phone-light/06-screen-edit.png) | [06](screenshots/course/phone-dark/06-screen-edit.png) | [06](screenshots/course/desktop-light/06-screen-edit.png) | [06](screenshots/course/desktop-dark/06-screen-edit.png) | [05](screenshots/course/wide-light/05-screen-edit.png) | [05](screenshots/course/wide-dark/05-screen-edit.png) |
| EDIT WHICH COURSE Calpol (fixed) | state | [07](screenshots/course/phone-light/07-edit-which-course-calpol-fixed.png) | [07](screenshots/course/phone-dark/07-edit-which-course-calpol-fixed.png) | [07](screenshots/course/desktop-light/07-edit-which-course-calpol-fixed.png) | [07](screenshots/course/desktop-dark/07-edit-which-course-calpol-fixed.png) | [06](screenshots/course/wide-light/06-edit-which-course-calpol-fixed.png) | [06](screenshots/course/wide-dark/06-edit-which-course-calpol-fixed.png) |
| EDIT WHICH COURSE Amoxil (hourly) | state | dup | dup | dup | dup | dup | dup |
| EDIT WHICH COURSE Saline (0 doses) | state | [08](screenshots/course/phone-light/08-edit-which-course-saline-0-doses.png) | [08](screenshots/course/phone-dark/08-edit-which-course-saline-0-doses.png) | [08](screenshots/course/desktop-light/08-edit-which-course-saline-0-doses.png) | [08](screenshots/course/desktop-dark/08-edit-which-course-saline-0-doses.png) | [07](screenshots/course/wide-light/07-edit-which-course-saline-0-doses.png) | [07](screenshots/course/wide-dark/07-edit-which-course-saline-0-doses.png) |
| All 12 (as built) | state | dup | dup | dup | dup | dup | dup |
| 3 + “More times” | state | dup | dup | dup | dup | dup | dup |
| As built | state | dup | dup | dup | dup | dup | dup |
| Omit the line | state | dup | dup | dup | dup | dup | dup |
| ROLE Guardian | state | dup | dup | dup | dup | dup | dup |
| ROLE Caregiver | state | [09](screenshots/course/phone-light/09-role-caregiver.png) | [09](screenshots/course/phone-dark/09-role-caregiver.png) | [09](screenshots/course/desktop-light/09-role-caregiver.png) | [09](screenshots/course/desktop-dark/09-role-caregiver.png) | [08](screenshots/course/wide-light/08-role-caregiver.png) | [08](screenshots/course/wide-dark/08-role-caregiver.png) |
| sheet More | sheet | [10](screenshots/course/phone-light/10-sheet-more.png) | [10](screenshots/course/phone-dark/10-sheet-more.png) | · | · | · | · |

## dose

Route in the app: `/kids/{kid}/medication/doses/{dose}`

| State | Kind | phone-light | phone-dark | desktop-light | desktop-dark | wide-light | wide-dark |
|---|---|---|---|---|---|---|---|
| default-frame | frame | [01](screenshots/dose/phone-light/01-default-frame.png) | [01](screenshots/dose/phone-dark/01-default-frame.png) | [01](screenshots/dose/desktop-light/01-default-frame.png) | [01](screenshots/dose/desktop-dark/01-default-frame.png) | [01](screenshots/dose/wide-light/01-default-frame.png) | [01](screenshots/dose/wide-dark/01-default-frame.png) |
| NEXT SAVE Works | state | dup | dup | [02](screenshots/dose/desktop-light/02-next-save-works.png) | dup | dup | dup |
| NEXT SAVE Works after action | state-after-action | [02](screenshots/dose/phone-light/02-next-save-works-after-action.png) | [02](screenshots/dose/phone-dark/02-next-save-works-after-action.png) | [03](screenshots/dose/desktop-light/03-next-save-works-after-action.png) | [02](screenshots/dose/desktop-dark/02-next-save-works-after-action.png) | [02](screenshots/dose/wide-light/02-next-save-works-after-action.png) | [02](screenshots/dose/wide-dark/02-next-save-works-after-action.png) |
| NEXT SAVE Offline | state | dup | dup | dup | dup | [03](screenshots/dose/wide-light/03-next-save-offline.png) | dup |
| NEXT SAVE Offline after action | state-after-action | [03](screenshots/dose/phone-light/03-next-save-offline-after-action.png) | [03](screenshots/dose/phone-dark/03-next-save-offline-after-action.png) | [04](screenshots/dose/desktop-light/04-next-save-offline-after-action.png) | [03](screenshots/dose/desktop-dark/03-next-save-offline-after-action.png) | [04](screenshots/dose/wide-light/04-next-save-offline-after-action.png) | [03](screenshots/dose/wide-dark/03-next-save-offline-after-action.png) |
| NEXT SAVE Error | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Error after action | state-after-action | [04](screenshots/dose/phone-light/04-next-save-error-after-action.png) | [04](screenshots/dose/phone-dark/04-next-save-error-after-action.png) | [05](screenshots/dose/desktop-light/05-next-save-error-after-action.png) | [04](screenshots/dose/desktop-dark/04-next-save-error-after-action.png) | [05](screenshots/dose/wide-light/05-next-save-error-after-action.png) | [04](screenshots/dose/wide-dark/04-next-save-error-after-action.png) |
| ROLE Guardian | state | dup | dup | dup | dup | dup | dup |
| ROLE Caregiver | state | [05](screenshots/dose/phone-light/05-role-caregiver.png) | [05](screenshots/dose/phone-dark/05-role-caregiver.png) | [06](screenshots/dose/desktop-light/06-role-caregiver.png) | [05](screenshots/dose/desktop-dark/05-role-caregiver.png) | [06](screenshots/dose/wide-light/06-role-caregiver.png) | [05](screenshots/dose/wide-dark/05-role-caregiver.png) |
| sheet More | sheet | [06](screenshots/dose/phone-light/06-sheet-more.png) | [06](screenshots/dose/phone-dark/06-sheet-more.png) | · | · | · | · |

## error

Route in the app: `/does-not-exist`

| State | Kind | phone-light | phone-dark | desktop-light | desktop-dark | wide-light | wide-dark |
|---|---|---|---|---|---|---|---|
| default-frame | frame | [01](screenshots/error/phone-light/01-default-frame.png) | [01](screenshots/error/phone-dark/01-default-frame.png) | [01](screenshots/error/desktop-light/01-default-frame.png) | [01](screenshots/error/desktop-dark/01-default-frame.png) | [01](screenshots/error/wide-light/01-default-frame.png) | [01](screenshots/error/wide-dark/01-default-frame.png) |
| PAGE 503 Unavailable | state | [02](screenshots/error/phone-light/02-page-503-unavailable.png) | [02](screenshots/error/phone-dark/02-page-503-unavailable.png) | [02](screenshots/error/desktop-light/02-page-503-unavailable.png) | [02](screenshots/error/desktop-dark/02-page-503-unavailable.png) | [02](screenshots/error/wide-light/02-page-503-unavailable.png) | [02](screenshots/error/wide-dark/02-page-503-unavailable.png) |

## export

Route in the app: `/kids/{kid}/export`

| State | Kind | phone-light | phone-dark | desktop-light | desktop-dark | wide-light | wide-dark |
|---|---|---|---|---|---|---|---|
| default-frame | frame | [01](screenshots/export/phone-light/01-default-frame.png) | [01](screenshots/export/phone-dark/01-default-frame.png) | [01](screenshots/export/desktop-light/01-default-frame.png) | [01](screenshots/export/desktop-dark/01-default-frame.png) | [01](screenshots/export/wide-light/01-default-frame.png) | [01](screenshots/export/wide-dark/01-default-frame.png) |
| NEXT SAVE Works | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Works after action | state-after-action | [02](screenshots/export/phone-light/02-next-save-works-after-action.png) | [02](screenshots/export/phone-dark/02-next-save-works-after-action.png) | [02](screenshots/export/desktop-light/02-next-save-works-after-action.png) | [02](screenshots/export/desktop-dark/02-next-save-works-after-action.png) | [02](screenshots/export/wide-light/02-next-save-works-after-action.png) | [02](screenshots/export/wide-dark/02-next-save-works-after-action.png) |
| NEXT SAVE Offline | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Offline after action | state-after-action | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Error | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Error after action | state-after-action | [03](screenshots/export/phone-light/03-next-save-error-after-action.png) | [03](screenshots/export/phone-dark/03-next-save-error-after-action.png) | dup | dup | dup | dup |
| MODULES All on | state | dup | dup | dup | dup | dup | dup |
| MODULES Some switched off | state | [04](screenshots/export/phone-light/04-modules-some-switched-off.png) | [04](screenshots/export/phone-dark/04-modules-some-switched-off.png) | [03](screenshots/export/desktop-light/03-modules-some-switched-off.png) | [03](screenshots/export/desktop-dark/03-modules-some-switched-off.png) | [03](screenshots/export/wide-light/03-modules-some-switched-off.png) | [03](screenshots/export/wide-dark/03-modules-some-switched-off.png) |
| AFTER A REFUSAL (REDIRECT) None | state | dup | dup | dup | dup | dup | dup |
| AFTER A REFUSAL (REDIRECT) None after action | state-after-action | dup | dup | dup | dup | dup | dup |
| AFTER A REFUSAL (REDIRECT) From | state | [05](screenshots/export/phone-light/05-after-a-refusal-redirect-from.png) | [05](screenshots/export/phone-dark/05-after-a-refusal-redirect-from.png) | [04](screenshots/export/desktop-light/04-after-a-refusal-redirect-from.png) | [04](screenshots/export/desktop-dark/04-after-a-refusal-redirect-from.png) | [04](screenshots/export/wide-light/04-after-a-refusal-redirect-from.png) | [04](screenshots/export/wide-dark/04-after-a-refusal-redirect-from.png) |
| AFTER A REFUSAL (REDIRECT) From after action | state-after-action | dup | dup | dup | dup | dup | dup |
| AFTER A REFUSAL (REDIRECT) To | state | [06](screenshots/export/phone-light/06-after-a-refusal-redirect-to.png) | [06](screenshots/export/phone-dark/06-after-a-refusal-redirect-to.png) | [05](screenshots/export/desktop-light/05-after-a-refusal-redirect-to.png) | [05](screenshots/export/desktop-dark/05-after-a-refusal-redirect-to.png) | [05](screenshots/export/wide-light/05-after-a-refusal-redirect-to.png) | [05](screenshots/export/wide-dark/05-after-a-refusal-redirect-to.png) |
| AFTER A REFUSAL (REDIRECT) To after action | state-after-action | dup | dup | dup | dup | dup | dup |
| AFTER A REFUSAL (REDIRECT) Modules | state | [07](screenshots/export/phone-light/07-after-a-refusal-redirect-modules.png) | [07](screenshots/export/phone-dark/07-after-a-refusal-redirect-modules.png) | [06](screenshots/export/desktop-light/06-after-a-refusal-redirect-modules.png) | [06](screenshots/export/desktop-dark/06-after-a-refusal-redirect-modules.png) | [06](screenshots/export/wide-light/06-after-a-refusal-redirect-modules.png) | [06](screenshots/export/wide-dark/06-after-a-refusal-redirect-modules.png) |
| AFTER A REFUSAL (REDIRECT) Modules after action | state-after-action | dup | dup | dup | dup | dup | dup |
| AFTER A REFUSAL (REDIRECT) Busy | state | [08](screenshots/export/phone-light/08-after-a-refusal-redirect-busy.png) | [08](screenshots/export/phone-dark/08-after-a-refusal-redirect-busy.png) | [07](screenshots/export/desktop-light/07-after-a-refusal-redirect-busy.png) | [07](screenshots/export/desktop-dark/07-after-a-refusal-redirect-busy.png) | [07](screenshots/export/wide-light/07-after-a-refusal-redirect-busy.png) | [07](screenshots/export/wide-dark/07-after-a-refusal-redirect-busy.png) |
| AFTER A REFUSAL (REDIRECT) Busy after action | state-after-action | [09](screenshots/export/phone-light/09-after-a-refusal-redirect-busy-after-action.png) | [09](screenshots/export/phone-dark/09-after-a-refusal-redirect-busy-after-action.png) | dup | dup | [08](screenshots/export/wide-light/08-after-a-refusal-redirect-busy-after-action.png) | [08](screenshots/export/wide-dark/08-after-a-refusal-redirect-busy-after-action.png) |
| PAGE STATE Normal | state | dup | dup | dup | dup | dup | dup |
| PAGE STATE Not found | state | [10](screenshots/export/phone-light/10-page-state-not-found.png) | [10](screenshots/export/phone-dark/10-page-state-not-found.png) | [08](screenshots/export/desktop-light/08-page-state-not-found.png) | [08](screenshots/export/desktop-dark/08-page-state-not-found.png) | [09](screenshots/export/wide-light/09-page-state-not-found.png) | [09](screenshots/export/wide-dark/09-page-state-not-found.png) |
| ROLE Guardian | state | dup | dup | dup | dup | dup | dup |
| ROLE Caregiver | state | [11](screenshots/export/phone-light/11-role-caregiver.png) | [11](screenshots/export/phone-dark/11-role-caregiver.png) | [09](screenshots/export/desktop-light/09-role-caregiver.png) | [09](screenshots/export/desktop-dark/09-role-caregiver.png) | [10](screenshots/export/wide-light/10-role-caregiver.png) | [10](screenshots/export/wide-dark/10-role-caregiver.png) |
| sheet Switch child | sheet | [12](screenshots/export/phone-light/12-sheet-switch-child.png) | [12](screenshots/export/phone-dark/12-sheet-switch-child.png) | [10](screenshots/export/desktop-light/10-sheet-switch-child.png) | [10](screenshots/export/desktop-dark/10-sheet-switch-child.png) | [11](screenshots/export/wide-light/11-sheet-switch-child.png) | [11](screenshots/export/wide-dark/11-sheet-switch-child.png) |
| sheet More | sheet | [13](screenshots/export/phone-light/13-sheet-more.png) | [13](screenshots/export/phone-dark/13-sheet-more.png) | · | · | · | · |

## growth

Route in the app: `/kids/{kid}/growth`

| State | Kind | phone-light | phone-dark | desktop-light | desktop-dark | wide-light | wide-dark |
|---|---|---|---|---|---|---|---|
| default-frame | frame | [01](screenshots/growth/phone-light/01-default-frame.png) | [01](screenshots/growth/phone-dark/01-default-frame.png) | [01](screenshots/growth/desktop-light/01-default-frame.png) | [01](screenshots/growth/desktop-dark/01-default-frame.png) | [01](screenshots/growth/wide-light/01-default-frame.png) | [01](screenshots/growth/wide-dark/01-default-frame.png) |
| default-full-height | full | [02](screenshots/growth/phone-light/02-default-full-height.png) | [02](screenshots/growth/phone-dark/02-default-full-height.png) | [02](screenshots/growth/desktop-light/02-default-full-height.png) | [02](screenshots/growth/desktop-dark/02-default-full-height.png) | [02](screenshots/growth/wide-light/02-default-full-height.png) | [02](screenshots/growth/wide-dark/02-default-full-height.png) |
| CHILD Ivy | state | [03](screenshots/growth/phone-light/03-child-ivy.png) | [03](screenshots/growth/phone-dark/03-child-ivy.png) | [03](screenshots/growth/desktop-light/03-child-ivy.png) | [03](screenshots/growth/desktop-dark/03-child-ivy.png) | [03](screenshots/growth/wide-light/03-child-ivy.png) | [03](screenshots/growth/wide-dark/03-child-ivy.png) |
| CHILD Mia | state | [04](screenshots/growth/phone-light/04-child-mia.png) | [04](screenshots/growth/phone-dark/04-child-mia.png) | [04](screenshots/growth/desktop-light/04-child-mia.png) | [04](screenshots/growth/desktop-dark/04-child-mia.png) | [04](screenshots/growth/wide-light/04-child-mia.png) | [04](screenshots/growth/wide-dark/04-child-mia.png) |
| CHILD Leo · preterm | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Works | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Works after action | state-after-action | [05](screenshots/growth/phone-light/05-next-save-works-after-action.png) | [05](screenshots/growth/phone-dark/05-next-save-works-after-action.png) | [05](screenshots/growth/desktop-light/05-next-save-works-after-action.png) | [05](screenshots/growth/desktop-dark/05-next-save-works-after-action.png) | [05](screenshots/growth/wide-light/05-next-save-works-after-action.png) | [05](screenshots/growth/wide-dark/05-next-save-works-after-action.png) |
| NEXT SAVE Offline | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Offline after action | state-after-action | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Error | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Error after action | state-after-action | dup | dup | dup | dup | dup | dup |
| SOURCE WHO | state | dup | dup | dup | dup | dup | dup |
| SOURCE CDC | state | [06](screenshots/growth/phone-light/06-source-cdc.png) | [06](screenshots/growth/phone-dark/06-source-cdc.png) | [06](screenshots/growth/desktop-light/06-source-cdc.png) | [06](screenshots/growth/desktop-dark/06-source-cdc.png) | [06](screenshots/growth/wide-light/06-source-cdc.png) | [06](screenshots/growth/wide-dark/06-source-cdc.png) |
| DATA Rows | state | dup | dup | dup | dup | dup | dup |
| DATA Empty | state | [07](screenshots/growth/phone-light/07-data-empty.png) | [07](screenshots/growth/phone-dark/07-data-empty.png) | [07](screenshots/growth/desktop-light/07-data-empty.png) | [07](screenshots/growth/desktop-dark/07-data-empty.png) | [07](screenshots/growth/wide-light/07-data-empty.png) | [07](screenshots/growth/wide-dark/07-data-empty.png) |
| UNITS ml · kg | state | dup | dup | dup | dup | dup | dup |
| UNITS fl oz · lb | state | [08](screenshots/growth/phone-light/08-units-fl-oz-lb.png) | [08](screenshots/growth/phone-dark/08-units-fl-oz-lb.png) | [08](screenshots/growth/desktop-light/08-units-fl-oz-lb.png) | [08](screenshots/growth/desktop-dark/08-units-fl-oz-lb.png) | [08](screenshots/growth/wide-light/08-units-fl-oz-lb.png) | [08](screenshots/growth/wide-dark/08-units-fl-oz-lb.png) |
| ROLE Guardian | state | dup | dup | dup | dup | dup | dup |
| ROLE Caregiver | state | [09](screenshots/growth/phone-light/09-role-caregiver.png) | [09](screenshots/growth/phone-dark/09-role-caregiver.png) | [09](screenshots/growth/desktop-light/09-role-caregiver.png) | [09](screenshots/growth/desktop-dark/09-role-caregiver.png) | [09](screenshots/growth/wide-light/09-role-caregiver.png) | [09](screenshots/growth/wide-dark/09-role-caregiver.png) |
| sheet Switch child | sheet | [10](screenshots/growth/phone-light/10-sheet-switch-child.png) | [10](screenshots/growth/phone-dark/10-sheet-switch-child.png) | [10](screenshots/growth/desktop-light/10-sheet-switch-child.png) | [10](screenshots/growth/desktop-dark/10-sheet-switch-child.png) | [10](screenshots/growth/wide-light/10-sheet-switch-child.png) | [10](screenshots/growth/wide-dark/10-sheet-switch-child.png) |
| sheet More | sheet | [11](screenshots/growth/phone-light/11-sheet-more.png) | [11](screenshots/growth/phone-dark/11-sheet-more.png) | · | · | · | · |

## kid-home

Route in the app: `/kids/{kid}`

| State | Kind | phone-light | phone-dark | desktop-light | desktop-dark | wide-light | wide-dark |
|---|---|---|---|---|---|---|---|
| default-frame | frame | [01](screenshots/kid-home/phone-light/01-default-frame.png) | [01](screenshots/kid-home/phone-dark/01-default-frame.png) | [01](screenshots/kid-home/desktop-light/01-default-frame.png) | [01](screenshots/kid-home/desktop-dark/01-default-frame.png) | [01](screenshots/kid-home/wide-light/01-default-frame.png) | [01](screenshots/kid-home/wide-dark/01-default-frame.png) |
| default-full-height | full | [02](screenshots/kid-home/phone-light/02-default-full-height.png) | [02](screenshots/kid-home/phone-dark/02-default-full-height.png) | [02](screenshots/kid-home/desktop-light/02-default-full-height.png) | [02](screenshots/kid-home/desktop-dark/02-default-full-height.png) | [02](screenshots/kid-home/wide-light/02-default-full-height.png) | [02](screenshots/kid-home/wide-dark/02-default-full-height.png) |
| AGE BAND 0–11 mo | state | dup | dup | dup | dup | dup | dup |
| AGE BAND 12–23 mo | state | [03](screenshots/kid-home/phone-light/03-age-band-12-23-mo.png) | [03](screenshots/kid-home/phone-dark/03-age-band-12-23-mo.png) | [03](screenshots/kid-home/desktop-light/03-age-band-12-23-mo.png) | [03](screenshots/kid-home/desktop-dark/03-age-band-12-23-mo.png) | [03](screenshots/kid-home/wide-light/03-age-band-12-23-mo.png) | [03](screenshots/kid-home/wide-dark/03-age-band-12-23-mo.png) |
| AGE BAND 24 mo + | state | [04](screenshots/kid-home/phone-light/04-age-band-24-mo.png) | [04](screenshots/kid-home/phone-dark/04-age-band-24-mo.png) | [04](screenshots/kid-home/desktop-light/04-age-band-24-mo.png) | [04](screenshots/kid-home/desktop-dark/04-age-band-24-mo.png) | [04](screenshots/kid-home/wide-light/04-age-band-24-mo.png) | [04](screenshots/kid-home/wide-dark/04-age-band-24-mo.png) |
| UNITS ml | state | dup | dup | dup | dup | dup | dup |
| UNITS fl oz | state | [05](screenshots/kid-home/phone-light/05-units-fl-oz.png) | [05](screenshots/kid-home/phone-dark/05-units-fl-oz.png) | [05](screenshots/kid-home/desktop-light/05-units-fl-oz.png) | [05](screenshots/kid-home/desktop-dark/05-units-fl-oz.png) | [05](screenshots/kid-home/wide-light/05-units-fl-oz.png) | [05](screenshots/kid-home/wide-dark/05-units-fl-oz.png) |
| ROLE Guardian | state | dup | dup | dup | dup | dup | dup |
| ROLE Caregiver | state | [06](screenshots/kid-home/phone-light/06-role-caregiver.png) | [06](screenshots/kid-home/phone-dark/06-role-caregiver.png) | [06](screenshots/kid-home/desktop-light/06-role-caregiver.png) | [06](screenshots/kid-home/desktop-dark/06-role-caregiver.png) | [06](screenshots/kid-home/wide-light/06-role-caregiver.png) | [06](screenshots/kid-home/wide-dark/06-role-caregiver.png) |
| NEXT SAVE Works | state | dup | dup | dup | [07](screenshots/kid-home/desktop-dark/07-next-save-works.png) | dup | dup |
| NEXT SAVE Works after action | state-after-action | [07](screenshots/kid-home/phone-light/07-next-save-works-after-action.png) | [07](screenshots/kid-home/phone-dark/07-next-save-works-after-action.png) | [07](screenshots/kid-home/desktop-light/07-next-save-works-after-action.png) | [08](screenshots/kid-home/desktop-dark/08-next-save-works-after-action.png) | [07](screenshots/kid-home/wide-light/07-next-save-works-after-action.png) | [07](screenshots/kid-home/wide-dark/07-next-save-works-after-action.png) |
| NEXT SAVE Offline | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Offline after action | state-after-action | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Error | state | dup | dup | [08](screenshots/kid-home/desktop-light/08-next-save-error.png) | dup | dup | dup |
| NEXT SAVE Error after action | state-after-action | dup | dup | [09](screenshots/kid-home/desktop-light/09-next-save-error-after-action.png) | dup | dup | dup |
| NEXT SAVE Clash | state | dup | dup | [10](screenshots/kid-home/desktop-light/10-next-save-clash.png) | dup | dup | dup |
| NEXT SAVE Clash after action | state-after-action | dup | dup | [11](screenshots/kid-home/desktop-light/11-next-save-clash-after-action.png) | dup | dup | dup |
| CHILDREN Has children | state | dup | dup | dup | dup | dup | dup |
| CHILDREN None yet | state | [08](screenshots/kid-home/phone-light/08-children-none-yet.png) | [08](screenshots/kid-home/phone-dark/08-children-none-yet.png) | [12](screenshots/kid-home/desktop-light/12-children-none-yet.png) | [09](screenshots/kid-home/desktop-dark/09-children-none-yet.png) | [08](screenshots/kid-home/wide-light/08-children-none-yet.png) | [08](screenshots/kid-home/wide-dark/08-children-none-yet.png) |
| Simulate: Cleo starts a sleep on her phone | state | [09](screenshots/kid-home/phone-light/09-simulate-cleo-starts-a-sleep-on-her-phone.png) | [09](screenshots/kid-home/phone-dark/09-simulate-cleo-starts-a-sleep-on-her-phone.png) | [13](screenshots/kid-home/desktop-light/13-simulate-cleo-starts-a-sleep-on-her-phone.png) | [10](screenshots/kid-home/desktop-dark/10-simulate-cleo-starts-a-sleep-on-her-phone.png) | [09](screenshots/kid-home/wide-light/09-simulate-cleo-starts-a-sleep-on-her-phone.png) | [09](screenshots/kid-home/wide-dark/09-simulate-cleo-starts-a-sleep-on-her-phone.png) |
| Simulate: Cleo starts a sleep on her phone after action | state-after-action | [10](screenshots/kid-home/phone-light/10-simulate-cleo-starts-a-sleep-on-her-phone-after-.png) | [10](screenshots/kid-home/phone-dark/10-simulate-cleo-starts-a-sleep-on-her-phone-after-.png) | [14](screenshots/kid-home/desktop-light/14-simulate-cleo-starts-a-sleep-on-her-phone-after-.png) | [11](screenshots/kid-home/desktop-dark/11-simulate-cleo-starts-a-sleep-on-her-phone-after-.png) | [10](screenshots/kid-home/wide-light/10-simulate-cleo-starts-a-sleep-on-her-phone-after-.png) | [10](screenshots/kid-home/wide-dark/10-simulate-cleo-starts-a-sleep-on-her-phone-after-.png) |
| Simulate: a sleep left running for 26 h | state | [11](screenshots/kid-home/phone-light/11-simulate-a-sleep-left-running-for-26-h.png) | [11](screenshots/kid-home/phone-dark/11-simulate-a-sleep-left-running-for-26-h.png) | [15](screenshots/kid-home/desktop-light/15-simulate-a-sleep-left-running-for-26-h.png) | [12](screenshots/kid-home/desktop-dark/12-simulate-a-sleep-left-running-for-26-h.png) | [11](screenshots/kid-home/wide-light/11-simulate-a-sleep-left-running-for-26-h.png) | [11](screenshots/kid-home/wide-dark/11-simulate-a-sleep-left-running-for-26-h.png) |
| Simulate: a sleep left running for 26 h after action | state-after-action | [12](screenshots/kid-home/phone-light/12-simulate-a-sleep-left-running-for-26-h-after-act.png) | [12](screenshots/kid-home/phone-dark/12-simulate-a-sleep-left-running-for-26-h-after-act.png) | [16](screenshots/kid-home/desktop-light/16-simulate-a-sleep-left-running-for-26-h-after-act.png) | [13](screenshots/kid-home/desktop-dark/13-simulate-a-sleep-left-running-for-26-h-after-act.png) | [12](screenshots/kid-home/wide-light/12-simulate-a-sleep-left-running-for-26-h-after-act.png) | [12](screenshots/kid-home/wide-dark/12-simulate-a-sleep-left-running-for-26-h-after-act.png) |
| Medication link: always shown (as built) | state | dup | [13](screenshots/kid-home/phone-dark/13-medication-link-always-shown-as-built.png) | [17](screenshots/kid-home/desktop-light/17-medication-link-always-shown-as-built.png) | [14](screenshots/kid-home/desktop-dark/14-medication-link-always-shown-as-built.png) | dup | [13](screenshots/kid-home/wide-dark/13-medication-link-always-shown-as-built.png) |
| Medication link: always shown (as built) after action | state-after-action | dup | [14](screenshots/kid-home/phone-dark/14-medication-link-always-shown-as-built-after-acti.png) | [18](screenshots/kid-home/desktop-light/18-medication-link-always-shown-as-built-after-acti.png) | [15](screenshots/kid-home/desktop-dark/15-medication-link-always-shown-as-built-after-acti.png) | dup | [14](screenshots/kid-home/wide-dark/14-medication-link-always-shown-as-built-after-acti.png) |
| sheet Switch child | sheet | [13](screenshots/kid-home/phone-light/13-sheet-switch-child.png) | [15](screenshots/kid-home/phone-dark/15-sheet-switch-child.png) | [19](screenshots/kid-home/desktop-light/19-sheet-switch-child.png) | [16](screenshots/kid-home/desktop-dark/16-sheet-switch-child.png) | [13](screenshots/kid-home/wide-light/13-sheet-switch-child.png) | [15](screenshots/kid-home/wide-dark/15-sheet-switch-child.png) |
| sheet More | sheet | [14](screenshots/kid-home/phone-light/14-sheet-more.png) | [16](screenshots/kid-home/phone-dark/16-sheet-more.png) | · | · | · | · |

## medication

Route in the app: `/kids/{kid}/medication`

| State | Kind | phone-light | phone-dark | desktop-light | desktop-dark | wide-light | wide-dark |
|---|---|---|---|---|---|---|---|
| default-frame | frame | [01](screenshots/medication/phone-light/01-default-frame.png) | [01](screenshots/medication/phone-dark/01-default-frame.png) | [01](screenshots/medication/desktop-light/01-default-frame.png) | [01](screenshots/medication/desktop-dark/01-default-frame.png) | [01](screenshots/medication/wide-light/01-default-frame.png) | [01](screenshots/medication/wide-dark/01-default-frame.png) |
| default-full-height | full | [02](screenshots/medication/phone-light/02-default-full-height.png) | [02](screenshots/medication/phone-dark/02-default-full-height.png) | [02](screenshots/medication/desktop-light/02-default-full-height.png) | [02](screenshots/medication/desktop-dark/02-default-full-height.png) | [02](screenshots/medication/wide-light/02-default-full-height.png) | [02](screenshots/medication/wide-dark/02-default-full-height.png) |
| NEXT SAVE Works | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Offline | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Error | state | dup | dup | dup | dup | dup | dup |
| CHILD Ivy | state | [03](screenshots/medication/phone-light/03-child-ivy.png) | [03](screenshots/medication/phone-dark/03-child-ivy.png) | [03](screenshots/medication/desktop-light/03-child-ivy.png) | [03](screenshots/medication/desktop-dark/03-child-ivy.png) | [03](screenshots/medication/wide-light/03-child-ivy.png) | [03](screenshots/medication/wide-dark/03-child-ivy.png) |
| CHILD Mia | state | [04](screenshots/medication/phone-light/04-child-mia.png) | [04](screenshots/medication/phone-dark/04-child-mia.png) | [04](screenshots/medication/desktop-light/04-child-mia.png) | [04](screenshots/medication/desktop-dark/04-child-mia.png) | [04](screenshots/medication/wide-light/04-child-mia.png) | [04](screenshots/medication/wide-dark/04-child-mia.png) |
| CHILD Leo | state | dup | dup | dup | [05](screenshots/medication/desktop-dark/05-child-leo.png) | dup | dup |
| Works | state | dup | dup | dup | dup | dup | dup |
| Course gone | state | dup | dup | dup | dup | dup | dup |
| Dose cap | state | dup | dup | dup | dup | dup | dup |
| As built (32 px) | state | dup | dup | dup | dup | dup | dup |
| Larger, one-hand | state | [05](screenshots/medication/phone-light/05-larger-one-hand.png) | [05](screenshots/medication/phone-dark/05-larger-one-hand.png) | [05](screenshots/medication/desktop-light/05-larger-one-hand.png) | [06](screenshots/medication/desktop-dark/06-larger-one-hand.png) | [05](screenshots/medication/wide-light/05-larger-one-hand.png) | [05](screenshots/medication/wide-dark/05-larger-one-hand.png) |
| ROLE Guardian | state | dup | dup | dup | dup | dup | [06](screenshots/medication/wide-dark/06-role-guardian.png) |
| ROLE Caregiver | state | [06](screenshots/medication/phone-light/06-role-caregiver.png) | [06](screenshots/medication/phone-dark/06-role-caregiver.png) | [06](screenshots/medication/desktop-light/06-role-caregiver.png) | [07](screenshots/medication/desktop-dark/07-role-caregiver.png) | [06](screenshots/medication/wide-light/06-role-caregiver.png) | [07](screenshots/medication/wide-dark/07-role-caregiver.png) |
| sheet Switch child | sheet | [07](screenshots/medication/phone-light/07-sheet-switch-child.png) | [07](screenshots/medication/phone-dark/07-sheet-switch-child.png) | [07](screenshots/medication/desktop-light/07-sheet-switch-child.png) | [08](screenshots/medication/desktop-dark/08-sheet-switch-child.png) | [07](screenshots/medication/wide-light/07-sheet-switch-child.png) | [08](screenshots/medication/wide-dark/08-sheet-switch-child.png) |
| sheet Add a course | sheet | [08](screenshots/medication/phone-light/08-sheet-add-a-course.png) | [08](screenshots/medication/phone-dark/08-sheet-add-a-course.png) | [08](screenshots/medication/desktop-light/08-sheet-add-a-course.png) | [09](screenshots/medication/desktop-dark/09-sheet-add-a-course.png) | [08](screenshots/medication/wide-light/08-sheet-add-a-course.png) | [09](screenshots/medication/wide-dark/09-sheet-add-a-course.png) |
| sheet Edit course | sheet | [09](screenshots/medication/phone-light/09-sheet-edit-course.png) | [09](screenshots/medication/phone-dark/09-sheet-edit-course.png) | [09](screenshots/medication/desktop-light/09-sheet-edit-course.png) | [10](screenshots/medication/desktop-dark/10-sheet-edit-course.png) | [09](screenshots/medication/wide-light/09-sheet-edit-course.png) | [10](screenshots/medication/wide-dark/10-sheet-edit-course.png) |
| sheet Edit dose | sheet | [10](screenshots/medication/phone-light/10-sheet-edit-dose.png) | [10](screenshots/medication/phone-dark/10-sheet-edit-dose.png) | [10](screenshots/medication/desktop-light/10-sheet-edit-dose.png) | [11](screenshots/medication/desktop-dark/11-sheet-edit-dose.png) | [10](screenshots/medication/wide-light/10-sheet-edit-dose.png) | [11](screenshots/medication/wide-dark/11-sheet-edit-dose.png) |
| sheet More | sheet | [11](screenshots/medication/phone-light/11-sheet-more.png) | [11](screenshots/medication/phone-dark/11-sheet-more.png) | · | · | · | · |

## milestone-form

Route in the app: `/kids/{kid}/milestones/new`

| State | Kind | phone-light | phone-dark | desktop-light | desktop-dark | wide-light | wide-dark |
|---|---|---|---|---|---|---|---|
| default-frame | frame | [01](screenshots/milestone-form/phone-light/01-default-frame.png) | [01](screenshots/milestone-form/phone-dark/01-default-frame.png) | [01](screenshots/milestone-form/desktop-light/01-default-frame.png) | [01](screenshots/milestone-form/desktop-dark/01-default-frame.png) | [01](screenshots/milestone-form/wide-light/01-default-frame.png) | [01](screenshots/milestone-form/wide-dark/01-default-frame.png) |
| NEXT SAVE Works | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Works after action | state-after-action | [02](screenshots/milestone-form/phone-light/02-next-save-works-after-action.png) | [02](screenshots/milestone-form/phone-dark/02-next-save-works-after-action.png) | [02](screenshots/milestone-form/desktop-light/02-next-save-works-after-action.png) | [02](screenshots/milestone-form/desktop-dark/02-next-save-works-after-action.png) | [02](screenshots/milestone-form/wide-light/02-next-save-works-after-action.png) | [02](screenshots/milestone-form/wide-dark/02-next-save-works-after-action.png) |
| NEXT SAVE Offline | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Offline after action | state-after-action | [03](screenshots/milestone-form/phone-light/03-next-save-offline-after-action.png) | [03](screenshots/milestone-form/phone-dark/03-next-save-offline-after-action.png) | [03](screenshots/milestone-form/desktop-light/03-next-save-offline-after-action.png) | [03](screenshots/milestone-form/desktop-dark/03-next-save-offline-after-action.png) | [03](screenshots/milestone-form/wide-light/03-next-save-offline-after-action.png) | [03](screenshots/milestone-form/wide-dark/03-next-save-offline-after-action.png) |
| NEXT SAVE Error | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Error after action | state-after-action | [04](screenshots/milestone-form/phone-light/04-next-save-error-after-action.png) | [04](screenshots/milestone-form/phone-dark/04-next-save-error-after-action.png) | [04](screenshots/milestone-form/desktop-light/04-next-save-error-after-action.png) | [04](screenshots/milestone-form/desktop-dark/04-next-save-error-after-action.png) | [04](screenshots/milestone-form/wide-light/04-next-save-error-after-action.png) | [04](screenshots/milestone-form/wide-dark/04-next-save-error-after-action.png) |
| SCREEN Mark as reached | state | dup | dup | dup | dup | dup | dup |
| SCREEN Add a milestone | state | [05](screenshots/milestone-form/phone-light/05-screen-add-a-milestone.png) | [05](screenshots/milestone-form/phone-dark/05-screen-add-a-milestone.png) | [05](screenshots/milestone-form/desktop-light/05-screen-add-a-milestone.png) | [05](screenshots/milestone-form/desktop-dark/05-screen-add-a-milestone.png) | [05](screenshots/milestone-form/wide-light/05-screen-add-a-milestone.png) | [05](screenshots/milestone-form/wide-dark/05-screen-add-a-milestone.png) |
| SCREEN Edit (catalog) | state | [06](screenshots/milestone-form/phone-light/06-screen-edit-catalog.png) | [06](screenshots/milestone-form/phone-dark/06-screen-edit-catalog.png) | [06](screenshots/milestone-form/desktop-light/06-screen-edit-catalog.png) | [06](screenshots/milestone-form/desktop-dark/06-screen-edit-catalog.png) | [06](screenshots/milestone-form/wide-light/06-screen-edit-catalog.png) | [06](screenshots/milestone-form/wide-dark/06-screen-edit-catalog.png) |
| SCREEN Edit (custom) | state | [07](screenshots/milestone-form/phone-light/07-screen-edit-custom.png) | [07](screenshots/milestone-form/phone-dark/07-screen-edit-custom.png) | [07](screenshots/milestone-form/desktop-light/07-screen-edit-custom.png) | [07](screenshots/milestone-form/desktop-dark/07-screen-edit-custom.png) | [07](screenshots/milestone-form/wide-light/07-screen-edit-custom.png) | [07](screenshots/milestone-form/wide-dark/07-screen-edit-custom.png) |
| UNDO / DELETE CONFIRM Closed | state | dup | dup | dup | dup | dup | dup |
| UNDO / DELETE CONFIRM Open | state | dup | dup | dup | dup | dup | dup |
| UNDO / DELETE CONFIRM Failed | state | dup | dup | dup | dup | dup | dup |
| ROLE Guardian | state | dup | dup | dup | dup | dup | dup |
| ROLE Caregiver | state | [08](screenshots/milestone-form/phone-light/08-role-caregiver.png) | [08](screenshots/milestone-form/phone-dark/08-role-caregiver.png) | [08](screenshots/milestone-form/desktop-light/08-role-caregiver.png) | [08](screenshots/milestone-form/desktop-dark/08-role-caregiver.png) | [08](screenshots/milestone-form/wide-light/08-role-caregiver.png) | [08](screenshots/milestone-form/wide-dark/08-role-caregiver.png) |
| sheet More | sheet | [09](screenshots/milestone-form/phone-light/09-sheet-more.png) | [09](screenshots/milestone-form/phone-dark/09-sheet-more.png) | · | · | · | · |

## milestones

Route in the app: `/kids/{kid}/milestones`

| State | Kind | phone-light | phone-dark | desktop-light | desktop-dark | wide-light | wide-dark |
|---|---|---|---|---|---|---|---|
| default-frame | frame | [01](screenshots/milestones/phone-light/01-default-frame.png) | [01](screenshots/milestones/phone-dark/01-default-frame.png) | [01](screenshots/milestones/desktop-light/01-default-frame.png) | [01](screenshots/milestones/desktop-dark/01-default-frame.png) | [01](screenshots/milestones/wide-light/01-default-frame.png) | [01](screenshots/milestones/wide-dark/01-default-frame.png) |
| default-full-height | full | [02](screenshots/milestones/phone-light/02-default-full-height.png) | [02](screenshots/milestones/phone-dark/02-default-full-height.png) | [02](screenshots/milestones/desktop-light/02-default-full-height.png) | [02](screenshots/milestones/desktop-dark/02-default-full-height.png) | [02](screenshots/milestones/wide-light/02-default-full-height.png) | [02](screenshots/milestones/wide-dark/02-default-full-height.png) |
| NEXT SAVE Works | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Offline | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Error | state | dup | dup | dup | dup | dup | dup |
| CHILD (SAMPLE SCENARIO) Stored birth date | state | dup | dup | dup | dup | dup | dup |
| CHILD (SAMPLE SCENARIO) 9 months | state | [03](screenshots/milestones/phone-light/03-child-sample-scenario-9-months.png) | [03](screenshots/milestones/phone-dark/03-child-sample-scenario-9-months.png) | [03](screenshots/milestones/desktop-light/03-child-sample-scenario-9-months.png) | [03](screenshots/milestones/desktop-dark/03-child-sample-scenario-9-months.png) | [03](screenshots/milestones/wide-light/03-child-sample-scenario-9-months.png) | [03](screenshots/milestones/wide-dark/03-child-sample-scenario-9-months.png) |
| CHILD (SAMPLE SCENARIO) Under 2 months | state | [04](screenshots/milestones/phone-light/04-child-sample-scenario-under-2-months.png) | [04](screenshots/milestones/phone-dark/04-child-sample-scenario-under-2-months.png) | [04](screenshots/milestones/desktop-light/04-child-sample-scenario-under-2-months.png) | [04](screenshots/milestones/desktop-dark/04-child-sample-scenario-under-2-months.png) | [04](screenshots/milestones/wide-light/04-child-sample-scenario-under-2-months.png) | [04](screenshots/milestones/wide-dark/04-child-sample-scenario-under-2-months.png) |
| CHILD (SAMPLE SCENARIO) Preterm (32w3d) | state | [05](screenshots/milestones/phone-light/05-child-sample-scenario-preterm-32w3d.png) | [05](screenshots/milestones/phone-dark/05-child-sample-scenario-preterm-32w3d.png) | [05](screenshots/milestones/desktop-light/05-child-sample-scenario-preterm-32w3d.png) | [05](screenshots/milestones/desktop-dark/05-child-sample-scenario-preterm-32w3d.png) | [05](screenshots/milestones/wide-light/05-child-sample-scenario-preterm-32w3d.png) | [05](screenshots/milestones/wide-dark/05-child-sample-scenario-preterm-32w3d.png) |
| CHILD (SAMPLE SCENARIO) 4 years 11 months | state | [06](screenshots/milestones/phone-light/06-child-sample-scenario-4-years-11-months.png) | [06](screenshots/milestones/phone-dark/06-child-sample-scenario-4-years-11-months.png) | [06](screenshots/milestones/desktop-light/06-child-sample-scenario-4-years-11-months.png) | [06](screenshots/milestones/desktop-dark/06-child-sample-scenario-4-years-11-months.png) | [06](screenshots/milestones/wide-light/06-child-sample-scenario-4-years-11-months.png) | [06](screenshots/milestones/wide-dark/06-child-sample-scenario-4-years-11-months.png) |
| CHILD (SAMPLE SCENARIO) After 5th birthday | state | [07](screenshots/milestones/phone-light/07-child-sample-scenario-after-5th-birthday.png) | [07](screenshots/milestones/phone-dark/07-child-sample-scenario-after-5th-birthday.png) | [07](screenshots/milestones/desktop-light/07-child-sample-scenario-after-5th-birthday.png) | [07](screenshots/milestones/desktop-dark/07-child-sample-scenario-after-5th-birthday.png) | [07](screenshots/milestones/wide-light/07-child-sample-scenario-after-5th-birthday.png) | [07](screenshots/milestones/wide-dark/07-child-sample-scenario-after-5th-birthday.png) |
| CHILD (SAMPLE SCENARIO) 30 months (module off) | state | [08](screenshots/milestones/phone-light/08-child-sample-scenario-30-months-module-off.png) | [08](screenshots/milestones/phone-dark/08-child-sample-scenario-30-months-module-off.png) | [08](screenshots/milestones/desktop-light/08-child-sample-scenario-30-months-module-off.png) | [08](screenshots/milestones/desktop-dark/08-child-sample-scenario-30-months-module-off.png) | [08](screenshots/milestones/wide-light/08-child-sample-scenario-30-months-module-off.png) | [08](screenshots/milestones/wide-dark/08-child-sample-scenario-30-months-module-off.png) |
| MILESTONES MODULE On | state | dup | dup | dup | dup | dup | dup |
| MILESTONES MODULE Off | state | [09](screenshots/milestones/phone-light/09-milestones-module-off.png) | [09](screenshots/milestones/phone-dark/09-milestones-module-off.png) | [09](screenshots/milestones/desktop-light/09-milestones-module-off.png) | [09](screenshots/milestones/desktop-dark/09-milestones-module-off.png) | [09](screenshots/milestones/wide-light/09-milestones-module-off.png) | [09](screenshots/milestones/wide-dark/09-milestones-module-off.png) |
| NEXT ONE-TAP (SIMULATED) Works | state | dup | dup | dup | dup | dup | dup |
| NEXT ONE-TAP (SIMULATED) Already marked | state | dup | dup | dup | dup | dup | dup |
| NEXT ONE-TAP (SIMULATED) Cap reached | state | dup | dup | dup | dup | dup | dup |
| YOUR MILESTONES Filled | state | dup | dup | dup | dup | dup | dup |
| YOUR MILESTONES Empty | state | [10](screenshots/milestones/phone-light/10-your-milestones-empty.png) | [10](screenshots/milestones/phone-dark/10-your-milestones-empty.png) | [10](screenshots/milestones/desktop-light/10-your-milestones-empty.png) | [10](screenshots/milestones/desktop-dark/10-your-milestones-empty.png) | [10](screenshots/milestones/wide-light/10-your-milestones-empty.png) | [10](screenshots/milestones/wide-dark/10-your-milestones-empty.png) |
| CATALOG UPDATE No retired row | state | dup | dup | dup | dup | dup | dup |
| CATALOG UPDATE One retired row | state | [11](screenshots/milestones/phone-light/11-catalog-update-one-retired-row.png) | [11](screenshots/milestones/phone-dark/11-catalog-update-one-retired-row.png) | [11](screenshots/milestones/desktop-light/11-catalog-update-one-retired-row.png) | [11](screenshots/milestones/desktop-dark/11-catalog-update-one-retired-row.png) | [11](screenshots/milestones/wide-light/11-catalog-update-one-retired-row.png) | [11](screenshots/milestones/wide-dark/11-catalog-update-one-retired-row.png) |
| CHOOSE A DATE Link to a page (as built) | state | dup | dup | dup | dup | dup | dup |
| CHOOSE A DATE Reveal in the row (proposal) | state | [12](screenshots/milestones/phone-light/12-choose-a-date-reveal-in-the-row-proposal.png) | [12](screenshots/milestones/phone-dark/12-choose-a-date-reveal-in-the-row-proposal.png) | [12](screenshots/milestones/desktop-light/12-choose-a-date-reveal-in-the-row-proposal.png) | [12](screenshots/milestones/desktop-dark/12-choose-a-date-reveal-in-the-row-proposal.png) | [12](screenshots/milestones/wide-light/12-choose-a-date-reveal-in-the-row-proposal.png) | [12](screenshots/milestones/wide-dark/12-choose-a-date-reveal-in-the-row-proposal.png) |
| PAGE STATE Normal | state | dup | dup | dup | dup | dup | dup |
| PAGE STATE Server down | state | [13](screenshots/milestones/phone-light/13-page-state-server-down.png) | [13](screenshots/milestones/phone-dark/13-page-state-server-down.png) | [13](screenshots/milestones/desktop-light/13-page-state-server-down.png) | [13](screenshots/milestones/desktop-dark/13-page-state-server-down.png) | [13](screenshots/milestones/wide-light/13-page-state-server-down.png) | [13](screenshots/milestones/wide-dark/13-page-state-server-down.png) |
| PAGE STATE Not found | state | [14](screenshots/milestones/phone-light/14-page-state-not-found.png) | [14](screenshots/milestones/phone-dark/14-page-state-not-found.png) | [14](screenshots/milestones/desktop-light/14-page-state-not-found.png) | [14](screenshots/milestones/desktop-dark/14-page-state-not-found.png) | [14](screenshots/milestones/wide-light/14-page-state-not-found.png) | [14](screenshots/milestones/wide-dark/14-page-state-not-found.png) |
| MARKS Reset marks | state | dup | dup | dup | dup | dup | dup |
| ROLE Guardian | state | dup | dup | dup | dup | dup | dup |
| ROLE Caregiver | state | [15](screenshots/milestones/phone-light/15-role-caregiver.png) | [15](screenshots/milestones/phone-dark/15-role-caregiver.png) | [15](screenshots/milestones/desktop-light/15-role-caregiver.png) | [15](screenshots/milestones/desktop-dark/15-role-caregiver.png) | [15](screenshots/milestones/wide-light/15-role-caregiver.png) | [15](screenshots/milestones/wide-dark/15-role-caregiver.png) |
| sheet Switch child | sheet | [16](screenshots/milestones/phone-light/16-sheet-switch-child.png) | [16](screenshots/milestones/phone-dark/16-sheet-switch-child.png) | [16](screenshots/milestones/desktop-light/16-sheet-switch-child.png) | [16](screenshots/milestones/desktop-dark/16-sheet-switch-child.png) | [16](screenshots/milestones/wide-light/16-sheet-switch-child.png) | [16](screenshots/milestones/wide-dark/16-sheet-switch-child.png) |
| sheet Choose a date | sheet | [17](screenshots/milestones/phone-light/17-sheet-choose-a-date.png) | [17](screenshots/milestones/phone-dark/17-sheet-choose-a-date.png) | [17](screenshots/milestones/desktop-light/17-sheet-choose-a-date.png) | [17](screenshots/milestones/desktop-dark/17-sheet-choose-a-date.png) | [17](screenshots/milestones/wide-light/17-sheet-choose-a-date.png) | [17](screenshots/milestones/wide-dark/17-sheet-choose-a-date.png) |
| sheet Edit milestone | sheet | [18](screenshots/milestones/phone-light/18-sheet-edit-milestone.png) | [18](screenshots/milestones/phone-dark/18-sheet-edit-milestone.png) | [18](screenshots/milestones/desktop-light/18-sheet-edit-milestone.png) | [18](screenshots/milestones/desktop-dark/18-sheet-edit-milestone.png) | [18](screenshots/milestones/wide-light/18-sheet-edit-milestone.png) | [18](screenshots/milestones/wide-dark/18-sheet-edit-milestone.png) |
| sheet Add a milestone | sheet | [19](screenshots/milestones/phone-light/19-sheet-add-a-milestone.png) | [19](screenshots/milestones/phone-dark/19-sheet-add-a-milestone.png) | [19](screenshots/milestones/desktop-light/19-sheet-add-a-milestone.png) | [19](screenshots/milestones/desktop-dark/19-sheet-add-a-milestone.png) | [19](screenshots/milestones/wide-light/19-sheet-add-a-milestone.png) | [19](screenshots/milestones/wide-dark/19-sheet-add-a-milestone.png) |
| sheet More | sheet | [20](screenshots/milestones/phone-light/20-sheet-more.png) | [20](screenshots/milestones/phone-dark/20-sheet-more.png) | · | · | · | · |

## my-profile

Route in the app: `/account`

| State | Kind | phone-light | phone-dark | desktop-light | desktop-dark | wide-light | wide-dark |
|---|---|---|---|---|---|---|---|
| default-frame | frame | [01](screenshots/my-profile/phone-light/01-default-frame.png) | [01](screenshots/my-profile/phone-dark/01-default-frame.png) | [01](screenshots/my-profile/desktop-light/01-default-frame.png) | [01](screenshots/my-profile/desktop-dark/01-default-frame.png) | [01](screenshots/my-profile/wide-light/01-default-frame.png) | [01](screenshots/my-profile/wide-dark/01-default-frame.png) |
| default-full-height | full | [02](screenshots/my-profile/phone-light/02-default-full-height.png) | [02](screenshots/my-profile/phone-dark/02-default-full-height.png) | [02](screenshots/my-profile/desktop-light/02-default-full-height.png) | [02](screenshots/my-profile/desktop-dark/02-default-full-height.png) | [02](screenshots/my-profile/wide-light/02-default-full-height.png) | [02](screenshots/my-profile/wide-dark/02-default-full-height.png) |
| NEXT SAVE Works | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Offline | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Error | state | dup | dup | dup | dup | dup | dup |
| ROLE Guardian | state | dup | dup | dup | dup | dup | dup |
| ROLE Caregiver | state | [03](screenshots/my-profile/phone-light/03-role-caregiver.png) | [03](screenshots/my-profile/phone-dark/03-role-caregiver.png) | [03](screenshots/my-profile/desktop-light/03-role-caregiver.png) | [03](screenshots/my-profile/desktop-dark/03-role-caregiver.png) | [03](screenshots/my-profile/wide-light/03-role-caregiver.png) | [03](screenshots/my-profile/wide-dark/03-role-caregiver.png) |
| sheet More | sheet | [04](screenshots/my-profile/phone-light/04-sheet-more.png) | [04](screenshots/my-profile/phone-dark/04-sheet-more.png) | · | · | · | · |

## notifications

Route in the app: `/account/notifications`

| State | Kind | phone-light | phone-dark | desktop-light | desktop-dark | wide-light | wide-dark |
|---|---|---|---|---|---|---|---|
| default-frame | frame | [01](screenshots/notifications/phone-light/01-default-frame.png) | [01](screenshots/notifications/phone-dark/01-default-frame.png) | [01](screenshots/notifications/desktop-light/01-default-frame.png) | [01](screenshots/notifications/desktop-dark/01-default-frame.png) | [01](screenshots/notifications/wide-light/01-default-frame.png) | [01](screenshots/notifications/wide-dark/01-default-frame.png) |
| NEXT SAVE Works | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Offline | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Error | state | dup | dup | dup | dup | dup | dup |
| CHANNELS ON THIS SERVER None | state | [02](screenshots/notifications/phone-light/02-channels-on-this-server-none.png) | [02](screenshots/notifications/phone-dark/02-channels-on-this-server-none.png) | [02](screenshots/notifications/desktop-light/02-channels-on-this-server-none.png) | [02](screenshots/notifications/desktop-dark/02-channels-on-this-server-none.png) | [02](screenshots/notifications/wide-light/02-channels-on-this-server-none.png) | [02](screenshots/notifications/wide-dark/02-channels-on-this-server-none.png) |
| CHANNELS ON THIS SERVER Web Push | state | [03](screenshots/notifications/phone-light/03-channels-on-this-server-web-push.png) | [03](screenshots/notifications/phone-dark/03-channels-on-this-server-web-push.png) | [03](screenshots/notifications/desktop-light/03-channels-on-this-server-web-push.png) | [03](screenshots/notifications/desktop-dark/03-channels-on-this-server-web-push.png) | [03](screenshots/notifications/wide-light/03-channels-on-this-server-web-push.png) | [03](screenshots/notifications/wide-dark/03-channels-on-this-server-web-push.png) |
| CHANNELS ON THIS SERVER + ntfy | state | dup | dup | dup | dup | dup | dup |
| THIS BROWSER Push OK | state | dup | dup | dup | dup | dup | dup |
| THIS BROWSER iPhone Safari | state | [04](screenshots/notifications/phone-light/04-this-browser-iphone-safari.png) | [04](screenshots/notifications/phone-dark/04-this-browser-iphone-safari.png) | [04](screenshots/notifications/desktop-light/04-this-browser-iphone-safari.png) | [04](screenshots/notifications/desktop-dark/04-this-browser-iphone-safari.png) | [04](screenshots/notifications/wide-light/04-this-browser-iphone-safari.png) | [04](screenshots/notifications/wide-dark/04-this-browser-iphone-safari.png) |
| THIS BROWSER No push | state | [05](screenshots/notifications/phone-light/05-this-browser-no-push.png) | [05](screenshots/notifications/phone-dark/05-this-browser-no-push.png) | [05](screenshots/notifications/desktop-light/05-this-browser-no-push.png) | [05](screenshots/notifications/desktop-dark/05-this-browser-no-push.png) | [05](screenshots/notifications/wide-light/05-this-browser-no-push.png) | [05](screenshots/notifications/wide-dark/05-this-browser-no-push.png) |
| THIS BROWSER JS off | state | [06](screenshots/notifications/phone-light/06-this-browser-js-off.png) | [06](screenshots/notifications/phone-dark/06-this-browser-js-off.png) | [06](screenshots/notifications/desktop-light/06-this-browser-js-off.png) | [06](screenshots/notifications/desktop-dark/06-this-browser-js-off.png) | [06](screenshots/notifications/wide-light/06-this-browser-js-off.png) | [06](screenshots/notifications/wide-dark/06-this-browser-js-off.png) |
| BROWSER PROMPT ANSWER Allow | state | dup | dup | dup | dup | dup | dup |
| BROWSER PROMPT ANSWER Block | state | dup | dup | dup | dup | dup | dup |
| BROWSER PROMPT ANSWER Dismiss | state | dup | dup | dup | dup | dup | dup |
| BROWSER PROMPT ANSWER Error | state | dup | dup | dup | dup | dup | dup |
| As built | state | dup | dup | dup | dup | dup | dup |
| One-line instruction | state | dup | dup | dup | dup | dup | dup |
| SIMULATE Ana enables here | state | dup | dup | dup | dup | dup | dup |
| SIMULATE Key rotated | state | [07](screenshots/notifications/phone-light/07-simulate-key-rotated.png) | [07](screenshots/notifications/phone-dark/07-simulate-key-rotated.png) | [07](screenshots/notifications/desktop-light/07-simulate-key-rotated.png) | [07](screenshots/notifications/desktop-dark/07-simulate-key-rotated.png) | [07](screenshots/notifications/wide-light/07-simulate-key-rotated.png) | [07](screenshots/notifications/wide-dark/07-simulate-key-rotated.png) |
| ROLE Guardian | state | dup | dup | dup | dup | dup | dup |
| ROLE Caregiver | state | [08](screenshots/notifications/phone-light/08-role-caregiver.png) | [08](screenshots/notifications/phone-dark/08-role-caregiver.png) | [08](screenshots/notifications/desktop-light/08-role-caregiver.png) | [08](screenshots/notifications/desktop-dark/08-role-caregiver.png) | [08](screenshots/notifications/wide-light/08-role-caregiver.png) | [08](screenshots/notifications/wide-dark/08-role-caregiver.png) |
| sheet More | sheet | [09](screenshots/notifications/phone-light/09-sheet-more.png) | [09](screenshots/notifications/phone-dark/09-sheet-more.png) | · | · | · | · |

## pdf-export

Route in the app: `/kids/{kid}/export/pdf`

| State | Kind | phone-light | phone-dark | desktop-light | desktop-dark | wide-light | wide-dark |
|---|---|---|---|---|---|---|---|
| sheet-1 | paper | [01](screenshots/pdf-export/phone-light/01-sheet-1.png) | [01](screenshots/pdf-export/phone-dark/01-sheet-1.png) | [01](screenshots/pdf-export/desktop-light/01-sheet-1.png) | [01](screenshots/pdf-export/desktop-dark/01-sheet-1.png) | [01](screenshots/pdf-export/wide-light/01-sheet-1.png) | [01](screenshots/pdf-export/wide-dark/01-sheet-1.png) |
| sheet-2 | paper | [02](screenshots/pdf-export/phone-light/02-sheet-2.png) | [02](screenshots/pdf-export/phone-dark/02-sheet-2.png) | [02](screenshots/pdf-export/desktop-light/02-sheet-2.png) | [02](screenshots/pdf-export/desktop-dark/02-sheet-2.png) | [02](screenshots/pdf-export/wide-light/02-sheet-2.png) | [02](screenshots/pdf-export/wide-dark/02-sheet-2.png) |
| sheet-3 | paper | [03](screenshots/pdf-export/phone-light/03-sheet-3.png) | [03](screenshots/pdf-export/phone-dark/03-sheet-3.png) | [03](screenshots/pdf-export/desktop-light/03-sheet-3.png) | [03](screenshots/pdf-export/desktop-dark/03-sheet-3.png) | [03](screenshots/pdf-export/wide-light/03-sheet-3.png) | [03](screenshots/pdf-export/wide-dark/03-sheet-3.png) |
| sheet-4 | paper | [04](screenshots/pdf-export/phone-light/04-sheet-4.png) | [04](screenshots/pdf-export/phone-dark/04-sheet-4.png) | [04](screenshots/pdf-export/desktop-light/04-sheet-4.png) | [04](screenshots/pdf-export/desktop-dark/04-sheet-4.png) | [04](screenshots/pdf-export/wide-light/04-sheet-4.png) | [04](screenshots/pdf-export/wide-dark/04-sheet-4.png) |
| sheet-5 | paper | [05](screenshots/pdf-export/phone-light/05-sheet-5.png) | [05](screenshots/pdf-export/phone-dark/05-sheet-5.png) | [05](screenshots/pdf-export/desktop-light/05-sheet-5.png) | [05](screenshots/pdf-export/desktop-dark/05-sheet-5.png) | [05](screenshots/pdf-export/wide-light/05-sheet-5.png) | [05](screenshots/pdf-export/wide-dark/05-sheet-5.png) |
| sheet-6 | paper | [06](screenshots/pdf-export/phone-light/06-sheet-6.png) | [06](screenshots/pdf-export/phone-dark/06-sheet-6.png) | [06](screenshots/pdf-export/desktop-light/06-sheet-6.png) | [06](screenshots/pdf-export/desktop-dark/06-sheet-6.png) | [06](screenshots/pdf-export/wide-light/06-sheet-6.png) | [06](screenshots/pdf-export/wide-dark/06-sheet-6.png) |
| sheet-7 | paper | [07](screenshots/pdf-export/phone-light/07-sheet-7.png) | [07](screenshots/pdf-export/phone-dark/07-sheet-7.png) | [07](screenshots/pdf-export/desktop-light/07-sheet-7.png) | [07](screenshots/pdf-export/desktop-dark/07-sheet-7.png) | [07](screenshots/pdf-export/wide-light/07-sheet-7.png) | [07](screenshots/pdf-export/wide-dark/07-sheet-7.png) |

## set-password

Route in the app: `/invite/{token}`

| State | Kind | phone-light | phone-dark | desktop-light | desktop-dark | wide-light | wide-dark |
|---|---|---|---|---|---|---|---|
| default-frame | frame | [01](screenshots/set-password/phone-light/01-default-frame.png) | [01](screenshots/set-password/phone-dark/01-default-frame.png) | [01](screenshots/set-password/desktop-light/01-default-frame.png) | [01](screenshots/set-password/desktop-dark/01-default-frame.png) | [01](screenshots/set-password/wide-light/01-default-frame.png) | [01](screenshots/set-password/wide-dark/01-default-frame.png) |
| LINK Reset | state | [02](screenshots/set-password/phone-light/02-link-reset.png) | [02](screenshots/set-password/phone-dark/02-link-reset.png) | [02](screenshots/set-password/desktop-light/02-link-reset.png) | [02](screenshots/set-password/desktop-dark/02-link-reset.png) | [02](screenshots/set-password/wide-light/02-link-reset.png) | [02](screenshots/set-password/wide-dark/02-link-reset.png) |
| LINK Reset after action | state-after-action | [03](screenshots/set-password/phone-light/03-link-reset-after-action.png) | [03](screenshots/set-password/phone-dark/03-link-reset-after-action.png) | [03](screenshots/set-password/desktop-light/03-link-reset-after-action.png) | [03](screenshots/set-password/desktop-dark/03-link-reset-after-action.png) | [03](screenshots/set-password/wide-light/03-link-reset-after-action.png) | [03](screenshots/set-password/wide-dark/03-link-reset-after-action.png) |
| LINK STATE Used or expired | state | [04](screenshots/set-password/phone-light/04-link-state-used-or-expired.png) | [04](screenshots/set-password/phone-dark/04-link-state-used-or-expired.png) | [04](screenshots/set-password/desktop-light/04-link-state-used-or-expired.png) | [04](screenshots/set-password/desktop-dark/04-link-state-used-or-expired.png) | [04](screenshots/set-password/wide-light/04-link-state-used-or-expired.png) | [04](screenshots/set-password/wide-dark/04-link-state-used-or-expired.png) |
| NEXT SAVE Rate limited | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Rate limited after action | state-after-action | [05](screenshots/set-password/phone-light/05-next-save-rate-limited-after-action.png) | [05](screenshots/set-password/phone-dark/05-next-save-rate-limited-after-action.png) | [05](screenshots/set-password/desktop-light/05-next-save-rate-limited-after-action.png) | [05](screenshots/set-password/desktop-dark/05-next-save-rate-limited-after-action.png) | [05](screenshots/set-password/wide-light/05-next-save-rate-limited-after-action.png) | [05](screenshots/set-password/wide-dark/05-next-save-rate-limited-after-action.png) |
| short password | flow | [06](screenshots/set-password/phone-light/06-short-password.png) | [06](screenshots/set-password/phone-dark/06-short-password.png) | [06](screenshots/set-password/desktop-light/06-short-password.png) | [06](screenshots/set-password/desktop-dark/06-short-password.png) | [06](screenshots/set-password/wide-light/06-short-password.png) | [06](screenshots/set-password/wide-dark/06-short-password.png) |
| mismatch | flow | [07](screenshots/set-password/phone-light/07-mismatch.png) | [07](screenshots/set-password/phone-dark/07-mismatch.png) | [07](screenshots/set-password/desktop-light/07-mismatch.png) | [07](screenshots/set-password/desktop-dark/07-mismatch.png) | [07](screenshots/set-password/wide-light/07-mismatch.png) | [07](screenshots/set-password/wide-dark/07-mismatch.png) |
| password set | flow | [08](screenshots/set-password/phone-light/08-password-set.png) | [08](screenshots/set-password/phone-dark/08-password-set.png) | [08](screenshots/set-password/desktop-light/08-password-set.png) | [08](screenshots/set-password/desktop-dark/08-password-set.png) | [08](screenshots/set-password/wide-light/08-password-set.png) | [08](screenshots/set-password/wide-dark/08-password-set.png) |

## sign-in

Route in the app: `/login`

| State | Kind | phone-light | phone-dark | desktop-light | desktop-dark | wide-light | wide-dark |
|---|---|---|---|---|---|---|---|
| default-frame | frame | [01](screenshots/sign-in/phone-light/01-default-frame.png) | [01](screenshots/sign-in/phone-dark/01-default-frame.png) | [01](screenshots/sign-in/desktop-light/01-default-frame.png) | [01](screenshots/sign-in/desktop-dark/01-default-frame.png) | [01](screenshots/sign-in/wide-light/01-default-frame.png) | [01](screenshots/sign-in/wide-dark/01-default-frame.png) |
| NEXT SIGN-IN Wrong password | state | dup | dup | dup | dup | dup | dup |
| NEXT SIGN-IN Wrong password after action | state-after-action | [02](screenshots/sign-in/phone-light/02-next-sign-in-wrong-password-after-action.png) | [02](screenshots/sign-in/phone-dark/02-next-sign-in-wrong-password-after-action.png) | [02](screenshots/sign-in/desktop-light/02-next-sign-in-wrong-password-after-action.png) | [02](screenshots/sign-in/desktop-dark/02-next-sign-in-wrong-password-after-action.png) | [02](screenshots/sign-in/wide-light/02-next-sign-in-wrong-password-after-action.png) | [02](screenshots/sign-in/wide-dark/02-next-sign-in-wrong-password-after-action.png) |
| NEXT SIGN-IN Rate limited | state | dup | dup | dup | dup | dup | dup |
| NEXT SIGN-IN Rate limited after action | state-after-action | [03](screenshots/sign-in/phone-light/03-next-sign-in-rate-limited-after-action.png) | [03](screenshots/sign-in/phone-dark/03-next-sign-in-rate-limited-after-action.png) | [03](screenshots/sign-in/desktop-light/03-next-sign-in-rate-limited-after-action.png) | [03](screenshots/sign-in/desktop-dark/03-next-sign-in-rate-limited-after-action.png) | [03](screenshots/sign-in/wide-light/03-next-sign-in-rate-limited-after-action.png) | [03](screenshots/sign-in/wide-dark/03-next-sign-in-rate-limited-after-action.png) |
| NEXT SIGN-IN API down | state | dup | dup | dup | dup | dup | dup |
| NEXT SIGN-IN API down after action | state-after-action | [04](screenshots/sign-in/phone-light/04-next-sign-in-api-down-after-action.png) | [04](screenshots/sign-in/phone-dark/04-next-sign-in-api-down-after-action.png) | [04](screenshots/sign-in/desktop-light/04-next-sign-in-api-down-after-action.png) | [04](screenshots/sign-in/desktop-dark/04-next-sign-in-api-down-after-action.png) | [04](screenshots/sign-in/wide-light/04-next-sign-in-api-down-after-action.png) | [04](screenshots/sign-in/wide-dark/04-next-sign-in-api-down-after-action.png) |
| blank submit | flow | [05](screenshots/sign-in/phone-light/05-blank-submit.png) | [05](screenshots/sign-in/phone-dark/05-blank-submit.png) | [05](screenshots/sign-in/desktop-light/05-blank-submit.png) | [05](screenshots/sign-in/desktop-dark/05-blank-submit.png) | [05](screenshots/sign-in/wide-light/05-blank-submit.png) | [05](screenshots/sign-in/wide-dark/05-blank-submit.png) |
| signed in | flow | [06](screenshots/sign-in/phone-light/06-signed-in.png) | [06](screenshots/sign-in/phone-dark/06-signed-in.png) | [06](screenshots/sign-in/desktop-light/06-signed-in.png) | [06](screenshots/sign-in/desktop-dark/06-signed-in.png) | [06](screenshots/sign-in/wide-light/06-signed-in.png) | [06](screenshots/sign-in/wide-dark/06-signed-in.png) |

## symptom

Route in the app: `/kids/{kid}/log/symptom`

| State | Kind | phone-light | phone-dark | desktop-light | desktop-dark | wide-light | wide-dark |
|---|---|---|---|---|---|---|---|
| default-frame | frame | [01](screenshots/symptom/phone-light/01-default-frame.png) | [01](screenshots/symptom/phone-dark/01-default-frame.png) | [01](screenshots/symptom/desktop-light/01-default-frame.png) | [01](screenshots/symptom/desktop-dark/01-default-frame.png) | [01](screenshots/symptom/wide-light/01-default-frame.png) | [01](screenshots/symptom/wide-dark/01-default-frame.png) |
| default-full-height | full | [02](screenshots/symptom/phone-light/02-default-full-height.png) | [02](screenshots/symptom/phone-dark/02-default-full-height.png) | · | · | · | · |
| NEXT SAVE Works | state | dup | dup | dup | dup | [02](screenshots/symptom/wide-light/02-next-save-works.png) | dup |
| NEXT SAVE Works after action | state-after-action | [03](screenshots/symptom/phone-light/03-next-save-works-after-action.png) | [03](screenshots/symptom/phone-dark/03-next-save-works-after-action.png) | [02](screenshots/symptom/desktop-light/02-next-save-works-after-action.png) | [02](screenshots/symptom/desktop-dark/02-next-save-works-after-action.png) | [03](screenshots/symptom/wide-light/03-next-save-works-after-action.png) | [02](screenshots/symptom/wide-dark/02-next-save-works-after-action.png) |
| NEXT SAVE Offline | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Offline after action | state-after-action | [04](screenshots/symptom/phone-light/04-next-save-offline-after-action.png) | [04](screenshots/symptom/phone-dark/04-next-save-offline-after-action.png) | [03](screenshots/symptom/desktop-light/03-next-save-offline-after-action.png) | [03](screenshots/symptom/desktop-dark/03-next-save-offline-after-action.png) | [04](screenshots/symptom/wide-light/04-next-save-offline-after-action.png) | [03](screenshots/symptom/wide-dark/03-next-save-offline-after-action.png) |
| NEXT SAVE Error | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Error after action | state-after-action | [05](screenshots/symptom/phone-light/05-next-save-error-after-action.png) | [05](screenshots/symptom/phone-dark/05-next-save-error-after-action.png) | [04](screenshots/symptom/desktop-light/04-next-save-error-after-action.png) | [04](screenshots/symptom/desktop-dark/04-next-save-error-after-action.png) | [05](screenshots/symptom/wide-light/05-next-save-error-after-action.png) | [04](screenshots/symptom/wide-dark/04-next-save-error-after-action.png) |
| SCREEN New | state | dup | dup | dup | dup | dup | dup |
| SCREEN Edit | state | [06](screenshots/symptom/phone-light/06-screen-edit.png) | [06](screenshots/symptom/phone-dark/06-screen-edit.png) | [05](screenshots/symptom/desktop-light/05-screen-edit.png) | [05](screenshots/symptom/desktop-dark/05-screen-edit.png) | [06](screenshots/symptom/wide-light/06-screen-edit.png) | [05](screenshots/symptom/wide-dark/05-screen-edit.png) |
| MY TEMPERATURE UNIT °C | state | dup | dup | dup | dup | dup | [06](screenshots/symptom/wide-dark/06-my-temperature-unit-c.png) |
| MY TEMPERATURE UNIT °F | state | dup | dup | dup | dup | dup | dup |
| ROLE Guardian | state | dup | dup | [06](screenshots/symptom/desktop-light/06-role-guardian.png) | dup | dup | dup |
| ROLE Caregiver | state | [07](screenshots/symptom/phone-light/07-role-caregiver.png) | [07](screenshots/symptom/phone-dark/07-role-caregiver.png) | [07](screenshots/symptom/desktop-light/07-role-caregiver.png) | [06](screenshots/symptom/desktop-dark/06-role-caregiver.png) | [07](screenshots/symptom/wide-light/07-role-caregiver.png) | [07](screenshots/symptom/wide-dark/07-role-caregiver.png) |
| sheet More | sheet | [08](screenshots/symptom/phone-light/08-sheet-more.png) | [08](screenshots/symptom/phone-dark/08-sheet-more.png) | · | · | · | · |

## timeline

Route in the app: `/kids/{kid}/timeline`

| State | Kind | phone-light | phone-dark | desktop-light | desktop-dark | wide-light | wide-dark |
|---|---|---|---|---|---|---|---|
| default-frame | frame | [01](screenshots/timeline/phone-light/01-default-frame.png) | [01](screenshots/timeline/phone-dark/01-default-frame.png) | [01](screenshots/timeline/desktop-light/01-default-frame.png) | [01](screenshots/timeline/desktop-dark/01-default-frame.png) | [01](screenshots/timeline/wide-light/01-default-frame.png) | [01](screenshots/timeline/wide-dark/01-default-frame.png) |
| default-full-height | full | [02](screenshots/timeline/phone-light/02-default-full-height.png) | [02](screenshots/timeline/phone-dark/02-default-full-height.png) | [02](screenshots/timeline/desktop-light/02-default-full-height.png) | [02](screenshots/timeline/desktop-dark/02-default-full-height.png) | [02](screenshots/timeline/wide-light/02-default-full-height.png) | [02](screenshots/timeline/wide-dark/02-default-full-height.png) |
| CHILD Ivy | state | [03](screenshots/timeline/phone-light/03-child-ivy.png) | [03](screenshots/timeline/phone-dark/03-child-ivy.png) | [03](screenshots/timeline/desktop-light/03-child-ivy.png) | [03](screenshots/timeline/desktop-dark/03-child-ivy.png) | [03](screenshots/timeline/wide-light/03-child-ivy.png) | [03](screenshots/timeline/wide-dark/03-child-ivy.png) |
| CHILD Mia | state | [04](screenshots/timeline/phone-light/04-child-mia.png) | [04](screenshots/timeline/phone-dark/04-child-mia.png) | [04](screenshots/timeline/desktop-light/04-child-mia.png) | [04](screenshots/timeline/desktop-dark/04-child-mia.png) | [04](screenshots/timeline/wide-light/04-child-mia.png) | [04](screenshots/timeline/wide-dark/04-child-mia.png) |
| CHILD Leo | state | dup | dup | dup | dup | dup | dup |
| ROLE Guardian | state | dup | dup | dup | dup | dup | dup |
| ROLE Caregiver | state | [05](screenshots/timeline/phone-light/05-role-caregiver.png) | [05](screenshots/timeline/phone-dark/05-role-caregiver.png) | [05](screenshots/timeline/desktop-light/05-role-caregiver.png) | [05](screenshots/timeline/desktop-dark/05-role-caregiver.png) | [05](screenshots/timeline/wide-light/05-role-caregiver.png) | [05](screenshots/timeline/wide-dark/05-role-caregiver.png) |
| NEXT SAVE Works | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Offline | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Error | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Clash | state | dup | dup | dup | dup | dup | dup |
| UNITS ml | state | dup | dup | dup | dup | dup | dup |
| UNITS fl oz | state | [06](screenshots/timeline/phone-light/06-units-fl-oz.png) | [06](screenshots/timeline/phone-dark/06-units-fl-oz.png) | [06](screenshots/timeline/desktop-light/06-units-fl-oz.png) | [06](screenshots/timeline/desktop-dark/06-units-fl-oz.png) | [06](screenshots/timeline/wide-light/06-units-fl-oz.png) | [06](screenshots/timeline/wide-dark/06-units-fl-oz.png) |
| sheet Switch child | sheet | [07](screenshots/timeline/phone-light/07-sheet-switch-child.png) | [07](screenshots/timeline/phone-dark/07-sheet-switch-child.png) | [07](screenshots/timeline/desktop-light/07-sheet-switch-child.png) | [07](screenshots/timeline/desktop-dark/07-sheet-switch-child.png) | [07](screenshots/timeline/wide-light/07-sheet-switch-child.png) | [07](screenshots/timeline/wide-dark/07-sheet-switch-child.png) |
| sheet More | sheet | [08](screenshots/timeline/phone-light/08-sheet-more.png) | [08](screenshots/timeline/phone-dark/08-sheet-more.png) | · | · | · | · |

## vaccines

Route in the app: `/kids/{kid}/vaccines`

| State | Kind | phone-light | phone-dark | desktop-light | desktop-dark | wide-light | wide-dark |
|---|---|---|---|---|---|---|---|
| default-frame | frame | [01](screenshots/vaccines/phone-light/01-default-frame.png) | [01](screenshots/vaccines/phone-dark/01-default-frame.png) | [01](screenshots/vaccines/desktop-light/01-default-frame.png) | [01](screenshots/vaccines/desktop-dark/01-default-frame.png) | [01](screenshots/vaccines/wide-light/01-default-frame.png) | [01](screenshots/vaccines/wide-dark/01-default-frame.png) |
| default-full-height | full | [02](screenshots/vaccines/phone-light/02-default-full-height.png) | [02](screenshots/vaccines/phone-dark/02-default-full-height.png) | [02](screenshots/vaccines/desktop-light/02-default-full-height.png) | [02](screenshots/vaccines/desktop-dark/02-default-full-height.png) | [02](screenshots/vaccines/wide-light/02-default-full-height.png) | [02](screenshots/vaccines/wide-dark/02-default-full-height.png) |
| NEXT SAVE Works | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Offline | state | dup | dup | dup | dup | dup | dup |
| NEXT SAVE Error | state | dup | dup | dup | dup | dup | dup |
| SCHEDULE England | state | dup | dup | dup | dup | dup | dup |
| SCHEDULE US | state | [03](screenshots/vaccines/phone-light/03-schedule-us.png) | [03](screenshots/vaccines/phone-dark/03-schedule-us.png) | [03](screenshots/vaccines/desktop-light/03-schedule-us.png) | [03](screenshots/vaccines/desktop-dark/03-schedule-us.png) | [03](screenshots/vaccines/wide-light/03-schedule-us.png) | [03](screenshots/vaccines/wide-dark/03-schedule-us.png) |
| SCHEDULE None | state | [04](screenshots/vaccines/phone-light/04-schedule-none.png) | [04](screenshots/vaccines/phone-dark/04-schedule-none.png) | [04](screenshots/vaccines/desktop-light/04-schedule-none.png) | [04](screenshots/vaccines/desktop-dark/04-schedule-none.png) | [04](screenshots/vaccines/wide-light/04-schedule-none.png) | [04](screenshots/vaccines/wide-dark/04-schedule-none.png) |
| AGE Stored | state | dup | dup | dup | dup | dup | dup |
| AGE 3 wk | state | [05](screenshots/vaccines/phone-light/05-age-3-wk.png) | [05](screenshots/vaccines/phone-dark/05-age-3-wk.png) | [05](screenshots/vaccines/desktop-light/05-age-3-wk.png) | [05](screenshots/vaccines/desktop-dark/05-age-3-wk.png) | [05](screenshots/vaccines/wide-light/05-age-3-wk.png) | [05](screenshots/vaccines/wide-dark/05-age-3-wk.png) |
| AGE 5 mo | state | [06](screenshots/vaccines/phone-light/06-age-5-mo.png) | [06](screenshots/vaccines/phone-dark/06-age-5-mo.png) | [06](screenshots/vaccines/desktop-light/06-age-5-mo.png) | [06](screenshots/vaccines/desktop-dark/06-age-5-mo.png) | [06](screenshots/vaccines/wide-light/06-age-5-mo.png) | [06](screenshots/vaccines/wide-dark/06-age-5-mo.png) |
| AGE 14 mo | state | [07](screenshots/vaccines/phone-light/07-age-14-mo.png) | [07](screenshots/vaccines/phone-dark/07-age-14-mo.png) | [07](screenshots/vaccines/desktop-light/07-age-14-mo.png) | [07](screenshots/vaccines/desktop-dark/07-age-14-mo.png) | [07](screenshots/vaccines/wide-light/07-age-14-mo.png) | [07](screenshots/vaccines/wide-dark/07-age-14-mo.png) |
| AGE 4 y | state | [08](screenshots/vaccines/phone-light/08-age-4-y.png) | [08](screenshots/vaccines/phone-dark/08-age-4-y.png) | [08](screenshots/vaccines/desktop-light/08-age-4-y.png) | [08](screenshots/vaccines/desktop-dark/08-age-4-y.png) | [08](screenshots/vaccines/wide-light/08-age-4-y.png) | [08](screenshots/vaccines/wide-dark/08-age-4-y.png) |
| RECORDS Sample | state | dup | dup | dup | dup | dup | dup |
| RECORDS None | state | dup | dup | dup | dup | dup | dup |
| RECORDS Some | state | [09](screenshots/vaccines/phone-light/09-records-some.png) | [09](screenshots/vaccines/phone-dark/09-records-some.png) | [09](screenshots/vaccines/desktop-light/09-records-some.png) | [09](screenshots/vaccines/desktop-dark/09-records-some.png) | [09](screenshots/vaccines/wide-light/09-records-some.png) | [09](screenshots/vaccines/wide-dark/09-records-some.png) |
| RECORDS Caught up | state | dup | dup | dup | dup | dup | dup |
| ROLE Guardian | state | dup | dup | dup | dup | dup | dup |
| ROLE Caregiver | state | [10](screenshots/vaccines/phone-light/10-role-caregiver.png) | [10](screenshots/vaccines/phone-dark/10-role-caregiver.png) | [10](screenshots/vaccines/desktop-light/10-role-caregiver.png) | [10](screenshots/vaccines/desktop-dark/10-role-caregiver.png) | [10](screenshots/vaccines/wide-light/10-role-caregiver.png) | [10](screenshots/vaccines/wide-dark/10-role-caregiver.png) |
| sheet Switch child | sheet | [11](screenshots/vaccines/phone-light/11-sheet-switch-child.png) | [11](screenshots/vaccines/phone-dark/11-sheet-switch-child.png) | [11](screenshots/vaccines/desktop-light/11-sheet-switch-child.png) | [11](screenshots/vaccines/desktop-dark/11-sheet-switch-child.png) | [11](screenshots/vaccines/wide-light/11-sheet-switch-child.png) | [11](screenshots/vaccines/wide-dark/11-sheet-switch-child.png) |
| sheet More | sheet | [12](screenshots/vaccines/phone-light/12-sheet-more.png) | [12](screenshots/vaccines/phone-dark/12-sheet-more.png) | · | · | · | · |
