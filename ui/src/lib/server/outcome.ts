import { error, fail, redirect } from '@sveltejs/kit';
import type { ApiResult } from './api';
import {
	FAILURE_MESSAGE,
	NOT_FOUND_MESSAGE,
	UNAVAILABLE_MESSAGE,
	UNREACHABLE_MESSAGE
} from './constants';

export type ApiFailure = Extract<ApiResult<unknown>, { ok: false }>;

// No grant (404) and a caregiver on a guardian page (403) show the same 404 page.
export function failPage(r: ApiFailure): never {
	if (r.reason === 'not_found' || r.reason === 'forbidden') error(404, NOT_FOUND_MESSAGE);
	if (r.reason === 'unauthorized' && r.code === 'no_session') redirect(303, '/login');
	// Any other 401 (internal token refused) is a deployment fault, shown as unavailable.
	if (r.reason === 'unavailable' || r.reason === 'unauthorized') error(503, UNAVAILABLE_MESSAGE);
	error(500, FAILURE_MESSAGE);
}

// The catch-all for an action: a refused page is 404, anything else a page-level message.
export function failAction<T extends object>(r: ApiFailure, extra: T) {
	if (r.reason === 'not_found' || r.reason === 'forbidden' || r.reason === 'unauthorized') {
		failPage(r);
	}
	if (r.reason === 'unavailable') return fail(503, { ...extra, message: UNREACHABLE_MESSAGE });
	return fail(500, { ...extra, message: FAILURE_MESSAGE });
}
