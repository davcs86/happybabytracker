import { getEnv } from '#lib/server/env.js';
import { ssoLinkedOf } from '#lib/server/oidc/link.js';
import { ssoOffered } from '#lib/server/oidc/start.js';
import type { AuthMode } from '#lib/types.js';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals }) => {
	const r = await locals.api.authMode();
	const mode: AuthMode = r.ok ? r.data : 'local';
	return {
		mode,
		sso: r.ok && ssoOffered(mode, getEnv().oidc),
		username: locals.session?.admin?.username ?? '',
		linked: ssoLinkedOf(locals.session)
	};
};
