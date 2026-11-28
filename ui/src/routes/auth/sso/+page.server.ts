import { fail, redirect } from '@sveltejs/kit';
import { safeNext } from '#lib/server/next.js';
import { SSO_UNAVAILABLE, ssoBack, startSso } from '#lib/server/oidc/start.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = () => ({ back: ssoBack('login', null, null) });

export const actions: Actions = {
	default: async (event) => {
		const form = await event.request.formData();
		const mode = await event.locals.api.authMode();
		if (!mode.ok || mode.data === 'local') return fail(503, { message: SSO_UNAVAILABLE });
		const r = await startSso(event, 'login', { next: safeNext(String(form.get('next') ?? '')) });
		if (!r.ok) return fail(503, { message: r.message });
		// The issuer's authorization endpoint, from its discovery document.
		redirect(303, r.location, { external: true });
	}
};
