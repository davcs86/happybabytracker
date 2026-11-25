import { redirect } from '@sveltejs/kit';
import { setThemeCookie } from '#lib/server/cookie.js';
import { parseTheme, returnPath } from '#lib/server/theme.js';
import type { RequestHandler } from './$types';

// Works without JavaScript: the page posts here and comes back.
export const POST: RequestHandler = async ({ request, cookies }) => {
	const form = await request.formData();
	const theme = parseTheme(form.get('theme'));
	if (theme) setThemeCookie(cookies, theme);
	redirect(303, returnPath(form.get('return')));
};
