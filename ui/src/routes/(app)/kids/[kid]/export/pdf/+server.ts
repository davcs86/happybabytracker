import { redirect } from '@sveltejs/kit';
import { exportPath } from '#lib/paths.js';
import { exportRetrySearch, readExportForm, type ExportError } from '#lib/server/export-form.js';
import { failPage, type ApiFailure } from '#lib/server/outcome.js';
import type { RequestHandler } from './$types';

// No grant answers on the page itself, so the browser gets the same 404 page as every kid route.
function refuse(r: ApiFailure, kidId: string): never {
	if (r.reason === 'not_found' || r.reason === 'forbidden') {
		redirect(303, exportPath(encodeURIComponent(kidId)));
	}
	failPage(r);
}

// The PDF streams through unread; the hook adds Cache-Control: no-store.
export const POST: RequestHandler = async ({ request, locals, params }) => {
	const kid = await locals.api.getKid(params.kid);
	if (!kid.ok) refuse(kid, params.kid);
	const form = await request.formData().catch(() => new FormData());
	const retry = (error: ExportError): never =>
		redirect(303, `${exportPath(kid.data.id)}?${exportRetrySearch(error, form)}`);

	const input = readExportForm(form, kid.data.timeZone);
	if ('field' in input) return retry(input.field);
	const r = await locals.api.exportPdf(kid.data.id, input);
	if (!r.ok) {
		if (
			r.reason === 'invalid' &&
			(r.field === 'from' || r.field === 'to' || r.field === 'modules')
		) {
			return retry(r.field);
		}
		if (r.reason === 'unavailable') return retry('busy');
		return refuse(r, kid.data.id);
	}
	return new Response(r.body, {
		status: 200,
		headers: { 'content-type': r.contentType, 'content-disposition': r.disposition }
	});
};
