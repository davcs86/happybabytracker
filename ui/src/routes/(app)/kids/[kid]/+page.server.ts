import { redirect } from '@sveltejs/kit';
import { formatAge } from '#lib/format.js';
import { todayInOrUtc } from '#lib/time-zone.js';
import { KID_COOKIE, PARENT_HOME } from '#lib/server/constants.js';
import { clearKidCookie, setKidCookie } from '#lib/server/cookie.js';
import { failAction, failPage } from '#lib/server/outcome.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, params, cookies }) => {
	const r = await locals.api.getKid(params.kid);
	if (!r.ok) failPage(r);
	setKidCookie(cookies, r.data.id);
	return { kid: r.data, age: formatAge(r.data.birthDate, todayInOrUtc(r.data.timeZone)) };
};

export const actions: Actions = {
	// The parent is always the signed-in principal; posted fields are never read.
	leave: async ({ locals, params, cookies }) => {
		const r = await locals.api.removeAccess(params.kid, locals.session?.parent?.id ?? '');
		// not_found: already left, or the grant was revoked meanwhile.
		if (r.ok || r.reason === 'not_found') {
			if (cookies.get(KID_COOKIE) === params.kid) clearKidCookie(cookies);
			redirect(303, PARENT_HOME);
		}
		return failAction(r, { field: null });
	}
};
