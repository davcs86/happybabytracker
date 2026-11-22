import { redirect } from '@sveltejs/kit';
import { growthPath } from '#lib/paths.js';
import { PARENT_HOME } from '#lib/server/constants.js';
import { currentKidId } from '#lib/server/current-kid.js';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ parent, cookies }) => {
	const { kids } = await parent();
	const id = currentKidId(kids, cookies);
	redirect(303, id ? growthPath(id) : PARENT_HOME);
};
