import { redirect } from '@sveltejs/kit';
import { vaccinesPath } from '#lib/paths.js';
import { todayInOrUtc } from '#lib/time-zone.js';
import { setKidCookie } from '#lib/server/cookie.js';
import { failAction, failPage } from '#lib/server/outcome.js';
import {
	readVaccineForm,
	vaccineApiFailure,
	vaccineEditValues,
	vaccineFieldFailure
} from '#lib/server/vaccine-form.js';
import type { Actions, PageServerLoad } from './$types';

// The kid and the dose, read together: no grant, an unknown dose and another kid's dose are the same 404 page.
async function stored(locals: App.Locals, params: { kid: string; administration: string }) {
	const [kid, administration] = await Promise.all([
		locals.api.getKid(params.kid),
		locals.api.getVaccination(params.kid, params.administration)
	]);
	if (!kid.ok) failPage(kid);
	if (!administration.ok) failPage(administration);
	return { kid: kid.data, administration: administration.data };
}

export const load: PageServerLoad = async ({ locals, params, cookies }) => {
	const [{ kid, administration }, catalogue] = await Promise.all([
		stored(locals, params),
		locals.api.vaccineCatalogue()
	]);
	if (!catalogue.ok) failPage(catalogue);
	setKidCookie(cookies, kid.id);
	return {
		kid,
		today: todayInOrUtc(kid.timeZone),
		catalogue: catalogue.data,
		values: vaccineEditValues(administration)
	};
};

export const actions: Actions = {
	save: async ({ request, locals, params }) => {
		const { kid } = await stored(locals, params);
		const { values, input, field, message } = readVaccineForm(await request.formData(), {
			kind: 'edit'
		});
		const extra = { form: 'save' as const };
		if (!input) return vaccineFieldFailure(values, field!, extra, message);
		const r = await locals.api.updateVaccination(kid.id, params.administration, input);
		if (!r.ok) return vaccineApiFailure(values, r, extra);
		redirect(303, vaccinesPath(kid.id));
	},
	delete: async ({ locals, params }) => {
		const { kid, administration } = await stored(locals, params);
		const r = await locals.api.deleteVaccination(kid.id, administration.id);
		if (!r.ok) return failAction(r, { form: 'delete' as const });
		redirect(303, vaccinesPath(kid.id));
	}
};
