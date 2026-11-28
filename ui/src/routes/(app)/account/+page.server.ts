import { fail } from '@sveltejs/kit';
import { DISPLAY_NAME_MAX_LEN, MAX_PASSWORD_LEN, MIN_PASSWORD_LEN } from '#lib/constants.js';
import { createApi } from '#lib/server/api.js';
import { SESSION_COOKIE } from '#lib/server/constants.js';
import { getEnv } from '#lib/server/env.js';
import { TOO_MANY_ATTEMPTS } from '#lib/server/oidc/link.js';
import { ssoOffered } from '#lib/server/oidc/start.js';
import { failAction, failPage } from '#lib/server/outcome.js';
import { passwordProblem } from '#lib/password.js';
import type { AuthMode, Units } from '#lib/types.js';
import type { Actions, PageServerLoad } from './$types';

const UNIT_KEYS = ['weight', 'length', 'volume', 'temp'] as const;
const DISPLAY_NAME_MESSAGE = `Use 1 to ${DISPLAY_NAME_MAX_LEN} characters`;
const UNIT_MESSAGE = 'Choose one of the listed units';

export const load: PageServerLoad = async ({ locals }) => {
	const [r, m] = await Promise.all([locals.api.listSessions(), locals.api.authMode()]);
	if (!r.ok) failPage(r);
	const mode: AuthMode = m.ok ? m.data : 'local';
	return { sessions: r.data, mode, sso: m.ok && ssoOffered(mode, getEnv().oidc) };
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
	password: async ({ request, cookies, getClientAddress }) => {
		const form = await request.formData();
		const current = String(form.get('current') ?? '');
		const next = String(form.get('new') ?? '');
		const problem = passwordProblem(next, String(form.get('confirm') ?? ''));
		if (problem) {
			const field = problem.field === 'password' ? 'new' : 'confirm';
			return fail(400, { action: 'password' as const, field, message: problem.message });
		}
		// The address is the limiter's addr: key; without it every parent would share one.
		const api = createApi(getEnv(), fetch, cookies.get(SESSION_COOKIE), getClientAddress());
		const r = await api.changePassword(current, next);
		if (r.ok) return { action: 'password' as const, changed: true };
		if (r.code === 'wrong_password') {
			return fail(403, {
				action: 'password' as const,
				field: 'current',
				message: 'The current password is incorrect'
			});
		}
		if (r.status === 429) {
			return fail(429, { action: 'password' as const, field: null, message: TOO_MANY_ATTEMPTS });
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
	// One call per other session: the list is read once, and a session that is already gone is no failure.
	signoutOthers: async ({ locals }) => {
		const list = await locals.api.listSessions();
		if (!list.ok) return failAction(list, { action: 'signout' as const, field: null });
		for (const s of list.data.filter((x) => !x.current)) {
			const r = await locals.api.deleteSession(s.id);
			if (!r.ok && r.reason !== 'not_found') {
				return failAction(r, { action: 'signout' as const, field: null });
			}
		}
		return { action: 'signout' as const, done: true };
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
