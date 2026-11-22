import { MEASURES } from '#lib/constants.js';
import { formatAge } from '#lib/format.js';
import { todayInOrUtc } from '#lib/time-zone.js';
import { setKidCookie } from '#lib/server/cookie.js';
import { failPage } from '#lib/server/outcome.js';
import type { Measure, Units } from '#lib/types.js';
import type { PageServerLoad } from './$types';

const DEFAULT_MEASURE: Measure = 'weight';
const METRIC: Pick<Units, 'weight' | 'length'> = { weight: 'kg', length: 'cm' };

// An unknown measure shows the default chart.
const readMeasure = (v: string | null): Measure => MEASURES.find((m) => m === v) ?? DEFAULT_MEASURE;

export const load: PageServerLoad = async ({ locals, params, url, cookies, parent }) => {
	const measure = readMeasure(url.searchParams.get('measure'));
	const [kid, rows, curves, sources, { session }] = await Promise.all([
		locals.api.getKid(params.kid),
		locals.api.listGrowth(params.kid),
		locals.api.curves(params.kid, { measure }),
		locals.api.growthSources(),
		parent()
	]);
	if (!kid.ok) failPage(kid);
	if (!rows.ok) failPage(rows);
	if (!curves.ok) failPage(curves);
	if (!sources.ok) failPage(sources);
	setKidCookie(cookies, kid.data.id);
	const units = session?.parent?.units;
	return {
		kid: kid.data,
		age: formatAge(kid.data.birthDate, todayInOrUtc(kid.data.timeZone)),
		measure,
		units: units ? { weight: units.weight, length: units.length } : METRIC,
		rows: rows.data,
		curves: curves.data,
		source: sources.data.find((s) => s.id === curves.data.source) ?? null
	};
};
