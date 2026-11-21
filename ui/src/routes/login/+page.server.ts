import { fail, redirect } from '@sveltejs/kit';
import { createApi } from '#lib/server/api.js';
import { setSessionCookie } from '#lib/server/cookie.js';
import { getEnv } from '#lib/server/env.js';
import { landing } from '#lib/server/landing.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = ({ locals, url }) => {
	if (locals.session) redirect(303, landing(locals.session.kind, url.searchParams.get('next')));
};

export const actions: Actions = {
	default: async ({ request, cookies, url, getClientAddress }) => {
		const form = await request.formData();
		const username = String(form.get('username') ?? '');
		const password = String(form.get('password') ?? '');
		const api = createApi(getEnv(), fetch, undefined, getClientAddress());
		const r = await api.login(username, password);
		if (!r.ok) {
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
