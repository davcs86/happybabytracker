import { json } from '@sveltejs/kit';
import { PUSH_OFF_MESSAGE } from '#lib/constants.js';
import type { ApiFailure } from '#lib/server/outcome.js';
import {
	FAILURE_MESSAGE,
	PUSH_DEVICE_COOKIE,
	UNAVAILABLE_MESSAGE,
	UNREACHABLE_MESSAGE
} from '#lib/server/constants.js';
import { setPushDeviceCookie } from '#lib/server/cookie.js';
import { carryOver } from '#lib/server/push-renew.js';
import {
	DEVICE_LIMIT_MESSAGE,
	SUBSCRIPTION_REFUSED_MESSAGE
} from '#lib/server/notification-form.js';
import type { RequestHandler } from './$types';

const nonEmpty = (v: unknown): v is string => typeof v === 'string' && v !== '';

// Only {endpoint, keys: {p256dh, auth}}; the endpoint and keys go to the API and nowhere else.
function readSubscription(v: unknown) {
	if (typeof v !== 'object' || v === null) return null;
	const { endpoint, keys } = v as { endpoint?: unknown; keys?: unknown };
	if (!nonEmpty(endpoint) || typeof keys !== 'object' || keys === null) return null;
	const { p256dh, auth } = keys as { p256dh?: unknown; auth?: unknown };
	if (!nonEmpty(p256dh) || !nonEmpty(auth)) return null;
	return { endpoint, keys: { p256dh, auth } };
}

// Every answer is a fixed message: never a status text, a field value or the API's body.
function failure(r: ApiFailure) {
	if (r.code === 'push_unavailable') return json({ message: PUSH_OFF_MESSAGE }, { status: 409 });
	if (r.code === 'limit') return json({ message: DEVICE_LIMIT_MESSAGE }, { status: 409 });
	if (r.reason === 'invalid')
		return json({ message: SUBSCRIPTION_REFUSED_MESSAGE }, { status: 422 });
	if (r.reason === 'unavailable') return json({ message: UNREACHABLE_MESSAGE }, { status: 503 });
	if (r.reason === 'unauthorized' && r.code !== 'no_session') {
		return json({ message: UNAVAILABLE_MESSAGE }, { status: 503 });
	}
	if (r.reason === 'unauthorized') return json({ message: FAILURE_MESSAGE }, { status: 401 });
	return json({ message: FAILURE_MESSAGE }, { status: 500 });
}

// Without a cookie-named device no label is sent: the API keeps a known endpoint's name.
export const POST: RequestHandler = async ({ request, locals, cookies }) => {
	const type = request.headers.get('content-type')?.split(';')[0].trim().toLowerCase();
	if (type !== 'application/json') return json({ message: FAILURE_MESSAGE }, { status: 415 });
	const state = await locals.api.push.state();
	if (!state.ok) return failure(state);
	if (!state.data.enabled) return json({ message: PUSH_OFF_MESSAGE }, { status: 409 });

	const sub = readSubscription(await request.json().catch(() => null));
	if (!sub) return json({ message: FAILURE_MESSAGE }, { status: 400 });

	let previous = null;
	const cookie = cookies.get(PUSH_DEVICE_COOKIE);
	if (cookie) {
		const devices = await locals.api.push.devices();
		if (!devices.ok) return failure(devices);
		previous = devices.data.find((d) => d.id === cookie) ?? null;
	}

	const r = await locals.api.push.upsertDevice(previous ? { ...sub, label: previous.label } : sub);
	if (!r.ok) return failure(r);
	// A renewed subscription is a new device: it takes the old one's kids, and the cookie moves only then.
	if (previous && previous.id !== r.data.id) {
		const refused = await carryOver(locals.api.push, previous, r.data);
		if (refused) return failure(refused);
	}
	// A device another parent registered moves here under a new id: the cookie always takes this one.
	setPushDeviceCookie(cookies, r.data.id);
	return new Response(null, { status: 204 });
};
