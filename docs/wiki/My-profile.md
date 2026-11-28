# My profile

The account page of the signed-in parent. It is titled "Your profile" and opened from **My profile** in the menu. It holds your display name, the units you read measurements in, your password, the devices you are signed in on and the children you can see. Examples use a fake parent, Ana (Mia and Ivy as guardian, Leo as caregiver).

Every signed-in parent (guardian or caregiver) has this page. The admin does not: the admin has its own **Account** page in the [Admin](Admin.md) panel.

## Guide

### How do I open it?

- On a phone: **More**, then **My profile**.
- On a desktop: **My profile** in the left menu.

The line under the title summarises your access, for example "Guardian for 2 children, Caregiver for 1 child" ("No children yet" if you have none).

| Phone | Desktop |
|---|---|
| <img src="../screenshots/account/01-account-mobile.png" alt="Your profile on a phone: a Details card with Display name Ana, username ana, four unit selects (kg, cm, ml, °C) and Save; a Password card with three fields; Signed-in devices with This device and Another device (with a Sign out button); and Children you can see: Mia Guardian, Leo Caregiver, Ivy Guardian" width="260"> | <img src="../screenshots/account/01-account-desktop.png" alt="The same page on a desktop in two columns: Details and Password on the left, Signed-in devices and Children you can see on the right" width="460"> |

### How do I change my display name?

1. In **Details**, edit **Display name** (1 to 64 characters).
2. Choose **Save**. A box above the field says "Saved".

Your **Username** is shown below it, with the note "Your username is permanent." It cannot be changed.

An empty name is refused: the page keeps your units, shows "Check the fields with a message." and the red message "Use 1 to 64 characters" under the name.

| Phone |
|---|
| <img src="../screenshots/account/03-display-name-blank-mobile.png" alt="Details card on a phone with an empty Display name field, the box Check the fields with a message. above it and the red message Use 1 to 64 characters below it; the units show lb, in, fl oz and °F" width="260"> |

### How do I change my units?

In **Details**, four lists set how measurements are shown to you:

- **Weight**: kg or lb
- **Length**: cm or in
- **Volume**: ml or fl oz
- **Temperature**: °C or °F

Choose the units, then **Save**. In the picture the four are set to lb, in, fl oz and °F and the page says "Saved". The setting belongs to you, not to the child: two parents can read the same child in different units. If a unit is not one of the listed values, the message is "Choose one of the listed units".

| Phone |
|---|
| <img src="../screenshots/account/02-units-imperial-saved-mobile.png" alt="Your profile on a phone after saving: the box Saved above Display name, and the units lb, in, fl oz and °F" width="260"> |

### How do I change my password?

1. In **Password**, type your **Current password**.
2. Type a **New password** (at least 12 characters, and up to 256) and type it again in **Confirm new password**.
3. Choose **Change password**.

The page says "Password changed. Your other sessions were signed out." You stay signed in on this device; every other device is signed out and must sign in again with the new password. The three fields are always emptied after you send them.

If the new password and the confirmation differ, the page says "The passwords do not match" under the confirmation, without leaving the page. A password under 12 characters gives "Use at least 12 characters".

| Phone |
|---|
| <img src="../screenshots/account/04-password-mismatch-mobile.png" alt="Password card on a phone with the box Check the fields with a message. and, under Confirm new password, the red message The passwords do not match" width="260"> |

If the current password is wrong, the message is "The current password is incorrect" under **Current password**, and the password is not changed.

| Phone |
|---|
| <img src="../screenshots/account/05-wrong-current-password-mobile.png" alt="Password card on a phone with the box Check the fields with a message. and, under Current password, the red message The current password is incorrect" width="260"> |

After a successful change:

| Phone |
|---|
| <img src="../screenshots/account/06-password-changed-mobile.png" alt="Password card on a phone with the box Password changed. Your other sessions were signed out.; Signed-in devices now lists only This device and says No other devices are signed in." width="260"> |

### How do I link single sign-on?

When the deployment allows both passwords and single sign-on, a **Single sign-on** card follows **Password**. In any other mode the card is not there, and the page looks as above.

1. In **Single sign-on**, type your current **Password**. It proves that it is you at this device, even though you are already signed in.
2. Choose **Link single sign-on**. The single sign-on page asks you to sign in again, even if you are already signed in there.
3. You come back to a page that reads **You are linking** *your single sign-on name* **to** *your username*, for example "You are linking stub-user to ana". Check that the name is the single sign-on account you mean: if someone else is signed in there, it shows their name.
4. Choose **Confirm**. You are back on Your profile, and the card says "Single sign-on is linked." From then on you can also sign in with **Continue with single sign-on**; your password keeps working.

