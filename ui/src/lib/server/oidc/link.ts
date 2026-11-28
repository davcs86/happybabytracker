import { createHash, timingSafeEqual } from 'node:crypto';
import { fail, redirect, type RequestEvent } from '@sveltejs/kit';
import { createApi } from '../api';
import { SESSION_COOKIE } from '../constants';
import { getEnv } from '../env';
import { failAction } from '../outcome';
import type { AuthMode, Session, SessionKind } from '../../types';
import { ssoOffered, startSso } from './start';

export const TOO_MANY_ATTEMPTS = 'Too many attempts, try again in a minute';
export const LINKING_OFF = 'Linking is turned off';
const WRONG_PASSWORD = 'The password is incorrect';

export function sessionBinding(token: string): string {
	return createHash('sha256').update(token, 'utf8').digest('base64url');
}

// Constant time; a missing token or a length mismatch is a refusal, never a throw.
export function bindingMatches(token: string | undefined, binding: string): boolean {
	if (!token) return false;
	const a = Buffer.from(sessionBinding(token), 'utf8');
	const b = Buffer.from(binding, 'utf8');
	return a.length === b.length && timingSafeEqual(a, b);
}

export const ssoLinkedOf = (s: Session | null) =>
	Boolean(s?.parent?.ssoLinked ?? s?.admin?.ssoLinked);

const usernameOf = (s: Session | null) => s?.parent?.username ?? s?.admin?.username ?? null;

// Serves /account/link-sso and /admin/account/link-sso.
export function linkSsoRoute(kind: SessionKind) {
	async function load({ locals }: RequestEvent) {
		const r = await locals.api.authMode();
		const mode: AuthMode = r.ok ? r.data : 'local';
		return {
			mode,
			sso: r.ok && ssoOffered(mode, getEnv().oidc),
			linked: ssoLinkedOf(locals.session)
		};
	}

	async function submit(event: RequestEvent) {
		const { request, cookies, locals, getClientAddress } = event;
		const password = String((await request.formData()).get('password') ?? '');
		const token = cookies.get(SESSION_COOKIE);
		const username = usernameOf(locals.session);
		if (!token || !username) redirect(303, '/login');
		const api = createApi(getEnv(), fetch, token, getClientAddress());
		const r = kind === 'admin' ? await api.adminReauth(password) : await api.reauth(password);
		if (!r.ok) {
			if (r.code === 'wrong_password') {
				return fail(403, { field: 'password', message: WRONG_PASSWORD });
			}
			if (r.status === 429) return fail(429, { message: TOO_MANY_ATTEMPTS });
			if (r.code === 'mode_refused') return fail(403, { message: LINKING_OFF });
			return failAction(r, {});
		}
		const s = await startSso(event, 'link', { binding: sessionBinding(token), username, kind });
		if (!s.ok) return fail(503, { message: s.message });
		// The issuer's authorization endpoint, from its discovery document.
		redirect(303, s.location, { external: true });
	}

	return { load, actions: { default: submit } };
}
