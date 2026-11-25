import { redirect } from '@sveltejs/kit';
import { visitsPath } from '#lib/paths.js';
import { setKidCookie } from '#lib/server/cookie.js';
import { failPage } from '#lib/server/outcome.js';
import {
	readVisitForm,
	visitApiFailure,
	visitCreateValues,
	visitFieldFailure
} from '#lib/server/visit-form.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, params, cookies }) => {
	const kid = await locals.api.getKid(params.kid);
	if (!kid.ok) failPage(kid);
	setKidCookie(cookies, kid.data.id);
	return { kid: kid.data, values: visitCreateValues(new Date(), kid.data.timeZone) };
};

export const actions: Actions = {
	save: async ({ request, locals, params }) => {
		const kid = await locals.api.getKid(params.kid);
		if (!kid.ok) failPage(kid);
		const { values, input, field } = readVisitForm(await request.formData(), {
			kind: 'create',
			zone: kid.data.timeZone
		});
		const extra = { form: 'save' as const };
		if (!input) return visitFieldFailure(values, field!, extra);
		const r = await locals.api.visits.create(kid.data.id, input);
		if (!r.ok) return visitApiFailure(values, r, extra);
		redirect(303, visitsPath(kid.data.id));
	}
};
