import { self } from '$app/service-worker';
import { isSameOriginPath, parsePayload } from '#lib/push.js';

// No fetch handler and no Cache API: offline, the browser shows its own error page.
const FALLBACK_TITLE = 'Reminder due';
const ICON = '/icons/icon-192.png';

self.addEventListener('install', (event) => {
	event.waitUntil(self.skipWaiting());
});

self.addEventListener('activate', (event) => {
	event.waitUntil(self.clients.claim());
});

// Chrome requires every push to show a notification, so a bad payload still shows a generic one.
self.addEventListener('push', (event) => {
	let payload: ReturnType<typeof parsePayload> = null;
	try {
		payload = parsePayload(event.data?.json());
	} catch {
		// Not JSON: the generic notification below.
	}
	event.waitUntil(
		self.registration.showNotification(payload?.title ?? FALLBACK_TITLE, {
			body: '',
			tag: payload?.tag,
			icon: ICON,
			data: { path: payload?.path ?? '/' }
		})
	);
});

async function focusOrOpen(path: string) {
	const target = new URL(path, self.location.origin);
	const windows = await self.clients.matchAll({ type: 'window', includeUncontrolled: true });
	const shown = windows.find((w) => {
		const url = new URL(w.url);
		return (
			url.origin === target.origin &&
			url.pathname === target.pathname &&
			url.search === target.search
		);
	});
	if (shown) await shown.focus();
	else await self.clients.openWindow(path);
}

self.addEventListener('notificationclick', (event) => {
	event.notification.close();
	const data: unknown = event.notification.data;
	const path = typeof data === 'object' && data !== null ? (data as { path?: unknown }).path : null;
	if (typeof path !== 'string' || !isSameOriginPath(path)) return;
	event.waitUntil(focusOrOpen(path));
});
