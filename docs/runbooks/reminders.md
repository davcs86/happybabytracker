# Reminders: health, a missing reminder, turning Web Push on

The API's worker ticks every 30 seconds. Each tick keeps a `reminders` row for every active scheduled
course (the occurrences due in the last 30 minutes and the next one ahead), opens one delivery per allowed
target of each configured channel when a reminder falls due, sends each target on its own, settles it, and
expires what is past its due time plus 30 minutes. The home Due list and the Medication page's "Next due"
read the same rows, so they work with no channel configured. Spec: sections 11 and 11.1 of
`docs/superpowers/specs/2026-11-20-happybabytracker-design.md`.

## Deploy and rollback

Both images ship together (the edge's bump PR pins both). Migration 0006 is forward-only:
`api/internal/store/migrate.go` makes an older API refuse a newer schema, and the previous UI rejects the
home list's medication entries, so rolling back the UI alone breaks the home page of a kid whose Due list
has a medication entry.
Take an `edge-backup` archive right before the deploy. A rollback is both image pins back plus a restore of
the last `edge-backup` archive; anything written since that archive is lost.

## Health answers

All three are reached through the UI (the API's port is never published).

| Path | 200 means | 503 means |
|---|---|---|
| `/healthz` | the UI process answers | (none: a UI that is down does not answer) |
| `/healthz/api` | the API's database answers **and** the reminder worker finished a tick within the last 60 seconds (the first 60 seconds after a start count from the start) | the database does not answer, or the worker has not ticked for over 60 seconds: reminders are neither materialised nor sent nor expired |
| `/healthz/reminders` | no delivery is stuck and none failed lately; always 200 with no channel configured (there are no deliveries) | a delivery of a still-pending reminder is `pending` or `sending` more than 15 minutes after its due time, or a delivery was settled `failed` or `expired` within the last 15 minutes |

Through the UI every body is plain text: `ok` with a 200, `unavailable` with a 503. The API's own
routes answer JSON: `/healthz` `{"status":"ok"}` or, on 503, `{"error":"unavailable"}`; `/healthz/reminders`
`{"status":"ok"}` or `{"status":"unavailable"}`. No body names a kid, a count or a target. A 503 from
`/healthz/reminders` clears on its own 15 minutes after the last failure once deliveries succeed again.

## A reminder did not arrive

Check in this order; each step names what to look at, never a secret.

1. **The course.** On the kid's Medication page the course is active, inside its dates, has a schedule
   (not "as needed") and shows "Next due". The kid's `medication` module is on (Modules page; with it off the page says "reminders are paused").
2. **The Due list.** The occurrence shows on the kid's home Due list: the row exists (course, dose and
   kid-zone writes create rows too, not only the worker). Whether it is sent depends on the worker: check
   `/healthz/api` (a stalled worker) and the API log for `reminder step failed` (its `step`
   and `cause` say which store call failed).
3. **A dose already logged.** A dose logged from 30 minutes before the due time marks that occurrence done,
   and nothing is sent for it.
4. **The channel is on.** The API logs `channels` with `push` and `ntfy` `on` or `off` at every start. The
   Notifications page says "Web Push is not configured on this server." while Web Push is off.
5. **The opt-in.** On the parent's Notifications page the device (or the ntfy topic) lists the kid. Only a
   parent who is enabled and has access to the kid receives reminders; revoking access removes that kid's
   opt-ins, and disabling the parent deletes the parent's devices, ntfy topic and opt-ins.
6. **The send.** The API logs one `delivery` line per attempt with its `outcome` (`sent`, `gone`,
   `failed`, `retry`, `cancelled`) and `status` (the push service's or ntfy's HTTP status). A 429 or 5xx is
   retried after 30 s, 2 min and 10 min; nothing is sent later than 30 minutes after the due time
   (`expired`). A 404 or 410 deletes that device (`gone`). A 400, 401, 403 or 413 is `failed` and the device
   is kept: on every device at once it usually means the VAPID key changed (below). ntfy's 401 or 403 is
   logged once per tick as `ntfy refused` with its status: the token or the server's ACL is wrong.
7. **The phone.** The notification arrived but did not show: the browser's or the phone's notification
   permission, focus modes, or, on iPhone and iPad, an app not opened from the Home Screen.

Known gaps by design:

- A browser that renews its subscription (the endpoint changes) keeps its kids and label only if the parent
  opens the Notifications page before the next reminder is sent: the page carries them over to the new
  subscription. If a send reaches the dead endpoint first (the push service answers 404 or 410), the
  worker deletes that device and its opt-ins, and the renewed subscription registers with no kids. The
  parent ticks the kids again. There is no `pushsubscriptionchange` handler.
- Signing out of the app does not unsubscribe a browser or stop ntfy on a phone: the parent turns them
  off first ("Turn off on this device", "Turn off ntfy"); both sections of the Notifications page say so.

## Turning Web Push on

Web Push is on only when the API has both `VAPID_PRIVATE_KEY` and `VAPID_SUBJECT` (both or neither; one
alone stops the API from starting, naming the missing variable). On the edge this is a separate PR in
`davcs86/homelab-edge` that renders the stored `happybabytracker_vapid_private_key` and
`VAPID_SUBJECT` (the app's `https:` origin) into the API's environment, then a deploy. The app needs no
change. After the deploy the API logs `channels` with `push` `on`, and each parent turns notifications on
per device under Account, Notifications.

The key is 43 characters of unpadded base64url (a raw P-256 private key). Never put it in a log, an issue
or a commit; `HBT_ALLOW_FAKE_PUSH` is for tests and the smoke script only and must never be set on the
edge.

## Rotating the VAPID key

Every registered device was subscribed under the old public key, so after a rotation every device stops
delivering: the push services answer 401 or 403, which is kept as `failed` (the device is not deleted) and
turns `/healthz/reminders` 503 for 15 minutes after each failure. Opened on such a device, the Notifications
page says "Turn notifications off and on again on this device." Each parent opens Account, Notifications on
each device, uses "Turn off on this device", then "Turn on
notifications on this device", which subscribes again under the new key, and ticks the kids again.

## iPhone and iPad

Web Push works only for the app added to the Home Screen (iOS and iPadOS 16.4 or later) and opened from
there; the Home Screen app has its own login. Only a real iPhone shows whether Apple accepts the
tailnet-only `https:` VAPID subject, delivers, and opens the app on a tap.

## Manual check after a deploy with Web Push on

The automated tests cannot show a displayed notification (the e2e service-worker test checks what the
worker passes to `showNotification` and what a `notificationclick` does with synthetic events; Playwright's
headless browser never displays notifications). On a real phone, with notifications on for a kid and a
course whose next dose is due in a few minutes:

1. When the dose falls due, the phone shows "Medication due for <kid's name>", with no medication name.
2. Tapping it opens the kid's Medication page, or focuses it when it is already open.
