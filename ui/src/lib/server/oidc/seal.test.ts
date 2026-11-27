import { describe, expect, it } from 'vitest';
import { OIDC_COOKIE_MAX_BYTES, OIDC_TXN_COOKIE, OIDC_TXN_MAX_AGE_S } from '../constants';
import { open, seal } from './seal';
import type { OidcPending, OidcStart } from './types';

const KEY = 'k'.repeat(32);
const NOW = 1_800_000_000;

const start: OidcStart = {
	stage: 'start',
	purpose: 'login',
	state: 's'.repeat(43),
	nonce: 'n'.repeat(43),
	verifier: 'v'.repeat(43),
	startedAt: NOW,
	next: '/kids',
	binding: null,
	username: null,
	kind: null,
	deviceLabel: 'Firefox on Linux',
	exp: NOW + 600
};

const pending: OidcPending = {
	stage: 'pending',
	purpose: 'invite',
	binding: 'b'.repeat(43),
	username: 'u'.repeat(32),
	kind: null,
	iss: 'https://' + 'i'.repeat(392),
	sub: 'x'.repeat(400),
	label: '\u{1F600}'.repeat(64),
	deviceLabel: 'd'.repeat(48),
	exp: NOW + 600
};

// The Set-Cookie header the browser stores, so the size check covers name and attributes.
function setCookieBytes(value: string): number {
	const header = `${OIDC_TXN_COOKIE}=${value}; Max-Age=${OIDC_TXN_MAX_AGE_S}; Path=/auth; HttpOnly; Secure; SameSite=Lax`;
	return Buffer.byteLength(header);
}

function flip(value: string, at: number): string {
	const bytes = Buffer.from(value, 'base64url');
	bytes[at] ^= 0x01;
	return bytes.toString('base64url');
}

describe('seal', () => {
	it('round-trips a start and a pending payload', () => {
		expect(open(KEY, seal(KEY, start), NOW)).toEqual(start);
		expect(open(KEY, seal(KEY, pending), NOW)).toEqual(pending);
	});

	it('is base64url', () => {
		expect(seal(KEY, start)).toMatch(/^[A-Za-z0-9_-]+$/);
	});

	it('differs on each seal of the same payload', () => {
		expect(seal(KEY, start)).not.toBe(seal(KEY, start));
	});

	it.each([0, 12, -1])('gives null on a flipped byte at %i', (at) => {
		const value = seal(KEY, start);
		const len = Buffer.from(value, 'base64url').length;
		expect(open(KEY, flip(value, at < 0 ? len + at : at), NOW)).toBeNull();
	});

	it('gives null for a value sealed with another key', () => {
		expect(open(KEY, seal('o'.repeat(32), start), NOW)).toBeNull();
	});

	it('gives null for a truncated value', () => {
		const value = seal(KEY, start);
		expect(open(KEY, value.slice(0, -4), NOW)).toBeNull();
		expect(open(KEY, value.slice(0, 20), NOW)).toBeNull();
	});

	it.each([undefined, '', '!!!', 'not-a-sealed-value'])('gives null for %j', (value) => {
		expect(open(KEY, value, NOW)).toBeNull();
	});

	it('gives null once exp is reached', () => {
		expect(open(KEY, seal(KEY, { ...start, exp: NOW }), NOW)).toBeNull();
		expect(open(KEY, seal(KEY, { ...start, exp: NOW - 1 }), NOW)).toBeNull();
		expect(open(KEY, seal(KEY, { ...start, exp: NOW + 1 }), NOW)).not.toBeNull();
	});

	it('gives null for an unknown stage', () => {
		const other = { ...start, stage: 'other' } as unknown as OidcStart;
		expect(open(KEY, seal(KEY, other), NOW)).toBeNull();
	});

	it.each([undefined, '1800000000', null])(
		'gives null for a start payload with startedAt %j',
		(startedAt) => {
			const bad = { ...start, startedAt } as unknown as OidcStart;
			expect(open(KEY, seal(KEY, bad), NOW)).toBeNull();
		}
	);

	it('keeps a pending payload with the longest fields under the cookie limit', () => {
		const value = seal(KEY, pending);
		expect(open(KEY, value, NOW)).toEqual(pending);
		expect(setCookieBytes(value)).toBeLessThan(OIDC_COOKIE_MAX_BYTES);
	});

	it('round-trips a link start under the cookie limit', () => {
		const link: OidcStart = {
			...start,
			purpose: 'link',
			next: null,
			binding: 'b'.repeat(43),
			username: 'u'.repeat(32),
			kind: 'parent',
			deviceLabel: null,
			startedAt: NOW - 5
		};
		const value = seal(KEY, link);
		expect(open(KEY, value, NOW)).toEqual(link);
		expect(setCookieBytes(value)).toBeLessThan(OIDC_COOKIE_MAX_BYTES);
	});
});
