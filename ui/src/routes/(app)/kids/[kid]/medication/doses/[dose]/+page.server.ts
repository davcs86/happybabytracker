import { redirect } from '@sveltejs/kit';
import { medicationPath } from '#lib/paths.js';
import { setKidCookie } from '#lib/server/cookie.js';
import {
	doseApiFailure,
	doseEditValues,
	doseFieldFailure,
	readDoseForm
} from '#lib/server/medication-form.js';
import { failPage } from '#lib/server/outcome.js';
import type { Actions, PageServerLoad } from './$types';

// The kid and the dose, read together: no grant, an unknown dose and another kid's dose are the same 404 page.
async function stored(locals: App.Locals, params: { kid: string; dose: string }) {
	const [kid, dose] = await Promise.all([
		locals.api.getKid(params.kid),
		locals.api.doses.get(params.kid, params.dose)
	]);
	if (!kid.ok) failPage(kid);
	if (!dose.ok) failPage(dose);
	return { kid: kid.data, dose: dose.data };
}

export const load: PageServerLoad = async ({ locals, params, cookies }) => {
	const { kid, dose } = await stored(locals, params);
	const course = await locals.api.medications.get(kid.id, dose.medication);
	if (!course.ok) failPage(course);
	setKidCookie(cookies, kid.id);
	return { kid, course: course.data, values: doseEditValues(dose, kid.timeZone) };
};

export const actions: Actions = {
	save: async ({ request, locals, params }) => {
		const { kid } = await stored(locals, params);
		const { values, input, field } = readDoseForm(await request.formData(), {
			kind: 'edit',
			zone: kid.timeZone
		});
		const extra = { form: 'save' as const };
		if (!input) return doseFieldFailure(values, field!, extra);
		const r = await locals.api.doses.update(kid.id, params.dose, input);
		if (!r.ok) return doseApiFailure(values, r, extra);
		redirect(303, medicationPath(kid.id));
	}
};
