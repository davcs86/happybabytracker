import { redirect } from '@sveltejs/kit';
import { medicationPath } from '#lib/paths.js';
import { setKidCookie } from '#lib/server/cookie.js';
import {
	courseApiFailure,
	courseEditValues,
	courseFieldFailure,
	readMedicationForm
} from '#lib/server/medication-form.js';
import { failAction, failPage } from '#lib/server/outcome.js';
import type { Actions, PageServerLoad } from './$types';

// The kid and the course, read together: no grant, an unknown course and another kid's course are the same 404 page.
async function stored(locals: App.Locals, params: { kid: string; medication: string }) {
	const [kid, course] = await Promise.all([
		locals.api.getKid(params.kid),
		locals.api.medications.get(params.kid, params.medication)
	]);
	if (!kid.ok) failPage(kid);
	if (!course.ok) failPage(course);
	return { kid: kid.data, course: course.data };
}

export const load: PageServerLoad = async ({ locals, params, cookies }) => {
	const { kid, course } = await stored(locals, params);
	setKidCookie(cookies, kid.id);
	return { kid, course, values: courseEditValues(course, kid.timeZone) };
};

export const actions: Actions = {
	save: async ({ request, locals, params }) => {
		const { kid } = await stored(locals, params);
		const { values, input, field } = readMedicationForm(await request.formData(), {
			kind: 'edit',
			zone: kid.timeZone
		});
		const extra = { form: 'save' as const };
		if (!input) return courseFieldFailure(values, field!, extra);
		const r = await locals.api.medications.update(kid.id, params.medication, input);
		if (!r.ok) return courseApiFailure(values, r, extra);
		redirect(303, medicationPath(kid.id));
	},
	delete: async ({ locals, params }) => {
		const { kid } = await stored(locals, params);
		const r = await locals.api.medications.remove(kid.id, params.medication);
		if (!r.ok) return failAction(r, { form: 'delete' as const });
		redirect(303, medicationPath(kid.id));
	}
};
