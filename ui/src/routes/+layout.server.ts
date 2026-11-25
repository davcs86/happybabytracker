import { THEME_COOKIE } from '#lib/server/constants.js';
import { themeOrAuto } from '#lib/server/theme.js';
import type { LayoutServerLoad } from './$types';

export const load: LayoutServerLoad = ({ locals, cookies }) => ({
	session: locals.session,
	theme: themeOrAuto(cookies.get(THEME_COOKIE))
});
