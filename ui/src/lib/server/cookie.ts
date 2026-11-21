import type { Cookies } from '@sveltejs/kit';
import { ADMIN_SESSION_MAX_AGE_S, PARENT_SESSION_MAX_AGE_S, SESSION_COOKIE } from './constants';
import type { SessionKind } from '../types';

const ATTRIBUTES = { path: '/', httpOnly: true, secure: true, sameSite: 'strict' } as const;

export function sessionCookieOptions(kind: SessionKind) {
	return {
		...ATTRIBUTES,
		maxAge: kind === 'admin' ? ADMIN_SESSION_MAX_AGE_S : PARENT_SESSION_MAX_AGE_S
	};
}

export function setSessionCookie(cookies: Cookies, token: string, kind: SessionKind): void {
	cookies.set(SESSION_COOKIE, token, sessionCookieOptions(kind));
}

export function clearSessionCookie(cookies: Cookies): void {
	cookies.delete(SESSION_COOKIE, { ...ATTRIBUTES });
}
