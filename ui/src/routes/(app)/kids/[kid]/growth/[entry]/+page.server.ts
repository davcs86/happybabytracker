import { redirect } from '@sveltejs/kit';
import { growthMeasurePath } from '#lib/paths.js';
import { todayInOrUtc } from '#lib/time-zone.js';
import { setKidCookie } from '#lib/server/cookie.js';
import {
	firstMeasure,
	growthApiFailure,
	growthEditValues,
	growthFieldFailure,
	METRIC_UNITS,
	readGrowthForm
} from '#lib/server/growth-form.js';
import { failAction, failPage } from '#lib/server/outcome.js';
import type { Actions, PageServerLoad } from './$types';

// The kid and the row, read together: no grant, an unknown row and another kid's row are the same 404 page.
async function stored(locals: App.Locals, params: { kid: string; entry: string }) {
	const [kid, entry] = await Promise.all([
		locals.api.getKid(params.kid),
		locals.api.getGrowth(params.kid, params.entry)
	]);
	if (!kid.ok) failPage(kid);
	if (!entry.ok) failPage(entry);
	return { kid: kid.data, entry: entry.data };
}

export const load: PageServerLoad = async ({ locals, params, cookies, parent }) => {
	const [{ kid, entry }, { session }] = await Promise.all([stored(locals, params), parent()]);
	setKidCookie(cookies, kid.id);
	return {
		kid,
		today: todayInOrUtc(kid.timeZone),
		values: growthEditValues(entry, session?.parent?.units ?? METRIC_UNITS)
	};
};

export const actions: Actions = {
	save: async ({ request, locals, params }) => {
		const { kid } = await stored(locals, params);
		const { values, input, field, message } = readGrowthForm(await request.formData(), {
			kind: 'edit'
		});
		const extra = { form: 'save' as const };
		if (!input) return growthFieldFailure(values, field!, extra, message);
		const r = await locals.api.updateGrowth(kid.id, params.entry, input);
		if (!r.ok) return growthApiFailure(values, r, extra);
		redirect(303, growthMeasurePath(kid.id, firstMeasure(r.data)));
	},
	delete: async ({ locals, params }) => {
		const { kid, entry } = await stored(locals, params);
		const r = await locals.api.deleteGrowth(kid.id, entry.id);
		if (!r.ok) return failAction(r, { form: 'delete' as const });
		redirect(303, growthMeasurePath(kid.id, firstMeasure(entry)));
	}
};
