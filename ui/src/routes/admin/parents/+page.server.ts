import { fail } from '@sveltejs/kit';
import { DISPLAY_NAME_MAX_LEN, USERNAME_MAX_LEN, USERNAME_MIN_LEN } from '#lib/constants.js';
import { failAction, failPage, type ApiFailure } from '#lib/server/outcome.js';
import type { ParentLink } from '#lib/types.js';
import type { Actions, PageServerLoad } from './$types';

const USERNAME_MESSAGE = `Use ${USERNAME_MIN_LEN} to ${USERNAME_MAX_LEN} characters: a–z, 0–9, dot, underscore or hyphen`;
const SOLE_GUARDIAN =
	'This parent is the only guardian of at least one child. Another guardian must be added first.';
const GONE = 'That parent no longer exists';
const UNLINKED = 'Single sign-on unlinked. Their sessions and notification devices were removed.';

export const load: PageServerLoad = async ({ locals }) => {
	const r = await locals.api.listParents();
	if (!r.ok) failPage(r);
	return { parents: r.data };
};

// The link lives only in this response's data: never a cookie, a URL or a log.
const shown = (username: string, link: ParentLink) => ({ link: { username, ...link } });

function parentFailure(r: ApiFailure) {
	if (r.reason === 'not_found') return fail(404, { field: null, message: GONE });
	return failAction(r, { field: null });
}

export const actions: Actions = {
	create: async ({ request, locals }) => {
		const form = await request.formData();
		const username = String(form.get('username') ?? '');
		const displayName = String(form.get('display_name') ?? '');
		const values = { username, displayName };
		const r = await locals.api.createParent(username, displayName);
		if (r.ok) return shown(r.data.parent.username, r.data.link);
		if (r.code === 'username_taken') {
			return fail(409, { values, field: 'username', message: 'That username is already taken' });
		}
		if (r.reason === 'invalid' && r.field === 'username') {
			return fail(400, { values, field: 'username', message: USERNAME_MESSAGE });
		}
		if (r.reason === 'invalid' && r.field === 'display_name') {
			return fail(400, {
				values,
				field: 'display_name',
				message: `Use 1 to ${DISPLAY_NAME_MAX_LEN} characters`
			});
		}
		return failAction(r, { values, field: null });
	},
	link: async ({ request, locals }) => {
		const form = await request.formData();
		const r = await locals.api.parentLink(String(form.get('parent_id') ?? ''));
		return r.ok ? shown(String(form.get('username') ?? ''), r.data) : parentFailure(r);
	},
	disable: async ({ request, locals }) => {
		const r = await locals.api.setParentDisabled(
			String((await request.formData()).get('parent_id') ?? ''),
			true
		);
		if (r.ok) return { done: true };
		if (r.code === 'sole_guardian') return fail(409, { field: null, message: SOLE_GUARDIAN });
		return parentFailure(r);
	},
	enable: async ({ request, locals }) => {
		const r = await locals.api.setParentDisabled(
			String((await request.formData()).get('parent_id') ?? ''),
			false
		);
		return r.ok ? { done: true } : parentFailure(r);
	},
	unlink: async ({ request, locals }) => {
		const r = await locals.api.unlinkParentSso(
			String((await request.formData()).get('parent_id') ?? '')
		);
		return r.ok ? { status: UNLINKED } : parentFailure(r);
	}
};
