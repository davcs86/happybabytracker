import { failPage } from '#lib/server/outcome.js';
import type { LayoutServerLoad } from './$types';

export const load: LayoutServerLoad = async ({ locals }) => {
	const r = await locals.api.listKids();
	if (!r.ok) failPage(r);
	return { kids: r.data };
};
