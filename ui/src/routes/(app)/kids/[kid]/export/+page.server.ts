import { EXPORT_MODULES } from '#lib/constants.js';
import { todayInOrUtc } from '#lib/time-zone.js';
import { setKidCookie } from '#lib/server/cookie.js';
import { EXPORT_MESSAGES, exportDefaults, readExportSearch } from '#lib/server/export-form.js';
import { failPage } from '#lib/server/outcome.js';
import type { PageServerLoad } from './$types';

// After a refused export the redirect's values fill the form; without them the defaults do.
export const load: PageServerLoad = async ({ locals, params, url, cookies }) => {
	const [kid, modules] = await Promise.all([
		locals.api.getKid(params.kid),
		locals.api.modules(params.kid)
	]);
	if (!kid.ok) failPage(kid);
	if (!modules.ok) failPage(modules);
	setKidCookie(cookies, kid.data.id);
	const defaults = exportDefaults(kid.data.birthDate, kid.data.timeZone);
	const search = readExportSearch(url.searchParams);
	const on = EXPORT_MODULES.filter((m) => modules.data.some((s) => s.module === m && s.enabled));
	const ticked = search.modules ?? on;
	return {
		kid: kid.data,
		today: todayInOrUtc(kid.data.timeZone),
		from: search.from ?? defaults.from,
		to: search.to ?? defaults.to,
		on: on.map((id) => ({ id, checked: ticked.includes(id) })),
		off: EXPORT_MODULES.filter((m) => !on.includes(m)).map((id) => ({
			id,
			checked: ticked.includes(id)
		})),
		error: search.error && { field: search.error, message: EXPORT_MESSAGES[search.error] }
	};
};
