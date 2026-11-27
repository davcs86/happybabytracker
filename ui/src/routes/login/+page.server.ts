import { fail, redirect } from '@sveltejs/kit';
import { createApi } from '#lib/server/api.js';
import { deviceLabel } from '#lib/server/device-label.js';
import { setSessionCookie } from '#lib/server/cookie.js';
import { getEnv } from '#lib/server/env.js';
import { landing } from '#lib/server/landing.js';
import { ssoOffered } from '#lib/server/oidc/start.js';
import type { AuthMode } from '#lib/types.js';
import type { Actions, PageServerLoad } from './$types';

// A failed mode read keeps the password form: the action's own 503 alert must stay on this page.
export const load: PageServerLoad = async ({ locals, url }) => {
	if (locals.session) redirect(303, landing(locals.session.kind, url.searchParams.get('next')));
	const r = await locals.api.authMode();
	const mode: AuthMode = r.ok ? r.data : 'local';
	return { mode, sso: r.ok && ssoOffered(mode, getEnv().oidc) };
};

export const actions: Actions = {
	default: async ({ request, cookies, url, getClientAddress }) => {
		const form = await request.formData();
		const username = String(form.get('username') ?? '');
		const password = String(form.get('password') ?? '');
		const usernameError = username.trim() ? null : 'Enter your username';
		const passwordError = password ? null : 'Enter your password';
		if (usernameError || passwordError)
			return fail(400, { username, usernameError, passwordError });
		const api = createApi(getEnv(), fetch, undefined, getClientAddress());
		const r = await api.login(
			username,
			password,
			deviceLabel(request.headers.get('user-agent') ?? '')
		);
		if (!r.ok) {
			if (r.code === 'mode_refused') {
				return fail(403, { username, message: 'Password sign-in is turned off' });
			}
			if (r.reason === 'rate_limited') {
				return fail(429, { username, message: 'Too many attempts, try again in a minute' });
			}
			if (r.code === 'invalid_credentials' || r.reason === 'invalid') {
				return fail(400, { username, message: 'Incorrect username or password' });
			}
			if (r.reason === 'unavailable') {
				return fail(503, { username, message: 'The app cannot reach its server, try again soon' });
			}
			return fail(500, { username, message: 'Something went wrong, try again' });
		}
		setSessionCookie(cookies, r.data.token, r.data.kind);
		redirect(303, landing(r.data.kind, url.searchParams.get('next')));
	}
};
