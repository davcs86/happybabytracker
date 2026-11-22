import { redirect } from '@sveltejs/kit';
import { timelinePath } from '#lib/paths.js';
import { PARENT_HOME } from '#lib/server/constants.js';
import { currentKidId } from '#lib/server/current-kid.js';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ parent, cookies }) => {
	const { kids } = await parent();
	const id = currentKidId(kids, cookies);
	redirect(303, id ? timelinePath(id) : PARENT_HOME);
};
