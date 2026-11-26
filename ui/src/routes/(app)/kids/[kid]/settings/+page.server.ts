import { fail, redirect } from '@sveltejs/kit';
import { KID_COOKIE, PARENT_HOME } from '#lib/server/constants.js';
import { clearKidCookie, setKidCookie } from '#lib/server/cookie.js';
import {
	kidApiFailure,
	kidFieldFailure,
	kidValues,
	readKidForm,
	scheduleOptions,
	sourceOptions
} from '#lib/server/kid-form.js';
import { failAction, failPage } from '#lib/server/outcome.js';
import { timeZoneOptions } from '#lib/server/time-zones.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, params, cookies }) => {
	const [r, sources, schemas, counts, access] = await Promise.all([
		locals.api.getKid(params.kid),
		locals.api.growthSources(),
		locals.api.vaccineSchemas(),
		locals.api.kidCounts(params.kid),
		locals.api.listAccess(params.kid)
	]);
	if (!r.ok) failPage(r);
	if (r.data.role !== 'guardian') failPage({ ok: false, status: 403, reason: 'forbidden' });
	if (!sources.ok) failPage(sources);
	if (!schemas.ok) failPage(schemas);
	setKidCookie(cookies, r.data.id);
	return {
		kid: r.data,
		values: kidValues(r.data),
		zones: timeZoneOptions(r.data.timeZone),
		sources: sourceOptions(sources.data),
		schedules: scheduleOptions(schemas.data),
		// The delete list's numbers: a failed read leaves that line uncounted.
		counts: {
			entries: counts.ok ? counts.data.entries : null,
			people: access.ok ? access.data.length : null
		}
	};
};

export const actions: Actions = {
	save: async ({ request, locals, params }) => {
		const { values, input, field } = readKidForm(await request.formData());
		const extra = { form: 'save' as const };
		if (!input) return kidFieldFailure(values, field!, extra);
		const r = await locals.api.updateKid(params.kid, input);
		if (!r.ok) return kidApiFailure(values, r, extra);
		return { form: 'save' as const, saved: true };
	},
	delete: async ({ request, locals, params, cookies }) => {
		const confirmName = String((await request.formData()).get('confirm_name') ?? '');
		const r = await locals.api.deleteKid(params.kid, confirmName);
		if (!r.ok) {
			if (r.reason === 'invalid' && r.field === 'confirm_name') {
				return fail(400, { form: 'delete' as const, confirmName, mismatch: true });
			}
			return failAction(r, { form: 'delete' as const });
		}
		if (cookies.get(KID_COOKIE) === params.kid) clearKidCookie(cookies);
		redirect(303, PARENT_HOME);
	}
};
