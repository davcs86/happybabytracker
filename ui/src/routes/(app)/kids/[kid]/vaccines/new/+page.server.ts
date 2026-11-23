import { redirect } from '@sveltejs/kit';
import { vaccinesPath } from '#lib/paths.js';
import { todayInOrUtc } from '#lib/time-zone.js';
import { setKidCookie } from '#lib/server/cookie.js';
import { failPage } from '#lib/server/outcome.js';
import {
	prefillAntigens,
	readVaccineForm,
	vaccineApiFailure,
	vaccineCreateValues,
	vaccineFieldFailure
} from '#lib/server/vaccine-form.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, params, url, cookies }) => {
	const [kid, catalogue] = await Promise.all([
		locals.api.getKid(params.kid),
		locals.api.vaccineCatalogue()
	]);
	if (!kid.ok) failPage(kid);
	if (!catalogue.ok) failPage(catalogue);
	setKidCookie(cookies, kid.data.id);
	const today = todayInOrUtc(kid.data.timeZone);
	return {
		kid: kid.data,
		today,
		catalogue: catalogue.data,
		values: {
			...vaccineCreateValues(today),
			antigens: prefillAntigens(url.searchParams.get('antigens'), catalogue.data)
		}
	};
};

export const actions: Actions = {
	save: async ({ request, locals, params }) => {
		const kid = await locals.api.getKid(params.kid);
		if (!kid.ok) failPage(kid);
		const { values, input, field, message } = readVaccineForm(await request.formData(), {
			kind: 'create'
		});
		const extra = { form: 'save' as const };
		if (!input) return vaccineFieldFailure(values, field!, extra, message);
		const r = await locals.api.createVaccination(kid.data.id, input);
		if (!r.ok) return vaccineApiFailure(values, r, extra);
		redirect(303, vaccinesPath(kid.data.id));
	}
};
