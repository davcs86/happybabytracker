import { redirect } from '@sveltejs/kit';
import { visitPath } from '#lib/paths.js';
import { todayInOrUtc } from '#lib/time-zone.js';
import type { ModuleId } from '#lib/types.js';
import { setKidCookie } from '#lib/server/cookie.js';
import { METRIC_UNITS } from '#lib/server/growth-form.js';
import { failPage } from '#lib/server/outcome.js';
import {
	outcomeApiFailure,
	outcomeBlanks,
	outcomeCreateValues,
	outcomeCurrent,
	outcomeFieldFailure,
	outcomeStale,
	readOutcomeForm,
	type OutcomeSections
} from '#lib/server/outcome-form.js';
import { VISIT_REFUSALS } from '#lib/server/visit-form.js';
import type { Actions, PageServerLoad } from './$types';

// The kid, the visit and the kid's modules, read together: a visit of another kid is the same 404 page.
async function stored(locals: App.Locals, params: { kid: string; visit: string }) {
	const [kid, visit, modules] = await Promise.all([
		locals.api.getKid(params.kid),
		locals.api.visits.get(params.kid, params.visit),
		locals.api.modules(params.kid)
	]);
	if (!kid.ok) failPage(kid);
	if (!visit.ok) failPage(visit);
	if (!modules.ok) failPage(modules);
	const on = (m: ModuleId) => modules.data.some((s) => s.module === m && s.enabled);
	const sections: OutcomeSections = {
		growth: on('growth'),
		vaccinations: on('vaccination'),
		medications: on('medication')
	};
	return { kid: kid.data, visit: visit.data, sections };
}

export const load: PageServerLoad = async ({ locals, params, cookies, parent }) => {
	const [{ kid, visit, sections }, { session }] = await Promise.all([
		stored(locals, params),
		parent()
	]);
	setKidCookie(cookies, kid.id);
	const catalogue = sections.vaccinations ? await locals.api.vaccineCatalogue() : null;
	if (catalogue && !catalogue.ok) failPage(catalogue);
	const blanks = outcomeBlanks(visit, kid.timeZone, session?.parent?.units ?? METRIC_UNITS);
	return {
		kid,
		visit,
		sections,
		catalogue: catalogue?.data ?? null,
		today: todayInOrUtc(kid.timeZone),
		blanks,
		values: outcomeCreateValues(blanks, visit),
		current: outcomeCurrent(visit),
		notPast: visit.status === 'past' ? null : VISIT_REFUSALS.not_past
	};
};

export const actions: Actions = {
	save: async ({ request, locals, params }) => {
		const { kid, visit, sections } = await stored(locals, params);
		const { values, input, field, message } = readOutcomeForm(await request.formData(), {
			zone: kid.timeZone,
			sections
		});
		const extra = { form: 'save' as const };
		const current = outcomeCurrent(visit);
		if (!input) {
			if (field === 'revision') return outcomeStale(values, current, extra);
			return outcomeFieldFailure(values, field!, message, extra);
		}
		const r = await locals.api.visits.outcome(kid.id, visit.id, input);
		if (!r.ok) return outcomeApiFailure(values, r, current, extra);
		redirect(303, visitPath(kid.id, visit.id));
	}
};
