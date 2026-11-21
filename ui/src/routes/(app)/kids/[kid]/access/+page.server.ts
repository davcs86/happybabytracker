import { fail, redirect } from '@sveltejs/kit';
import { KID_COOKIE, PARENT_HOME } from '#lib/server/constants.js';
import { clearKidCookie, setKidCookie } from '#lib/server/cookie.js';
import { failAction, failPage, type ApiFailure } from '#lib/server/outcome.js';
import type { Role } from '#lib/types.js';
import type { Actions, PageServerLoad } from './$types';

const PARENT_NOT_FOUND = 'No active account has that username.';
const LAST_GUARDIAN = 'A child needs at least one guardian.';
const ROLE_MESSAGE = 'Choose guardian or caregiver';

const readRole = (form: FormData) => String(form.get('role') ?? '') as Role;

// Refusals the access page words itself; anything else is the shared handling.
function refused(r: ApiFailure, action: 'add' | 'role' | 'remove', username?: string) {
	if (r.code === 'parent_not_found') {
		return fail(404, { action, username, field: 'username', message: PARENT_NOT_FOUND });
	}
	if (r.code === 'last_guardian') return fail(409, { action, field: null, message: LAST_GUARDIAN });
	if (r.reason === 'invalid' && r.field === 'role') {
		return fail(400, { action, username, field: 'role', message: ROLE_MESSAGE });
	}
	return failAction(r, { action, field: null });
}

export const load: PageServerLoad = async ({ locals, params, cookies }) => {
	const [kid, grants] = await Promise.all([
		locals.api.getKid(params.kid),
		locals.api.listAccess(params.kid)
	]);
	if (!kid.ok) failPage(kid);
	if (!grants.ok) failPage(grants);
	setKidCookie(cookies, kid.data.id);
	return { kid: kid.data, grants: grants.data, me: locals.session?.parent?.id };
};

export const actions: Actions = {
	add: async ({ request, locals, params }) => {
		const form = await request.formData();
		const username = String(form.get('username') ?? '');
		const r = await locals.api.setAccess(params.kid, username, readRole(form));
		return r.ok ? { action: 'add' as const, done: true } : refused(r, 'add', username);
	},
	role: async ({ request, locals, params }) => {
		const form = await request.formData();
		const r = await locals.api.setAccess(
			params.kid,
			String(form.get('username') ?? ''),
			readRole(form)
		);
		return r.ok ? { action: 'role' as const, done: true } : refused(r, 'role');
	},
	remove: async ({ request, locals, params, cookies }) => {
		const parentId = String((await request.formData()).get('parent_id') ?? '');
		const r = await locals.api.removeAccess(params.kid, parentId);
		if (!r.ok) return refused(r, 'remove');
		if (parentId === locals.session?.parent?.id) {
			if (cookies.get(KID_COOKIE) === params.kid) clearKidCookie(cookies);
			redirect(303, PARENT_HOME);
		}
		return { action: 'remove' as const, done: true };
	}
};
