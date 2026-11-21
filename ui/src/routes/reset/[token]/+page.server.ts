import { setPasswordRoute } from '#lib/server/set-password.js';
import type { Actions, PageServerLoad } from './$types';

const route = setPasswordRoute('reset');

export const load: PageServerLoad = route.load;
export const actions: Actions = route.actions;
