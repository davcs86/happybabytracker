import { redirect } from '@sveltejs/kit';
import { visitPath } from '#lib/paths.js';
import { setKidCookie } from '#lib/server/cookie.js';
import { failPage } from '#lib/server/outcome.js';
import {
	readVisitForm,
	visitApiFailure,
	visitEditValues,
	visitFieldFailure
} from '#lib/server/visit-form.js';
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
	return { kid, visit, values: visitEditValues(visit, kid.timeZone) };
};

export const actions: Actions = {
	save: async ({ request, locals, params }) => {
		const { kid, visit } = await stored(locals, params);
		const { values, input, field } = readVisitForm(await request.formData(), {
			kind: 'edit',
			zone: kid.timeZone
		});
		const extra = { form: 'save' as const };
		if (!input) return visitFieldFailure(values, field!, extra);
		const r = await locals.api.visits.update(kid.id, visit.id, input);
		if (!r.ok) return visitApiFailure(values, r, extra);
		redirect(303, visitPath(kid.id, visit.id));
	}
};
