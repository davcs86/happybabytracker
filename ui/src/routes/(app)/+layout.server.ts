import { formatAge } from '#lib/format.js';
import { todayInOrUtc } from '#lib/time-zone.js';
import { currentKidId } from '#lib/server/current-kid.js';
import { failPage } from '#lib/server/outcome.js';
import type { LayoutServerLoad } from './$types';

export const load: LayoutServerLoad = async ({ locals, cookies, params }) => {
	const r = await locals.api.listKids();
	if (!r.ok) failPage(r);
	const kidId = params.kid ?? currentKidId(r.data, cookies);
	const kid = r.data.find((k) => k.id === kidId) ?? null;
	// The switch is hidden-or-shown only: a failed read hides nothing.
	const modules = kid ? await locals.api.modules(kid.id) : null;
	return {
		kids: r.data,
		kid,
		modules: modules?.ok ? modules.data : null,
		ages: Object.fromEntries(
			r.data.map((k) => [k.id, formatAge(k.birthDate, todayInOrUtc(k.timeZone))])
		)
	};
};
