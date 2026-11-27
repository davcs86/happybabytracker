import { redirect } from '@sveltejs/kit';
import { MEASURES } from '#lib/constants.js';
import { formatAge } from '#lib/format.js';
import { ageHeader } from '#lib/growth.js';
import { todayInOrUtc } from '#lib/time-zone.js';
import { setKidCookie } from '#lib/server/cookie.js';
import { METRIC_UNITS } from '#lib/server/growth-form.js';
import { applySystem, type UnitSystem } from '#lib/growth-units.js';
import { failAction, failPage } from '#lib/server/outcome.js';
import type { Measure } from '#lib/types.js';
import type { Actions, PageServerLoad } from './$types';

const DEFAULT_MEASURE: Measure = 'weight';

// An unknown measure shows the default chart.
const readMeasure = (v: string | null): Measure => MEASURES.find((m) => m === v) ?? DEFAULT_MEASURE;

export const load: PageServerLoad = async ({ locals, params, url, cookies, parent }) => {
	const measure = readMeasure(url.searchParams.get('measure'));
	const [kid, home, rows, curves, sources, { session }] = await Promise.all([
		locals.api.getKid(params.kid),
		locals.api.home(params.kid),
		locals.api.listGrowth(params.kid),
		locals.api.curves(params.kid, { measure }),
		locals.api.growthSources(),
		parent()
	]);
	if (!kid.ok) failPage(kid);
	if (!home.ok) failPage(home);
	if (!rows.ok) failPage(rows);
	if (!curves.ok) failPage(curves);
	if (!sources.ok) failPage(sources);
	setKidCookie(cookies, kid.data.id);
	const units = session?.parent?.units;
	const sel = url.searchParams.get('sel');
	const today = todayInOrUtc(kid.data.timeZone);
	return {
		kid: kid.data,
		age: ageHeader(formatAge(kid.data.birthDate, today), home.data.correctedBirthDate, today),
		measure,
		units: units ? { weight: units.weight, length: units.length } : METRIC_UNITS,
		rows: rows.data,
		// A row that is not there is no selection.
		selected: rows.data.find((r) => r.id === sel)?.id ?? null,
		curves: curves.data,
		source: sources.data.find((s) => s.id === curves.data.source) ?? null
	};
};

const isSystem = (v: unknown): v is UnitSystem => v === 'metric' || v === 'imperial';

export const actions: Actions = {
	// The account's weight and length units: one switch for both, from the page they are read on.
	units: async ({ request, locals, url }) => {
		const system = (await request.formData()).get('system');
		const parent = locals.session?.parent;
		if (!isSystem(system) || !parent)
			return failAction({ ok: false, status: 400, reason: 'invalid' }, { form: 'units' as const });
		const r = await locals.api.updateProfile(parent.displayName, applySystem(parent.units, system));
		if (!r.ok) return failAction(r, { form: 'units' as const });
		redirect(303, `${url.pathname}${url.search}`);
	}
};
