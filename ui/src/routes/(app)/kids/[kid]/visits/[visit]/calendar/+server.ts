import { redirect } from '@sveltejs/kit';
import { visitPath } from '#lib/paths.js';
import { failPage, type ApiFailure } from '#lib/server/outcome.js';
import type { RequestHandler } from './$types';

// No grant answers on the visit page itself, so the browser gets the same 404 page as every kid route.
function refuse(r: ApiFailure, params: { kid: string; visit: string }): never {
	if (r.reason === 'not_found' || r.reason === 'forbidden') {
		redirect(303, visitPath(encodeURIComponent(params.kid), encodeURIComponent(params.visit)));
	}
	failPage(r);
}

// The file streams through unread; the hook adds Cache-Control: no-store. Any detail but full is the generic one.
export const GET: RequestHandler = async ({ locals, params, url }) => {
	const detail = url.searchParams.get('detail') === 'full' ? 'full' : 'generic';
	const r = await locals.api.visits.calendar(params.kid, params.visit, detail);
	if (!r.ok) return refuse(r, params);
	return new Response(r.body, {
		status: 200,
		headers: {
			'content-type': r.contentType,
			'content-disposition': r.disposition,
			'x-content-type-options': 'nosniff'
		}
	});
};
