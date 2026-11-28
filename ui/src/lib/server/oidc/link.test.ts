import { createHash } from 'node:crypto';
import { describe, expect, it } from 'vitest';
import { bindingMatches, sessionBinding } from './link';

const TOKEN = 'fake-session-token-0000';

describe('sessionBinding', () => {
	it('is the base64url SHA-256 of the cookie value', () => {
		expect(sessionBinding(TOKEN)).toBe(createHash('sha256').update(TOKEN).digest('base64url'));
		expect(sessionBinding(TOKEN)).toMatch(/^[A-Za-z0-9_-]{43}$/);
	});
});

describe('bindingMatches', () => {
	const bound = sessionBinding(TOKEN);

	it('accepts the bound session only', () => {
		expect(bindingMatches(TOKEN, bound)).toBe(true);
		expect(bindingMatches('fake-session-token-0001', bound)).toBe(false);
	});

	it('refuses a missing token or a binding of another length without throwing', () => {
		expect(bindingMatches(undefined, bound)).toBe(false);
		expect(bindingMatches('', bound)).toBe(false);
		expect(bindingMatches(TOKEN, bound.slice(1))).toBe(false);
		expect(bindingMatches(TOKEN, '')).toBe(false);
	});
});
