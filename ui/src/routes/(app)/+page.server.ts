import { redirect } from '@sveltejs/kit';
import { currentKidId } from '#lib/server/current-kid.js';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ parent, cookies }) => {
	const { kids } = await parent();
	const id = currentKidId(kids, cookies);
	if (id) redirect(303, `/kids/${id}`);
};
