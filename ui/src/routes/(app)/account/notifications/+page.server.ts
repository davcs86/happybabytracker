import { fail } from '@sveltejs/kit';
import { PUSH_OFF_MESSAGE } from '#lib/constants.js';
import type { ApiResult } from '#lib/server/api.js';
import { PUSH_DEVICE_COOKIE } from '#lib/server/constants.js';
import { clearPushDeviceCookie } from '#lib/server/cookie.js';
import {
	DEVICE_FIELD_MESSAGES,
	NTFY_OFF_MESSAGE,
	readDeviceForm
} from '#lib/server/notification-form.js';
import { failAction, failPage, type ApiFailure } from '#lib/server/outcome.js';
import type { Actions, PageServerLoad } from './$types';

const NOT_FOUND: ApiFailure = { ok: false, status: 404, reason: 'not_found' };
const NTFY_OFF: ApiFailure = {
	ok: false,
	status: 409,
	reason: 'conflict',
	code: 'ntfy_unavailable'
};

export const load: PageServerLoad = async ({ locals, cookies }) => {
	const [push, devices, ntfy] = await Promise.all([
		locals.api.push.state(),
		locals.api.push.devices(),
		locals.api.ntfy.state()
	]);
	if (!push.ok) failPage(push);
	if (!devices.ok) failPage(devices);
	if (!ntfy.ok) failPage(ntfy);
	const cookie = cookies.get(PUSH_DEVICE_COOKIE);
	return {
		push: push.data,
		devices: devices.data,
		ntfy: ntfy.data,
		// This browser's device, only while it is still this parent's.
		currentDeviceId: devices.data.find((d) => d.id === cookie)?.id ?? null
	};
};

// Writes each call in turn and stops at the first refusal.
async function writeAll(calls: (() => Promise<ApiResult<unknown>>)[]) {
	for (const call of calls) {
		const r = await call();
		if (!r.ok) return r;
	}
	return null;
}

// The kids to switch on and off so the stored set becomes the chosen one.
function changes(stored: string[], chosen: string[]) {
	return {
		on: chosen.filter((k) => !stored.includes(k)),
		off: stored.filter((k) => !chosen.includes(k))
	};
}

export const actions: Actions = {
	// Only a changed label or kid is written; a device not in this parent's list is the 404 page.
	device: async ({ request, locals }) => {
		const form = await request.formData();
		const id = String(form.get('device') ?? '');
		const { values, input, field } = readDeviceForm(form);
		const extra = { action: 'device' as const, device: id, values };
		if (!input) {
			const f = field ?? 'label';
			return fail(400, { ...extra, field: f, message: DEVICE_FIELD_MESSAGES[f] });
		}
		const list = await locals.api.push.devices();
		if (!list.ok) return failAction(list, { ...extra, field: null });
		const device = list.data.find((d) => d.id === id);
		if (!device) failPage(NOT_FOUND);
		const { on, off } = changes(device.kids, input.kids);
		const r = await writeAll([
			...(input.label === device.label
				? []
				: [() => locals.api.push.renameDevice(device.id, input.label)]),
			...on.map((k) => () => locals.api.push.optIn(k, device.id)),
			...off.map((k) => () => locals.api.push.optOut(k, device.id))
		]);
		if (!r) return { action: 'device' as const, saved: true };
		if (r.code === 'push_unavailable') {
			return fail(409, { ...extra, field: null, message: PUSH_OFF_MESSAGE });
		}
		if (r.reason === 'invalid' && r.field === 'label') {
			return fail(400, { ...extra, field: 'label', message: DEVICE_FIELD_MESSAGES.label });
		}
		return failAction(r, { ...extra, field: null });
	},
	remove: async ({ request, locals, cookies }) => {
		const id = String((await request.formData()).get('device') ?? '');
		const r = await locals.api.push.removeDevice(id);
		if (!r.ok) return failAction(r, { action: 'remove' as const, field: null });
		if (cookies.get(PUSH_DEVICE_COOKIE) === id) clearPushDeviceCookie(cookies);
		return { action: 'remove' as const, removed: true };
	},
	ntfy_on: async ({ locals }) => ntfyResult('ntfy_on', await locals.api.ntfy.enable()),
	ntfy_regenerate: async ({ locals }) =>
		ntfyResult('ntfy_regenerate', await locals.api.ntfy.regenerate()),
	ntfy_off: async ({ locals }) => {
		const r = await locals.api.ntfy.disable();
		// Already off (another tab turned it off): the reloaded page shows the truth.
		if (!r.ok && r.reason === 'not_found') return { action: 'ntfy_off' as const, saved: true };
		return ntfyResult('ntfy_off', r);
	},
	ntfy_kids: async ({ request, locals }) => {
		const chosen = [...new Set((await request.formData()).getAll('kid').map(String))];
		const state = await locals.api.ntfy.state();
		if (!state.ok) return ntfyResult('ntfy_kids', state);
		if (!state.data.enabled) return ntfyResult('ntfy_kids', NTFY_OFF);
		const { on, off } = changes(state.data.kids, chosen);
		const r = await writeAll([
			...on.map((k) => () => locals.api.ntfy.optIn(k)),
			...off.map((k) => () => locals.api.ntfy.optOut(k))
		]);
		return ntfyResult('ntfy_kids', r ?? { ok: true, data: null });
	}
};

type NtfyAction = 'ntfy_on' | 'ntfy_regenerate' | 'ntfy_off' | 'ntfy_kids';

function ntfyResult(action: NtfyAction, r: ApiResult<unknown>) {
	if (r.ok) return { action, saved: true };
	if (r.code === 'ntfy_unavailable') {
		return fail(409, { action, field: null, message: NTFY_OFF_MESSAGE });
	}
	return failAction(r, { action, field: null });
}
