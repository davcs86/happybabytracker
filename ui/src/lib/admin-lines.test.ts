import { describe, expect, it } from 'vitest';
import { lastActiveLine, linkLine } from './admin-lines';
import type { AdminParent } from './types';

const now = new Date('2026-11-25T12:00:00Z');
const at = (minutes: number) => new Date(now.getTime() - minutes * 60_000).toISOString();
const parent = (over: Partial<AdminParent> = {}): AdminParent => ({
	id: 'p',
	username: 'ana',
	displayName: 'Ana',
	disabled: false,
	hasPassword: true,
	ssoLinked: false,
	createdAt: at(1000),
	lastActiveAt: null,
	link: null,
	...over
});

describe('lastActiveLine', () => {
	it('says how long ago the newest session was active', () => {
		expect(lastActiveLine(parent({ lastActiveAt: at(94) }), now)).toBe(
			'Last active 1 h 34 min ago'
		);
	});
	it('says a parent who never set a password has not signed in', () => {
		expect(lastActiveLine(parent({ hasPassword: false }), now)).toBe('Hasn’t signed in yet');
	});
	it('says there is no recent sign-in for a parent with a password and no session, as a disabled one has none', () => {
		expect(lastActiveLine(parent(), now)).toBe('No recent sign-in');
		expect(lastActiveLine(parent({ disabled: true }), now)).toBe('No recent sign-in');
	});
});

describe('linkLine', () => {
	const link = (kind: 'invite' | 'reset', issued: number, expires: number) => ({
		kind,
		issuedAt: at(issued),
		expiresAt: new Date(now.getTime() + expires * 60_000).toISOString()
	});
	it('says when the link was issued and until when it is valid', () => {
		expect(linkLine(link('reset', 20, 40), now)).toBe(
			'Reset link issued 20 min ago. Valid until 2026-11-25 12:40 UTC.'
		);
		expect(linkLine(link('invite', 60, 4000), now)).toMatch(
			/^Invite link issued 1 h ago\. Valid until /
		);
	});
	it('says Expired once it is past', () => {
		expect(linkLine(link('invite', 4400, -20), now)).toBe('Invite link issued 3 d ago. Expired.');
	});
	it('is empty without a link', () => {
		expect(linkLine(null, now)).toBe('');
	});
});
