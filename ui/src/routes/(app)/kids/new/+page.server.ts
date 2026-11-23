import { redirect } from '@sveltejs/kit';
import {
	kidApiFailure,
	kidFieldFailure,
	kidValues,
	readKidForm,
	scheduleOptions,
	sourceOptions
} from '#lib/server/kid-form.js';
import { failPage } from '#lib/server/outcome.js';
import { timeZoneOptions } from '#lib/server/time-zones.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals }) => {
	const [sources, schemas] = await Promise.all([
		locals.api.growthSources(),
		locals.api.vaccineSchemas()
	]);
	if (!sources.ok) failPage(sources);
	if (!schemas.ok) failPage(schemas);
	return {
		zones: timeZoneOptions(),
		values: kidValues(),
		sources: sourceOptions(sources.data),
		schedules: scheduleOptions(schemas.data)
	};
};

export const actions: Actions = {
	default: async ({ request, locals }) => {
		const { values, input, field } = readKidForm(await request.formData());
		if (!input) return kidFieldFailure(values, field!, {});
		const r = await locals.api.createKid(input);
		if (!r.ok) return kidApiFailure(values, r, {});
		redirect(303, `/kids/${r.data.id}`);
	}
};
