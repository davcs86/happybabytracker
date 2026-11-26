import { redirect } from '@sveltejs/kit';
import { visitPath } from '#lib/paths.js';
import { setKidCookie } from '#lib/server/cookie.js';
import { failPage } from '#lib/server/outcome.js';
import { findProviders, pickProvider } from '#lib/server/provider-form.js';
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
	const [{ kid, visit }, search] = await Promise.all([
		stored(locals, params),
		locals.api.providers.state()
	]);
	setKidCookie(cookies, kid.id);
	return {
		kid,
		visit,
		values: visitEditValues(visit, kid.timeZone),
		// Any failure of the state call counts as off (plan 10 B9).
		providerSearch: search.ok && search.data.enabled
	};
};

export const actions: Actions = {
	// The panel's actions keep every typed value and save nothing.
	find: async ({ request, locals, params }) => {
		const { kid } = await stored(locals, params);
		const form = await request.formData();
		const { values } = readVisitForm(form, { kind: 'edit', zone: kid.timeZone });
		return findProviders(locals.api, form, values);
	},
	pick: async ({ request, locals, params }) => {
		const { kid } = await stored(locals, params);
		const form = await request.formData();
		const { values } = readVisitForm(form, { kind: 'edit', zone: kid.timeZone });
		return pickProvider(form, values);
	},
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
