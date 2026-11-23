import { redirect } from '@sveltejs/kit';
import { medicationPath } from '#lib/paths.js';
import { setKidCookie } from '#lib/server/cookie.js';
import {
	courseApiFailure,
	courseCreateValues,
	courseFieldFailure,
	readMedicationForm
} from '#lib/server/medication-form.js';
import { failPage } from '#lib/server/outcome.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, params, cookies }) => {
	const kid = await locals.api.getKid(params.kid);
	if (!kid.ok) failPage(kid);
	setKidCookie(cookies, kid.data.id);
	return { kid: kid.data, values: courseCreateValues() };
};

export const actions: Actions = {
	save: async ({ request, locals, params }) => {
		const kid = await locals.api.getKid(params.kid);
		if (!kid.ok) failPage(kid);
		const { values, input, field } = readMedicationForm(await request.formData(), {
			kind: 'create',
			zone: kid.data.timeZone
		});
		const extra = { form: 'save' as const };
		if (!input) return courseFieldFailure(values, field!, extra);
		const r = await locals.api.medications.create(kid.data.id, input);
		if (!r.ok) return courseApiFailure(values, r, extra);
		redirect(303, medicationPath(kid.data.id));
	}
};
