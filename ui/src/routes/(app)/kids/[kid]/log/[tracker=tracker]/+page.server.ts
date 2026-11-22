import { redirect } from '@sveltejs/kit';
import { createValues, entryInstant, localDate, trackerOf } from '#lib/entry-edit.js';
import { timelineDayPath } from '#lib/paths.js';
import { setKidCookie } from '#lib/server/cookie.js';
import { entryApiFailure, entryFieldFailure } from '#lib/server/entry-failure.js';
import { readEntryForm } from '#lib/server/entry-form.js';
import { failPage } from '#lib/server/outcome.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, params, cookies, parent }) => {
	const [kid, { session }] = await Promise.all([locals.api.getKid(params.kid), parent()]);
	if (!kid.ok) failPage(kid);
	setKidCookie(cookies, kid.data.id);
	const kind = trackerOf(params.tracker);
	const unit = session?.parent?.units.volume ?? 'ml';
	return {
		kid: kid.data,
		kind,
		values: createValues(kind, new Date().toISOString(), kid.data.timeZone, unit)
	};
};

export const actions: Actions = {
	save: async ({ request, locals, params }) => {
		const kid = await locals.api.getKid(params.kid);
		if (!kid.ok) failPage(kid);
		const kind = trackerOf(params.tracker);
		const { values, input, field, message } = readEntryForm(
			kind,
			await request.formData(),
			kid.data.timeZone,
			{ kind: 'create' }
		);
		const extra = { form: 'save' as const };
		if (!input) return entryFieldFailure(values, field!, extra, message);
		const r = await locals.api.tracker(kind).create(params.kid, input);
		if (!r.ok) return entryApiFailure(kind, values, r, extra);
		const date = localDate(entryInstant(kind, r.data), kid.data.timeZone);
		redirect(303, timelineDayPath(kid.data.id, date));
	}
};
