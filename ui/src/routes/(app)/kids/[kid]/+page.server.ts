import { formatAge, todayIn } from '#lib/format.js';
import { setKidCookie } from '#lib/server/cookie.js';
import { failPage } from '#lib/server/outcome.js';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, params, cookies }) => {
	const r = await locals.api.getKid(params.kid);
	if (!r.ok) failPage(r);
	setKidCookie(cookies, r.data.id);
	return { kid: r.data, age: formatAge(r.data.birthDate, todayIn(r.data.timeZone)) };
};
