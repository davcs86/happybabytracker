import { failAction, failPage } from '#lib/server/outcome.js';
import { setKidCookie } from '#lib/server/cookie.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, params, cookies }) => {
	const kid = await locals.api.getKid(params.kid);
	if (!kid.ok) failPage(kid);
	if (kid.data.role !== 'guardian') failPage({ ok: false, status: 403, reason: 'forbidden' });
	const modules = await locals.api.modules(params.kid);
	if (!modules.ok) failPage(modules);
	setKidCookie(cookies, kid.data.id);
	return { kid: kid.data, modules: modules.data };
};

export const actions: Actions = {
	// Only a module whose state differs from the stored one is written.
	save: async ({ request, locals, params }) => {
		const on = new Set((await request.formData()).getAll('module').map(String));
		const current = await locals.api.modules(params.kid);
		if (!current.ok) return failAction(current, { form: 'save' as const });
		for (const m of current.data) {
			if (m.enabled === on.has(m.module)) continue;
			const r = await locals.api.setModule(params.kid, m.module, !m.enabled);
			if (!r.ok) return failAction(r, { form: 'save' as const });
		}
		return { form: 'save' as const, saved: true };
	}
};
