import { fail } from '@sveltejs/kit';
import { DISPLAY_NAME_MAX_LEN, MAX_PASSWORD_LEN, MIN_PASSWORD_LEN } from '#lib/constants.js';
import { failAction, failPage } from '#lib/server/outcome.js';
import { passwordProblem } from '#lib/password.js';
import type { Units } from '#lib/types.js';
import type { Actions, PageServerLoad } from './$types';

const UNIT_KEYS = ['weight', 'length', 'volume', 'temp'] as const;
const DISPLAY_NAME_MESSAGE = `Use 1 to ${DISPLAY_NAME_MAX_LEN} characters`;
const UNIT_MESSAGE = 'Choose one of the listed units';

export const load: PageServerLoad = async ({ locals }) => {
	const r = await locals.api.listSessions();
	if (!r.ok) failPage(r);
	return { sessions: r.data };
};

export const actions: Actions = {
	profile: async ({ request, locals }) => {
		const form = await request.formData();
		const displayName = String(form.get('display_name') ?? '');
		const units = Object.fromEntries(
			UNIT_KEYS.map((k) => [k, String(form.get(k) ?? '')])
		) as unknown as Units;
		const values = { displayName, units };
		const r = await locals.api.updateProfile(displayName, units);
		if (r.ok) return { action: 'profile' as const, saved: true };
		if (r.reason === 'invalid' && r.field === 'display_name') {
			return fail(400, {
				action: 'profile' as const,
				values,
				field: 'display_name',
				message: DISPLAY_NAME_MESSAGE
			});
		}
		if (r.reason === 'invalid' && r.field?.startsWith('units.')) {
			return fail(400, {
				action: 'profile' as const,
				values,
				field: r.field.slice(6),
				message: UNIT_MESSAGE
			});
		}
		return failAction(r, { action: 'profile' as const, values, field: null });
	},
	password: async ({ request, locals }) => {
		const form = await request.formData();
		const current = String(form.get('current') ?? '');
		const next = String(form.get('new') ?? '');
		const problem = passwordProblem(next, String(form.get('confirm') ?? ''));
		if (problem) {
			const field = problem.field === 'password' ? 'new' : 'confirm';
			return fail(400, { action: 'password' as const, field, message: problem.message });
		}
		const r = await locals.api.changePassword(current, next);
		if (r.ok) return { action: 'password' as const, changed: true };
		if (r.code === 'wrong_password') {
			return fail(403, {
				action: 'password' as const,
				field: 'current',
				message: 'The current password is incorrect'
			});
		}
		if (r.reason === 'invalid' && r.field === 'new') {
			return fail(400, {
				action: 'password' as const,
				field: 'new',
				message: `Use ${MIN_PASSWORD_LEN} to ${MAX_PASSWORD_LEN} characters`
			});
		}
		return failAction(r, { action: 'password' as const, field: null });
	},
	signout: async ({ request, locals }) => {
		const id = String((await request.formData()).get('session_id') ?? '');
		const r = await locals.api.deleteSession(id);
		if (r.ok) return { action: 'signout' as const, done: true };
		// Already gone (another device signed out): the reloaded list shows the truth.
		if (r.reason === 'not_found') return { action: 'signout' as const, done: true };
		return failAction(r, { action: 'signout' as const, field: null });
	}
};
