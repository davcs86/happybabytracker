import { redirect } from '@sveltejs/kit';
import { growthMeasurePath } from '#lib/paths.js';
import { todayInOrUtc } from '#lib/time-zone.js';
import { setKidCookie } from '#lib/server/cookie.js';
import {
	firstMeasure,
	growthApiFailure,
	growthCreateValues,
	growthFieldFailure,
	METRIC_UNITS,
	readGrowthForm
} from '#lib/server/growth-form.js';
import { failPage } from '#lib/server/outcome.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, params, cookies, parent }) => {
	const [kid, { session }] = await Promise.all([locals.api.getKid(params.kid), parent()]);
	if (!kid.ok) failPage(kid);
	setKidCookie(cookies, kid.data.id);
	const today = todayInOrUtc(kid.data.timeZone);
	return {
		kid: kid.data,
		today,
		values: growthCreateValues(today, session?.parent?.units ?? METRIC_UNITS)
	};
};

export const actions: Actions = {
	save: async ({ request, locals, params }) => {
		const kid = await locals.api.getKid(params.kid);
		if (!kid.ok) failPage(kid);
		const { values, input, field, message } = readGrowthForm(await request.formData(), {
			kind: 'create'
		});
		const extra = { form: 'save' as const };
		if (!input) return growthFieldFailure(values, field!, extra, message);
		const r = await locals.api.createGrowth(kid.data.id, input);
		if (!r.ok) return growthApiFailure(values, r, extra);
		redirect(303, growthMeasurePath(kid.data.id, firstMeasure(r.data)));
	}
};
