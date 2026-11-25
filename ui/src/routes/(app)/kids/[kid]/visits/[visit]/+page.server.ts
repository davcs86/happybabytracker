import { redirect } from '@sveltejs/kit';
import { visitPath } from '#lib/paths.js';
import { setKidCookie } from '#lib/server/cookie.js';
import { failPage } from '#lib/server/outcome.js';
import { visitRefusal } from '#lib/server/visit-form.js';
import type { Actions, PageServerLoad } from './$types';

// The kid and the visit, read together: no grant, an unknown visit and another kid's visit are the same 404 page.
async function stored(locals: App.Locals, params: { kid: string; visit: string }) {
	const [kid, visit] = await Promise.all([
		locals.api.getKid(params.kid),
		locals.api.visits.get(params.kid, params.visit)
	]);
	if (!kid.ok) failPage(kid);
	if (!visit.ok) failPage(visit);
	return { kid: kid.data, visit: visit.data };
}

export const load: PageServerLoad = async ({ locals, params, cookies }) => {
	const { kid, visit } = await stored(locals, params);
	setKidCookie(cookies, kid.id);
	return { kid, visit };
};

async function setCancelled(
	locals: App.Locals,
	params: { kid: string; visit: string },
	cancelled: boolean
) {
	const { kid, visit } = await stored(locals, params);
	const r = await locals.api.visits.update(kid.id, visit.id, { cancelled });
	if (!r.ok) return visitRefusal(r, { form: 'status' as const });
	redirect(303, visitPath(kid.id, visit.id));
}

export const actions: Actions = {
	cancel: ({ locals, params }) => setCancelled(locals, params, true),
	restore: ({ locals, params }) => setCancelled(locals, params, false)
};
