import { INVALID_CURSOR } from '#lib/timeline.js';
import { rfc3339 } from '#lib/time-zone.js';
import { setKidCookie } from '#lib/server/cookie.js';
import { failPage } from '#lib/server/outcome.js';
import type { PageServerLoad } from './$types';

// Upcoming is [now, horizon) soonest first; Past is before now, newest first, paged by its cursor.
export const load: PageServerLoad = async ({ locals, params, cookies, url }) => {
	const cursor = url.searchParams.get('cursor') ?? undefined;
	const now = rfc3339(Date.now());
	const [kid, upcoming, past, modules] = await Promise.all([
		locals.api.getKid(params.kid),
		locals.api.visits.list(params.kid, { from: now, order: 'asc' }),
		locals.api.visits.list(params.kid, { to: now, cursor }),
		locals.api.modules(params.kid)
	]);
	if (!kid.ok) failPage(kid);
	if (!upcoming.ok) failPage(upcoming);
	// The API alone judges the cursor.
	const badCursor = !past.ok && past.reason === 'invalid' && past.field === 'cursor';
	if (!past.ok && !badCursor) failPage(past);
	setKidCookie(cookies, kid.data.id);
	return {
		kid: kid.data,
		paged: cursor !== undefined,
		upcoming: upcoming.data.items,
		past: past.ok ? past.data.items : [],
		older: past.ok ? past.data.nextCursor : null,
		cursorError: badCursor ? INVALID_CURSOR : null,
		// The switched-off card is advisory: a failed modules call shows the list.
		visitsOff: modules.ok && modules.data.some((m) => m.module === 'visits' && !m.enabled)
	};
};
