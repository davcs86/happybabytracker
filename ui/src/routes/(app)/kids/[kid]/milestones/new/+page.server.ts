import { redirect } from '@sveltejs/kit';
import { milestonesPath } from '#lib/paths.js';
import { todayInOrUtc } from '#lib/time-zone.js';
import { setKidCookie } from '#lib/server/cookie.js';
import {
	milestoneApiFailure,
	milestoneCreateValues,
	milestoneFieldFailure,
	readMilestoneForm
} from '#lib/server/milestone-form.js';
import { failPage } from '#lib/server/outcome.js';
import type { Actions, PageServerLoad } from './$types';

// An item the kid's lists do not show (unknown, retired, lists hidden) gives the custom form.
async function chosen(locals: App.Locals, kidId: string, url: URL) {
	const [kid, milestones] = await Promise.all([
		locals.api.getKid(kidId),
		locals.api.milestones.list(kidId)
	]);
	if (!kid.ok) failPage(kid);
	if (!milestones.ok) failPage(milestones);
	const id = url.searchParams.get('item');
	const item = milestones.data.checklists.flatMap((c) => c.items).find((i) => i.id === id) ?? null;
	return { kid: kid.data, item: item && { id: item.id, text: item.text } };
}

export const load: PageServerLoad = async ({ locals, params, url, cookies }) => {
	const { kid, item } = await chosen(locals, params.kid, url);
	setKidCookie(cookies, kid.id);
	const today = todayInOrUtc(kid.timeZone);
	return { kid, item, today, values: milestoneCreateValues(today, item?.id) };
};

export const actions: Actions = {
	save: async ({ request, locals, params }) => {
		const kid = await locals.api.getKid(params.kid);
		if (!kid.ok) failPage(kid);
		const { values, input, field, message } = readMilestoneForm(
			await request.formData(),
			{ kind: 'create' },
			kid.data.timeZone
		);
		const extra = { form: 'save' as const };
		if (!input) return milestoneFieldFailure(values, field!, extra, message);
		const r = await locals.api.milestones.create(kid.data.id, input);
		if (!r.ok) return milestoneApiFailure(values, r, extra);
		redirect(303, milestonesPath(kid.data.id));
	}
};
