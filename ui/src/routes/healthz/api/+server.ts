import { text } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

// The API's own status, or 503 when it does not answer in time.
export const GET: RequestHandler = async ({ locals }) => {
	const r = await locals.api.health();
	if (r.ok) return text('ok');
	return text('unavailable', { status: r.status >= 400 ? r.status : 503 });
};
