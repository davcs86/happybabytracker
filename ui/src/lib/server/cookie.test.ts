import { describe, expect, it, vi } from 'vitest';
import type { Cookies } from '@sveltejs/kit';
import {
	clearKidCookie,
	clearSessionCookie,
	sessionCookieOptions,
	setKidCookie,
	setSessionCookie
} from './cookie';

function fakeCookies() {
	return { set: vi.fn(), delete: vi.fn() } as unknown as Cookies & {
		set: ReturnType<typeof vi.fn>;
		delete: ReturnType<typeof vi.fn>;
	};
}

const hardened = { path: '/', httpOnly: true, secure: true, sameSite: 'strict' };

describe('session cookie', () => {
	it('lives 30 days for a parent', () => {
		expect(sessionCookieOptions('parent')).toEqual({ ...hardened, maxAge: 30 * 24 * 60 * 60 });
	});

	it('lives 12 hours for the admin', () => {
		expect(sessionCookieOptions('admin')).toEqual({ ...hardened, maxAge: 12 * 60 * 60 });
	});

	it.each(['parent', 'admin'] as const)('is set as hbt_session for %s', (kind) => {
		const cookies = fakeCookies();
		setSessionCookie(cookies, 'tok', kind);
		expect(cookies.set).toHaveBeenCalledWith('hbt_session', 'tok', sessionCookieOptions(kind));
	});

	it('is cleared with the same path and attributes', () => {
		const cookies = fakeCookies();
		clearSessionCookie(cookies);
		expect(cookies.delete).toHaveBeenCalledWith('hbt_session', hardened);
	});
});

describe('kid cookie', () => {
	it('is set as hbt_kid with the kid id only, hardened, for 30 days', () => {
		const cookies = fakeCookies();
		setKidCookie(cookies, 'k-1');
		expect(cookies.set).toHaveBeenCalledWith('hbt_kid', 'k-1', {
			...hardened,
			maxAge: 30 * 24 * 60 * 60
		});
	});

	it('is cleared with the same path and attributes', () => {
		const cookies = fakeCookies();
		clearKidCookie(cookies);
		expect(cookies.delete).toHaveBeenCalledWith('hbt_kid', hardened);
	});
});
