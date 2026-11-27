import { error, redirect, text } from '@sveltejs/kit';
import type { Handle, ServerInit } from '@sveltejs/kit/hooks';
import { createApi } from '#lib/server/api.js';
import {
	ADMIN_HOME,
	SESSION_COOKIE,
	THEME_COOKIE,
	UNAVAILABLE_MESSAGE
} from '#lib/server/constants.js';
import { clearSessionCookie, setSessionCookie } from '#lib/server/cookie.js';
import { getEnv } from '#lib/server/env.js';
import { themeOrAuto } from '#lib/server/theme.js';

const isHealth = (path: string) => path === '/healthz' || path.startsWith('/healthz/');
const isAdminArea = (path: string) => path === '/admin' || path.startsWith('/admin/');
const isAuth = (path: string) => path.startsWith('/auth/');
// Pages whose URL can carry a one-time token or an authorization code.
const sameOriginReferrer = (path: string) =>
	isAuth(path) || path.startsWith('/invite/') || path.startsWith('/reset/');
const isPublic = (path: string) =>
	path === '/login' ||
	isAuth(path) ||
	path === '/logout' ||
	path === '/theme' ||
	path.startsWith('/invite/') ||
	path.startsWith('/reset/') ||
	isHealth(path);

export const init: ServerInit = () => {
	getEnv();
};

export const handle: Handle = async ({ event, resolve }) => {
	const env = getEnv();
	const path = event.url.pathname;
	event.locals.session = null;
	if (isHealth(path)) {
		event.locals.api = createApi(env, fetch);
		return resolve(event);
	}
	if (event.url.origin !== env.origin) return text('Bad origin', { status: 400 });

	const token = event.cookies.get(SESSION_COOKIE);
	event.locals.api = createApi(env, fetch, token);
	if (token) {
		const r = await event.locals.api.session();
		if (r.ok) {
			event.locals.session = r.data;
			// The API slides a parent session; the cookie follows it.
			if (r.data.kind === 'parent') setSessionCookie(event.cookies, token, 'parent');
		} else if (r.reason === 'unauthorized' && r.code === 'no_session') {
			clearSessionCookie(event.cookies);
		} else if (path !== '/logout') {
			// Keeps the cookie: an unreachable API is not a signed-out user.
			error(503, UNAVAILABLE_MESSAGE);
		}
	}

	const session = event.locals.session;
	if (!session) {
		if (!isPublic(path)) {
			redirect(303, `/login?next=${encodeURIComponent(path + event.url.search)}`);
		}
	} else if (session.kind === 'admin') {
		if (!path.startsWith('/admin/') && path !== '/logout' && !isAuth(path)) {
			redirect(303, ADMIN_HOME);
		}
	} else if (isAdminArea(path)) {
		error(404, 'Not found');
	}

	const theme = themeOrAuto(event.cookies.get(THEME_COOKIE));
	const response = await resolve(event, {
		transformPageChunk: ({ html }) => html.replace('%hbt.theme%', theme)
	});
	if (!path.startsWith('/_app/')) response.headers.set('cache-control', 'no-store');
	if (sameOriginReferrer(path)) response.headers.set('referrer-policy', 'same-origin');
	return response;
};
