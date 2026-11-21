import { redirect } from '@sveltejs/kit';
import { kidApiFailure, kidFieldFailure, kidValues, readKidForm } from '#lib/server/kid-form.js';
import { timeZoneOptions } from '#lib/server/time-zones.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = () => ({ zones: timeZoneOptions(), values: kidValues() });

export const actions: Actions = {
	default: async ({ request, locals }) => {
		const { values, input, field } = readKidForm(await request.formData());
		if (!input) return kidFieldFailure(values, field!, {});
		const r = await locals.api.createKid(input);
		if (!r.ok) return kidApiFailure(values, r, {});
		redirect(303, `/kids/${r.data.id}`);
	}
};
