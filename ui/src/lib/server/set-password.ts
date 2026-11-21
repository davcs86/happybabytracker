import { error, fail, redirect, type RequestEvent } from '@sveltejs/kit';
import { createApi } from './api';
import { PARENT_HOME, UNAVAILABLE_MESSAGE } from './constants';
import { setSessionCookie } from './cookie';
import { getEnv } from './env';
import { MAX_PASSWORD_LEN, MIN_PASSWORD_LEN } from '../constants';
import { passwordProblem } from '../password';
import type { TokenInfo } from '../types';

type Event = RequestEvent<{ token: string }>;

const failure = (status: number, message: string) => fail(status, { field: null, message });

// Serves /invite/[token] and /reset/[token]; a link of the other kind shows as invalid.
export function setPasswordRoute(kind: TokenInfo['kind']) {
	async function load({ locals, params }: Event) {
		const r = await locals.api.peekToken(params.token);
		if (r.ok && r.data.kind === kind) return { valid: true as const, username: r.data.username };
		if (r.ok || r.reason === 'not_found') return { valid: false as const };
		error(503, UNAVAILABLE_MESSAGE);
	}

	async function submit({ request, cookies, params, getClientAddress }: Event) {
		const form = await request.formData();
		const password = String(form.get('password') ?? '');
		const problem = passwordProblem(password, String(form.get('confirm') ?? ''));
		if (problem) return fail(400, problem);

		const api = createApi(getEnv(), fetch, undefined, getClientAddress());
		const r = await api.redeemToken(params.token, password);
		if (r.ok) {
			setSessionCookie(cookies, r.data.token, r.data.kind);
			redirect(303, PARENT_HOME);
		}
		switch (r.reason) {
			case 'not_found':
				return fail(404, { invalid: true });
			case 'invalid':
				return fail(400, {
					field: 'password' as const,
					message: `Use ${MIN_PASSWORD_LEN} to ${MAX_PASSWORD_LEN} characters`
				});
			case 'rate_limited':
				return failure(429, 'Too many attempts, try again in a minute');
			case 'unavailable':
				return failure(503, 'The app cannot reach its server, try again soon');
			default:
				return failure(500, 'Something went wrong, try again');
		}
	}

	return { load, actions: { default: submit } };
}
