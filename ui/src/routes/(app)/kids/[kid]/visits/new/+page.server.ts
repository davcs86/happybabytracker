import { redirect } from '@sveltejs/kit';
import { visitsPath } from '#lib/paths.js';
import { setKidCookie } from '#lib/server/cookie.js';
import { failPage } from '#lib/server/outcome.js';
import { findProviders, pickProvider } from '#lib/server/provider-form.js';
import {
	readVisitForm,
	visitApiFailure,
	visitCreateValues,
	visitFieldFailure
} from '#lib/server/visit-form.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, params, cookies }) => {
	const [kid, search] = await Promise.all([
		locals.api.getKid(params.kid),
		locals.api.providers.state()
	]);
	if (!kid.ok) failPage(kid);
	setKidCookie(cookies, kid.data.id);
	return {
		kid: kid.data,
		values: visitCreateValues(new Date(), kid.data.timeZone),
		// Any failure of the state call counts as off (plan 10 B9).
		providerSearch: search.ok && search.data.enabled
	};
};

// The panel's actions keep every typed value and save nothing.
async function typed(locals: App.Locals, kidId: string) {
	const kid = await locals.api.getKid(kidId);
	if (!kid.ok) failPage(kid);
	return kid.data;
}

export const actions: Actions = {
	find: async ({ request, locals, params }) => {
		const kid = await typed(locals, params.kid);
		const form = await request.formData();
		const { values } = readVisitForm(form, { kind: 'create', zone: kid.timeZone });
		return findProviders(locals.api, form, values);
	},
	pick: async ({ request, locals, params }) => {
		const kid = await typed(locals, params.kid);
		const form = await request.formData();
		const { values } = readVisitForm(form, { kind: 'create', zone: kid.timeZone });
		return pickProvider(form, values);
	},
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
