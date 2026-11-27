import { error, fail, redirect, type RequestEvent } from '@sveltejs/kit';
import { createApi } from './api';
import { PARENT_HOME, UNAVAILABLE_MESSAGE } from './constants';
import { setSessionCookie } from './cookie';
import { getEnv } from './env';
import { SSO_UNAVAILABLE, ssoOffered, startSso } from './oidc/start';
import { MAX_PASSWORD_LEN, MIN_PASSWORD_LEN } from '../constants';
import { passwordProblem } from '../password';
import type { AuthMode, TokenInfo } from '../types';

type Event = RequestEvent<{ token: string }>;

const failure = (status: number, message: string) => fail(status, { field: null, message });

// Serves /invite/[token] and /reset/[token]; a link of the other kind shows as invalid.
export function setPasswordRoute(kind: TokenInfo['kind']) {
	async function load({ locals, params }: Event) {
		const [r, m] = await Promise.all([locals.api.peekToken(params.token), locals.api.authMode()]);
		const mode: AuthMode = m.ok ? m.data : 'local';
		const sso = m.ok && ssoOffered(mode, getEnv().oidc);
		if (r.ok && r.data.kind === kind) {
			return { valid: true as const, username: r.data.username, mode, sso };
		}
		if (r.ok || r.reason === 'not_found') return { valid: false as const, mode, sso };
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
		if (r.code === 'mode_refused') return failure(403, 'Password sign-in is turned off');
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

	// The mode is read again here, never taken from the page that posted.
	async function sso(event: Event) {
		const mode = await event.locals.api.authMode();
		if (!mode.ok || mode.data === 'local') return failure(503, SSO_UNAVAILABLE);
		const r = await startSso(event, 'invite', { binding: event.params.token });
		if (!r.ok) return failure(503, r.message);
		// The issuer's authorization endpoint, from its discovery document.
		redirect(303, r.location, { external: true });
	}

	// SvelteKit refuses a default action beside named ones.
	const actions: Record<string, typeof submit | typeof sso> =
		kind === 'invite' ? { password: submit, sso } : { default: submit };
	return { load, actions };
}
