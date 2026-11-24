import { redirect } from '@sveltejs/kit';
import { milestonesPath } from '#lib/paths.js';
import { todayInOrUtc } from '#lib/time-zone.js';
import { setKidCookie } from '#lib/server/cookie.js';
import {
	milestoneApiFailure,
	milestoneEditValues,
	milestoneFieldFailure,
	readMilestoneForm
} from '#lib/server/milestone-form.js';
import { failAction, failPage } from '#lib/server/outcome.js';
import type { Actions, PageServerLoad } from './$types';

// The kid and the row, read together: no grant, an unknown row and another kid's row are the same 404 page.
async function stored(locals: App.Locals, params: { kid: string; entry: string }) {
	const [kid, entry] = await Promise.all([
		locals.api.getKid(params.kid),
		locals.api.milestones.get(params.kid, params.entry)
	]);
	if (!kid.ok) failPage(kid);
	if (!entry.ok) failPage(entry);
	return { kid: kid.data, entry: entry.data };
}

export const load: PageServerLoad = async ({ locals, params, cookies }) => {
	const { kid, entry } = await stored(locals, params);
	setKidCookie(cookies, kid.id);
	return {
		kid,
		today: todayInOrUtc(kid.timeZone),
		custom: entry.item === null,
		text: entry.text,
		values: milestoneEditValues(entry)
	};
};

export const actions: Actions = {
	save: async ({ request, locals, params }) => {
		const { kid, entry } = await stored(locals, params);
		const { values, input, field, message } = readMilestoneForm(
			await request.formData(),
			{ kind: 'edit', custom: entry.item === null },
			kid.timeZone
		);
		const extra = { form: 'save' as const };
		if (!input) return milestoneFieldFailure(values, field!, extra, message);
		const r = await locals.api.milestones.update(kid.id, entry.id, input);
		if (!r.ok) return milestoneApiFailure(values, r, extra);
		redirect(303, milestonesPath(kid.id));
	},
	delete: async ({ locals, params }) => {
		const { kid, entry } = await stored(locals, params);
		const r = await locals.api.milestones.remove(kid.id, entry.id);
		if (!r.ok) return failAction(r, { form: 'delete' as const });
		redirect(303, milestonesPath(kid.id));
	}
};
