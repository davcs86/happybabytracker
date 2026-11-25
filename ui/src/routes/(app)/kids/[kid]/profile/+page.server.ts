import { setKidCookie } from '#lib/server/cookie.js';
import { failPage } from '#lib/server/outcome.js';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, params, cookies }) => {
	const [kid, grants] = await Promise.all([
		locals.api.getKid(params.kid),
		locals.api.listAccess(params.kid)
	]);
	if (!kid.ok) failPage(kid);
	if (kid.data.role !== 'guardian') failPage({ ok: false, status: 403, reason: 'forbidden' });
	if (!grants.ok) failPage(grants);
	setKidCookie(cookies, kid.data.id);
	return { kid: kid.data, grants: grants.data, me: locals.session?.parent?.id };
};