**Cancel** links nothing and takes you back to Your profile.

| Phone | Desktop |
|---|---|
| <img src="../screenshots/sso/10-account-linkable-mobile.png" alt="Your profile on a phone: after the Password card, a Single sign-on card with a Password field and an outline Link single sign-on button" width="260"> | <img src="../screenshots/sso/10-account-linkable-desktop.png" alt="The same page on a desktop: Details, Password and Single sign-on on the left, Signed-in devices and Children you can see on the right" width="460"> |

| Phone | Phone |
|---|---|
| <img src="../screenshots/sso/13-confirm-link-mobile.png" alt="The confirmation page on a phone: the heading You are linking stub-user to ana, and a card with a Confirm button and an outline Cancel button" width="220"> | <img src="../screenshots/sso/11-account-linked-mobile.png" alt="Your profile on a phone after linking: the Single sign-on card says Single sign-on is linked. and has no button" width="220"> |

A wrong password gives "The password is incorrect" under the field on a page titled **Link single sign-on**, and you are not sent to the single sign-on page. Too many tries give "Too many attempts, try again in a minute".

| Phone |
|---|
| <img src="../screenshots/sso/12-link-sso-wrong-password-mobile.png" alt="The Link single sign-on page on a phone: the Single sign-on card with the Password field marked invalid and the red message The password is incorrect under it" width="260"> |

If it does not complete:

| What you see | What it means |
|---|---|
| "Single sign-on did not complete" with **Go to My profile** | The single sign-on page did not ask you to sign in again, the round trip failed or took longer than 10 minutes, or you signed out or signed in as someone else in another tab before pressing Confirm. Nothing was linked; try again. |
| "Confirm your password again, then link" | More than 5 minutes passed between typing your password and pressing Confirm. Start again from Your profile. |
| "That single sign-on account is already linked to an account" | That single sign-on account already belongs to an account here. |
| "Linking is turned off" or "Single sign-on is turned off" | The deployment no longer allows both modes. |

Confirming twice links once; the second press shows "Single sign-on did not complete". You cannot unlink yourself: ask the admin (see [Admin](Admin.md)).

### How do I sign out another device?

**Signed-in devices** says "Each row is a device or browser where you're signed in. Signing one out ends that session only." Each row shows "This device" or "Another device" and when it was signed in and last seen. On another device, choose **Sign out** on its row. Your own device has no button here; use **Sign out** in the menu.

If no other device is signed in the card says "No other devices are signed in."

### Which children can I see?

**Children you can see** lists each child with a "Guardian" or "Caregiver" badge. To change who has access, a guardian uses [Children and access](Children-and-access.md). With no children it says "None yet. A guardian can invite you."

## Screens

| Screen | Route | Who can open it | States captured |
|---|---|---|---|
| Your profile | `/account` | guardian, caregiver (not the admin) | `01-account`, `02-units-imperial-saved`, `03-display-name-blank`, `04-password-mismatch`, `05-wrong-current-password`, `06-password-changed` |
| Your profile, Single sign-on card | `/account` (both modes on) | guardian, caregiver | `sso/10-account-linkable`, `sso/11-account-linked` |
| Link single sign-on | `/account/link-sso` (the card's form; also its page without JavaScript) | guardian, caregiver | `sso/12-link-sso-wrong-password` |
| Link confirmation | `/auth/callback`, then `/auth/confirm` | the signed-in parent who started the link | `sso/13-confirm-link` |

All the shots are in the [screenshot catalogue](../screenshots/README.md#account). In the shots the other device is a second session opened for the same parent, so its times equal this device's ("2026-11-24 10:30 UTC").

### Rules the screens show

- Display name: 1 to 64 characters. Username: permanent.
- Units: weight kg or lb, length cm or in, volume ml or fl oz, temperature °C or °F. They set how measurements are shown to this parent.
- Password: 12 to 256 characters (counted as characters, not bytes), checked in the browser and again by the server. A change ends every other session and keeps this one.
- Times of sign-in are shown in UTC, as "2026-11-24 10:30 UTC".
- No password, current or new, is ever kept in the page after it is sent.
- Password changes and the Link single sign-on password count against the same limit as signing in: "Too many attempts, try again in a minute".
- Linking needs your password within the last 5 minutes, and is confirmed only by the same session that started it.

### Not built yet

The design shows separate sheets for editing the name and changing the password, "last changed N days ago", device names and places, and "sign out all other devices". The forms are on the page instead and the others do not exist: see [Not built yet](Not-built-yet.md).

Back to the [wiki home](Home.md).
