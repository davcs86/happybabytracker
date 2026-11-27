import { afterEach, describe, expect, it, vi } from 'vitest';
import type { OidcEnv } from '../env';
import { ssoBack, ssoOffered } from './start';

const OIDC: OidcEnv = {
	issuer: 'https://issuer.test',
	clientId: 'fake-client',
	clientSecret: 'fake-client-secret-000000000000',
	scopes: 'openid profile',
	sealKey: 'fake-seal-key-00000000000000000000'
};

const NOT_CONFIGURED = JSON.stringify({
	level: 'warn',
	msg: 'sso refused',
	code: 'oidc_not_configured'
});

afterEach(() => vi.restoreAllMocks());

describe('ssoOffered', () => {
	it('is true only for oidc or both with settings', () => {
		const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
		expect(ssoOffered('oidc', OIDC)).toBe(true);
		expect(ssoOffered('both', OIDC)).toBe(true);
		expect(ssoOffered('local', OIDC)).toBe(false);
		expect(ssoOffered('local', null)).toBe(false);
		expect(warn).not.toHaveBeenCalled();
	});

	it('without settings it is false and logs oidc_not_configured once per process', () => {
		const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
		expect(ssoOffered('oidc', null)).toBe(false);
		expect(ssoOffered('both', null)).toBe(false);
		expect(ssoOffered('oidc', null)).toBe(false);
		expect(warn.mock.calls.map((c) => c.map(String).join(' '))).toEqual([NOT_CONFIGURED]);
	});
});

describe('ssoBack', () => {
	it.each<[Parameters<typeof ssoBack>, { href: string; label: string }]>([
		[['login', null, null], { href: '/login', label: 'Go to sign in' }],
		[[null, null, null], { href: '/login', label: 'Go to sign in' }],
		[
			['invite', null, 'i'.repeat(43)],
			{ href: `/invite/${'i'.repeat(43)}`, label: 'Go to your invite link' }
		],
		[['invite', null, null], { href: '/login', label: 'Go to sign in' }],
		[['link', 'parent', 'h'.repeat(43)], { href: '/account', label: 'Go to My profile' }],
		[['link', 'admin', 'h'.repeat(43)], { href: '/admin/account', label: 'Go to Account' }]
	])('%j', (args, want) => {
		expect(ssoBack(...args)).toEqual(want);
	});
});
