import { redirect } from '@sveltejs/kit';
import { formatAge } from '#lib/format.js';
import { ageHeader } from '#lib/growth.js';
import { milestonesPath } from '#lib/paths.js';
import { todayInOrUtc } from '#lib/time-zone.js';
import { setKidCookie } from '#lib/server/cookie.js';
import {
	milestoneApiFailure,
	milestoneFieldFailure,
	milestoneReachedPath,
	readMilestoneForm,
	readOpenItem
} from '#lib/server/milestone-form.js';
import { failPage } from '#lib/server/outcome.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, params, cookies, url }) => {
	const [kid, home, milestones, modules] = await Promise.all([
		locals.api.getKid(params.kid),
		locals.api.home(params.kid),
		locals.api.milestones.list(params.kid),
		locals.api.modules(params.kid)
	]);
	if (!kid.ok) failPage(kid);
	if (!home.ok) failPage(home);
	if (!milestones.ok) failPage(milestones);
	setKidCookie(cookies, kid.data.id);
	const today = todayInOrUtc(kid.data.timeZone);
	return {
		kid: kid.data,
		age: ageHeader(formatAge(kid.data.birthDate, today), home.data.correctedBirthDate, today),
		milestones: milestones.data,
		open: readOpenItem(url),
		// The off note is advisory: a failed modules call shows no note.
		milestonesOff: modules.ok && modules.data.some((m) => m.module === 'milestones' && !m.enabled)
	};
};

export const actions: Actions = {
	reach: async ({ request, locals, params }) => {
		const kid = await locals.api.getKid(params.kid);
		if (!kid.ok) failPage(kid);
		const { values, input, field, message } = readMilestoneForm(
			await request.formData(),
			{ kind: 'create' },
			kid.data.timeZone
		);
		const extra = { form: 'reach' as const };
		if (!input) return milestoneFieldFailure(values, field!, extra, message);
		const r = await locals.api.milestones.create(kid.data.id, input);
		if (!r.ok) return milestoneApiFailure(values, r, extra);
		const item = input.item;
		redirect(
			303,
			typeof item === 'string'
				? milestoneReachedPath(kid.data.id, item)
				: milestonesPath(kid.data.id)
		);
	}
};
