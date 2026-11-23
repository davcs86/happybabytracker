import { error, redirect } from '@sveltejs/kit';
import { medicationPath } from '#lib/paths.js';
import { todayInOrUtc } from '#lib/time-zone.js';
import { ID_PATTERN, NOT_FOUND_MESSAGE } from '#lib/server/constants.js';
import { setKidCookie } from '#lib/server/cookie.js';
import { doseApiFailure, doseFieldFailure, readDoseForm } from '#lib/server/medication-form.js';
import { failPage } from '#lib/server/outcome.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, params, cookies, url }) => {
	const medication = url.searchParams.get('medication');
	const [kid, courses, doses] = await Promise.all([
		locals.api.getKid(params.kid),
		locals.api.medications.list(params.kid),
		locals.api.doses.list(params.kid, medication === null ? {} : { medication })
	]);
	if (!kid.ok) failPage(kid);
	if (!courses.ok) failPage(courses);
	// A malformed course id answers not found without a request.
	if (!doses.ok) failPage(doses);
	const filter = courses.data.find((c) => c.id === medication) ?? null;
	// A well-formed id that is no course of this kid is the same 404 page.
	if (medication !== null && !filter) error(404, NOT_FOUND_MESSAGE);
	setKidCookie(cookies, kid.data.id);
	return {
		kid: kid.data,
		today: todayInOrUtc(kid.data.timeZone),
		courses: courses.data,
		filter,
		doses: doses.data.items
	};
};

export const actions: Actions = {
	log: async ({ request, locals, params }) => {
		const form = await request.formData();
		const { values, input, field } = readDoseForm(form, { kind: 'log' });
		const extra = { form: 'log' as const };
		if (!input) return doseFieldFailure(values, field!, extra);
		const r = await locals.api.doses.create(params.kid, input);
		if (!r.ok) return doseApiFailure(values, r, extra);
		const filter = form.get('filter');
		const kept =
			typeof filter === 'string' && ID_PATTERN.test(filter) ? `?medication=${filter}` : '';
		redirect(303, medicationPath(params.kid) + kept);
	}
};
