import { redirect } from '@sveltejs/kit';
import { clearSessionCookie } from '#lib/server/cookie.js';
import type { RequestHandler } from './$types';

// Signs out locally even when the API cannot be reached.
export const POST: RequestHandler = async ({ cookies, locals }) => {
	if (locals.session) await locals.api.logout();
	clearSessionCookie(cookies);
	redirect(303, '/login');
};
