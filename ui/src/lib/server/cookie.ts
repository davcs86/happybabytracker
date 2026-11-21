import type { Cookies } from '@sveltejs/kit';
import {
	ADMIN_SESSION_MAX_AGE_S,
	KID_COOKIE,
	KID_COOKIE_MAX_AGE_S,
	PARENT_SESSION_MAX_AGE_S,
	SESSION_COOKIE
} from './constants';
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

// Holds a kid id only: it preselects the last visited kid on /.
export function setKidCookie(cookies: Cookies, kidId: string): void {
	cookies.set(KID_COOKIE, kidId, { ...ATTRIBUTES, maxAge: KID_COOKIE_MAX_AGE_S });
}

export function clearKidCookie(cookies: Cookies): void {
	cookies.delete(KID_COOKIE, { ...ATTRIBUTES });
}
