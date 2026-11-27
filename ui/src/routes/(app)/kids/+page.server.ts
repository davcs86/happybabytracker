import { sectionOrHome } from '#lib/switch-target.js';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = ({ url }) => ({
	to: sectionOrHome(url.searchParams.get('to')),
	q: url.searchParams.get('q') ?? ''
});
