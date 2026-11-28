import { linkSsoRoute } from '#lib/server/oidc/link.js';
import type { Actions, PageServerLoad } from './$types';

const route = linkSsoRoute('admin');
export const load: PageServerLoad = route.load;
export const actions: Actions = route.actions;
